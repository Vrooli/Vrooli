from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class Degradation(_message.Message):
    __slots__ = ("reason", "leg", "detail", "retryable")
    REASON_FIELD_NUMBER: _ClassVar[int]
    LEG_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    RETRYABLE_FIELD_NUMBER: _ClassVar[int]
    reason: str
    leg: str
    detail: str
    retryable: bool
    def __init__(self, reason: _Optional[str] = ..., leg: _Optional[str] = ..., detail: _Optional[str] = ..., retryable: _Optional[bool] = ...) -> None: ...
