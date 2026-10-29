from measures.v1 import measures_pb2 as _measures_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class InvocationFilter(_message.Message):
    __slots__ = ("window", "ownership", "outcome", "executable", "fingerprint", "profile_id", "runner_type", "model", "tag_prefix", "run_status", "tool_name", "episode_pattern", "episode_cause_scope", "episode_fingerprint", "self_report_rule_id", "self_report_cause_scope", "target_scenario", "operation", "workload_key", "error_code")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    OWNERSHIP_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    EXECUTABLE_FIELD_NUMBER: _ClassVar[int]
    FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    RUNNER_TYPE_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    TAG_PREFIX_FIELD_NUMBER: _ClassVar[int]
    RUN_STATUS_FIELD_NUMBER: _ClassVar[int]
    TOOL_NAME_FIELD_NUMBER: _ClassVar[int]
    EPISODE_PATTERN_FIELD_NUMBER: _ClassVar[int]
    EPISODE_CAUSE_SCOPE_FIELD_NUMBER: _ClassVar[int]
    EPISODE_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    SELF_REPORT_RULE_ID_FIELD_NUMBER: _ClassVar[int]
    SELF_REPORT_CAUSE_SCOPE_FIELD_NUMBER: _ClassVar[int]
    TARGET_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    WORKLOAD_KEY_FIELD_NUMBER: _ClassVar[int]
    ERROR_CODE_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    ownership: str
    outcome: str
    executable: str
    fingerprint: str
    profile_id: str
    runner_type: str
    model: str
    tag_prefix: str
    run_status: str
    tool_name: str
    episode_pattern: str
    episode_cause_scope: str
    episode_fingerprint: str
    self_report_rule_id: str
    self_report_cause_scope: str
    target_scenario: str
    operation: str
    workload_key: str
    error_code: str
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., ownership: _Optional[str] = ..., outcome: _Optional[str] = ..., executable: _Optional[str] = ..., fingerprint: _Optional[str] = ..., profile_id: _Optional[str] = ..., runner_type: _Optional[str] = ..., model: _Optional[str] = ..., tag_prefix: _Optional[str] = ..., run_status: _Optional[str] = ..., tool_name: _Optional[str] = ..., episode_pattern: _Optional[str] = ..., episode_cause_scope: _Optional[str] = ..., episode_fingerprint: _Optional[str] = ..., self_report_rule_id: _Optional[str] = ..., self_report_cause_scope: _Optional[str] = ..., target_scenario: _Optional[str] = ..., operation: _Optional[str] = ..., workload_key: _Optional[str] = ..., error_code: _Optional[str] = ...) -> None: ...

class MeasureValidity(_message.Message):
    __slots__ = ("state", "reason", "sample_size", "largest_fingerprint_bucket", "largest_fingerprint_share", "classified_base", "unclassified_count", "unclassified_share", "minimum_classified_share")
    STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_SIZE_FIELD_NUMBER: _ClassVar[int]
    LARGEST_FINGERPRINT_BUCKET_FIELD_NUMBER: _ClassVar[int]
    LARGEST_FINGERPRINT_SHARE_FIELD_NUMBER: _ClassVar[int]
    CLASSIFIED_BASE_FIELD_NUMBER: _ClassVar[int]
    UNCLASSIFIED_COUNT_FIELD_NUMBER: _ClassVar[int]
    UNCLASSIFIED_SHARE_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_CLASSIFIED_SHARE_FIELD_NUMBER: _ClassVar[int]
    state: str
    reason: str
    sample_size: int
    largest_fingerprint_bucket: int
    largest_fingerprint_share: float
    classified_base: int
    unclassified_count: int
    unclassified_share: float
    minimum_classified_share: float
    def __init__(self, state: _Optional[str] = ..., reason: _Optional[str] = ..., sample_size: _Optional[int] = ..., largest_fingerprint_bucket: _Optional[int] = ..., largest_fingerprint_share: _Optional[float] = ..., classified_base: _Optional[int] = ..., unclassified_count: _Optional[int] = ..., unclassified_share: _Optional[float] = ..., minimum_classified_share: _Optional[float] = ...) -> None: ...

class MeasureFilter(_message.Message):
    __slots__ = ("field", "value")
    FIELD_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    field: str
    value: str
    def __init__(self, field: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...

class MeasureProvenance(_message.Message):
    __slots__ = ("source_table", "window_start", "window_end", "row_count", "applied_filters", "executed_query", "projection_at", "projection_age_ms", "projection_stale", "projection_stale_reason")
    SOURCE_TABLE_FIELD_NUMBER: _ClassVar[int]
    WINDOW_START_FIELD_NUMBER: _ClassVar[int]
    WINDOW_END_FIELD_NUMBER: _ClassVar[int]
    ROW_COUNT_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FILTERS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    PROJECTION_AT_FIELD_NUMBER: _ClassVar[int]
    PROJECTION_AGE_MS_FIELD_NUMBER: _ClassVar[int]
    PROJECTION_STALE_FIELD_NUMBER: _ClassVar[int]
    PROJECTION_STALE_REASON_FIELD_NUMBER: _ClassVar[int]
    source_table: str
    window_start: str
    window_end: str
    row_count: int
    applied_filters: _containers.RepeatedCompositeFieldContainer[MeasureFilter]
    executed_query: str
    projection_at: str
    projection_age_ms: int
    projection_stale: bool
    projection_stale_reason: str
    def __init__(self, source_table: _Optional[str] = ..., window_start: _Optional[str] = ..., window_end: _Optional[str] = ..., row_count: _Optional[int] = ..., applied_filters: _Optional[_Iterable[_Union[MeasureFilter, _Mapping]]] = ..., executed_query: _Optional[str] = ..., projection_at: _Optional[str] = ..., projection_age_ms: _Optional[int] = ..., projection_stale: _Optional[bool] = ..., projection_stale_reason: _Optional[str] = ...) -> None: ...

class MeasureDefinition(_message.Message):
    __slots__ = ("id", "counts", "numerator", "denominator", "source_table", "limitation")
    ID_FIELD_NUMBER: _ClassVar[int]
    COUNTS_FIELD_NUMBER: _ClassVar[int]
    NUMERATOR_FIELD_NUMBER: _ClassVar[int]
    DENOMINATOR_FIELD_NUMBER: _ClassVar[int]
    SOURCE_TABLE_FIELD_NUMBER: _ClassVar[int]
    LIMITATION_FIELD_NUMBER: _ClassVar[int]
    id: str
    counts: str
    numerator: str
    denominator: str
    source_table: str
    limitation: str
    def __init__(self, id: _Optional[str] = ..., counts: _Optional[str] = ..., numerator: _Optional[str] = ..., denominator: _Optional[str] = ..., source_table: _Optional[str] = ..., limitation: _Optional[str] = ...) -> None: ...

class AllMeasureDefinitionsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class AllMeasureDefinitionsResponse(_message.Message):
    __slots__ = ("definitions",)
    DEFINITIONS_FIELD_NUMBER: _ClassVar[int]
    definitions: _containers.RepeatedCompositeFieldContainer[MeasureDefinition]
    def __init__(self, definitions: _Optional[_Iterable[_Union[MeasureDefinition, _Mapping]]] = ...) -> None: ...

class ConversationSearchQualityRequest(_message.Message):
    __slots__ = ("window",)
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ...) -> None: ...

class ConversationSearchQualityResponse(_message.Message):
    __slots__ = ("queries", "no_result_queries", "no_result_rate", "weak_only_queries", "weak_only_rate", "reformulated_queries", "reformulation_rate", "selected_queries", "selected_query_rate", "p50_selected_rank", "p95_selected_rank", "lexical_contribution_rate", "semantic_contribution_rate", "degraded_queries", "degradation_rate", "p50_latency_ms", "p95_latency_ms", "error_queries", "error_rate", "current_index_lag_ms", "pending_documents", "orphan_documents", "truncated", "validity", "provenance", "definition_id")
    QUERIES_FIELD_NUMBER: _ClassVar[int]
    NO_RESULT_QUERIES_FIELD_NUMBER: _ClassVar[int]
    NO_RESULT_RATE_FIELD_NUMBER: _ClassVar[int]
    WEAK_ONLY_QUERIES_FIELD_NUMBER: _ClassVar[int]
    WEAK_ONLY_RATE_FIELD_NUMBER: _ClassVar[int]
    REFORMULATED_QUERIES_FIELD_NUMBER: _ClassVar[int]
    REFORMULATION_RATE_FIELD_NUMBER: _ClassVar[int]
    SELECTED_QUERIES_FIELD_NUMBER: _ClassVar[int]
    SELECTED_QUERY_RATE_FIELD_NUMBER: _ClassVar[int]
    P50_SELECTED_RANK_FIELD_NUMBER: _ClassVar[int]
    P95_SELECTED_RANK_FIELD_NUMBER: _ClassVar[int]
    LEXICAL_CONTRIBUTION_RATE_FIELD_NUMBER: _ClassVar[int]
    SEMANTIC_CONTRIBUTION_RATE_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_QUERIES_FIELD_NUMBER: _ClassVar[int]
    DEGRADATION_RATE_FIELD_NUMBER: _ClassVar[int]
    P50_LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    P95_LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    ERROR_QUERIES_FIELD_NUMBER: _ClassVar[int]
    ERROR_RATE_FIELD_NUMBER: _ClassVar[int]
    CURRENT_INDEX_LAG_MS_FIELD_NUMBER: _ClassVar[int]
    PENDING_DOCUMENTS_FIELD_NUMBER: _ClassVar[int]
    ORPHAN_DOCUMENTS_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    queries: int
    no_result_queries: int
    no_result_rate: float
    weak_only_queries: int
    weak_only_rate: float
    reformulated_queries: int
    reformulation_rate: float
    selected_queries: int
    selected_query_rate: float
    p50_selected_rank: float
    p95_selected_rank: float
    lexical_contribution_rate: float
    semantic_contribution_rate: float
    degraded_queries: int
    degradation_rate: float
    p50_latency_ms: float
    p95_latency_ms: float
    error_queries: int
    error_rate: float
    current_index_lag_ms: int
    pending_documents: int
    orphan_documents: int
    truncated: bool
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, queries: _Optional[int] = ..., no_result_queries: _Optional[int] = ..., no_result_rate: _Optional[float] = ..., weak_only_queries: _Optional[int] = ..., weak_only_rate: _Optional[float] = ..., reformulated_queries: _Optional[int] = ..., reformulation_rate: _Optional[float] = ..., selected_queries: _Optional[int] = ..., selected_query_rate: _Optional[float] = ..., p50_selected_rank: _Optional[float] = ..., p95_selected_rank: _Optional[float] = ..., lexical_contribution_rate: _Optional[float] = ..., semantic_contribution_rate: _Optional[float] = ..., degraded_queries: _Optional[int] = ..., degradation_rate: _Optional[float] = ..., p50_latency_ms: _Optional[float] = ..., p95_latency_ms: _Optional[float] = ..., error_queries: _Optional[int] = ..., error_rate: _Optional[float] = ..., current_index_lag_ms: _Optional[int] = ..., pending_documents: _Optional[int] = ..., orphan_documents: _Optional[int] = ..., truncated: _Optional[bool] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class ExternalToolShareRequest(_message.Message):
    __slots__ = ("window", "filter")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ...) -> None: ...

