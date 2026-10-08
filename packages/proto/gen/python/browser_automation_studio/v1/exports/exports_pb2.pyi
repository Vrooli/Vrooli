import datetime

from buf.validate import validate_pb2 as _validate_pb2
from browser_automation_studio.v1.base import geometry_pb2 as _geometry_pb2
from browser_automation_studio.v1.domain import selectors_pb2 as _selectors_pb2
from browser_automation_studio.v1.timeline import entry_pb2 as _entry_pb2
from common.v1 import types_pb2 as _types_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ReplaySpec(_message.Message):
    __slots__ = ("version", "generated_at", "execution", "theme", "cursor", "decor", "watermark", "intro_card", "outro_card", "playback", "presentation", "cursor_motion", "frames", "assets", "summary")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    THEME_FIELD_NUMBER: _ClassVar[int]
    CURSOR_FIELD_NUMBER: _ClassVar[int]
    DECOR_FIELD_NUMBER: _ClassVar[int]
    WATERMARK_FIELD_NUMBER: _ClassVar[int]
    INTRO_CARD_FIELD_NUMBER: _ClassVar[int]
    OUTRO_CARD_FIELD_NUMBER: _ClassVar[int]
    PLAYBACK_FIELD_NUMBER: _ClassVar[int]
    PRESENTATION_FIELD_NUMBER: _ClassVar[int]
    CURSOR_MOTION_FIELD_NUMBER: _ClassVar[int]
    FRAMES_FIELD_NUMBER: _ClassVar[int]
    ASSETS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    version: str
    generated_at: _timestamp_pb2.Timestamp
    execution: ReplayExecutionMetadata
    theme: ReplayTheme
    cursor: ReplayCursor
    decor: ReplayDecor
    watermark: ReplayWatermark
    intro_card: ReplayIntroCard
    outro_card: ReplayOutroCard
    playback: ReplayPlayback
    presentation: ReplayPresentation
    cursor_motion: ReplayCursorMotion
    frames: _containers.RepeatedCompositeFieldContainer[ReplayFrame]
    assets: _containers.RepeatedCompositeFieldContainer[ReplayAsset]
    summary: ReplaySummary
    def __init__(self, version: _Optional[str] = ..., generated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., execution: _Optional[_Union[ReplayExecutionMetadata, _Mapping]] = ..., theme: _Optional[_Union[ReplayTheme, _Mapping]] = ..., cursor: _Optional[_Union[ReplayCursor, _Mapping]] = ..., decor: _Optional[_Union[ReplayDecor, _Mapping]] = ..., watermark: _Optional[_Union[ReplayWatermark, _Mapping]] = ..., intro_card: _Optional[_Union[ReplayIntroCard, _Mapping]] = ..., outro_card: _Optional[_Union[ReplayOutroCard, _Mapping]] = ..., playback: _Optional[_Union[ReplayPlayback, _Mapping]] = ..., presentation: _Optional[_Union[ReplayPresentation, _Mapping]] = ..., cursor_motion: _Optional[_Union[ReplayCursorMotion, _Mapping]] = ..., frames: _Optional[_Iterable[_Union[ReplayFrame, _Mapping]]] = ..., assets: _Optional[_Iterable[_Union[ReplayAsset, _Mapping]]] = ..., summary: _Optional[_Union[ReplaySummary, _Mapping]] = ...) -> None: ...

class ReplayExecutionMetadata(_message.Message):
    __slots__ = ("execution_id", "workflow_id", "workflow_name", "status", "started_at", "completed_at", "progress", "total_duration_ms")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    workflow_id: str
    workflow_name: str
    status: str
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    progress: int
    total_duration_ms: int
    def __init__(self, execution_id: _Optional[str] = ..., workflow_id: _Optional[str] = ..., workflow_name: _Optional[str] = ..., status: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., progress: _Optional[int] = ..., total_duration_ms: _Optional[int] = ...) -> None: ...

