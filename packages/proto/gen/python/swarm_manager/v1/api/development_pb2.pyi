from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DevelopmentReference(_message.Message):
    __slots__ = ("effort_id", "revision", "authority_digest", "contract_digest", "commission_subject_digest")
    EFFORT_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    AUTHORITY_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CONTRACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    COMMISSION_SUBJECT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    effort_id: str
    revision: int
    authority_digest: str
    contract_digest: str
    commission_subject_digest: str
    def __init__(self, effort_id: _Optional[str] = ..., revision: _Optional[int] = ..., authority_digest: _Optional[str] = ..., contract_digest: _Optional[str] = ..., commission_subject_digest: _Optional[str] = ...) -> None: ...

class DevelopmentArtifact(_message.Message):
    __slots__ = ("id", "relative_path", "sha256", "size_bytes", "media_type", "captured_at", "snapshot_digest")
    ID_FIELD_NUMBER: _ClassVar[int]
    RELATIVE_PATH_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    id: str
    relative_path: str
    sha256: str
    size_bytes: int
    media_type: str
    captured_at: str
    snapshot_digest: str
    def __init__(self, id: _Optional[str] = ..., relative_path: _Optional[str] = ..., sha256: _Optional[str] = ..., size_bytes: _Optional[int] = ..., media_type: _Optional[str] = ..., captured_at: _Optional[str] = ..., snapshot_digest: _Optional[str] = ...) -> None: ...

class DevelopmentLimits(_message.Message):
    __slots__ = ("max_workers", "max_concurrency", "max_depth", "max_active_descendants", "max_premium_descendants", "max_tokens", "max_charge_micro_usd", "max_wall_seconds", "max_wait_seconds", "deadline")
    MAX_WORKERS_FIELD_NUMBER: _ClassVar[int]
    MAX_CONCURRENCY_FIELD_NUMBER: _ClassVar[int]
    MAX_DEPTH_FIELD_NUMBER: _ClassVar[int]
    MAX_ACTIVE_DESCENDANTS_FIELD_NUMBER: _ClassVar[int]
    MAX_PREMIUM_DESCENDANTS_FIELD_NUMBER: _ClassVar[int]
    MAX_TOKENS_FIELD_NUMBER: _ClassVar[int]
    MAX_CHARGE_MICRO_USD_FIELD_NUMBER: _ClassVar[int]
    MAX_WALL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    MAX_WAIT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    DEADLINE_FIELD_NUMBER: _ClassVar[int]
    max_workers: int
    max_concurrency: int
    max_depth: int
    max_active_descendants: int
    max_premium_descendants: int
    max_tokens: int
    max_charge_micro_usd: int
    max_wall_seconds: int
    max_wait_seconds: int
    deadline: str
    def __init__(self, max_workers: _Optional[int] = ..., max_concurrency: _Optional[int] = ..., max_depth: _Optional[int] = ..., max_active_descendants: _Optional[int] = ..., max_premium_descendants: _Optional[int] = ..., max_tokens: _Optional[int] = ..., max_charge_micro_usd: _Optional[int] = ..., max_wall_seconds: _Optional[int] = ..., max_wait_seconds: _Optional[int] = ..., deadline: _Optional[str] = ...) -> None: ...

