from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ExpirationPolicy(_message.Message):
    __slots__ = ("mode", "duration")
    MODE_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    mode: str
    duration: str
    def __init__(self, mode: _Optional[str] = ..., duration: _Optional[str] = ...) -> None: ...

class SessionDefaults(_message.Message):
    __slots__ = ("default_backend", "default_policy")
    DEFAULT_BACKEND_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_POLICY_FIELD_NUMBER: _ClassVar[int]
    default_backend: str
    default_policy: ExpirationPolicy
    def __init__(self, default_backend: _Optional[str] = ..., default_policy: _Optional[_Union[ExpirationPolicy, _Mapping]] = ...) -> None: ...

class GetSessionDefaultsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetSessionDefaultsResponse(_message.Message):
    __slots__ = ("defaults",)
    DEFAULTS_FIELD_NUMBER: _ClassVar[int]
    defaults: SessionDefaults
    def __init__(self, defaults: _Optional[_Union[SessionDefaults, _Mapping]] = ...) -> None: ...

class UpdateSessionDefaultsRequest(_message.Message):
    __slots__ = ("default_backend", "default_policy")
    DEFAULT_BACKEND_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_POLICY_FIELD_NUMBER: _ClassVar[int]
    default_backend: str
    default_policy: ExpirationPolicy
    def __init__(self, default_backend: _Optional[str] = ..., default_policy: _Optional[_Union[ExpirationPolicy, _Mapping]] = ...) -> None: ...

class UpdateSessionDefaultsResponse(_message.Message):
    __slots__ = ("defaults",)
    DEFAULTS_FIELD_NUMBER: _ClassVar[int]
    defaults: SessionDefaults
    def __init__(self, defaults: _Optional[_Union[SessionDefaults, _Mapping]] = ...) -> None: ...
