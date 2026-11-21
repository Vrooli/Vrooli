import datetime

from device_studio.v1.physical import physical_pb2 as _physical_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ImportPreparedPhotographRequest(_message.Message):
    __slots__ = ("preparation_id", "expected_bundle_sha256", "license_id", "attribution", "source_label", "allow_local_render", "source_use_rationale")
    PREPARATION_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_BUNDLE_SHA256_FIELD_NUMBER: _ClassVar[int]
    LICENSE_ID_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_LABEL_FIELD_NUMBER: _ClassVar[int]
    ALLOW_LOCAL_RENDER_FIELD_NUMBER: _ClassVar[int]
    SOURCE_USE_RATIONALE_FIELD_NUMBER: _ClassVar[int]
    preparation_id: str
    expected_bundle_sha256: str
    license_id: str
    attribution: str
    source_label: str
    allow_local_render: bool
    source_use_rationale: str
    def __init__(self, preparation_id: _Optional[str] = ..., expected_bundle_sha256: _Optional[str] = ..., license_id: _Optional[str] = ..., attribution: _Optional[str] = ..., source_label: _Optional[str] = ..., allow_local_render: _Optional[bool] = ..., source_use_rationale: _Optional[str] = ...) -> None: ...

class ImportPreparedPhotographResponse(_message.Message):
    __slots__ = ("revision", "library_bundle_id", "library_revision", "library_manifest_sha256", "scale_status", "estimated_scale_m")
    REVISION_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_BUNDLE_ID_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_REVISION_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_MANIFEST_SHA256_FIELD_NUMBER: _ClassVar[int]
    SCALE_STATUS_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_SCALE_M_FIELD_NUMBER: _ClassVar[int]
    revision: SceneRevision
    library_bundle_id: str
    library_revision: int
    library_manifest_sha256: str
    scale_status: str
    estimated_scale_m: float
    def __init__(self, revision: _Optional[_Union[SceneRevision, _Mapping]] = ..., library_bundle_id: _Optional[str] = ..., library_revision: _Optional[int] = ..., library_manifest_sha256: _Optional[str] = ..., scale_status: _Optional[str] = ..., estimated_scale_m: _Optional[float] = ...) -> None: ...

class FitKnownCameraPoseRequest(_message.Message):
    __slots__ = ("scene_id", "scene_revision", "camera_frame_id", "raster_width", "raster_height", "landmarks", "initial_pose_world", "hinge_joint_id", "hinge_angle_radians", "hinge_rule", "max_rms_pixels", "max_point_pixels", "geometry_asset_id", "geometry_sha256", "device_id", "device_revision", "device_revision_sha256")
    SCENE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENE_REVISION_FIELD_NUMBER: _ClassVar[int]
    CAMERA_FRAME_ID_FIELD_NUMBER: _ClassVar[int]
    RASTER_WIDTH_FIELD_NUMBER: _ClassVar[int]
    RASTER_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    LANDMARKS_FIELD_NUMBER: _ClassVar[int]
    INITIAL_POSE_WORLD_FIELD_NUMBER: _ClassVar[int]
    HINGE_JOINT_ID_FIELD_NUMBER: _ClassVar[int]
    HINGE_ANGLE_RADIANS_FIELD_NUMBER: _ClassVar[int]
    HINGE_RULE_FIELD_NUMBER: _ClassVar[int]
    MAX_RMS_PIXELS_FIELD_NUMBER: _ClassVar[int]
    MAX_POINT_PIXELS_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_SHA256_FIELD_NUMBER: _ClassVar[int]
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    DEVICE_REVISION_FIELD_NUMBER: _ClassVar[int]
    DEVICE_REVISION_SHA256_FIELD_NUMBER: _ClassVar[int]
    scene_id: str
    scene_revision: int
    camera_frame_id: str
    raster_width: int
    raster_height: int
    landmarks: _containers.RepeatedCompositeFieldContainer[DeviceFitLandmark]
    initial_pose_world: _physical_pb2.TRS
    hinge_joint_id: str
    hinge_angle_radians: float
    hinge_rule: str
    max_rms_pixels: float
    max_point_pixels: float
    geometry_asset_id: str
    geometry_sha256: str
    device_id: str
    device_revision: int
    device_revision_sha256: str
    def __init__(self, scene_id: _Optional[str] = ..., scene_revision: _Optional[int] = ..., camera_frame_id: _Optional[str] = ..., raster_width: _Optional[int] = ..., raster_height: _Optional[int] = ..., landmarks: _Optional[_Iterable[_Union[DeviceFitLandmark, _Mapping]]] = ..., initial_pose_world: _Optional[_Union[_physical_pb2.TRS, _Mapping]] = ..., hinge_joint_id: _Optional[str] = ..., hinge_angle_radians: _Optional[float] = ..., hinge_rule: _Optional[str] = ..., max_rms_pixels: _Optional[float] = ..., max_point_pixels: _Optional[float] = ..., geometry_asset_id: _Optional[str] = ..., geometry_sha256: _Optional[str] = ..., device_id: _Optional[str] = ..., device_revision: _Optional[int] = ..., device_revision_sha256: _Optional[str] = ...) -> None: ...

