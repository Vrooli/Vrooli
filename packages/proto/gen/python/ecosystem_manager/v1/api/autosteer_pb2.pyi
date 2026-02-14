from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from ecosystem_manager.v1.domain import autosteer_pb2 as _autosteer_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ProfileCreateRequest(_message.Message):
    __slots__ = ("id", "name", "description", "phases", "quality_gates", "tags")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    QUALITY_GATES_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    phases: _containers.RepeatedCompositeFieldContainer[_autosteer_pb2.SteerPhase]
    quality_gates: _containers.RepeatedCompositeFieldContainer[_autosteer_pb2.QualityGate]
    tags: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., phases: _Optional[_Iterable[_Union[_autosteer_pb2.SteerPhase, _Mapping]]] = ..., quality_gates: _Optional[_Iterable[_Union[_autosteer_pb2.QualityGate, _Mapping]]] = ..., tags: _Optional[_Iterable[str]] = ...) -> None: ...

class ProfileUpdateRequest(_message.Message):
    __slots__ = ("name", "description", "phases", "quality_gates", "tags")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    QUALITY_GATES_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    phases: _containers.RepeatedCompositeFieldContainer[_autosteer_pb2.SteerPhase]
    quality_gates: _containers.RepeatedCompositeFieldContainer[_autosteer_pb2.QualityGate]
    tags: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., phases: _Optional[_Iterable[_Union[_autosteer_pb2.SteerPhase, _Mapping]]] = ..., quality_gates: _Optional[_Iterable[_Union[_autosteer_pb2.QualityGate, _Mapping]]] = ..., tags: _Optional[_Iterable[str]] = ...) -> None: ...

class ProfileResponse(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: _autosteer_pb2.AutoSteerProfile
    def __init__(self, profile: _Optional[_Union[_autosteer_pb2.AutoSteerProfile, _Mapping]] = ...) -> None: ...

class ProfileListResponse(_message.Message):
    __slots__ = ("profiles", "count")
    PROFILES_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    profiles: _containers.RepeatedCompositeFieldContainer[_autosteer_pb2.AutoSteerProfile]
    count: int
    def __init__(self, profiles: _Optional[_Iterable[_Union[_autosteer_pb2.AutoSteerProfile, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class ExecutionStateResponse(_message.Message):
    __slots__ = ("state",)
    STATE_FIELD_NUMBER: _ClassVar[int]
    state: _autosteer_pb2.ProfileExecutionState
    def __init__(self, state: _Optional[_Union[_autosteer_pb2.ProfileExecutionState, _Mapping]] = ...) -> None: ...

class ExecutionResetRequest(_message.Message):
    __slots__ = ("task_id",)
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    def __init__(self, task_id: _Optional[str] = ...) -> None: ...

class ExecutionSeekRequest(_message.Message):
    __slots__ = ("task_id", "phase_index", "phase_iteration", "profile_id", "scenario_name")
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_INDEX_FIELD_NUMBER: _ClassVar[int]
    PHASE_ITERATION_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    phase_index: int
    phase_iteration: int
    profile_id: str
    scenario_name: str
    def __init__(self, task_id: _Optional[str] = ..., phase_index: _Optional[int] = ..., phase_iteration: _Optional[int] = ..., profile_id: _Optional[str] = ..., scenario_name: _Optional[str] = ...) -> None: ...

class HistoryListResponse(_message.Message):
    __slots__ = ("history", "count")
    HISTORY_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    history: _containers.RepeatedCompositeFieldContainer[_autosteer_pb2.ProfilePerformance]
    count: int
    def __init__(self, history: _Optional[_Iterable[_Union[_autosteer_pb2.ProfilePerformance, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class HistoryResponse(_message.Message):
    __slots__ = ("performance",)
    PERFORMANCE_FIELD_NUMBER: _ClassVar[int]
    performance: _autosteer_pb2.ProfilePerformance
    def __init__(self, performance: _Optional[_Union[_autosteer_pb2.ProfilePerformance, _Mapping]] = ...) -> None: ...

class FeedbackEntryRequest(_message.Message):
    __slots__ = ("category", "severity", "suggested_action", "comments", "metadata")
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    SUGGESTED_ACTION_FIELD_NUMBER: _ClassVar[int]
    COMMENTS_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    category: str
    severity: str
    suggested_action: str
    comments: str
    metadata: _struct_pb2.Struct
    def __init__(self, category: _Optional[str] = ..., severity: _Optional[str] = ..., suggested_action: _Optional[str] = ..., comments: _Optional[str] = ..., metadata: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class IterationRequest(_message.Message):
    __slots__ = ("execution_id", "output", "metrics")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    output: str
    metrics: _struct_pb2.Struct
    def __init__(self, execution_id: _Optional[str] = ..., output: _Optional[str] = ..., metrics: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class TemplateListResponse(_message.Message):
    __slots__ = ("templates", "count")
    TEMPLATES_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    templates: _containers.RepeatedCompositeFieldContainer[_autosteer_pb2.AutoSteerProfile]
    count: int
    def __init__(self, templates: _Optional[_Iterable[_Union[_autosteer_pb2.AutoSteerProfile, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...
