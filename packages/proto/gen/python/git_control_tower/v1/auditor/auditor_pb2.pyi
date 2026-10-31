from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class StartCheckRequest(_message.Message):
    __slots__ = ("repository_id", "scenario_name", "check_type")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    CHECK_TYPE_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    scenario_name: str
    check_type: str
    def __init__(self, repository_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., check_type: _Optional[str] = ...) -> None: ...

class GetJobStatusRequest(_message.Message):
    __slots__ = ("repository_id", "job_id")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    job_id: str
    def __init__(self, repository_id: _Optional[str] = ..., job_id: _Optional[str] = ...) -> None: ...

class ListRulesRequest(_message.Message):
    __slots__ = ("repository_id",)
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    def __init__(self, repository_id: _Optional[str] = ...) -> None: ...

class ListViolationsRequest(_message.Message):
    __slots__ = ("repository_id", "scenario_name")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    scenario_name: str
    def __init__(self, repository_id: _Optional[str] = ..., scenario_name: _Optional[str] = ...) -> None: ...

class StartCheckResponse(_message.Message):
    __slots__ = ("job_id", "status")
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    status: JobStatus
    def __init__(self, job_id: _Optional[str] = ..., status: _Optional[_Union[JobStatus, _Mapping]] = ...) -> None: ...

class JobStatus(_message.Message):
    __slots__ = ("id", "scenario", "scan_type", "status", "started_at", "completed_at", "elapsed_seconds", "total_scenarios", "processed_scenarios", "processed_files", "total_files", "current_scenario", "current_file", "message", "error", "result")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SCAN_TYPE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    ELAPSED_SECONDS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    PROCESSED_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    PROCESSED_FILES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FILES_FIELD_NUMBER: _ClassVar[int]
    CURRENT_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    CURRENT_FILE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario: str
    scan_type: str
    status: str
    started_at: str
    completed_at: str
    elapsed_seconds: float
    total_scenarios: int
    processed_scenarios: int
    processed_files: int
    total_files: int
    current_scenario: str
    current_file: str
    message: str
    error: str
    result: CheckResult
    def __init__(self, id: _Optional[str] = ..., scenario: _Optional[str] = ..., scan_type: _Optional[str] = ..., status: _Optional[str] = ..., started_at: _Optional[str] = ..., completed_at: _Optional[str] = ..., elapsed_seconds: _Optional[float] = ..., total_scenarios: _Optional[int] = ..., processed_scenarios: _Optional[int] = ..., processed_files: _Optional[int] = ..., total_files: _Optional[int] = ..., current_scenario: _Optional[str] = ..., current_file: _Optional[str] = ..., message: _Optional[str] = ..., error: _Optional[str] = ..., result: _Optional[_Union[CheckResult, _Mapping]] = ...) -> None: ...

class CheckResult(_message.Message):
    __slots__ = ("check_id", "status", "scan_type", "started_at", "completed_at", "duration_seconds", "files_scanned", "violations", "statistics", "message", "scenario_name", "summary")
    CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SCAN_TYPE_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    FILES_SCANNED_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    STATISTICS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    check_id: str
    status: str
    scan_type: str
    started_at: str
    completed_at: str
    duration_seconds: float
    files_scanned: int
    violations: _containers.RepeatedCompositeFieldContainer[Violation]
    statistics: _containers.RepeatedCompositeFieldContainer[IntCount]
    message: str
    scenario_name: str
    summary: ViolationSummary
    def __init__(self, check_id: _Optional[str] = ..., status: _Optional[str] = ..., scan_type: _Optional[str] = ..., started_at: _Optional[str] = ..., completed_at: _Optional[str] = ..., duration_seconds: _Optional[float] = ..., files_scanned: _Optional[int] = ..., violations: _Optional[_Iterable[_Union[Violation, _Mapping]]] = ..., statistics: _Optional[_Iterable[_Union[IntCount, _Mapping]]] = ..., message: _Optional[str] = ..., scenario_name: _Optional[str] = ..., summary: _Optional[_Union[ViolationSummary, _Mapping]] = ...) -> None: ...

class Violation(_message.Message):
    __slots__ = ("id", "scenario_name", "type", "severity", "title", "description", "file_path", "line_number", "code_snippet", "recommendation", "standard", "discovered_at", "source", "metadata")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    LINE_NUMBER_FIELD_NUMBER: _ClassVar[int]
    CODE_SNIPPET_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATION_FIELD_NUMBER: _ClassVar[int]
    STANDARD_FIELD_NUMBER: _ClassVar[int]
    DISCOVERED_AT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario_name: str
    type: str
    severity: str
    title: str
    description: str
    file_path: str
    line_number: int
    code_snippet: str
    recommendation: str
    standard: str
    discovered_at: str
    source: str
    metadata: _containers.RepeatedCompositeFieldContainer[KeyValue]
    def __init__(self, id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., type: _Optional[str] = ..., severity: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., file_path: _Optional[str] = ..., line_number: _Optional[int] = ..., code_snippet: _Optional[str] = ..., recommendation: _Optional[str] = ..., standard: _Optional[str] = ..., discovered_at: _Optional[str] = ..., source: _Optional[str] = ..., metadata: _Optional[_Iterable[_Union[KeyValue, _Mapping]]] = ...) -> None: ...

