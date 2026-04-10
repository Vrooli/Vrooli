from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SteerSet(_message.Message):
    __slots__ = ("skill_ids",)
    SKILL_IDS_FIELD_NUMBER: _ClassVar[int]
    skill_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, skill_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class Task(_message.Message):
    __slots__ = ("id", "title", "type", "operation", "target", "targets", "category", "priority", "effort_estimate", "urgency", "dependencies", "blocks", "related_scenarios", "related_resources", "status", "current_phase", "started_at", "completed_at", "cooldown_until", "completion_count", "last_completed_at", "validation_criteria", "created_by", "created_at", "updated_at", "tags", "notes", "results", "consecutive_completion_claims", "consecutive_failures", "processor_auto_requeue", "steer_set", "auto_steer_profile_id", "steering_queue")
    ID_FIELD_NUMBER: _ClassVar[int]
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
    BLOCKS_FIELD_NUMBER: _ClassVar[int]
    RELATED_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    RELATED_RESOURCES_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CURRENT_PHASE_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    COOLDOWN_UNTIL_FIELD_NUMBER: _ClassVar[int]
    COMPLETION_COUNT_FIELD_NUMBER: _ClassVar[int]
    LAST_COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_CRITERIA_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    CONSECUTIVE_COMPLETION_CLAIMS_FIELD_NUMBER: _ClassVar[int]
    CONSECUTIVE_FAILURES_FIELD_NUMBER: _ClassVar[int]
    PROCESSOR_AUTO_REQUEUE_FIELD_NUMBER: _ClassVar[int]
    STEER_SET_FIELD_NUMBER: _ClassVar[int]
    AUTO_STEER_PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    STEERING_QUEUE_FIELD_NUMBER: _ClassVar[int]
    id: str
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
    blocks: _containers.RepeatedScalarFieldContainer[str]
    related_scenarios: _containers.RepeatedScalarFieldContainer[str]
    related_resources: _containers.RepeatedScalarFieldContainer[str]
    status: str
    current_phase: str
    started_at: str
    completed_at: str
    cooldown_until: str
    completion_count: int
    last_completed_at: str
    validation_criteria: _containers.RepeatedScalarFieldContainer[str]
    created_by: str
    created_at: str
    updated_at: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    notes: str
    results: _struct_pb2.Struct
    consecutive_completion_claims: float
    consecutive_failures: int
    processor_auto_requeue: bool
    steer_set: _containers.RepeatedScalarFieldContainer[str]
    auto_steer_profile_id: str
    steering_queue: _containers.RepeatedCompositeFieldContainer[SteerSet]
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., type: _Optional[str] = ..., operation: _Optional[str] = ..., target: _Optional[str] = ..., targets: _Optional[_Iterable[str]] = ..., category: _Optional[str] = ..., priority: _Optional[str] = ..., effort_estimate: _Optional[str] = ..., urgency: _Optional[str] = ..., dependencies: _Optional[_Iterable[str]] = ..., blocks: _Optional[_Iterable[str]] = ..., related_scenarios: _Optional[_Iterable[str]] = ..., related_resources: _Optional[_Iterable[str]] = ..., status: _Optional[str] = ..., current_phase: _Optional[str] = ..., started_at: _Optional[str] = ..., completed_at: _Optional[str] = ..., cooldown_until: _Optional[str] = ..., completion_count: _Optional[int] = ..., last_completed_at: _Optional[str] = ..., validation_criteria: _Optional[_Iterable[str]] = ..., created_by: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., notes: _Optional[str] = ..., results: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., consecutive_completion_claims: _Optional[float] = ..., consecutive_failures: _Optional[int] = ..., processor_auto_requeue: _Optional[bool] = ..., steer_set: _Optional[_Iterable[str]] = ..., auto_steer_profile_id: _Optional[str] = ..., steering_queue: _Optional[_Iterable[_Union[SteerSet, _Mapping]]] = ...) -> None: ...

class ProcessInfo(_message.Message):
    __slots__ = ("task_id", "agent_tag", "run_id", "started_at", "is_timed_out")
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_TAG_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    IS_TIMED_OUT_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    agent_tag: str
    run_id: str
    started_at: str
    is_timed_out: bool
    def __init__(self, task_id: _Optional[str] = ..., agent_tag: _Optional[str] = ..., run_id: _Optional[str] = ..., started_at: _Optional[str] = ..., is_timed_out: _Optional[bool] = ...) -> None: ...

class ActiveTarget(_message.Message):
    __slots__ = ("target", "task_id", "status", "title")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    target: str
    task_id: str
    status: str
    title: str
    def __init__(self, target: _Optional[str] = ..., task_id: _Optional[str] = ..., status: _Optional[str] = ..., title: _Optional[str] = ...) -> None: ...
