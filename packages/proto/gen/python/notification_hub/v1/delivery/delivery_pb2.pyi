from notification_hub.v1.shared import types_pb2 as _types_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetTimelineRequest(_message.Message):
    __slots__ = ("limit",)
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    def __init__(self, limit: _Optional[int] = ...) -> None: ...

class GetTimelineResponse(_message.Message):
    __slots__ = ("notifications",)
    NOTIFICATIONS_FIELD_NUMBER: _ClassVar[int]
    notifications: _containers.RepeatedCompositeFieldContainer[_types_pb2.Notification]
    def __init__(self, notifications: _Optional[_Iterable[_Union[_types_pb2.Notification, _Mapping]]] = ...) -> None: ...

class DeliverRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeliverResponse(_message.Message):
    __slots__ = ("receipts",)
    RECEIPTS_FIELD_NUMBER: _ClassVar[int]
    receipts: _containers.RepeatedCompositeFieldContainer[_types_pb2.DeliveryReceipt]
    def __init__(self, receipts: _Optional[_Iterable[_Union[_types_pb2.DeliveryReceipt, _Mapping]]] = ...) -> None: ...

class GetAnalyticsRequest(_message.Message):
    __slots__ = ("since", "until")
    SINCE_FIELD_NUMBER: _ClassVar[int]
    UNTIL_FIELD_NUMBER: _ClassVar[int]
    since: str
    until: str
    def __init__(self, since: _Optional[str] = ..., until: _Optional[str] = ...) -> None: ...

class AnalyticsChannel(_message.Message):
    __slots__ = ("channel", "delivered", "failed", "attempts", "failure_rate", "average_latency_ms")
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    DELIVERED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    FAILURE_RATE_FIELD_NUMBER: _ClassVar[int]
    AVERAGE_LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    channel: str
    delivered: int
    failed: int
    attempts: int
    failure_rate: float
    average_latency_ms: float
    def __init__(self, channel: _Optional[str] = ..., delivered: _Optional[int] = ..., failed: _Optional[int] = ..., attempts: _Optional[int] = ..., failure_rate: _Optional[float] = ..., average_latency_ms: _Optional[float] = ...) -> None: ...

class GetAnalyticsResponse(_message.Message):
    __slots__ = ("since", "until", "channels", "total_notifications")
    SINCE_FIELD_NUMBER: _ClassVar[int]
    UNTIL_FIELD_NUMBER: _ClassVar[int]
    CHANNELS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_NOTIFICATIONS_FIELD_NUMBER: _ClassVar[int]
    since: str
    until: str
    channels: _containers.RepeatedCompositeFieldContainer[AnalyticsChannel]
    total_notifications: int
    def __init__(self, since: _Optional[str] = ..., until: _Optional[str] = ..., channels: _Optional[_Iterable[_Union[AnalyticsChannel, _Mapping]]] = ..., total_notifications: _Optional[int] = ...) -> None: ...
