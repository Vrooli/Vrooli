import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PlanningProfile(_message.Message):
    __slots__ = ("id", "timezone", "week_start", "daily_capacity_minutes", "reserve_minutes", "focus_session_minutes", "revision", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    TIMEZONE_FIELD_NUMBER: _ClassVar[int]
    WEEK_START_FIELD_NUMBER: _ClassVar[int]
    DAILY_CAPACITY_MINUTES_FIELD_NUMBER: _ClassVar[int]
    RESERVE_MINUTES_FIELD_NUMBER: _ClassVar[int]
    FOCUS_SESSION_MINUTES_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    timezone: str
    week_start: str
    daily_capacity_minutes: int
    reserve_minutes: int
    focus_session_minutes: int
    revision: int
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., timezone: _Optional[str] = ..., week_start: _Optional[str] = ..., daily_capacity_minutes: _Optional[int] = ..., reserve_minutes: _Optional[int] = ..., focus_session_minutes: _Optional[int] = ..., revision: _Optional[int] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetProfileRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetProfileResponse(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: PlanningProfile
    def __init__(self, profile: _Optional[_Union[PlanningProfile, _Mapping]] = ...) -> None: ...

class UpdateProfileRequest(_message.Message):
    __slots__ = ("timezone", "week_start", "daily_capacity_minutes", "reserve_minutes", "focus_session_minutes", "expected_revision")
    TIMEZONE_FIELD_NUMBER: _ClassVar[int]
    WEEK_START_FIELD_NUMBER: _ClassVar[int]
    DAILY_CAPACITY_MINUTES_FIELD_NUMBER: _ClassVar[int]
    RESERVE_MINUTES_FIELD_NUMBER: _ClassVar[int]
    FOCUS_SESSION_MINUTES_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    timezone: str
    week_start: str
    daily_capacity_minutes: int
    reserve_minutes: int
    focus_session_minutes: int
    expected_revision: int
    def __init__(self, timezone: _Optional[str] = ..., week_start: _Optional[str] = ..., daily_capacity_minutes: _Optional[int] = ..., reserve_minutes: _Optional[int] = ..., focus_session_minutes: _Optional[int] = ..., expected_revision: _Optional[int] = ...) -> None: ...

class UpdateProfileResponse(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: PlanningProfile
    def __init__(self, profile: _Optional[_Union[PlanningProfile, _Mapping]] = ...) -> None: ...

class AvailabilityWindow(_message.Message):
    __slots__ = ("id", "weekday", "start_minute", "end_minute", "timezone", "effective_start_date", "effective_end_date", "priority", "revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    WEEKDAY_FIELD_NUMBER: _ClassVar[int]
    START_MINUTE_FIELD_NUMBER: _ClassVar[int]
    END_MINUTE_FIELD_NUMBER: _ClassVar[int]
    TIMEZONE_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_START_DATE_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_END_DATE_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    weekday: int
    start_minute: int
    end_minute: int
    timezone: str
    effective_start_date: str
    effective_end_date: str
    priority: int
    revision: int
    def __init__(self, id: _Optional[str] = ..., weekday: _Optional[int] = ..., start_minute: _Optional[int] = ..., end_minute: _Optional[int] = ..., timezone: _Optional[str] = ..., effective_start_date: _Optional[str] = ..., effective_end_date: _Optional[str] = ..., priority: _Optional[int] = ..., revision: _Optional[int] = ...) -> None: ...

class AvailabilityException(_message.Message):
    __slots__ = ("id", "date", "start_minute", "end_minute", "kind", "reason", "revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    DATE_FIELD_NUMBER: _ClassVar[int]
    START_MINUTE_FIELD_NUMBER: _ClassVar[int]
    END_MINUTE_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    date: str
    start_minute: int
    end_minute: int
    kind: str
    reason: str
    revision: int
    def __init__(self, id: _Optional[str] = ..., date: _Optional[str] = ..., start_minute: _Optional[int] = ..., end_minute: _Optional[int] = ..., kind: _Optional[str] = ..., reason: _Optional[str] = ..., revision: _Optional[int] = ...) -> None: ...

class ListAvailabilityRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListAvailabilityResponse(_message.Message):
    __slots__ = ("windows", "exceptions", "revision")
    WINDOWS_FIELD_NUMBER: _ClassVar[int]
    EXCEPTIONS_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    windows: _containers.RepeatedCompositeFieldContainer[AvailabilityWindow]
    exceptions: _containers.RepeatedCompositeFieldContainer[AvailabilityException]
    revision: int
    def __init__(self, windows: _Optional[_Iterable[_Union[AvailabilityWindow, _Mapping]]] = ..., exceptions: _Optional[_Iterable[_Union[AvailabilityException, _Mapping]]] = ..., revision: _Optional[int] = ...) -> None: ...

class ReplaceAvailabilityRequest(_message.Message):
    __slots__ = ("windows", "exceptions", "expected_revision")
    WINDOWS_FIELD_NUMBER: _ClassVar[int]
    EXCEPTIONS_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    windows: _containers.RepeatedCompositeFieldContainer[AvailabilityWindow]
    exceptions: _containers.RepeatedCompositeFieldContainer[AvailabilityException]
    expected_revision: int
    def __init__(self, windows: _Optional[_Iterable[_Union[AvailabilityWindow, _Mapping]]] = ..., exceptions: _Optional[_Iterable[_Union[AvailabilityException, _Mapping]]] = ..., expected_revision: _Optional[int] = ...) -> None: ...

class ReplaceAvailabilityResponse(_message.Message):
    __slots__ = ("windows", "exceptions", "revision")
    WINDOWS_FIELD_NUMBER: _ClassVar[int]
    EXCEPTIONS_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    windows: _containers.RepeatedCompositeFieldContainer[AvailabilityWindow]
    exceptions: _containers.RepeatedCompositeFieldContainer[AvailabilityException]
    revision: int
    def __init__(self, windows: _Optional[_Iterable[_Union[AvailabilityWindow, _Mapping]]] = ..., exceptions: _Optional[_Iterable[_Union[AvailabilityException, _Mapping]]] = ..., revision: _Optional[int] = ...) -> None: ...
