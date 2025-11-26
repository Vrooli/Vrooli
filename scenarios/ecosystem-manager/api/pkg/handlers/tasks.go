package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/internal/timeutil"
	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/ecosystem-manager/api/pkg/queue"
	"github.com/ecosystem-manager/api/pkg/settings"
	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/ecosystem-manager/api/pkg/tasks"
	"github.com/ecosystem-manager/api/pkg/websocket"
	"github.com/gorilla/mux"
)

type taskSort string

const (
	taskSortUpdatedDesc taskSort = "updated_desc"
	taskSortUpdatedAsc  taskSort = "updated_asc"
	taskSortCreatedDesc taskSort = "created_desc"
	taskSortCreatedAsc  taskSort = "created_asc"
)

// TaskHandlers contains handlers for task-related endpoints
type TaskHandlers struct {
	storage           *tasks.Storage
	assembler         *prompts.Assembler
	processor         *queue.Processor
	wsManager         *websocket.Manager
	autoSteerProfiles *autosteer.ProfileService
}

// taskWithRuntime decorates a task with live execution metadata without mutating persisted files.
type taskWithRuntime struct {
	tasks.TaskItem
	CurrentProcess       *queue.ProcessInfo `json:"current_process,omitempty"`
	AutoSteerPhaseIndex  *int               `json:"auto_steer_phase_index,omitempty"`
	AutoSteerCurrentMode string             `json:"auto_steer_mode,omitempty"`
}

// buildRuntimeIndex returns a map of running processes keyed by task ID for quick enrichment.
func (h *TaskHandlers) buildRuntimeIndex() map[string]queue.ProcessInfo {
	index := make(map[string]queue.ProcessInfo)
	if h.processor == nil {
		return index
	}

	for _, proc := range h.processor.GetRunningProcessesInfo() {
		index[proc.TaskID] = proc
	}
	return index
}

// attachRuntime copies the task and adds runtime info when available.
func attachRuntime(task tasks.TaskItem, runtime map[string]queue.ProcessInfo) taskWithRuntime {
	enriched := taskWithRuntime{TaskItem: task}
	if proc, ok := runtime[task.ID]; ok {
		// Copy to avoid aliasing the map entry
		procCopy := proc
		enriched.CurrentProcess = &procCopy
	}
	return enriched
}

type autoSteerRuntime struct {
	phaseIndex *int
	mode       string
}

// buildAutoSteerRuntime gathers live Auto Steer state for the provided tasks.
func (h *TaskHandlers) buildAutoSteerRuntime(tasks []tasks.TaskItem) map[string]autoSteerRuntime {
	result := make(map[string]autoSteerRuntime)

	if h.processor == nil {
		return result
	}
	integration := h.processor.AutoSteerIntegration()
	if integration == nil {
		return result
	}
	engine := integration.ExecutionEngine()
	if engine == nil {
		return result
	}

	for _, task := range tasks {
		if strings.TrimSpace(task.AutoSteerProfileID) == "" {
			continue
		}

		state, err := engine.GetExecutionState(task.ID)
		if err != nil || state == nil {
			continue
		}

		var idxPtr *int
		idx := state.CurrentPhaseIndex
		idxPtr = &idx

		mode, _ := engine.GetCurrentMode(task.ID)

		result[task.ID] = autoSteerRuntime{
			phaseIndex: idxPtr,
			mode:       string(mode),
		}
	}

	return result
}

