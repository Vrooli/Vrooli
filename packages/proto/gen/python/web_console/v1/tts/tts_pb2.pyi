from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Config(_message.Message):
    __slots__ = ("auto_enabled", "backend", "kokoro_voice", "kokoro_speed")
    AUTO_ENABLED_FIELD_NUMBER: _ClassVar[int]
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    KOKORO_VOICE_FIELD_NUMBER: _ClassVar[int]
    KOKORO_SPEED_FIELD_NUMBER: _ClassVar[int]
    auto_enabled: bool
    backend: str
    kokoro_voice: str
    kokoro_speed: float
    def __init__(self, auto_enabled: _Optional[bool] = ..., backend: _Optional[str] = ..., kokoro_voice: _Optional[str] = ..., kokoro_speed: _Optional[float] = ...) -> None: ...

class GetConfigRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: Config
    def __init__(self, config: _Optional[_Union[Config, _Mapping]] = ...) -> None: ...

class UpdateConfigRequest(_message.Message):
    __slots__ = ("auto_enabled", "has_auto_enabled", "backend", "has_backend", "kokoro_voice", "has_kokoro_voice", "kokoro_speed", "has_kokoro_speed")
    AUTO_ENABLED_FIELD_NUMBER: _ClassVar[int]
    HAS_AUTO_ENABLED_FIELD_NUMBER: _ClassVar[int]
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    HAS_BACKEND_FIELD_NUMBER: _ClassVar[int]
    KOKORO_VOICE_FIELD_NUMBER: _ClassVar[int]
    HAS_KOKORO_VOICE_FIELD_NUMBER: _ClassVar[int]
    KOKORO_SPEED_FIELD_NUMBER: _ClassVar[int]
    HAS_KOKORO_SPEED_FIELD_NUMBER: _ClassVar[int]
    auto_enabled: bool
    has_auto_enabled: bool
    backend: str
    has_backend: bool
    kokoro_voice: str
    has_kokoro_voice: bool
    kokoro_speed: float
    has_kokoro_speed: bool
    def __init__(self, auto_enabled: _Optional[bool] = ..., has_auto_enabled: _Optional[bool] = ..., backend: _Optional[str] = ..., has_backend: _Optional[bool] = ..., kokoro_voice: _Optional[str] = ..., has_kokoro_voice: _Optional[bool] = ..., kokoro_speed: _Optional[float] = ..., has_kokoro_speed: _Optional[bool] = ...) -> None: ...

class UpdateConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: Config
    def __init__(self, config: _Optional[_Union[Config, _Mapping]] = ...) -> None: ...

