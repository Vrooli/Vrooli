package recycler

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ecosystem-manager/api/pkg/settings"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// stubRuntime implements tasks.RuntimeCoordinator to observe wake/start calls without launching real work.
type stubRuntime struct {
	terminations atomic.Int32
	forceStarts  atomic.Int32
	startIfSlots atomic.Int32
	wakes        atomic.Int32
}

func (s *stubRuntime) TerminateRunningProcess(string) error { s.terminations.Add(1); return nil }
func (s *stubRuntime) ForceStartTask(string, bool) error    { s.forceStarts.Add(1); return nil }
func (s *stubRuntime) StartTaskIfSlotAvailable(string) error {
	s.startIfSlots.Add(1)
	return nil
}
func (s *stubRuntime) Wake() { s.wakes.Add(1) }

// ensureSettings updates settings for the test and restores afterward.
func ensureSettings(t *testing.T, configure func(settings.Settings) settings.Settings) func() {
	t.Helper()
	prev := settings.GetSettings()
	next := configure(prev)
	settings.UpdateSettings(next)
	return func() { settings.UpdateSettings(prev) }
}

// buildTestStorage creates a queue directory with all needed buckets.
func buildTestStorage(t *testing.T) *tasks.Storage {
	t.Helper()
	queueDir := t.TempDir()
	for _, status := range []string{
		tasks.StatusPending,
		tasks.StatusInProgress,
		tasks.StatusCompleted,
		tasks.StatusFailed,
		tasks.StatusFailedBlocked,
		tasks.StatusCompletedFinalized,
	} {
		if err := os.MkdirAll(filepath.Join(queueDir, status), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", status, err)
		}
	}
	return tasks.NewStorage(queueDir)
}

// Test manual terminal moves lock auto-requeue, apply cooldown, and recycler skips them.
func TestRecyclerSkipsManualTerminalWithCooldown(t *testing.T) {
	restore := ensureSettings(t, func(s settings.Settings) settings.Settings {
		s.Recycler.EnabledFor = "both"
		s.CooldownSeconds = 5
		return s
	})
	defer restore()

	storage := buildTestStorage(t)
	task := tasks.TaskItem{
		ID:                   "manual-complete-skip",
		Status:               tasks.StatusInProgress,
		ProcessorAutoRequeue: true,
	}
	if err := storage.SaveQueueItem(task, tasks.StatusInProgress); err != nil {
		t.Fatalf("save in-progress: %v", err)
	}

	lc := tasks.Lifecycle{Store: storage}
	outcome, err := lc.ApplyTransition(tasks.TransitionRequest{
		TaskID:   task.ID,
		ToStatus: tasks.StatusCompleted,
		TransitionContext: tasks.TransitionContext{
			Intent: tasks.IntentManual,
			Now:    func() time.Time { return time.Unix(0, 0) },
		},
	})
	if err != nil {
		t.Fatalf("manual completion: %v", err)
	}
	if outcome.Task == nil || outcome.Task.Status != tasks.StatusCompleted {
		t.Fatalf("expected completed status, got %+v", outcome.Task)
	}
	if outcome.Task.ProcessorAutoRequeue {
		t.Fatalf("expected auto-requeue disabled on manual completion")
	}
	if outcome.Task.CooldownUntil == "" {
		t.Fatalf("expected cooldown applied on manual completion")
	}

	rt := &stubRuntime{}

	r := New(storage, nil)
	r.SetCoordinator(&tasks.Coordinator{LC: &lc, Store: storage, Runtime: rt})
	r.workCh = make(chan string, 2)
	r.pending = make(map[string]struct{})
	r.failureAttempts = make(map[string]int)
	r.active = true

	r.handleWork(task.ID)

	// Still completed and not enqueued for recycle.
	stored, status, err := storage.GetTaskByID(task.ID)
	if err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if status != tasks.StatusCompleted {
		t.Fatalf("expected status completed, got %s", status)
	}
	if stored.ProcessorAutoRequeue {
		t.Fatalf("auto-requeue should remain disabled")
	}
	if len(r.pending) != 0 {
		t.Fatalf("expected no pending entries, got %d", len(r.pending))
	}
}

