import datetime

from architecture_cartographer.v1.conflicts import conflicts_pb2 as _conflicts_pb2
from google.protobuf import duration_pb2 as _duration_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AuditOutcome(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    AUDIT_OUTCOME_UNSPECIFIED: _ClassVar[AuditOutcome]
    AUDIT_OUTCOME_CLEAN: _ClassVar[AuditOutcome]
    AUDIT_OUTCOME_FINDINGS: _ClassVar[AuditOutcome]
    AUDIT_OUTCOME_TOOL_ERROR: _ClassVar[AuditOutcome]

class AuthorityConfidence(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    AUTHORITY_CONFIDENCE_UNSPECIFIED: _ClassVar[AuthorityConfidence]
    AUTHORITY_CONFIDENCE_LOW: _ClassVar[AuthorityConfidence]
    AUTHORITY_CONFIDENCE_MEDIUM: _ClassVar[AuthorityConfidence]
    AUTHORITY_CONFIDENCE_HIGH: _ClassVar[AuthorityConfidence]

class SnapshotFreshness(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SNAPSHOT_FRESHNESS_UNSPECIFIED: _ClassVar[SnapshotFreshness]
    SNAPSHOT_FRESHNESS_CACHED: _ClassVar[SnapshotFreshness]
    SNAPSHOT_FRESHNESS_RE_EXTRACTED: _ClassVar[SnapshotFreshness]
    SNAPSHOT_FRESHNESS_FRESH: _ClassVar[SnapshotFreshness]
AUDIT_OUTCOME_UNSPECIFIED: AuditOutcome
AUDIT_OUTCOME_CLEAN: AuditOutcome
AUDIT_OUTCOME_FINDINGS: AuditOutcome
AUDIT_OUTCOME_TOOL_ERROR: AuditOutcome
AUTHORITY_CONFIDENCE_UNSPECIFIED: AuthorityConfidence
AUTHORITY_CONFIDENCE_LOW: AuthorityConfidence
AUTHORITY_CONFIDENCE_MEDIUM: AuthorityConfidence
AUTHORITY_CONFIDENCE_HIGH: AuthorityConfidence
SNAPSHOT_FRESHNESS_UNSPECIFIED: SnapshotFreshness
SNAPSHOT_FRESHNESS_CACHED: SnapshotFreshness
SNAPSHOT_FRESHNESS_RE_EXTRACTED: SnapshotFreshness
SNAPSHOT_FRESHNESS_FRESH: SnapshotFreshness

class ConflictSummary(_message.Message):
    __slots__ = ("id", "detector", "type", "subtype", "severity", "locations", "domains", "headline", "stable_id", "instance_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    DETECTOR_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    SUBTYPE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    LOCATIONS_FIELD_NUMBER: _ClassVar[int]
    DOMAINS_FIELD_NUMBER: _ClassVar[int]
    HEADLINE_FIELD_NUMBER: _ClassVar[int]
    STABLE_ID_FIELD_NUMBER: _ClassVar[int]
    INSTANCE_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    detector: str
    type: str
    subtype: str
    severity: _conflicts_pb2.Severity
    locations: _containers.RepeatedScalarFieldContainer[str]
    domains: _containers.RepeatedScalarFieldContainer[str]
    headline: str
    stable_id: str
    instance_id: str
    def __init__(self, id: _Optional[str] = ..., detector: _Optional[str] = ..., type: _Optional[str] = ..., subtype: _Optional[str] = ..., severity: _Optional[_Union[_conflicts_pb2.Severity, str]] = ..., locations: _Optional[_Iterable[str]] = ..., domains: _Optional[_Iterable[str]] = ..., headline: _Optional[str] = ..., stable_id: _Optional[str] = ..., instance_id: _Optional[str] = ...) -> None: ...

class DerivedDomainSummary(_message.Message):
    __slots__ = ("authority", "confidence", "domain_count")
    AUTHORITY_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_COUNT_FIELD_NUMBER: _ClassVar[int]
    authority: str
    confidence: str
    domain_count: int
    def __init__(self, authority: _Optional[str] = ..., confidence: _Optional[str] = ..., domain_count: _Optional[int] = ...) -> None: ...

class GraphSummary(_message.Message):
    __slots__ = ("snapshot_id", "file_count", "package_count", "import_edge_count")
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    FILE_COUNT_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    IMPORT_EDGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    snapshot_id: str
    file_count: int
    package_count: int
    import_edge_count: int
    def __init__(self, snapshot_id: _Optional[str] = ..., file_count: _Optional[int] = ..., package_count: _Optional[int] = ..., import_edge_count: _Optional[int] = ...) -> None: ...

class AuditRunRequest(_message.Message):
    __slots__ = ("scenario", "fail_on", "include_types", "exclude_types", "allow_low_authority")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    FAIL_ON_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_TYPES_FIELD_NUMBER: _ClassVar[int]
    EXCLUDE_TYPES_FIELD_NUMBER: _ClassVar[int]
    ALLOW_LOW_AUTHORITY_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    fail_on: _conflicts_pb2.Severity
    include_types: _containers.RepeatedScalarFieldContainer[str]
    exclude_types: _containers.RepeatedScalarFieldContainer[str]
    allow_low_authority: bool
    def __init__(self, scenario: _Optional[str] = ..., fail_on: _Optional[_Union[_conflicts_pb2.Severity, str]] = ..., include_types: _Optional[_Iterable[str]] = ..., exclude_types: _Optional[_Iterable[str]] = ..., allow_low_authority: _Optional[bool] = ...) -> None: ...

class AuditRunResponse(_message.Message):
    __slots__ = ("scenario", "outcome", "error", "total_findings", "by_severity", "by_type", "findings", "domains", "graph", "duration", "suppressed_findings", "by_domain", "snapshot_freshness", "authority_confidence", "outcome_reason")
    class BySeverityEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    class ByTypeEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    class ByDomainEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    BY_SEVERITY_FIELD_NUMBER: _ClassVar[int]
    BY_TYPE_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    DOMAINS_FIELD_NUMBER: _ClassVar[int]
    GRAPH_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    SUPPRESSED_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    BY_DOMAIN_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    AUTHORITY_CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_REASON_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    outcome: AuditOutcome
    error: str
    total_findings: int
    by_severity: _containers.ScalarMap[str, int]
    by_type: _containers.ScalarMap[str, int]
    findings: _containers.RepeatedCompositeFieldContainer[ConflictSummary]
    domains: DerivedDomainSummary
    graph: GraphSummary
    duration: _duration_pb2.Duration
    suppressed_findings: int
    by_domain: _containers.ScalarMap[str, int]
    snapshot_freshness: SnapshotFreshness
    authority_confidence: AuthorityConfidence
    outcome_reason: str
    def __init__(self, scenario: _Optional[str] = ..., outcome: _Optional[_Union[AuditOutcome, str]] = ..., error: _Optional[str] = ..., total_findings: _Optional[int] = ..., by_severity: _Optional[_Mapping[str, int]] = ..., by_type: _Optional[_Mapping[str, int]] = ..., findings: _Optional[_Iterable[_Union[ConflictSummary, _Mapping]]] = ..., domains: _Optional[_Union[DerivedDomainSummary, _Mapping]] = ..., graph: _Optional[_Union[GraphSummary, _Mapping]] = ..., duration: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., suppressed_findings: _Optional[int] = ..., by_domain: _Optional[_Mapping[str, int]] = ..., snapshot_freshness: _Optional[_Union[SnapshotFreshness, str]] = ..., authority_confidence: _Optional[_Union[AuthorityConfidence, str]] = ..., outcome_reason: _Optional[str] = ...) -> None: ...

class AuditRunAllRequest(_message.Message):
    __slots__ = ("fail_on", "include_types", "exclude_types", "include_scenarios", "exclude_scenarios", "allow_low_authority", "concurrency")
    FAIL_ON_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_TYPES_FIELD_NUMBER: _ClassVar[int]
    EXCLUDE_TYPES_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    EXCLUDE_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    ALLOW_LOW_AUTHORITY_FIELD_NUMBER: _ClassVar[int]
    CONCURRENCY_FIELD_NUMBER: _ClassVar[int]
    fail_on: _conflicts_pb2.Severity
    include_types: _containers.RepeatedScalarFieldContainer[str]
    exclude_types: _containers.RepeatedScalarFieldContainer[str]
    include_scenarios: _containers.RepeatedScalarFieldContainer[str]
    exclude_scenarios: _containers.RepeatedScalarFieldContainer[str]
    allow_low_authority: bool
    concurrency: int
    def __init__(self, fail_on: _Optional[_Union[_conflicts_pb2.Severity, str]] = ..., include_types: _Optional[_Iterable[str]] = ..., exclude_types: _Optional[_Iterable[str]] = ..., include_scenarios: _Optional[_Iterable[str]] = ..., exclude_scenarios: _Optional[_Iterable[str]] = ..., allow_low_authority: _Optional[bool] = ..., concurrency: _Optional[int] = ...) -> None: ...

class AuditRunAllResponse(_message.Message):
    __slots__ = ("reports", "total_scenarios", "total_findings", "total_suppressed", "by_severity", "by_outcome", "duration")
    class BySeverityEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    class ByOutcomeEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    REPORTS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_SUPPRESSED_FIELD_NUMBER: _ClassVar[int]
    BY_SEVERITY_FIELD_NUMBER: _ClassVar[int]
    BY_OUTCOME_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    reports: _containers.RepeatedCompositeFieldContainer[AuditRunResponse]
    total_scenarios: int
    total_findings: int
    total_suppressed: int
    by_severity: _containers.ScalarMap[str, int]
    by_outcome: _containers.ScalarMap[str, int]
    duration: _duration_pb2.Duration
    def __init__(self, reports: _Optional[_Iterable[_Union[AuditRunResponse, _Mapping]]] = ..., total_scenarios: _Optional[int] = ..., total_findings: _Optional[int] = ..., total_suppressed: _Optional[int] = ..., by_severity: _Optional[_Mapping[str, int]] = ..., by_outcome: _Optional[_Mapping[str, int]] = ..., duration: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ...) -> None: ...
