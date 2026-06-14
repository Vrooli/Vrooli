from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TargetKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TARGET_KIND_UNSPECIFIED: _ClassVar[TargetKind]
    TARGET_KIND_PATH: _ClassVar[TargetKind]
    TARGET_KIND_SCENARIO: _ClassVar[TargetKind]
    TARGET_KIND_REPO: _ClassVar[TargetKind]
    TARGET_KIND_MODULE: _ClassVar[TargetKind]
    TARGET_KIND_PROJECT: _ClassVar[TargetKind]

class FactFamily(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FACT_FAMILY_UNSPECIFIED: _ClassVar[FactFamily]
    FACT_FAMILY_SURFACES: _ClassVar[FactFamily]
    FACT_FAMILY_PARSE_UNITS: _ClassVar[FactFamily]
    FACT_FAMILY_IMPORTS: _ClassVar[FactFamily]
    FACT_FAMILY_SYMBOLS: _ClassVar[FactFamily]
    FACT_FAMILY_REFERENCES: _ClassVar[FactFamily]
    FACT_FAMILY_CALLS: _ClassVar[FactFamily]
    FACT_FAMILY_PROTO_ADOPTION: _ClassVar[FactFamily]
    FACT_FAMILY_ENDPOINT_PROOFS: _ClassVar[FactFamily]
    FACT_FAMILY_CLI_PROOFS: _ClassVar[FactFamily]
    FACT_FAMILY_UI_WIDGET_PROOFS: _ClassVar[FactFamily]
    FACT_FAMILY_ALL: _ClassVar[FactFamily]

class EvidenceStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EVIDENCE_STATUS_UNSPECIFIED: _ClassVar[EvidenceStatus]
    EVIDENCE_STATUS_PROVEN: _ClassVar[EvidenceStatus]
    EVIDENCE_STATUS_MISSING: _ClassVar[EvidenceStatus]
    EVIDENCE_STATUS_CONTRADICTED: _ClassVar[EvidenceStatus]
    EVIDENCE_STATUS_UNSUPPORTED: _ClassVar[EvidenceStatus]
    EVIDENCE_STATUS_UNKNOWN: _ClassVar[EvidenceStatus]

class SurfaceKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SURFACE_KIND_UNSPECIFIED: _ClassVar[SurfaceKind]
    SURFACE_KIND_API: _ClassVar[SurfaceKind]
    SURFACE_KIND_CLI: _ClassVar[SurfaceKind]
    SURFACE_KIND_UI: _ClassVar[SurfaceKind]
    SURFACE_KIND_SIDECAR: _ClassVar[SurfaceKind]
    SURFACE_KIND_WORKER: _ClassVar[SurfaceKind]
    SURFACE_KIND_JOB: _ClassVar[SurfaceKind]

class SurfaceStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SURFACE_STATUS_UNSPECIFIED: _ClassVar[SurfaceStatus]
    SURFACE_STATUS_KNOWN: _ClassVar[SurfaceStatus]
    SURFACE_STATUS_MISSING: _ClassVar[SurfaceStatus]
    SURFACE_STATUS_UNSUPPORTED: _ClassVar[SurfaceStatus]
    SURFACE_STATUS_AMBIGUOUS: _ClassVar[SurfaceStatus]
    SURFACE_STATUS_UNKNOWN: _ClassVar[SurfaceStatus]
TARGET_KIND_UNSPECIFIED: TargetKind
TARGET_KIND_PATH: TargetKind
TARGET_KIND_SCENARIO: TargetKind
TARGET_KIND_REPO: TargetKind
TARGET_KIND_MODULE: TargetKind
TARGET_KIND_PROJECT: TargetKind
FACT_FAMILY_UNSPECIFIED: FactFamily
FACT_FAMILY_SURFACES: FactFamily
FACT_FAMILY_PARSE_UNITS: FactFamily
FACT_FAMILY_IMPORTS: FactFamily
FACT_FAMILY_SYMBOLS: FactFamily
FACT_FAMILY_REFERENCES: FactFamily
FACT_FAMILY_CALLS: FactFamily
FACT_FAMILY_PROTO_ADOPTION: FactFamily
FACT_FAMILY_ENDPOINT_PROOFS: FactFamily
FACT_FAMILY_CLI_PROOFS: FactFamily
FACT_FAMILY_UI_WIDGET_PROOFS: FactFamily
FACT_FAMILY_ALL: FactFamily
EVIDENCE_STATUS_UNSPECIFIED: EvidenceStatus
EVIDENCE_STATUS_PROVEN: EvidenceStatus
EVIDENCE_STATUS_MISSING: EvidenceStatus
EVIDENCE_STATUS_CONTRADICTED: EvidenceStatus
EVIDENCE_STATUS_UNSUPPORTED: EvidenceStatus
EVIDENCE_STATUS_UNKNOWN: EvidenceStatus
SURFACE_KIND_UNSPECIFIED: SurfaceKind
SURFACE_KIND_API: SurfaceKind
SURFACE_KIND_CLI: SurfaceKind
SURFACE_KIND_UI: SurfaceKind
SURFACE_KIND_SIDECAR: SurfaceKind
SURFACE_KIND_WORKER: SurfaceKind
SURFACE_KIND_JOB: SurfaceKind
SURFACE_STATUS_UNSPECIFIED: SurfaceStatus
SURFACE_STATUS_KNOWN: SurfaceStatus
SURFACE_STATUS_MISSING: SurfaceStatus
SURFACE_STATUS_UNSUPPORTED: SurfaceStatus
SURFACE_STATUS_AMBIGUOUS: SurfaceStatus
SURFACE_STATUS_UNKNOWN: SurfaceStatus

class CodeTarget(_message.Message):
    __slots__ = ("kind", "path", "scenario", "repo_root", "language_filter", "strict")
    KIND_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    REPO_ROOT_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FILTER_FIELD_NUMBER: _ClassVar[int]
    STRICT_FIELD_NUMBER: _ClassVar[int]
    kind: TargetKind
    path: str
    scenario: str
    repo_root: str
    language_filter: _containers.RepeatedScalarFieldContainer[str]
    strict: bool
    def __init__(self, kind: _Optional[_Union[TargetKind, str]] = ..., path: _Optional[str] = ..., scenario: _Optional[str] = ..., repo_root: _Optional[str] = ..., language_filter: _Optional[_Iterable[str]] = ..., strict: _Optional[bool] = ...) -> None: ...

class DescribeCodeFactsRequest(_message.Message):
    __slots__ = ("target", "include", "endpoint_ids", "command_ids", "widget_ids", "max_depth", "use_cache")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_FIELD_NUMBER: _ClassVar[int]
    ENDPOINT_IDS_FIELD_NUMBER: _ClassVar[int]
    COMMAND_IDS_FIELD_NUMBER: _ClassVar[int]
    WIDGET_IDS_FIELD_NUMBER: _ClassVar[int]
    MAX_DEPTH_FIELD_NUMBER: _ClassVar[int]
    USE_CACHE_FIELD_NUMBER: _ClassVar[int]
    target: CodeTarget
    include: _containers.RepeatedScalarFieldContainer[FactFamily]
    endpoint_ids: _containers.RepeatedScalarFieldContainer[str]
    command_ids: _containers.RepeatedScalarFieldContainer[str]
    widget_ids: _containers.RepeatedScalarFieldContainer[str]
    max_depth: int
    use_cache: bool
    def __init__(self, target: _Optional[_Union[CodeTarget, _Mapping]] = ..., include: _Optional[_Iterable[_Union[FactFamily, str]]] = ..., endpoint_ids: _Optional[_Iterable[str]] = ..., command_ids: _Optional[_Iterable[str]] = ..., widget_ids: _Optional[_Iterable[str]] = ..., max_depth: _Optional[int] = ..., use_cache: _Optional[bool] = ...) -> None: ...

class DescribeFleetImportsRequest(_message.Message):
    __slots__ = ("scenarios", "limit", "use_cache", "repo_root", "language_filter")
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    USE_CACHE_FIELD_NUMBER: _ClassVar[int]
    REPO_ROOT_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FILTER_FIELD_NUMBER: _ClassVar[int]
    scenarios: _containers.RepeatedScalarFieldContainer[str]
    limit: int
    use_cache: bool
    repo_root: str
    language_filter: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenarios: _Optional[_Iterable[str]] = ..., limit: _Optional[int] = ..., use_cache: _Optional[bool] = ..., repo_root: _Optional[str] = ..., language_filter: _Optional[_Iterable[str]] = ...) -> None: ...

