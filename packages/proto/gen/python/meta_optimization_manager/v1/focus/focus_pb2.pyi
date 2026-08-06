from meta_optimization_manager.v1.shared import model_pb2 as _model_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Gap(_message.Message):
    __slots__ = ("id", "projection", "title", "status", "source_cell_id", "notes", "approaches", "follow_ups", "axis", "recurrence", "evidence_source", "evidence_locator", "availability_reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_CELL_ID_FIELD_NUMBER: _ClassVar[int]
    GLOBAL_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    APPROACHES_FIELD_NUMBER: _ClassVar[int]
    FOLLOW_UPS_FIELD_NUMBER: _ClassVar[int]
    AXIS_FIELD_NUMBER: _ClassVar[int]
    RECURRENCE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_SOURCE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_LOCATOR_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    projection: _model_pb2.Projection
    title: str
    status: _model_pb2.CellStatus
    source_cell_id: str
    notes: _containers.RepeatedScalarFieldContainer[str]
    approaches: _containers.RepeatedScalarFieldContainer[str]
    follow_ups: _containers.RepeatedScalarFieldContainer[str]
    axis: _model_pb2.GapAxis
    recurrence: int
    evidence_source: str
    evidence_locator: str
    availability_reason: str
    def __init__(self, id: _Optional[str] = ..., projection: _Optional[_Union[_model_pb2.Projection, str]] = ..., title: _Optional[str] = ..., status: _Optional[_Union[_model_pb2.CellStatus, str]] = ..., source_cell_id: _Optional[str] = ..., notes: _Optional[_Iterable[str]] = ..., approaches: _Optional[_Iterable[str]] = ..., follow_ups: _Optional[_Iterable[str]] = ..., axis: _Optional[_Union[_model_pb2.GapAxis, str]] = ..., recurrence: _Optional[int] = ..., evidence_source: _Optional[str] = ..., evidence_locator: _Optional[str] = ..., availability_reason: _Optional[str] = ..., **kwargs) -> None: ...

class FocusItem(_message.Message):
    __slots__ = ("gap", "impact", "importance", "priority_score", "rationale")
    GAP_FIELD_NUMBER: _ClassVar[int]
    IMPACT_FIELD_NUMBER: _ClassVar[int]
    IMPORTANCE_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_SCORE_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    gap: Gap
    impact: float
    importance: float
    priority_score: float
    rationale: str
    def __init__(self, gap: _Optional[_Union[Gap, _Mapping]] = ..., impact: _Optional[float] = ..., importance: _Optional[float] = ..., priority_score: _Optional[float] = ..., rationale: _Optional[str] = ...) -> None: ...

class GetFocusRequest(_message.Message):
    __slots__ = ("limit", "projection")
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    limit: int
    projection: _model_pb2.Projection
    def __init__(self, limit: _Optional[int] = ..., projection: _Optional[_Union[_model_pb2.Projection, str]] = ...) -> None: ...

class GetFocusResponse(_message.Message):
    __slots__ = ("items",)
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[FocusItem]
    def __init__(self, items: _Optional[_Iterable[_Union[FocusItem, _Mapping]]] = ...) -> None: ...

class ListGapsRequest(_message.Message):
    __slots__ = ("projection", "cell_id", "status")
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    CELL_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    projection: _model_pb2.Projection
    cell_id: str
    status: _model_pb2.CellStatus
    def __init__(self, projection: _Optional[_Union[_model_pb2.Projection, str]] = ..., cell_id: _Optional[str] = ..., status: _Optional[_Union[_model_pb2.CellStatus, str]] = ...) -> None: ...

class ListGapsResponse(_message.Message):
    __slots__ = ("gaps",)
    GAPS_FIELD_NUMBER: _ClassVar[int]
    gaps: _containers.RepeatedCompositeFieldContainer[Gap]
    def __init__(self, gaps: _Optional[_Iterable[_Union[Gap, _Mapping]]] = ...) -> None: ...

class GetGapRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetGapResponse(_message.Message):
    __slots__ = ("gap",)
    GAP_FIELD_NUMBER: _ClassVar[int]
    gap: Gap
    def __init__(self, gap: _Optional[_Union[Gap, _Mapping]] = ...) -> None: ...

class AddGapNoteRequest(_message.Message):
    __slots__ = ("id", "approach")
    ID_FIELD_NUMBER: _ClassVar[int]
    APPROACH_FIELD_NUMBER: _ClassVar[int]
    id: str
    approach: str
    def __init__(self, id: _Optional[str] = ..., approach: _Optional[str] = ...) -> None: ...

class AddGapNoteResponse(_message.Message):
    __slots__ = ("gap",)
    GAP_FIELD_NUMBER: _ClassVar[int]
    gap: Gap
    def __init__(self, gap: _Optional[_Union[Gap, _Mapping]] = ...) -> None: ...
