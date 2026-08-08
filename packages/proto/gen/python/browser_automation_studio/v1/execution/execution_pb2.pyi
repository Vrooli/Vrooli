import datetime

from browser_automation_studio.v1.actions import action_pb2 as _action_pb2
from browser_automation_studio.v1.base import browser_profile_pb2 as _browser_profile_pb2
from browser_automation_studio.v1.base import shared_pb2 as _shared_pb2
from browser_automation_studio.v1.domain import telemetry_pb2 as _telemetry_pb2
from browser_automation_studio.v1.workflows import definition_pb2 as _definition_pb2
from buf.validate import validate_pb2 as _validate_pb2
from common.v1 import types_pb2 as _types_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ArtifactCollectionConfig(_message.Message):
    __slots__ = ("profile", "collect_screenshots", "collect_dom_snapshots", "collect_console_logs", "collect_network_events", "collect_extracted_data", "collect_assertions", "collect_cursor_trails", "collect_telemetry", "max_screenshot_bytes", "max_dom_snapshot_bytes", "max_console_entry_bytes", "max_network_preview_bytes")
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    COLLECT_SCREENSHOTS_FIELD_NUMBER: _ClassVar[int]
    COLLECT_DOM_SNAPSHOTS_FIELD_NUMBER: _ClassVar[int]
    COLLECT_CONSOLE_LOGS_FIELD_NUMBER: _ClassVar[int]
    COLLECT_NETWORK_EVENTS_FIELD_NUMBER: _ClassVar[int]
    COLLECT_EXTRACTED_DATA_FIELD_NUMBER: _ClassVar[int]
    COLLECT_ASSERTIONS_FIELD_NUMBER: _ClassVar[int]
    COLLECT_CURSOR_TRAILS_FIELD_NUMBER: _ClassVar[int]
    COLLECT_TELEMETRY_FIELD_NUMBER: _ClassVar[int]
    MAX_SCREENSHOT_BYTES_FIELD_NUMBER: _ClassVar[int]
    MAX_DOM_SNAPSHOT_BYTES_FIELD_NUMBER: _ClassVar[int]
    MAX_CONSOLE_ENTRY_BYTES_FIELD_NUMBER: _ClassVar[int]
    MAX_NETWORK_PREVIEW_BYTES_FIELD_NUMBER: _ClassVar[int]
    profile: str
    collect_screenshots: bool
    collect_dom_snapshots: bool
    collect_console_logs: bool
    collect_network_events: bool
    collect_extracted_data: bool
    collect_assertions: bool
    collect_cursor_trails: bool
    collect_telemetry: bool
    max_screenshot_bytes: int
    max_dom_snapshot_bytes: int
    max_console_entry_bytes: int
    max_network_preview_bytes: int
    def __init__(self, profile: _Optional[str] = ..., collect_screenshots: _Optional[bool] = ..., collect_dom_snapshots: _Optional[bool] = ..., collect_console_logs: _Optional[bool] = ..., collect_network_events: _Optional[bool] = ..., collect_extracted_data: _Optional[bool] = ..., collect_assertions: _Optional[bool] = ..., collect_cursor_trails: _Optional[bool] = ..., collect_telemetry: _Optional[bool] = ..., max_screenshot_bytes: _Optional[int] = ..., max_dom_snapshot_bytes: _Optional[int] = ..., max_console_entry_bytes: _Optional[int] = ..., max_network_preview_bytes: _Optional[int] = ...) -> None: ...

