from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SearchResourceUsageRequest(_message.Message):
    __slots__ = ("query", "limit")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    query: str
    limit: int
    def __init__(self, query: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class SearchResourceUsageResponse(_message.Message):
    __slots__ = ("results",)
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[ResourceUsageHit]
    def __init__(self, results: _Optional[_Iterable[_Union[ResourceUsageHit, _Mapping]]] = ...) -> None: ...

class ResourceUsageHit(_message.Message):
    __slots__ = ("resource", "type", "used_by", "summary", "relevance_score")
    RESOURCE_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    USED_BY_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    RELEVANCE_SCORE_FIELD_NUMBER: _ClassVar[int]
    resource: str
    type: str
    used_by: _containers.RepeatedScalarFieldContainer[str]
    summary: str
    relevance_score: float
    def __init__(self, resource: _Optional[str] = ..., type: _Optional[str] = ..., used_by: _Optional[_Iterable[str]] = ..., summary: _Optional[str] = ..., relevance_score: _Optional[float] = ...) -> None: ...
