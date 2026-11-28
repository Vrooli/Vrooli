from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PublishRequest(_message.Message):
    __slots__ = ("package_id", "channel", "source_revision")
    PACKAGE_ID_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REVISION_FIELD_NUMBER: _ClassVar[int]
    package_id: str
    channel: str
    source_revision: str
    def __init__(self, package_id: _Optional[str] = ..., channel: _Optional[str] = ..., source_revision: _Optional[str] = ...) -> None: ...

class PublishResponse(_message.Message):
    __slots__ = ("published", "coordinate", "digest", "confirmation_reference", "refusal")
    PUBLISHED_FIELD_NUMBER: _ClassVar[int]
    COORDINATE_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    CONFIRMATION_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    REFUSAL_FIELD_NUMBER: _ClassVar[int]
    published: bool
    coordinate: str
    digest: str
    confirmation_reference: str
    refusal: str
    def __init__(self, published: _Optional[bool] = ..., coordinate: _Optional[str] = ..., digest: _Optional[str] = ..., confirmation_reference: _Optional[str] = ..., refusal: _Optional[str] = ...) -> None: ...

class RevokeRequest(_message.Message):
    __slots__ = ("package_id",)
    PACKAGE_ID_FIELD_NUMBER: _ClassVar[int]
    package_id: str
    def __init__(self, package_id: _Optional[str] = ...) -> None: ...

class RevokeResponse(_message.Message):
    __slots__ = ("complete", "outcomes")
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    complete: bool
    outcomes: _containers.RepeatedCompositeFieldContainer[ChannelOutcome]
    def __init__(self, complete: _Optional[bool] = ..., outcomes: _Optional[_Iterable[_Union[ChannelOutcome, _Mapping]]] = ...) -> None: ...

class ChannelOutcome(_message.Message):
    __slots__ = ("channel", "withdrawn", "detail")
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    WITHDRAWN_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    channel: str
    withdrawn: bool
    detail: str
    def __init__(self, channel: _Optional[str] = ..., withdrawn: _Optional[bool] = ..., detail: _Optional[str] = ...) -> None: ...
