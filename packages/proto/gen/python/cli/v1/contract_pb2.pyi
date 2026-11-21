from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ContractValidationOutput(_message.Message):
    __slots__ = ("success", "root", "schema", "report")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ROOT_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    root: str
    schema: ContractValidationCheck
    report: ContractCheckReport
    def __init__(self, success: _Optional[bool] = ..., root: _Optional[str] = ..., schema: _Optional[_Union[ContractValidationCheck, _Mapping]] = ..., report: _Optional[_Union[ContractCheckReport, _Mapping]] = ...) -> None: ...

class ContractValidationCheck(_message.Message):
    __slots__ = ("passed", "message")
    PASSED_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    passed: bool
    message: str
    def __init__(self, passed: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...

class ContractCheckReport(_message.Message):
    __slots__ = ("root", "contract_path", "success", "checks")
    ROOT_FIELD_NUMBER: _ClassVar[int]
    CONTRACT_PATH_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    root: str
    contract_path: str
    success: bool
    checks: _containers.RepeatedCompositeFieldContainer[ContractCheckResult]
    def __init__(self, root: _Optional[str] = ..., contract_path: _Optional[str] = ..., success: _Optional[bool] = ..., checks: _Optional[_Iterable[_Union[ContractCheckResult, _Mapping]]] = ...) -> None: ...

class ContractCheckResult(_message.Message):
    __slots__ = ("name", "passed", "message")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    name: str
    passed: bool
    message: str
    def __init__(self, name: _Optional[str] = ..., passed: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...

class ContractShowOutput(_message.Message):
    __slots__ = ("success", "root", "contract_path", "schema", "version", "platform", "markers", "layout", "scenario", "resource", "globs", "environment", "sandbox", "profiles")
    class EnvironmentEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class ProfilesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: ContractProfile
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[ContractProfile, _Mapping]] = ...) -> None: ...
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ROOT_FIELD_NUMBER: _ClassVar[int]
    CONTRACT_PATH_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    MARKERS_FIELD_NUMBER: _ClassVar[int]
    LAYOUT_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_FIELD_NUMBER: _ClassVar[int]
    GLOBS_FIELD_NUMBER: _ClassVar[int]
    ENVIRONMENT_FIELD_NUMBER: _ClassVar[int]
    SANDBOX_FIELD_NUMBER: _ClassVar[int]
    PROFILES_FIELD_NUMBER: _ClassVar[int]
    success: bool
    root: str
    contract_path: str
    schema: str
    version: str
    platform: ContractPlatform
    markers: ContractRootMarkers
    layout: ContractLayout
    scenario: ContractScenarioSpec
    resource: ContractResourceSpec
    globs: ContractGlobSpec
    environment: _containers.ScalarMap[str, str]
    sandbox: ContractShowSandbox
    profiles: _containers.MessageMap[str, ContractProfile]
    def __init__(self, success: _Optional[bool] = ..., root: _Optional[str] = ..., contract_path: _Optional[str] = ..., schema: _Optional[str] = ..., version: _Optional[str] = ..., platform: _Optional[_Union[ContractPlatform, _Mapping]] = ..., markers: _Optional[_Union[ContractRootMarkers, _Mapping]] = ..., layout: _Optional[_Union[ContractLayout, _Mapping]] = ..., scenario: _Optional[_Union[ContractScenarioSpec, _Mapping]] = ..., resource: _Optional[_Union[ContractResourceSpec, _Mapping]] = ..., globs: _Optional[_Union[ContractGlobSpec, _Mapping]] = ..., environment: _Optional[_Mapping[str, str]] = ..., sandbox: _Optional[_Union[ContractShowSandbox, _Mapping]] = ..., profiles: _Optional[_Mapping[str, ContractProfile]] = ...) -> None: ...

class ContractPlatform(_message.Message):
    __slots__ = ("mode", "legacy_project_bash_supported")
    MODE_FIELD_NUMBER: _ClassVar[int]
    LEGACY_PROJECT_BASH_SUPPORTED_FIELD_NUMBER: _ClassVar[int]
    mode: str
    legacy_project_bash_supported: bool
    def __init__(self, mode: _Optional[str] = ..., legacy_project_bash_supported: _Optional[bool] = ...) -> None: ...

class ContractRootMarkers(_message.Message):
    __slots__ = ("required_dirs", "required_files")
    REQUIRED_DIRS_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FILES_FIELD_NUMBER: _ClassVar[int]
    required_dirs: _containers.RepeatedScalarFieldContainer[str]
    required_files: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, required_dirs: _Optional[_Iterable[str]] = ..., required_files: _Optional[_Iterable[str]] = ...) -> None: ...

