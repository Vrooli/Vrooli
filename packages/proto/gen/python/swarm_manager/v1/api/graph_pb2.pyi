from swarm_manager.v1.domain import graph_pb2 as _graph_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GraphResponse(_message.Message):
    __slots__ = ("nodes", "edges", "meta")
    NODES_FIELD_NUMBER: _ClassVar[int]
    EDGES_FIELD_NUMBER: _ClassVar[int]
    META_FIELD_NUMBER: _ClassVar[int]
    nodes: _containers.RepeatedCompositeFieldContainer[_graph_pb2.GraphNode]
    edges: _containers.RepeatedCompositeFieldContainer[_graph_pb2.GraphEdge]
    meta: _graph_pb2.GraphMeta
    def __init__(self, nodes: _Optional[_Iterable[_Union[_graph_pb2.GraphNode, _Mapping]]] = ..., edges: _Optional[_Iterable[_Union[_graph_pb2.GraphEdge, _Mapping]]] = ..., meta: _Optional[_Union[_graph_pb2.GraphMeta, _Mapping]] = ...) -> None: ...