func (h *TaskHandlers) handleMultiTargetCreate(w http.ResponseWriter, baseTask tasks.TaskItem) {
	created := make([]tasks.TaskItem, 0, len(baseTask.Targets))
	skipped := make([]map[string]string, 0)
	errors := make([]map[string]string, 0)
	baseTitle := ""

	for _, target := range baseTask.Targets {
		existing, status, lookupErr := h.storage.FindActiveTargetTask(baseTask.Type, baseTask.Operation, target)
		if lookupErr != nil {
			errors = append(errors, map[string]string{
				"target": target,
				"error":  lookupErr.Error(),
			})
			continue
		}

		if existing != nil {
			skipped = append(skipped, map[string]string{
				"target": target,
				"reason": fmt.Sprintf("existing %s task %s in %s", baseTask.Operation, existing.ID, status),
			})
			continue
		}

		newTask := baseTask
		newTask.ID = generateTaskID(baseTask.Type, baseTask.Operation, target)
		newTask.Target = target
		newTask.Targets = []string{target}
		newTask.Title = deriveTaskTitle(baseTitle, baseTask.Operation, baseTask.Type, target)
		newTask.Status = "pending"
		newTask.Results = nil
		timestamp := timeutil.NowRFC3339()
		newTask.CreatedAt = timestamp
		newTask.UpdatedAt = timestamp

		if err := h.storage.SaveQueueItem(newTask, "pending"); err != nil {
			errors = append(errors, map[string]string{
				"target": target,
				"error":  err.Error(),
			})
			continue
		}

		created = append(created, newTask)
	}

	success := len(errors) == 0 && len(created) > 0
	response := map[string]any{
		"success": success,
		"created": created,
		"skipped": skipped,
	}

	if len(errors) > 0 {
		response["errors"] = errors
	}

	statusCode := http.StatusConflict
	if len(created) > 0 {
		statusCode = http.StatusCreated
	} else if len(errors) > 0 {
		statusCode = http.StatusInternalServerError
	}

	if len(created) > 0 && h.processor != nil {
		h.processor.Wake()
	}

	writeJSON(w, response, statusCode)
}

var targetSlugSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

func generateTaskID(taskType, operation, target string) string {
	baseTimestamp := time.Now().Format("20060102-150405")
	if strings.TrimSpace(target) == "" {
		return fmt.Sprintf("%s-%s-%s", taskType, operation, baseTimestamp)
	}

	slug := sanitizeTargetSlug(target)
	return fmt.Sprintf("%s-%s-%s-%s", taskType, operation, slug, baseTimestamp)
}

func sanitizeTargetSlug(target string) string {
	slug := strings.ToLower(strings.TrimSpace(target))
	slug = targetSlugSanitizer.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "target"
	}
	if len(slug) > 48 {
		slug = slug[:48]
	}
	return slug
}

func deriveTaskTitle(baseTitle, operation, taskType, target string) string {
	trimmedBase := strings.TrimSpace(baseTitle)
	trimmedTarget := strings.TrimSpace(target)

	if trimmedTarget == "" {
		if trimmedBase != "" {
			return trimmedBase
		}
		return fmt.Sprintf("%s %s", operationDisplayName(operation), taskType)
	}

	if trimmedBase == "" {
		return fmt.Sprintf("%s %s %s", operationDisplayName(operation), taskType, trimmedTarget)
	}

	lowerBase := strings.ToLower(trimmedBase)
	lowerTarget := strings.ToLower(trimmedTarget)
	if strings.Contains(trimmedBase, "{{target}}") {
		return strings.ReplaceAll(trimmedBase, "{{target}}", trimmedTarget)
	}
	if strings.Contains(lowerBase, lowerTarget) {
		return trimmedBase
	}

	return fmt.Sprintf("%s (%s)", trimmedBase, trimmedTarget)
}

func operationDisplayName(operation string) string {
	switch operation {
	case "generator":
		return "Generate"
	case "improver":
		return "Improve"
	default:
		if operation == "" {
			return "Task"
		}
		return strings.ToUpper(operation[:1]) + operation[1:]
	}
}

// NewTaskHandlers creates a new task handlers instance
func NewTaskHandlers(storage *tasks.Storage, assembler *prompts.Assembler, processor *queue.Processor, wsManager *websocket.Manager, autoSteerProfiles *autosteer.ProfileService) *TaskHandlers {
	return &TaskHandlers{
		storage:           storage,
		assembler:         assembler,
		processor:         processor,
		wsManager:         wsManager,
		autoSteerProfiles: autoSteerProfiles,
	}
}

// getTaskFromRequest extracts task ID from URL path and retrieves the task.
// Returns (task, status, true) on success or (nil, "", false) on error (response already written).
func (h *TaskHandlers) getTaskFromRequest(r *http.Request, w http.ResponseWriter) (*tasks.TaskItem, string, bool) {
	vars := mux.Vars(r)
	taskID := vars["id"]
	task, status, err := h.storage.GetTaskByID(taskID)
	if err != nil {
		writeError(w, "Task not found", http.StatusNotFound)
		return nil, "", false
	}
	return task, status, true
}

