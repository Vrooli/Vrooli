import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from common.v1 import types_pb2 as _types_pb2
from browser_automation_studio.v1.base import geometry_pb2 as _geometry_pb2
from browser_automation_studio.v1.domain import selectors_pb2 as _selectors_pb2
from browser_automation_studio.v1.timeline import entry_pb2 as _entry_pb2
from browser_automation_studio.v1.actions import action_pb2 as _action_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class FailureKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FAILURE_KIND_UNSPECIFIED: _ClassVar[FailureKind]
    FAILURE_KIND_ENGINE: _ClassVar[FailureKind]
    FAILURE_KIND_INFRA: _ClassVar[FailureKind]
    FAILURE_KIND_ORCHESTRATION: _ClassVar[FailureKind]
    FAILURE_KIND_USER: _ClassVar[FailureKind]
    FAILURE_KIND_TIMEOUT: _ClassVar[FailureKind]
    FAILURE_KIND_CANCELLED: _ClassVar[FailureKind]

class FailureSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FAILURE_SOURCE_UNSPECIFIED: _ClassVar[FailureSource]
    FAILURE_SOURCE_ENGINE: _ClassVar[FailureSource]
    FAILURE_SOURCE_EXECUTOR: _ClassVar[FailureSource]
    FAILURE_SOURCE_RECORDER: _ClassVar[FailureSource]
FAILURE_KIND_UNSPECIFIED: FailureKind
FAILURE_KIND_ENGINE: FailureKind
FAILURE_KIND_INFRA: FailureKind
FAILURE_KIND_ORCHESTRATION: FailureKind
FAILURE_KIND_USER: FailureKind
FAILURE_KIND_TIMEOUT: FailureKind
FAILURE_KIND_CANCELLED: FailureKind
FAILURE_SOURCE_UNSPECIFIED: FailureSource
FAILURE_SOURCE_ENGINE: FailureSource
FAILURE_SOURCE_EXECUTOR: FailureSource
FAILURE_SOURCE_RECORDER: FailureSource

class StepFailure(_message.Message):
    __slots__ = ()
    class DetailsEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    KIND_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    FATAL_FIELD_NUMBER: _ClassVar[int]
    RETRYABLE_FIELD_NUMBER: _ClassVar[int]
    OCCURRED_AT_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    kind: FailureKind
    code: str
    message: str
    fatal: bool
    retryable: bool
    occurred_at: _timestamp_pb2.Timestamp
    details: _containers.MessageMap[str, _types_pb2.JsonValue]
    source: FailureSource
    def __init__(self, kind: _Optional[_Union[FailureKind, str]] = ..., code: _Optional[str] = ..., message: _Optional[str] = ..., fatal: _Optional[bool] = ..., retryable: _Optional[bool] = ..., occurred_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., details: _Optional[_Mapping[str, _types_pb2.JsonValue]] = ..., source: _Optional[_Union[FailureSource, str]] = ...) -> None: ...

class DriverScreenshot(_message.Message):
    __slots__ = ()
    DATA_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_TIME_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    HASH_FIELD_NUMBER: _ClassVar[int]
    FROM_CACHE_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    data: bytes
    media_type: str
    capture_time: _timestamp_pb2.Timestamp
    width: int
    height: int
    hash: str
    from_cache: bool
    truncated: bool
    source: str
    def __init__(self, data: _Optional[bytes] = ..., media_type: _Optional[str] = ..., capture_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., hash: _Optional[str] = ..., from_cache: _Optional[bool] = ..., truncated: _Optional[bool] = ..., source: _Optional[str] = ...) -> None: ...

class DOMSnapshot(_message.Message):
    __slots__ = ()
    HTML_FIELD_NUMBER: _ClassVar[int]
    PREVIEW_FIELD_NUMBER: _ClassVar[int]
    HASH_FIELD_NUMBER: _ClassVar[int]
    COLLECTED_AT_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    html: str
    preview: str
    hash: str
    collected_at: _timestamp_pb2.Timestamp
    truncated: bool
    def __init__(self, html: _Optional[str] = ..., preview: _Optional[str] = ..., hash: _Optional[str] = ..., collected_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., truncated: _Optional[bool] = ...) -> None: ...

