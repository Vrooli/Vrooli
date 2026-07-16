package protoconv

import (
	"strings"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

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
		ProfileKey:           p.ProfileKey,
		Description:          p.Description,
		RoleRef:              p.RoleRef,
		MaxTurns:             int32(p.MaxTurns),
		Timeout:              DurationToProto(p.Timeout),
		AllowedTools:         p.AllowedTools,
		DeniedTools:          p.DeniedTools,
		SkipPermissionPrompt: p.SkipPermissionPrompt,
		Features:             FeatureFlagsToProto(p.Features),
		ExtraFlags:           RunnerExtraFlagsToProto(p.ExtraFlags),
		NetworkAccess:        NetworkAccessToProto(p.NetworkAccess),
		OwnerScenario:        p.OwnerScenario,
		SourcePath:           p.SourcePath,
		SourceHash:           p.SourceHash,
		LastAppliedHash:      p.LastAppliedHash,
		SourceUpdatedAt:      TimestampToProto(p.SourceUpdatedAt),
		LocalOverride:        p.LocalOverride,
		SandboxConfig:        SandboxConfigToProto(p.SandboxConfig),
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
		ProfileKey:           p.ProfileKey,
		Description:          p.Description,
		RoleRef:              p.RoleRef,
		MaxTurns:             int(p.MaxTurns),
		Timeout:              DurationFromProto(p.Timeout),
		AllowedTools:         p.AllowedTools,
		DeniedTools:          p.DeniedTools,
		SkipPermissionPrompt: p.SkipPermissionPrompt,
		Features:             FeatureFlagsFromProto(p.Features),
		ExtraFlags:           RunnerExtraFlagsFromProto(p.ExtraFlags),
		NetworkAccess:        NetworkAccessFromProto(p.NetworkAccess),
		OwnerScenario:        p.OwnerScenario,
		SourcePath:           p.SourcePath,
		SourceHash:           p.SourceHash,
		LastAppliedHash:      p.LastAppliedHash,
		SourceUpdatedAt:      TimestampFromProto(p.SourceUpdatedAt),
		LocalOverride:        p.LocalOverride,
		SandboxConfig:        SandboxConfigFromProto(p.SandboxConfig),
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
			Type:         a.Type,
			Key:          a.Key,
			Tags:         a.Tags,
			Path:         a.Path,
			Url:          a.URL,
			Content:      a.Content,
			Label:        a.Label,
			Summary:      a.Summary,
			Format:       a.Format,
			Priority:     a.Priority,
			AttachmentId: a.AttachmentID,
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
			Type:         a.Type,
			Key:          a.Key,
			Tags:         a.Tags,
			Path:         a.Path,
			URL:          a.Url,
			Content:      a.Content,
			Label:        a.Label,
			Summary:      a.Summary,
			Format:       a.Format,
			Priority:     a.Priority,
			AttachmentID: a.AttachmentId,
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
		Id:                   UUIDToString(r.ID),
		TaskId:               UUIDToString(r.TaskID),
		Tag:                  r.Tag,
		SessionId:            r.SessionID,
		RunMode:              RunModeToProto(r.RunMode),
		ExecutionMode:        ExecutionModeToProto(r.ExecutionMode),
		WebConsoleSessionId:  r.WebConsoleSessionID,
		WebConsoleSessionUrl: r.WebConsoleSessionURL,
		Status:               RunStatusToProto(r.Status),
		Phase:                RunPhaseToProto(r.Phase),
		ProgressPercent:      int32(r.ProgressPercent),
		IdempotencyKey:       r.IdempotencyKey,
		ErrorMsg:             r.ErrorMsg,
		ApprovalState:        ApprovalStateToProto(r.ApprovalState),
		ApprovedBy:           r.ApprovedBy,
		DiffPath:             r.DiffPath,
		LogPath:              r.LogPath,
		ChangedFiles:         int32(r.ChangedFiles),
		TotalSizeBytes:       r.TotalSizeBytes,
		PromptPreview:        r.PromptPreview,
		RequestedModel:       r.RequestedModel,
		ActualModel:          r.ActualModel,
		FinalizationStatus:   RunFinalizationStatusToProto(r.FinalizationStatus),
		FinalizationError:    r.FinalizationError,
		CreatedAt:            TimestampToProto(r.CreatedAt),
		UpdatedAt:            TimestampToProto(r.UpdatedAt),
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
	if r.FinalizedAt != nil {
		run.FinalizedAt = TimestampToProto(*r.FinalizedAt)
	}

	if r.AwaitHandle != nil {
		run.AwaitHandle = AwaitHandleToProto(r.AwaitHandle)
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
			ContextTokens: int32(r.Summary.ContextTokens),
		}
	}
	if r.Result != nil {
		run.Result = RunResultToProto(r.Result)
	}

	if r.ResolvedConfig != nil {
		run.ResolvedConfig = RunConfigToProto(r.ResolvedConfig)
	}
	if r.Actions != nil {
		run.Actions = &pb.RunActions{
			CanInvestigate:               r.Actions.CanInvestigate,
			CanApplyInvestigation:        r.Actions.CanApplyInvestigation,
			CanDelete:                    r.Actions.CanDelete,
			CanStop:                      r.Actions.CanStop,
			CanRetry:                     r.Actions.CanRetry,
			CanContinue:                  r.Actions.CanContinue,
			CanContinueReason:            r.Actions.CanContinueReason,
			CanApprove:                   r.Actions.CanApprove,
			CanReject:                    r.Actions.CanReject,
			CanReview:                    r.Actions.CanReview,
			CanExtractRecommendations:    r.Actions.CanExtractRecommendations,
			CanRegenerateRecommendations: r.Actions.CanRegenerateRecommendations,
			CanResumeFromFailure:         r.Actions.CanResumeFromFailure,
			CanResumeFromFailureReason:   r.Actions.CanResumeFromFailureReason,
			FinalizationWarning:          r.Actions.FinalizationWarning,
			CanRetryFinalization:         r.Actions.CanRetryFinalization,
		}
	}

	return run
}

