from landing_page_business_suite.v1 import account_security_pb2 as _account_security_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class LoginRequest(_message.Message):
    __slots__ = ("email", "password", "totp_code", "passkey_assertion", "passkey_ceremony_id")
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_FIELD_NUMBER: _ClassVar[int]
    TOTP_CODE_FIELD_NUMBER: _ClassVar[int]
    PASSKEY_ASSERTION_FIELD_NUMBER: _ClassVar[int]
    PASSKEY_CEREMONY_ID_FIELD_NUMBER: _ClassVar[int]
    email: str
    password: str
    totp_code: str
    passkey_assertion: bytes
    passkey_ceremony_id: str
    def __init__(self, email: _Optional[str] = ..., password: _Optional[str] = ..., totp_code: _Optional[str] = ..., passkey_assertion: _Optional[bytes] = ..., passkey_ceremony_id: _Optional[str] = ...) -> None: ...

class AdminSessionResponse(_message.Message):
    __slots__ = ("email", "authenticated", "reset_enabled", "session_id", "assurance")
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    AUTHENTICATED_FIELD_NUMBER: _ClassVar[int]
    RESET_ENABLED_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    ASSURANCE_FIELD_NUMBER: _ClassVar[int]
    email: str
    authenticated: bool
    reset_enabled: bool
    session_id: str
    assurance: str
    def __init__(self, email: _Optional[str] = ..., authenticated: _Optional[bool] = ..., reset_enabled: _Optional[bool] = ..., session_id: _Optional[str] = ..., assurance: _Optional[str] = ...) -> None: ...

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

class BeginAdminPasskeyRegistrationRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class BeginAdminPasskeyRegistrationResponse(_message.Message):
    __slots__ = ("options_json", "ceremony_id")
    OPTIONS_JSON_FIELD_NUMBER: _ClassVar[int]
    CEREMONY_ID_FIELD_NUMBER: _ClassVar[int]
    options_json: str
    ceremony_id: str
    def __init__(self, options_json: _Optional[str] = ..., ceremony_id: _Optional[str] = ...) -> None: ...

class FinishAdminPasskeyRegistrationRequest(_message.Message):
    __slots__ = ("credential_json", "nickname", "ceremony_id")
    CREDENTIAL_JSON_FIELD_NUMBER: _ClassVar[int]
    NICKNAME_FIELD_NUMBER: _ClassVar[int]
    CEREMONY_ID_FIELD_NUMBER: _ClassVar[int]
    credential_json: str
    nickname: str
    ceremony_id: str
    def __init__(self, credential_json: _Optional[str] = ..., nickname: _Optional[str] = ..., ceremony_id: _Optional[str] = ...) -> None: ...

class FinishAdminPasskeyRegistrationResponse(_message.Message):
    __slots__ = ("id", "nickname")
    ID_FIELD_NUMBER: _ClassVar[int]
    NICKNAME_FIELD_NUMBER: _ClassVar[int]
    id: str
    nickname: str
    def __init__(self, id: _Optional[str] = ..., nickname: _Optional[str] = ...) -> None: ...

class ListAdminPasskeysRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class AdminPasskey(_message.Message):
    __slots__ = ("id", "nickname", "created_at", "last_used_at", "backup_state")
    ID_FIELD_NUMBER: _ClassVar[int]
    NICKNAME_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_USED_AT_FIELD_NUMBER: _ClassVar[int]
    BACKUP_STATE_FIELD_NUMBER: _ClassVar[int]
    id: str
    nickname: str
    created_at: str
    last_used_at: str
    backup_state: str
    def __init__(self, id: _Optional[str] = ..., nickname: _Optional[str] = ..., created_at: _Optional[str] = ..., last_used_at: _Optional[str] = ..., backup_state: _Optional[str] = ...) -> None: ...

class ListAdminPasskeysResponse(_message.Message):
    __slots__ = ("passkeys",)
    PASSKEYS_FIELD_NUMBER: _ClassVar[int]
    passkeys: _containers.RepeatedCompositeFieldContainer[AdminPasskey]
    def __init__(self, passkeys: _Optional[_Iterable[_Union[AdminPasskey, _Mapping]]] = ...) -> None: ...

class RenameAdminPasskeyRequest(_message.Message):
    __slots__ = ("id", "nickname")
    ID_FIELD_NUMBER: _ClassVar[int]
    NICKNAME_FIELD_NUMBER: _ClassVar[int]
    id: str
    nickname: str
    def __init__(self, id: _Optional[str] = ..., nickname: _Optional[str] = ...) -> None: ...

class RenameAdminPasskeyResponse(_message.Message):
    __slots__ = ("passkey",)
    PASSKEY_FIELD_NUMBER: _ClassVar[int]
    passkey: AdminPasskey
    def __init__(self, passkey: _Optional[_Union[AdminPasskey, _Mapping]] = ...) -> None: ...

class RevokeAdminPasskeyRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RevokeAdminPasskeyResponse(_message.Message):
    __slots__ = ("revoked",)
    REVOKED_FIELD_NUMBER: _ClassVar[int]
    revoked: bool
    def __init__(self, revoked: _Optional[bool] = ...) -> None: ...

class BeginAdminSecondFactorRequest(_message.Message):
    __slots__ = ("email",)
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    email: str
    def __init__(self, email: _Optional[str] = ...) -> None: ...

class BeginAdminSecondFactorResponse(_message.Message):
    __slots__ = ("options_json", "ceremony_id")
    OPTIONS_JSON_FIELD_NUMBER: _ClassVar[int]
    CEREMONY_ID_FIELD_NUMBER: _ClassVar[int]
    options_json: str
    ceremony_id: str
    def __init__(self, options_json: _Optional[str] = ..., ceremony_id: _Optional[str] = ...) -> None: ...

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

class AdminProfile(_message.Message):
    __slots__ = ("email", "is_default_email", "is_default_password")
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    IS_DEFAULT_EMAIL_FIELD_NUMBER: _ClassVar[int]
    IS_DEFAULT_PASSWORD_FIELD_NUMBER: _ClassVar[int]
    email: str
    is_default_email: bool
    is_default_password: bool
    def __init__(self, email: _Optional[str] = ..., is_default_email: _Optional[bool] = ..., is_default_password: _Optional[bool] = ...) -> None: ...

class GetAdminProfileRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetAdminProfileResponse(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: AdminProfile
    def __init__(self, profile: _Optional[_Union[AdminProfile, _Mapping]] = ...) -> None: ...

class UpdateAdminProfileRequest(_message.Message):
    __slots__ = ("current_password", "new_email", "new_password")
    CURRENT_PASSWORD_FIELD_NUMBER: _ClassVar[int]
    NEW_EMAIL_FIELD_NUMBER: _ClassVar[int]
    NEW_PASSWORD_FIELD_NUMBER: _ClassVar[int]
    current_password: str
    new_email: str
    new_password: str
    def __init__(self, current_password: _Optional[str] = ..., new_email: _Optional[str] = ..., new_password: _Optional[str] = ...) -> None: ...

class UpdateAdminProfileResponse(_message.Message):
    __slots__ = ("profile",)
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    profile: AdminProfile
    def __init__(self, profile: _Optional[_Union[AdminProfile, _Mapping]] = ...) -> None: ...

class APIKey(_message.Message):
    __slots__ = ("id", "provider", "key_hint", "is_active", "last_verified_at", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    KEY_HINT_FIELD_NUMBER: _ClassVar[int]
    IS_ACTIVE_FIELD_NUMBER: _ClassVar[int]
    LAST_VERIFIED_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    provider: str
    key_hint: str
    is_active: bool
    last_verified_at: str
    created_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., provider: _Optional[str] = ..., key_hint: _Optional[str] = ..., is_active: _Optional[bool] = ..., last_verified_at: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class ListAPIKeysRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListAPIKeysResponse(_message.Message):
    __slots__ = ("keys",)
    KEYS_FIELD_NUMBER: _ClassVar[int]
    keys: _containers.RepeatedCompositeFieldContainer[APIKey]
    def __init__(self, keys: _Optional[_Iterable[_Union[APIKey, _Mapping]]] = ...) -> None: ...

class CreateAPIKeyRequest(_message.Message):
    __slots__ = ("provider", "key")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    KEY_FIELD_NUMBER: _ClassVar[int]
    provider: str
    key: str
    def __init__(self, provider: _Optional[str] = ..., key: _Optional[str] = ...) -> None: ...

class CreateAPIKeyResponse(_message.Message):
    __slots__ = ("key",)
    KEY_FIELD_NUMBER: _ClassVar[int]
    key: APIKey
    def __init__(self, key: _Optional[_Union[APIKey, _Mapping]] = ...) -> None: ...

class DeleteAPIKeyRequest(_message.Message):
    __slots__ = ("provider",)
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    provider: str
    def __init__(self, provider: _Optional[str] = ...) -> None: ...

class DeleteAPIKeyResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class TestAPIKeyRequest(_message.Message):
    __slots__ = ("provider",)
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    provider: str
    def __init__(self, provider: _Optional[str] = ...) -> None: ...

class TestAPIKeyResponse(_message.Message):
    __slots__ = ("success", "message", "provider")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    success: bool
    message: str
    provider: str
    def __init__(self, success: _Optional[bool] = ..., message: _Optional[str] = ..., provider: _Optional[str] = ...) -> None: ...

class SetAPIKeyActiveRequest(_message.Message):
    __slots__ = ("provider", "active")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    provider: str
    active: bool
    def __init__(self, provider: _Optional[str] = ..., active: _Optional[bool] = ...) -> None: ...

class SetAPIKeyActiveResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
