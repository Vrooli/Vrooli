import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from buf.validate import validate_pb2 as _validate_pb2
from common.v1 import types_pb2 as _types_pb2
from browser_automation_studio.v1.base import shared_pb2 as _shared_pb2
from browser_automation_studio.v1.base import geometry_pb2 as _geometry_pb2
from browser_automation_studio.v1.domain import selectors_pb2 as _selectors_pb2
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
    ACTION_TYPE_SUBFLOW: _ClassVar[ActionType]
    ACTION_TYPE_EXTRACT: _ClassVar[ActionType]
    ACTION_TYPE_UPLOAD_FILE: _ClassVar[ActionType]
    ACTION_TYPE_DOWNLOAD: _ClassVar[ActionType]
    ACTION_TYPE_FRAME_SWITCH: _ClassVar[ActionType]
    ACTION_TYPE_TAB_SWITCH: _ClassVar[ActionType]
    ACTION_TYPE_COOKIE_STORAGE: _ClassVar[ActionType]
    ACTION_TYPE_SHORTCUT: _ClassVar[ActionType]
    ACTION_TYPE_DRAG_DROP: _ClassVar[ActionType]
    ACTION_TYPE_GESTURE: _ClassVar[ActionType]
    ACTION_TYPE_NETWORK_MOCK: _ClassVar[ActionType]
    ACTION_TYPE_ROTATE: _ClassVar[ActionType]
    ACTION_TYPE_SET_VARIABLE: _ClassVar[ActionType]
    ACTION_TYPE_LOOP: _ClassVar[ActionType]
    ACTION_TYPE_CONDITIONAL: _ClassVar[ActionType]

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

class NavigateDestinationType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    NAVIGATE_DESTINATION_TYPE_UNSPECIFIED: _ClassVar[NavigateDestinationType]
    NAVIGATE_DESTINATION_TYPE_URL: _ClassVar[NavigateDestinationType]
    NAVIGATE_DESTINATION_TYPE_SCENARIO: _ClassVar[NavigateDestinationType]

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

class ExtractType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EXTRACT_TYPE_UNSPECIFIED: _ClassVar[ExtractType]
    EXTRACT_TYPE_TEXT: _ClassVar[ExtractType]
    EXTRACT_TYPE_INNER_HTML: _ClassVar[ExtractType]
    EXTRACT_TYPE_OUTER_HTML: _ClassVar[ExtractType]
    EXTRACT_TYPE_ATTRIBUTE: _ClassVar[ExtractType]
    EXTRACT_TYPE_PROPERTY: _ClassVar[ExtractType]
    EXTRACT_TYPE_VALUE: _ClassVar[ExtractType]

