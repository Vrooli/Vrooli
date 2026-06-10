import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from web_search.v1.livesearch import livesearch_pb2 as _livesearch_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Citation(_message.Message):
    __slots__ = ("result_index", "url", "title")
    RESULT_INDEX_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    result_index: int
    url: str
    title: str
    def __init__(self, result_index: _Optional[int] = ..., url: _Optional[str] = ..., title: _Optional[str] = ...) -> None: ...

class Brief(_message.Message):
    __slots__ = ("query", "level", "summary", "citations")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    CITATIONS_FIELD_NUMBER: _ClassVar[int]
    query: str
    level: str
    summary: str
    citations: _containers.RepeatedCompositeFieldContainer[Citation]
    def __init__(self, query: _Optional[str] = ..., level: _Optional[str] = ..., summary: _Optional[str] = ..., citations: _Optional[_Iterable[_Union[Citation, _Mapping]]] = ...) -> None: ...

class RunL2Request(_message.Message):
    __slots__ = ("query", "top_n", "capture")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    TOP_N_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_FIELD_NUMBER: _ClassVar[int]
    query: str
    top_n: int
    capture: bool
    def __init__(self, query: _Optional[str] = ..., top_n: _Optional[int] = ..., capture: _Optional[bool] = ...) -> None: ...

class RunL2Response(_message.Message):
    __slots__ = ("brief", "synthesis", "abstained", "captured_finding_ids", "degraded_engines", "abstain_reason", "excerpts")
    BRIEF_FIELD_NUMBER: _ClassVar[int]
    SYNTHESIS_FIELD_NUMBER: _ClassVar[int]
    ABSTAINED_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_FINDING_IDS_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_ENGINES_FIELD_NUMBER: _ClassVar[int]
    ABSTAIN_REASON_FIELD_NUMBER: _ClassVar[int]
    EXCERPTS_FIELD_NUMBER: _ClassVar[int]
    brief: Brief
    synthesis: str
    abstained: bool
    captured_finding_ids: _containers.RepeatedScalarFieldContainer[str]
    degraded_engines: _containers.RepeatedCompositeFieldContainer[_livesearch_pb2.EngineIssue]
    abstain_reason: str
    excerpts: _containers.RepeatedCompositeFieldContainer[DocumentExcerpt]
    def __init__(self, brief: _Optional[_Union[Brief, _Mapping]] = ..., synthesis: _Optional[str] = ..., abstained: _Optional[bool] = ..., captured_finding_ids: _Optional[_Iterable[str]] = ..., degraded_engines: _Optional[_Iterable[_Union[_livesearch_pb2.EngineIssue, _Mapping]]] = ..., abstain_reason: _Optional[str] = ..., excerpts: _Optional[_Iterable[_Union[DocumentExcerpt, _Mapping]]] = ...) -> None: ...

class DocumentExcerpt(_message.Message):
    __slots__ = ("url", "title", "excerpt")
    URL_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    EXCERPT_FIELD_NUMBER: _ClassVar[int]
    url: str
    title: str
    excerpt: str
    def __init__(self, url: _Optional[str] = ..., title: _Optional[str] = ..., excerpt: _Optional[str] = ...) -> None: ...

class RunL3Request(_message.Message):
    __slots__ = ("query",)
    QUERY_FIELD_NUMBER: _ClassVar[int]
    query: str
    def __init__(self, query: _Optional[str] = ...) -> None: ...

class RunL3Response(_message.Message):
    __slots__ = ("run_id", "status")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    status: str
    def __init__(self, run_id: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class GetResearchStatusRequest(_message.Message):
    __slots__ = ("run_id",)
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    def __init__(self, run_id: _Optional[str] = ...) -> None: ...

class GetResearchStatusResponse(_message.Message):
    __slots__ = ("run_id", "status", "summary", "started_at", "finished_at", "error_msg")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    ERROR_MSG_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    status: str
    summary: str
    started_at: _timestamp_pb2.Timestamp
    finished_at: _timestamp_pb2.Timestamp
    error_msg: str
    def __init__(self, run_id: _Optional[str] = ..., status: _Optional[str] = ..., summary: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., finished_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., error_msg: _Optional[str] = ...) -> None: ...

class GatherRelatedFindingsRequest(_message.Message):
    __slots__ = ("query", "max")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    MAX_FIELD_NUMBER: _ClassVar[int]
    query: str
    max: int
    def __init__(self, query: _Optional[str] = ..., max: _Optional[int] = ...) -> None: ...

class GatheredFinding(_message.Message):
    __slots__ = ("finding_id", "claim", "confidence", "status", "score")
    FINDING_ID_FIELD_NUMBER: _ClassVar[int]
    CLAIM_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    finding_id: str
    claim: str
    confidence: float
    status: str
    score: float
    def __init__(self, finding_id: _Optional[str] = ..., claim: _Optional[str] = ..., confidence: _Optional[float] = ..., status: _Optional[str] = ..., score: _Optional[float] = ...) -> None: ...

class GatherRelatedFindingsResponse(_message.Message):
    __slots__ = ("findings", "cap_applied")
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    CAP_APPLIED_FIELD_NUMBER: _ClassVar[int]
    findings: _containers.RepeatedCompositeFieldContainer[GatheredFinding]
    cap_applied: int
    def __init__(self, findings: _Optional[_Iterable[_Union[GatheredFinding, _Mapping]]] = ..., cap_applied: _Optional[int] = ...) -> None: ...
