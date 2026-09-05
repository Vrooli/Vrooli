package api

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/bootstrap"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/shell"
)

type RuntimeConfig struct {
	Root                string
	Home                string
	Logger              *slog.Logger
	LookPathFn          func(string) (string, error)
	CommandFn           func(context.Context, string, ...string) ([]byte, error)
	StartAllScenariosFn func() (control.StartReport, error)
	StopAllScenariosFn  func() (control.StopReport, error)
	StopScenarioFn      func(string) error
}

func DefaultHomeDir() string {
	home, err := config.HomeDir()
	if err != nil {
		return ""
	}
	return home
}

func ResolveRepoRoot() string {
	if root, err := buildinfo.ResolveSourceRoot(); err == nil {
		return CanonicalRepoRootFromOverride(root)
	}
	if root, err := repocontract.ResolveRepoRoot(); err == nil {
		return root
	}
	executable, _ := os.Executable()
	return filepath.Dir(filepath.Dir(executable))
}

func CanonicalRepoRootFromOverride(root string) string {
	if resolved, err := repocontract.FindRepoRootFromPath(root); err == nil {
		return resolved
	}
	return filepath.Clean(root)
}

func BuildRuntimeApp(cfg RuntimeConfig) *App {
	services := bootstrap.New(cfg.Root, cfg.Home, io.Discard, io.Discard, cfg.Logger)
	app := NewWithServices(services)
	if cfg.LookPathFn != nil {
		app.LookPathFn = cfg.LookPathFn
	}
	if cfg.CommandFn != nil {
		app.CommandFn = cfg.CommandFn
	}
	if cfg.StartAllScenariosFn != nil {
		app.StartAllScenariosFn = cfg.StartAllScenariosFn
	}
	if cfg.StopAllScenariosFn != nil {
		app.StopAllScenariosFn = cfg.StopAllScenariosFn
	}
	if cfg.StopScenarioFn != nil {
		app.StopScenarioFn = cfg.StopScenarioFn
	}
	return app
}

func DefaultLookPath(name string) (string, error) {
	return shell.LookPath(name)
}

func DefaultCommandOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	return shell.Output(shell.Spec{
		Context: ctx,
		Name:    name,
		Args:    args,
	})
}
