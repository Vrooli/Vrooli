from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CapabilityKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CAPABILITY_KIND_UNSPECIFIED: _ClassVar[CapabilityKind]
    CAPABILITY_KIND_SECTOR: _ClassVar[CapabilityKind]
    CAPABILITY_KIND_CAPABILITY: _ClassVar[CapabilityKind]
    CAPABILITY_KIND_COMPONENT: _ClassVar[CapabilityKind]
    CAPABILITY_KIND_CAPSTONE: _ClassVar[CapabilityKind]
    CAPABILITY_KIND_SIMULATION: _ClassVar[CapabilityKind]

class CapabilityEdgeType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CAPABILITY_EDGE_TYPE_UNSPECIFIED: _ClassVar[CapabilityEdgeType]
    CAPABILITY_EDGE_TYPE_DECOMPOSES: _ClassVar[CapabilityEdgeType]
    CAPABILITY_EDGE_TYPE_PROGRESSION: _ClassVar[CapabilityEdgeType]
    CAPABILITY_EDGE_TYPE_REQUIRES: _ClassVar[CapabilityEdgeType]

class CoverageState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COVERAGE_STATE_UNSPECIFIED: _ClassVar[CoverageState]
    COVERAGE_STATE_BUILT: _ClassVar[CoverageState]
    COVERAGE_STATE_IN_FLIGHT: _ClassVar[CoverageState]
    COVERAGE_STATE_GAP: _ClassVar[CoverageState]
    COVERAGE_STATE_UNMAPPED: _ClassVar[CoverageState]

