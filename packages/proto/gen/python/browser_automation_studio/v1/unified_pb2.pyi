import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from common.v1 import types_pb2 as _types_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ActionType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ACTION_TYPE_UNSPECIFIED: _ClassVar[ActionType]
    ACTION_TYPE_NAVIGATE: _ClassVar[ActionType]
    ACTION_TYPE_CLICK: _ClassVar[ActionType]
    ACTION_TYPE_INPUT: _ClassVar[ActionType]
    ACTION_TYPE_WAIT: _ClassVar[ActionType]
    ACTION_TYPE_ASSERT: _ClassVar[ActionType]
    ACTION_TYPE_SCROLL: _ClassVar[ActionType]
    ACTION_TYPE_SELECT: _ClassVar[ActionType]
    ACTION_TYPE_EVALUATE: _ClassVar[ActionType]
    ACTION_TYPE_KEYBOARD: _ClassVar[ActionType]
    ACTION_TYPE_HOVER: _ClassVar[ActionType]
    ACTION_TYPE_SCREENSHOT: _ClassVar[ActionType]
    ACTION_TYPE_FOCUS: _ClassVar[ActionType]
    ACTION_TYPE_BLUR: _ClassVar[ActionType]

class MouseButton(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    MOUSE_BUTTON_UNSPECIFIED: _ClassVar[MouseButton]
    MOUSE_BUTTON_LEFT: _ClassVar[MouseButton]
    MOUSE_BUTTON_RIGHT: _ClassVar[MouseButton]
    MOUSE_BUTTON_MIDDLE: _ClassVar[MouseButton]

class NavigateWaitEvent(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    NAVIGATE_WAIT_EVENT_UNSPECIFIED: _ClassVar[NavigateWaitEvent]
    NAVIGATE_WAIT_EVENT_LOAD: _ClassVar[NavigateWaitEvent]
    NAVIGATE_WAIT_EVENT_DOMCONTENTLOADED: _ClassVar[NavigateWaitEvent]
    NAVIGATE_WAIT_EVENT_NETWORKIDLE: _ClassVar[NavigateWaitEvent]

class WaitState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WAIT_STATE_UNSPECIFIED: _ClassVar[WaitState]
    WAIT_STATE_ATTACHED: _ClassVar[WaitState]
    WAIT_STATE_DETACHED: _ClassVar[WaitState]
    WAIT_STATE_VISIBLE: _ClassVar[WaitState]
    WAIT_STATE_HIDDEN: _ClassVar[WaitState]

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

class ScrollBehavior(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SCROLL_BEHAVIOR_UNSPECIFIED: _ClassVar[ScrollBehavior]
    SCROLL_BEHAVIOR_AUTO: _ClassVar[ScrollBehavior]
    SCROLL_BEHAVIOR_SMOOTH: _ClassVar[ScrollBehavior]

class KeyAction(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    KEY_ACTION_UNSPECIFIED: _ClassVar[KeyAction]
    KEY_ACTION_PRESS: _ClassVar[KeyAction]
    KEY_ACTION_DOWN: _ClassVar[KeyAction]
    KEY_ACTION_UP: _ClassVar[KeyAction]

class KeyboardModifier(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    KEYBOARD_MODIFIER_UNSPECIFIED: _ClassVar[KeyboardModifier]
    KEYBOARD_MODIFIER_CTRL: _ClassVar[KeyboardModifier]
    KEYBOARD_MODIFIER_SHIFT: _ClassVar[KeyboardModifier]
    KEYBOARD_MODIFIER_ALT: _ClassVar[KeyboardModifier]
    KEYBOARD_MODIFIER_META: _ClassVar[KeyboardModifier]

class TimelineMessageType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TIMELINE_MESSAGE_TYPE_UNSPECIFIED: _ClassVar[TimelineMessageType]
    TIMELINE_MESSAGE_TYPE_EVENT: _ClassVar[TimelineMessageType]
    TIMELINE_MESSAGE_TYPE_STATUS: _ClassVar[TimelineMessageType]
    TIMELINE_MESSAGE_TYPE_HEARTBEAT: _ClassVar[TimelineMessageType]
ACTION_TYPE_UNSPECIFIED: ActionType
ACTION_TYPE_NAVIGATE: ActionType
ACTION_TYPE_CLICK: ActionType
ACTION_TYPE_INPUT: ActionType
ACTION_TYPE_WAIT: ActionType
ACTION_TYPE_ASSERT: ActionType
ACTION_TYPE_SCROLL: ActionType
ACTION_TYPE_SELECT: ActionType
ACTION_TYPE_EVALUATE: ActionType
ACTION_TYPE_KEYBOARD: ActionType
ACTION_TYPE_HOVER: ActionType
ACTION_TYPE_SCREENSHOT: ActionType
ACTION_TYPE_FOCUS: ActionType
ACTION_TYPE_BLUR: ActionType
MOUSE_BUTTON_UNSPECIFIED: MouseButton
MOUSE_BUTTON_LEFT: MouseButton
MOUSE_BUTTON_RIGHT: MouseButton
MOUSE_BUTTON_MIDDLE: MouseButton
NAVIGATE_WAIT_EVENT_UNSPECIFIED: NavigateWaitEvent
NAVIGATE_WAIT_EVENT_LOAD: NavigateWaitEvent
NAVIGATE_WAIT_EVENT_DOMCONTENTLOADED: NavigateWaitEvent
NAVIGATE_WAIT_EVENT_NETWORKIDLE: NavigateWaitEvent
WAIT_STATE_UNSPECIFIED: WaitState
WAIT_STATE_ATTACHED: WaitState
WAIT_STATE_DETACHED: WaitState
WAIT_STATE_VISIBLE: WaitState
WAIT_STATE_HIDDEN: WaitState
ASSERTION_MODE_UNSPECIFIED: AssertionMode
ASSERTION_MODE_EXISTS: AssertionMode
ASSERTION_MODE_NOT_EXISTS: AssertionMode
ASSERTION_MODE_VISIBLE: AssertionMode
ASSERTION_MODE_HIDDEN: AssertionMode
ASSERTION_MODE_TEXT_EQUALS: AssertionMode
ASSERTION_MODE_TEXT_CONTAINS: AssertionMode
ASSERTION_MODE_ATTRIBUTE_EQUALS: AssertionMode
ASSERTION_MODE_ATTRIBUTE_CONTAINS: AssertionMode
SCROLL_BEHAVIOR_UNSPECIFIED: ScrollBehavior
SCROLL_BEHAVIOR_AUTO: ScrollBehavior
SCROLL_BEHAVIOR_SMOOTH: ScrollBehavior
KEY_ACTION_UNSPECIFIED: KeyAction
KEY_ACTION_PRESS: KeyAction
KEY_ACTION_DOWN: KeyAction
KEY_ACTION_UP: KeyAction
KEYBOARD_MODIFIER_UNSPECIFIED: KeyboardModifier
KEYBOARD_MODIFIER_CTRL: KeyboardModifier
KEYBOARD_MODIFIER_SHIFT: KeyboardModifier
KEYBOARD_MODIFIER_ALT: KeyboardModifier
KEYBOARD_MODIFIER_META: KeyboardModifier
TIMELINE_MESSAGE_TYPE_UNSPECIFIED: TimelineMessageType
TIMELINE_MESSAGE_TYPE_EVENT: TimelineMessageType
TIMELINE_MESSAGE_TYPE_STATUS: TimelineMessageType
TIMELINE_MESSAGE_TYPE_HEARTBEAT: TimelineMessageType

class BoundingBox(_message.Message):
    __slots__ = ()
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    x: float
    y: float
    width: float
    height: float
    def __init__(self, x: _Optional[float] = ..., y: _Optional[float] = ..., width: _Optional[float] = ..., height: _Optional[float] = ...) -> None: ...

class Point(_message.Message):
    __slots__ = ()
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    x: float
    y: float
    def __init__(self, x: _Optional[float] = ..., y: _Optional[float] = ...) -> None: ...

class SelectorCandidate(_message.Message):
    __slots__ = ()
    TYPE_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    SPECIFICITY_FIELD_NUMBER: _ClassVar[int]
    type: str
    value: str
    confidence: float
    specificity: int
    def __init__(self, type: _Optional[str] = ..., value: _Optional[str] = ..., confidence: _Optional[float] = ..., specificity: _Optional[int] = ...) -> None: ...

class SelectorSet(_message.Message):
    __slots__ = ()
    PRIMARY_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    primary: str
    candidates: _containers.RepeatedCompositeFieldContainer[SelectorCandidate]
    def __init__(self, primary: _Optional[str] = ..., candidates: _Optional[_Iterable[_Union[SelectorCandidate, _Mapping]]] = ...) -> None: ...

class ElementMeta(_message.Message):
    __slots__ = ()
    class AttributesEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TAG_NAME_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    CLASS_NAME_FIELD_NUMBER: _ClassVar[int]
    INNER_TEXT_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTES_FIELD_NUMBER: _ClassVar[int]
    IS_VISIBLE_FIELD_NUMBER: _ClassVar[int]
    IS_ENABLED_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    ARIA_LABEL_FIELD_NUMBER: _ClassVar[int]
    tag_name: str
    id: str
    class_name: str
    inner_text: str
    attributes: _containers.ScalarMap[str, str]
    is_visible: bool
    is_enabled: bool
    role: str
    aria_label: str
    def __init__(self, tag_name: _Optional[str] = ..., id: _Optional[str] = ..., class_name: _Optional[str] = ..., inner_text: _Optional[str] = ..., attributes: _Optional[_Mapping[str, str]] = ..., is_visible: _Optional[bool] = ..., is_enabled: _Optional[bool] = ..., role: _Optional[str] = ..., aria_label: _Optional[str] = ...) -> None: ...

class HighlightRegion(_message.Message):
    __slots__ = ()
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    BOUNDING_BOX_FIELD_NUMBER: _ClassVar[int]
    PADDING_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    selector: str
    bounding_box: BoundingBox
    padding: int
    color: str
    def __init__(self, selector: _Optional[str] = ..., bounding_box: _Optional[_Union[BoundingBox, _Mapping]] = ..., padding: _Optional[int] = ..., color: _Optional[str] = ...) -> None: ...

class MaskRegion(_message.Message):
    __slots__ = ()
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    BOUNDING_BOX_FIELD_NUMBER: _ClassVar[int]
    OPACITY_FIELD_NUMBER: _ClassVar[int]
    selector: str
    bounding_box: BoundingBox
    opacity: float
    def __init__(self, selector: _Optional[str] = ..., bounding_box: _Optional[_Union[BoundingBox, _Mapping]] = ..., opacity: _Optional[float] = ...) -> None: ...

class TimelineScreenshot(_message.Message):
    __slots__ = ()
    ARTIFACT_ID_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    THUMBNAIL_URL_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    artifact_id: str
    url: str
    thumbnail_url: str
    width: int
    height: int
    content_type: str
    size_bytes: int
    def __init__(self, artifact_id: _Optional[str] = ..., url: _Optional[str] = ..., thumbnail_url: _Optional[str] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., content_type: _Optional[str] = ..., size_bytes: _Optional[int] = ...) -> None: ...

class ActionDefinition(_message.Message):
    __slots__ = ()
    TYPE_FIELD_NUMBER: _ClassVar[int]
    NAVIGATE_FIELD_NUMBER: _ClassVar[int]
    CLICK_FIELD_NUMBER: _ClassVar[int]
    INPUT_FIELD_NUMBER: _ClassVar[int]
    WAIT_FIELD_NUMBER: _ClassVar[int]
    ASSERT_FIELD_NUMBER: _ClassVar[int]
    SCROLL_FIELD_NUMBER: _ClassVar[int]
    SELECT_OPTION_FIELD_NUMBER: _ClassVar[int]
    EVALUATE_FIELD_NUMBER: _ClassVar[int]
    KEYBOARD_FIELD_NUMBER: _ClassVar[int]
    HOVER_FIELD_NUMBER: _ClassVar[int]
    SCREENSHOT_FIELD_NUMBER: _ClassVar[int]
    FOCUS_FIELD_NUMBER: _ClassVar[int]
    BLUR_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    type: ActionType
    navigate: NavigateParams
    click: ClickParams
    input: InputParams
    wait: WaitParams
    scroll: ScrollParams
    select_option: SelectParams
    evaluate: EvaluateParams
    keyboard: KeyboardParams
    hover: HoverParams
    screenshot: ScreenshotParams
    focus: FocusParams
    blur: BlurParams
    metadata: ActionMetadata
    def __init__(self, type: _Optional[_Union[ActionType, str]] = ..., navigate: _Optional[_Union[NavigateParams, _Mapping]] = ..., click: _Optional[_Union[ClickParams, _Mapping]] = ..., input: _Optional[_Union[InputParams, _Mapping]] = ..., wait: _Optional[_Union[WaitParams, _Mapping]] = ..., scroll: _Optional[_Union[ScrollParams, _Mapping]] = ..., select_option: _Optional[_Union[SelectParams, _Mapping]] = ..., evaluate: _Optional[_Union[EvaluateParams, _Mapping]] = ..., keyboard: _Optional[_Union[KeyboardParams, _Mapping]] = ..., hover: _Optional[_Union[HoverParams, _Mapping]] = ..., screenshot: _Optional[_Union[ScreenshotParams, _Mapping]] = ..., focus: _Optional[_Union[FocusParams, _Mapping]] = ..., blur: _Optional[_Union[BlurParams, _Mapping]] = ..., metadata: _Optional[_Union[ActionMetadata, _Mapping]] = ..., **kwargs) -> None: ...

class NavigateParams(_message.Message):
    __slots__ = ()
    URL_FIELD_NUMBER: _ClassVar[int]
    WAIT_FOR_SELECTOR_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    WAIT_UNTIL_FIELD_NUMBER: _ClassVar[int]
    url: str
    wait_for_selector: str
    timeout_ms: int
    wait_until: NavigateWaitEvent
    def __init__(self, url: _Optional[str] = ..., wait_for_selector: _Optional[str] = ..., timeout_ms: _Optional[int] = ..., wait_until: _Optional[_Union[NavigateWaitEvent, str]] = ...) -> None: ...

class ClickParams(_message.Message):
    __slots__ = ()
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    BUTTON_FIELD_NUMBER: _ClassVar[int]
    CLICK_COUNT_FIELD_NUMBER: _ClassVar[int]
    DELAY_MS_FIELD_NUMBER: _ClassVar[int]
    MODIFIERS_FIELD_NUMBER: _ClassVar[int]
    FORCE_FIELD_NUMBER: _ClassVar[int]
    SCROLL_INTO_VIEW_FIELD_NUMBER: _ClassVar[int]
    selector: str
    button: MouseButton
    click_count: int
    delay_ms: int
    modifiers: _containers.RepeatedScalarFieldContainer[KeyboardModifier]
    force: bool
    scroll_into_view: bool
    def __init__(self, selector: _Optional[str] = ..., button: _Optional[_Union[MouseButton, str]] = ..., click_count: _Optional[int] = ..., delay_ms: _Optional[int] = ..., modifiers: _Optional[_Iterable[_Union[KeyboardModifier, str]]] = ..., force: _Optional[bool] = ..., scroll_into_view: _Optional[bool] = ...) -> None: ...

class InputParams(_message.Message):
    __slots__ = ()
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    IS_SENSITIVE_FIELD_NUMBER: _ClassVar[int]
    SUBMIT_FIELD_NUMBER: _ClassVar[int]
    CLEAR_FIRST_FIELD_NUMBER: _ClassVar[int]
    DELAY_MS_FIELD_NUMBER: _ClassVar[int]
    selector: str
    value: str
    is_sensitive: bool
    submit: bool
    clear_first: bool
    delay_ms: int
    def __init__(self, selector: _Optional[str] = ..., value: _Optional[str] = ..., is_sensitive: _Optional[bool] = ..., submit: _Optional[bool] = ..., clear_first: _Optional[bool] = ..., delay_ms: _Optional[int] = ...) -> None: ...

class WaitParams(_message.Message):
    __slots__ = ()
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    duration_ms: int
    selector: str
    state: WaitState
    timeout_ms: int
    def __init__(self, duration_ms: _Optional[int] = ..., selector: _Optional[str] = ..., state: _Optional[_Union[WaitState, str]] = ..., timeout_ms: _Optional[int] = ...) -> None: ...

class AssertParams(_message.Message):
    __slots__ = ()
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_FIELD_NUMBER: _ClassVar[int]
    NEGATED_FIELD_NUMBER: _ClassVar[int]
    CASE_SENSITIVE_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTE_NAME_FIELD_NUMBER: _ClassVar[int]
    FAILURE_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    selector: str
    mode: AssertionMode
    expected: _types_pb2.JsonValue
    negated: bool
    case_sensitive: bool
    attribute_name: str
    failure_message: str
    timeout_ms: int
    def __init__(self, selector: _Optional[str] = ..., mode: _Optional[_Union[AssertionMode, str]] = ..., expected: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ..., negated: _Optional[bool] = ..., case_sensitive: _Optional[bool] = ..., attribute_name: _Optional[str] = ..., failure_message: _Optional[str] = ..., timeout_ms: _Optional[int] = ...) -> None: ...

class ScrollParams(_message.Message):
    __slots__ = ()
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    BEHAVIOR_FIELD_NUMBER: _ClassVar[int]
    DELTA_X_FIELD_NUMBER: _ClassVar[int]
    DELTA_Y_FIELD_NUMBER: _ClassVar[int]
    selector: str
    x: int
    y: int
    behavior: ScrollBehavior
    delta_x: int
    delta_y: int
    def __init__(self, selector: _Optional[str] = ..., x: _Optional[int] = ..., y: _Optional[int] = ..., behavior: _Optional[_Union[ScrollBehavior, str]] = ..., delta_x: _Optional[int] = ..., delta_y: _Optional[int] = ...) -> None: ...

class SelectParams(_message.Message):
    __slots__ = ()
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    INDEX_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    selector: str
    value: str
    label: str
    index: int
    timeout_ms: int
    def __init__(self, selector: _Optional[str] = ..., value: _Optional[str] = ..., label: _Optional[str] = ..., index: _Optional[int] = ..., timeout_ms: _Optional[int] = ...) -> None: ...

class EvaluateParams(_message.Message):
    __slots__ = ()
    class ArgsEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    EXPRESSION_FIELD_NUMBER: _ClassVar[int]
    STORE_RESULT_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    expression: str
    store_result: str
    args: _containers.MessageMap[str, _types_pb2.JsonValue]
    def __init__(self, expression: _Optional[str] = ..., store_result: _Optional[str] = ..., args: _Optional[_Mapping[str, _types_pb2.JsonValue]] = ...) -> None: ...

class KeyboardParams(_message.Message):
    __slots__ = ()
    KEY_FIELD_NUMBER: _ClassVar[int]
    KEYS_FIELD_NUMBER: _ClassVar[int]
    MODIFIERS_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    key: str
    keys: _containers.RepeatedScalarFieldContainer[str]
    modifiers: _containers.RepeatedScalarFieldContainer[KeyboardModifier]
    action: KeyAction
    def __init__(self, key: _Optional[str] = ..., keys: _Optional[_Iterable[str]] = ..., modifiers: _Optional[_Iterable[_Union[KeyboardModifier, str]]] = ..., action: _Optional[_Union[KeyAction, str]] = ...) -> None: ...

class HoverParams(_message.Message):
    __slots__ = ()
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    selector: str
    timeout_ms: int
    def __init__(self, selector: _Optional[str] = ..., timeout_ms: _Optional[int] = ...) -> None: ...

class ScreenshotParams(_message.Message):
    __slots__ = ()
    FULL_PAGE_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    QUALITY_FIELD_NUMBER: _ClassVar[int]
    full_page: bool
    selector: str
    quality: int
    def __init__(self, full_page: _Optional[bool] = ..., selector: _Optional[str] = ..., quality: _Optional[int] = ...) -> None: ...

class FocusParams(_message.Message):
    __slots__ = ()
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    SCROLL_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    selector: str
    scroll: bool
    timeout_ms: int
    def __init__(self, selector: _Optional[str] = ..., scroll: _Optional[bool] = ..., timeout_ms: _Optional[int] = ...) -> None: ...

class BlurParams(_message.Message):
    __slots__ = ()
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    selector: str
    timeout_ms: int
    def __init__(self, selector: _Optional[str] = ..., timeout_ms: _Optional[int] = ...) -> None: ...

class ActionMetadata(_message.Message):
    __slots__ = ()
    LABEL_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    ELEMENT_SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    RECORDED_BOUNDING_BOX_FIELD_NUMBER: _ClassVar[int]
    label: str
    selector_candidates: _containers.RepeatedCompositeFieldContainer[SelectorCandidate]
    element_snapshot: ElementMeta
    confidence: float
    recorded_at: _timestamp_pb2.Timestamp
    recorded_bounding_box: BoundingBox
    def __init__(self, label: _Optional[str] = ..., selector_candidates: _Optional[_Iterable[_Union[SelectorCandidate, _Mapping]]] = ..., element_snapshot: _Optional[_Union[ElementMeta, _Mapping]] = ..., confidence: _Optional[float] = ..., recorded_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., recorded_bounding_box: _Optional[_Union[BoundingBox, _Mapping]] = ...) -> None: ...

class ActionTelemetry(_message.Message):
    __slots__ = ()
    URL_FIELD_NUMBER: _ClassVar[int]
    FRAME_ID_FIELD_NUMBER: _ClassVar[int]
    SCREENSHOT_FIELD_NUMBER: _ClassVar[int]
    DOM_SNAPSHOT_PREVIEW_FIELD_NUMBER: _ClassVar[int]
    DOM_SNAPSHOT_HTML_FIELD_NUMBER: _ClassVar[int]
    ELEMENT_BOUNDING_BOX_FIELD_NUMBER: _ClassVar[int]
    CLICK_POSITION_FIELD_NUMBER: _ClassVar[int]
    CURSOR_POSITION_FIELD_NUMBER: _ClassVar[int]
    CURSOR_TRAIL_FIELD_NUMBER: _ClassVar[int]
    HIGHLIGHT_REGIONS_FIELD_NUMBER: _ClassVar[int]
    MASK_REGIONS_FIELD_NUMBER: _ClassVar[int]
    ZOOM_FACTOR_FIELD_NUMBER: _ClassVar[int]
    CONSOLE_LOGS_FIELD_NUMBER: _ClassVar[int]
    NETWORK_EVENTS_FIELD_NUMBER: _ClassVar[int]
    url: str
    frame_id: str
    screenshot: TimelineScreenshot
    dom_snapshot_preview: str
    dom_snapshot_html: str
    element_bounding_box: BoundingBox
    click_position: Point
    cursor_position: Point
    cursor_trail: _containers.RepeatedCompositeFieldContainer[Point]
    highlight_regions: _containers.RepeatedCompositeFieldContainer[HighlightRegion]
    mask_regions: _containers.RepeatedCompositeFieldContainer[MaskRegion]
    zoom_factor: float
    console_logs: _containers.RepeatedCompositeFieldContainer[ConsoleLogEntryUnified]
    network_events: _containers.RepeatedCompositeFieldContainer[NetworkEventUnified]
    def __init__(self, url: _Optional[str] = ..., frame_id: _Optional[str] = ..., screenshot: _Optional[_Union[TimelineScreenshot, _Mapping]] = ..., dom_snapshot_preview: _Optional[str] = ..., dom_snapshot_html: _Optional[str] = ..., element_bounding_box: _Optional[_Union[BoundingBox, _Mapping]] = ..., click_position: _Optional[_Union[Point, _Mapping]] = ..., cursor_position: _Optional[_Union[Point, _Mapping]] = ..., cursor_trail: _Optional[_Iterable[_Union[Point, _Mapping]]] = ..., highlight_regions: _Optional[_Iterable[_Union[HighlightRegion, _Mapping]]] = ..., mask_regions: _Optional[_Iterable[_Union[MaskRegion, _Mapping]]] = ..., zoom_factor: _Optional[float] = ..., console_logs: _Optional[_Iterable[_Union[ConsoleLogEntryUnified, _Mapping]]] = ..., network_events: _Optional[_Iterable[_Union[NetworkEventUnified, _Mapping]]] = ...) -> None: ...

class ConsoleLogEntryUnified(_message.Message):
    __slots__ = ()
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    STACK_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    level: str
    text: str
    timestamp: _timestamp_pb2.Timestamp
    stack: str
    location: str
    def __init__(self, level: _Optional[str] = ..., text: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., stack: _Optional[str] = ..., location: _Optional[str] = ...) -> None: ...

class NetworkEventUnified(_message.Message):
    __slots__ = ()
    TYPE_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_TYPE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    OK_FIELD_NUMBER: _ClassVar[int]
    FAILURE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    type: str
    url: str
    method: str
    resource_type: str
    status: int
    ok: bool
    failure: str
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, type: _Optional[str] = ..., url: _Optional[str] = ..., method: _Optional[str] = ..., resource_type: _Optional[str] = ..., status: _Optional[int] = ..., ok: _Optional[bool] = ..., failure: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class TimelineEvent(_message.Message):
    __slots__ = ()
    ID_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_NUM_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    TELEMETRY_FIELD_NUMBER: _ClassVar[int]
    RECORDING_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    id: str
    sequence_num: int
    timestamp: _timestamp_pb2.Timestamp
    duration_ms: int
    action: ActionDefinition
    telemetry: ActionTelemetry
    recording: RecordingEventData
    execution: ExecutionEventData
    def __init__(self, id: _Optional[str] = ..., sequence_num: _Optional[int] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., duration_ms: _Optional[int] = ..., action: _Optional[_Union[ActionDefinition, _Mapping]] = ..., telemetry: _Optional[_Union[ActionTelemetry, _Mapping]] = ..., recording: _Optional[_Union[RecordingEventData, _Mapping]] = ..., execution: _Optional[_Union[ExecutionEventData, _Mapping]] = ...) -> None: ...

class RecordingEventData(_message.Message):
    __slots__ = ()
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    NEEDS_CONFIRMATION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    selector_candidates: _containers.RepeatedCompositeFieldContainer[SelectorCandidate]
    needs_confirmation: bool
    source: str
    def __init__(self, session_id: _Optional[str] = ..., selector_candidates: _Optional[_Iterable[_Union[SelectorCandidate, _Mapping]]] = ..., needs_confirmation: _Optional[bool] = ..., source: _Optional[str] = ...) -> None: ...

class ExecutionEventData(_message.Message):
    __slots__ = ()
    class ExtractedDataEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    ERROR_CODE_FIELD_NUMBER: _ClassVar[int]
    ATTEMPT_FIELD_NUMBER: _ClassVar[int]
    MAX_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    RETRY_DELAY_MS_FIELD_NUMBER: _ClassVar[int]
    RETRY_BACKOFF_FACTOR_FIELD_NUMBER: _ClassVar[int]
    RETRY_HISTORY_FIELD_NUMBER: _ClassVar[int]
    ASSERTION_FIELD_NUMBER: _ClassVar[int]
    EXTRACTED_DATA_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    node_id: str
    success: bool
    error: str
    error_code: str
    attempt: int
    max_attempts: int
    retry_delay_ms: int
    retry_backoff_factor: float
    retry_history: _containers.RepeatedCompositeFieldContainer[RetryAttemptProto]
    assertion: AssertionResultProto
    extracted_data: _containers.MessageMap[str, _types_pb2.JsonValue]
    def __init__(self, execution_id: _Optional[str] = ..., node_id: _Optional[str] = ..., success: _Optional[bool] = ..., error: _Optional[str] = ..., error_code: _Optional[str] = ..., attempt: _Optional[int] = ..., max_attempts: _Optional[int] = ..., retry_delay_ms: _Optional[int] = ..., retry_backoff_factor: _Optional[float] = ..., retry_history: _Optional[_Iterable[_Union[RetryAttemptProto, _Mapping]]] = ..., assertion: _Optional[_Union[AssertionResultProto, _Mapping]] = ..., extracted_data: _Optional[_Mapping[str, _types_pb2.JsonValue]] = ...) -> None: ...

class RetryAttemptProto(_message.Message):
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

class AssertionResultProto(_message.Message):
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

class WorkflowDefinitionV2(_message.Message):
    __slots__ = ()
    METADATA_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    NODES_FIELD_NUMBER: _ClassVar[int]
    EDGES_FIELD_NUMBER: _ClassVar[int]
    metadata: WorkflowMetadataV2
    settings: WorkflowSettingsV2
    nodes: _containers.RepeatedCompositeFieldContainer[WorkflowNodeV2]
    edges: _containers.RepeatedCompositeFieldContainer[WorkflowEdgeV2]
    def __init__(self, metadata: _Optional[_Union[WorkflowMetadataV2, _Mapping]] = ..., settings: _Optional[_Union[WorkflowSettingsV2, _Mapping]] = ..., nodes: _Optional[_Iterable[_Union[WorkflowNodeV2, _Mapping]]] = ..., edges: _Optional[_Iterable[_Union[WorkflowEdgeV2, _Mapping]]] = ...) -> None: ...

class WorkflowMetadataV2(_message.Message):
    __slots__ = ()
    class LabelsEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    LABELS_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    REQUIREMENT_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    labels: _containers.ScalarMap[str, str]
    version: str
    requirement: str
    owner: str
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., labels: _Optional[_Mapping[str, str]] = ..., version: _Optional[str] = ..., requirement: _Optional[str] = ..., owner: _Optional[str] = ...) -> None: ...

class WorkflowSettingsV2(_message.Message):
    __slots__ = ()
    VIEWPORT_WIDTH_FIELD_NUMBER: _ClassVar[int]
    VIEWPORT_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    USER_AGENT_FIELD_NUMBER: _ClassVar[int]
    LOCALE_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    HEADLESS_FIELD_NUMBER: _ClassVar[int]
    ENTRY_SELECTOR_FIELD_NUMBER: _ClassVar[int]
    ENTRY_SELECTOR_TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    viewport_width: int
    viewport_height: int
    user_agent: str
    locale: str
    timeout_seconds: int
    headless: bool
    entry_selector: str
    entry_selector_timeout_ms: int
    def __init__(self, viewport_width: _Optional[int] = ..., viewport_height: _Optional[int] = ..., user_agent: _Optional[str] = ..., locale: _Optional[str] = ..., timeout_seconds: _Optional[int] = ..., headless: _Optional[bool] = ..., entry_selector: _Optional[str] = ..., entry_selector_timeout_ms: _Optional[int] = ...) -> None: ...

class WorkflowNodeV2(_message.Message):
    __slots__ = ()
    ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    POSITION_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_SETTINGS_FIELD_NUMBER: _ClassVar[int]
    id: str
    action: ActionDefinition
    position: NodePosition
    execution_settings: NodeExecutionSettings
    def __init__(self, id: _Optional[str] = ..., action: _Optional[_Union[ActionDefinition, _Mapping]] = ..., position: _Optional[_Union[NodePosition, _Mapping]] = ..., execution_settings: _Optional[_Union[NodeExecutionSettings, _Mapping]] = ...) -> None: ...

class NodePosition(_message.Message):
    __slots__ = ()
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    x: float
    y: float
    def __init__(self, x: _Optional[float] = ..., y: _Optional[float] = ...) -> None: ...

class NodeExecutionSettings(_message.Message):
    __slots__ = ()
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    WAIT_AFTER_MS_FIELD_NUMBER: _ClassVar[int]
    CONTINUE_ON_ERROR_FIELD_NUMBER: _ClassVar[int]
    RESILIENCE_FIELD_NUMBER: _ClassVar[int]
    timeout_ms: int
    wait_after_ms: int
    continue_on_error: bool
    resilience: ResilienceConfig
    def __init__(self, timeout_ms: _Optional[int] = ..., wait_after_ms: _Optional[int] = ..., continue_on_error: _Optional[bool] = ..., resilience: _Optional[_Union[ResilienceConfig, _Mapping]] = ...) -> None: ...

class ResilienceConfig(_message.Message):
    __slots__ = ()
    MAX_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    DELAY_MS_FIELD_NUMBER: _ClassVar[int]
    BACKOFF_FACTOR_FIELD_NUMBER: _ClassVar[int]
    PRECONDITION_SELECTOR_FIELD_NUMBER: _ClassVar[int]
    PRECONDITION_TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    PRECONDITION_WAIT_MS_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_SELECTOR_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_WAIT_MS_FIELD_NUMBER: _ClassVar[int]
    max_attempts: int
    delay_ms: int
    backoff_factor: float
    precondition_selector: str
    precondition_timeout_ms: int
    precondition_wait_ms: int
    success_selector: str
    success_timeout_ms: int
    success_wait_ms: int
    def __init__(self, max_attempts: _Optional[int] = ..., delay_ms: _Optional[int] = ..., backoff_factor: _Optional[float] = ..., precondition_selector: _Optional[str] = ..., precondition_timeout_ms: _Optional[int] = ..., precondition_wait_ms: _Optional[int] = ..., success_selector: _Optional[str] = ..., success_timeout_ms: _Optional[int] = ..., success_wait_ms: _Optional[int] = ...) -> None: ...

class WorkflowEdgeV2(_message.Message):
    __slots__ = ()
    ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    SOURCE_HANDLE_FIELD_NUMBER: _ClassVar[int]
    TARGET_HANDLE_FIELD_NUMBER: _ClassVar[int]
    id: str
    source: str
    target: str
    type: str
    label: str
    source_handle: str
    target_handle: str
    def __init__(self, id: _Optional[str] = ..., source: _Optional[str] = ..., target: _Optional[str] = ..., type: _Optional[str] = ..., label: _Optional[str] = ..., source_handle: _Optional[str] = ..., target_handle: _Optional[str] = ...) -> None: ...

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
    status: str
    progress: int
    event_count: int
    error: str
    def __init__(self, id: _Optional[str] = ..., status: _Optional[str] = ..., progress: _Optional[int] = ..., event_count: _Optional[int] = ..., error: _Optional[str] = ...) -> None: ...

class TimelineHeartbeat(_message.Message):
    __slots__ = ()
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    timestamp: _timestamp_pb2.Timestamp
    session_id: str
    def __init__(self, timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., session_id: _Optional[str] = ...) -> None: ...
