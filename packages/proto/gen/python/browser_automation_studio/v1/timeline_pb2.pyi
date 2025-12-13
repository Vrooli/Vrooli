import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from common.v1 import types_pb2 as _types_pb2
from browser_automation_studio.v1 import shared_pb2 as _shared_pb2
from browser_automation_studio.v1 import unified_pb2 as _unified_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ExecutionTimeline(_message.Message):
    __slots__ = ()
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    FRAMES_FIELD_NUMBER: _ClassVar[int]
    LOGS_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    workflow_id: str
    status: _shared_pb2.ExecutionStatus
    progress: int
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    frames: _containers.RepeatedCompositeFieldContainer[TimelineFrame]
    logs: _containers.RepeatedCompositeFieldContainer[TimelineLog]
    def __init__(self, execution_id: _Optional[str] = ..., workflow_id: _Optional[str] = ..., status: _Optional[_Union[_shared_pb2.ExecutionStatus, str]] = ..., progress: _Optional[int] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., frames: _Optional[_Iterable[_Union[TimelineFrame, _Mapping]]] = ..., logs: _Optional[_Iterable[_Union[TimelineLog, _Mapping]]] = ...) -> None: ...

class TimelineFrame(_message.Message):
    __slots__ = ()
    STEP_INDEX_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_TYPE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    FINAL_URL_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    CONSOLE_LOG_COUNT_FIELD_NUMBER: _ClassVar[int]
    NETWORK_EVENT_COUNT_FIELD_NUMBER: _ClassVar[int]
    HIGHLIGHT_REGIONS_FIELD_NUMBER: _ClassVar[int]
    MASK_REGIONS_FIELD_NUMBER: _ClassVar[int]
    FOCUSED_ELEMENT_FIELD_NUMBER: _ClassVar[int]
    ELEMENT_BOUNDING_BOX_FIELD_NUMBER: _ClassVar[int]
    CLICK_POSITION_FIELD_NUMBER: _ClassVar[int]
    CURSOR_TRAIL_FIELD_NUMBER: _ClassVar[int]
    ZOOM_FACTOR_FIELD_NUMBER: _ClassVar[int]
    SCREENSHOT_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    ASSERTION_FIELD_NUMBER: _ClassVar[int]
    RETRY_ATTEMPT_FIELD_NUMBER: _ClassVar[int]
    RETRY_MAX_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    RETRY_CONFIGURED_FIELD_NUMBER: _ClassVar[int]
    RETRY_DELAY_MS_FIELD_NUMBER: _ClassVar[int]
    RETRY_BACKOFF_FACTOR_FIELD_NUMBER: _ClassVar[int]
    RETRY_HISTORY_FIELD_NUMBER: _ClassVar[int]
    DOM_SNAPSHOT_PREVIEW_FIELD_NUMBER: _ClassVar[int]
    DOM_SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    EXTRACTED_DATA_PREVIEW_FIELD_NUMBER: _ClassVar[int]
    step_index: int
    node_id: str
    action_type: _unified_pb2.ActionType
    status: _shared_pb2.StepStatus
    success: bool
    duration_ms: int
    total_duration_ms: int
    progress: int
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    final_url: str
    error: str
    console_log_count: int
    network_event_count: int
    highlight_regions: _containers.RepeatedCompositeFieldContainer[_unified_pb2.HighlightRegion]
    mask_regions: _containers.RepeatedCompositeFieldContainer[_unified_pb2.MaskRegion]
    focused_element: ElementFocus
    element_bounding_box: _unified_pb2.BoundingBox
    click_position: _unified_pb2.Point
    cursor_trail: _containers.RepeatedCompositeFieldContainer[_unified_pb2.Point]
    zoom_factor: float
    screenshot: _unified_pb2.TimelineScreenshot
    artifacts: _containers.RepeatedCompositeFieldContainer[TimelineArtifact]
    assertion: _unified_pb2.AssertionResultProto
    retry_attempt: int
    retry_max_attempts: int
    retry_configured: bool
    retry_delay_ms: int
    retry_backoff_factor: float
    retry_history: _containers.RepeatedCompositeFieldContainer[RetryHistoryEntry]
    dom_snapshot_preview: str
    dom_snapshot: TimelineArtifact
    extracted_data_preview: _types_pb2.JsonValue
    def __init__(self, step_index: _Optional[int] = ..., node_id: _Optional[str] = ..., action_type: _Optional[_Union[_unified_pb2.ActionType, str]] = ..., status: _Optional[_Union[_shared_pb2.StepStatus, str]] = ..., success: _Optional[bool] = ..., duration_ms: _Optional[int] = ..., total_duration_ms: _Optional[int] = ..., progress: _Optional[int] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., final_url: _Optional[str] = ..., error: _Optional[str] = ..., console_log_count: _Optional[int] = ..., network_event_count: _Optional[int] = ..., highlight_regions: _Optional[_Iterable[_Union[_unified_pb2.HighlightRegion, _Mapping]]] = ..., mask_regions: _Optional[_Iterable[_Union[_unified_pb2.MaskRegion, _Mapping]]] = ..., focused_element: _Optional[_Union[ElementFocus, _Mapping]] = ..., element_bounding_box: _Optional[_Union[_unified_pb2.BoundingBox, _Mapping]] = ..., click_position: _Optional[_Union[_unified_pb2.Point, _Mapping]] = ..., cursor_trail: _Optional[_Iterable[_Union[_unified_pb2.Point, _Mapping]]] = ..., zoom_factor: _Optional[float] = ..., screenshot: _Optional[_Union[_unified_pb2.TimelineScreenshot, _Mapping]] = ..., artifacts: _Optional[_Iterable[_Union[TimelineArtifact, _Mapping]]] = ..., assertion: _Optional[_Union[_unified_pb2.AssertionResultProto, _Mapping]] = ..., retry_attempt: _Optional[int] = ..., retry_max_attempts: _Optional[int] = ..., retry_configured: _Optional[bool] = ..., retry_delay_ms: _Optional[int] = ..., retry_backoff_factor: _Optional[float] = ..., retry_history: _Optional[_Iterable[_Union[RetryHistoryEntry, _Mapping]]] = ..., dom_snapshot_preview: _Optional[str] = ..., dom_snapshot: _Optional[_Union[TimelineArtifact, _Mapping]] = ..., extracted_data_preview: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...

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

class RetryHistoryEntry(_message.Message):
    __slots__ = ()
    ATTEMPT_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    CALL_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    attempt: int
    success: bool
    duration_ms: int
    call_duration_ms: int
    error: str
    def __init__(self, attempt: _Optional[int] = ..., success: _Optional[bool] = ..., duration_ms: _Optional[int] = ..., call_duration_ms: _Optional[int] = ..., error: _Optional[str] = ...) -> None: ...

class ElementFocus(_message.Message):
    __slots__ = ()
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    BOUNDING_BOX_FIELD_NUMBER: _ClassVar[int]
    selector: str
    bounding_box: _unified_pb2.BoundingBox
    def __init__(self, selector: _Optional[str] = ..., bounding_box: _Optional[_Union[_unified_pb2.BoundingBox, _Mapping]] = ...) -> None: ...

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
