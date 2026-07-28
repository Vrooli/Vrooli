from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RecallHit(_message.Message):
    __slots__ = ("entry_id", "facet_id", "text", "score", "depth")
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    FACET_ID_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    DEPTH_FIELD_NUMBER: _ClassVar[int]
    entry_id: str
    facet_id: str
    text: str
    score: float
    depth: int
    def __init__(self, entry_id: _Optional[str] = ..., facet_id: _Optional[str] = ..., text: _Optional[str] = ..., score: _Optional[float] = ..., depth: _Optional[int] = ...) -> None: ...

class RecallRequest(_message.Message):
    __slots__ = ("query", "limit")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    query: str
    limit: int
    def __init__(self, query: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class RecallResponse(_message.Message):
    __slots__ = ("hits",)
    HITS_FIELD_NUMBER: _ClassVar[int]
    hits: _containers.RepeatedCompositeFieldContainer[RecallHit]
    def __init__(self, hits: _Optional[_Iterable[_Union[RecallHit, _Mapping]]] = ...) -> None: ...

class WakeRequest(_message.Message):
    __slots__ = ("token_budget",)
    TOKEN_BUDGET_FIELD_NUMBER: _ClassVar[int]
    token_budget: int
    def __init__(self, token_budget: _Optional[int] = ...) -> None: ...

class WakeResponse(_message.Message):
    __slots__ = ("hits",)
    HITS_FIELD_NUMBER: _ClassVar[int]
    hits: _containers.RepeatedCompositeFieldContainer[RecallHit]
    def __init__(self, hits: _Optional[_Iterable[_Union[RecallHit, _Mapping]]] = ...) -> None: ...

class ZoomRequest(_message.Message):
    __slots__ = ("node_id",)
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    def __init__(self, node_id: _Optional[str] = ...) -> None: ...

class ZoomResponse(_message.Message):
    __slots__ = ("constituents",)
    CONSTITUENTS_FIELD_NUMBER: _ClassVar[int]
    constituents: _containers.RepeatedCompositeFieldContainer[RecallHit]
    def __init__(self, constituents: _Optional[_Iterable[_Union[RecallHit, _Mapping]]] = ...) -> None: ...

class ListSiblingEventsRequest(_message.Message):
    __slots__ = ("entry_id",)
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    entry_id: str
    def __init__(self, entry_id: _Optional[str] = ...) -> None: ...

class ListSiblingEventsResponse(_message.Message):
    __slots__ = ("entries",)
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[RecallHit]
    def __init__(self, entries: _Optional[_Iterable[_Union[RecallHit, _Mapping]]] = ...) -> None: ...
