from measures.v1 import measures_pb2 as _measures_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class OpenDebtCountRequest(_message.Message):
    __slots__ = ("window",)
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ...) -> None: ...

class OpenDebtCountResponse(_message.Message):
    __slots__ = ("count",)
    COUNT_FIELD_NUMBER: _ClassVar[int]
    count: int
    def __init__(self, count: _Optional[int] = ...) -> None: ...

class DeepValidateGreenStreakRequest(_message.Message):
    __slots__ = ("template_id",)
    TEMPLATE_ID_FIELD_NUMBER: _ClassVar[int]
    template_id: str
    def __init__(self, template_id: _Optional[str] = ...) -> None: ...

class DeepValidateGreenStreakResponse(_message.Message):
    __slots__ = ("streak",)
    STREAK_FIELD_NUMBER: _ClassVar[int]
    streak: int
    def __init__(self, streak: _Optional[int] = ...) -> None: ...

class FleetStandingDistributionRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class FleetStandingBucket(_message.Message):
    __slots__ = ("standing", "count")
    STANDING_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    standing: str
    count: int
    def __init__(self, standing: _Optional[str] = ..., count: _Optional[int] = ...) -> None: ...

class FleetStandingDistributionResponse(_message.Message):
    __slots__ = ("buckets",)
    BUCKETS_FIELD_NUMBER: _ClassVar[int]
    buckets: _containers.RepeatedCompositeFieldContainer[FleetStandingBucket]
    def __init__(self, buckets: _Optional[_Iterable[_Union[FleetStandingBucket, _Mapping]]] = ...) -> None: ...

class MaxVersionLagRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class MaxVersionLagResponse(_message.Message):
    __slots__ = ("lag",)
    LAG_FIELD_NUMBER: _ClassVar[int]
    lag: int
    def __init__(self, lag: _Optional[int] = ...) -> None: ...
