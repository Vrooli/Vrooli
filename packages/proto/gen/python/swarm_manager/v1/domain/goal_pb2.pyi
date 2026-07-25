from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Goal(_message.Message):
    __slots__ = ("name", "title", "description", "status", "priority", "targets", "milestones", "created", "updated", "archived_at")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    MILESTONES_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    UPDATED_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_AT_FIELD_NUMBER: _ClassVar[int]
    name: str
    title: str
    description: str
    status: str
    priority: int
    targets: _containers.RepeatedScalarFieldContainer[str]
    milestones: _containers.RepeatedCompositeFieldContainer[Milestone]
    created: str
    updated: str
    archived_at: str
    def __init__(self, name: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ..., priority: _Optional[int] = ..., targets: _Optional[_Iterable[str]] = ..., milestones: _Optional[_Iterable[_Union[Milestone, _Mapping]]] = ..., created: _Optional[str] = ..., updated: _Optional[str] = ..., archived_at: _Optional[str] = ...) -> None: ...

class Milestone(_message.Message):
    __slots__ = ("name", "title", "description", "items", "acceptance_criteria", "depends_on", "archived_at")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_CRITERIA_FIELD_NUMBER: _ClassVar[int]
    DEPENDS_ON_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_AT_FIELD_NUMBER: _ClassVar[int]
    name: str
    title: str
    description: str
    items: _containers.RepeatedScalarFieldContainer[str]
    acceptance_criteria: _containers.RepeatedScalarFieldContainer[str]
    depends_on: _containers.RepeatedScalarFieldContainer[str]
    archived_at: str
    def __init__(self, name: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., items: _Optional[_Iterable[str]] = ..., acceptance_criteria: _Optional[_Iterable[str]] = ..., depends_on: _Optional[_Iterable[str]] = ..., archived_at: _Optional[str] = ...) -> None: ...

class MilestoneRollup(_message.Message):
    __slots__ = ("milestone_name", "total", "completed", "ready", "blocked", "orphaned")
    MILESTONE_NAME_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    BLOCKED_FIELD_NUMBER: _ClassVar[int]
    ORPHANED_FIELD_NUMBER: _ClassVar[int]
    milestone_name: str
    total: int
    completed: int
    ready: int
    blocked: int
    orphaned: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, milestone_name: _Optional[str] = ..., total: _Optional[int] = ..., completed: _Optional[int] = ..., ready: _Optional[int] = ..., blocked: _Optional[int] = ..., orphaned: _Optional[_Iterable[str]] = ...) -> None: ...

class GoalScope(_message.Message):
    __slots__ = ("targets", "closure", "completed", "ready", "blocked", "milestones", "unassigned")
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    CLOSURE_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    BLOCKED_FIELD_NUMBER: _ClassVar[int]
    MILESTONES_FIELD_NUMBER: _ClassVar[int]
    UNASSIGNED_FIELD_NUMBER: _ClassVar[int]
    targets: _containers.RepeatedScalarFieldContainer[str]
    closure: _containers.RepeatedScalarFieldContainer[str]
    completed: _containers.RepeatedScalarFieldContainer[str]
    ready: _containers.RepeatedScalarFieldContainer[str]
    blocked: _containers.RepeatedScalarFieldContainer[str]
    milestones: _containers.RepeatedCompositeFieldContainer[MilestoneRollup]
    unassigned: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, targets: _Optional[_Iterable[str]] = ..., closure: _Optional[_Iterable[str]] = ..., completed: _Optional[_Iterable[str]] = ..., ready: _Optional[_Iterable[str]] = ..., blocked: _Optional[_Iterable[str]] = ..., milestones: _Optional[_Iterable[_Union[MilestoneRollup, _Mapping]]] = ..., unassigned: _Optional[_Iterable[str]] = ...) -> None: ...
