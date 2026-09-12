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
    __slots__ = ("resize", "crop", "rotate", "flip", "deskew", "thumbnail", "canvas", "adjust", "filter", "convert", "compress", "overlay", "metadata", "duotone", "posterize", "halftone", "dither_ordered", "dither_diffusion", "grain", "scrim", "line_screen", "stipple", "engraving", "aberration", "bloom", "curve", "defocus", "motion_blur", "ascii_mosaic", "pixel_sort", "displacement", "composite", "knockout")
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
    DUOTONE_FIELD_NUMBER: _ClassVar[int]
    POSTERIZE_FIELD_NUMBER: _ClassVar[int]
    HALFTONE_FIELD_NUMBER: _ClassVar[int]
    DITHER_ORDERED_FIELD_NUMBER: _ClassVar[int]
    DITHER_DIFFUSION_FIELD_NUMBER: _ClassVar[int]
    GRAIN_FIELD_NUMBER: _ClassVar[int]
    SCRIM_FIELD_NUMBER: _ClassVar[int]
    LINE_SCREEN_FIELD_NUMBER: _ClassVar[int]
    STIPPLE_FIELD_NUMBER: _ClassVar[int]
    ENGRAVING_FIELD_NUMBER: _ClassVar[int]
    ABERRATION_FIELD_NUMBER: _ClassVar[int]
    BLOOM_FIELD_NUMBER: _ClassVar[int]
    CURVE_FIELD_NUMBER: _ClassVar[int]
    DEFOCUS_FIELD_NUMBER: _ClassVar[int]
    MOTION_BLUR_FIELD_NUMBER: _ClassVar[int]
    ASCII_MOSAIC_FIELD_NUMBER: _ClassVar[int]
    PIXEL_SORT_FIELD_NUMBER: _ClassVar[int]
    DISPLACEMENT_FIELD_NUMBER: _ClassVar[int]
    COMPOSITE_FIELD_NUMBER: _ClassVar[int]
    KNOCKOUT_FIELD_NUMBER: _ClassVar[int]
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
    duotone: DuotoneParams
    posterize: PosterizeParams
    halftone: HalftoneParams
    dither_ordered: DitherParams
    dither_diffusion: DitherParams
    grain: GrainParams
    scrim: ScrimParams
    line_screen: LineScreenParams
    stipple: StippleParams
    engraving: EngravingParams
    aberration: AberrationParams
    bloom: BloomParams
    curve: CurveParams
    defocus: DefocusParams
    motion_blur: MotionBlurParams
    ascii_mosaic: AsciiMosaicParams
    pixel_sort: PixelSortParams
    displacement: DisplacementParams
    composite: CompositeParams
    knockout: Knockout
    def __init__(self, resize: _Optional[_Union[ResizeParams, _Mapping]] = ..., crop: _Optional[_Union[CropParams, _Mapping]] = ..., rotate: _Optional[_Union[RotateParams, _Mapping]] = ..., flip: _Optional[_Union[FlipParams, _Mapping]] = ..., deskew: _Optional[_Union[DeskewParams, _Mapping]] = ..., thumbnail: _Optional[_Union[ThumbnailParams, _Mapping]] = ..., canvas: _Optional[_Union[CanvasParams, _Mapping]] = ..., adjust: _Optional[_Union[AdjustParams, _Mapping]] = ..., filter: _Optional[_Union[FilterParams, _Mapping]] = ..., convert: _Optional[_Union[ConvertParams, _Mapping]] = ..., compress: _Optional[_Union[CompressParams, _Mapping]] = ..., overlay: _Optional[_Union[OverlayParams, _Mapping]] = ..., metadata: _Optional[_Union[MetadataParams, _Mapping]] = ..., duotone: _Optional[_Union[DuotoneParams, _Mapping]] = ..., posterize: _Optional[_Union[PosterizeParams, _Mapping]] = ..., halftone: _Optional[_Union[HalftoneParams, _Mapping]] = ..., dither_ordered: _Optional[_Union[DitherParams, _Mapping]] = ..., dither_diffusion: _Optional[_Union[DitherParams, _Mapping]] = ..., grain: _Optional[_Union[GrainParams, _Mapping]] = ..., scrim: _Optional[_Union[ScrimParams, _Mapping]] = ..., line_screen: _Optional[_Union[LineScreenParams, _Mapping]] = ..., stipple: _Optional[_Union[StippleParams, _Mapping]] = ..., engraving: _Optional[_Union[EngravingParams, _Mapping]] = ..., aberration: _Optional[_Union[AberrationParams, _Mapping]] = ..., bloom: _Optional[_Union[BloomParams, _Mapping]] = ..., curve: _Optional[_Union[CurveParams, _Mapping]] = ..., defocus: _Optional[_Union[DefocusParams, _Mapping]] = ..., motion_blur: _Optional[_Union[MotionBlurParams, _Mapping]] = ..., ascii_mosaic: _Optional[_Union[AsciiMosaicParams, _Mapping]] = ..., pixel_sort: _Optional[_Union[PixelSortParams, _Mapping]] = ..., displacement: _Optional[_Union[DisplacementParams, _Mapping]] = ..., composite: _Optional[_Union[CompositeParams, _Mapping]] = ..., knockout: _Optional[_Union[Knockout, _Mapping]] = ...) -> None: ...

