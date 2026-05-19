from development_toolchain_validator.v1.validation_record import validation_record_pb2 as _validation_record_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TupleVerdict(_message.Message):
    __slots__ = ("tuple_kind", "subject_id", "latest_verdict", "latest_record_id", "stale")
    TUPLE_KIND_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_ID_FIELD_NUMBER: _ClassVar[int]
    LATEST_VERDICT_FIELD_NUMBER: _ClassVar[int]
    LATEST_RECORD_ID_FIELD_NUMBER: _ClassVar[int]
    STALE_FIELD_NUMBER: _ClassVar[int]
    tuple_kind: _validation_record_pb2.TupleKind
    subject_id: str
    latest_verdict: _validation_record_pb2.Verdict
    latest_record_id: str
    stale: bool
    def __init__(self, tuple_kind: _Optional[_Union[_validation_record_pb2.TupleKind, str]] = ..., subject_id: _Optional[str] = ..., latest_verdict: _Optional[_Union[_validation_record_pb2.Verdict, str]] = ..., latest_record_id: _Optional[str] = ..., stale: _Optional[bool] = ...) -> None: ...

class GoldenSummary(_message.Message):
    __slots__ = ("golden_slug", "skill_verdicts", "tool_verdicts", "stale_count")
    GOLDEN_SLUG_FIELD_NUMBER: _ClassVar[int]
    SKILL_VERDICTS_FIELD_NUMBER: _ClassVar[int]
    TOOL_VERDICTS_FIELD_NUMBER: _ClassVar[int]
    STALE_COUNT_FIELD_NUMBER: _ClassVar[int]
    golden_slug: str
    skill_verdicts: _containers.RepeatedCompositeFieldContainer[TupleVerdict]
    tool_verdicts: _containers.RepeatedCompositeFieldContainer[TupleVerdict]
    stale_count: int
    def __init__(self, golden_slug: _Optional[str] = ..., skill_verdicts: _Optional[_Iterable[_Union[TupleVerdict, _Mapping]]] = ..., tool_verdicts: _Optional[_Iterable[_Union[TupleVerdict, _Mapping]]] = ..., stale_count: _Optional[int] = ...) -> None: ...

class GetGoldenSummaryRequest(_message.Message):
    __slots__ = ("golden_slug",)
    GOLDEN_SLUG_FIELD_NUMBER: _ClassVar[int]
    golden_slug: str
    def __init__(self, golden_slug: _Optional[str] = ...) -> None: ...

class GetGoldenSummaryResponse(_message.Message):
    __slots__ = ("summary",)
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    summary: GoldenSummary
    def __init__(self, summary: _Optional[_Union[GoldenSummary, _Mapping]] = ...) -> None: ...

class TupleHistory(_message.Message):
    __slots__ = ("tuple_kind", "subject_id", "golden_slug", "records", "next_page_token")
    TUPLE_KIND_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_ID_FIELD_NUMBER: _ClassVar[int]
    GOLDEN_SLUG_FIELD_NUMBER: _ClassVar[int]
    RECORDS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    tuple_kind: _validation_record_pb2.TupleKind
    subject_id: str
    golden_slug: str
    records: _containers.RepeatedCompositeFieldContainer[_validation_record_pb2.ValidationRecord]
    next_page_token: str
    def __init__(self, tuple_kind: _Optional[_Union[_validation_record_pb2.TupleKind, str]] = ..., subject_id: _Optional[str] = ..., golden_slug: _Optional[str] = ..., records: _Optional[_Iterable[_Union[_validation_record_pb2.ValidationRecord, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class GetTupleHistoryRequest(_message.Message):
    __slots__ = ("tuple_kind", "subject_id", "golden_slug", "page_size", "page_token")
    TUPLE_KIND_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_ID_FIELD_NUMBER: _ClassVar[int]
    GOLDEN_SLUG_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    tuple_kind: _validation_record_pb2.TupleKind
    subject_id: str
    golden_slug: str
    page_size: int
    page_token: str
    def __init__(self, tuple_kind: _Optional[_Union[_validation_record_pb2.TupleKind, str]] = ..., subject_id: _Optional[str] = ..., golden_slug: _Optional[str] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class GetTupleHistoryResponse(_message.Message):
    __slots__ = ("history",)
    HISTORY_FIELD_NUMBER: _ClassVar[int]
    history: TupleHistory
    def __init__(self, history: _Optional[_Union[TupleHistory, _Mapping]] = ...) -> None: ...

class CoverageRow(_message.Message):
    __slots__ = ("tuple_kind", "subject_id", "verdict", "stale", "has_manifest")
    TUPLE_KIND_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_ID_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    STALE_FIELD_NUMBER: _ClassVar[int]
    HAS_MANIFEST_FIELD_NUMBER: _ClassVar[int]
    tuple_kind: _validation_record_pb2.TupleKind
    subject_id: str
    verdict: _validation_record_pb2.Verdict
    stale: bool
    has_manifest: bool
    def __init__(self, tuple_kind: _Optional[_Union[_validation_record_pb2.TupleKind, str]] = ..., subject_id: _Optional[str] = ..., verdict: _Optional[_Union[_validation_record_pb2.Verdict, str]] = ..., stale: _Optional[bool] = ..., has_manifest: _Optional[bool] = ...) -> None: ...

class Coverage(_message.Message):
    __slots__ = ("golden_slug", "rows")
    GOLDEN_SLUG_FIELD_NUMBER: _ClassVar[int]
    ROWS_FIELD_NUMBER: _ClassVar[int]
    golden_slug: str
    rows: _containers.RepeatedCompositeFieldContainer[CoverageRow]
    def __init__(self, golden_slug: _Optional[str] = ..., rows: _Optional[_Iterable[_Union[CoverageRow, _Mapping]]] = ...) -> None: ...

class GetCoverageRequest(_message.Message):
    __slots__ = ("golden_slug",)
    GOLDEN_SLUG_FIELD_NUMBER: _ClassVar[int]
    golden_slug: str
    def __init__(self, golden_slug: _Optional[str] = ...) -> None: ...

class GetCoverageResponse(_message.Message):
    __slots__ = ("coverage",)
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    coverage: Coverage
    def __init__(self, coverage: _Optional[_Union[Coverage, _Mapping]] = ...) -> None: ...
