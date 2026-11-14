import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from document_manager.v1.shared import document_pb2 as _document_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Document(_message.Message):
    __slots__ = ("id", "content_sha256", "source_name", "detected_mime", "pdf_type", "pdf_confidence", "privacy_class", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    CONTENT_SHA256_FIELD_NUMBER: _ClassVar[int]
    SOURCE_NAME_FIELD_NUMBER: _ClassVar[int]
    DETECTED_MIME_FIELD_NUMBER: _ClassVar[int]
    PDF_TYPE_FIELD_NUMBER: _ClassVar[int]
    PDF_CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    PRIVACY_CLASS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    content_sha256: str
    source_name: str
    detected_mime: str
    pdf_type: str
    pdf_confidence: float
    privacy_class: _document_pb2.PrivacyClass
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., content_sha256: _Optional[str] = ..., source_name: _Optional[str] = ..., detected_mime: _Optional[str] = ..., pdf_type: _Optional[str] = ..., pdf_confidence: _Optional[float] = ..., privacy_class: _Optional[_Union[_document_pb2.PrivacyClass, str]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class IngestRequest(_message.Message):
    __slots__ = ("content", "source_name", "collection_id", "privacy_class")
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_NAME_FIELD_NUMBER: _ClassVar[int]
    COLLECTION_ID_FIELD_NUMBER: _ClassVar[int]
    PRIVACY_CLASS_FIELD_NUMBER: _ClassVar[int]
    content: bytes
    source_name: str
    collection_id: str
    privacy_class: _document_pb2.PrivacyClass
    def __init__(self, content: _Optional[bytes] = ..., source_name: _Optional[str] = ..., collection_id: _Optional[str] = ..., privacy_class: _Optional[_Union[_document_pb2.PrivacyClass, str]] = ...) -> None: ...

class IngestResponse(_message.Message):
    __slots__ = ("document", "duplicate")
    DOCUMENT_FIELD_NUMBER: _ClassVar[int]
    DUPLICATE_FIELD_NUMBER: _ClassVar[int]
    document: Document
    duplicate: bool
    def __init__(self, document: _Optional[_Union[Document, _Mapping]] = ..., duplicate: _Optional[bool] = ...) -> None: ...

class GetDocumentRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetDocumentResponse(_message.Message):
    __slots__ = ("document",)
    DOCUMENT_FIELD_NUMBER: _ClassVar[int]
    document: Document
    def __init__(self, document: _Optional[_Union[Document, _Mapping]] = ...) -> None: ...

class ListDocumentsRequest(_message.Message):
    __slots__ = ("limit",)
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    def __init__(self, limit: _Optional[int] = ...) -> None: ...

class ListDocumentsResponse(_message.Message):
    __slots__ = ("documents",)
    DOCUMENTS_FIELD_NUMBER: _ClassVar[int]
    documents: _containers.RepeatedCompositeFieldContainer[Document]
    def __init__(self, documents: _Optional[_Iterable[_Union[Document, _Mapping]]] = ...) -> None: ...

class ListSourcesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListSourcesResponse(_message.Message):
    __slots__ = ("sources",)
    SOURCES_FIELD_NUMBER: _ClassVar[int]
    sources: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, sources: _Optional[_Iterable[str]] = ...) -> None: ...

class ConfigureWatchRequest(_message.Message):
    __slots__ = ("path", "enabled")
    PATH_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    path: str
    enabled: bool
    def __init__(self, path: _Optional[str] = ..., enabled: _Optional[bool] = ...) -> None: ...

class ConfigureWatchResponse(_message.Message):
    __slots__ = ("path", "enabled")
    PATH_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    path: str
    enabled: bool
    def __init__(self, path: _Optional[str] = ..., enabled: _Optional[bool] = ...) -> None: ...

class GetTypeVerdictRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetTypeVerdictResponse(_message.Message):
    __slots__ = ("detected_mime", "pdf_type", "confidence")
    DETECTED_MIME_FIELD_NUMBER: _ClassVar[int]
    PDF_TYPE_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    detected_mime: str
    pdf_type: str
    confidence: float
    def __init__(self, detected_mime: _Optional[str] = ..., pdf_type: _Optional[str] = ..., confidence: _Optional[float] = ...) -> None: ...
