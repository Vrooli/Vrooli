package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
)

type scenarioCommandAction[Req any, Resp any] struct {
	parse  func(globals globalOptions, args []string) (Req, error)
	run    func(app *App, ctx *commandContext, req Req) (cliout.Format, Resp, error)
	render func(w io.Writer, format cliout.Format, resp Resp) error
}

func executeScenarioCommand[Req any, Resp any](app *App, ctx *commandContext, args []string, action scenarioCommandAction[Req, Resp]) error {
	req, err := action.parse(ctx.Globals, args)
	if err != nil {
		var helpErr commandHelpError
		if errors.As(err, &helpErr) {
			_, _ = fmt.Fprintln(ctx.Stdout, helpErr.message)
			return nil
		}
		return err
	}
	format, resp, err := action.run(app, ctx, req)
	if err != nil {
		return err
	}
	return action.render(ctx.Stdout, format, resp)
}

type scenarioStartRequest struct {
	Names     []string
	Options   lifecycle.StartOptions
	JSON      bool
	OpenAfter bool
}

type scenarioStopRequest struct {
	Name string
	JSON bool
}

type scenarioRestartRequest struct {
	Name      string
	Options   lifecycle.StartOptions
	JSON      bool
	OpenAfter bool
}

type scenarioListRequest struct {
	JSON         bool
	IncludePorts bool
}

type scenarioInfoRequest struct {
	Name string
	JSON bool
}

type scenarioStatusRequest struct {
	Name string
	JSON bool
}

type scenarioSetupRequest struct {
	Name string
	Opts lifecycle.PhaseOptions
	JSON bool
}

type scenarioTestRequest struct {
	Name string
	Opts lifecycle.PhaseOptions
}

type scenarioStartAllRequest struct {
	JSON bool
}

type scenarioStopAllRequest struct {
	JSON bool
}

type scenarioPortRequest struct {
	ScenarioName string
	PortName     string
	JSON         bool
}

type scenarioOpenRequest struct {
	ScenarioName string
	PortName     string
	PrintURL     bool
	JSON         bool
}

func parseScenarioStartRequest(globals globalOptions, args []string) (scenarioStartRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return scenarioStartRequest{}, commandHelpOnly("Usage: vrooli scenario start <name> [name2...] [--path <path>] [--best-effort] [--clean-stale] [--open] [--json]")
		}
	}

	names, opts, jsonFlag, openAfter, err := parseScenarioStartArgs(globals.json, args)
	if err != nil {
		return scenarioStartRequest{}, err
	}
	if len(names) == 0 {
		return scenarioStartRequest{}, usageErrorf("scenario start", "scenario start requires at least one scenario name")
	}
	if opts.CustomPath != "" && len(names) != 1 {
		return scenarioStartRequest{}, usageErrorf("scenario start", "scenario start with --path accepts exactly one scenario name")
	}
	return scenarioStartRequest{
		Names:     names,
		Options:   opts,
		JSON:      jsonFlag,
		OpenAfter: openAfter,
	}, nil
}

func runScenarioStartRequest(app *App, ctx *commandContext, req scenarioStartRequest) (cliout.Format, []scenarioLifecycleItemOutput, error) {
	service, format, err := app.newScenarioServiceForFormat(ctx, req.JSON)
	if err != nil {
		return "", nil, err
	}

	items := make([]scenarioLifecycleItemOutput, 0, len(req.Names))
	for _, name := range req.Names {
		result, err := service.StartDetailed(name, req.Options)
		if err != nil {
			return "", nil, err
		}

		status := "started"
		if result.AlreadyRunning {
			status = "already_running"
		}
		items = append(items, scenarioLifecycleItemOutput{
			Name:               result.Scenario.Slug,
			Status:             status,
			Health:             result.Details.Health,
			Ports:              envPortMap(result.Scenario.Manifest, result.AllocatedPorts),
			FailedDependencies: append([]string(nil), result.FailedDependencies...),
		})

		if req.OpenAfter {
			resolved, err := service.ResolvePort(name, "UI_PORT")
			if err != nil {
				return "", nil, err
			}
			if err := app.openScenarioURL(resolved.URL); err != nil {
				return "", nil, err
			}
		}
	}

	return format, items, nil
}

