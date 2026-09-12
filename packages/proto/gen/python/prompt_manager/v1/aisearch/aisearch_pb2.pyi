from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SearchSkillsRequest(_message.Message):
    __slots__ = ("query", "queries", "limit", "output", "format", "render_limit")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    QUERIES_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    RENDER_LIMIT_FIELD_NUMBER: _ClassVar[int]
    query: str
    queries: _containers.RepeatedScalarFieldContainer[str]
    limit: int
    output: str
    format: str
    render_limit: int
    def __init__(self, query: _Optional[str] = ..., queries: _Optional[_Iterable[str]] = ..., limit: _Optional[int] = ..., output: _Optional[str] = ..., format: _Optional[str] = ..., render_limit: _Optional[int] = ...) -> None: ...

class SkillResult(_message.Message):
    __slots__ = ("id", "name", "description", "folder", "tags", "modes", "score", "score_percent")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    FOLDER_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    MODES_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    SCORE_PERCENT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    folder: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    modes: _containers.RepeatedScalarFieldContainer[str]
    score: float
    score_percent: int
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., folder: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., modes: _Optional[_Iterable[str]] = ..., score: _Optional[float] = ..., score_percent: _Optional[int] = ...) -> None: ...

class SearchSkillsResponse(_message.Message):
    __slots__ = ("results", "combined", "skill_count", "total_tokens", "format", "total", "query", "method", "output")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    COMBINED_FIELD_NUMBER: _ClassVar[int]
    SKILL_COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_TOKENS_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[SkillResult]
    combined: str
    skill_count: int
    total_tokens: int
    format: str
    total: int
    query: str
    method: str
    output: str
    def __init__(self, results: _Optional[_Iterable[_Union[SkillResult, _Mapping]]] = ..., combined: _Optional[str] = ..., skill_count: _Optional[int] = ..., total_tokens: _Optional[int] = ..., format: _Optional[str] = ..., total: _Optional[int] = ..., query: _Optional[str] = ..., method: _Optional[str] = ..., output: _Optional[str] = ...) -> None: ...

