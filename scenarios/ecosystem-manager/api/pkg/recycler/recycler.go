package recycler

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/ecosystem-manager/api/pkg/settings"
	"github.com/ecosystem-manager/api/pkg/summarizer"
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

	mu     sync.Mutex
	stopCh chan struct{}
	wakeCh chan struct{}
	active bool
}

// New creates a recycler instance.
func New(storage *tasks.Storage, wsManager *websocket.Manager) *Recycler {
	return &Recycler{
		storage:   storage,
		wsManager: wsManager,
	}
}

// Start launches the background loop if not already running.
func (r *Recycler) Start() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active {
		return
	}

	r.stopCh = make(chan struct{})
	r.wakeCh = make(chan struct{}, 1)
	r.active = true

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
	for {
		cfg := settings.GetRecyclerSettings()
		interval := time.Duration(cfg.IntervalSeconds)
		if interval <= 0 {
			interval = 60
		}
		timer := time.NewTimer(interval * time.Second)

		select {
		case <-timer.C:
		case <-r.wakeCh:
			if !timer.Stop() {
				<-timer.C
			}
		case <-r.stopCh:
			if !timer.Stop() {
				<-timer.C
			}
			return
		}

		if err := r.processOnce(); err != nil {
			log.Printf("Recycler pass failed: %v", err)
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

			task, status, err := r.storage.GetTaskByID(candidate.ID)
			if err != nil {
				log.Printf("Recycler could not load task %s: %v", candidate.ID, err)
				continue
			}
			if status != bucket {
				continue
			}

			if bucket == "completed" {
				return r.processCompletedTask(task, cfg)
			}
			return r.processFailedTask(task, cfg)
		}
	}

	return nil
}

func (r *Recycler) processCompletedTask(task *tasks.TaskItem, cfg settings.RecyclerSettings) error {
	output := extractOutput(task.Results)
	result := summarizer.DefaultResult()
	var err error
	if strings.TrimSpace(output) != "" {
		result, err = summarizer.GenerateNote(context.Background(), summarizer.Config{
			Provider: cfg.ModelProvider,
			Model:    cfg.ModelName,
		}, summarizer.Input{Task: *task, Output: output, PreviousNote: task.Notes})
		if err != nil {
			log.Printf("Recycler summarizer error for task %s: %v", task.ID, err)
			result = summarizer.DefaultResult()
		}
	}

	now := time.Now().Format(time.RFC3339)
	ensureResultsMap(task)
	task.Results["recycler_classification"] = result.Classification
	task.Results["recycler_updated_at"] = now

	classification := strings.ToLower(result.Classification)
	switch classification {
	case "full_complete":
		task.ConsecutiveCompletionClaims++
	default:
		task.ConsecutiveCompletionClaims = 0
	}
	task.ConsecutiveFailures = 0

	task.Notes = result.Note
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
		log.Printf("Recycler finalized task %s after %d consecutive completion claims", task.ID, task.ConsecutiveCompletionClaims)
		return nil
	}

	// Requeue
	task.Status = "pending"
	task.CurrentPhase = ""
	task.StartedAt = ""
	task.CompletedAt = ""
	task.ProcessorAutoRequeue = true

	if err := r.persistTask(task, "pending"); err != nil {
		return err
	}
	r.broadcast(task, "task_recycled")
	log.Printf("Recycler requeued completed task %s", task.ID)
	return nil
}

func (r *Recycler) processFailedTask(task *tasks.TaskItem, cfg settings.RecyclerSettings) error {
	output := extractOutput(task.Results)
	now := time.Now().Format(time.RFC3339)

	ensureResultsMap(task)
	task.Results["recycler_updated_at"] = now

	// Failure streak increments regardless of summarizer outcome.
	task.ConsecutiveFailures++
	task.ConsecutiveCompletionClaims = 0

	if strings.TrimSpace(output) != "" {
		result, err := summarizer.GenerateNote(context.Background(), summarizer.Config{
			Provider: cfg.ModelProvider,
			Model:    cfg.ModelName,
		}, summarizer.Input{Task: *task, Output: output, PreviousNote: task.Notes})
		if err != nil {
			log.Printf("Recycler summarizer error for failed task %s: %v", task.ID, err)
			task.Notes = "Not sure current status"
			task.Results["recycler_classification"] = "uncertain"
		} else {
			task.Notes = result.Note
			task.Results["recycler_classification"] = result.Classification
		}
	} else {
		task.Notes = "Not sure current status"
		task.Results["recycler_classification"] = "uncertain"
	}

	task.UpdatedAt = now

	if shouldFinalize(task.ConsecutiveFailures, cfg.FailureThreshold) {
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
	task.ProcessorAutoRequeue = true

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
	if err := r.storage.SaveQueueItem(*task, targetStatus); err != nil {
		return fmt.Errorf("save task %s in %s: %w", task.ID, targetStatus, err)
	}
	return nil
}

func (r *Recycler) broadcast(task *tasks.TaskItem, event string) {
	r.wsManager.BroadcastUpdate(event, map[string]interface{}{
		"task_id": task.ID,
		"task":    task,
		"status":  task.Status,
	})
	if event != "task_status_changed" {
		r.wsManager.BroadcastUpdate("task_status_changed", map[string]interface{}{
			"task_id":    task.ID,
			"new_status": task.Status,
			"task":       task,
		})
	}
}

func ensureResultsMap(task *tasks.TaskItem) {
	if task.Results == nil {
		task.Results = make(map[string]interface{})
	}
}

func extractOutput(results map[string]interface{}) string {
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

func shouldFinalize(streak, threshold int) bool {
	if threshold <= 0 {
		return false
	}
	return streak >= threshold
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
