from common.v1 import maturity_pb2 as _maturity_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ValidateScenarioRequest(_message.Message):
    __slots__ = ("scenario", "path", "workspaces", "include_execution", "use_cache", "fast_test_only")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    WORKSPACES_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_EXECUTION_FIELD_NUMBER: _ClassVar[int]
    USE_CACHE_FIELD_NUMBER: _ClassVar[int]
    FAST_TEST_ONLY_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    workspaces: _containers.RepeatedScalarFieldContainer[str]
    include_execution: bool
    use_cache: bool
    fast_test_only: bool
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ..., workspaces: _Optional[_Iterable[str]] = ..., include_execution: _Optional[bool] = ..., use_cache: _Optional[bool] = ..., fast_test_only: _Optional[bool] = ...) -> None: ...

class ValidateScenarioResponse(_message.Message):
    __slots__ = ("run_id", "status", "summary", "scenario", "target_kind", "target_path", "degraded_reason", "surfaces", "workspaces", "plan", "command_results", "coverage", "findings", "diagnostics", "maturity", "counts", "next_steps", "assessment", "artifacts", "projection_checks", "suppressed_findings", "cache_hit", "cache_miss_reason", "cache_invalidated_dimensions", "cache_saved_wall_time_ms", "cache_saved_cpu_time_ms", "cache_retained_bytes")
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
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    PROJECTION_CHECKS_FIELD_NUMBER: _ClassVar[int]
    SUPPRESSED_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    CACHE_HIT_FIELD_NUMBER: _ClassVar[int]
    CACHE_MISS_REASON_FIELD_NUMBER: _ClassVar[int]
    CACHE_INVALIDATED_DIMENSIONS_FIELD_NUMBER: _ClassVar[int]
    CACHE_SAVED_WALL_TIME_MS_FIELD_NUMBER: _ClassVar[int]
    CACHE_SAVED_CPU_TIME_MS_FIELD_NUMBER: _ClassVar[int]
    CACHE_RETAINED_BYTES_FIELD_NUMBER: _ClassVar[int]
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
    artifacts: _containers.RepeatedCompositeFieldContainer[Artifact]
    projection_checks: _containers.RepeatedCompositeFieldContainer[ProjectionCheck]
    suppressed_findings: _containers.RepeatedCompositeFieldContainer[ValidationFinding]
    cache_hit: bool
    cache_miss_reason: str
    cache_invalidated_dimensions: _containers.RepeatedScalarFieldContainer[str]
    cache_saved_wall_time_ms: int
    cache_saved_cpu_time_ms: int
    cache_retained_bytes: int
    def __init__(self, run_id: _Optional[str] = ..., status: _Optional[str] = ..., summary: _Optional[str] = ..., scenario: _Optional[str] = ..., target_kind: _Optional[str] = ..., target_path: _Optional[str] = ..., degraded_reason: _Optional[str] = ..., surfaces: _Optional[_Iterable[_Union[TestSurface, _Mapping]]] = ..., workspaces: _Optional[_Iterable[_Union[TestWorkspace, _Mapping]]] = ..., plan: _Optional[_Union[ExecutionPlan, _Mapping]] = ..., command_results: _Optional[_Iterable[_Union[CommandResult, _Mapping]]] = ..., coverage: _Optional[_Iterable[_Union[CoverageTarget, _Mapping]]] = ..., findings: _Optional[_Iterable[_Union[ValidationFinding, _Mapping]]] = ..., diagnostics: _Optional[_Iterable[_Union[Diagnostic, _Mapping]]] = ..., maturity: _Optional[_Union[MaturitySummary, _Mapping]] = ..., counts: _Optional[_Union[ValidationCounts, _Mapping]] = ..., next_steps: _Optional[_Iterable[str]] = ..., assessment: _Optional[_Union[_maturity_pb2.MaturityAssessment, _Mapping]] = ..., artifacts: _Optional[_Iterable[_Union[Artifact, _Mapping]]] = ..., projection_checks: _Optional[_Iterable[_Union[ProjectionCheck, _Mapping]]] = ..., suppressed_findings: _Optional[_Iterable[_Union[ValidationFinding, _Mapping]]] = ..., cache_hit: _Optional[bool] = ..., cache_miss_reason: _Optional[str] = ..., cache_invalidated_dimensions: _Optional[_Iterable[str]] = ..., cache_saved_wall_time_ms: _Optional[int] = ..., cache_saved_cpu_time_ms: _Optional[int] = ..., cache_retained_bytes: _Optional[int] = ...) -> None: ...

