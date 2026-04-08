package main

import "encoding/json"

// ============================================================================
// Conversion functions: wire → API
// ============================================================================

func wireRunSummaryToAPI(w *wireRunSummary) *AgentRunSummary {
	if w == nil {
		return nil
	}
	return &AgentRunSummary{
		FilesModified: w.FilesModified,
		FilesCreated:  w.FilesCreated,
		FilesDeleted:  w.FilesDeleted,
		TokensUsed:    w.TokensUsed,
		TurnsUsed:     w.TurnsUsed,
		CostEstimate:  w.CostEstimate,
	}
}

func wireRunActionsToAPI(w *wireRunActions) *AgentRunActions {
	if w == nil {
		return nil
	}
	return &AgentRunActions{
		CanInvestigate:               w.CanInvestigate,
		CanApplyInvestigation:        w.CanApplyInvestigation,
		CanDelete:                    w.CanDelete,
		CanStop:                      w.CanStop,
		CanRetry:                     w.CanRetry,
		CanContinue:                  w.CanContinue,
		CanApprove:                   w.CanApprove,
		CanReject:                    w.CanReject,
		CanReview:                    w.CanReview,
		CanExtractRecommendations:    w.CanExtractRecommendations,
		CanRegenerateRecommendations: w.CanRegenerateRecommendations,
	}
}

func wireRunToAPI(w *wireRun) AgentRun {
	return AgentRun{
		ID:              w.ID,
		TaskID:          w.TaskID,
		SessionID:       w.SessionID,
		Status:          normalizeEnum(w.Status, "RUN_STATUS_"),
		Phase:           normalizeEnum(w.Phase, "RUN_PHASE_"),
		ProgressPercent: w.ProgressPercent,
		ErrorMsg:        w.ErrorMsg,
		ApprovalState:   normalizeEnum(w.ApprovalState, "APPROVAL_STATE_"),
		PromptPreview:   w.PromptPreview,
		SandboxID:       w.SandboxID,
		Summary:         wireRunSummaryToAPI(w.Summary),
		Actions:         wireRunActionsToAPI(w.Actions),
		CreatedAt:       w.CreatedAt,
		StartedAt:       w.StartedAt,
		EndedAt:         w.EndedAt,
	}
}

func wireRunEventToAPI(w *wireRunEvent) AgentRunEvent {
	evt := AgentRunEvent{
		ID:        w.ID,
		RunID:     w.RunID,
		Sequence:  w.Sequence,
		EventType: normalizeEventType(w.EventType),
		Timestamp: w.Timestamp,
	}

	if data := flattenEventData(w); len(data) > 0 {
		evt.Data = data
	}
	return evt
}

// flattenEventData converts oneof wire event fields into a flat map for the UI.
func flattenEventData(w *wireRunEvent) map[string]interface{} {
	data := map[string]interface{}{}
	flattenMessageData(data, w.Message)
	flattenToolCallData(data, w.ToolCall)
	flattenToolResultData(data, w.ToolResult)
	flattenErrorData(data, w.Error)
	flattenStatusData(data, w.Status)
	flattenLogData(data, w.Log)
	flattenProgressData(data, w.Progress)
	return data
}

func flattenMessageData(data map[string]interface{}, m *wireMessageData) {
	if m == nil {
		return
	}
	data["content"] = m.Content
	if m.Role != "" {
		data["role"] = m.Role
	}
}

func flattenToolCallData(data map[string]interface{}, tc *wireToolCallData) {
	if tc == nil {
		return
	}
	data["name"] = tc.ToolName
	if len(tc.Input) > 0 {
		var parsed interface{}
		if json.Unmarshal(tc.Input, &parsed) == nil {
			data["input"] = parsed
		}
	}
}

func flattenToolResultData(data map[string]interface{}, tr *wireToolResultData) {
	if tr == nil {
		return
	}
	data["result"] = tr.Output
	if tr.ToolName != "" {
		data["name"] = tr.ToolName
	}
}

func flattenErrorData(data map[string]interface{}, e *wireErrorData) {
	if e == nil {
		return
	}
	data["message"] = e.Message
	if e.Code != "" {
		data["code"] = e.Code
	}
}

func flattenStatusData(data map[string]interface{}, s *wireStatusData) {
	if s == nil {
		return
	}
	if s.NewStatus != "" {
		data["newStatus"] = normalizeEnum(s.NewStatus, "RUN_STATUS_")
	}
	if s.OldStatus != "" {
		data["oldStatus"] = normalizeEnum(s.OldStatus, "RUN_STATUS_")
	}
}

func flattenLogData(data map[string]interface{}, l *wireLogData) {
	if l == nil {
		return
	}
	data["message"] = l.Message
	if l.Level != "" {
		data["level"] = l.Level
	}
}

func flattenProgressData(data map[string]interface{}, p *wireProgressData) {
	if p == nil {
		return
	}
	data["percent"] = p.Percent
	if p.Message != "" {
		data["message"] = p.Message
	}
}

func wireFileDiffToAPI(w *wireFileDiff) AgentRunDiffFile {
	return AgentRunDiffFile{
		Path:       w.Path,
		ChangeType: w.ChangeType,
		Additions:  w.Additions,
		Deletions:  w.Deletions,
		IsBinary:   w.IsBinary,
		Patch:      w.Patch,
	}
}

func wireProfileToAPI(w *wireAgentProfile) AgentProfile {
	return AgentProfile{
		ID:          w.ID,
		Key:         w.ProfileKey,
		Name:        w.Name,
		Description: w.Description,
		Model:       w.Model,
		RunnerType:  runnerTypeToString(w.RunnerType),
	}
}
