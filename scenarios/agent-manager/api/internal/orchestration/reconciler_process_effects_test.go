package orchestration

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/repository"
)

type controlledProcessEffects struct {
	mu        sync.RWMutex
	processes []discoveredRunnerProcess
	runner    string
	tags      map[int]string
	alive     map[int]bool
	attached  map[string]bool
	started   time.Time
	killed    []int
	cleanups  int
}

func (f *controlledProcessEffects) DiscoverRunnerProcesses(_ context.Context, runner string) ([]discoveredRunnerProcess, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if runner != f.runner {
		return nil, nil
	}
	return append([]discoveredRunnerProcess(nil), f.processes...), nil
}

func installTestProcess(rec *Reconciler, runner string, pid int, tag, command string) *controlledProcessEffects {
	effects := &controlledProcessEffects{
		runner:    runner,
		processes: []discoveredRunnerProcess{{PID: pid, Command: command}},
		tags:      map[int]string{pid: tag},
		started:   time.Now().Add(-2 * time.Hour),
	}
	rec.processEffects = effects
	return effects
}
func (f *controlledProcessEffects) ProcessStartTime(int) time.Time {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.started
}
func (f *controlledProcessEffects) ProcessTag(pid int) string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.tags[pid]
}
func (f *controlledProcessEffects) ProcessAlive(pid int) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.alive[pid]
}
func (f *controlledProcessEffects) AttachedIdentityAlive(hash string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.attached[hash]
}
func (f *controlledProcessEffects) KillProcess(pid int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.killed = append(f.killed, pid)
	return nil
}
func (f *controlledProcessEffects) CleanupRunnerResources(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cleanups++
	return nil
}

func (f *controlledProcessEffects) setSnapshot(processes []discoveredRunnerProcess, tags map[int]string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.processes = append([]discoveredRunnerProcess(nil), processes...)
	f.tags = make(map[int]string, len(tags))
	for pid, tag := range tags {
		f.tags[pid] = tag
	}
}

func (f *controlledProcessEffects) setLiveness(pid int, alive bool, hash string, attached bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.alive == nil {
		f.alive = map[int]bool{}
	}
	if f.attached == nil {
		f.attached = map[string]bool{}
	}
	f.alive[pid] = alive
	f.attached[hash] = attached
}

func (f *controlledProcessEffects) snapshotEffects() ([]int, int) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return append([]int(nil), f.killed...), f.cleanups
}

func TestLegacyRunnerLivenessDoesNotAuthorizeOrphanCleanup(t *testing.T) {
	const tag = "00000000-0000-4000-8000-000000000099"
	for _, tc := range []struct {
		name, command, environmentTag, owner string
		wantAlive                            bool
	}{
		{"legacy command tag", "codex --tag " + tag, "", "", true},
		{"legacy environment tag", "codex", tag, "", true},
		{"different command tag", "codex --tag other", "", "", false},
		{"different environment tag", "codex", "other", "", false},
		{"foreign store", "codex --tag " + tag, "", "store-b", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			effects := &controlledProcessEffects{runner: "codex", started: time.Now().Add(-2 * time.Hour)}
			effects.setSnapshot([]discoveredRunnerProcess{{PID: 101, Command: tc.command, OwnerID: tc.owner}}, map[int]string{101: tc.environmentTag})
			r := NewReconciler(nil, nil, withReconcilerProcessEffects(effects),
				WithReconcilerConfig(ReconcilerConfig{StoreOwnerID: "store-a", KillOrphans: true, OrphanGracePeriod: time.Hour}))
			if got := r.scanRunnerProcessByTag("codex", tag); got != tc.wantAlive {
				t.Fatalf("observed liveness = %v, want %v", got, tc.wantAlive)
			}
			if orphans := r.detectOrphanProcesses(context.Background(), nil); len(orphans) != 0 {
				t.Fatalf("legacy or foreign process became a cleanup candidate: %v", orphans)
			}
			if killed, cleanups := effects.snapshotEffects(); len(killed) != 0 || cleanups != 0 {
				t.Fatalf("observation caused host effects: kills %v cleanups %d", killed, cleanups)
			}
		})
	}
}

func TestFullReconcileUsesOnlyInjectedProcessEffects(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	now := time.Now()
	effects := &controlledProcessEffects{
		runner:    "codex",
		processes: []discoveredRunnerProcess{{PID: 101, Command: "codex --tag 00000000-0000-4000-8000-000000000099", OwnerID: "store-a"}},
		started:   now.Add(-2 * time.Hour), tags: map[int]string{},
	}
	reconciler := NewReconciler(repos.Runs, nil,
		withReconcilerProcessEffects(effects),
		WithReconcilerClock(func() time.Time { return now }),
		WithReconcilerRunStateRoot(t.TempDir()),
		WithReconcilerConfig(ReconcilerConfig{KillOrphans: true, OrphanGracePeriod: time.Hour, StoreOwnerID: "store-a"}),
	)
	stats := reconciler.RunOnce(context.Background())
	if stats.OrphansFound != 1 || stats.OrphansKilled != 1 {
		t.Fatalf("synthetic orphan reconcile stats = found %d killed %d errors %v", stats.OrphansFound, stats.OrphansKilled, stats.Errors)
	}
	killed, cleanups := effects.snapshotEffects()
	if len(killed) != 1 || killed[0] != 101 || cleanups != 1 {
		t.Fatalf("synthetic effects = killed %v cleanups %d", killed, cleanups)
	}
}

