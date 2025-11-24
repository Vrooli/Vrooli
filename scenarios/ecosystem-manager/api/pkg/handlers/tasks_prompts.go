package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/internal/timeutil"
	"github.com/ecosystem-manager/api/pkg/queue"
	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// GetTaskPromptHandler retrieves prompt sections for a task
func (h *TaskHandlers) GetTaskPromptHandler(w http.ResponseWriter, r *http.Request) {
	task, _, ok := h.getTaskFromRequest(r, w)
	if !ok {
		return
	}

	// Generate prompt sections
	sections, err := h.assembler.GeneratePromptSections(*task)
	if err != nil {
		writeError(w, fmt.Sprintf("Failed to generate prompt: %v", err), http.StatusInternalServerError)
		return
	}

	operationConfig, _ := h.assembler.SelectPromptAssembly(task.Type, task.Operation)

	response := map[string]any{
		"task_id":          task.ID,
		"operation":        fmt.Sprintf("%s-%s", task.Type, task.Operation),
		"prompt_sections":  sections,
		"operation_config": operationConfig,
		"task_details":     task,
	}

	writeJSON(w, response, http.StatusOK)
}

// GetAssembledPromptHandler returns the fully assembled prompt for a task
func (h *TaskHandlers) GetAssembledPromptHandler(w http.ResponseWriter, r *http.Request) {
	task, status, ok := h.getTaskFromRequest(r, w)
	if !ok {
		return
	}
	taskID := task.ID

	fromCache := false
	defaultProgressApplied := false
	assembly, err := h.assembler.AssemblePromptForTask(*task)
	if err != nil {
		writeError(w, fmt.Sprintf("Failed to assemble prompt: %v", err), http.StatusInternalServerError)
		return
	}
	prompt := assembly.Prompt
	var manualSteerMode autosteer.SteerMode

	// Check for cached prompt content (legacy behavior)
	promptPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s%s.txt", queue.PromptFilePrefix, taskID))
	if cachedPrompt, err := os.ReadFile(promptPath); err == nil {
		prompt = string(cachedPrompt)
		fromCache = true
		systemlog.Debugf("Using cached prompt from %s", promptPath)
	}

	assembly.Prompt = prompt

	// Apply default Progress steering when Auto Steer is not configured
	if task.Type == "scenario" && task.Operation == "improver" && strings.TrimSpace(task.AutoSteerProfileID) == "" {
		section := h.manualOrDefaultSteeringSection(*task)
		prompt = autosteer.InjectSteeringSection(prompt, section)
		assembly.Prompt = prompt
		if strings.TrimSpace(section) != "" {
			defaultProgressApplied = true
			if mode := normalizeSteerMode(task.SteerMode); mode.IsValid() {
				manualSteerMode = mode
			}
		}
	}

	// Get operation config for metadata
	operationConfig, _ := h.assembler.SelectPromptAssembly(task.Type, task.Operation)

	response := map[string]any{
		"task_id":           task.ID,
		"operation":         fmt.Sprintf("%s-%s", task.Type, task.Operation),
		"prompt":            prompt,
		"prompt_length":     len(prompt),
		"prompt_cached":     fromCache,
		"sections_detailed": assembly.Sections,
		"operation_config":  operationConfig,
		"task_status":       status,
		"task_details":      task,
		"default_progress":  defaultProgressApplied,
	}
	if manualSteerMode.IsValid() {
		response["manual_steer_mode"] = manualSteerMode
	}

	writeJSON(w, response, http.StatusOK)
}

// promptPreviewRequest captures optional data for assembling a preview task
type promptPreviewRequest struct {
	Task               *tasks.TaskItem `json:"task,omitempty"`
	Display            string          `json:"display,omitempty"`
	Type               string          `json:"type,omitempty"`
	Operation          string          `json:"operation,omitempty"`
	Title              string          `json:"title,omitempty"`
	Category           string          `json:"category,omitempty"`
	Priority           string          `json:"priority,omitempty"`
	Notes              string          `json:"notes,omitempty"`
	Tags               []string        `json:"tags,omitempty"`
	Target             string          `json:"target,omitempty"`
	Targets            []string        `json:"targets,omitempty"`
	SteerMode          string          `json:"steer_mode,omitempty"`
	AutoSteerProfileID string          `json:"auto_steer_profile_id,omitempty"`
	AutoSteerPhaseIdx  *int            `json:"auto_steer_phase_index,omitempty"`
}

