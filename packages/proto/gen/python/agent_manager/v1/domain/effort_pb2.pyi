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
    __slots__ = ("effort_ref", "display_name", "destination_ref", "target_revision", "source_revision", "authority_ref", "supervisor_run_id", "subjects", "revision", "withdrawn", "workspace", "work_shape", "updated_at", "authorized_by", "withdrawal_reason")
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
    def __init__(self, effort_ref: _Optional[str] = ..., display_name: _Optional[str] = ..., destination_ref: _Optional[str] = ..., target_revision: _Optional[str] = ..., source_revision: _Optional[str] = ..., authority_ref: _Optional[str] = ..., supervisor_run_id: _Optional[str] = ..., subjects: _Optional[_Iterable[_Union[EffortSubject, _Mapping]]] = ..., revision: _Optional[int] = ..., withdrawn: _Optional[bool] = ..., workspace: _Optional[str] = ..., work_shape: _Optional[str] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., authorized_by: _Optional[str] = ..., withdrawal_reason: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("enrollment", "observed_at", "freshness", "runtime_state", "outcome_standing", "assignments", "pending_operations", "blockers", "next_action", "rationale", "usage", "limitations", "evidence_refs", "change_identity", "visibility_change_identity", "quota_observations")
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
    def __init__(self, enrollment: _Optional[_Union[EffortEnrollment, _Mapping]] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., freshness: _Optional[_Union[EffortFreshness, str]] = ..., runtime_state: _Optional[str] = ..., outcome_standing: _Optional[_Union[EffortOutcomeStanding, _Mapping]] = ..., assignments: _Optional[_Iterable[_Union[EffortAssignment, _Mapping]]] = ..., pending_operations: _Optional[_Iterable[str]] = ..., blockers: _Optional[_Iterable[str]] = ..., next_action: _Optional[str] = ..., rationale: _Optional[str] = ..., usage: _Optional[_Union[EffortUsage, _Mapping]] = ..., limitations: _Optional[_Iterable[str]] = ..., evidence_refs: _Optional[_Iterable[str]] = ..., change_identity: _Optional[str] = ..., visibility_change_identity: _Optional[str] = ..., quota_observations: _Optional[_Iterable[_Union[EffortQuotaObservation, _Mapping]]] = ...) -> None: ...

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
