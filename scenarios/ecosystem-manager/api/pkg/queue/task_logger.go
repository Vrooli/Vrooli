package queue

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TaskLogBuffer keeps a bounded rolling log for a running or recently completed task.
type TaskLogBuffer struct {
	AgentID      string
	ProcessID    int
	Entries      []LogEntry
	LastSequence int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Completed    bool
}

// LogEntry represents a single line of task execution output.
type LogEntry struct {
	Sequence  int64     `json:"sequence"`
	Timestamp time.Time `json:"timestamp"`
	Stream    string    `json:"stream"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

// TaskLogger manages buffered logging for task executions.
// It provides streaming logs to WebSocket clients and persists logs to disk.
type TaskLogger struct {
	logs      map[string]*TaskLogBuffer
	mu        sync.RWMutex
	logsDir   string
	broadcast chan<- any
}

// NewTaskLogger creates a new task logger with the specified logs directory.
func NewTaskLogger(logsDir string, broadcast chan<- any) *TaskLogger {
	if logsDir != "" {
		if err := os.MkdirAll(logsDir, 0o755); err != nil {
			log.Printf("Warning: unable to create task logs directory %s: %v", logsDir, err)
		}
	}

	return &TaskLogger{
		logs:      make(map[string]*TaskLogBuffer),
		logsDir:   logsDir,
		broadcast: broadcast,
	}
}

// InitBuffer prepares a fresh log buffer for a task execution.
func (tl *TaskLogger) InitBuffer(taskID, agentID string, pid int) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	tl.logs[taskID] = &TaskLogBuffer{
		AgentID:      agentID,
		ProcessID:    pid,
		Entries:      make([]LogEntry, 0, TaskLogBufferInitialCapacity),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		LastSequence: 0,
	}
}

// Append stores a log entry and emits websocket updates.
func (tl *TaskLogger) Append(taskID, agentID, stream, message string) LogEntry {
	entry := LogEntry{
		Timestamp: time.Now(),
		Stream:    stream,
		Message:   message,
	}

	// Map stream to log level
	switch stream {
	case "stderr":
		entry.Level = "error"
	case "stdout":
		entry.Level = "info"
	default:
		entry.Level = "info"
	}

	tl.mu.Lock()
	buffer, exists := tl.logs[taskID]
	if !exists {
		buffer = &TaskLogBuffer{
			AgentID:   agentID,
			ProcessID: 0,
			Entries:   make([]LogEntry, 0, TaskLogBufferInitialCapacity),
			CreatedAt: time.Now(),
		}
		tl.logs[taskID] = buffer
	}
	if buffer.AgentID == "" {
		buffer.AgentID = agentID
	}
	buffer.LastSequence++
	entry.Sequence = buffer.LastSequence
	buffer.UpdatedAt = entry.Timestamp
	buffer.Entries = append(buffer.Entries, entry)
	if len(buffer.Entries) > MaxTaskLogEntries {
		buffer.Entries = buffer.Entries[len(buffer.Entries)-MaxTaskLogEntries:]
	}
	tl.mu.Unlock()

	tl.broadcastEntry(taskID, buffer.AgentID, entry)

	return entry
}

// Finalize writes the buffered log output to disk and annotates completion state.
func (tl *TaskLogger) Finalize(taskID string, completed bool) {
	var bufferCopy *TaskLogBuffer

	tl.mu.Lock()
	if buffer, exists := tl.logs[taskID]; exists {
		buffer.Completed = completed
		buffer.UpdatedAt = time.Now()
		tmp := *buffer
		tmp.Entries = append([]LogEntry(nil), buffer.Entries...)
		bufferCopy = &tmp
	}
	tl.mu.Unlock()

	if bufferCopy != nil {
		if err := tl.writeToFile(taskID, bufferCopy); err != nil {
			log.Printf("Warning: failed to persist task logs for %s: %v", taskID, err)
		}
	}
}

// Clear removes the log buffer for a task.
func (tl *TaskLogger) Clear(taskID string) {
	tl.mu.Lock()
	delete(tl.logs, taskID)
	tl.mu.Unlock()
}

// GetLogs returns log entries newer than the requested sequence number.
// Returns (entries, lastSequence, agentID, completed, processID).
func (tl *TaskLogger) GetLogs(taskID string, afterSeq int64) ([]LogEntry, int64, string, bool, int) {
	tl.mu.RLock()
	buffer, exists := tl.logs[taskID]
	tl.mu.RUnlock()

	if !exists {
		return []LogEntry{}, afterSeq, "", false, 0
	}

	entries := make([]LogEntry, 0, len(buffer.Entries))
	for _, entry := range buffer.Entries {
		if entry.Sequence > afterSeq {
			entries = append(entries, entry)
		}
	}

	return entries, buffer.LastSequence, buffer.AgentID, buffer.Completed, buffer.ProcessID
}

// HasBuffer returns true if a log buffer exists for the task.
func (tl *TaskLogger) HasBuffer(taskID string) bool {
	tl.mu.RLock()
	_, exists := tl.logs[taskID]
	tl.mu.RUnlock()
	return exists
}

// writeToFile persists task logs to disk.
func (tl *TaskLogger) writeToFile(taskID string, buffer *TaskLogBuffer) error {
	if tl.logsDir == "" || buffer == nil {
		return nil
	}

	if err := os.MkdirAll(tl.logsDir, 0o755); err != nil {
		return err
	}

	logPath := filepath.Join(tl.logsDir, fmt.Sprintf("%s.log", taskID))
	file, err := os.Create(logPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	metadata := fmt.Sprintf("Task: %s\nAgent: %s\nProcessID: %d\nCompleted: %v\nCreatedAt: %s\nUpdatedAt: %s\n\n",
		taskID, buffer.AgentID, buffer.ProcessID, buffer.Completed,
		buffer.CreatedAt.Format(time.RFC3339), buffer.UpdatedAt.Format(time.RFC3339))
	if _, err := writer.WriteString(metadata); err != nil {
		return err
	}

	for _, entry := range buffer.Entries {
		line := fmt.Sprintf("%s [%s] (%s) %s\n",
			entry.Timestamp.Format(time.RFC3339Nano), entry.Stream, entry.Level, entry.Message)
		if _, err := writer.WriteString(line); err != nil {
			return err
		}
	}

	return nil
}

// broadcastEntry sends a log entry to the broadcast channel.
func (tl *TaskLogger) broadcastEntry(taskID, agentID string, entry LogEntry) {
	if tl.broadcast == nil {
		return
	}

	select {
	case tl.broadcast <- map[string]any{
		"type": "log_entry",
		"data": map[string]any{
			"task_id":   taskID,
			"agent_id":  agentID,
			"stream":    entry.Stream,
			"level":     entry.Level,
			"message":   entry.Message,
			"sequence":  entry.Sequence,
			"timestamp": entry.Timestamp.Format(time.RFC3339Nano),
		},
	}:
	default:
		// Channel full, skip broadcast
	}
}
