import datetime

from google.protobuf import any_pb2 as _any_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EventEnvelope(_message.Message):
    __slots__ = ("event_id", "event_type", "occurred_at", "source", "target", "correlation", "attribution", "data")
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    OCCURRED_AT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    CORRELATION_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    event_type: str
    occurred_at: _timestamp_pb2.Timestamp
    source: EventSource
    target: EventTarget
    correlation: EventCorrelation
    attribution: EventAttribution
    data: _any_pb2.Any
    def __init__(self, event_id: _Optional[str] = ..., event_type: _Optional[str] = ..., occurred_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., source: _Optional[_Union[EventSource, _Mapping]] = ..., target: _Optional[_Union[EventTarget, _Mapping]] = ..., correlation: _Optional[_Union[EventCorrelation, _Mapping]] = ..., attribution: _Optional[_Union[EventAttribution, _Mapping]] = ..., data: _Optional[_Union[_any_pb2.Any, _Mapping]] = ...) -> None: ...

class EventSource(_message.Message):
    __slots__ = ("scenario", "actor_kind")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ACTOR_KIND_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    actor_kind: str
    def __init__(self, scenario: _Optional[str] = ..., actor_kind: _Optional[str] = ...) -> None: ...

class EventTarget(_message.Message):
    __slots__ = ("scenario", "operation", "protocol")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    PROTOCOL_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    operation: str
    protocol: str
    def __init__(self, scenario: _Optional[str] = ..., operation: _Optional[str] = ..., protocol: _Optional[str] = ...) -> None: ...

class EventCorrelation(_message.Message):
    __slots__ = ("request_id", "agent_run_id", "task_id", "workflow_execution_id", "workflow_node_id", "attempt")
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_NODE_ID_FIELD_NUMBER: _ClassVar[int]
    ATTEMPT_FIELD_NUMBER: _ClassVar[int]
    request_id: str
    agent_run_id: str
    task_id: str
    workflow_execution_id: str
    workflow_node_id: str
    attempt: int
    def __init__(self, request_id: _Optional[str] = ..., agent_run_id: _Optional[str] = ..., task_id: _Optional[str] = ..., workflow_execution_id: _Optional[str] = ..., workflow_node_id: _Optional[str] = ..., attempt: _Optional[int] = ...) -> None: ...

class EventAttribution(_message.Message):
    __slots__ = ("subject_kind", "subject_id", "verified")
    SUBJECT_KIND_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_ID_FIELD_NUMBER: _ClassVar[int]
    VERIFIED_FIELD_NUMBER: _ClassVar[int]
    subject_kind: str
    subject_id: str
    verified: bool
    def __init__(self, subject_kind: _Optional[str] = ..., subject_id: _Optional[str] = ..., verified: _Optional[bool] = ...) -> None: ...

class ReceiptData(_message.Message):
    __slots__ = ("outcome", "status_code", "duration_ms", "policy_version", "idempotency_key", "projection")
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    STATUS_CODE_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    outcome: str
    status_code: int
    duration_ms: int
    policy_version: str
    idempotency_key: str
    projection: _struct_pb2.Struct
    def __init__(self, outcome: _Optional[str] = ..., status_code: _Optional[int] = ..., duration_ms: _Optional[int] = ..., policy_version: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., projection: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...
