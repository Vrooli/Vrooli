from cli.v1 import common_pb2 as _common_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ResourceListResponse(_message.Message):
    __slots__ = ("success", "resources", "discovery_failures")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    DISCOVERY_FAILURES_FIELD_NUMBER: _ClassVar[int]
    success: bool
    resources: _containers.RepeatedCompositeFieldContainer[Resource]
    discovery_failures: _containers.RepeatedCompositeFieldContainer[_common_pb2.DiscoveryFailure]
    def __init__(self, success: _Optional[bool] = ..., resources: _Optional[_Iterable[_Union[Resource, _Mapping]]] = ..., discovery_failures: _Optional[_Iterable[_Union[_common_pb2.DiscoveryFailure, _Mapping]]] = ...) -> None: ...

class Resource(_message.Message):
    __slots__ = ("name", "path", "exists", "registered", "enabled", "required", "declares_cli", "cli_installed", "cli_state_reason", "config", "control_mode", "driver", "template", "portability_tier", "manifest_path")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    EXISTS_FIELD_NUMBER: _ClassVar[int]
    REGISTERED_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    DECLARES_CLI_FIELD_NUMBER: _ClassVar[int]
    CLI_INSTALLED_FIELD_NUMBER: _ClassVar[int]
    CLI_STATE_REASON_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    CONTROL_MODE_FIELD_NUMBER: _ClassVar[int]
    DRIVER_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    PORTABILITY_TIER_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_PATH_FIELD_NUMBER: _ClassVar[int]
    name: str
    path: str
    exists: bool
    registered: bool
    enabled: bool
    required: bool
    declares_cli: bool
    cli_installed: bool
    cli_state_reason: str
    config: ResourceConfig
    control_mode: str
    driver: str
    template: str
    portability_tier: str
    manifest_path: str
    def __init__(self, name: _Optional[str] = ..., path: _Optional[str] = ..., exists: _Optional[bool] = ..., registered: _Optional[bool] = ..., enabled: _Optional[bool] = ..., required: _Optional[bool] = ..., declares_cli: _Optional[bool] = ..., cli_installed: _Optional[bool] = ..., cli_state_reason: _Optional[str] = ..., config: _Optional[_Union[ResourceConfig, _Mapping]] = ..., control_mode: _Optional[str] = ..., driver: _Optional[str] = ..., template: _Optional[str] = ..., portability_tier: _Optional[str] = ..., manifest_path: _Optional[str] = ...) -> None: ...

class ResourceConfig(_message.Message):
    __slots__ = ("enabled", "required", "description")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    required: bool
    description: str
    def __init__(self, enabled: _Optional[bool] = ..., required: _Optional[bool] = ..., description: _Optional[str] = ...) -> None: ...