class ExecutionParameters(_message.Message):
    __slots__ = ("start_url", "variables", "viewport_width", "viewport_height", "headless", "user_agent", "locale", "timeout_ms", "project_root", "initial_params", "initial_store", "env", "artifact_config", "browser_profile", "session_profile_id", "save_session_profile_id", "restore_tabs", "navigation_wait_until", "continue_on_error")
    class VariablesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class InitialParamsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    class InitialStoreEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    class EnvEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    START_URL_FIELD_NUMBER: _ClassVar[int]
    VARIABLES_FIELD_NUMBER: _ClassVar[int]
    VIEWPORT_WIDTH_FIELD_NUMBER: _ClassVar[int]
    VIEWPORT_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    HEADLESS_FIELD_NUMBER: _ClassVar[int]
    USER_AGENT_FIELD_NUMBER: _ClassVar[int]
    LOCALE_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    PROJECT_ROOT_FIELD_NUMBER: _ClassVar[int]
    INITIAL_PARAMS_FIELD_NUMBER: _ClassVar[int]
    INITIAL_STORE_FIELD_NUMBER: _ClassVar[int]
    ENV_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_CONFIG_FIELD_NUMBER: _ClassVar[int]
    BROWSER_PROFILE_FIELD_NUMBER: _ClassVar[int]
    SESSION_PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    SAVE_SESSION_PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    RESTORE_TABS_FIELD_NUMBER: _ClassVar[int]
    NAVIGATION_WAIT_UNTIL_FIELD_NUMBER: _ClassVar[int]
    CONTINUE_ON_ERROR_FIELD_NUMBER: _ClassVar[int]
    start_url: str
    variables: _containers.ScalarMap[str, str]
    viewport_width: int
    viewport_height: int
    headless: bool
    user_agent: str
    locale: str
    timeout_ms: int
    project_root: str
    initial_params: _containers.MessageMap[str, _types_pb2.JsonValue]
    initial_store: _containers.MessageMap[str, _types_pb2.JsonValue]
    env: _containers.MessageMap[str, _types_pb2.JsonValue]
    artifact_config: ArtifactCollectionConfig
    browser_profile: _browser_profile_pb2.BrowserProfile
    session_profile_id: str
    save_session_profile_id: str
    restore_tabs: bool
    navigation_wait_until: _action_pb2.NavigateWaitEvent
    continue_on_error: bool
    def __init__(self, start_url: _Optional[str] = ..., variables: _Optional[_Mapping[str, str]] = ..., viewport_width: _Optional[int] = ..., viewport_height: _Optional[int] = ..., headless: _Optional[bool] = ..., user_agent: _Optional[str] = ..., locale: _Optional[str] = ..., timeout_ms: _Optional[int] = ..., project_root: _Optional[str] = ..., initial_params: _Optional[_Mapping[str, _types_pb2.JsonValue]] = ..., initial_store: _Optional[_Mapping[str, _types_pb2.JsonValue]] = ..., env: _Optional[_Mapping[str, _types_pb2.JsonValue]] = ..., artifact_config: _Optional[_Union[ArtifactCollectionConfig, _Mapping]] = ..., browser_profile: _Optional[_Union[_browser_profile_pb2.BrowserProfile, _Mapping]] = ..., session_profile_id: _Optional[str] = ..., save_session_profile_id: _Optional[str] = ..., restore_tabs: _Optional[bool] = ..., navigation_wait_until: _Optional[_Union[_action_pb2.NavigateWaitEvent, str]] = ..., continue_on_error: _Optional[bool] = ...) -> None: ...