class FocusReason(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FOCUS_REASON_UNSPECIFIED: _ClassVar[FocusReason]
    FOCUS_REASON_GAP: _ClassVar[FocusReason]
    FOCUS_REASON_CLOSEST_TO_DONE: _ClassVar[FocusReason]
    FOCUS_REASON_UNMAPPED_SCENARIO: _ClassVar[FocusReason]

class OverlayNodeKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OVERLAY_NODE_KIND_UNSPECIFIED: _ClassVar[OverlayNodeKind]
    OVERLAY_NODE_KIND_LIVE: _ClassVar[OverlayNodeKind]
    OVERLAY_NODE_KIND_PLANNED: _ClassVar[OverlayNodeKind]
    OVERLAY_NODE_KIND_CAPABILITY: _ClassVar[OverlayNodeKind]

class OverlayEvidenceSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OVERLAY_EVIDENCE_SOURCE_UNSPECIFIED: _ClassVar[OverlayEvidenceSource]
    OVERLAY_EVIDENCE_SOURCE_PROTO_IMPORT: _ClassVar[OverlayEvidenceSource]
    OVERLAY_EVIDENCE_SOURCE_GO_IMPORT: _ClassVar[OverlayEvidenceSource]
    OVERLAY_EVIDENCE_SOURCE_PLANNED_PROTO_IMPORT: _ClassVar[OverlayEvidenceSource]
    OVERLAY_EVIDENCE_SOURCE_DECOMPOSES: _ClassVar[OverlayEvidenceSource]
    OVERLAY_EVIDENCE_SOURCE_FULFILLS: _ClassVar[OverlayEvidenceSource]
CAPABILITY_KIND_UNSPECIFIED: CapabilityKind
CAPABILITY_KIND_SECTOR: CapabilityKind
CAPABILITY_KIND_CAPABILITY: CapabilityKind
CAPABILITY_KIND_COMPONENT: CapabilityKind
CAPABILITY_KIND_CAPSTONE: CapabilityKind
CAPABILITY_KIND_SIMULATION: CapabilityKind
CAPABILITY_EDGE_TYPE_UNSPECIFIED: CapabilityEdgeType
CAPABILITY_EDGE_TYPE_DECOMPOSES: CapabilityEdgeType
CAPABILITY_EDGE_TYPE_PROGRESSION: CapabilityEdgeType
CAPABILITY_EDGE_TYPE_REQUIRES: CapabilityEdgeType
COVERAGE_STATE_UNSPECIFIED: CoverageState
COVERAGE_STATE_BUILT: CoverageState
COVERAGE_STATE_IN_FLIGHT: CoverageState
COVERAGE_STATE_GAP: CoverageState
COVERAGE_STATE_UNMAPPED: CoverageState
FOCUS_REASON_UNSPECIFIED: FocusReason
FOCUS_REASON_GAP: FocusReason
FOCUS_REASON_CLOSEST_TO_DONE: FocusReason
FOCUS_REASON_UNMAPPED_SCENARIO: FocusReason
OVERLAY_NODE_KIND_UNSPECIFIED: OverlayNodeKind
OVERLAY_NODE_KIND_LIVE: OverlayNodeKind
OVERLAY_NODE_KIND_PLANNED: OverlayNodeKind
OVERLAY_NODE_KIND_CAPABILITY: OverlayNodeKind
OVERLAY_EVIDENCE_SOURCE_UNSPECIFIED: OverlayEvidenceSource
OVERLAY_EVIDENCE_SOURCE_PROTO_IMPORT: OverlayEvidenceSource
OVERLAY_EVIDENCE_SOURCE_GO_IMPORT: OverlayEvidenceSource
OVERLAY_EVIDENCE_SOURCE_PLANNED_PROTO_IMPORT: OverlayEvidenceSource
OVERLAY_EVIDENCE_SOURCE_DECOMPOSES: OverlayEvidenceSource
OVERLAY_EVIDENCE_SOURCE_FULFILLS: OverlayEvidenceSource

class Capability(_message.Message):
    __slots__ = ("id", "slug", "name", "description", "kind", "parent_id", "sort_order", "importance", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    PARENT_ID_FIELD_NUMBER: _ClassVar[int]
    SORT_ORDER_FIELD_NUMBER: _ClassVar[int]
    IMPORTANCE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    slug: str
    name: str
    description: str
    kind: CapabilityKind
    parent_id: str
    sort_order: int
    importance: float
    created_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., slug: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., kind: _Optional[_Union[CapabilityKind, str]] = ..., parent_id: _Optional[str] = ..., sort_order: _Optional[int] = ..., importance: _Optional[float] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class CapabilityEdge(_message.Message):
    __slots__ = ("from_id", "to_id", "type")
    FROM_ID_FIELD_NUMBER: _ClassVar[int]
    TO_ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    from_id: str
    to_id: str
    type: CapabilityEdgeType
    def __init__(self, from_id: _Optional[str] = ..., to_id: _Optional[str] = ..., type: _Optional[_Union[CapabilityEdgeType, str]] = ...) -> None: ...

class Fulfillment(_message.Message):
    __slots__ = ("capability_id", "scenario_slug", "note", "created_at")
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_SLUG_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    capability_id: str
    scenario_slug: str
    note: str
    created_at: str
    def __init__(self, capability_id: _Optional[str] = ..., scenario_slug: _Optional[str] = ..., note: _Optional[str] = ..., created_at: _Optional[str] = ...) -> None: ...

class ListCapabilitiesRequest(_message.Message):
    __slots__ = ("parent_id", "kind", "include_descendants")
    PARENT_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_DESCENDANTS_FIELD_NUMBER: _ClassVar[int]
    parent_id: str
    kind: CapabilityKind
    include_descendants: bool
    def __init__(self, parent_id: _Optional[str] = ..., kind: _Optional[_Union[CapabilityKind, str]] = ..., include_descendants: _Optional[bool] = ...) -> None: ...

class ListCapabilitiesResponse(_message.Message):
    __slots__ = ("capabilities",)
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    capabilities: _containers.RepeatedCompositeFieldContainer[Capability]
    def __init__(self, capabilities: _Optional[_Iterable[_Union[Capability, _Mapping]]] = ...) -> None: ...

class GetCapabilityRequest(_message.Message):
    __slots__ = ("id", "slug")
    ID_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    id: str
    slug: str
    def __init__(self, id: _Optional[str] = ..., slug: _Optional[str] = ...) -> None: ...

class UpsertCapabilityRequest(_message.Message):
    __slots__ = ("capability",)
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    capability: Capability
    def __init__(self, capability: _Optional[_Union[Capability, _Mapping]] = ...) -> None: ...

class DeleteCapabilityRequest(_message.Message):
    __slots__ = ("id", "slug")
    ID_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    id: str
    slug: str
    def __init__(self, id: _Optional[str] = ..., slug: _Optional[str] = ...) -> None: ...

class DeleteCapabilityResponse(_message.Message):
    __slots__ = ("deleted",)
    DELETED_FIELD_NUMBER: _ClassVar[int]
    deleted: bool
    def __init__(self, deleted: _Optional[bool] = ...) -> None: ...

class UpsertCapabilityEdgeRequest(_message.Message):
    __slots__ = ("edge",)
    EDGE_FIELD_NUMBER: _ClassVar[int]
    edge: CapabilityEdge
    def __init__(self, edge: _Optional[_Union[CapabilityEdge, _Mapping]] = ...) -> None: ...

class DeleteCapabilityEdgeRequest(_message.Message):
    __slots__ = ("edge",)
    EDGE_FIELD_NUMBER: _ClassVar[int]
    edge: CapabilityEdge
    def __init__(self, edge: _Optional[_Union[CapabilityEdge, _Mapping]] = ...) -> None: ...

class DeleteCapabilityEdgeResponse(_message.Message):
    __slots__ = ("deleted",)
    DELETED_FIELD_NUMBER: _ClassVar[int]
    deleted: bool
    def __init__(self, deleted: _Optional[bool] = ...) -> None: ...

class ImportTopologyRequest(_message.Message):
    __slots__ = ("json",)
    JSON_FIELD_NUMBER: _ClassVar[int]
    json: str
    def __init__(self, json: _Optional[str] = ...) -> None: ...

class ImportTopologyResponse(_message.Message):
    __slots__ = ("sectors_imported", "capabilities_imported", "edges_imported", "sectors_total", "capabilities_total", "edges_total")
    SECTORS_IMPORTED_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_IMPORTED_FIELD_NUMBER: _ClassVar[int]
    EDGES_IMPORTED_FIELD_NUMBER: _ClassVar[int]
    SECTORS_TOTAL_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_TOTAL_FIELD_NUMBER: _ClassVar[int]
    EDGES_TOTAL_FIELD_NUMBER: _ClassVar[int]
    sectors_imported: int
    capabilities_imported: int
    edges_imported: int
    sectors_total: int
    capabilities_total: int
    edges_total: int
    def __init__(self, sectors_imported: _Optional[int] = ..., capabilities_imported: _Optional[int] = ..., edges_imported: _Optional[int] = ..., sectors_total: _Optional[int] = ..., capabilities_total: _Optional[int] = ..., edges_total: _Optional[int] = ...) -> None: ...

class LinkFulfillmentRequest(_message.Message):
    __slots__ = ("fulfillment",)
    FULFILLMENT_FIELD_NUMBER: _ClassVar[int]
    fulfillment: Fulfillment
    def __init__(self, fulfillment: _Optional[_Union[Fulfillment, _Mapping]] = ...) -> None: ...

class UnlinkFulfillmentRequest(_message.Message):
    __slots__ = ("capability_id", "scenario_slug")
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_SLUG_FIELD_NUMBER: _ClassVar[int]
    capability_id: str
    scenario_slug: str
    def __init__(self, capability_id: _Optional[str] = ..., scenario_slug: _Optional[str] = ...) -> None: ...

class UnlinkFulfillmentResponse(_message.Message):
    __slots__ = ("deleted",)
    DELETED_FIELD_NUMBER: _ClassVar[int]
    deleted: bool
    def __init__(self, deleted: _Optional[bool] = ...) -> None: ...

class ListFulfillmentsRequest(_message.Message):
    __slots__ = ("capability_id", "scenario_slug")
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_SLUG_FIELD_NUMBER: _ClassVar[int]
    capability_id: str
    scenario_slug: str
    def __init__(self, capability_id: _Optional[str] = ..., scenario_slug: _Optional[str] = ...) -> None: ...

class ListFulfillmentsResponse(_message.Message):
    __slots__ = ("fulfillments",)
    FULFILLMENTS_FIELD_NUMBER: _ClassVar[int]
    fulfillments: _containers.RepeatedCompositeFieldContainer[Fulfillment]
    def __init__(self, fulfillments: _Optional[_Iterable[_Union[Fulfillment, _Mapping]]] = ...) -> None: ...

class GetCoverageRequest(_message.Message):
    __slots__ = ("include_subtree_rollup",)
    INCLUDE_SUBTREE_ROLLUP_FIELD_NUMBER: _ClassVar[int]
    include_subtree_rollup: bool
    def __init__(self, include_subtree_rollup: _Optional[bool] = ...) -> None: ...

class CoverageSummary(_message.Message):
    __slots__ = ("built_capabilities", "inflight_capabilities", "gap_capabilities", "unmapped_scenarios", "total_capabilities", "total_scenarios", "ontology_completeness", "implementation_situatedness", "sectors", "classifications", "graph_error")
    BUILT_CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    INFLIGHT_CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    GAP_CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    UNMAPPED_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    ONTOLOGY_COMPLETENESS_FIELD_NUMBER: _ClassVar[int]
    IMPLEMENTATION_SITUATEDNESS_FIELD_NUMBER: _ClassVar[int]
    SECTORS_FIELD_NUMBER: _ClassVar[int]
    CLASSIFICATIONS_FIELD_NUMBER: _ClassVar[int]
    GRAPH_ERROR_FIELD_NUMBER: _ClassVar[int]
    built_capabilities: int
    inflight_capabilities: int
    gap_capabilities: int
    unmapped_scenarios: int
    total_capabilities: int
    total_scenarios: int
    ontology_completeness: float
    implementation_situatedness: float
    sectors: _containers.RepeatedCompositeFieldContainer[SectorCoverage]
    classifications: _containers.RepeatedCompositeFieldContainer[CoverageClassification]
    graph_error: str
    def __init__(self, built_capabilities: _Optional[int] = ..., inflight_capabilities: _Optional[int] = ..., gap_capabilities: _Optional[int] = ..., unmapped_scenarios: _Optional[int] = ..., total_capabilities: _Optional[int] = ..., total_scenarios: _Optional[int] = ..., ontology_completeness: _Optional[float] = ..., implementation_situatedness: _Optional[float] = ..., sectors: _Optional[_Iterable[_Union[SectorCoverage, _Mapping]]] = ..., classifications: _Optional[_Iterable[_Union[CoverageClassification, _Mapping]]] = ..., graph_error: _Optional[str] = ...) -> None: ...

class SectorCoverage(_message.Message):
    __slots__ = ("sector_id", "sector_slug", "sector_name", "built_capabilities", "inflight_capabilities", "gap_capabilities", "total_capabilities", "ontology_completeness")
    SECTOR_ID_FIELD_NUMBER: _ClassVar[int]
    SECTOR_SLUG_FIELD_NUMBER: _ClassVar[int]
    SECTOR_NAME_FIELD_NUMBER: _ClassVar[int]
    BUILT_CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    INFLIGHT_CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    GAP_CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    ONTOLOGY_COMPLETENESS_FIELD_NUMBER: _ClassVar[int]
    sector_id: str
    sector_slug: str
    sector_name: str
    built_capabilities: int
    inflight_capabilities: int
    gap_capabilities: int
    total_capabilities: int
    ontology_completeness: float
    def __init__(self, sector_id: _Optional[str] = ..., sector_slug: _Optional[str] = ..., sector_name: _Optional[str] = ..., built_capabilities: _Optional[int] = ..., inflight_capabilities: _Optional[int] = ..., gap_capabilities: _Optional[int] = ..., total_capabilities: _Optional[int] = ..., ontology_completeness: _Optional[float] = ...) -> None: ...

class CoverageClassification(_message.Message):
    __slots__ = ("capability_id", "capability_slug", "state", "directly_fulfilled", "subtree_covered", "built_scenarios", "planned_scenarios")
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_SLUG_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    DIRECTLY_FULFILLED_FIELD_NUMBER: _ClassVar[int]
    SUBTREE_COVERED_FIELD_NUMBER: _ClassVar[int]
    BUILT_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    PLANNED_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    capability_id: str
    capability_slug: str
    state: CoverageState
    directly_fulfilled: bool
    subtree_covered: bool
    built_scenarios: _containers.RepeatedScalarFieldContainer[str]
    planned_scenarios: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, capability_id: _Optional[str] = ..., capability_slug: _Optional[str] = ..., state: _Optional[_Union[CoverageState, str]] = ..., directly_fulfilled: _Optional[bool] = ..., subtree_covered: _Optional[bool] = ..., built_scenarios: _Optional[_Iterable[str]] = ..., planned_scenarios: _Optional[_Iterable[str]] = ...) -> None: ...

class ListFocusRequest(_message.Message):
    __slots__ = ("limit",)
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    def __init__(self, limit: _Optional[int] = ...) -> None: ...

class ListFocusResponse(_message.Message):
    __slots__ = ("items",)
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[FocusItem]
    def __init__(self, items: _Optional[_Iterable[_Union[FocusItem, _Mapping]]] = ...) -> None: ...

class FocusItem(_message.Message):
    __slots__ = ("capability_id", "capability_slug", "capability_name", "reason", "score", "downstream_dependents", "related_scenarios")
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_SLUG_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_NAME_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    DOWNSTREAM_DEPENDENTS_FIELD_NUMBER: _ClassVar[int]
    RELATED_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    capability_id: str
    capability_slug: str
    capability_name: str
    reason: FocusReason
    score: float
    downstream_dependents: int
    related_scenarios: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, capability_id: _Optional[str] = ..., capability_slug: _Optional[str] = ..., capability_name: _Optional[str] = ..., reason: _Optional[_Union[FocusReason, str]] = ..., score: _Optional[float] = ..., downstream_dependents: _Optional[int] = ..., related_scenarios: _Optional[_Iterable[str]] = ...) -> None: ...

