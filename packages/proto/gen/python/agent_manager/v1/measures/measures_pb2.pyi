from measures.v1 import measures_pb2 as _measures_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RunScope(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RUN_SCOPE_UNSPECIFIED: _ClassVar[RunScope]
    RUN_SCOPE_AGENT_MANAGER: _ClassVar[RunScope]
    RUN_SCOPE_OPERATOR_SESSIONS: _ClassVar[RunScope]
    RUN_SCOPE_ALL: _ClassVar[RunScope]
RUN_SCOPE_UNSPECIFIED: RunScope
RUN_SCOPE_AGENT_MANAGER: RunScope
RUN_SCOPE_OPERATOR_SESSIONS: RunScope
RUN_SCOPE_ALL: RunScope

class InvocationFilter(_message.Message):
    __slots__ = ("window", "ownership", "outcome", "executable", "fingerprint", "profile_id", "runner_type", "model", "tag_prefix", "run_status", "tool_name", "target_scenario", "operation", "workload_key", "scope")
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
    TARGET_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    WORKLOAD_KEY_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
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
    target_scenario: str
    operation: str
    workload_key: str
    scope: RunScope
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., ownership: _Optional[str] = ..., outcome: _Optional[str] = ..., executable: _Optional[str] = ..., fingerprint: _Optional[str] = ..., profile_id: _Optional[str] = ..., runner_type: _Optional[str] = ..., model: _Optional[str] = ..., tag_prefix: _Optional[str] = ..., run_status: _Optional[str] = ..., tool_name: _Optional[str] = ..., target_scenario: _Optional[str] = ..., operation: _Optional[str] = ..., workload_key: _Optional[str] = ..., scope: _Optional[_Union[RunScope, str]] = ...) -> None: ...

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

class ToolFrictionRequest(_message.Message):
    __slots__ = ("window", "owner", "limit")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    owner: str
    limit: int
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., owner: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ToolFrictionCoverage(_message.Message):
    __slots__ = ("commands", "outcome_known", "outcome_known_share", "vrooli_commands", "vrooli_outcome_known", "sentence", "older_parser_commands", "no_transcript_commands", "stale_runs", "stale_calls", "unknown_token_commands")
    COMMANDS_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_KNOWN_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_KNOWN_SHARE_FIELD_NUMBER: _ClassVar[int]
    VROOLI_COMMANDS_FIELD_NUMBER: _ClassVar[int]
    VROOLI_OUTCOME_KNOWN_FIELD_NUMBER: _ClassVar[int]
    SENTENCE_FIELD_NUMBER: _ClassVar[int]
    OLDER_PARSER_COMMANDS_FIELD_NUMBER: _ClassVar[int]
    NO_TRANSCRIPT_COMMANDS_FIELD_NUMBER: _ClassVar[int]
    STALE_RUNS_FIELD_NUMBER: _ClassVar[int]
    STALE_CALLS_FIELD_NUMBER: _ClassVar[int]
    UNKNOWN_TOKEN_COMMANDS_FIELD_NUMBER: _ClassVar[int]
    commands: int
    outcome_known: int
    outcome_known_share: float
    vrooli_commands: int
    vrooli_outcome_known: int
    sentence: str
    older_parser_commands: int
    no_transcript_commands: int
    stale_runs: int
    stale_calls: int
    unknown_token_commands: int
    def __init__(self, commands: _Optional[int] = ..., outcome_known: _Optional[int] = ..., outcome_known_share: _Optional[float] = ..., vrooli_commands: _Optional[int] = ..., vrooli_outcome_known: _Optional[int] = ..., sentence: _Optional[str] = ..., older_parser_commands: _Optional[int] = ..., no_transcript_commands: _Optional[int] = ..., stale_runs: _Optional[int] = ..., stale_calls: _Optional[int] = ..., unknown_token_commands: _Optional[int] = ...) -> None: ...

class ToolFrictionSummary(_message.Message):
    __slots__ = ("coverage", "calls", "failed_calls", "vrooli_failure_rate", "shape_guesses", "shape_guess_rate", "polled_calls", "polled_share", "bypass_calls", "context_tax_tokens", "duration_ms", "filtered_share", "profiles")
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    CALLS_FIELD_NUMBER: _ClassVar[int]
    FAILED_CALLS_FIELD_NUMBER: _ClassVar[int]
    VROOLI_FAILURE_RATE_FIELD_NUMBER: _ClassVar[int]
    SHAPE_GUESSES_FIELD_NUMBER: _ClassVar[int]
    SHAPE_GUESS_RATE_FIELD_NUMBER: _ClassVar[int]
    POLLED_CALLS_FIELD_NUMBER: _ClassVar[int]
    POLLED_SHARE_FIELD_NUMBER: _ClassVar[int]
    BYPASS_CALLS_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_TAX_TOKENS_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    FILTERED_SHARE_FIELD_NUMBER: _ClassVar[int]
    PROFILES_FIELD_NUMBER: _ClassVar[int]
    coverage: ToolFrictionCoverage
    calls: int
    failed_calls: int
    vrooli_failure_rate: float
    shape_guesses: int
    shape_guess_rate: float
    polled_calls: int
    polled_share: float
    bypass_calls: int
    context_tax_tokens: int
    duration_ms: int
    filtered_share: float
    profiles: int
    def __init__(self, coverage: _Optional[_Union[ToolFrictionCoverage, _Mapping]] = ..., calls: _Optional[int] = ..., failed_calls: _Optional[int] = ..., vrooli_failure_rate: _Optional[float] = ..., shape_guesses: _Optional[int] = ..., shape_guess_rate: _Optional[float] = ..., polled_calls: _Optional[int] = ..., polled_share: _Optional[float] = ..., bypass_calls: _Optional[int] = ..., context_tax_tokens: _Optional[int] = ..., duration_ms: _Optional[int] = ..., filtered_share: _Optional[float] = ..., profiles: _Optional[int] = ...) -> None: ...

class ToolProfile(_message.Message):
    __slots__ = ("command_path", "owner", "owner_kind", "calls", "runs", "decided", "failed", "no_match", "refused", "unknown", "failure_rate", "shape_guesses", "median_result_tokens", "p90_result_tokens", "max_result_tokens", "truncated_share", "context_tax_tokens", "filtered_share", "json_paths", "total_duration_ms", "p90_duration_ms", "polled_calls", "wrapped_polls", "cost_share")
    COMMAND_PATH_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    OWNER_KIND_FIELD_NUMBER: _ClassVar[int]
    CALLS_FIELD_NUMBER: _ClassVar[int]
    RUNS_FIELD_NUMBER: _ClassVar[int]
    DECIDED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    NO_MATCH_FIELD_NUMBER: _ClassVar[int]
    REFUSED_FIELD_NUMBER: _ClassVar[int]
    UNKNOWN_FIELD_NUMBER: _ClassVar[int]
    FAILURE_RATE_FIELD_NUMBER: _ClassVar[int]
    SHAPE_GUESSES_FIELD_NUMBER: _ClassVar[int]
    MEDIAN_RESULT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    P90_RESULT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    MAX_RESULT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_SHARE_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_TAX_TOKENS_FIELD_NUMBER: _ClassVar[int]
    FILTERED_SHARE_FIELD_NUMBER: _ClassVar[int]
    JSON_PATHS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    P90_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    POLLED_CALLS_FIELD_NUMBER: _ClassVar[int]
    WRAPPED_POLLS_FIELD_NUMBER: _ClassVar[int]
    COST_SHARE_FIELD_NUMBER: _ClassVar[int]
    command_path: str
    owner: str
    owner_kind: str
    calls: int
    runs: int
    decided: int
    failed: int
    no_match: int
    refused: int
    unknown: int
    failure_rate: float
    shape_guesses: int
    median_result_tokens: int
    p90_result_tokens: int
    max_result_tokens: int
    truncated_share: float
    context_tax_tokens: int
    filtered_share: float
    json_paths: _containers.RepeatedScalarFieldContainer[str]
    total_duration_ms: int
    p90_duration_ms: int
    polled_calls: int
    wrapped_polls: int
    cost_share: float
    def __init__(self, command_path: _Optional[str] = ..., owner: _Optional[str] = ..., owner_kind: _Optional[str] = ..., calls: _Optional[int] = ..., runs: _Optional[int] = ..., decided: _Optional[int] = ..., failed: _Optional[int] = ..., no_match: _Optional[int] = ..., refused: _Optional[int] = ..., unknown: _Optional[int] = ..., failure_rate: _Optional[float] = ..., shape_guesses: _Optional[int] = ..., median_result_tokens: _Optional[int] = ..., p90_result_tokens: _Optional[int] = ..., max_result_tokens: _Optional[int] = ..., truncated_share: _Optional[float] = ..., context_tax_tokens: _Optional[int] = ..., filtered_share: _Optional[float] = ..., json_paths: _Optional[_Iterable[str]] = ..., total_duration_ms: _Optional[int] = ..., p90_duration_ms: _Optional[int] = ..., polled_calls: _Optional[int] = ..., wrapped_polls: _Optional[int] = ..., cost_share: _Optional[float] = ...) -> None: ...

class ToolErrorCluster(_message.Message):
    __slots__ = ("signature", "kind", "owner", "owner_kind", "command_paths", "calls", "runs", "duration_ms", "line", "example_commands", "run_ids", "root_cause", "subs")
    SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    OWNER_KIND_FIELD_NUMBER: _ClassVar[int]
    COMMAND_PATHS_FIELD_NUMBER: _ClassVar[int]
    CALLS_FIELD_NUMBER: _ClassVar[int]
    RUNS_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    LINE_FIELD_NUMBER: _ClassVar[int]
    EXAMPLE_COMMANDS_FIELD_NUMBER: _ClassVar[int]
    RUN_IDS_FIELD_NUMBER: _ClassVar[int]
    ROOT_CAUSE_FIELD_NUMBER: _ClassVar[int]
    SUBS_FIELD_NUMBER: _ClassVar[int]
    signature: str
    kind: str
    owner: str
    owner_kind: str
    command_paths: _containers.RepeatedScalarFieldContainer[str]
    calls: int
    runs: int
    duration_ms: int
    line: str
    example_commands: _containers.RepeatedScalarFieldContainer[str]
    run_ids: _containers.RepeatedScalarFieldContainer[str]
    root_cause: str
    subs: _containers.RepeatedCompositeFieldContainer[UsageToolSub]
    def __init__(self, signature: _Optional[str] = ..., kind: _Optional[str] = ..., owner: _Optional[str] = ..., owner_kind: _Optional[str] = ..., command_paths: _Optional[_Iterable[str]] = ..., calls: _Optional[int] = ..., runs: _Optional[int] = ..., duration_ms: _Optional[int] = ..., line: _Optional[str] = ..., example_commands: _Optional[_Iterable[str]] = ..., run_ids: _Optional[_Iterable[str]] = ..., root_cause: _Optional[str] = ..., subs: _Optional[_Iterable[_Union[UsageToolSub, _Mapping]]] = ...) -> None: ...

class UsageToolSub(_message.Message):
    __slots__ = ("kind", "signature", "line", "calls")
    KIND_FIELD_NUMBER: _ClassVar[int]
    SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    LINE_FIELD_NUMBER: _ClassVar[int]
    CALLS_FIELD_NUMBER: _ClassVar[int]
    kind: str
    signature: str
    line: str
    calls: int
    def __init__(self, kind: _Optional[str] = ..., signature: _Optional[str] = ..., line: _Optional[str] = ..., calls: _Optional[int] = ...) -> None: ...

class UsageToolGuess(_message.Message):
    __slots__ = ("token", "suggestion", "calls")
    TOKEN_FIELD_NUMBER: _ClassVar[int]
    SUGGESTION_FIELD_NUMBER: _ClassVar[int]
    CALLS_FIELD_NUMBER: _ClassVar[int]
    token: str
    suggestion: str
    calls: int
    def __init__(self, token: _Optional[str] = ..., suggestion: _Optional[str] = ..., calls: _Optional[int] = ...) -> None: ...

