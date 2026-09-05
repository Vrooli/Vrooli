package capture

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
	captureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture/captureconnect"
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

// [REQ:PH-CAPTURE-005] A reachable BAS that rejects malformed interaction JSON
// is a failed capture request, not an unavailable capture mechanism.
func TestCapturePerfPreservesBASInvalidArgument(t *testing.T) {
	_, handler := captureconnect.NewCaptureServiceHandler(captureHandlerFunc(func(
		context.Context,
		*connect.Request[capturev1.CaptureRequest],
	) (*connect.Response[capturev1.CaptureResponse], error) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("interaction_flow_json is not a valid WorkflowDefinitionV2"))
	}))
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	c := &BASConnectClient{
		Resolve:    func(context.Context) (string, error) { return server.URL, nil },
		HTTPClient: server.Client(),
	}
	_, err := c.CapturePerf(context.Background(), "http://example.test", `{"edges":[{"from":"a","to":"b"}]}`)
	var reqErr BASRequestError
	if !errors.As(err, &reqErr) {
		t.Fatalf("expected BASRequestError, got %T: %v", err, err)
	}
	if reqErr.Code != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", reqErr.Code)
	}
	if errors.Is(err, ErrCaptureUnavailable) {
		t.Fatalf("BAS request errors must not be classified unavailable: %v", err)
	}
}

type captureHandlerFunc func(context.Context, *connect.Request[capturev1.CaptureRequest]) (*connect.Response[capturev1.CaptureResponse], error)

func (f captureHandlerFunc) Capture(ctx context.Context, req *connect.Request[capturev1.CaptureRequest]) (*connect.Response[capturev1.CaptureResponse], error) {
	return f(ctx, req)
}