class ReplayTheme(_message.Message):
    __slots__ = ("background_gradient", "background_pattern", "accent_color", "surface_color", "ambient_glow", "browser_chrome")
    BACKGROUND_GRADIENT_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_PATTERN_FIELD_NUMBER: _ClassVar[int]
    ACCENT_COLOR_FIELD_NUMBER: _ClassVar[int]
    SURFACE_COLOR_FIELD_NUMBER: _ClassVar[int]
    AMBIENT_GLOW_FIELD_NUMBER: _ClassVar[int]
    BROWSER_CHROME_FIELD_NUMBER: _ClassVar[int]
    background_gradient: _containers.RepeatedScalarFieldContainer[str]
    background_pattern: str
    accent_color: str
    surface_color: str
    ambient_glow: str
    browser_chrome: ReplayBrowserChrome
    def __init__(self, background_gradient: _Optional[_Iterable[str]] = ..., background_pattern: _Optional[str] = ..., accent_color: _Optional[str] = ..., surface_color: _Optional[str] = ..., ambient_glow: _Optional[str] = ..., browser_chrome: _Optional[_Union[ReplayBrowserChrome, _Mapping]] = ...) -> None: ...

class ReplayBrowserChrome(_message.Message):
    __slots__ = ("visible", "variant", "title", "show_address", "accent_color")
    VISIBLE_FIELD_NUMBER: _ClassVar[int]
    VARIANT_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SHOW_ADDRESS_FIELD_NUMBER: _ClassVar[int]
    ACCENT_COLOR_FIELD_NUMBER: _ClassVar[int]
    visible: bool
    variant: str
    title: str
    show_address: bool
    accent_color: str
    def __init__(self, visible: _Optional[bool] = ..., variant: _Optional[str] = ..., title: _Optional[str] = ..., show_address: _Optional[bool] = ..., accent_color: _Optional[str] = ...) -> None: ...

class ReplayCursor(_message.Message):
    __slots__ = ("style", "accent_color", "trail", "click_pulse", "scale", "initial_position", "click_animation")
    STYLE_FIELD_NUMBER: _ClassVar[int]
    ACCENT_COLOR_FIELD_NUMBER: _ClassVar[int]
    TRAIL_FIELD_NUMBER: _ClassVar[int]
    CLICK_PULSE_FIELD_NUMBER: _ClassVar[int]
    SCALE_FIELD_NUMBER: _ClassVar[int]
    INITIAL_POSITION_FIELD_NUMBER: _ClassVar[int]
    CLICK_ANIMATION_FIELD_NUMBER: _ClassVar[int]
    style: str
    accent_color: str
    trail: ReplayCursorTrail
    click_pulse: ReplayClickPulse
    scale: float
    initial_position: str
    click_animation: str
    def __init__(self, style: _Optional[str] = ..., accent_color: _Optional[str] = ..., trail: _Optional[_Union[ReplayCursorTrail, _Mapping]] = ..., click_pulse: _Optional[_Union[ReplayClickPulse, _Mapping]] = ..., scale: _Optional[float] = ..., initial_position: _Optional[str] = ..., click_animation: _Optional[str] = ...) -> None: ...

class ReplayDecor(_message.Message):
    __slots__ = ("chrome_theme", "background_theme", "background", "cursor_theme", "cursor_initial_position", "cursor_click_animation", "cursor_scale")
    CHROME_THEME_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_THEME_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_FIELD_NUMBER: _ClassVar[int]
    CURSOR_THEME_FIELD_NUMBER: _ClassVar[int]
    CURSOR_INITIAL_POSITION_FIELD_NUMBER: _ClassVar[int]
    CURSOR_CLICK_ANIMATION_FIELD_NUMBER: _ClassVar[int]
    CURSOR_SCALE_FIELD_NUMBER: _ClassVar[int]
    chrome_theme: str
    background_theme: str
    background: _struct_pb2.Struct
    cursor_theme: str
    cursor_initial_position: str
    cursor_click_animation: str
    cursor_scale: float
    def __init__(self, chrome_theme: _Optional[str] = ..., background_theme: _Optional[str] = ..., background: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., cursor_theme: _Optional[str] = ..., cursor_initial_position: _Optional[str] = ..., cursor_click_animation: _Optional[str] = ..., cursor_scale: _Optional[float] = ...) -> None: ...

