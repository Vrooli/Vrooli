from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from typing import ClassVar as _ClassVar

DESCRIPTOR: _descriptor.FileDescriptor

class ProviderTier(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PROVIDER_TIER_UNSPECIFIED: _ClassVar[ProviderTier]
    PROVIDER_TIER_LOCAL: _ClassVar[ProviderTier]
    PROVIDER_TIER_BYOK: _ClassVar[ProviderTier]
    PROVIDER_TIER_VROOLI: _ClassVar[ProviderTier]

class AudioFormat(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    AUDIO_FORMAT_UNSPECIFIED: _ClassVar[AudioFormat]
    AUDIO_FORMAT_WAV: _ClassVar[AudioFormat]
    AUDIO_FORMAT_MP3: _ClassVar[AudioFormat]
    AUDIO_FORMAT_FLAC: _ClassVar[AudioFormat]
    AUDIO_FORMAT_OGG: _ClassVar[AudioFormat]
    AUDIO_FORMAT_WEBM: _ClassVar[AudioFormat]
    AUDIO_FORMAT_OPUS: _ClassVar[AudioFormat]
    AUDIO_FORMAT_AAC: _ClassVar[AudioFormat]

class ResponseFormat(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RESPONSE_FORMAT_UNSPECIFIED: _ClassVar[ResponseFormat]
    RESPONSE_FORMAT_MP3: _ClassVar[ResponseFormat]
    RESPONSE_FORMAT_WAV: _ClassVar[ResponseFormat]
    RESPONSE_FORMAT_OPUS: _ClassVar[ResponseFormat]
    RESPONSE_FORMAT_FLAC: _ClassVar[ResponseFormat]

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

class SummarizeLevel(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SUMMARIZE_LEVEL_UNSPECIFIED: _ClassVar[SummarizeLevel]
    SUMMARIZE_LEVEL_LIGHT: _ClassVar[SummarizeLevel]
    SUMMARIZE_LEVEL_MODERATE: _ClassVar[SummarizeLevel]
    SUMMARIZE_LEVEL_HEAVY: _ClassVar[SummarizeLevel]

class SpeakerCapability(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SPEAKER_CAPABILITY_UNSPECIFIED: _ClassVar[SpeakerCapability]
    SPEAKER_CAPABILITY_AVAILABLE: _ClassVar[SpeakerCapability]
    SPEAKER_CAPABILITY_DEGRADED: _ClassVar[SpeakerCapability]
    SPEAKER_CAPABILITY_UNAVAILABLE: _ClassVar[SpeakerCapability]
    SPEAKER_CAPABILITY_UNINITIALIZED: _ClassVar[SpeakerCapability]
PROVIDER_TIER_UNSPECIFIED: ProviderTier
PROVIDER_TIER_LOCAL: ProviderTier
PROVIDER_TIER_BYOK: ProviderTier
PROVIDER_TIER_VROOLI: ProviderTier
AUDIO_FORMAT_UNSPECIFIED: AudioFormat
AUDIO_FORMAT_WAV: AudioFormat
AUDIO_FORMAT_MP3: AudioFormat
AUDIO_FORMAT_FLAC: AudioFormat
AUDIO_FORMAT_OGG: AudioFormat
AUDIO_FORMAT_WEBM: AudioFormat
AUDIO_FORMAT_OPUS: AudioFormat
AUDIO_FORMAT_AAC: AudioFormat
RESPONSE_FORMAT_UNSPECIFIED: ResponseFormat
RESPONSE_FORMAT_MP3: ResponseFormat
RESPONSE_FORMAT_WAV: ResponseFormat
RESPONSE_FORMAT_OPUS: ResponseFormat
RESPONSE_FORMAT_FLAC: ResponseFormat
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
SUMMARIZE_LEVEL_UNSPECIFIED: SummarizeLevel
SUMMARIZE_LEVEL_LIGHT: SummarizeLevel
SUMMARIZE_LEVEL_MODERATE: SummarizeLevel
SUMMARIZE_LEVEL_HEAVY: SummarizeLevel
SPEAKER_CAPABILITY_UNSPECIFIED: SpeakerCapability
SPEAKER_CAPABILITY_AVAILABLE: SpeakerCapability
SPEAKER_CAPABILITY_DEGRADED: SpeakerCapability
SPEAKER_CAPABILITY_UNAVAILABLE: SpeakerCapability
SPEAKER_CAPABILITY_UNINITIALIZED: SpeakerCapability
