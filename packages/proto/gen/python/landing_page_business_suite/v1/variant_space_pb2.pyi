from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class GetVariantSpaceRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetVariantSpaceResponse(_message.Message):
    __slots__ = ("raw_json",)
    RAW_JSON_FIELD_NUMBER: _ClassVar[int]
    raw_json: bytes
    def __init__(self, raw_json: _Optional[bytes] = ...) -> None: ...
