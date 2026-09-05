from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Node(_message.Message):
    __slots__ = ("id", "entry_id", "facet_id", "depth", "child_ids", "compaction_score")
    ID_FIELD_NUMBER: _ClassVar[int]
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    FACET_ID_FIELD_NUMBER: _ClassVar[int]
    DEPTH_FIELD_NUMBER: _ClassVar[int]
    CHILD_IDS_FIELD_NUMBER: _ClassVar[int]
    COMPACTION_SCORE_FIELD_NUMBER: _ClassVar[int]
    id: str
    entry_id: str
    facet_id: str
    depth: int
    child_ids: _containers.RepeatedScalarFieldContainer[str]
    compaction_score: float
    def __init__(self, id: _Optional[str] = ..., entry_id: _Optional[str] = ..., facet_id: _Optional[str] = ..., depth: _Optional[int] = ..., child_ids: _Optional[_Iterable[str]] = ..., compaction_score: _Optional[float] = ...) -> None: ...

class RunCompactionPassRequest(_message.Message):
    __slots__ = ("scope", "max_clusters")
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    MAX_CLUSTERS_FIELD_NUMBER: _ClassVar[int]
    scope: str
    max_clusters: int
    def __init__(self, scope: _Optional[str] = ..., max_clusters: _Optional[int] = ...) -> None: ...

class RunCompactionPassResponse(_message.Message):
    __slots__ = ("compacted_count", "eligible_frontier_before", "eligible_frontier_after", "target")
    COMPACTED_COUNT_FIELD_NUMBER: _ClassVar[int]
    ELIGIBLE_FRONTIER_BEFORE_FIELD_NUMBER: _ClassVar[int]
    ELIGIBLE_FRONTIER_AFTER_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    compacted_count: int
    eligible_frontier_before: int
    eligible_frontier_after: int
    target: int
    def __init__(self, compacted_count: _Optional[int] = ..., eligible_frontier_before: _Optional[int] = ..., eligible_frontier_after: _Optional[int] = ..., target: _Optional[int] = ...) -> None: ...

class GetFrontierRequest(_message.Message):
    __slots__ = ("limit", "scope")
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    limit: int
    scope: str
    def __init__(self, limit: _Optional[int] = ..., scope: _Optional[str] = ...) -> None: ...

class GetFrontierResponse(_message.Message):
    __slots__ = ("nodes", "eligible_count", "target")
    NODES_FIELD_NUMBER: _ClassVar[int]
    ELIGIBLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    nodes: _containers.RepeatedCompositeFieldContainer[Node]
    eligible_count: int
    target: int
    def __init__(self, nodes: _Optional[_Iterable[_Union[Node, _Mapping]]] = ..., eligible_count: _Optional[int] = ..., target: _Optional[int] = ...) -> None: ...

class GetNodeRequest(_message.Message):
    __slots__ = ("id", "scope")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    id: str
    scope: str
    def __init__(self, id: _Optional[str] = ..., scope: _Optional[str] = ...) -> None: ...

class GetNodeResponse(_message.Message):
    __slots__ = ("node",)
    NODE_FIELD_NUMBER: _ClassVar[int]
    node: Node
    def __init__(self, node: _Optional[_Union[Node, _Mapping]] = ...) -> None: ...

class RebuildForestRequest(_message.Message):
    __slots__ = ("scope",)
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    scope: str
    def __init__(self, scope: _Optional[str] = ...) -> None: ...

class RebuildForestResponse(_message.Message):
    __slots__ = ("node_count",)
    NODE_COUNT_FIELD_NUMBER: _ClassVar[int]
    node_count: int
    def __init__(self, node_count: _Optional[int] = ...) -> None: ...
