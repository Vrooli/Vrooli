package capture

import (
	"context"
	"testing"

	"performance-health/internal/readiness"
)

type fakeBAS struct {
	artifacts Artifacts
	err       error
}

func (f fakeBAS) CapturePerf(context.Context, string, string) (Artifacts, error) {
	return f.artifacts, f.err
}

type fakeBuild struct {
	uiURL       string
	err         error
	resolveURL  string
	resolveErr  error
	startCalls  int
	resolveHits int
}

func (f *fakeBuild) StartProfile(context.Context, string) (string, error) {
	f.startCalls++
	return f.uiURL, f.err
}
func (f *fakeBuild) RestoreDefault(context.Context, string) error { return nil }
func (f *fakeBuild) ResolveURL(context.Context, string) (string, error) {
	f.resolveHits++
	if f.resolveURL == "" && f.resolveErr == nil {
		return f.uiURL, nil
	}
	return f.resolveURL, f.resolveErr
}

// [REQ:PH-CAPTURE-001] A Tier-1 scenario whose page emits ⚛ marks captures at
// Tier 1; the trace artifact is returned.
func TestOrchestrateCapturesTier1(t *testing.T) {
	svc := NewService(
		fakeBAS{artifacts: Artifacts{TraceArtifact: "perf.json", WebVitalsArtifact: "wv.json", HasComponentMarks: true}},
		&fakeBuild{uiURL: "http://localhost:3000"},
	)
	res, err := svc.Orchestrate(context.Background(), "demo", "", readiness.Tier1)
	if err != nil {
		t.Fatalf("Orchestrate: %v", err)
	}
	if res.Outcome != OutcomeCaptured || res.Tier != readiness.Tier1 || res.TraceArtifact != "perf.json" {
		t.Fatalf("unexpected result: %#v", res)
	}
}

// [REQ:PH-CAPTURE-002] A Tier-1-eligible scenario whose page emits NO ⚛ marks
// falls back to a Tier-0 capture (not an error).
func TestOrchestrateTier1FallsBackToTier0(t *testing.T) {
	svc := NewService(
		fakeBAS{artifacts: Artifacts{TraceArtifact: "perf.json", HasComponentMarks: false}},
		&fakeBuild{uiURL: "http://localhost:3000"},
	)
	res, err := svc.Orchestrate(context.Background(), "demo", "", readiness.Tier1)
	if err != nil {
		t.Fatalf("Orchestrate: %v", err)
	}
	if res.Outcome != OutcomeCaptured || res.Tier != readiness.Tier0 {
		t.Fatalf("expected Tier-0 fallback capture, got %#v", res)
	}
}

// [REQ:PH-CAPTURE-004] A Tier-0-only scenario captures WITHOUT a profile-mode
// restart: StartProfile is never called; the running build is captured directly.
func TestOrchestrateTier0SkipsProfileBuild(t *testing.T) {
	build := &fakeBuild{resolveURL: "http://localhost:3000"}
	svc := NewService(
		fakeBAS{artifacts: Artifacts{TraceArtifact: "perf.json"}},
		build,
	)
	res, err := svc.Orchestrate(context.Background(), "demo", "", readiness.Tier0)
	if err != nil {
		t.Fatalf("Orchestrate: %v", err)
	}
	if res.Outcome != OutcomeCaptured || res.Tier != readiness.Tier0 {
		t.Fatalf("expected Tier-0 capture, got %#v", res)
	}
	if build.startCalls != 0 {
		t.Fatalf("Tier-0 must not profile-restart, startCalls=%d", build.startCalls)
	}
	if build.resolveHits == 0 {
		t.Fatalf("Tier-0 should resolve the running URL")
	}
}

// [REQ:PH-CAPTURE-003] A failed profile-mode restart (e.g. bundle not
// instrumented) returns FAILED, not a panic — and is not mislabeled as captured.
func TestOrchestrateTier1RestartFailure(t *testing.T) {
	svc := NewService(
		fakeBAS{artifacts: Artifacts{TraceArtifact: "perf.json", HasComponentMarks: true}},
		&fakeBuild{err: errBundleNotInstrumented},
	)
	res, _ := svc.Orchestrate(context.Background(), "demo", "", readiness.Tier1)
	if res.Outcome != OutcomeFailed {
		t.Fatalf("expected FAILED on restart failure, got %#v", res)
	}
}

var errBundleNotInstrumented = errStub("served bundle not instrumented")

type errStub string

func (e errStub) Error() string { return string(e) }
