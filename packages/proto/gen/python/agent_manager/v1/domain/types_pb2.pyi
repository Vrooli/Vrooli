from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from typing import ClassVar as _ClassVar

DESCRIPTOR: _descriptor.FileDescriptor

class RunnerType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RUNNER_TYPE_UNSPECIFIED: _ClassVar[RunnerType]
    RUNNER_TYPE_CLAUDE_CODE: _ClassVar[RunnerType]
    RUNNER_TYPE_CODEX: _ClassVar[RunnerType]
    RUNNER_TYPE_OPENCODE: _ClassVar[RunnerType]

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
