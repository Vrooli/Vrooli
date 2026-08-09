from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RecallHit(_message.Message):
    __slots__ = ("entry_id", "facet_id", "text", "score", "depth", "node_id", "summary", "span")
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    FACET_ID_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    DEPTH_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    SPAN_FIELD_NUMBER: _ClassVar[int]
    entry_id: str
    facet_id: str
    text: str
    score: float
    depth: int
    node_id: str
    summary: bool
    span: int
    def __init__(self, entry_id: _Optional[str] = ..., facet_id: _Optional[str] = ..., text: _Optional[str] = ..., score: _Optional[float] = ..., depth: _Optional[int] = ..., node_id: _Optional[str] = ..., summary: _Optional[bool] = ..., span: _Optional[int] = ...) -> None: ...

class RecallRequest(_message.Message):
    __slots__ = ("query", "limit", "scope")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    query: str
    limit: int
    scope: str
    def __init__(self, query: _Optional[str] = ..., limit: _Optional[int] = ..., scope: _Optional[str] = ...) -> None: ...

class RecallResponse(_message.Message):
    __slots__ = ("hits",)
    HITS_FIELD_NUMBER: _ClassVar[int]
    hits: _containers.RepeatedCompositeFieldContainer[RecallHit]
    def __init__(self, hits: _Optional[_Iterable[_Union[RecallHit, _Mapping]]] = ...) -> None: ...

class WakeRequest(_message.Message):
    __slots__ = ("line_budget", "scope")
    LINE_BUDGET_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    line_budget: int
    scope: str
    def __init__(self, line_budget: _Optional[int] = ..., scope: _Optional[str] = ...) -> None: ...

class WakeResponse(_message.Message):
    __slots__ = ("hits", "overflow")
    HITS_FIELD_NUMBER: _ClassVar[int]
    OVERFLOW_FIELD_NUMBER: _ClassVar[int]
    hits: _containers.RepeatedCompositeFieldContainer[RecallHit]
    overflow: bool
    def __init__(self, hits: _Optional[_Iterable[_Union[RecallHit, _Mapping]]] = ..., overflow: _Optional[bool] = ...) -> None: ...

class ZoomRequest(_message.Message):
    __slots__ = ("node_id", "scope")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    scope: str
    def __init__(self, node_id: _Optional[str] = ..., scope: _Optional[str] = ...) -> None: ...

class ZoomResponse(_message.Message):
    __slots__ = ("constituents",)
    CONSTITUENTS_FIELD_NUMBER: _ClassVar[int]
    constituents: _containers.RepeatedCompositeFieldContainer[RecallHit]
    def __init__(self, constituents: _Optional[_Iterable[_Union[RecallHit, _Mapping]]] = ...) -> None: ...

class ListSiblingEventsRequest(_message.Message):
    __slots__ = ("entry_id", "scope")
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    entry_id: str
    scope: str
    def __init__(self, entry_id: _Optional[str] = ..., scope: _Optional[str] = ...) -> None: ...

class ListSiblingEventsResponse(_message.Message):
    __slots__ = ("entries",)
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[RecallHit]
    def __init__(self, entries: _Optional[_Iterable[_Union[RecallHit, _Mapping]]] = ...) -> None: ...
