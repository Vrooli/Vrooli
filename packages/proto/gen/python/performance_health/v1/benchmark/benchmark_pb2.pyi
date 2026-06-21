from common.v1 import metrics_pb2 as _metrics_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class BenchmarkOutcome(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    BENCHMARK_OUTCOME_UNSPECIFIED: _ClassVar[BenchmarkOutcome]
    BENCHMARK_OUTCOME_MEASURED: _ClassVar[BenchmarkOutcome]
    BENCHMARK_OUTCOME_SKIPPED: _ClassVar[BenchmarkOutcome]
    BENCHMARK_OUTCOME_FAILED: _ClassVar[BenchmarkOutcome]
BENCHMARK_OUTCOME_UNSPECIFIED: BenchmarkOutcome
BENCHMARK_OUTCOME_MEASURED: BenchmarkOutcome
BENCHMARK_OUTCOME_SKIPPED: BenchmarkOutcome
BENCHMARK_OUTCOME_FAILED: BenchmarkOutcome

class RunBenchmarkRequest(_message.Message):
    __slots__ = ("scenario", "path")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class RunBenchmarkResponse(_message.Message):
    __slots__ = ("scenario", "outcome", "timings", "reason", "metrics")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    TIMINGS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    outcome: BenchmarkOutcome
    timings: _containers.RepeatedCompositeFieldContainer[BuildTiming]
    reason: str
    metrics: _metrics_pb2.ExecutionMetrics
    def __init__(self, scenario: _Optional[str] = ..., outcome: _Optional[_Union[BenchmarkOutcome, str]] = ..., timings: _Optional[_Iterable[_Union[BuildTiming, _Mapping]]] = ..., reason: _Optional[str] = ..., metrics: _Optional[_Union[_metrics_pb2.ExecutionMetrics, _Mapping]] = ...) -> None: ...

class BuildTiming(_message.Message):
    __slots__ = ("surface", "duration_ms", "budget_ms", "over_budget")
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    BUDGET_MS_FIELD_NUMBER: _ClassVar[int]
    OVER_BUDGET_FIELD_NUMBER: _ClassVar[int]
    surface: str
    duration_ms: int
    budget_ms: int
    over_budget: bool
    def __init__(self, surface: _Optional[str] = ..., duration_ms: _Optional[int] = ..., budget_ms: _Optional[int] = ..., over_budget: _Optional[bool] = ...) -> None: ...
