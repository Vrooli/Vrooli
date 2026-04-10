from system_monitor.v1.domain import metrics_pb2 as _metrics_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetCurrentMetricsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetCurrentMetricsResponse(_message.Message):
    __slots__ = ("metrics",)
    METRICS_FIELD_NUMBER: _ClassVar[int]
    metrics: _metrics_pb2.MetricsResponse
    def __init__(self, metrics: _Optional[_Union[_metrics_pb2.MetricsResponse, _Mapping]] = ...) -> None: ...

class GetDetailedMetricsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetDetailedMetricsResponse(_message.Message):
    __slots__ = ("metrics",)
    METRICS_FIELD_NUMBER: _ClassVar[int]
    metrics: _metrics_pb2.DetailedMetrics
    def __init__(self, metrics: _Optional[_Union[_metrics_pb2.DetailedMetrics, _Mapping]] = ...) -> None: ...

class GetProcessMonitorRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetProcessMonitorResponse(_message.Message):
    __slots__ = ("data",)
    DATA_FIELD_NUMBER: _ClassVar[int]
    data: _metrics_pb2.ProcessMonitorData
    def __init__(self, data: _Optional[_Union[_metrics_pb2.ProcessMonitorData, _Mapping]] = ...) -> None: ...

class GetInfrastructureMonitorRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetInfrastructureMonitorResponse(_message.Message):
    __slots__ = ("data",)
    DATA_FIELD_NUMBER: _ClassVar[int]
    data: _metrics_pb2.InfrastructureMonitorData
    def __init__(self, data: _Optional[_Union[_metrics_pb2.InfrastructureMonitorData, _Mapping]] = ...) -> None: ...

class GetMetricsTimelineRequest(_message.Message):
    __slots__ = ("window_seconds", "sample_interval_seconds")
    WINDOW_SECONDS_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_INTERVAL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    window_seconds: int
    sample_interval_seconds: int
    def __init__(self, window_seconds: _Optional[int] = ..., sample_interval_seconds: _Optional[int] = ...) -> None: ...

class GetMetricsTimelineResponse(_message.Message):
    __slots__ = ("timeline",)
    TIMELINE_FIELD_NUMBER: _ClassVar[int]
    timeline: _metrics_pb2.MetricsTimelineResponse
    def __init__(self, timeline: _Optional[_Union[_metrics_pb2.MetricsTimelineResponse, _Mapping]] = ...) -> None: ...

class GetDiskDetailRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetDiskDetailResponse(_message.Message):
    __slots__ = ("data",)
    DATA_FIELD_NUMBER: _ClassVar[int]
    data: _metrics_pb2.DiskDetailResponse
    def __init__(self, data: _Optional[_Union[_metrics_pb2.DiskDetailResponse, _Mapping]] = ...) -> None: ...
