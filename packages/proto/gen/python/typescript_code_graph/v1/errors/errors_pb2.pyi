from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class ErrorEnvelope(_message.Message):
    __slots__ = ("code", "message", "details")
    class DetailsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    details: _containers.ScalarMap[str, str]
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., details: _Optional[_Mapping[str, str]] = ...) -> None: ...

class ExtractError(_message.Message):
    __slots__ = ("kind", "message", "file_path")
    KIND_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    kind: str
    message: str
    file_path: str
    def __init__(self, kind: _Optional[str] = ..., message: _Optional[str] = ..., file_path: _Optional[str] = ...) -> None: ...

class RewriteError(_message.Message):
    __slots__ = ("kind", "message", "plan_id")
    KIND_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    kind: str
    message: str
    plan_id: str
    def __init__(self, kind: _Optional[str] = ..., message: _Optional[str] = ..., plan_id: _Optional[str] = ...) -> None: ...

class SidecarError(_message.Message):
    __slots__ = ("kind", "message", "request_id")
    KIND_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    kind: str
    message: str
    request_id: str
    def __init__(self, kind: _Optional[str] = ..., message: _Optional[str] = ..., request_id: _Optional[str] = ...) -> None: ...
