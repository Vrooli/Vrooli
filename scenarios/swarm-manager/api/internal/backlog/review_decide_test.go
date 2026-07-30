package backlog

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/attemptstore"
	"swarm-manager/internal/followup"
	"swarm-manager/internal/review"

	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

type recordingFollowUpDispatcher struct {
	called   bool
	steering string
}

func TestDecideReview_RecordsVerifiedAtAccept(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	item := BacklogItem{Name: "verification-snapshot", Title: "Verification snapshot", Status: StatusReviewPending, Kind: KindExecute}
	createTestItem(t, rootDir, KindExecute, item)
	itemDir := filepath.Join(rootDir, backlogKindDirs[KindExecute], item.Name)
	if err := attemptstore.SaveRound(itemDir, "review", review.Round{RoundNum: 1, Status: review.RoundStatusComplete, Evidence: []review.EvidenceItem{
		{ID: "verified", Title: "Verified", Verified: true},
		{ID: "unverified", Title: "Unverified", Verified: false},
	}}); err != nil {
		t.Fatalf("save review round: %v", err)
	}
	if _, err := h.DecideReview(context.Background(), KindExecute, item.Name, ReviewDecideRequest{Decision: ReviewDecisionAccept, DecidedBy: "operator:test", Rationale: "Reviewed the available proof."}); err != nil {
		t.Fatalf("DecideReview: %v", err)
	}
	decisionsDir := filepath.Join(itemDir, "review", "decisions")
	entries, err := os.ReadDir(decisionsDir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("decision entries = %d, %v", len(entries), err)
	}
	var record reviewDecisionRecord
	data, err := os.ReadFile(filepath.Join(decisionsDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("read decision: %v", err)
	}
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatalf("decode decision: %v", err)
	}
	if got := record.VerifiedAtAccept; got.Verified != 1 || got.Total != 2 || got.Ratio != 0.5 {
		t.Fatalf("verified_at_accept = %+v, want verified=1 total=2 ratio=.5", got)
	}
}

func (d *recordingFollowUpDispatcher) DispatchFollowUp(_ context.Context, _ BacklogKind, _ string, steering string) error {
	d.called = true
	d.steering = steering
	return nil
}

// TestReviewDecide_AcceptFromReviewPending verifies that an item in
// review_pending can be flipped to completed via the accept decision, and
// that a decision record is written to disk.
func TestReviewDecide_AcceptFromReviewPending(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	item := BacklogItem{
		Name:   "sample-item",
		Title:  "Sample",
		Status: StatusReviewPending,
		Kind:   KindExecute,
	}
	createTestItem(t, rootDir, KindExecute, item)

	body, _ := json.Marshal(ReviewDecideRequest{
		Decision:  ReviewDecisionAccept,
		Rationale: "Looks good to me",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/sample-item/review-decide", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "sample-item"})
	rec := httptest.NewRecorder()
	h.ReviewDecide(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	updated, err := h.store.LoadItem(KindExecute, "sample-item")
	if err != nil {
		t.Fatalf("reload item: %v", err)
	}
	if updated.Status != StatusCompleted {
		t.Errorf("status = %q, want %q", updated.Status, StatusCompleted)
	}

	// Verify a decision record was written.
	decisionsDir := filepath.Join(rootDir, backlogKindDirs[KindExecute], "sample-item", "review", "decisions")
	entries, err := os.ReadDir(decisionsDir)
	if err != nil {
		t.Fatalf("read decisions dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 decision file, got %d", len(entries))
	}
	if !strings.HasSuffix(entries[0].Name(), "-accept.json") {
		t.Errorf("decision filename = %q, expected suffix -accept.json", entries[0].Name())
	}
}

// TestReviewDecide_RejectedFromWrongStatus verifies that review-decide
// only fires when the item is in review_pending. Any other status (in_progress,
// in_review, already-terminal) must be rejected.
func TestReviewDecide_RejectedFromWrongStatus(t *testing.T) {
	tests := []struct {
		name   string
		status BacklogStatus
	}{
		{"in_progress", StatusInProgress},
		{"in_review", StatusInReview},
		{"completed", StatusCompleted},
		{"failed", StatusFailed},
		{"backlog", StatusBacklog},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, rootDir := setupTestHandler(t)
			createTestItem(t, rootDir, KindExecute, BacklogItem{
				Name:   "item-" + tc.name,
				Status: tc.status,
				Kind:   KindExecute,
			})

			body, _ := json.Marshal(ReviewDecideRequest{Decision: ReviewDecisionAccept})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/item-"+tc.name+"/review-decide", bytes.NewReader(body))
			req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "item-" + tc.name})
			rec := httptest.NewRecorder()
			h.ReviewDecide(rec, req)

			if rec.Code == http.StatusOK {
				t.Fatalf("expected non-200 for status %q, got 200: %s", tc.status, rec.Body.String())
			}
		})
	}
}