class ToolFrictionResponse(_message.Message):
    __slots__ = ("summary", "profiles", "clusters", "window_start", "window_end", "findings", "validity", "provenance", "definition_id")
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    PROFILES_FIELD_NUMBER: _ClassVar[int]
    CLUSTERS_FIELD_NUMBER: _ClassVar[int]
    WINDOW_START_FIELD_NUMBER: _ClassVar[int]
    WINDOW_END_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    summary: ToolFrictionSummary
    profiles: _containers.RepeatedCompositeFieldContainer[ToolProfile]
    clusters: _containers.RepeatedCompositeFieldContainer[ToolErrorCluster]
    window_start: str
    window_end: str
    findings: _containers.RepeatedCompositeFieldContainer[UsageToolFinding]
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, summary: _Optional[_Union[ToolFrictionSummary, _Mapping]] = ..., profiles: _Optional[_Iterable[_Union[ToolProfile, _Mapping]]] = ..., clusters: _Optional[_Iterable[_Union[ToolErrorCluster, _Mapping]]] = ..., window_start: _Optional[str] = ..., window_end: _Optional[str] = ..., findings: _Optional[_Iterable[_Union[UsageToolFinding, _Mapping]]] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("average_duration_ms", "completed_duration_runs", "executed_query", "p50_duration_ms", "p90_duration_ms", "validity", "provenance", "definition_id")
    AVERAGE_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_DURATION_RUNS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    P50_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    P90_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    average_duration_ms: float
    completed_duration_runs: int
    executed_query: str
    p50_duration_ms: float
    p90_duration_ms: float
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, average_duration_ms: _Optional[float] = ..., completed_duration_runs: _Optional[int] = ..., executed_query: _Optional[str] = ..., p50_duration_ms: _Optional[float] = ..., p90_duration_ms: _Optional[float] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class RunDurationStatisticsRequest(_message.Message):
    __slots__ = ("window", "filter")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ...) -> None: ...

class RunDurationStatisticsResponse(_message.Message):
    __slots__ = ("average_duration_ms", "p50_duration_ms", "p95_duration_ms", "p99_duration_ms", "min_duration_ms", "max_duration_ms", "count", "executed_query", "p90_duration_ms", "validity", "provenance", "definition_id")
    AVERAGE_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    P50_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    P95_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    P99_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    MIN_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    MAX_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    P90_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
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
    p90_duration_ms: float
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, average_duration_ms: _Optional[float] = ..., p50_duration_ms: _Optional[float] = ..., p95_duration_ms: _Optional[float] = ..., p99_duration_ms: _Optional[float] = ..., min_duration_ms: _Optional[int] = ..., max_duration_ms: _Optional[int] = ..., count: _Optional[int] = ..., executed_query: _Optional[str] = ..., p90_duration_ms: _Optional[float] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("total_cost_usd", "average_cost_usd", "total_tokens", "total_runs", "executed_query", "validity", "input_tokens", "output_tokens", "cache_read_tokens", "cache_creation_tokens", "input_cost_usd", "output_cost_usd", "cache_read_cost_usd", "cache_creation_cost_usd", "total_charge_micro_usd", "requests", "p50_tokens", "p90_tokens", "p95_tokens", "p99_tokens", "max_tokens", "p50_cost_usd", "p90_cost_usd", "p95_cost_usd", "p99_cost_usd", "max_cost_usd", "token_buckets", "distribution_sample_size", "provenance", "definition_id")
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
    TOTAL_CHARGE_MICRO_USD_FIELD_NUMBER: _ClassVar[int]
    REQUESTS_FIELD_NUMBER: _ClassVar[int]
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
    total_charge_micro_usd: int
    requests: int
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
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, total_cost_usd: _Optional[float] = ..., average_cost_usd: _Optional[float] = ..., total_tokens: _Optional[int] = ..., total_runs: _Optional[int] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., input_tokens: _Optional[int] = ..., output_tokens: _Optional[int] = ..., cache_read_tokens: _Optional[int] = ..., cache_creation_tokens: _Optional[int] = ..., input_cost_usd: _Optional[float] = ..., output_cost_usd: _Optional[float] = ..., cache_read_cost_usd: _Optional[float] = ..., cache_creation_cost_usd: _Optional[float] = ..., total_charge_micro_usd: _Optional[int] = ..., requests: _Optional[int] = ..., p50_tokens: _Optional[int] = ..., p90_tokens: _Optional[int] = ..., p95_tokens: _Optional[int] = ..., p99_tokens: _Optional[int] = ..., max_tokens: _Optional[int] = ..., p50_cost_usd: _Optional[float] = ..., p90_cost_usd: _Optional[float] = ..., p95_cost_usd: _Optional[float] = ..., p99_cost_usd: _Optional[float] = ..., max_cost_usd: _Optional[float] = ..., token_buckets: _Optional[_Iterable[_Union[RunTokenBucket, _Mapping]]] = ..., distribution_sample_size: _Optional[int] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("bucket", "terminal_runs", "completed_runs", "failed_runs", "cancelled_runs", "average_duration_ms")
    BUCKET_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_RUNS_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_RUNS_FIELD_NUMBER: _ClassVar[int]
    FAILED_RUNS_FIELD_NUMBER: _ClassVar[int]
    CANCELLED_RUNS_FIELD_NUMBER: _ClassVar[int]
    AVERAGE_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    bucket: str
    terminal_runs: int
    completed_runs: int
    failed_runs: int
    cancelled_runs: int
    average_duration_ms: float
    def __init__(self, bucket: _Optional[str] = ..., terminal_runs: _Optional[int] = ..., completed_runs: _Optional[int] = ..., failed_runs: _Optional[int] = ..., cancelled_runs: _Optional[int] = ..., average_duration_ms: _Optional[float] = ...) -> None: ...

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
    __slots__ = ("availability_state", "availability_reason", "goal_runs", "met_runs", "average_time_to_first_met_ms", "average_tokens_to_first_met", "interventions_before_met", "regressions", "restatements")
    AVAILABILITY_STATE_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_REASON_FIELD_NUMBER: _ClassVar[int]
    GOAL_RUNS_FIELD_NUMBER: _ClassVar[int]
    MET_RUNS_FIELD_NUMBER: _ClassVar[int]
    AVERAGE_TIME_TO_FIRST_MET_MS_FIELD_NUMBER: _ClassVar[int]
    AVERAGE_TOKENS_TO_FIRST_MET_FIELD_NUMBER: _ClassVar[int]
    INTERVENTIONS_BEFORE_MET_FIELD_NUMBER: _ClassVar[int]
    REGRESSIONS_FIELD_NUMBER: _ClassVar[int]
    RESTATEMENTS_FIELD_NUMBER: _ClassVar[int]
    availability_state: str
    availability_reason: str
    goal_runs: int
    met_runs: int
    average_time_to_first_met_ms: float
    average_tokens_to_first_met: float
    interventions_before_met: int
    regressions: int
    restatements: int
    def __init__(self, availability_state: _Optional[str] = ..., availability_reason: _Optional[str] = ..., goal_runs: _Optional[int] = ..., met_runs: _Optional[int] = ..., average_time_to_first_met_ms: _Optional[float] = ..., average_tokens_to_first_met: _Optional[float] = ..., interventions_before_met: _Optional[int] = ..., regressions: _Optional[int] = ..., restatements: _Optional[int] = ...) -> None: ...

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

