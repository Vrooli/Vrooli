package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
)

type (
	scenarioPortSingleOutput = scenariocli.PortSingleOutput
	scenarioPortListOutput   = scenariocli.PortListOutput
	scenarioOpenOutput       = scenariocli.OpenOutput
	scenarioBatchFailure     = scenariocli.BatchFailure
)

type scenarioRequirementsRequest struct {
	Snapshot bool
	Args     []string
}

type scenarioHealFromSandboxRequest struct {
	MergedPath string
	DryRun     bool
}

type scenarioHealFromSandboxResponse struct {
	Affected     []string
	DryRun       bool
	StoppedCount int
}

func runScenarioRunCommandForRoot(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	app, ctx := newConfiguredCommandContext(root, globals, stdout, stderr)
	return runScenarioStartCommandWithApp(app, ctx, args)
}

func runScenarioSetupCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeCommandAction(app, ctx, args, commandAction[scenarioSetupRequest, lifecycle.PhaseResult]{
		parse:  parseScenarioSetupRequestFromContext,
		run:    runScenarioSetupRequest,
		render: renderScenarioSetupResponse,
	})
}

func runScenarioTestCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeCommandAction(app, ctx, args, commandAction[scenarioTestRequest, struct{}]{
		parse:  parseScenarioTestRequestFromContext,
		run:    runScenarioTestRequest,
		render: renderScenarioTestResponse,
	})
}

func runScenarioStartAllCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeCommandAction(app, ctx, args, commandAction[scenarioStartAllRequest, scenarioBatchResponse]{
		parse:  parseScenarioStartAllRequestFromContext,
		run:    runScenarioStartAllRequest,
		render: renderScenarioBatchResponse,
	})
}

func runScenarioStopAllCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeCommandAction(app, ctx, args, commandAction[scenarioStopAllRequest, scenarioBatchResponse]{
		parse:  parseScenarioStopAllRequestFromContext,
		run:    runScenarioStopAllRequest,
		render: renderScenarioBatchResponse,
	})
}

func runScenarioPortCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeCommandAction(app, ctx, args, commandAction[scenarioPortRequest, scenarioPortResponse]{
		parse:  parseScenarioPortRequestFromContext,
		run:    runScenarioPortRequest,
		render: renderScenarioPortResponse,
	})
}

func runScenarioOpenCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeCommandAction(app, ctx, args, commandAction[scenarioOpenRequest, scenarioOpenOutput]{
		parse:  parseScenarioOpenRequestFromContext,
		run:    runScenarioOpenRequest,
		render: renderScenarioOpenResponse,
	})
}

func runScenarioUISmokeCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return app.runScenarioTestGenieCommand(ctx, buildUISmokeArgs(ctx.Globals, args))
}

func runScenarioCompletenessCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return app.runScenarioCompletenessSubprocessCommand(ctx, buildScenarioCompletenessArgs(ctx.Globals, args))
}

func runScenarioRequirementsCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeCommandAction(app, ctx, args, commandAction[scenarioRequirementsRequest, struct{}]{
		parse:  parseScenarioRequirementsRequestFromContext,
		run:    runScenarioRequirementsRequest,
		render: renderScenarioRequirementsResponse,
	})
}

func runScenarioHealFromSandboxCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeCommandAction(app, ctx, args, commandAction[scenarioHealFromSandboxRequest, scenarioHealFromSandboxResponse]{
		parse:  parseScenarioHealFromSandboxRequestFromContext,
		run:    runScenarioHealFromSandboxRequest,
		render: renderScenarioHealFromSandboxResponse,
	})
}

func parseScenarioRequirementsRequestFromContext(ctx *commandContext, args []string) (scenarioRequirementsRequest, error) {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		return scenarioRequirementsRequest{}, commandHelpOnly(scenarioRequirementsHelpText())
	}
	req := scenarioRequirementsRequest{Args: append([]string(nil), args...)}
	if args[0] == "snapshot" {
		req.Snapshot = true
	}
	return req, nil
}

func runScenarioRequirementsRequest(app *App, ctx *commandContext, req scenarioRequirementsRequest) (cliout.Format, struct{}, error) {
	if req.Snapshot {
		return cliout.FormatHuman, struct{}{}, runScenarioRequirementsSnapshot(ctx.Root, req.Args[1:], ctx.Stdout)
	}

	home, err := ctx.HomeDir()
	if err != nil {
		return "", struct{}{}, err
	}
	cliPath, err := app.locateTestGenieCLI(ctx.Root, home)
	if err != nil {
		return "", struct{}{}, err
	}

	commandArgs, workdir, err := translateScenarioRequirementsArgs(ctx.Root, ctx.Globals, req.Args)
	if err != nil {
		return "", struct{}{}, err
	}
	if err := app.runScenarioExternalCommand(ctx, scenarioExternalCommandSpec{
		name: cliPath,
		args: commandArgs,
		dir:  workdir,
	}); err != nil {
		return "", struct{}{}, err
	}
	return cliout.FormatHuman, struct{}{}, nil
}

func renderScenarioRequirementsResponse(w io.Writer, format cliout.Format, resp struct{}) error {
	return nil
}

