package main

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/diegosz/flaggy"
	"github.com/diegosz/go-archetype/generator"
	"github.com/diegosz/go-archetype/log"
	"github.com/gogs/git-module"
	"github.com/joho/godotenv"
	"go.uber.org/multierr"

	"github.com/diegosz/garchetype/internal/gitstat"
)

const (
	exeName                 = "garchetype"
	envPrefix               = "GARCHETYPE"
	transformationPrefix    = "transformations-"
	transformationExt       = "yaml"
	defaultArchetypesFolder = "archetypes"
	defaultTransformation   = "default"
	defaultArchetype        = "hello-world"
	featureNameID           = "feature_name"
)

var ErrSilentExit = errors.New("silent exit")

func main() {
	if err := run(context.Background(), os.Stdout, os.Stderr, os.Args); err != nil {
		if !errors.Is(err, ErrSilentExit) {
			fmt.Fprintf(os.Stderr, "💥 %s error: %s\n", exeName, err)
		}
		os.Exit(1)
	}
	os.Exit(0)
}

type Config struct {
	Verbose          bool
	FeatureName      string
	ArchetypesFolder string
	Archetype        string
	Transformation   string
	SourceDir        string
	SourceRepo       string
}

// newDefaultConfig returns a new default config with the default values set.
func newDefaultConfig() *Config {
	return &Config{
		Verbose:          strings.ToLower(os.Getenv(envPrefix+"_VERBOSE")) == "true",
		ArchetypesFolder: cmp.Or(os.Getenv(envPrefix+"_ARCHETYPES_FOLDER"), defaultArchetypesFolder),
		Archetype:        cmp.Or(os.Getenv(envPrefix+"_ARCHETYPE"), defaultArchetype),
		Transformation:   cmp.Or(os.Getenv(envPrefix+"_TRANSFORMATION"), defaultTransformation),
		SourceDir:        os.Getenv(envPrefix + "_SOURCE_DIR"),
		SourceRepo:       os.Getenv(envPrefix + "_SOURCE_REPO"),
	}
}

var environment = []string{
	envPrefix + "_ARCHETYPE",
	envPrefix + "_ARCHETYPES_FOLDER",
	envPrefix + "_ENV",
	envPrefix + "_SOURCE_DIR",
	envPrefix + "_SOURCE_REPO",
	envPrefix + "_TRANSFORMATION",
	envPrefix + "_VERBOSE",
}

func run(_ context.Context, stdout, _ io.Writer, args []string) (err error) {
	// Try to read the default .env file in the current path into ENV for this
	// process. It WILL NOT OVERRIDE an env variable that already exists -
	// consider the .env file to set dev vars or sensible defaults.
	_ = godotenv.Load()

	flaggy.ShowHelpOnUnexpectedEnable()
	flaggy.SetName(exeName)
	flaggy.SetDescription("Tool for scaffolding using archetypes.")
	flaggy.SetVersion(Version)

	if env := os.Getenv(envPrefix + "_ENV"); env != "" {
		if err := godotenv.Load(env); err != nil {
			return err
		}
	}

	cfg := newDefaultConfig() // Set the default values prior to parsing.

	addCommand := flaggy.NewSubcommand("add")
	addCommand.Description = "Add a feature using an archetype."
	addCommand.String(&cfg.FeatureName, "f", "feature", "Feature name to add.")
	addCommand.String(&cfg.Archetype, "a", "archetype", "Archetype to use.")
	addCommand.String(&cfg.Transformation, "t", "transformation", "Transformation to use.")
	addCommand.String(&cfg.SourceDir, "s", "source-dir", "Source directory to use.")
	addCommand.String(&cfg.SourceRepo, "r", "source-repo", "Source repository to use.")

	listCommand := flaggy.NewSubcommand("list")
	listCommand.Description = "List available archetypes."
	listCommand.String(&cfg.SourceDir, "s", "source-dir", "Source directory to use.")

	environmentCommand := flaggy.NewSubcommand("environment")
	environmentCommand.Hidden = true

	flaggy.AttachSubcommand(addCommand, 1)
	flaggy.AttachSubcommand(listCommand, 1)
	flaggy.AttachSubcommand(environmentCommand, 1)

	flaggy.ParseArgs(args[1:])

	switch {
	case addCommand.Used:
		if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
			return errors.New("go.mod file not found in the current folder")
		}
		if err := setSource(stdout, cfg); err != nil {
			return err
		}
		if cfg.Archetype == "" {
			err = multierr.Append(err, errors.New("archetype is required"))
		}
		if cfg.FeatureName == "" {
			cfg.FeatureName = cfg.Archetype
		}
		if cfg.SourceDir == "" {
			err = multierr.Append(err, errors.New("source directory is required"))
		}
		if err != nil {
			return err
		}
		return addFeature(stdout, cfg, flaggy.TrailingArguments...)
	case listCommand.Used:
		if err := setSource(stdout, cfg); err != nil {
			return err
		}
		if cfg.SourceDir == "" {
			err = multierr.Append(err, errors.New("source directory is required"))
		}
		if err != nil {
			return err
		}
		return list(stdout, cfg)
	case environmentCommand.Used:
		for _, e := range environment {
			fmt.Fprintf(stdout, "%s\n", e)
		}
		return nil
	default:
		flaggy.ShowHelp("")
		return nil
	}
}

