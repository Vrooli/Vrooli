import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ContainerStyle(_message.Message):
    __slots__ = ("id", "name", "shape", "corner_ratio", "background_kind", "background_top", "background_bottom", "mark_scale", "maskable_scale", "accent_color", "glow", "small_mark_threshold_px", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SHAPE_FIELD_NUMBER: _ClassVar[int]
    CORNER_RATIO_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_KIND_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_TOP_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_BOTTOM_FIELD_NUMBER: _ClassVar[int]
    MARK_SCALE_FIELD_NUMBER: _ClassVar[int]
    MASKABLE_SCALE_FIELD_NUMBER: _ClassVar[int]
    ACCENT_COLOR_FIELD_NUMBER: _ClassVar[int]
    GLOW_FIELD_NUMBER: _ClassVar[int]
    SMALL_MARK_THRESHOLD_PX_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    shape: str
    corner_ratio: float
    background_kind: str
    background_top: str
    background_bottom: str
    mark_scale: float
    maskable_scale: float
    accent_color: str
    glow: _containers.RepeatedCompositeFieldContainer[GlowLayer]
    small_mark_threshold_px: int
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., shape: _Optional[str] = ..., corner_ratio: _Optional[float] = ..., background_kind: _Optional[str] = ..., background_top: _Optional[str] = ..., background_bottom: _Optional[str] = ..., mark_scale: _Optional[float] = ..., maskable_scale: _Optional[float] = ..., accent_color: _Optional[str] = ..., glow: _Optional[_Iterable[_Union[GlowLayer, _Mapping]]] = ..., small_mark_threshold_px: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GlowLayer(_message.Message):
    __slots__ = ("width", "opacity")
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    OPACITY_FIELD_NUMBER: _ClassVar[int]
    width: float
    opacity: float
    def __init__(self, width: _Optional[float] = ..., opacity: _Optional[float] = ...) -> None: ...

class ProductLine(_message.Message):
    __slots__ = ("id", "name", "container_style_id", "products", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    CONTAINER_STYLE_ID_FIELD_NUMBER: _ClassVar[int]
    PRODUCTS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    container_style_id: str
    products: _containers.RepeatedScalarFieldContainer[str]
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., container_style_id: _Optional[str] = ..., products: _Optional[_Iterable[str]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListContainerStylesRequest(_message.Message):
    __slots__ = ("name_contains", "limit", "offset")
    NAME_CONTAINS_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    name_contains: str
    limit: int
    offset: int
    def __init__(self, name_contains: _Optional[str] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListContainerStylesResponse(_message.Message):
    __slots__ = ("styles",)
    STYLES_FIELD_NUMBER: _ClassVar[int]
    styles: _containers.RepeatedCompositeFieldContainer[ContainerStyle]
    def __init__(self, styles: _Optional[_Iterable[_Union[ContainerStyle, _Mapping]]] = ...) -> None: ...

class GetContainerStyleRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetContainerStyleResponse(_message.Message):
    __slots__ = ("style",)
    STYLE_FIELD_NUMBER: _ClassVar[int]
    style: ContainerStyle
    def __init__(self, style: _Optional[_Union[ContainerStyle, _Mapping]] = ...) -> None: ...

class CreateContainerStyleRequest(_message.Message):
    __slots__ = ("name", "shape", "corner_ratio", "background_kind", "background_top", "background_bottom", "mark_scale", "maskable_scale", "accent_color", "glow", "small_mark_threshold_px")
    NAME_FIELD_NUMBER: _ClassVar[int]
    SHAPE_FIELD_NUMBER: _ClassVar[int]
    CORNER_RATIO_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_KIND_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_TOP_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_BOTTOM_FIELD_NUMBER: _ClassVar[int]
    MARK_SCALE_FIELD_NUMBER: _ClassVar[int]
    MASKABLE_SCALE_FIELD_NUMBER: _ClassVar[int]
    ACCENT_COLOR_FIELD_NUMBER: _ClassVar[int]
    GLOW_FIELD_NUMBER: _ClassVar[int]
    SMALL_MARK_THRESHOLD_PX_FIELD_NUMBER: _ClassVar[int]
    name: str
    shape: str
    corner_ratio: float
    background_kind: str
    background_top: str
    background_bottom: str
    mark_scale: float
    maskable_scale: float
    accent_color: str
    glow: _containers.RepeatedCompositeFieldContainer[GlowLayer]
    small_mark_threshold_px: int
    def __init__(self, name: _Optional[str] = ..., shape: _Optional[str] = ..., corner_ratio: _Optional[float] = ..., background_kind: _Optional[str] = ..., background_top: _Optional[str] = ..., background_bottom: _Optional[str] = ..., mark_scale: _Optional[float] = ..., maskable_scale: _Optional[float] = ..., accent_color: _Optional[str] = ..., glow: _Optional[_Iterable[_Union[GlowLayer, _Mapping]]] = ..., small_mark_threshold_px: _Optional[int] = ...) -> None: ...

class CreateContainerStyleResponse(_message.Message):
    __slots__ = ("style",)
    STYLE_FIELD_NUMBER: _ClassVar[int]
    style: ContainerStyle
    def __init__(self, style: _Optional[_Union[ContainerStyle, _Mapping]] = ...) -> None: ...

class UpdateContainerStyleRequest(_message.Message):
    __slots__ = ("id", "name", "shape", "corner_ratio", "background_kind", "background_top", "background_bottom", "mark_scale", "maskable_scale", "accent_color", "glow", "small_mark_threshold_px")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SHAPE_FIELD_NUMBER: _ClassVar[int]
    CORNER_RATIO_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_KIND_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_TOP_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_BOTTOM_FIELD_NUMBER: _ClassVar[int]
    MARK_SCALE_FIELD_NUMBER: _ClassVar[int]
    MASKABLE_SCALE_FIELD_NUMBER: _ClassVar[int]
    ACCENT_COLOR_FIELD_NUMBER: _ClassVar[int]
    GLOW_FIELD_NUMBER: _ClassVar[int]
    SMALL_MARK_THRESHOLD_PX_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    shape: str
    corner_ratio: float
    background_kind: str
    background_top: str
    background_bottom: str
    mark_scale: float
    maskable_scale: float
    accent_color: str
    glow: _containers.RepeatedCompositeFieldContainer[GlowLayer]
    small_mark_threshold_px: int
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., shape: _Optional[str] = ..., corner_ratio: _Optional[float] = ..., background_kind: _Optional[str] = ..., background_top: _Optional[str] = ..., background_bottom: _Optional[str] = ..., mark_scale: _Optional[float] = ..., maskable_scale: _Optional[float] = ..., accent_color: _Optional[str] = ..., glow: _Optional[_Iterable[_Union[GlowLayer, _Mapping]]] = ..., small_mark_threshold_px: _Optional[int] = ...) -> None: ...

class UpdateContainerStyleResponse(_message.Message):
    __slots__ = ("style",)
    STYLE_FIELD_NUMBER: _ClassVar[int]
    style: ContainerStyle
    def __init__(self, style: _Optional[_Union[ContainerStyle, _Mapping]] = ...) -> None: ...

class ListProductLinesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListProductLinesResponse(_message.Message):
    __slots__ = ("lines",)
    LINES_FIELD_NUMBER: _ClassVar[int]
    lines: _containers.RepeatedCompositeFieldContainer[ProductLine]
    def __init__(self, lines: _Optional[_Iterable[_Union[ProductLine, _Mapping]]] = ...) -> None: ...

class CreateProductLineRequest(_message.Message):
    __slots__ = ("name", "container_style_id", "products")
    NAME_FIELD_NUMBER: _ClassVar[int]
    CONTAINER_STYLE_ID_FIELD_NUMBER: _ClassVar[int]
    PRODUCTS_FIELD_NUMBER: _ClassVar[int]
    name: str
    container_style_id: str
    products: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., container_style_id: _Optional[str] = ..., products: _Optional[_Iterable[str]] = ...) -> None: ...

class CreateProductLineResponse(_message.Message):
    __slots__ = ("line",)
    LINE_FIELD_NUMBER: _ClassVar[int]
    line: ProductLine
    def __init__(self, line: _Optional[_Union[ProductLine, _Mapping]] = ...) -> None: ...
