import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import field_mask_pb2 as _field_mask_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from web_console.v1.audio_common import audio_common_pb2 as _audio_common_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class StreamConfig(_message.Message):
    __slots__ = ("flush_interval_ms", "min_delta_bytes", "overlap_bytes", "persistent_mode", "wake_word_enabled", "wake_word_threshold", "segment_silence_ms", "streaming_mode", "strategy_preference", "vad_silence_ms", "overlap_window_ms", "overlap_commit_runs")
    FLUSH_INTERVAL_MS_FIELD_NUMBER: _ClassVar[int]
    MIN_DELTA_BYTES_FIELD_NUMBER: _ClassVar[int]
    OVERLAP_BYTES_FIELD_NUMBER: _ClassVar[int]
    PERSISTENT_MODE_FIELD_NUMBER: _ClassVar[int]
    WAKE_WORD_ENABLED_FIELD_NUMBER: _ClassVar[int]
    WAKE_WORD_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    SEGMENT_SILENCE_MS_FIELD_NUMBER: _ClassVar[int]
    STREAMING_MODE_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_PREFERENCE_FIELD_NUMBER: _ClassVar[int]
    VAD_SILENCE_MS_FIELD_NUMBER: _ClassVar[int]
    OVERLAP_WINDOW_MS_FIELD_NUMBER: _ClassVar[int]
    OVERLAP_COMMIT_RUNS_FIELD_NUMBER: _ClassVar[int]
    flush_interval_ms: int
    min_delta_bytes: int
    overlap_bytes: int
    persistent_mode: bool
    wake_word_enabled: bool
    wake_word_threshold: float
    segment_silence_ms: int
    streaming_mode: _audio_common_pb2.StreamingMode
    strategy_preference: _audio_common_pb2.StrategyPreference
    vad_silence_ms: int
    overlap_window_ms: int
    overlap_commit_runs: int
    def __init__(self, flush_interval_ms: _Optional[int] = ..., min_delta_bytes: _Optional[int] = ..., overlap_bytes: _Optional[int] = ..., persistent_mode: _Optional[bool] = ..., wake_word_enabled: _Optional[bool] = ..., wake_word_threshold: _Optional[float] = ..., segment_silence_ms: _Optional[int] = ..., streaming_mode: _Optional[_Union[_audio_common_pb2.StreamingMode, str]] = ..., strategy_preference: _Optional[_Union[_audio_common_pb2.StrategyPreference, str]] = ..., vad_silence_ms: _Optional[int] = ..., overlap_window_ms: _Optional[int] = ..., overlap_commit_runs: _Optional[int] = ...) -> None: ...

class GetStreamConfigRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetStreamConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: StreamConfig
    def __init__(self, config: _Optional[_Union[StreamConfig, _Mapping]] = ...) -> None: ...

class UpdateStreamConfigRequest(_message.Message):
    __slots__ = ("update_mask", "config")
    UPDATE_MASK_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    update_mask: _field_mask_pb2.FieldMask
    config: StreamConfig
    def __init__(self, update_mask: _Optional[_Union[_field_mask_pb2.FieldMask, _Mapping]] = ..., config: _Optional[_Union[StreamConfig, _Mapping]] = ...) -> None: ...

class UpdateStreamConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: StreamConfig
    def __init__(self, config: _Optional[_Union[StreamConfig, _Mapping]] = ...) -> None: ...

class WakeWordSample(_message.Message):
    __slots__ = ("audio", "format", "sample_rate_hz")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_RATE_HZ_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    format: _audio_common_pb2.AudioFormat
    sample_rate_hz: int
    def __init__(self, audio: _Optional[bytes] = ..., format: _Optional[_Union[_audio_common_pb2.AudioFormat, str]] = ..., sample_rate_hz: _Optional[int] = ...) -> None: ...

