from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SearchRequest(_message.Message):
    __slots__ = ("query", "limit", "synthesize")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    SYNTHESIZE_FIELD_NUMBER: _ClassVar[int]
    query: str
    limit: int
    synthesize: bool
    def __init__(self, query: _Optional[str] = ..., limit: _Optional[int] = ..., synthesize: _Optional[bool] = ...) -> None: ...

class SearchResult(_message.Message):
    __slots__ = ("url", "title", "snippet", "engine", "score", "category")
    URL_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SNIPPET_FIELD_NUMBER: _ClassVar[int]
    ENGINE_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    url: str
    title: str
    snippet: str
    engine: str
    score: float
    category: str
    def __init__(self, url: _Optional[str] = ..., title: _Optional[str] = ..., snippet: _Optional[str] = ..., engine: _Optional[str] = ..., score: _Optional[float] = ..., category: _Optional[str] = ...) -> None: ...

class Citation(_message.Message):
    __slots__ = ("result_index", "url", "title")
    RESULT_INDEX_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    result_index: int
    url: str
    title: str
    def __init__(self, result_index: _Optional[int] = ..., url: _Optional[str] = ..., title: _Optional[str] = ...) -> None: ...

class Synthesis(_message.Message):
    __slots__ = ("text", "citations", "abstained")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    CITATIONS_FIELD_NUMBER: _ClassVar[int]
    ABSTAINED_FIELD_NUMBER: _ClassVar[int]
    text: str
    citations: _containers.RepeatedCompositeFieldContainer[Citation]
    abstained: bool
    def __init__(self, text: _Optional[str] = ..., citations: _Optional[_Iterable[_Union[Citation, _Mapping]]] = ..., abstained: _Optional[bool] = ...) -> None: ...

class SearchResponse(_message.Message):
    __slots__ = ("results", "synthesis", "cached", "degraded", "degraded_reason")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    SYNTHESIS_FIELD_NUMBER: _ClassVar[int]
    CACHED_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASON_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[SearchResult]
    synthesis: Synthesis
    cached: bool
    degraded: bool
    degraded_reason: str
    def __init__(self, results: _Optional[_Iterable[_Union[SearchResult, _Mapping]]] = ..., synthesis: _Optional[_Union[Synthesis, _Mapping]] = ..., cached: _Optional[bool] = ..., degraded: _Optional[bool] = ..., degraded_reason: _Optional[str] = ...) -> None: ...
