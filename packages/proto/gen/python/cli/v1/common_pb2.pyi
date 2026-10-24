from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class DiscoveryFailure(_message.Message):
    __slots__ = ("kind", "name", "path", "stage", "error")
    KIND_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    STAGE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    kind: str
    name: str
    path: str
    stage: str
    error: str
    def __init__(self, kind: _Optional[str] = ..., name: _Optional[str] = ..., path: _Optional[str] = ..., stage: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...