class UsageForecastRequest(_message.Message):
    __slots__ = ("provider", "account_scope")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_SCOPE_FIELD_NUMBER: _ClassVar[int]
    provider: str
    account_scope: str
    def __init__(self, provider: _Optional[str] = ..., account_scope: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("provider", "pool", "window", "window_minutes", "status", "status_reason", "has_percent", "used_percent", "standing", "window_start", "resets_at", "observed_at", "sample_count", "burn", "central_percent_per_hour", "projected_exhaustion_at", "exhaustion_band_start", "exhaustion_band_end", "lasting_percent_per_hour", "early_reset", "previous_resets_at", "previous_final_percent", "previous_ended_at", "series", "last_reading_percent", "calibration_factor", "calibration_windows", "dormant", "primary", "focus")
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
    DORMANT_FIELD_NUMBER: _ClassVar[int]
    PRIMARY_FIELD_NUMBER: _ClassVar[int]
    FOCUS_FIELD_NUMBER: _ClassVar[int]
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
    dormant: bool
    primary: bool
    focus: bool
    def __init__(self, provider: _Optional[str] = ..., pool: _Optional[str] = ..., window: _Optional[str] = ..., window_minutes: _Optional[int] = ..., status: _Optional[str] = ..., status_reason: _Optional[str] = ..., has_percent: _Optional[bool] = ..., used_percent: _Optional[float] = ..., standing: _Optional[str] = ..., window_start: _Optional[str] = ..., resets_at: _Optional[str] = ..., observed_at: _Optional[str] = ..., sample_count: _Optional[int] = ..., burn: _Optional[_Iterable[_Union[UsageBurnRate, _Mapping]]] = ..., central_percent_per_hour: _Optional[float] = ..., projected_exhaustion_at: _Optional[str] = ..., exhaustion_band_start: _Optional[str] = ..., exhaustion_band_end: _Optional[str] = ..., lasting_percent_per_hour: _Optional[float] = ..., early_reset: _Optional[bool] = ..., previous_resets_at: _Optional[str] = ..., previous_final_percent: _Optional[float] = ..., previous_ended_at: _Optional[str] = ..., series: _Optional[_Iterable[_Union[UsagePercentPoint, _Mapping]]] = ..., last_reading_percent: _Optional[float] = ..., calibration_factor: _Optional[float] = ..., calibration_windows: _Optional[int] = ..., dormant: _Optional[bool] = ..., primary: _Optional[bool] = ..., focus: _Optional[bool] = ...) -> None: ...

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
    __slots__ = ("provider", "account_scope")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_SCOPE_FIELD_NUMBER: _ClassVar[int]
    provider: str
    account_scope: str
    def __init__(self, provider: _Optional[str] = ..., account_scope: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("key", "label", "points", "share", "tokens", "weighted_tokens", "sessions", "runs", "requests", "run_ids", "runner", "model", "profile_id", "workload", "first_at", "last_at", "modality", "source")
    KEY_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    POINTS_FIELD_NUMBER: _ClassVar[int]
    SHARE_FIELD_NUMBER: _ClassVar[int]
    TOKENS_FIELD_NUMBER: _ClassVar[int]
    WEIGHTED_TOKENS_FIELD_NUMBER: _ClassVar[int]
    SESSIONS_FIELD_NUMBER: _ClassVar[int]
    RUNS_FIELD_NUMBER: _ClassVar[int]
    REQUESTS_FIELD_NUMBER: _ClassVar[int]
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
    requests: int
    run_ids: _containers.RepeatedScalarFieldContainer[str]
    runner: str
    model: str
    profile_id: str
    workload: str
    first_at: str
    last_at: str
    modality: UsageModality
    source: str
    def __init__(self, key: _Optional[str] = ..., label: _Optional[str] = ..., points: _Optional[float] = ..., share: _Optional[float] = ..., tokens: _Optional[_Union[UsageTokens, _Mapping]] = ..., weighted_tokens: _Optional[float] = ..., sessions: _Optional[int] = ..., runs: _Optional[int] = ..., requests: _Optional[int] = ..., run_ids: _Optional[_Iterable[str]] = ..., runner: _Optional[str] = ..., model: _Optional[str] = ..., profile_id: _Optional[str] = ..., workload: _Optional[str] = ..., first_at: _Optional[str] = ..., last_at: _Optional[str] = ..., modality: _Optional[_Union[UsageModality, _Mapping]] = ..., source: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("provider", "pool", "window", "window_minutes", "window_start", "resets_at", "period_start", "period_end", "used_percent", "observed_increase_points", "attributed_points", "unattributed_points", "group_by", "burners", "other_burners", "other_points", "calibration", "pending_tokens", "metered_lines", "runner", "fold_below_points")
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
    FOLD_BELOW_POINTS_FIELD_NUMBER: _ClassVar[int]
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
    fold_below_points: float
    def __init__(self, provider: _Optional[str] = ..., pool: _Optional[str] = ..., window: _Optional[str] = ..., window_minutes: _Optional[int] = ..., window_start: _Optional[str] = ..., resets_at: _Optional[str] = ..., period_start: _Optional[str] = ..., period_end: _Optional[str] = ..., used_percent: _Optional[float] = ..., observed_increase_points: _Optional[float] = ..., attributed_points: _Optional[float] = ..., unattributed_points: _Optional[float] = ..., group_by: _Optional[str] = ..., burners: _Optional[_Iterable[_Union[UsageBurner, _Mapping]]] = ..., other_burners: _Optional[int] = ..., other_points: _Optional[float] = ..., calibration: _Optional[_Union[UsageCalibration, _Mapping]] = ..., pending_tokens: _Optional[_Union[UsageTokens, _Mapping]] = ..., metered_lines: _Optional[_Iterable[_Union[UsageMeteredLine, _Mapping]]] = ..., runner: _Optional[str] = ..., fold_below_points: _Optional[float] = ...) -> None: ...

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
    __slots__ = ("provider", "run_id", "limit", "pool", "window")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    POOL_FIELD_NUMBER: _ClassVar[int]
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    provider: str
    run_id: str
    limit: int
    pool: str
    window: str
    def __init__(self, provider: _Optional[str] = ..., run_id: _Optional[str] = ..., limit: _Optional[int] = ..., pool: _Optional[str] = ..., window: _Optional[str] = ...) -> None: ...

class UsageEvidenceRef(_message.Message):
    __slots__ = ("kind", "id", "label")
    KIND_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    kind: str
    id: str
    label: str
    def __init__(self, kind: _Optional[str] = ..., id: _Optional[str] = ..., label: _Optional[str] = ...) -> None: ...

class UsageActionEffect(_message.Message):
    __slots__ = ("value", "upper", "unit", "basis")
    VALUE_FIELD_NUMBER: _ClassVar[int]
    UPPER_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    BASIS_FIELD_NUMBER: _ClassVar[int]
    value: float
    upper: float
    unit: str
    basis: str
    def __init__(self, value: _Optional[float] = ..., upper: _Optional[float] = ..., unit: _Optional[str] = ..., basis: _Optional[str] = ...) -> None: ...

class UsageActionDrill(_message.Message):
    __slots__ = ("kind", "id", "label", "detail", "value", "still_running")
    KIND_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    STILL_RUNNING_FIELD_NUMBER: _ClassVar[int]
    kind: str
    id: str
    label: str
    detail: str
    value: float
    still_running: bool
    def __init__(self, kind: _Optional[str] = ..., id: _Optional[str] = ..., label: _Optional[str] = ..., detail: _Optional[str] = ..., value: _Optional[float] = ..., still_running: _Optional[bool] = ...) -> None: ...

class UsageAction(_message.Message):
    __slots__ = ("rank", "source", "kind", "owner_kind", "owner", "title", "action", "rationale", "effect", "score", "score_basis", "recoverable_points", "confidence", "evidence_refs", "drilldown", "verify", "fingerprints", "provider", "pool", "window", "top_run_share", "root_cause", "subs", "guesses")
    RANK_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    OWNER_KIND_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    EFFECT_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    SCORE_BASIS_FIELD_NUMBER: _ClassVar[int]
    RECOVERABLE_POINTS_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    DRILLDOWN_FIELD_NUMBER: _ClassVar[int]
    VERIFY_FIELD_NUMBER: _ClassVar[int]
    FINGERPRINTS_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    POOL_FIELD_NUMBER: _ClassVar[int]
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    TOP_RUN_SHARE_FIELD_NUMBER: _ClassVar[int]
    ROOT_CAUSE_FIELD_NUMBER: _ClassVar[int]
    SUBS_FIELD_NUMBER: _ClassVar[int]
    GUESSES_FIELD_NUMBER: _ClassVar[int]
    rank: int
    source: str
    kind: str
    owner_kind: str
    owner: str
    title: str
    action: str
    rationale: str
    effect: UsageActionEffect
    score: float
    score_basis: str
    recoverable_points: float
    confidence: str
    evidence_refs: _containers.RepeatedCompositeFieldContainer[UsageEvidenceRef]
    drilldown: _containers.RepeatedCompositeFieldContainer[UsageActionDrill]
    verify: str
    fingerprints: _containers.RepeatedScalarFieldContainer[str]
    provider: str
    pool: str
    window: str
    top_run_share: float
    root_cause: str
    subs: _containers.RepeatedCompositeFieldContainer[UsageToolSub]
    guesses: _containers.RepeatedCompositeFieldContainer[UsageToolGuess]
    def __init__(self, rank: _Optional[int] = ..., source: _Optional[str] = ..., kind: _Optional[str] = ..., owner_kind: _Optional[str] = ..., owner: _Optional[str] = ..., title: _Optional[str] = ..., action: _Optional[str] = ..., rationale: _Optional[str] = ..., effect: _Optional[_Union[UsageActionEffect, _Mapping]] = ..., score: _Optional[float] = ..., score_basis: _Optional[str] = ..., recoverable_points: _Optional[float] = ..., confidence: _Optional[str] = ..., evidence_refs: _Optional[_Iterable[_Union[UsageEvidenceRef, _Mapping]]] = ..., drilldown: _Optional[_Iterable[_Union[UsageActionDrill, _Mapping]]] = ..., verify: _Optional[str] = ..., fingerprints: _Optional[_Iterable[str]] = ..., provider: _Optional[str] = ..., pool: _Optional[str] = ..., window: _Optional[str] = ..., top_run_share: _Optional[float] = ..., root_cause: _Optional[str] = ..., subs: _Optional[_Iterable[_Union[UsageToolSub, _Mapping]]] = ..., guesses: _Optional[_Iterable[_Union[UsageToolGuess, _Mapping]]] = ...) -> None: ...

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
    __slots__ = ("actions", "rules", "generated_at", "below_materiality", "materiality_score", "scoring", "coverage", "outcome", "validity", "provenance", "definition_id")
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    RULES_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    BELOW_MATERIALITY_FIELD_NUMBER: _ClassVar[int]
    MATERIALITY_SCORE_FIELD_NUMBER: _ClassVar[int]
    SCORING_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    actions: _containers.RepeatedCompositeFieldContainer[UsageAction]
    rules: _containers.RepeatedCompositeFieldContainer[UsageRuleStatus]
    generated_at: str
    below_materiality: int
    materiality_score: float
    scoring: _containers.RepeatedScalarFieldContainer[str]
    coverage: _containers.RepeatedScalarFieldContainer[str]
    outcome: UsageWindowOutcome
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, actions: _Optional[_Iterable[_Union[UsageAction, _Mapping]]] = ..., rules: _Optional[_Iterable[_Union[UsageRuleStatus, _Mapping]]] = ..., generated_at: _Optional[str] = ..., below_materiality: _Optional[int] = ..., materiality_score: _Optional[float] = ..., scoring: _Optional[_Iterable[str]] = ..., coverage: _Optional[_Iterable[str]] = ..., outcome: _Optional[_Union[UsageWindowOutcome, _Mapping]] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class UsageWindowOutcome(_message.Message):
    __slots__ = ("recoverable_points", "levers", "projected", "end_percent", "reaches_reset", "run_out_at", "shortfall_seconds", "cut_needed")
    RECOVERABLE_POINTS_FIELD_NUMBER: _ClassVar[int]
    LEVERS_FIELD_NUMBER: _ClassVar[int]
    PROJECTED_FIELD_NUMBER: _ClassVar[int]
    END_PERCENT_FIELD_NUMBER: _ClassVar[int]
    REACHES_RESET_FIELD_NUMBER: _ClassVar[int]
    RUN_OUT_AT_FIELD_NUMBER: _ClassVar[int]
    SHORTFALL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    CUT_NEEDED_FIELD_NUMBER: _ClassVar[int]
    recoverable_points: float
    levers: int
    projected: bool
    end_percent: float
    reaches_reset: bool
    run_out_at: str
    shortfall_seconds: int
    cut_needed: float
    def __init__(self, recoverable_points: _Optional[float] = ..., levers: _Optional[int] = ..., projected: _Optional[bool] = ..., end_percent: _Optional[float] = ..., reaches_reset: _Optional[bool] = ..., run_out_at: _Optional[str] = ..., shortfall_seconds: _Optional[int] = ..., cut_needed: _Optional[float] = ...) -> None: ...

class UsageFrictionRequest(_message.Message):
    __slots__ = ("status", "limit", "since")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    SINCE_FIELD_NUMBER: _ClassVar[int]
    status: str
    limit: int
    since: str
    def __init__(self, status: _Optional[str] = ..., limit: _Optional[int] = ..., since: _Optional[str] = ...) -> None: ...

class UsageToolFinding(_message.Message):
    __slots__ = ("fingerprint", "kind", "owner", "owner_kind", "command_path", "description", "calls", "runs", "cost_unit", "cost", "cost_share", "signature", "example_commands", "evidence_refs", "status", "bug_id", "status_at", "top_run_share", "root_cause", "subs", "guesses", "printed_tokens", "read_tokens", "shares", "score")
    FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    OWNER_KIND_FIELD_NUMBER: _ClassVar[int]
    COMMAND_PATH_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CALLS_FIELD_NUMBER: _ClassVar[int]
    RUNS_FIELD_NUMBER: _ClassVar[int]
    COST_UNIT_FIELD_NUMBER: _ClassVar[int]
    COST_FIELD_NUMBER: _ClassVar[int]
    COST_SHARE_FIELD_NUMBER: _ClassVar[int]
    SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    EXAMPLE_COMMANDS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    BUG_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_AT_FIELD_NUMBER: _ClassVar[int]
    TOP_RUN_SHARE_FIELD_NUMBER: _ClassVar[int]
    ROOT_CAUSE_FIELD_NUMBER: _ClassVar[int]
    SUBS_FIELD_NUMBER: _ClassVar[int]
    GUESSES_FIELD_NUMBER: _ClassVar[int]
    PRINTED_TOKENS_FIELD_NUMBER: _ClassVar[int]
    READ_TOKENS_FIELD_NUMBER: _ClassVar[int]
    SHARES_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    fingerprint: str
    kind: str
    owner: str
    owner_kind: str
    command_path: str
    description: str
    calls: int
    runs: int
    cost_unit: str
    cost: int
    cost_share: float
    signature: str
    example_commands: _containers.RepeatedScalarFieldContainer[str]
    evidence_refs: _containers.RepeatedCompositeFieldContainer[UsageEvidenceRef]
    status: str
    bug_id: str
    status_at: str
    top_run_share: float
    root_cause: str
    subs: _containers.RepeatedCompositeFieldContainer[UsageToolSub]
    guesses: _containers.RepeatedCompositeFieldContainer[UsageToolGuess]
    printed_tokens: int
    read_tokens: int
    shares: _containers.RepeatedCompositeFieldContainer[UsageToolShare]
    score: float
    def __init__(self, fingerprint: _Optional[str] = ..., kind: _Optional[str] = ..., owner: _Optional[str] = ..., owner_kind: _Optional[str] = ..., command_path: _Optional[str] = ..., description: _Optional[str] = ..., calls: _Optional[int] = ..., runs: _Optional[int] = ..., cost_unit: _Optional[str] = ..., cost: _Optional[int] = ..., cost_share: _Optional[float] = ..., signature: _Optional[str] = ..., example_commands: _Optional[_Iterable[str]] = ..., evidence_refs: _Optional[_Iterable[_Union[UsageEvidenceRef, _Mapping]]] = ..., status: _Optional[str] = ..., bug_id: _Optional[str] = ..., status_at: _Optional[str] = ..., top_run_share: _Optional[float] = ..., root_cause: _Optional[str] = ..., subs: _Optional[_Iterable[_Union[UsageToolSub, _Mapping]]] = ..., guesses: _Optional[_Iterable[_Union[UsageToolGuess, _Mapping]]] = ..., printed_tokens: _Optional[int] = ..., read_tokens: _Optional[int] = ..., shares: _Optional[_Iterable[_Union[UsageToolShare, _Mapping]]] = ..., score: _Optional[float] = ...) -> None: ...

class UsageToolShare(_message.Message):
    __slots__ = ("kind", "unit", "cost", "share", "base", "sentence")
    KIND_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    COST_FIELD_NUMBER: _ClassVar[int]
    SHARE_FIELD_NUMBER: _ClassVar[int]
    BASE_FIELD_NUMBER: _ClassVar[int]
    SENTENCE_FIELD_NUMBER: _ClassVar[int]
    kind: str
    unit: str
    cost: int
    share: float
    base: str
    sentence: str
    def __init__(self, kind: _Optional[str] = ..., unit: _Optional[str] = ..., cost: _Optional[int] = ..., share: _Optional[float] = ..., base: _Optional[str] = ..., sentence: _Optional[str] = ...) -> None: ...

class UsageInterventionCategory(_message.Message):
    __slots__ = ("category", "owner", "interventions", "per_agent_hour", "repeats", "work_requests", "work_tool_calls", "work_tokens", "pooled", "hints")
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    INTERVENTIONS_FIELD_NUMBER: _ClassVar[int]
    PER_AGENT_HOUR_FIELD_NUMBER: _ClassVar[int]
    REPEATS_FIELD_NUMBER: _ClassVar[int]
    WORK_REQUESTS_FIELD_NUMBER: _ClassVar[int]
    WORK_TOOL_CALLS_FIELD_NUMBER: _ClassVar[int]
    WORK_TOKENS_FIELD_NUMBER: _ClassVar[int]
    POOLED_FIELD_NUMBER: _ClassVar[int]
    HINTS_FIELD_NUMBER: _ClassVar[int]
    category: str
    owner: str
    interventions: int
    per_agent_hour: float
    repeats: int
    work_requests: int
    work_tool_calls: int
    work_tokens: int
    pooled: bool
    hints: _containers.RepeatedCompositeFieldContainer[UsageInterventionHint]
    def __init__(self, category: _Optional[str] = ..., owner: _Optional[str] = ..., interventions: _Optional[int] = ..., per_agent_hour: _Optional[float] = ..., repeats: _Optional[int] = ..., work_requests: _Optional[int] = ..., work_tool_calls: _Optional[int] = ..., work_tokens: _Optional[int] = ..., pooled: _Optional[bool] = ..., hints: _Optional[_Iterable[_Union[UsageInterventionHint, _Mapping]]] = ...) -> None: ...

class UsageInterventionHint(_message.Message):
    __slots__ = ("category", "owner", "count")
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    category: str
    owner: str
    count: int
    def __init__(self, category: _Optional[str] = ..., owner: _Optional[str] = ..., count: _Optional[int] = ...) -> None: ...

class UsageInterventionExample(_message.Message):
    __slots__ = ("event_id", "run_id", "occurred_at", "category", "probability", "owner", "text", "work_requests", "work_tool_calls", "work_tokens", "repeat_of_event_id", "pooled")
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    OCCURRED_AT_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    PROBABILITY_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    WORK_REQUESTS_FIELD_NUMBER: _ClassVar[int]
    WORK_TOOL_CALLS_FIELD_NUMBER: _ClassVar[int]
    WORK_TOKENS_FIELD_NUMBER: _ClassVar[int]
    REPEAT_OF_EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    POOLED_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    run_id: str
    occurred_at: str
    category: str
    probability: float
    owner: str
    text: str
    work_requests: int
    work_tool_calls: int
    work_tokens: int
    repeat_of_event_id: str
    pooled: bool
    def __init__(self, event_id: _Optional[str] = ..., run_id: _Optional[str] = ..., occurred_at: _Optional[str] = ..., category: _Optional[str] = ..., probability: _Optional[float] = ..., owner: _Optional[str] = ..., text: _Optional[str] = ..., work_requests: _Optional[int] = ..., work_tool_calls: _Optional[int] = ..., work_tokens: _Optional[int] = ..., repeat_of_event_id: _Optional[str] = ..., pooled: _Optional[bool] = ...) -> None: ...

class UsageInterventions(_message.Message):
    __slots__ = ("categories", "examples", "operator_messages", "classified", "coverage", "agent_hours", "unshown_reason", "shown")
    CATEGORIES_FIELD_NUMBER: _ClassVar[int]
    EXAMPLES_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_MESSAGES_FIELD_NUMBER: _ClassVar[int]
    CLASSIFIED_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    AGENT_HOURS_FIELD_NUMBER: _ClassVar[int]
    UNSHOWN_REASON_FIELD_NUMBER: _ClassVar[int]
    SHOWN_FIELD_NUMBER: _ClassVar[int]
    categories: _containers.RepeatedCompositeFieldContainer[UsageInterventionCategory]
    examples: _containers.RepeatedCompositeFieldContainer[UsageInterventionExample]
    operator_messages: int
    classified: int
    coverage: float
    agent_hours: float
    unshown_reason: str
    shown: int
    def __init__(self, categories: _Optional[_Iterable[_Union[UsageInterventionCategory, _Mapping]]] = ..., examples: _Optional[_Iterable[_Union[UsageInterventionExample, _Mapping]]] = ..., operator_messages: _Optional[int] = ..., classified: _Optional[int] = ..., coverage: _Optional[float] = ..., agent_hours: _Optional[float] = ..., unshown_reason: _Optional[str] = ..., shown: _Optional[int] = ...) -> None: ...

class UsageFrictionResponse(_message.Message):
    __slots__ = ("interventions", "tool_findings", "below_floor", "beyond_limit", "materiality_score", "coverage", "window_start", "open", "reported", "dismissed", "generated_at", "validity", "provenance", "definition_id")
    INTERVENTIONS_FIELD_NUMBER: _ClassVar[int]
    TOOL_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    BELOW_FLOOR_FIELD_NUMBER: _ClassVar[int]
    BEYOND_LIMIT_FIELD_NUMBER: _ClassVar[int]
    MATERIALITY_SCORE_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    WINDOW_START_FIELD_NUMBER: _ClassVar[int]
    OPEN_FIELD_NUMBER: _ClassVar[int]
    REPORTED_FIELD_NUMBER: _ClassVar[int]
    DISMISSED_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    interventions: UsageInterventions
    tool_findings: _containers.RepeatedCompositeFieldContainer[UsageToolFinding]
    below_floor: int
    beyond_limit: int
    materiality_score: float
    coverage: ToolFrictionCoverage
    window_start: str
    open: int
    reported: int
    dismissed: int
    generated_at: str
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, interventions: _Optional[_Union[UsageInterventions, _Mapping]] = ..., tool_findings: _Optional[_Iterable[_Union[UsageToolFinding, _Mapping]]] = ..., below_floor: _Optional[int] = ..., beyond_limit: _Optional[int] = ..., materiality_score: _Optional[float] = ..., coverage: _Optional[_Union[ToolFrictionCoverage, _Mapping]] = ..., window_start: _Optional[str] = ..., open: _Optional[int] = ..., reported: _Optional[int] = ..., dismissed: _Optional[int] = ..., generated_at: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("runs", "accepted_units", "priced_runs", "tokens", "weighted_tokens", "points", "points_per_accepted", "tokens_per_accepted", "completed_runs", "window_start", "window_end", "unknown_units")
    RUNS_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_UNITS_FIELD_NUMBER: _ClassVar[int]
    PRICED_RUNS_FIELD_NUMBER: _ClassVar[int]
    TOKENS_FIELD_NUMBER: _ClassVar[int]
    WEIGHTED_TOKENS_FIELD_NUMBER: _ClassVar[int]
    POINTS_FIELD_NUMBER: _ClassVar[int]
    POINTS_PER_ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    TOKENS_PER_ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_RUNS_FIELD_NUMBER: _ClassVar[int]
    WINDOW_START_FIELD_NUMBER: _ClassVar[int]
    WINDOW_END_FIELD_NUMBER: _ClassVar[int]
    UNKNOWN_UNITS_FIELD_NUMBER: _ClassVar[int]
    runs: int
    accepted_units: int
    priced_runs: int
    tokens: UsageTokens
    weighted_tokens: float
    points: float
    points_per_accepted: float
    tokens_per_accepted: float
    completed_runs: int
    window_start: str
    window_end: str
    unknown_units: int
    def __init__(self, runs: _Optional[int] = ..., accepted_units: _Optional[int] = ..., priced_runs: _Optional[int] = ..., tokens: _Optional[_Union[UsageTokens, _Mapping]] = ..., weighted_tokens: _Optional[float] = ..., points: _Optional[float] = ..., points_per_accepted: _Optional[float] = ..., tokens_per_accepted: _Optional[float] = ..., completed_runs: _Optional[int] = ..., window_start: _Optional[str] = ..., window_end: _Optional[str] = ..., unknown_units: _Optional[int] = ...) -> None: ...

class UsageCompareResponse(_message.Message):
    __slots__ = ("left", "right", "change_binding", "points_per_accepted_change", "verdict", "caveats", "validity", "provenance", "definition_id")
    LEFT_FIELD_NUMBER: _ClassVar[int]
    RIGHT_FIELD_NUMBER: _ClassVar[int]
    CHANGE_BINDING_FIELD_NUMBER: _ClassVar[int]
    POINTS_PER_ACCEPTED_CHANGE_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    CAVEATS_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    left: UsageCompareSide
    right: UsageCompareSide
    change_binding: str
    points_per_accepted_change: float
    verdict: str
    caveats: _containers.RepeatedScalarFieldContainer[str]
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, left: _Optional[_Union[UsageCompareSide, _Mapping]] = ..., right: _Optional[_Union[UsageCompareSide, _Mapping]] = ..., change_binding: _Optional[str] = ..., points_per_accepted_change: _Optional[float] = ..., verdict: _Optional[str] = ..., caveats: _Optional[_Iterable[str]] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class FallbackInsightsRequest(_message.Message):
    __slots__ = ("window",)
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ...) -> None: ...

class FallbackPairRow(_message.Message):
    __slots__ = ("to", "reason", "count")
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    to: str
    reason: str
    count: int
    def __init__(self, to: _Optional[str] = ..., reason: _Optional[str] = ..., count: _Optional[int] = ..., **kwargs) -> None: ...

class FallbackCountRow(_message.Message):
    __slots__ = ("key", "count")
    KEY_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    key: str
    count: int
    def __init__(self, key: _Optional[str] = ..., count: _Optional[int] = ...) -> None: ...

class FallbackInsightsResponse(_message.Message):
    __slots__ = ("runner_attempts", "runner_exhausted", "model_attempts", "model_exhausted", "runner_pairs", "model_pairs", "runner_reasons", "model_reasons", "policy_candidate_events", "policy_outcomes", "policy_failure_classes", "validity", "provenance", "definition_id")
    RUNNER_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    RUNNER_EXHAUSTED_FIELD_NUMBER: _ClassVar[int]
    MODEL_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    MODEL_EXHAUSTED_FIELD_NUMBER: _ClassVar[int]
    RUNNER_PAIRS_FIELD_NUMBER: _ClassVar[int]
    MODEL_PAIRS_FIELD_NUMBER: _ClassVar[int]
    RUNNER_REASONS_FIELD_NUMBER: _ClassVar[int]
    MODEL_REASONS_FIELD_NUMBER: _ClassVar[int]
    POLICY_CANDIDATE_EVENTS_FIELD_NUMBER: _ClassVar[int]
    POLICY_OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    POLICY_FAILURE_CLASSES_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    runner_attempts: int
    runner_exhausted: int
    model_attempts: int
    model_exhausted: int
    runner_pairs: _containers.RepeatedCompositeFieldContainer[FallbackPairRow]
    model_pairs: _containers.RepeatedCompositeFieldContainer[FallbackPairRow]
    runner_reasons: _containers.RepeatedCompositeFieldContainer[FallbackCountRow]
    model_reasons: _containers.RepeatedCompositeFieldContainer[FallbackCountRow]
    policy_candidate_events: int
    policy_outcomes: _containers.RepeatedCompositeFieldContainer[FallbackCountRow]
    policy_failure_classes: _containers.RepeatedCompositeFieldContainer[FallbackCountRow]
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, runner_attempts: _Optional[int] = ..., runner_exhausted: _Optional[int] = ..., model_attempts: _Optional[int] = ..., model_exhausted: _Optional[int] = ..., runner_pairs: _Optional[_Iterable[_Union[FallbackPairRow, _Mapping]]] = ..., model_pairs: _Optional[_Iterable[_Union[FallbackPairRow, _Mapping]]] = ..., runner_reasons: _Optional[_Iterable[_Union[FallbackCountRow, _Mapping]]] = ..., model_reasons: _Optional[_Iterable[_Union[FallbackCountRow, _Mapping]]] = ..., policy_candidate_events: _Optional[int] = ..., policy_outcomes: _Optional[_Iterable[_Union[FallbackCountRow, _Mapping]]] = ..., policy_failure_classes: _Optional[_Iterable[_Union[FallbackCountRow, _Mapping]]] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class UsageModeScope(_message.Message):
    __slots__ = ("runner", "role", "profile")
    RUNNER_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    runner: str
    role: str
    profile: str
    def __init__(self, runner: _Optional[str] = ..., role: _Optional[str] = ..., profile: _Optional[str] = ...) -> None: ...

class UsageModeSettings(_message.Message):
    __slots__ = ("runner", "model_map", "session_model", "effort", "context_limit_tokens")
    class ModelMapEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    RUNNER_FIELD_NUMBER: _ClassVar[int]
    MODEL_MAP_FIELD_NUMBER: _ClassVar[int]
    SESSION_MODEL_FIELD_NUMBER: _ClassVar[int]
    EFFORT_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_LIMIT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    runner: str
    model_map: _containers.ScalarMap[str, str]
    session_model: str
    effort: str
    context_limit_tokens: int
    def __init__(self, runner: _Optional[str] = ..., model_map: _Optional[_Mapping[str, str]] = ..., session_model: _Optional[str] = ..., effort: _Optional[str] = ..., context_limit_tokens: _Optional[int] = ...) -> None: ...

class UsageMode(_message.Message):
    __slots__ = ("id", "label", "kind", "scope", "started_at", "started_by", "source", "until_kind", "until_provider", "until_window_minutes", "expires_at", "exempt_roles", "settings", "learn")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    STARTED_BY_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    UNTIL_KIND_FIELD_NUMBER: _ClassVar[int]
    UNTIL_PROVIDER_FIELD_NUMBER: _ClassVar[int]
    UNTIL_WINDOW_MINUTES_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    EXEMPT_ROLES_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    LEARN_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    kind: str
    scope: UsageModeScope
    started_at: str
    started_by: str
    source: str
    until_kind: str
    until_provider: str
    until_window_minutes: int
    expires_at: str
    exempt_roles: _containers.RepeatedScalarFieldContainer[str]
    settings: _containers.RepeatedCompositeFieldContainer[UsageModeSettings]
    learn: UsageModeLearn
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., kind: _Optional[str] = ..., scope: _Optional[_Union[UsageModeScope, _Mapping]] = ..., started_at: _Optional[str] = ..., started_by: _Optional[str] = ..., source: _Optional[str] = ..., until_kind: _Optional[str] = ..., until_provider: _Optional[str] = ..., until_window_minutes: _Optional[int] = ..., expires_at: _Optional[str] = ..., exempt_roles: _Optional[_Iterable[str]] = ..., settings: _Optional[_Iterable[_Union[UsageModeSettings, _Mapping]]] = ..., learn: _Optional[_Union[UsageModeLearn, _Mapping]] = ...) -> None: ...

class UsageModeLearn(_message.Message):
    __slots__ = ("split_percent", "challenger", "sessions_opt_in", "block_days", "paused", "paused_reason")
    SPLIT_PERCENT_FIELD_NUMBER: _ClassVar[int]
    CHALLENGER_FIELD_NUMBER: _ClassVar[int]
    SESSIONS_OPT_IN_FIELD_NUMBER: _ClassVar[int]
    BLOCK_DAYS_FIELD_NUMBER: _ClassVar[int]
    PAUSED_FIELD_NUMBER: _ClassVar[int]
    PAUSED_REASON_FIELD_NUMBER: _ClassVar[int]
    split_percent: int
    challenger: _containers.RepeatedCompositeFieldContainer[UsageModeSettings]
    sessions_opt_in: bool
    block_days: int
    paused: bool
    paused_reason: str
    def __init__(self, split_percent: _Optional[int] = ..., challenger: _Optional[_Iterable[_Union[UsageModeSettings, _Mapping]]] = ..., sessions_opt_in: _Optional[bool] = ..., block_days: _Optional[int] = ..., paused: _Optional[bool] = ..., paused_reason: _Optional[str] = ...) -> None: ...

class UsageModeBurn(_message.Message):
    __slots__ = ("provider", "pool", "window", "observed_increase_points", "attributed_points", "unattributed_points")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    POOL_FIELD_NUMBER: _ClassVar[int]
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_INCREASE_POINTS_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTED_POINTS_FIELD_NUMBER: _ClassVar[int]
    UNATTRIBUTED_POINTS_FIELD_NUMBER: _ClassVar[int]
    provider: str
    pool: str
    window: str
    observed_increase_points: float
    attributed_points: float
    unattributed_points: float
    def __init__(self, provider: _Optional[str] = ..., pool: _Optional[str] = ..., window: _Optional[str] = ..., observed_increase_points: _Optional[float] = ..., attributed_points: _Optional[float] = ..., unattributed_points: _Optional[float] = ...) -> None: ...

class UsageModeStatus(_message.Message):
    __slots__ = ("mode", "seconds_left", "burn")
    MODE_FIELD_NUMBER: _ClassVar[int]
    SECONDS_LEFT_FIELD_NUMBER: _ClassVar[int]
    BURN_FIELD_NUMBER: _ClassVar[int]
    mode: UsageMode
    seconds_left: int
    burn: _containers.RepeatedCompositeFieldContainer[UsageModeBurn]
    def __init__(self, mode: _Optional[_Union[UsageMode, _Mapping]] = ..., seconds_left: _Optional[int] = ..., burn: _Optional[_Iterable[_Union[UsageModeBurn, _Mapping]]] = ...) -> None: ...

class UsageModeReachRow(_message.Message):
    __slots__ = ("provider", "pool", "window", "resets_at", "observed_increase_points", "operator_points", "profile_points", "unattributed_points", "operator_share", "profile_share", "unattributed_share")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    POOL_FIELD_NUMBER: _ClassVar[int]
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    RESETS_AT_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_INCREASE_POINTS_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_POINTS_FIELD_NUMBER: _ClassVar[int]
    PROFILE_POINTS_FIELD_NUMBER: _ClassVar[int]
    UNATTRIBUTED_POINTS_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_SHARE_FIELD_NUMBER: _ClassVar[int]
    PROFILE_SHARE_FIELD_NUMBER: _ClassVar[int]
    UNATTRIBUTED_SHARE_FIELD_NUMBER: _ClassVar[int]
    provider: str
    pool: str
    window: str
    resets_at: str
    observed_increase_points: float
    operator_points: float
    profile_points: float
    unattributed_points: float
    operator_share: float
    profile_share: float
    unattributed_share: float
    def __init__(self, provider: _Optional[str] = ..., pool: _Optional[str] = ..., window: _Optional[str] = ..., resets_at: _Optional[str] = ..., observed_increase_points: _Optional[float] = ..., operator_points: _Optional[float] = ..., profile_points: _Optional[float] = ..., unattributed_points: _Optional[float] = ..., operator_share: _Optional[float] = ..., profile_share: _Optional[float] = ..., unattributed_share: _Optional[float] = ...) -> None: ...

class ListUsageModesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListUsageModesResponse(_message.Message):
    __slots__ = ("generated_at", "modes", "reach", "reach_notes", "mode_file", "periods")
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    MODES_FIELD_NUMBER: _ClassVar[int]
    REACH_FIELD_NUMBER: _ClassVar[int]
    REACH_NOTES_FIELD_NUMBER: _ClassVar[int]
    MODE_FILE_FIELD_NUMBER: _ClassVar[int]
    PERIODS_FIELD_NUMBER: _ClassVar[int]
    generated_at: str
    modes: _containers.RepeatedCompositeFieldContainer[UsageModeStatus]
    reach: _containers.RepeatedCompositeFieldContainer[UsageModeReachRow]
    reach_notes: _containers.RepeatedScalarFieldContainer[str]
    mode_file: str
    periods: _containers.RepeatedCompositeFieldContainer[UsageModePeriod]
    def __init__(self, generated_at: _Optional[str] = ..., modes: _Optional[_Iterable[_Union[UsageModeStatus, _Mapping]]] = ..., reach: _Optional[_Iterable[_Union[UsageModeReachRow, _Mapping]]] = ..., reach_notes: _Optional[_Iterable[str]] = ..., mode_file: _Optional[str] = ..., periods: _Optional[_Iterable[_Union[UsageModePeriod, _Mapping]]] = ...) -> None: ...

class StartUsageModeRequest(_message.Message):
    __slots__ = ("preset", "model", "effort", "context_limit_tokens", "runner", "role", "profile", "until", "label")
    PRESET_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    EFFORT_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_LIMIT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    RUNNER_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    UNTIL_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    preset: str
    model: str
    effort: str
    context_limit_tokens: int
    runner: str
    role: str
    profile: str
    until: str
    label: str
    def __init__(self, preset: _Optional[str] = ..., model: _Optional[str] = ..., effort: _Optional[str] = ..., context_limit_tokens: _Optional[int] = ..., runner: _Optional[str] = ..., role: _Optional[str] = ..., profile: _Optional[str] = ..., until: _Optional[str] = ..., label: _Optional[str] = ...) -> None: ...

class StartUsageModeResponse(_message.Message):
    __slots__ = ("mode", "replaced_ids", "notes")
    MODE_FIELD_NUMBER: _ClassVar[int]
    REPLACED_IDS_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    mode: UsageMode
    replaced_ids: _containers.RepeatedScalarFieldContainer[str]
    notes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, mode: _Optional[_Union[UsageMode, _Mapping]] = ..., replaced_ids: _Optional[_Iterable[str]] = ..., notes: _Optional[_Iterable[str]] = ...) -> None: ...

