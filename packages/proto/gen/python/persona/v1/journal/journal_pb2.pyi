import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class JournalEntry(_message.Message):
    __slots__ = ("id", "persona_id", "actor", "verb", "run_id", "authorising_human", "at", "outcome", "constraint", "details")
    class DetailsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    VERB_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    AUTHORISING_HUMAN_FIELD_NUMBER: _ClassVar[int]
    AT_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    CONSTRAINT_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    id: str
    persona_id: str
    actor: str
    verb: str
    run_id: str
    authorising_human: str
    at: _timestamp_pb2.Timestamp
    outcome: str
    constraint: str
    details: _containers.ScalarMap[str, str]
    def __init__(self, id: _Optional[str] = ..., persona_id: _Optional[str] = ..., actor: _Optional[str] = ..., verb: _Optional[str] = ..., run_id: _Optional[str] = ..., authorising_human: _Optional[str] = ..., at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., outcome: _Optional[str] = ..., constraint: _Optional[str] = ..., details: _Optional[_Mapping[str, str]] = ...) -> None: ...

class AppendRequest(_message.Message):
    __slots__ = ("entry",)
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    entry: JournalEntry
    def __init__(self, entry: _Optional[_Union[JournalEntry, _Mapping]] = ...) -> None: ...

class AppendResponse(_message.Message):
    __slots__ = ("entry",)
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    entry: JournalEntry
    def __init__(self, entry: _Optional[_Union[JournalEntry, _Mapping]] = ...) -> None: ...

class ListRequest(_message.Message):
    __slots__ = ("persona_id", "limit")
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    limit: int
    def __init__(self, persona_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListResponse(_message.Message):
    __slots__ = ("entries",)
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[JournalEntry]
    def __init__(self, entries: _Optional[_Iterable[_Union[JournalEntry, _Mapping]]] = ...) -> None: ...
