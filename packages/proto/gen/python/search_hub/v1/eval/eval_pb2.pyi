from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ReferentialOutcome(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    REFERENTIAL_OUTCOME_UNSPECIFIED: _ClassVar[ReferentialOutcome]
    REFERENTIAL_OUTCOME_LIVE: _ClassVar[ReferentialOutcome]
    REFERENTIAL_OUTCOME_HARD: _ClassVar[ReferentialOutcome]
    REFERENTIAL_OUTCOME_STALE: _ClassVar[ReferentialOutcome]
    REFERENTIAL_OUTCOME_INCONCLUSIVE: _ClassVar[ReferentialOutcome]
    REFERENTIAL_OUTCOME_PROVIDER_ERROR: _ClassVar[ReferentialOutcome]
REFERENTIAL_OUTCOME_UNSPECIFIED: ReferentialOutcome
REFERENTIAL_OUTCOME_LIVE: ReferentialOutcome
REFERENTIAL_OUTCOME_HARD: ReferentialOutcome
REFERENTIAL_OUTCOME_STALE: ReferentialOutcome
REFERENTIAL_OUTCOME_INCONCLUSIVE: ReferentialOutcome
REFERENTIAL_OUTCOME_PROVIDER_ERROR: ReferentialOutcome

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
    __slots__ = ("case_id", "query", "tags", "expect_ids", "expect_within_top_k", "expect_min_score", "expect_max_score", "expect_no_strong_hit", "note", "status", "scope", "expect_min_margin")
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    EXPECT_IDS_FIELD_NUMBER: _ClassVar[int]
    EXPECT_WITHIN_TOP_K_FIELD_NUMBER: _ClassVar[int]
    EXPECT_MIN_SCORE_FIELD_NUMBER: _ClassVar[int]
    EXPECT_MAX_SCORE_FIELD_NUMBER: _ClassVar[int]
    EXPECT_NO_STRONG_HIT_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    EXPECT_MIN_MARGIN_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    query: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    expect_ids: _containers.RepeatedScalarFieldContainer[str]
    expect_within_top_k: int
    expect_min_score: float
    expect_max_score: float
    expect_no_strong_hit: bool
    note: str
    status: str
    scope: str
    expect_min_margin: float
    def __init__(self, case_id: _Optional[str] = ..., query: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., expect_ids: _Optional[_Iterable[str]] = ..., expect_within_top_k: _Optional[int] = ..., expect_min_score: _Optional[float] = ..., expect_max_score: _Optional[float] = ..., expect_no_strong_hit: _Optional[bool] = ..., note: _Optional[str] = ..., status: _Optional[str] = ..., scope: _Optional[str] = ..., expect_min_margin: _Optional[float] = ...) -> None: ...

class EvalRun(_message.Message):
    __slots__ = ("run_id", "suite_id", "tag", "created_at", "config", "results", "aggregate", "tier", "degraded", "degraded_reason")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SUITE_ID_FIELD_NUMBER: _ClassVar[int]
    TAG_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    AGGREGATE_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASON_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    suite_id: str
    tag: str
    created_at: str
    config: ConfigSnapshot
    results: _containers.RepeatedCompositeFieldContainer[CaseResult]
    aggregate: EvalAggregate
    tier: str
    degraded: bool
    degraded_reason: str
    def __init__(self, run_id: _Optional[str] = ..., suite_id: _Optional[str] = ..., tag: _Optional[str] = ..., created_at: _Optional[str] = ..., config: _Optional[_Union[ConfigSnapshot, _Mapping]] = ..., results: _Optional[_Iterable[_Union[CaseResult, _Mapping]]] = ..., aggregate: _Optional[_Union[EvalAggregate, _Mapping]] = ..., tier: _Optional[str] = ..., degraded: _Optional[bool] = ..., degraded_reason: _Optional[str] = ...) -> None: ...

class ConfigSnapshot(_message.Message):
    __slots__ = ("rerank_enabled", "reranker_leg", "embed_model", "indexed_count", "provider_note", "embed_task_prefix", "rerank_blend", "engine", "floor_regime")
    RERANK_ENABLED_FIELD_NUMBER: _ClassVar[int]
    RERANKER_LEG_FIELD_NUMBER: _ClassVar[int]
    EMBED_MODEL_FIELD_NUMBER: _ClassVar[int]
    INDEXED_COUNT_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_NOTE_FIELD_NUMBER: _ClassVar[int]
    EMBED_TASK_PREFIX_FIELD_NUMBER: _ClassVar[int]
    RERANK_BLEND_FIELD_NUMBER: _ClassVar[int]
    ENGINE_FIELD_NUMBER: _ClassVar[int]
    FLOOR_REGIME_FIELD_NUMBER: _ClassVar[int]
    rerank_enabled: bool
    reranker_leg: str
    embed_model: str
    indexed_count: int
    provider_note: str
    embed_task_prefix: bool
    rerank_blend: bool
    engine: str
    floor_regime: str
    def __init__(self, rerank_enabled: _Optional[bool] = ..., reranker_leg: _Optional[str] = ..., embed_model: _Optional[str] = ..., indexed_count: _Optional[int] = ..., provider_note: _Optional[str] = ..., embed_task_prefix: _Optional[bool] = ..., rerank_blend: _Optional[bool] = ..., engine: _Optional[str] = ..., floor_regime: _Optional[str] = ...) -> None: ...

class CaseResult(_message.Message):
    __slots__ = ("case_id", "top", "expected_rank", "observed_top_score", "outcome", "expected_provider_id", "provider_routed", "margin", "outcome_reason")
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    TOP_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_RANK_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_TOP_SCORE_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ROUTED_FIELD_NUMBER: _ClassVar[int]
    MARGIN_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_REASON_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    top: _containers.RepeatedCompositeFieldContainer[ScoredHit]
    expected_rank: int
    observed_top_score: float
    outcome: str
    expected_provider_id: str
    provider_routed: bool
    margin: float
    outcome_reason: str
    def __init__(self, case_id: _Optional[str] = ..., top: _Optional[_Iterable[_Union[ScoredHit, _Mapping]]] = ..., expected_rank: _Optional[int] = ..., observed_top_score: _Optional[float] = ..., outcome: _Optional[str] = ..., expected_provider_id: _Optional[str] = ..., provider_routed: _Optional[bool] = ..., margin: _Optional[float] = ..., outcome_reason: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("cases", "met", "below", "mean_strong_top1", "max_gibberish_score", "latency_p95_ms", "graded_cases")
    CASES_FIELD_NUMBER: _ClassVar[int]
    MET_FIELD_NUMBER: _ClassVar[int]
    BELOW_FIELD_NUMBER: _ClassVar[int]
    MEAN_STRONG_TOP1_FIELD_NUMBER: _ClassVar[int]
    MAX_GIBBERISH_SCORE_FIELD_NUMBER: _ClassVar[int]
    LATENCY_P95_MS_FIELD_NUMBER: _ClassVar[int]
    GRADED_CASES_FIELD_NUMBER: _ClassVar[int]
    cases: int
    met: int
    below: int
    mean_strong_top1: float
    max_gibberish_score: float
    latency_p95_ms: int
    graded_cases: int
    def __init__(self, cases: _Optional[int] = ..., met: _Optional[int] = ..., below: _Optional[int] = ..., mean_strong_top1: _Optional[float] = ..., max_gibberish_score: _Optional[float] = ..., latency_p95_ms: _Optional[int] = ..., graded_cases: _Optional[int] = ...) -> None: ...

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
    __slots__ = ("suite", "adequacy")
    SUITE_FIELD_NUMBER: _ClassVar[int]
    ADEQUACY_FIELD_NUMBER: _ClassVar[int]
    suite: EvalSuite
    adequacy: _containers.RepeatedCompositeFieldContainer[AdequacyWarning]
    def __init__(self, suite: _Optional[_Union[EvalSuite, _Mapping]] = ..., adequacy: _Optional[_Iterable[_Union[AdequacyWarning, _Mapping]]] = ...) -> None: ...

class RunSuiteRequest(_message.Message):
    __slots__ = ("suite_id", "tag", "limit", "tier")
    SUITE_ID_FIELD_NUMBER: _ClassVar[int]
    TAG_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    suite_id: str
    tag: str
    limit: int
    tier: str
    def __init__(self, suite_id: _Optional[str] = ..., tag: _Optional[str] = ..., limit: _Optional[int] = ..., tier: _Optional[str] = ...) -> None: ...

class RunSuiteResponse(_message.Message):
    __slots__ = ("run", "adequacy")
    RUN_FIELD_NUMBER: _ClassVar[int]
    ADEQUACY_FIELD_NUMBER: _ClassVar[int]
    run: EvalRun
    adequacy: _containers.RepeatedCompositeFieldContainer[AdequacyWarning]
    def __init__(self, run: _Optional[_Union[EvalRun, _Mapping]] = ..., adequacy: _Optional[_Iterable[_Union[AdequacyWarning, _Mapping]]] = ...) -> None: ...

class ValidateCorpusRequest(_message.Message):
    __slots__ = ("suite_id", "deep_k")
    SUITE_ID_FIELD_NUMBER: _ClassVar[int]
    DEEP_K_FIELD_NUMBER: _ClassVar[int]
    suite_id: str
    deep_k: int
    def __init__(self, suite_id: _Optional[str] = ..., deep_k: _Optional[int] = ...) -> None: ...

class CorpusValidationCase(_message.Message):
    __slots__ = ("case_id", "referential", "observed_rank", "probed_queries", "message")
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    REFERENTIAL_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_RANK_FIELD_NUMBER: _ClassVar[int]
    PROBED_QUERIES_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    referential: ReferentialOutcome
    observed_rank: int
    probed_queries: _containers.RepeatedScalarFieldContainer[str]
    message: str
    def __init__(self, case_id: _Optional[str] = ..., referential: _Optional[_Union[ReferentialOutcome, str]] = ..., observed_rank: _Optional[int] = ..., probed_queries: _Optional[_Iterable[str]] = ..., message: _Optional[str] = ...) -> None: ...

class CorpusValidationRollup(_message.Message):
    __slots__ = ("positives", "live", "hard", "stale", "inconclusive", "candidate", "provider_errors")
    POSITIVES_FIELD_NUMBER: _ClassVar[int]
    LIVE_FIELD_NUMBER: _ClassVar[int]
    HARD_FIELD_NUMBER: _ClassVar[int]
    STALE_FIELD_NUMBER: _ClassVar[int]
    INCONCLUSIVE_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ERRORS_FIELD_NUMBER: _ClassVar[int]
    positives: int
    live: int
    hard: int
    stale: int
    inconclusive: int
    candidate: int
    provider_errors: int
    def __init__(self, positives: _Optional[int] = ..., live: _Optional[int] = ..., hard: _Optional[int] = ..., stale: _Optional[int] = ..., inconclusive: _Optional[int] = ..., candidate: _Optional[int] = ..., provider_errors: _Optional[int] = ...) -> None: ...

class ValidateCorpusResponse(_message.Message):
    __slots__ = ("suite_id", "provider_id", "cases", "rollup")
    SUITE_ID_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    CASES_FIELD_NUMBER: _ClassVar[int]
    ROLLUP_FIELD_NUMBER: _ClassVar[int]
    suite_id: str
    provider_id: str
    cases: _containers.RepeatedCompositeFieldContainer[CorpusValidationCase]
    rollup: CorpusValidationRollup
    def __init__(self, suite_id: _Optional[str] = ..., provider_id: _Optional[str] = ..., cases: _Optional[_Iterable[_Union[CorpusValidationCase, _Mapping]]] = ..., rollup: _Optional[_Union[CorpusValidationRollup, _Mapping]] = ...) -> None: ...

class ListRunsRequest(_message.Message):
    __slots__ = ("suite_id", "tag", "limit", "tier")
    SUITE_ID_FIELD_NUMBER: _ClassVar[int]
    TAG_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    suite_id: str
    tag: str
    limit: int
    tier: str
    def __init__(self, suite_id: _Optional[str] = ..., tag: _Optional[str] = ..., limit: _Optional[int] = ..., tier: _Optional[str] = ...) -> None: ...

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

class SweepRequest(_message.Message):
    __slots__ = ("suite_id", "query_time_only", "apply", "limit")
    SUITE_ID_FIELD_NUMBER: _ClassVar[int]
    QUERY_TIME_ONLY_FIELD_NUMBER: _ClassVar[int]
    APPLY_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    suite_id: str
    query_time_only: bool
    apply: bool
    limit: int
    def __init__(self, suite_id: _Optional[str] = ..., query_time_only: _Optional[bool] = ..., apply: _Optional[bool] = ..., limit: _Optional[int] = ...) -> None: ...

class SweepArm(_message.Message):
    __slots__ = ("tag", "tier", "config", "run_id", "score", "aggregate", "feasible", "note")
    TAG_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    AGGREGATE_FIELD_NUMBER: _ClassVar[int]
    FEASIBLE_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    tag: str
    tier: str
    config: ConfigSnapshot
    run_id: str
    score: float
    aggregate: EvalAggregate
    feasible: bool
    note: str
    def __init__(self, tag: _Optional[str] = ..., tier: _Optional[str] = ..., config: _Optional[_Union[ConfigSnapshot, _Mapping]] = ..., run_id: _Optional[str] = ..., score: _Optional[float] = ..., aggregate: _Optional[_Union[EvalAggregate, _Mapping]] = ..., feasible: _Optional[bool] = ..., note: _Optional[str] = ...) -> None: ...

class SweepResult(_message.Message):
    __slots__ = ("suite_id", "provider_id", "arms", "incumbent_tag", "winner_tag", "promoted", "recommendation", "stats")
    SUITE_ID_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    ARMS_FIELD_NUMBER: _ClassVar[int]
    INCUMBENT_TAG_FIELD_NUMBER: _ClassVar[int]
    WINNER_TAG_FIELD_NUMBER: _ClassVar[int]
    PROMOTED_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATION_FIELD_NUMBER: _ClassVar[int]
    STATS_FIELD_NUMBER: _ClassVar[int]
    suite_id: str
    provider_id: str
    arms: _containers.RepeatedCompositeFieldContainer[SweepArm]
    incumbent_tag: str
    winner_tag: str
    promoted: bool
    recommendation: str
    stats: SweepStats
    def __init__(self, suite_id: _Optional[str] = ..., provider_id: _Optional[str] = ..., arms: _Optional[_Iterable[_Union[SweepArm, _Mapping]]] = ..., incumbent_tag: _Optional[str] = ..., winner_tag: _Optional[str] = ..., promoted: _Optional[bool] = ..., recommendation: _Optional[str] = ..., stats: _Optional[_Union[SweepStats, _Mapping]] = ...) -> None: ...

class SweepStats(_message.Message):
    __slots__ = ("incumbent_score", "winner_score", "margin", "ci_low", "ci_high", "heldout_winner_score", "heldout_incumbent_score", "query_time_arms", "index_time_arms", "dropped_index_interactions")
    INCUMBENT_SCORE_FIELD_NUMBER: _ClassVar[int]
    WINNER_SCORE_FIELD_NUMBER: _ClassVar[int]
    MARGIN_FIELD_NUMBER: _ClassVar[int]
    CI_LOW_FIELD_NUMBER: _ClassVar[int]
    CI_HIGH_FIELD_NUMBER: _ClassVar[int]
    HELDOUT_WINNER_SCORE_FIELD_NUMBER: _ClassVar[int]
    HELDOUT_INCUMBENT_SCORE_FIELD_NUMBER: _ClassVar[int]
    QUERY_TIME_ARMS_FIELD_NUMBER: _ClassVar[int]
    INDEX_TIME_ARMS_FIELD_NUMBER: _ClassVar[int]
    DROPPED_INDEX_INTERACTIONS_FIELD_NUMBER: _ClassVar[int]
    incumbent_score: float
    winner_score: float
    margin: float
    ci_low: float
    ci_high: float
    heldout_winner_score: float
    heldout_incumbent_score: float
    query_time_arms: int
    index_time_arms: int
    dropped_index_interactions: int
    def __init__(self, incumbent_score: _Optional[float] = ..., winner_score: _Optional[float] = ..., margin: _Optional[float] = ..., ci_low: _Optional[float] = ..., ci_high: _Optional[float] = ..., heldout_winner_score: _Optional[float] = ..., heldout_incumbent_score: _Optional[float] = ..., query_time_arms: _Optional[int] = ..., index_time_arms: _Optional[int] = ..., dropped_index_interactions: _Optional[int] = ...) -> None: ...

class SweepResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: SweepResult
    def __init__(self, result: _Optional[_Union[SweepResult, _Mapping]] = ...) -> None: ...

class AdequacyWarning(_message.Message):
    __slots__ = ("code", "message", "subject")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    subject: str
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., subject: _Optional[str] = ...) -> None: ...

class GenerateRequest(_message.Message):
    __slots__ = ("suite_id", "count", "negatives", "apply")
    SUITE_ID_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    NEGATIVES_FIELD_NUMBER: _ClassVar[int]
    APPLY_FIELD_NUMBER: _ClassVar[int]
    suite_id: str
    count: int
    negatives: bool
    apply: bool
    def __init__(self, suite_id: _Optional[str] = ..., count: _Optional[int] = ..., negatives: _Optional[bool] = ..., apply: _Optional[bool] = ...) -> None: ...

class GeneratedCase(_message.Message):
    __slots__ = ("case", "source_id", "stratum")
    CASE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ID_FIELD_NUMBER: _ClassVar[int]
    STRATUM_FIELD_NUMBER: _ClassVar[int]
    case: EvalCase
    source_id: str
    stratum: str
    def __init__(self, case: _Optional[_Union[EvalCase, _Mapping]] = ..., source_id: _Optional[str] = ..., stratum: _Optional[str] = ...) -> None: ...

class GenerateResponse(_message.Message):
    __slots__ = ("suite_id", "provider_id", "proposed", "suite", "applied", "adequacy", "summary")
    SUITE_ID_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    PROPOSED_FIELD_NUMBER: _ClassVar[int]
    SUITE_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    ADEQUACY_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    suite_id: str
    provider_id: str
    proposed: _containers.RepeatedCompositeFieldContainer[GeneratedCase]
    suite: EvalSuite
    applied: bool
    adequacy: _containers.RepeatedCompositeFieldContainer[AdequacyWarning]
    summary: str
    def __init__(self, suite_id: _Optional[str] = ..., provider_id: _Optional[str] = ..., proposed: _Optional[_Iterable[_Union[GeneratedCase, _Mapping]]] = ..., suite: _Optional[_Union[EvalSuite, _Mapping]] = ..., applied: _Optional[bool] = ..., adequacy: _Optional[_Iterable[_Union[AdequacyWarning, _Mapping]]] = ..., summary: _Optional[str] = ...) -> None: ...

class PromoteCasesRequest(_message.Message):
    __slots__ = ("suite_id", "case_ids", "all")
    SUITE_ID_FIELD_NUMBER: _ClassVar[int]
    CASE_IDS_FIELD_NUMBER: _ClassVar[int]
    ALL_FIELD_NUMBER: _ClassVar[int]
    suite_id: str
    case_ids: _containers.RepeatedScalarFieldContainer[str]
    all: bool
    def __init__(self, suite_id: _Optional[str] = ..., case_ids: _Optional[_Iterable[str]] = ..., all: _Optional[bool] = ...) -> None: ...

class ReapOrphanSuitesRequest(_message.Message):
    __slots__ = ("confirm",)
    CONFIRM_FIELD_NUMBER: _ClassVar[int]
    confirm: bool
    def __init__(self, confirm: _Optional[bool] = ...) -> None: ...

class ReapOrphanSuitesResponse(_message.Message):
    __slots__ = ("orphan_suites", "reaped_suite_ids", "confirmed")
    ORPHAN_SUITES_FIELD_NUMBER: _ClassVar[int]
    REAPED_SUITE_IDS_FIELD_NUMBER: _ClassVar[int]
    CONFIRMED_FIELD_NUMBER: _ClassVar[int]
    orphan_suites: _containers.RepeatedCompositeFieldContainer[EvalSuite]
    reaped_suite_ids: _containers.RepeatedScalarFieldContainer[str]
    confirmed: bool
    def __init__(self, orphan_suites: _Optional[_Iterable[_Union[EvalSuite, _Mapping]]] = ..., reaped_suite_ids: _Optional[_Iterable[str]] = ..., confirmed: _Optional[bool] = ...) -> None: ...

class PromoteCasesResponse(_message.Message):
    __slots__ = ("suite_id", "provider_id", "promoted_case_ids", "already_reviewed_case_ids", "suite", "applied")
    SUITE_ID_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    PROMOTED_CASE_IDS_FIELD_NUMBER: _ClassVar[int]
    ALREADY_REVIEWED_CASE_IDS_FIELD_NUMBER: _ClassVar[int]
    SUITE_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    suite_id: str
    provider_id: str
    promoted_case_ids: _containers.RepeatedScalarFieldContainer[str]
    already_reviewed_case_ids: _containers.RepeatedScalarFieldContainer[str]
    suite: EvalSuite
    applied: bool
    def __init__(self, suite_id: _Optional[str] = ..., provider_id: _Optional[str] = ..., promoted_case_ids: _Optional[_Iterable[str]] = ..., already_reviewed_case_ids: _Optional[_Iterable[str]] = ..., suite: _Optional[_Union[EvalSuite, _Mapping]] = ..., applied: _Optional[bool] = ...) -> None: ...
