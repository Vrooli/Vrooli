from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SearchRequest(_message.Message):
    __slots__ = ("query", "limit")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    query: str
    limit: int
    def __init__(self, query: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class SearchResult(_message.Message):
    __slots__ = ("id", "title", "snippet", "score", "kind", "follow_up", "freshness", "historical")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SNIPPET_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    FOLLOW_UP_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    HISTORICAL_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    snippet: str
    score: float
    kind: str
    follow_up: str
    freshness: str
    historical: bool
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., snippet: _Optional[str] = ..., score: _Optional[float] = ..., kind: _Optional[str] = ..., follow_up: _Optional[str] = ..., freshness: _Optional[str] = ..., historical: _Optional[bool] = ...) -> None: ...

class SearchResponse(_message.Message):
    __slots__ = ("results", "generation", "materialized_at")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    MATERIALIZED_AT_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[SearchResult]
    generation: str
    materialized_at: str
    def __init__(self, results: _Optional[_Iterable[_Union[SearchResult, _Mapping]]] = ..., generation: _Optional[str] = ..., materialized_at: _Optional[str] = ...) -> None: ...

class StatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class StatusResponse(_message.Message):
    __slots__ = ("available", "indexed_count", "last_indexed_at", "generation")
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    INDEXED_COUNT_FIELD_NUMBER: _ClassVar[int]
    LAST_INDEXED_AT_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    available: bool
    indexed_count: int
    last_indexed_at: str
    generation: str
    def __init__(self, available: _Optional[bool] = ..., indexed_count: _Optional[int] = ..., last_indexed_at: _Optional[str] = ..., generation: _Optional[str] = ...) -> None: ...
