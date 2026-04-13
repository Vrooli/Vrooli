package main

import (
	"io"

	"github.com/vrooli/vrooli/internal/cli/resourcecli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/resources"
)

type resourceCommandAction[Req any, Resp any] struct {
	parse  func(globals globalOptions, args []string) (Req, error)
	run    func(controller *resources.Controller, ctx *commandContext, req Req) (cliout.Format, Resp, error)
	render func(w io.Writer, format cliout.Format, resp Resp) error
}

func executeResourceCommandWithApp[Req any, Resp any](app *App, ctx *commandContext, controller *resources.Controller, args []string, action boundCommandAction[*resources.Controller, Req, Resp]) error {
	return executeBoundCommand(app, ctx, controller, args, action)
}

func executeResourceCommand[Req any, Resp any](controller *resources.Controller, globals globalOptions, args []string, stdout, stderr io.Writer, action resourceCommandAction[Req, Resp]) error {
	return runResourceBoundCommand(controller, globals, args, stdout, stderr, boundCommandAction[*resources.Controller, Req, Resp]{
		parse:  action.parse,
		run:    action.run,
		render: action.render,
	})
}

func runResourceBoundCommand[Req any, Resp any](controller *resources.Controller, globals globalOptions, args []string, stdout, stderr io.Writer, action boundCommandAction[*resources.Controller, Req, Resp]) error {
	app, ctx := newConfiguredCommandContext("", globals, stdout, stderr)
	if controller != nil {
		ctx.Root = controller.Root
	}
	return executeResourceCommandWithApp(app, ctx, controller, args, action)
}

type (
	resourceNoArgsRequest struct{}
	resourceNameRequest   struct {
		Name string
	}
)

type resourceStatusRequest struct {
	Name string
	Fast bool
}
type resourceArchiveSubcommandRequest struct {
	Subcommand string
}
type resourceBlueprintSearchRequest struct {
	Query string
}

type resourceListResponse struct {
	Items []resources.Resource
}

type resourceStatusesResponse struct {
	Items []resources.Status
}

type resourceStatusResponse struct {
	Item  *resources.Status
	Items []resources.Status
}

type resourceReportResponse struct {
	Key    string
	Report any
}

type resourceBlueprintsResponse struct {
	Items []resources.Blueprint
}

type resourceBlueprintResponse struct {
	Item resources.Blueprint
}

type resourceBlueprintSearchResponse struct {
	Query string
	Items []resources.Blueprint
}

func parseResourceNoArgs(help, command string, args []string) (resourceNoArgsRequest, error) {
	if len(args) > 0 {
		for _, arg := range args {
			if arg == "--help" || arg == "-h" {
				return resourceNoArgsRequest{}, commandHelpOnly(help)
			}
		}
		return resourceNoArgsRequest{}, usageErrorf(command, "%s does not accept positional arguments", command)
	}
	return resourceNoArgsRequest{}, nil
}

func parseResourceSingleName(help, command string, args []string) (resourceNameRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return resourceNameRequest{}, commandHelpOnly(help)
		}
	}
	if len(args) != 1 {
		return resourceNameRequest{}, usageErrorf(command, "%s requires exactly one resource name", command)
	}
	return resourceNameRequest{Name: args[0]}, nil
}

func parseResourceListRequest(globals globalOptions, args []string) (resourceNoArgsRequest, error) {
	return parseResourceNoArgs("Usage: vrooli resource list", "resource list", args)
}

func runResourceListRequest(controller *resources.Controller, ctx *commandContext, req resourceNoArgsRequest) (cliout.Format, resourceListResponse, error) {
	items, err := controller.Discover()
	if err != nil {
		return "", resourceListResponse{}, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", resourceListResponse{}, err
	}
	return format, resourceListResponse{Items: items}, nil
}

func renderResourceListResponse(w io.Writer, format cliout.Format, resp resourceListResponse) error {
	return resourcecli.WriteList(w, format, resp.Items)
}

func parseResourceStatusRequest(globals globalOptions, args []string) (resourceStatusRequest, error) {
	req := resourceStatusRequest{Fast: true}
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			return resourceStatusRequest{}, commandHelpOnly("Usage: vrooli resource status [name] [--fast|--no-fast] [--json]")
		case "--fast":
			req.Fast = true
		case "--no-fast":
			req.Fast = false
		default:
			filtered = append(filtered, arg)
		}
	}
	if len(filtered) > 1 {
		return resourceStatusRequest{}, usageErrorf("resource status", "resource status accepts at most one resource name")
	}
	if len(filtered) == 1 {
		req.Name = filtered[0]
	}
	return req, nil
}

