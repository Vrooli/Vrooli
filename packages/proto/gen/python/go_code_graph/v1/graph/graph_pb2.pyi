from common.v1 import code_graph_pb2 as _code_graph_pb2
from go_code_graph.v1.rewrite import rewrite_pb2 as _rewrite_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GoNodeKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    GO_NODE_KIND_UNSPECIFIED: _ClassVar[GoNodeKind]
    GO_NODE_KIND_TYPE: _ClassVar[GoNodeKind]
    GO_NODE_KIND_FUNC: _ClassVar[GoNodeKind]
    GO_NODE_KIND_VAR: _ClassVar[GoNodeKind]
    GO_NODE_KIND_CONST: _ClassVar[GoNodeKind]
    GO_NODE_KIND_INTERFACE: _ClassVar[GoNodeKind]
    GO_NODE_KIND_METHOD: _ClassVar[GoNodeKind]
    GO_NODE_KIND_IMPORT_SPEC: _ClassVar[GoNodeKind]
    GO_NODE_KIND_REFERENCE: _ClassVar[GoNodeKind]
    GO_NODE_KIND_CALL: _ClassVar[GoNodeKind]
    GO_NODE_KIND_TYPE_USAGE: _ClassVar[GoNodeKind]
    GO_NODE_KIND_ROUTE_REGISTRATION: _ClassVar[GoNodeKind]

class ExtractionProfile(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EXTRACTION_PROFILE_UNSPECIFIED: _ClassVar[ExtractionProfile]
    EXTRACTION_PROFILE_SEMANTIC: _ClassVar[ExtractionProfile]
    EXTRACTION_PROFILE_STRUCTURAL: _ClassVar[ExtractionProfile]
    EXTRACTION_PROFILE_FULL: _ClassVar[ExtractionProfile]
GO_NODE_KIND_UNSPECIFIED: GoNodeKind
GO_NODE_KIND_TYPE: GoNodeKind
GO_NODE_KIND_FUNC: GoNodeKind
GO_NODE_KIND_VAR: GoNodeKind
GO_NODE_KIND_CONST: GoNodeKind
GO_NODE_KIND_INTERFACE: GoNodeKind
GO_NODE_KIND_METHOD: GoNodeKind
GO_NODE_KIND_IMPORT_SPEC: GoNodeKind
GO_NODE_KIND_REFERENCE: GoNodeKind
GO_NODE_KIND_CALL: GoNodeKind
GO_NODE_KIND_TYPE_USAGE: GoNodeKind
GO_NODE_KIND_ROUTE_REGISTRATION: GoNodeKind
EXTRACTION_PROFILE_UNSPECIFIED: ExtractionProfile
EXTRACTION_PROFILE_SEMANTIC: ExtractionProfile
EXTRACTION_PROFILE_STRUCTURAL: ExtractionProfile
EXTRACTION_PROFILE_FULL: ExtractionProfile

class ExtractRequest(_message.Message):
    __slots__ = ("module_path", "include_vendor", "profile", "package_patterns")
    MODULE_PATH_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_VENDOR_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_PATTERNS_FIELD_NUMBER: _ClassVar[int]
    module_path: str
    include_vendor: bool
    profile: ExtractionProfile
    package_patterns: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, module_path: _Optional[str] = ..., include_vendor: _Optional[bool] = ..., profile: _Optional[_Union[ExtractionProfile, str]] = ..., package_patterns: _Optional[_Iterable[str]] = ...) -> None: ...