func parseScenarioHealFromSandboxRequestFromContext(ctx *commandContext, args []string) (scenarioHealFromSandboxRequest, error) {
	req := scenarioHealFromSandboxRequest{
		MergedPath: strings.TrimSpace(os.Getenv("SANDBOX_MERGED_DIR")),
	}
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--merged-path":
			if index+1 >= len(args) {
				return scenarioHealFromSandboxRequest{}, usageErrorf("scenario heal-from-sandbox", "scenario heal-from-sandbox --merged-path requires a value")
			}
			index++
			req.MergedPath = args[index]
		case "--dry-run":
			req.DryRun = true
		case "--help", "-h":
			return scenarioHealFromSandboxRequest{}, commandHelpOnly("Usage: vrooli scenario heal-from-sandbox [--merged-path <path>] [--dry-run]")
		default:
			return scenarioHealFromSandboxRequest{}, unknownOptionError("scenario heal-from-sandbox", args[index])
		}
	}
	if strings.TrimSpace(req.MergedPath) == "" {
		return scenarioHealFromSandboxRequest{}, usageErrorf("scenario heal-from-sandbox", "heal-from-sandbox requires SANDBOX_MERGED_DIR or --merged-path")
	}
	return req, nil
}

func runScenarioHealFromSandboxRequest(app *App, ctx *commandContext, req scenarioHealFromSandboxRequest) (cliout.Format, scenarioHealFromSandboxResponse, error) {
	home, err := ctx.HomeDir()
	if err != nil {
		return "", scenarioHealFromSandboxResponse{}, err
	}
	affected, err := orchestrator.SandboxAffectedScenarios(home, req.MergedPath)
	if err != nil {
		return "", scenarioHealFromSandboxResponse{}, err
	}
	resp := scenarioHealFromSandboxResponse{
		Affected: append([]string(nil), affected...),
		DryRun:   req.DryRun,
	}
	if len(affected) == 0 || req.DryRun {
		return cliout.FormatHuman, resp, nil
	}

	runner, err := app.newScenarioLifecycleRunner(ctx)
	if err != nil {
		return "", scenarioHealFromSandboxResponse{}, err
	}
	for _, name := range affected {
		if stopErr := runner.Stop(name, lifecycle.StopOptions{}); stopErr != nil {
			_, _ = fmt.Fprintf(ctx.Stderr, "heal-from-sandbox: stop %s failed: %v\n", name, stopErr)
		}
	}
	time.Sleep(1 * time.Second)
	for _, name := range affected {
		if startErr := app.launchDetachedScenario(ctx.Root, ctx.Globals, "start", name); startErr != nil {
			return "", scenarioHealFromSandboxResponse{}, startErr
		}
		resp.StoppedCount++
	}
	return cliout.FormatHuman, resp, nil
}

func renderScenarioHealFromSandboxResponse(w io.Writer, format cliout.Format, resp scenarioHealFromSandboxResponse) error {
	if len(resp.Affected) == 0 {
		return nil
	}
	if resp.DryRun {
		_, _ = fmt.Fprintf(w, "heal-from-sandbox: dry-run mode, would stop and restart: %s\n", strings.Join(resp.Affected, ", "))
		return nil
	}
	_, _ = fmt.Fprintf(w, "heal-from-sandbox: stopped and relaunched %d scenario(s)\n", len(resp.Affected))
	return nil
}

func showScenarioRequirementsHelp(w io.Writer) {
	_, _ = fmt.Fprint(w, scenarioRequirementsHelpText())
}

