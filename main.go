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
	"github.com/joho/godotenv"
	"github.com/rantav/go-archetype/generator"
	"github.com/rantav/go-archetype/log"
	"go.uber.org/multierr"

	// "github.com/gogs/git-module"

	"github.com/diegosz/garchetype/internal/gitstat"
)

// Version can be set at link time to override debug.BuildInfo.Main.Version,
// which is "(devel)" when building from within the module. See
// golang.org/issue/29814 and golang.org/issue/29228.
var Version string

const (
	exeName   = "garchetype"
	envPrefix = "GARCHETYPE"
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

var (
	ArchetypePrefix  = "transformations-"
	ArchetypeExt     = "yaml"
	DefaultArchetype = "default"
)

type Config struct {
	Verbose     bool
	FeatureName string
	Archetype   string
	SourceDir   string
}

// newDefaultConfig returns a new default config with the default values set.
func newDefaultConfig() *Config {
	return &Config{
		Verbose:   strings.ToLower(os.Getenv(envPrefix+"_VERBOSE")) == "true",
		Archetype: cmp.Or(os.Getenv(envPrefix+"_ARCHETYPE"), DefaultArchetype),
		SourceDir: os.Getenv(envPrefix + "_SOURCE_DIR"),
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
	addCommand.String(&cfg.SourceDir, "s", "source-dir", "Source directory to use.")

	listCommand := flaggy.NewSubcommand("list")
	listCommand.Description = "List available archetypes."
	listCommand.String(&cfg.SourceDir, "s", "source-dir", "Source directory to use.")

	flaggy.AttachSubcommand(addCommand, 1)
	flaggy.AttachSubcommand(listCommand, 1)

	// flaggy.Parse()
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
		return listArchetypes(stdout, cfg)
	default:
		flaggy.ShowHelp("")
		return nil
	}
}

func addFeature(_ context.Context, stdout io.Writer, cfg *Config, args ...string) error {
	fmt.Fprintf(stdout, "🛠️  Adding '%s' feature using '%s' archetype.\n", cfg.FeatureName, cfg.Archetype)
	a := getArchetypeFilename(cfg.Archetype)
	fmt.Fprintf(stdout, "📦  Using archetype file: %s\n", a)
	t := filepath.Join(cfg.SourceDir, a)
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
	if err := generator.Generate(t, cfg.SourceDir, "destination", as, log.NewZeroLogger("info")); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "🎉  Feature '%s' added.\n", cfg.FeatureName)
	return nil
}

func listArchetypes(stdout io.Writer, cfg *Config) error {
	ts, err := getTransformations(cfg.SourceDir)
	if err != nil {
		return err
	}
	for _, f := range ts {
		fmt.Fprintf(stdout, "📦  %s\n", f)
	}
	return nil
}

func getArchetypeFilename(archetype string) string {
	return fmt.Sprintf("%s%s.%s", ArchetypePrefix, archetype, ArchetypeExt)
}

func getTransformations(dir string) ([]string, error) {
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
		if strings.HasPrefix(f, ArchetypePrefix) && strings.HasSuffix(f, ArchetypeExt) {
			t := strings.TrimSuffix(strings.TrimPrefix(f, ArchetypePrefix), "."+ArchetypeExt)
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
