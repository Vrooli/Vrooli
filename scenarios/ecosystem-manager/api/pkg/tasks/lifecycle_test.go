package tasks

import (
	"testing"
	"time"
)

// mockStorage is an in-memory StorageAPI for lifecycle tests.
type mockStorage struct {
	items map[string]map[string]TaskItem // status -> id -> task
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		items: map[string]map[string]TaskItem{
			StatusPending:            {},
			StatusInProgress:         {},
			StatusCompleted:          {},
			StatusFailed:             {},
			StatusFailedBlocked:      {},
			StatusCompletedFinalized: {},
			StatusArchived:           {},
		},
	}
}

func (m *mockStorage) MoveTaskTo(taskID, status string) (*TaskItem, string, error) {
	for s, bucket := range m.items {
		if task, ok := bucket[taskID]; ok {
			delete(bucket, taskID)
			if _, exists := m.items[status]; !exists {
				m.items[status] = map[string]TaskItem{}
			}
			m.items[status][taskID] = task
			return &task, s, nil
		}
	}
	// If not found, treat as no-op
	return nil, "", nil
}

func (m *mockStorage) SaveQueueItem(task TaskItem, status string) error {
	if _, exists := m.items[status]; !exists {
		m.items[status] = map[string]TaskItem{}
	}
	m.items[status][task.ID] = task
	return nil
}

func (m *mockStorage) SaveQueueItemSkipCleanup(task TaskItem, status string) error {
	return m.SaveQueueItem(task, status)
}

func (m *mockStorage) GetTaskByID(taskID string) (*TaskItem, string, error) {
	for status, bucket := range m.items {
		if task, ok := bucket[taskID]; ok {
			copy := task
			return &copy, status, nil
		}
	}
	return nil, "", nil
}

// ensureSingleBucket asserts that the task exists in exactly one status bucket.
func ensureSingleBucket(t *testing.T, store *mockStorage, taskID string) {
	t.Helper()
	count := 0
	for status, bucket := range store.items {
		if _, ok := bucket[taskID]; ok {
			count++
			if count > 1 {
				t.Fatalf("task %s found in multiple buckets (dup in %s)", taskID, status)
			}
		}
	}
	if count != 1 {
		t.Fatalf("task %s not found in any bucket", taskID)
	}
}

func TestStartPendingIgnoresLockForAlreadyPending(t *testing.T) {
	store := newMockStorage()
	task := TaskItem{ID: "t1", Status: StatusPending, ProcessorAutoRequeue: false}
	store.items[StatusPending][task.ID] = task

	lc := Lifecycle{Store: store}
	started, err := lc.StartPending(task.ID, TransitionContext{})
	if err != nil {
		t.Fatalf("expected pending task to start even when auto-requeue is false, got %v", err)
	}
	if started.Status != StatusInProgress {
		t.Fatalf("expected in-progress, got %s", started.Status)
	}
}

func TestCompleteAppliesCooldownAndDisablesAutoRequeue(t *testing.T) {
	store := newMockStorage()
	task := TaskItem{ID: "t2", Status: StatusInProgress, ProcessorAutoRequeue: true}
	store.items[StatusInProgress][task.ID] = task

	lc := Lifecycle{Store: store}
	updated, _, err := lc.Complete(task.ID, TransitionContext{
		Manual: true,
		Now:    func() time.Time { return time.Unix(0, 0) },
	})
	if err != nil {
		t.Fatalf("complete failed: %v", err)
	}
	if updated.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s", updated.Status)
	}
	if updated.ProcessorAutoRequeue {
		t.Fatalf("expected auto-requeue disabled on completion")
	}
	if updated.CooldownUntil == "" {
		t.Fatalf("expected cooldown to be set on completion")
	}
}

func TestRecycleRespectsCooldownAndLock(t *testing.T) {
	store := newMockStorage()
	task := TaskItem{
		ID:                   "t3",
		Status:               StatusCompleted,
		ProcessorAutoRequeue: true,
		CooldownUntil:        time.Now().Add(10 * time.Minute).Format(time.RFC3339),
	}
	store.items[StatusCompleted][task.ID] = task

	lc := Lifecycle{Store: store}
	if _, _, err := lc.Recycle(task.ID, TransitionContext{}); err == nil {
		t.Fatalf("expected recycle to fail while cooling down")
	}

	task.CooldownUntil = ""
	store.items[StatusCompleted][task.ID] = task

	recycled, fromStatus, err := lc.Recycle(task.ID, TransitionContext{})
	if err != nil {
		t.Fatalf("recycle failed: %v", err)
	}
	if fromStatus != StatusCompleted {
		t.Fatalf("expected from status completed, got %s", fromStatus)
	}
	if recycled.Status != StatusPending {
		t.Fatalf("expected pending after recycle, got %s", recycled.Status)
	}

	// Manual leave from terminal should re-enable auto-requeue even if previously disabled.
	recycled.ProcessorAutoRequeue = false
	store.items[StatusCompleted][task.ID] = *recycled
	recycled, _, err = lc.Recycle(task.ID, TransitionContext{})
	if err != nil {
		t.Fatalf("expected recycle to succeed even when auto-requeue was disabled, got %v", err)
	}
	if !recycled.ProcessorAutoRequeue {
		t.Fatalf("expected auto-requeue re-enabled when leaving terminal state")
	}
}

