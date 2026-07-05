from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DependencyStatus(_message.Message):
    __slots__ = ("connected", "latency_ms", "error", "database")
    CONNECTED_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    DATABASE_FIELD_NUMBER: _ClassVar[int]
    connected: bool
    latency_ms: float
    error: str
    database: str
    def __init__(self, connected: _Optional[bool] = ..., latency_ms: _Optional[float] = ..., error: _Optional[str] = ..., database: _Optional[str] = ...) -> None: ...

class Response(_message.Message):
    __slots__ = ("status", "service", "timestamp", "readiness", "version", "uptime_seconds", "dependencies")
    class DependenciesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: DependencyStatus
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[DependencyStatus, _Mapping]] = ...) -> None: ...
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SERVICE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    READINESS_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    UPTIME_SECONDS_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    status: str
    service: str
    timestamp: str
    readiness: bool
    version: str
    uptime_seconds: float
    dependencies: _containers.MessageMap[str, DependencyStatus]
    def __init__(self, status: _Optional[str] = ..., service: _Optional[str] = ..., timestamp: _Optional[str] = ..., readiness: _Optional[bool] = ..., version: _Optional[str] = ..., uptime_seconds: _Optional[float] = ..., dependencies: _Optional[_Mapping[str, DependencyStatus]] = ...) -> None: ...

class StatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