class ReplayWatermark(_message.Message):
    __slots__ = ("enabled", "asset_id", "position", "size", "opacity", "margin")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    POSITION_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    OPACITY_FIELD_NUMBER: _ClassVar[int]
    MARGIN_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    asset_id: str
    position: str
    size: int
    opacity: int
    margin: int
    def __init__(self, enabled: _Optional[bool] = ..., asset_id: _Optional[str] = ..., position: _Optional[str] = ..., size: _Optional[int] = ..., opacity: _Optional[int] = ..., margin: _Optional[int] = ...) -> None: ...

class ReplayIntroCard(_message.Message):
    __slots__ = ("enabled", "title", "subtitle", "logo_asset_id", "background_asset_id", "background_color", "text_color", "duration_ms")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SUBTITLE_FIELD_NUMBER: _ClassVar[int]
    LOGO_ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_COLOR_FIELD_NUMBER: _ClassVar[int]
    TEXT_COLOR_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    title: str
    subtitle: str
    logo_asset_id: str
    background_asset_id: str
    background_color: str
    text_color: str
    duration_ms: int
    def __init__(self, enabled: _Optional[bool] = ..., title: _Optional[str] = ..., subtitle: _Optional[str] = ..., logo_asset_id: _Optional[str] = ..., background_asset_id: _Optional[str] = ..., background_color: _Optional[str] = ..., text_color: _Optional[str] = ..., duration_ms: _Optional[int] = ...) -> None: ...

class ReplayOutroCard(_message.Message):
    __slots__ = ("enabled", "title", "cta_text", "cta_url", "logo_asset_id", "background_asset_id", "background_color", "text_color", "duration_ms")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    CTA_TEXT_FIELD_NUMBER: _ClassVar[int]
    CTA_URL_FIELD_NUMBER: _ClassVar[int]
    LOGO_ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_COLOR_FIELD_NUMBER: _ClassVar[int]
    TEXT_COLOR_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    title: str
    cta_text: str
    cta_url: str
    logo_asset_id: str
    background_asset_id: str
    background_color: str
    text_color: str
    duration_ms: int
    def __init__(self, enabled: _Optional[bool] = ..., title: _Optional[str] = ..., cta_text: _Optional[str] = ..., cta_url: _Optional[str] = ..., logo_asset_id: _Optional[str] = ..., background_asset_id: _Optional[str] = ..., background_color: _Optional[str] = ..., text_color: _Optional[str] = ..., duration_ms: _Optional[int] = ...) -> None: ...

class ReplayPlayback(_message.Message):
    __slots__ = ("fps", "duration_ms", "frame_interval_ms", "total_frames")
    FPS_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    FRAME_INTERVAL_MS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FRAMES_FIELD_NUMBER: _ClassVar[int]
    fps: int
    duration_ms: int
    frame_interval_ms: int
    total_frames: int
    def __init__(self, fps: _Optional[int] = ..., duration_ms: _Optional[int] = ..., frame_interval_ms: _Optional[int] = ..., total_frames: _Optional[int] = ...) -> None: ...

class ReplayDimensions(_message.Message):
    __slots__ = ("width", "height")
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    width: int
    height: int
    def __init__(self, width: _Optional[int] = ..., height: _Optional[int] = ...) -> None: ...

class ReplayFrameRect(_message.Message):
    __slots__ = ("x", "y", "width", "height", "radius")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    RADIUS_FIELD_NUMBER: _ClassVar[int]
    x: int
    y: int
    width: int
    height: int
    radius: int
    def __init__(self, x: _Optional[int] = ..., y: _Optional[int] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., radius: _Optional[int] = ...) -> None: ...

