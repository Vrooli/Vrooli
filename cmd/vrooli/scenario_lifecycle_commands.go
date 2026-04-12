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

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
)

type scenarioPortSingleOutput struct {
	Success  bool   `json:"success"`
	Scenario string `json:"scenario"`
	PortName string `json:"port_name"`
	Step     string `json:"step,omitempty"`
	Port     int    `json:"port,omitempty"`
	Error    string `json:"error,omitempty"`
}

type scenarioPortListOutput struct {
	Success  bool                     `json:"success"`
	Scenario string                   `json:"scenario"`
	Ports    []scenarioListPortOutput `json:"ports"`
	Metadata map[string]int           `json:"metadata,omitempty"`
	Error    string                   `json:"error,omitempty"`
}

type scenarioOpenOutput struct {
	Success  bool   `json:"success"`
	Scenario string `json:"scenario"`
	PortName string `json:"port_name"`
	Port     int    `json:"port"`
	URL      string `json:"url"`
}

type scenarioBatchFailure struct {
	Name  string `json:"name"`
	Error string `json:"error"`
}

func runScenarioRunCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	app, ctx := newConfiguredCommandContext(root, globals, stdout, stderr)
	return runScenarioStartCommandWithApp(app, ctx, args)
}

func runScenarioSetupCommandWithApp(app *App, ctx *commandContext, args []string) error {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			_, _ = fmt.Fprintln(ctx.Stdout, "Usage: vrooli scenario setup <name> [--path <path>]")
			return nil
		}
	}

	name, opts, err := parseScenarioPhaseArgs("setup", args)
	if err != nil {
		return err
	}

	runner, _, err := app.newScenarioLifecycleRunnerForFormat(ctx, false)
	if err != nil {
		return err
	}
	result, err := runner.RunPhaseDetailed(name, "setup", opts)
	if err != nil {
		return err
	}

	if ctx.Globals.json {
		return cliout.WriteJSON(ctx.Stdout, map[string]any{
			"success":  true,
			"scenario": name,
			"phase":    "setup",
			"status":   result.Status,
			"defined":  result.Defined,
			"steps": map[string]int{
				"executed": result.ExecutedSteps,
				"skipped":  result.SkippedSteps,
			},
		})
	}

	switch result.Status {
	case lifecycle.PhaseExecutionCompleted:
		_, _ = fmt.Fprintf(ctx.Stdout, "Completed setup for scenario '%s' (%d executed, %d skipped)\n", name, result.ExecutedSteps, result.SkippedSteps)
	case lifecycle.PhaseExecutionSkipped:
		_, _ = fmt.Fprintf(ctx.Stdout, "Setup phase for scenario '%s' ran no steps (%d skipped)\n", name, result.SkippedSteps)
	default:
		_, _ = fmt.Fprintf(ctx.Stdout, "Scenario '%s' does not define a setup phase\n", name)
	}
	return nil
}

func runScenarioTestCommandWithApp(app *App, ctx *commandContext, args []string) error {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			_, _ = fmt.Fprintln(ctx.Stdout, "Usage: vrooli scenario test <name> [phase|all|e2e] [--allow-skip-missing-runtime] [--manage-runtime]")
			return nil
		}
	}

	name, opts, err := parseScenarioTestArgs(ctx.Globals, args)
	if err != nil {
		return err
	}

	runner, _, err := app.newScenarioLifecycleRunnerForFormat(ctx, false)
	if err != nil {
		return err
	}
	return runner.RunPhase(name, "test", opts)
}

func runScenarioStartAllCommandWithApp(app *App, ctx *commandContext, args []string) error {
	jsonFlag := ctx.Globals.json
	if len(args) > 0 {
		for _, arg := range args {
			switch arg {
			case "--json":
				jsonFlag = true
			case "--help", "-h":
				_, _ = fmt.Fprintln(ctx.Stdout, "Usage: vrooli scenario start-all [--json]")
				return nil
			default:
				return unknownOptionError("scenario start-all", arg)
			}
		}
	}

	service, format, err := app.newScenarioServiceForFormat(ctx, jsonFlag)
	if err != nil {
		return err
	}
	report, err := service.StartAll()
	if err != nil {
		return err
	}

	started := make([]scenarioLifecycleItemOutput, 0, len(report.Started))
	for _, item := range report.Started {
		started = append(started, scenarioLifecycleItemOutput{Name: item.Name, Status: "started"})
	}
	failed := make([]scenarioBatchFailure, 0, len(report.Failed))
	for _, item := range report.Failed {
		failed = append(failed, scenarioBatchFailure{Name: item.Name, Error: item.Error})
	}

	return writeScenarioBatchReport(ctx.Stdout, format, "Started", started, nil, failed)
}