// AwaitHandleToProto converts a domain AwaitHandle to proto. Returns nil for a
// nil handle so a non-parked run carries no await_handle.
func AwaitHandleToProto(h *domain.AwaitHandle) *pb.AwaitHandle {
	if h == nil {
		return nil
	}
	out := &pb.AwaitHandle{
		Producer:     h.Producer,
		Key:          h.Key,
		RegisteredAt: TimestampToProto(h.RegisteredAt),
	}
	if h.Deadline != nil {
		out.Deadline = TimestampToProto(*h.Deadline)
	}
	return out
}

// AwaitHandleFromProto converts a proto AwaitHandle to domain. Returns nil for a
// nil handle.
func AwaitHandleFromProto(h *pb.AwaitHandle) *domain.AwaitHandle {
	if h == nil {
		return nil
	}
	out := &domain.AwaitHandle{
		Producer:     h.Producer,
		Key:          h.Key,
		RegisteredAt: TimestampFromProto(h.RegisteredAt),
	}
	if h.Deadline != nil {
		t := TimestampFromProto(h.Deadline)
		out.Deadline = &t
	}
	return out
}

// RunFromProto converts a proto Run to domain Run.
func RunFromProto(r *pb.Run) *domain.Run {
	if r == nil {
		return nil
	}

	run := &domain.Run{
		ID:                   UUIDFromString(r.Id),
		TaskID:               UUIDFromString(r.TaskId),
		Tag:                  r.Tag,
		SessionID:            r.SessionId,
		RunMode:              RunModeFromProto(r.RunMode),
		ExecutionMode:        ExecutionModeFromProto(r.ExecutionMode),
		WebConsoleSessionID:  r.WebConsoleSessionId,
		WebConsoleSessionURL: r.WebConsoleSessionUrl,
		Status:               RunStatusFromProto(r.Status),
		Phase:                RunPhaseFromProto(r.Phase),
		ProgressPercent:      int(r.ProgressPercent),
		IdempotencyKey:       r.IdempotencyKey,
		ErrorMsg:             r.ErrorMsg,
		ApprovalState:        ApprovalStateFromProto(r.ApprovalState),
		ApprovedBy:           r.ApprovedBy,
		FinalizationStatus:   RunFinalizationStatusFromProto(r.FinalizationStatus),
		FinalizationError:    r.FinalizationError,
		DiffPath:             r.DiffPath,
		LogPath:              r.LogPath,
		ChangedFiles:         int(r.ChangedFiles),
		TotalSizeBytes:       r.TotalSizeBytes,
		CreatedAt:            TimestampFromProto(r.CreatedAt),
		UpdatedAt:            TimestampFromProto(r.UpdatedAt),
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
	if r.FinalizedAt != nil {
		t := TimestampFromProto(r.FinalizedAt)
		run.FinalizedAt = &t
	}

	if r.AwaitHandle != nil {
		run.AwaitHandle = AwaitHandleFromProto(r.AwaitHandle)
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
			ContextTokens: int(r.Summary.ContextTokens),
		}
	}
	if r.Result != nil {
		run.Result = RunResultFromProto(r.Result)
	}

	if r.ResolvedConfig != nil {
		run.ResolvedConfig = RunConfigFromProto(r.ResolvedConfig)
	}
	if r.Actions != nil {
		run.Actions = &domain.RunActions{
			CanInvestigate:               r.Actions.CanInvestigate,
			CanApplyInvestigation:        r.Actions.CanApplyInvestigation,
			CanDelete:                    r.Actions.CanDelete,
			CanStop:                      r.Actions.CanStop,
			CanRetry:                     r.Actions.CanRetry,
			CanContinue:                  r.Actions.CanContinue,
			CanContinueReason:            r.Actions.CanContinueReason,
			CanApprove:                   r.Actions.CanApprove,
			CanReject:                    r.Actions.CanReject,
			CanReview:                    r.Actions.CanReview,
			CanExtractRecommendations:    r.Actions.CanExtractRecommendations,
			CanRegenerateRecommendations: r.Actions.CanRegenerateRecommendations,
			CanResumeFromFailure:         r.Actions.CanResumeFromFailure,
			CanResumeFromFailureReason:   r.Actions.CanResumeFromFailureReason,
			FinalizationWarning:          r.Actions.FinalizationWarning,
			CanRetryFinalization:         r.Actions.CanRetryFinalization,
		}
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
		RoleRef:              c.RoleRef,
		MaxTurns:             int32(c.MaxTurns),
		Timeout:              DurationToProto(c.Timeout),
		AllowedTools:         c.AllowedTools,
		DeniedTools:          c.DeniedTools,
		SkipPermissionPrompt: c.SkipPermissionPrompt,
		Features:             FeatureFlagsToProto(c.Features),
		ExtraFlags:           RunnerExtraFlagsToProto(c.ExtraFlags),
		NetworkAccess:        NetworkAccessToProto(c.NetworkAccess),
		PolicySnapshot:       ExecutionPolicySnapshotToProto(c.PolicySnapshot),
		ResultSpec:           ResultSpecToProto(c.ResultSpec),
		SandboxConfig:        SandboxConfigToProto(c.SandboxConfig),
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
		RoleRef:              c.RoleRef,
		MaxTurns:             int(c.MaxTurns),
		Timeout:              DurationFromProto(c.Timeout),
		AllowedTools:         c.AllowedTools,
		DeniedTools:          c.DeniedTools,
		SkipPermissionPrompt: c.SkipPermissionPrompt,
		Features:             FeatureFlagsFromProto(c.Features),
		ExtraFlags:           RunnerExtraFlagsFromProto(c.ExtraFlags),
		NetworkAccess:        NetworkAccessFromProto(c.NetworkAccess),
		PolicySnapshot:       ExecutionPolicySnapshotFromProto(c.PolicySnapshot),
		ResultSpec:           ResultSpecFromProto(c.ResultSpec),
		SandboxConfig:        SandboxConfigFromProto(c.SandboxConfig),
		AllowedPaths:         c.AllowedPaths,
		DeniedPaths:          c.DeniedPaths,
	}
}

