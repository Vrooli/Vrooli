import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ConversationSearchMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONVERSATION_SEARCH_MODE_UNSPECIFIED: _ClassVar[ConversationSearchMode]
    CONVERSATION_SEARCH_MODE_HYBRID: _ClassVar[ConversationSearchMode]
    CONVERSATION_SEARCH_MODE_TEXT: _ClassVar[ConversationSearchMode]
    CONVERSATION_SEARCH_MODE_REGEX: _ClassVar[ConversationSearchMode]
    CONVERSATION_SEARCH_MODE_SEMANTIC: _ClassVar[ConversationSearchMode]

class ConversationSearchSort(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONVERSATION_SEARCH_SORT_UNSPECIFIED: _ClassVar[ConversationSearchSort]
    CONVERSATION_SEARCH_SORT_RELEVANCE: _ClassVar[ConversationSearchSort]
    CONVERSATION_SEARCH_SORT_NEWEST: _ClassVar[ConversationSearchSort]
    CONVERSATION_SEARCH_SORT_OLDEST: _ClassVar[ConversationSearchSort]

class ConversationContentClass(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONVERSATION_CONTENT_CLASS_UNSPECIFIED: _ClassVar[ConversationContentClass]
    CONVERSATION_CONTENT_CLASS_PROSE: _ClassVar[ConversationContentClass]
    CONVERSATION_CONTENT_CLASS_QUOTED_PROSE: _ClassVar[ConversationContentClass]
    CONVERSATION_CONTENT_CLASS_TOOL_CALL: _ClassVar[ConversationContentClass]
    CONVERSATION_CONTENT_CLASS_TOOL_RESULT: _ClassVar[ConversationContentClass]
    CONVERSATION_CONTENT_CLASS_INJECTED_CONTEXT: _ClassVar[ConversationContentClass]
    CONVERSATION_CONTENT_CLASS_EVIDENCE_ONLY_DUPLICATE: _ClassVar[ConversationContentClass]

class ConversationSearchLeg(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONVERSATION_SEARCH_LEG_UNSPECIFIED: _ClassVar[ConversationSearchLeg]
    CONVERSATION_SEARCH_LEG_LEXICAL: _ClassVar[ConversationSearchLeg]
    CONVERSATION_SEARCH_LEG_REGEX: _ClassVar[ConversationSearchLeg]
    CONVERSATION_SEARCH_LEG_DENSE: _ClassVar[ConversationSearchLeg]
    CONVERSATION_SEARCH_LEG_SPARSE: _ClassVar[ConversationSearchLeg]
    CONVERSATION_SEARCH_LEG_RERANK: _ClassVar[ConversationSearchLeg]

class ConversationSearchDegradationReason(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONVERSATION_SEARCH_DEGRADATION_REASON_UNSPECIFIED: _ClassVar[ConversationSearchDegradationReason]
    CONVERSATION_SEARCH_DEGRADATION_REASON_SEMANTIC_UNAVAILABLE: _ClassVar[ConversationSearchDegradationReason]
    CONVERSATION_SEARCH_DEGRADATION_REASON_EMBEDDING_UNAVAILABLE: _ClassVar[ConversationSearchDegradationReason]
    CONVERSATION_SEARCH_DEGRADATION_REASON_VECTOR_STORE_UNAVAILABLE: _ClassVar[ConversationSearchDegradationReason]
    CONVERSATION_SEARCH_DEGRADATION_REASON_INDEX_STALE: _ClassVar[ConversationSearchDegradationReason]
    CONVERSATION_SEARCH_DEGRADATION_REASON_INDEX_LAYOUT_MISMATCH: _ClassVar[ConversationSearchDegradationReason]
    CONVERSATION_SEARCH_DEGRADATION_REASON_CANDIDATE_LIMIT: _ClassVar[ConversationSearchDegradationReason]
    CONVERSATION_SEARCH_DEGRADATION_REASON_DEADLINE: _ClassVar[ConversationSearchDegradationReason]
    CONVERSATION_SEARCH_DEGRADATION_REASON_AUTHORIZATION_FILTERED: _ClassVar[ConversationSearchDegradationReason]
    CONVERSATION_SEARCH_DEGRADATION_REASON_RERANK_UNAVAILABLE: _ClassVar[ConversationSearchDegradationReason]

class ConversationIndexState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONVERSATION_INDEX_STATE_UNSPECIFIED: _ClassVar[ConversationIndexState]
    CONVERSATION_INDEX_STATE_READY: _ClassVar[ConversationIndexState]
    CONVERSATION_INDEX_STATE_BUILDING: _ClassVar[ConversationIndexState]
    CONVERSATION_INDEX_STATE_DEGRADED: _ClassVar[ConversationIndexState]
    CONVERSATION_INDEX_STATE_STALE: _ClassVar[ConversationIndexState]
    CONVERSATION_INDEX_STATE_LAYOUT_MISMATCH: _ClassVar[ConversationIndexState]
    CONVERSATION_INDEX_STATE_UNAVAILABLE: _ClassVar[ConversationIndexState]

class ConversationReindexState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONVERSATION_REINDEX_STATE_UNSPECIFIED: _ClassVar[ConversationReindexState]
    CONVERSATION_REINDEX_STATE_PLANNED: _ClassVar[ConversationReindexState]
    CONVERSATION_REINDEX_STATE_QUEUED: _ClassVar[ConversationReindexState]
    CONVERSATION_REINDEX_STATE_RUNNING: _ClassVar[ConversationReindexState]
    CONVERSATION_REINDEX_STATE_CANCELLED: _ClassVar[ConversationReindexState]
    CONVERSATION_REINDEX_STATE_FAILED: _ClassVar[ConversationReindexState]
    CONVERSATION_REINDEX_STATE_COMPLETE: _ClassVar[ConversationReindexState]
CONVERSATION_SEARCH_MODE_UNSPECIFIED: ConversationSearchMode
CONVERSATION_SEARCH_MODE_HYBRID: ConversationSearchMode
CONVERSATION_SEARCH_MODE_TEXT: ConversationSearchMode
CONVERSATION_SEARCH_MODE_REGEX: ConversationSearchMode
CONVERSATION_SEARCH_MODE_SEMANTIC: ConversationSearchMode
CONVERSATION_SEARCH_SORT_UNSPECIFIED: ConversationSearchSort
CONVERSATION_SEARCH_SORT_RELEVANCE: ConversationSearchSort
CONVERSATION_SEARCH_SORT_NEWEST: ConversationSearchSort
CONVERSATION_SEARCH_SORT_OLDEST: ConversationSearchSort
CONVERSATION_CONTENT_CLASS_UNSPECIFIED: ConversationContentClass
CONVERSATION_CONTENT_CLASS_PROSE: ConversationContentClass
CONVERSATION_CONTENT_CLASS_QUOTED_PROSE: ConversationContentClass
CONVERSATION_CONTENT_CLASS_TOOL_CALL: ConversationContentClass
CONVERSATION_CONTENT_CLASS_TOOL_RESULT: ConversationContentClass
CONVERSATION_CONTENT_CLASS_INJECTED_CONTEXT: ConversationContentClass
CONVERSATION_CONTENT_CLASS_EVIDENCE_ONLY_DUPLICATE: ConversationContentClass
CONVERSATION_SEARCH_LEG_UNSPECIFIED: ConversationSearchLeg
CONVERSATION_SEARCH_LEG_LEXICAL: ConversationSearchLeg
CONVERSATION_SEARCH_LEG_REGEX: ConversationSearchLeg
CONVERSATION_SEARCH_LEG_DENSE: ConversationSearchLeg
CONVERSATION_SEARCH_LEG_SPARSE: ConversationSearchLeg
CONVERSATION_SEARCH_LEG_RERANK: ConversationSearchLeg
CONVERSATION_SEARCH_DEGRADATION_REASON_UNSPECIFIED: ConversationSearchDegradationReason
CONVERSATION_SEARCH_DEGRADATION_REASON_SEMANTIC_UNAVAILABLE: ConversationSearchDegradationReason
CONVERSATION_SEARCH_DEGRADATION_REASON_EMBEDDING_UNAVAILABLE: ConversationSearchDegradationReason
CONVERSATION_SEARCH_DEGRADATION_REASON_VECTOR_STORE_UNAVAILABLE: ConversationSearchDegradationReason
CONVERSATION_SEARCH_DEGRADATION_REASON_INDEX_STALE: ConversationSearchDegradationReason
CONVERSATION_SEARCH_DEGRADATION_REASON_INDEX_LAYOUT_MISMATCH: ConversationSearchDegradationReason
CONVERSATION_SEARCH_DEGRADATION_REASON_CANDIDATE_LIMIT: ConversationSearchDegradationReason
CONVERSATION_SEARCH_DEGRADATION_REASON_DEADLINE: ConversationSearchDegradationReason
CONVERSATION_SEARCH_DEGRADATION_REASON_AUTHORIZATION_FILTERED: ConversationSearchDegradationReason
CONVERSATION_SEARCH_DEGRADATION_REASON_RERANK_UNAVAILABLE: ConversationSearchDegradationReason
CONVERSATION_INDEX_STATE_UNSPECIFIED: ConversationIndexState
CONVERSATION_INDEX_STATE_READY: ConversationIndexState
CONVERSATION_INDEX_STATE_BUILDING: ConversationIndexState
CONVERSATION_INDEX_STATE_DEGRADED: ConversationIndexState
CONVERSATION_INDEX_STATE_STALE: ConversationIndexState
CONVERSATION_INDEX_STATE_LAYOUT_MISMATCH: ConversationIndexState
CONVERSATION_INDEX_STATE_UNAVAILABLE: ConversationIndexState
CONVERSATION_REINDEX_STATE_UNSPECIFIED: ConversationReindexState
CONVERSATION_REINDEX_STATE_PLANNED: ConversationReindexState
CONVERSATION_REINDEX_STATE_QUEUED: ConversationReindexState
CONVERSATION_REINDEX_STATE_RUNNING: ConversationReindexState
CONVERSATION_REINDEX_STATE_CANCELLED: ConversationReindexState
CONVERSATION_REINDEX_STATE_FAILED: ConversationReindexState
CONVERSATION_REINDEX_STATE_COMPLETE: ConversationReindexState

class ConversationSearchFilters(_message.Message):
    __slots__ = ("roles", "harnesses", "provider_origins", "project_scopes", "cwd_scopes", "runners", "models", "profiles", "run_statuses", "tags", "workloads", "occurred_after", "occurred_before", "content_classes", "include_tool_events")
    ROLES_FIELD_NUMBER: _ClassVar[int]
    HARNESSES_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ORIGINS_FIELD_NUMBER: _ClassVar[int]
    PROJECT_SCOPES_FIELD_NUMBER: _ClassVar[int]
    CWD_SCOPES_FIELD_NUMBER: _ClassVar[int]
    RUNNERS_FIELD_NUMBER: _ClassVar[int]
    MODELS_FIELD_NUMBER: _ClassVar[int]
    PROFILES_FIELD_NUMBER: _ClassVar[int]
    RUN_STATUSES_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    WORKLOADS_FIELD_NUMBER: _ClassVar[int]
    OCCURRED_AFTER_FIELD_NUMBER: _ClassVar[int]
    OCCURRED_BEFORE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_CLASSES_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_TOOL_EVENTS_FIELD_NUMBER: _ClassVar[int]
    roles: _containers.RepeatedScalarFieldContainer[str]
    harnesses: _containers.RepeatedScalarFieldContainer[str]
    provider_origins: _containers.RepeatedScalarFieldContainer[str]
    project_scopes: _containers.RepeatedScalarFieldContainer[str]
    cwd_scopes: _containers.RepeatedScalarFieldContainer[str]
    runners: _containers.RepeatedScalarFieldContainer[str]
    models: _containers.RepeatedScalarFieldContainer[str]
    profiles: _containers.RepeatedScalarFieldContainer[str]
    run_statuses: _containers.RepeatedScalarFieldContainer[str]
    tags: _containers.RepeatedScalarFieldContainer[str]
    workloads: _containers.RepeatedScalarFieldContainer[str]
    occurred_after: _timestamp_pb2.Timestamp
    occurred_before: _timestamp_pb2.Timestamp
    content_classes: _containers.RepeatedScalarFieldContainer[ConversationContentClass]
    include_tool_events: bool
    def __init__(self, roles: _Optional[_Iterable[str]] = ..., harnesses: _Optional[_Iterable[str]] = ..., provider_origins: _Optional[_Iterable[str]] = ..., project_scopes: _Optional[_Iterable[str]] = ..., cwd_scopes: _Optional[_Iterable[str]] = ..., runners: _Optional[_Iterable[str]] = ..., models: _Optional[_Iterable[str]] = ..., profiles: _Optional[_Iterable[str]] = ..., run_statuses: _Optional[_Iterable[str]] = ..., tags: _Optional[_Iterable[str]] = ..., workloads: _Optional[_Iterable[str]] = ..., occurred_after: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., occurred_before: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., content_classes: _Optional[_Iterable[_Union[ConversationContentClass, str]]] = ..., include_tool_events: _Optional[bool] = ...) -> None: ...

class SearchConversationsRequest(_message.Message):
    __slots__ = ("query", "mode", "filters", "sort", "page_size", "page_cursor")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    FILTERS_FIELD_NUMBER: _ClassVar[int]
    SORT_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_CURSOR_FIELD_NUMBER: _ClassVar[int]
    query: str
    mode: ConversationSearchMode
    filters: ConversationSearchFilters
    sort: ConversationSearchSort
    page_size: int
    page_cursor: str
    def __init__(self, query: _Optional[str] = ..., mode: _Optional[_Union[ConversationSearchMode, str]] = ..., filters: _Optional[_Union[ConversationSearchFilters, _Mapping]] = ..., sort: _Optional[_Union[ConversationSearchSort, str]] = ..., page_size: _Optional[int] = ..., page_cursor: _Optional[str] = ...) -> None: ...

class ConversationSearchCursor(_message.Message):
    __slots__ = ("version", "request_fingerprint", "sort", "relevance_score", "occurred_at", "stable_hit_id")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    REQUEST_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    SORT_FIELD_NUMBER: _ClassVar[int]
    RELEVANCE_SCORE_FIELD_NUMBER: _ClassVar[int]
    OCCURRED_AT_FIELD_NUMBER: _ClassVar[int]
    STABLE_HIT_ID_FIELD_NUMBER: _ClassVar[int]
    version: int
    request_fingerprint: str
    sort: ConversationSearchSort
    relevance_score: float
    occurred_at: _timestamp_pb2.Timestamp
    stable_hit_id: str
    def __init__(self, version: _Optional[int] = ..., request_fingerprint: _Optional[str] = ..., sort: _Optional[_Union[ConversationSearchSort, str]] = ..., relevance_score: _Optional[float] = ..., occurred_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., stable_hit_id: _Optional[str] = ...) -> None: ...

class ConversationRunSummary(_message.Message):
    __slots__ = ("run_id", "label", "status", "runner", "model", "profile", "tags", "workloads", "started_at", "ended_at")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    RUNNER_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    WORKLOADS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    ENDED_AT_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    label: str
    status: str
    runner: str
    model: str
    profile: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    workloads: _containers.RepeatedScalarFieldContainer[str]
    started_at: _timestamp_pb2.Timestamp
    ended_at: _timestamp_pb2.Timestamp
    def __init__(self, run_id: _Optional[str] = ..., label: _Optional[str] = ..., status: _Optional[str] = ..., runner: _Optional[str] = ..., model: _Optional[str] = ..., profile: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., workloads: _Optional[_Iterable[str]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., ended_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ConversationSourceProvenance(_message.Message):
    __slots__ = ("harness", "source_session_id", "provider_origin", "importer", "project_scope", "cwd_scope", "evidence_ref")
    HARNESS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ORIGIN_FIELD_NUMBER: _ClassVar[int]
    IMPORTER_FIELD_NUMBER: _ClassVar[int]
    PROJECT_SCOPE_FIELD_NUMBER: _ClassVar[int]
    CWD_SCOPE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REF_FIELD_NUMBER: _ClassVar[int]
    harness: str
    source_session_id: str
    provider_origin: str
    importer: str
    project_scope: str
    cwd_scope: str
    evidence_ref: str
    def __init__(self, harness: _Optional[str] = ..., source_session_id: _Optional[str] = ..., provider_origin: _Optional[str] = ..., importer: _Optional[str] = ..., project_scope: _Optional[str] = ..., cwd_scope: _Optional[str] = ..., evidence_ref: _Optional[str] = ...) -> None: ...

class ConversationHighlight(_message.Message):
    __slots__ = ("start_grapheme", "end_grapheme", "field")
    START_GRAPHEME_FIELD_NUMBER: _ClassVar[int]
    END_GRAPHEME_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    start_grapheme: int
    end_grapheme: int
    field: str
    def __init__(self, start_grapheme: _Optional[int] = ..., end_grapheme: _Optional[int] = ..., field: _Optional[str] = ...) -> None: ...

class ConversationRankEvidence(_message.Message):
    __slots__ = ("leg", "rank", "score", "explanation")
    LEG_FIELD_NUMBER: _ClassVar[int]
    RANK_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    EXPLANATION_FIELD_NUMBER: _ClassVar[int]
    leg: ConversationSearchLeg
    rank: int
    score: float
    explanation: str
    def __init__(self, leg: _Optional[_Union[ConversationSearchLeg, str]] = ..., rank: _Optional[int] = ..., score: _Optional[float] = ..., explanation: _Optional[str] = ...) -> None: ...

class ConversationSearchHit(_message.Message):
    __slots__ = ("stable_hit_id", "run_id", "event_id", "message_id", "chunk_id", "chunk_index", "event_sequence", "role", "occurred_at", "snippet", "highlights", "content_class", "provenance", "rank_evidence", "run", "deep_link", "weak")
    STABLE_HIT_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    CHUNK_ID_FIELD_NUMBER: _ClassVar[int]
    CHUNK_INDEX_FIELD_NUMBER: _ClassVar[int]
    EVENT_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    OCCURRED_AT_FIELD_NUMBER: _ClassVar[int]
    SNIPPET_FIELD_NUMBER: _ClassVar[int]
    HIGHLIGHTS_FIELD_NUMBER: _ClassVar[int]
    CONTENT_CLASS_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    RANK_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    RUN_FIELD_NUMBER: _ClassVar[int]
    DEEP_LINK_FIELD_NUMBER: _ClassVar[int]
    WEAK_FIELD_NUMBER: _ClassVar[int]
    stable_hit_id: str
    run_id: str
    event_id: str
    message_id: str
    chunk_id: str
    chunk_index: int
    event_sequence: int
    role: str
    occurred_at: _timestamp_pb2.Timestamp
    snippet: str
    highlights: _containers.RepeatedCompositeFieldContainer[ConversationHighlight]
    content_class: ConversationContentClass
    provenance: ConversationSourceProvenance
    rank_evidence: _containers.RepeatedCompositeFieldContainer[ConversationRankEvidence]
    run: ConversationRunSummary
    deep_link: str
    weak: bool
    def __init__(self, stable_hit_id: _Optional[str] = ..., run_id: _Optional[str] = ..., event_id: _Optional[str] = ..., message_id: _Optional[str] = ..., chunk_id: _Optional[str] = ..., chunk_index: _Optional[int] = ..., event_sequence: _Optional[int] = ..., role: _Optional[str] = ..., occurred_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., snippet: _Optional[str] = ..., highlights: _Optional[_Iterable[_Union[ConversationHighlight, _Mapping]]] = ..., content_class: _Optional[_Union[ConversationContentClass, str]] = ..., provenance: _Optional[_Union[ConversationSourceProvenance, _Mapping]] = ..., rank_evidence: _Optional[_Iterable[_Union[ConversationRankEvidence, _Mapping]]] = ..., run: _Optional[_Union[ConversationRunSummary, _Mapping]] = ..., deep_link: _Optional[str] = ..., weak: _Optional[bool] = ...) -> None: ...

class ConversationSearchCoverage(_message.Message):
    __slots__ = ("canonical_visible_messages", "catalog_documents", "lexical_documents", "semantic_documents", "pending_documents", "deleted_documents", "lexical_ratio", "semantic_ratio", "last_reconciled_at", "source_checkpoint")
    CANONICAL_VISIBLE_MESSAGES_FIELD_NUMBER: _ClassVar[int]
    CATALOG_DOCUMENTS_FIELD_NUMBER: _ClassVar[int]
    LEXICAL_DOCUMENTS_FIELD_NUMBER: _ClassVar[int]
    SEMANTIC_DOCUMENTS_FIELD_NUMBER: _ClassVar[int]
    PENDING_DOCUMENTS_FIELD_NUMBER: _ClassVar[int]
    DELETED_DOCUMENTS_FIELD_NUMBER: _ClassVar[int]
    LEXICAL_RATIO_FIELD_NUMBER: _ClassVar[int]
    SEMANTIC_RATIO_FIELD_NUMBER: _ClassVar[int]
    LAST_RECONCILED_AT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_CHECKPOINT_FIELD_NUMBER: _ClassVar[int]
    canonical_visible_messages: int
    catalog_documents: int
    lexical_documents: int
    semantic_documents: int
    pending_documents: int
    deleted_documents: int
    lexical_ratio: float
    semantic_ratio: float
    last_reconciled_at: _timestamp_pb2.Timestamp
    source_checkpoint: str
    def __init__(self, canonical_visible_messages: _Optional[int] = ..., catalog_documents: _Optional[int] = ..., lexical_documents: _Optional[int] = ..., semantic_documents: _Optional[int] = ..., pending_documents: _Optional[int] = ..., deleted_documents: _Optional[int] = ..., lexical_ratio: _Optional[float] = ..., semantic_ratio: _Optional[float] = ..., last_reconciled_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., source_checkpoint: _Optional[str] = ...) -> None: ...

class ConversationSearchDegradation(_message.Message):
    __slots__ = ("reason", "leg", "detail", "retryable")
    REASON_FIELD_NUMBER: _ClassVar[int]
    LEG_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    RETRYABLE_FIELD_NUMBER: _ClassVar[int]
    reason: ConversationSearchDegradationReason
    leg: ConversationSearchLeg
    detail: str
    retryable: bool
    def __init__(self, reason: _Optional[_Union[ConversationSearchDegradationReason, str]] = ..., leg: _Optional[_Union[ConversationSearchLeg, str]] = ..., detail: _Optional[str] = ..., retryable: _Optional[bool] = ...) -> None: ...

class SearchConversationsResponse(_message.Message):
    __slots__ = ("hits", "next_page_cursor", "mode_used", "sort_used", "coverage", "degradations", "took_ms")
    HITS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_CURSOR_FIELD_NUMBER: _ClassVar[int]
    MODE_USED_FIELD_NUMBER: _ClassVar[int]
    SORT_USED_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    DEGRADATIONS_FIELD_NUMBER: _ClassVar[int]
    TOOK_MS_FIELD_NUMBER: _ClassVar[int]
    hits: _containers.RepeatedCompositeFieldContainer[ConversationSearchHit]
    next_page_cursor: str
    mode_used: ConversationSearchMode
    sort_used: ConversationSearchSort
    coverage: ConversationSearchCoverage
    degradations: _containers.RepeatedCompositeFieldContainer[ConversationSearchDegradation]
    took_ms: int
    def __init__(self, hits: _Optional[_Iterable[_Union[ConversationSearchHit, _Mapping]]] = ..., next_page_cursor: _Optional[str] = ..., mode_used: _Optional[_Union[ConversationSearchMode, str]] = ..., sort_used: _Optional[_Union[ConversationSearchSort, str]] = ..., coverage: _Optional[_Union[ConversationSearchCoverage, _Mapping]] = ..., degradations: _Optional[_Iterable[_Union[ConversationSearchDegradation, _Mapping]]] = ..., took_ms: _Optional[int] = ...) -> None: ...

class GetConversationContextRequest(_message.Message):
    __slots__ = ("stable_hit_id", "before_events", "after_events")
    STABLE_HIT_ID_FIELD_NUMBER: _ClassVar[int]
    BEFORE_EVENTS_FIELD_NUMBER: _ClassVar[int]
    AFTER_EVENTS_FIELD_NUMBER: _ClassVar[int]
    stable_hit_id: str
    before_events: int
    after_events: int
    def __init__(self, stable_hit_id: _Optional[str] = ..., before_events: _Optional[int] = ..., after_events: _Optional[int] = ...) -> None: ...

class ConversationContextEvent(_message.Message):
    __slots__ = ("event_id", "event_sequence", "role", "occurred_at", "bounded_content", "content_class", "matched")
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    EVENT_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    OCCURRED_AT_FIELD_NUMBER: _ClassVar[int]
    BOUNDED_CONTENT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_CLASS_FIELD_NUMBER: _ClassVar[int]
    MATCHED_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    event_sequence: int
    role: str
    occurred_at: _timestamp_pb2.Timestamp
    bounded_content: str
    content_class: ConversationContentClass
    matched: bool
    def __init__(self, event_id: _Optional[str] = ..., event_sequence: _Optional[int] = ..., role: _Optional[str] = ..., occurred_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., bounded_content: _Optional[str] = ..., content_class: _Optional[_Union[ConversationContentClass, str]] = ..., matched: _Optional[bool] = ...) -> None: ...

class GetConversationContextResponse(_message.Message):
    __slots__ = ("hit", "events", "degradations", "truncated")
    HIT_FIELD_NUMBER: _ClassVar[int]
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    DEGRADATIONS_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    hit: ConversationSearchHit
    events: _containers.RepeatedCompositeFieldContainer[ConversationContextEvent]
    degradations: _containers.RepeatedCompositeFieldContainer[ConversationSearchDegradation]
    truncated: bool
    def __init__(self, hit: _Optional[_Union[ConversationSearchHit, _Mapping]] = ..., events: _Optional[_Iterable[_Union[ConversationContextEvent, _Mapping]]] = ..., degradations: _Optional[_Iterable[_Union[ConversationSearchDegradation, _Mapping]]] = ..., truncated: _Optional[bool] = ...) -> None: ...

class GetConversationIndexStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetConversationIndexStatusResponse(_message.Message):
    __slots__ = ("state", "coverage", "degradations", "collection_name", "active_generation", "recipe_version", "embedding_model", "last_indexed_at", "last_success_at", "last_error_code")
    STATE_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    DEGRADATIONS_FIELD_NUMBER: _ClassVar[int]
    COLLECTION_NAME_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_GENERATION_FIELD_NUMBER: _ClassVar[int]
    RECIPE_VERSION_FIELD_NUMBER: _ClassVar[int]
    EMBEDDING_MODEL_FIELD_NUMBER: _ClassVar[int]
    LAST_INDEXED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_SUCCESS_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_ERROR_CODE_FIELD_NUMBER: _ClassVar[int]
    state: ConversationIndexState
    coverage: ConversationSearchCoverage
    degradations: _containers.RepeatedCompositeFieldContainer[ConversationSearchDegradation]
    collection_name: str
    active_generation: str
    recipe_version: str
    embedding_model: str
    last_indexed_at: _timestamp_pb2.Timestamp
    last_success_at: _timestamp_pb2.Timestamp
    last_error_code: str
    def __init__(self, state: _Optional[_Union[ConversationIndexState, str]] = ..., coverage: _Optional[_Union[ConversationSearchCoverage, _Mapping]] = ..., degradations: _Optional[_Iterable[_Union[ConversationSearchDegradation, _Mapping]]] = ..., collection_name: _Optional[str] = ..., active_generation: _Optional[str] = ..., recipe_version: _Optional[str] = ..., embedding_model: _Optional[str] = ..., last_indexed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_success_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_error_code: _Optional[str] = ...) -> None: ...

class PlanConversationReindexRequest(_message.Message):
    __slots__ = ("full", "max_documents")
    FULL_FIELD_NUMBER: _ClassVar[int]
    MAX_DOCUMENTS_FIELD_NUMBER: _ClassVar[int]
    full: bool
    max_documents: int
    def __init__(self, full: _Optional[bool] = ..., max_documents: _Optional[int] = ...) -> None: ...

class ReindexConversationsRequest(_message.Message):
    __slots__ = ("full", "max_documents", "idempotency_key")
    FULL_FIELD_NUMBER: _ClassVar[int]
    MAX_DOCUMENTS_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    full: bool
    max_documents: int
    idempotency_key: str
    def __init__(self, full: _Optional[bool] = ..., max_documents: _Optional[int] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class CancelConversationReindexRequest(_message.Message):
    __slots__ = ("operation_id",)
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    def __init__(self, operation_id: _Optional[str] = ...) -> None: ...

class ConversationReindexResponse(_message.Message):
    __slots__ = ("operation_id", "state", "dry_run", "planned_documents", "processed_documents", "upserted_documents", "deleted_documents", "failed_documents", "source_checkpoint", "shadow_generation", "active_generation", "started_at", "updated_at", "degradations")
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    PLANNED_DOCUMENTS_FIELD_NUMBER: _ClassVar[int]
    PROCESSED_DOCUMENTS_FIELD_NUMBER: _ClassVar[int]
    UPSERTED_DOCUMENTS_FIELD_NUMBER: _ClassVar[int]
    DELETED_DOCUMENTS_FIELD_NUMBER: _ClassVar[int]
    FAILED_DOCUMENTS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_CHECKPOINT_FIELD_NUMBER: _ClassVar[int]
    SHADOW_GENERATION_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_GENERATION_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    DEGRADATIONS_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    state: ConversationReindexState
    dry_run: bool
    planned_documents: int
    processed_documents: int
    upserted_documents: int
    deleted_documents: int
    failed_documents: int
    source_checkpoint: str
    shadow_generation: str
    active_generation: str
    started_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    degradations: _containers.RepeatedCompositeFieldContainer[ConversationSearchDegradation]
    def __init__(self, operation_id: _Optional[str] = ..., state: _Optional[_Union[ConversationReindexState, str]] = ..., dry_run: _Optional[bool] = ..., planned_documents: _Optional[int] = ..., processed_documents: _Optional[int] = ..., upserted_documents: _Optional[int] = ..., deleted_documents: _Optional[int] = ..., failed_documents: _Optional[int] = ..., source_checkpoint: _Optional[str] = ..., shadow_generation: _Optional[str] = ..., active_generation: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., degradations: _Optional[_Iterable[_Union[ConversationSearchDegradation, _Mapping]]] = ...) -> None: ...

class WriteConversationSearchConfigRequest(_message.Message):
    __slots__ = ("tuning", "expected_digest")
    TUNING_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_DIGEST_FIELD_NUMBER: _ClassVar[int]
    tuning: _struct_pb2.Struct
    expected_digest: str
    def __init__(self, tuning: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., expected_digest: _Optional[str] = ...) -> None: ...

class WriteConversationSearchConfigResponse(_message.Message):
    __slots__ = ("digest", "reindex_required")
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    REINDEX_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    digest: str
    reindex_required: bool
    def __init__(self, digest: _Optional[str] = ..., reindex_required: _Optional[bool] = ...) -> None: ...

class WriteConversationSearchCorpusRequest(_message.Message):
    __slots__ = ("tests", "expected_digest")
    TESTS_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_DIGEST_FIELD_NUMBER: _ClassVar[int]
    tests: _struct_pb2.Struct
    expected_digest: str
    def __init__(self, tests: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., expected_digest: _Optional[str] = ...) -> None: ...

class WriteConversationSearchCorpusResponse(_message.Message):
    __slots__ = ("digest", "reviewed_cases")
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    REVIEWED_CASES_FIELD_NUMBER: _ClassVar[int]
    digest: str
    reviewed_cases: int
    def __init__(self, digest: _Optional[str] = ..., reviewed_cases: _Optional[int] = ...) -> None: ...
