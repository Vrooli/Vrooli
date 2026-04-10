import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SystemReport(_message.Message):
    __slots__ = ("id", "type", "generated_at", "period_start", "period_end", "summary", "metrics", "alerts", "recommendations")
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    PERIOD_START_FIELD_NUMBER: _ClassVar[int]
    PERIOD_END_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    ALERTS_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATIONS_FIELD_NUMBER: _ClassVar[int]
    id: str
    type: str
    generated_at: _timestamp_pb2.Timestamp
    period_start: _timestamp_pb2.Timestamp
    period_end: _timestamp_pb2.Timestamp
    summary: ReportSummary
    metrics: ReportMetrics
    alerts: _containers.RepeatedCompositeFieldContainer[ReportAlert]
    recommendations: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., type: _Optional[str] = ..., generated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., period_start: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., period_end: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., summary: _Optional[_Union[ReportSummary, _Mapping]] = ..., metrics: _Optional[_Union[ReportMetrics, _Mapping]] = ..., alerts: _Optional[_Iterable[_Union[ReportAlert, _Mapping]]] = ..., recommendations: _Optional[_Iterable[str]] = ...) -> None: ...

class ReportSummary(_message.Message):
    __slots__ = ("total_alerts", "avg_cpu_usage", "avg_memory_usage", "max_tcp_connections", "uptime_percentage")
    TOTAL_ALERTS_FIELD_NUMBER: _ClassVar[int]
    AVG_CPU_USAGE_FIELD_NUMBER: _ClassVar[int]
    AVG_MEMORY_USAGE_FIELD_NUMBER: _ClassVar[int]
    MAX_TCP_CONNECTIONS_FIELD_NUMBER: _ClassVar[int]
    UPTIME_PERCENTAGE_FIELD_NUMBER: _ClassVar[int]
    total_alerts: int
    avg_cpu_usage: float
    avg_memory_usage: float
    max_tcp_connections: int
    uptime_percentage: float
    def __init__(self, total_alerts: _Optional[int] = ..., avg_cpu_usage: _Optional[float] = ..., avg_memory_usage: _Optional[float] = ..., max_tcp_connections: _Optional[int] = ..., uptime_percentage: _Optional[float] = ...) -> None: ...

class ReportMetrics(_message.Message):
    __slots__ = ("cpu_trend", "memory_trend", "network_trend", "timestamps")
    CPU_TREND_FIELD_NUMBER: _ClassVar[int]
    MEMORY_TREND_FIELD_NUMBER: _ClassVar[int]
    NETWORK_TREND_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMPS_FIELD_NUMBER: _ClassVar[int]
    cpu_trend: _containers.RepeatedScalarFieldContainer[float]
    memory_trend: _containers.RepeatedScalarFieldContainer[float]
    network_trend: _containers.RepeatedScalarFieldContainer[float]
    timestamps: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, cpu_trend: _Optional[_Iterable[float]] = ..., memory_trend: _Optional[_Iterable[float]] = ..., network_trend: _Optional[_Iterable[float]] = ..., timestamps: _Optional[_Iterable[str]] = ...) -> None: ...

class ReportAlert(_message.Message):
    __slots__ = ("timestamp", "severity", "category", "message", "resolved")
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_FIELD_NUMBER: _ClassVar[int]
    timestamp: _timestamp_pb2.Timestamp
    severity: str
    category: str
    message: str
    resolved: bool
    def __init__(self, timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., severity: _Optional[str] = ..., category: _Optional[str] = ..., message: _Optional[str] = ..., resolved: _Optional[bool] = ...) -> None: ...

