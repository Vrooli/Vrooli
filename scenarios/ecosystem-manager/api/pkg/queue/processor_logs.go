package queue

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// TaskLogBuffer keeps a bounded rolling log for a running or recently completed task
type TaskLogBuffer struct {
	AgentID      string
	ProcessID    int
	Entries      []LogEntry
	LastSequence int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Completed    bool
}

// LogEntry represents a single line of task execution output
type LogEntry struct {
	Sequence  int64     `json:"sequence"`
	Timestamp time.Time `json:"timestamp"`
	Stream    string    `json:"stream"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

// initTaskLogBuffer prepares a fresh log buffer for a task execution
func (qp *Processor) initTaskLogBuffer(taskID, agentID string, pid int) {
	qp.taskLogsMutex.Lock()
	defer qp.taskLogsMutex.Unlock()

	qp.taskLogs[taskID] = &TaskLogBuffer{
		AgentID:      agentID,
		ProcessID:    pid,
		Entries:      make([]LogEntry, 0, TaskLogBufferInitialCapacity),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		LastSequence: 0,
	}
}

// appendTaskLog stores a log entry and emits websocket updates
func (qp *Processor) appendTaskLog(taskID, agentID, stream, message string) LogEntry {
	entry := LogEntry{
		Timestamp: time.Now(),
		Stream:    stream,
		Level:     map[string]string{"stderr": "error"}[stream],
		Message:   message,
	}

	if entry.Level == "" {
		entry.Level = "info"
	}

	qp.taskLogsMutex.Lock()
	buffer, exists := qp.taskLogs[taskID]
	if !exists {
		buffer = &TaskLogBuffer{
			AgentID:   agentID,
			ProcessID: 0,
			Entries:   make([]LogEntry, 0, 64),
			CreatedAt: time.Now(),
		}
		qp.taskLogs[taskID] = buffer
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
	qp.taskLogsMutex.Unlock()

	qp.broadcastUpdate("log_entry", map[string]any{
		"task_id":   taskID,
		"agent_id":  buffer.AgentID,
		"stream":    entry.Stream,
		"level":     entry.Level,
		"message":   entry.Message,
		"sequence":  entry.Sequence,
		"timestamp": entry.Timestamp.Format(time.RFC3339Nano),
	})

	return entry
}

// finalizeTaskLogs writes the buffered log output to disk and annotates completion state
func (qp *Processor) finalizeTaskLogs(taskID string, completed bool) {
	var bufferCopy *TaskLogBuffer

	qp.taskLogsMutex.Lock()
	if buffer, exists := qp.taskLogs[taskID]; exists {
		buffer.Completed = completed
		buffer.UpdatedAt = time.Now()
		tmp := *buffer
		tmp.Entries = append([]LogEntry(nil), buffer.Entries...)
		bufferCopy = &tmp
	}
	qp.taskLogsMutex.Unlock()

	if bufferCopy != nil {
		if err := qp.writeTaskLogsToFile(taskID, bufferCopy); err != nil {
			log.Printf("Warning: failed to persist task logs for %s: %v", taskID, err)
		}
	}
}

// clearTaskLogs removes the log buffer for a task (used when resetting state)
func (qp *Processor) clearTaskLogs(taskID string) {
	qp.taskLogsMutex.Lock()
	delete(qp.taskLogs, taskID)
	qp.taskLogsMutex.Unlock()
}

// ResetTaskLogs removes any cached logs for a task (used when task is retried)
func (qp *Processor) ResetTaskLogs(taskID string) {
	qp.clearTaskLogs(taskID)
}

// writeTaskLogsToFile persists task logs to disk
func (qp *Processor) writeTaskLogsToFile(taskID string, buffer *TaskLogBuffer) error {
	if qp.taskLogsDir == "" || buffer == nil {
		return nil
	}

	if err := os.MkdirAll(qp.taskLogsDir, 0755); err != nil {
		return err
	}

	logPath := filepath.Join(qp.taskLogsDir, fmt.Sprintf("%s.log", taskID))
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

// GetTaskLogs returns log entries newer than the requested sequence number
func (qp *Processor) GetTaskLogs(taskID string, afterSeq int64) ([]LogEntry, int64, bool, string, bool, int) {
	qp.taskLogsMutex.RLock()
	buffer, exists := qp.taskLogs[taskID]
	qp.taskLogsMutex.RUnlock()

	isRunning := qp.IsTaskRunning(taskID)
	if !exists {
		return []LogEntry{}, afterSeq, isRunning, "", false, 0
	}

	entries := make([]LogEntry, 0, len(buffer.Entries))
	for _, entry := range buffer.Entries {
		if entry.Sequence > afterSeq {
			entries = append(entries, entry)
		}
	}

	return entries, buffer.LastSequence, isRunning, buffer.AgentID, buffer.Completed, buffer.ProcessID
}
