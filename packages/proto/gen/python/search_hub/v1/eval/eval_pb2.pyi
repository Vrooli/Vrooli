from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EvalSuite(_message.Message):
    __slots__ = ("suite_id", "provider_id", "name", "description", "cases", "state", "created_at", "updated_at")
    SUITE_ID_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CASES_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    suite_id: str
    provider_id: str
    name: str
    description: str
    cases: _containers.RepeatedCompositeFieldContainer[EvalCase]
    state: str
    created_at: str
    updated_at: str
    def __init__(self, suite_id: _Optional[str] = ..., provider_id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., cases: _Optional[_Iterable[_Union[EvalCase, _Mapping]]] = ..., state: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class EvalCase(_message.Message):
    __slots__ = ("case_id", "query", "tags", "expect_ids", "expect_within_top_k", "expect_min_score", "expect_max_score", "expect_no_strong_hit", "note")
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    EXPECT_IDS_FIELD_NUMBER: _ClassVar[int]
    EXPECT_WITHIN_TOP_K_FIELD_NUMBER: _ClassVar[int]
    EXPECT_MIN_SCORE_FIELD_NUMBER: _ClassVar[int]
    EXPECT_MAX_SCORE_FIELD_NUMBER: _ClassVar[int]
    EXPECT_NO_STRONG_HIT_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    query: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    expect_ids: _containers.RepeatedScalarFieldContainer[str]
    expect_within_top_k: int
    expect_min_score: float
    expect_max_score: float
    expect_no_strong_hit: bool
    note: str
    def __init__(self, case_id: _Optional[str] = ..., query: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., expect_ids: _Optional[_Iterable[str]] = ..., expect_within_top_k: _Optional[int] = ..., expect_min_score: _Optional[float] = ..., expect_max_score: _Optional[float] = ..., expect_no_strong_hit: _Optional[bool] = ..., note: _Optional[str] = ...) -> None: ...

class EvalRun(_message.Message):
    __slots__ = ("run_id", "suite_id", "tag", "created_at", "config", "results", "aggregate")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SUITE_ID_FIELD_NUMBER: _ClassVar[int]
    TAG_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    AGGREGATE_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    suite_id: str
    tag: str
    created_at: str
    config: ConfigSnapshot
    results: _containers.RepeatedCompositeFieldContainer[CaseResult]
    aggregate: EvalAggregate
    def __init__(self, run_id: _Optional[str] = ..., suite_id: _Optional[str] = ..., tag: _Optional[str] = ..., created_at: _Optional[str] = ..., config: _Optional[_Union[ConfigSnapshot, _Mapping]] = ..., results: _Optional[_Iterable[_Union[CaseResult, _Mapping]]] = ..., aggregate: _Optional[_Union[EvalAggregate, _Mapping]] = ...) -> None: ...

class ConfigSnapshot(_message.Message):
    __slots__ = ("rerank_enabled", "reranker_leg", "embed_model", "indexed_count", "provider_note")
    RERANK_ENABLED_FIELD_NUMBER: _ClassVar[int]
    RERANKER_LEG_FIELD_NUMBER: _ClassVar[int]
    EMBED_MODEL_FIELD_NUMBER: _ClassVar[int]
    INDEXED_COUNT_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_NOTE_FIELD_NUMBER: _ClassVar[int]
    rerank_enabled: bool
    reranker_leg: str
    embed_model: str
    indexed_count: int
    provider_note: str
    def __init__(self, rerank_enabled: _Optional[bool] = ..., reranker_leg: _Optional[str] = ..., embed_model: _Optional[str] = ..., indexed_count: _Optional[int] = ..., provider_note: _Optional[str] = ...) -> None: ...

class CaseResult(_message.Message):
    __slots__ = ("case_id", "top", "expected_rank", "observed_top_score", "outcome")
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    TOP_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_RANK_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_TOP_SCORE_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    top: _containers.RepeatedCompositeFieldContainer[ScoredHit]
    expected_rank: int
    observed_top_score: float
    outcome: str
    def __init__(self, case_id: _Optional[str] = ..., top: _Optional[_Iterable[_Union[ScoredHit, _Mapping]]] = ..., expected_rank: _Optional[int] = ..., observed_top_score: _Optional[float] = ..., outcome: _Optional[str] = ...) -> None: ...

class ScoredHit(_message.Message):
    __slots__ = ("id", "title", "score")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    score: float
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., score: _Optional[float] = ...) -> None: ...

class EvalAggregate(_message.Message):
    __slots__ = ("cases", "met", "below", "mean_strong_top1", "max_gibberish_score", "latency_p95_ms")
    CASES_FIELD_NUMBER: _ClassVar[int]
    MET_FIELD_NUMBER: _ClassVar[int]
    BELOW_FIELD_NUMBER: _ClassVar[int]
    MEAN_STRONG_TOP1_FIELD_NUMBER: _ClassVar[int]
    MAX_GIBBERISH_SCORE_FIELD_NUMBER: _ClassVar[int]
    LATENCY_P95_MS_FIELD_NUMBER: _ClassVar[int]
    cases: int
    met: int
    below: int
    mean_strong_top1: float
    max_gibberish_score: float
    latency_p95_ms: int
    def __init__(self, cases: _Optional[int] = ..., met: _Optional[int] = ..., below: _Optional[int] = ..., mean_strong_top1: _Optional[float] = ..., max_gibberish_score: _Optional[float] = ..., latency_p95_ms: _Optional[int] = ...) -> None: ...

