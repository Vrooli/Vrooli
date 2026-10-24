import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from document_manager.v1.shared import document_pb2 as _document_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Collection(_message.Message):
    __slots__ = ("id", "name", "default_privacy_class", "federated", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_PRIVACY_CLASS_FIELD_NUMBER: _ClassVar[int]
    FEDERATED_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    default_privacy_class: _document_pb2.PrivacyClass
    federated: bool
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., default_privacy_class: _Optional[_Union[_document_pb2.PrivacyClass, str]] = ..., federated: _Optional[bool] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CollectionDocument(_message.Message):
    __slots__ = ("collection_id", "document_hash", "privacy_class", "created_at")
    COLLECTION_ID_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_HASH_FIELD_NUMBER: _ClassVar[int]
    PRIVACY_CLASS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    collection_id: str
    document_hash: str
    privacy_class: _document_pb2.PrivacyClass
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, collection_id: _Optional[str] = ..., document_hash: _Optional[str] = ..., privacy_class: _Optional[_Union[_document_pb2.PrivacyClass, str]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CreateCollectionRequest(_message.Message):
    __slots__ = ("name", "default_privacy_class", "federated")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_PRIVACY_CLASS_FIELD_NUMBER: _ClassVar[int]
    FEDERATED_FIELD_NUMBER: _ClassVar[int]
    name: str
    default_privacy_class: _document_pb2.PrivacyClass
    federated: bool
    def __init__(self, name: _Optional[str] = ..., default_privacy_class: _Optional[_Union[_document_pb2.PrivacyClass, str]] = ..., federated: _Optional[bool] = ...) -> None: ...

class CreateCollectionResponse(_message.Message):
    __slots__ = ("collection",)
    COLLECTION_FIELD_NUMBER: _ClassVar[int]
    collection: Collection
    def __init__(self, collection: _Optional[_Union[Collection, _Mapping]] = ...) -> None: ...

class GetCollectionRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetCollectionResponse(_message.Message):
    __slots__ = ("collection",)
    COLLECTION_FIELD_NUMBER: _ClassVar[int]
    collection: Collection
    def __init__(self, collection: _Optional[_Union[Collection, _Mapping]] = ...) -> None: ...

class ListCollectionsRequest(_message.Message):
    __slots__ = ("limit",)
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    def __init__(self, limit: _Optional[int] = ...) -> None: ...

class ListCollectionsResponse(_message.Message):
    __slots__ = ("collections",)
    COLLECTIONS_FIELD_NUMBER: _ClassVar[int]
    collections: _containers.RepeatedCompositeFieldContainer[Collection]
    def __init__(self, collections: _Optional[_Iterable[_Union[Collection, _Mapping]]] = ...) -> None: ...

class AddDocumentRequest(_message.Message):
    __slots__ = ("collection_id", "document_hash", "privacy_class")
    COLLECTION_ID_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_HASH_FIELD_NUMBER: _ClassVar[int]
    PRIVACY_CLASS_FIELD_NUMBER: _ClassVar[int]
    collection_id: str
    document_hash: str
    privacy_class: _document_pb2.PrivacyClass
    def __init__(self, collection_id: _Optional[str] = ..., document_hash: _Optional[str] = ..., privacy_class: _Optional[_Union[_document_pb2.PrivacyClass, str]] = ...) -> None: ...

class AddDocumentResponse(_message.Message):
    __slots__ = ("document",)
    DOCUMENT_FIELD_NUMBER: _ClassVar[int]
    document: CollectionDocument
    def __init__(self, document: _Optional[_Union[CollectionDocument, _Mapping]] = ...) -> None: ...

class ListDocumentsRequest(_message.Message):
    __slots__ = ("collection_id", "limit")
    COLLECTION_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    collection_id: str
    limit: int
    def __init__(self, collection_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListDocumentsResponse(_message.Message):
    __slots__ = ("documents",)
    DOCUMENTS_FIELD_NUMBER: _ClassVar[int]
    documents: _containers.RepeatedCompositeFieldContainer[CollectionDocument]
    def __init__(self, documents: _Optional[_Iterable[_Union[CollectionDocument, _Mapping]]] = ...) -> None: ...

class ExportRequest(_message.Message):
    __slots__ = ("collection_id",)
    COLLECTION_ID_FIELD_NUMBER: _ClassVar[int]
    collection_id: str
    def __init__(self, collection_id: _Optional[str] = ...) -> None: ...

class ExportResponse(_message.Message):
    __slots__ = ("archive_json", "format")
    ARCHIVE_JSON_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    archive_json: bytes
    format: str
    def __init__(self, archive_json: _Optional[bytes] = ..., format: _Optional[str] = ...) -> None: ...

class ImportRequest(_message.Message):
    __slots__ = ("archive_json",)
    ARCHIVE_JSON_FIELD_NUMBER: _ClassVar[int]
    archive_json: bytes
    def __init__(self, archive_json: _Optional[bytes] = ...) -> None: ...

class ImportResponse(_message.Message):
    __slots__ = ("collection", "documents_imported")
    COLLECTION_FIELD_NUMBER: _ClassVar[int]
    DOCUMENTS_IMPORTED_FIELD_NUMBER: _ClassVar[int]
    collection: Collection
    documents_imported: int
    def __init__(self, collection: _Optional[_Union[Collection, _Mapping]] = ..., documents_imported: _Optional[int] = ...) -> None: ...

class PruneRequest(_message.Message):
    __slots__ = ("dry_run",)
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    def __init__(self, dry_run: _Optional[bool] = ...) -> None: ...

class PruneResponse(_message.Message):
    __slots__ = ("dry_run", "removed_kinds", "reclaimed_bytes")
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    REMOVED_KINDS_FIELD_NUMBER: _ClassVar[int]
    RECLAIMED_BYTES_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    removed_kinds: _containers.RepeatedScalarFieldContainer[str]
    reclaimed_bytes: int
    def __init__(self, dry_run: _Optional[bool] = ..., removed_kinds: _Optional[_Iterable[str]] = ..., reclaimed_bytes: _Optional[int] = ...) -> None: ...