class DeviceFitLandmark(_message.Message):
    __slots__ = ("id", "device_local_m", "pixel_x", "pixel_y")
    ID_FIELD_NUMBER: _ClassVar[int]
    DEVICE_LOCAL_M_FIELD_NUMBER: _ClassVar[int]
    PIXEL_X_FIELD_NUMBER: _ClassVar[int]
    PIXEL_Y_FIELD_NUMBER: _ClassVar[int]
    id: str
    device_local_m: _physical_pb2.Vec3
    pixel_x: float
    pixel_y: float
    def __init__(self, id: _Optional[str] = ..., device_local_m: _Optional[_Union[_physical_pb2.Vec3, _Mapping]] = ..., pixel_x: _Optional[float] = ..., pixel_y: _Optional[float] = ...) -> None: ...

class FitKnownCameraPoseResponse(_message.Message):
    __slots__ = ("profile", "fitted_pose_world", "initial_pose_world", "hinge_joint_id", "hinge_angle_radians", "hinge_rule", "residuals", "rms_pixels", "max_pixels", "overlay_svg", "iterations", "scene_id", "scene_revision", "camera_frame_id", "geometry_asset_id", "geometry_sha256", "device_id", "device_revision", "device_revision_sha256", "camera_frame", "raster_width", "raster_height")
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    FITTED_POSE_WORLD_FIELD_NUMBER: _ClassVar[int]
    INITIAL_POSE_WORLD_FIELD_NUMBER: _ClassVar[int]
    HINGE_JOINT_ID_FIELD_NUMBER: _ClassVar[int]
    HINGE_ANGLE_RADIANS_FIELD_NUMBER: _ClassVar[int]
    HINGE_RULE_FIELD_NUMBER: _ClassVar[int]
    RESIDUALS_FIELD_NUMBER: _ClassVar[int]
    RMS_PIXELS_FIELD_NUMBER: _ClassVar[int]
    MAX_PIXELS_FIELD_NUMBER: _ClassVar[int]
    OVERLAY_SVG_FIELD_NUMBER: _ClassVar[int]
    ITERATIONS_FIELD_NUMBER: _ClassVar[int]
    SCENE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENE_REVISION_FIELD_NUMBER: _ClassVar[int]
    CAMERA_FRAME_ID_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_SHA256_FIELD_NUMBER: _ClassVar[int]
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    DEVICE_REVISION_FIELD_NUMBER: _ClassVar[int]
    DEVICE_REVISION_SHA256_FIELD_NUMBER: _ClassVar[int]
    CAMERA_FRAME_FIELD_NUMBER: _ClassVar[int]
    RASTER_WIDTH_FIELD_NUMBER: _ClassVar[int]
    RASTER_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    profile: str
    fitted_pose_world: _physical_pb2.TRS
    initial_pose_world: _physical_pb2.TRS
    hinge_joint_id: str
    hinge_angle_radians: float
    hinge_rule: str
    residuals: _containers.RepeatedCompositeFieldContainer[DeviceFitResidual]
    rms_pixels: float
    max_pixels: float
    overlay_svg: str
    iterations: int
    scene_id: str
    scene_revision: int
    camera_frame_id: str
    geometry_asset_id: str
    geometry_sha256: str
    device_id: str
    device_revision: int
    device_revision_sha256: str
    camera_frame: SceneCameraFrame
    raster_width: int
    raster_height: int
    def __init__(self, profile: _Optional[str] = ..., fitted_pose_world: _Optional[_Union[_physical_pb2.TRS, _Mapping]] = ..., initial_pose_world: _Optional[_Union[_physical_pb2.TRS, _Mapping]] = ..., hinge_joint_id: _Optional[str] = ..., hinge_angle_radians: _Optional[float] = ..., hinge_rule: _Optional[str] = ..., residuals: _Optional[_Iterable[_Union[DeviceFitResidual, _Mapping]]] = ..., rms_pixels: _Optional[float] = ..., max_pixels: _Optional[float] = ..., overlay_svg: _Optional[str] = ..., iterations: _Optional[int] = ..., scene_id: _Optional[str] = ..., scene_revision: _Optional[int] = ..., camera_frame_id: _Optional[str] = ..., geometry_asset_id: _Optional[str] = ..., geometry_sha256: _Optional[str] = ..., device_id: _Optional[str] = ..., device_revision: _Optional[int] = ..., device_revision_sha256: _Optional[str] = ..., camera_frame: _Optional[_Union[SceneCameraFrame, _Mapping]] = ..., raster_width: _Optional[int] = ..., raster_height: _Optional[int] = ...) -> None: ...

