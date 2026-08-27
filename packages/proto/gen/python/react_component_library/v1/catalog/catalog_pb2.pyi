from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AssetNode(_message.Message):
    __slots__ = ("asset_id", "name", "kind", "rung", "rung_name", "domain", "domain_order")
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    RUNG_FIELD_NUMBER: _ClassVar[int]
    RUNG_NAME_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_ORDER_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    name: str
    kind: str
    rung: int
    rung_name: str
    domain: str
    domain_order: int
    def __init__(self, asset_id: _Optional[str] = ..., name: _Optional[str] = ..., kind: _Optional[str] = ..., rung: _Optional[int] = ..., rung_name: _Optional[str] = ..., domain: _Optional[str] = ..., domain_order: _Optional[int] = ...) -> None: ...

class RungBand(_message.Message):
    __slots__ = ("rung", "rung_name", "assets", "count")
    RUNG_FIELD_NUMBER: _ClassVar[int]
    RUNG_NAME_FIELD_NUMBER: _ClassVar[int]
    ASSETS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    rung: int
    rung_name: str
    assets: _containers.RepeatedCompositeFieldContainer[AssetNode]
    count: int
    def __init__(self, rung: _Optional[int] = ..., rung_name: _Optional[str] = ..., assets: _Optional[_Iterable[_Union[AssetNode, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class AssetRelationships(_message.Message):
    __slots__ = ("root", "direct_dependencies", "closure", "closure_bands", "direct_dependents", "transitive_dependents", "transitive_dependent_count")
    ROOT_FIELD_NUMBER: _ClassVar[int]
    DIRECT_DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    CLOSURE_FIELD_NUMBER: _ClassVar[int]
    CLOSURE_BANDS_FIELD_NUMBER: _ClassVar[int]
    DIRECT_DEPENDENTS_FIELD_NUMBER: _ClassVar[int]
    TRANSITIVE_DEPENDENTS_FIELD_NUMBER: _ClassVar[int]
    TRANSITIVE_DEPENDENT_COUNT_FIELD_NUMBER: _ClassVar[int]
    root: AssetNode
    direct_dependencies: _containers.RepeatedCompositeFieldContainer[AssetNode]
    closure: _containers.RepeatedCompositeFieldContainer[AssetNode]
    closure_bands: _containers.RepeatedCompositeFieldContainer[RungBand]
    direct_dependents: _containers.RepeatedCompositeFieldContainer[AssetNode]
    transitive_dependents: _containers.RepeatedCompositeFieldContainer[AssetNode]
    transitive_dependent_count: int
    def __init__(self, root: _Optional[_Union[AssetNode, _Mapping]] = ..., direct_dependencies: _Optional[_Iterable[_Union[AssetNode, _Mapping]]] = ..., closure: _Optional[_Iterable[_Union[AssetNode, _Mapping]]] = ..., closure_bands: _Optional[_Iterable[_Union[RungBand, _Mapping]]] = ..., direct_dependents: _Optional[_Iterable[_Union[AssetNode, _Mapping]]] = ..., transitive_dependents: _Optional[_Iterable[_Union[AssetNode, _Mapping]]] = ..., transitive_dependent_count: _Optional[int] = ...) -> None: ...

class GetAssetRelationshipsRequest(_message.Message):
    __slots__ = ("asset_id",)
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    def __init__(self, asset_id: _Optional[str] = ...) -> None: ...

class GetAssetRelationshipsResponse(_message.Message):
    __slots__ = ("relationships",)
    RELATIONSHIPS_FIELD_NUMBER: _ClassVar[int]
    relationships: AssetRelationships
    def __init__(self, relationships: _Optional[_Union[AssetRelationships, _Mapping]] = ...) -> None: ...

class GetCatalogStructureRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RungPopulation(_message.Message):
    __slots__ = ("rung", "rung_name", "count")
    RUNG_FIELD_NUMBER: _ClassVar[int]
    RUNG_NAME_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    rung: int
    rung_name: str
    count: int
    def __init__(self, rung: _Optional[int] = ..., rung_name: _Optional[str] = ..., count: _Optional[int] = ...) -> None: ...

class StructureInvariant(_message.Message):
    __slots__ = ("id", "label", "status", "detail")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    status: str
    detail: str
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., status: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class BlastRadiusRow(_message.Message):
    __slots__ = ("asset", "transitive_dependent_count")
    ASSET_FIELD_NUMBER: _ClassVar[int]
    TRANSITIVE_DEPENDENT_COUNT_FIELD_NUMBER: _ClassVar[int]
    asset: AssetNode
    transitive_dependent_count: int
    def __init__(self, asset: _Optional[_Union[AssetNode, _Mapping]] = ..., transitive_dependent_count: _Optional[int] = ...) -> None: ...

class CatalogStructure(_message.Message):
    __slots__ = ("population", "invariants", "blast_radius")
    POPULATION_FIELD_NUMBER: _ClassVar[int]
    INVARIANTS_FIELD_NUMBER: _ClassVar[int]
    BLAST_RADIUS_FIELD_NUMBER: _ClassVar[int]
    population: _containers.RepeatedCompositeFieldContainer[RungPopulation]
    invariants: _containers.RepeatedCompositeFieldContainer[StructureInvariant]
    blast_radius: _containers.RepeatedCompositeFieldContainer[BlastRadiusRow]
    def __init__(self, population: _Optional[_Iterable[_Union[RungPopulation, _Mapping]]] = ..., invariants: _Optional[_Iterable[_Union[StructureInvariant, _Mapping]]] = ..., blast_radius: _Optional[_Iterable[_Union[BlastRadiusRow, _Mapping]]] = ...) -> None: ...

class GetCatalogStructureResponse(_message.Message):
    __slots__ = ("structure",)
    STRUCTURE_FIELD_NUMBER: _ClassVar[int]
    structure: CatalogStructure
    def __init__(self, structure: _Optional[_Union[CatalogStructure, _Mapping]] = ...) -> None: ...

class ReconcileGraphRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ReconciliationAsset(_message.Message):
    __slots__ = ("asset_id", "verdict", "cause", "catalog_edges", "manifest_edges", "import_edges")
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    CAUSE_FIELD_NUMBER: _ClassVar[int]
    CATALOG_EDGES_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_EDGES_FIELD_NUMBER: _ClassVar[int]
    IMPORT_EDGES_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    verdict: str
    cause: str
    catalog_edges: _containers.RepeatedScalarFieldContainer[str]
    manifest_edges: _containers.RepeatedScalarFieldContainer[str]
    import_edges: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, asset_id: _Optional[str] = ..., verdict: _Optional[str] = ..., cause: _Optional[str] = ..., catalog_edges: _Optional[_Iterable[str]] = ..., manifest_edges: _Optional[_Iterable[str]] = ..., import_edges: _Optional[_Iterable[str]] = ...) -> None: ...

class ReconciliationDistribution(_message.Message):
    __slots__ = ("counts",)
    class CountsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    COUNTS_FIELD_NUMBER: _ClassVar[int]
    counts: _containers.ScalarMap[str, int]
    def __init__(self, counts: _Optional[_Mapping[str, int]] = ...) -> None: ...

class ReconcileGraphResponse(_message.Message):
    __slots__ = ("assets", "distribution")
    ASSETS_FIELD_NUMBER: _ClassVar[int]
    DISTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    assets: _containers.RepeatedCompositeFieldContainer[ReconciliationAsset]
    distribution: ReconciliationDistribution
    def __init__(self, assets: _Optional[_Iterable[_Union[ReconciliationAsset, _Mapping]]] = ..., distribution: _Optional[_Union[ReconciliationDistribution, _Mapping]] = ...) -> None: ...

class GetAssetPortContractRequest(_message.Message):
    __slots__ = ("asset_id",)
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    def __init__(self, asset_id: _Optional[str] = ...) -> None: ...

class UnmetPort(_message.Message):
    __slots__ = ("capability_id", "demanding_assets", "candidate_satisfiers")
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    DEMANDING_ASSETS_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_SATISFIERS_FIELD_NUMBER: _ClassVar[int]
    capability_id: str
    demanding_assets: _containers.RepeatedCompositeFieldContainer[AssetNode]
    candidate_satisfiers: _containers.RepeatedCompositeFieldContainer[AssetNode]
    def __init__(self, capability_id: _Optional[str] = ..., demanding_assets: _Optional[_Iterable[_Union[AssetNode, _Mapping]]] = ..., candidate_satisfiers: _Optional[_Iterable[_Union[AssetNode, _Mapping]]] = ...) -> None: ...

class AssetPortContract(_message.Message):
    __slots__ = ("asset_id", "closure_count", "self_contained", "unmet_ports")
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    CLOSURE_COUNT_FIELD_NUMBER: _ClassVar[int]
    SELF_CONTAINED_FIELD_NUMBER: _ClassVar[int]
    UNMET_PORTS_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    closure_count: int
    self_contained: bool
    unmet_ports: _containers.RepeatedCompositeFieldContainer[UnmetPort]
    def __init__(self, asset_id: _Optional[str] = ..., closure_count: _Optional[int] = ..., self_contained: _Optional[bool] = ..., unmet_ports: _Optional[_Iterable[_Union[UnmetPort, _Mapping]]] = ...) -> None: ...

class GetAssetPortContractResponse(_message.Message):
    __slots__ = ("contract",)
    CONTRACT_FIELD_NUMBER: _ClassVar[int]
    contract: AssetPortContract
    def __init__(self, contract: _Optional[_Union[AssetPortContract, _Mapping]] = ...) -> None: ...

class GetCoverageRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class CoverageRow(_message.Message):
    __slots__ = ("asset_id", "name", "domain", "kind", "priority", "bucket", "platform", "target", "achieved", "implementation", "blocks_downstream", "rung", "rung_name", "domain_order", "asset_score", "weight", "passed_gates", "failed_gates", "nearest_blocking_gate", "newest_evidence", "visual_evidence")
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    BUCKET_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    ACHIEVED_FIELD_NUMBER: _ClassVar[int]
    IMPLEMENTATION_FIELD_NUMBER: _ClassVar[int]
    BLOCKS_DOWNSTREAM_FIELD_NUMBER: _ClassVar[int]
    RUNG_FIELD_NUMBER: _ClassVar[int]
    RUNG_NAME_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_ORDER_FIELD_NUMBER: _ClassVar[int]
    ASSET_SCORE_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FIELD_NUMBER: _ClassVar[int]
    PASSED_GATES_FIELD_NUMBER: _ClassVar[int]
    FAILED_GATES_FIELD_NUMBER: _ClassVar[int]
    NEAREST_BLOCKING_GATE_FIELD_NUMBER: _ClassVar[int]
    NEWEST_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    VISUAL_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    name: str
    domain: str
    kind: str
    priority: str
    bucket: str
    platform: str
    target: str
    achieved: str
    implementation: str
    blocks_downstream: int
    rung: int
    rung_name: str
    domain_order: int
    asset_score: float
    weight: float
    passed_gates: _containers.RepeatedScalarFieldContainer[str]
    failed_gates: _containers.RepeatedScalarFieldContainer[str]
    nearest_blocking_gate: str
    newest_evidence: str
    visual_evidence: bool
    def __init__(self, asset_id: _Optional[str] = ..., name: _Optional[str] = ..., domain: _Optional[str] = ..., kind: _Optional[str] = ..., priority: _Optional[str] = ..., bucket: _Optional[str] = ..., platform: _Optional[str] = ..., target: _Optional[str] = ..., achieved: _Optional[str] = ..., implementation: _Optional[str] = ..., blocks_downstream: _Optional[int] = ..., rung: _Optional[int] = ..., rung_name: _Optional[str] = ..., domain_order: _Optional[int] = ..., asset_score: _Optional[float] = ..., weight: _Optional[float] = ..., passed_gates: _Optional[_Iterable[str]] = ..., failed_gates: _Optional[_Iterable[str]] = ..., nearest_blocking_gate: _Optional[str] = ..., newest_evidence: _Optional[str] = ..., visual_evidence: _Optional[bool] = ...) -> None: ...

class Rollup(_message.Message):
    __slots__ = ("key", "planned", "built")
    KEY_FIELD_NUMBER: _ClassVar[int]
    PLANNED_FIELD_NUMBER: _ClassVar[int]
    BUILT_FIELD_NUMBER: _ClassVar[int]
    key: str
    planned: int
    built: int
    def __init__(self, key: _Optional[str] = ..., planned: _Optional[int] = ..., built: _Optional[int] = ...) -> None: ...

class MaturitySummary(_message.Message):
    __slots__ = ("total", "at_or_above_target", "by_rung", "catalog_completion", "mandatory_gate_coverage", "weighted_quality", "production_ready_coverage", "weighted_asset_score", "score_weight_numerator", "score_weight_denominator", "by_gate", "by_rung_score", "corpus", "pass_evidence", "fail_evidence", "unmeasured_evidence", "kind_mismatch_count", "instrument_moved_count")
    class ByRungEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    AT_OR_ABOVE_TARGET_FIELD_NUMBER: _ClassVar[int]
    BY_RUNG_FIELD_NUMBER: _ClassVar[int]
    CATALOG_COMPLETION_FIELD_NUMBER: _ClassVar[int]
    MANDATORY_GATE_COVERAGE_FIELD_NUMBER: _ClassVar[int]
    WEIGHTED_QUALITY_FIELD_NUMBER: _ClassVar[int]
    PRODUCTION_READY_COVERAGE_FIELD_NUMBER: _ClassVar[int]
    WEIGHTED_ASSET_SCORE_FIELD_NUMBER: _ClassVar[int]
    SCORE_WEIGHT_NUMERATOR_FIELD_NUMBER: _ClassVar[int]
    SCORE_WEIGHT_DENOMINATOR_FIELD_NUMBER: _ClassVar[int]
    BY_GATE_FIELD_NUMBER: _ClassVar[int]
    BY_RUNG_SCORE_FIELD_NUMBER: _ClassVar[int]
    CORPUS_FIELD_NUMBER: _ClassVar[int]
    PASS_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    FAIL_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    UNMEASURED_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    KIND_MISMATCH_COUNT_FIELD_NUMBER: _ClassVar[int]
    INSTRUMENT_MOVED_COUNT_FIELD_NUMBER: _ClassVar[int]
    total: int
    at_or_above_target: int
    by_rung: _containers.ScalarMap[str, int]
    catalog_completion: CoverageMetric
    mandatory_gate_coverage: CoverageMetric
    weighted_quality: CoverageMetric
    production_ready_coverage: CoverageMetric
    weighted_asset_score: float
    score_weight_numerator: float
    score_weight_denominator: float
    by_gate: _containers.RepeatedCompositeFieldContainer[ScoreBreakdown]
    by_rung_score: _containers.RepeatedCompositeFieldContainer[ScoreBreakdown]
    corpus: _containers.RepeatedCompositeFieldContainer[CorpusStatus]
    pass_evidence: int
    fail_evidence: int
    unmeasured_evidence: int
    kind_mismatch_count: int
    instrument_moved_count: int
    def __init__(self, total: _Optional[int] = ..., at_or_above_target: _Optional[int] = ..., by_rung: _Optional[_Mapping[str, int]] = ..., catalog_completion: _Optional[_Union[CoverageMetric, _Mapping]] = ..., mandatory_gate_coverage: _Optional[_Union[CoverageMetric, _Mapping]] = ..., weighted_quality: _Optional[_Union[CoverageMetric, _Mapping]] = ..., production_ready_coverage: _Optional[_Union[CoverageMetric, _Mapping]] = ..., weighted_asset_score: _Optional[float] = ..., score_weight_numerator: _Optional[float] = ..., score_weight_denominator: _Optional[float] = ..., by_gate: _Optional[_Iterable[_Union[ScoreBreakdown, _Mapping]]] = ..., by_rung_score: _Optional[_Iterable[_Union[ScoreBreakdown, _Mapping]]] = ..., corpus: _Optional[_Iterable[_Union[CorpusStatus, _Mapping]]] = ..., pass_evidence: _Optional[int] = ..., fail_evidence: _Optional[int] = ..., unmeasured_evidence: _Optional[int] = ..., kind_mismatch_count: _Optional[int] = ..., instrument_moved_count: _Optional[int] = ...) -> None: ...

class ScoreBreakdown(_message.Message):
    __slots__ = ("key", "passed", "applicable", "score")
    KEY_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    APPLICABLE_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    key: str
    passed: int
    applicable: int
    score: float
    def __init__(self, key: _Optional[str] = ..., passed: _Optional[int] = ..., applicable: _Optional[int] = ..., score: _Optional[float] = ...) -> None: ...

class CorpusStatus(_message.Message):
    __slots__ = ("gate", "result", "finding_count", "runner_error_count")
    GATE_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    FINDING_COUNT_FIELD_NUMBER: _ClassVar[int]
    RUNNER_ERROR_COUNT_FIELD_NUMBER: _ClassVar[int]
    gate: str
    result: str
    finding_count: int
    runner_error_count: int
    def __init__(self, gate: _Optional[str] = ..., result: _Optional[str] = ..., finding_count: _Optional[int] = ..., runner_error_count: _Optional[int] = ...) -> None: ...

class CoverageMetric(_message.Message):
    __slots__ = ("numerator", "denominator", "ratio")
    NUMERATOR_FIELD_NUMBER: _ClassVar[int]
    DENOMINATOR_FIELD_NUMBER: _ClassVar[int]
    RATIO_FIELD_NUMBER: _ClassVar[int]
    numerator: int
    denominator: int
    ratio: float
    def __init__(self, numerator: _Optional[int] = ..., denominator: _Optional[int] = ..., ratio: _Optional[float] = ...) -> None: ...

class DeclaredCapabilityCoverage(_message.Message):
    __slots__ = ("capability", "title", "status", "checkable", "unmeasured", "declared_asset_count", "asset_ids", "blockers")
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CHECKABLE_FIELD_NUMBER: _ClassVar[int]
    UNMEASURED_FIELD_NUMBER: _ClassVar[int]
    DECLARED_ASSET_COUNT_FIELD_NUMBER: _ClassVar[int]
    ASSET_IDS_FIELD_NUMBER: _ClassVar[int]
    BLOCKERS_FIELD_NUMBER: _ClassVar[int]
    capability: str
    title: str
    status: str
    checkable: bool
    unmeasured: bool
    declared_asset_count: int
    asset_ids: _containers.RepeatedScalarFieldContainer[str]
    blockers: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, capability: _Optional[str] = ..., title: _Optional[str] = ..., status: _Optional[str] = ..., checkable: _Optional[bool] = ..., unmeasured: _Optional[bool] = ..., declared_asset_count: _Optional[int] = ..., asset_ids: _Optional[_Iterable[str]] = ..., blockers: _Optional[_Iterable[str]] = ...) -> None: ...

class CoverageReport(_message.Message):
    __slots__ = ("rows", "totals", "by_domain", "by_priority", "maturity", "composition_scores", "composition_median", "bespoke_escape_count", "declared_capability_asset_count", "declared_uncheckable_asset_count", "unmeasured_capability_asset_count", "capability_declaration_count", "capability_coverage", "composition_blocked_asset_count")
    class TotalsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    class CompositionScoresEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    ROWS_FIELD_NUMBER: _ClassVar[int]
    TOTALS_FIELD_NUMBER: _ClassVar[int]
    BY_DOMAIN_FIELD_NUMBER: _ClassVar[int]
    BY_PRIORITY_FIELD_NUMBER: _ClassVar[int]
    MATURITY_FIELD_NUMBER: _ClassVar[int]
    COMPOSITION_SCORES_FIELD_NUMBER: _ClassVar[int]
    COMPOSITION_MEDIAN_FIELD_NUMBER: _ClassVar[int]
    BESPOKE_ESCAPE_COUNT_FIELD_NUMBER: _ClassVar[int]
    DECLARED_CAPABILITY_ASSET_COUNT_FIELD_NUMBER: _ClassVar[int]
    DECLARED_UNCHECKABLE_ASSET_COUNT_FIELD_NUMBER: _ClassVar[int]
    UNMEASURED_CAPABILITY_ASSET_COUNT_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_DECLARATION_COUNT_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_COVERAGE_FIELD_NUMBER: _ClassVar[int]
    COMPOSITION_BLOCKED_ASSET_COUNT_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[CoverageRow]
    totals: _containers.ScalarMap[str, int]
    by_domain: _containers.RepeatedCompositeFieldContainer[Rollup]
    by_priority: _containers.RepeatedCompositeFieldContainer[Rollup]
    maturity: MaturitySummary
    composition_scores: _containers.ScalarMap[str, float]
    composition_median: float
    bespoke_escape_count: int
    declared_capability_asset_count: int
    declared_uncheckable_asset_count: int
    unmeasured_capability_asset_count: int
    capability_declaration_count: int
    capability_coverage: _containers.RepeatedCompositeFieldContainer[DeclaredCapabilityCoverage]
    composition_blocked_asset_count: int
    def __init__(self, rows: _Optional[_Iterable[_Union[CoverageRow, _Mapping]]] = ..., totals: _Optional[_Mapping[str, int]] = ..., by_domain: _Optional[_Iterable[_Union[Rollup, _Mapping]]] = ..., by_priority: _Optional[_Iterable[_Union[Rollup, _Mapping]]] = ..., maturity: _Optional[_Union[MaturitySummary, _Mapping]] = ..., composition_scores: _Optional[_Mapping[str, float]] = ..., composition_median: _Optional[float] = ..., bespoke_escape_count: _Optional[int] = ..., declared_capability_asset_count: _Optional[int] = ..., declared_uncheckable_asset_count: _Optional[int] = ..., unmeasured_capability_asset_count: _Optional[int] = ..., capability_declaration_count: _Optional[int] = ..., capability_coverage: _Optional[_Iterable[_Union[DeclaredCapabilityCoverage, _Mapping]]] = ..., composition_blocked_asset_count: _Optional[int] = ...) -> None: ...

class GetCoverageResponse(_message.Message):
    __slots__ = ("report",)
    REPORT_FIELD_NUMBER: _ClassVar[int]
    report: CoverageReport
    def __init__(self, report: _Optional[_Union[CoverageReport, _Mapping]] = ...) -> None: ...

class ListNextWorkRequest(_message.Message):
    __slots__ = ("limit", "lane")
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    LANE_FIELD_NUMBER: _ClassVar[int]
    limit: int
    lane: str
    def __init__(self, limit: _Optional[int] = ..., lane: _Optional[str] = ...) -> None: ...

class ListNextWorkResponse(_message.Message):
    __slots__ = ("rows", "maturity", "lane", "promote", "build")
    ROWS_FIELD_NUMBER: _ClassVar[int]
    MATURITY_FIELD_NUMBER: _ClassVar[int]
    LANE_FIELD_NUMBER: _ClassVar[int]
    PROMOTE_FIELD_NUMBER: _ClassVar[int]
    BUILD_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[CoverageRow]
    maturity: MaturitySummary
    lane: str
    promote: _containers.RepeatedCompositeFieldContainer[CoverageRow]
    build: _containers.RepeatedCompositeFieldContainer[CoverageRow]
    def __init__(self, rows: _Optional[_Iterable[_Union[CoverageRow, _Mapping]]] = ..., maturity: _Optional[_Union[MaturitySummary, _Mapping]] = ..., lane: _Optional[str] = ..., promote: _Optional[_Iterable[_Union[CoverageRow, _Mapping]]] = ..., build: _Optional[_Iterable[_Union[CoverageRow, _Mapping]]] = ...) -> None: ...

class RunGateRequest(_message.Message):
    __slots__ = ("gate", "all", "calibration_only")
    GATE_FIELD_NUMBER: _ClassVar[int]
    ALL_FIELD_NUMBER: _ClassVar[int]
    CALIBRATION_ONLY_FIELD_NUMBER: _ClassVar[int]
    gate: str
    all: bool
    calibration_only: bool
    def __init__(self, gate: _Optional[str] = ..., all: _Optional[bool] = ..., calibration_only: _Optional[bool] = ...) -> None: ...

class GateFinding(_message.Message):
    __slots__ = ("code", "message", "asset_id", "severity", "file", "line", "remediation", "docs_ref")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    FILE_FIELD_NUMBER: _ClassVar[int]
    LINE_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    DOCS_REF_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    asset_id: str
    severity: str
    file: str
    line: int
    remediation: str
    docs_ref: str
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., asset_id: _Optional[str] = ..., severity: _Optional[str] = ..., file: _Optional[str] = ..., line: _Optional[int] = ..., remediation: _Optional[str] = ..., docs_ref: _Optional[str] = ...) -> None: ...

class RunGateResponse(_message.Message):
    __slots__ = ("gate", "findings", "inspected_files", "runner_errors", "evidence_rows_written", "calibration", "non_discriminating", "surface_verdict_counts", "composition_scores", "composition_median", "bespoke_escape_count", "composition_escapes")
    class SurfaceVerdictCountsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    class CompositionScoresEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    GATE_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    INSPECTED_FILES_FIELD_NUMBER: _ClassVar[int]
    RUNNER_ERRORS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_ROWS_WRITTEN_FIELD_NUMBER: _ClassVar[int]
    CALIBRATION_FIELD_NUMBER: _ClassVar[int]
    NON_DISCRIMINATING_FIELD_NUMBER: _ClassVar[int]
    SURFACE_VERDICT_COUNTS_FIELD_NUMBER: _ClassVar[int]
    COMPOSITION_SCORES_FIELD_NUMBER: _ClassVar[int]
    COMPOSITION_MEDIAN_FIELD_NUMBER: _ClassVar[int]
    BESPOKE_ESCAPE_COUNT_FIELD_NUMBER: _ClassVar[int]
    COMPOSITION_ESCAPES_FIELD_NUMBER: _ClassVar[int]
    gate: str
    findings: _containers.RepeatedCompositeFieldContainer[GateFinding]
    inspected_files: int
    runner_errors: _containers.RepeatedCompositeFieldContainer[GateFinding]
    evidence_rows_written: int
    calibration: _containers.RepeatedCompositeFieldContainer[CalibrationResult]
    non_discriminating: bool
    surface_verdict_counts: _containers.ScalarMap[str, int]
    composition_scores: _containers.ScalarMap[str, float]
    composition_median: float
    bespoke_escape_count: int
    composition_escapes: _containers.RepeatedCompositeFieldContainer[CompositionEscape]
    def __init__(self, gate: _Optional[str] = ..., findings: _Optional[_Iterable[_Union[GateFinding, _Mapping]]] = ..., inspected_files: _Optional[int] = ..., runner_errors: _Optional[_Iterable[_Union[GateFinding, _Mapping]]] = ..., evidence_rows_written: _Optional[int] = ..., calibration: _Optional[_Iterable[_Union[CalibrationResult, _Mapping]]] = ..., non_discriminating: _Optional[bool] = ..., surface_verdict_counts: _Optional[_Mapping[str, int]] = ..., composition_scores: _Optional[_Mapping[str, float]] = ..., composition_median: _Optional[float] = ..., bespoke_escape_count: _Optional[int] = ..., composition_escapes: _Optional[_Iterable[_Union[CompositionEscape, _Mapping]]] = ...) -> None: ...

class CompositionEscape(_message.Message):
    __slots__ = ("asset_id", "reason")
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    reason: str
    def __init__(self, asset_id: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class CalibrationResult(_message.Message):
    __slots__ = ("gate", "fixture", "required_failure_code", "observed_failure_code", "status", "message")
    GATE_FIELD_NUMBER: _ClassVar[int]
    FIXTURE_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FAILURE_CODE_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_FAILURE_CODE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    gate: str
    fixture: str
    required_failure_code: str
    observed_failure_code: str
    status: str
    message: str
    def __init__(self, gate: _Optional[str] = ..., fixture: _Optional[str] = ..., required_failure_code: _Optional[str] = ..., observed_failure_code: _Optional[str] = ..., status: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class ScoreHistoryPoint(_message.Message):
    __slots__ = ("recorded_at", "score", "assets_at_100", "assets_below_50", "weight_vector_regenerated", "scoring_model_version", "source_revision", "instrument_moved_count", "kind_mismatch_count", "events")
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    ASSETS_AT_100_FIELD_NUMBER: _ClassVar[int]
    ASSETS_BELOW_50_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_VECTOR_REGENERATED_FIELD_NUMBER: _ClassVar[int]
    SCORING_MODEL_VERSION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REVISION_FIELD_NUMBER: _ClassVar[int]
    INSTRUMENT_MOVED_COUNT_FIELD_NUMBER: _ClassVar[int]
    KIND_MISMATCH_COUNT_FIELD_NUMBER: _ClassVar[int]
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    recorded_at: str
    score: float
    assets_at_100: int
    assets_below_50: int
    weight_vector_regenerated: bool
    scoring_model_version: int
    source_revision: str
    instrument_moved_count: int
    kind_mismatch_count: int
    events: _containers.RepeatedCompositeFieldContainer[ScoreHistoryEvent]
    def __init__(self, recorded_at: _Optional[str] = ..., score: _Optional[float] = ..., assets_at_100: _Optional[int] = ..., assets_below_50: _Optional[int] = ..., weight_vector_regenerated: _Optional[bool] = ..., scoring_model_version: _Optional[int] = ..., source_revision: _Optional[str] = ..., instrument_moved_count: _Optional[int] = ..., kind_mismatch_count: _Optional[int] = ..., events: _Optional[_Iterable[_Union[ScoreHistoryEvent, _Mapping]]] = ...) -> None: ...

class ScoreHistoryEvent(_message.Message):
    __slots__ = ("type", "asset_id", "source_revision", "declared_kind", "derived_kind")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REVISION_FIELD_NUMBER: _ClassVar[int]
    DECLARED_KIND_FIELD_NUMBER: _ClassVar[int]
    DERIVED_KIND_FIELD_NUMBER: _ClassVar[int]
    type: str
    asset_id: str
    source_revision: str
    declared_kind: str
    derived_kind: str
    def __init__(self, type: _Optional[str] = ..., asset_id: _Optional[str] = ..., source_revision: _Optional[str] = ..., declared_kind: _Optional[str] = ..., derived_kind: _Optional[str] = ...) -> None: ...

class GetScoreHistoryRequest(_message.Message):
    __slots__ = ("since",)
    SINCE_FIELD_NUMBER: _ClassVar[int]
    since: str
    def __init__(self, since: _Optional[str] = ...) -> None: ...

class GetScoreHistoryResponse(_message.Message):
    __slots__ = ("points",)
    POINTS_FIELD_NUMBER: _ClassVar[int]
    points: _containers.RepeatedCompositeFieldContainer[ScoreHistoryPoint]
    def __init__(self, points: _Optional[_Iterable[_Union[ScoreHistoryPoint, _Mapping]]] = ...) -> None: ...

class HealthNode(_message.Message):
    __slots__ = ("asset", "score", "weight", "health", "staleness_days", "visual_current")
    ASSET_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FIELD_NUMBER: _ClassVar[int]
    HEALTH_FIELD_NUMBER: _ClassVar[int]
    STALENESS_DAYS_FIELD_NUMBER: _ClassVar[int]
    VISUAL_CURRENT_FIELD_NUMBER: _ClassVar[int]
    asset: AssetNode
    score: float
    weight: float
    health: str
    staleness_days: float
    visual_current: bool
    def __init__(self, asset: _Optional[_Union[AssetNode, _Mapping]] = ..., score: _Optional[float] = ..., weight: _Optional[float] = ..., health: _Optional[str] = ..., staleness_days: _Optional[float] = ..., visual_current: _Optional[bool] = ...) -> None: ...

class HealthEdge(_message.Message):
    __slots__ = ("from_asset_id", "to_asset_id", "relation")
    FROM_ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    TO_ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    RELATION_FIELD_NUMBER: _ClassVar[int]
    from_asset_id: str
    to_asset_id: str
    relation: str
    def __init__(self, from_asset_id: _Optional[str] = ..., to_asset_id: _Optional[str] = ..., relation: _Optional[str] = ...) -> None: ...

class GetHealthOverviewRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetHealthOverviewResponse(_message.Message):
    __slots__ = ("coverage", "history", "nodes", "edges", "promote", "quarantined_gates", "kind_mismatch_count", "instrument_moved_count", "kind_mismatches", "run", "config")
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    HISTORY_FIELD_NUMBER: _ClassVar[int]
    NODES_FIELD_NUMBER: _ClassVar[int]
    EDGES_FIELD_NUMBER: _ClassVar[int]
    PROMOTE_FIELD_NUMBER: _ClassVar[int]
    QUARANTINED_GATES_FIELD_NUMBER: _ClassVar[int]
    KIND_MISMATCH_COUNT_FIELD_NUMBER: _ClassVar[int]
    INSTRUMENT_MOVED_COUNT_FIELD_NUMBER: _ClassVar[int]
    KIND_MISMATCHES_FIELD_NUMBER: _ClassVar[int]
    RUN_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    coverage: CoverageReport
    history: _containers.RepeatedCompositeFieldContainer[ScoreHistoryPoint]
    nodes: _containers.RepeatedCompositeFieldContainer[HealthNode]
    edges: _containers.RepeatedCompositeFieldContainer[HealthEdge]
    promote: _containers.RepeatedCompositeFieldContainer[CoverageRow]
    quarantined_gates: _containers.RepeatedScalarFieldContainer[str]
    kind_mismatch_count: int
    instrument_moved_count: int
    kind_mismatches: _containers.RepeatedCompositeFieldContainer[KindMismatch]
    run: ReadinessRun
    config: ReadinessConfig
    def __init__(self, coverage: _Optional[_Union[CoverageReport, _Mapping]] = ..., history: _Optional[_Iterable[_Union[ScoreHistoryPoint, _Mapping]]] = ..., nodes: _Optional[_Iterable[_Union[HealthNode, _Mapping]]] = ..., edges: _Optional[_Iterable[_Union[HealthEdge, _Mapping]]] = ..., promote: _Optional[_Iterable[_Union[CoverageRow, _Mapping]]] = ..., quarantined_gates: _Optional[_Iterable[str]] = ..., kind_mismatch_count: _Optional[int] = ..., instrument_moved_count: _Optional[int] = ..., kind_mismatches: _Optional[_Iterable[_Union[KindMismatch, _Mapping]]] = ..., run: _Optional[_Union[ReadinessRun, _Mapping]] = ..., config: _Optional[_Union[ReadinessConfig, _Mapping]] = ...) -> None: ...

class ReadinessRun(_message.Message):
    __slots__ = ("run_id", "started_at", "completed_at", "completed", "evidence_age")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_AGE_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    started_at: str
    completed_at: str
    completed: bool
    evidence_age: str
    def __init__(self, run_id: _Optional[str] = ..., started_at: _Optional[str] = ..., completed_at: _Optional[str] = ..., completed: _Optional[bool] = ..., evidence_age: _Optional[str] = ...) -> None: ...

class ReadinessConfig(_message.Message):
    __slots__ = ("declared_floor", "achieved_rung", "rung_gap", "blocking_gates", "advisory_gates", "quarantined_gates", "attributable_gates", "corpus_gates")
    DECLARED_FLOOR_FIELD_NUMBER: _ClassVar[int]
    ACHIEVED_RUNG_FIELD_NUMBER: _ClassVar[int]
    RUNG_GAP_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_GATES_FIELD_NUMBER: _ClassVar[int]
    ADVISORY_GATES_FIELD_NUMBER: _ClassVar[int]
    QUARANTINED_GATES_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTABLE_GATES_FIELD_NUMBER: _ClassVar[int]
    CORPUS_GATES_FIELD_NUMBER: _ClassVar[int]
    declared_floor: str
    achieved_rung: str
    rung_gap: int
    blocking_gates: int
    advisory_gates: int
    quarantined_gates: int
    attributable_gates: int
    corpus_gates: int
    def __init__(self, declared_floor: _Optional[str] = ..., achieved_rung: _Optional[str] = ..., rung_gap: _Optional[int] = ..., blocking_gates: _Optional[int] = ..., advisory_gates: _Optional[int] = ..., quarantined_gates: _Optional[int] = ..., attributable_gates: _Optional[int] = ..., corpus_gates: _Optional[int] = ...) -> None: ...

class ReadinessTriageRow(_message.Message):
    __slots__ = ("gate", "asset_id", "message", "nearest_blocking_gate", "blocks_downstream", "weight")
    GATE_FIELD_NUMBER: _ClassVar[int]
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    NEAREST_BLOCKING_GATE_FIELD_NUMBER: _ClassVar[int]
    BLOCKS_DOWNSTREAM_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FIELD_NUMBER: _ClassVar[int]
    gate: str
    asset_id: str
    message: str
    nearest_blocking_gate: str
    blocks_downstream: int
    weight: float
    def __init__(self, gate: _Optional[str] = ..., asset_id: _Optional[str] = ..., message: _Optional[str] = ..., nearest_blocking_gate: _Optional[str] = ..., blocks_downstream: _Optional[int] = ..., weight: _Optional[float] = ...) -> None: ...

class GetReadinessRequest(_message.Message):
    __slots__ = ("floor",)
    FLOOR_FIELD_NUMBER: _ClassVar[int]
    floor: str
    def __init__(self, floor: _Optional[str] = ...) -> None: ...

class GetReadinessResponse(_message.Message):
    __slots__ = ("coverage", "run", "config", "triage", "next_steps", "verdict", "triage_omitted_count")
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    RUN_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    TRIAGE_FIELD_NUMBER: _ClassVar[int]
    NEXT_STEPS_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    TRIAGE_OMITTED_COUNT_FIELD_NUMBER: _ClassVar[int]
    coverage: CoverageReport
    run: ReadinessRun
    config: ReadinessConfig
    triage: _containers.RepeatedCompositeFieldContainer[ReadinessTriageRow]
    next_steps: _containers.RepeatedScalarFieldContainer[str]
    verdict: str
    triage_omitted_count: int
    def __init__(self, coverage: _Optional[_Union[CoverageReport, _Mapping]] = ..., run: _Optional[_Union[ReadinessRun, _Mapping]] = ..., config: _Optional[_Union[ReadinessConfig, _Mapping]] = ..., triage: _Optional[_Iterable[_Union[ReadinessTriageRow, _Mapping]]] = ..., next_steps: _Optional[_Iterable[str]] = ..., verdict: _Optional[str] = ..., triage_omitted_count: _Optional[int] = ...) -> None: ...

class KindMismatch(_message.Message):
    __slots__ = ("asset_id", "declared_kind", "derived_kind", "message")
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    DECLARED_KIND_FIELD_NUMBER: _ClassVar[int]
    DERIVED_KIND_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    declared_kind: str
    derived_kind: str
    message: str
    def __init__(self, asset_id: _Optional[str] = ..., declared_kind: _Optional[str] = ..., derived_kind: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class CaptureEvidenceRequest(_message.Message):
    __slots__ = ("asset_id", "all", "changed_only", "limit", "offset")
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    ALL_FIELD_NUMBER: _ClassVar[int]
    CHANGED_ONLY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    all: bool
    changed_only: bool
    limit: int
    offset: int
    def __init__(self, asset_id: _Optional[str] = ..., all: _Optional[bool] = ..., changed_only: _Optional[bool] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class CaptureEvidenceResponse(_message.Message):
    __slots__ = ("asset_id", "capture_directory", "workbench_url", "rows_written", "missing_contract_assets", "next_offset", "complete")
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_DIRECTORY_FIELD_NUMBER: _ClassVar[int]
    WORKBENCH_URL_FIELD_NUMBER: _ClassVar[int]
    ROWS_WRITTEN_FIELD_NUMBER: _ClassVar[int]
    MISSING_CONTRACT_ASSETS_FIELD_NUMBER: _ClassVar[int]
    NEXT_OFFSET_FIELD_NUMBER: _ClassVar[int]
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    asset_id: str
    capture_directory: str
    workbench_url: str
    rows_written: int
    missing_contract_assets: _containers.RepeatedScalarFieldContainer[str]
    next_offset: int
    complete: bool
    def __init__(self, asset_id: _Optional[str] = ..., capture_directory: _Optional[str] = ..., workbench_url: _Optional[str] = ..., rows_written: _Optional[int] = ..., missing_contract_assets: _Optional[_Iterable[str]] = ..., next_offset: _Optional[int] = ..., complete: _Optional[bool] = ...) -> None: ...
