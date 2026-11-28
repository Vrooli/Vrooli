// Package checks composes experience-manager's validation checks over the
// parser-owned spec model.
package checks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"experience-manager/internal/spec"
)

// Check is one experience contract check.
type Check interface {
	Name() string
	Run(ctx context.Context, report spec.Report) []spec.Finding
}

// Engine runs registered checks for one scenario.
type Engine struct {
	repoRoot string
	checks   []Check
}

// New builds an Engine.
func New(repoRoot string, checks ...Check) *Engine {
	return &Engine{repoRoot: repoRoot, checks: checks}
}

// Checks exposes the registered check set for ratchet tests.
func (e *Engine) Checks() []Check { return e.checks }

// ValidateScenario resolves the target, parses its experience/ contract, and
// runs registered checks.
func (e *Engine) ValidateScenario(ctx context.Context, scenario, path string) (spec.Report, error) {
	dir, err := e.resolveTarget(scenario, path)
	if err != nil {
		return spec.Report{}, err
	}
	report, err := spec.ParseScenario(dir)
	if err != nil {
		return spec.Report{}, err
	}
	if scenario != "" {
		report.Scenario = scenario
	}
	for _, check := range e.checks {
		report.Findings = append(report.Findings, check.Run(ctx, report)...)
	}
	report.Findings = CapSeverity(report.Findings)
	return report, nil
}

func (e *Engine) resolveTarget(scenario, path string) (string, error) {
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("target path %q: %w", path, err)
		}
		return path, nil
	}
	if scenario == "" {
		return "", fmt.Errorf("scenario is required")
	}
	dir := filepath.Join(e.repoRoot, "scenarios", scenario)
	if _, err := os.Stat(dir); err != nil {
		return "", fmt.Errorf("scenario %q not found under %s: %w", scenario, filepath.Join(e.repoRoot, "scenarios"), err)
	}
	return dir, nil
}

// CapSeverity enforces the experience dimension's advisory ceiling: findings
// never exceed ERROR in the parser/check layer.
func CapSeverity(findings []spec.Finding) []spec.Finding {
	for i := range findings {
		if findings[i].Severity == "SEVERITY_BLOCKER" || findings[i].Severity == "blocker" {
			findings[i].Severity = "SEVERITY_ERROR"
		}
	}
	return findings
}