// Test recycler respects cooldown, then recycles and wakes processor once cooldown expires.
func TestRecyclerRespectsCooldownThenRecycles(t *testing.T) {
	restore := ensureSettings(t, func(s settings.Settings) settings.Settings {
		s.Recycler.EnabledFor = "both"
		s.CooldownSeconds = 0 // use explicit CooldownUntil below
		return s
	})
	defer restore()

	storage := buildTestStorage(t)
	task := tasks.TaskItem{
		ID:                   "cooldown-then-recycle",
		Status:               tasks.StatusCompleted,
		ProcessorAutoRequeue: true,
		CooldownUntil:        time.Now().Add(1 * time.Second).Format(time.RFC3339),
	}
	if err := storage.SaveQueueItem(task, tasks.StatusCompleted); err != nil {
		t.Fatalf("save completed: %v", err)
	}

	lc := tasks.Lifecycle{Store: storage}
	rt := &stubRuntime{}

	r := New(storage, nil)
	r.SetCoordinator(&tasks.Coordinator{LC: &lc, Store: storage, Runtime: rt})
	r.workCh = make(chan string, 4)
	r.pending = make(map[string]struct{})
	r.failureAttempts = make(map[string]int)
	r.active = true

	// First pass should respect cooldown and schedule later.
	r.handleWork(task.ID)

	stored, status, err := storage.GetTaskByID(task.ID)
	if err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if status != tasks.StatusCompleted {
		t.Fatalf("expected still completed during cooldown, got %s", status)
	}
	if rt.wakes.Load() != 0 {
		t.Fatalf("expected no wake during cooldown, got %d", rt.wakes.Load())
	}

	// Wait for cooldown to expire and process the enqueued retry.
	time.Sleep(1200 * time.Millisecond)
	select {
	case id := <-r.workCh:
		r.handleWork(id)
	default:
		t.Fatalf("expected task enqueued after cooldown")
	}

	stored, status, err = storage.GetTaskByID(task.ID)
	if err != nil {
		t.Fatalf("reload task after recycle: %v", err)
	}
	if status != tasks.StatusPending || stored.Status != tasks.StatusPending {
		t.Fatalf("expected task moved to pending after cooldown, got file status %s, struct %s", status, stored.Status)
	}
	if stored.CooldownUntil != "" {
		t.Fatalf("expected cooldown cleared after recycle, got %s", stored.CooldownUntil)
	}
	if rt.wakes.Load() == 0 {
		t.Fatalf("expected wake invoked after recycle, got %d", rt.wakes.Load())
	}
}

// Manual terminal move locks auto-requeue; after a reconcile/pending move and automated completion,
// recycler should be allowed to recycle again and wake the processor.
func TestRecyclerUnlocksAfterManualRecovery(t *testing.T) {
	restore := ensureSettings(t, func(s settings.Settings) settings.Settings {
		s.Recycler.EnabledFor = "both"
		s.CooldownSeconds = 0
		return s
	})
	defer restore()

	storage := buildTestStorage(t)
	task := tasks.TaskItem{
		ID:                   "recycle-after-recovery",
		Status:               tasks.StatusInProgress,
		ProcessorAutoRequeue: true,
	}
	if err := storage.SaveQueueItem(task, tasks.StatusInProgress); err != nil {
		t.Fatalf("save in-progress: %v", err)
	}

	lc := tasks.Lifecycle{Store: storage}
	rt := &stubRuntime{}
	coord := &tasks.Coordinator{LC: &lc, Store: storage, Runtime: rt}

	r := New(storage, nil)
	r.SetCoordinator(coord)
	r.workCh = make(chan string, 4)
	r.pending = make(map[string]struct{})
	r.failureAttempts = make(map[string]int)
	r.active = true

	// Manual completion should lock auto-requeue.
	if _, err := lc.ApplyTransition(tasks.TransitionRequest{
		TaskID:   task.ID,
		ToStatus: tasks.StatusCompleted,
		TransitionContext: tasks.TransitionContext{
			Intent: tasks.IntentManual,
		},
	}); err != nil {
		t.Fatalf("manual completion: %v", err)
	}

	r.handleWork(task.ID)
	if rt.wakes.Load() != 0 {
		t.Fatalf("expected no wake for locked manual completion, got %d", rt.wakes.Load())
	}
	if status, _ := storage.CurrentStatus(task.ID); status != tasks.StatusCompleted {
		t.Fatalf("expected still completed when locked, got %s", status)
	}

	// Move back to pending via reconcile to clear lock/cooldown.
	if _, err := lc.ApplyTransition(tasks.TransitionRequest{
		TaskID:   task.ID,
		ToStatus: tasks.StatusPending,
		TransitionContext: tasks.TransitionContext{
			Intent: tasks.IntentReconcile,
		},
	}); err != nil {
		t.Fatalf("reconcile to pending: %v", err)
	}

	// Automated completion keeps auto-requeue enabled.
	if _, err := lc.ApplyTransition(tasks.TransitionRequest{
		TaskID:   task.ID,
		ToStatus: tasks.StatusCompleted,
		TransitionContext: tasks.TransitionContext{
			Intent: tasks.IntentAuto,
		},
	}); err != nil {
		t.Fatalf("auto completion: %v", err)
	}

	r.handleWork(task.ID)

	if status, _ := storage.CurrentStatus(task.ID); status != tasks.StatusPending {
		t.Fatalf("expected task recycled to pending, got %s", status)
	}
	if rt.wakes.Load() == 0 {
		t.Fatalf("expected wake after successful recycle")
	}
}
