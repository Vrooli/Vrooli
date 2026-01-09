from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class SourceInfo(_message.Message):
    __slots__ = ()
    class PositionsEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: int
        value: int
        def __init__(self, key: _Optional[int] = ..., value: _Optional[int] = ...) -> None: ...
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    LINE_OFFSETS_FIELD_NUMBER: _ClassVar[int]
    POSITIONS_FIELD_NUMBER: _ClassVar[int]
    location: str
    line_offsets: _containers.RepeatedScalarFieldContainer[int]
    positions: _containers.ScalarMap[int, int]
    def __init__(self, location: _Optional[str] = ..., line_offsets: _Optional[_Iterable[int]] = ..., positions: _Optional[_Mapping[int, int]] = ...) -> None: ...

class SourcePosition(_message.Message):
    __slots__ = ()
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    LINE_FIELD_NUMBER: _ClassVar[int]
    COLUMN_FIELD_NUMBER: _ClassVar[int]
    location: str
    offset: int
    line: int
    column: int
    def __init__(self, location: _Optional[str] = ..., offset: _Optional[int] = ..., line: _Optional[int] = ..., column: _Optional[int] = ...) -> None: ...