class ExtractResponse(_message.Message):
    __slots__ = ("graph", "warnings", "extraction_ms", "graph_hash", "fingerprint_ms", "load_ms", "normalize_ms", "cache_hit", "profile", "omitted_information")
    GRAPH_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    EXTRACTION_MS_FIELD_NUMBER: _ClassVar[int]
    GRAPH_HASH_FIELD_NUMBER: _ClassVar[int]
    FINGERPRINT_MS_FIELD_NUMBER: _ClassVar[int]
    LOAD_MS_FIELD_NUMBER: _ClassVar[int]
    NORMALIZE_MS_FIELD_NUMBER: _ClassVar[int]
    CACHE_HIT_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    OMITTED_INFORMATION_FIELD_NUMBER: _ClassVar[int]
    graph: _code_graph_pb2.CodeGraph
    warnings: _containers.RepeatedCompositeFieldContainer[_code_graph_pb2.CodeGraphWarning]
    extraction_ms: int
    graph_hash: str
    fingerprint_ms: int
    load_ms: int
    normalize_ms: int
    cache_hit: bool
    profile: ExtractionProfile
    omitted_information: _containers.RepeatedCompositeFieldContainer[_code_graph_pb2.CodeGraphOmission]
    def __init__(self, graph: _Optional[_Union[_code_graph_pb2.CodeGraph, _Mapping]] = ..., warnings: _Optional[_Iterable[_Union[_code_graph_pb2.CodeGraphWarning, _Mapping]]] = ..., extraction_ms: _Optional[int] = ..., graph_hash: _Optional[str] = ..., fingerprint_ms: _Optional[int] = ..., load_ms: _Optional[int] = ..., normalize_ms: _Optional[int] = ..., cache_hit: _Optional[bool] = ..., profile: _Optional[_Union[ExtractionProfile, str]] = ..., omitted_information: _Optional[_Iterable[_Union[_code_graph_pb2.CodeGraphOmission, _Mapping]]] = ...) -> None: ...

class RewritePlanRequest(_message.Message):
    __slots__ = ("module_path", "operations")
    MODULE_PATH_FIELD_NUMBER: _ClassVar[int]
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    module_path: str
    operations: _containers.RepeatedCompositeFieldContainer[_rewrite_pb2.Operation]
    def __init__(self, module_path: _Optional[str] = ..., operations: _Optional[_Iterable[_Union[_rewrite_pb2.Operation, _Mapping]]] = ...) -> None: ...

class RewritePlanResponse(_message.Message):
    __slots__ = ("plan_id", "normalized_operations")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    NORMALIZED_OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    normalized_operations: _containers.RepeatedCompositeFieldContainer[_rewrite_pb2.Operation]
    def __init__(self, plan_id: _Optional[str] = ..., normalized_operations: _Optional[_Iterable[_Union[_rewrite_pb2.Operation, _Mapping]]] = ...) -> None: ...

class RewriteApplyRequest(_message.Message):
    __slots__ = ("module_path", "plan_id", "apply")
    MODULE_PATH_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    APPLY_FIELD_NUMBER: _ClassVar[int]
    module_path: str
    plan_id: str
    apply: bool
    def __init__(self, module_path: _Optional[str] = ..., plan_id: _Optional[str] = ..., apply: _Optional[bool] = ...) -> None: ...

class RewriteApplyResponse(_message.Message):
    __slots__ = ("plan_id", "results", "dry_run")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    results: _containers.RepeatedCompositeFieldContainer[_rewrite_pb2.OperationResult]
    dry_run: bool
    def __init__(self, plan_id: _Optional[str] = ..., results: _Optional[_Iterable[_Union[_rewrite_pb2.OperationResult, _Mapping]]] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class ListFixturesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class FixtureInfo(_message.Message):
    __slots__ = ("name", "path", "has_expected")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    HAS_EXPECTED_FIELD_NUMBER: _ClassVar[int]
    name: str
    path: str
    has_expected: bool
    def __init__(self, name: _Optional[str] = ..., path: _Optional[str] = ..., has_expected: _Optional[bool] = ...) -> None: ...

class ListFixturesResponse(_message.Message):
    __slots__ = ("fixtures",)
    FIXTURES_FIELD_NUMBER: _ClassVar[int]
    fixtures: _containers.RepeatedCompositeFieldContainer[FixtureInfo]
    def __init__(self, fixtures: _Optional[_Iterable[_Union[FixtureInfo, _Mapping]]] = ...) -> None: ...

class ValidateFixtureRequest(_message.Message):
    __slots__ = ("name",)
    NAME_FIELD_NUMBER: _ClassVar[int]
    name: str
    def __init__(self, name: _Optional[str] = ...) -> None: ...

class ValidateFixtureResponse(_message.Message):
    __slots__ = ("passed", "diff", "expected_bytes", "actual_bytes", "graph_hash")
    PASSED_FIELD_NUMBER: _ClassVar[int]
    DIFF_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_BYTES_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    GRAPH_HASH_FIELD_NUMBER: _ClassVar[int]
    passed: bool
    diff: str
    expected_bytes: int
    actual_bytes: int
    graph_hash: str
    def __init__(self, passed: _Optional[bool] = ..., diff: _Optional[str] = ..., expected_bytes: _Optional[int] = ..., actual_bytes: _Optional[int] = ..., graph_hash: _Optional[str] = ...) -> None: ...
