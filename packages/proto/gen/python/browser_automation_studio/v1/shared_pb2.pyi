from common.v1 import types_pb2 as _types_pb2
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

class SelectorType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SELECTOR_TYPE_UNSPECIFIED: _ClassVar[SelectorType]
    SELECTOR_TYPE_CSS: _ClassVar[SelectorType]
    SELECTOR_TYPE_XPATH: _ClassVar[SelectorType]
    SELECTOR_TYPE_ID: _ClassVar[SelectorType]
    SELECTOR_TYPE_DATA_TESTID: _ClassVar[SelectorType]
    SELECTOR_TYPE_ARIA: _ClassVar[SelectorType]
    SELECTOR_TYPE_TEXT: _ClassVar[SelectorType]
    SELECTOR_TYPE_ROLE: _ClassVar[SelectorType]
    SELECTOR_TYPE_PLACEHOLDER: _ClassVar[SelectorType]
    SELECTOR_TYPE_ALT_TEXT: _ClassVar[SelectorType]
    SELECTOR_TYPE_TITLE: _ClassVar[SelectorType]

class NetworkEventType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    NETWORK_EVENT_TYPE_UNSPECIFIED: _ClassVar[NetworkEventType]
    NETWORK_EVENT_TYPE_REQUEST: _ClassVar[NetworkEventType]
    NETWORK_EVENT_TYPE_RESPONSE: _ClassVar[NetworkEventType]
    NETWORK_EVENT_TYPE_FAILURE: _ClassVar[NetworkEventType]

class RecordingSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RECORDING_SOURCE_UNSPECIFIED: _ClassVar[RecordingSource]
    RECORDING_SOURCE_AUTO: _ClassVar[RecordingSource]
    RECORDING_SOURCE_MANUAL: _ClassVar[RecordingSource]

class WorkflowEdgeType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WORKFLOW_EDGE_TYPE_UNSPECIFIED: _ClassVar[WorkflowEdgeType]
    WORKFLOW_EDGE_TYPE_DEFAULT: _ClassVar[WorkflowEdgeType]
    WORKFLOW_EDGE_TYPE_SMOOTHSTEP: _ClassVar[WorkflowEdgeType]
    WORKFLOW_EDGE_TYPE_STEP: _ClassVar[WorkflowEdgeType]
    WORKFLOW_EDGE_TYPE_STRAIGHT: _ClassVar[WorkflowEdgeType]
    WORKFLOW_EDGE_TYPE_BEZIER: _ClassVar[WorkflowEdgeType]

