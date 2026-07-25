from buf.validate import validate_pb2 as _validate_pb2
from swarm_manager.v1.domain import goal_pb2 as _goal_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListGoalsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetGoalRequest(_message.Message):
    __slots__ = ("name",)
    NAME_FIELD_NUMBER: _ClassVar[int]
    name: str
    def __init__(self, name: _Optional[str] = ...) -> None: ...

class DeleteGoalRequest(_message.Message):
    __slots__ = ("name",)
    NAME_FIELD_NUMBER: _ClassVar[int]
    name: str
    def __init__(self, name: _Optional[str] = ...) -> None: ...

class ArchiveGoalRequest(_message.Message):
    __slots__ = ("name",)
    NAME_FIELD_NUMBER: _ClassVar[int]
    name: str
    def __init__(self, name: _Optional[str] = ...) -> None: ...

class UnarchiveGoalRequest(_message.Message):
    __slots__ = ("name",)
    NAME_FIELD_NUMBER: _ClassVar[int]
    name: str
    def __init__(self, name: _Optional[str] = ...) -> None: ...

class CreateGoalRequest(_message.Message):
    __slots__ = ("name", "title", "description", "priority", "targets")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    name: str
    title: str
    description: str
    priority: int
    targets: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., priority: _Optional[int] = ..., targets: _Optional[_Iterable[str]] = ...) -> None: ...

class UpdateGoalRequest(_message.Message):
    __slots__ = ("name", "title", "description", "priority")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    name: str
    title: str
    description: str
    priority: int
    def __init__(self, name: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., priority: _Optional[int] = ...) -> None: ...

class UpdateGoalTargetsRequest(_message.Message):
    __slots__ = ("name", "targets")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    name: str
    targets: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., targets: _Optional[_Iterable[str]] = ...) -> None: ...

class CreateMilestoneRequest(_message.Message):
    __slots__ = ("goal_name", "milestone")
    GOAL_NAME_FIELD_NUMBER: _ClassVar[int]
    MILESTONE_FIELD_NUMBER: _ClassVar[int]
    goal_name: str
    milestone: _goal_pb2.Milestone
    def __init__(self, goal_name: _Optional[str] = ..., milestone: _Optional[_Union[_goal_pb2.Milestone, _Mapping]] = ...) -> None: ...

class UpdateMilestoneRequest(_message.Message):
    __slots__ = ("goal_name", "milestone")
    GOAL_NAME_FIELD_NUMBER: _ClassVar[int]
    MILESTONE_FIELD_NUMBER: _ClassVar[int]
    goal_name: str
    milestone: _goal_pb2.Milestone
    def __init__(self, goal_name: _Optional[str] = ..., milestone: _Optional[_Union[_goal_pb2.Milestone, _Mapping]] = ...) -> None: ...

class ArchiveMilestoneRequest(_message.Message):
    __slots__ = ("goal_name", "milestone_name")
    GOAL_NAME_FIELD_NUMBER: _ClassVar[int]
    MILESTONE_NAME_FIELD_NUMBER: _ClassVar[int]
    goal_name: str
    milestone_name: str
    def __init__(self, goal_name: _Optional[str] = ..., milestone_name: _Optional[str] = ...) -> None: ...

class UpdateMilestoneItemsRequest(_message.Message):
    __slots__ = ("goal_name", "milestone_name", "items")
    GOAL_NAME_FIELD_NUMBER: _ClassVar[int]
    MILESTONE_NAME_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    goal_name: str
    milestone_name: str
    items: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, goal_name: _Optional[str] = ..., milestone_name: _Optional[str] = ..., items: _Optional[_Iterable[str]] = ...) -> None: ...

class CloseOutGoalRequest(_message.Message):
    __slots__ = ("name",)
    NAME_FIELD_NUMBER: _ClassVar[int]
    name: str
    def __init__(self, name: _Optional[str] = ...) -> None: ...

