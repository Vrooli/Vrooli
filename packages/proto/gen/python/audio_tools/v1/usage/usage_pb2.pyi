import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from audio_tools.v1.common import common_pb2 as _common_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class UsageRow(_message.Message):
    __slots__ = ("operation_id", "emitted_at", "capability", "operation", "provider_tier", "provider_id", "model_id", "latency_ms", "credits_charged", "prompt_tokens", "output_tokens", "audio_duration_seconds", "error", "fallback_reason", "user_identity")
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    EMITTED_AT_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_TIER_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    CREDITS_CHARGED_FIELD_NUMBER: _ClassVar[int]
    PROMPT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    AUDIO_DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_REASON_FIELD_NUMBER: _ClassVar[int]
    USER_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    emitted_at: _timestamp_pb2.Timestamp
    capability: str
    operation: str
    provider_tier: _common_pb2.ProviderTier
    provider_id: str
    model_id: str
    latency_ms: float
    credits_charged: int
    prompt_tokens: int
    output_tokens: int
    audio_duration_seconds: float
    error: str
    fallback_reason: str
    user_identity: str
    def __init__(self, operation_id: _Optional[str] = ..., emitted_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., capability: _Optional[str] = ..., operation: _Optional[str] = ..., provider_tier: _Optional[_Union[_common_pb2.ProviderTier, str]] = ..., provider_id: _Optional[str] = ..., model_id: _Optional[str] = ..., latency_ms: _Optional[float] = ..., credits_charged: _Optional[int] = ..., prompt_tokens: _Optional[int] = ..., output_tokens: _Optional[int] = ..., audio_duration_seconds: _Optional[float] = ..., error: _Optional[str] = ..., fallback_reason: _Optional[str] = ..., user_identity: _Optional[str] = ...) -> None: ...

class ListRecentRequest(_message.Message):
    __slots__ = ("since_seconds", "after_emitted_at", "limit", "capability", "provider_tier")
    SINCE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    AFTER_EMITTED_AT_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_TIER_FIELD_NUMBER: _ClassVar[int]
    since_seconds: int
    after_emitted_at: _timestamp_pb2.Timestamp
    limit: int
    capability: str
    provider_tier: _common_pb2.ProviderTier
    def __init__(self, since_seconds: _Optional[int] = ..., after_emitted_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., limit: _Optional[int] = ..., capability: _Optional[str] = ..., provider_tier: _Optional[_Union[_common_pb2.ProviderTier, str]] = ...) -> None: ...

class ListRecentResponse(_message.Message):
    __slots__ = ("rows",)
    ROWS_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[UsageRow]
    def __init__(self, rows: _Optional[_Iterable[_Union[UsageRow, _Mapping]]] = ...) -> None: ...

class ProviderDistribution(_message.Message):
    __slots__ = ("provider_tier", "provider_id", "count", "credits_total", "avg_latency_ms")
    PROVIDER_TIER_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    CREDITS_TOTAL_FIELD_NUMBER: _ClassVar[int]
    AVG_LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    provider_tier: _common_pb2.ProviderTier
    provider_id: str
    count: int
    credits_total: int
    avg_latency_ms: float
    def __init__(self, provider_tier: _Optional[_Union[_common_pb2.ProviderTier, str]] = ..., provider_id: _Optional[str] = ..., count: _Optional[int] = ..., credits_total: _Optional[int] = ..., avg_latency_ms: _Optional[float] = ...) -> None: ...

class FallbackReason(_message.Message):
    __slots__ = ("reason", "count")
    REASON_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    reason: str
    count: int
    def __init__(self, reason: _Optional[str] = ..., count: _Optional[int] = ...) -> None: ...

class Summary(_message.Message):
    __slots__ = ("since", "until", "operations_total", "credits_total", "distribution", "fallback_reasons", "error_count")
    SINCE_FIELD_NUMBER: _ClassVar[int]
    UNTIL_FIELD_NUMBER: _ClassVar[int]
    OPERATIONS_TOTAL_FIELD_NUMBER: _ClassVar[int]
    CREDITS_TOTAL_FIELD_NUMBER: _ClassVar[int]
    DISTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_REASONS_FIELD_NUMBER: _ClassVar[int]
    ERROR_COUNT_FIELD_NUMBER: _ClassVar[int]
    since: _timestamp_pb2.Timestamp
    until: _timestamp_pb2.Timestamp
    operations_total: int
    credits_total: int
    distribution: _containers.RepeatedCompositeFieldContainer[ProviderDistribution]
    fallback_reasons: _containers.RepeatedCompositeFieldContainer[FallbackReason]
    error_count: int
    def __init__(self, since: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., until: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., operations_total: _Optional[int] = ..., credits_total: _Optional[int] = ..., distribution: _Optional[_Iterable[_Union[ProviderDistribution, _Mapping]]] = ..., fallback_reasons: _Optional[_Iterable[_Union[FallbackReason, _Mapping]]] = ..., error_count: _Optional[int] = ...) -> None: ...

class GetSummaryRequest(_message.Message):
    __slots__ = ("since_seconds", "capability")
    SINCE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    since_seconds: int
    capability: str
    def __init__(self, since_seconds: _Optional[int] = ..., capability: _Optional[str] = ...) -> None: ...

class GetSummaryResponse(_message.Message):
    __slots__ = ("summary",)
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    summary: Summary
    def __init__(self, summary: _Optional[_Union[Summary, _Mapping]] = ...) -> None: ...
