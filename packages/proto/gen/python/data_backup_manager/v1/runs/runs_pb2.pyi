import datetime

from buf.validate import validate_pb2 as _validate_pb2
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
    RUN_STATUS_PENDING: _ClassVar[RunStatus]
    RUN_STATUS_CAPTURING: _ClassVar[RunStatus]
    RUN_STATUS_SNAPSHOTTING: _ClassVar[RunStatus]
    RUN_STATUS_COMPLETED: _ClassVar[RunStatus]
    RUN_STATUS_PARTIAL_FAILED: _ClassVar[RunStatus]
    RUN_STATUS_FAILED: _ClassVar[RunStatus]

class TriggerSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TRIGGER_SOURCE_UNSPECIFIED: _ClassVar[TriggerSource]
    TRIGGER_SOURCE_SCHEDULER: _ClassVar[TriggerSource]
    TRIGGER_SOURCE_MANUAL: _ClassVar[TriggerSource]

class TargetOutcomeStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TARGET_OUTCOME_STATUS_UNSPECIFIED: _ClassVar[TargetOutcomeStatus]
    TARGET_OUTCOME_STATUS_SUCCEEDED: _ClassVar[TargetOutcomeStatus]
    TARGET_OUTCOME_STATUS_FAILED: _ClassVar[TargetOutcomeStatus]
    TARGET_OUTCOME_STATUS_BLOCKED: _ClassVar[TargetOutcomeStatus]
RUN_STATUS_UNSPECIFIED: RunStatus
RUN_STATUS_PENDING: RunStatus
RUN_STATUS_CAPTURING: RunStatus
RUN_STATUS_SNAPSHOTTING: RunStatus
RUN_STATUS_COMPLETED: RunStatus
RUN_STATUS_PARTIAL_FAILED: RunStatus
RUN_STATUS_FAILED: RunStatus
TRIGGER_SOURCE_UNSPECIFIED: TriggerSource
TRIGGER_SOURCE_SCHEDULER: TriggerSource
TRIGGER_SOURCE_MANUAL: TriggerSource
TARGET_OUTCOME_STATUS_UNSPECIFIED: TargetOutcomeStatus
TARGET_OUTCOME_STATUS_SUCCEEDED: TargetOutcomeStatus
TARGET_OUTCOME_STATUS_FAILED: TargetOutcomeStatus
TARGET_OUTCOME_STATUS_BLOCKED: TargetOutcomeStatus

class TargetOutcome(_message.Message):
    __slots__ = ("target_id", "destination_id", "status", "snapshot_id", "bytes", "error", "started_at", "finished_at", "failure_code", "failure_category", "warning")
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    BYTES_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    FAILURE_CODE_FIELD_NUMBER: _ClassVar[int]
    FAILURE_CATEGORY_FIELD_NUMBER: _ClassVar[int]
    WARNING_FIELD_NUMBER: _ClassVar[int]
    target_id: str
    destination_id: str
    status: TargetOutcomeStatus
    snapshot_id: str
    bytes: int
    error: str
    started_at: _timestamp_pb2.Timestamp
    finished_at: _timestamp_pb2.Timestamp
    failure_code: str
    failure_category: str
    warning: str
    def __init__(self, target_id: _Optional[str] = ..., destination_id: _Optional[str] = ..., status: _Optional[_Union[TargetOutcomeStatus, str]] = ..., snapshot_id: _Optional[str] = ..., bytes: _Optional[int] = ..., error: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., finished_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., failure_code: _Optional[str] = ..., failure_category: _Optional[str] = ..., warning: _Optional[str] = ...) -> None: ...

class FailureCause(_message.Message):
    __slots__ = ("code", "category", "scope", "message", "next_action", "destination_id", "target_ids", "first_observed", "last_observed", "last_known_good", "retryable", "retry_after_seconds")
    CODE_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_IDS_FIELD_NUMBER: _ClassVar[int]
    FIRST_OBSERVED_FIELD_NUMBER: _ClassVar[int]
    LAST_OBSERVED_FIELD_NUMBER: _ClassVar[int]
    LAST_KNOWN_GOOD_FIELD_NUMBER: _ClassVar[int]
    RETRYABLE_FIELD_NUMBER: _ClassVar[int]
    RETRY_AFTER_SECONDS_FIELD_NUMBER: _ClassVar[int]
    code: str
    category: str
    scope: str
    message: str
    next_action: str
    destination_id: str
    target_ids: _containers.RepeatedScalarFieldContainer[str]
    first_observed: _timestamp_pb2.Timestamp
    last_observed: _timestamp_pb2.Timestamp
    last_known_good: _timestamp_pb2.Timestamp
    retryable: bool
    retry_after_seconds: int
    def __init__(self, code: _Optional[str] = ..., category: _Optional[str] = ..., scope: _Optional[str] = ..., message: _Optional[str] = ..., next_action: _Optional[str] = ..., destination_id: _Optional[str] = ..., target_ids: _Optional[_Iterable[str]] = ..., first_observed: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_observed: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_known_good: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., retryable: _Optional[bool] = ..., retry_after_seconds: _Optional[int] = ...) -> None: ...

