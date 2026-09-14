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

class EffortDirectiveDelivery(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EFFORT_DIRECTIVE_DELIVERY_UNSPECIFIED: _ClassVar[EffortDirectiveDelivery]
    EFFORT_DIRECTIVE_DELIVERY_PENDING: _ClassVar[EffortDirectiveDelivery]
    EFFORT_DIRECTIVE_DELIVERY_DELIVERED: _ClassVar[EffortDirectiveDelivery]
    EFFORT_DIRECTIVE_DELIVERY_REFUSED: _ClassVar[EffortDirectiveDelivery]
    EFFORT_DIRECTIVE_DELIVERY_EXPIRED: _ClassVar[EffortDirectiveDelivery]
    EFFORT_DIRECTIVE_DELIVERY_SUPERSEDED: _ClassVar[EffortDirectiveDelivery]
    EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN: _ClassVar[EffortDirectiveDelivery]

class EffortDirectiveAcknowledgment(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EFFORT_DIRECTIVE_ACKNOWLEDGMENT_UNSPECIFIED: _ClassVar[EffortDirectiveAcknowledgment]
    EFFORT_DIRECTIVE_ACKNOWLEDGMENT_ACCEPTED: _ClassVar[EffortDirectiveAcknowledgment]
    EFFORT_DIRECTIVE_ACKNOWLEDGMENT_DEFERRED: _ClassVar[EffortDirectiveAcknowledgment]
    EFFORT_DIRECTIVE_ACKNOWLEDGMENT_CHALLENGED: _ClassVar[EffortDirectiveAcknowledgment]
EFFORT_FRESHNESS_UNSPECIFIED: EffortFreshness
EFFORT_FRESHNESS_FRESH: EffortFreshness
EFFORT_FRESHNESS_STALE: EffortFreshness
EFFORT_FRESHNESS_UNAVAILABLE: EffortFreshness
EFFORT_DIRECTIVE_DELIVERY_UNSPECIFIED: EffortDirectiveDelivery
EFFORT_DIRECTIVE_DELIVERY_PENDING: EffortDirectiveDelivery
EFFORT_DIRECTIVE_DELIVERY_DELIVERED: EffortDirectiveDelivery
EFFORT_DIRECTIVE_DELIVERY_REFUSED: EffortDirectiveDelivery
EFFORT_DIRECTIVE_DELIVERY_EXPIRED: EffortDirectiveDelivery
EFFORT_DIRECTIVE_DELIVERY_SUPERSEDED: EffortDirectiveDelivery
EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN: EffortDirectiveDelivery
EFFORT_DIRECTIVE_ACKNOWLEDGMENT_UNSPECIFIED: EffortDirectiveAcknowledgment
EFFORT_DIRECTIVE_ACKNOWLEDGMENT_ACCEPTED: EffortDirectiveAcknowledgment
EFFORT_DIRECTIVE_ACKNOWLEDGMENT_DEFERRED: EffortDirectiveAcknowledgment
EFFORT_DIRECTIVE_ACKNOWLEDGMENT_CHALLENGED: EffortDirectiveAcknowledgment

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
    __slots__ = ("effort_ref", "display_name", "destination_ref", "target_revision", "source_revision", "authority_ref", "supervisor_run_id", "subjects", "permitted_actions", "revision", "withdrawn", "workspace", "work_shape", "updated_at", "authorized_by", "authority_expires_at", "maximum_directives", "cooldown_seconds", "withdrawal_reason", "supervisor_owner_subject", "supervisor_scope", "dispatch_authorization")
    EFFORT_REF_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_REF_FIELD_NUMBER: _ClassVar[int]
    TARGET_REVISION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REVISION_FIELD_NUMBER: _ClassVar[int]
    AUTHORITY_REF_FIELD_NUMBER: _ClassVar[int]
    SUPERVISOR_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SUBJECTS_FIELD_NUMBER: _ClassVar[int]
    PERMITTED_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    WITHDRAWN_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    WORK_SHAPE_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    AUTHORIZED_BY_FIELD_NUMBER: _ClassVar[int]
    AUTHORITY_EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    MAXIMUM_DIRECTIVES_FIELD_NUMBER: _ClassVar[int]
    COOLDOWN_SECONDS_FIELD_NUMBER: _ClassVar[int]
    WITHDRAWAL_REASON_FIELD_NUMBER: _ClassVar[int]
    SUPERVISOR_OWNER_SUBJECT_FIELD_NUMBER: _ClassVar[int]
    SUPERVISOR_SCOPE_FIELD_NUMBER: _ClassVar[int]
    DISPATCH_AUTHORIZATION_FIELD_NUMBER: _ClassVar[int]
    effort_ref: str
    display_name: str
    destination_ref: str
    target_revision: str
    source_revision: str
    authority_ref: str
    supervisor_run_id: str
    subjects: _containers.RepeatedCompositeFieldContainer[EffortSubject]
    permitted_actions: _containers.RepeatedScalarFieldContainer[_watch_pb2.WatchActionKind]
    revision: int
    withdrawn: bool
    workspace: str
    work_shape: str
    updated_at: _timestamp_pb2.Timestamp
    authorized_by: str
    authority_expires_at: _timestamp_pb2.Timestamp
    maximum_directives: int
    cooldown_seconds: int
    withdrawal_reason: str
    supervisor_owner_subject: str
    supervisor_scope: str
    dispatch_authorization: SupervisorDispatchAuthorization
    def __init__(self, effort_ref: _Optional[str] = ..., display_name: _Optional[str] = ..., destination_ref: _Optional[str] = ..., target_revision: _Optional[str] = ..., source_revision: _Optional[str] = ..., authority_ref: _Optional[str] = ..., supervisor_run_id: _Optional[str] = ..., subjects: _Optional[_Iterable[_Union[EffortSubject, _Mapping]]] = ..., permitted_actions: _Optional[_Iterable[_Union[_watch_pb2.WatchActionKind, str]]] = ..., revision: _Optional[int] = ..., withdrawn: _Optional[bool] = ..., workspace: _Optional[str] = ..., work_shape: _Optional[str] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., authorized_by: _Optional[str] = ..., authority_expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., maximum_directives: _Optional[int] = ..., cooldown_seconds: _Optional[int] = ..., withdrawal_reason: _Optional[str] = ..., supervisor_owner_subject: _Optional[str] = ..., supervisor_scope: _Optional[str] = ..., dispatch_authorization: _Optional[_Union[SupervisorDispatchAuthorization, _Mapping]] = ...) -> None: ...

class SupervisorDispatchAuthorization(_message.Message):
    __slots__ = ("authorization_id", "owner_subject", "team_id", "member_id", "profile_key", "scopes", "issued_at", "expires_at", "revoked_at", "credential_hash", "target_revision", "issuance_key", "maximum_runs", "dispatched_runs", "minimum_interval_seconds", "last_dispatched_at")
    AUTHORIZATION_ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_SUBJECT_FIELD_NUMBER: _ClassVar[int]
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    MEMBER_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_KEY_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    ISSUED_AT_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    REVOKED_AT_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_HASH_FIELD_NUMBER: _ClassVar[int]
    TARGET_REVISION_FIELD_NUMBER: _ClassVar[int]
    ISSUANCE_KEY_FIELD_NUMBER: _ClassVar[int]
    MAXIMUM_RUNS_FIELD_NUMBER: _ClassVar[int]
    DISPATCHED_RUNS_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_INTERVAL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    LAST_DISPATCHED_AT_FIELD_NUMBER: _ClassVar[int]
    authorization_id: str
    owner_subject: str
    team_id: str
    member_id: str
    profile_key: str
    scopes: _containers.RepeatedScalarFieldContainer[str]
    issued_at: _timestamp_pb2.Timestamp
    expires_at: _timestamp_pb2.Timestamp
    revoked_at: _timestamp_pb2.Timestamp
    credential_hash: str
    target_revision: str
    issuance_key: str
    maximum_runs: int
    dispatched_runs: int
    minimum_interval_seconds: int
    last_dispatched_at: _timestamp_pb2.Timestamp
    def __init__(self, authorization_id: _Optional[str] = ..., owner_subject: _Optional[str] = ..., team_id: _Optional[str] = ..., member_id: _Optional[str] = ..., profile_key: _Optional[str] = ..., scopes: _Optional[_Iterable[str]] = ..., issued_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., revoked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., credential_hash: _Optional[str] = ..., target_revision: _Optional[str] = ..., issuance_key: _Optional[str] = ..., maximum_runs: _Optional[int] = ..., dispatched_runs: _Optional[int] = ..., minimum_interval_seconds: _Optional[int] = ..., last_dispatched_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

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
    __slots__ = ("enrollment", "observed_at", "freshness", "runtime_state", "outcome_standing", "assignments", "pending_operations", "blockers", "next_action", "rationale", "usage", "directives", "limitations", "evidence_refs", "change_identity", "last_assessment", "visibility_change_identity", "quota_observations")
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
    DIRECTIVES_FIELD_NUMBER: _ClassVar[int]
    LIMITATIONS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    CHANGE_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    LAST_ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
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
    directives: _containers.RepeatedCompositeFieldContainer[EffortDirective]
    limitations: _containers.RepeatedScalarFieldContainer[str]
    evidence_refs: _containers.RepeatedScalarFieldContainer[str]
    change_identity: str
    last_assessment: EffortAssessment
    visibility_change_identity: str
    quota_observations: _containers.RepeatedCompositeFieldContainer[EffortQuotaObservation]
    def __init__(self, enrollment: _Optional[_Union[EffortEnrollment, _Mapping]] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., freshness: _Optional[_Union[EffortFreshness, str]] = ..., runtime_state: _Optional[str] = ..., outcome_standing: _Optional[_Union[EffortOutcomeStanding, _Mapping]] = ..., assignments: _Optional[_Iterable[_Union[EffortAssignment, _Mapping]]] = ..., pending_operations: _Optional[_Iterable[str]] = ..., blockers: _Optional[_Iterable[str]] = ..., next_action: _Optional[str] = ..., rationale: _Optional[str] = ..., usage: _Optional[_Union[EffortUsage, _Mapping]] = ..., directives: _Optional[_Iterable[_Union[EffortDirective, _Mapping]]] = ..., limitations: _Optional[_Iterable[str]] = ..., evidence_refs: _Optional[_Iterable[str]] = ..., change_identity: _Optional[str] = ..., last_assessment: _Optional[_Union[EffortAssessment, _Mapping]] = ..., visibility_change_identity: _Optional[str] = ..., quota_observations: _Optional[_Iterable[_Union[EffortQuotaObservation, _Mapping]]] = ...) -> None: ...

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

class EffortDirective(_message.Message):
    __slots__ = ("directive_id", "effort_ref", "target_revision", "issuer", "target_run_id", "kind", "scope", "evidence_refs", "adjustment", "expected_result", "expires_at", "idempotency_key", "delivery", "acknowledgment", "acknowledgment_reason", "owner_wait_ref", "action_ref", "assessment", "assessment_evidence_refs", "superseded_by", "revision", "created_at", "delivered_at", "delivery_reason", "source_snapshot", "supervision_usage", "hypothesis", "comparison", "recovery_expectation", "recovery_verification", "authority_binding", "refusal_before_effects", "recovered_run_id")
    DIRECTIVE_ID_FIELD_NUMBER: _ClassVar[int]
    EFFORT_REF_FIELD_NUMBER: _ClassVar[int]
    TARGET_REVISION_FIELD_NUMBER: _ClassVar[int]
    ISSUER_FIELD_NUMBER: _ClassVar[int]
    TARGET_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    ADJUSTMENT_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_RESULT_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    DELIVERY_FIELD_NUMBER: _ClassVar[int]
    ACKNOWLEDGMENT_FIELD_NUMBER: _ClassVar[int]
    ACKNOWLEDGMENT_REASON_FIELD_NUMBER: _ClassVar[int]
    OWNER_WAIT_REF_FIELD_NUMBER: _ClassVar[int]
    ACTION_REF_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENT_EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDED_BY_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    DELIVERED_AT_FIELD_NUMBER: _ClassVar[int]
    DELIVERY_REASON_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    SUPERVISION_USAGE_FIELD_NUMBER: _ClassVar[int]
    HYPOTHESIS_FIELD_NUMBER: _ClassVar[int]
    COMPARISON_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_EXPECTATION_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    AUTHORITY_BINDING_FIELD_NUMBER: _ClassVar[int]
    REFUSAL_BEFORE_EFFECTS_FIELD_NUMBER: _ClassVar[int]
    RECOVERED_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    directive_id: str
    effort_ref: str
    target_revision: str
    issuer: str
    target_run_id: str
    kind: _watch_pb2.WatchActionKind
    scope: str
    evidence_refs: _containers.RepeatedScalarFieldContainer[str]
    adjustment: str
    expected_result: str
    expires_at: _timestamp_pb2.Timestamp
    idempotency_key: str
    delivery: EffortDirectiveDelivery
    acknowledgment: EffortDirectiveAcknowledgment
    acknowledgment_reason: str
    owner_wait_ref: str
    action_ref: str
    assessment: str
    assessment_evidence_refs: _containers.RepeatedScalarFieldContainer[str]
    superseded_by: str
    revision: int
    created_at: _timestamp_pb2.Timestamp
    delivered_at: _timestamp_pb2.Timestamp
    delivery_reason: str
    source_snapshot: EffortBoardRow
    supervision_usage: EffortUsage
    hypothesis: str
    comparison: str
    recovery_expectation: EffortRecoveryExpectation
    recovery_verification: EffortRecoveryVerification
    authority_binding: EffortDirectiveAuthorityBinding
    refusal_before_effects: bool
    recovered_run_id: str
    def __init__(self, directive_id: _Optional[str] = ..., effort_ref: _Optional[str] = ..., target_revision: _Optional[str] = ..., issuer: _Optional[str] = ..., target_run_id: _Optional[str] = ..., kind: _Optional[_Union[_watch_pb2.WatchActionKind, str]] = ..., scope: _Optional[str] = ..., evidence_refs: _Optional[_Iterable[str]] = ..., adjustment: _Optional[str] = ..., expected_result: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., idempotency_key: _Optional[str] = ..., delivery: _Optional[_Union[EffortDirectiveDelivery, str]] = ..., acknowledgment: _Optional[_Union[EffortDirectiveAcknowledgment, str]] = ..., acknowledgment_reason: _Optional[str] = ..., owner_wait_ref: _Optional[str] = ..., action_ref: _Optional[str] = ..., assessment: _Optional[str] = ..., assessment_evidence_refs: _Optional[_Iterable[str]] = ..., superseded_by: _Optional[str] = ..., revision: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., delivered_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., delivery_reason: _Optional[str] = ..., source_snapshot: _Optional[_Union[EffortBoardRow, _Mapping]] = ..., supervision_usage: _Optional[_Union[EffortUsage, _Mapping]] = ..., hypothesis: _Optional[str] = ..., comparison: _Optional[str] = ..., recovery_expectation: _Optional[_Union[EffortRecoveryExpectation, _Mapping]] = ..., recovery_verification: _Optional[_Union[EffortRecoveryVerification, _Mapping]] = ..., authority_binding: _Optional[_Union[EffortDirectiveAuthorityBinding, _Mapping]] = ..., refusal_before_effects: _Optional[bool] = ..., recovered_run_id: _Optional[str] = ...) -> None: ...

class EffortDirectiveAuthorityBinding(_message.Message):
    __slots__ = ("mode", "subject", "scope")
    MODE_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    mode: str
    subject: str
    scope: str
    def __init__(self, mode: _Optional[str] = ..., subject: _Optional[str] = ..., scope: _Optional[str] = ...) -> None: ...

class EffortRecoveryExpectation(_message.Message):
    __slots__ = ("progress_condition", "baseline_evidence_refs")
    PROGRESS_CONDITION_FIELD_NUMBER: _ClassVar[int]
    BASELINE_EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    progress_condition: str
    baseline_evidence_refs: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, progress_condition: _Optional[str] = ..., baseline_evidence_refs: _Optional[_Iterable[str]] = ...) -> None: ...

class EffortRecoveryVerification(_message.Message):
    __slots__ = ("state", "reason", "evidence_refs", "observed_at", "next_owner_condition", "verifier")
    STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    NEXT_OWNER_CONDITION_FIELD_NUMBER: _ClassVar[int]
    VERIFIER_FIELD_NUMBER: _ClassVar[int]
    state: str
    reason: str
    evidence_refs: _containers.RepeatedScalarFieldContainer[str]
    observed_at: _timestamp_pb2.Timestamp
    next_owner_condition: str
    verifier: str
    def __init__(self, state: _Optional[str] = ..., reason: _Optional[str] = ..., evidence_refs: _Optional[_Iterable[str]] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., next_owner_condition: _Optional[str] = ..., verifier: _Optional[str] = ...) -> None: ...

class EffortRepairLink(_message.Message):
    __slots__ = ("work_ref", "assigning_owner_ref", "next_operation", "completion_evidence_refs", "stopping_condition", "state")
    WORK_REF_FIELD_NUMBER: _ClassVar[int]
    ASSIGNING_OWNER_REF_FIELD_NUMBER: _ClassVar[int]
    NEXT_OPERATION_FIELD_NUMBER: _ClassVar[int]
    COMPLETION_EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    STOPPING_CONDITION_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    work_ref: str
    assigning_owner_ref: str
    next_operation: str
    completion_evidence_refs: _containers.RepeatedScalarFieldContainer[str]
    stopping_condition: str
    state: str
    def __init__(self, work_ref: _Optional[str] = ..., assigning_owner_ref: _Optional[str] = ..., next_operation: _Optional[str] = ..., completion_evidence_refs: _Optional[_Iterable[str]] = ..., stopping_condition: _Optional[str] = ..., state: _Optional[str] = ...) -> None: ...

class RequestEffortDirectiveRequest(_message.Message):
    __slots__ = ("directive", "expected_enrollment_revision", "authority")
    DIRECTIVE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_ENROLLMENT_REVISION_FIELD_NUMBER: _ClassVar[int]
    AUTHORITY_FIELD_NUMBER: _ClassVar[int]
    directive: EffortDirective
    expected_enrollment_revision: int
    authority: _watch_pb2.WatchAuthority
    def __init__(self, directive: _Optional[_Union[EffortDirective, _Mapping]] = ..., expected_enrollment_revision: _Optional[int] = ..., authority: _Optional[_Union[_watch_pb2.WatchAuthority, str]] = ...) -> None: ...

class ListEffortDirectivesRequest(_message.Message):
    __slots__ = ("effort_ref", "page_size", "page_token")
    EFFORT_REF_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    effort_ref: str
    page_size: int
    page_token: str
    def __init__(self, effort_ref: _Optional[str] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListEffortDirectivesResponse(_message.Message):
    __slots__ = ("directives", "next_page_token")
    DIRECTIVES_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    directives: _containers.RepeatedCompositeFieldContainer[EffortDirective]
    next_page_token: str
    def __init__(self, directives: _Optional[_Iterable[_Union[EffortDirective, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class UpdateEffortDirectiveRequest(_message.Message):
    __slots__ = ("directive_id", "expected_revision", "idempotency_key", "authority", "acknowledgment", "reason", "owner_wait_ref", "action_ref", "assessment", "evidence_refs", "superseded_by", "supervision_usage", "recovery_verification", "reconcile_delivery")
    DIRECTIVE_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    AUTHORITY_FIELD_NUMBER: _ClassVar[int]
    ACKNOWLEDGMENT_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    OWNER_WAIT_REF_FIELD_NUMBER: _ClassVar[int]
    ACTION_REF_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDED_BY_FIELD_NUMBER: _ClassVar[int]
    SUPERVISION_USAGE_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    RECONCILE_DELIVERY_FIELD_NUMBER: _ClassVar[int]
    directive_id: str
    expected_revision: int
    idempotency_key: str
    authority: _watch_pb2.WatchAuthority
    acknowledgment: EffortDirectiveAcknowledgment
    reason: str
    owner_wait_ref: str
    action_ref: str
    assessment: str
    evidence_refs: _containers.RepeatedScalarFieldContainer[str]
    superseded_by: str
    supervision_usage: EffortUsage
    recovery_verification: EffortRecoveryVerification
    reconcile_delivery: bool
    def __init__(self, directive_id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., idempotency_key: _Optional[str] = ..., authority: _Optional[_Union[_watch_pb2.WatchAuthority, str]] = ..., acknowledgment: _Optional[_Union[EffortDirectiveAcknowledgment, str]] = ..., reason: _Optional[str] = ..., owner_wait_ref: _Optional[str] = ..., action_ref: _Optional[str] = ..., assessment: _Optional[str] = ..., evidence_refs: _Optional[_Iterable[str]] = ..., superseded_by: _Optional[str] = ..., supervision_usage: _Optional[_Union[EffortUsage, _Mapping]] = ..., recovery_verification: _Optional[_Union[EffortRecoveryVerification, _Mapping]] = ..., reconcile_delivery: _Optional[bool] = ...) -> None: ...

class EffortAssessment(_message.Message):
    __slots__ = ("assessment_id", "effort_refs", "target_revisions", "supervisor_run_id", "disposition", "rationale", "evidence_refs", "source_ledger_ref", "shared_operation_ref", "allowance_ref", "allocation_rule", "observed_usage", "unallocated_usage", "observed_at", "idempotency_key", "hypothesis", "comparison", "benefit", "policy_version", "limitations", "repair_links")
    class TargetRevisionsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ASSESSMENT_ID_FIELD_NUMBER: _ClassVar[int]
    EFFORT_REFS_FIELD_NUMBER: _ClassVar[int]
    TARGET_REVISIONS_FIELD_NUMBER: _ClassVar[int]
    SUPERVISOR_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_LEDGER_REF_FIELD_NUMBER: _ClassVar[int]
    SHARED_OPERATION_REF_FIELD_NUMBER: _ClassVar[int]
    ALLOWANCE_REF_FIELD_NUMBER: _ClassVar[int]
    ALLOCATION_RULE_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_USAGE_FIELD_NUMBER: _ClassVar[int]
    UNALLOCATED_USAGE_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    HYPOTHESIS_FIELD_NUMBER: _ClassVar[int]
    COMPARISON_FIELD_NUMBER: _ClassVar[int]
    BENEFIT_FIELD_NUMBER: _ClassVar[int]
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    LIMITATIONS_FIELD_NUMBER: _ClassVar[int]
    REPAIR_LINKS_FIELD_NUMBER: _ClassVar[int]
    assessment_id: str
    effort_refs: _containers.RepeatedScalarFieldContainer[str]
    target_revisions: _containers.ScalarMap[str, str]
    supervisor_run_id: str
    disposition: str
    rationale: str
    evidence_refs: _containers.RepeatedScalarFieldContainer[str]
    source_ledger_ref: str
    shared_operation_ref: str
    allowance_ref: str
    allocation_rule: str
    observed_usage: EffortUsage
    unallocated_usage: EffortUsage
    observed_at: _timestamp_pb2.Timestamp
    idempotency_key: str
    hypothesis: str
    comparison: str
    benefit: str
    policy_version: str
    limitations: _containers.RepeatedScalarFieldContainer[str]
    repair_links: _containers.RepeatedCompositeFieldContainer[EffortRepairLink]
    def __init__(self, assessment_id: _Optional[str] = ..., effort_refs: _Optional[_Iterable[str]] = ..., target_revisions: _Optional[_Mapping[str, str]] = ..., supervisor_run_id: _Optional[str] = ..., disposition: _Optional[str] = ..., rationale: _Optional[str] = ..., evidence_refs: _Optional[_Iterable[str]] = ..., source_ledger_ref: _Optional[str] = ..., shared_operation_ref: _Optional[str] = ..., allowance_ref: _Optional[str] = ..., allocation_rule: _Optional[str] = ..., observed_usage: _Optional[_Union[EffortUsage, _Mapping]] = ..., unallocated_usage: _Optional[_Union[EffortUsage, _Mapping]] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., idempotency_key: _Optional[str] = ..., hypothesis: _Optional[str] = ..., comparison: _Optional[str] = ..., benefit: _Optional[str] = ..., policy_version: _Optional[str] = ..., limitations: _Optional[_Iterable[str]] = ..., repair_links: _Optional[_Iterable[_Union[EffortRepairLink, _Mapping]]] = ...) -> None: ...

class RecordEffortAssessmentRequest(_message.Message):
    __slots__ = ("assessment", "authority")
    ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    AUTHORITY_FIELD_NUMBER: _ClassVar[int]
    assessment: EffortAssessment
    authority: _watch_pb2.WatchAuthority
    def __init__(self, assessment: _Optional[_Union[EffortAssessment, _Mapping]] = ..., authority: _Optional[_Union[_watch_pb2.WatchAuthority, str]] = ...) -> None: ...