class ExternalToolShareResponse(_message.Message):
    __slots__ = ("share", "external_calls", "resolved_calls", "unknown_calls", "executed_query", "validity", "provenance", "definition_id")
    SHARE_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_CALLS_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_CALLS_FIELD_NUMBER: _ClassVar[int]
    UNKNOWN_CALLS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    share: float
    external_calls: int
    resolved_calls: int
    unknown_calls: int
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, share: _Optional[float] = ..., external_calls: _Optional[int] = ..., resolved_calls: _Optional[int] = ..., unknown_calls: _Optional[int] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class RetryRateRequest(_message.Message):
    __slots__ = ("window", "filter")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ...) -> None: ...

class RetryRateResponse(_message.Message):
    __slots__ = ("rate", "retry_calls", "total_calls", "executed_query", "validity", "provenance", "definition_id")
    RATE_FIELD_NUMBER: _ClassVar[int]
    RETRY_CALLS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_CALLS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    rate: float
    retry_calls: int
    total_calls: int
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, rate: _Optional[float] = ..., retry_calls: _Optional[int] = ..., total_calls: _Optional[int] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class HelpRecoveryRateRequest(_message.Message):
    __slots__ = ("window", "filter")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ...) -> None: ...

class HelpRecoveryRateResponse(_message.Message):
    __slots__ = ("rate", "help_recoveries", "total_calls", "executed_query", "validity", "provenance", "definition_id")
    RATE_FIELD_NUMBER: _ClassVar[int]
    HELP_RECOVERIES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_CALLS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    rate: float
    help_recoveries: int
    total_calls: int
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, rate: _Optional[float] = ..., help_recoveries: _Optional[int] = ..., total_calls: _Optional[int] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class RepeatedWorkRateRequest(_message.Message):
    __slots__ = ("window", "filter")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ...) -> None: ...

class RepeatedWorkRateResponse(_message.Message):
    __slots__ = ("rate", "repeated_calls", "total_calls", "executed_query", "validity", "provenance", "definition_id")
    RATE_FIELD_NUMBER: _ClassVar[int]
    REPEATED_CALLS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_CALLS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    rate: float
    repeated_calls: int
    total_calls: int
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, rate: _Optional[float] = ..., repeated_calls: _Optional[int] = ..., total_calls: _Optional[int] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class ToolFailureRateRequest(_message.Message):
    __slots__ = ("window", "filter")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ...) -> None: ...

class ToolFailureRateResponse(_message.Message):
    __slots__ = ("rate", "failed_calls", "total_calls", "executed_query", "validity", "provenance", "definition_id")
    RATE_FIELD_NUMBER: _ClassVar[int]
    FAILED_CALLS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_CALLS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    rate: float
    failed_calls: int
    total_calls: int
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, rate: _Optional[float] = ..., failed_calls: _Optional[int] = ..., total_calls: _Optional[int] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class RunSuccessRateRequest(_message.Message):
    __slots__ = ("window", "filter")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ...) -> None: ...

class RunSuccessRateResponse(_message.Message):
    __slots__ = ("rate", "successful_runs", "terminal_runs", "executed_query", "validity", "provenance", "definition_id")
    RATE_FIELD_NUMBER: _ClassVar[int]
    SUCCESSFUL_RUNS_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_RUNS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    rate: float
    successful_runs: int
    terminal_runs: int
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, rate: _Optional[float] = ..., successful_runs: _Optional[int] = ..., terminal_runs: _Optional[int] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class RunCycleTimeRequest(_message.Message):
    __slots__ = ("window", "filter")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ...) -> None: ...

class RunCycleTimeResponse(_message.Message):
    __slots__ = ("average_duration_ms", "completed_duration_runs", "executed_query", "validity", "provenance", "definition_id")
    AVERAGE_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_DURATION_RUNS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    average_duration_ms: float
    completed_duration_runs: int
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, average_duration_ms: _Optional[float] = ..., completed_duration_runs: _Optional[int] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class RunDurationStatisticsRequest(_message.Message):
    __slots__ = ("window", "filter")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ...) -> None: ...

class RunDurationStatisticsResponse(_message.Message):
    __slots__ = ("average_duration_ms", "p50_duration_ms", "p95_duration_ms", "p99_duration_ms", "min_duration_ms", "max_duration_ms", "count", "executed_query", "validity", "provenance", "definition_id")
    AVERAGE_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    P50_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    P95_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    P99_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    MIN_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    MAX_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    average_duration_ms: float
    p50_duration_ms: float
    p95_duration_ms: float
    p99_duration_ms: float
    min_duration_ms: int
    max_duration_ms: int
    count: int
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, average_duration_ms: _Optional[float] = ..., p50_duration_ms: _Optional[float] = ..., p95_duration_ms: _Optional[float] = ..., p99_duration_ms: _Optional[float] = ..., min_duration_ms: _Optional[int] = ..., max_duration_ms: _Optional[int] = ..., count: _Optional[int] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class RunCostRequest(_message.Message):
    __slots__ = ("window", "filter", "allocate_subscription", "allocation_basis")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    ALLOCATE_SUBSCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ALLOCATION_BASIS_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    allocate_subscription: bool
    allocation_basis: str
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ..., allocate_subscription: _Optional[bool] = ..., allocation_basis: _Optional[str] = ...) -> None: ...

class ChargeByBasis(_message.Message):
    __slots__ = ("basis", "run_count", "charge_micro_usd", "token_count", "charge_reason")
    BASIS_FIELD_NUMBER: _ClassVar[int]
    RUN_COUNT_FIELD_NUMBER: _ClassVar[int]
    CHARGE_MICRO_USD_FIELD_NUMBER: _ClassVar[int]
    TOKEN_COUNT_FIELD_NUMBER: _ClassVar[int]
    CHARGE_REASON_FIELD_NUMBER: _ClassVar[int]
    basis: str
    run_count: int
    charge_micro_usd: int
    token_count: int
    charge_reason: str
    def __init__(self, basis: _Optional[str] = ..., run_count: _Optional[int] = ..., charge_micro_usd: _Optional[int] = ..., token_count: _Optional[int] = ..., charge_reason: _Optional[str] = ...) -> None: ...

class RunTokenBucket(_message.Message):
    __slots__ = ("label", "min_tokens", "max_tokens", "run_count")
    LABEL_FIELD_NUMBER: _ClassVar[int]
    MIN_TOKENS_FIELD_NUMBER: _ClassVar[int]
    MAX_TOKENS_FIELD_NUMBER: _ClassVar[int]
    RUN_COUNT_FIELD_NUMBER: _ClassVar[int]
    label: str
    min_tokens: int
    max_tokens: int
    run_count: int
    def __init__(self, label: _Optional[str] = ..., min_tokens: _Optional[int] = ..., max_tokens: _Optional[int] = ..., run_count: _Optional[int] = ...) -> None: ...

class RunCostResponse(_message.Message):
    __slots__ = ("total_cost_usd", "average_cost_usd", "total_tokens", "total_runs", "executed_query", "validity", "input_tokens", "output_tokens", "cache_read_tokens", "cache_creation_tokens", "input_cost_usd", "output_cost_usd", "cache_read_cost_usd", "cache_creation_cost_usd", "charge_by_basis", "total_charge_micro_usd", "unpriced_token_count", "p50_tokens", "p90_tokens", "p95_tokens", "p99_tokens", "max_tokens", "p50_cost_usd", "p90_cost_usd", "p95_cost_usd", "p99_cost_usd", "max_cost_usd", "token_buckets", "distribution_sample_size", "distribution_unobserved_runs", "provenance", "definition_id")
    TOTAL_COST_USD_FIELD_NUMBER: _ClassVar[int]
    AVERAGE_COST_USD_FIELD_NUMBER: _ClassVar[int]
    TOTAL_TOKENS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_RUNS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    INPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    CACHE_READ_TOKENS_FIELD_NUMBER: _ClassVar[int]
    CACHE_CREATION_TOKENS_FIELD_NUMBER: _ClassVar[int]
    INPUT_COST_USD_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_COST_USD_FIELD_NUMBER: _ClassVar[int]
    CACHE_READ_COST_USD_FIELD_NUMBER: _ClassVar[int]
    CACHE_CREATION_COST_USD_FIELD_NUMBER: _ClassVar[int]
    CHARGE_BY_BASIS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_CHARGE_MICRO_USD_FIELD_NUMBER: _ClassVar[int]
    UNPRICED_TOKEN_COUNT_FIELD_NUMBER: _ClassVar[int]
    P50_TOKENS_FIELD_NUMBER: _ClassVar[int]
    P90_TOKENS_FIELD_NUMBER: _ClassVar[int]
    P95_TOKENS_FIELD_NUMBER: _ClassVar[int]
    P99_TOKENS_FIELD_NUMBER: _ClassVar[int]
    MAX_TOKENS_FIELD_NUMBER: _ClassVar[int]
    P50_COST_USD_FIELD_NUMBER: _ClassVar[int]
    P90_COST_USD_FIELD_NUMBER: _ClassVar[int]
    P95_COST_USD_FIELD_NUMBER: _ClassVar[int]
    P99_COST_USD_FIELD_NUMBER: _ClassVar[int]
    MAX_COST_USD_FIELD_NUMBER: _ClassVar[int]
    TOKEN_BUCKETS_FIELD_NUMBER: _ClassVar[int]
    DISTRIBUTION_SAMPLE_SIZE_FIELD_NUMBER: _ClassVar[int]
    DISTRIBUTION_UNOBSERVED_RUNS_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    total_cost_usd: float
    average_cost_usd: float
    total_tokens: int
    total_runs: int
    executed_query: str
    validity: MeasureValidity
    input_tokens: int
    output_tokens: int
    cache_read_tokens: int
    cache_creation_tokens: int
    input_cost_usd: float
    output_cost_usd: float
    cache_read_cost_usd: float
    cache_creation_cost_usd: float
    charge_by_basis: _containers.RepeatedCompositeFieldContainer[ChargeByBasis]
    total_charge_micro_usd: int
    unpriced_token_count: int
    p50_tokens: int
    p90_tokens: int
    p95_tokens: int
    p99_tokens: int
    max_tokens: int
    p50_cost_usd: float
    p90_cost_usd: float
    p95_cost_usd: float
    p99_cost_usd: float
    max_cost_usd: float
    token_buckets: _containers.RepeatedCompositeFieldContainer[RunTokenBucket]
    distribution_sample_size: int
    distribution_unobserved_runs: int
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, total_cost_usd: _Optional[float] = ..., average_cost_usd: _Optional[float] = ..., total_tokens: _Optional[int] = ..., total_runs: _Optional[int] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., input_tokens: _Optional[int] = ..., output_tokens: _Optional[int] = ..., cache_read_tokens: _Optional[int] = ..., cache_creation_tokens: _Optional[int] = ..., input_cost_usd: _Optional[float] = ..., output_cost_usd: _Optional[float] = ..., cache_read_cost_usd: _Optional[float] = ..., cache_creation_cost_usd: _Optional[float] = ..., charge_by_basis: _Optional[_Iterable[_Union[ChargeByBasis, _Mapping]]] = ..., total_charge_micro_usd: _Optional[int] = ..., unpriced_token_count: _Optional[int] = ..., p50_tokens: _Optional[int] = ..., p90_tokens: _Optional[int] = ..., p95_tokens: _Optional[int] = ..., p99_tokens: _Optional[int] = ..., max_tokens: _Optional[int] = ..., p50_cost_usd: _Optional[float] = ..., p90_cost_usd: _Optional[float] = ..., p95_cost_usd: _Optional[float] = ..., p99_cost_usd: _Optional[float] = ..., max_cost_usd: _Optional[float] = ..., token_buckets: _Optional[_Iterable[_Union[RunTokenBucket, _Mapping]]] = ..., distribution_sample_size: _Optional[int] = ..., distribution_unobserved_runs: _Optional[int] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class RunVolumeRequest(_message.Message):
    __slots__ = ("window", "filter")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ...) -> None: ...

