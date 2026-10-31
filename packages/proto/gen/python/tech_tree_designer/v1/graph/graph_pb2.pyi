from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class NodeKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    NODE_KIND_UNSPECIFIED: _ClassVar[NodeKind]
    NODE_KIND_LIVE: _ClassVar[NodeKind]
    NODE_KIND_PLANNED: _ClassVar[NodeKind]
    NODE_KIND_CAPABILITY: _ClassVar[NodeKind]

class EvidenceSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EVIDENCE_SOURCE_UNSPECIFIED: _ClassVar[EvidenceSource]
    EVIDENCE_SOURCE_PROTO_IMPORT: _ClassVar[EvidenceSource]
    EVIDENCE_SOURCE_GO_IMPORT: _ClassVar[EvidenceSource]
    EVIDENCE_SOURCE_PLANNED_PROTO_IMPORT: _ClassVar[EvidenceSource]
    EVIDENCE_SOURCE_DECOMPOSES: _ClassVar[EvidenceSource]
    EVIDENCE_SOURCE_FULFILLS: _ClassVar[EvidenceSource]

class ExportFormat(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EXPORT_FORMAT_UNSPECIFIED: _ClassVar[ExportFormat]
    EXPORT_FORMAT_DOT: _ClassVar[ExportFormat]
    EXPORT_FORMAT_JSON: _ClassVar[ExportFormat]
    EXPORT_FORMAT_TEXT: _ClassVar[ExportFormat]
NODE_KIND_UNSPECIFIED: NodeKind
NODE_KIND_LIVE: NodeKind
NODE_KIND_PLANNED: NodeKind
NODE_KIND_CAPABILITY: NodeKind
EVIDENCE_SOURCE_UNSPECIFIED: EvidenceSource
EVIDENCE_SOURCE_PROTO_IMPORT: EvidenceSource
EVIDENCE_SOURCE_GO_IMPORT: EvidenceSource
EVIDENCE_SOURCE_PLANNED_PROTO_IMPORT: EvidenceSource
EVIDENCE_SOURCE_DECOMPOSES: EvidenceSource
EVIDENCE_SOURCE_FULFILLS: EvidenceSource
EXPORT_FORMAT_UNSPECIFIED: ExportFormat
EXPORT_FORMAT_DOT: ExportFormat
EXPORT_FORMAT_JSON: ExportFormat
EXPORT_FORMAT_TEXT: ExportFormat

class DescribeTechTreeRequest(_message.Message):
    __slots__ = ("scenario_filter", "limit", "stability_filter", "group_by")
    SCENARIO_FILTER_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    STABILITY_FILTER_FIELD_NUMBER: _ClassVar[int]
    GROUP_BY_FIELD_NUMBER: _ClassVar[int]
    scenario_filter: _containers.RepeatedScalarFieldContainer[str]
    limit: int
    stability_filter: str
    group_by: str
    def __init__(self, scenario_filter: _Optional[_Iterable[str]] = ..., limit: _Optional[int] = ..., stability_filter: _Optional[str] = ..., group_by: _Optional[str] = ...) -> None: ...

class DescribeTechTreeResponse(_message.Message):
    __slots__ = ("graph",)
    GRAPH_FIELD_NUMBER: _ClassVar[int]
    graph: TechTreeGraph
    def __init__(self, graph: _Optional[_Union[TechTreeGraph, _Mapping]] = ...) -> None: ...

class GetNeighborhoodRequest(_message.Message):
    __slots__ = ("scenario", "depth", "scenario_filter")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DEPTH_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FILTER_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    depth: int
    scenario_filter: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., depth: _Optional[int] = ..., scenario_filter: _Optional[_Iterable[str]] = ...) -> None: ...

class FindPathRequest(_message.Message):
    __slots__ = ("from_scenario", "to_scenario", "scenario_filter")
    FROM_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TO_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FILTER_FIELD_NUMBER: _ClassVar[int]
    from_scenario: str
    to_scenario: str
    scenario_filter: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, from_scenario: _Optional[str] = ..., to_scenario: _Optional[str] = ..., scenario_filter: _Optional[_Iterable[str]] = ...) -> None: ...

