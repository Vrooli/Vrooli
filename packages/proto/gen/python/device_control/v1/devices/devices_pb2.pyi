import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListDevicesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListDevicesResponse(_message.Message):
    __slots__ = ("devices",)
    DEVICES_FIELD_NUMBER: _ClassVar[int]
    devices: _containers.RepeatedCompositeFieldContainer[Device]
    def __init__(self, devices: _Optional[_Iterable[_Union[Device, _Mapping]]] = ...) -> None: ...

class ConnectDeviceRequest(_message.Message):
    __slots__ = ("kind",)
    KIND_FIELD_NUMBER: _ClassVar[int]
    kind: str
    def __init__(self, kind: _Optional[str] = ...) -> None: ...

class ConnectDeviceResponse(_message.Message):
    __slots__ = ("rungs", "first_next_action")
    RUNGS_FIELD_NUMBER: _ClassVar[int]
    FIRST_NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    rungs: _containers.RepeatedCompositeFieldContainer[OnboardingRung]
    first_next_action: str
    def __init__(self, rungs: _Optional[_Iterable[_Union[OnboardingRung, _Mapping]]] = ..., first_next_action: _Optional[str] = ...) -> None: ...

class Device(_message.Message):
    __slots__ = ("id", "name", "kind", "strategy_id", "status", "health_reason", "host_node_id", "capabilities", "observed_at", "serial", "model", "os_version", "transport", "health", "first_seen_at", "last_seen_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    HEALTH_REASON_FIELD_NUMBER: _ClassVar[int]
    HOST_NODE_ID_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    SERIAL_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    OS_VERSION_FIELD_NUMBER: _ClassVar[int]
    TRANSPORT_FIELD_NUMBER: _ClassVar[int]
    HEALTH_FIELD_NUMBER: _ClassVar[int]
    FIRST_SEEN_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_SEEN_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    kind: str
    strategy_id: str
    status: str
    health_reason: str
    host_node_id: str
    capabilities: _containers.RepeatedCompositeFieldContainer[CapabilitySnapshot]
    observed_at: _timestamp_pb2.Timestamp
    serial: str
    model: str
    os_version: str
    transport: str
    health: str
    first_seen_at: str
    last_seen_at: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., kind: _Optional[str] = ..., strategy_id: _Optional[str] = ..., status: _Optional[str] = ..., health_reason: _Optional[str] = ..., host_node_id: _Optional[str] = ..., capabilities: _Optional[_Iterable[_Union[CapabilitySnapshot, _Mapping]]] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., serial: _Optional[str] = ..., model: _Optional[str] = ..., os_version: _Optional[str] = ..., transport: _Optional[str] = ..., health: _Optional[str] = ..., first_seen_at: _Optional[str] = ..., last_seen_at: _Optional[str] = ...) -> None: ...

class CapabilitySnapshot(_message.Message):
    __slots__ = ("name", "status", "prerequisite", "next_action")
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PREREQUISITE_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    name: str
    status: str
    prerequisite: str
    next_action: str
    def __init__(self, name: _Optional[str] = ..., status: _Optional[str] = ..., prerequisite: _Optional[str] = ..., next_action: _Optional[str] = ...) -> None: ...

class OnboardingRung(_message.Message):
    __slots__ = ("id", "prerequisite", "owner", "status", "next_action")
    ID_FIELD_NUMBER: _ClassVar[int]
    PREREQUISITE_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    id: str
    prerequisite: str
    owner: str
    status: str
    next_action: str
    def __init__(self, id: _Optional[str] = ..., prerequisite: _Optional[str] = ..., owner: _Optional[str] = ..., status: _Optional[str] = ..., next_action: _Optional[str] = ...) -> None: ...
