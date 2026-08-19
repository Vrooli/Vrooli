from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class WorldScale(_message.Message):
    __slots__ = ("agent", "furniture", "decoration", "overlay")
    AGENT_FIELD_NUMBER: _ClassVar[int]
    FURNITURE_FIELD_NUMBER: _ClassVar[int]
    DECORATION_FIELD_NUMBER: _ClassVar[int]
    OVERLAY_FIELD_NUMBER: _ClassVar[int]
    agent: float
    furniture: float
    decoration: float
    overlay: float
    def __init__(self, agent: _Optional[float] = ..., furniture: _Optional[float] = ..., decoration: _Optional[float] = ..., overlay: _Optional[float] = ...) -> None: ...

class GetWorldScaleRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SetWorldScaleRequest(_message.Message):
    __slots__ = ("scale",)
    SCALE_FIELD_NUMBER: _ClassVar[int]
    scale: WorldScale
    def __init__(self, scale: _Optional[_Union[WorldScale, _Mapping]] = ...) -> None: ...
