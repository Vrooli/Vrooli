from buf.validate import validate_pb2 as _validate_pb2
from ecosystem_manager.v1.domain import task_pb2 as _task_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TaskCreateRequest(_message.Message):
    __slots__ = ("title", "type", "operation", "target", "targets", "category", "priority", "effort_estimate", "urgency", "dependencies", "related_scenarios", "related_resources", "tags", "notes", "steer_set", "auto_steer_profile_id", "steering_queue")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    EFFORT_ESTIMATE_FIELD_NUMBER: _ClassVar[int]
    URGENCY_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    RELATED_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    RELATED_RESOURCES_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    STEER_SET_FIELD_NUMBER: _ClassVar[int]
    AUTO_STEER_PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    STEERING_QUEUE_FIELD_NUMBER: _ClassVar[int]
    title: str
    type: str
    operation: str
    target: str
    targets: _containers.RepeatedScalarFieldContainer[str]
    category: str
    priority: str
    effort_estimate: str
    urgency: str
    dependencies: _containers.RepeatedScalarFieldContainer[str]
    related_scenarios: _containers.RepeatedScalarFieldContainer[str]
    related_resources: _containers.RepeatedScalarFieldContainer[str]
    tags: _containers.RepeatedScalarFieldContainer[str]
    notes: str
    steer_set: _containers.RepeatedScalarFieldContainer[str]
    auto_steer_profile_id: str
    steering_queue: _containers.RepeatedCompositeFieldContainer[_task_pb2.SteerSet]
    def __init__(self, title: _Optional[str] = ..., type: _Optional[str] = ..., operation: _Optional[str] = ..., target: _Optional[str] = ..., targets: _Optional[_Iterable[str]] = ..., category: _Optional[str] = ..., priority: _Optional[str] = ..., effort_estimate: _Optional[str] = ..., urgency: _Optional[str] = ..., dependencies: _Optional[_Iterable[str]] = ..., related_scenarios: _Optional[_Iterable[str]] = ..., related_resources: _Optional[_Iterable[str]] = ..., tags: _Optional[_Iterable[str]] = ..., notes: _Optional[str] = ..., steer_set: _Optional[_Iterable[str]] = ..., auto_steer_profile_id: _Optional[str] = ..., steering_queue: _Optional[_Iterable[_Union[_task_pb2.SteerSet, _Mapping]]] = ...) -> None: ...

class TaskUpdateRequest(_message.Message):
    __slots__ = ("title", "priority", "status", "target", "targets", "tags", "notes", "steer_set", "auto_steer_profile_id", "steering_queue")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    STEER_SET_FIELD_NUMBER: _ClassVar[int]
    AUTO_STEER_PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    STEERING_QUEUE_FIELD_NUMBER: _ClassVar[int]
    title: str
    priority: str
    status: str
    target: str
    targets: _containers.RepeatedScalarFieldContainer[str]
    tags: _containers.RepeatedScalarFieldContainer[str]
    notes: str
    steer_set: _containers.RepeatedScalarFieldContainer[str]
    auto_steer_profile_id: str
    steering_queue: _containers.RepeatedCompositeFieldContainer[_task_pb2.SteerSet]
    def __init__(self, title: _Optional[str] = ..., priority: _Optional[str] = ..., status: _Optional[str] = ..., target: _Optional[str] = ..., targets: _Optional[_Iterable[str]] = ..., tags: _Optional[_Iterable[str]] = ..., notes: _Optional[str] = ..., steer_set: _Optional[_Iterable[str]] = ..., auto_steer_profile_id: _Optional[str] = ..., steering_queue: _Optional[_Iterable[_Union[_task_pb2.SteerSet, _Mapping]]] = ...) -> None: ...

class TaskListResponse(_message.Message):
    __slots__ = ("tasks", "count", "total")
    TASKS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    tasks: _containers.RepeatedCompositeFieldContainer[_task_pb2.Task]
    count: int
    total: int
    def __init__(self, tasks: _Optional[_Iterable[_Union[_task_pb2.Task, _Mapping]]] = ..., count: _Optional[int] = ..., total: _Optional[int] = ...) -> None: ...

class TaskDetailResponse(_message.Message):
    __slots__ = ("task", "current_process", "auto_steer_phase_index")
    TASK_FIELD_NUMBER: _ClassVar[int]
    CURRENT_PROCESS_FIELD_NUMBER: _ClassVar[int]
    AUTO_STEER_PHASE_INDEX_FIELD_NUMBER: _ClassVar[int]
    task: _task_pb2.Task
    current_process: _task_pb2.ProcessInfo
    auto_steer_phase_index: int
    def __init__(self, task: _Optional[_Union[_task_pb2.Task, _Mapping]] = ..., current_process: _Optional[_Union[_task_pb2.ProcessInfo, _Mapping]] = ..., auto_steer_phase_index: _Optional[int] = ...) -> None: ...

class TaskActionResponse(_message.Message):
    __slots__ = ("success", "message", "task_id", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    message: str
    task_id: str
    error: str
    def __init__(self, success: _Optional[bool] = ..., message: _Optional[str] = ..., task_id: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...
