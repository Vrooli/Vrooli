from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class GetAudioToolsEndpointRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetAudioToolsEndpointResponse(_message.Message):
    __slots__ = ("available", "base_url", "ws_base_url", "unavailable_reason")
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    BASE_URL_FIELD_NUMBER: _ClassVar[int]
    WS_BASE_URL_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_REASON_FIELD_NUMBER: _ClassVar[int]
    available: bool
    base_url: str
    ws_base_url: str
    unavailable_reason: str
    def __init__(self, available: _Optional[bool] = ..., base_url: _Optional[str] = ..., ws_base_url: _Optional[str] = ..., unavailable_reason: _Optional[str] = ...) -> None: ...
