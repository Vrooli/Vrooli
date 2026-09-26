from measures.v1 import measures_pb2 as _measures_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class FederatedLatencyRequest(_message.Message):
    __slots__ = ("window",)
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ...) -> None: ...

class FederatedLatencyResponse(_message.Message):
    __slots__ = ("p95_ms", "p50_ms")
    P95_MS_FIELD_NUMBER: _ClassVar[int]
    P50_MS_FIELD_NUMBER: _ClassVar[int]
    p95_ms: int
    p50_ms: int
    def __init__(self, p95_ms: _Optional[int] = ..., p50_ms: _Optional[int] = ...) -> None: ...

class DegradedQueryRateRequest(_message.Message):
    __slots__ = ("window",)
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ...) -> None: ...

class DegradedQueryRateResponse(_message.Message):
    __slots__ = ("rate", "degraded_queries", "total_queries")
    RATE_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_QUERIES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_QUERIES_FIELD_NUMBER: _ClassVar[int]
    rate: float
    degraded_queries: int
    total_queries: int
    def __init__(self, rate: _Optional[float] = ..., degraded_queries: _Optional[int] = ..., total_queries: _Optional[int] = ...) -> None: ...

class ProviderDegradationRateRequest(_message.Message):
    __slots__ = ("window", "provider_id")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    provider_id: str
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., provider_id: _Optional[str] = ...) -> None: ...

class ProviderDegradationRateResponse(_message.Message):
    __slots__ = ("rate", "degraded_count", "times_routed")
    RATE_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_COUNT_FIELD_NUMBER: _ClassVar[int]
    TIMES_ROUTED_FIELD_NUMBER: _ClassVar[int]
    rate: float
    degraded_count: int
    times_routed: int
    def __init__(self, rate: _Optional[float] = ..., degraded_count: _Optional[int] = ..., times_routed: _Optional[int] = ...) -> None: ...
