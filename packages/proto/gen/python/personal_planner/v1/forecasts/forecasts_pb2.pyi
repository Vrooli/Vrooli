from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Forecast(_message.Message):
    __slots__ = ("generated_at", "freshness", "input_fingerprint", "horizon_start", "horizon_end", "algorithm_version", "central_finish", "cautious_finish", "result_state", "risk_state", "explanation", "known_work_minutes", "available_minutes", "reserve_minutes", "unresolved_work_count", "commitment_outlooks", "snapshot_id", "previous_snapshot_id", "change_explanation")
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    INPUT_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    HORIZON_START_FIELD_NUMBER: _ClassVar[int]
    HORIZON_END_FIELD_NUMBER: _ClassVar[int]
    ALGORITHM_VERSION_FIELD_NUMBER: _ClassVar[int]
    CENTRAL_FINISH_FIELD_NUMBER: _ClassVar[int]
    CAUTIOUS_FINISH_FIELD_NUMBER: _ClassVar[int]
    RESULT_STATE_FIELD_NUMBER: _ClassVar[int]
    RISK_STATE_FIELD_NUMBER: _ClassVar[int]
    EXPLANATION_FIELD_NUMBER: _ClassVar[int]
    KNOWN_WORK_MINUTES_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_MINUTES_FIELD_NUMBER: _ClassVar[int]
    RESERVE_MINUTES_FIELD_NUMBER: _ClassVar[int]
    UNRESOLVED_WORK_COUNT_FIELD_NUMBER: _ClassVar[int]
    COMMITMENT_OUTLOOKS_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    CHANGE_EXPLANATION_FIELD_NUMBER: _ClassVar[int]
    generated_at: str
    freshness: str
    input_fingerprint: str
    horizon_start: str
    horizon_end: str
    algorithm_version: str
    central_finish: str
    cautious_finish: str
    result_state: str
    risk_state: str
    explanation: str
    known_work_minutes: int
    available_minutes: int
    reserve_minutes: int
    unresolved_work_count: int
    commitment_outlooks: _containers.RepeatedCompositeFieldContainer[CommitmentOutlook]
    snapshot_id: str
    previous_snapshot_id: str
    change_explanation: str
    def __init__(self, generated_at: _Optional[str] = ..., freshness: _Optional[str] = ..., input_fingerprint: _Optional[str] = ..., horizon_start: _Optional[str] = ..., horizon_end: _Optional[str] = ..., algorithm_version: _Optional[str] = ..., central_finish: _Optional[str] = ..., cautious_finish: _Optional[str] = ..., result_state: _Optional[str] = ..., risk_state: _Optional[str] = ..., explanation: _Optional[str] = ..., known_work_minutes: _Optional[int] = ..., available_minutes: _Optional[int] = ..., reserve_minutes: _Optional[int] = ..., unresolved_work_count: _Optional[int] = ..., commitment_outlooks: _Optional[_Iterable[_Union[CommitmentOutlook, _Mapping]]] = ..., snapshot_id: _Optional[str] = ..., previous_snapshot_id: _Optional[str] = ..., change_explanation: _Optional[str] = ...) -> None: ...

class CommitmentOutlook(_message.Message):
    __slots__ = ("id", "result", "promised_boundary", "forecast_finish", "risk_state", "explanation")
    ID_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    PROMISED_BOUNDARY_FIELD_NUMBER: _ClassVar[int]
    FORECAST_FINISH_FIELD_NUMBER: _ClassVar[int]
    RISK_STATE_FIELD_NUMBER: _ClassVar[int]
    EXPLANATION_FIELD_NUMBER: _ClassVar[int]
    id: str
    result: str
    promised_boundary: str
    forecast_finish: str
    risk_state: str
    explanation: str
    def __init__(self, id: _Optional[str] = ..., result: _Optional[str] = ..., promised_boundary: _Optional[str] = ..., forecast_finish: _Optional[str] = ..., risk_state: _Optional[str] = ..., explanation: _Optional[str] = ...) -> None: ...

class GetForecastRequest(_message.Message):
    __slots__ = ("local_date", "timezone", "horizon_days")
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    TIMEZONE_FIELD_NUMBER: _ClassVar[int]
    HORIZON_DAYS_FIELD_NUMBER: _ClassVar[int]
    local_date: str
    timezone: str
    horizon_days: int
    def __init__(self, local_date: _Optional[str] = ..., timezone: _Optional[str] = ..., horizon_days: _Optional[int] = ...) -> None: ...

class GetForecastResponse(_message.Message):
    __slots__ = ("forecast",)
    FORECAST_FIELD_NUMBER: _ClassVar[int]
    forecast: Forecast
    def __init__(self, forecast: _Optional[_Union[Forecast, _Mapping]] = ...) -> None: ...

class ForecastSnapshotSummary(_message.Message):
    __slots__ = ("id", "generated_at", "input_fingerprint", "horizon_start", "horizon_end", "central_finish", "cautious_finish", "result_state", "risk_state", "explanation", "previous_snapshot_id", "change_explanation")
    ID_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    INPUT_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    HORIZON_START_FIELD_NUMBER: _ClassVar[int]
    HORIZON_END_FIELD_NUMBER: _ClassVar[int]
    CENTRAL_FINISH_FIELD_NUMBER: _ClassVar[int]
    CAUTIOUS_FINISH_FIELD_NUMBER: _ClassVar[int]
    RESULT_STATE_FIELD_NUMBER: _ClassVar[int]
    RISK_STATE_FIELD_NUMBER: _ClassVar[int]
    EXPLANATION_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    CHANGE_EXPLANATION_FIELD_NUMBER: _ClassVar[int]
    id: str
    generated_at: str
    input_fingerprint: str
    horizon_start: str
    horizon_end: str
    central_finish: str
    cautious_finish: str
    result_state: str
    risk_state: str
    explanation: str
    previous_snapshot_id: str
    change_explanation: str
    def __init__(self, id: _Optional[str] = ..., generated_at: _Optional[str] = ..., input_fingerprint: _Optional[str] = ..., horizon_start: _Optional[str] = ..., horizon_end: _Optional[str] = ..., central_finish: _Optional[str] = ..., cautious_finish: _Optional[str] = ..., result_state: _Optional[str] = ..., risk_state: _Optional[str] = ..., explanation: _Optional[str] = ..., previous_snapshot_id: _Optional[str] = ..., change_explanation: _Optional[str] = ...) -> None: ...

class ListForecastSnapshotsRequest(_message.Message):
    __slots__ = ("limit",)
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    def __init__(self, limit: _Optional[int] = ...) -> None: ...

class ListForecastSnapshotsResponse(_message.Message):
    __slots__ = ("snapshots",)
    SNAPSHOTS_FIELD_NUMBER: _ClassVar[int]
    snapshots: _containers.RepeatedCompositeFieldContainer[ForecastSnapshotSummary]
    def __init__(self, snapshots: _Optional[_Iterable[_Union[ForecastSnapshotSummary, _Mapping]]] = ...) -> None: ...