class CodeFactsResult(_message.Message):
    __slots__ = ("scenario", "report", "error")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    report: CodeFactsReport
    error: str
    def __init__(self, scenario: _Optional[str] = ..., report: _Optional[_Union[CodeFactsReport, _Mapping]] = ..., error: _Optional[str] = ...) -> None: ...

class DescribeFleetImportsResponse(_message.Message):
    __slots__ = ("results",)
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[CodeFactsResult]
    def __init__(self, results: _Optional[_Iterable[_Union[CodeFactsResult, _Mapping]]] = ...) -> None: ...

class ListSurfacesRequest(_message.Message):
    __slots__ = ("target", "use_cache")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    USE_CACHE_FIELD_NUMBER: _ClassVar[int]
    target: CodeTarget
    use_cache: bool
    def __init__(self, target: _Optional[_Union[CodeTarget, _Mapping]] = ..., use_cache: _Optional[bool] = ...) -> None: ...

class CheckProtoAdoptionRequest(_message.Message):
    __slots__ = ("target", "surfaces", "use_cache")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    USE_CACHE_FIELD_NUMBER: _ClassVar[int]
    target: CodeTarget
    surfaces: _containers.RepeatedScalarFieldContainer[str]
    use_cache: bool
    def __init__(self, target: _Optional[_Union[CodeTarget, _Mapping]] = ..., surfaces: _Optional[_Iterable[str]] = ..., use_cache: _Optional[bool] = ...) -> None: ...