class Knockout(_message.Message):
    __slots__ = ("x", "y", "width", "height", "feather", "solid")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    FEATHER_FIELD_NUMBER: _ClassVar[int]
    SOLID_FIELD_NUMBER: _ClassVar[int]
    x: float
    y: float
    width: float
    height: float
    feather: float
    solid: bool
    def __init__(self, x: _Optional[float] = ..., y: _Optional[float] = ..., width: _Optional[float] = ..., height: _Optional[float] = ..., feather: _Optional[float] = ..., solid: _Optional[bool] = ...) -> None: ...

class CompositeParams(_message.Message):
    __slots__ = ("plates", "width", "height", "background")
    PLATES_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_FIELD_NUMBER: _ClassVar[int]
    plates: _containers.RepeatedCompositeFieldContainer[Plate]
    width: int
    height: int
    background: str
    def __init__(self, plates: _Optional[_Iterable[_Union[Plate, _Mapping]]] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., background: _Optional[str] = ...) -> None: ...

class Plate(_message.Message):
    __slots__ = ("name", "depth", "blend", "opacity")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DEPTH_FIELD_NUMBER: _ClassVar[int]
    BLEND_FIELD_NUMBER: _ClassVar[int]
    OPACITY_FIELD_NUMBER: _ClassVar[int]
    name: str
    depth: int
    blend: str
    opacity: float
    def __init__(self, name: _Optional[str] = ..., depth: _Optional[int] = ..., blend: _Optional[str] = ..., opacity: _Optional[float] = ...) -> None: ...

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

class DuotoneParams(_message.Message):
    __slots__ = ("dark", "light", "mid", "mid_low", "mid_high", "normalize")
    DARK_FIELD_NUMBER: _ClassVar[int]
    LIGHT_FIELD_NUMBER: _ClassVar[int]
    MID_FIELD_NUMBER: _ClassVar[int]
    MID_LOW_FIELD_NUMBER: _ClassVar[int]
    MID_HIGH_FIELD_NUMBER: _ClassVar[int]
    NORMALIZE_FIELD_NUMBER: _ClassVar[int]
    dark: str
    light: str
    mid: str
    mid_low: float
    mid_high: float
    normalize: bool
    def __init__(self, dark: _Optional[str] = ..., light: _Optional[str] = ..., mid: _Optional[str] = ..., mid_low: _Optional[float] = ..., mid_high: _Optional[float] = ..., normalize: _Optional[bool] = ...) -> None: ...

class PosterizeParams(_message.Message):
    __slots__ = ("levels", "dark", "light", "normalize")
    LEVELS_FIELD_NUMBER: _ClassVar[int]
    DARK_FIELD_NUMBER: _ClassVar[int]
    LIGHT_FIELD_NUMBER: _ClassVar[int]
    NORMALIZE_FIELD_NUMBER: _ClassVar[int]
    levels: int
    dark: str
    light: str
    normalize: bool
    def __init__(self, levels: _Optional[int] = ..., dark: _Optional[str] = ..., light: _Optional[str] = ..., normalize: _Optional[bool] = ...) -> None: ...

class HalftoneParams(_message.Message):
    __slots__ = ("lpi", "angle", "dot", "dark", "light", "normalize")
    LPI_FIELD_NUMBER: _ClassVar[int]
    ANGLE_FIELD_NUMBER: _ClassVar[int]
    DOT_FIELD_NUMBER: _ClassVar[int]
    DARK_FIELD_NUMBER: _ClassVar[int]
    LIGHT_FIELD_NUMBER: _ClassVar[int]
    NORMALIZE_FIELD_NUMBER: _ClassVar[int]
    lpi: int
    angle: float
    dot: str
    dark: str
    light: str
    normalize: bool
    def __init__(self, lpi: _Optional[int] = ..., angle: _Optional[float] = ..., dot: _Optional[str] = ..., dark: _Optional[str] = ..., light: _Optional[str] = ..., normalize: _Optional[bool] = ...) -> None: ...

