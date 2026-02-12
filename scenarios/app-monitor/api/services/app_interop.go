package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"app-monitor-api/rules"
	_ "app-monitor-api/rules/interop" // trigger rule registrations
)

// InteropComplianceReport is the compliance report for a scenario's UI,
// now backed by the rules engine.
type InteropComplianceReport struct {
	Scenario   string             `json:"scenario"`
	CheckedAt  time.Time          `json:"checked_at"`
	Results    []rules.RuleResult `json:"results"`
	PassCount  int                `json:"pass_count"`
	FailCount  int                `json:"fail_count"`
	SkipCount  int                `json:"skip_count"`
	TotalCount int                `json:"total_count"`
	Score      int                `json:"score"`
	HasUI      bool               `json:"has_ui"`
	Warnings   []string           `json:"warnings,omitempty"`
}

// InteropStandardsResponse is the quality endpoint response format.
type InteropStandardsResponse struct {
	EntityName string            `json:"entity_name"`
	Violations []rules.Violation `json:"violations"`
}

// CheckInteropCompliance runs the rules engine against a scenario's UI.
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

	root := strings.TrimSpace(app.Path)
	report := &InteropComplianceReport{
		Scenario:  scenarioName,
		CheckedAt: s.timeNow().UTC(),
		Results:   make([]rules.RuleResult, 0),
		HasUI:     true,
	}

	if root == "" {
		report.HasUI = false
		report.Warnings = append(report.Warnings, "scenario path unknown; skipping interop scan")
		return report, nil
	}

	uiDir := filepath.Join(root, "ui")
	if info, err := os.Stat(uiDir); err != nil || !info.IsDir() {
		report.HasUI = false
		report.Warnings = append(report.Warnings, "no ui/ directory found; interop checks not applicable")
		return report, nil
	}

	report.Results = rules.RunAll(root, scenarioName)
	report.TotalCount = len(report.Results)
	for _, r := range report.Results {
		switch {
		case r.Skipped:
			report.SkipCount++
		case r.Passed:
			report.PassCount++
		default:
			report.FailCount++
		}
	}

	scorable := report.TotalCount - report.SkipCount
	if scorable > 0 {
		report.Score = (report.PassCount * 100) / scorable
	} else {
		report.Score = 100
	}

	return report, nil
}

// GetInteropStandards returns interop violations in scenario-auditor quality format.
func (s *AppService) GetInteropStandards(ctx context.Context, scenarioName string) (*InteropStandardsResponse, error) {
	cleaned := strings.TrimSpace(scenarioName)
	if cleaned == "" {
		return nil, fmt.Errorf("scenario name is required")
	}

	root := s.resolveScenarioRoot(cleaned)
	if root == "" {
		return nil, fmt.Errorf("could not resolve scenario path for %q", cleaned)
	}

	uiDir := filepath.Join(root, "ui")
	if info, err := os.Stat(uiDir); err != nil || !info.IsDir() {
		return &InteropStandardsResponse{
			EntityName: cleaned,
			Violations: []rules.Violation{},
		}, nil
	}

	results := rules.RunAll(root, cleaned)
	allDefs := rules.AllDefs()
	defsByID := map[string]rules.RuleDef{}
	for _, d := range allDefs {
		defsByID[d.ID] = d
	}

	violations := make([]rules.Violation, 0)
	for _, r := range results {
		if r.Passed || r.Skipped {
			continue
		}
		if len(r.Violations) > 0 {
			violations = append(violations, r.Violations...)
		} else {
			// Create a violation from the result metadata.
			def := defsByID[r.RuleID]
			violations = append(violations, rules.Violation{
				RuleID:         r.RuleID,
				Severity:       def.Severity,
				Title:          def.Name,
				Description:    r.Message,
				Recommendation: def.Recommendation,
			})
		}
	}

	return &InteropStandardsResponse{
		EntityName: cleaned,
		Violations: violations,
	}, nil
}

// resolveScenarioRoot resolves the filesystem path for a scenario name.
func (s *AppService) resolveScenarioRoot(scenarioName string) string {
	if s.repoRoot == "" {
		return ""
	}
	candidate := filepath.Join(s.repoRoot, "scenarios", scenarioName)
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return candidate
	}
	return ""
}
