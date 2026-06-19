from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AnalysisOperationInfo(_message.Message):
    __slots__ = ("name", "summary", "model_backed", "default_model_id")
    NAME_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    MODEL_BACKED_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    name: str
    summary: str
    model_backed: bool
    default_model_id: str
    def __init__(self, name: _Optional[str] = ..., summary: _Optional[str] = ..., model_backed: _Optional[bool] = ..., default_model_id: _Optional[str] = ...) -> None: ...

class ListAnalysisOperationsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListAnalysisOperationsResponse(_message.Message):
    __slots__ = ("operations",)
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    operations: _containers.RepeatedCompositeFieldContainer[AnalysisOperationInfo]
    def __init__(self, operations: _Optional[_Iterable[_Union[AnalysisOperationInfo, _Mapping]]] = ...) -> None: ...

class BoundingBox(_message.Message):
    __slots__ = ("x", "y", "width", "height")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    x: int
    y: int
    width: int
    height: int
    def __init__(self, x: _Optional[int] = ..., y: _Optional[int] = ..., width: _Optional[int] = ..., height: _Optional[int] = ...) -> None: ...

class OCRBlock(_message.Message):
    __slots__ = ("text", "confidence", "box")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    BOX_FIELD_NUMBER: _ClassVar[int]
    text: str
    confidence: float
    box: BoundingBox
    def __init__(self, text: _Optional[str] = ..., confidence: _Optional[float] = ..., box: _Optional[_Union[BoundingBox, _Mapping]] = ...) -> None: ...

class OCRResult(_message.Message):
    __slots__ = ("full_text", "blocks", "language")
    FULL_TEXT_FIELD_NUMBER: _ClassVar[int]
    BLOCKS_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    full_text: str
    blocks: _containers.RepeatedCompositeFieldContainer[OCRBlock]
    language: str
    def __init__(self, full_text: _Optional[str] = ..., blocks: _Optional[_Iterable[_Union[OCRBlock, _Mapping]]] = ..., language: _Optional[str] = ...) -> None: ...

class NSFWCategory(_message.Message):
    __slots__ = ("label", "score")
    LABEL_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    label: str
    score: float
    def __init__(self, label: _Optional[str] = ..., score: _Optional[float] = ...) -> None: ...

class NSFWResult(_message.Message):
    __slots__ = ("nsfw", "score", "label", "threshold", "categories")
    NSFW_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    CATEGORIES_FIELD_NUMBER: _ClassVar[int]
    nsfw: bool
    score: float
    label: str
    threshold: float
    categories: _containers.RepeatedCompositeFieldContainer[NSFWCategory]
    def __init__(self, nsfw: _Optional[bool] = ..., score: _Optional[float] = ..., label: _Optional[str] = ..., threshold: _Optional[float] = ..., categories: _Optional[_Iterable[_Union[NSFWCategory, _Mapping]]] = ...) -> None: ...

class DominantColor(_message.Message):
    __slots__ = ("hex", "fraction")
    HEX_FIELD_NUMBER: _ClassVar[int]
    FRACTION_FIELD_NUMBER: _ClassVar[int]
    hex: str
    fraction: float
    def __init__(self, hex: _Optional[str] = ..., fraction: _Optional[float] = ...) -> None: ...

class ProbeResult(_message.Message):
    __slots__ = ("width", "height", "format", "color_model", "has_alpha", "frame_count", "megapixels", "size_bytes", "has_exif", "has_gps", "orientation", "dominant_colors")
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    COLOR_MODEL_FIELD_NUMBER: _ClassVar[int]
    HAS_ALPHA_FIELD_NUMBER: _ClassVar[int]
    FRAME_COUNT_FIELD_NUMBER: _ClassVar[int]
    MEGAPIXELS_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    HAS_EXIF_FIELD_NUMBER: _ClassVar[int]
    HAS_GPS_FIELD_NUMBER: _ClassVar[int]
    ORIENTATION_FIELD_NUMBER: _ClassVar[int]
    DOMINANT_COLORS_FIELD_NUMBER: _ClassVar[int]
    width: int
    height: int
    format: str
    color_model: str
    has_alpha: bool
    frame_count: int
    megapixels: float
    size_bytes: int
    has_exif: bool
    has_gps: bool
    orientation: int
    dominant_colors: _containers.RepeatedCompositeFieldContainer[DominantColor]
    def __init__(self, width: _Optional[int] = ..., height: _Optional[int] = ..., format: _Optional[str] = ..., color_model: _Optional[str] = ..., has_alpha: _Optional[bool] = ..., frame_count: _Optional[int] = ..., megapixels: _Optional[float] = ..., size_bytes: _Optional[int] = ..., has_exif: _Optional[bool] = ..., has_gps: _Optional[bool] = ..., orientation: _Optional[int] = ..., dominant_colors: _Optional[_Iterable[_Union[DominantColor, _Mapping]]] = ...) -> None: ...

class DuplicateResult(_message.Message):
    __slots__ = ("phash_hex", "ahash_hex", "hash_bits")
    PHASH_HEX_FIELD_NUMBER: _ClassVar[int]
    AHASH_HEX_FIELD_NUMBER: _ClassVar[int]
    HASH_BITS_FIELD_NUMBER: _ClassVar[int]
    phash_hex: str
    ahash_hex: str
    hash_bits: int
    def __init__(self, phash_hex: _Optional[str] = ..., ahash_hex: _Optional[str] = ..., hash_bits: _Optional[int] = ...) -> None: ...

class QualityResult(_message.Message):
    __slots__ = ("overall_score", "sharpness", "blurry", "brightness", "contrast", "exposure", "notes")
    OVERALL_SCORE_FIELD_NUMBER: _ClassVar[int]
    SHARPNESS_FIELD_NUMBER: _ClassVar[int]
    BLURRY_FIELD_NUMBER: _ClassVar[int]
    BRIGHTNESS_FIELD_NUMBER: _ClassVar[int]
    CONTRAST_FIELD_NUMBER: _ClassVar[int]
    EXPOSURE_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    overall_score: float
    sharpness: float
    blurry: bool
    brightness: float
    contrast: float
    exposure: str
    notes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, overall_score: _Optional[float] = ..., sharpness: _Optional[float] = ..., blurry: _Optional[bool] = ..., brightness: _Optional[float] = ..., contrast: _Optional[float] = ..., exposure: _Optional[str] = ..., notes: _Optional[_Iterable[str]] = ...) -> None: ...

class AnalyzeResponse(_message.Message):
    __slots__ = ("job_id", "ocr", "nsfw", "probe", "duplicate", "quality")
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    OCR_FIELD_NUMBER: _ClassVar[int]
    NSFW_FIELD_NUMBER: _ClassVar[int]
    PROBE_FIELD_NUMBER: _ClassVar[int]
    DUPLICATE_FIELD_NUMBER: _ClassVar[int]
    QUALITY_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    ocr: OCRResult
    nsfw: NSFWResult
    probe: ProbeResult
    duplicate: DuplicateResult
    quality: QualityResult
    def __init__(self, job_id: _Optional[str] = ..., ocr: _Optional[_Union[OCRResult, _Mapping]] = ..., nsfw: _Optional[_Union[NSFWResult, _Mapping]] = ..., probe: _Optional[_Union[ProbeResult, _Mapping]] = ..., duplicate: _Optional[_Union[DuplicateResult, _Mapping]] = ..., quality: _Optional[_Union[QualityResult, _Mapping]] = ...) -> None: ...
