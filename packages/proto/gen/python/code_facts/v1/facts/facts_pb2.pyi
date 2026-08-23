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
    TARGET_KIND_CONTROL_PLANE: _ClassVar[TargetKind]
    TARGET_KIND_PACKAGE: _ClassVar[TargetKind]
    TARGET_KIND_RESOURCE: _ClassVar[TargetKind]
    TARGET_KIND_TOOL: _ClassVar[TargetKind]
    TARGET_KIND_SAFEGUARD: _ClassVar[TargetKind]
    TARGET_KIND_DOCS: _ClassVar[TargetKind]
    TARGET_KIND_TEAM: _ClassVar[TargetKind]

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
    FACT_FAMILY_FILE_DOMAIN: _ClassVar[FactFamily]
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
    SURFACE_KIND_RUNTIME: _ClassVar[SurfaceKind]

class SurfaceStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SURFACE_STATUS_UNSPECIFIED: _ClassVar[SurfaceStatus]
    SURFACE_STATUS_KNOWN: _ClassVar[SurfaceStatus]
    SURFACE_STATUS_MISSING: _ClassVar[SurfaceStatus]
    SURFACE_STATUS_UNSUPPORTED: _ClassVar[SurfaceStatus]
    SURFACE_STATUS_AMBIGUOUS: _ClassVar[SurfaceStatus]
    SURFACE_STATUS_UNKNOWN: _ClassVar[SurfaceStatus]

class IndexJobState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    INDEX_JOB_STATE_UNSPECIFIED: _ClassVar[IndexJobState]
    INDEX_JOB_STATE_QUEUED: _ClassVar[IndexJobState]
    INDEX_JOB_STATE_RUNNING: _ClassVar[IndexJobState]
    INDEX_JOB_STATE_CANCELLATION_REQUESTED: _ClassVar[IndexJobState]
    INDEX_JOB_STATE_SUCCEEDED: _ClassVar[IndexJobState]
    INDEX_JOB_STATE_FAILED: _ClassVar[IndexJobState]
    INDEX_JOB_STATE_CANCELLED: _ClassVar[IndexJobState]
    INDEX_JOB_STATE_INTERRUPTED: _ClassVar[IndexJobState]
TARGET_KIND_UNSPECIFIED: TargetKind
TARGET_KIND_PATH: TargetKind
TARGET_KIND_SCENARIO: TargetKind
TARGET_KIND_REPO: TargetKind
TARGET_KIND_MODULE: TargetKind
TARGET_KIND_PROJECT: TargetKind
TARGET_KIND_CONTROL_PLANE: TargetKind
TARGET_KIND_PACKAGE: TargetKind
TARGET_KIND_RESOURCE: TargetKind
TARGET_KIND_TOOL: TargetKind
TARGET_KIND_SAFEGUARD: TargetKind
TARGET_KIND_DOCS: TargetKind
TARGET_KIND_TEAM: TargetKind
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
FACT_FAMILY_FILE_DOMAIN: FactFamily
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
SURFACE_KIND_RUNTIME: SurfaceKind
SURFACE_STATUS_UNSPECIFIED: SurfaceStatus
SURFACE_STATUS_KNOWN: SurfaceStatus
SURFACE_STATUS_MISSING: SurfaceStatus
SURFACE_STATUS_UNSUPPORTED: SurfaceStatus
SURFACE_STATUS_AMBIGUOUS: SurfaceStatus
SURFACE_STATUS_UNKNOWN: SurfaceStatus
INDEX_JOB_STATE_UNSPECIFIED: IndexJobState
INDEX_JOB_STATE_QUEUED: IndexJobState
INDEX_JOB_STATE_RUNNING: IndexJobState
INDEX_JOB_STATE_CANCELLATION_REQUESTED: IndexJobState
INDEX_JOB_STATE_SUCCEEDED: IndexJobState
INDEX_JOB_STATE_FAILED: IndexJobState
INDEX_JOB_STATE_CANCELLED: IndexJobState
INDEX_JOB_STATE_INTERRUPTED: IndexJobState

