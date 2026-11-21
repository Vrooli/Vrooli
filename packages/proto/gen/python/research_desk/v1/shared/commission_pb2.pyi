from research_desk.v1.shared import evidence_pb2 as _evidence_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ReadWhen(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    READ_WHEN_UNSPECIFIED: _ClassVar[ReadWhen]
    READ_WHEN_NEXT_UNIT: _ClassVar[ReadWhen]
    READ_WHEN_AFTER_RESUME: _ClassVar[ReadWhen]
    READ_WHEN_NOT_UNTIL_STEERING: _ClassVar[ReadWhen]
    READ_WHEN_NEVER: _ClassVar[ReadWhen]
    READ_WHEN_AT_START: _ClassVar[ReadWhen]
    READ_WHEN_UNKNOWN: _ClassVar[ReadWhen]
READ_WHEN_UNSPECIFIED: ReadWhen
READ_WHEN_NEXT_UNIT: ReadWhen
READ_WHEN_AFTER_RESUME: ReadWhen
READ_WHEN_NOT_UNTIL_STEERING: ReadWhen
READ_WHEN_NEVER: ReadWhen
READ_WHEN_AT_START: ReadWhen
READ_WHEN_UNKNOWN: ReadWhen

class NeverStarted(_message.Message):
    __slots__ = ("assertable", "reason")
    ASSERTABLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    assertable: bool
    reason: str
    def __init__(self, assertable: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class EntryReading(_message.Message):
    __slots__ = ("when", "message", "read_since")
    WHEN_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    READ_SINCE_FIELD_NUMBER: _ClassVar[int]
    when: ReadWhen
    message: str
    read_since: _evidence_pb2.DerivedTime
    def __init__(self, when: _Optional[_Union[ReadWhen, str]] = ..., message: _Optional[str] = ..., read_since: _Optional[_Union[_evidence_pb2.DerivedTime, _Mapping]] = ...) -> None: ...

class ReadSet(_message.Message):
    __slots__ = ("role", "files", "bytes", "estimated_tokens", "flags")
    ROLE_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    BYTES_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_TOKENS_FIELD_NUMBER: _ClassVar[int]
    FLAGS_FIELD_NUMBER: _ClassVar[int]
    role: str
    files: _containers.RepeatedCompositeFieldContainer[_evidence_pb2.FileFact]
    bytes: int
    estimated_tokens: int
    flags: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, role: _Optional[str] = ..., files: _Optional[_Iterable[_Union[_evidence_pb2.FileFact, _Mapping]]] = ..., bytes: _Optional[int] = ..., estimated_tokens: _Optional[int] = ..., flags: _Optional[_Iterable[str]] = ...) -> None: ...

class Workstream(_message.Message):
    __slots__ = ("id", "status", "showcase", "title", "magnitude")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SHOWCASE_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    MAGNITUDE_FIELD_NUMBER: _ClassVar[int]
    id: str
    status: str
    showcase: str
    title: str
    magnitude: str
    def __init__(self, id: _Optional[str] = ..., status: _Optional[str] = ..., showcase: _Optional[str] = ..., title: _Optional[str] = ..., magnitude: _Optional[str] = ...) -> None: ...