func TestLifecycleTransitionMatrix(t *testing.T) {
	now := time.Unix(0, 0)
	makeCtx := func(manual, override bool) TransitionContext {
		return TransitionContext{Manual: manual, ForceOverride: override, Now: func() time.Time { return now }}
	}

	tests := []struct {
		name          string
		startStatus   string
		targetStatus  string
		autoRequeue   bool
		forceOverride bool
		expectErr     bool
		expectStatus  string
		expectLock    bool
	}{
		{"Pending->Active ignores lock for running", StatusPending, StatusInProgress, false, false, false, StatusInProgress, false},
		{"Pending->Active auto true", StatusPending, StatusInProgress, true, false, false, StatusInProgress, true},
		{"Active->Completed locks auto-requeue", StatusInProgress, StatusCompleted, true, false, false, StatusCompleted, false},
		{"Completed->Pending re-enables auto-requeue", StatusCompleted, StatusPending, false, true, false, StatusPending, true},
		{"Failed->Pending re-enables auto-requeue", StatusFailed, StatusPending, false, true, false, StatusPending, true},
		{"Completed->Pending allowed without override", StatusCompleted, StatusPending, false, false, false, StatusPending, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockStorage()
			task := TaskItem{
				ID:                   "matrix",
				Status:               tt.startStatus,
				ProcessorAutoRequeue: tt.autoRequeue,
			}
			if _, exists := store.items[tt.startStatus]; !exists {
				store.items[tt.startStatus] = map[string]TaskItem{}
			}
			store.items[tt.startStatus][task.ID] = task

			lc := Lifecycle{Store: store}
			ctx := makeCtx(true, tt.forceOverride)
			var updated *TaskItem
			var err error

			switch tt.targetStatus {
			case StatusInProgress:
				updated, err = lc.StartPending(task.ID, ctx)
			case StatusCompleted:
				updated, _, err = lc.Complete(task.ID, ctx)
			case StatusPending:
				updated, _, err = lc.Recycle(task.ID, ctx)
			default:
				t.Fatalf("unsupported target %s", tt.targetStatus)
			}

			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if updated.Status != tt.expectStatus {
				t.Fatalf("expected status %s, got %s", tt.expectStatus, updated.Status)
			}
			if updated.ProcessorAutoRequeue != tt.expectLock {
				t.Fatalf("expected auto-requeue=%v, got %v", tt.expectLock, updated.ProcessorAutoRequeue)
			}
		})
	}
}

func TestApplyTransitionSideEffects(t *testing.T) {
	store := newMockStorage()
	task := TaskItem{ID: "t-side", Status: StatusInProgress, ProcessorAutoRequeue: true}
	store.items[StatusInProgress][task.ID] = task

	lc := Lifecycle{Store: store}

	// Active -> Pending should signal termination and opportunistic start.
	outcome, err := lc.ApplyTransition(TransitionRequest{
		TaskID:   task.ID,
		ToStatus: StatusPending,
		TransitionContext: TransitionContext{
			Manual: true,
		},
	})
	if err != nil {
		t.Fatalf("apply transition: %v", err)
	}
	if !outcome.Effects.TerminateProcess {
		t.Fatalf("expected terminate flag when leaving in-progress")
	}
	if !outcome.Effects.StartIfSlotAvailable {
		t.Fatalf("expected start-if-slot-available when moving to pending")
	}
	if !outcome.Effects.WakeProcessorAfterSave {
		t.Fatalf("expected wake flag when moving to pending")
	}

	// Pending -> In-progress (manual) should request force start.
	store.items[StatusPending] = map[string]TaskItem{
		task.ID: {ID: task.ID, Status: StatusPending, ProcessorAutoRequeue: true},
	}
	outcome, err = lc.ApplyTransition(TransitionRequest{
		TaskID:   task.ID,
		ToStatus: StatusInProgress,
		TransitionContext: TransitionContext{
			Manual: true,
		},
	})
	if err != nil {
		t.Fatalf("apply transition: %v", err)
	}
	if !outcome.Effects.ForceStart {
		t.Fatalf("expected force-start flag on manual move to active")
	}

	// Pending -> In-progress (automated) should not force overflow, only opportunistic start.
	store.items[StatusPending] = map[string]TaskItem{
		task.ID: {ID: task.ID, Status: StatusPending, ProcessorAutoRequeue: true},
	}
	outcome, err = lc.ApplyTransition(TransitionRequest{
		TaskID:   task.ID,
		ToStatus: StatusInProgress,
		TransitionContext: TransitionContext{
			Manual: false,
		},
	})
	if err != nil {
		t.Fatalf("apply transition: %v", err)
	}
	if outcome.Effects.ForceStart {
		t.Fatalf("expected automated move to avoid force-start overflow")
	}
	if !outcome.Effects.StartIfSlotAvailable {
		t.Fatalf("expected automated move to request opportunistic start")
	}
}

