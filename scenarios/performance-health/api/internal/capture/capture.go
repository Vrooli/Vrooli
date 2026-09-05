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
	"strings"
	"time"

	"performance-health/internal/readiness"
)

// Outcome describes whether a capture ran or was cleanly skipped.
type Outcome int

const (
	OutcomeUnspecified Outcome = iota
	OutcomeCaptured
	OutcomeSkipped
	OutcomeFailed
	// OutcomeUnavailable means the capture mechanism itself was unreachable —
	// BAS down or no browser in the environment. Distinct from a clean SKIP (no
	// UI / Tier None) so a degraded headless run is not read as success.
	OutcomeUnavailable
)

// ErrCaptureUnavailable signals that the capture mechanism was unreachable (BAS
// down / no browser), as opposed to a clean skip or a hard failure. The
// orchestrator maps it to OutcomeUnavailable.
var ErrCaptureUnavailable = errors.New("capture mechanism unavailable (no browser / BAS unreachable)")

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
	// Unavailable is true when BAS was reached but explicitly reported that it
	// could not produce a trace (the perf artifact carried unavailable=true).
	// Distinct from a transport failure (ErrCaptureUnavailable).
	Unavailable bool
	// UnavailableReason is the verbatim reason BAS gave for an unavailable
	// perf artifact, used to decide between a true no-browser environment and
	// a retryable transient capture failure.
	UnavailableReason string
}

// BASClient is the seam to Browser Automation Studio's perf-capture RPC. The
// real implementation is a Connect client; tests drive a fake.
type BASClient interface {
	// CapturePerf captures a perf trace for the given URL. interactionFlowJSON
	// is a raw bas/flows-shape JSON body driving a specific interaction inside
	// the trace window (empty = default navigate+mount-settle); the orchestrator
	// resolves the --workflow slug to these bytes before calling. A nil error
	// with empty artifacts means BAS could not capture (e.g. no browser).
	CapturePerf(ctx context.Context, url, interactionFlowJSON string) (Artifacts, error)
}

// FlowResolver resolves a perf-flow slug to its raw bas/flows JSON for a
// scenario (<scenarioRoot>/bas/flows/<slug>.json). It returns a typed error
// when the slug names no readable flow, which the orchestrator surfaces as a
// FAILED audit (never a silent skip).
type FlowResolver interface {
	Resolve(scenario, slug string) ([]byte, error)
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
	flows FlowResolver

	// captureAttempts bounds how many times a single capture is attempted when
	// a reachable BAS returns no usable trace. A perf capture is occasionally a
	// transient casualty of concurrent capture load on the shared BAS, so one
	// retry usually recovers it. Default 3.
	captureAttempts int
	// captureBackoff is the pause between capture attempts. Default 2s.
	captureBackoff time.Duration
	// sleep is the backoff sleeper; overridable in tests to keep them fast.
	sleep func(time.Duration)
}

const defaultCaptureAttempts = 3

// NewService wires a capture Service over the BAS + build seams.
func NewService(bas BASClient, build BuildController) *Service {
	return &Service{
		bas:             bas,
		build:           build,
		captureAttempts: defaultCaptureAttempts,
		captureBackoff:  2 * time.Second,
		sleep:           time.Sleep,
	}
}

// WithCaptureRetry tunes the bounded retry for transient empty-trace captures:
// attempts is the total number of capture attempts (clamped to ≥1) and backoff
// is the pause between them (a zero backoff makes retries immediate, which tests
// use to stay fast). Returns the receiver for chaining.
func (s *Service) WithCaptureRetry(attempts int, backoff time.Duration) *Service {
	if attempts < 1 {
		attempts = 1
	}
	s.captureAttempts = attempts
	s.captureBackoff = backoff
	return s
}