func runResourceStatusRequest(controller *resources.Controller, ctx *commandContext, req resourceStatusRequest) (cliout.Format, resourceStatusResponse, error) {
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", resourceStatusResponse{}, err
	}
	if req.Name == "" {
		items, err := controller.ListStatuses(req.Fast, false)
		if err != nil {
			return "", resourceStatusResponse{}, err
		}
		return format, resourceStatusResponse{Items: items}, nil
	}
	item, err := controller.Status(req.Name, req.Fast)
	if err != nil {
		return "", resourceStatusResponse{}, err
	}
	return format, resourceStatusResponse{Item: &item}, nil
}

func renderResourceStatusResponse(w io.Writer, format cliout.Format, resp resourceStatusResponse) error {
	if resp.Item != nil {
		return resourcecli.WriteStatus(w, format, *resp.Item)
	}
	return resourcecli.WriteStatuses(w, format, resp.Items)
}

func parseResourceInfoRequest(globals globalOptions, args []string) (resourceNameRequest, error) {
	return parseResourceSingleName("Usage: vrooli resource info <name>", "resource info", args)
}

func runResourceInfoRequest(controller *resources.Controller, ctx *commandContext, req resourceNameRequest) (cliout.Format, resourceStatusResponse, error) {
	item, err := controller.Status(req.Name, true)
	if err != nil {
		return "", resourceStatusResponse{}, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", resourceStatusResponse{}, err
	}
	return format, resourceStatusResponse{Item: &item}, nil
}

func renderResourceInfoResponse(w io.Writer, format cliout.Format, resp resourceStatusResponse) error {
	return resourcecli.WriteInfo(w, format, *resp.Item)
}

func parseResourceDeprecateRequest(globals globalOptions, args []string) (resourceNameRequest, error) {
	return parseResourceSingleName("Usage: vrooli resource deprecate <name>", "resource deprecate", args)
}

func runResourceDeprecateRequest(controller *resources.Controller, ctx *commandContext, req resourceNameRequest) (cliout.Format, resources.DeprecationReport, error) {
	report, err := controller.DeprecateResource(req.Name)
	if err != nil {
		return "", resources.DeprecationReport{}, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", resources.DeprecationReport{}, err
	}
	return format, report, nil
}

func renderResourceDeprecateResponse(w io.Writer, format cliout.Format, report resources.DeprecationReport) error {
	return resourcecli.WriteDeprecationReport(w, format, report)
}

func parseResourceListDeprecatedRequest(globals globalOptions, args []string) (resourceNoArgsRequest, error) {
	return parseResourceNoArgs("Usage: vrooli resource list-deprecated", "resource list-deprecated", args)
}

func runResourceListDeprecatedRequest(controller *resources.Controller, ctx *commandContext, req resourceNoArgsRequest) (cliout.Format, []resources.DeprecatedResource, error) {
	items, err := controller.ListDeprecatedResources()
	if err != nil {
		return "", nil, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", nil, err
	}
	return format, items, nil
}

func renderResourceListDeprecatedResponse(w io.Writer, format cliout.Format, items []resources.DeprecatedResource) error {
	return resourcecli.WriteDeprecatedList(w, format, items)
}

func parseResourceRestoreRequest(globals globalOptions, args []string) (resourceNameRequest, error) {
	return parseResourceSingleName("Usage: vrooli resource restore <name>", "resource restore", args)
}

func runResourceRestoreRequest(controller *resources.Controller, ctx *commandContext, req resourceNameRequest) (cliout.Format, resources.RestoreReport, error) {
	report, err := controller.RestoreDeprecatedResource(req.Name)
	if err != nil {
		return "", resources.RestoreReport{}, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", resources.RestoreReport{}, err
	}
	return format, report, nil
}

func renderResourceRestoreResponse(w io.Writer, format cliout.Format, report resources.RestoreReport) error {
	return resourcecli.WriteRestoreReport(w, format, report)
}

func parseResourceArchiveToBlueprintRequest(globals globalOptions, args []string) (resourceNameRequest, error) {
	return parseResourceSingleName("Usage: vrooli resource archive-to-blueprint <name>", "resource archive-to-blueprint", args)
}

func runResourceArchiveToBlueprintRequest(controller *resources.Controller, ctx *commandContext, req resourceNameRequest) (cliout.Format, resources.BlueprintArchiveReport, error) {
	report, err := controller.ArchiveResourceToBlueprint(req.Name)
	if err != nil {
		return "", resources.BlueprintArchiveReport{}, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", resources.BlueprintArchiveReport{}, err
	}
	return format, report, nil
}

