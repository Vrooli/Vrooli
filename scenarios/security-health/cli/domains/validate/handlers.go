package validate

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	maturityreport "github.com/vrooli/maturity-go/report"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each RunContext
// func has typed access to the generated Connect client without re-resolving it.
type handlers struct {
	core   *cliapp.ScenarioApp
	client scenariovalidationconnect.ScenarioValidationServiceClient
}

// validateScanTimeout is the HTTP client timeout for the validate call. Security
// scans run gosec/govulncheck/osv-scanner/gitleaks across every substrate in the
// target, which routinely exceeds the chatty-RPC default (30s) — a warm test-genie
// scan alone is ~31s, and cold vuln-DB caches push it higher. The default timeout
// silently breaks the test-genie → security-health producer path, so use a generous
// ceiling here instead.
const validateScanTimeout = 5 * time.Minute

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, validateScanTimeout)
	return &handlers{
		core:   core,
		client: scenariovalidationconnect.NewScenarioValidationServiceClient(httpClient, baseURL),
	}
}

// validateScenario calls ScenarioValidationService.ValidateScenario, renders
// assessment findings to human / JSON output, and returns a non-nil error when
// the shared status is failed or errored.
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
	for _, f := range assessment.GetFindings() {
		line := formatFinding(f)
		results = append(results, line)
	}
	errors := severityCount(assessment.GetFindingsBySeverity(), "ERROR")
	warnings := severityCount(assessment.GetFindingsBySeverity(), "WARNING")
	infos := severityCount(assessment.GetFindingsBySeverity(), "INFO")
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
		return fmt.Errorf("scenario %s did not pass security validation (%d error finding(s))", msg.GetScenario(), errors)
	}
	if msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR {
		return fmt.Errorf("scenario %s security validation errored", msg.GetScenario())
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

func formatFinding(f *commonv1.AssessmentFinding) string {
	if f == nil {
		return ""
	}
	line := fmt.Sprintf("[%s] %s — %s", severityLabel(f.GetSeverity()), f.GetCode(), f.GetTitle())
	if f.GetLocation() != "" {
		line += fmt.Sprintf(" (%s)", f.GetLocation())
	}
	if f.GetRemediation() != "" {
		line += "\n    fix: " + f.GetRemediation()
	}
	return line
}

func severityLabel(s string) string {
	switch s {
	case "SEVERITY_ERROR", "FINDING_SEVERITY_ERROR":
		return "ERROR"
	case "SEVERITY_WARNING", "FINDING_SEVERITY_WARNING":
		return "WARN"
	case "SEVERITY_INFO", "FINDING_SEVERITY_INFO":
		return "INFO"
	default:
		return "UNSPECIFIED"
	}
}

func severityCount(counts map[string]int32, severity string) int {
	if counts == nil {
		return 0
	}
	return int(counts["SEVERITY_"+severity] + counts["FINDING_SEVERITY_"+severity])
}
