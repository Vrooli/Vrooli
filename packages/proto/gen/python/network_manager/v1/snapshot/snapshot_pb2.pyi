from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Snapshot(_message.Message):
    __slots__ = ("id", "status", "profile", "summary", "metrics", "findings", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    status: str
    profile: str
    summary: str
    metrics: _containers.RepeatedCompositeFieldContainer[Metric]
    findings: _containers.RepeatedScalarFieldContainer[str]
    created_at: str
    def __init__(self, id: _Optional[str] = ..., status: _Optional[str] = ..., profile: _Optional[str] = ..., summary: _Optional[str] = ..., metrics: _Optional[_Iterable[_Union[Metric, _Mapping]]] = ..., findings: _Optional[_Iterable[str]] = ..., created_at: _Optional[str] = ...) -> None: ...

class Metric(_message.Message):
    __slots__ = ("name", "value", "unit", "status")
    NAME_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    name: str
    value: str
    unit: str
    status: str
    def __init__(self, name: _Optional[str] = ..., value: _Optional[str] = ..., unit: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class RunSnapshotRequest(_message.Message):
    __slots__ = ("profile", "dry_run")
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    profile: str
    dry_run: bool
    def __init__(self, profile: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class RunSnapshotResponse(_message.Message):
    __slots__ = ("snapshot",)
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    snapshot: Snapshot
    def __init__(self, snapshot: _Optional[_Union[Snapshot, _Mapping]] = ...) -> None: ...

class ListSnapshotsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListSnapshotsResponse(_message.Message):
    __slots__ = ("snapshots",)
    SNAPSHOTS_FIELD_NUMBER: _ClassVar[int]
    snapshots: _containers.RepeatedCompositeFieldContainer[Snapshot]
    def __init__(self, snapshots: _Optional[_Iterable[_Union[Snapshot, _Mapping]]] = ...) -> None: ...

class GetSnapshotRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetSnapshotResponse(_message.Message):
    __slots__ = ("snapshot",)
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    snapshot: Snapshot
    def __init__(self, snapshot: _Optional[_Union[Snapshot, _Mapping]] = ...) -> None: ...

class ExportSnapshotReportRequest(_message.Message):
    __slots__ = ("id", "format")
    ID_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    id: str
    format: str
    def __init__(self, id: _Optional[str] = ..., format: _Optional[str] = ...) -> None: ...

class ExportSnapshotReportResponse(_message.Message):
    __slots__ = ("id", "format", "report")
    ID_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    id: str
    format: str
    report: str
    def __init__(self, id: _Optional[str] = ..., format: _Optional[str] = ..., report: _Optional[str] = ...) -> None: ...
