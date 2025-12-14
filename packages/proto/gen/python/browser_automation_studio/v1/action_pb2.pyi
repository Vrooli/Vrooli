import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from common.v1 import types_pb2 as _types_pb2
from browser_automation_studio.v1 import shared_pb2 as _shared_pb2
from browser_automation_studio.v1 import selectors_pb2 as _selectors_pb2
from browser_automation_studio.v1 import geometry_pb2 as _geometry_pb2
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
    mode: _shared_pb2.AssertionMode
    expected: _types_pb2.JsonValue
    negated: bool
    case_sensitive: bool
    attribute_name: str
    failure_message: str
    timeout_ms: int
    def __init__(self, selector: _Optional[str] = ..., mode: _Optional[_Union[_shared_pb2.AssertionMode, str]] = ..., expected: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ..., negated: _Optional[bool] = ..., case_sensitive: _Optional[bool] = ..., attribute_name: _Optional[str] = ..., failure_message: _Optional[str] = ..., timeout_ms: _Optional[int] = ...) -> None: ...

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
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_BOUNDING_BOX_FIELD_NUMBER: _ClassVar[int]
    label: str
    selector_candidates: _containers.RepeatedCompositeFieldContainer[_selectors_pb2.SelectorCandidate]
    element_snapshot: _selectors_pb2.ElementMeta
    confidence: float
    captured_at: _timestamp_pb2.Timestamp
    captured_bounding_box: _geometry_pb2.BoundingBox
    def __init__(self, label: _Optional[str] = ..., selector_candidates: _Optional[_Iterable[_Union[_selectors_pb2.SelectorCandidate, _Mapping]]] = ..., element_snapshot: _Optional[_Union[_selectors_pb2.ElementMeta, _Mapping]] = ..., confidence: _Optional[float] = ..., captured_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., captured_bounding_box: _Optional[_Union[_geometry_pb2.BoundingBox, _Mapping]] = ...) -> None: ...

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
