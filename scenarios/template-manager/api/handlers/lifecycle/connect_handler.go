package lifecycle

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/catalog"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templateengine"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/validationrunner"

	"connectrpc.com/connect"
	lifecyclev1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/lifecycle"
)

type connectHandler struct {
	engine     *templateengine.Engine
	validation *validationrunner.Service
}

func NewConnectHandler(engine *templateengine.Engine, validation *validationrunner.Service) *connectHandler {
	return &connectHandler{engine: engine, validation: validation}
}

func (h *connectHandler) GenerateScenario(ctx context.Context, req *connect.Request[lifecyclev1.GenerateScenarioRequest]) (*connect.Response[lifecyclev1.GenerateScenarioResponse], error) {
	engine, err := h.requireEngine()
	if err != nil {
		return nil, err
	}
	info, err := engine.ShowTemplate(ctx, req.Msg.Template)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	result, err := engine.GenerateScenario(ctx, templatecontracts.GenerateRequest{
		TemplateInfo: info,
		Options: templatecontracts.GenerateOptions{
			Destination: req.Msg.Destination,
			Design:      req.Msg.Design,
			Force:       req.Msg.Force,
			DryRun:      req.Msg.DryRun,
			RunHooks:    req.Msg.RunHooks,
			Values:      generateScenarioValues(req.Msg),
		},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&lifecyclev1.GenerateScenarioResponse{
		Template:        result.TemplateName,
		DisplayName:     result.DisplayName,
		Destination:     result.Destination,
		DryRun:          result.DryRun,
		RunHooks:        result.RunHooks,
		ManifestVersion: result.Manifest.Version,
		DesignKit:       result.Design.KitID,
	}), nil
}

func (h *connectHandler) OrientScenario(ctx context.Context, req *connect.Request[lifecyclev1.OrientScenarioRequest]) (*connect.Response[lifecyclev1.OrientScenarioResponse], error) {
	engine, err := h.requireEngine()
	if err != nil {
		return nil, err
	}
	report, err := engine.OrientScenario(ctx, templatecontracts.OrientationRequest{Name: req.Msg.Scenario, Finalize: req.Msg.Finalize, JSON: true})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	next := ""
	if report.NextStep != nil {
		next = report.NextStep.ID
	}
	return connect.NewResponse(&lifecyclev1.OrientScenarioResponse{
		Scenario:         report.Scenario,
		ScenarioPath:     report.ScenarioPath,
		Finalized:        report.Finalized,
		Completed:        int32(report.Completed),
		Required:         int32(report.Required),
		FinalizeRequired: report.FinalizeRequired,
		NextStep:         next,
		Message:          report.Message,
	}), nil
}

func (h *connectHandler) DetemplateScenario(ctx context.Context, req *connect.Request[lifecyclev1.DetemplateScenarioRequest]) (*connect.Response[lifecyclev1.DetemplateScenarioResponse], error) {
	engine, err := h.requireEngine()
	if err != nil {
		return nil, err
	}
	result, err := engine.DetemplateScenario(ctx, templatecontracts.DetemplateRequest{Name: req.Msg.Scenario, DryRun: req.Msg.DryRun, JSON: true})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&lifecyclev1.DetemplateScenarioResponse{
		Scenario:      result.Scenario,
		ScenarioPath:  result.ScenarioPath,
		Marker:        result.Marker,
		DryRun:        result.DryRun,
		BlocksRemoved: int32(result.BlocksRemoved),
		LinesStripped: int32(result.LinesStripped),
		PathsDeleted:  result.PathsDeleted,
		Message:       result.Message,
	}), nil
}

func (h *connectHandler) DestroyScenario(ctx context.Context, req *connect.Request[lifecyclev1.DestroyScenarioRequest]) (*connect.Response[lifecyclev1.DestroyScenarioResponse], error) {
	engine, err := h.requireEngine()
	if err != nil {
		return nil, err
	}
	result, err := engine.DestroyScenario(ctx, templatecontracts.DestroyRequest{
		Name:      req.Msg.Scenario,
		DryRun:    req.Msg.DryRun,
		ProtoOnly: req.Msg.ProtoOnly,
		Force:     req.Msg.Force,
	})
	if err != nil {
		// A refused destroy (missing --force, bad id) is a caller error, not a
		// server fault: surfacing it as Internal would hide the actionable
		// message behind a 500.
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&lifecyclev1.DestroyScenarioResponse{
		Scenario:           result.Scenario,
		DryRun:             result.DryRun,
		ProtoOnly:          result.ProtoOnly,
		PathsRemoved:       result.PathsRemoved,
		PathsAbsent:        result.PathsAbsent,
		NeedsProtoGenerate: result.NeedsProtoGenerate,
		Message:            result.Message,
	}), nil
}

