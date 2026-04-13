package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/resources"
)

type resourceCommandAction[Req any, Resp any] struct {
	parse  func(globals globalOptions, args []string) (Req, error)
	run    func(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer, req Req) (cliout.Format, Resp, error)
	render func(w io.Writer, format cliout.Format, resp Resp) error
}

func executeResourceCommand[Req any, Resp any](controller *resources.Controller, globals globalOptions, args []string, stdout, stderr io.Writer, action resourceCommandAction[Req, Resp]) error {
	req, err := action.parse(globals, args)
	if err != nil {
		var helpErr commandHelpError
		if errors.As(err, &helpErr) {
			_, _ = fmt.Fprintln(stdout, helpErr.message)
			return nil
		}
		return err
	}
	format, resp, err := action.run(controller, globals, stdout, stderr, req)
	if err != nil {
		return err
	}
	return action.render(stdout, format, resp)
}

type resourceNoArgsRequest struct{}
type resourceNameRequest struct {
	Name string
}
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
	Item resources.Status
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

func runResourceListRequest(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer, req resourceNoArgsRequest) (cliout.Format, resourceListResponse, error) {
	items, err := controller.Discover()
	if err != nil {
		return "", resourceListResponse{}, err
	}
	format, err := parseOutputFormat(globals)
	if err != nil {
		return "", resourceListResponse{}, err
	}
	return format, resourceListResponse{Items: items}, nil
}

func renderResourceListResponse(w io.Writer, format cliout.Format, resp resourceListResponse) error {
	return writeResourceList(w, format, resp.Items)
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

func runResourceStatusRequest(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer, req resourceStatusRequest) (cliout.Format, any, error) {
	format, err := parseOutputFormat(globals)
	if err != nil {
		return "", nil, err
	}
	if req.Name == "" {
		items, err := controller.ListStatuses(req.Fast, false)
		if err != nil {
			return "", nil, err
		}
		return format, resourceStatusesResponse{Items: items}, nil
	}
	item, err := controller.Status(req.Name, req.Fast)
	if err != nil {
		return "", nil, err
	}
	return format, resourceStatusResponse{Item: item}, nil
}

func renderResourceStatusResponse(w io.Writer, format cliout.Format, resp any) error {
	switch typed := resp.(type) {
	case resourceStatusesResponse:
		return writeResourceStatuses(w, format, typed.Items)
	case resourceStatusResponse:
		return writeResourceStatus(w, format, typed.Item)
	default:
		return fmt.Errorf("unexpected resource status response type %T", resp)
	}
}

func parseResourceInfoRequest(globals globalOptions, args []string) (resourceNameRequest, error) {
	return parseResourceSingleName("Usage: vrooli resource info <name>", "resource info", args)
}

func runResourceInfoRequest(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer, req resourceNameRequest) (cliout.Format, resourceStatusResponse, error) {
	item, err := controller.Status(req.Name, true)
	if err != nil {
		return "", resourceStatusResponse{}, err
	}
	format, err := parseOutputFormat(globals)
	if err != nil {
		return "", resourceStatusResponse{}, err
	}
	return format, resourceStatusResponse{Item: item}, nil
}

func renderResourceInfoResponse(w io.Writer, format cliout.Format, resp resourceStatusResponse) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "resource", resp.Item)
	}
	return writeResourceStatus(w, format, resp.Item)
}

func parseResourceDeprecateRequest(globals globalOptions, args []string) (resourceNameRequest, error) {
	return parseResourceSingleName("Usage: vrooli resource deprecate <name>", "resource deprecate", args)
}

func runResourceDeprecateRequest(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer, req resourceNameRequest) (cliout.Format, resources.DeprecationReport, error) {
	report, err := controller.DeprecateResource(req.Name)
	if err != nil {
		return "", resources.DeprecationReport{}, err
	}
	format, err := parseOutputFormat(globals)
	if err != nil {
		return "", resources.DeprecationReport{}, err
	}
	return format, report, nil
}

func renderResourceDeprecateResponse(w io.Writer, format cliout.Format, report resources.DeprecationReport) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "report", report)
	}
	_, _ = fmt.Fprintf(w, "Deprecated %s\n", report.Resource.Name)
	if report.ArchiveDir != "" {
		_, _ = fmt.Fprintf(w, "Archive: %s\n", report.ArchiveDir)
	}
	return nil
}

