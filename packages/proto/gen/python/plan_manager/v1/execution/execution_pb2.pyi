from plan_manager.v1.shared import model_pb2 as _model_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Execution(_message.Message):
    __slots__ = ("id", "plan_id", "run_id", "current_phase_id", "complete", "started_at", "updated_at", "baseline_set", "scope_amendments", "degraded_reason", "lifecycle_state", "abandoned_reason", "abandoned_at", "abandoned_by", "boundary_extensions")
    ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    CURRENT_PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    BASELINE_SET_FIELD_NUMBER: _ClassVar[int]
    SCOPE_AMENDMENTS_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASON_FIELD_NUMBER: _ClassVar[int]
    LIFECYCLE_STATE_FIELD_NUMBER: _ClassVar[int]
    ABANDONED_REASON_FIELD_NUMBER: _ClassVar[int]
    ABANDONED_AT_FIELD_NUMBER: _ClassVar[int]
    ABANDONED_BY_FIELD_NUMBER: _ClassVar[int]
    BOUNDARY_EXTENSIONS_FIELD_NUMBER: _ClassVar[int]
    id: str
    plan_id: str
    run_id: str
    current_phase_id: str
    complete: bool
    started_at: str
    updated_at: str
    baseline_set: BaselineSetState
    scope_amendments: _containers.RepeatedCompositeFieldContainer[ScopeAmendment]
    degraded_reason: str
    lifecycle_state: str
    abandoned_reason: str
    abandoned_at: str
    abandoned_by: str
    boundary_extensions: _containers.RepeatedCompositeFieldContainer[BoundaryExtension]
    def __init__(self, id: _Optional[str] = ..., plan_id: _Optional[str] = ..., run_id: _Optional[str] = ..., current_phase_id: _Optional[str] = ..., complete: _Optional[bool] = ..., started_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., baseline_set: _Optional[_Union[BaselineSetState, _Mapping]] = ..., scope_amendments: _Optional[_Iterable[_Union[ScopeAmendment, _Mapping]]] = ..., degraded_reason: _Optional[str] = ..., lifecycle_state: _Optional[str] = ..., abandoned_reason: _Optional[str] = ..., abandoned_at: _Optional[str] = ..., abandoned_by: _Optional[str] = ..., boundary_extensions: _Optional[_Iterable[_Union[BoundaryExtension, _Mapping]]] = ...) -> None: ...

