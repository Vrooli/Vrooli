package readiness

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"deployment-manager/releases"
)

type latestReadiness struct{}

func (latestReadiness) GetLatestReadiness(context.Context, string) (*releases.ReadinessRecord, error) {
	return &releases.ReadinessRecord{ReadinessGoalRef: "goal-1", GoalClosed: true, ApprovedAtCommit: "abc", ObservedAt: time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)}, nil
}

func TestStateHandlerReturnsLatestProjection(t *testing.T) {
	rec := httptest.NewRecorder()
	StateHandler(latestReadiness{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?scenario=demo", nil))
	var projection map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &projection); err != nil {
		t.Fatalf("decode projection: %v", err)
	}
	if rec.Code != http.StatusOK || projection["observed_at"] != "2026-11-21T12:00:00Z" || projection["goal_exists"] != true {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

type missingObservationTime struct{}

func (missingObservationTime) GetLatestReadiness(context.Context, string) (*releases.ReadinessRecord, error) {
	return &releases.ReadinessRecord{ReadinessGoalRef: "goal-1"}, nil
}

func TestStateHandlerRejectsProjectionWithoutObservationTime(t *testing.T) {
	rec := httptest.NewRecorder()
	StateHandler(missingObservationTime{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?scenario=demo", nil))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s, want unqualified projection rejected", rec.Code, rec.Body.String())
	}
}