class EndUsageModeRequest(_message.Message):
    __slots__ = ("ids", "all")
    IDS_FIELD_NUMBER: _ClassVar[int]
    ALL_FIELD_NUMBER: _ClassVar[int]
    ids: _containers.RepeatedScalarFieldContainer[str]
    all: bool
    def __init__(self, ids: _Optional[_Iterable[str]] = ..., all: _Optional[bool] = ...) -> None: ...

class EndUsageModeResponse(_message.Message):
    __slots__ = ("ended_ids", "not_found_ids")
    ENDED_IDS_FIELD_NUMBER: _ClassVar[int]
    NOT_FOUND_IDS_FIELD_NUMBER: _ClassVar[int]
    ended_ids: _containers.RepeatedScalarFieldContainer[str]
    not_found_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, ended_ids: _Optional[_Iterable[str]] = ..., not_found_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class ExtendUsageModeRequest(_message.Message):
    __slots__ = ("id", "to")
    ID_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    id: str
    to: str
    def __init__(self, id: _Optional[str] = ..., to: _Optional[str] = ...) -> None: ...

class ExtendUsageModeResponse(_message.Message):
    __slots__ = ("mode",)
    MODE_FIELD_NUMBER: _ClassVar[int]
    mode: UsageMode
    def __init__(self, mode: _Optional[_Union[UsageMode, _Mapping]] = ...) -> None: ...

