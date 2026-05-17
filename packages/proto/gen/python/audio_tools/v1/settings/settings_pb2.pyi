import datetime

from google.protobuf import field_mask_pb2 as _field_mask_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ProviderConfig(_message.Message):
    __slots__ = ("byok_enabled", "vrooli_enabled", "local_enabled", "whisper_url", "kokoro_url", "ollama_url", "lpbs_base_url", "lpbs_app_bundle_key", "avail_ttl_byok_seconds", "avail_ttl_vrooli_seconds")
    BYOK_ENABLED_FIELD_NUMBER: _ClassVar[int]
    VROOLI_ENABLED_FIELD_NUMBER: _ClassVar[int]
    LOCAL_ENABLED_FIELD_NUMBER: _ClassVar[int]
    WHISPER_URL_FIELD_NUMBER: _ClassVar[int]
    KOKORO_URL_FIELD_NUMBER: _ClassVar[int]
    OLLAMA_URL_FIELD_NUMBER: _ClassVar[int]
    LPBS_BASE_URL_FIELD_NUMBER: _ClassVar[int]
    LPBS_APP_BUNDLE_KEY_FIELD_NUMBER: _ClassVar[int]
    AVAIL_TTL_BYOK_SECONDS_FIELD_NUMBER: _ClassVar[int]
    AVAIL_TTL_VROOLI_SECONDS_FIELD_NUMBER: _ClassVar[int]
    byok_enabled: bool
    vrooli_enabled: bool
    local_enabled: bool
    whisper_url: str
    kokoro_url: str
    ollama_url: str
    lpbs_base_url: str
    lpbs_app_bundle_key: str
    avail_ttl_byok_seconds: int
    avail_ttl_vrooli_seconds: int
    def __init__(self, byok_enabled: _Optional[bool] = ..., vrooli_enabled: _Optional[bool] = ..., local_enabled: _Optional[bool] = ..., whisper_url: _Optional[str] = ..., kokoro_url: _Optional[str] = ..., ollama_url: _Optional[str] = ..., lpbs_base_url: _Optional[str] = ..., lpbs_app_bundle_key: _Optional[str] = ..., avail_ttl_byok_seconds: _Optional[int] = ..., avail_ttl_vrooli_seconds: _Optional[int] = ...) -> None: ...

class GetProviderConfigRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetProviderConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: ProviderConfig
    def __init__(self, config: _Optional[_Union[ProviderConfig, _Mapping]] = ...) -> None: ...

class UpdateProviderConfigRequest(_message.Message):
    __slots__ = ("update_mask", "config")
    UPDATE_MASK_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    update_mask: _field_mask_pb2.FieldMask
    config: ProviderConfig
    def __init__(self, update_mask: _Optional[_Union[_field_mask_pb2.FieldMask, _Mapping]] = ..., config: _Optional[_Union[ProviderConfig, _Mapping]] = ...) -> None: ...

class UpdateProviderConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: ProviderConfig
    def __init__(self, config: _Optional[_Union[ProviderConfig, _Mapping]] = ...) -> None: ...

class BYOKCredentialSummary(_message.Message):
    __slots__ = ("provider_id", "capability", "fingerprint", "created_at", "last_used_at")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_USED_AT_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    capability: str
    fingerprint: str
    created_at: _timestamp_pb2.Timestamp
    last_used_at: _timestamp_pb2.Timestamp
    def __init__(self, provider_id: _Optional[str] = ..., capability: _Optional[str] = ..., fingerprint: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_used_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListBYOKCredentialsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListBYOKCredentialsResponse(_message.Message):
    __slots__ = ("credentials",)
    CREDENTIALS_FIELD_NUMBER: _ClassVar[int]
    credentials: _containers.RepeatedCompositeFieldContainer[BYOKCredentialSummary]
    def __init__(self, credentials: _Optional[_Iterable[_Union[BYOKCredentialSummary, _Mapping]]] = ...) -> None: ...

class UpsertBYOKCredentialRequest(_message.Message):
    __slots__ = ("provider_id", "capability", "api_key")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    API_KEY_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    capability: str
    api_key: str
    def __init__(self, provider_id: _Optional[str] = ..., capability: _Optional[str] = ..., api_key: _Optional[str] = ...) -> None: ...

class UpsertBYOKCredentialResponse(_message.Message):
    __slots__ = ("credential",)
    CREDENTIAL_FIELD_NUMBER: _ClassVar[int]
    credential: BYOKCredentialSummary
    def __init__(self, credential: _Optional[_Union[BYOKCredentialSummary, _Mapping]] = ...) -> None: ...

class DeleteBYOKCredentialRequest(_message.Message):
    __slots__ = ("provider_id", "capability")
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    capability: str
    def __init__(self, provider_id: _Optional[str] = ..., capability: _Optional[str] = ...) -> None: ...

class DeleteBYOKCredentialResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class VoiceOverride(_message.Message):
    __slots__ = ("canonical_voice", "tier_provider", "adapter_voice")
    CANONICAL_VOICE_FIELD_NUMBER: _ClassVar[int]
    TIER_PROVIDER_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_VOICE_FIELD_NUMBER: _ClassVar[int]
    canonical_voice: str
    tier_provider: str
    adapter_voice: str
    def __init__(self, canonical_voice: _Optional[str] = ..., tier_provider: _Optional[str] = ..., adapter_voice: _Optional[str] = ...) -> None: ...

class GetVoiceOverridesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetVoiceOverridesResponse(_message.Message):
    __slots__ = ("overrides",)
    OVERRIDES_FIELD_NUMBER: _ClassVar[int]
    overrides: _containers.RepeatedCompositeFieldContainer[VoiceOverride]
    def __init__(self, overrides: _Optional[_Iterable[_Union[VoiceOverride, _Mapping]]] = ...) -> None: ...

class SetVoiceOverrideRequest(_message.Message):
    __slots__ = ("override",)
    OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    override: VoiceOverride
    def __init__(self, override: _Optional[_Union[VoiceOverride, _Mapping]] = ...) -> None: ...

class SetVoiceOverrideResponse(_message.Message):
    __slots__ = ("overrides",)
    OVERRIDES_FIELD_NUMBER: _ClassVar[int]
    overrides: _containers.RepeatedCompositeFieldContainer[VoiceOverride]
    def __init__(self, overrides: _Optional[_Iterable[_Union[VoiceOverride, _Mapping]]] = ...) -> None: ...
