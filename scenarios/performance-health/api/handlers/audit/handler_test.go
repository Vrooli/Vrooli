package audit

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	"performance-health/internal/capture"
	"performance-health/internal/readiness"

	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/audit"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/readiness"
)

// fakeTierer decides the capture tier the orchestrator runs at.
type fakeTierer struct {
	tier   readiness.Tier
	err    error
	called bool
}

func (f *fakeTierer) Validate(_ context.Context, scenario, _ string) (readiness.Result, error) {
	f.called = true
	return readiness.Result{Scenario: scenario, Tier: f.tier}, f.err
}

// fakeBAS drives the capture Service's BAS seam.
type fakeBAS struct {
	arts capture.Artifacts
	err  error
}

func (f *fakeBAS) CapturePerf(_ context.Context, _, _ string) (capture.Artifacts, error) {
	return f.arts, f.err
}

// fakeBuild drives the capture Service's build-controller seam.
type fakeBuild struct {
	url string
	err error
}

func (f *fakeBuild) StartProfile(_ context.Context, _ string) (string, error) { return f.url, f.err }
func (f *fakeBuild) RestoreDefault(_ context.Context, _ string) error         { return nil }
func (f *fakeBuild) ResolveURL(_ context.Context, _ string) (string, error)   { return f.url, f.err }

// TestRunAuditCapturedMapsToProto builds the REAL capture service over fake BAS
// + build seams, drives a Tier-1 capture via a fake tierer, and asserts the
// captured outcome, tier, and artifacts map correctly to the proto response.
func TestRunAuditCapturedMapsToProto(t *testing.T) {
	tierer := &fakeTierer{tier: readiness.Tier1}
	bas := &fakeBAS{arts: capture.Artifacts{
		TraceArtifact:     "trace.json",
		WebVitalsArtifact: "vitals.json",
		HasComponentMarks: true, // ⚛ marks present => Tier 1 reached
	}}
	build := &fakeBuild{url: "http://localhost:3000/"}
	h := NewHandler(capture.NewService(bas, build), tierer, nil)

	resp, err := h.RunAudit(context.Background(), connect.NewRequest(&auditv1.RunAuditRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("RunAudit: %v", err)
	}
	if !tierer.called {
		t.Fatal("tierer seam was not exercised")
	}
	msg := resp.Msg
	if msg.GetScenario() != "demo" {
		t.Errorf("scenario = %q", msg.GetScenario())
	}
	if msg.GetOutcome() != auditv1.AuditOutcome_AUDIT_OUTCOME_CAPTURED {
		t.Errorf("outcome = %v, want CAPTURED", msg.GetOutcome())
	}
	if msg.GetTier() != readinessv1.CaptureTier_CAPTURE_TIER_1 {
		t.Errorf("tier = %v, want TIER_1", msg.GetTier())
	}
	if msg.GetTraceArtifact() != "trace.json" || msg.GetWebVitalsArtifact() != "vitals.json" {
		t.Errorf("artifacts mapped wrong: trace=%q vitals=%q", msg.GetTraceArtifact(), msg.GetWebVitalsArtifact())
	}
}

// TestRunAuditNoTraceUnavailable proves BAS returning no trace (no browser) maps
// to a loud UNAVAILABLE outcome with a reason — distinct from an N/A SKIP, never
// a hard error.
func TestRunAuditNoBrowserUnavailable(t *testing.T) {
	tierer := &fakeTierer{tier: readiness.Tier0}
	// BAS reachable but explicitly reports no browser => UNAVAILABLE.
	bas := &fakeBAS{arts: capture.Artifacts{Unavailable: true, UnavailableReason: "no browser available"}}
	build := &fakeBuild{url: "http://localhost:3000/"}
	h := NewHandler(capture.NewService(bas, build).WithCaptureRetry(3, 0), tierer, nil)

	resp, err := h.RunAudit(context.Background(), connect.NewRequest(&auditv1.RunAuditRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("RunAudit: %v", err)
	}
	if resp.Msg.GetOutcome() != auditv1.AuditOutcome_AUDIT_OUTCOME_UNAVAILABLE {
		t.Errorf("outcome = %v, want UNAVAILABLE", resp.Msg.GetOutcome())
	}
	if resp.Msg.GetReason() == "" {
		t.Error("an unavailable capture must carry a reason")
	}
}

// A reachable BAS that returns no trace WITHOUT a no-browser signal is a
// retryable FAILED — never a "no browser" UNAVAILABLE. Guards the honesty fix.
func TestRunAuditTransientNoTraceFailed(t *testing.T) {
	tierer := &fakeTierer{tier: readiness.Tier0}
	bas := &fakeBAS{arts: capture.Artifacts{Unavailable: true, UnavailableReason: "capture failed (transient under load)"}}
	build := &fakeBuild{url: "http://localhost:3000/"}
	h := NewHandler(capture.NewService(bas, build).WithCaptureRetry(2, 0), tierer, nil)

	resp, err := h.RunAudit(context.Background(), connect.NewRequest(&auditv1.RunAuditRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("RunAudit: %v", err)
	}
	if resp.Msg.GetOutcome() != auditv1.AuditOutcome_AUDIT_OUTCOME_FAILED {
		t.Errorf("outcome = %v, want FAILED (transient capture failure, not no-browser)", resp.Msg.GetOutcome())
	}
	if resp.Msg.GetReason() == "" {
		t.Error("a failed capture must carry a reason")
	}
}

// TestRunAuditDegradesOnTiererError proves a readiness failure does NOT hard-fail
// the audit: it falls back to the safe Tier-0 path and still attempts a capture.
func TestRunAuditDegradesOnTiererError(t *testing.T) {
	tierer := &fakeTierer{err: errors.New("code-facts unavailable")}
	bas := &fakeBAS{arts: capture.Artifacts{TraceArtifact: "trace.json"}}
	build := &fakeBuild{url: "http://localhost:3000/"}
	h := NewHandler(capture.NewService(bas, build), tierer, nil)

	resp, err := h.RunAudit(context.Background(), connect.NewRequest(&auditv1.RunAuditRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("RunAudit must not hard-fail on tierer error: %v", err)
	}
	if resp.Msg.GetOutcome() != auditv1.AuditOutcome_AUDIT_OUTCOME_CAPTURED {
		t.Errorf("outcome = %v, want CAPTURED via Tier-0 fallback", resp.Msg.GetOutcome())
	}
	if resp.Msg.GetTier() != readinessv1.CaptureTier_CAPTURE_TIER_0 {
		t.Errorf("tier = %v, want TIER_0 fallback", resp.Msg.GetTier())
	}
}

// TestRunAuditRequiresScenario asserts the empty-scenario guard maps to
// InvalidArgument.
func TestRunAuditRequiresScenario(t *testing.T) {
	h := NewHandler(capture.NewService(&fakeBAS{}, &fakeBuild{}), &fakeTierer{}, nil)
	_, err := h.RunAudit(context.Background(), connect.NewRequest(&auditv1.RunAuditRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want InvalidArgument, got %v (err=%v)", connect.CodeOf(err), err)
	}
}
