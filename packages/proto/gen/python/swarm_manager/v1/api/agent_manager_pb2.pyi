from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class AgentManagerStatusResponse(_message.Message):
    __slots__ = ("enabled", "available", "url", "profile_id")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    available: bool
    url: str
    profile_id: str
    def __init__(self, enabled: _Optional[bool] = ..., available: _Optional[bool] = ..., url: _Optional[str] = ..., profile_id: _Optional[str] = ...) -> None: ...
