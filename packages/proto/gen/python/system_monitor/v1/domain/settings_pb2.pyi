from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

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