class RunVolumeResponse(_message.Message):
    __slots__ = ("total_runs", "terminal_runs", "executed_query", "validity", "provenance", "definition_id", "history_floor", "outside_history_run_count")
    TOTAL_RUNS_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_RUNS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    HISTORY_FLOOR_FIELD_NUMBER: _ClassVar[int]
    OUTSIDE_HISTORY_RUN_COUNT_FIELD_NUMBER: _ClassVar[int]
    total_runs: int
    terminal_runs: int
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    history_floor: str
    outside_history_run_count: int
    def __init__(self, total_runs: _Optional[int] = ..., terminal_runs: _Optional[int] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ..., history_floor: _Optional[str] = ..., outside_history_run_count: _Optional[int] = ...) -> None: ...

class RunStatusDistributionRequest(_message.Message):
    __slots__ = ("window", "filter")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ...) -> None: ...

class RunStatusCount(_message.Message):
    __slots__ = ("status", "count")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    status: str
    count: int
    def __init__(self, status: _Optional[str] = ..., count: _Optional[int] = ...) -> None: ...

class RunStatusDistributionResponse(_message.Message):
    __slots__ = ("rows", "executed_query", "validity", "provenance", "definition_id")
    ROWS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[RunStatusCount]
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, rows: _Optional[_Iterable[_Union[RunStatusCount, _Mapping]]] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class RunBreakdownRow(_message.Message):
    __slots__ = ("value", "run_count", "success_count", "failed_count", "total_cost_usd", "average_duration_ms", "total_tokens", "key", "total_charge_micro_usd", "consumption_per_successful_completion", "completion_rate", "input_tokens", "output_tokens", "cache_read_tokens", "cache_creation_tokens")
    VALUE_FIELD_NUMBER: _ClassVar[int]
    RUN_COUNT_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_COUNT_FIELD_NUMBER: _ClassVar[int]
    FAILED_COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_COST_USD_FIELD_NUMBER: _ClassVar[int]
    AVERAGE_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_TOKENS_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    TOTAL_CHARGE_MICRO_USD_FIELD_NUMBER: _ClassVar[int]
    CONSUMPTION_PER_SUCCESSFUL_COMPLETION_FIELD_NUMBER: _ClassVar[int]
    COMPLETION_RATE_FIELD_NUMBER: _ClassVar[int]
    INPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    CACHE_READ_TOKENS_FIELD_NUMBER: _ClassVar[int]
    CACHE_CREATION_TOKENS_FIELD_NUMBER: _ClassVar[int]
    value: str
    run_count: int
    success_count: int
    failed_count: int
    total_cost_usd: float
    average_duration_ms: float
    total_tokens: int
    key: str
    total_charge_micro_usd: int
    consumption_per_successful_completion: float
    completion_rate: float
    input_tokens: int
    output_tokens: int
    cache_read_tokens: int
    cache_creation_tokens: int
    def __init__(self, value: _Optional[str] = ..., run_count: _Optional[int] = ..., success_count: _Optional[int] = ..., failed_count: _Optional[int] = ..., total_cost_usd: _Optional[float] = ..., average_duration_ms: _Optional[float] = ..., total_tokens: _Optional[int] = ..., key: _Optional[str] = ..., total_charge_micro_usd: _Optional[int] = ..., consumption_per_successful_completion: _Optional[float] = ..., completion_rate: _Optional[float] = ..., input_tokens: _Optional[int] = ..., output_tokens: _Optional[int] = ..., cache_read_tokens: _Optional[int] = ..., cache_creation_tokens: _Optional[int] = ...) -> None: ...

class RunnerBreakdownRequest(_message.Message):
    __slots__ = ("window", "filter")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ...) -> None: ...

class RunnerBreakdownResponse(_message.Message):
    __slots__ = ("rows", "executed_query", "validity", "provenance", "definition_id")
    ROWS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[RunBreakdownRow]
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, rows: _Optional[_Iterable[_Union[RunBreakdownRow, _Mapping]]] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class ModelBreakdownRequest(_message.Message):
    __slots__ = ("window", "filter")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ...) -> None: ...

class ModelBreakdownResponse(_message.Message):
    __slots__ = ("rows", "executed_query", "validity", "provenance", "definition_id")
    ROWS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[RunBreakdownRow]
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, rows: _Optional[_Iterable[_Union[RunBreakdownRow, _Mapping]]] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class ProfileBreakdownRequest(_message.Message):
    __slots__ = ("window", "filter")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ...) -> None: ...

class ProfileBreakdownResponse(_message.Message):
    __slots__ = ("rows", "executed_query", "validity", "provenance", "definition_id")
    ROWS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[RunBreakdownRow]
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, rows: _Optional[_Iterable[_Union[RunBreakdownRow, _Mapping]]] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class WorkloadBreakdownRequest(_message.Message):
    __slots__ = ("window", "filter", "allocate_subscription", "allocation_basis")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    ALLOCATE_SUBSCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ALLOCATION_BASIS_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    allocate_subscription: bool
    allocation_basis: str
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ..., allocate_subscription: _Optional[bool] = ..., allocation_basis: _Optional[str] = ...) -> None: ...

class WorkloadBreakdownResponse(_message.Message):
    __slots__ = ("rows", "executed_query", "validity", "provenance", "definition_id")
    ROWS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[RunBreakdownRow]
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, rows: _Optional[_Iterable[_Union[RunBreakdownRow, _Mapping]]] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class WorkloadEfficiencyRequest(_message.Message):
    __slots__ = ("window", "filter", "allocate_subscription", "allocation_basis")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    ALLOCATE_SUBSCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ALLOCATION_BASIS_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    allocate_subscription: bool
    allocation_basis: str
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ..., allocate_subscription: _Optional[bool] = ..., allocation_basis: _Optional[str] = ...) -> None: ...

class WorkloadEfficiencyResponse(_message.Message):
    __slots__ = ("consumption_per_successful_completion", "completion_rate", "total_tokens", "terminal_runs", "successful_runs", "observational_limitation", "executed_query", "validity", "provenance", "definition_id")
    CONSUMPTION_PER_SUCCESSFUL_COMPLETION_FIELD_NUMBER: _ClassVar[int]
    COMPLETION_RATE_FIELD_NUMBER: _ClassVar[int]
    TOTAL_TOKENS_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_RUNS_FIELD_NUMBER: _ClassVar[int]
    SUCCESSFUL_RUNS_FIELD_NUMBER: _ClassVar[int]
    OBSERVATIONAL_LIMITATION_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    consumption_per_successful_completion: float
    completion_rate: float
    total_tokens: int
    terminal_runs: int
    successful_runs: int
    observational_limitation: str
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, consumption_per_successful_completion: _Optional[float] = ..., completion_rate: _Optional[float] = ..., total_tokens: _Optional[int] = ..., terminal_runs: _Optional[int] = ..., successful_runs: _Optional[int] = ..., observational_limitation: _Optional[str] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class TerminalRunTrendRequest(_message.Message):
    __slots__ = ("window", "filter")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ...) -> None: ...

class TerminalRunTrendRow(_message.Message):
    __slots__ = ("bucket", "terminal_runs", "completed_runs", "failed_runs", "cancelled_runs", "total_cost_usd", "average_duration_ms")
    BUCKET_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_RUNS_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_RUNS_FIELD_NUMBER: _ClassVar[int]
    FAILED_RUNS_FIELD_NUMBER: _ClassVar[int]
    CANCELLED_RUNS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_COST_USD_FIELD_NUMBER: _ClassVar[int]
    AVERAGE_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    bucket: str
    terminal_runs: int
    completed_runs: int
    failed_runs: int
    cancelled_runs: int
    total_cost_usd: float
    average_duration_ms: float
    def __init__(self, bucket: _Optional[str] = ..., terminal_runs: _Optional[int] = ..., completed_runs: _Optional[int] = ..., failed_runs: _Optional[int] = ..., cancelled_runs: _Optional[int] = ..., total_cost_usd: _Optional[float] = ..., average_duration_ms: _Optional[float] = ...) -> None: ...

class TerminalRunTrendResponse(_message.Message):
    __slots__ = ("rows", "executed_query", "validity", "provenance", "definition_id")
    ROWS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[TerminalRunTrendRow]
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, rows: _Optional[_Iterable[_Union[TerminalRunTrendRow, _Mapping]]] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class ToolUsageRequest(_message.Message):
    __slots__ = ("window", "filter")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ...) -> None: ...

class ToolUsageRow(_message.Message):
    __slots__ = ("tool_name", "call_count", "success_count", "failed_count", "total_tokens", "estimated_token_share")
    TOOL_NAME_FIELD_NUMBER: _ClassVar[int]
    CALL_COUNT_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_COUNT_FIELD_NUMBER: _ClassVar[int]
    FAILED_COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_TOKENS_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_TOKEN_SHARE_FIELD_NUMBER: _ClassVar[int]
    tool_name: str
    call_count: int
    success_count: int
    failed_count: int
    total_tokens: int
    estimated_token_share: float
    def __init__(self, tool_name: _Optional[str] = ..., call_count: _Optional[int] = ..., success_count: _Optional[int] = ..., failed_count: _Optional[int] = ..., total_tokens: _Optional[int] = ..., estimated_token_share: _Optional[float] = ...) -> None: ...

class ToolUsageResponse(_message.Message):
    __slots__ = ("rows", "executed_query", "validity", "provenance", "definition_id")
    ROWS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[ToolUsageRow]
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, rows: _Optional[_Iterable[_Union[ToolUsageRow, _Mapping]]] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class ToolCommandBreakdownRequest(_message.Message):
    __slots__ = ("window", "filter", "limit")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    limit: int
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ..., limit: _Optional[int] = ...) -> None: ...

class ToolCommandBreakdownRow(_message.Message):
    __slots__ = ("executable", "command_path", "call_count", "success_count", "failed_count", "run_count", "truncated", "total_tokens", "estimated_token_share", "p50_footprint_tokens", "p95_footprint_tokens", "max_footprint_tokens")
    EXECUTABLE_FIELD_NUMBER: _ClassVar[int]
    COMMAND_PATH_FIELD_NUMBER: _ClassVar[int]
    CALL_COUNT_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_COUNT_FIELD_NUMBER: _ClassVar[int]
    FAILED_COUNT_FIELD_NUMBER: _ClassVar[int]
    RUN_COUNT_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    TOTAL_TOKENS_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_TOKEN_SHARE_FIELD_NUMBER: _ClassVar[int]
    P50_FOOTPRINT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    P95_FOOTPRINT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    MAX_FOOTPRINT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    executable: str
    command_path: str
    call_count: int
    success_count: int
    failed_count: int
    run_count: int
    truncated: bool
    total_tokens: int
    estimated_token_share: float
    p50_footprint_tokens: int
    p95_footprint_tokens: int
    max_footprint_tokens: int
    def __init__(self, executable: _Optional[str] = ..., command_path: _Optional[str] = ..., call_count: _Optional[int] = ..., success_count: _Optional[int] = ..., failed_count: _Optional[int] = ..., run_count: _Optional[int] = ..., truncated: _Optional[bool] = ..., total_tokens: _Optional[int] = ..., estimated_token_share: _Optional[float] = ..., p50_footprint_tokens: _Optional[int] = ..., p95_footprint_tokens: _Optional[int] = ..., max_footprint_tokens: _Optional[int] = ...) -> None: ...

