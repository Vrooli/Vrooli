import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class MonitorStatus(_message.Message):
    __slots__ = ("enabled", "interval_seconds", "in_flight", "last_run_id", "last_status", "last_started_at", "last_finished_at", "next_run_at", "green_streak", "updated_at")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    INTERVAL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    IN_FLIGHT_FIELD_NUMBER: _ClassVar[int]
    LAST_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    LAST_STATUS_FIELD_NUMBER: _ClassVar[int]
    LAST_STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    NEXT_RUN_AT_FIELD_NUMBER: _ClassVar[int]
    GREEN_STREAK_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    interval_seconds: int
    in_flight: bool
    last_run_id: str
    last_status: str
    last_started_at: _timestamp_pb2.Timestamp
    last_finished_at: _timestamp_pb2.Timestamp
    next_run_at: _timestamp_pb2.Timestamp
    green_streak: int
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, enabled: _Optional[bool] = ..., interval_seconds: _Optional[int] = ..., in_flight: _Optional[bool] = ..., last_run_id: _Optional[str] = ..., last_status: _Optional[str] = ..., last_started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_finished_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., next_run_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., green_streak: _Optional[int] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetMonitorStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetMonitorStatusResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: MonitorStatus
    def __init__(self, status: _Optional[_Union[MonitorStatus, _Mapping]] = ...) -> None: ...
