import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Viewport(_message.Message):
    __slots__ = ("width", "height")
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    width: int
    height: int
    def __init__(self, width: _Optional[int] = ..., height: _Optional[int] = ...) -> None: ...

class Rectangle(_message.Message):
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

class SelectorOption(_message.Message):
    __slots__ = ("selector", "type", "robustness", "fallback")
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    ROBUSTNESS_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_FIELD_NUMBER: _ClassVar[int]
    selector: str
    type: str
    robustness: float
    fallback: bool
    def __init__(self, selector: _Optional[str] = ..., type: _Optional[str] = ..., robustness: _Optional[float] = ..., fallback: _Optional[bool] = ...) -> None: ...

class ElementInfo(_message.Message):
    __slots__ = ("text", "tag_name", "type", "selectors", "bounding_box", "confidence", "category", "attributes")
    class AttributesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TEXT_FIELD_NUMBER: _ClassVar[int]
    TAG_NAME_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    SELECTORS_FIELD_NUMBER: _ClassVar[int]
    BOUNDING_BOX_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTES_FIELD_NUMBER: _ClassVar[int]
    text: str
    tag_name: str
    type: str
    selectors: _containers.RepeatedCompositeFieldContainer[SelectorOption]
    bounding_box: Rectangle
    confidence: float
    category: str
    attributes: _containers.ScalarMap[str, str]
    def __init__(self, text: _Optional[str] = ..., tag_name: _Optional[str] = ..., type: _Optional[str] = ..., selectors: _Optional[_Iterable[_Union[SelectorOption, _Mapping]]] = ..., bounding_box: _Optional[_Union[Rectangle, _Mapping]] = ..., confidence: _Optional[float] = ..., category: _Optional[str] = ..., attributes: _Optional[_Mapping[str, str]] = ...) -> None: ...

class PageContext(_message.Message):
    __slots__ = ("title", "url", "has_login", "has_search", "form_count", "button_count", "link_count")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    HAS_LOGIN_FIELD_NUMBER: _ClassVar[int]
    HAS_SEARCH_FIELD_NUMBER: _ClassVar[int]
    FORM_COUNT_FIELD_NUMBER: _ClassVar[int]
    BUTTON_COUNT_FIELD_NUMBER: _ClassVar[int]
    LINK_COUNT_FIELD_NUMBER: _ClassVar[int]
    title: str
    url: str
    has_login: bool
    has_search: bool
    form_count: int
    button_count: int
    link_count: int
    def __init__(self, title: _Optional[str] = ..., url: _Optional[str] = ..., has_login: _Optional[bool] = ..., has_search: _Optional[bool] = ..., form_count: _Optional[int] = ..., button_count: _Optional[int] = ..., link_count: _Optional[int] = ...) -> None: ...

class AISuggestion(_message.Message):
    __slots__ = ("action", "description", "element_text", "selector", "confidence", "category", "reasoning")
    ACTION_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ELEMENT_TEXT_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    REASONING_FIELD_NUMBER: _ClassVar[int]
    action: str
    description: str
    element_text: str
    selector: str
    confidence: float
    category: str
    reasoning: str
    def __init__(self, action: _Optional[str] = ..., description: _Optional[str] = ..., element_text: _Optional[str] = ..., selector: _Optional[str] = ..., confidence: _Optional[float] = ..., category: _Optional[str] = ..., reasoning: _Optional[str] = ...) -> None: ...

class ConsoleLog(_message.Message):
    __slots__ = ("level", "message", "timestamp")
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    level: str
    message: str
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, level: _Optional[str] = ..., message: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ElementHierarchyEntry(_message.Message):
    __slots__ = ("element", "selector", "depth", "path", "path_summary")
    ELEMENT_FIELD_NUMBER: _ClassVar[int]
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    DEPTH_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    PATH_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    element: ElementInfo
    selector: str
    depth: int
    path: _containers.RepeatedScalarFieldContainer[str]
    path_summary: str
    def __init__(self, element: _Optional[_Union[ElementInfo, _Mapping]] = ..., selector: _Optional[str] = ..., depth: _Optional[int] = ..., path: _Optional[_Iterable[str]] = ..., path_summary: _Optional[str] = ...) -> None: ...

class ElementSelectionResult(_message.Message):
    __slots__ = ("element", "candidates", "selected_index")
    ELEMENT_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    SELECTED_INDEX_FIELD_NUMBER: _ClassVar[int]
    element: ElementInfo
    candidates: _containers.RepeatedCompositeFieldContainer[ElementHierarchyEntry]
    selected_index: int
    def __init__(self, element: _Optional[_Union[ElementInfo, _Mapping]] = ..., candidates: _Optional[_Iterable[_Union[ElementHierarchyEntry, _Mapping]]] = ..., selected_index: _Optional[int] = ...) -> None: ...

class TakePreviewScreenshotRequest(_message.Message):
    __slots__ = ("url", "viewport")
    URL_FIELD_NUMBER: _ClassVar[int]
    VIEWPORT_FIELD_NUMBER: _ClassVar[int]
    url: str
    viewport: Viewport
    def __init__(self, url: _Optional[str] = ..., viewport: _Optional[_Union[Viewport, _Mapping]] = ...) -> None: ...

