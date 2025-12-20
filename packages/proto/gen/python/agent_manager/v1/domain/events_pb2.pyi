import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from agent_manager.v1.domain import types_pb2 as _types_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RunEvent(_message.Message):
    __slots__ = ()
    ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    EVENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    LOG_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    TOOL_CALL_FIELD_NUMBER: _ClassVar[int]
    TOOL_RESULT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    METRIC_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    COST_FIELD_NUMBER: _ClassVar[int]
    RATE_LIMIT_FIELD_NUMBER: _ClassVar[int]
    id: str
    run_id: str
    sequence: int
    event_type: _types_pb2.RunEventType
    timestamp: _timestamp_pb2.Timestamp
    log: LogEventData
    message: MessageEventData
    tool_call: ToolCallEventData
    tool_result: ToolResultEventData
    status: StatusEventData
    metric: MetricEventData
    artifact: ArtifactEventData
    error: ErrorEventData
    progress: ProgressEventData
    cost: CostEventData
    rate_limit: RateLimitEventData
    def __init__(self, id: _Optional[str] = ..., run_id: _Optional[str] = ..., sequence: _Optional[int] = ..., event_type: _Optional[_Union[_types_pb2.RunEventType, str]] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., log: _Optional[_Union[LogEventData, _Mapping]] = ..., message: _Optional[_Union[MessageEventData, _Mapping]] = ..., tool_call: _Optional[_Union[ToolCallEventData, _Mapping]] = ..., tool_result: _Optional[_Union[ToolResultEventData, _Mapping]] = ..., status: _Optional[_Union[StatusEventData, _Mapping]] = ..., metric: _Optional[_Union[MetricEventData, _Mapping]] = ..., artifact: _Optional[_Union[ArtifactEventData, _Mapping]] = ..., error: _Optional[_Union[ErrorEventData, _Mapping]] = ..., progress: _Optional[_Union[ProgressEventData, _Mapping]] = ..., cost: _Optional[_Union[CostEventData, _Mapping]] = ..., rate_limit: _Optional[_Union[RateLimitEventData, _Mapping]] = ...) -> None: ...

