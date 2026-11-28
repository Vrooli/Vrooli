package jobs_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"device-studio/internal/jobs"
)

func TestRunScopeLifecycleTransitionMatrix(t *testing.T) {
	states := []jobs.RunScopeState{
		jobs.RunScopeActive,
		jobs.RunScopeClosing,
		jobs.RunScopeCancelled,
		jobs.RunScopeCompleted,
	}
	want := map[jobs.RunScopeState]map[jobs.RunScopeState]bool{
		jobs.RunScopeActive: {
			jobs.RunScopeActive: true, jobs.RunScopeClosing: true,
			jobs.RunScopeCancelled: true, jobs.RunScopeCompleted: true,
		},
		jobs.RunScopeClosing: {
			jobs.RunScopeClosing: true, jobs.RunScopeCancelled: true,
		},
		jobs.RunScopeCancelled: {jobs.RunScopeCancelled: true},
		jobs.RunScopeCompleted: {jobs.RunScopeCompleted: true},
	}

	for _, from := range states {
		for _, to := range states {
			t.Run(string(from)+"->"+string(to), func(t *testing.T) {
				scope := jobs.RunScope{State: from}
				err := scope.Transition(to)
				if want[from][to] {
					if err != nil || scope.State != to {
						t.Fatalf("transition state=%s, want=%s err=%v", scope.State, to, err)
					}
					return
				}
				if !errors.Is(err, jobs.ErrInvalidTransition) {
					t.Fatalf("transition %s -> %s error=%v, want invalid transition", from, to, err)
				}
				if scope.State != from {
					t.Fatalf("rejected transition mutated state to %s", scope.State)
				}
			})
		}
	}
}

func TestRunScopeLifecycleRejectsUnknownStates(t *testing.T) {
	db, repo := openRepo(t, filepath.Join(t.TempDir(), "scope-invalid-state.db"))
	store := repo.(jobs.RunScopeRepository)
	ctx := context.Background()
	if err := store.CreateRunScope(ctx, jobs.RunScope{ID: "invalid-create", State: jobs.RunScopeState("unknown")}); !errors.Is(err, jobs.ErrInvalidTransition) {
		t.Fatalf("invalid initial state error=%v", err)
	}

	scopes := jobs.NewRunScopeService(store, func() time.Time { return baseTime })
	policy, _ := json.Marshal(map[string]any{"kind": "test"})
	if err := scopes.Create(ctx, jobs.RunScope{ID: "corrupt-row", OwnerKind: "test", Policy: policy, MaxChildren: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA ignore_check_constraints=ON`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE render_job_scopes SET state=? WHERE id=?`, "unknown", "corrupt-row"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA ignore_check_constraints=OFF`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetRunScope(ctx, "corrupt-row"); err == nil {
		t.Fatal("unknown persisted state was accepted")
	}
	if _, _, err := store.CloseRunScope(ctx, "corrupt-row", 1, baseTime); err == nil {
		t.Fatal("close accepted an unknown persisted state")
	}
	if _, err := store.FinalizeRunScope(ctx, "corrupt-row", jobs.RunScopeCancelled, 1, baseTime); err == nil {
		t.Fatal("finalize accepted an unknown persisted state")
	}
	var state string
	if err := db.QueryRowContext(ctx, `SELECT state FROM render_job_scopes WHERE id=?`, "corrupt-row").Scan(&state); err != nil || state != "unknown" {
		t.Fatalf("fail-closed reads changed persisted state: state=%q err=%v", state, err)
	}
}
