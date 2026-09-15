from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ResourceControlResultItem(_message.Message):
    __slots__ = ("name", "message", "error")
    NAME_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    name: str
    message: str
    error: str
    def __init__(self, name: _Optional[str] = ..., message: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...

class ResourceStartAllResponse(_message.Message):
    __slots__ = ("success", "report")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    report: ResourceStartReport
    def __init__(self, success: _Optional[bool] = ..., report: _Optional[_Union[ResourceStartReport, _Mapping]] = ...) -> None: ...

class ResourceStartReport(_message.Message):
    __slots__ = ("started", "failed", "message")
    STARTED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    started: _containers.RepeatedCompositeFieldContainer[ResourceControlResultItem]
    failed: _containers.RepeatedCompositeFieldContainer[ResourceControlResultItem]
    message: str
    def __init__(self, started: _Optional[_Iterable[_Union[ResourceControlResultItem, _Mapping]]] = ..., failed: _Optional[_Iterable[_Union[ResourceControlResultItem, _Mapping]]] = ..., message: _Optional[str] = ...) -> None: ...

class ResourceStopAllResponse(_message.Message):
    __slots__ = ("success", "report")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    report: ResourceStopReport
    def __init__(self, success: _Optional[bool] = ..., report: _Optional[_Union[ResourceStopReport, _Mapping]] = ...) -> None: ...

class ResourceStopReport(_message.Message):
    __slots__ = ("stopped", "failed", "message")
    STOPPED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    stopped: _containers.RepeatedCompositeFieldContainer[ResourceControlResultItem]
    failed: _containers.RepeatedCompositeFieldContainer[ResourceControlResultItem]
    message: str
    def __init__(self, stopped: _Optional[_Iterable[_Union[ResourceControlResultItem, _Mapping]]] = ..., failed: _Optional[_Iterable[_Union[ResourceControlResultItem, _Mapping]]] = ..., message: _Optional[str] = ...) -> None: ...

class ResourceDeprecatedResource(_message.Message):
    __slots__ = ("name", "deprecated_at", "reason", "replacement", "archive_path", "archive_hash", "retention_policy_days", "restore_supported", "purge_after", "purged_at")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DEPRECATED_AT_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    REPLACEMENT_FIELD_NUMBER: _ClassVar[int]
    ARCHIVE_PATH_FIELD_NUMBER: _ClassVar[int]
    ARCHIVE_HASH_FIELD_NUMBER: _ClassVar[int]
    RETENTION_POLICY_DAYS_FIELD_NUMBER: _ClassVar[int]
    RESTORE_SUPPORTED_FIELD_NUMBER: _ClassVar[int]
    PURGE_AFTER_FIELD_NUMBER: _ClassVar[int]
    PURGED_AT_FIELD_NUMBER: _ClassVar[int]
    name: str
    deprecated_at: str
    reason: str
    replacement: str
    archive_path: str
    archive_hash: str
    retention_policy_days: int
    restore_supported: bool
    purge_after: str
    purged_at: str
    def __init__(self, name: _Optional[str] = ..., deprecated_at: _Optional[str] = ..., reason: _Optional[str] = ..., replacement: _Optional[str] = ..., archive_path: _Optional[str] = ..., archive_hash: _Optional[str] = ..., retention_policy_days: _Optional[int] = ..., restore_supported: _Optional[bool] = ..., purge_after: _Optional[str] = ..., purged_at: _Optional[str] = ...) -> None: ...

class ResourceListDeprecatedResponse(_message.Message):
    __slots__ = ("success", "resources")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    success: bool
    resources: _containers.RepeatedCompositeFieldContainer[ResourceDeprecatedResource]
    def __init__(self, success: _Optional[bool] = ..., resources: _Optional[_Iterable[_Union[ResourceDeprecatedResource, _Mapping]]] = ...) -> None: ...

class ResourceDeprecationResponse(_message.Message):
    __slots__ = ("success", "report")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    report: ResourceDeprecationReport
    def __init__(self, success: _Optional[bool] = ..., report: _Optional[_Union[ResourceDeprecationReport, _Mapping]] = ...) -> None: ...

class ResourceDeprecationReport(_message.Message):
    __slots__ = ("resource", "archived", "archive_dir")
    RESOURCE_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_FIELD_NUMBER: _ClassVar[int]
    ARCHIVE_DIR_FIELD_NUMBER: _ClassVar[int]
    resource: ResourceDeprecatedResource
    archived: bool
    archive_dir: str
    def __init__(self, resource: _Optional[_Union[ResourceDeprecatedResource, _Mapping]] = ..., archived: _Optional[bool] = ..., archive_dir: _Optional[str] = ...) -> None: ...

class ResourceRestoreResponse(_message.Message):
    __slots__ = ("success", "report")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    report: ResourceRestoreReport
    def __init__(self, success: _Optional[bool] = ..., report: _Optional[_Union[ResourceRestoreReport, _Mapping]] = ...) -> None: ...

class ResourceRestoreReport(_message.Message):
    __slots__ = ("resource", "restored", "restored_path")
    RESOURCE_FIELD_NUMBER: _ClassVar[int]
    RESTORED_FIELD_NUMBER: _ClassVar[int]
    RESTORED_PATH_FIELD_NUMBER: _ClassVar[int]
    resource: ResourceDeprecatedResource
    restored: bool
    restored_path: str
    def __init__(self, resource: _Optional[_Union[ResourceDeprecatedResource, _Mapping]] = ..., restored: _Optional[bool] = ..., restored_path: _Optional[str] = ...) -> None: ...

class ResourceBlueprintArchivedResource(_message.Message):
    __slots__ = ("name", "archived_at", "reason", "blueprint_name", "archive_path", "archive_hash", "retention_policy_days", "restore_supported", "purge_after", "purged_at")
    NAME_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_AT_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    BLUEPRINT_NAME_FIELD_NUMBER: _ClassVar[int]
    ARCHIVE_PATH_FIELD_NUMBER: _ClassVar[int]
    ARCHIVE_HASH_FIELD_NUMBER: _ClassVar[int]
    RETENTION_POLICY_DAYS_FIELD_NUMBER: _ClassVar[int]
    RESTORE_SUPPORTED_FIELD_NUMBER: _ClassVar[int]
    PURGE_AFTER_FIELD_NUMBER: _ClassVar[int]
    PURGED_AT_FIELD_NUMBER: _ClassVar[int]
    name: str
    archived_at: str
    reason: str
    blueprint_name: str
    archive_path: str
    archive_hash: str
    retention_policy_days: int
    restore_supported: bool
    purge_after: str
    purged_at: str
    def __init__(self, name: _Optional[str] = ..., archived_at: _Optional[str] = ..., reason: _Optional[str] = ..., blueprint_name: _Optional[str] = ..., archive_path: _Optional[str] = ..., archive_hash: _Optional[str] = ..., retention_policy_days: _Optional[int] = ..., restore_supported: _Optional[bool] = ..., purge_after: _Optional[str] = ..., purged_at: _Optional[str] = ...) -> None: ...

class ResourceListBlueprintArchivedResponse(_message.Message):
    __slots__ = ("success", "resources")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    success: bool
    resources: _containers.RepeatedCompositeFieldContainer[ResourceBlueprintArchivedResource]
    def __init__(self, success: _Optional[bool] = ..., resources: _Optional[_Iterable[_Union[ResourceBlueprintArchivedResource, _Mapping]]] = ...) -> None: ...

class ResourceBlueprintArchiveResponse(_message.Message):
    __slots__ = ("success", "report")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    report: ResourceBlueprintArchiveReport
    def __init__(self, success: _Optional[bool] = ..., report: _Optional[_Union[ResourceBlueprintArchiveReport, _Mapping]] = ...) -> None: ...

class ResourceBlueprintArchiveReport(_message.Message):
    __slots__ = ("resource", "archived", "archive_dir")
    RESOURCE_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_FIELD_NUMBER: _ClassVar[int]
    ARCHIVE_DIR_FIELD_NUMBER: _ClassVar[int]
    resource: ResourceBlueprintArchivedResource
    archived: bool
    archive_dir: str
    def __init__(self, resource: _Optional[_Union[ResourceBlueprintArchivedResource, _Mapping]] = ..., archived: _Optional[bool] = ..., archive_dir: _Optional[str] = ...) -> None: ...

class ResourceBlueprintRestoreResponse(_message.Message):
    __slots__ = ("success", "report")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    report: ResourceBlueprintRestoreReport
    def __init__(self, success: _Optional[bool] = ..., report: _Optional[_Union[ResourceBlueprintRestoreReport, _Mapping]] = ...) -> None: ...

class ResourceBlueprintRestoreReport(_message.Message):
    __slots__ = ("resource", "restored", "restored_path")
    RESOURCE_FIELD_NUMBER: _ClassVar[int]
    RESTORED_FIELD_NUMBER: _ClassVar[int]
    RESTORED_PATH_FIELD_NUMBER: _ClassVar[int]
    resource: ResourceBlueprintArchivedResource
    restored: bool
    restored_path: str
    def __init__(self, resource: _Optional[_Union[ResourceBlueprintArchivedResource, _Mapping]] = ..., restored: _Optional[bool] = ..., restored_path: _Optional[str] = ...) -> None: ...

class ResourceArchiveGCItem(_message.Message):
    __slots__ = ("name", "archive_path", "removed")
    NAME_FIELD_NUMBER: _ClassVar[int]
    ARCHIVE_PATH_FIELD_NUMBER: _ClassVar[int]
    REMOVED_FIELD_NUMBER: _ClassVar[int]
    name: str
    archive_path: str
    removed: bool
    def __init__(self, name: _Optional[str] = ..., archive_path: _Optional[str] = ..., removed: _Optional[bool] = ...) -> None: ...

class ResourceArchiveGCResponse(_message.Message):
    __slots__ = ("success", "report")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    report: ResourceArchiveGCReport
    def __init__(self, success: _Optional[bool] = ..., report: _Optional[_Union[ResourceArchiveGCReport, _Mapping]] = ...) -> None: ...

class ResourceArchiveGCReport(_message.Message):
    __slots__ = ("removed", "skipped")
    REMOVED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    removed: _containers.RepeatedCompositeFieldContainer[ResourceArchiveGCItem]
    skipped: _containers.RepeatedCompositeFieldContainer[ResourceArchiveGCItem]
    def __init__(self, removed: _Optional[_Iterable[_Union[ResourceArchiveGCItem, _Mapping]]] = ..., skipped: _Optional[_Iterable[_Union[ResourceArchiveGCItem, _Mapping]]] = ...) -> None: ...

class ResourceSchemaArtifactIssue(_message.Message):
    __slots__ = ("path", "message")
    PATH_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    path: str
    message: str
    def __init__(self, path: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class ResourceScenarioResourceReference(_message.Message):
    __slots__ = ("scenario", "resource", "manifest_path")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_PATH_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    resource: str
    manifest_path: str
    def __init__(self, scenario: _Optional[str] = ..., resource: _Optional[str] = ..., manifest_path: _Optional[str] = ...) -> None: ...

class ResourceSchemaValidationResponse(_message.Message):
    __slots__ = ("success", "report")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    report: ResourceSchemaValidationReport
    def __init__(self, success: _Optional[bool] = ..., report: _Optional[_Union[ResourceSchemaValidationReport, _Mapping]] = ...) -> None: ...

class ResourceSchemaValidationReport(_message.Message):
    __slots__ = ("passed", "resource_count", "definition_path", "artifact_issues", "missing_references")
    PASSED_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_COUNT_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_PATH_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_ISSUES_FIELD_NUMBER: _ClassVar[int]
    MISSING_REFERENCES_FIELD_NUMBER: _ClassVar[int]
    passed: bool
    resource_count: int
    definition_path: str
    artifact_issues: _containers.RepeatedCompositeFieldContainer[ResourceSchemaArtifactIssue]
    missing_references: _containers.RepeatedCompositeFieldContainer[ResourceScenarioResourceReference]
    def __init__(self, passed: _Optional[bool] = ..., resource_count: _Optional[int] = ..., definition_path: _Optional[str] = ..., artifact_issues: _Optional[_Iterable[_Union[ResourceSchemaArtifactIssue, _Mapping]]] = ..., missing_references: _Optional[_Iterable[_Union[ResourceScenarioResourceReference, _Mapping]]] = ...) -> None: ...

class ResourceSchemaSyncResponse(_message.Message):
    __slots__ = ("success", "report")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    report: ResourceSchemaSyncReport
    def __init__(self, success: _Optional[bool] = ..., report: _Optional[_Union[ResourceSchemaSyncReport, _Mapping]] = ...) -> None: ...

class ResourceSchemaSyncReport(_message.Message):
    __slots__ = ("passed", "resource_count", "definition_path", "written_paths", "missing_references")
    PASSED_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_COUNT_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_PATH_FIELD_NUMBER: _ClassVar[int]
    WRITTEN_PATHS_FIELD_NUMBER: _ClassVar[int]
    MISSING_REFERENCES_FIELD_NUMBER: _ClassVar[int]
    passed: bool
    resource_count: int
    definition_path: str
    written_paths: _containers.RepeatedScalarFieldContainer[str]
    missing_references: _containers.RepeatedCompositeFieldContainer[ResourceScenarioResourceReference]
    def __init__(self, passed: _Optional[bool] = ..., resource_count: _Optional[int] = ..., definition_path: _Optional[str] = ..., written_paths: _Optional[_Iterable[str]] = ..., missing_references: _Optional[_Iterable[_Union[ResourceScenarioResourceReference, _Mapping]]] = ...) -> None: ...

class ResourceValidationIssue(_message.Message):
    __slots__ = ("severity", "message")
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    severity: str
    message: str
    def __init__(self, severity: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class ResourceValidationItem(_message.Message):
    __slots__ = ("name", "manifest_path", "driver", "issues")
    NAME_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_PATH_FIELD_NUMBER: _ClassVar[int]
    DRIVER_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    name: str
    manifest_path: str
    driver: str
    issues: _containers.RepeatedCompositeFieldContainer[ResourceValidationIssue]
    def __init__(self, name: _Optional[str] = ..., manifest_path: _Optional[str] = ..., driver: _Optional[str] = ..., issues: _Optional[_Iterable[_Union[ResourceValidationIssue, _Mapping]]] = ...) -> None: ...

class ResourceValidationReport(_message.Message):
    __slots__ = ("count", "passed", "items", "issues")
    COUNT_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    count: int
    passed: bool
    items: _containers.RepeatedCompositeFieldContainer[ResourceValidationItem]
    issues: _containers.RepeatedCompositeFieldContainer[ResourceValidationIssue]
    def __init__(self, count: _Optional[int] = ..., passed: _Optional[bool] = ..., items: _Optional[_Iterable[_Union[ResourceValidationItem, _Mapping]]] = ..., issues: _Optional[_Iterable[_Union[ResourceValidationIssue, _Mapping]]] = ...) -> None: ...

class ResourceValidationResponse(_message.Message):
    __slots__ = ("success", "report")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    report: ResourceValidationReport
    def __init__(self, success: _Optional[bool] = ..., report: _Optional[_Union[ResourceValidationReport, _Mapping]] = ...) -> None: ...
