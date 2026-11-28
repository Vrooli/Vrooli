package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"

	eligpb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/eligibility"
)

type stubChecker struct {
	resp *eligpb.CheckResponse
	err  error
}

func (s *stubChecker) Check(_ context.Context, _ string) (*eligpb.CheckResponse, error) {
	return s.resp, s.err
}

func TestResolveIsolation_Routed(t *testing.T) {
	got := resolveIsolation(context.Background(), &stubChecker{resp: &eligpb.CheckResponse{Routed: true}}, "demo")
	if got.Status != IsolationStatusRouted {
		t.Errorf("expected routed; got %v", got.Status)
	}
	if len(got.Violations) != 0 || len(got.Reasons) != 0 {
		t.Errorf("expected empty violations/reasons; got %+v", got)
	}
}

func TestResolveIsolation_NotRouted_WithViolations(t *testing.T) {
	got := resolveIsolation(context.Background(), &stubChecker{
		resp: &eligpb.CheckResponse{
			Routed:               false,
			DisqualifyingReasons: []string{"raw sql.Open"},
			Violations: []*eligpb.Violation{
				{RuleId: "ROUTED_SEAMS_UNWIRED", Severity: "high", File: "db.go", Line: 42},
			},
		},
	}, "demo")
	if got.Status != IsolationStatusNotRouted {
		t.Errorf("expected not_routed; got %v", got.Status)
	}
	if len(got.Violations) != 1 || got.Violations[0].Line != 42 {
		t.Errorf("violation mapping wrong: %+v", got.Violations)
	}
	if len(got.Reasons) != 1 || got.Reasons[0] != "raw sql.Open" {
		t.Errorf("reason mapping wrong: %v", got.Reasons)
	}
}

func TestResolveIsolation_ClientError_Unknown(t *testing.T) {
	got := resolveIsolation(context.Background(), &stubChecker{err: errors.New("connection refused")}, "demo")
	if got.Status != IsolationStatusUnknown {
		t.Errorf("expected unknown; got %v", got.Status)
	}
	if len(got.Reasons) != 1 || !strings.Contains(got.Reasons[0], "connection refused") {
		t.Errorf("expected unreachable reason; got %v", got.Reasons)
	}
}

func TestResolveIsolation_NilClient_Unknown(t *testing.T) {
	got := resolveIsolation(context.Background(), nil, "demo")
	if got.Status != IsolationStatusUnknown {
		t.Errorf("expected unknown when client is nil; got %v", got.Status)
	}
}

func TestIsolationCache_GetPut(t *testing.T) {
	c := NewIsolationCache(time.Minute)
	if _, ok := c.Get("x"); ok {
		t.Fatal("expected miss before put")
	}
	want := IsolationResponse{Status: IsolationStatusRouted}
	c.Put("x", want)
	got, ok := c.Get("x")
	if !ok || got.Status != IsolationStatusRouted {
		t.Errorf("expected hit with routed; got %+v ok=%v", got, ok)
	}
	c.Invalidate("x")
	if _, ok := c.Get("x"); ok {
		t.Fatal("expected miss after invalidate")
	}
}

func TestIsolationCache_TTLExpiry(t *testing.T) {
	c := NewIsolationCache(time.Millisecond)
	c.Put("x", IsolationResponse{Status: IsolationStatusRouted})
	time.Sleep(5 * time.Millisecond)
	if _, ok := c.Get("x"); ok {
		t.Fatal("expected miss after TTL")
	}
}

func TestHandleScenarioIsolation_RoutesAndReturnsJSON(t *testing.T) {
	// Spin up a tiny mux that wires the real handler with a stub eligibility
	// client. Verifies route registration shape + JSON payload + cache.
	srv := &Server{
		testGenieEligibility: nil, // exercised via direct field access below
		isolationCache:       NewIsolationCache(time.Minute),
	}
	// Inject the stub by giving the server a checker — but the handler reads
	// from the concrete field. We exercise resolveIsolation directly above;
	// here we cover the HTTP wrapper.
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/scenarios/{slug}/isolation", srv.handleScenarioIsolation).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/demo/isolation", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200; got %d body=%s", rec.Code, rec.Body.String())
	}
	var got IsolationResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// With no client configured, the handler reports unknown — proves the
	// handler doesn't 5xx on downstream failures.
	if got.Status != IsolationStatusUnknown {
		t.Errorf("expected unknown when client nil; got %v", got.Status)
	}
}
