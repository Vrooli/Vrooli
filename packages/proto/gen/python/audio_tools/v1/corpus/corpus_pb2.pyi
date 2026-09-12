import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ClipSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CLIP_SOURCE_UNSPECIFIED: _ClassVar[ClipSource]
    CLIP_SOURCE_FREE_FORM: _ClassVar[ClipSource]
    CLIP_SOURCE_SCRIPTED: _ClassVar[ClipSource]
CLIP_SOURCE_UNSPECIFIED: ClipSource
CLIP_SOURCE_FREE_FORM: ClipSource
CLIP_SOURCE_SCRIPTED: ClipSource

class Clip(_message.Message):
    __slots__ = ("id", "reference_text", "tags", "duration_ms", "sample_rate_hz", "format", "blob_key", "source", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_TEXT_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_RATE_HZ_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    BLOB_KEY_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    reference_text: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    duration_ms: int
    sample_rate_hz: int
    format: str
    blob_key: str
    source: ClipSource
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., reference_text: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., duration_ms: _Optional[int] = ..., sample_rate_hz: _Optional[int] = ..., format: _Optional[str] = ..., blob_key: _Optional[str] = ..., source: _Optional[_Union[ClipSource, str]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CreateClipRequest(_message.Message):
    __slots__ = ("audio", "reference_text", "tags", "duration_ms", "sample_rate_hz", "format", "source")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_TEXT_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_RATE_HZ_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    reference_text: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    duration_ms: int
    sample_rate_hz: int
    format: str
    source: ClipSource
    def __init__(self, audio: _Optional[bytes] = ..., reference_text: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., duration_ms: _Optional[int] = ..., sample_rate_hz: _Optional[int] = ..., format: _Optional[str] = ..., source: _Optional[_Union[ClipSource, str]] = ...) -> None: ...

class CreateClipResponse(_message.Message):
    __slots__ = ("clip",)
    CLIP_FIELD_NUMBER: _ClassVar[int]
    clip: Clip
    def __init__(self, clip: _Optional[_Union[Clip, _Mapping]] = ...) -> None: ...

class ListClipsRequest(_message.Message):
    __slots__ = ("tag_contains", "limit", "offset")
    TAG_CONTAINS_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    tag_contains: str
    limit: int
    offset: int
    def __init__(self, tag_contains: _Optional[str] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListClipsResponse(_message.Message):
    __slots__ = ("clips",)
    CLIPS_FIELD_NUMBER: _ClassVar[int]
    clips: _containers.RepeatedCompositeFieldContainer[Clip]
    def __init__(self, clips: _Optional[_Iterable[_Union[Clip, _Mapping]]] = ...) -> None: ...

class GetClipRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetClipResponse(_message.Message):
    __slots__ = ("clip",)
    CLIP_FIELD_NUMBER: _ClassVar[int]
    clip: Clip
    def __init__(self, clip: _Optional[_Union[Clip, _Mapping]] = ...) -> None: ...

class GetClipAudioRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetClipAudioResponse(_message.Message):
    __slots__ = ("audio", "clip")
    AUDIO_FIELD_NUMBER: _ClassVar[int]
    CLIP_FIELD_NUMBER: _ClassVar[int]
    audio: bytes
    clip: Clip
    def __init__(self, audio: _Optional[bytes] = ..., clip: _Optional[_Union[Clip, _Mapping]] = ...) -> None: ...

class DeleteClipRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteClipResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
