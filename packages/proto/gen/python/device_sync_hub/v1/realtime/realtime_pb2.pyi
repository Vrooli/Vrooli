import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EventType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EVENT_TYPE_UNSPECIFIED: _ClassVar[EventType]
    EVENT_TYPE_ITEM_ARRIVED: _ClassVar[EventType]
    EVENT_TYPE_ITEM_DELETED: _ClassVar[EventType]
    EVENT_TYPE_PRESENCE_CHANGED: _ClassVar[EventType]
    EVENT_TYPE_PAIRING_REQUESTED: _ClassVar[EventType]
EVENT_TYPE_UNSPECIFIED: EventType
EVENT_TYPE_ITEM_ARRIVED: EventType
EVENT_TYPE_ITEM_DELETED: EventType
EVENT_TYPE_PRESENCE_CHANGED: EventType
EVENT_TYPE_PAIRING_REQUESTED: EventType

class DevicePresence(_message.Message):
    __slots__ = ("device_id", "online")
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    ONLINE_FIELD_NUMBER: _ClassVar[int]
    device_id: str
    online: bool
    def __init__(self, device_id: _Optional[str] = ..., online: _Optional[bool] = ...) -> None: ...

class ItemRef(_message.Message):
    __slots__ = ("id", "target_device_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    target_device_id: str
    def __init__(self, id: _Optional[str] = ..., target_device_id: _Optional[str] = ...) -> None: ...

class PairingRequest(_message.Message):
    __slots__ = ("device_id", "name", "kind")
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    device_id: str
    name: str
    kind: str
    def __init__(self, device_id: _Optional[str] = ..., name: _Optional[str] = ..., kind: _Optional[str] = ...) -> None: ...

class Event(_message.Message):
    __slots__ = ("type", "at", "item", "presence", "pairing")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    AT_FIELD_NUMBER: _ClassVar[int]
    ITEM_FIELD_NUMBER: _ClassVar[int]
    PRESENCE_FIELD_NUMBER: _ClassVar[int]
    PAIRING_FIELD_NUMBER: _ClassVar[int]
    type: EventType
    at: _timestamp_pb2.Timestamp
    item: ItemRef
    presence: _containers.RepeatedCompositeFieldContainer[DevicePresence]
    pairing: PairingRequest
    def __init__(self, type: _Optional[_Union[EventType, str]] = ..., at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., item: _Optional[_Union[ItemRef, _Mapping]] = ..., presence: _Optional[_Iterable[_Union[DevicePresence, _Mapping]]] = ..., pairing: _Optional[_Union[PairingRequest, _Mapping]] = ...) -> None: ...
