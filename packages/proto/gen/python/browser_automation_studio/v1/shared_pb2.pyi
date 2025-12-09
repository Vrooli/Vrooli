from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ExecutionStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EXECUTION_STATUS_UNSPECIFIED: _ClassVar[ExecutionStatus]
    EXECUTION_STATUS_PENDING: _ClassVar[ExecutionStatus]
    EXECUTION_STATUS_RUNNING: _ClassVar[ExecutionStatus]
    EXECUTION_STATUS_COMPLETED: _ClassVar[ExecutionStatus]
    EXECUTION_STATUS_FAILED: _ClassVar[ExecutionStatus]
    EXECUTION_STATUS_CANCELLED: _ClassVar[ExecutionStatus]

class TriggerType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TRIGGER_TYPE_UNSPECIFIED: _ClassVar[TriggerType]
    TRIGGER_TYPE_MANUAL: _ClassVar[TriggerType]
    TRIGGER_TYPE_SCHEDULED: _ClassVar[TriggerType]
    TRIGGER_TYPE_API: _ClassVar[TriggerType]
    TRIGGER_TYPE_WEBHOOK: _ClassVar[TriggerType]

class StepType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    STEP_TYPE_UNSPECIFIED: _ClassVar[StepType]
    STEP_TYPE_NAVIGATE: _ClassVar[StepType]
    STEP_TYPE_CLICK: _ClassVar[StepType]
    STEP_TYPE_ASSERT: _ClassVar[StepType]
    STEP_TYPE_SUBFLOW: _ClassVar[StepType]
    STEP_TYPE_INPUT: _ClassVar[StepType]
    STEP_TYPE_CUSTOM: _ClassVar[StepType]
    STEP_TYPE_WAIT: _ClassVar[StepType]

class StepStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    STEP_STATUS_UNSPECIFIED: _ClassVar[StepStatus]
    STEP_STATUS_PENDING: _ClassVar[StepStatus]
    STEP_STATUS_RUNNING: _ClassVar[StepStatus]
    STEP_STATUS_COMPLETED: _ClassVar[StepStatus]
    STEP_STATUS_FAILED: _ClassVar[StepStatus]
    STEP_STATUS_CANCELLED: _ClassVar[StepStatus]
    STEP_STATUS_SKIPPED: _ClassVar[StepStatus]
    STEP_STATUS_RETRYING: _ClassVar[StepStatus]

