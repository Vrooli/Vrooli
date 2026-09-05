import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class FacetText(_message.Message):
    __slots__ = ("kind", "text", "embedding_ref")
    KIND_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    EMBEDDING_REF_FIELD_NUMBER: _ClassVar[int]
    kind: str
    text: str
    embedding_ref: str
    def __init__(self, kind: _Optional[str] = ..., text: _Optional[str] = ..., embedding_ref: _Optional[str] = ...) -> None: ...

class Attribution(_message.Message):
    __slots__ = ("actor_id", "actor_kind", "source_runtime", "verification_status", "harness_session_id", "harness_kind")
    ACTOR_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_KIND_FIELD_NUMBER: _ClassVar[int]
    SOURCE_RUNTIME_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_STATUS_FIELD_NUMBER: _ClassVar[int]
    HARNESS_SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    HARNESS_KIND_FIELD_NUMBER: _ClassVar[int]
    actor_id: str
    actor_kind: str
    source_runtime: str
    verification_status: str
    harness_session_id: str
    harness_kind: str
    def __init__(self, actor_id: _Optional[str] = ..., actor_kind: _Optional[str] = ..., source_runtime: _Optional[str] = ..., verification_status: _Optional[str] = ..., harness_session_id: _Optional[str] = ..., harness_kind: _Optional[str] = ...) -> None: ...

class Correlation(_message.Message):
    __slots__ = ("run_id", "workflow_execution_id", "actor_kind")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_KIND_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    workflow_execution_id: str
    actor_kind: str
    def __init__(self, run_id: _Optional[str] = ..., workflow_execution_id: _Optional[str] = ..., actor_kind: _Optional[str] = ...) -> None: ...

class ImportProvenance(_message.Message):
    __slots__ = ("runtime", "source_locator", "content_hash")
    RUNTIME_FIELD_NUMBER: _ClassVar[int]
    SOURCE_LOCATOR_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    runtime: str
    source_locator: str
    content_hash: str
    def __init__(self, runtime: _Optional[str] = ..., source_locator: _Optional[str] = ..., content_hash: _Optional[str] = ...) -> None: ...

class Entry(_message.Message):
    __slots__ = ("id", "body", "facet_id", "attribution", "correlation", "import_provenance", "facet_texts", "created_at", "superseded_at", "kind")
    ID_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    FACET_ID_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    CORRELATION_FIELD_NUMBER: _ClassVar[int]
    IMPORT_PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    FACET_TEXTS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDED_AT_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    id: str
    body: str
    facet_id: str
    attribution: Attribution
    correlation: Correlation
    import_provenance: ImportProvenance
    facet_texts: _containers.RepeatedCompositeFieldContainer[FacetText]
    created_at: _timestamp_pb2.Timestamp
    superseded_at: _timestamp_pb2.Timestamp
    kind: str
    def __init__(self, id: _Optional[str] = ..., body: _Optional[str] = ..., facet_id: _Optional[str] = ..., attribution: _Optional[_Union[Attribution, _Mapping]] = ..., correlation: _Optional[_Union[Correlation, _Mapping]] = ..., import_provenance: _Optional[_Union[ImportProvenance, _Mapping]] = ..., facet_texts: _Optional[_Iterable[_Union[FacetText, _Mapping]]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., superseded_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., kind: _Optional[str] = ...) -> None: ...

class AppendEntryRequest(_message.Message):
    __slots__ = ("body", "facet_id", "kind", "attribution", "correlation", "import_provenance", "trigger", "approach", "evidence", "outcome", "scope")
    BODY_FIELD_NUMBER: _ClassVar[int]
    FACET_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    CORRELATION_FIELD_NUMBER: _ClassVar[int]
    IMPORT_PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_FIELD_NUMBER: _ClassVar[int]
    APPROACH_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    body: str
    facet_id: str
    kind: str
    attribution: Attribution
    correlation: Correlation
    import_provenance: ImportProvenance
    trigger: str
    approach: str
    evidence: str
    outcome: str
    scope: str
    def __init__(self, body: _Optional[str] = ..., facet_id: _Optional[str] = ..., kind: _Optional[str] = ..., attribution: _Optional[_Union[Attribution, _Mapping]] = ..., correlation: _Optional[_Union[Correlation, _Mapping]] = ..., import_provenance: _Optional[_Union[ImportProvenance, _Mapping]] = ..., trigger: _Optional[str] = ..., approach: _Optional[str] = ..., evidence: _Optional[str] = ..., outcome: _Optional[str] = ..., scope: _Optional[str] = ...) -> None: ...

class AppendEntryResponse(_message.Message):
    __slots__ = ("entry",)
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    entry: Entry
    def __init__(self, entry: _Optional[_Union[Entry, _Mapping]] = ...) -> None: ...

class GetEntryRequest(_message.Message):
    __slots__ = ("id", "scope")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    id: str
    scope: str
    def __init__(self, id: _Optional[str] = ..., scope: _Optional[str] = ...) -> None: ...

class GetEntryResponse(_message.Message):
    __slots__ = ("entry",)
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    entry: Entry
    def __init__(self, entry: _Optional[_Union[Entry, _Mapping]] = ...) -> None: ...

class ListEntriesRequest(_message.Message):
    __slots__ = ("facet_id", "limit", "cursor", "scope")
    FACET_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    CURSOR_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    facet_id: str
    limit: int
    cursor: str
    scope: str
    def __init__(self, facet_id: _Optional[str] = ..., limit: _Optional[int] = ..., cursor: _Optional[str] = ..., scope: _Optional[str] = ...) -> None: ...

class ListEntriesResponse(_message.Message):
    __slots__ = ("entries", "next_cursor")
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    NEXT_CURSOR_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[Entry]
    next_cursor: str
    def __init__(self, entries: _Optional[_Iterable[_Union[Entry, _Mapping]]] = ..., next_cursor: _Optional[str] = ...) -> None: ...

class ProcessClassificationRetriesRequest(_message.Message):
    __slots__ = ("limit", "scope")
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    limit: int
    scope: str
    def __init__(self, limit: _Optional[int] = ..., scope: _Optional[str] = ...) -> None: ...

class ProcessClassificationRetriesResponse(_message.Message):
    __slots__ = ("processed", "deferred", "already_resolved")
    PROCESSED_FIELD_NUMBER: _ClassVar[int]
    DEFERRED_FIELD_NUMBER: _ClassVar[int]
    ALREADY_RESOLVED_FIELD_NUMBER: _ClassVar[int]
    processed: int
    deferred: int
    already_resolved: int
    def __init__(self, processed: _Optional[int] = ..., deferred: _Optional[int] = ..., already_resolved: _Optional[int] = ...) -> None: ...

class ProcessEmbeddingRetriesRequest(_message.Message):
    __slots__ = ("limit", "scope")
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    limit: int
    scope: str
    def __init__(self, limit: _Optional[int] = ..., scope: _Optional[str] = ...) -> None: ...

class ProcessEmbeddingRetriesResponse(_message.Message):
    __slots__ = ("processed", "deferred", "already_resolved")
    PROCESSED_FIELD_NUMBER: _ClassVar[int]
    DEFERRED_FIELD_NUMBER: _ClassVar[int]
    ALREADY_RESOLVED_FIELD_NUMBER: _ClassVar[int]
    processed: int
    deferred: int
    already_resolved: int
    def __init__(self, processed: _Optional[int] = ..., deferred: _Optional[int] = ..., already_resolved: _Optional[int] = ...) -> None: ...
