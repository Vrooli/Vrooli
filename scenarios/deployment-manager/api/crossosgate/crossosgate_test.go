package crossosgate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeBridge is an in-memory Bridge for the Gate orchestration tests.
type fakeBridge struct {
	gotRun     Request
	gateID     string
	runResults []OSResult
	runErr     error

	waitVerdict  string
	waitTimedOut bool
	waitResults  []OSResult
	waitErr      error
	gotWaitID    string
}

func (f *fakeBridge) RunGate(_ context.Context, in Request) (string, []OSResult, error) {
	f.gotRun = in
	return f.gateID, f.runResults, f.runErr
}

func (f *fakeBridge) WaitGate(_ context.Context, gateID string, _ int64) (string, bool, []OSResult, error) {
	f.gotWaitID = gateID
	return f.waitVerdict, f.waitTimedOut, f.waitResults, f.waitErr
}

// A green gate across every OS is production-ready, and deployment-manager owns
// that boolean.
func TestEvaluate_AllGreenIsProductionReady(t *testing.T) {
	fb := &fakeBridge{
		gateID:      "gate-1",
		waitVerdict: "passed",
		waitResults: []OSResult{
			{OS: "linux", Disposition: "passed"},
			{OS: "darwin", Disposition: "passed"},
			{OS: "windows", Disposition: "passed"},
		},
	}
	v, err := New(fb).Evaluate(context.Background(), Request{Scenario: "web-search", Revision: "r", TargetOSes: []string{"linux", "darwin", "windows"}})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !v.ProductionReady {
		t.Fatalf("expected ProductionReady, got %+v", v)
	}
	if v.GateID != "gate-1" || fb.gotWaitID != "gate-1" {
		t.Fatalf("gate id not threaded run→wait: %q / %q", v.GateID, fb.gotWaitID)
	}
}

// One failing OS withholds production-readiness.
func TestEvaluate_OneFailingOSNotReady(t *testing.T) {
	fb := &fakeBridge{
		gateID:      "gate-2",
		waitVerdict: "failed",
		waitResults: []OSResult{
			{OS: "linux", Disposition: "passed"},
			{OS: "windows", Disposition: "failed", ExitCode: 1, RunID: "run-win"},
		},
	}
	v, err := New(fb).Evaluate(context.Background(), Request{Scenario: "web-search", Revision: "r", TargetOSes: []string{"linux", "windows"}})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if v.ProductionReady {
		t.Fatalf("expected not production-ready, got %+v", v)
	}
	if v.Verdict != "failed" {
		t.Fatalf("verdict = %q, want failed", v.Verdict)
	}
}

// A timed-out gate is never assumed green — promotion is withheld.
func TestEvaluate_TimedOutWithholdsReadiness(t *testing.T) {
	fb := &fakeBridge{gateID: "gate-3", waitVerdict: "pending", waitTimedOut: true}
	v, err := New(fb).Evaluate(context.Background(), Request{Scenario: "s", Revision: "r", TargetOSes: []string{"linux"}})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if v.ProductionReady {
		t.Fatalf("a timed-out gate must not be production-ready, got %+v", v)
	}
	if !v.TimedOut {
		t.Fatalf("expected TimedOut")
	}
}

// The HTTP handler returns 503 when bridge is unconfigured (the additive,
// inert-by-default route).
func TestHandler_UnconfiguredReturns503(t *testing.T) {
	h := NewHTTPHandler("", "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cross-os-gate/evaluate", strings.NewReader(`{"scenario":"s","revision":"r","target_oses":["linux"]}`))
	h.Evaluate(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

// The handler validates required fields.
func TestHandler_ValidatesRequired(t *testing.T) {
	h := NewHandler(New(&fakeBridge{}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"scenario":"","revision":"r","target_oses":["linux"]}`))
	h.Evaluate(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// The httpBridge speaks bridge's Connect/JSON GateService contract end-to-end
// against a stand-in server — proving the wire shapes match.
func TestHTTPBridge_SpeaksConnectJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/RunGate"):
			var req runGateWire
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.Scenario != "web-search" || len(req.TargetOses) != 2 {
				t.Errorf("unexpected RunGate body: %+v", req)
			}
			_ = json.NewEncoder(w).Encode(runGateRespWire{GateID: "gate-9", Verdict: "pending"})
		case strings.HasSuffix(r.URL.Path, "/WaitGate"):
			var req waitGateReqWire
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.ID != "gate-9" {
				t.Errorf("WaitGate id = %q, want gate-9", req.ID)
			}
			_ = json.NewEncoder(w).Encode(waitGateRespWire{
				Gate:    gateWire{ID: "gate-9", Verdict: "passed"},
				Results: []osResultWire{{OS: "linux", Disposition: "passed"}, {OS: "darwin", Disposition: "passed"}},
			})
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	v, err := New(NewHTTPBridge(srv.URL, "tok", srv.Client())).Evaluate(context.Background(), Request{
		Scenario: "web-search", Revision: "abc", TargetOSes: []string{"linux", "darwin"},
	})
	if err != nil {
		t.Fatalf("Evaluate over HTTP: %v", err)
	}
	if !v.ProductionReady || v.GateID != "gate-9" || len(v.Results) != 2 {
		t.Fatalf("unexpected verdict: %+v", v)
	}
}

// A Connect error body on a non-200 surfaces as an error.
func TestHTTPBridge_ConnectErrorSurfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"code": "unauthenticated", "message": "owner token required"})
	}))
	defer srv.Close()

	_, err := New(NewHTTPBridge(srv.URL, "", srv.Client())).Evaluate(context.Background(), Request{
		Scenario: "s", Revision: "r", TargetOSes: []string{"linux"},
	})
	if err == nil || !strings.Contains(err.Error(), "owner token required") {
		t.Fatalf("expected unauthenticated error, got %v", err)
	}
}