func runScenarioStopAllCommandWithApp(app *App, ctx *commandContext, args []string) error {
	jsonFlag := ctx.Globals.json
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonFlag = true
		case "--help", "-h":
			_, _ = fmt.Fprintln(ctx.Stdout, "Usage: vrooli scenario stop-all [--json]")
			return nil
		default:
			return unknownOptionError("scenario stop-all", arg)
		}
	}

	service, format, err := app.newScenarioServiceForFormat(ctx, jsonFlag)
	if err != nil {
		return err
	}
	report, err := service.StopAll()
	if err != nil {
		return err
	}

	stopped := make([]string, 0, len(report.Stopped))
	for _, item := range report.Stopped {
		stopped = append(stopped, item.Name)
	}
	failed := make([]scenarioBatchFailure, 0, len(report.Failed))
	for _, item := range report.Failed {
		failed = append(failed, scenarioBatchFailure{Name: item.Name, Error: item.Error})
	}

	return writeScenarioBatchReport(ctx.Stdout, format, "Stopped", nil, stopped, failed)
}

func runScenarioPortCommandWithApp(app *App, ctx *commandContext, args []string) error {
	scenarioName := ""
	portName := ""
	jsonFlag := ctx.Globals.json

	for _, arg := range args {
		switch {
		case arg == "--json":
			jsonFlag = true
		case arg == "--help" || arg == "-h":
			_, _ = fmt.Fprintln(ctx.Stdout, "Usage: vrooli scenario port <scenario-name> [<port-name>] [--json]")
			return nil
		case strings.HasPrefix(arg, "-"):
			return unknownOptionError("scenario port", arg)
		case scenarioName == "":
			scenarioName = arg
		case portName == "":
			portName = arg
		default:
			return usageErrorf("scenario port", "scenario port accepts at most two positional arguments")
		}
	}

	if scenarioName == "" {
		return usageErrorf("scenario port", "scenario port requires a scenario name")
	}

	serviceCtx := *ctx
	serviceCtx.Stdout = io.Discard
	serviceCtx.Stderr = io.Discard
	service, err := app.newScenarioService(&serviceCtx)
	if err != nil {
		return err
	}
	detail, err := service.Detail(scenarioName)
	if err != nil {
		return err
	}
	listPorts, portsMap := buildListPorts(detail.Scenario.Manifest, detail.Runtime.Records)

	if portName == "" {
		if detail.Runtime.ProcessCount == 0 || len(portsMap) == 0 {
			if jsonFlag {
				return cliout.WriteJSON(ctx.Stdout, scenarioPortListOutput{
					Success:  false,
					Scenario: scenarioName,
					Ports:    []scenarioListPortOutput{},
					Error:    "No running processes found for scenario",
				})
			}
			return fmt.Errorf("no running processes found for scenario %q", scenarioName)
		}
		if jsonFlag {
			return cliout.WriteJSON(ctx.Stdout, scenarioPortListOutput{
				Success:  true,
				Scenario: scenarioName,
				Ports:    listPorts,
				Metadata: map[string]int{"count": len(listPorts)},
			})
		}
		for _, port := range listPorts {
			_, _ = fmt.Fprintf(ctx.Stdout, "%s=%d\n", port.Key, port.Port)
		}
		return nil
	}

	resolved, err := service.ResolvePort(scenarioName, portName)
	if err != nil {
		if jsonFlag {
			return cliout.WriteJSON(ctx.Stdout, scenarioPortSingleOutput{
				Success:  false,
				Scenario: scenarioName,
				PortName: portName,
				Error:    err.Error(),
			})
		}
		return err
	}

	if jsonFlag {
		return cliout.WriteJSON(ctx.Stdout, scenarioPortSingleOutput{
			Success:  true,
			Scenario: scenarioName,
			PortName: resolved.Name,
			Step:     resolved.Step,
			Port:     resolved.Port,
		})
	}
	_, _ = fmt.Fprintln(ctx.Stdout, resolved.Port)
	return nil
}

