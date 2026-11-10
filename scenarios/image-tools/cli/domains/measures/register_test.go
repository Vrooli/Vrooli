package measures

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQueryProfilesFiltersAndDecodesOwnerMeasurements(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/measures/models" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL)
		}
		if got := r.URL.Query().Get("model_id"); got != "smolvlm-256m" {
			t.Fatalf("model_id query = %q", got)
		}
		if got := r.URL.Query().Get("operation"); got != "caption" {
			t.Fatalf("operation query = %q", got)
		}
		_, _ = io.WriteString(w, `[{"operation":"caption","model_id":"smolvlm-256m","profile":"warm","status":"measured","samples":3,"latency_p50_ms":80,"latency_p95_ms":110,"peak_memory_bytes":1000,"device":"cpu","last_evidence":"run-1","last_recorded_at":"2026-11-09T21:00:00Z"}]`)
	}))
	defer server.Close()

	profiles, err := queryProfiles(context.Background(), server.Client(), server.URL, "smolvlm-256m", "caption")
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 || profiles[0].ModelID != "smolvlm-256m" || profiles[0].LatencyP95MS != 110 || profiles[0].LastEvidence != "run-1" {
		t.Fatalf("decoded profiles = %#v", profiles)
	}
}

func TestQueryProfilesRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[`+string(make([]byte, (4<<20)+1))+`]`)
	}))
	defer server.Close()
	if _, err := queryProfiles(context.Background(), server.Client(), server.URL, "", ""); err == nil {
		t.Fatal("expected oversized response to fail")
	}
}