func ResultSpecToProto(spec *domain.ResultSpec) *pb.ResultSpec {
	if spec == nil {
		return nil
	}
	return &pb.ResultSpec{
		Version: spec.Version, Kind: ResultSpecKindToProto(spec.Kind),
		Schema: append([]byte(nil), spec.Schema...), SchemaDigest: spec.SchemaDigest,
		ClassificationValues: append([]string(nil), spec.ClassificationValues...),
		ExtractionMode:       StructuredExtractionModeToProto(spec.ExtractionMode), ExtractionRole: spec.ExtractionRole,
	}
}

func ResultSpecFromProto(spec *pb.ResultSpec) *domain.ResultSpec {
	if spec == nil {
		return nil
	}
	return &domain.ResultSpec{
		Version: spec.Version, Kind: ResultSpecKindFromProto(spec.Kind),
		Schema: append([]byte(nil), spec.Schema...), SchemaDigest: spec.SchemaDigest,
		ClassificationValues: append([]string(nil), spec.ClassificationValues...),
		ExtractionMode:       StructuredExtractionModeFromProto(spec.ExtractionMode), ExtractionRole: spec.ExtractionRole,
	}
}

func ResultSpecKindToProto(kind domain.ResultSpecKind) pb.ResultSpecKind {
	switch kind {
	case domain.ResultSpecKindNone:
		return pb.ResultSpecKind_RESULT_SPEC_KIND_NONE
	case domain.ResultSpecKindJSONSchema:
		return pb.ResultSpecKind_RESULT_SPEC_KIND_JSON_SCHEMA
	case domain.ResultSpecKindClassification:
		return pb.ResultSpecKind_RESULT_SPEC_KIND_CLASSIFICATION
	default:
		return pb.ResultSpecKind_RESULT_SPEC_KIND_UNSPECIFIED
	}
}

func ResultSpecKindFromProto(kind pb.ResultSpecKind) domain.ResultSpecKind {
	switch kind {
	case pb.ResultSpecKind_RESULT_SPEC_KIND_NONE:
		return domain.ResultSpecKindNone
	case pb.ResultSpecKind_RESULT_SPEC_KIND_JSON_SCHEMA:
		return domain.ResultSpecKindJSONSchema
	case pb.ResultSpecKind_RESULT_SPEC_KIND_CLASSIFICATION:
		return domain.ResultSpecKindClassification
	default:
		return ""
	}
}

func StructuredExtractionModeToProto(mode domain.StructuredExtractionMode) pb.StructuredExtractionMode {
	switch mode {
	case domain.StructuredExtractionDeterministic:
		return pb.StructuredExtractionMode_STRUCTURED_EXTRACTION_MODE_DETERMINISTIC_ONLY
	case domain.StructuredExtractionConstrained:
		return pb.StructuredExtractionMode_STRUCTURED_EXTRACTION_MODE_CONSTRAINED_FALLBACK
	default:
		return pb.StructuredExtractionMode_STRUCTURED_EXTRACTION_MODE_UNSPECIFIED
	}
}

