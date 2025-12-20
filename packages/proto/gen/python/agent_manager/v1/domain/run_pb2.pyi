import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import duration_pb2 as _duration_pb2
from buf.validate import validate_pb2 as _validate_pb2
from agent_manager.v1.domain import types_pb2 as _types_pb2
from agent_manager.v1.domain import profile_pb2 as _profile_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Run(_message.Message):
    __slots__ = ()
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
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
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
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., task_id: _Optional[str] = ..., agent_profile_id: _Optional[str] = ..., tag: _Optional[str] = ..., sandbox_id: _Optional[str] = ..., run_mode: _Optional[_Union[_types_pb2.RunMode, str]] = ..., status: _Optional[_Union[_types_pb2.RunStatus, str]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., ended_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., phase: _Optional[_Union[_types_pb2.RunPhase, str]] = ..., last_checkpoint_id: _Optional[str] = ..., last_heartbeat: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., progress_percent: _Optional[int] = ..., idempotency_key: _Optional[str] = ..., summary: _Optional[_Union[RunSummary, _Mapping]] = ..., error_msg: _Optional[str] = ..., exit_code: _Optional[int] = ..., approval_state: _Optional[_Union[_types_pb2.ApprovalState, str]] = ..., approved_by: _Optional[str] = ..., approved_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., resolved_config: _Optional[_Union[_profile_pb2.RunConfig, _Mapping]] = ..., diff_path: _Optional[str] = ..., log_path: _Optional[str] = ..., changed_files: _Optional[int] = ..., total_size_bytes: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RunSummary(_message.Message):
    __slots__ = ()
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    FILES_MODIFIED_FIELD_NUMBER: _ClassVar[int]
    FILES_CREATED_FIELD_NUMBER: _ClassVar[int]
    FILES_DELETED_FIELD_NUMBER: _ClassVar[int]
    TOKENS_USED_FIELD_NUMBER: _ClassVar[int]
    TURNS_USED_FIELD_NUMBER: _ClassVar[int]
    COST_ESTIMATE_FIELD_NUMBER: _ClassVar[int]
    description: str
    files_modified: _containers.RepeatedScalarFieldContainer[str]
    files_created: _containers.RepeatedScalarFieldContainer[str]
    files_deleted: _containers.RepeatedScalarFieldContainer[str]
    tokens_used: int
    turns_used: int
    cost_estimate: float
    def __init__(self, description: _Optional[str] = ..., files_modified: _Optional[_Iterable[str]] = ..., files_created: _Optional[_Iterable[str]] = ..., files_deleted: _Optional[_Iterable[str]] = ..., tokens_used: _Optional[int] = ..., turns_used: _Optional[int] = ..., cost_estimate: _Optional[float] = ...) -> None: ...

class RunCheckpoint(_message.Message):
    __slots__ = ()
    class MetadataEntry(_message.Message):
        __slots__ = ()
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
    __slots__ = ()
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
    __slots__ = ()
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
    __slots__ = ()
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
    __slots__ = ()
    SUPPORTS_STREAMING_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_MESSAGES_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_TOOL_EVENTS_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_COST_TRACKING_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_CANCELLATION_FIELD_NUMBER: _ClassVar[int]
    MAX_TURNS_FIELD_NUMBER: _ClassVar[int]
    supports_streaming: bool
    supports_messages: bool
    supports_tool_events: bool
    supports_cost_tracking: bool
    supports_cancellation: bool
    max_turns: int
    def __init__(self, supports_streaming: _Optional[bool] = ..., supports_messages: _Optional[bool] = ..., supports_tool_events: _Optional[bool] = ..., supports_cost_tracking: _Optional[bool] = ..., supports_cancellation: _Optional[bool] = ..., max_turns: _Optional[int] = ...) -> None: ...

class ProbeResult(_message.Message):
    __slots__ = ()
    class DetailsEntry(_message.Message):
        __slots__ = ()
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
    __slots__ = ()
    STOPPED_COUNT_FIELD_NUMBER: _ClassVar[int]
    FAILURES_FIELD_NUMBER: _ClassVar[int]
    stopped_count: int
    failures: _containers.RepeatedCompositeFieldContainer[StopFailure]
    def __init__(self, stopped_count: _Optional[int] = ..., failures: _Optional[_Iterable[_Union[StopFailure, _Mapping]]] = ...) -> None: ...

class StopFailure(_message.Message):
    __slots__ = ()
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    error: str
    def __init__(self, run_id: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...

class ApproveResult(_message.Message):
    __slots__ = ()
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    FILES_APPLIED_FIELD_NUMBER: _ClassVar[int]
    COMMIT_HASH_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    success: bool
    files_applied: int
    commit_hash: str
    message: str
    def __init__(self, success: _Optional[bool] = ..., files_applied: _Optional[int] = ..., commit_hash: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class RunDiff(_message.Message):
    __slots__ = ()
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
    __slots__ = ()
    PATH_FIELD_NUMBER: _ClassVar[int]
    CHANGE_TYPE_FIELD_NUMBER: _ClassVar[int]
    ADDITIONS_FIELD_NUMBER: _ClassVar[int]
    DELETIONS_FIELD_NUMBER: _ClassVar[int]
    IS_BINARY_FIELD_NUMBER: _ClassVar[int]
    PATCH_FIELD_NUMBER: _ClassVar[int]
    path: str
    change_type: str
    additions: int
    deletions: int
    is_binary: bool
    patch: str
    def __init__(self, path: _Optional[str] = ..., change_type: _Optional[str] = ..., additions: _Optional[int] = ..., deletions: _Optional[int] = ..., is_binary: _Optional[bool] = ..., patch: _Optional[str] = ...) -> None: ...