class LogLevel(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    LOG_LEVEL_UNSPECIFIED: _ClassVar[LogLevel]
    LOG_LEVEL_DEBUG: _ClassVar[LogLevel]
    LOG_LEVEL_INFO: _ClassVar[LogLevel]
    LOG_LEVEL_WARN: _ClassVar[LogLevel]
    LOG_LEVEL_ERROR: _ClassVar[LogLevel]

class ArtifactType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ARTIFACT_TYPE_UNSPECIFIED: _ClassVar[ArtifactType]
    ARTIFACT_TYPE_TIMELINE_FRAME: _ClassVar[ArtifactType]
    ARTIFACT_TYPE_CONSOLE_LOG: _ClassVar[ArtifactType]
    ARTIFACT_TYPE_NETWORK_EVENT: _ClassVar[ArtifactType]
    ARTIFACT_TYPE_SCREENSHOT: _ClassVar[ArtifactType]
    ARTIFACT_TYPE_DOM_SNAPSHOT: _ClassVar[ArtifactType]
    ARTIFACT_TYPE_TRACE: _ClassVar[ArtifactType]
    ARTIFACT_TYPE_CUSTOM: _ClassVar[ArtifactType]

class EventKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EVENT_KIND_UNSPECIFIED: _ClassVar[EventKind]
    EVENT_KIND_STATUS_UPDATE: _ClassVar[EventKind]
    EVENT_KIND_TIMELINE_FRAME: _ClassVar[EventKind]
    EVENT_KIND_LOG: _ClassVar[EventKind]
    EVENT_KIND_HEARTBEAT: _ClassVar[EventKind]
    EVENT_KIND_TELEMETRY: _ClassVar[EventKind]

class ExportStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EXPORT_STATUS_UNSPECIFIED: _ClassVar[ExportStatus]
    EXPORT_STATUS_READY: _ClassVar[ExportStatus]
    EXPORT_STATUS_PENDING: _ClassVar[ExportStatus]
    EXPORT_STATUS_ERROR: _ClassVar[ExportStatus]
    EXPORT_STATUS_UNAVAILABLE: _ClassVar[ExportStatus]
EXECUTION_STATUS_UNSPECIFIED: ExecutionStatus
EXECUTION_STATUS_PENDING: ExecutionStatus
EXECUTION_STATUS_RUNNING: ExecutionStatus
EXECUTION_STATUS_COMPLETED: ExecutionStatus
EXECUTION_STATUS_FAILED: ExecutionStatus
EXECUTION_STATUS_CANCELLED: ExecutionStatus
TRIGGER_TYPE_UNSPECIFIED: TriggerType
TRIGGER_TYPE_MANUAL: TriggerType
TRIGGER_TYPE_SCHEDULED: TriggerType
TRIGGER_TYPE_API: TriggerType
TRIGGER_TYPE_WEBHOOK: TriggerType
STEP_TYPE_UNSPECIFIED: StepType
STEP_TYPE_NAVIGATE: StepType
STEP_TYPE_CLICK: StepType
STEP_TYPE_ASSERT: StepType
STEP_TYPE_SUBFLOW: StepType
STEP_TYPE_INPUT: StepType
STEP_TYPE_CUSTOM: StepType
STEP_TYPE_WAIT: StepType
STEP_STATUS_UNSPECIFIED: StepStatus
STEP_STATUS_PENDING: StepStatus
STEP_STATUS_RUNNING: StepStatus
STEP_STATUS_COMPLETED: StepStatus
STEP_STATUS_FAILED: StepStatus
STEP_STATUS_CANCELLED: StepStatus
STEP_STATUS_SKIPPED: StepStatus
STEP_STATUS_RETRYING: StepStatus
LOG_LEVEL_UNSPECIFIED: LogLevel
LOG_LEVEL_DEBUG: LogLevel
LOG_LEVEL_INFO: LogLevel
LOG_LEVEL_WARN: LogLevel
LOG_LEVEL_ERROR: LogLevel
ARTIFACT_TYPE_UNSPECIFIED: ArtifactType
ARTIFACT_TYPE_TIMELINE_FRAME: ArtifactType
ARTIFACT_TYPE_CONSOLE_LOG: ArtifactType
ARTIFACT_TYPE_NETWORK_EVENT: ArtifactType
ARTIFACT_TYPE_SCREENSHOT: ArtifactType
ARTIFACT_TYPE_DOM_SNAPSHOT: ArtifactType
ARTIFACT_TYPE_TRACE: ArtifactType
ARTIFACT_TYPE_CUSTOM: ArtifactType
EVENT_KIND_UNSPECIFIED: EventKind
EVENT_KIND_STATUS_UPDATE: EventKind
EVENT_KIND_TIMELINE_FRAME: EventKind
EVENT_KIND_LOG: EventKind
EVENT_KIND_HEARTBEAT: EventKind
EVENT_KIND_TELEMETRY: EventKind
EXPORT_STATUS_UNSPECIFIED: ExportStatus
EXPORT_STATUS_READY: ExportStatus
EXPORT_STATUS_PENDING: ExportStatus
EXPORT_STATUS_ERROR: ExportStatus
EXPORT_STATUS_UNAVAILABLE: ExportStatus

class JsonValue(_message.Message):
    __slots__ = ()
    BOOL_VALUE_FIELD_NUMBER: _ClassVar[int]
    INT_VALUE_FIELD_NUMBER: _ClassVar[int]
    DOUBLE_VALUE_FIELD_NUMBER: _ClassVar[int]
    STRING_VALUE_FIELD_NUMBER: _ClassVar[int]
    OBJECT_VALUE_FIELD_NUMBER: _ClassVar[int]
    LIST_VALUE_FIELD_NUMBER: _ClassVar[int]
    NULL_VALUE_FIELD_NUMBER: _ClassVar[int]
    BYTES_VALUE_FIELD_NUMBER: _ClassVar[int]
    bool_value: bool
    int_value: int
    double_value: float
    string_value: str
    object_value: JsonObject
    list_value: JsonList
    null_value: _struct_pb2.NullValue
    bytes_value: bytes
    def __init__(self, bool_value: _Optional[bool] = ..., int_value: _Optional[int] = ..., double_value: _Optional[float] = ..., string_value: _Optional[str] = ..., object_value: _Optional[_Union[JsonObject, _Mapping]] = ..., list_value: _Optional[_Union[JsonList, _Mapping]] = ..., null_value: _Optional[_Union[_struct_pb2.NullValue, str]] = ..., bytes_value: _Optional[bytes] = ...) -> None: ...

class JsonObject(_message.Message):
    __slots__ = ()
    class FieldsEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[JsonValue, _Mapping]] = ...) -> None: ...
    FIELDS_FIELD_NUMBER: _ClassVar[int]
    fields: _containers.MessageMap[str, JsonValue]
    def __init__(self, fields: _Optional[_Mapping[str, JsonValue]] = ...) -> None: ...

class JsonList(_message.Message):
    __slots__ = ()
    VALUES_FIELD_NUMBER: _ClassVar[int]
    values: _containers.RepeatedCompositeFieldContainer[JsonValue]
    def __init__(self, values: _Optional[_Iterable[_Union[JsonValue, _Mapping]]] = ...) -> None: ...
