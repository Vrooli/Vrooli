import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Attachment(_message.Message):
    __slots__ = ("key", "mime_type", "size_bytes", "note_id", "uploaded_at")
    KEY_FIELD_NUMBER: _ClassVar[int]
    MIME_TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    NOTE_ID_FIELD_NUMBER: _ClassVar[int]
    UPLOADED_AT_FIELD_NUMBER: _ClassVar[int]
    key: str
    mime_type: str
    size_bytes: int
    note_id: str
    uploaded_at: _timestamp_pb2.Timestamp
    def __init__(self, key: _Optional[str] = ..., mime_type: _Optional[str] = ..., size_bytes: _Optional[int] = ..., note_id: _Optional[str] = ..., uploaded_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class UploadAttachmentResponse(_message.Message):
    __slots__ = ("attachment",)
    ATTACHMENT_FIELD_NUMBER: _ClassVar[int]
    attachment: Attachment
    def __init__(self, attachment: _Optional[_Union[Attachment, _Mapping]] = ...) -> None: ...
