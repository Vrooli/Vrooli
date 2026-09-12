from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListSurfacesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class Surface(_message.Message):
    __slots__ = ("id", "name", "kind", "width", "height", "placements", "authority", "confirmed_on")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    PLACEMENTS_FIELD_NUMBER: _ClassVar[int]
    AUTHORITY_FIELD_NUMBER: _ClassVar[int]
    CONFIRMED_ON_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    kind: str
    width: int
    height: int
    placements: _containers.RepeatedScalarFieldContainer[str]
    authority: str
    confirmed_on: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., kind: _Optional[str] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., placements: _Optional[_Iterable[str]] = ..., authority: _Optional[str] = ..., confirmed_on: _Optional[str] = ...) -> None: ...

class ListSurfacesResponse(_message.Message):
    __slots__ = ("surfaces",)
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    surfaces: _containers.RepeatedCompositeFieldContainer[Surface]
    def __init__(self, surfaces: _Optional[_Iterable[_Union[Surface, _Mapping]]] = ...) -> None: ...