class DevelopmentView(_message.Message):
    __slots__ = ("reference", "work_shape", "owner_subject", "scope_allow", "scope_deny", "limits", "generation", "approved", "revoked", "artifacts", "launch_blockers", "outcome_accepted", "tested_product_digest", "evidence_receipt_ids", "accounting_status", "owner", "evidence_set_digest", "criteria", "disposition_version", "retained_decisions", "accounting", "required_criterion_ids")
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    WORK_SHAPE_FIELD_NUMBER: _ClassVar[int]
    OWNER_SUBJECT_FIELD_NUMBER: _ClassVar[int]
    SCOPE_ALLOW_FIELD_NUMBER: _ClassVar[int]
    SCOPE_DENY_FIELD_NUMBER: _ClassVar[int]
    LIMITS_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    APPROVED_FIELD_NUMBER: _ClassVar[int]
    REVOKED_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    LAUNCH_BLOCKERS_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    TESTED_PRODUCT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_RECEIPT_IDS_FIELD_NUMBER: _ClassVar[int]
    ACCOUNTING_STATUS_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_SET_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CRITERIA_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_VERSION_FIELD_NUMBER: _ClassVar[int]
    RETAINED_DECISIONS_FIELD_NUMBER: _ClassVar[int]
    ACCOUNTING_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_CRITERION_IDS_FIELD_NUMBER: _ClassVar[int]
    reference: DevelopmentReference
    work_shape: str
    owner_subject: str
    scope_allow: _containers.RepeatedScalarFieldContainer[str]
    scope_deny: _containers.RepeatedScalarFieldContainer[str]
    limits: DevelopmentLimits
    generation: int
    approved: bool
    revoked: bool
    artifacts: _containers.RepeatedCompositeFieldContainer[DevelopmentArtifact]
    launch_blockers: _containers.RepeatedScalarFieldContainer[str]
    outcome_accepted: bool
    tested_product_digest: str
    evidence_receipt_ids: _containers.RepeatedScalarFieldContainer[str]
    accounting_status: str
    owner: DevelopmentActor
    evidence_set_digest: str
    criteria: _containers.RepeatedCompositeFieldContainer[DevelopmentCriterionEvidence]
    disposition_version: int
    retained_decisions: _containers.RepeatedCompositeFieldContainer[DevelopmentDecisionResponse]
    accounting: DevelopmentAccounting
    required_criterion_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, reference: _Optional[_Union[DevelopmentReference, _Mapping]] = ..., work_shape: _Optional[str] = ..., owner_subject: _Optional[str] = ..., scope_allow: _Optional[_Iterable[str]] = ..., scope_deny: _Optional[_Iterable[str]] = ..., limits: _Optional[_Union[DevelopmentLimits, _Mapping]] = ..., generation: _Optional[int] = ..., approved: _Optional[bool] = ..., revoked: _Optional[bool] = ..., artifacts: _Optional[_Iterable[_Union[DevelopmentArtifact, _Mapping]]] = ..., launch_blockers: _Optional[_Iterable[str]] = ..., outcome_accepted: _Optional[bool] = ..., tested_product_digest: _Optional[str] = ..., evidence_receipt_ids: _Optional[_Iterable[str]] = ..., accounting_status: _Optional[str] = ..., owner: _Optional[_Union[DevelopmentActor, _Mapping]] = ..., evidence_set_digest: _Optional[str] = ..., criteria: _Optional[_Iterable[_Union[DevelopmentCriterionEvidence, _Mapping]]] = ..., disposition_version: _Optional[int] = ..., retained_decisions: _Optional[_Iterable[_Union[DevelopmentDecisionResponse, _Mapping]]] = ..., accounting: _Optional[_Union[DevelopmentAccounting, _Mapping]] = ..., required_criterion_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class GetDevelopmentRequest(_message.Message):
    __slots__ = ("effort_id",)
    EFFORT_ID_FIELD_NUMBER: _ClassVar[int]
    effort_id: str
    def __init__(self, effort_id: _Optional[str] = ...) -> None: ...

class GetDevelopmentResponse(_message.Message):
    __slots__ = ("development",)
    DEVELOPMENT_FIELD_NUMBER: _ClassVar[int]
    development: DevelopmentView
    def __init__(self, development: _Optional[_Union[DevelopmentView, _Mapping]] = ...) -> None: ...

class GetDevelopmentArtifactRequest(_message.Message):
    __slots__ = ("reference", "artifact_id")
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_ID_FIELD_NUMBER: _ClassVar[int]
    reference: DevelopmentReference
    artifact_id: str
    def __init__(self, reference: _Optional[_Union[DevelopmentReference, _Mapping]] = ..., artifact_id: _Optional[str] = ...) -> None: ...

class GetDevelopmentArtifactResponse(_message.Message):
    __slots__ = ("artifact", "content", "reference")
    ARTIFACT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    artifact: DevelopmentArtifact
    content: bytes
    reference: DevelopmentReference
    def __init__(self, artifact: _Optional[_Union[DevelopmentArtifact, _Mapping]] = ..., content: _Optional[bytes] = ..., reference: _Optional[_Union[DevelopmentReference, _Mapping]] = ...) -> None: ...

class ApproveDevelopmentRequest(_message.Message):
    __slots__ = ("reference", "expected_generation", "request_id")
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_GENERATION_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    reference: DevelopmentReference
    expected_generation: int
    request_id: str
    def __init__(self, reference: _Optional[_Union[DevelopmentReference, _Mapping]] = ..., expected_generation: _Optional[int] = ..., request_id: _Optional[str] = ...) -> None: ...

