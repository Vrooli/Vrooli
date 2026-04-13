package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
)

type (
	scenarioListPortOutput      = scenariocli.ListPortOutput
	scenarioListItemOutput      = scenariocli.ListItemOutput
	scenarioStatusItemOutput    = scenariocli.StatusItemOutput
	scenarioInfoOutput          = scenariocli.InfoOutput
	scenarioInfoScenarioData    = scenariocli.InfoScenarioData
	scenarioInfoRuntimeData     = scenariocli.InfoRuntimeData
	scenarioStatusSingleOutput  = scenariocli.StatusSingleOutput
	scenarioLifecycleItemOutput = scenariocli.LifecycleItemOutput
)

func runScenarioStartCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeCommandAction(app, ctx, args, commandAction[scenarioStartRequest, []scenarioLifecycleItemOutput]{
		parse:  parseScenarioStartRequestFromContext,
		run:    runScenarioStartRequest,
		render: renderScenarioLifecycleResponse,
	})
}

func runScenarioStopCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeCommandAction(app, ctx, args, commandAction[scenarioStopRequest, []scenarioLifecycleItemOutput]{
		parse:  parseScenarioStopRequestFromContext,
		run:    runScenarioStopRequest,
		render: renderScenarioLifecycleResponse,
	})
}

func runScenarioRestartCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeCommandAction(app, ctx, args, commandAction[scenarioRestartRequest, []scenarioLifecycleItemOutput]{
		parse:  parseScenarioRestartRequestFromContext,
		run:    runScenarioRestartRequest,
		render: renderScenarioLifecycleResponse,
	})
}

func runScenarioListCommand(root string, globals globalOptions, args []string, stdout io.Writer) error {
	app, ctx := newConfiguredCommandContext(root, globals, stdout, io.Discard)
	return runScenarioListCommandWithApp(app, ctx, args)
}

func runScenarioListCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeCommandAction(app, ctx, args, commandAction[scenarioListRequest, scenarioListResponse]{
		parse:  parseScenarioListRequestFromContext,
		run:    runScenarioListRequest,
		render: renderScenarioListResponse,
	})
}

func runScenarioInfoCommand(root string, globals globalOptions, args []string, stdout io.Writer) error {
	app, ctx := newConfiguredCommandContext(root, globals, stdout, io.Discard)
	return runScenarioInfoCommandWithApp(app, ctx, args)
}

func runScenarioInfoCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeCommandAction(app, ctx, args, commandAction[scenarioInfoRequest, scenarioInfoOutput]{
		parse:  parseScenarioInfoRequestFromContext,
		run:    runScenarioInfoRequest,
		render: renderScenarioInfoResponse,
	})
}

func runScenarioStatusCommand(root string, globals globalOptions, args []string, stdout io.Writer) error {
	app, ctx := newConfiguredCommandContext(root, globals, stdout, io.Discard)
	return runScenarioStatusCommandWithApp(app, ctx, args)
}

func runScenarioStatusCommandWithApp(app *App, ctx *commandContext, args []string) error {
	return executeCommandAction(app, ctx, args, commandAction[scenarioStatusRequest, scenarioStatusResponse]{
		parse:  parseScenarioStatusRequestFromContext,
		run:    runScenarioStatusRequest,
		render: renderScenarioStatusResponse,
	})
}

func loadScenarioState(root string) ([]scenario.Scenario, map[string]process.ScenarioRuntime, error) {
	app, ctx := newConfiguredCommandContext(root, globalOptions{}, io.Discard, io.Discard)
	service, err := app.newScenarioService(ctx)
	if err != nil {
		return nil, nil, err
	}
	inventory, err := service.Inventory()
	if err != nil {
		return nil, nil, err
	}

	items := make([]scenario.Scenario, 0, len(inventory))
	runtimes := make(map[string]process.ScenarioRuntime, len(inventory))
	for _, item := range inventory {
		items = append(items, item.Scenario)
		if item.Runtime.ProcessCount > 0 {
			runtimes[item.Scenario.Slug] = item.Runtime
		}
	}
	return items, runtimes, nil
}

func loadScenarioDetail(root, name string) (scenario.Scenario, process.ScenarioRuntime, string, error) {
	app, ctx := newConfiguredCommandContext(root, globalOptions{}, io.Discard, io.Discard)
	service, err := app.newScenarioService(ctx)
	if err != nil {
		return scenario.Scenario{}, process.ScenarioRuntime{}, "", err
	}
	detail, err := service.Detail(name)
	if err != nil {
		return scenario.Scenario{}, process.ScenarioRuntime{}, "", err
	}
	return detail.Scenario, detail.Runtime, detail.Details.Health, nil
}

func buildScenarioStatusItem(item scenario.Scenario, runtime process.ScenarioRuntime) scenarioStatusItemOutput {
	return scenariocli.BuildStatusItem(item, runtime)
}

func buildScenarioStatusDetail(detail orchestrator.Detail) scenarioStatusItemOutput {
	return scenariocli.BuildStatusDetail(detail)
}

