package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchScenarioAuditorSummary(t *testing.T) {
	summaryPayload := ScenarioAuditorSummary{
		Total:           5,
		BySeverity:      map[string]int{"critical": 2, "high": 3},
		HighestSeverity: "critical",
		TopViolations:   []ScenarioAuditorViolationExcerpt{{ID: "V-1", Severity: "critical", Title: "Something"}},
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/standards/violations/summary" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"summary": summaryPayload})
	}))
	defer server.Close()

	t.Setenv("SCENARIO_AUDITOR_API_BASE_URL", server.URL)

	service := &AppService{
		httpClient: server.Client(),
		timeNow:    func() time.Time { return time.Now() },
	}

	summary, err := service.fetchScenarioAuditorSummary(context.Background(), "demo")
	if err != nil {
		t.Fatalf("expected summary, got error: %v", err)
	}
	if summary == nil {
		t.Fatalf("expected non-nil summary")
	}
	if summary.Total != 5 || summary.HighestSeverity != "critical" {
		t.Fatalf("unexpected summary contents: %#v", summary)
	}
}
