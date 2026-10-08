package validate

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	maturityreport "github.com/vrooli/maturity-go/report"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/measures-health/v1/validation/validation_v1connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

// validateTimeout is the HTTP client timeout for the validate calls. Coverage
// validation reads manifests + the proto descriptor and (with --probe) round-
// trips live measure endpoints; the fleet rollup grades every scenario. Both can
// exceed the chatty-RPC default, so use a generous ceiling (mirrors
// security-health's producer path).
const validateTimeout = 5 * time.Minute

type handlers struct {
	core             *cliapp.ScenarioApp
	validationClient scenariovalidationconnect.ScenarioValidationServiceClient
	fleetClient      validationconnect.ValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, validateTimeout)
	return &handlers{
		core:             core,
		validationClient: scenariovalidationconnect.NewScenarioValidationServiceClient(httpClient, baseURL),
		fleetClient:      validationconnect.NewValidationServiceClient(httpClient, baseURL),
	}
}

// validateScenario calls ScenarioValidationService.ValidateScenario, renders the
// shared assessment report, and returns a non-nil error when the shared status
// is failed or errored.
func (h *handlers) validateScenario(ctx cliapp.RunContext) error {
	name := ctx.Positional("name")
	resp, err := h.validationClient.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario:         name,
		IncludeExecution: ctx.BoolFlag("probe"),
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
		results = append(results, formatAssessmentFinding(f))
	}
	errors := severityCount(assessment.GetFindingsBySeverity(), "ERROR")
	warnings := severityCount(assessment.GetFindingsBySeverity(), "WARNING")
	infos := severityCount(assessment.GetFindingsBySeverity(), "INFO")
	summary := []string{
		fmt.Sprintf("%s — measures coverage: status=%s errors=%d warnings=%d infos=%d",
			msg.GetScenario(), statusLabel(msg.GetStatus()), errors, warnings, infos,
		),
	}
	if assessmentReport := maturityreport.BuildMaturityListReport(msg.GetAssessment()); len(assessmentReport.Summary) > 0 {
		summary = append(summary, assessmentReport.Summary...)
		if len(assessmentReport.Results) > 0 {
			results = append(results, "")
			results = append(results, assessmentReport.Results...)
		}
	}
	if err := cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Findings",
		Results:        results,
	}); err != nil {
		return err
	}
	if msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		return fmt.Errorf("scenario %s did not pass measures validation (%d error finding(s))", msg.GetScenario(), errors)
	}
	if msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_ERROR {
		return fmt.Errorf("scenario %s measures validation errored", msg.GetScenario())
	}
	return nil
}

// coverage calls ValidationService.ListFleetCoverage and renders one rollup per
// scenario.
func (h *handlers) coverage(ctx cliapp.RunContext) error {
	resp, err := h.fleetClient.ListFleetCoverage(context.Background(), connect.NewRequest(&validationv1.ListFleetCoverageRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("fleet coverage", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no coverage response")
	}
	results := make([]string, 0, len(resp.Msg.Entries))
	for _, e := range resp.Msg.Entries {
		results = append(results, formatFleetEntry(e))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Measures coverage across %d scenario(s).", len(resp.Msg.Entries))},
		ResultsHeading: "Fleet coverage",
		Results:        results,
		RetrievalHints: []string{"`measures-health validate scenario <name>` — drill into one scenario"},
	})
}

func formatDomain(d *validationv1.DomainCoverage) string {
	switch d.GetStatus() {
	case validationv1.DomainStatus_DOMAIN_STATUS_COVERED:
		return fmt.Sprintf("%-20s ✓ covered    %d measure(s)   tier: %s", d.GetDomain(), d.GetMeasureCount(), tierLabel(d.GetTier()))
	case validationv1.DomainStatus_DOMAIN_STATUS_WAIVED:
		return fmt.Sprintf("%-20s ⊘ waived     %q", d.GetDomain(), d.GetWaiverReason())
	case validationv1.DomainStatus_DOMAIN_STATUS_UNCOVERED:
		return fmt.Sprintf("%-20s ✗ uncovered  ERROR (stateful, no measure, no waiver)", d.GetDomain())
	default:
		note := d.GetNote()
		if note == "" {
			note = "not stateful"
		}
		return fmt.Sprintf("%-20s – not expected (%s)", d.GetDomain(), note)
	}
}

func formatFleetEntry(e *validationv1.FleetEntry) string {
	verdict := "PASS"
	if !e.GetPassed() {
		verdict = "FAIL"
	}
	return fmt.Sprintf("%-24s %s  expected=%d covered=%d waived=%d uncovered=%d  measures=%d  worst-tier=%s",
		e.GetScenario(), verdict, e.GetExpected(), e.GetCovered(), e.GetWaived(), e.GetUncovered(), e.GetMeasureCount(), tierLabel(e.GetWorstTier()))
}

func findingHints(findings []*validationv1.Finding) []string {
	hints := make([]string, 0, len(findings))
	for _, f := range findings {
		if f.GetSeverity() != validationv1.Severity_SEVERITY_ERROR {
			continue
		}
		hints = append(hints, fmt.Sprintf("[ERROR] %s — %s\n    fix: %s", f.GetRuleId(), f.GetTitle(), f.GetRemediation()))
	}
	return hints
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

func formatAssessmentFinding(f *commonv1.AssessmentFinding) string {
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

func tierLabel(t validationv1.Tier) string {
	switch t {
	case validationv1.Tier_TIER_FULL:
		return "full"
	case validationv1.Tier_TIER_PARTIAL:
		return "partial"
	case validationv1.Tier_TIER_FALLBACK:
		return "fallback"
	default:
		return "n/a"
	}
}
