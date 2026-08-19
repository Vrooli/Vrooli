from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Position(_message.Message):
    __slots__ = ("x", "y", "z")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    Z_FIELD_NUMBER: _ClassVar[int]
    x: float
    y: float
    z: float
    def __init__(self, x: _Optional[float] = ..., y: _Optional[float] = ..., z: _Optional[float] = ...) -> None: ...

class Seat(_message.Message):
    __slots__ = ("position", "rotation")
    POSITION_FIELD_NUMBER: _ClassVar[int]
    ROTATION_FIELD_NUMBER: _ClassVar[int]
    position: Position
    rotation: float
    def __init__(self, position: _Optional[_Union[Position, _Mapping]] = ..., rotation: _Optional[float] = ...) -> None: ...

class SeatGroup(_message.Message):
    __slots__ = ("furniture_type", "seats")
    FURNITURE_TYPE_FIELD_NUMBER: _ClassVar[int]
    SEATS_FIELD_NUMBER: _ClassVar[int]
    furniture_type: str
    seats: _containers.RepeatedCompositeFieldContainer[Seat]
    def __init__(self, furniture_type: _Optional[str] = ..., seats: _Optional[_Iterable[_Union[Seat, _Mapping]]] = ...) -> None: ...

class WorldSeats(_message.Message):
    __slots__ = ("groups",)
    GROUPS_FIELD_NUMBER: _ClassVar[int]
    groups: _containers.RepeatedCompositeFieldContainer[SeatGroup]
    def __init__(self, groups: _Optional[_Iterable[_Union[SeatGroup, _Mapping]]] = ...) -> None: ...

class GetWorldSeatsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SetWorldSeatsRequest(_message.Message):
    __slots__ = ("seats",)
    SEATS_FIELD_NUMBER: _ClassVar[int]
    seats: WorldSeats
    def __init__(self, seats: _Optional[_Union[WorldSeats, _Mapping]] = ...) -> None: ...
