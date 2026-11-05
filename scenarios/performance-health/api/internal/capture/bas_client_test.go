package capture

import (
	"context"
	"errors"
	"net/http/httptest"
	"performance-health/internal/readiness"
	"strings"
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

func TestComponentMarksDecodeJSONWithoutCountingSchedulerTracks(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want bool
	}{
		{`{"traceEvents":[{"name":"\u269b App (mount)"}]}`, true},
		{`[{"name":"\u269b App (update)"}]`, true},
		{`{"traceEvents":[{"name":"TimeStamp","args":{"data":{"trackGroup":"Scheduler \u269b"}}}]}`, false},
		{`broken JSON with ⚛ App`, false},
	} {
		c := &BASConnectClient{ReadTrace: func(string) ([]byte, error) { return []byte(tc.raw), nil }}
		if got := c.traceHasComponentMarks("trace"); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.raw, got, tc.want)
		}
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

// A reachable capture operation failure must retain its code and explanation;
// it must not claim the browser or BAS service was unreachable.
func TestCapturePerfPreservesOperationFailureThroughAudit(t *testing.T) {
	for _, code := range []connect.Code{connect.CodeInternal, connect.CodeDeadlineExceeded,
		connect.CodePermissionDenied, connect.CodeUnauthenticated, connect.CodeCanceled,
		connect.CodeUnknown, connect.CodeUnavailable} {
		t.Run(code.String(), func(t *testing.T) {
			const detail = "instruction click-plan: target element is not visible"
			_, handler := captureconnect.NewCaptureServiceHandler(captureHandlerFunc(func(context.Context, *connect.Request[capturev1.CaptureRequest]) (*connect.Response[capturev1.CaptureResponse], error) {
				return nil, connect.NewError(code, errors.New(detail))
			}))
			server := httptest.NewServer(handler)
			t.Cleanup(server.Close)
			client := &BASConnectClient{Resolve: func(context.Context) (string, error) { return server.URL, nil }, HTTPClient: server.Client()}
			build := &fakeBuild{uiURL: "http://example.test"}
			result, err := NewService(client, build).WithCaptureRetry(1, 0).Orchestrate(context.Background(), "demo", "", readiness.Tier1)
			if err != nil {
				t.Fatal(err)
			}
			want := OutcomeFailed
			if code == connect.CodeUnavailable {
				want = OutcomeUnavailable
			}
			if result.Outcome != want {
				t.Errorf("outcome=%v, want %v for %s", result.Outcome, want, code)
			}
			if !strings.Contains(result.Reason, detail) || !strings.Contains(result.Reason, code.String()) {
				t.Errorf("capture cause lost: %q", result.Reason)
			}
			if strings.Contains(result.Reason, "unreachable") || strings.Contains(result.Reason, "no browser") {
				t.Errorf("unproved mechanism claim: %q", result.Reason)
			}
			if build.restoreCalls != 1 {
				t.Errorf("default restore calls=%d, want1", build.restoreCalls)
			}
		})
	}
}

func TestCapturePerfPreservesDiscoveryFailureThroughAudit(t *testing.T) {
	const detail = "scenario discovery registry is unavailable"
	client := &BASConnectClient{Resolve: func(context.Context) (string, error) { return "", errors.New(detail) }}
	result, err := NewService(client, &fakeBuild{uiURL: "http://example.test"}).WithCaptureRetry(1, 0).Orchestrate(context.Background(), "demo", "", readiness.Tier0)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != OutcomeUnavailable || !strings.Contains(result.Reason, detail) {
		t.Fatalf("discovery cause lost: %#v", result)
	}
}
