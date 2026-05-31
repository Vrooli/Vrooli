// DOC: docs/concepts/ARCHITECTURE.md
// DOC: docs/reference/api-endpoints.md
package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
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
	"github.com/ecosystem-manager/api/pkg/steering"
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

// TargetValidator checks whether a task target exists on disk.
type TargetValidator interface {
	TargetExists(taskType, targetName string) bool
}

// TaskHandlers contains handlers for task-related endpoints
type TaskHandlers struct {
	storage           tasks.StorageAPI
	assembler         *prompts.Assembler
	processor         ProcessorAPI
	wsManager         *websocket.Manager
	autoSteerProfiles autosteer.ProfileRepository
	coordinator       *tasks.Coordinator
	lifecycle         *tasks.Lifecycle
	queueStateRepo    steering.QueueStateRepository
	targetValidator   TargetValidator
}

func writeTransitionError(w http.ResponseWriter, prefix string, err error) bool {
	var terr *tasks.TransitionError
	if !errors.As(err, &terr) {
		return false
	}

	status := http.StatusConflict
	if terr.Code == tasks.TransitionErrorCodeUnsupported {
		status = http.StatusBadRequest
	}

	message := terr.Error()
	if prefix != "" {
		message = fmt.Sprintf("%s: %s", prefix, message)
	}

	writeError(w, message, status)
	return true
}

func (h *TaskHandlers) maybeInitializeAutoSteer(task *tasks.TaskItem) {
	if task == nil || strings.TrimSpace(task.AutoSteerProfileID) == "" {
		return
	}
	if h.processor == nil || h.processor.AutoSteerIntegration() == nil {
		return
	}

	scenarioName := strings.TrimSpace(queue.GetScenarioNameFromTask(task))
	if scenarioName == "" {
		scenarioName = strings.TrimSpace(task.Target)
	}

	if err := h.processor.AutoSteerIntegration().InitializeAutoSteer(task, scenarioName); err != nil {
		systemlog.Warnf("Auto Steer initialization failed for task %s after update/create: %v", task.ID, err)
	}
}

func (h *TaskHandlers) validateAutoSteerProfile(task *tasks.TaskItem, w http.ResponseWriter) bool {
	if task == nil {
		return true
	}
	profileID := strings.TrimSpace(task.AutoSteerProfileID)
	task.AutoSteerProfileID = profileID
	if profileID == "" {
		return true
	}
	if h.autoSteerProfiles == nil {
		writeError(w, "auto-steer profile validation is unavailable", http.StatusServiceUnavailable)
		return false
	}
	if _, err := h.autoSteerProfiles.GetProfile(profileID); err != nil {
		writeStructuredError(w, ErrorOpts{
			Code:         "profile_not_found",
			Message:      fmt.Sprintf("Auto Steer profile %q not found", profileID),
			RecoveryHint: "List profiles: ecosystem-manager steer profiles\nList templates: ecosystem-manager steer templates",
		}, http.StatusBadRequest)
		return false
	}
	return true
}

// taskWithRuntime decorates a task with live execution metadata without mutating persisted files.
type taskWithRuntime struct {
	tasks.TaskItem
	CurrentProcess           *queue.ProcessInfo `json:"current_process,omitempty"`
	AutoSteerPhaseIndex      *int               `json:"auto_steer_phase_index,omitempty"`
	AutoSteerCurrentSet      []string           `json:"auto_steer_set,omitempty"`
	SteeringQueueIndex       *int               `json:"steering_queue_index,omitempty"`
	SteeringQueueSet         []string           `json:"steering_queue_set,omitempty"`
	SteeringQueueTotal       int                `json:"steering_queue_total,omitempty"`
	SteeringQueueIsExhausted bool               `json:"steering_queue_exhausted,omitempty"`
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
	skillSet   []string
}

type queueSteeringRuntime struct {
	index       *int
	skillSet    []string
	total       int
	isExhausted bool
}