class CheckEndpointProofRequest(_message.Message):
    __slots__ = ("target", "endpoint_ids", "use_cache")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ENDPOINT_IDS_FIELD_NUMBER: _ClassVar[int]
    USE_CACHE_FIELD_NUMBER: _ClassVar[int]
    target: CodeTarget
    endpoint_ids: _containers.RepeatedScalarFieldContainer[str]
    use_cache: bool
    def __init__(self, target: _Optional[_Union[CodeTarget, _Mapping]] = ..., endpoint_ids: _Optional[_Iterable[str]] = ..., use_cache: _Optional[bool] = ...) -> None: ...

class GetCacheStatusRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: CodeTarget
    def __init__(self, target: _Optional[_Union[CodeTarget, _Mapping]] = ...) -> None: ...

class InspectCacheRequest(_message.Message):
    __slots__ = ("target", "cache_key")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    CACHE_KEY_FIELD_NUMBER: _ClassVar[int]
    target: CodeTarget
    cache_key: str
    def __init__(self, target: _Optional[_Union[CodeTarget, _Mapping]] = ..., cache_key: _Optional[str] = ...) -> None: ...

class ClearCacheRequest(_message.Message):
    __slots__ = ("target", "dry_run")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    target: CodeTarget
    dry_run: bool
    def __init__(self, target: _Optional[_Union[CodeTarget, _Mapping]] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class TargetContext(_message.Message):
    __slots__ = ("requested", "resolved_kind", "root_path", "scenario", "scenario_aware")
    REQUESTED_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_KIND_FIELD_NUMBER: _ClassVar[int]
    ROOT_PATH_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_AWARE_FIELD_NUMBER: _ClassVar[int]
    requested: CodeTarget
    resolved_kind: TargetKind
    root_path: str
    scenario: str
    scenario_aware: bool
    def __init__(self, requested: _Optional[_Union[CodeTarget, _Mapping]] = ..., resolved_kind: _Optional[_Union[TargetKind, str]] = ..., root_path: _Optional[str] = ..., scenario: _Optional[str] = ..., scenario_aware: _Optional[bool] = ...) -> None: ...

class SourceRange(_message.Message):
    __slots__ = ("file", "start_line", "start_column", "end_line", "end_column")
    FILE_FIELD_NUMBER: _ClassVar[int]
    START_LINE_FIELD_NUMBER: _ClassVar[int]
    START_COLUMN_FIELD_NUMBER: _ClassVar[int]
    END_LINE_FIELD_NUMBER: _ClassVar[int]
    END_COLUMN_FIELD_NUMBER: _ClassVar[int]
    file: str
    start_line: int
    start_column: int
    end_line: int
    end_column: int
    def __init__(self, file: _Optional[str] = ..., start_line: _Optional[int] = ..., start_column: _Optional[int] = ..., end_line: _Optional[int] = ..., end_column: _Optional[int] = ...) -> None: ...

class Evidence(_message.Message):
    __slots__ = ("status", "confidence", "range", "symbol", "analyzer", "message")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    RANGE_FIELD_NUMBER: _ClassVar[int]
    SYMBOL_FIELD_NUMBER: _ClassVar[int]
    ANALYZER_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    status: EvidenceStatus
    confidence: float
    range: SourceRange
    symbol: str
    analyzer: str
    message: str
    def __init__(self, status: _Optional[_Union[EvidenceStatus, str]] = ..., confidence: _Optional[float] = ..., range: _Optional[_Union[SourceRange, _Mapping]] = ..., symbol: _Optional[str] = ..., analyzer: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class ParseUnit(_message.Message):
    __slots__ = ("id", "language", "root_path", "config_path", "status", "evidence")
    ID_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    ROOT_PATH_FIELD_NUMBER: _ClassVar[int]
    CONFIG_PATH_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    language: str
    root_path: str
    config_path: str
    status: EvidenceStatus
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    def __init__(self, id: _Optional[str] = ..., language: _Optional[str] = ..., root_path: _Optional[str] = ..., config_path: _Optional[str] = ..., status: _Optional[_Union[EvidenceStatus, str]] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ...) -> None: ...

class Surface(_message.Message):
    __slots__ = ("id", "kind", "path", "status", "evidence")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: SurfaceKind
    path: str
    status: SurfaceStatus
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[_Union[SurfaceKind, str]] = ..., path: _Optional[str] = ..., status: _Optional[_Union[SurfaceStatus, str]] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ...) -> None: ...

class GenericFact(_message.Message):
    __slots__ = ("id", "family", "kind", "subject", "evidence", "attributes")
    class AttributesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    FAMILY_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTES_FIELD_NUMBER: _ClassVar[int]
    id: str
    family: FactFamily
    kind: str
    subject: str
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    attributes: _containers.ScalarMap[str, str]
    def __init__(self, id: _Optional[str] = ..., family: _Optional[_Union[FactFamily, str]] = ..., kind: _Optional[str] = ..., subject: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ..., attributes: _Optional[_Mapping[str, str]] = ...) -> None: ...

class Warning(_message.Message):
    __slots__ = ("code", "message", "status")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    status: EvidenceStatus
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., status: _Optional[_Union[EvidenceStatus, str]] = ...) -> None: ...

