from document_manager.v1.shared import document_pb2 as _document_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class QueryRequest(_message.Message):
    __slots__ = ("text", "collection_id", "caller_max_privacy", "federated", "limit")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    COLLECTION_ID_FIELD_NUMBER: _ClassVar[int]
    CALLER_MAX_PRIVACY_FIELD_NUMBER: _ClassVar[int]
    FEDERATED_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    text: str
    collection_id: str
    caller_max_privacy: _document_pb2.PrivacyClass
    federated: bool
    limit: int
    def __init__(self, text: _Optional[str] = ..., collection_id: _Optional[str] = ..., caller_max_privacy: _Optional[_Union[_document_pb2.PrivacyClass, str]] = ..., federated: _Optional[bool] = ..., limit: _Optional[int] = ...) -> None: ...

class QueryResult(_message.Message):
    __slots__ = ("unit_id", "document_hash", "anchor_uri", "score")
    UNIT_ID_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_HASH_FIELD_NUMBER: _ClassVar[int]
    ANCHOR_URI_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    unit_id: str
    document_hash: str
    anchor_uri: str
    score: float
    def __init__(self, unit_id: _Optional[str] = ..., document_hash: _Optional[str] = ..., anchor_uri: _Optional[str] = ..., score: _Optional[float] = ...) -> None: ...

class QueryResponse(_message.Message):
    __slots__ = ("results", "partial")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    PARTIAL_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[QueryResult]
    partial: bool
    def __init__(self, results: _Optional[_Iterable[_Union[QueryResult, _Mapping]]] = ..., partial: _Optional[bool] = ...) -> None: ...

class SimilarRequest(_message.Message):
    __slots__ = ("document_hash", "limit", "caller_max_privacy")
    DOCUMENT_HASH_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    CALLER_MAX_PRIVACY_FIELD_NUMBER: _ClassVar[int]
    document_hash: str
    limit: int
    caller_max_privacy: _document_pb2.PrivacyClass
    def __init__(self, document_hash: _Optional[str] = ..., limit: _Optional[int] = ..., caller_max_privacy: _Optional[_Union[_document_pb2.PrivacyClass, str]] = ...) -> None: ...

class SimilarResponse(_message.Message):
    __slots__ = ("results", "partial")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    PARTIAL_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[QueryResult]
    partial: bool
    def __init__(self, results: _Optional[_Iterable[_Union[QueryResult, _Mapping]]] = ..., partial: _Optional[bool] = ...) -> None: ...
