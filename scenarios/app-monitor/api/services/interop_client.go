package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

// interopProviderScenario is the scenario that owns all static UI-interop
// validation. app-monitor used to compute interop compliance locally; the rules
// engine moved to ui-health, which is now the single authority. app-monitor
// consumes it API-to-API over Connect (resolved via discovery) and surfaces the
// result in its diagnostics aggregator.
const interopProviderScenario = "ui-health"

// InteropComplianceReport is the diagnostics-facing interop result. ui-health is
// now the authority, reached over ScenarioValidationService.ValidateScenario,
// which surfaces only *failing* findings — so this report carries the interop
// failures and their count, not a pass/total ratio or compliance percentage.
// Those derived fields were dropped deliberately: they are not reconstructable
// from a failures-only contract (we never see the passing rules), and nothing
// consumed them — the aggregator folds Results into warnings and FailCount into
// severity. Results/Violations stay decoupled local types so no proto leaks into
// the diagnostics JSON.
type InteropComplianceReport struct {
	Scenario  string          `json:"scenario"`
	Results   []InteropResult `json:"results"`
	FailCount int             `json:"fail_count"`
	HasUI     bool            `json:"has_ui"`
	Source    string          `json:"source,omitempty"`
	Warnings  []string        `json:"warnings,omitempty"`
}

// InteropResult is one failing interop rule, mirroring the previous engine shape
// the aggregator iterates (Passed/Skipped/Violations/Message).
type InteropResult struct {
	RuleID     string             `json:"rule_id"`
	Passed     bool               `json:"passed"`
	Skipped    bool               `json:"skipped,omitempty"`
	Message    string             `json:"message"`
	Violations []InteropViolation `json:"violations,omitempty"`
}

// InteropViolation is a single interop violation in the diagnostics vocabulary.
type InteropViolation struct {
	RuleID         string `json:"rule_id"`
	Severity       string `json:"severity"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	FilePath       string `json:"file_path,omitempty"`
	Line           int    `json:"line,omitempty"`
	Recommendation string `json:"recommendation,omitempty"`
}

// CheckInteropCompliance asks ui-health to validate a scenario's UI interop and
// reshapes the failing interop findings into the diagnostics report. It degrades
// gracefully: if ui-health is unreachable the report is returned with a warning
// and HasUI derived locally, never an error — interop is an informational,
// non-blocking diagnostic and must not fail the whole diagnostics aggregation.
func (s *AppService) CheckInteropCompliance(ctx context.Context, appID string) (*InteropComplianceReport, error) {
	id := strings.TrimSpace(appID)
	if id == "" {
		return nil, ErrAppIdentifierRequired
	}

	app, err := s.GetApp(ctx, id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return nil, fmt.Errorf("%w: %v", ErrAppNotFound, err)
		}
		return nil, err
	}

	scenarioName := strings.TrimSpace(app.ScenarioName)
	if scenarioName == "" {
		scenarioName = strings.TrimSpace(app.Name)
	}
	if scenarioName == "" {
		scenarioName = id
	}

	report := &InteropComplianceReport{
		Scenario: scenarioName,
		Results:  make([]InteropResult, 0),
		HasUI:    localHasUI(strings.TrimSpace(app.Path)),
		Source:   interopProviderScenario,
	}

	if s.httpClient == nil || s.scenarioURL == nil {
		report.Warnings = append(report.Warnings, "interop checks unavailable: ui-health client not configured")
		return report, nil
	}

	baseURL, err := s.scenarioURL(ctx, interopProviderScenario)
	if err != nil || strings.TrimSpace(baseURL) == "" {
		report.Warnings = append(report.Warnings, fmt.Sprintf("interop checks unavailable: cannot resolve %s (%v)", interopProviderScenario, err))
		return report, nil
	}

	client := scenariovalidationconnect.NewScenarioValidationServiceClient(s.httpClient, strings.TrimRight(baseURL, "/"))
	resp, err := client.ValidateScenario(ctx, connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: scenarioName}))
	if err != nil {
		report.Warnings = append(report.Warnings, fmt.Sprintf("interop checks unavailable: %s validation failed (%v)", interopProviderScenario, err))
		return report, nil
	}

	assessment := resp.Msg.GetAssessment()
	if assessment == nil {
		report.Warnings = append(report.Warnings, "interop checks unavailable: ui-health returned no assessment")
		return report, nil
	}

	for _, f := range assessment.GetFindings() {
		code := f.GetCode()
		if code == "no_ui_surface" {
			report.HasUI = false
			continue
		}
		if !strings.HasPrefix(code, "interop_") {
			continue
		}
		report.Results = append(report.Results, InteropResult{
			RuleID:  code,
			Passed:  false,
			Message: f.GetMessage(),
			Violations: []InteropViolation{{
				RuleID:         code,
				Severity:       interopSeverityToken(f.GetSeverity()),
				Title:          f.GetTitle(),
				Description:    f.GetMessage(),
				FilePath:       f.GetLocation(),
				Recommendation: f.GetRemediation(),
			}},
		})
	}

	report.FailCount = len(report.Results)
	return report, nil
}

// interopSeverityToken maps ui-health's SEVERITY_* tokens to the diagnostics
// severity vocabulary the aggregator understands (it escalates "critical" to an
// error-level warning; everything else stays informational).
func interopSeverityToken(sev string) string {
	switch strings.ToUpper(strings.TrimSpace(sev)) {
	case "SEVERITY_ERROR":
		return "critical"
	case "SEVERITY_WARNING":
		return "medium"
	default:
		return "low"
	}
}

// localHasUI is a cheap local fallback for the HasUI flag when ui-health is
// unreachable: a scenario has a UI if it has a ui/ directory on disk.
func localHasUI(scenarioPath string) bool {
	if scenarioPath == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(scenarioPath, "ui"))
	return err == nil && info.IsDir()
}