class ListAncestorsRequest(_message.Message):
    __slots__ = ("scenario", "scenario_filter")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FILTER_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    scenario_filter: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., scenario_filter: _Optional[_Iterable[str]] = ...) -> None: ...

class ExportTechTreeRequest(_message.Message):
    __slots__ = ("format", "scenario_filter", "stability_filter")
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FILTER_FIELD_NUMBER: _ClassVar[int]
    STABILITY_FILTER_FIELD_NUMBER: _ClassVar[int]
    format: ExportFormat
    scenario_filter: _containers.RepeatedScalarFieldContainer[str]
    stability_filter: str
    def __init__(self, format: _Optional[_Union[ExportFormat, str]] = ..., scenario_filter: _Optional[_Iterable[str]] = ..., stability_filter: _Optional[str] = ...) -> None: ...

class ExportTechTreeResponse(_message.Message):
    __slots__ = ("format", "content", "media_type")
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    format: ExportFormat
    content: str
    media_type: str
    def __init__(self, format: _Optional[_Union[ExportFormat, str]] = ..., content: _Optional[str] = ..., media_type: _Optional[str] = ...) -> None: ...

class TechTreeGraph(_message.Message):
    __slots__ = ("nodes", "edges", "errors")
    NODES_FIELD_NUMBER: _ClassVar[int]
    EDGES_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    nodes: _containers.RepeatedCompositeFieldContainer[TechNode]
    edges: _containers.RepeatedCompositeFieldContainer[TechEdge]
    errors: _containers.RepeatedCompositeFieldContainer[GraphError]
    def __init__(self, nodes: _Optional[_Iterable[_Union[TechNode, _Mapping]]] = ..., edges: _Optional[_Iterable[_Union[TechEdge, _Mapping]]] = ..., errors: _Optional[_Iterable[_Union[GraphError, _Mapping]]] = ...) -> None: ...

class TechNode(_message.Message):
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
    kind: NodeKind
    display_name: str
    transport_world: str
    stability: _containers.RepeatedScalarFieldContainer[str]
    sector: str
    tier: str
    parent: str
    def __init__(self, scenario: _Optional[str] = ..., kind: _Optional[_Union[NodeKind, str]] = ..., display_name: _Optional[str] = ..., transport_world: _Optional[str] = ..., stability: _Optional[_Iterable[str]] = ..., sector: _Optional[str] = ..., tier: _Optional[str] = ..., parent: _Optional[str] = ...) -> None: ...

class TechEdge(_message.Message):
    __slots__ = ("from_scenario", "to_scenario", "evidence", "transport_world", "stability")
    FROM_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TO_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    TRANSPORT_WORLD_FIELD_NUMBER: _ClassVar[int]
    STABILITY_FIELD_NUMBER: _ClassVar[int]
    from_scenario: str
    to_scenario: str
    evidence: _containers.RepeatedCompositeFieldContainer[GraphEvidence]
    transport_world: str
    stability: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, from_scenario: _Optional[str] = ..., to_scenario: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[GraphEvidence, _Mapping]]] = ..., transport_world: _Optional[str] = ..., stability: _Optional[_Iterable[str]] = ...) -> None: ...

class GraphEvidence(_message.Message):
    __slots__ = ("source", "import_path", "from_file", "to_file", "path", "analyzer")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    IMPORT_PATH_FIELD_NUMBER: _ClassVar[int]
    FROM_FILE_FIELD_NUMBER: _ClassVar[int]
    TO_FILE_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    ANALYZER_FIELD_NUMBER: _ClassVar[int]
    source: EvidenceSource
    import_path: str
    from_file: str
    to_file: str
    path: str
    analyzer: str
    def __init__(self, source: _Optional[_Union[EvidenceSource, str]] = ..., import_path: _Optional[str] = ..., from_file: _Optional[str] = ..., to_file: _Optional[str] = ..., path: _Optional[str] = ..., analyzer: _Optional[str] = ...) -> None: ...

class GraphError(_message.Message):
    __slots__ = ("source", "scenario", "message")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    source: str
    scenario: str
    message: str
    def __init__(self, source: _Optional[str] = ..., scenario: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...
