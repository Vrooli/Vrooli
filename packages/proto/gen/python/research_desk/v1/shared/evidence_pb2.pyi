import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Evidence(_message.Message):
    __slots__ = ("known", "source", "as_of", "reason")
    KNOWN_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    AS_OF_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    known: bool
    source: str
    as_of: _timestamp_pb2.Timestamp
    reason: str
    def __init__(self, known: _Optional[bool] = ..., source: _Optional[str] = ..., as_of: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., reason: _Optional[str] = ...) -> None: ...

class DerivedText(_message.Message):
    __slots__ = ("value", "evidence")
    VALUE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    value: str
    evidence: Evidence
    def __init__(self, value: _Optional[str] = ..., evidence: _Optional[_Union[Evidence, _Mapping]] = ...) -> None: ...

class DerivedTime(_message.Message):
    __slots__ = ("value", "evidence", "at_least")
    VALUE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    AT_LEAST_FIELD_NUMBER: _ClassVar[int]
    value: _timestamp_pb2.Timestamp
    evidence: Evidence
    at_least: bool
    def __init__(self, value: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., evidence: _Optional[_Union[Evidence, _Mapping]] = ..., at_least: _Optional[bool] = ...) -> None: ...

class FileFact(_message.Message):
    __slots__ = ("name", "path", "bytes", "modified")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    BYTES_FIELD_NUMBER: _ClassVar[int]
    MODIFIED_FIELD_NUMBER: _ClassVar[int]
    name: str
    path: str
    bytes: int
    modified: _timestamp_pb2.Timestamp
    def __init__(self, name: _Optional[str] = ..., path: _Optional[str] = ..., bytes: _Optional[int] = ..., modified: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class DerivedFlag(_message.Message):
    __slots__ = ("value", "evidence")
    VALUE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    value: bool
    evidence: Evidence
    def __init__(self, value: _Optional[bool] = ..., evidence: _Optional[_Union[Evidence, _Mapping]] = ...) -> None: ...