func parseScenarioStopRequest(globals globalOptions, args []string) (scenarioStopRequest, error) {
	name, jsonFlag, err := parseScenarioNameAndJSON("stop", globals.json, args)
	if err != nil {
		return scenarioStopRequest{}, err
	}
	return scenarioStopRequest{Name: name, JSON: jsonFlag}, nil
}

func runScenarioStopRequest(app *App, ctx *commandContext, req scenarioStopRequest) (cliout.Format, []scenarioLifecycleItemOutput, error) {
	runner, format, err := app.newScenarioLifecycleRunnerForFormat(ctx, req.JSON)
	if err != nil {
		return "", nil, err
	}
	if err := runner.Stop(req.Name, lifecycle.StopOptions{}); err != nil {
		return "", nil, err
	}
	return format, []scenarioLifecycleItemOutput{{Name: req.Name, Status: "stopped"}}, nil
}

func parseScenarioRestartRequest(globals globalOptions, args []string) (scenarioRestartRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return scenarioRestartRequest{}, commandHelpOnly("Usage: vrooli scenario restart <name> [--path <path>] [--best-effort] [--clean-stale] [--open] [--json]")
		}
	}

	name, opts, jsonFlag, openAfter, err := parseScenarioSingleStartArgs("restart", globals.json, args)
	if err != nil {
		return scenarioRestartRequest{}, err
	}
	return scenarioRestartRequest{
		Name:      name,
		Options:   opts,
		JSON:      jsonFlag,
		OpenAfter: openAfter,
	}, nil
}

func runScenarioRestartRequest(app *App, ctx *commandContext, req scenarioRestartRequest) (cliout.Format, []scenarioLifecycleItemOutput, error) {
	service, format, err := app.newScenarioServiceForFormat(ctx, req.JSON)
	if err != nil {
		return "", nil, err
	}
	result, err := service.RestartDetailed(req.Name, req.Options)
	if err != nil {
		return "", nil, err
	}

	item := scenarioLifecycleItemOutput{
		Name:               result.Scenario.Slug,
		Status:             "restarted",
		Health:             result.Details.Health,
		Ports:              envPortMap(result.Scenario.Manifest, result.AllocatedPorts),
		FailedDependencies: append([]string(nil), result.FailedDependencies...),
	}

	if req.OpenAfter {
		resolved, err := service.ResolvePort(req.Name, "UI_PORT")
		if err != nil {
			return "", nil, err
		}
		if err := app.openScenarioURL(resolved.URL); err != nil {
			return "", nil, err
		}
	}

	return format, []scenarioLifecycleItemOutput{item}, nil
}

func renderScenarioLifecycleResponse(w io.Writer, format cliout.Format, items []scenarioLifecycleItemOutput) error {
	return writeScenarioLifecycleItems(w, format, items)
}

func parseScenarioListRequest(globals globalOptions, args []string) (scenarioListRequest, error) {
	req := scenarioListRequest{JSON: globals.json}
	for _, arg := range args {
		switch arg {
		case "--json":
			req.JSON = true
		case "--include-ports":
			req.IncludePorts = true
		case "--help", "-h":
			return scenarioListRequest{}, commandHelpOnly("Usage: vrooli scenario list [--json] [--include-ports]")
		default:
			return scenarioListRequest{}, unknownOptionError("scenario list", arg)
		}
	}
	return req, nil
}

type scenarioListResponse struct {
	Items        []scenarioListItemOutput
	RunningCount int
}

