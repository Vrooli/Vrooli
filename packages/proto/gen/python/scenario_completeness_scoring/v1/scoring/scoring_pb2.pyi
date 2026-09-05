import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ScoreSortBy(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SCORE_SORT_BY_UNSPECIFIED: _ClassVar[ScoreSortBy]
    SCORE_SORT_BY_COMPOSITE: _ClassVar[ScoreSortBy]
    SCORE_SORT_BY_RUNG: _ClassVar[ScoreSortBy]
    SCORE_SORT_BY_LAST_SCORED: _ClassVar[ScoreSortBy]
    SCORE_SORT_BY_SCENARIO: _ClassVar[ScoreSortBy]
    SCORE_SORT_BY_PRIORITY: _ClassVar[ScoreSortBy]

class SortOrder(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SORT_ORDER_UNSPECIFIED: _ClassVar[SortOrder]
    SORT_ORDER_ASC: _ClassVar[SortOrder]
    SORT_ORDER_DESC: _ClassVar[SortOrder]
SCORE_SORT_BY_UNSPECIFIED: ScoreSortBy
SCORE_SORT_BY_COMPOSITE: ScoreSortBy
SCORE_SORT_BY_RUNG: ScoreSortBy
SCORE_SORT_BY_LAST_SCORED: ScoreSortBy
SCORE_SORT_BY_SCENARIO: ScoreSortBy
SCORE_SORT_BY_PRIORITY: ScoreSortBy
SORT_ORDER_UNSPECIFIED: SortOrder
SORT_ORDER_ASC: SortOrder
SORT_ORDER_DESC: SortOrder

class GetScoreRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class GetScoreResponse(_message.Message):
    __slots__ = ("scenario", "category", "maturity", "composite", "freshness", "recommendations", "action_plan", "degradations", "calculated_at", "importance", "trend")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    MATURITY_FIELD_NUMBER: _ClassVar[int]
    COMPOSITE_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATIONS_FIELD_NUMBER: _ClassVar[int]
    ACTION_PLAN_FIELD_NUMBER: _ClassVar[int]
    DEGRADATIONS_FIELD_NUMBER: _ClassVar[int]
    CALCULATED_AT_FIELD_NUMBER: _ClassVar[int]
    IMPORTANCE_FIELD_NUMBER: _ClassVar[int]
    TREND_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    category: str
    maturity: MaturityHeadline
    composite: CompositeScore
    freshness: FreshnessBlock
    recommendations: _containers.RepeatedCompositeFieldContainer[Recommendation]
    action_plan: _containers.RepeatedCompositeFieldContainer[ActionPhase]
    degradations: _containers.RepeatedCompositeFieldContainer[CollectorDegradation]
    calculated_at: _timestamp_pb2.Timestamp
    importance: ImportanceSummary
    trend: TrendSummary
    def __init__(self, scenario: _Optional[str] = ..., category: _Optional[str] = ..., maturity: _Optional[_Union[MaturityHeadline, _Mapping]] = ..., composite: _Optional[_Union[CompositeScore, _Mapping]] = ..., freshness: _Optional[_Union[FreshnessBlock, _Mapping]] = ..., recommendations: _Optional[_Iterable[_Union[Recommendation, _Mapping]]] = ..., action_plan: _Optional[_Iterable[_Union[ActionPhase, _Mapping]]] = ..., degradations: _Optional[_Iterable[_Union[CollectorDegradation, _Mapping]]] = ..., calculated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., importance: _Optional[_Union[ImportanceSummary, _Mapping]] = ..., trend: _Optional[_Union[TrendSummary, _Mapping]] = ...) -> None: ...

class TrendSummary(_message.Message):
    __slots__ = ("previous_score", "previous_calculated_at", "delta")
    PREVIOUS_SCORE_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_CALCULATED_AT_FIELD_NUMBER: _ClassVar[int]
    DELTA_FIELD_NUMBER: _ClassVar[int]
    previous_score: int
    previous_calculated_at: _timestamp_pb2.Timestamp
    delta: int
    def __init__(self, previous_score: _Optional[int] = ..., previous_calculated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., delta: _Optional[int] = ...) -> None: ...

class GetScoreTrendRequest(_message.Message):
    __slots__ = ("scenario", "limit", "since")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    SINCE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    limit: int
    since: _timestamp_pb2.Timestamp
    def __init__(self, scenario: _Optional[str] = ..., limit: _Optional[int] = ..., since: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetScoreTrendResponse(_message.Message):
    __slots__ = ("scenario", "snapshots")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOTS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    snapshots: _containers.RepeatedCompositeFieldContainer[ScoreSnapshot]
    def __init__(self, scenario: _Optional[str] = ..., snapshots: _Optional[_Iterable[_Union[ScoreSnapshot, _Mapping]]] = ...) -> None: ...

class ListScoresRequest(_message.Message):
    __slots__ = ("sort_by", "order", "page_size", "page_token", "min_score", "max_score", "rung", "category", "recompute")
    SORT_BY_FIELD_NUMBER: _ClassVar[int]
    ORDER_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    MIN_SCORE_FIELD_NUMBER: _ClassVar[int]
    MAX_SCORE_FIELD_NUMBER: _ClassVar[int]
    RUNG_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    RECOMPUTE_FIELD_NUMBER: _ClassVar[int]
    sort_by: ScoreSortBy
    order: SortOrder
    page_size: int
    page_token: str
    min_score: int
    max_score: int
    rung: str
    category: str
    recompute: bool
    def __init__(self, sort_by: _Optional[_Union[ScoreSortBy, str]] = ..., order: _Optional[_Union[SortOrder, str]] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ..., min_score: _Optional[int] = ..., max_score: _Optional[int] = ..., rung: _Optional[str] = ..., category: _Optional[str] = ..., recompute: _Optional[bool] = ...) -> None: ...

class ListScoresResponse(_message.Message):
    __slots__ = ("scores", "next_page_token")
    SCORES_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    scores: _containers.RepeatedCompositeFieldContainer[ScoreRow]
    next_page_token: str
    def __init__(self, scores: _Optional[_Iterable[_Union[ScoreRow, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class ScoreRow(_message.Message):
    __slots__ = ("scenario", "category", "score", "classification", "working_rung", "importance", "priority", "calculated_at", "digest", "last_run_at", "last_status")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    WORKING_RUNG_FIELD_NUMBER: _ClassVar[int]
    IMPORTANCE_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    CALCULATED_AT_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    LAST_RUN_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_STATUS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    category: str
    score: int
    classification: str
    working_rung: str
    importance: float
    priority: float
    calculated_at: _timestamp_pb2.Timestamp
    digest: str
    last_run_at: _timestamp_pb2.Timestamp
    last_status: str
    def __init__(self, scenario: _Optional[str] = ..., category: _Optional[str] = ..., score: _Optional[int] = ..., classification: _Optional[str] = ..., working_rung: _Optional[str] = ..., importance: _Optional[float] = ..., priority: _Optional[float] = ..., calculated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., digest: _Optional[str] = ..., last_run_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_status: _Optional[str] = ...) -> None: ...

class ScoreSnapshot(_message.Message):
    __slots__ = ("scenario", "category", "digest", "score", "classification", "working_rung", "breakdown_json", "importance", "importance_present", "source", "calculated_at")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    WORKING_RUNG_FIELD_NUMBER: _ClassVar[int]
    BREAKDOWN_JSON_FIELD_NUMBER: _ClassVar[int]
    IMPORTANCE_FIELD_NUMBER: _ClassVar[int]
    IMPORTANCE_PRESENT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CALCULATED_AT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    category: str
    digest: str
    score: int
    classification: str
    working_rung: str
    breakdown_json: str
    importance: float
    importance_present: bool
    source: str
    calculated_at: _timestamp_pb2.Timestamp
    def __init__(self, scenario: _Optional[str] = ..., category: _Optional[str] = ..., digest: _Optional[str] = ..., score: _Optional[int] = ..., classification: _Optional[str] = ..., working_rung: _Optional[str] = ..., breakdown_json: _Optional[str] = ..., importance: _Optional[float] = ..., importance_present: _Optional[bool] = ..., source: _Optional[str] = ..., calculated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class MaturityHeadline(_message.Message):
    __slots__ = ("working_rung", "ladder_clean", "satisfied_through", "dimensions", "build_passing")
    WORKING_RUNG_FIELD_NUMBER: _ClassVar[int]
    LADDER_CLEAN_FIELD_NUMBER: _ClassVar[int]
    SATISFIED_THROUGH_FIELD_NUMBER: _ClassVar[int]
    DIMENSIONS_FIELD_NUMBER: _ClassVar[int]
    BUILD_PASSING_FIELD_NUMBER: _ClassVar[int]
    working_rung: str
    ladder_clean: bool
    satisfied_through: str
    dimensions: _containers.RepeatedCompositeFieldContainer[DimensionCount]
    build_passing: bool
    def __init__(self, working_rung: _Optional[str] = ..., ladder_clean: _Optional[bool] = ..., satisfied_through: _Optional[str] = ..., dimensions: _Optional[_Iterable[_Union[DimensionCount, _Mapping]]] = ..., build_passing: _Optional[bool] = ...) -> None: ...

class DimensionCount(_message.Message):
    __slots__ = ("dimension", "error_plus", "total", "approximate")
    DIMENSION_FIELD_NUMBER: _ClassVar[int]
    ERROR_PLUS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    APPROXIMATE_FIELD_NUMBER: _ClassVar[int]
    dimension: str
    error_plus: int
    total: int
    approximate: bool
    def __init__(self, dimension: _Optional[str] = ..., error_plus: _Optional[int] = ..., total: _Optional[int] = ..., approximate: _Optional[bool] = ...) -> None: ...

class CompositeScore(_message.Message):
    __slots__ = ("score", "classification", "classification_label", "groups")
    SCORE_FIELD_NUMBER: _ClassVar[int]
    CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    CLASSIFICATION_LABEL_FIELD_NUMBER: _ClassVar[int]
    GROUPS_FIELD_NUMBER: _ClassVar[int]
    score: int
    classification: str
    classification_label: str
    groups: _containers.RepeatedCompositeFieldContainer[ScoreGroup]
    def __init__(self, score: _Optional[int] = ..., classification: _Optional[str] = ..., classification_label: _Optional[str] = ..., groups: _Optional[_Iterable[_Union[ScoreGroup, _Mapping]]] = ...) -> None: ...

class ScoreGroup(_message.Message):
    __slots__ = ("id", "label", "score", "max", "metrics")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    MAX_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    score: float
    max: float
    metrics: _containers.RepeatedCompositeFieldContainer[MetricLine]
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., score: _Optional[float] = ..., max: _Optional[float] = ..., metrics: _Optional[_Iterable[_Union[MetricLine, _Mapping]]] = ...) -> None: ...

class MetricLine(_message.Message):
    __slots__ = ("id", "label", "observed", "points", "max_points", "threshold")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_FIELD_NUMBER: _ClassVar[int]
    POINTS_FIELD_NUMBER: _ClassVar[int]
    MAX_POINTS_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    observed: str
    points: float
    max_points: float
    threshold: str
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., observed: _Optional[str] = ..., points: _Optional[float] = ..., max_points: _Optional[float] = ..., threshold: _Optional[str] = ...) -> None: ...

class FreshnessBlock(_message.Message):
    __slots__ = ("current_digest", "digest_error", "phases", "suggested_command")
    CURRENT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    DIGEST_ERROR_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    SUGGESTED_COMMAND_FIELD_NUMBER: _ClassVar[int]
    current_digest: str
    digest_error: str
    phases: _containers.RepeatedCompositeFieldContainer[PhaseFreshness]
    suggested_command: str
    def __init__(self, current_digest: _Optional[str] = ..., digest_error: _Optional[str] = ..., phases: _Optional[_Iterable[_Union[PhaseFreshness, _Mapping]]] = ..., suggested_command: _Optional[str] = ...) -> None: ...

class PhaseFreshness(_message.Message):
    __slots__ = ("phase", "verdict", "last_run_id", "last_run_at", "last_digest", "last_status")
    PHASE_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    LAST_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    LAST_RUN_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_DIGEST_FIELD_NUMBER: _ClassVar[int]
    LAST_STATUS_FIELD_NUMBER: _ClassVar[int]
    phase: str
    verdict: str
    last_run_id: str
    last_run_at: _timestamp_pb2.Timestamp
    last_digest: str
    last_status: str
    def __init__(self, phase: _Optional[str] = ..., verdict: _Optional[str] = ..., last_run_id: _Optional[str] = ..., last_run_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_digest: _Optional[str] = ..., last_status: _Optional[str] = ...) -> None: ...

class Recommendation(_message.Message):
    __slots__ = ("priority", "description", "impact_points")
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    IMPACT_POINTS_FIELD_NUMBER: _ClassVar[int]
    priority: str
    description: str
    impact_points: float
    def __init__(self, priority: _Optional[str] = ..., description: _Optional[str] = ..., impact_points: _Optional[float] = ...) -> None: ...

class ActionPhase(_message.Message):
    __slots__ = ("title", "actions", "estimated_points")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_POINTS_FIELD_NUMBER: _ClassVar[int]
    title: str
    actions: _containers.RepeatedScalarFieldContainer[str]
    estimated_points: float
    def __init__(self, title: _Optional[str] = ..., actions: _Optional[_Iterable[str]] = ..., estimated_points: _Optional[float] = ...) -> None: ...

class CollectorDegradation(_message.Message):
    __slots__ = ("collector", "state", "reason")
    COLLECTOR_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    collector: str
    state: str
    reason: str
    def __init__(self, collector: _Optional[str] = ..., state: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class ImportanceSummary(_message.Message):
    __slots__ = ("score", "system_required", "components", "signals", "degraded")
    SCORE_FIELD_NUMBER: _ClassVar[int]
    SYSTEM_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    COMPONENTS_FIELD_NUMBER: _ClassVar[int]
    SIGNALS_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    score: float
    system_required: bool
    components: ImportanceComponents
    signals: ImportanceSignals
    degraded: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, score: _Optional[float] = ..., system_required: _Optional[bool] = ..., components: _Optional[_Union[ImportanceComponents, _Mapping]] = ..., signals: _Optional[_Union[ImportanceSignals, _Mapping]] = ..., degraded: _Optional[_Iterable[str]] = ...) -> None: ...

class ImportanceComponents(_message.Message):
    __slots__ = ("centrality", "core_proximity", "recency")
    CENTRALITY_FIELD_NUMBER: _ClassVar[int]
    CORE_PROXIMITY_FIELD_NUMBER: _ClassVar[int]
    RECENCY_FIELD_NUMBER: _ClassVar[int]
    centrality: float
    core_proximity: float
    recency: float
    def __init__(self, centrality: _Optional[float] = ..., core_proximity: _Optional[float] = ..., recency: _Optional[float] = ...) -> None: ...

class ImportanceSignals(_message.Message):
    __slots__ = ("direct_reverse_dependency_count", "transitive_reverse_dependency_count", "required_reverse_dependency_count", "required_edge_weighted_score", "distance_to_core_seed", "nearest_core_seed", "recent_activity_count")
    DIRECT_REVERSE_DEPENDENCY_COUNT_FIELD_NUMBER: _ClassVar[int]
    TRANSITIVE_REVERSE_DEPENDENCY_COUNT_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_REVERSE_DEPENDENCY_COUNT_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_EDGE_WEIGHTED_SCORE_FIELD_NUMBER: _ClassVar[int]
    DISTANCE_TO_CORE_SEED_FIELD_NUMBER: _ClassVar[int]
    NEAREST_CORE_SEED_FIELD_NUMBER: _ClassVar[int]
    RECENT_ACTIVITY_COUNT_FIELD_NUMBER: _ClassVar[int]
    direct_reverse_dependency_count: int
    transitive_reverse_dependency_count: int
    required_reverse_dependency_count: int
    required_edge_weighted_score: float
    distance_to_core_seed: int
    nearest_core_seed: str
    recent_activity_count: int
    def __init__(self, direct_reverse_dependency_count: _Optional[int] = ..., transitive_reverse_dependency_count: _Optional[int] = ..., required_reverse_dependency_count: _Optional[int] = ..., required_edge_weighted_score: _Optional[float] = ..., distance_to_core_seed: _Optional[int] = ..., nearest_core_seed: _Optional[str] = ..., recent_activity_count: _Optional[int] = ...) -> None: ...
