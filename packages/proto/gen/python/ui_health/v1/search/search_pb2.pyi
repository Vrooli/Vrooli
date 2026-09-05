from ui_health.v1.contracts.provenance import provenance_pb2 as _provenance_pb2
from ui_health.v1.contracts.widget import widget_pb2 as _widget_pb2
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

class SurfaceKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SURFACE_KIND_UNSPECIFIED: _ClassVar[SurfaceKind]
    SURFACE_KIND_COMPONENT: _ClassVar[SurfaceKind]
    SURFACE_KIND_PAGE: _ClassVar[SurfaceKind]
    SURFACE_KIND_FEATURE: _ClassVar[SurfaceKind]
    SURFACE_KIND_HOOK: _ClassVar[SurfaceKind]
    SURFACE_KIND_LAYOUT: _ClassVar[SurfaceKind]
    SURFACE_KIND_OTHER: _ClassVar[SurfaceKind]
MODE_UNSPECIFIED: Mode
MODE_AI: Mode
MODE_TEXT: Mode
SURFACE_KIND_UNSPECIFIED: SurfaceKind
SURFACE_KIND_COMPONENT: SurfaceKind
SURFACE_KIND_PAGE: SurfaceKind
SURFACE_KIND_FEATURE: SurfaceKind
SURFACE_KIND_HOOK: SurfaceKind
SURFACE_KIND_LAYOUT: SurfaceKind
SURFACE_KIND_OTHER: SurfaceKind

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
    __slots__ = ("scenario", "slot", "kind", "display_name", "description", "file_path", "score", "provenance", "widget")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SLOT_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    WIDGET_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    slot: str
    kind: SurfaceKind
    display_name: str
    description: str
    file_path: str
    score: float
    provenance: _provenance_pb2.ComponentProvenance
    widget: _widget_pb2.WidgetDeclaration
    def __init__(self, scenario: _Optional[str] = ..., slot: _Optional[str] = ..., kind: _Optional[_Union[SurfaceKind, str]] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., file_path: _Optional[str] = ..., score: _Optional[float] = ..., provenance: _Optional[_Union[_provenance_pb2.ComponentProvenance, _Mapping]] = ..., widget: _Optional[_Union[_widget_pb2.WidgetDeclaration, _Mapping]] = ...) -> None: ...

class SearchResponse(_message.Message):
    __slots__ = ("results", "mode_used", "indexed_count")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    MODE_USED_FIELD_NUMBER: _ClassVar[int]
    INDEXED_COUNT_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[SearchResult]
    mode_used: Mode
    indexed_count: int
    def __init__(self, results: _Optional[_Iterable[_Union[SearchResult, _Mapping]]] = ..., mode_used: _Optional[_Union[Mode, str]] = ..., indexed_count: _Optional[int] = ...) -> None: ...

class StatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class StatusResponse(_message.Message):
    __slots__ = ("available", "ollama", "qdrant", "indexed_count", "last_reconcile_at", "last_reconcile_outcome", "backends_reachable")
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    OLLAMA_FIELD_NUMBER: _ClassVar[int]
    QDRANT_FIELD_NUMBER: _ClassVar[int]
    INDEXED_COUNT_FIELD_NUMBER: _ClassVar[int]
    LAST_RECONCILE_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_RECONCILE_OUTCOME_FIELD_NUMBER: _ClassVar[int]
    BACKENDS_REACHABLE_FIELD_NUMBER: _ClassVar[int]
    available: bool
    ollama: bool
    qdrant: bool
    indexed_count: int
    last_reconcile_at: str
    last_reconcile_outcome: str
    backends_reachable: bool
    def __init__(self, available: _Optional[bool] = ..., ollama: _Optional[bool] = ..., qdrant: _Optional[bool] = ..., indexed_count: _Optional[int] = ..., last_reconcile_at: _Optional[str] = ..., last_reconcile_outcome: _Optional[str] = ..., backends_reachable: _Optional[bool] = ...) -> None: ...
