from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GenerateScenarioRequest(_message.Message):
    __slots__ = ("template", "id", "display_name", "description", "destination", "design", "force", "dry_run", "run_hooks", "values")
    class ValuesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_FIELD_NUMBER: _ClassVar[int]
    DESIGN_FIELD_NUMBER: _ClassVar[int]
    FORCE_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    RUN_HOOKS_FIELD_NUMBER: _ClassVar[int]
    VALUES_FIELD_NUMBER: _ClassVar[int]
    template: str
    id: str
    display_name: str
    description: str
    destination: str
    design: str
    force: bool
    dry_run: bool
    run_hooks: bool
    values: _containers.ScalarMap[str, str]
    def __init__(self, template: _Optional[str] = ..., id: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., destination: _Optional[str] = ..., design: _Optional[str] = ..., force: _Optional[bool] = ..., dry_run: _Optional[bool] = ..., run_hooks: _Optional[bool] = ..., values: _Optional[_Mapping[str, str]] = ...) -> None: ...

class GenerateScenarioResponse(_message.Message):
    __slots__ = ("template", "display_name", "destination", "dry_run", "run_hooks", "manifest_version", "design_kit")
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    RUN_HOOKS_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_VERSION_FIELD_NUMBER: _ClassVar[int]
    DESIGN_KIT_FIELD_NUMBER: _ClassVar[int]
    template: str
    display_name: str
    destination: str
    dry_run: bool
    run_hooks: bool
    manifest_version: str
    design_kit: str
    def __init__(self, template: _Optional[str] = ..., display_name: _Optional[str] = ..., destination: _Optional[str] = ..., dry_run: _Optional[bool] = ..., run_hooks: _Optional[bool] = ..., manifest_version: _Optional[str] = ..., design_kit: _Optional[str] = ...) -> None: ...

class OrientScenarioRequest(_message.Message):
    __slots__ = ("scenario", "finalize")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    FINALIZE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    finalize: bool
    def __init__(self, scenario: _Optional[str] = ..., finalize: _Optional[bool] = ...) -> None: ...

