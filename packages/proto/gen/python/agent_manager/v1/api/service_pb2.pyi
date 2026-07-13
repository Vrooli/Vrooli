import datetime

from agent_manager.v1.domain import events_pb2 as _events_pb2
from agent_manager.v1.domain import profile_pb2 as _profile_pb2
from agent_manager.v1.domain import run_pb2 as _run_pb2
from agent_manager.v1.domain import task_pb2 as _task_pb2
from agent_manager.v1.domain import types_pb2 as _types_pb2
from buf.validate import validate_pb2 as _validate_pb2
from common.v1 import types_pb2 as _types_pb2_1
from google.api import annotations_pb2 as _annotations_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ProfileReconcileStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PROFILE_RECONCILE_STATUS_UNSPECIFIED: _ClassVar[ProfileReconcileStatus]
    PROFILE_RECONCILE_STATUS_CREATED: _ClassVar[ProfileReconcileStatus]
    PROFILE_RECONCILE_STATUS_UPDATED: _ClassVar[ProfileReconcileStatus]
    PROFILE_RECONCILE_STATUS_UNCHANGED: _ClassVar[ProfileReconcileStatus]
    PROFILE_RECONCILE_STATUS_SKIPPED: _ClassVar[ProfileReconcileStatus]
    PROFILE_RECONCILE_STATUS_CONFLICTED_LOCAL_OVERRIDE: _ClassVar[ProfileReconcileStatus]
    PROFILE_RECONCILE_STATUS_FAILED_VALIDATION: _ClassVar[ProfileReconcileStatus]

class PurgeTarget(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PURGE_TARGET_UNSPECIFIED: _ClassVar[PurgeTarget]
    PURGE_TARGET_PROFILES: _ClassVar[PurgeTarget]
    PURGE_TARGET_TASKS: _ClassVar[PurgeTarget]
    PURGE_TARGET_RUNS: _ClassVar[PurgeTarget]
PROFILE_RECONCILE_STATUS_UNSPECIFIED: ProfileReconcileStatus
PROFILE_RECONCILE_STATUS_CREATED: ProfileReconcileStatus
PROFILE_RECONCILE_STATUS_UPDATED: ProfileReconcileStatus
PROFILE_RECONCILE_STATUS_UNCHANGED: ProfileReconcileStatus
PROFILE_RECONCILE_STATUS_SKIPPED: ProfileReconcileStatus
PROFILE_RECONCILE_STATUS_CONFLICTED_LOCAL_OVERRIDE: ProfileReconcileStatus
PROFILE_RECONCILE_STATUS_FAILED_VALIDATION: ProfileReconcileStatus
PURGE_TARGET_UNSPECIFIED: PurgeTarget
PURGE_TARGET_PROFILES: PurgeTarget
PURGE_TARGET_TASKS: PurgeTarget
PURGE_TARGET_RUNS: PurgeTarget

class HealthRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class HealthResponse(_message.Message):
    __slots__ = ("status", "service", "timestamp", "readiness", "version", "dependencies", "metrics")
    class DependenciesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2_1.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2_1.JsonValue, _Mapping]] = ...) -> None: ...
    class MetricsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _types_pb2_1.JsonValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_types_pb2_1.JsonValue, _Mapping]] = ...) -> None: ...
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SERVICE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    READINESS_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    status: _types_pb2_1.HealthStatus
    service: str
    timestamp: str
    readiness: bool
    version: str
    dependencies: _containers.MessageMap[str, _types_pb2_1.JsonValue]
    metrics: _containers.MessageMap[str, _types_pb2_1.JsonValue]
    def __init__(self, status: _Optional[_Union[_types_pb2_1.HealthStatus, str]] = ..., service: _Optional[str] = ..., timestamp: _Optional[str] = ..., readiness: _Optional[bool] = ..., version: _Optional[str] = ..., dependencies: _Optional[_Mapping[str, _types_pb2_1.JsonValue]] = ..., metrics: _Optional[_Mapping[str, _types_pb2_1.JsonValue]] = ...) -> None: ...

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

