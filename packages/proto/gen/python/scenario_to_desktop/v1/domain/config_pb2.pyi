import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from scenario_to_desktop.v1.shared import common_pb2 as _common_pb2
from scenario_to_desktop.v1.shared import metadata_pb2 as _metadata_pb2
from scenario_to_desktop.v1.shared import update_config_pb2 as _update_config_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AppIdentity(_message.Message):
    __slots__ = ("name", "display_name", "description", "version", "author", "email", "icon", "homepage", "license", "app_id", "app_url")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    AUTHOR_FIELD_NUMBER: _ClassVar[int]
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    ICON_FIELD_NUMBER: _ClassVar[int]
    HOMEPAGE_FIELD_NUMBER: _ClassVar[int]
    LICENSE_FIELD_NUMBER: _ClassVar[int]
    APP_ID_FIELD_NUMBER: _ClassVar[int]
    APP_URL_FIELD_NUMBER: _ClassVar[int]
    name: str
    display_name: str
    description: str
    version: str
    author: str
    email: str
    icon: str
    homepage: str
    license: str
    app_id: str
    app_url: str
    def __init__(self, name: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., version: _Optional[str] = ..., author: _Optional[str] = ..., email: _Optional[str] = ..., icon: _Optional[str] = ..., homepage: _Optional[str] = ..., license: _Optional[str] = ..., app_id: _Optional[str] = ..., app_url: _Optional[str] = ...) -> None: ...

class ServerConfig(_message.Message):
    __slots__ = ("server_type", "port", "path", "api_endpoint", "scenario_path", "auto_manage_vrooli", "vrooli_binary_path", "deployment_mode", "proxy_url", "external_server_url", "external_api_url")
    SERVER_TYPE_FIELD_NUMBER: _ClassVar[int]
    PORT_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    API_ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_PATH_FIELD_NUMBER: _ClassVar[int]
    AUTO_MANAGE_VROOLI_FIELD_NUMBER: _ClassVar[int]
    VROOLI_BINARY_PATH_FIELD_NUMBER: _ClassVar[int]
    DEPLOYMENT_MODE_FIELD_NUMBER: _ClassVar[int]
    PROXY_URL_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_SERVER_URL_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_API_URL_FIELD_NUMBER: _ClassVar[int]
    server_type: str
    port: int
    path: str
    api_endpoint: str
    scenario_path: str
    auto_manage_vrooli: bool
    vrooli_binary_path: str
    deployment_mode: _common_pb2.DeploymentMode
    proxy_url: str
    external_server_url: str
    external_api_url: str
    def __init__(self, server_type: _Optional[str] = ..., port: _Optional[int] = ..., path: _Optional[str] = ..., api_endpoint: _Optional[str] = ..., scenario_path: _Optional[str] = ..., auto_manage_vrooli: _Optional[bool] = ..., vrooli_binary_path: _Optional[str] = ..., deployment_mode: _Optional[_Union[_common_pb2.DeploymentMode, str]] = ..., proxy_url: _Optional[str] = ..., external_server_url: _Optional[str] = ..., external_api_url: _Optional[str] = ...) -> None: ...

class BundleIPCConfig(_message.Message):
    __slots__ = ("host", "port", "auth_token_rel_path")
    HOST_FIELD_NUMBER: _ClassVar[int]
    PORT_FIELD_NUMBER: _ClassVar[int]
    AUTH_TOKEN_REL_PATH_FIELD_NUMBER: _ClassVar[int]
    host: str
    port: int
    auth_token_rel_path: str
    def __init__(self, host: _Optional[str] = ..., port: _Optional[int] = ..., auth_token_rel_path: _Optional[str] = ...) -> None: ...

