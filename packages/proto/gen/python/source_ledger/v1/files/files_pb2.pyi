from source_ledger.v1.shared import degradation_pb2 as _degradation_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SearchRequest(_message.Message):
    __slots__ = ("query", "corpus", "topic", "feed", "limit", "include_raw")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    CORPUS_FIELD_NUMBER: _ClassVar[int]
    TOPIC_FIELD_NUMBER: _ClassVar[int]
    FEED_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_RAW_FIELD_NUMBER: _ClassVar[int]
    query: str
    corpus: str
    topic: str
    feed: str
    limit: int
    include_raw: bool
    def __init__(self, query: _Optional[str] = ..., corpus: _Optional[str] = ..., topic: _Optional[str] = ..., feed: _Optional[str] = ..., limit: _Optional[int] = ..., include_raw: _Optional[bool] = ...) -> None: ...

class Hit(_message.Message):
    __slots__ = ("citation", "path", "abs_path", "topic", "feed", "line_start", "line_end", "heading", "snippet", "score", "modified_at", "truncated")
    CITATION_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    ABS_PATH_FIELD_NUMBER: _ClassVar[int]
    TOPIC_FIELD_NUMBER: _ClassVar[int]
    FEED_FIELD_NUMBER: _ClassVar[int]
    LINE_START_FIELD_NUMBER: _ClassVar[int]
    LINE_END_FIELD_NUMBER: _ClassVar[int]
    HEADING_FIELD_NUMBER: _ClassVar[int]
    SNIPPET_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    MODIFIED_AT_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    citation: str
    path: str
    abs_path: str
    topic: str
    feed: str
    line_start: int
    line_end: int
    heading: str
    snippet: str
    score: float
    modified_at: str
    truncated: bool
    def __init__(self, citation: _Optional[str] = ..., path: _Optional[str] = ..., abs_path: _Optional[str] = ..., topic: _Optional[str] = ..., feed: _Optional[str] = ..., line_start: _Optional[int] = ..., line_end: _Optional[int] = ..., heading: _Optional[str] = ..., snippet: _Optional[str] = ..., score: _Optional[float] = ..., modified_at: _Optional[str] = ..., truncated: _Optional[bool] = ...) -> None: ...

class SyncReport(_message.Message):
    __slots__ = ("files_seen", "files_changed", "files_removed", "duration_ms", "synced_at", "partial", "pending", "skipped_private", "skipped_excluded", "summarised", "skipped_oversize_json", "truncated", "problems")
    FILES_SEEN_FIELD_NUMBER: _ClassVar[int]
    FILES_CHANGED_FIELD_NUMBER: _ClassVar[int]
    FILES_REMOVED_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    SYNCED_AT_FIELD_NUMBER: _ClassVar[int]
    PARTIAL_FIELD_NUMBER: _ClassVar[int]
    PENDING_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_PRIVATE_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_EXCLUDED_FIELD_NUMBER: _ClassVar[int]
    SUMMARISED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_OVERSIZE_JSON_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    PROBLEMS_FIELD_NUMBER: _ClassVar[int]
    files_seen: int
    files_changed: int
    files_removed: int
    duration_ms: int
    synced_at: str
    partial: bool
    pending: int
    skipped_private: int
    skipped_excluded: int
    summarised: int
    skipped_oversize_json: int
    truncated: int
    problems: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, files_seen: _Optional[int] = ..., files_changed: _Optional[int] = ..., files_removed: _Optional[int] = ..., duration_ms: _Optional[int] = ..., synced_at: _Optional[str] = ..., partial: _Optional[bool] = ..., pending: _Optional[int] = ..., skipped_private: _Optional[int] = ..., skipped_excluded: _Optional[int] = ..., summarised: _Optional[int] = ..., skipped_oversize_json: _Optional[int] = ..., truncated: _Optional[int] = ..., problems: _Optional[_Iterable[str]] = ...) -> None: ...