class LogEventData(_message.Message):
    __slots__ = ()
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    level: str
    message: str
    def __init__(self, level: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class MessageEventData(_message.Message):
    __slots__ = ()
    ROLE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    role: str
    content: str
    def __init__(self, role: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class ToolCallEventData(_message.Message):
    __slots__ = ()
    TOOL_NAME_FIELD_NUMBER: _ClassVar[int]
    INPUT_FIELD_NUMBER: _ClassVar[int]
    TOOL_CALL_ID_FIELD_NUMBER: _ClassVar[int]
    tool_name: str
    input: _struct_pb2.Struct
    tool_call_id: str
    def __init__(self, tool_name: _Optional[str] = ..., input: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., tool_call_id: _Optional[str] = ...) -> None: ...

class ToolResultEventData(_message.Message):
    __slots__ = ()
    TOOL_NAME_FIELD_NUMBER: _ClassVar[int]
    TOOL_CALL_ID_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    tool_name: str
    tool_call_id: str
    output: str
    error: str
    success: bool
    def __init__(self, tool_name: _Optional[str] = ..., tool_call_id: _Optional[str] = ..., output: _Optional[str] = ..., error: _Optional[str] = ..., success: _Optional[bool] = ...) -> None: ...

class StatusEventData(_message.Message):
    __slots__ = ()
    OLD_STATUS_FIELD_NUMBER: _ClassVar[int]
    NEW_STATUS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    old_status: str
    new_status: str
    reason: str
    def __init__(self, old_status: _Optional[str] = ..., new_status: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class MetricEventData(_message.Message):
    __slots__ = ()
    class TagsEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    name: str
    value: float
    unit: str
    tags: _containers.ScalarMap[str, str]
    def __init__(self, name: _Optional[str] = ..., value: _Optional[float] = ..., unit: _Optional[str] = ..., tags: _Optional[_Mapping[str, str]] = ...) -> None: ...

class ArtifactEventData(_message.Message):
    __slots__ = ()
    TYPE_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    MIME_TYPE_FIELD_NUMBER: _ClassVar[int]
    type: str
    path: str
    size: int
    mime_type: str
    def __init__(self, type: _Optional[str] = ..., path: _Optional[str] = ..., size: _Optional[int] = ..., mime_type: _Optional[str] = ...) -> None: ...

class ErrorEventData(_message.Message):
    __slots__ = ()
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    RETRYABLE_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_FIELD_NUMBER: _ClassVar[int]
    STACK_TRACE_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    retryable: bool
    recovery: _types_pb2.RecoveryAction
    stack_trace: str
    details: _struct_pb2.Struct
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., retryable: _Optional[bool] = ..., recovery: _Optional[_Union[_types_pb2.RecoveryAction, str]] = ..., stack_trace: _Optional[str] = ..., details: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class ProgressEventData(_message.Message):
    __slots__ = ()
    PHASE_FIELD_NUMBER: _ClassVar[int]
    PERCENT_COMPLETE_FIELD_NUMBER: _ClassVar[int]
    CURRENT_ACTION_FIELD_NUMBER: _ClassVar[int]
    TURNS_COMPLETED_FIELD_NUMBER: _ClassVar[int]
    TURNS_TOTAL_FIELD_NUMBER: _ClassVar[int]
    TOKENS_USED_FIELD_NUMBER: _ClassVar[int]
    ELAPSED_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_REMAINING_FIELD_NUMBER: _ClassVar[int]
    phase: _types_pb2.RunPhase
    percent_complete: int
    current_action: str
    turns_completed: int
    turns_total: int
    tokens_used: int
    elapsed_seconds: float
    estimated_remaining: float
    def __init__(self, phase: _Optional[_Union[_types_pb2.RunPhase, str]] = ..., percent_complete: _Optional[int] = ..., current_action: _Optional[str] = ..., turns_completed: _Optional[int] = ..., turns_total: _Optional[int] = ..., tokens_used: _Optional[int] = ..., elapsed_seconds: _Optional[float] = ..., estimated_remaining: _Optional[float] = ...) -> None: ...

class CostEventData(_message.Message):
    __slots__ = ()
    INPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    CACHE_CREATION_TOKENS_FIELD_NUMBER: _ClassVar[int]
    CACHE_READ_TOKENS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_COST_USD_FIELD_NUMBER: _ClassVar[int]
    SERVICE_TIER_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    WEB_SEARCH_REQUESTS_FIELD_NUMBER: _ClassVar[int]
    SERVER_TOOL_USE_REQUESTS_FIELD_NUMBER: _ClassVar[int]
    input_tokens: int
    output_tokens: int
    cache_creation_tokens: int
    cache_read_tokens: int
    total_cost_usd: float
    service_tier: str
    model: str
    web_search_requests: int
    server_tool_use_requests: int
    def __init__(self, input_tokens: _Optional[int] = ..., output_tokens: _Optional[int] = ..., cache_creation_tokens: _Optional[int] = ..., cache_read_tokens: _Optional[int] = ..., total_cost_usd: _Optional[float] = ..., service_tier: _Optional[str] = ..., model: _Optional[str] = ..., web_search_requests: _Optional[int] = ..., server_tool_use_requests: _Optional[int] = ...) -> None: ...

class RateLimitEventData(_message.Message):
    __slots__ = ()
    LIMIT_TYPE_FIELD_NUMBER: _ClassVar[int]
    RESET_TIME_FIELD_NUMBER: _ClassVar[int]
    RETRY_AFTER_FIELD_NUMBER: _ClassVar[int]
    CURRENT_USED_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    limit_type: str
    reset_time: _timestamp_pb2.Timestamp
    retry_after: int
    current_used: int
    limit: int
    message: str
    def __init__(self, limit_type: _Optional[str] = ..., reset_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., retry_after: _Optional[int] = ..., current_used: _Optional[int] = ..., limit: _Optional[int] = ..., message: _Optional[str] = ...) -> None: ...
