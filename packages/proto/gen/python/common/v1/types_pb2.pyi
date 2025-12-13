from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class JsonValue(_message.Message):
    __slots__ = ()
    BOOL_VALUE_FIELD_NUMBER: _ClassVar[int]
    INT_VALUE_FIELD_NUMBER: _ClassVar[int]
    DOUBLE_VALUE_FIELD_NUMBER: _ClassVar[int]
    STRING_VALUE_FIELD_NUMBER: _ClassVar[int]
    OBJECT_VALUE_FIELD_NUMBER: _ClassVar[int]
    LIST_VALUE_FIELD_NUMBER: _ClassVar[int]
    NULL_VALUE_FIELD_NUMBER: _ClassVar[int]
    BYTES_VALUE_FIELD_NUMBER: _ClassVar[int]
    bool_value: bool
    int_value: int
    double_value: float
    string_value: str
    object_value: JsonObject
    list_value: JsonList
    null_value: _struct_pb2.NullValue
    bytes_value: bytes
    def __init__(self, bool_value: _Optional[bool] = ..., int_value: _Optional[int] = ..., double_value: _Optional[float] = ..., string_value: _Optional[str] = ..., object_value: _Optional[_Union[JsonObject, _Mapping]] = ..., list_value: _Optional[_Union[JsonList, _Mapping]] = ..., null_value: _Optional[_Union[_struct_pb2.NullValue, str]] = ..., bytes_value: _Optional[bytes] = ...) -> None: ...

class JsonObject(_message.Message):
    __slots__ = ()
    class FieldsEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[JsonValue, _Mapping]] = ...) -> None: ...
    FIELDS_FIELD_NUMBER: _ClassVar[int]
    fields: _containers.MessageMap[str, JsonValue]
    def __init__(self, fields: _Optional[_Mapping[str, JsonValue]] = ...) -> None: ...

class JsonList(_message.Message):
    __slots__ = ()
    VALUES_FIELD_NUMBER: _ClassVar[int]
    values: _containers.RepeatedCompositeFieldContainer[JsonValue]
    def __init__(self, values: _Optional[_Iterable[_Union[JsonValue, _Mapping]]] = ...) -> None: ...

class PaginationRequest(_message.Message):
    __slots__ = ()
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    limit: int
    offset: int
    def __init__(self, limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class PaginationResponse(_message.Message):
    __slots__ = ()
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    HAS_MORE_FIELD_NUMBER: _ClassVar[int]
    total: int
    limit: int
    offset: int
    has_more: bool
    def __init__(self, total: _Optional[int] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ..., has_more: _Optional[bool] = ...) -> None: ...

class ErrorResponse(_message.Message):
    __slots__ = ()
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    details: _struct_pb2.Struct
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., details: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class HealthResponse(_message.Message):
    __slots__ = ()
    class DependenciesEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _struct_pb2.Value
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...
    class MetricsEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _struct_pb2.Value
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SERVICE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    READINESS_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    status: str
    service: str
    timestamp: str
    readiness: bool
    version: str
    dependencies: _containers.MessageMap[str, _struct_pb2.Value]
    metrics: _containers.MessageMap[str, _struct_pb2.Value]
    def __init__(self, status: _Optional[str] = ..., service: _Optional[str] = ..., timestamp: _Optional[str] = ..., readiness: _Optional[bool] = ..., version: _Optional[str] = ..., dependencies: _Optional[_Mapping[str, _struct_pb2.Value]] = ..., metrics: _Optional[_Mapping[str, _struct_pb2.Value]] = ...) -> None: ...
