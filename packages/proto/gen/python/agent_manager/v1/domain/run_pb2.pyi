import datetime

from agent_manager.v1.domain import profile_pb2 as _profile_pb2
from agent_manager.v1.domain import types_pb2 as _types_pb2
from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import duration_pb2 as _duration_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class FinalOutputSelectionStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FINAL_OUTPUT_SELECTION_STATUS_UNSPECIFIED: _ClassVar[FinalOutputSelectionStatus]
    FINAL_OUTPUT_SELECTION_STATUS_SELECTED: _ClassVar[FinalOutputSelectionStatus]
    FINAL_OUTPUT_SELECTION_STATUS_AMBIGUOUS: _ClassVar[FinalOutputSelectionStatus]
    FINAL_OUTPUT_SELECTION_STATUS_UNAVAILABLE: _ClassVar[FinalOutputSelectionStatus]

class StructuredResultStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    STRUCTURED_RESULT_STATUS_UNSPECIFIED: _ClassVar[StructuredResultStatus]
    STRUCTURED_RESULT_STATUS_SUCCESS: _ClassVar[StructuredResultStatus]
    STRUCTURED_RESULT_STATUS_UNAVAILABLE: _ClassVar[StructuredResultStatus]
    STRUCTURED_RESULT_STATUS_INVALID: _ClassVar[StructuredResultStatus]
    STRUCTURED_RESULT_STATUS_AMBIGUOUS: _ClassVar[StructuredResultStatus]
    STRUCTURED_RESULT_STATUS_ABSTAINED: _ClassVar[StructuredResultStatus]

class ReceiptObservationState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RECEIPT_OBSERVATION_STATE_UNSPECIFIED: _ClassVar[ReceiptObservationState]
    RECEIPT_OBSERVATION_STATE_AVAILABLE: _ClassVar[ReceiptObservationState]
    RECEIPT_OBSERVATION_STATE_UNOBSERVED: _ClassVar[ReceiptObservationState]
    RECEIPT_OBSERVATION_STATE_DEGRADED: _ClassVar[ReceiptObservationState]
    RECEIPT_OBSERVATION_STATE_UNAVAILABLE: _ClassVar[ReceiptObservationState]
FINAL_OUTPUT_SELECTION_STATUS_UNSPECIFIED: FinalOutputSelectionStatus
FINAL_OUTPUT_SELECTION_STATUS_SELECTED: FinalOutputSelectionStatus
FINAL_OUTPUT_SELECTION_STATUS_AMBIGUOUS: FinalOutputSelectionStatus
FINAL_OUTPUT_SELECTION_STATUS_UNAVAILABLE: FinalOutputSelectionStatus
STRUCTURED_RESULT_STATUS_UNSPECIFIED: StructuredResultStatus
STRUCTURED_RESULT_STATUS_SUCCESS: StructuredResultStatus
STRUCTURED_RESULT_STATUS_UNAVAILABLE: StructuredResultStatus
STRUCTURED_RESULT_STATUS_INVALID: StructuredResultStatus
STRUCTURED_RESULT_STATUS_AMBIGUOUS: StructuredResultStatus
STRUCTURED_RESULT_STATUS_ABSTAINED: StructuredResultStatus
RECEIPT_OBSERVATION_STATE_UNSPECIFIED: ReceiptObservationState
RECEIPT_OBSERVATION_STATE_AVAILABLE: ReceiptObservationState
RECEIPT_OBSERVATION_STATE_UNOBSERVED: ReceiptObservationState
RECEIPT_OBSERVATION_STATE_DEGRADED: ReceiptObservationState
RECEIPT_OBSERVATION_STATE_UNAVAILABLE: ReceiptObservationState