class GetCapabilityScenariosRequest(_message.Message):
    __slots__ = ("capability_id", "capability_slug", "include_descendants")
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_SLUG_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_DESCENDANTS_FIELD_NUMBER: _ClassVar[int]
    capability_id: str
    capability_slug: str
    include_descendants: bool
    def __init__(self, capability_id: _Optional[str] = ..., capability_slug: _Optional[str] = ..., include_descendants: _Optional[bool] = ...) -> None: ...

class CapabilityScenarios(_message.Message):
    __slots__ = ("capability_id", "capability_slug", "built_scenarios", "planned_scenarios", "fulfillments")
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_SLUG_FIELD_NUMBER: _ClassVar[int]
    BUILT_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    PLANNED_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    FULFILLMENTS_FIELD_NUMBER: _ClassVar[int]
    capability_id: str
    capability_slug: str
    built_scenarios: _containers.RepeatedScalarFieldContainer[str]
    planned_scenarios: _containers.RepeatedScalarFieldContainer[str]
    fulfillments: _containers.RepeatedCompositeFieldContainer[Fulfillment]
    def __init__(self, capability_id: _Optional[str] = ..., capability_slug: _Optional[str] = ..., built_scenarios: _Optional[_Iterable[str]] = ..., planned_scenarios: _Optional[_Iterable[str]] = ..., fulfillments: _Optional[_Iterable[_Union[Fulfillment, _Mapping]]] = ...) -> None: ...

