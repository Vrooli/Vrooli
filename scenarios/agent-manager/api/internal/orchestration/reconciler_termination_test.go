package orchestration

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"github.com/google/uuid"
)

func TestReconcilerTerminationUsesNativeGraceAndRetainsCleanupFailures(t *testing.T) {
	gracefulFailure := errors.New("native graceful request unavailable")
	groupFailure := errors.New("group cleanup failed")
	pidFailure := errors.New("leader cleanup failed")
	groupUnknown := errors.New("group absence is unconfirmed")
	for _, tc := range []struct {
		name                          string
		group                         int
		alive                         bool
		groupGone                     bool
		gracefulErr, groupErr, pidErr error
		wantEvents                    []string
		wantErrors                    []error
	}{
		{name: "graceful exit still sweeps children", group: 41, wantEvents: []string{"group", "graceful", "wait", "alive", "kill-group"}},
		{name: "live leader escalates after grace", group: 41, alive: true, wantEvents: []string{"group", "graceful", "wait", "alive", "kill-group"}},
		{name: "graceful error with live leader keeps grace", group: 41, alive: true, gracefulErr: gracefulFailure, wantEvents: []string{"group", "graceful", "alive", "wait", "alive", "kill-group"}},
		{name: "already gone sweeps children", group: 41, gracefulErr: gracefulFailure, wantEvents: []string{"group", "graceful", "alive", "alive", "kill-group"}},
		{name: "no group falls back to leader", alive: true, wantEvents: []string{"group", "graceful", "wait", "alive", "kill-pid"}},
		{name: "normal graceful group exit is successful", group: 41, groupErr: groupFailure, groupGone: true, wantEvents: []string{"group", "graceful", "wait", "alive", "kill-group", "confirm-group-exit"}},
		{name: "failed sweep remains visible after leader fallback", group: 41, alive: true, groupErr: groupFailure, wantEvents: []string{"group", "graceful", "wait", "alive", "kill-group", "confirm-group-exit", "kill-pid"}, wantErrors: []error{groupFailure, groupUnknown}},
		{name: "failed sweep after leader exit remains visible", group: 41, groupErr: groupFailure, wantEvents: []string{"group", "graceful", "wait", "alive", "kill-group", "confirm-group-exit"}, wantErrors: []error{groupFailure, groupUnknown}},
		{name: "all errors retained", group: 41, alive: true, gracefulErr: gracefulFailure, groupErr: groupFailure, pidErr: pidFailure, wantEvents: []string{"group", "graceful", "alive", "wait", "alive", "kill-group", "confirm-group-exit", "kill-pid"}, wantErrors: []error{gracefulFailure, groupFailure, groupUnknown, pidFailure}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var events []string
			err := terminateReconcilerProcess(42, processTerminationEffects{
				groupID: func(pid int) int {
					if pid != 42 {
						t.Fatalf("unexpected PID %d", pid)
					}
					events = append(events, "group")
					return tc.group
				},
				graceful: func() error { events = append(events, "graceful"); return tc.gracefulErr },
				alive:    func(int) bool { events = append(events, "alive"); return tc.alive },
				wait: func(d time.Duration) {
					if d != 500*time.Millisecond {
						t.Fatalf("grace = %v", d)
					}
					events = append(events, "wait")
				},
				killGroup: func(group int) error {
					if group != 41 {
						t.Fatalf("unexpected group %d", group)
					}
					events = append(events, "kill-group")
					return tc.groupErr
				},
				confirmGroupExit: func(group int) error {
					if group != 41 {
						t.Fatalf("unexpected confirmation group %d", group)
					}
					events = append(events, "confirm-group-exit")
					if tc.groupGone {
						return nil
					}
					return groupUnknown
				},
				killPID: func() error { events = append(events, "kill-pid"); return tc.pidErr },
			})
			if !reflect.DeepEqual(events, tc.wantEvents) {
				t.Fatalf("events %v, want %v", events, tc.wantEvents)
			}
			if len(tc.wantErrors) == 0 && err != nil {
				t.Fatalf("termination failed: %v", err)
			}
			for _, expected := range tc.wantErrors {
				if !errors.Is(err, expected) {
					t.Fatalf("error %v lost %v", err, expected)
				}
			}
		})
	}
}

func TestAttachedReconcilePreservesLiveIdentityAndCompletesOnlyAbsentExecutor(t *testing.T) {
	for _, byToken := range []bool{false, true} {
		for _, alive := range []bool{false, true} {
			name := "pid"
			if byToken {
				name = "token-hash"
			}
			if alive {
				name += "/live"
			} else {
				name += "/absent"
			}
			t.Run(name, func(t *testing.T) {
				ctx := context.Background()
				repos, _, cleanup := testutil.SetupTestRepos(t)
				t.Cleanup(cleanup)
				now := time.Now().UTC()
				run := &domain.Run{ID: uuid.New(), Tag: uuid.NewString(), RunMode: domain.RunModeInPlace, ExecutionMode: domain.ExecutionModeAttached, HarnessKind: "fixture", HarnessSessionID: "attached-fixture", Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, IdentityTokenHash: "fixture-identity-hash"}
				if !byToken {
					run.RunnerPID = 4242
				}
				if err := repos.Runs.Create(ctx, run); err != nil {
					t.Fatal(err)
				}
				effects := &controlledProcessEffects{}
				effects.setLiveness(4242, alive, run.IdentityTokenHash, alive)
				r := NewReconciler(repos.Runs, nil, withReconcilerProcessEffects(effects), WithReconcilerClock(func() time.Time { return now.Add(2 * time.Minute) }), WithReconcilerRunStateRoot(t.TempDir()))
				r.RunOnce(ctx)
				stored, err := repos.Runs.Get(ctx, run.ID)
				if err != nil {
					t.Fatal(err)
				}
				if alive {
					if stored.Status != domain.RunStatusRunning || stored.IdentityTokenRevokedAt != nil || stored.EndedAt != nil {
						t.Fatalf("live attached executor changed: status=%s revoked=%v ended=%v", stored.Status, stored.IdentityTokenRevokedAt, stored.EndedAt)
					}
				} else if stored.Status != domain.RunStatusComplete || stored.IdentityTokenRevokedAt == nil || stored.EndedAt == nil {
					t.Fatalf("absent executor not completed/revoked: status=%s revoked=%v ended=%v", stored.Status, stored.IdentityTokenRevokedAt, stored.EndedAt)
				}
				if killed, cleanups := effects.snapshotEffects(); len(killed) != 0 || cleanups != 0 {
					t.Fatalf("attached observation caused cleanup: %v %d", killed, cleanups)
				}
			})
		}
	}
}
