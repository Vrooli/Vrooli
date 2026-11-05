from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class Confidence(_message.Message):
    __slots__ = ("weak", "regime")
    WEAK_FIELD_NUMBER: _ClassVar[int]
    REGIME_FIELD_NUMBER: _ClassVar[int]
    weak: bool
    regime: str
    def __init__(self, weak: _Optional[bool] = ..., regime: _Optional[str] = ...) -> None: ...
