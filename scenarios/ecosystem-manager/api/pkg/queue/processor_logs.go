package queue

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