class BaselineSetState(_message.Message):
    __slots__ = ("version", "name", "scenario_targets", "repo_paths", "captured_at", "status", "required", "ready", "pending", "failed", "skipped", "stale", "detail", "collection_branch", "members", "path_snapshots", "capture_argv", "wait_argv", "sync_argv", "last_synced_at", "source_preflight", "preflight_unavailable")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_TARGETS_FIELD_NUMBER: _ClassVar[int]
    REPO_PATHS_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    PENDING_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    STALE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    COLLECTION_BRANCH_FIELD_NUMBER: _ClassVar[int]
    MEMBERS_FIELD_NUMBER: _ClassVar[int]
    PATH_SNAPSHOTS_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_ARGV_FIELD_NUMBER: _ClassVar[int]
    WAIT_ARGV_FIELD_NUMBER: _ClassVar[int]
    SYNC_ARGV_FIELD_NUMBER: _ClassVar[int]
    LAST_SYNCED_AT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PREFLIGHT_FIELD_NUMBER: _ClassVar[int]
    PREFLIGHT_UNAVAILABLE_FIELD_NUMBER: _ClassVar[int]
    version: int
    name: str
    scenario_targets: _containers.RepeatedScalarFieldContainer[str]
    repo_paths: _containers.RepeatedScalarFieldContainer[str]
    captured_at: str
    status: str
    required: int
    ready: int
    pending: int
    failed: int
    skipped: int
    stale: int
    detail: str
    collection_branch: str
    members: _containers.RepeatedCompositeFieldContainer[BaselineSetMember]
    path_snapshots: _containers.RepeatedCompositeFieldContainer[BaselineSetPathSnapshot]
    capture_argv: _containers.RepeatedScalarFieldContainer[str]
    wait_argv: _containers.RepeatedScalarFieldContainer[str]
    sync_argv: _containers.RepeatedScalarFieldContainer[str]
    last_synced_at: str
    source_preflight: SourceEvidencePreflight
    preflight_unavailable: bool
    def __init__(self, version: _Optional[int] = ..., name: _Optional[str] = ..., scenario_targets: _Optional[_Iterable[str]] = ..., repo_paths: _Optional[_Iterable[str]] = ..., captured_at: _Optional[str] = ..., status: _Optional[str] = ..., required: _Optional[int] = ..., ready: _Optional[int] = ..., pending: _Optional[int] = ..., failed: _Optional[int] = ..., skipped: _Optional[int] = ..., stale: _Optional[int] = ..., detail: _Optional[str] = ..., collection_branch: _Optional[str] = ..., members: _Optional[_Iterable[_Union[BaselineSetMember, _Mapping]]] = ..., path_snapshots: _Optional[_Iterable[_Union[BaselineSetPathSnapshot, _Mapping]]] = ..., capture_argv: _Optional[_Iterable[str]] = ..., wait_argv: _Optional[_Iterable[str]] = ..., sync_argv: _Optional[_Iterable[str]] = ..., last_synced_at: _Optional[str] = ..., source_preflight: _Optional[_Union[SourceEvidencePreflight, _Mapping]] = ..., preflight_unavailable: _Optional[bool] = ...) -> None: ...

class SourceEvidencePreflight(_message.Message):
    __slots__ = ("eligible_files", "eligible_bytes", "retained_content_bytes", "repair_required", "issues", "recommendations", "policy_version", "include_ignored", "retain_content", "excluded_ignored_files", "excluded_ignored_bytes", "excluded_sensitive_files", "excluded_binary_files", "oversized_files", "top_contributors")
    ELIGIBLE_FILES_FIELD_NUMBER: _ClassVar[int]
    ELIGIBLE_BYTES_FIELD_NUMBER: _ClassVar[int]
    RETAINED_CONTENT_BYTES_FIELD_NUMBER: _ClassVar[int]
    REPAIR_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATIONS_FIELD_NUMBER: _ClassVar[int]
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_IGNORED_FIELD_NUMBER: _ClassVar[int]
    RETAIN_CONTENT_FIELD_NUMBER: _ClassVar[int]
    EXCLUDED_IGNORED_FILES_FIELD_NUMBER: _ClassVar[int]
    EXCLUDED_IGNORED_BYTES_FIELD_NUMBER: _ClassVar[int]
    EXCLUDED_SENSITIVE_FILES_FIELD_NUMBER: _ClassVar[int]
    EXCLUDED_BINARY_FILES_FIELD_NUMBER: _ClassVar[int]
    OVERSIZED_FILES_FIELD_NUMBER: _ClassVar[int]
    TOP_CONTRIBUTORS_FIELD_NUMBER: _ClassVar[int]
    eligible_files: int
    eligible_bytes: int
    retained_content_bytes: int
    repair_required: bool
    issues: _containers.RepeatedCompositeFieldContainer[SourceEvidenceIssue]
    recommendations: _containers.RepeatedCompositeFieldContainer[SourceEvidenceRecommendation]
    policy_version: int
    include_ignored: bool
    retain_content: bool
    excluded_ignored_files: int
    excluded_ignored_bytes: int
    excluded_sensitive_files: int
    excluded_binary_files: int
    oversized_files: int
    top_contributors: _containers.RepeatedCompositeFieldContainer[SourceEvidenceContributor]
    def __init__(self, eligible_files: _Optional[int] = ..., eligible_bytes: _Optional[int] = ..., retained_content_bytes: _Optional[int] = ..., repair_required: _Optional[bool] = ..., issues: _Optional[_Iterable[_Union[SourceEvidenceIssue, _Mapping]]] = ..., recommendations: _Optional[_Iterable[_Union[SourceEvidenceRecommendation, _Mapping]]] = ..., policy_version: _Optional[int] = ..., include_ignored: _Optional[bool] = ..., retain_content: _Optional[bool] = ..., excluded_ignored_files: _Optional[int] = ..., excluded_ignored_bytes: _Optional[int] = ..., excluded_sensitive_files: _Optional[int] = ..., excluded_binary_files: _Optional[int] = ..., oversized_files: _Optional[int] = ..., top_contributors: _Optional[_Iterable[_Union[SourceEvidenceContributor, _Mapping]]] = ...) -> None: ...

