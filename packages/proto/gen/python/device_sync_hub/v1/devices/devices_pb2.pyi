import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TrustState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TRUST_STATE_UNSPECIFIED: _ClassVar[TrustState]
    TRUST_STATE_PENDING: _ClassVar[TrustState]
    TRUST_STATE_TRUSTED: _ClassVar[TrustState]
    TRUST_STATE_REVOKED: _ClassVar[TrustState]
TRUST_STATE_UNSPECIFIED: TrustState
TRUST_STATE_PENDING: TrustState
TRUST_STATE_TRUSTED: TrustState
TRUST_STATE_REVOKED: TrustState

class Device(_message.Message):
    __slots__ = ("id", "owner_id", "name", "kind", "platform", "capabilities", "trust_state", "online", "last_seen_at", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    TRUST_STATE_FIELD_NUMBER: _ClassVar[int]
    ONLINE_FIELD_NUMBER: _ClassVar[int]
    LAST_SEEN_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    owner_id: str
    name: str
    kind: str
    platform: str
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    trust_state: TrustState
    online: bool
    last_seen_at: _timestamp_pb2.Timestamp
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., owner_id: _Optional[str] = ..., name: _Optional[str] = ..., kind: _Optional[str] = ..., platform: _Optional[str] = ..., capabilities: _Optional[_Iterable[str]] = ..., trust_state: _Optional[_Union[TrustState, str]] = ..., online: _Optional[bool] = ..., last_seen_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class PairingCode(_message.Message):
    __slots__ = ("code", "owner_id", "expires_at", "created_at")
    CODE_FIELD_NUMBER: _ClassVar[int]
    OWNER_ID_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    code: str
    owner_id: str
    expires_at: _timestamp_pb2.Timestamp
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, code: _Optional[str] = ..., owner_id: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListDevicesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListDevicesResponse(_message.Message):
    __slots__ = ("devices",)
    DEVICES_FIELD_NUMBER: _ClassVar[int]
    devices: _containers.RepeatedCompositeFieldContainer[Device]
    def __init__(self, devices: _Optional[_Iterable[_Union[Device, _Mapping]]] = ...) -> None: ...

class GetDeviceRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetDeviceResponse(_message.Message):
    __slots__ = ("device",)
    DEVICE_FIELD_NUMBER: _ClassVar[int]
    device: Device
    def __init__(self, device: _Optional[_Union[Device, _Mapping]] = ...) -> None: ...

class IssuePairingCodeRequest(_message.Message):
    __slots__ = ("device_name",)
    DEVICE_NAME_FIELD_NUMBER: _ClassVar[int]
    device_name: str
    def __init__(self, device_name: _Optional[str] = ...) -> None: ...

class IssuePairingCodeResponse(_message.Message):
    __slots__ = ("pairing_code",)
    PAIRING_CODE_FIELD_NUMBER: _ClassVar[int]
    pairing_code: PairingCode
    def __init__(self, pairing_code: _Optional[_Union[PairingCode, _Mapping]] = ...) -> None: ...

class DeviceProfile(_message.Message):
    __slots__ = ("device_name", "kind", "platform", "capabilities")
    DEVICE_NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    device_name: str
    kind: str
    platform: str
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, device_name: _Optional[str] = ..., kind: _Optional[str] = ..., platform: _Optional[str] = ..., capabilities: _Optional[_Iterable[str]] = ...) -> None: ...

class SetupOwnerDeviceRequest(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: DeviceProfile
    def __init__(self, profile: _Optional[_Union[DeviceProfile, _Mapping]] = ...) -> None: ...

class SetupOwnerDeviceResponse(_message.Message):
    __slots__ = ("device", "device_token")
    DEVICE_FIELD_NUMBER: _ClassVar[int]
    DEVICE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    device: Device
    device_token: str
    def __init__(self, device: _Optional[_Union[Device, _Mapping]] = ..., device_token: _Optional[str] = ...) -> None: ...

class RedeemPairingCodeRequest(_message.Message):
    __slots__ = ("code", "profile")
    CODE_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    code: str
    profile: DeviceProfile
    def __init__(self, code: _Optional[str] = ..., profile: _Optional[_Union[DeviceProfile, _Mapping]] = ...) -> None: ...

class RedeemPairingCodeResponse(_message.Message):
    __slots__ = ("device", "device_token")
    DEVICE_FIELD_NUMBER: _ClassVar[int]
    DEVICE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    device: Device
    device_token: str
    def __init__(self, device: _Optional[_Union[Device, _Mapping]] = ..., device_token: _Optional[str] = ...) -> None: ...

class RequestPairingRequest(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: DeviceProfile
    def __init__(self, profile: _Optional[_Union[DeviceProfile, _Mapping]] = ...) -> None: ...

class RequestPairingResponse(_message.Message):
    __slots__ = ("device", "device_token")
    DEVICE_FIELD_NUMBER: _ClassVar[int]
    DEVICE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    device: Device
    device_token: str
    def __init__(self, device: _Optional[_Union[Device, _Mapping]] = ..., device_token: _Optional[str] = ...) -> None: ...

class ApprovePairingRequest(_message.Message):
    __slots__ = ("device_id",)
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    device_id: str
    def __init__(self, device_id: _Optional[str] = ...) -> None: ...

class ApprovePairingResponse(_message.Message):
    __slots__ = ("device",)
    DEVICE_FIELD_NUMBER: _ClassVar[int]
    device: Device
    def __init__(self, device: _Optional[_Union[Device, _Mapping]] = ...) -> None: ...

class RenameDeviceRequest(_message.Message):
    __slots__ = ("device_id", "name")
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    device_id: str
    name: str
    def __init__(self, device_id: _Optional[str] = ..., name: _Optional[str] = ...) -> None: ...

class RenameDeviceResponse(_message.Message):
    __slots__ = ("device",)
    DEVICE_FIELD_NUMBER: _ClassVar[int]
    device: Device
    def __init__(self, device: _Optional[_Union[Device, _Mapping]] = ...) -> None: ...

class RevokeDeviceRequest(_message.Message):
    __slots__ = ("device_id",)
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    device_id: str
    def __init__(self, device_id: _Optional[str] = ...) -> None: ...

class RevokeDeviceResponse(_message.Message):
    __slots__ = ("device",)
    DEVICE_FIELD_NUMBER: _ClassVar[int]
    device: Device
    def __init__(self, device: _Optional[_Union[Device, _Mapping]] = ...) -> None: ...