func buildScenarioInfoData(item scenario.Scenario) scenarioInfoScenarioData {
	return scenariocli.BuildInfoData(item)
}

func buildScenarioRuntimeData(manifest scenario.ServiceManifest, runtime process.ScenarioRuntime) scenarioInfoRuntimeData {
	return scenariocli.BuildRuntimeData(manifest, runtime)
}

func runtimePortOutputs(bindings []scenario.RuntimePortBinding) []scenarioListPortOutput {
	return scenariocli.RuntimePortOutputs(bindings)
}

// buildListPorts preserves the historical CLI output contract while the
// underlying runtime/port logic lives in internal/scenario.
func buildListPorts(manifest scenario.ServiceManifest, records []process.Record) ([]scenarioListPortOutput, map[string]int) {
	return scenariocli.BuildListPorts(manifest, records)
}

func inferPortEnvVar(manifest scenario.ServiceManifest, step string) string {
	return scenario.InferPortEnvVar(manifest, step)
}

func copyIntMap(src map[string]int) map[string]int {
	return scenariocli.CopyIntMap(src)
}

func parseScenarioNameAndJSON(command string, defaultJSON bool, args []string) (string, bool, error) {
	name, jsonFlag, err := parseOptionalScenarioNameAndJSON(command, defaultJSON, args)
	if err != nil {
		return "", false, err
	}
	if name == "" {
		return "", false, usageErrorf("scenario "+command, "scenario %s requires a scenario name", command)
	}
	return name, jsonFlag, nil
}

func parseOptionalScenarioNameAndJSON(command string, defaultJSON bool, args []string) (string, bool, error) {
	name := ""
	jsonFlag := defaultJSON
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonFlag = true
		case "--help", "-h":
			return "", false, fmt.Errorf("usage requested")
		default:
			if strings.HasPrefix(arg, "-") {
				return "", false, unknownOptionError("scenario "+command, arg)
			}
			if name != "" {
				return "", false, usageErrorf("scenario "+command, "scenario %s accepts at most one scenario name", command)
			}
			name = arg
		}
	}
	return name, jsonFlag, nil
}

func parseScenarioStartArgs(defaultJSON bool, args []string) ([]string, lifecycle.StartOptions, bool, bool, error) {
	names := []string{}
	jsonFlag := defaultJSON
	openAfter := false
	opts := lifecycle.StartOptions{}

	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch arg {
		case "--json":
			jsonFlag = true
		case "--open":
			openAfter = true
		case "--best-effort":
			opts.BestEffort = true
		case "--clean-stale":
			opts.CleanStale = true
		case "--path":
			if index+1 >= len(args) {
				return nil, lifecycle.StartOptions{}, false, false, usageErrorf("scenario start", "scenario start --path requires a value")
			}
			index++
			opts.CustomPath = args[index]
		case "--help", "-h":
			return nil, lifecycle.StartOptions{}, false, false, fmt.Errorf("usage requested")
		default:
			if strings.HasPrefix(arg, "-") {
				return nil, lifecycle.StartOptions{}, false, false, unknownOptionError("scenario start", arg)
			}
			names = append(names, arg)
		}
	}
	return names, opts, jsonFlag, openAfter, nil
}

func parseScenarioSingleStartArgs(command string, defaultJSON bool, args []string) (string, lifecycle.StartOptions, bool, bool, error) {
	names, opts, jsonFlag, openAfter, err := parseScenarioStartArgs(defaultJSON, args)
	if err != nil {
		return "", lifecycle.StartOptions{}, false, false, err
	}
	if len(names) == 0 {
		return "", lifecycle.StartOptions{}, false, false, usageErrorf("scenario "+command, "scenario %s requires a scenario name", command)
	}
	if len(names) > 1 {
		return "", lifecycle.StartOptions{}, false, false, usageErrorf("scenario "+command, "scenario %s accepts exactly one scenario name", command)
	}
	return names[0], opts, jsonFlag, openAfter, nil
}

func showScenarioHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Vrooli Scenario Commands")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  vrooli scenario <subcommand> [options]")
	_, _ = fmt.Fprintln(w)
	renderCommandGroups(w, groupedScenarioCommands())
}

func writeScenarioInfoHuman(w io.Writer, info scenarioInfoScenarioData, runtime scenarioInfoRuntimeData) {
	scenariocli.WriteInfoHuman(w, info, runtime)
}

func writeScenarioStatusTable(w io.Writer, items []scenarioStatusItemOutput) {
	scenariocli.WriteStatusTable(w, items)
}

func writeScenarioStatusHuman(w io.Writer, output scenarioStatusSingleOutput) {
	scenariocli.WriteStatusHuman(w, output)
}

func formatPortMap(ports map[string]int) string {
	return scenariocli.FormatPortMap(ports)
}

func copyStrings(values []string) []string {
	return scenariocli.CopyStrings(values)
}

func copyProcessRecords(values []process.Record) []process.Record {
	return scenariocli.CopyProcessRecords(values)
}