class BundleConfig(_message.Message):
    __slots__ = ("manifest_path", "runtime_root", "ipc", "ui_service_id", "port_name", "telemetry_upload_url")
    MANIFEST_PATH_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_ROOT_FIELD_NUMBER: _ClassVar[int]
    IPC_FIELD_NUMBER: _ClassVar[int]
    UI_SERVICE_ID_FIELD_NUMBER: _ClassVar[int]
    PORT_NAME_FIELD_NUMBER: _ClassVar[int]
    TELEMETRY_UPLOAD_URL_FIELD_NUMBER: _ClassVar[int]
    manifest_path: str
    runtime_root: str
    ipc: BundleIPCConfig
    ui_service_id: str
    port_name: str
    telemetry_upload_url: str
    def __init__(self, manifest_path: _Optional[str] = ..., runtime_root: _Optional[str] = ..., ipc: _Optional[_Union[BundleIPCConfig, _Mapping]] = ..., ui_service_id: _Optional[str] = ..., port_name: _Optional[str] = ..., telemetry_upload_url: _Optional[str] = ...) -> None: ...

class WindowConfig(_message.Message):
    __slots__ = ("width", "height", "min_width", "min_height", "resizable", "frame", "dev_tools")
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    MIN_WIDTH_FIELD_NUMBER: _ClassVar[int]
    MIN_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    RESIZABLE_FIELD_NUMBER: _ClassVar[int]
    FRAME_FIELD_NUMBER: _ClassVar[int]
    DEV_TOOLS_FIELD_NUMBER: _ClassVar[int]
    width: int
    height: int
    min_width: int
    min_height: int
    resizable: bool
    frame: bool
    dev_tools: bool
    def __init__(self, width: _Optional[int] = ..., height: _Optional[int] = ..., min_width: _Optional[int] = ..., min_height: _Optional[int] = ..., resizable: _Optional[bool] = ..., frame: _Optional[bool] = ..., dev_tools: _Optional[bool] = ...) -> None: ...

class DesktopConfig(_message.Message):
    __slots__ = ("app", "server", "bundle", "update", "window", "framework", "template_type", "platforms", "output_path", "features", "styling", "signing_enabled")
    class FeaturesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: bool
        def __init__(self, key: _Optional[str] = ..., value: _Optional[bool] = ...) -> None: ...
    class StylingEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    APP_FIELD_NUMBER: _ClassVar[int]
    SERVER_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_FIELD_NUMBER: _ClassVar[int]
    UPDATE_FIELD_NUMBER: _ClassVar[int]
    WINDOW_FIELD_NUMBER: _ClassVar[int]
    FRAMEWORK_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_TYPE_FIELD_NUMBER: _ClassVar[int]
    PLATFORMS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_PATH_FIELD_NUMBER: _ClassVar[int]
    FEATURES_FIELD_NUMBER: _ClassVar[int]
    STYLING_FIELD_NUMBER: _ClassVar[int]
    SIGNING_ENABLED_FIELD_NUMBER: _ClassVar[int]
    app: AppIdentity
    server: ServerConfig
    bundle: BundleConfig
    update: _update_config_pb2.UpdateConfig
    window: WindowConfig
    framework: _common_pb2.Framework
    template_type: _common_pb2.TemplateType
    platforms: _containers.RepeatedScalarFieldContainer[_common_pb2.Platform]
    output_path: str
    features: _containers.ScalarMap[str, bool]
    styling: _containers.ScalarMap[str, str]
    signing_enabled: bool
    def __init__(self, app: _Optional[_Union[AppIdentity, _Mapping]] = ..., server: _Optional[_Union[ServerConfig, _Mapping]] = ..., bundle: _Optional[_Union[BundleConfig, _Mapping]] = ..., update: _Optional[_Union[_update_config_pb2.UpdateConfig, _Mapping]] = ..., window: _Optional[_Union[WindowConfig, _Mapping]] = ..., framework: _Optional[_Union[_common_pb2.Framework, str]] = ..., template_type: _Optional[_Union[_common_pb2.TemplateType, str]] = ..., platforms: _Optional[_Iterable[_Union[_common_pb2.Platform, str]]] = ..., output_path: _Optional[str] = ..., features: _Optional[_Mapping[str, bool]] = ..., styling: _Optional[_Mapping[str, str]] = ..., signing_enabled: _Optional[bool] = ...) -> None: ...