// validateTaskTypeAndOperation validates task type and operation fields.
// Returns true if valid, writes error response and returns false if invalid.
func (h *TaskHandlers) validateTaskTypeAndOperation(task *tasks.TaskItem, w http.ResponseWriter) bool {
	if !tasks.IsValidTaskType(task.Type) {
		writeError(w, fmt.Sprintf("Invalid type: %s. Must be one of: %v", task.Type, tasks.ValidTaskTypes), http.StatusBadRequest)
		return false
	}
	if !tasks.IsValidTaskOperation(task.Operation) {
		writeError(w, fmt.Sprintf("Invalid operation: %s. Must be one of: %v", task.Operation, tasks.ValidTaskOperations), http.StatusBadRequest)
		return false
	}
	return true
}

// normalizeSteerMode trims and lowercases a provided steer mode string, treating "none" as empty.
func normalizeSteerMode(raw string) autosteer.SteerMode {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "none" {
		return ""
	}
	return autosteer.SteerMode(trimmed)
}

// validateAndNormalizeSteerMode ensures the steer mode is supported for the task shape and valid.
func validateAndNormalizeSteerMode(task *tasks.TaskItem, w http.ResponseWriter) bool {
	mode := normalizeSteerMode(task.SteerMode)
	if mode == "" {
		task.SteerMode = ""
		return true
	}

	if task.Type != "scenario" || task.Operation != "improver" {
		writeError(w, "Manual steering is only supported for scenario improver tasks", http.StatusBadRequest)
		return false
	}

	if !mode.IsValid() {
		writeError(w, fmt.Sprintf("Invalid steer_mode: %s. Must be one of: %v", mode, []autosteer.SteerMode{
			autosteer.ModeProgress,
			autosteer.ModeUX,
			autosteer.ModeRefactor,
			autosteer.ModeTest,
			autosteer.ModeExplore,
			autosteer.ModePolish,
			autosteer.ModePerformance,
			autosteer.ModeSecurity,
		}), http.StatusBadRequest)
		return false
	}

	task.SteerMode = string(mode)
	return true
}

// preserveUnsetFields copies non-zero values from current to updated for fields that are unset.
// This helper consolidates field preservation logic used when updating tasks.
func preserveUnsetFields(updated, current *tasks.TaskItem, preserveSteerMode bool) {
	if updated.Title == "" {
		updated.Title = current.Title
	}
	if updated.Priority == "" {
		updated.Priority = current.Priority
	}
	if updated.Category == "" {
		updated.Category = current.Category
	}
	if updated.EffortEstimate == "" {
		updated.EffortEstimate = current.EffortEstimate
	}
	if updated.CurrentPhase == "" && current.CurrentPhase != "" {
		updated.CurrentPhase = current.CurrentPhase
	}
	if updated.CompletionCount == 0 && current.CompletionCount > 0 {
		updated.CompletionCount = current.CompletionCount
	}
	if updated.LastCompletedAt == "" {
		updated.LastCompletedAt = current.LastCompletedAt
	}
	if len(updated.Targets) == 0 && len(current.Targets) > 0 {
		updated.Targets = current.Targets
		updated.Target = current.Target
	}
	if preserveSteerMode && updated.SteerMode == "" && current.SteerMode != "" {
		updated.SteerMode = current.SteerMode
	}
}

// isBackwardsTransition checks if a status change represents a backwards transition
// (moving from a terminal state back to an active state for re-execution)
func isBackwardsTransition(fromStatus, toStatus string) bool {
	terminalStatuses := map[string]bool{"completed": true, "failed": true}
	activeStatuses := map[string]bool{"pending": true, "in-progress": true}
	return terminalStatuses[fromStatus] && activeStatuses[toStatus]
}

// clearTaskExecutionData resets task fields for fresh execution after a backwards transition
func clearTaskExecutionData(task *tasks.TaskItem, taskID, fromStatus, toStatus string) {
	systemlog.Infof("Task %s moved backwards from %s to %s - clearing execution data", taskID, fromStatus, toStatus)
	task.Results = nil
	task.StartedAt = ""
	task.CompletedAt = ""
	task.CurrentPhase = ""
	task.CooldownUntil = ""
}