class ExecutionResult(_message.Message):
    __slots__ = ("success", "steps_executed", "steps_failed", "final_url", "error", "error_code", "extracted_data", "screenshot_artifacts")
    class ExtractedDataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    class ScreenshotArtifactsEntry(_message.Message):
        __slots__ = ("key", "value")
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
    __slots__ = ("user_id", "client_id", "schedule_id", "webhook_id", "external_request_id", "source_ip", "user_agent")
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
    __slots__ = ("execution_id", "workflow_id", "workflow_version", "status", "trigger_type", "started_at", "completed_at", "last_heartbeat_at", "error", "progress", "current_step", "created_at", "updated_at", "parameters", "result", "trigger_metadata", "trace_id", "correlation_id", "request_id", "resumed_from")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_VERSION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_TYPE_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_HEARTBEAT_AT_FIELD_NUMBER: _ClassVar[int]
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
    RESUMED_FROM_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    workflow_id: str
    workflow_version: int
    status: _shared_pb2.ExecutionStatus
    trigger_type: _shared_pb2.TriggerType
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    last_heartbeat_at: _timestamp_pb2.Timestamp
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
    resumed_from: str
    def __init__(self, execution_id: _Optional[str] = ..., workflow_id: _Optional[str] = ..., workflow_version: _Optional[int] = ..., status: _Optional[_Union[_shared_pb2.ExecutionStatus, str]] = ..., trigger_type: _Optional[_Union[_shared_pb2.TriggerType, str]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_heartbeat_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., error: _Optional[str] = ..., progress: _Optional[int] = ..., current_step: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., parameters: _Optional[_Union[ExecutionParameters, _Mapping]] = ..., result: _Optional[_Union[ExecutionResult, _Mapping]] = ..., trigger_metadata: _Optional[_Union[TriggerMetadata, _Mapping]] = ..., trace_id: _Optional[str] = ..., correlation_id: _Optional[str] = ..., request_id: _Optional[str] = ..., resumed_from: _Optional[str] = ...) -> None: ...

class ExecuteAdhocRequest(_message.Message):
    __slots__ = ("flow_definition", "wait_for_completion", "metadata", "parameters", "options")
    FLOW_DEFINITION_FIELD_NUMBER: _ClassVar[int]
    WAIT_FOR_COMPLETION_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    flow_definition: _definition_pb2.WorkflowDefinitionV2
    wait_for_completion: bool
    metadata: ExecutionMetadata
    parameters: ExecutionParameters
    options: ExecuteWorkflowOptions
    def __init__(self, flow_definition: _Optional[_Union[_definition_pb2.WorkflowDefinitionV2, _Mapping]] = ..., wait_for_completion: _Optional[bool] = ..., metadata: _Optional[_Union[ExecutionMetadata, _Mapping]] = ..., parameters: _Optional[_Union[ExecutionParameters, _Mapping]] = ..., options: _Optional[_Union[ExecuteWorkflowOptions, _Mapping]] = ...) -> None: ...

class ExecuteWorkflowOptions(_message.Message):
    __slots__ = ("requires_video", "requires_trace", "requires_har", "frame_streaming", "frame_streaming_quality", "frame_streaming_fps", "seed_mode", "seed_scenario", "electron_target", "validation_context")
    REQUIRES_VIDEO_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_TRACE_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_HAR_FIELD_NUMBER: _ClassVar[int]
    FRAME_STREAMING_FIELD_NUMBER: _ClassVar[int]
    FRAME_STREAMING_QUALITY_FIELD_NUMBER: _ClassVar[int]
    FRAME_STREAMING_FPS_FIELD_NUMBER: _ClassVar[int]
    SEED_MODE_FIELD_NUMBER: _ClassVar[int]
    SEED_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ELECTRON_TARGET_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    requires_video: bool
    requires_trace: bool
    requires_har: bool
    frame_streaming: bool
    frame_streaming_quality: int
    frame_streaming_fps: int
    seed_mode: str
    seed_scenario: str
    electron_target: ElectronTarget
    validation_context: ValidationContext
    def __init__(self, requires_video: _Optional[bool] = ..., requires_trace: _Optional[bool] = ..., requires_har: _Optional[bool] = ..., frame_streaming: _Optional[bool] = ..., frame_streaming_quality: _Optional[int] = ..., frame_streaming_fps: _Optional[int] = ..., seed_mode: _Optional[str] = ..., seed_scenario: _Optional[str] = ..., electron_target: _Optional[_Union[ElectronTarget, _Mapping]] = ..., validation_context: _Optional[_Union[ValidationContext, _Mapping]] = ...) -> None: ...

class ElectronTarget(_message.Message):
    __slots__ = ("target_id", "cdp_endpoint", "renderer_id", "renderer_url", "renderer_title", "scenario_name", "artifact_digest", "context_id", "cdp_transport")
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    CDP_ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    RENDERER_ID_FIELD_NUMBER: _ClassVar[int]
    RENDERER_URL_FIELD_NUMBER: _ClassVar[int]
    RENDERER_TITLE_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_ID_FIELD_NUMBER: _ClassVar[int]
    CDP_TRANSPORT_FIELD_NUMBER: _ClassVar[int]
    target_id: str
    cdp_endpoint: str
    renderer_id: str
    renderer_url: str
    renderer_title: str
    scenario_name: str
    artifact_digest: str
    context_id: str
    cdp_transport: str
    def __init__(self, target_id: _Optional[str] = ..., cdp_endpoint: _Optional[str] = ..., renderer_id: _Optional[str] = ..., renderer_url: _Optional[str] = ..., renderer_title: _Optional[str] = ..., scenario_name: _Optional[str] = ..., artifact_digest: _Optional[str] = ..., context_id: _Optional[str] = ..., cdp_transport: _Optional[str] = ...) -> None: ...

class ValidationContext(_message.Message):
    __slots__ = ("context_id", "scenario_name", "artifact_digest", "target_id", "workflow_id", "profile_id", "isolation_lease_id")
    CONTEXT_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    ISOLATION_LEASE_ID_FIELD_NUMBER: _ClassVar[int]
    context_id: str
    scenario_name: str
    artifact_digest: str
    target_id: str
    workflow_id: str
    profile_id: str
    isolation_lease_id: str
    def __init__(self, context_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., artifact_digest: _Optional[str] = ..., target_id: _Optional[str] = ..., workflow_id: _Optional[str] = ..., profile_id: _Optional[str] = ..., isolation_lease_id: _Optional[str] = ...) -> None: ...

class ExecutionMetadata(_message.Message):
    __slots__ = ("name", "description")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class ExecuteAdhocResponse(_message.Message):
    __slots__ = ("execution_id", "status", "workflow_id", "message", "completed_at", "error")
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
    __slots__ = ("screenshot", "step_index", "node_id", "step_label", "timestamp")
    SCREENSHOT_FIELD_NUMBER: _ClassVar[int]
    STEP_INDEX_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    STEP_LABEL_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    screenshot: _telemetry_pb2.TimelineScreenshot
    step_index: int
    node_id: str
    step_label: str
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, screenshot: _Optional[_Union[_telemetry_pb2.TimelineScreenshot, _Mapping]] = ..., step_index: _Optional[int] = ..., node_id: _Optional[str] = ..., step_label: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetScreenshotsResponse(_message.Message):
    __slots__ = ("execution_id", "screenshots", "total")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    SCREENSHOTS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    screenshots: _containers.RepeatedCompositeFieldContainer[ExecutionScreenshot]
    total: int
    def __init__(self, execution_id: _Optional[str] = ..., screenshots: _Optional[_Iterable[_Union[ExecutionScreenshot, _Mapping]]] = ..., total: _Optional[int] = ...) -> None: ...

class ExecutionExportPreview(_message.Message):
    __slots__ = ("execution_id", "spec_id", "status", "message", "captured_frame_count", "available_asset_count", "total_duration_ms", "package")
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

class ExecutorMetrics(_message.Message):
    __slots__ = ("memory_bytes", "cpu_percent", "active_pages")
    MEMORY_BYTES_FIELD_NUMBER: _ClassVar[int]
    CPU_PERCENT_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_PAGES_FIELD_NUMBER: _ClassVar[int]
    memory_bytes: int
    cpu_percent: float
    active_pages: int
    def __init__(self, memory_bytes: _Optional[int] = ..., cpu_percent: _Optional[float] = ..., active_pages: _Optional[int] = ...) -> None: ...

class PerformanceMetrics(_message.Message):
    __slots__ = ("network_requests", "bytes_transferred", "dom_nodes", "js_heap_bytes", "ttfb_ms", "lcp_ms", "fid_ms", "cls")
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
