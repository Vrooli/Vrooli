from google.api import annotations_pb2 as _annotations_pb2
from buf.validate import validate_pb2 as _validate_pb2
from common.v1 import types_pb2 as _types_pb2
from agent_manager.v1.domain import types_pb2 as _types_pb2_1
from agent_manager.v1.domain import profile_pb2 as _profile_pb2
from agent_manager.v1.domain import task_pb2 as _task_pb2
from agent_manager.v1.domain import run_pb2 as _run_pb2
from agent_manager.v1.domain import events_pb2 as _events_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class HealthRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class CreateProfileRequest(_message.Message):
    __slots__ = ()
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: _profile_pb2.AgentProfile
    def __init__(self, profile: _Optional[_Union[_profile_pb2.AgentProfile, _Mapping]] = ...) -> None: ...

class CreateProfileResponse(_message.Message):
    __slots__ = ()
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: _profile_pb2.AgentProfile
    def __init__(self, profile: _Optional[_Union[_profile_pb2.AgentProfile, _Mapping]] = ...) -> None: ...

class GetProfileRequest(_message.Message):
    __slots__ = ()
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    def __init__(self, profile_id: _Optional[str] = ...) -> None: ...

class GetProfileResponse(_message.Message):
    __slots__ = ()
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: _profile_pb2.AgentProfile
    def __init__(self, profile: _Optional[_Union[_profile_pb2.AgentProfile, _Mapping]] = ...) -> None: ...