class DriverConsoleLogEntry(_message.Message):
    __slots__ = ()
    TYPE_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    STACK_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    type: str
    text: str
    timestamp: _timestamp_pb2.Timestamp
    stack: str
    location: str
    def __init__(self, type: _Optional[str] = ..., text: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., stack: _Optional[str] = ..., location: _Optional[str] = ...) -> None: ...

class DriverNetworkEvent(_message.Message):
    __slots__ = ()
    class RequestHeadersEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class ResponseHeadersEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TYPE_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_TYPE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    OK_FIELD_NUMBER: _ClassVar[int]
    FAILURE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    REQUEST_HEADERS_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_HEADERS_FIELD_NUMBER: _ClassVar[int]
    REQUEST_BODY_PREVIEW_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_BODY_PREVIEW_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    type: str
    url: str
    method: str
    resource_type: str
    status: int
    ok: bool
    failure: str
    timestamp: _timestamp_pb2.Timestamp
    request_headers: _containers.ScalarMap[str, str]
    response_headers: _containers.ScalarMap[str, str]
    request_body_preview: str
    response_body_preview: str
    truncated: bool
    def __init__(self, type: _Optional[str] = ..., url: _Optional[str] = ..., method: _Optional[str] = ..., resource_type: _Optional[str] = ..., status: _Optional[int] = ..., ok: _Optional[bool] = ..., failure: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., request_headers: _Optional[_Mapping[str, str]] = ..., response_headers: _Optional[_Mapping[str, str]] = ..., request_body_preview: _Optional[str] = ..., response_body_preview: _Optional[str] = ..., truncated: _Optional[bool] = ...) -> None: ...

class CursorPosition(_message.Message):
    __slots__ = ()
    POINT_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    ELAPSED_MS_FIELD_NUMBER: _ClassVar[int]
    point: _geometry_pb2.Point
    recorded_at: _timestamp_pb2.Timestamp
    elapsed_ms: int
    def __init__(self, point: _Optional[_Union[_geometry_pb2.Point, _Mapping]] = ..., recorded_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., elapsed_ms: _Optional[int] = ...) -> None: ...

