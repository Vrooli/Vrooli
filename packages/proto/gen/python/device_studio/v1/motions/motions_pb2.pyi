import datetime

from device_studio.v1.physical import physical_pb2 as _physical_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SaveMotionCandidateRequest(_message.Message):
    __slots__ = ("candidate", "expected_head")
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_HEAD_FIELD_NUMBER: _ClassVar[int]
    candidate: MotionCandidate
    expected_head: int
    def __init__(self, candidate: _Optional[_Union[MotionCandidate, _Mapping]] = ..., expected_head: _Optional[int] = ...) -> None: ...

class SaveMotionCandidateResponse(_message.Message):
    __slots__ = ("revision",)
    REVISION_FIELD_NUMBER: _ClassVar[int]
    revision: MotionRevision
    def __init__(self, revision: _Optional[_Union[MotionRevision, _Mapping]] = ...) -> None: ...

class GetMotionRevisionRequest(_message.Message):
    __slots__ = ("motion_id", "revision")
    MOTION_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    motion_id: str
    revision: int
    def __init__(self, motion_id: _Optional[str] = ..., revision: _Optional[int] = ...) -> None: ...

class GetMotionRevisionResponse(_message.Message):
    __slots__ = ("revision",)
    REVISION_FIELD_NUMBER: _ClassVar[int]
    revision: MotionRevision
    def __init__(self, revision: _Optional[_Union[MotionRevision, _Mapping]] = ...) -> None: ...

class GetMotionHeadRequest(_message.Message):
    __slots__ = ("motion_id",)
    MOTION_ID_FIELD_NUMBER: _ClassVar[int]
    motion_id: str
    def __init__(self, motion_id: _Optional[str] = ...) -> None: ...

class GetMotionHeadResponse(_message.Message):
    __slots__ = ("revision",)
    REVISION_FIELD_NUMBER: _ClassVar[int]
    revision: MotionRevision
    def __init__(self, revision: _Optional[_Union[MotionRevision, _Mapping]] = ...) -> None: ...

class ListMotionRevisionsRequest(_message.Message):
    __slots__ = ("motion_id", "after_revision", "page_size")
    MOTION_ID_FIELD_NUMBER: _ClassVar[int]
    AFTER_REVISION_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    motion_id: str
    after_revision: int
    page_size: int
    def __init__(self, motion_id: _Optional[str] = ..., after_revision: _Optional[int] = ..., page_size: _Optional[int] = ...) -> None: ...

class ListMotionRevisionsResponse(_message.Message):
    __slots__ = ("revisions", "next_after_revision")
    REVISIONS_FIELD_NUMBER: _ClassVar[int]
    NEXT_AFTER_REVISION_FIELD_NUMBER: _ClassVar[int]
    revisions: _containers.RepeatedCompositeFieldContainer[MotionRevision]
    next_after_revision: int
    def __init__(self, revisions: _Optional[_Iterable[_Union[MotionRevision, _Mapping]]] = ..., next_after_revision: _Optional[int] = ...) -> None: ...

class ListMotionCandidatesRequest(_message.Message):
    __slots__ = ("after_motion_id", "page_size")
    AFTER_MOTION_ID_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    after_motion_id: str
    page_size: int
    def __init__(self, after_motion_id: _Optional[str] = ..., page_size: _Optional[int] = ...) -> None: ...

class ListMotionCandidatesResponse(_message.Message):
    __slots__ = ("candidates", "next_after_motion_id")
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    NEXT_AFTER_MOTION_ID_FIELD_NUMBER: _ClassVar[int]
    candidates: _containers.RepeatedCompositeFieldContainer[MotionRevision]
    next_after_motion_id: str
    def __init__(self, candidates: _Optional[_Iterable[_Union[MotionRevision, _Mapping]]] = ..., next_after_motion_id: _Optional[str] = ...) -> None: ...