class Run(_message.Message):
    __slots__ = ("id", "task_id", "agent_profile_id", "tag", "sandbox_id", "run_mode", "status", "started_at", "ended_at", "phase", "last_checkpoint_id", "last_heartbeat", "progress_percent", "idempotency_key", "summary", "error_msg", "exit_code", "approval_state", "approved_by", "approved_at", "resolved_config", "diff_path", "log_path", "changed_files", "total_size_bytes", "commit_hash", "session_id", "created_at", "updated_at", "actions", "prompt_preview", "requested_model", "actual_model", "finalization_status", "finalization_error", "finalized_at", "await_handle", "execution_mode", "web_console_session_id", "web_console_session_url", "result")
    ID_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    TAG_FIELD_NUMBER: _ClassVar[int]
    SANDBOX_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_MODE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    ENDED_AT_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    LAST_CHECKPOINT_ID_FIELD_NUMBER: _ClassVar[int]
    LAST_HEARTBEAT_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_PERCENT_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    ERROR_MSG_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    APPROVAL_STATE_FIELD_NUMBER: _ClassVar[int]
    APPROVED_BY_FIELD_NUMBER: _ClassVar[int]
    APPROVED_AT_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_CONFIG_FIELD_NUMBER: _ClassVar[int]
    DIFF_PATH_FIELD_NUMBER: _ClassVar[int]
    LOG_PATH_FIELD_NUMBER: _ClassVar[int]
    CHANGED_FILES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    COMMIT_HASH_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    PROMPT_PREVIEW_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_MODEL_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_MODEL_FIELD_NUMBER: _ClassVar[int]
    FINALIZATION_STATUS_FIELD_NUMBER: _ClassVar[int]
    FINALIZATION_ERROR_FIELD_NUMBER: _ClassVar[int]
    FINALIZED_AT_FIELD_NUMBER: _ClassVar[int]
    AWAIT_HANDLE_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_MODE_FIELD_NUMBER: _ClassVar[int]
    WEB_CONSOLE_SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    WEB_CONSOLE_SESSION_URL_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    id: str
    task_id: str
    agent_profile_id: str
    tag: str
    sandbox_id: str
    run_mode: _types_pb2.RunMode
    status: _types_pb2.RunStatus
    started_at: _timestamp_pb2.Timestamp
    ended_at: _timestamp_pb2.Timestamp
    phase: _types_pb2.RunPhase
    last_checkpoint_id: str
    last_heartbeat: _timestamp_pb2.Timestamp
    progress_percent: int
    idempotency_key: str
    summary: RunSummary
    error_msg: str
    exit_code: int
    approval_state: _types_pb2.ApprovalState
    approved_by: str
    approved_at: _timestamp_pb2.Timestamp
    resolved_config: _profile_pb2.RunConfig
    diff_path: str
    log_path: str
    changed_files: int
    total_size_bytes: int
    commit_hash: str
    session_id: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    actions: RunActions
    prompt_preview: str
    requested_model: str
    actual_model: str
    finalization_status: _types_pb2.RunFinalizationStatus
    finalization_error: str
    finalized_at: _timestamp_pb2.Timestamp
    await_handle: AwaitHandle
    execution_mode: _types_pb2.ExecutionMode
    web_console_session_id: str
    web_console_session_url: str
    result: RunResult
    def __init__(self, id: _Optional[str] = ..., task_id: _Optional[str] = ..., agent_profile_id: _Optional[str] = ..., tag: _Optional[str] = ..., sandbox_id: _Optional[str] = ..., run_mode: _Optional[_Union[_types_pb2.RunMode, str]] = ..., status: _Optional[_Union[_types_pb2.RunStatus, str]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., ended_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., phase: _Optional[_Union[_types_pb2.RunPhase, str]] = ..., last_checkpoint_id: _Optional[str] = ..., last_heartbeat: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., progress_percent: _Optional[int] = ..., idempotency_key: _Optional[str] = ..., summary: _Optional[_Union[RunSummary, _Mapping]] = ..., error_msg: _Optional[str] = ..., exit_code: _Optional[int] = ..., approval_state: _Optional[_Union[_types_pb2.ApprovalState, str]] = ..., approved_by: _Optional[str] = ..., approved_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., resolved_config: _Optional[_Union[_profile_pb2.RunConfig, _Mapping]] = ..., diff_path: _Optional[str] = ..., log_path: _Optional[str] = ..., changed_files: _Optional[int] = ..., total_size_bytes: _Optional[int] = ..., commit_hash: _Optional[str] = ..., session_id: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., actions: _Optional[_Union[RunActions, _Mapping]] = ..., prompt_preview: _Optional[str] = ..., requested_model: _Optional[str] = ..., actual_model: _Optional[str] = ..., finalization_status: _Optional[_Union[_types_pb2.RunFinalizationStatus, str]] = ..., finalization_error: _Optional[str] = ..., finalized_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., await_handle: _Optional[_Union[AwaitHandle, _Mapping]] = ..., execution_mode: _Optional[_Union[_types_pb2.ExecutionMode, str]] = ..., web_console_session_id: _Optional[str] = ..., web_console_session_url: _Optional[str] = ..., result: _Optional[_Union[RunResult, _Mapping]] = ...) -> None: ...

class FinalOutputCandidate(_message.Message):
    __slots__ = ("id", "event_id", "sequence", "content", "message_id", "conversation_id", "turn_id", "provider_origin", "completion_reason", "terminal", "parent_message_id", "provider_event_type", "raw_evidence_ref", "evidence_tier")
    ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    CONVERSATION_ID_FIELD_NUMBER: _ClassVar[int]
    TURN_ID_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ORIGIN_FIELD_NUMBER: _ClassVar[int]
    COMPLETION_REASON_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_FIELD_NUMBER: _ClassVar[int]
    PARENT_MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_EVENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    RAW_EVIDENCE_REF_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_TIER_FIELD_NUMBER: _ClassVar[int]
    id: str
    event_id: str
    sequence: int
    content: str
    message_id: str
    conversation_id: str
    turn_id: str
    provider_origin: str
    completion_reason: str
    terminal: bool
    parent_message_id: str
    provider_event_type: str
    raw_evidence_ref: str
    evidence_tier: int
    def __init__(self, id: _Optional[str] = ..., event_id: _Optional[str] = ..., sequence: _Optional[int] = ..., content: _Optional[str] = ..., message_id: _Optional[str] = ..., conversation_id: _Optional[str] = ..., turn_id: _Optional[str] = ..., provider_origin: _Optional[str] = ..., completion_reason: _Optional[str] = ..., terminal: _Optional[bool] = ..., parent_message_id: _Optional[str] = ..., provider_event_type: _Optional[str] = ..., raw_evidence_ref: _Optional[str] = ..., evidence_tier: _Optional[int] = ...) -> None: ...

class FinalOutputSelection(_message.Message):
    __slots__ = ("status", "selected_candidate_id", "rule", "algorithm_version", "evidence")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SELECTED_CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    RULE_FIELD_NUMBER: _ClassVar[int]
    ALGORITHM_VERSION_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    status: FinalOutputSelectionStatus
    selected_candidate_id: str
    rule: str
    algorithm_version: str
    evidence: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, status: _Optional[_Union[FinalOutputSelectionStatus, str]] = ..., selected_candidate_id: _Optional[str] = ..., rule: _Optional[str] = ..., algorithm_version: _Optional[str] = ..., evidence: _Optional[_Iterable[str]] = ...) -> None: ...

