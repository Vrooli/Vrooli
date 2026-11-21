from document_manager.v1.shared import document_pb2 as _document_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EnrichRequest(_message.Message):
    __slots__ = ("document_hash", "text", "privacy_class")
    DOCUMENT_HASH_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    PRIVACY_CLASS_FIELD_NUMBER: _ClassVar[int]
    document_hash: str
    text: str
    privacy_class: _document_pb2.PrivacyClass
    def __init__(self, document_hash: _Optional[str] = ..., text: _Optional[str] = ..., privacy_class: _Optional[_Union[_document_pb2.PrivacyClass, str]] = ...) -> None: ...

class EnrichResponse(_message.Message):
    __slots__ = ("enriched", "summary", "suggested_privacy_class", "status")
    ENRICHED_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    SUGGESTED_PRIVACY_CLASS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    enriched: bool
    summary: str
    suggested_privacy_class: _document_pb2.PrivacyClass
    status: str
    def __init__(self, enriched: _Optional[bool] = ..., summary: _Optional[str] = ..., suggested_privacy_class: _Optional[_Union[_document_pb2.PrivacyClass, str]] = ..., status: _Optional[str] = ...) -> None: ...

class EmbedRequest(_message.Message):
    __slots__ = ("document_hash", "unit_id", "text", "privacy_class", "role")
    DOCUMENT_HASH_FIELD_NUMBER: _ClassVar[int]
    UNIT_ID_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    PRIVACY_CLASS_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    document_hash: str
    unit_id: str
    text: str
    privacy_class: _document_pb2.PrivacyClass
    role: str
    def __init__(self, document_hash: _Optional[str] = ..., unit_id: _Optional[str] = ..., text: _Optional[str] = ..., privacy_class: _Optional[_Union[_document_pb2.PrivacyClass, str]] = ..., role: _Optional[str] = ...) -> None: ...

class EmbedResponse(_message.Message):
    __slots__ = ("embedding_id", "enriched", "dimension", "status")
    EMBEDDING_ID_FIELD_NUMBER: _ClassVar[int]
    ENRICHED_FIELD_NUMBER: _ClassVar[int]
    DIMENSION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    embedding_id: str
    enriched: bool
    dimension: int
    status: str
    def __init__(self, embedding_id: _Optional[str] = ..., enriched: _Optional[bool] = ..., dimension: _Optional[int] = ..., status: _Optional[str] = ...) -> None: ...
