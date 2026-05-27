import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import field_mask_pb2 as _field_mask_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from audio_tools.v1.common import common_pb2 as _common_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SpeakerMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SPEAKER_MODE_UNSPECIFIED: _ClassVar[SpeakerMode]
    SPEAKER_MODE_OFF: _ClassVar[SpeakerMode]
    SPEAKER_MODE_FILTER: _ClassVar[SpeakerMode]
    SPEAKER_MODE_ADVISORY: _ClassVar[SpeakerMode]

class RejectBehavior(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    REJECT_BEHAVIOR_UNSPECIFIED: _ClassVar[RejectBehavior]
    REJECT_BEHAVIOR_DROP: _ClassVar[RejectBehavior]
    REJECT_BEHAVIOR_SHOW_MUTED: _ClassVar[RejectBehavior]

class StreamingMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    STREAMING_MODE_UNSPECIFIED: _ClassVar[StreamingMode]
    STREAMING_MODE_AUTO: _ClassVar[StreamingMode]
    STREAMING_MODE_OFF: _ClassVar[StreamingMode]

class StrategyPreference(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    STRATEGY_PREFERENCE_UNSPECIFIED: _ClassVar[StrategyPreference]
    STRATEGY_PREFERENCE_AUTO: _ClassVar[StrategyPreference]
    STRATEGY_PREFERENCE_VAD: _ClassVar[StrategyPreference]
    STRATEGY_PREFERENCE_OVERLAP: _ClassVar[StrategyPreference]
    STRATEGY_PREFERENCE_PASSTHROUGH: _ClassVar[StrategyPreference]
SPEAKER_MODE_UNSPECIFIED: SpeakerMode
SPEAKER_MODE_OFF: SpeakerMode
SPEAKER_MODE_FILTER: SpeakerMode
SPEAKER_MODE_ADVISORY: SpeakerMode
REJECT_BEHAVIOR_UNSPECIFIED: RejectBehavior
REJECT_BEHAVIOR_DROP: RejectBehavior
REJECT_BEHAVIOR_SHOW_MUTED: RejectBehavior
STREAMING_MODE_UNSPECIFIED: StreamingMode
STREAMING_MODE_AUTO: StreamingMode
STREAMING_MODE_OFF: StreamingMode
STRATEGY_PREFERENCE_UNSPECIFIED: StrategyPreference
STRATEGY_PREFERENCE_AUTO: StrategyPreference
STRATEGY_PREFERENCE_VAD: StrategyPreference
STRATEGY_PREFERENCE_OVERLAP: StrategyPreference
STRATEGY_PREFERENCE_PASSTHROUGH: StrategyPreference

class TranscribeRequest(_message.Message):
    __slots__ = ("audio", "format", "language", "skip_speaker_verification", "initial_prompt")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    SKIP_SPEAKER_VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    INITIAL_PROMPT_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    format: _common_pb2.AudioFormat
    language: str
    skip_speaker_verification: bool
    initial_prompt: str
    def __init__(self, audio: _Optional[bytes] = ..., format: _Optional[_Union[_common_pb2.AudioFormat, str]] = ..., language: _Optional[str] = ..., skip_speaker_verification: _Optional[bool] = ..., initial_prompt: _Optional[str] = ...) -> None: ...

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
    provider_tier: _common_pb2.ProviderTier
    provider_id: str
    model_id: str
    latency_ms: float
    def __init__(self, text: _Optional[str] = ..., detected_language: _Optional[str] = ..., duration_seconds: _Optional[float] = ..., provider_tier: _Optional[_Union[_common_pb2.ProviderTier, str]] = ..., provider_id: _Optional[str] = ..., model_id: _Optional[str] = ..., latency_ms: _Optional[float] = ...) -> None: ...

class StreamConfig(_message.Message):
    __slots__ = ("flush_interval_ms", "min_delta_bytes", "overlap_bytes", "persistent_mode", "wake_word_enabled", "wake_word_threshold", "segment_silence_ms", "streaming_mode", "strategy_preference", "vad_silence_ms", "overlap_window_ms", "overlap_commit_runs", "hallucination_filter_enabled", "vad_filter_enabled", "no_speech_threshold", "logprob_threshold", "engine_id")
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
    HALLUCINATION_FILTER_ENABLED_FIELD_NUMBER: _ClassVar[int]
    VAD_FILTER_ENABLED_FIELD_NUMBER: _ClassVar[int]
    NO_SPEECH_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    LOGPROB_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    ENGINE_ID_FIELD_NUMBER: _ClassVar[int]
    flush_interval_ms: int
    min_delta_bytes: int
    overlap_bytes: int
    persistent_mode: bool
    wake_word_enabled: bool
    wake_word_threshold: float
    segment_silence_ms: int
    streaming_mode: StreamingMode
    strategy_preference: StrategyPreference
    vad_silence_ms: int
    overlap_window_ms: int
    overlap_commit_runs: int
    hallucination_filter_enabled: bool
    vad_filter_enabled: bool
    no_speech_threshold: float
    logprob_threshold: float
    engine_id: str
    def __init__(self, flush_interval_ms: _Optional[int] = ..., min_delta_bytes: _Optional[int] = ..., overlap_bytes: _Optional[int] = ..., persistent_mode: _Optional[bool] = ..., wake_word_enabled: _Optional[bool] = ..., wake_word_threshold: _Optional[float] = ..., segment_silence_ms: _Optional[int] = ..., streaming_mode: _Optional[_Union[StreamingMode, str]] = ..., strategy_preference: _Optional[_Union[StrategyPreference, str]] = ..., vad_silence_ms: _Optional[int] = ..., overlap_window_ms: _Optional[int] = ..., overlap_commit_runs: _Optional[int] = ..., hallucination_filter_enabled: _Optional[bool] = ..., vad_filter_enabled: _Optional[bool] = ..., no_speech_threshold: _Optional[float] = ..., logprob_threshold: _Optional[float] = ..., engine_id: _Optional[str] = ...) -> None: ...

class GetStreamConfigRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetStreamConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: StreamConfig
    def __init__(self, config: _Optional[_Union[StreamConfig, _Mapping]] = ...) -> None: ...

class GetSupportedFormatsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetSupportedFormatsResponse(_message.Message):
    __slots__ = ("accepted_formats", "ffmpeg_available", "canonical_sample_rate_hz", "canonical_channels")
    ACCEPTED_FORMATS_FIELD_NUMBER: _ClassVar[int]
    FFMPEG_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    CANONICAL_SAMPLE_RATE_HZ_FIELD_NUMBER: _ClassVar[int]
    CANONICAL_CHANNELS_FIELD_NUMBER: _ClassVar[int]
    accepted_formats: _containers.RepeatedScalarFieldContainer[_common_pb2.AudioFormat]
    ffmpeg_available: bool
    canonical_sample_rate_hz: int
    canonical_channels: int
    def __init__(self, accepted_formats: _Optional[_Iterable[_Union[_common_pb2.AudioFormat, str]]] = ..., ffmpeg_available: _Optional[bool] = ..., canonical_sample_rate_hz: _Optional[int] = ..., canonical_channels: _Optional[int] = ...) -> None: ...

class ListEnginesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class EngineInfo(_message.Message):
    __slots__ = ("id", "display_name", "kind", "available", "native_streaming", "is_active")
    ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    NATIVE_STREAMING_FIELD_NUMBER: _ClassVar[int]
    IS_ACTIVE_FIELD_NUMBER: _ClassVar[int]
    id: str
    display_name: str
    kind: str
    available: bool
    native_streaming: bool
    is_active: bool
    def __init__(self, id: _Optional[str] = ..., display_name: _Optional[str] = ..., kind: _Optional[str] = ..., available: _Optional[bool] = ..., native_streaming: _Optional[bool] = ..., is_active: _Optional[bool] = ...) -> None: ...

class ListEnginesResponse(_message.Message):
    __slots__ = ("engines",)
    ENGINES_FIELD_NUMBER: _ClassVar[int]
    engines: _containers.RepeatedCompositeFieldContainer[EngineInfo]
    def __init__(self, engines: _Optional[_Iterable[_Union[EngineInfo, _Mapping]]] = ...) -> None: ...

class GetEngineSwitchImpactRequest(_message.Message):
    __slots__ = ("from_engine_id",)
    FROM_ENGINE_ID_FIELD_NUMBER: _ClassVar[int]
    from_engine_id: str
    def __init__(self, from_engine_id: _Optional[str] = ...) -> None: ...

class ScenarioResourceConsumer(_message.Message):
    __slots__ = ("scenario", "display_name", "required")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    display_name: str
    required: bool
    def __init__(self, scenario: _Optional[str] = ..., display_name: _Optional[str] = ..., required: _Optional[bool] = ...) -> None: ...

class GetEngineSwitchImpactResponse(_message.Message):
    __slots__ = ("resource", "consumers", "safe_to_stop", "stop_command", "consumers_known")
    RESOURCE_FIELD_NUMBER: _ClassVar[int]
    CONSUMERS_FIELD_NUMBER: _ClassVar[int]
    SAFE_TO_STOP_FIELD_NUMBER: _ClassVar[int]
    STOP_COMMAND_FIELD_NUMBER: _ClassVar[int]
    CONSUMERS_KNOWN_FIELD_NUMBER: _ClassVar[int]
    resource: str
    consumers: _containers.RepeatedCompositeFieldContainer[ScenarioResourceConsumer]
    safe_to_stop: bool
    stop_command: str
    consumers_known: bool
    def __init__(self, resource: _Optional[str] = ..., consumers: _Optional[_Iterable[_Union[ScenarioResourceConsumer, _Mapping]]] = ..., safe_to_stop: _Optional[bool] = ..., stop_command: _Optional[str] = ..., consumers_known: _Optional[bool] = ...) -> None: ...

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
    format: _common_pb2.AudioFormat
    sample_rate_hz: int
    def __init__(self, audio: _Optional[bytes] = ..., format: _Optional[_Union[_common_pb2.AudioFormat, str]] = ..., sample_rate_hz: _Optional[int] = ...) -> None: ...

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
    mode: SpeakerMode
    reject_behavior: RejectBehavior
    fallback_without_verification: bool
    extraction_enabled: bool
    def __init__(self, enabled: _Optional[bool] = ..., profile_ids: _Optional[_Iterable[str]] = ..., threshold: _Optional[float] = ..., mode: _Optional[_Union[SpeakerMode, str]] = ..., reject_behavior: _Optional[_Union[RejectBehavior, str]] = ..., fallback_without_verification: _Optional[bool] = ..., extraction_enabled: _Optional[bool] = ...) -> None: ...

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
    capability: str
    capability_label: str
    resource_ready: bool
    profile_configured: bool
    profile_exists: bool
    profile_count: int
    profiles: _containers.RepeatedCompositeFieldContainer[SpeakerProfile]
    info: SpeakerResourceInfo
    checked_at: _timestamp_pb2.Timestamp
    def __init__(self, config: _Optional[_Union[SpeakerConfig, _Mapping]] = ..., capability: _Optional[str] = ..., capability_label: _Optional[str] = ..., resource_ready: _Optional[bool] = ..., profile_configured: _Optional[bool] = ..., profile_exists: _Optional[bool] = ..., profile_count: _Optional[int] = ..., profiles: _Optional[_Iterable[_Union[SpeakerProfile, _Mapping]]] = ..., info: _Optional[_Union[SpeakerResourceInfo, _Mapping]] = ..., checked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

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
    format: _common_pb2.AudioFormat
    profile_id: str
    display_name: str
    notes: str
    add_to_active: bool
    enable: bool
    def __init__(self, audio: _Optional[bytes] = ..., format: _Optional[_Union[_common_pb2.AudioFormat, str]] = ..., profile_id: _Optional[str] = ..., display_name: _Optional[str] = ..., notes: _Optional[str] = ..., add_to_active: _Optional[bool] = ..., enable: _Optional[bool] = ...) -> None: ...

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

class TranscribeStreamRequest(_message.Message):
    __slots__ = ("start", "audio_chunk", "end")
    START_FIELD_NUMBER: _ClassVar[int]
    AUDIO_CHUNK_FIELD_NUMBER: _ClassVar[int]
    END_FIELD_NUMBER: _ClassVar[int]
    start: StreamStart
    audio_chunk: bytes
    end: StreamEnd
    def __init__(self, start: _Optional[_Union[StreamStart, _Mapping]] = ..., audio_chunk: _Optional[bytes] = ..., end: _Optional[_Union[StreamEnd, _Mapping]] = ...) -> None: ...

class StreamStart(_message.Message):
    __slots__ = ("config", "language", "initial_prompt", "skip_speaker_verification", "input_format", "input_sample_rate_hz")
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    INITIAL_PROMPT_FIELD_NUMBER: _ClassVar[int]
    SKIP_SPEAKER_VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    INPUT_FORMAT_FIELD_NUMBER: _ClassVar[int]
    INPUT_SAMPLE_RATE_HZ_FIELD_NUMBER: _ClassVar[int]
    config: StreamConfig
    language: str
    initial_prompt: str
    skip_speaker_verification: bool
    input_format: _common_pb2.AudioFormat
    input_sample_rate_hz: int
    def __init__(self, config: _Optional[_Union[StreamConfig, _Mapping]] = ..., language: _Optional[str] = ..., initial_prompt: _Optional[str] = ..., skip_speaker_verification: _Optional[bool] = ..., input_format: _Optional[_Union[_common_pb2.AudioFormat, str]] = ..., input_sample_rate_hz: _Optional[int] = ...) -> None: ...

class StreamEnd(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class TranscribeStreamEvent(_message.Message):
    __slots__ = ("segment", "partial", "wake_word", "speaker_rejection", "error", "done", "vad_state")
    SEGMENT_FIELD_NUMBER: _ClassVar[int]
    PARTIAL_FIELD_NUMBER: _ClassVar[int]
    WAKE_WORD_FIELD_NUMBER: _ClassVar[int]
    SPEAKER_REJECTION_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    DONE_FIELD_NUMBER: _ClassVar[int]
    VAD_STATE_FIELD_NUMBER: _ClassVar[int]
    segment: StreamSegment
    partial: StreamPartial
    wake_word: StreamWakeWord
    speaker_rejection: StreamSpeakerRejection
    error: StreamError
    done: StreamDone
    vad_state: StreamVadState
    def __init__(self, segment: _Optional[_Union[StreamSegment, _Mapping]] = ..., partial: _Optional[_Union[StreamPartial, _Mapping]] = ..., wake_word: _Optional[_Union[StreamWakeWord, _Mapping]] = ..., speaker_rejection: _Optional[_Union[StreamSpeakerRejection, _Mapping]] = ..., error: _Optional[_Union[StreamError, _Mapping]] = ..., done: _Optional[_Union[StreamDone, _Mapping]] = ..., vad_state: _Optional[_Union[StreamVadState, _Mapping]] = ...) -> None: ...

class StreamSegment(_message.Message):
    __slots__ = ("text", "start_ms", "end_ms", "detected_language", "provider_tier", "model_id", "latency_ms")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    START_MS_FIELD_NUMBER: _ClassVar[int]
    END_MS_FIELD_NUMBER: _ClassVar[int]
    DETECTED_LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_TIER_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    text: str
    start_ms: int
    end_ms: int
    detected_language: str
    provider_tier: _common_pb2.ProviderTier
    model_id: str
    latency_ms: float
    def __init__(self, text: _Optional[str] = ..., start_ms: _Optional[int] = ..., end_ms: _Optional[int] = ..., detected_language: _Optional[str] = ..., provider_tier: _Optional[_Union[_common_pb2.ProviderTier, str]] = ..., model_id: _Optional[str] = ..., latency_ms: _Optional[float] = ...) -> None: ...

class StreamPartial(_message.Message):
    __slots__ = ("text",)
    TEXT_FIELD_NUMBER: _ClassVar[int]
    text: str
    def __init__(self, text: _Optional[str] = ...) -> None: ...

class StreamWakeWord(_message.Message):
    __slots__ = ("score", "sample_id")
    SCORE_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_ID_FIELD_NUMBER: _ClassVar[int]
    score: float
    sample_id: str
    def __init__(self, score: _Optional[float] = ..., sample_id: _Optional[str] = ...) -> None: ...

class StreamSpeakerRejection(_message.Message):
    __slots__ = ("reason", "fallback_used")
    REASON_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_USED_FIELD_NUMBER: _ClassVar[int]
    reason: str
    fallback_used: bool
    def __init__(self, reason: _Optional[str] = ..., fallback_used: _Optional[bool] = ...) -> None: ...

class StreamError(_message.Message):
    __slots__ = ("code", "message")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class StreamVadState(_message.Message):
    __slots__ = ("voiced", "silence_elapsed_ms", "silence_timeout_ms", "tick_seq")
    VOICED_FIELD_NUMBER: _ClassVar[int]
    SILENCE_ELAPSED_MS_FIELD_NUMBER: _ClassVar[int]
    SILENCE_TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    TICK_SEQ_FIELD_NUMBER: _ClassVar[int]
    voiced: bool
    silence_elapsed_ms: int
    silence_timeout_ms: int
    tick_seq: int
    def __init__(self, voiced: _Optional[bool] = ..., silence_elapsed_ms: _Optional[int] = ..., silence_timeout_ms: _Optional[int] = ..., tick_seq: _Optional[int] = ...) -> None: ...

class StreamDone(_message.Message):
    __slots__ = ("final_text", "provider_tier", "provider_id", "model_id", "latency_ms", "fell_back_to_unary")
    FINAL_TEXT_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_TIER_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    FELL_BACK_TO_UNARY_FIELD_NUMBER: _ClassVar[int]
    final_text: str
    provider_tier: _common_pb2.ProviderTier
    provider_id: str
    model_id: str
    latency_ms: float
    fell_back_to_unary: bool
    def __init__(self, final_text: _Optional[str] = ..., provider_tier: _Optional[_Union[_common_pb2.ProviderTier, str]] = ..., provider_id: _Optional[str] = ..., model_id: _Optional[str] = ..., latency_ms: _Optional[float] = ..., fell_back_to_unary: _Optional[bool] = ...) -> None: ...
