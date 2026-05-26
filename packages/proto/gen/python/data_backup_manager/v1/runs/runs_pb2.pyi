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
    __slots__ = ("target_id", "destination_id", "status", "snapshot_id", "bytes", "error", "started_at", "finished_at")
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    BYTES_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    target_id: str
    destination_id: str
    status: TargetOutcomeStatus
    snapshot_id: str
    bytes: int
    error: str
    started_at: _timestamp_pb2.Timestamp
    finished_at: _timestamp_pb2.Timestamp
    def __init__(self, target_id: _Optional[str] = ..., destination_id: _Optional[str] = ..., status: _Optional[_Union[TargetOutcomeStatus, str]] = ..., snapshot_id: _Optional[str] = ..., bytes: _Optional[int] = ..., error: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., finished_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Run(_message.Message):
    __slots__ = ("id", "plan_id", "trigger", "status", "started_at", "finished_at", "outcomes")
    ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    id: str
    plan_id: str
    trigger: TriggerSource
    status: RunStatus
    started_at: _timestamp_pb2.Timestamp
    finished_at: _timestamp_pb2.Timestamp
    outcomes: _containers.RepeatedCompositeFieldContainer[TargetOutcome]
    def __init__(self, id: _Optional[str] = ..., plan_id: _Optional[str] = ..., trigger: _Optional[_Union[TriggerSource, str]] = ..., status: _Optional[_Union[RunStatus, str]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., finished_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., outcomes: _Optional[_Iterable[_Union[TargetOutcome, _Mapping]]] = ...) -> None: ...

class TargetStatus(_message.Message):
    __slots__ = ("target_id", "last_success_at", "last_run_status", "last_run_at")
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    LAST_SUCCESS_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_RUN_STATUS_FIELD_NUMBER: _ClassVar[int]
    LAST_RUN_AT_FIELD_NUMBER: _ClassVar[int]
    target_id: str
    last_success_at: _timestamp_pb2.Timestamp
    last_run_status: RunStatus
    last_run_at: _timestamp_pb2.Timestamp
    def __init__(self, target_id: _Optional[str] = ..., last_success_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_run_status: _Optional[_Union[RunStatus, str]] = ..., last_run_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

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
