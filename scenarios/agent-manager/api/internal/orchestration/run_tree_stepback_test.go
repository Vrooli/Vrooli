package orchestration

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/stepback"

	"github.com/google/uuid"
)

// The run tree and the effort board read step-back triggers from the facts
// Agent Manager already stores: the scheduler's transcript events and the
// lifetimes of the runs below it.
func TestRunTreeShowsStepBackTriggersFromStoredTranscriptFacts(t *testing.T) {
	ctx := context.Background()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	o := New(repos.Profiles, repos.Tasks, repos.Runs, WithEvents(events))
	task := interactiveTestTask(t, o)
	root := persistInteractiveRun(t, repos.Runs, task.ID, domain.RunStatusRunning, "root-session", "wc-root")
	for i := 0; i < 4; i++ {
		child := persistInteractiveRun(t, repos.Runs, task.ID, domain.RunStatusComplete, "", "")
		started, ended := time.Now().Add(-time.Hour), time.Now().Add(-time.Hour+3*time.Minute)
		child.ParentRunID, child.Depth, child.StartedAt, child.EndedAt = &root.ID, 1, &started, &ended
		if err := repos.Runs.Update(ctx, child); err != nil {
			t.Fatal(err)
		}
	}

	// One open turn that started 4 h ago; a compaction omitted the objective
	// at 1 h and the turn kept working.
	turnID := uuid.Must(uuid.NewV7())
	start := time.Now().Add(-4 * time.Hour)
	stamp := func(e *domain.RunEvent, at time.Duration) *domain.RunEvent {
		e.Timestamp = start.Add(at)
		return e
	}
	omitted := domain.NewCompactionEvent(root.ID, "", "auto", "", 0, 0, 0, "")
	omitted.Data.(*domain.CompactionEventData).GoalObjective = domain.CompactionGoalOmitted
	if err := events.Append(ctx, root.ID,
		stamp(domain.NewProviderMessageEvent(root.ID, "assistant", "starting", domain.MessageEventData{TurnID: turnID.String()}), time.Minute),
		stamp(omitted, time.Hour),
		stamp(domain.NewToolCallEvent(root.ID, "exec", "late", nil), 4*time.Hour),
	); err != nil {
		t.Fatal(err)
	}

	tree, err := o.GetRunTree(ctx, root.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := stepback.IDs(tree.Root.StepBack); got != stepback.ObjectiveLost+","+stepback.ShortWorkers {
		t.Fatalf("root step-back = %q, want objective-lost and short-workers", got)
	}
	for _, child := range tree.Root.Children {
		if len(child.StepBack) != 0 {
			t.Fatalf("a child without transcript facts or descendants fired %v", child.StepBack)
		}
	}
	board, err := o.ReadRunStepBack(ctx, []uuid.UUID{root.ID, uuid.New()})
	if err != nil {
		t.Fatal(err)
	}
	if got := stepback.IDs(board[root.ID]); got != stepback.IDs(tree.Root.StepBack) {
		t.Fatalf("board step-back = %q, want the run tree's %q", got, stepback.IDs(tree.Root.StepBack))
	}
	if len(board) != 1 {
		t.Fatalf("an unreadable run must be absent from the board read: %v", board)
	}
}
