import datetime

from device_studio.v1.physical import physical_pb2 as _physical_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SaveDeviceCandidateRequest(_message.Message):
    __slots__ = ("candidate", "expected_head")
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_HEAD_FIELD_NUMBER: _ClassVar[int]
    candidate: DeviceCandidate
    expected_head: int
    def __init__(self, candidate: _Optional[_Union[DeviceCandidate, _Mapping]] = ..., expected_head: _Optional[int] = ...) -> None: ...

class SaveDeviceCandidateResponse(_message.Message):
    __slots__ = ("revision",)
    REVISION_FIELD_NUMBER: _ClassVar[int]
    revision: DeviceRevision
    def __init__(self, revision: _Optional[_Union[DeviceRevision, _Mapping]] = ...) -> None: ...

class GetDeviceRevisionRequest(_message.Message):
    __slots__ = ("device_id", "revision")
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    device_id: str
    revision: int
    def __init__(self, device_id: _Optional[str] = ..., revision: _Optional[int] = ...) -> None: ...

class GetDeviceRevisionResponse(_message.Message):
    __slots__ = ("revision",)
    REVISION_FIELD_NUMBER: _ClassVar[int]
    revision: DeviceRevision
    def __init__(self, revision: _Optional[_Union[DeviceRevision, _Mapping]] = ...) -> None: ...

class GetDeviceHeadRequest(_message.Message):
    __slots__ = ("device_id",)
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    device_id: str
    def __init__(self, device_id: _Optional[str] = ...) -> None: ...

class GetDeviceHeadResponse(_message.Message):
    __slots__ = ("revision",)
    REVISION_FIELD_NUMBER: _ClassVar[int]
    revision: DeviceRevision
    def __init__(self, revision: _Optional[_Union[DeviceRevision, _Mapping]] = ...) -> None: ...

class ListDeviceRevisionsRequest(_message.Message):
    __slots__ = ("device_id", "after_revision", "page_size")
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    AFTER_REVISION_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    device_id: str
    after_revision: int
    page_size: int
    def __init__(self, device_id: _Optional[str] = ..., after_revision: _Optional[int] = ..., page_size: _Optional[int] = ...) -> None: ...

class ListDeviceRevisionsResponse(_message.Message):
    __slots__ = ("revisions", "next_after_revision")
    REVISIONS_FIELD_NUMBER: _ClassVar[int]
    NEXT_AFTER_REVISION_FIELD_NUMBER: _ClassVar[int]
    revisions: _containers.RepeatedCompositeFieldContainer[DeviceRevision]
    next_after_revision: int
    def __init__(self, revisions: _Optional[_Iterable[_Union[DeviceRevision, _Mapping]]] = ..., next_after_revision: _Optional[int] = ...) -> None: ...

class ListDeviceCandidatesRequest(_message.Message):
    __slots__ = ("after_device_id", "page_size")
    AFTER_DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    after_device_id: str
    page_size: int
    def __init__(self, after_device_id: _Optional[str] = ..., page_size: _Optional[int] = ...) -> None: ...

class ListDeviceCandidatesResponse(_message.Message):
    __slots__ = ("candidates", "next_after_device_id")
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    NEXT_AFTER_DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    candidates: _containers.RepeatedCompositeFieldContainer[DeviceRevision]
    next_after_device_id: str
    def __init__(self, candidates: _Optional[_Iterable[_Union[DeviceRevision, _Mapping]]] = ..., next_after_device_id: _Optional[str] = ...) -> None: ...

class ReviewDeviceCandidateRequest(_message.Message):
    __slots__ = ("device_id", "candidate_revision", "candidate_sha256", "expected_head", "review", "project_id")
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_REVISION_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_SHA256_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_HEAD_FIELD_NUMBER: _ClassVar[int]
    REVIEW_FIELD_NUMBER: _ClassVar[int]
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    device_id: str
    candidate_revision: int
    candidate_sha256: str
    expected_head: int
    review: QualificationReview
    project_id: str
    def __init__(self, device_id: _Optional[str] = ..., candidate_revision: _Optional[int] = ..., candidate_sha256: _Optional[str] = ..., expected_head: _Optional[int] = ..., review: _Optional[_Union[QualificationReview, _Mapping]] = ..., project_id: _Optional[str] = ...) -> None: ...