class StructuredDiagnostic(_message.Message):
    __slots__ = ("code", "path", "message")
    CODE_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    code: str
    path: str
    message: str
    def __init__(self, code: _Optional[str] = ..., path: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class StructuredExtractionProvenance(_message.Message):
    __slots__ = ("role_ref", "provider", "model", "policy_snapshot")
    ROLE_REF_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    POLICY_SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    role_ref: str
    provider: str
    model: str
    policy_snapshot: _profile_pb2.ExecutionPolicySnapshot
    def __init__(self, role_ref: _Optional[str] = ..., provider: _Optional[str] = ..., model: _Optional[str] = ..., policy_snapshot: _Optional[_Union[_profile_pb2.ExecutionPolicySnapshot, _Mapping]] = ...) -> None: ...

class StructuredResult(_message.Message):
    __slots__ = ("status", "spec_kind", "schema_digest", "value", "method", "source_candidate_id", "extractor", "diagnostics")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SPEC_KIND_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_DIGEST_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    SOURCE_CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    EXTRACTOR_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTICS_FIELD_NUMBER: _ClassVar[int]
    status: StructuredResultStatus
    spec_kind: _profile_pb2.ResultSpecKind
    schema_digest: str
    value: bytes
    method: str
    source_candidate_id: str
    extractor: StructuredExtractionProvenance
    diagnostics: _containers.RepeatedCompositeFieldContainer[StructuredDiagnostic]
    def __init__(self, status: _Optional[_Union[StructuredResultStatus, str]] = ..., spec_kind: _Optional[_Union[_profile_pb2.ResultSpecKind, str]] = ..., schema_digest: _Optional[str] = ..., value: _Optional[bytes] = ..., method: _Optional[str] = ..., source_candidate_id: _Optional[str] = ..., extractor: _Optional[_Union[StructuredExtractionProvenance, _Mapping]] = ..., diagnostics: _Optional[_Iterable[_Union[StructuredDiagnostic, _Mapping]]] = ...) -> None: ...

class RunResult(_message.Message):
    __slots__ = ("final_output", "selection", "candidates", "success", "exit_code", "terminal_reason", "structured", "observations")
    FINAL_OUTPUT_FIELD_NUMBER: _ClassVar[int]
    SELECTION_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_REASON_FIELD_NUMBER: _ClassVar[int]
    STRUCTURED_FIELD_NUMBER: _ClassVar[int]
    OBSERVATIONS_FIELD_NUMBER: _ClassVar[int]
    final_output: str
    selection: FinalOutputSelection
    candidates: _containers.RepeatedCompositeFieldContainer[FinalOutputCandidate]
    success: bool
    exit_code: int
    terminal_reason: str
    structured: StructuredResult
    observations: ReceiptObservations
    def __init__(self, final_output: _Optional[str] = ..., selection: _Optional[_Union[FinalOutputSelection, _Mapping]] = ..., candidates: _Optional[_Iterable[_Union[FinalOutputCandidate, _Mapping]]] = ..., success: _Optional[bool] = ..., exit_code: _Optional[int] = ..., terminal_reason: _Optional[str] = ..., structured: _Optional[_Union[StructuredResult, _Mapping]] = ..., observations: _Optional[_Union[ReceiptObservations, _Mapping]] = ...) -> None: ...

class ReceiptObservations(_message.Message):
    __slots__ = ("state", "receipts", "reason")
    STATE_FIELD_NUMBER: _ClassVar[int]
    RECEIPTS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    state: ReceiptObservationState
    receipts: _containers.RepeatedCompositeFieldContainer[ObservedReceipt]
    reason: str
    def __init__(self, state: _Optional[_Union[ReceiptObservationState, str]] = ..., receipts: _Optional[_Iterable[_Union[ObservedReceipt, _Mapping]]] = ..., reason: _Optional[str] = ...) -> None: ...

class ObservedReceipt(_message.Message):
    __slots__ = ("event_id", "target_scenario", "operation", "agent_run_id", "workflow_execution_id", "workflow_node_id", "attempt", "attribution_verified", "outcome", "status_code", "projection")
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    AGENT_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_NODE_ID_FIELD_NUMBER: _ClassVar[int]
    ATTEMPT_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTION_VERIFIED_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    STATUS_CODE_FIELD_NUMBER: _ClassVar[int]
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    target_scenario: str
    operation: str
    agent_run_id: str
    workflow_execution_id: str
    workflow_node_id: str
    attempt: int
    attribution_verified: bool
    outcome: str
    status_code: int
    projection: _struct_pb2.Struct
    def __init__(self, event_id: _Optional[str] = ..., target_scenario: _Optional[str] = ..., operation: _Optional[str] = ..., agent_run_id: _Optional[str] = ..., workflow_execution_id: _Optional[str] = ..., workflow_node_id: _Optional[str] = ..., attempt: _Optional[int] = ..., attribution_verified: _Optional[bool] = ..., outcome: _Optional[str] = ..., status_code: _Optional[int] = ..., projection: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class AwaitHandle(_message.Message):
    __slots__ = ("producer", "key", "deadline", "registered_at")
    PRODUCER_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    DEADLINE_FIELD_NUMBER: _ClassVar[int]
    REGISTERED_AT_FIELD_NUMBER: _ClassVar[int]
    producer: str
    key: str
    deadline: _timestamp_pb2.Timestamp
    registered_at: _timestamp_pb2.Timestamp
    def __init__(self, producer: _Optional[str] = ..., key: _Optional[str] = ..., deadline: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., registered_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RunActions(_message.Message):
    __slots__ = ("can_investigate", "can_apply_investigation", "can_delete", "can_stop", "can_retry", "can_continue", "can_approve", "can_reject", "can_review", "can_continue_reason", "can_resume_from_failure", "can_resume_from_failure_reason", "finalization_warning", "can_retry_finalization")
    CAN_INVESTIGATE_FIELD_NUMBER: _ClassVar[int]
    CAN_APPLY_INVESTIGATION_FIELD_NUMBER: _ClassVar[int]
    CAN_DELETE_FIELD_NUMBER: _ClassVar[int]
    CAN_STOP_FIELD_NUMBER: _ClassVar[int]
    CAN_RETRY_FIELD_NUMBER: _ClassVar[int]
    CAN_CONTINUE_FIELD_NUMBER: _ClassVar[int]
    CAN_APPROVE_FIELD_NUMBER: _ClassVar[int]
    CAN_REJECT_FIELD_NUMBER: _ClassVar[int]
    CAN_REVIEW_FIELD_NUMBER: _ClassVar[int]
    CAN_CONTINUE_REASON_FIELD_NUMBER: _ClassVar[int]
    CAN_RESUME_FROM_FAILURE_FIELD_NUMBER: _ClassVar[int]
    CAN_RESUME_FROM_FAILURE_REASON_FIELD_NUMBER: _ClassVar[int]
    FINALIZATION_WARNING_FIELD_NUMBER: _ClassVar[int]
    CAN_RETRY_FINALIZATION_FIELD_NUMBER: _ClassVar[int]
    can_investigate: bool
    can_apply_investigation: bool
    can_delete: bool
    can_stop: bool
    can_retry: bool
    can_continue: bool
    can_approve: bool
    can_reject: bool
    can_review: bool
    can_continue_reason: str
    can_resume_from_failure: bool
    can_resume_from_failure_reason: str
    finalization_warning: str
    can_retry_finalization: bool
    def __init__(self, can_investigate: _Optional[bool] = ..., can_apply_investigation: _Optional[bool] = ..., can_delete: _Optional[bool] = ..., can_stop: _Optional[bool] = ..., can_retry: _Optional[bool] = ..., can_continue: _Optional[bool] = ..., can_approve: _Optional[bool] = ..., can_reject: _Optional[bool] = ..., can_review: _Optional[bool] = ..., can_continue_reason: _Optional[str] = ..., can_resume_from_failure: _Optional[bool] = ..., can_resume_from_failure_reason: _Optional[str] = ..., finalization_warning: _Optional[str] = ..., can_retry_finalization: _Optional[bool] = ...) -> None: ...

class RunSummary(_message.Message):
    __slots__ = ("description", "files_modified", "files_created", "files_deleted", "tokens_used", "turns_used", "cost_estimate", "context_tokens")
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    FILES_MODIFIED_FIELD_NUMBER: _ClassVar[int]
    FILES_CREATED_FIELD_NUMBER: _ClassVar[int]
    FILES_DELETED_FIELD_NUMBER: _ClassVar[int]
    TOKENS_USED_FIELD_NUMBER: _ClassVar[int]
    TURNS_USED_FIELD_NUMBER: _ClassVar[int]
    COST_ESTIMATE_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    description: str
    files_modified: _containers.RepeatedScalarFieldContainer[str]
    files_created: _containers.RepeatedScalarFieldContainer[str]
    files_deleted: _containers.RepeatedScalarFieldContainer[str]
    tokens_used: int
    turns_used: int
    cost_estimate: float
    context_tokens: int
    def __init__(self, description: _Optional[str] = ..., files_modified: _Optional[_Iterable[str]] = ..., files_created: _Optional[_Iterable[str]] = ..., files_deleted: _Optional[_Iterable[str]] = ..., tokens_used: _Optional[int] = ..., turns_used: _Optional[int] = ..., cost_estimate: _Optional[float] = ..., context_tokens: _Optional[int] = ...) -> None: ...

class RunCheckpoint(_message.Message):
    __slots__ = ("run_id", "phase", "step_within_phase", "sandbox_id", "work_dir", "lock_id", "last_event_sequence", "last_heartbeat", "retry_count", "saved_at", "metadata")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    STEP_WITHIN_PHASE_FIELD_NUMBER: _ClassVar[int]
    SANDBOX_ID_FIELD_NUMBER: _ClassVar[int]
    WORK_DIR_FIELD_NUMBER: _ClassVar[int]
    LOCK_ID_FIELD_NUMBER: _ClassVar[int]
    LAST_EVENT_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    LAST_HEARTBEAT_FIELD_NUMBER: _ClassVar[int]
    RETRY_COUNT_FIELD_NUMBER: _ClassVar[int]
    SAVED_AT_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    phase: _types_pb2.RunPhase
    step_within_phase: int
    sandbox_id: str
    work_dir: str
    lock_id: str
    last_event_sequence: int
    last_heartbeat: _timestamp_pb2.Timestamp
    retry_count: int
    saved_at: _timestamp_pb2.Timestamp
    metadata: _containers.ScalarMap[str, str]
    def __init__(self, run_id: _Optional[str] = ..., phase: _Optional[_Union[_types_pb2.RunPhase, str]] = ..., step_within_phase: _Optional[int] = ..., sandbox_id: _Optional[str] = ..., work_dir: _Optional[str] = ..., lock_id: _Optional[str] = ..., last_event_sequence: _Optional[int] = ..., last_heartbeat: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., retry_count: _Optional[int] = ..., saved_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., metadata: _Optional[_Mapping[str, str]] = ...) -> None: ...

class RunProgress(_message.Message):
    __slots__ = ("phase", "phase_description", "percent_complete", "current_action", "elapsed_time", "estimated_remaining", "last_update")
    PHASE_FIELD_NUMBER: _ClassVar[int]
    PHASE_DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PERCENT_COMPLETE_FIELD_NUMBER: _ClassVar[int]
    CURRENT_ACTION_FIELD_NUMBER: _ClassVar[int]
    ELAPSED_TIME_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_REMAINING_FIELD_NUMBER: _ClassVar[int]
    LAST_UPDATE_FIELD_NUMBER: _ClassVar[int]
    phase: _types_pb2.RunPhase
    phase_description: str
    percent_complete: int
    current_action: str
    elapsed_time: _duration_pb2.Duration
    estimated_remaining: _duration_pb2.Duration
    last_update: _timestamp_pb2.Timestamp
    def __init__(self, phase: _Optional[_Union[_types_pb2.RunPhase, str]] = ..., phase_description: _Optional[str] = ..., percent_complete: _Optional[int] = ..., current_action: _Optional[str] = ..., elapsed_time: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., estimated_remaining: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., last_update: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class IdempotencyRecord(_message.Message):
    __slots__ = ("key", "status", "entity_id", "entity_type", "created_at", "expires_at", "response")
    KEY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ENTITY_ID_FIELD_NUMBER: _ClassVar[int]
    ENTITY_TYPE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_FIELD_NUMBER: _ClassVar[int]
    key: str
    status: _types_pb2.IdempotencyStatus
    entity_id: str
    entity_type: str
    created_at: _timestamp_pb2.Timestamp
    expires_at: _timestamp_pb2.Timestamp
    response: bytes
    def __init__(self, key: _Optional[str] = ..., status: _Optional[_Union[_types_pb2.IdempotencyStatus, str]] = ..., entity_id: _Optional[str] = ..., entity_type: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., response: _Optional[bytes] = ...) -> None: ...

class RunnerStatus(_message.Message):
    __slots__ = ("runner_type", "available", "message", "install_hint", "supported_models", "capabilities")
    RUNNER_TYPE_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    INSTALL_HINT_FIELD_NUMBER: _ClassVar[int]
    SUPPORTED_MODELS_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    runner_type: _types_pb2.RunnerType
    available: bool
    message: str
    install_hint: str
    supported_models: _containers.RepeatedScalarFieldContainer[str]
    capabilities: RunnerCapabilities
    def __init__(self, runner_type: _Optional[_Union[_types_pb2.RunnerType, str]] = ..., available: _Optional[bool] = ..., message: _Optional[str] = ..., install_hint: _Optional[str] = ..., supported_models: _Optional[_Iterable[str]] = ..., capabilities: _Optional[_Union[RunnerCapabilities, _Mapping]] = ...) -> None: ...

class RunnerCapabilities(_message.Message):
    __slots__ = ("supports_streaming", "supports_messages", "supports_tool_events", "supports_cost_tracking", "supports_cancellation", "max_turns", "supports_continuation", "supported_features", "allowed_extra_flags", "supports_tool_restriction", "tool_restriction_mappings")
    class ToolRestrictionMappingsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SUPPORTS_STREAMING_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_MESSAGES_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_TOOL_EVENTS_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_COST_TRACKING_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_CANCELLATION_FIELD_NUMBER: _ClassVar[int]
    MAX_TURNS_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_CONTINUATION_FIELD_NUMBER: _ClassVar[int]
    SUPPORTED_FEATURES_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_EXTRA_FLAGS_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_TOOL_RESTRICTION_FIELD_NUMBER: _ClassVar[int]
    TOOL_RESTRICTION_MAPPINGS_FIELD_NUMBER: _ClassVar[int]
    supports_streaming: bool
    supports_messages: bool
    supports_tool_events: bool
    supports_cost_tracking: bool
    supports_cancellation: bool
    max_turns: int
    supports_continuation: bool
    supported_features: _containers.RepeatedScalarFieldContainer[str]
    allowed_extra_flags: _containers.RepeatedScalarFieldContainer[str]
    supports_tool_restriction: bool
    tool_restriction_mappings: _containers.ScalarMap[str, str]
    def __init__(self, supports_streaming: _Optional[bool] = ..., supports_messages: _Optional[bool] = ..., supports_tool_events: _Optional[bool] = ..., supports_cost_tracking: _Optional[bool] = ..., supports_cancellation: _Optional[bool] = ..., max_turns: _Optional[int] = ..., supports_continuation: _Optional[bool] = ..., supported_features: _Optional[_Iterable[str]] = ..., allowed_extra_flags: _Optional[_Iterable[str]] = ..., supports_tool_restriction: _Optional[bool] = ..., tool_restriction_mappings: _Optional[_Mapping[str, str]] = ...) -> None: ...

class ProbeResult(_message.Message):
    __slots__ = ("success", "latency_ms", "error", "details")
    class DetailsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    latency_ms: int
    error: str
    details: _containers.ScalarMap[str, str]
    def __init__(self, success: _Optional[bool] = ..., latency_ms: _Optional[int] = ..., error: _Optional[str] = ..., details: _Optional[_Mapping[str, str]] = ...) -> None: ...

class StopAllResult(_message.Message):
    __slots__ = ("stopped_count", "failures")
    STOPPED_COUNT_FIELD_NUMBER: _ClassVar[int]
    FAILURES_FIELD_NUMBER: _ClassVar[int]
    stopped_count: int
    failures: _containers.RepeatedCompositeFieldContainer[StopFailure]
    def __init__(self, stopped_count: _Optional[int] = ..., failures: _Optional[_Iterable[_Union[StopFailure, _Mapping]]] = ...) -> None: ...

class StopFailure(_message.Message):
    __slots__ = ("run_id", "error")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    error: str
    def __init__(self, run_id: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...

class ApproveResult(_message.Message):
    __slots__ = ("success", "files_applied", "commit_hash", "message", "remaining", "is_partial")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    FILES_APPLIED_FIELD_NUMBER: _ClassVar[int]
    COMMIT_HASH_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    REMAINING_FIELD_NUMBER: _ClassVar[int]
    IS_PARTIAL_FIELD_NUMBER: _ClassVar[int]
    success: bool
    files_applied: int
    commit_hash: str
    message: str
    remaining: int
    is_partial: bool
    def __init__(self, success: _Optional[bool] = ..., files_applied: _Optional[int] = ..., commit_hash: _Optional[str] = ..., message: _Optional[str] = ..., remaining: _Optional[int] = ..., is_partial: _Optional[bool] = ...) -> None: ...

class RunDiff(_message.Message):
    __slots__ = ("run_id", "content", "files", "generated_at")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    content: str
    files: _containers.RepeatedCompositeFieldContainer[FileDiff]
    generated_at: _timestamp_pb2.Timestamp
    def __init__(self, run_id: _Optional[str] = ..., content: _Optional[str] = ..., files: _Optional[_Iterable[_Union[FileDiff, _Mapping]]] = ..., generated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class FileDiff(_message.Message):
    __slots__ = ("path", "change_type", "additions", "deletions", "is_binary", "patch", "id")
    PATH_FIELD_NUMBER: _ClassVar[int]
    CHANGE_TYPE_FIELD_NUMBER: _ClassVar[int]
    ADDITIONS_FIELD_NUMBER: _ClassVar[int]
    DELETIONS_FIELD_NUMBER: _ClassVar[int]
    IS_BINARY_FIELD_NUMBER: _ClassVar[int]
    PATCH_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    path: str
    change_type: str
    additions: int
    deletions: int
    is_binary: bool
    patch: str
    id: str
    def __init__(self, path: _Optional[str] = ..., change_type: _Optional[str] = ..., additions: _Optional[int] = ..., deletions: _Optional[int] = ..., is_binary: _Optional[bool] = ..., patch: _Optional[str] = ..., id: _Optional[str] = ...) -> None: ...

class Attachment(_message.Message):
    __slots__ = ("id", "file_name", "content_type", "file_size", "storage_path", "url")
    ID_FIELD_NUMBER: _ClassVar[int]
    FILE_NAME_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    FILE_SIZE_FIELD_NUMBER: _ClassVar[int]
    STORAGE_PATH_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    id: str
    file_name: str
    content_type: str
    file_size: int
    storage_path: str
    url: str
    def __init__(self, id: _Optional[str] = ..., file_name: _Optional[str] = ..., content_type: _Optional[str] = ..., file_size: _Optional[int] = ..., storage_path: _Optional[str] = ..., url: _Optional[str] = ...) -> None: ...

class ContinueRunRequest(_message.Message):
    __slots__ = ("run_id", "message", "attachment_ids", "idempotency_key")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENT_IDS_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    message: str
    attachment_ids: _containers.RepeatedScalarFieldContainer[str]
    idempotency_key: str
    def __init__(self, run_id: _Optional[str] = ..., message: _Optional[str] = ..., attachment_ids: _Optional[_Iterable[str]] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class ContinueRunResponse(_message.Message):
    __slots__ = ("success", "run", "error", "error_code")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    RUN_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    ERROR_CODE_FIELD_NUMBER: _ClassVar[int]
    success: bool
    run: Run
    error: str
    error_code: str
    def __init__(self, success: _Optional[bool] = ..., run: _Optional[_Union[Run, _Mapping]] = ..., error: _Optional[str] = ..., error_code: _Optional[str] = ...) -> None: ...

class DeleteRunMessageRequest(_message.Message):
    __slots__ = ("run_id", "event_id")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    event_id: str
    def __init__(self, run_id: _Optional[str] = ..., event_id: _Optional[str] = ...) -> None: ...

class DeleteRunMessageResponse(_message.Message):
    __slots__ = ("success",)
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    def __init__(self, success: _Optional[bool] = ...) -> None: ...

class ParkRunRequest(_message.Message):
    __slots__ = ("run_id", "producer", "key", "deadline_unix", "identity_token")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    PRODUCER_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    DEADLINE_UNIX_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_TOKEN_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    producer: str
    key: str
    deadline_unix: int
    identity_token: str
    def __init__(self, run_id: _Optional[str] = ..., producer: _Optional[str] = ..., key: _Optional[str] = ..., deadline_unix: _Optional[int] = ..., identity_token: _Optional[str] = ...) -> None: ...

class ParkRunResponse(_message.Message):
    __slots__ = ("success", "run", "message", "refused", "result")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    RUN_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    REFUSED_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    run: Run
    message: str
    refused: bool
    result: str
    def __init__(self, success: _Optional[bool] = ..., run: _Optional[_Union[Run, _Mapping]] = ..., message: _Optional[str] = ..., refused: _Optional[bool] = ..., result: _Optional[str] = ...) -> None: ...

class GetAwaitResultResponse(_message.Message):
    __slots__ = ("found", "key", "result", "resolved_at")
    FOUND_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_AT_FIELD_NUMBER: _ClassVar[int]
    found: bool
    key: str
    result: str
    resolved_at: str
    def __init__(self, found: _Optional[bool] = ..., key: _Optional[str] = ..., result: _Optional[str] = ..., resolved_at: _Optional[str] = ...) -> None: ...

class WakeRunRequest(_message.Message):
    __slots__ = ("run_id", "result", "timed_out")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    TIMED_OUT_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    result: str
    timed_out: bool
    def __init__(self, run_id: _Optional[str] = ..., result: _Optional[str] = ..., timed_out: _Optional[bool] = ...) -> None: ...

class WakeRunResponse(_message.Message):
    __slots__ = ("success", "run")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    RUN_FIELD_NUMBER: _ClassVar[int]
    success: bool
    run: Run
    def __init__(self, success: _Optional[bool] = ..., run: _Optional[_Union[Run, _Mapping]] = ...) -> None: ...
