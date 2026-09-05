import datetime

from architecture_cartographer.v1.shared import shared_pb2 as _shared_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class OperationKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OPERATION_KIND_UNSPECIFIED: _ClassVar[OperationKind]
    OPERATION_KIND_MOVE_FILE: _ClassVar[OperationKind]
    OPERATION_KIND_REWRITE_IMPORT: _ClassVar[OperationKind]
    OPERATION_KIND_DELETE_FILE: _ClassVar[OperationKind]
    OPERATION_KIND_CREATE_FILE: _ClassVar[OperationKind]

class ApplyStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    APPLY_STATUS_UNSPECIFIED: _ClassVar[ApplyStatus]
    APPLY_STATUS_PLANNED: _ClassVar[ApplyStatus]
    APPLY_STATUS_RUNNING: _ClassVar[ApplyStatus]
    APPLY_STATUS_BUILD_GREEN: _ClassVar[ApplyStatus]
    APPLY_STATUS_BUILD_RED: _ClassVar[ApplyStatus]
    APPLY_STATUS_REVERTED: _ClassVar[ApplyStatus]
    APPLY_STATUS_COMMITTED: _ClassVar[ApplyStatus]
OPERATION_KIND_UNSPECIFIED: OperationKind
OPERATION_KIND_MOVE_FILE: OperationKind
OPERATION_KIND_REWRITE_IMPORT: OperationKind
OPERATION_KIND_DELETE_FILE: OperationKind
OPERATION_KIND_CREATE_FILE: OperationKind
APPLY_STATUS_UNSPECIFIED: ApplyStatus
APPLY_STATUS_PLANNED: ApplyStatus
APPLY_STATUS_RUNNING: ApplyStatus
APPLY_STATUS_BUILD_GREEN: ApplyStatus
APPLY_STATUS_BUILD_RED: ApplyStatus
APPLY_STATUS_REVERTED: ApplyStatus
APPLY_STATUS_COMMITTED: ApplyStatus