func parseResourceListDeprecatedRequest(globals globalOptions, args []string) (resourceNoArgsRequest, error) {
	return parseResourceNoArgs("Usage: vrooli resource list-deprecated", "resource list-deprecated", args)
}

func runResourceListDeprecatedRequest(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer, req resourceNoArgsRequest) (cliout.Format, []resources.DeprecatedResource, error) {
	items, err := controller.ListDeprecatedResources()
	if err != nil {
		return "", nil, err
	}
	format, err := parseOutputFormat(globals)
	if err != nil {
		return "", nil, err
	}
	return format, items, nil
}

func renderResourceListDeprecatedResponse(w io.Writer, format cliout.Format, items []resources.DeprecatedResource) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "resources", items)
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		state := "deprecated"
		if strings.TrimSpace(item.PurgedAt) != "" {
			state = "purged"
		}
		rows = append(rows, []string{item.Name, state, item.DeprecatedAt, item.PurgeAfter, item.Replacement})
	}
	return cliout.RenderTable(w, []string{"Name", "State", "Deprecated", "Purge After", "Replacement"}, rows)
}

func parseResourceRestoreRequest(globals globalOptions, args []string) (resourceNameRequest, error) {
	return parseResourceSingleName("Usage: vrooli resource restore <name>", "resource restore", args)
}

func runResourceRestoreRequest(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer, req resourceNameRequest) (cliout.Format, resources.RestoreReport, error) {
	report, err := controller.RestoreDeprecatedResource(req.Name)
	if err != nil {
		return "", resources.RestoreReport{}, err
	}
	format, err := parseOutputFormat(globals)
	if err != nil {
		return "", resources.RestoreReport{}, err
	}
	return format, report, nil
}

func renderResourceRestoreResponse(w io.Writer, format cliout.Format, report resources.RestoreReport) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "report", report)
	}
	_, _ = fmt.Fprintf(w, "Restored %s to %s\n", report.Resource.Name, report.RestoredPath)
	return nil
}

func parseResourceArchiveToBlueprintRequest(globals globalOptions, args []string) (resourceNameRequest, error) {
	return parseResourceSingleName("Usage: vrooli resource archive-to-blueprint <name>", "resource archive-to-blueprint", args)
}

func runResourceArchiveToBlueprintRequest(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer, req resourceNameRequest) (cliout.Format, resources.BlueprintArchiveReport, error) {
	report, err := controller.ArchiveResourceToBlueprint(req.Name)
	if err != nil {
		return "", resources.BlueprintArchiveReport{}, err
	}
	format, err := parseOutputFormat(globals)
	if err != nil {
		return "", resources.BlueprintArchiveReport{}, err
	}
	return format, report, nil
}

func renderResourceArchiveToBlueprintResponse(w io.Writer, format cliout.Format, report resources.BlueprintArchiveReport) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "report", report)
	}
	_, _ = fmt.Fprintf(w, "Archived %s to blueprint-only state\n", report.Resource.Name)
	if report.ArchiveDir != "" {
		_, _ = fmt.Fprintf(w, "Archive: %s\n", report.ArchiveDir)
	}
	return nil
}

func parseResourceListBlueprintArchivedRequest(globals globalOptions, args []string) (resourceNoArgsRequest, error) {
	return parseResourceNoArgs("Usage: vrooli resource list-blueprint-archived", "resource list-blueprint-archived", args)
}

func runResourceListBlueprintArchivedRequest(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer, req resourceNoArgsRequest) (cliout.Format, []resources.BlueprintArchivedResource, error) {
	items, err := controller.ListBlueprintArchivedResources()
	if err != nil {
		return "", nil, err
	}
	format, err := parseOutputFormat(globals)
	if err != nil {
		return "", nil, err
	}
	return format, items, nil
}

func renderResourceListBlueprintArchivedResponse(w io.Writer, format cliout.Format, items []resources.BlueprintArchivedResource) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "resources", items)
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		state := "blueprint-archived"
		if strings.TrimSpace(item.PurgedAt) != "" {
			state = "purged"
		}
		rows = append(rows, []string{item.Name, state, item.ArchivedAt, item.PurgeAfter, item.BlueprintName})
	}
	return cliout.RenderTable(w, []string{"Name", "State", "Archived", "Purge After", "Blueprint"}, rows)
}

func parseResourceRestoreBlueprintRequest(globals globalOptions, args []string) (resourceNameRequest, error) {
	return parseResourceSingleName("Usage: vrooli resource restore-blueprint <name>", "resource restore-blueprint", args)
}

