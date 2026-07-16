import datetime

from agent_manager.v1.domain import run_pb2 as _run_pb2
from agent_manager.v1.domain import types_pb2 as _types_pb2
from agent_manager.v1.domain import workflow_pb2 as _workflow_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AgentManagerWsMessageType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    AGENT_MANAGER_WS_MESSAGE_TYPE_UNSPECIFIED: _ClassVar[AgentManagerWsMessageType]
    AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_EVENT: _ClassVar[AgentManagerWsMessageType]
    AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_STATUS: _ClassVar[AgentManagerWsMessageType]
    AGENT_MANAGER_WS_MESSAGE_TYPE_TASK_STATUS: _ClassVar[AgentManagerWsMessageType]
    AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_PROGRESS: _ClassVar[AgentManagerWsMessageType]
    AGENT_MANAGER_WS_MESSAGE_TYPE_CONNECTED: _ClassVar[AgentManagerWsMessageType]
    AGENT_MANAGER_WS_MESSAGE_TYPE_PONG: _ClassVar[AgentManagerWsMessageType]
    AGENT_MANAGER_WS_MESSAGE_TYPE_WORKFLOW_LIFECYCLE: _ClassVar[AgentManagerWsMessageType]

class AgentManagerWsClientMessageType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_UNSPECIFIED: _ClassVar[AgentManagerWsClientMessageType]
    AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_SUBSCRIBE: _ClassVar[AgentManagerWsClientMessageType]
    AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_UNSUBSCRIBE: _ClassVar[AgentManagerWsClientMessageType]
    AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_SUBSCRIBE_ALL: _ClassVar[AgentManagerWsClientMessageType]
    AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_UNSUBSCRIBE_ALL: _ClassVar[AgentManagerWsClientMessageType]
    AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_PING: _ClassVar[AgentManagerWsClientMessageType]
AGENT_MANAGER_WS_MESSAGE_TYPE_UNSPECIFIED: AgentManagerWsMessageType
AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_EVENT: AgentManagerWsMessageType
AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_STATUS: AgentManagerWsMessageType
AGENT_MANAGER_WS_MESSAGE_TYPE_TASK_STATUS: AgentManagerWsMessageType
AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_PROGRESS: AgentManagerWsMessageType
AGENT_MANAGER_WS_MESSAGE_TYPE_CONNECTED: AgentManagerWsMessageType
AGENT_MANAGER_WS_MESSAGE_TYPE_PONG: AgentManagerWsMessageType
AGENT_MANAGER_WS_MESSAGE_TYPE_WORKFLOW_LIFECYCLE: AgentManagerWsMessageType
AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_UNSPECIFIED: AgentManagerWsClientMessageType
AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_SUBSCRIBE: AgentManagerWsClientMessageType
AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_UNSUBSCRIBE: AgentManagerWsClientMessageType
AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_SUBSCRIBE_ALL: AgentManagerWsClientMessageType
AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_UNSUBSCRIBE_ALL: AgentManagerWsClientMessageType
AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_PING: AgentManagerWsClientMessageType