class RebaseUsageModeRunsRequest(_message.Message):
    __slots__ = ("mode_id", "run_ids", "dry_run")
    MODE_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_IDS_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    mode_id: str
    run_ids: _containers.RepeatedScalarFieldContainer[str]
    dry_run: bool
    def __init__(self, mode_id: _Optional[str] = ..., run_ids: _Optional[_Iterable[str]] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class UsageModeRebaseResult(_message.Message):
    __slots__ = ("run_id", "status", "reason", "from_model", "to_model", "estimated_reread_tokens")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    FROM_MODEL_FIELD_NUMBER: _ClassVar[int]
    TO_MODEL_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_REREAD_TOKENS_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    status: str
    reason: str
    from_model: str
    to_model: str
    estimated_reread_tokens: int
    def __init__(self, run_id: _Optional[str] = ..., status: _Optional[str] = ..., reason: _Optional[str] = ..., from_model: _Optional[str] = ..., to_model: _Optional[str] = ..., estimated_reread_tokens: _Optional[int] = ...) -> None: ...

class RebaseUsageModeRunsResponse(_message.Message):
    __slots__ = ("dry_run", "results")
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    results: _containers.RepeatedCompositeFieldContainer[UsageModeRebaseResult]
    def __init__(self, dry_run: _Optional[bool] = ..., results: _Optional[_Iterable[_Union[UsageModeRebaseResult, _Mapping]]] = ...) -> None: ...

class UsageParityRequest(_message.Message):
    __slots__ = ("window",)
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ...) -> None: ...

