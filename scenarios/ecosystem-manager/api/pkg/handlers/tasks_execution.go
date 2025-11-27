package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ecosystem-manager/api/pkg/queue"
	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/gorilla/mux"
)

// GetTaskLogsHandler retrieves real-time logs for a task execution
func (h *TaskHandlers) GetTaskLogsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	afterParam := r.URL.Query().Get("after")
	var afterSeq int64
	if afterParam != "" {
		seq, err := strconv.ParseInt(afterParam, 10, 64)
		if err != nil {
			writeError(w, "after must be an integer", http.StatusBadRequest)
			return
		}
		afterSeq = seq
	}

	entries, nextSeq, running, agentID, completed, processID := h.processor.GetTaskLogs(taskID, afterSeq)

	writeJSON(w, map[string]any{
		"task_id":       taskID,
		"agent_id":      agentID,
		"process_id":    processID,
		"running":       running,
		"completed":     completed,
		"next_sequence": nextSeq,
		"entries":       entries,
		"timestamp":     time.Now().Unix(),
	}, http.StatusOK)
}

// GetExecutionHistoryHandler retrieves execution history for a task
func (h *TaskHandlers) GetExecutionHistoryHandler(w http.ResponseWriter, r *http.Request) {
	task, _, ok := h.getTaskFromRequest(r, w)
	if !ok {
		return
	}
	taskID := task.ID

	// Load execution history
	history, err := h.processor.LoadExecutionHistory(taskID)
	if err != nil {
		log.Printf("ERROR: Failed to load execution history for task %s: %v", taskID, err)
		systemlog.Errorf("Failed to load execution history for task %s: %v", taskID, err)
		writeError(w, fmt.Sprintf("Failed to load execution history: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"task_id":    taskID,
		"executions": history,
		"count":      len(history),
	}, http.StatusOK)
}

// GetAllExecutionHistoryHandler retrieves execution history for all tasks
func (h *TaskHandlers) GetAllExecutionHistoryHandler(w http.ResponseWriter, r *http.Request) {
	// Load all execution history
	history, err := h.processor.LoadAllExecutionHistory()
	if err != nil {
		log.Printf("ERROR: Failed to load all execution history: %v", err)
		systemlog.Errorf("Failed to load all execution history: %v", err)
		writeError(w, fmt.Sprintf("Failed to load execution history: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"executions": history,
		"count":      len(history),
	}, http.StatusOK)
}

// GetExecutionPromptHandler retrieves the prompt file for a specific execution
func (h *TaskHandlers) GetExecutionPromptHandler(w http.ResponseWriter, r *http.Request) {
	task, _, ok := h.getTaskFromRequest(r, w)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	executionID := vars["execution_id"]
	taskID := task.ID

	// Load the prompt file from execution history
	promptPath := h.processor.GetExecutionFilePath(taskID, executionID, "prompt.txt")
	content, err := os.ReadFile(promptPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("ERROR: Execution prompt not found for task %s execution %s: %s", taskID, executionID, promptPath)
			writeError(w, "Execution prompt not found", http.StatusNotFound)
			return
		}
		log.Printf("ERROR: Failed to read prompt file for task %s execution %s: %v", taskID, executionID, err)
		systemlog.Errorf("Failed to read prompt file for task %s execution %s: %v", taskID, executionID, err)
		writeError(w, fmt.Sprintf("Failed to read prompt file: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"task_id":      taskID,
		"execution_id": executionID,
		"prompt":       string(content),
		"content":      string(content),
		"size":         len(content),
	}, http.StatusOK)
}

// GetExecutionOutputHandler retrieves the output log for a specific execution
func (h *TaskHandlers) GetExecutionOutputHandler(w http.ResponseWriter, r *http.Request) {
	task, _, ok := h.getTaskFromRequest(r, w)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	executionID := vars["execution_id"]
	taskID := task.ID

	// Prefer clean output if present; otherwise sanitize legacy timestamped logs
	cleanPath := h.processor.GetExecutionFilePath(taskID, executionID, "clean_output.txt")
	outputPath := h.processor.GetExecutionFilePath(taskID, executionID, "output.log")

	if content, err := os.ReadFile(cleanPath); err == nil {
		writeJSON(w, map[string]any{
			"task_id":      taskID,
			"execution_id": executionID,
			"output":       string(content),
			"content":      string(content),
			"size":         len(content),
			"source":       "clean_output",
		}, http.StatusOK)
		return
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("ERROR: Execution output not found for task %s execution %s: %s", taskID, executionID, outputPath)
			writeError(w, "Execution output not found", http.StatusNotFound)
			return
		}
		log.Printf("ERROR: Failed to read output file for task %s execution %s: %v", taskID, executionID, err)
		systemlog.Errorf("Failed to read output file for task %s execution %s: %v", taskID, executionID, err)
		writeError(w, fmt.Sprintf("Failed to read output file: %v", err), http.StatusInternalServerError)
		return
	}

	sanitized := stripLogPrefixes(string(content))

	writeJSON(w, map[string]any{
		"task_id":      taskID,
		"execution_id": executionID,
		"output":       sanitized,
		"content":      sanitized,
		"size":         len(sanitized),
		"source":       "output_log",
	}, http.StatusOK)
}

// GetExecutionMetadataHandler retrieves metadata for a specific execution
func (h *TaskHandlers) GetExecutionMetadataHandler(w http.ResponseWriter, r *http.Request) {
	task, _, ok := h.getTaskFromRequest(r, w)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	executionID := vars["execution_id"]
	taskID := task.ID

	// Load execution history and find the specific execution
	history, err := h.processor.LoadExecutionHistory(taskID)
	if err != nil {
		log.Printf("ERROR: Failed to load execution history for task %s (metadata request): %v", taskID, err)
		systemlog.Errorf("Failed to load execution history for task %s (metadata request): %v", taskID, err)
		writeError(w, fmt.Sprintf("Failed to load execution history: %v", err), http.StatusInternalServerError)
		return
	}

	// Find the specific execution
	var execution *queue.ExecutionHistory
	for i := range history {
		if history[i].ExecutionID == executionID {
			execution = &history[i]
			break
		}
	}

	if execution == nil {
		log.Printf("ERROR: Execution metadata not found for task %s execution %s", taskID, executionID)
		writeError(w, "Execution metadata not found", http.StatusNotFound)
		return
	}

	writeJSON(w, execution, http.StatusOK)
}

// stripLogPrefixes removes timestamp/stream prefixes from legacy output logs for readability.
func stripLogPrefixes(content string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		parts := strings.SplitN(trimmed, " ", 4)
		if len(parts) == 4 && strings.HasPrefix(parts[0], "20") && strings.HasPrefix(parts[1], "[") && strings.HasPrefix(parts[2], "(") {
			lines[i] = strings.TrimSpace(parts[3])
			continue
		}
		lines[i] = trimmed
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// GetExecutionBulkAnalysisHandler retrieves execution data with file contents for analysis
func (h *TaskHandlers) GetExecutionBulkAnalysisHandler(w http.ResponseWriter, r *http.Request) {
	task, _, ok := h.getTaskFromRequest(r, w)
	if !ok {
		return
	}

	// Parse query parameters
	limit := parseIntParam(r, "limit", 10)
	statusFilter := r.URL.Query().Get("status") // e.g., "failed,timeout,completed"
	includeFields := parseIncludeParam(r)       // e.g., "output,prompt,last_message"

	// Load execution history for task
	history, err := h.processor.LoadExecutionHistory(task.ID)
	if err != nil {
		log.Printf("ERROR: Failed to load execution history for bulk analysis (task %s): %v", task.ID, err)
		systemlog.Errorf("Failed to load execution history for bulk analysis (task %s): %v", task.ID, err)
		writeError(w, fmt.Sprintf("Failed to load execution history: %v", err), http.StatusInternalServerError)
		return
	}

	// Filter and limit
	filtered := filterExecutions(history, statusFilter, limit)

	// Load file contents for each execution
	results := make([]map[string]any, 0, len(filtered))
	for _, exec := range filtered {
		data := map[string]any{
			"metadata": exec,
		}

		// Load requested files
		files := loadExecutionFiles(h.processor, exec, includeFields)
		for key, content := range files {
			data[key] = content
		}

		results = append(results, data)
	}

	writeJSON(w, map[string]any{
		"task_id":         task.ID,
		"executions":      results,
		"count":           len(results),
		"total_available": len(history),
		"filters": map[string]any{
			"limit":  limit,
			"status": statusFilter,
		},
	}, http.StatusOK)
}

// Helper functions

// parseIntParam parses an integer query parameter with a default value
func parseIntParam(r *http.Request, key string, defaultVal int) int {
	if val := r.URL.Query().Get(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return defaultVal
}

// parseIncludeParam parses the "include" query parameter into a slice
func parseIncludeParam(r *http.Request) []string {
	include := r.URL.Query().Get("include")
	if include == "" {
		return []string{"output", "last_message"} // Default includes
	}

	fields := strings.Split(include, ",")
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		trimmed := strings.TrimSpace(field)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// filterExecutions filters execution history by status and applies limit
func filterExecutions(history []queue.ExecutionHistory, statusFilter string, limit int) []queue.ExecutionHistory {
	// If no status filter, just apply limit
	if statusFilter == "" {
		if limit > 0 && len(history) > limit {
			return history[:limit]
		}
		return history
	}

	// Build status filter map
	statuses := make(map[string]bool)
	for _, s := range strings.Split(statusFilter, ",") {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			statuses[trimmed] = true
		}
	}

	// Filter by status and apply limit
	filtered := make([]queue.ExecutionHistory, 0, limit)
	for _, exec := range history {
		if statuses[exec.ExitReason] {
			filtered = append(filtered, exec)
			if limit > 0 && len(filtered) >= limit {
				break
			}
		}
	}

	return filtered
}

// loadExecutionFiles loads requested file contents for an execution
func loadExecutionFiles(processor ProcessorAPI, exec queue.ExecutionHistory, include []string) map[string]string {
	files := make(map[string]string)

	for _, field := range include {
		switch field {
		case "output":
			// Prefer clean output if available
			if exec.CleanOutputPath != "" {
				if content, err := os.ReadFile(processor.GetExecutionFilePath(exec.TaskID, exec.ExecutionID, "clean_output.txt")); err == nil {
					files["output"] = string(content)
					continue
				}
			}
			// Fall back to sanitized regular output
			if exec.OutputPath != "" {
				if content, err := os.ReadFile(processor.GetExecutionFilePath(exec.TaskID, exec.ExecutionID, "output.log")); err == nil {
					files["output"] = stripLogPrefixes(string(content))
				}
			}

		case "prompt":
			if exec.PromptPath != "" {
				if content, err := os.ReadFile(processor.GetExecutionFilePath(exec.TaskID, exec.ExecutionID, "prompt.txt")); err == nil {
					files["prompt"] = string(content)
				}
			}

		case "last_message":
			if exec.LastMessagePath != "" {
				if content, err := os.ReadFile(processor.GetExecutionFilePath(exec.TaskID, exec.ExecutionID, "last-agent-message.txt")); err == nil {
					files["last_message"] = string(content)
				}
			}

		case "transcript":
			if exec.TranscriptPath != "" {
				if content, err := os.ReadFile(processor.GetExecutionFilePath(exec.TaskID, exec.ExecutionID, "transcript.jsonl")); err == nil {
					files["transcript"] = string(content)
				}
			}
		}
	}

	return files
}