func runScenarioOpenCommandWithApp(app *App, ctx *commandContext, args []string) error {
	scenarioName := ""
	portName := "UI_PORT"
	printURL := false
	jsonFlag := ctx.Globals.json

	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch arg {
		case "--help", "-h":
			_, _ = fmt.Fprintln(ctx.Stdout, "Usage: vrooli scenario open <scenario-name> [--port <name>] [--print-url]")
			return nil
		case "--port":
			if index+1 >= len(args) {
				return usageErrorf("scenario open", "scenario open --port requires a value")
			}
			index++
			portName = args[index]
		case "--print-url":
			printURL = true
		case "--json":
			jsonFlag = true
		default:
			if strings.HasPrefix(arg, "-") {
				return unknownOptionError("scenario open", arg)
			}
			if scenarioName != "" {
				return usageErrorf("scenario open", "scenario open accepts exactly one scenario name")
			}
			scenarioName = arg
		}
	}

	if scenarioName == "" {
		return usageErrorf("scenario open", "scenario open requires a scenario name")
	}

	serviceCtx := *ctx
	serviceCtx.Stdout = io.Discard
	serviceCtx.Stderr = io.Discard
	service, err := app.newScenarioService(&serviceCtx)
	if err != nil {
		return err
	}
	resolved, err := service.ResolvePort(scenarioName, portName)
	if err != nil {
		return err
	}

	if jsonFlag {
		return cliout.WriteJSON(ctx.Stdout, scenarioOpenOutput{
			Success:  true,
			Scenario: scenarioName,
			PortName: resolved.Name,
			Port:     resolved.Port,
			URL:      resolved.URL,
		})
	}
	if printURL {
		_, _ = fmt.Fprintln(ctx.Stdout, resolved.URL)
		return nil
	}
	if err := app.openScenarioURL(resolved.URL); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(ctx.Stderr, "Opening %s at %s\n", scenarioName, resolved.URL)
	return nil
}

func runScenarioUISmokeCommandWithApp(app *App, ctx *commandContext, args []string) error {
	home, err := ctx.HomeDir()
	if err != nil {
		return err
	}
	cliPath, err := app.locateTestGenieCLI(ctx.Root, home)
	if err != nil {
		return err
	}

	commandArgs := []string{"ui-smoke"}
	commandArgs = append(commandArgs, args...)
	commandArgs = append(commandArgs, app.passthroughFlags(ctx.Globals, commandArgs)...)
	return app.runScenarioSubprocess(scenarioSubprocessSpec{
		name:   cliPath,
		args:   commandArgs,
		dir:    ctx.Root,
		env:    app.commandEnv(ctx.Root, ctx.Globals),
		stdout: ctx.Stdout,
		stderr: ctx.Stderr,
	})
}

func runScenarioCompletenessCommandWithApp(app *App, ctx *commandContext, args []string) error {
	cliPath, err := app.locateScenarioCompletenessCLI(ctx.Root)
	if err != nil {
		return err
	}

	commandArgs := append([]string{}, args...)
	if ctx.Globals.json && !containsArg(commandArgs, "--json") && !containsArg(commandArgs, "--format") {
		commandArgs = append(commandArgs, "--json")
	}
	return app.runScenarioSubprocess(scenarioSubprocessSpec{
		name:   cliPath,
		args:   commandArgs,
		dir:    ctx.Root,
		env:    app.commandEnv(ctx.Root, ctx.Globals),
		stdout: ctx.Stdout,
		stderr: ctx.Stderr,
	})
}

