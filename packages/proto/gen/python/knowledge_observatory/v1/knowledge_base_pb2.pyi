from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SearchDocumentsRequest(_message.Message):
    __slots__ = ("query", "scope", "target", "mode", "limit")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    query: str
    scope: str
    target: str
    mode: str
    limit: int
    def __init__(self, query: _Optional[str] = ..., scope: _Optional[str] = ..., target: _Optional[str] = ..., mode: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class DocumentHit(_message.Message):
    __slots__ = ("path", "title", "snippet", "score", "metadata")
    PATH_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SNIPPET_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    path: str
    title: str
    snippet: str
    score: float
    metadata: _struct_pb2.Struct
    def __init__(self, path: _Optional[str] = ..., title: _Optional[str] = ..., snippet: _Optional[str] = ..., score: _Optional[float] = ..., metadata: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class SearchDocumentsResponse(_message.Message):
    __slots__ = ("results", "method", "reranker")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    RERANKER_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[DocumentHit]
    method: str
    reranker: str
    def __init__(self, results: _Optional[_Iterable[_Union[DocumentHit, _Mapping]]] = ..., method: _Optional[str] = ..., reranker: _Optional[str] = ...) -> None: ...

class InspectDocumentRequest(_message.Message):
    __slots__ = ("path", "offset", "limit", "expected_sha256")
    PATH_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_SHA256_FIELD_NUMBER: _ClassVar[int]
    path: str
    offset: int
    limit: int
    expected_sha256: str
    def __init__(self, path: _Optional[str] = ..., offset: _Optional[int] = ..., limit: _Optional[int] = ..., expected_sha256: _Optional[str] = ...) -> None: ...

class DocumentReference(_message.Message):
    __slots__ = ("target", "path", "exists", "kind")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    EXISTS_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    target: str
    path: str
    exists: bool
    kind: str
    def __init__(self, target: _Optional[str] = ..., path: _Optional[str] = ..., exists: _Optional[bool] = ..., kind: _Optional[str] = ...) -> None: ...

class InspectDocumentResponse(_message.Message):
    __slots__ = ("path", "sha256", "content", "offset", "next_offset", "truncated", "size_bytes", "metadata", "references", "references_truncated")
    PATH_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    NEXT_OFFSET_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    REFERENCES_FIELD_NUMBER: _ClassVar[int]
    REFERENCES_TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    path: str
    sha256: str
    content: str
    offset: int
    next_offset: int
    truncated: bool
    size_bytes: int
    metadata: _struct_pb2.Struct
    references: _containers.RepeatedCompositeFieldContainer[DocumentReference]
    references_truncated: bool
    def __init__(self, path: _Optional[str] = ..., sha256: _Optional[str] = ..., content: _Optional[str] = ..., offset: _Optional[int] = ..., next_offset: _Optional[int] = ..., truncated: _Optional[bool] = ..., size_bytes: _Optional[int] = ..., metadata: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., references: _Optional[_Iterable[_Union[DocumentReference, _Mapping]]] = ..., references_truncated: _Optional[bool] = ...) -> None: ...

class ReviewDocumentsRequest(_message.Message):
    __slots__ = ("paths", "base_path", "max_files")
    PATHS_FIELD_NUMBER: _ClassVar[int]
    BASE_PATH_FIELD_NUMBER: _ClassVar[int]
    MAX_FILES_FIELD_NUMBER: _ClassVar[int]
    paths: _containers.RepeatedScalarFieldContainer[str]
    base_path: str
    max_files: int
    def __init__(self, paths: _Optional[_Iterable[str]] = ..., base_path: _Optional[str] = ..., max_files: _Optional[int] = ...) -> None: ...

class DocumentObservation(_message.Message):
    __slots__ = ("path", "kind", "related_path", "detail")
    PATH_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    RELATED_PATH_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    path: str
    kind: str
    related_path: str
    detail: str
    def __init__(self, path: _Optional[str] = ..., kind: _Optional[str] = ..., related_path: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class ReviewDocumentsResponse(_message.Message):
    __slots__ = ("documents", "observations", "files_checked", "truncated", "base_path", "gaps")
    DOCUMENTS_FIELD_NUMBER: _ClassVar[int]
    OBSERVATIONS_FIELD_NUMBER: _ClassVar[int]
    FILES_CHECKED_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    BASE_PATH_FIELD_NUMBER: _ClassVar[int]
    GAPS_FIELD_NUMBER: _ClassVar[int]
    documents: _containers.RepeatedCompositeFieldContainer[InspectDocumentResponse]
    observations: _containers.RepeatedCompositeFieldContainer[DocumentObservation]
    files_checked: int
    truncated: bool
    base_path: str
    gaps: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, documents: _Optional[_Iterable[_Union[InspectDocumentResponse, _Mapping]]] = ..., observations: _Optional[_Iterable[_Union[DocumentObservation, _Mapping]]] = ..., files_checked: _Optional[int] = ..., truncated: _Optional[bool] = ..., base_path: _Optional[str] = ..., gaps: _Optional[_Iterable[str]] = ...) -> None: ...

class KnowledgeStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class KnowledgeStatusResponse(_message.Message):
    __slots__ = ("available", "indexed_count", "last_reconcile_at", "last_reconcile_outcome", "reranker")
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    INDEXED_COUNT_FIELD_NUMBER: _ClassVar[int]
    LAST_RECONCILE_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_RECONCILE_OUTCOME_FIELD_NUMBER: _ClassVar[int]
    RERANKER_FIELD_NUMBER: _ClassVar[int]
    available: bool
    indexed_count: int
    last_reconcile_at: str
    last_reconcile_outcome: str
    reranker: str
    def __init__(self, available: _Optional[bool] = ..., indexed_count: _Optional[int] = ..., last_reconcile_at: _Optional[str] = ..., last_reconcile_outcome: _Optional[str] = ..., reranker: _Optional[str] = ...) -> None: ...
