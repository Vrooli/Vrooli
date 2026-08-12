from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Region(_message.Message):
    __slots__ = ("x", "y", "width", "height", "kind", "text_color")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    TEXT_COLOR_FIELD_NUMBER: _ClassVar[int]
    x: float
    y: float
    width: float
    height: float
    kind: str
    text_color: str
    def __init__(self, x: _Optional[float] = ..., y: _Optional[float] = ..., width: _Optional[float] = ..., height: _Optional[float] = ..., kind: _Optional[str] = ..., text_color: _Optional[str] = ...) -> None: ...

class MeasureRequest(_message.Message):
    __slots__ = ("image_png", "regions", "threshold", "placement")
    IMAGE_PNG_FIELD_NUMBER: _ClassVar[int]
    REGIONS_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    PLACEMENT_FIELD_NUMBER: _ClassVar[int]
    image_png: bytes
    regions: _containers.RepeatedCompositeFieldContainer[Region]
    threshold: float
    placement: str
    def __init__(self, image_png: _Optional[bytes] = ..., regions: _Optional[_Iterable[_Union[Region, _Mapping]]] = ..., threshold: _Optional[float] = ..., placement: _Optional[str] = ...) -> None: ...

class Amendment(_message.Message):
    __slots__ = ("kind", "description", "value")
    KIND_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    kind: str
    description: str
    value: float
    def __init__(self, kind: _Optional[str] = ..., description: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...

class RegionVerdict(_message.Message):
    __slots__ = ("region_index", "minimum_ratio", "passes")
    REGION_INDEX_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_RATIO_FIELD_NUMBER: _ClassVar[int]
    PASSES_FIELD_NUMBER: _ClassVar[int]
    region_index: int
    minimum_ratio: float
    passes: bool
    def __init__(self, region_index: _Optional[int] = ..., minimum_ratio: _Optional[float] = ..., passes: _Optional[bool] = ...) -> None: ...

class Verdict(_message.Message):
    __slots__ = ("passes", "minimum_ratio", "threshold", "regions", "amendments", "placement")
    PASSES_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_RATIO_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    REGIONS_FIELD_NUMBER: _ClassVar[int]
    AMENDMENTS_FIELD_NUMBER: _ClassVar[int]
    PLACEMENT_FIELD_NUMBER: _ClassVar[int]
    passes: bool
    minimum_ratio: float
    threshold: float
    regions: _containers.RepeatedCompositeFieldContainer[RegionVerdict]
    amendments: _containers.RepeatedCompositeFieldContainer[Amendment]
    placement: str
    def __init__(self, passes: _Optional[bool] = ..., minimum_ratio: _Optional[float] = ..., threshold: _Optional[float] = ..., regions: _Optional[_Iterable[_Union[RegionVerdict, _Mapping]]] = ..., amendments: _Optional[_Iterable[_Union[Amendment, _Mapping]]] = ..., placement: _Optional[str] = ...) -> None: ...
