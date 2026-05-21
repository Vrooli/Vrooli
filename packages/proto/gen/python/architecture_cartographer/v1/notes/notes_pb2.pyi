import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Note(_message.Message):
    __slots__ = ("id", "title", "body", "created_at", "updated_at", "attachment_keys")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENT_KEYS_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    body: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    attachment_keys: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., body: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., attachment_keys: _Optional[_Iterable[str]] = ...) -> None: ...

class ListNotesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListNotesResponse(_message.Message):
    __slots__ = ("notes",)
    NOTES_FIELD_NUMBER: _ClassVar[int]
    notes: _containers.RepeatedCompositeFieldContainer[Note]
    def __init__(self, notes: _Optional[_Iterable[_Union[Note, _Mapping]]] = ...) -> None: ...

class CreateNoteRequest(_message.Message):
    __slots__ = ("title", "body")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    title: str
    body: str
    def __init__(self, title: _Optional[str] = ..., body: _Optional[str] = ...) -> None: ...

class CreateNoteResponse(_message.Message):
    __slots__ = ("note",)
    NOTE_FIELD_NUMBER: _ClassVar[int]
    note: Note
    def __init__(self, note: _Optional[_Union[Note, _Mapping]] = ...) -> None: ...

class GetNoteRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetNoteResponse(_message.Message):
    __slots__ = ("note",)
    NOTE_FIELD_NUMBER: _ClassVar[int]
    note: Note
    def __init__(self, note: _Optional[_Union[Note, _Mapping]] = ...) -> None: ...
