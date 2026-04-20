package scenariohandlers

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	. "github.com/vrooli/vrooli/internal/cli/scenariocli" //nolint:revive // scenariohandlers is a thin glue layer over scenariocli; dot-import keeps wiring readable.
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/scenarioexec"
)

func UISmokeHandler[C any](deps HandlerDeps[C]) func(C, []string) error {
	return func(ctx C, args []string) error {
		cliPath, err := deps.LocateTestGenieCLI(ctx)
		if err != nil {
			return err
		}
		emitScenarioStaleWarning(deps.Stderr(ctx), deps.Root(ctx), extractUISmokeScenario(args), deps.Globals(ctx))
		commandArgs := BuildUISmokeArgs(deps.Globals(ctx), args)
		return deps.RunSubprocess(ctx, scenarioexec.SubprocessSpec{
			Name:   cliPath,
			Args:   commandArgs,
			Dir:    deps.Root(ctx),
			Env:    deps.CommandEnv(ctx),
			Stdout: deps.Stdout(ctx),
			Stderr: deps.Stderr(ctx),
		})
	}
}

func BuildUISmokeArgs(globals rootcli.GlobalOptions, args []string) []string {
	commandArgs := []string{"ui-smoke"}
	commandArgs = append(commandArgs, args...)
	return append(commandArgs, rootcli.PassthroughFlags(globals, commandArgs)...)
}

func CompletenessHandler[C any](deps HandlerDeps[C]) func(C, []string) error {
	return func(ctx C, args []string) error {
		cliPath, err := deps.LocateCompleteCLI(ctx)
		if err != nil {
			return err
		}
		emitScenarioStaleWarning(deps.Stderr(ctx), deps.Root(ctx), extractCompletenessScenario(args), deps.Globals(ctx))
		commandArgs := BuildScenarioCompletenessArgs(deps.Globals(ctx), args)
		return deps.RunSubprocess(ctx, scenarioexec.SubprocessSpec{
			Name:   cliPath,
			Args:   commandArgs,
			Dir:    deps.Root(ctx),
			Env:    deps.CommandEnv(ctx),
			Stdout: deps.Stdout(ctx),
			Stderr: deps.Stderr(ctx),
		})
	}
}

// BuildScenarioCompletenessArgs forwards the user args to the scenario's
// cli-core-based CLI, re-emitting the global flags that vrooli consumed at its
// own arg layer. cli-core requires ANSI/color globals (`--no-color`,
// `--color`) to appear BEFORE the subcommand name; subcommand-level flags
// (`--json`, `--verbose`) are appended after the user args.
func BuildScenarioCompletenessArgs(globals rootcli.GlobalOptions, args []string) []string {
	prefix := make([]string, 0, 2)
	if globals.NoColor && !rootcli.ContainsArg(args, "--no-color") {
		prefix = append(prefix, "--no-color")
	}

	commandArgs := make([]string, 0, len(prefix)+len(args)+2)
	commandArgs = append(commandArgs, prefix...)
	commandArgs = append(commandArgs, args...)

	if globals.JSON && !rootcli.ContainsFlag(commandArgs, "--json") && !rootcli.ContainsFlag(commandArgs, "--format") {
		commandArgs = append(commandArgs, "--json")
	}
	if globals.Verbose && !rootcli.ContainsFlag(commandArgs, "--verbose") && !rootcli.ContainsArg(commandArgs, "-v") {
		commandArgs = append(commandArgs, "--verbose")
	}
	return commandArgs
}

func RequirementsHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	return bindGlobal(deps.Stdout,
		func(ctx C, args []string) (RequirementsRequest, error) { return ParseRequirementsRequest(args) },
		func(ctx C, req RequirementsRequest) (cliout.Format, struct{}, error) {
			if req.Snapshot {
				return cliout.FormatHuman, struct{}{}, runScenarioRequirementsSnapshot(deps.Root(ctx), req.Args[1:], deps.Stdout(ctx))
			}
			cliPath, err := deps.LocateTestGenieCLI(ctx)
			if err != nil {
				return "", struct{}{}, err
			}
			commandArgs, workdir, err := buildScenarioRequirementsCommand(deps.Root(ctx), deps.Globals(ctx), req.Args)
			if err != nil {
				return "", struct{}{}, err
			}
			err = deps.RunSubprocess(ctx, scenarioexec.SubprocessSpec{
				Name:   cliPath,
				Args:   commandArgs,
				Dir:    workdir,
				Env:    deps.CommandEnv(ctx),
				Stdout: deps.Stdout(ctx),
				Stderr: deps.Stderr(ctx),
			})
			return cliout.FormatHuman, struct{}{}, err
		},
		func(w io.Writer, _ cliout.Format, _ struct{}) error { return nil },
	)
}