class ReviewDeviceCandidateResponse(_message.Message):
    __slots__ = ("revision", "preset_id", "preset_revision", "preset_sha256")
    REVISION_FIELD_NUMBER: _ClassVar[int]
    PRESET_ID_FIELD_NUMBER: _ClassVar[int]
    PRESET_REVISION_FIELD_NUMBER: _ClassVar[int]
    PRESET_SHA256_FIELD_NUMBER: _ClassVar[int]
    revision: DeviceRevision
    preset_id: str
    preset_revision: int
    preset_sha256: str
    def __init__(self, revision: _Optional[_Union[DeviceRevision, _Mapping]] = ..., preset_id: _Optional[str] = ..., preset_revision: _Optional[int] = ..., preset_sha256: _Optional[str] = ...) -> None: ...

class ExportQualificationPackageRequest(_message.Message):
    __slots__ = ("project_id", "device_id", "qualified_revision")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    QUALIFIED_REVISION_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    device_id: str
    qualified_revision: int
    def __init__(self, project_id: _Optional[str] = ..., device_id: _Optional[str] = ..., qualified_revision: _Optional[int] = ...) -> None: ...

class ExportQualificationPackageResponse(_message.Message):
    __slots__ = ("package_json",)
    PACKAGE_JSON_FIELD_NUMBER: _ClassVar[int]
    package_json: bytes
    def __init__(self, package_json: _Optional[bytes] = ...) -> None: ...

class ImportQualificationPackageRequest(_message.Message):
    __slots__ = ("package_json", "expected_head")
    PACKAGE_JSON_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_HEAD_FIELD_NUMBER: _ClassVar[int]
    package_json: bytes
    expected_head: int
    def __init__(self, package_json: _Optional[bytes] = ..., expected_head: _Optional[int] = ...) -> None: ...

