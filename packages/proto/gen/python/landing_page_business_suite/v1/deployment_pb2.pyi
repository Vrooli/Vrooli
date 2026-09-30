from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CheckDeploymentReadinessRequest(_message.Message):
    __slots__ = ("app_key", "remote_profile", "channel")
    APP_KEY_FIELD_NUMBER: _ClassVar[int]
    REMOTE_PROFILE_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    app_key: str
    remote_profile: str
    channel: str
    def __init__(self, app_key: _Optional[str] = ..., remote_profile: _Optional[str] = ..., channel: _Optional[str] = ...) -> None: ...

class DeploymentReadinessGate(_message.Message):
    __slots__ = ("name", "ready", "message")
    NAME_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    name: str
    ready: bool
    message: str
    def __init__(self, name: _Optional[str] = ..., ready: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...

class CheckDeploymentReadinessResponse(_message.Message):
    __slots__ = ("ready", "gates", "error")
    READY_FIELD_NUMBER: _ClassVar[int]
    GATES_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    ready: bool
    gates: _containers.RepeatedCompositeFieldContainer[DeploymentReadinessGate]
    error: str
    def __init__(self, ready: _Optional[bool] = ..., gates: _Optional[_Iterable[_Union[DeploymentReadinessGate, _Mapping]]] = ..., error: _Optional[str] = ...) -> None: ...