// applyStatusTransitionLogic handles status change logic including timestamps, completion counts,
// backwards transitions, and process termination. Returns the updated timestamp string.
func (h *TaskHandlers) applyStatusTransitionLogic(task *tasks.TaskItem, taskID, currentStatus, newStatus string) (string, error) {
	now := timeutil.NowRFC3339()

	// Handle backwards status transitions (completed/failed -> pending/in-progress)
	backwards := isBackwardsTransition(currentStatus, newStatus)
	if backwards {
		clearTaskExecutionData(task, taskID, currentStatus, newStatus)
	} else {
		// Normal transitions - preserve existing data if not explicitly cleared
		// (Frontend can clear by sending empty values in the update)
	}

	// Set status-specific timestamps and state
	if newStatus == "in-progress" {
		task.StartedAt = now
	}

	if newStatus == "completed" {
		if task.CompletionCount <= 0 || backwards {
			task.CompletionCount = 1
		} else {
			task.CompletionCount++
		}
		if task.CompletedAt == "" {
			task.CompletedAt = now
		}
		task.LastCompletedAt = task.CompletedAt
	}

	if newStatus == "failed" && task.CompletedAt == "" {
		task.CompletedAt = now
	}

	if newStatus == "completed" || newStatus == "failed" {
		if task.ProcessorAutoRequeue {
			cooldownSeconds := settings.GetSettings().CooldownSeconds
			if cooldownSeconds > 0 {
				task.CooldownUntil = time.Now().Add(time.Duration(cooldownSeconds) * time.Second).Format(time.RFC3339)
			} else {
				task.CooldownUntil = ""
			}
		} else {
			task.CooldownUntil = ""
		}
	}

	if newStatus == "archived" {
		if strings.TrimSpace(task.CurrentPhase) == "" {
			task.CurrentPhase = "archived"
		}
		task.ProcessorAutoRequeue = false
	}

	if newStatus == "pending" {
		task.ConsecutiveCompletionClaims = 0
		task.ConsecutiveFailures = 0
		// Do not override an explicit disable. Respect the caller's setting.
		if !task.ProcessorAutoRequeue {
			systemlog.Debugf("Task %s pending transition: auto-requeue remains disabled", taskID)
		} else {
			task.ProcessorAutoRequeue = true
		}
		task.CooldownUntil = ""
	}

	if newStatus == "completed-finalized" {
		task.ProcessorAutoRequeue = false
	}

	if newStatus == "failed-blocked" {
		task.ProcessorAutoRequeue = false
	}

	// CRITICAL: If task is moved OUT of in-progress, terminate any running process
	if currentStatus == "in-progress" && newStatus != "in-progress" {
		if err := h.processor.TerminateRunningProcess(taskID); err != nil {
			systemlog.Warnf("Failed to terminate process for task %s: %v", taskID, err)
		} else {
			systemlog.Infof("Successfully terminated process for task %s (moved from in-progress to %s)", taskID, newStatus)
			// Update task to reflect cancellation
			cancelTime := timeutil.NowRFC3339()
			task.Results = map[string]any{
				"success":      false,
				"error":        fmt.Sprintf("Task execution was cancelled (moved to %s)", newStatus),
				"cancelled_at": cancelTime,
			}
			task.CurrentPhase = "cancelled"
		}
	}

	return now, nil
}

func parseTaskSortParam(raw string) taskSort {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "updated_asc", "updated-asc", "updated_at_asc", "updated-at-asc":
		return taskSortUpdatedAsc
	case "created_desc", "created-desc", "created_at_desc", "created-at-desc":
		return taskSortCreatedDesc
	case "created_asc", "created-asc", "created_at_asc", "created-at-asc":
		return taskSortCreatedAsc
	default:
		return taskSortUpdatedDesc
	}
}

func parseTimestamp(raw string) time.Time {
	if strings.TrimSpace(raw) == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return t
}

func updatedTimestamp(task tasks.TaskItem) time.Time {
	if ts := parseTimestamp(task.UpdatedAt); !ts.IsZero() {
		return ts
	}
	return parseTimestamp(task.CreatedAt)
}

func createdTimestamp(task tasks.TaskItem) time.Time {
	return parseTimestamp(task.CreatedAt)
}