class RunEvent(_message.Message):
    __slots__ = ("id", "run_id", "sequence", "event_type", "timestamp", "log", "message", "message_deleted", "tool_call", "tool_result", "status", "metric", "artifact", "error", "progress", "cost", "rate_limit", "compaction")
    ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    EVENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    LOG_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_DELETED_FIELD_NUMBER: _ClassVar[int]
    TOOL_CALL_FIELD_NUMBER: _ClassVar[int]
    TOOL_RESULT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    METRIC_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    COST_FIELD_NUMBER: _ClassVar[int]
    RATE_LIMIT_FIELD_NUMBER: _ClassVar[int]
    COMPACTION_FIELD_NUMBER: _ClassVar[int]
    id: str
    run_id: str
    sequence: int
    event_type: _types_pb2.RunEventType
    timestamp: _timestamp_pb2.Timestamp
    log: LogEventData
    message: MessageEventData
    message_deleted: MessageDeletedEventData
    tool_call: ToolCallEventData
    tool_result: ToolResultEventData
    status: StatusEventData
    metric: MetricEventData
    artifact: ArtifactEventData
    error: ErrorEventData
    progress: ProgressEventData
    cost: CostEventData
    rate_limit: RateLimitEventData
    compaction: CompactionEventData
    def __init__(self, id: _Optional[str] = ..., run_id: _Optional[str] = ..., sequence: _Optional[int] = ..., event_type: _Optional[_Union[_types_pb2.RunEventType, str]] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., log: _Optional[_Union[LogEventData, _Mapping]] = ..., message: _Optional[_Union[MessageEventData, _Mapping]] = ..., message_deleted: _Optional[_Union[MessageDeletedEventData, _Mapping]] = ..., tool_call: _Optional[_Union[ToolCallEventData, _Mapping]] = ..., tool_result: _Optional[_Union[ToolResultEventData, _Mapping]] = ..., status: _Optional[_Union[StatusEventData, _Mapping]] = ..., metric: _Optional[_Union[MetricEventData, _Mapping]] = ..., artifact: _Optional[_Union[ArtifactEventData, _Mapping]] = ..., error: _Optional[_Union[ErrorEventData, _Mapping]] = ..., progress: _Optional[_Union[ProgressEventData, _Mapping]] = ..., cost: _Optional[_Union[CostEventData, _Mapping]] = ..., rate_limit: _Optional[_Union[RateLimitEventData, _Mapping]] = ..., compaction: _Optional[_Union[CompactionEventData, _Mapping]] = ...) -> None: ...

class AgentManagerWsMessage(_message.Message):
    __slots__ = ("type", "run_id", "run_event", "run_status", "task_status", "run_progress", "connected", "pong", "workflow_lifecycle")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_EVENT_FIELD_NUMBER: _ClassVar[int]
    RUN_STATUS_FIELD_NUMBER: _ClassVar[int]
    TASK_STATUS_FIELD_NUMBER: _ClassVar[int]
    RUN_PROGRESS_FIELD_NUMBER: _ClassVar[int]
    CONNECTED_FIELD_NUMBER: _ClassVar[int]
    PONG_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_LIFECYCLE_FIELD_NUMBER: _ClassVar[int]
    type: AgentManagerWsMessageType
    run_id: str
    run_event: RunEvent
    run_status: RunStatusUpdate
    task_status: TaskStatusUpdate
    run_progress: ProgressEventData
    connected: WsConnected
    pong: WsPong
    workflow_lifecycle: WorkflowLifecycleUpdate
    def __init__(self, type: _Optional[_Union[AgentManagerWsMessageType, str]] = ..., run_id: _Optional[str] = ..., run_event: _Optional[_Union[RunEvent, _Mapping]] = ..., run_status: _Optional[_Union[RunStatusUpdate, _Mapping]] = ..., task_status: _Optional[_Union[TaskStatusUpdate, _Mapping]] = ..., run_progress: _Optional[_Union[ProgressEventData, _Mapping]] = ..., connected: _Optional[_Union[WsConnected, _Mapping]] = ..., pong: _Optional[_Union[WsPong, _Mapping]] = ..., workflow_lifecycle: _Optional[_Union[WorkflowLifecycleUpdate, _Mapping]] = ...) -> None: ...

class WorkflowLifecycleUpdate(_message.Message):
    __slots__ = ("execution_id", "definition_digest", "status", "node_id", "strategy", "profile_identity", "run_id", "conversation_id", "source_attempt_id", "journal_sequence", "journal_kind", "journal_payload_digest", "budget_usage", "terminal_reason")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_DIGEST_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    PROFILE_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    CONVERSATION_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ATTEMPT_ID_FIELD_NUMBER: _ClassVar[int]
    JOURNAL_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    JOURNAL_KIND_FIELD_NUMBER: _ClassVar[int]
    JOURNAL_PAYLOAD_DIGEST_FIELD_NUMBER: _ClassVar[int]
    BUDGET_USAGE_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_REASON_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    definition_digest: str
    status: str
    node_id: str
    strategy: str
    profile_identity: str
    run_id: str
    conversation_id: str
    source_attempt_id: str
    journal_sequence: int
    journal_kind: str
    journal_payload_digest: str
    budget_usage: _workflow_pb2.WorkflowBudgetUsage
    terminal_reason: _workflow_pb2.WorkflowTerminalReason
    def __init__(self, execution_id: _Optional[str] = ..., definition_digest: _Optional[str] = ..., status: _Optional[str] = ..., node_id: _Optional[str] = ..., strategy: _Optional[str] = ..., profile_identity: _Optional[str] = ..., run_id: _Optional[str] = ..., conversation_id: _Optional[str] = ..., source_attempt_id: _Optional[str] = ..., journal_sequence: _Optional[int] = ..., journal_kind: _Optional[str] = ..., journal_payload_digest: _Optional[str] = ..., budget_usage: _Optional[_Union[_workflow_pb2.WorkflowBudgetUsage, _Mapping]] = ..., terminal_reason: _Optional[_Union[_workflow_pb2.WorkflowTerminalReason, _Mapping]] = ...) -> None: ...

