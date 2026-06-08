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
	var manualSteerSet []string

	// Check for cached prompt content (legacy behavior)
	promptPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s%s.txt", queue.PromptFilePrefix, taskID))
	if cachedPrompt, err := os.ReadFile(promptPath); err == nil {
		prompt = string(cachedPrompt)
		fromCache = true
		systemlog.Debugf("Using cached prompt from %s", promptPath)
		_ = os.Remove(promptPath) // best effort cleanup to avoid tmp accumulation
	}

	assembly.Prompt = prompt

	// Apply default Progress steering when Auto Steer is not configured
	if task.Type == "scenario" && task.Operation == "improver" && strings.TrimSpace(task.AutoSteerProfileID) == "" {
		section := h.manualOrDefaultSteeringSection(*task)
		prompt = autosteer.InjectSteeringSection(prompt, section)
		assembly.Prompt = prompt
		if strings.TrimSpace(section) != "" {
			defaultProgressApplied = true
			if len(task.SteerSet) > 0 {
				manualSteerSet = append([]string(nil), task.SteerSet...)
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
	if len(manualSteerSet) > 0 {
		response["manual_steer_set"] = manualSteerSet
	}

	writeJSON(w, response, http.StatusOK)
}

// promptPreviewRequest captures optional data for assembling a preview task
type promptPreviewRequest struct {
	Task               *tasks.TaskItem `json:"task,omitempty"`
	Display            string          `json:"display,omitempty"`
	Type               string          `json:"type,omitempty"`
	Operation          string          `json:"operation,omitempty"`
	Category           string          `json:"category,omitempty"`
	Priority           string          `json:"priority,omitempty"`
	Notes              string          `json:"notes,omitempty"`
	Tags               []string        `json:"tags,omitempty"`
	Target             string          `json:"target,omitempty"`
	Targets            []string        `json:"targets,omitempty"`
	SteerSet           []string        `json:"steer_set,omitempty"`
	AutoSteerProfileID string          `json:"auto_steer_profile_id,omitempty"`
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
	if r.Category != "" {
		task.Category = r.Category
	}
	if r.Priority != "" {
		task.Priority = r.Priority
	}
	if len(r.SteerSet) > 0 {
		task.SteerSet = r.SteerSet
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

	// Always derive title from the canonical task fields.
	task.Title = deriveTaskTitle("", task.Operation, task.Type, task.Target)

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

	if autoSteerEligible && h.autoSteerProfiles != nil {
		profile, err := h.autoSteerProfiles.GetProfile(tempTask.AutoSteerProfileID)
		if err != nil {
			response["auto_steer_applied"] = false
			response["auto_steer_error"] = fmt.Sprintf("Failed to load profile: %v", err)
		} else {
			// The controller selects a skill from a LIVE test-genie audit at run
			// time, which is far too costly to run on a prompt preview. The
			// preview is therefore a static rendering of the objective and the
			// allowed-skill set the controller will choose from.
			section := autosteer.PreviewControllerSection(profile)
			prompt = autosteer.InjectSteeringSection(prompt, section)
			response["auto_steer_profile_id"] = profile.ID
			response["auto_steer_objective"] = profile.Objective
			response["auto_steer_allowed_skills"] = profile.AllowedSkills
			response["auto_steer_preview"] = true
			response["auto_steer_applied"] = strings.TrimSpace(section) != ""
		}
	} else if tempTask.AutoSteerProfileID != "" {
		response["auto_steer_applied"] = false
		if tempTask.Type != "scenario" || tempTask.Operation != "improver" {
			response["auto_steer_error"] = "Auto Steer currently supports scenario improver tasks only"
		} else {
			response["auto_steer_error"] = "Auto Steer profile service unavailable"
		}
	}

	// Apply default Progress steering when Auto Steer is not configured
	if tempTask.Type == "scenario" && tempTask.Operation == "improver" && strings.TrimSpace(tempTask.AutoSteerProfileID) == "" {
		section := h.manualOrDefaultSteeringSection(tempTask)
		prompt = autosteer.InjectSteeringSection(prompt, section)
		if strings.TrimSpace(section) != "" {
			response["default_progress_applied"] = true
			if len(tempTask.SteerSet) > 0 {
				response["manual_steer_set"] = tempTask.SteerSet
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

// manualOrDefaultSteeringSection renders the manual steer set (if provided) or defaults to Progress.
func (h *TaskHandlers) manualOrDefaultSteeringSection(task tasks.TaskItem) string {
	if h == nil || h.assembler == nil {
		return ""
	}

	skillSet := task.SteerSet
	if len(skillSet) == 0 {
		skillSet = []string{string(autosteer.ModeProgress)}
	}

	enhancer := autosteer.NewPromptEnhancer()
	return strings.TrimSpace(enhancer.GenerateSkillSetSection(skillSet, false, ""))
}