class SourceEvidenceIssue(_message.Message):
    __slots__ = ("code", "severity", "detail")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    code: str
    severity: str
    detail: str
    def __init__(self, code: _Optional[str] = ..., severity: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class SourceEvidenceRecommendation(_message.Message):
    __slots__ = ("selection", "reason")
    SELECTION_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    selection: str
    reason: str
    def __init__(self, selection: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class SourceEvidenceContributor(_message.Message):
    __slots__ = ("path", "files", "bytes")
    PATH_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    BYTES_FIELD_NUMBER: _ClassVar[int]
    path: str
    files: int
    bytes: int
    def __init__(self, path: _Optional[str] = ..., files: _Optional[int] = ..., bytes: _Optional[int] = ...) -> None: ...

class BaselineSetMember(_message.Message):
    __slots__ = ("scenario", "baseline_name", "required", "status", "run_id", "error", "git_sha")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    BASELINE_NAME_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    GIT_SHA_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    baseline_name: str
    required: bool
    status: str
    run_id: str
    error: str
    git_sha: str
    def __init__(self, scenario: _Optional[str] = ..., baseline_name: _Optional[str] = ..., required: _Optional[bool] = ..., status: _Optional[str] = ..., run_id: _Optional[str] = ..., error: _Optional[str] = ..., git_sha: _Optional[str] = ...) -> None: ...

class BaselineSetPathSnapshot(_message.Message):
    __slots__ = ("name", "branch", "created_at")
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    name: str
    branch: str
    created_at: str
    def __init__(self, name: _Optional[str] = ..., branch: _Optional[str] = ..., created_at: _Optional[str] = ...) -> None: ...

class ScopeAmendment(_message.Message):
    __slots__ = ("id", "phase_id", "author", "reason", "old_minimum", "new_minimum", "invalidated_at", "created_at", "invalidated_ticket_ids")
    ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    AUTHOR_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    OLD_MINIMUM_FIELD_NUMBER: _ClassVar[int]
    NEW_MINIMUM_FIELD_NUMBER: _ClassVar[int]
    INVALIDATED_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    INVALIDATED_TICKET_IDS_FIELD_NUMBER: _ClassVar[int]
    id: str
    phase_id: str
    author: str
    reason: str
    old_minimum: _containers.RepeatedScalarFieldContainer[str]
    new_minimum: _containers.RepeatedScalarFieldContainer[str]
    invalidated_at: str
    created_at: str
    invalidated_ticket_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., phase_id: _Optional[str] = ..., author: _Optional[str] = ..., reason: _Optional[str] = ..., old_minimum: _Optional[_Iterable[str]] = ..., new_minimum: _Optional[_Iterable[str]] = ..., invalidated_at: _Optional[str] = ..., created_at: _Optional[str] = ..., invalidated_ticket_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class PhaseContext(_message.Message):
    __slots__ = ("current_phase", "next_phase", "required_reading", "reminders", "last_validation", "staleness", "resume_phase_id", "completeness", "relevant_context", "log_summary", "inputs_freshened", "freshen_status", "freshen_detail", "change_boundary", "feedback_checkpoint", "baseline_set", "scope_generation")
    CURRENT_PHASE_FIELD_NUMBER: _ClassVar[int]
    NEXT_PHASE_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_READING_FIELD_NUMBER: _ClassVar[int]
    REMINDERS_FIELD_NUMBER: _ClassVar[int]
    LAST_VALIDATION_FIELD_NUMBER: _ClassVar[int]
    STALENESS_FIELD_NUMBER: _ClassVar[int]
    RESUME_PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    COMPLETENESS_FIELD_NUMBER: _ClassVar[int]
    RELEVANT_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    LOG_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    INPUTS_FRESHENED_FIELD_NUMBER: _ClassVar[int]
    FRESHEN_STATUS_FIELD_NUMBER: _ClassVar[int]
    FRESHEN_DETAIL_FIELD_NUMBER: _ClassVar[int]
    CHANGE_BOUNDARY_FIELD_NUMBER: _ClassVar[int]
    FEEDBACK_CHECKPOINT_FIELD_NUMBER: _ClassVar[int]
    BASELINE_SET_FIELD_NUMBER: _ClassVar[int]
    SCOPE_GENERATION_FIELD_NUMBER: _ClassVar[int]
    current_phase: _model_pb2.Phase
    next_phase: _model_pb2.Phase
    required_reading: _containers.RepeatedScalarFieldContainer[str]
    reminders: _containers.RepeatedScalarFieldContainer[str]
    last_validation: _model_pb2.ValidationResult
    staleness: _model_pb2.StalenessTier
    resume_phase_id: str
    completeness: _model_pb2.Completeness
    relevant_context: _containers.RepeatedCompositeFieldContainer[_model_pb2.RelevantContextItem]
    log_summary: _model_pb2.LogSummary
    inputs_freshened: bool
    freshen_status: str
    freshen_detail: str
    change_boundary: _model_pb2.ChangeBoundary
    feedback_checkpoint: PhaseFeedbackCheckpoint
    baseline_set: BaselineSetState
    scope_generation: int
    def __init__(self, current_phase: _Optional[_Union[_model_pb2.Phase, _Mapping]] = ..., next_phase: _Optional[_Union[_model_pb2.Phase, _Mapping]] = ..., required_reading: _Optional[_Iterable[str]] = ..., reminders: _Optional[_Iterable[str]] = ..., last_validation: _Optional[_Union[_model_pb2.ValidationResult, _Mapping]] = ..., staleness: _Optional[_Union[_model_pb2.StalenessTier, str]] = ..., resume_phase_id: _Optional[str] = ..., completeness: _Optional[_Union[_model_pb2.Completeness, str]] = ..., relevant_context: _Optional[_Iterable[_Union[_model_pb2.RelevantContextItem, _Mapping]]] = ..., log_summary: _Optional[_Union[_model_pb2.LogSummary, _Mapping]] = ..., inputs_freshened: _Optional[bool] = ..., freshen_status: _Optional[str] = ..., freshen_detail: _Optional[str] = ..., change_boundary: _Optional[_Union[_model_pb2.ChangeBoundary, _Mapping]] = ..., feedback_checkpoint: _Optional[_Union[PhaseFeedbackCheckpoint, _Mapping]] = ..., baseline_set: _Optional[_Union[BaselineSetState, _Mapping]] = ..., scope_generation: _Optional[int] = ...) -> None: ...

class PhaseFeedbackCheckpoint(_message.Message):
    __slots__ = ("phase_id", "reviewed", "satisfied", "summary", "decisions", "findings", "bug_reports", "records", "notes", "pending_sync", "failed_sync", "no_feedback_title", "no_feedback_detail")
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    REVIEWED_FIELD_NUMBER: _ClassVar[int]
    SATISFIED_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    DECISIONS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    BUG_REPORTS_FIELD_NUMBER: _ClassVar[int]
    RECORDS_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    PENDING_SYNC_FIELD_NUMBER: _ClassVar[int]
    FAILED_SYNC_FIELD_NUMBER: _ClassVar[int]
    NO_FEEDBACK_TITLE_FIELD_NUMBER: _ClassVar[int]
    NO_FEEDBACK_DETAIL_FIELD_NUMBER: _ClassVar[int]
    phase_id: str
    reviewed: bool
    satisfied: bool
    summary: str
    decisions: int
    findings: int
    bug_reports: int
    records: int
    notes: int
    pending_sync: int
    failed_sync: int
    no_feedback_title: str
    no_feedback_detail: str
    def __init__(self, phase_id: _Optional[str] = ..., reviewed: _Optional[bool] = ..., satisfied: _Optional[bool] = ..., summary: _Optional[str] = ..., decisions: _Optional[int] = ..., findings: _Optional[int] = ..., bug_reports: _Optional[int] = ..., records: _Optional[int] = ..., notes: _Optional[int] = ..., pending_sync: _Optional[int] = ..., failed_sync: _Optional[int] = ..., no_feedback_title: _Optional[str] = ..., no_feedback_detail: _Optional[str] = ...) -> None: ...

class CompletionNudge(_message.Message):
    __slots__ = ("kind", "message", "satisfied")
    KIND_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SATISFIED_FIELD_NUMBER: _ClassVar[int]
    kind: str
    message: str
    satisfied: bool
    def __init__(self, kind: _Optional[str] = ..., message: _Optional[str] = ..., satisfied: _Optional[bool] = ...) -> None: ...

class StartRequest(_message.Message):
    __slots__ = ("plan_id", "run_id")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    run_id: str
    def __init__(self, plan_id: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class StartResponse(_message.Message):
    __slots__ = ("execution", "step", "context")
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    execution: Execution
    step: _model_pb2.GuidedStep
    context: PhaseContext
    def __init__(self, execution: _Optional[_Union[Execution, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ..., context: _Optional[_Union[PhaseContext, _Mapping]] = ...) -> None: ...

class GetStatusRequest(_message.Message):
    __slots__ = ("execution_id",)
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    def __init__(self, execution_id: _Optional[str] = ...) -> None: ...

class GetStatusResponse(_message.Message):
    __slots__ = ("execution", "context", "step")
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    execution: Execution
    context: PhaseContext
    step: _model_pb2.GuidedStep
    def __init__(self, execution: _Optional[_Union[Execution, _Mapping]] = ..., context: _Optional[_Union[PhaseContext, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class GetContextRequest(_message.Message):
    __slots__ = ("execution_id", "phase_id")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    phase_id: str
    def __init__(self, execution_id: _Optional[str] = ..., phase_id: _Optional[str] = ...) -> None: ...

class GetContextResponse(_message.Message):
    __slots__ = ("execution", "context", "step")
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    execution: Execution
    context: PhaseContext
    step: _model_pb2.GuidedStep
    def __init__(self, execution: _Optional[_Union[Execution, _Mapping]] = ..., context: _Optional[_Union[PhaseContext, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class ResumeRequest(_message.Message):
    __slots__ = ("plan_or_execution", "phase_id", "run_id")
    PLAN_OR_EXECUTION_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    plan_or_execution: str
    phase_id: str
    run_id: str
    def __init__(self, plan_or_execution: _Optional[str] = ..., phase_id: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class ResumeResponse(_message.Message):
    __slots__ = ("execution", "context", "step")
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    execution: Execution
    context: PhaseContext
    step: _model_pb2.GuidedStep
    def __init__(self, execution: _Optional[_Union[Execution, _Mapping]] = ..., context: _Optional[_Union[PhaseContext, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class ContinueExecutionRequest(_message.Message):
    __slots__ = ("plan_or_execution", "phase_id", "run_id")
    PLAN_OR_EXECUTION_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    plan_or_execution: str
    phase_id: str
    run_id: str
    def __init__(self, plan_or_execution: _Optional[str] = ..., phase_id: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class ContinueExecutionResponse(_message.Message):
    __slots__ = ("execution", "context", "step")
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    execution: Execution
    context: PhaseContext
    step: _model_pb2.GuidedStep
    def __init__(self, execution: _Optional[_Union[Execution, _Mapping]] = ..., context: _Optional[_Union[PhaseContext, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class AbandonExecutionRequest(_message.Message):
    __slots__ = ("execution_id", "reason", "actor")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    reason: str
    actor: str
    def __init__(self, execution_id: _Optional[str] = ..., reason: _Optional[str] = ..., actor: _Optional[str] = ...) -> None: ...

class AbandonExecutionResponse(_message.Message):
    __slots__ = ("execution", "already_abandoned", "step")
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    ALREADY_ABANDONED_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    execution: Execution
    already_abandoned: bool
    step: _model_pb2.GuidedStep
    def __init__(self, execution: _Optional[_Union[Execution, _Mapping]] = ..., already_abandoned: _Optional[bool] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class ActiveExecutionConflict(_message.Message):
    __slots__ = ("execution_ids", "resume_commands", "abandon_commands")
    EXECUTION_IDS_FIELD_NUMBER: _ClassVar[int]
    RESUME_COMMANDS_FIELD_NUMBER: _ClassVar[int]
    ABANDON_COMMANDS_FIELD_NUMBER: _ClassVar[int]
    execution_ids: _containers.RepeatedScalarFieldContainer[str]
    resume_commands: _containers.RepeatedScalarFieldContainer[str]
    abandon_commands: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, execution_ids: _Optional[_Iterable[str]] = ..., resume_commands: _Optional[_Iterable[str]] = ..., abandon_commands: _Optional[_Iterable[str]] = ...) -> None: ...

class SyncBaselineRequest(_message.Message):
    __slots__ = ("execution_id",)
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    def __init__(self, execution_id: _Optional[str] = ...) -> None: ...

class SyncBaselineResponse(_message.Message):
    __slots__ = ("execution", "context", "step")
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    execution: Execution
    context: PhaseContext
    step: _model_pb2.GuidedStep
    def __init__(self, execution: _Optional[_Union[Execution, _Mapping]] = ..., context: _Optional[_Union[PhaseContext, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class AmendScopeRequest(_message.Message):
    __slots__ = ("execution_id", "phase_id", "member", "author", "reason")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    MEMBER_FIELD_NUMBER: _ClassVar[int]
    AUTHOR_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    phase_id: str
    member: _containers.RepeatedScalarFieldContainer[str]
    author: str
    reason: str
    def __init__(self, execution_id: _Optional[str] = ..., phase_id: _Optional[str] = ..., member: _Optional[_Iterable[str]] = ..., author: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class AmendScopeResponse(_message.Message):
    __slots__ = ("execution", "context", "step")
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    execution: Execution
    context: PhaseContext
    step: _model_pb2.GuidedStep
    def __init__(self, execution: _Optional[_Union[Execution, _Mapping]] = ..., context: _Optional[_Union[PhaseContext, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class AdoptBaselineRequest(_message.Message):
    __slots__ = ("execution_id", "mode", "name", "member", "path", "reason")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    MEMBER_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    mode: str
    name: str
    member: _containers.RepeatedScalarFieldContainer[str]
    path: _containers.RepeatedScalarFieldContainer[str]
    reason: str
    def __init__(self, execution_id: _Optional[str] = ..., mode: _Optional[str] = ..., name: _Optional[str] = ..., member: _Optional[_Iterable[str]] = ..., path: _Optional[_Iterable[str]] = ..., reason: _Optional[str] = ...) -> None: ...

class AdoptBaselineResponse(_message.Message):
    __slots__ = ("execution", "context", "step")
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    execution: Execution
    context: PhaseContext
    step: _model_pb2.GuidedStep
    def __init__(self, execution: _Optional[_Union[Execution, _Mapping]] = ..., context: _Optional[_Union[PhaseContext, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class RepairSourceScopeRequest(_message.Message):
    __slots__ = ("execution_id", "path", "reason")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    path: _containers.RepeatedScalarFieldContainer[str]
    reason: str
    def __init__(self, execution_id: _Optional[str] = ..., path: _Optional[_Iterable[str]] = ..., reason: _Optional[str] = ...) -> None: ...

class RepairSourceScopeResponse(_message.Message):
    __slots__ = ("execution", "context", "step")
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    execution: Execution
    context: PhaseContext
    step: _model_pb2.GuidedStep
    def __init__(self, execution: _Optional[_Union[Execution, _Mapping]] = ..., context: _Optional[_Union[PhaseContext, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class BoundaryExtension(_message.Message):
    __slots__ = ("id", "phase_id", "author", "reason", "added_allow", "old_allow", "new_allow", "invalidated_at", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    AUTHOR_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    ADDED_ALLOW_FIELD_NUMBER: _ClassVar[int]
    OLD_ALLOW_FIELD_NUMBER: _ClassVar[int]
    NEW_ALLOW_FIELD_NUMBER: _ClassVar[int]
    INVALIDATED_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    phase_id: str
    author: str
    reason: str
    added_allow: _containers.RepeatedScalarFieldContainer[str]
    old_allow: _containers.RepeatedScalarFieldContainer[str]
    new_allow: _containers.RepeatedScalarFieldContainer[str]
    invalidated_at: str
    created_at: str
    def __init__(self, id: _Optional[str] = ..., phase_id: _Optional[str] = ..., author: _Optional[str] = ..., reason: _Optional[str] = ..., added_allow: _Optional[_Iterable[str]] = ..., old_allow: _Optional[_Iterable[str]] = ..., new_allow: _Optional[_Iterable[str]] = ..., invalidated_at: _Optional[str] = ..., created_at: _Optional[str] = ...) -> None: ...

class ExtendBoundaryRequest(_message.Message):
    __slots__ = ("execution_id", "path", "reason", "author")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    AUTHOR_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    path: _containers.RepeatedScalarFieldContainer[str]
    reason: str
    author: str
    def __init__(self, execution_id: _Optional[str] = ..., path: _Optional[_Iterable[str]] = ..., reason: _Optional[str] = ..., author: _Optional[str] = ...) -> None: ...

class ExtendBoundaryResponse(_message.Message):
    __slots__ = ("execution", "context", "step", "added_allow")
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    ADDED_ALLOW_FIELD_NUMBER: _ClassVar[int]
    execution: Execution
    context: PhaseContext
    step: _model_pb2.GuidedStep
    added_allow: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, execution: _Optional[_Union[Execution, _Mapping]] = ..., context: _Optional[_Union[PhaseContext, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ..., added_allow: _Optional[_Iterable[str]] = ...) -> None: ...

class GetNextRequest(_message.Message):
    __slots__ = ("execution_id",)
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    def __init__(self, execution_id: _Optional[str] = ...) -> None: ...

class GetNextResponse(_message.Message):
    __slots__ = ("context", "complete", "step")
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    context: PhaseContext
    complete: bool
    step: _model_pb2.GuidedStep
    def __init__(self, context: _Optional[_Union[PhaseContext, _Mapping]] = ..., complete: _Optional[bool] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class TransitionPhaseRequest(_message.Message):
    __slots__ = ("execution_id", "phase_id", "to_status", "validation_override", "feedback_override")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    TO_STATUS_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    FEEDBACK_OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    phase_id: str
    to_status: _model_pb2.PhaseStatus
    validation_override: ValidationOverride
    feedback_override: FeedbackOverride
    def __init__(self, execution_id: _Optional[str] = ..., phase_id: _Optional[str] = ..., to_status: _Optional[_Union[_model_pb2.PhaseStatus, str]] = ..., validation_override: _Optional[_Union[ValidationOverride, _Mapping]] = ..., feedback_override: _Optional[_Union[FeedbackOverride, _Mapping]] = ...) -> None: ...

class TransitionPhaseResponse(_message.Message):
    __slots__ = ("execution", "plan", "step")
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    PLAN_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    execution: Execution
    plan: _model_pb2.Plan
    step: _model_pb2.GuidedStep
    def __init__(self, execution: _Optional[_Union[Execution, _Mapping]] = ..., plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class ValidationOverride(_message.Message):
    __slots__ = ("reason",)
    REASON_FIELD_NUMBER: _ClassVar[int]
    reason: str
    def __init__(self, reason: _Optional[str] = ...) -> None: ...

class FeedbackOverride(_message.Message):
    __slots__ = ("reason",)
    REASON_FIELD_NUMBER: _ClassVar[int]
    reason: str
    def __init__(self, reason: _Optional[str] = ...) -> None: ...

class CompleteRequest(_message.Message):
    __slots__ = ("execution_id", "tokens", "iterations")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    TOKENS_FIELD_NUMBER: _ClassVar[int]
    ITERATIONS_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    tokens: int
    iterations: int
    def __init__(self, execution_id: _Optional[str] = ..., tokens: _Optional[int] = ..., iterations: _Optional[int] = ...) -> None: ...

class CompleteResponse(_message.Message):
    __slots__ = ("handoff", "nudges", "step")
    HANDOFF_FIELD_NUMBER: _ClassVar[int]
    NUDGES_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    handoff: _model_pb2.Handoff
    nudges: _containers.RepeatedCompositeFieldContainer[CompletionNudge]
    step: _model_pb2.GuidedStep
    def __init__(self, handoff: _Optional[_Union[_model_pb2.Handoff, _Mapping]] = ..., nudges: _Optional[_Iterable[_Union[CompletionNudge, _Mapping]]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class PartialHandoffRequest(_message.Message):
    __slots__ = ("execution_id", "tokens", "iterations")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    TOKENS_FIELD_NUMBER: _ClassVar[int]
    ITERATIONS_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    tokens: int
    iterations: int
    def __init__(self, execution_id: _Optional[str] = ..., tokens: _Optional[int] = ..., iterations: _Optional[int] = ...) -> None: ...

class PartialHandoffResponse(_message.Message):
    __slots__ = ("handoff", "nudges", "step")
    HANDOFF_FIELD_NUMBER: _ClassVar[int]
    NUDGES_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    handoff: _model_pb2.Handoff
    nudges: _containers.RepeatedCompositeFieldContainer[CompletionNudge]
    step: _model_pb2.GuidedStep
    def __init__(self, handoff: _Optional[_Union[_model_pb2.Handoff, _Mapping]] = ..., nudges: _Optional[_Iterable[_Union[CompletionNudge, _Mapping]]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class GetHandoffRequest(_message.Message):
    __slots__ = ("execution_id",)
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    def __init__(self, execution_id: _Optional[str] = ...) -> None: ...

class GetHandoffResponse(_message.Message):
    __slots__ = ("handoff", "step")
    HANDOFF_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    handoff: _model_pb2.Handoff
    step: _model_pb2.GuidedStep
    def __init__(self, handoff: _Optional[_Union[_model_pb2.Handoff, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class GetVelocityRequest(_message.Message):
    __slots__ = ("plan_id",)
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    def __init__(self, plan_id: _Optional[str] = ...) -> None: ...

class GetVelocityResponse(_message.Message):
    __slots__ = ("points", "step")
    POINTS_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    points: _containers.RepeatedCompositeFieldContainer[_model_pb2.VelocityPoint]
    step: _model_pb2.GuidedStep
    def __init__(self, points: _Optional[_Iterable[_Union[_model_pb2.VelocityPoint, _Mapping]]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...