class GetScenarioCapabilitiesRequest(_message.Message):
    __slots__ = ("scenario_slug",)
    SCENARIO_SLUG_FIELD_NUMBER: _ClassVar[int]
    scenario_slug: str
    def __init__(self, scenario_slug: _Optional[str] = ...) -> None: ...

class ScenarioCapabilities(_message.Message):
    __slots__ = ("scenario_slug", "capabilities", "fulfillments")
    SCENARIO_SLUG_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    FULFILLMENTS_FIELD_NUMBER: _ClassVar[int]
    scenario_slug: str
    capabilities: _containers.RepeatedCompositeFieldContainer[Capability]
    fulfillments: _containers.RepeatedCompositeFieldContainer[Fulfillment]
    def __init__(self, scenario_slug: _Optional[str] = ..., capabilities: _Optional[_Iterable[_Union[Capability, _Mapping]]] = ..., fulfillments: _Optional[_Iterable[_Union[Fulfillment, _Mapping]]] = ...) -> None: ...

class DescribeOverlayGraphRequest(_message.Message):
    __slots__ = ("include_implementation", "include_ontology", "include_fulfillment")
    INCLUDE_IMPLEMENTATION_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_ONTOLOGY_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_FULFILLMENT_FIELD_NUMBER: _ClassVar[int]
    include_implementation: bool
    include_ontology: bool
    include_fulfillment: bool
    def __init__(self, include_implementation: _Optional[bool] = ..., include_ontology: _Optional[bool] = ..., include_fulfillment: _Optional[bool] = ...) -> None: ...

