import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TunnelStatus(_message.Message):
    __slots__ = ("status", "systemd", "ready", "ready_latency_ms", "score", "message", "checked_at")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SYSTEMD_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    READY_LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    status: str
    systemd: str
    ready: str
    ready_latency_ms: int
    score: int
    message: str
    checked_at: _timestamp_pb2.Timestamp
    def __init__(self, status: _Optional[str] = ..., systemd: _Optional[str] = ..., ready: _Optional[str] = ..., ready_latency_ms: _Optional[int] = ..., score: _Optional[int] = ..., message: _Optional[str] = ..., checked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class MetricsSample(_message.Message):
    __slots__ = ("id", "ha_connections", "request_errors", "active_streams", "smoothed_rtt_ms", "scraped_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    HA_CONNECTIONS_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ERRORS_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_STREAMS_FIELD_NUMBER: _ClassVar[int]
    SMOOTHED_RTT_MS_FIELD_NUMBER: _ClassVar[int]
    SCRAPED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    ha_connections: int
    request_errors: float
    active_streams: int
    smoothed_rtt_ms: float
    scraped_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., ha_connections: _Optional[int] = ..., request_errors: _Optional[float] = ..., active_streams: _Optional[int] = ..., smoothed_rtt_ms: _Optional[float] = ..., scraped_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetStatusResponse(_message.Message):
    __slots__ = ("status", "latest_metrics")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LATEST_METRICS_FIELD_NUMBER: _ClassVar[int]
    status: TunnelStatus
    latest_metrics: MetricsSample
    def __init__(self, status: _Optional[_Union[TunnelStatus, _Mapping]] = ..., latest_metrics: _Optional[_Union[MetricsSample, _Mapping]] = ...) -> None: ...

class ListMetricsRequest(_message.Message):
    __slots__ = ("to",)
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    to: _timestamp_pb2.Timestamp
    def __init__(self, to: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., **kwargs) -> None: ...

class ListMetricsResponse(_message.Message):
    __slots__ = ("samples",)
    SAMPLES_FIELD_NUMBER: _ClassVar[int]
    samples: _containers.RepeatedCompositeFieldContainer[MetricsSample]
    def __init__(self, samples: _Optional[_Iterable[_Union[MetricsSample, _Mapping]]] = ...) -> None: ...

class ScrapeRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ScrapeResponse(_message.Message):
    __slots__ = ("sample",)
    SAMPLE_FIELD_NUMBER: _ClassVar[int]
    sample: MetricsSample
    def __init__(self, sample: _Optional[_Union[MetricsSample, _Mapping]] = ...) -> None: ...