func runResourceRestoreBlueprintRequest(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer, req resourceNameRequest) (cliout.Format, resources.BlueprintRestoreReport, error) {
	report, err := controller.RestoreBlueprintArchivedResource(req.Name)
	if err != nil {
		return "", resources.BlueprintRestoreReport{}, err
	}
	format, err := parseOutputFormat(globals)
	if err != nil {
		return "", resources.BlueprintRestoreReport{}, err
	}
	return format, report, nil
}

func renderResourceRestoreBlueprintResponse(w io.Writer, format cliout.Format, report resources.BlueprintRestoreReport) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "report", report)
	}
	_, _ = fmt.Fprintf(w, "Restored blueprint-archived %s to %s\n", report.Resource.Name, report.RestoredPath)
	return nil
}

func parseResourceArchiveGCRequest(globals globalOptions, args []string) (resourceNoArgsRequest, error) {
	return parseResourceNoArgs("Usage: vrooli resource archive gc", "resource archive gc", args)
}

func runResourceArchiveGCRequest(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer, req resourceNoArgsRequest) (cliout.Format, resources.ArchiveGCReport, error) {
	report, err := controller.GarbageCollectDeprecatedArchives(timeNowForResourceGC())
	if err != nil {
		return "", resources.ArchiveGCReport{}, err
	}
	format, err := parseOutputFormat(globals)
	if err != nil {
		return "", resources.ArchiveGCReport{}, err
	}
	return format, report, nil
}

func renderResourceArchiveGCResponse(w io.Writer, format cliout.Format, report resources.ArchiveGCReport) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "report", report)
	}
	_, _ = fmt.Fprintf(w, "Purged %d deprecated resource archives\n", len(report.Removed))
	return nil
}

func parseResourceArchiveBlueprintGCRequest(globals globalOptions, args []string) (resourceNoArgsRequest, error) {
	return parseResourceNoArgs("Usage: vrooli resource archive gc-blueprints", "resource archive gc-blueprints", args)
}

func runResourceArchiveBlueprintGCRequest(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer, req resourceNoArgsRequest) (cliout.Format, resources.ArchiveGCReport, error) {
	report, err := controller.GarbageCollectBlueprintArchives(timeNowForResourceGC())
	if err != nil {
		return "", resources.ArchiveGCReport{}, err
	}
	format, err := parseOutputFormat(globals)
	if err != nil {
		return "", resources.ArchiveGCReport{}, err
	}
	return format, report, nil
}

func renderResourceArchiveBlueprintGCResponse(w io.Writer, format cliout.Format, report resources.ArchiveGCReport) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "report", report)
	}
	_, _ = fmt.Fprintf(w, "Purged %d blueprint resource archives\n", len(report.Removed))
	return nil
}

func parseResourceBlueprintListRequest(globals globalOptions, args []string) (resourceNoArgsRequest, error) {
	return parseResourceNoArgs("Usage: vrooli resource blueprint list", "resource blueprint list", args)
}

func runResourceBlueprintListRequest(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer, req resourceNoArgsRequest) (cliout.Format, resourceBlueprintsResponse, error) {
	items, err := controller.ListBlueprints()
	if err != nil {
		return "", resourceBlueprintsResponse{}, err
	}
	format, err := parseOutputFormat(globals)
	if err != nil {
		return "", resourceBlueprintsResponse{}, err
	}
	return format, resourceBlueprintsResponse{Items: items}, nil
}

func renderResourceBlueprintListResponse(w io.Writer, format cliout.Format, resp resourceBlueprintsResponse) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "blueprints", resp.Items)
	}
	rows := make([][]string, 0, len(resp.Items))
	for _, item := range resp.Items {
		rows = append(rows, []string{item.Name, item.Category, item.Status, item.SuggestedTemplate, item.LastReviewed})
	}
	return cliout.RenderTable(w, []string{"Name", "Category", "Status", "Template", "Reviewed"}, rows)
}

func parseResourceBlueprintInfoRequest(globals globalOptions, args []string) (resourceNameRequest, error) {
	return parseResourceSingleName("Usage: vrooli resource blueprint info <name>", "resource blueprint info", args)
}

func runResourceBlueprintInfoRequest(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer, req resourceNameRequest) (cliout.Format, resourceBlueprintResponse, error) {
	item, err := controller.Blueprint(req.Name)
	if err != nil {
		return "", resourceBlueprintResponse{}, err
	}
	format, err := parseOutputFormat(globals)
	if err != nil {
		return "", resourceBlueprintResponse{}, err
	}
	return format, resourceBlueprintResponse{Item: item}, nil
}