class DeviceFitResidual(_message.Message):
    __slots__ = ("landmark_id", "observed_x", "observed_y", "projected_x", "projected_y", "dx", "dy", "magnitude")
    LANDMARK_ID_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_X_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_Y_FIELD_NUMBER: _ClassVar[int]
    PROJECTED_X_FIELD_NUMBER: _ClassVar[int]
    PROJECTED_Y_FIELD_NUMBER: _ClassVar[int]
    DX_FIELD_NUMBER: _ClassVar[int]
    DY_FIELD_NUMBER: _ClassVar[int]
    MAGNITUDE_FIELD_NUMBER: _ClassVar[int]
    landmark_id: str
    observed_x: float
    observed_y: float
    projected_x: float
    projected_y: float
    dx: float
    dy: float
    magnitude: float
    def __init__(self, landmark_id: _Optional[str] = ..., observed_x: _Optional[float] = ..., observed_y: _Optional[float] = ..., projected_x: _Optional[float] = ..., projected_y: _Optional[float] = ..., dx: _Optional[float] = ..., dy: _Optional[float] = ..., magnitude: _Optional[float] = ...) -> None: ...

class SaveSceneCandidateRequest(_message.Message):
    __slots__ = ("candidate", "expected_head")
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_HEAD_FIELD_NUMBER: _ClassVar[int]
    candidate: SceneCandidate
    expected_head: int
    def __init__(self, candidate: _Optional[_Union[SceneCandidate, _Mapping]] = ..., expected_head: _Optional[int] = ...) -> None: ...

class SaveSceneCandidateResponse(_message.Message):
    __slots__ = ("revision",)
    REVISION_FIELD_NUMBER: _ClassVar[int]
    revision: SceneRevision
    def __init__(self, revision: _Optional[_Union[SceneRevision, _Mapping]] = ...) -> None: ...

class GetSceneRevisionRequest(_message.Message):
    __slots__ = ("scene_id", "revision")
    SCENE_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    scene_id: str
    revision: int
    def __init__(self, scene_id: _Optional[str] = ..., revision: _Optional[int] = ...) -> None: ...

class GetSceneRevisionResponse(_message.Message):
    __slots__ = ("revision",)
    REVISION_FIELD_NUMBER: _ClassVar[int]
    revision: SceneRevision
    def __init__(self, revision: _Optional[_Union[SceneRevision, _Mapping]] = ...) -> None: ...

class GetSceneHeadRequest(_message.Message):
    __slots__ = ("scene_id",)
    SCENE_ID_FIELD_NUMBER: _ClassVar[int]
    scene_id: str
    def __init__(self, scene_id: _Optional[str] = ...) -> None: ...

class GetSceneHeadResponse(_message.Message):
    __slots__ = ("revision",)
    REVISION_FIELD_NUMBER: _ClassVar[int]
    revision: SceneRevision
    def __init__(self, revision: _Optional[_Union[SceneRevision, _Mapping]] = ...) -> None: ...

class ListSceneRevisionsRequest(_message.Message):
    __slots__ = ("scene_id", "after_revision", "page_size")
    SCENE_ID_FIELD_NUMBER: _ClassVar[int]
    AFTER_REVISION_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    scene_id: str
    after_revision: int
    page_size: int
    def __init__(self, scene_id: _Optional[str] = ..., after_revision: _Optional[int] = ..., page_size: _Optional[int] = ...) -> None: ...

class ListSceneRevisionsResponse(_message.Message):
    __slots__ = ("revisions", "next_after_revision")
    REVISIONS_FIELD_NUMBER: _ClassVar[int]
    NEXT_AFTER_REVISION_FIELD_NUMBER: _ClassVar[int]
    revisions: _containers.RepeatedCompositeFieldContainer[SceneRevision]
    next_after_revision: int
    def __init__(self, revisions: _Optional[_Iterable[_Union[SceneRevision, _Mapping]]] = ..., next_after_revision: _Optional[int] = ...) -> None: ...

class ListSceneCandidatesRequest(_message.Message):
    __slots__ = ("after_scene_id", "page_size")
    AFTER_SCENE_ID_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    after_scene_id: str
    page_size: int
    def __init__(self, after_scene_id: _Optional[str] = ..., page_size: _Optional[int] = ...) -> None: ...

class ListSceneCandidatesResponse(_message.Message):
    __slots__ = ("candidates", "next_after_scene_id")
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    NEXT_AFTER_SCENE_ID_FIELD_NUMBER: _ClassVar[int]
    candidates: _containers.RepeatedCompositeFieldContainer[SceneRevision]
    next_after_scene_id: str
    def __init__(self, candidates: _Optional[_Iterable[_Union[SceneRevision, _Mapping]]] = ..., next_after_scene_id: _Optional[str] = ...) -> None: ...

