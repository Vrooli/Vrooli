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

// validateScenario calls ScenarioValidationService.ValidateScenario, renders
// assessment findings to human / JSON output, and returns a non-nil error
// when the report contains any SEVERITY_ERROR finding so shells get a non-zero
// exit code without a duplicated stderr noise line.
func (h *handlers) validateScenario(ctx cliapp.RunContext) error {
	name := ctx.Positional("name")
	// Direct CLI use runs the full report (static + runtime/render) by default;
	// --static-only restricts to the no-browser groups (no BAS, no auto-start).
	staticOnly := ctx.FlagDeclared("static-only") && ctx.BoolFlag("static-only")
	resp, err := h.client.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario:         name,
		IncludeExecution: !staticOnly,
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
	for _, f := range assessment.GetFindings() {
		line := formatFinding(f)
		results = append(results, line)
	}
	errors := countSeverity(assessment, "ERROR")
	warnings := countSeverity(assessment, "WARN")
	infos := countSeverity(assessment, "INFO")
	summaryLines := []string{
		fmt.Sprintf("Validated %s — status=%s errors=%d warnings=%d infos=%d",
			msg.GetScenario(), statusLabel(msg.GetStatus()), errors, warnings, infos,
		),
	}
	if assessmentReport := maturityreport.BuildMaturityListReport(msg.GetAssessment()); len(assessmentReport.Summary) > 0 {
		summaryLines = append(summaryLines, assessmentReport.Summary...)
		if len(assessmentReport.Results) > 0 {
			results = append(results, "")
			results = append(results, assessmentReport.Results...)
		}
	}
	if err := cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summaryLines,
		ResultsHeading: "Findings",
		Results:        results,
	}); err != nil {
		return err
	}
	if msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		return fmt.Errorf("scenario %s did not pass validation (%d error finding(s))", msg.GetScenario(), errors)
	}
	if msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR {
		return fmt.Errorf("scenario %s validation errored", msg.GetScenario())
	}
	return nil
}

func statusLabel(s scenariovalidationv1.ValidationStatus) string {
	switch s {
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
	line := fmt.Sprintf("[%s] %s — %s (%s)", severityLabel(f.GetSeverity()), f.GetCode(), f.GetMessage(), f.GetLocation())
	if f.GetRemediation() != "" {
		line += "\n    suggestion: " + f.GetRemediation()
	}
	return line
}

func severityLabel(s string) string {
	switch s {
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
