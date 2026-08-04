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
    __slots__ = ("source_table", "window_start", "window_end", "row_count", "applied_filters", "executed_query")
    SOURCE_TABLE_FIELD_NUMBER: _ClassVar[int]
    WINDOW_START_FIELD_NUMBER: _ClassVar[int]
    WINDOW_END_FIELD_NUMBER: _ClassVar[int]
    ROW_COUNT_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FILTERS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_QUERY_FIELD_NUMBER: _ClassVar[int]
    source_table: str
    window_start: str
    window_end: str
    row_count: int
    applied_filters: _containers.RepeatedCompositeFieldContainer[MeasureFilter]
    executed_query: str
    def __init__(self, source_table: _Optional[str] = ..., window_start: _Optional[str] = ..., window_end: _Optional[str] = ..., row_count: _Optional[int] = ..., applied_filters: _Optional[_Iterable[_Union[MeasureFilter, _Mapping]]] = ..., executed_query: _Optional[str] = ...) -> None: ...

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

class RunCostResponse(_message.Message):
    __slots__ = ("total_cost_usd", "average_cost_usd", "total_tokens", "total_runs", "executed_query", "validity", "input_tokens", "output_tokens", "cache_read_tokens", "cache_creation_tokens", "input_cost_usd", "output_cost_usd", "cache_read_cost_usd", "cache_creation_cost_usd", "charge_by_basis", "total_charge_micro_usd", "unpriced_token_count", "provenance", "definition_id")
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
    provenance: MeasureProvenance
    definition_id: str
    def __init__(self, total_cost_usd: _Optional[float] = ..., average_cost_usd: _Optional[float] = ..., total_tokens: _Optional[int] = ..., total_runs: _Optional[int] = ..., executed_query: _Optional[str] = ..., validity: _Optional[_Union[MeasureValidity, _Mapping]] = ..., input_tokens: _Optional[int] = ..., output_tokens: _Optional[int] = ..., cache_read_tokens: _Optional[int] = ..., cache_creation_tokens: _Optional[int] = ..., input_cost_usd: _Optional[float] = ..., output_cost_usd: _Optional[float] = ..., cache_read_cost_usd: _Optional[float] = ..., cache_creation_cost_usd: _Optional[float] = ..., charge_by_basis: _Optional[_Iterable[_Union[ChargeByBasis, _Mapping]]] = ..., total_charge_micro_usd: _Optional[int] = ..., unpriced_token_count: _Optional[int] = ..., provenance: _Optional[_Union[MeasureProvenance, _Mapping]] = ..., definition_id: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("value", "run_count", "success_count", "failed_count", "total_cost_usd", "average_duration_ms", "total_tokens", "key", "total_charge_micro_usd", "consumption_per_successful_completion", "completion_rate")
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
    def __init__(self, value: _Optional[str] = ..., run_count: _Optional[int] = ..., success_count: _Optional[int] = ..., failed_count: _Optional[int] = ..., total_cost_usd: _Optional[float] = ..., average_duration_ms: _Optional[float] = ..., total_tokens: _Optional[int] = ..., key: _Optional[str] = ..., total_charge_micro_usd: _Optional[int] = ..., consumption_per_successful_completion: _Optional[float] = ..., completion_rate: _Optional[float] = ...) -> None: ...

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
    __slots__ = ("tool_name", "call_count", "success_count", "failed_count")
    TOOL_NAME_FIELD_NUMBER: _ClassVar[int]
    CALL_COUNT_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_COUNT_FIELD_NUMBER: _ClassVar[int]
    FAILED_COUNT_FIELD_NUMBER: _ClassVar[int]
    tool_name: str
    call_count: int
    success_count: int
    failed_count: int
    def __init__(self, tool_name: _Optional[str] = ..., call_count: _Optional[int] = ..., success_count: _Optional[int] = ..., failed_count: _Optional[int] = ...) -> None: ...

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
    __slots__ = ("executable", "command_path", "call_count", "success_count", "failed_count", "run_count", "truncated")
    EXECUTABLE_FIELD_NUMBER: _ClassVar[int]
    COMMAND_PATH_FIELD_NUMBER: _ClassVar[int]
    CALL_COUNT_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_COUNT_FIELD_NUMBER: _ClassVar[int]
    FAILED_COUNT_FIELD_NUMBER: _ClassVar[int]
    RUN_COUNT_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    executable: str
    command_path: str
    call_count: int
    success_count: int
    failed_count: int
    run_count: int
    truncated: bool
    def __init__(self, executable: _Optional[str] = ..., command_path: _Optional[str] = ..., call_count: _Optional[int] = ..., success_count: _Optional[int] = ..., failed_count: _Optional[int] = ..., run_count: _Optional[int] = ..., truncated: _Optional[bool] = ...) -> None: ...

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

class CapabilityUsageRequest(_message.Message):
    __slots__ = ("window", "filter")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ...) -> None: ...

class CapabilityUsageRow(_message.Message):
    __slots__ = ("target_scenario", "operation", "call_count", "success_count", "failed_count", "total_duration_ms")
    TARGET_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    CALL_COUNT_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_COUNT_FIELD_NUMBER: _ClassVar[int]
    FAILED_COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    target_scenario: str
    operation: str
    call_count: int
    success_count: int
    failed_count: int
    total_duration_ms: int
    def __init__(self, target_scenario: _Optional[str] = ..., operation: _Optional[str] = ..., call_count: _Optional[int] = ..., success_count: _Optional[int] = ..., failed_count: _Optional[int] = ..., total_duration_ms: _Optional[int] = ...) -> None: ...

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

class SelectCohortRequest(_message.Message):
    __slots__ = ("window", "filter", "limit")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    limit: int
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ..., limit: _Optional[int] = ...) -> None: ...

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
    __slots__ = ("window", "filter", "limit")
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FILTER_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    window: _measures_pb2.TimeWindow
    filter: InvocationFilter
    limit: int
    def __init__(self, window: _Optional[_Union[_measures_pb2.TimeWindow, _Mapping]] = ..., filter: _Optional[_Union[InvocationFilter, _Mapping]] = ..., limit: _Optional[int] = ...) -> None: ...

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
