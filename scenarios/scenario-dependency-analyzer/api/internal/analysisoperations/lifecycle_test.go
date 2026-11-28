package analysisoperations

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestOperationStateTransitionGraph(t *testing.T) {
	allowed := map[State][]State{
		Queued:  {Running},
		Running: {Succeeded, Partial, Failed, Interrupted},
	}
	states := []State{Queued, Running, Succeeded, Partial, Failed, Interrupted, "corrupt"}
	for _, from := range states {
		wantValid := from != "corrupt"
		if from.Valid() != wantValid {
			t.Errorf("Valid(%q) = %v, want %v", from, from.Valid(), wantValid)
		}
		wantTerminal := wantValid && len(allowed[from]) == 0
		if from.Terminal() != wantTerminal {
			t.Errorf("Terminal(%q) = %v, want %v", from, from.Terminal(), wantTerminal)
		}
		for _, to := range states {
			want := false
			for _, candidate := range allowed[from] {
				want = want || candidate == to
			}
			if got := from.CanTransitionTo(to); got != want {
				t.Errorf("CanTransitionTo(%q, %q) = %v, want %v", from, to, got, want)
			}
		}
	}
}

func TestFinishRejectsInvalidOperationEdgesAndPreservesRunningReceipt(t *testing.T) {
	ctx := context.Background()
	repo := testRepository(t)
	op, _, err := repo.Submit(ctx, "lifecycle", Spec{Kind: FleetAnalysis}, []string{"alpha"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ClaimNext(ctx, time.Now()); err != nil {
		t.Fatal(err)
	}

	for _, invalid := range []State{Queued, Running, State("corrupt")} {
		err := repo.Finish(ctx, op.ID, invalid, "", nil, time.Now())
		if err == nil {
			t.Errorf("Finish accepted invalid edge running -> %q", invalid)
			continue
		}
		current, getErr := repo.Get(ctx, op.ID)
		if getErr != nil {
			t.Fatal(getErr)
		}
		if current.State != Running {
			t.Fatalf("invalid edge %q changed persisted state to %q", invalid, current.State)
		}
	}

	if err := repo.Finish(ctx, op.ID, Failed, "expected failure", nil, time.Now()); err != nil {
		t.Fatalf("Finish valid terminal edge: %v", err)
	}
	current, err := repo.Get(ctx, op.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.State != Failed {
		t.Fatalf("persisted state = %q, want %q", current.State, Failed)
	}
}

func TestGetRejectsUnknownPersistedOperationState(t *testing.T) {
	ctx := context.Background()
	repo := testRepository(t)
	op, _, err := repo.Submit(ctx, "unknown-state", Spec{Kind: FleetAnalysis}, []string{}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.db.ExecContext(ctx, `UPDATE analysis_operations SET state=? WHERE id=?`, State("corrupt"), op.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get(ctx, op.ID); err == nil || !strings.Contains(err.Error(), "unknown persisted analysis operation state") {
		t.Fatalf("Get unknown state error = %v", err)
	}
}

func TestRecoverReturnsOnlyOperationsItTransitions(t *testing.T) {
	ctx := context.Background()
	repo := testRepository(t)
	now := time.Now()
	running, _, err := repo.Submit(ctx, "running-before-recovery", Spec{Kind: FleetAnalysis}, []string{"alpha"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ClaimNext(ctx, now); err != nil {
		t.Fatal(err)
	}
	queued, _, err := repo.Submit(ctx, "queued-during-recovery", Spec{Kind: GraphRebuild}, []string{"beta"}, now)
	if err != nil {
		t.Fatal(err)
	}

	interrupted, err := repo.Recover(ctx, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if len(interrupted) != 1 || interrupted[0] != running.ID {
		t.Fatalf("recovery returned %v, want only %s", interrupted, running.ID)
	}
	current, err := repo.Get(ctx, queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.State != Queued {
		t.Fatalf("queued receipt changed to %q", current.State)
	}
}