// buildQueueSteeringRuntime gathers live queue steering state for tasks with steering_queue.
func (h *TaskHandlers) buildQueueSteeringRuntime(taskItems []tasks.TaskItem) map[string]queueSteeringRuntime {
	result := make(map[string]queueSteeringRuntime)

	if h.queueStateRepo == nil {
		return result
	}

	for _, task := range taskItems {
		if len(task.SteeringQueue) == 0 {
			continue
		}

		state, err := h.queueStateRepo.Get(task.ID)
		if err != nil || state == nil {
			// No state yet - queue hasn't started, show index 0
			idx := 0
			result[task.ID] = queueSteeringRuntime{
				index:       &idx,
				skillSet:    task.SteeringQueue[0],
				total:       len(task.SteeringQueue),
				isExhausted: false,
			}
			continue
		}

		idx := state.CurrentIndex

		// Use task.SteeringQueue (the source of truth) for set, total, and exhausted check.
		// This ensures runtime state reflects queue edits from task payload updates.
		queueLen := len(task.SteeringQueue)
		isExhausted := idx >= queueLen

		var skillSet []string
		if !isExhausted && idx >= 0 && idx < queueLen {
			skillSet = task.SteeringQueue[idx]
		}

		result[task.ID] = queueSteeringRuntime{
			index:       &idx,
			skillSet:    skillSet,
			total:       queueLen,
			isExhausted: isExhausted,
		}
	}

	return result
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
	orchestrator := integration.ExecutionOrchestrator()
	if orchestrator == nil {
		return result
	}

	for _, task := range tasks {
		if strings.TrimSpace(task.AutoSteerProfileID) == "" {
			continue
		}

		state, err := orchestrator.GetExecutionState(task.ID)
		if err != nil || state == nil {
			continue
		}

		var idxPtr *int
		// The controller has no phases; surface the global iteration count.
		idx := state.Iteration
		idxPtr = &idx

		skillSet, _ := orchestrator.GetCurrentSet(task.ID)

		result[task.ID] = autoSteerRuntime{
			phaseIndex: idxPtr,
			skillSet:   skillSet,
		}
	}

	return result
}