class ReplayPresentation(_message.Message):
    __slots__ = ("canvas", "viewport", "browser_frame", "device_scale_factor")
    CANVAS_FIELD_NUMBER: _ClassVar[int]
    VIEWPORT_FIELD_NUMBER: _ClassVar[int]
    BROWSER_FRAME_FIELD_NUMBER: _ClassVar[int]
    DEVICE_SCALE_FACTOR_FIELD_NUMBER: _ClassVar[int]
    canvas: ReplayDimensions
    viewport: ReplayDimensions
    browser_frame: ReplayFrameRect
    device_scale_factor: float
    def __init__(self, canvas: _Optional[_Union[ReplayDimensions, _Mapping]] = ..., viewport: _Optional[_Union[ReplayDimensions, _Mapping]] = ..., browser_frame: _Optional[_Union[ReplayFrameRect, _Mapping]] = ..., device_scale_factor: _Optional[float] = ...) -> None: ...

class ReplayCursorMotion(_message.Message):
    __slots__ = ("speed_profile", "path_style", "initial_position", "click_animation", "cursor_scale")
    SPEED_PROFILE_FIELD_NUMBER: _ClassVar[int]
    PATH_STYLE_FIELD_NUMBER: _ClassVar[int]
    INITIAL_POSITION_FIELD_NUMBER: _ClassVar[int]
    CLICK_ANIMATION_FIELD_NUMBER: _ClassVar[int]
    CURSOR_SCALE_FIELD_NUMBER: _ClassVar[int]
    speed_profile: str
    path_style: str
    initial_position: str
    click_animation: str
    cursor_scale: float
    def __init__(self, speed_profile: _Optional[str] = ..., path_style: _Optional[str] = ..., initial_position: _Optional[str] = ..., click_animation: _Optional[str] = ..., cursor_scale: _Optional[float] = ...) -> None: ...

class ReplayCursorTrail(_message.Message):
    __slots__ = ("enabled", "fade_ms", "weight", "opacity")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    FADE_MS_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FIELD_NUMBER: _ClassVar[int]
    OPACITY_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    fade_ms: int
    weight: float
    opacity: float
    def __init__(self, enabled: _Optional[bool] = ..., fade_ms: _Optional[int] = ..., weight: _Optional[float] = ..., opacity: _Optional[float] = ...) -> None: ...

class ReplayClickPulse(_message.Message):
    __slots__ = ("enabled", "radius", "duration_ms", "opacity")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    RADIUS_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    OPACITY_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    radius: float
    duration_ms: int
    opacity: float
    def __init__(self, enabled: _Optional[bool] = ..., radius: _Optional[float] = ..., duration_ms: _Optional[int] = ..., opacity: _Optional[float] = ...) -> None: ...

