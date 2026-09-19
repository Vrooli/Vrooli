from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Goal(_message.Message):
    __slots__ = ("id", "title", "purpose", "status", "progress_method", "progress_basis_points", "target_basis_points", "created_at_unix_seconds", "updated_at_unix_seconds", "revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    PURPOSE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_METHOD_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_BASIS_POINTS_FIELD_NUMBER: _ClassVar[int]
    TARGET_BASIS_POINTS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_UNIX_SECONDS_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_UNIX_SECONDS_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    purpose: str
    status: str
    progress_method: str
    progress_basis_points: int
    target_basis_points: int
    created_at_unix_seconds: int
    updated_at_unix_seconds: int
    revision: int
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., purpose: _Optional[str] = ..., status: _Optional[str] = ..., progress_method: _Optional[str] = ..., progress_basis_points: _Optional[int] = ..., target_basis_points: _Optional[int] = ..., created_at_unix_seconds: _Optional[int] = ..., updated_at_unix_seconds: _Optional[int] = ..., revision: _Optional[int] = ...) -> None: ...

class Milestone(_message.Message):
    __slots__ = ("id", "goal_id", "title", "criteria", "due_date", "status", "created_at_unix_seconds", "updated_at_unix_seconds", "revision", "linked_work_item_id", "prerequisite_milestone_ids")
    ID_FIELD_NUMBER: _ClassVar[int]
    GOAL_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    CRITERIA_FIELD_NUMBER: _ClassVar[int]
    DUE_DATE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_UNIX_SECONDS_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_UNIX_SECONDS_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    LINKED_WORK_ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    PREREQUISITE_MILESTONE_IDS_FIELD_NUMBER: _ClassVar[int]
    id: str
    goal_id: str
    title: str
    criteria: str
    due_date: str
    status: str
    created_at_unix_seconds: int
    updated_at_unix_seconds: int
    revision: int
    linked_work_item_id: str
    prerequisite_milestone_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., goal_id: _Optional[str] = ..., title: _Optional[str] = ..., criteria: _Optional[str] = ..., due_date: _Optional[str] = ..., status: _Optional[str] = ..., created_at_unix_seconds: _Optional[int] = ..., updated_at_unix_seconds: _Optional[int] = ..., revision: _Optional[int] = ..., linked_work_item_id: _Optional[str] = ..., prerequisite_milestone_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class ListGoalsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListGoalsResponse(_message.Message):
    __slots__ = ("goals",)
    GOALS_FIELD_NUMBER: _ClassVar[int]
    goals: _containers.RepeatedCompositeFieldContainer[Goal]
    def __init__(self, goals: _Optional[_Iterable[_Union[Goal, _Mapping]]] = ...) -> None: ...

class CreateGoalRequest(_message.Message):
    __slots__ = ("title", "purpose", "progress_method", "target_basis_points")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    PURPOSE_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_METHOD_FIELD_NUMBER: _ClassVar[int]
    TARGET_BASIS_POINTS_FIELD_NUMBER: _ClassVar[int]
    title: str
    purpose: str
    progress_method: str
    target_basis_points: int
    def __init__(self, title: _Optional[str] = ..., purpose: _Optional[str] = ..., progress_method: _Optional[str] = ..., target_basis_points: _Optional[int] = ...) -> None: ...

class CreateGoalResponse(_message.Message):
    __slots__ = ("goal",)
    GOAL_FIELD_NUMBER: _ClassVar[int]
    goal: Goal
    def __init__(self, goal: _Optional[_Union[Goal, _Mapping]] = ...) -> None: ...

class UpdateGoalProgressRequest(_message.Message):
    __slots__ = ("id", "progress_basis_points", "expected_revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_BASIS_POINTS_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    progress_basis_points: int
    expected_revision: int
    def __init__(self, id: _Optional[str] = ..., progress_basis_points: _Optional[int] = ..., expected_revision: _Optional[int] = ...) -> None: ...

class UpdateGoalProgressResponse(_message.Message):
    __slots__ = ("goal",)
    GOAL_FIELD_NUMBER: _ClassVar[int]
    goal: Goal
    def __init__(self, goal: _Optional[_Union[Goal, _Mapping]] = ...) -> None: ...

class ListMilestonesRequest(_message.Message):
    __slots__ = ("goal_id",)
    GOAL_ID_FIELD_NUMBER: _ClassVar[int]
    goal_id: str
    def __init__(self, goal_id: _Optional[str] = ...) -> None: ...

class ListMilestonesResponse(_message.Message):
    __slots__ = ("milestones",)
    MILESTONES_FIELD_NUMBER: _ClassVar[int]
    milestones: _containers.RepeatedCompositeFieldContainer[Milestone]
    def __init__(self, milestones: _Optional[_Iterable[_Union[Milestone, _Mapping]]] = ...) -> None: ...

class CreateMilestoneRequest(_message.Message):
    __slots__ = ("goal_id", "title", "criteria", "due_date", "linked_work_item_id", "prerequisite_milestone_ids")
    GOAL_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    CRITERIA_FIELD_NUMBER: _ClassVar[int]
    DUE_DATE_FIELD_NUMBER: _ClassVar[int]
    LINKED_WORK_ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    PREREQUISITE_MILESTONE_IDS_FIELD_NUMBER: _ClassVar[int]
    goal_id: str
    title: str
    criteria: str
    due_date: str
    linked_work_item_id: str
    prerequisite_milestone_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, goal_id: _Optional[str] = ..., title: _Optional[str] = ..., criteria: _Optional[str] = ..., due_date: _Optional[str] = ..., linked_work_item_id: _Optional[str] = ..., prerequisite_milestone_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class CreateMilestoneResponse(_message.Message):
    __slots__ = ("milestone",)
    MILESTONE_FIELD_NUMBER: _ClassVar[int]
    milestone: Milestone
    def __init__(self, milestone: _Optional[_Union[Milestone, _Mapping]] = ...) -> None: ...

class UpdateMilestoneStatusRequest(_message.Message):
    __slots__ = ("id", "status", "expected_revision")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    id: str
    status: str
    expected_revision: int
    def __init__(self, id: _Optional[str] = ..., status: _Optional[str] = ..., expected_revision: _Optional[int] = ...) -> None: ...

class UpdateMilestoneStatusResponse(_message.Message):
    __slots__ = ("milestone",)
    MILESTONE_FIELD_NUMBER: _ClassVar[int]
    milestone: Milestone
    def __init__(self, milestone: _Optional[_Union[Milestone, _Mapping]] = ...) -> None: ...
