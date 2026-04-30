from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class Initiative(_message.Message):
    __slots__ = ("name", "title", "description", "status", "items", "created", "updated", "note", "archived_at", "priority", "depends_on", "mode", "acceptance_criteria")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    UPDATED_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_AT_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    DEPENDS_ON_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_CRITERIA_FIELD_NUMBER: _ClassVar[int]
    name: str
    title: str
    description: str
    status: str
    items: _containers.RepeatedScalarFieldContainer[str]
    created: str
    updated: str
    note: str
    archived_at: str
    priority: int
    depends_on: _containers.RepeatedScalarFieldContainer[str]
    mode: str
    acceptance_criteria: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ..., items: _Optional[_Iterable[str]] = ..., created: _Optional[str] = ..., updated: _Optional[str] = ..., note: _Optional[str] = ..., archived_at: _Optional[str] = ..., priority: _Optional[int] = ..., depends_on: _Optional[_Iterable[str]] = ..., mode: _Optional[str] = ..., acceptance_criteria: _Optional[_Iterable[str]] = ...) -> None: ...

class InitiativeRollup(_message.Message):
    __slots__ = ("total", "completed", "in_progress", "failed", "pending", "archived")
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_FIELD_NUMBER: _ClassVar[int]
    IN_PROGRESS_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    PENDING_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_FIELD_NUMBER: _ClassVar[int]
    total: int
    completed: int
    in_progress: int
    failed: int
    pending: int
    archived: int
    def __init__(self, total: _Optional[int] = ..., completed: _Optional[int] = ..., in_progress: _Optional[int] = ..., failed: _Optional[int] = ..., pending: _Optional[int] = ..., archived: _Optional[int] = ...) -> None: ...