class UsageParityMeasure(_message.Message):
    __slots__ = ("name", "numerator", "denominator", "share", "not_applicable", "reason")
    NAME_FIELD_NUMBER: _ClassVar[int]
    NUMERATOR_FIELD_NUMBER: _ClassVar[int]
    DENOMINATOR_FIELD_NUMBER: _ClassVar[int]
    SHARE_FIELD_NUMBER: _ClassVar[int]
    NOT_APPLICABLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    name: str
    numerator: int
    denominator: int
    share: float
    not_applicable: int
    reason: str
    def __init__(self, name: _Optional[str] = ..., numerator: _Optional[int] = ..., denominator: _Optional[int] = ..., share: _Optional[float] = ..., not_applicable: _Optional[int] = ..., reason: _Optional[str] = ...) -> None: ...

class UsageParityRow(_message.Message):
    __slots__ = ("path", "harness", "runs", "eligible_runs", "no_requests", "excluded", "measures")
    PATH_FIELD_NUMBER: _ClassVar[int]
    HARNESS_FIELD_NUMBER: _ClassVar[int]
    RUNS_FIELD_NUMBER: _ClassVar[int]
    ELIGIBLE_RUNS_FIELD_NUMBER: _ClassVar[int]
    NO_REQUESTS_FIELD_NUMBER: _ClassVar[int]
    EXCLUDED_FIELD_NUMBER: _ClassVar[int]
    MEASURES_FIELD_NUMBER: _ClassVar[int]
    path: str
    harness: str
    runs: int
    eligible_runs: int
    no_requests: int
    excluded: _containers.RepeatedCompositeFieldContainer[UsageParityCount]
    measures: _containers.RepeatedCompositeFieldContainer[UsageParityMeasure]
    def __init__(self, path: _Optional[str] = ..., harness: _Optional[str] = ..., runs: _Optional[int] = ..., eligible_runs: _Optional[int] = ..., no_requests: _Optional[int] = ..., excluded: _Optional[_Iterable[_Union[UsageParityCount, _Mapping]]] = ..., measures: _Optional[_Iterable[_Union[UsageParityMeasure, _Mapping]]] = ...) -> None: ...

class UsageParityCount(_message.Message):
    __slots__ = ("name", "count")
    NAME_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    name: str
    count: int
    def __init__(self, name: _Optional[str] = ..., count: _Optional[int] = ...) -> None: ...

class UsageParityGap(_message.Message):
    __slots__ = ("subject", "status", "count", "reason")
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    subject: str
    status: str
    count: int
    reason: str
    def __init__(self, subject: _Optional[str] = ..., status: _Optional[str] = ..., count: _Optional[int] = ..., reason: _Optional[str] = ...) -> None: ...

class UsageParityResponse(_message.Message):
    __slots__ = ("generated_at", "window_start", "window_end", "rows", "gaps", "backfill_runs", "ledger_pending", "transcript_roots", "validity", "provenance", "definition_id")
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    WINDOW_START_FIELD_NUMBER: _ClassVar[int]
    WINDOW_END_FIELD_NUMBER: _ClassVar[int]
    ROWS_FIELD_NUMBER: _ClassVar[int]
    GAPS_FIELD_NUMBER: _ClassVar[int]
    BACKFILL_RUNS_FIELD_NUMBER: _ClassVar[int]
    LEDGER_PENDING_FIELD_NUMBER: _ClassVar[int]
    TRANSCRIPT_ROOTS_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    generated_at: str
    window_start: str
    window_end: str
    rows: _containers.RepeatedCompositeFieldContainer[UsageParityRow]
    gaps: _containers.RepeatedCompositeFieldContainer[UsageParityGap]
    backfill_runs: int
    ledger_pending: int
    transcript_roots: _containers.RepeatedCompositeFieldContainer[UsageCoverageRoot]
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, generated_at: _Optional[str] = ..., window_start: _Optional[str] = ..., window_end: _Optional[str] = ..., rows: _Optional[_Iterable[_Union[UsageParityRow, _Mapping]]] = ..., gaps: _Optional[_Iterable[_Union[UsageParityGap, _Mapping]]] = ..., backfill_runs: _Optional[int] = ..., ledger_pending: _Optional[int] = ..., transcript_roots: _Optional[_Iterable[_Union[UsageCoverageRoot, _Mapping]]] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class UsageOutcomesRequest(_message.Message):
    __slots__ = ("window", "by", "baseline")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    BY_FIELD_NUMBER: _ClassVar[int]
    BASELINE_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    by: str
    baseline: str
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., by: _Optional[str] = ..., baseline: _Optional[str] = ...) -> None: ...

class UsageRange(_message.Message):
    __slots__ = ("available", "value", "low", "high", "samples")
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    LOW_FIELD_NUMBER: _ClassVar[int]
    HIGH_FIELD_NUMBER: _ClassVar[int]
    SAMPLES_FIELD_NUMBER: _ClassVar[int]
    available: bool
    value: float
    low: float
    high: float
    samples: int
    def __init__(self, available: _Optional[bool] = ..., value: _Optional[float] = ..., low: _Optional[float] = ..., high: _Optional[float] = ..., samples: _Optional[int] = ...) -> None: ...

class UsageEvidenceCount(_message.Message):
    __slots__ = ("source", "units")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    UNITS_FIELD_NUMBER: _ClassVar[int]
    source: str
    units: int
    def __init__(self, source: _Optional[str] = ..., units: _Optional[int] = ...) -> None: ...

class UsageOutcomeRow(_message.Message):
    __slots__ = ("key", "runner", "status", "status_reason", "units", "runs", "accepted", "corrected", "failed", "abandoned", "unknown", "unknown_rate", "rework_rate", "points_total", "points_known", "points_per_accepted", "consumption_per_accepted", "points_per_run", "consumption_per_run", "active_minutes_per_run", "wall_hours_mean", "wall_hours_median", "runs_over_24h", "idle_share", "baseline", "verdict", "cheaper_by", "consumption_ratio", "run_length_ratio", "break_even_ratio", "price_source", "evidence", "stratum", "completed", "points_per_completed", "completed_baseline", "completed_reading", "cheaper_by_completed", "model_price_ratio", "model_price_source")
    KEY_FIELD_NUMBER: _ClassVar[int]
    RUNNER_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STATUS_REASON_FIELD_NUMBER: _ClassVar[int]
    UNITS_FIELD_NUMBER: _ClassVar[int]
    RUNS_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    CORRECTED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    ABANDONED_FIELD_NUMBER: _ClassVar[int]
    UNKNOWN_FIELD_NUMBER: _ClassVar[int]
    UNKNOWN_RATE_FIELD_NUMBER: _ClassVar[int]
    REWORK_RATE_FIELD_NUMBER: _ClassVar[int]
    POINTS_TOTAL_FIELD_NUMBER: _ClassVar[int]
    POINTS_KNOWN_FIELD_NUMBER: _ClassVar[int]
    POINTS_PER_ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    CONSUMPTION_PER_ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    POINTS_PER_RUN_FIELD_NUMBER: _ClassVar[int]
    CONSUMPTION_PER_RUN_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_MINUTES_PER_RUN_FIELD_NUMBER: _ClassVar[int]
    WALL_HOURS_MEAN_FIELD_NUMBER: _ClassVar[int]
    WALL_HOURS_MEDIAN_FIELD_NUMBER: _ClassVar[int]
    RUNS_OVER_24H_FIELD_NUMBER: _ClassVar[int]
    IDLE_SHARE_FIELD_NUMBER: _ClassVar[int]
    BASELINE_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    CHEAPER_BY_FIELD_NUMBER: _ClassVar[int]
    CONSUMPTION_RATIO_FIELD_NUMBER: _ClassVar[int]
    RUN_LENGTH_RATIO_FIELD_NUMBER: _ClassVar[int]
    BREAK_EVEN_RATIO_FIELD_NUMBER: _ClassVar[int]
    PRICE_SOURCE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    STRATUM_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_FIELD_NUMBER: _ClassVar[int]
    POINTS_PER_COMPLETED_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_BASELINE_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_READING_FIELD_NUMBER: _ClassVar[int]
    CHEAPER_BY_COMPLETED_FIELD_NUMBER: _ClassVar[int]
    MODEL_PRICE_RATIO_FIELD_NUMBER: _ClassVar[int]
    MODEL_PRICE_SOURCE_FIELD_NUMBER: _ClassVar[int]
    key: str
    runner: str
    status: str
    status_reason: str
    units: int
    runs: int
    accepted: int
    corrected: int
    failed: int
    abandoned: int
    unknown: int
    unknown_rate: float
    rework_rate: UsageRange
    points_total: float
    points_known: float
    points_per_accepted: UsageRange
    consumption_per_accepted: UsageRange
    points_per_run: UsageRange
    consumption_per_run: UsageRange
    active_minutes_per_run: UsageRange
    wall_hours_mean: float
    wall_hours_median: float
    runs_over_24h: int
    idle_share: float
    baseline: str
    verdict: str
    cheaper_by: UsageRange
    consumption_ratio: UsageRange
    run_length_ratio: UsageRange
    break_even_ratio: float
    price_source: str
    evidence: _containers.RepeatedCompositeFieldContainer[UsageEvidenceCount]
    stratum: str
    completed: int
    points_per_completed: UsageRange
    completed_baseline: str
    completed_reading: str
    cheaper_by_completed: UsageRange
    model_price_ratio: float
    model_price_source: str
    def __init__(self, key: _Optional[str] = ..., runner: _Optional[str] = ..., status: _Optional[str] = ..., status_reason: _Optional[str] = ..., units: _Optional[int] = ..., runs: _Optional[int] = ..., accepted: _Optional[int] = ..., corrected: _Optional[int] = ..., failed: _Optional[int] = ..., abandoned: _Optional[int] = ..., unknown: _Optional[int] = ..., unknown_rate: _Optional[float] = ..., rework_rate: _Optional[_Union[UsageRange, _Mapping]] = ..., points_total: _Optional[float] = ..., points_known: _Optional[float] = ..., points_per_accepted: _Optional[_Union[UsageRange, _Mapping]] = ..., consumption_per_accepted: _Optional[_Union[UsageRange, _Mapping]] = ..., points_per_run: _Optional[_Union[UsageRange, _Mapping]] = ..., consumption_per_run: _Optional[_Union[UsageRange, _Mapping]] = ..., active_minutes_per_run: _Optional[_Union[UsageRange, _Mapping]] = ..., wall_hours_mean: _Optional[float] = ..., wall_hours_median: _Optional[float] = ..., runs_over_24h: _Optional[int] = ..., idle_share: _Optional[float] = ..., baseline: _Optional[str] = ..., verdict: _Optional[str] = ..., cheaper_by: _Optional[_Union[UsageRange, _Mapping]] = ..., consumption_ratio: _Optional[_Union[UsageRange, _Mapping]] = ..., run_length_ratio: _Optional[_Union[UsageRange, _Mapping]] = ..., break_even_ratio: _Optional[float] = ..., price_source: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[UsageEvidenceCount, _Mapping]]] = ..., stratum: _Optional[str] = ..., completed: _Optional[int] = ..., points_per_completed: _Optional[_Union[UsageRange, _Mapping]] = ..., completed_baseline: _Optional[str] = ..., completed_reading: _Optional[str] = ..., cheaper_by_completed: _Optional[_Union[UsageRange, _Mapping]] = ..., model_price_ratio: _Optional[float] = ..., model_price_source: _Optional[str] = ...) -> None: ...

