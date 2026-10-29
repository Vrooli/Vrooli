import datetime

from agent_manager.v1.domain import watch_pb2 as _watch_pb2
from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EffortFreshness(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EFFORT_FRESHNESS_UNSPECIFIED: _ClassVar[EffortFreshness]
    EFFORT_FRESHNESS_FRESH: _ClassVar[EffortFreshness]
    EFFORT_FRESHNESS_STALE: _ClassVar[EffortFreshness]
    EFFORT_FRESHNESS_UNAVAILABLE: _ClassVar[EffortFreshness]
EFFORT_FRESHNESS_UNSPECIFIED: EffortFreshness
EFFORT_FRESHNESS_FRESH: EffortFreshness
EFFORT_FRESHNESS_STALE: EffortFreshness
EFFORT_FRESHNESS_UNAVAILABLE: EffortFreshness

class EffortSubject(_message.Message):
    __slots__ = ("owner", "kind", "reference", "run_id", "role", "assignment")
    OWNER_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    ASSIGNMENT_FIELD_NUMBER: _ClassVar[int]
    owner: str
    kind: str
    reference: str
    run_id: str
    role: str
    assignment: str
    def __init__(self, owner: _Optional[str] = ..., kind: _Optional[str] = ..., reference: _Optional[str] = ..., run_id: _Optional[str] = ..., role: _Optional[str] = ..., assignment: _Optional[str] = ...) -> None: ...

class EffortEnrollment(_message.Message):
    __slots__ = ("effort_ref", "display_name", "destination_ref", "target_revision", "source_revision", "authority_ref", "supervisor_run_id", "subjects", "revision", "withdrawn", "workspace", "work_shape", "updated_at", "authorized_by", "withdrawal_reason", "delivery")
    EFFORT_REF_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_REF_FIELD_NUMBER: _ClassVar[int]
    TARGET_REVISION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REVISION_FIELD_NUMBER: _ClassVar[int]
    AUTHORITY_REF_FIELD_NUMBER: _ClassVar[int]
    SUPERVISOR_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SUBJECTS_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    WITHDRAWN_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    WORK_SHAPE_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    AUTHORIZED_BY_FIELD_NUMBER: _ClassVar[int]
    WITHDRAWAL_REASON_FIELD_NUMBER: _ClassVar[int]
    DELIVERY_FIELD_NUMBER: _ClassVar[int]
    effort_ref: str
    display_name: str
    destination_ref: str
    target_revision: str
    source_revision: str
    authority_ref: str
    supervisor_run_id: str
    subjects: _containers.RepeatedCompositeFieldContainer[EffortSubject]
    revision: int
    withdrawn: bool
    workspace: str
    work_shape: str
    updated_at: _timestamp_pb2.Timestamp
    authorized_by: str
    withdrawal_reason: str
    delivery: EffortDelivery
    def __init__(self, effort_ref: _Optional[str] = ..., display_name: _Optional[str] = ..., destination_ref: _Optional[str] = ..., target_revision: _Optional[str] = ..., source_revision: _Optional[str] = ..., authority_ref: _Optional[str] = ..., supervisor_run_id: _Optional[str] = ..., subjects: _Optional[_Iterable[_Union[EffortSubject, _Mapping]]] = ..., revision: _Optional[int] = ..., withdrawn: _Optional[bool] = ..., workspace: _Optional[str] = ..., work_shape: _Optional[str] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., authorized_by: _Optional[str] = ..., withdrawal_reason: _Optional[str] = ..., delivery: _Optional[_Union[EffortDelivery, _Mapping]] = ...) -> None: ...

class EffortDelivery(_message.Message):
    __slots__ = ("commission_id", "budget_seconds", "max_workers", "state", "started_at", "paused_at", "paused_seconds", "reason", "stopped_at", "finalized_at", "outcome", "decision_id")
    COMMISSION_ID_FIELD_NUMBER: _ClassVar[int]
    BUDGET_SECONDS_FIELD_NUMBER: _ClassVar[int]
    MAX_WORKERS_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    PAUSED_AT_FIELD_NUMBER: _ClassVar[int]
    PAUSED_SECONDS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    STOPPED_AT_FIELD_NUMBER: _ClassVar[int]
    FINALIZED_AT_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    DECISION_ID_FIELD_NUMBER: _ClassVar[int]
    commission_id: str
    budget_seconds: int
    max_workers: int
    state: str
    started_at: _timestamp_pb2.Timestamp
    paused_at: _timestamp_pb2.Timestamp
    paused_seconds: int
    reason: str
    stopped_at: _timestamp_pb2.Timestamp
    finalized_at: _timestamp_pb2.Timestamp
    outcome: EffortGoalHome
    decision_id: str
    def __init__(self, commission_id: _Optional[str] = ..., budget_seconds: _Optional[int] = ..., max_workers: _Optional[int] = ..., state: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., paused_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., paused_seconds: _Optional[int] = ..., reason: _Optional[str] = ..., stopped_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., finalized_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., outcome: _Optional[_Union[EffortGoalHome, _Mapping]] = ..., decision_id: _Optional[str] = ...) -> None: ...

class EffortDeliveryPressure(_message.Message):
    __slots__ = ("commission_id", "phase", "elapsed_seconds", "budget_seconds", "paused_seconds", "hard_stop_at", "checkpoint_deadline", "instruction", "disposition")
    COMMISSION_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    ELAPSED_SECONDS_FIELD_NUMBER: _ClassVar[int]
    BUDGET_SECONDS_FIELD_NUMBER: _ClassVar[int]
    PAUSED_SECONDS_FIELD_NUMBER: _ClassVar[int]
    HARD_STOP_AT_FIELD_NUMBER: _ClassVar[int]
    CHECKPOINT_DEADLINE_FIELD_NUMBER: _ClassVar[int]
    INSTRUCTION_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    commission_id: str
    phase: str
    elapsed_seconds: int
    budget_seconds: int
    paused_seconds: int
    hard_stop_at: _timestamp_pb2.Timestamp
    checkpoint_deadline: _timestamp_pb2.Timestamp
    instruction: str
    disposition: str
    def __init__(self, commission_id: _Optional[str] = ..., phase: _Optional[str] = ..., elapsed_seconds: _Optional[int] = ..., budget_seconds: _Optional[int] = ..., paused_seconds: _Optional[int] = ..., hard_stop_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., checkpoint_deadline: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., instruction: _Optional[str] = ..., disposition: _Optional[str] = ...) -> None: ...

class EffortDiscoveryFinding(_message.Message):
    __slots__ = ("source", "code", "reason")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    source: str
    code: str
    reason: str
    def __init__(self, source: _Optional[str] = ..., code: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class EffortDiscovery(_message.Message):
    __slots__ = ("generation", "change_identity", "last_scan_at", "last_successful_scan_at", "scan_limit", "scanned_count", "partial", "findings", "byte_limit", "root", "scan_cursor", "directory_limit", "standing_allowance_ref", "standing_usage")
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    CHANGE_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    LAST_SCAN_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_SUCCESSFUL_SCAN_AT_FIELD_NUMBER: _ClassVar[int]
    SCAN_LIMIT_FIELD_NUMBER: _ClassVar[int]
    SCANNED_COUNT_FIELD_NUMBER: _ClassVar[int]
    PARTIAL_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    BYTE_LIMIT_FIELD_NUMBER: _ClassVar[int]
    ROOT_FIELD_NUMBER: _ClassVar[int]
    SCAN_CURSOR_FIELD_NUMBER: _ClassVar[int]
    DIRECTORY_LIMIT_FIELD_NUMBER: _ClassVar[int]
    STANDING_ALLOWANCE_REF_FIELD_NUMBER: _ClassVar[int]
    STANDING_USAGE_FIELD_NUMBER: _ClassVar[int]
    generation: int
    change_identity: str
    last_scan_at: _timestamp_pb2.Timestamp
    last_successful_scan_at: _timestamp_pb2.Timestamp
    scan_limit: int
    scanned_count: int
    partial: bool
    findings: _containers.RepeatedCompositeFieldContainer[EffortDiscoveryFinding]
    byte_limit: int
    root: str
    scan_cursor: str
    directory_limit: int
    standing_allowance_ref: str
    standing_usage: EffortUsage
    def __init__(self, generation: _Optional[int] = ..., change_identity: _Optional[str] = ..., last_scan_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_successful_scan_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., scan_limit: _Optional[int] = ..., scanned_count: _Optional[int] = ..., partial: _Optional[bool] = ..., findings: _Optional[_Iterable[_Union[EffortDiscoveryFinding, _Mapping]]] = ..., byte_limit: _Optional[int] = ..., root: _Optional[str] = ..., scan_cursor: _Optional[str] = ..., directory_limit: _Optional[int] = ..., standing_allowance_ref: _Optional[str] = ..., standing_usage: _Optional[_Union[EffortUsage, _Mapping]] = ...) -> None: ...

class EffortUsage(_message.Message):
    __slots__ = ("tokens", "reported_cost_usd", "agent_seconds", "observed_runs", "declared_runs", "partial", "limitations", "source")
    TOKENS_FIELD_NUMBER: _ClassVar[int]
    REPORTED_COST_USD_FIELD_NUMBER: _ClassVar[int]
    AGENT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_RUNS_FIELD_NUMBER: _ClassVar[int]
    DECLARED_RUNS_FIELD_NUMBER: _ClassVar[int]
    PARTIAL_FIELD_NUMBER: _ClassVar[int]
    LIMITATIONS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    tokens: int
    reported_cost_usd: float
    agent_seconds: float
    observed_runs: int
    declared_runs: int
    partial: bool
    limitations: _containers.RepeatedScalarFieldContainer[str]
    source: str
    def __init__(self, tokens: _Optional[int] = ..., reported_cost_usd: _Optional[float] = ..., agent_seconds: _Optional[float] = ..., observed_runs: _Optional[int] = ..., declared_runs: _Optional[int] = ..., partial: _Optional[bool] = ..., limitations: _Optional[_Iterable[str]] = ..., source: _Optional[str] = ...) -> None: ...

class EffortQuotaObservation(_message.Message):
    __slots__ = ("provider", "pool", "window", "observed_at", "standing", "used_percent", "window_minutes", "reset_at", "source_run_id", "freshness", "provenance", "uncertainty", "evidence_ref")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    POOL_FIELD_NUMBER: _ClassVar[int]
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    STANDING_FIELD_NUMBER: _ClassVar[int]
    USED_PERCENT_FIELD_NUMBER: _ClassVar[int]
    WINDOW_MINUTES_FIELD_NUMBER: _ClassVar[int]
    RESET_AT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    UNCERTAINTY_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REF_FIELD_NUMBER: _ClassVar[int]
    provider: str
    pool: str
    window: str
    observed_at: _timestamp_pb2.Timestamp
    standing: str
    used_percent: float
    window_minutes: int
    reset_at: _timestamp_pb2.Timestamp
    source_run_id: str
    freshness: str
    provenance: str
    uncertainty: str
    evidence_ref: str
    def __init__(self, provider: _Optional[str] = ..., pool: _Optional[str] = ..., window: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., standing: _Optional[str] = ..., used_percent: _Optional[float] = ..., window_minutes: _Optional[int] = ..., reset_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., source_run_id: _Optional[str] = ..., freshness: _Optional[str] = ..., provenance: _Optional[str] = ..., uncertainty: _Optional[str] = ..., evidence_ref: _Optional[str] = ...) -> None: ...

class EffortAssignment(_message.Message):
    __slots__ = ("subject", "runtime_state", "requested_model", "effective_model", "requested_runner", "effective_runner", "requested_reasoning", "effective_reasoning", "usage", "observed_at", "unavailable_reason", "execution_identity")
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_STATE_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_MODEL_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_MODEL_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_RUNNER_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_RUNNER_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_REASONING_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_REASONING_FIELD_NUMBER: _ClassVar[int]
    USAGE_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_REASON_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    subject: EffortSubject
    runtime_state: str
    requested_model: str
    effective_model: str
    requested_runner: str
    effective_runner: str
    requested_reasoning: str
    effective_reasoning: str
    usage: EffortUsage
    observed_at: _timestamp_pb2.Timestamp
    unavailable_reason: str
    execution_identity: str
    def __init__(self, subject: _Optional[_Union[EffortSubject, _Mapping]] = ..., runtime_state: _Optional[str] = ..., requested_model: _Optional[str] = ..., effective_model: _Optional[str] = ..., requested_runner: _Optional[str] = ..., effective_runner: _Optional[str] = ..., requested_reasoning: _Optional[str] = ..., effective_reasoning: _Optional[str] = ..., usage: _Optional[_Union[EffortUsage, _Mapping]] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., unavailable_reason: _Optional[str] = ..., execution_identity: _Optional[str] = ...) -> None: ...

class EffortOutcomeStanding(_message.Message):
    __slots__ = ("state", "attribution", "evidence_refs", "met_count", "required_count", "limitations")
    STATE_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    MET_COUNT_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_COUNT_FIELD_NUMBER: _ClassVar[int]
    LIMITATIONS_FIELD_NUMBER: _ClassVar[int]
    state: str
    attribution: str
    evidence_refs: _containers.RepeatedScalarFieldContainer[str]
    met_count: int
    required_count: int
    limitations: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, state: _Optional[str] = ..., attribution: _Optional[str] = ..., evidence_refs: _Optional[_Iterable[str]] = ..., met_count: _Optional[int] = ..., required_count: _Optional[int] = ..., limitations: _Optional[_Iterable[str]] = ...) -> None: ...

class EffortBoardRow(_message.Message):
    __slots__ = ("enrollment", "observed_at", "freshness", "runtime_state", "outcome_standing", "assignments", "pending_operations", "blockers", "next_action", "rationale", "usage", "limitations", "evidence_refs", "change_identity", "visibility_change_identity", "quota_observations", "goal_home", "delivery_pressure")
    ENROLLMENT_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_STATE_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_STANDING_FIELD_NUMBER: _ClassVar[int]
    ASSIGNMENTS_FIELD_NUMBER: _ClassVar[int]
    PENDING_OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    BLOCKERS_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    USAGE_FIELD_NUMBER: _ClassVar[int]
    LIMITATIONS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    CHANGE_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    VISIBILITY_CHANGE_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    QUOTA_OBSERVATIONS_FIELD_NUMBER: _ClassVar[int]
    GOAL_HOME_FIELD_NUMBER: _ClassVar[int]
    DELIVERY_PRESSURE_FIELD_NUMBER: _ClassVar[int]
    enrollment: EffortEnrollment
    observed_at: _timestamp_pb2.Timestamp
    freshness: EffortFreshness
    runtime_state: str
    outcome_standing: EffortOutcomeStanding
    assignments: _containers.RepeatedCompositeFieldContainer[EffortAssignment]
    pending_operations: _containers.RepeatedScalarFieldContainer[str]
    blockers: _containers.RepeatedScalarFieldContainer[str]
    next_action: str
    rationale: str
    usage: EffortUsage
    limitations: _containers.RepeatedScalarFieldContainer[str]
    evidence_refs: _containers.RepeatedScalarFieldContainer[str]
    change_identity: str
    visibility_change_identity: str
    quota_observations: _containers.RepeatedCompositeFieldContainer[EffortQuotaObservation]
    goal_home: EffortGoalHome
    delivery_pressure: EffortDeliveryPressure
    def __init__(self, enrollment: _Optional[_Union[EffortEnrollment, _Mapping]] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., freshness: _Optional[_Union[EffortFreshness, str]] = ..., runtime_state: _Optional[str] = ..., outcome_standing: _Optional[_Union[EffortOutcomeStanding, _Mapping]] = ..., assignments: _Optional[_Iterable[_Union[EffortAssignment, _Mapping]]] = ..., pending_operations: _Optional[_Iterable[str]] = ..., blockers: _Optional[_Iterable[str]] = ..., next_action: _Optional[str] = ..., rationale: _Optional[str] = ..., usage: _Optional[_Union[EffortUsage, _Mapping]] = ..., limitations: _Optional[_Iterable[str]] = ..., evidence_refs: _Optional[_Iterable[str]] = ..., change_identity: _Optional[str] = ..., visibility_change_identity: _Optional[str] = ..., quota_observations: _Optional[_Iterable[_Union[EffortQuotaObservation, _Mapping]]] = ..., goal_home: _Optional[_Union[EffortGoalHome, _Mapping]] = ..., delivery_pressure: _Optional[_Union[EffortDeliveryPressure, _Mapping]] = ...) -> None: ...

class EffortBoard(_message.Message):
    __slots__ = ("rows", "discovery", "observed_at", "active_count", "partial", "limitations", "next_page_token", "change_identity", "quota_observations")
    ROWS_FIELD_NUMBER: _ClassVar[int]
    DISCOVERY_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_COUNT_FIELD_NUMBER: _ClassVar[int]
    PARTIAL_FIELD_NUMBER: _ClassVar[int]
    LIMITATIONS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    CHANGE_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    QUOTA_OBSERVATIONS_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[EffortBoardRow]
    discovery: EffortDiscovery
    observed_at: _timestamp_pb2.Timestamp
    active_count: int
    partial: bool
    limitations: _containers.RepeatedScalarFieldContainer[str]
    next_page_token: str
    change_identity: str
    quota_observations: _containers.RepeatedCompositeFieldContainer[EffortQuotaObservation]
    def __init__(self, rows: _Optional[_Iterable[_Union[EffortBoardRow, _Mapping]]] = ..., discovery: _Optional[_Union[EffortDiscovery, _Mapping]] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., active_count: _Optional[int] = ..., partial: _Optional[bool] = ..., limitations: _Optional[_Iterable[str]] = ..., next_page_token: _Optional[str] = ..., change_identity: _Optional[str] = ..., quota_observations: _Optional[_Iterable[_Union[EffortQuotaObservation, _Mapping]]] = ...) -> None: ...

class GetEffortBoardRequest(_message.Message):
    __slots__ = ("effort_ref", "page_size", "page_token")
    EFFORT_REF_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    effort_ref: str
    page_size: int
    page_token: str
    def __init__(self, effort_ref: _Optional[str] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListEffortsRequest(_message.Message):
    __slots__ = ("page_size", "page_token")
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    page_size: int
    page_token: str
    def __init__(self, page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListEffortsResponse(_message.Message):
    __slots__ = ("efforts", "next_page_token", "active_count")
    EFFORTS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_COUNT_FIELD_NUMBER: _ClassVar[int]
    efforts: _containers.RepeatedCompositeFieldContainer[EffortEnrollment]
    next_page_token: str
    active_count: int
    def __init__(self, efforts: _Optional[_Iterable[_Union[EffortEnrollment, _Mapping]]] = ..., next_page_token: _Optional[str] = ..., active_count: _Optional[int] = ...) -> None: ...

class EnrollEffortRequest(_message.Message):
    __slots__ = ("enrollment", "expected_revision", "idempotency_key")
    ENROLLMENT_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    enrollment: EffortEnrollment
    expected_revision: int
    idempotency_key: str
    def __init__(self, enrollment: _Optional[_Union[EffortEnrollment, _Mapping]] = ..., expected_revision: _Optional[int] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class ReconcileEffortMetadataRequest(_message.Message):
    __slots__ = ("enrollment", "expected_revision", "idempotency_key", "authority")
    ENROLLMENT_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    AUTHORITY_FIELD_NUMBER: _ClassVar[int]
    enrollment: EffortEnrollment
    expected_revision: int
    idempotency_key: str
    authority: _watch_pb2.WatchAuthority
    def __init__(self, enrollment: _Optional[_Union[EffortEnrollment, _Mapping]] = ..., expected_revision: _Optional[int] = ..., idempotency_key: _Optional[str] = ..., authority: _Optional[_Union[_watch_pb2.WatchAuthority, str]] = ...) -> None: ...

class WithdrawEffortRequest(_message.Message):
    __slots__ = ("effort_ref", "expected_revision", "reason", "idempotency_key")
    EFFORT_REF_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    effort_ref: str
    expected_revision: int
    reason: str
    idempotency_key: str
    def __init__(self, effort_ref: _Optional[str] = ..., expected_revision: _Optional[int] = ..., reason: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class ReconcileEffortDiscoveryRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class EffortCapUsage(_message.Message):
    __slots__ = ("run_id", "role", "weighted_tokens", "cap", "mode")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    WEIGHTED_TOKENS_FIELD_NUMBER: _ClassVar[int]
    CAP_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    role: str
    weighted_tokens: float
    cap: int
    mode: str
    def __init__(self, run_id: _Optional[str] = ..., role: _Optional[str] = ..., weighted_tokens: _Optional[float] = ..., cap: _Optional[int] = ..., mode: _Optional[str] = ...) -> None: ...

class SurfaceProgress(_message.Message):
    __slots__ = ("surface", "target", "state", "owner_run")
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    OWNER_RUN_FIELD_NUMBER: _ClassVar[int]
    surface: str
    target: str
    state: str
    owner_run: str
    def __init__(self, surface: _Optional[str] = ..., target: _Optional[str] = ..., state: _Optional[str] = ..., owner_run: _Optional[str] = ...) -> None: ...

class EffortActiveUnit(_message.Message):
    __slots__ = ("unit_id", "status", "workers")
    UNIT_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    WORKERS_FIELD_NUMBER: _ClassVar[int]
    unit_id: str
    status: str
    workers: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, unit_id: _Optional[str] = ..., status: _Optional[str] = ..., workers: _Optional[_Iterable[str]] = ...) -> None: ...

class EffortGoalHome(_message.Message):
    __slots__ = ("path", "state", "limitations", "destination_metric", "destination_op", "destination_target", "current_value", "current_as_of", "active_unit", "active_status", "active_workers", "last_accepted_unit", "last_accepted_at", "last_yield", "needs_operator", "open_workarounds", "open_feedback_ids", "open_decisions", "weighted_tokens_per_accepted_unit", "accepted_units_measured", "cap_usage", "canary_status", "lint_findings", "read_at", "budget_seconds", "surface_scoreboard", "active_units", "handoff")
    PATH_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    LIMITATIONS_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_METRIC_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_OP_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_TARGET_FIELD_NUMBER: _ClassVar[int]
    CURRENT_VALUE_FIELD_NUMBER: _ClassVar[int]
    CURRENT_AS_OF_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_UNIT_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_STATUS_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_WORKERS_FIELD_NUMBER: _ClassVar[int]
    LAST_ACCEPTED_UNIT_FIELD_NUMBER: _ClassVar[int]
    LAST_ACCEPTED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_YIELD_FIELD_NUMBER: _ClassVar[int]
    NEEDS_OPERATOR_FIELD_NUMBER: _ClassVar[int]
    OPEN_WORKAROUNDS_FIELD_NUMBER: _ClassVar[int]
    OPEN_FEEDBACK_IDS_FIELD_NUMBER: _ClassVar[int]
    OPEN_DECISIONS_FIELD_NUMBER: _ClassVar[int]
    WEIGHTED_TOKENS_PER_ACCEPTED_UNIT_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_UNITS_MEASURED_FIELD_NUMBER: _ClassVar[int]
    CAP_USAGE_FIELD_NUMBER: _ClassVar[int]
    CANARY_STATUS_FIELD_NUMBER: _ClassVar[int]
    LINT_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    READ_AT_FIELD_NUMBER: _ClassVar[int]
    BUDGET_SECONDS_FIELD_NUMBER: _ClassVar[int]
    SURFACE_SCOREBOARD_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_UNITS_FIELD_NUMBER: _ClassVar[int]
    HANDOFF_FIELD_NUMBER: _ClassVar[int]
    path: str
    state: str
    limitations: _containers.RepeatedScalarFieldContainer[str]
    destination_metric: str
    destination_op: str
    destination_target: float
    current_value: float
    current_as_of: _timestamp_pb2.Timestamp
    active_unit: str
    active_status: str
    active_workers: _containers.RepeatedScalarFieldContainer[str]
    last_accepted_unit: str
    last_accepted_at: _timestamp_pb2.Timestamp
    last_yield: str
    needs_operator: _containers.RepeatedScalarFieldContainer[str]
    open_workarounds: int
    open_feedback_ids: _containers.RepeatedScalarFieldContainer[str]
    open_decisions: _containers.RepeatedCompositeFieldContainer[EffortDecision]
    weighted_tokens_per_accepted_unit: float
    accepted_units_measured: int
    cap_usage: _containers.RepeatedCompositeFieldContainer[EffortCapUsage]
    canary_status: str
    lint_findings: _containers.RepeatedScalarFieldContainer[str]
    read_at: _timestamp_pb2.Timestamp
    budget_seconds: int
    surface_scoreboard: _containers.RepeatedCompositeFieldContainer[SurfaceProgress]
    active_units: _containers.RepeatedCompositeFieldContainer[EffortActiveUnit]
    handoff: str
    def __init__(self, path: _Optional[str] = ..., state: _Optional[str] = ..., limitations: _Optional[_Iterable[str]] = ..., destination_metric: _Optional[str] = ..., destination_op: _Optional[str] = ..., destination_target: _Optional[float] = ..., current_value: _Optional[float] = ..., current_as_of: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., active_unit: _Optional[str] = ..., active_status: _Optional[str] = ..., active_workers: _Optional[_Iterable[str]] = ..., last_accepted_unit: _Optional[str] = ..., last_accepted_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_yield: _Optional[str] = ..., needs_operator: _Optional[_Iterable[str]] = ..., open_workarounds: _Optional[int] = ..., open_feedback_ids: _Optional[_Iterable[str]] = ..., open_decisions: _Optional[_Iterable[_Union[EffortDecision, _Mapping]]] = ..., weighted_tokens_per_accepted_unit: _Optional[float] = ..., accepted_units_measured: _Optional[int] = ..., cap_usage: _Optional[_Iterable[_Union[EffortCapUsage, _Mapping]]] = ..., canary_status: _Optional[str] = ..., lint_findings: _Optional[_Iterable[str]] = ..., read_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., budget_seconds: _Optional[int] = ..., surface_scoreboard: _Optional[_Iterable[_Union[SurfaceProgress, _Mapping]]] = ..., active_units: _Optional[_Iterable[_Union[EffortActiveUnit, _Mapping]]] = ..., handoff: _Optional[str] = ...) -> None: ...

class EffortSignal(_message.Message):
    __slots__ = ("seq", "effort_ref", "kind", "run_id", "detail", "created_at", "newly_recorded")
    SEQ_FIELD_NUMBER: _ClassVar[int]
    EFFORT_REF_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    NEWLY_RECORDED_FIELD_NUMBER: _ClassVar[int]
    seq: int
    effort_ref: str
    kind: str
    run_id: str
    detail: str
    created_at: _timestamp_pb2.Timestamp
    newly_recorded: bool
    def __init__(self, seq: _Optional[int] = ..., effort_ref: _Optional[str] = ..., kind: _Optional[str] = ..., run_id: _Optional[str] = ..., detail: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., newly_recorded: _Optional[bool] = ...) -> None: ...

class ListEffortSignalsRequest(_message.Message):
    __slots__ = ("effort_ref", "after_seq", "limit")
    EFFORT_REF_FIELD_NUMBER: _ClassVar[int]
    AFTER_SEQ_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    effort_ref: str
    after_seq: int
    limit: int
    def __init__(self, effort_ref: _Optional[str] = ..., after_seq: _Optional[int] = ..., limit: _Optional[int] = ...) -> None: ...

class ListEffortSignalsResponse(_message.Message):
    __slots__ = ("signals", "latest_seq")
    SIGNALS_FIELD_NUMBER: _ClassVar[int]
    LATEST_SEQ_FIELD_NUMBER: _ClassVar[int]
    signals: _containers.RepeatedCompositeFieldContainer[EffortSignal]
    latest_seq: int
    def __init__(self, signals: _Optional[_Iterable[_Union[EffortSignal, _Mapping]]] = ..., latest_seq: _Optional[int] = ...) -> None: ...

class RecordEffortSignalRequest(_message.Message):
    __slots__ = ("effort_ref", "kind", "run_id", "detail", "dedupe_key")
    EFFORT_REF_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    DEDUPE_KEY_FIELD_NUMBER: _ClassVar[int]
    effort_ref: str
    kind: str
    run_id: str
    detail: str
    dedupe_key: str
    def __init__(self, effort_ref: _Optional[str] = ..., kind: _Optional[str] = ..., run_id: _Optional[str] = ..., detail: _Optional[str] = ..., dedupe_key: _Optional[str] = ...) -> None: ...

class EffortDecisionOption(_message.Message):
    __slots__ = ("key", "label")
    KEY_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    key: str
    label: str
    def __init__(self, key: _Optional[str] = ..., label: _Optional[str] = ...) -> None: ...

class EffortDecision(_message.Message):
    __slots__ = ("effort_ref", "goal_home", "decision_id", "question", "options", "recommended", "recommendation_reason", "default_answer", "reversible", "urgency", "note", "context_url", "deadline", "state", "answer", "answer_label", "answer_note", "answered_via", "late", "answered_at", "resolution_reason", "event_id", "ask_id", "raised_by_run_id", "created_at", "published_at", "publish_error", "recorded_at", "big_picture", "technical_details")
    EFFORT_REF_FIELD_NUMBER: _ClassVar[int]
    GOAL_HOME_FIELD_NUMBER: _ClassVar[int]
    DECISION_ID_FIELD_NUMBER: _ClassVar[int]
    QUESTION_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATION_REASON_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_ANSWER_FIELD_NUMBER: _ClassVar[int]
    REVERSIBLE_FIELD_NUMBER: _ClassVar[int]
    URGENCY_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_URL_FIELD_NUMBER: _ClassVar[int]
    DEADLINE_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    ANSWER_FIELD_NUMBER: _ClassVar[int]
    ANSWER_LABEL_FIELD_NUMBER: _ClassVar[int]
    ANSWER_NOTE_FIELD_NUMBER: _ClassVar[int]
    ANSWERED_VIA_FIELD_NUMBER: _ClassVar[int]
    LATE_FIELD_NUMBER: _ClassVar[int]
    ANSWERED_AT_FIELD_NUMBER: _ClassVar[int]
    RESOLUTION_REASON_FIELD_NUMBER: _ClassVar[int]
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    RAISED_BY_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    PUBLISHED_AT_FIELD_NUMBER: _ClassVar[int]
    PUBLISH_ERROR_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    BIG_PICTURE_FIELD_NUMBER: _ClassVar[int]
    TECHNICAL_DETAILS_FIELD_NUMBER: _ClassVar[int]
    effort_ref: str
    goal_home: str
    decision_id: str
    question: str
    options: _containers.RepeatedCompositeFieldContainer[EffortDecisionOption]
    recommended: str
    recommendation_reason: str
    default_answer: str
    reversible: bool
    urgency: str
    note: str
    context_url: str
    deadline: _timestamp_pb2.Timestamp
    state: str
    answer: str
    answer_label: str
    answer_note: str
    answered_via: str
    late: bool
    answered_at: _timestamp_pb2.Timestamp
    resolution_reason: str
    event_id: str
    ask_id: str
    raised_by_run_id: str
    created_at: _timestamp_pb2.Timestamp
    published_at: _timestamp_pb2.Timestamp
    publish_error: str
    recorded_at: _timestamp_pb2.Timestamp
    big_picture: str
    technical_details: str
    def __init__(self, effort_ref: _Optional[str] = ..., goal_home: _Optional[str] = ..., decision_id: _Optional[str] = ..., question: _Optional[str] = ..., options: _Optional[_Iterable[_Union[EffortDecisionOption, _Mapping]]] = ..., recommended: _Optional[str] = ..., recommendation_reason: _Optional[str] = ..., default_answer: _Optional[str] = ..., reversible: _Optional[bool] = ..., urgency: _Optional[str] = ..., note: _Optional[str] = ..., context_url: _Optional[str] = ..., deadline: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., state: _Optional[str] = ..., answer: _Optional[str] = ..., answer_label: _Optional[str] = ..., answer_note: _Optional[str] = ..., answered_via: _Optional[str] = ..., late: _Optional[bool] = ..., answered_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., resolution_reason: _Optional[str] = ..., event_id: _Optional[str] = ..., ask_id: _Optional[str] = ..., raised_by_run_id: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., published_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., publish_error: _Optional[str] = ..., recorded_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., big_picture: _Optional[str] = ..., technical_details: _Optional[str] = ...) -> None: ...

class RaiseEffortDecisionRequest(_message.Message):
    __slots__ = ("decision",)
    DECISION_FIELD_NUMBER: _ClassVar[int]
    decision: EffortDecision
    def __init__(self, decision: _Optional[_Union[EffortDecision, _Mapping]] = ...) -> None: ...

class RaiseEffortDecisionResponse(_message.Message):
    __slots__ = ("decision", "created", "published")
    DECISION_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    PUBLISHED_FIELD_NUMBER: _ClassVar[int]
    decision: EffortDecision
    created: bool
    published: bool
    def __init__(self, decision: _Optional[_Union[EffortDecision, _Mapping]] = ..., created: _Optional[bool] = ..., published: _Optional[bool] = ...) -> None: ...

class ListEffortDecisionsRequest(_message.Message):
    __slots__ = ("goal_home", "effort_ref", "unrecorded_only", "limit", "page_token")
    GOAL_HOME_FIELD_NUMBER: _ClassVar[int]
    EFFORT_REF_FIELD_NUMBER: _ClassVar[int]
    UNRECORDED_ONLY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    goal_home: str
    effort_ref: str
    unrecorded_only: bool
    limit: int
    page_token: str
    def __init__(self, goal_home: _Optional[str] = ..., effort_ref: _Optional[str] = ..., unrecorded_only: _Optional[bool] = ..., limit: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListEffortDecisionsResponse(_message.Message):
    __slots__ = ("decisions", "next_page_token")
    DECISIONS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    decisions: _containers.RepeatedCompositeFieldContainer[EffortDecision]
    next_page_token: str
    def __init__(self, decisions: _Optional[_Iterable[_Union[EffortDecision, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class AnswerEffortDecisionRequest(_message.Message):
    __slots__ = ("goal_home", "decision_id", "answer", "note")
    GOAL_HOME_FIELD_NUMBER: _ClassVar[int]
    DECISION_ID_FIELD_NUMBER: _ClassVar[int]
    ANSWER_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    goal_home: str
    decision_id: str
    answer: str
    note: str
    def __init__(self, goal_home: _Optional[str] = ..., decision_id: _Optional[str] = ..., answer: _Optional[str] = ..., note: _Optional[str] = ...) -> None: ...

class MarkEffortDecisionRecordedRequest(_message.Message):
    __slots__ = ("goal_home", "decision_id", "answered_at")
    GOAL_HOME_FIELD_NUMBER: _ClassVar[int]
    DECISION_ID_FIELD_NUMBER: _ClassVar[int]
    ANSWERED_AT_FIELD_NUMBER: _ClassVar[int]
    goal_home: str
    decision_id: str
    answered_at: _timestamp_pb2.Timestamp
    def __init__(self, goal_home: _Optional[str] = ..., decision_id: _Optional[str] = ..., answered_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
