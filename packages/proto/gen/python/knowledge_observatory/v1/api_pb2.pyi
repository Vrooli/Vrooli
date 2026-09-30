from buf.validate import validate_pb2 as _validate_pb2
from common.v1 import maturity_pb2 as _maturity_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DocHealthSeverity(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DOC_HEALTH_SEVERITY_UNSPECIFIED: _ClassVar[DocHealthSeverity]
    DOC_HEALTH_SEVERITY_INFO: _ClassVar[DocHealthSeverity]
    DOC_HEALTH_SEVERITY_WARNING: _ClassVar[DocHealthSeverity]
    DOC_HEALTH_SEVERITY_FAILURE: _ClassVar[DocHealthSeverity]
DOC_HEALTH_SEVERITY_UNSPECIFIED: DocHealthSeverity
DOC_HEALTH_SEVERITY_INFO: DocHealthSeverity
DOC_HEALTH_SEVERITY_WARNING: DocHealthSeverity
DOC_HEALTH_SEVERITY_FAILURE: DocHealthSeverity

class SearchRequest(_message.Message):
    __slots__ = ("query", "collection", "namespaces", "visibility", "tags", "ingested_after", "ingested_before", "limit", "threshold")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    COLLECTION_FIELD_NUMBER: _ClassVar[int]
    NAMESPACES_FIELD_NUMBER: _ClassVar[int]
    VISIBILITY_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    INGESTED_AFTER_FIELD_NUMBER: _ClassVar[int]
    INGESTED_BEFORE_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    query: str
    collection: str
    namespaces: _containers.RepeatedScalarFieldContainer[str]
    visibility: _containers.RepeatedScalarFieldContainer[str]
    tags: _containers.RepeatedScalarFieldContainer[str]
    ingested_after: str
    ingested_before: str
    limit: int
    threshold: float
    def __init__(self, query: _Optional[str] = ..., collection: _Optional[str] = ..., namespaces: _Optional[_Iterable[str]] = ..., visibility: _Optional[_Iterable[str]] = ..., tags: _Optional[_Iterable[str]] = ..., ingested_after: _Optional[str] = ..., ingested_before: _Optional[str] = ..., limit: _Optional[int] = ..., threshold: _Optional[float] = ...) -> None: ...

class SearchResult(_message.Message):
    __slots__ = ("id", "score", "content", "metadata")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    id: str
    score: float
    content: str
    metadata: _struct_pb2.Struct
    def __init__(self, id: _Optional[str] = ..., score: _Optional[float] = ..., content: _Optional[str] = ..., metadata: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class SearchResponse(_message.Message):
    __slots__ = ("results", "query", "took_ms")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    TOOK_MS_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[SearchResult]
    query: str
    took_ms: int
    def __init__(self, results: _Optional[_Iterable[_Union[SearchResult, _Mapping]]] = ..., query: _Optional[str] = ..., took_ms: _Optional[int] = ...) -> None: ...

class DependencyStatus(_message.Message):
    __slots__ = ("connected", "latency_ms", "error", "database")
    CONNECTED_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    DATABASE_FIELD_NUMBER: _ClassVar[int]
    connected: bool
    latency_ms: float
    error: _struct_pb2.Value
    database: str
    def __init__(self, connected: _Optional[bool] = ..., latency_ms: _Optional[float] = ..., error: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., database: _Optional[str] = ...) -> None: ...

class InfrastructureHealthResponse(_message.Message):
    __slots__ = ("status", "service", "timestamp", "readiness", "version", "uptime_seconds", "dependencies", "metrics")
    class DependenciesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: DependencyStatus
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[DependencyStatus, _Mapping]] = ...) -> None: ...
    class MetricsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _struct_pb2.Value
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SERVICE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    READINESS_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    UPTIME_SECONDS_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    status: str
    service: str
    timestamp: str
    readiness: bool
    version: str
    uptime_seconds: float
    dependencies: _containers.MessageMap[str, DependencyStatus]
    metrics: _containers.MessageMap[str, _struct_pb2.Value]
    def __init__(self, status: _Optional[str] = ..., service: _Optional[str] = ..., timestamp: _Optional[str] = ..., readiness: _Optional[bool] = ..., version: _Optional[str] = ..., uptime_seconds: _Optional[float] = ..., dependencies: _Optional[_Mapping[str, DependencyStatus]] = ..., metrics: _Optional[_Mapping[str, _struct_pb2.Value]] = ...) -> None: ...