class ReplayFrame(_message.Message):
    __slots__ = ("index", "step_index", "node_id", "step_type", "title", "status", "start_offset_ms", "duration_ms", "hold_ms", "enter", "exit", "screenshot_asset_id", "viewport", "zoom_factor", "highlight_regions", "mask_regions", "focused_element", "element_bounding_box", "normalized_focus_bounds", "normalized_element_bounds", "click_position", "normalized_click_position", "cursor_trail", "normalized_cursor_trail", "console_log_count", "network_event_count", "final_url", "error", "assertion", "resilience")
    INDEX_FIELD_NUMBER: _ClassVar[int]
    STEP_INDEX_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    STEP_TYPE_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    START_OFFSET_MS_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    HOLD_MS_FIELD_NUMBER: _ClassVar[int]
    ENTER_FIELD_NUMBER: _ClassVar[int]
    EXIT_FIELD_NUMBER: _ClassVar[int]
    SCREENSHOT_ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    VIEWPORT_FIELD_NUMBER: _ClassVar[int]
    ZOOM_FACTOR_FIELD_NUMBER: _ClassVar[int]
    HIGHLIGHT_REGIONS_FIELD_NUMBER: _ClassVar[int]
    MASK_REGIONS_FIELD_NUMBER: _ClassVar[int]
    FOCUSED_ELEMENT_FIELD_NUMBER: _ClassVar[int]
    ELEMENT_BOUNDING_BOX_FIELD_NUMBER: _ClassVar[int]
    NORMALIZED_FOCUS_BOUNDS_FIELD_NUMBER: _ClassVar[int]
    NORMALIZED_ELEMENT_BOUNDS_FIELD_NUMBER: _ClassVar[int]
    CLICK_POSITION_FIELD_NUMBER: _ClassVar[int]
    NORMALIZED_CLICK_POSITION_FIELD_NUMBER: _ClassVar[int]
    CURSOR_TRAIL_FIELD_NUMBER: _ClassVar[int]
    NORMALIZED_CURSOR_TRAIL_FIELD_NUMBER: _ClassVar[int]
    CONSOLE_LOG_COUNT_FIELD_NUMBER: _ClassVar[int]
    NETWORK_EVENT_COUNT_FIELD_NUMBER: _ClassVar[int]
    FINAL_URL_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    ASSERTION_FIELD_NUMBER: _ClassVar[int]
    RESILIENCE_FIELD_NUMBER: _ClassVar[int]
    index: int
    step_index: int
    node_id: str
    step_type: str
    title: str
    status: str
    start_offset_ms: int
    duration_ms: int
    hold_ms: int
    enter: ReplayTransition
    exit: ReplayTransition
    screenshot_asset_id: str
    viewport: ReplayDimensions
    zoom_factor: float
    highlight_regions: _containers.RepeatedCompositeFieldContainer[_selectors_pb2.HighlightRegion]
    mask_regions: _containers.RepeatedCompositeFieldContainer[_selectors_pb2.MaskRegion]
    focused_element: _entry_pb2.ElementFocus
    element_bounding_box: _geometry_pb2.BoundingBox
    normalized_focus_bounds: ReplayNormalizedRect
    normalized_element_bounds: ReplayNormalizedRect
    click_position: _geometry_pb2.Point
    normalized_click_position: ReplayNormalizedPoint
    cursor_trail: _containers.RepeatedCompositeFieldContainer[_geometry_pb2.Point]
    normalized_cursor_trail: _containers.RepeatedCompositeFieldContainer[ReplayNormalizedPoint]
    console_log_count: int
    network_event_count: int
    final_url: str
    error: str
    assertion: ReplayAssertionOutcome
    resilience: ReplayResilience
    def __init__(self, index: _Optional[int] = ..., step_index: _Optional[int] = ..., node_id: _Optional[str] = ..., step_type: _Optional[str] = ..., title: _Optional[str] = ..., status: _Optional[str] = ..., start_offset_ms: _Optional[int] = ..., duration_ms: _Optional[int] = ..., hold_ms: _Optional[int] = ..., enter: _Optional[_Union[ReplayTransition, _Mapping]] = ..., exit: _Optional[_Union[ReplayTransition, _Mapping]] = ..., screenshot_asset_id: _Optional[str] = ..., viewport: _Optional[_Union[ReplayDimensions, _Mapping]] = ..., zoom_factor: _Optional[float] = ..., highlight_regions: _Optional[_Iterable[_Union[_selectors_pb2.HighlightRegion, _Mapping]]] = ..., mask_regions: _Optional[_Iterable[_Union[_selectors_pb2.MaskRegion, _Mapping]]] = ..., focused_element: _Optional[_Union[_entry_pb2.ElementFocus, _Mapping]] = ..., element_bounding_box: _Optional[_Union[_geometry_pb2.BoundingBox, _Mapping]] = ..., normalized_focus_bounds: _Optional[_Union[ReplayNormalizedRect, _Mapping]] = ..., normalized_element_bounds: _Optional[_Union[ReplayNormalizedRect, _Mapping]] = ..., click_position: _Optional[_Union[_geometry_pb2.Point, _Mapping]] = ..., normalized_click_position: _Optional[_Union[ReplayNormalizedPoint, _Mapping]] = ..., cursor_trail: _Optional[_Iterable[_Union[_geometry_pb2.Point, _Mapping]]] = ..., normalized_cursor_trail: _Optional[_Iterable[_Union[ReplayNormalizedPoint, _Mapping]]] = ..., console_log_count: _Optional[int] = ..., network_event_count: _Optional[int] = ..., final_url: _Optional[str] = ..., error: _Optional[str] = ..., assertion: _Optional[_Union[ReplayAssertionOutcome, _Mapping]] = ..., resilience: _Optional[_Union[ReplayResilience, _Mapping]] = ...) -> None: ...

