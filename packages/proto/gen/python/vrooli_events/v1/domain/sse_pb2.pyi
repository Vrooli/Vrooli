import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from vrooli_events.v1.domain import envelope_pb2 as _envelope_pb2
from vrooli_events.v1.domain import policy_pb2 as _policy_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SubscriptionRequest(_message.Message):
    __slots__ = ("event_type_pattern", "source_scenario_pattern", "target_scenario_pattern")
    EVENT_TYPE_PATTERN_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SCENARIO_PATTERN_FIELD_NUMBER: _ClassVar[int]
    TARGET_SCENARIO_PATTERN_FIELD_NUMBER: _ClassVar[int]
    event_type_pattern: str
    source_scenario_pattern: str
    target_scenario_pattern: str
    def __init__(self, event_type_pattern: _Optional[str] = ..., source_scenario_pattern: _Optional[str] = ..., target_scenario_pattern: _Optional[str] = ...) -> None: ...

class EventNotification(_message.Message):
    __slots__ = ("stream_sequence", "envelope")
    STREAM_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    ENVELOPE_FIELD_NUMBER: _ClassVar[int]
    stream_sequence: int
    envelope: _envelope_pb2.EventEnvelope
    def __init__(self, stream_sequence: _Optional[int] = ..., envelope: _Optional[_Union[_envelope_pb2.EventEnvelope, _Mapping]] = ...) -> None: ...

class PolicySnapshot(_message.Message):
    __slots__ = ("version", "generated_at", "access_rules", "rate_limits", "circuit_breakers")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    ACCESS_RULES_FIELD_NUMBER: _ClassVar[int]
    RATE_LIMITS_FIELD_NUMBER: _ClassVar[int]
    CIRCUIT_BREAKERS_FIELD_NUMBER: _ClassVar[int]
    version: int
    generated_at: _timestamp_pb2.Timestamp
    access_rules: _containers.RepeatedCompositeFieldContainer[_policy_pb2.AccessRule]
    rate_limits: _containers.RepeatedCompositeFieldContainer[_policy_pb2.RateLimit]
    circuit_breakers: _containers.RepeatedCompositeFieldContainer[_policy_pb2.CircuitBreaker]
    def __init__(self, version: _Optional[int] = ..., generated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., access_rules: _Optional[_Iterable[_Union[_policy_pb2.AccessRule, _Mapping]]] = ..., rate_limits: _Optional[_Iterable[_Union[_policy_pb2.RateLimit, _Mapping]]] = ..., circuit_breakers: _Optional[_Iterable[_Union[_policy_pb2.CircuitBreaker, _Mapping]]] = ...) -> None: ...

class HeartbeatMessage(_message.Message):
    __slots__ = ("timestamp", "dropped_count")
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    DROPPED_COUNT_FIELD_NUMBER: _ClassVar[int]
    timestamp: _timestamp_pb2.Timestamp
    dropped_count: int
    def __init__(self, timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., dropped_count: _Optional[int] = ...) -> None: ...
