from cli.v1 import common_pb2 as _common_pb2
from cli.v1 import resource_list_pb2 as _resource_list_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ResourceStatus(_message.Message):
    __slots__ = ("resource", "installed", "running", "healthy", "health", "status_code", "message", "probe_error", "raw")
    RESOURCE_FIELD_NUMBER: _ClassVar[int]
    INSTALLED_FIELD_NUMBER: _ClassVar[int]
    RUNNING_FIELD_NUMBER: _ClassVar[int]
    HEALTHY_FIELD_NUMBER: _ClassVar[int]
    HEALTH_FIELD_NUMBER: _ClassVar[int]
    STATUS_CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    PROBE_ERROR_FIELD_NUMBER: _ClassVar[int]
    RAW_FIELD_NUMBER: _ClassVar[int]
    resource: _resource_list_pb2.Resource
    installed: bool
    running: bool
    healthy: _struct_pb2.Value
    health: str
    status_code: str
    message: str
    probe_error: str
    raw: _struct_pb2.Value
    def __init__(self, resource: _Optional[_Union[_resource_list_pb2.Resource, _Mapping]] = ..., installed: _Optional[bool] = ..., running: _Optional[bool] = ..., healthy: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., health: _Optional[str] = ..., status_code: _Optional[str] = ..., message: _Optional[str] = ..., probe_error: _Optional[str] = ..., raw: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...

class ResourceStatusesResponse(_message.Message):
    __slots__ = ("success", "resources", "discovery_failures")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    DISCOVERY_FAILURES_FIELD_NUMBER: _ClassVar[int]
    success: bool
    resources: _containers.RepeatedCompositeFieldContainer[ResourceStatus]
    discovery_failures: _containers.RepeatedCompositeFieldContainer[_common_pb2.DiscoveryFailure]
    def __init__(self, success: _Optional[bool] = ..., resources: _Optional[_Iterable[_Union[ResourceStatus, _Mapping]]] = ..., discovery_failures: _Optional[_Iterable[_Union[_common_pb2.DiscoveryFailure, _Mapping]]] = ...) -> None: ...

class ResourceStatusResponse(_message.Message):
    __slots__ = ("success", "name", "installed", "running", "healthy", "status", "resource")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    INSTALLED_FIELD_NUMBER: _ClassVar[int]
    RUNNING_FIELD_NUMBER: _ClassVar[int]
    HEALTHY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_FIELD_NUMBER: _ClassVar[int]
    success: bool
    name: str
    installed: bool
    running: bool
    healthy: _struct_pb2.Value
    status: str
    resource: ResourceStatus
    def __init__(self, success: _Optional[bool] = ..., name: _Optional[str] = ..., installed: _Optional[bool] = ..., running: _Optional[bool] = ..., healthy: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., status: _Optional[str] = ..., resource: _Optional[_Union[ResourceStatus, _Mapping]] = ...) -> None: ...

class ResourceInfoResponse(_message.Message):
    __slots__ = ("success", "resource")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_FIELD_NUMBER: _ClassVar[int]
    success: bool
    resource: ResourceStatus
    def __init__(self, success: _Optional[bool] = ..., resource: _Optional[_Union[ResourceStatus, _Mapping]] = ...) -> None: ...
