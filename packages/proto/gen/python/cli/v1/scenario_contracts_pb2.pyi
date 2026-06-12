from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ScenarioTemplateListResponse(_message.Message):
    __slots__ = ("success", "templates")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    TEMPLATES_FIELD_NUMBER: _ClassVar[int]
    success: bool
    templates: _containers.RepeatedCompositeFieldContainer[ScenarioTemplateInfo]
    def __init__(self, success: _Optional[bool] = ..., templates: _Optional[_Iterable[_Union[ScenarioTemplateInfo, _Mapping]]] = ...) -> None: ...

class ScenarioTemplateInfo(_message.Message):
    __slots__ = ("name", "path", "manifest", "missing")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_FIELD_NUMBER: _ClassVar[int]
    MISSING_FIELD_NUMBER: _ClassVar[int]
    name: str
    path: str
    manifest: ScenarioTemplateManifest
    missing: bool
    def __init__(self, name: _Optional[str] = ..., path: _Optional[str] = ..., manifest: _Optional[_Union[ScenarioTemplateManifest, _Mapping]] = ..., missing: _Optional[bool] = ...) -> None: ...

class ScenarioTemplateManifest(_message.Message):
    __slots__ = ("name", "version", "display_name", "description", "stack", "start_document", "design", "orientation", "required_vars", "optional_vars", "docs", "copy_excludes", "post_hooks", "relocations")
    class RequiredVarsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: ScenarioTemplateVar
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[ScenarioTemplateVar, _Mapping]] = ...) -> None: ...
    class OptionalVarsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: ScenarioTemplateVar
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[ScenarioTemplateVar, _Mapping]] = ...) -> None: ...
    class DocsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STACK_FIELD_NUMBER: _ClassVar[int]
    START_DOCUMENT_FIELD_NUMBER: _ClassVar[int]
    DESIGN_FIELD_NUMBER: _ClassVar[int]
    ORIENTATION_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_VARS_FIELD_NUMBER: _ClassVar[int]
    OPTIONAL_VARS_FIELD_NUMBER: _ClassVar[int]
    DOCS_FIELD_NUMBER: _ClassVar[int]
    COPY_EXCLUDES_FIELD_NUMBER: _ClassVar[int]
    POST_HOOKS_FIELD_NUMBER: _ClassVar[int]
    RELOCATIONS_FIELD_NUMBER: _ClassVar[int]
    name: str
    version: str
    display_name: str
    description: str
    stack: _containers.RepeatedScalarFieldContainer[str]
    start_document: str
    design: ScenarioTemplateDesign
    orientation: ScenarioTemplateOrientation
    required_vars: _containers.MessageMap[str, ScenarioTemplateVar]
    optional_vars: _containers.MessageMap[str, ScenarioTemplateVar]
    docs: _containers.ScalarMap[str, str]
    copy_excludes: _containers.RepeatedScalarFieldContainer[str]
    post_hooks: _containers.RepeatedCompositeFieldContainer[ScenarioTemplateHook]
    relocations: _containers.RepeatedCompositeFieldContainer[ScenarioTemplateRelocation]
    def __init__(self, name: _Optional[str] = ..., version: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., stack: _Optional[_Iterable[str]] = ..., start_document: _Optional[str] = ..., design: _Optional[_Union[ScenarioTemplateDesign, _Mapping]] = ..., orientation: _Optional[_Union[ScenarioTemplateOrientation, _Mapping]] = ..., required_vars: _Optional[_Mapping[str, ScenarioTemplateVar]] = ..., optional_vars: _Optional[_Mapping[str, ScenarioTemplateVar]] = ..., docs: _Optional[_Mapping[str, str]] = ..., copy_excludes: _Optional[_Iterable[str]] = ..., post_hooks: _Optional[_Iterable[_Union[ScenarioTemplateHook, _Mapping]]] = ..., relocations: _Optional[_Iterable[_Union[ScenarioTemplateRelocation, _Mapping]]] = ...) -> None: ...