class ToolCommandBreakdownResponse(_message.Message):
    __slots__ = ("rows", "executed_query", "validity", "provenance", "definition_id")
    ROWS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[ToolCommandBreakdownRow]
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, rows: _Optional[_Iterable[_Union[ToolCommandBreakdownRow, _Mapping]]] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class TokenAttributionRequest(_message.Message):
    __slots__ = ("window", "filter", "group_by", "view", "limit")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    GROUP_BY_FIELD_NUMBER: _ClassVar[int]
    VIEW_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    group_by: str
    view: str
    limit: int
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ..., group_by: _Optional[str] = ..., view: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class TokenAttributionRow(_message.Message):
    __slots__ = ("group_by", "value", "call_count", "total_tokens", "estimated_tokens", "estimated_token_share", "p50_footprint_tokens", "p95_footprint_tokens", "max_footprint_tokens")
    GROUP_BY_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    CALL_COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_TOKENS_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_TOKENS_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_TOKEN_SHARE_FIELD_NUMBER: _ClassVar[int]
    P50_FOOTPRINT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    P95_FOOTPRINT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    MAX_FOOTPRINT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    group_by: str
    value: str
    call_count: int
    total_tokens: int
    estimated_tokens: int
    estimated_token_share: float
    p50_footprint_tokens: int
    p95_footprint_tokens: int
    max_footprint_tokens: int
    def __init__(self, group_by: _Optional[str] = ..., value: _Optional[str] = ..., call_count: _Optional[int] = ..., total_tokens: _Optional[int] = ..., estimated_tokens: _Optional[int] = ..., estimated_token_share: _Optional[float] = ..., p50_footprint_tokens: _Optional[int] = ..., p95_footprint_tokens: _Optional[int] = ..., max_footprint_tokens: _Optional[int] = ...) -> None: ...

class TokenAttributionResponse(_message.Message):
    __slots__ = ("rows", "group_by", "view", "estimated_token_share", "executed_query", "validity", "provenance", "definition_id")
    ROWS_FIELD_NUMBER: _ClassVar[int]
    GROUP_BY_FIELD_NUMBER: _ClassVar[int]
    VIEW_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_TOKEN_SHARE_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[TokenAttributionRow]
    group_by: str
    view: str
    estimated_token_share: float
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, rows: _Optional[_Iterable[_Union[TokenAttributionRow, _Mapping]]] = ..., group_by: _Optional[str] = ..., view: _Optional[str] = ..., estimated_token_share: _Optional[float] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class CapabilityUsageRequest(_message.Message):
    __slots__ = ("window", "filter")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ...) -> None: ...

class CapabilityUsageRow(_message.Message):
    __slots__ = ("target_scenario", "operation", "call_count", "success_count", "failed_count", "total_duration_ms", "total_tokens", "estimated_token_share")
    TARGET_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    CALL_COUNT_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_COUNT_FIELD_NUMBER: _ClassVar[int]
    FAILED_COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_TOKENS_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_TOKEN_SHARE_FIELD_NUMBER: _ClassVar[int]
    target_scenario: str
    operation: str
    call_count: int
    success_count: int
    failed_count: int
    total_duration_ms: int
    total_tokens: int
    estimated_token_share: float
    def __init__(self, target_scenario: _Optional[str] = ..., operation: _Optional[str] = ..., call_count: _Optional[int] = ..., success_count: _Optional[int] = ..., failed_count: _Optional[int] = ..., total_duration_ms: _Optional[int] = ..., total_tokens: _Optional[int] = ..., estimated_token_share: _Optional[float] = ...) -> None: ...

class CapabilityUsageResponse(_message.Message):
    __slots__ = ("rows", "executed_query", "validity", "provenance", "definition_id")
    ROWS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[CapabilityUsageRow]
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, rows: _Optional[_Iterable[_Union[CapabilityUsageRow, _Mapping]]] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class CapabilityEfficacyRequest(_message.Message):
    __slots__ = ("window", "filter")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ...) -> None: ...

class CapabilityEfficacyRow(_message.Message):
    __slots__ = ("target_scenario", "operation", "call_count", "success_count", "fallback_after_count", "abandoned_count")
    TARGET_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    CALL_COUNT_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_COUNT_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_AFTER_COUNT_FIELD_NUMBER: _ClassVar[int]
    ABANDONED_COUNT_FIELD_NUMBER: _ClassVar[int]
    target_scenario: str
    operation: str
    call_count: int
    success_count: int
    fallback_after_count: int
    abandoned_count: int
    def __init__(self, target_scenario: _Optional[str] = ..., operation: _Optional[str] = ..., call_count: _Optional[int] = ..., success_count: _Optional[int] = ..., fallback_after_count: _Optional[int] = ..., abandoned_count: _Optional[int] = ...) -> None: ...

class CapabilityEfficacyResponse(_message.Message):
    __slots__ = ("rows", "executed_query", "validity", "provenance", "definition_id")
    ROWS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[CapabilityEfficacyRow]
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, rows: _Optional[_Iterable[_Union[CapabilityEfficacyRow, _Mapping]]] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class ErrorPatternsRequest(_message.Message):
    __slots__ = ("window", "filter")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ...) -> None: ...

class ErrorPatternRow(_message.Message):
    __slots__ = ("error_code", "count", "last_seen", "sample_run_id")
    ERROR_CODE_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    LAST_SEEN_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    error_code: str
    count: int
    last_seen: str
    sample_run_id: str
    def __init__(self, error_code: _Optional[str] = ..., count: _Optional[int] = ..., last_seen: _Optional[str] = ..., sample_run_id: _Optional[str] = ...) -> None: ...

class ErrorPatternsResponse(_message.Message):
    __slots__ = ("rows", "executed_query", "validity", "provenance", "definition_id")
    ROWS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[ErrorPatternRow]
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, rows: _Optional[_Iterable[_Union[ErrorPatternRow, _Mapping]]] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class FileRereadRateRequest(_message.Message):
    __slots__ = ("window", "filter")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ...) -> None: ...

class FileRereadRateResponse(_message.Message):
    __slots__ = ("rate", "files_read_more_than_once", "read_calls", "executed_query", "validity", "provenance", "definition_id")
    RATE_FIELD_NUMBER: _ClassVar[int]
    FILES_READ_MORE_THAN_ONCE_FIELD_NUMBER: _ClassVar[int]
    READ_CALLS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    rate: float
    files_read_more_than_once: int
    read_calls: int
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, rate: _Optional[float] = ..., files_read_more_than_once: _Optional[int] = ..., read_calls: _Optional[int] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class FindingRecurrenceRateRequest(_message.Message):
    __slots__ = ("window", "filter")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ...) -> None: ...

class FindingRecurrenceRateResponse(_message.Message):
    __slots__ = ("rate", "recurring_findings", "total_findings", "recurring_fingerprints", "executed_query", "validity", "provenance", "definition_id")
    RATE_FIELD_NUMBER: _ClassVar[int]
    RECURRING_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    RECURRING_FINGERPRINTS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    rate: float
    recurring_findings: int
    total_findings: int
    recurring_fingerprints: int
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, rate: _Optional[float] = ..., recurring_findings: _Optional[int] = ..., total_findings: _Optional[int] = ..., recurring_fingerprints: _Optional[int] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class CohortSelector(_message.Message):
    __slots__ = ("window", "filter", "cohort", "run_ids")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    COHORT_FIELD_NUMBER: _ClassVar[int]
    RUN_IDS_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    cohort: str
    run_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ..., cohort: _Optional[str] = ..., run_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class SelectCohortRequest(_message.Message):
    __slots__ = ("window", "filter", "limit", "cohort", "run_ids")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    COHORT_FIELD_NUMBER: _ClassVar[int]
    RUN_IDS_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    limit: int
    cohort: str
    run_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ..., limit: _Optional[int] = ..., cohort: _Optional[str] = ..., run_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class CohortRun(_message.Message):
    __slots__ = ("run_id", "task_title", "profile_id", "profile_name", "status", "created_at", "model", "runner_type", "workload_key", "total_tokens", "total_charge_micro_usd", "charge_basis", "tool_call_count")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_TITLE_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    RUNNER_TYPE_FIELD_NUMBER: _ClassVar[int]
    WORKLOAD_KEY_FIELD_NUMBER: _ClassVar[int]
    TOTAL_TOKENS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_CHARGE_MICRO_USD_FIELD_NUMBER: _ClassVar[int]
    CHARGE_BASIS_FIELD_NUMBER: _ClassVar[int]
    TOOL_CALL_COUNT_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    task_title: str
    profile_id: str
    profile_name: str
    status: str
    created_at: str
    model: str
    runner_type: str
    workload_key: str
    total_tokens: int
    total_charge_micro_usd: int
    charge_basis: str
    tool_call_count: int
    def __init__(self, run_id: _Optional[str] = ..., task_title: _Optional[str] = ..., profile_id: _Optional[str] = ..., profile_name: _Optional[str] = ..., status: _Optional[str] = ..., created_at: _Optional[str] = ..., model: _Optional[str] = ..., runner_type: _Optional[str] = ..., workload_key: _Optional[str] = ..., total_tokens: _Optional[int] = ..., total_charge_micro_usd: _Optional[int] = ..., charge_basis: _Optional[str] = ..., tool_call_count: _Optional[int] = ...) -> None: ...

