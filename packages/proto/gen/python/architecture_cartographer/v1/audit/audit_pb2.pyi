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
AUDIT_OUTCOME_UNSPECIFIED: AuditOutcome
AUDIT_OUTCOME_CLEAN: AuditOutcome
AUDIT_OUTCOME_FINDINGS: AuditOutcome
AUDIT_OUTCOME_TOOL_ERROR: AuditOutcome

class ConflictSummary(_message.Message):
    __slots__ = ("id", "detector", "type", "subtype", "severity", "locations", "domains", "headline")
    ID_FIELD_NUMBER: _ClassVar[int]
    DETECTOR_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    SUBTYPE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    LOCATIONS_FIELD_NUMBER: _ClassVar[int]
    DOMAINS_FIELD_NUMBER: _ClassVar[int]
    HEADLINE_FIELD_NUMBER: _ClassVar[int]
    id: str
    detector: str
    type: str
    subtype: str
    severity: _conflicts_pb2.Severity
    locations: _containers.RepeatedScalarFieldContainer[str]
    domains: _containers.RepeatedScalarFieldContainer[str]
    headline: str
    def __init__(self, id: _Optional[str] = ..., detector: _Optional[str] = ..., type: _Optional[str] = ..., subtype: _Optional[str] = ..., severity: _Optional[_Union[_conflicts_pb2.Severity, str]] = ..., locations: _Optional[_Iterable[str]] = ..., domains: _Optional[_Iterable[str]] = ..., headline: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("scenario", "fail_on", "include_types", "exclude_types")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    FAIL_ON_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_TYPES_FIELD_NUMBER: _ClassVar[int]
    EXCLUDE_TYPES_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    fail_on: _conflicts_pb2.Severity
    include_types: _containers.RepeatedScalarFieldContainer[str]
    exclude_types: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., fail_on: _Optional[_Union[_conflicts_pb2.Severity, str]] = ..., include_types: _Optional[_Iterable[str]] = ..., exclude_types: _Optional[_Iterable[str]] = ...) -> None: ...

class AuditRunResponse(_message.Message):
    __slots__ = ("scenario", "outcome", "error", "total_findings", "by_severity", "by_type", "findings", "domains", "graph", "duration")
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
    def __init__(self, scenario: _Optional[str] = ..., outcome: _Optional[_Union[AuditOutcome, str]] = ..., error: _Optional[str] = ..., total_findings: _Optional[int] = ..., by_severity: _Optional[_Mapping[str, int]] = ..., by_type: _Optional[_Mapping[str, int]] = ..., findings: _Optional[_Iterable[_Union[ConflictSummary, _Mapping]]] = ..., domains: _Optional[_Union[DerivedDomainSummary, _Mapping]] = ..., graph: _Optional[_Union[GraphSummary, _Mapping]] = ..., duration: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ...) -> None: ...