class CodeTarget(_message.Message):
    __slots__ = ("kind", "path", "scenario", "repo_root", "language_filter", "strict", "package_name")
    KIND_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    REPO_ROOT_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FILTER_FIELD_NUMBER: _ClassVar[int]
    STRICT_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_NAME_FIELD_NUMBER: _ClassVar[int]
    kind: TargetKind
    path: str
    scenario: str
    repo_root: str
    language_filter: _containers.RepeatedScalarFieldContainer[str]
    strict: bool
    package_name: str
    def __init__(self, kind: _Optional[_Union[TargetKind, str]] = ..., path: _Optional[str] = ..., scenario: _Optional[str] = ..., repo_root: _Optional[str] = ..., language_filter: _Optional[_Iterable[str]] = ..., strict: _Optional[bool] = ..., package_name: _Optional[str] = ...) -> None: ...

class DescribeCodeFactsRequest(_message.Message):
    __slots__ = ("target", "include", "endpoint_ids", "command_ids", "widget_ids", "max_depth", "use_cache", "page_size", "page_token")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_FIELD_NUMBER: _ClassVar[int]
    ENDPOINT_IDS_FIELD_NUMBER: _ClassVar[int]
    COMMAND_IDS_FIELD_NUMBER: _ClassVar[int]
    WIDGET_IDS_FIELD_NUMBER: _ClassVar[int]
    MAX_DEPTH_FIELD_NUMBER: _ClassVar[int]
    USE_CACHE_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    target: CodeTarget
    include: _containers.RepeatedScalarFieldContainer[FactFamily]
    endpoint_ids: _containers.RepeatedScalarFieldContainer[str]
    command_ids: _containers.RepeatedScalarFieldContainer[str]
    widget_ids: _containers.RepeatedScalarFieldContainer[str]
    max_depth: int
    use_cache: bool
    page_size: int
    page_token: str
    def __init__(self, target: _Optional[_Union[CodeTarget, _Mapping]] = ..., include: _Optional[_Iterable[_Union[FactFamily, str]]] = ..., endpoint_ids: _Optional[_Iterable[str]] = ..., command_ids: _Optional[_Iterable[str]] = ..., widget_ids: _Optional[_Iterable[str]] = ..., max_depth: _Optional[int] = ..., use_cache: _Optional[bool] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class SearchRequest(_message.Message):
    __slots__ = ("query", "limit", "target", "families", "expand_edges", "roles", "languages", "scope", "budget_ms", "lexical_budget_ms", "semantic_budget_ms", "graph_budget_ms")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    FAMILIES_FIELD_NUMBER: _ClassVar[int]
    EXPAND_EDGES_FIELD_NUMBER: _ClassVar[int]
    ROLES_FIELD_NUMBER: _ClassVar[int]
    LANGUAGES_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    BUDGET_MS_FIELD_NUMBER: _ClassVar[int]
    LEXICAL_BUDGET_MS_FIELD_NUMBER: _ClassVar[int]
    SEMANTIC_BUDGET_MS_FIELD_NUMBER: _ClassVar[int]
    GRAPH_BUDGET_MS_FIELD_NUMBER: _ClassVar[int]
    query: str
    limit: int
    target: CodeTarget
    families: _containers.RepeatedScalarFieldContainer[FactFamily]
    expand_edges: bool
    roles: _containers.RepeatedScalarFieldContainer[str]
    languages: _containers.RepeatedScalarFieldContainer[str]
    scope: str
    budget_ms: int
    lexical_budget_ms: int
    semantic_budget_ms: int
    graph_budget_ms: int
    def __init__(self, query: _Optional[str] = ..., limit: _Optional[int] = ..., target: _Optional[_Union[CodeTarget, _Mapping]] = ..., families: _Optional[_Iterable[_Union[FactFamily, str]]] = ..., expand_edges: _Optional[bool] = ..., roles: _Optional[_Iterable[str]] = ..., languages: _Optional[_Iterable[str]] = ..., scope: _Optional[str] = ..., budget_ms: _Optional[int] = ..., lexical_budget_ms: _Optional[int] = ..., semantic_budget_ms: _Optional[int] = ..., graph_budget_ms: _Optional[int] = ...) -> None: ...

class SearchHit(_message.Message):
    __slots__ = ("id", "title", "text", "score", "path", "analyzer", "evidence_status", "fact_kind", "edge_expansions", "source_hash", "generation", "role", "scope", "retrieval_regime", "retrieval_explanation", "proof_status", "start_line", "end_line", "rank_factors")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    ANALYZER_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_STATUS_FIELD_NUMBER: _ClassVar[int]
    FACT_KIND_FIELD_NUMBER: _ClassVar[int]
    EDGE_EXPANSIONS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_HASH_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    RETRIEVAL_REGIME_FIELD_NUMBER: _ClassVar[int]
    RETRIEVAL_EXPLANATION_FIELD_NUMBER: _ClassVar[int]
    PROOF_STATUS_FIELD_NUMBER: _ClassVar[int]
    START_LINE_FIELD_NUMBER: _ClassVar[int]
    END_LINE_FIELD_NUMBER: _ClassVar[int]
    RANK_FACTORS_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    text: str
    score: float
    path: str
    analyzer: str
    evidence_status: EvidenceStatus
    fact_kind: str
    edge_expansions: _containers.RepeatedCompositeFieldContainer[SearchExpansion]
    source_hash: str
    generation: str
    role: str
    scope: str
    retrieval_regime: str
    retrieval_explanation: str
    proof_status: str
    start_line: int
    end_line: int
    rank_factors: _containers.RepeatedCompositeFieldContainer[SearchRankFactor]
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., text: _Optional[str] = ..., score: _Optional[float] = ..., path: _Optional[str] = ..., analyzer: _Optional[str] = ..., evidence_status: _Optional[_Union[EvidenceStatus, str]] = ..., fact_kind: _Optional[str] = ..., edge_expansions: _Optional[_Iterable[_Union[SearchExpansion, _Mapping]]] = ..., source_hash: _Optional[str] = ..., generation: _Optional[str] = ..., role: _Optional[str] = ..., scope: _Optional[str] = ..., retrieval_regime: _Optional[str] = ..., retrieval_explanation: _Optional[str] = ..., proof_status: _Optional[str] = ..., start_line: _Optional[int] = ..., end_line: _Optional[int] = ..., rank_factors: _Optional[_Iterable[_Union[SearchRankFactor, _Mapping]]] = ...) -> None: ...