class SelectCohortResponse(_message.Message):
    __slots__ = ("run_ids", "truncated", "executed_query", "validity", "rows", "provenance", "definition_id")
    RUN_IDS_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    ROWS_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    run_ids: _containers.RepeatedScalarFieldContainer[str]
    truncated: bool
    executed_query: str
    validity: MeasureValidity
    rows: _containers.RepeatedCompositeFieldContainer[CohortRun]
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, run_ids: _Optional[_Iterable[str]] = ..., truncated: _Optional[bool] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., rows: _Optional[_Iterable[_Union[CohortRun, _Mapping]]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class EpisodeCohortRequest(_message.Message):
    __slots__ = ("window", "filter", "limit", "cohort", "run_ids")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    COHORT_FIELD_NUMBER: _ClassVar[int]
    RUN_IDS_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    limit: int
    cohort: str
    run_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ..., limit: _Optional[int] = ..., cohort: _Optional[str] = ..., run_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class EpisodeCohortSignal(_message.Message):
    __slots__ = ("fingerprint", "occurrences", "distinct_runs", "summed_cost_ms", "confidence", "representative_run_ids")
    FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    OCCURRENCES_FIELD_NUMBER: _ClassVar[int]
    DISTINCT_RUNS_FIELD_NUMBER: _ClassVar[int]
    SUMMED_COST_MS_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    REPRESENTATIVE_RUN_IDS_FIELD_NUMBER: _ClassVar[int]
    fingerprint: str
    occurrences: int
    distinct_runs: int
    summed_cost_ms: int
    confidence: str
    representative_run_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, fingerprint: _Optional[str] = ..., occurrences: _Optional[int] = ..., distinct_runs: _Optional[int] = ..., summed_cost_ms: _Optional[int] = ..., confidence: _Optional[str] = ..., representative_run_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class EpisodeCohortResponse(_message.Message):
    __slots__ = ("availability_state", "availability_reason", "signals", "executed_query", "validity", "provenance", "definition_id")
    AVAILABILITY_STATE_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_REASON_FIELD_NUMBER: _ClassVar[int]
    SIGNALS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    availability_state: str
    availability_reason: str
    signals: _containers.RepeatedCompositeFieldContainer[EpisodeCohortSignal]
    executed_query: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, availability_state: _Optional[str] = ..., availability_reason: _Optional[str] = ..., signals: _Optional[_Iterable[_Union[EpisodeCohortSignal, _Mapping]]] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class CohortReportRequest(_message.Message):
    __slots__ = ("window", "filter", "limit", "cohort", "run_ids")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    COHORT_FIELD_NUMBER: _ClassVar[int]
    RUN_IDS_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    limit: int
    cohort: str
    run_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ..., limit: _Optional[int] = ..., cohort: _Optional[str] = ..., run_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class CohortReportSignal(_message.Message):
    __slots__ = ("kind", "count", "impact", "confidence", "representative_run_ids")
    KIND_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    IMPACT_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    REPRESENTATIVE_RUN_IDS_FIELD_NUMBER: _ClassVar[int]
    kind: str
    count: int
    impact: int
    confidence: str
    representative_run_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, kind: _Optional[str] = ..., count: _Optional[int] = ..., impact: _Optional[int] = ..., confidence: _Optional[str] = ..., representative_run_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class CohortTimeAccounting(_message.Message):
    __slots__ = ("model_generating_ms", "tool_executing_ms", "idle_waiting_ms", "awaiting_human_ms", "unattributable_ms", "model_tokens", "tool_tokens", "idle_tokens", "human_tokens", "unattributable_tokens")
    MODEL_GENERATING_MS_FIELD_NUMBER: _ClassVar[int]
    TOOL_EXECUTING_MS_FIELD_NUMBER: _ClassVar[int]
    IDLE_WAITING_MS_FIELD_NUMBER: _ClassVar[int]
    AWAITING_HUMAN_MS_FIELD_NUMBER: _ClassVar[int]
    UNATTRIBUTABLE_MS_FIELD_NUMBER: _ClassVar[int]
    MODEL_TOKENS_FIELD_NUMBER: _ClassVar[int]
    TOOL_TOKENS_FIELD_NUMBER: _ClassVar[int]
    IDLE_TOKENS_FIELD_NUMBER: _ClassVar[int]
    HUMAN_TOKENS_FIELD_NUMBER: _ClassVar[int]
    UNATTRIBUTABLE_TOKENS_FIELD_NUMBER: _ClassVar[int]
    model_generating_ms: int
    tool_executing_ms: int
    idle_waiting_ms: int
    awaiting_human_ms: int
    unattributable_ms: int
    model_tokens: int
    tool_tokens: int
    idle_tokens: int
    human_tokens: int
    unattributable_tokens: int
    def __init__(self, model_generating_ms: _Optional[int] = ..., tool_executing_ms: _Optional[int] = ..., idle_waiting_ms: _Optional[int] = ..., awaiting_human_ms: _Optional[int] = ..., unattributable_ms: _Optional[int] = ..., model_tokens: _Optional[int] = ..., tool_tokens: _Optional[int] = ..., idle_tokens: _Optional[int] = ..., human_tokens: _Optional[int] = ..., unattributable_tokens: _Optional[int] = ...) -> None: ...

class CohortInvocationValidity(_message.Message):
    __slots__ = ("total_calls", "classified_base", "unclassified_count")
    TOTAL_CALLS_FIELD_NUMBER: _ClassVar[int]
    CLASSIFIED_BASE_FIELD_NUMBER: _ClassVar[int]
    UNCLASSIFIED_COUNT_FIELD_NUMBER: _ClassVar[int]
    total_calls: int
    classified_base: int
    unclassified_count: int
    def __init__(self, total_calls: _Optional[int] = ..., classified_base: _Optional[int] = ..., unclassified_count: _Optional[int] = ...) -> None: ...

class CohortGoalProgression(_message.Message):
    __slots__ = ("availability_state", "availability_reason", "goal_runs", "met_runs", "average_time_to_first_met_ms", "average_tokens_to_first_met", "interventions_before_met", "handoffs_before_met", "regressions", "restatements")
    AVAILABILITY_STATE_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_REASON_FIELD_NUMBER: _ClassVar[int]
    GOAL_RUNS_FIELD_NUMBER: _ClassVar[int]
    MET_RUNS_FIELD_NUMBER: _ClassVar[int]
    AVERAGE_TIME_TO_FIRST_MET_MS_FIELD_NUMBER: _ClassVar[int]
    AVERAGE_TOKENS_TO_FIRST_MET_FIELD_NUMBER: _ClassVar[int]
    INTERVENTIONS_BEFORE_MET_FIELD_NUMBER: _ClassVar[int]
    HANDOFFS_BEFORE_MET_FIELD_NUMBER: _ClassVar[int]
    REGRESSIONS_FIELD_NUMBER: _ClassVar[int]
    RESTATEMENTS_FIELD_NUMBER: _ClassVar[int]
    availability_state: str
    availability_reason: str
    goal_runs: int
    met_runs: int
    average_time_to_first_met_ms: float
    average_tokens_to_first_met: float
    interventions_before_met: int
    handoffs_before_met: int
    regressions: int
    restatements: int
    def __init__(self, availability_state: _Optional[str] = ..., availability_reason: _Optional[str] = ..., goal_runs: _Optional[int] = ..., met_runs: _Optional[int] = ..., average_time_to_first_met_ms: _Optional[float] = ..., average_tokens_to_first_met: _Optional[float] = ..., interventions_before_met: _Optional[int] = ..., handoffs_before_met: _Optional[int] = ..., regressions: _Optional[int] = ..., restatements: _Optional[int] = ...) -> None: ...

class CohortReportResponse(_message.Message):
    __slots__ = ("classifier_version", "run_ids", "truncated", "availability_state", "availability_reason", "signals", "time_accounting", "invocation_validity", "goal", "validity", "provenance", "definition_id")
    CLASSIFIER_VERSION_FIELD_NUMBER: _ClassVar[int]
    RUN_IDS_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_STATE_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_REASON_FIELD_NUMBER: _ClassVar[int]
    SIGNALS_FIELD_NUMBER: _ClassVar[int]
    TIME_ACCOUNTING_FIELD_NUMBER: _ClassVar[int]
    INVOCATION_VALIDITY_FIELD_NUMBER: _ClassVar[int]
    GOAL_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    classifier_version: str
    run_ids: _containers.RepeatedScalarFieldContainer[str]
    truncated: bool
    availability_state: str
    availability_reason: str
    signals: _containers.RepeatedCompositeFieldContainer[CohortReportSignal]
    time_accounting: CohortTimeAccounting
    invocation_validity: CohortInvocationValidity
    goal: CohortGoalProgression
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, classifier_version: _Optional[str] = ..., run_ids: _Optional[_Iterable[str]] = ..., truncated: _Optional[bool] = ..., availability_state: _Optional[str] = ..., availability_reason: _Optional[str] = ..., signals: _Optional[_Iterable[_Union[CohortReportSignal, _Mapping]]] = ..., time_accounting: _Optional[_Union[CohortTimeAccounting, _Mapping]] = ..., invocation_validity: _Optional[_Union[CohortInvocationValidity, _Mapping]] = ..., goal: _Optional[_Union[CohortGoalProgression, _Mapping]] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class CompareEpisodeCohortsRequest(_message.Message):
    __slots__ = ("left", "right", "change_binding", "limit")
    LEFT_FIELD_NUMBER: _ClassVar[int]
    RIGHT_FIELD_NUMBER: _ClassVar[int]
    CHANGE_BINDING_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    left: CohortSelector
    right: CohortSelector
    change_binding: str
    limit: int
    def __init__(self, left: _Optional[_Union[CohortSelector, _Mapping]] = ..., right: _Optional[_Union[CohortSelector, _Mapping]] = ..., change_binding: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class EpisodeComparisonSignal(_message.Message):
    __slots__ = ("fingerprint", "left_occurrences", "right_occurrences", "left_distinct_runs", "right_distinct_runs", "left_cost_ms", "right_cost_ms", "occurrence_delta", "cost_delta_ms")
    FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    LEFT_OCCURRENCES_FIELD_NUMBER: _ClassVar[int]
    RIGHT_OCCURRENCES_FIELD_NUMBER: _ClassVar[int]
    LEFT_DISTINCT_RUNS_FIELD_NUMBER: _ClassVar[int]
    RIGHT_DISTINCT_RUNS_FIELD_NUMBER: _ClassVar[int]
    LEFT_COST_MS_FIELD_NUMBER: _ClassVar[int]
    RIGHT_COST_MS_FIELD_NUMBER: _ClassVar[int]
    OCCURRENCE_DELTA_FIELD_NUMBER: _ClassVar[int]
    COST_DELTA_MS_FIELD_NUMBER: _ClassVar[int]
    fingerprint: str
    left_occurrences: int
    right_occurrences: int
    left_distinct_runs: int
    right_distinct_runs: int
    left_cost_ms: int
    right_cost_ms: int
    occurrence_delta: int
    cost_delta_ms: int
    def __init__(self, fingerprint: _Optional[str] = ..., left_occurrences: _Optional[int] = ..., right_occurrences: _Optional[int] = ..., left_distinct_runs: _Optional[int] = ..., right_distinct_runs: _Optional[int] = ..., left_cost_ms: _Optional[int] = ..., right_cost_ms: _Optional[int] = ..., occurrence_delta: _Optional[int] = ..., cost_delta_ms: _Optional[int] = ...) -> None: ...

class CompareEpisodeCohortsResponse(_message.Message):
    __slots__ = ("classifier_version", "change_binding", "left_population", "right_population", "availability_state", "availability_reason", "signals", "validity", "provenance", "definition_id")
    CLASSIFIER_VERSION_FIELD_NUMBER: _ClassVar[int]
    CHANGE_BINDING_FIELD_NUMBER: _ClassVar[int]
    LEFT_POPULATION_FIELD_NUMBER: _ClassVar[int]
    RIGHT_POPULATION_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_STATE_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_REASON_FIELD_NUMBER: _ClassVar[int]
    SIGNALS_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    classifier_version: str
    change_binding: str
    left_population: int
    right_population: int
    availability_state: str
    availability_reason: str
    signals: _containers.RepeatedCompositeFieldContainer[EpisodeComparisonSignal]
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, classifier_version: _Optional[str] = ..., change_binding: _Optional[str] = ..., left_population: _Optional[int] = ..., right_population: _Optional[int] = ..., availability_state: _Optional[str] = ..., availability_reason: _Optional[str] = ..., signals: _Optional[_Iterable[_Union[EpisodeComparisonSignal, _Mapping]]] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class UsageForecastRequest(_message.Message):
    __slots__ = ("provider",)
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    provider: str
    def __init__(self, provider: _Optional[str] = ...) -> None: ...

class UsageBurnRate(_message.Message):
    __slots__ = ("horizon", "available", "percent_per_hour", "span_hours", "stale")
    HORIZON_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    PERCENT_PER_HOUR_FIELD_NUMBER: _ClassVar[int]
    SPAN_HOURS_FIELD_NUMBER: _ClassVar[int]
    STALE_FIELD_NUMBER: _ClassVar[int]
    horizon: str
    available: bool
    percent_per_hour: float
    span_hours: float
    stale: bool
    def __init__(self, horizon: _Optional[str] = ..., available: _Optional[bool] = ..., percent_per_hour: _Optional[float] = ..., span_hours: _Optional[float] = ..., stale: _Optional[bool] = ...) -> None: ...

class UsagePercentPoint(_message.Message):
    __slots__ = ("at", "used_percent")
    AT_FIELD_NUMBER: _ClassVar[int]
    USED_PERCENT_FIELD_NUMBER: _ClassVar[int]
    at: str
    used_percent: float
    def __init__(self, at: _Optional[str] = ..., used_percent: _Optional[float] = ...) -> None: ...

class UsageWindowForecast(_message.Message):
    __slots__ = ("provider", "pool", "window", "window_minutes", "status", "status_reason", "has_percent", "used_percent", "standing", "window_start", "resets_at", "observed_at", "sample_count", "burn", "central_percent_per_hour", "projected_exhaustion_at", "exhaustion_band_start", "exhaustion_band_end", "lasting_percent_per_hour", "early_reset", "previous_resets_at", "previous_final_percent", "previous_ended_at", "series", "last_reading_percent", "calibration_factor", "calibration_windows")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    POOL_FIELD_NUMBER: _ClassVar[int]
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    WINDOW_MINUTES_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STATUS_REASON_FIELD_NUMBER: _ClassVar[int]
    HAS_PERCENT_FIELD_NUMBER: _ClassVar[int]
    USED_PERCENT_FIELD_NUMBER: _ClassVar[int]
    STANDING_FIELD_NUMBER: _ClassVar[int]
    WINDOW_START_FIELD_NUMBER: _ClassVar[int]
    RESETS_AT_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    BURN_FIELD_NUMBER: _ClassVar[int]
    CENTRAL_PERCENT_PER_HOUR_FIELD_NUMBER: _ClassVar[int]
    PROJECTED_EXHAUSTION_AT_FIELD_NUMBER: _ClassVar[int]
    EXHAUSTION_BAND_START_FIELD_NUMBER: _ClassVar[int]
    EXHAUSTION_BAND_END_FIELD_NUMBER: _ClassVar[int]
    LASTING_PERCENT_PER_HOUR_FIELD_NUMBER: _ClassVar[int]
    EARLY_RESET_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_RESETS_AT_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_FINAL_PERCENT_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_ENDED_AT_FIELD_NUMBER: _ClassVar[int]
    SERIES_FIELD_NUMBER: _ClassVar[int]
    LAST_READING_PERCENT_FIELD_NUMBER: _ClassVar[int]
    CALIBRATION_FACTOR_FIELD_NUMBER: _ClassVar[int]
    CALIBRATION_WINDOWS_FIELD_NUMBER: _ClassVar[int]
    provider: str
    pool: str
    window: str
    window_minutes: int
    status: str
    status_reason: str
    has_percent: bool
    used_percent: float
    standing: str
    window_start: str
    resets_at: str
    observed_at: str
    sample_count: int
    burn: _containers.RepeatedCompositeFieldContainer[UsageBurnRate]
    central_percent_per_hour: float
    projected_exhaustion_at: str
    exhaustion_band_start: str
    exhaustion_band_end: str
    lasting_percent_per_hour: float
    early_reset: bool
    previous_resets_at: str
    previous_final_percent: float
    previous_ended_at: str
    series: _containers.RepeatedCompositeFieldContainer[UsagePercentPoint]
    last_reading_percent: float
    calibration_factor: float
    calibration_windows: int
    def __init__(self, provider: _Optional[str] = ..., pool: _Optional[str] = ..., window: _Optional[str] = ..., window_minutes: _Optional[int] = ..., status: _Optional[str] = ..., status_reason: _Optional[str] = ..., has_percent: _Optional[bool] = ..., used_percent: _Optional[float] = ..., standing: _Optional[str] = ..., window_start: _Optional[str] = ..., resets_at: _Optional[str] = ..., observed_at: _Optional[str] = ..., sample_count: _Optional[int] = ..., burn: _Optional[_Iterable[_Union[UsageBurnRate, _Mapping]]] = ..., central_percent_per_hour: _Optional[float] = ..., projected_exhaustion_at: _Optional[str] = ..., exhaustion_band_start: _Optional[str] = ..., exhaustion_band_end: _Optional[str] = ..., lasting_percent_per_hour: _Optional[float] = ..., early_reset: _Optional[bool] = ..., previous_resets_at: _Optional[str] = ..., previous_final_percent: _Optional[float] = ..., previous_ended_at: _Optional[str] = ..., series: _Optional[_Iterable[_Union[UsagePercentPoint, _Mapping]]] = ..., last_reading_percent: _Optional[float] = ..., calibration_factor: _Optional[float] = ..., calibration_windows: _Optional[int] = ...) -> None: ...

class UsageForecastResponse(_message.Message):
    __slots__ = ("windows", "generated_at", "provenance", "definition_id")
    WINDOWS_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    windows: _containers.RepeatedCompositeFieldContainer[UsageWindowForecast]
    generated_at: str
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, windows: _Optional[_Iterable[_Union[UsageWindowForecast, _Mapping]]] = ..., generated_at: _Optional[str] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class UsageForecastBacktestRequest(_message.Message):
    __slots__ = ("provider",)
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    provider: str
    def __init__(self, provider: _Optional[str] = ...) -> None: ...

class UsageBacktestCheckpoint(_message.Message):
    __slots__ = ("elapsed_fraction", "as_of", "used_percent_at", "predicted_final_percent", "baseline_predicted_final_percent", "abs_error_points", "baseline_abs_error_points", "predicted_exhaustion_at", "baseline_predicted_exhaustion_at", "abs_error_hours", "baseline_abs_error_hours", "outcome_correct", "baseline_outcome_correct", "calibration_factor", "calibration_windows")
    ELAPSED_FRACTION_FIELD_NUMBER: _ClassVar[int]
    AS_OF_FIELD_NUMBER: _ClassVar[int]
    USED_PERCENT_AT_FIELD_NUMBER: _ClassVar[int]
    PREDICTED_FINAL_PERCENT_FIELD_NUMBER: _ClassVar[int]
    BASELINE_PREDICTED_FINAL_PERCENT_FIELD_NUMBER: _ClassVar[int]
    ABS_ERROR_POINTS_FIELD_NUMBER: _ClassVar[int]
    BASELINE_ABS_ERROR_POINTS_FIELD_NUMBER: _ClassVar[int]
    PREDICTED_EXHAUSTION_AT_FIELD_NUMBER: _ClassVar[int]
    BASELINE_PREDICTED_EXHAUSTION_AT_FIELD_NUMBER: _ClassVar[int]
    ABS_ERROR_HOURS_FIELD_NUMBER: _ClassVar[int]
    BASELINE_ABS_ERROR_HOURS_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_CORRECT_FIELD_NUMBER: _ClassVar[int]
    BASELINE_OUTCOME_CORRECT_FIELD_NUMBER: _ClassVar[int]
    CALIBRATION_FACTOR_FIELD_NUMBER: _ClassVar[int]
    CALIBRATION_WINDOWS_FIELD_NUMBER: _ClassVar[int]
    elapsed_fraction: float
    as_of: str
    used_percent_at: float
    predicted_final_percent: float
    baseline_predicted_final_percent: float
    abs_error_points: float
    baseline_abs_error_points: float
    predicted_exhaustion_at: str
    baseline_predicted_exhaustion_at: str
    abs_error_hours: float
    baseline_abs_error_hours: float
    outcome_correct: bool
    baseline_outcome_correct: bool
    calibration_factor: float
    calibration_windows: int
    def __init__(self, elapsed_fraction: _Optional[float] = ..., as_of: _Optional[str] = ..., used_percent_at: _Optional[float] = ..., predicted_final_percent: _Optional[float] = ..., baseline_predicted_final_percent: _Optional[float] = ..., abs_error_points: _Optional[float] = ..., baseline_abs_error_points: _Optional[float] = ..., predicted_exhaustion_at: _Optional[str] = ..., baseline_predicted_exhaustion_at: _Optional[str] = ..., abs_error_hours: _Optional[float] = ..., baseline_abs_error_hours: _Optional[float] = ..., outcome_correct: _Optional[bool] = ..., baseline_outcome_correct: _Optional[bool] = ..., calibration_factor: _Optional[float] = ..., calibration_windows: _Optional[int] = ...) -> None: ...

class UsageBacktestWindow(_message.Message):
    __slots__ = ("provider", "pool", "window", "window_start", "ended_at", "resets_at", "early_reset", "final_percent", "actual_exhaustion_at", "checkpoints", "outcome_observed")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    POOL_FIELD_NUMBER: _ClassVar[int]
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    WINDOW_START_FIELD_NUMBER: _ClassVar[int]
    ENDED_AT_FIELD_NUMBER: _ClassVar[int]
    RESETS_AT_FIELD_NUMBER: _ClassVar[int]
    EARLY_RESET_FIELD_NUMBER: _ClassVar[int]
    FINAL_PERCENT_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_EXHAUSTION_AT_FIELD_NUMBER: _ClassVar[int]
    CHECKPOINTS_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_OBSERVED_FIELD_NUMBER: _ClassVar[int]
    provider: str
    pool: str
    window: str
    window_start: str
    ended_at: str
    resets_at: str
    early_reset: bool
    final_percent: float
    actual_exhaustion_at: str
    checkpoints: _containers.RepeatedCompositeFieldContainer[UsageBacktestCheckpoint]
    outcome_observed: bool
    def __init__(self, provider: _Optional[str] = ..., pool: _Optional[str] = ..., window: _Optional[str] = ..., window_start: _Optional[str] = ..., ended_at: _Optional[str] = ..., resets_at: _Optional[str] = ..., early_reset: _Optional[bool] = ..., final_percent: _Optional[float] = ..., actual_exhaustion_at: _Optional[str] = ..., checkpoints: _Optional[_Iterable[_Union[UsageBacktestCheckpoint, _Mapping]]] = ..., outcome_observed: _Optional[bool] = ...) -> None: ...

class UsageBacktestSummary(_message.Message):
    __slots__ = ("population", "elapsed_fraction", "windows", "median_abs_error_points", "baseline_median_abs_error_points", "hour_error_windows", "median_abs_error_hours", "baseline_median_abs_error_hours", "outcome_accuracy", "baseline_outcome_accuracy", "beats_baseline")
    POPULATION_FIELD_NUMBER: _ClassVar[int]
    ELAPSED_FRACTION_FIELD_NUMBER: _ClassVar[int]
    WINDOWS_FIELD_NUMBER: _ClassVar[int]
    MEDIAN_ABS_ERROR_POINTS_FIELD_NUMBER: _ClassVar[int]
    BASELINE_MEDIAN_ABS_ERROR_POINTS_FIELD_NUMBER: _ClassVar[int]
    HOUR_ERROR_WINDOWS_FIELD_NUMBER: _ClassVar[int]
    MEDIAN_ABS_ERROR_HOURS_FIELD_NUMBER: _ClassVar[int]
    BASELINE_MEDIAN_ABS_ERROR_HOURS_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_ACCURACY_FIELD_NUMBER: _ClassVar[int]
    BASELINE_OUTCOME_ACCURACY_FIELD_NUMBER: _ClassVar[int]
    BEATS_BASELINE_FIELD_NUMBER: _ClassVar[int]
    population: str
    elapsed_fraction: float
    windows: int
    median_abs_error_points: float
    baseline_median_abs_error_points: float
    hour_error_windows: int
    median_abs_error_hours: float
    baseline_median_abs_error_hours: float
    outcome_accuracy: float
    baseline_outcome_accuracy: float
    beats_baseline: bool
    def __init__(self, population: _Optional[str] = ..., elapsed_fraction: _Optional[float] = ..., windows: _Optional[int] = ..., median_abs_error_points: _Optional[float] = ..., baseline_median_abs_error_points: _Optional[float] = ..., hour_error_windows: _Optional[int] = ..., median_abs_error_hours: _Optional[float] = ..., baseline_median_abs_error_hours: _Optional[float] = ..., outcome_accuracy: _Optional[float] = ..., baseline_outcome_accuracy: _Optional[float] = ..., beats_baseline: _Optional[bool] = ...) -> None: ...

class UsageForecastBacktestResponse(_message.Message):
    __slots__ = ("windows", "summaries", "limitation", "unobserved_windows", "validity", "provenance", "definition_id")
    WINDOWS_FIELD_NUMBER: _ClassVar[int]
    SUMMARIES_FIELD_NUMBER: _ClassVar[int]
    LIMITATION_FIELD_NUMBER: _ClassVar[int]
    UNOBSERVED_WINDOWS_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    windows: _containers.RepeatedCompositeFieldContainer[UsageBacktestWindow]
    summaries: _containers.RepeatedCompositeFieldContainer[UsageBacktestSummary]
    limitation: str
    unobserved_windows: int
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, windows: _Optional[_Iterable[_Union[UsageBacktestWindow, _Mapping]]] = ..., summaries: _Optional[_Iterable[_Union[UsageBacktestSummary, _Mapping]]] = ..., limitation: _Optional[str] = ..., unobserved_windows: _Optional[int] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class UsageBurnersRequest(_message.Message):
    __slots__ = ("provider", "group_by", "since", "run_id", "limit")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    GROUP_BY_FIELD_NUMBER: _ClassVar[int]
    SINCE_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    provider: str
    group_by: str
    since: str
    run_id: str
    limit: int
    def __init__(self, provider: _Optional[str] = ..., group_by: _Optional[str] = ..., since: _Optional[str] = ..., run_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class UsageTokens(_message.Message):
    __slots__ = ("input", "cached_input", "cache_creation", "output", "total")
    INPUT_FIELD_NUMBER: _ClassVar[int]
    CACHED_INPUT_FIELD_NUMBER: _ClassVar[int]
    CACHE_CREATION_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    input: int
    cached_input: int
    cache_creation: int
    output: int
    total: int
    def __init__(self, input: _Optional[int] = ..., cached_input: _Optional[int] = ..., cache_creation: _Optional[int] = ..., output: _Optional[int] = ..., total: _Optional[int] = ...) -> None: ...

class UsageToolKindCount(_message.Message):
    __slots__ = ("kind", "calls")
    KIND_FIELD_NUMBER: _ClassVar[int]
    CALLS_FIELD_NUMBER: _ClassVar[int]
    kind: str
    calls: int
    def __init__(self, kind: _Optional[str] = ..., calls: _Optional[int] = ...) -> None: ...

class UsageModality(_message.Message):
    __slots__ = ("images_in", "images_in_user", "images_in_high_detail", "image_bytes_in", "estimated_image_input_tokens", "images_out", "web_searches", "web_fetches", "audio_in", "tool_calls", "tool_kinds")
    IMAGES_IN_FIELD_NUMBER: _ClassVar[int]
    IMAGES_IN_USER_FIELD_NUMBER: _ClassVar[int]
    IMAGES_IN_HIGH_DETAIL_FIELD_NUMBER: _ClassVar[int]
    IMAGE_BYTES_IN_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_IMAGE_INPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    IMAGES_OUT_FIELD_NUMBER: _ClassVar[int]
    WEB_SEARCHES_FIELD_NUMBER: _ClassVar[int]
    WEB_FETCHES_FIELD_NUMBER: _ClassVar[int]
    AUDIO_IN_FIELD_NUMBER: _ClassVar[int]
    TOOL_CALLS_FIELD_NUMBER: _ClassVar[int]
    TOOL_KINDS_FIELD_NUMBER: _ClassVar[int]
    images_in: int
    images_in_user: int
    images_in_high_detail: int
    image_bytes_in: int
    estimated_image_input_tokens: int
    images_out: int
    web_searches: int
    web_fetches: int
    audio_in: int
    tool_calls: int
    tool_kinds: _containers.RepeatedCompositeFieldContainer[UsageToolKindCount]
    def __init__(self, images_in: _Optional[int] = ..., images_in_user: _Optional[int] = ..., images_in_high_detail: _Optional[int] = ..., image_bytes_in: _Optional[int] = ..., estimated_image_input_tokens: _Optional[int] = ..., images_out: _Optional[int] = ..., web_searches: _Optional[int] = ..., web_fetches: _Optional[int] = ..., audio_in: _Optional[int] = ..., tool_calls: _Optional[int] = ..., tool_kinds: _Optional[_Iterable[_Union[UsageToolKindCount, _Mapping]]] = ...) -> None: ...

class UsageBurner(_message.Message):
    __slots__ = ("key", "label", "points", "share", "tokens", "weighted_tokens", "sessions", "runs", "turns", "run_ids", "runner", "model", "profile_id", "workload", "first_at", "last_at", "modality", "source")
    KEY_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    POINTS_FIELD_NUMBER: _ClassVar[int]
    SHARE_FIELD_NUMBER: _ClassVar[int]
    TOKENS_FIELD_NUMBER: _ClassVar[int]
    WEIGHTED_TOKENS_FIELD_NUMBER: _ClassVar[int]
    SESSIONS_FIELD_NUMBER: _ClassVar[int]
    RUNS_FIELD_NUMBER: _ClassVar[int]
    TURNS_FIELD_NUMBER: _ClassVar[int]
    RUN_IDS_FIELD_NUMBER: _ClassVar[int]
    RUNNER_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    WORKLOAD_FIELD_NUMBER: _ClassVar[int]
    FIRST_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_AT_FIELD_NUMBER: _ClassVar[int]
    MODALITY_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    key: str
    label: str
    points: float
    share: float
    tokens: UsageTokens
    weighted_tokens: float
    sessions: int
    runs: int
    turns: int
    run_ids: _containers.RepeatedScalarFieldContainer[str]
    runner: str
    model: str
    profile_id: str
    workload: str
    first_at: str
    last_at: str
    modality: UsageModality
    source: str
    def __init__(self, key: _Optional[str] = ..., label: _Optional[str] = ..., points: _Optional[float] = ..., share: _Optional[float] = ..., tokens: _Optional[_Union[UsageTokens, _Mapping]] = ..., weighted_tokens: _Optional[float] = ..., sessions: _Optional[int] = ..., runs: _Optional[int] = ..., turns: _Optional[int] = ..., run_ids: _Optional[_Iterable[str]] = ..., runner: _Optional[str] = ..., model: _Optional[str] = ..., profile_id: _Optional[str] = ..., workload: _Optional[str] = ..., first_at: _Optional[str] = ..., last_at: _Optional[str] = ..., modality: _Optional[_Union[UsageModality, _Mapping]] = ..., source: _Optional[str] = ...) -> None: ...

class UsageModelWeight(_message.Message):
    __slots__ = ("runner", "model", "cached_input_tokens_per_point", "input_weight", "cache_creation_weight", "output_weight", "source", "slices")
    RUNNER_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    CACHED_INPUT_TOKENS_PER_POINT_FIELD_NUMBER: _ClassVar[int]
    INPUT_WEIGHT_FIELD_NUMBER: _ClassVar[int]
    CACHE_CREATION_WEIGHT_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_WEIGHT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SLICES_FIELD_NUMBER: _ClassVar[int]
    runner: str
    model: str
    cached_input_tokens_per_point: float
    input_weight: float
    cache_creation_weight: float
    output_weight: float
    source: str
    slices: int
    def __init__(self, runner: _Optional[str] = ..., model: _Optional[str] = ..., cached_input_tokens_per_point: _Optional[float] = ..., input_weight: _Optional[float] = ..., cache_creation_weight: _Optional[float] = ..., output_weight: _Optional[float] = ..., source: _Optional[str] = ..., slices: _Optional[int] = ...) -> None: ...

class UsageCalibration(_message.Message):
    __slots__ = ("method", "weights", "r_squared", "slices", "off_disk_points_per_hour", "note")
    METHOD_FIELD_NUMBER: _ClassVar[int]
    WEIGHTS_FIELD_NUMBER: _ClassVar[int]
    R_SQUARED_FIELD_NUMBER: _ClassVar[int]
    SLICES_FIELD_NUMBER: _ClassVar[int]
    OFF_DISK_POINTS_PER_HOUR_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    method: str
    weights: _containers.RepeatedCompositeFieldContainer[UsageModelWeight]
    r_squared: float
    slices: int
    off_disk_points_per_hour: float
    note: str
    def __init__(self, method: _Optional[str] = ..., weights: _Optional[_Iterable[_Union[UsageModelWeight, _Mapping]]] = ..., r_squared: _Optional[float] = ..., slices: _Optional[int] = ..., off_disk_points_per_hour: _Optional[float] = ..., note: _Optional[str] = ...) -> None: ...

class UsageMeteredLine(_message.Message):
    __slots__ = ("kind", "count", "state", "amount", "note")
    KIND_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    kind: str
    count: int
    state: str
    amount: str
    note: str
    def __init__(self, kind: _Optional[str] = ..., count: _Optional[int] = ..., state: _Optional[str] = ..., amount: _Optional[str] = ..., note: _Optional[str] = ...) -> None: ...

class UsageWindowAttribution(_message.Message):
    __slots__ = ("provider", "pool", "window", "window_minutes", "window_start", "resets_at", "period_start", "period_end", "used_percent", "observed_increase_points", "attributed_points", "unattributed_points", "group_by", "burners", "other_burners", "other_points", "calibration", "pending_tokens", "metered_lines", "runner")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    POOL_FIELD_NUMBER: _ClassVar[int]
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    WINDOW_MINUTES_FIELD_NUMBER: _ClassVar[int]
    WINDOW_START_FIELD_NUMBER: _ClassVar[int]
    RESETS_AT_FIELD_NUMBER: _ClassVar[int]
    PERIOD_START_FIELD_NUMBER: _ClassVar[int]
    PERIOD_END_FIELD_NUMBER: _ClassVar[int]
    USED_PERCENT_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_INCREASE_POINTS_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTED_POINTS_FIELD_NUMBER: _ClassVar[int]
    UNATTRIBUTED_POINTS_FIELD_NUMBER: _ClassVar[int]
    GROUP_BY_FIELD_NUMBER: _ClassVar[int]
    BURNERS_FIELD_NUMBER: _ClassVar[int]
    OTHER_BURNERS_FIELD_NUMBER: _ClassVar[int]
    OTHER_POINTS_FIELD_NUMBER: _ClassVar[int]
    CALIBRATION_FIELD_NUMBER: _ClassVar[int]
    PENDING_TOKENS_FIELD_NUMBER: _ClassVar[int]
    METERED_LINES_FIELD_NUMBER: _ClassVar[int]
    RUNNER_FIELD_NUMBER: _ClassVar[int]
    provider: str
    pool: str
    window: str
    window_minutes: int
    window_start: str
    resets_at: str
    period_start: str
    period_end: str
    used_percent: float
    observed_increase_points: float
    attributed_points: float
    unattributed_points: float
    group_by: str
    burners: _containers.RepeatedCompositeFieldContainer[UsageBurner]
    other_burners: int
    other_points: float
    calibration: UsageCalibration
    pending_tokens: UsageTokens
    metered_lines: _containers.RepeatedCompositeFieldContainer[UsageMeteredLine]
    runner: str
    def __init__(self, provider: _Optional[str] = ..., pool: _Optional[str] = ..., window: _Optional[str] = ..., window_minutes: _Optional[int] = ..., window_start: _Optional[str] = ..., resets_at: _Optional[str] = ..., period_start: _Optional[str] = ..., period_end: _Optional[str] = ..., used_percent: _Optional[float] = ..., observed_increase_points: _Optional[float] = ..., attributed_points: _Optional[float] = ..., unattributed_points: _Optional[float] = ..., group_by: _Optional[str] = ..., burners: _Optional[_Iterable[_Union[UsageBurner, _Mapping]]] = ..., other_burners: _Optional[int] = ..., other_points: _Optional[float] = ..., calibration: _Optional[_Union[UsageCalibration, _Mapping]] = ..., pending_tokens: _Optional[_Union[UsageTokens, _Mapping]] = ..., metered_lines: _Optional[_Iterable[_Union[UsageMeteredLine, _Mapping]]] = ..., runner: _Optional[str] = ...) -> None: ...

class UsageBurnersResponse(_message.Message):
    __slots__ = ("windows", "generated_at", "validity", "provenance", "definition_id")
    WINDOWS_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    windows: _containers.RepeatedCompositeFieldContainer[UsageWindowAttribution]
    generated_at: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, windows: _Optional[_Iterable[_Union[UsageWindowAttribution, _Mapping]]] = ..., generated_at: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class UsageRecommendationsRequest(_message.Message):
    __slots__ = ("provider", "run_id", "limit")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    provider: str
    run_id: str
    limit: int
    def __init__(self, provider: _Optional[str] = ..., run_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class UsageEvidenceRef(_message.Message):
    __slots__ = ("kind", "id", "label")
    KIND_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    kind: str
    id: str
    label: str
    def __init__(self, kind: _Optional[str] = ..., id: _Optional[str] = ..., label: _Optional[str] = ...) -> None: ...

class UsageEvidenceMetric(_message.Message):
    __slots__ = ("name", "value", "unit")
    NAME_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    name: str
    value: float
    unit: str
    def __init__(self, name: _Optional[str] = ..., value: _Optional[float] = ..., unit: _Optional[str] = ...) -> None: ...

class UsageRecommendation(_message.Message):
    __slots__ = ("rank", "rule", "lever", "target_kind", "target_id", "target_label", "provider", "pool", "window", "est_weekly_quota_points_saved", "confidence", "action", "rationale", "consumer_points", "evidence_refs", "metrics")
    RANK_FIELD_NUMBER: _ClassVar[int]
    RULE_FIELD_NUMBER: _ClassVar[int]
    LEVER_FIELD_NUMBER: _ClassVar[int]
    TARGET_KIND_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_LABEL_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    POOL_FIELD_NUMBER: _ClassVar[int]
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    EST_WEEKLY_QUOTA_POINTS_SAVED_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    CONSUMER_POINTS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    rank: int
    rule: str
    lever: str
    target_kind: str
    target_id: str
    target_label: str
    provider: str
    pool: str
    window: str
    est_weekly_quota_points_saved: float
    confidence: str
    action: str
    rationale: str
    consumer_points: float
    evidence_refs: _containers.RepeatedCompositeFieldContainer[UsageEvidenceRef]
    metrics: _containers.RepeatedCompositeFieldContainer[UsageEvidenceMetric]
    def __init__(self, rank: _Optional[int] = ..., rule: _Optional[str] = ..., lever: _Optional[str] = ..., target_kind: _Optional[str] = ..., target_id: _Optional[str] = ..., target_label: _Optional[str] = ..., provider: _Optional[str] = ..., pool: _Optional[str] = ..., window: _Optional[str] = ..., est_weekly_quota_points_saved: _Optional[float] = ..., confidence: _Optional[str] = ..., action: _Optional[str] = ..., rationale: _Optional[str] = ..., consumer_points: _Optional[float] = ..., evidence_refs: _Optional[_Iterable[_Union[UsageEvidenceRef, _Mapping]]] = ..., metrics: _Optional[_Iterable[_Union[UsageEvidenceMetric, _Mapping]]] = ...) -> None: ...

class UsageRuleStatus(_message.Message):
    __slots__ = ("rule", "fired", "rows", "candidates", "reason")
    RULE_FIELD_NUMBER: _ClassVar[int]
    FIRED_FIELD_NUMBER: _ClassVar[int]
    ROWS_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    rule: str
    fired: bool
    rows: int
    candidates: int
    reason: str
    def __init__(self, rule: _Optional[str] = ..., fired: _Optional[bool] = ..., rows: _Optional[int] = ..., candidates: _Optional[int] = ..., reason: _Optional[str] = ...) -> None: ...

class UsageRecommendationsResponse(_message.Message):
    __slots__ = ("recommendations", "rules", "generated_at", "dominant_consumer", "dominant_consumer_label", "validity", "provenance", "definition_id")
    RECOMMENDATIONS_FIELD_NUMBER: _ClassVar[int]
    RULES_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    DOMINANT_CONSUMER_FIELD_NUMBER: _ClassVar[int]
    DOMINANT_CONSUMER_LABEL_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    recommendations: _containers.RepeatedCompositeFieldContainer[UsageRecommendation]
    rules: _containers.RepeatedCompositeFieldContainer[UsageRuleStatus]
    generated_at: str
    dominant_consumer: str
    dominant_consumer_label: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, recommendations: _Optional[_Iterable[_Union[UsageRecommendation, _Mapping]]] = ..., rules: _Optional[_Iterable[_Union[UsageRuleStatus, _Mapping]]] = ..., generated_at: _Optional[str] = ..., dominant_consumer: _Optional[str] = ..., dominant_consumer_label: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class UsageCoverageRequest(_message.Message):
    __slots__ = ("since",)
    SINCE_FIELD_NUMBER: _ClassVar[int]
    since: str
    def __init__(self, since: _Optional[str] = ...) -> None: ...

class UsageCoverageIssue(_message.Message):
    __slots__ = ("path", "reason")
    PATH_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    path: str
    reason: str
    def __init__(self, path: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class UsageCoverageRoot(_message.Message):
    __slots__ = ("kind", "harness", "files", "indexed", "with_usage", "owned", "excluded", "pending", "unreadable", "bytes", "captured_bytes", "captured_share", "turns", "usage_reports", "issues")
    KIND_FIELD_NUMBER: _ClassVar[int]
    HARNESS_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    INDEXED_FIELD_NUMBER: _ClassVar[int]
    WITH_USAGE_FIELD_NUMBER: _ClassVar[int]
    OWNED_FIELD_NUMBER: _ClassVar[int]
    EXCLUDED_FIELD_NUMBER: _ClassVar[int]
    PENDING_FIELD_NUMBER: _ClassVar[int]
    UNREADABLE_FIELD_NUMBER: _ClassVar[int]
    BYTES_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_BYTES_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_SHARE_FIELD_NUMBER: _ClassVar[int]
    TURNS_FIELD_NUMBER: _ClassVar[int]
    USAGE_REPORTS_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    kind: str
    harness: str
    files: int
    indexed: int
    with_usage: int
    owned: int
    excluded: int
    pending: int
    unreadable: int
    bytes: int
    captured_bytes: int
    captured_share: float
    turns: int
    usage_reports: int
    issues: _containers.RepeatedCompositeFieldContainer[UsageCoverageIssue]
    def __init__(self, kind: _Optional[str] = ..., harness: _Optional[str] = ..., files: _Optional[int] = ..., indexed: _Optional[int] = ..., with_usage: _Optional[int] = ..., owned: _Optional[int] = ..., excluded: _Optional[int] = ..., pending: _Optional[int] = ..., unreadable: _Optional[int] = ..., bytes: _Optional[int] = ..., captured_bytes: _Optional[int] = ..., captured_share: _Optional[float] = ..., turns: _Optional[int] = ..., usage_reports: _Optional[int] = ..., issues: _Optional[_Iterable[_Union[UsageCoverageIssue, _Mapping]]] = ...) -> None: ...

class UsageCoverageResponse(_message.Message):
    __slots__ = ("since", "roots", "totals", "method", "generated_at", "provenance", "definition_id")
    SINCE_FIELD_NUMBER: _ClassVar[int]
    ROOTS_FIELD_NUMBER: _ClassVar[int]
    TOTALS_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    since: str
    roots: _containers.RepeatedCompositeFieldContainer[UsageCoverageRoot]
    totals: UsageCoverageRoot
    method: str
    generated_at: str
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, since: _Optional[str] = ..., roots: _Optional[_Iterable[_Union[UsageCoverageRoot, _Mapping]]] = ..., totals: _Optional[_Union[UsageCoverageRoot, _Mapping]] = ..., method: _Optional[str] = ..., generated_at: _Optional[str] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class UsageCompareRequest(_message.Message):
    __slots__ = ("left", "right", "change_binding", "provider")
    LEFT_FIELD_NUMBER: _ClassVar[int]
    RIGHT_FIELD_NUMBER: _ClassVar[int]
    CHANGE_BINDING_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    left: CohortSelector
    right: CohortSelector
    change_binding: str
    provider: str
    def __init__(self, left: _Optional[_Union[CohortSelector, _Mapping]] = ..., right: _Optional[_Union[CohortSelector, _Mapping]] = ..., change_binding: _Optional[str] = ..., provider: _Optional[str] = ...) -> None: ...

class UsageCompareSide(_message.Message):
    __slots__ = ("runs", "successful_runs", "priced_runs", "tokens", "weighted_tokens", "points", "points_per_success", "tokens_per_success", "success_rate", "window_start", "window_end")
    RUNS_FIELD_NUMBER: _ClassVar[int]
    SUCCESSFUL_RUNS_FIELD_NUMBER: _ClassVar[int]
    PRICED_RUNS_FIELD_NUMBER: _ClassVar[int]
    TOKENS_FIELD_NUMBER: _ClassVar[int]
    WEIGHTED_TOKENS_FIELD_NUMBER: _ClassVar[int]
    POINTS_FIELD_NUMBER: _ClassVar[int]
    POINTS_PER_SUCCESS_FIELD_NUMBER: _ClassVar[int]
    TOKENS_PER_SUCCESS_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_RATE_FIELD_NUMBER: _ClassVar[int]
    WINDOW_START_FIELD_NUMBER: _ClassVar[int]
    WINDOW_END_FIELD_NUMBER: _ClassVar[int]
    runs: int
    successful_runs: int
    priced_runs: int
    tokens: UsageTokens
    weighted_tokens: float
    points: float
    points_per_success: float
    tokens_per_success: float
    success_rate: float
    window_start: str
    window_end: str
    def __init__(self, runs: _Optional[int] = ..., successful_runs: _Optional[int] = ..., priced_runs: _Optional[int] = ..., tokens: _Optional[_Union[UsageTokens, _Mapping]] = ..., weighted_tokens: _Optional[float] = ..., points: _Optional[float] = ..., points_per_success: _Optional[float] = ..., tokens_per_success: _Optional[float] = ..., success_rate: _Optional[float] = ..., window_start: _Optional[str] = ..., window_end: _Optional[str] = ...) -> None: ...

class UsageCompareResponse(_message.Message):
    __slots__ = ("left", "right", "change_binding", "points_per_success_change", "verdict", "caveats", "validity", "provenance", "definition_id")
    LEFT_FIELD_NUMBER: _ClassVar[int]
    RIGHT_FIELD_NUMBER: _ClassVar[int]
    CHANGE_BINDING_FIELD_NUMBER: _ClassVar[int]
    POINTS_PER_SUCCESS_CHANGE_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    CAVEATS_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    left: UsageCompareSide
    right: UsageCompareSide
    change_binding: str
    points_per_success_change: float
    verdict: str
    caveats: _containers.RepeatedScalarFieldContainer[str]
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, left: _Optional[_Union[UsageCompareSide, _Mapping]] = ..., right: _Optional[_Union[UsageCompareSide, _Mapping]] = ..., change_binding: _Optional[str] = ..., points_per_success_change: _Optional[float] = ..., verdict: _Optional[str] = ..., caveats: _Optional[_Iterable[str]] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...
