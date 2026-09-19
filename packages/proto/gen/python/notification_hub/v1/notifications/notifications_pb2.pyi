from notification_hub.v1.shared import types_pb2 as _types_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SendRequest(_message.Message):
    __slots__ = ("title", "body", "urgency", "sensitivity_label", "idempotency_key", "dedupe_key", "dedupe_window_seconds", "scheduled_at", "digest_window_seconds")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    URGENCY_FIELD_NUMBER: _ClassVar[int]
    SENSITIVITY_LABEL_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    DEDUPE_KEY_FIELD_NUMBER: _ClassVar[int]
    DEDUPE_WINDOW_SECONDS_FIELD_NUMBER: _ClassVar[int]
    SCHEDULED_AT_FIELD_NUMBER: _ClassVar[int]
    DIGEST_WINDOW_SECONDS_FIELD_NUMBER: _ClassVar[int]
    title: str
    body: str
    urgency: str
    sensitivity_label: str
    idempotency_key: str
    dedupe_key: str
    dedupe_window_seconds: int
    scheduled_at: str
    digest_window_seconds: int
    def __init__(self, title: _Optional[str] = ..., body: _Optional[str] = ..., urgency: _Optional[str] = ..., sensitivity_label: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., dedupe_key: _Optional[str] = ..., dedupe_window_seconds: _Optional[int] = ..., scheduled_at: _Optional[str] = ..., digest_window_seconds: _Optional[int] = ...) -> None: ...

class SendResponse(_message.Message):
    __slots__ = ("notification",)
    NOTIFICATION_FIELD_NUMBER: _ClassVar[int]
    notification: _types_pb2.Notification
    def __init__(self, notification: _Optional[_Union[_types_pb2.Notification, _Mapping]] = ...) -> None: ...

class RelayRequest(_message.Message):
    __slots__ = ("payload_base64",)
    PAYLOAD_BASE64_FIELD_NUMBER: _ClassVar[int]
    payload_base64: str
    def __init__(self, payload_base64: _Optional[str] = ...) -> None: ...

class RelayResponse(_message.Message):
    __slots__ = ("notification",)
    NOTIFICATION_FIELD_NUMBER: _ClassVar[int]
    notification: _types_pb2.Notification
    def __init__(self, notification: _Optional[_Union[_types_pb2.Notification, _Mapping]] = ...) -> None: ...

class GetRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetResponse(_message.Message):
    __slots__ = ("notification", "receipts")
    NOTIFICATION_FIELD_NUMBER: _ClassVar[int]
    RECEIPTS_FIELD_NUMBER: _ClassVar[int]
    notification: _types_pb2.Notification
    receipts: _containers.RepeatedCompositeFieldContainer[_types_pb2.DeliveryReceipt]
    def __init__(self, notification: _Optional[_Union[_types_pb2.Notification, _Mapping]] = ..., receipts: _Optional[_Iterable[_Union[_types_pb2.DeliveryReceipt, _Mapping]]] = ...) -> None: ...

class ListRequest(_message.Message):
    __slots__ = ("limit",)
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    def __init__(self, limit: _Optional[int] = ...) -> None: ...

class ListResponse(_message.Message):
    __slots__ = ("notifications",)
    NOTIFICATIONS_FIELD_NUMBER: _ClassVar[int]
    notifications: _containers.RepeatedCompositeFieldContainer[_types_pb2.Notification]
    def __init__(self, notifications: _Optional[_Iterable[_Union[_types_pb2.Notification, _Mapping]]] = ...) -> None: ...