class MotionRevision(_message.Message):
    __slots__ = ("motion_id", "revision", "sha256", "state", "candidate", "created_at")
    MOTION_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    motion_id: str
    revision: int
    sha256: str
    state: str
    candidate: MotionCandidate
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, motion_id: _Optional[str] = ..., revision: _Optional[int] = ..., sha256: _Optional[str] = ..., state: _Optional[str] = ..., candidate: _Optional[_Union[MotionCandidate, _Mapping]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Rational(_message.Message):
    __slots__ = ("num", "den")
    NUM_FIELD_NUMBER: _ClassVar[int]
    DEN_FIELD_NUMBER: _ClassVar[int]
    num: int
    den: int
    def __init__(self, num: _Optional[int] = ..., den: _Optional[int] = ...) -> None: ...

class MotionCandidate(_message.Message):
    __slots__ = ("version", "motion_id", "timebase_ticks_per_second", "duration_ticks", "tracks", "required_capabilities")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    MOTION_ID_FIELD_NUMBER: _ClassVar[int]
    TIMEBASE_TICKS_PER_SECOND_FIELD_NUMBER: _ClassVar[int]
    DURATION_TICKS_FIELD_NUMBER: _ClassVar[int]
    TRACKS_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    version: str
    motion_id: str
    timebase_ticks_per_second: Rational
    duration_ticks: int
    tracks: _containers.RepeatedCompositeFieldContainer[MotionTrack]
    required_capabilities: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, version: _Optional[str] = ..., motion_id: _Optional[str] = ..., timebase_ticks_per_second: _Optional[_Union[Rational, _Mapping]] = ..., duration_ticks: _Optional[int] = ..., tracks: _Optional[_Iterable[_Union[MotionTrack, _Mapping]]] = ..., required_capabilities: _Optional[_Iterable[str]] = ...) -> None: ...

class TrackTarget(_message.Message):
    __slots__ = ("device_root", "joint_id", "scene_camera_id")
    DEVICE_ROOT_FIELD_NUMBER: _ClassVar[int]
    JOINT_ID_FIELD_NUMBER: _ClassVar[int]
    SCENE_CAMERA_ID_FIELD_NUMBER: _ClassVar[int]
    device_root: bool
    joint_id: str
    scene_camera_id: str
    def __init__(self, device_root: _Optional[bool] = ..., joint_id: _Optional[str] = ..., scene_camera_id: _Optional[str] = ...) -> None: ...

class MotionTrack(_message.Message):
    __slots__ = ("target", "interpolation", "keys")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    INTERPOLATION_FIELD_NUMBER: _ClassVar[int]
    KEYS_FIELD_NUMBER: _ClassVar[int]
    target: TrackTarget
    interpolation: str
    keys: _containers.RepeatedCompositeFieldContainer[MotionKey]
    def __init__(self, target: _Optional[_Union[TrackTarget, _Mapping]] = ..., interpolation: _Optional[str] = ..., keys: _Optional[_Iterable[_Union[MotionKey, _Mapping]]] = ...) -> None: ...

class MotionKey(_message.Message):
    __slots__ = ("tick", "transform", "angle_rad")
    TICK_FIELD_NUMBER: _ClassVar[int]
    TRANSFORM_FIELD_NUMBER: _ClassVar[int]
    ANGLE_RAD_FIELD_NUMBER: _ClassVar[int]
    tick: int
    transform: TransformKey
    angle_rad: float
    def __init__(self, tick: _Optional[int] = ..., transform: _Optional[_Union[TransformKey, _Mapping]] = ..., angle_rad: _Optional[float] = ...) -> None: ...

class TransformKey(_message.Message):
    __slots__ = ("translation_m", "rotation_xyzw")
    TRANSLATION_M_FIELD_NUMBER: _ClassVar[int]
    ROTATION_XYZW_FIELD_NUMBER: _ClassVar[int]
    translation_m: _physical_pb2.Vec3
    rotation_xyzw: _physical_pb2.QuatXYZW
    def __init__(self, translation_m: _Optional[_Union[_physical_pb2.Vec3, _Mapping]] = ..., rotation_xyzw: _Optional[_Union[_physical_pb2.QuatXYZW, _Mapping]] = ...) -> None: ...