func runScenarioListRequest(app *App, ctx *commandContext, req scenarioListRequest) (cliout.Format, scenarioListResponse, error) {
	format, err := ctx.outputFormat(req.JSON)
	if err != nil {
		return "", scenarioListResponse{}, err
	}

	service, err := app.newQuietScenarioService(ctx)
	if err != nil {
		return "", scenarioListResponse{}, err
	}
	inventory, err := service.Inventory()
	if err != nil {
		return "", scenarioListResponse{}, err
	}

	resp := scenarioListResponse{Items: make([]scenarioListItemOutput, 0, len(inventory))}
	for _, item := range inventory {
		status := "available"
		if item.Details.Status == "running" {
			status = item.Details.Status
			resp.RunningCount++
		}

		listPorts := []scenarioListPortOutput{}
		if req.IncludePorts && item.Details.Status == "running" {
			listPorts = runtimePortOutputs(item.Details.PortBindings)
		}

		resp.Items = append(resp.Items, scenarioListItemOutput{
			Name:        item.Scenario.Slug,
			Description: item.Scenario.Manifest.Service.Description,
			Version:     item.Scenario.Manifest.Service.Version,
			Status:      status,
			Tags:        copyStrings(item.Scenario.Manifest.Service.Tags),
			Path:        item.Scenario.Path + string(os.PathSeparator),
			Ports:       listPorts,
		})
	}
	return format, resp, nil
}

func renderScenarioListResponse(w io.Writer, format cliout.Format, resp scenarioListResponse) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success": true,
			"summary": map[string]int{
				"total_scenarios": len(resp.Items),
				"running":         resp.RunningCount,
				"available":       len(resp.Items) - resp.RunningCount,
			},
			"scenarios": resp.Items,
		})
	}

	_, _ = fmt.Fprintln(w, "[INFO]    Available scenarios:")
	for _, item := range resp.Items {
		line := "  • " + item.Name
		if item.Description != "" {
			line += " - " + item.Description
		}
		if len(item.Ports) > 0 {
			portParts := make([]string, 0, len(item.Ports))
			for _, port := range item.Ports {
				portParts = append(portParts, fmt.Sprintf("%s=%d", port.Key, port.Port))
			}
			line += " (ports: " + strings.Join(portParts, ", ") + ")"
		}
		_, _ = fmt.Fprintln(w, line)
	}
	return nil
}

func parseScenarioInfoRequest(globals globalOptions, args []string) (scenarioInfoRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return scenarioInfoRequest{}, commandHelpOnly("Usage: vrooli scenario info <name> [--json]")
		}
	}
	name, jsonFlag, err := parseScenarioNameAndJSON("info", globals.json, args)
	if err != nil {
		return scenarioInfoRequest{}, err
	}
	return scenarioInfoRequest{Name: name, JSON: jsonFlag}, nil
}

func runScenarioInfoRequest(app *App, ctx *commandContext, req scenarioInfoRequest) (cliout.Format, scenarioInfoOutput, error) {
	format, err := ctx.outputFormat(req.JSON)
	if err != nil {
		return "", scenarioInfoOutput{}, err
	}
	service, err := app.newQuietScenarioService(ctx)
	if err != nil {
		return "", scenarioInfoOutput{}, err
	}
	detail, err := service.Detail(req.Name)
	if err != nil {
		return "", scenarioInfoOutput{}, err
	}

	return format, scenarioInfoOutput{
		Success:  true,
		Scenario: buildScenarioInfoData(detail.Scenario),
		Runtime:  buildScenarioRuntimeData(detail.Scenario.Manifest, detail.Runtime),
	}, nil
}

func renderScenarioInfoResponse(w io.Writer, format cliout.Format, resp scenarioInfoOutput) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, resp)
	}
	writeScenarioInfoHuman(w, resp.Scenario, resp.Runtime)
	return nil
}

func parseScenarioStatusRequest(globals globalOptions, args []string) (scenarioStatusRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return scenarioStatusRequest{}, commandHelpOnly("Usage: vrooli scenario status [name] [--json]")
		}
	}
	name, jsonFlag, err := parseOptionalScenarioNameAndJSON("status", globals.json, args)
	if err != nil {
		return scenarioStatusRequest{}, err
	}
	return scenarioStatusRequest{Name: name, JSON: jsonFlag}, nil
}