class DitherParams(_message.Message):
    __slots__ = ("dark", "light", "normalize")
    DARK_FIELD_NUMBER: _ClassVar[int]
    LIGHT_FIELD_NUMBER: _ClassVar[int]
    NORMALIZE_FIELD_NUMBER: _ClassVar[int]
    dark: str
    light: str
    normalize: bool
    def __init__(self, dark: _Optional[str] = ..., light: _Optional[str] = ..., normalize: _Optional[bool] = ...) -> None: ...

class GrainParams(_message.Message):
    __slots__ = ("seed", "amount", "contrast_multiplier")
    SEED_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_FIELD_NUMBER: _ClassVar[int]
    CONTRAST_MULTIPLIER_FIELD_NUMBER: _ClassVar[int]
    seed: int
    amount: float
    contrast_multiplier: float
    def __init__(self, seed: _Optional[int] = ..., amount: _Optional[float] = ..., contrast_multiplier: _Optional[float] = ...) -> None: ...

class ScrimParams(_message.Message):
    __slots__ = ("color", "opacity", "direction", "region_x", "region_y", "region_width", "region_height", "region_feather")
    COLOR_FIELD_NUMBER: _ClassVar[int]
    OPACITY_FIELD_NUMBER: _ClassVar[int]
    DIRECTION_FIELD_NUMBER: _ClassVar[int]
    REGION_X_FIELD_NUMBER: _ClassVar[int]
    REGION_Y_FIELD_NUMBER: _ClassVar[int]
    REGION_WIDTH_FIELD_NUMBER: _ClassVar[int]
    REGION_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    REGION_FEATHER_FIELD_NUMBER: _ClassVar[int]
    color: str
    opacity: float
    direction: str
    region_x: float
    region_y: float
    region_width: float
    region_height: float
    region_feather: float
    def __init__(self, color: _Optional[str] = ..., opacity: _Optional[float] = ..., direction: _Optional[str] = ..., region_x: _Optional[float] = ..., region_y: _Optional[float] = ..., region_width: _Optional[float] = ..., region_height: _Optional[float] = ..., region_feather: _Optional[float] = ...) -> None: ...

class LineScreenParams(_message.Message):
    __slots__ = ("spacing", "angle", "dark", "light", "normalize", "spacing_rel")
    SPACING_FIELD_NUMBER: _ClassVar[int]
    ANGLE_FIELD_NUMBER: _ClassVar[int]
    DARK_FIELD_NUMBER: _ClassVar[int]
    LIGHT_FIELD_NUMBER: _ClassVar[int]
    NORMALIZE_FIELD_NUMBER: _ClassVar[int]
    SPACING_REL_FIELD_NUMBER: _ClassVar[int]
    spacing: float
    angle: float
    dark: str
    light: str
    normalize: bool
    spacing_rel: float
    def __init__(self, spacing: _Optional[float] = ..., angle: _Optional[float] = ..., dark: _Optional[str] = ..., light: _Optional[str] = ..., normalize: _Optional[bool] = ..., spacing_rel: _Optional[float] = ...) -> None: ...

class StippleParams(_message.Message):
    __slots__ = ("spacing", "seed", "dark", "light", "normalize", "spacing_rel")
    SPACING_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    DARK_FIELD_NUMBER: _ClassVar[int]
    LIGHT_FIELD_NUMBER: _ClassVar[int]
    NORMALIZE_FIELD_NUMBER: _ClassVar[int]
    SPACING_REL_FIELD_NUMBER: _ClassVar[int]
    spacing: float
    seed: int
    dark: str
    light: str
    normalize: bool
    spacing_rel: float
    def __init__(self, spacing: _Optional[float] = ..., seed: _Optional[int] = ..., dark: _Optional[str] = ..., light: _Optional[str] = ..., normalize: _Optional[bool] = ..., spacing_rel: _Optional[float] = ...) -> None: ...

class EngravingParams(_message.Message):
    __slots__ = ("spacing", "dark", "light", "normalize", "spacing_rel")
    SPACING_FIELD_NUMBER: _ClassVar[int]
    DARK_FIELD_NUMBER: _ClassVar[int]
    LIGHT_FIELD_NUMBER: _ClassVar[int]
    NORMALIZE_FIELD_NUMBER: _ClassVar[int]
    SPACING_REL_FIELD_NUMBER: _ClassVar[int]
    spacing: float
    dark: str
    light: str
    normalize: bool
    spacing_rel: float
    def __init__(self, spacing: _Optional[float] = ..., dark: _Optional[str] = ..., light: _Optional[str] = ..., normalize: _Optional[bool] = ..., spacing_rel: _Optional[float] = ...) -> None: ...

class AberrationParams(_message.Message):
    __slots__ = ("amplitude", "distance", "distance_rel")
    AMPLITUDE_FIELD_NUMBER: _ClassVar[int]
    DISTANCE_FIELD_NUMBER: _ClassVar[int]
    DISTANCE_REL_FIELD_NUMBER: _ClassVar[int]
    amplitude: float
    distance: int
    distance_rel: float
    def __init__(self, amplitude: _Optional[float] = ..., distance: _Optional[int] = ..., distance_rel: _Optional[float] = ...) -> None: ...

