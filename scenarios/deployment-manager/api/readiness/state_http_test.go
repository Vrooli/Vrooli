package readiness

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"deployment-manager/releases"
)

type latestReadiness struct{}

func (latestReadiness) GetLatestReadiness(context.Context, string) (*releases.ReadinessRecord, error) {
	return &releases.ReadinessRecord{ReadinessGoalRef: "goal-1", GoalClosed: true, ApprovedAtCommit: "abc"}, nil
}

func TestStateHandlerReturnsLatestProjection(t *testing.T) {
	rec := httptest.NewRecorder()
	StateHandler(latestReadiness{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?scenario=demo", nil))
	if rec.Code != http.StatusOK || rec.Body.String() == "" {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
