from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EvidenceSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EVIDENCE_SOURCE_UNSPECIFIED: _ClassVar[EvidenceSource]
    EVIDENCE_SOURCE_PROTO_IMPORT: _ClassVar[EvidenceSource]
    EVIDENCE_SOURCE_GO_IMPORT: _ClassVar[EvidenceSource]
EVIDENCE_SOURCE_UNSPECIFIED: EvidenceSource
EVIDENCE_SOURCE_PROTO_IMPORT: EvidenceSource
EVIDENCE_SOURCE_GO_IMPORT: EvidenceSource

class DescribeInterfaceGraphRequest(_message.Message):
    __slots__ = ("scenarios", "limit", "stability_filter", "language_filter", "max_scenario_hops")
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    STABILITY_FILTER_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FILTER_FIELD_NUMBER: _ClassVar[int]
    MAX_SCENARIO_HOPS_FIELD_NUMBER: _ClassVar[int]
    scenarios: _containers.RepeatedScalarFieldContainer[str]
    limit: int
    stability_filter: str
    language_filter: _containers.RepeatedScalarFieldContainer[str]
    max_scenario_hops: int
    def __init__(self, scenarios: _Optional[_Iterable[str]] = ..., limit: _Optional[int] = ..., stability_filter: _Optional[str] = ..., language_filter: _Optional[_Iterable[str]] = ..., max_scenario_hops: _Optional[int] = ...) -> None: ...

class DescribeInterfaceGraphResponse(_message.Message):
    __slots__ = ("graph", "computed_at")
    GRAPH_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    graph: InterfaceGraph
    computed_at: str
    def __init__(self, graph: _Optional[_Union[InterfaceGraph, _Mapping]] = ..., computed_at: _Optional[str] = ...) -> None: ...

class InterfaceGraph(_message.Message):
    __slots__ = ("nodes", "edges", "errors")
    NODES_FIELD_NUMBER: _ClassVar[int]
    EDGES_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    nodes: _containers.RepeatedCompositeFieldContainer[GraphNode]
    edges: _containers.RepeatedCompositeFieldContainer[GraphEdge]
    errors: _containers.RepeatedCompositeFieldContainer[GraphError]
    def __init__(self, nodes: _Optional[_Iterable[_Union[GraphNode, _Mapping]]] = ..., edges: _Optional[_Iterable[_Union[GraphEdge, _Mapping]]] = ..., errors: _Optional[_Iterable[_Union[GraphError, _Mapping]]] = ...) -> None: ...

class GraphNode(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class GraphEdge(_message.Message):
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