class ConnectionConfig(_message.Message):
    __slots__ = ("proxy_url", "server_type", "auto_manage_vrooli", "vrooli_binary_path", "deployment_mode", "bundle_manifest_path", "app_display_name", "app_description", "icon")
    PROXY_URL_FIELD_NUMBER: _ClassVar[int]
    SERVER_TYPE_FIELD_NUMBER: _ClassVar[int]
    AUTO_MANAGE_VROOLI_FIELD_NUMBER: _ClassVar[int]
    VROOLI_BINARY_PATH_FIELD_NUMBER: _ClassVar[int]
    DEPLOYMENT_MODE_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_MANIFEST_PATH_FIELD_NUMBER: _ClassVar[int]
    APP_DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    APP_DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ICON_FIELD_NUMBER: _ClassVar[int]
    proxy_url: str
    server_type: str
    auto_manage_vrooli: bool
    vrooli_binary_path: str
    deployment_mode: _common_pb2.DeploymentMode
    bundle_manifest_path: str
    app_display_name: str
    app_description: str
    icon: str
    def __init__(self, proxy_url: _Optional[str] = ..., server_type: _Optional[str] = ..., auto_manage_vrooli: _Optional[bool] = ..., vrooli_binary_path: _Optional[str] = ..., deployment_mode: _Optional[_Union[_common_pb2.DeploymentMode, str]] = ..., bundle_manifest_path: _Optional[str] = ..., app_display_name: _Optional[str] = ..., app_description: _Optional[str] = ..., icon: _Optional[str] = ...) -> None: ...

class GetScenarioMetadataRequest(_message.Message):
    __slots__ = ("scenario_name",)
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    def __init__(self, scenario_name: _Optional[str] = ...) -> None: ...

class CreateDesktopConfigRequest(_message.Message):
    __slots__ = ("metadata", "template_type")
    METADATA_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_TYPE_FIELD_NUMBER: _ClassVar[int]
    metadata: _metadata_pb2.ScenarioMetadata
    template_type: _common_pb2.TemplateType
    def __init__(self, metadata: _Optional[_Union[_metadata_pb2.ScenarioMetadata, _Mapping]] = ..., template_type: _Optional[_Union[_common_pb2.TemplateType, str]] = ...) -> None: ...

class GetSystemStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SystemServiceInfo(_message.Message):
    __slots__ = ("name", "version", "description", "status")
    NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    name: str
    version: str
    description: str
    status: str
    def __init__(self, name: _Optional[str] = ..., version: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class SystemBuildStatistics(_message.Message):
    __slots__ = ("total_builds", "active_builds", "completed_builds", "failed_builds")
    TOTAL_BUILDS_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_BUILDS_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_BUILDS_FIELD_NUMBER: _ClassVar[int]
    FAILED_BUILDS_FIELD_NUMBER: _ClassVar[int]
    total_builds: int
    active_builds: int
    completed_builds: int
    failed_builds: int
    def __init__(self, total_builds: _Optional[int] = ..., active_builds: _Optional[int] = ..., completed_builds: _Optional[int] = ..., failed_builds: _Optional[int] = ...) -> None: ...

class SystemStatusResponse(_message.Message):
    __slots__ = ("service", "statistics", "capabilities", "supported_frameworks", "supported_templates")
    SERVICE_FIELD_NUMBER: _ClassVar[int]
    STATISTICS_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    SUPPORTED_FRAMEWORKS_FIELD_NUMBER: _ClassVar[int]
    SUPPORTED_TEMPLATES_FIELD_NUMBER: _ClassVar[int]
    service: SystemServiceInfo
    statistics: SystemBuildStatistics
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    supported_frameworks: _containers.RepeatedScalarFieldContainer[str]
    supported_templates: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, service: _Optional[_Union[SystemServiceInfo, _Mapping]] = ..., statistics: _Optional[_Union[SystemBuildStatistics, _Mapping]] = ..., capabilities: _Optional[_Iterable[str]] = ..., supported_frameworks: _Optional[_Iterable[str]] = ..., supported_templates: _Optional[_Iterable[str]] = ...) -> None: ...

class ListTemplatesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class TemplateInfo(_message.Message):
    __slots__ = ("name", "description", "type", "framework", "use_cases", "features", "complexity", "examples")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    FRAMEWORK_FIELD_NUMBER: _ClassVar[int]
    USE_CASES_FIELD_NUMBER: _ClassVar[int]
    FEATURES_FIELD_NUMBER: _ClassVar[int]
    COMPLEXITY_FIELD_NUMBER: _ClassVar[int]
    EXAMPLES_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    type: str
    framework: str
    use_cases: _containers.RepeatedScalarFieldContainer[str]
    features: _containers.RepeatedScalarFieldContainer[str]
    complexity: str
    examples: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., type: _Optional[str] = ..., framework: _Optional[str] = ..., use_cases: _Optional[_Iterable[str]] = ..., features: _Optional[_Iterable[str]] = ..., complexity: _Optional[str] = ..., examples: _Optional[_Iterable[str]] = ...) -> None: ...

class ListTemplatesResponse(_message.Message):
    __slots__ = ("templates", "count")
    TEMPLATES_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    templates: _containers.RepeatedCompositeFieldContainer[TemplateInfo]
    count: int
    def __init__(self, templates: _Optional[_Iterable[_Union[TemplateInfo, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class GetTemplateRequest(_message.Message):
    __slots__ = ("type",)
    TYPE_FIELD_NUMBER: _ClassVar[int]
    type: str
    def __init__(self, type: _Optional[str] = ...) -> None: ...

class TemplateConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: _struct_pb2.Struct
    def __init__(self, config: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class CheckWineRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class WineInstallMethod(_message.Message):
    __slots__ = ("id", "name", "description", "requires_sudo", "estimated", "steps")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_SUDO_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    requires_sudo: bool
    estimated: str
    steps: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., requires_sudo: _Optional[bool] = ..., estimated: _Optional[str] = ..., steps: _Optional[_Iterable[str]] = ...) -> None: ...

class WineCheckResponse(_message.Message):
    __slots__ = ("installed", "version", "platform", "required_for", "install_methods", "recommended_method")
    INSTALLED_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FOR_FIELD_NUMBER: _ClassVar[int]
    INSTALL_METHODS_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_METHOD_FIELD_NUMBER: _ClassVar[int]
    installed: bool
    version: str
    platform: str
    required_for: _containers.RepeatedScalarFieldContainer[str]
    install_methods: _containers.RepeatedCompositeFieldContainer[WineInstallMethod]
    recommended_method: str
    def __init__(self, installed: _Optional[bool] = ..., version: _Optional[str] = ..., platform: _Optional[str] = ..., required_for: _Optional[_Iterable[str]] = ..., install_methods: _Optional[_Iterable[_Union[WineInstallMethod, _Mapping]]] = ..., recommended_method: _Optional[str] = ...) -> None: ...

class InstallWineRequest(_message.Message):
    __slots__ = ("method",)
    METHOD_FIELD_NUMBER: _ClassVar[int]
    method: str
    def __init__(self, method: _Optional[str] = ...) -> None: ...

class WineInstallResponse(_message.Message):
    __slots__ = ("install_id", "status", "method", "status_url")
    INSTALL_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    STATUS_URL_FIELD_NUMBER: _ClassVar[int]
    install_id: str
    status: str
    method: str
    status_url: str
    def __init__(self, install_id: _Optional[str] = ..., status: _Optional[str] = ..., method: _Optional[str] = ..., status_url: _Optional[str] = ...) -> None: ...

class GetWineInstallStatusRequest(_message.Message):
    __slots__ = ("install_id",)
    INSTALL_ID_FIELD_NUMBER: _ClassVar[int]
    install_id: str
    def __init__(self, install_id: _Optional[str] = ...) -> None: ...

class WineInstallStatusResponse(_message.Message):
    __slots__ = ("install_id", "status", "method", "started_at", "completed_at", "log", "error_log")
    INSTALL_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    LOG_FIELD_NUMBER: _ClassVar[int]
    ERROR_LOG_FIELD_NUMBER: _ClassVar[int]
    install_id: str
    status: str
    method: str
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    log: _containers.RepeatedScalarFieldContainer[str]
    error_log: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, install_id: _Optional[str] = ..., status: _Optional[str] = ..., method: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., log: _Optional[_Iterable[str]] = ..., error_log: _Optional[_Iterable[str]] = ...) -> None: ...