class TakePreviewScreenshotResponse(_message.Message):
    __slots__ = ("screenshot_png", "content_type", "console_logs", "url", "captured_at", "duration_ms", "viewport_width", "viewport_height", "events")
    SCREENSHOT_PNG_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    CONSOLE_LOGS_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    VIEWPORT_WIDTH_FIELD_NUMBER: _ClassVar[int]
    VIEWPORT_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    screenshot_png: bytes
    content_type: str
    console_logs: _containers.RepeatedCompositeFieldContainer[ConsoleLog]
    url: str
    captured_at: _timestamp_pb2.Timestamp
    duration_ms: int
    viewport_width: int
    viewport_height: int
    events: _containers.RepeatedCompositeFieldContainer[_struct_pb2.Struct]
    def __init__(self, screenshot_png: _Optional[bytes] = ..., content_type: _Optional[str] = ..., console_logs: _Optional[_Iterable[_Union[ConsoleLog, _Mapping]]] = ..., url: _Optional[str] = ..., captured_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., duration_ms: _Optional[int] = ..., viewport_width: _Optional[int] = ..., viewport_height: _Optional[int] = ..., events: _Optional[_Iterable[_Union[_struct_pb2.Struct, _Mapping]]] = ...) -> None: ...

class GetLinkPreviewRequest(_message.Message):
    __slots__ = ("url",)
    URL_FIELD_NUMBER: _ClassVar[int]
    url: str
    def __init__(self, url: _Optional[str] = ...) -> None: ...

class GetLinkPreviewResponse(_message.Message):
    __slots__ = ("found", "title", "description", "image", "favicon", "site_name")
    FOUND_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    IMAGE_FIELD_NUMBER: _ClassVar[int]
    FAVICON_FIELD_NUMBER: _ClassVar[int]
    SITE_NAME_FIELD_NUMBER: _ClassVar[int]
    found: bool
    title: str
    description: str
    image: str
    favicon: str
    site_name: str
    def __init__(self, found: _Optional[bool] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., image: _Optional[str] = ..., favicon: _Optional[str] = ..., site_name: _Optional[str] = ...) -> None: ...

class AnalyzeElementsRequest(_message.Message):
    __slots__ = ("url",)
    URL_FIELD_NUMBER: _ClassVar[int]
    url: str
    def __init__(self, url: _Optional[str] = ...) -> None: ...

class AnalyzeElementsResponse(_message.Message):
    __slots__ = ("success", "elements", "ai_suggestions", "page_context", "screenshot", "captured_at")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ELEMENTS_FIELD_NUMBER: _ClassVar[int]
    AI_SUGGESTIONS_FIELD_NUMBER: _ClassVar[int]
    PAGE_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    SCREENSHOT_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    elements: _containers.RepeatedCompositeFieldContainer[ElementInfo]
    ai_suggestions: _containers.RepeatedCompositeFieldContainer[AISuggestion]
    page_context: PageContext
    screenshot: str
    captured_at: _timestamp_pb2.Timestamp
    def __init__(self, success: _Optional[bool] = ..., elements: _Optional[_Iterable[_Union[ElementInfo, _Mapping]]] = ..., ai_suggestions: _Optional[_Iterable[_Union[AISuggestion, _Mapping]]] = ..., page_context: _Optional[_Union[PageContext, _Mapping]] = ..., screenshot: _Optional[str] = ..., captured_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetElementAtCoordinateRequest(_message.Message):
    __slots__ = ("url", "x", "y")
    URL_FIELD_NUMBER: _ClassVar[int]
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    url: str
    x: int
    y: int
    def __init__(self, url: _Optional[str] = ..., x: _Optional[int] = ..., y: _Optional[int] = ...) -> None: ...

class GetElementAtCoordinateResponse(_message.Message):
    __slots__ = ("selection",)
    SELECTION_FIELD_NUMBER: _ClassVar[int]
    selection: ElementSelectionResult
    def __init__(self, selection: _Optional[_Union[ElementSelectionResult, _Mapping]] = ...) -> None: ...

class AIAnalyzeElementsRequest(_message.Message):
    __slots__ = ("url", "intent", "screenshot")
    URL_FIELD_NUMBER: _ClassVar[int]
    INTENT_FIELD_NUMBER: _ClassVar[int]
    SCREENSHOT_FIELD_NUMBER: _ClassVar[int]
    url: str
    intent: str
    screenshot: str
    def __init__(self, url: _Optional[str] = ..., intent: _Optional[str] = ..., screenshot: _Optional[str] = ...) -> None: ...

class AIAnalyzeElementsResponse(_message.Message):
    __slots__ = ("suggestions",)
    SUGGESTIONS_FIELD_NUMBER: _ClassVar[int]
    suggestions: _containers.RepeatedCompositeFieldContainer[ElementInfo]
    def __init__(self, suggestions: _Optional[_Iterable[_Union[ElementInfo, _Mapping]]] = ...) -> None: ...

class GetDOMTreeRequest(_message.Message):
    __slots__ = ("url",)
    URL_FIELD_NUMBER: _ClassVar[int]
    url: str
    def __init__(self, url: _Optional[str] = ...) -> None: ...

class GetDOMTreeResponse(_message.Message):
    __slots__ = ("tree",)
    TREE_FIELD_NUMBER: _ClassVar[int]
    tree: _struct_pb2.Struct
    def __init__(self, tree: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...
