from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PackageListResponse(_message.Message):
    __slots__ = ("success", "packages")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    PACKAGES_FIELD_NUMBER: _ClassVar[int]
    success: bool
    packages: _containers.RepeatedCompositeFieldContainer[PackageInfo]
    def __init__(self, success: _Optional[bool] = ..., packages: _Optional[_Iterable[_Union[PackageInfo, _Mapping]]] = ...) -> None: ...

class PackageInfoResponse(_message.Message):
    __slots__ = ("success", "package")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_FIELD_NUMBER: _ClassVar[int]
    success: bool
    package: PackageInfo
    def __init__(self, success: _Optional[bool] = ..., package: _Optional[_Union[PackageInfo, _Mapping]] = ...) -> None: ...

class PackageInfo(_message.Message):
    __slots__ = ("name", "root_path", "manifest_path", "manifest")
    NAME_FIELD_NUMBER: _ClassVar[int]
    ROOT_PATH_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_PATH_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_FIELD_NUMBER: _ClassVar[int]
    name: str
    root_path: str
    manifest_path: str
    manifest: PackageManifest
    def __init__(self, name: _Optional[str] = ..., root_path: _Optional[str] = ..., manifest_path: _Optional[str] = ..., manifest: _Optional[_Union[PackageManifest, _Mapping]] = ...) -> None: ...

class PackageManifest(_message.Message):
    __slots__ = ("schema", "version", "package")
    SCHEMA_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_FIELD_NUMBER: _ClassVar[int]
    schema: str
    version: str
    package: PackageManifestEntry
    def __init__(self, schema: _Optional[str] = ..., version: _Optional[str] = ..., package: _Optional[_Union[PackageManifestEntry, _Mapping]] = ...) -> None: ...

class PackageManifestEntry(_message.Message):
    __slots__ = ("name", "display_name", "description", "kind", "language", "module_identifiers", "generated_outputs", "adoption", "lifecycle", "refresh", "docs")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    MODULE_IDENTIFIERS_FIELD_NUMBER: _ClassVar[int]
    GENERATED_OUTPUTS_FIELD_NUMBER: _ClassVar[int]
    ADOPTION_FIELD_NUMBER: _ClassVar[int]
    LIFECYCLE_FIELD_NUMBER: _ClassVar[int]
    REFRESH_FIELD_NUMBER: _ClassVar[int]
    DOCS_FIELD_NUMBER: _ClassVar[int]
    name: str
    display_name: str
    description: str
    kind: str
    language: str
    module_identifiers: _containers.RepeatedScalarFieldContainer[str]
    generated_outputs: _containers.RepeatedCompositeFieldContainer[PackageGeneratedOutput]
    adoption: PackageAdoptionPolicy
    lifecycle: PackageLifecyclePolicy
    refresh: PackageRefreshPolicy
    docs: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., kind: _Optional[str] = ..., language: _Optional[str] = ..., module_identifiers: _Optional[_Iterable[str]] = ..., generated_outputs: _Optional[_Iterable[_Union[PackageGeneratedOutput, _Mapping]]] = ..., adoption: _Optional[_Union[PackageAdoptionPolicy, _Mapping]] = ..., lifecycle: _Optional[_Union[PackageLifecyclePolicy, _Mapping]] = ..., refresh: _Optional[_Union[PackageRefreshPolicy, _Mapping]] = ..., docs: _Optional[_Iterable[str]] = ...) -> None: ...

class PackageGeneratedOutput(_message.Message):
    __slots__ = ("name", "identifiers", "consumers")
    NAME_FIELD_NUMBER: _ClassVar[int]
    IDENTIFIERS_FIELD_NUMBER: _ClassVar[int]
    CONSUMERS_FIELD_NUMBER: _ClassVar[int]
    name: str
    identifiers: _containers.RepeatedScalarFieldContainer[str]
    consumers: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., identifiers: _Optional[_Iterable[str]] = ..., consumers: _Optional[_Iterable[str]] = ...) -> None: ...

class PackageAdoptionPolicy(_message.Message):
    __slots__ = ("scenario_adoptable", "allowed_consumers", "adoption_modes")
    SCENARIO_ADOPTABLE_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_CONSUMERS_FIELD_NUMBER: _ClassVar[int]
    ADOPTION_MODES_FIELD_NUMBER: _ClassVar[int]
    scenario_adoptable: bool
    allowed_consumers: _containers.RepeatedScalarFieldContainer[str]
    adoption_modes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario_adoptable: _Optional[bool] = ..., allowed_consumers: _Optional[_Iterable[str]] = ..., adoption_modes: _Optional[_Iterable[str]] = ...) -> None: ...