func runScenarioRequirementsCommandWithApp(app *App, ctx *commandContext, args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		showScenarioRequirementsHelp(ctx.Stdout)
		return nil
	}

	if args[0] == "snapshot" {
		return runScenarioRequirementsSnapshot(ctx.Root, args[1:], ctx.Stdout)
	}

	home, err := ctx.HomeDir()
	if err != nil {
		return err
	}
	cliPath, err := app.locateTestGenieCLI(ctx.Root, home)
	if err != nil {
		return err
	}

	commandArgs, workdir, err := translateScenarioRequirementsArgs(ctx.Root, ctx.Globals, args)
	if err != nil {
		return err
	}
	return app.runScenarioSubprocess(scenarioSubprocessSpec{
		name:   cliPath,
		args:   commandArgs,
		dir:    workdir,
		env:    app.commandEnv(ctx.Root, ctx.Globals),
		stdout: ctx.Stdout,
		stderr: ctx.Stderr,
	})
}

func runScenarioHealFromSandboxCommandWithApp(app *App, ctx *commandContext, args []string) error {
	mergedPath := strings.TrimSpace(os.Getenv("SANDBOX_MERGED_DIR"))
	dryRun := false

	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--merged-path":
			if index+1 >= len(args) {
				return usageErrorf("scenario heal-from-sandbox", "scenario heal-from-sandbox --merged-path requires a value")
			}
			index++
			mergedPath = args[index]
		case "--dry-run":
			dryRun = true
		case "--help", "-h":
			_, _ = fmt.Fprintln(ctx.Stdout, "Usage: vrooli scenario heal-from-sandbox [--merged-path <path>] [--dry-run]")
			return nil
		default:
			return unknownOptionError("scenario heal-from-sandbox", args[index])
		}
	}

	if strings.TrimSpace(mergedPath) == "" {
		return usageErrorf("scenario heal-from-sandbox", "heal-from-sandbox requires SANDBOX_MERGED_DIR or --merged-path")
	}

	home, err := ctx.HomeDir()
	if err != nil {
		return err
	}
	processRoot := filepath.Join(home, ".vrooli", "processes", "scenarios")
	entries, err := os.ReadDir(processRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	affected := []string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		records, readErr := process.ReadScenarioRecords(home, name)
		if readErr != nil {
			return readErr
		}
		for _, record := range records {
			if strings.HasPrefix(record.WorkingDir, mergedPath) {
				affected = append(affected, name)
				break
			}
		}
	}
	sort.Strings(affected)

	if len(affected) == 0 {
		return nil
	}
	if dryRun {
		_, _ = fmt.Fprintf(ctx.Stdout, "heal-from-sandbox: dry-run mode, would stop and restart: %s\n", strings.Join(affected, ", "))
		return nil
	}

	runner, err := app.newScenarioLifecycleRunner(ctx)
	if err != nil {
		return err
	}
	for _, name := range affected {
		if stopErr := runner.Stop(name, lifecycle.StopOptions{}); stopErr != nil {
			_, _ = fmt.Fprintf(ctx.Stderr, "heal-from-sandbox: stop %s failed: %v\n", name, stopErr)
		}
	}
	time.Sleep(1 * time.Second)
	for _, name := range affected {
		if startErr := app.launchDetachedScenario(ctx.Root, ctx.Globals, "start", name); startErr != nil {
			return startErr
		}
	}
	_, _ = fmt.Fprintf(ctx.Stdout, "heal-from-sandbox: stopped and relaunched %d scenario(s)\n", len(affected))
	return nil
}

func showScenarioRequirementsHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli scenario requirements <subcommand> [options]")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Subcommands:")
	_, _ = fmt.Fprintln(w, "  report <name> [options]          Generate requirement coverage summary")
	_, _ = fmt.Fprintln(w, "  validate <name> [--quiet]        Validate requirement files")
	_, _ = fmt.Fprintln(w, "  sync <name>                      Sync requirement statuses from local evidence")
	_, _ = fmt.Fprintln(w, "  manual-log <name> <req> [opts]   Record manual validation evidence")
	_, _ = fmt.Fprintln(w, "  snapshot <name>                  Show latest requirements sync snapshot")
	_, _ = fmt.Fprintln(w, "  lint-prd <name> [--json]         Check PRD to requirements mapping")
	_, _ = fmt.Fprintln(w, "  phase <name> --phase <phase>     Inspect validations for a single phase")
	_, _ = fmt.Fprintln(w, "  init <name> [options]            Scaffold a requirements registry")
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
