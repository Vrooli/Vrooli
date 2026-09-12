from audio_tools.v1.common import common_pb2 as _common_pb2
from audio_tools.v1.diagnostics import diagnostics_pb2 as _diagnostics_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ProviderState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PROVIDER_STATE_UNSPECIFIED: _ClassVar[ProviderState]
    PROVIDER_STATE_AVAILABLE: _ClassVar[ProviderState]
    PROVIDER_STATE_UNAVAILABLE: _ClassVar[ProviderState]
    PROVIDER_STATE_UNKNOWN: _ClassVar[ProviderState]
PROVIDER_STATE_UNSPECIFIED: ProviderState
PROVIDER_STATE_AVAILABLE: ProviderState
PROVIDER_STATE_UNAVAILABLE: ProviderState
PROVIDER_STATE_UNKNOWN: ProviderState

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
    state: ProviderState
    last_checked_at: str
    latency_ms: float
    error_code: str
    error_message: str
    serving: bool
    def __init__(self, capability: _Optional[_Union[_diagnostics_pb2.Capability, str]] = ..., tier: _Optional[_Union[_common_pb2.ProviderTier, str]] = ..., provider_id: _Optional[str] = ..., state: _Optional[_Union[ProviderState, str]] = ..., last_checked_at: _Optional[str] = ..., latency_ms: _Optional[float] = ..., error_code: _Optional[str] = ..., error_message: _Optional[str] = ..., serving: _Optional[bool] = ...) -> None: ...