type scenarioStatusResponse struct {
	Single *scenarioStatusSingleOutput
	List   []scenarioStatusItemOutput
}

func runScenarioStatusRequest(app *App, ctx *commandContext, req scenarioStatusRequest) (cliout.Format, scenarioStatusResponse, error) {
	format, err := ctx.outputFormat(req.JSON)
	if err != nil {
		return "", scenarioStatusResponse{}, err
	}
	service, err := app.newQuietScenarioService(ctx)
	if err != nil {
		return "", scenarioStatusResponse{}, err
	}

	if req.Name == "" {
		inventory, err := service.Inventory()
		if err != nil {
			return "", scenarioStatusResponse{}, err
		}
		items := make([]scenarioStatusItemOutput, 0, len(inventory))
		for _, item := range inventory {
			items = append(items, buildScenarioStatusDetail(item))
		}
		return format, scenarioStatusResponse{List: items}, nil
	}

	detail, err := service.Detail(req.Name)
	if err != nil {
		return "", scenarioStatusResponse{}, err
	}
	output := scenarioStatusSingleOutput{
		Success:  true,
		Scenario: buildScenarioStatusDetail(detail),
		Info:     buildScenarioInfoData(detail.Scenario),
		Runtime:  buildScenarioRuntimeData(detail.Scenario.Manifest, detail.Runtime),
	}
	return format, scenarioStatusResponse{Single: &output}, nil
}

func renderScenarioStatusResponse(w io.Writer, format cliout.Format, resp scenarioStatusResponse) error {
	if resp.Single == nil {
		if format == cliout.FormatJSON {
			runningCount := 0
			for _, item := range resp.List {
				if item.Status == "running" {
					runningCount++
				}
			}
			return cliout.WriteJSON(w, map[string]any{
				"success": true,
				"summary": map[string]int{
					"total_scenarios": len(resp.List),
					"running":         runningCount,
					"stopped":         len(resp.List) - runningCount,
				},
				"scenarios": resp.List,
			})
		}
		writeScenarioStatusTable(w, resp.List)
		return nil
	}

	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, resp.Single)
	}
	writeScenarioStatusHuman(w, *resp.Single)
	return nil
}

func (app *App) newQuietScenarioService(ctx *commandContext) (*orchestrator.Service, error) {
	quietCtx := *ctx
	quietCtx.Stdout = io.Discard
	quietCtx.Stderr = io.Discard
	return app.newScenarioService(&quietCtx)
}

func parseScenarioSetupRequest(globals globalOptions, args []string) (scenarioSetupRequest, error) {
	jsonFlag := globals.json
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			return scenarioSetupRequest{}, commandHelpOnly("Usage: vrooli scenario setup <name> [--path <path>]")
		case "--json":
			jsonFlag = true
		}
	}
	name, opts, err := parseScenarioPhaseArgs("setup", args)
	if err != nil {
		return scenarioSetupRequest{}, err
	}
	return scenarioSetupRequest{Name: name, Opts: opts, JSON: jsonFlag}, nil
}

func runScenarioSetupRequest(app *App, ctx *commandContext, req scenarioSetupRequest) (cliout.Format, lifecycle.PhaseResult, error) {
	runner, format, err := app.newScenarioLifecycleRunnerForFormat(ctx, req.JSON)
	if err != nil {
		return "", lifecycle.PhaseResult{}, err
	}
	result, err := runner.RunPhaseDetailed(req.Name, "setup", req.Opts)
	if err != nil {
		return "", lifecycle.PhaseResult{}, err
	}
	return format, result, nil
}

func renderScenarioSetupResponse(w io.Writer, format cliout.Format, result lifecycle.PhaseResult) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success": true,
			"phase":   "setup",
			"status":  result.Status,
			"defined": result.Defined,
			"steps": map[string]int{
				"executed": result.ExecutedSteps,
				"skipped":  result.SkippedSteps,
			},
		})
	}

	switch result.Status {
	case lifecycle.PhaseExecutionCompleted:
		_, _ = fmt.Fprintf(w, "Completed setup for scenario (%d executed, %d skipped)\n", result.ExecutedSteps, result.SkippedSteps)
	case lifecycle.PhaseExecutionSkipped:
		_, _ = fmt.Fprintf(w, "Setup phase ran no steps (%d skipped)\n", result.SkippedSteps)
	default:
		_, _ = fmt.Fprintln(w, "Scenario does not define a setup phase")
	}
	return nil
}