class PackageLifecyclePolicy(_message.Message):
    __slots__ = ("generate", "build")
    GENERATE_FIELD_NUMBER: _ClassVar[int]
    BUILD_FIELD_NUMBER: _ClassVar[int]
    generate: _containers.RepeatedCompositeFieldContainer[PackageCommandSpec]
    build: _containers.RepeatedCompositeFieldContainer[PackageCommandSpec]
    def __init__(self, generate: _Optional[_Iterable[_Union[PackageCommandSpec, _Mapping]]] = ..., build: _Optional[_Iterable[_Union[PackageCommandSpec, _Mapping]]] = ...) -> None: ...

class PackageCommandSpec(_message.Message):
    __slots__ = ("name", "run")
    NAME_FIELD_NUMBER: _ClassVar[int]
    RUN_FIELD_NUMBER: _ClassVar[int]
    name: str
    run: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., run: _Optional[_Iterable[str]] = ...) -> None: ...

class PackageRefreshPolicy(_message.Message):
    __slots__ = ("strategy", "restart_running_consumers")
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    RESTART_RUNNING_CONSUMERS_FIELD_NUMBER: _ClassVar[int]
    strategy: str
    restart_running_consumers: bool
    def __init__(self, strategy: _Optional[str] = ..., restart_running_consumers: _Optional[bool] = ...) -> None: ...

class PackageDependentsResponse(_message.Message):
    __slots__ = ("success", "dependents")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DEPENDENTS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    dependents: PackageDependents
    def __init__(self, success: _Optional[bool] = ..., dependents: _Optional[_Union[PackageDependents, _Mapping]] = ...) -> None: ...

class PackageDependents(_message.Message):
    __slots__ = ("package_name", "dependents", "issues")
    PACKAGE_NAME_FIELD_NUMBER: _ClassVar[int]
    DEPENDENTS_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    package_name: str
    dependents: _containers.RepeatedCompositeFieldContainer[PackageDependent]
    issues: _containers.RepeatedCompositeFieldContainer[PackageValidationIssue]
    def __init__(self, package_name: _Optional[str] = ..., dependents: _Optional[_Iterable[_Union[PackageDependent, _Mapping]]] = ..., issues: _Optional[_Iterable[_Union[PackageValidationIssue, _Mapping]]] = ...) -> None: ...

class PackageDependent(_message.Message):
    __slots__ = ("package_name", "consumer_name", "consumer_path", "consumer_class", "adoption_mode", "dependency_file", "dependency_target", "version")
    PACKAGE_NAME_FIELD_NUMBER: _ClassVar[int]
    CONSUMER_NAME_FIELD_NUMBER: _ClassVar[int]
    CONSUMER_PATH_FIELD_NUMBER: _ClassVar[int]
    CONSUMER_CLASS_FIELD_NUMBER: _ClassVar[int]
    ADOPTION_MODE_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCY_FILE_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCY_TARGET_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    package_name: str
    consumer_name: str
    consumer_path: str
    consumer_class: str
    adoption_mode: str
    dependency_file: str
    dependency_target: str
    version: str
    def __init__(self, package_name: _Optional[str] = ..., consumer_name: _Optional[str] = ..., consumer_path: _Optional[str] = ..., consumer_class: _Optional[str] = ..., adoption_mode: _Optional[str] = ..., dependency_file: _Optional[str] = ..., dependency_target: _Optional[str] = ..., version: _Optional[str] = ...) -> None: ...

class PackageValidateResponse(_message.Message):
    __slots__ = ("success", "report")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    report: PackageValidationReport
    def __init__(self, success: _Optional[bool] = ..., report: _Optional[_Union[PackageValidationReport, _Mapping]] = ...) -> None: ...

class PackageValidationReport(_message.Message):
    __slots__ = ("packages", "issues")
    PACKAGES_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    packages: _containers.RepeatedCompositeFieldContainer[PackageInfo]
    issues: _containers.RepeatedCompositeFieldContainer[PackageValidationIssue]
    def __init__(self, packages: _Optional[_Iterable[_Union[PackageInfo, _Mapping]]] = ..., issues: _Optional[_Iterable[_Union[PackageValidationIssue, _Mapping]]] = ...) -> None: ...

