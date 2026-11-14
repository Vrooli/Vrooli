from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Device(_message.Message):
    __slots__ = ("id", "hostname", "ip_address", "mac_address", "group", "identity_confidence", "notes")
    ID_FIELD_NUMBER: _ClassVar[int]
    HOSTNAME_FIELD_NUMBER: _ClassVar[int]
    IP_ADDRESS_FIELD_NUMBER: _ClassVar[int]
    MAC_ADDRESS_FIELD_NUMBER: _ClassVar[int]
    GROUP_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    id: str
    hostname: str
    ip_address: str
    mac_address: str
    group: str
    identity_confidence: str
    notes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., hostname: _Optional[str] = ..., ip_address: _Optional[str] = ..., mac_address: _Optional[str] = ..., group: _Optional[str] = ..., identity_confidence: _Optional[str] = ..., notes: _Optional[_Iterable[str]] = ...) -> None: ...

class RefreshInventoryRequest(_message.Message):
    __slots__ = ("dry_run",)
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    def __init__(self, dry_run: _Optional[bool] = ...) -> None: ...

class RefreshInventoryResponse(_message.Message):
    __slots__ = ("devices", "findings")
    DEVICES_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    devices: _containers.RepeatedCompositeFieldContainer[Device]
    findings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, devices: _Optional[_Iterable[_Union[Device, _Mapping]]] = ..., findings: _Optional[_Iterable[str]] = ...) -> None: ...

class ListDevicesRequest(_message.Message):
    __slots__ = ("group",)
    GROUP_FIELD_NUMBER: _ClassVar[int]
    group: str
    def __init__(self, group: _Optional[str] = ...) -> None: ...

class ListDevicesResponse(_message.Message):
    __slots__ = ("devices",)
    DEVICES_FIELD_NUMBER: _ClassVar[int]
    devices: _containers.RepeatedCompositeFieldContainer[Device]
    def __init__(self, devices: _Optional[_Iterable[_Union[Device, _Mapping]]] = ...) -> None: ...

class UpdateDeviceGroupRequest(_message.Message):
    __slots__ = ("id", "group")
    ID_FIELD_NUMBER: _ClassVar[int]
    GROUP_FIELD_NUMBER: _ClassVar[int]
    id: str
    group: str
    def __init__(self, id: _Optional[str] = ..., group: _Optional[str] = ...) -> None: ...

class UpdateDeviceGroupResponse(_message.Message):
    __slots__ = ("device",)
    DEVICE_FIELD_NUMBER: _ClassVar[int]
    device: Device
    def __init__(self, device: _Optional[_Union[Device, _Mapping]] = ...) -> None: ...

class ExplainDeviceIdentityRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ExplainDeviceIdentityResponse(_message.Message):
    __slots__ = ("device", "evidence")
    DEVICE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    device: Device
    evidence: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, device: _Optional[_Union[Device, _Mapping]] = ..., evidence: _Optional[_Iterable[str]] = ...) -> None: ...