// TestReviewDecide_ItemTerminalHandlerFires verifies the SetItemTerminalHandler
// callback is invoked synchronously after a successful review-decide with the
// target terminal status. This is the hook the milestone review service
// uses to check whether the milestone is now ready for its own review.
func TestReviewDecide_ItemTerminalHandlerFires(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindExecute, BacklogItem{
		Name:   "hook-item",
		Status: StatusReviewPending,
		Kind:   KindExecute,
	})

	var gotKind, gotName string
	var gotStatus BacklogStatus
	var called int
	h.SetItemTerminalHandler(func(_ context.Context, kind, name string, status BacklogStatus) {
		called++
		gotKind = kind
		gotName = name
		gotStatus = status
	})

	body, _ := json.Marshal(ReviewDecideRequest{Decision: ReviewDecisionAccept})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/hook-item/review-decide", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "hook-item"})
	rec := httptest.NewRecorder()
	h.ReviewDecide(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if called != 1 {
		t.Fatalf("callback fired %d times, want 1", called)
	}
	if gotKind != "execute" || gotName != "hook-item" {
		t.Errorf("callback args = (%q, %q), want (execute, hook-item)", gotKind, gotName)
	}
	if gotStatus != StatusCompleted {
		t.Errorf("callback status = %q, want completed", gotStatus)
	}
}