class RunStatusUpdate(_message.Message):
    __slots__ = ("run_id", "status", "task_id", "prompt_preview", "result_selection_status", "result_selection_rule")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    PROMPT_PREVIEW_FIELD_NUMBER: _ClassVar[int]
    RESULT_SELECTION_STATUS_FIELD_NUMBER: _ClassVar[int]
    RESULT_SELECTION_RULE_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    status: _types_pb2.RunStatus
    task_id: str
    prompt_preview: str
    result_selection_status: _run_pb2.FinalOutputSelectionStatus
    result_selection_rule: str
    def __init__(self, run_id: _Optional[str] = ..., status: _Optional[_Union[_types_pb2.RunStatus, str]] = ..., task_id: _Optional[str] = ..., prompt_preview: _Optional[str] = ..., result_selection_status: _Optional[_Union[_run_pb2.FinalOutputSelectionStatus, str]] = ..., result_selection_rule: _Optional[str] = ...) -> None: ...

class TaskStatusUpdate(_message.Message):
    __slots__ = ("task_id", "status")
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    status: _types_pb2.TaskStatus
    def __init__(self, task_id: _Optional[str] = ..., status: _Optional[_Union[_types_pb2.TaskStatus, str]] = ...) -> None: ...

class WsConnected(_message.Message):
    __slots__ = ("message", "timestamp")
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    message: str
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, message: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class WsPong(_message.Message):
    __slots__ = ("timestamp",)
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class AgentManagerWsClientMessage(_message.Message):
    __slots__ = ("type", "run_subscription")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    RUN_SUBSCRIPTION_FIELD_NUMBER: _ClassVar[int]
    type: AgentManagerWsClientMessageType
    run_subscription: RunSubscription
    def __init__(self, type: _Optional[_Union[AgentManagerWsClientMessageType, str]] = ..., run_subscription: _Optional[_Union[RunSubscription, _Mapping]] = ...) -> None: ...

class RunSubscription(_message.Message):
    __slots__ = ("run_id",)
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    def __init__(self, run_id: _Optional[str] = ...) -> None: ...

