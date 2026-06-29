import datetime

from google.protobuf import duration_pb2 as _duration_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RunnerType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RUNNER_TYPE_UNSPECIFIED: _ClassVar[RunnerType]
    RUNNER_TYPE_CLAUDE_CODE: _ClassVar[RunnerType]
    RUNNER_TYPE_CODEX: _ClassVar[RunnerType]
    RUNNER_TYPE_OPENCODE: _ClassVar[RunnerType]
    RUNNER_TYPE_GROK: _ClassVar[RunnerType]

class ModelPreset(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    MODEL_PRESET_UNSPECIFIED: _ClassVar[ModelPreset]
    MODEL_PRESET_FAST: _ClassVar[ModelPreset]
    MODEL_PRESET_CHEAP: _ClassVar[ModelPreset]
    MODEL_PRESET_SMART: _ClassVar[ModelPreset]

class NetworkAccess(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    NETWORK_ACCESS_UNSPECIFIED: _ClassVar[NetworkAccess]
    NETWORK_ACCESS_NONE: _ClassVar[NetworkAccess]
    NETWORK_ACCESS_LOCALHOST: _ClassVar[NetworkAccess]
    NETWORK_ACCESS_FULL: _ClassVar[NetworkAccess]

class SandboxLifecycleEvent(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SANDBOX_LIFECYCLE_EVENT_UNSPECIFIED: _ClassVar[SandboxLifecycleEvent]
    SANDBOX_LIFECYCLE_EVENT_RUN_COMPLETED: _ClassVar[SandboxLifecycleEvent]
    SANDBOX_LIFECYCLE_EVENT_RUN_FAILED: _ClassVar[SandboxLifecycleEvent]
    SANDBOX_LIFECYCLE_EVENT_RUN_CANCELLED: _ClassVar[SandboxLifecycleEvent]
    SANDBOX_LIFECYCLE_EVENT_APPROVED: _ClassVar[SandboxLifecycleEvent]
    SANDBOX_LIFECYCLE_EVENT_REJECTED: _ClassVar[SandboxLifecycleEvent]
    SANDBOX_LIFECYCLE_EVENT_TERMINAL: _ClassVar[SandboxLifecycleEvent]
    SANDBOX_LIFECYCLE_EVENT_TURN_COMPLETED: _ClassVar[SandboxLifecycleEvent]
    SANDBOX_LIFECYCLE_EVENT_TURN_FAILED: _ClassVar[SandboxLifecycleEvent]
    SANDBOX_LIFECYCLE_EVENT_TURN_CANCELLED: _ClassVar[SandboxLifecycleEvent]
    SANDBOX_LIFECYCLE_EVENT_RUN_FINALIZED: _ClassVar[SandboxLifecycleEvent]

class SandboxAcceptanceMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SANDBOX_ACCEPTANCE_MODE_UNSPECIFIED: _ClassVar[SandboxAcceptanceMode]
    SANDBOX_ACCEPTANCE_MODE_ALLOWLIST: _ClassVar[SandboxAcceptanceMode]

class SandboxMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SANDBOX_MODE_UNSPECIFIED: _ClassVar[SandboxMode]
    SANDBOX_MODE_TRACKING: _ClassVar[SandboxMode]
    SANDBOX_MODE_PROTECTED: _ClassVar[SandboxMode]
    SANDBOX_MODE_OFF: _ClassVar[SandboxMode]

class TaskStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TASK_STATUS_UNSPECIFIED: _ClassVar[TaskStatus]
    TASK_STATUS_QUEUED: _ClassVar[TaskStatus]
    TASK_STATUS_RUNNING: _ClassVar[TaskStatus]
    TASK_STATUS_NEEDS_REVIEW: _ClassVar[TaskStatus]
    TASK_STATUS_APPROVED: _ClassVar[TaskStatus]
    TASK_STATUS_REJECTED: _ClassVar[TaskStatus]
    TASK_STATUS_FAILED: _ClassVar[TaskStatus]
    TASK_STATUS_CANCELLED: _ClassVar[TaskStatus]

class RunStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RUN_STATUS_UNSPECIFIED: _ClassVar[RunStatus]
    RUN_STATUS_PENDING: _ClassVar[RunStatus]
    RUN_STATUS_STARTING: _ClassVar[RunStatus]
    RUN_STATUS_RUNNING: _ClassVar[RunStatus]
    RUN_STATUS_NEEDS_REVIEW: _ClassVar[RunStatus]
    RUN_STATUS_COMPLETE: _ClassVar[RunStatus]
    RUN_STATUS_FAILED: _ClassVar[RunStatus]
    RUN_STATUS_CANCELLED: _ClassVar[RunStatus]
    RUN_STATUS_PARKED: _ClassVar[RunStatus]

class RunFinalizationStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RUN_FINALIZATION_STATUS_UNSPECIFIED: _ClassVar[RunFinalizationStatus]
    RUN_FINALIZATION_STATUS_NONE: _ClassVar[RunFinalizationStatus]
    RUN_FINALIZATION_STATUS_PENDING: _ClassVar[RunFinalizationStatus]
    RUN_FINALIZATION_STATUS_RUNNING: _ClassVar[RunFinalizationStatus]
    RUN_FINALIZATION_STATUS_SUCCEEDED: _ClassVar[RunFinalizationStatus]
    RUN_FINALIZATION_STATUS_FAILED: _ClassVar[RunFinalizationStatus]
    RUN_FINALIZATION_STATUS_SKIPPED: _ClassVar[RunFinalizationStatus]

class RunPhase(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RUN_PHASE_UNSPECIFIED: _ClassVar[RunPhase]
    RUN_PHASE_QUEUED: _ClassVar[RunPhase]
    RUN_PHASE_INITIALIZING: _ClassVar[RunPhase]
    RUN_PHASE_SANDBOX_CREATING: _ClassVar[RunPhase]
    RUN_PHASE_RUNNER_ACQUIRING: _ClassVar[RunPhase]
    RUN_PHASE_EXECUTING: _ClassVar[RunPhase]
    RUN_PHASE_COLLECTING_RESULTS: _ClassVar[RunPhase]
    RUN_PHASE_AWAITING_REVIEW: _ClassVar[RunPhase]
    RUN_PHASE_APPLYING: _ClassVar[RunPhase]
    RUN_PHASE_CLEANING_UP: _ClassVar[RunPhase]
    RUN_PHASE_COMPLETED: _ClassVar[RunPhase]

class RunMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RUN_MODE_UNSPECIFIED: _ClassVar[RunMode]
    RUN_MODE_SANDBOXED: _ClassVar[RunMode]
    RUN_MODE_IN_PLACE: _ClassVar[RunMode]

class ApprovalState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    APPROVAL_STATE_UNSPECIFIED: _ClassVar[ApprovalState]
    APPROVAL_STATE_NONE: _ClassVar[ApprovalState]
    APPROVAL_STATE_PENDING: _ClassVar[ApprovalState]
    APPROVAL_STATE_PARTIALLY_APPROVED: _ClassVar[ApprovalState]
    APPROVAL_STATE_APPROVED: _ClassVar[ApprovalState]
    APPROVAL_STATE_REJECTED: _ClassVar[ApprovalState]

class RunEventType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RUN_EVENT_TYPE_UNSPECIFIED: _ClassVar[RunEventType]
    RUN_EVENT_TYPE_LOG: _ClassVar[RunEventType]
    RUN_EVENT_TYPE_MESSAGE: _ClassVar[RunEventType]
    RUN_EVENT_TYPE_TOOL_CALL: _ClassVar[RunEventType]
    RUN_EVENT_TYPE_TOOL_RESULT: _ClassVar[RunEventType]
    RUN_EVENT_TYPE_STATUS: _ClassVar[RunEventType]
    RUN_EVENT_TYPE_METRIC: _ClassVar[RunEventType]
    RUN_EVENT_TYPE_ARTIFACT: _ClassVar[RunEventType]
    RUN_EVENT_TYPE_ERROR: _ClassVar[RunEventType]
    RUN_EVENT_TYPE_MESSAGE_DELETED: _ClassVar[RunEventType]
    RUN_EVENT_TYPE_COMPACTION: _ClassVar[RunEventType]
    RUN_EVENT_TYPE_LIFECYCLE: _ClassVar[RunEventType]

class RecoveryAction(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RECOVERY_ACTION_UNSPECIFIED: _ClassVar[RecoveryAction]
    RECOVERY_ACTION_NONE: _ClassVar[RecoveryAction]
    RECOVERY_ACTION_RETRY: _ClassVar[RecoveryAction]
    RECOVERY_ACTION_RETRY_BACKOFF: _ClassVar[RecoveryAction]
    RECOVERY_ACTION_FIX_INPUT: _ClassVar[RecoveryAction]
    RECOVERY_ACTION_WAIT: _ClassVar[RecoveryAction]
    RECOVERY_ACTION_ESCALATE: _ClassVar[RecoveryAction]

class IdempotencyStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    IDEMPOTENCY_STATUS_UNSPECIFIED: _ClassVar[IdempotencyStatus]
    IDEMPOTENCY_STATUS_PENDING: _ClassVar[IdempotencyStatus]
    IDEMPOTENCY_STATUS_COMPLETE: _ClassVar[IdempotencyStatus]
    IDEMPOTENCY_STATUS_FAILED: _ClassVar[IdempotencyStatus]

class RunOutcome(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RUN_OUTCOME_UNSPECIFIED: _ClassVar[RunOutcome]
    RUN_OUTCOME_SUCCESS: _ClassVar[RunOutcome]
    RUN_OUTCOME_EXIT_ERROR: _ClassVar[RunOutcome]
    RUN_OUTCOME_EXCEPTION: _ClassVar[RunOutcome]
    RUN_OUTCOME_CANCELLED: _ClassVar[RunOutcome]
    RUN_OUTCOME_TIMEOUT: _ClassVar[RunOutcome]
    RUN_OUTCOME_SANDBOX_FAIL: _ClassVar[RunOutcome]
    RUN_OUTCOME_RUNNER_FAIL: _ClassVar[RunOutcome]

class StaleRunAction(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    STALE_RUN_ACTION_UNSPECIFIED: _ClassVar[StaleRunAction]
    STALE_RUN_ACTION_NONE: _ClassVar[StaleRunAction]
    STALE_RUN_ACTION_RESUME: _ClassVar[StaleRunAction]
    STALE_RUN_ACTION_FAIL: _ClassVar[StaleRunAction]
    STALE_RUN_ACTION_ALERT: _ClassVar[StaleRunAction]
RUNNER_TYPE_UNSPECIFIED: RunnerType
RUNNER_TYPE_CLAUDE_CODE: RunnerType
RUNNER_TYPE_CODEX: RunnerType
RUNNER_TYPE_OPENCODE: RunnerType
RUNNER_TYPE_GROK: RunnerType
MODEL_PRESET_UNSPECIFIED: ModelPreset
MODEL_PRESET_FAST: ModelPreset
MODEL_PRESET_CHEAP: ModelPreset
MODEL_PRESET_SMART: ModelPreset
NETWORK_ACCESS_UNSPECIFIED: NetworkAccess
NETWORK_ACCESS_NONE: NetworkAccess
NETWORK_ACCESS_LOCALHOST: NetworkAccess
NETWORK_ACCESS_FULL: NetworkAccess
SANDBOX_LIFECYCLE_EVENT_UNSPECIFIED: SandboxLifecycleEvent
SANDBOX_LIFECYCLE_EVENT_RUN_COMPLETED: SandboxLifecycleEvent
SANDBOX_LIFECYCLE_EVENT_RUN_FAILED: SandboxLifecycleEvent
SANDBOX_LIFECYCLE_EVENT_RUN_CANCELLED: SandboxLifecycleEvent
SANDBOX_LIFECYCLE_EVENT_APPROVED: SandboxLifecycleEvent
SANDBOX_LIFECYCLE_EVENT_REJECTED: SandboxLifecycleEvent
SANDBOX_LIFECYCLE_EVENT_TERMINAL: SandboxLifecycleEvent
SANDBOX_LIFECYCLE_EVENT_TURN_COMPLETED: SandboxLifecycleEvent
SANDBOX_LIFECYCLE_EVENT_TURN_FAILED: SandboxLifecycleEvent
SANDBOX_LIFECYCLE_EVENT_TURN_CANCELLED: SandboxLifecycleEvent
SANDBOX_LIFECYCLE_EVENT_RUN_FINALIZED: SandboxLifecycleEvent
SANDBOX_ACCEPTANCE_MODE_UNSPECIFIED: SandboxAcceptanceMode
SANDBOX_ACCEPTANCE_MODE_ALLOWLIST: SandboxAcceptanceMode
SANDBOX_MODE_UNSPECIFIED: SandboxMode
SANDBOX_MODE_TRACKING: SandboxMode
SANDBOX_MODE_PROTECTED: SandboxMode
SANDBOX_MODE_OFF: SandboxMode
TASK_STATUS_UNSPECIFIED: TaskStatus
TASK_STATUS_QUEUED: TaskStatus
TASK_STATUS_RUNNING: TaskStatus
TASK_STATUS_NEEDS_REVIEW: TaskStatus
TASK_STATUS_APPROVED: TaskStatus
TASK_STATUS_REJECTED: TaskStatus
TASK_STATUS_FAILED: TaskStatus
TASK_STATUS_CANCELLED: TaskStatus
RUN_STATUS_UNSPECIFIED: RunStatus
RUN_STATUS_PENDING: RunStatus
RUN_STATUS_STARTING: RunStatus
RUN_STATUS_RUNNING: RunStatus
RUN_STATUS_NEEDS_REVIEW: RunStatus
RUN_STATUS_COMPLETE: RunStatus
RUN_STATUS_FAILED: RunStatus
RUN_STATUS_CANCELLED: RunStatus
RUN_STATUS_PARKED: RunStatus
RUN_FINALIZATION_STATUS_UNSPECIFIED: RunFinalizationStatus
RUN_FINALIZATION_STATUS_NONE: RunFinalizationStatus
RUN_FINALIZATION_STATUS_PENDING: RunFinalizationStatus
RUN_FINALIZATION_STATUS_RUNNING: RunFinalizationStatus
RUN_FINALIZATION_STATUS_SUCCEEDED: RunFinalizationStatus
RUN_FINALIZATION_STATUS_FAILED: RunFinalizationStatus
RUN_FINALIZATION_STATUS_SKIPPED: RunFinalizationStatus
RUN_PHASE_UNSPECIFIED: RunPhase
RUN_PHASE_QUEUED: RunPhase
RUN_PHASE_INITIALIZING: RunPhase
RUN_PHASE_SANDBOX_CREATING: RunPhase
RUN_PHASE_RUNNER_ACQUIRING: RunPhase
RUN_PHASE_EXECUTING: RunPhase
RUN_PHASE_COLLECTING_RESULTS: RunPhase
RUN_PHASE_AWAITING_REVIEW: RunPhase
RUN_PHASE_APPLYING: RunPhase
RUN_PHASE_CLEANING_UP: RunPhase
RUN_PHASE_COMPLETED: RunPhase
RUN_MODE_UNSPECIFIED: RunMode
RUN_MODE_SANDBOXED: RunMode
RUN_MODE_IN_PLACE: RunMode
APPROVAL_STATE_UNSPECIFIED: ApprovalState
APPROVAL_STATE_NONE: ApprovalState
APPROVAL_STATE_PENDING: ApprovalState
APPROVAL_STATE_PARTIALLY_APPROVED: ApprovalState
APPROVAL_STATE_APPROVED: ApprovalState
APPROVAL_STATE_REJECTED: ApprovalState
RUN_EVENT_TYPE_UNSPECIFIED: RunEventType
RUN_EVENT_TYPE_LOG: RunEventType
RUN_EVENT_TYPE_MESSAGE: RunEventType
RUN_EVENT_TYPE_TOOL_CALL: RunEventType
RUN_EVENT_TYPE_TOOL_RESULT: RunEventType
RUN_EVENT_TYPE_STATUS: RunEventType
RUN_EVENT_TYPE_METRIC: RunEventType
RUN_EVENT_TYPE_ARTIFACT: RunEventType
RUN_EVENT_TYPE_ERROR: RunEventType
RUN_EVENT_TYPE_MESSAGE_DELETED: RunEventType
RUN_EVENT_TYPE_COMPACTION: RunEventType
RUN_EVENT_TYPE_LIFECYCLE: RunEventType
RECOVERY_ACTION_UNSPECIFIED: RecoveryAction
RECOVERY_ACTION_NONE: RecoveryAction
RECOVERY_ACTION_RETRY: RecoveryAction
RECOVERY_ACTION_RETRY_BACKOFF: RecoveryAction
RECOVERY_ACTION_FIX_INPUT: RecoveryAction
RECOVERY_ACTION_WAIT: RecoveryAction
RECOVERY_ACTION_ESCALATE: RecoveryAction
IDEMPOTENCY_STATUS_UNSPECIFIED: IdempotencyStatus
IDEMPOTENCY_STATUS_PENDING: IdempotencyStatus
IDEMPOTENCY_STATUS_COMPLETE: IdempotencyStatus
IDEMPOTENCY_STATUS_FAILED: IdempotencyStatus
RUN_OUTCOME_UNSPECIFIED: RunOutcome
RUN_OUTCOME_SUCCESS: RunOutcome
RUN_OUTCOME_EXIT_ERROR: RunOutcome
RUN_OUTCOME_EXCEPTION: RunOutcome
RUN_OUTCOME_CANCELLED: RunOutcome
RUN_OUTCOME_TIMEOUT: RunOutcome
RUN_OUTCOME_SANDBOX_FAIL: RunOutcome
RUN_OUTCOME_RUNNER_FAIL: RunOutcome
STALE_RUN_ACTION_UNSPECIFIED: StaleRunAction
STALE_RUN_ACTION_NONE: StaleRunAction
STALE_RUN_ACTION_RESUME: StaleRunAction
STALE_RUN_ACTION_FAIL: StaleRunAction
STALE_RUN_ACTION_ALERT: StaleRunAction

class SandboxFileCriteria(_message.Message):
    __slots__ = ("path_globs", "extensions")
    PATH_GLOBS_FIELD_NUMBER: _ClassVar[int]
    EXTENSIONS_FIELD_NUMBER: _ClassVar[int]
    path_globs: _containers.RepeatedScalarFieldContainer[str]
    extensions: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, path_globs: _Optional[_Iterable[str]] = ..., extensions: _Optional[_Iterable[str]] = ...) -> None: ...

class SandboxAcceptanceConfig(_message.Message):
    __slots__ = ("mode", "allow", "deny", "ignore_binary")
    MODE_FIELD_NUMBER: _ClassVar[int]
    ALLOW_FIELD_NUMBER: _ClassVar[int]
    DENY_FIELD_NUMBER: _ClassVar[int]
    IGNORE_BINARY_FIELD_NUMBER: _ClassVar[int]
    mode: SandboxAcceptanceMode
    allow: SandboxFileCriteria
    deny: SandboxFileCriteria
    ignore_binary: bool
    def __init__(self, mode: _Optional[_Union[SandboxAcceptanceMode, str]] = ..., allow: _Optional[_Union[SandboxFileCriteria, _Mapping]] = ..., deny: _Optional[_Union[SandboxFileCriteria, _Mapping]] = ..., ignore_binary: _Optional[bool] = ...) -> None: ...

class SandboxLifecycleConfig(_message.Message):
    __slots__ = ("stop_on", "delete_on", "ttl", "idle_timeout", "checkpoint_on")
    STOP_ON_FIELD_NUMBER: _ClassVar[int]
    DELETE_ON_FIELD_NUMBER: _ClassVar[int]
    TTL_FIELD_NUMBER: _ClassVar[int]
    IDLE_TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    CHECKPOINT_ON_FIELD_NUMBER: _ClassVar[int]
    stop_on: _containers.RepeatedScalarFieldContainer[SandboxLifecycleEvent]
    delete_on: _containers.RepeatedScalarFieldContainer[SandboxLifecycleEvent]
    ttl: _duration_pb2.Duration
    idle_timeout: _duration_pb2.Duration
    checkpoint_on: _containers.RepeatedScalarFieldContainer[SandboxLifecycleEvent]
    def __init__(self, stop_on: _Optional[_Iterable[_Union[SandboxLifecycleEvent, str]]] = ..., delete_on: _Optional[_Iterable[_Union[SandboxLifecycleEvent, str]]] = ..., ttl: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., idle_timeout: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., checkpoint_on: _Optional[_Iterable[_Union[SandboxLifecycleEvent, str]]] = ...) -> None: ...

class SandboxConfig(_message.Message):
    __slots__ = ("lifecycle", "acceptance", "mode", "manual_review", "auto_apply", "apply_on_failure", "network_mode", "no_lock")
    LIFECYCLE_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    MANUAL_REVIEW_FIELD_NUMBER: _ClassVar[int]
    AUTO_APPLY_FIELD_NUMBER: _ClassVar[int]
    APPLY_ON_FAILURE_FIELD_NUMBER: _ClassVar[int]
    NETWORK_MODE_FIELD_NUMBER: _ClassVar[int]
    NO_LOCK_FIELD_NUMBER: _ClassVar[int]
    lifecycle: SandboxLifecycleConfig
    acceptance: SandboxAcceptanceConfig
    mode: SandboxMode
    manual_review: bool
    auto_apply: bool
    apply_on_failure: bool
    network_mode: NetworkAccess
    no_lock: bool
    def __init__(self, lifecycle: _Optional[_Union[SandboxLifecycleConfig, _Mapping]] = ..., acceptance: _Optional[_Union[SandboxAcceptanceConfig, _Mapping]] = ..., mode: _Optional[_Union[SandboxMode, str]] = ..., manual_review: _Optional[bool] = ..., auto_apply: _Optional[bool] = ..., apply_on_failure: _Optional[bool] = ..., network_mode: _Optional[_Union[NetworkAccess, str]] = ..., no_lock: _Optional[bool] = ...) -> None: ...

class FeatureFlags(_message.Message):
    __slots__ = ("enable_browser",)
    ENABLE_BROWSER_FIELD_NUMBER: _ClassVar[int]
    enable_browser: bool
    def __init__(self, enable_browser: _Optional[bool] = ...) -> None: ...

class ExtraFlagList(_message.Message):
    __slots__ = ("flags",)
    FLAGS_FIELD_NUMBER: _ClassVar[int]
    flags: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, flags: _Optional[_Iterable[str]] = ...) -> None: ...
