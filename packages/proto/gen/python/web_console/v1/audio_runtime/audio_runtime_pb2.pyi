from buf.validate import validate_pb2 as _validate_pb2
from web_console.v1.audio_common import audio_common_pb2 as _audio_common_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TranscribeRequest(_message.Message):
    __slots__ = ("audio", "format", "language", "skip_speaker_verification", "initial_prompt")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    SKIP_SPEAKER_VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    INITIAL_PROMPT_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    format: _audio_common_pb2.AudioFormat
    language: str
    skip_speaker_verification: bool
    initial_prompt: str
    def __init__(self, audio: _Optional[bytes] = ..., format: _Optional[_Union[_audio_common_pb2.AudioFormat, str]] = ..., language: _Optional[str] = ..., skip_speaker_verification: _Optional[bool] = ..., initial_prompt: _Optional[str] = ...) -> None: ...

class TranscribeResponse(_message.Message):
    __slots__ = ("text", "detected_language", "duration_seconds", "provider_tier", "provider_id", "model_id", "latency_ms")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    DETECTED_LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_TIER_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    text: str
    detected_language: str
    duration_seconds: float
    provider_tier: _audio_common_pb2.ProviderTier
    provider_id: str
    model_id: str
    latency_ms: float
    def __init__(self, text: _Optional[str] = ..., detected_language: _Optional[str] = ..., duration_seconds: _Optional[float] = ..., provider_tier: _Optional[_Union[_audio_common_pb2.ProviderTier, str]] = ..., provider_id: _Optional[str] = ..., model_id: _Optional[str] = ..., latency_ms: _Optional[float] = ...) -> None: ...

class AdapterMapping(_message.Message):
    __slots__ = ("tier", "provider_id", "backend_voice_id")
    TIER_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    BACKEND_VOICE_ID_FIELD_NUMBER: _ClassVar[int]
    tier: _audio_common_pb2.ProviderTier
    provider_id: str
    backend_voice_id: str
    def __init__(self, tier: _Optional[_Union[_audio_common_pb2.ProviderTier, str]] = ..., provider_id: _Optional[str] = ..., backend_voice_id: _Optional[str] = ...) -> None: ...

class SynthesizeRequest(_message.Message):
    __slots__ = ("text", "voice", "voice_overrides", "speed", "response_format", "event_id", "version", "chunk_index")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    VOICE_FIELD_NUMBER: _ClassVar[int]
    VOICE_OVERRIDES_FIELD_NUMBER: _ClassVar[int]
    SPEED_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_FORMAT_FIELD_NUMBER: _ClassVar[int]
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CHUNK_INDEX_FIELD_NUMBER: _ClassVar[int]
    text: str
    voice: str
    voice_overrides: _containers.RepeatedCompositeFieldContainer[AdapterMapping]
    speed: float
    response_format: _audio_common_pb2.ResponseFormat
    event_id: str
    version: str
    chunk_index: int
    def __init__(self, text: _Optional[str] = ..., voice: _Optional[str] = ..., voice_overrides: _Optional[_Iterable[_Union[AdapterMapping, _Mapping]]] = ..., speed: _Optional[float] = ..., response_format: _Optional[_Union[_audio_common_pb2.ResponseFormat, str]] = ..., event_id: _Optional[str] = ..., version: _Optional[str] = ..., chunk_index: _Optional[int] = ...) -> None: ...

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
    provider_tier: _audio_common_pb2.ProviderTier
    provider_id: str
    model_id: str
    voice_used: str
    latency_ms: float
    def __init__(self, audio: _Optional[bytes] = ..., content_type: _Optional[str] = ..., content_hash: _Optional[str] = ..., provider_tier: _Optional[_Union[_audio_common_pb2.ProviderTier, str]] = ..., provider_id: _Optional[str] = ..., model_id: _Optional[str] = ..., voice_used: _Optional[str] = ..., latency_ms: _Optional[float] = ...) -> None: ...

class Voice(_message.Message):
    __slots__ = ("id", "name", "description", "adapter_mappings")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_MAPPINGS_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    adapter_mappings: _containers.RepeatedCompositeFieldContainer[AdapterMapping]
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., adapter_mappings: _Optional[_Iterable[_Union[AdapterMapping, _Mapping]]] = ...) -> None: ...

class ListVoicesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListVoicesResponse(_message.Message):
    __slots__ = ("voices",)
    VOICES_FIELD_NUMBER: _ClassVar[int]
    voices: _containers.RepeatedCompositeFieldContainer[Voice]
    def __init__(self, voices: _Optional[_Iterable[_Union[Voice, _Mapping]]] = ...) -> None: ...

class GetTTSCacheRequest(_message.Message):
    __slots__ = ("event_id", "voice", "speed", "version", "content_hash", "response_format", "chunk_index")
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    VOICE_FIELD_NUMBER: _ClassVar[int]
    SPEED_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_FORMAT_FIELD_NUMBER: _ClassVar[int]
    CHUNK_INDEX_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    voice: str
    speed: float
    version: str
    content_hash: str
    response_format: _audio_common_pb2.ResponseFormat
    chunk_index: int
    def __init__(self, event_id: _Optional[str] = ..., voice: _Optional[str] = ..., speed: _Optional[float] = ..., version: _Optional[str] = ..., content_hash: _Optional[str] = ..., response_format: _Optional[_Union[_audio_common_pb2.ResponseFormat, str]] = ..., chunk_index: _Optional[int] = ...) -> None: ...

class GetTTSCacheResponse(_message.Message):
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

class SummarizeRequest(_message.Message):
    __slots__ = ("text", "level", "model", "timeout_seconds")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    text: str
    level: _audio_common_pb2.SummarizeLevel
    model: str
    timeout_seconds: int
    def __init__(self, text: _Optional[str] = ..., level: _Optional[_Union[_audio_common_pb2.SummarizeLevel, str]] = ..., model: _Optional[str] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class SummarizeResponse(_message.Message):
    __slots__ = ("text", "prompt_tokens", "output_tokens", "provider_tier", "provider_id", "model_id", "latency_ms")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    PROMPT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_TIER_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    text: str
    prompt_tokens: int
    output_tokens: int
    provider_tier: _audio_common_pb2.ProviderTier
    provider_id: str
    model_id: str
    latency_ms: float
    def __init__(self, text: _Optional[str] = ..., prompt_tokens: _Optional[int] = ..., output_tokens: _Optional[int] = ..., provider_tier: _Optional[_Union[_audio_common_pb2.ProviderTier, str]] = ..., provider_id: _Optional[str] = ..., model_id: _Optional[str] = ..., latency_ms: _Optional[float] = ...) -> None: ...
