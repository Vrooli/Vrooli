from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class BacklogItem(_message.Message):
    __slots__ = ("name", "title", "description", "status", "priority", "tags", "created", "updated", "kind", "research_target", "depends_on", "initiative", "effort", "scope", "acceptance_allow", "acceptance_deny")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    UPDATED_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    RESEARCH_TARGET_FIELD_NUMBER: _ClassVar[int]
    DEPENDS_ON_FIELD_NUMBER: _ClassVar[int]
    INITIATIVE_FIELD_NUMBER: _ClassVar[int]
    EFFORT_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_ALLOW_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_DENY_FIELD_NUMBER: _ClassVar[int]
    name: str
    title: str
    description: str
    status: str
    priority: int
    tags: _containers.RepeatedScalarFieldContainer[str]
    created: str
    updated: str
    kind: str
    research_target: str
    depends_on: _containers.RepeatedScalarFieldContainer[str]
    initiative: str
    effort: str
    scope: str
    acceptance_allow: _containers.RepeatedScalarFieldContainer[str]
    acceptance_deny: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ..., priority: _Optional[int] = ..., tags: _Optional[_Iterable[str]] = ..., created: _Optional[str] = ..., updated: _Optional[str] = ..., kind: _Optional[str] = ..., research_target: _Optional[str] = ..., depends_on: _Optional[_Iterable[str]] = ..., initiative: _Optional[str] = ..., effort: _Optional[str] = ..., scope: _Optional[str] = ..., acceptance_allow: _Optional[_Iterable[str]] = ..., acceptance_deny: _Optional[_Iterable[str]] = ...) -> None: ...

class BacklogFile(_message.Message):
    __slots__ = ("name", "path", "type", "size", "children")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    CHILDREN_FIELD_NUMBER: _ClassVar[int]
    name: str
    path: str
    type: str
    size: int
    children: _containers.RepeatedCompositeFieldContainer[BacklogFile]
    def __init__(self, name: _Optional[str] = ..., path: _Optional[str] = ..., type: _Optional[str] = ..., size: _Optional[int] = ..., children: _Optional[_Iterable[_Union[BacklogFile, _Mapping]]] = ...) -> None: ...
