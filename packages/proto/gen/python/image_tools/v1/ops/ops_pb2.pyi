from image_tools.v1.jobs import jobs_pb2 as _jobs_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class OperationInfo(_message.Message):
    __slots__ = ("name", "category", "summary")
    NAME_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    name: str
    category: str
    summary: str
    def __init__(self, name: _Optional[str] = ..., category: _Optional[str] = ..., summary: _Optional[str] = ...) -> None: ...

class ListOperationsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListOperationsResponse(_message.Message):
    __slots__ = ("operations", "decodable_formats", "encodable_formats")
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    DECODABLE_FORMATS_FIELD_NUMBER: _ClassVar[int]
    ENCODABLE_FORMATS_FIELD_NUMBER: _ClassVar[int]
    operations: _containers.RepeatedCompositeFieldContainer[OperationInfo]
    decodable_formats: _containers.RepeatedScalarFieldContainer[str]
    encodable_formats: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, operations: _Optional[_Iterable[_Union[OperationInfo, _Mapping]]] = ..., decodable_formats: _Optional[_Iterable[str]] = ..., encodable_formats: _Optional[_Iterable[str]] = ...) -> None: ...

class OpParams(_message.Message):
    __slots__ = ("resize", "crop", "rotate", "flip", "deskew", "thumbnail", "canvas", "adjust", "filter", "convert", "compress", "overlay", "metadata")
    RESIZE_FIELD_NUMBER: _ClassVar[int]
    CROP_FIELD_NUMBER: _ClassVar[int]
    ROTATE_FIELD_NUMBER: _ClassVar[int]
    FLIP_FIELD_NUMBER: _ClassVar[int]
    DESKEW_FIELD_NUMBER: _ClassVar[int]
    THUMBNAIL_FIELD_NUMBER: _ClassVar[int]
    CANVAS_FIELD_NUMBER: _ClassVar[int]
    ADJUST_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    CONVERT_FIELD_NUMBER: _ClassVar[int]
    COMPRESS_FIELD_NUMBER: _ClassVar[int]
    OVERLAY_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    resize: ResizeParams
    crop: CropParams
    rotate: RotateParams
    flip: FlipParams
    deskew: DeskewParams
    thumbnail: ThumbnailParams
    canvas: CanvasParams
    adjust: AdjustParams
    filter: FilterParams
    convert: ConvertParams
    compress: CompressParams
    overlay: OverlayParams
    metadata: MetadataParams
    def __init__(self, resize: _Optional[_Union[ResizeParams, _Mapping]] = ..., crop: _Optional[_Union[CropParams, _Mapping]] = ..., rotate: _Optional[_Union[RotateParams, _Mapping]] = ..., flip: _Optional[_Union[FlipParams, _Mapping]] = ..., deskew: _Optional[_Union[DeskewParams, _Mapping]] = ..., thumbnail: _Optional[_Union[ThumbnailParams, _Mapping]] = ..., canvas: _Optional[_Union[CanvasParams, _Mapping]] = ..., adjust: _Optional[_Union[AdjustParams, _Mapping]] = ..., filter: _Optional[_Union[FilterParams, _Mapping]] = ..., convert: _Optional[_Union[ConvertParams, _Mapping]] = ..., compress: _Optional[_Union[CompressParams, _Mapping]] = ..., overlay: _Optional[_Union[OverlayParams, _Mapping]] = ..., metadata: _Optional[_Union[MetadataParams, _Mapping]] = ...) -> None: ...

class ResizeParams(_message.Message):
    __slots__ = ("width", "height", "fit", "gravity")
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    FIT_FIELD_NUMBER: _ClassVar[int]
    GRAVITY_FIELD_NUMBER: _ClassVar[int]
    width: int
    height: int
    fit: str
    gravity: str
    def __init__(self, width: _Optional[int] = ..., height: _Optional[int] = ..., fit: _Optional[str] = ..., gravity: _Optional[str] = ...) -> None: ...

class CropParams(_message.Message):
    __slots__ = ("x", "y", "width", "height", "gravity")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    GRAVITY_FIELD_NUMBER: _ClassVar[int]
    x: int
    y: int
    width: int
    height: int
    gravity: str
    def __init__(self, x: _Optional[int] = ..., y: _Optional[int] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., gravity: _Optional[str] = ...) -> None: ...

class RotateParams(_message.Message):
    __slots__ = ("angle", "expand", "background")
    ANGLE_FIELD_NUMBER: _ClassVar[int]
    EXPAND_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_FIELD_NUMBER: _ClassVar[int]
    angle: float
    expand: bool
    background: str
    def __init__(self, angle: _Optional[float] = ..., expand: _Optional[bool] = ..., background: _Optional[str] = ...) -> None: ...

class FlipParams(_message.Message):
    __slots__ = ("axis",)
    AXIS_FIELD_NUMBER: _ClassVar[int]
    axis: str
    def __init__(self, axis: _Optional[str] = ...) -> None: ...

