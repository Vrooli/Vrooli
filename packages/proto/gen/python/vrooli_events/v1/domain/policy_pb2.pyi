from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Effect(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EFFECT_UNSPECIFIED: _ClassVar[Effect]
    EFFECT_ALLOW: _ClassVar[Effect]
    EFFECT_DENY: _ClassVar[Effect]

class CircuitBreakerState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CIRCUIT_BREAKER_STATE_UNSPECIFIED: _ClassVar[CircuitBreakerState]
    CIRCUIT_BREAKER_STATE_CLOSED: _ClassVar[CircuitBreakerState]
    CIRCUIT_BREAKER_STATE_OPEN: _ClassVar[CircuitBreakerState]
    CIRCUIT_BREAKER_STATE_HALF_OPEN: _ClassVar[CircuitBreakerState]
EFFECT_UNSPECIFIED: Effect
EFFECT_ALLOW: Effect
EFFECT_DENY: Effect
CIRCUIT_BREAKER_STATE_UNSPECIFIED: CircuitBreakerState
CIRCUIT_BREAKER_STATE_CLOSED: CircuitBreakerState
CIRCUIT_BREAKER_STATE_OPEN: CircuitBreakerState
CIRCUIT_BREAKER_STATE_HALF_OPEN: CircuitBreakerState

class PolicyMatcher(_message.Message):
    __slots__ = ("source_pattern", "target_pattern", "action_pattern")
    SOURCE_PATTERN_FIELD_NUMBER: _ClassVar[int]
    TARGET_PATTERN_FIELD_NUMBER: _ClassVar[int]
    ACTION_PATTERN_FIELD_NUMBER: _ClassVar[int]
    source_pattern: str
    target_pattern: str
    action_pattern: str
    def __init__(self, source_pattern: _Optional[str] = ..., target_pattern: _Optional[str] = ..., action_pattern: _Optional[str] = ...) -> None: ...

class AccessRule(_message.Message):
    __slots__ = ("matcher", "effect", "specificity_score")
    MATCHER_FIELD_NUMBER: _ClassVar[int]
    EFFECT_FIELD_NUMBER: _ClassVar[int]
    SPECIFICITY_SCORE_FIELD_NUMBER: _ClassVar[int]
    matcher: PolicyMatcher
    effect: Effect
    specificity_score: int
    def __init__(self, matcher: _Optional[_Union[PolicyMatcher, _Mapping]] = ..., effect: _Optional[_Union[Effect, str]] = ..., specificity_score: _Optional[int] = ...) -> None: ...

class RateLimit(_message.Message):
    __slots__ = ("matcher", "capacity", "refill_rate")
    MATCHER_FIELD_NUMBER: _ClassVar[int]
    CAPACITY_FIELD_NUMBER: _ClassVar[int]
    REFILL_RATE_FIELD_NUMBER: _ClassVar[int]
    matcher: PolicyMatcher
    capacity: int
    refill_rate: float
    def __init__(self, matcher: _Optional[_Union[PolicyMatcher, _Mapping]] = ..., capacity: _Optional[int] = ..., refill_rate: _Optional[float] = ...) -> None: ...

class CircuitBreaker(_message.Message):
    __slots__ = ("matcher", "failure_threshold", "cooldown_seconds", "state")
    MATCHER_FIELD_NUMBER: _ClassVar[int]
    FAILURE_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    COOLDOWN_SECONDS_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    matcher: PolicyMatcher
    failure_threshold: int
    cooldown_seconds: int
    state: CircuitBreakerState
    def __init__(self, matcher: _Optional[_Union[PolicyMatcher, _Mapping]] = ..., failure_threshold: _Optional[int] = ..., cooldown_seconds: _Optional[int] = ..., state: _Optional[_Union[CircuitBreakerState, str]] = ...) -> None: ...