// TestReviewDecide_NoHandler_IsOK verifies that review-decide still succeeds
// when no terminal handler has been installed (the hook is purely optional).
func TestReviewDecide_NoHandler_IsOK(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindExecute, BacklogItem{
		Name:   "no-hook",
		Status: StatusReviewPending,
		Kind:   KindExecute,
	})

	body, _ := json.Marshal(ReviewDecideRequest{Decision: ReviewDecisionAccept})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/no-hook/review-decide", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "no-hook"})
	rec := httptest.NewRecorder()
	h.ReviewDecide(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with no hook installed, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestReviewDecide_FollowupMapsToNeedsFollowup verifies the followup decision
// produces the needs_followup terminal status (not failed, not completed).
func TestReviewDecide_FollowupMapsToNeedsFollowup(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindExecute, BacklogItem{
		Name:   "followup-item",
		Status: StatusReviewPending,
		Kind:   KindExecute,
	})

	body, _ := json.Marshal(ReviewDecideRequest{Decision: ReviewDecisionFollowup, Rationale: "UI needs another pass", FollowUp: &FollowUp{Steering: "Correct the mobile layout and revalidate it.", Disposition: FollowUpRun}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/followup-item/review-decide", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "followup-item"})
	rec := httptest.NewRecorder()
	h.ReviewDecide(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	updated, _ := h.store.LoadItem(KindExecute, "followup-item")
	if updated.Status != StatusNeedsFollowup {
		t.Errorf("status = %q, want %q", updated.Status, StatusNeedsFollowup)
	}
	if updated.PendingFollowUp == nil || updated.PendingFollowUp.Disposition != FollowUpRun {
		t.Fatalf("pending follow-up = %#v", updated.PendingFollowUp)
	}
}

func TestReviewDecide_FollowupRequiresTypedInstruction(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindExecute, BacklogItem{Name: "missing-followup", Status: StatusReviewPending, Kind: KindExecute})
	body, _ := json.Marshal(ReviewDecideRequest{Decision: ReviewDecisionFollowup})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/missing-followup/review-decide", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "missing-followup"})
	rec := httptest.NewRecorder()
	h.ReviewDecide(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	item, _ := h.store.LoadItem(KindExecute, "missing-followup")
	if item.Status != StatusReviewPending {
		t.Fatalf("status mutated to %q", item.Status)
	}
}

// TestUpdatePatch_RejectsInternalStatuses verifies that the generic PATCH
// endpoint cannot be used to set statuses that belong to the execution or
// review systems. These statuses must only be set by the execution/review
// flows (for in_review/review_pending) or via review-decide (for terminal
// statuses).
func TestUpdatePatch_RejectsInternalStatuses(t *testing.T) {
	internalStatuses := []string{"in_review", "review_pending"}
	for _, st := range internalStatuses {
		t.Run(st, func(t *testing.T) {
			fields := backlogUpdateFieldSet{updateFieldStatus: {}}
			status := st
			req := &apipb.UpdateBacklogItemRequest{Status: &status}
			if errMsg := validateUpdateBacklogItemRequest(req, fields, KindExecute, StatusBacklog); errMsg == "" {
				t.Fatalf("expected status %q to be rejected by validator, got empty error", st)
			}
		})
	}
}

// TestUpdatePatch_HTTP_RejectsInternalStatusesAsTarget is the HTTP-layer
// companion to TestUpdatePatch_RejectsInternalStatuses. It drives a real
// PATCH request through the Update handler to verify the validator is wired
// into the HTTP layer (not just a unit-level check).
func TestUpdatePatch_HTTP_RejectsInternalStatusesAsTarget(t *testing.T) {
	for _, target := range []string{"in_review", "review_pending"} {
		t.Run(target, func(t *testing.T) {
			h, rootDir := setupTestHandler(t)
			createTestItem(t, rootDir, KindExecute, BacklogItem{
				Name:   "internal-target",
				Title:  "T",
				Status: StatusBacklog,
				Kind:   KindExecute,
			})
			w := doUpdate(t, h, "execute", "internal-target", map[string]any{"status": target})
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 for PATCH status=%q, got %d: %s", target, w.Code, w.Body.String())
			}
		})
	}
}

// TestUpdatePatch_HTTP_RejectsStatusChangeFromReviewStates verifies that PATCH
// cannot flip an item out of in_review or review_pending to any other status;
// the review-decide endpoint is the only allowed path. This is the guard that
// preserves the audit trail (decision record + itemTerminalHandler).
// [REQ:SWM-P0-008] terminal status reachable only through operator review decision
func TestUpdatePatch_HTTP_RejectsStatusChangeFromReviewStates(t *testing.T) {
	cases := []struct {
		existing BacklogStatus
		target   string
	}{
		{StatusReviewPending, "completed"},
		{StatusReviewPending, "failed"},
		{StatusReviewPending, "needs_followup"},
		{StatusInReview, "completed"},
		{StatusInReview, "backlog"},
	}
	for _, tc := range cases {
		t.Run(string(tc.existing)+"->"+tc.target, func(t *testing.T) {
			h, rootDir := setupTestHandler(t)
			createTestItem(t, rootDir, KindExecute, BacklogItem{
				Name:   "guarded",
				Title:  "T",
				Status: tc.existing,
				Kind:   KindExecute,
			})
			w := doUpdate(t, h, "execute", "guarded", map[string]any{"status": tc.target})
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 for %s→%s, got %d: %s", tc.existing, tc.target, w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), "review-decide") {
				t.Errorf("expected error to mention review-decide, got: %s", w.Body.String())
			}

			updated, _ := h.store.LoadItem(KindExecute, "guarded")
			if updated.Status != tc.existing {
				t.Errorf("item status changed despite rejection: %q → %q", tc.existing, updated.Status)
			}
		})
	}
}

// TestUpdatePatch_HTTP_AllowsStatusChangeFromNonReviewStates verifies the
// manual-accept path (failed → completed via PATCH) is preserved. This is the
// existing escape hatch users rely on to override a failed run without going
// through review-decide (which only accepts review_pending input).
func TestUpdatePatch_HTTP_AllowsStatusChangeFromNonReviewStates(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindExecute, BacklogItem{
		Name:   "manual-accept",
		Title:  "T",
		Status: StatusFailed,
		Kind:   KindExecute,
	})
	w := doUpdate(t, h, "execute", "manual-accept", map[string]any{"status": "completed"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for failed→completed manual accept, got %d: %s", w.Code, w.Body.String())
	}
	updated, _ := h.store.LoadItem(KindExecute, "manual-accept")
	if updated.Status != StatusCompleted {
		t.Errorf("status = %q, want completed", updated.Status)
	}
}

// TestUpdatePatch_HTTP_AllowsNonStatusFieldsInReviewStates verifies that while
// status is guarded in review states, other fields (title, tags) can still be
// edited — the guard is narrowly scoped to status transitions.
func TestUpdatePatch_HTTP_AllowsNonStatusFieldsInReviewStates(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindExecute, BacklogItem{
		Name:   "review-edit",
		Title:  "Original",
		Status: StatusReviewPending,
		Kind:   KindExecute,
	})
	w := doUpdate(t, h, "execute", "review-edit", map[string]any{"title": "Revised"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for non-status edit in review_pending, got %d: %s", w.Code, w.Body.String())
	}
	updated, _ := h.store.LoadItem(KindExecute, "review-edit")
	if updated.Status != StatusReviewPending {
		t.Errorf("status changed unexpectedly: %q", updated.Status)
	}
	if updated.Title != "Revised" {
		t.Errorf("title = %q, want Revised", updated.Title)
	}
}

// TestReviewDecide_DoubleDecide verifies that calling review-decide twice on
// the same item rejects the second call: after the first accept, the status
// is completed (not review_pending), so the guard triggers.
func TestReviewDecide_DoubleDecide(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindExecute, BacklogItem{
		Name:   "twice",
		Status: StatusReviewPending,
		Kind:   KindExecute,
	})

	body1, _ := json.Marshal(ReviewDecideRequest{Decision: ReviewDecisionAccept})
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/twice/review-decide", bytes.NewReader(body1))
	req1 = mux.SetURLVars(req1, map[string]string{"kind": "execute", "name": "twice"})
	rec1 := httptest.NewRecorder()
	h.ReviewDecide(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("first decide: expected 200, got %d: %s", rec1.Code, rec1.Body.String())
	}

	body2, _ := json.Marshal(ReviewDecideRequest{Decision: ReviewDecisionFail})
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/twice/review-decide", bytes.NewReader(body2))
	req2 = mux.SetURLVars(req2, map[string]string{"kind": "execute", "name": "twice"})
	rec2 := httptest.NewRecorder()
	h.ReviewDecide(rec2, req2)
	if rec2.Code == http.StatusOK {
		t.Fatalf("second decide unexpectedly succeeded: %s", rec2.Body.String())
	}
	if !strings.Contains(rec2.Body.String(), "review_pending") {
		t.Errorf("expected error to cite review_pending requirement, got: %s", rec2.Body.String())
	}
}

