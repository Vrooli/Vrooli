from signal_inbox.v1.shared import signals_pb2 as _signals_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CaptureSignalRequest(_message.Message):
    __slots__ = ("url", "text", "image_payload_ref", "capture_note", "tags")
    URL_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    IMAGE_PAYLOAD_REF_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_NOTE_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    url: str
    text: str
    image_payload_ref: str
    capture_note: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, url: _Optional[str] = ..., text: _Optional[str] = ..., image_payload_ref: _Optional[str] = ..., capture_note: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ...) -> None: ...

class CaptureSignalResponse(_message.Message):
    __slots__ = ("signal", "duplicate")
    SIGNAL_FIELD_NUMBER: _ClassVar[int]
    DUPLICATE_FIELD_NUMBER: _ClassVar[int]
    signal: _signals_pb2.Signal
    duplicate: bool
    def __init__(self, signal: _Optional[_Union[_signals_pb2.Signal, _Mapping]] = ..., duplicate: _Optional[bool] = ...) -> None: ...

class ImageUpload(_message.Message):
    __slots__ = ("payload_ref", "content_type", "size_bytes")
    PAYLOAD_REF_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    payload_ref: str
    content_type: str
    size_bytes: int
    def __init__(self, payload_ref: _Optional[str] = ..., content_type: _Optional[str] = ..., size_bytes: _Optional[int] = ...) -> None: ...

class UploadImageResponse(_message.Message):
    __slots__ = ("image",)
    IMAGE_FIELD_NUMBER: _ClassVar[int]
    image: ImageUpload
    def __init__(self, image: _Optional[_Union[ImageUpload, _Mapping]] = ...) -> None: ...

class GetSignalRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetSignalResponse(_message.Message):
    __slots__ = ("signal",)
    SIGNAL_FIELD_NUMBER: _ClassVar[int]
    signal: _signals_pb2.Signal
    def __init__(self, signal: _Optional[_Union[_signals_pb2.Signal, _Mapping]] = ...) -> None: ...

class ListSignalsRequest(_message.Message):
    __slots__ = ("page_size",)
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    page_size: int
    def __init__(self, page_size: _Optional[int] = ...) -> None: ...

class ListSignalsResponse(_message.Message):
    __slots__ = ("signals",)
    SIGNALS_FIELD_NUMBER: _ClassVar[int]
    signals: _containers.RepeatedCompositeFieldContainer[_signals_pb2.Signal]
    def __init__(self, signals: _Optional[_Iterable[_Union[_signals_pb2.Signal, _Mapping]]] = ...) -> None: ...
