import datetime

from common.v1 import attestation_pb2 as _attestation_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from meta_optimization_manager.v1.shared import model_pb2 as _model_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ProjectionCoverage(_message.Message):
    __slots__ = ("projection", "now_count", "in_reach_count", "missing_count", "total_cells", "coverage_ratio", "denominator_confidence", "confidence_rationale", "available", "unavailable_reason", "condition_counts")
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    NOW_COUNT_FIELD_NUMBER: _ClassVar[int]
    IN_REACH_COUNT_FIELD_NUMBER: _ClassVar[int]
    MISSING_COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_CELLS_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_RATIO_FIELD_NUMBER: _ClassVar[int]
    DENOMINATOR_CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_RATIONALE_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_REASON_FIELD_NUMBER: _ClassVar[int]
    CONDITION_COUNTS_FIELD_NUMBER: _ClassVar[int]
    projection: _model_pb2.Projection
    now_count: int
    in_reach_count: int
    missing_count: int
    total_cells: int
    coverage_ratio: float
    denominator_confidence: _model_pb2.DenominatorConfidence
    confidence_rationale: str
    available: bool
    unavailable_reason: str
    condition_counts: _containers.RepeatedCompositeFieldContainer[ConditionCount]
    def __init__(self, projection: _Optional[_Union[_model_pb2.Projection, str]] = ..., now_count: _Optional[int] = ..., in_reach_count: _Optional[int] = ..., missing_count: _Optional[int] = ..., total_cells: _Optional[int] = ..., coverage_ratio: _Optional[float] = ..., denominator_confidence: _Optional[_Union[_model_pb2.DenominatorConfidence, str]] = ..., confidence_rationale: _Optional[str] = ..., available: _Optional[bool] = ..., unavailable_reason: _Optional[str] = ..., condition_counts: _Optional[_Iterable[_Union[ConditionCount, _Mapping]]] = ...) -> None: ...

class ConditionCount(_message.Message):
    __slots__ = ("condition", "count")
    CONDITION_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    condition: str
    count: int
    def __init__(self, condition: _Optional[str] = ..., count: _Optional[int] = ...) -> None: ...

