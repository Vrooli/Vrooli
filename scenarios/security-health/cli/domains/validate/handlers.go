package validate

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/validation/validation_v1connect"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each RunContext
// func has typed access to the generated Connect client without re-resolving it.
type handlers struct {
	core   *cliapp.ScenarioApp
	client validationconnect.ValidationServiceClient
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
		client: validationconnect.NewValidationServiceClient(httpClient, baseURL),
	}
}

// validateScenario calls ValidationService.ValidateScenario, renders the
// findings (proto JSON under --json, which is the exact shape test-genie's
// `security` phase parses), and returns a non-nil error when any
// SEVERITY_ERROR finding is present so shells get a non-zero exit code.
func (h *handlers) validateScenario(ctx cliapp.RunContext) error {
	name := ctx.Positional("name")
	resp, err := h.client.ValidateScenario(context.Background(), connect.NewRequest(&validationv1.ValidateScenarioRequest{
		Scenario: name,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("validate scenario %q", name), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no validation response")
	}
	msg := resp.Msg
	results := make([]string, 0, len(msg.Findings))
	for _, f := range msg.Findings {
		line := fmt.Sprintf("[%s] %s — %s", severityLabel(f.Severity), f.GetRuleId(), f.GetTitle())
		if f.GetFilePath() != "" {
			line += fmt.Sprintf(" (%s)", f.GetFilePath())
		}
		if f.GetRemediation() != "" {
			line += "\n    fix: " + f.GetRemediation()
		}
		results = append(results, line)
	}
	summaryLines := []string{
		fmt.Sprintf("Validated %s — passed=%v errors=%d warnings=%d infos=%d",
			msg.GetScenario(), msg.GetPassed(),
			int(msg.GetSummary().GetErrors()),
			int(msg.GetSummary().GetWarnings()),
			int(msg.GetSummary().GetInfos()),
		),
	}
	if len(msg.GetSkippedScanners()) > 0 {
		summaryLines = append(summaryLines, fmt.Sprintf("Skipped (not installed): %v", msg.GetSkippedScanners()))
	}
	if assessmentReport := cliapp.BuildMaturityListReport(msg.GetAssessment()); len(assessmentReport.Summary) > 0 {
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
	if !msg.GetPassed() {
		return fmt.Errorf("scenario %s did not pass security validation (%d error finding(s))", msg.GetScenario(), msg.GetSummary().GetErrors())
	}
	return nil
}

func severityLabel(s validationv1.Severity) string {
	switch s {
	case validationv1.Severity_SEVERITY_ERROR:
		return "ERROR"
	case validationv1.Severity_SEVERITY_WARNING:
		return "WARN"
	case validationv1.Severity_SEVERITY_INFO:
		return "INFO"
	default:
		return "UNSPECIFIED"
	}
}
