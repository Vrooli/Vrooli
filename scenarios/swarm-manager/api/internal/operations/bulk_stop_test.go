package operations

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"swarm-manager/internal/agentactivity"

	"github.com/gorilla/mux"
)

// recordingStopper captures the order in which StopRun is called and
// optionally returns a configured error per run id. The mutex is needed
// only because the test that asserts serial cancellation also asserts
// non-overlap; in production the handler iterates serially so concurrent
// access to this field never happens.
type recordingStopper struct {
	mu    sync.Mutex
	calls []string
	// errors maps runID → error to return. Missing entries return nil.
	errors map[string]error
	// activeOverlap counts how many calls are in flight at any moment;
	// the handler must keep this at 1 for every observation to satisfy
	// the serial-cancellation contract.
	activeMu sync.Mutex
	active   int
	maxSeen  int
	// hold blocks each StopRun until the test releases it; used by the
	// serial-cancellation test to widen the overlap window.
	hold chan struct{}
}

func (r *recordingStopper) StopRun(_ context.Context, runID string) error {
	r.activeMu.Lock()
	r.active++
	if r.active > r.maxSeen {
		r.maxSeen = r.active
	}
	r.activeMu.Unlock()

	if r.hold != nil {
		<-r.hold
	}

	r.mu.Lock()
	r.calls = append(r.calls, runID)
	err := r.errors[runID]
	r.mu.Unlock()

	r.activeMu.Lock()
	r.active--
	r.activeMu.Unlock()
	return err
}

func newBulkRouter(t *testing.T, stopper Stopper, lister ActivityLister) *mux.Router {
	t.Helper()
	h, err := NewBulkStopHandler(stopper, lister)
	if err != nil {
		t.Fatalf("NewBulkStopHandler: %v", err)
	}
	r := mux.NewRouter()
	h.RegisterRoutes(r)
	return r
}

func decodeBulk(t *testing.T, body []byte) BulkStopResponse {
	t.Helper()
	var resp BulkStopResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, string(body))
	}
	return resp
}

func bulkRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// --- happy-path / shape tests -------------------------------------------

func TestBulkStop_RunIDs_HappyPath(t *testing.T) {
	stopper := &recordingStopper{}
	lister := &fakeActivityLister{}
	r := newBulkRouter(t, stopper, lister)

	body := `{"run_ids": ["run-a", "run-b", "run-c"]}`
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, bulkRequest("POST", "/api/v1/operations/bulk-stop", body))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	resp := decodeBulk(t, rec.Body.Bytes())
	if resp.Total != 3 || resp.Stopped != 3 || resp.Failed != 0 {
		t.Fatalf("counts = (total=%d, stopped=%d, failed=%d), want (3,3,0)", resp.Total, resp.Stopped, resp.Failed)
	}
	if got := []string{stopper.calls[0], stopper.calls[1], stopper.calls[2]}; got[0] != "run-a" || got[1] != "run-b" || got[2] != "run-c" {
		t.Fatalf("calls = %v, want [run-a run-b run-c] in order", stopper.calls)
	}
	for i, o := range resp.Outcomes {
		if !o.Success || o.Error != "" {
			t.Fatalf("outcome[%d] = %+v, want success", i, o)
		}
	}
}

func TestBulkStop_RunIDs_DedupesAndTrims(t *testing.T) {
	stopper := &recordingStopper{}
	r := newBulkRouter(t, stopper, &fakeActivityLister{})

	body := `{"run_ids": ["run-a", " run-a", "  ", "run-b", "run-a"]}`
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, bulkRequest("POST", "/api/v1/operations/bulk-stop", body))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	resp := decodeBulk(t, rec.Body.Bytes())
	if resp.Total != 2 {
		t.Fatalf("Total = %d, want 2 (after dedupe + trim)", resp.Total)
	}
	if len(stopper.calls) != 2 || stopper.calls[0] != "run-a" || stopper.calls[1] != "run-b" {
		t.Fatalf("calls = %v, want [run-a run-b]", stopper.calls)
	}
}

