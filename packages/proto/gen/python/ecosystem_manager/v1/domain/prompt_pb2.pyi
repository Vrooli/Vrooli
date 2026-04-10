from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class PromptFile(_message.Message):
    __slots__ = ("id", "path", "content", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    path: str
    content: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., path: _Optional[str] = ..., content: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class PromptFileInfo(_message.Message):
    __slots__ = ("id", "path", "size", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    path: str
    size: int
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., path: _Optional[str] = ..., size: _Optional[int] = ..., updated_at: _Optional[str] = ...) -> None: ...