func TestFullReconcileOrphanCounterexamples(t *testing.T) {
	for _, tc := range []struct {
		name      string
		known     map[string]*domain.Run
		started   time.Time
		kill      bool
		wantFound int
		wantKill  int
		ownerID   string
		storeID   string
	}{
		{name: "known tag", known: map[string]*domain.Run{"00000000-0000-4000-8000-000000000099": {}}, started: time.Now().Add(-2 * time.Hour), kill: true, ownerID: "store-a", storeID: "store-a"},
		{name: "young process", started: time.Now(), kill: true, ownerID: "store-a", storeID: "store-a"},
		{name: "environment tag", started: time.Now().Add(-2 * time.Hour), kill: true, wantFound: 1, wantKill: 1, ownerID: "store-a", storeID: "store-a"},
		{name: "killing disabled", started: time.Now().Add(-2 * time.Hour), kill: false, wantFound: 1, ownerID: "store-a", storeID: "store-a"},
		{name: "foreign store", started: time.Now().Add(-2 * time.Hour), kill: true, ownerID: "store-b", storeID: "store-a"},
		{name: "missing owner evidence", started: time.Now().Add(-2 * time.Hour), kill: true, storeID: "store-a"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repos, _, cleanup := testutil.SetupTestRepos(t)
			t.Cleanup(cleanup)
			now := time.Now()
			effects := &controlledProcessEffects{runner: "codex", started: tc.started}
			effects.setSnapshot([]discoveredRunnerProcess{{PID: 101, Command: "codex", OwnerID: tc.ownerID}}, map[int]string{101: "00000000-0000-4000-8000-000000000099"})
			r := NewReconciler(repos.Runs, nil, withReconcilerProcessEffects(effects), WithReconcilerClock(func() time.Time { return now }), WithReconcilerRunStateRoot(t.TempDir()), WithReconcilerConfig(ReconcilerConfig{KillOrphans: tc.kill, OrphanGracePeriod: time.Hour, StaleThreshold: time.Hour, PendingThreshold: time.Hour, StoreOwnerID: tc.storeID}))
			if len(tc.known) > 0 {
				run := createRecoveryTestRun(t, repos, domain.RunnerTypeCodex)
				run.Tag = "00000000-0000-4000-8000-000000000099"
				if err := repos.Runs.Update(context.Background(), run); err != nil {
					t.Fatal(err)
				}
				stats := r.RunOnce(context.Background())
				killed, cleanups := effects.snapshotEffects()
				if stats.OrphansFound != 0 || len(killed) != 0 || cleanups != 0 {
					t.Fatalf("persisted live tag was reaped: stats=%+v killed=%v cleanups=%d", stats, killed, cleanups)
				}
				return
			}
			stats := r.RunOnce(context.Background())
			killed, cleanups := effects.snapshotEffects()
			wantCleanup := tc.wantKill
			if stats.OrphansFound != tc.wantFound || len(killed) != tc.wantKill || cleanups != wantCleanup {
				t.Fatalf("found=%d killed=%v cleanups=%d, want found=%d killed=%d cleanups=%d", stats.OrphansFound, killed, cleanups, tc.wantFound, tc.wantKill, wantCleanup)
			}
		})
	}
}

func TestFullReconcileDefaultsToInertHostEffects(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	r := NewReconciler(repos.Runs, nil,
		WithReconcilerConfig(ReconcilerConfig{KillOrphans: true, OrphanGracePeriod: time.Nanosecond}),
		WithReconcilerRunStateRoot(t.TempDir()),
	)
	if _, ok := r.processEffects.(inertReconcilerProcessEffects); !ok {
		t.Fatalf("unit reconciler process effects = %T, want inert", r.processEffects)
	}
	stats := r.RunOnce(context.Background())
	if stats.OrphansFound != 0 || stats.OrphansKilled != 0 {
		t.Fatalf("inert reconcile touched host process inventory: %+v", stats)
	}
}

func TestHostProcessEffectsRecognizeCurrentPID(t *testing.T) {
	effects := hostReconcilerProcessEffects{}
	if !effects.ProcessAlive(os.Getpid()) {
		t.Fatal("host liveness adapter rejected the current live process")
	}
	if effects.ProcessAlive(-1) {
		t.Fatal("host liveness adapter accepted an invalid PID")
	}
}

