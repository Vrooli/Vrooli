from ui_health.v1.contracts.provenance import provenance_pb2 as _provenance_pb2
from ui_health.v1.contracts.widget import widget_pb2 as _widget_pb2
from ui_health.v1.search import search_pb2 as _search_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ScanScenarioRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class SurfaceRecord(_message.Message):
    __slots__ = ("scenario", "slot", "kind", "display_name", "description", "file_path")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SLOT_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    slot: str
    kind: _search_pb2.SurfaceKind
    display_name: str
    description: str
    file_path: str
    def __init__(self, scenario: _Optional[str] = ..., slot: _Optional[str] = ..., kind: _Optional[_Union[_search_pb2.SurfaceKind, str]] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., file_path: _Optional[str] = ...) -> None: ...

class ScanScenarioResponse(_message.Message):
    __slots__ = ("scenario", "provenance", "widgets", "surfaces")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    WIDGETS_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    provenance: _containers.RepeatedCompositeFieldContainer[_provenance_pb2.ComponentProvenance]
    widgets: _containers.RepeatedCompositeFieldContainer[_widget_pb2.WidgetDeclaration]
    surfaces: _containers.RepeatedCompositeFieldContainer[SurfaceRecord]
    def __init__(self, scenario: _Optional[str] = ..., provenance: _Optional[_Iterable[_Union[_provenance_pb2.ComponentProvenance, _Mapping]]] = ..., widgets: _Optional[_Iterable[_Union[_widget_pb2.WidgetDeclaration, _Mapping]]] = ..., surfaces: _Optional[_Iterable[_Union[SurfaceRecord, _Mapping]]] = ...) -> None: ...
