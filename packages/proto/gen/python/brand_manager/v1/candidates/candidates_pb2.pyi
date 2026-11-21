import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CandidateOrigin(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CANDIDATE_ORIGIN_UNSPECIFIED: _ClassVar[CandidateOrigin]
    CANDIDATE_ORIGIN_GENERATED: _ClassVar[CandidateOrigin]
    CANDIDATE_ORIGIN_IMPORTED: _ClassVar[CandidateOrigin]
    CANDIDATE_ORIGIN_EDITED: _ClassVar[CandidateOrigin]
    CANDIDATE_ORIGIN_OBJECT_REMOVED: _ClassVar[CandidateOrigin]
    CANDIDATE_ORIGIN_BACKGROUND_REMOVED: _ClassVar[CandidateOrigin]
    CANDIDATE_ORIGIN_VECTORIZED: _ClassVar[CandidateOrigin]

class CandidateStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CANDIDATE_STATUS_UNSPECIFIED: _ClassVar[CandidateStatus]
    CANDIDATE_STATUS_PROPOSED: _ClassVar[CandidateStatus]
    CANDIDATE_STATUS_PICKED: _ClassVar[CandidateStatus]
    CANDIDATE_STATUS_REJECTED: _ClassVar[CandidateStatus]
    CANDIDATE_STATUS_SUPERSEDED: _ClassVar[CandidateStatus]
CANDIDATE_ORIGIN_UNSPECIFIED: CandidateOrigin
CANDIDATE_ORIGIN_GENERATED: CandidateOrigin
CANDIDATE_ORIGIN_IMPORTED: CandidateOrigin
CANDIDATE_ORIGIN_EDITED: CandidateOrigin
CANDIDATE_ORIGIN_OBJECT_REMOVED: CandidateOrigin
CANDIDATE_ORIGIN_BACKGROUND_REMOVED: CandidateOrigin
CANDIDATE_ORIGIN_VECTORIZED: CandidateOrigin
CANDIDATE_STATUS_UNSPECIFIED: CandidateStatus
CANDIDATE_STATUS_PROPOSED: CandidateStatus
CANDIDATE_STATUS_PICKED: CandidateStatus
CANDIDATE_STATUS_REJECTED: CandidateStatus
CANDIDATE_STATUS_SUPERSEDED: CandidateStatus

class VectorizeOptions(_message.Message):
    __slots__ = ("colors", "keep_colors", "drop_background_layers", "clip_to_largest_rounded_region", "inset_px", "tolerance_px", "smoothing", "min_area_px")
    COLORS_FIELD_NUMBER: _ClassVar[int]
    KEEP_COLORS_FIELD_NUMBER: _ClassVar[int]
    DROP_BACKGROUND_LAYERS_FIELD_NUMBER: _ClassVar[int]
    CLIP_TO_LARGEST_ROUNDED_REGION_FIELD_NUMBER: _ClassVar[int]
    INSET_PX_FIELD_NUMBER: _ClassVar[int]
    TOLERANCE_PX_FIELD_NUMBER: _ClassVar[int]
    SMOOTHING_FIELD_NUMBER: _ClassVar[int]
    MIN_AREA_PX_FIELD_NUMBER: _ClassVar[int]
    colors: int
    keep_colors: _containers.RepeatedScalarFieldContainer[str]
    drop_background_layers: bool
    clip_to_largest_rounded_region: bool
    inset_px: float
    tolerance_px: float
    smoothing: bool
    min_area_px: float
    def __init__(self, colors: _Optional[int] = ..., keep_colors: _Optional[_Iterable[str]] = ..., drop_background_layers: _Optional[bool] = ..., clip_to_largest_rounded_region: _Optional[bool] = ..., inset_px: _Optional[float] = ..., tolerance_px: _Optional[float] = ..., smoothing: _Optional[bool] = ..., min_area_px: _Optional[float] = ...) -> None: ...

class LogoCandidate(_message.Message):
    __slots__ = ("id", "brand_id", "asset_id", "media_type", "concept", "prompt", "role", "model", "seed", "origin", "parent_id", "status", "note", "created_at", "thumbnail_url")
    ID_FIELD_NUMBER: _ClassVar[int]
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    CONCEPT_FIELD_NUMBER: _ClassVar[int]
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    ORIGIN_FIELD_NUMBER: _ClassVar[int]
    PARENT_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    THUMBNAIL_URL_FIELD_NUMBER: _ClassVar[int]
    id: str
    brand_id: str
    asset_id: str
    media_type: str
    concept: str
    prompt: str
    role: str
    model: str
    seed: int
    origin: CandidateOrigin
    parent_id: str
    status: CandidateStatus
    note: str
    created_at: _timestamp_pb2.Timestamp
    thumbnail_url: str
    def __init__(self, id: _Optional[str] = ..., brand_id: _Optional[str] = ..., asset_id: _Optional[str] = ..., media_type: _Optional[str] = ..., concept: _Optional[str] = ..., prompt: _Optional[str] = ..., role: _Optional[str] = ..., model: _Optional[str] = ..., seed: _Optional[int] = ..., origin: _Optional[_Union[CandidateOrigin, str]] = ..., parent_id: _Optional[str] = ..., status: _Optional[_Union[CandidateStatus, str]] = ..., note: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., thumbnail_url: _Optional[str] = ...) -> None: ...

class ExploreCandidatesRequest(_message.Message):
    __slots__ = ("brand_id", "brief", "concepts", "variations", "prefer_vector", "quality_policy", "fallback_policy", "allow_byok", "role", "style_reference_brand", "style_reference_asset_id")
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    BRIEF_FIELD_NUMBER: _ClassVar[int]
    CONCEPTS_FIELD_NUMBER: _ClassVar[int]
    VARIATIONS_FIELD_NUMBER: _ClassVar[int]
    PREFER_VECTOR_FIELD_NUMBER: _ClassVar[int]
    QUALITY_POLICY_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_POLICY_FIELD_NUMBER: _ClassVar[int]
    ALLOW_BYOK_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    STYLE_REFERENCE_BRAND_FIELD_NUMBER: _ClassVar[int]
    STYLE_REFERENCE_ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    brand_id: str
    brief: str
    concepts: _containers.RepeatedScalarFieldContainer[str]
    variations: int
    prefer_vector: bool
    quality_policy: str
    fallback_policy: str
    allow_byok: bool
    role: str
    style_reference_brand: str
    style_reference_asset_id: str
    def __init__(self, brand_id: _Optional[str] = ..., brief: _Optional[str] = ..., concepts: _Optional[_Iterable[str]] = ..., variations: _Optional[int] = ..., prefer_vector: _Optional[bool] = ..., quality_policy: _Optional[str] = ..., fallback_policy: _Optional[str] = ..., allow_byok: _Optional[bool] = ..., role: _Optional[str] = ..., style_reference_brand: _Optional[str] = ..., style_reference_asset_id: _Optional[str] = ...) -> None: ...

class ExploreCandidatesResponse(_message.Message):
    __slots__ = ("candidates", "warnings")
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    candidates: _containers.RepeatedCompositeFieldContainer[LogoCandidate]
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, candidates: _Optional[_Iterable[_Union[LogoCandidate, _Mapping]]] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...

class ImportCandidateRequest(_message.Message):
    __slots__ = ("brand_id", "asset_id", "concept", "parent_id", "note")
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    CONCEPT_FIELD_NUMBER: _ClassVar[int]
    PARENT_ID_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    brand_id: str
    asset_id: str
    concept: str
    parent_id: str
    note: str
    def __init__(self, brand_id: _Optional[str] = ..., asset_id: _Optional[str] = ..., concept: _Optional[str] = ..., parent_id: _Optional[str] = ..., note: _Optional[str] = ...) -> None: ...

class ImportCandidateResponse(_message.Message):
    __slots__ = ("candidate",)
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    candidate: LogoCandidate
    def __init__(self, candidate: _Optional[_Union[LogoCandidate, _Mapping]] = ...) -> None: ...

class ListCandidatesRequest(_message.Message):
    __slots__ = ("brand_id", "status", "limit", "offset")
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    brand_id: str
    status: CandidateStatus
    limit: int
    offset: int
    def __init__(self, brand_id: _Optional[str] = ..., status: _Optional[_Union[CandidateStatus, str]] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListCandidatesResponse(_message.Message):
    __slots__ = ("candidates",)
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    candidates: _containers.RepeatedCompositeFieldContainer[LogoCandidate]
    def __init__(self, candidates: _Optional[_Iterable[_Union[LogoCandidate, _Mapping]]] = ...) -> None: ...

class GetCandidateRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetCandidateResponse(_message.Message):
    __slots__ = ("candidate",)
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    candidate: LogoCandidate
    def __init__(self, candidate: _Optional[_Union[LogoCandidate, _Mapping]] = ...) -> None: ...

class RefineCandidateRequest(_message.Message):
    __slots__ = ("candidate_id", "instruction", "mask_asset_id", "remove_background", "vectorize")
    CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    INSTRUCTION_FIELD_NUMBER: _ClassVar[int]
    MASK_ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    REMOVE_BACKGROUND_FIELD_NUMBER: _ClassVar[int]
    VECTORIZE_FIELD_NUMBER: _ClassVar[int]
    candidate_id: str
    instruction: str
    mask_asset_id: str
    remove_background: bool
    vectorize: VectorizeOptions
    def __init__(self, candidate_id: _Optional[str] = ..., instruction: _Optional[str] = ..., mask_asset_id: _Optional[str] = ..., remove_background: _Optional[bool] = ..., vectorize: _Optional[_Union[VectorizeOptions, _Mapping]] = ...) -> None: ...

class RefineCandidateResponse(_message.Message):
    __slots__ = ("candidate",)
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    candidate: LogoCandidate
    def __init__(self, candidate: _Optional[_Union[LogoCandidate, _Mapping]] = ...) -> None: ...

class PickCandidateRequest(_message.Message):
    __slots__ = ("candidate_id",)
    CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    candidate_id: str
    def __init__(self, candidate_id: _Optional[str] = ...) -> None: ...

class PickCandidateResponse(_message.Message):
    __slots__ = ("candidate", "mark_asset_id", "vectorized")
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    MARK_ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    VECTORIZED_FIELD_NUMBER: _ClassVar[int]
    candidate: LogoCandidate
    mark_asset_id: str
    vectorized: bool
    def __init__(self, candidate: _Optional[_Union[LogoCandidate, _Mapping]] = ..., mark_asset_id: _Optional[str] = ..., vectorized: _Optional[bool] = ...) -> None: ...

class RejectCandidateRequest(_message.Message):
    __slots__ = ("candidate_id", "note")
    CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    candidate_id: str
    note: str
    def __init__(self, candidate_id: _Optional[str] = ..., note: _Optional[str] = ...) -> None: ...

class RejectCandidateResponse(_message.Message):
    __slots__ = ("candidate",)
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    candidate: LogoCandidate
    def __init__(self, candidate: _Optional[_Union[LogoCandidate, _Mapping]] = ...) -> None: ...

class RestoreCandidateRequest(_message.Message):
    __slots__ = ("candidate_id",)
    CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    candidate_id: str
    def __init__(self, candidate_id: _Optional[str] = ...) -> None: ...

class RestoreCandidateResponse(_message.Message):
    __slots__ = ("candidate",)
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    candidate: LogoCandidate
    def __init__(self, candidate: _Optional[_Union[LogoCandidate, _Mapping]] = ...) -> None: ...