class LogEventData(_message.Message):
    __slots__ = ("level", "message")
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    level: str
    message: str
    def __init__(self, level: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class MessageEventData(_message.Message):
    __slots__ = ("role", "content", "attachments", "message_id", "conversation_id", "turn_id", "provider_origin", "completion_reason", "terminal", "parent_message_id", "provider_event_type", "raw_evidence_ref", "evidence_only", "evidence_for_event_id")
    ROLE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENTS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    CONVERSATION_ID_FIELD_NUMBER: _ClassVar[int]
    TURN_ID_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ORIGIN_FIELD_NUMBER: _ClassVar[int]
    COMPLETION_REASON_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_FIELD_NUMBER: _ClassVar[int]
    PARENT_MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_EVENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    RAW_EVIDENCE_REF_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_ONLY_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FOR_EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    role: str
    content: str
    attachments: _containers.RepeatedCompositeFieldContainer[MessageAttachmentInfo]
    message_id: str
    conversation_id: str
    turn_id: str
    provider_origin: str
    completion_reason: str
    terminal: bool
    parent_message_id: str
    provider_event_type: str
    raw_evidence_ref: str
    evidence_only: bool
    evidence_for_event_id: str
    def __init__(self, role: _Optional[str] = ..., content: _Optional[str] = ..., attachments: _Optional[_Iterable[_Union[MessageAttachmentInfo, _Mapping]]] = ..., message_id: _Optional[str] = ..., conversation_id: _Optional[str] = ..., turn_id: _Optional[str] = ..., provider_origin: _Optional[str] = ..., completion_reason: _Optional[str] = ..., terminal: _Optional[bool] = ..., parent_message_id: _Optional[str] = ..., provider_event_type: _Optional[str] = ..., raw_evidence_ref: _Optional[str] = ..., evidence_only: _Optional[bool] = ..., evidence_for_event_id: _Optional[str] = ...) -> None: ...

class MessageAttachmentInfo(_message.Message):
    __slots__ = ("id", "file_name", "content_type", "url")
    ID_FIELD_NUMBER: _ClassVar[int]
    FILE_NAME_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    id: str
    file_name: str
    content_type: str
    url: str
    def __init__(self, id: _Optional[str] = ..., file_name: _Optional[str] = ..., content_type: _Optional[str] = ..., url: _Optional[str] = ...) -> None: ...

class MessageDeletedEventData(_message.Message):
    __slots__ = ("target_event_id",)
    TARGET_EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    target_event_id: str
    def __init__(self, target_event_id: _Optional[str] = ...) -> None: ...

class ToolCallEventData(_message.Message):
    __slots__ = ("tool_name", "input", "tool_call_id")
    TOOL_NAME_FIELD_NUMBER: _ClassVar[int]
    INPUT_FIELD_NUMBER: _ClassVar[int]
    TOOL_CALL_ID_FIELD_NUMBER: _ClassVar[int]
    tool_name: str
    input: _struct_pb2.Struct
    tool_call_id: str
    def __init__(self, tool_name: _Optional[str] = ..., input: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., tool_call_id: _Optional[str] = ...) -> None: ...

class ToolResultEventData(_message.Message):
    __slots__ = ("tool_name", "tool_call_id", "output", "error", "success")
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
    __slots__ = ("old_status", "new_status", "reason")
    OLD_STATUS_FIELD_NUMBER: _ClassVar[int]
    NEW_STATUS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    old_status: str
    new_status: str
    reason: str
    def __init__(self, old_status: _Optional[str] = ..., new_status: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class MetricEventData(_message.Message):
    __slots__ = ("name", "value", "unit", "tags")
    class TagsEntry(_message.Message):
        __slots__ = ("key", "value")
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
    __slots__ = ("type", "path", "size", "mime_type")
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
    __slots__ = ("code", "message", "retryable", "recovery", "stack_trace", "details")
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
    __slots__ = ("phase", "percent_complete", "current_action", "turns_completed", "turns_total", "tokens_used", "elapsed_seconds", "estimated_remaining")
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
    __slots__ = ("input_tokens", "output_tokens", "cache_creation_tokens", "cache_read_tokens", "total_cost_usd", "service_tier", "model", "web_search_requests", "server_tool_use_requests")
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
    __slots__ = ("limit_type", "reset_time", "retry_after", "current_used", "limit", "message")
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

class CompactionEventData(_message.Message):
    __slots__ = ("summary", "trigger", "focus", "messages_compacted", "tokens_before", "tokens_after", "original_command")
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_FIELD_NUMBER: _ClassVar[int]
    FOCUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGES_COMPACTED_FIELD_NUMBER: _ClassVar[int]
    TOKENS_BEFORE_FIELD_NUMBER: _ClassVar[int]
    TOKENS_AFTER_FIELD_NUMBER: _ClassVar[int]
    ORIGINAL_COMMAND_FIELD_NUMBER: _ClassVar[int]
    summary: str
    trigger: str
    focus: str
    messages_compacted: int
    tokens_before: int
    tokens_after: int
    original_command: str
    def __init__(self, summary: _Optional[str] = ..., trigger: _Optional[str] = ..., focus: _Optional[str] = ..., messages_compacted: _Optional[int] = ..., tokens_before: _Optional[int] = ..., tokens_after: _Optional[int] = ..., original_command: _Optional[str] = ...) -> None: ...
