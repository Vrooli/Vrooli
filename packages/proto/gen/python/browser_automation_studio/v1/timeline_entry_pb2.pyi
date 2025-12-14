import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from common.v1 import types_pb2 as _types_pb2
from browser_automation_studio.v1 import shared_pb2 as _shared_pb2
from browser_automation_studio.v1 import geometry_pb2 as _geometry_pb2
from browser_automation_studio.v1 import action_pb2 as _action_pb2
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
    TIMELINE_MESSAGE_TYPE_ENTRY: _ClassVar[TimelineMessageType]
    TIMELINE_MESSAGE_TYPE_STATUS: _ClassVar[TimelineMessageType]
    TIMELINE_MESSAGE_TYPE_HEARTBEAT: _ClassVar[TimelineMessageType]
    TIMELINE_MESSAGE_TYPE_LOG: _ClassVar[TimelineMessageType]
TIMELINE_MESSAGE_TYPE_UNSPECIFIED: TimelineMessageType
TIMELINE_MESSAGE_TYPE_ENTRY: TimelineMessageType
TIMELINE_MESSAGE_TYPE_STATUS: TimelineMessageType
TIMELINE_MESSAGE_TYPE_HEARTBEAT: TimelineMessageType
TIMELINE_MESSAGE_TYPE_LOG: TimelineMessageType

class TimelineEntry(_message.Message):
    __slots__ = ()
    ID_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_NUM_FIELD_NUMBER: _ClassVar[int]
    STEP_INDEX_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    TELEMETRY_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    AGGREGATES_FIELD_NUMBER: _ClassVar[int]
    TRACE_ID_FIELD_NUMBER: _ClassVar[int]
    CORRELATION_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    sequence_num: int
    step_index: int
    node_id: str
    timestamp: _timestamp_pb2.Timestamp
    duration_ms: int
    total_duration_ms: int
    action: _action_pb2.ActionDefinition
    telemetry: _telemetry_pb2.ActionTelemetry
    context: _shared_pb2.EventContext
    aggregates: TimelineEntryAggregates
    trace_id: str
    correlation_id: str
    def __init__(self, id: _Optional[str] = ..., sequence_num: _Optional[int] = ..., step_index: _Optional[int] = ..., node_id: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., duration_ms: _Optional[int] = ..., total_duration_ms: _Optional[int] = ..., action: _Optional[_Union[_action_pb2.ActionDefinition, _Mapping]] = ..., telemetry: _Optional[_Union[_telemetry_pb2.ActionTelemetry, _Mapping]] = ..., context: _Optional[_Union[_shared_pb2.EventContext, _Mapping]] = ..., aggregates: _Optional[_Union[TimelineEntryAggregates, _Mapping]] = ..., trace_id: _Optional[str] = ..., correlation_id: _Optional[str] = ...) -> None: ...

class TimelineEntryAggregates(_message.Message):
    __slots__ = ()
    STATUS_FIELD_NUMBER: _ClassVar[int]
    FINAL_URL_FIELD_NUMBER: _ClassVar[int]
    CONSOLE_LOG_COUNT_FIELD_NUMBER: _ClassVar[int]
    NETWORK_EVENT_COUNT_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    DOM_SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    DOM_SNAPSHOT_PREVIEW_FIELD_NUMBER: _ClassVar[int]
    EXTRACTED_DATA_PREVIEW_FIELD_NUMBER: _ClassVar[int]
    FOCUSED_ELEMENT_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    status: _shared_pb2.StepStatus
    final_url: str
    console_log_count: int
    network_event_count: int
    artifacts: _containers.RepeatedCompositeFieldContainer[TimelineArtifact]
    dom_snapshot: TimelineArtifact
    dom_snapshot_preview: str
    extracted_data_preview: _types_pb2.JsonValue
    focused_element: ElementFocus
    progress: int
    def __init__(self, status: _Optional[_Union[_shared_pb2.StepStatus, str]] = ..., final_url: _Optional[str] = ..., console_log_count: _Optional[int] = ..., network_event_count: _Optional[int] = ..., artifacts: _Optional[_Iterable[_Union[TimelineArtifact, _Mapping]]] = ..., dom_snapshot: _Optional[_Union[TimelineArtifact, _Mapping]] = ..., dom_snapshot_preview: _Optional[str] = ..., extracted_data_preview: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ..., focused_element: _Optional[_Union[ElementFocus, _Mapping]] = ..., progress: _Optional[int] = ...) -> None: ...