class ReplayResilience(_message.Message):
    __slots__ = ("attempt", "max_attempts", "configured_retries", "delay_ms", "backoff_factor", "history")
    ATTEMPT_FIELD_NUMBER: _ClassVar[int]
    MAX_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    CONFIGURED_RETRIES_FIELD_NUMBER: _ClassVar[int]
    DELAY_MS_FIELD_NUMBER: _ClassVar[int]
    BACKOFF_FACTOR_FIELD_NUMBER: _ClassVar[int]
    HISTORY_FIELD_NUMBER: _ClassVar[int]
    attempt: int
    max_attempts: int
    configured_retries: int
    delay_ms: int
    backoff_factor: float
    history: _containers.RepeatedCompositeFieldContainer[RetryHistoryEntry]
    def __init__(self, attempt: _Optional[int] = ..., max_attempts: _Optional[int] = ..., configured_retries: _Optional[int] = ..., delay_ms: _Optional[int] = ..., backoff_factor: _Optional[float] = ..., history: _Optional[_Iterable[_Union[RetryHistoryEntry, _Mapping]]] = ...) -> None: ...

class RetryHistoryEntry(_message.Message):
    __slots__ = ("attempt", "success", "duration_ms", "call_duration_ms", "error")
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

class ReplayAssertionOutcome(_message.Message):
    __slots__ = ("mode", "selector", "expected", "actual", "success", "negated", "case_sensitive", "message")
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

class ReplayTransition(_message.Message):
    __slots__ = ("type", "duration_ms", "easing")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    EASING_FIELD_NUMBER: _ClassVar[int]
    type: str
    duration_ms: int
    easing: str
    def __init__(self, type: _Optional[str] = ..., duration_ms: _Optional[int] = ..., easing: _Optional[str] = ...) -> None: ...

class ReplayAsset(_message.Message):
    __slots__ = ("id", "type", "source", "thumbnail", "width", "height", "size_bytes")
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    THUMBNAIL_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    id: str
    type: str
    source: str
    thumbnail: str
    width: int
    height: int
    size_bytes: int
    def __init__(self, id: _Optional[str] = ..., type: _Optional[str] = ..., source: _Optional[str] = ..., thumbnail: _Optional[str] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., size_bytes: _Optional[int] = ...) -> None: ...

class ReplaySummary(_message.Message):
    __slots__ = ("frame_count", "screenshot_count", "total_duration_ms", "max_frame_duration_ms")
    FRAME_COUNT_FIELD_NUMBER: _ClassVar[int]
    SCREENSHOT_COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    MAX_FRAME_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    frame_count: int
    screenshot_count: int
    total_duration_ms: int
    max_frame_duration_ms: int
    def __init__(self, frame_count: _Optional[int] = ..., screenshot_count: _Optional[int] = ..., total_duration_ms: _Optional[int] = ..., max_frame_duration_ms: _Optional[int] = ...) -> None: ...

class ReplayNormalizedPoint(_message.Message):
    __slots__ = ("x", "y")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    x: float
    y: float
    def __init__(self, x: _Optional[float] = ..., y: _Optional[float] = ...) -> None: ...

