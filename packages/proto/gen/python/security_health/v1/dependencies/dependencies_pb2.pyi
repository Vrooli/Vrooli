from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Mode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    MODE_UNSPECIFIED: _ClassVar[Mode]
    MODE_AI: _ClassVar[Mode]
    MODE_TEXT: _ClassVar[Mode]

class Ecosystem(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ECOSYSTEM_UNSPECIFIED: _ClassVar[Ecosystem]
    ECOSYSTEM_GO: _ClassVar[Ecosystem]
    ECOSYSTEM_NPM: _ClassVar[Ecosystem]
    ECOSYSTEM_YARN: _ClassVar[Ecosystem]
    ECOSYSTEM_BUN: _ClassVar[Ecosystem]
    ECOSYSTEM_PYTHON: _ClassVar[Ecosystem]
    ECOSYSTEM_RUST: _ClassVar[Ecosystem]
    ECOSYSTEM_C: _ClassVar[Ecosystem]
    ECOSYSTEM_CPP: _ClassVar[Ecosystem]

class VulnerabilitySource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VULNERABILITY_SOURCE_UNSPECIFIED: _ClassVar[VulnerabilitySource]
    VULNERABILITY_SOURCE_OSV: _ClassVar[VulnerabilitySource]
    VULNERABILITY_SOURCE_GOVULNCHECK: _ClassVar[VulnerabilitySource]
    VULNERABILITY_SOURCE_PNPM_AUDIT: _ClassVar[VulnerabilitySource]

class Reachability(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    REACHABILITY_UNSPECIFIED: _ClassVar[Reachability]
    REACHABILITY_UNKNOWN: _ClassVar[Reachability]
    REACHABILITY_LOCKFILE_AFFECTED: _ClassVar[Reachability]
    REACHABILITY_REACHABLE: _ClassVar[Reachability]

class EvidenceConfidence(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EVIDENCE_CONFIDENCE_UNSPECIFIED: _ClassVar[EvidenceConfidence]
    EVIDENCE_CONFIDENCE_DEGRADED: _ClassVar[EvidenceConfidence]
    EVIDENCE_CONFIDENCE_ADVISORY: _ClassVar[EvidenceConfidence]
    EVIDENCE_CONFIDENCE_GATING: _ClassVar[EvidenceConfidence]
MODE_UNSPECIFIED: Mode
MODE_AI: Mode
MODE_TEXT: Mode
ECOSYSTEM_UNSPECIFIED: Ecosystem
ECOSYSTEM_GO: Ecosystem
ECOSYSTEM_NPM: Ecosystem
ECOSYSTEM_YARN: Ecosystem
ECOSYSTEM_BUN: Ecosystem
ECOSYSTEM_PYTHON: Ecosystem
ECOSYSTEM_RUST: Ecosystem
ECOSYSTEM_C: Ecosystem
ECOSYSTEM_CPP: Ecosystem
VULNERABILITY_SOURCE_UNSPECIFIED: VulnerabilitySource
VULNERABILITY_SOURCE_OSV: VulnerabilitySource
VULNERABILITY_SOURCE_GOVULNCHECK: VulnerabilitySource
VULNERABILITY_SOURCE_PNPM_AUDIT: VulnerabilitySource
REACHABILITY_UNSPECIFIED: Reachability
REACHABILITY_UNKNOWN: Reachability
REACHABILITY_LOCKFILE_AFFECTED: Reachability
REACHABILITY_REACHABLE: Reachability
EVIDENCE_CONFIDENCE_UNSPECIFIED: EvidenceConfidence
EVIDENCE_CONFIDENCE_DEGRADED: EvidenceConfidence
EVIDENCE_CONFIDENCE_ADVISORY: EvidenceConfidence
EVIDENCE_CONFIDENCE_GATING: EvidenceConfidence

class DependencyRecord(_message.Message):
    __slots__ = ("scenario", "ecosystem", "name", "version", "source_file", "vuln_ids", "max_severity", "last_seen")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ECOSYSTEM_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FILE_FIELD_NUMBER: _ClassVar[int]
    VULN_IDS_FIELD_NUMBER: _ClassVar[int]
    MAX_SEVERITY_FIELD_NUMBER: _ClassVar[int]
    LAST_SEEN_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    ecosystem: Ecosystem
    name: str
    version: str
    source_file: str
    vuln_ids: _containers.RepeatedScalarFieldContainer[str]
    max_severity: str
    last_seen: str
    def __init__(self, scenario: _Optional[str] = ..., ecosystem: _Optional[_Union[Ecosystem, str]] = ..., name: _Optional[str] = ..., version: _Optional[str] = ..., source_file: _Optional[str] = ..., vuln_ids: _Optional[_Iterable[str]] = ..., max_severity: _Optional[str] = ..., last_seen: _Optional[str] = ...) -> None: ...

class AffectedVersionRange(_message.Message):
    __slots__ = ("range", "introduced", "fixed", "last_affected")
    RANGE_FIELD_NUMBER: _ClassVar[int]
    INTRODUCED_FIELD_NUMBER: _ClassVar[int]
    FIXED_FIELD_NUMBER: _ClassVar[int]
    LAST_AFFECTED_FIELD_NUMBER: _ClassVar[int]
    range: str
    introduced: str
    fixed: str
    last_affected: str
    def __init__(self, range: _Optional[str] = ..., introduced: _Optional[str] = ..., fixed: _Optional[str] = ..., last_affected: _Optional[str] = ...) -> None: ...

class FixedVersionRange(_message.Message):
    __slots__ = ("range", "version")
    RANGE_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    range: str
    version: str
    def __init__(self, range: _Optional[str] = ..., version: _Optional[str] = ...) -> None: ...

class VulnerabilityRecord(_message.Message):
    __slots__ = ("vulnerability_id", "aliases", "ecosystem", "name", "version", "affected_ranges", "fixed_ranges", "severity", "normalized_severity", "advisory_url", "summary", "details", "source", "reachability", "confidence", "production", "dev_only", "first_seen", "last_seen", "scenarios", "source_files", "remediation")
    VULNERABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    ALIASES_FIELD_NUMBER: _ClassVar[int]
    ECOSYSTEM_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    AFFECTED_RANGES_FIELD_NUMBER: _ClassVar[int]
    FIXED_RANGES_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    NORMALIZED_SEVERITY_FIELD_NUMBER: _ClassVar[int]
    ADVISORY_URL_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    REACHABILITY_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    PRODUCTION_FIELD_NUMBER: _ClassVar[int]
    DEV_ONLY_FIELD_NUMBER: _ClassVar[int]
    FIRST_SEEN_FIELD_NUMBER: _ClassVar[int]
    LAST_SEEN_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FILES_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    vulnerability_id: str
    aliases: _containers.RepeatedScalarFieldContainer[str]
    ecosystem: Ecosystem
    name: str
    version: str
    affected_ranges: _containers.RepeatedCompositeFieldContainer[AffectedVersionRange]
    fixed_ranges: _containers.RepeatedCompositeFieldContainer[FixedVersionRange]
    severity: str
    normalized_severity: str
    advisory_url: str
    summary: str
    details: str
    source: VulnerabilitySource
    reachability: Reachability
    confidence: EvidenceConfidence
    production: bool
    dev_only: bool
    first_seen: str
    last_seen: str
    scenarios: _containers.RepeatedScalarFieldContainer[str]
    source_files: _containers.RepeatedScalarFieldContainer[str]
    remediation: str
    def __init__(self, vulnerability_id: _Optional[str] = ..., aliases: _Optional[_Iterable[str]] = ..., ecosystem: _Optional[_Union[Ecosystem, str]] = ..., name: _Optional[str] = ..., version: _Optional[str] = ..., affected_ranges: _Optional[_Iterable[_Union[AffectedVersionRange, _Mapping]]] = ..., fixed_ranges: _Optional[_Iterable[_Union[FixedVersionRange, _Mapping]]] = ..., severity: _Optional[str] = ..., normalized_severity: _Optional[str] = ..., advisory_url: _Optional[str] = ..., summary: _Optional[str] = ..., details: _Optional[str] = ..., source: _Optional[_Union[VulnerabilitySource, str]] = ..., reachability: _Optional[_Union[Reachability, str]] = ..., confidence: _Optional[_Union[EvidenceConfidence, str]] = ..., production: _Optional[bool] = ..., dev_only: _Optional[bool] = ..., first_seen: _Optional[str] = ..., last_seen: _Optional[str] = ..., scenarios: _Optional[_Iterable[str]] = ..., source_files: _Optional[_Iterable[str]] = ..., remediation: _Optional[str] = ...) -> None: ...

class SearchRequest(_message.Message):
    __slots__ = ("query", "limit", "mode", "ecosystem", "vulnerable_only", "name_glob")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    ECOSYSTEM_FIELD_NUMBER: _ClassVar[int]
    VULNERABLE_ONLY_FIELD_NUMBER: _ClassVar[int]
    NAME_GLOB_FIELD_NUMBER: _ClassVar[int]
    query: str
    limit: int
    mode: Mode
    ecosystem: Ecosystem
    vulnerable_only: bool
    name_glob: str
    def __init__(self, query: _Optional[str] = ..., limit: _Optional[int] = ..., mode: _Optional[_Union[Mode, str]] = ..., ecosystem: _Optional[_Union[Ecosystem, str]] = ..., vulnerable_only: _Optional[bool] = ..., name_glob: _Optional[str] = ...) -> None: ...

class SearchResult(_message.Message):
    __slots__ = ("record", "score")
    RECORD_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    record: DependencyRecord
    score: float
    def __init__(self, record: _Optional[_Union[DependencyRecord, _Mapping]] = ..., score: _Optional[float] = ...) -> None: ...

class SearchResponse(_message.Message):
    __slots__ = ("results", "mode_used")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    MODE_USED_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[SearchResult]
    mode_used: Mode
    def __init__(self, results: _Optional[_Iterable[_Union[SearchResult, _Mapping]]] = ..., mode_used: _Optional[_Union[Mode, str]] = ...) -> None: ...

class ListVulnerabilitiesRequest(_message.Message):
    __slots__ = ("ecosystem", "package_name", "scenario", "vulnerability_id", "minimum_confidence", "limit")
    ECOSYSTEM_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_NAME_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    VULNERABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    ecosystem: Ecosystem
    package_name: str
    scenario: str
    vulnerability_id: str
    minimum_confidence: EvidenceConfidence
    limit: int
    def __init__(self, ecosystem: _Optional[_Union[Ecosystem, str]] = ..., package_name: _Optional[str] = ..., scenario: _Optional[str] = ..., vulnerability_id: _Optional[str] = ..., minimum_confidence: _Optional[_Union[EvidenceConfidence, str]] = ..., limit: _Optional[int] = ...) -> None: ...

class ListVulnerabilitiesResponse(_message.Message):
    __slots__ = ("vulnerabilities", "total")
    VULNERABILITIES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    vulnerabilities: _containers.RepeatedCompositeFieldContainer[VulnerabilityRecord]
    total: int
    def __init__(self, vulnerabilities: _Optional[_Iterable[_Union[VulnerabilityRecord, _Mapping]]] = ..., total: _Optional[int] = ...) -> None: ...

class ExplainVulnerabilityRequest(_message.Message):
    __slots__ = ("vulnerability_id", "ecosystem", "package_name")
    VULNERABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    ECOSYSTEM_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_NAME_FIELD_NUMBER: _ClassVar[int]
    vulnerability_id: str
    ecosystem: Ecosystem
    package_name: str
    def __init__(self, vulnerability_id: _Optional[str] = ..., ecosystem: _Optional[_Union[Ecosystem, str]] = ..., package_name: _Optional[str] = ...) -> None: ...

class ExplainVulnerabilityResponse(_message.Message):
    __slots__ = ("vulnerability", "found")
    VULNERABILITY_FIELD_NUMBER: _ClassVar[int]
    FOUND_FIELD_NUMBER: _ClassVar[int]
    vulnerability: VulnerabilityRecord
    found: bool
    def __init__(self, vulnerability: _Optional[_Union[VulnerabilityRecord, _Mapping]] = ..., found: _Optional[bool] = ...) -> None: ...

class StatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class StatusResponse(_message.Message):
    __slots__ = ("available", "ollama", "qdrant", "indexed_count", "vulnerable_count", "last_reconcile_at", "last_reconcile_outcome", "indexed_vectors", "expected_vectors", "index_ready")
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    OLLAMA_FIELD_NUMBER: _ClassVar[int]
    QDRANT_FIELD_NUMBER: _ClassVar[int]
    INDEXED_COUNT_FIELD_NUMBER: _ClassVar[int]
    VULNERABLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    LAST_RECONCILE_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_RECONCILE_OUTCOME_FIELD_NUMBER: _ClassVar[int]
    INDEXED_VECTORS_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_VECTORS_FIELD_NUMBER: _ClassVar[int]
    INDEX_READY_FIELD_NUMBER: _ClassVar[int]
    available: bool
    ollama: bool
    qdrant: bool
    indexed_count: int
    vulnerable_count: int
    last_reconcile_at: str
    last_reconcile_outcome: str
    indexed_vectors: int
    expected_vectors: int
    index_ready: bool
    def __init__(self, available: _Optional[bool] = ..., ollama: _Optional[bool] = ..., qdrant: _Optional[bool] = ..., indexed_count: _Optional[int] = ..., vulnerable_count: _Optional[int] = ..., last_reconcile_at: _Optional[str] = ..., last_reconcile_outcome: _Optional[str] = ..., indexed_vectors: _Optional[int] = ..., expected_vectors: _Optional[int] = ..., index_ready: _Optional[bool] = ...) -> None: ...
