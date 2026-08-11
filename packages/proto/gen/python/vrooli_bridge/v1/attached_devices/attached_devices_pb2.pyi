import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PairAttachedDeviceRequest(_message.Message):
    __slots__ = ("name", "host_node_id", "kind", "transport", "serial", "os_version", "host_node_online")
    NAME_FIELD_NUMBER: _ClassVar[int]
    HOST_NODE_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    TRANSPORT_FIELD_NUMBER: _ClassVar[int]
    SERIAL_FIELD_NUMBER: _ClassVar[int]
    OS_VERSION_FIELD_NUMBER: _ClassVar[int]
    HOST_NODE_ONLINE_FIELD_NUMBER: _ClassVar[int]
    name: str
    host_node_id: str
    kind: str
    transport: str
    serial: str
    os_version: str
    host_node_online: bool
    def __init__(self, name: _Optional[str] = ..., host_node_id: _Optional[str] = ..., kind: _Optional[str] = ..., transport: _Optional[str] = ..., serial: _Optional[str] = ..., os_version: _Optional[str] = ..., host_node_online: _Optional[bool] = ...) -> None: ...

class ListAttachedDevicesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListAttachedDevicesResponse(_message.Message):
    __slots__ = ("devices",)
    DEVICES_FIELD_NUMBER: _ClassVar[int]
    devices: _containers.RepeatedCompositeFieldContainer[AttachedDevice]
    def __init__(self, devices: _Optional[_Iterable[_Union[AttachedDevice, _Mapping]]] = ...) -> None: ...

class RevokeAttachedDeviceRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class AttachedDeviceResponse(_message.Message):
    __slots__ = ("device",)
    DEVICE_FIELD_NUMBER: _ClassVar[int]
    device: AttachedDevice
    def __init__(self, device: _Optional[_Union[AttachedDevice, _Mapping]] = ...) -> None: ...

class AttachedDevice(_message.Message):
    __slots__ = ("id", "name", "host_node_id", "kind", "transport", "serial", "os_version", "trust_state", "reachability", "health_reason", "created_at", "revoked_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    HOST_NODE_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    TRANSPORT_FIELD_NUMBER: _ClassVar[int]
    SERIAL_FIELD_NUMBER: _ClassVar[int]
    OS_VERSION_FIELD_NUMBER: _ClassVar[int]
    TRUST_STATE_FIELD_NUMBER: _ClassVar[int]
    REACHABILITY_FIELD_NUMBER: _ClassVar[int]
    HEALTH_REASON_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    REVOKED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    host_node_id: str
    kind: str
    transport: str
    serial: str
    os_version: str
    trust_state: str
    reachability: str
    health_reason: str
    created_at: _timestamp_pb2.Timestamp
    revoked_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., host_node_id: _Optional[str] = ..., kind: _Optional[str] = ..., transport: _Optional[str] = ..., serial: _Optional[str] = ..., os_version: _Optional[str] = ..., trust_state: _Optional[str] = ..., reachability: _Optional[str] = ..., health_reason: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., revoked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
