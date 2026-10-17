from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class LoginRequest(_message.Message):
    __slots__ = ("email", "password")
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_FIELD_NUMBER: _ClassVar[int]
    email: str
    password: str
    def __init__(self, email: _Optional[str] = ..., password: _Optional[str] = ...) -> None: ...

class AdminSessionResponse(_message.Message):
    __slots__ = ("email", "authenticated", "reset_enabled")
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    AUTHENTICATED_FIELD_NUMBER: _ClassVar[int]
    RESET_ENABLED_FIELD_NUMBER: _ClassVar[int]
    email: str
    authenticated: bool
    reset_enabled: bool
    def __init__(self, email: _Optional[str] = ..., authenticated: _Optional[bool] = ..., reset_enabled: _Optional[bool] = ...) -> None: ...

class LogoutRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class LogoutResponse(_message.Message):
    __slots__ = ("success",)
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    def __init__(self, success: _Optional[bool] = ...) -> None: ...

class SessionRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ResetDemoDataRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ResetDemoDataResponse(_message.Message):
    __slots__ = ("reset", "timestamp")
    RESET_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    reset: bool
    timestamp: str
    def __init__(self, reset: _Optional[bool] = ..., timestamp: _Optional[str] = ...) -> None: ...
