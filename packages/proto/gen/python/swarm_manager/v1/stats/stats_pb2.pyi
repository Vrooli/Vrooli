import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetPortfolioStatsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class PortfolioStats(_message.Message):
    __slots__ = ("observed_at", "swarm_throughput", "throughput_stats", "swarm_active_agents", "agent_stats", "timing_stats", "blocking_stats", "dashboard_stats", "composite_throughput", "review_stats", "scope_stats")
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    SWARM_THROUGHPUT_FIELD_NUMBER: _ClassVar[int]
    THROUGHPUT_STATS_FIELD_NUMBER: _ClassVar[int]
    SWARM_ACTIVE_AGENTS_FIELD_NUMBER: _ClassVar[int]
    AGENT_STATS_FIELD_NUMBER: _ClassVar[int]
    TIMING_STATS_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_STATS_FIELD_NUMBER: _ClassVar[int]
    DASHBOARD_STATS_FIELD_NUMBER: _ClassVar[int]
    COMPOSITE_THROUGHPUT_FIELD_NUMBER: _ClassVar[int]
    REVIEW_STATS_FIELD_NUMBER: _ClassVar[int]
    SCOPE_STATS_FIELD_NUMBER: _ClassVar[int]
    observed_at: _timestamp_pb2.Timestamp
    swarm_throughput: int
    throughput_stats: int
    swarm_active_agents: int
    agent_stats: float
    timing_stats: float
    blocking_stats: int
    dashboard_stats: int
    composite_throughput: int
    review_stats: int
    scope_stats: int
    def __init__(self, observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., swarm_throughput: _Optional[int] = ..., throughput_stats: _Optional[int] = ..., swarm_active_agents: _Optional[int] = ..., agent_stats: _Optional[float] = ..., timing_stats: _Optional[float] = ..., blocking_stats: _Optional[int] = ..., dashboard_stats: _Optional[int] = ..., composite_throughput: _Optional[int] = ..., review_stats: _Optional[int] = ..., scope_stats: _Optional[int] = ...) -> None: ...