// TestReviewDecide_InvalidDecision verifies that an unknown decision string
// (anything other than accept|fail|followup) is rejected with 400.
func TestReviewDecide_InvalidDecision(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindExecute, BacklogItem{
		Name:   "bogus-decision",
		Status: StatusReviewPending,
		Kind:   KindExecute,
	})

	body := []byte(`{"decision":"maybe"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/bogus-decision/review-decide", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "bogus-decision"})
	rec := httptest.NewRecorder()
	h.ReviewDecide(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	// Item must remain in review_pending — a rejected request should not
	// mutate the item.
	updated, _ := h.store.LoadItem(KindExecute, "bogus-decision")
	if updated.Status != StatusReviewPending {
		t.Errorf("status changed despite rejection: %q", updated.Status)
	}
}

// TestReviewDecide_MissingRationaleOK verifies rationale is optional: a
// request without rationale succeeds (defaults to empty) and is not an error.
func TestReviewDecide_MissingRationaleOK(t *testing.T) {
	h, rootDir := setupTestHandler(t)
	createTestItem(t, rootDir, KindExecute, BacklogItem{
		Name:   "no-rationale",
		Status: StatusReviewPending,
		Kind:   KindExecute,
	})

	body := []byte(`{"decision":"accept"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/no-rationale/review-decide", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "no-rationale"})
	rec := httptest.NewRecorder()
	h.ReviewDecide(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with empty rationale, got %d: %s", rec.Code, rec.Body.String())
	}

	decisionsDir := filepath.Join(rootDir, backlogKindDirs[KindExecute], "no-rationale", "review", "decisions")
	entries, err := os.ReadDir(decisionsDir)
	if err != nil {
		t.Fatalf("read decisions dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 decision file, got %d", len(entries))
	}
}

// [REQ:SWM-P0-009] A persisted recovery instruction is executable, not a
// terminal status dead-end. Each disposition consumes the instruction.
func TestDispatchFollowUpDispositions(t *testing.T) {
	tests := []struct {
		name       string
		followUp   *FollowUp
		wantStatus BacklogStatus
		wantChild  bool
	}{
		{name: "replan", followUp: &FollowUp{Steering: "Revise the plan.", Disposition: FollowUpReplan}, wantStatus: StatusBacklog},
		{name: "new items", followUp: &FollowUp{Steering: "Split the work.", Disposition: FollowUpNewItems, Items: []followup.ItemSpec{{Kind: "execute", Name: "child", Title: "Child"}}}, wantStatus: StatusCompleted, wantChild: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, root := setupTestHandler(t)
			lifecycle, err := NewService(ServiceConfig{Store: h.store})
			if err != nil {
				t.Fatal(err)
			}
			h.SetLifecycleService(lifecycle)
			item := BacklogItem{Name: "parent", Title: "Parent", Kind: KindExecute, Status: StatusNeedsFollowup, PlanAcceptance: &PlanAcceptance{}, PendingFollowUp: tt.followUp}
			createTestItem(t, root, item.Kind, item)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/parent/follow-up/dispatch", nil)
			req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "parent"})
			rec := httptest.NewRecorder()
			h.DispatchFollowUp(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
			}
			updated, err := h.store.LoadItem(KindExecute, "parent")
			if err != nil {
				t.Fatal(err)
			}
			if updated.Status != tt.wantStatus || updated.PendingFollowUp != nil {
				t.Fatalf("updated = %#v", updated)
			}
			if tt.name == "replan" && (updated.PlanAcceptance != nil || !strings.Contains(updated.Note, "Revise the plan.")) {
				t.Fatalf("replan did not clear acceptance and preserve steering: %#v", updated)
			}
			if tt.wantChild {
				if _, err := h.store.LoadItem(KindExecute, "child"); err != nil {
					t.Fatalf("follow-up child not created: %v", err)
				}
			}
		})
	}
}