class OrientScenarioResponse(_message.Message):
    __slots__ = ("scenario", "scenario_path", "finalized", "completed", "required", "finalize_required", "next_step", "message")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_PATH_FIELD_NUMBER: _ClassVar[int]
    FINALIZED_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    FINALIZE_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    NEXT_STEP_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    scenario_path: str
    finalized: bool
    completed: int
    required: int
    finalize_required: bool
    next_step: str
    message: str
    def __init__(self, scenario: _Optional[str] = ..., scenario_path: _Optional[str] = ..., finalized: _Optional[bool] = ..., completed: _Optional[int] = ..., required: _Optional[int] = ..., finalize_required: _Optional[bool] = ..., next_step: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class DetemplateScenarioRequest(_message.Message):
    __slots__ = ("scenario", "dry_run")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    dry_run: bool
    def __init__(self, scenario: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class DetemplateScenarioResponse(_message.Message):
    __slots__ = ("scenario", "scenario_path", "marker", "dry_run", "blocks_removed", "lines_stripped", "paths_deleted", "message")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_PATH_FIELD_NUMBER: _ClassVar[int]
    MARKER_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    BLOCKS_REMOVED_FIELD_NUMBER: _ClassVar[int]
    LINES_STRIPPED_FIELD_NUMBER: _ClassVar[int]
    PATHS_DELETED_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    scenario_path: str
    marker: str
    dry_run: bool
    blocks_removed: int
    lines_stripped: int
    paths_deleted: _containers.RepeatedScalarFieldContainer[str]
    message: str
    def __init__(self, scenario: _Optional[str] = ..., scenario_path: _Optional[str] = ..., marker: _Optional[str] = ..., dry_run: _Optional[bool] = ..., blocks_removed: _Optional[int] = ..., lines_stripped: _Optional[int] = ..., paths_deleted: _Optional[_Iterable[str]] = ..., message: _Optional[str] = ...) -> None: ...

class DestroyScenarioRequest(_message.Message):
    __slots__ = ("scenario", "dry_run", "proto_only", "force")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    PROTO_ONLY_FIELD_NUMBER: _ClassVar[int]
    FORCE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    dry_run: bool
    proto_only: bool
    force: bool
    def __init__(self, scenario: _Optional[str] = ..., dry_run: _Optional[bool] = ..., proto_only: _Optional[bool] = ..., force: _Optional[bool] = ...) -> None: ...

class DestroyScenarioResponse(_message.Message):
    __slots__ = ("scenario", "dry_run", "proto_only", "paths_removed", "paths_absent", "needs_proto_generate", "message")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    PROTO_ONLY_FIELD_NUMBER: _ClassVar[int]
    PATHS_REMOVED_FIELD_NUMBER: _ClassVar[int]
    PATHS_ABSENT_FIELD_NUMBER: _ClassVar[int]
    NEEDS_PROTO_GENERATE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    dry_run: bool
    proto_only: bool
    paths_removed: _containers.RepeatedScalarFieldContainer[str]
    paths_absent: _containers.RepeatedScalarFieldContainer[str]
    needs_proto_generate: bool
    message: str
    def __init__(self, scenario: _Optional[str] = ..., dry_run: _Optional[bool] = ..., proto_only: _Optional[bool] = ..., paths_removed: _Optional[_Iterable[str]] = ..., paths_absent: _Optional[_Iterable[str]] = ..., needs_proto_generate: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...

class ValidateTemplateRequest(_message.Message):
    __slots__ = ("template", "mode", "test_preset", "warning_policy", "retain_temp")
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    TEST_PRESET_FIELD_NUMBER: _ClassVar[int]
    WARNING_POLICY_FIELD_NUMBER: _ClassVar[int]
    RETAIN_TEMP_FIELD_NUMBER: _ClassVar[int]
    template: str
    mode: str
    test_preset: str
    warning_policy: str
    retain_temp: bool
    def __init__(self, template: _Optional[str] = ..., mode: _Optional[str] = ..., test_preset: _Optional[str] = ..., warning_policy: _Optional[str] = ..., retain_temp: _Optional[bool] = ...) -> None: ...

class TemplateValidationIssue(_message.Message):
    __slots__ = ("template", "path", "message")
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    template: str
    path: str
    message: str
    def __init__(self, template: _Optional[str] = ..., path: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class ValidateTemplateResponse(_message.Message):
    __slots__ = ("mode", "template", "count", "issues", "status", "issues_count", "warnings")
    MODE_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ISSUES_COUNT_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    mode: str
    template: str
    count: int
    issues: _containers.RepeatedCompositeFieldContainer[TemplateValidationIssue]
    status: str
    issues_count: int
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, mode: _Optional[str] = ..., template: _Optional[str] = ..., count: _Optional[int] = ..., issues: _Optional[_Iterable[_Union[TemplateValidationIssue, _Mapping]]] = ..., status: _Optional[str] = ..., issues_count: _Optional[int] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...

class DriftReportRequest(_message.Message):
    __slots__ = ("scenario", "all", "verbose")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ALL_FIELD_NUMBER: _ClassVar[int]
    VERBOSE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    all: bool
    verbose: bool
    def __init__(self, scenario: _Optional[str] = ..., all: _Optional[bool] = ..., verbose: _Optional[bool] = ...) -> None: ...

class DriftScenario(_message.Message):
    __slots__ = ("scenario", "template", "status", "manifest_drifted", "content_drifted", "message")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_DRIFTED_FIELD_NUMBER: _ClassVar[int]
    CONTENT_DRIFTED_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    template: str
    status: str
    manifest_drifted: bool
    content_drifted: bool
    message: str
    def __init__(self, scenario: _Optional[str] = ..., template: _Optional[str] = ..., status: _Optional[str] = ..., manifest_drifted: _Optional[bool] = ..., content_drifted: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...

class DriftReportResponse(_message.Message):
    __slots__ = ("scenarios",)
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    scenarios: _containers.RepeatedCompositeFieldContainer[DriftScenario]
    def __init__(self, scenarios: _Optional[_Iterable[_Union[DriftScenario, _Mapping]]] = ...) -> None: ...

class CleanupRunsRequest(_message.Message):
    __slots__ = ("dry_run", "older_than", "include_retained", "run_id")
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    OLDER_THAN_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_RETAINED_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    older_than: str
    include_retained: bool
    run_id: str
    def __init__(self, dry_run: _Optional[bool] = ..., older_than: _Optional[str] = ..., include_retained: _Optional[bool] = ..., run_id: _Optional[str] = ...) -> None: ...

class CleanupSkippedRun(_message.Message):
    __slots__ = ("run_id", "path", "reason")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    path: str
    reason: str
    def __init__(self, run_id: _Optional[str] = ..., path: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class CleanupRunsResponse(_message.Message):
    __slots__ = ("matched", "removed", "dry_run", "message", "skipped", "skipped_runs")
    MATCHED_FIELD_NUMBER: _ClassVar[int]
    REMOVED_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_RUNS_FIELD_NUMBER: _ClassVar[int]
    matched: int
    removed: int
    dry_run: bool
    message: str
    skipped: int
    skipped_runs: _containers.RepeatedCompositeFieldContainer[CleanupSkippedRun]
    def __init__(self, matched: _Optional[int] = ..., removed: _Optional[int] = ..., dry_run: _Optional[bool] = ..., message: _Optional[str] = ..., skipped: _Optional[int] = ..., skipped_runs: _Optional[_Iterable[_Union[CleanupSkippedRun, _Mapping]]] = ...) -> None: ...

class ListDesignKitsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class DesignKit(_message.Message):
    __slots__ = ("id", "name", "version", "default", "description", "tags", "adapters")
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
    adapters: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., version: _Optional[str] = ..., default: _Optional[bool] = ..., description: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., adapters: _Optional[_Iterable[str]] = ...) -> None: ...

class ListDesignKitsResponse(_message.Message):
    __slots__ = ("kits",)
    KITS_FIELD_NUMBER: _ClassVar[int]
    kits: _containers.RepeatedCompositeFieldContainer[DesignKit]
    def __init__(self, kits: _Optional[_Iterable[_Union[DesignKit, _Mapping]]] = ...) -> None: ...

class GetDesignKitRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetDesignKitResponse(_message.Message):
    __slots__ = ("kit",)
    KIT_FIELD_NUMBER: _ClassVar[int]
    kit: DesignKit
    def __init__(self, kit: _Optional[_Union[DesignKit, _Mapping]] = ...) -> None: ...

class ValidateDesignKitsRequest(_message.Message):
    __slots__ = ("id", "all")
    ID_FIELD_NUMBER: _ClassVar[int]
    ALL_FIELD_NUMBER: _ClassVar[int]
    id: str
    all: bool
    def __init__(self, id: _Optional[str] = ..., all: _Optional[bool] = ...) -> None: ...

class DesignValidationIssue(_message.Message):
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

class DesignKitValidationResult(_message.Message):
    __slots__ = ("kit", "status", "issues")
    KIT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    kit: str
    status: str
    issues: _containers.RepeatedCompositeFieldContainer[DesignValidationIssue]
    def __init__(self, kit: _Optional[str] = ..., status: _Optional[str] = ..., issues: _Optional[_Iterable[_Union[DesignValidationIssue, _Mapping]]] = ...) -> None: ...

class ValidateDesignKitsResponse(_message.Message):
    __slots__ = ("count", "issues", "status", "issues_count", "results")
    COUNT_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ISSUES_COUNT_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    count: int
    issues: _containers.RepeatedCompositeFieldContainer[DesignValidationIssue]
    status: str
    issues_count: int
    results: _containers.RepeatedCompositeFieldContainer[DesignKitValidationResult]
    def __init__(self, count: _Optional[int] = ..., issues: _Optional[_Iterable[_Union[DesignValidationIssue, _Mapping]]] = ..., status: _Optional[str] = ..., issues_count: _Optional[int] = ..., results: _Optional[_Iterable[_Union[DesignKitValidationResult, _Mapping]]] = ...) -> None: ...
