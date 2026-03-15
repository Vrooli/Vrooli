from agent_manager.v1.domain import events_pb2 as _events_pb2
from agent_manager.v1.domain import profile_pb2 as _profile_pb2
from agent_manager.v1.domain import run_pb2 as _run_pb2
from agent_manager.v1.domain import task_pb2 as _task_pb2
from agent_manager.v1.domain import types_pb2 as _types_pb2
from buf.validate import validate_pb2 as _validate_pb2
from common.v1 import types_pb2 as _types_pb2_1
from google.api import annotations_pb2 as _annotations_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PurgeTarget(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PURGE_TARGET_UNSPECIFIED: _ClassVar[PurgeTarget]
    PURGE_TARGET_PROFILES: _ClassVar[PurgeTarget]
    PURGE_TARGET_TASKS: _ClassVar[PurgeTarget]
    PURGE_TARGET_RUNS: _ClassVar[PurgeTarget]
PURGE_TARGET_UNSPECIFIED: PurgeTarget
PURGE_TARGET_PROFILES: PurgeTarget
PURGE_TARGET_TASKS: PurgeTarget
PURGE_TARGET_RUNS: PurgeTarget

class HealthRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class CreateProfileRequest(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: _profile_pb2.AgentProfile
    def __init__(self, profile: _Optional[_Union[_profile_pb2.AgentProfile, _Mapping]] = ...) -> None: ...

class CreateProfileResponse(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: _profile_pb2.AgentProfile
    def __init__(self, profile: _Optional[_Union[_profile_pb2.AgentProfile, _Mapping]] = ...) -> None: ...

class EnsureProfileRequest(_message.Message):
    __slots__ = ("profile_key", "defaults", "update_existing")
    PROFILE_KEY_FIELD_NUMBER: _ClassVar[int]
    DEFAULTS_FIELD_NUMBER: _ClassVar[int]
    UPDATE_EXISTING_FIELD_NUMBER: _ClassVar[int]
    profile_key: str
    defaults: _profile_pb2.AgentProfile
    update_existing: bool
    def __init__(self, profile_key: _Optional[str] = ..., defaults: _Optional[_Union[_profile_pb2.AgentProfile, _Mapping]] = ..., update_existing: _Optional[bool] = ...) -> None: ...

class EnsureProfileResponse(_message.Message):
    __slots__ = ("profile", "created", "updated")
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    UPDATED_FIELD_NUMBER: _ClassVar[int]
    profile: _profile_pb2.AgentProfile
    created: bool
    updated: bool
    def __init__(self, profile: _Optional[_Union[_profile_pb2.AgentProfile, _Mapping]] = ..., created: _Optional[bool] = ..., updated: _Optional[bool] = ...) -> None: ...

class GetProfileRequest(_message.Message):
    __slots__ = ("profile_id",)
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    def __init__(self, profile_id: _Optional[str] = ...) -> None: ...

class AvailableModel(_message.Message):
    __slots__ = ("id", "label", "description", "provider", "sources")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    SOURCES_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    description: str
    provider: str
    sources: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., description: _Optional[str] = ..., provider: _Optional[str] = ..., sources: _Optional[_Iterable[str]] = ...) -> None: ...

class GetProfileResponse(_message.Message):
    __slots__ = ("profile", "available_models", "model_presets")
    class ModelPresetsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_MODELS_FIELD_NUMBER: _ClassVar[int]
    MODEL_PRESETS_FIELD_NUMBER: _ClassVar[int]
    profile: _profile_pb2.AgentProfile
    available_models: _containers.RepeatedCompositeFieldContainer[AvailableModel]
    model_presets: _containers.ScalarMap[str, str]
    def __init__(self, profile: _Optional[_Union[_profile_pb2.AgentProfile, _Mapping]] = ..., available_models: _Optional[_Iterable[_Union[AvailableModel, _Mapping]]] = ..., model_presets: _Optional[_Mapping[str, str]] = ...) -> None: ...

class ListProfilesRequest(_message.Message):
    __slots__ = ("runner_type", "limit", "offset")
    RUNNER_TYPE_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    runner_type: _types_pb2.RunnerType
    limit: int
    offset: int
    def __init__(self, runner_type: _Optional[_Union[_types_pb2.RunnerType, str]] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListProfilesResponse(_message.Message):
    __slots__ = ("profiles", "total", "has_more")
    PROFILES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    HAS_MORE_FIELD_NUMBER: _ClassVar[int]
    profiles: _containers.RepeatedCompositeFieldContainer[_profile_pb2.AgentProfile]
    total: int
    has_more: bool
    def __init__(self, profiles: _Optional[_Iterable[_Union[_profile_pb2.AgentProfile, _Mapping]]] = ..., total: _Optional[int] = ..., has_more: _Optional[bool] = ...) -> None: ...

class UpdateProfileRequest(_message.Message):
    __slots__ = ("profile_id", "profile")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    profile: _profile_pb2.AgentProfile
    def __init__(self, profile_id: _Optional[str] = ..., profile: _Optional[_Union[_profile_pb2.AgentProfile, _Mapping]] = ...) -> None: ...

class UpdateProfileResponse(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: _profile_pb2.AgentProfile
    def __init__(self, profile: _Optional[_Union[_profile_pb2.AgentProfile, _Mapping]] = ...) -> None: ...

class DeleteProfileRequest(_message.Message):
    __slots__ = ("profile_id",)
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    def __init__(self, profile_id: _Optional[str] = ...) -> None: ...

class DeleteProfileResponse(_message.Message):
    __slots__ = ("success",)
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    def __init__(self, success: _Optional[bool] = ...) -> None: ...

class CreateTaskRequest(_message.Message):
    __slots__ = ("task",)
    TASK_FIELD_NUMBER: _ClassVar[int]
    task: _task_pb2.Task
    def __init__(self, task: _Optional[_Union[_task_pb2.Task, _Mapping]] = ...) -> None: ...

class CreateTaskResponse(_message.Message):
    __slots__ = ("task",)
    TASK_FIELD_NUMBER: _ClassVar[int]
    task: _task_pb2.Task
    def __init__(self, task: _Optional[_Union[_task_pb2.Task, _Mapping]] = ...) -> None: ...

class GetTaskRequest(_message.Message):
    __slots__ = ("task_id",)
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    def __init__(self, task_id: _Optional[str] = ...) -> None: ...

class GetTaskResponse(_message.Message):
    __slots__ = ("task",)
    TASK_FIELD_NUMBER: _ClassVar[int]
    task: _task_pb2.Task
    def __init__(self, task: _Optional[_Union[_task_pb2.Task, _Mapping]] = ...) -> None: ...

class ListTasksRequest(_message.Message):
    __slots__ = ("status", "scope_prefix", "limit", "offset")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SCOPE_PREFIX_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    status: _types_pb2.TaskStatus
    scope_prefix: str
    limit: int
    offset: int
    def __init__(self, status: _Optional[_Union[_types_pb2.TaskStatus, str]] = ..., scope_prefix: _Optional[str] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListTasksResponse(_message.Message):
    __slots__ = ("tasks", "total", "has_more")
    TASKS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    HAS_MORE_FIELD_NUMBER: _ClassVar[int]
    tasks: _containers.RepeatedCompositeFieldContainer[_task_pb2.Task]
    total: int
    has_more: bool
    def __init__(self, tasks: _Optional[_Iterable[_Union[_task_pb2.Task, _Mapping]]] = ..., total: _Optional[int] = ..., has_more: _Optional[bool] = ...) -> None: ...

class UpdateTaskRequest(_message.Message):
    __slots__ = ("task_id", "task")
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    task: _task_pb2.Task
    def __init__(self, task_id: _Optional[str] = ..., task: _Optional[_Union[_task_pb2.Task, _Mapping]] = ...) -> None: ...

class UpdateTaskResponse(_message.Message):
    __slots__ = ("task",)
    TASK_FIELD_NUMBER: _ClassVar[int]
    task: _task_pb2.Task
    def __init__(self, task: _Optional[_Union[_task_pb2.Task, _Mapping]] = ...) -> None: ...

class DeleteTaskRequest(_message.Message):
    __slots__ = ("task_id",)
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    def __init__(self, task_id: _Optional[str] = ...) -> None: ...

class DeleteTaskResponse(_message.Message):
    __slots__ = ("success",)
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    def __init__(self, success: _Optional[bool] = ...) -> None: ...

class CancelTaskRequest(_message.Message):
    __slots__ = ("task_id",)
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    def __init__(self, task_id: _Optional[str] = ...) -> None: ...

class CancelTaskResponse(_message.Message):
    __slots__ = ("success", "status")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    status: str
    def __init__(self, success: _Optional[bool] = ..., status: _Optional[str] = ...) -> None: ...

class ProfileRef(_message.Message):
    __slots__ = ("profile_key", "defaults")
    PROFILE_KEY_FIELD_NUMBER: _ClassVar[int]
    DEFAULTS_FIELD_NUMBER: _ClassVar[int]
    profile_key: str
    defaults: _profile_pb2.AgentProfile
    def __init__(self, profile_key: _Optional[str] = ..., defaults: _Optional[_Union[_profile_pb2.AgentProfile, _Mapping]] = ...) -> None: ...

class CreateRunRequest(_message.Message):
    __slots__ = ("task_id", "agent_profile_id", "tag", "run_mode", "inline_config", "force", "idempotency_key", "profile_ref", "prompt", "existing_sandbox_id")
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    TAG_FIELD_NUMBER: _ClassVar[int]
    RUN_MODE_FIELD_NUMBER: _ClassVar[int]
    INLINE_CONFIG_FIELD_NUMBER: _ClassVar[int]
    FORCE_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    PROFILE_REF_FIELD_NUMBER: _ClassVar[int]
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    EXISTING_SANDBOX_ID_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    agent_profile_id: str
    tag: str
    run_mode: _types_pb2.RunMode
    inline_config: _profile_pb2.RunConfigOverrides
    force: bool
    idempotency_key: str
    profile_ref: ProfileRef
    prompt: str
    existing_sandbox_id: str
    def __init__(self, task_id: _Optional[str] = ..., agent_profile_id: _Optional[str] = ..., tag: _Optional[str] = ..., run_mode: _Optional[_Union[_types_pb2.RunMode, str]] = ..., inline_config: _Optional[_Union[_profile_pb2.RunConfigOverrides, _Mapping]] = ..., force: _Optional[bool] = ..., idempotency_key: _Optional[str] = ..., profile_ref: _Optional[_Union[ProfileRef, _Mapping]] = ..., prompt: _Optional[str] = ..., existing_sandbox_id: _Optional[str] = ...) -> None: ...

class DeleteRunRequest(_message.Message):
    __slots__ = ("run_id",)
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    def __init__(self, run_id: _Optional[str] = ...) -> None: ...

class DeleteRunResponse(_message.Message):
    __slots__ = ("success",)
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    def __init__(self, success: _Optional[bool] = ...) -> None: ...

class CreateRunResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: _run_pb2.Run
    def __init__(self, run: _Optional[_Union[_run_pb2.Run, _Mapping]] = ...) -> None: ...

class GetRunRequest(_message.Message):
    __slots__ = ("run_id",)
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    def __init__(self, run_id: _Optional[str] = ...) -> None: ...

class GetRunResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: _run_pb2.Run
    def __init__(self, run: _Optional[_Union[_run_pb2.Run, _Mapping]] = ...) -> None: ...

class GetRunByTagRequest(_message.Message):
    __slots__ = ("tag",)
    TAG_FIELD_NUMBER: _ClassVar[int]
    tag: str
    def __init__(self, tag: _Optional[str] = ...) -> None: ...

class GetRunByTagResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: _run_pb2.Run
    def __init__(self, run: _Optional[_Union[_run_pb2.Run, _Mapping]] = ...) -> None: ...

class ListRunsRequest(_message.Message):
    __slots__ = ("status", "task_id", "agent_profile_id", "tag_prefix", "limit", "offset")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    TAG_PREFIX_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    status: _types_pb2.RunStatus
    task_id: str
    agent_profile_id: str
    tag_prefix: str
    limit: int
    offset: int
    def __init__(self, status: _Optional[_Union[_types_pb2.RunStatus, str]] = ..., task_id: _Optional[str] = ..., agent_profile_id: _Optional[str] = ..., tag_prefix: _Optional[str] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListRunsResponse(_message.Message):
    __slots__ = ("runs", "total", "has_more")
    RUNS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    HAS_MORE_FIELD_NUMBER: _ClassVar[int]
    runs: _containers.RepeatedCompositeFieldContainer[_run_pb2.Run]
    total: int
    has_more: bool
    def __init__(self, runs: _Optional[_Iterable[_Union[_run_pb2.Run, _Mapping]]] = ..., total: _Optional[int] = ..., has_more: _Optional[bool] = ...) -> None: ...

class StopRunRequest(_message.Message):
    __slots__ = ("run_id",)
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    def __init__(self, run_id: _Optional[str] = ...) -> None: ...

class StopRunResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class StopRunByTagRequest(_message.Message):
    __slots__ = ("tag",)
    TAG_FIELD_NUMBER: _ClassVar[int]
    tag: str
    def __init__(self, tag: _Optional[str] = ...) -> None: ...

class StopRunByTagResponse(_message.Message):
    __slots__ = ("status", "tag")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TAG_FIELD_NUMBER: _ClassVar[int]
    status: str
    tag: str
    def __init__(self, status: _Optional[str] = ..., tag: _Optional[str] = ...) -> None: ...

class StopAllRunsRequest(_message.Message):
    __slots__ = ("tag_prefix", "force")
    TAG_PREFIX_FIELD_NUMBER: _ClassVar[int]
    FORCE_FIELD_NUMBER: _ClassVar[int]
    tag_prefix: str
    force: bool
    def __init__(self, tag_prefix: _Optional[str] = ..., force: _Optional[bool] = ...) -> None: ...

class StopAllRunsResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: _run_pb2.StopAllResult
    def __init__(self, result: _Optional[_Union[_run_pb2.StopAllResult, _Mapping]] = ...) -> None: ...

class GetRunEventsRequest(_message.Message):
    __slots__ = ("run_id", "after_sequence", "limit", "event_types")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    AFTER_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    EVENT_TYPES_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    after_sequence: int
    limit: int
    event_types: _containers.RepeatedScalarFieldContainer[_types_pb2.RunEventType]
    def __init__(self, run_id: _Optional[str] = ..., after_sequence: _Optional[int] = ..., limit: _Optional[int] = ..., event_types: _Optional[_Iterable[_Union[_types_pb2.RunEventType, str]]] = ...) -> None: ...

class GetRunEventsResponse(_message.Message):
    __slots__ = ("events", "has_more")
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    HAS_MORE_FIELD_NUMBER: _ClassVar[int]
    events: _containers.RepeatedCompositeFieldContainer[_events_pb2.RunEvent]
    has_more: bool
    def __init__(self, events: _Optional[_Iterable[_Union[_events_pb2.RunEvent, _Mapping]]] = ..., has_more: _Optional[bool] = ...) -> None: ...

class GetRunDiffRequest(_message.Message):
    __slots__ = ("run_id",)
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    def __init__(self, run_id: _Optional[str] = ...) -> None: ...

class GetRunDiffResponse(_message.Message):
    __slots__ = ("diff",)
    DIFF_FIELD_NUMBER: _ClassVar[int]
    diff: _run_pb2.RunDiff
    def __init__(self, diff: _Optional[_Union[_run_pb2.RunDiff, _Mapping]] = ...) -> None: ...

class ApproveRunRequest(_message.Message):
    __slots__ = ("run_id", "actor", "commit_msg", "force")
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
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: _run_pb2.ApproveResult
    def __init__(self, result: _Optional[_Union[_run_pb2.ApproveResult, _Mapping]] = ...) -> None: ...

class RejectRunRequest(_message.Message):
    __slots__ = ("run_id", "actor", "reason")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    actor: str
    reason: str
    def __init__(self, run_id: _Optional[str] = ..., actor: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class RejectRunResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class PartialApproveRunRequest(_message.Message):
    __slots__ = ("run_id", "file_ids", "actor", "commit_msg")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    FILE_IDS_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    COMMIT_MSG_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    file_ids: _containers.RepeatedScalarFieldContainer[str]
    actor: str
    commit_msg: str
    def __init__(self, run_id: _Optional[str] = ..., file_ids: _Optional[_Iterable[str]] = ..., actor: _Optional[str] = ..., commit_msg: _Optional[str] = ...) -> None: ...

class PartialApproveRunResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: _run_pb2.ApproveResult
    def __init__(self, result: _Optional[_Union[_run_pb2.ApproveResult, _Mapping]] = ...) -> None: ...

class GetRunnerStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetRunnerStatusResponse(_message.Message):
    __slots__ = ("runners",)
    RUNNERS_FIELD_NUMBER: _ClassVar[int]
    runners: _containers.RepeatedCompositeFieldContainer[_run_pb2.RunnerStatus]
    def __init__(self, runners: _Optional[_Iterable[_Union[_run_pb2.RunnerStatus, _Mapping]]] = ...) -> None: ...

class ProbeRunnerRequest(_message.Message):
    __slots__ = ("runner_type",)
    RUNNER_TYPE_FIELD_NUMBER: _ClassVar[int]
    runner_type: _types_pb2.RunnerType
    def __init__(self, runner_type: _Optional[_Union[_types_pb2.RunnerType, str]] = ...) -> None: ...

class ProbeRunnerResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: _run_pb2.ProbeResult
    def __init__(self, result: _Optional[_Union[_run_pb2.ProbeResult, _Mapping]] = ...) -> None: ...

class PurgeDataRequest(_message.Message):
    __slots__ = ("pattern", "targets", "dry_run")
    PATTERN_FIELD_NUMBER: _ClassVar[int]
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    pattern: str
    targets: _containers.RepeatedScalarFieldContainer[PurgeTarget]
    dry_run: bool
    def __init__(self, pattern: _Optional[str] = ..., targets: _Optional[_Iterable[_Union[PurgeTarget, str]]] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class PurgeCounts(_message.Message):
    __slots__ = ("profiles", "tasks", "runs")
    PROFILES_FIELD_NUMBER: _ClassVar[int]
    TASKS_FIELD_NUMBER: _ClassVar[int]
    RUNS_FIELD_NUMBER: _ClassVar[int]
    profiles: int
    tasks: int
    runs: int
    def __init__(self, profiles: _Optional[int] = ..., tasks: _Optional[int] = ..., runs: _Optional[int] = ...) -> None: ...

class PurgeDataResponse(_message.Message):
    __slots__ = ("matched", "deleted", "dry_run")
    MATCHED_FIELD_NUMBER: _ClassVar[int]
    DELETED_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    matched: PurgeCounts
    deleted: PurgeCounts
    dry_run: bool
    def __init__(self, matched: _Optional[_Union[PurgeCounts, _Mapping]] = ..., deleted: _Optional[_Union[PurgeCounts, _Mapping]] = ..., dry_run: _Optional[bool] = ...) -> None: ...
