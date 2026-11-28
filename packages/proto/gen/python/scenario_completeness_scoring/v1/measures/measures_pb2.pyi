from measures.v1 import measures_pb2 as _measures_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RungThreshold(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RUNG_THRESHOLD_UNSPECIFIED: _ClassVar[RungThreshold]
    RUNG_THRESHOLD_R0: _ClassVar[RungThreshold]
    RUNG_THRESHOLD_R1: _ClassVar[RungThreshold]
    RUNG_THRESHOLD_R2: _ClassVar[RungThreshold]
    RUNG_THRESHOLD_R3: _ClassVar[RungThreshold]
    RUNG_THRESHOLD_R4: _ClassVar[RungThreshold]
RUNG_THRESHOLD_UNSPECIFIED: RungThreshold
RUNG_THRESHOLD_R0: RungThreshold
RUNG_THRESHOLD_R1: RungThreshold
RUNG_THRESHOLD_R2: RungThreshold
RUNG_THRESHOLD_R3: RungThreshold
RUNG_THRESHOLD_R4: RungThreshold

class CountFleetBelowRungRequest(_message.Message):
    __slots__ = ("window", "rung")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    RUNG_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    rung: RungThreshold
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., rung: _Optional[_Union[RungThreshold, str]] = ...) -> None: ...

class CountFleetBelowRungResponse(_message.Message):
    __slots__ = ("count",)
    COUNT_FIELD_NUMBER: _ClassVar[int]
    count: int
    def __init__(self, count: _Optional[int] = ...) -> None: ...

class AverageCompositeRequest(_message.Message):
    __slots__ = ("window",)
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ...) -> None: ...

class AverageCompositeResponse(_message.Message):
    __slots__ = ("average",)
    AVERAGE_FIELD_NUMBER: _ClassVar[int]
    average: float
    def __init__(self, average: _Optional[float] = ...) -> None: ...

class ScoreSeriesRequest(_message.Message):
    __slots__ = ("window",)
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ...) -> None: ...

class ScoreSeriesPoint(_message.Message):
    __slots__ = ("bucket", "average", "count")
    BUCKET_FIELD_NUMBER: _ClassVar[int]
    AVERAGE_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    bucket: str
    average: float
    count: int
    def __init__(self, bucket: _Optional[str] = ..., average: _Optional[float] = ..., count: _Optional[int] = ...) -> None: ...

class ScoreSeriesResponse(_message.Message):
    __slots__ = ("points",)
    POINTS_FIELD_NUMBER: _ClassVar[int]
    points: _containers.RepeatedCompositeFieldContainer[ScoreSeriesPoint]
    def __init__(self, points: _Optional[_Iterable[_Union[ScoreSeriesPoint, _Mapping]]] = ...) -> None: ...