class Operation(_message.Message):
    __slots__ = ("id", "kind", "from_path", "to_path", "payload", "resolves_conflict_ids")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    FROM_PATH_FIELD_NUMBER: _ClassVar[int]
    TO_PATH_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    RESOLVES_CONFLICT_IDS_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: OperationKind
    from_path: str
    to_path: str
    payload: bytes
    resolves_conflict_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[_Union[OperationKind, str]] = ..., from_path: _Optional[str] = ..., to_path: _Optional[str] = ..., payload: _Optional[bytes] = ..., resolves_conflict_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class Plan(_message.Message):
    __slots__ = ("id", "scenario", "domain", "operations", "conflict_ids", "planned_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    CONFLICT_IDS_FIELD_NUMBER: _ClassVar[int]
    PLANNED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario: str
    domain: str
    operations: _containers.RepeatedCompositeFieldContainer[Operation]
    conflict_ids: _containers.RepeatedScalarFieldContainer[str]
    planned_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., scenario: _Optional[str] = ..., domain: _Optional[str] = ..., operations: _Optional[_Iterable[_Union[Operation, _Mapping]]] = ..., conflict_ids: _Optional[_Iterable[str]] = ..., planned_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ApplyRun(_message.Message):
    __slots__ = ("id", "plan_id", "scenario", "domain", "status", "build_log", "started_at", "finished_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    BUILD_LOG_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    plan_id: str
    scenario: str
    domain: str
    status: ApplyStatus
    build_log: str
    started_at: _timestamp_pb2.Timestamp
    finished_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., plan_id: _Optional[str] = ..., scenario: _Optional[str] = ..., domain: _Optional[str] = ..., status: _Optional[_Union[ApplyStatus, str]] = ..., build_log: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., finished_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class BuildBaseline(_message.Message):
    __slots__ = ("scenario", "green", "toolchain", "log", "captured_at")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    GREEN_FIELD_NUMBER: _ClassVar[int]
    TOOLCHAIN_FIELD_NUMBER: _ClassVar[int]
    LOG_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    green: bool
    toolchain: str
    log: str
    captured_at: _timestamp_pb2.Timestamp
    def __init__(self, scenario: _Optional[str] = ..., green: _Optional[bool] = ..., toolchain: _Optional[str] = ..., log: _Optional[str] = ..., captured_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class WriteSuppressionRequest(_message.Message):
    __slots__ = ("scenario", "file", "id", "reason", "expires", "line")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    FILE_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_FIELD_NUMBER: _ClassVar[int]
    LINE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    file: str
    id: str
    reason: str
    expires: str
    line: int
    def __init__(self, scenario: _Optional[str] = ..., file: _Optional[str] = ..., id: _Optional[str] = ..., reason: _Optional[str] = ..., expires: _Optional[str] = ..., line: _Optional[int] = ...) -> None: ...

class WriteSuppressionResponse(_message.Message):
    __slots__ = ("file", "line", "marker")
    FILE_FIELD_NUMBER: _ClassVar[int]
    LINE_FIELD_NUMBER: _ClassVar[int]
    MARKER_FIELD_NUMBER: _ClassVar[int]
    file: str
    line: int
    marker: str
    def __init__(self, file: _Optional[str] = ..., line: _Optional[int] = ..., marker: _Optional[str] = ...) -> None: ...

class PlanApplyRequest(_message.Message):
    __slots__ = ("scenario", "domain", "conflict_ids", "dry_run")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    CONFLICT_IDS_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    domain: str
    conflict_ids: _containers.RepeatedScalarFieldContainer[str]
    dry_run: bool
    def __init__(self, scenario: _Optional[str] = ..., domain: _Optional[str] = ..., conflict_ids: _Optional[_Iterable[str]] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class PlanApplyResponse(_message.Message):
    __slots__ = ("plan", "dry_run")
    PLAN_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    plan: Plan
    dry_run: bool
    def __init__(self, plan: _Optional[_Union[Plan, _Mapping]] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class RunApplyRequest(_message.Message):
    __slots__ = ("plan_id", "acknowledge_v01_unimplemented")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    ACKNOWLEDGE_V01_UNIMPLEMENTED_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    acknowledge_v01_unimplemented: bool
    def __init__(self, plan_id: _Optional[str] = ..., acknowledge_v01_unimplemented: _Optional[bool] = ...) -> None: ...

class RunApplyResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: ApplyRun
    def __init__(self, run: _Optional[_Union[ApplyRun, _Mapping]] = ...) -> None: ...

class ListApplyHistoryRequest(_message.Message):
    __slots__ = ("scenario", "domain", "page_size", "page_token")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    domain: str
    page_size: int
    page_token: str
    def __init__(self, scenario: _Optional[str] = ..., domain: _Optional[str] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListApplyHistoryResponse(_message.Message):
    __slots__ = ("runs", "next_page_token")
    RUNS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    runs: _containers.RepeatedCompositeFieldContainer[ApplyRun]
    next_page_token: str
    def __init__(self, runs: _Optional[_Iterable[_Union[ApplyRun, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class GetBuildBaselineRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class GetBuildBaselineResponse(_message.Message):
    __slots__ = ("baseline",)
    BASELINE_FIELD_NUMBER: _ClassVar[int]
    baseline: BuildBaseline
    def __init__(self, baseline: _Optional[_Union[BuildBaseline, _Mapping]] = ...) -> None: ...

class PlanWithContext(_message.Message):
    __slots__ = ("plan", "conflicts")
    PLAN_FIELD_NUMBER: _ClassVar[int]
    CONFLICTS_FIELD_NUMBER: _ClassVar[int]
    plan: Plan
    conflicts: _containers.RepeatedCompositeFieldContainer[_shared_pb2.Conflict]
    def __init__(self, plan: _Optional[_Union[Plan, _Mapping]] = ..., conflicts: _Optional[_Iterable[_Union[_shared_pb2.Conflict, _Mapping]]] = ...) -> None: ...