class SearchRankFactor(_message.Message):
    __slots__ = ("name", "value", "leg", "rank")
    NAME_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    LEG_FIELD_NUMBER: _ClassVar[int]
    RANK_FIELD_NUMBER: _ClassVar[int]
    name: str
    value: float
    leg: str
    rank: int
    def __init__(self, name: _Optional[str] = ..., value: _Optional[float] = ..., leg: _Optional[str] = ..., rank: _Optional[int] = ...) -> None: ...

class SearchExpansion(_message.Message):
    __slots__ = ("id", "title", "text", "path", "analyzer", "evidence_status", "fact_kind", "family")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    ANALYZER_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_STATUS_FIELD_NUMBER: _ClassVar[int]
    FACT_KIND_FIELD_NUMBER: _ClassVar[int]
    FAMILY_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    text: str
    path: str
    analyzer: str
    evidence_status: EvidenceStatus
    fact_kind: str
    family: FactFamily
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., text: _Optional[str] = ..., path: _Optional[str] = ..., analyzer: _Optional[str] = ..., evidence_status: _Optional[_Union[EvidenceStatus, str]] = ..., fact_kind: _Optional[str] = ..., family: _Optional[_Union[FactFamily, str]] = ...) -> None: ...

class SearchResponse(_message.Message):
    __slots__ = ("results", "degraded_stages", "generation", "retrieval_regime")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_STAGES_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    RETRIEVAL_REGIME_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[SearchHit]
    degraded_stages: _containers.RepeatedScalarFieldContainer[str]
    generation: str
    retrieval_regime: str
    def __init__(self, results: _Optional[_Iterable[_Union[SearchHit, _Mapping]]] = ..., degraded_stages: _Optional[_Iterable[str]] = ..., generation: _Optional[str] = ..., retrieval_regime: _Optional[str] = ...) -> None: ...

class IndexJob(_message.Message):
    __slots__ = ("id", "kind", "state", "generation", "processed", "total", "cursor", "error", "created_at_unix", "updated_at_unix", "cancellation_requested")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    PROCESSED_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    CURSOR_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_UNIX_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_UNIX_FIELD_NUMBER: _ClassVar[int]
    CANCELLATION_REQUESTED_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    state: IndexJobState
    generation: str
    processed: int
    total: int
    cursor: str
    error: str
    created_at_unix: int
    updated_at_unix: int
    cancellation_requested: bool
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., state: _Optional[_Union[IndexJobState, str]] = ..., generation: _Optional[str] = ..., processed: _Optional[int] = ..., total: _Optional[int] = ..., cursor: _Optional[str] = ..., error: _Optional[str] = ..., created_at_unix: _Optional[int] = ..., updated_at_unix: _Optional[int] = ..., cancellation_requested: _Optional[bool] = ...) -> None: ...

class GetIndexStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class IndexStatus(_message.Message):
    __slots__ = ("active_generation", "previous_generation", "state", "source_files", "search_documents", "semantic_cards", "graph_facts", "storage_bytes", "last_reconcile_at_unix", "last_reconcile_outcome", "descriptor_digest", "source_digest", "degraded_stages", "active_jobs")
    ACTIVE_GENERATION_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_GENERATION_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FILES_FIELD_NUMBER: _ClassVar[int]
    SEARCH_DOCUMENTS_FIELD_NUMBER: _ClassVar[int]
    SEMANTIC_CARDS_FIELD_NUMBER: _ClassVar[int]
    GRAPH_FACTS_FIELD_NUMBER: _ClassVar[int]
    STORAGE_BYTES_FIELD_NUMBER: _ClassVar[int]
    LAST_RECONCILE_AT_UNIX_FIELD_NUMBER: _ClassVar[int]
    LAST_RECONCILE_OUTCOME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTOR_DIGEST_FIELD_NUMBER: _ClassVar[int]
    SOURCE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_STAGES_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_JOBS_FIELD_NUMBER: _ClassVar[int]
    active_generation: str
    previous_generation: str
    state: str
    source_files: int
    search_documents: int
    semantic_cards: int
    graph_facts: int
    storage_bytes: int
    last_reconcile_at_unix: int
    last_reconcile_outcome: str
    descriptor_digest: str
    source_digest: str
    degraded_stages: _containers.RepeatedScalarFieldContainer[str]
    active_jobs: _containers.RepeatedCompositeFieldContainer[IndexJob]
    def __init__(self, active_generation: _Optional[str] = ..., previous_generation: _Optional[str] = ..., state: _Optional[str] = ..., source_files: _Optional[int] = ..., search_documents: _Optional[int] = ..., semantic_cards: _Optional[int] = ..., graph_facts: _Optional[int] = ..., storage_bytes: _Optional[int] = ..., last_reconcile_at_unix: _Optional[int] = ..., last_reconcile_outcome: _Optional[str] = ..., descriptor_digest: _Optional[str] = ..., source_digest: _Optional[str] = ..., degraded_stages: _Optional[_Iterable[str]] = ..., active_jobs: _Optional[_Iterable[_Union[IndexJob, _Mapping]]] = ...) -> None: ...

class ReconcileIndexRequest(_message.Message):
    __slots__ = ("generation",)
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    generation: str
    def __init__(self, generation: _Optional[str] = ...) -> None: ...

class ReindexRequest(_message.Message):
    __slots__ = ("generation", "confirmed")
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    CONFIRMED_FIELD_NUMBER: _ClassVar[int]
    generation: str
    confirmed: bool
    def __init__(self, generation: _Optional[str] = ..., confirmed: _Optional[bool] = ...) -> None: ...

class CancelIndexJobRequest(_message.Message):
    __slots__ = ("job_id",)
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    def __init__(self, job_id: _Optional[str] = ...) -> None: ...

class PromoteIndexGenerationRequest(_message.Message):
    __slots__ = ("generation", "confirmed")
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    CONFIRMED_FIELD_NUMBER: _ClassVar[int]
    generation: str
    confirmed: bool
    def __init__(self, generation: _Optional[str] = ..., confirmed: _Optional[bool] = ...) -> None: ...

class RollbackIndexGenerationRequest(_message.Message):
    __slots__ = ("generation", "confirmed")
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    CONFIRMED_FIELD_NUMBER: _ClassVar[int]
    generation: str
    confirmed: bool
    def __init__(self, generation: _Optional[str] = ..., confirmed: _Optional[bool] = ...) -> None: ...

class CleanupIndexRequest(_message.Message):
    __slots__ = ("dry_run", "confirmed")
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    CONFIRMED_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    confirmed: bool
    def __init__(self, dry_run: _Optional[bool] = ..., confirmed: _Optional[bool] = ...) -> None: ...

