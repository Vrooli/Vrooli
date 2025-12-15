from google.protobuf import any_pb2 as _any_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Value(_message.Message):
    __slots__ = ()
    NULL_VALUE_FIELD_NUMBER: _ClassVar[int]
    BOOL_VALUE_FIELD_NUMBER: _ClassVar[int]
    INT64_VALUE_FIELD_NUMBER: _ClassVar[int]
    UINT64_VALUE_FIELD_NUMBER: _ClassVar[int]
    DOUBLE_VALUE_FIELD_NUMBER: _ClassVar[int]
    STRING_VALUE_FIELD_NUMBER: _ClassVar[int]
    BYTES_VALUE_FIELD_NUMBER: _ClassVar[int]
    ENUM_VALUE_FIELD_NUMBER: _ClassVar[int]
    OBJECT_VALUE_FIELD_NUMBER: _ClassVar[int]
    MAP_VALUE_FIELD_NUMBER: _ClassVar[int]
    LIST_VALUE_FIELD_NUMBER: _ClassVar[int]
    TYPE_VALUE_FIELD_NUMBER: _ClassVar[int]
    null_value: _struct_pb2.NullValue
    bool_value: bool
    int64_value: int
    uint64_value: int
    double_value: float
    string_value: str
    bytes_value: bytes
    enum_value: EnumValue
    object_value: _any_pb2.Any
    map_value: MapValue
    list_value: ListValue
    type_value: str
    def __init__(self, null_value: _Optional[_Union[_struct_pb2.NullValue, str]] = ..., bool_value: _Optional[bool] = ..., int64_value: _Optional[int] = ..., uint64_value: _Optional[int] = ..., double_value: _Optional[float] = ..., string_value: _Optional[str] = ..., bytes_value: _Optional[bytes] = ..., enum_value: _Optional[_Union[EnumValue, _Mapping]] = ..., object_value: _Optional[_Union[_any_pb2.Any, _Mapping]] = ..., map_value: _Optional[_Union[MapValue, _Mapping]] = ..., list_value: _Optional[_Union[ListValue, _Mapping]] = ..., type_value: _Optional[str] = ...) -> None: ...

class EnumValue(_message.Message):
    __slots__ = ()
    TYPE_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    type: str
    value: int
    def __init__(self, type: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...

class ListValue(_message.Message):
    __slots__ = ()
    VALUES_FIELD_NUMBER: _ClassVar[int]
    values: _containers.RepeatedCompositeFieldContainer[Value]
    def __init__(self, values: _Optional[_Iterable[_Union[Value, _Mapping]]] = ...) -> None: ...

class MapValue(_message.Message):
    __slots__ = ()
    class Entry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: Value
        value: Value
        def __init__(self, key: _Optional[_Union[Value, _Mapping]] = ..., value: _Optional[_Union[Value, _Mapping]] = ...) -> None: ...
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[MapValue.Entry]
    def __init__(self, entries: _Optional[_Iterable[_Union[MapValue.Entry, _Mapping]]] = ...) -> None: ...
