import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SourceKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SOURCE_KIND_UNSPECIFIED: _ClassVar[SourceKind]
    SOURCE_KIND_URL: _ClassVar[SourceKind]
    SOURCE_KIND_TEXT: _ClassVar[SourceKind]
    SOURCE_KIND_IMAGE: _ClassVar[SourceKind]
SOURCE_KIND_UNSPECIFIED: SourceKind
SOURCE_KIND_URL: SourceKind
SOURCE_KIND_TEXT: SourceKind
SOURCE_KIND_IMAGE: SourceKind

class Signal(_message.Message):
    __slots__ = ("id", "source_kind", "source_identity", "source_url", "captured_at", "raw_payload_ref", "extracted_content", "content_hash", "needs_attention", "capture_note", "tags")
    ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_KIND_FIELD_NUMBER: _ClassVar[int]
    SOURCE_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    SOURCE_URL_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    RAW_PAYLOAD_REF_FIELD_NUMBER: _ClassVar[int]
    EXTRACTED_CONTENT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    NEEDS_ATTENTION_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_NOTE_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    id: str
    source_kind: SourceKind
    source_identity: str
    source_url: str
    captured_at: _timestamp_pb2.Timestamp
    raw_payload_ref: str
    extracted_content: str
    content_hash: str
    needs_attention: bool
    capture_note: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., source_kind: _Optional[_Union[SourceKind, str]] = ..., source_identity: _Optional[str] = ..., source_url: _Optional[str] = ..., captured_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., raw_payload_ref: _Optional[str] = ..., extracted_content: _Optional[str] = ..., content_hash: _Optional[str] = ..., needs_attention: _Optional[bool] = ..., capture_note: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ...) -> None: ...
