from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class LoginRequest(_message.Message):
    __slots__ = ("email", "password", "mfa_challenge", "totp_code", "recovery_code")
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_FIELD_NUMBER: _ClassVar[int]
    MFA_CHALLENGE_FIELD_NUMBER: _ClassVar[int]
    TOTP_CODE_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_CODE_FIELD_NUMBER: _ClassVar[int]
    email: str
    password: str
    mfa_challenge: str
    totp_code: str
    recovery_code: str
    def __init__(self, email: _Optional[str] = ..., password: _Optional[str] = ..., mfa_challenge: _Optional[str] = ..., totp_code: _Optional[str] = ..., recovery_code: _Optional[str] = ...) -> None: ...

class LoginResponse(_message.Message):
    __slots__ = ("token", "refresh_token", "email", "user_id", "mfa_required", "mfa_challenge")
    TOKEN_FIELD_NUMBER: _ClassVar[int]
    REFRESH_TOKEN_FIELD_NUMBER: _ClassVar[int]
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    MFA_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    MFA_CHALLENGE_FIELD_NUMBER: _ClassVar[int]
    token: str
    refresh_token: str
    email: str
    user_id: str
    mfa_required: bool
    mfa_challenge: str
    def __init__(self, token: _Optional[str] = ..., refresh_token: _Optional[str] = ..., email: _Optional[str] = ..., user_id: _Optional[str] = ..., mfa_required: _Optional[bool] = ..., mfa_challenge: _Optional[str] = ...) -> None: ...

class GetSessionRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetSessionResponse(_message.Message):
    __slots__ = ("signed_in", "subject", "email")
    SIGNED_IN_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    signed_in: bool
    subject: str
    email: str
    def __init__(self, signed_in: _Optional[bool] = ..., subject: _Optional[str] = ..., email: _Optional[str] = ...) -> None: ...

class LogoutRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class LogoutResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
