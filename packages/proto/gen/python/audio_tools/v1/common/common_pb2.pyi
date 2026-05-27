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
    AUDIO_FORMAT_PCM_S16LE: _ClassVar[AudioFormat]

class ResponseFormat(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RESPONSE_FORMAT_UNSPECIFIED: _ClassVar[ResponseFormat]
    RESPONSE_FORMAT_MP3: _ClassVar[ResponseFormat]
    RESPONSE_FORMAT_WAV: _ClassVar[ResponseFormat]
    RESPONSE_FORMAT_OPUS: _ClassVar[ResponseFormat]
    RESPONSE_FORMAT_FLAC: _ClassVar[ResponseFormat]

class AudioToolsFeature(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    AUDIO_TOOLS_FEATURE_UNSPECIFIED: _ClassVar[AudioToolsFeature]
    AUDIO_TOOLS_FEATURE_VOICE_INPUT: _ClassVar[AudioToolsFeature]
    AUDIO_TOOLS_FEATURE_VOICE_STREAMING: _ClassVar[AudioToolsFeature]
    AUDIO_TOOLS_FEATURE_VOICE_SPEAKER_VERIFICATION: _ClassVar[AudioToolsFeature]
    AUDIO_TOOLS_FEATURE_VOICE_ENROLLMENT: _ClassVar[AudioToolsFeature]
    AUDIO_TOOLS_FEATURE_VOICE_OUTPUT: _ClassVar[AudioToolsFeature]
    AUDIO_TOOLS_FEATURE_TTS_SUMMARIZATION: _ClassVar[AudioToolsFeature]
    AUDIO_TOOLS_FEATURE_TTS_CACHE: _ClassVar[AudioToolsFeature]
    AUDIO_TOOLS_FEATURE_TTS_PARAGRAPH_SPLIT: _ClassVar[AudioToolsFeature]
    AUDIO_TOOLS_FEATURE_AUDIO_PROVIDER_ROUTING: _ClassVar[AudioToolsFeature]
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
AUDIO_FORMAT_PCM_S16LE: AudioFormat
RESPONSE_FORMAT_UNSPECIFIED: ResponseFormat
RESPONSE_FORMAT_MP3: ResponseFormat
RESPONSE_FORMAT_WAV: ResponseFormat
RESPONSE_FORMAT_OPUS: ResponseFormat
RESPONSE_FORMAT_FLAC: ResponseFormat
AUDIO_TOOLS_FEATURE_UNSPECIFIED: AudioToolsFeature
AUDIO_TOOLS_FEATURE_VOICE_INPUT: AudioToolsFeature
AUDIO_TOOLS_FEATURE_VOICE_STREAMING: AudioToolsFeature
AUDIO_TOOLS_FEATURE_VOICE_SPEAKER_VERIFICATION: AudioToolsFeature
AUDIO_TOOLS_FEATURE_VOICE_ENROLLMENT: AudioToolsFeature
AUDIO_TOOLS_FEATURE_VOICE_OUTPUT: AudioToolsFeature
AUDIO_TOOLS_FEATURE_TTS_SUMMARIZATION: AudioToolsFeature
AUDIO_TOOLS_FEATURE_TTS_CACHE: AudioToolsFeature
AUDIO_TOOLS_FEATURE_TTS_PARAGRAPH_SPLIT: AudioToolsFeature
AUDIO_TOOLS_FEATURE_AUDIO_PROVIDER_ROUTING: AudioToolsFeature