class DeskewParams(_message.Message):
    __slots__ = ("background",)
    BACKGROUND_FIELD_NUMBER: _ClassVar[int]
    background: str
    def __init__(self, background: _Optional[str] = ...) -> None: ...

class ThumbnailParams(_message.Message):
    __slots__ = ("width", "height")
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    width: int
    height: int
    def __init__(self, width: _Optional[int] = ..., height: _Optional[int] = ...) -> None: ...

class CanvasParams(_message.Message):
    __slots__ = ("width", "height", "background", "gravity")
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_FIELD_NUMBER: _ClassVar[int]
    GRAVITY_FIELD_NUMBER: _ClassVar[int]
    width: int
    height: int
    background: str
    gravity: str
    def __init__(self, width: _Optional[int] = ..., height: _Optional[int] = ..., background: _Optional[str] = ..., gravity: _Optional[str] = ...) -> None: ...

class AdjustParams(_message.Message):
    __slots__ = ("brightness", "contrast", "gamma", "saturation", "hue")
    BRIGHTNESS_FIELD_NUMBER: _ClassVar[int]
    CONTRAST_FIELD_NUMBER: _ClassVar[int]
    GAMMA_FIELD_NUMBER: _ClassVar[int]
    SATURATION_FIELD_NUMBER: _ClassVar[int]
    HUE_FIELD_NUMBER: _ClassVar[int]
    brightness: float
    contrast: float
    gamma: float
    saturation: float
    hue: float
    def __init__(self, brightness: _Optional[float] = ..., contrast: _Optional[float] = ..., gamma: _Optional[float] = ..., saturation: _Optional[float] = ..., hue: _Optional[float] = ...) -> None: ...

class FilterParams(_message.Message):
    __slots__ = ("filter", "amount")
    FILTER_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_FIELD_NUMBER: _ClassVar[int]
    filter: str
    amount: float
    def __init__(self, filter: _Optional[str] = ..., amount: _Optional[float] = ...) -> None: ...

class ConvertParams(_message.Message):
    __slots__ = ("format", "quality", "lossless")
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    QUALITY_FIELD_NUMBER: _ClassVar[int]
    LOSSLESS_FIELD_NUMBER: _ClassVar[int]
    format: str
    quality: int
    lossless: bool
    def __init__(self, format: _Optional[str] = ..., quality: _Optional[int] = ..., lossless: _Optional[bool] = ...) -> None: ...

class CompressParams(_message.Message):
    __slots__ = ("format", "quality", "lossless", "target_bytes")
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    QUALITY_FIELD_NUMBER: _ClassVar[int]
    LOSSLESS_FIELD_NUMBER: _ClassVar[int]
    TARGET_BYTES_FIELD_NUMBER: _ClassVar[int]
    format: str
    quality: int
    lossless: bool
    target_bytes: int
    def __init__(self, format: _Optional[str] = ..., quality: _Optional[int] = ..., lossless: _Optional[bool] = ..., target_bytes: _Optional[int] = ...) -> None: ...

class OverlayParams(_message.Message):
    __slots__ = ("text", "position", "opacity", "color", "font_size")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    POSITION_FIELD_NUMBER: _ClassVar[int]
    OPACITY_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    FONT_SIZE_FIELD_NUMBER: _ClassVar[int]
    text: str
    position: str
    opacity: float
    color: str
    font_size: float
    def __init__(self, text: _Optional[str] = ..., position: _Optional[str] = ..., opacity: _Optional[float] = ..., color: _Optional[str] = ..., font_size: _Optional[float] = ...) -> None: ...

class MetadataParams(_message.Message):
    __slots__ = ("strip_all", "strip_gps", "auto_orient")
    STRIP_ALL_FIELD_NUMBER: _ClassVar[int]
    STRIP_GPS_FIELD_NUMBER: _ClassVar[int]
    AUTO_ORIENT_FIELD_NUMBER: _ClassVar[int]
    strip_all: bool
    strip_gps: bool
    auto_orient: bool
    def __init__(self, strip_all: _Optional[bool] = ..., strip_gps: _Optional[bool] = ..., auto_orient: _Optional[bool] = ...) -> None: ...

class OpResult(_message.Message):
    __slots__ = ("ref", "format", "mime", "width", "height", "size_bytes")
    REF_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    MIME_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    ref: str
    format: str
    mime: str
    width: int
    height: int
    size_bytes: int
    def __init__(self, ref: _Optional[str] = ..., format: _Optional[str] = ..., mime: _Optional[str] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., size_bytes: _Optional[int] = ...) -> None: ...

class RunOpResponse(_message.Message):
    __slots__ = ("job", "result")
    JOB_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    job: _jobs_pb2.Job
    result: OpResult
    def __init__(self, job: _Optional[_Union[_jobs_pb2.Job, _Mapping]] = ..., result: _Optional[_Union[OpResult, _Mapping]] = ...) -> None: ...
