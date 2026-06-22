from google.api import annotations_pb2 as _annotations_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SystemSettings(_message.Message):
    __slots__ = ("active", "metric_collection_interval", "anomaly_detection_interval", "threshold_check_interval", "cooldown_period_seconds", "cpu_threshold", "memory_threshold", "disk_threshold", "metrics_retention_days", "retention_check_interval_seconds", "retention_run_on_startup", "compact_after_retention")
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    METRIC_COLLECTION_INTERVAL_FIELD_NUMBER: _ClassVar[int]
    ANOMALY_DETECTION_INTERVAL_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_CHECK_INTERVAL_FIELD_NUMBER: _ClassVar[int]
    COOLDOWN_PERIOD_SECONDS_FIELD_NUMBER: _ClassVar[int]
    CPU_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    MEMORY_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    DISK_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    METRICS_RETENTION_DAYS_FIELD_NUMBER: _ClassVar[int]
    RETENTION_CHECK_INTERVAL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    RETENTION_RUN_ON_STARTUP_FIELD_NUMBER: _ClassVar[int]
    COMPACT_AFTER_RETENTION_FIELD_NUMBER: _ClassVar[int]
    active: bool
    metric_collection_interval: int
    anomaly_detection_interval: int
    threshold_check_interval: int
    cooldown_period_seconds: int
    cpu_threshold: float
    memory_threshold: float
    disk_threshold: float
    metrics_retention_days: int
    retention_check_interval_seconds: int
    retention_run_on_startup: bool
    compact_after_retention: bool
    def __init__(self, active: _Optional[bool] = ..., metric_collection_interval: _Optional[int] = ..., anomaly_detection_interval: _Optional[int] = ..., threshold_check_interval: _Optional[int] = ..., cooldown_period_seconds: _Optional[int] = ..., cpu_threshold: _Optional[float] = ..., memory_threshold: _Optional[float] = ..., disk_threshold: _Optional[float] = ..., metrics_retention_days: _Optional[int] = ..., retention_check_interval_seconds: _Optional[int] = ..., retention_run_on_startup: _Optional[bool] = ..., compact_after_retention: _Optional[bool] = ...) -> None: ...

class GetSettingsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetSettingsResponse(_message.Message):
    __slots__ = ("success", "settings", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    settings: SystemSettings
    error: str
    def __init__(self, success: _Optional[bool] = ..., settings: _Optional[_Union[SystemSettings, _Mapping]] = ..., error: _Optional[str] = ...) -> None: ...

class UpdateSettingsRequest(_message.Message):
    __slots__ = ("settings",)
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    settings: SystemSettings
    def __init__(self, settings: _Optional[_Union[SystemSettings, _Mapping]] = ...) -> None: ...

class UpdateSettingsResponse(_message.Message):
    __slots__ = ("success", "settings", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    settings: SystemSettings
    error: str
    def __init__(self, success: _Optional[bool] = ..., settings: _Optional[_Union[SystemSettings, _Mapping]] = ..., error: _Optional[str] = ...) -> None: ...

class ResetSettingsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ResetSettingsResponse(_message.Message):
    __slots__ = ("success", "settings", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    settings: SystemSettings
    error: str
    def __init__(self, success: _Optional[bool] = ..., settings: _Optional[_Union[SystemSettings, _Mapping]] = ..., error: _Optional[str] = ...) -> None: ...

class GetMaintenanceStateRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetMaintenanceStateResponse(_message.Message):
    __slots__ = ("success", "maintenance_state", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    MAINTENANCE_STATE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    maintenance_state: str
    error: str
    def __init__(self, success: _Optional[bool] = ..., maintenance_state: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...

class SetMaintenanceStateRequest(_message.Message):
    __slots__ = ("maintenance_state",)
    MAINTENANCE_STATE_FIELD_NUMBER: _ClassVar[int]
    maintenance_state: str
    def __init__(self, maintenance_state: _Optional[str] = ...) -> None: ...

class SetMaintenanceStateResponse(_message.Message):
    __slots__ = ("success", "maintenance_state", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    MAINTENANCE_STATE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    maintenance_state: str
    error: str
    def __init__(self, success: _Optional[bool] = ..., maintenance_state: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...