class UsageRefitRow(_message.Message):
    __slots__ = ("runner", "model", "price_source", "dominant_blocks", "mode_blocks", "implied_points_per_million", "prior_points_per_million", "status", "reason")
    RUNNER_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    PRICE_SOURCE_FIELD_NUMBER: _ClassVar[int]
    DOMINANT_BLOCKS_FIELD_NUMBER: _ClassVar[int]
    MODE_BLOCKS_FIELD_NUMBER: _ClassVar[int]
    IMPLIED_POINTS_PER_MILLION_FIELD_NUMBER: _ClassVar[int]
    PRIOR_POINTS_PER_MILLION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    runner: str
    model: str
    price_source: str
    dominant_blocks: int
    mode_blocks: int
    implied_points_per_million: UsageRange
    prior_points_per_million: float
    status: str
    reason: str
    def __init__(self, runner: _Optional[str] = ..., model: _Optional[str] = ..., price_source: _Optional[str] = ..., dominant_blocks: _Optional[int] = ..., mode_blocks: _Optional[int] = ..., implied_points_per_million: _Optional[_Union[UsageRange, _Mapping]] = ..., prior_points_per_million: _Optional[float] = ..., status: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class UsageModePeriod(_message.Message):
    __slots__ = ("id", "label", "runner", "started_at", "ended_at", "expires_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    RUNNER_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    ENDED_AT_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    runner: str
    started_at: str
    ended_at: str
    expires_at: str
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., runner: _Optional[str] = ..., started_at: _Optional[str] = ..., ended_at: _Optional[str] = ..., expires_at: _Optional[str] = ...) -> None: ...

class UsageSeriesBucket(_message.Message):
    __slots__ = ("at", "points", "runs")
    AT_FIELD_NUMBER: _ClassVar[int]
    POINTS_FIELD_NUMBER: _ClassVar[int]
    RUNS_FIELD_NUMBER: _ClassVar[int]
    at: str
    points: float
    runs: int
    def __init__(self, at: _Optional[str] = ..., points: _Optional[float] = ..., runs: _Optional[int] = ...) -> None: ...

class UsageModelSeries(_message.Message):
    __slots__ = ("model", "runner", "buckets")
    MODEL_FIELD_NUMBER: _ClassVar[int]
    RUNNER_FIELD_NUMBER: _ClassVar[int]
    BUCKETS_FIELD_NUMBER: _ClassVar[int]
    model: str
    runner: str
    buckets: _containers.RepeatedCompositeFieldContainer[UsageSeriesBucket]
    def __init__(self, model: _Optional[str] = ..., runner: _Optional[str] = ..., buckets: _Optional[_Iterable[_Union[UsageSeriesBucket, _Mapping]]] = ...) -> None: ...

class UsageOutcomesResponse(_message.Message):
    __slots__ = ("generated_at", "window_start", "window_end", "by", "rows", "evidence", "paths", "units", "unknown_units", "link_confidence", "linked_units", "notes", "refit", "mode_periods", "series", "design", "completed_units", "spanning_units", "validity", "provenance", "definition_id")
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    WINDOW_START_FIELD_NUMBER: _ClassVar[int]
    WINDOW_END_FIELD_NUMBER: _ClassVar[int]
    BY_FIELD_NUMBER: _ClassVar[int]
    ROWS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    PATHS_FIELD_NUMBER: _ClassVar[int]
    UNITS_FIELD_NUMBER: _ClassVar[int]
    UNKNOWN_UNITS_FIELD_NUMBER: _ClassVar[int]
    LINK_CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    LINKED_UNITS_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    REFIT_FIELD_NUMBER: _ClassVar[int]
    MODE_PERIODS_FIELD_NUMBER: _ClassVar[int]
    SERIES_FIELD_NUMBER: _ClassVar[int]
    DESIGN_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_UNITS_FIELD_NUMBER: _ClassVar[int]
    SPANNING_UNITS_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    generated_at: str
    window_start: str
    window_end: str
    by: str
    rows: _containers.RepeatedCompositeFieldContainer[UsageOutcomeRow]
    evidence: _containers.RepeatedCompositeFieldContainer[UsageEvidenceCount]
    paths: _containers.RepeatedCompositeFieldContainer[UsageEvidenceCount]
    units: int
    unknown_units: int
    link_confidence: float
    linked_units: int
    notes: _containers.RepeatedScalarFieldContainer[str]
    refit: _containers.RepeatedCompositeFieldContainer[UsageRefitRow]
    mode_periods: _containers.RepeatedCompositeFieldContainer[UsageModePeriod]
    series: _containers.RepeatedCompositeFieldContainer[UsageModelSeries]
    design: str
    completed_units: int
    spanning_units: int
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, generated_at: _Optional[str] = ..., window_start: _Optional[str] = ..., window_end: _Optional[str] = ..., by: _Optional[str] = ..., rows: _Optional[_Iterable[_Union[UsageOutcomeRow, _Mapping]]] = ..., evidence: _Optional[_Iterable[_Union[UsageEvidenceCount, _Mapping]]] = ..., paths: _Optional[_Iterable[_Union[UsageEvidenceCount, _Mapping]]] = ..., units: _Optional[int] = ..., unknown_units: _Optional[int] = ..., link_confidence: _Optional[float] = ..., linked_units: _Optional[int] = ..., notes: _Optional[_Iterable[str]] = ..., refit: _Optional[_Iterable[_Union[UsageRefitRow, _Mapping]]] = ..., mode_periods: _Optional[_Iterable[_Union[UsageModePeriod, _Mapping]]] = ..., series: _Optional[_Iterable[_Union[UsageModelSeries, _Mapping]]] = ..., design: _Optional[str] = ..., completed_units: _Optional[int] = ..., spanning_units: _Optional[int] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class UsageExperimentArm(_message.Message):
    __slots__ = ("arm", "units", "known", "accepted", "exceptions", "points_per_accepted", "rework_rate", "completed", "unknown", "total_points", "unknown_points", "points_per_completed", "clusters", "assumed_accepted")
    ARM_FIELD_NUMBER: _ClassVar[int]
    UNITS_FIELD_NUMBER: _ClassVar[int]
    KNOWN_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    EXCEPTIONS_FIELD_NUMBER: _ClassVar[int]
    POINTS_PER_ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    REWORK_RATE_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_FIELD_NUMBER: _ClassVar[int]
    UNKNOWN_FIELD_NUMBER: _ClassVar[int]
    TOTAL_POINTS_FIELD_NUMBER: _ClassVar[int]
    UNKNOWN_POINTS_FIELD_NUMBER: _ClassVar[int]
    POINTS_PER_COMPLETED_FIELD_NUMBER: _ClassVar[int]
    CLUSTERS_FIELD_NUMBER: _ClassVar[int]
    ASSUMED_ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    arm: str
    units: int
    known: int
    accepted: int
    exceptions: int
    points_per_accepted: UsageRange
    rework_rate: UsageRange
    completed: int
    unknown: int
    total_points: float
    unknown_points: float
    points_per_completed: UsageRange
    clusters: int
    assumed_accepted: int
    def __init__(self, arm: _Optional[str] = ..., units: _Optional[int] = ..., known: _Optional[int] = ..., accepted: _Optional[int] = ..., exceptions: _Optional[int] = ..., points_per_accepted: _Optional[_Union[UsageRange, _Mapping]] = ..., rework_rate: _Optional[_Union[UsageRange, _Mapping]] = ..., completed: _Optional[int] = ..., unknown: _Optional[int] = ..., total_points: _Optional[float] = ..., unknown_points: _Optional[float] = ..., points_per_completed: _Optional[_Union[UsageRange, _Mapping]] = ..., clusters: _Optional[int] = ..., assumed_accepted: _Optional[int] = ...) -> None: ...

class UsageExperimentHorizon(_message.Message):
    __slots__ = ("reached", "days", "low_days", "high_days", "per_arm")
    REACHED_FIELD_NUMBER: _ClassVar[int]
    DAYS_FIELD_NUMBER: _ClassVar[int]
    LOW_DAYS_FIELD_NUMBER: _ClassVar[int]
    HIGH_DAYS_FIELD_NUMBER: _ClassVar[int]
    PER_ARM_FIELD_NUMBER: _ClassVar[int]
    reached: bool
    days: float
    low_days: float
    high_days: float
    per_arm: int
    def __init__(self, reached: _Optional[bool] = ..., days: _Optional[float] = ..., low_days: _Optional[float] = ..., high_days: _Optional[float] = ..., per_arm: _Optional[int] = ...) -> None: ...

class UsageExperimentPlan(_message.Message):
    __slots__ = ("units_per_day", "cost_cv", "variance_assumed", "pilot_units", "effect", "detect", "same", "detectable_in_90_days", "power", "notes", "min_accepted_per_arm", "min_effect", "rework_margin", "accepted_share", "completed_share", "power_at_look", "same_at_look", "completed_only", "completed_only_power", "alpha_per_test")
    UNITS_PER_DAY_FIELD_NUMBER: _ClassVar[int]
    COST_CV_FIELD_NUMBER: _ClassVar[int]
    VARIANCE_ASSUMED_FIELD_NUMBER: _ClassVar[int]
    PILOT_UNITS_FIELD_NUMBER: _ClassVar[int]
    EFFECT_FIELD_NUMBER: _ClassVar[int]
    DETECT_FIELD_NUMBER: _ClassVar[int]
    SAME_FIELD_NUMBER: _ClassVar[int]
    DETECTABLE_IN_90_DAYS_FIELD_NUMBER: _ClassVar[int]
    POWER_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    MIN_ACCEPTED_PER_ARM_FIELD_NUMBER: _ClassVar[int]
    MIN_EFFECT_FIELD_NUMBER: _ClassVar[int]
    REWORK_MARGIN_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_SHARE_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_SHARE_FIELD_NUMBER: _ClassVar[int]
    POWER_AT_LOOK_FIELD_NUMBER: _ClassVar[int]
    SAME_AT_LOOK_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_ONLY_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_ONLY_POWER_FIELD_NUMBER: _ClassVar[int]
    ALPHA_PER_TEST_FIELD_NUMBER: _ClassVar[int]
    units_per_day: float
    cost_cv: float
    variance_assumed: bool
    pilot_units: int
    effect: float
    detect: UsageExperimentHorizon
    same: UsageExperimentHorizon
    detectable_in_90_days: float
    power: float
    notes: _containers.RepeatedScalarFieldContainer[str]
    min_accepted_per_arm: int
    min_effect: float
    rework_margin: float
    accepted_share: float
    completed_share: float
    power_at_look: float
    same_at_look: float
    completed_only: UsageExperimentHorizon
    completed_only_power: float
    alpha_per_test: float
    def __init__(self, units_per_day: _Optional[float] = ..., cost_cv: _Optional[float] = ..., variance_assumed: _Optional[bool] = ..., pilot_units: _Optional[int] = ..., effect: _Optional[float] = ..., detect: _Optional[_Union[UsageExperimentHorizon, _Mapping]] = ..., same: _Optional[_Union[UsageExperimentHorizon, _Mapping]] = ..., detectable_in_90_days: _Optional[float] = ..., power: _Optional[float] = ..., notes: _Optional[_Iterable[str]] = ..., min_accepted_per_arm: _Optional[int] = ..., min_effect: _Optional[float] = ..., rework_margin: _Optional[float] = ..., accepted_share: _Optional[float] = ..., completed_share: _Optional[float] = ..., power_at_look: _Optional[float] = ..., same_at_look: _Optional[float] = ..., completed_only: _Optional[_Union[UsageExperimentHorizon, _Mapping]] = ..., completed_only_power: _Optional[float] = ..., alpha_per_test: _Optional[float] = ...) -> None: ...

class UsageRoleRecommendation(_message.Message):
    __slots__ = ("available", "role", "runner", "from_model", "to_model", "verdict", "evidence", "policy_file", "policy_ops", "policy_note", "decision_id", "decision_state", "decision_note", "applied")
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    RUNNER_FIELD_NUMBER: _ClassVar[int]
    FROM_MODEL_FIELD_NUMBER: _ClassVar[int]
    TO_MODEL_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    POLICY_FILE_FIELD_NUMBER: _ClassVar[int]
    POLICY_OPS_FIELD_NUMBER: _ClassVar[int]
    POLICY_NOTE_FIELD_NUMBER: _ClassVar[int]
    DECISION_ID_FIELD_NUMBER: _ClassVar[int]
    DECISION_STATE_FIELD_NUMBER: _ClassVar[int]
    DECISION_NOTE_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    available: bool
    role: str
    runner: str
    from_model: str
    to_model: str
    verdict: str
    evidence: str
    policy_file: str
    policy_ops: _containers.RepeatedScalarFieldContainer[str]
    policy_note: str
    decision_id: str
    decision_state: str
    decision_note: str
    applied: bool
    def __init__(self, available: _Optional[bool] = ..., role: _Optional[str] = ..., runner: _Optional[str] = ..., from_model: _Optional[str] = ..., to_model: _Optional[str] = ..., verdict: _Optional[str] = ..., evidence: _Optional[str] = ..., policy_file: _Optional[str] = ..., policy_ops: _Optional[_Iterable[str]] = ..., policy_note: _Optional[str] = ..., decision_id: _Optional[str] = ..., decision_state: _Optional[str] = ..., decision_note: _Optional[str] = ..., applied: _Optional[bool] = ...) -> None: ...

class UsageExperiment(_message.Message):
    __slots__ = ("mode_id", "label", "role", "runner", "incumbent_model", "challenger_model", "design", "split_percent", "sessions_opt_in", "block_days", "active", "paused", "paused_reason", "started_at", "ends_at", "incumbent", "challenger", "cost_ratio", "cost_p_value", "rework_difference", "rework_p_value", "verdict", "reason", "need_accepted", "plan", "recommendation", "look_reached", "planned_accepted", "look_at", "alpha_per_test", "comparisons", "spanning_units", "units_after_look", "unknown_rate_difference", "cost_bound", "decision_range", "decision_level")
    MODE_ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    RUNNER_FIELD_NUMBER: _ClassVar[int]
    INCUMBENT_MODEL_FIELD_NUMBER: _ClassVar[int]
    CHALLENGER_MODEL_FIELD_NUMBER: _ClassVar[int]
    DESIGN_FIELD_NUMBER: _ClassVar[int]
    SPLIT_PERCENT_FIELD_NUMBER: _ClassVar[int]
    SESSIONS_OPT_IN_FIELD_NUMBER: _ClassVar[int]
    BLOCK_DAYS_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    PAUSED_FIELD_NUMBER: _ClassVar[int]
    PAUSED_REASON_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    ENDS_AT_FIELD_NUMBER: _ClassVar[int]
    INCUMBENT_FIELD_NUMBER: _ClassVar[int]
    CHALLENGER_FIELD_NUMBER: _ClassVar[int]
    COST_RATIO_FIELD_NUMBER: _ClassVar[int]
    COST_P_VALUE_FIELD_NUMBER: _ClassVar[int]
    REWORK_DIFFERENCE_FIELD_NUMBER: _ClassVar[int]
    REWORK_P_VALUE_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    NEED_ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    PLAN_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATION_FIELD_NUMBER: _ClassVar[int]
    LOOK_REACHED_FIELD_NUMBER: _ClassVar[int]
    PLANNED_ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    LOOK_AT_FIELD_NUMBER: _ClassVar[int]
    ALPHA_PER_TEST_FIELD_NUMBER: _ClassVar[int]
    COMPARISONS_FIELD_NUMBER: _ClassVar[int]
    SPANNING_UNITS_FIELD_NUMBER: _ClassVar[int]
    UNITS_AFTER_LOOK_FIELD_NUMBER: _ClassVar[int]
    UNKNOWN_RATE_DIFFERENCE_FIELD_NUMBER: _ClassVar[int]
    COST_BOUND_FIELD_NUMBER: _ClassVar[int]
    DECISION_RANGE_FIELD_NUMBER: _ClassVar[int]
    DECISION_LEVEL_FIELD_NUMBER: _ClassVar[int]
    mode_id: str
    label: str
    role: str
    runner: str
    incumbent_model: str
    challenger_model: str
    design: str
    split_percent: int
    sessions_opt_in: bool
    block_days: int
    active: bool
    paused: bool
    paused_reason: str
    started_at: str
    ends_at: str
    incumbent: UsageExperimentArm
    challenger: UsageExperimentArm
    cost_ratio: UsageRange
    cost_p_value: float
    rework_difference: UsageRange
    rework_p_value: float
    verdict: str
    reason: str
    need_accepted: int
    plan: UsageExperimentPlan
    recommendation: UsageRoleRecommendation
    look_reached: bool
    planned_accepted: int
    look_at: str
    alpha_per_test: float
    comparisons: int
    spanning_units: int
    units_after_look: int
    unknown_rate_difference: float
    cost_bound: UsageRange
    decision_range: UsageRange
    decision_level: float
    def __init__(self, mode_id: _Optional[str] = ..., label: _Optional[str] = ..., role: _Optional[str] = ..., runner: _Optional[str] = ..., incumbent_model: _Optional[str] = ..., challenger_model: _Optional[str] = ..., design: _Optional[str] = ..., split_percent: _Optional[int] = ..., sessions_opt_in: _Optional[bool] = ..., block_days: _Optional[int] = ..., active: _Optional[bool] = ..., paused: _Optional[bool] = ..., paused_reason: _Optional[str] = ..., started_at: _Optional[str] = ..., ends_at: _Optional[str] = ..., incumbent: _Optional[_Union[UsageExperimentArm, _Mapping]] = ..., challenger: _Optional[_Union[UsageExperimentArm, _Mapping]] = ..., cost_ratio: _Optional[_Union[UsageRange, _Mapping]] = ..., cost_p_value: _Optional[float] = ..., rework_difference: _Optional[_Union[UsageRange, _Mapping]] = ..., rework_p_value: _Optional[float] = ..., verdict: _Optional[str] = ..., reason: _Optional[str] = ..., need_accepted: _Optional[int] = ..., plan: _Optional[_Union[UsageExperimentPlan, _Mapping]] = ..., recommendation: _Optional[_Union[UsageRoleRecommendation, _Mapping]] = ..., look_reached: _Optional[bool] = ..., planned_accepted: _Optional[int] = ..., look_at: _Optional[str] = ..., alpha_per_test: _Optional[float] = ..., comparisons: _Optional[int] = ..., spanning_units: _Optional[int] = ..., units_after_look: _Optional[int] = ..., unknown_rate_difference: _Optional[float] = ..., cost_bound: _Optional[_Union[UsageRange, _Mapping]] = ..., decision_range: _Optional[_Union[UsageRange, _Mapping]] = ..., decision_level: _Optional[float] = ...) -> None: ...

class UsageExperimentRequest(_message.Message):
    __slots__ = ("mode_id",)
    MODE_ID_FIELD_NUMBER: _ClassVar[int]
    mode_id: str
    def __init__(self, mode_id: _Optional[str] = ...) -> None: ...

class UsageExperimentResponse(_message.Message):
    __slots__ = ("generated_at", "experiments", "validity", "provenance", "definition_id")
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    EXPERIMENTS_FIELD_NUMBER: _ClassVar[int]
    VALIDITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_ID_FIELD_NUMBER: _ClassVar[int]
    generated_at: str
    experiments: _containers.RepeatedCompositeFieldContainer[UsageExperiment]
    validity: MeasureValidity
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, generated_at: _Optional[str] = ..., experiments: _Optional[_Iterable[_Union[UsageExperiment, _Mapping]]] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

class PlanUsageExperimentRequest(_message.Message):
    __slots__ = ("runner", "role", "split_percent", "effect")
    RUNNER_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    SPLIT_PERCENT_FIELD_NUMBER: _ClassVar[int]
    EFFECT_FIELD_NUMBER: _ClassVar[int]
    runner: str
    role: str
    split_percent: int
    effect: float
    def __init__(self, runner: _Optional[str] = ..., role: _Optional[str] = ..., split_percent: _Optional[int] = ..., effect: _Optional[float] = ...) -> None: ...

class PlanUsageExperimentResponse(_message.Message):
    __slots__ = ("plan", "refusal")
    PLAN_FIELD_NUMBER: _ClassVar[int]
    REFUSAL_FIELD_NUMBER: _ClassVar[int]
    plan: UsageExperimentPlan
    refusal: str
    def __init__(self, plan: _Optional[_Union[UsageExperimentPlan, _Mapping]] = ..., refusal: _Optional[str] = ...) -> None: ...

class StartUsageExperimentRequest(_message.Message):
    __slots__ = ("runner", "role", "profile", "model", "until", "label", "split_percent", "sessions", "block_days")
    RUNNER_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    UNTIL_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    SPLIT_PERCENT_FIELD_NUMBER: _ClassVar[int]
    SESSIONS_FIELD_NUMBER: _ClassVar[int]
    BLOCK_DAYS_FIELD_NUMBER: _ClassVar[int]
    runner: str
    role: str
    profile: str
    model: str
    until: str
    label: str
    split_percent: int
    sessions: bool
    block_days: int
    def __init__(self, runner: _Optional[str] = ..., role: _Optional[str] = ..., profile: _Optional[str] = ..., model: _Optional[str] = ..., until: _Optional[str] = ..., label: _Optional[str] = ..., split_percent: _Optional[int] = ..., sessions: _Optional[bool] = ..., block_days: _Optional[int] = ...) -> None: ...

class StartUsageExperimentResponse(_message.Message):
    __slots__ = ("mode", "plan", "notes")
    MODE_FIELD_NUMBER: _ClassVar[int]
    PLAN_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    mode: UsageMode
    plan: UsageExperimentPlan
    notes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, mode: _Optional[_Union[UsageMode, _Mapping]] = ..., plan: _Optional[_Union[UsageExperimentPlan, _Mapping]] = ..., notes: _Optional[_Iterable[str]] = ...) -> None: ...