class ProjectionCheck(_message.Message):
    __slots__ = ("id", "workspace_id", "surface_id", "key", "owner", "file_path", "policy_value", "native_value", "status", "remediation", "finding_code")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    SURFACE_ID_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    POLICY_VALUE_FIELD_NUMBER: _ClassVar[int]
    NATIVE_VALUE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    FINDING_CODE_FIELD_NUMBER: _ClassVar[int]
    id: str
    workspace_id: str
    surface_id: str
    key: str
    owner: str
    file_path: str
    policy_value: str
    native_value: str
    status: str
    remediation: str
    finding_code: str
    def __init__(self, id: _Optional[str] = ..., workspace_id: _Optional[str] = ..., surface_id: _Optional[str] = ..., key: _Optional[str] = ..., owner: _Optional[str] = ..., file_path: _Optional[str] = ..., policy_value: _Optional[str] = ..., native_value: _Optional[str] = ..., status: _Optional[str] = ..., remediation: _Optional[str] = ..., finding_code: _Optional[str] = ...) -> None: ...

class Artifact(_message.Message):
    __slots__ = ("label", "kind", "reference")
    LABEL_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    label: str
    kind: str
    reference: str
    def __init__(self, label: _Optional[str] = ..., kind: _Optional[str] = ..., reference: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("id", "language", "root_path", "framework", "canonical_framework", "test_command", "coverage_command", "package_manager", "status", "degraded_reason", "runner_profile", "resources", "adapter_id", "adapter_version", "test_kind")
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
    RUNNER_PROFILE_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_ID_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_VERSION_FIELD_NUMBER: _ClassVar[int]
    TEST_KIND_FIELD_NUMBER: _ClassVar[int]
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
    runner_profile: str
    resources: ResourceLimits
    adapter_id: str
    adapter_version: str
    test_kind: str
    def __init__(self, id: _Optional[str] = ..., language: _Optional[str] = ..., root_path: _Optional[str] = ..., framework: _Optional[str] = ..., canonical_framework: _Optional[str] = ..., test_command: _Optional[str] = ..., coverage_command: _Optional[str] = ..., package_manager: _Optional[str] = ..., status: _Optional[str] = ..., degraded_reason: _Optional[str] = ..., runner_profile: _Optional[str] = ..., resources: _Optional[_Union[ResourceLimits, _Mapping]] = ..., adapter_id: _Optional[str] = ..., adapter_version: _Optional[str] = ..., test_kind: _Optional[str] = ...) -> None: ...

class ExecutionPlan(_message.Message):
    __slots__ = ("commands", "notes")
    COMMANDS_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    commands: _containers.RepeatedCompositeFieldContainer[PlannedCommand]
    notes: str
    def __init__(self, commands: _Optional[_Iterable[_Union[PlannedCommand, _Mapping]]] = ..., notes: _Optional[str] = ...) -> None: ...

class PlannedCommand(_message.Message):
    __slots__ = ("workspace_id", "name", "command", "working_directory", "timeout_seconds", "executable", "args", "environment", "artifacts", "resources", "kind", "no_output_timeout_seconds", "test_kind", "hermetic")
    class EnvironmentEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    WORKING_DIRECTORY_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    EXECUTABLE_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    ENVIRONMENT_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    NO_OUTPUT_TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    TEST_KIND_FIELD_NUMBER: _ClassVar[int]
    HERMETIC_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    name: str
    command: str
    working_directory: str
    timeout_seconds: int
    executable: str
    args: _containers.RepeatedScalarFieldContainer[str]
    environment: _containers.ScalarMap[str, str]
    artifacts: _containers.RepeatedCompositeFieldContainer[CommandArtifact]
    resources: ResourceLimits
    kind: str
    no_output_timeout_seconds: int
    test_kind: str
    hermetic: HermeticPolicy
    def __init__(self, workspace_id: _Optional[str] = ..., name: _Optional[str] = ..., command: _Optional[str] = ..., working_directory: _Optional[str] = ..., timeout_seconds: _Optional[int] = ..., executable: _Optional[str] = ..., args: _Optional[_Iterable[str]] = ..., environment: _Optional[_Mapping[str, str]] = ..., artifacts: _Optional[_Iterable[_Union[CommandArtifact, _Mapping]]] = ..., resources: _Optional[_Union[ResourceLimits, _Mapping]] = ..., kind: _Optional[str] = ..., no_output_timeout_seconds: _Optional[int] = ..., test_kind: _Optional[str] = ..., hermetic: _Optional[_Union[HermeticPolicy, _Mapping]] = ...) -> None: ...

class CommandArtifact(_message.Message):
    __slots__ = ("label", "kind", "path")
    LABEL_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    label: str
    kind: str
    path: str
    def __init__(self, label: _Optional[str] = ..., kind: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class ResourceLimits(_message.Message):
    __slots__ = ("cpu_weight", "memory_bytes", "max_workers")
    CPU_WEIGHT_FIELD_NUMBER: _ClassVar[int]
    MEMORY_BYTES_FIELD_NUMBER: _ClassVar[int]
    MAX_WORKERS_FIELD_NUMBER: _ClassVar[int]
    cpu_weight: int
    memory_bytes: int
    max_workers: int
    def __init__(self, cpu_weight: _Optional[int] = ..., memory_bytes: _Optional[int] = ..., max_workers: _Optional[int] = ...) -> None: ...

class HermeticPolicy(_message.Message):
    __slots__ = ("network", "filesystem", "temporary_root", "restore_environment", "detect_child_leaks", "detect_open_handles", "order_independent")
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    FILESYSTEM_FIELD_NUMBER: _ClassVar[int]
    TEMPORARY_ROOT_FIELD_NUMBER: _ClassVar[int]
    RESTORE_ENVIRONMENT_FIELD_NUMBER: _ClassVar[int]
    DETECT_CHILD_LEAKS_FIELD_NUMBER: _ClassVar[int]
    DETECT_OPEN_HANDLES_FIELD_NUMBER: _ClassVar[int]
    ORDER_INDEPENDENT_FIELD_NUMBER: _ClassVar[int]
    network: str
    filesystem: str
    temporary_root: bool
    restore_environment: bool
    detect_child_leaks: bool
    detect_open_handles: bool
    order_independent: bool
    def __init__(self, network: _Optional[str] = ..., filesystem: _Optional[str] = ..., temporary_root: _Optional[bool] = ..., restore_environment: _Optional[bool] = ..., detect_child_leaks: _Optional[bool] = ..., detect_open_handles: _Optional[bool] = ..., order_independent: _Optional[bool] = ...) -> None: ...

class CommandResult(_message.Message):
    __slots__ = ("name", "command", "working_directory", "status", "exit_code", "stdout_excerpt", "stderr_excerpt", "timeout_seconds", "failure_reason", "failure_class", "duration_ms", "cpu_time_ms", "peak_rss_bytes")
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
    CPU_TIME_MS_FIELD_NUMBER: _ClassVar[int]
    PEAK_RSS_BYTES_FIELD_NUMBER: _ClassVar[int]
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
    cpu_time_ms: int
    peak_rss_bytes: int
    def __init__(self, name: _Optional[str] = ..., command: _Optional[str] = ..., working_directory: _Optional[str] = ..., status: _Optional[str] = ..., exit_code: _Optional[int] = ..., stdout_excerpt: _Optional[str] = ..., stderr_excerpt: _Optional[str] = ..., timeout_seconds: _Optional[int] = ..., failure_reason: _Optional[str] = ..., failure_class: _Optional[str] = ..., duration_ms: _Optional[int] = ..., cpu_time_ms: _Optional[int] = ..., peak_rss_bytes: _Optional[int] = ...) -> None: ...

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
    __slots__ = ("errors", "warnings", "infos", "surfaces", "workspaces", "coverage_targets", "suppressed_findings")
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    INFOS_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    WORKSPACES_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_TARGETS_FIELD_NUMBER: _ClassVar[int]
    SUPPRESSED_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    errors: int
    warnings: int
    infos: int
    surfaces: int
    workspaces: int
    coverage_targets: int
    suppressed_findings: int
    def __init__(self, errors: _Optional[int] = ..., warnings: _Optional[int] = ..., infos: _Optional[int] = ..., surfaces: _Optional[int] = ..., workspaces: _Optional[int] = ..., coverage_targets: _Optional[int] = ..., suppressed_findings: _Optional[int] = ...) -> None: ...
