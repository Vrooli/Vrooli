from image_tools.v1.ai import ai_pb2 as _ai_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PhotographRoute(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PHOTOGRAPH_ROUTE_UNSPECIFIED: _ClassVar[PhotographRoute]
    PHOTOGRAPH_ROUTE_EMPTY_SCENE: _ClassVar[PhotographRoute]
    PHOTOGRAPH_ROUTE_DEVICE_REMOVAL: _ClassVar[PhotographRoute]
PHOTOGRAPH_ROUTE_UNSPECIFIED: PhotographRoute
PHOTOGRAPH_ROUTE_EMPTY_SCENE: PhotographRoute
PHOTOGRAPH_ROUTE_DEVICE_REMOVAL: PhotographRoute

class PreparedPhotograph(_message.Message):
    __slots__ = ("contract_version", "route", "source", "geometry", "background", "scale_status", "camera_envelope", "review")
    CONTRACT_VERSION_FIELD_NUMBER: _ClassVar[int]
    ROUTE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_FIELD_NUMBER: _ClassVar[int]
    SCALE_STATUS_FIELD_NUMBER: _ClassVar[int]
    CAMERA_ENVELOPE_FIELD_NUMBER: _ClassVar[int]
    REVIEW_FIELD_NUMBER: _ClassVar[int]
    contract_version: str
    route: PhotographRoute
    source: SourceRaster
    geometry: Geometry
    background: Background
    scale_status: str
    camera_envelope: CameraEnvelope
    review: VisualReview
    def __init__(self, contract_version: _Optional[str] = ..., route: _Optional[_Union[PhotographRoute, str]] = ..., source: _Optional[_Union[SourceRaster, _Mapping]] = ..., geometry: _Optional[_Union[Geometry, _Mapping]] = ..., background: _Optional[_Union[Background, _Mapping]] = ..., scale_status: _Optional[str] = ..., camera_envelope: _Optional[_Union[CameraEnvelope, _Mapping]] = ..., review: _Optional[_Union[VisualReview, _Mapping]] = ...) -> None: ...

class Artifact(_message.Message):
    __slots__ = ("path", "sha256", "size_bytes", "media_type")
    PATH_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    path: str
    sha256: str
    size_bytes: int
    media_type: str
    def __init__(self, path: _Optional[str] = ..., sha256: _Optional[str] = ..., size_bytes: _Optional[int] = ..., media_type: _Optional[str] = ...) -> None: ...

class SourceRaster(_message.Message):
    __slots__ = ("original", "oriented", "original_width", "original_height", "width", "height", "original_exif_orientation", "original_to_oriented")
    ORIGINAL_FIELD_NUMBER: _ClassVar[int]
    ORIENTED_FIELD_NUMBER: _ClassVar[int]
    ORIGINAL_WIDTH_FIELD_NUMBER: _ClassVar[int]
    ORIGINAL_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    ORIGINAL_EXIF_ORIENTATION_FIELD_NUMBER: _ClassVar[int]
    ORIGINAL_TO_ORIENTED_FIELD_NUMBER: _ClassVar[int]
    original: Artifact
    oriented: Artifact
    original_width: int
    original_height: int
    width: int
    height: int
    original_exif_orientation: int
    original_to_oriented: _containers.RepeatedScalarFieldContainer[float]
    def __init__(self, original: _Optional[_Union[Artifact, _Mapping]] = ..., oriented: _Optional[_Union[Artifact, _Mapping]] = ..., original_width: _Optional[int] = ..., original_height: _Optional[int] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., original_exif_orientation: _Optional[int] = ..., original_to_oriented: _Optional[_Iterable[float]] = ...) -> None: ...

class Geometry(_message.Message):
    __slots__ = ("archive", "camera", "support_plane")
    ARCHIVE_FIELD_NUMBER: _ClassVar[int]
    CAMERA_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_PLANE_FIELD_NUMBER: _ClassVar[int]
    archive: Artifact
    camera: Camera
    support_plane: SupportPlane
    def __init__(self, archive: _Optional[_Union[Artifact, _Mapping]] = ..., camera: _Optional[_Union[Camera, _Mapping]] = ..., support_plane: _Optional[_Union[SupportPlane, _Mapping]] = ...) -> None: ...

class Camera(_message.Message):
    __slots__ = ("basis", "raster_width", "raster_height", "normalized_intrinsics", "world_basis", "camera_to_world")
    BASIS_FIELD_NUMBER: _ClassVar[int]
    RASTER_WIDTH_FIELD_NUMBER: _ClassVar[int]
    RASTER_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    NORMALIZED_INTRINSICS_FIELD_NUMBER: _ClassVar[int]
    WORLD_BASIS_FIELD_NUMBER: _ClassVar[int]
    CAMERA_TO_WORLD_FIELD_NUMBER: _ClassVar[int]
    basis: str
    raster_width: int
    raster_height: int
    normalized_intrinsics: _containers.RepeatedScalarFieldContainer[float]
    world_basis: str
    camera_to_world: _containers.RepeatedScalarFieldContainer[float]
    def __init__(self, basis: _Optional[str] = ..., raster_width: _Optional[int] = ..., raster_height: _Optional[int] = ..., normalized_intrinsics: _Optional[_Iterable[float]] = ..., world_basis: _Optional[str] = ..., camera_to_world: _Optional[_Iterable[float]] = ...) -> None: ...

class SupportPlane(_message.Message):
    __slots__ = ("normal", "offset", "valid_pixel_count", "inlier_count", "residual_p50", "residual_p95")
    NORMAL_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    VALID_PIXEL_COUNT_FIELD_NUMBER: _ClassVar[int]
    INLIER_COUNT_FIELD_NUMBER: _ClassVar[int]
    RESIDUAL_P50_FIELD_NUMBER: _ClassVar[int]
    RESIDUAL_P95_FIELD_NUMBER: _ClassVar[int]
    normal: _containers.RepeatedScalarFieldContainer[float]
    offset: float
    valid_pixel_count: int
    inlier_count: int
    residual_p50: float
    residual_p95: float
    def __init__(self, normal: _Optional[_Iterable[float]] = ..., offset: _Optional[float] = ..., valid_pixel_count: _Optional[int] = ..., inlier_count: _Optional[int] = ..., residual_p50: _Optional[float] = ..., residual_p95: _Optional[float] = ...) -> None: ...

class Background(_message.Message):
    __slots__ = ("clean_plate", "device_mask", "removal_mask", "occluder_masks", "repair_provenance")
    CLEAN_PLATE_FIELD_NUMBER: _ClassVar[int]
    DEVICE_MASK_FIELD_NUMBER: _ClassVar[int]
    REMOVAL_MASK_FIELD_NUMBER: _ClassVar[int]
    OCCLUDER_MASKS_FIELD_NUMBER: _ClassVar[int]
    REPAIR_PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    clean_plate: Artifact
    device_mask: Artifact
    removal_mask: Artifact
    occluder_masks: _containers.RepeatedCompositeFieldContainer[Artifact]
    repair_provenance: Artifact
    def __init__(self, clean_plate: _Optional[_Union[Artifact, _Mapping]] = ..., device_mask: _Optional[_Union[Artifact, _Mapping]] = ..., removal_mask: _Optional[_Union[Artifact, _Mapping]] = ..., occluder_masks: _Optional[_Iterable[_Union[Artifact, _Mapping]]] = ..., repair_provenance: _Optional[_Union[Artifact, _Mapping]] = ...) -> None: ...

class CameraEnvelope(_message.Message):
    __slots__ = ("max_translation_m", "max_rotation_degrees", "overscan_reviewed")
    MAX_TRANSLATION_M_FIELD_NUMBER: _ClassVar[int]
    MAX_ROTATION_DEGREES_FIELD_NUMBER: _ClassVar[int]
    OVERSCAN_REVIEWED_FIELD_NUMBER: _ClassVar[int]
    max_translation_m: float
    max_rotation_degrees: float
    overscan_reviewed: bool
    def __init__(self, max_translation_m: _Optional[float] = ..., max_rotation_degrees: _Optional[float] = ..., overscan_reviewed: _Optional[bool] = ...) -> None: ...

class VisualReview(_message.Message):
    __slots__ = ("decision", "reviewed_by", "rationale", "evidence_refs", "reviewed_at_unix_ms")
    DECISION_FIELD_NUMBER: _ClassVar[int]
    REVIEWED_BY_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    REVIEWED_AT_UNIX_MS_FIELD_NUMBER: _ClassVar[int]
    decision: str
    reviewed_by: str
    rationale: str
    evidence_refs: _containers.RepeatedScalarFieldContainer[str]
    reviewed_at_unix_ms: int
    def __init__(self, decision: _Optional[str] = ..., reviewed_by: _Optional[str] = ..., rationale: _Optional[str] = ..., evidence_refs: _Optional[_Iterable[str]] = ..., reviewed_at_unix_ms: _Optional[int] = ...) -> None: ...

class ArtifactBinding(_message.Message):
    __slots__ = ("artifact_path", "stored_ref", "inline_json")
    ARTIFACT_PATH_FIELD_NUMBER: _ClassVar[int]
    STORED_REF_FIELD_NUMBER: _ClassVar[int]
    INLINE_JSON_FIELD_NUMBER: _ClassVar[int]
    artifact_path: str
    stored_ref: str
    inline_json: str
    def __init__(self, artifact_path: _Optional[str] = ..., stored_ref: _Optional[str] = ..., inline_json: _Optional[str] = ...) -> None: ...

class CreatePhotographRequest(_message.Message):
    __slots__ = ("manifest", "artifacts", "label")
    MANIFEST_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    manifest: PreparedPhotograph
    artifacts: _containers.RepeatedCompositeFieldContainer[ArtifactBinding]
    label: str
    def __init__(self, manifest: _Optional[_Union[PreparedPhotograph, _Mapping]] = ..., artifacts: _Optional[_Iterable[_Union[ArtifactBinding, _Mapping]]] = ..., label: _Optional[str] = ...) -> None: ...

class StoredPhotograph(_message.Message):
    __slots__ = ("id", "label", "bundle_ref", "manifest", "parent_id", "created_at_unix_ms", "source_preview_ref", "clean_plate_preview_ref", "repair_assessment")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_REF_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_FIELD_NUMBER: _ClassVar[int]
    PARENT_ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_UNIX_MS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PREVIEW_REF_FIELD_NUMBER: _ClassVar[int]
    CLEAN_PLATE_PREVIEW_REF_FIELD_NUMBER: _ClassVar[int]
    REPAIR_ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    bundle_ref: str
    manifest: PreparedPhotograph
    parent_id: str
    created_at_unix_ms: int
    source_preview_ref: str
    clean_plate_preview_ref: str
    repair_assessment: RepairAssessment
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., bundle_ref: _Optional[str] = ..., manifest: _Optional[_Union[PreparedPhotograph, _Mapping]] = ..., parent_id: _Optional[str] = ..., created_at_unix_ms: _Optional[int] = ..., source_preview_ref: _Optional[str] = ..., clean_plate_preview_ref: _Optional[str] = ..., repair_assessment: _Optional[_Union[RepairAssessment, _Mapping]] = ...) -> None: ...