func StructuredExtractionModeFromProto(mode pb.StructuredExtractionMode) domain.StructuredExtractionMode {
	switch mode {
	case pb.StructuredExtractionMode_STRUCTURED_EXTRACTION_MODE_DETERMINISTIC_ONLY:
		return domain.StructuredExtractionDeterministic
	case pb.StructuredExtractionMode_STRUCTURED_EXTRACTION_MODE_CONSTRAINED_FALLBACK:
		return domain.StructuredExtractionConstrained
	default:
		return ""
	}
}

// ExecutionPolicySnapshotToProto exposes the run-owned immutable policy
// decision through run detail without reconstructing it from current policy.
func ExecutionPolicySnapshotToProto(snapshot *domain.ExecutionPolicySnapshot) *pb.ExecutionPolicySnapshot {
	if snapshot == nil {
		return nil
	}
	candidates := make([]*pb.ExecutionCandidate, 0, len(snapshot.Candidates))
	for _, candidate := range snapshot.Candidates {
		candidates = append(candidates, ExecutionCandidateToProto(candidate))
	}
	return &pb.ExecutionPolicySnapshot{
		CatalogDigest:     snapshot.CatalogDigest,
		RoleRef:           snapshot.RoleRef,
		Candidates:        candidates,
		SelectedIndex:     int32(snapshot.SelectedIndex),
		SelectedCandidate: ExecutionCandidateToProto(snapshot.SelectedCandidate),
		Explanation:       PolicyResolutionExplanationToProto(snapshot.Explanation),
	}
}

// ExecutionPolicySnapshotFromProto converts a persisted policy decision from
// the generated API contract into its domain representation.
func ExecutionPolicySnapshotFromProto(snapshot *pb.ExecutionPolicySnapshot) *domain.ExecutionPolicySnapshot {
	if snapshot == nil {
		return nil
	}
	candidates := make([]domain.ExecutionCandidate, 0, len(snapshot.Candidates))
	for _, candidate := range snapshot.Candidates {
		if candidate != nil {
			candidates = append(candidates, ExecutionCandidateFromProto(candidate))
		}
	}
	return &domain.ExecutionPolicySnapshot{
		CatalogDigest:     snapshot.CatalogDigest,
		RoleRef:           snapshot.RoleRef,
		Candidates:        candidates,
		SelectedIndex:     int(snapshot.SelectedIndex),
		SelectedCandidate: ExecutionCandidateFromProto(snapshot.SelectedCandidate),
		Explanation:       PolicyResolutionExplanationFromProto(snapshot.Explanation),
	}
}

func ExecutionCandidateToProto(candidate domain.ExecutionCandidate) *pb.ExecutionCandidate {
	return &pb.ExecutionCandidate{
		RunnerType:    RunnerTypeToProto(candidate.RunnerType),
		SelectionType: ModelSelectionTypeToProto(candidate.SelectionType),
		Model:         candidate.Model,
		ResourceRole:  candidate.ResourceRole,
		Fallbacks:     append([]string(nil), candidate.Fallbacks...),
		Available:     candidate.Available,
		FailureCode:   candidate.FailureCode,
		Failure:       candidate.Failure,
		Provenance:    &pb.ResourceProvenance{Source: candidate.Provenance.Source, ObservedAt: candidate.Provenance.ObservedAt},
		Enforcement:   &pb.PermissionEnforcement{Permissions: candidate.Enforcement.Permissions, Caveats: append([]string(nil), candidate.Enforcement.Caveats...)},
		PolicyPath:    candidate.PolicyPath,
		PolicyDigest:  candidate.PolicyDigest,
	}
}

func ExecutionCandidateFromProto(candidate *pb.ExecutionCandidate) domain.ExecutionCandidate {
	if candidate == nil {
		return domain.ExecutionCandidate{}
	}
	result := domain.ExecutionCandidate{
		RunnerType:    RunnerTypeFromProto(candidate.RunnerType),
		SelectionType: ModelSelectionTypeFromProto(candidate.SelectionType),
		Model:         candidate.Model,
		ResourceRole:  candidate.ResourceRole,
		Fallbacks:     append([]string(nil), candidate.Fallbacks...),
		Available:     candidate.Available,
		FailureCode:   candidate.FailureCode,
		Failure:       candidate.Failure,
		PolicyPath:    candidate.PolicyPath,
		PolicyDigest:  candidate.PolicyDigest,
	}
	if candidate.Provenance != nil {
		result.Provenance = domain.ResourceProvenance{Source: candidate.Provenance.Source, ObservedAt: candidate.Provenance.ObservedAt}
	}
	if candidate.Enforcement != nil {
		result.Enforcement = domain.PermissionEnforcement{Permissions: candidate.Enforcement.Permissions, Caveats: append([]string(nil), candidate.Enforcement.Caveats...)}
	}
	return result
}