func TestBulkStop_RunIDs_PartialFailure(t *testing.T) {
	stopper := &recordingStopper{
		errors: map[string]error{
			"run-b": errors.New("agent-manager unreachable"),
		},
	}
	r := newBulkRouter(t, stopper, &fakeActivityLister{})

	body := `{"run_ids": ["run-a", "run-b", "run-c"]}`
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, bulkRequest("POST", "/api/v1/operations/bulk-stop", body))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	resp := decodeBulk(t, rec.Body.Bytes())
	if resp.Total != 3 || resp.Stopped != 2 || resp.Failed != 1 {
		t.Fatalf("counts = (total=%d, stopped=%d, failed=%d), want (3,2,1)", resp.Total, resp.Stopped, resp.Failed)
	}
	// Cancellation must continue past failures so the operator gets a
	// complete outcome list rather than aborting at the first error.
	if len(stopper.calls) != 3 {
		t.Fatalf("calls = %v, want all 3 attempted", stopper.calls)
	}
	for _, o := range resp.Outcomes {
		if o.RunID == "run-b" {
			if o.Success || o.Error == "" {
				t.Fatalf("run-b outcome = %+v, want failure with message", o)
			}
			if !strings.Contains(o.Error, "unreachable") {
				t.Fatalf("run-b error = %q, want it to mention 'unreachable'", o.Error)
			}
		} else if !o.Success {
			t.Fatalf("%s outcome = %+v, want success", o.RunID, o)
		}
	}
}

func TestBulkStop_SerializesCancellation(t *testing.T) {
	hold := make(chan struct{})
	stopper := &recordingStopper{hold: hold}
	r := newBulkRouter(t, stopper, &fakeActivityLister{})

	body := `{"run_ids": ["run-a", "run-b", "run-c"]}`
	done := make(chan struct{})
	go func() {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, bulkRequest("POST", "/api/v1/operations/bulk-stop", body))
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d", rec.Code)
		}
		close(done)
	}()

	// Release one StopRun at a time and assert the observed concurrency
	// never exceeds 1 — i.e. each call started after the previous one
	// returned, never in parallel.
	for i := 0; i < 3; i++ {
		hold <- struct{}{}
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("handler did not return within 2s")
	}

	if stopper.maxSeen > 1 {
		t.Fatalf("observed concurrent StopRun calls (maxSeen=%d), want strictly serial", stopper.maxSeen)
	}
	if len(stopper.calls) != 3 {
		t.Fatalf("len(calls) = %d, want 3", len(stopper.calls))
	}
}

// --- input validation ---------------------------------------------------