class ConditionOutcome(_message.Message):
    __slots__ = ()
    TYPE_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    NEGATED_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_FIELD_NUMBER: _ClassVar[int]
    VARIABLE_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    EXPRESSION_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_FIELD_NUMBER: _ClassVar[int]
    type: str
    outcome: bool
    negated: bool
    operator: str
    variable: str
    selector: str
    expression: str
    actual: _types_pb2.JsonValue
    expected: _types_pb2.JsonValue
    def __init__(self, type: _Optional[str] = ..., outcome: _Optional[bool] = ..., negated: _Optional[bool] = ..., operator: _Optional[str] = ..., variable: _Optional[str] = ..., selector: _Optional[str] = ..., expression: _Optional[str] = ..., actual: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ..., expected: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...

class AssertionOutcome(_message.Message):
    __slots__ = ()
    MODE_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    NEGATED_FIELD_NUMBER: _ClassVar[int]
    CASE_SENSITIVE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    mode: str
    selector: str
    expected: _types_pb2.JsonValue
    actual: _types_pb2.JsonValue
    success: bool
    negated: bool
    case_sensitive: bool
    message: str
    def __init__(self, mode: _Optional[str] = ..., selector: _Optional[str] = ..., expected: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ..., actual: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ..., success: _Optional[bool] = ..., negated: _Optional[bool] = ..., case_sensitive: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...

class StepOutcome(_message.Message):
    __slots__ = ()
    class ExtractedDataEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    class ProbeResultEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    class NotesEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_VERSION_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    CORRELATION_ID_FIELD_NUMBER: _ClassVar[int]
    STEP_INDEX_FIELD_NUMBER: _ClassVar[int]
    ATTEMPT_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    STEP_TYPE_FIELD_NUMBER: _ClassVar[int]
    INSTRUCTION_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    FINAL_URL_FIELD_NUMBER: _ClassVar[int]
    SCREENSHOT_FIELD_NUMBER: _ClassVar[int]
    DOM_SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    CONSOLE_LOGS_FIELD_NUMBER: _ClassVar[int]
    NETWORK_EVENTS_FIELD_NUMBER: _ClassVar[int]
    EXTRACTED_DATA_FIELD_NUMBER: _ClassVar[int]
    ASSERTION_FIELD_NUMBER: _ClassVar[int]
    CONDITION_FIELD_NUMBER: _ClassVar[int]
    PROBE_RESULT_FIELD_NUMBER: _ClassVar[int]
    ELEMENT_BOUNDING_BOX_FIELD_NUMBER: _ClassVar[int]
    CLICK_POSITION_FIELD_NUMBER: _ClassVar[int]
    FOCUSED_ELEMENT_FIELD_NUMBER: _ClassVar[int]
    HIGHLIGHT_REGIONS_FIELD_NUMBER: _ClassVar[int]
    MASK_REGIONS_FIELD_NUMBER: _ClassVar[int]
    ZOOM_FACTOR_FIELD_NUMBER: _ClassVar[int]
    CURSOR_TRAIL_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    FAILURE_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    payload_version: str
    execution_id: str
    correlation_id: str
    step_index: int
    attempt: int
    node_id: str
    step_type: str
    instruction: str
    success: bool
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    duration_ms: int
    final_url: str
    screenshot: DriverScreenshot
    dom_snapshot: DOMSnapshot
    console_logs: _containers.RepeatedCompositeFieldContainer[DriverConsoleLogEntry]
    network_events: _containers.RepeatedCompositeFieldContainer[DriverNetworkEvent]
    extracted_data: _containers.MessageMap[str, _types_pb2.JsonValue]
    assertion: AssertionOutcome
    condition: ConditionOutcome
    probe_result: _containers.MessageMap[str, _types_pb2.JsonValue]
    element_bounding_box: _geometry_pb2.BoundingBox
    click_position: _geometry_pb2.Point
    focused_element: _entry_pb2.ElementFocus
    highlight_regions: _containers.RepeatedCompositeFieldContainer[_selectors_pb2.HighlightRegion]
    mask_regions: _containers.RepeatedCompositeFieldContainer[_selectors_pb2.MaskRegion]
    zoom_factor: float
    cursor_trail: _containers.RepeatedCompositeFieldContainer[CursorPosition]
    notes: _containers.ScalarMap[str, str]
    failure: StepFailure
    def __init__(self, schema_version: _Optional[str] = ..., payload_version: _Optional[str] = ..., execution_id: _Optional[str] = ..., correlation_id: _Optional[str] = ..., step_index: _Optional[int] = ..., attempt: _Optional[int] = ..., node_id: _Optional[str] = ..., step_type: _Optional[str] = ..., instruction: _Optional[str] = ..., success: _Optional[bool] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., duration_ms: _Optional[int] = ..., final_url: _Optional[str] = ..., screenshot: _Optional[_Union[DriverScreenshot, _Mapping]] = ..., dom_snapshot: _Optional[_Union[DOMSnapshot, _Mapping]] = ..., console_logs: _Optional[_Iterable[_Union[DriverConsoleLogEntry, _Mapping]]] = ..., network_events: _Optional[_Iterable[_Union[DriverNetworkEvent, _Mapping]]] = ..., extracted_data: _Optional[_Mapping[str, _types_pb2.JsonValue]] = ..., assertion: _Optional[_Union[AssertionOutcome, _Mapping]] = ..., condition: _Optional[_Union[ConditionOutcome, _Mapping]] = ..., probe_result: _Optional[_Mapping[str, _types_pb2.JsonValue]] = ..., element_bounding_box: _Optional[_Union[_geometry_pb2.BoundingBox, _Mapping]] = ..., click_position: _Optional[_Union[_geometry_pb2.Point, _Mapping]] = ..., focused_element: _Optional[_Union[_entry_pb2.ElementFocus, _Mapping]] = ..., highlight_regions: _Optional[_Iterable[_Union[_selectors_pb2.HighlightRegion, _Mapping]]] = ..., mask_regions: _Optional[_Iterable[_Union[_selectors_pb2.MaskRegion, _Mapping]]] = ..., zoom_factor: _Optional[float] = ..., cursor_trail: _Optional[_Iterable[_Union[CursorPosition, _Mapping]]] = ..., notes: _Optional[_Mapping[str, str]] = ..., failure: _Optional[_Union[StepFailure, _Mapping]] = ...) -> None: ...

class CompiledInstruction(_message.Message):
    __slots__ = ()
    class ParamsEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    class ContextEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    class MetadataEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    INDEX_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    PARAMS_FIELD_NUMBER: _ClassVar[int]
    PRELOAD_HTML_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    index: int
    node_id: str
    type: str
    params: _containers.MessageMap[str, _types_pb2.JsonValue]
    preload_html: str
    context: _containers.MessageMap[str, _types_pb2.JsonValue]
    metadata: _containers.ScalarMap[str, str]
    action: _action_pb2.ActionDefinition
    def __init__(self, index: _Optional[int] = ..., node_id: _Optional[str] = ..., type: _Optional[str] = ..., params: _Optional[_Mapping[str, _types_pb2.JsonValue]] = ..., preload_html: _Optional[str] = ..., context: _Optional[_Mapping[str, _types_pb2.JsonValue]] = ..., metadata: _Optional[_Mapping[str, str]] = ..., action: _Optional[_Union[_action_pb2.ActionDefinition, _Mapping]] = ...) -> None: ...

class PlanEdge(_message.Message):
    __slots__ = ()
    ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    CONDITION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PORT_FIELD_NUMBER: _ClassVar[int]
    TARGET_PORT_FIELD_NUMBER: _ClassVar[int]
    id: str
    target: str
    condition: str
    source_port: str
    target_port: str
    def __init__(self, id: _Optional[str] = ..., target: _Optional[str] = ..., condition: _Optional[str] = ..., source_port: _Optional[str] = ..., target_port: _Optional[str] = ...) -> None: ...

class PlanStep(_message.Message):
    __slots__ = ()
    class ParamsEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    class MetadataEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class ContextEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    class SourcePositionEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    INDEX_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    PARAMS_FIELD_NUMBER: _ClassVar[int]
    OUTGOING_FIELD_NUMBER: _ClassVar[int]
    LOOP_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    PRELOAD_HTML_FIELD_NUMBER: _ClassVar[int]
    SOURCE_POSITION_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    index: int
    node_id: str
    type: str
    params: _containers.MessageMap[str, _types_pb2.JsonValue]
    outgoing: _containers.RepeatedCompositeFieldContainer[PlanEdge]
    loop: PlanGraph
    metadata: _containers.ScalarMap[str, str]
    context: _containers.MessageMap[str, _types_pb2.JsonValue]
    preload_html: str
    source_position: _containers.MessageMap[str, _types_pb2.JsonValue]
    action: _action_pb2.ActionDefinition
    def __init__(self, index: _Optional[int] = ..., node_id: _Optional[str] = ..., type: _Optional[str] = ..., params: _Optional[_Mapping[str, _types_pb2.JsonValue]] = ..., outgoing: _Optional[_Iterable[_Union[PlanEdge, _Mapping]]] = ..., loop: _Optional[_Union[PlanGraph, _Mapping]] = ..., metadata: _Optional[_Mapping[str, str]] = ..., context: _Optional[_Mapping[str, _types_pb2.JsonValue]] = ..., preload_html: _Optional[str] = ..., source_position: _Optional[_Mapping[str, _types_pb2.JsonValue]] = ..., action: _Optional[_Union[_action_pb2.ActionDefinition, _Mapping]] = ...) -> None: ...

class PlanGraph(_message.Message):
    __slots__ = ()
    STEPS_FIELD_NUMBER: _ClassVar[int]
    steps: _containers.RepeatedCompositeFieldContainer[PlanStep]
    def __init__(self, steps: _Optional[_Iterable[_Union[PlanStep, _Mapping]]] = ...) -> None: ...

class ExecutionPlan(_message.Message):
    __slots__ = ()
    class MetadataEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_VERSION_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    INSTRUCTIONS_FIELD_NUMBER: _ClassVar[int]
    GRAPH_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    payload_version: str
    execution_id: str
    workflow_id: str
    instructions: _containers.RepeatedCompositeFieldContainer[CompiledInstruction]
    graph: PlanGraph
    metadata: _containers.MessageMap[str, _types_pb2.JsonValue]
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, schema_version: _Optional[str] = ..., payload_version: _Optional[str] = ..., execution_id: _Optional[str] = ..., workflow_id: _Optional[str] = ..., instructions: _Optional[_Iterable[_Union[CompiledInstruction, _Mapping]]] = ..., graph: _Optional[_Union[PlanGraph, _Mapping]] = ..., metadata: _Optional[_Mapping[str, _types_pb2.JsonValue]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
