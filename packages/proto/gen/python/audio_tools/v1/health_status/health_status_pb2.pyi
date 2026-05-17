from audio_tools.v1.common import common_pb2 as _common_pb2
from audio_tools.v1.diagnostics import diagnostics_pb2 as _diagnostics_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class State(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    STATE_UNSPECIFIED: _ClassVar[State]
    STATE_AVAILABLE: _ClassVar[State]
    STATE_UNAVAILABLE: _ClassVar[State]
    STATE_UNKNOWN: _ClassVar[State]
STATE_UNSPECIFIED: State
STATE_AVAILABLE: State
STATE_UNAVAILABLE: State
STATE_UNKNOWN: State

class ProviderHealth(_message.Message):
    __slots__ = ("capability", "tier", "provider_id", "state", "last_checked_at", "latency_ms", "error_code", "error_message", "serving")
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    LAST_CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    ERROR_CODE_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SERVING_FIELD_NUMBER: _ClassVar[int]
    capability: _diagnostics_pb2.Capability
    tier: _common_pb2.ProviderTier
    provider_id: str
    state: State
    last_checked_at: str
    latency_ms: float
    error_code: str
    error_message: str
    serving: bool
    def __init__(self, capability: _Optional[_Union[_diagnostics_pb2.Capability, str]] = ..., tier: _Optional[_Union[_common_pb2.ProviderTier, str]] = ..., provider_id: _Optional[str] = ..., state: _Optional[_Union[State, str]] = ..., last_checked_at: _Optional[str] = ..., latency_ms: _Optional[float] = ..., error_code: _Optional[str] = ..., error_message: _Optional[str] = ..., serving: _Optional[bool] = ...) -> None: ...

class CapabilityHealth(_message.Message):
    __slots__ = ("capability", "providers", "effective_state")
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    PROVIDERS_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_STATE_FIELD_NUMBER: _ClassVar[int]
    capability: _diagnostics_pb2.Capability
    providers: _containers.RepeatedCompositeFieldContainer[ProviderHealth]
    effective_state: State
    def __init__(self, capability: _Optional[_Union[_diagnostics_pb2.Capability, str]] = ..., providers: _Optional[_Iterable[_Union[ProviderHealth, _Mapping]]] = ..., effective_state: _Optional[_Union[State, str]] = ...) -> None: ...

class GetProviderHealthRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetProviderHealthResponse(_message.Message):
    __slots__ = ("capabilities", "generated_at", "cache_ttl_seconds")
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    CACHE_TTL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    capabilities: _containers.RepeatedCompositeFieldContainer[CapabilityHealth]
    generated_at: str
    cache_ttl_seconds: int
    def __init__(self, capabilities: _Optional[_Iterable[_Union[CapabilityHealth, _Mapping]]] = ..., generated_at: _Optional[str] = ..., cache_ttl_seconds: _Optional[int] = ...) -> None: ...

class RefreshProviderHealthRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RefreshProviderHealthResponse(_message.Message):
    __slots__ = ("capabilities", "generated_at", "cache_ttl_seconds")
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    CACHE_TTL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    capabilities: _containers.RepeatedCompositeFieldContainer[CapabilityHealth]
    generated_at: str
    cache_ttl_seconds: int
    def __init__(self, capabilities: _Optional[_Iterable[_Union[CapabilityHealth, _Mapping]]] = ..., generated_at: _Optional[str] = ..., cache_ttl_seconds: _Optional[int] = ...) -> None: ...

class StreamProviderHealthRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ProviderHealthEvent(_message.Message):
    __slots__ = ("generated_at", "capabilities")
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    generated_at: str
    capabilities: _containers.RepeatedCompositeFieldContainer[CapabilityHealth]
    def __init__(self, generated_at: _Optional[str] = ..., capabilities: _Optional[_Iterable[_Union[CapabilityHealth, _Mapping]]] = ...) -> None: ...