class ScenarioTemplateVar(_message.Message):
    __slots__ = ("flag", "description", "default")
    FLAG_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_FIELD_NUMBER: _ClassVar[int]
    flag: str
    description: str
    default: str
    def __init__(self, flag: _Optional[str] = ..., description: _Optional[str] = ..., default: _Optional[str] = ...) -> None: ...

class ScenarioTemplateHook(_message.Message):
    __slots__ = ("description", "cmd", "cwd")
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CMD_FIELD_NUMBER: _ClassVar[int]
    CWD_FIELD_NUMBER: _ClassVar[int]
    description: str
    cmd: str
    cwd: str
    def __init__(self, description: _Optional[str] = ..., cmd: _Optional[str] = ..., cwd: _Optional[str] = ...) -> None: ...

class ScenarioTemplateDesign(_message.Message):
    __slots__ = ("adapter", "default", "required")
    ADAPTER_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    adapter: str
    default: str
    required: bool
    def __init__(self, adapter: _Optional[str] = ..., default: _Optional[str] = ..., required: _Optional[bool] = ...) -> None: ...

class ScenarioTemplateRelocation(_message.Message):
    __slots__ = ("description", "to", "post")
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    POST_FIELD_NUMBER: _ClassVar[int]
    description: str
    to: str
    post: _containers.RepeatedCompositeFieldContainer[ScenarioTemplateHook]
    def __init__(self, description: _Optional[str] = ..., to: _Optional[str] = ..., post: _Optional[_Iterable[_Union[ScenarioTemplateHook, _Mapping]]] = ..., **kwargs) -> None: ...

class ScenarioTemplateOrientation(_message.Message):
    __slots__ = ("version", "copy_to", "start_document", "finalize", "steps")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    COPY_TO_FIELD_NUMBER: _ClassVar[int]
    START_DOCUMENT_FIELD_NUMBER: _ClassVar[int]
    FINALIZE_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    version: str
    copy_to: str
    start_document: str
    finalize: ScenarioTemplateOrientationFinalize
    steps: _containers.RepeatedCompositeFieldContainer[ScenarioTemplateOrientationStep]
    def __init__(self, version: _Optional[str] = ..., copy_to: _Optional[str] = ..., start_document: _Optional[str] = ..., finalize: _Optional[_Union[ScenarioTemplateOrientationFinalize, _Mapping]] = ..., steps: _Optional[_Iterable[_Union[ScenarioTemplateOrientationStep, _Mapping]]] = ...) -> None: ...

class ScenarioTemplateOrientationFinalize(_message.Message):
    __slots__ = ("cleanup", "message")
    CLEANUP_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    cleanup: _containers.RepeatedScalarFieldContainer[str]
    message: str
    def __init__(self, cleanup: _Optional[_Iterable[str]] = ..., message: _Optional[str] = ...) -> None: ...

class ScenarioTemplateOrientationStep(_message.Message):
    __slots__ = ("id", "title", "description", "docs", "required", "checks")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    DOCS_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    description: str
    docs: _containers.RepeatedScalarFieldContainer[str]
    required: bool
    checks: _containers.RepeatedCompositeFieldContainer[ScenarioTemplateOrientationCheck]
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., docs: _Optional[_Iterable[str]] = ..., required: _Optional[bool] = ..., checks: _Optional[_Iterable[_Union[ScenarioTemplateOrientationCheck, _Mapping]]] = ...) -> None: ...

class ScenarioTemplateOrientationCheck(_message.Message):
    __slots__ = ("kind", "path", "pattern", "query", "text", "run", "timeout", "optional")
    KIND_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    PATTERN_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    RUN_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    OPTIONAL_FIELD_NUMBER: _ClassVar[int]
    kind: str
    path: str
    pattern: str
    query: str
    text: str
    run: str
    timeout: str
    optional: bool
    def __init__(self, kind: _Optional[str] = ..., path: _Optional[str] = ..., pattern: _Optional[str] = ..., query: _Optional[str] = ..., text: _Optional[str] = ..., run: _Optional[str] = ..., timeout: _Optional[str] = ..., optional: _Optional[bool] = ...) -> None: ...