class IndexControlResponse(_message.Message):
    __slots__ = ("job", "status", "message")
    JOB_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    job: IndexJob
    status: IndexStatus
    message: str
    def __init__(self, job: _Optional[_Union[IndexJob, _Mapping]] = ..., status: _Optional[_Union[IndexStatus, _Mapping]] = ..., message: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("target", "dry_run", "all")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    ALL_FIELD_NUMBER: _ClassVar[int]
    target: CodeTarget
    dry_run: bool
    all: bool
    def __init__(self, target: _Optional[_Union[CodeTarget, _Mapping]] = ..., dry_run: _Optional[bool] = ..., all: _Optional[bool] = ...) -> None: ...

class TargetContext(_message.Message):
    __slots__ = ("requested", "resolved_kind", "root_path", "scenario", "scenario_aware", "root_paths")
    REQUESTED_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_KIND_FIELD_NUMBER: _ClassVar[int]
    ROOT_PATH_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_AWARE_FIELD_NUMBER: _ClassVar[int]
    ROOT_PATHS_FIELD_NUMBER: _ClassVar[int]
    requested: CodeTarget
    resolved_kind: TargetKind
    root_path: str
    scenario: str
    scenario_aware: bool
    root_paths: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, requested: _Optional[_Union[CodeTarget, _Mapping]] = ..., resolved_kind: _Optional[_Union[TargetKind, str]] = ..., root_path: _Optional[str] = ..., scenario: _Optional[str] = ..., scenario_aware: _Optional[bool] = ..., root_paths: _Optional[_Iterable[str]] = ...) -> None: ...

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
    __slots__ = ("id", "language", "root_path", "config_path", "status", "evidence", "toolchain")
    ID_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    ROOT_PATH_FIELD_NUMBER: _ClassVar[int]
    CONFIG_PATH_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    TOOLCHAIN_FIELD_NUMBER: _ClassVar[int]
    id: str
    language: str
    root_path: str
    config_path: str
    status: EvidenceStatus
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    toolchain: ToolchainObservation
    def __init__(self, id: _Optional[str] = ..., language: _Optional[str] = ..., root_path: _Optional[str] = ..., config_path: _Optional[str] = ..., status: _Optional[_Union[EvidenceStatus, str]] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ..., toolchain: _Optional[_Union[ToolchainObservation, _Mapping]] = ...) -> None: ...

class ToolchainObservation(_message.Message):
    __slots__ = ("ecosystem", "manifest_paths", "lockfile_paths", "build_systems", "runner_indicators", "package_manager", "toolchain_identity", "status", "evidence")
    ECOSYSTEM_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_PATHS_FIELD_NUMBER: _ClassVar[int]
    LOCKFILE_PATHS_FIELD_NUMBER: _ClassVar[int]
    BUILD_SYSTEMS_FIELD_NUMBER: _ClassVar[int]
    RUNNER_INDICATORS_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_MANAGER_FIELD_NUMBER: _ClassVar[int]
    TOOLCHAIN_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    ecosystem: str
    manifest_paths: _containers.RepeatedScalarFieldContainer[str]
    lockfile_paths: _containers.RepeatedScalarFieldContainer[str]
    build_systems: _containers.RepeatedScalarFieldContainer[str]
    runner_indicators: _containers.RepeatedScalarFieldContainer[str]
    package_manager: str
    toolchain_identity: str
    status: EvidenceStatus
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    def __init__(self, ecosystem: _Optional[str] = ..., manifest_paths: _Optional[_Iterable[str]] = ..., lockfile_paths: _Optional[_Iterable[str]] = ..., build_systems: _Optional[_Iterable[str]] = ..., runner_indicators: _Optional[_Iterable[str]] = ..., package_manager: _Optional[str] = ..., toolchain_identity: _Optional[str] = ..., status: _Optional[_Union[EvidenceStatus, str]] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ...) -> None: ...

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
    __slots__ = ("cache_key", "hit", "analyzer_version", "graph_hash", "age_seconds", "state", "reason", "source_hash", "config_hash", "provider_version", "schema_version", "created_at_unix", "last_used_at_unix", "hit_count", "scope", "logical_key", "payload_bytes", "codec")
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
    LOGICAL_KEY_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_BYTES_FIELD_NUMBER: _ClassVar[int]
    CODEC_FIELD_NUMBER: _ClassVar[int]
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
    logical_key: str
    payload_bytes: int
    codec: str
    def __init__(self, cache_key: _Optional[str] = ..., hit: _Optional[bool] = ..., analyzer_version: _Optional[str] = ..., graph_hash: _Optional[str] = ..., age_seconds: _Optional[int] = ..., state: _Optional[str] = ..., reason: _Optional[str] = ..., source_hash: _Optional[str] = ..., config_hash: _Optional[str] = ..., provider_version: _Optional[str] = ..., schema_version: _Optional[str] = ..., created_at_unix: _Optional[int] = ..., last_used_at_unix: _Optional[int] = ..., hit_count: _Optional[int] = ..., scope: _Optional[str] = ..., logical_key: _Optional[str] = ..., payload_bytes: _Optional[int] = ..., codec: _Optional[str] = ...) -> None: ...

