from common.v1 import metrics_pb2 as _metrics_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class BenchmarkStartupRequest(_message.Message):
    __slots__ = ("scenario", "timeout_seconds")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    timeout_seconds: int
    def __init__(self, scenario: _Optional[str] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class BenchmarkStartupResponse(_message.Message):
    __slots__ = ("measurement",)
    MEASUREMENT_FIELD_NUMBER: _ClassVar[int]
    measurement: PerfMeasurement
    def __init__(self, measurement: _Optional[_Union[PerfMeasurement, _Mapping]] = ...) -> None: ...

class GetPerfTrendRequest(_message.Message):
    __slots__ = ("scenario", "limit")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    limit: int
    def __init__(self, scenario: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class GetPerfTrendResponse(_message.Message):
    __slots__ = ("scenario", "measurements")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    MEASUREMENTS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    measurements: _containers.RepeatedCompositeFieldContainer[PerfMeasurement]
    def __init__(self, scenario: _Optional[str] = ..., measurements: _Optional[_Iterable[_Union[PerfMeasurement, _Mapping]]] = ...) -> None: ...

class PerfMeasurement(_message.Message):
    __slots__ = ("scenario", "captured_at", "time_to_healthy_ms", "healthy", "surface_timings", "metrics", "note")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    TIME_TO_HEALTHY_MS_FIELD_NUMBER: _ClassVar[int]
    HEALTHY_FIELD_NUMBER: _ClassVar[int]
    SURFACE_TIMINGS_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    captured_at: str
    time_to_healthy_ms: int
    healthy: bool
    surface_timings: _containers.RepeatedCompositeFieldContainer[SurfaceTiming]
    metrics: _metrics_pb2.ExecutionMetrics
    note: str
    def __init__(self, scenario: _Optional[str] = ..., captured_at: _Optional[str] = ..., time_to_healthy_ms: _Optional[int] = ..., healthy: _Optional[bool] = ..., surface_timings: _Optional[_Iterable[_Union[SurfaceTiming, _Mapping]]] = ..., metrics: _Optional[_Union[_metrics_pb2.ExecutionMetrics, _Mapping]] = ..., note: _Optional[str] = ...) -> None: ...

class SurfaceTiming(_message.Message):
    __slots__ = ("surface", "time_to_healthy_ms", "healthy")
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    TIME_TO_HEALTHY_MS_FIELD_NUMBER: _ClassVar[int]
    HEALTHY_FIELD_NUMBER: _ClassVar[int]
    surface: str
    time_to_healthy_ms: int
    healthy: bool
    def __init__(self, surface: _Optional[str] = ..., time_to_healthy_ms: _Optional[int] = ..., healthy: _Optional[bool] = ...) -> None: ...
