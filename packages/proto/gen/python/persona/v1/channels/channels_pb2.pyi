import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ChannelKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CHANNEL_KIND_UNSPECIFIED: _ClassVar[ChannelKind]
    CHANNEL_KIND_EMAIL: _ClassVar[ChannelKind]
    CHANNEL_KIND_SMS: _ClassVar[ChannelKind]
    CHANNEL_KIND_DEVICE: _ClassVar[ChannelKind]
CHANNEL_KIND_UNSPECIFIED: ChannelKind
CHANNEL_KIND_EMAIL: ChannelKind
CHANNEL_KIND_SMS: ChannelKind
CHANNEL_KIND_DEVICE: ChannelKind

class Channel(_message.Message):
    __slots__ = ("id", "persona_id", "kind", "address", "credential_ref", "adapter", "enabled", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_REF_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    persona_id: str
    kind: ChannelKind
    address: str
    credential_ref: str
    adapter: str
    enabled: bool
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., persona_id: _Optional[str] = ..., kind: _Optional[_Union[ChannelKind, str]] = ..., address: _Optional[str] = ..., credential_ref: _Optional[str] = ..., adapter: _Optional[str] = ..., enabled: _Optional[bool] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class BindChannelRequest(_message.Message):
    __slots__ = ("persona_id", "kind", "address", "credential_ref", "adapter")
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_REF_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    kind: ChannelKind
    address: str
    credential_ref: str
    adapter: str
    def __init__(self, persona_id: _Optional[str] = ..., kind: _Optional[_Union[ChannelKind, str]] = ..., address: _Optional[str] = ..., credential_ref: _Optional[str] = ..., adapter: _Optional[str] = ...) -> None: ...

class BindChannelResponse(_message.Message):
    __slots__ = ("channel",)
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    channel: Channel
    def __init__(self, channel: _Optional[_Union[Channel, _Mapping]] = ...) -> None: ...

class ListChannelsRequest(_message.Message):
    __slots__ = ("persona_id",)
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    def __init__(self, persona_id: _Optional[str] = ...) -> None: ...

class ListChannelsResponse(_message.Message):
    __slots__ = ("channels",)
    CHANNELS_FIELD_NUMBER: _ClassVar[int]
    channels: _containers.RepeatedCompositeFieldContainer[Channel]
    def __init__(self, channels: _Optional[_Iterable[_Union[Channel, _Mapping]]] = ...) -> None: ...

class SendMessageRequest(_message.Message):
    __slots__ = ("persona_id", "channel_id", "recipient", "subject", "body")
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_ID_FIELD_NUMBER: _ClassVar[int]
    RECIPIENT_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    channel_id: str
    recipient: str
    subject: str
    body: str
    def __init__(self, persona_id: _Optional[str] = ..., channel_id: _Optional[str] = ..., recipient: _Optional[str] = ..., subject: _Optional[str] = ..., body: _Optional[str] = ...) -> None: ...

class SendMessageResponse(_message.Message):
    __slots__ = ("channel_id", "message_id", "from_address")
    CHANNEL_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    FROM_ADDRESS_FIELD_NUMBER: _ClassVar[int]
    channel_id: str
    message_id: str
    from_address: str
    def __init__(self, channel_id: _Optional[str] = ..., message_id: _Optional[str] = ..., from_address: _Optional[str] = ...) -> None: ...

class RetrieveCodeRequest(_message.Message):
    __slots__ = ("persona_id", "channel_id", "purpose")
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_ID_FIELD_NUMBER: _ClassVar[int]
    PURPOSE_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    channel_id: str
    purpose: str
    def __init__(self, persona_id: _Optional[str] = ..., channel_id: _Optional[str] = ..., purpose: _Optional[str] = ...) -> None: ...

class RetrieveCodeResponse(_message.Message):
    __slots__ = ("channel_id", "code", "expires_at", "adapter")
    CHANNEL_ID_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_FIELD_NUMBER: _ClassVar[int]
    channel_id: str
    code: str
    expires_at: _timestamp_pb2.Timestamp
    adapter: str
    def __init__(self, channel_id: _Optional[str] = ..., code: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., adapter: _Optional[str] = ...) -> None: ...
