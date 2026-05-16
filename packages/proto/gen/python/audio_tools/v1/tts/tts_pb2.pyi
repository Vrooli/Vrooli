from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SynthesizeRequest(_message.Message):
    __slots__ = ("text", "voice", "voice_overrides", "speed", "response_format", "event_id", "version")
    class VoiceOverridesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TEXT_FIELD_NUMBER: _ClassVar[int]
    VOICE_FIELD_NUMBER: _ClassVar[int]
    VOICE_OVERRIDES_FIELD_NUMBER: _ClassVar[int]
    SPEED_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_FORMAT_FIELD_NUMBER: _ClassVar[int]
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    text: str
    voice: str
    voice_overrides: _containers.ScalarMap[str, str]
    speed: float
    response_format: str
    event_id: str
    version: str
    def __init__(self, text: _Optional[str] = ..., voice: _Optional[str] = ..., voice_overrides: _Optional[_Mapping[str, str]] = ..., speed: _Optional[float] = ..., response_format: _Optional[str] = ..., event_id: _Optional[str] = ..., version: _Optional[str] = ...) -> None: ...

class SynthesizeResponse(_message.Message):
    __slots__ = ("audio", "content_type", "content_hash", "provider_tier", "provider_id", "model_id", "voice_used", "latency_ms")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_TIER_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    VOICE_USED_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    content_type: str
    content_hash: str
    provider_tier: str
    provider_id: str
    model_id: str
    voice_used: str
    latency_ms: float
    def __init__(self, audio: _Optional[bytes] = ..., content_type: _Optional[str] = ..., content_hash: _Optional[str] = ..., provider_tier: _Optional[str] = ..., provider_id: _Optional[str] = ..., model_id: _Optional[str] = ..., voice_used: _Optional[str] = ..., latency_ms: _Optional[float] = ...) -> None: ...

class Voice(_message.Message):
    __slots__ = ("id", "name", "description", "adapter_mappings")
    class AdapterMappingsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_MAPPINGS_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    adapter_mappings: _containers.ScalarMap[str, str]
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., adapter_mappings: _Optional[_Mapping[str, str]] = ...) -> None: ...

class ListVoicesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListVoicesResponse(_message.Message):
    __slots__ = ("voices",)
    VOICES_FIELD_NUMBER: _ClassVar[int]
    voices: _containers.RepeatedCompositeFieldContainer[Voice]
    def __init__(self, voices: _Optional[_Iterable[_Union[Voice, _Mapping]]] = ...) -> None: ...

class GetCacheRequest(_message.Message):
    __slots__ = ("event_id", "voice", "speed", "version", "content_hash", "response_format")
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    VOICE_FIELD_NUMBER: _ClassVar[int]
    SPEED_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_FORMAT_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    voice: str
    speed: float
    version: str
    content_hash: str
    response_format: str
    def __init__(self, event_id: _Optional[str] = ..., voice: _Optional[str] = ..., speed: _Optional[float] = ..., version: _Optional[str] = ..., content_hash: _Optional[str] = ..., response_format: _Optional[str] = ...) -> None: ...

class GetCacheResponse(_message.Message):
    __slots__ = ("audio", "content_type", "content_hash", "hit")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    HIT_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    content_type: str
    content_hash: str
    hit: bool
    def __init__(self, audio: _Optional[bytes] = ..., content_type: _Optional[str] = ..., content_hash: _Optional[str] = ..., hit: _Optional[bool] = ...) -> None: ...

class Config(_message.Message):
    __slots__ = ("auto_enabled", "default_voice", "default_speed", "default_response_format", "summarize_enabled", "summarize_char_threshold", "summarize_level", "summarize_model", "summarize_timeout_seconds")
    AUTO_ENABLED_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_VOICE_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_SPEED_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_RESPONSE_FORMAT_FIELD_NUMBER: _ClassVar[int]
    SUMMARIZE_ENABLED_FIELD_NUMBER: _ClassVar[int]
    SUMMARIZE_CHAR_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    SUMMARIZE_LEVEL_FIELD_NUMBER: _ClassVar[int]
    SUMMARIZE_MODEL_FIELD_NUMBER: _ClassVar[int]
    SUMMARIZE_TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    auto_enabled: bool
    default_voice: str
    default_speed: float
    default_response_format: str
    summarize_enabled: bool
    summarize_char_threshold: int
    summarize_level: str
    summarize_model: str
    summarize_timeout_seconds: int
    def __init__(self, auto_enabled: _Optional[bool] = ..., default_voice: _Optional[str] = ..., default_speed: _Optional[float] = ..., default_response_format: _Optional[str] = ..., summarize_enabled: _Optional[bool] = ..., summarize_char_threshold: _Optional[int] = ..., summarize_level: _Optional[str] = ..., summarize_model: _Optional[str] = ..., summarize_timeout_seconds: _Optional[int] = ...) -> None: ...

class GetConfigRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: Config
    def __init__(self, config: _Optional[_Union[Config, _Mapping]] = ...) -> None: ...

