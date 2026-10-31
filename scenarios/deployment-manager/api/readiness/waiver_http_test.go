package readiness

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type recordingWaiver struct{ profile, commit, reason, actor string }

func (r *recordingWaiver) RecordReadinessWaiver(_ context.Context, profile, commit, reason, actor string) error {
	r.profile, r.commit, r.reason, r.actor = profile, commit, reason, actor
	return nil
}

func TestWaiverHandlerRecordsExactOperatorException(t *testing.T) {
	recorder := &recordingWaiver{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/readiness/waiver", strings.NewReader(`{"profile_id":"p1","commit":"abc","reason":"incident","actor":"operator"}`))
	rec := httptest.NewRecorder()
	WaiverHandler(recorder).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if recorder.profile != "p1" || recorder.commit != "abc" || recorder.reason != "incident" || recorder.actor != "operator" {
		t.Fatalf("recorded = %+v", recorder)
	}
}

func TestWaiverHandlerRequiresReasonAndActor(t *testing.T) {
	rec := httptest.NewRecorder()
	WaiverHandler(&recordingWaiver{}).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"profile_id":"p1","commit":"abc"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
