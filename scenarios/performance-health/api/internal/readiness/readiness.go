// Package readiness is performance-health's native validation engine and the
// engine behind both the native ReadinessService and the shared
// ScenarioValidationService (dual-mounted by handlers/readiness).
//
// It asks Code Facts for the target scenario's surfaces + UI framework, decides
// the reachable capture tier, detects the four Tier-1 perf-build infra pieces,
// and emits a finding per missing piece. Autofix toward Tier 1 is delegated to
// the autofix domain (shared packages/maturity-go/autofix helper).
package readiness

import (
	"context"
	"errors"
	"strings"
)

// Tier is the highest browser-perf capture tier a scenario can reach.
type Tier int

const (
	// TierNone — no UI surface; browser perf is skipped.
	TierNone Tier = iota
	// Tier0 — UI present but non-React (or React-divergent); CDP trace only.
	Tier0
	// Tier1 — React UI with perf-build infra; Tier 0 plus ⚛ component attribution.
	Tier1
)

// Facts is the narrow code-facts result readiness reasons over. It mirrors the
// fields readiness needs (surfaces + UI framework) without binding the engine to
// the code-facts proto.
type Facts struct {
	Scenario       string
	Surfaces       []string
	UIFramework    string
	RootPath       string
	DegradedReason string
}

// FactsClient is the code-facts intake seam: ask Code Facts for a scenario's
// surfaces + framework, with a filesystem fallback that records a degraded
// reason. The production implementation is CodeFactsClient (facts_client.go);
// tests drive a fake.
type FactsClient interface {
	Describe(ctx context.Context, scenario, path string) (Facts, error)
}

// Finding is one perf-build-infra readiness finding.
type Finding struct {
	Code        string
	Message     string
	Severity    string
	Autofixable bool
}

// Result is the outcome of a readiness validation.
type Result struct {
	Scenario       string
	Tier           Tier
	UIFramework    string
	Surfaces       []string
	Findings       []Finding
	DegradedReason string
	// Divergent flags a React UI that is missing some-or-all of the four
	// perf-build infra pieces: Tier 1 is reachable, but only after autofix.
	Divergent bool
}

// AutofixableCount counts the deterministically auto-fixable findings.
func (r Result) AutofixableCount() int {
	n := 0
	for _, f := range r.Findings {
		if f.Autofixable {
			n++
		}
	}
	return n
}

// Service is the readiness engine.
type Service struct {
	facts FactsClient
}

// NewService wires a readiness Service over the code-facts seam.
func NewService(facts FactsClient) *Service { return &Service{facts: facts} }

// Validate decides the reachable tier, detects the perf-build infra, and emits
// a finding per missing piece.
//
// Tier decision (React is NEVER assumed from the filesystem):
//   - no UI surface          → TierNone (browser perf skipped, no findings)
//   - UI present, non-React   → Tier0 (CDP trace only, no ⚛ attribution)
//   - React + full infra      → Tier1
//   - React + missing infra   → Tier1 (reachable) flagged Divergent, with one
//     finding per missing perf-build piece (all autofixable)
func (s *Service) Validate(ctx context.Context, scenario, path string) (Result, error) {
	if s == nil {
		return Result{}, errors.New("readiness: service not wired")
	}
	if scenario == "" && path == "" {
		return Result{}, errors.New("readiness: scenario or path is required")
	}
	res := Result{Scenario: scenario}
	if s.facts == nil {
		return Result{}, errors.New("readiness: facts client not wired")
	}
	facts, err := s.facts.Describe(ctx, scenario, path)
	if err != nil {
		return Result{}, err
	}
	if facts.Scenario != "" {
		res.Scenario = facts.Scenario
	}
	res.Surfaces = facts.Surfaces
	res.UIFramework = facts.UIFramework
	res.DegradedReason = facts.DegradedReason

	res.Tier = decideTier(facts)
	if res.Tier != Tier1 {
		// TierNone / Tier0: no perf-build infra to assess.
		return res, nil
	}

	// React UI: Tier 1 is the reachable ceiling. Detect the four perf-build
	// infra pieces; any missing piece is a finding and flags the scenario as
	// React-but-divergent (Tier 1 reachable only after autofix).
	root := strings.TrimSpace(facts.RootPath)
	if root == "" {
		root = strings.TrimSpace(path)
	}
	if root == "" {
		// Without a resolvable root we cannot inspect infra. Surface this as a
		// degraded reason rather than silently claiming Tier-1 readiness.
		res.DegradedReason = strings.TrimSpace(res.DegradedReason + " Could not resolve a filesystem root to inspect perf-build infra.")
		return res, nil
	}
	res.Findings = detectInfra(root)
	res.Divergent = len(res.Findings) > 0
	return res, nil
}

// decideTier maps facts to the reachable tier from framework + surfaces alone.
// React is NEVER assumed: only an explicit "react"/"react-vite" framework
// reaches Tier 1. Whether Tier-1 readiness is *actually achieved* (vs. divergent)
// is decided by infra detection in Validate.
func decideTier(f Facts) Tier {
	if !hasUI(f.Surfaces) {
		return TierNone
	}
	if isReactFramework(f.UIFramework) {
		return Tier1
	}
	return Tier0
}

func isReactFramework(framework string) bool {
	switch strings.ToLower(strings.TrimSpace(framework)) {
	case "react", "react-vite":
		return true
	default:
		return false
	}
}

func hasUI(surfaces []string) bool {
	for _, s := range surfaces {
		if strings.EqualFold(strings.TrimSpace(s), "ui") {
			return true
		}
	}
	return false
}
