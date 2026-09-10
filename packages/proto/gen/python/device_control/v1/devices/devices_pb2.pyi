import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import struct_pb2 as _struct_pb2
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

class ReconnectDeviceRequest(_message.Message):
    __slots__ = ("device_id",)
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    device_id: str
    def __init__(self, device_id: _Optional[str] = ...) -> None: ...

class ReconnectDeviceResponse(_message.Message):
    __slots__ = ("device",)
    DEVICE_FIELD_NUMBER: _ClassVar[int]
    device: Device
    def __init__(self, device: _Optional[_Union[Device, _Mapping]] = ...) -> None: ...

class ExecuteVolumeRequest(_message.Message):
    __slots__ = ("device", "actor", "goal", "operation", "value", "direction", "verification_policy", "operation_id")
    DEVICE_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    GOAL_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    DIRECTION_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_POLICY_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    device: str
    actor: str
    goal: str
    operation: str
    value: float
    direction: str
    verification_policy: str
    operation_id: str
    def __init__(self, device: _Optional[str] = ..., actor: _Optional[str] = ..., goal: _Optional[str] = ..., operation: _Optional[str] = ..., value: _Optional[float] = ..., direction: _Optional[str] = ..., verification_policy: _Optional[str] = ..., operation_id: _Optional[str] = ...) -> None: ...

class ExecuteVolumeResponse(_message.Message):
    __slots__ = ("status", "operation_id", "device_id", "device_name", "plan", "before", "after", "verification_class", "evidence", "recovery_attempts", "next_action")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    DEVICE_NAME_FIELD_NUMBER: _ClassVar[int]
    PLAN_FIELD_NUMBER: _ClassVar[int]
    BEFORE_FIELD_NUMBER: _ClassVar[int]
    AFTER_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_CLASS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    status: str
    operation_id: str
    device_id: str
    device_name: str
    plan: _struct_pb2.Struct
    before: _struct_pb2.Struct
    after: _struct_pb2.Struct
    verification_class: str
    evidence: _containers.RepeatedScalarFieldContainer[str]
    recovery_attempts: int
    next_action: str
    def __init__(self, status: _Optional[str] = ..., operation_id: _Optional[str] = ..., device_id: _Optional[str] = ..., device_name: _Optional[str] = ..., plan: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., before: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., after: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., verification_class: _Optional[str] = ..., evidence: _Optional[_Iterable[str]] = ..., recovery_attempts: _Optional[int] = ..., next_action: _Optional[str] = ...) -> None: ...

class Device(_message.Message):
    __slots__ = ("id", "name", "kind", "strategy_id", "status", "health_reason", "host_node_id", "capabilities", "observed_at", "serial", "model", "os_version", "transport", "health", "first_seen_at", "last_seen_at", "transports", "identity_key", "identity_reason")
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
    TRANSPORTS_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_KEY_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_REASON_FIELD_NUMBER: _ClassVar[int]
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
    transports: _containers.RepeatedCompositeFieldContainer[TransportProfile]
    identity_key: str
    identity_reason: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., kind: _Optional[str] = ..., strategy_id: _Optional[str] = ..., status: _Optional[str] = ..., health_reason: _Optional[str] = ..., host_node_id: _Optional[str] = ..., capabilities: _Optional[_Iterable[_Union[CapabilitySnapshot, _Mapping]]] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., serial: _Optional[str] = ..., model: _Optional[str] = ..., os_version: _Optional[str] = ..., transport: _Optional[str] = ..., health: _Optional[str] = ..., first_seen_at: _Optional[str] = ..., last_seen_at: _Optional[str] = ..., transports: _Optional[_Iterable[_Union[TransportProfile, _Mapping]]] = ..., identity_key: _Optional[str] = ..., identity_reason: _Optional[str] = ...) -> None: ...

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

class TransportProfile(_message.Message):
    __slots__ = ("strategy_id", "name", "role", "endpoint", "health", "health_reason", "capabilities", "operations", "endpoints")
    STRATEGY_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    HEALTH_FIELD_NUMBER: _ClassVar[int]
    HEALTH_REASON_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    ENDPOINTS_FIELD_NUMBER: _ClassVar[int]
    strategy_id: str
    name: str
    role: str
    endpoint: str
    health: str
    health_reason: str
    capabilities: _containers.RepeatedCompositeFieldContainer[CapabilitySnapshot]
    operations: _containers.RepeatedScalarFieldContainer[str]
    endpoints: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, strategy_id: _Optional[str] = ..., name: _Optional[str] = ..., role: _Optional[str] = ..., endpoint: _Optional[str] = ..., health: _Optional[str] = ..., health_reason: _Optional[str] = ..., capabilities: _Optional[_Iterable[_Union[CapabilitySnapshot, _Mapping]]] = ..., operations: _Optional[_Iterable[str]] = ..., endpoints: _Optional[_Iterable[str]] = ...) -> None: ...

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