func PolicyResolutionExplanationToProto(explanation domain.PolicyResolutionExplanation) *pb.PolicyResolutionExplanation {
	preflight := make([]*pb.CandidatePreflight, 0, len(explanation.Preflight))
	for _, check := range explanation.Preflight {
		preflight = append(preflight, &pb.CandidatePreflight{
			Index:     int32(check.Index),
			Candidate: ExecutionCandidateToProto(check.Candidate),
			Available: check.Available,
			Reason:    check.Reason,
		})
	}
	return &pb.PolicyResolutionExplanation{
		Source:           explanation.Source,
		Summary:          explanation.Summary,
		RequestedRoleRef: explanation.RequestedRoleRef,
		Preflight:        preflight,
	}
}

func PolicyResolutionExplanationFromProto(explanation *pb.PolicyResolutionExplanation) domain.PolicyResolutionExplanation {
	if explanation == nil {
		return domain.PolicyResolutionExplanation{}
	}
	preflight := make([]domain.CandidatePreflight, 0, len(explanation.Preflight))
	for _, check := range explanation.Preflight {
		if check == nil {
			continue
		}
		preflight = append(preflight, domain.CandidatePreflight{
			Index:     int(check.Index),
			Candidate: ExecutionCandidateFromProto(check.Candidate),
			Available: check.Available,
			Reason:    check.Reason,
		})
	}
	return domain.PolicyResolutionExplanation{
		Source:           explanation.Source,
		Summary:          explanation.Summary,
		RequestedRoleRef: explanation.RequestedRoleRef,
		Preflight:        preflight,
	}
}

// =============================================================================
// FEATURE FLAGS
// =============================================================================

// FeatureFlagsToProto converts domain FeatureFlags to proto FeatureFlags.
func FeatureFlagsToProto(f domain.FeatureFlags) *pb.FeatureFlags {
	if f.IsZero() {
		return nil
	}
	return &pb.FeatureFlags{EnableBrowser: f.EnableBrowser}
}

// FeatureFlagsFromProto converts proto FeatureFlags to domain FeatureFlags.
func FeatureFlagsFromProto(f *pb.FeatureFlags) domain.FeatureFlags {
	if f == nil {
		return domain.FeatureFlags{}
	}
	return domain.FeatureFlags{EnableBrowser: f.EnableBrowser}
}

// RunnerExtraFlagsToProto converts domain RunnerExtraFlags to proto map.
func RunnerExtraFlagsToProto(flags domain.RunnerExtraFlags) map[string]*pb.ExtraFlagList {
	if len(flags) == 0 {
		return nil
	}
	result := make(map[string]*pb.ExtraFlagList, len(flags))
	for rt, flagList := range flags {
		result[string(rt)] = &pb.ExtraFlagList{Flags: flagList}
	}
	return result
}

