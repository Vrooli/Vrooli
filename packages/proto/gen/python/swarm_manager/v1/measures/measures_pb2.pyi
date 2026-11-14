from measures.v1 import measures_pb2 as _measures_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CountBacklogCompletedRequest(_message.Message):
    __slots__ = ("window",)
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ...) -> None: ...

class CountBacklogCompletedResponse(_message.Message):
    __slots__ = ("count",)
    COUNT_FIELD_NUMBER: _ClassVar[int]
    count: int
    def __init__(self, count: _Optional[int] = ...) -> None: ...

class CountBacklogCreatedRequest(_message.Message):
    __slots__ = ("window",)
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ...) -> None: ...

class CountBacklogCreatedResponse(_message.Message):
    __slots__ = ("count",)
    COUNT_FIELD_NUMBER: _ClassVar[int]
    count: int
    def __init__(self, count: _Optional[int] = ...) -> None: ...

class CountBacklogNetDeltaRequest(_message.Message):
    __slots__ = ("window",)
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ...) -> None: ...

class CountBacklogNetDeltaResponse(_message.Message):
    __slots__ = ("count",)
    COUNT_FIELD_NUMBER: _ClassVar[int]
    count: int
    def __init__(self, count: _Optional[int] = ...) -> None: ...

class CountBacklogOpenRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class CountBacklogOpenResponse(_message.Message):
    __slots__ = ("count",)
    COUNT_FIELD_NUMBER: _ClassVar[int]
    count: int
    def __init__(self, count: _Optional[int] = ...) -> None: ...

class CountBacklogBlockedRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class CountBacklogBlockedResponse(_message.Message):
    __slots__ = ("count",)
    COUNT_FIELD_NUMBER: _ClassVar[int]
    count: int
    def __init__(self, count: _Optional[int] = ...) -> None: ...

class BacklogLeadTimeRequest(_message.Message):
    __slots__ = ("window",)
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ...) -> None: ...

class BacklogLeadTimeResponse(_message.Message):
    __slots__ = ("average_hours", "median_hours", "sample_size")
    AVERAGE_HOURS_FIELD_NUMBER: _ClassVar[int]
    MEDIAN_HOURS_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_SIZE_FIELD_NUMBER: _ClassVar[int]
    average_hours: float
    median_hours: float
    sample_size: int
    def __init__(self, average_hours: _Optional[float] = ..., median_hours: _Optional[float] = ..., sample_size: _Optional[int] = ...) -> None: ...

class CountExecutionsCompletedRequest(_message.Message):
    __slots__ = ("window",)
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ...) -> None: ...

class CountExecutionsCompletedResponse(_message.Message):
    __slots__ = ("count",)
    COUNT_FIELD_NUMBER: _ClassVar[int]
    count: int
    def __init__(self, count: _Optional[int] = ...) -> None: ...

class CountGoalsCreatedRequest(_message.Message):
    __slots__ = ("window",)
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ...) -> None: ...

class CountGoalsCreatedResponse(_message.Message):
    __slots__ = ("count",)
    COUNT_FIELD_NUMBER: _ClassVar[int]
    count: int
    def __init__(self, count: _Optional[int] = ...) -> None: ...

class CountAgentSessionsCreatedRequest(_message.Message):
    __slots__ = ("window",)
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ...) -> None: ...

class CountAgentSessionsCreatedResponse(_message.Message):
    __slots__ = ("count",)
    COUNT_FIELD_NUMBER: _ClassVar[int]
    count: int
    def __init__(self, count: _Optional[int] = ...) -> None: ...

class CountPlanRefsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class CountPlanRefsResponse(_message.Message):
    __slots__ = ("count",)
    COUNT_FIELD_NUMBER: _ClassVar[int]
    count: int
    def __init__(self, count: _Optional[int] = ...) -> None: ...

class GoalMilestoneHealthRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class MilestoneHealth(_message.Message):
    __slots__ = ("milestone", "total", "completed", "in_progress", "blocked")
    MILESTONE_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_FIELD_NUMBER: _ClassVar[int]
    IN_PROGRESS_FIELD_NUMBER: _ClassVar[int]
    BLOCKED_FIELD_NUMBER: _ClassVar[int]
    milestone: str
    total: int
    completed: int
    in_progress: int
    blocked: int
    def __init__(self, milestone: _Optional[str] = ..., total: _Optional[int] = ..., completed: _Optional[int] = ..., in_progress: _Optional[int] = ..., blocked: _Optional[int] = ...) -> None: ...

class GoalMilestoneHealthResponse(_message.Message):
    __slots__ = ("milestones",)
    MILESTONES_FIELD_NUMBER: _ClassVar[int]
    milestones: _containers.RepeatedCompositeFieldContainer[MilestoneHealth]
    def __init__(self, milestones: _Optional[_Iterable[_Union[MilestoneHealth, _Mapping]]] = ...) -> None: ...

class AgentSessionProposalRateRequest(_message.Message):
    __slots__ = ("window",)
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ...) -> None: ...

class AgentSessionProposalRateResponse(_message.Message):
    __slots__ = ("rate", "sample_size")
    RATE_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_SIZE_FIELD_NUMBER: _ClassVar[int]
    rate: float
    sample_size: int
    def __init__(self, rate: _Optional[float] = ..., sample_size: _Optional[int] = ...) -> None: ...

class ExecutionSuccessRateRequest(_message.Message):
    __slots__ = ("window",)
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ...) -> None: ...

class ExecutionSuccessRateResponse(_message.Message):
    __slots__ = ("rate", "sample_size")
    RATE_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_SIZE_FIELD_NUMBER: _ClassVar[int]
    rate: float
    sample_size: int
    def __init__(self, rate: _Optional[float] = ..., sample_size: _Optional[int] = ...) -> None: ...

class ExecutionDurationRequest(_message.Message):
    __slots__ = ("window",)
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ...) -> None: ...

class ExecutionDurationResponse(_message.Message):
    __slots__ = ("average_minutes", "median_minutes", "sample_size")
    AVERAGE_MINUTES_FIELD_NUMBER: _ClassVar[int]
    MEDIAN_MINUTES_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_SIZE_FIELD_NUMBER: _ClassVar[int]
    average_minutes: float
    median_minutes: float
    sample_size: int
    def __init__(self, average_minutes: _Optional[float] = ..., median_minutes: _Optional[float] = ..., sample_size: _Optional[int] = ...) -> None: ...

class ExecutionReviewRateRequest(_message.Message):
    __slots__ = ("window",)
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ...) -> None: ...

class ExecutionReviewRateResponse(_message.Message):
    __slots__ = ("rate", "sample_size")
    RATE_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_SIZE_FIELD_NUMBER: _ClassVar[int]
    rate: float
    sample_size: int
    def __init__(self, rate: _Optional[float] = ..., sample_size: _Optional[int] = ...) -> None: ...
