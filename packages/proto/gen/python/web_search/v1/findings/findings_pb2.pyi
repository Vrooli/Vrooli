import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from measures.v1 import measures_pb2 as _measures_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class FindingStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FINDING_STATUS_UNSPECIFIED: _ClassVar[FindingStatus]
    FINDING_STATUS_ACTIVE: _ClassVar[FindingStatus]
    FINDING_STATUS_DISPUTED: _ClassVar[FindingStatus]
    FINDING_STATUS_SUPERSEDED: _ClassVar[FindingStatus]

class FindingSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FINDING_SOURCE_UNSPECIFIED: _ClassVar[FindingSource]
    FINDING_SOURCE_MANUAL: _ClassVar[FindingSource]
    FINDING_SOURCE_L2: _ClassVar[FindingSource]
    FINDING_SOURCE_L3: _ClassVar[FindingSource]
FINDING_STATUS_UNSPECIFIED: FindingStatus
FINDING_STATUS_ACTIVE: FindingStatus
FINDING_STATUS_DISPUTED: FindingStatus
FINDING_STATUS_SUPERSEDED: FindingStatus
FINDING_SOURCE_UNSPECIFIED: FindingSource
FINDING_SOURCE_MANUAL: FindingSource
FINDING_SOURCE_L2: FindingSource
FINDING_SOURCE_L3: FindingSource

