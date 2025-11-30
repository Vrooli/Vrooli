package recycler

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"sync/atomic"

	"github.com/ecosystem-manager/api/pkg/internal/timeutil"
	"github.com/ecosystem-manager/api/pkg/settings"
	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/ecosystem-manager/api/pkg/tasks"
	"github.com/ecosystem-manager/api/pkg/websocket"
)

// status buckets processed by the recycler.
var recycleStatuses = []string{"completed", "failed"}

// terminal statuses assigned when thresholds are hit.
const (
	statusCompletedFinalized = "completed-finalized"
	statusFailedBlocked      = "failed-blocked"
)

// enabledFor constants control which task types are eligible.
const (
	enabledOff       = "off"
	enabledResources = "resources"
	enabledScenarios = "scenarios"
	enabledBoth      = "both"
)

// Recycler coordinates the background auto-requeue workflow.
type Recycler struct {
	storage   *tasks.Storage
	wsManager *websocket.Manager
	lifecycle *tasks.Lifecycle

	mu              sync.Mutex
	stopCh          chan struct{}
	wakeCh          chan struct{}
	workCh          chan string
	pending         map[string]struct{}
	failureAttempts map[string]int
	cooldownTimers  map[string]struct{}
	active          bool
	sweepOnly       bool
	coordinator     *tasks.Coordinator

	retryDelay func(int) time.Duration

	stats recyclerStats

	wake func()
}

// New creates a recycler instance.
func New(storage *tasks.Storage, wsManager *websocket.Manager) *Recycler {
	return &Recycler{
		storage:   storage,
		wsManager: wsManager,
		lifecycle: &tasks.Lifecycle{Store: storage},
	}
}

// SetCoordinator wires a centralized transition coordinator for lifecycle enforcement and side effects.
func (r *Recycler) SetCoordinator(coord *tasks.Coordinator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.coordinator = coord
	if coord != nil && coord.LC != nil {
		r.lifecycle = coord.LC
	}
}

// SetWakeFunc registers a callback to nudge the queue processor after requeues.
func (r *Recycler) SetWakeFunc(fn func()) {
	r.mu.Lock()
	r.wake = fn
	r.mu.Unlock()
}

// wakeProcessor notifies the processor via direct callback and runtime (if coordinator wired).
func (r *Recycler) wakeProcessor(taskID string) {
	r.mu.Lock()
	wake := r.wake
	coord := r.coordinator
	r.mu.Unlock()

	if coord != nil && coord.Runtime != nil {
		coord.Runtime.Wake()
	}
	if wake != nil {
		wake()
	}
}

// Start launches the background loop if not already running.
func (r *Recycler) Start() {
	r.mu.Lock()
	if r.active {
		r.mu.Unlock()
		return
	}

	r.stopCh = make(chan struct{})
	r.wakeCh = make(chan struct{}, 1)
	r.workCh = make(chan string, 256)
	r.pending = make(map[string]struct{})
	r.failureAttempts = make(map[string]int)
	r.cooldownTimers = make(map[string]struct{})
	r.active = true

	r.mu.Unlock()

	log.Println("ℹ️  Recycler seeding existing queue items")
	// Seed initial work if enabled; ignore errors to avoid blocking startup
	r.seedFromQueues()
	log.Println("ℹ️  Recycler seeding complete")

	go r.loop()
}

// Stop terminates the background loop.
func (r *Recycler) Stop() {
	r.mu.Lock()
	if !r.active {
		r.mu.Unlock()
		return
	}
	close(r.stopCh)
	r.active = false
	r.mu.Unlock()
}

// Enqueue schedules a task ID for recycler processing if enabled.
func (r *Recycler) Enqueue(taskID string) {
	if r == nil {
		return
	}

	cfg := settings.GetRecyclerSettings()
	if !r.isEnabled(cfg.EnabledFor) {
		return
	}

	// Guard against accidental enqueue of non-recyclable tasks (e.g., finalized/blocked).
	if r.storage != nil {
		task, status, err := r.storage.GetTaskByID(taskID)
		if err == nil {
			if status != "completed" && status != "failed" {
				return
			}
			if !task.ProcessorAutoRequeue {
				return
			}
		}
	}

	r.mu.Lock()
	if !r.active {
		r.mu.Unlock()
		return
	}
	if _, exists := r.pending[taskID]; exists {
		r.mu.Unlock()
		return
	}
	r.pending[taskID] = struct{}{}
	atomic.AddUint64(&r.stats.Enqueued, 1)
	select {
	case r.workCh <- taskID:
	default:
		// Channel full; drop and remove pending entry to avoid leaks
		delete(r.pending, taskID)
		atomic.AddUint64(&r.stats.Dropped, 1)
		log.Printf("Recycler work channel full; dropping enqueue for task %s", taskID)
	}
	r.mu.Unlock()
}