class RegisterSuiteRequest(_message.Message):
    __slots__ = ("suite",)
    SUITE_FIELD_NUMBER: _ClassVar[int]
    suite: EvalSuite
    def __init__(self, suite: _Optional[_Union[EvalSuite, _Mapping]] = ...) -> None: ...

class RegisterSuiteResponse(_message.Message):
    __slots__ = ("suite", "created")
    SUITE_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    suite: EvalSuite
    created: bool
    def __init__(self, suite: _Optional[_Union[EvalSuite, _Mapping]] = ..., created: _Optional[bool] = ...) -> None: ...

class ListSuitesRequest(_message.Message):
    __slots__ = ("provider_id",)
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    provider_id: str
    def __init__(self, provider_id: _Optional[str] = ...) -> None: ...

class ListSuitesResponse(_message.Message):
    __slots__ = ("suites",)
    SUITES_FIELD_NUMBER: _ClassVar[int]
    suites: _containers.RepeatedCompositeFieldContainer[EvalSuite]
    def __init__(self, suites: _Optional[_Iterable[_Union[EvalSuite, _Mapping]]] = ...) -> None: ...

class GetSuiteRequest(_message.Message):
    __slots__ = ("suite_id",)
    SUITE_ID_FIELD_NUMBER: _ClassVar[int]
    suite_id: str
    def __init__(self, suite_id: _Optional[str] = ...) -> None: ...

class GetSuiteResponse(_message.Message):
    __slots__ = ("suite",)
    SUITE_FIELD_NUMBER: _ClassVar[int]
    suite: EvalSuite
    def __init__(self, suite: _Optional[_Union[EvalSuite, _Mapping]] = ...) -> None: ...

class RunSuiteRequest(_message.Message):
    __slots__ = ("suite_id", "tag", "limit")
    SUITE_ID_FIELD_NUMBER: _ClassVar[int]
    TAG_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    suite_id: str
    tag: str
    limit: int
    def __init__(self, suite_id: _Optional[str] = ..., tag: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class RunSuiteResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: EvalRun
    def __init__(self, run: _Optional[_Union[EvalRun, _Mapping]] = ...) -> None: ...

class ListRunsRequest(_message.Message):
    __slots__ = ("suite_id", "tag", "limit")
    SUITE_ID_FIELD_NUMBER: _ClassVar[int]
    TAG_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    suite_id: str
    tag: str
    limit: int
    def __init__(self, suite_id: _Optional[str] = ..., tag: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListRunsResponse(_message.Message):
    __slots__ = ("runs",)
    RUNS_FIELD_NUMBER: _ClassVar[int]
    runs: _containers.RepeatedCompositeFieldContainer[EvalRun]
    def __init__(self, runs: _Optional[_Iterable[_Union[EvalRun, _Mapping]]] = ...) -> None: ...

class GetRunRequest(_message.Message):
    __slots__ = ("run_id",)
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    def __init__(self, run_id: _Optional[str] = ...) -> None: ...

class GetRunResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: EvalRun
    def __init__(self, run: _Optional[_Union[EvalRun, _Mapping]] = ...) -> None: ...

class CompareRunsRequest(_message.Message):
    __slots__ = ("run_a", "run_b")
    RUN_A_FIELD_NUMBER: _ClassVar[int]
    RUN_B_FIELD_NUMBER: _ClassVar[int]
    run_a: str
    run_b: str
    def __init__(self, run_a: _Optional[str] = ..., run_b: _Optional[str] = ...) -> None: ...

class CompareRunsResponse(_message.Message):
    __slots__ = ("run_a", "run_b", "deltas")
    RUN_A_FIELD_NUMBER: _ClassVar[int]
    RUN_B_FIELD_NUMBER: _ClassVar[int]
    DELTAS_FIELD_NUMBER: _ClassVar[int]
    run_a: EvalRun
    run_b: EvalRun
    deltas: _containers.RepeatedCompositeFieldContainer[CaseDelta]
    def __init__(self, run_a: _Optional[_Union[EvalRun, _Mapping]] = ..., run_b: _Optional[_Union[EvalRun, _Mapping]] = ..., deltas: _Optional[_Iterable[_Union[CaseDelta, _Mapping]]] = ...) -> None: ...

class CaseDelta(_message.Message):
    __slots__ = ("case_id", "outcome_a", "outcome_b", "top_score_a", "top_score_b", "expected_rank_a", "expected_rank_b")
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_A_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_B_FIELD_NUMBER: _ClassVar[int]
    TOP_SCORE_A_FIELD_NUMBER: _ClassVar[int]
    TOP_SCORE_B_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_RANK_A_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_RANK_B_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    outcome_a: str
    outcome_b: str
    top_score_a: float
    top_score_b: float
    expected_rank_a: int
    expected_rank_b: int
    def __init__(self, case_id: _Optional[str] = ..., outcome_a: _Optional[str] = ..., outcome_b: _Optional[str] = ..., top_score_a: _Optional[float] = ..., top_score_b: _Optional[float] = ..., expected_rank_a: _Optional[int] = ..., expected_rank_b: _Optional[int] = ...) -> None: ...
