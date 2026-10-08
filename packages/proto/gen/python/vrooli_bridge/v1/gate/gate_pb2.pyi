import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GateVerdict(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    GATE_VERDICT_UNSPECIFIED: _ClassVar[GateVerdict]
    GATE_VERDICT_PENDING: _ClassVar[GateVerdict]
    GATE_VERDICT_PASSED: _ClassVar[GateVerdict]
    GATE_VERDICT_FAILED: _ClassVar[GateVerdict]

class OSDisposition(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OS_DISPOSITION_UNSPECIFIED: _ClassVar[OSDisposition]
    OS_DISPOSITION_PENDING: _ClassVar[OSDisposition]
    OS_DISPOSITION_PASSED: _ClassVar[OSDisposition]
    OS_DISPOSITION_FAILED: _ClassVar[OSDisposition]
    OS_DISPOSITION_ABORTED: _ClassVar[OSDisposition]
    OS_DISPOSITION_NO_NODE: _ClassVar[OSDisposition]
    OS_DISPOSITION_DISPATCH_FAILED: _ClassVar[OSDisposition]
GATE_VERDICT_UNSPECIFIED: GateVerdict
GATE_VERDICT_PENDING: GateVerdict
GATE_VERDICT_PASSED: GateVerdict
GATE_VERDICT_FAILED: GateVerdict
OS_DISPOSITION_UNSPECIFIED: OSDisposition
OS_DISPOSITION_PENDING: OSDisposition
OS_DISPOSITION_PASSED: OSDisposition
OS_DISPOSITION_FAILED: OSDisposition
OS_DISPOSITION_ABORTED: OSDisposition
OS_DISPOSITION_NO_NODE: OSDisposition
OS_DISPOSITION_DISPATCH_FAILED: OSDisposition

class OSResult(_message.Message):
    __slots__ = ("os", "node_id", "run_id", "disposition", "exit_code", "detail")
    OS_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    os: str
    node_id: str
    run_id: str
    disposition: OSDisposition
    exit_code: int
    detail: str
    def __init__(self, os: _Optional[str] = ..., node_id: _Optional[str] = ..., run_id: _Optional[str] = ..., disposition: _Optional[_Union[OSDisposition, str]] = ..., exit_code: _Optional[int] = ..., detail: _Optional[str] = ...) -> None: ...

class Gate(_message.Message):
    __slots__ = ("id", "scenario", "target_revision", "verb", "args", "verdict", "total_targets", "passed", "failed", "pending", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TARGET_REVISION_FIELD_NUMBER: _ClassVar[int]
    VERB_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_TARGETS_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    PENDING_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario: str
    target_revision: str
    verb: str
    args: _containers.RepeatedScalarFieldContainer[str]
    verdict: GateVerdict
    total_targets: int
    passed: int
    failed: int
    pending: int
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., scenario: _Optional[str] = ..., target_revision: _Optional[str] = ..., verb: _Optional[str] = ..., args: _Optional[_Iterable[str]] = ..., verdict: _Optional[_Union[GateVerdict, str]] = ..., total_targets: _Optional[int] = ..., passed: _Optional[int] = ..., failed: _Optional[int] = ..., pending: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RunGateRequest(_message.Message):
    __slots__ = ("scenario", "target_revision", "target_oses", "verb", "args", "timeout_seconds")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TARGET_REVISION_FIELD_NUMBER: _ClassVar[int]
    TARGET_OSES_FIELD_NUMBER: _ClassVar[int]
    VERB_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    target_revision: str
    target_oses: _containers.RepeatedScalarFieldContainer[str]
    verb: str
    args: _containers.RepeatedScalarFieldContainer[str]
    timeout_seconds: int
    def __init__(self, scenario: _Optional[str] = ..., target_revision: _Optional[str] = ..., target_oses: _Optional[_Iterable[str]] = ..., verb: _Optional[str] = ..., args: _Optional[_Iterable[str]] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class RunGateResponse(_message.Message):
    __slots__ = ("gate_id", "dry_run", "verdict", "results")
    GATE_ID_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    gate_id: str
    dry_run: bool
    verdict: GateVerdict
    results: _containers.RepeatedCompositeFieldContainer[OSResult]
    def __init__(self, gate_id: _Optional[str] = ..., dry_run: _Optional[bool] = ..., verdict: _Optional[_Union[GateVerdict, str]] = ..., results: _Optional[_Iterable[_Union[OSResult, _Mapping]]] = ...) -> None: ...

class GetGateRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetGateResponse(_message.Message):
    __slots__ = ("gate", "results")
    GATE_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    gate: Gate
    results: _containers.RepeatedCompositeFieldContainer[OSResult]
    def __init__(self, gate: _Optional[_Union[Gate, _Mapping]] = ..., results: _Optional[_Iterable[_Union[OSResult, _Mapping]]] = ...) -> None: ...

class WaitGateRequest(_message.Message):
    __slots__ = ("id", "timeout_seconds")
    ID_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    id: str
    timeout_seconds: int
    def __init__(self, id: _Optional[str] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class WaitGateResponse(_message.Message):
    __slots__ = ("gate", "results", "timed_out")
    GATE_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    TIMED_OUT_FIELD_NUMBER: _ClassVar[int]
    gate: Gate
    results: _containers.RepeatedCompositeFieldContainer[OSResult]
    timed_out: bool
    def __init__(self, gate: _Optional[_Union[Gate, _Mapping]] = ..., results: _Optional[_Iterable[_Union[OSResult, _Mapping]]] = ..., timed_out: _Optional[bool] = ...) -> None: ...

class ListGatesRequest(_message.Message):
    __slots__ = ("limit",)
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    def __init__(self, limit: _Optional[int] = ...) -> None: ...

class ListGatesResponse(_message.Message):
    __slots__ = ("gates",)
    GATES_FIELD_NUMBER: _ClassVar[int]
    gates: _containers.RepeatedCompositeFieldContainer[Gate]
    def __init__(self, gates: _Optional[_Iterable[_Union[Gate, _Mapping]]] = ...) -> None: ...
