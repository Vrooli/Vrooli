package agentactivity

func (s *Service) dispatchStatusUpdate(record Record) {
	if s.eventDispatcher == nil {
		return
	}
	s.eventDispatcher.DispatchNodeUpdate("AgentActivity", "agent-activity/"+record.ActivityID, map[string]any{
		"activity_id":      record.ActivityID,
		"owner_type":       string(record.OwnerType),
		"owner_kind":       record.OwnerKind,
		"owner_name":       record.OwnerName,
		"execution_id":     record.ExecutionID,
		"purpose":          string(record.Purpose),
		"interaction_type": string(record.InteractionType),
		"status":           string(record.Status),
		"run_id":           record.RunID,
		"task_id":          record.TaskID,
		"requested_at":     record.RequestedAt,
	})
	s.eventDispatcher.DispatchInvalidate("flow", "operations")
}