func scenarioRequirementsHelpText() string {
	return "Usage: vrooli scenario requirements <subcommand> [options]\n\nSubcommands:\n  report <name> [options]          Generate requirement coverage summary\n  validate <name> [--quiet]        Validate requirement files\n  sync <name>                      Sync requirement statuses from local evidence\n  manual-log <name> <req> [opts]   Record manual validation evidence\n  snapshot <name>                  Show latest requirements sync snapshot\n  lint-prd <name> [--json]         Check PRD to requirements mapping\n  phase <name> --phase <phase>     Inspect validations for a single phase\n  init <name> [options]            Scaffold a requirements registry\n"
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
			if requiresOptionValue(arg) && index+1 < len(args) {
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

func requiresOptionValue(flag string) bool {
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

func parseScenarioPhaseArgs(command string, args []string) (string, lifecycle.PhaseOptions, error) {
	name := ""
	opts := lifecycle.PhaseOptions{}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch arg {
		case "--path":
			if index+1 >= len(args) {
				return "", lifecycle.PhaseOptions{}, usageErrorf("scenario "+command, "scenario %s --path requires a value", command)
			}
			index++
			opts.CustomPath = args[index]
		default:
			if strings.HasPrefix(arg, "-") {
				opts.Args = append(opts.Args, arg)
				continue
			}
			if name == "" {
				name = arg
			} else {
				opts.Args = append(opts.Args, arg)
			}
		}
	}
	if name == "" {
		return "", lifecycle.PhaseOptions{}, usageErrorf("scenario "+command, "scenario %s requires a scenario name", command)
	}
	return name, opts, nil
}

func parseScenarioTestArgs(globals globalOptions, args []string) (string, lifecycle.PhaseOptions, error) {
	name := ""
	selection := ""
	opts := lifecycle.PhaseOptions{}
	remaining := []string{}

	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch arg {
		case "--path":
			if index+1 >= len(args) {
				return "", lifecycle.PhaseOptions{}, usageErrorf("scenario test", "scenario test --path requires a value")
			}
			index++
			opts.CustomPath = args[index]
		case "--allow-skip-missing-runtime":
			opts.AllowSkipMissingRuntime = true
		case "--manage-runtime":
			opts.ManageRuntime = true
		default:
			if strings.HasPrefix(arg, "-") {
				remaining = append(remaining, arg)
				continue
			}
			if name == "" {
				name = arg
			} else if selection == "" {
				selection = arg
			} else {
				remaining = append(remaining, arg)
			}
		}
	}

	if name == "" {
		return "", lifecycle.PhaseOptions{}, usageErrorf("scenario test", "scenario test requires a scenario name")
	}
	if selection != "" {
		valid := map[string]string{
			"structure":    "structure",
			"dependencies": "dependencies",
			"unit":         "unit",
			"integration":  "integration",
			"business":     "business",
			"performance":  "performance",
			"all":          "all",
			"e2e":          "integration",
		}
		mapped, ok := valid[selection]
		if !ok {
			return "", lifecycle.PhaseOptions{}, usageErrorf("scenario test", "invalid test selector: %s", selection)
		}
		remaining = append([]string{mapped}, remaining...)
	}
	if globals.json && !containsArg(remaining, "--json") {
		remaining = append(remaining, "--json")
	}
	if globals.verbose && !containsArg(remaining, "--verbose") {
		remaining = append(remaining, "--verbose")
	}
	opts.Args = remaining
	return name, opts, nil
}

func loadScenarioPorts(root, name string) (scenario.Scenario, process.ScenarioRuntime, []scenarioListPortOutput, map[string]int, error) {
	app, ctx := newConfiguredCommandContext(root, globalOptions{}, io.Discard, io.Discard)
	service, err := app.newScenarioService(ctx)
	if err != nil {
		return scenario.Scenario{}, process.ScenarioRuntime{}, nil, nil, err
	}
	detail, err := service.Detail(name)
	if err != nil {
		return scenario.Scenario{}, process.ScenarioRuntime{}, nil, nil, err
	}
	item := detail.Scenario
	runtimeState := detail.Runtime
	listPorts, portsMap := buildListPorts(item.Manifest, runtimeState.Records)
	seen := make(map[string]struct{}, len(listPorts))
	for _, item := range listPorts {
		seen[item.Key] = struct{}{}
	}
	for key, port := range portsMap {
		if _, ok := seen[key]; ok {
			continue
		}
		listPorts = append(listPorts, scenarioListPortOutput{Key: key, Port: port})
	}
	sort.Slice(listPorts, func(i, j int) bool {
		if listPorts[i].Key == listPorts[j].Key {
			return listPorts[i].Step < listPorts[j].Step
		}
		return listPorts[i].Key < listPorts[j].Key
	})
	return item, runtimeState, listPorts, portsMap, nil
}

func resolveRequestedPort(manifest scenario.ServiceManifest, listPorts []scenarioListPortOutput, portsMap map[string]int, requested string) (string, int, string, bool) {
	candidates := []string{requested}
	if envVar := manifest.PortEnvVar(strings.ToLower(strings.TrimSuffix(requested, "_PORT"))); envVar != "" {
		candidates = append(candidates, envVar)
	}
	normalized := strings.ToUpper(strings.TrimSpace(requested))
	if normalized != "" && normalized != requested {
		candidates = append(candidates, normalized)
		if !strings.HasSuffix(normalized, "_PORT") {
			candidates = append(candidates, normalized+"_PORT")
		}
	}

	seen := map[string]struct{}{}
	for _, key := range candidates {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if portValue, ok := portsMap[key]; ok {
			stepName := ""
			for _, entry := range listPorts {
				if entry.Key == key && entry.Port == portValue {
					stepName = entry.Step
					break
				}
			}
			return key, portValue, stepName, true
		}
	}
	return "", 0, "", false
}

func scenarioURLForPort(root, scenarioName, portName string) (string, string, int, error) {
	app, ctx := newConfiguredCommandContext(root, globalOptions{}, io.Discard, io.Discard)
	service, err := app.newScenarioService(ctx)
	if err != nil {
		return "", "", 0, err
	}
	resolved, err := service.ResolvePort(scenarioName, portName)
	if err != nil {
		return "", "", 0, err
	}
	return resolved.URL, resolved.Name, resolved.Port, nil
}

func envPortMap(manifest scenario.ServiceManifest, ports map[string]int) map[string]int {
	out := make(map[string]int, len(ports))
	for portName, port := range ports {
		envVar := manifest.PortEnvVar(portName)
		if envVar == "" {
			envVar = strings.ToUpper(strings.ReplaceAll(portName, "-", "_")) + "_PORT"
		}
		out[envVar] = port
	}
	return out
}
