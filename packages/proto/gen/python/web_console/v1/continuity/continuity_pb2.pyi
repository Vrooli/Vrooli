from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class IntegrityRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class IntegrityResponse(_message.Message):
    __slots__ = ("sessions", "conversation_sessions", "conversation_events", "checkpoints", "workspace_panes", "orphan_conversations", "orphan_checkpoints", "orphan_workspace_panes", "generation", "event_content_hash")
    SESSIONS_FIELD_NUMBER: _ClassVar[int]
    CONVERSATION_SESSIONS_FIELD_NUMBER: _ClassVar[int]
    CONVERSATION_EVENTS_FIELD_NUMBER: _ClassVar[int]
    CHECKPOINTS_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_PANES_FIELD_NUMBER: _ClassVar[int]
    ORPHAN_CONVERSATIONS_FIELD_NUMBER: _ClassVar[int]
    ORPHAN_CHECKPOINTS_FIELD_NUMBER: _ClassVar[int]
    ORPHAN_WORKSPACE_PANES_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    EVENT_CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    sessions: int
    conversation_sessions: int
    conversation_events: int
    checkpoints: int
    workspace_panes: int
    orphan_conversations: int
    orphan_checkpoints: int
    orphan_workspace_panes: int
    generation: str
    event_content_hash: str
    def __init__(self, sessions: _Optional[int] = ..., conversation_sessions: _Optional[int] = ..., conversation_events: _Optional[int] = ..., checkpoints: _Optional[int] = ..., workspace_panes: _Optional[int] = ..., orphan_conversations: _Optional[int] = ..., orphan_checkpoints: _Optional[int] = ..., orphan_workspace_panes: _Optional[int] = ..., generation: _Optional[str] = ..., event_content_hash: _Optional[str] = ...) -> None: ...

class ReconcileRequest(_message.Message):
    __slots__ = ("apply", "generation", "operation_id", "manifest_hash", "offset", "batch_size")
    APPLY_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_HASH_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    BATCH_SIZE_FIELD_NUMBER: _ClassVar[int]
    apply: bool
    generation: str
    operation_id: str
    manifest_hash: str
    offset: int
    batch_size: int
    def __init__(self, apply: _Optional[bool] = ..., generation: _Optional[str] = ..., operation_id: _Optional[str] = ..., manifest_hash: _Optional[str] = ..., offset: _Optional[int] = ..., batch_size: _Optional[int] = ...) -> None: ...

class ReconcileItem(_message.Message):
    __slots__ = ("action", "session_id", "reason", "lifecycle_state", "source_fingerprint")
    ACTION_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    LIFECYCLE_STATE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    action: str
    session_id: str
    reason: str
    lifecycle_state: str
    source_fingerprint: str
    def __init__(self, action: _Optional[str] = ..., session_id: _Optional[str] = ..., reason: _Optional[str] = ..., lifecycle_state: _Optional[str] = ..., source_fingerprint: _Optional[str] = ...) -> None: ...

class ReconcileResponse(_message.Message):
    __slots__ = ("applied", "observations", "mutations", "items", "receipt_id", "receipt_status", "manifest_hash", "next_offset", "complete")
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    OBSERVATIONS_FIELD_NUMBER: _ClassVar[int]
    MUTATIONS_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_STATUS_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_HASH_FIELD_NUMBER: _ClassVar[int]
    NEXT_OFFSET_FIELD_NUMBER: _ClassVar[int]
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    applied: bool
    observations: int
    mutations: int
    items: _containers.RepeatedCompositeFieldContainer[ReconcileItem]
    receipt_id: str
    receipt_status: str
    manifest_hash: str
    next_offset: int
    complete: bool
    def __init__(self, applied: _Optional[bool] = ..., observations: _Optional[int] = ..., mutations: _Optional[int] = ..., items: _Optional[_Iterable[_Union[ReconcileItem, _Mapping]]] = ..., receipt_id: _Optional[str] = ..., receipt_status: _Optional[str] = ..., manifest_hash: _Optional[str] = ..., next_offset: _Optional[int] = ..., complete: _Optional[bool] = ...) -> None: ...

