from portal.v1.shared import common_pb2 as _common_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class BehaviorOverride(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    BEHAVIOR_OVERRIDE_UNSPECIFIED: _ClassVar[BehaviorOverride]
    BEHAVIOR_OVERRIDE_AUTO: _ClassVar[BehaviorOverride]
    BEHAVIOR_OVERRIDE_FORCE_OFF: _ClassVar[BehaviorOverride]
    BEHAVIOR_OVERRIDE_FORCE_PASSIVE: _ClassVar[BehaviorOverride]
BEHAVIOR_OVERRIDE_UNSPECIFIED: BehaviorOverride
BEHAVIOR_OVERRIDE_AUTO: BehaviorOverride
BEHAVIOR_OVERRIDE_FORCE_OFF: BehaviorOverride
BEHAVIOR_OVERRIDE_FORCE_PASSIVE: BehaviorOverride

class RollingStats(_message.Message):
    __slots__ = ("latency_p50_ms", "latency_p95_ms", "error_rate", "degraded_rate", "last_ok_at", "sample_count")
    LATENCY_P50_MS_FIELD_NUMBER: _ClassVar[int]
    LATENCY_P95_MS_FIELD_NUMBER: _ClassVar[int]
    ERROR_RATE_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_RATE_FIELD_NUMBER: _ClassVar[int]
    LAST_OK_AT_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    latency_p50_ms: float
    latency_p95_ms: float
    error_rate: float
    degraded_rate: float
    last_ok_at: str
    sample_count: int
    def __init__(self, latency_p50_ms: _Optional[float] = ..., latency_p95_ms: _Optional[float] = ..., error_rate: _Optional[float] = ..., degraded_rate: _Optional[float] = ..., last_ok_at: _Optional[str] = ..., sample_count: _Optional[int] = ...) -> None: ...

class IntegrationStatus(_message.Message):
    __slots__ = ("id", "display_name", "state", "stats", "reason", "required")
    ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    STATS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    id: str
    display_name: str
    state: _common_pb2.IntegrationState
    stats: RollingStats
    reason: str
    required: bool
    def __init__(self, id: _Optional[str] = ..., display_name: _Optional[str] = ..., state: _Optional[_Union[_common_pb2.IntegrationState, str]] = ..., stats: _Optional[_Union[RollingStats, _Mapping]] = ..., reason: _Optional[str] = ..., required: _Optional[bool] = ...) -> None: ...

class StatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class StatusResponse(_message.Message):
    __slots__ = ("integrations", "active_mode", "override", "reason", "evaluated_at")
    INTEGRATIONS_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_MODE_FIELD_NUMBER: _ClassVar[int]
    OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EVALUATED_AT_FIELD_NUMBER: _ClassVar[int]
    integrations: _containers.RepeatedCompositeFieldContainer[IntegrationStatus]
    active_mode: _common_pb2.BehaviorMode
    override: BehaviorOverride
    reason: str
    evaluated_at: str
    def __init__(self, integrations: _Optional[_Iterable[_Union[IntegrationStatus, _Mapping]]] = ..., active_mode: _Optional[_Union[_common_pb2.BehaviorMode, str]] = ..., override: _Optional[_Union[BehaviorOverride, str]] = ..., reason: _Optional[str] = ..., evaluated_at: _Optional[str] = ...) -> None: ...

class UpdateOverrideRequest(_message.Message):
    __slots__ = ("override",)
    OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    override: BehaviorOverride
    def __init__(self, override: _Optional[_Union[BehaviorOverride, str]] = ...) -> None: ...

class UpdateOverrideResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: StatusResponse
    def __init__(self, status: _Optional[_Union[StatusResponse, _Mapping]] = ...) -> None: ...