// Wake requests an immediate pass in addition to the scheduled interval.
func (r *Recycler) Wake() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.active {
		return
	}
	select {
	case r.wakeCh <- struct{}{}:
	default:
	}
}

func (r *Recycler) loop() {
	// Keep draining work items quickly; still honor a periodic sweep as a backstop.
	for {
		cfg := settings.GetRecyclerSettings()
		interval := time.Duration(cfg.IntervalSeconds)
		if interval <= 0 {
			interval = 60
		}
		timer := time.NewTimer(interval * time.Second)

		select {
		case <-timer.C:
			r.runSweep(cfg)
		case <-r.wakeCh:
			if !timer.Stop() {
				<-timer.C
			}
			r.runSweep(cfg)
		case <-r.stopCh:
			if !timer.Stop() {
				<-timer.C
			}
			return
		case id := <-r.workCh:
			if !timer.Stop() {
				<-timer.C
			}
			r.handleWork(id)
			// Drain any burst to avoid starvation between timer resets
			r.drainWorkQueue()
			timer.Reset(interval * time.Second)
		}
	}
}

// runSweep scans completed/failed queues as a backstop (e.g., manual moves).
func (r *Recycler) runSweep(cfg settings.RecyclerSettings) {
	if !r.isEnabled(cfg.EnabledFor) {
		return
	}

	if err := r.processOnce(); err != nil {
		log.Printf("Recycler pass failed: %v", err)
	}
}

func (r *Recycler) drainWorkQueue() {
	for {
		select {
		case id := <-r.workCh:
			r.handleWork(id)
		default:
			return
		}
	}
}

func (r *Recycler) processOnce() error {
	cfg := settings.GetRecyclerSettings()
	enabled := strings.ToLower(strings.TrimSpace(cfg.EnabledFor))
	if enabled == "" || enabled == enabledOff {
		return nil
	}

	for _, bucket := range recycleStatuses {
		items, err := r.storage.GetQueueItems(bucket)
		if err != nil {
			log.Printf("Recycler failed to read %s queue: %v", bucket, err)
			continue
		}

		for _, candidate := range items {
			if !isTypeEnabled(candidate.Type, enabled) {
				continue
			}
			if !candidate.ProcessorAutoRequeue {
				continue
			}

			r.handleWork(candidate.ID)
		}
	}

	return nil
}

// handleWork revalidates and processes a single task ID from the work queue.
func (r *Recycler) handleWork(taskID string) {
	cfg := settings.GetRecyclerSettings()
	if !r.isEnabled(cfg.EnabledFor) {
		r.removePending(taskID)
		r.resetFailures(taskID)
		return
	}

	if r.storage == nil {
		log.Printf("Recycler storage unavailable; dropping task %s", taskID)
		r.removePending(taskID)
		r.resetFailures(taskID)
		return
	}

	task, status, err := r.storage.GetTaskByID(taskID)
	r.removePending(taskID)
	if err != nil {
		log.Printf("Recycler could not load task %s: %v", taskID, err)
		r.resetFailures(taskID)
		return
	}
	if status != "completed" && status != "failed" {
		r.resetFailures(taskID)
		return
	}
	if !task.ProcessorAutoRequeue {
		r.resetFailures(taskID)
		return
	}
	if !isTypeEnabled(task.Type, strings.ToLower(strings.TrimSpace(cfg.EnabledFor))) {
		r.resetFailures(taskID)
		return
	}

	if remaining := cooldownRemaining(task); remaining > 0 {
		r.scheduleAfterCooldown(taskID, remaining)
		r.resetFailures(taskID)
		return
	}

	atomic.AddUint64(&r.stats.Processed, 1)

	// Use lifecycle/coordinator to perform the recycle move.
	if r.coordinator != nil {
		outcomeTask, _, err := r.coordinator.ApplyTransition(tasks.TransitionRequest{
			TaskID:   taskID,
			ToStatus: tasks.StatusPending,
			TransitionContext: tasks.TransitionContext{
				Intent: tasks.IntentRecycler,
			},
		}, tasks.ApplyOptions{
			BroadcastEvent: "task_recycled",
			ForceResave:    true,
		})
		if err != nil {
			r.handleProcessingError(taskID, err)
			return
		}
		task = outcomeTask
		r.resetFailures(taskID)
		r.wakeProcessor(taskID)
		return
	} else if r.lifecycle != nil {
		outcome, err := r.lifecycle.ApplyTransition(tasks.TransitionRequest{
			TaskID:   taskID,
			ToStatus: tasks.StatusPending,
			TransitionContext: tasks.TransitionContext{
				Intent: tasks.IntentRecycler,
			},
		})
		if err != nil {
			r.handleProcessingError(taskID, err)
			return
		}
		task = outcome.Task
		r.broadcast(task, "task_recycled")
		r.resetFailures(taskID)
		r.wakeProcessor(taskID)
		return
	}

	// If we reach here, lifecycle/coordinator are unavailable; drop quietly.
	r.resetFailures(taskID)
}

