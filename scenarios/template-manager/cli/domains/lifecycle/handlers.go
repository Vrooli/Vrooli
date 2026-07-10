package lifecycle

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	lifecyclev1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/lifecycle"
	lifecycleconnect "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/lifecycle/lifecycle_v1connect"
)

type handlers struct {
	lifecycle           lifecycleconnect.TemplateLifecycleServiceClient
	validationLifecycle lifecycleconnect.TemplateLifecycleServiceClient
	design              lifecycleconnect.DesignKitServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	validationHTTPClient, validationBaseURL := cliapp.NewConnectHTTPClientWithTimeout(core, 0)
	return &handlers{
		lifecycle:           lifecycleconnect.NewTemplateLifecycleServiceClient(httpClient, baseURL),
		validationLifecycle: lifecycleconnect.NewTemplateLifecycleServiceClient(validationHTTPClient, validationBaseURL),
		design:              lifecycleconnect.NewDesignKitServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) generateCall(ctx cliapp.OperationContext) (*lifecyclev1.GenerateScenarioResponse, error) {
	values, err := parseVars(ctx.FlagValues("var"))
	if err != nil {
		return nil, err
	}
	resp, err := h.lifecycle.GenerateScenario(context.Background(), connect.NewRequest(&lifecyclev1.GenerateScenarioRequest{
		Template:    ctx.Positional("template"),
		Id:          ctx.Flag("id"),
		DisplayName: ctx.Flag("display-name"),
		Description: ctx.Flag("description"),
		Destination: ctx.Flag("dest"),
		Design:      ctx.Flag("design"),
		Force:       ctx.BoolFlag("force"),
		DryRun:      ctx.BoolFlag("dry-run"),
		RunHooks:    ctx.BoolFlag("run-hooks"),
		Values:      values,
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("generate scenario", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) generateReport(_ cliapp.OperationContext, msg *lifecyclev1.GenerateScenarioResponse) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Generated %s at %s.", msg.DisplayName, msg.Destination)},
		Changes: []string{fmt.Sprintf("template=%s version=%s design=%s dry_run=%t hooks=%t", msg.Template, msg.ManifestVersion, msg.DesignKit, msg.DryRun, msg.RunHooks)},
	}
}

func (h *handlers) orientCall(ctx cliapp.OperationContext) (*lifecyclev1.OrientScenarioResponse, error) {
	resp, err := h.lifecycle.OrientScenario(context.Background(), connect.NewRequest(&lifecyclev1.OrientScenarioRequest{
		Scenario: ctx.Positional("scenario"),
		Finalize: ctx.BoolFlag("finalize"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("orient scenario", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) orientReport(_ cliapp.OperationContext, msg *lifecyclev1.OrientScenarioResponse) cliapp.ListReport {
	result := fmt.Sprintf("%s %d/%d complete", msg.Scenario, msg.Completed, msg.Required)
	if msg.NextStep != "" {
		result += " next=" + msg.NextStep
	}
	if msg.FinalizeRequired {
		result += " finalize_required=true"
	}
	return cliapp.ListReport{Summary: []string{result}, ResultsHeading: "Orientation", Results: []string{msg.Message}}
}

func (h *handlers) detemplateCall(ctx cliapp.OperationContext) (*lifecyclev1.DetemplateScenarioResponse, error) {
	resp, err := h.lifecycle.DetemplateScenario(context.Background(), connect.NewRequest(&lifecyclev1.DetemplateScenarioRequest{
		Scenario: ctx.Positional("scenario"),
		DryRun:   ctx.BoolFlag("dry-run"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("detemplate scenario", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) detemplateReport(_ cliapp.OperationContext, msg *lifecyclev1.DetemplateScenarioResponse) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Detemplated %s.", msg.Scenario)},
		Changes: []string{fmt.Sprintf("blocks=%d lines=%d deleted=%d dry_run=%t", msg.BlocksRemoved, msg.LinesStripped, len(msg.PathsDeleted), msg.DryRun)},
	}
}

func (h *handlers) validateCall(ctx cliapp.OperationContext) (*lifecyclev1.ValidateTemplateResponse, error) {
	resp, err := h.validationLifecycle.ValidateTemplate(context.Background(), connect.NewRequest(&lifecyclev1.ValidateTemplateRequest{
		Template:      ctx.Flag("template"),
		Mode:          ctx.Flag("mode"),
		TestPreset:    ctx.Flag("test-preset"),
		WarningPolicy: ctx.Flag("warning-policy"),
		RetainTemp:    ctx.BoolFlag("retain-temp"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("validate template", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) validateReport(_ cliapp.OperationContext, msg *lifecyclev1.ValidateTemplateResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Issues))
	for _, issue := range msg.Issues {
		results = append(results, fmt.Sprintf("%s %s: %s", issue.Template, issue.Path, issue.Message))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Validated %d template(s); issues=%d.", msg.Count, len(msg.Issues))}, ResultsHeading: "Issues", Results: results}
}

func (h *handlers) driftCall(ctx cliapp.OperationContext) (*lifecyclev1.DriftReportResponse, error) {
	resp, err := h.lifecycle.DriftReport(context.Background(), connect.NewRequest(&lifecyclev1.DriftReportRequest{
		Scenario: ctx.Positional("scenario"),
		All:      ctx.BoolFlag("all"),
		Verbose:  ctx.BoolFlag("verbose"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("template drift", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) driftReport(_ cliapp.OperationContext, msg *lifecyclev1.DriftReportResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Scenarios))
	for _, scenario := range msg.Scenarios {
		results = append(results, fmt.Sprintf("%s template=%s status=%s manifest=%t content=%t", scenario.Scenario, scenario.Template, scenario.Status, scenario.ManifestDrifted, scenario.ContentDrifted))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d scenario drift report(s).", len(msg.Scenarios))}, ResultsHeading: "Drift", Results: results}
}

func (h *handlers) cleanupCall(ctx cliapp.OperationContext) (*lifecyclev1.CleanupRunsResponse, error) {
	resp, err := h.lifecycle.CleanupRuns(context.Background(), connect.NewRequest(&lifecyclev1.CleanupRunsRequest{
		DryRun:          ctx.BoolFlag("dry-run"),
		OlderThan:       ctx.Flag("older-than"),
		IncludeRetained: ctx.BoolFlag("include-retained"),
		RunId:           ctx.Flag("run"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("cleanup template runs", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) cleanupReport(_ cliapp.OperationContext, msg *lifecyclev1.CleanupRunsResponse) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Matched %d run(s); removed %d.", msg.Matched, msg.Removed)},
		Changes: []string{msg.Message},
	}
}

func (h *handlers) designListCall(_ cliapp.OperationContext) (*lifecyclev1.ListDesignKitsResponse, error) {
	resp, err := h.design.ListDesignKits(context.Background(), connect.NewRequest(&lifecyclev1.ListDesignKitsRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list design kits", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) designListReport(_ cliapp.OperationContext, msg *lifecyclev1.ListDesignKitsResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Kits))
	for _, kit := range msg.Kits {
		results = append(results, formatKit(kit))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d design kit(s).", len(msg.Kits))}, ResultsHeading: "Design kits", Results: results}
}

func (h *handlers) designShowCall(ctx cliapp.OperationContext) (*lifecyclev1.GetDesignKitResponse, error) {
	resp, err := h.design.GetDesignKit(context.Background(), connect.NewRequest(&lifecyclev1.GetDesignKitRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("show design kit", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) designShowReport(_ cliapp.OperationContext, msg *lifecyclev1.GetDesignKitResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{"Fetched design kit."}, ResultsHeading: "Design kit", Results: []string{formatKit(msg.Kit)}}
}

func (h *handlers) designValidateCall(ctx cliapp.OperationContext) (*lifecyclev1.ValidateDesignKitsResponse, error) {
	resp, err := h.design.ValidateDesignKits(context.Background(), connect.NewRequest(&lifecyclev1.ValidateDesignKitsRequest{Id: ctx.Flag("id"), All: ctx.BoolFlag("all")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("validate design kits", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) designValidateReport(_ cliapp.OperationContext, msg *lifecyclev1.ValidateDesignKitsResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Issues))
	for _, issue := range msg.Issues {
		results = append(results, fmt.Sprintf("%s/%s %s: %s", issue.Kit, issue.Adapter, issue.Path, issue.Message))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Validated %d design kit(s); issues=%d.", msg.Count, len(msg.Issues))}, ResultsHeading: "Issues", Results: results}
}

func (h *handlers) designValidateOutcome(resp *lifecyclev1.ValidateDesignKitsResponse) error {
	if len(resp.Issues) > 0 {
		return fmt.Errorf("design validation failed")
	}
	return nil
}

func formatKit(kit *lifecyclev1.DesignKit) string {
	if kit == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s name=%q version=%s default=%t adapters=%d", kit.Id, kit.Name, kit.Version, kit.Default, len(kit.Adapters))
}
