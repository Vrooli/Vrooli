import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from common.v1 import types_pb2 as _types_pb2
from browser_automation_studio.v1 import action_pb2 as _action_pb2
from browser_automation_studio.v1 import shared_pb2 as _shared_pb2
from browser_automation_studio.v1 import telemetry_pb2 as _telemetry_pb2
from browser_automation_studio.v1 import timeline_pb2 as _timeline_pb2
from browser_automation_studio.v1 import workflow_v2_pb2 as _workflow_v2_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ExecutionParameters(_message.Message):
    __slots__ = ()
    class VariablesEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    START_URL_FIELD_NUMBER: _ClassVar[int]
    VARIABLES_FIELD_NUMBER: _ClassVar[int]
    VIEWPORT_WIDTH_FIELD_NUMBER: _ClassVar[int]
    VIEWPORT_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    HEADLESS_FIELD_NUMBER: _ClassVar[int]
    USER_AGENT_FIELD_NUMBER: _ClassVar[int]
    LOCALE_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    start_url: str
    variables: _containers.ScalarMap[str, str]
    viewport_width: int
    viewport_height: int
    headless: bool
    user_agent: str
    locale: str
    timeout_ms: int
    def __init__(self, start_url: _Optional[str] = ..., variables: _Optional[_Mapping[str, str]] = ..., viewport_width: _Optional[int] = ..., viewport_height: _Optional[int] = ..., headless: _Optional[bool] = ..., user_agent: _Optional[str] = ..., locale: _Optional[str] = ..., timeout_ms: _Optional[int] = ...) -> None: ...

class ExecutionResult(_message.Message):
    __slots__ = ()
    class ExtractedDataEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    class ScreenshotArtifactsEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: int
        value: str
        def __init__(self, key: _Optional[int] = ..., value: _Optional[str] = ...) -> None: ...
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    STEPS_EXECUTED_FIELD_NUMBER: _ClassVar[int]
    STEPS_FAILED_FIELD_NUMBER: _ClassVar[int]
    FINAL_URL_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    ERROR_CODE_FIELD_NUMBER: _ClassVar[int]
    EXTRACTED_DATA_FIELD_NUMBER: _ClassVar[int]
    SCREENSHOT_ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    steps_executed: int
    steps_failed: int
    final_url: str
    error: str
    error_code: str
    extracted_data: _containers.MessageMap[str, _types_pb2.JsonValue]
    screenshot_artifacts: _containers.ScalarMap[int, str]
    def __init__(self, success: _Optional[bool] = ..., steps_executed: _Optional[int] = ..., steps_failed: _Optional[int] = ..., final_url: _Optional[str] = ..., error: _Optional[str] = ..., error_code: _Optional[str] = ..., extracted_data: _Optional[_Mapping[str, _types_pb2.JsonValue]] = ..., screenshot_artifacts: _Optional[_Mapping[int, str]] = ...) -> None: ...

class TriggerMetadata(_message.Message):
    __slots__ = ()
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    CLIENT_ID_FIELD_NUMBER: _ClassVar[int]
    SCHEDULE_ID_FIELD_NUMBER: _ClassVar[int]
    WEBHOOK_ID_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_IP_FIELD_NUMBER: _ClassVar[int]
    USER_AGENT_FIELD_NUMBER: _ClassVar[int]
    user_id: str
    client_id: str
    schedule_id: str
    webhook_id: str
    external_request_id: str
    source_ip: str
    user_agent: str
    def __init__(self, user_id: _Optional[str] = ..., client_id: _Optional[str] = ..., schedule_id: _Optional[str] = ..., webhook_id: _Optional[str] = ..., external_request_id: _Optional[str] = ..., source_ip: _Optional[str] = ..., user_agent: _Optional[str] = ...) -> None: ...

