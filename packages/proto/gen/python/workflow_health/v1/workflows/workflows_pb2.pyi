from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SearchWorkflowsRequest(_message.Message):
    __slots__ = ("scenario", "path", "query", "types", "include_fragments", "limit")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    TYPES_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_FRAGMENTS_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    query: str
    types: _containers.RepeatedScalarFieldContainer[str]
    include_fragments: bool
    limit: int
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ..., query: _Optional[str] = ..., types: _Optional[_Iterable[str]] = ..., include_fragments: _Optional[bool] = ..., limit: _Optional[int] = ...) -> None: ...

class SearchWorkflowsResponse(_message.Message):
    __slots__ = ("scenario", "query", "results", "total")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    query: str
    results: _containers.RepeatedCompositeFieldContainer[WorkflowSearchResult]
    total: int
    def __init__(self, scenario: _Optional[str] = ..., query: _Optional[str] = ..., results: _Optional[_Iterable[_Union[WorkflowSearchResult, _Mapping]]] = ..., total: _Optional[int] = ...) -> None: ...

class WorkflowSearchResult(_message.Message):
    __slots__ = ("id", "leaf_type", "asset_type", "role", "title", "snippet", "path", "score", "runnable", "mutating", "requires_confirmation", "requires_isolation", "safety_summary", "requirement_ids", "selectors", "routes", "labels", "dependency_paths", "guardrails")
    ID_FIELD_NUMBER: _ClassVar[int]
    LEAF_TYPE_FIELD_NUMBER: _ClassVar[int]
    ASSET_TYPE_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SNIPPET_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    RUNNABLE_FIELD_NUMBER: _ClassVar[int]
    MUTATING_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_CONFIRMATION_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_ISOLATION_FIELD_NUMBER: _ClassVar[int]
    SAFETY_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    REQUIREMENT_IDS_FIELD_NUMBER: _ClassVar[int]
    SELECTORS_FIELD_NUMBER: _ClassVar[int]
    ROUTES_FIELD_NUMBER: _ClassVar[int]
    LABELS_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCY_PATHS_FIELD_NUMBER: _ClassVar[int]
    GUARDRAILS_FIELD_NUMBER: _ClassVar[int]
    id: str
    leaf_type: str
    asset_type: str
    role: str
    title: str
    snippet: str
    path: str
    score: float
    runnable: bool
    mutating: bool
    requires_confirmation: bool
    requires_isolation: bool
    safety_summary: str
    requirement_ids: _containers.RepeatedScalarFieldContainer[str]
    selectors: _containers.RepeatedScalarFieldContainer[str]
    routes: _containers.RepeatedScalarFieldContainer[str]
    labels: _containers.RepeatedScalarFieldContainer[str]
    dependency_paths: _containers.RepeatedScalarFieldContainer[str]
    guardrails: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., leaf_type: _Optional[str] = ..., asset_type: _Optional[str] = ..., role: _Optional[str] = ..., title: _Optional[str] = ..., snippet: _Optional[str] = ..., path: _Optional[str] = ..., score: _Optional[float] = ..., runnable: _Optional[bool] = ..., mutating: _Optional[bool] = ..., requires_confirmation: _Optional[bool] = ..., requires_isolation: _Optional[bool] = ..., safety_summary: _Optional[str] = ..., requirement_ids: _Optional[_Iterable[str]] = ..., selectors: _Optional[_Iterable[str]] = ..., routes: _Optional[_Iterable[str]] = ..., labels: _Optional[_Iterable[str]] = ..., dependency_paths: _Optional[_Iterable[str]] = ..., guardrails: _Optional[_Iterable[str]] = ...) -> None: ...