class ReplayNormalizedRect(_message.Message):
    __slots__ = ("x", "y", "width", "height")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    x: float
    y: float
    width: float
    height: float
    def __init__(self, x: _Optional[float] = ..., y: _Optional[float] = ..., width: _Optional[float] = ..., height: _Optional[float] = ...) -> None: ...

class Export(_message.Message):
    __slots__ = ("id", "execution_id", "workflow_id", "name", "format", "settings", "storage_url", "thumbnail_url", "file_size_bytes", "duration_ms", "frame_count", "ai_caption", "ai_caption_generated_at", "status", "error", "created_at", "updated_at", "workflow_name", "execution_date")
    ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    STORAGE_URL_FIELD_NUMBER: _ClassVar[int]
    THUMBNAIL_URL_FIELD_NUMBER: _ClassVar[int]
    FILE_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    FRAME_COUNT_FIELD_NUMBER: _ClassVar[int]
    AI_CAPTION_FIELD_NUMBER: _ClassVar[int]
    AI_CAPTION_GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_NAME_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_DATE_FIELD_NUMBER: _ClassVar[int]
    id: str
    execution_id: str
    workflow_id: str
    name: str
    format: str
    settings: _struct_pb2.Struct
    storage_url: str
    thumbnail_url: str
    file_size_bytes: int
    duration_ms: int
    frame_count: int
    ai_caption: str
    ai_caption_generated_at: _timestamp_pb2.Timestamp
    status: str
    error: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    workflow_name: str
    execution_date: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., execution_id: _Optional[str] = ..., workflow_id: _Optional[str] = ..., name: _Optional[str] = ..., format: _Optional[str] = ..., settings: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., storage_url: _Optional[str] = ..., thumbnail_url: _Optional[str] = ..., file_size_bytes: _Optional[int] = ..., duration_ms: _Optional[int] = ..., frame_count: _Optional[int] = ..., ai_caption: _Optional[str] = ..., ai_caption_generated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., status: _Optional[str] = ..., error: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., workflow_name: _Optional[str] = ..., execution_date: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListExportsRequest(_message.Message):
    __slots__ = ("execution_id", "workflow_id", "limit", "offset")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    workflow_id: str
    limit: int
    offset: int
    def __init__(self, execution_id: _Optional[str] = ..., workflow_id: _Optional[str] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListExportsResponse(_message.Message):
    __slots__ = ("exports", "total")
    EXPORTS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    exports: _containers.RepeatedCompositeFieldContainer[Export]
    total: int
    def __init__(self, exports: _Optional[_Iterable[_Union[Export, _Mapping]]] = ..., total: _Optional[int] = ...) -> None: ...

class CreateExportRequest(_message.Message):
    __slots__ = ("execution_id", "workflow_id", "name", "format", "settings", "storage_url", "thumbnail_url", "file_size_bytes", "duration_ms", "frame_count", "status")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    STORAGE_URL_FIELD_NUMBER: _ClassVar[int]
    THUMBNAIL_URL_FIELD_NUMBER: _ClassVar[int]
    FILE_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    FRAME_COUNT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    workflow_id: str
    name: str
    format: str
    settings: _struct_pb2.Struct
    storage_url: str
    thumbnail_url: str
    file_size_bytes: int
    duration_ms: int
    frame_count: int
    status: str
    def __init__(self, execution_id: _Optional[str] = ..., workflow_id: _Optional[str] = ..., name: _Optional[str] = ..., format: _Optional[str] = ..., settings: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., storage_url: _Optional[str] = ..., thumbnail_url: _Optional[str] = ..., file_size_bytes: _Optional[int] = ..., duration_ms: _Optional[int] = ..., frame_count: _Optional[int] = ..., status: _Optional[str] = ...) -> None: ...

class CreateExportResponse(_message.Message):
    __slots__ = ("export_id", "status", "export")
    EXPORT_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EXPORT_FIELD_NUMBER: _ClassVar[int]
    export_id: str
    status: str
    export: Export
    def __init__(self, export_id: _Optional[str] = ..., status: _Optional[str] = ..., export: _Optional[_Union[Export, _Mapping]] = ...) -> None: ...

class GetExportRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetExportResponse(_message.Message):
    __slots__ = ("export",)
    EXPORT_FIELD_NUMBER: _ClassVar[int]
    export: Export
    def __init__(self, export: _Optional[_Union[Export, _Mapping]] = ...) -> None: ...

class UpdateExportRequest(_message.Message):
    __slots__ = ("id", "name", "settings", "storage_url", "thumbnail_url", "file_size_bytes", "duration_ms", "frame_count", "ai_caption", "status", "error")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    STORAGE_URL_FIELD_NUMBER: _ClassVar[int]
    THUMBNAIL_URL_FIELD_NUMBER: _ClassVar[int]
    FILE_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    FRAME_COUNT_FIELD_NUMBER: _ClassVar[int]
    AI_CAPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    settings: _struct_pb2.Struct
    storage_url: str
    thumbnail_url: str
    file_size_bytes: int
    duration_ms: int
    frame_count: int
    ai_caption: str
    status: str
    error: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., settings: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., storage_url: _Optional[str] = ..., thumbnail_url: _Optional[str] = ..., file_size_bytes: _Optional[int] = ..., duration_ms: _Optional[int] = ..., frame_count: _Optional[int] = ..., ai_caption: _Optional[str] = ..., status: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...

class UpdateExportResponse(_message.Message):
    __slots__ = ("export_id", "status", "export")
    EXPORT_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EXPORT_FIELD_NUMBER: _ClassVar[int]
    export_id: str
    status: str
    export: Export
    def __init__(self, export_id: _Optional[str] = ..., status: _Optional[str] = ..., export: _Optional[_Union[Export, _Mapping]] = ...) -> None: ...

class DeleteExportRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteExportResponse(_message.Message):
    __slots__ = ("export_id", "status")
    EXPORT_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    export_id: str
    status: str
    def __init__(self, export_id: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class GetExportStatusRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetExportStatusResponse(_message.Message):
    __slots__ = ("export_id", "execution_id", "status", "format", "name", "storage_url", "file_size_bytes", "error")
    EXPORT_ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    STORAGE_URL_FIELD_NUMBER: _ClassVar[int]
    FILE_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    export_id: str
    execution_id: str
    status: str
    format: str
    name: str
    storage_url: str
    file_size_bytes: int
    error: str
    def __init__(self, export_id: _Optional[str] = ..., execution_id: _Optional[str] = ..., status: _Optional[str] = ..., format: _Optional[str] = ..., name: _Optional[str] = ..., storage_url: _Optional[str] = ..., file_size_bytes: _Optional[int] = ..., error: _Optional[str] = ...) -> None: ...

class GenerateExportCaptionRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GenerateExportCaptionResponse(_message.Message):
    __slots__ = ("export_id", "caption", "export")
    EXPORT_ID_FIELD_NUMBER: _ClassVar[int]
    CAPTION_FIELD_NUMBER: _ClassVar[int]
    EXPORT_FIELD_NUMBER: _ClassVar[int]
    export_id: str
    caption: str
    export: Export
    def __init__(self, export_id: _Optional[str] = ..., caption: _Optional[str] = ..., export: _Optional[_Union[Export, _Mapping]] = ...) -> None: ...

class RevealExportRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RevealExportResponse(_message.Message):
    __slots__ = ("export_id", "path", "status")
    EXPORT_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    export_id: str
    path: str
    status: str
    def __init__(self, export_id: _Optional[str] = ..., path: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class OpenExportFolderRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class OpenExportFolderResponse(_message.Message):
    __slots__ = ("export_id", "folder", "status")
    EXPORT_ID_FIELD_NUMBER: _ClassVar[int]
    FOLDER_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    export_id: str
    folder: str
    status: str
    def __init__(self, export_id: _Optional[str] = ..., folder: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...
