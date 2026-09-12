from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SearchGlossaryRequest(_message.Message):
    __slots__ = ("query",)
    QUERY_FIELD_NUMBER: _ClassVar[int]
    query: str
    def __init__(self, query: _Optional[str] = ...) -> None: ...

class GlossaryEntry(_message.Message):
    __slots__ = ("term", "description", "category")
    TERM_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    term: str
    description: str
    category: str
    def __init__(self, term: _Optional[str] = ..., description: _Optional[str] = ..., category: _Optional[str] = ...) -> None: ...

class SearchGlossaryResponse(_message.Message):
    __slots__ = ("entries", "count", "query")
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[GlossaryEntry]
    count: int
    query: str
    def __init__(self, entries: _Optional[_Iterable[_Union[GlossaryEntry, _Mapping]]] = ..., count: _Optional[int] = ..., query: _Optional[str] = ...) -> None: ...

class SearchConfigurationRequest(_message.Message):
    __slots__ = ("query", "target")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    query: str
    target: str
    def __init__(self, query: _Optional[str] = ..., target: _Optional[str] = ...) -> None: ...

class ConfigurationDescriptor(_message.Message):
    __slots__ = ("id", "title", "purpose", "route", "step_id", "target_kinds", "tags", "prerequisites", "score")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    PURPOSE_FIELD_NUMBER: _ClassVar[int]
    ROUTE_FIELD_NUMBER: _ClassVar[int]
    STEP_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_KINDS_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    PREREQUISITES_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    purpose: str
    route: str
    step_id: str
    target_kinds: _containers.RepeatedScalarFieldContainer[str]
    tags: _containers.RepeatedScalarFieldContainer[str]
    prerequisites: _containers.RepeatedScalarFieldContainer[str]
    score: float
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., purpose: _Optional[str] = ..., route: _Optional[str] = ..., step_id: _Optional[str] = ..., target_kinds: _Optional[_Iterable[str]] = ..., tags: _Optional[_Iterable[str]] = ..., prerequisites: _Optional[_Iterable[str]] = ..., score: _Optional[float] = ...) -> None: ...

class SearchConfigurationResponse(_message.Message):
    __slots__ = ("results", "count", "query", "target", "fallback")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[ConfigurationDescriptor]
    count: int
    query: str
    target: str
    fallback: bool
    def __init__(self, results: _Optional[_Iterable[_Union[ConfigurationDescriptor, _Mapping]]] = ..., count: _Optional[int] = ..., query: _Optional[str] = ..., target: _Optional[str] = ..., fallback: _Optional[bool] = ...) -> None: ...
