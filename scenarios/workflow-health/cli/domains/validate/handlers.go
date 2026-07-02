package validate

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

type handlers struct {
	client scenariovalidationconnect.ScenarioValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: scenariovalidationconnect.NewScenarioValidationServiceClient(httpClient, baseURL)}
}

func (h *handlers) scenario(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario:         scenario,
		Path:             ctx.Flag("path"),
		IncludeExecution: ctx.BoolFlag("include-execution"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("validate scenario %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no validation response")
	}
	msg := resp.Msg
	assessment := msg.GetAssessment()
	results := make([]string, 0, len(assessment.GetFindings()))
	for _, finding := range assessment.GetFindings() {
		results = append(results, formatFinding(finding))
	}
	summary := []string{
		fmt.Sprintf("Validated %s: status=%s errors=%d warnings=%d infos=%d",
			msg.GetScenario(),
			statusLabel(msg.GetStatus()),
			countSeverity(assessment, "ERROR"),
			countSeverity(assessment, "WARN"),
			countSeverity(assessment, "INFO"),
		),
	}
	if local := assessment.GetLocal(); local != nil {
		line := fmt.Sprintf("Maturity: %s", local.GetCurrentLevel())
		if local.GetNextLevel() != "" {
			line += " -> " + local.GetNextLevel()
		}
		summary = append(summary, line)
	}
	if err := cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Workflow findings",
		Results:        results,
	}); err != nil {
		return err
	}
	switch msg.GetStatus() {
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED:
		return fmt.Errorf("scenario %s did not pass workflow validation", msg.GetScenario())
	case scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR:
		return fmt.Errorf("scenario %s workflow validation errored", msg.GetScenario())
	default:
		return nil
	}
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

func countSeverity(a *commonv1.MaturityAssessment, severity string) int {
	if a == nil {
		return 0
	}
	total := 0
	for key, count := range a.GetFindingsBySeverity() {
		if severityLabel(key) == severity {
			total += int(count)
		}
	}
	return total
}

func formatFinding(f *commonv1.AssessmentFinding) string {
	if f == nil {
		return ""
	}
	line := fmt.Sprintf("[%s] %s - %s", severityLabel(f.GetSeverity()), f.GetCode(), f.GetMessage())
	if f.GetLocation() != "" {
		line += " (" + f.GetLocation() + ")"
	}
	if f.GetRemediation() != "" {
		line += "\n    remediation: " + f.GetRemediation()
	}
	return line
}

func severityLabel(severity string) string {
	switch severity {
	case "SEVERITY_ERROR", "FINDING_SEVERITY_ERROR", "ERROR":
		return "ERROR"
	case "SEVERITY_WARNING", "FINDING_SEVERITY_WARNING", "WARNING", "WARN":
		return "WARN"
	case "SEVERITY_INFO", "FINDING_SEVERITY_INFO", "INFO":
		return "INFO"
	case "SEVERITY_BLOCKER", "FINDING_SEVERITY_BLOCKER", "BLOCKER":
		return "BLOCKER"
	default:
		return "UNSPECIFIED"
	}
}
