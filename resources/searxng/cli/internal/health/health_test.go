package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newFakeSearxng(t *testing.T, searchBody, errorsBody string, searchStatus int) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("format") != "json" {
			t.Errorf("expected format=json, got %q", r.URL.Query().Get("format"))
		}
		w.WriteHeader(searchStatus)
		_, _ = w.Write([]byte(searchBody))
	})
	mux.HandleFunc("/stats/errors", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(errorsBody))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func TestProbeHealthyWithMultipleEngines(t *testing.T) {
	searchBody := `{
		"results": [
			{"engine": "google", "engines": ["google"]},
			{"engine": "duckduckgo", "engines": ["duckduckgo", "brave"]},
			{"engine": "google", "engines": ["google"]}
		],
		"unresponsive_engines": []
	}`
	server := newFakeSearxng(t, searchBody, `{}`, http.StatusOK)

	report, err := Probe(context.Background(), nil, server.URL, "canary")
	if err != nil {
		t.Fatalf("Probe returned error: %v", err)
	}
	if report.Status != StatusHealthy {
		t.Errorf("expected status %q, got %q", StatusHealthy, report.Status)
	}
	want := []string{"brave", "duckduckgo", "google"}
	if len(report.ResponsiveEngines) != len(want) {
		t.Fatalf("expected %d responsive engines, got %v", len(want), report.ResponsiveEngines)
	}
	for i, engine := range want {
		if report.ResponsiveEngines[i] != engine {
			t.Errorf("responsive engine %d: expected %q, got %q", i, engine, report.ResponsiveEngines[i])
		}
	}
	if report.ResultCount != 3 {
		t.Errorf("expected 3 results, got %d", report.ResultCount)
	}
}

func TestProbeDegradedWithSingleEngine(t *testing.T) {
	searchBody := `{
		"results": [{"engine": "bing", "engines": ["bing"]}],
		"unresponsive_engines": [
			["duckduckgo", "CAPTCHA"],
			["google", "Suspended: too many requests"]
		]
	}`
	server := newFakeSearxng(t, searchBody, `{}`, http.StatusOK)

	report, err := Probe(context.Background(), nil, server.URL, "canary")
	if err != nil {
		t.Fatalf("Probe returned error: %v", err)
	}
	if report.Status != StatusDegraded {
		t.Errorf("expected status %q, got %q", StatusDegraded, report.Status)
	}
	if len(report.UnresponsiveEngines) != 2 {
		t.Fatalf("expected 2 unresponsive engines, got %v", report.UnresponsiveEngines)
	}
	if report.UnresponsiveEngines[0].Engine != "duckduckgo" || report.UnresponsiveEngines[0].Reason != "CAPTCHA" {
		t.Errorf("unexpected first issue: %+v", report.UnresponsiveEngines[0])
	}
}

func TestProbeCriticalWithNoResults(t *testing.T) {
	searchBody := `{"results": [], "unresponsive_engines": [["google", "Suspended"]]}`
	server := newFakeSearxng(t, searchBody, `{}`, http.StatusOK)

	report, err := Probe(context.Background(), nil, server.URL, "canary")
	if err != nil {
		t.Fatalf("Probe returned error: %v", err)
	}
	if report.Status != StatusCritical {
		t.Errorf("expected status %q, got %q", StatusCritical, report.Status)
	}
}

func TestProbeCollectsErrorStats(t *testing.T) {
	searchBody := `{"results": [{"engine": "google"}, {"engine": "brave"}], "unresponsive_engines": []}`
	// Real /stats/errors shape captured live 2026-06-10: log_message may be
	// JSON null.
	errorsBody := `{
		"qwant": [{"exception_classname": "json.decoder.JSONDecodeError", "log_message": null, "percentage": 100}],
		"wikidata": [
			{"exception_classname": "searx.exceptions.SearxEngineAccessDeniedException", "log_message": null},
			{"exception_classname": "searx.exceptions.SearxEngineAccessDeniedException", "log_message": null}
		]
	}`
	server := newFakeSearxng(t, searchBody, errorsBody, http.StatusOK)

	report, err := Probe(context.Background(), nil, server.URL, "canary")
	if err != nil {
		t.Fatalf("Probe returned error: %v", err)
	}
	if got := report.ErrorStats["qwant"]; len(got) != 1 || got[0] != "json.decoder.JSONDecodeError" {
		t.Errorf("unexpected qwant error stats: %v", got)
	}
	if got := report.ErrorStats["wikidata"]; len(got) != 1 {
		t.Errorf("expected deduplicated wikidata error stats, got %v", got)
	}
}

func TestProbeFailsOnNonOKSearch(t *testing.T) {
	server := newFakeSearxng(t, `Forbidden`, `{}`, http.StatusForbidden)

	if _, err := Probe(context.Background(), nil, server.URL, "canary"); err == nil {
		t.Fatal("expected error for non-200 canary search")
	}
}

func TestProbeRequiresBaseURL(t *testing.T) {
	if _, err := Probe(context.Background(), nil, "  ", "canary"); err == nil {
		t.Fatal("expected error for empty base URL")
	}
}

func TestClassifyBands(t *testing.T) {
	cases := []struct {
		engines int
		want    string
	}{
		{0, StatusCritical},
		{1, StatusDegraded},
		{2, StatusHealthy},
		{5, StatusHealthy},
	}
	for _, tc := range cases {
		if got := Classify(tc.engines); got != tc.want {
			t.Errorf("Classify(%d): expected %q, got %q", tc.engines, tc.want, got)
		}
	}
}