class RevokeDevelopmentRequest(_message.Message):
    __slots__ = ("reference", "expected_generation", "request_id")
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_GENERATION_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    reference: DevelopmentReference
    expected_generation: int
    request_id: str
    def __init__(self, reference: _Optional[_Union[DevelopmentReference, _Mapping]] = ..., expected_generation: _Optional[int] = ..., request_id: _Optional[str] = ...) -> None: ...

class AcceptDevelopmentRequest(_message.Message):
    __slots__ = ("reference", "expected_generation", "request_id", "tested_product_digest", "expected_disposition_version")
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_GENERATION_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    TESTED_PRODUCT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_DISPOSITION_VERSION_FIELD_NUMBER: _ClassVar[int]
    reference: DevelopmentReference
    expected_generation: int
    request_id: str
    tested_product_digest: str
    expected_disposition_version: int
    def __init__(self, reference: _Optional[_Union[DevelopmentReference, _Mapping]] = ..., expected_generation: _Optional[int] = ..., request_id: _Optional[str] = ..., tested_product_digest: _Optional[str] = ..., expected_disposition_version: _Optional[int] = ...) -> None: ...

class DevelopmentDecisionResponse(_message.Message):
    __slots__ = ("request_id", "action", "reference", "generation", "actor_subject", "decided_at", "evidence_set_digest", "actor", "tested_product_digest", "criteria", "receipt_digest", "disposition_version")
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    ACTOR_SUBJECT_FIELD_NUMBER: _ClassVar[int]
    DECIDED_AT_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_SET_DIGEST_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    TESTED_PRODUCT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CRITERIA_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_VERSION_FIELD_NUMBER: _ClassVar[int]
    request_id: str
    action: str
    reference: DevelopmentReference
    generation: int
    actor_subject: str
    decided_at: str
    evidence_set_digest: str
    actor: DevelopmentActor
    tested_product_digest: str
    criteria: _containers.RepeatedCompositeFieldContainer[DevelopmentCriterionEvidence]
    receipt_digest: str
    disposition_version: int
    def __init__(self, request_id: _Optional[str] = ..., action: _Optional[str] = ..., reference: _Optional[_Union[DevelopmentReference, _Mapping]] = ..., generation: _Optional[int] = ..., actor_subject: _Optional[str] = ..., decided_at: _Optional[str] = ..., evidence_set_digest: _Optional[str] = ..., actor: _Optional[_Union[DevelopmentActor, _Mapping]] = ..., tested_product_digest: _Optional[str] = ..., criteria: _Optional[_Iterable[_Union[DevelopmentCriterionEvidence, _Mapping]]] = ..., receipt_digest: _Optional[str] = ..., disposition_version: _Optional[int] = ...) -> None: ...

class DevelopmentProposal(_message.Message):
    __slots__ = ("work_shape", "title", "description", "required_criterion_ids", "scope_allow", "scope_deny", "limits", "source_relative_paths", "proposed_effects", "target_subject_ref")
    WORK_SHAPE_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_CRITERION_IDS_FIELD_NUMBER: _ClassVar[int]
    SCOPE_ALLOW_FIELD_NUMBER: _ClassVar[int]
    SCOPE_DENY_FIELD_NUMBER: _ClassVar[int]
    LIMITS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_RELATIVE_PATHS_FIELD_NUMBER: _ClassVar[int]
    PROPOSED_EFFECTS_FIELD_NUMBER: _ClassVar[int]
    TARGET_SUBJECT_REF_FIELD_NUMBER: _ClassVar[int]
    work_shape: str
    title: str
    description: str
    required_criterion_ids: _containers.RepeatedScalarFieldContainer[str]
    scope_allow: _containers.RepeatedScalarFieldContainer[str]
    scope_deny: _containers.RepeatedScalarFieldContainer[str]
    limits: DevelopmentLimits
    source_relative_paths: _containers.RepeatedScalarFieldContainer[str]
    proposed_effects: _containers.RepeatedScalarFieldContainer[str]
    target_subject_ref: str
    def __init__(self, work_shape: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., required_criterion_ids: _Optional[_Iterable[str]] = ..., scope_allow: _Optional[_Iterable[str]] = ..., scope_deny: _Optional[_Iterable[str]] = ..., limits: _Optional[_Union[DevelopmentLimits, _Mapping]] = ..., source_relative_paths: _Optional[_Iterable[str]] = ..., proposed_effects: _Optional[_Iterable[str]] = ..., target_subject_ref: _Optional[str] = ...) -> None: ...