class EmpiricalTrendPoint(_message.Message):
    __slots__ = ("success_rate", "median_tokens", "median_duration_ms", "at")
    SUCCESS_RATE_FIELD_NUMBER: _ClassVar[int]
    MEDIAN_TOKENS_FIELD_NUMBER: _ClassVar[int]
    MEDIAN_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    AT_FIELD_NUMBER: _ClassVar[int]
    success_rate: float
    median_tokens: int
    median_duration_ms: int
    at: _timestamp_pb2.Timestamp
    def __init__(self, success_rate: _Optional[float] = ..., median_tokens: _Optional[int] = ..., median_duration_ms: _Optional[int] = ..., at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetStatusRequest(_message.Message):
    __slots__ = ("projection",)
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    projection: _model_pb2.Projection
    def __init__(self, projection: _Optional[_Union[_model_pb2.Projection, str]] = ...) -> None: ...

class GetStatusResponse(_message.Message):
    __slots__ = ("projections", "latest_trial_trend", "computed_at", "determinism_checked", "deterministic", "determinism_evidence", "deltas")
    PROJECTIONS_FIELD_NUMBER: _ClassVar[int]
    LATEST_TRIAL_TREND_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    DETERMINISM_CHECKED_FIELD_NUMBER: _ClassVar[int]
    DETERMINISTIC_FIELD_NUMBER: _ClassVar[int]
    DETERMINISM_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    DELTAS_FIELD_NUMBER: _ClassVar[int]
    projections: _containers.RepeatedCompositeFieldContainer[ProjectionCoverage]
    latest_trial_trend: EmpiricalTrendPoint
    computed_at: _timestamp_pb2.Timestamp
    determinism_checked: bool
    deterministic: bool
    determinism_evidence: str
    deltas: _containers.RepeatedCompositeFieldContainer[ProjectionDelta]
    def __init__(self, projections: _Optional[_Iterable[_Union[ProjectionCoverage, _Mapping]]] = ..., latest_trial_trend: _Optional[_Union[EmpiricalTrendPoint, _Mapping]] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., determinism_checked: _Optional[bool] = ..., deterministic: _Optional[bool] = ..., determinism_evidence: _Optional[str] = ..., deltas: _Optional[_Iterable[_Union[ProjectionDelta, _Mapping]]] = ...) -> None: ...

class ProjectionDelta(_message.Message):
    __slots__ = ("projection", "previous_ratio", "current_ratio", "delta")
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_RATIO_FIELD_NUMBER: _ClassVar[int]
    CURRENT_RATIO_FIELD_NUMBER: _ClassVar[int]
    DELTA_FIELD_NUMBER: _ClassVar[int]
    projection: _model_pb2.Projection
    previous_ratio: float
    current_ratio: float
    delta: float
    def __init__(self, projection: _Optional[_Union[_model_pb2.Projection, str]] = ..., previous_ratio: _Optional[float] = ..., current_ratio: _Optional[float] = ..., delta: _Optional[float] = ...) -> None: ...

class Citation(_message.Message):
    __slots__ = ("locator", "kind", "note")
    LOCATOR_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    locator: str
    kind: str
    note: str
    def __init__(self, locator: _Optional[str] = ..., kind: _Optional[str] = ..., note: _Optional[str] = ...) -> None: ...

class Cell(_message.Message):
    __slots__ = ("id", "projection", "question", "owner", "status", "basis", "sufficiency", "notes", "citations", "condition")
    ID_FIELD_NUMBER: _ClassVar[int]
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    QUESTION_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    BASIS_FIELD_NUMBER: _ClassVar[int]
    SUFFICIENCY_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    CITATIONS_FIELD_NUMBER: _ClassVar[int]
    CONDITION_FIELD_NUMBER: _ClassVar[int]
    id: str
    projection: _model_pb2.Projection
    question: str
    owner: str
    status: _model_pb2.CellStatus
    basis: _attestation_pb2.Basis
    sufficiency: _attestation_pb2.Sufficiency
    notes: _containers.RepeatedScalarFieldContainer[str]
    citations: _containers.RepeatedCompositeFieldContainer[Citation]
    condition: str
    def __init__(self, id: _Optional[str] = ..., projection: _Optional[_Union[_model_pb2.Projection, str]] = ..., question: _Optional[str] = ..., owner: _Optional[str] = ..., status: _Optional[_Union[_model_pb2.CellStatus, str]] = ..., basis: _Optional[_Union[_attestation_pb2.Basis, str]] = ..., sufficiency: _Optional[_Union[_attestation_pb2.Sufficiency, str]] = ..., notes: _Optional[_Iterable[str]] = ..., citations: _Optional[_Iterable[_Union[Citation, _Mapping]]] = ..., condition: _Optional[str] = ...) -> None: ...

class ListCellsRequest(_message.Message):
    __slots__ = ("projection", "status")
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    projection: _model_pb2.Projection
    status: _model_pb2.CellStatus
    def __init__(self, projection: _Optional[_Union[_model_pb2.Projection, str]] = ..., status: _Optional[_Union[_model_pb2.CellStatus, str]] = ...) -> None: ...

class ListCellsResponse(_message.Message):
    __slots__ = ("cells",)
    CELLS_FIELD_NUMBER: _ClassVar[int]
    cells: _containers.RepeatedCompositeFieldContainer[Cell]
    def __init__(self, cells: _Optional[_Iterable[_Union[Cell, _Mapping]]] = ...) -> None: ...

class ExplainCellRequest(_message.Message):
    __slots__ = ("cell_id",)
    CELL_ID_FIELD_NUMBER: _ClassVar[int]
    cell_id: str
    def __init__(self, cell_id: _Optional[str] = ...) -> None: ...

class ExplainCellResponse(_message.Message):
    __slots__ = ("cell",)
    CELL_FIELD_NUMBER: _ClassVar[int]
    cell: Cell
    def __init__(self, cell: _Optional[_Union[Cell, _Mapping]] = ...) -> None: ...

class BaseDocIssue(_message.Message):
    __slots__ = ("projection", "code", "message", "location", "severity")
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    projection: _model_pb2.Projection
    code: str
    message: str
    location: str
    severity: _model_pb2.Severity
    def __init__(self, projection: _Optional[_Union[_model_pb2.Projection, str]] = ..., code: _Optional[str] = ..., message: _Optional[str] = ..., location: _Optional[str] = ..., severity: _Optional[_Union[_model_pb2.Severity, str]] = ...) -> None: ...

class ValidateBaseDocsRequest(_message.Message):
    __slots__ = ("projection",)
    PROJECTION_FIELD_NUMBER: _ClassVar[int]
    projection: _model_pb2.Projection
    def __init__(self, projection: _Optional[_Union[_model_pb2.Projection, str]] = ...) -> None: ...

class ValidateBaseDocsResponse(_message.Message):
    __slots__ = ("issues", "ok")
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    OK_FIELD_NUMBER: _ClassVar[int]
    issues: _containers.RepeatedCompositeFieldContainer[BaseDocIssue]
    ok: bool
    def __init__(self, issues: _Optional[_Iterable[_Union[BaseDocIssue, _Mapping]]] = ..., ok: _Optional[bool] = ...) -> None: ...
