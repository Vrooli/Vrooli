from common.v1 import maturity_pb2 as _maturity_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ValidateScenarioRequest(_message.Message):
    __slots__ = ("scenario", "path", "workspaces", "include_execution", "use_cache")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    WORKSPACES_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_EXECUTION_FIELD_NUMBER: _ClassVar[int]
    USE_CACHE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    workspaces: _containers.RepeatedScalarFieldContainer[str]
    include_execution: bool
    use_cache: bool
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ..., workspaces: _Optional[_Iterable[str]] = ..., include_execution: _Optional[bool] = ..., use_cache: _Optional[bool] = ...) -> None: ...

class ValidateScenarioResponse(_message.Message):
    __slots__ = ("run_id", "status", "summary", "scenario", "target_kind", "target_path", "degraded_reason", "surfaces", "workspaces", "plan", "command_results", "coverage", "findings", "diagnostics", "maturity", "counts", "next_steps", "assessment")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TARGET_KIND_FIELD_NUMBER: _ClassVar[int]
    TARGET_PATH_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASON_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    WORKSPACES_FIELD_NUMBER: _ClassVar[int]
    PLAN_FIELD_NUMBER: _ClassVar[int]
    COMMAND_RESULTS_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTICS_FIELD_NUMBER: _ClassVar[int]
    MATURITY_FIELD_NUMBER: _ClassVar[int]
    COUNTS_FIELD_NUMBER: _ClassVar[int]
    NEXT_STEPS_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    status: str
    summary: str
    scenario: str
    target_kind: str
    target_path: str
    degraded_reason: str
    surfaces: _containers.RepeatedCompositeFieldContainer[TestSurface]
    workspaces: _containers.RepeatedCompositeFieldContainer[TestWorkspace]
    plan: ExecutionPlan
    command_results: _containers.RepeatedCompositeFieldContainer[CommandResult]
    coverage: _containers.RepeatedCompositeFieldContainer[CoverageTarget]
    findings: _containers.RepeatedCompositeFieldContainer[ValidationFinding]
    diagnostics: _containers.RepeatedCompositeFieldContainer[Diagnostic]
    maturity: MaturitySummary
    counts: ValidationCounts
    next_steps: _containers.RepeatedScalarFieldContainer[str]
    assessment: _maturity_pb2.MaturityAssessment
    def __init__(self, run_id: _Optional[str] = ..., status: _Optional[str] = ..., summary: _Optional[str] = ..., scenario: _Optional[str] = ..., target_kind: _Optional[str] = ..., target_path: _Optional[str] = ..., degraded_reason: _Optional[str] = ..., surfaces: _Optional[_Iterable[_Union[TestSurface, _Mapping]]] = ..., workspaces: _Optional[_Iterable[_Union[TestWorkspace, _Mapping]]] = ..., plan: _Optional[_Union[ExecutionPlan, _Mapping]] = ..., command_results: _Optional[_Iterable[_Union[CommandResult, _Mapping]]] = ..., coverage: _Optional[_Iterable[_Union[CoverageTarget, _Mapping]]] = ..., findings: _Optional[_Iterable[_Union[ValidationFinding, _Mapping]]] = ..., diagnostics: _Optional[_Iterable[_Union[Diagnostic, _Mapping]]] = ..., maturity: _Optional[_Union[MaturitySummary, _Mapping]] = ..., counts: _Optional[_Union[ValidationCounts, _Mapping]] = ..., next_steps: _Optional[_Iterable[str]] = ..., assessment: _Optional[_Union[_maturity_pb2.MaturityAssessment, _Mapping]] = ...) -> None: ...

class TestSurface(_message.Message):
    __slots__ = ("id", "kind", "language", "framework", "root_path", "package_manager", "status", "confidence")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    FRAMEWORK_FIELD_NUMBER: _ClassVar[int]
    ROOT_PATH_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_MANAGER_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    language: str
    framework: str
    root_path: str
    package_manager: str
    status: str
    confidence: float
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., language: _Optional[str] = ..., framework: _Optional[str] = ..., root_path: _Optional[str] = ..., package_manager: _Optional[str] = ..., status: _Optional[str] = ..., confidence: _Optional[float] = ...) -> None: ...