class PreviewDevelopmentRequest(_message.Message):
    __slots__ = ("reference", "proposal")
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    PROPOSAL_FIELD_NUMBER: _ClassVar[int]
    reference: DevelopmentReference
    proposal: DevelopmentProposal
    def __init__(self, reference: _Optional[_Union[DevelopmentReference, _Mapping]] = ..., proposal: _Optional[_Union[DevelopmentProposal, _Mapping]] = ...) -> None: ...

class PreviewDevelopmentResponse(_message.Message):
    __slots__ = ("reference", "observed_sources", "proposal_digest", "launch_blockers", "proposal", "completeness_findings", "conflicts")
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_SOURCES_FIELD_NUMBER: _ClassVar[int]
    PROPOSAL_DIGEST_FIELD_NUMBER: _ClassVar[int]
    LAUNCH_BLOCKERS_FIELD_NUMBER: _ClassVar[int]
    PROPOSAL_FIELD_NUMBER: _ClassVar[int]
    COMPLETENESS_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    CONFLICTS_FIELD_NUMBER: _ClassVar[int]
    reference: DevelopmentReference
    observed_sources: _containers.RepeatedCompositeFieldContainer[DevelopmentSourceObservation]
    proposal_digest: str
    launch_blockers: _containers.RepeatedScalarFieldContainer[str]
    proposal: DevelopmentProposal
    completeness_findings: _containers.RepeatedScalarFieldContainer[str]
    conflicts: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, reference: _Optional[_Union[DevelopmentReference, _Mapping]] = ..., observed_sources: _Optional[_Iterable[_Union[DevelopmentSourceObservation, _Mapping]]] = ..., proposal_digest: _Optional[str] = ..., launch_blockers: _Optional[_Iterable[str]] = ..., proposal: _Optional[_Union[DevelopmentProposal, _Mapping]] = ..., completeness_findings: _Optional[_Iterable[str]] = ..., conflicts: _Optional[_Iterable[str]] = ...) -> None: ...

class DevelopmentActor(_message.Message):
    __slots__ = ("subject", "provider", "realm", "source")
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    REALM_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    subject: str
    provider: str
    realm: str
    source: str
    def __init__(self, subject: _Optional[str] = ..., provider: _Optional[str] = ..., realm: _Optional[str] = ..., source: _Optional[str] = ...) -> None: ...

class DevelopmentCriterionEvidence(_message.Message):
    __slots__ = ("criterion_id", "receipt_ids", "receipt_set_digest", "tested_product_digest", "satisfied", "unavailable_reason")
    CRITERION_ID_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_IDS_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_SET_DIGEST_FIELD_NUMBER: _ClassVar[int]
    TESTED_PRODUCT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    SATISFIED_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_REASON_FIELD_NUMBER: _ClassVar[int]
    criterion_id: str
    receipt_ids: _containers.RepeatedScalarFieldContainer[str]
    receipt_set_digest: str
    tested_product_digest: str
    satisfied: bool
    unavailable_reason: str
    def __init__(self, criterion_id: _Optional[str] = ..., receipt_ids: _Optional[_Iterable[str]] = ..., receipt_set_digest: _Optional[str] = ..., tested_product_digest: _Optional[str] = ..., satisfied: _Optional[bool] = ..., unavailable_reason: _Optional[str] = ...) -> None: ...

class DevelopmentSourceObservation(_message.Message):
    __slots__ = ("relative_path", "sha256", "size_bytes", "media_type")
    RELATIVE_PATH_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    relative_path: str
    sha256: str
    size_bytes: int
    media_type: str
    def __init__(self, relative_path: _Optional[str] = ..., sha256: _Optional[str] = ..., size_bytes: _Optional[int] = ..., media_type: _Optional[str] = ...) -> None: ...

class DevelopmentAccounting(_message.Message):
    __slots__ = ("status", "tokens", "charge_micro_usd", "wall_seconds", "active_descendants")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TOKENS_FIELD_NUMBER: _ClassVar[int]
    CHARGE_MICRO_USD_FIELD_NUMBER: _ClassVar[int]
    WALL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_DESCENDANTS_FIELD_NUMBER: _ClassVar[int]
    status: str
    tokens: int
    charge_micro_usd: int
    wall_seconds: int
    active_descendants: int
    def __init__(self, status: _Optional[str] = ..., tokens: _Optional[int] = ..., charge_micro_usd: _Optional[int] = ..., wall_seconds: _Optional[int] = ..., active_descendants: _Optional[int] = ...) -> None: ...