// seedFromQueues enqueues existing eligible tasks on startup or enable.
func (r *Recycler) seedFromQueues() {
	cfg := settings.GetRecyclerSettings()
	if !r.isEnabled(cfg.EnabledFor) {
		return
	}
	if r.storage == nil {
		return
	}

	for _, bucket := range recycleStatuses {
		items, err := r.storage.GetQueueItems(bucket)
		if err != nil {
			log.Printf("Recycler seed: failed to read %s queue: %v", bucket, err)
			continue
		}
		for _, candidate := range items {
			if !candidate.ProcessorAutoRequeue {
				continue
			}
			if !isTypeEnabled(candidate.Type, strings.ToLower(strings.TrimSpace(cfg.EnabledFor))) {
				continue
			}
			r.Enqueue(candidate.ID)
		}
	}
}

// OnSettingsUpdated reacts to enabled_for toggles by reseeding and waking.
func (r *Recycler) OnSettingsUpdated(previous, next settings.Settings) {
	if r == nil {
		return
	}
	prevEnabled := r.isEnabled(previous.Recycler.EnabledFor)
	nextEnabled := r.isEnabled(next.Recycler.EnabledFor)

	// Clear pending when disabling to avoid stale burst on re-enable.
	if prevEnabled && !nextEnabled {
		r.clearPending()
		return
	}

	if !prevEnabled && nextEnabled {
		r.seedFromQueues()
		r.Wake()
	}
}

func (r *Recycler) removePending(taskID string) {
	r.mu.Lock()
	delete(r.pending, taskID)
	r.mu.Unlock()
}

func (r *Recycler) clearPending() {
	r.mu.Lock()
	r.pending = make(map[string]struct{})
	r.failureAttempts = make(map[string]int)
	r.cooldownTimers = make(map[string]struct{})
	for {
		select {
		case <-r.workCh:
			atomic.AddUint64(&r.stats.Dropped, 1)
		default:
			r.mu.Unlock()
			return
		}
	}
}

func (r *Recycler) handleProcessingError(taskID string, err error) {
	cfg := settings.GetRecyclerSettings()
	maxRetries := cfg.MaxRetries
	if maxRetries < settings.MinRecyclerMaxRetries {
		maxRetries = settings.MinRecyclerMaxRetries
	}
	if maxRetries > settings.MaxRecyclerMaxRetries {
		maxRetries = settings.MaxRecyclerMaxRetries
	}
	delaySeconds := cfg.RetryDelaySeconds
	if delaySeconds < settings.MinRecyclerRetryDelaySecs {
		delaySeconds = settings.MinRecyclerRetryDelaySecs
	}
	if delaySeconds > settings.MaxRecyclerRetryDelaySecs {
		delaySeconds = settings.MaxRecyclerRetryDelaySecs
	}

	attempt := r.incrementFailure(taskID)
	log.Printf("Recycler failed to process task %s (attempt %d): %v", taskID, attempt, err)

	if attempt > maxRetries {
		log.Printf("Recycler giving up on task %s after %d attempts", taskID, attempt-1)
		r.resetFailures(taskID)
		return
	}

	atomic.AddUint64(&r.stats.Requeued, 1)
	delay := time.Duration(delaySeconds*attempt) * time.Second
	if r.retryDelay != nil {
		delay = r.retryDelay(attempt)
	}
	if delay < 0 {
		delay = 0
	}
	go func(id string, d time.Duration) {
		time.Sleep(d)
		r.Enqueue(id)
	}(taskID, delay)
}

func (r *Recycler) incrementFailure(taskID string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failureAttempts[taskID]++
	return r.failureAttempts[taskID]
}

