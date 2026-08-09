import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Account(_message.Message):
    __slots__ = ("id", "email", "username", "roles", "realm", "email_verified", "created_at", "scopes")
    ID_FIELD_NUMBER: _ClassVar[int]
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    USERNAME_FIELD_NUMBER: _ClassVar[int]
    ROLES_FIELD_NUMBER: _ClassVar[int]
    REALM_FIELD_NUMBER: _ClassVar[int]
    EMAIL_VERIFIED_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    id: str
    email: str
    username: str
    roles: _containers.RepeatedScalarFieldContainer[str]
    realm: str
    email_verified: bool
    created_at: _timestamp_pb2.Timestamp
    scopes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., email: _Optional[str] = ..., username: _Optional[str] = ..., roles: _Optional[_Iterable[str]] = ..., realm: _Optional[str] = ..., email_verified: _Optional[bool] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., scopes: _Optional[_Iterable[str]] = ...) -> None: ...

class TokenPair(_message.Message):
    __slots__ = ("access_token", "refresh_token", "access_token_expires_at")
    ACCESS_TOKEN_FIELD_NUMBER: _ClassVar[int]
    REFRESH_TOKEN_FIELD_NUMBER: _ClassVar[int]
    ACCESS_TOKEN_EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    access_token: str
    refresh_token: str
    access_token_expires_at: _timestamp_pb2.Timestamp
    def __init__(self, access_token: _Optional[str] = ..., refresh_token: _Optional[str] = ..., access_token_expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RegisterRequest(_message.Message):
    __slots__ = ("email", "password", "username", "realm")
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_FIELD_NUMBER: _ClassVar[int]
    USERNAME_FIELD_NUMBER: _ClassVar[int]
    REALM_FIELD_NUMBER: _ClassVar[int]
    email: str
    password: str
    username: str
    realm: str
    def __init__(self, email: _Optional[str] = ..., password: _Optional[str] = ..., username: _Optional[str] = ..., realm: _Optional[str] = ...) -> None: ...

class RegisterResponse(_message.Message):
    __slots__ = ("account", "tokens")
    ACCOUNT_FIELD_NUMBER: _ClassVar[int]
    TOKENS_FIELD_NUMBER: _ClassVar[int]
    account: Account
    tokens: TokenPair
    def __init__(self, account: _Optional[_Union[Account, _Mapping]] = ..., tokens: _Optional[_Union[TokenPair, _Mapping]] = ...) -> None: ...

class LoginRequest(_message.Message):
    __slots__ = ("email", "password", "realm")
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_FIELD_NUMBER: _ClassVar[int]
    REALM_FIELD_NUMBER: _ClassVar[int]
    email: str
    password: str
    realm: str
    def __init__(self, email: _Optional[str] = ..., password: _Optional[str] = ..., realm: _Optional[str] = ...) -> None: ...

class LoginResponse(_message.Message):
    __slots__ = ("account", "tokens")
    ACCOUNT_FIELD_NUMBER: _ClassVar[int]
    TOKENS_FIELD_NUMBER: _ClassVar[int]
    account: Account
    tokens: TokenPair
    def __init__(self, account: _Optional[_Union[Account, _Mapping]] = ..., tokens: _Optional[_Union[TokenPair, _Mapping]] = ...) -> None: ...

class ChangePasswordRequest(_message.Message):
    __slots__ = ("access_token", "current_password", "new_password")
    ACCESS_TOKEN_FIELD_NUMBER: _ClassVar[int]
    CURRENT_PASSWORD_FIELD_NUMBER: _ClassVar[int]
    NEW_PASSWORD_FIELD_NUMBER: _ClassVar[int]
    access_token: str
    current_password: str
    new_password: str
    def __init__(self, access_token: _Optional[str] = ..., current_password: _Optional[str] = ..., new_password: _Optional[str] = ...) -> None: ...

class ChangePasswordResponse(_message.Message):
    __slots__ = ("revoked_sessions",)
    REVOKED_SESSIONS_FIELD_NUMBER: _ClassVar[int]
    revoked_sessions: int
    def __init__(self, revoked_sessions: _Optional[int] = ...) -> None: ...

class RefreshRequest(_message.Message):
    __slots__ = ("refresh_token",)
    REFRESH_TOKEN_FIELD_NUMBER: _ClassVar[int]
    refresh_token: str
    def __init__(self, refresh_token: _Optional[str] = ...) -> None: ...

class RefreshResponse(_message.Message):
    __slots__ = ("tokens",)
    TOKENS_FIELD_NUMBER: _ClassVar[int]
    tokens: TokenPair
    def __init__(self, tokens: _Optional[_Union[TokenPair, _Mapping]] = ...) -> None: ...

class LogoutRequest(_message.Message):
    __slots__ = ("access_token",)
    ACCESS_TOKEN_FIELD_NUMBER: _ClassVar[int]
    access_token: str
    def __init__(self, access_token: _Optional[str] = ...) -> None: ...

class LogoutResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ValidateRequest(_message.Message):
    __slots__ = ("access_token",)
    ACCESS_TOKEN_FIELD_NUMBER: _ClassVar[int]
    access_token: str
    def __init__(self, access_token: _Optional[str] = ...) -> None: ...

class ValidateResponse(_message.Message):
    __slots__ = ("valid", "user_id", "email", "roles", "realm", "expires_at", "scopes")
    VALID_FIELD_NUMBER: _ClassVar[int]
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    ROLES_FIELD_NUMBER: _ClassVar[int]
    REALM_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    valid: bool
    user_id: str
    email: str
    roles: _containers.RepeatedScalarFieldContainer[str]
    realm: str
    expires_at: _timestamp_pb2.Timestamp
    scopes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, valid: _Optional[bool] = ..., user_id: _Optional[str] = ..., email: _Optional[str] = ..., roles: _Optional[_Iterable[str]] = ..., realm: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., scopes: _Optional[_Iterable[str]] = ...) -> None: ...

