package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"

	"github.com/google/uuid"
)

type fakeEventRepository struct {
	records []eventlog.Record
	err     error
	limit   int
	mode    string
}

func (f *fakeEventRepository) SinceForRun(context.Context, uuid.UUID, int64, int) ([]eventlog.Record, error) {
	f.mode = "run"
	return f.records, f.err
}

func (f *fakeEventRepository) SinceID(_ context.Context, _ int64, limit int) ([]eventlog.Record, error) {
	f.mode, f.limit = "all", limit
	return f.records, f.err
}

func (f *fakeEventRepository) ByEventType(context.Context, domain.RunEventType, time.Time, int) ([]eventlog.Record, error) {
	f.mode = "type"
	return f.records, f.err
}

func TestEventsHandlerValidatesFiltersAndCapsLimit(t *testing.T) {
	repo := &fakeEventRepository{}
	h := NewEventsHandler(repo)
	for _, raw := range []string{"?run=not-a-uuid", "?type=legacy", "?type=model.health.transition&since=not-a-time"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/events"+raw, nil)
		rw := httptest.NewRecorder()
		h.ListEvents(rw, req)
		if rw.Code != http.StatusBadRequest {
			t.Fatalf("%s status=%d", raw, rw.Code)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?limit=9999", nil)
	rw := httptest.NewRecorder()
	h.ListEvents(rw, req)
	if rw.Code != http.StatusOK || repo.mode != "all" || repo.limit != 1000 {
		t.Fatalf("capped query status=%d mode=%s limit=%d", rw.Code, repo.mode, repo.limit)
	}
}

func TestEventsHandlerReturnsRowsAndRepositoryFailure(t *testing.T) {
	runID := uuid.New()
	repo := &fakeEventRepository{records: []eventlog.Record{{ID: uuid.New(), RunID: runID, Sequence: 2, EventType: domain.EventTypeModelHealthTransition, Timestamp: time.Now().UTC()}}}
	h := NewEventsHandler(repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?run="+runID.String(), nil)
	rw := httptest.NewRecorder()
	h.ListEvents(rw, req)
	if rw.Code != http.StatusOK || repo.mode != "run" {
		t.Fatalf("run query status=%d mode=%s", rw.Code, repo.mode)
	}
	repo.err = errors.New("boom")
	rw = httptest.NewRecorder()
	h.ListEvents(rw, httptest.NewRequest(http.MethodGet, "/api/v1/events", nil))
	if rw.Code != http.StatusInternalServerError {
		t.Fatalf("failure status=%d", rw.Code)
	}
}