func TestRetainedInteractiveExecutorIsNotReapedFromStoredStatus(t *testing.T) {
	for _, status := range []domain.RunStatus{domain.RunStatusParked, domain.RunStatusComplete, domain.RunStatusFailed, domain.RunStatusNeedsReview, domain.RunStatusCancelled} {
		t.Run(string(status), func(t *testing.T) {
			repos, _, cleanup := testutil.SetupTestRepos(t)
			t.Cleanup(cleanup)
			svc := New(repos.Profiles, repos.Tasks, repos.Runs)
			task := interactiveTestTask(t, svc)
			run := persistInteractiveRun(t, repos.Runs, task.ID, status, "retained-thread", "retained-web-session")
			effects := &controlledProcessEffects{runner: "codex", started: time.Now().Add(-2 * time.Hour)}
			effects.setSnapshot([]discoveredRunnerProcess{{PID: 101, Command: "codex --tag " + run.GetTag(), OwnerID: "store-a"}}, nil)
			r := NewReconciler(repos.Runs, nil, withReconcilerProcessEffects(effects), WithReconcilerInteractive(newRecordingSessions()), WithReconcilerConfig(ReconcilerConfig{StoreOwnerID: "store-a", KillOrphans: true, OrphanGracePeriod: time.Hour}))
			orphans := r.detectOrphanProcesses(t.Context(), nil)
			want := 0
			if status == domain.RunStatusCancelled {
				want = 1
			}
			if len(orphans) != want {
				t.Fatalf("stored %s yielded %d orphan candidates, want %d", status, len(orphans), want)
			}
			if killed, cleanups := effects.snapshotEffects(); len(killed) != 0 || cleanups != 0 {
				t.Fatal("classification caused process effects")
			}
		})
	}
}

func TestHistoricalInteractiveBindingRequiresCurrentSessionRetention(t *testing.T) {
	for _, unavailable := range []bool{false, true} {
		t.Run(map[bool]string{false: "positively retired", true: "lookup unknown"}[unavailable], func(t *testing.T) {
			repos, _, cleanup := testutil.SetupTestRepos(t)
			t.Cleanup(cleanup)
			svc := New(repos.Profiles, repos.Tasks, repos.Runs)
			task := interactiveTestTask(t, svc)
			run := persistInteractiveRun(t, repos.Runs, task.ID, domain.RunStatusComplete, "retained-thread", "historical-web-session")
			sessions := newRecordingSessions()
			if unavailable {
				sessions.getErr = errors.New("session owner unavailable")
			} else {
				sessions.gone[run.WebConsoleSessionID] = true
			}
			effects := &controlledProcessEffects{runner: "codex", started: time.Now().Add(-2 * time.Hour)}
			effects.setSnapshot([]discoveredRunnerProcess{{PID: 101, Command: "codex --tag " + run.GetTag(), OwnerID: "store-a"}}, nil)
			r := NewReconciler(repos.Runs, nil, withReconcilerProcessEffects(effects), WithReconcilerInteractive(sessions), WithReconcilerConfig(ReconcilerConfig{StoreOwnerID: "store-a", KillOrphans: true, OrphanGracePeriod: time.Hour}))
			want := 1
			if unavailable {
				want = 0
			}
			if got := r.detectOrphanProcesses(t.Context(), nil); len(got) != want {
				t.Fatalf("historical binding yielded %d candidates, want %d", len(got), want)
			}
		})
	}
}

type orphanOwnershipLookupUnavailable struct{ repository.RunRepository }

func (orphanOwnershipLookupUnavailable) List(context.Context, repository.RunListFilter) ([]*domain.Run, error) {
	return nil, errors.New("authoritative run inventory unavailable")
}

func TestOrphanClassificationRetainsProcessWhenOwnerInventoryIsUnknown(t *testing.T) {
	effects := &controlledProcessEffects{runner: "codex", started: time.Now().Add(-2 * time.Hour)}
	effects.setSnapshot([]discoveredRunnerProcess{{PID: 101, Command: "codex --tag 00000000-0000-4000-8000-000000000099", OwnerID: "store-a"}}, nil)
	r := NewReconciler(orphanOwnershipLookupUnavailable{}, nil, withReconcilerProcessEffects(effects), WithReconcilerConfig(ReconcilerConfig{StoreOwnerID: "store-a", KillOrphans: true, OrphanGracePeriod: time.Hour}))
	if candidates := r.detectOrphanProcesses(t.Context(), nil); len(candidates) != 0 {
		t.Fatalf("unknown inventory authorized cleanup: %+v", candidates)
	}
	if killed, cleanups := effects.snapshotEffects(); len(killed) != 0 || cleanups != 0 {
		t.Fatal("unknown ownership triggered effects")
	}
}

var _ reconcilerProcessEffects = (*controlledProcessEffects)(nil)
