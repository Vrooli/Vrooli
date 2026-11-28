import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from vrooli_autoheal.v1.checks import checks_pb2 as _checks_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EventWeightedUptime(_message.Message):
    __slots__ = ("total_events", "ok_events", "warning_events", "critical_events", "uptime_percentage", "window_hours", "computed_at")
    TOTAL_EVENTS_FIELD_NUMBER: _ClassVar[int]
    OK_EVENTS_FIELD_NUMBER: _ClassVar[int]
    WARNING_EVENTS_FIELD_NUMBER: _ClassVar[int]
    CRITICAL_EVENTS_FIELD_NUMBER: _ClassVar[int]
    UPTIME_PERCENTAGE_FIELD_NUMBER: _ClassVar[int]
    WINDOW_HOURS_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    total_events: int
    ok_events: int
    warning_events: int
    critical_events: int
    uptime_percentage: float
    window_hours: int
    computed_at: _timestamp_pb2.Timestamp
    def __init__(self, total_events: _Optional[int] = ..., ok_events: _Optional[int] = ..., warning_events: _Optional[int] = ..., critical_events: _Optional[int] = ..., uptime_percentage: _Optional[float] = ..., window_hours: _Optional[int] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class PerCheckTrend(_message.Message):
    __slots__ = ("check_id", "total", "ok", "warning", "critical", "uptime_percent", "current_status", "recent_statuses", "last_checked")
    CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    OK_FIELD_NUMBER: _ClassVar[int]
    WARNING_FIELD_NUMBER: _ClassVar[int]
    CRITICAL_FIELD_NUMBER: _ClassVar[int]
    UPTIME_PERCENT_FIELD_NUMBER: _ClassVar[int]
    CURRENT_STATUS_FIELD_NUMBER: _ClassVar[int]
    RECENT_STATUSES_FIELD_NUMBER: _ClassVar[int]
    LAST_CHECKED_FIELD_NUMBER: _ClassVar[int]
    check_id: str
    total: int
    ok: int
    warning: int
    critical: int
    uptime_percent: float
    current_status: _checks_pb2.CheckStatus
    recent_statuses: _containers.RepeatedScalarFieldContainer[_checks_pb2.CheckStatus]
    last_checked: _timestamp_pb2.Timestamp
    def __init__(self, check_id: _Optional[str] = ..., total: _Optional[int] = ..., ok: _Optional[int] = ..., warning: _Optional[int] = ..., critical: _Optional[int] = ..., uptime_percent: _Optional[float] = ..., current_status: _Optional[_Union[_checks_pb2.CheckStatus, str]] = ..., recent_statuses: _Optional[_Iterable[_Union[_checks_pb2.CheckStatus, str]]] = ..., last_checked: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ActionHistoryEntry(_message.Message):
    __slots__ = ("id", "check_id", "action_id", "success", "timed_out", "message", "error", "duration_ms", "observed_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_ID_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    TIMED_OUT_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    id: int
    check_id: str
    action_id: str
    success: bool
    timed_out: bool
    message: str
    error: str
    duration_ms: int
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[int] = ..., check_id: _Optional[str] = ..., action_id: _Optional[str] = ..., success: _Optional[bool] = ..., timed_out: _Optional[bool] = ..., message: _Optional[str] = ..., error: _Optional[str] = ..., duration_ms: _Optional[int] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Transition(_message.Message):
    __slots__ = ("check_id", "from_status", "to_status", "message", "observed_at")
    CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    FROM_STATUS_FIELD_NUMBER: _ClassVar[int]
    TO_STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    check_id: str
    from_status: _checks_pb2.CheckStatus
    to_status: _checks_pb2.CheckStatus
    message: str
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, check_id: _Optional[str] = ..., from_status: _Optional[_Union[_checks_pb2.CheckStatus, str]] = ..., to_status: _Optional[_Union[_checks_pb2.CheckStatus, str]] = ..., message: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetEventWeightedUptimeRequest(_message.Message):
    __slots__ = ("window_hours",)
    WINDOW_HOURS_FIELD_NUMBER: _ClassVar[int]
    window_hours: int
    def __init__(self, window_hours: _Optional[int] = ...) -> None: ...

class GetEventWeightedUptimeResponse(_message.Message):
    __slots__ = ("uptime",)
    UPTIME_FIELD_NUMBER: _ClassVar[int]
    uptime: EventWeightedUptime
    def __init__(self, uptime: _Optional[_Union[EventWeightedUptime, _Mapping]] = ...) -> None: ...

class GetPerCheckTrendsRequest(_message.Message):
    __slots__ = ("window_hours",)
    WINDOW_HOURS_FIELD_NUMBER: _ClassVar[int]
    window_hours: int
    def __init__(self, window_hours: _Optional[int] = ...) -> None: ...

class GetPerCheckTrendsResponse(_message.Message):
    __slots__ = ("trends", "window_hours")
    TRENDS_FIELD_NUMBER: _ClassVar[int]
    WINDOW_HOURS_FIELD_NUMBER: _ClassVar[int]
    trends: _containers.RepeatedCompositeFieldContainer[PerCheckTrend]
    window_hours: int
    def __init__(self, trends: _Optional[_Iterable[_Union[PerCheckTrend, _Mapping]]] = ..., window_hours: _Optional[int] = ...) -> None: ...

class GetHistoryRequest(_message.Message):
    __slots__ = ("check_id", "limit")
    CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    check_id: str
    limit: int
    def __init__(self, check_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class GetHistoryResponse(_message.Message):
    __slots__ = ("entries",)
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[ActionHistoryEntry]
    def __init__(self, entries: _Optional[_Iterable[_Union[ActionHistoryEntry, _Mapping]]] = ...) -> None: ...

class GetTransitionsRequest(_message.Message):
    __slots__ = ("window_hours", "limit")
    WINDOW_HOURS_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    window_hours: int
    limit: int
    def __init__(self, window_hours: _Optional[int] = ..., limit: _Optional[int] = ...) -> None: ...

class GetTransitionsResponse(_message.Message):
    __slots__ = ("transitions",)
    TRANSITIONS_FIELD_NUMBER: _ClassVar[int]
    transitions: _containers.RepeatedCompositeFieldContainer[Transition]
    def __init__(self, transitions: _Optional[_Iterable[_Union[Transition, _Mapping]]] = ...) -> None: ...