class CacheMetadata(_message.Message):
    __slots__ = ("cache_key", "hit", "analyzer_version", "graph_hash", "age_seconds", "state", "reason", "source_hash", "config_hash", "provider_version", "schema_version", "created_at_unix", "last_used_at_unix", "hit_count", "scope")
    CACHE_KEY_FIELD_NUMBER: _ClassVar[int]
    HIT_FIELD_NUMBER: _ClassVar[int]
    ANALYZER_VERSION_FIELD_NUMBER: _ClassVar[int]
    GRAPH_HASH_FIELD_NUMBER: _ClassVar[int]
    AGE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    SOURCE_HASH_FIELD_NUMBER: _ClassVar[int]
    CONFIG_HASH_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_VERSION_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_UNIX_FIELD_NUMBER: _ClassVar[int]
    LAST_USED_AT_UNIX_FIELD_NUMBER: _ClassVar[int]
    HIT_COUNT_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    cache_key: str
    hit: bool
    analyzer_version: str
    graph_hash: str
    age_seconds: int
    state: str
    reason: str
    source_hash: str
    config_hash: str
    provider_version: str
    schema_version: str
    created_at_unix: int
    last_used_at_unix: int
    hit_count: int
    scope: str
    def __init__(self, cache_key: _Optional[str] = ..., hit: _Optional[bool] = ..., analyzer_version: _Optional[str] = ..., graph_hash: _Optional[str] = ..., age_seconds: _Optional[int] = ..., state: _Optional[str] = ..., reason: _Optional[str] = ..., source_hash: _Optional[str] = ..., config_hash: _Optional[str] = ..., provider_version: _Optional[str] = ..., schema_version: _Optional[str] = ..., created_at_unix: _Optional[int] = ..., last_used_at_unix: _Optional[int] = ..., hit_count: _Optional[int] = ..., scope: _Optional[str] = ...) -> None: ...

