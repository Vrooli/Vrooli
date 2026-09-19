from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class MonitoringSchedule(_message.Message):
    __slots__ = ("id", "name", "profile", "baseline_snapshot_id", "interval_minutes", "enabled", "latency_threshold_ms", "unavailable_threshold", "effects", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    BASELINE_SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    INTERVAL_MINUTES_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    LATENCY_THRESHOLD_MS_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    EFFECTS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    profile: str
    baseline_snapshot_id: str
    interval_minutes: int
    enabled: bool
    latency_threshold_ms: int
    unavailable_threshold: int
    effects: _containers.RepeatedScalarFieldContainer[str]
    created_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., profile: _Optional[str] = ..., baseline_snapshot_id: _Optional[str] = ..., interval_minutes: _Optional[int] = ..., enabled: _Optional[bool] = ..., latency_threshold_ms: _Optional[int] = ..., unavailable_threshold: _Optional[int] = ..., effects: _Optional[_Iterable[str]] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class MonitoringAlert(_message.Message):
    __slots__ = ("id", "schedule_id", "snapshot_id", "severity", "status", "summary", "evidence", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCHEDULE_ID_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    schedule_id: str
    snapshot_id: str
    severity: str
    status: str
    summary: str
    evidence: _containers.RepeatedScalarFieldContainer[str]
    created_at: str
    def __init__(self, id: _Optional[str] = ..., schedule_id: _Optional[str] = ..., snapshot_id: _Optional[str] = ..., severity: _Optional[str] = ..., status: _Optional[str] = ..., summary: _Optional[str] = ..., evidence: _Optional[_Iterable[str]] = ..., created_at: _Optional[str] = ...) -> None: ...

class MonitoringRun(_message.Message):
    __slots__ = ("id", "schedule_id", "snapshot_id", "status", "summary", "regression_detected", "alerts", "effects", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCHEDULE_ID_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    REGRESSION_DETECTED_FIELD_NUMBER: _ClassVar[int]
    ALERTS_FIELD_NUMBER: _ClassVar[int]
    EFFECTS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    schedule_id: str
    snapshot_id: str
    status: str
    summary: str
    regression_detected: bool
    alerts: _containers.RepeatedCompositeFieldContainer[MonitoringAlert]
    effects: _containers.RepeatedScalarFieldContainer[str]
    created_at: str
    def __init__(self, id: _Optional[str] = ..., schedule_id: _Optional[str] = ..., snapshot_id: _Optional[str] = ..., status: _Optional[str] = ..., summary: _Optional[str] = ..., regression_detected: _Optional[bool] = ..., alerts: _Optional[_Iterable[_Union[MonitoringAlert, _Mapping]]] = ..., effects: _Optional[_Iterable[str]] = ..., created_at: _Optional[str] = ...) -> None: ...

class ListMonitoringSchedulesRequest(_message.Message):
    __slots__ = ("include_disabled",)
    INCLUDE_DISABLED_FIELD_NUMBER: _ClassVar[int]
    include_disabled: bool
    def __init__(self, include_disabled: _Optional[bool] = ...) -> None: ...

class ListMonitoringSchedulesResponse(_message.Message):
    __slots__ = ("schedules",)
    SCHEDULES_FIELD_NUMBER: _ClassVar[int]
    schedules: _containers.RepeatedCompositeFieldContainer[MonitoringSchedule]
    def __init__(self, schedules: _Optional[_Iterable[_Union[MonitoringSchedule, _Mapping]]] = ...) -> None: ...

class UpsertMonitoringScheduleRequest(_message.Message):
    __slots__ = ("schedule",)
    SCHEDULE_FIELD_NUMBER: _ClassVar[int]
    schedule: MonitoringSchedule
    def __init__(self, schedule: _Optional[_Union[MonitoringSchedule, _Mapping]] = ...) -> None: ...

class UpsertMonitoringScheduleResponse(_message.Message):
    __slots__ = ("schedule",)
    SCHEDULE_FIELD_NUMBER: _ClassVar[int]
    schedule: MonitoringSchedule
    def __init__(self, schedule: _Optional[_Union[MonitoringSchedule, _Mapping]] = ...) -> None: ...

class RunMonitoringCheckRequest(_message.Message):
    __slots__ = ("schedule_id", "dry_run")
    SCHEDULE_ID_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    schedule_id: str
    dry_run: bool
    def __init__(self, schedule_id: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class RunMonitoringCheckResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: MonitoringRun
    def __init__(self, run: _Optional[_Union[MonitoringRun, _Mapping]] = ...) -> None: ...

class ListMonitoringAlertsRequest(_message.Message):
    __slots__ = ("schedule_id", "open_only")
    SCHEDULE_ID_FIELD_NUMBER: _ClassVar[int]
    OPEN_ONLY_FIELD_NUMBER: _ClassVar[int]
    schedule_id: str
    open_only: bool
    def __init__(self, schedule_id: _Optional[str] = ..., open_only: _Optional[bool] = ...) -> None: ...

class ListMonitoringAlertsResponse(_message.Message):
    __slots__ = ("alerts",)
    ALERTS_FIELD_NUMBER: _ClassVar[int]
    alerts: _containers.RepeatedCompositeFieldContainer[MonitoringAlert]
    def __init__(self, alerts: _Optional[_Iterable[_Union[MonitoringAlert, _Mapping]]] = ...) -> None: ...
