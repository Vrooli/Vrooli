package capture

import (
	"context"
	"testing"
	"time"

	"performance-health/internal/readiness"
)

// basResult is one scripted CapturePerf return for seqBAS.
type basResult struct {
	art Artifacts
	err error
}

// seqBAS is a BASClient fake that returns a scripted sequence of results, one
// per CapturePerf call, so retry behaviour is testable. The last result is
// repeated if calls exceed the script.
type seqBAS struct {
	results []basResult
	calls   int
}

func (s *seqBAS) CapturePerf(context.Context, string, string) (Artifacts, error) {
	i := s.calls
	s.calls++
	if i >= len(s.results) {
		i = len(s.results) - 1
	}
	r := s.results[i]
	return r.art, r.err
}

// [REQ:PH-CAPTURE-003] Capture cleanly SKIPS (never errors) when the scenario
// has no UI surface, mirroring the Lighthouse silent-skip.
func TestOrchestrateSkipsNoUI(t *testing.T) {
	svc := NewService(fakeBAS{}, &fakeBuild{})
	res, err := svc.Orchestrate(context.Background(), "demo", "", readiness.TierNone)
	if err != nil {
		t.Fatalf("Orchestrate should not error on skip: %v", err)
	}
	if res.Outcome != OutcomeSkipped {
		t.Fatalf("expected SKIPPED, got %#v", res)
	}
}

// fastRetry disables the inter-attempt backoff so failure-path tests don't
// sleep through real retries.
func fastRetry(s *Service) *Service {
	s.sleep = func(time.Duration) {}
	return s
}

// [REQ:PH-CAPTURE-007] When BAS is reachable and explicitly reports no browser,
// the capture is UNAVAILABLE — distinct from a clean N/A skip — so a degraded
// headless run is not read as success.
func TestOrchestrateUnavailableNoBrowser(t *testing.T) {
	svc := fastRetry(NewService(
		fakeBAS{artifacts: Artifacts{Unavailable: true, UnavailableReason: "no browser available"}},
		&fakeBuild{uiURL: "http://localhost:3000"},
	))
	res, err := svc.Orchestrate(context.Background(), "demo", "", readiness.Tier0)
	if err != nil {
		t.Fatalf("Orchestrate: %v", err)
	}
	if res.Outcome != OutcomeUnavailable {
		t.Fatalf("expected UNAVAILABLE when BAS reports no browser, got %#v", res)
	}
}

// [REQ:PH-CAPTURE-007] When BAS is reachable, the browser ran, but no trace was
// finalized (a transient capture failure — no no-browser signal), the capture
// is a retryable FAILED, NOT a "no browser" UNAVAILABLE. This is the honesty
// guarantee: a busy-BAS hiccup must not read as a headless environment.
func TestOrchestrateTransientCaptureFailureIsFailedNotNoBrowser(t *testing.T) {
	svc := fastRetry(NewService(
		fakeBAS{artifacts: Artifacts{Unavailable: true, UnavailableReason: "the browser session did not finalize a performance trace this run (capture failed)"}},
		&fakeBuild{uiURL: "http://localhost:3000"},
	))
	res, err := svc.Orchestrate(context.Background(), "demo", "", readiness.Tier0)
	if err != nil {
		t.Fatalf("Orchestrate: %v", err)
	}
	if res.Outcome != OutcomeFailed {
		t.Fatalf("expected FAILED on a transient (non-no-browser) empty trace, got %#v", res)
	}
}

// [REQ:PH-CAPTURE-007] An ErrCaptureUnavailable from the BAS client (BAS
// unreachable) yields UNAVAILABLE, not FAILED and not SKIPPED.
func TestOrchestrateUnavailableBASUnreachable(t *testing.T) {
	svc := fastRetry(NewService(fakeBAS{err: ErrCaptureUnavailable}, &fakeBuild{uiURL: "http://localhost:3000"}))
	res, err := svc.Orchestrate(context.Background(), "demo", "", readiness.Tier0)
	if err != nil {
		t.Fatalf("Orchestrate: %v", err)
	}
	if res.Outcome != OutcomeUnavailable {
		t.Fatalf("expected UNAVAILABLE on ErrCaptureUnavailable, got %#v", res)
	}
}

// [REQ:PH-CAPTURE-007] A transient empty-trace on the first attempt that
// succeeds on a retry yields CAPTURED — the bounded retry recovers a capture
// that was a casualty of concurrent capture load.
func TestOrchestrateRetryRecoversTransientFailure(t *testing.T) {
	seq := &seqBAS{results: []basResult{
		{art: Artifacts{Unavailable: true, UnavailableReason: "capture failed (transient)"}},
		{art: Artifacts{TraceArtifact: "/runs/performance.json"}},
	}}
	svc := fastRetry(NewService(seq, &fakeBuild{uiURL: "http://localhost:3000"}))
	res, err := svc.Orchestrate(context.Background(), "demo", "", readiness.Tier0)
	if err != nil {
		t.Fatalf("Orchestrate: %v", err)
	}
	if res.Outcome != OutcomeCaptured {
		t.Fatalf("expected CAPTURED after a recovering retry, got %#v", res)
	}
	if seq.calls != 2 {
		t.Fatalf("expected exactly 2 capture attempts, got %d", seq.calls)
	}
}

// [REQ:PH-CAPTURE-007] A definitive no-browser result is NOT retried — retrying
// cannot grow a browser, so the capture stops after a single attempt.
func TestOrchestrateNoBrowserDoesNotRetry(t *testing.T) {
	seq := &seqBAS{results: []basResult{
		{art: Artifacts{Unavailable: true, UnavailableReason: "no browser available"}},
	}}
	svc := fastRetry(NewService(seq, &fakeBuild{uiURL: "http://localhost:3000"}))
	res, err := svc.Orchestrate(context.Background(), "demo", "", readiness.Tier0)
	if err != nil {
		t.Fatalf("Orchestrate: %v", err)
	}
	if res.Outcome != OutcomeUnavailable {
		t.Fatalf("expected UNAVAILABLE, got %#v", res)
	}
	if seq.calls != 1 {
		t.Fatalf("a definitive no-browser must not retry; got %d attempts", seq.calls)
	}
}
