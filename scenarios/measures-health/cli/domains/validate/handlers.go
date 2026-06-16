package validate

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/measures-health/v1/validation/validation_v1connect"
)

// validateTimeout is the HTTP client timeout for the validate calls. Coverage
// validation reads manifests + the proto descriptor and (with --probe) round-
// trips live measure endpoints; the fleet rollup grades every scenario. Both can
// exceed the chatty-RPC default, so use a generous ceiling (mirrors
// security-health's producer path).
const validateTimeout = 5 * time.Minute

type handlers struct {
	core   *cliapp.ScenarioApp
	client validationconnect.ValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, validateTimeout)
	return &handlers{
		core:   core,
		client: validationconnect.NewValidationServiceClient(httpClient, baseURL),
	}
}

// validateScenario calls ValidationService.ValidateScenario, renders the
// coverage report (proto JSON under --json — the exact shape test-genie's
// `measures` phase parses), and returns a non-nil error when any SEVERITY_ERROR
// finding is present so shells get a non-zero exit code.
func (h *handlers) validateScenario(ctx cliapp.RunContext) error {
	name := ctx.Positional("name")
	resp, err := h.client.ValidateScenario(context.Background(), connect.NewRequest(&validationv1.ValidateScenarioRequest{
		Scenario: name,
		// BoolFlag, not Flag: a bool flag sets a presence bit, not the string
		// "true" — Flag("probe") returns the empty default, so the old check was
		// always false and --probe silently never reached the RPC (defeating the
		// test-genie producer's behavioral probe).
		Probe: ctx.BoolFlag("probe"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("validate scenario %q", name), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no validation response")
	}
	msg := resp.Msg

	results := make([]string, 0, len(msg.Domains))
	for _, d := range msg.Domains {
		results = append(results, formatDomain(d))
	}
	summary := []string{
		fmt.Sprintf("%s — measures coverage: passed=%v errors=%d warnings=%d infos=%d",
			msg.GetScenario(), msg.GetPassed(),
			int(msg.GetSummary().GetErrors()),
			int(msg.GetSummary().GetWarnings()),
			int(msg.GetSummary().GetInfos()),
		),
	}
	if len(msg.GetSkippedScanners()) > 0 {
		summary = append(summary, fmt.Sprintf("Skipped: %v", msg.GetSkippedScanners()))
	}
	if assessmentReport := cliapp.BuildMaturityListReport(msg.GetAssessment()); len(assessmentReport.Summary) > 0 {
		summary = append(summary, assessmentReport.Summary...)
		if len(assessmentReport.Results) > 0 {
			results = append(results, "")
			results = append(results, assessmentReport.Results...)
		}
	}
	if err := cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Domains",
		Results:        results,
		RetrievalHints: findingHints(msg.Findings),
	}); err != nil {
		return err
	}
	if !msg.GetPassed() {
		return fmt.Errorf("scenario %s did not pass measures validation (%d error finding(s))", msg.GetScenario(), msg.GetSummary().GetErrors())
	}
	return nil
}

// coverage calls ValidationService.ListFleetCoverage and renders one rollup per
// scenario.
func (h *handlers) coverage(ctx cliapp.RunContext) error {
	resp, err := h.client.ListFleetCoverage(context.Background(), connect.NewRequest(&validationv1.ListFleetCoverageRequest{}))
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