func renderResourceBlueprintInfoResponse(w io.Writer, format cliout.Format, resp resourceBlueprintResponse) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "blueprint", resp.Item)
	}
	rows := [][]string{
		{"Name", resp.Item.Name},
		{"Display Name", resp.Item.DisplayName},
		{"Category", resp.Item.Category},
		{"Status", resp.Item.Status},
		{"Integration Kind", resp.Item.IntegrationKind},
		{"Template", resp.Item.SuggestedTemplate},
		{"Reviewed", resp.Item.LastReviewed},
		{"Summary", resp.Item.Summary},
		{"Why It Matters", resp.Item.WhyItMatters},
	}
	return cliout.RenderTable(w, []string{"Field", "Value"}, rows)
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

func runResourceBlueprintSearchRequest(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer, req resourceBlueprintSearchRequest) (cliout.Format, resourceBlueprintSearchResponse, error) {
	items, err := controller.SearchBlueprints(req.Query)
	if err != nil {
		return "", resourceBlueprintSearchResponse{}, err
	}
	format, err := parseOutputFormat(globals)
	if err != nil {
		return "", resourceBlueprintSearchResponse{}, err
	}
	return format, resourceBlueprintSearchResponse{Query: req.Query, Items: items}, nil
}

func renderResourceBlueprintSearchResponse(w io.Writer, format cliout.Format, resp resourceBlueprintSearchResponse) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success":    true,
			"query":      resp.Query,
			"blueprints": resp.Items,
		})
	}
	rows := make([][]string, 0, len(resp.Items))
	for _, item := range resp.Items {
		rows = append(rows, []string{item.Name, item.Category, item.Status, item.Summary})
	}
	return cliout.RenderTable(w, []string{"Name", "Category", "Status", "Summary"}, rows)
}

func parseResourceBlueprintValidateRequest(globals globalOptions, args []string) (resourceNoArgsRequest, error) {
	return parseResourceNoArgs("Usage: vrooli resource blueprint validate", "resource blueprint validate", args)
}

func runResourceBlueprintValidateRequest(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer, req resourceNoArgsRequest) (cliout.Format, resources.BlueprintValidationReport, error) {
	report, err := controller.ValidateBlueprints()
	if err != nil {
		return "", resources.BlueprintValidationReport{}, err
	}
	format, err := parseOutputFormat(globals)
	if err != nil {
		return "", resources.BlueprintValidationReport{}, err
	}
	return format, report, nil
}

func renderResourceBlueprintValidateResponse(w io.Writer, format cliout.Format, report resources.BlueprintValidationReport) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "report", report)
	}
	_, _ = fmt.Fprintf(w, "Validated %d resource blueprints\n", report.Count)
	return nil
}

func parseResourceStartAllRequest(globals globalOptions, args []string) (resourceNoArgsRequest, error) {
	return parseResourceNoArgs("Usage: vrooli resource start-all", "resource start-all", args)
}

func runResourceStartAllRequest(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer, req resourceNoArgsRequest) (cliout.Format, any, error) {
	report, err := controller.StartAll(stdout, stderr)
	if err != nil {
		return "", nil, err
	}
	format, err := parseOutputFormat(globals)
	if err != nil {
		return "", nil, err
	}
	return format, report, nil
}

func parseResourceStopAllRequest(globals globalOptions, args []string) (resourceNoArgsRequest, error) {
	return parseResourceNoArgs("Usage: vrooli resource stop-all", "resource stop-all", args)
}

func runResourceStopAllRequest(controller *resources.Controller, globals globalOptions, stdout, stderr io.Writer, req resourceNoArgsRequest) (cliout.Format, any, error) {
	report, err := controller.StopAll(stdout, stderr)
	if err != nil {
		return "", nil, err
	}
	format, err := parseOutputFormat(globals)
	if err != nil {
		return "", nil, err
	}
	return format, report, nil
}

func renderResourceControlReportResponse(w io.Writer, format cliout.Format, resp any) error {
	switch report := resp.(type) {
	case control.StartReport:
		return writeControlReport(w, format, "report", "Started", report, report.Started, report.Failed)
	case control.StopReport:
		return writeControlReport(w, format, "report", "Stopped", report, report.Stopped, report.Failed)
	default:
		return fmt.Errorf("unexpected control report type %T", resp)
	}
}
