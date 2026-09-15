from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetPolicyRequest(_message.Message):
    __slots__ = ("family_id", "execution_id", "phase_id")
    FAMILY_ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    family_id: str
    execution_id: str
    phase_id: str
    def __init__(self, family_id: _Optional[str] = ..., execution_id: _Optional[str] = ..., phase_id: _Optional[str] = ...) -> None: ...

class PutPolicyRequest(_message.Message):
    __slots__ = ("policy_json", "active", "expected_version")
    POLICY_JSON_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_VERSION_FIELD_NUMBER: _ClassVar[int]
    policy_json: str
    active: bool
    expected_version: str
    def __init__(self, policy_json: _Optional[str] = ..., active: _Optional[bool] = ..., expected_version: _Optional[str] = ...) -> None: ...

class PolicyRecord(_message.Message):
    __slots__ = ("schema_version", "version", "mode", "policy_json", "active", "digest", "scope_json", "source_scope")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    POLICY_JSON_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    SCOPE_JSON_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SCOPE_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    version: str
    mode: str
    policy_json: str
    active: bool
    digest: str
    scope_json: str
    source_scope: str
    def __init__(self, schema_version: _Optional[str] = ..., version: _Optional[str] = ..., mode: _Optional[str] = ..., policy_json: _Optional[str] = ..., active: _Optional[bool] = ..., digest: _Optional[str] = ..., scope_json: _Optional[str] = ..., source_scope: _Optional[str] = ...) -> None: ...

class PreviewTriggerRequest(_message.Message):
    __slots__ = ("observation_json",)
    OBSERVATION_JSON_FIELD_NUMBER: _ClassVar[int]
    observation_json: str
    def __init__(self, observation_json: _Optional[str] = ...) -> None: ...