class DeviceRevision(_message.Message):
    __slots__ = ("device_id", "revision", "sha256", "state", "candidate", "created_at")
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    device_id: str
    revision: int
    sha256: str
    state: str
    candidate: DeviceCandidate
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, device_id: _Optional[str] = ..., revision: _Optional[int] = ..., sha256: _Optional[str] = ..., state: _Optional[str] = ..., candidate: _Optional[_Union[DeviceCandidate, _Mapping]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class DeviceCandidate(_message.Message):
    __slots__ = ("version", "definition", "geometry", "geometry_unit", "coordinate_convention", "source_to_canonical", "parts", "display_bindings", "support_frames")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_UNIT_FIELD_NUMBER: _ClassVar[int]
    COORDINATE_CONVENTION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_TO_CANONICAL_FIELD_NUMBER: _ClassVar[int]
    PARTS_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_BINDINGS_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_FRAMES_FIELD_NUMBER: _ClassVar[int]
    version: str
    definition: DeviceDefinition
    geometry: _physical_pb2.ArtifactRef
    geometry_unit: str
    coordinate_convention: str
    source_to_canonical: _physical_pb2.TRS
    parts: _containers.RepeatedCompositeFieldContainer[SemanticPart]
    display_bindings: _containers.RepeatedCompositeFieldContainer[DisplayPartBinding]
    support_frames: _containers.RepeatedCompositeFieldContainer[DeviceSupportFrame]
    def __init__(self, version: _Optional[str] = ..., definition: _Optional[_Union[DeviceDefinition, _Mapping]] = ..., geometry: _Optional[_Union[_physical_pb2.ArtifactRef, _Mapping]] = ..., geometry_unit: _Optional[str] = ..., coordinate_convention: _Optional[str] = ..., source_to_canonical: _Optional[_Union[_physical_pb2.TRS, _Mapping]] = ..., parts: _Optional[_Iterable[_Union[SemanticPart, _Mapping]]] = ..., display_bindings: _Optional[_Iterable[_Union[DisplayPartBinding, _Mapping]]] = ..., support_frames: _Optional[_Iterable[_Union[DeviceSupportFrame, _Mapping]]] = ...) -> None: ...

class SemanticPart(_message.Message):
    __slots__ = ("id", "role", "parent_id", "node_index", "node_name", "local_frame")
    ID_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    PARENT_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_INDEX_FIELD_NUMBER: _ClassVar[int]
    NODE_NAME_FIELD_NUMBER: _ClassVar[int]
    LOCAL_FRAME_FIELD_NUMBER: _ClassVar[int]
    id: str
    role: str
    parent_id: str
    node_index: int
    node_name: str
    local_frame: _physical_pb2.TRS
    def __init__(self, id: _Optional[str] = ..., role: _Optional[str] = ..., parent_id: _Optional[str] = ..., node_index: _Optional[int] = ..., node_name: _Optional[str] = ..., local_frame: _Optional[_Union[_physical_pb2.TRS, _Mapping]] = ...) -> None: ...

class DisplayPartBinding(_message.Message):
    __slots__ = ("display_id", "part_id", "local_frame", "screen_primitive_index", "screen_material_index")
    DISPLAY_ID_FIELD_NUMBER: _ClassVar[int]
    PART_ID_FIELD_NUMBER: _ClassVar[int]
    LOCAL_FRAME_FIELD_NUMBER: _ClassVar[int]
    SCREEN_PRIMITIVE_INDEX_FIELD_NUMBER: _ClassVar[int]
    SCREEN_MATERIAL_INDEX_FIELD_NUMBER: _ClassVar[int]
    display_id: str
    part_id: str
    local_frame: _physical_pb2.TRS
    screen_primitive_index: int
    screen_material_index: int
    def __init__(self, display_id: _Optional[str] = ..., part_id: _Optional[str] = ..., local_frame: _Optional[_Union[_physical_pb2.TRS, _Mapping]] = ..., screen_primitive_index: _Optional[int] = ..., screen_material_index: _Optional[int] = ...) -> None: ...

class DeviceSupportFrame(_message.Message):
    __slots__ = ("id", "interface_id", "part_id", "local_frame")
    ID_FIELD_NUMBER: _ClassVar[int]
    INTERFACE_ID_FIELD_NUMBER: _ClassVar[int]
    PART_ID_FIELD_NUMBER: _ClassVar[int]
    LOCAL_FRAME_FIELD_NUMBER: _ClassVar[int]
    id: str
    interface_id: str
    part_id: str
    local_frame: _physical_pb2.TRS
    def __init__(self, id: _Optional[str] = ..., interface_id: _Optional[str] = ..., part_id: _Optional[str] = ..., local_frame: _Optional[_Union[_physical_pb2.TRS, _Mapping]] = ...) -> None: ...

class DeviceDefinition(_message.Message):
    __slots__ = ("schema_version", "id", "group", "kind", "identity_status", "manufacturer", "model", "variant", "description", "dimensions", "displays", "joints", "supported_poses", "supported_motion", "supported_profiles", "supports", "model_candidates", "provenance", "qualification", "qualification_reviews")
    class DimensionsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: Dimensions
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[Dimensions, _Mapping]] = ...) -> None: ...
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    GROUP_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_STATUS_FIELD_NUMBER: _ClassVar[int]
    MANUFACTURER_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    VARIANT_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    DIMENSIONS_FIELD_NUMBER: _ClassVar[int]
    DISPLAYS_FIELD_NUMBER: _ClassVar[int]
    JOINTS_FIELD_NUMBER: _ClassVar[int]
    SUPPORTED_POSES_FIELD_NUMBER: _ClassVar[int]
    SUPPORTED_MOTION_FIELD_NUMBER: _ClassVar[int]
    SUPPORTED_PROFILES_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_FIELD_NUMBER: _ClassVar[int]
    MODEL_CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    QUALIFICATION_FIELD_NUMBER: _ClassVar[int]
    QUALIFICATION_REVIEWS_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    id: str
    group: str
    kind: str
    identity_status: str
    manufacturer: str
    model: str
    variant: str
    description: str
    dimensions: _containers.MessageMap[str, Dimensions]
    displays: _containers.RepeatedCompositeFieldContainer[DisplaySurface]
    joints: _containers.RepeatedCompositeFieldContainer[Joint]
    supported_poses: _containers.RepeatedCompositeFieldContainer[Pose]
    supported_motion: _containers.RepeatedCompositeFieldContainer[Capability]
    supported_profiles: _containers.RepeatedCompositeFieldContainer[SupportedProfile]
    supports: _containers.RepeatedCompositeFieldContainer[SupportCapability]
    model_candidates: _containers.RepeatedCompositeFieldContainer[ModelCandidate]
    provenance: _containers.RepeatedCompositeFieldContainer[Evidence]
    qualification: str
    qualification_reviews: _containers.RepeatedCompositeFieldContainer[QualificationReview]
    def __init__(self, schema_version: _Optional[str] = ..., id: _Optional[str] = ..., group: _Optional[str] = ..., kind: _Optional[str] = ..., identity_status: _Optional[str] = ..., manufacturer: _Optional[str] = ..., model: _Optional[str] = ..., variant: _Optional[str] = ..., description: _Optional[str] = ..., dimensions: _Optional[_Mapping[str, Dimensions]] = ..., displays: _Optional[_Iterable[_Union[DisplaySurface, _Mapping]]] = ..., joints: _Optional[_Iterable[_Union[Joint, _Mapping]]] = ..., supported_poses: _Optional[_Iterable[_Union[Pose, _Mapping]]] = ..., supported_motion: _Optional[_Iterable[_Union[Capability, _Mapping]]] = ..., supported_profiles: _Optional[_Iterable[_Union[SupportedProfile, _Mapping]]] = ..., supports: _Optional[_Iterable[_Union[SupportCapability, _Mapping]]] = ..., model_candidates: _Optional[_Iterable[_Union[ModelCandidate, _Mapping]]] = ..., provenance: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ..., qualification: _Optional[str] = ..., qualification_reviews: _Optional[_Iterable[_Union[QualificationReview, _Mapping]]] = ...) -> None: ...

