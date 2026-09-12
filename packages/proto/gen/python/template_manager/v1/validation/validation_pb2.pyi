import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ValidationMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VALIDATION_MODE_UNSPECIFIED: _ClassVar[ValidationMode]
    VALIDATION_MODE_SHALLOW: _ClassVar[ValidationMode]
    VALIDATION_MODE_DEEP: _ClassVar[ValidationMode]
    VALIDATION_MODE_DRIFT: _ClassVar[ValidationMode]
VALIDATION_MODE_UNSPECIFIED: ValidationMode
VALIDATION_MODE_SHALLOW: ValidationMode
VALIDATION_MODE_DEEP: ValidationMode
VALIDATION_MODE_DRIFT: ValidationMode

class ValidationRun(_message.Message):
    __slots__ = ("id", "template_id", "mode", "target", "status", "started_at", "finished_at", "phase_results", "findings", "trigger")
    ID_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_ID_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    PHASE_RESULTS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_FIELD_NUMBER: _ClassVar[int]
    id: str
    template_id: str
    mode: ValidationMode
    target: str
    status: str
    started_at: _timestamp_pb2.Timestamp
    finished_at: _timestamp_pb2.Timestamp
    phase_results: _containers.RepeatedCompositeFieldContainer[PhaseResult]
    findings: _containers.RepeatedCompositeFieldContainer[ValidationFinding]
    trigger: str
    def __init__(self, id: _Optional[str] = ..., template_id: _Optional[str] = ..., mode: _Optional[_Union[ValidationMode, str]] = ..., target: _Optional[str] = ..., status: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., finished_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., phase_results: _Optional[_Iterable[_Union[PhaseResult, _Mapping]]] = ..., findings: _Optional[_Iterable[_Union[ValidationFinding, _Mapping]]] = ..., trigger: _Optional[str] = ...) -> None: ...

class PhaseResult(_message.Message):
    __slots__ = ("phase", "status", "finding_count")
    PHASE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    FINDING_COUNT_FIELD_NUMBER: _ClassVar[int]
    phase: str
    status: str
    finding_count: int
    def __init__(self, phase: _Optional[str] = ..., status: _Optional[str] = ..., finding_count: _Optional[int] = ...) -> None: ...

class ValidationFinding(_message.Message):
    __slots__ = ("key", "severity", "summary", "source")
    KEY_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    key: str
    severity: str
    summary: str
    source: str
    def __init__(self, key: _Optional[str] = ..., severity: _Optional[str] = ..., summary: _Optional[str] = ..., source: _Optional[str] = ...) -> None: ...

class DriftSnapshot(_message.Message):
    __slots__ = ("id", "template_id", "target", "status", "drift_count", "captured_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DRIFT_COUNT_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    template_id: str
    target: str
    status: str
    drift_count: int
    captured_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., template_id: _Optional[str] = ..., target: _Optional[str] = ..., status: _Optional[str] = ..., drift_count: _Optional[int] = ..., captured_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListValidationRunsRequest(_message.Message):
    __slots__ = ("template_id",)
    TEMPLATE_ID_FIELD_NUMBER: _ClassVar[int]
    template_id: str
    def __init__(self, template_id: _Optional[str] = ...) -> None: ...

class ListValidationRunsResponse(_message.Message):
    __slots__ = ("runs",)
    RUNS_FIELD_NUMBER: _ClassVar[int]
    runs: _containers.RepeatedCompositeFieldContainer[ValidationRun]
    def __init__(self, runs: _Optional[_Iterable[_Union[ValidationRun, _Mapping]]] = ...) -> None: ...

class GetValidationRunRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetValidationRunResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: ValidationRun
    def __init__(self, run: _Optional[_Union[ValidationRun, _Mapping]] = ...) -> None: ...

class ListDriftSnapshotsRequest(_message.Message):
    __slots__ = ("template_id",)
    TEMPLATE_ID_FIELD_NUMBER: _ClassVar[int]
    template_id: str
    def __init__(self, template_id: _Optional[str] = ...) -> None: ...

class ListDriftSnapshotsResponse(_message.Message):
    __slots__ = ("snapshots",)
    SNAPSHOTS_FIELD_NUMBER: _ClassVar[int]
    snapshots: _containers.RepeatedCompositeFieldContainer[DriftSnapshot]
    def __init__(self, snapshots: _Optional[_Iterable[_Union[DriftSnapshot, _Mapping]]] = ...) -> None: ...

class RunTemplateValidationRequest(_message.Message):
    __slots__ = ("template_id", "mode")
    TEMPLATE_ID_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    template_id: str
    mode: ValidationMode
    def __init__(self, template_id: _Optional[str] = ..., mode: _Optional[_Union[ValidationMode, str]] = ...) -> None: ...

class RunTemplateValidationResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: ValidationRun
    def __init__(self, run: _Optional[_Union[ValidationRun, _Mapping]] = ...) -> None: ...

class RecordFleetDriftRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RecordFleetDriftResponse(_message.Message):
    __slots__ = ("snapshot",)
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    snapshot: DriftSnapshot
    def __init__(self, snapshot: _Optional[_Union[DriftSnapshot, _Mapping]] = ...) -> None: ...