func (r *Recycler) resetFailures(taskID string) {
	r.mu.Lock()
	delete(r.failureAttempts, taskID)
	r.mu.Unlock()
}

func cooldownRemaining(task *tasks.TaskItem) time.Duration {
	if task == nil {
		return 0
	}
	if strings.TrimSpace(task.CooldownUntil) == "" {
		return 0
	}
	ts, err := time.Parse(time.RFC3339, task.CooldownUntil)
	if err != nil {
		return 0
	}
	remaining := time.Until(ts)
	if remaining <= 0 {
		return 0
	}
	return remaining
}

func (r *Recycler) scheduleAfterCooldown(taskID string, delay time.Duration) {
	if delay <= 0 {
		r.Enqueue(taskID)
		return
	}

	r.mu.Lock()
	if r.cooldownTimers == nil {
		r.cooldownTimers = make(map[string]struct{})
	}
	if _, exists := r.cooldownTimers[taskID]; exists {
		r.mu.Unlock()
		return
	}
	r.cooldownTimers[taskID] = struct{}{}
	r.mu.Unlock()

	log.Printf("Recycler delaying task %s until cooldown expires (%v)", taskID, delay.Round(time.Second))

	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		<-timer.C
		r.mu.Lock()
		delete(r.cooldownTimers, taskID)
		r.mu.Unlock()
		r.Enqueue(taskID)
	}()
}

// Stats exposes basic recycler counters for observability/testing.
func (r *Recycler) Stats() recyclerStats {
	return recyclerStats{
		Enqueued:  atomic.LoadUint64(&r.stats.Enqueued),
		Dropped:   atomic.LoadUint64(&r.stats.Dropped),
		Processed: atomic.LoadUint64(&r.stats.Processed),
		Requeued:  atomic.LoadUint64(&r.stats.Requeued),
	}
}

func (r *Recycler) processCompletedTask(task *tasks.TaskItem, cfg settings.RecyclerSettings) error {
	now := timeutil.NowRFC3339()

	if !task.ProcessorAutoRequeue {
		log.Printf("Recycler skipping task %s (auto-requeue disabled)", task.ID)
		return nil
	}

	// Get metrics classification (NEW - this is the source of truth)
	// Use Title as fallback if Target is empty (generator tasks use title as scenario name)
	scenarioName := task.Target
	if scenarioName == "" {
		scenarioName = task.Title
	}

	metricsResult, err := getCompletenessClassification(scenarioName)
	if err != nil {
		log.Printf("Completeness check failed for %s: %v", scenarioName, err)
		// Fallback: treat as early stage if metrics unavailable
		metricsResult = CompletenessResult{
			Classification: "early_stage",
			Score:          0,
		}
	}

	// Use structured results for recycler info
	taskResults := tasks.FromMap(task.Results)
	taskResults.SetRecyclerInfo(metricsResult.Classification, now)
	task.Results = taskResults.ToMap()

	// NEW: Use metrics classification directly (no legacy mapping)
	classification := strings.ToLower(metricsResult.Classification)
	switch classification {
	case "production_ready":
		task.ConsecutiveCompletionClaims++ // 96-100: full increment (3x → finalize)
	case "nearly_ready":
		task.ConsecutiveCompletionClaims += 0.5 // 81-95: partial increment (6x → finalize)
	default:
		task.ConsecutiveCompletionClaims = 0 // <81: reset
	}
	task.ConsecutiveFailures = 0

	// Recycler no longer mutates user notes; rely on agent-facing artifacts instead
	task.UpdatedAt = now

	if shouldFinalize(task.ConsecutiveCompletionClaims, cfg.CompletionThreshold) {
		task.Status = statusCompletedFinalized
		task.CurrentPhase = "finalized"
		task.ProcessorAutoRequeue = false
		if task.CompletedAt == "" {
			task.CompletedAt = now
		}
		if err := r.persistTask(task, statusCompletedFinalized); err != nil {
			return err
		}
		r.broadcast(task, "task_finalized")
		log.Printf("Recycler finalized task %s after %.1f consecutive completion claims", task.ID, task.ConsecutiveCompletionClaims)
		return nil
	}

	// Requeue
	task.Status = "pending"
	task.CurrentPhase = ""
	task.StartedAt = ""
	task.CompletedAt = ""

	// CRITICAL: Convert Generator tasks to Improver after first completion
	// Generator creates NEW resources/scenarios, Improver enhances EXISTING ones
	// Once a task completes successfully once, the target exists and should be improved, not regenerated
	if task.Operation == "generator" {
		task.Operation = "improver"
		log.Printf("Recycler converted task %s from generator to improver (completion count: %d)", task.ID, task.CompletionCount)
		systemlog.Infof("Task %s converted from generator to improver after completion", task.ID)
	}

	if err := r.persistTask(task, "pending"); err != nil {
		return err
	}
	r.broadcast(task, "task_recycled")
	log.Printf("Recycler requeued completed task %s", task.ID)
	return nil
}