func parseScenarioTestRequest(globals globalOptions, args []string) (scenarioTestRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return scenarioTestRequest{}, commandHelpOnly("Usage: vrooli scenario test <name> [phase|all|e2e] [--allow-skip-missing-runtime] [--manage-runtime]")
		}
	}
	name, opts, err := parseScenarioTestArgs(globals, args)
	if err != nil {
		return scenarioTestRequest{}, err
	}
	return scenarioTestRequest{Name: name, Opts: opts}, nil
}

func runScenarioTestRequest(app *App, ctx *commandContext, req scenarioTestRequest) (cliout.Format, struct{}, error) {
	runner, _, err := app.newScenarioLifecycleRunnerForFormat(ctx, false)
	if err != nil {
		return "", struct{}{}, err
	}
	return cliout.FormatHuman, struct{}{}, runner.RunPhase(req.Name, "test", req.Opts)
}

func renderScenarioTestResponse(w io.Writer, format cliout.Format, _ struct{}) error {
	return nil
}

func parseScenarioStartAllRequest(globals globalOptions, args []string) (scenarioStartAllRequest, error) {
	req := scenarioStartAllRequest{JSON: globals.json}
	for _, arg := range args {
		switch arg {
		case "--json":
			req.JSON = true
		case "--help", "-h":
			return scenarioStartAllRequest{}, commandHelpOnly("Usage: vrooli scenario start-all [--json]")
		default:
			return scenarioStartAllRequest{}, unknownOptionError("scenario start-all", arg)
		}
	}
	return req, nil
}

type scenarioBatchResponse struct {
	Verb    string
	Started []scenarioLifecycleItemOutput
	Stopped []string
	Failed  []scenarioBatchFailure
}

func runScenarioStartAllRequest(app *App, ctx *commandContext, req scenarioStartAllRequest) (cliout.Format, scenarioBatchResponse, error) {
	service, format, err := app.newScenarioServiceForFormat(ctx, req.JSON)
	if err != nil {
		return "", scenarioBatchResponse{}, err
	}
	report, err := service.StartAll()
	if err != nil {
		return "", scenarioBatchResponse{}, err
	}
	started := make([]scenarioLifecycleItemOutput, 0, len(report.Started))
	for _, item := range report.Started {
		started = append(started, scenarioLifecycleItemOutput{Name: item.Name, Status: "started"})
	}
	failed := make([]scenarioBatchFailure, 0, len(report.Failed))
	for _, item := range report.Failed {
		failed = append(failed, scenarioBatchFailure{Name: item.Name, Error: item.Error})
	}
	return format, scenarioBatchResponse{Verb: "Started", Started: started, Failed: failed}, nil
}

func parseScenarioStopAllRequest(globals globalOptions, args []string) (scenarioStopAllRequest, error) {
	req := scenarioStopAllRequest{JSON: globals.json}
	for _, arg := range args {
		switch arg {
		case "--json":
			req.JSON = true
		case "--help", "-h":
			return scenarioStopAllRequest{}, commandHelpOnly("Usage: vrooli scenario stop-all [--json]")
		default:
			return scenarioStopAllRequest{}, unknownOptionError("scenario stop-all", arg)
		}
	}
	return req, nil
}