class ScenarioTemplateDriftResponse(_message.Message):
    __slots__ = ("success", "drift")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DRIFT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    drift: ScenarioTemplateDriftReport
    def __init__(self, success: _Optional[bool] = ..., drift: _Optional[_Union[ScenarioTemplateDriftReport, _Mapping]] = ...) -> None: ...

class ScenarioTemplateDriftReport(_message.Message):
    __slots__ = ("scenarios",)
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    scenarios: _containers.RepeatedCompositeFieldContainer[ScenarioTemplateDriftScenario]
    def __init__(self, scenarios: _Optional[_Iterable[_Union[ScenarioTemplateDriftScenario, _Mapping]]] = ...) -> None: ...

class ScenarioTemplateDriftScenario(_message.Message):
    __slots__ = ("scenario", "template_id", "recorded_version", "current_version", "recorded_manifest_sha", "current_manifest_sha", "recorded_content_sha", "current_content_sha", "manifest_drifted", "content_drifted", "status", "message", "file_diffs")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_ID_FIELD_NUMBER: _ClassVar[int]
    RECORDED_VERSION_FIELD_NUMBER: _ClassVar[int]
    CURRENT_VERSION_FIELD_NUMBER: _ClassVar[int]
    RECORDED_MANIFEST_SHA_FIELD_NUMBER: _ClassVar[int]
    CURRENT_MANIFEST_SHA_FIELD_NUMBER: _ClassVar[int]
    RECORDED_CONTENT_SHA_FIELD_NUMBER: _ClassVar[int]
    CURRENT_CONTENT_SHA_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_DRIFTED_FIELD_NUMBER: _ClassVar[int]
    CONTENT_DRIFTED_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    FILE_DIFFS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    template_id: str
    recorded_version: str
    current_version: str
    recorded_manifest_sha: str
    current_manifest_sha: str
    recorded_content_sha: str
    current_content_sha: str
    manifest_drifted: bool
    content_drifted: bool
    status: str
    message: str
    file_diffs: _containers.RepeatedCompositeFieldContainer[ScenarioTemplateDriftFileDiff]
    def __init__(self, scenario: _Optional[str] = ..., template_id: _Optional[str] = ..., recorded_version: _Optional[str] = ..., current_version: _Optional[str] = ..., recorded_manifest_sha: _Optional[str] = ..., current_manifest_sha: _Optional[str] = ..., recorded_content_sha: _Optional[str] = ..., current_content_sha: _Optional[str] = ..., manifest_drifted: _Optional[bool] = ..., content_drifted: _Optional[bool] = ..., status: _Optional[str] = ..., message: _Optional[str] = ..., file_diffs: _Optional[_Iterable[_Union[ScenarioTemplateDriftFileDiff, _Mapping]]] = ...) -> None: ...

