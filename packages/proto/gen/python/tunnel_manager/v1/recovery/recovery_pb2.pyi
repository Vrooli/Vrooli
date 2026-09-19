import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RecoveryStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RECOVERY_STATUS_UNSPECIFIED: _ClassVar[RecoveryStatus]
    RECOVERY_STATUS_IDLE: _ClassVar[RecoveryStatus]
    RECOVERY_STATUS_MONITORING: _ClassVar[RecoveryStatus]
    RECOVERY_STATUS_RECOVERING: _ClassVar[RecoveryStatus]
    RECOVERY_STATUS_CIRCUIT_OPEN: _ClassVar[RecoveryStatus]

class EventOutcome(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EVENT_OUTCOME_UNSPECIFIED: _ClassVar[EventOutcome]
    EVENT_OUTCOME_SUCCESS: _ClassVar[EventOutcome]
    EVENT_OUTCOME_FAILURE: _ClassVar[EventOutcome]
    EVENT_OUTCOME_SKIPPED: _ClassVar[EventOutcome]
RECOVERY_STATUS_UNSPECIFIED: RecoveryStatus
RECOVERY_STATUS_IDLE: RecoveryStatus
RECOVERY_STATUS_MONITORING: RecoveryStatus
RECOVERY_STATUS_RECOVERING: RecoveryStatus
RECOVERY_STATUS_CIRCUIT_OPEN: RecoveryStatus
EVENT_OUTCOME_UNSPECIFIED: EventOutcome
EVENT_OUTCOME_SUCCESS: EventOutcome
EVENT_OUTCOME_FAILURE: EventOutcome
EVENT_OUTCOME_SKIPPED: EventOutcome

class RecoveryState(_message.Message):
    __slots__ = ("status", "consec_failures", "backoff_level", "failed_recoveries", "circuit_open", "last_check", "last_recovery", "next_retry_after")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CONSEC_FAILURES_FIELD_NUMBER: _ClassVar[int]
    BACKOFF_LEVEL_FIELD_NUMBER: _ClassVar[int]
    FAILED_RECOVERIES_FIELD_NUMBER: _ClassVar[int]
    CIRCUIT_OPEN_FIELD_NUMBER: _ClassVar[int]
    LAST_CHECK_FIELD_NUMBER: _ClassVar[int]
    LAST_RECOVERY_FIELD_NUMBER: _ClassVar[int]
    NEXT_RETRY_AFTER_FIELD_NUMBER: _ClassVar[int]
    status: RecoveryStatus
    consec_failures: int
    backoff_level: int
    failed_recoveries: int
    circuit_open: bool
    last_check: _timestamp_pb2.Timestamp
    last_recovery: _timestamp_pb2.Timestamp
    next_retry_after: _timestamp_pb2.Timestamp
    def __init__(self, status: _Optional[_Union[RecoveryStatus, str]] = ..., consec_failures: _Optional[int] = ..., backoff_level: _Optional[int] = ..., failed_recoveries: _Optional[int] = ..., circuit_open: _Optional[bool] = ..., last_check: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_recovery: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., next_retry_after: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RecoveryEvent(_message.Message):
    __slots__ = ("id", "trigger", "action", "outcome", "details", "attempt", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    ATTEMPT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    trigger: str
    action: str
    outcome: EventOutcome
    details: str
    attempt: int
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., trigger: _Optional[str] = ..., action: _Optional[str] = ..., outcome: _Optional[_Union[EventOutcome, str]] = ..., details: _Optional[str] = ..., attempt: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetStateRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetStateResponse(_message.Message):
    __slots__ = ("state",)
    STATE_FIELD_NUMBER: _ClassVar[int]
    state: RecoveryState
    def __init__(self, state: _Optional[_Union[RecoveryState, _Mapping]] = ...) -> None: ...

class ListEventsRequest(_message.Message):
    __slots__ = ("limit",)
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    def __init__(self, limit: _Optional[int] = ...) -> None: ...

class ListEventsResponse(_message.Message):
    __slots__ = ("events",)
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    events: _containers.RepeatedCompositeFieldContainer[RecoveryEvent]
    def __init__(self, events: _Optional[_Iterable[_Union[RecoveryEvent, _Mapping]]] = ...) -> None: ...

class RecoverRequest(_message.Message):
    __slots__ = ("force",)
    FORCE_FIELD_NUMBER: _ClassVar[int]
    force: bool
    def __init__(self, force: _Optional[bool] = ...) -> None: ...

class RecoverResponse(_message.Message):
    __slots__ = ("outcome", "event")
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    EVENT_FIELD_NUMBER: _ClassVar[int]
    outcome: EventOutcome
    event: RecoveryEvent
    def __init__(self, outcome: _Optional[_Union[EventOutcome, str]] = ..., event: _Optional[_Union[RecoveryEvent, _Mapping]] = ...) -> None: ...
