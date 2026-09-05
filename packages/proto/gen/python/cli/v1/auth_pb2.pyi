from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AuthStatusResponse(_message.Message):
    __slots__ = ("success", "data")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    success: bool
    data: AuthStatusData
    def __init__(self, success: _Optional[bool] = ..., data: _Optional[_Union[AuthStatusData, _Mapping]] = ...) -> None: ...

class AuthStatusData(_message.Message):
    __slots__ = ("statuses",)
    STATUSES_FIELD_NUMBER: _ClassVar[int]
    statuses: _containers.RepeatedCompositeFieldContainer[AuthToolStatus]
    def __init__(self, statuses: _Optional[_Iterable[_Union[AuthToolStatus, _Mapping]]] = ...) -> None: ...

class AuthToolStatus(_message.Message):
    __slots__ = ("name", "result")
    NAME_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    name: str
    result: AuthProbeResult
    def __init__(self, name: _Optional[str] = ..., result: _Optional[_Union[AuthProbeResult, _Mapping]] = ...) -> None: ...

class AuthProbeResult(_message.Message):
    __slots__ = ("state", "detail", "sign_in_command")
    STATE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    SIGN_IN_COMMAND_FIELD_NUMBER: _ClassVar[int]
    state: str
    detail: str
    sign_in_command: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, state: _Optional[str] = ..., detail: _Optional[str] = ..., sign_in_command: _Optional[_Iterable[str]] = ...) -> None: ...
