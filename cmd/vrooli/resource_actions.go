package main

import (
	"io"

	resourceapp "github.com/vrooli/vrooli/internal/app/resource"
	"github.com/vrooli/vrooli/internal/cli/resourcecli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/resources"
)

type resourceCommandAction[Req any, Resp any] struct {
	parse  func(globals globalOptions, args []string) (Req, error)
	run    func(controller *resources.Controller, ctx *commandContext, req Req) (cliout.Format, Resp, error)
	render func(w io.Writer, format cliout.Format, resp Resp) error
}

func executeResourceCommandWithApp[Req any, Resp any](app *App, ctx *commandContext, controller *resources.Controller, args []string, action resourceCommandAction[Req, Resp]) error {
	return bindResourceCommand(action.parse, action.run, action.render)(app, ctx, controller, args)
}

func executeResourceCommand[Req any, Resp any](controller *resources.Controller, globals globalOptions, args []string, stdout, stderr io.Writer, action resourceCommandAction[Req, Resp]) error {
	app, ctx := newConfiguredCommandContext("", globals, stdout, stderr)
	if controller != nil {
		ctx.Root = controller.Root
	}
	return executeResourceCommandWithApp(app, ctx, controller, args, action)
}

type (
	resourceNoArgsRequest = resourcecli.NoArgsRequest
	resourceNameRequest   = resourcecli.NameRequest
)

type (
	resourceStatusRequest          = resourcecli.StatusRequest
	resourceValidateRequest        = resourcecli.ValidateRequest
	resourceBlueprintSearchRequest = resourcecli.BlueprintSearchRequest
)

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

func parseResourceListRequest(globals globalOptions, args []string) (resourceNoArgsRequest, error) {
	_ = globals
	req, err := resourcecli.ParseListRequest(args)
	return req, mapResourceParseError("resource list", err)
}

