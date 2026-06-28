from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetHealthScoresRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetHealthScoresResponse(_message.Message):
    __slots__ = ("scores",)
    SCORES_FIELD_NUMBER: _ClassVar[int]
    scores: _containers.RepeatedCompositeFieldContainer[HealthScore]
    def __init__(self, scores: _Optional[_Iterable[_Union[HealthScore, _Mapping]]] = ...) -> None: ...

class HealthScore(_message.Message):
    __slots__ = ("node_id", "score", "factors", "messages")
    class FactorsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    FACTORS_FIELD_NUMBER: _ClassVar[int]
    MESSAGES_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    score: float
    factors: _containers.ScalarMap[str, float]
    messages: _containers.RepeatedCompositeFieldContainer[HealthMessage]
    def __init__(self, node_id: _Optional[str] = ..., score: _Optional[float] = ..., factors: _Optional[_Mapping[str, float]] = ..., messages: _Optional[_Iterable[_Union[HealthMessage, _Mapping]]] = ...) -> None: ...

class HealthMessage(_message.Message):
    __slots__ = ("key", "severity", "factor", "summary", "detail", "recommendation", "metric_value", "target")
    KEY_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    FACTOR_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATION_FIELD_NUMBER: _ClassVar[int]
    METRIC_VALUE_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    key: str
    severity: str
    factor: str
    summary: str
    detail: str
    recommendation: str
    metric_value: float
    target: str
    def __init__(self, key: _Optional[str] = ..., severity: _Optional[str] = ..., factor: _Optional[str] = ..., summary: _Optional[str] = ..., detail: _Optional[str] = ..., recommendation: _Optional[str] = ..., metric_value: _Optional[float] = ..., target: _Optional[str] = ...) -> None: ...
