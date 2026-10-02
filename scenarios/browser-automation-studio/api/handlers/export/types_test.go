package export

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRequestUnmarshalUsesGeneratedReplaySpecContract(t *testing.T) {
	const payload = `{"format":"json","movie_spec":{"version":"2025-11-07","generated_at":"2026-09-15T12:30:00Z","execution":{"execution_id":"00000000-0000-4000-8000-000000000001","workflow_id":"00000000-0000-4000-8000-000000000002","status":"completed","started_at":"2026-09-15T12:00:00Z","progress":100,"total_duration_ms":4200},"frames":[],"assets":[]}}`

	var request Request
	if err := json.Unmarshal([]byte(payload), &request); err != nil {
		t.Fatalf("unmarshal generated ReplaySpec payload: %v", err)
	}
	if request.MovieSpec == nil {
		t.Fatal("movie_spec was not decoded as generated ReplaySpec")
	}
	if request.MovieSpec.GetExecution().GetExecutionId() != "00000000-0000-4000-8000-000000000001" {
		t.Fatalf("execution id = %q", request.MovieSpec.GetExecution().GetExecutionId())
	}
	if got := request.MovieSpec.GetGeneratedAt().AsTime().UTC().Format(time.RFC3339); got != "2026-09-15T12:30:00Z" {
		t.Fatalf("generated_at = %s", got)
	}
}

func TestRequestUnmarshalRejectsUnknownReplaySpecField(t *testing.T) {
	var request Request
	err := json.Unmarshal([]byte(`{"movie_spec":{"version":"v1","unexpected":"value"}}`), &request)
	if err == nil {
		t.Fatal("expected generated ReplaySpec validation to reject unknown fields")
	}
}
