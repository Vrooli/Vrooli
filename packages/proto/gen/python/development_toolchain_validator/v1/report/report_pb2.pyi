from development_toolchain_validator.v1.validation_record import validation_record_pb2 as _validation_record_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SkillFitnessVerdict(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SKILL_FITNESS_VERDICT_UNSPECIFIED: _ClassVar[SkillFitnessVerdict]
    SKILL_FITNESS_VERDICT_UNKNOWN: _ClassVar[SkillFitnessVerdict]
    SKILL_FITNESS_VERDICT_GREEN: _ClassVar[SkillFitnessVerdict]
    SKILL_FITNESS_VERDICT_YELLOW: _ClassVar[SkillFitnessVerdict]
    SKILL_FITNESS_VERDICT_RED: _ClassVar[SkillFitnessVerdict]
SKILL_FITNESS_VERDICT_UNSPECIFIED: SkillFitnessVerdict
SKILL_FITNESS_VERDICT_UNKNOWN: SkillFitnessVerdict
SKILL_FITNESS_VERDICT_GREEN: SkillFitnessVerdict
SKILL_FITNESS_VERDICT_YELLOW: SkillFitnessVerdict
SKILL_FITNESS_VERDICT_RED: SkillFitnessVerdict

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

class GoldenSkillSnapshot(_message.Message):
    __slots__ = ("golden_slug", "latest_verdict", "stale", "run_count")
    GOLDEN_SLUG_FIELD_NUMBER: _ClassVar[int]
    LATEST_VERDICT_FIELD_NUMBER: _ClassVar[int]
    STALE_FIELD_NUMBER: _ClassVar[int]
    RUN_COUNT_FIELD_NUMBER: _ClassVar[int]
    golden_slug: str
    latest_verdict: _validation_record_pb2.Verdict
    stale: bool
    run_count: int
    def __init__(self, golden_slug: _Optional[str] = ..., latest_verdict: _Optional[_Union[_validation_record_pb2.Verdict, str]] = ..., stale: _Optional[bool] = ..., run_count: _Optional[int] = ...) -> None: ...

class SkillFitness(_message.Message):
    __slots__ = ("skill_id", "pass_count", "unexpected_mutation_count", "run_failure_count", "tool_failure_count", "total_runs", "pass_rate", "total_tokens", "avg_tokens", "total_cost_usd_micro", "avg_cost_usd_micro", "total_duration_ms", "avg_duration_ms", "unique_diff_hashes", "convergence_ratio", "latest_verdict", "any_stale", "verdict", "by_golden")
    class ByGoldenEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: GoldenSkillSnapshot
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[GoldenSkillSnapshot, _Mapping]] = ...) -> None: ...
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    PASS_COUNT_FIELD_NUMBER: _ClassVar[int]
    UNEXPECTED_MUTATION_COUNT_FIELD_NUMBER: _ClassVar[int]
    RUN_FAILURE_COUNT_FIELD_NUMBER: _ClassVar[int]
    TOOL_FAILURE_COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_RUNS_FIELD_NUMBER: _ClassVar[int]
    PASS_RATE_FIELD_NUMBER: _ClassVar[int]
    TOTAL_TOKENS_FIELD_NUMBER: _ClassVar[int]
    AVG_TOKENS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_COST_USD_MICRO_FIELD_NUMBER: _ClassVar[int]
    AVG_COST_USD_MICRO_FIELD_NUMBER: _ClassVar[int]
    TOTAL_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    AVG_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    UNIQUE_DIFF_HASHES_FIELD_NUMBER: _ClassVar[int]
    CONVERGENCE_RATIO_FIELD_NUMBER: _ClassVar[int]
    LATEST_VERDICT_FIELD_NUMBER: _ClassVar[int]
    ANY_STALE_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    BY_GOLDEN_FIELD_NUMBER: _ClassVar[int]
    skill_id: str
    pass_count: int
    unexpected_mutation_count: int
    run_failure_count: int
    tool_failure_count: int
    total_runs: int
    pass_rate: float
    total_tokens: int
    avg_tokens: float
    total_cost_usd_micro: int
    avg_cost_usd_micro: float
    total_duration_ms: int
    avg_duration_ms: float
    unique_diff_hashes: int
    convergence_ratio: float
    latest_verdict: _validation_record_pb2.Verdict
    any_stale: bool
    verdict: SkillFitnessVerdict
    by_golden: _containers.MessageMap[str, GoldenSkillSnapshot]
    def __init__(self, skill_id: _Optional[str] = ..., pass_count: _Optional[int] = ..., unexpected_mutation_count: _Optional[int] = ..., run_failure_count: _Optional[int] = ..., tool_failure_count: _Optional[int] = ..., total_runs: _Optional[int] = ..., pass_rate: _Optional[float] = ..., total_tokens: _Optional[int] = ..., avg_tokens: _Optional[float] = ..., total_cost_usd_micro: _Optional[int] = ..., avg_cost_usd_micro: _Optional[float] = ..., total_duration_ms: _Optional[int] = ..., avg_duration_ms: _Optional[float] = ..., unique_diff_hashes: _Optional[int] = ..., convergence_ratio: _Optional[float] = ..., latest_verdict: _Optional[_Union[_validation_record_pb2.Verdict, str]] = ..., any_stale: _Optional[bool] = ..., verdict: _Optional[_Union[SkillFitnessVerdict, str]] = ..., by_golden: _Optional[_Mapping[str, GoldenSkillSnapshot]] = ...) -> None: ...

class GetSkillFitnessRequest(_message.Message):
    __slots__ = ("skill_id",)
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    skill_id: str
    def __init__(self, skill_id: _Optional[str] = ...) -> None: ...

class GetSkillFitnessResponse(_message.Message):
    __slots__ = ("fitness",)
    FITNESS_FIELD_NUMBER: _ClassVar[int]
    fitness: SkillFitness
    def __init__(self, fitness: _Optional[_Union[SkillFitness, _Mapping]] = ...) -> None: ...