class EnhancedSystemReport(_message.Message):
    __slots__ = ("report_id", "report_type", "generated_at", "time_range", "actual_duration", "date_range_display", "executive_summary", "performance", "trends", "recommendations", "highlights", "metrics_count", "alerts_count", "investigations_count")
    REPORT_ID_FIELD_NUMBER: _ClassVar[int]
    REPORT_TYPE_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    TIME_RANGE_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_DURATION_FIELD_NUMBER: _ClassVar[int]
    DATE_RANGE_DISPLAY_FIELD_NUMBER: _ClassVar[int]
    EXECUTIVE_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    PERFORMANCE_FIELD_NUMBER: _ClassVar[int]
    TRENDS_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATIONS_FIELD_NUMBER: _ClassVar[int]
    HIGHLIGHTS_FIELD_NUMBER: _ClassVar[int]
    METRICS_COUNT_FIELD_NUMBER: _ClassVar[int]
    ALERTS_COUNT_FIELD_NUMBER: _ClassVar[int]
    INVESTIGATIONS_COUNT_FIELD_NUMBER: _ClassVar[int]
    report_id: str
    report_type: str
    generated_at: _timestamp_pb2.Timestamp
    time_range: ReportTimeRange
    actual_duration: str
    date_range_display: str
    executive_summary: EnhancedExecutiveSummary
    performance: PerformanceAnalysis
    trends: _containers.RepeatedCompositeFieldContainer[Trend]
    recommendations: _containers.RepeatedScalarFieldContainer[str]
    highlights: _containers.RepeatedScalarFieldContainer[str]
    metrics_count: int
    alerts_count: int
    investigations_count: int
    def __init__(self, report_id: _Optional[str] = ..., report_type: _Optional[str] = ..., generated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., time_range: _Optional[_Union[ReportTimeRange, _Mapping]] = ..., actual_duration: _Optional[str] = ..., date_range_display: _Optional[str] = ..., executive_summary: _Optional[_Union[EnhancedExecutiveSummary, _Mapping]] = ..., performance: _Optional[_Union[PerformanceAnalysis, _Mapping]] = ..., trends: _Optional[_Iterable[_Union[Trend, _Mapping]]] = ..., recommendations: _Optional[_Iterable[str]] = ..., highlights: _Optional[_Iterable[str]] = ..., metrics_count: _Optional[int] = ..., alerts_count: _Optional[int] = ..., investigations_count: _Optional[int] = ...) -> None: ...

class ReportTimeRange(_message.Message):
    __slots__ = ("start_time", "end_time")
    START_TIME_FIELD_NUMBER: _ClassVar[int]
    END_TIME_FIELD_NUMBER: _ClassVar[int]
    start_time: _timestamp_pb2.Timestamp
    end_time: _timestamp_pb2.Timestamp
    def __init__(self, start_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., end_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class EnhancedExecutiveSummary(_message.Message):
    __slots__ = ("overall_health", "key_findings", "time_description", "metrics_analyzed")
    OVERALL_HEALTH_FIELD_NUMBER: _ClassVar[int]
    KEY_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    TIME_DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    METRICS_ANALYZED_FIELD_NUMBER: _ClassVar[int]
    overall_health: str
    key_findings: _containers.RepeatedScalarFieldContainer[str]
    time_description: str
    metrics_analyzed: int
    def __init__(self, overall_health: _Optional[str] = ..., key_findings: _Optional[_Iterable[str]] = ..., time_description: _Optional[str] = ..., metrics_analyzed: _Optional[int] = ...) -> None: ...

class PerformanceAnalysis(_message.Message):
    __slots__ = ("cpu", "memory", "time_range")
    CPU_FIELD_NUMBER: _ClassVar[int]
    MEMORY_FIELD_NUMBER: _ClassVar[int]
    TIME_RANGE_FIELD_NUMBER: _ClassVar[int]
    cpu: MetricStats
    memory: MetricStats
    time_range: str
    def __init__(self, cpu: _Optional[_Union[MetricStats, _Mapping]] = ..., memory: _Optional[_Union[MetricStats, _Mapping]] = ..., time_range: _Optional[str] = ...) -> None: ...

class MetricStats(_message.Message):
    __slots__ = ("average", "min", "max", "std_dev", "peak_value", "peak_time", "min_time")
    AVERAGE_FIELD_NUMBER: _ClassVar[int]
    MIN_FIELD_NUMBER: _ClassVar[int]
    MAX_FIELD_NUMBER: _ClassVar[int]
    STD_DEV_FIELD_NUMBER: _ClassVar[int]
    PEAK_VALUE_FIELD_NUMBER: _ClassVar[int]
    PEAK_TIME_FIELD_NUMBER: _ClassVar[int]
    MIN_TIME_FIELD_NUMBER: _ClassVar[int]
    average: float
    min: float
    max: float
    std_dev: float
    peak_value: float
    peak_time: _timestamp_pb2.Timestamp
    min_time: _timestamp_pb2.Timestamp
    def __init__(self, average: _Optional[float] = ..., min: _Optional[float] = ..., max: _Optional[float] = ..., std_dev: _Optional[float] = ..., peak_value: _Optional[float] = ..., peak_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., min_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Trend(_message.Message):
    __slots__ = ("name", "direction", "change", "change_percent")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DIRECTION_FIELD_NUMBER: _ClassVar[int]
    CHANGE_FIELD_NUMBER: _ClassVar[int]
    CHANGE_PERCENT_FIELD_NUMBER: _ClassVar[int]
    name: str
    direction: str
    change: float
    change_percent: float
    def __init__(self, name: _Optional[str] = ..., direction: _Optional[str] = ..., change: _Optional[float] = ..., change_percent: _Optional[float] = ...) -> None: ...
