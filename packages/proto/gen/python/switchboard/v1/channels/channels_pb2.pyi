from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListChannelsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetChannelRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ListBindingsRequest(_message.Message):
    __slots__ = ("agent_id",)
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    agent_id: str
    def __init__(self, agent_id: _Optional[str] = ...) -> None: ...

class CreateBindingRequest(_message.Message):
    __slots__ = ("agent_id", "channel_id", "address", "thread_key")
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_ID_FIELD_NUMBER: _ClassVar[int]
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    THREAD_KEY_FIELD_NUMBER: _ClassVar[int]
    agent_id: str
    channel_id: str
    address: str
    thread_key: str
    def __init__(self, agent_id: _Optional[str] = ..., channel_id: _Optional[str] = ..., address: _Optional[str] = ..., thread_key: _Optional[str] = ...) -> None: ...

class Channel(_message.Message):
    __slots__ = ("id", "display_name", "availability", "reason", "friction")
    ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    FRICTION_FIELD_NUMBER: _ClassVar[int]
    id: str
    display_name: str
    availability: str
    reason: str
    friction: int
    def __init__(self, id: _Optional[str] = ..., display_name: _Optional[str] = ..., availability: _Optional[str] = ..., reason: _Optional[str] = ..., friction: _Optional[int] = ...) -> None: ...

class Binding(_message.Message):
    __slots__ = ("id", "agent_id", "channel_id", "address", "thread_key")
    ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_ID_FIELD_NUMBER: _ClassVar[int]
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    THREAD_KEY_FIELD_NUMBER: _ClassVar[int]
    id: str
    agent_id: str
    channel_id: str
    address: str
    thread_key: str
    def __init__(self, id: _Optional[str] = ..., agent_id: _Optional[str] = ..., channel_id: _Optional[str] = ..., address: _Optional[str] = ..., thread_key: _Optional[str] = ...) -> None: ...

class ListChannelsResponse(_message.Message):
    __slots__ = ("channels",)
    CHANNELS_FIELD_NUMBER: _ClassVar[int]
    channels: _containers.RepeatedCompositeFieldContainer[Channel]
    def __init__(self, channels: _Optional[_Iterable[_Union[Channel, _Mapping]]] = ...) -> None: ...

class GetChannelResponse(_message.Message):
    __slots__ = ("channel",)
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    channel: Channel
    def __init__(self, channel: _Optional[_Union[Channel, _Mapping]] = ...) -> None: ...

class ListBindingsResponse(_message.Message):
    __slots__ = ("bindings",)
    BINDINGS_FIELD_NUMBER: _ClassVar[int]
    bindings: _containers.RepeatedCompositeFieldContainer[Binding]
    def __init__(self, bindings: _Optional[_Iterable[_Union[Binding, _Mapping]]] = ...) -> None: ...

class CreateBindingResponse(_message.Message):
    __slots__ = ("binding",)
    BINDING_FIELD_NUMBER: _ClassVar[int]
    binding: Binding
    def __init__(self, binding: _Optional[_Union[Binding, _Mapping]] = ...) -> None: ...

class SendMessageRequest(_message.Message):
    __slots__ = ("channel_id", "thread_key", "text", "media", "reply_to_remote_id")
    CHANNEL_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_KEY_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    MEDIA_FIELD_NUMBER: _ClassVar[int]
    REPLY_TO_REMOTE_ID_FIELD_NUMBER: _ClassVar[int]
    channel_id: str
    thread_key: str
    text: str
    media: _containers.RepeatedCompositeFieldContainer[Media]
    reply_to_remote_id: str
    def __init__(self, channel_id: _Optional[str] = ..., thread_key: _Optional[str] = ..., text: _Optional[str] = ..., media: _Optional[_Iterable[_Union[Media, _Mapping]]] = ..., reply_to_remote_id: _Optional[str] = ...) -> None: ...

class SendMessageResponse(_message.Message):
    __slots__ = ("accepted",)
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    accepted: bool
    def __init__(self, accepted: _Optional[bool] = ...) -> None: ...

class Media(_message.Message):
    __slots__ = ("name", "mime", "url", "size")
    NAME_FIELD_NUMBER: _ClassVar[int]
    MIME_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    name: str
    mime: str
    url: str
    size: int
    def __init__(self, name: _Optional[str] = ..., mime: _Optional[str] = ..., url: _Optional[str] = ..., size: _Optional[int] = ...) -> None: ...
