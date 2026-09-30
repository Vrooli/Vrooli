import datetime

from google.protobuf import duration_pb2 as _duration_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class WatchStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WATCH_STATUS_UNSPECIFIED: _ClassVar[WatchStatus]
    WATCH_STATUS_ACTIVE: _ClassVar[WatchStatus]
    WATCH_STATUS_TERMINAL: _ClassVar[WatchStatus]
    WATCH_STATUS_CANCELED: _ClassVar[WatchStatus]
    WATCH_STATUS_FAILED: _ClassVar[WatchStatus]

class WatchDisposition(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WATCH_DISPOSITION_UNSPECIFIED: _ClassVar[WatchDisposition]
    WATCH_DISPOSITION_QUIET: _ClassVar[WatchDisposition]
    WATCH_DISPOSITION_SIGNAL: _ClassVar[WatchDisposition]
    WATCH_DISPOSITION_TERMINAL: _ClassVar[WatchDisposition]
    WATCH_DISPOSITION_CURSOR_RESET: _ClassVar[WatchDisposition]
    WATCH_DISPOSITION_UNAVAILABLE: _ClassVar[WatchDisposition]

class WatchActionKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WATCH_ACTION_KIND_UNSPECIFIED: _ClassVar[WatchActionKind]
    WATCH_ACTION_KIND_OBSERVE: _ClassVar[WatchActionKind]
    WATCH_ACTION_KIND_NUDGE: _ClassVar[WatchActionKind]
    WATCH_ACTION_KIND_PARK: _ClassVar[WatchActionKind]
    WATCH_ACTION_KIND_CONTINUE: _ClassVar[WatchActionKind]
    WATCH_ACTION_KIND_STOP: _ClassVar[WatchActionKind]
    WATCH_ACTION_KIND_ESCALATE: _ClassVar[WatchActionKind]
    WATCH_ACTION_KIND_WAKE_PARENT: _ClassVar[WatchActionKind]

class WatchActionState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WATCH_ACTION_STATE_UNSPECIFIED: _ClassVar[WatchActionState]
    WATCH_ACTION_STATE_REQUESTED: _ClassVar[WatchActionState]
    WATCH_ACTION_STATE_QUEUED: _ClassVar[WatchActionState]
    WATCH_ACTION_STATE_DELIVERED: _ClassVar[WatchActionState]
    WATCH_ACTION_STATE_ACCEPTED: _ClassVar[WatchActionState]
    WATCH_ACTION_STATE_APPLIED: _ClassVar[WatchActionState]
    WATCH_ACTION_STATE_REJECTED: _ClassVar[WatchActionState]
    WATCH_ACTION_STATE_SUPERSEDED: _ClassVar[WatchActionState]
    WATCH_ACTION_STATE_EXPIRED: _ClassVar[WatchActionState]

class WatchAuthority(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WATCH_AUTHORITY_UNSPECIFIED: _ClassVar[WatchAuthority]
    WATCH_AUTHORITY_SYSTEM: _ClassVar[WatchAuthority]
    WATCH_AUTHORITY_FAMILY_PARENT: _ClassVar[WatchAuthority]
    WATCH_AUTHORITY_OPERATOR: _ClassVar[WatchAuthority]

class SupervisionPolicyState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SUPERVISION_POLICY_STATE_UNSPECIFIED: _ClassVar[SupervisionPolicyState]
    SUPERVISION_POLICY_STATE_CANDIDATE: _ClassVar[SupervisionPolicyState]
    SUPERVISION_POLICY_STATE_ACTIVE: _ClassVar[SupervisionPolicyState]
    SUPERVISION_POLICY_STATE_RETIRED: _ClassVar[SupervisionPolicyState]
    SUPERVISION_POLICY_STATE_REJECTED: _ClassVar[SupervisionPolicyState]
    SUPERVISION_POLICY_STATE_ROLLED_BACK: _ClassVar[SupervisionPolicyState]
WATCH_STATUS_UNSPECIFIED: WatchStatus
WATCH_STATUS_ACTIVE: WatchStatus
WATCH_STATUS_TERMINAL: WatchStatus
WATCH_STATUS_CANCELED: WatchStatus
WATCH_STATUS_FAILED: WatchStatus
WATCH_DISPOSITION_UNSPECIFIED: WatchDisposition
WATCH_DISPOSITION_QUIET: WatchDisposition
WATCH_DISPOSITION_SIGNAL: WatchDisposition
WATCH_DISPOSITION_TERMINAL: WatchDisposition
WATCH_DISPOSITION_CURSOR_RESET: WatchDisposition
WATCH_DISPOSITION_UNAVAILABLE: WatchDisposition
WATCH_ACTION_KIND_UNSPECIFIED: WatchActionKind
WATCH_ACTION_KIND_OBSERVE: WatchActionKind
WATCH_ACTION_KIND_NUDGE: WatchActionKind
WATCH_ACTION_KIND_PARK: WatchActionKind
WATCH_ACTION_KIND_CONTINUE: WatchActionKind
WATCH_ACTION_KIND_STOP: WatchActionKind
WATCH_ACTION_KIND_ESCALATE: WatchActionKind
WATCH_ACTION_KIND_WAKE_PARENT: WatchActionKind
WATCH_ACTION_STATE_UNSPECIFIED: WatchActionState
WATCH_ACTION_STATE_REQUESTED: WatchActionState
WATCH_ACTION_STATE_QUEUED: WatchActionState
WATCH_ACTION_STATE_DELIVERED: WatchActionState
WATCH_ACTION_STATE_ACCEPTED: WatchActionState
WATCH_ACTION_STATE_APPLIED: WatchActionState
WATCH_ACTION_STATE_REJECTED: WatchActionState
WATCH_ACTION_STATE_SUPERSEDED: WatchActionState
WATCH_ACTION_STATE_EXPIRED: WatchActionState
WATCH_AUTHORITY_UNSPECIFIED: WatchAuthority
WATCH_AUTHORITY_SYSTEM: WatchAuthority
WATCH_AUTHORITY_FAMILY_PARENT: WatchAuthority
WATCH_AUTHORITY_OPERATOR: WatchAuthority
SUPERVISION_POLICY_STATE_UNSPECIFIED: SupervisionPolicyState
SUPERVISION_POLICY_STATE_CANDIDATE: SupervisionPolicyState
SUPERVISION_POLICY_STATE_ACTIVE: SupervisionPolicyState
SUPERVISION_POLICY_STATE_RETIRED: SupervisionPolicyState
SUPERVISION_POLICY_STATE_REJECTED: SupervisionPolicyState
SUPERVISION_POLICY_STATE_ROLLED_BACK: SupervisionPolicyState

class WatchSubject(_message.Message):
    __slots__ = ("family_execution_id", "plan_id", "run_id")
    FAMILY_EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    family_execution_id: str
    plan_id: str
    run_id: str
    def __init__(self, family_execution_id: _Optional[str] = ..., plan_id: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class WatchTriggers(_message.Message):
    __slots__ = ("event_count", "quiet_time", "deadline", "friction_score", "terminal")
    EVENT_COUNT_FIELD_NUMBER: _ClassVar[int]
    QUIET_TIME_FIELD_NUMBER: _ClassVar[int]
    DEADLINE_FIELD_NUMBER: _ClassVar[int]
    FRICTION_SCORE_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_FIELD_NUMBER: _ClassVar[int]
    event_count: int
    quiet_time: _duration_pb2.Duration
    deadline: _timestamp_pb2.Timestamp
    friction_score: float
    terminal: bool
    def __init__(self, event_count: _Optional[int] = ..., quiet_time: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., deadline: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., friction_score: _Optional[float] = ..., terminal: _Optional[bool] = ...) -> None: ...

class WatchSpec(_message.Message):
    __slots__ = ("family_execution_id", "parent_run_id", "subjects", "triggers", "policy_version", "created_by")
    FAMILY_EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PARENT_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SUBJECTS_FIELD_NUMBER: _ClassVar[int]
    TRIGGERS_FIELD_NUMBER: _ClassVar[int]
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    family_execution_id: str
    parent_run_id: str
    subjects: _containers.RepeatedCompositeFieldContainer[WatchSubject]
    triggers: WatchTriggers
    policy_version: str
    created_by: str
    def __init__(self, family_execution_id: _Optional[str] = ..., parent_run_id: _Optional[str] = ..., subjects: _Optional[_Iterable[_Union[WatchSubject, _Mapping]]] = ..., triggers: _Optional[_Union[WatchTriggers, _Mapping]] = ..., policy_version: _Optional[str] = ..., created_by: _Optional[str] = ...) -> None: ...

class WatchCursor(_message.Message):
    __slots__ = ("token",)
    TOKEN_FIELD_NUMBER: _ClassVar[int]
    token: str
    def __init__(self, token: _Optional[str] = ...) -> None: ...

class WatchDecision(_message.Message):
    __slots__ = ("decision_id", "idempotency_key", "disposition", "evidence_ids", "classification", "confidence", "recommended_action", "next_cursor", "next_wake_at", "created_at")
    DECISION_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_IDS_FIELD_NUMBER: _ClassVar[int]
    CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_ACTION_FIELD_NUMBER: _ClassVar[int]
    NEXT_CURSOR_FIELD_NUMBER: _ClassVar[int]
    NEXT_WAKE_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    decision_id: str
    idempotency_key: str
    disposition: WatchDisposition
    evidence_ids: _containers.RepeatedScalarFieldContainer[str]
    classification: str
    confidence: float
    recommended_action: WatchActionKind
    next_cursor: WatchCursor
    next_wake_at: _timestamp_pb2.Timestamp
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, decision_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., disposition: _Optional[_Union[WatchDisposition, str]] = ..., evidence_ids: _Optional[_Iterable[str]] = ..., classification: _Optional[str] = ..., confidence: _Optional[float] = ..., recommended_action: _Optional[_Union[WatchActionKind, str]] = ..., next_cursor: _Optional[_Union[WatchCursor, _Mapping]] = ..., next_wake_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class WatchAction(_message.Message):
    __slots__ = ("action_id", "idempotency_key", "kind", "target_run_id", "rationale", "state", "created_at", "acknowledged_at", "watch_id", "decision_id", "requested_by", "authority", "message", "cooldown", "maximum_count", "rejection_reason")
    ACTION_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    TARGET_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    ACKNOWLEDGED_AT_FIELD_NUMBER: _ClassVar[int]
    WATCH_ID_FIELD_NUMBER: _ClassVar[int]
    DECISION_ID_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_BY_FIELD_NUMBER: _ClassVar[int]
    AUTHORITY_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    COOLDOWN_FIELD_NUMBER: _ClassVar[int]
    MAXIMUM_COUNT_FIELD_NUMBER: _ClassVar[int]
    REJECTION_REASON_FIELD_NUMBER: _ClassVar[int]
    action_id: str
    idempotency_key: str
    kind: WatchActionKind
    target_run_id: str
    rationale: str
    state: WatchActionState
    created_at: _timestamp_pb2.Timestamp
    acknowledged_at: _timestamp_pb2.Timestamp
    watch_id: str
    decision_id: str
    requested_by: str
    authority: WatchAuthority
    message: str
    cooldown: _duration_pb2.Duration
    maximum_count: int
    rejection_reason: str
    def __init__(self, action_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., kind: _Optional[_Union[WatchActionKind, str]] = ..., target_run_id: _Optional[str] = ..., rationale: _Optional[str] = ..., state: _Optional[_Union[WatchActionState, str]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., acknowledged_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., watch_id: _Optional[str] = ..., decision_id: _Optional[str] = ..., requested_by: _Optional[str] = ..., authority: _Optional[_Union[WatchAuthority, str]] = ..., message: _Optional[str] = ..., cooldown: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., maximum_count: _Optional[int] = ..., rejection_reason: _Optional[str] = ...) -> None: ...

class CohortWatch(_message.Message):
    __slots__ = ("watch_id", "revision", "status", "spec", "cursor", "last_decision", "next_wake_at", "created_at", "updated_at", "terminal_at")
    WATCH_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SPEC_FIELD_NUMBER: _ClassVar[int]
    CURSOR_FIELD_NUMBER: _ClassVar[int]
    LAST_DECISION_FIELD_NUMBER: _ClassVar[int]
    NEXT_WAKE_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_AT_FIELD_NUMBER: _ClassVar[int]
    watch_id: str
    revision: int
    status: WatchStatus
    spec: WatchSpec
    cursor: WatchCursor
    last_decision: WatchDecision
    next_wake_at: _timestamp_pb2.Timestamp
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    terminal_at: _timestamp_pb2.Timestamp
    def __init__(self, watch_id: _Optional[str] = ..., revision: _Optional[int] = ..., status: _Optional[_Union[WatchStatus, str]] = ..., spec: _Optional[_Union[WatchSpec, _Mapping]] = ..., cursor: _Optional[_Union[WatchCursor, _Mapping]] = ..., last_decision: _Optional[_Union[WatchDecision, _Mapping]] = ..., next_wake_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., terminal_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CreateCohortWatchRequest(_message.Message):
    __slots__ = ("spec", "idempotency_key")
    SPEC_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    spec: WatchSpec
    idempotency_key: str
    def __init__(self, spec: _Optional[_Union[WatchSpec, _Mapping]] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class GetCohortWatchRequest(_message.Message):
    __slots__ = ("watch_id",)
    WATCH_ID_FIELD_NUMBER: _ClassVar[int]
    watch_id: str
    def __init__(self, watch_id: _Optional[str] = ...) -> None: ...

class ListCohortWatchesRequest(_message.Message):
    __slots__ = ("family_execution_id", "status", "page_size", "page_token")
    FAMILY_EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    family_execution_id: str
    status: WatchStatus
    page_size: int
    page_token: str
    def __init__(self, family_execution_id: _Optional[str] = ..., status: _Optional[_Union[WatchStatus, str]] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListCohortWatchesResponse(_message.Message):
    __slots__ = ("watches", "next_page_token")
    WATCHES_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    watches: _containers.RepeatedCompositeFieldContainer[CohortWatch]
    next_page_token: str
    def __init__(self, watches: _Optional[_Iterable[_Union[CohortWatch, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class WaitCohortWatchRequest(_message.Message):
    __slots__ = ("watch_id", "timeout", "after_revision")
    WATCH_ID_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    AFTER_REVISION_FIELD_NUMBER: _ClassVar[int]
    watch_id: str
    timeout: _duration_pb2.Duration
    after_revision: int
    def __init__(self, watch_id: _Optional[str] = ..., timeout: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., after_revision: _Optional[int] = ...) -> None: ...

class WaitCohortWatchResponse(_message.Message):
    __slots__ = ("watch", "timed_out")
    WATCH_FIELD_NUMBER: _ClassVar[int]
    TIMED_OUT_FIELD_NUMBER: _ClassVar[int]
    watch: CohortWatch
    timed_out: bool
    def __init__(self, watch: _Optional[_Union[CohortWatch, _Mapping]] = ..., timed_out: _Optional[bool] = ...) -> None: ...

class CancelCohortWatchRequest(_message.Message):
    __slots__ = ("watch_id", "expected_revision", "reason")
    WATCH_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    watch_id: str
    expected_revision: int
    reason: str
    def __init__(self, watch_id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., reason: _Optional[str] = ...) -> None: ...

class InspectCohortWatchRequest(_message.Message):
    __slots__ = ("watch_id", "event_limit")
    WATCH_ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_LIMIT_FIELD_NUMBER: _ClassVar[int]
    watch_id: str
    event_limit: int
    def __init__(self, watch_id: _Optional[str] = ..., event_limit: _Optional[int] = ...) -> None: ...

class WatchEventEnvelope(_message.Message):
    __slots__ = ("event_id", "run_id", "sequence", "event_type", "timestamp")
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    EVENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    run_id: str
    sequence: int
    event_type: str
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, event_id: _Optional[str] = ..., run_id: _Optional[str] = ..., sequence: _Optional[int] = ..., event_type: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class InspectCohortWatchResponse(_message.Message):
    __slots__ = ("watch", "events", "cursor_reset_required", "reset_reason", "subject_states", "subject_state_unavailable")
    WATCH_FIELD_NUMBER: _ClassVar[int]
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    CURSOR_RESET_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    RESET_REASON_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_STATES_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_STATE_UNAVAILABLE_FIELD_NUMBER: _ClassVar[int]
    watch: CohortWatch
    events: _containers.RepeatedCompositeFieldContainer[WatchEventEnvelope]
    cursor_reset_required: bool
    reset_reason: str
    subject_states: _containers.RepeatedCompositeFieldContainer[WatchSubjectState]
    subject_state_unavailable: str
    def __init__(self, watch: _Optional[_Union[CohortWatch, _Mapping]] = ..., events: _Optional[_Iterable[_Union[WatchEventEnvelope, _Mapping]]] = ..., cursor_reset_required: _Optional[bool] = ..., reset_reason: _Optional[str] = ..., subject_states: _Optional[_Iterable[_Union[WatchSubjectState, _Mapping]]] = ..., subject_state_unavailable: _Optional[str] = ...) -> None: ...

class RequestCohortWatchActionRequest(_message.Message):
    __slots__ = ("watch_id", "expected_watch_revision", "idempotency_key", "kind", "target_run_id", "requested_by", "authority", "rationale", "message", "cooldown", "maximum_count", "waive_hard_validation_gate")
    WATCH_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_WATCH_REVISION_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    TARGET_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_BY_FIELD_NUMBER: _ClassVar[int]
    AUTHORITY_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    COOLDOWN_FIELD_NUMBER: _ClassVar[int]
    MAXIMUM_COUNT_FIELD_NUMBER: _ClassVar[int]
    WAIVE_HARD_VALIDATION_GATE_FIELD_NUMBER: _ClassVar[int]
    watch_id: str
    expected_watch_revision: int
    idempotency_key: str
    kind: WatchActionKind
    target_run_id: str
    requested_by: str
    authority: WatchAuthority
    rationale: str
    message: str
    cooldown: _duration_pb2.Duration
    maximum_count: int
    waive_hard_validation_gate: bool
    def __init__(self, watch_id: _Optional[str] = ..., expected_watch_revision: _Optional[int] = ..., idempotency_key: _Optional[str] = ..., kind: _Optional[_Union[WatchActionKind, str]] = ..., target_run_id: _Optional[str] = ..., requested_by: _Optional[str] = ..., authority: _Optional[_Union[WatchAuthority, str]] = ..., rationale: _Optional[str] = ..., message: _Optional[str] = ..., cooldown: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., maximum_count: _Optional[int] = ..., waive_hard_validation_gate: _Optional[bool] = ...) -> None: ...

class RequestCohortWatchActionResponse(_message.Message):
    __slots__ = ("action", "watch", "idempotent_replay")
    ACTION_FIELD_NUMBER: _ClassVar[int]
    WATCH_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENT_REPLAY_FIELD_NUMBER: _ClassVar[int]
    action: WatchAction
    watch: CohortWatch
    idempotent_replay: bool
    def __init__(self, action: _Optional[_Union[WatchAction, _Mapping]] = ..., watch: _Optional[_Union[CohortWatch, _Mapping]] = ..., idempotent_replay: _Optional[bool] = ...) -> None: ...

class ListCohortWatchActionsRequest(_message.Message):
    __slots__ = ("watch_id", "limit")
    WATCH_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    watch_id: str
    limit: int
    def __init__(self, watch_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListCohortWatchActionsResponse(_message.Message):
    __slots__ = ("actions",)
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    actions: _containers.RepeatedCompositeFieldContainer[WatchAction]
    def __init__(self, actions: _Optional[_Iterable[_Union[WatchAction, _Mapping]]] = ...) -> None: ...

class SupervisionPolicyDefinition(_message.Message):
    __slots__ = ("version", "event_count", "quiet_seconds", "friction_threshold", "terminal", "allowed_actions", "classifier_revision", "evaluator_digest")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    EVENT_COUNT_FIELD_NUMBER: _ClassVar[int]
    QUIET_SECONDS_FIELD_NUMBER: _ClassVar[int]
    FRICTION_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    CLASSIFIER_REVISION_FIELD_NUMBER: _ClassVar[int]
    EVALUATOR_DIGEST_FIELD_NUMBER: _ClassVar[int]
    version: str
    event_count: int
    quiet_seconds: int
    friction_threshold: float
    terminal: bool
    allowed_actions: _containers.RepeatedScalarFieldContainer[str]
    classifier_revision: str
    evaluator_digest: str
    def __init__(self, version: _Optional[str] = ..., event_count: _Optional[int] = ..., quiet_seconds: _Optional[int] = ..., friction_threshold: _Optional[float] = ..., terminal: _Optional[bool] = ..., allowed_actions: _Optional[_Iterable[str]] = ..., classifier_revision: _Optional[str] = ..., evaluator_digest: _Optional[str] = ...) -> None: ...

class SupervisionPolicyRecord(_message.Message):
    __slots__ = ("policy", "state", "digest", "supersedes", "created_by", "reviewed_by", "rejection_reason", "evaluation", "inference_identity_digest")
    POLICY_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDES_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    REVIEWED_BY_FIELD_NUMBER: _ClassVar[int]
    REJECTION_REASON_FIELD_NUMBER: _ClassVar[int]
    EVALUATION_FIELD_NUMBER: _ClassVar[int]
    INFERENCE_IDENTITY_DIGEST_FIELD_NUMBER: _ClassVar[int]
    policy: SupervisionPolicyDefinition
    state: SupervisionPolicyState
    digest: str
    supersedes: str
    created_by: str
    reviewed_by: str
    rejection_reason: str
    evaluation: SupervisionReplayReport
    inference_identity_digest: str
    def __init__(self, policy: _Optional[_Union[SupervisionPolicyDefinition, _Mapping]] = ..., state: _Optional[_Union[SupervisionPolicyState, str]] = ..., digest: _Optional[str] = ..., supersedes: _Optional[str] = ..., created_by: _Optional[str] = ..., reviewed_by: _Optional[str] = ..., rejection_reason: _Optional[str] = ..., evaluation: _Optional[_Union[SupervisionReplayReport, _Mapping]] = ..., inference_identity_digest: _Optional[str] = ...) -> None: ...

class SupervisionOutcomeRecord(_message.Message):
    __slots__ = ("outcome_id", "idempotency_key", "policy_version", "family_execution_id", "watch_id", "decision_id", "action_id", "child_run_id", "evidence_ids", "predicted_class", "observed_class", "overridden", "counterexample", "safety_violation", "completion_impact", "supersedes_outcome_id", "created_at", "expires_at", "completion_impact_observed")
    OUTCOME_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    FAMILY_EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    WATCH_ID_FIELD_NUMBER: _ClassVar[int]
    DECISION_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_ID_FIELD_NUMBER: _ClassVar[int]
    CHILD_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_IDS_FIELD_NUMBER: _ClassVar[int]
    PREDICTED_CLASS_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_CLASS_FIELD_NUMBER: _ClassVar[int]
    OVERRIDDEN_FIELD_NUMBER: _ClassVar[int]
    COUNTEREXAMPLE_FIELD_NUMBER: _ClassVar[int]
    SAFETY_VIOLATION_FIELD_NUMBER: _ClassVar[int]
    COMPLETION_IMPACT_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDES_OUTCOME_ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETION_IMPACT_OBSERVED_FIELD_NUMBER: _ClassVar[int]
    outcome_id: str
    idempotency_key: str
    policy_version: str
    family_execution_id: str
    watch_id: str
    decision_id: str
    action_id: str
    child_run_id: str
    evidence_ids: _containers.RepeatedScalarFieldContainer[str]
    predicted_class: str
    observed_class: str
    overridden: bool
    counterexample: bool
    safety_violation: bool
    completion_impact: float
    supersedes_outcome_id: str
    created_at: _timestamp_pb2.Timestamp
    expires_at: _timestamp_pb2.Timestamp
    completion_impact_observed: bool
    def __init__(self, outcome_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., policy_version: _Optional[str] = ..., family_execution_id: _Optional[str] = ..., watch_id: _Optional[str] = ..., decision_id: _Optional[str] = ..., action_id: _Optional[str] = ..., child_run_id: _Optional[str] = ..., evidence_ids: _Optional[_Iterable[str]] = ..., predicted_class: _Optional[str] = ..., observed_class: _Optional[str] = ..., overridden: _Optional[bool] = ..., counterexample: _Optional[bool] = ..., safety_violation: _Optional[bool] = ..., completion_impact: _Optional[float] = ..., supersedes_outcome_id: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completion_impact_observed: _Optional[bool] = ...) -> None: ...

class SupervisionReplayReport(_message.Message):
    __slots__ = ("version", "sample_count", "false_positives", "false_negatives", "safety_violations", "completion_impact", "rollout_samples", "replay_passed", "rollout_passed", "incumbent_version", "heldout_families", "candidate_errors", "incumbent_errors", "comparison_passed", "selection", "completion_impact_observed", "completion_impact_reason")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    FALSE_POSITIVES_FIELD_NUMBER: _ClassVar[int]
    FALSE_NEGATIVES_FIELD_NUMBER: _ClassVar[int]
    SAFETY_VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    COMPLETION_IMPACT_FIELD_NUMBER: _ClassVar[int]
    ROLLOUT_SAMPLES_FIELD_NUMBER: _ClassVar[int]
    REPLAY_PASSED_FIELD_NUMBER: _ClassVar[int]
    ROLLOUT_PASSED_FIELD_NUMBER: _ClassVar[int]
    INCUMBENT_VERSION_FIELD_NUMBER: _ClassVar[int]
    HELDOUT_FAMILIES_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_ERRORS_FIELD_NUMBER: _ClassVar[int]
    INCUMBENT_ERRORS_FIELD_NUMBER: _ClassVar[int]
    COMPARISON_PASSED_FIELD_NUMBER: _ClassVar[int]
    SELECTION_FIELD_NUMBER: _ClassVar[int]
    COMPLETION_IMPACT_OBSERVED_FIELD_NUMBER: _ClassVar[int]
    COMPLETION_IMPACT_REASON_FIELD_NUMBER: _ClassVar[int]
    version: str
    sample_count: int
    false_positives: int
    false_negatives: int
    safety_violations: int
    completion_impact: float
    rollout_samples: int
    replay_passed: bool
    rollout_passed: bool
    incumbent_version: str
    heldout_families: int
    candidate_errors: int
    incumbent_errors: int
    comparison_passed: bool
    selection: str
    completion_impact_observed: bool
    completion_impact_reason: str
    def __init__(self, version: _Optional[str] = ..., sample_count: _Optional[int] = ..., false_positives: _Optional[int] = ..., false_negatives: _Optional[int] = ..., safety_violations: _Optional[int] = ..., completion_impact: _Optional[float] = ..., rollout_samples: _Optional[int] = ..., replay_passed: _Optional[bool] = ..., rollout_passed: _Optional[bool] = ..., incumbent_version: _Optional[str] = ..., heldout_families: _Optional[int] = ..., candidate_errors: _Optional[int] = ..., incumbent_errors: _Optional[int] = ..., comparison_passed: _Optional[bool] = ..., selection: _Optional[str] = ..., completion_impact_observed: _Optional[bool] = ..., completion_impact_reason: _Optional[str] = ...) -> None: ...

class GetSupervisionPolicyRequest(_message.Message):
    __slots__ = ("version",)
    VERSION_FIELD_NUMBER: _ClassVar[int]
    version: str
    def __init__(self, version: _Optional[str] = ...) -> None: ...

class CreateSupervisionPolicyCandidateRequest(_message.Message):
    __slots__ = ("policy", "supersedes", "created_by")
    POLICY_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDES_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    policy: SupervisionPolicyDefinition
    supersedes: str
    created_by: str
    def __init__(self, policy: _Optional[_Union[SupervisionPolicyDefinition, _Mapping]] = ..., supersedes: _Optional[str] = ..., created_by: _Optional[str] = ...) -> None: ...

class RecordSupervisionOutcomeRequest(_message.Message):
    __slots__ = ("outcome",)
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    outcome: SupervisionOutcomeRecord
    def __init__(self, outcome: _Optional[_Union[SupervisionOutcomeRecord, _Mapping]] = ...) -> None: ...

class RecordSupervisionOutcomeResponse(_message.Message):
    __slots__ = ("outcome", "idempotent_replay", "source_ledger_synced", "source_ledger_id", "degradation_reason")
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENT_REPLAY_FIELD_NUMBER: _ClassVar[int]
    SOURCE_LEDGER_SYNCED_FIELD_NUMBER: _ClassVar[int]
    SOURCE_LEDGER_ID_FIELD_NUMBER: _ClassVar[int]
    DEGRADATION_REASON_FIELD_NUMBER: _ClassVar[int]
    outcome: SupervisionOutcomeRecord
    idempotent_replay: bool
    source_ledger_synced: bool
    source_ledger_id: str
    degradation_reason: str
    def __init__(self, outcome: _Optional[_Union[SupervisionOutcomeRecord, _Mapping]] = ..., idempotent_replay: _Optional[bool] = ..., source_ledger_synced: _Optional[bool] = ..., source_ledger_id: _Optional[str] = ..., degradation_reason: _Optional[str] = ...) -> None: ...

class EvaluateSupervisionPolicyRequest(_message.Message):
    __slots__ = ("version", "rollout_samples", "min_samples", "max_false_positive_rate", "max_false_negative_rate", "min_rollout_samples")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    ROLLOUT_SAMPLES_FIELD_NUMBER: _ClassVar[int]
    MIN_SAMPLES_FIELD_NUMBER: _ClassVar[int]
    MAX_FALSE_POSITIVE_RATE_FIELD_NUMBER: _ClassVar[int]
    MAX_FALSE_NEGATIVE_RATE_FIELD_NUMBER: _ClassVar[int]
    MIN_ROLLOUT_SAMPLES_FIELD_NUMBER: _ClassVar[int]
    version: str
    rollout_samples: int
    min_samples: int
    max_false_positive_rate: float
    max_false_negative_rate: float
    min_rollout_samples: int
    def __init__(self, version: _Optional[str] = ..., rollout_samples: _Optional[int] = ..., min_samples: _Optional[int] = ..., max_false_positive_rate: _Optional[float] = ..., max_false_negative_rate: _Optional[float] = ..., min_rollout_samples: _Optional[int] = ...) -> None: ...

class PromoteSupervisionPolicyRequest(_message.Message):
    __slots__ = ("version", "reviewed_by")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    REVIEWED_BY_FIELD_NUMBER: _ClassVar[int]
    version: str
    reviewed_by: str
    def __init__(self, version: _Optional[str] = ..., reviewed_by: _Optional[str] = ...) -> None: ...

class RejectSupervisionPolicyRequest(_message.Message):
    __slots__ = ("version", "reviewed_by", "reason")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    REVIEWED_BY_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    version: str
    reviewed_by: str
    reason: str
    def __init__(self, version: _Optional[str] = ..., reviewed_by: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class RollbackSupervisionPolicyRequest(_message.Message):
    __slots__ = ("active_version", "reviewed_by")
    ACTIVE_VERSION_FIELD_NUMBER: _ClassVar[int]
    REVIEWED_BY_FIELD_NUMBER: _ClassVar[int]
    active_version: str
    reviewed_by: str
    def __init__(self, active_version: _Optional[str] = ..., reviewed_by: _Optional[str] = ...) -> None: ...

class SetSupervisionPolicyDisabledRequest(_message.Message):
    __slots__ = ("disabled", "reason", "actor")
    DISABLED_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    disabled: bool
    reason: str
    actor: str
    def __init__(self, disabled: _Optional[bool] = ..., reason: _Optional[str] = ..., actor: _Optional[str] = ...) -> None: ...

class SupervisionPolicyControl(_message.Message):
    __slots__ = ("disabled", "reason")
    DISABLED_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    disabled: bool
    reason: str
    def __init__(self, disabled: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class ListSupervisionOutcomesRequest(_message.Message):
    __slots__ = ("policy_version", "limit", "watch_id")
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    WATCH_ID_FIELD_NUMBER: _ClassVar[int]
    policy_version: str
    limit: int
    watch_id: str
    def __init__(self, policy_version: _Optional[str] = ..., limit: _Optional[int] = ..., watch_id: _Optional[str] = ...) -> None: ...

class SupervisionEvidenceCoverage(_message.Message):
    __slots__ = ("outcomes", "assessed_outcomes", "decisions", "actions", "applied_actions", "children", "families", "measured_impact_outcomes", "population")
    OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    ASSESSED_OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    DECISIONS_FIELD_NUMBER: _ClassVar[int]
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    APPLIED_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    CHILDREN_FIELD_NUMBER: _ClassVar[int]
    FAMILIES_FIELD_NUMBER: _ClassVar[int]
    MEASURED_IMPACT_OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    POPULATION_FIELD_NUMBER: _ClassVar[int]
    outcomes: int
    assessed_outcomes: int
    decisions: int
    actions: int
    applied_actions: int
    children: int
    families: int
    measured_impact_outcomes: int
    population: str
    def __init__(self, outcomes: _Optional[int] = ..., assessed_outcomes: _Optional[int] = ..., decisions: _Optional[int] = ..., actions: _Optional[int] = ..., applied_actions: _Optional[int] = ..., children: _Optional[int] = ..., families: _Optional[int] = ..., measured_impact_outcomes: _Optional[int] = ..., population: _Optional[str] = ...) -> None: ...

class ListSupervisionOutcomesResponse(_message.Message):
    __slots__ = ("outcomes", "coverage", "truncated")
    OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    outcomes: _containers.RepeatedCompositeFieldContainer[SupervisionOutcomeRecord]
    coverage: SupervisionEvidenceCoverage
    truncated: bool
    def __init__(self, outcomes: _Optional[_Iterable[_Union[SupervisionOutcomeRecord, _Mapping]]] = ..., coverage: _Optional[_Union[SupervisionEvidenceCoverage, _Mapping]] = ..., truncated: _Optional[bool] = ...) -> None: ...

class WatchSubjectState(_message.Message):
    __slots__ = ("run_id", "status", "terminal", "friction_unavailable", "friction_through")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_FIELD_NUMBER: _ClassVar[int]
    FRICTION_UNAVAILABLE_FIELD_NUMBER: _ClassVar[int]
    FRICTION_THROUGH_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    status: str
    terminal: bool
    friction_unavailable: bool
    friction_through: _timestamp_pb2.Timestamp
    def __init__(self, run_id: _Optional[str] = ..., status: _Optional[str] = ..., terminal: _Optional[bool] = ..., friction_unavailable: _Optional[bool] = ..., friction_through: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
