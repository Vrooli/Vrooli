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

class DrillStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DRILL_STATUS_UNSPECIFIED: _ClassVar[DrillStatus]
    DRILL_STATUS_REQUESTED: _ClassVar[DrillStatus]
    DRILL_STATUS_RUNNING: _ClassVar[DrillStatus]
    DRILL_STATUS_VERIFIED: _ClassVar[DrillStatus]
    DRILL_STATUS_FAILED: _ClassVar[DrillStatus]
DRILL_STATUS_UNSPECIFIED: DrillStatus
DRILL_STATUS_REQUESTED: DrillStatus
DRILL_STATUS_RUNNING: DrillStatus
DRILL_STATUS_VERIFIED: DrillStatus
DRILL_STATUS_FAILED: DrillStatus

class RecoveryDrill(_message.Message):
    __slots__ = ("id", "plan_id", "target_id", "destination_id", "snapshot_id", "restore_id", "status", "scheduled", "error", "next_action", "requested_at", "started_at", "finished_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_ID_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    RESTORE_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SCHEDULED_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_AT_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    plan_id: str
    target_id: str
    destination_id: str
    snapshot_id: str
    restore_id: str
    status: DrillStatus
    scheduled: bool
    error: str
    next_action: str
    requested_at: _timestamp_pb2.Timestamp
    started_at: _timestamp_pb2.Timestamp
    finished_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., plan_id: _Optional[str] = ..., target_id: _Optional[str] = ..., destination_id: _Optional[str] = ..., snapshot_id: _Optional[str] = ..., restore_id: _Optional[str] = ..., status: _Optional[_Union[DrillStatus, str]] = ..., scheduled: _Optional[bool] = ..., error: _Optional[str] = ..., next_action: _Optional[str] = ..., requested_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., finished_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class PreviewDrillRequest(_message.Message):
    __slots__ = ("plan_id", "target_id", "destination_id")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_ID_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    target_id: str
    destination_id: str
    def __init__(self, plan_id: _Optional[str] = ..., target_id: _Optional[str] = ..., destination_id: _Optional[str] = ...) -> None: ...

class PreviewDrillResponse(_message.Message):
    __slots__ = ("eligible", "plan_id", "target_id", "destination_id", "snapshot_id", "warnings", "reason")
    ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_ID_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    eligible: bool
    plan_id: str
    target_id: str
    destination_id: str
    snapshot_id: str
    warnings: _containers.RepeatedScalarFieldContainer[str]
    reason: str
    def __init__(self, eligible: _Optional[bool] = ..., plan_id: _Optional[str] = ..., target_id: _Optional[str] = ..., destination_id: _Optional[str] = ..., snapshot_id: _Optional[str] = ..., warnings: _Optional[_Iterable[str]] = ..., reason: _Optional[str] = ...) -> None: ...

class RunDrillRequest(_message.Message):
    __slots__ = ("plan_id", "target_id", "destination_id", "idempotency_key", "scheduled")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    SCHEDULED_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    target_id: str
    destination_id: str
    idempotency_key: str
    scheduled: bool
    def __init__(self, plan_id: _Optional[str] = ..., target_id: _Optional[str] = ..., destination_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., scheduled: _Optional[bool] = ...) -> None: ...

class RunDrillResponse(_message.Message):
    __slots__ = ("drill",)
    DRILL_FIELD_NUMBER: _ClassVar[int]
    drill: RecoveryDrill
    def __init__(self, drill: _Optional[_Union[RecoveryDrill, _Mapping]] = ...) -> None: ...

class GetDrillRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetDrillResponse(_message.Message):
    __slots__ = ("drill",)
    DRILL_FIELD_NUMBER: _ClassVar[int]
    drill: RecoveryDrill
    def __init__(self, drill: _Optional[_Union[RecoveryDrill, _Mapping]] = ...) -> None: ...

class ListDrillsRequest(_message.Message):
    __slots__ = ("plan_id", "target_id", "page_size", "page_token")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    target_id: str
    page_size: int
    page_token: str
    def __init__(self, plan_id: _Optional[str] = ..., target_id: _Optional[str] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListDrillsResponse(_message.Message):
    __slots__ = ("drills", "next_page_token")
    DRILLS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    drills: _containers.RepeatedCompositeFieldContainer[RecoveryDrill]
    next_page_token: str
    def __init__(self, drills: _Optional[_Iterable[_Union[RecoveryDrill, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...
