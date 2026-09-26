from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Mode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    MODE_UNSPECIFIED: _ClassVar[Mode]
    MODE_AI: _ClassVar[Mode]
    MODE_TEXT: _ClassVar[Mode]
MODE_UNSPECIFIED: Mode
MODE_AI: Mode
MODE_TEXT: Mode

class SearchRequest(_message.Message):
    __slots__ = ("query", "limit", "mode")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    query: str
    limit: int
    mode: Mode
    def __init__(self, query: _Optional[str] = ..., limit: _Optional[int] = ..., mode: _Optional[_Union[Mode, str]] = ...) -> None: ...

class SearchResult(_message.Message):
    __slots__ = ("origin", "group", "name", "description", "score", "source", "full_path", "tags", "binding", "weak")
    ORIGIN_FIELD_NUMBER: _ClassVar[int]
    GROUP_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    FULL_PATH_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    BINDING_FIELD_NUMBER: _ClassVar[int]
    WEAK_FIELD_NUMBER: _ClassVar[int]
    origin: str
    group: str
    name: str
    description: str
    score: float
    source: str
    full_path: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    binding: str
    weak: bool
    def __init__(self, origin: _Optional[str] = ..., group: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., score: _Optional[float] = ..., source: _Optional[str] = ..., full_path: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., binding: _Optional[str] = ..., weak: _Optional[bool] = ...) -> None: ...

class SearchResponse(_message.Message):
    __slots__ = ("results", "mode_used", "reranker")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    MODE_USED_FIELD_NUMBER: _ClassVar[int]
    RERANKER_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[SearchResult]
    mode_used: Mode
    reranker: str
    def __init__(self, results: _Optional[_Iterable[_Union[SearchResult, _Mapping]]] = ..., mode_used: _Optional[_Union[Mode, str]] = ..., reranker: _Optional[str] = ...) -> None: ...

class StatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class StatusResponse(_message.Message):
    __slots__ = ("available", "ollama", "qdrant", "indexed_count", "last_reconcile_at", "last_reconcile_outcome", "reranker")
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    OLLAMA_FIELD_NUMBER: _ClassVar[int]
    QDRANT_FIELD_NUMBER: _ClassVar[int]
    INDEXED_COUNT_FIELD_NUMBER: _ClassVar[int]
    LAST_RECONCILE_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_RECONCILE_OUTCOME_FIELD_NUMBER: _ClassVar[int]
    RERANKER_FIELD_NUMBER: _ClassVar[int]
    available: bool
    ollama: bool
    qdrant: bool
    indexed_count: int
    last_reconcile_at: str
    last_reconcile_outcome: str
    reranker: str
    def __init__(self, available: _Optional[bool] = ..., ollama: _Optional[bool] = ..., qdrant: _Optional[bool] = ..., indexed_count: _Optional[int] = ..., last_reconcile_at: _Optional[str] = ..., last_reconcile_outcome: _Optional[str] = ..., reranker: _Optional[str] = ...) -> None: ...