class Citation(_message.Message):
    __slots__ = ("id", "url", "title", "retrieved_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    RETRIEVED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    url: str
    title: str
    retrieved_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., url: _Optional[str] = ..., title: _Optional[str] = ..., retrieved_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CitationInput(_message.Message):
    __slots__ = ("url", "title")
    URL_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    url: str
    title: str
    def __init__(self, url: _Optional[str] = ..., title: _Optional[str] = ...) -> None: ...

class Brief(_message.Message):
    __slots__ = ("id", "query", "level", "summary", "agent_run_id", "run_timestamp", "metadata")
    ID_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    AGENT_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    id: str
    query: str
    level: str
    summary: str
    agent_run_id: str
    run_timestamp: _timestamp_pb2.Timestamp
    metadata: str
    def __init__(self, id: _Optional[str] = ..., query: _Optional[str] = ..., level: _Optional[str] = ..., summary: _Optional[str] = ..., agent_run_id: _Optional[str] = ..., run_timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., metadata: _Optional[str] = ...) -> None: ...

class Finding(_message.Message):
    __slots__ = ("id", "claim", "brief_id", "confidence", "status", "retrieval_date", "query", "superseded_by", "dispute_note", "source", "citations", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    CLAIM_FIELD_NUMBER: _ClassVar[int]
    BRIEF_ID_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    RETRIEVAL_DATE_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDED_BY_FIELD_NUMBER: _ClassVar[int]
    DISPUTE_NOTE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CITATIONS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    claim: str
    brief_id: str
    confidence: float
    status: FindingStatus
    retrieval_date: _timestamp_pb2.Timestamp
    query: str
    superseded_by: str
    dispute_note: str
    source: FindingSource
    citations: _containers.RepeatedCompositeFieldContainer[Citation]
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., claim: _Optional[str] = ..., brief_id: _Optional[str] = ..., confidence: _Optional[float] = ..., status: _Optional[_Union[FindingStatus, str]] = ..., retrieval_date: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., query: _Optional[str] = ..., superseded_by: _Optional[str] = ..., dispute_note: _Optional[str] = ..., source: _Optional[_Union[FindingSource, str]] = ..., citations: _Optional[_Iterable[_Union[Citation, _Mapping]]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListFindingsRequest(_message.Message):
    __slots__ = ("status", "include_archived", "limit")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_ARCHIVED_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    status: FindingStatus
    include_archived: bool
    limit: int
    def __init__(self, status: _Optional[_Union[FindingStatus, str]] = ..., include_archived: _Optional[bool] = ..., limit: _Optional[int] = ...) -> None: ...

class ListFindingsResponse(_message.Message):
    __slots__ = ("findings",)
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    findings: _containers.RepeatedCompositeFieldContainer[Finding]
    def __init__(self, findings: _Optional[_Iterable[_Union[Finding, _Mapping]]] = ...) -> None: ...

class GetFindingRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetFindingResponse(_message.Message):
    __slots__ = ("finding",)
    FINDING_FIELD_NUMBER: _ClassVar[int]
    finding: Finding
    def __init__(self, finding: _Optional[_Union[Finding, _Mapping]] = ...) -> None: ...

class AddFindingRequest(_message.Message):
    __slots__ = ("claim", "confidence", "query", "source", "brief_id", "citations")
    CLAIM_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    BRIEF_ID_FIELD_NUMBER: _ClassVar[int]
    CITATIONS_FIELD_NUMBER: _ClassVar[int]
    claim: str
    confidence: float
    query: str
    source: FindingSource
    brief_id: str
    citations: _containers.RepeatedCompositeFieldContainer[CitationInput]
    def __init__(self, claim: _Optional[str] = ..., confidence: _Optional[float] = ..., query: _Optional[str] = ..., source: _Optional[_Union[FindingSource, str]] = ..., brief_id: _Optional[str] = ..., citations: _Optional[_Iterable[_Union[CitationInput, _Mapping]]] = ...) -> None: ...

class AddFindingResponse(_message.Message):
    __slots__ = ("finding",)
    FINDING_FIELD_NUMBER: _ClassVar[int]
    finding: Finding
    def __init__(self, finding: _Optional[_Union[Finding, _Mapping]] = ...) -> None: ...

class EditFindingRequest(_message.Message):
    __slots__ = ("id", "claim", "confidence")
    ID_FIELD_NUMBER: _ClassVar[int]
    CLAIM_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    claim: str
    confidence: float
    def __init__(self, id: _Optional[str] = ..., claim: _Optional[str] = ..., confidence: _Optional[float] = ...) -> None: ...

class EditFindingResponse(_message.Message):
    __slots__ = ("finding",)
    FINDING_FIELD_NUMBER: _ClassVar[int]
    finding: Finding
    def __init__(self, finding: _Optional[_Union[Finding, _Mapping]] = ...) -> None: ...

class SupersedeFindingRequest(_message.Message):
    __slots__ = ("id", "replacement", "reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    REPLACEMENT_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    replacement: str
    reason: str
    def __init__(self, id: _Optional[str] = ..., replacement: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class SupersedeFindingResponse(_message.Message):
    __slots__ = ("finding",)
    FINDING_FIELD_NUMBER: _ClassVar[int]
    finding: Finding
    def __init__(self, finding: _Optional[_Union[Finding, _Mapping]] = ...) -> None: ...

class FlagFindingRequest(_message.Message):
    __slots__ = ("id", "reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    reason: str
    def __init__(self, id: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class FlagFindingResponse(_message.Message):
    __slots__ = ("finding",)
    FINDING_FIELD_NUMBER: _ClassVar[int]
    finding: Finding
    def __init__(self, finding: _Optional[_Union[Finding, _Mapping]] = ...) -> None: ...

class ListDisputesRequest(_message.Message):
    __slots__ = ("limit",)
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    def __init__(self, limit: _Optional[int] = ...) -> None: ...

class ListDisputesResponse(_message.Message):
    __slots__ = ("findings",)
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    findings: _containers.RepeatedCompositeFieldContainer[Finding]
    def __init__(self, findings: _Optional[_Iterable[_Union[Finding, _Mapping]]] = ...) -> None: ...

class ResolveDisputeRequest(_message.Message):
    __slots__ = ("id", "resolution", "replacement", "reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    RESOLUTION_FIELD_NUMBER: _ClassVar[int]
    REPLACEMENT_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    resolution: str
    replacement: str
    reason: str
    def __init__(self, id: _Optional[str] = ..., resolution: _Optional[str] = ..., replacement: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class ResolveDisputeResponse(_message.Message):
    __slots__ = ("finding",)
    FINDING_FIELD_NUMBER: _ClassVar[int]
    finding: Finding
    def __init__(self, finding: _Optional[_Union[Finding, _Mapping]] = ...) -> None: ...

class PruneFindingsRequest(_message.Message):
    __slots__ = ("dry_run",)
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    def __init__(self, dry_run: _Optional[bool] = ...) -> None: ...

class PruneFindingsResponse(_message.Message):
    __slots__ = ("pruned", "finding_ids")
    PRUNED_FIELD_NUMBER: _ClassVar[int]
    FINDING_IDS_FIELD_NUMBER: _ClassVar[int]
    pruned: int
    finding_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, pruned: _Optional[int] = ..., finding_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class SearchFindingsRequest(_message.Message):
    __slots__ = ("query", "limit", "include_archived")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_ARCHIVED_FIELD_NUMBER: _ClassVar[int]
    query: str
    limit: int
    include_archived: bool
    def __init__(self, query: _Optional[str] = ..., limit: _Optional[int] = ..., include_archived: _Optional[bool] = ...) -> None: ...

class FindingHit(_message.Message):
    __slots__ = ("finding", "score", "weak")
    FINDING_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    WEAK_FIELD_NUMBER: _ClassVar[int]
    finding: Finding
    score: float
    weak: bool
    def __init__(self, finding: _Optional[_Union[Finding, _Mapping]] = ..., score: _Optional[float] = ..., weak: _Optional[bool] = ...) -> None: ...

class SearchFindingsResponse(_message.Message):
    __slots__ = ("hits", "method")
    HITS_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    hits: _containers.RepeatedCompositeFieldContainer[FindingHit]
    method: str
    def __init__(self, hits: _Optional[_Iterable[_Union[FindingHit, _Mapping]]] = ..., method: _Optional[str] = ...) -> None: ...

class CountFindingsRequest(_message.Message):
    __slots__ = ("window",)
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ...) -> None: ...

class CountFindingsResponse(_message.Message):
    __slots__ = ("count",)
    COUNT_FIELD_NUMBER: _ClassVar[int]
    count: int
    def __init__(self, count: _Optional[int] = ...) -> None: ...

class FindingEffectiveness(_message.Message):
    __slots__ = ("finding", "surfaced_count", "used_count", "last_surfaced_at", "effective_confidence", "usage_factor", "effective_score")
    FINDING_FIELD_NUMBER: _ClassVar[int]
    SURFACED_COUNT_FIELD_NUMBER: _ClassVar[int]
    USED_COUNT_FIELD_NUMBER: _ClassVar[int]
    LAST_SURFACED_AT_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    USAGE_FACTOR_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_SCORE_FIELD_NUMBER: _ClassVar[int]
    finding: Finding
    surfaced_count: int
    used_count: int
    last_surfaced_at: _timestamp_pb2.Timestamp
    effective_confidence: float
    usage_factor: float
    effective_score: float
    def __init__(self, finding: _Optional[_Union[Finding, _Mapping]] = ..., surfaced_count: _Optional[int] = ..., used_count: _Optional[int] = ..., last_surfaced_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., effective_confidence: _Optional[float] = ..., usage_factor: _Optional[float] = ..., effective_score: _Optional[float] = ...) -> None: ...

class ListEffectivenessRequest(_message.Message):
    __slots__ = ("limit", "include_disputed")
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_DISPUTED_FIELD_NUMBER: _ClassVar[int]
    limit: int
    include_disputed: bool
    def __init__(self, limit: _Optional[int] = ..., include_disputed: _Optional[bool] = ...) -> None: ...

class UsageMeasureRequest(_message.Message):
    __slots__ = ("window",)
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ...) -> None: ...

class UsageRateResponse(_message.Message):
    __slots__ = ("rate",)
    RATE_FIELD_NUMBER: _ClassVar[int]
    rate: float
    def __init__(self, rate: _Optional[float] = ...) -> None: ...

class UsageCountResponse(_message.Message):
    __slots__ = ("count",)
    COUNT_FIELD_NUMBER: _ClassVar[int]
    count: int
    def __init__(self, count: _Optional[int] = ...) -> None: ...

class ListEffectivenessResponse(_message.Message):
    __slots__ = ("items",)
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[FindingEffectiveness]
    def __init__(self, items: _Optional[_Iterable[_Union[FindingEffectiveness, _Mapping]]] = ...) -> None: ...

class RecordUsageRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RecordUsageResponse(_message.Message):
    __slots__ = ("finding",)
    FINDING_FIELD_NUMBER: _ClassVar[int]
    finding: Finding
    def __init__(self, finding: _Optional[_Union[Finding, _Mapping]] = ...) -> None: ...

class RunGCRequest(_message.Message):
    __slots__ = ("dry_run",)
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    def __init__(self, dry_run: _Optional[bool] = ...) -> None: ...

class RunGCResponse(_message.Message):
    __slots__ = ("dry_run", "superseded_decayed", "cold_archive_candidates", "stale_disputes", "orphans")
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDED_DECAYED_FIELD_NUMBER: _ClassVar[int]
    COLD_ARCHIVE_CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    STALE_DISPUTES_FIELD_NUMBER: _ClassVar[int]
    ORPHANS_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    superseded_decayed: _containers.RepeatedScalarFieldContainer[str]
    cold_archive_candidates: _containers.RepeatedScalarFieldContainer[str]
    stale_disputes: _containers.RepeatedScalarFieldContainer[str]
    orphans: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, dry_run: _Optional[bool] = ..., superseded_decayed: _Optional[_Iterable[str]] = ..., cold_archive_candidates: _Optional[_Iterable[str]] = ..., stale_disputes: _Optional[_Iterable[str]] = ..., orphans: _Optional[_Iterable[str]] = ...) -> None: ...