class TriggerMatch(_message.Message):
    __slots__ = ("kind", "detail")
    KIND_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    kind: str
    detail: str
    def __init__(self, kind: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class TriggerDecision(_message.Message):
    __slots__ = ("eligible", "mode", "matches", "reasons", "incident_fingerprint", "queued")
    ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    MATCHES_FIELD_NUMBER: _ClassVar[int]
    REASONS_FIELD_NUMBER: _ClassVar[int]
    INCIDENT_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    QUEUED_FIELD_NUMBER: _ClassVar[int]
    eligible: bool
    mode: str
    matches: _containers.RepeatedCompositeFieldContainer[TriggerMatch]
    reasons: _containers.RepeatedScalarFieldContainer[str]
    incident_fingerprint: str
    queued: bool
    def __init__(self, eligible: _Optional[bool] = ..., mode: _Optional[str] = ..., matches: _Optional[_Iterable[_Union[TriggerMatch, _Mapping]]] = ..., reasons: _Optional[_Iterable[str]] = ..., incident_fingerprint: _Optional[str] = ..., queued: _Optional[bool] = ...) -> None: ...

class RecordTriggerRequest(_message.Message):
    __slots__ = ("observation_json",)
    OBSERVATION_JSON_FIELD_NUMBER: _ClassVar[int]
    observation_json: str
    def __init__(self, observation_json: _Optional[str] = ...) -> None: ...

class TriggerRecord(_message.Message):
    __slots__ = ("decision", "incident_json", "reused", "occurrence_id")
    DECISION_FIELD_NUMBER: _ClassVar[int]
    INCIDENT_JSON_FIELD_NUMBER: _ClassVar[int]
    REUSED_FIELD_NUMBER: _ClassVar[int]
    OCCURRENCE_ID_FIELD_NUMBER: _ClassVar[int]
    decision: TriggerDecision
    incident_json: str
    reused: bool
    occurrence_id: str
    def __init__(self, decision: _Optional[_Union[TriggerDecision, _Mapping]] = ..., incident_json: _Optional[str] = ..., reused: _Optional[bool] = ..., occurrence_id: _Optional[str] = ...) -> None: ...

class LinkTriggerRequest(_message.Message):
    __slots__ = ("incident_fingerprint", "investigation_id")
    INCIDENT_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    INVESTIGATION_ID_FIELD_NUMBER: _ClassVar[int]
    incident_fingerprint: str
    investigation_id: str
    def __init__(self, incident_fingerprint: _Optional[str] = ..., investigation_id: _Optional[str] = ...) -> None: ...

class ListIncidentsRequest(_message.Message):
    __slots__ = ("execution_id", "state", "limit", "family_id")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    FAMILY_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    state: str
    limit: int
    family_id: str
    def __init__(self, execution_id: _Optional[str] = ..., state: _Optional[str] = ..., limit: _Optional[int] = ..., family_id: _Optional[str] = ...) -> None: ...

class GetIncidentRequest(_message.Message):
    __slots__ = ("incident_fingerprint",)
    INCIDENT_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    incident_fingerprint: str
    def __init__(self, incident_fingerprint: _Optional[str] = ...) -> None: ...

class ListOccurrencesRequest(_message.Message):
    __slots__ = ("execution_id", "eligible_only", "limit")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    ELIGIBLE_ONLY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    eligible_only: bool
    limit: int
    def __init__(self, execution_id: _Optional[str] = ..., eligible_only: _Optional[bool] = ..., limit: _Optional[int] = ...) -> None: ...

class GetBriefRequest(_message.Message):
    __slots__ = ("execution_id", "phase_id", "run_ids", "validation_operation_id")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_IDS_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    phase_id: str
    run_ids: _containers.RepeatedScalarFieldContainer[str]
    validation_operation_id: str
    def __init__(self, execution_id: _Optional[str] = ..., phase_id: _Optional[str] = ..., run_ids: _Optional[_Iterable[str]] = ..., validation_operation_id: _Optional[str] = ...) -> None: ...

class BriefProgressMarker(_message.Message):
    __slots__ = ("kind", "evidence_ref", "detail", "observed_at")
    KIND_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REF_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    kind: str
    evidence_ref: str
    detail: str
    observed_at: str
    def __init__(self, kind: _Optional[str] = ..., evidence_ref: _Optional[str] = ..., detail: _Optional[str] = ..., observed_at: _Optional[str] = ...) -> None: ...

class BriefProducerWait(_message.Message):
    __slots__ = ("operation_id", "status", "queue_reason", "wait_argv", "sync_argv", "explicit")
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    QUEUE_REASON_FIELD_NUMBER: _ClassVar[int]
    WAIT_ARGV_FIELD_NUMBER: _ClassVar[int]
    SYNC_ARGV_FIELD_NUMBER: _ClassVar[int]
    EXPLICIT_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    status: str
    queue_reason: str
    wait_argv: _containers.RepeatedScalarFieldContainer[str]
    sync_argv: _containers.RepeatedScalarFieldContainer[str]
    explicit: bool
    def __init__(self, operation_id: _Optional[str] = ..., status: _Optional[str] = ..., queue_reason: _Optional[str] = ..., wait_argv: _Optional[_Iterable[str]] = ..., sync_argv: _Optional[_Iterable[str]] = ..., explicit: _Optional[bool] = ...) -> None: ...

class BriefBudget(_message.Message):
    __slots__ = ("known", "basis", "queue_seconds", "execution_seconds", "transport_wait_seconds")
    KNOWN_FIELD_NUMBER: _ClassVar[int]
    BASIS_FIELD_NUMBER: _ClassVar[int]
    QUEUE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    TRANSPORT_WAIT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    known: bool
    basis: str
    queue_seconds: int
    execution_seconds: int
    transport_wait_seconds: int
    def __init__(self, known: _Optional[bool] = ..., basis: _Optional[str] = ..., queue_seconds: _Optional[int] = ..., execution_seconds: _Optional[int] = ..., transport_wait_seconds: _Optional[int] = ...) -> None: ...

class PlanInvestigationBrief(_message.Message):
    __slots__ = ("schema_version", "status", "execution_id", "plan_id", "plan_revision", "phase_id", "phase_generation", "run_ids", "expected_outcome", "acceptance_criteria", "material_progress", "producer_wait", "wall_time_seconds", "active_work_seconds", "known_wait_seconds", "budget", "omitted_reasons")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_REVISION_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_GENERATION_FIELD_NUMBER: _ClassVar[int]
    RUN_IDS_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_OUTCOME_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_CRITERIA_FIELD_NUMBER: _ClassVar[int]
    MATERIAL_PROGRESS_FIELD_NUMBER: _ClassVar[int]
    PRODUCER_WAIT_FIELD_NUMBER: _ClassVar[int]
    WALL_TIME_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_WORK_SECONDS_FIELD_NUMBER: _ClassVar[int]
    KNOWN_WAIT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    BUDGET_FIELD_NUMBER: _ClassVar[int]
    OMITTED_REASONS_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    status: str
    execution_id: str
    plan_id: str
    plan_revision: str
    phase_id: str
    phase_generation: int
    run_ids: _containers.RepeatedScalarFieldContainer[str]
    expected_outcome: str
    acceptance_criteria: str
    material_progress: _containers.RepeatedCompositeFieldContainer[BriefProgressMarker]
    producer_wait: BriefProducerWait
    wall_time_seconds: int
    active_work_seconds: int
    known_wait_seconds: int
    budget: BriefBudget
    omitted_reasons: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, schema_version: _Optional[str] = ..., status: _Optional[str] = ..., execution_id: _Optional[str] = ..., plan_id: _Optional[str] = ..., plan_revision: _Optional[str] = ..., phase_id: _Optional[str] = ..., phase_generation: _Optional[int] = ..., run_ids: _Optional[_Iterable[str]] = ..., expected_outcome: _Optional[str] = ..., acceptance_criteria: _Optional[str] = ..., material_progress: _Optional[_Iterable[_Union[BriefProgressMarker, _Mapping]]] = ..., producer_wait: _Optional[_Union[BriefProducerWait, _Mapping]] = ..., wall_time_seconds: _Optional[int] = ..., active_work_seconds: _Optional[int] = ..., known_wait_seconds: _Optional[int] = ..., budget: _Optional[_Union[BriefBudget, _Mapping]] = ..., omitted_reasons: _Optional[_Iterable[str]] = ...) -> None: ...

