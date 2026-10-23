import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from device_studio.v1.physical import physical_pb2 as _physical_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PhysicalMotionOutputProfile(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PHYSICAL_MOTION_OUTPUT_PROFILE_UNSPECIFIED: _ClassVar[PhysicalMotionOutputProfile]
    PHYSICAL_MOTION_OUTPUT_PROFILE_1080P_V1: _ClassVar[PhysicalMotionOutputProfile]
    PHYSICAL_MOTION_OUTPUT_PROFILE_1080P_V2: _ClassVar[PhysicalMotionOutputProfile]

class CoordinateConvention(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COORDINATE_CONVENTION_UNSPECIFIED: _ClassVar[CoordinateConvention]
    COORDINATE_CONVENTION_PLATE_RASTER_EDGE_PIXELS_V1: _ClassVar[CoordinateConvention]

class CaptureArea(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CAPTURE_AREA_UNSPECIFIED: _ClassVar[CaptureArea]
    CAPTURE_AREA_APP_ONLY: _ClassVar[CaptureArea]
    CAPTURE_AREA_ALREADY_COMPOSED: _ClassVar[CaptureArea]

class PresentationKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PRESENTATION_KIND_UNSPECIFIED: _ClassVar[PresentationKind]
    PRESENTATION_KIND_PLAIN: _ClassVar[PresentationKind]
    PRESENTATION_KIND_PHONE_SYSTEM: _ClassVar[PresentationKind]
    PRESENTATION_KIND_BROWSER_DESKTOP: _ClassVar[PresentationKind]

class ScreenFitMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SCREEN_FIT_MODE_UNSPECIFIED: _ClassVar[ScreenFitMode]
    SCREEN_FIT_MODE_CONTAIN: _ClassVar[ScreenFitMode]
    SCREEN_FIT_MODE_COVER: _ClassVar[ScreenFitMode]

class PresentationAppearance(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PRESENTATION_APPEARANCE_UNSPECIFIED: _ClassVar[PresentationAppearance]
    PRESENTATION_APPEARANCE_LIGHT: _ClassVar[PresentationAppearance]
    PRESENTATION_APPEARANCE_DARK: _ClassVar[PresentationAppearance]
    PRESENTATION_APPEARANCE_AUTO: _ClassVar[PresentationAppearance]

class MotionMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    MOTION_MODE_UNSPECIFIED: _ClassVar[MotionMode]
    MOTION_MODE_FIXED: _ClassVar[MotionMode]
    MOTION_MODE_ROTATION: _ClassVar[MotionMode]
    MOTION_MODE_HINGE: _ClassVar[MotionMode]
    MOTION_MODE_PARALLAX: _ClassVar[MotionMode]
    MOTION_MODE_CROP_PUSH: _ClassVar[MotionMode]

class ReadabilityVerdict(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    READABILITY_VERDICT_UNSPECIFIED: _ClassVar[ReadabilityVerdict]
    READABILITY_VERDICT_ACCEPTED: _ClassVar[ReadabilityVerdict]
    READABILITY_VERDICT_REJECTED: _ClassVar[ReadabilityVerdict]

class PhysicalCameraEndpointTargetKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PHYSICAL_CAMERA_ENDPOINT_TARGET_KIND_ACTIVE_DISPLAY: _ClassVar[PhysicalCameraEndpointTargetKind]
    PHYSICAL_CAMERA_ENDPOINT_TARGET_KIND_RECORDING_CONTENT: _ClassVar[PhysicalCameraEndpointTargetKind]

class PhysicalCompatibilityStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PHYSICAL_COMPATIBILITY_STATUS_UNSPECIFIED: _ClassVar[PhysicalCompatibilityStatus]
    PHYSICAL_COMPATIBILITY_STATUS_COMPATIBLE: _ClassVar[PhysicalCompatibilityStatus]
    PHYSICAL_COMPATIBILITY_STATUS_INCOMPATIBLE: _ClassVar[PhysicalCompatibilityStatus]
    PHYSICAL_COMPATIBILITY_STATUS_NEEDS_REVIEW: _ClassVar[PhysicalCompatibilityStatus]

class PhysicalRenderBackend(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PHYSICAL_RENDER_BACKEND_UNSPECIFIED: _ClassVar[PhysicalRenderBackend]
    PHYSICAL_RENDER_BACKEND_CPU: _ClassVar[PhysicalRenderBackend]
    PHYSICAL_RENDER_BACKEND_CUDA: _ClassVar[PhysicalRenderBackend]
    PHYSICAL_RENDER_BACKEND_OPTIX: _ClassVar[PhysicalRenderBackend]

class PhysicalStillOutputProfile(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PHYSICAL_STILL_OUTPUT_PROFILE_UNSPECIFIED: _ClassVar[PhysicalStillOutputProfile]
    PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1: _ClassVar[PhysicalStillOutputProfile]
    PHYSICAL_STILL_OUTPUT_PROFILE_3840X2160_V1: _ClassVar[PhysicalStillOutputProfile]

class VideoEOFPolicy(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VIDEO_EOF_POLICY_UNSPECIFIED: _ClassVar[VideoEOFPolicy]
    VIDEO_EOF_POLICY_REJECT: _ClassVar[VideoEOFPolicy]

class VideoAudioPolicy(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VIDEO_AUDIO_POLICY_UNSPECIFIED: _ClassVar[VideoAudioPolicy]
    VIDEO_AUDIO_POLICY_SILENT: _ClassVar[VideoAudioPolicy]
    VIDEO_AUDIO_POLICY_PASSTHROUGH: _ClassVar[VideoAudioPolicy]

class ShotIssueCode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SHOT_ISSUE_CODE_UNSPECIFIED: _ClassVar[ShotIssueCode]
    SHOT_ISSUE_CODE_DRAFT_INCOMPLETE: _ClassVar[ShotIssueCode]
    SHOT_ISSUE_CODE_MALFORMED_INPUT: _ClassVar[ShotIssueCode]
    SHOT_ISSUE_CODE_UNSAFE_GEOMETRY: _ClassVar[ShotIssueCode]
    SHOT_ISSUE_CODE_UNSUPPORTED_MOTION: _ClassVar[ShotIssueCode]
    SHOT_ISSUE_CODE_UNSUPPORTED_PRESENTATION: _ClassVar[ShotIssueCode]
    SHOT_ISSUE_CODE_UNSUPPORTED_FIT_MODE: _ClassVar[ShotIssueCode]
    SHOT_ISSUE_CODE_UNSUPPORTED_PROFILE: _ClassVar[ShotIssueCode]
    SHOT_ISSUE_CODE_UNSUPPORTED_SOURCE_MEDIA: _ClassVar[ShotIssueCode]
    SHOT_ISSUE_CODE_UNSUPPORTED_AUDIO_POLICY: _ClassVar[ShotIssueCode]
    SHOT_ISSUE_CODE_PHYSICAL_COMPATIBILITY: _ClassVar[ShotIssueCode]

class RenderOutputProfile(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RENDER_OUTPUT_PROFILE_UNSPECIFIED: _ClassVar[RenderOutputProfile]
    RENDER_OUTPUT_PROFILE_PNG_V1: _ClassVar[RenderOutputProfile]
    RENDER_OUTPUT_PROFILE_MP4_H264_SDR_V1: _ClassVar[RenderOutputProfile]
    RENDER_OUTPUT_PROFILE_PHYSICAL_PREVIEW_PNG_V1: _ClassVar[RenderOutputProfile]
    RENDER_OUTPUT_PROFILE_PHYSICAL_DIAGNOSTIC_PNG_V1: _ClassVar[RenderOutputProfile]
    RENDER_OUTPUT_PROFILE_PHYSICAL_DIAGNOSTIC_MOTION_MP4_V1: _ClassVar[RenderOutputProfile]

class RenderMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RENDER_MODE_UNSPECIFIED: _ClassVar[RenderMode]
    RENDER_MODE_STILL: _ClassVar[RenderMode]
    RENDER_MODE_VIDEO_SAMPLE_PNG: _ClassVar[RenderMode]
    RENDER_MODE_VIDEO_FINAL_MP4: _ClassVar[RenderMode]
    RENDER_MODE_PHYSICAL_STILL_PNG: _ClassVar[RenderMode]
    RENDER_MODE_PHYSICAL_VIDEO_FINAL_MP4: _ClassVar[RenderMode]
    RENDER_MODE_PHYSICAL_PREVIEW_PNG: _ClassVar[RenderMode]
    RENDER_MODE_PHYSICAL_DIAGNOSTIC_MOTION_MP4: _ClassVar[RenderMode]

class RenderJobStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RENDER_JOB_STATUS_UNSPECIFIED: _ClassVar[RenderJobStatus]
    RENDER_JOB_STATUS_QUEUED: _ClassVar[RenderJobStatus]
    RENDER_JOB_STATUS_OBSERVED: _ClassVar[RenderJobStatus]
    RENDER_JOB_STATUS_NEEDS_RECONCILIATION: _ClassVar[RenderJobStatus]
    RENDER_JOB_STATUS_CANCEL_REQUESTED: _ClassVar[RenderJobStatus]
    RENDER_JOB_STATUS_SUCCEEDED: _ClassVar[RenderJobStatus]
    RENDER_JOB_STATUS_FAILED: _ClassVar[RenderJobStatus]
    RENDER_JOB_STATUS_CANCELLED: _ClassVar[RenderJobStatus]

class RenderJobIssueCode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RENDER_JOB_ISSUE_CODE_UNSPECIFIED: _ClassVar[RenderJobIssueCode]
    RENDER_JOB_ISSUE_CODE_SHOT_NOT_RENDER_ELIGIBLE: _ClassVar[RenderJobIssueCode]
    RENDER_JOB_ISSUE_CODE_OUTPUT_UNAVAILABLE: _ClassVar[RenderJobIssueCode]
    RENDER_JOB_ISSUE_CODE_OUTCOME_UNRESOLVED: _ClassVar[RenderJobIssueCode]
    RENDER_JOB_ISSUE_CODE_CANCELLATION_UNAVAILABLE: _ClassVar[RenderJobIssueCode]
PHYSICAL_MOTION_OUTPUT_PROFILE_UNSPECIFIED: PhysicalMotionOutputProfile
PHYSICAL_MOTION_OUTPUT_PROFILE_1080P_V1: PhysicalMotionOutputProfile
PHYSICAL_MOTION_OUTPUT_PROFILE_1080P_V2: PhysicalMotionOutputProfile
COORDINATE_CONVENTION_UNSPECIFIED: CoordinateConvention
COORDINATE_CONVENTION_PLATE_RASTER_EDGE_PIXELS_V1: CoordinateConvention
CAPTURE_AREA_UNSPECIFIED: CaptureArea
CAPTURE_AREA_APP_ONLY: CaptureArea
CAPTURE_AREA_ALREADY_COMPOSED: CaptureArea
PRESENTATION_KIND_UNSPECIFIED: PresentationKind
PRESENTATION_KIND_PLAIN: PresentationKind
PRESENTATION_KIND_PHONE_SYSTEM: PresentationKind
PRESENTATION_KIND_BROWSER_DESKTOP: PresentationKind
SCREEN_FIT_MODE_UNSPECIFIED: ScreenFitMode
SCREEN_FIT_MODE_CONTAIN: ScreenFitMode
SCREEN_FIT_MODE_COVER: ScreenFitMode
PRESENTATION_APPEARANCE_UNSPECIFIED: PresentationAppearance
PRESENTATION_APPEARANCE_LIGHT: PresentationAppearance
PRESENTATION_APPEARANCE_DARK: PresentationAppearance
PRESENTATION_APPEARANCE_AUTO: PresentationAppearance
MOTION_MODE_UNSPECIFIED: MotionMode
MOTION_MODE_FIXED: MotionMode
MOTION_MODE_ROTATION: MotionMode
MOTION_MODE_HINGE: MotionMode
MOTION_MODE_PARALLAX: MotionMode
MOTION_MODE_CROP_PUSH: MotionMode
READABILITY_VERDICT_UNSPECIFIED: ReadabilityVerdict
READABILITY_VERDICT_ACCEPTED: ReadabilityVerdict
READABILITY_VERDICT_REJECTED: ReadabilityVerdict
PHYSICAL_CAMERA_ENDPOINT_TARGET_KIND_ACTIVE_DISPLAY: PhysicalCameraEndpointTargetKind
PHYSICAL_CAMERA_ENDPOINT_TARGET_KIND_RECORDING_CONTENT: PhysicalCameraEndpointTargetKind
PHYSICAL_COMPATIBILITY_STATUS_UNSPECIFIED: PhysicalCompatibilityStatus
PHYSICAL_COMPATIBILITY_STATUS_COMPATIBLE: PhysicalCompatibilityStatus
PHYSICAL_COMPATIBILITY_STATUS_INCOMPATIBLE: PhysicalCompatibilityStatus
PHYSICAL_COMPATIBILITY_STATUS_NEEDS_REVIEW: PhysicalCompatibilityStatus
PHYSICAL_RENDER_BACKEND_UNSPECIFIED: PhysicalRenderBackend
PHYSICAL_RENDER_BACKEND_CPU: PhysicalRenderBackend
PHYSICAL_RENDER_BACKEND_CUDA: PhysicalRenderBackend
PHYSICAL_RENDER_BACKEND_OPTIX: PhysicalRenderBackend
PHYSICAL_STILL_OUTPUT_PROFILE_UNSPECIFIED: PhysicalStillOutputProfile
PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1: PhysicalStillOutputProfile
PHYSICAL_STILL_OUTPUT_PROFILE_3840X2160_V1: PhysicalStillOutputProfile
VIDEO_EOF_POLICY_UNSPECIFIED: VideoEOFPolicy
VIDEO_EOF_POLICY_REJECT: VideoEOFPolicy
VIDEO_AUDIO_POLICY_UNSPECIFIED: VideoAudioPolicy
VIDEO_AUDIO_POLICY_SILENT: VideoAudioPolicy
VIDEO_AUDIO_POLICY_PASSTHROUGH: VideoAudioPolicy
SHOT_ISSUE_CODE_UNSPECIFIED: ShotIssueCode
SHOT_ISSUE_CODE_DRAFT_INCOMPLETE: ShotIssueCode
SHOT_ISSUE_CODE_MALFORMED_INPUT: ShotIssueCode
SHOT_ISSUE_CODE_UNSAFE_GEOMETRY: ShotIssueCode
SHOT_ISSUE_CODE_UNSUPPORTED_MOTION: ShotIssueCode
SHOT_ISSUE_CODE_UNSUPPORTED_PRESENTATION: ShotIssueCode
SHOT_ISSUE_CODE_UNSUPPORTED_FIT_MODE: ShotIssueCode
SHOT_ISSUE_CODE_UNSUPPORTED_PROFILE: ShotIssueCode
SHOT_ISSUE_CODE_UNSUPPORTED_SOURCE_MEDIA: ShotIssueCode
SHOT_ISSUE_CODE_UNSUPPORTED_AUDIO_POLICY: ShotIssueCode
SHOT_ISSUE_CODE_PHYSICAL_COMPATIBILITY: ShotIssueCode
RENDER_OUTPUT_PROFILE_UNSPECIFIED: RenderOutputProfile
RENDER_OUTPUT_PROFILE_PNG_V1: RenderOutputProfile
RENDER_OUTPUT_PROFILE_MP4_H264_SDR_V1: RenderOutputProfile
RENDER_OUTPUT_PROFILE_PHYSICAL_PREVIEW_PNG_V1: RenderOutputProfile
RENDER_OUTPUT_PROFILE_PHYSICAL_DIAGNOSTIC_PNG_V1: RenderOutputProfile
RENDER_OUTPUT_PROFILE_PHYSICAL_DIAGNOSTIC_MOTION_MP4_V1: RenderOutputProfile
RENDER_MODE_UNSPECIFIED: RenderMode
RENDER_MODE_STILL: RenderMode
RENDER_MODE_VIDEO_SAMPLE_PNG: RenderMode
RENDER_MODE_VIDEO_FINAL_MP4: RenderMode
RENDER_MODE_PHYSICAL_STILL_PNG: RenderMode
RENDER_MODE_PHYSICAL_VIDEO_FINAL_MP4: RenderMode
RENDER_MODE_PHYSICAL_PREVIEW_PNG: RenderMode
RENDER_MODE_PHYSICAL_DIAGNOSTIC_MOTION_MP4: RenderMode
RENDER_JOB_STATUS_UNSPECIFIED: RenderJobStatus
RENDER_JOB_STATUS_QUEUED: RenderJobStatus
RENDER_JOB_STATUS_OBSERVED: RenderJobStatus
RENDER_JOB_STATUS_NEEDS_RECONCILIATION: RenderJobStatus
RENDER_JOB_STATUS_CANCEL_REQUESTED: RenderJobStatus
RENDER_JOB_STATUS_SUCCEEDED: RenderJobStatus
RENDER_JOB_STATUS_FAILED: RenderJobStatus
RENDER_JOB_STATUS_CANCELLED: RenderJobStatus
RENDER_JOB_ISSUE_CODE_UNSPECIFIED: RenderJobIssueCode
RENDER_JOB_ISSUE_CODE_SHOT_NOT_RENDER_ELIGIBLE: RenderJobIssueCode
RENDER_JOB_ISSUE_CODE_OUTPUT_UNAVAILABLE: RenderJobIssueCode
RENDER_JOB_ISSUE_CODE_OUTCOME_UNRESOLVED: RenderJobIssueCode
RENDER_JOB_ISSUE_CODE_CANCELLATION_UNAVAILABLE: RenderJobIssueCode

class PhysicalShotSetup(_message.Message):
    __slots__ = ("device", "scene", "motion", "display_id", "device_support_id", "scene_support_id", "camera_id", "scene_support_world", "support_pivot_m", "device_root_rest", "still_profiles", "motion_profiles", "scene_support_slot_id")
    DEVICE_FIELD_NUMBER: _ClassVar[int]
    SCENE_FIELD_NUMBER: _ClassVar[int]
    MOTION_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_ID_FIELD_NUMBER: _ClassVar[int]
    DEVICE_SUPPORT_ID_FIELD_NUMBER: _ClassVar[int]
    SCENE_SUPPORT_ID_FIELD_NUMBER: _ClassVar[int]
    CAMERA_ID_FIELD_NUMBER: _ClassVar[int]
    SCENE_SUPPORT_WORLD_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_PIVOT_M_FIELD_NUMBER: _ClassVar[int]
    DEVICE_ROOT_REST_FIELD_NUMBER: _ClassVar[int]
    STILL_PROFILES_FIELD_NUMBER: _ClassVar[int]
    MOTION_PROFILES_FIELD_NUMBER: _ClassVar[int]
    SCENE_SUPPORT_SLOT_ID_FIELD_NUMBER: _ClassVar[int]
    device: ComponentRevisionRef
    scene: ComponentRevisionRef
    motion: ComponentRevisionRef
    display_id: str
    device_support_id: str
    scene_support_id: str
    camera_id: str
    scene_support_world: _physical_pb2.TRS
    support_pivot_m: _physical_pb2.Vec3
    device_root_rest: _physical_pb2.TRS
    still_profiles: _containers.RepeatedScalarFieldContainer[PhysicalStillOutputProfile]
    motion_profiles: _containers.RepeatedScalarFieldContainer[PhysicalMotionOutputProfile]
    scene_support_slot_id: str
    def __init__(self, device: _Optional[_Union[ComponentRevisionRef, _Mapping]] = ..., scene: _Optional[_Union[ComponentRevisionRef, _Mapping]] = ..., motion: _Optional[_Union[ComponentRevisionRef, _Mapping]] = ..., display_id: _Optional[str] = ..., device_support_id: _Optional[str] = ..., scene_support_id: _Optional[str] = ..., camera_id: _Optional[str] = ..., scene_support_world: _Optional[_Union[_physical_pb2.TRS, _Mapping]] = ..., support_pivot_m: _Optional[_Union[_physical_pb2.Vec3, _Mapping]] = ..., device_root_rest: _Optional[_Union[_physical_pb2.TRS, _Mapping]] = ..., still_profiles: _Optional[_Iterable[_Union[PhysicalStillOutputProfile, str]]] = ..., motion_profiles: _Optional[_Iterable[_Union[PhysicalMotionOutputProfile, str]]] = ..., scene_support_slot_id: _Optional[str] = ...) -> None: ...

class PhysicalShotPresetRevision(_message.Message):
    __slots__ = ("preset_id", "revision", "sha256", "name", "setup", "state", "candidate_only", "created_at")
    PRESET_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SETUP_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_ONLY_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    preset_id: str
    revision: int
    sha256: str
    name: str
    setup: PhysicalShotSetup
    state: str
    candidate_only: bool
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, preset_id: _Optional[str] = ..., revision: _Optional[int] = ..., sha256: _Optional[str] = ..., name: _Optional[str] = ..., setup: _Optional[_Union[PhysicalShotSetup, _Mapping]] = ..., state: _Optional[str] = ..., candidate_only: _Optional[bool] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class SavePhysicalShotPresetRequest(_message.Message):
    __slots__ = ("preset_id", "expected_revision", "name", "setup")
    PRESET_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SETUP_FIELD_NUMBER: _ClassVar[int]
    preset_id: str
    expected_revision: int
    name: str
    setup: PhysicalShotSetup
    def __init__(self, preset_id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., name: _Optional[str] = ..., setup: _Optional[_Union[PhysicalShotSetup, _Mapping]] = ...) -> None: ...

class SavePhysicalShotPresetResponse(_message.Message):
    __slots__ = ("revision",)
    REVISION_FIELD_NUMBER: _ClassVar[int]
    revision: PhysicalShotPresetRevision
    def __init__(self, revision: _Optional[_Union[PhysicalShotPresetRevision, _Mapping]] = ...) -> None: ...

class GetPhysicalShotPresetRequest(_message.Message):
    __slots__ = ("preset_id", "revision")
    PRESET_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    preset_id: str
    revision: int
    def __init__(self, preset_id: _Optional[str] = ..., revision: _Optional[int] = ...) -> None: ...

class GetPhysicalShotPresetResponse(_message.Message):
    __slots__ = ("revision",)
    REVISION_FIELD_NUMBER: _ClassVar[int]
    revision: PhysicalShotPresetRevision
    def __init__(self, revision: _Optional[_Union[PhysicalShotPresetRevision, _Mapping]] = ...) -> None: ...

class GetPhysicalShotPresetHeadRequest(_message.Message):
    __slots__ = ("preset_id",)
    PRESET_ID_FIELD_NUMBER: _ClassVar[int]
    preset_id: str
    def __init__(self, preset_id: _Optional[str] = ...) -> None: ...

class GetPhysicalShotPresetHeadResponse(_message.Message):
    __slots__ = ("revision",)
    REVISION_FIELD_NUMBER: _ClassVar[int]
    revision: PhysicalShotPresetRevision
    def __init__(self, revision: _Optional[_Union[PhysicalShotPresetRevision, _Mapping]] = ...) -> None: ...

class ListPhysicalShotPresetsRequest(_message.Message):
    __slots__ = ("after_preset_id", "page_size")
    AFTER_PRESET_ID_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    after_preset_id: str
    page_size: int
    def __init__(self, after_preset_id: _Optional[str] = ..., page_size: _Optional[int] = ...) -> None: ...

class ListPhysicalShotPresetsResponse(_message.Message):
    __slots__ = ("presets", "next_after_preset_id")
    PRESETS_FIELD_NUMBER: _ClassVar[int]
    NEXT_AFTER_PRESET_ID_FIELD_NUMBER: _ClassVar[int]
    presets: _containers.RepeatedCompositeFieldContainer[PhysicalShotPresetRevision]
    next_after_preset_id: str
    def __init__(self, presets: _Optional[_Iterable[_Union[PhysicalShotPresetRevision, _Mapping]]] = ..., next_after_preset_id: _Optional[str] = ...) -> None: ...

class ListPhysicalShotPresetRevisionsRequest(_message.Message):
    __slots__ = ("preset_id", "after_revision", "page_size")
    PRESET_ID_FIELD_NUMBER: _ClassVar[int]
    AFTER_REVISION_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    preset_id: str
    after_revision: int
    page_size: int
    def __init__(self, preset_id: _Optional[str] = ..., after_revision: _Optional[int] = ..., page_size: _Optional[int] = ...) -> None: ...

class ListPhysicalShotPresetRevisionsResponse(_message.Message):
    __slots__ = ("revisions", "next_after_revision")
    REVISIONS_FIELD_NUMBER: _ClassVar[int]
    NEXT_AFTER_REVISION_FIELD_NUMBER: _ClassVar[int]
    revisions: _containers.RepeatedCompositeFieldContainer[PhysicalShotPresetRevision]
    next_after_revision: int
    def __init__(self, revisions: _Optional[_Iterable[_Union[PhysicalShotPresetRevision, _Mapping]]] = ..., next_after_revision: _Optional[int] = ...) -> None: ...

class GetVideoSampleBindingRequest(_message.Message):
    __slots__ = ("project_id", "shot_id", "render_id")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    SHOT_ID_FIELD_NUMBER: _ClassVar[int]
    RENDER_ID_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    shot_id: str
    render_id: str
    def __init__(self, project_id: _Optional[str] = ..., shot_id: _Optional[str] = ..., render_id: _Optional[str] = ...) -> None: ...

class GetVideoSampleBindingResponse(_message.Message):
    __slots__ = ("binding",)
    BINDING_FIELD_NUMBER: _ClassVar[int]
    binding: VideoSampleBinding
    def __init__(self, binding: _Optional[_Union[VideoSampleBinding, _Mapping]] = ...) -> None: ...

class Shot(_message.Message):
    __slots__ = ("id", "project_id", "name", "created_at", "updated_at", "head_revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    HEAD_REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    project_id: str
    name: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    head_revision: int
    def __init__(self, id: _Optional[str] = ..., project_id: _Optional[str] = ..., name: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., head_revision: _Optional[int] = ...) -> None: ...

class ShotPoint(_message.Message):
    __slots__ = ("x", "y")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    x: float
    y: float
    def __init__(self, x: _Optional[float] = ..., y: _Optional[float] = ...) -> None: ...

class PlateQuad(_message.Message):
    __slots__ = ("top_left", "top_right", "bottom_right", "bottom_left")
    TOP_LEFT_FIELD_NUMBER: _ClassVar[int]
    TOP_RIGHT_FIELD_NUMBER: _ClassVar[int]
    BOTTOM_RIGHT_FIELD_NUMBER: _ClassVar[int]
    BOTTOM_LEFT_FIELD_NUMBER: _ClassVar[int]
    top_left: ShotPoint
    top_right: ShotPoint
    bottom_right: ShotPoint
    bottom_left: ShotPoint
    def __init__(self, top_left: _Optional[_Union[ShotPoint, _Mapping]] = ..., top_right: _Optional[_Union[ShotPoint, _Mapping]] = ..., bottom_right: _Optional[_Union[ShotPoint, _Mapping]] = ..., bottom_left: _Optional[_Union[ShotPoint, _Mapping]] = ...) -> None: ...

class PhotographicScene(_message.Message):
    __slots__ = ("plate_media_id", "active_mask_media_id", "hardware_mask_media_id", "display_width_pixels", "display_height_pixels", "plate_quad", "coordinate_convention", "depiction_label")
    PLATE_MEDIA_ID_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_MASK_MEDIA_ID_FIELD_NUMBER: _ClassVar[int]
    HARDWARE_MASK_MEDIA_ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_WIDTH_PIXELS_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_HEIGHT_PIXELS_FIELD_NUMBER: _ClassVar[int]
    PLATE_QUAD_FIELD_NUMBER: _ClassVar[int]
    COORDINATE_CONVENTION_FIELD_NUMBER: _ClassVar[int]
    DEPICTION_LABEL_FIELD_NUMBER: _ClassVar[int]
    plate_media_id: str
    active_mask_media_id: str
    hardware_mask_media_id: str
    display_width_pixels: int
    display_height_pixels: int
    plate_quad: PlateQuad
    coordinate_convention: CoordinateConvention
    depiction_label: str
    def __init__(self, plate_media_id: _Optional[str] = ..., active_mask_media_id: _Optional[str] = ..., hardware_mask_media_id: _Optional[str] = ..., display_width_pixels: _Optional[int] = ..., display_height_pixels: _Optional[int] = ..., plate_quad: _Optional[_Union[PlateQuad, _Mapping]] = ..., coordinate_convention: _Optional[_Union[CoordinateConvention, str]] = ..., depiction_label: _Optional[str] = ...) -> None: ...

class PresentationRGB(_message.Message):
    __slots__ = ("red", "green", "blue")
    RED_FIELD_NUMBER: _ClassVar[int]
    GREEN_FIELD_NUMBER: _ClassVar[int]
    BLUE_FIELD_NUMBER: _ClassVar[int]
    red: int
    green: int
    blue: int
    def __init__(self, red: _Optional[int] = ..., green: _Optional[int] = ..., blue: _Optional[int] = ...) -> None: ...

class PresentationRect(_message.Message):
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

class PhoneSystemChrome(_message.Message):
    __slots__ = ("carrier_text", "clock_text", "battery_percent", "show_cellular_indicator", "show_wifi_indicator", "show_home_indicator", "show_status", "show_battery_percent", "appearance", "top_fill", "bottom_fill", "status_ink", "home_ink")
    CARRIER_TEXT_FIELD_NUMBER: _ClassVar[int]
    CLOCK_TEXT_FIELD_NUMBER: _ClassVar[int]
    BATTERY_PERCENT_FIELD_NUMBER: _ClassVar[int]
    SHOW_CELLULAR_INDICATOR_FIELD_NUMBER: _ClassVar[int]
    SHOW_WIFI_INDICATOR_FIELD_NUMBER: _ClassVar[int]
    SHOW_HOME_INDICATOR_FIELD_NUMBER: _ClassVar[int]
    SHOW_STATUS_FIELD_NUMBER: _ClassVar[int]
    SHOW_BATTERY_PERCENT_FIELD_NUMBER: _ClassVar[int]
    APPEARANCE_FIELD_NUMBER: _ClassVar[int]
    TOP_FILL_FIELD_NUMBER: _ClassVar[int]
    BOTTOM_FILL_FIELD_NUMBER: _ClassVar[int]
    STATUS_INK_FIELD_NUMBER: _ClassVar[int]
    HOME_INK_FIELD_NUMBER: _ClassVar[int]
    carrier_text: str
    clock_text: str
    battery_percent: int
    show_cellular_indicator: bool
    show_wifi_indicator: bool
    show_home_indicator: bool
    show_status: bool
    show_battery_percent: bool
    appearance: PresentationAppearance
    top_fill: PresentationRGB
    bottom_fill: PresentationRGB
    status_ink: PresentationRGB
    home_ink: PresentationRGB
    def __init__(self, carrier_text: _Optional[str] = ..., clock_text: _Optional[str] = ..., battery_percent: _Optional[int] = ..., show_cellular_indicator: _Optional[bool] = ..., show_wifi_indicator: _Optional[bool] = ..., show_home_indicator: _Optional[bool] = ..., show_status: _Optional[bool] = ..., show_battery_percent: _Optional[bool] = ..., appearance: _Optional[_Union[PresentationAppearance, str]] = ..., top_fill: _Optional[_Union[PresentationRGB, _Mapping]] = ..., bottom_fill: _Optional[_Union[PresentationRGB, _Mapping]] = ..., status_ink: _Optional[_Union[PresentationRGB, _Mapping]] = ..., home_ink: _Optional[_Union[PresentationRGB, _Mapping]] = ...) -> None: ...

class BrowserDesktopChrome(_message.Message):
    __slots__ = ("window_title", "address_text", "show_window_controls", "show_toolbar", "appearance", "window", "show_menu_bar", "show_dock", "desktop_color", "window_color", "chrome_color")
    WINDOW_TITLE_FIELD_NUMBER: _ClassVar[int]
    ADDRESS_TEXT_FIELD_NUMBER: _ClassVar[int]
    SHOW_WINDOW_CONTROLS_FIELD_NUMBER: _ClassVar[int]
    SHOW_TOOLBAR_FIELD_NUMBER: _ClassVar[int]
    APPEARANCE_FIELD_NUMBER: _ClassVar[int]
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    SHOW_MENU_BAR_FIELD_NUMBER: _ClassVar[int]
    SHOW_DOCK_FIELD_NUMBER: _ClassVar[int]
    DESKTOP_COLOR_FIELD_NUMBER: _ClassVar[int]
    WINDOW_COLOR_FIELD_NUMBER: _ClassVar[int]
    CHROME_COLOR_FIELD_NUMBER: _ClassVar[int]
    window_title: str
    address_text: str
    show_window_controls: bool
    show_toolbar: bool
    appearance: PresentationAppearance
    window: PresentationRect
    show_menu_bar: bool
    show_dock: bool
    desktop_color: PresentationRGB
    window_color: PresentationRGB
    chrome_color: PresentationRGB
    def __init__(self, window_title: _Optional[str] = ..., address_text: _Optional[str] = ..., show_window_controls: _Optional[bool] = ..., show_toolbar: _Optional[bool] = ..., appearance: _Optional[_Union[PresentationAppearance, str]] = ..., window: _Optional[_Union[PresentationRect, _Mapping]] = ..., show_menu_bar: _Optional[bool] = ..., show_dock: _Optional[bool] = ..., desktop_color: _Optional[_Union[PresentationRGB, _Mapping]] = ..., window_color: _Optional[_Union[PresentationRGB, _Mapping]] = ..., chrome_color: _Optional[_Union[PresentationRGB, _Mapping]] = ...) -> None: ...

class ScreenPresentation(_message.Message):
    __slots__ = ("source_media_id", "capture_area", "kind", "fit_mode", "viewport_width_pixels", "viewport_height_pixels", "phone_system_chrome", "browser_desktop_chrome", "matte")
    SOURCE_MEDIA_ID_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_AREA_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    FIT_MODE_FIELD_NUMBER: _ClassVar[int]
    VIEWPORT_WIDTH_PIXELS_FIELD_NUMBER: _ClassVar[int]
    VIEWPORT_HEIGHT_PIXELS_FIELD_NUMBER: _ClassVar[int]
    PHONE_SYSTEM_CHROME_FIELD_NUMBER: _ClassVar[int]
    BROWSER_DESKTOP_CHROME_FIELD_NUMBER: _ClassVar[int]
    MATTE_FIELD_NUMBER: _ClassVar[int]
    source_media_id: str
    capture_area: CaptureArea
    kind: PresentationKind
    fit_mode: ScreenFitMode
    viewport_width_pixels: int
    viewport_height_pixels: int
    phone_system_chrome: PhoneSystemChrome
    browser_desktop_chrome: BrowserDesktopChrome
    matte: PresentationRGB
    def __init__(self, source_media_id: _Optional[str] = ..., capture_area: _Optional[_Union[CaptureArea, str]] = ..., kind: _Optional[_Union[PresentationKind, str]] = ..., fit_mode: _Optional[_Union[ScreenFitMode, str]] = ..., viewport_width_pixels: _Optional[int] = ..., viewport_height_pixels: _Optional[int] = ..., phone_system_chrome: _Optional[_Union[PhoneSystemChrome, _Mapping]] = ..., browser_desktop_chrome: _Optional[_Union[BrowserDesktopChrome, _Mapping]] = ..., matte: _Optional[_Union[PresentationRGB, _Mapping]] = ...) -> None: ...

class ShotSnapshot(_message.Message):
    __slots__ = ("scene", "presentation", "motion_mode", "video_timing", "camera_motion", "physical_binding", "readability_proof")
    SCENE_FIELD_NUMBER: _ClassVar[int]
    PRESENTATION_FIELD_NUMBER: _ClassVar[int]
    MOTION_MODE_FIELD_NUMBER: _ClassVar[int]
    VIDEO_TIMING_FIELD_NUMBER: _ClassVar[int]
    CAMERA_MOTION_FIELD_NUMBER: _ClassVar[int]
    PHYSICAL_BINDING_FIELD_NUMBER: _ClassVar[int]
    READABILITY_PROOF_FIELD_NUMBER: _ClassVar[int]
    scene: PhotographicScene
    presentation: ScreenPresentation
    motion_mode: MotionMode
    video_timing: VideoTiming
    camera_motion: PhotographicCameraMotion
    physical_binding: PhysicalShotBinding
    readability_proof: ReadabilityProof
    def __init__(self, scene: _Optional[_Union[PhotographicScene, _Mapping]] = ..., presentation: _Optional[_Union[ScreenPresentation, _Mapping]] = ..., motion_mode: _Optional[_Union[MotionMode, str]] = ..., video_timing: _Optional[_Union[VideoTiming, _Mapping]] = ..., camera_motion: _Optional[_Union[PhotographicCameraMotion, _Mapping]] = ..., physical_binding: _Optional[_Union[PhysicalShotBinding, _Mapping]] = ..., readability_proof: _Optional[_Union[ReadabilityProof, _Mapping]] = ...) -> None: ...

class ReadabilityProof(_message.Message):
    __slots__ = ("region_name", "required_content", "criteria", "region_x", "region_y", "region_width", "region_height", "minimum_output_width_pixels", "minimum_output_height_pixels", "minimum_region_width_pixels", "minimum_region_height_pixels", "proof_start_ms", "proof_end_ms")
    REGION_NAME_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_CONTENT_FIELD_NUMBER: _ClassVar[int]
    CRITERIA_FIELD_NUMBER: _ClassVar[int]
    REGION_X_FIELD_NUMBER: _ClassVar[int]
    REGION_Y_FIELD_NUMBER: _ClassVar[int]
    REGION_WIDTH_FIELD_NUMBER: _ClassVar[int]
    REGION_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_OUTPUT_WIDTH_PIXELS_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_OUTPUT_HEIGHT_PIXELS_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_REGION_WIDTH_PIXELS_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_REGION_HEIGHT_PIXELS_FIELD_NUMBER: _ClassVar[int]
    PROOF_START_MS_FIELD_NUMBER: _ClassVar[int]
    PROOF_END_MS_FIELD_NUMBER: _ClassVar[int]
    region_name: str
    required_content: str
    criteria: str
    region_x: float
    region_y: float
    region_width: float
    region_height: float
    minimum_output_width_pixels: int
    minimum_output_height_pixels: int
    minimum_region_width_pixels: int
    minimum_region_height_pixels: int
    proof_start_ms: int
    proof_end_ms: int
    def __init__(self, region_name: _Optional[str] = ..., required_content: _Optional[str] = ..., criteria: _Optional[str] = ..., region_x: _Optional[float] = ..., region_y: _Optional[float] = ..., region_width: _Optional[float] = ..., region_height: _Optional[float] = ..., minimum_output_width_pixels: _Optional[int] = ..., minimum_output_height_pixels: _Optional[int] = ..., minimum_region_width_pixels: _Optional[int] = ..., minimum_region_height_pixels: _Optional[int] = ..., proof_start_ms: _Optional[int] = ..., proof_end_ms: _Optional[int] = ...) -> None: ...

class OutputReadabilityReview(_message.Message):
    __slots__ = ("id", "project_id", "shot_id", "shot_revision", "render_id", "output_sha256", "proof_sha256", "verdict", "note", "output_width_pixels", "output_height_pixels", "region_width_pixels", "region_height_pixels", "reviewed_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    SHOT_ID_FIELD_NUMBER: _ClassVar[int]
    SHOT_REVISION_FIELD_NUMBER: _ClassVar[int]
    RENDER_ID_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_SHA256_FIELD_NUMBER: _ClassVar[int]
    PROOF_SHA256_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_WIDTH_PIXELS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_HEIGHT_PIXELS_FIELD_NUMBER: _ClassVar[int]
    REGION_WIDTH_PIXELS_FIELD_NUMBER: _ClassVar[int]
    REGION_HEIGHT_PIXELS_FIELD_NUMBER: _ClassVar[int]
    REVIEWED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    project_id: str
    shot_id: str
    shot_revision: int
    render_id: str
    output_sha256: str
    proof_sha256: str
    verdict: ReadabilityVerdict
    note: str
    output_width_pixels: int
    output_height_pixels: int
    region_width_pixels: int
    region_height_pixels: int
    reviewed_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., project_id: _Optional[str] = ..., shot_id: _Optional[str] = ..., shot_revision: _Optional[int] = ..., render_id: _Optional[str] = ..., output_sha256: _Optional[str] = ..., proof_sha256: _Optional[str] = ..., verdict: _Optional[_Union[ReadabilityVerdict, str]] = ..., note: _Optional[str] = ..., output_width_pixels: _Optional[int] = ..., output_height_pixels: _Optional[int] = ..., region_width_pixels: _Optional[int] = ..., region_height_pixels: _Optional[int] = ..., reviewed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ReviewOutputReadabilityRequest(_message.Message):
    __slots__ = ("project_id", "shot_id", "shot_revision", "render_id", "output_sha256", "verdict", "note", "explicitly_inspected_final_size")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    SHOT_ID_FIELD_NUMBER: _ClassVar[int]
    SHOT_REVISION_FIELD_NUMBER: _ClassVar[int]
    RENDER_ID_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_SHA256_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    EXPLICITLY_INSPECTED_FINAL_SIZE_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    shot_id: str
    shot_revision: int
    render_id: str
    output_sha256: str
    verdict: ReadabilityVerdict
    note: str
    explicitly_inspected_final_size: bool
    def __init__(self, project_id: _Optional[str] = ..., shot_id: _Optional[str] = ..., shot_revision: _Optional[int] = ..., render_id: _Optional[str] = ..., output_sha256: _Optional[str] = ..., verdict: _Optional[_Union[ReadabilityVerdict, str]] = ..., note: _Optional[str] = ..., explicitly_inspected_final_size: _Optional[bool] = ...) -> None: ...

class ReviewOutputReadabilityResponse(_message.Message):
    __slots__ = ("review",)
    REVIEW_FIELD_NUMBER: _ClassVar[int]
    review: OutputReadabilityReview
    def __init__(self, review: _Optional[_Union[OutputReadabilityReview, _Mapping]] = ...) -> None: ...

class GetOutputReadabilityReviewRequest(_message.Message):
    __slots__ = ("project_id", "shot_id", "shot_revision", "render_id", "output_sha256")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    SHOT_ID_FIELD_NUMBER: _ClassVar[int]
    SHOT_REVISION_FIELD_NUMBER: _ClassVar[int]
    RENDER_ID_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_SHA256_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    shot_id: str
    shot_revision: int
    render_id: str
    output_sha256: str
    def __init__(self, project_id: _Optional[str] = ..., shot_id: _Optional[str] = ..., shot_revision: _Optional[int] = ..., render_id: _Optional[str] = ..., output_sha256: _Optional[str] = ...) -> None: ...

class GetOutputReadabilityReviewResponse(_message.Message):
    __slots__ = ("review",)
    REVIEW_FIELD_NUMBER: _ClassVar[int]
    review: OutputReadabilityReview
    def __init__(self, review: _Optional[_Union[OutputReadabilityReview, _Mapping]] = ...) -> None: ...

class ComponentRevisionRef(_message.Message):
    __slots__ = ("id", "revision", "sha256")
    ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    id: str
    revision: int
    sha256: str
    def __init__(self, id: _Optional[str] = ..., revision: _Optional[int] = ..., sha256: _Optional[str] = ...) -> None: ...

class PhysicalShotBinding(_message.Message):
    __slots__ = ("device", "scene", "motion", "display_id", "device_support_id", "scene_support_id", "camera_id", "scene_support_world", "support_pivot_m", "device_root_rest", "screen_width_pixels", "screen_height_pixels", "renderer_profile_id", "renderer_profile_version", "output_profile", "motion_sample_tick", "backend", "motion_output", "scene_support_slot_id", "camera_endpoint")
    DEVICE_FIELD_NUMBER: _ClassVar[int]
    SCENE_FIELD_NUMBER: _ClassVar[int]
    MOTION_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_ID_FIELD_NUMBER: _ClassVar[int]
    DEVICE_SUPPORT_ID_FIELD_NUMBER: _ClassVar[int]
    SCENE_SUPPORT_ID_FIELD_NUMBER: _ClassVar[int]
    CAMERA_ID_FIELD_NUMBER: _ClassVar[int]
    SCENE_SUPPORT_WORLD_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_PIVOT_M_FIELD_NUMBER: _ClassVar[int]
    DEVICE_ROOT_REST_FIELD_NUMBER: _ClassVar[int]
    SCREEN_WIDTH_PIXELS_FIELD_NUMBER: _ClassVar[int]
    SCREEN_HEIGHT_PIXELS_FIELD_NUMBER: _ClassVar[int]
    RENDERER_PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    RENDERER_PROFILE_VERSION_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_PROFILE_FIELD_NUMBER: _ClassVar[int]
    MOTION_SAMPLE_TICK_FIELD_NUMBER: _ClassVar[int]
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    MOTION_OUTPUT_FIELD_NUMBER: _ClassVar[int]
    SCENE_SUPPORT_SLOT_ID_FIELD_NUMBER: _ClassVar[int]
    CAMERA_ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    device: ComponentRevisionRef
    scene: ComponentRevisionRef
    motion: ComponentRevisionRef
    display_id: str
    device_support_id: str
    scene_support_id: str
    camera_id: str
    scene_support_world: _physical_pb2.TRS
    support_pivot_m: _physical_pb2.Vec3
    device_root_rest: _physical_pb2.TRS
    screen_width_pixels: int
    screen_height_pixels: int
    renderer_profile_id: str
    renderer_profile_version: int
    output_profile: PhysicalStillOutputProfile
    motion_sample_tick: RationalTime
    backend: PhysicalRenderBackend
    motion_output: PhysicalMotionOutputBinding
    scene_support_slot_id: str
    camera_endpoint: PhysicalCameraEndpoint
    def __init__(self, device: _Optional[_Union[ComponentRevisionRef, _Mapping]] = ..., scene: _Optional[_Union[ComponentRevisionRef, _Mapping]] = ..., motion: _Optional[_Union[ComponentRevisionRef, _Mapping]] = ..., display_id: _Optional[str] = ..., device_support_id: _Optional[str] = ..., scene_support_id: _Optional[str] = ..., camera_id: _Optional[str] = ..., scene_support_world: _Optional[_Union[_physical_pb2.TRS, _Mapping]] = ..., support_pivot_m: _Optional[_Union[_physical_pb2.Vec3, _Mapping]] = ..., device_root_rest: _Optional[_Union[_physical_pb2.TRS, _Mapping]] = ..., screen_width_pixels: _Optional[int] = ..., screen_height_pixels: _Optional[int] = ..., renderer_profile_id: _Optional[str] = ..., renderer_profile_version: _Optional[int] = ..., output_profile: _Optional[_Union[PhysicalStillOutputProfile, str]] = ..., motion_sample_tick: _Optional[_Union[RationalTime, _Mapping]] = ..., backend: _Optional[_Union[PhysicalRenderBackend, str]] = ..., motion_output: _Optional[_Union[PhysicalMotionOutputBinding, _Mapping]] = ..., scene_support_slot_id: _Optional[str] = ..., camera_endpoint: _Optional[_Union[PhysicalCameraEndpoint, _Mapping]] = ...) -> None: ...

class PhysicalCameraEndpoint(_message.Message):
    __slots__ = ("display_id", "raster_width", "raster_height", "x", "y", "width", "height", "max_error_pixels", "target_kind", "content_width_pixels", "content_height_pixels", "arrival_motion_tick")
    DISPLAY_ID_FIELD_NUMBER: _ClassVar[int]
    RASTER_WIDTH_FIELD_NUMBER: _ClassVar[int]
    RASTER_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    MAX_ERROR_PIXELS_FIELD_NUMBER: _ClassVar[int]
    TARGET_KIND_FIELD_NUMBER: _ClassVar[int]
    CONTENT_WIDTH_PIXELS_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HEIGHT_PIXELS_FIELD_NUMBER: _ClassVar[int]
    ARRIVAL_MOTION_TICK_FIELD_NUMBER: _ClassVar[int]
    display_id: str
    raster_width: int
    raster_height: int
    x: float
    y: float
    width: float
    height: float
    max_error_pixels: float
    target_kind: PhysicalCameraEndpointTargetKind
    content_width_pixels: int
    content_height_pixels: int
    arrival_motion_tick: int
    def __init__(self, display_id: _Optional[str] = ..., raster_width: _Optional[int] = ..., raster_height: _Optional[int] = ..., x: _Optional[float] = ..., y: _Optional[float] = ..., width: _Optional[float] = ..., height: _Optional[float] = ..., max_error_pixels: _Optional[float] = ..., target_kind: _Optional[_Union[PhysicalCameraEndpointTargetKind, str]] = ..., content_width_pixels: _Optional[int] = ..., content_height_pixels: _Optional[int] = ..., arrival_motion_tick: _Optional[int] = ...) -> None: ...

class PhysicalCompatibilityReport(_message.Message):
    __slots__ = ("status", "reasons")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REASONS_FIELD_NUMBER: _ClassVar[int]
    status: PhysicalCompatibilityStatus
    reasons: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, status: _Optional[_Union[PhysicalCompatibilityStatus, str]] = ..., reasons: _Optional[_Iterable[str]] = ...) -> None: ...

class PhysicalMotionOutputBinding(_message.Message):
    __slots__ = ("profile_id", "profile_version", "width", "height", "output_rate_numerator", "output_rate_denominator", "frame_count")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_VERSION_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_RATE_NUMERATOR_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_RATE_DENOMINATOR_FIELD_NUMBER: _ClassVar[int]
    FRAME_COUNT_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    profile_version: int
    width: int
    height: int
    output_rate_numerator: int
    output_rate_denominator: int
    frame_count: int
    def __init__(self, profile_id: _Optional[str] = ..., profile_version: _Optional[int] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., output_rate_numerator: _Optional[int] = ..., output_rate_denominator: _Optional[int] = ..., frame_count: _Optional[int] = ...) -> None: ...

class RationalTime(_message.Message):
    __slots__ = ("numerator", "denominator")
    NUMERATOR_FIELD_NUMBER: _ClassVar[int]
    DENOMINATOR_FIELD_NUMBER: _ClassVar[int]
    numerator: int
    denominator: int
    def __init__(self, numerator: _Optional[int] = ..., denominator: _Optional[int] = ...) -> None: ...

class PhotographicCameraMotion(_message.Message):
    __slots__ = ("profile_id", "profile_version", "duration_output_frames", "matte_rgb", "target_rect", "scene_rect")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_VERSION_FIELD_NUMBER: _ClassVar[int]
    DURATION_OUTPUT_FRAMES_FIELD_NUMBER: _ClassVar[int]
    MATTE_RGB_FIELD_NUMBER: _ClassVar[int]
    TARGET_RECT_FIELD_NUMBER: _ClassVar[int]
    SCENE_RECT_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    profile_version: int
    duration_output_frames: int
    matte_rgb: PresentationRGB
    target_rect: PresentationRect
    scene_rect: PresentationRect
    def __init__(self, profile_id: _Optional[str] = ..., profile_version: _Optional[int] = ..., duration_output_frames: _Optional[int] = ..., matte_rgb: _Optional[_Union[PresentationRGB, _Mapping]] = ..., target_rect: _Optional[_Union[PresentationRect, _Mapping]] = ..., scene_rect: _Optional[_Union[PresentationRect, _Mapping]] = ...) -> None: ...

class VideoTiming(_message.Message):
    __slots__ = ("source_media_id", "source_sha256", "source_size_bytes", "inspection_job_id", "inspection_attempt_id", "inspection_receipt_id", "inspection_receipt_sha256", "clock_artifact_id", "clock_sha256", "clock_size_bytes", "selected_video_stream", "time_base_num", "time_base_den", "stream_start_pts", "source_end_pts", "last_frame_duration_ticks", "trim_start_pts", "trim_end_pts", "playback_rate_num", "playback_rate_den", "output_rate_num", "output_rate_den", "eof_policy", "video_composition_profile_id", "video_composition_profile_version", "sample_binding", "sample_base_recipe_version", "audio_policy")
    SOURCE_MEDIA_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SHA256_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    INSPECTION_JOB_ID_FIELD_NUMBER: _ClassVar[int]
    INSPECTION_ATTEMPT_ID_FIELD_NUMBER: _ClassVar[int]
    INSPECTION_RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    INSPECTION_RECEIPT_SHA256_FIELD_NUMBER: _ClassVar[int]
    CLOCK_ARTIFACT_ID_FIELD_NUMBER: _ClassVar[int]
    CLOCK_SHA256_FIELD_NUMBER: _ClassVar[int]
    CLOCK_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    SELECTED_VIDEO_STREAM_FIELD_NUMBER: _ClassVar[int]
    TIME_BASE_NUM_FIELD_NUMBER: _ClassVar[int]
    TIME_BASE_DEN_FIELD_NUMBER: _ClassVar[int]
    STREAM_START_PTS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_END_PTS_FIELD_NUMBER: _ClassVar[int]
    LAST_FRAME_DURATION_TICKS_FIELD_NUMBER: _ClassVar[int]
    TRIM_START_PTS_FIELD_NUMBER: _ClassVar[int]
    TRIM_END_PTS_FIELD_NUMBER: _ClassVar[int]
    PLAYBACK_RATE_NUM_FIELD_NUMBER: _ClassVar[int]
    PLAYBACK_RATE_DEN_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_RATE_NUM_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_RATE_DEN_FIELD_NUMBER: _ClassVar[int]
    EOF_POLICY_FIELD_NUMBER: _ClassVar[int]
    VIDEO_COMPOSITION_PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    VIDEO_COMPOSITION_PROFILE_VERSION_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_BINDING_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_BASE_RECIPE_VERSION_FIELD_NUMBER: _ClassVar[int]
    AUDIO_POLICY_FIELD_NUMBER: _ClassVar[int]
    source_media_id: str
    source_sha256: str
    source_size_bytes: int
    inspection_job_id: str
    inspection_attempt_id: str
    inspection_receipt_id: str
    inspection_receipt_sha256: str
    clock_artifact_id: str
    clock_sha256: str
    clock_size_bytes: int
    selected_video_stream: int
    time_base_num: int
    time_base_den: int
    stream_start_pts: int
    source_end_pts: int
    last_frame_duration_ticks: int
    trim_start_pts: int
    trim_end_pts: int
    playback_rate_num: int
    playback_rate_den: int
    output_rate_num: int
    output_rate_den: int
    eof_policy: VideoEOFPolicy
    video_composition_profile_id: str
    video_composition_profile_version: int
    sample_binding: VideoSampleBinding
    sample_base_recipe_version: int
    audio_policy: VideoAudioPolicy
    def __init__(self, source_media_id: _Optional[str] = ..., source_sha256: _Optional[str] = ..., source_size_bytes: _Optional[int] = ..., inspection_job_id: _Optional[str] = ..., inspection_attempt_id: _Optional[str] = ..., inspection_receipt_id: _Optional[str] = ..., inspection_receipt_sha256: _Optional[str] = ..., clock_artifact_id: _Optional[str] = ..., clock_sha256: _Optional[str] = ..., clock_size_bytes: _Optional[int] = ..., selected_video_stream: _Optional[int] = ..., time_base_num: _Optional[int] = ..., time_base_den: _Optional[int] = ..., stream_start_pts: _Optional[int] = ..., source_end_pts: _Optional[int] = ..., last_frame_duration_ticks: _Optional[int] = ..., trim_start_pts: _Optional[int] = ..., trim_end_pts: _Optional[int] = ..., playback_rate_num: _Optional[int] = ..., playback_rate_den: _Optional[int] = ..., output_rate_num: _Optional[int] = ..., output_rate_den: _Optional[int] = ..., eof_policy: _Optional[_Union[VideoEOFPolicy, str]] = ..., video_composition_profile_id: _Optional[str] = ..., video_composition_profile_version: _Optional[int] = ..., sample_binding: _Optional[_Union[VideoSampleBinding, _Mapping]] = ..., sample_base_recipe_version: _Optional[int] = ..., audio_policy: _Optional[_Union[VideoAudioPolicy, str]] = ...) -> None: ...

class VideoSampleBinding(_message.Message):
    __slots__ = ("job_id", "attempt_id", "receipt_id", "receipt_sha256", "source_sha256", "source_size_bytes", "clock_sha256", "trim_start_pts", "trim_end_pts", "video_composition_profile_id", "video_composition_profile_version", "source_ordinal", "source_pts", "projected_png_sha256", "projected_png_size_bytes", "projected_png_width", "projected_png_height", "composed_frame_sha256", "composed_frame_width", "composed_frame_height", "pixel_profile", "base_recipe_sha256", "resolved_top_fill_red", "resolved_top_fill_green", "resolved_top_fill_blue", "resolved_bottom_fill_red", "resolved_bottom_fill_green", "resolved_bottom_fill_blue", "resolved_status_ink_red", "resolved_status_ink_green", "resolved_status_ink_blue", "resolved_home_ink_red", "resolved_home_ink_green", "resolved_home_ink_blue", "base_shot_revision", "base_recipe_version", "top_fill_source_resolved", "bottom_fill_source_resolved", "status_ink_source_resolved", "home_ink_source_resolved")
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    ATTEMPT_ID_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_SHA256_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SHA256_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    CLOCK_SHA256_FIELD_NUMBER: _ClassVar[int]
    TRIM_START_PTS_FIELD_NUMBER: _ClassVar[int]
    TRIM_END_PTS_FIELD_NUMBER: _ClassVar[int]
    VIDEO_COMPOSITION_PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    VIDEO_COMPOSITION_PROFILE_VERSION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ORDINAL_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PTS_FIELD_NUMBER: _ClassVar[int]
    PROJECTED_PNG_SHA256_FIELD_NUMBER: _ClassVar[int]
    PROJECTED_PNG_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    PROJECTED_PNG_WIDTH_FIELD_NUMBER: _ClassVar[int]
    PROJECTED_PNG_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    COMPOSED_FRAME_SHA256_FIELD_NUMBER: _ClassVar[int]
    COMPOSED_FRAME_WIDTH_FIELD_NUMBER: _ClassVar[int]
    COMPOSED_FRAME_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    PIXEL_PROFILE_FIELD_NUMBER: _ClassVar[int]
    BASE_RECIPE_SHA256_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_TOP_FILL_RED_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_TOP_FILL_GREEN_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_TOP_FILL_BLUE_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_BOTTOM_FILL_RED_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_BOTTOM_FILL_GREEN_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_BOTTOM_FILL_BLUE_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_STATUS_INK_RED_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_STATUS_INK_GREEN_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_STATUS_INK_BLUE_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_HOME_INK_RED_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_HOME_INK_GREEN_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_HOME_INK_BLUE_FIELD_NUMBER: _ClassVar[int]
    BASE_SHOT_REVISION_FIELD_NUMBER: _ClassVar[int]
    BASE_RECIPE_VERSION_FIELD_NUMBER: _ClassVar[int]
    TOP_FILL_SOURCE_RESOLVED_FIELD_NUMBER: _ClassVar[int]
    BOTTOM_FILL_SOURCE_RESOLVED_FIELD_NUMBER: _ClassVar[int]
    STATUS_INK_SOURCE_RESOLVED_FIELD_NUMBER: _ClassVar[int]
    HOME_INK_SOURCE_RESOLVED_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    attempt_id: str
    receipt_id: str
    receipt_sha256: str
    source_sha256: str
    source_size_bytes: int
    clock_sha256: str
    trim_start_pts: int
    trim_end_pts: int
    video_composition_profile_id: str
    video_composition_profile_version: int
    source_ordinal: int
    source_pts: int
    projected_png_sha256: str
    projected_png_size_bytes: int
    projected_png_width: int
    projected_png_height: int
    composed_frame_sha256: str
    composed_frame_width: int
    composed_frame_height: int
    pixel_profile: str
    base_recipe_sha256: str
    resolved_top_fill_red: int
    resolved_top_fill_green: int
    resolved_top_fill_blue: int
    resolved_bottom_fill_red: int
    resolved_bottom_fill_green: int
    resolved_bottom_fill_blue: int
    resolved_status_ink_red: int
    resolved_status_ink_green: int
    resolved_status_ink_blue: int
    resolved_home_ink_red: int
    resolved_home_ink_green: int
    resolved_home_ink_blue: int
    base_shot_revision: int
    base_recipe_version: str
    top_fill_source_resolved: bool
    bottom_fill_source_resolved: bool
    status_ink_source_resolved: bool
    home_ink_source_resolved: bool
    def __init__(self, job_id: _Optional[str] = ..., attempt_id: _Optional[str] = ..., receipt_id: _Optional[str] = ..., receipt_sha256: _Optional[str] = ..., source_sha256: _Optional[str] = ..., source_size_bytes: _Optional[int] = ..., clock_sha256: _Optional[str] = ..., trim_start_pts: _Optional[int] = ..., trim_end_pts: _Optional[int] = ..., video_composition_profile_id: _Optional[str] = ..., video_composition_profile_version: _Optional[int] = ..., source_ordinal: _Optional[int] = ..., source_pts: _Optional[int] = ..., projected_png_sha256: _Optional[str] = ..., projected_png_size_bytes: _Optional[int] = ..., projected_png_width: _Optional[int] = ..., projected_png_height: _Optional[int] = ..., composed_frame_sha256: _Optional[str] = ..., composed_frame_width: _Optional[int] = ..., composed_frame_height: _Optional[int] = ..., pixel_profile: _Optional[str] = ..., base_recipe_sha256: _Optional[str] = ..., resolved_top_fill_red: _Optional[int] = ..., resolved_top_fill_green: _Optional[int] = ..., resolved_top_fill_blue: _Optional[int] = ..., resolved_bottom_fill_red: _Optional[int] = ..., resolved_bottom_fill_green: _Optional[int] = ..., resolved_bottom_fill_blue: _Optional[int] = ..., resolved_status_ink_red: _Optional[int] = ..., resolved_status_ink_green: _Optional[int] = ..., resolved_status_ink_blue: _Optional[int] = ..., resolved_home_ink_red: _Optional[int] = ..., resolved_home_ink_green: _Optional[int] = ..., resolved_home_ink_blue: _Optional[int] = ..., base_shot_revision: _Optional[int] = ..., base_recipe_version: _Optional[str] = ..., top_fill_source_resolved: _Optional[bool] = ..., bottom_fill_source_resolved: _Optional[bool] = ..., status_ink_source_resolved: _Optional[bool] = ..., home_ink_source_resolved: _Optional[bool] = ...) -> None: ...

class ResolvedSourceMedia(_message.Message):
    __slots__ = ("id", "sha256", "size_bytes", "media_type", "width_pixels", "height_pixels")
    ID_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    WIDTH_PIXELS_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_PIXELS_FIELD_NUMBER: _ClassVar[int]
    id: str
    sha256: str
    size_bytes: int
    media_type: str
    width_pixels: int
    height_pixels: int
    def __init__(self, id: _Optional[str] = ..., sha256: _Optional[str] = ..., size_bytes: _Optional[int] = ..., media_type: _Optional[str] = ..., width_pixels: _Optional[int] = ..., height_pixels: _Optional[int] = ...) -> None: ...

class ShotIssue(_message.Message):
    __slots__ = ("code", "field_path", "summary", "repair", "blocks_render")
    CODE_FIELD_NUMBER: _ClassVar[int]
    FIELD_PATH_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    REPAIR_FIELD_NUMBER: _ClassVar[int]
    BLOCKS_RENDER_FIELD_NUMBER: _ClassVar[int]
    code: ShotIssueCode
    field_path: str
    summary: str
    repair: str
    blocks_render: bool
    def __init__(self, code: _Optional[_Union[ShotIssueCode, str]] = ..., field_path: _Optional[str] = ..., summary: _Optional[str] = ..., repair: _Optional[str] = ..., blocks_render: _Optional[bool] = ...) -> None: ...

class ShotRevision(_message.Message):
    __slots__ = ("shot_id", "revision", "created_at", "name", "snapshot", "recipe_sha256", "scene_sha256", "presentation_sha256", "scene_version", "presentation_version", "compositor_profile", "plate", "active_mask", "hardware_mask", "screen", "render_eligible", "issues")
    SHOT_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    RECIPE_SHA256_FIELD_NUMBER: _ClassVar[int]
    SCENE_SHA256_FIELD_NUMBER: _ClassVar[int]
    PRESENTATION_SHA256_FIELD_NUMBER: _ClassVar[int]
    SCENE_VERSION_FIELD_NUMBER: _ClassVar[int]
    PRESENTATION_VERSION_FIELD_NUMBER: _ClassVar[int]
    COMPOSITOR_PROFILE_FIELD_NUMBER: _ClassVar[int]
    PLATE_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_MASK_FIELD_NUMBER: _ClassVar[int]
    HARDWARE_MASK_FIELD_NUMBER: _ClassVar[int]
    SCREEN_FIELD_NUMBER: _ClassVar[int]
    RENDER_ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    shot_id: str
    revision: int
    created_at: _timestamp_pb2.Timestamp
    name: str
    snapshot: ShotSnapshot
    recipe_sha256: str
    scene_sha256: str
    presentation_sha256: str
    scene_version: str
    presentation_version: str
    compositor_profile: str
    plate: ResolvedSourceMedia
    active_mask: ResolvedSourceMedia
    hardware_mask: ResolvedSourceMedia
    screen: ResolvedSourceMedia
    render_eligible: bool
    issues: _containers.RepeatedCompositeFieldContainer[ShotIssue]
    def __init__(self, shot_id: _Optional[str] = ..., revision: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., name: _Optional[str] = ..., snapshot: _Optional[_Union[ShotSnapshot, _Mapping]] = ..., recipe_sha256: _Optional[str] = ..., scene_sha256: _Optional[str] = ..., presentation_sha256: _Optional[str] = ..., scene_version: _Optional[str] = ..., presentation_version: _Optional[str] = ..., compositor_profile: _Optional[str] = ..., plate: _Optional[_Union[ResolvedSourceMedia, _Mapping]] = ..., active_mask: _Optional[_Union[ResolvedSourceMedia, _Mapping]] = ..., hardware_mask: _Optional[_Union[ResolvedSourceMedia, _Mapping]] = ..., screen: _Optional[_Union[ResolvedSourceMedia, _Mapping]] = ..., render_eligible: _Optional[bool] = ..., issues: _Optional[_Iterable[_Union[ShotIssue, _Mapping]]] = ...) -> None: ...

class CompiledShot(_message.Message):
    __slots__ = ("project_id", "shot_id", "revision", "recipe_sha256", "compositor_profile", "snapshot", "plate", "active_mask", "hardware_mask", "screen", "issues", "render_eligible", "scene_sha256", "presentation_sha256", "scene_version", "presentation_version", "physical_compatibility")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    SHOT_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    RECIPE_SHA256_FIELD_NUMBER: _ClassVar[int]
    COMPOSITOR_PROFILE_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    PLATE_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_MASK_FIELD_NUMBER: _ClassVar[int]
    HARDWARE_MASK_FIELD_NUMBER: _ClassVar[int]
    SCREEN_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    RENDER_ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    SCENE_SHA256_FIELD_NUMBER: _ClassVar[int]
    PRESENTATION_SHA256_FIELD_NUMBER: _ClassVar[int]
    SCENE_VERSION_FIELD_NUMBER: _ClassVar[int]
    PRESENTATION_VERSION_FIELD_NUMBER: _ClassVar[int]
    PHYSICAL_COMPATIBILITY_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    shot_id: str
    revision: int
    recipe_sha256: str
    compositor_profile: str
    snapshot: ShotSnapshot
    plate: ResolvedSourceMedia
    active_mask: ResolvedSourceMedia
    hardware_mask: ResolvedSourceMedia
    screen: ResolvedSourceMedia
    issues: _containers.RepeatedCompositeFieldContainer[ShotIssue]
    render_eligible: bool
    scene_sha256: str
    presentation_sha256: str
    scene_version: str
    presentation_version: str
    physical_compatibility: PhysicalCompatibilityReport
    def __init__(self, project_id: _Optional[str] = ..., shot_id: _Optional[str] = ..., revision: _Optional[int] = ..., recipe_sha256: _Optional[str] = ..., compositor_profile: _Optional[str] = ..., snapshot: _Optional[_Union[ShotSnapshot, _Mapping]] = ..., plate: _Optional[_Union[ResolvedSourceMedia, _Mapping]] = ..., active_mask: _Optional[_Union[ResolvedSourceMedia, _Mapping]] = ..., hardware_mask: _Optional[_Union[ResolvedSourceMedia, _Mapping]] = ..., screen: _Optional[_Union[ResolvedSourceMedia, _Mapping]] = ..., issues: _Optional[_Iterable[_Union[ShotIssue, _Mapping]]] = ..., render_eligible: _Optional[bool] = ..., scene_sha256: _Optional[str] = ..., presentation_sha256: _Optional[str] = ..., scene_version: _Optional[str] = ..., presentation_version: _Optional[str] = ..., physical_compatibility: _Optional[_Union[PhysicalCompatibilityReport, _Mapping]] = ...) -> None: ...

class CreateShotRequest(_message.Message):
    __slots__ = ("project_id", "name", "snapshot")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    name: str
    snapshot: ShotSnapshot
    def __init__(self, project_id: _Optional[str] = ..., name: _Optional[str] = ..., snapshot: _Optional[_Union[ShotSnapshot, _Mapping]] = ...) -> None: ...

class CreateShotResponse(_message.Message):
    __slots__ = ("shot", "revision")
    SHOT_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    shot: Shot
    revision: ShotRevision
    def __init__(self, shot: _Optional[_Union[Shot, _Mapping]] = ..., revision: _Optional[_Union[ShotRevision, _Mapping]] = ...) -> None: ...

class SaveShotRequest(_message.Message):
    __slots__ = ("shot_id", "expected_revision", "name", "snapshot")
    SHOT_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    shot_id: str
    expected_revision: int
    name: str
    snapshot: ShotSnapshot
    def __init__(self, shot_id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., name: _Optional[str] = ..., snapshot: _Optional[_Union[ShotSnapshot, _Mapping]] = ...) -> None: ...

class SaveShotResponse(_message.Message):
    __slots__ = ("shot", "revision")
    SHOT_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    shot: Shot
    revision: ShotRevision
    def __init__(self, shot: _Optional[_Union[Shot, _Mapping]] = ..., revision: _Optional[_Union[ShotRevision, _Mapping]] = ...) -> None: ...

class ListShotsRequest(_message.Message):
    __slots__ = ("project_id", "page_size", "page_token")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    page_size: int
    page_token: str
    def __init__(self, project_id: _Optional[str] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListShotsResponse(_message.Message):
    __slots__ = ("shots", "next_page_token")
    SHOTS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    shots: _containers.RepeatedCompositeFieldContainer[Shot]
    next_page_token: str
    def __init__(self, shots: _Optional[_Iterable[_Union[Shot, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class GetShotRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetShotResponse(_message.Message):
    __slots__ = ("shot", "head")
    SHOT_FIELD_NUMBER: _ClassVar[int]
    HEAD_FIELD_NUMBER: _ClassVar[int]
    shot: Shot
    head: ShotRevision
    def __init__(self, shot: _Optional[_Union[Shot, _Mapping]] = ..., head: _Optional[_Union[ShotRevision, _Mapping]] = ...) -> None: ...

class GetShotRevisionRequest(_message.Message):
    __slots__ = ("shot_id", "revision")
    SHOT_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    shot_id: str
    revision: int
    def __init__(self, shot_id: _Optional[str] = ..., revision: _Optional[int] = ...) -> None: ...

class GetShotRevisionResponse(_message.Message):
    __slots__ = ("revision",)
    REVISION_FIELD_NUMBER: _ClassVar[int]
    revision: ShotRevision
    def __init__(self, revision: _Optional[_Union[ShotRevision, _Mapping]] = ...) -> None: ...

class ListShotRevisionsRequest(_message.Message):
    __slots__ = ("shot_id", "page_size", "page_token")
    SHOT_ID_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    shot_id: str
    page_size: int
    page_token: str
    def __init__(self, shot_id: _Optional[str] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListShotRevisionsResponse(_message.Message):
    __slots__ = ("revisions", "next_page_token")
    REVISIONS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    revisions: _containers.RepeatedCompositeFieldContainer[ShotRevision]
    next_page_token: str
    def __init__(self, revisions: _Optional[_Iterable[_Union[ShotRevision, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class CompileShotRequest(_message.Message):
    __slots__ = ("shot_id", "revision")
    SHOT_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    shot_id: str
    revision: int
    def __init__(self, shot_id: _Optional[str] = ..., revision: _Optional[int] = ...) -> None: ...

class CompileShotResponse(_message.Message):
    __slots__ = ("compiled",)
    COMPILED_FIELD_NUMBER: _ClassVar[int]
    compiled: CompiledShot
    def __init__(self, compiled: _Optional[_Union[CompiledShot, _Mapping]] = ...) -> None: ...

class ValidateShotRequest(_message.Message):
    __slots__ = ("project_id", "snapshot")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    snapshot: ShotSnapshot
    def __init__(self, project_id: _Optional[str] = ..., snapshot: _Optional[_Union[ShotSnapshot, _Mapping]] = ...) -> None: ...

class ValidateShotResponse(_message.Message):
    __slots__ = ("render_eligible", "issues", "compiled")
    RENDER_ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    COMPILED_FIELD_NUMBER: _ClassVar[int]
    render_eligible: bool
    issues: _containers.RepeatedCompositeFieldContainer[ShotIssue]
    compiled: CompiledShot
    def __init__(self, render_eligible: _Optional[bool] = ..., issues: _Optional[_Iterable[_Union[ShotIssue, _Mapping]]] = ..., compiled: _Optional[_Union[CompiledShot, _Mapping]] = ...) -> None: ...

class ShotConflict(_message.Message):
    __slots__ = ("shot_id", "expected_revision", "current_revision")
    SHOT_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    CURRENT_REVISION_FIELD_NUMBER: _ClassVar[int]
    shot_id: str
    expected_revision: int
    current_revision: int
    def __init__(self, shot_id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., current_revision: _Optional[int] = ...) -> None: ...

class RenderJobIssue(_message.Message):
    __slots__ = ("code", "summary")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    code: RenderJobIssueCode
    summary: str
    def __init__(self, code: _Optional[_Union[RenderJobIssueCode, str]] = ..., summary: _Optional[str] = ...) -> None: ...

class RenderOutput(_message.Message):
    __slots__ = ("media_type", "size_bytes", "width_pixels", "height_pixels", "sha256", "content_url")
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    WIDTH_PIXELS_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_PIXELS_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    CONTENT_URL_FIELD_NUMBER: _ClassVar[int]
    media_type: str
    size_bytes: int
    width_pixels: int
    height_pixels: int
    sha256: str
    content_url: str
    def __init__(self, media_type: _Optional[str] = ..., size_bytes: _Optional[int] = ..., width_pixels: _Optional[int] = ..., height_pixels: _Optional[int] = ..., sha256: _Optional[str] = ..., content_url: _Optional[str] = ...) -> None: ...

class RenderJob(_message.Message):
    __slots__ = ("id", "project_id", "shot_id", "shot_revision", "status", "version", "created_at", "updated_at", "summary", "issue", "output", "render_mode", "output_profile")
    ID_FIELD_NUMBER: _ClassVar[int]
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    SHOT_ID_FIELD_NUMBER: _ClassVar[int]
    SHOT_REVISION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    ISSUE_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    RENDER_MODE_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_PROFILE_FIELD_NUMBER: _ClassVar[int]
    id: str
    project_id: str
    shot_id: str
    shot_revision: int
    status: RenderJobStatus
    version: int
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    summary: str
    issue: RenderJobIssue
    output: RenderOutput
    render_mode: RenderMode
    output_profile: RenderOutputProfile
    def __init__(self, id: _Optional[str] = ..., project_id: _Optional[str] = ..., shot_id: _Optional[str] = ..., shot_revision: _Optional[int] = ..., status: _Optional[_Union[RenderJobStatus, str]] = ..., version: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., summary: _Optional[str] = ..., issue: _Optional[_Union[RenderJobIssue, _Mapping]] = ..., output: _Optional[_Union[RenderOutput, _Mapping]] = ..., render_mode: _Optional[_Union[RenderMode, str]] = ..., output_profile: _Optional[_Union[RenderOutputProfile, str]] = ...) -> None: ...

class SubmitRenderRequest(_message.Message):
    __slots__ = ("project_id", "shot_id", "revision", "idempotency_key", "output_profile", "render_mode", "sample_output_index", "reuse_cached_success")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    SHOT_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_PROFILE_FIELD_NUMBER: _ClassVar[int]
    RENDER_MODE_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_OUTPUT_INDEX_FIELD_NUMBER: _ClassVar[int]
    REUSE_CACHED_SUCCESS_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    shot_id: str
    revision: int
    idempotency_key: str
    output_profile: RenderOutputProfile
    render_mode: RenderMode
    sample_output_index: int
    reuse_cached_success: bool
    def __init__(self, project_id: _Optional[str] = ..., shot_id: _Optional[str] = ..., revision: _Optional[int] = ..., idempotency_key: _Optional[str] = ..., output_profile: _Optional[_Union[RenderOutputProfile, str]] = ..., render_mode: _Optional[_Union[RenderMode, str]] = ..., sample_output_index: _Optional[int] = ..., reuse_cached_success: _Optional[bool] = ...) -> None: ...

class SubmitRenderResponse(_message.Message):
    __slots__ = ("job", "reused_existing", "cache_reuse")
    JOB_FIELD_NUMBER: _ClassVar[int]
    REUSED_EXISTING_FIELD_NUMBER: _ClassVar[int]
    CACHE_REUSE_FIELD_NUMBER: _ClassVar[int]
    job: RenderJob
    reused_existing: bool
    cache_reuse: CachedRenderReuse
    def __init__(self, job: _Optional[_Union[RenderJob, _Mapping]] = ..., reused_existing: _Optional[bool] = ..., cache_reuse: _Optional[_Union[CachedRenderReuse, _Mapping]] = ...) -> None: ...

class CachedRenderReuse(_message.Message):
    __slots__ = ("source_render_id", "source_attempt_id", "artifact_sha256", "output_kind")
    SOURCE_RENDER_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ATTEMPT_ID_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_SHA256_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_KIND_FIELD_NUMBER: _ClassVar[int]
    source_render_id: str
    source_attempt_id: str
    artifact_sha256: str
    output_kind: str
    def __init__(self, source_render_id: _Optional[str] = ..., source_attempt_id: _Optional[str] = ..., artifact_sha256: _Optional[str] = ..., output_kind: _Optional[str] = ...) -> None: ...

class GetRenderRequest(_message.Message):
    __slots__ = ("project_id", "render_id")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    RENDER_ID_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    render_id: str
    def __init__(self, project_id: _Optional[str] = ..., render_id: _Optional[str] = ...) -> None: ...

class GetRenderResponse(_message.Message):
    __slots__ = ("job",)
    JOB_FIELD_NUMBER: _ClassVar[int]
    job: RenderJob
    def __init__(self, job: _Optional[_Union[RenderJob, _Mapping]] = ...) -> None: ...

class ListRendersRequest(_message.Message):
    __slots__ = ("project_id", "page_size", "page_token")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    page_size: int
    page_token: str
    def __init__(self, project_id: _Optional[str] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListRendersResponse(_message.Message):
    __slots__ = ("jobs", "next_page_token")
    JOBS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    jobs: _containers.RepeatedCompositeFieldContainer[RenderJob]
    next_page_token: str
    def __init__(self, jobs: _Optional[_Iterable[_Union[RenderJob, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class CancelRenderRequest(_message.Message):
    __slots__ = ("project_id", "render_id", "expected_version")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    RENDER_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_VERSION_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    render_id: str
    expected_version: int
    def __init__(self, project_id: _Optional[str] = ..., render_id: _Optional[str] = ..., expected_version: _Optional[int] = ...) -> None: ...

class CancelRenderResponse(_message.Message):
    __slots__ = ("job",)
    JOB_FIELD_NUMBER: _ClassVar[int]
    job: RenderJob
    def __init__(self, job: _Optional[_Union[RenderJob, _Mapping]] = ...) -> None: ...
