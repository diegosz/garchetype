package main

import (
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
	"github.com/joho/godotenv"
	"go.uber.org/multierr"

	// "github.com/gogs/git-module"

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
)

var ErrSilentExit = errors.New("silent exit")

func main() {
	if err := run(context.Background(), os.Stdin, os.Stdout, os.Stderr, os.Args); err != nil {
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
}

// newDefaultConfig returns a new default config with the default values set.
func newDefaultConfig() *Config {
	return &Config{
		Verbose:          strings.ToLower(os.Getenv(envPrefix+"_VERBOSE")) == "true",
		ArchetypesFolder: cmp.Or(os.Getenv(envPrefix+"_ARCHETYPES_FOLDER"), defaultArchetypesFolder),
		Archetype:        cmp.Or(os.Getenv(envPrefix+"_ARCHETYPE"), defaultArchetype),
		Transformation:   cmp.Or(os.Getenv(envPrefix+"_TRANSFORMATION"), defaultTransformation),
		SourceDir:        os.Getenv(envPrefix + "_SOURCE_DIR"),
	}
}

func run(ctx context.Context, _ io.Reader, stdout, _ io.Writer, args []string) (err error) {
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

	listCommand := flaggy.NewSubcommand("list")
	listCommand.Description = "List available archetypes."
	listCommand.String(&cfg.SourceDir, "s", "source-dir", "Source directory to use.")

	flaggy.AttachSubcommand(addCommand, 1)
	flaggy.AttachSubcommand(listCommand, 1)

	flaggy.ParseArgs(args[1:])

	switch {
	case addCommand.Used:
		if cfg.FeatureName == "" {
			err = multierr.Append(err, errors.New("feature name is required"))
		}
		if cfg.Archetype == "" {
			err = multierr.Append(err, errors.New("archetype is required"))
		}
		if cfg.SourceDir == "" {
			err = multierr.Append(err, errors.New("source directory is required"))
		}
		if err != nil {
			return err
		}
		return addFeature(ctx, stdout, cfg, flaggy.TrailingArguments...)
	case listCommand.Used:
		if cfg.SourceDir == "" {
			err = multierr.Append(err, errors.New("source directory is required"))
		}
		if err != nil {
			return err
		}
		return list(stdout, cfg)
	default:
		flaggy.ShowHelp("")
		return nil
	}
}

func addFeature(_ context.Context, stdout io.Writer, cfg *Config, args ...string) error {
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
	as, err := getFeatureArgs(cfg.FeatureName, args)
	if err != nil {
		return err
	}
	if err := generator.OverlayGenerate(tf, ad, dest, as, log.NewZeroLogger("warn")); err != nil {
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

func getFeatureArgs(name string, args []string) ([]string, error) {
	if name == "" {
		return nil, errors.New("feature name is required")
	}
	as := []string{"--name", name}
	var removeNext bool
	for _, a := range args {
		switch {
		case removeNext:
			removeNext = false
			continue
		case a == "--name":
			removeNext = true
			continue
		case strings.HasPrefix(a, "--name="):
			continue
		default:
			as = append(as, a)
		}
	}
	return as, nil
}