class FrameSwitchAction(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FRAME_SWITCH_ACTION_UNSPECIFIED: _ClassVar[FrameSwitchAction]
    FRAME_SWITCH_ACTION_ENTER: _ClassVar[FrameSwitchAction]
    FRAME_SWITCH_ACTION_PARENT: _ClassVar[FrameSwitchAction]
    FRAME_SWITCH_ACTION_EXIT: _ClassVar[FrameSwitchAction]

class TabSwitchAction(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TAB_SWITCH_ACTION_UNSPECIFIED: _ClassVar[TabSwitchAction]
    TAB_SWITCH_ACTION_OPEN: _ClassVar[TabSwitchAction]
    TAB_SWITCH_ACTION_SWITCH: _ClassVar[TabSwitchAction]
    TAB_SWITCH_ACTION_CLOSE: _ClassVar[TabSwitchAction]
    TAB_SWITCH_ACTION_LIST: _ClassVar[TabSwitchAction]

class CookieOperation(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COOKIE_OPERATION_UNSPECIFIED: _ClassVar[CookieOperation]
    COOKIE_OPERATION_GET: _ClassVar[CookieOperation]
    COOKIE_OPERATION_SET: _ClassVar[CookieOperation]
    COOKIE_OPERATION_DELETE: _ClassVar[CookieOperation]
    COOKIE_OPERATION_CLEAR: _ClassVar[CookieOperation]

class StorageType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    STORAGE_TYPE_UNSPECIFIED: _ClassVar[StorageType]
    STORAGE_TYPE_COOKIE: _ClassVar[StorageType]
    STORAGE_TYPE_LOCAL_STORAGE: _ClassVar[StorageType]
    STORAGE_TYPE_SESSION_STORAGE: _ClassVar[StorageType]

class GestureType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    GESTURE_TYPE_UNSPECIFIED: _ClassVar[GestureType]
    GESTURE_TYPE_SWIPE: _ClassVar[GestureType]
    GESTURE_TYPE_PINCH: _ClassVar[GestureType]
    GESTURE_TYPE_ZOOM: _ClassVar[GestureType]
    GESTURE_TYPE_LONG_PRESS: _ClassVar[GestureType]
    GESTURE_TYPE_DOUBLE_TAP: _ClassVar[GestureType]

class SwipeDirection(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SWIPE_DIRECTION_UNSPECIFIED: _ClassVar[SwipeDirection]
    SWIPE_DIRECTION_UP: _ClassVar[SwipeDirection]
    SWIPE_DIRECTION_DOWN: _ClassVar[SwipeDirection]
    SWIPE_DIRECTION_LEFT: _ClassVar[SwipeDirection]
    SWIPE_DIRECTION_RIGHT: _ClassVar[SwipeDirection]

class NetworkMockOperation(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    NETWORK_MOCK_OPERATION_UNSPECIFIED: _ClassVar[NetworkMockOperation]
    NETWORK_MOCK_OPERATION_MOCK: _ClassVar[NetworkMockOperation]
    NETWORK_MOCK_OPERATION_BLOCK: _ClassVar[NetworkMockOperation]
    NETWORK_MOCK_OPERATION_MODIFY_REQUEST: _ClassVar[NetworkMockOperation]
    NETWORK_MOCK_OPERATION_MODIFY_RESPONSE: _ClassVar[NetworkMockOperation]
    NETWORK_MOCK_OPERATION_CLEAR: _ClassVar[NetworkMockOperation]

class DeviceOrientation(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DEVICE_ORIENTATION_UNSPECIFIED: _ClassVar[DeviceOrientation]
    DEVICE_ORIENTATION_PORTRAIT: _ClassVar[DeviceOrientation]
    DEVICE_ORIENTATION_LANDSCAPE: _ClassVar[DeviceOrientation]

class CookieSameSite(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COOKIE_SAME_SITE_UNSPECIFIED: _ClassVar[CookieSameSite]
    COOKIE_SAME_SITE_STRICT: _ClassVar[CookieSameSite]
    COOKIE_SAME_SITE_LAX: _ClassVar[CookieSameSite]
    COOKIE_SAME_SITE_NONE: _ClassVar[CookieSameSite]

class SetVariableSourceType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SET_VARIABLE_SOURCE_TYPE_UNSPECIFIED: _ClassVar[SetVariableSourceType]
    SET_VARIABLE_SOURCE_TYPE_STATIC: _ClassVar[SetVariableSourceType]
    SET_VARIABLE_SOURCE_TYPE_EXPRESSION: _ClassVar[SetVariableSourceType]
    SET_VARIABLE_SOURCE_TYPE_EXTRACT: _ClassVar[SetVariableSourceType]

class SetVariableValueType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SET_VARIABLE_VALUE_TYPE_UNSPECIFIED: _ClassVar[SetVariableValueType]
    SET_VARIABLE_VALUE_TYPE_TEXT: _ClassVar[SetVariableValueType]
    SET_VARIABLE_VALUE_TYPE_NUMBER: _ClassVar[SetVariableValueType]
    SET_VARIABLE_VALUE_TYPE_BOOLEAN: _ClassVar[SetVariableValueType]
    SET_VARIABLE_VALUE_TYPE_JSON: _ClassVar[SetVariableValueType]

class SetVariableExtractType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SET_VARIABLE_EXTRACT_TYPE_UNSPECIFIED: _ClassVar[SetVariableExtractType]
    SET_VARIABLE_EXTRACT_TYPE_TEXT: _ClassVar[SetVariableExtractType]
    SET_VARIABLE_EXTRACT_TYPE_ATTRIBUTE: _ClassVar[SetVariableExtractType]
    SET_VARIABLE_EXTRACT_TYPE_VALUE: _ClassVar[SetVariableExtractType]
    SET_VARIABLE_EXTRACT_TYPE_HTML: _ClassVar[SetVariableExtractType]

class LoopType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    LOOP_TYPE_UNSPECIFIED: _ClassVar[LoopType]
    LOOP_TYPE_FOREACH: _ClassVar[LoopType]
    LOOP_TYPE_REPEAT: _ClassVar[LoopType]
    LOOP_TYPE_WHILE: _ClassVar[LoopType]

class LoopConditionType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    LOOP_CONDITION_TYPE_UNSPECIFIED: _ClassVar[LoopConditionType]
    LOOP_CONDITION_TYPE_VARIABLE: _ClassVar[LoopConditionType]
    LOOP_CONDITION_TYPE_EXPRESSION: _ClassVar[LoopConditionType]

class LoopConditionOperator(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    LOOP_CONDITION_OPERATOR_UNSPECIFIED: _ClassVar[LoopConditionOperator]
    LOOP_CONDITION_OPERATOR_TRUTHY: _ClassVar[LoopConditionOperator]
    LOOP_CONDITION_OPERATOR_EQUALS: _ClassVar[LoopConditionOperator]
    LOOP_CONDITION_OPERATOR_NOT_EQUALS: _ClassVar[LoopConditionOperator]
    LOOP_CONDITION_OPERATOR_CONTAINS: _ClassVar[LoopConditionOperator]
    LOOP_CONDITION_OPERATOR_STARTS_WITH: _ClassVar[LoopConditionOperator]
    LOOP_CONDITION_OPERATOR_ENDS_WITH: _ClassVar[LoopConditionOperator]
    LOOP_CONDITION_OPERATOR_GT: _ClassVar[LoopConditionOperator]
    LOOP_CONDITION_OPERATOR_GTE: _ClassVar[LoopConditionOperator]
    LOOP_CONDITION_OPERATOR_LT: _ClassVar[LoopConditionOperator]
    LOOP_CONDITION_OPERATOR_LTE: _ClassVar[LoopConditionOperator]

class ConditionalType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONDITIONAL_TYPE_UNSPECIFIED: _ClassVar[ConditionalType]
    CONDITIONAL_TYPE_EXPRESSION: _ClassVar[ConditionalType]
    CONDITIONAL_TYPE_ELEMENT: _ClassVar[ConditionalType]
    CONDITIONAL_TYPE_VARIABLE: _ClassVar[ConditionalType]

class ConditionalOperator(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONDITIONAL_OPERATOR_UNSPECIFIED: _ClassVar[ConditionalOperator]
    CONDITIONAL_OPERATOR_EQUALS: _ClassVar[ConditionalOperator]
    CONDITIONAL_OPERATOR_NOT_EQUALS: _ClassVar[ConditionalOperator]
    CONDITIONAL_OPERATOR_CONTAINS: _ClassVar[ConditionalOperator]
    CONDITIONAL_OPERATOR_STARTS_WITH: _ClassVar[ConditionalOperator]
    CONDITIONAL_OPERATOR_ENDS_WITH: _ClassVar[ConditionalOperator]
    CONDITIONAL_OPERATOR_GT: _ClassVar[ConditionalOperator]
    CONDITIONAL_OPERATOR_GTE: _ClassVar[ConditionalOperator]
    CONDITIONAL_OPERATOR_LT: _ClassVar[ConditionalOperator]
    CONDITIONAL_OPERATOR_LTE: _ClassVar[ConditionalOperator]
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
ACTION_TYPE_SUBFLOW: ActionType
ACTION_TYPE_EXTRACT: ActionType
ACTION_TYPE_UPLOAD_FILE: ActionType
ACTION_TYPE_DOWNLOAD: ActionType
ACTION_TYPE_FRAME_SWITCH: ActionType
ACTION_TYPE_TAB_SWITCH: ActionType
ACTION_TYPE_COOKIE_STORAGE: ActionType
ACTION_TYPE_SHORTCUT: ActionType
ACTION_TYPE_DRAG_DROP: ActionType
ACTION_TYPE_GESTURE: ActionType
ACTION_TYPE_NETWORK_MOCK: ActionType
ACTION_TYPE_ROTATE: ActionType
ACTION_TYPE_SET_VARIABLE: ActionType
ACTION_TYPE_LOOP: ActionType
ACTION_TYPE_CONDITIONAL: ActionType
MOUSE_BUTTON_UNSPECIFIED: MouseButton
MOUSE_BUTTON_LEFT: MouseButton
MOUSE_BUTTON_RIGHT: MouseButton
MOUSE_BUTTON_MIDDLE: MouseButton
NAVIGATE_WAIT_EVENT_UNSPECIFIED: NavigateWaitEvent
NAVIGATE_WAIT_EVENT_LOAD: NavigateWaitEvent
NAVIGATE_WAIT_EVENT_DOMCONTENTLOADED: NavigateWaitEvent
NAVIGATE_WAIT_EVENT_NETWORKIDLE: NavigateWaitEvent
NAVIGATE_DESTINATION_TYPE_UNSPECIFIED: NavigateDestinationType
NAVIGATE_DESTINATION_TYPE_URL: NavigateDestinationType
NAVIGATE_DESTINATION_TYPE_SCENARIO: NavigateDestinationType
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
EXTRACT_TYPE_UNSPECIFIED: ExtractType
EXTRACT_TYPE_TEXT: ExtractType
EXTRACT_TYPE_INNER_HTML: ExtractType
EXTRACT_TYPE_OUTER_HTML: ExtractType
EXTRACT_TYPE_ATTRIBUTE: ExtractType
EXTRACT_TYPE_PROPERTY: ExtractType
EXTRACT_TYPE_VALUE: ExtractType
FRAME_SWITCH_ACTION_UNSPECIFIED: FrameSwitchAction
FRAME_SWITCH_ACTION_ENTER: FrameSwitchAction
FRAME_SWITCH_ACTION_PARENT: FrameSwitchAction
FRAME_SWITCH_ACTION_EXIT: FrameSwitchAction
TAB_SWITCH_ACTION_UNSPECIFIED: TabSwitchAction
TAB_SWITCH_ACTION_OPEN: TabSwitchAction
TAB_SWITCH_ACTION_SWITCH: TabSwitchAction
TAB_SWITCH_ACTION_CLOSE: TabSwitchAction
TAB_SWITCH_ACTION_LIST: TabSwitchAction
COOKIE_OPERATION_UNSPECIFIED: CookieOperation
COOKIE_OPERATION_GET: CookieOperation
COOKIE_OPERATION_SET: CookieOperation
COOKIE_OPERATION_DELETE: CookieOperation
COOKIE_OPERATION_CLEAR: CookieOperation
STORAGE_TYPE_UNSPECIFIED: StorageType
STORAGE_TYPE_COOKIE: StorageType
STORAGE_TYPE_LOCAL_STORAGE: StorageType
STORAGE_TYPE_SESSION_STORAGE: StorageType
GESTURE_TYPE_UNSPECIFIED: GestureType
GESTURE_TYPE_SWIPE: GestureType
GESTURE_TYPE_PINCH: GestureType
GESTURE_TYPE_ZOOM: GestureType
GESTURE_TYPE_LONG_PRESS: GestureType
GESTURE_TYPE_DOUBLE_TAP: GestureType
SWIPE_DIRECTION_UNSPECIFIED: SwipeDirection
SWIPE_DIRECTION_UP: SwipeDirection
SWIPE_DIRECTION_DOWN: SwipeDirection
SWIPE_DIRECTION_LEFT: SwipeDirection
SWIPE_DIRECTION_RIGHT: SwipeDirection
NETWORK_MOCK_OPERATION_UNSPECIFIED: NetworkMockOperation
NETWORK_MOCK_OPERATION_MOCK: NetworkMockOperation
NETWORK_MOCK_OPERATION_BLOCK: NetworkMockOperation
NETWORK_MOCK_OPERATION_MODIFY_REQUEST: NetworkMockOperation
NETWORK_MOCK_OPERATION_MODIFY_RESPONSE: NetworkMockOperation
NETWORK_MOCK_OPERATION_CLEAR: NetworkMockOperation
DEVICE_ORIENTATION_UNSPECIFIED: DeviceOrientation
DEVICE_ORIENTATION_PORTRAIT: DeviceOrientation
DEVICE_ORIENTATION_LANDSCAPE: DeviceOrientation
COOKIE_SAME_SITE_UNSPECIFIED: CookieSameSite
COOKIE_SAME_SITE_STRICT: CookieSameSite
COOKIE_SAME_SITE_LAX: CookieSameSite
COOKIE_SAME_SITE_NONE: CookieSameSite
SET_VARIABLE_SOURCE_TYPE_UNSPECIFIED: SetVariableSourceType
SET_VARIABLE_SOURCE_TYPE_STATIC: SetVariableSourceType
SET_VARIABLE_SOURCE_TYPE_EXPRESSION: SetVariableSourceType
SET_VARIABLE_SOURCE_TYPE_EXTRACT: SetVariableSourceType
SET_VARIABLE_VALUE_TYPE_UNSPECIFIED: SetVariableValueType
SET_VARIABLE_VALUE_TYPE_TEXT: SetVariableValueType
SET_VARIABLE_VALUE_TYPE_NUMBER: SetVariableValueType
SET_VARIABLE_VALUE_TYPE_BOOLEAN: SetVariableValueType
SET_VARIABLE_VALUE_TYPE_JSON: SetVariableValueType
SET_VARIABLE_EXTRACT_TYPE_UNSPECIFIED: SetVariableExtractType
SET_VARIABLE_EXTRACT_TYPE_TEXT: SetVariableExtractType
SET_VARIABLE_EXTRACT_TYPE_ATTRIBUTE: SetVariableExtractType
SET_VARIABLE_EXTRACT_TYPE_VALUE: SetVariableExtractType
SET_VARIABLE_EXTRACT_TYPE_HTML: SetVariableExtractType
LOOP_TYPE_UNSPECIFIED: LoopType
LOOP_TYPE_FOREACH: LoopType
LOOP_TYPE_REPEAT: LoopType
LOOP_TYPE_WHILE: LoopType
LOOP_CONDITION_TYPE_UNSPECIFIED: LoopConditionType
LOOP_CONDITION_TYPE_VARIABLE: LoopConditionType
LOOP_CONDITION_TYPE_EXPRESSION: LoopConditionType
LOOP_CONDITION_OPERATOR_UNSPECIFIED: LoopConditionOperator
LOOP_CONDITION_OPERATOR_TRUTHY: LoopConditionOperator
LOOP_CONDITION_OPERATOR_EQUALS: LoopConditionOperator
LOOP_CONDITION_OPERATOR_NOT_EQUALS: LoopConditionOperator
LOOP_CONDITION_OPERATOR_CONTAINS: LoopConditionOperator
LOOP_CONDITION_OPERATOR_STARTS_WITH: LoopConditionOperator
LOOP_CONDITION_OPERATOR_ENDS_WITH: LoopConditionOperator
LOOP_CONDITION_OPERATOR_GT: LoopConditionOperator
LOOP_CONDITION_OPERATOR_GTE: LoopConditionOperator
LOOP_CONDITION_OPERATOR_LT: LoopConditionOperator
LOOP_CONDITION_OPERATOR_LTE: LoopConditionOperator
CONDITIONAL_TYPE_UNSPECIFIED: ConditionalType
CONDITIONAL_TYPE_EXPRESSION: ConditionalType
CONDITIONAL_TYPE_ELEMENT: ConditionalType
CONDITIONAL_TYPE_VARIABLE: ConditionalType
CONDITIONAL_OPERATOR_UNSPECIFIED: ConditionalOperator
CONDITIONAL_OPERATOR_EQUALS: ConditionalOperator
CONDITIONAL_OPERATOR_NOT_EQUALS: ConditionalOperator
CONDITIONAL_OPERATOR_CONTAINS: ConditionalOperator
CONDITIONAL_OPERATOR_STARTS_WITH: ConditionalOperator
CONDITIONAL_OPERATOR_ENDS_WITH: ConditionalOperator
CONDITIONAL_OPERATOR_GT: ConditionalOperator
CONDITIONAL_OPERATOR_GTE: ConditionalOperator
CONDITIONAL_OPERATOR_LT: ConditionalOperator
CONDITIONAL_OPERATOR_LTE: ConditionalOperator

class NavigateParams(_message.Message):
    __slots__ = ()
    URL_FIELD_NUMBER: _ClassVar[int]
    WAIT_FOR_SELECTOR_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    WAIT_UNTIL_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_TYPE_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_PATH_FIELD_NUMBER: _ClassVar[int]
    url: str
    wait_for_selector: str
    timeout_ms: int
    wait_until: NavigateWaitEvent
    destination_type: NavigateDestinationType
    scenario: str
    scenario_path: str
    def __init__(self, url: _Optional[str] = ..., wait_for_selector: _Optional[str] = ..., timeout_ms: _Optional[int] = ..., wait_until: _Optional[_Union[NavigateWaitEvent, str]] = ..., destination_type: _Optional[_Union[NavigateDestinationType, str]] = ..., scenario: _Optional[str] = ..., scenario_path: _Optional[str] = ...) -> None: ...

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

class SubflowParams(_message.Message):
    __slots__ = ()
    class ArgsEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_PATH_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_VERSION_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    workflow_id: str
    workflow_path: str
    workflow_version: int
    args: _containers.MessageMap[str, _types_pb2.JsonValue]
    def __init__(self, workflow_id: _Optional[str] = ..., workflow_path: _Optional[str] = ..., workflow_version: _Optional[int] = ..., args: _Optional[_Mapping[str, _types_pb2.JsonValue]] = ...) -> None: ...

class ExtractParams(_message.Message):
    __slots__ = ()
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    EXTRACT_TYPE_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTE_NAME_FIELD_NUMBER: _ClassVar[int]
    PROPERTY_NAME_FIELD_NUMBER: _ClassVar[int]
    STORE_AS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    selector: str
    extract_type: ExtractType
    attribute_name: str
    property_name: str
    store_as: str
    timeout_ms: int
    def __init__(self, selector: _Optional[str] = ..., extract_type: _Optional[_Union[ExtractType, str]] = ..., attribute_name: _Optional[str] = ..., property_name: _Optional[str] = ..., store_as: _Optional[str] = ..., timeout_ms: _Optional[int] = ...) -> None: ...

class UploadFileParams(_message.Message):
    __slots__ = ()
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    FILE_PATHS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    selector: str
    file_paths: _containers.RepeatedScalarFieldContainer[str]
    timeout_ms: int
    def __init__(self, selector: _Optional[str] = ..., file_paths: _Optional[_Iterable[str]] = ..., timeout_ms: _Optional[int] = ...) -> None: ...

class DownloadParams(_message.Message):
    __slots__ = ()
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    SAVE_PATH_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    selector: str
    url: str
    save_path: str
    timeout_ms: int
    def __init__(self, selector: _Optional[str] = ..., url: _Optional[str] = ..., save_path: _Optional[str] = ..., timeout_ms: _Optional[int] = ...) -> None: ...

class FrameSwitchParams(_message.Message):
    __slots__ = ()
    ACTION_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    FRAME_ID_FIELD_NUMBER: _ClassVar[int]
    FRAME_URL_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    action: FrameSwitchAction
    selector: str
    frame_id: str
    frame_url: str
    timeout_ms: int
    def __init__(self, action: _Optional[_Union[FrameSwitchAction, str]] = ..., selector: _Optional[str] = ..., frame_id: _Optional[str] = ..., frame_url: _Optional[str] = ..., timeout_ms: _Optional[int] = ...) -> None: ...

class TabSwitchParams(_message.Message):
    __slots__ = ()
    ACTION_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    INDEX_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    URL_PATTERN_FIELD_NUMBER: _ClassVar[int]
    action: TabSwitchAction
    url: str
    index: int
    title: str
    url_pattern: str
    def __init__(self, action: _Optional[_Union[TabSwitchAction, str]] = ..., url: _Optional[str] = ..., index: _Optional[int] = ..., title: _Optional[str] = ..., url_pattern: _Optional[str] = ...) -> None: ...

class CookieOptions(_message.Message):
    __slots__ = ()
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_FIELD_NUMBER: _ClassVar[int]
    HTTP_ONLY_FIELD_NUMBER: _ClassVar[int]
    SECURE_FIELD_NUMBER: _ClassVar[int]
    SAME_SITE_FIELD_NUMBER: _ClassVar[int]
    domain: str
    path: str
    expires: int
    http_only: bool
    secure: bool
    same_site: CookieSameSite
    def __init__(self, domain: _Optional[str] = ..., path: _Optional[str] = ..., expires: _Optional[int] = ..., http_only: _Optional[bool] = ..., secure: _Optional[bool] = ..., same_site: _Optional[_Union[CookieSameSite, str]] = ...) -> None: ...

class CookieStorageParams(_message.Message):
    __slots__ = ()
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    STORAGE_TYPE_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    COOKIE_OPTIONS_FIELD_NUMBER: _ClassVar[int]
    operation: CookieOperation
    storage_type: StorageType
    key: str
    value: str
    cookie_options: CookieOptions
    def __init__(self, operation: _Optional[_Union[CookieOperation, str]] = ..., storage_type: _Optional[_Union[StorageType, str]] = ..., key: _Optional[str] = ..., value: _Optional[str] = ..., cookie_options: _Optional[_Union[CookieOptions, _Mapping]] = ...) -> None: ...

class ShortcutParams(_message.Message):
    __slots__ = ()
    SHORTCUT_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    shortcut: str
    selector: str
    def __init__(self, shortcut: _Optional[str] = ..., selector: _Optional[str] = ...) -> None: ...

class DragDropParams(_message.Message):
    __slots__ = ()
    SOURCE_SELECTOR_FIELD_NUMBER: _ClassVar[int]
    TARGET_SELECTOR_FIELD_NUMBER: _ClassVar[int]
    OFFSET_X_FIELD_NUMBER: _ClassVar[int]
    OFFSET_Y_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    DELAY_MS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    source_selector: str
    target_selector: str
    offset_x: int
    offset_y: int
    steps: int
    delay_ms: int
    timeout_ms: int
    def __init__(self, source_selector: _Optional[str] = ..., target_selector: _Optional[str] = ..., offset_x: _Optional[int] = ..., offset_y: _Optional[int] = ..., steps: _Optional[int] = ..., delay_ms: _Optional[int] = ..., timeout_ms: _Optional[int] = ...) -> None: ...

class GestureParams(_message.Message):
    __slots__ = ()
    GESTURE_TYPE_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    DIRECTION_FIELD_NUMBER: _ClassVar[int]
    DISTANCE_FIELD_NUMBER: _ClassVar[int]
    SCALE_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    gesture_type: GestureType
    selector: str
    direction: SwipeDirection
    distance: int
    scale: float
    duration_ms: int
    def __init__(self, gesture_type: _Optional[_Union[GestureType, str]] = ..., selector: _Optional[str] = ..., direction: _Optional[_Union[SwipeDirection, str]] = ..., distance: _Optional[int] = ..., scale: _Optional[float] = ..., duration_ms: _Optional[int] = ...) -> None: ...

class NetworkMockParams(_message.Message):
    __slots__ = ()
    class HeadersEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    URL_PATTERN_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    STATUS_CODE_FIELD_NUMBER: _ClassVar[int]
    HEADERS_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    DELAY_MS_FIELD_NUMBER: _ClassVar[int]
    operation: NetworkMockOperation
    url_pattern: str
    method: str
    status_code: int
    headers: _containers.ScalarMap[str, str]
    body: str
    delay_ms: int
    def __init__(self, operation: _Optional[_Union[NetworkMockOperation, str]] = ..., url_pattern: _Optional[str] = ..., method: _Optional[str] = ..., status_code: _Optional[int] = ..., headers: _Optional[_Mapping[str, str]] = ..., body: _Optional[str] = ..., delay_ms: _Optional[int] = ...) -> None: ...

class RotateParams(_message.Message):
    __slots__ = ()
    ORIENTATION_FIELD_NUMBER: _ClassVar[int]
    ANGLE_FIELD_NUMBER: _ClassVar[int]
    orientation: DeviceOrientation
    angle: int
    def __init__(self, orientation: _Optional[_Union[DeviceOrientation, str]] = ..., angle: _Optional[int] = ...) -> None: ...

class SetVariableParams(_message.Message):
    __slots__ = ()
    NAME_FIELD_NUMBER: _ClassVar[int]
    SOURCE_TYPE_FIELD_NUMBER: _ClassVar[int]
    VALUE_TYPE_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    EXPRESSION_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    EXTRACT_TYPE_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTE_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    ALL_MATCHES_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    name: str
    source_type: SetVariableSourceType
    value_type: SetVariableValueType
    value: _types_pb2.JsonValue
    expression: str
    selector: str
    extract_type: SetVariableExtractType
    attribute: str
    timeout_ms: int
    all_matches: bool
    url: str
    def __init__(self, name: _Optional[str] = ..., source_type: _Optional[_Union[SetVariableSourceType, str]] = ..., value_type: _Optional[_Union[SetVariableValueType, str]] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ..., expression: _Optional[str] = ..., selector: _Optional[str] = ..., extract_type: _Optional[_Union[SetVariableExtractType, str]] = ..., attribute: _Optional[str] = ..., timeout_ms: _Optional[int] = ..., all_matches: _Optional[bool] = ..., url: _Optional[str] = ...) -> None: ...

class LoopCondition(_message.Message):
    __slots__ = ()
    TYPE_FIELD_NUMBER: _ClassVar[int]
    VARIABLE_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    EXPRESSION_FIELD_NUMBER: _ClassVar[int]
    type: LoopConditionType
    variable: str
    operator: LoopConditionOperator
    value: _types_pb2.JsonValue
    expression: str
    def __init__(self, type: _Optional[_Union[LoopConditionType, str]] = ..., variable: _Optional[str] = ..., operator: _Optional[_Union[LoopConditionOperator, str]] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ..., expression: _Optional[str] = ...) -> None: ...

class LoopParams(_message.Message):
    __slots__ = ()
    LOOP_TYPE_FIELD_NUMBER: _ClassVar[int]
    ARRAY_SOURCE_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    MAX_ITERATIONS_FIELD_NUMBER: _ClassVar[int]
    ITEM_VARIABLE_FIELD_NUMBER: _ClassVar[int]
    INDEX_VARIABLE_FIELD_NUMBER: _ClassVar[int]
    CONDITION_FIELD_NUMBER: _ClassVar[int]
    ITERATION_TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    loop_type: LoopType
    array_source: str
    count: int
    max_iterations: int
    item_variable: str
    index_variable: str
    condition: LoopCondition
    iteration_timeout_ms: int
    total_timeout_ms: int
    def __init__(self, loop_type: _Optional[_Union[LoopType, str]] = ..., array_source: _Optional[str] = ..., count: _Optional[int] = ..., max_iterations: _Optional[int] = ..., item_variable: _Optional[str] = ..., index_variable: _Optional[str] = ..., condition: _Optional[_Union[LoopCondition, _Mapping]] = ..., iteration_timeout_ms: _Optional[int] = ..., total_timeout_ms: _Optional[int] = ...) -> None: ...

class ConditionalParams(_message.Message):
    __slots__ = ()
    CONDITION_TYPE_FIELD_NUMBER: _ClassVar[int]
    EXPRESSION_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    VARIABLE_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    NEGATE_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    POLL_INTERVAL_MS_FIELD_NUMBER: _ClassVar[int]
    condition_type: ConditionalType
    expression: str
    selector: str
    variable: str
    operator: ConditionalOperator
    value: _types_pb2.JsonValue
    negate: bool
    timeout_ms: int
    poll_interval_ms: int
    def __init__(self, condition_type: _Optional[_Union[ConditionalType, str]] = ..., expression: _Optional[str] = ..., selector: _Optional[str] = ..., variable: _Optional[str] = ..., operator: _Optional[_Union[ConditionalOperator, str]] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ..., negate: _Optional[bool] = ..., timeout_ms: _Optional[int] = ..., poll_interval_ms: _Optional[int] = ...) -> None: ...

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
    SUBFLOW_FIELD_NUMBER: _ClassVar[int]
    EXTRACT_FIELD_NUMBER: _ClassVar[int]
    UPLOAD_FILE_FIELD_NUMBER: _ClassVar[int]
    DOWNLOAD_FIELD_NUMBER: _ClassVar[int]
    FRAME_SWITCH_FIELD_NUMBER: _ClassVar[int]
    TAB_SWITCH_FIELD_NUMBER: _ClassVar[int]
    COOKIE_STORAGE_FIELD_NUMBER: _ClassVar[int]
    SHORTCUT_FIELD_NUMBER: _ClassVar[int]
    DRAG_DROP_FIELD_NUMBER: _ClassVar[int]
    GESTURE_FIELD_NUMBER: _ClassVar[int]
    NETWORK_MOCK_FIELD_NUMBER: _ClassVar[int]
    ROTATE_FIELD_NUMBER: _ClassVar[int]
    SET_VARIABLE_FIELD_NUMBER: _ClassVar[int]
    LOOP_FIELD_NUMBER: _ClassVar[int]
    CONDITIONAL_FIELD_NUMBER: _ClassVar[int]
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
    subflow: SubflowParams
    extract: ExtractParams
    upload_file: UploadFileParams
    download: DownloadParams
    frame_switch: FrameSwitchParams
    tab_switch: TabSwitchParams
    cookie_storage: CookieStorageParams
    shortcut: ShortcutParams
    drag_drop: DragDropParams
    gesture: GestureParams
    network_mock: NetworkMockParams
    rotate: RotateParams
    set_variable: SetVariableParams
    loop: LoopParams
    conditional: ConditionalParams
    metadata: ActionMetadata
    def __init__(self, type: _Optional[_Union[ActionType, str]] = ..., navigate: _Optional[_Union[NavigateParams, _Mapping]] = ..., click: _Optional[_Union[ClickParams, _Mapping]] = ..., input: _Optional[_Union[InputParams, _Mapping]] = ..., wait: _Optional[_Union[WaitParams, _Mapping]] = ..., scroll: _Optional[_Union[ScrollParams, _Mapping]] = ..., select_option: _Optional[_Union[SelectParams, _Mapping]] = ..., evaluate: _Optional[_Union[EvaluateParams, _Mapping]] = ..., keyboard: _Optional[_Union[KeyboardParams, _Mapping]] = ..., hover: _Optional[_Union[HoverParams, _Mapping]] = ..., screenshot: _Optional[_Union[ScreenshotParams, _Mapping]] = ..., focus: _Optional[_Union[FocusParams, _Mapping]] = ..., blur: _Optional[_Union[BlurParams, _Mapping]] = ..., subflow: _Optional[_Union[SubflowParams, _Mapping]] = ..., extract: _Optional[_Union[ExtractParams, _Mapping]] = ..., upload_file: _Optional[_Union[UploadFileParams, _Mapping]] = ..., download: _Optional[_Union[DownloadParams, _Mapping]] = ..., frame_switch: _Optional[_Union[FrameSwitchParams, _Mapping]] = ..., tab_switch: _Optional[_Union[TabSwitchParams, _Mapping]] = ..., cookie_storage: _Optional[_Union[CookieStorageParams, _Mapping]] = ..., shortcut: _Optional[_Union[ShortcutParams, _Mapping]] = ..., drag_drop: _Optional[_Union[DragDropParams, _Mapping]] = ..., gesture: _Optional[_Union[GestureParams, _Mapping]] = ..., network_mock: _Optional[_Union[NetworkMockParams, _Mapping]] = ..., rotate: _Optional[_Union[RotateParams, _Mapping]] = ..., set_variable: _Optional[_Union[SetVariableParams, _Mapping]] = ..., loop: _Optional[_Union[LoopParams, _Mapping]] = ..., conditional: _Optional[_Union[ConditionalParams, _Mapping]] = ..., metadata: _Optional[_Union[ActionMetadata, _Mapping]] = ..., **kwargs) -> None: ...