func runResourceListRequest(controller *resources.Controller, ctx *commandContext, req resourceNoArgsRequest) (cliout.Format, resourceListResponse, error) {
	_ = req
	items, err := newResourceCommandService(ctx, controller).List()
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

func parseResourceValidateRequest(globals globalOptions, args []string) (resourceValidateRequest, error) {
	_ = globals
	req, err := resourcecli.ParseValidateRequest(args)
	return req, mapResourceParseError("resource validate", err)
}

func runResourceValidateRequest(controller *resources.Controller, ctx *commandContext, req resourceValidateRequest) (cliout.Format, resources.ResourceValidationReport, error) {
	report, err := newResourceCommandService(ctx, controller).Validate(req.Name)
	if err != nil {
		return "", resources.ResourceValidationReport{}, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", resources.ResourceValidationReport{}, err
	}
	return format, report, nil
}

func renderResourceValidateResponse(w io.Writer, format cliout.Format, report resources.ResourceValidationReport) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{"success": report.Passed, "report": report})
	}
	status := "passed"
	if !report.Passed {
		status = "failed"
	}
	if _, err := io.WriteString(w, "Resource validation "+status+"\n"); err != nil {
		return err
	}
	for _, item := range report.Items {
		if len(item.Issues) == 0 {
			continue
		}
		if _, err := io.WriteString(w, "- "+item.Name+"\n"); err != nil {
			return err
		}
		for _, issue := range item.Issues {
			if _, err := io.WriteString(w, "  ["+issue.Severity+"] "+issue.Message+"\n"); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseResourceStatusRequest(globals globalOptions, args []string) (resourceStatusRequest, error) {
	_ = globals
	req, err := resourcecli.ParseStatusRequest(args)
	return req, mapResourceParseError("resource status", err)
}

func runResourceStatusRequest(controller *resources.Controller, ctx *commandContext, req resourceStatusRequest) (cliout.Format, resourceStatusResponse, error) {
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", resourceStatusResponse{}, err
	}
	items, item, err := newResourceCommandService(ctx, controller).Status(req.Name, req.Fast)
	if err != nil {
		return "", resourceStatusResponse{}, err
	}
	if item == nil {
		return format, resourceStatusResponse{Items: items}, nil
	}
	return format, resourceStatusResponse{Item: item}, nil
}

func renderResourceStatusResponse(w io.Writer, format cliout.Format, resp resourceStatusResponse) error {
	if resp.Item != nil {
		return resourcecli.WriteStatus(w, format, *resp.Item)
	}
	return resourcecli.WriteStatuses(w, format, resp.Items)
}

func parseResourceInfoRequest(globals globalOptions, args []string) (resourceNameRequest, error) {
	_ = globals
	req, err := resourcecli.ParseInfoRequest(args)
	return req, mapResourceParseError("resource info", err)
}

func runResourceInfoRequest(controller *resources.Controller, ctx *commandContext, req resourceNameRequest) (cliout.Format, resourceStatusResponse, error) {
	item, err := newResourceCommandService(ctx, controller).Info(req.Name)
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
	_ = globals
	req, err := resourcecli.ParseDeprecateRequest(args)
	return req, mapResourceParseError("resource deprecate", err)
}

func runResourceDeprecateRequest(controller *resources.Controller, ctx *commandContext, req resourceNameRequest) (cliout.Format, resources.DeprecationReport, error) {
	report, err := newResourceCommandService(ctx, controller).Deprecate(req.Name)
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
	_ = globals
	req, err := resourcecli.ParseListDeprecatedRequest(args)
	return req, mapResourceParseError("resource list-deprecated", err)
}

func runResourceListDeprecatedRequest(controller *resources.Controller, ctx *commandContext, req resourceNoArgsRequest) (cliout.Format, []resources.DeprecatedResource, error) {
	_ = req
	items, err := newResourceCommandService(ctx, controller).ListDeprecated()
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
	_ = globals
	req, err := resourcecli.ParseRestoreRequest(args)
	return req, mapResourceParseError("resource restore", err)
}

func runResourceRestoreRequest(controller *resources.Controller, ctx *commandContext, req resourceNameRequest) (cliout.Format, resources.RestoreReport, error) {
	report, err := newResourceCommandService(ctx, controller).Restore(req.Name)
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
	_ = globals
	req, err := resourcecli.ParseArchiveToBlueprintRequest(args)
	return req, mapResourceParseError("resource archive-to-blueprint", err)
}

func runResourceArchiveToBlueprintRequest(controller *resources.Controller, ctx *commandContext, req resourceNameRequest) (cliout.Format, resources.BlueprintArchiveReport, error) {
	report, err := newResourceCommandService(ctx, controller).ArchiveToBlueprint(req.Name)
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
	_ = globals
	req, err := resourcecli.ParseListBlueprintArchivedRequest(args)
	return req, mapResourceParseError("resource list-blueprint-archived", err)
}

func runResourceListBlueprintArchivedRequest(controller *resources.Controller, ctx *commandContext, req resourceNoArgsRequest) (cliout.Format, []resources.BlueprintArchivedResource, error) {
	_ = req
	items, err := newResourceCommandService(ctx, controller).ListBlueprintArchived()
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
	_ = globals
	req, err := resourcecli.ParseRestoreBlueprintRequest(args)
	return req, mapResourceParseError("resource restore-blueprint", err)
}

func runResourceRestoreBlueprintRequest(controller *resources.Controller, ctx *commandContext, req resourceNameRequest) (cliout.Format, resources.BlueprintRestoreReport, error) {
	report, err := newResourceCommandService(ctx, controller).RestoreBlueprint(req.Name)
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
	_ = globals
	req, err := resourcecli.ParseArchiveGCRequest(args)
	return req, mapResourceParseError("resource archive gc", err)
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
	_ = globals
	req, err := resourcecli.ParseArchiveBlueprintGCRequest(args)
	return req, mapResourceParseError("resource archive gc-blueprints", err)
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
	_ = globals
	req, err := resourcecli.ParseBlueprintListRequest(args)
	return req, mapResourceParseError("resource blueprint list", err)
}

func runResourceBlueprintListRequest(controller *resources.Controller, ctx *commandContext, req resourceNoArgsRequest) (cliout.Format, resourceBlueprintsResponse, error) {
	_ = req
	items, err := newResourceCommandService(ctx, controller).BlueprintList()
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
	_ = globals
	req, err := resourcecli.ParseBlueprintInfoRequest(args)
	return req, mapResourceParseError("resource blueprint info", err)
}

func runResourceBlueprintInfoRequest(controller *resources.Controller, ctx *commandContext, req resourceNameRequest) (cliout.Format, resourceBlueprintResponse, error) {
	item, err := newResourceCommandService(ctx, controller).BlueprintInfo(req.Name)
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
	_ = globals
	req, err := resourcecli.ParseBlueprintSearchRequest(args)
	return req, mapResourceParseError("resource blueprint search", err)
}

func runResourceBlueprintSearchRequest(controller *resources.Controller, ctx *commandContext, req resourceBlueprintSearchRequest) (cliout.Format, resourceBlueprintSearchResponse, error) {
	items, err := newResourceCommandService(ctx, controller).BlueprintSearch(req.Query)
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
	_ = globals
	req, err := resourcecli.ParseBlueprintValidateRequest(args)
	return req, mapResourceParseError("resource blueprint validate", err)
}

func runResourceBlueprintValidateRequest(controller *resources.Controller, ctx *commandContext, req resourceNoArgsRequest) (cliout.Format, resources.BlueprintValidationReport, error) {
	_ = req
	report, err := newResourceCommandService(ctx, controller).BlueprintValidate()
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
	_ = globals
	req, err := resourcecli.ParseStartAllRequest(args)
	return req, mapResourceParseError("resource start-all", err)
}

func runResourceStartAllRequest(controller *resources.Controller, ctx *commandContext, req resourceNoArgsRequest) (cliout.Format, resourceapp.ControlReportResponse, error) {
	_ = req
	report, err := newResourceCommandService(ctx, controller).StartAll()
	if err != nil {
		return "", resourceapp.ControlReportResponse{}, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", resourceapp.ControlReportResponse{}, err
	}
	return format, report, nil
}

func parseResourceStopAllRequest(globals globalOptions, args []string) (resourceNoArgsRequest, error) {
	_ = globals
	req, err := resourcecli.ParseStopAllRequest(args)
	return req, mapResourceParseError("resource stop-all", err)
}

func runResourceStopAllRequest(controller *resources.Controller, ctx *commandContext, req resourceNoArgsRequest) (cliout.Format, resourceapp.ControlReportResponse, error) {
	_ = req
	report, err := newResourceCommandService(ctx, controller).StopAll()
	if err != nil {
		return "", resourceapp.ControlReportResponse{}, err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", resourceapp.ControlReportResponse{}, err
	}
	return format, report, nil
}

func renderResourceControlReportResponse(w io.Writer, format cliout.Format, resp resourceapp.ControlReportResponse) error {
	if resp.Start != nil {
		report := *resp.Start
		return resourcecli.WriteControlReport(w, format, "report", "Started", report, report.Started, report.Failed)
	}
	report := *resp.Stop
	return resourcecli.WriteControlReport(w, format, "report", "Stopped", report, report.Stopped, report.Failed)
}

func newResourceCommandService(ctx *commandContext, controller *resources.Controller) resourceapp.Service {
	return resourceapp.Service{
		Resources: controller,
		Stdout:    ctx.Stdout,
		Stderr:    ctx.Stderr,
	}
}

func mapResourceParseError(command string, err error) error {
	if err == nil {
		return nil
	}
	if helpErr, ok := err.(interface{ HelpText() string }); ok {
		return commandHelpOnly(helpErr.HelpText())
	}
	return usageErrorf(command, err.Error())
}
