package app

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"github.com/vrooli/resources/sqlite/cli/internal/discovery"
	"github.com/vrooli/resources/sqlite/cli/internal/env"
)

// BuildInfo captures build-time metadata for the sqlite CLI entrypoint.
type BuildInfo struct {
	Name        string
	Version     string
	Description string
	Fingerprint string
	Timestamp   string
	SourceRoot  string
}

// exitError carries an exit code without printing additional noise.
type exitError int

func (e exitError) Error() string { return fmt.Sprintf("exit code %d", int(e)) }

// BuildCommandApp wires the SQLite command surface behind the shared cli/
// entrypoint so custom resource logic lives under cli/internal/.
func BuildCommandApp(info BuildInfo) (*cliapp.App, error) {
	cfg := env.Load()

	stale := cliutil.NewStaleChecker(
		info.Name,
		info.Fingerprint,
		info.Timestamp,
		info.SourceRoot,
		"SQLITE_CLI_SOURCE_ROOT",
		"VROOLI_CLI_SOURCE_ROOT",
	)
	stale.SourceContextPath = ".."
	stale.ManifestSourcePath = "resource.json"
	stale.FreshnessInputs = []string{
		"cli/**",
		"cli/internal/**",
		"docs/**",
		"README.md",
		"resource.json",
	}

	runtime := discovery.DiscoverRuntime(info.SourceRoot)
	if runtime.InstalledManifest == "" {
		return nil, fmt.Errorf("resolve installed manifest path for %s", info.Name)
	}
	legacy := New(cfg, runtime)

	return NewCommandSurface(info, cfg, legacy, stale), nil
}

// NewCommandSurface builds the operator-facing SQLite command app.
func NewCommandSurface(info BuildInfo, cfg env.Config, legacy *CLI, stale *cliutil.StaleChecker) *cliapp.App {
	_ = cfg // reserved for future config-driven wiring

	commands := []cliapp.CommandGroup{
		{
			Title: "Resource",
			Commands: []cliapp.Command{
				{Name: "status", Description: "Show resource status", Run: passthrough(legacy, "status")},
				{Name: "info", Description: "Show runtime info", Run: passthrough(legacy, "info")},
				{Name: "logs", Description: "Logs (serverless no-op)", Run: passthrough(legacy, "logs")},
				{Name: "manage", Description: "install|uninstall|start|stop|restart", Run: passthrough(legacy, "manage")},
			},
		},
		{
			Title: "Content",
			Commands: []cliapp.Command{
				{Name: "content", Description: "create|execute|list|get|backup|restore|remove|batch|import_csv|export_csv|encrypt|decrypt", Run: passthrough(legacy, "content")},
			},
		},
		{
			Title: "Replication",
			Commands: []cliapp.Command{
				{Name: "replicate", Description: "add|remove|list|sync|verify|toggle", Run: passthrough(legacy, "replicate")},
			},
		},
		{
			Title: "Migrations",
			Commands: []cliapp.Command{
				{Name: "migrate", Description: "init|create|up|status", Run: passthrough(legacy, "migrate")},
			},
		},
		{
			Title: "Query",
			Commands: []cliapp.Command{
				{Name: "query", Description: "select|insert|update helpers", Run: passthrough(legacy, "query")},
			},
		},
		{
			Title: "Stats",
			Commands: []cliapp.Command{
				{Name: "stats", Description: "enable|show|analyze|vacuum", Run: passthrough(legacy, "stats")},
			},
		},
		{
			Title: "Testing",
			Commands: []cliapp.Command{
				{Name: "test", Description: "smoke|integration|unit (go test ./...)", Run: passthrough(legacy, "test")},
			},
		},
	}

	return cliapp.NewApp(cliapp.AppOptions{
		Name:         info.Name,
		Version:      info.Version,
		Description:  info.Description,
		Commands:     commands,
		StaleChecker: stale,
		Preflight: func(cmd cliapp.Command, global cliapp.GlobalOptions) error {
			if stale != nil && stale.CheckAndMaybeRebuild() {
				return exitError(0)
			}
			return nil
		},
	})
}

func passthrough(legacy *CLI, name string) func(args []string) error {
	return func(args []string) error {
		code := legacy.Run(append([]string{name}, args...))
		if code != 0 {
			return exitError(code)
		}
		return nil
	}
}
