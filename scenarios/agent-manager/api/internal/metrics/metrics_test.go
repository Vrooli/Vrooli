package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetSingleton(t *testing.T) {
	first := Get()
	if first == nil {
		t.Fatal("expected metrics instance, got nil")
	}
	second := Get()
	if first != second {
		t.Error("expected Get() to return singleton instance")
	}
	if first.RunsTotal == nil || first.HTTPRequestsTotal == nil {
		t.Error("expected core metrics to be initialized")
	}
}

func TestHandlerNotNil(t *testing.T) {
	if Handler() == nil {
		t.Error("expected metrics handler, got nil")
	}
}

func TestMetricsRecordOperationalSignalsAndExposePrometheusOutput(t *testing.T) {
	m := Get()
	m.RecordRunCreated("codex", "sandboxed")
	m.RecordSandboxAdoption("sandboxed", "protected", "false")
	m.RecordProvenanceWrite()
	m.RecordProvenanceSkipped()
	m.RecordRunCompleted("codex", "complete", 2*time.Second)
	m.RecordRunStop("api", true, time.Second)
	m.RecordRunStop("api", false, time.Second)
	m.RecordRunnerAvailability("codex", true)
	m.RecordRunnerAvailability("offline", false)
	m.RecordRunnerError("codex", "timeout")
	m.RecordEvent("status")
	m.RecordWebSocketConnection(2)
	m.RecordWebSocketConnection(-1)
	m.RecordWebSocketMessage("progress")
	m.RecordHTTPRequest(http.MethodGet, "/api/v1/runs", "200", 15*time.Millisecond, 128)
	m.RecordCost("codex", 1.25)
	m.RecordTokens("codex", 10, 20, 3, 4)
	m.UpdateStatusCounts(map[string]int{"running": 2}, map[string]int{"queued": 3})

	recorder := httptest.NewRecorder()
	Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if recorder.Code != http.StatusOK || recorder.Body.Len() == 0 {
		t.Fatalf("metrics status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
