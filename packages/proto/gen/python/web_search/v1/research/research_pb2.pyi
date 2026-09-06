import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from web_search.v1.shared import search_pb2 as _search_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AssessmentDisposition(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ASSESSMENT_DISPOSITION_UNSPECIFIED: _ClassVar[AssessmentDisposition]
    ASSESSMENT_SUPPORTED: _ClassVar[AssessmentDisposition]
    ASSESSMENT_CONTRADICTED: _ClassVar[AssessmentDisposition]
    ASSESSMENT_UNRESOLVED: _ClassVar[AssessmentDisposition]
    ASSESSMENT_UNKNOWN: _ClassVar[AssessmentDisposition]
ASSESSMENT_DISPOSITION_UNSPECIFIED: AssessmentDisposition
ASSESSMENT_SUPPORTED: AssessmentDisposition
ASSESSMENT_CONTRADICTED: AssessmentDisposition
ASSESSMENT_UNRESOLVED: AssessmentDisposition
ASSESSMENT_UNKNOWN: AssessmentDisposition

class Citation(_message.Message):
    __slots__ = ("result_index", "url", "title", "retrieved_at")
    RESULT_INDEX_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    RETRIEVED_AT_FIELD_NUMBER: _ClassVar[int]
    result_index: int
    url: str
    title: str
    retrieved_at: str
    def __init__(self, result_index: _Optional[int] = ..., url: _Optional[str] = ..., title: _Optional[str] = ..., retrieved_at: _Optional[str] = ...) -> None: ...

class Brief(_message.Message):
    __slots__ = ("query", "level", "summary", "citations")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    CITATIONS_FIELD_NUMBER: _ClassVar[int]
    query: str
    level: str
    summary: str
    citations: _containers.RepeatedCompositeFieldContainer[Citation]
    def __init__(self, query: _Optional[str] = ..., level: _Optional[str] = ..., summary: _Optional[str] = ..., citations: _Optional[_Iterable[_Union[Citation, _Mapping]]] = ...) -> None: ...

class RunL2Request(_message.Message):
    __slots__ = ("query", "top_n", "capture", "policy", "questions")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    TOP_N_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_FIELD_NUMBER: _ClassVar[int]
    POLICY_FIELD_NUMBER: _ClassVar[int]
    QUESTIONS_FIELD_NUMBER: _ClassVar[int]
    query: str
    top_n: int
    capture: bool
    policy: EvidencePolicy
    questions: _containers.RepeatedCompositeFieldContainer[ResearchQuestion]
    def __init__(self, query: _Optional[str] = ..., top_n: _Optional[int] = ..., capture: _Optional[bool] = ..., policy: _Optional[_Union[EvidencePolicy, _Mapping]] = ..., questions: _Optional[_Iterable[_Union[ResearchQuestion, _Mapping]]] = ...) -> None: ...

class RunL2Response(_message.Message):
    __slots__ = ("brief", "synthesis", "abstained", "captured_finding_ids", "degraded_engines", "abstain_reason", "excerpts", "evidence_receipt_ids", "fetch_failures", "assessments", "coverage")
    BRIEF_FIELD_NUMBER: _ClassVar[int]
    SYNTHESIS_FIELD_NUMBER: _ClassVar[int]
    ABSTAINED_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_FINDING_IDS_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_ENGINES_FIELD_NUMBER: _ClassVar[int]
    ABSTAIN_REASON_FIELD_NUMBER: _ClassVar[int]
    EXCERPTS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_RECEIPT_IDS_FIELD_NUMBER: _ClassVar[int]
    FETCH_FAILURES_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENTS_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    brief: Brief
    synthesis: str
    abstained: bool
    captured_finding_ids: _containers.RepeatedScalarFieldContainer[str]
    degraded_engines: _containers.RepeatedCompositeFieldContainer[_search_pb2.EngineIssue]
    abstain_reason: str
    excerpts: _containers.RepeatedCompositeFieldContainer[DocumentExcerpt]
    evidence_receipt_ids: _containers.RepeatedScalarFieldContainer[str]
    fetch_failures: _containers.RepeatedCompositeFieldContainer[FetchFailure]
    assessments: _containers.RepeatedCompositeFieldContainer[ClaimAssessment]
    coverage: _containers.RepeatedCompositeFieldContainer[QuestionCoverage]
    def __init__(self, brief: _Optional[_Union[Brief, _Mapping]] = ..., synthesis: _Optional[str] = ..., abstained: _Optional[bool] = ..., captured_finding_ids: _Optional[_Iterable[str]] = ..., degraded_engines: _Optional[_Iterable[_Union[_search_pb2.EngineIssue, _Mapping]]] = ..., abstain_reason: _Optional[str] = ..., excerpts: _Optional[_Iterable[_Union[DocumentExcerpt, _Mapping]]] = ..., evidence_receipt_ids: _Optional[_Iterable[str]] = ..., fetch_failures: _Optional[_Iterable[_Union[FetchFailure, _Mapping]]] = ..., assessments: _Optional[_Iterable[_Union[ClaimAssessment, _Mapping]]] = ..., coverage: _Optional[_Iterable[_Union[QuestionCoverage, _Mapping]]] = ...) -> None: ...

class FetchFailure(_message.Message):
    __slots__ = ("url", "code", "message", "retryable", "receipt_id")
    URL_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    RETRYABLE_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    url: str
    code: str
    message: str
    retryable: bool
    receipt_id: str
    def __init__(self, url: _Optional[str] = ..., code: _Optional[str] = ..., message: _Optional[str] = ..., retryable: _Optional[bool] = ..., receipt_id: _Optional[str] = ...) -> None: ...

class DocumentExcerpt(_message.Message):
    __slots__ = ("url", "title", "excerpt")
    URL_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    EXCERPT_FIELD_NUMBER: _ClassVar[int]
    url: str
    title: str
    excerpt: str
    def __init__(self, url: _Optional[str] = ..., title: _Optional[str] = ..., excerpt: _Optional[str] = ...) -> None: ...

class RunL3Request(_message.Message):
    __slots__ = ("query", "idempotency_key", "policy", "questions")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    POLICY_FIELD_NUMBER: _ClassVar[int]
    QUESTIONS_FIELD_NUMBER: _ClassVar[int]
    query: str
    idempotency_key: str
    policy: EvidencePolicy
    questions: _containers.RepeatedCompositeFieldContainer[ResearchQuestion]
    def __init__(self, query: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., policy: _Optional[_Union[EvidencePolicy, _Mapping]] = ..., questions: _Optional[_Iterable[_Union[ResearchQuestion, _Mapping]]] = ...) -> None: ...

class RunL3Response(_message.Message):
    __slots__ = ("run_id", "status")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    status: str
    def __init__(self, run_id: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class GetResearchStatusRequest(_message.Message):
    __slots__ = ("run_id",)
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    def __init__(self, run_id: _Optional[str] = ...) -> None: ...

class GetResearchStatusResponse(_message.Message):
    __slots__ = ("run_id", "status", "summary", "started_at", "finished_at", "error_msg", "result", "timed_out")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    ERROR_MSG_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    TIMED_OUT_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    status: str
    summary: str
    started_at: _timestamp_pb2.Timestamp
    finished_at: _timestamp_pb2.Timestamp
    error_msg: str
    result: _struct_pb2.Struct
    timed_out: bool
    def __init__(self, run_id: _Optional[str] = ..., status: _Optional[str] = ..., summary: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., finished_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., error_msg: _Optional[str] = ..., result: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., timed_out: _Optional[bool] = ...) -> None: ...

class GatherRelatedFindingsRequest(_message.Message):
    __slots__ = ("query", "max")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    MAX_FIELD_NUMBER: _ClassVar[int]
    query: str
    max: int
    def __init__(self, query: _Optional[str] = ..., max: _Optional[int] = ...) -> None: ...

class GatheredFinding(_message.Message):
    __slots__ = ("finding_id", "claim", "confidence", "status", "score")
    FINDING_ID_FIELD_NUMBER: _ClassVar[int]
    CLAIM_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    finding_id: str
    claim: str
    confidence: float
    status: str
    score: float
    def __init__(self, finding_id: _Optional[str] = ..., claim: _Optional[str] = ..., confidence: _Optional[float] = ..., status: _Optional[str] = ..., score: _Optional[float] = ...) -> None: ...

class GatherRelatedFindingsResponse(_message.Message):
    __slots__ = ("findings", "cap_applied")
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    CAP_APPLIED_FIELD_NUMBER: _ClassVar[int]
    findings: _containers.RepeatedCompositeFieldContainer[GatheredFinding]
    cap_applied: int
    def __init__(self, findings: _Optional[_Iterable[_Union[GatheredFinding, _Mapping]]] = ..., cap_applied: _Optional[int] = ...) -> None: ...

class GetCaptureStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetCaptureStatusResponse(_message.Message):
    __slots__ = ("pending", "delivered", "failed", "oldest_pending_at", "status")
    PENDING_FIELD_NUMBER: _ClassVar[int]
    DELIVERED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    OLDEST_PENDING_AT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    pending: int
    delivered: int
    failed: int
    oldest_pending_at: str
    status: str
    def __init__(self, pending: _Optional[int] = ..., delivered: _Optional[int] = ..., failed: _Optional[int] = ..., oldest_pending_at: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class GetEvidenceReceiptRequest(_message.Message):
    __slots__ = ("receipt_id",)
    RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    receipt_id: str
    def __init__(self, receipt_id: _Optional[str] = ...) -> None: ...

class GetEvidenceReceiptResponse(_message.Message):
    __slots__ = ("receipt_id", "observation_id", "original_url", "retrieved_at", "content_sha256", "artifact_id", "extraction_revision", "retention", "failure_code", "producer_execution_id")
    RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    OBSERVATION_ID_FIELD_NUMBER: _ClassVar[int]
    ORIGINAL_URL_FIELD_NUMBER: _ClassVar[int]
    RETRIEVED_AT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_SHA256_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_ID_FIELD_NUMBER: _ClassVar[int]
    EXTRACTION_REVISION_FIELD_NUMBER: _ClassVar[int]
    RETENTION_FIELD_NUMBER: _ClassVar[int]
    FAILURE_CODE_FIELD_NUMBER: _ClassVar[int]
    PRODUCER_EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    receipt_id: str
    observation_id: str
    original_url: str
    retrieved_at: str
    content_sha256: str
    artifact_id: str
    extraction_revision: str
    retention: str
    failure_code: str
    producer_execution_id: str
    def __init__(self, receipt_id: _Optional[str] = ..., observation_id: _Optional[str] = ..., original_url: _Optional[str] = ..., retrieved_at: _Optional[str] = ..., content_sha256: _Optional[str] = ..., artifact_id: _Optional[str] = ..., extraction_revision: _Optional[str] = ..., retention: _Optional[str] = ..., failure_code: _Optional[str] = ..., producer_execution_id: _Optional[str] = ...) -> None: ...

class GetEvidencePassageRequest(_message.Message):
    __slots__ = ("passage_id",)
    PASSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    passage_id: str
    def __init__(self, passage_id: _Optional[str] = ...) -> None: ...

class GetEvidencePassageResponse(_message.Message):
    __slots__ = ("passage_id", "receipt_id", "start_byte", "end_byte", "content", "content_sha256")
    PASSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    START_BYTE_FIELD_NUMBER: _ClassVar[int]
    END_BYTE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_SHA256_FIELD_NUMBER: _ClassVar[int]
    passage_id: str
    receipt_id: str
    start_byte: int
    end_byte: int
    content: str
    content_sha256: str
    def __init__(self, passage_id: _Optional[str] = ..., receipt_id: _Optional[str] = ..., start_byte: _Optional[int] = ..., end_byte: _Optional[int] = ..., content: _Optional[str] = ..., content_sha256: _Optional[str] = ...) -> None: ...

class GetEvidenceAssessmentRequest(_message.Message):
    __slots__ = ("assessment_id",)
    ASSESSMENT_ID_FIELD_NUMBER: _ClassVar[int]
    assessment_id: str
    def __init__(self, assessment_id: _Optional[str] = ...) -> None: ...

class GetEvidenceAssessmentResponse(_message.Message):
    __slots__ = ("assessment_id", "claim_id", "disposition", "policy_revision", "reason", "evidence_json")
    ASSESSMENT_ID_FIELD_NUMBER: _ClassVar[int]
    CLAIM_ID_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    POLICY_REVISION_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_JSON_FIELD_NUMBER: _ClassVar[int]
    assessment_id: str
    claim_id: str
    disposition: str
    policy_revision: str
    reason: str
    evidence_json: str
    def __init__(self, assessment_id: _Optional[str] = ..., claim_id: _Optional[str] = ..., disposition: _Optional[str] = ..., policy_revision: _Optional[str] = ..., reason: _Optional[str] = ..., evidence_json: _Optional[str] = ...) -> None: ...

class MethodRevision(_message.Message):
    __slots__ = ("id", "program_hash", "config_hash")
    ID_FIELD_NUMBER: _ClassVar[int]
    PROGRAM_HASH_FIELD_NUMBER: _ClassVar[int]
    CONFIG_HASH_FIELD_NUMBER: _ClassVar[int]
    id: str
    program_hash: str
    config_hash: str
    def __init__(self, id: _Optional[str] = ..., program_hash: _Optional[str] = ..., config_hash: _Optional[str] = ...) -> None: ...

class EvaluationReceipt(_message.Message):
    __slots__ = ("id", "candidate_hash", "report_hash", "accepted")
    ID_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_HASH_FIELD_NUMBER: _ClassVar[int]
    REPORT_HASH_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    id: str
    candidate_hash: str
    report_hash: str
    accepted: bool
    def __init__(self, id: _Optional[str] = ..., candidate_hash: _Optional[str] = ..., report_hash: _Optional[str] = ..., accepted: _Optional[bool] = ...) -> None: ...

class EvaluationReport(_message.Message):
    __slots__ = ("accepted", "reason", "baseline_method", "candidate_method", "population", "mean_baseline_effort", "mean_candidate_effort")
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    BASELINE_METHOD_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_METHOD_FIELD_NUMBER: _ClassVar[int]
    POPULATION_FIELD_NUMBER: _ClassVar[int]
    MEAN_BASELINE_EFFORT_FIELD_NUMBER: _ClassVar[int]
    MEAN_CANDIDATE_EFFORT_FIELD_NUMBER: _ClassVar[int]
    accepted: bool
    reason: str
    baseline_method: str
    candidate_method: str
    population: int
    mean_baseline_effort: float
    mean_candidate_effort: float
    def __init__(self, accepted: _Optional[bool] = ..., reason: _Optional[str] = ..., baseline_method: _Optional[str] = ..., candidate_method: _Optional[str] = ..., population: _Optional[int] = ..., mean_baseline_effort: _Optional[float] = ..., mean_candidate_effort: _Optional[float] = ...) -> None: ...

class MethodRelease(_message.Message):
    __slots__ = ("revision", "evaluation_id", "evaluation_hash", "rollback_of", "revision_hash")
    REVISION_FIELD_NUMBER: _ClassVar[int]
    EVALUATION_ID_FIELD_NUMBER: _ClassVar[int]
    EVALUATION_HASH_FIELD_NUMBER: _ClassVar[int]
    ROLLBACK_OF_FIELD_NUMBER: _ClassVar[int]
    REVISION_HASH_FIELD_NUMBER: _ClassVar[int]
    revision: MethodRevision
    evaluation_id: str
    evaluation_hash: str
    rollback_of: str
    revision_hash: str
    def __init__(self, revision: _Optional[_Union[MethodRevision, _Mapping]] = ..., evaluation_id: _Optional[str] = ..., evaluation_hash: _Optional[str] = ..., rollback_of: _Optional[str] = ..., revision_hash: _Optional[str] = ...) -> None: ...

class GetMethodReleaseRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetMethodReleaseResponse(_message.Message):
    __slots__ = ("release", "found")
    RELEASE_FIELD_NUMBER: _ClassVar[int]
    FOUND_FIELD_NUMBER: _ClassVar[int]
    release: MethodRelease
    found: bool
    def __init__(self, release: _Optional[_Union[MethodRelease, _Mapping]] = ..., found: _Optional[bool] = ...) -> None: ...

class PromoteMethodRequest(_message.Message):
    __slots__ = ("expected_current_hash", "revision", "receipt", "report", "grant")
    EXPECTED_CURRENT_HASH_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    GRANT_FIELD_NUMBER: _ClassVar[int]
    expected_current_hash: str
    revision: MethodRevision
    receipt: EvaluationReceipt
    report: EvaluationReport
    grant: str
    def __init__(self, expected_current_hash: _Optional[str] = ..., revision: _Optional[_Union[MethodRevision, _Mapping]] = ..., receipt: _Optional[_Union[EvaluationReceipt, _Mapping]] = ..., report: _Optional[_Union[EvaluationReport, _Mapping]] = ..., grant: _Optional[str] = ...) -> None: ...

class PromoteMethodResponse(_message.Message):
    __slots__ = ("release",)
    RELEASE_FIELD_NUMBER: _ClassVar[int]
    release: MethodRelease
    def __init__(self, release: _Optional[_Union[MethodRelease, _Mapping]] = ...) -> None: ...

class RollbackMethodRequest(_message.Message):
    __slots__ = ("expected_current_hash", "target_hash")
    EXPECTED_CURRENT_HASH_FIELD_NUMBER: _ClassVar[int]
    TARGET_HASH_FIELD_NUMBER: _ClassVar[int]
    expected_current_hash: str
    target_hash: str
    def __init__(self, expected_current_hash: _Optional[str] = ..., target_hash: _Optional[str] = ...) -> None: ...

class RollbackMethodResponse(_message.Message):
    __slots__ = ("release",)
    RELEASE_FIELD_NUMBER: _ClassVar[int]
    release: MethodRelease
    def __init__(self, release: _Optional[_Union[MethodRelease, _Mapping]] = ...) -> None: ...

class SuspendMethodRequest(_message.Message):
    __slots__ = ("revision_hash", "reason")
    REVISION_HASH_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    revision_hash: str
    reason: str
    def __init__(self, revision_hash: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class SuspendMethodResponse(_message.Message):
    __slots__ = ("revision_hash",)
    REVISION_HASH_FIELD_NUMBER: _ClassVar[int]
    revision_hash: str
    def __init__(self, revision_hash: _Optional[str] = ...) -> None: ...

class AnswerRequest(_message.Message):
    __slots__ = ("query", "effort", "max_age_seconds", "source_domains", "minimum_sources", "top_n", "capture", "finding_id", "policy", "questions")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    EFFORT_FIELD_NUMBER: _ClassVar[int]
    MAX_AGE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_DOMAINS_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_SOURCES_FIELD_NUMBER: _ClassVar[int]
    TOP_N_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_FIELD_NUMBER: _ClassVar[int]
    FINDING_ID_FIELD_NUMBER: _ClassVar[int]
    POLICY_FIELD_NUMBER: _ClassVar[int]
    QUESTIONS_FIELD_NUMBER: _ClassVar[int]
    query: str
    effort: str
    max_age_seconds: int
    source_domains: _containers.RepeatedScalarFieldContainer[str]
    minimum_sources: int
    top_n: int
    capture: bool
    finding_id: str
    policy: EvidencePolicy
    questions: _containers.RepeatedCompositeFieldContainer[ResearchQuestion]
    def __init__(self, query: _Optional[str] = ..., effort: _Optional[str] = ..., max_age_seconds: _Optional[int] = ..., source_domains: _Optional[_Iterable[str]] = ..., minimum_sources: _Optional[int] = ..., top_n: _Optional[int] = ..., capture: _Optional[bool] = ..., finding_id: _Optional[str] = ..., policy: _Optional[_Union[EvidencePolicy, _Mapping]] = ..., questions: _Optional[_Iterable[_Union[ResearchQuestion, _Mapping]]] = ...) -> None: ...

class AnswerResponse(_message.Message):
    __slots__ = ("status", "answer_kind", "brief", "results", "finding_ids", "abstained", "reason", "checked_at", "live_calls", "cached", "gaps", "captured_finding_ids", "assessments", "coverage")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ANSWER_KIND_FIELD_NUMBER: _ClassVar[int]
    BRIEF_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    FINDING_IDS_FIELD_NUMBER: _ClassVar[int]
    ABSTAINED_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    LIVE_CALLS_FIELD_NUMBER: _ClassVar[int]
    CACHED_FIELD_NUMBER: _ClassVar[int]
    GAPS_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_FINDING_IDS_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENTS_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    status: str
    answer_kind: str
    brief: Brief
    results: _containers.RepeatedCompositeFieldContainer[_search_pb2.SearchResult]
    finding_ids: _containers.RepeatedScalarFieldContainer[str]
    abstained: bool
    reason: str
    checked_at: _timestamp_pb2.Timestamp
    live_calls: int
    cached: bool
    gaps: _containers.RepeatedScalarFieldContainer[str]
    captured_finding_ids: _containers.RepeatedScalarFieldContainer[str]
    assessments: _containers.RepeatedCompositeFieldContainer[ClaimAssessment]
    coverage: _containers.RepeatedCompositeFieldContainer[QuestionCoverage]
    def __init__(self, status: _Optional[str] = ..., answer_kind: _Optional[str] = ..., brief: _Optional[_Union[Brief, _Mapping]] = ..., results: _Optional[_Iterable[_Union[_search_pb2.SearchResult, _Mapping]]] = ..., finding_ids: _Optional[_Iterable[str]] = ..., abstained: _Optional[bool] = ..., reason: _Optional[str] = ..., checked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., live_calls: _Optional[int] = ..., cached: _Optional[bool] = ..., gaps: _Optional[_Iterable[str]] = ..., captured_finding_ids: _Optional[_Iterable[str]] = ..., assessments: _Optional[_Iterable[_Union[ClaimAssessment, _Mapping]]] = ..., coverage: _Optional[_Iterable[_Union[QuestionCoverage, _Mapping]]] = ...) -> None: ...

class WaitResearchRequest(_message.Message):
    __slots__ = ("run_id", "timeout_seconds")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    timeout_seconds: int
    def __init__(self, run_id: _Optional[str] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class CancelResearchRequest(_message.Message):
    __slots__ = ("run_id", "reason", "idempotency_key")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    reason: str
    idempotency_key: str
    def __init__(self, run_id: _Optional[str] = ..., reason: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class EvidencePolicy(_message.Message):
    __slots__ = ("max_age_seconds", "source_domains", "minimum_sources", "top_n", "max_questions", "max_evidence_bytes", "require_independent_sources")
    MAX_AGE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_DOMAINS_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_SOURCES_FIELD_NUMBER: _ClassVar[int]
    TOP_N_FIELD_NUMBER: _ClassVar[int]
    MAX_QUESTIONS_FIELD_NUMBER: _ClassVar[int]
    MAX_EVIDENCE_BYTES_FIELD_NUMBER: _ClassVar[int]
    REQUIRE_INDEPENDENT_SOURCES_FIELD_NUMBER: _ClassVar[int]
    max_age_seconds: int
    source_domains: _containers.RepeatedScalarFieldContainer[str]
    minimum_sources: int
    top_n: int
    max_questions: int
    max_evidence_bytes: int
    require_independent_sources: bool
    def __init__(self, max_age_seconds: _Optional[int] = ..., source_domains: _Optional[_Iterable[str]] = ..., minimum_sources: _Optional[int] = ..., top_n: _Optional[int] = ..., max_questions: _Optional[int] = ..., max_evidence_bytes: _Optional[int] = ..., require_independent_sources: _Optional[bool] = ...) -> None: ...

class ResearchQuestion(_message.Message):
    __slots__ = ("id", "prompt", "required")
    ID_FIELD_NUMBER: _ClassVar[int]
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    id: str
    prompt: str
    required: bool
    def __init__(self, id: _Optional[str] = ..., prompt: _Optional[str] = ..., required: _Optional[bool] = ...) -> None: ...

class EvidencePassageRef(_message.Message):
    __slots__ = ("receipt_id", "passage_id", "content_hash", "extraction_revision")
    RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    PASSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    EXTRACTION_REVISION_FIELD_NUMBER: _ClassVar[int]
    receipt_id: str
    passage_id: str
    content_hash: str
    extraction_revision: str
    def __init__(self, receipt_id: _Optional[str] = ..., passage_id: _Optional[str] = ..., content_hash: _Optional[str] = ..., extraction_revision: _Optional[str] = ...) -> None: ...

class ClaimAssessment(_message.Message):
    __slots__ = ("claim_id", "disposition", "evidence", "policy_revision", "reason", "assessment_id")
    CLAIM_ID_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    POLICY_REVISION_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENT_ID_FIELD_NUMBER: _ClassVar[int]
    claim_id: str
    disposition: AssessmentDisposition
    evidence: _containers.RepeatedCompositeFieldContainer[EvidencePassageRef]
    policy_revision: str
    reason: str
    assessment_id: str
    def __init__(self, claim_id: _Optional[str] = ..., disposition: _Optional[_Union[AssessmentDisposition, str]] = ..., evidence: _Optional[_Iterable[_Union[EvidencePassageRef, _Mapping]]] = ..., policy_revision: _Optional[str] = ..., reason: _Optional[str] = ..., assessment_id: _Optional[str] = ...) -> None: ...

class QuestionCoverage(_message.Message):
    __slots__ = ("question_id", "status", "claim_ids", "unresolved_reason")
    QUESTION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CLAIM_IDS_FIELD_NUMBER: _ClassVar[int]
    UNRESOLVED_REASON_FIELD_NUMBER: _ClassVar[int]
    question_id: str
    status: str
    claim_ids: _containers.RepeatedScalarFieldContainer[str]
    unresolved_reason: str
    def __init__(self, question_id: _Optional[str] = ..., status: _Optional[str] = ..., claim_ids: _Optional[_Iterable[str]] = ..., unresolved_reason: _Optional[str] = ...) -> None: ...

class EvidenceReceipt(_message.Message):
    __slots__ = ("receipt_id", "observation_id", "url", "retrieved_at", "content_hash", "extraction_revision", "retention", "failure_code")
    RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    OBSERVATION_ID_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    RETRIEVED_AT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    EXTRACTION_REVISION_FIELD_NUMBER: _ClassVar[int]
    RETENTION_FIELD_NUMBER: _ClassVar[int]
    FAILURE_CODE_FIELD_NUMBER: _ClassVar[int]
    receipt_id: str
    observation_id: str
    url: str
    retrieved_at: str
    content_hash: str
    extraction_revision: str
    retention: str
    failure_code: str
    def __init__(self, receipt_id: _Optional[str] = ..., observation_id: _Optional[str] = ..., url: _Optional[str] = ..., retrieved_at: _Optional[str] = ..., content_hash: _Optional[str] = ..., extraction_revision: _Optional[str] = ..., retention: _Optional[str] = ..., failure_code: _Optional[str] = ...) -> None: ...
