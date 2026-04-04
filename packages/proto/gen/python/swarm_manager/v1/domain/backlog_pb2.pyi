from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class BacklogItem(_message.Message):
    __slots__ = ("name", "title", "description", "status", "priority", "tags", "created", "updated", "kind", "depends_on", "initiative", "effort", "acceptance_allow", "acceptance_deny", "spawned_from", "note")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    UPDATED_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    DEPENDS_ON_FIELD_NUMBER: _ClassVar[int]
    INITIATIVE_FIELD_NUMBER: _ClassVar[int]
    EFFORT_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_ALLOW_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_DENY_FIELD_NUMBER: _ClassVar[int]
    SPAWNED_FROM_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    name: str
    title: str
    description: str
    status: str
    priority: int
    tags: _containers.RepeatedScalarFieldContainer[str]
    created: str
    updated: str
    kind: str
    depends_on: _containers.RepeatedScalarFieldContainer[str]
    initiative: str
    effort: str
    acceptance_allow: _containers.RepeatedScalarFieldContainer[str]
    acceptance_deny: _containers.RepeatedScalarFieldContainer[str]
    spawned_from: str
    note: str
    def __init__(self, name: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ..., priority: _Optional[int] = ..., tags: _Optional[_Iterable[str]] = ..., created: _Optional[str] = ..., updated: _Optional[str] = ..., kind: _Optional[str] = ..., depends_on: _Optional[_Iterable[str]] = ..., initiative: _Optional[str] = ..., effort: _Optional[str] = ..., acceptance_allow: _Optional[_Iterable[str]] = ..., acceptance_deny: _Optional[_Iterable[str]] = ..., spawned_from: _Optional[str] = ..., note: _Optional[str] = ...) -> None: ...

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

class ClarificationMessage(_message.Message):
    __slots__ = ("role", "content", "created_at", "attachment_ids")
    ROLE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENT_IDS_FIELD_NUMBER: _ClassVar[int]
    role: str
    content: str
    created_at: str
    attachment_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, role: _Optional[str] = ..., content: _Optional[str] = ..., created_at: _Optional[str] = ..., attachment_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class ClarificationImpact(_message.Message):
    __slots__ = ("level", "reasoning", "context_note", "suggested_update")
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    REASONING_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_NOTE_FIELD_NUMBER: _ClassVar[int]
    SUGGESTED_UPDATE_FIELD_NUMBER: _ClassVar[int]
    level: str
    reasoning: str
    context_note: str
    suggested_update: str
    def __init__(self, level: _Optional[str] = ..., reasoning: _Optional[str] = ..., context_note: _Optional[str] = ..., suggested_update: _Optional[str] = ...) -> None: ...

class ClarificationThread(_message.Message):
    __slots__ = ("id", "round_number", "item_id", "run_id", "messages", "latest_impact", "status", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    ROUND_NUMBER_FIELD_NUMBER: _ClassVar[int]
    ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGES_FIELD_NUMBER: _ClassVar[int]
    LATEST_IMPACT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    round_number: int
    item_id: str
    run_id: str
    messages: _containers.RepeatedCompositeFieldContainer[ClarificationMessage]
    latest_impact: ClarificationImpact
    status: str
    created_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., round_number: _Optional[int] = ..., item_id: _Optional[str] = ..., run_id: _Optional[str] = ..., messages: _Optional[_Iterable[_Union[ClarificationMessage, _Mapping]]] = ..., latest_impact: _Optional[_Union[ClarificationImpact, _Mapping]] = ..., status: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...