func runScenarioStopAllRequest(app *App, ctx *commandContext, req scenarioStopAllRequest) (cliout.Format, scenarioBatchResponse, error) {
	service, format, err := app.newScenarioServiceForFormat(ctx, req.JSON)
	if err != nil {
		return "", scenarioBatchResponse{}, err
	}
	report, err := service.StopAll()
	if err != nil {
		return "", scenarioBatchResponse{}, err
	}
	stopped := make([]string, 0, len(report.Stopped))
	for _, item := range report.Stopped {
		stopped = append(stopped, item.Name)
	}
	failed := make([]scenarioBatchFailure, 0, len(report.Failed))
	for _, item := range report.Failed {
		failed = append(failed, scenarioBatchFailure{Name: item.Name, Error: item.Error})
	}
	return format, scenarioBatchResponse{Verb: "Stopped", Stopped: stopped, Failed: failed}, nil
}

func renderScenarioBatchResponse(w io.Writer, format cliout.Format, resp scenarioBatchResponse) error {
	return writeScenarioBatchReport(w, format, resp.Verb, resp.Started, resp.Stopped, resp.Failed)
}

func parseScenarioPortRequest(globals globalOptions, args []string) (scenarioPortRequest, error) {
	req := scenarioPortRequest{JSON: globals.json}
	for _, arg := range args {
		switch {
		case arg == "--json":
			req.JSON = true
		case arg == "--help" || arg == "-h":
			return scenarioPortRequest{}, commandHelpOnly("Usage: vrooli scenario port <scenario-name> [<port-name>] [--json]")
		case strings.HasPrefix(arg, "-"):
			return scenarioPortRequest{}, unknownOptionError("scenario port", arg)
		case req.ScenarioName == "":
			req.ScenarioName = arg
		case req.PortName == "":
			req.PortName = arg
		default:
			return scenarioPortRequest{}, usageErrorf("scenario port", "scenario port accepts at most two positional arguments")
		}
	}
	if req.ScenarioName == "" {
		return scenarioPortRequest{}, usageErrorf("scenario port", "scenario port requires a scenario name")
	}
	return req, nil
}

type scenarioPortResponse struct {
	Single *scenarioPortSingleOutput
	List   *scenarioPortListOutput
}

func runScenarioPortRequest(app *App, ctx *commandContext, req scenarioPortRequest) (cliout.Format, scenarioPortResponse, error) {
	service, err := app.newQuietScenarioService(ctx)
	if err != nil {
		return "", scenarioPortResponse{}, err
	}
	detail, err := service.Detail(req.ScenarioName)
	if err != nil {
		return "", scenarioPortResponse{}, err
	}
	listPorts, portsMap := buildListPorts(detail.Scenario.Manifest, detail.Runtime.Records)

	if req.PortName == "" {
		if detail.Runtime.ProcessCount == 0 || len(portsMap) == 0 {
			if req.JSON {
				return cliout.FormatJSON, scenarioPortResponse{List: &scenarioPortListOutput{
					Success:  false,
					Scenario: req.ScenarioName,
					Ports:    []scenarioListPortOutput{},
					Error:    "No running processes found for scenario",
				}}, nil
			}
			return "", scenarioPortResponse{}, fmt.Errorf("no running processes found for scenario %q", req.ScenarioName)
		}
		if req.JSON {
			return cliout.FormatJSON, scenarioPortResponse{List: &scenarioPortListOutput{
				Success:  true,
				Scenario: req.ScenarioName,
				Ports:    listPorts,
				Metadata: map[string]int{"count": len(listPorts)},
			}}, nil
		}
		return cliout.FormatHuman, scenarioPortResponse{List: &scenarioPortListOutput{
			Success:  true,
			Scenario: req.ScenarioName,
			Ports:    listPorts,
		}}, nil
	}

	key, port, step, ok := resolveRequestedPort(detail.Scenario.Manifest, listPorts, portsMap, req.PortName)
	if !ok {
		if req.JSON {
			return cliout.FormatJSON, scenarioPortResponse{Single: &scenarioPortSingleOutput{
				Success:  false,
				Scenario: req.ScenarioName,
				PortName: req.PortName,
				Error:    fmt.Sprintf("No running port named %s for scenario", req.PortName),
			}}, nil
		}
		return "", scenarioPortResponse{}, fmt.Errorf("no running port named %s for scenario %q", req.PortName, req.ScenarioName)
	}
	format, err := formatFromJSON(req.JSON)
	if err != nil {
		return "", scenarioPortResponse{}, err
	}
	return format, scenarioPortResponse{Single: &scenarioPortSingleOutput{
		Success:  true,
		Scenario: req.ScenarioName,
		PortName: key,
		Step:     step,
		Port:     port,
	}}, nil
}

