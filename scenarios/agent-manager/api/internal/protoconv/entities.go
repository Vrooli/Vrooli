package protoconv

import (
	"time"

	"github.com/google/uuid"

	"agent-manager/internal/domain"

	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// =============================================================================
// AGENT PROFILE
// =============================================================================

// AgentProfileToProto converts a domain AgentProfile to proto AgentProfile.
func AgentProfileToProto(p *domain.AgentProfile) *pb.AgentProfile {
	if p == nil {
		return nil
	}
	return &pb.AgentProfile{
		Id:                   UUIDToString(p.ID),
		Name:                 p.Name,
		Description:          p.Description,
		RunnerType:           RunnerTypeToProto(p.RunnerType),
		Model:                p.Model,
		MaxTurns:             int32(p.MaxTurns),
		Timeout:              DurationToProto(p.Timeout),
		AllowedTools:         p.AllowedTools,
		DeniedTools:          p.DeniedTools,
		SkipPermissionPrompt: p.SkipPermissionPrompt,
		RequiresSandbox:      p.RequiresSandbox,
		RequiresApproval:     p.RequiresApproval,
		AllowedPaths:         p.AllowedPaths,
		DeniedPaths:          p.DeniedPaths,
		CreatedBy:            p.CreatedBy,
		CreatedAt:            TimestampToProto(p.CreatedAt),
		UpdatedAt:            TimestampToProto(p.UpdatedAt),
	}
}

// AgentProfileFromProto converts a proto AgentProfile to domain AgentProfile.
func AgentProfileFromProto(p *pb.AgentProfile) *domain.AgentProfile {
	if p == nil {
		return nil
	}
	return &domain.AgentProfile{
		ID:                   UUIDFromString(p.Id),
		Name:                 p.Name,
		Description:          p.Description,
		RunnerType:           RunnerTypeFromProto(p.RunnerType),
		Model:                p.Model,
		MaxTurns:             int(p.MaxTurns),
		Timeout:              DurationFromProto(p.Timeout),
		AllowedTools:         p.AllowedTools,
		DeniedTools:          p.DeniedTools,
		SkipPermissionPrompt: p.SkipPermissionPrompt,
		RequiresSandbox:      p.RequiresSandbox,
		RequiresApproval:     p.RequiresApproval,
		AllowedPaths:         p.AllowedPaths,
		DeniedPaths:          p.DeniedPaths,
		CreatedBy:            p.CreatedBy,
		CreatedAt:            TimestampFromProto(p.CreatedAt),
		UpdatedAt:            TimestampFromProto(p.UpdatedAt),
	}
}

// AgentProfilesToProto converts a slice of domain AgentProfile to proto.
func AgentProfilesToProto(profiles []*domain.AgentProfile) []*pb.AgentProfile {
	result := make([]*pb.AgentProfile, len(profiles))
	for i, p := range profiles {
		result[i] = AgentProfileToProto(p)
	}
	return result
}

// =============================================================================
// TASK
// =============================================================================

// TaskToProto converts a domain Task to proto Task.
func TaskToProto(t *domain.Task) *pb.Task {
	if t == nil {
		return nil
	}

	phasePromptIDs := make([]string, len(t.PhasePromptIDs))
	for i, id := range t.PhasePromptIDs {
		phasePromptIDs[i] = id.String()
	}

	attachments := make([]*pb.ContextAttachment, len(t.ContextAttachments))
	for i, a := range t.ContextAttachments {
		attachments[i] = &pb.ContextAttachment{
			Type:    a.Type,
			Path:    a.Path,
			Url:     a.URL,
			Content: a.Content,
		}
	}

	return &pb.Task{
		Id:                 UUIDToString(t.ID),
		Title:              t.Title,
		Description:        t.Description,
		ScopePath:          t.ScopePath,
		ProjectRoot:        t.ProjectRoot,
		PhasePromptIds:     phasePromptIDs,
		ContextAttachments: attachments,
		Status:             TaskStatusToProto(t.Status),
		CreatedBy:          t.CreatedBy,
		CreatedAt:          TimestampToProto(t.CreatedAt),
		UpdatedAt:          TimestampToProto(t.UpdatedAt),
	}
}

// TaskFromProto converts a proto Task to domain Task.
func TaskFromProto(t *pb.Task) *domain.Task {
	if t == nil {
		return nil
	}

	phasePromptIDs := make([]uuid.UUID, len(t.PhasePromptIds))
	for i, id := range t.PhasePromptIds {
		phasePromptIDs[i] = UUIDFromString(id)
	}

	attachments := make([]domain.ContextAttachment, len(t.ContextAttachments))
	for i, a := range t.ContextAttachments {
		attachments[i] = domain.ContextAttachment{
			Type:    a.Type,
			Path:    a.Path,
			URL:     a.Url,
			Content: a.Content,
		}
	}

	return &domain.Task{
		ID:                 UUIDFromString(t.Id),
		Title:              t.Title,
		Description:        t.Description,
		ScopePath:          t.ScopePath,
		ProjectRoot:        t.ProjectRoot,
		PhasePromptIDs:     phasePromptIDs,
		ContextAttachments: attachments,
		Status:             TaskStatusFromProto(t.Status),
		CreatedBy:          t.CreatedBy,
		CreatedAt:          TimestampFromProto(t.CreatedAt),
		UpdatedAt:          TimestampFromProto(t.UpdatedAt),
	}
}

// TasksToProto converts a slice of domain Task to proto.
func TasksToProto(tasks []*domain.Task) []*pb.Task {
	result := make([]*pb.Task, len(tasks))
	for i, t := range tasks {
		result[i] = TaskToProto(t)
	}
	return result
}

// =============================================================================
// RUN
// =============================================================================

// RunToProto converts a domain Run to proto Run.
func RunToProto(r *domain.Run) *pb.Run {
	if r == nil {
		return nil
	}

	run := &pb.Run{
		Id:              UUIDToString(r.ID),
		TaskId:          UUIDToString(r.TaskID),
		Tag:             r.Tag,
		RunMode:         RunModeToProto(r.RunMode),
		Status:          RunStatusToProto(r.Status),
		Phase:           RunPhaseToProto(r.Phase),
		ProgressPercent: int32(r.ProgressPercent),
		IdempotencyKey:  r.IdempotencyKey,
		ErrorMsg:        r.ErrorMsg,
		ApprovalState:   ApprovalStateToProto(r.ApprovalState),
		ApprovedBy:      r.ApprovedBy,
		DiffPath:        r.DiffPath,
		LogPath:         r.LogPath,
		ChangedFiles:    int32(r.ChangedFiles),
		TotalSizeBytes:  r.TotalSizeBytes,
		CreatedAt:       TimestampToProto(r.CreatedAt),
		UpdatedAt:       TimestampToProto(r.UpdatedAt),
	}

	if r.AgentProfileID != nil {
		s := r.AgentProfileID.String()
		run.AgentProfileId = &s
	}
	if r.SandboxID != nil {
		s := r.SandboxID.String()
		run.SandboxId = &s
	}
	if r.LastCheckpointID != nil {
		s := r.LastCheckpointID.String()
		run.LastCheckpointId = &s
	}
	if r.ExitCode != nil {
		i := int32(*r.ExitCode)
		run.ExitCode = &i
	}

	if r.StartedAt != nil {
		run.StartedAt = TimestampToProto(*r.StartedAt)
	}
	if r.EndedAt != nil {
		run.EndedAt = TimestampToProto(*r.EndedAt)
	}
	if r.LastHeartbeat != nil {
		run.LastHeartbeat = TimestampToProto(*r.LastHeartbeat)
	}
	if r.ApprovedAt != nil {
		run.ApprovedAt = TimestampToProto(*r.ApprovedAt)
	}

	if r.Summary != nil {
		run.Summary = &pb.RunSummary{
			Description:   r.Summary.Description,
			FilesModified: r.Summary.FilesModified,
			FilesCreated:  r.Summary.FilesCreated,
			FilesDeleted:  r.Summary.FilesDeleted,
			TokensUsed:    int32(r.Summary.TokensUsed),
			TurnsUsed:     int32(r.Summary.TurnsUsed),
			CostEstimate:  r.Summary.CostEstimate,
		}
	}

	if r.ResolvedConfig != nil {
		run.ResolvedConfig = RunConfigToProto(r.ResolvedConfig)
	}

	return run
}

// RunFromProto converts a proto Run to domain Run.
func RunFromProto(r *pb.Run) *domain.Run {
	if r == nil {
		return nil
	}

	run := &domain.Run{
		ID:              UUIDFromString(r.Id),
		TaskID:          UUIDFromString(r.TaskId),
		Tag:             r.Tag,
		RunMode:         RunModeFromProto(r.RunMode),
		Status:          RunStatusFromProto(r.Status),
		Phase:           RunPhaseFromProto(r.Phase),
		ProgressPercent: int(r.ProgressPercent),
		IdempotencyKey:  r.IdempotencyKey,
		ErrorMsg:        r.ErrorMsg,
		ApprovalState:   ApprovalStateFromProto(r.ApprovalState),
		ApprovedBy:      r.ApprovedBy,
		DiffPath:        r.DiffPath,
		LogPath:         r.LogPath,
		ChangedFiles:    int(r.ChangedFiles),
		TotalSizeBytes:  r.TotalSizeBytes,
		CreatedAt:       TimestampFromProto(r.CreatedAt),
		UpdatedAt:       TimestampFromProto(r.UpdatedAt),
	}

	// Handle optional timestamps (pointer fields)
	if r.StartedAt != nil {
		t := TimestampFromProto(r.StartedAt)
		run.StartedAt = &t
	}
	if r.EndedAt != nil {
		t := TimestampFromProto(r.EndedAt)
		run.EndedAt = &t
	}
	if r.LastHeartbeat != nil {
		t := TimestampFromProto(r.LastHeartbeat)
		run.LastHeartbeat = &t
	}
	if r.ApprovedAt != nil {
		t := TimestampFromProto(r.ApprovedAt)
		run.ApprovedAt = &t
	}

	run.AgentProfileID = OptionalStringToUUID(r.AgentProfileId)
	run.SandboxID = OptionalStringToUUID(r.SandboxId)
	run.LastCheckpointID = OptionalStringToUUID(r.LastCheckpointId)

	if r.ExitCode != nil {
		i := int(*r.ExitCode)
		run.ExitCode = &i
	}

	if r.Summary != nil {
		run.Summary = &domain.RunSummary{
			Description:   r.Summary.Description,
			FilesModified: r.Summary.FilesModified,
			FilesCreated:  r.Summary.FilesCreated,
			FilesDeleted:  r.Summary.FilesDeleted,
			TokensUsed:    int(r.Summary.TokensUsed),
			TurnsUsed:     int(r.Summary.TurnsUsed),
			CostEstimate:  r.Summary.CostEstimate,
		}
	}

	if r.ResolvedConfig != nil {
		run.ResolvedConfig = RunConfigFromProto(r.ResolvedConfig)
	}

	return run
}

// RunsToProto converts a slice of domain Run to proto.
func RunsToProto(runs []*domain.Run) []*pb.Run {
	result := make([]*pb.Run, len(runs))
	for i, r := range runs {
		result[i] = RunToProto(r)
	}
	return result
}

// =============================================================================
// RUN CONFIG
// =============================================================================

// RunConfigToProto converts a domain RunConfig to proto RunConfig.
func RunConfigToProto(c *domain.RunConfig) *pb.RunConfig {
	if c == nil {
		return nil
	}
	return &pb.RunConfig{
		RunnerType:           RunnerTypeToProto(c.RunnerType),
		Model:                c.Model,
		MaxTurns:             int32(c.MaxTurns),
		Timeout:              DurationToProto(c.Timeout),
		AllowedTools:         c.AllowedTools,
		DeniedTools:          c.DeniedTools,
		SkipPermissionPrompt: c.SkipPermissionPrompt,
		RequiresSandbox:      c.RequiresSandbox,
		RequiresApproval:     c.RequiresApproval,
		AllowedPaths:         c.AllowedPaths,
		DeniedPaths:          c.DeniedPaths,
	}
}

// RunConfigFromProto converts a proto RunConfig to domain RunConfig.
func RunConfigFromProto(c *pb.RunConfig) *domain.RunConfig {
	if c == nil {
		return nil
	}
	return &domain.RunConfig{
		RunnerType:           RunnerTypeFromProto(c.RunnerType),
		Model:                c.Model,
		MaxTurns:             int(c.MaxTurns),
		Timeout:              DurationFromProto(c.Timeout),
		AllowedTools:         c.AllowedTools,
		DeniedTools:          c.DeniedTools,
		SkipPermissionPrompt: c.SkipPermissionPrompt,
		RequiresSandbox:      c.RequiresSandbox,
		RequiresApproval:     c.RequiresApproval,
		AllowedPaths:         c.AllowedPaths,
		DeniedPaths:          c.DeniedPaths,
	}
}

// =============================================================================
// RUN EVENT
// =============================================================================

// RunEventToProto converts a domain RunEvent to proto RunEvent.
func RunEventToProto(e *domain.RunEvent) *pb.RunEvent {
	if e == nil {
		return nil
	}

	event := &pb.RunEvent{
		Id:        UUIDToString(e.ID),
		RunId:     UUIDToString(e.RunID),
		EventType: RunEventTypeToProto(e.EventType),
		Timestamp: TimestampToProto(e.Timestamp),
		Sequence:  e.Sequence,
	}

	// Convert event data based on type
	switch data := e.Data.(type) {
	case *domain.LogEventData:
		event.Data = &pb.RunEvent_Log{
			Log: &pb.LogEventData{
				Level:   data.Level,
				Message: data.Message,
			},
		}
	case *domain.MessageEventData:
		event.Data = &pb.RunEvent_Message{
			Message: &pb.MessageEventData{
				Role:    data.Role,
				Content: data.Content,
			},
		}
	case *domain.ToolCallEventData:
		event.Data = &pb.RunEvent_ToolCall{
			ToolCall: &pb.ToolCallEventData{
				ToolName: data.ToolName,
			},
		}
	case *domain.ToolResultEventData:
		event.Data = &pb.RunEvent_ToolResult{
			ToolResult: &pb.ToolResultEventData{
				ToolName:   data.ToolName,
				ToolCallId: data.ToolCallID,
				Output:     data.Output,
				Error:      data.Error,
			},
		}
	case *domain.StatusEventData:
		event.Data = &pb.RunEvent_Status{
			Status: &pb.StatusEventData{
				OldStatus: data.OldStatus,
				NewStatus: data.NewStatus,
				Reason:    data.Reason,
			},
		}
	case *domain.CostEventData:
		event.Data = &pb.RunEvent_Cost{
			Cost: &pb.CostEventData{
				InputTokens:         int32(data.InputTokens),
				OutputTokens:        int32(data.OutputTokens),
				CacheCreationTokens: int32(data.CacheCreationTokens),
				CacheReadTokens:     int32(data.CacheReadTokens),
				TotalCostUsd:        data.TotalCostUSD,
				Model:               data.Model,
			},
		}
	case *domain.ErrorEventData:
		event.Data = &pb.RunEvent_Error{
			Error: &pb.ErrorEventData{
				Code:      data.Code,
				Message:   data.Message,
				Retryable: data.Retryable,
			},
		}
	}

	return event
}

// RunEventsToProto converts a slice of domain RunEvent to proto.
func RunEventsToProto(events []*domain.RunEvent) []*pb.RunEvent {
	result := make([]*pb.RunEvent, len(events))
	for i, e := range events {
		result[i] = RunEventToProto(e)
	}
	return result
}

// =============================================================================
// RUNNER STATUS
// =============================================================================

// RunnerStatusToProto converts runner information to proto RunnerStatus.
func RunnerStatusToProto(runnerType domain.RunnerType, available bool, message, installHint string, models []string) *pb.RunnerStatus {
	return &pb.RunnerStatus{
		RunnerType:      RunnerTypeToProto(runnerType),
		Available:       available,
		Message:         message,
		InstallHint:     installHint,
		SupportedModels: models,
	}
}

// =============================================================================
// STOP ALL RESULT
// =============================================================================

// StopAllResultToProto converts an orchestration StopAllResult to proto StopAllResult.
func StopAllResultToProto(r *StopAllResult) *pb.StopAllResult {
	if r == nil {
		return nil
	}
	failures := make([]*pb.StopFailure, len(r.FailedIDs))
	for i, id := range r.FailedIDs {
		failures[i] = &pb.StopFailure{
			RunId: id,
			Error: "stop failed",
		}
	}
	return &pb.StopAllResult{
		StoppedCount: int32(r.Stopped),
		Failures:     failures,
	}
}

// StopAllResult mirrors orchestration.StopAllResult for import avoidance.
type StopAllResult struct {
	Stopped   int
	Failed    int
	Skipped   int
	FailedIDs []string
}

// =============================================================================
// APPROVE RESULT
// =============================================================================

// ApproveResultToProto converts an orchestration ApproveResult to proto ApproveResult.
func ApproveResultToProto(r *ApproveResult) *pb.ApproveResult {
	if r == nil {
		return nil
	}
	return &pb.ApproveResult{
		Success:      r.Success,
		FilesApplied: int32(r.Applied),
		CommitHash:   r.CommitHash,
		Message:      r.ErrorMsg,
	}
}

// ApproveResult mirrors orchestration.ApproveResult for import avoidance.
type ApproveResult struct {
	Success    bool
	Applied    int
	Remaining  int
	IsPartial  bool
	CommitHash string
	ErrorMsg   string
}

// =============================================================================
// PROBE RESULT
// =============================================================================

// ProbeResultToProto converts an orchestration ProbeResult to proto ProbeResult.
func ProbeResultToProto(r *ProbeResult) *pb.ProbeResult {
	if r == nil {
		return nil
	}
	details := make(map[string]string)
	if r.Response != "" {
		details["response"] = r.Response
	}
	return &pb.ProbeResult{
		Success:   r.Success,
		LatencyMs: r.DurationMs,
		Error:     r.Message,
		Details:   details,
	}
}

// ProbeResult mirrors orchestration.ProbeResult for import avoidance.
type ProbeResult struct {
	RunnerType domain.RunnerType
	Success    bool
	Message    string
	Response   string
	DurationMs int64
}

// =============================================================================
// RUN DIFF
// =============================================================================

// DiffResultToProto converts a sandbox DiffResult to proto RunDiff.
func DiffResultToProto(runID uuid.UUID, r *DiffResult) *pb.RunDiff {
	if r == nil {
		return nil
	}
	files := make([]*pb.FileDiff, len(r.Files))
	for i, f := range r.Files {
		files[i] = &pb.FileDiff{
			Path:       f.FilePath,
			ChangeType: string(f.ChangeType),
			Additions:  int32(f.LinesAdded),
			Deletions:  int32(f.LinesRemoved),
			IsBinary:   false,
		}
	}
	return &pb.RunDiff{
		RunId:       UUIDToString(runID),
		Content:     r.UnifiedDiff,
		Files:       files,
		GeneratedAt: TimestampToProto(r.Generated),
	}
}

// DiffResult mirrors sandbox.DiffResult for import avoidance.
type DiffResult struct {
	SandboxID   uuid.UUID
	Files       []FileChange
	UnifiedDiff string
	Generated   time.Time
}

// FileChange mirrors sandbox.FileChange for import avoidance.
type FileChange struct {
	ID           uuid.UUID
	FilePath     string
	ChangeType   string
	FileSize     int64
	LinesAdded   int
	LinesRemoved int
}

// =============================================================================
// ORCHESTRATION RUNNER STATUS
// =============================================================================

// OrchestratorRunnerStatusToProto converts orchestration RunnerStatus to proto.
func OrchestratorRunnerStatusToProto(r *OrchestratorRunnerStatus) *pb.RunnerStatus {
	if r == nil {
		return nil
	}
	return &pb.RunnerStatus{
		RunnerType:  RunnerTypeToProto(r.Type),
		Available:   r.Available,
		Message:     r.Message,
		InstallHint: "",
		Capabilities: &pb.RunnerCapabilities{
			SupportsStreaming:    r.Capabilities.SupportsStreaming,
			SupportsMessages:     r.Capabilities.SupportsMessages,
			SupportsToolEvents:   r.Capabilities.SupportsToolEvents,
			SupportsCostTracking: r.Capabilities.SupportsCostTracking,
			SupportsCancellation: r.Capabilities.SupportsCancellation,
			MaxTurns:             int32(r.Capabilities.MaxTurns),
		},
		SupportedModels: r.Capabilities.SupportedModels,
	}
}

// OrchestratorRunnerStatusesToProto converts a slice of runner statuses.
func OrchestratorRunnerStatusesToProto(statuses []*OrchestratorRunnerStatus) []*pb.RunnerStatus {
	result := make([]*pb.RunnerStatus, len(statuses))
	for i, s := range statuses {
		result[i] = OrchestratorRunnerStatusToProto(s)
	}
	return result
}

// OrchestratorRunnerStatus mirrors orchestration.RunnerStatus for import avoidance.
type OrchestratorRunnerStatus struct {
	Type         domain.RunnerType
	Available    bool
	Message      string
	Capabilities RunnerCapabilities
}

// RunnerCapabilities mirrors runner.Capabilities for import avoidance.
type RunnerCapabilities struct {
	SupportsMessages     bool
	SupportsToolEvents   bool
	SupportsCostTracking bool
	SupportsStreaming    bool
	SupportsCancellation bool
	MaxTurns             int
	SupportedModels      []string
}
