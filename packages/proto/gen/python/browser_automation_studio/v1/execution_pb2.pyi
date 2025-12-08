import datetime

from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from browser_automation_studio.v1 import shared_pb2 as _shared_pb2
from browser_automation_studio.v1 import timeline_pb2 as _timeline_pb2
from browser_automation_studio.v1 import workflow_pb2 as _workflow_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Execution(_message.Message):
    __slots__ = ()
    class TriggerMetadataEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _struct_pb2.Value
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...
    class ParametersEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _struct_pb2.Value
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...
    class ParametersTypedEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _shared_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_shared_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    class ResultEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _struct_pb2.Value
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...
    class ResultTypedEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _shared_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_shared_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_VERSION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_TYPE_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_METADATA_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_TYPED_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_HEARTBEAT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    RESULT_TYPED_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    CURRENT_STEP_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    workflow_id: str
    workflow_version: int
    status: _shared_pb2.ExecutionStatus
    trigger_type: _shared_pb2.TriggerType
    trigger_metadata: _containers.MessageMap[str, _struct_pb2.Value]
    parameters: _containers.MessageMap[str, _struct_pb2.Value]
    parameters_typed: _containers.MessageMap[str, _shared_pb2.JsonValue]
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    last_heartbeat: _timestamp_pb2.Timestamp
    error: str
    result: _containers.MessageMap[str, _struct_pb2.Value]
    result_typed: _containers.MessageMap[str, _shared_pb2.JsonValue]
    progress: int
    current_step: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, execution_id: _Optional[str] = ..., workflow_id: _Optional[str] = ..., workflow_version: _Optional[int] = ..., status: _Optional[_Union[_shared_pb2.ExecutionStatus, str]] = ..., trigger_type: _Optional[_Union[_shared_pb2.TriggerType, str]] = ..., trigger_metadata: _Optional[_Mapping[str, _struct_pb2.Value]] = ..., parameters: _Optional[_Mapping[str, _struct_pb2.Value]] = ..., parameters_typed: _Optional[_Mapping[str, _shared_pb2.JsonValue]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_heartbeat: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., error: _Optional[str] = ..., result: _Optional[_Mapping[str, _struct_pb2.Value]] = ..., result_typed: _Optional[_Mapping[str, _shared_pb2.JsonValue]] = ..., progress: _Optional[int] = ..., current_step: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ExecuteAdhocRequest(_message.Message):
    __slots__ = ()
    class ParametersEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _struct_pb2.Value
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...
    class ParametersTypedEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _shared_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_shared_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    FLOW_DEFINITION_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_TYPED_FIELD_NUMBER: _ClassVar[int]
    WAIT_FOR_COMPLETION_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    flow_definition: _workflow_pb2.WorkflowDefinition
    parameters: _containers.MessageMap[str, _struct_pb2.Value]
    parameters_typed: _containers.MessageMap[str, _shared_pb2.JsonValue]
    wait_for_completion: bool
    metadata: ExecutionMetadata
    def __init__(self, flow_definition: _Optional[_Union[_workflow_pb2.WorkflowDefinition, _Mapping]] = ..., parameters: _Optional[_Mapping[str, _struct_pb2.Value]] = ..., parameters_typed: _Optional[_Mapping[str, _shared_pb2.JsonValue]] = ..., wait_for_completion: _Optional[bool] = ..., metadata: _Optional[_Union[ExecutionMetadata, _Mapping]] = ...) -> None: ...

class ExecutionMetadata(_message.Message):
    __slots__ = ()
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class ExecuteAdhocResponse(_message.Message):
    __slots__ = ()
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    status: _shared_pb2.ExecutionStatus
    workflow_id: str
    message: str
    completed_at: _timestamp_pb2.Timestamp
    error: str
    def __init__(self, execution_id: _Optional[str] = ..., status: _Optional[_Union[_shared_pb2.ExecutionStatus, str]] = ..., workflow_id: _Optional[str] = ..., message: _Optional[str] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., error: _Optional[str] = ...) -> None: ...

class Screenshot(_message.Message):
    __slots__ = ()
    ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    STEP_NAME_FIELD_NUMBER: _ClassVar[int]
    STEP_INDEX_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    STORAGE_URL_FIELD_NUMBER: _ClassVar[int]
    THUMBNAIL_URL_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    id: str
    execution_id: str
    step_name: str
    step_index: int
    timestamp: _timestamp_pb2.Timestamp
    storage_url: str
    thumbnail_url: str
    width: int
    height: int
    size_bytes: int
    def __init__(self, id: _Optional[str] = ..., execution_id: _Optional[str] = ..., step_name: _Optional[str] = ..., step_index: _Optional[int] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., storage_url: _Optional[str] = ..., thumbnail_url: _Optional[str] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., size_bytes: _Optional[int] = ...) -> None: ...

class GetScreenshotsResponse(_message.Message):
    __slots__ = ()
    SCREENSHOTS_FIELD_NUMBER: _ClassVar[int]
    screenshots: _containers.RepeatedCompositeFieldContainer[Screenshot]
    def __init__(self, screenshots: _Optional[_Iterable[_Union[Screenshot, _Mapping]]] = ...) -> None: ...

class ExecutionEventEnvelope(_message.Message):
    __slots__ = ()
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_VERSION_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    STEP_INDEX_FIELD_NUMBER: _ClassVar[int]
    ATTEMPT_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    STATUS_UPDATE_FIELD_NUMBER: _ClassVar[int]
    TIMELINE_FRAME_FIELD_NUMBER: _ClassVar[int]
    LOG_FIELD_NUMBER: _ClassVar[int]
    HEARTBEAT_FIELD_NUMBER: _ClassVar[int]
    TELEMETRY_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    payload_version: str
    kind: _shared_pb2.EventKind
    execution_id: str
    workflow_id: str
    step_index: int
    attempt: int
    sequence: int
    timestamp: _timestamp_pb2.Timestamp
    status_update: StatusUpdateEvent
    timeline_frame: TimelineFrameEvent
    log: LogEvent
    heartbeat: HeartbeatEvent
    telemetry: TelemetryEvent
    def __init__(self, schema_version: _Optional[str] = ..., payload_version: _Optional[str] = ..., kind: _Optional[_Union[_shared_pb2.EventKind, str]] = ..., execution_id: _Optional[str] = ..., workflow_id: _Optional[str] = ..., step_index: _Optional[int] = ..., attempt: _Optional[int] = ..., sequence: _Optional[int] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., status_update: _Optional[_Union[StatusUpdateEvent, _Mapping]] = ..., timeline_frame: _Optional[_Union[TimelineFrameEvent, _Mapping]] = ..., log: _Optional[_Union[LogEvent, _Mapping]] = ..., heartbeat: _Optional[_Union[HeartbeatEvent, _Mapping]] = ..., telemetry: _Optional[_Union[TelemetryEvent, _Mapping]] = ...) -> None: ...

class ExecutionExportPreview(_message.Message):
    __slots__ = ()
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    SPEC_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_FRAME_COUNT_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_ASSET_COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_TYPED_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    spec_id: str
    status: _shared_pb2.ExportStatus
    message: str
    captured_frame_count: int
    available_asset_count: int
    total_duration_ms: int
    package: _struct_pb2.Struct
    package_typed: _shared_pb2.JsonObject
    def __init__(self, execution_id: _Optional[str] = ..., spec_id: _Optional[str] = ..., status: _Optional[_Union[_shared_pb2.ExportStatus, str]] = ..., message: _Optional[str] = ..., captured_frame_count: _Optional[int] = ..., available_asset_count: _Optional[int] = ..., total_duration_ms: _Optional[int] = ..., package: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., package_typed: _Optional[_Union[_shared_pb2.JsonObject, _Mapping]] = ...) -> None: ...

class StatusUpdateEvent(_message.Message):
    __slots__ = ()
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    CURRENT_STEP_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    OCCURRED_AT_FIELD_NUMBER: _ClassVar[int]
    status: _shared_pb2.ExecutionStatus
    progress: int
    current_step: str
    error: str
    occurred_at: _timestamp_pb2.Timestamp
    def __init__(self, status: _Optional[_Union[_shared_pb2.ExecutionStatus, str]] = ..., progress: _Optional[int] = ..., current_step: _Optional[str] = ..., error: _Optional[str] = ..., occurred_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class TimelineFrameEvent(_message.Message):
    __slots__ = ()
    FRAME_FIELD_NUMBER: _ClassVar[int]
    frame: _timeline_pb2.TimelineFrame
    def __init__(self, frame: _Optional[_Union[_timeline_pb2.TimelineFrame, _Mapping]] = ...) -> None: ...

class LogEvent(_message.Message):
    __slots__ = ()
    class MetadataEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _struct_pb2.Value
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    STEP_INDEX_FIELD_NUMBER: _ClassVar[int]
    OCCURRED_AT_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    level: _shared_pb2.LogLevel
    message: str
    step_index: int
    occurred_at: _timestamp_pb2.Timestamp
    metadata: _containers.MessageMap[str, _struct_pb2.Value]
    def __init__(self, level: _Optional[_Union[_shared_pb2.LogLevel, str]] = ..., message: _Optional[str] = ..., step_index: _Optional[int] = ..., occurred_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., metadata: _Optional[_Mapping[str, _struct_pb2.Value]] = ...) -> None: ...

class HeartbeatEvent(_message.Message):
    __slots__ = ()
    class MetricsEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _struct_pb2.Value
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...
    RECEIVED_AT_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    received_at: _timestamp_pb2.Timestamp
    progress: int
    metrics: _containers.MessageMap[str, _struct_pb2.Value]
    def __init__(self, received_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., progress: _Optional[int] = ..., metrics: _Optional[_Mapping[str, _struct_pb2.Value]] = ...) -> None: ...

class TelemetryEvent(_message.Message):
    __slots__ = ()
    class MetricsEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _struct_pb2.Value
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...
    METRICS_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    metrics: _containers.MessageMap[str, _struct_pb2.Value]
    recorded_at: _timestamp_pb2.Timestamp
    def __init__(self, metrics: _Optional[_Mapping[str, _struct_pb2.Value]] = ..., recorded_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
