package queue

// initTaskLogBuffer prepares a fresh log buffer for a task execution.
// Delegates to the TaskLogger component.
func (qp *Processor) initTaskLogBuffer(taskID, agentID string, pid int) {
	qp.taskLogger.InitBuffer(taskID, agentID, pid)
}

// appendTaskLog stores a log entry and emits websocket updates.
// Delegates to the TaskLogger component.
func (qp *Processor) appendTaskLog(taskID, agentID, stream, message string) LogEntry {
	return qp.taskLogger.Append(taskID, agentID, stream, message)
}

// finalizeTaskLogs writes the buffered log output to disk and annotates completion state.
// Delegates to the TaskLogger component.
func (qp *Processor) finalizeTaskLogs(taskID string, completed bool) {
	qp.taskLogger.Finalize(taskID, completed)
}

// clearTaskLogs removes the log buffer for a task.
// Delegates to the TaskLogger component.
func (qp *Processor) clearTaskLogs(taskID string) {
	qp.taskLogger.Clear(taskID)
}

// ResetTaskLogs removes any cached logs for a task (used when task is retried).
// Delegates to the TaskLogger component.
func (qp *Processor) ResetTaskLogs(taskID string) {
	qp.taskLogger.Clear(taskID)
}

// GetTaskLogs returns log entries newer than the requested sequence number.
// Delegates to the TaskLogger component.
func (qp *Processor) GetTaskLogs(taskID string, afterSeq int64) ([]LogEntry, int64, bool, string, bool, int) {
	entries, lastSeq, agentID, completed, pid := qp.taskLogger.GetLogs(taskID, afterSeq)
	isRunning := qp.IsTaskRunning(taskID)
	return entries, lastSeq, isRunning, agentID, completed, pid
}