class CacheStatus(_message.Message):
    __slots__ = ("target", "cache_key", "entries", "entries_metadata")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    CACHE_KEY_FIELD_NUMBER: _ClassVar[int]
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    ENTRIES_METADATA_FIELD_NUMBER: _ClassVar[int]
    target: CodeTarget
    cache_key: str
    entries: int
    entries_metadata: _containers.RepeatedCompositeFieldContainer[CacheMetadata]
    def __init__(self, target: _Optional[_Union[CodeTarget, _Mapping]] = ..., cache_key: _Optional[str] = ..., entries: _Optional[int] = ..., entries_metadata: _Optional[_Iterable[_Union[CacheMetadata, _Mapping]]] = ...) -> None: ...

class ClearCacheResponse(_message.Message):
    __slots__ = ("cache_key", "matched_entries", "cleared_entries", "dry_run")
    CACHE_KEY_FIELD_NUMBER: _ClassVar[int]
    MATCHED_ENTRIES_FIELD_NUMBER: _ClassVar[int]
    CLEARED_ENTRIES_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    cache_key: str
    matched_entries: int
    cleared_entries: int
    dry_run: bool
    def __init__(self, cache_key: _Optional[str] = ..., matched_entries: _Optional[int] = ..., cleared_entries: _Optional[int] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class CodeFactsReport(_message.Message):
    __slots__ = ("target", "parse_units", "surfaces", "facts", "evidence", "warnings", "cache")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    PARSE_UNITS_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    FACTS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    CACHE_FIELD_NUMBER: _ClassVar[int]
    target: TargetContext
    parse_units: _containers.RepeatedCompositeFieldContainer[ParseUnit]
    surfaces: _containers.RepeatedCompositeFieldContainer[Surface]
    facts: _containers.RepeatedCompositeFieldContainer[GenericFact]
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    warnings: _containers.RepeatedCompositeFieldContainer[Warning]
    cache: CacheMetadata
    def __init__(self, target: _Optional[_Union[TargetContext, _Mapping]] = ..., parse_units: _Optional[_Iterable[_Union[ParseUnit, _Mapping]]] = ..., surfaces: _Optional[_Iterable[_Union[Surface, _Mapping]]] = ..., facts: _Optional[_Iterable[_Union[GenericFact, _Mapping]]] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ..., warnings: _Optional[_Iterable[_Union[Warning, _Mapping]]] = ..., cache: _Optional[_Union[CacheMetadata, _Mapping]] = ...) -> None: ...

class ListSurfacesResponse(_message.Message):
    __slots__ = ("target", "surfaces", "warnings", "cache")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    CACHE_FIELD_NUMBER: _ClassVar[int]
    target: TargetContext
    surfaces: _containers.RepeatedCompositeFieldContainer[Surface]
    warnings: _containers.RepeatedCompositeFieldContainer[Warning]
    cache: CacheMetadata
    def __init__(self, target: _Optional[_Union[TargetContext, _Mapping]] = ..., surfaces: _Optional[_Iterable[_Union[Surface, _Mapping]]] = ..., warnings: _Optional[_Iterable[_Union[Warning, _Mapping]]] = ..., cache: _Optional[_Union[CacheMetadata, _Mapping]] = ...) -> None: ...

class ProofReport(_message.Message):
    __slots__ = ("target", "family", "facts", "evidence", "warnings", "cache")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    FAMILY_FIELD_NUMBER: _ClassVar[int]
    FACTS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    CACHE_FIELD_NUMBER: _ClassVar[int]
    target: TargetContext
    family: FactFamily
    facts: _containers.RepeatedCompositeFieldContainer[GenericFact]
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    warnings: _containers.RepeatedCompositeFieldContainer[Warning]
    cache: CacheMetadata
    def __init__(self, target: _Optional[_Union[TargetContext, _Mapping]] = ..., family: _Optional[_Union[FactFamily, str]] = ..., facts: _Optional[_Iterable[_Union[GenericFact, _Mapping]]] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ..., warnings: _Optional[_Iterable[_Union[Warning, _Mapping]]] = ..., cache: _Optional[_Union[CacheMetadata, _Mapping]] = ...) -> None: ...
