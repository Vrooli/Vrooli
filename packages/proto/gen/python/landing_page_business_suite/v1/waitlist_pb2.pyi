import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class WaitlistEntry(_message.Message):
    __slots__ = ("id", "email", "source", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: int
    email: str
    source: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[int] = ..., email: _Optional[str] = ..., source: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CreateWaitlistEntryRequest(_message.Message):
    __slots__ = ("email", "source")
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    email: str
    source: str
    def __init__(self, email: _Optional[str] = ..., source: _Optional[str] = ...) -> None: ...

class CreateWaitlistEntryResponse(_message.Message):
    __slots__ = ("success", "message", "entry")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    success: bool
    message: str
    entry: WaitlistEntry
    def __init__(self, success: _Optional[bool] = ..., message: _Optional[str] = ..., entry: _Optional[_Union[WaitlistEntry, _Mapping]] = ...) -> None: ...

class ListWaitlistEntriesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListWaitlistEntriesResponse(_message.Message):
    __slots__ = ("entries",)
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[WaitlistEntry]
    def __init__(self, entries: _Optional[_Iterable[_Union[WaitlistEntry, _Mapping]]] = ...) -> None: ...

class DeleteWaitlistEntryRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: int
    def __init__(self, id: _Optional[int] = ...) -> None: ...

class DeleteWaitlistEntryResponse(_message.Message):
    __slots__ = ("deleted",)
    DELETED_FIELD_NUMBER: _ClassVar[int]
    deleted: bool
    def __init__(self, deleted: _Optional[bool] = ...) -> None: ...

class ExportWaitlistEntriesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ExportWaitlistEntriesResponse(_message.Message):
    __slots__ = ("csv", "filename")
    CSV_FIELD_NUMBER: _ClassVar[int]
    FILENAME_FIELD_NUMBER: _ClassVar[int]
    csv: str
    filename: str
    def __init__(self, csv: _Optional[str] = ..., filename: _Optional[str] = ...) -> None: ...
