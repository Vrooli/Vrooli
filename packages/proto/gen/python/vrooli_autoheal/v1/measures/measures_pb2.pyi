import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetUptimeByCheckRequest(_message.Message):
    __slots__ = ("check_id", "window_hours")
    CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    WINDOW_HOURS_FIELD_NUMBER: _ClassVar[int]
    check_id: str
    window_hours: int
    def __init__(self, check_id: _Optional[str] = ..., window_hours: _Optional[int] = ...) -> None: ...

class UptimeByCheck(_message.Message):
    __slots__ = ("check_id", "uptime_percent", "total", "ok", "warning", "critical", "computed_at")
    CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    UPTIME_PERCENT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    OK_FIELD_NUMBER: _ClassVar[int]
    WARNING_FIELD_NUMBER: _ClassVar[int]
    CRITICAL_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    check_id: str
    uptime_percent: float
    total: int
    ok: int
    warning: int
    critical: int
    computed_at: _timestamp_pb2.Timestamp
    def __init__(self, check_id: _Optional[str] = ..., uptime_percent: _Optional[float] = ..., total: _Optional[int] = ..., ok: _Optional[int] = ..., warning: _Optional[int] = ..., critical: _Optional[int] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetUptimeByCheckResponse(_message.Message):
    __slots__ = ("uptime",)
    UPTIME_FIELD_NUMBER: _ClassVar[int]
    uptime: UptimeByCheck
    def __init__(self, uptime: _Optional[_Union[UptimeByCheck, _Mapping]] = ...) -> None: ...

class GetRestartCountRequest(_message.Message):
    __slots__ = ("window_hours",)
    WINDOW_HOURS_FIELD_NUMBER: _ClassVar[int]
    window_hours: int
    def __init__(self, window_hours: _Optional[int] = ...) -> None: ...

class RestartCount(_message.Message):
    __slots__ = ("count", "window_hours", "computed_at")
    COUNT_FIELD_NUMBER: _ClassVar[int]
    WINDOW_HOURS_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    count: int
    window_hours: int
    computed_at: _timestamp_pb2.Timestamp
    def __init__(self, count: _Optional[int] = ..., window_hours: _Optional[int] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetRestartCountResponse(_message.Message):
    __slots__ = ("restarts",)
    RESTARTS_FIELD_NUMBER: _ClassVar[int]
    restarts: RestartCount
    def __init__(self, restarts: _Optional[_Union[RestartCount, _Mapping]] = ...) -> None: ...

class GetHealOutcomesRequest(_message.Message):
    __slots__ = ("window_hours",)
    WINDOW_HOURS_FIELD_NUMBER: _ClassVar[int]
    window_hours: int
    def __init__(self, window_hours: _Optional[int] = ...) -> None: ...

class HealOutcomeCount(_message.Message):
    __slots__ = ("outcome", "count")
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    outcome: str
    count: int
    def __init__(self, outcome: _Optional[str] = ..., count: _Optional[int] = ...) -> None: ...

class GetHealOutcomesResponse(_message.Message):
    __slots__ = ("outcomes", "window_hours", "computed_at")
    OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    WINDOW_HOURS_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    outcomes: _containers.RepeatedCompositeFieldContainer[HealOutcomeCount]
    window_hours: int
    computed_at: _timestamp_pb2.Timestamp
    def __init__(self, outcomes: _Optional[_Iterable[_Union[HealOutcomeCount, _Mapping]]] = ..., window_hours: _Optional[int] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetCriticalCountRequest(_message.Message):
    __slots__ = ("window_hours",)
    WINDOW_HOURS_FIELD_NUMBER: _ClassVar[int]
    window_hours: int
    def __init__(self, window_hours: _Optional[int] = ...) -> None: ...

class CriticalCount(_message.Message):
    __slots__ = ("count", "window_hours", "computed_at")
    COUNT_FIELD_NUMBER: _ClassVar[int]
    WINDOW_HOURS_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    count: int
    window_hours: int
    computed_at: _timestamp_pb2.Timestamp
    def __init__(self, count: _Optional[int] = ..., window_hours: _Optional[int] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetCriticalCountResponse(_message.Message):
    __slots__ = ("critical",)
    CRITICAL_FIELD_NUMBER: _ClassVar[int]
    critical: CriticalCount
    def __init__(self, critical: _Optional[_Union[CriticalCount, _Mapping]] = ...) -> None: ...

class GetOutageSummaryRequest(_message.Message):
    __slots__ = ("member_id", "window_hours")
    MEMBER_ID_FIELD_NUMBER: _ClassVar[int]
    WINDOW_HOURS_FIELD_NUMBER: _ClassVar[int]
    member_id: str
    window_hours: int
    def __init__(self, member_id: _Optional[str] = ..., window_hours: _Optional[int] = ...) -> None: ...

class OutageSummary(_message.Message):
    __slots__ = ("member_id", "total_unavailable_seconds", "distinct_outage_count", "open_outage_count", "window_start", "window_end", "computed_at")
    MEMBER_ID_FIELD_NUMBER: _ClassVar[int]
    TOTAL_UNAVAILABLE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    DISTINCT_OUTAGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    OPEN_OUTAGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    WINDOW_START_FIELD_NUMBER: _ClassVar[int]
    WINDOW_END_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    member_id: str
    total_unavailable_seconds: float
    distinct_outage_count: int
    open_outage_count: int
    window_start: _timestamp_pb2.Timestamp
    window_end: _timestamp_pb2.Timestamp
    computed_at: _timestamp_pb2.Timestamp
    def __init__(self, member_id: _Optional[str] = ..., total_unavailable_seconds: _Optional[float] = ..., distinct_outage_count: _Optional[int] = ..., open_outage_count: _Optional[int] = ..., window_start: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., window_end: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetOutageSummaryResponse(_message.Message):
    __slots__ = ("outage",)
    OUTAGE_FIELD_NUMBER: _ClassVar[int]
    outage: OutageSummary
    def __init__(self, outage: _Optional[_Union[OutageSummary, _Mapping]] = ...) -> None: ...
