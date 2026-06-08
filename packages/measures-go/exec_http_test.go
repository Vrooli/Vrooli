package measures

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPExecutor_RoundTrip(t *testing.T) {
	var gotReq MeasureRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, DefaultExecutePath) {
			http.Error(w, "bad path", http.StatusNotFound)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		_ = json.NewEncoder(w).Encode(MeasureResult{
			Value:      "42",
			Provenance: Provenance{ExecutedQuery: "SELECT ..."},
		})
	}))
	defer srv.Close()

	x := NewHTTPExecutor(BaseURLResolverFunc(func(_ context.Context, scenario string) (string, error) {
		if scenario != "swarm-manager" {
			t.Fatalf("unexpected scenario %q", scenario)
		}
		return srv.URL, nil
	}))

	decl := MeasureDeclaration{Name: "backlog.completed", Scenario: "swarm-manager"}
	out, err := x.Execute(context.Background(), decl, map[string]string{"window": "this_week"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Value != "42" || out.Provenance.ExecutedQuery != "SELECT ..." {
		t.Fatalf("result = %+v", out)
	}
	if gotReq.Measure != "backlog.completed" || gotReq.Params["window"] != "this_week" {
		t.Fatalf("server saw wrong request: %+v", gotReq)
	}
}

func TestHTTPExecutor_Non2xxIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "required param missing", http.StatusUnprocessableEntity)
	}))
	defer srv.Close()
	x := NewHTTPExecutor(BaseURLResolverFunc(func(_ context.Context, _ string) (string, error) {
		return srv.URL, nil
	}))
	_, err := x.Execute(context.Background(), MeasureDeclaration{Name: "m", Scenario: "s"}, nil)
	if err == nil || !strings.Contains(err.Error(), "HTTP 422") {
		t.Fatalf("expected HTTP 422 error, got %v", err)
	}
}

// End-to-end through the serve helper: the Registry.Handler and the HTTPExecutor
// are the two ends of the same contract, so they must round-trip.
func TestHTTPExecutor_AgainstServeHandler(t *testing.T) {
	reg := NewRegistry(WithClock(anchorClock()))
	decl := backlogCompleted(EffectRead)
	if err := reg.Register(decl, func(_ context.Context, req MeasureRequest) (MeasureResult, error) {
		return MeasureResult{Value: "11", Provenance: Provenance{ExecutedQuery: "count(" + req.Params["window"] + ")"}}, nil
	}); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(reg.Handler())
	defer srv.Close()

	x := NewHTTPExecutor(BaseURLResolverFunc(func(_ context.Context, _ string) (string, error) {
		return srv.URL, nil
	}))
	out, err := x.Execute(context.Background(), decl, map[string]string{"window": "this_week"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Value != "11" || out.Provenance.ExecutedQuery != "count(this_week)" {
		t.Fatalf("round-trip result = %+v", out)
	}
	if out.Provenance.ComputedAt.IsZero() {
		t.Fatal("provenance ComputedAt must be stamped by the serve helper")
	}
}