class GrantScopeRequest(_message.Message):
    __slots__ = ("access_token", "principal_id", "scope")
    ACCESS_TOKEN_FIELD_NUMBER: _ClassVar[int]
    PRINCIPAL_ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    access_token: str
    principal_id: str
    scope: str
    def __init__(self, access_token: _Optional[str] = ..., principal_id: _Optional[str] = ..., scope: _Optional[str] = ...) -> None: ...

class RevokeScopeRequest(_message.Message):
    __slots__ = ("access_token", "principal_id", "scope")
    ACCESS_TOKEN_FIELD_NUMBER: _ClassVar[int]
    PRINCIPAL_ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    access_token: str
    principal_id: str
    scope: str
    def __init__(self, access_token: _Optional[str] = ..., principal_id: _Optional[str] = ..., scope: _Optional[str] = ...) -> None: ...

class ListScopesRequest(_message.Message):
    __slots__ = ("access_token", "principal_id")
    ACCESS_TOKEN_FIELD_NUMBER: _ClassVar[int]
    PRINCIPAL_ID_FIELD_NUMBER: _ClassVar[int]
    access_token: str
    principal_id: str
    def __init__(self, access_token: _Optional[str] = ..., principal_id: _Optional[str] = ...) -> None: ...

class ScopeResponse(_message.Message):
    __slots__ = ("principal_id", "scopes")
    PRINCIPAL_ID_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    principal_id: str
    scopes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, principal_id: _Optional[str] = ..., scopes: _Optional[_Iterable[str]] = ...) -> None: ...

class ListScopesResponse(_message.Message):
    __slots__ = ("principal_id", "scopes")
    PRINCIPAL_ID_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    principal_id: str
    scopes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, principal_id: _Optional[str] = ..., scopes: _Optional[_Iterable[str]] = ...) -> None: ...

class LinkMachineAccountRequest(_message.Message):
    __slots__ = ("access_token", "machine_id", "local_principal", "realm", "is_default")
    ACCESS_TOKEN_FIELD_NUMBER: _ClassVar[int]
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    LOCAL_PRINCIPAL_FIELD_NUMBER: _ClassVar[int]
    REALM_FIELD_NUMBER: _ClassVar[int]
    IS_DEFAULT_FIELD_NUMBER: _ClassVar[int]
    access_token: str
    machine_id: str
    local_principal: str
    realm: str
    is_default: bool
    def __init__(self, access_token: _Optional[str] = ..., machine_id: _Optional[str] = ..., local_principal: _Optional[str] = ..., realm: _Optional[str] = ..., is_default: _Optional[bool] = ...) -> None: ...

class LinkMachineAccountResponse(_message.Message):
    __slots__ = ("machine_id", "local_principal", "account_id", "realm", "is_default", "linked_at")
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    LOCAL_PRINCIPAL_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_ID_FIELD_NUMBER: _ClassVar[int]
    REALM_FIELD_NUMBER: _ClassVar[int]
    IS_DEFAULT_FIELD_NUMBER: _ClassVar[int]
    LINKED_AT_FIELD_NUMBER: _ClassVar[int]
    machine_id: str
    local_principal: str
    account_id: str
    realm: str
    is_default: bool
    linked_at: _timestamp_pb2.Timestamp
    def __init__(self, machine_id: _Optional[str] = ..., local_principal: _Optional[str] = ..., account_id: _Optional[str] = ..., realm: _Optional[str] = ..., is_default: _Optional[bool] = ..., linked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ExchangeMachinePrincipalRequest(_message.Message):
    __slots__ = ("machine_id",)
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    machine_id: str
    def __init__(self, machine_id: _Optional[str] = ...) -> None: ...

class IssueBreakGlassRequest(_message.Message):
    __slots__ = ("access_token", "scopes")
    ACCESS_TOKEN_FIELD_NUMBER: _ClassVar[int]
    SCOPES_FIELD_NUMBER: _ClassVar[int]
    access_token: str
    scopes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, access_token: _Optional[str] = ..., scopes: _Optional[_Iterable[str]] = ...) -> None: ...

class IssueBreakGlassResponse(_message.Message):
    __slots__ = ("credential", "expires_at")
    CREDENTIAL_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    credential: str
    expires_at: int
    def __init__(self, credential: _Optional[str] = ..., expires_at: _Optional[int] = ...) -> None: ...