class Evidence(_message.Message):
    __slots__ = ("source", "kind", "retrieved_at", "qualifier")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    RETRIEVED_AT_FIELD_NUMBER: _ClassVar[int]
    QUALIFIER_FIELD_NUMBER: _ClassVar[int]
    source: str
    kind: str
    retrieved_at: str
    qualifier: str
    def __init__(self, source: _Optional[str] = ..., kind: _Optional[str] = ..., retrieved_at: _Optional[str] = ..., qualifier: _Optional[str] = ...) -> None: ...

class Uncertainty(_message.Message):
    __slots__ = ("status", "lower", "upper", "unit", "explanation")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LOWER_FIELD_NUMBER: _ClassVar[int]
    UPPER_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    EXPLANATION_FIELD_NUMBER: _ClassVar[int]
    status: str
    lower: float
    upper: float
    unit: str
    explanation: str
    def __init__(self, status: _Optional[str] = ..., lower: _Optional[float] = ..., upper: _Optional[float] = ..., unit: _Optional[str] = ..., explanation: _Optional[str] = ...) -> None: ...

class Measurement(_message.Message):
    __slots__ = ("value", "unit", "meaning", "configuration", "method", "uncertainty", "evidence")
    VALUE_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    MEANING_FIELD_NUMBER: _ClassVar[int]
    CONFIGURATION_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    UNCERTAINTY_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    value: float
    unit: str
    meaning: str
    configuration: str
    method: str
    uncertainty: Uncertainty
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    def __init__(self, value: _Optional[float] = ..., unit: _Optional[str] = ..., meaning: _Optional[str] = ..., configuration: _Optional[str] = ..., method: _Optional[str] = ..., uncertainty: _Optional[_Union[Uncertainty, _Mapping]] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ...) -> None: ...

class Dimensions(_message.Message):
    __slots__ = ("width", "height", "depth")
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    DEPTH_FIELD_NUMBER: _ClassVar[int]
    width: Measurement
    height: Measurement
    depth: Measurement
    def __init__(self, width: _Optional[_Union[Measurement, _Mapping]] = ..., height: _Optional[_Union[Measurement, _Mapping]] = ..., depth: _Optional[_Union[Measurement, _Mapping]] = ...) -> None: ...