class QualityMetrics(_message.Message):
    __slots__ = ("coherence", "freshness", "redundancy", "coverage")
    COHERENCE_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    REDUNDANCY_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    coherence: float
    freshness: float
    redundancy: float
    coverage: float
    def __init__(self, coherence: _Optional[float] = ..., freshness: _Optional[float] = ..., redundancy: _Optional[float] = ..., coverage: _Optional[float] = ...) -> None: ...

class CollectionHealth(_message.Message):
    __slots__ = ("name", "size", "metrics")
    NAME_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    name: str
    size: int
    metrics: QualityMetrics
    def __init__(self, name: _Optional[str] = ..., size: _Optional[int] = ..., metrics: _Optional[_Union[QualityMetrics, _Mapping]] = ...) -> None: ...

class KnowledgeHealthResponse(_message.Message):
    __slots__ = ("total_entries", "collections", "overall_health", "overall_metrics", "timestamp")
    TOTAL_ENTRIES_FIELD_NUMBER: _ClassVar[int]
    COLLECTIONS_FIELD_NUMBER: _ClassVar[int]
    OVERALL_HEALTH_FIELD_NUMBER: _ClassVar[int]
    OVERALL_METRICS_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    total_entries: int
    collections: _containers.RepeatedCompositeFieldContainer[CollectionHealth]
    overall_health: str
    overall_metrics: QualityMetrics
    timestamp: str
    def __init__(self, total_entries: _Optional[int] = ..., collections: _Optional[_Iterable[_Union[CollectionHealth, _Mapping]]] = ..., overall_health: _Optional[str] = ..., overall_metrics: _Optional[_Union[QualityMetrics, _Mapping]] = ..., timestamp: _Optional[str] = ...) -> None: ...

class DocHealthFinding(_message.Message):
    __slots__ = ("code", "severity", "message", "path", "doc_type", "line", "target")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    DOC_TYPE_FIELD_NUMBER: _ClassVar[int]
    LINE_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    code: str
    severity: DocHealthSeverity
    message: str
    path: str
    doc_type: str
    line: int
    target: str
    def __init__(self, code: _Optional[str] = ..., severity: _Optional[_Union[DocHealthSeverity, str]] = ..., message: _Optional[str] = ..., path: _Optional[str] = ..., doc_type: _Optional[str] = ..., line: _Optional[int] = ..., target: _Optional[str] = ...) -> None: ...

class DocHealthMisplacedDoc(_message.Message):
    __slots__ = ("actual_path", "expected_path", "severity", "doc_type", "message")
    ACTUAL_PATH_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_PATH_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    DOC_TYPE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    actual_path: str
    expected_path: str
    severity: DocHealthSeverity
    doc_type: str
    message: str
    def __init__(self, actual_path: _Optional[str] = ..., expected_path: _Optional[str] = ..., severity: _Optional[_Union[DocHealthSeverity, str]] = ..., doc_type: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class DocHealthMissingDoc(_message.Message):
    __slots__ = ("doc_type", "path", "severity", "completion", "required_by")
    DOC_TYPE_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    COMPLETION_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_BY_FIELD_NUMBER: _ClassVar[int]
    doc_type: str
    path: str
    severity: DocHealthSeverity
    completion: str
    required_by: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, doc_type: _Optional[str] = ..., path: _Optional[str] = ..., severity: _Optional[_Union[DocHealthSeverity, str]] = ..., completion: _Optional[str] = ..., required_by: _Optional[_Iterable[str]] = ...) -> None: ...

