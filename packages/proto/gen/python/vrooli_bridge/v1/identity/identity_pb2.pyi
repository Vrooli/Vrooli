import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class LoginRequest(_message.Message):
    __slots__ = ("email", "password")
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_FIELD_NUMBER: _ClassVar[int]
    email: str
    password: str
    def __init__(self, email: _Optional[str] = ..., password: _Optional[str] = ...) -> None: ...

class LoginResponse(_message.Message):
    __slots__ = ("token", "email", "user_id", "refresh_token")
    TOKEN_FIELD_NUMBER: _ClassVar[int]
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    REFRESH_TOKEN_FIELD_NUMBER: _ClassVar[int]
    token: str
    email: str
    user_id: str
    refresh_token: str
    def __init__(self, token: _Optional[str] = ..., email: _Optional[str] = ..., user_id: _Optional[str] = ..., refresh_token: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("token", "email", "user_id", "refresh_token")
    TOKEN_FIELD_NUMBER: _ClassVar[int]
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    REFRESH_TOKEN_FIELD_NUMBER: _ClassVar[int]
    token: str
    email: str
    user_id: str
    refresh_token: str
    def __init__(self, token: _Optional[str] = ..., email: _Optional[str] = ..., user_id: _Optional[str] = ..., refresh_token: _Optional[str] = ...) -> None: ...

class RefreshRequest(_message.Message):
    __slots__ = ("refresh_token",)
    REFRESH_TOKEN_FIELD_NUMBER: _ClassVar[int]
    refresh_token: str
    def __init__(self, refresh_token: _Optional[str] = ...) -> None: ...

class RefreshResponse(_message.Message):
    __slots__ = ("token", "refresh_token")
    TOKEN_FIELD_NUMBER: _ClassVar[int]
    REFRESH_TOKEN_FIELD_NUMBER: _ClassVar[int]
    token: str
    refresh_token: str
    def __init__(self, token: _Optional[str] = ..., refresh_token: _Optional[str] = ...) -> None: ...

class EnrollOperatorSessionRequest(_message.Message):
    __slots__ = ("public_key", "mode", "requested_scopes")
    PUBLIC_KEY_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_SCOPES_FIELD_NUMBER: _ClassVar[int]
    public_key: bytes
    mode: str
    requested_scopes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, public_key: _Optional[bytes] = ..., mode: _Optional[str] = ..., requested_scopes: _Optional[_Iterable[str]] = ...) -> None: ...

class EnrollOperatorSessionResponse(_message.Message):
    __slots__ = ("enrollment_reference", "operator_id", "identity_provider", "mode", "scope_ceiling", "enrolled_at", "session_ttl_seconds")
    ENROLLMENT_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_ID_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_PROVIDER_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    SCOPE_CEILING_FIELD_NUMBER: _ClassVar[int]
    ENROLLED_AT_FIELD_NUMBER: _ClassVar[int]
    SESSION_TTL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    enrollment_reference: str
    operator_id: str
    identity_provider: str
    mode: str
    scope_ceiling: _containers.RepeatedScalarFieldContainer[str]
    enrolled_at: _timestamp_pb2.Timestamp
    session_ttl_seconds: int
    def __init__(self, enrollment_reference: _Optional[str] = ..., operator_id: _Optional[str] = ..., identity_provider: _Optional[str] = ..., mode: _Optional[str] = ..., scope_ceiling: _Optional[_Iterable[str]] = ..., enrolled_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., session_ttl_seconds: _Optional[int] = ...) -> None: ...
