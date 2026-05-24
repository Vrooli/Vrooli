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
    NODE_KIND_FILE: _ClassVar[NodeKind]
    NODE_KIND_PACKAGE: _ClassVar[NodeKind]
    NODE_KIND_MODULE: _ClassVar[NodeKind]

class EdgeKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EDGE_KIND_UNSPECIFIED: _ClassVar[EdgeKind]
    EDGE_KIND_IMPORT: _ClassVar[EdgeKind]
    EDGE_KIND_INTRA_PACKAGE_REF: _ClassVar[EdgeKind]
    EDGE_KIND_RE_EXPORT: _ClassVar[EdgeKind]

class CodeGraphWarningKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CODE_GRAPH_WARNING_KIND_UNSPECIFIED: _ClassVar[CodeGraphWarningKind]
    CODE_GRAPH_WARNING_KIND_PARSE_ERROR: _ClassVar[CodeGraphWarningKind]
    CODE_GRAPH_WARNING_KIND_UNRESOLVED_IMPORT: _ClassVar[CodeGraphWarningKind]
    CODE_GRAPH_WARNING_KIND_TYPE_CHECK_FAILURE: _ClassVar[CodeGraphWarningKind]
    CODE_GRAPH_WARNING_KIND_AMBIGUOUS_DECLARATION: _ClassVar[CodeGraphWarningKind]
NODE_KIND_UNSPECIFIED: NodeKind
NODE_KIND_FILE: NodeKind
NODE_KIND_PACKAGE: NodeKind
NODE_KIND_MODULE: NodeKind
EDGE_KIND_UNSPECIFIED: EdgeKind
EDGE_KIND_IMPORT: EdgeKind
EDGE_KIND_INTRA_PACKAGE_REF: EdgeKind
EDGE_KIND_RE_EXPORT: EdgeKind
CODE_GRAPH_WARNING_KIND_UNSPECIFIED: CodeGraphWarningKind
CODE_GRAPH_WARNING_KIND_PARSE_ERROR: CodeGraphWarningKind
CODE_GRAPH_WARNING_KIND_UNRESOLVED_IMPORT: CodeGraphWarningKind
CODE_GRAPH_WARNING_KIND_TYPE_CHECK_FAILURE: CodeGraphWarningKind
CODE_GRAPH_WARNING_KIND_AMBIGUOUS_DECLARATION: CodeGraphWarningKind

class CodeGraph(_message.Message):
    __slots__ = ("nodes", "edges")
    NODES_FIELD_NUMBER: _ClassVar[int]
    EDGES_FIELD_NUMBER: _ClassVar[int]
    nodes: _containers.RepeatedCompositeFieldContainer[CodeGraphNode]
    edges: _containers.RepeatedCompositeFieldContainer[CodeGraphEdge]
    def __init__(self, nodes: _Optional[_Iterable[_Union[CodeGraphNode, _Mapping]]] = ..., edges: _Optional[_Iterable[_Union[CodeGraphEdge, _Mapping]]] = ...) -> None: ...

class CodeGraphNode(_message.Message):
    __slots__ = ("id", "kind", "name", "path", "attributes", "leading_comments")
    class AttributesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTES_FIELD_NUMBER: _ClassVar[int]
    LEADING_COMMENTS_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: NodeKind
    name: str
    path: str
    attributes: _containers.ScalarMap[str, str]
    leading_comments: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[_Union[NodeKind, str]] = ..., name: _Optional[str] = ..., path: _Optional[str] = ..., attributes: _Optional[_Mapping[str, str]] = ..., leading_comments: _Optional[_Iterable[str]] = ...) -> None: ...

class CodeGraphEdge(_message.Message):
    __slots__ = ("id", "kind", "from_node_id", "to_node_id", "attributes")
    class AttributesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    FROM_NODE_ID_FIELD_NUMBER: _ClassVar[int]
    TO_NODE_ID_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTES_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: EdgeKind
    from_node_id: str
    to_node_id: str
    attributes: _containers.ScalarMap[str, str]
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[_Union[EdgeKind, str]] = ..., from_node_id: _Optional[str] = ..., to_node_id: _Optional[str] = ..., attributes: _Optional[_Mapping[str, str]] = ...) -> None: ...

class CodeGraphWarning(_message.Message):
    __slots__ = ("kind", "file", "message")
    KIND_FIELD_NUMBER: _ClassVar[int]
    FILE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    kind: CodeGraphWarningKind
    file: str
    message: str
    def __init__(self, kind: _Optional[_Union[CodeGraphWarningKind, str]] = ..., file: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...
