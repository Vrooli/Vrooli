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
