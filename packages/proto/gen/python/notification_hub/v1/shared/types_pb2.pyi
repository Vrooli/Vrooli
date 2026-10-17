from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class NotificationState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    NOTIFICATION_STATE_UNSPECIFIED: _ClassVar[NotificationState]
    NOTIFICATION_STATE_PENDING: _ClassVar[NotificationState]
    NOTIFICATION_STATE_HELD: _ClassVar[NotificationState]
    NOTIFICATION_STATE_ROUTED: _ClassVar[NotificationState]
    NOTIFICATION_STATE_DELIVERED: _ClassVar[NotificationState]
    NOTIFICATION_STATE_FAILED: _ClassVar[NotificationState]
    NOTIFICATION_STATE_UNROUTABLE: _ClassVar[NotificationState]
    NOTIFICATION_STATE_SUPPRESSED: _ClassVar[NotificationState]

class ChannelDisposition(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CHANNEL_DISPOSITION_UNSPECIFIED: _ClassVar[ChannelDisposition]
    CHANNEL_DISPOSITION_READY: _ClassVar[ChannelDisposition]
    CHANNEL_DISPOSITION_NOT_INSTALLED: _ClassVar[ChannelDisposition]
    CHANNEL_DISPOSITION_NOT_PERMITTED: _ClassVar[ChannelDisposition]
    CHANNEL_DISPOSITION_NOT_CONFIGURED: _ClassVar[ChannelDisposition]
    CHANNEL_DISPOSITION_DEGRADED: _ClassVar[ChannelDisposition]
    CHANNEL_DISPOSITION_UNKNOWN: _ClassVar[ChannelDisposition]
NOTIFICATION_STATE_UNSPECIFIED: NotificationState
NOTIFICATION_STATE_PENDING: NotificationState
NOTIFICATION_STATE_HELD: NotificationState
NOTIFICATION_STATE_ROUTED: NotificationState
NOTIFICATION_STATE_DELIVERED: NotificationState
NOTIFICATION_STATE_FAILED: NotificationState
NOTIFICATION_STATE_UNROUTABLE: NotificationState
NOTIFICATION_STATE_SUPPRESSED: NotificationState
CHANNEL_DISPOSITION_UNSPECIFIED: ChannelDisposition
CHANNEL_DISPOSITION_READY: ChannelDisposition
CHANNEL_DISPOSITION_NOT_INSTALLED: ChannelDisposition
CHANNEL_DISPOSITION_NOT_PERMITTED: ChannelDisposition
CHANNEL_DISPOSITION_NOT_CONFIGURED: ChannelDisposition
CHANNEL_DISPOSITION_DEGRADED: ChannelDisposition
CHANNEL_DISPOSITION_UNKNOWN: ChannelDisposition

class Notification(_message.Message):
    __slots__ = ("id", "requested_by", "body", "title", "urgency", "sensitivity_label", "idempotency_key", "dedupe_key", "state", "reason", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_BY_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    URGENCY_FIELD_NUMBER: _ClassVar[int]
    SENSITIVITY_LABEL_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    DEDUPE_KEY_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    requested_by: str
    body: str
    title: str
    urgency: str
    sensitivity_label: str
    idempotency_key: str
    dedupe_key: str
    state: NotificationState
    reason: str
    created_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., requested_by: _Optional[str] = ..., body: _Optional[str] = ..., title: _Optional[str] = ..., urgency: _Optional[str] = ..., sensitivity_label: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., dedupe_key: _Optional[str] = ..., state: _Optional[_Union[NotificationState, str]] = ..., reason: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class ChannelStatus(_message.Message):
    __slots__ = ("channel", "machine_id", "disposition", "reason", "observed_at")
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    channel: str
    machine_id: str
    disposition: ChannelDisposition
    reason: str
    observed_at: str
    def __init__(self, channel: _Optional[str] = ..., machine_id: _Optional[str] = ..., disposition: _Optional[_Union[ChannelDisposition, str]] = ..., reason: _Optional[str] = ..., observed_at: _Optional[str] = ...) -> None: ...

class DeliveryReceipt(_message.Message):
    __slots__ = ("id", "notification_id", "channel", "machine_id", "provider_id", "delivered_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NOTIFICATION_ID_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    DELIVERED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    notification_id: str
    channel: str
    machine_id: str
    provider_id: str
    delivered_at: str
    def __init__(self, id: _Optional[str] = ..., notification_id: _Optional[str] = ..., channel: _Optional[str] = ..., machine_id: _Optional[str] = ..., provider_id: _Optional[str] = ..., delivered_at: _Optional[str] = ...) -> None: ...
