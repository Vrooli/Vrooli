// Responsibility: retain entities declarations within their original package.
package protoconv

import (
	"agent-manager/internal/domain"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// RunToProto converts a domain Run to proto Run.
func RunToProto(r *domain.Run) *pb.Run {
	if r == nil {
		return nil
	}

	run := &pb.Run{
		Id:                    UUIDToString(r.ID),
		TaskId:                UUIDToString(r.TaskID),
		Tag:                   validUTF8(r.Tag),
		Label:                 validUTF8(r.Label),
		LabelSource:           validUTF8(string(r.LabelSource)),
		SessionId:             validUTF8(r.SessionID),
		RunMode:               RunModeToProto(r.RunMode),
		ExecutionMode:         ExecutionModeToProto(r.ExecutionMode),
		HarnessKind:           validUTF8(r.HarnessKind),
		HarnessSessionId:      validUTF8(r.HarnessSessionID),
		GoalDelivery:          validUTF8(r.GoalDelivery),
		ObservedGoalStatus:    validUTF8(r.ObservedGoalStatus),
		TerminalClass:         validUTF8(string(r.TerminalClass)),
		StopReason:            validUTF8(string(r.StopReason)),
		LastHandoff:           validUTF8(r.LastHandoff),
		WebConsoleSessionId:   validUTF8(r.WebConsoleSessionID),
		WebConsoleSessionUrl:  validUTF8(r.WebConsoleSessionURL),
		Status:                RunStatusToProto(r.Status),
		Phase:                 RunPhaseToProto(r.Phase),
		ProgressPercent:       int32(r.ProgressPercent),
		IdempotencyKey:        validUTF8(r.IdempotencyKey),
		ErrorMsg:              validUTF8(r.ErrorMsg),
		ApprovalState:         ApprovalStateToProto(r.ApprovalState),
		ApprovedBy:            validUTF8(r.ApprovedBy),
		DiffPath:              validUTF8(r.DiffPath),
		LogPath:               validUTF8(r.LogPath),
		ChangedFiles:          int32(r.ChangedFiles),
		TotalSizeBytes:        r.TotalSizeBytes,
		CommitHash:            validUTF8(r.CommitHash),
		PromptPreview:         validUTF8(r.PromptPreview),
		RequestedModel:        validUTF8(r.RequestedModel),
		ActualModel:           validUTF8(r.ActualModel),
		FinalizationStatus:    RunFinalizationStatusToProto(r.FinalizationStatus),
		FinalizationError:     validUTF8(r.FinalizationError),
		CreatedAt:             TimestampToProto(r.CreatedAt),
		UpdatedAt:             TimestampToProto(r.UpdatedAt),
		ImportSourceHarness:   validUTF8(r.ImportSourceHarness),
		ImportSourceSessionId: validUTF8(r.ImportSourceSessionID),
		GoalId:                validUTF8(r.GoalID),
		WorkReferences:        r.WorkReferences,
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
	if r.ObservedGoalAt != nil {
		run.ObservedGoalAt = TimestampToProto(*r.ObservedGoalAt)
	}
	if r.ProviderActivityAt != nil {
		run.ProviderActivityAt = TimestampToProto(*r.ProviderActivityAt)
	}
	if r.ApprovedAt != nil {
		run.ApprovedAt = TimestampToProto(*r.ApprovedAt)
	}
	if r.FinalizedAt != nil {
		run.FinalizedAt = TimestampToProto(*r.FinalizedAt)
	}
	if r.ImportedAt != nil {
		t := TimestampToProto(*r.ImportedAt)
		run.ImportedAt = t
	}

	if r.AwaitHandle != nil {
		run.AwaitHandle = AwaitHandleToProto(r.AwaitHandle)
	}

	if r.Summary != nil {
		run.Summary = &pb.RunSummary{
			Description:   validUTF8(r.Summary.Description),
			FilesModified: validUTF8Slice(r.Summary.FilesModified),
			FilesCreated:  validUTF8Slice(r.Summary.FilesCreated),
			FilesDeleted:  validUTF8Slice(r.Summary.FilesDeleted),
			TokensUsed:    int32(r.Summary.TokensUsed),
			TurnsUsed:     int32(r.Summary.TurnsUsed),
			CostEstimate:  r.Summary.CostEstimate,
			ContextTokens: int32(r.Summary.ContextTokens),
		}
	}
	if r.Result != nil {
		run.Result = RunResultToProto(r.Result)
	}
	run.Subject = validUTF8Slice(r.Subject)

	if r.ResolvedConfig != nil {
		run.ResolvedConfig = RunConfigToProto(r.ResolvedConfig)
	}
	if r.Actions != nil {
		run.Actions = &pb.RunActions{
			CanInvestigate:             r.Actions.CanInvestigate,
			CanApplyInvestigation:      r.Actions.CanApplyInvestigation,
			CanDelete:                  r.Actions.CanDelete,
			CanStop:                    r.Actions.CanStop,
			CanRetry:                   r.Actions.CanRetry,
			CanContinue:                r.Actions.CanContinue,
			CanContinueReason:          validUTF8(r.Actions.CanContinueReason),
			CanApprove:                 r.Actions.CanApprove,
			CanReject:                  r.Actions.CanReject,
			CanReview:                  r.Actions.CanReview,
			CanResumeFromFailure:       r.Actions.CanResumeFromFailure,
			CanResumeFromFailureReason: validUTF8(r.Actions.CanResumeFromFailureReason),
			FinalizationWarning:        validUTF8(r.Actions.FinalizationWarning),
			CanRetryFinalization:       r.Actions.CanRetryFinalization,
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
		Producer:     validUTF8(h.Producer),
		Key:          validUTF8(h.Key),
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
		ID:                    UUIDFromString(r.Id),
		TaskID:                UUIDFromString(r.TaskId),
		Tag:                   r.Tag,
		SessionID:             r.SessionId,
		RunMode:               RunModeFromProto(r.RunMode),
		ExecutionMode:         ExecutionModeFromProto(r.ExecutionMode),
		HarnessKind:           r.HarnessKind,
		HarnessSessionID:      r.HarnessSessionId,
		GoalDelivery:          r.GoalDelivery,
		ObservedGoalStatus:    r.ObservedGoalStatus,
		TerminalClass:         domain.RunTerminalClass(r.TerminalClass),
		StopReason:            domain.RunStopReason(r.StopReason),
		LastHandoff:           r.LastHandoff,
		WebConsoleSessionID:   r.WebConsoleSessionId,
		WebConsoleSessionURL:  r.WebConsoleSessionUrl,
		Status:                RunStatusFromProto(r.Status),
		Phase:                 RunPhaseFromProto(r.Phase),
		ProgressPercent:       int(r.ProgressPercent),
		IdempotencyKey:        r.IdempotencyKey,
		ErrorMsg:              r.ErrorMsg,
		ApprovalState:         ApprovalStateFromProto(r.ApprovalState),
		ApprovedBy:            r.ApprovedBy,
		FinalizationStatus:    RunFinalizationStatusFromProto(r.FinalizationStatus),
		FinalizationError:     r.FinalizationError,
		DiffPath:              r.DiffPath,
		LogPath:               r.LogPath,
		ChangedFiles:          int(r.ChangedFiles),
		TotalSizeBytes:        r.TotalSizeBytes,
		CommitHash:            r.CommitHash,
		CreatedAt:             TimestampFromProto(r.CreatedAt),
		UpdatedAt:             TimestampFromProto(r.UpdatedAt),
		ImportSourceHarness:   r.ImportSourceHarness,
		ImportSourceSessionID: r.ImportSourceSessionId,
		GoalID:                r.GoalId,
		WorkReferences:        r.WorkReferences,
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
	if r.ObservedGoalAt != nil {
		t := TimestampFromProto(r.ObservedGoalAt)
		run.ObservedGoalAt = &t
	}
	if r.ProviderActivityAt != nil {
		t := TimestampFromProto(r.ProviderActivityAt)
		run.ProviderActivityAt = &t
	}
	if r.ApprovedAt != nil {
		t := TimestampFromProto(r.ApprovedAt)
		run.ApprovedAt = &t
	}
	if r.FinalizedAt != nil {
		t := TimestampFromProto(r.FinalizedAt)
		run.FinalizedAt = &t
	}
	if r.ImportedAt != nil {
		t := TimestampFromProto(r.ImportedAt)
		run.ImportedAt = &t
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
	run.Subject = append([]string(nil), r.Subject...)

	if r.ResolvedConfig != nil {
		run.ResolvedConfig = RunConfigFromProto(r.ResolvedConfig)
	}
	if r.Actions != nil {
		run.Actions = &domain.RunActions{
			CanInvestigate:             r.Actions.CanInvestigate,
			CanApplyInvestigation:      r.Actions.CanApplyInvestigation,
			CanDelete:                  r.Actions.CanDelete,
			CanStop:                    r.Actions.CanStop,
			CanRetry:                   r.Actions.CanRetry,
			CanContinue:                r.Actions.CanContinue,
			CanContinueReason:          r.Actions.CanContinueReason,
			CanApprove:                 r.Actions.CanApprove,
			CanReject:                  r.Actions.CanReject,
			CanReview:                  r.Actions.CanReview,
			CanResumeFromFailure:       r.Actions.CanResumeFromFailure,
			CanResumeFromFailureReason: r.Actions.CanResumeFromFailureReason,
			FinalizationWarning:        r.Actions.FinalizationWarning,
			CanRetryFinalization:       r.Actions.CanRetryFinalization,
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
		RunnerType:            RunnerTypeToProto(c.RunnerType),
		Model:                 c.Model,
		RoleRef:               c.RoleRef,
		MaxTurns:              int32(c.MaxTurns),
		Timeout:               DurationToProto(c.Timeout),
		Effort:                string(c.Effort),
		AllowedTools:          c.AllowedTools,
		DeniedTools:           c.DeniedTools,
		ToolRestrictionPolicy: string(c.ToolRestrictionPolicy.Effective()),
		SkipPermissionPrompt:  c.SkipPermissionPrompt,
		Features:              FeatureFlagsToProto(c.Features),
		ExtraFlags:            RunnerExtraFlagsToProto(c.ExtraFlags),
		NetworkAccess:         NetworkAccessToProto(c.NetworkAccess),
		PolicySnapshot:        ExecutionPolicySnapshotToProto(c.PolicySnapshot),
		ResultSpec:            ResultSpecToProto(c.ResultSpec),
		SandboxConfig:         SandboxConfigToProto(c.SandboxConfig),
		AllowedPaths:          c.AllowedPaths,
		DeniedPaths:           c.DeniedPaths,
		ManifestIndexSnapshot: c.ManifestIndexSnapshot,
		TranscriptCodec:       c.TranscriptCodec,
		TranscriptCodecScore:  c.TranscriptCodecScore,
		Until:                 c.Until,
		Admission:             RunAdmissionToProto(c.Admission),
	}
}

// RunConfigFromProto converts a proto RunConfig to domain RunConfig.
func RunConfigFromProto(c *pb.RunConfig) *domain.RunConfig {
	if c == nil {
		return nil
	}
	return &domain.RunConfig{
		RunnerType:            RunnerTypeFromProto(c.RunnerType),
		Model:                 c.Model,
		RoleRef:               c.RoleRef,
		MaxTurns:              int(c.MaxTurns),
		Timeout:               DurationFromProto(c.Timeout),
		Effort:                domain.Effort(c.Effort),
		AllowedTools:          c.AllowedTools,
		DeniedTools:           c.DeniedTools,
		ToolRestrictionPolicy: domain.ToolRestrictionPolicy(c.ToolRestrictionPolicy),
		SkipPermissionPrompt:  c.SkipPermissionPrompt,
		Features:              FeatureFlagsFromProto(c.Features),
		ExtraFlags:            RunnerExtraFlagsFromProto(c.ExtraFlags),
		NetworkAccess:         NetworkAccessFromProto(c.NetworkAccess),
		PolicySnapshot:        ExecutionPolicySnapshotFromProto(c.PolicySnapshot),
		ResultSpec:            ResultSpecFromProto(c.ResultSpec),
		SandboxConfig:         SandboxConfigFromProto(c.SandboxConfig),
		AllowedPaths:          c.AllowedPaths,
		DeniedPaths:           c.DeniedPaths,
		ManifestIndexSnapshot: c.ManifestIndexSnapshot,
		TranscriptCodec:       c.TranscriptCodec,
		TranscriptCodecScore:  c.TranscriptCodecScore,
		Until:                 c.Until,
		Admission:             RunAdmissionFromProto(c.Admission),
	}
}

// RunAdmissionToProto converts the creation-time admission record to its proto mirror.
func RunAdmissionToProto(a *domain.RunAdmission) *pb.RunAdmission {
	if a == nil {
		return nil
	}
	return &pb.RunAdmission{
		RequestedRunner:        a.RequestedRunner,
		RequestedModel:         a.RequestedModel,
		RequestedRoleRef:       a.RequestedRoleRef,
		RequestedEffort:        a.RequestedEffort,
		RequestedTimeout:       DurationToProto(a.RequestedTimeout),
		RequestedMaxTurns:      int32(a.RequestedMaxTurns),
		RequestedGoalMode:      a.RequestedGoalMode,
		EffectiveRunner:        a.EffectiveRunner,
		EffectiveModel:         a.EffectiveModel,
		EffectiveEffort:        a.EffectiveEffort,
		EffectiveTimeout:       DurationToProto(a.EffectiveTimeout),
		EffectiveMaxTurns:      int32(a.EffectiveMaxTurns),
		EffectiveUntil:         a.EffectiveUntil,
		CatalogDigest:          a.CatalogDigest,
		PolicyDigest:           a.PolicyDigest,
		PolicyPath:             a.PolicyPath,
		SelectionReason:        a.SelectionReason,
		PassedControlArgs:      append([]string(nil), a.PassedControlArgs...),
		TranslationDiagnostics: append([]string(nil), a.TranslationDiagnostics...),
		RuntimeVersion:         a.RuntimeVersion,
		ProviderAcknowledgment: append([]string(nil), a.ProviderAcknowledgment...),
		Receipt:                QualificationReceiptToProto(a.Receipt),
	}
}

// RunAdmissionFromProto converts a proto admission record to the domain type.
func RunAdmissionFromProto(a *pb.RunAdmission) *domain.RunAdmission {
	if a == nil {
		return nil
	}
	return &domain.RunAdmission{
		RequestedRunner:        a.RequestedRunner,
		RequestedModel:         a.RequestedModel,
		RequestedRoleRef:       a.RequestedRoleRef,
		RequestedEffort:        a.RequestedEffort,
		RequestedTimeout:       DurationFromProto(a.RequestedTimeout),
		RequestedMaxTurns:      int(a.RequestedMaxTurns),
		RequestedGoalMode:      a.RequestedGoalMode,
		EffectiveRunner:        a.EffectiveRunner,
		EffectiveModel:         a.EffectiveModel,
		EffectiveEffort:        a.EffectiveEffort,
		EffectiveTimeout:       DurationFromProto(a.EffectiveTimeout),
		EffectiveMaxTurns:      int(a.EffectiveMaxTurns),
		EffectiveUntil:         a.EffectiveUntil,
		CatalogDigest:          a.CatalogDigest,
		PolicyDigest:           a.PolicyDigest,
		PolicyPath:             a.PolicyPath,
		SelectionReason:        a.SelectionReason,
		PassedControlArgs:      append([]string(nil), a.PassedControlArgs...),
		TranslationDiagnostics: append([]string(nil), a.TranslationDiagnostics...),
		RuntimeVersion:         a.RuntimeVersion,
		ProviderAcknowledgment: append([]string(nil), a.ProviderAcknowledgment...),
		Receipt:                QualificationReceiptFromProto(a.Receipt),
	}
}

// QualificationReceiptToProto converts the route-keyed qualification receipt to
// its proto mirror. A nil receipt stays nil so a run with no live
// qualification evidence is never reported as qualified.
func QualificationReceiptToProto(r *domain.QualificationReceipt) *pb.QualificationReceipt {
	if r == nil {
		return nil
	}
	return &pb.QualificationReceipt{
		Route:                  r.Route,
		RequestedRunner:        r.RequestedRunner,
		RequestedModel:         r.RequestedModel,
		RequestedRoleRef:       r.RequestedRoleRef,
		RequestedEffort:        r.RequestedEffort,
		EffectiveRunner:        r.EffectiveRunner,
		EffectiveModel:         r.EffectiveModel,
		EffectiveEffort:        r.EffectiveEffort,
		PassedControlArgs:      append([]string(nil), r.PassedControlArgs...),
		TranslationDiagnostics: append([]string(nil), r.TranslationDiagnostics...),
		ProviderAcknowledgment: append([]string(nil), r.ProviderAcknowledgment...),
		CatalogDigest:          r.CatalogDigest,
		PolicyDigest:           r.PolicyDigest,
		PolicyPath:             r.PolicyPath,
		RuntimeVersion:         r.RuntimeVersion,
		RunId:                  r.RunID,
		OperationId:            r.OperationID,
		AcceptedOutput:         r.AcceptedOutput,
		Usage:                  QualificationUsageToProto(r.Usage),
		Limitations:            append([]string(nil), r.Limitations...),
		CapturedAt:             TimestampToProto(r.CapturedAt),
	}
}

// QualificationReceiptFromProto converts a proto qualification receipt to the
// domain type, preserving unobserved fields as empty rather than backfilling
// them from the requested or effective layers.
func QualificationReceiptFromProto(r *pb.QualificationReceipt) *domain.QualificationReceipt {
	if r == nil {
		return nil
	}
	return &domain.QualificationReceipt{
		Route:                  r.Route,
		RequestedRunner:        r.RequestedRunner,
		RequestedModel:         r.RequestedModel,
		RequestedRoleRef:       r.RequestedRoleRef,
		RequestedEffort:        r.RequestedEffort,
		EffectiveRunner:        r.EffectiveRunner,
		EffectiveModel:         r.EffectiveModel,
		EffectiveEffort:        r.EffectiveEffort,
		PassedControlArgs:      append([]string(nil), r.PassedControlArgs...),
		TranslationDiagnostics: append([]string(nil), r.TranslationDiagnostics...),
		ProviderAcknowledgment: append([]string(nil), r.ProviderAcknowledgment...),
		CatalogDigest:          r.CatalogDigest,
		PolicyDigest:           r.PolicyDigest,
		PolicyPath:             r.PolicyPath,
		RuntimeVersion:         r.RuntimeVersion,
		RunID:                  r.RunId,
		OperationID:            r.OperationId,
		AcceptedOutput:         r.AcceptedOutput,
		Usage:                  QualificationUsageFromProto(r.Usage),
		Limitations:            append([]string(nil), r.Limitations...),
		CapturedAt:             TimestampFromProto(r.CapturedAt),
	}
}

// QualificationUsageToProto converts usage state to its proto mirror. A zero
// usage stays a zero message so a reader can still tell measured from reserved.
func QualificationUsageToProto(u domain.QualificationUsage) *pb.QualificationUsage {
	return &pb.QualificationUsage{
		State:           string(u.State),
		InputTokens:     u.InputTokens,
		OutputTokens:    u.OutputTokens,
		CostUsd:         u.CostUSD,
		ReservedUnknown: u.ReservedUnknown,
	}
}

// QualificationUsageFromProto converts proto usage state to the domain type.
func QualificationUsageFromProto(u *pb.QualificationUsage) domain.QualificationUsage {
	if u == nil {
		return domain.QualificationUsage{}
	}
	return domain.QualificationUsage{
		State:           domain.QualificationUsageState(u.State),
		InputTokens:     u.InputTokens,
		OutputTokens:    u.OutputTokens,
		CostUSD:         u.CostUsd,
		ReservedUnknown: u.ReservedUnknown,
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
		SchemaRepairAttempts: intToInt32Ptr(spec.SchemaRepairAttempts),
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
		SchemaRepairAttempts: int32ToIntPtr(spec.SchemaRepairAttempts),
	}
}

func intToInt32Ptr(value *int) *int32 {
	if value == nil {
		return nil
	}
	converted := int32(*value)
	return &converted
}

func int32ToIntPtr(value *int32) *int {
	if value == nil {
		return nil
	}
	converted := int(*value)
	return &converted
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
		SelectionReason:   snapshot.SelectionReason,
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
		SelectionReason:   snapshot.SelectionReason,
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
