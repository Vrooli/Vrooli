package validate

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

type handlers struct {
	client scenariovalidationconnect.ScenarioValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		client: scenariovalidationconnect.NewScenarioValidationServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) validateScenario(ctx cliapp.RunContext) error {
	name := ctx.Positional("name")
	resp, err := h.client.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: name,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("validate scenario %q", name), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no validation response")
	}
	msg := resp.Msg
	assessment := msg.GetAssessment()
	results := make([]string, 0, len(assessment.GetFindings()))
	summary := fmt.Sprintf("Validated %s - status=%s errors=%d warnings=%d infos=%d",
		msg.Scenario,
		statusLabel(msg.GetStatus()),
		severityCount(assessment, architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
		severityCount(assessment, architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING),
		severityCount(assessment, architecturev1.FindingSeverity_FINDING_SEVERITY_INFO),
	)
	summaryLines := []string{summary}
	if assessmentReport := cliapp.BuildMaturityListReport(assessment); len(assessmentReport.Summary) > 0 {
		summaryLines = append(summaryLines, assessmentReport.Summary...)
		if len(assessmentReport.Results) > 0 {
			results = append(results, assessmentReport.Results...)
		}
	}
	if err := cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summaryLines,
		ResultsHeading: "Findings",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("`describe scenario %s` - inspect the proto surface facts", name),
		},
	}); err != nil {
		return err
	}
	if msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		return fmt.Errorf("scenario %s did not pass proto validation (%d error finding(s))", msg.Scenario, failedFindingCount(assessment))
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

func severityCount(a interface{ GetFindingsBySeverity() map[string]int32 }, severity architecturev1.FindingSeverity) int {
	if a == nil {
		return 0
	}
	return int(a.GetFindingsBySeverity()[severity.String()])
}

func failedFindingCount(a interface{ GetFindingsBySeverity() map[string]int32 }) int {
	return severityCount(a, architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR) +
		severityCount(a, architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER)
}
