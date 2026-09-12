from audio_tools.v1.common import common_pb2 as _common_pb2
from audio_tools.v1.diagnostics import diagnostics_pb2 as _diagnostics_pb2
from audio_tools.v1.shared import shared_pb2 as _shared_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CapabilityHealth(_message.Message):
    __slots__ = ("capability", "providers", "effective_state")
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    PROVIDERS_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_STATE_FIELD_NUMBER: _ClassVar[int]
    capability: _diagnostics_pb2.Capability
    providers: _containers.RepeatedCompositeFieldContainer[_shared_pb2.ProviderHealth]
    effective_state: _shared_pb2.ProviderState
    def __init__(self, capability: _Optional[_Union[_diagnostics_pb2.Capability, str]] = ..., providers: _Optional[_Iterable[_Union[_shared_pb2.ProviderHealth, _Mapping]]] = ..., effective_state: _Optional[_Union[_shared_pb2.ProviderState, str]] = ...) -> None: ...

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