func renderResourceArchiveToBlueprintResponse(w io.Writer, format cliout.Format, report resources.BlueprintArchiveReport) error {
	return resourcecli.WriteBlueprintArchiveReport(w, format, report)
}

func parseResourceListBlueprintArchivedRequest(globals globalOptions, args []string) (resourceNoArgsRequest, error) {
	return parseResourceNoArgs("Usage: vrooli resource list-blueprint-archived", "resource list-blueprint-archived", args)
}

func runResourceListBlueprintArchivedRequest(controller *resources.Controller, ctx *commandContext, req resourceNoArgsRequest) (cliout.Format, []resources.BlueprintArchivedResource, error) {
	items, err := controller.ListBlueprintArchivedResources()
	if err != nil {
		return "", nil, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", nil, err
	}
	return format, items, nil
}

func renderResourceListBlueprintArchivedResponse(w io.Writer, format cliout.Format, items []resources.BlueprintArchivedResource) error {
	return resourcecli.WriteBlueprintArchivedList(w, format, items)
}

func parseResourceRestoreBlueprintRequest(globals globalOptions, args []string) (resourceNameRequest, error) {
	return parseResourceSingleName("Usage: vrooli resource restore-blueprint <name>", "resource restore-blueprint", args)
}

func runResourceRestoreBlueprintRequest(controller *resources.Controller, ctx *commandContext, req resourceNameRequest) (cliout.Format, resources.BlueprintRestoreReport, error) {
	report, err := controller.RestoreBlueprintArchivedResource(req.Name)
	if err != nil {
		return "", resources.BlueprintRestoreReport{}, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", resources.BlueprintRestoreReport{}, err
	}
	return format, report, nil
}

func renderResourceRestoreBlueprintResponse(w io.Writer, format cliout.Format, report resources.BlueprintRestoreReport) error {
	return resourcecli.WriteBlueprintRestoreReport(w, format, report)
}

func parseResourceArchiveGCRequest(globals globalOptions, args []string) (resourceNoArgsRequest, error) {
	return parseResourceNoArgs("Usage: vrooli resource archive gc", "resource archive gc", args)
}

func runResourceArchiveGCRequest(controller *resources.Controller, ctx *commandContext, req resourceNoArgsRequest) (cliout.Format, resources.ArchiveGCReport, error) {
	report, err := controller.GarbageCollectDeprecatedArchives(timeNowForResourceGC())
	if err != nil {
		return "", resources.ArchiveGCReport{}, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", resources.ArchiveGCReport{}, err
	}
	return format, report, nil
}

func renderResourceArchiveGCResponse(w io.Writer, format cliout.Format, report resources.ArchiveGCReport) error {
	return resourcecli.WriteArchiveGCReport(w, format, report, "deprecated resource")
}

func parseResourceArchiveBlueprintGCRequest(globals globalOptions, args []string) (resourceNoArgsRequest, error) {
	return parseResourceNoArgs("Usage: vrooli resource archive gc-blueprints", "resource archive gc-blueprints", args)
}

func runResourceArchiveBlueprintGCRequest(controller *resources.Controller, ctx *commandContext, req resourceNoArgsRequest) (cliout.Format, resources.ArchiveGCReport, error) {
	report, err := controller.GarbageCollectBlueprintArchives(timeNowForResourceGC())
	if err != nil {
		return "", resources.ArchiveGCReport{}, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", resources.ArchiveGCReport{}, err
	}
	return format, report, nil
}

func renderResourceArchiveBlueprintGCResponse(w io.Writer, format cliout.Format, report resources.ArchiveGCReport) error {
	return resourcecli.WriteArchiveGCReport(w, format, report, "blueprint resource")
}

func parseResourceBlueprintListRequest(globals globalOptions, args []string) (resourceNoArgsRequest, error) {
	return parseResourceNoArgs("Usage: vrooli resource blueprint list", "resource blueprint list", args)
}

func runResourceBlueprintListRequest(controller *resources.Controller, ctx *commandContext, req resourceNoArgsRequest) (cliout.Format, resourceBlueprintsResponse, error) {
	items, err := controller.ListBlueprints()
	if err != nil {
		return "", resourceBlueprintsResponse{}, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", resourceBlueprintsResponse{}, err
	}
	return format, resourceBlueprintsResponse{Items: items}, nil
}

func renderResourceBlueprintListResponse(w io.Writer, format cliout.Format, resp resourceBlueprintsResponse) error {
	return resourcecli.WriteBlueprintList(w, format, resp.Items)
}