class ListPendingGoalWorkflowsRequest(_message.Message):
    __slots__ = ("goal_name",)
    GOAL_NAME_FIELD_NUMBER: _ClassVar[int]
    goal_name: str
    def __init__(self, goal_name: _Optional[str] = ...) -> None: ...

class ApplyGoalWorkflowRequest(_message.Message):
    __slots__ = ("goal_name", "execution_id")
    GOAL_NAME_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    goal_name: str
    execution_id: str
    def __init__(self, goal_name: _Optional[str] = ..., execution_id: _Optional[str] = ...) -> None: ...

class PendingGoalWorkflow(_message.Message):
    __slots__ = ("goal_name", "execution_id", "transition", "milestone", "goal_version", "stale", "attempts", "last_attempt_at", "last_error")
    GOAL_NAME_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    TRANSITION_FIELD_NUMBER: _ClassVar[int]
    MILESTONE_FIELD_NUMBER: _ClassVar[int]
    GOAL_VERSION_FIELD_NUMBER: _ClassVar[int]
    STALE_FIELD_NUMBER: _ClassVar[int]
    ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    LAST_ATTEMPT_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_ERROR_FIELD_NUMBER: _ClassVar[int]
    goal_name: str
    execution_id: str
    transition: str
    milestone: str
    goal_version: str
    stale: bool
    attempts: int
    last_attempt_at: str
    last_error: str
    def __init__(self, goal_name: _Optional[str] = ..., execution_id: _Optional[str] = ..., transition: _Optional[str] = ..., milestone: _Optional[str] = ..., goal_version: _Optional[str] = ..., stale: _Optional[bool] = ..., attempts: _Optional[int] = ..., last_attempt_at: _Optional[str] = ..., last_error: _Optional[str] = ...) -> None: ...

class ListPendingGoalWorkflowsResponse(_message.Message):
    __slots__ = ("pending",)
    PENDING_FIELD_NUMBER: _ClassVar[int]
    pending: _containers.RepeatedCompositeFieldContainer[PendingGoalWorkflow]
    def __init__(self, pending: _Optional[_Iterable[_Union[PendingGoalWorkflow, _Mapping]]] = ...) -> None: ...

class ApplyGoalWorkflowResponse(_message.Message):
    __slots__ = ("execution_id", "session_id", "proposal_ids", "outcome", "already_applied")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    PROPOSAL_IDS_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    ALREADY_APPLIED_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    session_id: str
    proposal_ids: _containers.RepeatedScalarFieldContainer[str]
    outcome: str
    already_applied: bool
    def __init__(self, execution_id: _Optional[str] = ..., session_id: _Optional[str] = ..., proposal_ids: _Optional[_Iterable[str]] = ..., outcome: _Optional[str] = ..., already_applied: _Optional[bool] = ...) -> None: ...

class GoalResponse(_message.Message):
    __slots__ = ("goal", "scope")
    GOAL_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    goal: _goal_pb2.Goal
    scope: _goal_pb2.GoalScope
    def __init__(self, goal: _Optional[_Union[_goal_pb2.Goal, _Mapping]] = ..., scope: _Optional[_Union[_goal_pb2.GoalScope, _Mapping]] = ...) -> None: ...

class ListGoalsResponse(_message.Message):
    __slots__ = ("goals",)
    GOALS_FIELD_NUMBER: _ClassVar[int]
    goals: _containers.RepeatedCompositeFieldContainer[GoalResponse]
    def __init__(self, goals: _Optional[_Iterable[_Union[GoalResponse, _Mapping]]] = ...) -> None: ...

class GoalScopeResponse(_message.Message):
    __slots__ = ("scope",)
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    scope: _goal_pb2.GoalScope
    def __init__(self, scope: _Optional[_Union[_goal_pb2.GoalScope, _Mapping]] = ...) -> None: ...

class EmptyGoalResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