class ViolationSummary(_message.Message):
    __slots__ = ("total", "by_severity", "by_rule", "highest_severity", "top_violations", "recommended_steps", "generated_at")
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    BY_SEVERITY_FIELD_NUMBER: _ClassVar[int]
    BY_RULE_FIELD_NUMBER: _ClassVar[int]
    HIGHEST_SEVERITY_FIELD_NUMBER: _ClassVar[int]
    TOP_VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_STEPS_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    total: int
    by_severity: _containers.RepeatedCompositeFieldContainer[IntCount]
    by_rule: _containers.RepeatedCompositeFieldContainer[RuleCount]
    highest_severity: str
    top_violations: _containers.RepeatedCompositeFieldContainer[ViolationExcerpt]
    recommended_steps: _containers.RepeatedScalarFieldContainer[str]
    generated_at: str
    def __init__(self, total: _Optional[int] = ..., by_severity: _Optional[_Iterable[_Union[IntCount, _Mapping]]] = ..., by_rule: _Optional[_Iterable[_Union[RuleCount, _Mapping]]] = ..., highest_severity: _Optional[str] = ..., top_violations: _Optional[_Iterable[_Union[ViolationExcerpt, _Mapping]]] = ..., recommended_steps: _Optional[_Iterable[str]] = ..., generated_at: _Optional[str] = ...) -> None: ...

class IntCount(_message.Message):
    __slots__ = ("key", "count")
    KEY_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    key: str
    count: int
    def __init__(self, key: _Optional[str] = ..., count: _Optional[int] = ...) -> None: ...

class RuleCount(_message.Message):
    __slots__ = ("rule_id", "count")
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    rule_id: str
    count: int
    def __init__(self, rule_id: _Optional[str] = ..., count: _Optional[int] = ...) -> None: ...

class ViolationExcerpt(_message.Message):
    __slots__ = ("id", "severity", "rule_id", "title", "file_path")
    ID_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    id: str
    severity: str
    rule_id: str
    title: str
    file_path: str
    def __init__(self, id: _Optional[str] = ..., severity: _Optional[str] = ..., rule_id: _Optional[str] = ..., title: _Optional[str] = ..., file_path: _Optional[str] = ...) -> None: ...

class KeyValue(_message.Message):
    __slots__ = ("key", "value")
    KEY_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    key: str
    value: str
    def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...

class Rule(_message.Message):
    __slots__ = ("id", "name", "description", "category", "severity", "enabled", "standard", "targets")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    STANDARD_FIELD_NUMBER: _ClassVar[int]
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    category: str
    severity: str
    enabled: bool
    standard: str
    targets: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., category: _Optional[str] = ..., severity: _Optional[str] = ..., enabled: _Optional[bool] = ..., standard: _Optional[str] = ..., targets: _Optional[_Iterable[str]] = ...) -> None: ...

class RulesResponse(_message.Message):
    __slots__ = ("rules", "categories", "count", "total")
    RULES_FIELD_NUMBER: _ClassVar[int]
    CATEGORIES_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    rules: _containers.RepeatedCompositeFieldContainer[Rule]
    categories: _containers.RepeatedCompositeFieldContainer[KeyValue]
    count: int
    total: int
    def __init__(self, rules: _Optional[_Iterable[_Union[Rule, _Mapping]]] = ..., categories: _Optional[_Iterable[_Union[KeyValue, _Mapping]]] = ..., count: _Optional[int] = ..., total: _Optional[int] = ...) -> None: ...

class ViolationsResponse(_message.Message):
    __slots__ = ("violations",)
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    violations: _containers.RepeatedCompositeFieldContainer[Violation]
    def __init__(self, violations: _Optional[_Iterable[_Union[Violation, _Mapping]]] = ...) -> None: ...

class FixRequest(_message.Message):
    __slots__ = ("repository_id", "scenario_names", "rule_ids")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAMES_FIELD_NUMBER: _ClassVar[int]
    RULE_IDS_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    scenario_names: _containers.RepeatedScalarFieldContainer[str]
    rule_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, repository_id: _Optional[str] = ..., scenario_names: _Optional[_Iterable[str]] = ..., rule_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class ApplyFixRequest(_message.Message):
    __slots__ = ("repository_id", "scenario_names", "rule_ids", "intent_id", "expected_revision", "subject_digest")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAMES_FIELD_NUMBER: _ClassVar[int]
    RULE_IDS_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    scenario_names: _containers.RepeatedScalarFieldContainer[str]
    rule_ids: _containers.RepeatedScalarFieldContainer[str]
    intent_id: str
    expected_revision: str
    subject_digest: str
    def __init__(self, repository_id: _Optional[str] = ..., scenario_names: _Optional[_Iterable[str]] = ..., rule_ids: _Optional[_Iterable[str]] = ..., intent_id: _Optional[str] = ..., expected_revision: _Optional[str] = ..., subject_digest: _Optional[str] = ...) -> None: ...

class FixChange(_message.Message):
    __slots__ = ("type", "detail")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    type: str
    detail: str
    def __init__(self, type: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class FixResult(_message.Message):
    __slots__ = ("scenario_name", "rule_id", "fixed", "file_path", "changes", "error")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    FIXED_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    CHANGES_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    rule_id: str
    fixed: bool
    file_path: str
    changes: _containers.RepeatedCompositeFieldContainer[FixChange]
    error: str
    def __init__(self, scenario_name: _Optional[str] = ..., rule_id: _Optional[str] = ..., fixed: _Optional[bool] = ..., file_path: _Optional[str] = ..., changes: _Optional[_Iterable[_Union[FixChange, _Mapping]]] = ..., error: _Optional[str] = ...) -> None: ...

class FixResponse(_message.Message):
    __slots__ = ("results", "count", "unfixable_rules", "errors")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    UNFIXABLE_RULES_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[FixResult]
    count: int
    unfixable_rules: _containers.RepeatedScalarFieldContainer[str]
    errors: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, results: _Optional[_Iterable[_Union[FixResult, _Mapping]]] = ..., count: _Optional[int] = ..., unfixable_rules: _Optional[_Iterable[str]] = ..., errors: _Optional[_Iterable[str]] = ...) -> None: ...
