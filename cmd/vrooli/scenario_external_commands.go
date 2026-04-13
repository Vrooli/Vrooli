package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type scenarioExternalCommandSpec struct {
	name string
	args []string
	dir  string
}

func (app *App) runScenarioExternalCommand(ctx *commandContext, spec scenarioExternalCommandSpec) error {
	return app.runScenarioSubprocess(scenarioSubprocessSpec{
		name:   spec.name,
		args:   spec.args,
		dir:    spec.dir,
		env:    app.commandEnv(ctx.Root, ctx.Globals),
		stdout: ctx.Stdout,
		stderr: ctx.Stderr,
	})
}

func (app *App) runScenarioBuiltExternalCommand(
	ctx *commandContext,
	args []string,
	build func(ctx *commandContext, args []string) (scenarioExternalCommandSpec, error),
) error {
	spec, err := build(ctx, args)
	if err != nil {
		return err
	}
	return app.runScenarioExternalCommand(ctx, spec)
}

func (app *App) runScenarioTestGenieCommand(ctx *commandContext, args []string) error {
	return app.runScenarioBuiltExternalCommand(ctx, args, func(ctx *commandContext, args []string) (scenarioExternalCommandSpec, error) {
		home, err := ctx.HomeDir()
		if err != nil {
			return scenarioExternalCommandSpec{}, err
		}
		cliPath, err := app.locateTestGenieCLI(ctx.Root, home)
		if err != nil {
			return scenarioExternalCommandSpec{}, err
		}
		return scenarioExternalCommandSpec{
			name: cliPath,
			args: args,
			dir:  ctx.Root,
		}, nil
	})
}

func (app *App) runScenarioCompletenessSubprocessCommand(ctx *commandContext, args []string) error {
	return app.runScenarioBuiltExternalCommand(ctx, args, func(ctx *commandContext, args []string) (scenarioExternalCommandSpec, error) {
		cliPath, err := app.locateScenarioCompletenessCLI(ctx.Root)
		if err != nil {
			return scenarioExternalCommandSpec{}, err
		}
		return scenarioExternalCommandSpec{
			name: cliPath,
			args: args,
			dir:  ctx.Root,
		}, nil
	})
}

func buildUISmokeArgs(globals globalOptions, args []string) []string {
	commandArgs := []string{"ui-smoke"}
	commandArgs = append(commandArgs, args...)
	return append(commandArgs, passthroughFlags(globals, commandArgs)...)
}

func buildScenarioCompletenessArgs(globals globalOptions, args []string) []string {
	commandArgs := append([]string{}, args...)
	if globals.json && !containsArg(commandArgs, "--json") && !containsArg(commandArgs, "--format") {
		commandArgs = append(commandArgs, "--json")
	}
	return commandArgs
}

func translateScenarioRequirementsArgs(root string, globals globalOptions, args []string) ([]string, string, error) {
	known := map[string]struct{}{
		"report": {}, "validate": {}, "sync": {}, "manual-log": {}, "lint-prd": {}, "phase": {}, "phase-inspect": {}, "init": {},
	}

	subcommand := args[0]
	rest := args[1:]
	if _, ok := known[subcommand]; !ok {
		subcommand = "report"
		rest = args
	}

	switch subcommand {
	case "report":
		return translateScenarioRequirementsSimple(root, globals, "report", rest, false, true)
	case "validate":
		return translateScenarioRequirementsSimple(root, globals, "validate", rest, true, true)
	case "sync":
		return translateScenarioRequirementsSimple(root, globals, "sync", rest, true, false)
	case "lint-prd":
		return translateScenarioRequirementsSimple(root, globals, "lint-prd", rest, true, false)
	case "init":
		return translateScenarioRequirementsSimple(root, globals, "init", rest, true, false)
	case "phase", "phase-inspect":
		return translateScenarioRequirementsSimple(root, globals, "phase", rest, true, false)
	case "manual-log":
		return translateScenarioRequirementsSimple(root, globals, "manual-log", rest, true, false)
	default:
		return nil, "", usageErrorf("scenario requirements", "unsupported requirements subcommand: %s", subcommand)
	}
}