// GetTasksHandler retrieves tasks with optional filtering
func (h *TaskHandlers) GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}

	taskType := r.URL.Query().Get("type")       // filter by resource/scenario
	operation := r.URL.Query().Get("operation") // filter by generator/improver
	category := r.URL.Query().Get("category")   // filter by category
	sortParam := parseTaskSortParam(r.URL.Query().Get("sort"))

	items, err := h.storage.GetQueueItems(status)
	if err != nil {
		writeError(w, fmt.Sprintf("Failed to get tasks: %v", err), http.StatusInternalServerError)
		return
	}

	// Apply filters
	filteredItems := []tasks.TaskItem{} // Initialize as empty slice, not nil
	for _, item := range items {
		if taskType != "" && item.Type != taskType {
			continue
		}
		if operation != "" && item.Operation != operation {
			continue
		}
		if category != "" && item.Category != category {
			continue
		}
		filteredItems = append(filteredItems, item)
	}

	// Apply sorting (default: most recently updated first)
	sort.SliceStable(filteredItems, func(i, j int) bool {
		left := filteredItems[i]
		right := filteredItems[j]

		switch sortParam {
		case taskSortUpdatedAsc:
			li, ri := updatedTimestamp(left), updatedTimestamp(right)
			if li.Equal(ri) {
				return left.ID < right.ID
			}
			return li.Before(ri)
		case taskSortCreatedDesc:
			li, ri := createdTimestamp(left), createdTimestamp(right)
			if li.Equal(ri) {
				return left.ID < right.ID
			}
			return li.After(ri)
		case taskSortCreatedAsc:
			li, ri := createdTimestamp(left), createdTimestamp(right)
			if li.Equal(ri) {
				return left.ID < right.ID
			}
			return li.Before(ri)
		case taskSortUpdatedDesc:
			fallthrough
		default:
			li, ri := updatedTimestamp(left), updatedTimestamp(right)
			if li.Equal(ri) {
				return left.ID < right.ID
			}
			return li.After(ri)
		}
	})

	systemlog.Debugf("Task list requested: status=%s count=%d", status, len(filteredItems))

	runtimeIndex := h.buildRuntimeIndex()
	autoSteerIndex := h.buildAutoSteerRuntime(filteredItems)
	enriched := make([]taskWithRuntime, 0, len(filteredItems))
	for _, item := range filteredItems {
		enrichedTask := attachRuntime(item, runtimeIndex)
		if steer, ok := autoSteerIndex[item.ID]; ok {
			enrichedTask.AutoSteerPhaseIndex = steer.phaseIndex
			if strings.TrimSpace(steer.mode) != "" {
				enrichedTask.AutoSteerCurrentMode = steer.mode
			}
		}
		enriched = append(enriched, enrichedTask)
	}

	// Wrap response in object for consistency with other endpoints
	response := map[string]any{
		"tasks": enriched,
		"count": len(enriched),
	}

	writeJSON(w, response, http.StatusOK)
}

// CreateTaskHandler creates a new task
func (h *TaskHandlers) CreateTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskPtr, ok := decodeJSONBody[tasks.TaskItem](w, r)
	if !ok {
		return
	}
	task := *taskPtr

	// Normalize target inputs before validation
	task.Targets, task.Target = tasks.NormalizeTargets(task.Target, task.Targets)

	// Validate task type and operation
	if !h.validateTaskTypeAndOperation(&task, w) {
		return
	}

	// Validate that we have configuration for this operation
	_, err := h.assembler.SelectPromptAssembly(task.Type, task.Operation)
	if err != nil {
		writeError(w, fmt.Sprintf("Unsupported operation combination: %v", err), http.StatusBadRequest)
		return
	}

	if !validateAndNormalizeSteerMode(&task, w) {
		return
	}

	// Target validation for improver operations
	if task.Operation == "improver" && len(task.Targets) == 0 {
		writeError(w, "Improver tasks require at least one target", http.StatusBadRequest)
		return
	}

	// Handle multi-target creation as a batch operation
	if len(task.Targets) > 1 {
		h.handleMultiTargetCreate(w, task)
		return
	}

	// Guard against duplicate improver tasks for the same target
	if len(task.Targets) == 1 {
		existing, status, lookupErr := h.storage.FindActiveTargetTask(task.Type, task.Operation, task.Targets[0])
		if lookupErr != nil {
			writeError(w, fmt.Sprintf("Failed to verify existing tasks: %v", lookupErr), http.StatusInternalServerError)
			return
		}

		if existing != nil {
			writeError(w, fmt.Sprintf("An active %s task (%s) already exists for %s (%s status)", task.Operation, existing.ID, task.Targets[0], status), http.StatusConflict)
			return
		}
	}

	// Set defaults
	if task.ID == "" {
		task.ID = generateTaskID(task.Type, task.Operation, task.Target)
	}

	if task.Status == "" {
		task.Status = "pending"
	}

	task.Title = deriveTaskTitle("", task.Operation, task.Type, task.Target)

	if task.CreatedAt == "" {
		task.CreatedAt = timeutil.NowRFC3339()
	}

	task.UpdatedAt = timeutil.NowRFC3339()

	// Ensure canonical single-target representation is persisted
	if len(task.Targets) == 1 {
		task.Target = task.Targets[0]
	}

	// Save to pending queue
	if err := h.storage.SaveQueueItem(task, "pending"); err != nil {
		writeError(w, fmt.Sprintf("Failed to save task: %v", err), http.StatusInternalServerError)
		return
	}

	if h.processor != nil {
		h.processor.Wake()
	}

	writeJSON(w, map[string]any{
		"success": true,
		"task":    task,
	}, http.StatusCreated)
}