class SearchResponse(_message.Message):
    __slots__ = ("hits", "sync", "mode", "degradations", "source")
    HITS_FIELD_NUMBER: _ClassVar[int]
    SYNC_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    DEGRADATIONS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    hits: _containers.RepeatedCompositeFieldContainer[Hit]
    sync: SyncReport
    mode: str
    degradations: _containers.RepeatedCompositeFieldContainer[_degradation_pb2.Degradation]
    source: str
    def __init__(self, hits: _Optional[_Iterable[_Union[Hit, _Mapping]]] = ..., sync: _Optional[_Union[SyncReport, _Mapping]] = ..., mode: _Optional[str] = ..., degradations: _Optional[_Iterable[_Union[_degradation_pb2.Degradation, _Mapping]]] = ..., source: _Optional[str] = ...) -> None: ...

class ListFeedsRequest(_message.Message):
    __slots__ = ("corpus", "topic", "recent")
    CORPUS_FIELD_NUMBER: _ClassVar[int]
    TOPIC_FIELD_NUMBER: _ClassVar[int]
    RECENT_FIELD_NUMBER: _ClassVar[int]
    corpus: str
    topic: str
    recent: int
    def __init__(self, corpus: _Optional[str] = ..., topic: _Optional[str] = ..., recent: _Optional[int] = ...) -> None: ...

class FeedItem(_message.Message):
    __slots__ = ("path", "heading", "modified_at")
    PATH_FIELD_NUMBER: _ClassVar[int]
    HEADING_FIELD_NUMBER: _ClassVar[int]
    MODIFIED_AT_FIELD_NUMBER: _ClassVar[int]
    path: str
    heading: str
    modified_at: str
    def __init__(self, path: _Optional[str] = ..., heading: _Optional[str] = ..., modified_at: _Optional[str] = ...) -> None: ...

class Feed(_message.Message):
    __slots__ = ("topic", "feed", "files", "bytes", "last_modified", "recent", "chunks", "embedded")
    TOPIC_FIELD_NUMBER: _ClassVar[int]
    FEED_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    BYTES_FIELD_NUMBER: _ClassVar[int]
    LAST_MODIFIED_FIELD_NUMBER: _ClassVar[int]
    RECENT_FIELD_NUMBER: _ClassVar[int]
    CHUNKS_FIELD_NUMBER: _ClassVar[int]
    EMBEDDED_FIELD_NUMBER: _ClassVar[int]
    topic: str
    feed: str
    files: int
    bytes: int
    last_modified: str
    recent: _containers.RepeatedCompositeFieldContainer[FeedItem]
    chunks: int
    embedded: int
    def __init__(self, topic: _Optional[str] = ..., feed: _Optional[str] = ..., files: _Optional[int] = ..., bytes: _Optional[int] = ..., last_modified: _Optional[str] = ..., recent: _Optional[_Iterable[_Union[FeedItem, _Mapping]]] = ..., chunks: _Optional[int] = ..., embedded: _Optional[int] = ...) -> None: ...

class ListFeedsResponse(_message.Message):
    __slots__ = ("feeds", "sync", "degradations")
    FEEDS_FIELD_NUMBER: _ClassVar[int]
    SYNC_FIELD_NUMBER: _ClassVar[int]
    DEGRADATIONS_FIELD_NUMBER: _ClassVar[int]
    feeds: _containers.RepeatedCompositeFieldContainer[Feed]
    sync: SyncReport
    degradations: _containers.RepeatedCompositeFieldContainer[_degradation_pb2.Degradation]
    def __init__(self, feeds: _Optional[_Iterable[_Union[Feed, _Mapping]]] = ..., sync: _Optional[_Union[SyncReport, _Mapping]] = ..., degradations: _Optional[_Iterable[_Union[_degradation_pb2.Degradation, _Mapping]]] = ...) -> None: ...
