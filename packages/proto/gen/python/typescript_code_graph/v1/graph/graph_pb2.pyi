from common.v1 import code_graph_pb2 as _code_graph_pb2
from typescript_code_graph.v1.rewrite import rewrite_pb2 as _rewrite_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TsNodeKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TS_NODE_KIND_UNSPECIFIED: _ClassVar[TsNodeKind]
    TS_NODE_KIND_MODULE: _ClassVar[TsNodeKind]
    TS_NODE_KIND_COMPONENT: _ClassVar[TsNodeKind]
    TS_NODE_KIND_HOOK: _ClassVar[TsNodeKind]
    TS_NODE_KIND_CLASS: _ClassVar[TsNodeKind]
    TS_NODE_KIND_INTERFACE: _ClassVar[TsNodeKind]
    TS_NODE_KIND_TYPE: _ClassVar[TsNodeKind]
    TS_NODE_KIND_FUNCTION: _ClassVar[TsNodeKind]
    TS_NODE_KIND_VAR: _ClassVar[TsNodeKind]
    TS_NODE_KIND_CONST: _ClassVar[TsNodeKind]
    TS_NODE_KIND_RE_EXPORT: _ClassVar[TsNodeKind]
TS_NODE_KIND_UNSPECIFIED: TsNodeKind
TS_NODE_KIND_MODULE: TsNodeKind
TS_NODE_KIND_COMPONENT: TsNodeKind
TS_NODE_KIND_HOOK: TsNodeKind
TS_NODE_KIND_CLASS: TsNodeKind
TS_NODE_KIND_INTERFACE: TsNodeKind
TS_NODE_KIND_TYPE: TsNodeKind
TS_NODE_KIND_FUNCTION: TsNodeKind
TS_NODE_KIND_VAR: TsNodeKind
TS_NODE_KIND_CONST: TsNodeKind
TS_NODE_KIND_RE_EXPORT: TsNodeKind

class ExtractRequest(_message.Message):
    __slots__ = ("scenario_path",)
    SCENARIO_PATH_FIELD_NUMBER: _ClassVar[int]
    scenario_path: str
    def __init__(self, scenario_path: _Optional[str] = ...) -> None: ...

class ExtractResponse(_message.Message):
    __slots__ = ("graph", "warnings", "extraction_ms", "graph_hash", "sidecar_request_id")
    GRAPH_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    EXTRACTION_MS_FIELD_NUMBER: _ClassVar[int]
    GRAPH_HASH_FIELD_NUMBER: _ClassVar[int]
    SIDECAR_REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    graph: _code_graph_pb2.CodeGraph
    warnings: _containers.RepeatedCompositeFieldContainer[_code_graph_pb2.CodeGraphWarning]
    extraction_ms: int
    graph_hash: str
    sidecar_request_id: str
    def __init__(self, graph: _Optional[_Union[_code_graph_pb2.CodeGraph, _Mapping]] = ..., warnings: _Optional[_Iterable[_Union[_code_graph_pb2.CodeGraphWarning, _Mapping]]] = ..., extraction_ms: _Optional[int] = ..., graph_hash: _Optional[str] = ..., sidecar_request_id: _Optional[str] = ...) -> None: ...

class RewritePlanRequest(_message.Message):
    __slots__ = ("scenario_path", "operations")
    SCENARIO_PATH_FIELD_NUMBER: _ClassVar[int]
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    scenario_path: str
    operations: _containers.RepeatedCompositeFieldContainer[_rewrite_pb2.Operation]
    def __init__(self, scenario_path: _Optional[str] = ..., operations: _Optional[_Iterable[_Union[_rewrite_pb2.Operation, _Mapping]]] = ...) -> None: ...

class RewritePlanResponse(_message.Message):
    __slots__ = ("plan_id", "normalized_operations")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    NORMALIZED_OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    normalized_operations: _containers.RepeatedCompositeFieldContainer[_rewrite_pb2.Operation]
    def __init__(self, plan_id: _Optional[str] = ..., normalized_operations: _Optional[_Iterable[_Union[_rewrite_pb2.Operation, _Mapping]]] = ...) -> None: ...

class RewriteApplyRequest(_message.Message):
    __slots__ = ("scenario_path", "plan_id", "apply")
    SCENARIO_PATH_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    APPLY_FIELD_NUMBER: _ClassVar[int]
    scenario_path: str
    plan_id: str
    apply: bool
    def __init__(self, scenario_path: _Optional[str] = ..., plan_id: _Optional[str] = ..., apply: _Optional[bool] = ...) -> None: ...

class RewriteApplyResponse(_message.Message):
    __slots__ = ("plan_id", "results", "dry_run")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    results: _containers.RepeatedCompositeFieldContainer[_rewrite_pb2.OperationResult]
    dry_run: bool
    def __init__(self, plan_id: _Optional[str] = ..., results: _Optional[_Iterable[_Union[_rewrite_pb2.OperationResult, _Mapping]]] = ..., dry_run: _Optional[bool] = ...) -> None: ...
