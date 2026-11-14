from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class TrafficExclusions(_message.Message):
    __slots__ = ("bot_events", "internal_events")
    BOT_EVENTS_FIELD_NUMBER: _ClassVar[int]
    INTERNAL_EVENTS_FIELD_NUMBER: _ClassVar[int]
    bot_events: int
    internal_events: int
    def __init__(self, bot_events: _Optional[int] = ..., internal_events: _Optional[int] = ...) -> None: ...
