package capture

import (
	"context"
	"testing"

	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
)

func perfArtifact(kind, path string) *capturev1.CaptureArtifact {
	return &capturev1.CaptureArtifact{
		Type:     capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE,
		Path:     path,
		Metadata: map[string]string{"artifact": kind},
	}
}

// [REQ:PH-CAPTURE-001] A response carrying a trace with ⚛ marks resolves to
// Artifacts with HasComponentMarks=true (Tier 1 detected from the trace).
func TestArtifactsFromResponseDetectsComponentMarks(t *testing.T) {
	c := &BASConnectClient{
		ReadTrace: func(string) ([]byte, error) { return []byte(`{"name":"⚛ List (mount)"}`), nil },
	}
	art := c.artifactsFromResponse(&capturev1.CaptureResponse{
		Artifacts: []*capturev1.CaptureArtifact{
			perfArtifact("cdp-trace", "/runs/perf.json"),
			perfArtifact("web-vitals", "/runs/perf.web-vitals.json"),
		},
	})
	if art.TraceArtifact != "/runs/perf.json" || art.WebVitalsArtifact != "/runs/perf.web-vitals.json" {
		t.Fatalf("artifact paths not mapped: %#v", art)
	}
	if !art.HasComponentMarks {
		t.Fatal("expected ⚛ marks detected (Tier 1)")
	}
}

// [REQ:PH-CAPTURE-002] A trace without ⚛ marks resolves to HasComponentMarks=false
// (Tier 0); capture never fails for lack of instrumentation.
func TestArtifactsFromResponseTier0Trace(t *testing.T) {
	c := &BASConnectClient{
		ReadTrace: func(string) ([]byte, error) { return []byte(`{"name":"FunctionCall"}`), nil },
	}
	art := c.artifactsFromResponse(&capturev1.CaptureResponse{
		Artifacts: []*capturev1.CaptureArtifact{perfArtifact("cdp-trace", "/runs/perf.json")},
	})
	if art.TraceArtifact == "" || art.HasComponentMarks {
		t.Fatalf("expected Tier-0 trace, got %#v", art)
	}
}

// [REQ:PH-CAPTURE-003] An unavailable perf artifact (no browser) resolves to
// empty Artifacts → the orchestrator skips cleanly.
func TestArtifactsFromResponseUnavailable(t *testing.T) {
	c := &BASConnectClient{}
	unavailable := &capturev1.CaptureArtifact{
		Type:     capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE,
		Path:     "/runs/performance.json",
		Metadata: map[string]string{"unavailable": "true", "reason": "no browser"},
	}
	art := c.artifactsFromResponse(&capturev1.CaptureResponse{
		Artifacts: []*capturev1.CaptureArtifact{unavailable},
	})
	if art.TraceArtifact != "" {
		t.Fatalf("unavailable artifact should not be treated as a trace: %#v", art)
	}
	if !art.Unavailable {
		t.Fatalf("expected Unavailable=true to be propagated, got %#v", art)
	}
	if art.UnavailableReason != "no browser" {
		t.Fatalf("expected the BAS reason to be propagated verbatim, got %q", art.UnavailableReason)
	}
}

// [REQ:PH-CAPTURE-003] CapturePerf requires a URL.
func TestCapturePerfRequiresURL(t *testing.T) {
	c := &BASConnectClient{}
	if _, err := c.CapturePerf(context.Background(), "", ""); err == nil {
		t.Fatal("expected error for empty url")
	}
}
