from measures.v1 import measures_pb2 as _measures_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RouteMeasureRequest(_message.Message):
    __slots__ = ("window",)
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ...) -> None: ...

class RouteCountResponse(_message.Message):
    __slots__ = ("count",)
    COUNT_FIELD_NUMBER: _ClassVar[int]
    count: int
    def __init__(self, count: _Optional[int] = ...) -> None: ...

class RouteRateResponse(_message.Message):
    __slots__ = ("rate",)
    RATE_FIELD_NUMBER: _ClassVar[int]
    rate: float
    def __init__(self, rate: _Optional[float] = ...) -> None: ...

class RouteLatencyResponse(_message.Message):
    __slots__ = ("latency_ms",)
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    latency_ms: int
    def __init__(self, latency_ms: _Optional[int] = ...) -> None: ...
