import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DebtEntry(_message.Message):
    __slots__ = ("key", "template_id", "source", "severity", "status", "title", "detail", "first_seen_at", "last_seen_at")
    KEY_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    FIRST_SEEN_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_SEEN_AT_FIELD_NUMBER: _ClassVar[int]
    key: str
    template_id: str
    source: str
    severity: str
    status: str
    title: str
    detail: str
    first_seen_at: _timestamp_pb2.Timestamp
    last_seen_at: _timestamp_pb2.Timestamp
    def __init__(self, key: _Optional[str] = ..., template_id: _Optional[str] = ..., source: _Optional[str] = ..., severity: _Optional[str] = ..., status: _Optional[str] = ..., title: _Optional[str] = ..., detail: _Optional[str] = ..., first_seen_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_seen_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListDebtRequest(_message.Message):
    __slots__ = ("template_id", "status")
    TEMPLATE_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    template_id: str
    status: str
    def __init__(self, template_id: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class ListDebtResponse(_message.Message):
    __slots__ = ("entries",)
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[DebtEntry]
    def __init__(self, entries: _Optional[_Iterable[_Union[DebtEntry, _Mapping]]] = ...) -> None: ...

class GetDebtRequest(_message.Message):
    __slots__ = ("key",)
    KEY_FIELD_NUMBER: _ClassVar[int]
    key: str
    def __init__(self, key: _Optional[str] = ...) -> None: ...

class GetDebtResponse(_message.Message):
    __slots__ = ("entry",)
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    entry: DebtEntry
    def __init__(self, entry: _Optional[_Union[DebtEntry, _Mapping]] = ...) -> None: ...
