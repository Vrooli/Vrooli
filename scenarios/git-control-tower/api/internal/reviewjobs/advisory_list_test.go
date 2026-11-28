package reviewjobs

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestListRecoverableAdvisoryFiltersCancelledRowsBeforePaging(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	created := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	jobs := []Job{
		{ID: "00-cancelled-queued", State: Queued, CancelRequested: true},
		{ID: "01-queued", State: Queued},
		{ID: "02-dispatching", State: Dispatching},
		{ID: "03-running", State: Running, ExecutionID: "execution-3"},
		{ID: "04-cancelling", State: Cancelling, CancelRequested: true},
		{ID: "05-other-operation", State: Queued, Operation: "other"},
		{ID: "06-terminal", State: Succeeded, ExecutionID: "execution-6"},
	}
	for i := range jobs {
		jobs[i].IdempotencyKey = "key-" + jobs[i].ID
		jobs[i].InputDigest = "digest-" + jobs[i].ID
		jobs[i].PolicyVersion = "policy-v1"
		if jobs[i].Operation == "" {
			jobs[i].Operation = "staged-message"
		}
		jobs[i].RepositoryID = "repo-a"
		jobs[i].CreatedAt = created.Add(time.Duration(i) * time.Second)
		if _, _, err := s.Create(ctx, jobs[i]); err != nil {
			t.Fatalf("create %s: %v", jobs[i].ID, err)
		}
	}

	states := []State{Queued, Dispatching, Running, Cancelling}
	first, err := s.ListRecoverableAdvisory(ctx, "staged-message", states, 2, "")
	if err != nil {
		t.Fatalf("first recoverable page: %v", err)
	}
	if got := advisoryIDs(first); !equalAdvisoryIDs(got, []string{"01-queued", "02-dispatching"}) {
		t.Fatalf("first recoverable page = %v; cancelled work must not starve active rows", got)
	}
	second, err := s.ListRecoverableAdvisory(ctx, "staged-message", states, 2, "02-dispatching")
	if err != nil {
		t.Fatalf("second recoverable page: %v", err)
	}
	if got := advisoryIDs(second); !equalAdvisoryIDs(got, []string{"03-running", "04-cancelling"}) {
		t.Fatalf("second recoverable page = %v", got)
	}
	if _, err := s.ListRecoverableAdvisory(ctx, "staged-message", states, 0, ""); err == nil {
		t.Fatal("zero limit must be rejected")
	}
}

func TestListAdvisoryUsesRecentFirstAndValidatesStoredState(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	base := time.Date(2026, 11, 13, 13, 0, 0, 0, time.UTC)
	for i, id := range []string{"older", "newer", "other-operation"} {
		operation := "staged-message"
		if id == "other-operation" {
			operation = "other"
		}
		if _, _, err := s.Create(ctx, Job{
			ID: id, IdempotencyKey: "key-" + id, InputDigest: "digest-" + id,
			PolicyVersion: "policy-v1", Operation: operation, RepositoryID: "repo-a",
			State: Queued, ResultJSON: "result-" + id, CreatedAt: base.Add(time.Duration(i) * time.Minute),
		}); err != nil {
			t.Fatalf("create %s: %v", id, err)
		}
	}
	rows, err := s.ListAdvisory(ctx, "staged-message", []State{Queued}, 1)
	if err != nil {
		t.Fatalf("list recent advisory: %v", err)
	}
	if got := advisoryIDs(rows); !equalAdvisoryIDs(got, []string{"newer"}) {
		t.Fatalf("recent advisory page = %v", got)
	}
	if rows[0].ResultJSON != "result-newer" || !rows[0].UpdatedAt.Equal(base.Add(time.Minute)) {
		t.Fatalf("advisory projection lost result or update time: %+v", rows[0])
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE gct_review_jobs SET state=? WHERE id=?`, "corrupt-state", "older"); err != nil {
		t.Fatalf("corrupt stored state fixture: %v", err)
	}
	if _, err := s.ListAdvisory(ctx, "staged-message", []State{State("corrupt-state")}, 1); err == nil || !strings.Contains(err.Error(), "unknown state") {
		t.Fatalf("unknown stored state error = %v, want validation error", err)
	}
	if _, err := s.ListAdvisory(ctx, " ", []State{Queued}, 1); err == nil {
		t.Fatal("empty operation must be rejected")
	}
	if _, err := s.ListAdvisory(ctx, "staged-message", nil, 1); err == nil {
		t.Fatal("empty states must be rejected")
	}
}

func advisoryIDs(rows []AdvisorySummary) []string {
	ids := make([]string, len(rows))
	for i := range rows {
		ids[i] = rows[i].ID
	}
	return ids
}

func equalAdvisoryIDs(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
