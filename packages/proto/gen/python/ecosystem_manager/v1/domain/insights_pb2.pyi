from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AnalysisWindow(_message.Message):
    __slots__ = ("start_time", "end_time", "limit", "status_filter")
    START_TIME_FIELD_NUMBER: _ClassVar[int]
    END_TIME_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FILTER_FIELD_NUMBER: _ClassVar[int]
    start_time: str
    end_time: str
    limit: int
    status_filter: str
    def __init__(self, start_time: _Optional[str] = ..., end_time: _Optional[str] = ..., limit: _Optional[int] = ..., status_filter: _Optional[str] = ...) -> None: ...

class Pattern(_message.Message):
    __slots__ = ("id", "type", "frequency", "severity", "description", "examples", "evidence")
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    FREQUENCY_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    EXAMPLES_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    type: str
    frequency: int
    severity: str
    description: str
    examples: _containers.RepeatedScalarFieldContainer[str]
    evidence: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., type: _Optional[str] = ..., frequency: _Optional[int] = ..., severity: _Optional[str] = ..., description: _Optional[str] = ..., examples: _Optional[_Iterable[str]] = ..., evidence: _Optional[_Iterable[str]] = ...) -> None: ...

class ProposedChange(_message.Message):
    __slots__ = ("file", "type", "description", "before", "after", "content", "config_path", "config_value")
    FILE_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    BEFORE_FIELD_NUMBER: _ClassVar[int]
    AFTER_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    CONFIG_PATH_FIELD_NUMBER: _ClassVar[int]
    CONFIG_VALUE_FIELD_NUMBER: _ClassVar[int]
    file: str
    type: str
    description: str
    before: str
    after: str
    content: str
    config_path: str
    config_value: _struct_pb2.Value
    def __init__(self, file: _Optional[str] = ..., type: _Optional[str] = ..., description: _Optional[str] = ..., before: _Optional[str] = ..., after: _Optional[str] = ..., content: _Optional[str] = ..., config_path: _Optional[str] = ..., config_value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...

class ImpactEstimate(_message.Message):
    __slots__ = ("success_rate_improvement", "time_reduction", "confidence", "rationale")
    SUCCESS_RATE_IMPROVEMENT_FIELD_NUMBER: _ClassVar[int]
    TIME_REDUCTION_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    success_rate_improvement: str
    time_reduction: str
    confidence: str
    rationale: str
    def __init__(self, success_rate_improvement: _Optional[str] = ..., time_reduction: _Optional[str] = ..., confidence: _Optional[str] = ..., rationale: _Optional[str] = ...) -> None: ...

class Suggestion(_message.Message):
    __slots__ = ("id", "pattern_id", "type", "priority", "title", "description", "changes", "impact", "status", "applied_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    PATTERN_ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CHANGES_FIELD_NUMBER: _ClassVar[int]
    IMPACT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    APPLIED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    pattern_id: str
    type: str
    priority: str
    title: str
    description: str
    changes: _containers.RepeatedCompositeFieldContainer[ProposedChange]
    impact: ImpactEstimate
    status: str
    applied_at: str
    def __init__(self, id: _Optional[str] = ..., pattern_id: _Optional[str] = ..., type: _Optional[str] = ..., priority: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., changes: _Optional[_Iterable[_Union[ProposedChange, _Mapping]]] = ..., impact: _Optional[_Union[ImpactEstimate, _Mapping]] = ..., status: _Optional[str] = ..., applied_at: _Optional[str] = ...) -> None: ...

class ExecutionStatistics(_message.Message):
    __slots__ = ("total_executions", "success_count", "failure_count", "timeout_count", "rate_limit_count", "success_rate", "avg_duration", "median_duration", "most_common_exit_reason")
    TOTAL_EXECUTIONS_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_COUNT_FIELD_NUMBER: _ClassVar[int]
    FAILURE_COUNT_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_COUNT_FIELD_NUMBER: _ClassVar[int]
    RATE_LIMIT_COUNT_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_RATE_FIELD_NUMBER: _ClassVar[int]
    AVG_DURATION_FIELD_NUMBER: _ClassVar[int]
    MEDIAN_DURATION_FIELD_NUMBER: _ClassVar[int]
    MOST_COMMON_EXIT_REASON_FIELD_NUMBER: _ClassVar[int]
    total_executions: int
    success_count: int
    failure_count: int
    timeout_count: int
    rate_limit_count: int
    success_rate: float
    avg_duration: str
    median_duration: str
    most_common_exit_reason: str
    def __init__(self, total_executions: _Optional[int] = ..., success_count: _Optional[int] = ..., failure_count: _Optional[int] = ..., timeout_count: _Optional[int] = ..., rate_limit_count: _Optional[int] = ..., success_rate: _Optional[float] = ..., avg_duration: _Optional[str] = ..., median_duration: _Optional[str] = ..., most_common_exit_reason: _Optional[str] = ...) -> None: ...

class InsightReport(_message.Message):
    __slots__ = ("id", "task_id", "generated_at", "analysis_window", "execution_count", "patterns", "suggestions", "statistics", "generated_by")
    ID_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    ANALYSIS_WINDOW_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_COUNT_FIELD_NUMBER: _ClassVar[int]
    PATTERNS_FIELD_NUMBER: _ClassVar[int]
    SUGGESTIONS_FIELD_NUMBER: _ClassVar[int]
    STATISTICS_FIELD_NUMBER: _ClassVar[int]
    GENERATED_BY_FIELD_NUMBER: _ClassVar[int]
    id: str
    task_id: str
    generated_at: str
    analysis_window: AnalysisWindow
    execution_count: int
    patterns: _containers.RepeatedCompositeFieldContainer[Pattern]
    suggestions: _containers.RepeatedCompositeFieldContainer[Suggestion]
    statistics: ExecutionStatistics
    generated_by: str
    def __init__(self, id: _Optional[str] = ..., task_id: _Optional[str] = ..., generated_at: _Optional[str] = ..., analysis_window: _Optional[_Union[AnalysisWindow, _Mapping]] = ..., execution_count: _Optional[int] = ..., patterns: _Optional[_Iterable[_Union[Pattern, _Mapping]]] = ..., suggestions: _Optional[_Iterable[_Union[Suggestion, _Mapping]]] = ..., statistics: _Optional[_Union[ExecutionStatistics, _Mapping]] = ..., generated_by: _Optional[str] = ...) -> None: ...

class TaskTypeStats(_message.Message):
    __slots__ = ("count", "success_rate", "avg_duration", "top_pattern")
    COUNT_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_RATE_FIELD_NUMBER: _ClassVar[int]
    AVG_DURATION_FIELD_NUMBER: _ClassVar[int]
    TOP_PATTERN_FIELD_NUMBER: _ClassVar[int]
    count: int
    success_rate: float
    avg_duration: str
    top_pattern: str
    def __init__(self, count: _Optional[int] = ..., success_rate: _Optional[float] = ..., avg_duration: _Optional[str] = ..., top_pattern: _Optional[str] = ...) -> None: ...

class CrossTaskPattern(_message.Message):
    __slots__ = ("pattern", "affected_tasks", "task_types")
    PATTERN_FIELD_NUMBER: _ClassVar[int]
    AFFECTED_TASKS_FIELD_NUMBER: _ClassVar[int]
    TASK_TYPES_FIELD_NUMBER: _ClassVar[int]
    pattern: Pattern
    affected_tasks: _containers.RepeatedScalarFieldContainer[str]
    task_types: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, pattern: _Optional[_Union[Pattern, _Mapping]] = ..., affected_tasks: _Optional[_Iterable[str]] = ..., task_types: _Optional[_Iterable[str]] = ...) -> None: ...

class SystemInsightReport(_message.Message):
    __slots__ = ("id", "generated_at", "time_window", "task_count", "total_executions", "cross_task_patterns", "system_suggestions", "by_task_type", "by_operation")
    class ByTaskTypeEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: TaskTypeStats
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[TaskTypeStats, _Mapping]] = ...) -> None: ...
    class ByOperationEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: TaskTypeStats
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[TaskTypeStats, _Mapping]] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    TIME_WINDOW_FIELD_NUMBER: _ClassVar[int]
    TASK_COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_EXECUTIONS_FIELD_NUMBER: _ClassVar[int]
    CROSS_TASK_PATTERNS_FIELD_NUMBER: _ClassVar[int]
    SYSTEM_SUGGESTIONS_FIELD_NUMBER: _ClassVar[int]
    BY_TASK_TYPE_FIELD_NUMBER: _ClassVar[int]
    BY_OPERATION_FIELD_NUMBER: _ClassVar[int]
    id: str
    generated_at: str
    time_window: AnalysisWindow
    task_count: int
    total_executions: int
    cross_task_patterns: _containers.RepeatedCompositeFieldContainer[CrossTaskPattern]
    system_suggestions: _containers.RepeatedCompositeFieldContainer[Suggestion]
    by_task_type: _containers.MessageMap[str, TaskTypeStats]
    by_operation: _containers.MessageMap[str, TaskTypeStats]
    def __init__(self, id: _Optional[str] = ..., generated_at: _Optional[str] = ..., time_window: _Optional[_Union[AnalysisWindow, _Mapping]] = ..., task_count: _Optional[int] = ..., total_executions: _Optional[int] = ..., cross_task_patterns: _Optional[_Iterable[_Union[CrossTaskPattern, _Mapping]]] = ..., system_suggestions: _Optional[_Iterable[_Union[Suggestion, _Mapping]]] = ..., by_task_type: _Optional[_Mapping[str, TaskTypeStats]] = ..., by_operation: _Optional[_Mapping[str, TaskTypeStats]] = ...) -> None: ...
