import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from common.v1 import types_pb2 as _types_pb2
from browser_automation_studio.v1 import shared_pb2 as _shared_pb2
from browser_automation_studio.v1 import action_pb2 as _action_pb2
from browser_automation_studio.v1 import selectors_pb2 as _selectors_pb2
from browser_automation_studio.v1 import telemetry_pb2 as _telemetry_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TimelineMessageType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TIMELINE_MESSAGE_TYPE_UNSPECIFIED: _ClassVar[TimelineMessageType]
    TIMELINE_MESSAGE_TYPE_EVENT: _ClassVar[TimelineMessageType]
    TIMELINE_MESSAGE_TYPE_STATUS: _ClassVar[TimelineMessageType]
    TIMELINE_MESSAGE_TYPE_HEARTBEAT: _ClassVar[TimelineMessageType]
TIMELINE_MESSAGE_TYPE_UNSPECIFIED: TimelineMessageType
TIMELINE_MESSAGE_TYPE_EVENT: TimelineMessageType
TIMELINE_MESSAGE_TYPE_STATUS: TimelineMessageType
TIMELINE_MESSAGE_TYPE_HEARTBEAT: TimelineMessageType

class TimelineEvent(_message.Message):
    __slots__ = ()
    ID_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_NUM_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    STEP_INDEX_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    TELEMETRY_FIELD_NUMBER: _ClassVar[int]
    RECORDING_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    TRACE_ID_FIELD_NUMBER: _ClassVar[int]
    CORRELATION_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    sequence_num: int
    timestamp: _timestamp_pb2.Timestamp
    duration_ms: int
    step_index: int
    node_id: str
    action: _action_pb2.ActionDefinition
    telemetry: _telemetry_pb2.ActionTelemetry
    recording: RecordingContext
    execution: ExecutionContext
    trace_id: str
    correlation_id: str
    def __init__(self, id: _Optional[str] = ..., sequence_num: _Optional[int] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., duration_ms: _Optional[int] = ..., step_index: _Optional[int] = ..., node_id: _Optional[str] = ..., action: _Optional[_Union[_action_pb2.ActionDefinition, _Mapping]] = ..., telemetry: _Optional[_Union[_telemetry_pb2.ActionTelemetry, _Mapping]] = ..., recording: _Optional[_Union[RecordingContext, _Mapping]] = ..., execution: _Optional[_Union[ExecutionContext, _Mapping]] = ..., trace_id: _Optional[str] = ..., correlation_id: _Optional[str] = ...) -> None: ...

class RecordingContext(_message.Message):
    __slots__ = ()
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    NEEDS_CONFIRMATION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    selector_candidates: _containers.RepeatedCompositeFieldContainer[_selectors_pb2.SelectorCandidate]
    needs_confirmation: bool
    source: _shared_pb2.RecordingSource
    def __init__(self, session_id: _Optional[str] = ..., selector_candidates: _Optional[_Iterable[_Union[_selectors_pb2.SelectorCandidate, _Mapping]]] = ..., needs_confirmation: _Optional[bool] = ..., source: _Optional[_Union[_shared_pb2.RecordingSource, str]] = ...) -> None: ...

class ExecutionContext(_message.Message):
    __slots__ = ()
    class ExtractedDataEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    ERROR_CODE_FIELD_NUMBER: _ClassVar[int]
    RETRY_STATUS_FIELD_NUMBER: _ClassVar[int]
    ASSERTION_FIELD_NUMBER: _ClassVar[int]
    EXTRACTED_DATA_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    success: bool
    error: str
    error_code: str
    retry_status: _shared_pb2.RetryStatus
    assertion: _shared_pb2.AssertionResult
    extracted_data: _containers.MessageMap[str, _types_pb2.JsonValue]
    def __init__(self, execution_id: _Optional[str] = ..., success: _Optional[bool] = ..., error: _Optional[str] = ..., error_code: _Optional[str] = ..., retry_status: _Optional[_Union[_shared_pb2.RetryStatus, _Mapping]] = ..., assertion: _Optional[_Union[_shared_pb2.AssertionResult, _Mapping]] = ..., extracted_data: _Optional[_Mapping[str, _types_pb2.JsonValue]] = ...) -> None: ...

class TimelineStreamMessage(_message.Message):
    __slots__ = ()
    TYPE_FIELD_NUMBER: _ClassVar[int]
    EVENT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    HEARTBEAT_FIELD_NUMBER: _ClassVar[int]
    type: TimelineMessageType
    event: TimelineEvent
    status: TimelineStatusUpdate
    heartbeat: TimelineHeartbeat
    def __init__(self, type: _Optional[_Union[TimelineMessageType, str]] = ..., event: _Optional[_Union[TimelineEvent, _Mapping]] = ..., status: _Optional[_Union[TimelineStatusUpdate, _Mapping]] = ..., heartbeat: _Optional[_Union[TimelineHeartbeat, _Mapping]] = ...) -> None: ...

class TimelineStatusUpdate(_message.Message):
    __slots__ = ()
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    EVENT_COUNT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    id: str
    status: _shared_pb2.ExecutionStatus
    progress: int
    event_count: int
    error: str
    def __init__(self, id: _Optional[str] = ..., status: _Optional[_Union[_shared_pb2.ExecutionStatus, str]] = ..., progress: _Optional[int] = ..., event_count: _Optional[int] = ..., error: _Optional[str] = ...) -> None: ...

class TimelineHeartbeat(_message.Message):
    __slots__ = ()
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    timestamp: _timestamp_pb2.Timestamp
    session_id: str
    def __init__(self, timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., session_id: _Optional[str] = ...) -> None: ...
