import datetime

from data_backup_manager.v1.sources import sources_pb2 as _sources_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RequestSensitiveReviewRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RequestSensitiveReviewResponse(_message.Message):
    __slots__ = ("ask_id", "candidate_count", "pending")
    ASK_ID_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_COUNT_FIELD_NUMBER: _ClassVar[int]
    PENDING_FIELD_NUMBER: _ClassVar[int]
    ask_id: str
    candidate_count: int
    pending: bool
    def __init__(self, ask_id: _Optional[str] = ..., candidate_count: _Optional[int] = ..., pending: _Optional[bool] = ...) -> None: ...

class SensitivePolicyAddition(_message.Message):
    __slots__ = ("owner", "name", "group", "approx_bytes", "added_at")
    OWNER_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    GROUP_FIELD_NUMBER: _ClassVar[int]
    APPROX_BYTES_FIELD_NUMBER: _ClassVar[int]
    ADDED_AT_FIELD_NUMBER: _ClassVar[int]
    owner: str
    name: str
    group: str
    approx_bytes: int
    added_at: _timestamp_pb2.Timestamp
    def __init__(self, owner: _Optional[str] = ..., name: _Optional[str] = ..., group: _Optional[str] = ..., approx_bytes: _Optional[int] = ..., added_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class SensitivePolicyExclusion(_message.Message):
    __slots__ = ("owner", "name", "reason", "excluded_by", "excluded_at")
    OWNER_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EXCLUDED_BY_FIELD_NUMBER: _ClassVar[int]
    EXCLUDED_AT_FIELD_NUMBER: _ClassVar[int]
    owner: str
    name: str
    reason: str
    excluded_by: str
    excluded_at: _timestamp_pb2.Timestamp
    def __init__(self, owner: _Optional[str] = ..., name: _Optional[str] = ..., reason: _Optional[str] = ..., excluded_by: _Optional[str] = ..., excluded_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class SensitivePolicyEvent(_message.Message):
    __slots__ = ("at", "kind", "by", "detail")
    AT_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    BY_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    at: _timestamp_pb2.Timestamp
    kind: str
    by: str
    detail: str
    def __init__(self, at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., kind: _Optional[str] = ..., by: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class SensitivePolicy(_message.Message):
    __slots__ = ("active", "grant_ask_id", "granted_by", "granted_by_identity_kind", "granted_by_identity_source", "granted_at", "revoked_by", "revoked_at", "excluded", "unreported", "added_total", "last_note_at", "last_sweep_at", "history", "grant_basis", "grant_text")
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    GRANT_ASK_ID_FIELD_NUMBER: _ClassVar[int]
    GRANTED_BY_FIELD_NUMBER: _ClassVar[int]
    GRANTED_BY_IDENTITY_KIND_FIELD_NUMBER: _ClassVar[int]
    GRANTED_BY_IDENTITY_SOURCE_FIELD_NUMBER: _ClassVar[int]
    GRANTED_AT_FIELD_NUMBER: _ClassVar[int]
    REVOKED_BY_FIELD_NUMBER: _ClassVar[int]
    REVOKED_AT_FIELD_NUMBER: _ClassVar[int]
    EXCLUDED_FIELD_NUMBER: _ClassVar[int]
    UNREPORTED_FIELD_NUMBER: _ClassVar[int]
    ADDED_TOTAL_FIELD_NUMBER: _ClassVar[int]
    LAST_NOTE_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_SWEEP_AT_FIELD_NUMBER: _ClassVar[int]
    HISTORY_FIELD_NUMBER: _ClassVar[int]
    GRANT_BASIS_FIELD_NUMBER: _ClassVar[int]
    GRANT_TEXT_FIELD_NUMBER: _ClassVar[int]
    active: bool
    grant_ask_id: str
    granted_by: str
    granted_by_identity_kind: str
    granted_by_identity_source: str
    granted_at: _timestamp_pb2.Timestamp
    revoked_by: str
    revoked_at: _timestamp_pb2.Timestamp
    excluded: _containers.RepeatedCompositeFieldContainer[SensitivePolicyExclusion]
    unreported: _containers.RepeatedCompositeFieldContainer[SensitivePolicyAddition]
    added_total: int
    last_note_at: _timestamp_pb2.Timestamp
    last_sweep_at: _timestamp_pb2.Timestamp
    history: _containers.RepeatedCompositeFieldContainer[SensitivePolicyEvent]
    grant_basis: str
    grant_text: str
    def __init__(self, active: _Optional[bool] = ..., grant_ask_id: _Optional[str] = ..., granted_by: _Optional[str] = ..., granted_by_identity_kind: _Optional[str] = ..., granted_by_identity_source: _Optional[str] = ..., granted_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., revoked_by: _Optional[str] = ..., revoked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., excluded: _Optional[_Iterable[_Union[SensitivePolicyExclusion, _Mapping]]] = ..., unreported: _Optional[_Iterable[_Union[SensitivePolicyAddition, _Mapping]]] = ..., added_total: _Optional[int] = ..., last_note_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_sweep_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., history: _Optional[_Iterable[_Union[SensitivePolicyEvent, _Mapping]]] = ..., grant_basis: _Optional[str] = ..., grant_text: _Optional[str] = ...) -> None: ...

class GetSensitivePolicyRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetSensitivePolicyResponse(_message.Message):
    __slots__ = ("policy",)
    POLICY_FIELD_NUMBER: _ClassVar[int]
    policy: SensitivePolicy
    def __init__(self, policy: _Optional[_Union[SensitivePolicy, _Mapping]] = ...) -> None: ...

class RevokeSensitivePolicyRequest(_message.Message):
    __slots__ = ("revoked_by",)
    REVOKED_BY_FIELD_NUMBER: _ClassVar[int]
    revoked_by: str
    def __init__(self, revoked_by: _Optional[str] = ...) -> None: ...

class RevokeSensitivePolicyResponse(_message.Message):
    __slots__ = ("policy",)
    POLICY_FIELD_NUMBER: _ClassVar[int]
    policy: SensitivePolicy
    def __init__(self, policy: _Optional[_Union[SensitivePolicy, _Mapping]] = ...) -> None: ...

class ExcludeSensitiveTargetRequest(_message.Message):
    __slots__ = ("owner", "name", "reason", "excluded_by")
    OWNER_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EXCLUDED_BY_FIELD_NUMBER: _ClassVar[int]
    owner: str
    name: str
    reason: str
    excluded_by: str
    def __init__(self, owner: _Optional[str] = ..., name: _Optional[str] = ..., reason: _Optional[str] = ..., excluded_by: _Optional[str] = ...) -> None: ...

class ExcludeSensitiveTargetResponse(_message.Message):
    __slots__ = ("policy",)
    POLICY_FIELD_NUMBER: _ClassVar[int]
    policy: SensitivePolicy
    def __init__(self, policy: _Optional[_Union[SensitivePolicy, _Mapping]] = ...) -> None: ...

class RegisteredTarget(_message.Message):
    __slots__ = ("id", "owner", "name", "source_kind", "locator", "planned", "last_success_at", "last_verified_at", "critical", "reacquirable", "reacquisition_command", "reacquisition_evidence", "reacquisition_needs_operator", "reacquisition_auth_reason", "destinations", "all_destinations_backed_up", "all_destinations_verified", "physical_independence_proven", "independence_warnings")
    ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SOURCE_KIND_FIELD_NUMBER: _ClassVar[int]
    LOCATOR_FIELD_NUMBER: _ClassVar[int]
    PLANNED_FIELD_NUMBER: _ClassVar[int]
    LAST_SUCCESS_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_VERIFIED_AT_FIELD_NUMBER: _ClassVar[int]
    CRITICAL_FIELD_NUMBER: _ClassVar[int]
    REACQUIRABLE_FIELD_NUMBER: _ClassVar[int]
    REACQUISITION_COMMAND_FIELD_NUMBER: _ClassVar[int]
    REACQUISITION_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    REACQUISITION_NEEDS_OPERATOR_FIELD_NUMBER: _ClassVar[int]
    REACQUISITION_AUTH_REASON_FIELD_NUMBER: _ClassVar[int]
    DESTINATIONS_FIELD_NUMBER: _ClassVar[int]
    ALL_DESTINATIONS_BACKED_UP_FIELD_NUMBER: _ClassVar[int]
    ALL_DESTINATIONS_VERIFIED_FIELD_NUMBER: _ClassVar[int]
    PHYSICAL_INDEPENDENCE_PROVEN_FIELD_NUMBER: _ClassVar[int]
    INDEPENDENCE_WARNINGS_FIELD_NUMBER: _ClassVar[int]
    id: str
    owner: str
    name: str
    source_kind: _sources_pb2.SourceKind
    locator: str
    planned: bool
    last_success_at: _timestamp_pb2.Timestamp
    last_verified_at: _timestamp_pb2.Timestamp
    critical: bool
    reacquirable: bool
    reacquisition_command: str
    reacquisition_evidence: str
    reacquisition_needs_operator: bool
    reacquisition_auth_reason: str
    destinations: _containers.RepeatedCompositeFieldContainer[TargetDestinationStatus]
    all_destinations_backed_up: bool
    all_destinations_verified: bool
    physical_independence_proven: bool
    independence_warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., owner: _Optional[str] = ..., name: _Optional[str] = ..., source_kind: _Optional[_Union[_sources_pb2.SourceKind, str]] = ..., locator: _Optional[str] = ..., planned: _Optional[bool] = ..., last_success_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_verified_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., critical: _Optional[bool] = ..., reacquirable: _Optional[bool] = ..., reacquisition_command: _Optional[str] = ..., reacquisition_evidence: _Optional[str] = ..., reacquisition_needs_operator: _Optional[bool] = ..., reacquisition_auth_reason: _Optional[str] = ..., destinations: _Optional[_Iterable[_Union[TargetDestinationStatus, _Mapping]]] = ..., all_destinations_backed_up: _Optional[bool] = ..., all_destinations_verified: _Optional[bool] = ..., physical_independence_proven: _Optional[bool] = ..., independence_warnings: _Optional[_Iterable[str]] = ...) -> None: ...

class TargetDestinationStatus(_message.Message):
    __slots__ = ("destination_id", "last_success_at", "last_verified_at")
    DESTINATION_ID_FIELD_NUMBER: _ClassVar[int]
    LAST_SUCCESS_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_VERIFIED_AT_FIELD_NUMBER: _ClassVar[int]
    destination_id: str
    last_success_at: _timestamp_pb2.Timestamp
    last_verified_at: _timestamp_pb2.Timestamp
    def __init__(self, destination_id: _Optional[str] = ..., last_success_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_verified_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class SuggestedTarget(_message.Message):
    __slots__ = ("id", "owner", "name", "source_kind", "locator", "rationale", "approx_bytes", "sensitive", "warning", "critical", "reacquirable", "budget_max_bytes", "budget_max_age", "retention", "reacquisition_command", "reacquisition_evidence", "reacquisition_needs_operator", "reacquisition_auth_reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SOURCE_KIND_FIELD_NUMBER: _ClassVar[int]
    LOCATOR_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    APPROX_BYTES_FIELD_NUMBER: _ClassVar[int]
    SENSITIVE_FIELD_NUMBER: _ClassVar[int]
    WARNING_FIELD_NUMBER: _ClassVar[int]
    CRITICAL_FIELD_NUMBER: _ClassVar[int]
    REACQUIRABLE_FIELD_NUMBER: _ClassVar[int]
    BUDGET_MAX_BYTES_FIELD_NUMBER: _ClassVar[int]
    BUDGET_MAX_AGE_FIELD_NUMBER: _ClassVar[int]
    RETENTION_FIELD_NUMBER: _ClassVar[int]
    REACQUISITION_COMMAND_FIELD_NUMBER: _ClassVar[int]
    REACQUISITION_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    REACQUISITION_NEEDS_OPERATOR_FIELD_NUMBER: _ClassVar[int]
    REACQUISITION_AUTH_REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    owner: str
    name: str
    source_kind: _sources_pb2.SourceKind
    locator: str
    rationale: str
    approx_bytes: int
    sensitive: bool
    warning: str
    critical: bool
    reacquirable: bool
    budget_max_bytes: str
    budget_max_age: str
    retention: str
    reacquisition_command: str
    reacquisition_evidence: str
    reacquisition_needs_operator: bool
    reacquisition_auth_reason: str
    def __init__(self, id: _Optional[str] = ..., owner: _Optional[str] = ..., name: _Optional[str] = ..., source_kind: _Optional[_Union[_sources_pb2.SourceKind, str]] = ..., locator: _Optional[str] = ..., rationale: _Optional[str] = ..., approx_bytes: _Optional[int] = ..., sensitive: _Optional[bool] = ..., warning: _Optional[str] = ..., critical: _Optional[bool] = ..., reacquirable: _Optional[bool] = ..., budget_max_bytes: _Optional[str] = ..., budget_max_age: _Optional[str] = ..., retention: _Optional[str] = ..., reacquisition_command: _Optional[str] = ..., reacquisition_evidence: _Optional[str] = ..., reacquisition_needs_operator: _Optional[bool] = ..., reacquisition_auth_reason: _Optional[str] = ...) -> None: ...

class CoverageSummary(_message.Message):
    __slots__ = ("registered_count", "recommended_count", "sensitive_count", "planned_count", "backed_up_count", "verified_count", "default_coverage_complete", "has_sensitive_unreviewed", "has_unplanned_registered_targets", "has_unverified_targets", "destination_copy_count", "all_destination_backed_up_count", "all_destination_verified_count", "multi_destination_target_count", "physically_independent_target_count")
    REGISTERED_COUNT_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_COUNT_FIELD_NUMBER: _ClassVar[int]
    SENSITIVE_COUNT_FIELD_NUMBER: _ClassVar[int]
    PLANNED_COUNT_FIELD_NUMBER: _ClassVar[int]
    BACKED_UP_COUNT_FIELD_NUMBER: _ClassVar[int]
    VERIFIED_COUNT_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_COVERAGE_COMPLETE_FIELD_NUMBER: _ClassVar[int]
    HAS_SENSITIVE_UNREVIEWED_FIELD_NUMBER: _ClassVar[int]
    HAS_UNPLANNED_REGISTERED_TARGETS_FIELD_NUMBER: _ClassVar[int]
    HAS_UNVERIFIED_TARGETS_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_COPY_COUNT_FIELD_NUMBER: _ClassVar[int]
    ALL_DESTINATION_BACKED_UP_COUNT_FIELD_NUMBER: _ClassVar[int]
    ALL_DESTINATION_VERIFIED_COUNT_FIELD_NUMBER: _ClassVar[int]
    MULTI_DESTINATION_TARGET_COUNT_FIELD_NUMBER: _ClassVar[int]
    PHYSICALLY_INDEPENDENT_TARGET_COUNT_FIELD_NUMBER: _ClassVar[int]
    registered_count: int
    recommended_count: int
    sensitive_count: int
    planned_count: int
    backed_up_count: int
    verified_count: int
    default_coverage_complete: bool
    has_sensitive_unreviewed: bool
    has_unplanned_registered_targets: bool
    has_unverified_targets: bool
    destination_copy_count: int
    all_destination_backed_up_count: int
    all_destination_verified_count: int
    multi_destination_target_count: int
    physically_independent_target_count: int
    def __init__(self, registered_count: _Optional[int] = ..., recommended_count: _Optional[int] = ..., sensitive_count: _Optional[int] = ..., planned_count: _Optional[int] = ..., backed_up_count: _Optional[int] = ..., verified_count: _Optional[int] = ..., default_coverage_complete: _Optional[bool] = ..., has_sensitive_unreviewed: _Optional[bool] = ..., has_unplanned_registered_targets: _Optional[bool] = ..., has_unverified_targets: _Optional[bool] = ..., destination_copy_count: _Optional[int] = ..., all_destination_backed_up_count: _Optional[int] = ..., all_destination_verified_count: _Optional[int] = ..., multi_destination_target_count: _Optional[int] = ..., physically_independent_target_count: _Optional[int] = ...) -> None: ...

class CoverageReport(_message.Message):
    __slots__ = ("summary", "registered_targets", "recommended_targets", "sensitive_targets", "reacquirable_targets")
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    REGISTERED_TARGETS_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_TARGETS_FIELD_NUMBER: _ClassVar[int]
    SENSITIVE_TARGETS_FIELD_NUMBER: _ClassVar[int]
    REACQUIRABLE_TARGETS_FIELD_NUMBER: _ClassVar[int]
    summary: CoverageSummary
    registered_targets: _containers.RepeatedCompositeFieldContainer[RegisteredTarget]
    recommended_targets: _containers.RepeatedCompositeFieldContainer[SuggestedTarget]
    sensitive_targets: _containers.RepeatedCompositeFieldContainer[SuggestedTarget]
    reacquirable_targets: _containers.RepeatedCompositeFieldContainer[SuggestedTarget]
    def __init__(self, summary: _Optional[_Union[CoverageSummary, _Mapping]] = ..., registered_targets: _Optional[_Iterable[_Union[RegisteredTarget, _Mapping]]] = ..., recommended_targets: _Optional[_Iterable[_Union[SuggestedTarget, _Mapping]]] = ..., sensitive_targets: _Optional[_Iterable[_Union[SuggestedTarget, _Mapping]]] = ..., reacquirable_targets: _Optional[_Iterable[_Union[SuggestedTarget, _Mapping]]] = ...) -> None: ...

class GetCoverageReportRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetCoverageReportResponse(_message.Message):
    __slots__ = ("report",)
    REPORT_FIELD_NUMBER: _ClassVar[int]
    report: CoverageReport
    def __init__(self, report: _Optional[_Union[CoverageReport, _Mapping]] = ...) -> None: ...

class AcceptedTarget(_message.Message):
    __slots__ = ("target_id", "suggestion_id", "owner", "name", "source_kind", "locator", "sensitive", "critical")
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    SUGGESTION_ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SOURCE_KIND_FIELD_NUMBER: _ClassVar[int]
    LOCATOR_FIELD_NUMBER: _ClassVar[int]
    SENSITIVE_FIELD_NUMBER: _ClassVar[int]
    CRITICAL_FIELD_NUMBER: _ClassVar[int]
    target_id: str
    suggestion_id: str
    owner: str
    name: str
    source_kind: _sources_pb2.SourceKind
    locator: str
    sensitive: bool
    critical: bool
    def __init__(self, target_id: _Optional[str] = ..., suggestion_id: _Optional[str] = ..., owner: _Optional[str] = ..., name: _Optional[str] = ..., source_kind: _Optional[_Union[_sources_pb2.SourceKind, str]] = ..., locator: _Optional[str] = ..., sensitive: _Optional[bool] = ..., critical: _Optional[bool] = ...) -> None: ...

class AcceptError(_message.Message):
    __slots__ = ("suggestion_id", "owner", "name", "message")
    SUGGESTION_ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    suggestion_id: str
    owner: str
    name: str
    message: str
    def __init__(self, suggestion_id: _Optional[str] = ..., owner: _Optional[str] = ..., name: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class AcceptDefaultTargetsRequest(_message.Message):
    __slots__ = ("include_sensitive", "dry_run")
    INCLUDE_SENSITIVE_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    include_sensitive: bool
    dry_run: bool
    def __init__(self, include_sensitive: _Optional[bool] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class AcceptDefaultTargetsResponse(_message.Message):
    __slots__ = ("accepted", "skipped_sensitive", "failed", "dry_run")
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_SENSITIVE_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    accepted: _containers.RepeatedCompositeFieldContainer[AcceptedTarget]
    skipped_sensitive: _containers.RepeatedCompositeFieldContainer[SuggestedTarget]
    failed: _containers.RepeatedCompositeFieldContainer[AcceptError]
    dry_run: bool
    def __init__(self, accepted: _Optional[_Iterable[_Union[AcceptedTarget, _Mapping]]] = ..., skipped_sensitive: _Optional[_Iterable[_Union[SuggestedTarget, _Mapping]]] = ..., failed: _Optional[_Iterable[_Union[AcceptError, _Mapping]]] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class AdoptSensitiveConsentRequest(_message.Message):
    __slots__ = ("hub_ask_id",)
    HUB_ASK_ID_FIELD_NUMBER: _ClassVar[int]
    hub_ask_id: str
    def __init__(self, hub_ask_id: _Optional[str] = ...) -> None: ...

class AdoptSensitiveConsentResponse(_message.Message):
    __slots__ = ("policy", "registered_count", "failed_count")
    POLICY_FIELD_NUMBER: _ClassVar[int]
    REGISTERED_COUNT_FIELD_NUMBER: _ClassVar[int]
    FAILED_COUNT_FIELD_NUMBER: _ClassVar[int]
    policy: SensitivePolicy
    registered_count: int
    failed_count: int
    def __init__(self, policy: _Optional[_Union[SensitivePolicy, _Mapping]] = ..., registered_count: _Optional[int] = ..., failed_count: _Optional[int] = ...) -> None: ...
