from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class WriteRefusal(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WRITE_REFUSAL_UNSPECIFIED: _ClassVar[WriteRefusal]
    WRITE_REFUSAL_SIGNED_OUT: _ClassVar[WriteRefusal]
    WRITE_REFUSAL_UNVERIFIED: _ClassVar[WriteRefusal]
    WRITE_REFUSAL_NOT_OWNER: _ClassVar[WriteRefusal]
    WRITE_REFUSAL_AGENT: _ClassVar[WriteRefusal]
    WRITE_REFUSAL_UNDECLARED_SERVICE: _ClassVar[WriteRefusal]
    WRITE_REFUSAL_CONFLICT: _ClassVar[WriteRefusal]
WRITE_REFUSAL_UNSPECIFIED: WriteRefusal
WRITE_REFUSAL_SIGNED_OUT: WriteRefusal
WRITE_REFUSAL_UNVERIFIED: WriteRefusal
WRITE_REFUSAL_NOT_OWNER: WriteRefusal
WRITE_REFUSAL_AGENT: WriteRefusal
WRITE_REFUSAL_UNDECLARED_SERVICE: WriteRefusal
WRITE_REFUSAL_CONFLICT: WriteRefusal

class GetWriteAccessRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetWriteAccessResponse(_message.Message):
    __slots__ = ("can_write", "actor", "refusal", "reason", "signed_in_as", "browser_session")
    CAN_WRITE_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    REFUSAL_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    SIGNED_IN_AS_FIELD_NUMBER: _ClassVar[int]
    BROWSER_SESSION_FIELD_NUMBER: _ClassVar[int]
    can_write: bool
    actor: str
    refusal: WriteRefusal
    reason: str
    signed_in_as: str
    browser_session: bool
    def __init__(self, can_write: _Optional[bool] = ..., actor: _Optional[str] = ..., refusal: _Optional[_Union[WriteRefusal, str]] = ..., reason: _Optional[str] = ..., signed_in_as: _Optional[str] = ..., browser_session: _Optional[bool] = ...) -> None: ...

class SignInRequest(_message.Message):
    __slots__ = ("email", "password", "totp_code", "recovery_code", "mfa_challenge")
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_FIELD_NUMBER: _ClassVar[int]
    TOTP_CODE_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_CODE_FIELD_NUMBER: _ClassVar[int]
    MFA_CHALLENGE_FIELD_NUMBER: _ClassVar[int]
    email: str
    password: str
    totp_code: str
    recovery_code: str
    mfa_challenge: str
    def __init__(self, email: _Optional[str] = ..., password: _Optional[str] = ..., totp_code: _Optional[str] = ..., recovery_code: _Optional[str] = ..., mfa_challenge: _Optional[str] = ...) -> None: ...

class SignInResponse(_message.Message):
    __slots__ = ("mfa_required", "mfa_challenge", "access")
    MFA_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    MFA_CHALLENGE_FIELD_NUMBER: _ClassVar[int]
    ACCESS_FIELD_NUMBER: _ClassVar[int]
    mfa_required: bool
    mfa_challenge: str
    access: GetWriteAccessResponse
    def __init__(self, mfa_required: _Optional[bool] = ..., mfa_challenge: _Optional[str] = ..., access: _Optional[_Union[GetWriteAccessResponse, _Mapping]] = ...) -> None: ...

class SignOutRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SignOutResponse(_message.Message):
    __slots__ = ("revocation_confirmed",)
    REVOCATION_CONFIRMED_FIELD_NUMBER: _ClassVar[int]
    revocation_confirmed: bool
    def __init__(self, revocation_confirmed: _Optional[bool] = ...) -> None: ...
