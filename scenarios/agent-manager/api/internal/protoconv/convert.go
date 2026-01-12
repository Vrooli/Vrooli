// Package protoconv provides conversions between internal domain types and proto types.
//
// This package bridges the internal domain model with the generated protobuf types
// for API serialization. It enables:
//   - Type-safe API contracts via protobuf
//   - Internal domain logic with Go-native types
//   - Consistent JSON encoding using protojson
package protoconv

import (
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"agent-manager/internal/domain"

	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// JSONMarshalOptions are the protojson options for API serialization.
// UseProtoNames ensures snake_case JSON field names.
var JSONMarshalOptions = protojson.MarshalOptions{
	UseProtoNames:   true,
	EmitUnpopulated: false,
}

// JSONUnmarshalOptions are the protojson options for API deserialization.
// DiscardUnknown is false to fail fast on schema mismatches.
var JSONUnmarshalOptions = protojson.UnmarshalOptions{
	DiscardUnknown: false,
}

// MarshalJSON marshals a proto message to JSON bytes.
func MarshalJSON(m proto.Message) ([]byte, error) {
	return JSONMarshalOptions.Marshal(m)
}

// UnmarshalJSON unmarshals JSON bytes into a proto message.
func UnmarshalJSON(data []byte, m proto.Message) error {
	return JSONUnmarshalOptions.Unmarshal(data, m)
}

// =============================================================================
// RUNNER TYPE
// =============================================================================

// RunnerTypeToProto converts domain RunnerType to proto RunnerType.
func RunnerTypeToProto(r domain.RunnerType) pb.RunnerType {
	switch r {
	case domain.RunnerTypeClaudeCode:
		return pb.RunnerType_RUNNER_TYPE_CLAUDE_CODE
	case domain.RunnerTypeCodex:
		return pb.RunnerType_RUNNER_TYPE_CODEX
	case domain.RunnerTypeOpenCode:
		return pb.RunnerType_RUNNER_TYPE_OPENCODE
	default:
		return pb.RunnerType_RUNNER_TYPE_UNSPECIFIED
	}
}

// RunnerTypeFromProto converts proto RunnerType to domain RunnerType.
func RunnerTypeFromProto(r pb.RunnerType) domain.RunnerType {
	switch r {
	case pb.RunnerType_RUNNER_TYPE_CLAUDE_CODE:
		return domain.RunnerTypeClaudeCode
	case pb.RunnerType_RUNNER_TYPE_CODEX:
		return domain.RunnerTypeCodex
	case pb.RunnerType_RUNNER_TYPE_OPENCODE:
		return domain.RunnerTypeOpenCode
	default:
		return domain.RunnerType("")
	}
}

// =============================================================================
// MODEL PRESET
// =============================================================================

// ModelPresetToProto converts domain ModelPreset to proto ModelPreset.
func ModelPresetToProto(preset domain.ModelPreset) pb.ModelPreset {
	switch preset {
	case domain.ModelPresetFast:
		return pb.ModelPreset_MODEL_PRESET_FAST
	case domain.ModelPresetCheap:
		return pb.ModelPreset_MODEL_PRESET_CHEAP
	case domain.ModelPresetSmart:
		return pb.ModelPreset_MODEL_PRESET_SMART
	default:
		return pb.ModelPreset_MODEL_PRESET_UNSPECIFIED
	}
}

// ModelPresetFromProto converts proto ModelPreset to domain ModelPreset.
func ModelPresetFromProto(preset pb.ModelPreset) domain.ModelPreset {
	switch preset {
	case pb.ModelPreset_MODEL_PRESET_FAST:
		return domain.ModelPresetFast
	case pb.ModelPreset_MODEL_PRESET_CHEAP:
		return domain.ModelPresetCheap
	case pb.ModelPreset_MODEL_PRESET_SMART:
		return domain.ModelPresetSmart
	default:
		return domain.ModelPresetUnspecified
	}
}

// =============================================================================
// SANDBOX CONFIG
// =============================================================================

// SandboxLifecycleEventToProto converts domain SandboxLifecycleEvent to proto.
func SandboxLifecycleEventToProto(event domain.SandboxLifecycleEvent) pb.SandboxLifecycleEvent {
	switch event {
	case domain.SandboxLifecycleRunCompleted:
		return pb.SandboxLifecycleEvent_SANDBOX_LIFECYCLE_EVENT_RUN_COMPLETED
	case domain.SandboxLifecycleRunFailed:
		return pb.SandboxLifecycleEvent_SANDBOX_LIFECYCLE_EVENT_RUN_FAILED
	case domain.SandboxLifecycleRunCancelled:
		return pb.SandboxLifecycleEvent_SANDBOX_LIFECYCLE_EVENT_RUN_CANCELLED
	case domain.SandboxLifecycleApproved:
		return pb.SandboxLifecycleEvent_SANDBOX_LIFECYCLE_EVENT_APPROVED
	case domain.SandboxLifecycleRejected:
		return pb.SandboxLifecycleEvent_SANDBOX_LIFECYCLE_EVENT_REJECTED
	case domain.SandboxLifecycleTerminal:
		return pb.SandboxLifecycleEvent_SANDBOX_LIFECYCLE_EVENT_TERMINAL
	default:
		return pb.SandboxLifecycleEvent_SANDBOX_LIFECYCLE_EVENT_UNSPECIFIED
	}
}

// SandboxLifecycleEventFromProto converts proto SandboxLifecycleEvent to domain.
func SandboxLifecycleEventFromProto(event pb.SandboxLifecycleEvent) domain.SandboxLifecycleEvent {
	switch event {
	case pb.SandboxLifecycleEvent_SANDBOX_LIFECYCLE_EVENT_RUN_COMPLETED:
		return domain.SandboxLifecycleRunCompleted
	case pb.SandboxLifecycleEvent_SANDBOX_LIFECYCLE_EVENT_RUN_FAILED:
		return domain.SandboxLifecycleRunFailed
	case pb.SandboxLifecycleEvent_SANDBOX_LIFECYCLE_EVENT_RUN_CANCELLED:
		return domain.SandboxLifecycleRunCancelled
	case pb.SandboxLifecycleEvent_SANDBOX_LIFECYCLE_EVENT_APPROVED:
		return domain.SandboxLifecycleApproved
	case pb.SandboxLifecycleEvent_SANDBOX_LIFECYCLE_EVENT_REJECTED:
		return domain.SandboxLifecycleRejected
	case pb.SandboxLifecycleEvent_SANDBOX_LIFECYCLE_EVENT_TERMINAL:
		return domain.SandboxLifecycleTerminal
	default:
		return ""
	}
}

// SandboxAcceptanceModeToProto converts domain acceptance mode to proto.
func SandboxAcceptanceModeToProto(mode string) pb.SandboxAcceptanceMode {
	switch mode {
	case "allowlist":
		return pb.SandboxAcceptanceMode_SANDBOX_ACCEPTANCE_MODE_ALLOWLIST
	default:
		return pb.SandboxAcceptanceMode_SANDBOX_ACCEPTANCE_MODE_UNSPECIFIED
	}
}

// SandboxAcceptanceModeFromProto converts proto acceptance mode to domain.
func SandboxAcceptanceModeFromProto(mode pb.SandboxAcceptanceMode) string {
	switch mode {
	case pb.SandboxAcceptanceMode_SANDBOX_ACCEPTANCE_MODE_ALLOWLIST:
		return "allowlist"
	default:
		return ""
	}
}

// SandboxFileCriteriaToProto converts domain SandboxFileCriteria to proto.
func SandboxFileCriteriaToProto(criteria domain.SandboxFileCriteria) *pb.SandboxFileCriteria {
	return &pb.SandboxFileCriteria{
		PathGlobs:  criteria.PathGlobs,
		Extensions: criteria.Extensions,
	}
}

// SandboxFileCriteriaFromProto converts proto SandboxFileCriteria to domain.
func SandboxFileCriteriaFromProto(criteria *pb.SandboxFileCriteria) domain.SandboxFileCriteria {
	if criteria == nil {
		return domain.SandboxFileCriteria{}
	}
	return domain.SandboxFileCriteria{
		PathGlobs:  criteria.PathGlobs,
		Extensions: criteria.Extensions,
	}
}

// SandboxAcceptanceConfigToProto converts domain SandboxAcceptanceConfig to proto.
func SandboxAcceptanceConfigToProto(cfg domain.SandboxAcceptanceConfig) *pb.SandboxAcceptanceConfig {
	return &pb.SandboxAcceptanceConfig{
		Mode:         SandboxAcceptanceModeToProto(cfg.Mode),
		Allow:        SandboxFileCriteriaToProto(cfg.Allow),
		Deny:         SandboxFileCriteriaToProto(cfg.Deny),
		IgnoreBinary: cfg.IgnoreBinary,
		AutoApprove:  cfg.AutoApprove,
		AutoReject:   cfg.AutoReject,
	}
}

// SandboxAcceptanceConfigFromProto converts proto SandboxAcceptanceConfig to domain.
func SandboxAcceptanceConfigFromProto(cfg *pb.SandboxAcceptanceConfig) domain.SandboxAcceptanceConfig {
	if cfg == nil {
		return domain.SandboxAcceptanceConfig{}
	}
	return domain.SandboxAcceptanceConfig{
		Mode:         SandboxAcceptanceModeFromProto(cfg.Mode),
		Allow:        SandboxFileCriteriaFromProto(cfg.Allow),
		Deny:         SandboxFileCriteriaFromProto(cfg.Deny),
		IgnoreBinary: cfg.IgnoreBinary,
		AutoApprove:  cfg.AutoApprove,
		AutoReject:   cfg.AutoReject,
	}
}

// SandboxLifecycleConfigToProto converts domain SandboxLifecycleConfig to proto.
func SandboxLifecycleConfigToProto(cfg domain.SandboxLifecycleConfig) *pb.SandboxLifecycleConfig {
	stopOn := make([]pb.SandboxLifecycleEvent, 0, len(cfg.StopOn))
	for _, event := range cfg.StopOn {
		stopOn = append(stopOn, SandboxLifecycleEventToProto(event))
	}
	deleteOn := make([]pb.SandboxLifecycleEvent, 0, len(cfg.DeleteOn))
	for _, event := range cfg.DeleteOn {
		deleteOn = append(deleteOn, SandboxLifecycleEventToProto(event))
	}
	return &pb.SandboxLifecycleConfig{
		StopOn:      stopOn,
		DeleteOn:    deleteOn,
		Ttl:         DurationToProto(cfg.TTL),
		IdleTimeout: DurationToProto(cfg.IdleTimeout),
	}
}

// SandboxLifecycleConfigFromProto converts proto SandboxLifecycleConfig to domain.
func SandboxLifecycleConfigFromProto(cfg *pb.SandboxLifecycleConfig) domain.SandboxLifecycleConfig {
	if cfg == nil {
		return domain.SandboxLifecycleConfig{}
	}
	stopOn := make([]domain.SandboxLifecycleEvent, 0, len(cfg.StopOn))
	for _, event := range cfg.StopOn {
		if event == pb.SandboxLifecycleEvent_SANDBOX_LIFECYCLE_EVENT_UNSPECIFIED {
			continue
		}
		stopOn = append(stopOn, SandboxLifecycleEventFromProto(event))
	}
	deleteOn := make([]domain.SandboxLifecycleEvent, 0, len(cfg.DeleteOn))
	for _, event := range cfg.DeleteOn {
		if event == pb.SandboxLifecycleEvent_SANDBOX_LIFECYCLE_EVENT_UNSPECIFIED {
			continue
		}
		deleteOn = append(deleteOn, SandboxLifecycleEventFromProto(event))
	}
	return domain.SandboxLifecycleConfig{
		StopOn:      stopOn,
		DeleteOn:    deleteOn,
		TTL:         DurationFromProto(cfg.Ttl),
		IdleTimeout: DurationFromProto(cfg.IdleTimeout),
	}
}

// SandboxConfigToProto converts domain SandboxConfig to proto.
func SandboxConfigToProto(cfg *domain.SandboxConfig) *pb.SandboxConfig {
	if cfg == nil {
		return nil
	}
	return &pb.SandboxConfig{
		Lifecycle:  SandboxLifecycleConfigToProto(cfg.Lifecycle),
		Acceptance: SandboxAcceptanceConfigToProto(cfg.Acceptance),
	}
}

// SandboxConfigFromProto converts proto SandboxConfig to domain.
func SandboxConfigFromProto(cfg *pb.SandboxConfig) *domain.SandboxConfig {
	if cfg == nil {
		return nil
	}
	return &domain.SandboxConfig{
		Lifecycle:  SandboxLifecycleConfigFromProto(cfg.Lifecycle),
		Acceptance: SandboxAcceptanceConfigFromProto(cfg.Acceptance),
	}
}

// =============================================================================
// TASK STATUS
// =============================================================================

// TaskStatusToProto converts domain TaskStatus to proto TaskStatus.
func TaskStatusToProto(s domain.TaskStatus) pb.TaskStatus {
	switch s {
	case domain.TaskStatusQueued:
		return pb.TaskStatus_TASK_STATUS_QUEUED
	case domain.TaskStatusRunning:
		return pb.TaskStatus_TASK_STATUS_RUNNING
	case domain.TaskStatusNeedsReview:
		return pb.TaskStatus_TASK_STATUS_NEEDS_REVIEW
	case domain.TaskStatusApproved:
		return pb.TaskStatus_TASK_STATUS_APPROVED
	case domain.TaskStatusRejected:
		return pb.TaskStatus_TASK_STATUS_REJECTED
	case domain.TaskStatusFailed:
		return pb.TaskStatus_TASK_STATUS_FAILED
	case domain.TaskStatusCancelled:
		return pb.TaskStatus_TASK_STATUS_CANCELLED
	default:
		return pb.TaskStatus_TASK_STATUS_UNSPECIFIED
	}
}

// TaskStatusFromProto converts proto TaskStatus to domain TaskStatus.
func TaskStatusFromProto(s pb.TaskStatus) domain.TaskStatus {
	switch s {
	case pb.TaskStatus_TASK_STATUS_QUEUED:
		return domain.TaskStatusQueued
	case pb.TaskStatus_TASK_STATUS_RUNNING:
		return domain.TaskStatusRunning
	case pb.TaskStatus_TASK_STATUS_NEEDS_REVIEW:
		return domain.TaskStatusNeedsReview
	case pb.TaskStatus_TASK_STATUS_APPROVED:
		return domain.TaskStatusApproved
	case pb.TaskStatus_TASK_STATUS_REJECTED:
		return domain.TaskStatusRejected
	case pb.TaskStatus_TASK_STATUS_FAILED:
		return domain.TaskStatusFailed
	case pb.TaskStatus_TASK_STATUS_CANCELLED:
		return domain.TaskStatusCancelled
	default:
		return domain.TaskStatus("")
	}
}

// =============================================================================
// RUN STATUS
// =============================================================================

// RunStatusToProto converts domain RunStatus to proto RunStatus.
func RunStatusToProto(s domain.RunStatus) pb.RunStatus {
	switch s {
	case domain.RunStatusPending:
		return pb.RunStatus_RUN_STATUS_PENDING
	case domain.RunStatusStarting:
		return pb.RunStatus_RUN_STATUS_STARTING
	case domain.RunStatusRunning:
		return pb.RunStatus_RUN_STATUS_RUNNING
	case domain.RunStatusNeedsReview:
		return pb.RunStatus_RUN_STATUS_NEEDS_REVIEW
	case domain.RunStatusComplete:
		return pb.RunStatus_RUN_STATUS_COMPLETE
	case domain.RunStatusFailed:
		return pb.RunStatus_RUN_STATUS_FAILED
	case domain.RunStatusCancelled:
		return pb.RunStatus_RUN_STATUS_CANCELLED
	default:
		return pb.RunStatus_RUN_STATUS_UNSPECIFIED
	}
}

// RunStatusFromProto converts proto RunStatus to domain RunStatus.
func RunStatusFromProto(s pb.RunStatus) domain.RunStatus {
	switch s {
	case pb.RunStatus_RUN_STATUS_PENDING:
		return domain.RunStatusPending
	case pb.RunStatus_RUN_STATUS_STARTING:
		return domain.RunStatusStarting
	case pb.RunStatus_RUN_STATUS_RUNNING:
		return domain.RunStatusRunning
	case pb.RunStatus_RUN_STATUS_NEEDS_REVIEW:
		return domain.RunStatusNeedsReview
	case pb.RunStatus_RUN_STATUS_COMPLETE:
		return domain.RunStatusComplete
	case pb.RunStatus_RUN_STATUS_FAILED:
		return domain.RunStatusFailed
	case pb.RunStatus_RUN_STATUS_CANCELLED:
		return domain.RunStatusCancelled
	default:
		return domain.RunStatus("")
	}
}

// =============================================================================
// RUN PHASE
// =============================================================================

// RunPhaseToProto converts domain RunPhase to proto RunPhase.
func RunPhaseToProto(p domain.RunPhase) pb.RunPhase {
	switch p {
	case domain.RunPhaseQueued:
		return pb.RunPhase_RUN_PHASE_QUEUED
	case domain.RunPhaseInitializing:
		return pb.RunPhase_RUN_PHASE_INITIALIZING
	case domain.RunPhaseSandboxCreating:
		return pb.RunPhase_RUN_PHASE_SANDBOX_CREATING
	case domain.RunPhaseRunnerAcquiring:
		return pb.RunPhase_RUN_PHASE_RUNNER_ACQUIRING
	case domain.RunPhaseExecuting:
		return pb.RunPhase_RUN_PHASE_EXECUTING
	case domain.RunPhaseCollectingResults:
		return pb.RunPhase_RUN_PHASE_COLLECTING_RESULTS
	case domain.RunPhaseAwaitingReview:
		return pb.RunPhase_RUN_PHASE_AWAITING_REVIEW
	case domain.RunPhaseApplying:
		return pb.RunPhase_RUN_PHASE_APPLYING
	case domain.RunPhaseCleaningUp:
		return pb.RunPhase_RUN_PHASE_CLEANING_UP
	case domain.RunPhaseCompleted:
		return pb.RunPhase_RUN_PHASE_COMPLETED
	default:
		return pb.RunPhase_RUN_PHASE_UNSPECIFIED
	}
}

// RunPhaseFromProto converts proto RunPhase to domain RunPhase.
func RunPhaseFromProto(p pb.RunPhase) domain.RunPhase {
	switch p {
	case pb.RunPhase_RUN_PHASE_QUEUED:
		return domain.RunPhaseQueued
	case pb.RunPhase_RUN_PHASE_INITIALIZING:
		return domain.RunPhaseInitializing
	case pb.RunPhase_RUN_PHASE_SANDBOX_CREATING:
		return domain.RunPhaseSandboxCreating
	case pb.RunPhase_RUN_PHASE_RUNNER_ACQUIRING:
		return domain.RunPhaseRunnerAcquiring
	case pb.RunPhase_RUN_PHASE_EXECUTING:
		return domain.RunPhaseExecuting
	case pb.RunPhase_RUN_PHASE_COLLECTING_RESULTS:
		return domain.RunPhaseCollectingResults
	case pb.RunPhase_RUN_PHASE_AWAITING_REVIEW:
		return domain.RunPhaseAwaitingReview
	case pb.RunPhase_RUN_PHASE_APPLYING:
		return domain.RunPhaseApplying
	case pb.RunPhase_RUN_PHASE_CLEANING_UP:
		return domain.RunPhaseCleaningUp
	case pb.RunPhase_RUN_PHASE_COMPLETED:
		return domain.RunPhaseCompleted
	default:
		return domain.RunPhaseQueued
	}
}

// =============================================================================
// RUN MODE
// =============================================================================

// RunModeToProto converts domain RunMode to proto RunMode.
func RunModeToProto(m domain.RunMode) pb.RunMode {
	switch m {
	case domain.RunModeSandboxed:
		return pb.RunMode_RUN_MODE_SANDBOXED
	case domain.RunModeInPlace:
		return pb.RunMode_RUN_MODE_IN_PLACE
	default:
		return pb.RunMode_RUN_MODE_SANDBOXED
	}
}

// RunModeFromProto converts proto RunMode to domain RunMode.
func RunModeFromProto(m pb.RunMode) domain.RunMode {
	switch m {
	case pb.RunMode_RUN_MODE_SANDBOXED:
		return domain.RunModeSandboxed
	case pb.RunMode_RUN_MODE_IN_PLACE:
		return domain.RunModeInPlace
	default:
		return domain.RunModeSandboxed
	}
}

// =============================================================================
// APPROVAL STATE
// =============================================================================

// ApprovalStateToProto converts domain ApprovalState to proto ApprovalState.
func ApprovalStateToProto(s domain.ApprovalState) pb.ApprovalState {
	switch s {
	case domain.ApprovalStateNone:
		return pb.ApprovalState_APPROVAL_STATE_NONE
	case domain.ApprovalStatePending:
		return pb.ApprovalState_APPROVAL_STATE_PENDING
	case domain.ApprovalStatePartiallyApproved:
		return pb.ApprovalState_APPROVAL_STATE_PARTIALLY_APPROVED
	case domain.ApprovalStateApproved:
		return pb.ApprovalState_APPROVAL_STATE_APPROVED
	case domain.ApprovalStateRejected:
		return pb.ApprovalState_APPROVAL_STATE_REJECTED
	default:
		return pb.ApprovalState_APPROVAL_STATE_UNSPECIFIED
	}
}

// ApprovalStateFromProto converts proto ApprovalState to domain ApprovalState.
func ApprovalStateFromProto(s pb.ApprovalState) domain.ApprovalState {
	switch s {
	case pb.ApprovalState_APPROVAL_STATE_NONE:
		return domain.ApprovalStateNone
	case pb.ApprovalState_APPROVAL_STATE_PENDING:
		return domain.ApprovalStatePending
	case pb.ApprovalState_APPROVAL_STATE_PARTIALLY_APPROVED:
		return domain.ApprovalStatePartiallyApproved
	case pb.ApprovalState_APPROVAL_STATE_APPROVED:
		return domain.ApprovalStateApproved
	case pb.ApprovalState_APPROVAL_STATE_REJECTED:
		return domain.ApprovalStateRejected
	default:
		return domain.ApprovalStateNone
	}
}

// =============================================================================
// RUN EVENT TYPE
// =============================================================================

// RunEventTypeToProto converts domain RunEventType to proto RunEventType.
func RunEventTypeToProto(t domain.RunEventType) pb.RunEventType {
	switch t {
	case domain.EventTypeLog:
		return pb.RunEventType_RUN_EVENT_TYPE_LOG
	case domain.EventTypeMessage:
		return pb.RunEventType_RUN_EVENT_TYPE_MESSAGE
	case domain.EventTypeMessageDeleted:
		return pb.RunEventType_RUN_EVENT_TYPE_MESSAGE_DELETED
	case domain.EventTypeToolCall:
		return pb.RunEventType_RUN_EVENT_TYPE_TOOL_CALL
	case domain.EventTypeToolResult:
		return pb.RunEventType_RUN_EVENT_TYPE_TOOL_RESULT
	case domain.EventTypeStatus:
		return pb.RunEventType_RUN_EVENT_TYPE_STATUS
	case domain.EventTypeMetric:
		return pb.RunEventType_RUN_EVENT_TYPE_METRIC
	case domain.EventTypeArtifact:
		return pb.RunEventType_RUN_EVENT_TYPE_ARTIFACT
	case domain.EventTypeError:
		return pb.RunEventType_RUN_EVENT_TYPE_ERROR
	default:
		return pb.RunEventType_RUN_EVENT_TYPE_UNSPECIFIED
	}
}

// RunEventTypeFromProto converts proto RunEventType to domain RunEventType.
func RunEventTypeFromProto(t pb.RunEventType) domain.RunEventType {
	switch t {
	case pb.RunEventType_RUN_EVENT_TYPE_LOG:
		return domain.EventTypeLog
	case pb.RunEventType_RUN_EVENT_TYPE_MESSAGE:
		return domain.EventTypeMessage
	case pb.RunEventType_RUN_EVENT_TYPE_MESSAGE_DELETED:
		return domain.EventTypeMessageDeleted
	case pb.RunEventType_RUN_EVENT_TYPE_TOOL_CALL:
		return domain.EventTypeToolCall
	case pb.RunEventType_RUN_EVENT_TYPE_TOOL_RESULT:
		return domain.EventTypeToolResult
	case pb.RunEventType_RUN_EVENT_TYPE_STATUS:
		return domain.EventTypeStatus
	case pb.RunEventType_RUN_EVENT_TYPE_METRIC:
		return domain.EventTypeMetric
	case pb.RunEventType_RUN_EVENT_TYPE_ARTIFACT:
		return domain.EventTypeArtifact
	case pb.RunEventType_RUN_EVENT_TYPE_ERROR:
		return domain.EventTypeError
	default:
		return domain.EventTypeLog
	}
}

// =============================================================================
// RECOVERY ACTION
// =============================================================================

// RecoveryActionToProto converts domain RecoveryAction to proto RecoveryAction.
func RecoveryActionToProto(a domain.RecoveryAction) pb.RecoveryAction {
	switch a {
	case domain.RecoveryNone:
		return pb.RecoveryAction_RECOVERY_ACTION_NONE
	case domain.RecoveryRetryImmediate:
		return pb.RecoveryAction_RECOVERY_ACTION_RETRY
	case domain.RecoveryRetryBackoff:
		return pb.RecoveryAction_RECOVERY_ACTION_RETRY_BACKOFF
	case domain.RecoveryFixInput:
		return pb.RecoveryAction_RECOVERY_ACTION_FIX_INPUT
	case domain.RecoveryWait:
		return pb.RecoveryAction_RECOVERY_ACTION_WAIT
	case domain.RecoveryEscalate:
		return pb.RecoveryAction_RECOVERY_ACTION_ESCALATE
	case domain.RecoveryAbort:
		return pb.RecoveryAction_RECOVERY_ACTION_NONE
	case domain.RecoveryUseAlternative:
		return pb.RecoveryAction_RECOVERY_ACTION_UNSPECIFIED
	default:
		return pb.RecoveryAction_RECOVERY_ACTION_UNSPECIFIED
	}
}

// =============================================================================
// HELPERS
// =============================================================================

// TimestampToProto converts a time.Time to a protobuf Timestamp.
func TimestampToProto(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

// TimestampFromProto converts a protobuf Timestamp to time.Time.
func TimestampFromProto(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

// DurationToProto converts a time.Duration to a protobuf Duration.
func DurationToProto(d time.Duration) *durationpb.Duration {
	if d == 0 {
		return nil
	}
	return durationpb.New(d)
}

// DurationFromProto converts a protobuf Duration to time.Duration.
func DurationFromProto(d *durationpb.Duration) time.Duration {
	if d == nil {
		return 0
	}
	return d.AsDuration()
}

// UUIDToString converts a UUID to string, handling nil.
func UUIDToString(id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}
	return id.String()
}

// UUIDFromString parses a string to UUID, returning Nil on error.
func UUIDFromString(s string) uuid.UUID {
	if s == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}

// OptionalUUIDToString converts an optional UUID pointer to string.
func OptionalUUIDToString(id *uuid.UUID) *string {
	if id == nil || *id == uuid.Nil {
		return nil
	}
	s := id.String()
	return &s
}

// OptionalStringToUUID parses an optional string to UUID pointer.
func OptionalStringToUUID(s *string) *uuid.UUID {
	if s == nil || *s == "" {
		return nil
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return nil
	}
	return &id
}