class BloomParams(_message.Message):
    __slots__ = ("radius", "threshold", "radius_rel")
    RADIUS_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    RADIUS_REL_FIELD_NUMBER: _ClassVar[int]
    radius: int
    threshold: float
    radius_rel: float
    def __init__(self, radius: _Optional[int] = ..., threshold: _Optional[float] = ..., radius_rel: _Optional[float] = ...) -> None: ...

class CurveParams(_message.Message):
    __slots__ = ("exponent",)
    EXPONENT_FIELD_NUMBER: _ClassVar[int]
    exponent: float
    def __init__(self, exponent: _Optional[float] = ...) -> None: ...

class DefocusParams(_message.Message):
    __slots__ = ("radius", "blade_count", "radius_rel")
    RADIUS_FIELD_NUMBER: _ClassVar[int]
    BLADE_COUNT_FIELD_NUMBER: _ClassVar[int]
    RADIUS_REL_FIELD_NUMBER: _ClassVar[int]
    radius: int
    blade_count: int
    radius_rel: float
    def __init__(self, radius: _Optional[int] = ..., blade_count: _Optional[int] = ..., radius_rel: _Optional[float] = ...) -> None: ...

class MotionBlurParams(_message.Message):
    __slots__ = ("distance", "angle", "distance_rel")
    DISTANCE_FIELD_NUMBER: _ClassVar[int]
    ANGLE_FIELD_NUMBER: _ClassVar[int]
    DISTANCE_REL_FIELD_NUMBER: _ClassVar[int]
    distance: int
    angle: float
    distance_rel: float
    def __init__(self, distance: _Optional[int] = ..., angle: _Optional[float] = ..., distance_rel: _Optional[float] = ...) -> None: ...

class AsciiMosaicParams(_message.Message):
    __slots__ = ("block_size", "dark", "light", "normalize", "block_size_rel")
    BLOCK_SIZE_FIELD_NUMBER: _ClassVar[int]
    DARK_FIELD_NUMBER: _ClassVar[int]
    LIGHT_FIELD_NUMBER: _ClassVar[int]
    NORMALIZE_FIELD_NUMBER: _ClassVar[int]
    BLOCK_SIZE_REL_FIELD_NUMBER: _ClassVar[int]
    block_size: int
    dark: str
    light: str
    normalize: bool
    block_size_rel: float
    def __init__(self, block_size: _Optional[int] = ..., dark: _Optional[str] = ..., light: _Optional[str] = ..., normalize: _Optional[bool] = ..., block_size_rel: _Optional[float] = ...) -> None: ...

class PixelSortParams(_message.Message):
    __slots__ = ("threshold", "axis")
    THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    AXIS_FIELD_NUMBER: _ClassVar[int]
    threshold: float
    axis: str
    def __init__(self, threshold: _Optional[float] = ..., axis: _Optional[str] = ...) -> None: ...

class DisplacementParams(_message.Message):
    __slots__ = ("amplitude", "seed", "spacing", "spacing_rel", "amplitude_rel")
    AMPLITUDE_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    SPACING_FIELD_NUMBER: _ClassVar[int]
    SPACING_REL_FIELD_NUMBER: _ClassVar[int]
    AMPLITUDE_REL_FIELD_NUMBER: _ClassVar[int]
    amplitude: float
    seed: int
    spacing: float
    spacing_rel: float
    amplitude_rel: float
    def __init__(self, amplitude: _Optional[float] = ..., seed: _Optional[int] = ..., spacing: _Optional[float] = ..., spacing_rel: _Optional[float] = ..., amplitude_rel: _Optional[float] = ...) -> None: ...

class OpResult(_message.Message):
    __slots__ = ("ref", "format", "mime", "width", "height", "size_bytes", "resolved_params")
    class ResolvedParamsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    REF_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    MIME_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_PARAMS_FIELD_NUMBER: _ClassVar[int]
    ref: str
    format: str
    mime: str
    width: int
    height: int
    size_bytes: int
    resolved_params: _containers.ScalarMap[str, float]
    def __init__(self, ref: _Optional[str] = ..., format: _Optional[str] = ..., mime: _Optional[str] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., size_bytes: _Optional[int] = ..., resolved_params: _Optional[_Mapping[str, float]] = ...) -> None: ...

class RunOpResponse(_message.Message):
    __slots__ = ("job_id", "result")
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    result: OpResult
    def __init__(self, job_id: _Optional[str] = ..., result: _Optional[_Union[OpResult, _Mapping]] = ...) -> None: ...