class ValidationSeverity(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VALIDATION_SEVERITY_UNSPECIFIED: _ClassVar[ValidationSeverity]
    VALIDATION_SEVERITY_ERROR: _ClassVar[ValidationSeverity]
    VALIDATION_SEVERITY_WARNING: _ClassVar[ValidationSeverity]
    VALIDATION_SEVERITY_INFO: _ClassVar[ValidationSeverity]

class ChangeSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CHANGE_SOURCE_UNSPECIFIED: _ClassVar[ChangeSource]
    CHANGE_SOURCE_MANUAL: _ClassVar[ChangeSource]
    CHANGE_SOURCE_AUTOSAVE: _ClassVar[ChangeSource]
    CHANGE_SOURCE_IMPORT: _ClassVar[ChangeSource]
    CHANGE_SOURCE_AI_GENERATED: _ClassVar[ChangeSource]
    CHANGE_SOURCE_RECORDING: _ClassVar[ChangeSource]

class AssertionMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ASSERTION_MODE_UNSPECIFIED: _ClassVar[AssertionMode]
    ASSERTION_MODE_EXISTS: _ClassVar[AssertionMode]
    ASSERTION_MODE_NOT_EXISTS: _ClassVar[AssertionMode]
    ASSERTION_MODE_VISIBLE: _ClassVar[AssertionMode]
    ASSERTION_MODE_HIDDEN: _ClassVar[AssertionMode]
    ASSERTION_MODE_TEXT_EQUALS: _ClassVar[AssertionMode]
    ASSERTION_MODE_TEXT_CONTAINS: _ClassVar[AssertionMode]
    ASSERTION_MODE_ATTRIBUTE_EQUALS: _ClassVar[AssertionMode]
    ASSERTION_MODE_ATTRIBUTE_CONTAINS: _ClassVar[AssertionMode]
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
SELECTOR_TYPE_UNSPECIFIED: SelectorType
SELECTOR_TYPE_CSS: SelectorType
SELECTOR_TYPE_XPATH: SelectorType
SELECTOR_TYPE_ID: SelectorType
SELECTOR_TYPE_DATA_TESTID: SelectorType
SELECTOR_TYPE_ARIA: SelectorType
SELECTOR_TYPE_TEXT: SelectorType
SELECTOR_TYPE_ROLE: SelectorType
SELECTOR_TYPE_PLACEHOLDER: SelectorType
SELECTOR_TYPE_ALT_TEXT: SelectorType
SELECTOR_TYPE_TITLE: SelectorType
NETWORK_EVENT_TYPE_UNSPECIFIED: NetworkEventType
NETWORK_EVENT_TYPE_REQUEST: NetworkEventType
NETWORK_EVENT_TYPE_RESPONSE: NetworkEventType
NETWORK_EVENT_TYPE_FAILURE: NetworkEventType
RECORDING_SOURCE_UNSPECIFIED: RecordingSource
RECORDING_SOURCE_AUTO: RecordingSource
RECORDING_SOURCE_MANUAL: RecordingSource
WORKFLOW_EDGE_TYPE_UNSPECIFIED: WorkflowEdgeType
WORKFLOW_EDGE_TYPE_DEFAULT: WorkflowEdgeType
WORKFLOW_EDGE_TYPE_SMOOTHSTEP: WorkflowEdgeType
WORKFLOW_EDGE_TYPE_STEP: WorkflowEdgeType
WORKFLOW_EDGE_TYPE_STRAIGHT: WorkflowEdgeType
WORKFLOW_EDGE_TYPE_BEZIER: WorkflowEdgeType
VALIDATION_SEVERITY_UNSPECIFIED: ValidationSeverity
VALIDATION_SEVERITY_ERROR: ValidationSeverity
VALIDATION_SEVERITY_WARNING: ValidationSeverity
VALIDATION_SEVERITY_INFO: ValidationSeverity
CHANGE_SOURCE_UNSPECIFIED: ChangeSource
CHANGE_SOURCE_MANUAL: ChangeSource
CHANGE_SOURCE_AUTOSAVE: ChangeSource
CHANGE_SOURCE_IMPORT: ChangeSource
CHANGE_SOURCE_AI_GENERATED: ChangeSource
CHANGE_SOURCE_RECORDING: ChangeSource
ASSERTION_MODE_UNSPECIFIED: AssertionMode
ASSERTION_MODE_EXISTS: AssertionMode
ASSERTION_MODE_NOT_EXISTS: AssertionMode
ASSERTION_MODE_VISIBLE: AssertionMode
ASSERTION_MODE_HIDDEN: AssertionMode
ASSERTION_MODE_TEXT_EQUALS: AssertionMode
ASSERTION_MODE_TEXT_CONTAINS: AssertionMode
ASSERTION_MODE_ATTRIBUTE_EQUALS: AssertionMode
ASSERTION_MODE_ATTRIBUTE_CONTAINS: AssertionMode

class RetryAttempt(_message.Message):
    __slots__ = ()
    ATTEMPT_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    attempt: int
    success: bool
    duration_ms: int
    error: str
    def __init__(self, attempt: _Optional[int] = ..., success: _Optional[bool] = ..., duration_ms: _Optional[int] = ..., error: _Optional[str] = ...) -> None: ...

class RetryStatus(_message.Message):
    __slots__ = ()
    CURRENT_ATTEMPT_FIELD_NUMBER: _ClassVar[int]
    MAX_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    DELAY_MS_FIELD_NUMBER: _ClassVar[int]
    BACKOFF_FACTOR_FIELD_NUMBER: _ClassVar[int]
    CONFIGURED_FIELD_NUMBER: _ClassVar[int]
    HISTORY_FIELD_NUMBER: _ClassVar[int]
    current_attempt: int
    max_attempts: int
    delay_ms: int
    backoff_factor: float
    configured: bool
    history: _containers.RepeatedCompositeFieldContainer[RetryAttempt]
    def __init__(self, current_attempt: _Optional[int] = ..., max_attempts: _Optional[int] = ..., delay_ms: _Optional[int] = ..., backoff_factor: _Optional[float] = ..., configured: _Optional[bool] = ..., history: _Optional[_Iterable[_Union[RetryAttempt, _Mapping]]] = ...) -> None: ...

class AssertionResult(_message.Message):
    __slots__ = ()
    MODE_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    NEGATED_FIELD_NUMBER: _ClassVar[int]
    CASE_SENSITIVE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    mode: AssertionMode
    selector: str
    expected: _types_pb2.JsonValue
    actual: _types_pb2.JsonValue
    success: bool
    negated: bool
    case_sensitive: bool
    message: str
    def __init__(self, mode: _Optional[_Union[AssertionMode, str]] = ..., selector: _Optional[str] = ..., expected: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ..., actual: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ..., success: _Optional[bool] = ..., negated: _Optional[bool] = ..., case_sensitive: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...
