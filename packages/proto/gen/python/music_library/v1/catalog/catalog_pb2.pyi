from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Track(_message.Message):
    __slots__ = ("sha256", "title", "source_path", "take_id", "receipt_path", "tags", "approval", "feedback", "uses")
    SHA256_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    TAKE_ID_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_PATH_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    APPROVAL_FIELD_NUMBER: _ClassVar[int]
    FEEDBACK_FIELD_NUMBER: _ClassVar[int]
    USES_FIELD_NUMBER: _ClassVar[int]
    sha256: str
    title: str
    source_path: str
    take_id: str
    receipt_path: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    approval: str
    feedback: str
    uses: _containers.RepeatedCompositeFieldContainer[TrackUse]
    def __init__(self, sha256: _Optional[str] = ..., title: _Optional[str] = ..., source_path: _Optional[str] = ..., take_id: _Optional[str] = ..., receipt_path: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., approval: _Optional[str] = ..., feedback: _Optional[str] = ..., uses: _Optional[_Iterable[_Union[TrackUse, _Mapping]]] = ...) -> None: ...

class TrackUse(_message.Message):
    __slots__ = ("project", "revision", "status")
    PROJECT_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    project: str
    revision: str
    status: str
    def __init__(self, project: _Optional[str] = ..., revision: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class PutTrackRequest(_message.Message):
    __slots__ = ("track",)
    TRACK_FIELD_NUMBER: _ClassVar[int]
    track: Track
    def __init__(self, track: _Optional[_Union[Track, _Mapping]] = ...) -> None: ...

class PutTrackResponse(_message.Message):
    __slots__ = ("track",)
    TRACK_FIELD_NUMBER: _ClassVar[int]
    track: Track
    def __init__(self, track: _Optional[_Union[Track, _Mapping]] = ...) -> None: ...

class ListTracksRequest(_message.Message):
    __slots__ = ("tag", "unused_only", "project")
    TAG_FIELD_NUMBER: _ClassVar[int]
    UNUSED_ONLY_FIELD_NUMBER: _ClassVar[int]
    PROJECT_FIELD_NUMBER: _ClassVar[int]
    tag: str
    unused_only: bool
    project: str
    def __init__(self, tag: _Optional[str] = ..., unused_only: _Optional[bool] = ..., project: _Optional[str] = ...) -> None: ...

class ListTracksResponse(_message.Message):
    __slots__ = ("tracks",)
    TRACKS_FIELD_NUMBER: _ClassVar[int]
    tracks: _containers.RepeatedCompositeFieldContainer[Track]
    def __init__(self, tracks: _Optional[_Iterable[_Union[Track, _Mapping]]] = ...) -> None: ...

class RecordUseRequest(_message.Message):
    __slots__ = ("sha256", "use")
    SHA256_FIELD_NUMBER: _ClassVar[int]
    USE_FIELD_NUMBER: _ClassVar[int]
    sha256: str
    use: TrackUse
    def __init__(self, sha256: _Optional[str] = ..., use: _Optional[_Union[TrackUse, _Mapping]] = ...) -> None: ...

class RecordUseResponse(_message.Message):
    __slots__ = ("track",)
    TRACK_FIELD_NUMBER: _ClassVar[int]
    track: Track
    def __init__(self, track: _Optional[_Union[Track, _Mapping]] = ...) -> None: ...