func TestPendingMoveBlockedWhenAutoRequeueLocked(t *testing.T) {
	store := newMockStorage()
	task := TaskItem{ID: "locked", Status: StatusInProgress, ProcessorAutoRequeue: false}
	store.items[StatusInProgress][task.ID] = task

	lc := Lifecycle{Store: store}
	_, err := lc.ApplyTransition(TransitionRequest{
		TaskID:   task.ID,
		ToStatus: StatusPending,
		TransitionContext: TransitionContext{
			Manual: true,
		},
	})
	if err == nil {
		t.Fatalf("expected pending move to be blocked when auto-requeue is false and not terminal")
	}
	ensureSingleBucket(t, store, task.ID)
}

func TestPendingMoveFromTerminalReEnablesAutoRequeue(t *testing.T) {
	store := newMockStorage()
	task := TaskItem{ID: "terminal", Status: StatusCompleted, ProcessorAutoRequeue: false}
	store.items[StatusCompleted][task.ID] = task

	lc := Lifecycle{Store: store}
	outcome, err := lc.ApplyTransition(TransitionRequest{
		TaskID:   task.ID,
		ToStatus: StatusPending,
		TransitionContext: TransitionContext{
			Manual: false,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outcome.Task.Status != StatusPending {
		t.Fatalf("expected pending, got %s", outcome.Task.Status)
	}
	if !outcome.Task.ProcessorAutoRequeue {
		t.Fatalf("expected auto-requeue re-enabled when leaving terminal state")
	}
	ensureSingleBucket(t, store, task.ID)
}

func TestLifecycleRuleMatrix(t *testing.T) {
	now := time.Unix(0, 0)
	tests := []struct {
		name          string
		startStatus   string
		targetStatus  string
		autoRequeue   bool
		cooldownUntil string
		manual        bool
		override      bool
		expectErr     bool
		expectAuto    bool
		expectEffect  TransitionEffects
	}{
		{
			name:         "Manual move pending->active ignores lock and force starts",
			startStatus:  StatusPending,
			targetStatus: StatusInProgress,
			autoRequeue:  false,
			manual:       true,
			expectAuto:   false,
			expectEffect: TransitionEffects{ForceStart: true, WakeProcessorAfterSave: true},
		},
		{
			name:         "Pending->active automated requests opportunistic start",
			startStatus:  StatusPending,
			targetStatus: StatusInProgress,
			autoRequeue:  true,
			manual:       false,
			expectAuto:   true,
			expectEffect: TransitionEffects{StartIfSlotAvailable: true, WakeProcessorAfterSave: true},
		},
		{
			name:         "Manual completion disables auto-requeue and terminates",
			startStatus:  StatusInProgress,
			targetStatus: StatusCompleted,
			autoRequeue:  true,
			manual:       true,
			expectAuto:   false,
			expectEffect: TransitionEffects{TerminateProcess: true},
		},
		{
			name:         "Automated completion keeps auto-requeue enabled",
			startStatus:  StatusInProgress,
			targetStatus: StatusCompleted,
			autoRequeue:  true,
			manual:       false,
			expectAuto:   true,
			expectEffect: TransitionEffects{TerminateProcess: true},
		},
		{
			name:         "Manual blocked disables auto-requeue",
			startStatus:  StatusInProgress,
			targetStatus: StatusFailedBlocked,
			autoRequeue:  true,
			manual:       true,
			expectAuto:   false,
			expectEffect: TransitionEffects{TerminateProcess: true},
		},
		{
			name:         "Pending move blocked when lock set and not leaving terminal",
			startStatus:  StatusInProgress,
			targetStatus: StatusPending,
			autoRequeue:  false,
			manual:       true,
			expectErr:    true,
			expectAuto:   false,
			expectEffect: TransitionEffects{},
		},
		{
			name:          "Pending move from terminal respects cooldown unless override",
			startStatus:   StatusCompleted,
			targetStatus:  StatusPending,
			autoRequeue:   false,
			cooldownUntil: time.Now().Add(time.Hour).Format(time.RFC3339),
			manual:        true,
			expectErr:     true,
			expectAuto:    false,
			expectEffect:  TransitionEffects{},
		},
		{
			name:          "Pending move from terminal with override clears lock and cooldown",
			startStatus:   StatusCompleted,
			targetStatus:  StatusPending,
			autoRequeue:   false,
			cooldownUntil: time.Now().Add(time.Hour).Format(time.RFC3339),
			manual:        true,
			override:      true,
			expectAuto:    true,
			expectEffect:  TransitionEffects{StartIfSlotAvailable: true, WakeProcessorAfterSave: true},
		},
		{
			name:         "Leaving blocked to active re-enables auto and force starts",
			startStatus:  StatusFailedBlocked,
			targetStatus: StatusInProgress,
			autoRequeue:  false,
			manual:       true,
			expectAuto:   true,
			expectEffect: TransitionEffects{ForceStart: true, WakeProcessorAfterSave: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockStorage()
			task := TaskItem{
				ID:                   "rule",
				Status:               tt.startStatus,
				ProcessorAutoRequeue: tt.autoRequeue,
				CooldownUntil:        tt.cooldownUntil,
			}
			if _, exists := store.items[tt.startStatus]; !exists {
				store.items[tt.startStatus] = map[string]TaskItem{}
			}
			store.items[tt.startStatus][task.ID] = task

			lc := Lifecycle{Store: store}
			outcome, err := lc.ApplyTransition(TransitionRequest{
				TaskID:   task.ID,
				ToStatus: tt.targetStatus,
				TransitionContext: TransitionContext{
					Manual:        tt.manual,
					ForceOverride: tt.override,
					Now:           func() time.Time { return now },
				},
			})

			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if outcome.Task.ProcessorAutoRequeue != tt.expectAuto {
				t.Fatalf("expected auto-requeue %v, got %v", tt.expectAuto, outcome.Task.ProcessorAutoRequeue)
			}
			// Check side effects flags of interest
			if tt.expectEffect.TerminateProcess != outcome.Effects.TerminateProcess {
				t.Fatalf("expected terminate=%v got %v", tt.expectEffect.TerminateProcess, outcome.Effects.TerminateProcess)
			}
			if tt.expectEffect.ForceStart != outcome.Effects.ForceStart {
				t.Fatalf("expected forceStart=%v got %v", tt.expectEffect.ForceStart, outcome.Effects.ForceStart)
			}
			if tt.expectEffect.StartIfSlotAvailable != outcome.Effects.StartIfSlotAvailable {
				t.Fatalf("expected startIfSlotAvailable=%v got %v", tt.expectEffect.StartIfSlotAvailable, outcome.Effects.StartIfSlotAvailable)
			}
			if tt.expectEffect.WakeProcessorAfterSave != outcome.Effects.WakeProcessorAfterSave {
				t.Fatalf("expected wakeAfterSave=%v got %v", tt.expectEffect.WakeProcessorAfterSave, outcome.Effects.WakeProcessorAfterSave)
			}
			ensureSingleBucket(t, store, task.ID)
		})
	}
}

func TestManualCompletionDisablesAutoRequeueWhileAutomatedKeepsIt(t *testing.T) {
	store := newMockStorage()
	task := TaskItem{ID: "manual-complete", Status: StatusInProgress, ProcessorAutoRequeue: true}
	store.items[StatusInProgress][task.ID] = task

	lc := Lifecycle{Store: store}
	manualOutcome, err := lc.ApplyTransition(TransitionRequest{
		TaskID:   task.ID,
		ToStatus: StatusCompleted,
		TransitionContext: TransitionContext{
			Manual: true,
		},
	})
	if err != nil {
		t.Fatalf("manual completion failed: %v", err)
	}
	if manualOutcome.Task.ProcessorAutoRequeue {
		t.Fatalf("expected manual completion to disable auto-requeue")
	}
	ensureSingleBucket(t, store, task.ID)

	// Reset for automated completion case.
	store = newMockStorage()
	task = TaskItem{ID: "auto-complete", Status: StatusInProgress, ProcessorAutoRequeue: true}
	store.items[StatusInProgress][task.ID] = task
	lc = Lifecycle{Store: store}
	autoOutcome, err := lc.ApplyTransition(TransitionRequest{
		TaskID:   task.ID,
		ToStatus: StatusCompleted,
		TransitionContext: TransitionContext{
			Manual: false,
		},
	})
	if err != nil {
		t.Fatalf("automated completion failed: %v", err)
	}
	if !autoOutcome.Task.ProcessorAutoRequeue {
		t.Fatalf("expected automated completion to preserve auto-requeue state")
	}
	ensureSingleBucket(t, store, task.ID)
}