class PhotographResponse(_message.Message):
    __slots__ = ("photograph",)
    PHOTOGRAPH_FIELD_NUMBER: _ClassVar[int]
    photograph: StoredPhotograph
    def __init__(self, photograph: _Optional[_Union[StoredPhotograph, _Mapping]] = ...) -> None: ...

class GetPhotographRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ListPhotographsRequest(_message.Message):
    __slots__ = ("limit",)
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    def __init__(self, limit: _Optional[int] = ...) -> None: ...

class ListPhotographsResponse(_message.Message):
    __slots__ = ("photographs",)
    PHOTOGRAPHS_FIELD_NUMBER: _ClassVar[int]
    photographs: _containers.RepeatedCompositeFieldContainer[StoredPhotograph]
    def __init__(self, photographs: _Optional[_Iterable[_Union[StoredPhotograph, _Mapping]]] = ...) -> None: ...

class ReviewPhotographRequest(_message.Message):
    __slots__ = ("id", "review", "overscan_reviewed", "operation_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    REVIEW_FIELD_NUMBER: _ClassVar[int]
    OVERSCAN_REVIEWED_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    review: VisualReview
    overscan_reviewed: bool
    operation_id: str
    def __init__(self, id: _Optional[str] = ..., review: _Optional[_Union[VisualReview, _Mapping]] = ..., overscan_reviewed: _Optional[bool] = ..., operation_id: _Optional[str] = ...) -> None: ...

class RepairAssessment(_message.Message):
    __slots__ = ("mask_area_fraction", "risk_level", "risk_signals", "context_pixels", "context_clipped_sides", "inference_scale")
    MASK_AREA_FRACTION_FIELD_NUMBER: _ClassVar[int]
    RISK_LEVEL_FIELD_NUMBER: _ClassVar[int]
    RISK_SIGNALS_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_PIXELS_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_CLIPPED_SIDES_FIELD_NUMBER: _ClassVar[int]
    INFERENCE_SCALE_FIELD_NUMBER: _ClassVar[int]
    mask_area_fraction: float
    risk_level: str
    risk_signals: _containers.RepeatedScalarFieldContainer[str]
    context_pixels: int
    context_clipped_sides: _containers.RepeatedScalarFieldContainer[str]
    inference_scale: float
    def __init__(self, mask_area_fraction: _Optional[float] = ..., risk_level: _Optional[str] = ..., risk_signals: _Optional[_Iterable[str]] = ..., context_pixels: _Optional[int] = ..., context_clipped_sides: _Optional[_Iterable[str]] = ..., inference_scale: _Optional[float] = ...) -> None: ...

class AssessRepairRequest(_message.Message):
    __slots__ = ("mask_ref", "inpaint")
    MASK_REF_FIELD_NUMBER: _ClassVar[int]
    INPAINT_FIELD_NUMBER: _ClassVar[int]
    mask_ref: str
    inpaint: _ai_pb2.InpaintParams
    def __init__(self, mask_ref: _Optional[str] = ..., inpaint: _Optional[_Union[_ai_pb2.InpaintParams, _Mapping]] = ...) -> None: ...

class AssessRepairResponse(_message.Message):
    __slots__ = ("assessment",)
    ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    assessment: RepairAssessment
    def __init__(self, assessment: _Optional[_Union[RepairAssessment, _Mapping]] = ...) -> None: ...