class SceneRevision(_message.Message):
    __slots__ = ("scene_id", "revision", "sha256", "state", "candidate", "created_at")
    SCENE_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    scene_id: str
    revision: int
    sha256: str
    state: str
    candidate: SceneCandidate
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, scene_id: _Optional[str] = ..., revision: _Optional[int] = ..., sha256: _Optional[str] = ..., state: _Optional[str] = ..., candidate: _Optional[_Union[SceneCandidate, _Mapping]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class SceneCandidate(_message.Message):
    __slots__ = ("version", "scene_id", "coordinate_convention", "artifacts", "materials", "material_slots", "parts", "support_frames", "camera_frames", "light_frames", "environment", "regions", "support_slots", "photographic_background")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    SCENE_ID_FIELD_NUMBER: _ClassVar[int]
    COORDINATE_CONVENTION_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    MATERIALS_FIELD_NUMBER: _ClassVar[int]
    MATERIAL_SLOTS_FIELD_NUMBER: _ClassVar[int]
    PARTS_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_FRAMES_FIELD_NUMBER: _ClassVar[int]
    CAMERA_FRAMES_FIELD_NUMBER: _ClassVar[int]
    LIGHT_FRAMES_FIELD_NUMBER: _ClassVar[int]
    ENVIRONMENT_FIELD_NUMBER: _ClassVar[int]
    REGIONS_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_SLOTS_FIELD_NUMBER: _ClassVar[int]
    PHOTOGRAPHIC_BACKGROUND_FIELD_NUMBER: _ClassVar[int]
    version: str
    scene_id: str
    coordinate_convention: str
    artifacts: _containers.RepeatedCompositeFieldContainer[SceneArtifactBinding]
    materials: _containers.RepeatedCompositeFieldContainer[SceneMaterial]
    material_slots: _containers.RepeatedCompositeFieldContainer[SceneMaterialSlot]
    parts: _containers.RepeatedCompositeFieldContainer[ScenePart]
    support_frames: _containers.RepeatedCompositeFieldContainer[SceneSupportFrame]
    camera_frames: _containers.RepeatedCompositeFieldContainer[SceneCameraFrame]
    light_frames: _containers.RepeatedCompositeFieldContainer[SceneLightFrame]
    environment: SceneEnvironment
    regions: _containers.RepeatedCompositeFieldContainer[SceneRegion]
    support_slots: _containers.RepeatedCompositeFieldContainer[SceneSupportSlot]
    photographic_background: PhotographicBackground
    def __init__(self, version: _Optional[str] = ..., scene_id: _Optional[str] = ..., coordinate_convention: _Optional[str] = ..., artifacts: _Optional[_Iterable[_Union[SceneArtifactBinding, _Mapping]]] = ..., materials: _Optional[_Iterable[_Union[SceneMaterial, _Mapping]]] = ..., material_slots: _Optional[_Iterable[_Union[SceneMaterialSlot, _Mapping]]] = ..., parts: _Optional[_Iterable[_Union[ScenePart, _Mapping]]] = ..., support_frames: _Optional[_Iterable[_Union[SceneSupportFrame, _Mapping]]] = ..., camera_frames: _Optional[_Iterable[_Union[SceneCameraFrame, _Mapping]]] = ..., light_frames: _Optional[_Iterable[_Union[SceneLightFrame, _Mapping]]] = ..., environment: _Optional[_Union[SceneEnvironment, _Mapping]] = ..., regions: _Optional[_Iterable[_Union[SceneRegion, _Mapping]]] = ..., support_slots: _Optional[_Iterable[_Union[SceneSupportSlot, _Mapping]]] = ..., photographic_background: _Optional[_Union[PhotographicBackground, _Mapping]] = ...) -> None: ...

class PhotographicBackground(_message.Message):
    __slots__ = ("contract_version", "route", "source_bundle_sha256", "source_raster_sha256", "geometry_archive_sha256", "repair_provenance_sha256", "device_mask_sha256", "removal_mask_sha256", "scale_status", "estimated_scale_meters", "geometry_model", "checkpoint_sha256", "postprocess", "support_plane_normal_opencv", "support_plane_offset_estimate_m", "support_plane_inliers", "support_plane_residual_p95_estimate_m", "camera_envelope_max_translation_m", "camera_envelope_max_rotation_degrees", "overscan_reviewed", "source_visual_review_sha256", "source_visual_reviewer", "source_visual_review_rationale", "source_visual_review_evidence", "source_use_claim_sha256", "local_render_authorized", "source_use_rationale", "render_mode")
    CONTRACT_VERSION_FIELD_NUMBER: _ClassVar[int]
    ROUTE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_BUNDLE_SHA256_FIELD_NUMBER: _ClassVar[int]
    SOURCE_RASTER_SHA256_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_ARCHIVE_SHA256_FIELD_NUMBER: _ClassVar[int]
    REPAIR_PROVENANCE_SHA256_FIELD_NUMBER: _ClassVar[int]
    DEVICE_MASK_SHA256_FIELD_NUMBER: _ClassVar[int]
    REMOVAL_MASK_SHA256_FIELD_NUMBER: _ClassVar[int]
    SCALE_STATUS_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_SCALE_METERS_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_MODEL_FIELD_NUMBER: _ClassVar[int]
    CHECKPOINT_SHA256_FIELD_NUMBER: _ClassVar[int]
    POSTPROCESS_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_PLANE_NORMAL_OPENCV_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_PLANE_OFFSET_ESTIMATE_M_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_PLANE_INLIERS_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_PLANE_RESIDUAL_P95_ESTIMATE_M_FIELD_NUMBER: _ClassVar[int]
    CAMERA_ENVELOPE_MAX_TRANSLATION_M_FIELD_NUMBER: _ClassVar[int]
    CAMERA_ENVELOPE_MAX_ROTATION_DEGREES_FIELD_NUMBER: _ClassVar[int]
    OVERSCAN_REVIEWED_FIELD_NUMBER: _ClassVar[int]
    SOURCE_VISUAL_REVIEW_SHA256_FIELD_NUMBER: _ClassVar[int]
    SOURCE_VISUAL_REVIEWER_FIELD_NUMBER: _ClassVar[int]
    SOURCE_VISUAL_REVIEW_RATIONALE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_VISUAL_REVIEW_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_USE_CLAIM_SHA256_FIELD_NUMBER: _ClassVar[int]
    LOCAL_RENDER_AUTHORIZED_FIELD_NUMBER: _ClassVar[int]
    SOURCE_USE_RATIONALE_FIELD_NUMBER: _ClassVar[int]
    RENDER_MODE_FIELD_NUMBER: _ClassVar[int]
    contract_version: str
    route: str
    source_bundle_sha256: str
    source_raster_sha256: str
    geometry_archive_sha256: str
    repair_provenance_sha256: str
    device_mask_sha256: str
    removal_mask_sha256: str
    scale_status: str
    estimated_scale_meters: float
    geometry_model: str
    checkpoint_sha256: str
    postprocess: str
    support_plane_normal_opencv: _containers.RepeatedScalarFieldContainer[float]
    support_plane_offset_estimate_m: float
    support_plane_inliers: int
    support_plane_residual_p95_estimate_m: float
    camera_envelope_max_translation_m: float
    camera_envelope_max_rotation_degrees: float
    overscan_reviewed: bool
    source_visual_review_sha256: str
    source_visual_reviewer: str
    source_visual_review_rationale: str
    source_visual_review_evidence: _containers.RepeatedScalarFieldContainer[str]
    source_use_claim_sha256: str
    local_render_authorized: bool
    source_use_rationale: str
    render_mode: str
    def __init__(self, contract_version: _Optional[str] = ..., route: _Optional[str] = ..., source_bundle_sha256: _Optional[str] = ..., source_raster_sha256: _Optional[str] = ..., geometry_archive_sha256: _Optional[str] = ..., repair_provenance_sha256: _Optional[str] = ..., device_mask_sha256: _Optional[str] = ..., removal_mask_sha256: _Optional[str] = ..., scale_status: _Optional[str] = ..., estimated_scale_meters: _Optional[float] = ..., geometry_model: _Optional[str] = ..., checkpoint_sha256: _Optional[str] = ..., postprocess: _Optional[str] = ..., support_plane_normal_opencv: _Optional[_Iterable[float]] = ..., support_plane_offset_estimate_m: _Optional[float] = ..., support_plane_inliers: _Optional[int] = ..., support_plane_residual_p95_estimate_m: _Optional[float] = ..., camera_envelope_max_translation_m: _Optional[float] = ..., camera_envelope_max_rotation_degrees: _Optional[float] = ..., overscan_reviewed: _Optional[bool] = ..., source_visual_review_sha256: _Optional[str] = ..., source_visual_reviewer: _Optional[str] = ..., source_visual_review_rationale: _Optional[str] = ..., source_visual_review_evidence: _Optional[_Iterable[str]] = ..., source_use_claim_sha256: _Optional[str] = ..., local_render_authorized: _Optional[bool] = ..., source_use_rationale: _Optional[str] = ..., render_mode: _Optional[str] = ...) -> None: ...

class SceneArtifactBinding(_message.Message):
    __slots__ = ("id", "ref", "source_unit", "source_to_canonical")
    ID_FIELD_NUMBER: _ClassVar[int]
    REF_FIELD_NUMBER: _ClassVar[int]
    SOURCE_UNIT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_TO_CANONICAL_FIELD_NUMBER: _ClassVar[int]
    id: str
    ref: _physical_pb2.ArtifactRef
    source_unit: str
    source_to_canonical: _physical_pb2.TRS
    def __init__(self, id: _Optional[str] = ..., ref: _Optional[_Union[_physical_pb2.ArtifactRef, _Mapping]] = ..., source_unit: _Optional[str] = ..., source_to_canonical: _Optional[_Union[_physical_pb2.TRS, _Mapping]] = ...) -> None: ...

class SceneMaterialInput(_message.Message):
    __slots__ = ("socket", "texture_binding_id", "semantic", "color_space", "channel", "sampler", "coordinates", "uv_set", "affine_uv", "physical_repeat_m", "normal_convention", "normal_strength", "bump_strength", "bump_distance_m")
    SOCKET_FIELD_NUMBER: _ClassVar[int]
    TEXTURE_BINDING_ID_FIELD_NUMBER: _ClassVar[int]
    SEMANTIC_FIELD_NUMBER: _ClassVar[int]
    COLOR_SPACE_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    SAMPLER_FIELD_NUMBER: _ClassVar[int]
    COORDINATES_FIELD_NUMBER: _ClassVar[int]
    UV_SET_FIELD_NUMBER: _ClassVar[int]
    AFFINE_UV_FIELD_NUMBER: _ClassVar[int]
    PHYSICAL_REPEAT_M_FIELD_NUMBER: _ClassVar[int]
    NORMAL_CONVENTION_FIELD_NUMBER: _ClassVar[int]
    NORMAL_STRENGTH_FIELD_NUMBER: _ClassVar[int]
    BUMP_STRENGTH_FIELD_NUMBER: _ClassVar[int]
    BUMP_DISTANCE_M_FIELD_NUMBER: _ClassVar[int]
    socket: str
    texture_binding_id: str
    semantic: str
    color_space: str
    channel: str
    sampler: str
    coordinates: str
    uv_set: str
    affine_uv: _containers.RepeatedScalarFieldContainer[float]
    physical_repeat_m: _containers.RepeatedScalarFieldContainer[float]
    normal_convention: str
    normal_strength: float
    bump_strength: float
    bump_distance_m: float
    def __init__(self, socket: _Optional[str] = ..., texture_binding_id: _Optional[str] = ..., semantic: _Optional[str] = ..., color_space: _Optional[str] = ..., channel: _Optional[str] = ..., sampler: _Optional[str] = ..., coordinates: _Optional[str] = ..., uv_set: _Optional[str] = ..., affine_uv: _Optional[_Iterable[float]] = ..., physical_repeat_m: _Optional[_Iterable[float]] = ..., normal_convention: _Optional[str] = ..., normal_strength: _Optional[float] = ..., bump_strength: _Optional[float] = ..., bump_distance_m: _Optional[float] = ...) -> None: ...

class SceneMaterialGrade(_message.Message):
    __slots__ = ("recipe", "saturation", "value", "mix_factor", "mix_linear_rgb")
    RECIPE_FIELD_NUMBER: _ClassVar[int]
    SATURATION_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    MIX_FACTOR_FIELD_NUMBER: _ClassVar[int]
    MIX_LINEAR_RGB_FIELD_NUMBER: _ClassVar[int]
    recipe: str
    saturation: float
    value: float
    mix_factor: float
    mix_linear_rgb: _containers.RepeatedScalarFieldContainer[float]
    def __init__(self, recipe: _Optional[str] = ..., saturation: _Optional[float] = ..., value: _Optional[float] = ..., mix_factor: _Optional[float] = ..., mix_linear_rgb: _Optional[_Iterable[float]] = ...) -> None: ...

class SceneMaterial(_message.Message):
    __slots__ = ("id", "surface_profile", "base_color_linear_rgba", "metallic", "roughness", "alpha_mode", "alpha_cutoff", "inputs", "grade")
    ID_FIELD_NUMBER: _ClassVar[int]
    SURFACE_PROFILE_FIELD_NUMBER: _ClassVar[int]
    BASE_COLOR_LINEAR_RGBA_FIELD_NUMBER: _ClassVar[int]
    METALLIC_FIELD_NUMBER: _ClassVar[int]
    ROUGHNESS_FIELD_NUMBER: _ClassVar[int]
    ALPHA_MODE_FIELD_NUMBER: _ClassVar[int]
    ALPHA_CUTOFF_FIELD_NUMBER: _ClassVar[int]
    INPUTS_FIELD_NUMBER: _ClassVar[int]
    GRADE_FIELD_NUMBER: _ClassVar[int]
    id: str
    surface_profile: str
    base_color_linear_rgba: _containers.RepeatedScalarFieldContainer[float]
    metallic: float
    roughness: float
    alpha_mode: str
    alpha_cutoff: float
    inputs: _containers.RepeatedCompositeFieldContainer[SceneMaterialInput]
    grade: SceneMaterialGrade
    def __init__(self, id: _Optional[str] = ..., surface_profile: _Optional[str] = ..., base_color_linear_rgba: _Optional[_Iterable[float]] = ..., metallic: _Optional[float] = ..., roughness: _Optional[float] = ..., alpha_mode: _Optional[str] = ..., alpha_cutoff: _Optional[float] = ..., inputs: _Optional[_Iterable[_Union[SceneMaterialInput, _Mapping]]] = ..., grade: _Optional[_Union[SceneMaterialGrade, _Mapping]] = ...) -> None: ...

class SceneMaterialSlot(_message.Message):
    __slots__ = ("part_id", "primitive_index", "slot_index", "material_id")
    PART_ID_FIELD_NUMBER: _ClassVar[int]
    PRIMITIVE_INDEX_FIELD_NUMBER: _ClassVar[int]
    SLOT_INDEX_FIELD_NUMBER: _ClassVar[int]
    MATERIAL_ID_FIELD_NUMBER: _ClassVar[int]
    part_id: str
    primitive_index: int
    slot_index: int
    material_id: str
    def __init__(self, part_id: _Optional[str] = ..., primitive_index: _Optional[int] = ..., slot_index: _Optional[int] = ..., material_id: _Optional[str] = ...) -> None: ...

class ScenePart(_message.Message):
    __slots__ = ("id", "role", "artifact_binding_id", "parent_id", "node_index", "node_name", "local_frame")
    ID_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_BINDING_ID_FIELD_NUMBER: _ClassVar[int]
    PARENT_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_INDEX_FIELD_NUMBER: _ClassVar[int]
    NODE_NAME_FIELD_NUMBER: _ClassVar[int]
    LOCAL_FRAME_FIELD_NUMBER: _ClassVar[int]
    id: str
    role: str
    artifact_binding_id: str
    parent_id: str
    node_index: int
    node_name: str
    local_frame: _physical_pb2.TRS
    def __init__(self, id: _Optional[str] = ..., role: _Optional[str] = ..., artifact_binding_id: _Optional[str] = ..., parent_id: _Optional[str] = ..., node_index: _Optional[int] = ..., node_name: _Optional[str] = ..., local_frame: _Optional[_Union[_physical_pb2.TRS, _Mapping]] = ...) -> None: ...

class SceneSupportFrame(_message.Message):
    __slots__ = ("id", "interface_id", "frame_part_id", "local_frame", "contact_region_part_id", "load_direction_local")
    ID_FIELD_NUMBER: _ClassVar[int]
    INTERFACE_ID_FIELD_NUMBER: _ClassVar[int]
    FRAME_PART_ID_FIELD_NUMBER: _ClassVar[int]
    LOCAL_FRAME_FIELD_NUMBER: _ClassVar[int]
    CONTACT_REGION_PART_ID_FIELD_NUMBER: _ClassVar[int]
    LOAD_DIRECTION_LOCAL_FIELD_NUMBER: _ClassVar[int]
    id: str
    interface_id: str
    frame_part_id: str
    local_frame: _physical_pb2.TRS
    contact_region_part_id: str
    load_direction_local: _physical_pb2.Vec3
    def __init__(self, id: _Optional[str] = ..., interface_id: _Optional[str] = ..., frame_part_id: _Optional[str] = ..., local_frame: _Optional[_Union[_physical_pb2.TRS, _Mapping]] = ..., contact_region_part_id: _Optional[str] = ..., load_direction_local: _Optional[_Union[_physical_pb2.Vec3, _Mapping]] = ...) -> None: ...

class SceneSupportSlot(_message.Message):
    __slots__ = ("id", "frame_id", "kind", "required_interface", "allowed_groups", "dimension_state", "required_pose", "max_width_m", "max_height_m", "max_depth_m")
    ID_FIELD_NUMBER: _ClassVar[int]
    FRAME_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_INTERFACE_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_GROUPS_FIELD_NUMBER: _ClassVar[int]
    DIMENSION_STATE_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_POSE_FIELD_NUMBER: _ClassVar[int]
    MAX_WIDTH_M_FIELD_NUMBER: _ClassVar[int]
    MAX_HEIGHT_M_FIELD_NUMBER: _ClassVar[int]
    MAX_DEPTH_M_FIELD_NUMBER: _ClassVar[int]
    id: str
    frame_id: str
    kind: str
    required_interface: str
    allowed_groups: _containers.RepeatedScalarFieldContainer[str]
    dimension_state: str
    required_pose: str
    max_width_m: float
    max_height_m: float
    max_depth_m: float
    def __init__(self, id: _Optional[str] = ..., frame_id: _Optional[str] = ..., kind: _Optional[str] = ..., required_interface: _Optional[str] = ..., allowed_groups: _Optional[_Iterable[str]] = ..., dimension_state: _Optional[str] = ..., required_pose: _Optional[str] = ..., max_width_m: _Optional[float] = ..., max_height_m: _Optional[float] = ..., max_depth_m: _Optional[float] = ...) -> None: ...

class FocusTarget(_message.Message):
    __slots__ = ("mode", "scene_part_id", "distance_m")
    MODE_FIELD_NUMBER: _ClassVar[int]
    SCENE_PART_ID_FIELD_NUMBER: _ClassVar[int]
    DISTANCE_M_FIELD_NUMBER: _ClassVar[int]
    mode: str
    scene_part_id: str
    distance_m: float
    def __init__(self, mode: _Optional[str] = ..., scene_part_id: _Optional[str] = ..., distance_m: _Optional[float] = ...) -> None: ...

class SceneCameraFrame(_message.Message):
    __slots__ = ("id", "frame", "projection", "focal_length_mm", "sensor_width_mm", "sensor_height_mm", "sensor_fit", "near_clip_m", "far_clip_m", "orthographic_vertical_span_m", "focus", "dof_f_number")
    ID_FIELD_NUMBER: _ClassVar[int]
    FRAME_FIELD_NUMBER: _ClassVar[int]
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    FOCAL_LENGTH_MM_FIELD_NUMBER: _ClassVar[int]
    SENSOR_WIDTH_MM_FIELD_NUMBER: _ClassVar[int]
    SENSOR_HEIGHT_MM_FIELD_NUMBER: _ClassVar[int]
    SENSOR_FIT_FIELD_NUMBER: _ClassVar[int]
    NEAR_CLIP_M_FIELD_NUMBER: _ClassVar[int]
    FAR_CLIP_M_FIELD_NUMBER: _ClassVar[int]
    ORTHOGRAPHIC_VERTICAL_SPAN_M_FIELD_NUMBER: _ClassVar[int]
    FOCUS_FIELD_NUMBER: _ClassVar[int]
    DOF_F_NUMBER_FIELD_NUMBER: _ClassVar[int]
    id: str
    frame: _physical_pb2.TRS
    projection: str
    focal_length_mm: float
    sensor_width_mm: float
    sensor_height_mm: float
    sensor_fit: str
    near_clip_m: float
    far_clip_m: float
    orthographic_vertical_span_m: float
    focus: FocusTarget
    dof_f_number: float
    def __init__(self, id: _Optional[str] = ..., frame: _Optional[_Union[_physical_pb2.TRS, _Mapping]] = ..., projection: _Optional[str] = ..., focal_length_mm: _Optional[float] = ..., sensor_width_mm: _Optional[float] = ..., sensor_height_mm: _Optional[float] = ..., sensor_fit: _Optional[str] = ..., near_clip_m: _Optional[float] = ..., far_clip_m: _Optional[float] = ..., orthographic_vertical_span_m: _Optional[float] = ..., focus: _Optional[_Union[FocusTarget, _Mapping]] = ..., dof_f_number: _Optional[float] = ...) -> None: ...

class SceneLightFrame(_message.Message):
    __slots__ = ("id", "frame", "local_axis_profile", "shape", "dimensions_m", "power_watts", "linear_rgb")
    ID_FIELD_NUMBER: _ClassVar[int]
    FRAME_FIELD_NUMBER: _ClassVar[int]
    LOCAL_AXIS_PROFILE_FIELD_NUMBER: _ClassVar[int]
    SHAPE_FIELD_NUMBER: _ClassVar[int]
    DIMENSIONS_M_FIELD_NUMBER: _ClassVar[int]
    POWER_WATTS_FIELD_NUMBER: _ClassVar[int]
    LINEAR_RGB_FIELD_NUMBER: _ClassVar[int]
    id: str
    frame: _physical_pb2.TRS
    local_axis_profile: str
    shape: str
    dimensions_m: _containers.RepeatedScalarFieldContainer[float]
    power_watts: float
    linear_rgb: _containers.RepeatedScalarFieldContainer[float]
    def __init__(self, id: _Optional[str] = ..., frame: _Optional[_Union[_physical_pb2.TRS, _Mapping]] = ..., local_axis_profile: _Optional[str] = ..., shape: _Optional[str] = ..., dimensions_m: _Optional[_Iterable[float]] = ..., power_watts: _Optional[float] = ..., linear_rgb: _Optional[_Iterable[float]] = ...) -> None: ...

class SceneEnvironment(_message.Message):
    __slots__ = ("model", "linear_rgb", "strength")
    MODEL_FIELD_NUMBER: _ClassVar[int]
    LINEAR_RGB_FIELD_NUMBER: _ClassVar[int]
    STRENGTH_FIELD_NUMBER: _ClassVar[int]
    model: str
    linear_rgb: _containers.RepeatedScalarFieldContainer[float]
    strength: float
    def __init__(self, model: _Optional[str] = ..., linear_rgb: _Optional[_Iterable[float]] = ..., strength: _Optional[float] = ...) -> None: ...

class SceneRegion(_message.Message):
    __slots__ = ("id", "part_id", "purpose")
    ID_FIELD_NUMBER: _ClassVar[int]
    PART_ID_FIELD_NUMBER: _ClassVar[int]
    PURPOSE_FIELD_NUMBER: _ClassVar[int]
    id: str
    part_id: str
    purpose: str
    def __init__(self, id: _Optional[str] = ..., part_id: _Optional[str] = ..., purpose: _Optional[str] = ...) -> None: ...