class PackageValidationIssue(_message.Message):
    __slots__ = ("severity", "code", "message", "path", "package_name")
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_NAME_FIELD_NUMBER: _ClassVar[int]
    severity: str
    code: str
    message: str
    path: str
    package_name: str
    def __init__(self, severity: _Optional[str] = ..., code: _Optional[str] = ..., message: _Optional[str] = ..., path: _Optional[str] = ..., package_name: _Optional[str] = ...) -> None: ...

class PackageRunResponse(_message.Message):
    __slots__ = ("success", "result")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    result: PackageRunResult
    def __init__(self, success: _Optional[bool] = ..., result: _Optional[_Union[PackageRunResult, _Mapping]] = ...) -> None: ...

class PackageRunResult(_message.Message):
    __slots__ = ("package_name", "action")
    PACKAGE_NAME_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    package_name: str
    action: str
    def __init__(self, package_name: _Optional[str] = ..., action: _Optional[str] = ...) -> None: ...

class PackageRefreshResponse(_message.Message):
    __slots__ = ("success", "refresh")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    REFRESH_FIELD_NUMBER: _ClassVar[int]
    success: bool
    refresh: PackageRefreshResult
    def __init__(self, success: _Optional[bool] = ..., refresh: _Optional[_Union[PackageRefreshResult, _Mapping]] = ...) -> None: ...

class PackageRefreshResult(_message.Message):
    __slots__ = ("package_name", "items")
    PACKAGE_NAME_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    package_name: str
    items: _containers.RepeatedCompositeFieldContainer[PackageRefreshItem]
    def __init__(self, package_name: _Optional[str] = ..., items: _Optional[_Iterable[_Union[PackageRefreshItem, _Mapping]]] = ...) -> None: ...

class PackageRefreshItem(_message.Message):
    __slots__ = ("consumer", "consumer_class", "consumer_classes", "action", "status")
    CONSUMER_FIELD_NUMBER: _ClassVar[int]
    CONSUMER_CLASS_FIELD_NUMBER: _ClassVar[int]
    CONSUMER_CLASSES_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    consumer: str
    consumer_class: str
    consumer_classes: _containers.RepeatedScalarFieldContainer[str]
    action: str
    status: str
    def __init__(self, consumer: _Optional[str] = ..., consumer_class: _Optional[str] = ..., consumer_classes: _Optional[_Iterable[str]] = ..., action: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class PackageAuditResponse(_message.Message):
    __slots__ = ("success", "audit")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    AUDIT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    audit: PackageAuditReport
    def __init__(self, success: _Optional[bool] = ..., audit: _Optional[_Union[PackageAuditReport, _Mapping]] = ...) -> None: ...

class PackageAuditReport(_message.Message):
    __slots__ = ("validation", "issues", "scan_stats")
    VALIDATION_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    SCAN_STATS_FIELD_NUMBER: _ClassVar[int]
    validation: PackageValidationReport
    issues: _containers.RepeatedCompositeFieldContainer[PackageValidationIssue]
    scan_stats: PackageAuditScanStats
    def __init__(self, validation: _Optional[_Union[PackageValidationReport, _Mapping]] = ..., issues: _Optional[_Iterable[_Union[PackageValidationIssue, _Mapping]]] = ..., scan_stats: _Optional[_Union[PackageAuditScanStats, _Mapping]] = ...) -> None: ...

class PackageAuditScanStats(_message.Message):
    __slots__ = ("files_visited", "files_scanned", "files_skipped", "bytes_scanned", "skipped_by_reason", "budget_exceeded")
    class SkippedByReasonEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    FILES_VISITED_FIELD_NUMBER: _ClassVar[int]
    FILES_SCANNED_FIELD_NUMBER: _ClassVar[int]
    FILES_SKIPPED_FIELD_NUMBER: _ClassVar[int]
    BYTES_SCANNED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_BY_REASON_FIELD_NUMBER: _ClassVar[int]
    BUDGET_EXCEEDED_FIELD_NUMBER: _ClassVar[int]
    files_visited: int
    files_scanned: int
    files_skipped: int
    bytes_scanned: int
    skipped_by_reason: _containers.ScalarMap[str, int]
    budget_exceeded: bool
    def __init__(self, files_visited: _Optional[int] = ..., files_scanned: _Optional[int] = ..., files_skipped: _Optional[int] = ..., bytes_scanned: _Optional[int] = ..., skipped_by_reason: _Optional[_Mapping[str, int]] = ..., budget_exceeded: _Optional[bool] = ...) -> None: ...
