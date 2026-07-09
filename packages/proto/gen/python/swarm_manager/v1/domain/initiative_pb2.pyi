from buf.validate import validate_pb2 as _validate_pb2
from swarm_manager.v1.domain import agent_session_pb2 as _agent_session_pb2
from swarm_manager.v1.domain import plan_ref_pb2 as _plan_ref_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Initiative(_message.Message):
    __slots__ = ("name", "title", "description", "status", "items", "created", "updated", "note", "archived_at", "priority", "depends_on", "mode", "acceptance_criteria", "created_by", "plan_ref")
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
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    PLAN_REF_FIELD_NUMBER: _ClassVar[int]
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
    created_by: _agent_session_pb2.AgentSessionAttribution
    plan_ref: _plan_ref_pb2.PlanRef
    def __init__(self, name: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ..., items: _Optional[_Iterable[str]] = ..., created: _Optional[str] = ..., updated: _Optional[str] = ..., note: _Optional[str] = ..., archived_at: _Optional[str] = ..., priority: _Optional[int] = ..., depends_on: _Optional[_Iterable[str]] = ..., mode: _Optional[str] = ..., acceptance_criteria: _Optional[_Iterable[str]] = ..., created_by: _Optional[_Union[_agent_session_pb2.AgentSessionAttribution, _Mapping]] = ..., plan_ref: _Optional[_Union[_plan_ref_pb2.PlanRef, _Mapping]] = ...) -> None: ...

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
