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
	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/scenarioexec"
)

const (
	toolRuntimeHelp = "--help"
)

const (
	toolRuntimeInit = "init"
)

const (
	toolRuntimeParameterA = 2
)

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
	prefix := make([]string, 0, toolRuntimeParameterA)
	if globals.NoColor && !rootcli.ContainsArg(args, "--no-color") {
		prefix = append(prefix, "--no-color")
	}

	commandArgs := make([]string, 0, len(prefix)+len(args)+toolRuntimeParameterA)
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

// RequirementsHandler routes the `vrooli scenario requirements` facade.
// Contract-side verbs (validate, report, lint-prd, drift, phase, init,
// manual-log) route to business-health, which owns the business contract;
// the run-coupled verbs stay with test-genie (`sync`, the evidence writer)
// or read the artifact directly (`snapshot`).
func RequirementsHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	return scenarioServiceCommand(deps.Stdout,
		func(ctx C, args []string) (RequirementsRequest, error) { return ParseRequirementsRequest(args) },
		func(ctx C, req RequirementsRequest) (cliout.Format, struct{}, error) {
			if req.Snapshot {
				return cliout.FormatHuman, struct{}{}, runScenarioRequirementsSnapshot(deps.Root(ctx), req.Args[1:], deps.Stdout(ctx))
			}
			route, err := buildScenarioRequirementsRoute(deps.Root(ctx), deps.Globals(ctx), req.Args)
			if err != nil {
				return "", struct{}{}, err
			}
			var cliPath string
			if route.testGenie {
				cliPath, err = deps.LocateTestGenieCLI(ctx)
			} else {
				if deps.LocateBusinessHealthCLI == nil {
					return "", struct{}{}, fmt.Errorf("business-health CLI locator is not wired")
				}
				cliPath, err = deps.LocateBusinessHealthCLI(ctx)
			}
			if err != nil {
				return "", struct{}{}, err
			}
			err = deps.RunSubprocess(ctx, scenarioexec.SubprocessSpec{
				Name:   cliPath,
				Args:   route.args,
				Dir:    route.workdir,
				Env:    deps.CommandEnv(ctx),
				Stdout: deps.Stdout(ctx),
				Stderr: deps.Stderr(ctx),
			})
			return cliout.FormatHuman, struct{}{}, err
		},
		func(w io.Writer, _ cliout.Format, _ struct{}) error { return nil },
	)
}

// requirementsRoute is one resolved facade dispatch.
type requirementsRoute struct {
	// testGenie selects the test-genie binary (sync); false = business-health.
	testGenie bool
	args      []string
	workdir   string
}

// buildScenarioRequirementsRoute maps the legacy verb surface onto the two
// owners. UX stays `vrooli scenario requirements <verb> <scenario> [flags]`.
func buildScenarioRequirementsRoute(root string, globals rootcli.GlobalOptions, args []string) (requirementsRoute, error) {
	known := map[string]struct{}{"report": {}, "validate": {}, "sync": {}, "manual-log": {}, "lint-prd": {}, "phase": {}, "phase-inspect": {}, "drift": {}, toolRuntimeInit: {}}
	subcommand := args[0]
	rest := args[1:]
	if _, ok := known[subcommand]; !ok {
		subcommand = "report"
		rest = args
	}
	if subcommand == "sync" {
		commandArgs, workdir, err := buildScenarioRequirementsSubcommand(root, globals, "sync", rest, true, false)
		return requirementsRoute{testGenie: true, args: commandArgs, workdir: workdir}, err
	}
	return buildBusinessHealthRequirementsRoute(root, globals, subcommand, rest)
}

// buildBusinessHealthRequirementsRoute translates one contract verb into
// the business-health CLI surface.
func buildBusinessHealthRequirementsRoute(root string, globals rootcli.GlobalOptions, subcommand string, args []string) (requirementsRoute, error) {
	scenarioName := ""
	var positionals, flags []string
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == toolRuntimeHelp || arg == "-h":
			flags = append(flags, arg)
		case strings.HasPrefix(arg, "-"):
			flags = append(flags, arg)
			if requiresScenarioRequirementsOptionValue(arg) && index+1 < len(args) {
				index++
				flags = append(flags, args[index])
			}
		case scenarioName == "":
			scenarioName = arg
		default:
			positionals = append(positionals, arg)
		}
	}
	if rootcli.ContainsArg(flags, toolRuntimeHelp) || rootcli.ContainsArg(flags, "-h") {
		return requirementsRoute{args: append(requirementsVerbCommand(subcommand, "", nil, nil), toolRuntimeHelp), workdir: root}, nil
	}
	if scenarioName == "" {
		return requirementsRoute{}, rootcli.UsageErrorf("scenario requirements", "scenario requirements %s requires a scenario name", subcommand)
	}
	scenarioDir := filepath.Join(root, repocontractmeta.ScenarioDir, scenarioName)
	if info, err := os.Stat(scenarioDir); err != nil || !info.IsDir() {
		return requirementsRoute{}, rootcli.UsageErrorf("scenario requirements", "scenario directory not found: %s", scenarioDir)
	}
	translated, err := translateRequirementsFlags(subcommand, flags)
	if err != nil {
		return requirementsRoute{}, err
	}
	commandArgs := requirementsVerbCommand(subcommand, scenarioName, positionals, translated)
	if commandArgs == nil {
		return requirementsRoute{}, rootcli.UsageErrorf("scenario requirements", "unsupported requirements subcommand: %s", subcommand)
	}
	if globals.JSON && !rootcli.ContainsArg(commandArgs, "--json") {
		commandArgs = append(commandArgs, "--json")
	}
	// --auto-start is a cli-core GLOBAL (must precede the command): the
	// provider answers over Connect-RPC, so bring it up when needed.
	commandArgs = append([]string{"--auto-start"}, commandArgs...)
	return requirementsRoute{args: commandArgs, workdir: root}, nil
}