class WakeWordTemplate(_message.Message):
    __slots__ = ("label", "threshold", "samples", "updated_at")
    LABEL_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    SAMPLES_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    label: str
    threshold: float
    samples: _containers.RepeatedCompositeFieldContainer[WakeWordSample]
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, label: _Optional[str] = ..., threshold: _Optional[float] = ..., samples: _Optional[_Iterable[_Union[WakeWordSample, _Mapping]]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class WakeWordConfig(_message.Message):
    __slots__ = ("configured", "template")
    CONFIGURED_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    configured: bool
    template: WakeWordTemplate
    def __init__(self, configured: _Optional[bool] = ..., template: _Optional[_Union[WakeWordTemplate, _Mapping]] = ...) -> None: ...

class GetWakeWordConfigRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetWakeWordConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: WakeWordConfig
    def __init__(self, config: _Optional[_Union[WakeWordConfig, _Mapping]] = ...) -> None: ...

class UpdateWakeWordTemplateRequest(_message.Message):
    __slots__ = ("template",)
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    template: WakeWordTemplate
    def __init__(self, template: _Optional[_Union[WakeWordTemplate, _Mapping]] = ...) -> None: ...

class UpdateWakeWordTemplateResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: WakeWordConfig
    def __init__(self, config: _Optional[_Union[WakeWordConfig, _Mapping]] = ...) -> None: ...

class DeleteWakeWordTemplateRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class DeleteWakeWordTemplateResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: WakeWordConfig
    def __init__(self, config: _Optional[_Union[WakeWordConfig, _Mapping]] = ...) -> None: ...

class SpeakerConfig(_message.Message):
    __slots__ = ("enabled", "profile_ids", "threshold", "mode", "reject_behavior", "fallback_without_verification", "extraction_enabled")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    PROFILE_IDS_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    REJECT_BEHAVIOR_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_WITHOUT_VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    EXTRACTION_ENABLED_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    profile_ids: _containers.RepeatedScalarFieldContainer[str]
    threshold: float
    mode: _audio_common_pb2.SpeakerMode
    reject_behavior: _audio_common_pb2.RejectBehavior
    fallback_without_verification: bool
    extraction_enabled: bool
    def __init__(self, enabled: _Optional[bool] = ..., profile_ids: _Optional[_Iterable[str]] = ..., threshold: _Optional[float] = ..., mode: _Optional[_Union[_audio_common_pb2.SpeakerMode, str]] = ..., reject_behavior: _Optional[_Union[_audio_common_pb2.RejectBehavior, str]] = ..., fallback_without_verification: _Optional[bool] = ..., extraction_enabled: _Optional[bool] = ...) -> None: ...

class GetSpeakerConfigRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetSpeakerConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: SpeakerConfig
    def __init__(self, config: _Optional[_Union[SpeakerConfig, _Mapping]] = ...) -> None: ...

class UpdateSpeakerConfigRequest(_message.Message):
    __slots__ = ("update_mask", "config")
    UPDATE_MASK_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    update_mask: _field_mask_pb2.FieldMask
    config: SpeakerConfig
    def __init__(self, update_mask: _Optional[_Union[_field_mask_pb2.FieldMask, _Mapping]] = ..., config: _Optional[_Union[SpeakerConfig, _Mapping]] = ...) -> None: ...

class UpdateSpeakerConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: SpeakerConfig
    def __init__(self, config: _Optional[_Union[SpeakerConfig, _Mapping]] = ...) -> None: ...

class SpeakerProfile(_message.Message):
    __slots__ = ("id", "display_name", "created_at", "updated_at", "model_name", "embedding_dim", "sample_rate", "enrollment_audio_seconds", "notes")
    ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    MODEL_NAME_FIELD_NUMBER: _ClassVar[int]
    EMBEDDING_DIM_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_RATE_FIELD_NUMBER: _ClassVar[int]
    ENROLLMENT_AUDIO_SECONDS_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    id: str
    display_name: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    model_name: str
    embedding_dim: int
    sample_rate: int
    enrollment_audio_seconds: float
    notes: str
    def __init__(self, id: _Optional[str] = ..., display_name: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., model_name: _Optional[str] = ..., embedding_dim: _Optional[int] = ..., sample_rate: _Optional[int] = ..., enrollment_audio_seconds: _Optional[float] = ..., notes: _Optional[str] = ...) -> None: ...

class SpeakerResourceInfo(_message.Message):
    __slots__ = ("backend", "model", "device", "sample_rate", "version", "embedding_dim")
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    DEVICE_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_RATE_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    EMBEDDING_DIM_FIELD_NUMBER: _ClassVar[int]
    backend: str
    model: str
    device: str
    sample_rate: int
    version: str
    embedding_dim: int
    def __init__(self, backend: _Optional[str] = ..., model: _Optional[str] = ..., device: _Optional[str] = ..., sample_rate: _Optional[int] = ..., version: _Optional[str] = ..., embedding_dim: _Optional[int] = ...) -> None: ...

class SpeakerStatus(_message.Message):
    __slots__ = ("config", "capability", "capability_label", "resource_ready", "profile_configured", "profile_exists", "profile_count", "profiles", "info", "checked_at")
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_LABEL_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_READY_FIELD_NUMBER: _ClassVar[int]
    PROFILE_CONFIGURED_FIELD_NUMBER: _ClassVar[int]
    PROFILE_EXISTS_FIELD_NUMBER: _ClassVar[int]
    PROFILE_COUNT_FIELD_NUMBER: _ClassVar[int]
    PROFILES_FIELD_NUMBER: _ClassVar[int]
    INFO_FIELD_NUMBER: _ClassVar[int]
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    config: SpeakerConfig
    capability: _audio_common_pb2.SpeakerCapability
    capability_label: str
    resource_ready: bool
    profile_configured: bool
    profile_exists: bool
    profile_count: int
    profiles: _containers.RepeatedCompositeFieldContainer[SpeakerProfile]
    info: SpeakerResourceInfo
    checked_at: _timestamp_pb2.Timestamp
    def __init__(self, config: _Optional[_Union[SpeakerConfig, _Mapping]] = ..., capability: _Optional[_Union[_audio_common_pb2.SpeakerCapability, str]] = ..., capability_label: _Optional[str] = ..., resource_ready: _Optional[bool] = ..., profile_configured: _Optional[bool] = ..., profile_exists: _Optional[bool] = ..., profile_count: _Optional[int] = ..., profiles: _Optional[_Iterable[_Union[SpeakerProfile, _Mapping]]] = ..., info: _Optional[_Union[SpeakerResourceInfo, _Mapping]] = ..., checked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetSpeakerStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetSpeakerStatusResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: SpeakerStatus
    def __init__(self, status: _Optional[_Union[SpeakerStatus, _Mapping]] = ...) -> None: ...

class ListSpeakerProfilesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListSpeakerProfilesResponse(_message.Message):
    __slots__ = ("profiles", "count")
    PROFILES_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    profiles: _containers.RepeatedCompositeFieldContainer[SpeakerProfile]
    count: int
    def __init__(self, profiles: _Optional[_Iterable[_Union[SpeakerProfile, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class SpeakerEnrollment(_message.Message):
    __slots__ = ("profile_id", "display_name", "embedding_dim", "sample_rate", "enrollment_audio_seconds", "model_name", "created_at")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    EMBEDDING_DIM_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_RATE_FIELD_NUMBER: _ClassVar[int]
    ENROLLMENT_AUDIO_SECONDS_FIELD_NUMBER: _ClassVar[int]
    MODEL_NAME_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    display_name: str
    embedding_dim: int
    sample_rate: int
    enrollment_audio_seconds: float
    model_name: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, profile_id: _Optional[str] = ..., display_name: _Optional[str] = ..., embedding_dim: _Optional[int] = ..., sample_rate: _Optional[int] = ..., enrollment_audio_seconds: _Optional[float] = ..., model_name: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class EnrollSpeakerProfileRequest(_message.Message):
    __slots__ = ("audio", "format", "profile_id", "display_name", "notes", "add_to_active", "enable")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    ADD_TO_ACTIVE_FIELD_NUMBER: _ClassVar[int]
    ENABLE_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    format: _audio_common_pb2.AudioFormat
    profile_id: str
    display_name: str
    notes: str
    add_to_active: bool
    enable: bool
    def __init__(self, audio: _Optional[bytes] = ..., format: _Optional[_Union[_audio_common_pb2.AudioFormat, str]] = ..., profile_id: _Optional[str] = ..., display_name: _Optional[str] = ..., notes: _Optional[str] = ..., add_to_active: _Optional[bool] = ..., enable: _Optional[bool] = ...) -> None: ...

class EnrollSpeakerProfileResponse(_message.Message):
    __slots__ = ("enrollment", "config")
    ENROLLMENT_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    enrollment: SpeakerEnrollment
    config: SpeakerConfig
    def __init__(self, enrollment: _Optional[_Union[SpeakerEnrollment, _Mapping]] = ..., config: _Optional[_Union[SpeakerConfig, _Mapping]] = ...) -> None: ...

class ClearSpeakerProfileBindingRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ClearSpeakerProfileBindingResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: SpeakerConfig
    def __init__(self, config: _Optional[_Union[SpeakerConfig, _Mapping]] = ...) -> None: ...

class UnbindSpeakerProfileRequest(_message.Message):
    __slots__ = ("profile_id",)
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    def __init__(self, profile_id: _Optional[str] = ...) -> None: ...

class UnbindSpeakerProfileResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: SpeakerConfig
    def __init__(self, config: _Optional[_Union[SpeakerConfig, _Mapping]] = ...) -> None: ...

class DeleteSpeakerProfileRequest(_message.Message):
    __slots__ = ("profile_id",)
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    def __init__(self, profile_id: _Optional[str] = ...) -> None: ...

class DeleteSpeakerProfileResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: SpeakerConfig
    def __init__(self, config: _Optional[_Union[SpeakerConfig, _Mapping]] = ...) -> None: ...

class TTSConfig(_message.Message):
    __slots__ = ("auto_enabled", "default_voice", "default_speed", "default_response_format")
    AUTO_ENABLED_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_VOICE_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_SPEED_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_RESPONSE_FORMAT_FIELD_NUMBER: _ClassVar[int]
    auto_enabled: bool
    default_voice: str
    default_speed: float
    default_response_format: _audio_common_pb2.ResponseFormat
    def __init__(self, auto_enabled: _Optional[bool] = ..., default_voice: _Optional[str] = ..., default_speed: _Optional[float] = ..., default_response_format: _Optional[_Union[_audio_common_pb2.ResponseFormat, str]] = ...) -> None: ...

class GetTTSConfigRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetTTSConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: TTSConfig
    def __init__(self, config: _Optional[_Union[TTSConfig, _Mapping]] = ...) -> None: ...

class UpdateTTSConfigRequest(_message.Message):
    __slots__ = ("update_mask", "config")
    UPDATE_MASK_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    update_mask: _field_mask_pb2.FieldMask
    config: TTSConfig
    def __init__(self, update_mask: _Optional[_Union[_field_mask_pb2.FieldMask, _Mapping]] = ..., config: _Optional[_Union[TTSConfig, _Mapping]] = ...) -> None: ...

class UpdateTTSConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: TTSConfig
    def __init__(self, config: _Optional[_Union[TTSConfig, _Mapping]] = ...) -> None: ...

class SummarizeConfig(_message.Message):
    __slots__ = ("enabled", "char_threshold", "level", "model", "timeout_seconds")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    CHAR_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    char_threshold: int
    level: _audio_common_pb2.SummarizeLevel
    model: str
    timeout_seconds: int
    def __init__(self, enabled: _Optional[bool] = ..., char_threshold: _Optional[int] = ..., level: _Optional[_Union[_audio_common_pb2.SummarizeLevel, str]] = ..., model: _Optional[str] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class GetSummarizeConfigRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetSummarizeConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: SummarizeConfig
    def __init__(self, config: _Optional[_Union[SummarizeConfig, _Mapping]] = ...) -> None: ...

class UpdateSummarizeConfigRequest(_message.Message):
    __slots__ = ("update_mask", "config")
    UPDATE_MASK_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    update_mask: _field_mask_pb2.FieldMask
    config: SummarizeConfig
    def __init__(self, update_mask: _Optional[_Union[_field_mask_pb2.FieldMask, _Mapping]] = ..., config: _Optional[_Union[SummarizeConfig, _Mapping]] = ...) -> None: ...

class UpdateSummarizeConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: SummarizeConfig
    def __init__(self, config: _Optional[_Union[SummarizeConfig, _Mapping]] = ...) -> None: ...

class ListSummarizeModelsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SummarizeModel(_message.Message):
    __slots__ = ("id", "display_name", "installed", "recommended", "default_eligible", "reasoning", "status_label", "pull_command", "size_bytes", "parameter_size", "source_url", "notes")
    ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    INSTALLED_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    REASONING_FIELD_NUMBER: _ClassVar[int]
    STATUS_LABEL_FIELD_NUMBER: _ClassVar[int]
    PULL_COMMAND_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    PARAMETER_SIZE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_URL_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    id: str
    display_name: str
    installed: bool
    recommended: bool
    default_eligible: bool
    reasoning: bool
    status_label: str
    pull_command: str
    size_bytes: int
    parameter_size: str
    source_url: str
    notes: str
    def __init__(self, id: _Optional[str] = ..., display_name: _Optional[str] = ..., installed: _Optional[bool] = ..., recommended: _Optional[bool] = ..., default_eligible: _Optional[bool] = ..., reasoning: _Optional[bool] = ..., status_label: _Optional[str] = ..., pull_command: _Optional[str] = ..., size_bytes: _Optional[int] = ..., parameter_size: _Optional[str] = ..., source_url: _Optional[str] = ..., notes: _Optional[str] = ...) -> None: ...

class ListSummarizeModelsResponse(_message.Message):
    __slots__ = ("models",)
    MODELS_FIELD_NUMBER: _ClassVar[int]
    models: _containers.RepeatedCompositeFieldContainer[SummarizeModel]
    def __init__(self, models: _Optional[_Iterable[_Union[SummarizeModel, _Mapping]]] = ...) -> None: ...