class Run(_message.Message):
    __slots__ = ("id", "plan_id", "trigger", "status", "started_at", "finished_at", "outcomes", "error", "failure_code", "failure_category", "next_action", "preflight_incidents", "updated_at", "physical_bytes")
    ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    FAILURE_CODE_FIELD_NUMBER: _ClassVar[int]
    FAILURE_CATEGORY_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    PREFLIGHT_INCIDENTS_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    PHYSICAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    id: str
    plan_id: str
    trigger: TriggerSource
    status: RunStatus
    started_at: _timestamp_pb2.Timestamp
    finished_at: _timestamp_pb2.Timestamp
    outcomes: _containers.RepeatedCompositeFieldContainer[TargetOutcome]
    error: str
    failure_code: str
    failure_category: str
    next_action: str
    preflight_incidents: _containers.RepeatedCompositeFieldContainer[FailureCause]
    updated_at: _timestamp_pb2.Timestamp
    physical_bytes: int
    def __init__(self, id: _Optional[str] = ..., plan_id: _Optional[str] = ..., trigger: _Optional[_Union[TriggerSource, str]] = ..., status: _Optional[_Union[RunStatus, str]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., finished_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., outcomes: _Optional[_Iterable[_Union[TargetOutcome, _Mapping]]] = ..., error: _Optional[str] = ..., failure_code: _Optional[str] = ..., failure_category: _Optional[str] = ..., next_action: _Optional[str] = ..., preflight_incidents: _Optional[_Iterable[_Union[FailureCause, _Mapping]]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., physical_bytes: _Optional[int] = ...) -> None: ...

class TargetStatus(_message.Message):
    __slots__ = ("target_id", "last_success_at", "last_run_status", "last_run_at", "last_verified_at", "last_verified_snapshot_id", "overdue", "last_success_age_seconds", "next_scheduled_at")
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    LAST_SUCCESS_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_RUN_STATUS_FIELD_NUMBER: _ClassVar[int]
    LAST_RUN_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_VERIFIED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_VERIFIED_SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    OVERDUE_FIELD_NUMBER: _ClassVar[int]
    LAST_SUCCESS_AGE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    NEXT_SCHEDULED_AT_FIELD_NUMBER: _ClassVar[int]
    target_id: str
    last_success_at: _timestamp_pb2.Timestamp
    last_run_status: RunStatus
    last_run_at: _timestamp_pb2.Timestamp
    last_verified_at: _timestamp_pb2.Timestamp
    last_verified_snapshot_id: str
    overdue: bool
    last_success_age_seconds: int
    next_scheduled_at: _timestamp_pb2.Timestamp
    def __init__(self, target_id: _Optional[str] = ..., last_success_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_run_status: _Optional[_Union[RunStatus, str]] = ..., last_run_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_verified_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_verified_snapshot_id: _Optional[str] = ..., overdue: _Optional[bool] = ..., last_success_age_seconds: _Optional[int] = ..., next_scheduled_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class TriggerRunRequest(_message.Message):
    __slots__ = ("plan_id",)
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    def __init__(self, plan_id: _Optional[str] = ...) -> None: ...

class TriggerRunResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: Run
    def __init__(self, run: _Optional[_Union[Run, _Mapping]] = ...) -> None: ...

class PreflightRunRequest(_message.Message):
    __slots__ = ("plan_id",)
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    def __init__(self, plan_id: _Optional[str] = ...) -> None: ...

class PreflightRunResponse(_message.Message):
    __slots__ = ("ready", "checked_at", "incidents")
    READY_FIELD_NUMBER: _ClassVar[int]
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    INCIDENTS_FIELD_NUMBER: _ClassVar[int]
    ready: bool
    checked_at: _timestamp_pb2.Timestamp
    incidents: _containers.RepeatedCompositeFieldContainer[FailureCause]
    def __init__(self, ready: _Optional[bool] = ..., checked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., incidents: _Optional[_Iterable[_Union[FailureCause, _Mapping]]] = ...) -> None: ...

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

class ListRunsRequest(_message.Message):
    __slots__ = ("plan_id", "page_size", "page_token")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    page_size: int
    page_token: str
    def __init__(self, plan_id: _Optional[str] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListRunsResponse(_message.Message):
    __slots__ = ("runs", "next_page_token")
    RUNS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    runs: _containers.RepeatedCompositeFieldContainer[Run]
    next_page_token: str
    def __init__(self, runs: _Optional[_Iterable[_Union[Run, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class ListTargetStatusRequest(_message.Message):
    __slots__ = ("owner",)
    OWNER_FIELD_NUMBER: _ClassVar[int]
    owner: str
    def __init__(self, owner: _Optional[str] = ...) -> None: ...

class ListTargetStatusResponse(_message.Message):
    __slots__ = ("statuses",)
    STATUSES_FIELD_NUMBER: _ClassVar[int]
    statuses: _containers.RepeatedCompositeFieldContainer[TargetStatus]
    def __init__(self, statuses: _Optional[_Iterable[_Union[TargetStatus, _Mapping]]] = ...) -> None: ...

class SnapshotEntry(_message.Message):
    __slots__ = ("path", "size_bytes", "is_dir")
    PATH_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    IS_DIR_FIELD_NUMBER: _ClassVar[int]
    path: str
    size_bytes: int
    is_dir: bool
    def __init__(self, path: _Optional[str] = ..., size_bytes: _Optional[int] = ..., is_dir: _Optional[bool] = ...) -> None: ...

class BrowseSnapshotRequest(_message.Message):
    __slots__ = ("destination_id", "snapshot_id", "path")
    DESTINATION_ID_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    destination_id: str
    snapshot_id: str
    path: str
    def __init__(self, destination_id: _Optional[str] = ..., snapshot_id: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class BrowseSnapshotResponse(_message.Message):
    __slots__ = ("entries",)
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[SnapshotEntry]
    def __init__(self, entries: _Optional[_Iterable[_Union[SnapshotEntry, _Mapping]]] = ...) -> None: ...

class GetRunStatsRequest(_message.Message):
    __slots__ = ("plan_id",)
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    def __init__(self, plan_id: _Optional[str] = ...) -> None: ...

class RunStats(_message.Message):
    __slots__ = ("total_runs", "completed", "partial_failed", "failed", "success_rate", "p50_duration_ms", "p95_duration_ms", "total_bytes", "avg_bytes_per_run", "avg_throughput_bytes_per_sec", "window", "total_physical_bytes", "dedup_ratio")
    TOTAL_RUNS_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_FIELD_NUMBER: _ClassVar[int]
    PARTIAL_FAILED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_RATE_FIELD_NUMBER: _ClassVar[int]
    P50_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    P95_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    AVG_BYTES_PER_RUN_FIELD_NUMBER: _ClassVar[int]
    AVG_THROUGHPUT_BYTES_PER_SEC_FIELD_NUMBER: _ClassVar[int]
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    TOTAL_PHYSICAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    DEDUP_RATIO_FIELD_NUMBER: _ClassVar[int]
    total_runs: int
    completed: int
    partial_failed: int
    failed: int
    success_rate: float
    p50_duration_ms: int
    p95_duration_ms: int
    total_bytes: int
    avg_bytes_per_run: int
    avg_throughput_bytes_per_sec: float
    window: int
    total_physical_bytes: int
    dedup_ratio: float
    def __init__(self, total_runs: _Optional[int] = ..., completed: _Optional[int] = ..., partial_failed: _Optional[int] = ..., failed: _Optional[int] = ..., success_rate: _Optional[float] = ..., p50_duration_ms: _Optional[int] = ..., p95_duration_ms: _Optional[int] = ..., total_bytes: _Optional[int] = ..., avg_bytes_per_run: _Optional[int] = ..., avg_throughput_bytes_per_sec: _Optional[float] = ..., window: _Optional[int] = ..., total_physical_bytes: _Optional[int] = ..., dedup_ratio: _Optional[float] = ...) -> None: ...

class GetRunStatsResponse(_message.Message):
    __slots__ = ("stats",)
    STATS_FIELD_NUMBER: _ClassVar[int]
    stats: RunStats
    def __init__(self, stats: _Optional[_Union[RunStats, _Mapping]] = ...) -> None: ...