class DocHealthCounts(_message.Message):
    __slots__ = ("files_checked", "markdown_warnings", "markdown_failures", "local_links", "external_links", "broken_links", "external_warnings", "external_failures", "mermaid_validated", "mermaid_failures", "absolute_path_hits", "absolute_failures", "code_files_scanned", "code_refs_found", "code_refs_broken", "doc_refs_found", "doc_refs_broken", "marked_refs_found", "marked_refs_broken", "marked_refs_skipped", "marked_refs_unknown", "docs_in_manifest", "docs_not_in_manifest", "numbers_flagged")
    FILES_CHECKED_FIELD_NUMBER: _ClassVar[int]
    MARKDOWN_WARNINGS_FIELD_NUMBER: _ClassVar[int]
    MARKDOWN_FAILURES_FIELD_NUMBER: _ClassVar[int]
    LOCAL_LINKS_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_LINKS_FIELD_NUMBER: _ClassVar[int]
    BROKEN_LINKS_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_WARNINGS_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_FAILURES_FIELD_NUMBER: _ClassVar[int]
    MERMAID_VALIDATED_FIELD_NUMBER: _ClassVar[int]
    MERMAID_FAILURES_FIELD_NUMBER: _ClassVar[int]
    ABSOLUTE_PATH_HITS_FIELD_NUMBER: _ClassVar[int]
    ABSOLUTE_FAILURES_FIELD_NUMBER: _ClassVar[int]
    CODE_FILES_SCANNED_FIELD_NUMBER: _ClassVar[int]
    CODE_REFS_FOUND_FIELD_NUMBER: _ClassVar[int]
    CODE_REFS_BROKEN_FIELD_NUMBER: _ClassVar[int]
    DOC_REFS_FOUND_FIELD_NUMBER: _ClassVar[int]
    DOC_REFS_BROKEN_FIELD_NUMBER: _ClassVar[int]
    MARKED_REFS_FOUND_FIELD_NUMBER: _ClassVar[int]
    MARKED_REFS_BROKEN_FIELD_NUMBER: _ClassVar[int]
    MARKED_REFS_SKIPPED_FIELD_NUMBER: _ClassVar[int]
    MARKED_REFS_UNKNOWN_FIELD_NUMBER: _ClassVar[int]
    DOCS_IN_MANIFEST_FIELD_NUMBER: _ClassVar[int]
    DOCS_NOT_IN_MANIFEST_FIELD_NUMBER: _ClassVar[int]
    NUMBERS_FLAGGED_FIELD_NUMBER: _ClassVar[int]
    files_checked: int
    markdown_warnings: int
    markdown_failures: int
    local_links: int
    external_links: int
    broken_links: int
    external_warnings: int
    external_failures: int
    mermaid_validated: int
    mermaid_failures: int
    absolute_path_hits: int
    absolute_failures: int
    code_files_scanned: int
    code_refs_found: int
    code_refs_broken: int
    doc_refs_found: int
    doc_refs_broken: int
    marked_refs_found: int
    marked_refs_broken: int
    marked_refs_skipped: int
    marked_refs_unknown: int
    docs_in_manifest: int
    docs_not_in_manifest: int
    numbers_flagged: int
    def __init__(self, files_checked: _Optional[int] = ..., markdown_warnings: _Optional[int] = ..., markdown_failures: _Optional[int] = ..., local_links: _Optional[int] = ..., external_links: _Optional[int] = ..., broken_links: _Optional[int] = ..., external_warnings: _Optional[int] = ..., external_failures: _Optional[int] = ..., mermaid_validated: _Optional[int] = ..., mermaid_failures: _Optional[int] = ..., absolute_path_hits: _Optional[int] = ..., absolute_failures: _Optional[int] = ..., code_files_scanned: _Optional[int] = ..., code_refs_found: _Optional[int] = ..., code_refs_broken: _Optional[int] = ..., doc_refs_found: _Optional[int] = ..., doc_refs_broken: _Optional[int] = ..., marked_refs_found: _Optional[int] = ..., marked_refs_broken: _Optional[int] = ..., marked_refs_skipped: _Optional[int] = ..., marked_refs_unknown: _Optional[int] = ..., docs_in_manifest: _Optional[int] = ..., docs_not_in_manifest: _Optional[int] = ..., numbers_flagged: _Optional[int] = ...) -> None: ...

class DocHealthRequest(_message.Message):
    __slots__ = ("scenario_name", "strict_external_links", "require_all_docs_registered", "skip_external_links", "scope", "path", "checks")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    STRICT_EXTERNAL_LINKS_FIELD_NUMBER: _ClassVar[int]
    REQUIRE_ALL_DOCS_REGISTERED_FIELD_NUMBER: _ClassVar[int]
    SKIP_EXTERNAL_LINKS_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    strict_external_links: bool
    require_all_docs_registered: bool
    skip_external_links: bool
    scope: str
    path: str
    checks: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario_name: _Optional[str] = ..., strict_external_links: _Optional[bool] = ..., require_all_docs_registered: _Optional[bool] = ..., skip_external_links: _Optional[bool] = ..., scope: _Optional[str] = ..., path: _Optional[str] = ..., checks: _Optional[_Iterable[str]] = ...) -> None: ...