class DescribeOverlayGraphResponse(_message.Message):
    __slots__ = ("graph",)
    GRAPH_FIELD_NUMBER: _ClassVar[int]
    graph: OverlayGraph
    def __init__(self, graph: _Optional[_Union[OverlayGraph, _Mapping]] = ...) -> None: ...

class OverlayGraph(_message.Message):
    __slots__ = ("nodes", "edges", "errors")
    NODES_FIELD_NUMBER: _ClassVar[int]
    EDGES_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    nodes: _containers.RepeatedCompositeFieldContainer[OverlayNode]
    edges: _containers.RepeatedCompositeFieldContainer[OverlayEdge]
    errors: _containers.RepeatedCompositeFieldContainer[OverlayError]
    def __init__(self, nodes: _Optional[_Iterable[_Union[OverlayNode, _Mapping]]] = ..., edges: _Optional[_Iterable[_Union[OverlayEdge, _Mapping]]] = ..., errors: _Optional[_Iterable[_Union[OverlayError, _Mapping]]] = ...) -> None: ...

class OverlayNode(_message.Message):
    __slots__ = ("scenario", "kind", "display_name", "transport_world", "stability", "sector", "tier", "parent")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    TRANSPORT_WORLD_FIELD_NUMBER: _ClassVar[int]
    STABILITY_FIELD_NUMBER: _ClassVar[int]
    SECTOR_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    PARENT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    kind: OverlayNodeKind
    display_name: str
    transport_world: str
    stability: _containers.RepeatedScalarFieldContainer[str]
    sector: str
    tier: str
    parent: str
    def __init__(self, scenario: _Optional[str] = ..., kind: _Optional[_Union[OverlayNodeKind, str]] = ..., display_name: _Optional[str] = ..., transport_world: _Optional[str] = ..., stability: _Optional[_Iterable[str]] = ..., sector: _Optional[str] = ..., tier: _Optional[str] = ..., parent: _Optional[str] = ...) -> None: ...