class ContractLayout(_message.Message):
    __slots__ = ("project_config_dir", "scenario_dir", "resource_dir", "template_dir", "package_dir", "command_dir", "internal_dir", "docs_dir")
    PROJECT_CONFIG_DIR_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_DIR_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_DIR_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_DIR_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_DIR_FIELD_NUMBER: _ClassVar[int]
    COMMAND_DIR_FIELD_NUMBER: _ClassVar[int]
    INTERNAL_DIR_FIELD_NUMBER: _ClassVar[int]
    DOCS_DIR_FIELD_NUMBER: _ClassVar[int]
    project_config_dir: str
    scenario_dir: str
    resource_dir: str
    template_dir: str
    package_dir: str
    command_dir: str
    internal_dir: str
    docs_dir: str
    def __init__(self, project_config_dir: _Optional[str] = ..., scenario_dir: _Optional[str] = ..., resource_dir: _Optional[str] = ..., template_dir: _Optional[str] = ..., package_dir: _Optional[str] = ..., command_dir: _Optional[str] = ..., internal_dir: _Optional[str] = ..., docs_dir: _Optional[str] = ...) -> None: ...

class ContractScenarioSpec(_message.Message):
    __slots__ = ("required_files", "well_known_paths")
    class WellKnownPathsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    REQUIRED_FILES_FIELD_NUMBER: _ClassVar[int]
    WELL_KNOWN_PATHS_FIELD_NUMBER: _ClassVar[int]
    required_files: _containers.RepeatedScalarFieldContainer[str]
    well_known_paths: _containers.ScalarMap[str, str]
    def __init__(self, required_files: _Optional[_Iterable[str]] = ..., well_known_paths: _Optional[_Mapping[str, str]] = ...) -> None: ...

class ContractResourceSpec(_message.Message):
    __slots__ = ("manifest", "well_known_paths")
    class WellKnownPathsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    MANIFEST_FIELD_NUMBER: _ClassVar[int]
    WELL_KNOWN_PATHS_FIELD_NUMBER: _ClassVar[int]
    manifest: str
    well_known_paths: _containers.ScalarMap[str, str]
    def __init__(self, manifest: _Optional[str] = ..., well_known_paths: _Optional[_Mapping[str, str]] = ...) -> None: ...

class ContractGlobSpec(_message.Message):
    __slots__ = ("syntax", "root_relative", "case_sensitive", "allow_absolute", "path_format")
    SYNTAX_FIELD_NUMBER: _ClassVar[int]
    ROOT_RELATIVE_FIELD_NUMBER: _ClassVar[int]
    CASE_SENSITIVE_FIELD_NUMBER: _ClassVar[int]
    ALLOW_ABSOLUTE_FIELD_NUMBER: _ClassVar[int]
    PATH_FORMAT_FIELD_NUMBER: _ClassVar[int]
    syntax: str
    root_relative: bool
    case_sensitive: bool
    allow_absolute: bool
    path_format: str
    def __init__(self, syntax: _Optional[str] = ..., root_relative: _Optional[bool] = ..., case_sensitive: _Optional[bool] = ..., allow_absolute: _Optional[bool] = ..., path_format: _Optional[str] = ...) -> None: ...

class ContractShowSandbox(_message.Message):
    __slots__ = ("full_repo_scopes", "scenario_scope_prefix")
    FULL_REPO_SCOPES_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_SCOPE_PREFIX_FIELD_NUMBER: _ClassVar[int]
    full_repo_scopes: _containers.RepeatedScalarFieldContainer[str]
    scenario_scope_prefix: str
    def __init__(self, full_repo_scopes: _Optional[_Iterable[str]] = ..., scenario_scope_prefix: _Optional[str] = ...) -> None: ...

class ContractProfile(_message.Message):
    __slots__ = ("description", "parameters", "include", "optional_include", "exclude")
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PARAMETERS_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_FIELD_NUMBER: _ClassVar[int]
    OPTIONAL_INCLUDE_FIELD_NUMBER: _ClassVar[int]
    EXCLUDE_FIELD_NUMBER: _ClassVar[int]
    description: str
    parameters: _containers.RepeatedScalarFieldContainer[str]
    include: _containers.RepeatedScalarFieldContainer[str]
    optional_include: _containers.RepeatedScalarFieldContainer[str]
    exclude: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, description: _Optional[str] = ..., parameters: _Optional[_Iterable[str]] = ..., include: _Optional[_Iterable[str]] = ..., optional_include: _Optional[_Iterable[str]] = ..., exclude: _Optional[_Iterable[str]] = ...) -> None: ...

class ContractResolveScenarioOutput(_message.Message):
    __slots__ = ("success", "root", "scenario", "file", "path")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ROOT_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    FILE_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    success: bool
    root: str
    scenario: str
    file: str
    path: str
    def __init__(self, success: _Optional[bool] = ..., root: _Optional[str] = ..., scenario: _Optional[str] = ..., file: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class ContractMatchGlobOutput(_message.Message):
    __slots__ = ("success", "pattern", "path", "matched")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    PATTERN_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    MATCHED_FIELD_NUMBER: _ClassVar[int]
    success: bool
    pattern: str
    path: str
    matched: bool
    def __init__(self, success: _Optional[bool] = ..., pattern: _Optional[str] = ..., path: _Optional[str] = ..., matched: _Optional[bool] = ...) -> None: ...
