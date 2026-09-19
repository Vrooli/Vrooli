from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListPresetsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class Preset(_message.Message):
    __slots__ = ("id", "name", "subject", "parameters")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    subject: str
    parameters: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., subject: _Optional[str] = ..., parameters: _Optional[_Iterable[str]] = ...) -> None: ...

class ListPresetsResponse(_message.Message):
    __slots__ = ("presets",)
    PRESETS_FIELD_NUMBER: _ClassVar[int]
    presets: _containers.RepeatedCompositeFieldContainer[Preset]
    def __init__(self, presets: _Optional[_Iterable[_Union[Preset, _Mapping]]] = ...) -> None: ...

class ReservedRegion(_message.Message):
    __slots__ = ("x", "y", "width", "height")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    x: float
    y: float
    width: float
    height: float
    def __init__(self, x: _Optional[float] = ..., y: _Optional[float] = ..., width: _Optional[float] = ..., height: _Optional[float] = ...) -> None: ...

class RenderRequest(_message.Message):
    __slots__ = ("preset", "width", "height", "seed", "conditioner", "params_json", "reserved_regions")
    PRESET_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    CONDITIONER_FIELD_NUMBER: _ClassVar[int]
    PARAMS_JSON_FIELD_NUMBER: _ClassVar[int]
    RESERVED_REGIONS_FIELD_NUMBER: _ClassVar[int]
    preset: str
    width: int
    height: int
    seed: int
    conditioner: str
    params_json: str
    reserved_regions: _containers.RepeatedCompositeFieldContainer[ReservedRegion]
    def __init__(self, preset: _Optional[str] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., seed: _Optional[int] = ..., conditioner: _Optional[str] = ..., params_json: _Optional[str] = ..., reserved_regions: _Optional[_Iterable[_Union[ReservedRegion, _Mapping]]] = ...) -> None: ...

class RenderResponse(_message.Message):
    __slots__ = ("image_png", "sha256", "width", "height", "conditioner")
    IMAGE_PNG_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    CONDITIONER_FIELD_NUMBER: _ClassVar[int]
    image_png: bytes
    sha256: str
    width: int
    height: int
    conditioner: str
    def __init__(self, image_png: _Optional[bytes] = ..., sha256: _Optional[str] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., conditioner: _Optional[str] = ...) -> None: ...
