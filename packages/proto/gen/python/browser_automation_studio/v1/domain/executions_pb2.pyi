from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class Executions(_message.Message):
    __slots__ = ("id", "status", "started_at", "completed_at", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    status: str
    started_at: str
    completed_at: str
    created_at: str
    def __init__(self, id: _Optional[str] = ..., status: _Optional[str] = ..., started_at: _Optional[str] = ..., completed_at: _Optional[str] = ..., created_at: _Optional[str] = ...) -> None: ...
