from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class BeginPasskeyRegistrationRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class BeginPasskeyRegistrationResponse(_message.Message):
    __slots__ = ("options_json", "ceremony_id")
    OPTIONS_JSON_FIELD_NUMBER: _ClassVar[int]
    CEREMONY_ID_FIELD_NUMBER: _ClassVar[int]
    options_json: str
    ceremony_id: str
    def __init__(self, options_json: _Optional[str] = ..., ceremony_id: _Optional[str] = ...) -> None: ...

class FinishPasskeyRegistrationRequest(_message.Message):
    __slots__ = ("credential_json", "nickname", "ceremony_id")
    CREDENTIAL_JSON_FIELD_NUMBER: _ClassVar[int]
    NICKNAME_FIELD_NUMBER: _ClassVar[int]
    CEREMONY_ID_FIELD_NUMBER: _ClassVar[int]
    credential_json: str
    nickname: str
    ceremony_id: str
    def __init__(self, credential_json: _Optional[str] = ..., nickname: _Optional[str] = ..., ceremony_id: _Optional[str] = ...) -> None: ...

class FinishPasskeyRegistrationResponse(_message.Message):
    __slots__ = ("id", "nickname")
    ID_FIELD_NUMBER: _ClassVar[int]
    NICKNAME_FIELD_NUMBER: _ClassVar[int]
    id: str
    nickname: str
    def __init__(self, id: _Optional[str] = ..., nickname: _Optional[str] = ...) -> None: ...

class BeginPasskeyAuthenticationRequest(_message.Message):
    __slots__ = ("context",)
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    context: str
    def __init__(self, context: _Optional[str] = ...) -> None: ...

class BeginPasskeyAuthenticationResponse(_message.Message):
    __slots__ = ("options_json", "ceremony_id")
    OPTIONS_JSON_FIELD_NUMBER: _ClassVar[int]
    CEREMONY_ID_FIELD_NUMBER: _ClassVar[int]
    options_json: str
    ceremony_id: str
    def __init__(self, options_json: _Optional[str] = ..., ceremony_id: _Optional[str] = ...) -> None: ...

class FinishPasskeyAuthenticationRequest(_message.Message):
    __slots__ = ("credential_json", "context", "ceremony_id")
    CREDENTIAL_JSON_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    CEREMONY_ID_FIELD_NUMBER: _ClassVar[int]
    credential_json: str
    context: str
    ceremony_id: str
    def __init__(self, credential_json: _Optional[str] = ..., context: _Optional[str] = ..., ceremony_id: _Optional[str] = ...) -> None: ...

class FinishPasskeyAuthenticationResponse(_message.Message):
    __slots__ = ("authenticated", "redirect_url", "desktop_continuation")
    AUTHENTICATED_FIELD_NUMBER: _ClassVar[int]
    REDIRECT_URL_FIELD_NUMBER: _ClassVar[int]
    DESKTOP_CONTINUATION_FIELD_NUMBER: _ClassVar[int]
    authenticated: bool
    redirect_url: str
    desktop_continuation: bool
    def __init__(self, authenticated: _Optional[bool] = ..., redirect_url: _Optional[str] = ..., desktop_continuation: _Optional[bool] = ...) -> None: ...

class ListPasskeysRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class Passkey(_message.Message):
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

class ListPasskeysResponse(_message.Message):
    __slots__ = ("passkeys",)
    PASSKEYS_FIELD_NUMBER: _ClassVar[int]
    passkeys: _containers.RepeatedCompositeFieldContainer[Passkey]
    def __init__(self, passkeys: _Optional[_Iterable[_Union[Passkey, _Mapping]]] = ...) -> None: ...

class RenamePasskeyRequest(_message.Message):
    __slots__ = ("id", "nickname")
    ID_FIELD_NUMBER: _ClassVar[int]
    NICKNAME_FIELD_NUMBER: _ClassVar[int]
    id: str
    nickname: str
    def __init__(self, id: _Optional[str] = ..., nickname: _Optional[str] = ...) -> None: ...

class RenamePasskeyResponse(_message.Message):
    __slots__ = ("passkey",)
    PASSKEY_FIELD_NUMBER: _ClassVar[int]
    passkey: Passkey
    def __init__(self, passkey: _Optional[_Union[Passkey, _Mapping]] = ...) -> None: ...

class RevokePasskeyRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RevokePasskeyResponse(_message.Message):
    __slots__ = ("revoked",)
    REVOKED_FIELD_NUMBER: _ClassVar[int]
    revoked: bool
    def __init__(self, revoked: _Optional[bool] = ...) -> None: ...
