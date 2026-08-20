import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
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

class CellStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CELL_STATUS_UNSPECIFIED: _ClassVar[CellStatus]
    CELL_STATUS_NOW: _ClassVar[CellStatus]
    CELL_STATUS_IN_REACH: _ClassVar[CellStatus]
    CELL_STATUS_MISSING: _ClassVar[CellStatus]

class ConfidenceLevel(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONFIDENCE_LEVEL_UNSPECIFIED: _ClassVar[ConfidenceLevel]
    CONFIDENCE_LEVEL_AUTHORITATIVE: _ClassVar[ConfidenceLevel]
    CONFIDENCE_LEVEL_PARTIAL: _ClassVar[ConfidenceLevel]
    CONFIDENCE_LEVEL_SKETCH: _ClassVar[ConfidenceLevel]
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
CELL_STATUS_UNSPECIFIED: CellStatus
CELL_STATUS_NOW: CellStatus
CELL_STATUS_IN_REACH: CellStatus
CELL_STATUS_MISSING: CellStatus
CONFIDENCE_LEVEL_UNSPECIFIED: ConfidenceLevel
CONFIDENCE_LEVEL_AUTHORITATIVE: ConfidenceLevel
CONFIDENCE_LEVEL_PARTIAL: ConfidenceLevel
CONFIDENCE_LEVEL_SKETCH: ConfidenceLevel

class Confidence(_message.Message):
    __slots__ = ("level", "rationale")
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    level: ConfidenceLevel
    rationale: str
    def __init__(self, level: _Optional[_Union[ConfidenceLevel, str]] = ..., rationale: _Optional[str] = ...) -> None: ...

class Ratio(_message.Message):
    __slots__ = ("value", "confidence", "numerator", "denominator")
    VALUE_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    NUMERATOR_FIELD_NUMBER: _ClassVar[int]
    DENOMINATOR_FIELD_NUMBER: _ClassVar[int]
    value: float
    confidence: Confidence
    numerator: int
    denominator: int
    def __init__(self, value: _Optional[float] = ..., confidence: _Optional[_Union[Confidence, _Mapping]] = ..., numerator: _Optional[int] = ..., denominator: _Optional[int] = ...) -> None: ...

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

class Bar(_message.Message):
    __slots__ = ("id", "cell_ref", "projection", "target_kind", "deadband", "sustain", "actuator", "decision_ref")
    ID_FIELD_NUMBER: _ClassVar[int]
    CELL_REF_FIELD_NUMBER: _ClassVar[int]
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    TARGET_KIND_FIELD_NUMBER: _ClassVar[int]
    DEADBAND_FIELD_NUMBER: _ClassVar[int]
    SUSTAIN_FIELD_NUMBER: _ClassVar[int]
    ACTUATOR_FIELD_NUMBER: _ClassVar[int]
    DECISION_REF_FIELD_NUMBER: _ClassVar[int]
    id: str
    cell_ref: str
    projection: Projection
    target_kind: str
    deadband: str
    sustain: str
    actuator: str
    decision_ref: str
    def __init__(self, id: _Optional[str] = ..., cell_ref: _Optional[str] = ..., projection: _Optional[_Union[Projection, str]] = ..., target_kind: _Optional[str] = ..., deadband: _Optional[str] = ..., sustain: _Optional[str] = ..., actuator: _Optional[str] = ..., decision_ref: _Optional[str] = ...) -> None: ...

class ProjectionCoverage(_message.Message):
    __slots__ = ("projection", "ratio", "now_count", "in_reach_count", "missing_count", "total_cells", "confidence", "computed_at", "available", "unavailable_reason")
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    RATIO_FIELD_NUMBER: _ClassVar[int]
    NOW_COUNT_FIELD_NUMBER: _ClassVar[int]
    IN_REACH_COUNT_FIELD_NUMBER: _ClassVar[int]
    MISSING_COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_CELLS_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_REASON_FIELD_NUMBER: _ClassVar[int]
    projection: Projection
    ratio: Ratio
    now_count: int
    in_reach_count: int
    missing_count: int
    total_cells: int
    confidence: Confidence
    computed_at: _timestamp_pb2.Timestamp
    available: bool
    unavailable_reason: str
    def __init__(self, projection: _Optional[_Union[Projection, str]] = ..., ratio: _Optional[_Union[Ratio, _Mapping]] = ..., now_count: _Optional[int] = ..., in_reach_count: _Optional[int] = ..., missing_count: _Optional[int] = ..., total_cells: _Optional[int] = ..., confidence: _Optional[_Union[Confidence, _Mapping]] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., available: _Optional[bool] = ..., unavailable_reason: _Optional[str] = ...) -> None: ...

class IntegrityFinding(_message.Message):
    __slots__ = ("code", "message", "location", "severity", "decision_ref")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    DECISION_REF_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    location: str
    severity: str
    decision_ref: str
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., location: _Optional[str] = ..., severity: _Optional[str] = ..., decision_ref: _Optional[str] = ...) -> None: ...