class ProposeRoleDefaultRequest(_message.Message):
    __slots__ = ("mode_id",)
    MODE_ID_FIELD_NUMBER: _ClassVar[int]
    mode_id: str
    def __init__(self, mode_id: _Optional[str] = ...) -> None: ...

class ProposeRoleDefaultResponse(_message.Message):
    __slots__ = ("recommendation",)
    RECOMMENDATION_FIELD_NUMBER: _ClassVar[int]
    recommendation: UsageRoleRecommendation
    def __init__(self, recommendation: _Optional[_Union[UsageRoleRecommendation, _Mapping]] = ...) -> None: ...

class UsageSignalsStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class UsageSignalsStatusResponse(_message.Message):
    __slots__ = ("enabled", "source", "changed_at", "changed_by", "stored", "note")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CHANGED_AT_FIELD_NUMBER: _ClassVar[int]
    CHANGED_BY_FIELD_NUMBER: _ClassVar[int]
    STORED_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    source: str
    changed_at: str
    changed_by: str
    stored: int
    note: str
    def __init__(self, enabled: _Optional[bool] = ..., source: _Optional[str] = ..., changed_at: _Optional[str] = ..., changed_by: _Optional[str] = ..., stored: _Optional[int] = ..., note: _Optional[str] = ...) -> None: ...

class SetUsageSignalsRequest(_message.Message):
    __slots__ = ("action",)
    ACTION_FIELD_NUMBER: _ClassVar[int]
    action: str
    def __init__(self, action: _Optional[str] = ...) -> None: ...

class SetUsageSignalsResponse(_message.Message):
    __slots__ = ("status", "purged")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PURGED_FIELD_NUMBER: _ClassVar[int]
    status: UsageSignalsStatusResponse
    purged: int
    def __init__(self, status: _Optional[_Union[UsageSignalsStatusResponse, _Mapping]] = ..., purged: _Optional[int] = ...) -> None: ...
