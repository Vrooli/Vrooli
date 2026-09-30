import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from signal_inbox.v1.shared import signals_pb2 as _signals_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SearchFilter(_message.Message):
    __slots__ = ("text", "category_id", "disposition", "source_kind", "captured_after", "captured_before", "page_size", "tags", "page_after")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_ID_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_KIND_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AFTER_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_BEFORE_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    PAGE_AFTER_FIELD_NUMBER: _ClassVar[int]
    text: str
    category_id: str
    disposition: str
    source_kind: str
    captured_after: _timestamp_pb2.Timestamp
    captured_before: _timestamp_pb2.Timestamp
    page_size: int
    tags: _containers.RepeatedScalarFieldContainer[str]
    page_after: str
    def __init__(self, text: _Optional[str] = ..., category_id: _Optional[str] = ..., disposition: _Optional[str] = ..., source_kind: _Optional[str] = ..., captured_after: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., captured_before: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., page_size: _Optional[int] = ..., tags: _Optional[_Iterable[str]] = ..., page_after: _Optional[str] = ...) -> None: ...

class RetrievedSignal(_message.Message):
    __slots__ = ("signal", "category_id", "disposition", "score")
    SIGNAL_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_ID_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    signal: _signals_pb2.Signal
    category_id: str
    disposition: str
    score: float
    def __init__(self, signal: _Optional[_Union[_signals_pb2.Signal, _Mapping]] = ..., category_id: _Optional[str] = ..., disposition: _Optional[str] = ..., score: _Optional[float] = ...) -> None: ...

class SearchRequest(_message.Message):
    __slots__ = ("filter",)
    FILTER_FIELD_NUMBER: _ClassVar[int]
    filter: SearchFilter
    def __init__(self, filter: _Optional[_Union[SearchFilter, _Mapping]] = ...) -> None: ...

class SearchResponse(_message.Message):
    __slots__ = ("results", "next_page_after")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_AFTER_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[RetrievedSignal]
    next_page_after: str
    def __init__(self, results: _Optional[_Iterable[_Union[RetrievedSignal, _Mapping]]] = ..., next_page_after: _Optional[str] = ...) -> None: ...

class AmbientRequest(_message.Message):
    __slots__ = ("category_id", "budget")
    CATEGORY_ID_FIELD_NUMBER: _ClassVar[int]
    BUDGET_FIELD_NUMBER: _ClassVar[int]
    category_id: str
    budget: int
    def __init__(self, category_id: _Optional[str] = ..., budget: _Optional[int] = ...) -> None: ...

class AmbientResponse(_message.Message):
    __slots__ = ("results",)
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[RetrievedSignal]
    def __init__(self, results: _Optional[_Iterable[_Union[RetrievedSignal, _Mapping]]] = ...) -> None: ...