class ReconcileScenarioProfilesRequest(_message.Message):
    __slots__ = ("scenario", "dry_run")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    dry_run: bool
    def __init__(self, scenario: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class ProfileReconcileResult(_message.Message):
    __slots__ = ("profile_key", "source_path", "source_hash", "profile_id", "status", "message")
    PROFILE_KEY_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    SOURCE_HASH_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    profile_key: str
    source_path: str
    source_hash: str
    profile_id: str
    status: ProfileReconcileStatus
    message: str
    def __init__(self, profile_key: _Optional[str] = ..., source_path: _Optional[str] = ..., source_hash: _Optional[str] = ..., profile_id: _Optional[str] = ..., status: _Optional[_Union[ProfileReconcileStatus, str]] = ..., message: _Optional[str] = ...) -> None: ...

class ReconcileScenarioProfilesResponse(_message.Message):
    __slots__ = ("scenario", "results", "created", "updated", "unchanged", "skipped", "conflicted", "failed", "dry_run")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    UPDATED_FIELD_NUMBER: _ClassVar[int]
    UNCHANGED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    CONFLICTED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    results: _containers.RepeatedCompositeFieldContainer[ProfileReconcileResult]
    created: int
    updated: int
    unchanged: int
    skipped: int
    conflicted: int
    failed: int
    dry_run: bool
    def __init__(self, scenario: _Optional[str] = ..., results: _Optional[_Iterable[_Union[ProfileReconcileResult, _Mapping]]] = ..., created: _Optional[int] = ..., updated: _Optional[int] = ..., unchanged: _Optional[int] = ..., skipped: _Optional[int] = ..., conflicted: _Optional[int] = ..., failed: _Optional[int] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class GetProfileRequest(_message.Message):
    __slots__ = ("profile_id",)
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    def __init__(self, profile_id: _Optional[str] = ...) -> None: ...

class GetProfileResponse(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: _profile_pb2.AgentProfile
    def __init__(self, profile: _Optional[_Union[_profile_pb2.AgentProfile, _Mapping]] = ...) -> None: ...

class ListProfilesRequest(_message.Message):
    __slots__ = ("limit", "offset")
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    limit: int
    offset: int
    def __init__(self, limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

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
    __slots__ = ("profile_key", "defaults", "update_existing")
    PROFILE_KEY_FIELD_NUMBER: _ClassVar[int]
    DEFAULTS_FIELD_NUMBER: _ClassVar[int]
    UPDATE_EXISTING_FIELD_NUMBER: _ClassVar[int]
    profile_key: str
    defaults: _profile_pb2.AgentProfile
    update_existing: bool
    def __init__(self, profile_key: _Optional[str] = ..., defaults: _Optional[_Union[_profile_pb2.AgentProfile, _Mapping]] = ..., update_existing: _Optional[bool] = ...) -> None: ...

class CreateRunRequest(_message.Message):
    __slots__ = ("task_id", "agent_profile_id", "tag", "run_mode", "inline_config", "force", "idempotency_key", "profile_ref", "prompt", "existing_sandbox_id", "environment", "conversation_id", "parent_run_id", "execution_mode")
    class EnvironmentEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
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
    ENVIRONMENT_FIELD_NUMBER: _ClassVar[int]
    CONVERSATION_ID_FIELD_NUMBER: _ClassVar[int]
    PARENT_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_MODE_FIELD_NUMBER: _ClassVar[int]
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
    environment: _containers.ScalarMap[str, str]
    conversation_id: str
    parent_run_id: str
    execution_mode: _types_pb2.ExecutionMode
    def __init__(self, task_id: _Optional[str] = ..., agent_profile_id: _Optional[str] = ..., tag: _Optional[str] = ..., run_mode: _Optional[_Union[_types_pb2.RunMode, str]] = ..., inline_config: _Optional[_Union[_profile_pb2.RunConfigOverrides, _Mapping]] = ..., force: _Optional[bool] = ..., idempotency_key: _Optional[str] = ..., profile_ref: _Optional[_Union[ProfileRef, _Mapping]] = ..., prompt: _Optional[str] = ..., existing_sandbox_id: _Optional[str] = ..., environment: _Optional[_Mapping[str, str]] = ..., conversation_id: _Optional[str] = ..., parent_run_id: _Optional[str] = ..., execution_mode: _Optional[_Union[_types_pb2.ExecutionMode, str]] = ...) -> None: ...

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
    __slots__ = ("run", "queue_depth", "active_count", "starting_count")
    RUN_FIELD_NUMBER: _ClassVar[int]
    QUEUE_DEPTH_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_COUNT_FIELD_NUMBER: _ClassVar[int]
    STARTING_COUNT_FIELD_NUMBER: _ClassVar[int]
    run: _run_pb2.Run
    queue_depth: int
    active_count: int
    starting_count: int
    def __init__(self, run: _Optional[_Union[_run_pb2.Run, _Mapping]] = ..., queue_depth: _Optional[int] = ..., active_count: _Optional[int] = ..., starting_count: _Optional[int] = ...) -> None: ...

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
    __slots__ = ("status", "run")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    RUN_FIELD_NUMBER: _ClassVar[int]
    status: str
    run: _run_pb2.Run
    def __init__(self, status: _Optional[str] = ..., run: _Optional[_Union[_run_pb2.Run, _Mapping]] = ...) -> None: ...

class StopRunByTagRequest(_message.Message):
    __slots__ = ("tag",)
    TAG_FIELD_NUMBER: _ClassVar[int]
    tag: str
    def __init__(self, tag: _Optional[str] = ...) -> None: ...

class StopRunByTagResponse(_message.Message):
    __slots__ = ("status", "tag", "run")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TAG_FIELD_NUMBER: _ClassVar[int]
    RUN_FIELD_NUMBER: _ClassVar[int]
    status: str
    tag: str
    run: _run_pb2.Run
    def __init__(self, status: _Optional[str] = ..., tag: _Optional[str] = ..., run: _Optional[_Union[_run_pb2.Run, _Mapping]] = ...) -> None: ...

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

class QuiesceScenarioRequest(_message.Message):
    __slots__ = ("scenario", "scope_prefix", "tag_prefix", "exclude_run_id", "timeout", "force")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SCOPE_PREFIX_FIELD_NUMBER: _ClassVar[int]
    TAG_PREFIX_FIELD_NUMBER: _ClassVar[int]
    EXCLUDE_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    FORCE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    scope_prefix: str
    tag_prefix: str
    exclude_run_id: str
    timeout: str
    force: bool
    def __init__(self, scenario: _Optional[str] = ..., scope_prefix: _Optional[str] = ..., tag_prefix: _Optional[str] = ..., exclude_run_id: _Optional[str] = ..., timeout: _Optional[str] = ..., force: _Optional[bool] = ...) -> None: ...

class QuiesceScenarioResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: QuiesceResult
    def __init__(self, result: _Optional[_Union[QuiesceResult, _Mapping]] = ...) -> None: ...

class QuiesceResult(_message.Message):
    __slots__ = ("scenario", "drained", "aborted", "initial", "in_flight", "cancelled", "waited_ms", "reason")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DRAINED_FIELD_NUMBER: _ClassVar[int]
    ABORTED_FIELD_NUMBER: _ClassVar[int]
    INITIAL_FIELD_NUMBER: _ClassVar[int]
    IN_FLIGHT_FIELD_NUMBER: _ClassVar[int]
    CANCELLED_FIELD_NUMBER: _ClassVar[int]
    WAITED_MS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    drained: bool
    aborted: bool
    initial: int
    in_flight: _containers.RepeatedCompositeFieldContainer[QuiesceRunRef]
    cancelled: _containers.RepeatedCompositeFieldContainer[QuiesceRunRef]
    waited_ms: int
    reason: str
    def __init__(self, scenario: _Optional[str] = ..., drained: _Optional[bool] = ..., aborted: _Optional[bool] = ..., initial: _Optional[int] = ..., in_flight: _Optional[_Iterable[_Union[QuiesceRunRef, _Mapping]]] = ..., cancelled: _Optional[_Iterable[_Union[QuiesceRunRef, _Mapping]]] = ..., waited_ms: _Optional[int] = ..., reason: _Optional[str] = ...) -> None: ...

class QuiesceRunRef(_message.Message):
    __slots__ = ("id", "tag", "status", "scope_path")
    ID_FIELD_NUMBER: _ClassVar[int]
    TAG_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SCOPE_PATH_FIELD_NUMBER: _ClassVar[int]
    id: str
    tag: str
    status: str
    scope_path: str
    def __init__(self, id: _Optional[str] = ..., tag: _Optional[str] = ..., status: _Optional[str] = ..., scope_path: _Optional[str] = ...) -> None: ...

class RecoverRunRequest(_message.Message):
    __slots__ = ("run_id",)
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    def __init__(self, run_id: _Optional[str] = ...) -> None: ...

class RecoverRunResponse(_message.Message):
    __slots__ = ("run", "recovered", "idempotent", "message")
    RUN_FIELD_NUMBER: _ClassVar[int]
    RECOVERED_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENT_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    run: _run_pb2.Run
    recovered: bool
    idempotent: bool
    message: str
    def __init__(self, run: _Optional[_Union[_run_pb2.Run, _Mapping]] = ..., recovered: _Optional[bool] = ..., idempotent: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...

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

class RolePolicyDiagnostic(_message.Message):
    __slots__ = ("code", "message", "cause")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    CAUSE_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    cause: str
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., cause: _Optional[str] = ...) -> None: ...

class RolePolicyRequirement(_message.Message):
    __slots__ = ("required", "reason")
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    required: bool
    reason: str
    def __init__(self, required: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class RolePolicyReloadAttempt(_message.Message):
    __slots__ = ("attempted_at", "succeeded", "digest", "diagnostic")
    ATTEMPTED_AT_FIELD_NUMBER: _ClassVar[int]
    SUCCEEDED_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTIC_FIELD_NUMBER: _ClassVar[int]
    attempted_at: _timestamp_pb2.Timestamp
    succeeded: bool
    digest: str
    diagnostic: RolePolicyDiagnostic
    def __init__(self, attempted_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., succeeded: _Optional[bool] = ..., digest: _Optional[str] = ..., diagnostic: _Optional[_Union[RolePolicyDiagnostic, _Mapping]] = ...) -> None: ...

class RolePolicyStatus(_message.Message):
    __slots__ = ("path", "requirement", "ready", "active_digest", "activated_at", "last_reload_attempt")
    PATH_FIELD_NUMBER: _ClassVar[int]
    REQUIREMENT_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    ACTIVATED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_RELOAD_ATTEMPT_FIELD_NUMBER: _ClassVar[int]
    path: str
    requirement: RolePolicyRequirement
    ready: bool
    active_digest: str
    activated_at: _timestamp_pb2.Timestamp
    last_reload_attempt: RolePolicyReloadAttempt
    def __init__(self, path: _Optional[str] = ..., requirement: _Optional[_Union[RolePolicyRequirement, _Mapping]] = ..., ready: _Optional[bool] = ..., active_digest: _Optional[str] = ..., activated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_reload_attempt: _Optional[_Union[RolePolicyReloadAttempt, _Mapping]] = ...) -> None: ...

class RolePolicyCatalogMetadata(_message.Message):
    __slots__ = ("catalog_id", "updated_at")
    CATALOG_ID_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    catalog_id: str
    updated_at: str
    def __init__(self, catalog_id: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class RolePolicyCandidate(_message.Message):
    __slots__ = ("runner_type", "resource_role")
    RUNNER_TYPE_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_ROLE_FIELD_NUMBER: _ClassVar[int]
    runner_type: _types_pb2.RunnerType
    resource_role: str
    def __init__(self, runner_type: _Optional[_Union[_types_pb2.RunnerType, str]] = ..., resource_role: _Optional[str] = ...) -> None: ...

class RolePolicyDefinition(_message.Message):
    __slots__ = ("role_ref", "intent", "description", "candidates")
    ROLE_REF_FIELD_NUMBER: _ClassVar[int]
    INTENT_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    role_ref: str
    intent: str
    description: str
    candidates: _containers.RepeatedCompositeFieldContainer[RolePolicyCandidate]
    def __init__(self, role_ref: _Optional[str] = ..., intent: _Optional[str] = ..., description: _Optional[str] = ..., candidates: _Optional[_Iterable[_Union[RolePolicyCandidate, _Mapping]]] = ...) -> None: ...

class RolePolicyCatalog(_message.Message):
    __slots__ = ("schema_version", "metadata", "default_role", "roles")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_ROLE_FIELD_NUMBER: _ClassVar[int]
    ROLES_FIELD_NUMBER: _ClassVar[int]
    schema_version: int
    metadata: RolePolicyCatalogMetadata
    default_role: str
    roles: _containers.RepeatedCompositeFieldContainer[RolePolicyDefinition]
    def __init__(self, schema_version: _Optional[int] = ..., metadata: _Optional[_Union[RolePolicyCatalogMetadata, _Mapping]] = ..., default_role: _Optional[str] = ..., roles: _Optional[_Iterable[_Union[RolePolicyDefinition, _Mapping]]] = ...) -> None: ...

class GetRolePolicyStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetRolePolicyStatusResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: RolePolicyStatus
    def __init__(self, status: _Optional[_Union[RolePolicyStatus, _Mapping]] = ...) -> None: ...

class GetRolePolicyCatalogRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetRolePolicyCatalogResponse(_message.Message):
    __slots__ = ("status", "catalog")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CATALOG_FIELD_NUMBER: _ClassVar[int]
    status: RolePolicyStatus
    catalog: RolePolicyCatalog
    def __init__(self, status: _Optional[_Union[RolePolicyStatus, _Mapping]] = ..., catalog: _Optional[_Union[RolePolicyCatalog, _Mapping]] = ...) -> None: ...

class ValidateRolePolicyCatalogRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ValidateRolePolicyCatalogResponse(_message.Message):
    __slots__ = ("valid", "candidate_digest", "active_digest", "diagnostic")
    VALID_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTIC_FIELD_NUMBER: _ClassVar[int]
    valid: bool
    candidate_digest: str
    active_digest: str
    diagnostic: RolePolicyDiagnostic
    def __init__(self, valid: _Optional[bool] = ..., candidate_digest: _Optional[str] = ..., active_digest: _Optional[str] = ..., diagnostic: _Optional[_Union[RolePolicyDiagnostic, _Mapping]] = ...) -> None: ...

class ReloadRolePolicyCatalogRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ReloadRolePolicyCatalogResponse(_message.Message):
    __slots__ = ("activated", "status", "diagnostic")
    ACTIVATED_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTIC_FIELD_NUMBER: _ClassVar[int]
    activated: bool
    status: RolePolicyStatus
    diagnostic: RolePolicyDiagnostic
    def __init__(self, activated: _Optional[bool] = ..., status: _Optional[_Union[RolePolicyStatus, _Mapping]] = ..., diagnostic: _Optional[_Union[RolePolicyDiagnostic, _Mapping]] = ...) -> None: ...

class ExplainRolePolicyRequest(_message.Message):
    __slots__ = ("profile_id", "run_id")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    run_id: str
    def __init__(self, profile_id: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class ExplainRolePolicyResponse(_message.Message):
    __slots__ = ("target_type", "target_id", "snapshot", "summary", "historical_without_snapshot")
    TARGET_TYPE_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    HISTORICAL_WITHOUT_SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    target_type: str
    target_id: str
    snapshot: _profile_pb2.ExecutionPolicySnapshot
    summary: str
    historical_without_snapshot: bool
    def __init__(self, target_type: _Optional[str] = ..., target_id: _Optional[str] = ..., snapshot: _Optional[_Union[_profile_pb2.ExecutionPolicySnapshot, _Mapping]] = ..., summary: _Optional[str] = ..., historical_without_snapshot: _Optional[bool] = ...) -> None: ...

class PermissionPolicyDiagnostic(_message.Message):
    __slots__ = ("code", "message", "cause")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    CAUSE_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    cause: str
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., cause: _Optional[str] = ...) -> None: ...

class PermissionPolicyRequirement(_message.Message):
    __slots__ = ("required", "reason")
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    required: bool
    reason: str
    def __init__(self, required: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class PermissionPolicyReloadAttempt(_message.Message):
    __slots__ = ("attempted_at", "succeeded", "digest", "diagnostic")
    ATTEMPTED_AT_FIELD_NUMBER: _ClassVar[int]
    SUCCEEDED_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTIC_FIELD_NUMBER: _ClassVar[int]
    attempted_at: _timestamp_pb2.Timestamp
    succeeded: bool
    digest: str
    diagnostic: PermissionPolicyDiagnostic
    def __init__(self, attempted_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., succeeded: _Optional[bool] = ..., digest: _Optional[str] = ..., diagnostic: _Optional[_Union[PermissionPolicyDiagnostic, _Mapping]] = ...) -> None: ...

class PermissionPolicyStatus(_message.Message):
    __slots__ = ("path", "requirement", "ready", "active_digest", "activated_at", "last_reload_attempt")
    PATH_FIELD_NUMBER: _ClassVar[int]
    REQUIREMENT_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    ACTIVATED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_RELOAD_ATTEMPT_FIELD_NUMBER: _ClassVar[int]
    path: str
    requirement: PermissionPolicyRequirement
    ready: bool
    active_digest: str
    activated_at: _timestamp_pb2.Timestamp
    last_reload_attempt: PermissionPolicyReloadAttempt
    def __init__(self, path: _Optional[str] = ..., requirement: _Optional[_Union[PermissionPolicyRequirement, _Mapping]] = ..., ready: _Optional[bool] = ..., active_digest: _Optional[str] = ..., activated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_reload_attempt: _Optional[_Union[PermissionPolicyReloadAttempt, _Mapping]] = ...) -> None: ...

class PermissionPolicyCatalogMetadata(_message.Message):
    __slots__ = ("catalog_id", "updated_at")
    CATALOG_ID_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    catalog_id: str
    updated_at: str
    def __init__(self, catalog_id: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class PermissionPolicyMatcher(_message.Message):
    __slots__ = ("kind", "pattern")
    KIND_FIELD_NUMBER: _ClassVar[int]
    PATTERN_FIELD_NUMBER: _ClassVar[int]
    kind: str
    pattern: str
    def __init__(self, kind: _Optional[str] = ..., pattern: _Optional[str] = ...) -> None: ...

class PermissionPolicyRule(_message.Message):
    __slots__ = ("id", "action", "matcher", "rationale", "owner", "target_scope", "requires_hard_enforcement")
    ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    MATCHER_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    TARGET_SCOPE_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_HARD_ENFORCEMENT_FIELD_NUMBER: _ClassVar[int]
    id: str
    action: str
    matcher: PermissionPolicyMatcher
    rationale: str
    owner: str
    target_scope: str
    requires_hard_enforcement: bool
    def __init__(self, id: _Optional[str] = ..., action: _Optional[str] = ..., matcher: _Optional[_Union[PermissionPolicyMatcher, _Mapping]] = ..., rationale: _Optional[str] = ..., owner: _Optional[str] = ..., target_scope: _Optional[str] = ..., requires_hard_enforcement: _Optional[bool] = ...) -> None: ...

class PermissionPolicyCatalog(_message.Message):
    __slots__ = ("schema_version", "metadata", "target_scopes", "rules")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    TARGET_SCOPES_FIELD_NUMBER: _ClassVar[int]
    RULES_FIELD_NUMBER: _ClassVar[int]
    schema_version: int
    metadata: PermissionPolicyCatalogMetadata
    target_scopes: _containers.RepeatedScalarFieldContainer[str]
    rules: _containers.RepeatedCompositeFieldContainer[PermissionPolicyRule]
    def __init__(self, schema_version: _Optional[int] = ..., metadata: _Optional[_Union[PermissionPolicyCatalogMetadata, _Mapping]] = ..., target_scopes: _Optional[_Iterable[str]] = ..., rules: _Optional[_Iterable[_Union[PermissionPolicyRule, _Mapping]]] = ...) -> None: ...

class PermissionPolicyEnforcement(_message.Message):
    __slots__ = ("permissions", "caveats")
    PERMISSIONS_FIELD_NUMBER: _ClassVar[int]
    CAVEATS_FIELD_NUMBER: _ClassVar[int]
    permissions: str
    caveats: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, permissions: _Optional[str] = ..., caveats: _Optional[_Iterable[str]] = ...) -> None: ...

class PermissionPolicyResourceResult(_message.Message):
    __slots__ = ("runner_type", "scope", "installed", "status", "error", "desired_digest", "desired_fingerprint", "live_fingerprint", "drift", "changes", "native_paths", "enforcement", "unsupported_matchers")
    RUNNER_TYPE_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    INSTALLED_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    DESIRED_DIGEST_FIELD_NUMBER: _ClassVar[int]
    DESIRED_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    LIVE_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    DRIFT_FIELD_NUMBER: _ClassVar[int]
    CHANGES_FIELD_NUMBER: _ClassVar[int]
    NATIVE_PATHS_FIELD_NUMBER: _ClassVar[int]
    ENFORCEMENT_FIELD_NUMBER: _ClassVar[int]
    UNSUPPORTED_MATCHERS_FIELD_NUMBER: _ClassVar[int]
    runner_type: _types_pb2.RunnerType
    scope: str
    installed: bool
    status: str
    error: str
    desired_digest: str
    desired_fingerprint: str
    live_fingerprint: str
    drift: bool
    changes: _containers.RepeatedScalarFieldContainer[str]
    native_paths: _containers.RepeatedScalarFieldContainer[str]
    enforcement: PermissionPolicyEnforcement
    unsupported_matchers: _containers.RepeatedCompositeFieldContainer[PermissionPolicyMatcher]
    def __init__(self, runner_type: _Optional[_Union[_types_pb2.RunnerType, str]] = ..., scope: _Optional[str] = ..., installed: _Optional[bool] = ..., status: _Optional[str] = ..., error: _Optional[str] = ..., desired_digest: _Optional[str] = ..., desired_fingerprint: _Optional[str] = ..., live_fingerprint: _Optional[str] = ..., drift: _Optional[bool] = ..., changes: _Optional[_Iterable[str]] = ..., native_paths: _Optional[_Iterable[str]] = ..., enforcement: _Optional[_Union[PermissionPolicyEnforcement, _Mapping]] = ..., unsupported_matchers: _Optional[_Iterable[_Union[PermissionPolicyMatcher, _Mapping]]] = ...) -> None: ...

class PermissionPolicyPlan(_message.Message):
    __slots__ = ("catalog_digest", "resources", "hard_enforcement_satisfied", "missing_hard_enforcement_rule_ids")
    CATALOG_DIGEST_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    HARD_ENFORCEMENT_SATISFIED_FIELD_NUMBER: _ClassVar[int]
    MISSING_HARD_ENFORCEMENT_RULE_IDS_FIELD_NUMBER: _ClassVar[int]
    catalog_digest: str
    resources: _containers.RepeatedCompositeFieldContainer[PermissionPolicyResourceResult]
    hard_enforcement_satisfied: bool
    missing_hard_enforcement_rule_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, catalog_digest: _Optional[str] = ..., resources: _Optional[_Iterable[_Union[PermissionPolicyResourceResult, _Mapping]]] = ..., hard_enforcement_satisfied: _Optional[bool] = ..., missing_hard_enforcement_rule_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class PermissionPolicyReconcileResult(_message.Message):
    __slots__ = ("catalog_digest", "started_at", "finished_at", "explicitly_authorized", "success", "hard_enforcement_satisfied", "missing_hard_enforcement_rule_ids", "resources")
    CATALOG_DIGEST_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    EXPLICITLY_AUTHORIZED_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    HARD_ENFORCEMENT_SATISFIED_FIELD_NUMBER: _ClassVar[int]
    MISSING_HARD_ENFORCEMENT_RULE_IDS_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    catalog_digest: str
    started_at: _timestamp_pb2.Timestamp
    finished_at: _timestamp_pb2.Timestamp
    explicitly_authorized: bool
    success: bool
    hard_enforcement_satisfied: bool
    missing_hard_enforcement_rule_ids: _containers.RepeatedScalarFieldContainer[str]
    resources: _containers.RepeatedCompositeFieldContainer[PermissionPolicyResourceResult]
    def __init__(self, catalog_digest: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., finished_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., explicitly_authorized: _Optional[bool] = ..., success: _Optional[bool] = ..., hard_enforcement_satisfied: _Optional[bool] = ..., missing_hard_enforcement_rule_ids: _Optional[_Iterable[str]] = ..., resources: _Optional[_Iterable[_Union[PermissionPolicyResourceResult, _Mapping]]] = ...) -> None: ...

class GetPermissionPolicyStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetPermissionPolicyStatusResponse(_message.Message):
    __slots__ = ("status", "last_reconcile")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LAST_RECONCILE_FIELD_NUMBER: _ClassVar[int]
    status: PermissionPolicyStatus
    last_reconcile: PermissionPolicyReconcileResult
    def __init__(self, status: _Optional[_Union[PermissionPolicyStatus, _Mapping]] = ..., last_reconcile: _Optional[_Union[PermissionPolicyReconcileResult, _Mapping]] = ...) -> None: ...

class GetPermissionPolicyCatalogRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetPermissionPolicyCatalogResponse(_message.Message):
    __slots__ = ("status", "catalog")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CATALOG_FIELD_NUMBER: _ClassVar[int]
    status: PermissionPolicyStatus
    catalog: PermissionPolicyCatalog
    def __init__(self, status: _Optional[_Union[PermissionPolicyStatus, _Mapping]] = ..., catalog: _Optional[_Union[PermissionPolicyCatalog, _Mapping]] = ...) -> None: ...

class ValidatePermissionPolicyCatalogRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ValidatePermissionPolicyCatalogResponse(_message.Message):
    __slots__ = ("valid", "candidate_digest", "active_digest", "diagnostic")
    VALID_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTIC_FIELD_NUMBER: _ClassVar[int]
    valid: bool
    candidate_digest: str
    active_digest: str
    diagnostic: PermissionPolicyDiagnostic
    def __init__(self, valid: _Optional[bool] = ..., candidate_digest: _Optional[str] = ..., active_digest: _Optional[str] = ..., diagnostic: _Optional[_Union[PermissionPolicyDiagnostic, _Mapping]] = ...) -> None: ...

class ReloadPermissionPolicyCatalogRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ReloadPermissionPolicyCatalogResponse(_message.Message):
    __slots__ = ("activated", "status", "diagnostic")
    ACTIVATED_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTIC_FIELD_NUMBER: _ClassVar[int]
    activated: bool
    status: PermissionPolicyStatus
    diagnostic: PermissionPolicyDiagnostic
    def __init__(self, activated: _Optional[bool] = ..., status: _Optional[_Union[PermissionPolicyStatus, _Mapping]] = ..., diagnostic: _Optional[_Union[PermissionPolicyDiagnostic, _Mapping]] = ...) -> None: ...

class PlanPermissionPolicyRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class PlanPermissionPolicyResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: PermissionPolicyPlan
    def __init__(self, plan: _Optional[_Union[PermissionPolicyPlan, _Mapping]] = ...) -> None: ...

class ReconcilePermissionPolicyRequest(_message.Message):
    __slots__ = ("explicitly_authorized",)
    EXPLICITLY_AUTHORIZED_FIELD_NUMBER: _ClassVar[int]
    explicitly_authorized: bool
    def __init__(self, explicitly_authorized: _Optional[bool] = ...) -> None: ...

class ReconcilePermissionPolicyResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: PermissionPolicyReconcileResult
    def __init__(self, result: _Optional[_Union[PermissionPolicyReconcileResult, _Mapping]] = ...) -> None: ...

class DoctorPermissionPolicyRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class DoctorPermissionPolicyResponse(_message.Message):
    __slots__ = ("status", "plan", "healthy", "summary")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PLAN_FIELD_NUMBER: _ClassVar[int]
    HEALTHY_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    status: PermissionPolicyStatus
    plan: PermissionPolicyPlan
    healthy: bool
    summary: str
    def __init__(self, status: _Optional[_Union[PermissionPolicyStatus, _Mapping]] = ..., plan: _Optional[_Union[PermissionPolicyPlan, _Mapping]] = ..., healthy: _Optional[bool] = ..., summary: _Optional[str] = ...) -> None: ...

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