class Rect(_message.Message):
    __slots__ = ("x", "y", "width", "height")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    x: float
    y: float
    width: float
    height: float
    def __init__(self, x: _Optional[float] = ..., y: _Optional[float] = ..., width: _Optional[float] = ..., height: _Optional[float] = ...) -> None: ...

class Region(_message.Message):
    __slots__ = ("status", "shape", "geometry_ref", "geometry_space", "bounds", "evidence")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SHAPE_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_REF_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_SPACE_FIELD_NUMBER: _ClassVar[int]
    BOUNDS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    status: str
    shape: str
    geometry_ref: str
    geometry_space: str
    bounds: Rect
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    def __init__(self, status: _Optional[str] = ..., shape: _Optional[str] = ..., geometry_ref: _Optional[str] = ..., geometry_space: _Optional[str] = ..., bounds: _Optional[_Union[Rect, _Mapping]] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ...) -> None: ...

class DisplayMetric(_message.Message):
    __slots__ = ("width_mm", "height_mm", "evidence")
    WIDTH_MM_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_MM_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    width_mm: float
    height_mm: float
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    def __init__(self, width_mm: _Optional[float] = ..., height_mm: _Optional[float] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ...) -> None: ...

class GeometryRelation(_message.Message):
    __slots__ = ("subject", "relation", "object", "status", "evidence")
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    RELATION_FIELD_NUMBER: _ClassVar[int]
    OBJECT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    subject: str
    relation: str
    object: str
    status: str
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    def __init__(self, subject: _Optional[str] = ..., relation: _Optional[str] = ..., object: _Optional[str] = ..., status: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ...) -> None: ...

class HardwareOcclusion(_message.Message):
    __slots__ = ("id", "kind", "region")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    REGION_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    region: Region
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., region: _Optional[_Union[Region, _Mapping]] = ...) -> None: ...

class DisplaySurface(_message.Message):
    __slots__ = ("id", "topology", "coordinate_metric", "geometry_relations", "physical_boundary", "active_area", "app_safe_area", "hardware_occlusions", "evidence")
    ID_FIELD_NUMBER: _ClassVar[int]
    TOPOLOGY_FIELD_NUMBER: _ClassVar[int]
    COORDINATE_METRIC_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_RELATIONS_FIELD_NUMBER: _ClassVar[int]
    PHYSICAL_BOUNDARY_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_AREA_FIELD_NUMBER: _ClassVar[int]
    APP_SAFE_AREA_FIELD_NUMBER: _ClassVar[int]
    HARDWARE_OCCLUSIONS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    topology: str
    coordinate_metric: DisplayMetric
    geometry_relations: _containers.RepeatedCompositeFieldContainer[GeometryRelation]
    physical_boundary: Region
    active_area: Region
    app_safe_area: Region
    hardware_occlusions: _containers.RepeatedCompositeFieldContainer[HardwareOcclusion]
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    def __init__(self, id: _Optional[str] = ..., topology: _Optional[str] = ..., coordinate_metric: _Optional[_Union[DisplayMetric, _Mapping]] = ..., geometry_relations: _Optional[_Iterable[_Union[GeometryRelation, _Mapping]]] = ..., physical_boundary: _Optional[_Union[Region, _Mapping]] = ..., active_area: _Optional[_Union[Region, _Mapping]] = ..., app_safe_area: _Optional[_Union[Region, _Mapping]] = ..., hardware_occlusions: _Optional[_Iterable[_Union[HardwareOcclusion, _Mapping]]] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ...) -> None: ...

