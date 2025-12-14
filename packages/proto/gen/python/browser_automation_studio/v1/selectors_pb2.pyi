from browser_automation_studio.v1 import shared_pb2 as _shared_pb2
from browser_automation_studio.v1 import geometry_pb2 as _geometry_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SelectorCandidate(_message.Message):
    __slots__ = ()
    TYPE_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    SPECIFICITY_FIELD_NUMBER: _ClassVar[int]
    type: _shared_pb2.SelectorType
    value: str
    confidence: float
    specificity: int
    def __init__(self, type: _Optional[_Union[_shared_pb2.SelectorType, str]] = ..., value: _Optional[str] = ..., confidence: _Optional[float] = ..., specificity: _Optional[int] = ...) -> None: ...

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
    HIGHLIGHT_COLOR_FIELD_NUMBER: _ClassVar[int]
    CUSTOM_RGBA_FIELD_NUMBER: _ClassVar[int]
    selector: str
    bounding_box: _geometry_pb2.BoundingBox
    padding: int
    color: str
    highlight_color: _shared_pb2.HighlightColor
    custom_rgba: str
    def __init__(self, selector: _Optional[str] = ..., bounding_box: _Optional[_Union[_geometry_pb2.BoundingBox, _Mapping]] = ..., padding: _Optional[int] = ..., color: _Optional[str] = ..., highlight_color: _Optional[_Union[_shared_pb2.HighlightColor, str]] = ..., custom_rgba: _Optional[str] = ...) -> None: ...

class MaskRegion(_message.Message):
    __slots__ = ()
    SELECTOR_FIELD_NUMBER: _ClassVar[int]
    BOUNDING_BOX_FIELD_NUMBER: _ClassVar[int]
    OPACITY_FIELD_NUMBER: _ClassVar[int]
    selector: str
    bounding_box: _geometry_pb2.BoundingBox
    opacity: float
    def __init__(self, selector: _Optional[str] = ..., bounding_box: _Optional[_Union[_geometry_pb2.BoundingBox, _Mapping]] = ..., opacity: _Optional[float] = ...) -> None: ...