// GET /api/tasks/{id}
// GetTaskHandler retrieves a specific task by ID
func (h *TaskHandlers) GetTaskHandler(w http.ResponseWriter, r *http.Request) {
	task, _, ok := h.getTaskFromRequest(r, w)
	if !ok {
		return
	}

	runtimeIndex := h.buildRuntimeIndex()
	enriched := attachRuntime(*task, runtimeIndex)

	writeJSON(w, enriched, http.StatusOK)
}

// GetActiveTargetsHandler returns active targets for the specified type and operation across relevant queues.
func (h *TaskHandlers) GetActiveTargetsHandler(w http.ResponseWriter, r *http.Request) {
	taskType := strings.TrimSpace(r.URL.Query().Get("type"))
	operation := strings.TrimSpace(r.URL.Query().Get("operation"))

	if taskType == "" || operation == "" {
		writeError(w, "type and operation query parameters are required", http.StatusBadRequest)
		return
	}

	reqType := strings.ToLower(taskType)
	reqOperation := strings.ToLower(operation)

	statuses := []string{"pending", "in-progress"}
	response := make([]map[string]string, 0)
	seen := make(map[string]struct{})

	for _, status := range statuses {
		items, err := h.storage.GetQueueItems(status)
		if err != nil {
			writeError(w, fmt.Sprintf("Failed to load %s tasks: %v", status, err), http.StatusInternalServerError)
			return
		}

		for i := range items {
			item := items[i]
			itemType := strings.ToLower(strings.TrimSpace(item.Type))
			itemOperation := strings.ToLower(strings.TrimSpace(item.Operation))

			if itemType != reqType || itemOperation != reqOperation {
				continue
			}

			targets := tasks.CollectTargets(&item)
			for _, target := range targets {
				normalized := strings.ToLower(strings.TrimSpace(target))
				if normalized == "" {
					continue
				}

				if _, exists := seen[normalized]; exists {
					continue
				}
				seen[normalized] = struct{}{}

				response = append(response, map[string]string{
					"target":  target,
					"task_id": item.ID,
					"status":  status,
					"title":   item.Title,
				})
			}
		}
	}

	writeJSON(w, response, http.StatusOK)
}

