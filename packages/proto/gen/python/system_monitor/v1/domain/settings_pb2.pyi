from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class SystemSettings(_message.Message):
    __slots__ = ("active", "metric_collection_interval", "anomaly_detection_interval", "threshold_check_interval", "cooldown_period_seconds", "cpu_threshold", "memory_threshold", "disk_threshold")
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    METRIC_COLLECTION_INTERVAL_FIELD_NUMBER: _ClassVar[int]
    ANOMALY_DETECTION_INTERVAL_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_CHECK_INTERVAL_FIELD_NUMBER: _ClassVar[int]
    COOLDOWN_PERIOD_SECONDS_FIELD_NUMBER: _ClassVar[int]
    CPU_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    MEMORY_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    DISK_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    active: bool
    metric_collection_interval: int
    anomaly_detection_interval: int
    threshold_check_interval: int
    cooldown_period_seconds: int
    cpu_threshold: float
    memory_threshold: float
    disk_threshold: float
    def __init__(self, active: _Optional[bool] = ..., metric_collection_interval: _Optional[int] = ..., anomaly_detection_interval: _Optional[int] = ..., threshold_check_interval: _Optional[int] = ..., cooldown_period_seconds: _Optional[int] = ..., cpu_threshold: _Optional[float] = ..., memory_threshold: _Optional[float] = ..., disk_threshold: _Optional[float] = ...) -> None: ...