class Joint(_message.Message):
    __slots__ = ("id", "parent_part", "child_part", "axis", "pivot_m", "lower_rad", "upper_rad", "status", "evidence")
    ID_FIELD_NUMBER: _ClassVar[int]
    PARENT_PART_FIELD_NUMBER: _ClassVar[int]
    CHILD_PART_FIELD_NUMBER: _ClassVar[int]
    AXIS_FIELD_NUMBER: _ClassVar[int]
    PIVOT_M_FIELD_NUMBER: _ClassVar[int]
    LOWER_RAD_FIELD_NUMBER: _ClassVar[int]
    UPPER_RAD_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    parent_part: str
    child_part: str
    axis: _physical_pb2.Vec3
    pivot_m: _physical_pb2.Vec3
    lower_rad: float
    upper_rad: float
    status: str
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    def __init__(self, id: _Optional[str] = ..., parent_part: _Optional[str] = ..., child_part: _Optional[str] = ..., axis: _Optional[_Union[_physical_pb2.Vec3, _Mapping]] = ..., pivot_m: _Optional[_Union[_physical_pb2.Vec3, _Mapping]] = ..., lower_rad: _Optional[float] = ..., upper_rad: _Optional[float] = ..., status: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ...) -> None: ...

class Capability(_message.Message):
    __slots__ = ("name", "status", "motion_kind", "joint_ids", "evidence")
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MOTION_KIND_FIELD_NUMBER: _ClassVar[int]
    JOINT_IDS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    name: str
    status: str
    motion_kind: str
    joint_ids: _containers.RepeatedScalarFieldContainer[str]
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    def __init__(self, name: _Optional[str] = ..., status: _Optional[str] = ..., motion_kind: _Optional[str] = ..., joint_ids: _Optional[_Iterable[str]] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ...) -> None: ...

class SupportedProfile(_message.Message):
    __slots__ = ("name", "status", "evidence")
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    name: str
    status: str
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    def __init__(self, name: _Optional[str] = ..., status: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ...) -> None: ...

class Pose(_message.Message):
    __slots__ = ("name", "status", "joint_state_rad", "evidence", "root_rotation_xyzw")
    class JointStateRadEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    JOINT_STATE_RAD_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    ROOT_ROTATION_XYZW_FIELD_NUMBER: _ClassVar[int]
    name: str
    status: str
    joint_state_rad: _containers.ScalarMap[str, float]
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    root_rotation_xyzw: _physical_pb2.QuatXYZW
    def __init__(self, name: _Optional[str] = ..., status: _Optional[str] = ..., joint_state_rad: _Optional[_Mapping[str, float]] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ..., root_rotation_xyzw: _Optional[_Union[_physical_pb2.QuatXYZW, _Mapping]] = ...) -> None: ...

class SupportCapability(_message.Message):
    __slots__ = ("kind", "interface_id", "status", "evidence")
    KIND_FIELD_NUMBER: _ClassVar[int]
    INTERFACE_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    kind: str
    interface_id: str
    status: str
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    def __init__(self, kind: _Optional[str] = ..., interface_id: _Optional[str] = ..., status: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ...) -> None: ...

