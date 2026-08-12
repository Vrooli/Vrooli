from backdrop_studio.v1.shared import shared_pb2 as _shared_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ReleaseRequest(_message.Message):
    __slots__ = ("candidate_id", "style_id", "strategy", "surface_id", "width", "height", "expected_width", "expected_height", "placement", "alt_text", "decorative", "ai_generated", "ai_generated_set", "contrast_ratio", "contrast_threshold", "legibility_passes", "reserved_regions", "image_png")
    CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    STYLE_ID_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    SURFACE_ID_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_WIDTH_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    PLACEMENT_FIELD_NUMBER: _ClassVar[int]
    ALT_TEXT_FIELD_NUMBER: _ClassVar[int]
    DECORATIVE_FIELD_NUMBER: _ClassVar[int]
    AI_GENERATED_FIELD_NUMBER: _ClassVar[int]
    AI_GENERATED_SET_FIELD_NUMBER: _ClassVar[int]
    CONTRAST_RATIO_FIELD_NUMBER: _ClassVar[int]
    CONTRAST_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    LEGIBILITY_PASSES_FIELD_NUMBER: _ClassVar[int]
    RESERVED_REGIONS_FIELD_NUMBER: _ClassVar[int]
    IMAGE_PNG_FIELD_NUMBER: _ClassVar[int]
    candidate_id: str
    style_id: str
    strategy: str
    surface_id: str
    width: int
    height: int
    expected_width: int
    expected_height: int
    placement: str
    alt_text: str
    decorative: bool
    ai_generated: bool
    ai_generated_set: bool
    contrast_ratio: float
    contrast_threshold: float
    legibility_passes: bool
    reserved_regions: _containers.RepeatedCompositeFieldContainer[_shared_pb2.ReservedRegion]
    image_png: bytes
    def __init__(self, candidate_id: _Optional[str] = ..., style_id: _Optional[str] = ..., strategy: _Optional[str] = ..., surface_id: _Optional[str] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., expected_width: _Optional[int] = ..., expected_height: _Optional[int] = ..., placement: _Optional[str] = ..., alt_text: _Optional[str] = ..., decorative: _Optional[bool] = ..., ai_generated: _Optional[bool] = ..., ai_generated_set: _Optional[bool] = ..., contrast_ratio: _Optional[float] = ..., contrast_threshold: _Optional[float] = ..., legibility_passes: _Optional[bool] = ..., reserved_regions: _Optional[_Iterable[_Union[_shared_pb2.ReservedRegion, _Mapping]]] = ..., image_png: _Optional[bytes] = ...) -> None: ...

class GetReferenceRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ReleasedBackdrop(_message.Message):
    __slots__ = ("id", "candidate_id", "style_id", "surface_id", "placement", "width", "height", "alt_text", "decorative", "ai_generated", "contrast_ratio", "contrast_threshold", "reserved_regions", "uri", "asset_studio_ref")
    ID_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    STYLE_ID_FIELD_NUMBER: _ClassVar[int]
    SURFACE_ID_FIELD_NUMBER: _ClassVar[int]
    PLACEMENT_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    ALT_TEXT_FIELD_NUMBER: _ClassVar[int]
    DECORATIVE_FIELD_NUMBER: _ClassVar[int]
    AI_GENERATED_FIELD_NUMBER: _ClassVar[int]
    CONTRAST_RATIO_FIELD_NUMBER: _ClassVar[int]
    CONTRAST_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    RESERVED_REGIONS_FIELD_NUMBER: _ClassVar[int]
    URI_FIELD_NUMBER: _ClassVar[int]
    ASSET_STUDIO_REF_FIELD_NUMBER: _ClassVar[int]
    id: str
    candidate_id: str
    style_id: str
    surface_id: str
    placement: str
    width: int
    height: int
    alt_text: str
    decorative: bool
    ai_generated: bool
    contrast_ratio: float
    contrast_threshold: float
    reserved_regions: _containers.RepeatedCompositeFieldContainer[_shared_pb2.ReservedRegion]
    uri: str
    asset_studio_ref: str
    def __init__(self, id: _Optional[str] = ..., candidate_id: _Optional[str] = ..., style_id: _Optional[str] = ..., surface_id: _Optional[str] = ..., placement: _Optional[str] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., alt_text: _Optional[str] = ..., decorative: _Optional[bool] = ..., ai_generated: _Optional[bool] = ..., contrast_ratio: _Optional[float] = ..., contrast_threshold: _Optional[float] = ..., reserved_regions: _Optional[_Iterable[_Union[_shared_pb2.ReservedRegion, _Mapping]]] = ..., uri: _Optional[str] = ..., asset_studio_ref: _Optional[str] = ...) -> None: ...
