package validate

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	maturityreport "github.com/vrooli/maturity-go/report"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each RunCtx
// func has typed access to the generated Connect client without
// re-resolving it.
type handlers struct {
	core   *cliapp.ScenarioApp
	client scenariovalidationconnect.ScenarioValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: scenariovalidationconnect.NewScenarioValidationServiceClient(httpClient, baseURL),
	}
}

// scenarioCall runs ScenarioValidationService.ValidateScenario (operation half of
// the proto_list primitive); scenarioReport renders it and scenarioOutcome
// derives the exit code from the response.
func (h *handlers) scenarioCall(ctx cliapp.OperationContext) (*scenariovalidationv1.ValidateScenarioResponse, error) {
	name := ctx.Positional("name")
	resp, err := h.client.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario:         name,
		IncludeExecution: ctx.FlagProvided("include-execution"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError(fmt.Sprintf("validate scenario %q", name), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no validation response")
	}
	return resp.Msg, nil
}

// scenarioReport maps the assessment to the human ListReport.
func (h *handlers) scenarioReport(_ cliapp.OperationContext, msg *scenariovalidationv1.ValidateScenarioResponse) cliapp.ListReport {
	assessment := msg.GetAssessment()
	results := make([]string, 0, len(assessment.GetFindings()))
	summaryLines := []string{
		fmt.Sprintf("Validated %s - status=%s errors=%d warnings=%d infos=%d",
			msg.GetScenario(),
			statusLabel(msg.GetStatus()),
			severityCount(assessment, "SEVERITY_ERROR"),
			severityCount(assessment, "SEVERITY_WARNING"),
			severityCount(assessment, "SEVERITY_INFO"),
		),
	}
	if assessmentReport := maturityreport.BuildMaturityListReport(assessment); len(assessmentReport.Summary) > 0 {
		summaryLines = append(summaryLines, assessmentReport.Summary...)
		if len(assessmentReport.Results) > 0 {
			results = append(results, assessmentReport.Results...)
		}
	}
	return cliapp.ListReport{
		Summary:        summaryLines,
		ResultsHeading: "Findings",
		Results:        results,
	}
}

// scenarioOutcome returns a non-nil error when the shared status is FAILED so
// shells get a non-zero exit code. It runs AFTER rendering, in both output modes,
// so the exit code is identical for human and --json — the failure signal is a
// property of the response, not of the output format.
func (h *handlers) scenarioOutcome(msg *scenariovalidationv1.ValidateScenarioResponse) error {
	if msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		return fmt.Errorf("scenario %s did not pass validation (%d error finding(s))", msg.GetScenario(), severityCount(msg.GetAssessment(), "SEVERITY_ERROR"))
	}
	return nil
}

func (h *handlers) projectCall(ctx cliapp.OperationContext) (*scenariovalidationv1.ValidateTargetResponse, error) {
	resp, err := h.client.ValidateTarget(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateTargetRequest{
		Target: &commonv1.ValidationTarget{
			Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PROJECT,
			Id:   "repo",
		},
		IncludeExecution: ctx.FlagProvided("include-execution"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("validate project", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no project validation response")
	}
	return resp.Msg, nil
}

func (h *handlers) projectReport(_ cliapp.OperationContext, msg *scenariovalidationv1.ValidateTargetResponse) cliapp.ListReport {
	assessment := msg.GetAssessment()
	results := make([]string, 0, len(assessment.GetFindings()))
	summary := []string{
		fmt.Sprintf("Validated project CLI - status=%s errors=%d warnings=%d infos=%d",
			statusLabel(msg.GetStatus()),
			severityCount(assessment, "SEVERITY_ERROR"),
			severityCount(assessment, "SEVERITY_WARNING"),
			severityCount(assessment, "SEVERITY_INFO"),
		),
	}
	if assessmentReport := maturityreport.BuildMaturityListReport(assessment); len(assessmentReport.Summary) > 0 {
		summary = append(summary, assessmentReport.Summary...)
		results = append(results, assessmentReport.Results...)
	}
	return cliapp.ListReport{Summary: summary, ResultsHeading: "Migration findings", Results: results}
}

func (h *handlers) projectOutcome(msg *scenariovalidationv1.ValidateTargetResponse) error {
	if msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		return fmt.Errorf("project CLI did not pass validation (%d error finding(s))", severityCount(msg.GetAssessment(), "SEVERITY_ERROR"))
	}
	return nil
}

func statusLabel(status scenariovalidationv1.ValidationStatus) string {
	switch status {
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED:
		return "passed"
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED:
		return "failed"
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED:
		return "degraded"
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR:
		return "error"
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_SKIPPED:
		return "skipped"
	default:
		return "unspecified"
	}
}

func severityCount(a interface{ GetFindingsBySeverity() map[string]int32 }, severity string) int {
	if a == nil {
		return 0
	}
	return int(a.GetFindingsBySeverity()[severity])
}
