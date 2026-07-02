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

class SearchResponse(_message.Message):
    __slots__ = ("results", "mode", "degraded_reason")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASON_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[IntentHit]
    mode: Mode
    degraded_reason: str
    def __init__(self, results: _Optional[_Iterable[_Union[IntentHit, _Mapping]]] = ..., mode: _Optional[_Union[Mode, str]] = ..., degraded_reason: _Optional[str] = ...) -> None: ...

class IntentHit(_message.Message):
    __slots__ = ("id", "scenario", "type", "title", "snippet", "anchor", "prd_ref", "score", "weak")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SNIPPET_FIELD_NUMBER: _ClassVar[int]
    ANCHOR_FIELD_NUMBER: _ClassVar[int]
    PRD_REF_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    WEAK_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario: str
    type: str
    title: str
    snippet: str
    anchor: str
    prd_ref: str
    score: float
    weak: bool
    def __init__(self, id: _Optional[str] = ..., scenario: _Optional[str] = ..., type: _Optional[str] = ..., title: _Optional[str] = ..., snippet: _Optional[str] = ..., anchor: _Optional[str] = ..., prd_ref: _Optional[str] = ..., score: _Optional[float] = ..., weak: _Optional[bool] = ...) -> None: ...

class StatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class StatusResponse(_message.Message):
    __slots__ = ("available", "ollama_up", "qdrant_up", "reranker_up", "indexed", "collection", "detail")
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    OLLAMA_UP_FIELD_NUMBER: _ClassVar[int]
    QDRANT_UP_FIELD_NUMBER: _ClassVar[int]
    RERANKER_UP_FIELD_NUMBER: _ClassVar[int]
    INDEXED_FIELD_NUMBER: _ClassVar[int]
    COLLECTION_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    available: bool
    ollama_up: bool
    qdrant_up: bool
    reranker_up: bool
    indexed: int
    collection: str
    detail: str
    def __init__(self, available: _Optional[bool] = ..., ollama_up: _Optional[bool] = ..., qdrant_up: _Optional[bool] = ..., reranker_up: _Optional[bool] = ..., indexed: _Optional[int] = ..., collection: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...
