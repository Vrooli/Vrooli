from portal.v1.shared import common_pb2 as _common_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SuggestRequest(_message.Message):
    __slots__ = ("query", "types", "limit", "group")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    TYPES_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    GROUP_FIELD_NUMBER: _ClassVar[int]
    query: str
    types: _containers.RepeatedScalarFieldContainer[str]
    limit: int
    group: str
    def __init__(self, query: _Optional[str] = ..., types: _Optional[_Iterable[str]] = ..., limit: _Optional[int] = ..., group: _Optional[str] = ...) -> None: ...

class SuggestResponse(_message.Message):
    __slots__ = ("hits", "degraded", "reason", "latency_ms")
    HITS_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    hits: _containers.RepeatedCompositeFieldContainer[_common_pb2.SearchHit]
    degraded: bool
    reason: str
    latency_ms: int
    def __init__(self, hits: _Optional[_Iterable[_Union[_common_pb2.SearchHit, _Mapping]]] = ..., degraded: _Optional[bool] = ..., reason: _Optional[str] = ..., latency_ms: _Optional[int] = ...) -> None: ...
