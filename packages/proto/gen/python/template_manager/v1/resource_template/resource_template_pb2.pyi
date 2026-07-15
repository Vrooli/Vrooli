from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ResourceTemplateVar(_message.Message):
    __slots__ = ("flag", "description", "default")
    FLAG_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_FIELD_NUMBER: _ClassVar[int]
    flag: str
    description: str
    default: str
    def __init__(self, flag: _Optional[str] = ..., description: _Optional[str] = ..., default: _Optional[str] = ...) -> None: ...

class ResourceTemplateManifest(_message.Message):
    __slots__ = ("name", "display_name", "description", "driver", "required_vars", "optional_vars", "docs", "platform_expectations", "transitional")
    class RequiredVarsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: ResourceTemplateVar
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[ResourceTemplateVar, _Mapping]] = ...) -> None: ...
    class OptionalVarsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: ResourceTemplateVar
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[ResourceTemplateVar, _Mapping]] = ...) -> None: ...
    class DocsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    DRIVER_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_VARS_FIELD_NUMBER: _ClassVar[int]
    OPTIONAL_VARS_FIELD_NUMBER: _ClassVar[int]
    DOCS_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_EXPECTATIONS_FIELD_NUMBER: _ClassVar[int]
    TRANSITIONAL_FIELD_NUMBER: _ClassVar[int]
    name: str
    display_name: str
    description: str
    driver: str
    required_vars: _containers.MessageMap[str, ResourceTemplateVar]
    optional_vars: _containers.MessageMap[str, ResourceTemplateVar]
    docs: _containers.ScalarMap[str, str]
    platform_expectations: _containers.RepeatedScalarFieldContainer[str]
    transitional: bool
    def __init__(self, name: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., driver: _Optional[str] = ..., required_vars: _Optional[_Mapping[str, ResourceTemplateVar]] = ..., optional_vars: _Optional[_Mapping[str, ResourceTemplateVar]] = ..., docs: _Optional[_Mapping[str, str]] = ..., platform_expectations: _Optional[_Iterable[str]] = ..., transitional: _Optional[bool] = ...) -> None: ...

class ResourceTemplateInfo(_message.Message):
    __slots__ = ("name", "path", "manifest")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_FIELD_NUMBER: _ClassVar[int]
    name: str
    path: str
    manifest: ResourceTemplateManifest
    def __init__(self, name: _Optional[str] = ..., path: _Optional[str] = ..., manifest: _Optional[_Union[ResourceTemplateManifest, _Mapping]] = ...) -> None: ...

class ResourceTemplateSummary(_message.Message):
    __slots__ = ("name", "display_name", "driver", "transitional", "description")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DRIVER_FIELD_NUMBER: _ClassVar[int]
    TRANSITIONAL_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    name: str
    display_name: str
    driver: str
    transitional: bool
    description: str
    def __init__(self, name: _Optional[str] = ..., display_name: _Optional[str] = ..., driver: _Optional[str] = ..., transitional: _Optional[bool] = ..., description: _Optional[str] = ...) -> None: ...

class ListResourceTemplatesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListResourceTemplatesResponse(_message.Message):
    __slots__ = ("templates",)
    TEMPLATES_FIELD_NUMBER: _ClassVar[int]
    templates: _containers.RepeatedCompositeFieldContainer[ResourceTemplateInfo]
    def __init__(self, templates: _Optional[_Iterable[_Union[ResourceTemplateInfo, _Mapping]]] = ...) -> None: ...

class GetResourceTemplateRequest(_message.Message):
    __slots__ = ("name",)
    NAME_FIELD_NUMBER: _ClassVar[int]
    name: str
    def __init__(self, name: _Optional[str] = ...) -> None: ...

class GetResourceTemplateResponse(_message.Message):
    __slots__ = ("template",)
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    template: ResourceTemplateInfo
    def __init__(self, template: _Optional[_Union[ResourceTemplateInfo, _Mapping]] = ...) -> None: ...

class ValidateResourceTemplatesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ResourceTemplateValidationResult(_message.Message):
    __slots__ = ("name", "driver", "transitional", "status", "issues")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DRIVER_FIELD_NUMBER: _ClassVar[int]
    TRANSITIONAL_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    name: str
    driver: str
    transitional: bool
    status: str
    issues: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., driver: _Optional[str] = ..., transitional: _Optional[bool] = ..., status: _Optional[str] = ..., issues: _Optional[_Iterable[str]] = ...) -> None: ...

class ValidateResourceTemplatesResponse(_message.Message):
    __slots__ = ("count", "status", "issues_count", "results", "issues")
    COUNT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ISSUES_COUNT_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    count: int
    status: str
    issues_count: int
    results: _containers.RepeatedCompositeFieldContainer[ResourceTemplateValidationResult]
    issues: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, count: _Optional[int] = ..., status: _Optional[str] = ..., issues_count: _Optional[int] = ..., results: _Optional[_Iterable[_Union[ResourceTemplateValidationResult, _Mapping]]] = ..., issues: _Optional[_Iterable[str]] = ...) -> None: ...

class GenerateResourceTemplateRequest(_message.Message):
    __slots__ = ("template", "from_blueprint", "destination", "force", "dry_run", "values")
    class ValuesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    FROM_BLUEPRINT_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_FIELD_NUMBER: _ClassVar[int]
    FORCE_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    VALUES_FIELD_NUMBER: _ClassVar[int]
    template: str
    from_blueprint: str
    destination: str
    force: bool
    dry_run: bool
    values: _containers.ScalarMap[str, str]
    def __init__(self, template: _Optional[str] = ..., from_blueprint: _Optional[str] = ..., destination: _Optional[str] = ..., force: _Optional[bool] = ..., dry_run: _Optional[bool] = ..., values: _Optional[_Mapping[str, str]] = ...) -> None: ...

class GenerateResourceTemplateResponse(_message.Message):
    __slots__ = ("template", "blueprint_name", "destination", "values", "files", "dry_run")
    class ValuesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    BLUEPRINT_NAME_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_FIELD_NUMBER: _ClassVar[int]
    VALUES_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    template: ResourceTemplateSummary
    blueprint_name: str
    destination: str
    values: _containers.ScalarMap[str, str]
    files: _containers.RepeatedScalarFieldContainer[str]
    dry_run: bool
    def __init__(self, template: _Optional[_Union[ResourceTemplateSummary, _Mapping]] = ..., blueprint_name: _Optional[str] = ..., destination: _Optional[str] = ..., values: _Optional[_Mapping[str, str]] = ..., files: _Optional[_Iterable[str]] = ..., dry_run: _Optional[bool] = ...) -> None: ...