func (r promptPreviewRequest) buildTask(defaultID string) tasks.TaskItem {
	var task tasks.TaskItem

	if r.Task != nil {
		task = *r.Task
	}

	if r.Type != "" {
		task.Type = r.Type
	}
	if r.Operation != "" {
		task.Operation = r.Operation
	}
	if r.Title != "" {
		task.Title = r.Title
	}
	if r.Category != "" {
		task.Category = r.Category
	}
	if r.Priority != "" {
		task.Priority = r.Priority
	}
	if r.SteerMode != "" {
		task.SteerMode = r.SteerMode
	}
	if r.Notes != "" {
		task.Notes = r.Notes
	}
	if len(r.Tags) > 0 {
		task.Tags = r.Tags
	}
	if r.Target != "" {
		task.Target = r.Target
	}
	if len(r.Targets) > 0 {
		task.Targets = r.Targets
	}
	if r.AutoSteerProfileID != "" {
		task.AutoSteerProfileID = r.AutoSteerProfileID
	}

	if task.ID == "" {
		task.ID = defaultID
	}
	if task.Type == "" {
		task.Type = "resource"
	}
	if task.Operation == "" {
		task.Operation = "generator"
	}
	if task.Title == "" {
		task.Title = "Prompt Viewer Test"
	}
	if task.Category == "" {
		task.Category = "test"
	}
	if task.Priority == "" {
		task.Priority = "medium"
	}
	if task.Notes == "" {
		task.Notes = "Temporary task for prompt viewing"
	}
	if task.CreatedAt == "" {
		task.CreatedAt = timeutil.NowRFC3339()
	}
	if task.Status == "" {
		task.Status = "pending"
	}
	if task.AutoSteerProfileID == "" {
		task.AutoSteerProfileID = r.AutoSteerProfileID
	}

	return task
}

