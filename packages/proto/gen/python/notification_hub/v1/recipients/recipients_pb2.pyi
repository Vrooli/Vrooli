from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetRecipientRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class Recipient(_message.Message):
    __slots__ = ("id", "subject", "trust_posture", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    TRUST_POSTURE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    subject: str
    trust_posture: str
    created_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., subject: _Optional[str] = ..., trust_posture: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class RegisterPushSubscriptionRequest(_message.Message):
    __slots__ = ("endpoint", "p256dh", "auth", "origin")
    ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    P256DH_FIELD_NUMBER: _ClassVar[int]
    AUTH_FIELD_NUMBER: _ClassVar[int]
    ORIGIN_FIELD_NUMBER: _ClassVar[int]
    endpoint: str
    p256dh: str
    auth: str
    origin: str
    def __init__(self, endpoint: _Optional[str] = ..., p256dh: _Optional[str] = ..., auth: _Optional[str] = ..., origin: _Optional[str] = ...) -> None: ...

class RegisterPushSubscriptionResponse(_message.Message):
    __slots__ = ("subscription_id",)
    SUBSCRIPTION_ID_FIELD_NUMBER: _ClassVar[int]
    subscription_id: str
    def __init__(self, subscription_id: _Optional[str] = ...) -> None: ...

class RemovePushSubscriptionRequest(_message.Message):
    __slots__ = ("endpoint",)
    ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    endpoint: str
    def __init__(self, endpoint: _Optional[str] = ...) -> None: ...

class RemovePushSubscriptionResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListDevicesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class Device(_message.Message):
    __slots__ = ("id", "name", "machine_id", "channels")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    CHANNELS_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    machine_id: str
    channels: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., machine_id: _Optional[str] = ..., channels: _Optional[_Iterable[str]] = ...) -> None: ...

class ListDevicesResponse(_message.Message):
    __slots__ = ("devices", "vapid_public_key")
    DEVICES_FIELD_NUMBER: _ClassVar[int]
    VAPID_PUBLIC_KEY_FIELD_NUMBER: _ClassVar[int]
    devices: _containers.RepeatedCompositeFieldContainer[Device]
    vapid_public_key: str
    def __init__(self, devices: _Optional[_Iterable[_Union[Device, _Mapping]]] = ..., vapid_public_key: _Optional[str] = ...) -> None: ...

class UpsertDeviceRequest(_message.Message):
    __slots__ = ("id", "name", "machine_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    machine_id: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., machine_id: _Optional[str] = ...) -> None: ...

class RemoveDeviceRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RemoveDeviceResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ChannelAddress(_message.Message):
    __slots__ = ("id", "device_id", "channel", "address", "approved_labels")
    ID_FIELD_NUMBER: _ClassVar[int]
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    APPROVED_LABELS_FIELD_NUMBER: _ClassVar[int]
    id: str
    device_id: str
    channel: str
    address: str
    approved_labels: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., device_id: _Optional[str] = ..., channel: _Optional[str] = ..., address: _Optional[str] = ..., approved_labels: _Optional[_Iterable[str]] = ...) -> None: ...

class UpsertChannelAddressRequest(_message.Message):
    __slots__ = ("id", "device_id", "channel", "address", "approved_labels")
    ID_FIELD_NUMBER: _ClassVar[int]
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    APPROVED_LABELS_FIELD_NUMBER: _ClassVar[int]
    id: str
    device_id: str
    channel: str
    address: str
    approved_labels: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., device_id: _Optional[str] = ..., channel: _Optional[str] = ..., address: _Optional[str] = ..., approved_labels: _Optional[_Iterable[str]] = ...) -> None: ...

class RemoveChannelAddressRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RemoveChannelAddressResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SetQuietWindowRequest(_message.Message):
    __slots__ = ("weekday", "start", "end", "timezone", "critical_override")
    WEEKDAY_FIELD_NUMBER: _ClassVar[int]
    START_FIELD_NUMBER: _ClassVar[int]
    END_FIELD_NUMBER: _ClassVar[int]
    TIMEZONE_FIELD_NUMBER: _ClassVar[int]
    CRITICAL_OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    weekday: int
    start: str
    end: str
    timezone: str
    critical_override: bool
    def __init__(self, weekday: _Optional[int] = ..., start: _Optional[str] = ..., end: _Optional[str] = ..., timezone: _Optional[str] = ..., critical_override: _Optional[bool] = ...) -> None: ...

class QuietWindow(_message.Message):
    __slots__ = ("id", "weekday", "start", "end", "timezone", "critical_override")
    ID_FIELD_NUMBER: _ClassVar[int]
    WEEKDAY_FIELD_NUMBER: _ClassVar[int]
    START_FIELD_NUMBER: _ClassVar[int]
    END_FIELD_NUMBER: _ClassVar[int]
    TIMEZONE_FIELD_NUMBER: _ClassVar[int]
    CRITICAL_OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    id: str
    weekday: int
    start: str
    end: str
    timezone: str
    critical_override: bool
    def __init__(self, id: _Optional[str] = ..., weekday: _Optional[int] = ..., start: _Optional[str] = ..., end: _Optional[str] = ..., timezone: _Optional[str] = ..., critical_override: _Optional[bool] = ...) -> None: ...

class ListQuietWindowsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListQuietWindowsResponse(_message.Message):
    __slots__ = ("windows",)
    WINDOWS_FIELD_NUMBER: _ClassVar[int]
    windows: _containers.RepeatedCompositeFieldContainer[QuietWindow]
    def __init__(self, windows: _Optional[_Iterable[_Union[QuietWindow, _Mapping]]] = ...) -> None: ...

class DeleteQuietWindowRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteQuietWindowResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class EscalationStep(_message.Message):
    __slots__ = ("ordinal", "channel")
    ORDINAL_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    ordinal: int
    channel: str
    def __init__(self, ordinal: _Optional[int] = ..., channel: _Optional[str] = ...) -> None: ...

class EscalationChain(_message.Message):
    __slots__ = ("recipient_id", "steps")
    RECIPIENT_ID_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    recipient_id: str
    steps: _containers.RepeatedCompositeFieldContainer[EscalationStep]
    def __init__(self, recipient_id: _Optional[str] = ..., steps: _Optional[_Iterable[_Union[EscalationStep, _Mapping]]] = ...) -> None: ...

class SetEscalationChainRequest(_message.Message):
    __slots__ = ("channels",)
    CHANNELS_FIELD_NUMBER: _ClassVar[int]
    channels: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, channels: _Optional[_Iterable[str]] = ...) -> None: ...

class GetEscalationChainRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