func (r *Recycler) processFailedTask(task *tasks.TaskItem, cfg settings.RecyclerSettings) error {
	now := timeutil.NowRFC3339()

	if !task.ProcessorAutoRequeue {
		log.Printf("Recycler skipping task %s (auto-requeue disabled)", task.ID)
		return nil
	}

	// Failure streak increments regardless of summarizer outcome.
	task.ConsecutiveFailures++
	task.ConsecutiveCompletionClaims = 0

	// Use structured results for recycler info; do not mutate user notes
	taskResults := tasks.FromMap(task.Results)
	taskResults.SetRecyclerInfo("failed", now)
	task.Results = taskResults.ToMap()

	task.UpdatedAt = now

	if shouldFinalize(float64(task.ConsecutiveFailures), cfg.FailureThreshold) {
		task.Status = statusFailedBlocked
		task.CurrentPhase = "blocked"
		task.ProcessorAutoRequeue = false
		if err := r.persistTask(task, statusFailedBlocked); err != nil {
			return err
		}
		r.broadcast(task, "task_blocked")
		log.Printf("Recycler blocked task %s after %d consecutive failures", task.ID, task.ConsecutiveFailures)
		return nil
	}

	// Requeue the failure for another attempt.
	task.Status = "pending"
	task.CurrentPhase = ""
	task.StartedAt = ""
	task.CompletedAt = ""

	if err := r.persistTask(task, "pending"); err != nil {
		return err
	}
	r.broadcast(task, "task_recycled")
	log.Printf("Recycler requeued failed task %s (failure streak %d)", task.ID, task.ConsecutiveFailures)
	return nil
}

func (r *Recycler) persistTask(task *tasks.TaskItem, targetStatus string) error {
	if _, _, err := r.storage.MoveTaskTo(task.ID, targetStatus); err != nil {
		return fmt.Errorf("move task %s to %s: %w", task.ID, targetStatus, err)
	}
	// Use SkipCleanup since MoveTaskTo already cleaned up duplicates
	if err := r.storage.SaveQueueItemSkipCleanup(*task, targetStatus); err != nil {
		return fmt.Errorf("save task %s in %s: %w", task.ID, targetStatus, err)
	}
	if targetStatus == "pending" {
		r.mu.Lock()
		wake := r.wake
		r.mu.Unlock()
		if wake != nil {
			wake()
		}
	}
	return nil
}

func (r *Recycler) broadcast(task *tasks.TaskItem, event string) {
	if r == nil || r.wsManager == nil || task == nil {
		return
	}
	r.wsManager.BroadcastUpdate(event, map[string]any{
		"task_id": task.ID,
		"task":    task,
		"status":  task.Status,
	})
	if event != "task_status_changed" {
		r.wsManager.BroadcastUpdate("task_status_changed", map[string]any{
			"task_id":    task.ID,
			"new_status": task.Status,
			"task":       task,
		})
	}
}

func ensureResultsMap(task *tasks.TaskItem) {
	if task.Results == nil {
		task.Results = make(map[string]any)
	}
}

func extractOutput(results map[string]any) string {
	if results == nil {
		return ""
	}
	if raw, ok := results["output"]; ok {
		if text, ok := raw.(string); ok {
			return text
		}
	}
	return ""
}

func shouldFinalize(streak float64, threshold int) bool {
	if threshold <= 0 {
		return false
	}
	return streak >= float64(threshold)
}

func isTypeEnabled(taskType string, enabled string) bool {
	switch enabled {
	case enabledBoth:
		return true
	case enabledResources:
		return strings.EqualFold(taskType, "resource")
	case enabledScenarios:
		return strings.EqualFold(taskType, "scenario")
	default:
		return false
	}
}

func (r *Recycler) isEnabled(enabledFor string) bool {
	enabled := strings.ToLower(strings.TrimSpace(enabledFor))
	return enabled != "" && enabled != enabledOff
}

type recyclerStats struct {
	Enqueued  uint64
	Dropped   uint64
	Processed uint64
	Requeued  uint64
}