func TestBulkStop_RejectsEmptyBody(t *testing.T) {
	r := newBulkRouter(t, &recordingStopper{}, &fakeActivityLister{})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, bulkRequest("POST", "/api/v1/operations/bulk-stop", `{}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestBulkStop_RejectsBothRunIDsAndFilter(t *testing.T) {
	r := newBulkRouter(t, &recordingStopper{}, &fakeActivityLister{})

	body := `{"run_ids": ["run-a"], "filter": {"lane": "execute"}}`
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, bulkRequest("POST", "/api/v1/operations/bulk-stop", body))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestBulkStop_RejectsAllEmptyRunIDs(t *testing.T) {
	r := newBulkRouter(t, &recordingStopper{}, &fakeActivityLister{})

	body := `{"run_ids": ["  ", ""]}`
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, bulkRequest("POST", "/api/v1/operations/bulk-stop", body))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (no usable IDs)", rec.Code)
	}
}

func TestBulkStop_RejectsTooManyRunIDs(t *testing.T) {
	r := newBulkRouter(t, &recordingStopper{}, &fakeActivityLister{})

	ids := make([]string, maxBulkStopRunIDs+1)
	for i := range ids {
		ids[i] = "run-" + strings.Repeat("x", 4) + string(rune('A'+(i%26))) + string(rune('0'+(i/26%10)))
	}
	body, _ := json.Marshal(map[string]any{"run_ids": ids})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, bulkRequest("POST", "/api/v1/operations/bulk-stop", string(body)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestBulkStop_RejectsMalformedJSON(t *testing.T) {
	r := newBulkRouter(t, &recordingStopper{}, &fakeActivityLister{})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, bulkRequest("POST", "/api/v1/operations/bulk-stop", `{not-json`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestBulkStop_RejectsInvalidLaneFilter(t *testing.T) {
	r := newBulkRouter(t, &recordingStopper{}, &fakeActivityLister{})

	body := `{"filter": {"lane": "deploy"}}`
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, bulkRequest("POST", "/api/v1/operations/bulk-stop", body))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestBulkStop_RejectsNonActiveStatusFilter(t *testing.T) {
	r := newBulkRouter(t, &recordingStopper{}, &fakeActivityLister{})

	body := `{"filter": {"status": "completed"}}`
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, bulkRequest("POST", "/api/v1/operations/bulk-stop", body))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (completed is not stoppable)", rec.Code)
	}
}

// --- filter-mode resolution --------------------------------------------

func TestBulkStop_FilterByLane(t *testing.T) {
	now := fixedNow()()
	records := []agentactivity.Record{
		{
			ActivityID: "i1", RunID: "run-i1", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "i1", Purpose: agentactivity.PurposeWorkshop, Status: agentactivity.StatusRunning,
			RequestedAt: now.Add(-time.Minute).Format(time.RFC3339),
		},
		{
			ActivityID: "e1", RunID: "run-e1", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "e1", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusRunning,
			RequestedAt: now.Format(time.RFC3339),
		},
	}
	stopper := &recordingStopper{}
	r := newBulkRouter(t, stopper, &fakeActivityLister{records: records})

	body := `{"filter": {"lane": "execute"}}`
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, bulkRequest("POST", "/api/v1/operations/bulk-stop", body))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	resp := decodeBulk(t, rec.Body.Bytes())
	if resp.Total != 1 || resp.Outcomes[0].RunID != "run-e1" {
		t.Fatalf("Outcomes = %+v, want [run-e1] only", resp.Outcomes)
	}
}

func TestBulkStop_FilterByStatus(t *testing.T) {
	now := fixedNow()()
	records := []agentactivity.Record{
		{
			ActivityID: "running", RunID: "run-running", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "x", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusRunning,
			RequestedAt: now.Format(time.RFC3339),
		},
		{
			ActivityID: "needs", RunID: "run-needs", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "y", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusNeedsReview,
			RequestedAt: now.Format(time.RFC3339),
		},
	}
	stopper := &recordingStopper{}
	r := newBulkRouter(t, stopper, &fakeActivityLister{records: records})

	body := `{"filter": {"status": "needs_review"}}`
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, bulkRequest("POST", "/api/v1/operations/bulk-stop", body))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	resp := decodeBulk(t, rec.Body.Bytes())
	if resp.Total != 1 || resp.Outcomes[0].RunID != "run-needs" {
		t.Fatalf("Outcomes = %+v, want [run-needs] only", resp.Outcomes)
	}
}

func TestBulkStop_FilterCombinesLaneAndStatus(t *testing.T) {
	now := fixedNow()()
	records := []agentactivity.Record{
		// match: execute + running
		{
			ActivityID: "a", RunID: "run-a", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "a", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusRunning,
			RequestedAt: now.Format(time.RFC3339),
		},
		// no match: execute but needs_review
		{
			ActivityID: "b", RunID: "run-b", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "b", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusNeedsReview,
			RequestedAt: now.Format(time.RFC3339),
		},
		// no match: investigate + running
		{
			ActivityID: "c", RunID: "run-c", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "c", Purpose: agentactivity.PurposeWorkshop, Status: agentactivity.StatusRunning,
			RequestedAt: now.Format(time.RFC3339),
		},
	}
	stopper := &recordingStopper{}
	r := newBulkRouter(t, stopper, &fakeActivityLister{records: records})

	body := `{"filter": {"lane": "execute", "status": "running"}}`
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, bulkRequest("POST", "/api/v1/operations/bulk-stop", body))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	resp := decodeBulk(t, rec.Body.Bytes())
	if resp.Total != 1 || resp.Outcomes[0].RunID != "run-a" {
		t.Fatalf("Outcomes = %+v, want [run-a] only", resp.Outcomes)
	}
}

func TestBulkStop_FilterEmptyMatch(t *testing.T) {
	stopper := &recordingStopper{}
	r := newBulkRouter(t, stopper, &fakeActivityLister{})

	body := `{"filter": {"lane": "reconcile"}}`
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, bulkRequest("POST", "/api/v1/operations/bulk-stop", body))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	resp := decodeBulk(t, rec.Body.Bytes())
	if resp.Total != 0 || resp.Stopped != 0 || resp.Failed != 0 {
		t.Fatalf("counts = (%d,%d,%d), want all zero", resp.Total, resp.Stopped, resp.Failed)
	}
	if len(stopper.calls) != 0 {
		t.Fatalf("calls = %v, want none", stopper.calls)
	}
}

func TestBulkStop_FilterNewestFirst(t *testing.T) {
	now := fixedNow()()
	records := []agentactivity.Record{
		{
			ActivityID: "old", RunID: "run-old", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "old", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusRunning,
			RequestedAt: now.Add(-10 * time.Minute).Format(time.RFC3339),
		},
		{
			ActivityID: "new", RunID: "run-new", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "new", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusRunning,
			RequestedAt: now.Format(time.RFC3339),
		},
	}
	stopper := &recordingStopper{}
	r := newBulkRouter(t, stopper, &fakeActivityLister{records: records})

	body := `{"filter": {"lane": "execute"}}`
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, bulkRequest("POST", "/api/v1/operations/bulk-stop", body))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if len(stopper.calls) != 2 || stopper.calls[0] != "run-new" || stopper.calls[1] != "run-old" {
		t.Fatalf("calls = %v, want [run-new run-old] (newest first)", stopper.calls)
	}
}

func TestBulkStop_FilterSkipsRecordsWithoutRunID(t *testing.T) {
	now := fixedNow()()
	records := []agentactivity.Record{
		{
			ActivityID: "no-run", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "x", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusPending,
			RequestedAt: now.Format(time.RFC3339),
		},
		{
			ActivityID: "with-run", RunID: "run-with", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "y", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusRunning,
			RequestedAt: now.Format(time.RFC3339),
		},
	}
	stopper := &recordingStopper{}
	r := newBulkRouter(t, stopper, &fakeActivityLister{records: records})

	body := `{"filter": {"lane": "execute"}}`
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, bulkRequest("POST", "/api/v1/operations/bulk-stop", body))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	resp := decodeBulk(t, rec.Body.Bytes())
	if resp.Total != 1 || resp.Outcomes[0].RunID != "run-with" {
		t.Fatalf("Outcomes = %+v, want [run-with] only", resp.Outcomes)
	}
}

// --- constructor validation --------------------------------------------

func TestNewBulkStopHandler_RequiresStopper(t *testing.T) {
	if _, err := NewBulkStopHandler(nil, &fakeActivityLister{}); err == nil {
		t.Fatalf("want error when stopper is nil")
	}
}

func TestNewBulkStopHandler_RequiresActivities(t *testing.T) {
	if _, err := NewBulkStopHandler(&recordingStopper{}, nil); err == nil {
		t.Fatalf("want error when activities is nil")
	}
}