class Execution(_message.Message):
    __slots__ = ()
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_VERSION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_TYPE_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_HEARTBEAT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    CURRENT_STEP_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_METADATA_FIELD_NUMBER: _ClassVar[int]
    TRACE_ID_FIELD_NUMBER: _ClassVar[int]
    CORRELATION_ID_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    workflow_id: str
    workflow_version: int
    status: _shared_pb2.ExecutionStatus
    trigger_type: _shared_pb2.TriggerType
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    last_heartbeat: _timestamp_pb2.Timestamp
    error: str
    progress: int
    current_step: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    parameters: ExecutionParameters
    result: ExecutionResult
    trigger_metadata: TriggerMetadata
    trace_id: str
    correlation_id: str
    request_id: str
    def __init__(self, execution_id: _Optional[str] = ..., workflow_id: _Optional[str] = ..., workflow_version: _Optional[int] = ..., status: _Optional[_Union[_shared_pb2.ExecutionStatus, str]] = ..., trigger_type: _Optional[_Union[_shared_pb2.TriggerType, str]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_heartbeat: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., error: _Optional[str] = ..., progress: _Optional[int] = ..., current_step: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., parameters: _Optional[_Union[ExecutionParameters, _Mapping]] = ..., result: _Optional[_Union[ExecutionResult, _Mapping]] = ..., trigger_metadata: _Optional[_Union[TriggerMetadata, _Mapping]] = ..., trace_id: _Optional[str] = ..., correlation_id: _Optional[str] = ..., request_id: _Optional[str] = ...) -> None: ...

class ExecuteAdhocRequest(_message.Message):
    __slots__ = ()
    FLOW_DEFINITION_FIELD_NUMBER: _ClassVar[int]
    WAIT_FOR_COMPLETION_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_FIELD_NUMBER: _ClassVar[int]
    flow_definition: _workflow_v2_pb2.WorkflowDefinitionV2
    wait_for_completion: bool
    metadata: ExecutionMetadata
    parameters: ExecutionParameters
    def __init__(self, flow_definition: _Optional[_Union[_workflow_v2_pb2.WorkflowDefinitionV2, _Mapping]] = ..., wait_for_completion: _Optional[bool] = ..., metadata: _Optional[_Union[ExecutionMetadata, _Mapping]] = ..., parameters: _Optional[_Union[ExecutionParameters, _Mapping]] = ...) -> None: ...

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

class ExecutionScreenshot(_message.Message):
    __slots__ = ()
    SCREENSHOT_FIELD_NUMBER: _ClassVar[int]
    STEP_INDEX_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    STEP_LABEL_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    screenshot: _telemetry_pb2.TimelineScreenshot
    step_index: int
    node_id: str
    step_label: str
    captured_at: _timestamp_pb2.Timestamp
    def __init__(self, screenshot: _Optional[_Union[_telemetry_pb2.TimelineScreenshot, _Mapping]] = ..., step_index: _Optional[int] = ..., node_id: _Optional[str] = ..., step_label: _Optional[str] = ..., captured_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetScreenshotsResponse(_message.Message):
    __slots__ = ()
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    SCREENSHOTS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    screenshots: _containers.RepeatedCompositeFieldContainer[ExecutionScreenshot]
    total: int
    def __init__(self, execution_id: _Optional[str] = ..., screenshots: _Optional[_Iterable[_Union[ExecutionScreenshot, _Mapping]]] = ..., total: _Optional[int] = ...) -> None: ...

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
    execution_id: str
    spec_id: str
    status: _shared_pb2.ExportStatus
    message: str
    captured_frame_count: int
    available_asset_count: int
    total_duration_ms: int
    package: _types_pb2.JsonObject
    def __init__(self, execution_id: _Optional[str] = ..., spec_id: _Optional[str] = ..., status: _Optional[_Union[_shared_pb2.ExportStatus, str]] = ..., message: _Optional[str] = ..., captured_frame_count: _Optional[int] = ..., available_asset_count: _Optional[int] = ..., total_duration_ms: _Optional[int] = ..., package: _Optional[_Union[_types_pb2.JsonObject, _Mapping]] = ...) -> None: ...

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

class LogMetadata(_message.Message):
    __slots__ = ()
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_TYPE_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    STACK_TRACE_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    action_type: _action_pb2.ActionType
    url: str
    stack_trace: str
    component: str
    def __init__(self, node_id: _Optional[str] = ..., action_type: _Optional[_Union[_action_pb2.ActionType, str]] = ..., url: _Optional[str] = ..., stack_trace: _Optional[str] = ..., component: _Optional[str] = ...) -> None: ...

class LogEvent(_message.Message):
    __slots__ = ()
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    STEP_INDEX_FIELD_NUMBER: _ClassVar[int]
    OCCURRED_AT_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    level: _shared_pb2.LogLevel
    message: str
    step_index: int
    occurred_at: _timestamp_pb2.Timestamp
    metadata: LogMetadata
    def __init__(self, level: _Optional[_Union[_shared_pb2.LogLevel, str]] = ..., message: _Optional[str] = ..., step_index: _Optional[int] = ..., occurred_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., metadata: _Optional[_Union[LogMetadata, _Mapping]] = ...) -> None: ...

class HeartbeatMetrics(_message.Message):
    __slots__ = ()
    MEMORY_BYTES_FIELD_NUMBER: _ClassVar[int]
    CPU_PERCENT_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_PAGES_FIELD_NUMBER: _ClassVar[int]
    CURRENT_STEP_FIELD_NUMBER: _ClassVar[int]
    TOTAL_STEPS_FIELD_NUMBER: _ClassVar[int]
    memory_bytes: int
    cpu_percent: float
    active_pages: int
    current_step: int
    total_steps: int
    def __init__(self, memory_bytes: _Optional[int] = ..., cpu_percent: _Optional[float] = ..., active_pages: _Optional[int] = ..., current_step: _Optional[int] = ..., total_steps: _Optional[int] = ...) -> None: ...

class HeartbeatEvent(_message.Message):
    __slots__ = ()
    RECEIVED_AT_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    received_at: _timestamp_pb2.Timestamp
    progress: int
    metrics: HeartbeatMetrics
    def __init__(self, received_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., progress: _Optional[int] = ..., metrics: _Optional[_Union[HeartbeatMetrics, _Mapping]] = ...) -> None: ...

class TelemetryMetrics(_message.Message):
    __slots__ = ()
    NETWORK_REQUESTS_FIELD_NUMBER: _ClassVar[int]
    BYTES_TRANSFERRED_FIELD_NUMBER: _ClassVar[int]
    DOM_NODES_FIELD_NUMBER: _ClassVar[int]
    JS_HEAP_BYTES_FIELD_NUMBER: _ClassVar[int]
    TTFB_MS_FIELD_NUMBER: _ClassVar[int]
    LCP_MS_FIELD_NUMBER: _ClassVar[int]
    FID_MS_FIELD_NUMBER: _ClassVar[int]
    CLS_FIELD_NUMBER: _ClassVar[int]
    network_requests: int
    bytes_transferred: int
    dom_nodes: int
    js_heap_bytes: int
    ttfb_ms: int
    lcp_ms: int
    fid_ms: int
    cls: float
    def __init__(self, network_requests: _Optional[int] = ..., bytes_transferred: _Optional[int] = ..., dom_nodes: _Optional[int] = ..., js_heap_bytes: _Optional[int] = ..., ttfb_ms: _Optional[int] = ..., lcp_ms: _Optional[int] = ..., fid_ms: _Optional[int] = ..., cls: _Optional[float] = ...) -> None: ...

class TelemetryEvent(_message.Message):
    __slots__ = ()
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    recorded_at: _timestamp_pb2.Timestamp
    metrics: TelemetryMetrics
    def __init__(self, recorded_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., metrics: _Optional[_Union[TelemetryMetrics, _Mapping]] = ...) -> None: ...
