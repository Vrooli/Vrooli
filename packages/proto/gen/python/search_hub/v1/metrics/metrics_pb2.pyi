from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class InsightsRequest(_message.Message):
    __slots__ = ("window_days",)
    WINDOW_DAYS_FIELD_NUMBER: _ClassVar[int]
    window_days: int
    def __init__(self, window_days: _Optional[int] = ...) -> None: ...

class ProviderUtilization(_message.Message):
    __slots__ = ("provider_id", "provider_group", "type", "times_routed", "total_hits", "under_utilized", "latency_p50_ms", "latency_p95_ms", "degraded_count", "degradation_rate", "degradation_reasons")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_GROUP_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    TIMES_ROUTED_FIELD_NUMBER: _ClassVar[int]
    TOTAL_HITS_FIELD_NUMBER: _ClassVar[int]
    UNDER_UTILIZED_FIELD_NUMBER: _ClassVar[int]
    LATENCY_P50_MS_FIELD_NUMBER: _ClassVar[int]
    LATENCY_P95_MS_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_COUNT_FIELD_NUMBER: _ClassVar[int]
    DEGRADATION_RATE_FIELD_NUMBER: _ClassVar[int]
    DEGRADATION_REASONS_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    provider_group: str
    type: str
    times_routed: int
    total_hits: int
    under_utilized: bool
    latency_p50_ms: int
    latency_p95_ms: int
    degraded_count: int
    degradation_rate: float
    degradation_reasons: _containers.RepeatedCompositeFieldContainer[ProviderDegradationReason]
    def __init__(self, provider_id: _Optional[str] = ..., provider_group: _Optional[str] = ..., type: _Optional[str] = ..., times_routed: _Optional[int] = ..., total_hits: _Optional[int] = ..., under_utilized: _Optional[bool] = ..., latency_p50_ms: _Optional[int] = ..., latency_p95_ms: _Optional[int] = ..., degraded_count: _Optional[int] = ..., degradation_rate: _Optional[float] = ..., degradation_reasons: _Optional[_Iterable[_Union[ProviderDegradationReason, _Mapping]]] = ...) -> None: ...

class ProviderDegradationReason(_message.Message):
    __slots__ = ("reason", "count")
    REASON_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    reason: str
    count: int
    def __init__(self, reason: _Optional[str] = ..., count: _Optional[int] = ...) -> None: ...

class InsightsResponse(_message.Message):
    __slots__ = ("total_queries", "zero_result_queries", "zero_result_rate", "degraded_queries", "reranked_queries", "latency_p50_ms", "latency_p95_ms", "providers")
    TOTAL_QUERIES_FIELD_NUMBER: _ClassVar[int]
    ZERO_RESULT_QUERIES_FIELD_NUMBER: _ClassVar[int]
    ZERO_RESULT_RATE_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_QUERIES_FIELD_NUMBER: _ClassVar[int]
    RERANKED_QUERIES_FIELD_NUMBER: _ClassVar[int]
    LATENCY_P50_MS_FIELD_NUMBER: _ClassVar[int]
    LATENCY_P95_MS_FIELD_NUMBER: _ClassVar[int]
    PROVIDERS_FIELD_NUMBER: _ClassVar[int]
    total_queries: int
    zero_result_queries: int
    zero_result_rate: float
    degraded_queries: int
    reranked_queries: int
    latency_p50_ms: int
    latency_p95_ms: int
    providers: _containers.RepeatedCompositeFieldContainer[ProviderUtilization]
    def __init__(self, total_queries: _Optional[int] = ..., zero_result_queries: _Optional[int] = ..., zero_result_rate: _Optional[float] = ..., degraded_queries: _Optional[int] = ..., reranked_queries: _Optional[int] = ..., latency_p50_ms: _Optional[int] = ..., latency_p95_ms: _Optional[int] = ..., providers: _Optional[_Iterable[_Union[ProviderUtilization, _Mapping]]] = ...) -> None: ...
