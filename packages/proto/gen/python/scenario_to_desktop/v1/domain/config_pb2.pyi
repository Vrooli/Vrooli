from scenario_to_desktop.v1.base import shared_pb2 as _shared_pb2
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
    deployment_mode: _shared_pb2.DeploymentMode
    proxy_url: str
    external_server_url: str
    external_api_url: str
    def __init__(self, server_type: _Optional[str] = ..., port: _Optional[int] = ..., path: _Optional[str] = ..., api_endpoint: _Optional[str] = ..., scenario_path: _Optional[str] = ..., auto_manage_vrooli: _Optional[bool] = ..., vrooli_binary_path: _Optional[str] = ..., deployment_mode: _Optional[_Union[_shared_pb2.DeploymentMode, str]] = ..., proxy_url: _Optional[str] = ..., external_server_url: _Optional[str] = ..., external_api_url: _Optional[str] = ...) -> None: ...

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

class GitHubUpdateConfig(_message.Message):
    __slots__ = ("owner", "repo", "private")
    OWNER_FIELD_NUMBER: _ClassVar[int]
    REPO_FIELD_NUMBER: _ClassVar[int]
    PRIVATE_FIELD_NUMBER: _ClassVar[int]
    owner: str
    repo: str
    private: bool
    def __init__(self, owner: _Optional[str] = ..., repo: _Optional[str] = ..., private: _Optional[bool] = ...) -> None: ...

class GenericUpdateConfig(_message.Message):
    __slots__ = ("url", "channel_path")
    URL_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_PATH_FIELD_NUMBER: _ClassVar[int]
    url: str
    channel_path: str
    def __init__(self, url: _Optional[str] = ..., channel_path: _Optional[str] = ...) -> None: ...

class UpdateConfig(_message.Message):
    __slots__ = ("channel", "provider", "auto_check", "github", "generic")
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    AUTO_CHECK_FIELD_NUMBER: _ClassVar[int]
    GITHUB_FIELD_NUMBER: _ClassVar[int]
    GENERIC_FIELD_NUMBER: _ClassVar[int]
    channel: str
    provider: str
    auto_check: bool
    github: GitHubUpdateConfig
    generic: GenericUpdateConfig
    def __init__(self, channel: _Optional[str] = ..., provider: _Optional[str] = ..., auto_check: _Optional[bool] = ..., github: _Optional[_Union[GitHubUpdateConfig, _Mapping]] = ..., generic: _Optional[_Union[GenericUpdateConfig, _Mapping]] = ...) -> None: ...

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
    update: UpdateConfig
    window: WindowConfig
    framework: _shared_pb2.Framework
    template_type: _shared_pb2.TemplateType
    platforms: _containers.RepeatedScalarFieldContainer[_shared_pb2.Platform]
    output_path: str
    features: _containers.ScalarMap[str, bool]
    styling: _containers.ScalarMap[str, str]
    signing_enabled: bool
    def __init__(self, app: _Optional[_Union[AppIdentity, _Mapping]] = ..., server: _Optional[_Union[ServerConfig, _Mapping]] = ..., bundle: _Optional[_Union[BundleConfig, _Mapping]] = ..., update: _Optional[_Union[UpdateConfig, _Mapping]] = ..., window: _Optional[_Union[WindowConfig, _Mapping]] = ..., framework: _Optional[_Union[_shared_pb2.Framework, str]] = ..., template_type: _Optional[_Union[_shared_pb2.TemplateType, str]] = ..., platforms: _Optional[_Iterable[_Union[_shared_pb2.Platform, str]]] = ..., output_path: _Optional[str] = ..., features: _Optional[_Mapping[str, bool]] = ..., styling: _Optional[_Mapping[str, str]] = ..., signing_enabled: _Optional[bool] = ...) -> None: ...

class ScenarioMetadata(_message.Message):
    __slots__ = ("name", "display_name", "description", "version", "author", "license", "app_id", "has_ui", "ui_dist_path", "ui_port", "api_port", "scenario_path", "category", "tags", "service_json_path", "package_json_path")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    AUTHOR_FIELD_NUMBER: _ClassVar[int]
    LICENSE_FIELD_NUMBER: _ClassVar[int]
    APP_ID_FIELD_NUMBER: _ClassVar[int]
    HAS_UI_FIELD_NUMBER: _ClassVar[int]
    UI_DIST_PATH_FIELD_NUMBER: _ClassVar[int]
    UI_PORT_FIELD_NUMBER: _ClassVar[int]
    API_PORT_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_PATH_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    SERVICE_JSON_PATH_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_JSON_PATH_FIELD_NUMBER: _ClassVar[int]
    name: str
    display_name: str
    description: str
    version: str
    author: str
    license: str
    app_id: str
    has_ui: bool
    ui_dist_path: str
    ui_port: int
    api_port: int
    scenario_path: str
    category: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    service_json_path: str
    package_json_path: str
    def __init__(self, name: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., version: _Optional[str] = ..., author: _Optional[str] = ..., license: _Optional[str] = ..., app_id: _Optional[str] = ..., has_ui: _Optional[bool] = ..., ui_dist_path: _Optional[str] = ..., ui_port: _Optional[int] = ..., api_port: _Optional[int] = ..., scenario_path: _Optional[str] = ..., category: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., service_json_path: _Optional[str] = ..., package_json_path: _Optional[str] = ...) -> None: ...

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
    deployment_mode: _shared_pb2.DeploymentMode
    bundle_manifest_path: str
    app_display_name: str
    app_description: str
    icon: str
    def __init__(self, proxy_url: _Optional[str] = ..., server_type: _Optional[str] = ..., auto_manage_vrooli: _Optional[bool] = ..., vrooli_binary_path: _Optional[str] = ..., deployment_mode: _Optional[_Union[_shared_pb2.DeploymentMode, str]] = ..., bundle_manifest_path: _Optional[str] = ..., app_display_name: _Optional[str] = ..., app_description: _Optional[str] = ..., icon: _Optional[str] = ...) -> None: ...