class IncidentRecord(_message.Message):
    __slots__ = ("incident_id", "execution_id", "phase_id", "phase_generation", "policy_version", "incident_fingerprint", "mode", "state", "decision", "investigation_id", "program_id", "program_status", "dispatch_error", "created_at", "updated_at", "occurrence_count", "occurrences", "family_id", "subject_execution_ids")
    INCIDENT_ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_GENERATION_FIELD_NUMBER: _ClassVar[int]
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    INCIDENT_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    DECISION_FIELD_NUMBER: _ClassVar[int]
    INVESTIGATION_ID_FIELD_NUMBER: _ClassVar[int]
    PROGRAM_ID_FIELD_NUMBER: _ClassVar[int]
    PROGRAM_STATUS_FIELD_NUMBER: _ClassVar[int]
    DISPATCH_ERROR_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    OCCURRENCE_COUNT_FIELD_NUMBER: _ClassVar[int]
    OCCURRENCES_FIELD_NUMBER: _ClassVar[int]
    FAMILY_ID_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_EXECUTION_IDS_FIELD_NUMBER: _ClassVar[int]
    incident_id: str
    execution_id: str
    phase_id: str
    phase_generation: str
    policy_version: str
    incident_fingerprint: str
    mode: str
    state: str
    decision: TriggerDecision
    investigation_id: str
    program_id: str
    program_status: str
    dispatch_error: str
    created_at: str
    updated_at: str
    occurrence_count: int
    occurrences: _containers.RepeatedCompositeFieldContainer[IncidentOccurrence]
    family_id: str
    subject_execution_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, incident_id: _Optional[str] = ..., execution_id: _Optional[str] = ..., phase_id: _Optional[str] = ..., phase_generation: _Optional[str] = ..., policy_version: _Optional[str] = ..., incident_fingerprint: _Optional[str] = ..., mode: _Optional[str] = ..., state: _Optional[str] = ..., decision: _Optional[_Union[TriggerDecision, _Mapping]] = ..., investigation_id: _Optional[str] = ..., program_id: _Optional[str] = ..., program_status: _Optional[str] = ..., dispatch_error: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., occurrence_count: _Optional[int] = ..., occurrences: _Optional[_Iterable[_Union[IncidentOccurrence, _Mapping]]] = ..., family_id: _Optional[str] = ..., subject_execution_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class IncidentOccurrence(_message.Message):
    __slots__ = ("occurrence_id", "incident_fingerprint", "decision", "observed_at", "created_at", "execution_id", "phase_id", "phase_generation", "policy_version", "family_id", "shared_failure_ref")
    OCCURRENCE_ID_FIELD_NUMBER: _ClassVar[int]
    INCIDENT_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    DECISION_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_GENERATION_FIELD_NUMBER: _ClassVar[int]
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    FAMILY_ID_FIELD_NUMBER: _ClassVar[int]
    SHARED_FAILURE_REF_FIELD_NUMBER: _ClassVar[int]
    occurrence_id: str
    incident_fingerprint: str
    decision: TriggerDecision
    observed_at: str
    created_at: str
    execution_id: str
    phase_id: str
    phase_generation: str
    policy_version: str
    family_id: str
    shared_failure_ref: str
    def __init__(self, occurrence_id: _Optional[str] = ..., incident_fingerprint: _Optional[str] = ..., decision: _Optional[_Union[TriggerDecision, _Mapping]] = ..., observed_at: _Optional[str] = ..., created_at: _Optional[str] = ..., execution_id: _Optional[str] = ..., phase_id: _Optional[str] = ..., phase_generation: _Optional[str] = ..., policy_version: _Optional[str] = ..., family_id: _Optional[str] = ..., shared_failure_ref: _Optional[str] = ...) -> None: ...

class ListIncidentsResponse(_message.Message):
    __slots__ = ("incidents",)
    INCIDENTS_FIELD_NUMBER: _ClassVar[int]
    incidents: _containers.RepeatedCompositeFieldContainer[IncidentRecord]
    def __init__(self, incidents: _Optional[_Iterable[_Union[IncidentRecord, _Mapping]]] = ...) -> None: ...

class ListOccurrencesResponse(_message.Message):
    __slots__ = ("occurrences",)
    OCCURRENCES_FIELD_NUMBER: _ClassVar[int]
    occurrences: _containers.RepeatedCompositeFieldContainer[IncidentOccurrence]
    def __init__(self, occurrences: _Optional[_Iterable[_Union[IncidentOccurrence, _Mapping]]] = ...) -> None: ...
