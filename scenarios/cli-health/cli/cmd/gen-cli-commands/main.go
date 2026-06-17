// gen-cli-commands emits cli/cli-commands.gen.json from the live CLI
// registration tree — the single source of truth for the scenario's command
// surface. It builds the same ScenarioApp the runtime CLI builds (domain
// command groups + subcommand groups + StandardScenarioApp built-ins), then
// enumerates every registered command via cliapp.EnumerateCommands.
//
// The generated artifact is the cross-module bridge: api/cmd/gen-endpoints
// reads it as data to render the cli_commands[] section of
// .vrooli/endpoints.json, with no API↔CLI Go import.
//
// Usage (run from the cli/ directory, as `make endpoints` does):
//
//	go run ./cmd/gen-cli-commands
//
// It reads cli/manifest.json from the working directory and writes
// cli/cli-commands.gen.json. CI runs `make endpoints` and diffs the result,
// so a stale artifact fails the build with an actionable diff.
package main

import (
	"fmt"
	"os"

	"cli-health/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName    = "cli-health"
	outputPath = "cli-commands.gen.json"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "gen-cli-commands: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	manifestBytes, err := os.ReadFile("manifest.json")
	if err != nil {
		return fmt.Errorf("read manifest.json: %w", err)
	}

	app, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:           appName,
		Version:        "0.0.0",
		Description:    "cli-health CLI",
		AllowAnonymous: true,
		CommandGroups:  domains.CommandGroups,
		SubcommandGroups: func(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
			groups, err := domains.SubcommandGroups(core, manifestBytes)
			if err != nil {
				panic(err)
			}
			return groups
		},
	})
	if err != nil {
		return fmt.Errorf("build CLI app: %w", err)
	}

	return cliapp.WriteCommandsJSON(app, outputPath)
}
