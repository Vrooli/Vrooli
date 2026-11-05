from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EnsureCampaignRequest(_message.Message):
    __slots__ = ("location", "tag", "patterns", "max_files")
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    TAG_FIELD_NUMBER: _ClassVar[int]
    PATTERNS_FIELD_NUMBER: _ClassVar[int]
    MAX_FILES_FIELD_NUMBER: _ClassVar[int]
    location: str
    tag: str
    patterns: _containers.RepeatedScalarFieldContainer[str]
    max_files: int
    def __init__(self, location: _Optional[str] = ..., tag: _Optional[str] = ..., patterns: _Optional[_Iterable[str]] = ..., max_files: _Optional[int] = ...) -> None: ...

class CampaignResponse(_message.Message):
    __slots__ = ("campaign_id", "revision")
    CAMPAIGN_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    campaign_id: str
    revision: int
    def __init__(self, campaign_id: _Optional[str] = ..., revision: _Optional[int] = ...) -> None: ...

class PreviewRequest(_message.Message):
    __slots__ = ("campaign_id", "limit")
    CAMPAIGN_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    campaign_id: str
    limit: int
    def __init__(self, campaign_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class Candidate(_message.Message):
    __slots__ = ("file_id", "path", "revision", "score", "priority", "reviewed")
    FILE_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    REVIEWED_FIELD_NUMBER: _ClassVar[int]
    file_id: str
    path: str
    revision: str
    score: float
    priority: float
    reviewed: bool
    def __init__(self, file_id: _Optional[str] = ..., path: _Optional[str] = ..., revision: _Optional[str] = ..., score: _Optional[float] = ..., priority: _Optional[float] = ..., reviewed: _Optional[bool] = ...) -> None: ...

class PreviewResponse(_message.Message):
    __slots__ = ("candidates", "campaign_revision", "observed_at", "eligible_count", "active_claim_count", "oldest_eligible_age_seconds", "oldest_active_claim_age_seconds", "integrity", "storage_writes")
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    CAMPAIGN_REVISION_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    ELIGIBLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_CLAIM_COUNT_FIELD_NUMBER: _ClassVar[int]
    OLDEST_ELIGIBLE_AGE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    OLDEST_ACTIVE_CLAIM_AGE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    INTEGRITY_FIELD_NUMBER: _ClassVar[int]
    STORAGE_WRITES_FIELD_NUMBER: _ClassVar[int]
    candidates: _containers.RepeatedCompositeFieldContainer[Candidate]
    campaign_revision: int
    observed_at: str
    eligible_count: int
    active_claim_count: int
    oldest_eligible_age_seconds: int
    oldest_active_claim_age_seconds: int
    integrity: ClaimIntegrity
    storage_writes: StorageWriteObservation
    def __init__(self, candidates: _Optional[_Iterable[_Union[Candidate, _Mapping]]] = ..., campaign_revision: _Optional[int] = ..., observed_at: _Optional[str] = ..., eligible_count: _Optional[int] = ..., active_claim_count: _Optional[int] = ..., oldest_eligible_age_seconds: _Optional[int] = ..., oldest_active_claim_age_seconds: _Optional[int] = ..., integrity: _Optional[_Union[ClaimIntegrity, _Mapping]] = ..., storage_writes: _Optional[_Union[StorageWriteObservation, _Mapping]] = ...) -> None: ...

class ClaimIntegrity(_message.Message):
    __slots__ = ("check_schema_version", "observed_at", "examined_claims", "violations")
    CHECK_SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    EXAMINED_CLAIMS_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    check_schema_version: int
    observed_at: str
    examined_claims: int
    violations: int
    def __init__(self, check_schema_version: _Optional[int] = ..., observed_at: _Optional[str] = ..., examined_claims: _Optional[int] = ..., violations: _Optional[int] = ...) -> None: ...

class StorageWriteObservation(_message.Message):
    __slots__ = ("epoch", "observed_at", "completed_total", "failed_total", "consecutive_failures", "last_completed_at", "last_completed_age_seconds", "durable_sync_supported")
    EPOCH_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_TOTAL_FIELD_NUMBER: _ClassVar[int]
    FAILED_TOTAL_FIELD_NUMBER: _ClassVar[int]
    CONSECUTIVE_FAILURES_FIELD_NUMBER: _ClassVar[int]
    LAST_COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_COMPLETED_AGE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    DURABLE_SYNC_SUPPORTED_FIELD_NUMBER: _ClassVar[int]
    epoch: str
    observed_at: str
    completed_total: int
    failed_total: int
    consecutive_failures: int
    last_completed_at: str
    last_completed_age_seconds: int
    durable_sync_supported: bool
    def __init__(self, epoch: _Optional[str] = ..., observed_at: _Optional[str] = ..., completed_total: _Optional[int] = ..., failed_total: _Optional[int] = ..., consecutive_failures: _Optional[int] = ..., last_completed_at: _Optional[str] = ..., last_completed_age_seconds: _Optional[int] = ..., durable_sync_supported: _Optional[bool] = ...) -> None: ...

class ClaimRequest(_message.Message):
    __slots__ = ("campaign_id", "request_id", "worker", "ttl_seconds")
    CAMPAIGN_ID_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    WORKER_FIELD_NUMBER: _ClassVar[int]
    TTL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    campaign_id: str
    request_id: str
    worker: str
    ttl_seconds: int
    def __init__(self, campaign_id: _Optional[str] = ..., request_id: _Optional[str] = ..., worker: _Optional[str] = ..., ttl_seconds: _Optional[int] = ...) -> None: ...

class Claim(_message.Message):
    __slots__ = ("id", "request_id", "worker", "file_id", "path", "revision", "expires_at", "outcome", "evidence", "completed_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    WORKER_FIELD_NUMBER: _ClassVar[int]
    FILE_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    request_id: str
    worker: str
    file_id: str
    path: str
    revision: str
    expires_at: str
    outcome: str
    evidence: str
    completed_at: str
    def __init__(self, id: _Optional[str] = ..., request_id: _Optional[str] = ..., worker: _Optional[str] = ..., file_id: _Optional[str] = ..., path: _Optional[str] = ..., revision: _Optional[str] = ..., expires_at: _Optional[str] = ..., outcome: _Optional[str] = ..., evidence: _Optional[str] = ..., completed_at: _Optional[str] = ...) -> None: ...

class ClaimResponse(_message.Message):
    __slots__ = ("claim", "no_work")
    CLAIM_FIELD_NUMBER: _ClassVar[int]
    NO_WORK_FIELD_NUMBER: _ClassVar[int]
    claim: Claim
    no_work: bool
    def __init__(self, claim: _Optional[_Union[Claim, _Mapping]] = ..., no_work: _Optional[bool] = ...) -> None: ...

class CompleteRequest(_message.Message):
    __slots__ = ("campaign_id", "claim_id", "worker", "outcome", "evidence")
    CAMPAIGN_ID_FIELD_NUMBER: _ClassVar[int]
    CLAIM_ID_FIELD_NUMBER: _ClassVar[int]
    WORKER_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    campaign_id: str
    claim_id: str
    worker: str
    outcome: str
    evidence: str
    def __init__(self, campaign_id: _Optional[str] = ..., claim_id: _Optional[str] = ..., worker: _Optional[str] = ..., outcome: _Optional[str] = ..., evidence: _Optional[str] = ...) -> None: ...
