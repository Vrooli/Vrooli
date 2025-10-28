package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	assembly, err := h.assembler.AssemblePromptForTask(*task)
	if err != nil {
		writeError(w, fmt.Sprintf("Failed to assemble prompt: %v", err), http.StatusInternalServerError)
		return
	}
	prompt := assembly.Prompt

	// Check for cached prompt content (legacy behavior)
	promptPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s%s.txt", queue.PromptFilePrefix, taskID))
	if cachedPrompt, err := os.ReadFile(promptPath); err == nil {
		prompt = string(cachedPrompt)
		fromCache = true
		systemlog.Debugf("Using cached prompt from %s", promptPath)
	}

	assembly.Prompt = prompt

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
	}

	writeJSON(w, response, http.StatusOK)
}

// promptPreviewRequest captures optional data for assembling a preview task
type promptPreviewRequest struct {
	Task      *tasks.TaskItem `json:"task,omitempty"`
	Display   string          `json:"display,omitempty"`
	Type      string          `json:"type,omitempty"`
	Operation string          `json:"operation,omitempty"`
	Title     string          `json:"title,omitempty"`
	Category  string          `json:"category,omitempty"`
	Priority  string          `json:"priority,omitempty"`
	Notes     string          `json:"notes,omitempty"`
	Tags      []string        `json:"tags,omitempty"`
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
	if r.Notes != "" {
		task.Notes = r.Notes
	}
	if len(r.Tags) > 0 {
		task.Tags = r.Tags
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

	promptSize := len(prompt)
	promptSizeKB := float64(promptSize) / 1024.0
	promptSizeMB := promptSizeKB / 1024.0

	response := map[string]any{
		"task_type":         tempTask.Type,
		"operation":         tempTask.Operation,
		"title":             tempTask.Title,
		"sections":          sections,
		"section_count":     len(sections),
		"sections_detailed": assembly.Sections,
		"prompt_size":       promptSize,
		"prompt_size_kb":    fmt.Sprintf("%.2f", promptSizeKB),
		"prompt_size_mb":    fmt.Sprintf("%.3f", promptSizeMB),
		"timestamp":         timeutil.NowRFC3339(),
		"task":              tempTask,
	}

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
