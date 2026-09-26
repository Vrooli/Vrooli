from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import empty_pb2 as _empty_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ScenarioPortRequest(_message.Message):
    __slots__ = ("scenario_name", "port_name")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    PORT_NAME_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    port_name: str
    def __init__(self, scenario_name: _Optional[str] = ..., port_name: _Optional[str] = ...) -> None: ...

class ScenarioPortResponse(_message.Message):
    __slots__ = ("scenario_name", "port_name", "host", "port", "url")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    PORT_NAME_FIELD_NUMBER: _ClassVar[int]
    HOST_FIELD_NUMBER: _ClassVar[int]
    PORT_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    port_name: str
    host: str
    port: int
    url: str
    def __init__(self, scenario_name: _Optional[str] = ..., port_name: _Optional[str] = ..., host: _Optional[str] = ..., port: _Optional[int] = ..., url: _Optional[str] = ...) -> None: ...

class DesktopBuildArtifactStatus(_message.Message):
    __slots__ = ("platform", "file_name", "size_bytes", "modified_at", "absolute_path", "relative_path")
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    FILE_NAME_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    MODIFIED_AT_FIELD_NUMBER: _ClassVar[int]
    ABSOLUTE_PATH_FIELD_NUMBER: _ClassVar[int]
    RELATIVE_PATH_FIELD_NUMBER: _ClassVar[int]
    platform: str
    file_name: str
    size_bytes: int
    modified_at: str
    absolute_path: str
    relative_path: str
    def __init__(self, platform: _Optional[str] = ..., file_name: _Optional[str] = ..., size_bytes: _Optional[int] = ..., modified_at: _Optional[str] = ..., absolute_path: _Optional[str] = ..., relative_path: _Optional[str] = ...) -> None: ...

class DesktopConnectionStatus(_message.Message):
    __slots__ = ("mode", "endpoint")
    MODE_FIELD_NUMBER: _ClassVar[int]
    ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    mode: str
    endpoint: str
    def __init__(self, mode: _Optional[str] = ..., endpoint: _Optional[str] = ...) -> None: ...

class DesktopScenarioStatus(_message.Message):
    __slots__ = ("name", "display_name", "service_display_name", "service_description", "service_icon_path", "has_desktop", "desktop_path", "version", "platforms", "built", "dist_path", "last_modified", "package_size", "connection_config", "build_artifacts", "artifacts_source", "artifacts_path", "artifacts_expected_path", "record_id", "record_output_path", "record_location_mode", "record_updated_at")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    SERVICE_DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    SERVICE_DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    SERVICE_ICON_PATH_FIELD_NUMBER: _ClassVar[int]
    HAS_DESKTOP_FIELD_NUMBER: _ClassVar[int]
    DESKTOP_PATH_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    PLATFORMS_FIELD_NUMBER: _ClassVar[int]
    BUILT_FIELD_NUMBER: _ClassVar[int]
    DIST_PATH_FIELD_NUMBER: _ClassVar[int]
    LAST_MODIFIED_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    CONNECTION_CONFIG_FIELD_NUMBER: _ClassVar[int]
    BUILD_ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_SOURCE_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_PATH_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_EXPECTED_PATH_FIELD_NUMBER: _ClassVar[int]
    RECORD_ID_FIELD_NUMBER: _ClassVar[int]
    RECORD_OUTPUT_PATH_FIELD_NUMBER: _ClassVar[int]
    RECORD_LOCATION_MODE_FIELD_NUMBER: _ClassVar[int]
    RECORD_UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    name: str
    display_name: str
    service_display_name: str
    service_description: str
    service_icon_path: str
    has_desktop: bool
    desktop_path: str
    version: str
    platforms: _containers.RepeatedScalarFieldContainer[str]
    built: bool
    dist_path: str
    last_modified: str
    package_size: int
    connection_config: DesktopConnectionStatus
    build_artifacts: _containers.RepeatedCompositeFieldContainer[DesktopBuildArtifactStatus]
    artifacts_source: str
    artifacts_path: str
    artifacts_expected_path: str
    record_id: str
    record_output_path: str
    record_location_mode: str
    record_updated_at: str
    def __init__(self, name: _Optional[str] = ..., display_name: _Optional[str] = ..., service_display_name: _Optional[str] = ..., service_description: _Optional[str] = ..., service_icon_path: _Optional[str] = ..., has_desktop: _Optional[bool] = ..., desktop_path: _Optional[str] = ..., version: _Optional[str] = ..., platforms: _Optional[_Iterable[str]] = ..., built: _Optional[bool] = ..., dist_path: _Optional[str] = ..., last_modified: _Optional[str] = ..., package_size: _Optional[int] = ..., connection_config: _Optional[_Union[DesktopConnectionStatus, _Mapping]] = ..., build_artifacts: _Optional[_Iterable[_Union[DesktopBuildArtifactStatus, _Mapping]]] = ..., artifacts_source: _Optional[str] = ..., artifacts_path: _Optional[str] = ..., artifacts_expected_path: _Optional[str] = ..., record_id: _Optional[str] = ..., record_output_path: _Optional[str] = ..., record_location_mode: _Optional[str] = ..., record_updated_at: _Optional[str] = ...) -> None: ...