func renderScenarioPortResponse(w io.Writer, format cliout.Format, resp scenarioPortResponse) error {
	if resp.List != nil {
		if format == cliout.FormatJSON {
			return cliout.WriteJSON(w, resp.List)
		}
		if !resp.List.Success {
			return fmt.Errorf(resp.List.Error)
		}
		for _, port := range resp.List.Ports {
			_, _ = fmt.Fprintf(w, "%s=%d\n", port.Key, port.Port)
		}
		return nil
	}
	if resp.Single == nil {
		return nil
	}
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, resp.Single)
	}
	if !resp.Single.Success {
		return fmt.Errorf(resp.Single.Error)
	}
	_, _ = fmt.Fprintf(w, "%d\n", resp.Single.Port)
	return nil
}

func parseScenarioOpenRequest(globals globalOptions, args []string) (scenarioOpenRequest, error) {
	req := scenarioOpenRequest{PortName: "UI_PORT", JSON: globals.json}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch arg {
		case "--help", "-h":
			return scenarioOpenRequest{}, commandHelpOnly("Usage: vrooli scenario open <scenario-name> [--port <name>] [--print-url]")
		case "--port":
			if index+1 >= len(args) {
				return scenarioOpenRequest{}, usageErrorf("scenario open", "scenario open --port requires a value")
			}
			index++
			req.PortName = args[index]
		case "--print-url":
			req.PrintURL = true
		case "--json":
			req.JSON = true
		default:
			if strings.HasPrefix(arg, "-") {
				return scenarioOpenRequest{}, unknownOptionError("scenario open", arg)
			}
			if req.ScenarioName != "" {
				return scenarioOpenRequest{}, usageErrorf("scenario open", "scenario open accepts exactly one scenario name")
			}
			req.ScenarioName = arg
		}
	}
	if req.ScenarioName == "" {
		return scenarioOpenRequest{}, usageErrorf("scenario open", "scenario open requires a scenario name")
	}
	return req, nil
}

func runScenarioOpenRequest(app *App, ctx *commandContext, req scenarioOpenRequest) (cliout.Format, scenarioOpenOutput, error) {
	service, err := app.newQuietScenarioService(ctx)
	if err != nil {
		return "", scenarioOpenOutput{}, err
	}
	resolved, err := service.ResolvePort(req.ScenarioName, req.PortName)
	if err != nil {
		return "", scenarioOpenOutput{}, err
	}
	if !req.PrintURL {
		if req.JSON {
			return cliout.FormatJSON, scenarioOpenOutput{
				Success:  true,
				Scenario: req.ScenarioName,
				PortName: resolved.Name,
				Port:     resolved.Port,
				URL:      resolved.URL,
			}, nil
		}
		if err := app.openScenarioURL(resolved.URL); err != nil {
			return "", scenarioOpenOutput{}, err
		}
		_, _ = fmt.Fprintf(ctx.Stderr, "Opening %s at %s\n", req.ScenarioName, resolved.URL)
		return cliout.FormatHuman, scenarioOpenOutput{}, nil
	}
	return cliout.FormatHuman, scenarioOpenOutput{
		Success:  true,
		Scenario: req.ScenarioName,
		PortName: resolved.Name,
		Port:     resolved.Port,
		URL:      resolved.URL,
	}, nil
}

func renderScenarioOpenResponse(w io.Writer, _ cliout.Format, resp scenarioOpenOutput) error {
	if resp.URL == "" {
		return nil
	}
	_, _ = fmt.Fprintln(w, resp.URL)
	return nil
}

type commandHelpError struct {
	message string
}

func (e commandHelpError) Error() string { return e.message }

func commandHelpOnly(message string) error {
	return commandHelpError{message: message}
}