func translateScenarioRequirementsSimple(root string, globals globalOptions, subcommand string, args []string, includeScenario bool, includeJSON bool) ([]string, string, error) {
	scenarioName := ""
	translated := []string{"requirements", subcommand}

	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "--help" || arg == "-h":
			translated = append(translated, arg)
		case strings.HasPrefix(arg, "-"):
			translated = append(translated, arg)
			if requiresScenarioRequirementsOptionValue(arg) && index+1 < len(args) {
				index++
				translated = append(translated, args[index])
			}
		case scenarioName == "":
			scenarioName = arg
		default:
			translated = append(translated, arg)
		}
	}

	if containsArg(translated, "--help") || containsArg(translated, "-h") {
		return translated, root, nil
	}
	if scenarioName == "" {
		return nil, "", usageErrorf("scenario requirements", "scenario requirements %s requires a scenario name", subcommand)
	}

	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	if info, err := os.Stat(scenarioDir); err != nil || !info.IsDir() {
		return nil, "", usageErrorf("scenario requirements", "scenario directory not found: %s", scenarioDir)
	}

	if subcommand != "init" {
		if info, err := os.Stat(filepath.Join(scenarioDir, "requirements")); err != nil || !info.IsDir() {
			return nil, "", usageErrorf("scenario requirements", "scenario %s does not define requirements/", scenarioName)
		}
	}

	translated = append(translated, "--dir", scenarioDir)
	if includeScenario {
		translated = append(translated, "--scenario", scenarioName)
	}
	if includeJSON && globals.json && !containsArg(translated, "--json") {
		translated = append(translated, "--json")
	}
	return translated, scenarioDir, nil
}

func requiresScenarioRequirementsOptionValue(flag string) bool {
	switch flag {
	case "--format", "--output", "--status", "--notes", "--artifact", "--validated-by", "--validated-at", "--expires-in", "--expires-at", "--manifest", "--phase", "--template", "--owner":
		return true
	default:
		return false
	}
}

func runScenarioRequirementsSnapshot(root string, args []string, stdout io.Writer) error {
	scenarioName := ""
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			_, _ = fmt.Fprintln(stdout, "Usage: vrooli scenario requirements snapshot <name>")
			return nil
		default:
			if strings.HasPrefix(arg, "-") {
				return unknownOptionError("scenario requirements snapshot", arg)
			}
			if scenarioName != "" {
				return usageErrorf("scenario requirements snapshot", "scenario requirements snapshot accepts exactly one scenario name")
			}
			scenarioName = arg
		}
	}
	if scenarioName == "" {
		return usageErrorf("scenario requirements snapshot", "scenario requirements snapshot requires a scenario name")
	}

	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	snapshotPath := filepath.Join(scenarioDir, "coverage", "requirements-sync", "latest.json")
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("snapshot not found: %s", snapshotPath)
		}
		return err
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(stdout, "Requirements snapshot (%s)\n", scenarioName)
	_, _ = fmt.Fprintf(stdout, "  File: %s\n", filepath.Join("coverage", "requirements-sync", "latest.json"))
	if syncedAt, _ := payload["synced_at"].(string); syncedAt != "" {
		_, _ = fmt.Fprintf(stdout, "  Synced at: %s\n", syncedAt)
	}
	if tests, ok := payload["tests_run"].([]any); ok && len(tests) > 0 {
		_, _ = fmt.Fprintln(stdout, "  Tests run:")
		for _, test := range tests {
			if command, ok := test.(string); ok && strings.TrimSpace(command) != "" {
				_, _ = fmt.Fprintf(stdout, "    - %s\n", command)
			}
		}
	}
	return nil
}
