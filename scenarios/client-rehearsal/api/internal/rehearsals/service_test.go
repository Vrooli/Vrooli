package rehearsals

import (
	"context"
	"testing"
	"time"
)

type memoryRepository struct{ items map[string]Rehearsal }

func newMemoryRepository() *memoryRepository { return &memoryRepository{items: map[string]Rehearsal{}} }
func (r *memoryRepository) Create(_ context.Context, item Rehearsal) (Rehearsal, error) {
	r.items[item.ID] = item
	return item, nil
}

func (r *memoryRepository) Get(_ context.Context, id string) (Rehearsal, error) {
	item, ok := r.items[id]
	if !ok {
		return Rehearsal{}, ErrNotFound{ID: id}
	}
	return item, nil
}

func (r *memoryRepository) List(_ context.Context, limit int) ([]Rehearsal, error) {
	out := make([]Rehearsal, 0, len(r.items))
	for _, item := range r.items {
		out = append(out, item)
	}
	return out, nil
}

func (r *memoryRepository) RecordReview(_ context.Context, id string, d Decision, note string, at time.Time) (Rehearsal, error) {
	item, ok := r.items[id]
	if !ok {
		return Rehearsal{}, ErrNotFound{ID: id}
	}
	if item.State != StateNeedsReview {
		return Rehearsal{}, ErrConflict{Reason: "review already recorded"}
	}
	item.State = stateForDecision(d)
	item.ReviewDecision = d
	item.ReviewNote = note
	item.ReviewedAt = &at
	r.items[id] = item
	return item, nil
}

// [REQ:CR-P0-001]
func TestCreateAndRunKeepsReviewHumanAndCapturesRecovery(t *testing.T) {
	repo := newMemoryRepository()
	fixed := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	svc := NewService(repo, SyntheticEngine{}, func() time.Time { return fixed })
	workflow := `{"name":"synthetic contact intake","nodes":[{"name":"Manual","type":"n8n-nodes-base.manualTrigger","parameters":{}},{"name":"Create contact","type":"n8n-nodes-base.httpRequest","parameters":{"method":"POST","url":"https://crm.synthetic.invalid/v1/contacts"}}],"connections":{"Manual":{"main":[[{"node":"Create contact"}]]}}}`
	got, err := svc.CreateAndRun(context.Background(), CreateInput{Title: "Synthetic contact intake", WorkflowJSON: workflow, InjectSeededFault: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.State != StateNeedsReview {
		t.Fatalf("run state = %s", got.State)
	}
	if got.ReviewDecision != "" || got.ReviewedAt != nil {
		t.Fatal("successful execution must not imply an operator review")
	}
	if len(got.Trace) < 3 || !traceHasStatus(got.Trace, "failed") || !traceHasStatus(got.Trace, "succeeded") {
		t.Fatalf("fault/repair trace = %#v", got.Trace)
	}
	if len(got.HandoffMarkdown) == 0 {
		t.Fatal("expected handoff snapshot")
	}
	reviewed, err := svc.Review(context.Background(), ReviewInput{ID: got.ID, Decision: DecisionApprove, Note: "Synthetic evidence reviewed"})
	if err != nil {
		t.Fatal(err)
	}
	if reviewed.State != StateApproved || reviewed.ReviewedAt == nil {
		t.Fatalf("review result = %#v", reviewed)
	}
}

func traceHasStatus(trace []TraceEvent, status string) bool {
	for _, event := range trace {
		if event.Status == status {
			return true
		}
	}
	return false
}

func TestCreateAndRunRequiresSelectedTriggerForMultiTriggerWorkflow(t *testing.T) {
	workflow := `{"id":"multi-trigger","nodes":[{"id":"manual","name":"Manual","type":"n8n-nodes-base.manualTrigger","parameters":{}},{"id":"scheduled","name":"Schedule","type":"n8n-nodes-base.scheduleTrigger","parameters":{"rule":{"interval":[]}}},{"id":"finish","name":"Finish","type":"n8n-nodes-base.noOp","parameters":{}}],"connections":{"Manual":{"main":[[{"node":"Finish"}]]},"Schedule":{"main":[[{"node":"Finish"}]]}}}`
	svc := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return time.Unix(1, 0) })
	if _, err := svc.CreateAndRun(context.Background(), CreateInput{Title: "Multi trigger", WorkflowJSON: workflow}); err == nil {
		t.Fatal("service must reject ambiguous trigger selection")
	}
	got, err := svc.CreateAndRun(context.Background(), CreateInput{Title: "Multi trigger", WorkflowJSON: workflow, TriggerNodeID: "scheduled"})
	if err != nil {
		t.Fatal(err)
	}
	if !traceHasOperation(got.Trace, "scheduled Trigger") || traceHasOperation(got.Trace, "manual Trigger") {
		t.Fatalf("unexpected selected trigger trace: %#v", got.Trace)
	}
}
