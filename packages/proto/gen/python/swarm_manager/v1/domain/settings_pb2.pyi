from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class Settings(_message.Message):
    __slots__ = ("theme",)
    THEME_FIELD_NUMBER: _ClassVar[int]
    theme: str
    def __init__(self, theme: _Optional[str] = ...) -> None: ...
