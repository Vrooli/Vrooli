from vrooli_events.v1.domain import envelope_pb2 as _envelope_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ReceiptCapturePolicy(_message.Message):
    __slots__ = ("policy_id", "enabled", "selector", "response_type", "response_projection_paths", "retention_days", "access", "version")
    POLICY_ID_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_TYPE_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_PROJECTION_PATHS_FIELD_NUMBER: _ClassVar[int]
    RETENTION_DAYS_FIELD_NUMBER: _ClassVar[int]
    ACCESS_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    policy_id: str
    enabled: bool
    selector: ReceiptOperationSelector
    response_type: str
    response_projection_paths: _containers.RepeatedScalarFieldContainer[str]
    retention_days: int
    access: ReceiptAccessPolicy
    version: str
    def __init__(self, policy_id: _Optional[str] = ..., enabled: _Optional[bool] = ..., selector: _Optional[_Union[ReceiptOperationSelector, _Mapping]] = ..., response_type: _Optional[str] = ..., response_projection_paths: _Optional[_Iterable[str]] = ..., retention_days: _Optional[int] = ..., access: _Optional[_Union[ReceiptAccessPolicy, _Mapping]] = ..., version: _Optional[str] = ...) -> None: ...

class ReceiptOperationSelector(_message.Message):
    __slots__ = ("target_scenario", "operation", "protocol", "event_type")
    TARGET_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    PROTOCOL_FIELD_NUMBER: _ClassVar[int]
    EVENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    target_scenario: str
    operation: str
    protocol: str
    event_type: str
    def __init__(self, target_scenario: _Optional[str] = ..., operation: _Optional[str] = ..., protocol: _Optional[str] = ..., event_type: _Optional[str] = ...) -> None: ...

class ReceiptAccessPolicy(_message.Message):
    __slots__ = ("read_principals",)
    READ_PRINCIPALS_FIELD_NUMBER: _ClassVar[int]
    read_principals: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, read_principals: _Optional[_Iterable[str]] = ...) -> None: ...

class ReceiptQueryFilter(_message.Message):
    __slots__ = ("event_type", "target_scenario", "operation", "agent_run_id", "task_id", "workflow_execution_id", "workflow_node_id", "attempt", "verified_only", "page_token", "page_size")
    EVENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    TARGET_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    AGENT_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_NODE_ID_FIELD_NUMBER: _ClassVar[int]
    ATTEMPT_FIELD_NUMBER: _ClassVar[int]
    VERIFIED_ONLY_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    event_type: str
    target_scenario: str
    operation: str
    agent_run_id: str
    task_id: str
    workflow_execution_id: str
    workflow_node_id: str
    attempt: int
    verified_only: bool
    page_token: str
    page_size: int
    def __init__(self, event_type: _Optional[str] = ..., target_scenario: _Optional[str] = ..., operation: _Optional[str] = ..., agent_run_id: _Optional[str] = ..., task_id: _Optional[str] = ..., workflow_execution_id: _Optional[str] = ..., workflow_node_id: _Optional[str] = ..., attempt: _Optional[int] = ..., verified_only: _Optional[bool] = ..., page_token: _Optional[str] = ..., page_size: _Optional[int] = ...) -> None: ...

class ReceiptQueryResult(_message.Message):
    __slots__ = ("events", "next_page_token")
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    events: _containers.RepeatedCompositeFieldContainer[_envelope_pb2.EventEnvelope]
    next_page_token: str
    def __init__(self, events: _Optional[_Iterable[_Union[_envelope_pb2.EventEnvelope, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...
