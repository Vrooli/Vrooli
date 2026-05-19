import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Skill(_message.Message):
    __slots__ = ("id", "version", "content_hash", "synced_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    SYNCED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    version: str
    content_hash: str
    synced_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., version: _Optional[str] = ..., content_hash: _Optional[str] = ..., synced_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class SyncRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SyncResponse(_message.Message):
    __slots__ = ("skills", "added", "updated", "removed")
    SKILLS_FIELD_NUMBER: _ClassVar[int]
    ADDED_FIELD_NUMBER: _ClassVar[int]
    UPDATED_FIELD_NUMBER: _ClassVar[int]
    REMOVED_FIELD_NUMBER: _ClassVar[int]
    skills: _containers.RepeatedCompositeFieldContainer[Skill]
    added: int
    updated: int
    removed: int
    def __init__(self, skills: _Optional[_Iterable[_Union[Skill, _Mapping]]] = ..., added: _Optional[int] = ..., updated: _Optional[int] = ..., removed: _Optional[int] = ...) -> None: ...

class ListSkillsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListSkillsResponse(_message.Message):
    __slots__ = ("skills",)
    SKILLS_FIELD_NUMBER: _ClassVar[int]
    skills: _containers.RepeatedCompositeFieldContainer[Skill]
    def __init__(self, skills: _Optional[_Iterable[_Union[Skill, _Mapping]]] = ...) -> None: ...

class GetSkillRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetSkillResponse(_message.Message):
    __slots__ = ("skill",)
    SKILL_FIELD_NUMBER: _ClassVar[int]
    skill: Skill
    def __init__(self, skill: _Optional[_Union[Skill, _Mapping]] = ...) -> None: ...