class DesktopScenarioStats(_message.Message):
    __slots__ = ("total", "with_desktop", "built", "web_only")
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    WITH_DESKTOP_FIELD_NUMBER: _ClassVar[int]
    BUILT_FIELD_NUMBER: _ClassVar[int]
    WEB_ONLY_FIELD_NUMBER: _ClassVar[int]
    total: int
    with_desktop: int
    built: int
    web_only: int
    def __init__(self, total: _Optional[int] = ..., with_desktop: _Optional[int] = ..., built: _Optional[int] = ..., web_only: _Optional[int] = ...) -> None: ...

class DesktopScenarioStatusResponse(_message.Message):
    __slots__ = ("scenarios", "stats")
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    STATS_FIELD_NUMBER: _ClassVar[int]
    scenarios: _containers.RepeatedCompositeFieldContainer[DesktopScenarioStatus]
    stats: DesktopScenarioStats
    def __init__(self, scenarios: _Optional[_Iterable[_Union[DesktopScenarioStatus, _Mapping]]] = ..., stats: _Optional[_Union[DesktopScenarioStats, _Mapping]] = ...) -> None: ...

class ProxyHintsRequest(_message.Message):
    __slots__ = ("scenario_name",)
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    def __init__(self, scenario_name: _Optional[str] = ...) -> None: ...

class ProxyHint(_message.Message):
    __slots__ = ("url", "source", "confidence", "message")
    URL_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    url: str
    source: str
    confidence: str
    message: str
    def __init__(self, url: _Optional[str] = ..., source: _Optional[str] = ..., confidence: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class ProxyHintsResponse(_message.Message):
    __slots__ = ("scenario_name", "hints")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    HINTS_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    hints: _containers.RepeatedCompositeFieldContainer[ProxyHint]
    def __init__(self, scenario_name: _Optional[str] = ..., hints: _Optional[_Iterable[_Union[ProxyHint, _Mapping]]] = ...) -> None: ...

class ProbeEndpointsRequest(_message.Message):
    __slots__ = ("proxy_url", "server_url", "api_url", "timeout_ms")
    PROXY_URL_FIELD_NUMBER: _ClassVar[int]
    SERVER_URL_FIELD_NUMBER: _ClassVar[int]
    API_URL_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    proxy_url: str
    server_url: str
    api_url: str
    timeout_ms: int
    def __init__(self, proxy_url: _Optional[str] = ..., server_url: _Optional[str] = ..., api_url: _Optional[str] = ..., timeout_ms: _Optional[int] = ...) -> None: ...

class ProbeEndpointResult(_message.Message):
    __slots__ = ("status", "status_code", "message")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STATUS_CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    status: str
    status_code: int
    message: str
    def __init__(self, status: _Optional[str] = ..., status_code: _Optional[int] = ..., message: _Optional[str] = ...) -> None: ...

class ProbeEndpointsResponse(_message.Message):
    __slots__ = ("proxy_url", "server", "api")
    PROXY_URL_FIELD_NUMBER: _ClassVar[int]
    SERVER_FIELD_NUMBER: _ClassVar[int]
    API_FIELD_NUMBER: _ClassVar[int]
    proxy_url: str
    server: ProbeEndpointResult
    api: ProbeEndpointResult
    def __init__(self, proxy_url: _Optional[str] = ..., server: _Optional[_Union[ProbeEndpointResult, _Mapping]] = ..., api: _Optional[_Union[ProbeEndpointResult, _Mapping]] = ...) -> None: ...
