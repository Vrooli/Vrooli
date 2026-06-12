from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ResourceBlueprint(_message.Message):
    __slots__ = ("schema", "name", "display_name", "category", "summary", "why_it_matters", "when_to_use", "example_scenarios", "integration_kind", "platform_support", "prerequisites", "dependencies", "suggested_template", "implementation_notes", "operational_notes", "risks", "status", "replacement_for", "references", "last_reviewed")
    SCHEMA_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    WHY_IT_MATTERS_FIELD_NUMBER: _ClassVar[int]
    WHEN_TO_USE_FIELD_NUMBER: _ClassVar[int]
    EXAMPLE_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    INTEGRATION_KIND_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_SUPPORT_FIELD_NUMBER: _ClassVar[int]
    PREREQUISITES_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    SUGGESTED_TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    IMPLEMENTATION_NOTES_FIELD_NUMBER: _ClassVar[int]
    OPERATIONAL_NOTES_FIELD_NUMBER: _ClassVar[int]
    RISKS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REPLACEMENT_FOR_FIELD_NUMBER: _ClassVar[int]
    REFERENCES_FIELD_NUMBER: _ClassVar[int]
    LAST_REVIEWED_FIELD_NUMBER: _ClassVar[int]
    schema: str
    name: str
    display_name: str
    category: str
    summary: str
    why_it_matters: str
    when_to_use: _containers.RepeatedScalarFieldContainer[str]
    example_scenarios: _containers.RepeatedScalarFieldContainer[str]
    integration_kind: str
    platform_support: ResourceBlueprintPlatformSupport
    prerequisites: _containers.RepeatedScalarFieldContainer[str]
    dependencies: _containers.RepeatedScalarFieldContainer[str]
    suggested_template: str
    implementation_notes: _containers.RepeatedScalarFieldContainer[str]
    operational_notes: _containers.RepeatedScalarFieldContainer[str]
    risks: _containers.RepeatedScalarFieldContainer[str]
    status: str
    replacement_for: _containers.RepeatedScalarFieldContainer[str]
    references: _containers.RepeatedCompositeFieldContainer[ResourceBlueprintReference]
    last_reviewed: str
    def __init__(self, schema: _Optional[str] = ..., name: _Optional[str] = ..., display_name: _Optional[str] = ..., category: _Optional[str] = ..., summary: _Optional[str] = ..., why_it_matters: _Optional[str] = ..., when_to_use: _Optional[_Iterable[str]] = ..., example_scenarios: _Optional[_Iterable[str]] = ..., integration_kind: _Optional[str] = ..., platform_support: _Optional[_Union[ResourceBlueprintPlatformSupport, _Mapping]] = ..., prerequisites: _Optional[_Iterable[str]] = ..., dependencies: _Optional[_Iterable[str]] = ..., suggested_template: _Optional[str] = ..., implementation_notes: _Optional[_Iterable[str]] = ..., operational_notes: _Optional[_Iterable[str]] = ..., risks: _Optional[_Iterable[str]] = ..., status: _Optional[str] = ..., replacement_for: _Optional[_Iterable[str]] = ..., references: _Optional[_Iterable[_Union[ResourceBlueprintReference, _Mapping]]] = ..., last_reviewed: _Optional[str] = ...) -> None: ...

class ResourceBlueprintPlatformSupport(_message.Message):
    __slots__ = ("portability_tier", "notes", "linux", "macos", "windows")
    PORTABILITY_TIER_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    LINUX_FIELD_NUMBER: _ClassVar[int]
    MACOS_FIELD_NUMBER: _ClassVar[int]
    WINDOWS_FIELD_NUMBER: _ClassVar[int]
    portability_tier: str
    notes: str
    linux: str
    macos: str
    windows: str
    def __init__(self, portability_tier: _Optional[str] = ..., notes: _Optional[str] = ..., linux: _Optional[str] = ..., macos: _Optional[str] = ..., windows: _Optional[str] = ...) -> None: ...

class ResourceBlueprintReference(_message.Message):
    __slots__ = ("kind", "value")
    KIND_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    kind: str
    value: str
    def __init__(self, kind: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...

class ResourceBlueprintSummary(_message.Message):
    __slots__ = ("name", "display_name", "category", "status", "integration_kind", "suggested_template", "last_reviewed", "summary")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    INTEGRATION_KIND_FIELD_NUMBER: _ClassVar[int]
    SUGGESTED_TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    LAST_REVIEWED_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    name: str
    display_name: str
    category: str
    status: str
    integration_kind: str
    suggested_template: str
    last_reviewed: str
    summary: str
    def __init__(self, name: _Optional[str] = ..., display_name: _Optional[str] = ..., category: _Optional[str] = ..., status: _Optional[str] = ..., integration_kind: _Optional[str] = ..., suggested_template: _Optional[str] = ..., last_reviewed: _Optional[str] = ..., summary: _Optional[str] = ...) -> None: ...

class ResourceBlueprintListResponse(_message.Message):
    __slots__ = ("success", "blueprints")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    BLUEPRINTS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    blueprints: _containers.RepeatedCompositeFieldContainer[ResourceBlueprint]
    def __init__(self, success: _Optional[bool] = ..., blueprints: _Optional[_Iterable[_Union[ResourceBlueprint, _Mapping]]] = ...) -> None: ...

class ResourceBlueprintInfoResponse(_message.Message):
    __slots__ = ("success", "blueprint")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    BLUEPRINT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    blueprint: ResourceBlueprint
    def __init__(self, success: _Optional[bool] = ..., blueprint: _Optional[_Union[ResourceBlueprint, _Mapping]] = ...) -> None: ...

class ResourceBlueprintSearchResponse(_message.Message):
    __slots__ = ("success", "query", "blueprints")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    BLUEPRINTS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    query: str
    blueprints: _containers.RepeatedCompositeFieldContainer[ResourceBlueprint]
    def __init__(self, success: _Optional[bool] = ..., query: _Optional[str] = ..., blueprints: _Optional[_Iterable[_Union[ResourceBlueprint, _Mapping]]] = ...) -> None: ...

class ResourceBlueprintValidationReport(_message.Message):
    __slots__ = ("success", "report")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    report: ResourceBlueprintValidationReportBody
    def __init__(self, success: _Optional[bool] = ..., report: _Optional[_Union[ResourceBlueprintValidationReportBody, _Mapping]] = ...) -> None: ...

class ResourceBlueprintValidationReportBody(_message.Message):
    __slots__ = ("blueprints", "count")
    BLUEPRINTS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    blueprints: _containers.RepeatedCompositeFieldContainer[ResourceBlueprintSummary]
    count: int
    def __init__(self, blueprints: _Optional[_Iterable[_Union[ResourceBlueprintSummary, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...
