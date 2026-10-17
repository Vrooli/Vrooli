from web_search.v1.shared import search_pb2 as _search_pb2
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
    __slots__ = ("results", "synthesis", "cached", "degraded", "degraded_reason", "degraded_engines")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    SYNTHESIS_FIELD_NUMBER: _ClassVar[int]
    CACHED_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASON_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_ENGINES_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[_search_pb2.SearchResult]
    synthesis: Synthesis
    cached: bool
    degraded: bool
    degraded_reason: str
    degraded_engines: _containers.RepeatedCompositeFieldContainer[_search_pb2.EngineIssue]
    def __init__(self, results: _Optional[_Iterable[_Union[_search_pb2.SearchResult, _Mapping]]] = ..., synthesis: _Optional[_Union[Synthesis, _Mapping]] = ..., cached: _Optional[bool] = ..., degraded: _Optional[bool] = ..., degraded_reason: _Optional[str] = ..., degraded_engines: _Optional[_Iterable[_Union[_search_pb2.EngineIssue, _Mapping]]] = ...) -> None: ...