func buildScenarioRequirementsCommand(root string, globals rootcli.GlobalOptions, args []string) ([]string, string, error) {
	known := map[string]struct{}{"report": {}, "validate": {}, "sync": {}, "manual-log": {}, "lint-prd": {}, "phase": {}, "phase-inspect": {}, "init": {}}
	subcommand := args[0]
	rest := args[1:]
	if _, ok := known[subcommand]; !ok {
		subcommand = "report"
		rest = args
	}
	switch subcommand {
	case "report":
		return buildScenarioRequirementsSubcommand(root, globals, "report", rest, false, true)
	case "validate":
		return buildScenarioRequirementsSubcommand(root, globals, "validate", rest, true, true)
	case "sync":
		return buildScenarioRequirementsSubcommand(root, globals, "sync", rest, true, false)
	case "lint-prd":
		return buildScenarioRequirementsSubcommand(root, globals, "lint-prd", rest, true, false)
	case "init":
		return buildScenarioRequirementsSubcommand(root, globals, "init", rest, true, false)
	case "phase", "phase-inspect":
		return buildScenarioRequirementsSubcommand(root, globals, "phase", rest, true, false)
	case "manual-log":
		return buildScenarioRequirementsSubcommand(root, globals, "manual-log", rest, true, false)
	default:
		return nil, "", rootcli.UsageErrorf("scenario requirements", "unsupported requirements subcommand: %s", subcommand)
	}
}

func buildScenarioRequirementsSubcommand(root string, globals rootcli.GlobalOptions, subcommand string, args []string, includeScenario, includeJSON bool) ([]string, string, error) {
	scenarioName := ""
	commandArgs := []string{"requirements", subcommand}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "--help" || arg == "-h":
			commandArgs = append(commandArgs, arg)
		case strings.HasPrefix(arg, "-"):
			commandArgs = append(commandArgs, arg)
			if requiresScenarioRequirementsOptionValue(arg) && index+1 < len(args) {
				index++
				commandArgs = append(commandArgs, args[index])
			}
		case scenarioName == "":
			scenarioName = arg
		default:
			commandArgs = append(commandArgs, arg)
		}
	}
	if rootcli.ContainsArg(commandArgs, "--help") || rootcli.ContainsArg(commandArgs, "-h") {
		return commandArgs, root, nil
	}
	if scenarioName == "" {
		return nil, "", rootcli.UsageErrorf("scenario requirements", "scenario requirements %s requires a scenario name", subcommand)
	}
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	if info, err := os.Stat(scenarioDir); err != nil || !info.IsDir() {
		return nil, "", rootcli.UsageErrorf("scenario requirements", "scenario directory not found: %s", scenarioDir)
	}
	if subcommand != "init" {
		if info, err := os.Stat(filepath.Join(scenarioDir, "requirements")); err != nil || !info.IsDir() {
			return nil, "", rootcli.UsageErrorf("scenario requirements", "scenario %s does not define requirements/", scenarioName)
		}
	}
	commandArgs = append(commandArgs, "--dir", scenarioDir)
	if includeScenario {
		commandArgs = append(commandArgs, "--scenario", scenarioName)
	}
	if includeJSON && globals.JSON && !rootcli.ContainsArg(commandArgs, "--json") {
		commandArgs = append(commandArgs, "--json")
	}
	return commandArgs, scenarioDir, nil
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
			commandtree.WriteHelp(stdout, RequirementsSnapshotHelpText())
			return nil
		default:
			if strings.HasPrefix(arg, "-") {
				return rootcli.UnknownOptionError("scenario requirements snapshot", arg)
			}
			if scenarioName != "" {
				return rootcli.UsageErrorf("scenario requirements snapshot", "scenario requirements snapshot accepts exactly one scenario name")
			}
			scenarioName = arg
		}
	}
	if scenarioName == "" {
		return rootcli.UsageErrorf("scenario requirements snapshot", "scenario requirements snapshot requires a scenario name")
	}
	snapshotPath := filepath.Join(root, "scenarios", scenarioName, "coverage", "requirements-sync", "latest.json")
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
