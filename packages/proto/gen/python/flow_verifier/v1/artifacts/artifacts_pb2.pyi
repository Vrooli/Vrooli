import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ArtifactStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ARTIFACT_STATUS_UNSPECIFIED: _ClassVar[ArtifactStatus]
    ARTIFACT_STATUS_FRESH: _ClassVar[ArtifactStatus]
    ARTIFACT_STATUS_MISSING: _ClassVar[ArtifactStatus]
    ARTIFACT_STATUS_STALE: _ClassVar[ArtifactStatus]
ARTIFACT_STATUS_UNSPECIFIED: ArtifactStatus
ARTIFACT_STATUS_FRESH: ArtifactStatus
ARTIFACT_STATUS_MISSING: ArtifactStatus
ARTIFACT_STATUS_STALE: ArtifactStatus

class ArtifactFile(_message.Message):
    __slots__ = ("path", "exists", "size", "mtime")
    PATH_FIELD_NUMBER: _ClassVar[int]
    EXISTS_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    MTIME_FIELD_NUMBER: _ClassVar[int]
    path: str
    exists: bool
    size: int
    mtime: _timestamp_pb2.Timestamp
    def __init__(self, path: _Optional[str] = ..., exists: _Optional[bool] = ..., size: _Optional[int] = ..., mtime: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ArtifactReport(_message.Message):
    __slots__ = ("flow_id", "scenario_path", "generated_dir", "status", "files", "missing")
    FLOW_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_PATH_FIELD_NUMBER: _ClassVar[int]
    GENERATED_DIR_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    MISSING_FIELD_NUMBER: _ClassVar[int]
    flow_id: str
    scenario_path: str
    generated_dir: str
    status: ArtifactStatus
    files: _containers.RepeatedCompositeFieldContainer[ArtifactFile]
    missing: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, flow_id: _Optional[str] = ..., scenario_path: _Optional[str] = ..., generated_dir: _Optional[str] = ..., status: _Optional[_Union[ArtifactStatus, str]] = ..., files: _Optional[_Iterable[_Union[ArtifactFile, _Mapping]]] = ..., missing: _Optional[_Iterable[str]] = ...) -> None: ...

class GetArtifactStatusRequest(_message.Message):
    __slots__ = ("flow_id", "scenario_id", "root")
    FLOW_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    ROOT_FIELD_NUMBER: _ClassVar[int]
    flow_id: str
    scenario_id: str
    root: str
    def __init__(self, flow_id: _Optional[str] = ..., scenario_id: _Optional[str] = ..., root: _Optional[str] = ...) -> None: ...

class GetArtifactStatusResponse(_message.Message):
    __slots__ = ("report",)
    REPORT_FIELD_NUMBER: _ClassVar[int]
    report: ArtifactReport
    def __init__(self, report: _Optional[_Union[ArtifactReport, _Mapping]] = ...) -> None: ...

class GenerateArtifactsRequest(_message.Message):
    __slots__ = ("flow_id", "scenario_id", "root")
    FLOW_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    ROOT_FIELD_NUMBER: _ClassVar[int]
    flow_id: str
    scenario_id: str
    root: str
    def __init__(self, flow_id: _Optional[str] = ..., scenario_id: _Optional[str] = ..., root: _Optional[str] = ...) -> None: ...

class GenerateArtifactsResponse(_message.Message):
    __slots__ = ("report", "run_id")
    REPORT_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    report: ArtifactReport
    run_id: str
    def __init__(self, report: _Optional[_Union[ArtifactReport, _Mapping]] = ..., run_id: _Optional[str] = ...) -> None: ...

class ClearArtifactsRequest(_message.Message):
    __slots__ = ("flow_id", "scenario_id", "root")
    FLOW_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    ROOT_FIELD_NUMBER: _ClassVar[int]
    flow_id: str
    scenario_id: str
    root: str
    def __init__(self, flow_id: _Optional[str] = ..., scenario_id: _Optional[str] = ..., root: _Optional[str] = ...) -> None: ...

class ClearArtifactsResponse(_message.Message):
    __slots__ = ("flow_id", "removed")
    FLOW_ID_FIELD_NUMBER: _ClassVar[int]
    REMOVED_FIELD_NUMBER: _ClassVar[int]
    flow_id: str
    removed: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, flow_id: _Optional[str] = ..., removed: _Optional[_Iterable[str]] = ...) -> None: ...
