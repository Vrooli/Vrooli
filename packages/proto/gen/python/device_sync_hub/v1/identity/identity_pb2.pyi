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

class LoginResponse(_message.Message):
    __slots__ = ("token", "refresh_token", "email", "user_id")
    TOKEN_FIELD_NUMBER: _ClassVar[int]
    REFRESH_TOKEN_FIELD_NUMBER: _ClassVar[int]
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    token: str
    refresh_token: str
    email: str
    user_id: str
    def __init__(self, token: _Optional[str] = ..., refresh_token: _Optional[str] = ..., email: _Optional[str] = ..., user_id: _Optional[str] = ...) -> None: ...

class RegisterRequest(_message.Message):
    __slots__ = ("email", "password", "username")
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_FIELD_NUMBER: _ClassVar[int]
    USERNAME_FIELD_NUMBER: _ClassVar[int]
    email: str
    password: str
    username: str
    def __init__(self, email: _Optional[str] = ..., password: _Optional[str] = ..., username: _Optional[str] = ...) -> None: ...

class RegisterResponse(_message.Message):
    __slots__ = ("token", "refresh_token", "email", "user_id")
    TOKEN_FIELD_NUMBER: _ClassVar[int]
    REFRESH_TOKEN_FIELD_NUMBER: _ClassVar[int]
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    token: str
    refresh_token: str
    email: str
    user_id: str
    def __init__(self, token: _Optional[str] = ..., refresh_token: _Optional[str] = ..., email: _Optional[str] = ..., user_id: _Optional[str] = ...) -> None: ...