func TestDispatchFollowUpRunUsesExecutionBridge(t *testing.T) {
	h, root := setupTestHandler(t)
	dispatcher := &recordingFollowUpDispatcher{}
	h.SetFollowUpDispatcher(dispatcher)
	item := BacklogItem{Name: "parent", Title: "Parent", Kind: KindExecute, Status: StatusNeedsFollowup, PendingFollowUp: &FollowUp{Steering: "Address the failing check.", Disposition: FollowUpRun}}
	createTestItem(t, root, item.Kind, item)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/parent/follow-up/dispatch", nil)
	req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "parent"})
	rec := httptest.NewRecorder()
	h.DispatchFollowUp(rec, req)
	if rec.Code != http.StatusOK || !dispatcher.called || dispatcher.steering != "Address the failing check." {
		t.Fatalf("status=%d dispatcher=%#v", rec.Code, dispatcher)
	}
	updated, err := h.store.LoadItem(KindExecute, "parent")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != StatusQueued || updated.PendingFollowUp != nil {
		t.Fatalf("updated = %#v", updated)
	}
}

func TestAuthorFollowUpConvertsLegacyItemToTypedInstruction(t *testing.T) {
	h, root := setupTestHandler(t)
	item := BacklogItem{Name: "legacy", Title: "Legacy", Kind: KindExecute, Status: StatusNeedsFollowup}
	createTestItem(t, root, item.Kind, item)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/execute/legacy/follow-up/author", bytes.NewBufferString(`{"follow_up":{"steering":"Replan around the evidence.","disposition":"replan"}}`))
	req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "legacy"})
	rec := httptest.NewRecorder()
	h.AuthorFollowUp(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	updated, err := h.store.LoadItem(KindExecute, "legacy")
	if err != nil {
		t.Fatal(err)
	}
	if updated.PendingFollowUp == nil || updated.PendingFollowUp.Steering != "Replan around the evidence." || updated.PendingFollowUp.Disposition != FollowUpReplan {
		t.Fatalf("pending follow-up = %#v", updated.PendingFollowUp)
	}
}
