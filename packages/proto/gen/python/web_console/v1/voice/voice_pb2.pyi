from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TranscribeRequest(_message.Message):
    __slots__ = ("audio", "content_type", "language", "skip_speaker_verification")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    SKIP_SPEAKER_VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    content_type: str
    language: str
    skip_speaker_verification: bool
    def __init__(self, audio: _Optional[bytes] = ..., content_type: _Optional[str] = ..., language: _Optional[str] = ..., skip_speaker_verification: _Optional[bool] = ...) -> None: ...

class TranscribeResponse(_message.Message):
    __slots__ = ("text",)
    TEXT_FIELD_NUMBER: _ClassVar[int]
    text: str
    def __init__(self, text: _Optional[str] = ...) -> None: ...

class StreamConfig(_message.Message):
    __slots__ = ("flush_interval_ms", "min_delta_bytes", "overlap_bytes", "persistent_mode", "wake_word_enabled", "wake_word_threshold", "segment_silence_ms")
    FLUSH_INTERVAL_MS_FIELD_NUMBER: _ClassVar[int]
    MIN_DELTA_BYTES_FIELD_NUMBER: _ClassVar[int]
    OVERLAP_BYTES_FIELD_NUMBER: _ClassVar[int]
    PERSISTENT_MODE_FIELD_NUMBER: _ClassVar[int]
    WAKE_WORD_ENABLED_FIELD_NUMBER: _ClassVar[int]
    WAKE_WORD_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    SEGMENT_SILENCE_MS_FIELD_NUMBER: _ClassVar[int]
    flush_interval_ms: int
    min_delta_bytes: int
    overlap_bytes: int
    persistent_mode: bool
    wake_word_enabled: bool
    wake_word_threshold: float
    segment_silence_ms: int
    def __init__(self, flush_interval_ms: _Optional[int] = ..., min_delta_bytes: _Optional[int] = ..., overlap_bytes: _Optional[int] = ..., persistent_mode: _Optional[bool] = ..., wake_word_enabled: _Optional[bool] = ..., wake_word_threshold: _Optional[float] = ..., segment_silence_ms: _Optional[int] = ...) -> None: ...

class GetStreamConfigRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetStreamConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: StreamConfig
    def __init__(self, config: _Optional[_Union[StreamConfig, _Mapping]] = ...) -> None: ...

class UpdateStreamConfigRequest(_message.Message):
    __slots__ = ("flush_interval_ms", "has_flush_interval_ms", "min_delta_bytes", "has_min_delta_bytes", "overlap_bytes", "has_overlap_bytes", "persistent_mode", "has_persistent_mode", "wake_word_enabled", "has_wake_word_enabled", "wake_word_threshold", "has_wake_word_threshold", "segment_silence_ms", "has_segment_silence_ms")
    FLUSH_INTERVAL_MS_FIELD_NUMBER: _ClassVar[int]
    HAS_FLUSH_INTERVAL_MS_FIELD_NUMBER: _ClassVar[int]
    MIN_DELTA_BYTES_FIELD_NUMBER: _ClassVar[int]
    HAS_MIN_DELTA_BYTES_FIELD_NUMBER: _ClassVar[int]
    OVERLAP_BYTES_FIELD_NUMBER: _ClassVar[int]
    HAS_OVERLAP_BYTES_FIELD_NUMBER: _ClassVar[int]
    PERSISTENT_MODE_FIELD_NUMBER: _ClassVar[int]
    HAS_PERSISTENT_MODE_FIELD_NUMBER: _ClassVar[int]
    WAKE_WORD_ENABLED_FIELD_NUMBER: _ClassVar[int]
    HAS_WAKE_WORD_ENABLED_FIELD_NUMBER: _ClassVar[int]
    WAKE_WORD_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    HAS_WAKE_WORD_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    SEGMENT_SILENCE_MS_FIELD_NUMBER: _ClassVar[int]
    HAS_SEGMENT_SILENCE_MS_FIELD_NUMBER: _ClassVar[int]
    flush_interval_ms: int
    has_flush_interval_ms: bool
    min_delta_bytes: int
    has_min_delta_bytes: bool
    overlap_bytes: int
    has_overlap_bytes: bool
    persistent_mode: bool
    has_persistent_mode: bool
    wake_word_enabled: bool
    has_wake_word_enabled: bool
    wake_word_threshold: float
    has_wake_word_threshold: bool
    segment_silence_ms: int
    has_segment_silence_ms: bool
    def __init__(self, flush_interval_ms: _Optional[int] = ..., has_flush_interval_ms: _Optional[bool] = ..., min_delta_bytes: _Optional[int] = ..., has_min_delta_bytes: _Optional[bool] = ..., overlap_bytes: _Optional[int] = ..., has_overlap_bytes: _Optional[bool] = ..., persistent_mode: _Optional[bool] = ..., has_persistent_mode: _Optional[bool] = ..., wake_word_enabled: _Optional[bool] = ..., has_wake_word_enabled: _Optional[bool] = ..., wake_word_threshold: _Optional[float] = ..., has_wake_word_threshold: _Optional[bool] = ..., segment_silence_ms: _Optional[int] = ..., has_segment_silence_ms: _Optional[bool] = ...) -> None: ...

class UpdateStreamConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: StreamConfig
    def __init__(self, config: _Optional[_Union[StreamConfig, _Mapping]] = ...) -> None: ...

class WakeWordConfig(_message.Message):
    __slots__ = ("configured", "template_json")
    CONFIGURED_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_JSON_FIELD_NUMBER: _ClassVar[int]
    configured: bool
    template_json: str
    def __init__(self, configured: _Optional[bool] = ..., template_json: _Optional[str] = ...) -> None: ...

class GetWakeWordConfigRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetWakeWordConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: WakeWordConfig
    def __init__(self, config: _Optional[_Union[WakeWordConfig, _Mapping]] = ...) -> None: ...

class UpdateWakeWordTemplateRequest(_message.Message):
    __slots__ = ("template_json",)
    TEMPLATE_JSON_FIELD_NUMBER: _ClassVar[int]
    template_json: str
    def __init__(self, template_json: _Optional[str] = ...) -> None: ...

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
    mode: str
    reject_behavior: str
    fallback_without_verification: bool
    extraction_enabled: bool
    def __init__(self, enabled: _Optional[bool] = ..., profile_ids: _Optional[_Iterable[str]] = ..., threshold: _Optional[float] = ..., mode: _Optional[str] = ..., reject_behavior: _Optional[str] = ..., fallback_without_verification: _Optional[bool] = ..., extraction_enabled: _Optional[bool] = ...) -> None: ...

class GetSpeakerConfigRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetSpeakerConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: SpeakerConfig
    def __init__(self, config: _Optional[_Union[SpeakerConfig, _Mapping]] = ...) -> None: ...

