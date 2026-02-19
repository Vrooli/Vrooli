from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class StopCondition(_message.Message):
    __slots__ = ("type", "operator", "conditions", "metric", "compare_operator", "value")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_FIELD_NUMBER: _ClassVar[int]
    CONDITIONS_FIELD_NUMBER: _ClassVar[int]
    METRIC_FIELD_NUMBER: _ClassVar[int]
    COMPARE_OPERATOR_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    type: str
    operator: str
    conditions: _containers.RepeatedCompositeFieldContainer[StopCondition]
    metric: str
    compare_operator: str
    value: float
    def __init__(self, type: _Optional[str] = ..., operator: _Optional[str] = ..., conditions: _Optional[_Iterable[_Union[StopCondition, _Mapping]]] = ..., metric: _Optional[str] = ..., compare_operator: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...

class SteerPhase(_message.Message):
    __slots__ = ("id", "skill_ids", "skill_name", "with_scope", "scope", "stop_conditions", "max_iterations", "description")
    ID_FIELD_NUMBER: _ClassVar[int]
    SKILL_IDS_FIELD_NUMBER: _ClassVar[int]
    SKILL_NAME_FIELD_NUMBER: _ClassVar[int]
    WITH_SCOPE_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    STOP_CONDITIONS_FIELD_NUMBER: _ClassVar[int]
    MAX_ITERATIONS_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    id: str
    skill_ids: _containers.RepeatedScalarFieldContainer[str]
    skill_name: str
    with_scope: bool
    scope: str
    stop_conditions: _containers.RepeatedCompositeFieldContainer[StopCondition]
    max_iterations: int
    description: str
    def __init__(self, id: _Optional[str] = ..., skill_ids: _Optional[_Iterable[str]] = ..., skill_name: _Optional[str] = ..., with_scope: _Optional[bool] = ..., scope: _Optional[str] = ..., stop_conditions: _Optional[_Iterable[_Union[StopCondition, _Mapping]]] = ..., max_iterations: _Optional[int] = ..., description: _Optional[str] = ...) -> None: ...

class QualityGate(_message.Message):
    __slots__ = ("name", "condition", "failure_action", "message")
    NAME_FIELD_NUMBER: _ClassVar[int]
    CONDITION_FIELD_NUMBER: _ClassVar[int]
    FAILURE_ACTION_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    name: str
    condition: StopCondition
    failure_action: str
    message: str
    def __init__(self, name: _Optional[str] = ..., condition: _Optional[_Union[StopCondition, _Mapping]] = ..., failure_action: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class AutoSteerProfile(_message.Message):
    __slots__ = ("id", "name", "description", "phases", "quality_gates", "created_at", "updated_at", "tags")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    QUALITY_GATES_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    phases: _containers.RepeatedCompositeFieldContainer[SteerPhase]
    quality_gates: _containers.RepeatedCompositeFieldContainer[QualityGate]
    created_at: str
    updated_at: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., phases: _Optional[_Iterable[_Union[SteerPhase, _Mapping]]] = ..., quality_gates: _Optional[_Iterable[_Union[QualityGate, _Mapping]]] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ...) -> None: ...

class UXMetrics(_message.Message):
    __slots__ = ("accessibility_score", "ui_test_coverage", "responsive_breakpoints", "user_flows_implemented", "loading_states_count", "error_handling_coverage")
    ACCESSIBILITY_SCORE_FIELD_NUMBER: _ClassVar[int]
    UI_TEST_COVERAGE_FIELD_NUMBER: _ClassVar[int]
    RESPONSIVE_BREAKPOINTS_FIELD_NUMBER: _ClassVar[int]
    USER_FLOWS_IMPLEMENTED_FIELD_NUMBER: _ClassVar[int]
    LOADING_STATES_COUNT_FIELD_NUMBER: _ClassVar[int]
    ERROR_HANDLING_COVERAGE_FIELD_NUMBER: _ClassVar[int]
    accessibility_score: float
    ui_test_coverage: float
    responsive_breakpoints: int
    user_flows_implemented: int
    loading_states_count: int
    error_handling_coverage: float
    def __init__(self, accessibility_score: _Optional[float] = ..., ui_test_coverage: _Optional[float] = ..., responsive_breakpoints: _Optional[int] = ..., user_flows_implemented: _Optional[int] = ..., loading_states_count: _Optional[int] = ..., error_handling_coverage: _Optional[float] = ...) -> None: ...

class RefactorMetrics(_message.Message):
    __slots__ = ("cyclomatic_complexity_avg", "duplication_percentage", "standards_violations", "tidiness_score", "tech_debt_items")
    CYCLOMATIC_COMPLEXITY_AVG_FIELD_NUMBER: _ClassVar[int]
    DUPLICATION_PERCENTAGE_FIELD_NUMBER: _ClassVar[int]
    STANDARDS_VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    TIDINESS_SCORE_FIELD_NUMBER: _ClassVar[int]
    TECH_DEBT_ITEMS_FIELD_NUMBER: _ClassVar[int]
    cyclomatic_complexity_avg: float
    duplication_percentage: float
    standards_violations: int
    tidiness_score: float
    tech_debt_items: int
    def __init__(self, cyclomatic_complexity_avg: _Optional[float] = ..., duplication_percentage: _Optional[float] = ..., standards_violations: _Optional[int] = ..., tidiness_score: _Optional[float] = ..., tech_debt_items: _Optional[int] = ...) -> None: ...

class TestMetrics(_message.Message):
    __slots__ = ("unit_test_coverage", "integration_test_coverage", "ui_test_coverage", "edge_cases_covered", "flaky_tests", "test_quality_score")
    UNIT_TEST_COVERAGE_FIELD_NUMBER: _ClassVar[int]
    INTEGRATION_TEST_COVERAGE_FIELD_NUMBER: _ClassVar[int]
    UI_TEST_COVERAGE_FIELD_NUMBER: _ClassVar[int]
    EDGE_CASES_COVERED_FIELD_NUMBER: _ClassVar[int]
    FLAKY_TESTS_FIELD_NUMBER: _ClassVar[int]
    TEST_QUALITY_SCORE_FIELD_NUMBER: _ClassVar[int]
    unit_test_coverage: float
    integration_test_coverage: float
    ui_test_coverage: float
    edge_cases_covered: int
    flaky_tests: int
    test_quality_score: float
    def __init__(self, unit_test_coverage: _Optional[float] = ..., integration_test_coverage: _Optional[float] = ..., ui_test_coverage: _Optional[float] = ..., edge_cases_covered: _Optional[int] = ..., flaky_tests: _Optional[int] = ..., test_quality_score: _Optional[float] = ...) -> None: ...

class PerformanceMetrics(_message.Message):
    __slots__ = ("bundle_size_kb", "initial_load_time_ms", "lcp_ms", "fid_ms", "cls_score")
    BUNDLE_SIZE_KB_FIELD_NUMBER: _ClassVar[int]
    INITIAL_LOAD_TIME_MS_FIELD_NUMBER: _ClassVar[int]
    LCP_MS_FIELD_NUMBER: _ClassVar[int]
    FID_MS_FIELD_NUMBER: _ClassVar[int]
    CLS_SCORE_FIELD_NUMBER: _ClassVar[int]
    bundle_size_kb: float
    initial_load_time_ms: int
    lcp_ms: int
    fid_ms: int
    cls_score: float
    def __init__(self, bundle_size_kb: _Optional[float] = ..., initial_load_time_ms: _Optional[int] = ..., lcp_ms: _Optional[int] = ..., fid_ms: _Optional[int] = ..., cls_score: _Optional[float] = ...) -> None: ...

class SecurityMetrics(_message.Message):
    __slots__ = ("vulnerability_count", "input_validation_coverage", "auth_implementation_score", "security_scan_score")
    VULNERABILITY_COUNT_FIELD_NUMBER: _ClassVar[int]
    INPUT_VALIDATION_COVERAGE_FIELD_NUMBER: _ClassVar[int]
    AUTH_IMPLEMENTATION_SCORE_FIELD_NUMBER: _ClassVar[int]
    SECURITY_SCAN_SCORE_FIELD_NUMBER: _ClassVar[int]
    vulnerability_count: int
    input_validation_coverage: float
    auth_implementation_score: float
    security_scan_score: float
    def __init__(self, vulnerability_count: _Optional[int] = ..., input_validation_coverage: _Optional[float] = ..., auth_implementation_score: _Optional[float] = ..., security_scan_score: _Optional[float] = ...) -> None: ...

class MetricsSnapshot(_message.Message):
    __slots__ = ("timestamp", "phase_loops", "total_loops", "build_status", "operational_targets_total", "operational_targets_passing", "operational_targets_percentage", "ux", "refactor", "test", "performance", "security")
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    PHASE_LOOPS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_LOOPS_FIELD_NUMBER: _ClassVar[int]
    BUILD_STATUS_FIELD_NUMBER: _ClassVar[int]
    OPERATIONAL_TARGETS_TOTAL_FIELD_NUMBER: _ClassVar[int]
    OPERATIONAL_TARGETS_PASSING_FIELD_NUMBER: _ClassVar[int]
    OPERATIONAL_TARGETS_PERCENTAGE_FIELD_NUMBER: _ClassVar[int]
    UX_FIELD_NUMBER: _ClassVar[int]
    REFACTOR_FIELD_NUMBER: _ClassVar[int]
    TEST_FIELD_NUMBER: _ClassVar[int]
    PERFORMANCE_FIELD_NUMBER: _ClassVar[int]
    SECURITY_FIELD_NUMBER: _ClassVar[int]
    timestamp: str
    phase_loops: int
    total_loops: int
    build_status: int
    operational_targets_total: int
    operational_targets_passing: int
    operational_targets_percentage: float
    ux: UXMetrics
    refactor: RefactorMetrics
    test: TestMetrics
    performance: PerformanceMetrics
    security: SecurityMetrics
    def __init__(self, timestamp: _Optional[str] = ..., phase_loops: _Optional[int] = ..., total_loops: _Optional[int] = ..., build_status: _Optional[int] = ..., operational_targets_total: _Optional[int] = ..., operational_targets_passing: _Optional[int] = ..., operational_targets_percentage: _Optional[float] = ..., ux: _Optional[_Union[UXMetrics, _Mapping]] = ..., refactor: _Optional[_Union[RefactorMetrics, _Mapping]] = ..., test: _Optional[_Union[TestMetrics, _Mapping]] = ..., performance: _Optional[_Union[PerformanceMetrics, _Mapping]] = ..., security: _Optional[_Union[SecurityMetrics, _Mapping]] = ...) -> None: ...

class PhaseExecution(_message.Message):
    __slots__ = ("phase_id", "skill_ids", "skill_name", "with_scope", "scope", "iterations", "start_metrics", "end_metrics", "commits", "started_at", "completed_at", "stop_reason")
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    SKILL_IDS_FIELD_NUMBER: _ClassVar[int]
    SKILL_NAME_FIELD_NUMBER: _ClassVar[int]
    WITH_SCOPE_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    ITERATIONS_FIELD_NUMBER: _ClassVar[int]
    START_METRICS_FIELD_NUMBER: _ClassVar[int]
    END_METRICS_FIELD_NUMBER: _ClassVar[int]
    COMMITS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    STOP_REASON_FIELD_NUMBER: _ClassVar[int]
    phase_id: str
    skill_ids: _containers.RepeatedScalarFieldContainer[str]
    skill_name: str
    with_scope: bool
    scope: str
    iterations: int
    start_metrics: MetricsSnapshot
    end_metrics: MetricsSnapshot
    commits: _containers.RepeatedScalarFieldContainer[str]
    started_at: str
    completed_at: str
    stop_reason: str
    def __init__(self, phase_id: _Optional[str] = ..., skill_ids: _Optional[_Iterable[str]] = ..., skill_name: _Optional[str] = ..., with_scope: _Optional[bool] = ..., scope: _Optional[str] = ..., iterations: _Optional[int] = ..., start_metrics: _Optional[_Union[MetricsSnapshot, _Mapping]] = ..., end_metrics: _Optional[_Union[MetricsSnapshot, _Mapping]] = ..., commits: _Optional[_Iterable[str]] = ..., started_at: _Optional[str] = ..., completed_at: _Optional[str] = ..., stop_reason: _Optional[str] = ...) -> None: ...

class ProfileExecutionState(_message.Message):
    __slots__ = ("task_id", "profile_id", "current_phase_index", "current_phase_iteration", "auto_steer_iteration", "phase_started_at", "phase_history", "metrics", "phase_start_metrics", "started_at", "last_updated")
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    CURRENT_PHASE_INDEX_FIELD_NUMBER: _ClassVar[int]
    CURRENT_PHASE_ITERATION_FIELD_NUMBER: _ClassVar[int]
    AUTO_STEER_ITERATION_FIELD_NUMBER: _ClassVar[int]
    PHASE_STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    PHASE_HISTORY_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    PHASE_START_METRICS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_UPDATED_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    profile_id: str
    current_phase_index: int
    current_phase_iteration: int
    auto_steer_iteration: int
    phase_started_at: str
    phase_history: _containers.RepeatedCompositeFieldContainer[PhaseExecution]
    metrics: MetricsSnapshot
    phase_start_metrics: MetricsSnapshot
    started_at: str
    last_updated: str
    def __init__(self, task_id: _Optional[str] = ..., profile_id: _Optional[str] = ..., current_phase_index: _Optional[int] = ..., current_phase_iteration: _Optional[int] = ..., auto_steer_iteration: _Optional[int] = ..., phase_started_at: _Optional[str] = ..., phase_history: _Optional[_Iterable[_Union[PhaseExecution, _Mapping]]] = ..., metrics: _Optional[_Union[MetricsSnapshot, _Mapping]] = ..., phase_start_metrics: _Optional[_Union[MetricsSnapshot, _Mapping]] = ..., started_at: _Optional[str] = ..., last_updated: _Optional[str] = ...) -> None: ...

class PhasePerformance(_message.Message):
    __slots__ = ("skill_ids", "skill_name", "iterations", "metric_deltas", "duration", "effectiveness")
    class MetricDeltasEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    SKILL_IDS_FIELD_NUMBER: _ClassVar[int]
    SKILL_NAME_FIELD_NUMBER: _ClassVar[int]
    ITERATIONS_FIELD_NUMBER: _ClassVar[int]
    METRIC_DELTAS_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVENESS_FIELD_NUMBER: _ClassVar[int]
    skill_ids: _containers.RepeatedScalarFieldContainer[str]
    skill_name: str
    iterations: int
    metric_deltas: _containers.ScalarMap[str, float]
    duration: int
    effectiveness: float
    def __init__(self, skill_ids: _Optional[_Iterable[str]] = ..., skill_name: _Optional[str] = ..., iterations: _Optional[int] = ..., metric_deltas: _Optional[_Mapping[str, float]] = ..., duration: _Optional[int] = ..., effectiveness: _Optional[float] = ...) -> None: ...

class UserFeedback(_message.Message):
    __slots__ = ("rating", "comments", "submitted_at")
    RATING_FIELD_NUMBER: _ClassVar[int]
    COMMENTS_FIELD_NUMBER: _ClassVar[int]
    SUBMITTED_AT_FIELD_NUMBER: _ClassVar[int]
    rating: int
    comments: str
    submitted_at: str
    def __init__(self, rating: _Optional[int] = ..., comments: _Optional[str] = ..., submitted_at: _Optional[str] = ...) -> None: ...

class ExecutionFeedbackEntry(_message.Message):
    __slots__ = ("id", "category", "severity", "suggested_action", "comments", "metadata", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    SUGGESTED_ACTION_FIELD_NUMBER: _ClassVar[int]
    COMMENTS_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    category: str
    severity: str
    suggested_action: str
    comments: str
    metadata: _struct_pb2.Struct
    created_at: str
    def __init__(self, id: _Optional[str] = ..., category: _Optional[str] = ..., severity: _Optional[str] = ..., suggested_action: _Optional[str] = ..., comments: _Optional[str] = ..., metadata: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., created_at: _Optional[str] = ...) -> None: ...

class ProfilePerformance(_message.Message):
    __slots__ = ("id", "profile_id", "scenario_name", "execution_id", "start_metrics", "end_metrics", "phase_breakdown", "total_iterations", "total_duration", "user_feedback", "feedback_entries", "executed_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    START_METRICS_FIELD_NUMBER: _ClassVar[int]
    END_METRICS_FIELD_NUMBER: _ClassVar[int]
    PHASE_BREAKDOWN_FIELD_NUMBER: _ClassVar[int]
    TOTAL_ITERATIONS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_DURATION_FIELD_NUMBER: _ClassVar[int]
    USER_FEEDBACK_FIELD_NUMBER: _ClassVar[int]
    FEEDBACK_ENTRIES_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    profile_id: str
    scenario_name: str
    execution_id: str
    start_metrics: MetricsSnapshot
    end_metrics: MetricsSnapshot
    phase_breakdown: _containers.RepeatedCompositeFieldContainer[PhasePerformance]
    total_iterations: int
    total_duration: int
    user_feedback: UserFeedback
    feedback_entries: _containers.RepeatedCompositeFieldContainer[ExecutionFeedbackEntry]
    executed_at: str
    def __init__(self, id: _Optional[str] = ..., profile_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., execution_id: _Optional[str] = ..., start_metrics: _Optional[_Union[MetricsSnapshot, _Mapping]] = ..., end_metrics: _Optional[_Union[MetricsSnapshot, _Mapping]] = ..., phase_breakdown: _Optional[_Iterable[_Union[PhasePerformance, _Mapping]]] = ..., total_iterations: _Optional[int] = ..., total_duration: _Optional[int] = ..., user_feedback: _Optional[_Union[UserFeedback, _Mapping]] = ..., feedback_entries: _Optional[_Iterable[_Union[ExecutionFeedbackEntry, _Mapping]]] = ..., executed_at: _Optional[str] = ...) -> None: ...

class IterationEvaluation(_message.Message):
    __slots__ = ("should_stop", "reason", "next_phase")
    SHOULD_STOP_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    NEXT_PHASE_FIELD_NUMBER: _ClassVar[int]
    should_stop: bool
    reason: str
    next_phase: int
    def __init__(self, should_stop: _Optional[bool] = ..., reason: _Optional[str] = ..., next_phase: _Optional[int] = ...) -> None: ...

class PhaseAdvanceResult(_message.Message):
    __slots__ = ("success", "next_phase_index", "completed", "message")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PHASE_INDEX_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    success: bool
    next_phase_index: int
    completed: bool
    message: str
    def __init__(self, success: _Optional[bool] = ..., next_phase_index: _Optional[int] = ..., completed: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...

class QualityGateResult(_message.Message):
    __slots__ = ("gate_name", "passed", "message", "action")
    GATE_NAME_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    gate_name: str
    passed: bool
    message: str
    action: str
    def __init__(self, gate_name: _Optional[str] = ..., passed: _Optional[bool] = ..., message: _Optional[str] = ..., action: _Optional[str] = ...) -> None: ...