func (h *connectHandler) ValidateTemplate(ctx context.Context, req *connect.Request[lifecyclev1.ValidateTemplateRequest]) (*connect.Response[lifecyclev1.ValidateTemplateResponse], error) {
	if h.validation == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("validation runner unavailable"))
	}
	run, err := h.validation.RunValidation(ctx, validationrunner.ValidateRequest{
		TemplateID:    req.Msg.Template,
		Mode:          validationMode(req.Msg.Mode),
		TestPreset:    req.Msg.TestPreset,
		WarningPolicy: req.Msg.WarningPolicy,
		RetainTemp:    req.Msg.RetainTemp,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &lifecyclev1.ValidateTemplateResponse{
		Mode:        string(run.Mode),
		Template:    run.TemplateID,
		Count:       1,
		Status:      validationVerdict(run.Status),
		IssuesCount: int32(len(run.Findings)),
		Warnings:    retainTempWarnings(req.Msg.Mode, req.Msg.RetainTemp),
	}
	for _, finding := range run.Findings {
		resp.Issues = append(resp.Issues, &lifecyclev1.TemplateValidationIssue{Template: run.TemplateID, Path: finding.Key, Message: finding.Summary})
	}
	return connect.NewResponse(resp), nil
}

func validationMode(mode string) catalog.ValidationMode {
	return catalog.ValidationMode(mode)
}

// validationVerdict maps the persisted run status ("passed"/"failed") onto the
// stable machine verdict emitted to clients ("pass"/"fail").
func validationVerdict(status string) string {
	if status == "passed" {
		return "pass"
	}
	return "fail"
}

// retainTempWarnings reports the advisory that --retain-temp is a no-op outside
// deep validation. The deep workspace is the only artifact retain-temp keeps, so
// passing it with shallow (the default) mode silently does nothing without this.
func retainTempWarnings(mode string, retainTemp bool) []string {
	if !retainTemp {
		return nil
	}
	if catalog.ValidationMode(mode) == catalog.ModeDeep {
		return nil
	}
	return []string{"--retain-temp only applies to deep validation; it has no effect in shallow mode"}
}

func (h *connectHandler) DriftReport(ctx context.Context, req *connect.Request[lifecyclev1.DriftReportRequest]) (*connect.Response[lifecyclev1.DriftReportResponse], error) {
	engine, err := h.requireEngine()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Msg.Scenario) == "" && !req.Msg.All {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("specify a scenario slug to audit one scenario, or pass --all to audit every generated scenario"))
	}
	report, err := engine.DriftReport(ctx, templatecontracts.TemplateDriftRequest{Scenario: req.Msg.Scenario, All: req.Msg.All, Verbose: req.Msg.Verbose, JSON: true})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &lifecyclev1.DriftReportResponse{}
	for _, scenario := range report.Scenarios {
		resp.Scenarios = append(resp.Scenarios, &lifecyclev1.DriftScenario{
			Scenario:        scenario.Scenario,
			Template:        scenario.TemplateID,
			Status:          string(scenario.Status),
			ManifestDrifted: scenario.ManifestDrifted,
			ContentDrifted:  scenario.ContentDrifted,
			Message:         scenario.Message,
		})
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) CleanupRuns(ctx context.Context, req *connect.Request[lifecyclev1.CleanupRunsRequest]) (*connect.Response[lifecyclev1.CleanupRunsResponse], error) {
	engine, err := h.requireEngine()
	if err != nil {
		return nil, err
	}
	result, err := engine.CleanupRuns(ctx, templatecontracts.TemplateCleanupRequest{
		DryRun:          req.Msg.DryRun,
		OlderThan:       req.Msg.OlderThan,
		IncludeRetained: req.Msg.IncludeRetained,
		RunID:           req.Msg.RunId,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &lifecyclev1.CleanupRunsResponse{
		Matched: int32(len(result.Eligible)),
		Removed: int32(len(result.Removed)),
		DryRun:  result.DryRun,
		Message: result.Message,
		Skipped: int32(len(result.Skipped)),
	}
	for _, skipped := range result.Skipped {
		entry := &lifecyclev1.CleanupSkippedRun{Path: skipped.Path, Reason: skipped.Reason}
		if skipped.Run != nil {
			entry.RunId = skipped.Run.Marker.RunID
			if entry.Path == "" {
				entry.Path = skipped.Run.MarkerPath
			}
		}
		resp.SkippedRuns = append(resp.SkippedRuns, entry)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ListDesignKits(ctx context.Context, _ *connect.Request[lifecyclev1.ListDesignKitsRequest]) (*connect.Response[lifecyclev1.ListDesignKitsResponse], error) {
	engine, err := h.requireEngine()
	if err != nil {
		return nil, err
	}
	kits, err := engine.ListDesignKits(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &lifecyclev1.ListDesignKitsResponse{}
	for _, kit := range kits {
		resp.Kits = append(resp.Kits, designKitToProto(kit))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetDesignKit(ctx context.Context, req *connect.Request[lifecyclev1.GetDesignKitRequest]) (*connect.Response[lifecyclev1.GetDesignKitResponse], error) {
	engine, err := h.requireEngine()
	if err != nil {
		return nil, err
	}
	kit, err := engine.ShowDesignKit(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&lifecyclev1.GetDesignKitResponse{Kit: designKitToProto(kit)}), nil
}

func (h *connectHandler) ValidateDesignKits(ctx context.Context, req *connect.Request[lifecyclev1.ValidateDesignKitsRequest]) (*connect.Response[lifecyclev1.ValidateDesignKitsResponse], error) {
	engine, err := h.requireEngine()
	if err != nil {
		return nil, err
	}
	report, err := engine.ValidateDesignKits(ctx, templatecontracts.DesignValidateRequest{ID: req.Msg.Id, All: req.Msg.All})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &lifecyclev1.ValidateDesignKitsResponse{
		Count:       int32(report.Count),
		Status:      verdict(len(report.Issues) == 0),
		IssuesCount: int32(len(report.Issues)),
	}
	for _, issue := range report.Issues {
		resp.Issues = append(resp.Issues, designIssueToProto(issue))
	}
	for _, result := range report.Results {
		kitResult := &lifecyclev1.DesignKitValidationResult{Kit: result.Kit, Status: result.Status}
		for _, issue := range result.Issues {
			kitResult.Issues = append(kitResult.Issues, designIssueToProto(issue))
		}
		resp.Results = append(resp.Results, kitResult)
	}
	return connect.NewResponse(resp), nil
}

// verdict maps a boolean pass state onto the stable "pass"/"fail" verdict string
// emitted across the validation surfaces.
func verdict(passed bool) string {
	if passed {
		return "pass"
	}
	return "fail"
}

func designIssueToProto(issue templatecontracts.DesignValidationIssue) *lifecyclev1.DesignValidationIssue {
	return &lifecyclev1.DesignValidationIssue{Kit: issue.Kit, Adapter: issue.Adapter, Path: issue.Path, Message: issue.Message}
}

func (h *connectHandler) requireEngine() (*templateengine.Engine, error) {
	if h.engine == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("template engine unavailable"))
	}
	return h.engine, nil
}

func mergeValues(base, extra map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(extra))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range extra {
		if value != "" {
			out[key] = value
		}
	}
	return out
}

func generateScenarioValues(req *lifecyclev1.GenerateScenarioRequest) map[string]string {
	if req == nil {
		return map[string]string{}
	}
	return mergeValues(req.Values, map[string]string{
		"SCENARIO_ID":           req.Id,
		"SCENARIO_DISPLAY_NAME": req.DisplayName,
		"SCENARIO_DESCRIPTION":  req.Description,
	})
}

func designKitToProto(kit templatecontracts.DesignKitInfo) *lifecyclev1.DesignKit {
	adapters := make([]string, 0, len(kit.Manifest.Adapters))
	for id := range kit.Manifest.Adapters {
		adapters = append(adapters, id)
	}
	sort.Strings(adapters)
	return &lifecyclev1.DesignKit{
		Id:          kit.ID,
		Name:        kit.Manifest.Name,
		Version:     kit.Manifest.Version,
		Default:     kit.Manifest.Default,
		Description: kit.Manifest.Description,
		Tags:        kit.Manifest.Tags,
		Adapters:    adapters,
	}
}
