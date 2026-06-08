from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class QueryRequest(_message.Message):
    __slots__ = ("query", "types", "all", "limit", "group", "explain", "overrides", "control_token")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    TYPES_FIELD_NUMBER: _ClassVar[int]
    ALL_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    GROUP_FIELD_NUMBER: _ClassVar[int]
    EXPLAIN_FIELD_NUMBER: _ClassVar[int]
    OVERRIDES_FIELD_NUMBER: _ClassVar[int]
    CONTROL_TOKEN_FIELD_NUMBER: _ClassVar[int]
    query: str
    types: _containers.RepeatedScalarFieldContainer[str]
    all: bool
    limit: int
    group: str
    explain: bool
    overrides: SearchOverrides
    control_token: str
    def __init__(self, query: _Optional[str] = ..., types: _Optional[_Iterable[str]] = ..., all: _Optional[bool] = ..., limit: _Optional[int] = ..., group: _Optional[str] = ..., explain: _Optional[bool] = ..., overrides: _Optional[_Union[SearchOverrides, _Mapping]] = ..., control_token: _Optional[str] = ...) -> None: ...

class SearchOverrides(_message.Message):
    __slots__ = ("rerank_enabled", "rerank_blend", "rerank_shortlist", "floor_max_gap", "floor_hard_floor")
    RERANK_ENABLED_FIELD_NUMBER: _ClassVar[int]
    RERANK_BLEND_FIELD_NUMBER: _ClassVar[int]
    RERANK_SHORTLIST_FIELD_NUMBER: _ClassVar[int]
    FLOOR_MAX_GAP_FIELD_NUMBER: _ClassVar[int]
    FLOOR_HARD_FLOOR_FIELD_NUMBER: _ClassVar[int]
    rerank_enabled: bool
    rerank_blend: bool
    rerank_shortlist: int
    floor_max_gap: float
    floor_hard_floor: float
    def __init__(self, rerank_enabled: _Optional[bool] = ..., rerank_blend: _Optional[bool] = ..., rerank_shortlist: _Optional[int] = ..., floor_max_gap: _Optional[float] = ..., floor_hard_floor: _Optional[float] = ...) -> None: ...

class SearchHit(_message.Message):
    __slots__ = ("provider_id", "provider_group", "type", "id", "title", "snippet", "path", "score", "rerank_score")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_GROUP_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SNIPPET_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    RERANK_SCORE_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    provider_group: str
    type: str
    id: str
    title: str
    snippet: str
    path: str
    score: float
    rerank_score: float
    def __init__(self, provider_id: _Optional[str] = ..., provider_group: _Optional[str] = ..., type: _Optional[str] = ..., id: _Optional[str] = ..., title: _Optional[str] = ..., snippet: _Optional[str] = ..., path: _Optional[str] = ..., score: _Optional[float] = ..., rerank_score: _Optional[float] = ...) -> None: ...

class ProviderResultGroup(_message.Message):
    __slots__ = ("provider_id", "hits", "count", "degraded", "note")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    HITS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    hits: _containers.RepeatedCompositeFieldContainer[SearchHit]
    count: int
    degraded: bool
    note: str
    def __init__(self, provider_id: _Optional[str] = ..., hits: _Optional[_Iterable[_Union[SearchHit, _Mapping]]] = ..., count: _Optional[int] = ..., degraded: _Optional[bool] = ..., note: _Optional[str] = ...) -> None: ...

class QueryResponse(_message.Message):
    __slots__ = ("ranked", "groups", "corpora_searched", "routing_explanation", "reranked", "degraded", "latency_ms")
    RANKED_FIELD_NUMBER: _ClassVar[int]
    GROUPS_FIELD_NUMBER: _ClassVar[int]
    CORPORA_SEARCHED_FIELD_NUMBER: _ClassVar[int]
    ROUTING_EXPLANATION_FIELD_NUMBER: _ClassVar[int]
    RERANKED_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    ranked: _containers.RepeatedCompositeFieldContainer[SearchHit]
    groups: _containers.RepeatedCompositeFieldContainer[ProviderResultGroup]
    corpora_searched: _containers.RepeatedScalarFieldContainer[str]
    routing_explanation: _containers.RepeatedScalarFieldContainer[str]
    reranked: bool
    degraded: bool
    latency_ms: int
    def __init__(self, ranked: _Optional[_Iterable[_Union[SearchHit, _Mapping]]] = ..., groups: _Optional[_Iterable[_Union[ProviderResultGroup, _Mapping]]] = ..., corpora_searched: _Optional[_Iterable[str]] = ..., routing_explanation: _Optional[_Iterable[str]] = ..., reranked: _Optional[bool] = ..., degraded: _Optional[bool] = ..., latency_ms: _Optional[int] = ...) -> None: ...

class StatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ProviderHealth(_message.Message):
    __slots__ = ("provider_id", "reachable", "freshness", "point_count", "degraded")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    REACHABLE_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    POINT_COUNT_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    reachable: bool
    freshness: str
    point_count: int
    degraded: bool
    def __init__(self, provider_id: _Optional[str] = ..., reachable: _Optional[bool] = ..., freshness: _Optional[str] = ..., point_count: _Optional[int] = ..., degraded: _Optional[bool] = ...) -> None: ...

class StatusResponse(_message.Message):
    __slots__ = ("providers", "classifier_available", "reranker_available")
    PROVIDERS_FIELD_NUMBER: _ClassVar[int]
    CLASSIFIER_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    RERANKER_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    providers: _containers.RepeatedCompositeFieldContainer[ProviderHealth]
    classifier_available: bool
    reranker_available: bool
    def __init__(self, providers: _Optional[_Iterable[_Union[ProviderHealth, _Mapping]]] = ..., classifier_available: _Optional[bool] = ..., reranker_available: _Optional[bool] = ...) -> None: ...