class DriftFinding(_message.Message):
    __slots__ = ("code", "cell_ref", "sensor_ref", "message", "source")
    CODE_FIELD_NUMBER: _ClassVar[int]
    CELL_REF_FIELD_NUMBER: _ClassVar[int]
    SENSOR_REF_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    code: str
    cell_ref: str
    sensor_ref: str
    message: str
    source: str
    def __init__(self, code: _Optional[str] = ..., cell_ref: _Optional[str] = ..., sensor_ref: _Optional[str] = ..., message: _Optional[str] = ..., source: _Optional[str] = ...) -> None: ...

class GetCoverageRequest(_message.Message):
    __slots__ = ("projections",)
    PROJECTIONS_FIELD_NUMBER: _ClassVar[int]
    projections: _containers.RepeatedScalarFieldContainer[Projection]
    def __init__(self, projections: _Optional[_Iterable[_Union[Projection, str]]] = ...) -> None: ...

class GetCoverageResponse(_message.Message):
    __slots__ = ("projections", "integrity_findings", "computed_at")
    PROJECTIONS_FIELD_NUMBER: _ClassVar[int]
    INTEGRITY_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    projections: _containers.RepeatedCompositeFieldContainer[ProjectionCoverage]
    integrity_findings: _containers.RepeatedCompositeFieldContainer[IntegrityFinding]
    computed_at: _timestamp_pb2.Timestamp
    def __init__(self, projections: _Optional[_Iterable[_Union[ProjectionCoverage, _Mapping]]] = ..., integrity_findings: _Optional[_Iterable[_Union[IntegrityFinding, _Mapping]]] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListCellsRequest(_message.Message):
    __slots__ = ("projection", "status")
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    projection: Projection
    status: CellStatus
    def __init__(self, projection: _Optional[_Union[Projection, str]] = ..., status: _Optional[_Union[CellStatus, str]] = ...) -> None: ...

class ListCellsResponse(_message.Message):
    __slots__ = ("cells",)
    CELLS_FIELD_NUMBER: _ClassVar[int]
    cells: _containers.RepeatedCompositeFieldContainer[Cell]
    def __init__(self, cells: _Optional[_Iterable[_Union[Cell, _Mapping]]] = ...) -> None: ...

class GetProjectionRequest(_message.Message):
    __slots__ = ("projection",)
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    projection: Projection
    def __init__(self, projection: _Optional[_Union[Projection, str]] = ...) -> None: ...

class GetProjectionResponse(_message.Message):
    __slots__ = ("projection", "cells", "coverage", "bars")
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    CELLS_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    BARS_FIELD_NUMBER: _ClassVar[int]
    projection: Projection
    cells: _containers.RepeatedCompositeFieldContainer[Cell]
    coverage: ProjectionCoverage
    bars: _containers.RepeatedCompositeFieldContainer[Bar]
    def __init__(self, projection: _Optional[_Union[Projection, str]] = ..., cells: _Optional[_Iterable[_Union[Cell, _Mapping]]] = ..., coverage: _Optional[_Union[ProjectionCoverage, _Mapping]] = ..., bars: _Optional[_Iterable[_Union[Bar, _Mapping]]] = ...) -> None: ...

class ValidateSetpointRequest(_message.Message):
    __slots__ = ("include_advisories",)
    INCLUDE_ADVISORIES_FIELD_NUMBER: _ClassVar[int]
    include_advisories: bool
    def __init__(self, include_advisories: _Optional[bool] = ...) -> None: ...

class ValidateSetpointResponse(_message.Message):
    __slots__ = ("ok", "findings")
    OK_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    ok: bool
    findings: _containers.RepeatedCompositeFieldContainer[IntegrityFinding]
    def __init__(self, ok: _Optional[bool] = ..., findings: _Optional[_Iterable[_Union[IntegrityFinding, _Mapping]]] = ...) -> None: ...

class GetDriftRequest(_message.Message):
    __slots__ = ("projection",)
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    projection: Projection
    def __init__(self, projection: _Optional[_Union[Projection, str]] = ...) -> None: ...

class GetDriftResponse(_message.Message):
    __slots__ = ("findings",)
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    findings: _containers.RepeatedCompositeFieldContainer[DriftFinding]
    def __init__(self, findings: _Optional[_Iterable[_Union[DriftFinding, _Mapping]]] = ...) -> None: ...
