package recycler

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ecosystem-manager/api/pkg/settings"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// helper to enable recycler for tests and restore previous settings.
func withRecyclerEnabled(t *testing.T) func() {
	t.Helper()
	prev := settings.GetSettings()
	updated := prev
	updated.Recycler.EnabledFor = "both"
	settings.UpdateSettings(updated)
	return func() {
		settings.UpdateSettings(prev)
	}
}

func TestEnqueueDeduplicatesAndRespectsEnabled(t *testing.T) {
	restore := withRecyclerEnabled(t)
	defer restore()

	r := New(nil, nil)
	r.workCh = make(chan string, 2)
	r.pending = make(map[string]struct{})
	r.failureAttempts = make(map[string]int)
	r.active = true

	r.Enqueue("task-1")
	r.Enqueue("task-1")

	if len(r.pending) != 1 {
		t.Fatalf("expected pending to contain 1 task, got %d", len(r.pending))
	}
	if got := len(r.workCh); got != 1 {
		t.Fatalf("expected workCh len 1, got %d", got)
	}
	if stats := r.Stats(); stats.Enqueued != 1 {
		t.Fatalf("expected 1 enqueued, got %+v", stats)
	}
}

func TestOnSettingsUpdatedClearsPendingWhenDisabled(t *testing.T) {
	restore := withRecyclerEnabled(t)
	defer restore()

	r := New(nil, nil)
	r.workCh = make(chan string, 4)
	r.pending = map[string]struct{}{"task-a": {}, "task-b": {}}
	r.failureAttempts = map[string]int{"task-a": 2}
	r.active = true

	r.workCh <- "task-a"

	prev := settings.GetSettings()
	next := prev
	next.Recycler.EnabledFor = "off"

	r.OnSettingsUpdated(prev, next)

	if len(r.pending) != 0 {
		t.Fatalf("expected pending to be cleared when disabled, got %d entries", len(r.pending))
	}
	if len(r.failureAttempts) != 0 {
		t.Fatalf("expected failureAttempts cleared when disabled, got %d", len(r.failureAttempts))
	}
	if got := len(r.workCh); got != 0 {
		t.Fatalf("expected work channel drained when disabled, got %d", got)
	}
	if stats := r.Stats(); stats.Dropped == 0 {
		t.Fatalf("expected drops to increment when draining channel, stats=%+v", stats)
	}
}

func TestSeedFromQueuesEnqueuesEligibleTasks(t *testing.T) {
	restore := withRecyclerEnabled(t)
	defer restore()

	queueDir := t.TempDir()
	storage := tasks.NewStorage(queueDir)

	if err := os.MkdirAll(filepath.Join(queueDir, "completed"), 0o755); err != nil {
		t.Fatalf("mkdir completed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(queueDir, "pending"), 0o755); err != nil {
		t.Fatalf("mkdir pending: %v", err)
	}

	eligible := tasks.TaskItem{ID: "eligible", Type: "scenario", ProcessorAutoRequeue: true}
	ineligible := tasks.TaskItem{ID: "ineligible", Type: "scenario", ProcessorAutoRequeue: false}
	otherStatus := tasks.TaskItem{ID: "ignored", Type: "scenario", ProcessorAutoRequeue: true}

	if err := storage.SaveQueueItem(eligible, "completed"); err != nil {
		t.Fatalf("save eligible: %v", err)
	}
	if err := storage.SaveQueueItem(ineligible, "completed"); err != nil {
		t.Fatalf("save ineligible: %v", err)
	}
	if err := storage.SaveQueueItem(otherStatus, "pending"); err != nil {
		t.Fatalf("save other status: %v", err)
	}

	r := New(storage, nil)
	r.workCh = make(chan string, 4)
	r.pending = make(map[string]struct{})
	r.failureAttempts = make(map[string]int)
	r.active = true

	r.seedFromQueues()

	if len(r.pending) != 1 {
		t.Fatalf("expected only eligible task enqueued, got %d", len(r.pending))
	}
	if _, ok := r.pending["eligible"]; !ok {
		t.Fatalf("expected eligible task in pending set")
	}
	if got := len(r.workCh); got != 1 {
		t.Fatalf("expected work channel to contain eligible task, got %d", got)
	}
	if stats := r.Stats(); stats.Enqueued != 1 {
		t.Fatalf("expected enqueued stat to equal 1, got %+v", stats)
	}
}

func TestHandleProcessingErrorRetriesAndStops(t *testing.T) {
	restore := withRecyclerEnabled(t)
	defer restore()

	queueDir := t.TempDir()
	storage := tasks.NewStorage(queueDir)
	if err := os.MkdirAll(filepath.Join(queueDir, "completed"), 0o755); err != nil {
		t.Fatalf("mkdir completed: %v", err)
	}
	task := tasks.TaskItem{
		ID:                   "retry-task",
		Type:                 "scenario",
		Status:               "completed",
		ProcessorAutoRequeue: true,
	}
	if err := storage.SaveQueueItem(task, "completed"); err != nil {
		t.Fatalf("save task: %v", err)
	}

	var attempts int32

	r := New(storage, nil)
	r.workCh = make(chan string, 8)
	r.pending = make(map[string]struct{})
	r.failureAttempts = make(map[string]int)
	r.processCompleted = func(*tasks.TaskItem, settings.RecyclerSettings) error {
		atomic.AddInt32(&attempts, 1)
		return fmt.Errorf("boom")
	}
	r.retryDelay = func(int) time.Duration { return 0 }
	r.active = true

	// Kick off first attempt
	r.handleWork("retry-task")

	// Process any requeues until channel is empty
	drained := false
	for i := 0; i < 5 && !drained; i++ {
		time.Sleep(5 * time.Millisecond)
		select {
		case id := <-r.workCh:
			r.handleWork(id)
		default:
			drained = true
		}
	}

	if got := atomic.LoadInt32(&attempts); got != 4 { // initial + 3 retries
		t.Fatalf("expected 4 attempts, got %d", got)
	}
	if _, exists := r.failureAttempts["retry-task"]; exists {
		t.Fatalf("expected failureAttempts cleared after max retries")
	}
	stats := r.Stats()
	if stats.Requeued != 3 {
		t.Fatalf("expected 3 requeues, got %+v", stats)
	}
}
