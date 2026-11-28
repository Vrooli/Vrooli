import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class OperatorEntry(_message.Message):
    __slots__ = ("at", "kind", "subject", "via", "body")
    AT_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    VIA_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    at: _timestamp_pb2.Timestamp
    kind: str
    subject: str
    via: str
    body: str
    def __init__(self, at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., kind: _Optional[str] = ..., subject: _Optional[str] = ..., via: _Optional[str] = ..., body: _Optional[str] = ...) -> None: ...

class UnparsedHeading(_message.Message):
    __slots__ = ("line", "text")
    LINE_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    line: int
    text: str
    def __init__(self, line: _Optional[int] = ..., text: _Optional[str] = ...) -> None: ...