class QualificationReview(_message.Message):
    __slots__ = ("scope", "result", "reviewer", "evidence", "candidate_revision", "candidate_sha256", "source_bundle_id", "source_bundle_revision", "source_manifest_sha256", "source_review_sha256", "source_artifact_id", "source_artifact_sha256", "source_allowed_uses", "dimension_states", "display_ids", "allowed_motion_ids", "support_ids", "preset_id", "preset_revision", "preset_sha256", "scene_id", "scene_revision", "scene_sha256", "motion_id", "motion_revision", "motion_sha256", "native_render_job_id", "native_output_sha256", "native_output_kind", "rights_claims_acknowledged", "reviewed_at")
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    REVIEWER_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_REVISION_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_SHA256_FIELD_NUMBER: _ClassVar[int]
    SOURCE_BUNDLE_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_BUNDLE_REVISION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_MANIFEST_SHA256_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REVIEW_SHA256_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ARTIFACT_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ARTIFACT_SHA256_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ALLOWED_USES_FIELD_NUMBER: _ClassVar[int]
    DIMENSION_STATES_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_IDS_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_MOTION_IDS_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_IDS_FIELD_NUMBER: _ClassVar[int]
    PRESET_ID_FIELD_NUMBER: _ClassVar[int]
    PRESET_REVISION_FIELD_NUMBER: _ClassVar[int]
    PRESET_SHA256_FIELD_NUMBER: _ClassVar[int]
    SCENE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENE_REVISION_FIELD_NUMBER: _ClassVar[int]
    SCENE_SHA256_FIELD_NUMBER: _ClassVar[int]
    MOTION_ID_FIELD_NUMBER: _ClassVar[int]
    MOTION_REVISION_FIELD_NUMBER: _ClassVar[int]
    MOTION_SHA256_FIELD_NUMBER: _ClassVar[int]
    NATIVE_RENDER_JOB_ID_FIELD_NUMBER: _ClassVar[int]
    NATIVE_OUTPUT_SHA256_FIELD_NUMBER: _ClassVar[int]
    NATIVE_OUTPUT_KIND_FIELD_NUMBER: _ClassVar[int]
    RIGHTS_CLAIMS_ACKNOWLEDGED_FIELD_NUMBER: _ClassVar[int]
    REVIEWED_AT_FIELD_NUMBER: _ClassVar[int]
    scope: str
    result: str
    reviewer: str
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    candidate_revision: int
    candidate_sha256: str
    source_bundle_id: str
    source_bundle_revision: int
    source_manifest_sha256: str
    source_review_sha256: str
    source_artifact_id: str
    source_artifact_sha256: str
    source_allowed_uses: _containers.RepeatedScalarFieldContainer[str]
    dimension_states: _containers.RepeatedScalarFieldContainer[str]
    display_ids: _containers.RepeatedScalarFieldContainer[str]
    allowed_motion_ids: _containers.RepeatedScalarFieldContainer[str]
    support_ids: _containers.RepeatedScalarFieldContainer[str]
    preset_id: str
    preset_revision: int
    preset_sha256: str
    scene_id: str
    scene_revision: int
    scene_sha256: str
    motion_id: str
    motion_revision: int
    motion_sha256: str
    native_render_job_id: str
    native_output_sha256: str
    native_output_kind: str
    rights_claims_acknowledged: bool
    reviewed_at: str
    def __init__(self, scope: _Optional[str] = ..., result: _Optional[str] = ..., reviewer: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ..., candidate_revision: _Optional[int] = ..., candidate_sha256: _Optional[str] = ..., source_bundle_id: _Optional[str] = ..., source_bundle_revision: _Optional[int] = ..., source_manifest_sha256: _Optional[str] = ..., source_review_sha256: _Optional[str] = ..., source_artifact_id: _Optional[str] = ..., source_artifact_sha256: _Optional[str] = ..., source_allowed_uses: _Optional[_Iterable[str]] = ..., dimension_states: _Optional[_Iterable[str]] = ..., display_ids: _Optional[_Iterable[str]] = ..., allowed_motion_ids: _Optional[_Iterable[str]] = ..., support_ids: _Optional[_Iterable[str]] = ..., preset_id: _Optional[str] = ..., preset_revision: _Optional[int] = ..., preset_sha256: _Optional[str] = ..., scene_id: _Optional[str] = ..., scene_revision: _Optional[int] = ..., scene_sha256: _Optional[str] = ..., motion_id: _Optional[str] = ..., motion_revision: _Optional[int] = ..., motion_sha256: _Optional[str] = ..., native_render_job_id: _Optional[str] = ..., native_output_sha256: _Optional[str] = ..., native_output_kind: _Optional[str] = ..., rights_claims_acknowledged: _Optional[bool] = ..., reviewed_at: _Optional[str] = ...) -> None: ...

class ModelCandidate(_message.Message):
    __slots__ = ("id", "status", "source_path", "sha256", "listing_url", "author", "license", "attribution", "rights_notes")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    LISTING_URL_FIELD_NUMBER: _ClassVar[int]
    AUTHOR_FIELD_NUMBER: _ClassVar[int]
    LICENSE_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    RIGHTS_NOTES_FIELD_NUMBER: _ClassVar[int]
    id: str
    status: str
    source_path: str
    sha256: str
    listing_url: str
    author: str
    license: str
    attribution: str
    rights_notes: str
    def __init__(self, id: _Optional[str] = ..., status: _Optional[str] = ..., source_path: _Optional[str] = ..., sha256: _Optional[str] = ..., listing_url: _Optional[str] = ..., author: _Optional[str] = ..., license: _Optional[str] = ..., attribution: _Optional[str] = ..., rights_notes: _Optional[str] = ...) -> None: ...