class SummarizeConfig(_message.Message):
    __slots__ = ("enabled", "char_threshold", "level", "model", "timeout_seconds")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    CHAR_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    char_threshold: int
    level: str
    model: str
    timeout_seconds: int
    def __init__(self, enabled: _Optional[bool] = ..., char_threshold: _Optional[int] = ..., level: _Optional[str] = ..., model: _Optional[str] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class GetSummarizeConfigRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetSummarizeConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: SummarizeConfig
    def __init__(self, config: _Optional[_Union[SummarizeConfig, _Mapping]] = ...) -> None: ...

class UpdateSummarizeConfigRequest(_message.Message):
    __slots__ = ("enabled", "has_enabled", "char_threshold", "has_char_threshold", "level", "has_level", "model", "has_model", "timeout_seconds", "has_timeout_seconds")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    HAS_ENABLED_FIELD_NUMBER: _ClassVar[int]
    CHAR_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    HAS_CHAR_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    HAS_LEVEL_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    HAS_MODEL_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    HAS_TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    has_enabled: bool
    char_threshold: int
    has_char_threshold: bool
    level: str
    has_level: bool
    model: str
    has_model: bool
    timeout_seconds: int
    has_timeout_seconds: bool
    def __init__(self, enabled: _Optional[bool] = ..., has_enabled: _Optional[bool] = ..., char_threshold: _Optional[int] = ..., has_char_threshold: _Optional[bool] = ..., level: _Optional[str] = ..., has_level: _Optional[bool] = ..., model: _Optional[str] = ..., has_model: _Optional[bool] = ..., timeout_seconds: _Optional[int] = ..., has_timeout_seconds: _Optional[bool] = ...) -> None: ...

class UpdateSummarizeConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: SummarizeConfig
    def __init__(self, config: _Optional[_Union[SummarizeConfig, _Mapping]] = ...) -> None: ...

class AppendResult(_message.Message):
    __slots__ = ("appended", "code", "reason", "source", "session_id", "event_id", "sequence", "duplicate")
    APPENDED_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    DUPLICATE_FIELD_NUMBER: _ClassVar[int]
    appended: bool
    code: str
    reason: str
    source: str
    session_id: str
    event_id: str
    sequence: int
    duplicate: bool
    def __init__(self, appended: _Optional[bool] = ..., code: _Optional[str] = ..., reason: _Optional[str] = ..., source: _Optional[str] = ..., session_id: _Optional[str] = ..., event_id: _Optional[str] = ..., sequence: _Optional[int] = ..., duplicate: _Optional[bool] = ...) -> None: ...

class ClientAck(_message.Message):
    __slots__ = ("event_id", "source", "session_id", "stage", "backend", "message")
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    STAGE_FIELD_NUMBER: _ClassVar[int]
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    source: str
    session_id: str
    stage: str
    backend: str
    message: str
    def __init__(self, event_id: _Optional[str] = ..., source: _Optional[str] = ..., session_id: _Optional[str] = ..., stage: _Optional[str] = ..., backend: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class PlaybackEvent(_message.Message):
    __slots__ = ("source", "stage", "backend", "session_id", "message")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    STAGE_FIELD_NUMBER: _ClassVar[int]
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    source: str
    stage: str
    backend: str
    session_id: str
    message: str
    def __init__(self, source: _Optional[str] = ..., stage: _Optional[str] = ..., backend: _Optional[str] = ..., session_id: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class Status(_message.Message):
    __slots__ = ("config", "hook_registered", "hook_code", "hook_reason", "hook_settings_path", "last_routing", "last_routing_at", "last_hook_routing", "last_hook_routing_at", "last_tailer_routing", "last_tailer_routing_at", "last_ack", "last_ack_at", "last_hook_ack", "last_hook_ack_at", "last_tailer_ack", "last_tailer_ack_at", "last_playback_event", "last_playback_at", "kokoro_capability", "kokoro_capability_label")
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    HOOK_REGISTERED_FIELD_NUMBER: _ClassVar[int]
    HOOK_CODE_FIELD_NUMBER: _ClassVar[int]
    HOOK_REASON_FIELD_NUMBER: _ClassVar[int]
    HOOK_SETTINGS_PATH_FIELD_NUMBER: _ClassVar[int]
    LAST_ROUTING_FIELD_NUMBER: _ClassVar[int]
    LAST_ROUTING_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_HOOK_ROUTING_FIELD_NUMBER: _ClassVar[int]
    LAST_HOOK_ROUTING_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_TAILER_ROUTING_FIELD_NUMBER: _ClassVar[int]
    LAST_TAILER_ROUTING_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_ACK_FIELD_NUMBER: _ClassVar[int]
    LAST_ACK_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_HOOK_ACK_FIELD_NUMBER: _ClassVar[int]
    LAST_HOOK_ACK_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_TAILER_ACK_FIELD_NUMBER: _ClassVar[int]
    LAST_TAILER_ACK_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_PLAYBACK_EVENT_FIELD_NUMBER: _ClassVar[int]
    LAST_PLAYBACK_AT_FIELD_NUMBER: _ClassVar[int]
    KOKORO_CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    KOKORO_CAPABILITY_LABEL_FIELD_NUMBER: _ClassVar[int]
    config: Config
    hook_registered: bool
    hook_code: str
    hook_reason: str
    hook_settings_path: str
    last_routing: AppendResult
    last_routing_at: str
    last_hook_routing: AppendResult
    last_hook_routing_at: str
    last_tailer_routing: AppendResult
    last_tailer_routing_at: str
    last_ack: ClientAck
    last_ack_at: str
    last_hook_ack: ClientAck
    last_hook_ack_at: str
    last_tailer_ack: ClientAck
    last_tailer_ack_at: str
    last_playback_event: PlaybackEvent
    last_playback_at: str
    kokoro_capability: str
    kokoro_capability_label: str
    def __init__(self, config: _Optional[_Union[Config, _Mapping]] = ..., hook_registered: _Optional[bool] = ..., hook_code: _Optional[str] = ..., hook_reason: _Optional[str] = ..., hook_settings_path: _Optional[str] = ..., last_routing: _Optional[_Union[AppendResult, _Mapping]] = ..., last_routing_at: _Optional[str] = ..., last_hook_routing: _Optional[_Union[AppendResult, _Mapping]] = ..., last_hook_routing_at: _Optional[str] = ..., last_tailer_routing: _Optional[_Union[AppendResult, _Mapping]] = ..., last_tailer_routing_at: _Optional[str] = ..., last_ack: _Optional[_Union[ClientAck, _Mapping]] = ..., last_ack_at: _Optional[str] = ..., last_hook_ack: _Optional[_Union[ClientAck, _Mapping]] = ..., last_hook_ack_at: _Optional[str] = ..., last_tailer_ack: _Optional[_Union[ClientAck, _Mapping]] = ..., last_tailer_ack_at: _Optional[str] = ..., last_playback_event: _Optional[_Union[PlaybackEvent, _Mapping]] = ..., last_playback_at: _Optional[str] = ..., kokoro_capability: _Optional[str] = ..., kokoro_capability_label: _Optional[str] = ...) -> None: ...

class GetStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetStatusResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: Status
    def __init__(self, status: _Optional[_Union[Status, _Mapping]] = ...) -> None: ...

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

class SynthesizeRequest(_message.Message):
    __slots__ = ("input", "voice", "response_format", "speed", "event_id", "version")
    INPUT_FIELD_NUMBER: _ClassVar[int]
    VOICE_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_FORMAT_FIELD_NUMBER: _ClassVar[int]
    SPEED_FIELD_NUMBER: _ClassVar[int]
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    input: str
    voice: str
    response_format: str
    speed: float
    event_id: str
    version: str
    def __init__(self, input: _Optional[str] = ..., voice: _Optional[str] = ..., response_format: _Optional[str] = ..., speed: _Optional[float] = ..., event_id: _Optional[str] = ..., version: _Optional[str] = ...) -> None: ...

class SynthesizeResponse(_message.Message):
    __slots__ = ("audio", "content_type")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    content_type: str
    def __init__(self, audio: _Optional[bytes] = ..., content_type: _Optional[str] = ...) -> None: ...

class GetCacheRequest(_message.Message):
    __slots__ = ("event_id", "voice", "speed", "version")
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    VOICE_FIELD_NUMBER: _ClassVar[int]
    SPEED_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    voice: str
    speed: float
    version: str
    def __init__(self, event_id: _Optional[str] = ..., voice: _Optional[str] = ..., speed: _Optional[float] = ..., version: _Optional[str] = ...) -> None: ...

class GetCacheResponse(_message.Message):
    __slots__ = ("audio", "content_type")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    content_type: str
    def __init__(self, audio: _Optional[bytes] = ..., content_type: _Optional[str] = ...) -> None: ...

class Voice(_message.Message):
    __slots__ = ("id", "name")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ...) -> None: ...

class ListVoicesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListVoicesResponse(_message.Message):
    __slots__ = ("voices",)
    VOICES_FIELD_NUMBER: _ClassVar[int]
    voices: _containers.RepeatedCompositeFieldContainer[Voice]
    def __init__(self, voices: _Optional[_Iterable[_Union[Voice, _Mapping]]] = ...) -> None: ...
