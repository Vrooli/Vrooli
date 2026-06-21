// Package lighthouse wraps the Lighthouse CLI (its own Chrome — NOT via BAS) to
// score a scenario's UI against per-page error/warn thresholds from
// .vrooli/lighthouse.json. It is Tier 0 (works on any URL) and silently SKIPS
// when the Lighthouse CLI is absent or there is no UI URL.
//
// The production Runner is CLIRunner (runner.go): it loads .vrooli/lighthouse.json,
// resolves the served UI URL, shells the Lighthouse CLI per configured page, and
// records a violation for any category below its threshold. Tests drive a fake.
package lighthouse

import (
	"context"
	"errors"
)

// Outcome describes whether a Lighthouse run scored or was cleanly skipped.
type Outcome int

const (
	OutcomeUnspecified Outcome = iota
	OutcomeScored
	OutcomeSkipped
	OutcomeFailed
)

// PageScore is one Lighthouse run against one page.
type PageScore struct {
	URL           string
	Performance   float64
	Accessibility float64
	BestPractices float64
	SEO           float64
	// Violations holds every threshold breach (error- and warn-level) as a
	// human string, for display and the proto response.
	Violations []string
	// ErrorViolations holds only the error-threshold breaches — the subset that
	// must FAIL the phase (restoring the native perf phase's Lighthouse gating).
	// Warn-level breaches stay in Violations only and do not fail.
	ErrorViolations []string
}

// Result is the outcome of a Lighthouse run.
type Result struct {
	Scenario string
	Outcome  Outcome
	Pages    []PageScore
	Reason   string
}

// Runner is the seam that invokes the Lighthouse CLI for a scenario's pages.
// The real implementation shells the CLI; tests drive a fake. An empty result
// with a reason signals a clean skip (CLI absent / no UI URL).
type Runner interface {
	Run(ctx context.Context, scenario, path string) (Result, error)
}

// Service is the Lighthouse engine.
type Service struct {
	runner Runner
}

// NewService wires a Lighthouse Service over the runner seam.
func NewService(runner Runner) *Service { return &Service{runner: runner} }

// Score runs Lighthouse for a scenario, returning a clean skip when impossible.
func (s *Service) Score(ctx context.Context, scenario, path string) (Result, error) {
	if s == nil {
		return Result{}, errors.New("lighthouse: service not wired")
	}
	if scenario == "" {
		return Result{}, errors.New("lighthouse: scenario is required")
	}
	if s.runner == nil {
		return Result{Scenario: scenario, Outcome: OutcomeSkipped, Reason: "lighthouse runner not wired"}, nil
	}
	res, err := s.runner.Run(ctx, scenario, path)
	if err != nil {
		return Result{}, err
	}
	res.Scenario = scenario
	return res, nil
}