// PromptViewerHandler creates a temporary task to view assembled prompts
func (h *TaskHandlers) PromptViewerHandler(w http.ResponseWriter, r *http.Request) {
	defaultID := fmt.Sprintf("temp-prompt-viewer-%d", time.Now().UnixNano())

	req, ok := decodeJSONBody[promptPreviewRequest](w, r)
	if !ok {
		return
	}

	display := strings.ToLower(strings.TrimSpace(req.Display))
	if display == "" {
		display = "preview"
	}

	tempTask := req.buildTask(defaultID)
	normalizedTargets, canonicalTarget := tasks.NormalizeTargets(tempTask.Target, tempTask.Targets)
	if canonicalTarget != "" {
		tempTask.Target = canonicalTarget
	}
	if len(normalizedTargets) > 0 {
		tempTask.Targets = normalizedTargets
	}

	if _, err := h.assembler.SelectPromptAssembly(tempTask.Type, tempTask.Operation); err != nil {
		writeError(w, fmt.Sprintf("Unsupported operation combination %s/%s: %v", tempTask.Type, tempTask.Operation, err), http.StatusBadRequest)
		return
	}

	sections, err := h.assembler.GeneratePromptSections(tempTask)
	if err != nil {
		writeError(w, fmt.Sprintf("Failed to get prompt sections: %v", err), http.StatusInternalServerError)
		return
	}

	assembly, err := h.assembler.AssemblePromptForTask(tempTask)
	if err != nil {
		writeError(w, fmt.Sprintf("Failed to assemble prompt: %v", err), http.StatusInternalServerError)
		return
	}
	prompt := assembly.Prompt

	response := map[string]any{
		"task_type":         tempTask.Type,
		"operation":         tempTask.Operation,
		"title":             tempTask.Title,
		"sections":          sections,
		"section_count":     len(sections),
		"sections_detailed": assembly.Sections,
		"timestamp":         timeutil.NowRFC3339(),
		"task":              tempTask,
	}

	autoSteerEligible := tempTask.AutoSteerProfileID != "" && tempTask.Type == "scenario" && tempTask.Operation == "improver"

	if autoSteerEligible && req.AutoSteerPhaseIdx != nil {
		response["auto_steer_profile_id"] = tempTask.AutoSteerProfileID
		response["auto_steer_phase_index"] = *req.AutoSteerPhaseIdx

		if h.autoSteerProfiles == nil {
			response["auto_steer_applied"] = false
			response["auto_steer_error"] = "Auto Steer profile service unavailable"
		} else {
			profile, err := h.autoSteerProfiles.GetProfile(tempTask.AutoSteerProfileID)
			if err != nil {
				response["auto_steer_applied"] = false
				response["auto_steer_error"] = fmt.Sprintf("Failed to load profile: %v", err)
			} else if *req.AutoSteerPhaseIdx < 0 || *req.AutoSteerPhaseIdx >= len(profile.Phases) {
				response["auto_steer_applied"] = false
				response["auto_steer_error"] = "Invalid phase index for profile"
			} else {
				phaseIdx := *req.AutoSteerPhaseIdx
				state := autosteer.ProfileExecutionState{
					TaskID:                tempTask.ID,
					ProfileID:             profile.ID,
					CurrentPhaseIndex:     phaseIdx,
					CurrentPhaseIteration: 0,
					AutoSteerIteration:    0,
					PhaseHistory:          []autosteer.PhaseExecution{},
					Metrics:               previewMetricsSnapshot(),
					PhaseStartMetrics:     previewMetricsSnapshot(),
					StartedAt:             time.Now(),
					LastUpdated:           time.Now(),
				}

				enhancer := autosteer.NewPromptEnhancer(filepath.Join(h.assembler.PromptsDir, "phases"))
				evaluator := autosteer.NewConditionEvaluator()
				autoSteerSection := enhancer.GenerateAutoSteerSection(&state, profile, evaluator)
				prompt = autosteer.InjectSteeringSection(prompt, autoSteerSection)
				if strings.TrimSpace(autoSteerSection) != "" {
					response["auto_steer_applied"] = true
					response["auto_steer_mode"] = profile.Phases[phaseIdx].Mode
					response["auto_steer_phase_label"] = fmt.Sprintf("Phase %d", phaseIdx+1)
				} else {
					response["auto_steer_applied"] = false
					response["auto_steer_error"] = "Auto Steer section was empty"
				}
			}
		}
	} else if autoSteerEligible && h.processor != nil && h.processor.AutoSteerIntegration() != nil {
		autoSteer := h.processor.AutoSteerIntegration()
		scenarioName := queue.GetScenarioNameFromTask(&tempTask)
		if strings.TrimSpace(scenarioName) == "" {
			scenarioName = "preview-scenario"
		}

		if err := autoSteer.InitializeAutoSteer(&tempTask, scenarioName); err != nil {
			systemlog.Warnf("Prompt preview: failed to initialize Auto Steer for temp task %s: %v", tempTask.ID, err)
			response["auto_steer_applied"] = false
			response["auto_steer_error"] = err.Error()
		} else {
			if enhancedPrompt, err := autoSteer.EnhancePrompt(&tempTask, prompt); err != nil {
				systemlog.Warnf("Prompt preview: failed to enhance prompt with Auto Steer for temp task %s: %v", tempTask.ID, err)
				response["auto_steer_applied"] = false
				response["auto_steer_error"] = err.Error()
			} else if enhancedPrompt != "" {
				prompt = enhancedPrompt
				response["auto_steer_applied"] = true
			}

			if engine := autoSteer.ExecutionEngine(); engine != nil {
				if err := engine.DeleteExecutionState(tempTask.ID); err != nil {
					systemlog.Warnf("Prompt preview: failed to clean up Auto Steer state for temp task %s: %v", tempTask.ID, err)
				}
			}
		}
	} else if tempTask.AutoSteerProfileID != "" {
		response["auto_steer_applied"] = false
		if tempTask.Type != "scenario" || tempTask.Operation != "improver" {
			response["auto_steer_error"] = "Auto Steer currently supports scenario improver tasks only"
		} else {
			response["auto_steer_error"] = "Auto Steer integration unavailable"
		}
	}

	// Apply default Progress steering when Auto Steer is not configured
	if tempTask.Type == "scenario" && tempTask.Operation == "improver" && strings.TrimSpace(tempTask.AutoSteerProfileID) == "" {
		section := h.manualOrDefaultSteeringSection(tempTask)
		prompt = autosteer.InjectSteeringSection(prompt, section)
		if strings.TrimSpace(section) != "" {
			response["default_progress_applied"] = true
			if mode := normalizeSteerMode(tempTask.SteerMode); mode.IsValid() {
				response["manual_steer_mode"] = mode
			}
		}
	}

	promptSize := len(prompt)
	promptSizeKB := float64(promptSize) / 1024.0
	promptSizeMB := promptSizeKB / 1024.0
	response["prompt_size"] = promptSize
	response["prompt_size_kb"] = fmt.Sprintf("%.2f", promptSizeKB)
	response["prompt_size_mb"] = fmt.Sprintf("%.3f", promptSizeMB)

	switch display {
	case "full", "all":
		response["prompt"] = prompt
		response["display"] = "full"
	case "first", "preview":
		response["prompt"] = prompt
		response["display"] = "preview"
		response["truncated"] = false
	case "size", "stats":
		response["display"] = "size"
	default:
		response["display"] = "summary"
		response["available_displays"] = []string{"full", "preview", "size"}
	}

	writeJSON(w, response, http.StatusOK)
}

// previewMetricsSnapshot returns a minimal metrics snapshot suitable for preview prompts.
func previewMetricsSnapshot() autosteer.MetricsSnapshot {
	now := time.Now()
	return autosteer.MetricsSnapshot{
		Timestamp:                    now,
		PhaseLoops:                   0,
		TotalLoops:                   0,
		BuildStatus:                  1,
		OperationalTargetsTotal:      0,
		OperationalTargetsPassing:    0,
		OperationalTargetsPercentage: 0,
	}
}

// manualOrDefaultSteeringSection renders the manual steer mode (if provided) or defaults to Progress.
func (h *TaskHandlers) manualOrDefaultSteeringSection(task tasks.TaskItem) string {
	if h == nil || h.assembler == nil {
		return ""
	}

	mode := normalizeSteerMode(task.SteerMode)
	if !mode.IsValid() {
		mode = autosteer.ModeProgress
	}

	phasesDir := filepath.Join(h.assembler.PromptsDir, "phases")
	return strings.TrimSpace(autosteer.NewPromptEnhancer(phasesDir).GenerateModeSection(mode))
}