class TestWorkspace(_message.Message):
    __slots__ = ("id", "language", "root_path", "framework", "canonical_framework", "test_command", "coverage_command", "package_manager", "status", "degraded_reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    ROOT_PATH_FIELD_NUMBER: _ClassVar[int]
    FRAMEWORK_FIELD_NUMBER: _ClassVar[int]
    CANONICAL_FRAMEWORK_FIELD_NUMBER: _ClassVar[int]
    TEST_COMMAND_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_COMMAND_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_MANAGER_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    language: str
    root_path: str
    framework: str
    canonical_framework: str
    test_command: str
    coverage_command: str
    package_manager: str
    status: str
    degraded_reason: str
    def __init__(self, id: _Optional[str] = ..., language: _Optional[str] = ..., root_path: _Optional[str] = ..., framework: _Optional[str] = ..., canonical_framework: _Optional[str] = ..., test_command: _Optional[str] = ..., coverage_command: _Optional[str] = ..., package_manager: _Optional[str] = ..., status: _Optional[str] = ..., degraded_reason: _Optional[str] = ...) -> None: ...

class ExecutionPlan(_message.Message):
    __slots__ = ("commands", "notes")
    COMMANDS_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    commands: _containers.RepeatedCompositeFieldContainer[PlannedCommand]
    notes: str
    def __init__(self, commands: _Optional[_Iterable[_Union[PlannedCommand, _Mapping]]] = ..., notes: _Optional[str] = ...) -> None: ...

class PlannedCommand(_message.Message):
    __slots__ = ("workspace_id", "name", "command", "working_directory", "timeout_seconds")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    WORKING_DIRECTORY_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    name: str
    command: str
    working_directory: str
    timeout_seconds: int
    def __init__(self, workspace_id: _Optional[str] = ..., name: _Optional[str] = ..., command: _Optional[str] = ..., working_directory: _Optional[str] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class CommandResult(_message.Message):
    __slots__ = ("name", "command", "working_directory", "status", "exit_code", "stdout_excerpt", "stderr_excerpt", "timeout_seconds", "failure_reason", "failure_class", "duration_ms")
    NAME_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    WORKING_DIRECTORY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    STDOUT_EXCERPT_FIELD_NUMBER: _ClassVar[int]
    STDERR_EXCERPT_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    FAILURE_REASON_FIELD_NUMBER: _ClassVar[int]
    FAILURE_CLASS_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    name: str
    command: str
    working_directory: str
    status: str
    exit_code: int
    stdout_excerpt: str
    stderr_excerpt: str
    timeout_seconds: int
    failure_reason: str
    failure_class: str
    duration_ms: int
    def __init__(self, name: _Optional[str] = ..., command: _Optional[str] = ..., working_directory: _Optional[str] = ..., status: _Optional[str] = ..., exit_code: _Optional[int] = ..., stdout_excerpt: _Optional[str] = ..., stderr_excerpt: _Optional[str] = ..., timeout_seconds: _Optional[int] = ..., failure_reason: _Optional[str] = ..., failure_class: _Optional[str] = ..., duration_ms: _Optional[int] = ...) -> None: ...

class CoverageTarget(_message.Message):
    __slots__ = ("id", "language", "surface_id", "file_path", "covered_lines", "total_lines", "coverage_percent", "threshold", "status")
    ID_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    SURFACE_ID_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    COVERED_LINES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_LINES_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_PERCENT_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    id: str
    language: str
    surface_id: str
    file_path: str
    covered_lines: int
    total_lines: int
    coverage_percent: float
    threshold: float
    status: str
    def __init__(self, id: _Optional[str] = ..., language: _Optional[str] = ..., surface_id: _Optional[str] = ..., file_path: _Optional[str] = ..., covered_lines: _Optional[int] = ..., total_lines: _Optional[int] = ..., coverage_percent: _Optional[float] = ..., threshold: _Optional[float] = ..., status: _Optional[str] = ...) -> None: ...

class ValidationFinding(_message.Message):
    __slots__ = ("id", "scenario", "surface_id", "workspace_id", "language", "framework", "code", "category", "severity", "file_path", "symbol", "message", "evidence", "expected", "observed", "why_it_matters", "remediation", "source_command", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SURFACE_ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    FRAMEWORK_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    SYMBOL_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_FIELD_NUMBER: _ClassVar[int]
    WHY_IT_MATTERS_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_COMMAND_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario: str
    surface_id: str
    workspace_id: str
    language: str
    framework: str
    code: str
    category: str
    severity: str
    file_path: str
    symbol: str
    message: str
    evidence: str
    expected: str
    observed: str
    why_it_matters: str
    remediation: str
    source_command: str
    created_at: str
    def __init__(self, id: _Optional[str] = ..., scenario: _Optional[str] = ..., surface_id: _Optional[str] = ..., workspace_id: _Optional[str] = ..., language: _Optional[str] = ..., framework: _Optional[str] = ..., code: _Optional[str] = ..., category: _Optional[str] = ..., severity: _Optional[str] = ..., file_path: _Optional[str] = ..., symbol: _Optional[str] = ..., message: _Optional[str] = ..., evidence: _Optional[str] = ..., expected: _Optional[str] = ..., observed: _Optional[str] = ..., why_it_matters: _Optional[str] = ..., remediation: _Optional[str] = ..., source_command: _Optional[str] = ..., created_at: _Optional[str] = ...) -> None: ...

class Diagnostic(_message.Message):
    __slots__ = ("kind", "workspace_id", "message", "evidence", "severity")
    KIND_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    kind: str
    workspace_id: str
    message: str
    evidence: str
    severity: str
    def __init__(self, kind: _Optional[str] = ..., workspace_id: _Optional[str] = ..., message: _Optional[str] = ..., evidence: _Optional[str] = ..., severity: _Optional[str] = ...) -> None: ...

class MaturitySummary(_message.Message):
    __slots__ = ("rung", "label", "rationale")
    RUNG_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    rung: int
    label: str
    rationale: str
    def __init__(self, rung: _Optional[int] = ..., label: _Optional[str] = ..., rationale: _Optional[str] = ...) -> None: ...

class ValidationCounts(_message.Message):
    __slots__ = ("errors", "warnings", "infos", "surfaces", "workspaces", "coverage_targets")
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    INFOS_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    WORKSPACES_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_TARGETS_FIELD_NUMBER: _ClassVar[int]
    errors: int
    warnings: int
    infos: int
    surfaces: int
    workspaces: int
    coverage_targets: int
    def __init__(self, errors: _Optional[int] = ..., warnings: _Optional[int] = ..., infos: _Optional[int] = ..., surfaces: _Optional[int] = ..., workspaces: _Optional[int] = ..., coverage_targets: _Optional[int] = ...) -> None: ...