// RunnerExtraFlagsFromProto converts proto map to domain RunnerExtraFlags.
func RunnerExtraFlagsFromProto(flags map[string]*pb.ExtraFlagList) domain.RunnerExtraFlags {
	if len(flags) == 0 {
		return nil
	}
	result := make(domain.RunnerExtraFlags, len(flags))
	for rt, flagList := range flags {
		if flagList != nil && len(flagList.Flags) > 0 {
			result[domain.RunnerType(rt)] = flagList.Flags
		}
	}
	return result
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
		var pbAttachments []*pb.MessageAttachmentInfo
		for _, att := range data.Attachments {
			pbAttachments = append(pbAttachments, &pb.MessageAttachmentInfo{
				Id:          att.ID,
				FileName:    att.FileName,
				ContentType: att.ContentType,
				Url:         att.URL,
			})
		}
		event.Data = &pb.RunEvent_Message{
			Message: &pb.MessageEventData{
				Role:               data.Role,
				Content:            data.Content,
				Attachments:        pbAttachments,
				MessageId:          data.MessageID,
				ConversationId:     data.ConversationID,
				TurnId:             data.TurnID,
				ProviderOrigin:     data.ProviderOrigin,
				CompletionReason:   data.CompletionReason,
				Terminal:           data.Terminal,
				ParentMessageId:    data.ParentMessageID,
				ProviderEventType:  data.ProviderEventType,
				RawEvidenceRef:     data.RawEvidenceRef,
				EvidenceOnly:       data.EvidenceOnly,
				EvidenceForEventId: data.EvidenceForEventID,
			},
		}
	case *domain.MessageDeletedEventData:
		event.Data = &pb.RunEvent_MessageDeleted{
			MessageDeleted: &pb.MessageDeletedEventData{
				TargetEventId: data.TargetEventID,
			},
		}
	case *domain.ToolCallEventData:
		var input *structpb.Struct
		if len(data.Input) > 0 {
			if parsed, err := structpb.NewStruct(data.Input); err == nil {
				input = parsed
			}
		}
		event.Data = &pb.RunEvent_ToolCall{
			ToolCall: &pb.ToolCallEventData{
				ToolName:   data.ToolName,
				ToolCallId: data.ToolCallID,
				Input:      input,
			},
		}
	case *domain.ToolResultEventData:
		event.Data = &pb.RunEvent_ToolResult{
			ToolResult: &pb.ToolResultEventData{
				ToolName:   data.ToolName,
				ToolCallId: data.ToolCallID,
				Output:     data.Output,
				Error:      data.Error,
				Success:    data.Success,
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
	case *domain.MetricEventData:
		event.Data = &pb.RunEvent_Metric{
			Metric: &pb.MetricEventData{
				Name:  data.Name,
				Value: data.Value,
				Unit:  data.Unit,
				Tags:  data.Tags,
			},
		}
	case *domain.ArtifactEventData:
		event.Data = &pb.RunEvent_Artifact{
			Artifact: &pb.ArtifactEventData{
				Type:     data.Type,
				Path:     data.Path,
				Size:     data.Size,
				MimeType: data.MimeType,
			},
		}
	case *domain.CostEventData:
		event.Data = &pb.RunEvent_Cost{
			Cost: &pb.CostEventData{
				InputTokens:           int32(data.InputTokens),
				OutputTokens:          int32(data.OutputTokens),
				CacheCreationTokens:   int32(data.CacheCreationTokens),
				CacheReadTokens:       int32(data.CacheReadTokens),
				TotalCostUsd:          data.TotalCostUSD,
				ServiceTier:           data.ServiceTier,
				Model:                 data.Model,
				WebSearchRequests:     int32(data.WebSearchRequests),
				ServerToolUseRequests: int32(data.ServerToolUseRequests),
			},
		}
	case *domain.ProgressEventData:
		event.Data = &pb.RunEvent_Progress{
			Progress: &pb.ProgressEventData{
				Phase:              RunPhaseToProto(data.Phase),
				PercentComplete:    int32(data.PercentComplete),
				CurrentAction:      data.CurrentAction,
				TurnsCompleted:     int32(data.TurnsCompleted),
				TurnsTotal:         int32(data.TurnsTotal),
				TokensUsed:         int32(data.TokensUsed),
				ElapsedSeconds:     data.ElapsedSeconds,
				EstimatedRemaining: data.EstimatedRemaining,
			},
		}
	case *domain.RateLimitEventData:
		var resetTime *timestamppb.Timestamp
		if data.ResetTime != nil {
			resetTime = TimestampToProto(*data.ResetTime)
		}
		event.Data = &pb.RunEvent_RateLimit{
			RateLimit: &pb.RateLimitEventData{
				LimitType:   data.LimitType,
				ResetTime:   resetTime,
				RetryAfter:  int32(data.RetryAfter),
				CurrentUsed: int32(data.CurrentUsed),
				Limit:       int32(data.Limit),
				Message:     data.Message,
			},
		}
	case *domain.ErrorEventData:
		var details *structpb.Struct
		if len(data.Details) > 0 {
			if parsed, err := structpb.NewStruct(data.Details); err == nil {
				details = parsed
			}
		}
		event.Data = &pb.RunEvent_Error{
			Error: &pb.ErrorEventData{
				Code:       data.Code,
				Message:    data.Message,
				Retryable:  data.Retryable,
				Recovery:   RecoveryActionToProto(data.Recovery),
				StackTrace: data.StackTrace,
				Details:    details,
			},
		}
	}

	return event
}

// RunResultToProto converts the canonical terminal result.
func RunResultToProto(result *domain.RunResult) *pb.RunResult {
	if result == nil {
		return nil
	}
	candidates := make([]*pb.FinalOutputCandidate, 0, len(result.Candidates))
	for _, candidate := range result.Candidates {
		candidates = append(candidates, &pb.FinalOutputCandidate{
			Id: candidate.ID, EventId: candidate.EventID, Sequence: candidate.Sequence,
			Content: candidate.Content, MessageId: candidate.MessageID,
			ConversationId: candidate.ConversationID, TurnId: candidate.TurnID,
			ProviderOrigin: candidate.ProviderOrigin, CompletionReason: candidate.CompletionReason,
			Terminal: candidate.Terminal, ParentMessageId: candidate.ParentMessageID,
			ProviderEventType: candidate.ProviderEventType, RawEvidenceRef: candidate.RawEvidenceRef,
			EvidenceTier: int32(candidate.EvidenceTier),
		})
	}
	return &pb.RunResult{
		FinalOutput: result.FinalOutput,
		Selection: &pb.FinalOutputSelection{
			Status:              FinalOutputSelectionStatusToProto(result.Selection.Status),
			SelectedCandidateId: result.Selection.SelectedCandidateID,
			Rule:                result.Selection.Rule,
			AlgorithmVersion:    result.Selection.AlgorithmVersion,
			Evidence:            result.Selection.Evidence,
		},
		Candidates:     candidates,
		Success:        result.Success,
		ExitCode:       int32(result.ExitCode),
		TerminalReason: result.TerminalReason,
		Structured:     StructuredResultToProto(result.Structured),
	}
}

func StructuredResultToProto(result *domain.StructuredResult) *pb.StructuredResult {
	if result == nil {
		return nil
	}
	diagnostics := make([]*pb.StructuredDiagnostic, 0, len(result.Diagnostics))
	for _, diagnostic := range result.Diagnostics {
		diagnostics = append(diagnostics, &pb.StructuredDiagnostic{Code: diagnostic.Code, Path: diagnostic.Path, Message: diagnostic.Message})
	}
	var extractor *pb.StructuredExtractionProvenance
	if result.Extractor != nil {
		extractor = &pb.StructuredExtractionProvenance{
			RoleRef: result.Extractor.RoleRef, Provider: result.Extractor.Provider, Model: result.Extractor.Model,
			PolicySnapshot: ExecutionPolicySnapshotToProto(result.Extractor.PolicySnapshot),
		}
	}
	return &pb.StructuredResult{
		Status: StructuredResultStatusToProto(result.Status), SpecKind: ResultSpecKindToProto(result.SpecKind),
		SchemaDigest: result.SchemaDigest, Value: append([]byte(nil), result.Value...), Method: result.Method,
		SourceCandidateId: result.SourceCandidateID, Extractor: extractor, Diagnostics: diagnostics,
	}
}

func FinalOutputSelectionStatusToProto(status domain.FinalOutputSelectionStatus) pb.FinalOutputSelectionStatus {
	switch status {
	case domain.FinalOutputSelectionSelected:
		return pb.FinalOutputSelectionStatus_FINAL_OUTPUT_SELECTION_STATUS_SELECTED
	case domain.FinalOutputSelectionAmbiguous:
		return pb.FinalOutputSelectionStatus_FINAL_OUTPUT_SELECTION_STATUS_AMBIGUOUS
	case domain.FinalOutputSelectionUnavailable:
		return pb.FinalOutputSelectionStatus_FINAL_OUTPUT_SELECTION_STATUS_UNAVAILABLE
	default:
		return pb.FinalOutputSelectionStatus_FINAL_OUTPUT_SELECTION_STATUS_UNSPECIFIED
	}
}

func RunResultFromProto(result *pb.RunResult) *domain.RunResult {
	if result == nil {
		return nil
	}
	candidates := make([]domain.FinalOutputCandidate, 0, len(result.Candidates))
	for _, candidate := range result.Candidates {
		if candidate == nil {
			continue
		}
		candidates = append(candidates, domain.FinalOutputCandidate{
			ID: candidate.Id, EventID: candidate.EventId, Sequence: candidate.Sequence,
			Content: candidate.Content, MessageID: candidate.MessageId,
			ConversationID: candidate.ConversationId, TurnID: candidate.TurnId,
			ProviderOrigin: candidate.ProviderOrigin, CompletionReason: candidate.CompletionReason,
			Terminal: candidate.Terminal, ParentMessageID: candidate.ParentMessageId,
			ProviderEventType: candidate.ProviderEventType, RawEvidenceRef: candidate.RawEvidenceRef,
			EvidenceTier: int(candidate.EvidenceTier),
		})
	}
	selection := domain.FinalOutputSelection{}
	if result.Selection != nil {
		selection = domain.FinalOutputSelection{
			Status:              FinalOutputSelectionStatusFromProto(result.Selection.Status),
			SelectedCandidateID: result.Selection.SelectedCandidateId,
			Rule:                result.Selection.Rule,
			AlgorithmVersion:    result.Selection.AlgorithmVersion,
			Evidence:            result.Selection.Evidence,
		}
	}
	return &domain.RunResult{
		FinalOutput: result.FinalOutput, Selection: selection, Candidates: candidates,
		Success: result.Success, ExitCode: int(result.ExitCode), TerminalReason: result.TerminalReason,
		Structured: StructuredResultFromProto(result.Structured),
	}
}

func StructuredResultFromProto(result *pb.StructuredResult) *domain.StructuredResult {
	if result == nil {
		return nil
	}
	diagnostics := make([]domain.StructuredDiagnostic, 0, len(result.Diagnostics))
	for _, diagnostic := range result.Diagnostics {
		if diagnostic != nil {
			diagnostics = append(diagnostics, domain.StructuredDiagnostic{Code: diagnostic.Code, Path: diagnostic.Path, Message: diagnostic.Message})
		}
	}
	var extractor *domain.StructuredExtractionProvenance
	if result.Extractor != nil {
		extractor = &domain.StructuredExtractionProvenance{
			RoleRef: result.Extractor.RoleRef, Provider: result.Extractor.Provider, Model: result.Extractor.Model,
			PolicySnapshot: ExecutionPolicySnapshotFromProto(result.Extractor.PolicySnapshot),
		}
	}
	return &domain.StructuredResult{
		Status: StructuredResultStatusFromProto(result.Status), SpecKind: ResultSpecKindFromProto(result.SpecKind),
		SchemaDigest: result.SchemaDigest, Value: append([]byte(nil), result.Value...), Method: result.Method,
		SourceCandidateID: result.SourceCandidateId, Extractor: extractor, Diagnostics: diagnostics,
	}
}

func StructuredResultStatusToProto(status domain.StructuredResultStatus) pb.StructuredResultStatus {
	switch status {
	case domain.StructuredResultSuccess:
		return pb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_SUCCESS
	case domain.StructuredResultUnavailable:
		return pb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_UNAVAILABLE
	case domain.StructuredResultInvalid:
		return pb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_INVALID
	case domain.StructuredResultAmbiguous:
		return pb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_AMBIGUOUS
	case domain.StructuredResultAbstained:
		return pb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_ABSTAINED
	default:
		return pb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_UNSPECIFIED
	}
}

func StructuredResultStatusFromProto(status pb.StructuredResultStatus) domain.StructuredResultStatus {
	switch status {
	case pb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_SUCCESS:
		return domain.StructuredResultSuccess
	case pb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_UNAVAILABLE:
		return domain.StructuredResultUnavailable
	case pb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_INVALID:
		return domain.StructuredResultInvalid
	case pb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_AMBIGUOUS:
		return domain.StructuredResultAmbiguous
	case pb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_ABSTAINED:
		return domain.StructuredResultAbstained
	default:
		return ""
	}
}

func FinalOutputSelectionStatusFromProto(status pb.FinalOutputSelectionStatus) domain.FinalOutputSelectionStatus {
	switch status {
	case pb.FinalOutputSelectionStatus_FINAL_OUTPUT_SELECTION_STATUS_SELECTED:
		return domain.FinalOutputSelectionSelected
	case pb.FinalOutputSelectionStatus_FINAL_OUTPUT_SELECTION_STATUS_AMBIGUOUS:
		return domain.FinalOutputSelectionAmbiguous
	case pb.FinalOutputSelectionStatus_FINAL_OUTPUT_SELECTION_STATUS_UNAVAILABLE:
		return domain.FinalOutputSelectionUnavailable
	default:
		return ""
	}
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
		Remaining:    int32(r.Remaining),
		IsPartial:    r.IsPartial,
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

	// Extract per-file patches from the unified diff.
	patchByPath := splitUnifiedDiff(r.UnifiedDiff)

	files := make([]*pb.FileDiff, len(r.Files))
	for i, f := range r.Files {
		patch := f.Patch
		if patch == "" {
			patch = lookupPatch(patchByPath, f.FilePath)
		}
		files[i] = &pb.FileDiff{
			Id:         UUIDToString(f.ID),
			Path:       f.FilePath,
			ChangeType: string(f.ChangeType),
			Additions:  int32(f.LinesAdded),
			Deletions:  int32(f.LinesRemoved),
			IsBinary:   false,
			Patch:      patch,
		}
	}
	return &pb.RunDiff{
		RunId:       UUIDToString(runID),
		Content:     r.UnifiedDiff,
		Files:       files,
		GeneratedAt: TimestampToProto(r.Generated),
	}
}

// lookupPatch finds a patch for filePath in the map. It first tries an exact
// match, then falls back to a suffix match. This handles the common case where
// the unified diff uses project-root-relative paths (e.g.
// "scenarios/foo/api/main.go") but file metadata uses sandbox-scope-relative
// paths (e.g. "api/main.go").
func lookupPatch(patchByPath map[string]string, filePath string) string {
	if p, ok := patchByPath[filePath]; ok {
		return p
	}
	suffix := "/" + filePath
	for k, v := range patchByPath {
		if strings.HasSuffix(k, suffix) {
			return v
		}
	}
	return ""
}

// splitUnifiedDiff splits a unified diff string into per-file patches.
// It looks for "diff --git a/... b/..." markers and maps each section
// to the file path (the "b/" side).
func splitUnifiedDiff(unified string) map[string]string {
	if unified == "" {
		return nil
	}

	result := make(map[string]string)
	lines := strings.Split(unified, "\n")

	var currentPath string
	var currentStart int
	inSection := false

	for i, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			// Flush previous section.
			if inSection && currentPath != "" {
				result[currentPath] = strings.Join(lines[currentStart:i], "\n")
			}
			// Extract the "b/" path from "diff --git a/foo b/foo".
			currentPath = extractDiffPath(line)
			currentStart = i
			inSection = true
		}
	}
	// Flush last section.
	if inSection && currentPath != "" {
		result[currentPath] = strings.Join(lines[currentStart:], "\n")
	}

	return result
}

// extractDiffPath extracts the file path from a "diff --git a/X b/Y" line.
// Returns Y without the "b/" prefix.
func extractDiffPath(line string) string {
	// Format: "diff --git a/path/to/file b/path/to/file"
	idx := strings.LastIndex(line, " b/")
	if idx < 0 {
		return ""
	}
	return line[idx+3:]
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
	Patch        string
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
			SupportedFeatures:    r.Capabilities.SupportedFeatures,
			AllowedExtraFlags:    r.Capabilities.AllowedExtraFlags,
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
	SupportedFeatures    []string
	AllowedExtraFlags    []string
}
