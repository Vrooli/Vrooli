from audio_tools.v1.common import common_pb2 as _common_pb2
from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class NormalizationMethod(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    NORMALIZATION_METHOD_UNSPECIFIED: _ClassVar[NormalizationMethod]
    NORMALIZATION_METHOD_EBU_R128: _ClassVar[NormalizationMethod]
    NORMALIZATION_METHOD_RMS: _ClassVar[NormalizationMethod]
    NORMALIZATION_METHOD_PEAK: _ClassVar[NormalizationMethod]
NORMALIZATION_METHOD_UNSPECIFIED: NormalizationMethod
NORMALIZATION_METHOD_EBU_R128: NormalizationMethod
NORMALIZATION_METHOD_RMS: NormalizationMethod
NORMALIZATION_METHOD_PEAK: NormalizationMethod

class TranscodeRequest(_message.Message):
    __slots__ = ("audio", "input_format", "output_format", "sample_rate", "channels", "bitrate")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    INPUT_FORMAT_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FORMAT_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_RATE_FIELD_NUMBER: _ClassVar[int]
    CHANNELS_FIELD_NUMBER: _ClassVar[int]
    BITRATE_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    input_format: _common_pb2.AudioFormat
    output_format: _common_pb2.AudioFormat
    sample_rate: int
    channels: int
    bitrate: int
    def __init__(self, audio: _Optional[bytes] = ..., input_format: _Optional[_Union[_common_pb2.AudioFormat, str]] = ..., output_format: _Optional[_Union[_common_pb2.AudioFormat, str]] = ..., sample_rate: _Optional[int] = ..., channels: _Optional[int] = ..., bitrate: _Optional[int] = ...) -> None: ...

class TranscodeResponse(_message.Message):
    __slots__ = ("audio", "content_type", "duration_seconds")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    content_type: str
    duration_seconds: float
    def __init__(self, audio: _Optional[bytes] = ..., content_type: _Optional[str] = ..., duration_seconds: _Optional[float] = ...) -> None: ...

class TrimRequest(_message.Message):
    __slots__ = ("audio", "format", "start_seconds", "end_seconds")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    START_SECONDS_FIELD_NUMBER: _ClassVar[int]
    END_SECONDS_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    format: _common_pb2.AudioFormat
    start_seconds: float
    end_seconds: float
    def __init__(self, audio: _Optional[bytes] = ..., format: _Optional[_Union[_common_pb2.AudioFormat, str]] = ..., start_seconds: _Optional[float] = ..., end_seconds: _Optional[float] = ...) -> None: ...

class TrimResponse(_message.Message):
    __slots__ = ("audio", "content_type", "duration_seconds")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    content_type: str
    duration_seconds: float
    def __init__(self, audio: _Optional[bytes] = ..., content_type: _Optional[str] = ..., duration_seconds: _Optional[float] = ...) -> None: ...

class MergeSource(_message.Message):
    __slots__ = ("audio", "format")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    format: _common_pb2.AudioFormat
    def __init__(self, audio: _Optional[bytes] = ..., format: _Optional[_Union[_common_pb2.AudioFormat, str]] = ...) -> None: ...

class MergeRequest(_message.Message):
    __slots__ = ("sources", "output_format", "crossfade_seconds")
    SOURCES_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FORMAT_FIELD_NUMBER: _ClassVar[int]
    CROSSFADE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    sources: _containers.RepeatedCompositeFieldContainer[MergeSource]
    output_format: _common_pb2.AudioFormat
    crossfade_seconds: float
    def __init__(self, sources: _Optional[_Iterable[_Union[MergeSource, _Mapping]]] = ..., output_format: _Optional[_Union[_common_pb2.AudioFormat, str]] = ..., crossfade_seconds: _Optional[float] = ...) -> None: ...

class MergeResponse(_message.Message):
    __slots__ = ("audio", "content_type", "duration_seconds")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    content_type: str
    duration_seconds: float
    def __init__(self, audio: _Optional[bytes] = ..., content_type: _Optional[str] = ..., duration_seconds: _Optional[float] = ...) -> None: ...

class SplitRequest(_message.Message):
    __slots__ = ("audio", "format", "chunk_seconds", "boundaries_seconds", "output_format")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    CHUNK_SECONDS_FIELD_NUMBER: _ClassVar[int]
    BOUNDARIES_SECONDS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FORMAT_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    format: _common_pb2.AudioFormat
    chunk_seconds: float
    boundaries_seconds: _containers.RepeatedScalarFieldContainer[float]
    output_format: _common_pb2.AudioFormat
    def __init__(self, audio: _Optional[bytes] = ..., format: _Optional[_Union[_common_pb2.AudioFormat, str]] = ..., chunk_seconds: _Optional[float] = ..., boundaries_seconds: _Optional[_Iterable[float]] = ..., output_format: _Optional[_Union[_common_pb2.AudioFormat, str]] = ...) -> None: ...

class SplitChunk(_message.Message):
    __slots__ = ("audio", "content_type", "start_seconds", "duration_seconds")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    START_SECONDS_FIELD_NUMBER: _ClassVar[int]
    DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    content_type: str
    start_seconds: float
    duration_seconds: float
    def __init__(self, audio: _Optional[bytes] = ..., content_type: _Optional[str] = ..., start_seconds: _Optional[float] = ..., duration_seconds: _Optional[float] = ...) -> None: ...

class SplitResponse(_message.Message):
    __slots__ = ("chunks",)
    CHUNKS_FIELD_NUMBER: _ClassVar[int]
    chunks: _containers.RepeatedCompositeFieldContainer[SplitChunk]
    def __init__(self, chunks: _Optional[_Iterable[_Union[SplitChunk, _Mapping]]] = ...) -> None: ...

class FadeRequest(_message.Message):
    __slots__ = ("audio", "format", "fade_in_seconds", "fade_out_seconds", "output_format")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    FADE_IN_SECONDS_FIELD_NUMBER: _ClassVar[int]
    FADE_OUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FORMAT_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    format: _common_pb2.AudioFormat
    fade_in_seconds: float
    fade_out_seconds: float
    output_format: _common_pb2.AudioFormat
    def __init__(self, audio: _Optional[bytes] = ..., format: _Optional[_Union[_common_pb2.AudioFormat, str]] = ..., fade_in_seconds: _Optional[float] = ..., fade_out_seconds: _Optional[float] = ..., output_format: _Optional[_Union[_common_pb2.AudioFormat, str]] = ...) -> None: ...

class FadeResponse(_message.Message):
    __slots__ = ("audio", "content_type")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    content_type: str
    def __init__(self, audio: _Optional[bytes] = ..., content_type: _Optional[str] = ...) -> None: ...

class VolumeRequest(_message.Message):
    __slots__ = ("audio", "format", "gain_db", "output_format")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    GAIN_DB_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FORMAT_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    format: _common_pb2.AudioFormat
    gain_db: float
    output_format: _common_pb2.AudioFormat
    def __init__(self, audio: _Optional[bytes] = ..., format: _Optional[_Union[_common_pb2.AudioFormat, str]] = ..., gain_db: _Optional[float] = ..., output_format: _Optional[_Union[_common_pb2.AudioFormat, str]] = ...) -> None: ...

class VolumeResponse(_message.Message):
    __slots__ = ("audio", "content_type")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    content_type: str
    def __init__(self, audio: _Optional[bytes] = ..., content_type: _Optional[str] = ...) -> None: ...

class NormalizeRequest(_message.Message):
    __slots__ = ("audio", "format", "method", "target_lufs", "output_format")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    TARGET_LUFS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FORMAT_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    format: _common_pb2.AudioFormat
    method: NormalizationMethod
    target_lufs: float
    output_format: _common_pb2.AudioFormat
    def __init__(self, audio: _Optional[bytes] = ..., format: _Optional[_Union[_common_pb2.AudioFormat, str]] = ..., method: _Optional[_Union[NormalizationMethod, str]] = ..., target_lufs: _Optional[float] = ..., output_format: _Optional[_Union[_common_pb2.AudioFormat, str]] = ...) -> None: ...

class NormalizeResponse(_message.Message):
    __slots__ = ("audio", "content_type", "measured_lufs")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    MEASURED_LUFS_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    content_type: str
    measured_lufs: float
    def __init__(self, audio: _Optional[bytes] = ..., content_type: _Optional[str] = ..., measured_lufs: _Optional[float] = ...) -> None: ...

class AudioMetadata(_message.Message):
    __slots__ = ("duration_seconds", "sample_rate", "channels", "bitrate", "codec", "format", "tags")
    class TagsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_RATE_FIELD_NUMBER: _ClassVar[int]
    CHANNELS_FIELD_NUMBER: _ClassVar[int]
    BITRATE_FIELD_NUMBER: _ClassVar[int]
    CODEC_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    duration_seconds: float
    sample_rate: int
    channels: int
    bitrate: int
    codec: str
    format: _common_pb2.AudioFormat
    tags: _containers.ScalarMap[str, str]
    def __init__(self, duration_seconds: _Optional[float] = ..., sample_rate: _Optional[int] = ..., channels: _Optional[int] = ..., bitrate: _Optional[int] = ..., codec: _Optional[str] = ..., format: _Optional[_Union[_common_pb2.AudioFormat, str]] = ..., tags: _Optional[_Mapping[str, str]] = ...) -> None: ...

class ExtractMetadataRequest(_message.Message):
    __slots__ = ("audio", "format")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    format: _common_pb2.AudioFormat
    def __init__(self, audio: _Optional[bytes] = ..., format: _Optional[_Union[_common_pb2.AudioFormat, str]] = ...) -> None: ...

class ExtractMetadataResponse(_message.Message):
    __slots__ = ("metadata",)
    METADATA_FIELD_NUMBER: _ClassVar[int]
    metadata: AudioMetadata
    def __init__(self, metadata: _Optional[_Union[AudioMetadata, _Mapping]] = ...) -> None: ...