class DocHealthResponse(_message.Message):
    __slots__ = ("scenario_name", "source_template_id", "manifest_path", "manifest_status", "health_score", "total_docs", "misplaced_docs", "missing_docs", "extra_docs", "temporary_docs", "contract_findings", "content_findings", "reference_findings", "manifest_findings", "counts", "timestamp", "assessment")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    SOURCE_TEMPLATE_ID_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_PATH_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_STATUS_FIELD_NUMBER: _ClassVar[int]
    HEALTH_SCORE_FIELD_NUMBER: _ClassVar[int]
    TOTAL_DOCS_FIELD_NUMBER: _ClassVar[int]
    MISPLACED_DOCS_FIELD_NUMBER: _ClassVar[int]
    MISSING_DOCS_FIELD_NUMBER: _ClassVar[int]
    EXTRA_DOCS_FIELD_NUMBER: _ClassVar[int]
    TEMPORARY_DOCS_FIELD_NUMBER: _ClassVar[int]
    CONTRACT_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    COUNTS_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    source_template_id: str
    manifest_path: str
    manifest_status: str
    health_score: float
    total_docs: int
    misplaced_docs: _containers.RepeatedCompositeFieldContainer[DocHealthMisplacedDoc]
    missing_docs: _containers.RepeatedCompositeFieldContainer[DocHealthMissingDoc]
    extra_docs: _containers.RepeatedScalarFieldContainer[str]
    temporary_docs: _containers.RepeatedScalarFieldContainer[str]
    contract_findings: _containers.RepeatedCompositeFieldContainer[DocHealthFinding]
    content_findings: _containers.RepeatedCompositeFieldContainer[DocHealthFinding]
    reference_findings: _containers.RepeatedCompositeFieldContainer[DocHealthFinding]
    manifest_findings: _containers.RepeatedCompositeFieldContainer[DocHealthFinding]
    counts: DocHealthCounts
    timestamp: str
    assessment: _maturity_pb2.MaturityAssessment
    def __init__(self, scenario_name: _Optional[str] = ..., source_template_id: _Optional[str] = ..., manifest_path: _Optional[str] = ..., manifest_status: _Optional[str] = ..., health_score: _Optional[float] = ..., total_docs: _Optional[int] = ..., misplaced_docs: _Optional[_Iterable[_Union[DocHealthMisplacedDoc, _Mapping]]] = ..., missing_docs: _Optional[_Iterable[_Union[DocHealthMissingDoc, _Mapping]]] = ..., extra_docs: _Optional[_Iterable[str]] = ..., temporary_docs: _Optional[_Iterable[str]] = ..., contract_findings: _Optional[_Iterable[_Union[DocHealthFinding, _Mapping]]] = ..., content_findings: _Optional[_Iterable[_Union[DocHealthFinding, _Mapping]]] = ..., reference_findings: _Optional[_Iterable[_Union[DocHealthFinding, _Mapping]]] = ..., manifest_findings: _Optional[_Iterable[_Union[DocHealthFinding, _Mapping]]] = ..., counts: _Optional[_Union[DocHealthCounts, _Mapping]] = ..., timestamp: _Optional[str] = ..., assessment: _Optional[_Union[_maturity_pb2.MaturityAssessment, _Mapping]] = ...) -> None: ...

class ValidateMarkdownDiagramsRequest(_message.Message):
    __slots__ = ("content", "source_label")
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_LABEL_FIELD_NUMBER: _ClassVar[int]
    content: str
    source_label: str
    def __init__(self, content: _Optional[str] = ..., source_label: _Optional[str] = ...) -> None: ...

class ValidateMarkdownDiagramsResponse(_message.Message):
    __slots__ = ("findings", "engine", "unverified")
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    ENGINE_FIELD_NUMBER: _ClassVar[int]
    UNVERIFIED_FIELD_NUMBER: _ClassVar[int]
    findings: _containers.RepeatedCompositeFieldContainer[DocHealthFinding]
    engine: str
    unverified: bool
    def __init__(self, findings: _Optional[_Iterable[_Union[DocHealthFinding, _Mapping]]] = ..., engine: _Optional[str] = ..., unverified: _Optional[bool] = ...) -> None: ...