func setSource(stdout io.Writer, cfg *Config) error {
	if cfg.SourceDir == "" {
		return errors.New("source directory is required")
	}
	g, err := git.Open(cfg.SourceDir)
	switch err != nil {
	case true:
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		switch cfg.SourceRepo == "" {
		case true:
			return fmt.Errorf("source directory not found: %s", cfg.SourceDir)
		default:
			if err := git.Clone(cfg.SourceRepo, cfg.SourceDir); err != nil {
				switch strings.Contains(err.Error(), "ssh: Could not resolve hostname") {
				case true:
					fmt.Fprintln(stdout, "🚨 Could not connect to remote repository.")
					return fmt.Errorf("source directory not found: %s", cfg.SourceDir)
				default:
					return err
				}
			}
		}
	default:
		if _, err := g.RemoteGetURL("origin"); err == nil {
			if err := g.Fetch(); err != nil {
				switch strings.Contains(err.Error(), "ssh: Could not resolve hostname") {
				case true:
					fmt.Fprintln(stdout, "🚨 Could not connect to remote repository.")
					return nil
				default:
					return err
				}
			}
			if err := g.Pull(); err != nil {
				return err
			}
		} else {
			e := err.Error()
			if !strings.Contains(e, "not a git repository") &&
				!strings.Contains(e, "No such remote") {
				return err
			}
		}
	}
	return nil
}

func addFeature(stdout io.Writer, cfg *Config, args ...string) error {
	dest := "."
	dest, err := filepath.Abs(dest)
	if err != nil {
		return err
	}
	asd, err := getArchetypesFolder(cfg.SourceDir, cfg.ArchetypesFolder)
	if err != nil {
		return err
	}
	ad, err := getArchetypeFolder(asd, cfg.Archetype)
	if err != nil {
		return err
	}
	tf, err := getTransformationFile(cfg.Transformation)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "🌱 Adding '%s' feature using '%s' archetype.\n", cfg.FeatureName, cfg.Archetype)
	tf = filepath.Join(ad, tf)
	fi, err := os.Stat(tf)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return fmt.Errorf("invalid transformation file: %s", tf)
	}
	fmt.Fprintf(stdout, "📦 Using transformation file: %s\n", tf)
	gs, err := gitstat.Get()
	if err != nil {
		return err
	}
	if gs.Dirty {
		return errors.New("git repository is dirty")
	}
	var fn string
	b, err := os.ReadFile(tf)
	if err != nil {
		return err
	}
	if bytes.Contains(b, []byte("- id: feature_name")) {
		fn = cfg.FeatureName
	}
	if err := generator.OverlayGenerate(tf, ad, dest, getFeatureArgs(fn, args), log.NewZeroLogger("warn")); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "🎉 Feature '%s' added.\n", cfg.FeatureName)
	return nil
}

func list(stdout io.Writer, cfg *Config) error {
	ad, err := getArchetypesFolder(cfg.SourceDir, cfg.ArchetypesFolder)
	if err != nil {
		return err
	}
	as, err := getArchetypes(ad)
	if err != nil {
		return err
	}
	for _, a := range as {
		ts, err := getTransformations(filepath.Join(ad, a))
		if err != nil {
			return err
		}
		if len(ts) == 0 {
			continue
		}
		fmt.Fprintf(stdout, "📦 Archetype: %s\n", a)
		if len(ts) == 1 && ts[0] == defaultTransformation {
			continue
		}
		for _, t := range ts {
			fmt.Fprintf(stdout, " 📄 Transformation: %s\n", t)
		}
	}
	return nil
}

func getArchetypesFolder(dir, archetypes string) (string, error) {
	if dir == "" {
		return "", errors.New("undefined dir")
	}
	if archetypes == "" {
		return "", errors.New("undefined archetypes")
	}
	ad := filepath.Join(dir, archetypes)
	fi, err := os.Stat(ad)
	if err != nil {
		return "", err
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("invalid archetypes folder: %s", ad)
	}
	return ad, nil
}

func getArchetypeFolder(dir, archetype string) (string, error) {
	if dir == "" {
		return "", errors.New("undefined dir")
	}
	if archetype == "" {
		return "", errors.New("undefined archetype")
	}
	ad := filepath.Join(dir, archetype)
	fi, err := os.Stat(ad)
	if err != nil {
		return "", err
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("invalid archetype folder: %s", ad)
	}
	return ad, nil
}

func getTransformationFile(transformation string) (string, error) {
	if transformation == "" {
		return "", errors.New("undefined transformation")
	}
	return fmt.Sprintf("%s%s.%s", transformationPrefix, transformation, transformationExt), nil
}

func getArchetypes(dir string) ([]string, error) {
	if dir == "" {
		return nil, errors.New("undefined dir")
	}
	var as []string
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	entries, err := f.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	for _, f := range entries {
		fi, err := os.Stat(filepath.Join(dir, f))
		if err != nil {
			return nil, err
		}
		if fi.IsDir() {
			as = append(as, f)
		}
	}
	return as, nil
}

func getTransformations(dir string) ([]string, error) {
	if dir == "" {
		return nil, errors.New("undefined dir")
	}
	var ts []string
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	entries, err := f.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	for _, f := range entries {
		if strings.HasPrefix(f, transformationPrefix) && strings.HasSuffix(f, transformationExt) {
			t := strings.TrimSuffix(strings.TrimPrefix(f, transformationPrefix), "."+transformationExt)
			ts = append(ts, t)
		}
	}
	return ts, nil
}

func getFeatureArgs(featureName string, args []string) []string {
	as := []string{}
	if featureName != "" {
		as = append(as, []string{"--" + featureNameID, featureName}...)
	}
	var removeNext bool
	for _, a := range args {
		switch {
		case removeNext:
			removeNext = false
			continue
		case a == "--"+featureNameID:
			removeNext = true
			continue
		case strings.HasPrefix(a, "--"+featureNameID+"="):
			continue
		default:
			as = append(as, a)
		}
	}
	return as
}
