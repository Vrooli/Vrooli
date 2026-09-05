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

// --- Phase 2: --workflow slug resolution + close the CapturePerf no-op -------

// recordingBAS records the interactionFlowJSON it was handed so a test can
// assert the orchestrator resolved the slug to the flow file's bytes.
type recordingBAS struct {
	artifacts    Artifacts
	lastFlowJSON string
}

func (r *recordingBAS) CapturePerf(_ context.Context, _ string, interactionFlowJSON string) (Artifacts, error) {
	r.lastFlowJSON = interactionFlowJSON
	return r.artifacts, nil
}

// fakeFlows is an in-memory FlowResolver keyed by "scenario/slug".
type fakeFlows struct {
	bySlug map[string][]byte
	err    error
}

func (f fakeFlows) Resolve(scenario, slug string) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	b, ok := f.bySlug[scenario+"/"+slug]
	if !ok {
		return nil, errStub("no such flow " + scenario + "/" + slug)
	}
	return b, nil
}

// [REQ:PH-CAPTURE-005] A --workflow slug is resolved to its bas/flows bytes and
// handed to BAS on the capture request (the silent no-op is gone).
func TestOrchestrateResolvesWorkflowSlugToFlowBytes(t *testing.T) {
	bas := &recordingBAS{artifacts: Artifacts{TraceArtifact: "perf.json"}}
	flowJSON := []byte(`{"metadata":{"name":"scroll-list"},"nodes":[{"id":"s","action":{"type":"ACTION_TYPE_SCROLL","scroll":{"selector":"[data-testid='list']","delta_y":2000}}}]}`)
	svc := NewService(bas, &fakeBuild{uiURL: "http://localhost:3000"}).
		WithFlowResolver(fakeFlows{bySlug: map[string][]byte{"demo/scroll-list": flowJSON}})

	res, err := svc.Orchestrate(context.Background(), "demo", "scroll-list", readiness.Tier1)
	if err != nil {
		t.Fatalf("Orchestrate: %v", err)
	}
	if res.Outcome != OutcomeCaptured {
		t.Fatalf("expected CAPTURED, got %#v", res)
	}
	if bas.lastFlowJSON != string(flowJSON) {
		t.Fatalf("BAS did not receive the resolved flow bytes:\n got: %s", bas.lastFlowJSON)
	}
}

// [REQ:PH-CAPTURE-005] An empty --workflow keeps the default navigate+settle
// capture: BAS receives no interaction JSON.
func TestOrchestrateEmptyWorkflowSendsNoInteraction(t *testing.T) {
	bas := &recordingBAS{artifacts: Artifacts{TraceArtifact: "perf.json"}}
	svc := NewService(bas, &fakeBuild{uiURL: "http://localhost:3000"}).
		WithFlowResolver(fakeFlows{})
	if _, err := svc.Orchestrate(context.Background(), "demo", "", readiness.Tier0); err != nil {
		t.Fatalf("Orchestrate: %v", err)
	}
	if bas.lastFlowJSON != "" {
		t.Fatalf("empty slug must send no interaction, got %q", bas.lastFlowJSON)
	}
}

// [REQ:PH-CAPTURE-005] An unresolvable --workflow slug is a FAILED audit with a
// reason — never a silent skip and never a capture.
func TestOrchestrateBadWorkflowSlugFails(t *testing.T) {
	bas := &recordingBAS{artifacts: Artifacts{TraceArtifact: "perf.json"}}
	svc := NewService(bas, &fakeBuild{uiURL: "http://localhost:3000"}).
		WithFlowResolver(fakeFlows{err: errStub("read perf flow: no such file")})
	res, _ := svc.Orchestrate(context.Background(), "demo", "missing", readiness.Tier1)
	if res.Outcome != OutcomeFailed {
		t.Fatalf("expected FAILED, got %#v", res)
	}
	if res.Reason == "" {
		t.Fatalf("FAILED must carry a reason")
	}
	if bas.lastFlowJSON != "" {
		t.Fatalf("BAS must not be called when slug resolution fails")
	}
}

// A --workflow with no resolver wired is a FAILED audit, not a panic.
func TestOrchestrateWorkflowWithoutResolverFails(t *testing.T) {
	svc := NewService(&recordingBAS{}, &fakeBuild{uiURL: "http://localhost:3000"})
	res, _ := svc.Orchestrate(context.Background(), "demo", "scroll-list", readiness.Tier1)
	if res.Outcome != OutcomeFailed {
		t.Fatalf("expected FAILED without a resolver, got %#v", res)
	}
}
