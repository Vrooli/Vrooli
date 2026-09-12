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

class SearchResponse(_message.Message):
    __slots__ = ("results", "matcher")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    MATCHER_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[MeasureResult]
    matcher: str
    def __init__(self, results: _Optional[_Iterable[_Union[MeasureResult, _Mapping]]] = ..., matcher: _Optional[str] = ...) -> None: ...

class MeasureResult(_message.Message):
    __slots__ = ("score", "measure")
    SCORE_FIELD_NUMBER: _ClassVar[int]
    MEASURE_FIELD_NUMBER: _ClassVar[int]
    score: float
    measure: MeasureHit
    def __init__(self, score: _Optional[float] = ..., measure: _Optional[_Union[MeasureHit, _Mapping]] = ...) -> None: ...

class MeasureHit(_message.Message):
    __slots__ = ("measure_id", "scenario", "params", "answer", "needs", "effect", "executed_query", "confidence")
    class ParamsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    MEASURE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PARAMS_FIELD_NUMBER: _ClassVar[int]
    ANSWER_FIELD_NUMBER: _ClassVar[int]
    NEEDS_FIELD_NUMBER: _ClassVar[int]
    EFFECT_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    measure_id: str
    scenario: str
    params: _containers.ScalarMap[str, str]
    answer: str
    needs: _containers.RepeatedScalarFieldContainer[str]
    effect: str
    executed_query: str
    confidence: float
    def __init__(self, measure_id: _Optional[str] = ..., scenario: _Optional[str] = ..., params: _Optional[_Mapping[str, str]] = ..., answer: _Optional[str] = ..., needs: _Optional[_Iterable[str]] = ..., effect: _Optional[str] = ..., executed_query: _Optional[str] = ..., confidence: _Optional[float] = ...) -> None: ...

class StatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class StatusResponse(_message.Message):
    __slots__ = ("available", "ollama", "qdrant", "indexed_count", "matcher", "last_indexed_at")
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    OLLAMA_FIELD_NUMBER: _ClassVar[int]
    QDRANT_FIELD_NUMBER: _ClassVar[int]
    INDEXED_COUNT_FIELD_NUMBER: _ClassVar[int]
    MATCHER_FIELD_NUMBER: _ClassVar[int]
    LAST_INDEXED_AT_FIELD_NUMBER: _ClassVar[int]
    available: bool
    ollama: bool
    qdrant: bool
    indexed_count: int
    matcher: str
    last_indexed_at: str
    def __init__(self, available: _Optional[bool] = ..., ollama: _Optional[bool] = ..., qdrant: _Optional[bool] = ..., indexed_count: _Optional[int] = ..., matcher: _Optional[str] = ..., last_indexed_at: _Optional[str] = ...) -> None: ...