class ListProfilesRequest(_message.Message):
    __slots__ = ()
    RUNNER_TYPE_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    runner_type: _types_pb2_1.RunnerType
    limit: int
    offset: int
    def __init__(self, runner_type: _Optional[_Union[_types_pb2_1.RunnerType, str]] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListProfilesResponse(_message.Message):
    __slots__ = ()
    PROFILES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    HAS_MORE_FIELD_NUMBER: _ClassVar[int]
    profiles: _containers.RepeatedCompositeFieldContainer[_profile_pb2.AgentProfile]
    total: int
    has_more: bool
    def __init__(self, profiles: _Optional[_Iterable[_Union[_profile_pb2.AgentProfile, _Mapping]]] = ..., total: _Optional[int] = ..., has_more: _Optional[bool] = ...) -> None: ...

class UpdateProfileRequest(_message.Message):
    __slots__ = ()
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    profile: _profile_pb2.AgentProfile
    def __init__(self, profile_id: _Optional[str] = ..., profile: _Optional[_Union[_profile_pb2.AgentProfile, _Mapping]] = ...) -> None: ...

class UpdateProfileResponse(_message.Message):
    __slots__ = ()
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: _profile_pb2.AgentProfile
    def __init__(self, profile: _Optional[_Union[_profile_pb2.AgentProfile, _Mapping]] = ...) -> None: ...

class DeleteProfileRequest(_message.Message):
    __slots__ = ()
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    def __init__(self, profile_id: _Optional[str] = ...) -> None: ...

class DeleteProfileResponse(_message.Message):
    __slots__ = ()
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    def __init__(self, success: _Optional[bool] = ...) -> None: ...

class CreateTaskRequest(_message.Message):
    __slots__ = ()
    TASK_FIELD_NUMBER: _ClassVar[int]
    task: _task_pb2.Task
    def __init__(self, task: _Optional[_Union[_task_pb2.Task, _Mapping]] = ...) -> None: ...

class CreateTaskResponse(_message.Message):
    __slots__ = ()
    TASK_FIELD_NUMBER: _ClassVar[int]
    task: _task_pb2.Task
    def __init__(self, task: _Optional[_Union[_task_pb2.Task, _Mapping]] = ...) -> None: ...

class GetTaskRequest(_message.Message):
    __slots__ = ()
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    def __init__(self, task_id: _Optional[str] = ...) -> None: ...

class GetTaskResponse(_message.Message):
    __slots__ = ()
    TASK_FIELD_NUMBER: _ClassVar[int]
    task: _task_pb2.Task
    def __init__(self, task: _Optional[_Union[_task_pb2.Task, _Mapping]] = ...) -> None: ...

class ListTasksRequest(_message.Message):
    __slots__ = ()
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SCOPE_PREFIX_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    status: _types_pb2_1.TaskStatus
    scope_prefix: str
    limit: int
    offset: int
    def __init__(self, status: _Optional[_Union[_types_pb2_1.TaskStatus, str]] = ..., scope_prefix: _Optional[str] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListTasksResponse(_message.Message):
    __slots__ = ()
    TASKS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    HAS_MORE_FIELD_NUMBER: _ClassVar[int]
    tasks: _containers.RepeatedCompositeFieldContainer[_task_pb2.Task]
    total: int
    has_more: bool
    def __init__(self, tasks: _Optional[_Iterable[_Union[_task_pb2.Task, _Mapping]]] = ..., total: _Optional[int] = ..., has_more: _Optional[bool] = ...) -> None: ...

class UpdateTaskRequest(_message.Message):
    __slots__ = ()
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    task: _task_pb2.Task
    def __init__(self, task_id: _Optional[str] = ..., task: _Optional[_Union[_task_pb2.Task, _Mapping]] = ...) -> None: ...

class UpdateTaskResponse(_message.Message):
    __slots__ = ()
    TASK_FIELD_NUMBER: _ClassVar[int]
    task: _task_pb2.Task
    def __init__(self, task: _Optional[_Union[_task_pb2.Task, _Mapping]] = ...) -> None: ...

class DeleteTaskRequest(_message.Message):
    __slots__ = ()
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    def __init__(self, task_id: _Optional[str] = ...) -> None: ...

class DeleteTaskResponse(_message.Message):
    __slots__ = ()
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    def __init__(self, success: _Optional[bool] = ...) -> None: ...

class CancelTaskRequest(_message.Message):
    __slots__ = ()
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    def __init__(self, task_id: _Optional[str] = ...) -> None: ...

class CancelTaskResponse(_message.Message):
    __slots__ = ()
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    status: str
    def __init__(self, success: _Optional[bool] = ..., status: _Optional[str] = ...) -> None: ...

class CreateRunRequest(_message.Message):
    __slots__ = ()
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    TAG_FIELD_NUMBER: _ClassVar[int]
    RUN_MODE_FIELD_NUMBER: _ClassVar[int]
    INLINE_CONFIG_FIELD_NUMBER: _ClassVar[int]
    FORCE_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    agent_profile_id: str
    tag: str
    run_mode: _types_pb2_1.RunMode
    inline_config: _profile_pb2.RunConfig
    force: bool
    idempotency_key: str
    def __init__(self, task_id: _Optional[str] = ..., agent_profile_id: _Optional[str] = ..., tag: _Optional[str] = ..., run_mode: _Optional[_Union[_types_pb2_1.RunMode, str]] = ..., inline_config: _Optional[_Union[_profile_pb2.RunConfig, _Mapping]] = ..., force: _Optional[bool] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class CreateRunResponse(_message.Message):
    __slots__ = ()
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: _run_pb2.Run
    def __init__(self, run: _Optional[_Union[_run_pb2.Run, _Mapping]] = ...) -> None: ...

class GetRunRequest(_message.Message):
    __slots__ = ()
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    def __init__(self, run_id: _Optional[str] = ...) -> None: ...

class GetRunResponse(_message.Message):
    __slots__ = ()
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: _run_pb2.Run
    def __init__(self, run: _Optional[_Union[_run_pb2.Run, _Mapping]] = ...) -> None: ...

class GetRunByTagRequest(_message.Message):
    __slots__ = ()
    TAG_FIELD_NUMBER: _ClassVar[int]
    tag: str
    def __init__(self, tag: _Optional[str] = ...) -> None: ...

class GetRunByTagResponse(_message.Message):
    __slots__ = ()
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: _run_pb2.Run
    def __init__(self, run: _Optional[_Union[_run_pb2.Run, _Mapping]] = ...) -> None: ...

class ListRunsRequest(_message.Message):
    __slots__ = ()
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    TAG_PREFIX_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    status: _types_pb2_1.RunStatus
    task_id: str
    agent_profile_id: str
    tag_prefix: str
    limit: int
    offset: int
    def __init__(self, status: _Optional[_Union[_types_pb2_1.RunStatus, str]] = ..., task_id: _Optional[str] = ..., agent_profile_id: _Optional[str] = ..., tag_prefix: _Optional[str] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListRunsResponse(_message.Message):
    __slots__ = ()
    RUNS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    HAS_MORE_FIELD_NUMBER: _ClassVar[int]
    runs: _containers.RepeatedCompositeFieldContainer[_run_pb2.Run]
    total: int
    has_more: bool
    def __init__(self, runs: _Optional[_Iterable[_Union[_run_pb2.Run, _Mapping]]] = ..., total: _Optional[int] = ..., has_more: _Optional[bool] = ...) -> None: ...

class StopRunRequest(_message.Message):
    __slots__ = ()
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    def __init__(self, run_id: _Optional[str] = ...) -> None: ...

class StopRunResponse(_message.Message):
    __slots__ = ()
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class StopRunByTagRequest(_message.Message):
    __slots__ = ()
    TAG_FIELD_NUMBER: _ClassVar[int]
    tag: str
    def __init__(self, tag: _Optional[str] = ...) -> None: ...

class StopRunByTagResponse(_message.Message):
    __slots__ = ()
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TAG_FIELD_NUMBER: _ClassVar[int]
    status: str
    tag: str
    def __init__(self, status: _Optional[str] = ..., tag: _Optional[str] = ...) -> None: ...

class StopAllRunsRequest(_message.Message):
    __slots__ = ()
    TAG_PREFIX_FIELD_NUMBER: _ClassVar[int]
    FORCE_FIELD_NUMBER: _ClassVar[int]
    tag_prefix: str
    force: bool
    def __init__(self, tag_prefix: _Optional[str] = ..., force: _Optional[bool] = ...) -> None: ...

class StopAllRunsResponse(_message.Message):
    __slots__ = ()
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: _run_pb2.StopAllResult
    def __init__(self, result: _Optional[_Union[_run_pb2.StopAllResult, _Mapping]] = ...) -> None: ...

class GetRunEventsRequest(_message.Message):
    __slots__ = ()
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    AFTER_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    EVENT_TYPES_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    after_sequence: int
    limit: int
    event_types: _containers.RepeatedScalarFieldContainer[_types_pb2_1.RunEventType]
    def __init__(self, run_id: _Optional[str] = ..., after_sequence: _Optional[int] = ..., limit: _Optional[int] = ..., event_types: _Optional[_Iterable[_Union[_types_pb2_1.RunEventType, str]]] = ...) -> None: ...

class GetRunEventsResponse(_message.Message):
    __slots__ = ()
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    HAS_MORE_FIELD_NUMBER: _ClassVar[int]
    events: _containers.RepeatedCompositeFieldContainer[_events_pb2.RunEvent]
    has_more: bool
    def __init__(self, events: _Optional[_Iterable[_Union[_events_pb2.RunEvent, _Mapping]]] = ..., has_more: _Optional[bool] = ...) -> None: ...

class GetRunDiffRequest(_message.Message):
    __slots__ = ()
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    def __init__(self, run_id: _Optional[str] = ...) -> None: ...

class GetRunDiffResponse(_message.Message):
    __slots__ = ()
    DIFF_FIELD_NUMBER: _ClassVar[int]
    diff: _run_pb2.RunDiff
    def __init__(self, diff: _Optional[_Union[_run_pb2.RunDiff, _Mapping]] = ...) -> None: ...

class ApproveRunRequest(_message.Message):
    __slots__ = ()
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    COMMIT_MSG_FIELD_NUMBER: _ClassVar[int]
    FORCE_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    actor: str
    commit_msg: str
    force: bool
    def __init__(self, run_id: _Optional[str] = ..., actor: _Optional[str] = ..., commit_msg: _Optional[str] = ..., force: _Optional[bool] = ...) -> None: ...

class ApproveRunResponse(_message.Message):
    __slots__ = ()
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: _run_pb2.ApproveResult
    def __init__(self, result: _Optional[_Union[_run_pb2.ApproveResult, _Mapping]] = ...) -> None: ...

class RejectRunRequest(_message.Message):
    __slots__ = ()
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    actor: str
    reason: str
    def __init__(self, run_id: _Optional[str] = ..., actor: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class RejectRunResponse(_message.Message):
    __slots__ = ()
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class GetRunnerStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetRunnerStatusResponse(_message.Message):
    __slots__ = ()
    RUNNERS_FIELD_NUMBER: _ClassVar[int]
    runners: _containers.RepeatedCompositeFieldContainer[_run_pb2.RunnerStatus]
    def __init__(self, runners: _Optional[_Iterable[_Union[_run_pb2.RunnerStatus, _Mapping]]] = ...) -> None: ...

class ProbeRunnerRequest(_message.Message):
    __slots__ = ()
    RUNNER_TYPE_FIELD_NUMBER: _ClassVar[int]
    runner_type: _types_pb2_1.RunnerType
    def __init__(self, runner_type: _Optional[_Union[_types_pb2_1.RunnerType, str]] = ...) -> None: ...

class ProbeRunnerResponse(_message.Message):
    __slots__ = ()
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: _run_pb2.ProbeResult
    def __init__(self, result: _Optional[_Union[_run_pb2.ProbeResult, _Mapping]] = ...) -> None: ...