class ScenarioTemplateDriftFileDiff(_message.Message):
    __slots__ = ("path", "reason")
    PATH_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    path: str
    reason: str
    def __init__(self, path: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class ScenarioTemplateValidateResponse(_message.Message):
    __slots__ = ("success", "report")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    report: ScenarioTemplateValidationReport
    def __init__(self, success: _Optional[bool] = ..., report: _Optional[_Union[ScenarioTemplateValidationReport, _Mapping]] = ...) -> None: ...

class ScenarioTemplateValidationReport(_message.Message):
    __slots__ = ("mode", "template_name", "test_preset", "warning_policy", "warning_summary", "count", "deep_runs", "issues")
    MODE_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_NAME_FIELD_NUMBER: _ClassVar[int]
    TEST_PRESET_FIELD_NUMBER: _ClassVar[int]
    WARNING_POLICY_FIELD_NUMBER: _ClassVar[int]
    WARNING_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    DEEP_RUNS_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    mode: str
    template_name: str
    test_preset: str
    warning_policy: str
    warning_summary: ScenarioTemplateValidationWarningSummary
    count: int
    deep_runs: _containers.RepeatedCompositeFieldContainer[ScenarioTemplateValidationDeepRun]
    issues: _containers.RepeatedCompositeFieldContainer[ScenarioTemplateValidationIssue]
    def __init__(self, mode: _Optional[str] = ..., template_name: _Optional[str] = ..., test_preset: _Optional[str] = ..., warning_policy: _Optional[str] = ..., warning_summary: _Optional[_Union[ScenarioTemplateValidationWarningSummary, _Mapping]] = ..., count: _Optional[int] = ..., deep_runs: _Optional[_Iterable[_Union[ScenarioTemplateValidationDeepRun, _Mapping]]] = ..., issues: _Optional[_Iterable[_Union[ScenarioTemplateValidationIssue, _Mapping]]] = ...) -> None: ...

class ScenarioTemplateValidationIssue(_message.Message):
    __slots__ = ("template", "path", "message")
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    template: str
    path: str
    message: str
    def __init__(self, template: _Optional[str] = ..., path: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class ScenarioTemplateValidationWarning(_message.Message):
    __slots__ = ("message", "source", "log_path", "artifact_path")
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    LOG_PATH_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_PATH_FIELD_NUMBER: _ClassVar[int]
    message: str
    source: str
    log_path: str
    artifact_path: str
    def __init__(self, message: _Optional[str] = ..., source: _Optional[str] = ..., log_path: _Optional[str] = ..., artifact_path: _Optional[str] = ...) -> None: ...

class ScenarioTemplateValidationPhaseWarningSummary(_message.Message):
    __slots__ = ("name", "count", "warnings")
    NAME_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    name: str
    count: int
    warnings: _containers.RepeatedCompositeFieldContainer[ScenarioTemplateValidationWarning]
    def __init__(self, name: _Optional[str] = ..., count: _Optional[int] = ..., warnings: _Optional[_Iterable[_Union[ScenarioTemplateValidationWarning, _Mapping]]] = ...) -> None: ...

class ScenarioTemplateValidationWarningSummary(_message.Message):
    __slots__ = ("total", "phases")
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    total: int
    phases: _containers.RepeatedCompositeFieldContainer[ScenarioTemplateValidationPhaseWarningSummary]
    def __init__(self, total: _Optional[int] = ..., phases: _Optional[_Iterable[_Union[ScenarioTemplateValidationPhaseWarningSummary, _Mapping]]] = ...) -> None: ...

class ScenarioTemplateValidationDeepRun(_message.Message):
    __slots__ = ("template", "run_id", "scenario_id", "scenario_path", "temp_root", "test_preset", "warning_summary", "retained_temp", "cleanup_status", "relocation_artifacts", "cleanup_command")
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_PATH_FIELD_NUMBER: _ClassVar[int]
    TEMP_ROOT_FIELD_NUMBER: _ClassVar[int]
    TEST_PRESET_FIELD_NUMBER: _ClassVar[int]
    WARNING_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    RETAINED_TEMP_FIELD_NUMBER: _ClassVar[int]
    CLEANUP_STATUS_FIELD_NUMBER: _ClassVar[int]
    RELOCATION_ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    CLEANUP_COMMAND_FIELD_NUMBER: _ClassVar[int]
    template: str
    run_id: str
    scenario_id: str
    scenario_path: str
    temp_root: str
    test_preset: str
    warning_summary: ScenarioTemplateValidationWarningSummary
    retained_temp: bool
    cleanup_status: str
    relocation_artifacts: _containers.RepeatedScalarFieldContainer[str]
    cleanup_command: str
    def __init__(self, template: _Optional[str] = ..., run_id: _Optional[str] = ..., scenario_id: _Optional[str] = ..., scenario_path: _Optional[str] = ..., temp_root: _Optional[str] = ..., test_preset: _Optional[str] = ..., warning_summary: _Optional[_Union[ScenarioTemplateValidationWarningSummary, _Mapping]] = ..., retained_temp: _Optional[bool] = ..., cleanup_status: _Optional[str] = ..., relocation_artifacts: _Optional[_Iterable[str]] = ..., cleanup_command: _Optional[str] = ...) -> None: ...

class ScenarioTemplateCleanupResponse(_message.Message):
    __slots__ = ("success", "cleanup")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    CLEANUP_FIELD_NUMBER: _ClassVar[int]
    success: bool
    cleanup: ScenarioTemplateCleanupResult
    def __init__(self, success: _Optional[bool] = ..., cleanup: _Optional[_Union[ScenarioTemplateCleanupResult, _Mapping]] = ...) -> None: ...

class ScenarioTemplateCleanupResult(_message.Message):
    __slots__ = ("dry_run", "older_than", "include_retained", "run_id", "eligible", "skipped", "failures", "removed", "needs_proto_generate", "proto_generate_ran", "message")
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    OLDER_THAN_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_RETAINED_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    FAILURES_FIELD_NUMBER: _ClassVar[int]
    REMOVED_FIELD_NUMBER: _ClassVar[int]
    NEEDS_PROTO_GENERATE_FIELD_NUMBER: _ClassVar[int]
    PROTO_GENERATE_RAN_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    older_than: int
    include_retained: bool
    run_id: str
    eligible: _containers.RepeatedCompositeFieldContainer[ScenarioTemplateCleanupRun]
    skipped: _containers.RepeatedCompositeFieldContainer[ScenarioTemplateCleanupSkippedRun]
    failures: _containers.RepeatedCompositeFieldContainer[ScenarioTemplateCleanupFailedRun]
    removed: _containers.RepeatedCompositeFieldContainer[ScenarioTemplateCleanupRun]
    needs_proto_generate: bool
    proto_generate_ran: bool
    message: str
    def __init__(self, dry_run: _Optional[bool] = ..., older_than: _Optional[int] = ..., include_retained: _Optional[bool] = ..., run_id: _Optional[str] = ..., eligible: _Optional[_Iterable[_Union[ScenarioTemplateCleanupRun, _Mapping]]] = ..., skipped: _Optional[_Iterable[_Union[ScenarioTemplateCleanupSkippedRun, _Mapping]]] = ..., failures: _Optional[_Iterable[_Union[ScenarioTemplateCleanupFailedRun, _Mapping]]] = ..., removed: _Optional[_Iterable[_Union[ScenarioTemplateCleanupRun, _Mapping]]] = ..., needs_proto_generate: _Optional[bool] = ..., proto_generate_ran: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...

class ScenarioTemplateCleanupRun(_message.Message):
    __slots__ = ("marker_path", "marker", "age")
    MARKER_PATH_FIELD_NUMBER: _ClassVar[int]
    MARKER_FIELD_NUMBER: _ClassVar[int]
    AGE_FIELD_NUMBER: _ClassVar[int]
    marker_path: str
    marker: ScenarioTemplateCleanupRunMarker
    age: str
    def __init__(self, marker_path: _Optional[str] = ..., marker: _Optional[_Union[ScenarioTemplateCleanupRunMarker, _Mapping]] = ..., age: _Optional[str] = ...) -> None: ...

class ScenarioTemplateCleanupRunMarker(_message.Message):
    __slots__ = ("version", "run_id", "repo_root", "template", "scenario_id", "scenario_path", "temp_root", "created_at", "retained", "creator_pid", "completed", "cleanup_status", "relocation_artifacts")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    REPO_ROOT_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_PATH_FIELD_NUMBER: _ClassVar[int]
    TEMP_ROOT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    RETAINED_FIELD_NUMBER: _ClassVar[int]
    CREATOR_PID_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_FIELD_NUMBER: _ClassVar[int]
    CLEANUP_STATUS_FIELD_NUMBER: _ClassVar[int]
    RELOCATION_ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    version: str
    run_id: str
    repo_root: str
    template: str
    scenario_id: str
    scenario_path: str
    temp_root: str
    created_at: str
    retained: bool
    creator_pid: int
    completed: bool
    cleanup_status: str
    relocation_artifacts: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, version: _Optional[str] = ..., run_id: _Optional[str] = ..., repo_root: _Optional[str] = ..., template: _Optional[str] = ..., scenario_id: _Optional[str] = ..., scenario_path: _Optional[str] = ..., temp_root: _Optional[str] = ..., created_at: _Optional[str] = ..., retained: _Optional[bool] = ..., creator_pid: _Optional[int] = ..., completed: _Optional[bool] = ..., cleanup_status: _Optional[str] = ..., relocation_artifacts: _Optional[_Iterable[str]] = ...) -> None: ...

class ScenarioTemplateCleanupSkippedRun(_message.Message):
    __slots__ = ("run", "path", "reason")
    RUN_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    run: ScenarioTemplateCleanupRun
    path: str
    reason: str
    def __init__(self, run: _Optional[_Union[ScenarioTemplateCleanupRun, _Mapping]] = ..., path: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class ScenarioTemplateCleanupFailedRun(_message.Message):
    __slots__ = ("run", "path", "error")
    RUN_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    run: ScenarioTemplateCleanupRun
    path: str
    error: str
    def __init__(self, run: _Optional[_Union[ScenarioTemplateCleanupRun, _Mapping]] = ..., path: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...

class ScenarioDesignListResponse(_message.Message):
    __slots__ = ("success", "design_kits")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DESIGN_KITS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    design_kits: _containers.RepeatedCompositeFieldContainer[ScenarioDesignKitInfo]
    def __init__(self, success: _Optional[bool] = ..., design_kits: _Optional[_Iterable[_Union[ScenarioDesignKitInfo, _Mapping]]] = ...) -> None: ...

class ScenarioDesignShowResponse(_message.Message):
    __slots__ = ("success", "design_kit")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DESIGN_KIT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    design_kit: ScenarioDesignKitInfo
    def __init__(self, success: _Optional[bool] = ..., design_kit: _Optional[_Union[ScenarioDesignKitInfo, _Mapping]] = ...) -> None: ...

class ScenarioDesignKitInfo(_message.Message):
    __slots__ = ("id", "path", "manifest", "missing")
    ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_FIELD_NUMBER: _ClassVar[int]
    MISSING_FIELD_NUMBER: _ClassVar[int]
    id: str
    path: str
    manifest: ScenarioDesignKitManifest
    missing: bool
    def __init__(self, id: _Optional[str] = ..., path: _Optional[str] = ..., manifest: _Optional[_Union[ScenarioDesignKitManifest, _Mapping]] = ..., missing: _Optional[bool] = ...) -> None: ...

class ScenarioDesignKitManifest(_message.Message):
    __slots__ = ("id", "name", "version", "default", "description", "tags", "adapters")
    class AdaptersEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: ScenarioDesignKitAdapter
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[ScenarioDesignKitAdapter, _Mapping]] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    ADAPTERS_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    version: str
    default: bool
    description: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    adapters: _containers.MessageMap[str, ScenarioDesignKitAdapter]
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., version: _Optional[str] = ..., default: _Optional[bool] = ..., description: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., adapters: _Optional[_Mapping[str, ScenarioDesignKitAdapter]] = ...) -> None: ...

class ScenarioDesignKitAdapter(_message.Message):
    __slots__ = ("path", "supports")
    PATH_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_FIELD_NUMBER: _ClassVar[int]
    path: str
    supports: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, path: _Optional[str] = ..., supports: _Optional[_Iterable[str]] = ...) -> None: ...

class ScenarioDesignValidateResponse(_message.Message):
    __slots__ = ("success", "design_validation")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DESIGN_VALIDATION_FIELD_NUMBER: _ClassVar[int]
    success: bool
    design_validation: ScenarioDesignValidationReport
    def __init__(self, success: _Optional[bool] = ..., design_validation: _Optional[_Union[ScenarioDesignValidationReport, _Mapping]] = ...) -> None: ...

class ScenarioDesignValidationReport(_message.Message):
    __slots__ = ("count", "issues")
    COUNT_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    count: int
    issues: _containers.RepeatedCompositeFieldContainer[ScenarioDesignValidationIssue]
    def __init__(self, count: _Optional[int] = ..., issues: _Optional[_Iterable[_Union[ScenarioDesignValidationIssue, _Mapping]]] = ...) -> None: ...

class ScenarioDesignValidationIssue(_message.Message):
    __slots__ = ("kit", "adapter", "path", "message")
    KIT_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    kit: str
    adapter: str
    path: str
    message: str
    def __init__(self, kit: _Optional[str] = ..., adapter: _Optional[str] = ..., path: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class ScenarioOrientationResponse(_message.Message):
    __slots__ = ("success", "orientation")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ORIENTATION_FIELD_NUMBER: _ClassVar[int]
    success: bool
    orientation: ScenarioOrientationReport
    def __init__(self, success: _Optional[bool] = ..., orientation: _Optional[_Union[ScenarioOrientationReport, _Mapping]] = ...) -> None: ...

class ScenarioOrientationReport(_message.Message):
    __slots__ = ("scenario", "scenario_path", "orientation_path", "finalized", "template", "design", "start_document", "completed", "required", "steps", "next_step", "message", "finalize_required")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_PATH_FIELD_NUMBER: _ClassVar[int]
    ORIENTATION_PATH_FIELD_NUMBER: _ClassVar[int]
    FINALIZED_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    DESIGN_FIELD_NUMBER: _ClassVar[int]
    START_DOCUMENT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    NEXT_STEP_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    FINALIZE_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    scenario_path: str
    orientation_path: str
    finalized: bool
    template: ScenarioGenerationTemplateRef
    design: ScenarioGenerationDesignRef
    start_document: str
    completed: int
    required: int
    steps: _containers.RepeatedCompositeFieldContainer[ScenarioOrientationStep]
    next_step: ScenarioOrientationStep
    message: str
    finalize_required: bool
    def __init__(self, scenario: _Optional[str] = ..., scenario_path: _Optional[str] = ..., orientation_path: _Optional[str] = ..., finalized: _Optional[bool] = ..., template: _Optional[_Union[ScenarioGenerationTemplateRef, _Mapping]] = ..., design: _Optional[_Union[ScenarioGenerationDesignRef, _Mapping]] = ..., start_document: _Optional[str] = ..., completed: _Optional[int] = ..., required: _Optional[int] = ..., steps: _Optional[_Iterable[_Union[ScenarioOrientationStep, _Mapping]]] = ..., next_step: _Optional[_Union[ScenarioOrientationStep, _Mapping]] = ..., message: _Optional[str] = ..., finalize_required: _Optional[bool] = ...) -> None: ...

class ScenarioGenerationTemplateRef(_message.Message):
    __slots__ = ("id", "version")
    ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    id: str
    version: str
    def __init__(self, id: _Optional[str] = ..., version: _Optional[str] = ...) -> None: ...

class ScenarioGenerationDesignRef(_message.Message):
    __slots__ = ("id", "version", "adapter")
    ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_FIELD_NUMBER: _ClassVar[int]
    id: str
    version: str
    adapter: str
    def __init__(self, id: _Optional[str] = ..., version: _Optional[str] = ..., adapter: _Optional[str] = ...) -> None: ...

class ScenarioOrientationStep(_message.Message):
    __slots__ = ("id", "title", "description", "docs", "required", "complete", "checks")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    DOCS_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    description: str
    docs: _containers.RepeatedScalarFieldContainer[str]
    required: bool
    complete: bool
    checks: _containers.RepeatedCompositeFieldContainer[ScenarioOrientationCheck]
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., docs: _Optional[_Iterable[str]] = ..., required: _Optional[bool] = ..., complete: _Optional[bool] = ..., checks: _Optional[_Iterable[_Union[ScenarioOrientationCheck, _Mapping]]] = ...) -> None: ...

class ScenarioOrientationCheck(_message.Message):
    __slots__ = ("kind", "label", "passed", "skipped", "message", "optional")
    KIND_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    OPTIONAL_FIELD_NUMBER: _ClassVar[int]
    kind: str
    label: str
    passed: bool
    skipped: bool
    message: str
    optional: bool
    def __init__(self, kind: _Optional[str] = ..., label: _Optional[str] = ..., passed: _Optional[bool] = ..., skipped: _Optional[bool] = ..., message: _Optional[str] = ..., optional: _Optional[bool] = ...) -> None: ...