// UpdateTaskHandler updates an existing task
func (h *TaskHandlers) UpdateTaskHandler(w http.ResponseWriter, r *http.Request) {
	currentTask, currentStatus, ok := h.getTaskFromRequest(r, w)
	if !ok {
		return
	}

	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var raw map[string]any
	if err := json.Unmarshal(bodyBytes, &raw); err != nil {
		// Keep going with structured decode; just log for debugging
		systemlog.Warnf("UpdateTaskHandler: could not decode raw body for presence detection: %v", err)
	}
	notesProvided := false
	if raw != nil {
		if _, ok := raw["notes"]; ok {
			notesProvided = true
		}
	}

	updatedTaskPtr, ok := decodeJSONBody[tasks.TaskItem](w, r)
	if !ok {
		return
	}
	updatedTask := *updatedTaskPtr
	taskID := currentTask.ID

	updatedTask.Targets, updatedTask.Target = tasks.NormalizeTargets(updatedTask.Target, updatedTask.Targets)
	steerModeCleared := strings.EqualFold(strings.TrimSpace(updatedTask.SteerMode), "none")

	// Preserve certain fields that shouldn't be changed via general update
	updatedTask.ID = taskID
	updatedTask.Type = currentTask.Type
	updatedTask.CreatedBy = currentTask.CreatedBy
	updatedTask.CreatedAt = currentTask.CreatedAt

	// Allow operation to be updated but preserve if not provided
	if updatedTask.Operation == "" {
		updatedTask.Operation = currentTask.Operation
	}

	// Validate operation if it was changed
	if !tasks.IsValidTaskOperation(updatedTask.Operation) {
		writeError(w, fmt.Sprintf("Invalid operation: %s. Must be one of: %v", updatedTask.Operation, tasks.ValidTaskOperations), http.StatusBadRequest)
		return
	}

	// Validate that we have configuration for the new operation combination
	if updatedTask.Operation != currentTask.Operation {
		_, err := h.assembler.SelectPromptAssembly(updatedTask.Type, updatedTask.Operation)
		if err != nil {
			writeError(w, fmt.Sprintf("Unsupported operation combination after update: %v", err), http.StatusBadRequest)
			return
		}
	}

	// Preserve all other fields if they weren't provided in the update
	preserveUnsetFields(&updatedTask, currentTask, !steerModeCleared)

	// Notes: only preserve when not provided; allow explicit clearing
	if !notesProvided {
		updatedTask.Notes = currentTask.Notes
	}

	if !validateAndNormalizeSteerMode(&updatedTask, w) {
		return
	}

	// Validate status if provided
	newStatus := updatedTask.Status
	if newStatus != "" && !tasks.IsValidStatus(newStatus) {
		writeError(w, fmt.Sprintf("Invalid status: %s. Must be one of: %v", newStatus, tasks.GetValidStatuses()), http.StatusBadRequest)
		return
	}

	// If no status was provided, keep the current status so we don't attempt a move with an empty destination.
	if newStatus == "" {
		newStatus = currentStatus
		updatedTask.Status = currentStatus
	}

	if updatedTask.Operation == "improver" && len(updatedTask.Targets) == 1 {
		existing, status, lookupErr := h.storage.FindActiveTargetTask(updatedTask.Type, updatedTask.Operation, updatedTask.Targets[0])
		if lookupErr != nil {
			writeError(w, fmt.Sprintf("Failed to verify existing tasks: %v", lookupErr), http.StatusInternalServerError)
			return
		}

		if existing != nil && existing.ID != taskID {
			writeError(w, fmt.Sprintf("An active %s task (%s) already exists for %s (%s status)", updatedTask.Operation, existing.ID, updatedTask.Targets[0], status), http.StatusConflict)
			return
		}
	}

	// Handle status transitions using consolidated logic
	backwards := isBackwardsTransition(currentStatus, newStatus)

	// Debug logging
	systemlog.Debugf("Task %s status transition: '%s' -> '%s' (backwards: %v)",
		taskID, currentStatus, newStatus, backwards)
	systemlog.Debugf("Task %s incoming data - has results: %v, status: '%s'",
		taskID, updatedTask.Results != nil, updatedTask.Status)

	// Preserve existing data if not explicitly provided by frontend
	if !backwards {
		if updatedTask.StartedAt == "" {
			updatedTask.StartedAt = currentTask.StartedAt
		}
		if updatedTask.CompletedAt == "" {
			updatedTask.CompletedAt = currentTask.CompletedAt
		}
		if updatedTask.LastCompletedAt == "" {
			updatedTask.LastCompletedAt = currentTask.LastCompletedAt
		}
		// IMPORTANT: Don't preserve results if moving to active state
		if updatedTask.Results == nil && currentTask.Results != nil {
			if newStatus == "pending" || newStatus == "in-progress" {
				systemlog.Debugf("Task %s: Not preserving results when moving to %s", taskID, newStatus)
			} else {
				updatedTask.Results = currentTask.Results
			}
		}
	}

	// Apply status transition logic
	if newStatus != currentStatus {
		updatedTask.CompletionCount = currentTask.CompletionCount // Preserve for increment logic
		now, err := h.applyStatusTransitionLogic(&updatedTask, taskID, currentStatus, newStatus)
		if err != nil {
			writeError(w, fmt.Sprintf("Failed to apply status transition: %v", err), http.StatusInternalServerError)
			return
		}
		updatedTask.UpdatedAt = now
	} else {
		// Just update timestamp
		updatedTask.UpdatedAt = timeutil.NowRFC3339()
	}

	// If the Auto Steer profile changed, clear any existing Auto Steer execution state and reset iteration counter.
	autoSteerChanged := strings.TrimSpace(updatedTask.AutoSteerProfileID) != strings.TrimSpace(currentTask.AutoSteerProfileID)
	if autoSteerChanged && h.processor != nil {
		if integration := h.processor.AutoSteerIntegration(); integration != nil {
			if engine := integration.ExecutionEngine(); engine != nil {
				if err := engine.DeleteExecutionState(taskID); err != nil {
					systemlog.Warnf("Failed to reset Auto Steer state for task %s after profile change: %v", taskID, err)
				} else {
					systemlog.Infof("Reset Auto Steer state for task %s due to profile change", taskID)
				}
			}
		}
	}

	updatedTask.Title = deriveTaskTitle("", updatedTask.Operation, updatedTask.Type, updatedTask.Target)

	// Save updated task
	if newStatus != currentStatus {
		// Move the task file to the new status directory
		if err := h.storage.MoveTask(taskID, currentStatus, newStatus); err != nil {
			systemlog.Errorf("Failed to move task %s from %s to %s via MoveTask: %v", taskID, currentStatus, newStatus, err)
			writeError(w, fmt.Sprintf("Failed to move task: %v", err), http.StatusInternalServerError)
			return
		}
		// CRITICAL: Save the updated task with all field changes to the new location
		// Do NOT reload from disk as that would discard all the field updates we just applied
		if err := h.storage.SaveQueueItem(updatedTask, newStatus); err != nil {
			systemlog.Errorf("Failed to save updated task %s after move to %s: %v", taskID, newStatus, err)
			writeError(w, fmt.Sprintf("Error saving updated task: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		if err := h.storage.SaveQueueItem(updatedTask, currentStatus); err != nil {
			writeError(w, fmt.Sprintf("Error saving updated task: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Broadcast the update via WebSocket
	h.wsManager.BroadcastUpdate("task_updated", updatedTask)

	systemlog.Infof("Task %s updated successfully", taskID)

	writeJSON(w, map[string]any{
		"success": true,
		"task":    updatedTask,
	}, http.StatusOK)
}

// DeleteTaskHandler deletes a task
func (h *TaskHandlers) DeleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Check if task is running and terminate if necessary
	if err := h.processor.TerminateRunningProcess(taskID); err == nil {
		systemlog.Infof("Terminated running process for deleted task %s", taskID)
	}

	// Delete the task file
	status, err := h.storage.DeleteTask(taskID)
	if err != nil {
		writeError(w, fmt.Sprintf("Failed to delete task: %v", err), http.StatusInternalServerError)
		return
	}

	// Send WebSocket notification
	h.wsManager.BroadcastUpdate("task_deleted", map[string]any{
		"id":     taskID,
		"status": status,
	})

	w.WriteHeader(http.StatusNoContent) // 204 No Content for successful deletion
}

// UpdateTaskStatusHandler updates just the status/progress of a task (simpler than full update)
func (h *TaskHandlers) UpdateTaskStatusHandler(w http.ResponseWriter, r *http.Request) {
	task, currentStatus, ok := h.getTaskFromRequest(r, w)
	if !ok {
		return
	}
	taskID := task.ID

	type updateRequest struct {
		Status       string `json:"status"`
		CurrentPhase string `json:"current_phase"`
	}

	update, ok := decodeJSONBody[updateRequest](w, r)
	if !ok {
		return
	}

	// Validate status if provided
	if update.Status != "" && !tasks.IsValidStatus(update.Status) {
		writeError(w, fmt.Sprintf("Invalid status: %s. Must be one of: %v", update.Status, tasks.GetValidStatuses()), http.StatusBadRequest)
		return
	}

	// Update task fields
	if update.Status != "" && update.Status != currentStatus {
		task.Status = update.Status

		// Apply consolidated status transition logic
		now, err := h.applyStatusTransitionLogic(task, task.ID, currentStatus, update.Status)
		if err != nil {
			writeError(w, fmt.Sprintf("Failed to apply status transition: %v", err), http.StatusInternalServerError)
			return
		}
		task.UpdatedAt = now

		// Move task to new status if needed
		if update.Status != currentStatus {
			if err := h.storage.MoveTask(taskID, currentStatus, update.Status); err != nil {
				writeError(w, fmt.Sprintf("Failed to move task: %v", err), http.StatusInternalServerError)
				return
			}
		}
	}

	if update.CurrentPhase != "" {
		task.CurrentPhase = update.CurrentPhase
	}

	task.UpdatedAt = timeutil.NowRFC3339()

	// Save updated task
	if err := h.storage.SaveQueueItem(*task, task.Status); err != nil {
		writeError(w, fmt.Sprintf("Failed to save task: %v", err), http.StatusInternalServerError)
		return
	}

	// Broadcast the update via WebSocket
	h.wsManager.BroadcastUpdate("task_status_updated", *task)

	if h.processor != nil && (task.Status == "pending" || currentStatus == "in-progress") {
		h.processor.Wake()
	}

	writeJSON(w, *task, http.StatusOK)
}