class ListCatalogRequest(_message.Message):
    __slots__ = ("lifecycle_state", "limit")
    LIFECYCLE_STATE_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    lifecycle_state: str
    limit: int
    def __init__(self, lifecycle_state: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class CatalogAlias(_message.Message):
    __slots__ = ("kind", "value")
    KIND_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    kind: str
    value: str
    def __init__(self, kind: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...

class CatalogRecord(_message.Message):
    __slots__ = ("session_id", "lifecycle_state", "backend", "agent_type", "agent_session_id", "agent_home_ref", "rollout_ref", "original_title", "current_title", "topic_summary", "cwd", "created_at", "last_activity_at", "source_fingerprint", "aliases")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    LIFECYCLE_STATE_FIELD_NUMBER: _ClassVar[int]
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    AGENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    AGENT_SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_HOME_REF_FIELD_NUMBER: _ClassVar[int]
    ROLLOUT_REF_FIELD_NUMBER: _ClassVar[int]
    ORIGINAL_TITLE_FIELD_NUMBER: _ClassVar[int]
    CURRENT_TITLE_FIELD_NUMBER: _ClassVar[int]
    TOPIC_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    CWD_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_ACTIVITY_AT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    ALIASES_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    lifecycle_state: str
    backend: str
    agent_type: str
    agent_session_id: str
    agent_home_ref: str
    rollout_ref: str
    original_title: str
    current_title: str
    topic_summary: str
    cwd: str
    created_at: str
    last_activity_at: str
    source_fingerprint: str
    aliases: _containers.RepeatedCompositeFieldContainer[CatalogAlias]
    def __init__(self, session_id: _Optional[str] = ..., lifecycle_state: _Optional[str] = ..., backend: _Optional[str] = ..., agent_type: _Optional[str] = ..., agent_session_id: _Optional[str] = ..., agent_home_ref: _Optional[str] = ..., rollout_ref: _Optional[str] = ..., original_title: _Optional[str] = ..., current_title: _Optional[str] = ..., topic_summary: _Optional[str] = ..., cwd: _Optional[str] = ..., created_at: _Optional[str] = ..., last_activity_at: _Optional[str] = ..., source_fingerprint: _Optional[str] = ..., aliases: _Optional[_Iterable[_Union[CatalogAlias, _Mapping]]] = ...) -> None: ...

class ListCatalogResponse(_message.Message):
    __slots__ = ("records", "truncated")
    RECORDS_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    records: _containers.RepeatedCompositeFieldContainer[CatalogRecord]
    truncated: bool
    def __init__(self, records: _Optional[_Iterable[_Union[CatalogRecord, _Mapping]]] = ..., truncated: _Optional[bool] = ...) -> None: ...

class SearchRequest(_message.Message):
    __slots__ = ("query", "created_after", "agent_type", "lifecycle_state", "limit", "cwd")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    CREATED_AFTER_FIELD_NUMBER: _ClassVar[int]
    AGENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    LIFECYCLE_STATE_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    CWD_FIELD_NUMBER: _ClassVar[int]
    query: str
    created_after: str
    agent_type: str
    lifecycle_state: str
    limit: int
    cwd: str
    def __init__(self, query: _Optional[str] = ..., created_after: _Optional[str] = ..., agent_type: _Optional[str] = ..., lifecycle_state: _Optional[str] = ..., limit: _Optional[int] = ..., cwd: _Optional[str] = ...) -> None: ...

class SearchMatch(_message.Message):
    __slots__ = ("event_id", "session_id", "sequence", "role", "created_at", "excerpt", "lifecycle_state")
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    EXCERPT_FIELD_NUMBER: _ClassVar[int]
    LIFECYCLE_STATE_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    session_id: str
    sequence: int
    role: str
    created_at: str
    excerpt: str
    lifecycle_state: str
    def __init__(self, event_id: _Optional[str] = ..., session_id: _Optional[str] = ..., sequence: _Optional[int] = ..., role: _Optional[str] = ..., created_at: _Optional[str] = ..., excerpt: _Optional[str] = ..., lifecycle_state: _Optional[str] = ...) -> None: ...

class SearchResponse(_message.Message):
    __slots__ = ("matches", "truncated", "total_matches", "distinct_sessions")
    MATCHES_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    TOTAL_MATCHES_FIELD_NUMBER: _ClassVar[int]
    DISTINCT_SESSIONS_FIELD_NUMBER: _ClassVar[int]
    matches: _containers.RepeatedCompositeFieldContainer[SearchMatch]
    truncated: bool
    total_matches: int
    distinct_sessions: int
    def __init__(self, matches: _Optional[_Iterable[_Union[SearchMatch, _Mapping]]] = ..., truncated: _Optional[bool] = ..., total_matches: _Optional[int] = ..., distinct_sessions: _Optional[int] = ...) -> None: ...

class GetReceiptRequest(_message.Message):
    __slots__ = ("operation_id",)
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    def __init__(self, operation_id: _Optional[str] = ...) -> None: ...

class GetReceiptResponse(_message.Message):
    __slots__ = ("operation_id", "session_id", "command", "from_state", "to_state", "status", "error_code", "created_at", "completed_at")
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    FROM_STATE_FIELD_NUMBER: _ClassVar[int]
    TO_STATE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ERROR_CODE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    session_id: str
    command: str
    from_state: str
    to_state: str
    status: str
    error_code: str
    created_at: str
    completed_at: str
    def __init__(self, operation_id: _Optional[str] = ..., session_id: _Optional[str] = ..., command: _Optional[str] = ..., from_state: _Optional[str] = ..., to_state: _Optional[str] = ..., status: _Optional[str] = ..., error_code: _Optional[str] = ..., created_at: _Optional[str] = ..., completed_at: _Optional[str] = ...) -> None: ...

class RollbackRequest(_message.Message):
    __slots__ = ("manifest_hash", "operation_id")
    MANIFEST_HASH_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    manifest_hash: str
    operation_id: str
    def __init__(self, manifest_hash: _Optional[str] = ..., operation_id: _Optional[str] = ...) -> None: ...

class RollbackResponse(_message.Message):
    __slots__ = ("manifest_hash", "receipt_id", "receipt_status")
    MANIFEST_HASH_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_STATUS_FIELD_NUMBER: _ClassVar[int]
    manifest_hash: str
    receipt_id: str
    receipt_status: str
    def __init__(self, manifest_hash: _Optional[str] = ..., receipt_id: _Optional[str] = ..., receipt_status: _Optional[str] = ...) -> None: ...

class PublishRequest(_message.Message):
    __slots__ = ("limit",)
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    def __init__(self, limit: _Optional[int] = ...) -> None: ...

class PublishResponse(_message.Message):
    __slots__ = ("attempted", "published", "failed", "next")
    ATTEMPTED_FIELD_NUMBER: _ClassVar[int]
    PUBLISHED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    NEXT_FIELD_NUMBER: _ClassVar[int]
    attempted: int
    published: int
    failed: int
    next: int
    def __init__(self, attempted: _Optional[int] = ..., published: _Optional[int] = ..., failed: _Optional[int] = ..., next: _Optional[int] = ...) -> None: ...
