import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Usage(_message.Message):
    __slots__ = ("instance_id", "tenant", "quantity", "started_at", "ended_at")
    INSTANCE_ID_FIELD_NUMBER: _ClassVar[int]
    TENANT_FIELD_NUMBER: _ClassVar[int]
    QUANTITY_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    ENDED_AT_FIELD_NUMBER: _ClassVar[int]
    instance_id: str
    tenant: str
    quantity: int
    started_at: _timestamp_pb2.Timestamp
    ended_at: _timestamp_pb2.Timestamp
    def __init__(self, instance_id: _Optional[str] = ..., tenant: _Optional[str] = ..., quantity: _Optional[int] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., ended_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Reservation(_message.Message):
    __slots__ = ("id", "intent_id", "instance_id", "supersedes", "meter_key", "state", "held_at", "settled_at", "quantity")
    ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    INSTANCE_ID_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDES_FIELD_NUMBER: _ClassVar[int]
    METER_KEY_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    HELD_AT_FIELD_NUMBER: _ClassVar[int]
    SETTLED_AT_FIELD_NUMBER: _ClassVar[int]
    QUANTITY_FIELD_NUMBER: _ClassVar[int]
    id: str
    intent_id: str
    instance_id: str
    supersedes: str
    meter_key: str
    state: str
    held_at: _timestamp_pb2.Timestamp
    settled_at: _timestamp_pb2.Timestamp
    quantity: int
    def __init__(self, id: _Optional[str] = ..., intent_id: _Optional[str] = ..., instance_id: _Optional[str] = ..., supersedes: _Optional[str] = ..., meter_key: _Optional[str] = ..., state: _Optional[str] = ..., held_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., settled_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., quantity: _Optional[int] = ...) -> None: ...

class UsageRequest(_message.Message):
    __slots__ = ("instance_id", "tenant")
    INSTANCE_ID_FIELD_NUMBER: _ClassVar[int]
    TENANT_FIELD_NUMBER: _ClassVar[int]
    instance_id: str
    tenant: str
    def __init__(self, instance_id: _Optional[str] = ..., tenant: _Optional[str] = ...) -> None: ...

class UsageResponse(_message.Message):
    __slots__ = ("usage",)
    USAGE_FIELD_NUMBER: _ClassVar[int]
    usage: _containers.RepeatedCompositeFieldContainer[Usage]
    def __init__(self, usage: _Optional[_Iterable[_Union[Usage, _Mapping]]] = ...) -> None: ...

class ReservationsRequest(_message.Message):
    __slots__ = ("instance_id", "state")
    INSTANCE_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    instance_id: str
    state: str
    def __init__(self, instance_id: _Optional[str] = ..., state: _Optional[str] = ...) -> None: ...

class ReservationsResponse(_message.Message):
    __slots__ = ("reservations",)
    RESERVATIONS_FIELD_NUMBER: _ClassVar[int]
    reservations: _containers.RepeatedCompositeFieldContainer[Reservation]
    def __init__(self, reservations: _Optional[_Iterable[_Union[Reservation, _Mapping]]] = ...) -> None: ...

class CeilingRequest(_message.Message):
    __slots__ = ("tenant",)
    TENANT_FIELD_NUMBER: _ClassVar[int]
    tenant: str
    def __init__(self, tenant: _Optional[str] = ...) -> None: ...

class CeilingResponse(_message.Message):
    __slots__ = ("used", "limit")
    USED_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    used: int
    limit: int
    def __init__(self, used: _Optional[int] = ..., limit: _Optional[int] = ...) -> None: ...