func parseResourceBlueprintInfoRequest(globals globalOptions, args []string) (resourceNameRequest, error) {
	return parseResourceSingleName("Usage: vrooli resource blueprint info <name>", "resource blueprint info", args)
}

func runResourceBlueprintInfoRequest(controller *resources.Controller, ctx *commandContext, req resourceNameRequest) (cliout.Format, resourceBlueprintResponse, error) {
	item, err := controller.Blueprint(req.Name)
	if err != nil {
		return "", resourceBlueprintResponse{}, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", resourceBlueprintResponse{}, err
	}
	return format, resourceBlueprintResponse{Item: item}, nil
}

func renderResourceBlueprintInfoResponse(w io.Writer, format cliout.Format, resp resourceBlueprintResponse) error {
	return resourcecli.WriteBlueprintInfo(w, format, resp.Item)
}

func parseResourceBlueprintSearchRequest(globals globalOptions, args []string) (resourceBlueprintSearchRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return resourceBlueprintSearchRequest{}, commandHelpOnly("Usage: vrooli resource blueprint search <query>")
		}
	}
	if len(args) != 1 {
		return resourceBlueprintSearchRequest{}, usageErrorf("resource blueprint", "resource blueprint search requires exactly one query")
	}
	return resourceBlueprintSearchRequest{Query: args[0]}, nil
}

func runResourceBlueprintSearchRequest(controller *resources.Controller, ctx *commandContext, req resourceBlueprintSearchRequest) (cliout.Format, resourceBlueprintSearchResponse, error) {
	items, err := controller.SearchBlueprints(req.Query)
	if err != nil {
		return "", resourceBlueprintSearchResponse{}, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", resourceBlueprintSearchResponse{}, err
	}
	return format, resourceBlueprintSearchResponse{Query: req.Query, Items: items}, nil
}

func renderResourceBlueprintSearchResponse(w io.Writer, format cliout.Format, resp resourceBlueprintSearchResponse) error {
	return resourcecli.WriteBlueprintSearch(w, format, resp.Query, resp.Items)
}

func parseResourceBlueprintValidateRequest(globals globalOptions, args []string) (resourceNoArgsRequest, error) {
	return parseResourceNoArgs("Usage: vrooli resource blueprint validate", "resource blueprint validate", args)
}

func runResourceBlueprintValidateRequest(controller *resources.Controller, ctx *commandContext, req resourceNoArgsRequest) (cliout.Format, resources.BlueprintValidationReport, error) {
	report, err := controller.ValidateBlueprints()
	if err != nil {
		return "", resources.BlueprintValidationReport{}, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", resources.BlueprintValidationReport{}, err
	}
	return format, report, nil
}

func renderResourceBlueprintValidateResponse(w io.Writer, format cliout.Format, report resources.BlueprintValidationReport) error {
	return resourcecli.WriteBlueprintValidationReport(w, format, report)
}

func parseResourceStartAllRequest(globals globalOptions, args []string) (resourceNoArgsRequest, error) {
	return parseResourceNoArgs("Usage: vrooli resource start-all", "resource start-all", args)
}

type resourceControlReportResponse struct {
	Start *control.StartReport
	Stop  *control.StopReport
}

func runResourceStartAllRequest(controller *resources.Controller, ctx *commandContext, req resourceNoArgsRequest) (cliout.Format, resourceControlReportResponse, error) {
	report, err := controller.StartAll(ctx.Stdout, ctx.Stderr)
	if err != nil {
		return "", resourceControlReportResponse{}, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", resourceControlReportResponse{}, err
	}
	return format, resourceControlReportResponse{Start: &report}, nil
}

func parseResourceStopAllRequest(globals globalOptions, args []string) (resourceNoArgsRequest, error) {
	return parseResourceNoArgs("Usage: vrooli resource stop-all", "resource stop-all", args)
}

func runResourceStopAllRequest(controller *resources.Controller, ctx *commandContext, req resourceNoArgsRequest) (cliout.Format, resourceControlReportResponse, error) {
	report, err := controller.StopAll(ctx.Stdout, ctx.Stderr)
	if err != nil {
		return "", resourceControlReportResponse{}, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", resourceControlReportResponse{}, err
	}
	return format, resourceControlReportResponse{Stop: &report}, nil
}

func renderResourceControlReportResponse(w io.Writer, format cliout.Format, resp resourceControlReportResponse) error {
	if resp.Start != nil {
		report := *resp.Start
		return resourcecli.WriteControlReport(w, format, "report", "Started", report, report.Started, report.Failed)
	}
	report := *resp.Stop
	return resourcecli.WriteControlReport(w, format, "report", "Stopped", report, report.Stopped, report.Failed)
}