class CacheScopeSummary(_message.Message):
    __slots__ = ("scope", "row_count", "payload_bytes")
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    ROW_COUNT_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_BYTES_FIELD_NUMBER: _ClassVar[int]
    scope: str
    row_count: int
    payload_bytes: int
    def __init__(self, scope: _Optional[str] = ..., row_count: _Optional[int] = ..., payload_bytes: _Optional[int] = ...) -> None: ...

class CacheStatus(_message.Message):
    __slots__ = ("target", "cache_key", "entries", "entries_metadata", "total_rows", "total_payload_bytes", "budget_bytes", "utilization", "scopes", "last_sweep_at_unix")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    CACHE_KEY_FIELD_NUMBER: _ClassVar[int]
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    ENTRIES_METADATA_FIELD_NUMBER: _ClassVar[int]
    TOTAL_ROWS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_PAYLOAD_BYTES_FIELD_NUMBER: _ClassVar[int]
    BUDGET_BYTES_FIELD_NUMBER: _ClassVar[int]
    UTILIZATION_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    LAST_SWEEP_AT_UNIX_FIELD_NUMBER: _ClassVar[int]
    target: CodeTarget
    cache_key: str
    entries: int
    entries_metadata: _containers.RepeatedCompositeFieldContainer[CacheMetadata]
    total_rows: int
    total_payload_bytes: int
    budget_bytes: int
    utilization: float
    scopes: _containers.RepeatedCompositeFieldContainer[CacheScopeSummary]
    last_sweep_at_unix: int
    def __init__(self, target: _Optional[_Union[CodeTarget, _Mapping]] = ..., cache_key: _Optional[str] = ..., entries: _Optional[int] = ..., entries_metadata: _Optional[_Iterable[_Union[CacheMetadata, _Mapping]]] = ..., total_rows: _Optional[int] = ..., total_payload_bytes: _Optional[int] = ..., budget_bytes: _Optional[int] = ..., utilization: _Optional[float] = ..., scopes: _Optional[_Iterable[_Union[CacheScopeSummary, _Mapping]]] = ..., last_sweep_at_unix: _Optional[int] = ...) -> None: ...

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
    __slots__ = ("target", "parse_units", "surfaces", "facts", "evidence", "warnings", "cache", "next_page_token", "total_facts")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    PARSE_UNITS_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    FACTS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    CACHE_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FACTS_FIELD_NUMBER: _ClassVar[int]
    target: TargetContext
    parse_units: _containers.RepeatedCompositeFieldContainer[ParseUnit]
    surfaces: _containers.RepeatedCompositeFieldContainer[Surface]
    facts: _containers.RepeatedCompositeFieldContainer[GenericFact]
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    warnings: _containers.RepeatedCompositeFieldContainer[Warning]
    cache: CacheMetadata
    next_page_token: str
    total_facts: int
    def __init__(self, target: _Optional[_Union[TargetContext, _Mapping]] = ..., parse_units: _Optional[_Iterable[_Union[ParseUnit, _Mapping]]] = ..., surfaces: _Optional[_Iterable[_Union[Surface, _Mapping]]] = ..., facts: _Optional[_Iterable[_Union[GenericFact, _Mapping]]] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ..., warnings: _Optional[_Iterable[_Union[Warning, _Mapping]]] = ..., cache: _Optional[_Union[CacheMetadata, _Mapping]] = ..., next_page_token: _Optional[str] = ..., total_facts: _Optional[int] = ...) -> None: ...

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
