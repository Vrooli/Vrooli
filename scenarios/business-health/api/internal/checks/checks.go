// Package checks is business-health's finding engine: it composes the
// extraction layer with the registered contract checks and produces the
// neutral intent.Finding set the assessment layer maps into the shared
// maturity envelope.
//
// Phase discipline: this package holds the SEAMS in Phase 2 (engine +
// registration + severity policy); the domain checks themselves land in
// Phase 3 mapped one-to-one onto the finding vocabulary frozen in
// .vrooli/maturity.json.
package checks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"business-health/internal/extraction"

	intent "intent-go"
)

// Check is one contract check. Checks are pure: they read the extracted
// Contract (and the filesystem through intent-go helpers only) and emit
// neutral findings.
type Check interface {
	// Name is the stable check identifier (used in logs and metrics stages).
	Name() string
	// Run evaluates the check against one extracted contract.
	Run(ctx context.Context, c extraction.Contract) []intent.Finding
}

// Report is the engine's result for one scenario.
type Report struct {
	Scenario   string
	TargetPath string
	// Neutral findings; severity already capped by the advisory policy.
	Findings []intent.Finding
	// Non-empty when validation ran degraded (e.g. missing artifacts made
	// some checks unevaluable).
	DegradedReason string
}

// Engine runs the registered checks over a target scenario.
type Engine struct {
	extractor extraction.Extractor
	checks    []Check
	repoRoot  string
}

// New builds an Engine. repoRoot anchors scenario resolution for targets
// passed by slug.
func New(repoRoot string, extractor extraction.Extractor, checks ...Check) *Engine {
	return &Engine{extractor: extractor, checks: checks, repoRoot: repoRoot}
}

// Checks exposes the registered check set (used by tests asserting the
// vocabulary is fully implemented).
func (e *Engine) Checks() []Check { return e.checks }

// ValidateScenario resolves the target and runs every registered check.
// The scenario may be a slug (resolved under repoRoot/scenarios) or an
// explicit path may be supplied.
func (e *Engine) ValidateScenario(ctx context.Context, scenario, path string) (Report, error) {
	dir, err := e.resolveTarget(scenario, path)
	if err != nil {
		return Report{}, err
	}
	contract, err := e.extractor.Load(scenario, dir)
	if err != nil {
		return Report{}, err
	}
	rep := Report{Scenario: scenario, TargetPath: dir}
	collector := metricsFrom(ctx)
	for _, chk := range e.checks {
		stage := collector.Stage(chk.Name())
		findings := chk.Run(ctx, contract)
		stage.Gauge("findings", float64(len(findings)))
		stage.End()
		rep.Findings = append(rep.Findings, findings...)
	}
	rep.Findings = capSeverity(rep.Findings)
	if !contract.PRDPresent && !contract.RegistryPresent {
		rep.DegradedReason = "scenario has neither PRD.md nor requirements/index.json; only presence checks ran"
	}
	return rep, nil
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

// capSeverity enforces the business dimension's advisory posture: findings
// top out at ERROR, never BLOCKER (mirrors the native business phase the
// delegated provider replaces).
func capSeverity(findings []intent.Finding) []intent.Finding {
	for i := range findings {
		if findings[i].Severity == "SEVERITY_BLOCKER" || findings[i].Severity == "blocker" {
			findings[i].Severity = "SEVERITY_ERROR"
		}
	}
	return findings
}
