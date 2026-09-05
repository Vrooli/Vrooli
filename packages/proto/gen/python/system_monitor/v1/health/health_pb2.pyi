from common.v1 import types_pb2 as _types_pb2
from google.api import annotations_pb2 as _annotations_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class HealthRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class HealthResponse(_message.Message):
    __slots__ = ("status", "service", "timestamp", "readiness", "version", "dependencies", "metrics", "processor_active", "maintenance_state")
    class DependenciesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    class MetricsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2.JsonValue, _Mapping]] = ...) -> None: ...
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SERVICE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    READINESS_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    PROCESSOR_ACTIVE_FIELD_NUMBER: _ClassVar[int]
    MAINTENANCE_STATE_FIELD_NUMBER: _ClassVar[int]
    status: _types_pb2.HealthStatus
    service: str
    timestamp: str
    readiness: bool
    version: str
    dependencies: _containers.MessageMap[str, _types_pb2.JsonValue]
    metrics: _containers.MessageMap[str, _types_pb2.JsonValue]
    processor_active: bool
    maintenance_state: str
    def __init__(self, status: _Optional[_Union[_types_pb2.HealthStatus, str]] = ..., service: _Optional[str] = ..., timestamp: _Optional[str] = ..., readiness: _Optional[bool] = ..., version: _Optional[str] = ..., dependencies: _Optional[_Mapping[str, _types_pb2.JsonValue]] = ..., metrics: _Optional[_Mapping[str, _types_pb2.JsonValue]] = ..., processor_active: _Optional[bool] = ..., maintenance_state: _Optional[str] = ...) -> None: ...