class SearchAgentsRequest(_message.Message):
    __slots__ = ("query", "limit")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    query: str
    limit: int
    def __init__(self, query: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class AgentResult(_message.Message):
    __slots__ = ("id", "display_name", "description", "status", "tags", "score", "score_percent")
    ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    SCORE_PERCENT_FIELD_NUMBER: _ClassVar[int]
    id: str
    display_name: str
    description: str
    status: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    score: float
    score_percent: int
    def __init__(self, id: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., score: _Optional[float] = ..., score_percent: _Optional[int] = ...) -> None: ...

class SearchAgentsResponse(_message.Message):
    __slots__ = ("results", "total", "query", "method")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[AgentResult]
    total: int
    query: str
    method: str
    def __init__(self, results: _Optional[_Iterable[_Union[AgentResult, _Mapping]]] = ..., total: _Optional[int] = ..., query: _Optional[str] = ..., method: _Optional[str] = ...) -> None: ...

class SearchActionsRequest(_message.Message):
    __slots__ = ("query", "limit")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    query: str
    limit: int
    def __init__(self, query: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ActionResult(_message.Message):
    __slots__ = ("id", "name", "description", "status", "owner", "command", "tags", "score", "score_percent")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    SCORE_PERCENT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    status: str
    owner: str
    command: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    score: float
    score_percent: int
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ..., owner: _Optional[str] = ..., command: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., score: _Optional[float] = ..., score_percent: _Optional[int] = ...) -> None: ...

class SearchActionsResponse(_message.Message):
    __slots__ = ("results", "total", "query", "method")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[ActionResult]
    total: int
    query: str
    method: str
    def __init__(self, results: _Optional[_Iterable[_Union[ActionResult, _Mapping]]] = ..., total: _Optional[int] = ..., query: _Optional[str] = ..., method: _Optional[str] = ...) -> None: ...

class SearchTeamsRequest(_message.Message):
    __slots__ = ("query", "limit")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    query: str
    limit: int
    def __init__(self, query: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class TeamResult(_message.Message):
    __slots__ = ("id", "display_name", "mission", "enabled", "member_count", "score", "score_percent")
    ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    MISSION_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    MEMBER_COUNT_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    SCORE_PERCENT_FIELD_NUMBER: _ClassVar[int]
    id: str
    display_name: str
    mission: str
    enabled: bool
    member_count: int
    score: float
    score_percent: int
    def __init__(self, id: _Optional[str] = ..., display_name: _Optional[str] = ..., mission: _Optional[str] = ..., enabled: _Optional[bool] = ..., member_count: _Optional[int] = ..., score: _Optional[float] = ..., score_percent: _Optional[int] = ...) -> None: ...

class SearchTeamsResponse(_message.Message):
    __slots__ = ("results", "total", "query", "method")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[TeamResult]
    total: int
    query: str
    method: str
    def __init__(self, results: _Optional[_Iterable[_Union[TeamResult, _Mapping]]] = ..., total: _Optional[int] = ..., query: _Optional[str] = ..., method: _Optional[str] = ...) -> None: ...

class GetStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetStatusResponse(_message.Message):
    __slots__ = ("available", "ollama", "qdrant", "indexed_count", "message")
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    OLLAMA_FIELD_NUMBER: _ClassVar[int]
    QDRANT_FIELD_NUMBER: _ClassVar[int]
    INDEXED_COUNT_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    available: bool
    ollama: bool
    qdrant: bool
    indexed_count: int
    message: str
    def __init__(self, available: _Optional[bool] = ..., ollama: _Optional[bool] = ..., qdrant: _Optional[bool] = ..., indexed_count: _Optional[int] = ..., message: _Optional[str] = ...) -> None: ...

class ItemRef(_message.Message):
    __slots__ = ("kind", "point_id", "name", "payload_hash")
    KIND_FIELD_NUMBER: _ClassVar[int]
    POINT_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_HASH_FIELD_NUMBER: _ClassVar[int]
    kind: str
    point_id: str
    name: str
    payload_hash: str
    def __init__(self, kind: _Optional[str] = ..., point_id: _Optional[str] = ..., name: _Optional[str] = ..., payload_hash: _Optional[str] = ...) -> None: ...

class CollectionDriftReport(_message.Message):
    __slots__ = ("kind", "to_upsert", "to_delete", "unchanged_count", "legacy_count")
    KIND_FIELD_NUMBER: _ClassVar[int]
    TO_UPSERT_FIELD_NUMBER: _ClassVar[int]
    TO_DELETE_FIELD_NUMBER: _ClassVar[int]
    UNCHANGED_COUNT_FIELD_NUMBER: _ClassVar[int]
    LEGACY_COUNT_FIELD_NUMBER: _ClassVar[int]
    kind: str
    to_upsert: _containers.RepeatedCompositeFieldContainer[ItemRef]
    to_delete: _containers.RepeatedScalarFieldContainer[str]
    unchanged_count: int
    legacy_count: int
    def __init__(self, kind: _Optional[str] = ..., to_upsert: _Optional[_Iterable[_Union[ItemRef, _Mapping]]] = ..., to_delete: _Optional[_Iterable[str]] = ..., unchanged_count: _Optional[int] = ..., legacy_count: _Optional[int] = ...) -> None: ...

class DriftReport(_message.Message):
    __slots__ = ("planned_at", "collections")
    PLANNED_AT_FIELD_NUMBER: _ClassVar[int]
    COLLECTIONS_FIELD_NUMBER: _ClassVar[int]
    planned_at: str
    collections: _containers.RepeatedCompositeFieldContainer[CollectionDriftReport]
    def __init__(self, planned_at: _Optional[str] = ..., collections: _Optional[_Iterable[_Union[CollectionDriftReport, _Mapping]]] = ...) -> None: ...

class CollectionApplyResult(_message.Message):
    __slots__ = ("kind", "upserted", "deleted")
    KIND_FIELD_NUMBER: _ClassVar[int]
    UPSERTED_FIELD_NUMBER: _ClassVar[int]
    DELETED_FIELD_NUMBER: _ClassVar[int]
    kind: str
    upserted: int
    deleted: int
    def __init__(self, kind: _Optional[str] = ..., upserted: _Optional[int] = ..., deleted: _Optional[int] = ...) -> None: ...

class ReconcileError(_message.Message):
    __slots__ = ("kind", "point_id", "name", "op", "err")
    KIND_FIELD_NUMBER: _ClassVar[int]
    POINT_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    OP_FIELD_NUMBER: _ClassVar[int]
    ERR_FIELD_NUMBER: _ClassVar[int]
    kind: str
    point_id: str
    name: str
    op: str
    err: str
    def __init__(self, kind: _Optional[str] = ..., point_id: _Optional[str] = ..., name: _Optional[str] = ..., op: _Optional[str] = ..., err: _Optional[str] = ...) -> None: ...

class ApplyResult(_message.Message):
    __slots__ = ("started_at", "finished_at", "collections", "errors")
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    COLLECTIONS_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    started_at: str
    finished_at: str
    collections: _containers.RepeatedCompositeFieldContainer[CollectionApplyResult]
    errors: _containers.RepeatedCompositeFieldContainer[ReconcileError]
    def __init__(self, started_at: _Optional[str] = ..., finished_at: _Optional[str] = ..., collections: _Optional[_Iterable[_Union[CollectionApplyResult, _Mapping]]] = ..., errors: _Optional[_Iterable[_Union[ReconcileError, _Mapping]]] = ...) -> None: ...

class ReconcileStatus(_message.Message):
    __slots__ = ("running", "started_at", "finished_at", "last_plan", "last_result", "last_error", "canceled")
    RUNNING_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_PLAN_FIELD_NUMBER: _ClassVar[int]
    LAST_RESULT_FIELD_NUMBER: _ClassVar[int]
    LAST_ERROR_FIELD_NUMBER: _ClassVar[int]
    CANCELED_FIELD_NUMBER: _ClassVar[int]
    running: bool
    started_at: str
    finished_at: str
    last_plan: DriftReport
    last_result: ApplyResult
    last_error: str
    canceled: bool
    def __init__(self, running: _Optional[bool] = ..., started_at: _Optional[str] = ..., finished_at: _Optional[str] = ..., last_plan: _Optional[_Union[DriftReport, _Mapping]] = ..., last_result: _Optional[_Union[ApplyResult, _Mapping]] = ..., last_error: _Optional[str] = ..., canceled: _Optional[bool] = ...) -> None: ...

class ReconcileRequest(_message.Message):
    __slots__ = ("collection", "dry_run")
    COLLECTION_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    collection: str
    dry_run: bool
    def __init__(self, collection: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class ReconcileResponse(_message.Message):
    __slots__ = ("dry_run", "plan", "status")
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    PLAN_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    plan: DriftReport
    status: ReconcileStatus
    def __init__(self, dry_run: _Optional[bool] = ..., plan: _Optional[_Union[DriftReport, _Mapping]] = ..., status: _Optional[_Union[ReconcileStatus, _Mapping]] = ...) -> None: ...

class GetReconcileStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class CancelReconcileRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