func (h *TaskHandlers) handleMultiTargetCreate(w http.ResponseWriter, r *http.Request, baseTask tasks.TaskItem) {
	// In dry-run mode, return validated tasks without persisting.
	if isDryRun(r) {
		preview := make([]tasks.TaskItem, 0, len(baseTask.Targets))
		for _, target := range baseTask.Targets {
			t := baseTask
			t.ID = generateTaskID(baseTask.Type, baseTask.Operation, target)
			t.Target = target
			t.Targets = []string{target}
			t.Title = deriveTaskTitle("", baseTask.Operation, baseTask.Type, target)
			t.Status = "pending"
			preview = append(preview, t)
		}
		writeJSON(w, map[string]any{
			"success": true,
			"dry_run": true,
			"created": preview,
		}, http.StatusOK)
		return
	}

	created := make([]tasks.TaskItem, 0, len(baseTask.Targets))
	skipped := make([]map[string]string, 0)
	errors := make([]map[string]string, 0)

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
		newTask.Title = deriveTaskTitle("", baseTask.Operation, baseTask.Type, target)
		newTask.Status = "pending"
		newTask.ProcessorAutoRequeue = true
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

// NewTaskHandlers creates a new task handlers instance.
// targetValidator is optional (nil skips target existence checks).
func NewTaskHandlers(storage tasks.StorageAPI, assembler *prompts.Assembler, processor ProcessorAPI, wsManager *websocket.Manager, autoSteerProfiles autosteer.ProfileRepository, coordinator *tasks.Coordinator, queueStateRepo steering.QueueStateRepository, targetValidator TargetValidator) *TaskHandlers {
	lc := &tasks.Lifecycle{Store: storage}
	if coordinator != nil && coordinator.LC != nil {
		lc = coordinator.LC
	}
	coord := coordinator
	if coord == nil {
		coord = &tasks.Coordinator{
			LC:          lc,
			Store:       storage,
			Runtime:     processor,
			Broadcaster: wsManager,
		}
	}
	return &TaskHandlers{
		storage:           storage,
		assembler:         assembler,
		processor:         processor,
		wsManager:         wsManager,
		autoSteerProfiles: autoSteerProfiles,
		coordinator:       coord,
		lifecycle:         lc,
		queueStateRepo:    queueStateRepo,
		targetValidator:   targetValidator,
	}
}

// getTaskFromRequest extracts task ID from URL path and retrieves the task.
// Returns (task, status, true) on success or (nil, "", false) on error (response already written).
func (h *TaskHandlers) getTaskFromRequest(r *http.Request, w http.ResponseWriter) (*tasks.TaskItem, string, bool) {
	vars := mux.Vars(r)
	taskID := vars["id"]
	task, status, err := h.storage.GetTaskByID(taskID)
	if err != nil {
		writeStructuredError(w, ErrorOpts{
			Code:         "task_not_found",
			Message:      "Task not found",
			RecoveryHint: "List tasks: ecosystem-manager task list",
		}, http.StatusNotFound)
		return nil, "", false
	}
	return task, status, true
}

// validateTaskTypeAndOperation validates task type and operation fields.
// Returns true if valid, writes error response and returns false if invalid.
func (h *TaskHandlers) validateTaskTypeAndOperation(task *tasks.TaskItem, w http.ResponseWriter) bool {
	if !tasks.IsValidTaskType(task.Type) {
		writeStructuredError(w, ErrorOpts{
			Code:         "invalid_task_type",
			Message:      fmt.Sprintf("Invalid type: %s. Must be one of: %v", task.Type, tasks.ValidTaskTypes),
			RecoveryHint: fmt.Sprintf("Valid types: %v", tasks.ValidTaskTypes),
		}, http.StatusBadRequest)
		return false
	}
	if !tasks.IsValidTaskOperation(task.Operation) {
		writeStructuredError(w, ErrorOpts{
			Code:         "invalid_operation",
			Message:      fmt.Sprintf("Invalid operation: %s. Must be one of: %v", task.Operation, tasks.ValidTaskOperations),
			RecoveryHint: fmt.Sprintf("Valid operations: %v", tasks.ValidTaskOperations),
		}, http.StatusBadRequest)
		return false
	}
	return true
}

func normalizeSkillID(raw string) string {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "none" {
		return ""
	}
	return trimmed
}

func allowedSteerSkillSet() map[string]struct{} {
	allowed := make(map[string]struct{})
	for _, mode := range autosteer.AllowedSteerModes() {
		allowed[string(mode)] = struct{}{}
	}
	return allowed
}

func normalizeSteerSet(raw []string, allowed map[string]struct{}) ([]string, error) {
	normalized := make([]string, 0, len(raw))
	for _, skillID := range raw {
		normalizedID := normalizeSkillID(skillID)
		if normalizedID == "" {
			continue
		}
		if _, ok := allowed[normalizedID]; !ok {
			return nil, fmt.Errorf("invalid skill id %q", skillID)
		}
		normalized = append(normalized, normalizedID)
	}
	return normalized, nil
}

func validateAndNormalizeSteerSet(task *tasks.TaskItem, w http.ResponseWriter) bool {
	if len(task.SteerSet) == 0 {
		return true
	}
	if task.Type != "scenario" || task.Operation != "improver" {
		writeError(w, "Manual steering (--steer-set/--steering-queue) is only supported for improver tasks. Use --steer-profile instead, or switch to 'task improve'", http.StatusBadRequest)
		return false
	}
	normalized, err := normalizeSteerSet(task.SteerSet, allowedSteerSkillSet())
	if err != nil {
		writeError(w, fmt.Sprintf("Invalid steer_set: %v. Must be one of: %v", err, autosteer.AllowedSteerModes()), http.StatusBadRequest)
		return false
	}
	if len(normalized) == 0 {
		task.SteerSet = nil
		return true
	}
	task.SteerSet = normalized
	return true
}

// validateAndNormalizeSteeringQueue ensures queue entries are non-empty valid skill sets.
func validateAndNormalizeSteeringQueue(task *tasks.TaskItem, w http.ResponseWriter) bool {
	if len(task.SteeringQueue) == 0 {
		return true
	}

	if task.Type != "scenario" || task.Operation != "improver" {
		writeError(w, "Steering queue (--steering-queue) is only supported for scenario improver tasks. Use --steer-profile instead, or switch to 'task improve scenario <name>'", http.StatusBadRequest)
		return false
	}

	allowed := allowedSteerSkillSet()
	normalizedQueue := make([][]string, 0, len(task.SteeringQueue))
	for i, rawSet := range task.SteeringQueue {
		normalizedSet, err := normalizeSteerSet(rawSet, allowed)
		if err != nil {
			writeError(w, fmt.Sprintf("Invalid skill in steering_queue[%d]: %v. Must be one of: %v", i, err, autosteer.AllowedSteerModes()), http.StatusBadRequest)
			return false
		}
		if len(normalizedSet) == 0 {
			writeError(w, fmt.Sprintf("steering_queue[%d] cannot be empty", i), http.StatusBadRequest)
			return false
		}
		normalizedQueue = append(normalizedQueue, normalizedSet)
	}

	task.SteeringQueue = normalizedQueue
	return true
}

// preserveUnsetFields copies non-zero values from current to updated for fields that are unset.
// This helper consolidates field preservation logic used when updating tasks.
func preserveUnsetFields(updated, current *tasks.TaskItem, preserveSteerSet bool) {
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
	if preserveSteerSet && len(updated.SteerSet) == 0 && len(current.SteerSet) > 0 {
		updated.SteerSet = append([]string(nil), current.SteerSet...)
	}
}

// applyUserEditableFields copies non-status user-editable fields from src into dst.
func applyUserEditableFields(dst *tasks.TaskItem, src tasks.TaskItem, notesProvided, originProvided bool) {
	dst.Title = src.Title
	dst.Priority = src.Priority
	dst.Category = src.Category
	dst.EffortEstimate = src.EffortEstimate
	dst.Urgency = src.Urgency
	dst.Dependencies = src.Dependencies
	dst.Blocks = src.Blocks
	dst.RelatedScenarios = src.RelatedScenarios
	dst.RelatedResources = src.RelatedResources
	dst.ValidationCriteria = src.ValidationCriteria
	dst.Target = src.Target
	dst.Targets = src.Targets
	dst.Tags = src.Tags
	dst.SteerSet = src.SteerSet
	dst.AutoSteerProfileID = src.AutoSteerProfileID
	dst.SteeringQueue = src.SteeringQueue
	dst.ProcessorAutoRequeue = src.ProcessorAutoRequeue
	if notesProvided {
		dst.Notes = src.Notes
	}
	if originProvided {
		dst.Origin = src.Origin
	}
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
	queueSteerIndex := h.buildQueueSteeringRuntime(filteredItems)
	enriched := make([]taskWithRuntime, 0, len(filteredItems))
	for _, item := range filteredItems {
		enrichedTask := attachRuntime(item, runtimeIndex)
		if steer, ok := autoSteerIndex[item.ID]; ok {
			enrichedTask.AutoSteerPhaseIndex = steer.phaseIndex
			if len(steer.skillSet) > 0 {
				enrichedTask.AutoSteerCurrentSet = steer.skillSet
			}
		}
		if queueSteer, ok := queueSteerIndex[item.ID]; ok {
			enrichedTask.SteeringQueueIndex = queueSteer.index
			enrichedTask.SteeringQueueSet = queueSteer.skillSet
			enrichedTask.SteeringQueueTotal = queueSteer.total
			enrichedTask.SteeringQueueIsExhausted = queueSteer.isExhausted
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

	if !validateAndNormalizeSteerSet(&task, w) {
		return
	}

	if !validateAndNormalizeSteeringQueue(&task, w) {
		return
	}
	if !h.validateAutoSteerProfile(&task, w) {
		return
	}

	// Target validation for improver operations
	if task.Operation == "improver" && len(task.Targets) == 0 {
		writeError(w, "Improver tasks require at least one target", http.StatusBadRequest)
		return
	}

	// Validate that improver targets exist on disk (generators create new targets)
	if task.Operation == "improver" && h.targetValidator != nil {
		for _, target := range task.Targets {
			if !h.targetValidator.TargetExists(task.Type, target) {
				writeStructuredError(w, ErrorOpts{
					Code:         "target_not_found",
					Message:      fmt.Sprintf("Target %s %q not found. Verify it exists before creating an improver task", task.Type, target),
					RecoveryHint: fmt.Sprintf("For new targets use 'task add'. Check: ls scenarios/%s", target),
				}, http.StatusBadRequest)
				return
			}
		}
	}

	// Handle multi-target creation as a batch operation
	if len(task.Targets) > 1 {
		h.handleMultiTargetCreate(w, r, task)
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
			writeStructuredError(w, ErrorOpts{
				Code:         "duplicate_task",
				Message:      fmt.Sprintf("An active %s task (%s) already exists for %s (%s status)", task.Operation, existing.ID, task.Targets[0], status),
				RecoveryHint: fmt.Sprintf("View existing: ecosystem-manager task show %s", existing.ID),
			}, http.StatusConflict)
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
	task.ProcessorAutoRequeue = true

	// Ensure canonical single-target representation is persisted
	if len(task.Targets) == 1 {
		task.Target = task.Targets[0]
	}

	// In dry-run mode, return the validated task without persisting.
	if isDryRun(r) {
		writeJSON(w, map[string]any{
			"success": true,
			"dry_run": true,
			"task":    task,
		}, http.StatusOK)
		return
	}

	// Save to pending queue
	if err := h.storage.SaveQueueItem(task, "pending"); err != nil {
		writeError(w, fmt.Sprintf("Failed to save task: %v", err), http.StatusInternalServerError)
		return
	}

	h.maybeInitializeAutoSteer(&task)

	if h.processor != nil {
		h.processor.Wake()
	}

	writeJSON(w, map[string]any{
		"success": true,
		"task":    task,
		"next_steps": []string{
			fmt.Sprintf("ecosystem-manager task show %s", task.ID),
			"ecosystem-manager queue start",
		},
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
	originProvided := false
	if raw != nil {
		if _, ok := raw["notes"]; ok {
			notesProvided = true
		}
		if _, ok := raw["origin"]; ok {
			originProvided = true
		}
	}

	updatedTaskPtr, ok := decodeJSONBody[tasks.TaskItem](w, r)
	if !ok {
		return
	}
	updatedTask := *updatedTaskPtr
	taskID := currentTask.ID

	updatedTask.Targets, updatedTask.Target = tasks.NormalizeTargets(updatedTask.Target, updatedTask.Targets)
	steerSetCleared := len(updatedTask.SteerSet) == 1 && strings.EqualFold(strings.TrimSpace(updatedTask.SteerSet[0]), "none")

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
	preserveUnsetFields(&updatedTask, currentTask, !steerSetCleared)

	// Notes: only preserve when not provided; allow explicit clearing
	if !notesProvided {
		updatedTask.Notes = currentTask.Notes
	}
	if !originProvided {
		updatedTask.Origin = currentTask.Origin
	}

	if !validateAndNormalizeSteerSet(&updatedTask, w) {
		return
	}

	if !validateAndNormalizeSteeringQueue(&updatedTask, w) {
		return
	}

	// Validate status if provided
	newStatus := updatedTask.Status
	if newStatus != "" && !tasks.IsValidStatus(newStatus) {
		writeStructuredError(w, ErrorOpts{
			Code:         "invalid_status",
			Message:      fmt.Sprintf("Invalid status: %s. Must be one of: %v", newStatus, tasks.GetValidStatuses()),
			RecoveryHint: fmt.Sprintf("Valid statuses: %v", tasks.GetValidStatuses()),
		}, http.StatusBadRequest)
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
			writeStructuredError(w, ErrorOpts{
				Code:         "duplicate_task",
				Message:      fmt.Sprintf("An active %s task (%s) already exists for %s (%s status)", updatedTask.Operation, existing.ID, updatedTask.Targets[0], status),
				RecoveryHint: fmt.Sprintf("View existing: ecosystem-manager task show %s", existing.ID),
			}, http.StatusConflict)
			return
		}
	}

	// In dry-run mode, return the validated update without persisting.
	if isDryRun(r) {
		writeJSON(w, map[string]any{
			"success": true,
			"dry_run": true,
			"task":    updatedTask,
		}, http.StatusOK)
		return
	}

	// Route transitions through coordinator for consistency and centralized side effects.
	ctx := tasks.TransitionContext{Intent: tasks.IntentManual}

	updated, outcome, err := h.coordinator.ApplyTransition(tasks.TransitionRequest{
		TaskID:            taskID,
		ToStatus:          newStatus,
		TransitionContext: ctx,
	}, tasks.ApplyOptions{
		Mutate: func(t *tasks.TaskItem) {
			applyUserEditableFields(t, updatedTask, notesProvided, originProvided)
			t.Title = deriveTaskTitle("", t.Operation, t.Type, t.Target)
		},
		BroadcastEvent: "task_updated",
		ForceResave:    true,
	})
	if err != nil {
		if writeTransitionError(w, "Failed to apply status transition", err) {
			return
		}
		writeError(w, fmt.Sprintf("Failed to apply status transition: %v", err), http.StatusInternalServerError)
		return
	}

	// Defensive: ensure outcome not nil
	if outcome == nil || updated == nil {
		writeError(w, "Failed to apply status transition", http.StatusInternalServerError)
		return
	}

	systemlog.Infof("Task %s updated successfully", taskID)
	h.maybeInitializeAutoSteer(updated)

	writeJSON(w, map[string]any{
		"success": true,
		"task":    updated,
		"next_steps": []string{
			fmt.Sprintf("ecosystem-manager task show %s", taskID),
		},
	}, http.StatusOK)
}

// DeleteTaskHandler deletes a task
func (h *TaskHandlers) DeleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// In dry-run mode, verify the task exists but don't delete.
	if isDryRun(r) {
		if _, _, err := h.storage.GetTaskByID(taskID); err != nil {
			writeError(w, "Task not found", http.StatusNotFound)
			return
		}
		writeJSON(w, map[string]any{
			"success": true,
			"dry_run": true,
			"id":      taskID,
		}, http.StatusOK)
		return
	}

	// If possible, rely on coordinator runtime to stop any running process before deletion.
	if h.coordinator != nil && h.processor != nil {
		_ = h.processor.TerminateRunningProcess(taskID)
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

	writeJSON(w, map[string]any{
		"success": true,
		"id":      taskID,
		"next_steps": []string{
			"ecosystem-manager task list",
		},
	}, http.StatusOK)
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
		writeStructuredError(w, ErrorOpts{
			Code:         "invalid_status",
			Message:      fmt.Sprintf("Invalid status: %s. Must be one of: %v", update.Status, tasks.GetValidStatuses()),
			RecoveryHint: fmt.Sprintf("Valid statuses: %v", tasks.GetValidStatuses()),
		}, http.StatusBadRequest)
		return
	}

	targetStatus := update.Status
	if targetStatus == "" {
		targetStatus = currentStatus
	}

	if isDryRun(r) {
		writeJSON(w, map[string]any{
			"success":       true,
			"dry_run":       true,
			"task_id":       taskID,
			"target_status": targetStatus,
		}, http.StatusOK)
		return
	}

	updated, outcome, err := h.coordinator.ApplyTransition(tasks.TransitionRequest{
		TaskID:   taskID,
		ToStatus: targetStatus,
		TransitionContext: tasks.TransitionContext{
			Intent: tasks.IntentManual,
		},
	}, tasks.ApplyOptions{
		Mutate: func(t *tasks.TaskItem) {
			if update.CurrentPhase != "" {
				t.CurrentPhase = update.CurrentPhase
			}
			if targetStatus == tasks.StatusPending && currentStatus != tasks.StatusPending {
				t.Results = nil
				t.CurrentPhase = ""
			}
		},
		BroadcastEvent: "task_status_updated",
		ForceResave:    true,
	})
	if err != nil {
		if writeTransitionError(w, "Failed to apply status transition", err) {
			return
		}
		writeError(w, fmt.Sprintf("Failed to apply status transition: %v", err), http.StatusInternalServerError)
		return
	}
	if outcome == nil || updated == nil {
		writeError(w, "Failed to apply status transition", http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"success": true,
		"task":    updated,
		"next_steps": []string{
			fmt.Sprintf("ecosystem-manager task show %s", taskID),
			"ecosystem-manager task list --status " + updated.Status,
		},
	}, http.StatusOK)
}

// SetQueuePositionHandler sets the steering queue position for a task
// PUT /api/tasks/{id}/queue-position
func (h *TaskHandlers) SetQueuePositionHandler(w http.ResponseWriter, r *http.Request) {
	task, _, ok := h.getTaskFromRequest(r, w)
	if !ok {
		return
	}

	type positionRequest struct {
		Position int `json:"position"`
	}

	req, ok := decodeJSONBody[positionRequest](w, r)
	if !ok {
		return
	}

	// Validate task has a steering queue
	if len(task.SteeringQueue) == 0 {
		writeError(w, "Task does not have a steering queue", http.StatusBadRequest)
		return
	}

	// Validate bounds
	if req.Position < 0 || req.Position >= len(task.SteeringQueue) {
		writeError(w, fmt.Sprintf("Position %d out of bounds (0-%d)", req.Position, len(task.SteeringQueue)-1), http.StatusBadRequest)
		return
	}

	if isDryRun(r) {
		writeJSON(w, map[string]any{
			"success":   true,
			"dry_run":   true,
			"position":  req.Position,
			"steer_set": task.SteeringQueue[req.Position],
		}, http.StatusOK)
		return
	}

	// Set position
	if h.queueStateRepo == nil {
		writeError(w, "Queue state repository not available", http.StatusInternalServerError)
		return
	}

	if err := h.queueStateRepo.SetPosition(task.ID, req.Position); err != nil {
		writeError(w, fmt.Sprintf("Failed to set queue position: %v", err), http.StatusInternalServerError)
		return
	}

	// Broadcast update
	if h.wsManager != nil {
		h.wsManager.BroadcastUpdate("queue_position_changed", map[string]any{
			"task_id":   task.ID,
			"position":  req.Position,
			"steer_set": task.SteeringQueue[req.Position],
		})
	}

	writeJSON(w, map[string]any{
		"success":   true,
		"position":  req.Position,
		"steer_set": task.SteeringQueue[req.Position],
	}, http.StatusOK)
}