class OverlayEdge(_message.Message):
    __slots__ = ("from_scenario", "to_scenario", "evidence", "transport_world", "stability")
    FROM_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TO_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    TRANSPORT_WORLD_FIELD_NUMBER: _ClassVar[int]
    STABILITY_FIELD_NUMBER: _ClassVar[int]
    from_scenario: str
    to_scenario: str
    evidence: _containers.RepeatedCompositeFieldContainer[OverlayEvidence]
    transport_world: str
    stability: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, from_scenario: _Optional[str] = ..., to_scenario: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[OverlayEvidence, _Mapping]]] = ..., transport_world: _Optional[str] = ..., stability: _Optional[_Iterable[str]] = ...) -> None: ...

class OverlayEvidence(_message.Message):
    __slots__ = ("source", "import_path", "from_file", "to_file", "path", "analyzer")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    IMPORT_PATH_FIELD_NUMBER: _ClassVar[int]
    FROM_FILE_FIELD_NUMBER: _ClassVar[int]
    TO_FILE_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    ANALYZER_FIELD_NUMBER: _ClassVar[int]
    source: OverlayEvidenceSource
    import_path: str
    from_file: str
    to_file: str
    path: str
    analyzer: str
    def __init__(self, source: _Optional[_Union[OverlayEvidenceSource, str]] = ..., import_path: _Optional[str] = ..., from_file: _Optional[str] = ..., to_file: _Optional[str] = ..., path: _Optional[str] = ..., analyzer: _Optional[str] = ...) -> None: ...

class OverlayError(_message.Message):
    __slots__ = ("source", "scenario", "message")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    source: str
    scenario: str
    message: str
    def __init__(self, source: _Optional[str] = ..., scenario: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...
