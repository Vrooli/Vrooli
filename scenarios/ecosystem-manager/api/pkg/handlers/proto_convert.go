package handlers

import (
	"time"

	"github.com/ecosystem-manager/api/pkg/queue"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/ecosystem-manager/v1/domain"
)

// executionHistoryToProto converts a queue.ExecutionHistory to a proto ExecutionRecord.
// Fields are nearly 1:1; key differences: time.Time → RFC3339 string, int → int32.
func executionHistoryToProto(h queue.ExecutionHistory) *domain.ExecutionRecord {
	return &domain.ExecutionRecord{
		TaskId:              h.TaskID,
		TaskTitle:           h.TaskTitle,
		TaskType:            h.TaskType,
		TaskOperation:       h.TaskOperation,
		ExecutionId:         h.ExecutionID,
		AgentTag:            h.AgentTag,
		ProcessId:           int32(h.ProcessID),
		StartTime:           formatTimeRFC3339(h.StartTime),
		EndTime:             formatTimeRFC3339(h.EndTime),
		Duration:            h.Duration,
		Success:             h.Success,
		ExitReason:          h.ExitReason,
		PromptSize:          int32(h.PromptSize),
		PromptPath:          h.PromptPath,
		OutputPath:          h.OutputPath,
		CleanOutputPath:     h.CleanOutputPath,
		LastMessagePath:     h.LastMessagePath,
		TranscriptPath:      h.TranscriptPath,
		AutoSteerProfileId:  h.AutoSteerProfileID,
		AutoSteerIteration:  int32(h.AutoSteerIteration),
		SteerSkillIds:       h.SteerSkillIDs,
		SteerSetLabel:       h.SteerSetLabel,
		SteerPhaseIndex:     int32(h.SteerPhaseIndex),
		SteerPhaseIteration: int32(h.SteerPhaseIteration),
		SteeringSource:      h.SteeringSource,
		SteeringQueueTotal:  int32(h.SteeringQueueTotal),
		TimeoutAllowed:      h.TimeoutAllowed,
		RateLimited:         h.RateLimited,
		RetryAfter:          int32(h.RetryAfter),
	}
}

// formatTimeRFC3339 formats a time.Time as RFC3339, returning empty string for zero times.
func formatTimeRFC3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