class UpdateSpeakerConfigRequest(_message.Message):
    __slots__ = ("enabled", "has_enabled", "profile_ids", "has_profile_ids", "threshold", "has_threshold", "mode", "has_mode", "reject_behavior", "has_reject_behavior", "fallback_without_verification", "has_fallback_without_verification", "extraction_enabled", "has_extraction_enabled")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    HAS_ENABLED_FIELD_NUMBER: _ClassVar[int]
    PROFILE_IDS_FIELD_NUMBER: _ClassVar[int]
    HAS_PROFILE_IDS_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    HAS_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    HAS_MODE_FIELD_NUMBER: _ClassVar[int]
    REJECT_BEHAVIOR_FIELD_NUMBER: _ClassVar[int]
    HAS_REJECT_BEHAVIOR_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_WITHOUT_VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    HAS_FALLBACK_WITHOUT_VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    EXTRACTION_ENABLED_FIELD_NUMBER: _ClassVar[int]
    HAS_EXTRACTION_ENABLED_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    has_enabled: bool
    profile_ids: _containers.RepeatedScalarFieldContainer[str]
    has_profile_ids: bool
    threshold: float
    has_threshold: bool
    mode: str
    has_mode: bool
    reject_behavior: str
    has_reject_behavior: bool
    fallback_without_verification: bool
    has_fallback_without_verification: bool
    extraction_enabled: bool
    has_extraction_enabled: bool
    def __init__(self, enabled: _Optional[bool] = ..., has_enabled: _Optional[bool] = ..., profile_ids: _Optional[_Iterable[str]] = ..., has_profile_ids: _Optional[bool] = ..., threshold: _Optional[float] = ..., has_threshold: _Optional[bool] = ..., mode: _Optional[str] = ..., has_mode: _Optional[bool] = ..., reject_behavior: _Optional[str] = ..., has_reject_behavior: _Optional[bool] = ..., fallback_without_verification: _Optional[bool] = ..., has_fallback_without_verification: _Optional[bool] = ..., extraction_enabled: _Optional[bool] = ..., has_extraction_enabled: _Optional[bool] = ...) -> None: ...

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
    created_at: str
    updated_at: str
    model_name: str
    embedding_dim: int
    sample_rate: int
    enrollment_audio_seconds: float
    notes: str
    def __init__(self, id: _Optional[str] = ..., display_name: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., model_name: _Optional[str] = ..., embedding_dim: _Optional[int] = ..., sample_rate: _Optional[int] = ..., enrollment_audio_seconds: _Optional[float] = ..., notes: _Optional[str] = ...) -> None: ...

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
    checked_at: str
    def __init__(self, config: _Optional[_Union[SpeakerConfig, _Mapping]] = ..., capability: _Optional[str] = ..., capability_label: _Optional[str] = ..., resource_ready: _Optional[bool] = ..., profile_configured: _Optional[bool] = ..., profile_exists: _Optional[bool] = ..., profile_count: _Optional[int] = ..., profiles: _Optional[_Iterable[_Union[SpeakerProfile, _Mapping]]] = ..., info: _Optional[_Union[SpeakerResourceInfo, _Mapping]] = ..., checked_at: _Optional[str] = ...) -> None: ...

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
    created_at: str
    def __init__(self, profile_id: _Optional[str] = ..., display_name: _Optional[str] = ..., embedding_dim: _Optional[int] = ..., sample_rate: _Optional[int] = ..., enrollment_audio_seconds: _Optional[float] = ..., model_name: _Optional[str] = ..., created_at: _Optional[str] = ...) -> None: ...

class EnrollSpeakerProfileRequest(_message.Message):
    __slots__ = ("audio", "content_type", "profile_id", "display_name", "notes", "add_to_active", "has_add_to_active", "enable", "has_enable")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    ADD_TO_ACTIVE_FIELD_NUMBER: _ClassVar[int]
    HAS_ADD_TO_ACTIVE_FIELD_NUMBER: _ClassVar[int]
    ENABLE_FIELD_NUMBER: _ClassVar[int]
    HAS_ENABLE_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    content_type: str
    profile_id: str
    display_name: str
    notes: str
    add_to_active: bool
    has_add_to_active: bool
    enable: bool
    has_enable: bool
    def __init__(self, audio: _Optional[bytes] = ..., content_type: _Optional[str] = ..., profile_id: _Optional[str] = ..., display_name: _Optional[str] = ..., notes: _Optional[str] = ..., add_to_active: _Optional[bool] = ..., has_add_to_active: _Optional[bool] = ..., enable: _Optional[bool] = ..., has_enable: _Optional[bool] = ...) -> None: ...

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

class RemoveSpeakerProfileRequest(_message.Message):
    __slots__ = ("profile_id",)
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    def __init__(self, profile_id: _Optional[str] = ...) -> None: ...

class RemoveSpeakerProfileResponse(_message.Message):
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