class TimelineArtifact(_message.Message):
    __slots__ = ()
    class PayloadEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    STORAGE_URL_FIELD_NUMBER: _ClassVar[int]
    THUMBNAIL_URL_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    STEP_INDEX_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    id: str
    type: _shared_pb2.ArtifactType
    label: str
    storage_url: str
    thumbnail_url: str
    content_type: str
    size_bytes: int
    step_index: int
    payload: _containers.MessageMap[str, _types_pb2.JsonValue]
    def __init__(self, id: _Optional[str] = ..., type: _Optional[_Union[_shared_pb2.ArtifactType, str]] = ..., label: _Optional[str] = ..., storage_url: _Optional[str] = ..., thumbnail_url: _Optional[str] = ..., content_type: _Optional[str] = ..., size_bytes: _Optional[int] = ..., step_index: _Optional[int] = ..., payload: _Optional[_Mapping[str, _types_pb2.JsonValue]] = ...) -> None: ...

class ElementFocus(_message.Message):
    __slots__ = ()
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    BOUNDING_BOX_FIELD_NUMBER: _ClassVar[int]
    selector: str
    bounding_box: _geometry_pb2.BoundingBox
    def __init__(self, selector: _Optional[str] = ..., bounding_box: _Optional[_Union[_geometry_pb2.BoundingBox, _Mapping]] = ...) -> None: ...

class TimelineLog(_message.Message):
    __slots__ = ()
    ID_FIELD_NUMBER: _ClassVar[int]
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    STEP_NAME_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    id: str
    level: _shared_pb2.LogLevel
    message: str
    step_name: str
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., level: _Optional[_Union[_shared_pb2.LogLevel, str]] = ..., message: _Optional[str] = ..., step_name: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class TimelineStreamMessage(_message.Message):
    __slots__ = ()
    TYPE_FIELD_NUMBER: _ClassVar[int]
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    HEARTBEAT_FIELD_NUMBER: _ClassVar[int]
    LOG_FIELD_NUMBER: _ClassVar[int]
    type: TimelineMessageType
    entry: TimelineEntry
    status: TimelineStatusUpdate
    heartbeat: TimelineHeartbeat
    log: TimelineLog
    def __init__(self, type: _Optional[_Union[TimelineMessageType, str]] = ..., entry: _Optional[_Union[TimelineEntry, _Mapping]] = ..., status: _Optional[_Union[TimelineStatusUpdate, _Mapping]] = ..., heartbeat: _Optional[_Union[TimelineHeartbeat, _Mapping]] = ..., log: _Optional[_Union[TimelineLog, _Mapping]] = ...) -> None: ...

class TimelineStatusUpdate(_message.Message):
    __slots__ = ()
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    ENTRY_COUNT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    id: str
    status: _shared_pb2.ExecutionStatus
    progress: int
    entry_count: int
    error: str
    def __init__(self, id: _Optional[str] = ..., status: _Optional[_Union[_shared_pb2.ExecutionStatus, str]] = ..., progress: _Optional[int] = ..., entry_count: _Optional[int] = ..., error: _Optional[str] = ...) -> None: ...

class TimelineHeartbeat(_message.Message):
    __slots__ = ()
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    timestamp: _timestamp_pb2.Timestamp
    session_id: str
    def __init__(self, timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., session_id: _Optional[str] = ...) -> None: ...