// requirementsVerbCommand maps a legacy verb to the business-health CLI
// command shape. nil = unsupported verb.
func requirementsVerbCommand(subcommand, scenario string, positionals, flags []string) []string {
	withScenario := func(base ...string) []string {
		out := append([]string{}, base...)
		if scenario != "" {
			out = append(out, scenario)
		}
		out = append(out, positionals...)
		out = append(out, flags...)
		return out
	}
	switch subcommand {
	case "validate", "lint-prd":
		// lint-prd's linkage checks are part of the full contract validation.
		return withScenario("validate", "scenario")
	case "report":
		out := withScenario("matrix", "show")
		if !rootcli.ContainsFlag(out, "--format") {
			out = append(out, "--format", "summary")
		}
		return out
	case "drift":
		return withScenario("drift", "show")
	case "phase", "phase-inspect":
		out := withScenario("matrix", "show")
		if !rootcli.ContainsFlag(out, "--phase") {
			out = append(out, "--phase", "all")
		}
		return out
	case toolRuntimeInit:
		// The registry scaffold is the prd_missing_requirements fixer (the
		// wizard is the richer authoring path).
		return withScenario("fix", "apply", "--rules", "prd_missing_requirements")
	case "manual-log":
		return withScenario("manual-log", "add")
	default:
		return nil
	}
}

// translateRequirementsFlags maps legacy flag spellings onto the
// business-health surface and rejects flags whose semantics moved (expiry
// is policy-owned now, not caller-supplied).
func translateRequirementsFlags(subcommand string, flags []string) ([]string, error) {
	var out []string
	for index := 0; index < len(flags); index++ {
		flag := flags[index]
		value := ""
		hasValue := false
		if requiresScenarioRequirementsOptionValue(flag) && index+1 < len(flags) {
			index++
			value = flags[index]
			hasValue = true
		}
		switch flag {
		case "--validated-by":
			flag = "--by"
		case "--validated-at", "--expires-in", "--expires-at", "--status", "--artifact", "--manifest", "--owner", "--template":
			if subcommand == "manual-log" || subcommand == toolRuntimeInit {
				return nil, rootcli.UsageErrorf("scenario requirements", "%s no longer accepts %s: attestation time and expiry are stamped by business-health's policy (see business-health manual-log add --help)", subcommand, flag)
			}
		}
		out = append(out, flag)
		if hasValue {
			out = append(out, value)
		}
	}
	return out, nil
}

func buildScenarioRequirementsSubcommand(root string, globals rootcli.GlobalOptions, subcommand string, args []string, includeScenario, includeJSON bool) ([]string, string, error) {
	scenarioName := ""
	commandArgs := []string{"requirements", subcommand}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == toolRuntimeHelp || arg == "-h":
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
	if rootcli.ContainsArg(commandArgs, toolRuntimeHelp) || rootcli.ContainsArg(commandArgs, "-h") {
		return commandArgs, root, nil
	}
	if scenarioName == "" {
		return nil, "", rootcli.UsageErrorf("scenario requirements", "scenario requirements %s requires a scenario name", subcommand)
	}
	scenarioDir := filepath.Join(root, repocontractmeta.ScenarioDir, scenarioName)
	if info, err := os.Stat(scenarioDir); err != nil || !info.IsDir() {
		return nil, "", rootcli.UsageErrorf("scenario requirements", "scenario directory not found: %s", scenarioDir)
	}
	if subcommand != toolRuntimeInit {
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
	case "--format", "--output", "--status", "--notes", "--artifact", "--validated-by", "--validated-at", "--expires-in", "--expires-at", "--manifest", "--phase", "--template", "--owner", "--by", "--rules":
		return true
	default:
		return false
	}
}

func runScenarioRequirementsSnapshot(root string, args []string, stdout io.Writer) error {
	scenarioName := ""
	for _, arg := range args {
		switch arg {
		case toolRuntimeHelp, "-h":
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
	snapshotPath := filepath.Join(root, repocontractmeta.ScenarioDir, scenarioName, "coverage", "requirements-sync", "latest.json")
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