// WithFlowResolver attaches the seam that resolves a --workflow slug to its
// bas/flows JSON. Without it, a non-empty --workflow is a FAILED audit (the
// orchestrator cannot resolve the interaction).
func (s *Service) WithFlowResolver(fr FlowResolver) *Service {
	s.flows = fr
	return s
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

	// Resolve a --workflow slug to its interaction JSON up front: a missing or
	// invalid flow is a hard FAILED audit (never a silent skip). An empty slug
	// keeps the default navigate+settle capture.
	interactionFlowJSON := ""
	if strings.TrimSpace(workflow) != "" {
		if s.flows == nil {
			return Result{Scenario: scenario, Outcome: OutcomeFailed, Tier: tier, Reason: "no flow resolver wired; --workflow cannot be resolved"}, nil
		}
		raw, ferr := s.flows.Resolve(scenario, workflow)
		if ferr != nil {
			return Result{Scenario: scenario, Outcome: OutcomeFailed, Tier: tier, Reason: ferr.Error()}, nil
		}
		interactionFlowJSON = string(raw)
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

	artifacts, err := s.captureWithRetry(ctx, uiURL, interactionFlowJSON)
	if errors.Is(err, ErrCaptureUnavailable) {
		// The capture mechanism itself was unreachable (BAS down / discovery
		// failed) across every attempt — genuinely UNAVAILABLE.
		return Result{Scenario: scenario, Outcome: OutcomeUnavailable, Tier: tier, Reason: "browser-automation-studio unreachable: " + err.Error()}, nil
	}
	if err != nil {
		return Result{Scenario: scenario, Outcome: OutcomeFailed, Tier: tier, Reason: err.Error()}, nil
	}
	if artifacts.TraceArtifact == "" {
		// BAS was reachable but produced no trace. Only call this UNAVAILABLE
		// (a true degraded environment) when BAS's own reason says the browser
		// was absent. Otherwise the browser ran but the trace was not finalized
		// — a retryable FAILED, NOT a "no browser" claim. Conflating the two is
		// exactly the trap that makes a busy-BAS hiccup read as "headless".
		if artifacts.Unavailable && isNoBrowserReason(artifacts.UnavailableReason) {
			return Result{Scenario: scenario, Outcome: OutcomeUnavailable, Tier: tier, Reason: "no browser available: " + artifacts.UnavailableReason}, nil
		}
		reason := strings.TrimSpace(artifacts.UnavailableReason)
		if reason == "" {
			reason = "no reason reported"
		}
		return Result{
			Scenario: scenario, Outcome: OutcomeFailed, Tier: tier,
			Reason: "BAS was reachable but produced no perf trace after retries (the browser ran; the trace was not finalized — often transient under concurrent capture load); reason: " + reason,
		}, nil
	}

	return Result{
		Scenario:          scenario,
		Outcome:           OutcomeCaptured,
		Tier:              reachedTier(tier, artifacts.HasComponentMarks),
		TraceArtifact:     artifacts.TraceArtifact,
		WebVitalsArtifact: artifacts.WebVitalsArtifact,
	}, nil
}

// captureWithRetry drives s.bas.CapturePerf with bounded retries. A perf
// capture on the shared BAS is occasionally a transient casualty of concurrent
// capture load (session evicted / trace not finalized), so a reachable-but-
// empty result is retried. It does NOT retry a definitive no-browser result
// (retrying won't grow a browser) and stops early if the context is done.
func (s *Service) captureWithRetry(ctx context.Context, url, interactionFlowJSON string) (Artifacts, error) {
	attempts := s.captureAttempts
	if attempts < 1 {
		attempts = 1
	}
	var (
		lastArt Artifacts
		lastErr error
	)
	for i := 0; i < attempts; i++ {
		if i > 0 {
			if s.sleep != nil {
				s.sleep(s.captureBackoff)
			}
			if ctx.Err() != nil {
				break
			}
		}
		art, err := s.bas.CapturePerf(ctx, url, interactionFlowJSON)
		lastArt, lastErr = art, err

		switch {
		case err == nil && art.TraceArtifact != "":
			return art, nil // captured — done
		case err == nil && art.Unavailable && isNoBrowserReason(art.UnavailableReason):
			return art, nil // definitive no-browser — retrying won't help
		}
		// Otherwise (unreachable BAS, or reachable-but-empty/transient) retry.
		if ctx.Err() != nil {
			break
		}
	}
	return lastArt, lastErr
}

// isNoBrowserReason reports whether a BAS unavailable-reason indicates a
// genuinely absent browser (a permanent degraded environment) rather than a
// retryable capture failure.
func isNoBrowserReason(reason string) bool {
	return strings.Contains(strings.ToLower(reason), "no browser")
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