class UpdateConfigRequest(_message.Message):
    __slots__ = ("auto_enabled", "has_auto_enabled", "default_voice", "has_default_voice", "default_speed", "has_default_speed", "default_response_format", "has_default_response_format", "summarize_enabled", "has_summarize_enabled", "summarize_char_threshold", "has_summarize_char_threshold", "summarize_level", "has_summarize_level", "summarize_model", "has_summarize_model", "summarize_timeout_seconds", "has_summarize_timeout_seconds")
    AUTO_ENABLED_FIELD_NUMBER: _ClassVar[int]
    HAS_AUTO_ENABLED_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_VOICE_FIELD_NUMBER: _ClassVar[int]
    HAS_DEFAULT_VOICE_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_SPEED_FIELD_NUMBER: _ClassVar[int]
    HAS_DEFAULT_SPEED_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_RESPONSE_FORMAT_FIELD_NUMBER: _ClassVar[int]
    HAS_DEFAULT_RESPONSE_FORMAT_FIELD_NUMBER: _ClassVar[int]
    SUMMARIZE_ENABLED_FIELD_NUMBER: _ClassVar[int]
    HAS_SUMMARIZE_ENABLED_FIELD_NUMBER: _ClassVar[int]
    SUMMARIZE_CHAR_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    HAS_SUMMARIZE_CHAR_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    SUMMARIZE_LEVEL_FIELD_NUMBER: _ClassVar[int]
    HAS_SUMMARIZE_LEVEL_FIELD_NUMBER: _ClassVar[int]
    SUMMARIZE_MODEL_FIELD_NUMBER: _ClassVar[int]
    HAS_SUMMARIZE_MODEL_FIELD_NUMBER: _ClassVar[int]
    SUMMARIZE_TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    HAS_SUMMARIZE_TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    auto_enabled: bool
    has_auto_enabled: bool
    default_voice: str
    has_default_voice: bool
    default_speed: float
    has_default_speed: bool
    default_response_format: str
    has_default_response_format: bool
    summarize_enabled: bool
    has_summarize_enabled: bool
    summarize_char_threshold: int
    has_summarize_char_threshold: bool
    summarize_level: str
    has_summarize_level: bool
    summarize_model: str
    has_summarize_model: bool
    summarize_timeout_seconds: int
    has_summarize_timeout_seconds: bool
    def __init__(self, auto_enabled: _Optional[bool] = ..., has_auto_enabled: _Optional[bool] = ..., default_voice: _Optional[str] = ..., has_default_voice: _Optional[bool] = ..., default_speed: _Optional[float] = ..., has_default_speed: _Optional[bool] = ..., default_response_format: _Optional[str] = ..., has_default_response_format: _Optional[bool] = ..., summarize_enabled: _Optional[bool] = ..., has_summarize_enabled: _Optional[bool] = ..., summarize_char_threshold: _Optional[int] = ..., has_summarize_char_threshold: _Optional[bool] = ..., summarize_level: _Optional[str] = ..., has_summarize_level: _Optional[bool] = ..., summarize_model: _Optional[str] = ..., has_summarize_model: _Optional[bool] = ..., summarize_timeout_seconds: _Optional[int] = ..., has_summarize_timeout_seconds: _Optional[bool] = ...) -> None: ...

class UpdateConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: Config
    def __init__(self, config: _Optional[_Union[Config, _Mapping]] = ...) -> None: ...

class ProviderAvailability(_message.Message):
    __slots__ = ("tier", "provider_id", "available", "error", "checked_at")
    TIER_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    tier: str
    provider_id: str
    available: bool
    error: str
    checked_at: str
    def __init__(self, tier: _Optional[str] = ..., provider_id: _Optional[str] = ..., available: _Optional[bool] = ..., error: _Optional[str] = ..., checked_at: _Optional[str] = ...) -> None: ...

class Status(_message.Message):
    __slots__ = ("config", "availability", "capability", "capability_label")
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_LABEL_FIELD_NUMBER: _ClassVar[int]
    config: Config
    availability: _containers.RepeatedCompositeFieldContainer[ProviderAvailability]
    capability: str
    capability_label: str
    def __init__(self, config: _Optional[_Union[Config, _Mapping]] = ..., availability: _Optional[_Iterable[_Union[ProviderAvailability, _Mapping]]] = ..., capability: _Optional[str] = ..., capability_label: _Optional[str] = ...) -> None: ...

class GetStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetStatusResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: Status
    def __init__(self, status: _Optional[_Union[Status, _Mapping]] = ...) -> None: ...

class PlaybackEvent(_message.Message):
    __slots__ = ("source", "stage", "backend", "session_id", "message", "event_id")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    STAGE_FIELD_NUMBER: _ClassVar[int]
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    source: str
    stage: str
    backend: str
    session_id: str
    message: str
    event_id: str
    def __init__(self, source: _Optional[str] = ..., stage: _Optional[str] = ..., backend: _Optional[str] = ..., session_id: _Optional[str] = ..., message: _Optional[str] = ..., event_id: _Optional[str] = ...) -> None: ...

class RecordPlaybackEventRequest(_message.Message):
    __slots__ = ("event",)
    EVENT_FIELD_NUMBER: _ClassVar[int]
    event: PlaybackEvent
    def __init__(self, event: _Optional[_Union[PlaybackEvent, _Mapping]] = ...) -> None: ...

class RecordPlaybackEventResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class NormalizeForSpeechRequest(_message.Message):
    __slots__ = ("text", "voice")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    VOICE_FIELD_NUMBER: _ClassVar[int]
    text: str
    voice: str
    def __init__(self, text: _Optional[str] = ..., voice: _Optional[str] = ...) -> None: ...

class NormalizeForSpeechResponse(_message.Message):
    __slots__ = ("text",)
    TEXT_FIELD_NUMBER: _ClassVar[int]
    text: str
    def __init__(self, text: _Optional[str] = ...) -> None: ...

class SplitParagraphsRequest(_message.Message):
    __slots__ = ("text", "max_chars")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    MAX_CHARS_FIELD_NUMBER: _ClassVar[int]
    text: str
    max_chars: int
    def __init__(self, text: _Optional[str] = ..., max_chars: _Optional[int] = ...) -> None: ...

class SplitParagraphsResponse(_message.Message):
    __slots__ = ("paragraphs",)
    PARAGRAPHS_FIELD_NUMBER: _ClassVar[int]
    paragraphs: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, paragraphs: _Optional[_Iterable[str]] = ...) -> None: ...
