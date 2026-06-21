// Package capture is the capture-orchestration conductor: it does a profile-mode
// restart of the target scenario, verifies the served bundle is instrumented,
// drives a BAS perf capture (Tier 0 always; Tier 1 ⚛ marks ride along when the
// page emits them) on a chosen interaction, then restores the default build. It
// gracefully SKIPS (never fails) when capture is impossible (no browser / no UI /
// headless), mirroring the Lighthouse silent-skip.
//
// BAS stays a dumb mechanism; tier and build-mode meaning live here.
//
// The external effects ride behind two seams: BASConnectClient (bas_client.go)
// calls Browser Automation Studio's perf capture and derives Tier 0/1 from the
// trace's ⚛ marks; CLIBuildController (build_controller.go) does the profile-mode
// restart + served-bundle verification. Tests drive fakes for both.
package capture

import (
	"context"
	"errors"

	"performance-health/internal/readiness"
)

// Outcome describes whether a capture ran or was cleanly skipped.
type Outcome int

const (
	OutcomeUnspecified Outcome = iota
	OutcomeCaptured
	OutcomeSkipped
	OutcomeFailed
)

// Result is the outcome of one capture orchestration.
type Result struct {
	Scenario          string
	Outcome           Outcome
	Tier              readiness.Tier
	TraceArtifact     string
	WebVitalsArtifact string
	Reason            string
}

// Artifacts is what a BAS perf capture returns.
type Artifacts struct {
	TraceArtifact     string
	WebVitalsArtifact string
	// HasComponentMarks is true when ⚛ marks were present (Tier 1 reached).
	HasComponentMarks bool
}

// BASClient is the seam to Browser Automation Studio's perf-capture RPC. The
// real implementation is a Connect client; tests drive a fake.
type BASClient interface {
	// CapturePerf captures a perf trace for the given URL with the given
	// interaction workflow (empty = default navigate+mount-settle). A nil error
	// with empty artifacts means BAS could not capture (e.g. no browser).
	CapturePerf(ctx context.Context, url, workflow string) (Artifacts, error)
}

// BuildController is the seam that restarts a scenario in profile build mode and
// restores the default build afterward.
type BuildController interface {
	// StartProfile restarts the scenario with VROOLI_BUILD_MODE=profile and
	// returns the served UI URL once the served bundle is verified instrumented
	// (empty when there is no UI to capture). An error means the restart failed
	// or the served bundle was NOT instrumented (so the orchestrator must not
	// mislabel a non-profile capture as Tier 1).
	StartProfile(ctx context.Context, scenario string) (uiURL string, err error)
	// RestoreDefault restarts the scenario back to the default build.
	RestoreDefault(ctx context.Context, scenario string) error
	// ResolveURL returns the scenario's already-served UI URL without any
	// restart — used for Tier-0 captures, which need no profile build. Empty
	// means there is no served UI (a clean skip).
	ResolveURL(ctx context.Context, scenario string) (uiURL string, err error)
}

// Service is the capture-orchestration engine.
type Service struct {
	bas   BASClient
	build BuildController
}

// NewService wires a capture Service over the BAS + build seams.
func NewService(bas BASClient, build BuildController) *Service {
	return &Service{bas: bas, build: build}
}

// Orchestrate runs the perf capture for one scenario at the given tier.
//
// Flow:
//   - TierNone (no UI), or unwired seams → clean SKIP (never an error).
//   - Tier 1 (React instrumented-capable): profile-mode restart → verify the
//     served bundle is instrumented → capture → restore the default build.
//     ⚛ marks ride along when present; their absence downgrades the *result* to
//     Tier 0 (still CAPTURED, never an error).
//   - Tier 0 (UI, non-React or React-divergent): NO profile build — capture the
//     already-served UI directly.
//
// A capture that yields no trace (e.g. no browser available) is a clean SKIP,
// mirroring the Lighthouse silent-skip.
func (s *Service) Orchestrate(ctx context.Context, scenario, workflow string, tier readiness.Tier) (Result, error) {
	if s == nil {
		return Result{}, errors.New("capture: service not wired")
	}
	if scenario == "" {
		return Result{}, errors.New("capture: scenario is required")
	}
	if reason, skip := skipReason(tier, s.bas, s.build); skip {
		return Result{Scenario: scenario, Outcome: OutcomeSkipped, Tier: tier, Reason: reason}, nil
	}

	uiURL, restore, result, done := s.resolveCaptureURL(ctx, scenario, tier)
	if done {
		return result, nil
	}
	if restore != nil {
		defer restore()
	}
	if uiURL == "" {
		return Result{Scenario: scenario, Outcome: OutcomeSkipped, Tier: tier, Reason: "scenario served no UI URL"}, nil
	}

	artifacts, err := s.bas.CapturePerf(ctx, uiURL, workflow)
	if err != nil {
		return Result{Scenario: scenario, Outcome: OutcomeFailed, Tier: tier, Reason: err.Error()}, nil
	}
	if artifacts.TraceArtifact == "" {
		return Result{Scenario: scenario, Outcome: OutcomeSkipped, Tier: tier, Reason: "BAS returned no trace (no browser available)"}, nil
	}

	return Result{
		Scenario:          scenario,
		Outcome:           OutcomeCaptured,
		Tier:              reachedTier(tier, artifacts.HasComponentMarks),
		TraceArtifact:     artifacts.TraceArtifact,
		WebVitalsArtifact: artifacts.WebVitalsArtifact,
	}, nil
}

// resolveCaptureURL resolves the URL to capture for the given tier. For Tier 1
// it does a profile-mode restart (returning a restore closure to run on defer);
// for Tier 0 it resolves the already-served URL with no restart. When it returns
// done=true the supplied result is terminal (e.g. a failed restart).
func (s *Service) resolveCaptureURL(ctx context.Context, scenario string, tier readiness.Tier) (uiURL string, restore func(), result Result, done bool) {
	if tier == readiness.Tier1 {
		url, err := s.build.StartProfile(ctx, scenario)
		if err != nil {
			return "", nil, Result{Scenario: scenario, Outcome: OutcomeFailed, Tier: tier, Reason: err.Error()}, true
		}
		return url, func() { _ = s.build.RestoreDefault(ctx, scenario) }, Result{}, false
	}
	// Tier 0: capture the running build directly — no profile restart needed.
	url, err := s.build.ResolveURL(ctx, scenario)
	if err != nil {
		return "", nil, Result{Scenario: scenario, Outcome: OutcomeSkipped, Tier: tier, Reason: "could not resolve UI URL: " + err.Error()}, true
	}
	return url, nil, Result{}, false
}
