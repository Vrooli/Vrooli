from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ArtifactRef(_message.Message):
    __slots__ = ("bundle_id", "bundle_revision", "manifest_sha256", "review_sha256", "artifact_id", "role", "size_bytes", "sha256")
    BUNDLE_ID_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_REVISION_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_SHA256_FIELD_NUMBER: _ClassVar[int]
    REVIEW_SHA256_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_ID_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    bundle_id: str
    bundle_revision: int
    manifest_sha256: str
    review_sha256: str
    artifact_id: str
    role: str
    size_bytes: int
    sha256: str
    def __init__(self, bundle_id: _Optional[str] = ..., bundle_revision: _Optional[int] = ..., manifest_sha256: _Optional[str] = ..., review_sha256: _Optional[str] = ..., artifact_id: _Optional[str] = ..., role: _Optional[str] = ..., size_bytes: _Optional[int] = ..., sha256: _Optional[str] = ...) -> None: ...

class Vec3(_message.Message):
    __slots__ = ("x", "y", "z")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    Z_FIELD_NUMBER: _ClassVar[int]
    x: float
    y: float
    z: float
    def __init__(self, x: _Optional[float] = ..., y: _Optional[float] = ..., z: _Optional[float] = ...) -> None: ...

class QuatXYZW(_message.Message):
    __slots__ = ("x", "y", "z", "w")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    Z_FIELD_NUMBER: _ClassVar[int]
    W_FIELD_NUMBER: _ClassVar[int]
    x: float
    y: float
    z: float
    w: float
    def __init__(self, x: _Optional[float] = ..., y: _Optional[float] = ..., z: _Optional[float] = ..., w: _Optional[float] = ...) -> None: ...

class TRS(_message.Message):
    __slots__ = ("translation_m", "rotation_xyzw", "scale")
    TRANSLATION_M_FIELD_NUMBER: _ClassVar[int]
    ROTATION_XYZW_FIELD_NUMBER: _ClassVar[int]
    SCALE_FIELD_NUMBER: _ClassVar[int]
    translation_m: Vec3
    rotation_xyzw: QuatXYZW
    scale: Vec3
    def __init__(self, translation_m: _Optional[_Union[Vec3, _Mapping]] = ..., rotation_xyzw: _Optional[_Union[QuatXYZW, _Mapping]] = ..., scale: _Optional[_Union[Vec3, _Mapping]] = ...) -> None: ...

class ComponentRevisionRef(_message.Message):
    __slots__ = ("id", "revision", "sha256")
    ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    id: str
    revision: int
    sha256: str
    def __init__(self, id: _Optional[str] = ..., revision: _Optional[int] = ..., sha256: _Optional[str] = ...) -> None: ...

class InitializeReviewedStarterRequest(_message.Message):
    __slots__ = ("starter_id",)
    STARTER_ID_FIELD_NUMBER: _ClassVar[int]
    starter_id: str
    def __init__(self, starter_id: _Optional[str] = ...) -> None: ...

class InitializeReviewedStarterResponse(_message.Message):
    __slots__ = ("phone_device", "phone_study_scene", "phone_still_motion", "reviewed_artifacts", "candidate_only", "phone_study_placement", "phone_study_preset", "selected_starter_id", "initialized_preset")
    PHONE_DEVICE_FIELD_NUMBER: _ClassVar[int]
    PHONE_STUDY_SCENE_FIELD_NUMBER: _ClassVar[int]
    PHONE_STILL_MOTION_FIELD_NUMBER: _ClassVar[int]
    REVIEWED_ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_ONLY_FIELD_NUMBER: _ClassVar[int]
    PHONE_STUDY_PLACEMENT_FIELD_NUMBER: _ClassVar[int]
    PHONE_STUDY_PRESET_FIELD_NUMBER: _ClassVar[int]
    SELECTED_STARTER_ID_FIELD_NUMBER: _ClassVar[int]
    INITIALIZED_PRESET_FIELD_NUMBER: _ClassVar[int]
    phone_device: ComponentRevisionRef
    phone_study_scene: ComponentRevisionRef
    phone_still_motion: ComponentRevisionRef
    reviewed_artifacts: _containers.RepeatedCompositeFieldContainer[ArtifactRef]
    candidate_only: bool
    phone_study_placement: PhysicalStarterPlacement
    phone_study_preset: ComponentRevisionRef
    selected_starter_id: str
    initialized_preset: ComponentRevisionRef
    def __init__(self, phone_device: _Optional[_Union[ComponentRevisionRef, _Mapping]] = ..., phone_study_scene: _Optional[_Union[ComponentRevisionRef, _Mapping]] = ..., phone_still_motion: _Optional[_Union[ComponentRevisionRef, _Mapping]] = ..., reviewed_artifacts: _Optional[_Iterable[_Union[ArtifactRef, _Mapping]]] = ..., candidate_only: _Optional[bool] = ..., phone_study_placement: _Optional[_Union[PhysicalStarterPlacement, _Mapping]] = ..., phone_study_preset: _Optional[_Union[ComponentRevisionRef, _Mapping]] = ..., selected_starter_id: _Optional[str] = ..., initialized_preset: _Optional[_Union[ComponentRevisionRef, _Mapping]] = ...) -> None: ...

class PhysicalStarterPlacement(_message.Message):
    __slots__ = ("profile_id", "profile_version", "display_id", "device_support_id", "scene_support_id", "selected_camera_id", "scene_support_world", "support_pivot_m", "device_root_rest", "device", "scene", "motion")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_VERSION_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_ID_FIELD_NUMBER: _ClassVar[int]
    DEVICE_SUPPORT_ID_FIELD_NUMBER: _ClassVar[int]
    SCENE_SUPPORT_ID_FIELD_NUMBER: _ClassVar[int]
    SELECTED_CAMERA_ID_FIELD_NUMBER: _ClassVar[int]
    SCENE_SUPPORT_WORLD_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_PIVOT_M_FIELD_NUMBER: _ClassVar[int]
    DEVICE_ROOT_REST_FIELD_NUMBER: _ClassVar[int]
    DEVICE_FIELD_NUMBER: _ClassVar[int]
    SCENE_FIELD_NUMBER: _ClassVar[int]
    MOTION_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    profile_version: int
    display_id: str
    device_support_id: str
    scene_support_id: str
    selected_camera_id: str
    scene_support_world: TRS
    support_pivot_m: Vec3
    device_root_rest: TRS
    device: ComponentRevisionRef
    scene: ComponentRevisionRef
    motion: ComponentRevisionRef
    def __init__(self, profile_id: _Optional[str] = ..., profile_version: _Optional[int] = ..., display_id: _Optional[str] = ..., device_support_id: _Optional[str] = ..., scene_support_id: _Optional[str] = ..., selected_camera_id: _Optional[str] = ..., scene_support_world: _Optional[_Union[TRS, _Mapping]] = ..., support_pivot_m: _Optional[_Union[Vec3, _Mapping]] = ..., device_root_rest: _Optional[_Union[TRS, _Mapping]] = ..., device: _Optional[_Union[ComponentRevisionRef, _Mapping]] = ..., scene: _Optional[_Union[ComponentRevisionRef, _Mapping]] = ..., motion: _Optional[_Union[ComponentRevisionRef, _Mapping]] = ...) -> None: ...
