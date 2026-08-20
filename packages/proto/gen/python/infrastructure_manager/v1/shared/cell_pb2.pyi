from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Projection(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PROJECTION_UNSPECIFIED: _ClassVar[Projection]
    PROJECTION_SUPERVISION: _ClassVar[Projection]
    PROJECTION_AVAILABILITY: _ClassVar[Projection]
    PROJECTION_RECOVERY: _ClassVar[Projection]
    PROJECTION_CAPACITY: _ClassVar[Projection]
    PROJECTION_HEADROOM: _ClassVar[Projection]
    PROJECTION_DURABILITY: _ClassVar[Projection]
    PROJECTION_ATTRIBUTION: _ClassVar[Projection]
    PROJECTION_VALIDATION_COST: _ClassVar[Projection]
    PROJECTION_AGENT_THROUGHPUT: _ClassVar[Projection]
    PROJECTION_COMMISSIONING: _ClassVar[Projection]
    PROJECTION_SUBSTRATE: _ClassVar[Projection]

class CellStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CELL_STATUS_UNSPECIFIED: _ClassVar[CellStatus]
    CELL_STATUS_NOW: _ClassVar[CellStatus]
    CELL_STATUS_IN_REACH: _ClassVar[CellStatus]
    CELL_STATUS_MISSING: _ClassVar[CellStatus]
PROJECTION_UNSPECIFIED: Projection
PROJECTION_SUPERVISION: Projection
PROJECTION_AVAILABILITY: Projection
PROJECTION_RECOVERY: Projection
PROJECTION_CAPACITY: Projection
PROJECTION_HEADROOM: Projection
PROJECTION_DURABILITY: Projection
PROJECTION_ATTRIBUTION: Projection
PROJECTION_VALIDATION_COST: Projection
PROJECTION_AGENT_THROUGHPUT: Projection
PROJECTION_COMMISSIONING: Projection
PROJECTION_SUBSTRATE: Projection
CELL_STATUS_UNSPECIFIED: CellStatus
CELL_STATUS_NOW: CellStatus
CELL_STATUS_IN_REACH: CellStatus
CELL_STATUS_MISSING: CellStatus

class Cell(_message.Message):
    __slots__ = ("id", "projection", "question", "owner", "leg_unit", "status", "sensor_ref", "gap_opened_on", "gap_open_days", "notes")
    ID_FIELD_NUMBER: _ClassVar[int]
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    QUESTION_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    LEG_UNIT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SENSOR_REF_FIELD_NUMBER: _ClassVar[int]
    GAP_OPENED_ON_FIELD_NUMBER: _ClassVar[int]
    GAP_OPEN_DAYS_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    id: str
    projection: Projection
    question: str
    owner: str
    leg_unit: str
    status: CellStatus
    sensor_ref: str
    gap_opened_on: str
    gap_open_days: int
    notes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., projection: _Optional[_Union[Projection, str]] = ..., question: _Optional[str] = ..., owner: _Optional[str] = ..., leg_unit: _Optional[str] = ..., status: _Optional[_Union[CellStatus, str]] = ..., sensor_ref: _Optional[str] = ..., gap_opened_on: _Optional[str] = ..., gap_open_days: _Optional[int] = ..., notes: _Optional[_Iterable[str]] = ...) -> None: ...
