from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class InsightsRequest(_message.Message):
    __slots__ = ("window_days", "window")
    WINDOW_DAYS_FIELD_NUMBER: _ClassVar[int]
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window_days: int
    window: str
    def __init__(self, window_days: _Optional[int] = ..., window: _Optional[str] = ...) -> None: ...

class ProviderUtilization(_message.Message):
    __slots__ = ("provider_id", "provider_group", "type", "times_routed", "total_hits", "under_utilized", "latency_p50_ms", "latency_p95_ms", "degraded_count", "degradation_rate", "degradation_reasons", "active_reranker_leg")
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
    ACTIVE_RERANKER_LEG_FIELD_NUMBER: _ClassVar[int]
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
    active_reranker_leg: str
    def __init__(self, provider_id: _Optional[str] = ..., provider_group: _Optional[str] = ..., type: _Optional[str] = ..., times_routed: _Optional[int] = ..., total_hits: _Optional[int] = ..., under_utilized: _Optional[bool] = ..., latency_p50_ms: _Optional[int] = ..., latency_p95_ms: _Optional[int] = ..., degraded_count: _Optional[int] = ..., degradation_rate: _Optional[float] = ..., degradation_reasons: _Optional[_Iterable[_Union[ProviderDegradationReason, _Mapping]]] = ..., active_reranker_leg: _Optional[str] = ...) -> None: ...

class ProviderDegradationReason(_message.Message):
    __slots__ = ("reason", "count")
    REASON_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    reason: str
    count: int
    def __init__(self, reason: _Optional[str] = ..., count: _Optional[int] = ...) -> None: ...

class ProviderRetirementCandidate(_message.Message):
    __slots__ = ("provider_id", "times_routed", "total_hits", "reason")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    TIMES_ROUTED_FIELD_NUMBER: _ClassVar[int]
    TOTAL_HITS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    times_routed: int
    total_hits: int
    reason: str
    def __init__(self, provider_id: _Optional[str] = ..., times_routed: _Optional[int] = ..., total_hits: _Optional[int] = ..., reason: _Optional[str] = ...) -> None: ...

class ProviderGroupAdvisory(_message.Message):
    __slots__ = ("provider_group", "active_leaves", "share", "reason")
    PROVIDER_GROUP_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_LEAVES_FIELD_NUMBER: _ClassVar[int]
    SHARE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    provider_group: str
    active_leaves: int
    share: float
    reason: str
    def __init__(self, provider_group: _Optional[str] = ..., active_leaves: _Optional[int] = ..., share: _Optional[float] = ..., reason: _Optional[str] = ...) -> None: ...

class InsightsResponse(_message.Message):
    __slots__ = ("total_queries", "zero_result_queries", "zero_result_rate", "degraded_queries", "reranked_queries", "latency_p50_ms", "latency_p95_ms", "providers", "retirement_candidates", "group_advisories", "resolver_cache_hits", "resolver_cache_misses", "resolver_cache_hit_rate", "window_from", "window_to", "sample_count", "minimum_sample_count", "sample_sufficient", "recent_sample_count", "recent_latency_p50_ms", "recent_latency_p95_ms")
    TOTAL_QUERIES_FIELD_NUMBER: _ClassVar[int]
    ZERO_RESULT_QUERIES_FIELD_NUMBER: _ClassVar[int]
    ZERO_RESULT_RATE_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_QUERIES_FIELD_NUMBER: _ClassVar[int]
    RERANKED_QUERIES_FIELD_NUMBER: _ClassVar[int]
    LATENCY_P50_MS_FIELD_NUMBER: _ClassVar[int]
    LATENCY_P95_MS_FIELD_NUMBER: _ClassVar[int]
    PROVIDERS_FIELD_NUMBER: _ClassVar[int]
    RETIREMENT_CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    GROUP_ADVISORIES_FIELD_NUMBER: _ClassVar[int]
    RESOLVER_CACHE_HITS_FIELD_NUMBER: _ClassVar[int]
    RESOLVER_CACHE_MISSES_FIELD_NUMBER: _ClassVar[int]
    RESOLVER_CACHE_HIT_RATE_FIELD_NUMBER: _ClassVar[int]
    WINDOW_FROM_FIELD_NUMBER: _ClassVar[int]
    WINDOW_TO_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_SAMPLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_SUFFICIENT_FIELD_NUMBER: _ClassVar[int]
    RECENT_SAMPLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    RECENT_LATENCY_P50_MS_FIELD_NUMBER: _ClassVar[int]
    RECENT_LATENCY_P95_MS_FIELD_NUMBER: _ClassVar[int]
    total_queries: int
    zero_result_queries: int
    zero_result_rate: float
    degraded_queries: int
    reranked_queries: int
    latency_p50_ms: int
    latency_p95_ms: int
    providers: _containers.RepeatedCompositeFieldContainer[ProviderUtilization]
    retirement_candidates: _containers.RepeatedCompositeFieldContainer[ProviderRetirementCandidate]
    group_advisories: _containers.RepeatedCompositeFieldContainer[ProviderGroupAdvisory]
    resolver_cache_hits: int
    resolver_cache_misses: int
    resolver_cache_hit_rate: float
    window_from: str
    window_to: str
    sample_count: int
    minimum_sample_count: int
    sample_sufficient: bool
    recent_sample_count: int
    recent_latency_p50_ms: int
    recent_latency_p95_ms: int
    def __init__(self, total_queries: _Optional[int] = ..., zero_result_queries: _Optional[int] = ..., zero_result_rate: _Optional[float] = ..., degraded_queries: _Optional[int] = ..., reranked_queries: _Optional[int] = ..., latency_p50_ms: _Optional[int] = ..., latency_p95_ms: _Optional[int] = ..., providers: _Optional[_Iterable[_Union[ProviderUtilization, _Mapping]]] = ..., retirement_candidates: _Optional[_Iterable[_Union[ProviderRetirementCandidate, _Mapping]]] = ..., group_advisories: _Optional[_Iterable[_Union[ProviderGroupAdvisory, _Mapping]]] = ..., resolver_cache_hits: _Optional[int] = ..., resolver_cache_misses: _Optional[int] = ..., resolver_cache_hit_rate: _Optional[float] = ..., window_from: _Optional[str] = ..., window_to: _Optional[str] = ..., sample_count: _Optional[int] = ..., minimum_sample_count: _Optional[int] = ..., sample_sufficient: _Optional[bool] = ..., recent_sample_count: _Optional[int] = ..., recent_latency_p50_ms: _Optional[int] = ..., recent_latency_p95_ms: _Optional[int] = ...) -> None: ...
