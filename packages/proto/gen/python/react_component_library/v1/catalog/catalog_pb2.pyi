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
    __slots__ = ("asset_id", "name", "domain", "kind", "priority", "bucket", "platform", "target", "achieved", "implementation", "blocks_downstream", "rung", "rung_name", "domain_order")
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
    def __init__(self, asset_id: _Optional[str] = ..., name: _Optional[str] = ..., domain: _Optional[str] = ..., kind: _Optional[str] = ..., priority: _Optional[str] = ..., bucket: _Optional[str] = ..., platform: _Optional[str] = ..., target: _Optional[str] = ..., achieved: _Optional[str] = ..., implementation: _Optional[str] = ..., blocks_downstream: _Optional[int] = ..., rung: _Optional[int] = ..., rung_name: _Optional[str] = ..., domain_order: _Optional[int] = ...) -> None: ...

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
    __slots__ = ("total", "at_or_above_target", "by_rung", "catalog_completion", "mandatory_gate_coverage", "weighted_quality", "production_ready_coverage")
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
    total: int
    at_or_above_target: int
    by_rung: _containers.ScalarMap[str, int]
    catalog_completion: CoverageMetric
    mandatory_gate_coverage: CoverageMetric
    weighted_quality: CoverageMetric
    production_ready_coverage: CoverageMetric
    def __init__(self, total: _Optional[int] = ..., at_or_above_target: _Optional[int] = ..., by_rung: _Optional[_Mapping[str, int]] = ..., catalog_completion: _Optional[_Union[CoverageMetric, _Mapping]] = ..., mandatory_gate_coverage: _Optional[_Union[CoverageMetric, _Mapping]] = ..., weighted_quality: _Optional[_Union[CoverageMetric, _Mapping]] = ..., production_ready_coverage: _Optional[_Union[CoverageMetric, _Mapping]] = ...) -> None: ...

class CoverageMetric(_message.Message):
    __slots__ = ("numerator", "denominator", "ratio")
    NUMERATOR_FIELD_NUMBER: _ClassVar[int]
    DENOMINATOR_FIELD_NUMBER: _ClassVar[int]
    RATIO_FIELD_NUMBER: _ClassVar[int]
    numerator: int
    denominator: int
    ratio: float
    def __init__(self, numerator: _Optional[int] = ..., denominator: _Optional[int] = ..., ratio: _Optional[float] = ...) -> None: ...

class CoverageReport(_message.Message):
    __slots__ = ("rows", "totals", "by_domain", "by_priority", "maturity")
    class TotalsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    ROWS_FIELD_NUMBER: _ClassVar[int]
    TOTALS_FIELD_NUMBER: _ClassVar[int]
    BY_DOMAIN_FIELD_NUMBER: _ClassVar[int]
    BY_PRIORITY_FIELD_NUMBER: _ClassVar[int]
    MATURITY_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[CoverageRow]
    totals: _containers.ScalarMap[str, int]
    by_domain: _containers.RepeatedCompositeFieldContainer[Rollup]
    by_priority: _containers.RepeatedCompositeFieldContainer[Rollup]
    maturity: MaturitySummary
    def __init__(self, rows: _Optional[_Iterable[_Union[CoverageRow, _Mapping]]] = ..., totals: _Optional[_Mapping[str, int]] = ..., by_domain: _Optional[_Iterable[_Union[Rollup, _Mapping]]] = ..., by_priority: _Optional[_Iterable[_Union[Rollup, _Mapping]]] = ..., maturity: _Optional[_Union[MaturitySummary, _Mapping]] = ...) -> None: ...

class GetCoverageResponse(_message.Message):
    __slots__ = ("report",)
    REPORT_FIELD_NUMBER: _ClassVar[int]
    report: CoverageReport
    def __init__(self, report: _Optional[_Union[CoverageReport, _Mapping]] = ...) -> None: ...

class ListNextWorkRequest(_message.Message):
    __slots__ = ("limit",)
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    def __init__(self, limit: _Optional[int] = ...) -> None: ...

class ListNextWorkResponse(_message.Message):
    __slots__ = ("rows", "maturity")
    ROWS_FIELD_NUMBER: _ClassVar[int]
    MATURITY_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[CoverageRow]
    maturity: MaturitySummary
    def __init__(self, rows: _Optional[_Iterable[_Union[CoverageRow, _Mapping]]] = ..., maturity: _Optional[_Union[MaturitySummary, _Mapping]] = ...) -> None: ...

class RunGateRequest(_message.Message):
    __slots__ = ("gate",)
    GATE_FIELD_NUMBER: _ClassVar[int]
    gate: str
    def __init__(self, gate: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("gate", "findings", "inspected_files")
    GATE_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    INSPECTED_FILES_FIELD_NUMBER: _ClassVar[int]
    gate: str
    findings: _containers.RepeatedCompositeFieldContainer[GateFinding]
    inspected_files: int
    def __init__(self, gate: _Optional[str] = ..., findings: _Optional[_Iterable[_Union[GateFinding, _Mapping]]] = ..., inspected_files: _Optional[int] = ...) -> None: ...
