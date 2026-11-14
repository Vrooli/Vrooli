import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RunStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RUN_STATUS_UNSPECIFIED: _ClassVar[RunStatus]
    RUN_STATUS_PASSED: _ClassVar[RunStatus]
    RUN_STATUS_FAILED: _ClassVar[RunStatus]
    RUN_STATUS_ERROR: _ClassVar[RunStatus]

class RunMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RUN_MODE_UNSPECIFIED: _ClassVar[RunMode]
    RUN_MODE_RUN: _ClassVar[RunMode]
    RUN_MODE_CHECK: _ClassVar[RunMode]

class FailureReason(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FAILURE_REASON_UNSPECIFIED: _ClassVar[FailureReason]
    FAILURE_REASON_MISSING_ARTIFACTS: _ClassVar[FailureReason]
    FAILURE_REASON_STALE_ARTIFACTS: _ClassVar[FailureReason]
    FAILURE_REASON_COUNTEREXAMPLE: _ClassVar[FailureReason]
    FAILURE_REASON_LINT: _ClassVar[FailureReason]
    FAILURE_REASON_QUINT_FAILURE: _ClassVar[FailureReason]
    FAILURE_REASON_IO: _ClassVar[FailureReason]
RUN_STATUS_UNSPECIFIED: RunStatus
RUN_STATUS_PASSED: RunStatus
RUN_STATUS_FAILED: RunStatus
RUN_STATUS_ERROR: RunStatus
RUN_MODE_UNSPECIFIED: RunMode
RUN_MODE_RUN: RunMode
RUN_MODE_CHECK: RunMode
FAILURE_REASON_UNSPECIFIED: FailureReason
FAILURE_REASON_MISSING_ARTIFACTS: FailureReason
FAILURE_REASON_STALE_ARTIFACTS: FailureReason
FAILURE_REASON_COUNTEREXAMPLE: FailureReason
FAILURE_REASON_LINT: FailureReason
FAILURE_REASON_QUINT_FAILURE: FailureReason
FAILURE_REASON_IO: FailureReason

class Run(_message.Message):
    __slots__ = ("id", "flow_id", "flow_path", "root", "source_sha256", "model_sha256", "gen_sha256", "mode", "status", "counterexample", "error_message", "failure_reason", "missing_artifacts", "output", "started_at", "finished_at", "duration_ms")
    ID_FIELD_NUMBER: _ClassVar[int]
    FLOW_ID_FIELD_NUMBER: _ClassVar[int]
    FLOW_PATH_FIELD_NUMBER: _ClassVar[int]
    ROOT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SHA256_FIELD_NUMBER: _ClassVar[int]
    MODEL_SHA256_FIELD_NUMBER: _ClassVar[int]
    GEN_SHA256_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    COUNTEREXAMPLE_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    FAILURE_REASON_FIELD_NUMBER: _ClassVar[int]
    MISSING_ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    id: str
    flow_id: str
    flow_path: str
    root: str
    source_sha256: str
    model_sha256: str
    gen_sha256: str
    mode: RunMode
    status: RunStatus
    counterexample: str
    error_message: str
    failure_reason: FailureReason
    missing_artifacts: _containers.RepeatedScalarFieldContainer[str]
    output: str
    started_at: _timestamp_pb2.Timestamp
    finished_at: _timestamp_pb2.Timestamp
    duration_ms: int
    def __init__(self, id: _Optional[str] = ..., flow_id: _Optional[str] = ..., flow_path: _Optional[str] = ..., root: _Optional[str] = ..., source_sha256: _Optional[str] = ..., model_sha256: _Optional[str] = ..., gen_sha256: _Optional[str] = ..., mode: _Optional[_Union[RunMode, str]] = ..., status: _Optional[_Union[RunStatus, str]] = ..., counterexample: _Optional[str] = ..., error_message: _Optional[str] = ..., failure_reason: _Optional[_Union[FailureReason, str]] = ..., missing_artifacts: _Optional[_Iterable[str]] = ..., output: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., finished_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., duration_ms: _Optional[int] = ...) -> None: ...

class ListRunsRequest(_message.Message):
    __slots__ = ("flow_id", "limit")
    FLOW_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    flow_id: str
    limit: int
    def __init__(self, flow_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListRunsResponse(_message.Message):
    __slots__ = ("runs",)
    RUNS_FIELD_NUMBER: _ClassVar[int]
    runs: _containers.RepeatedCompositeFieldContainer[Run]
    def __init__(self, runs: _Optional[_Iterable[_Union[Run, _Mapping]]] = ...) -> None: ...

class GetRunRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetRunResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: Run
    def __init__(self, run: _Optional[_Union[Run, _Mapping]] = ...) -> None: ...
