from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SearchRequest(_message.Message):
    __slots__ = ("query", "collection", "namespaces", "visibility", "tags", "ingested_after", "ingested_before", "limit", "threshold")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    COLLECTION_FIELD_NUMBER: _ClassVar[int]
    NAMESPACES_FIELD_NUMBER: _ClassVar[int]
    VISIBILITY_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    INGESTED_AFTER_FIELD_NUMBER: _ClassVar[int]
    INGESTED_BEFORE_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    query: str
    collection: str
    namespaces: _containers.RepeatedScalarFieldContainer[str]
    visibility: _containers.RepeatedScalarFieldContainer[str]
    tags: _containers.RepeatedScalarFieldContainer[str]
    ingested_after: str
    ingested_before: str
    limit: int
    threshold: float
    def __init__(self, query: _Optional[str] = ..., collection: _Optional[str] = ..., namespaces: _Optional[_Iterable[str]] = ..., visibility: _Optional[_Iterable[str]] = ..., tags: _Optional[_Iterable[str]] = ..., ingested_after: _Optional[str] = ..., ingested_before: _Optional[str] = ..., limit: _Optional[int] = ..., threshold: _Optional[float] = ...) -> None: ...

class SearchResult(_message.Message):
    __slots__ = ("id", "score", "content", "metadata")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    id: str
    score: float
    content: str
    metadata: _struct_pb2.Struct
    def __init__(self, id: _Optional[str] = ..., score: _Optional[float] = ..., content: _Optional[str] = ..., metadata: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class SearchResponse(_message.Message):
    __slots__ = ("results", "query", "took_ms")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    TOOK_MS_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[SearchResult]
    query: str
    took_ms: int
    def __init__(self, results: _Optional[_Iterable[_Union[SearchResult, _Mapping]]] = ..., query: _Optional[str] = ..., took_ms: _Optional[int] = ...) -> None: ...

class DependencyStatus(_message.Message):
    __slots__ = ("connected", "latency_ms", "error", "database")
    CONNECTED_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    DATABASE_FIELD_NUMBER: _ClassVar[int]
    connected: bool
    latency_ms: float
    error: _struct_pb2.Value
    database: str
    def __init__(self, connected: _Optional[bool] = ..., latency_ms: _Optional[float] = ..., error: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., database: _Optional[str] = ...) -> None: ...

class InfrastructureHealthResponse(_message.Message):
    __slots__ = ("status", "service", "timestamp", "readiness", "version", "uptime_seconds", "dependencies", "metrics")
    class DependenciesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: DependencyStatus
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[DependencyStatus, _Mapping]] = ...) -> None: ...
    class MetricsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _struct_pb2.Value
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SERVICE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    READINESS_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    UPTIME_SECONDS_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    status: str
    service: str
    timestamp: str
    readiness: bool
    version: str
    uptime_seconds: float
    dependencies: _containers.MessageMap[str, DependencyStatus]
    metrics: _containers.MessageMap[str, _struct_pb2.Value]
    def __init__(self, status: _Optional[str] = ..., service: _Optional[str] = ..., timestamp: _Optional[str] = ..., readiness: _Optional[bool] = ..., version: _Optional[str] = ..., uptime_seconds: _Optional[float] = ..., dependencies: _Optional[_Mapping[str, DependencyStatus]] = ..., metrics: _Optional[_Mapping[str, _struct_pb2.Value]] = ...) -> None: ...

class QualityMetrics(_message.Message):
    __slots__ = ("coherence", "freshness", "redundancy", "coverage")
    COHERENCE_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    REDUNDANCY_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    coherence: float
    freshness: float
    redundancy: float
    coverage: float
    def __init__(self, coherence: _Optional[float] = ..., freshness: _Optional[float] = ..., redundancy: _Optional[float] = ..., coverage: _Optional[float] = ...) -> None: ...

class CollectionHealth(_message.Message):
    __slots__ = ("name", "size", "metrics")
    NAME_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    name: str
    size: int
    metrics: QualityMetrics
    def __init__(self, name: _Optional[str] = ..., size: _Optional[int] = ..., metrics: _Optional[_Union[QualityMetrics, _Mapping]] = ...) -> None: ...

class KnowledgeHealthResponse(_message.Message):
    __slots__ = ("total_entries", "collections", "overall_health", "overall_metrics", "timestamp")
    TOTAL_ENTRIES_FIELD_NUMBER: _ClassVar[int]
    COLLECTIONS_FIELD_NUMBER: _ClassVar[int]
    OVERALL_HEALTH_FIELD_NUMBER: _ClassVar[int]
    OVERALL_METRICS_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    total_entries: int
    collections: _containers.RepeatedCompositeFieldContainer[CollectionHealth]
    overall_health: str
    overall_metrics: QualityMetrics
    timestamp: str
    def __init__(self, total_entries: _Optional[int] = ..., collections: _Optional[_Iterable[_Union[CollectionHealth, _Mapping]]] = ..., overall_health: _Optional[str] = ..., overall_metrics: _Optional[_Union[QualityMetrics, _Mapping]] = ..., timestamp: _Optional[str] = ...) -> None: ...
