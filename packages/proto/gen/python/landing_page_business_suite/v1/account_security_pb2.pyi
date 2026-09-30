import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AccountSession(_message.Message):
    __slots__ = ("id", "created_at", "last_used_at", "expires_at", "absolute_expires_at", "device_label", "ip_hint", "auth_method", "current")
    ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_USED_AT_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    ABSOLUTE_EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    DEVICE_LABEL_FIELD_NUMBER: _ClassVar[int]
    IP_HINT_FIELD_NUMBER: _ClassVar[int]
    AUTH_METHOD_FIELD_NUMBER: _ClassVar[int]
    CURRENT_FIELD_NUMBER: _ClassVar[int]
    id: str
    created_at: _timestamp_pb2.Timestamp
    last_used_at: _timestamp_pb2.Timestamp
    expires_at: _timestamp_pb2.Timestamp
    absolute_expires_at: _timestamp_pb2.Timestamp
    device_label: str
    ip_hint: str
    auth_method: str
    current: bool
    def __init__(self, id: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., last_used_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., absolute_expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., device_label: _Optional[str] = ..., ip_hint: _Optional[str] = ..., auth_method: _Optional[str] = ..., current: _Optional[bool] = ...) -> None: ...

class ListSessionsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListSessionsResponse(_message.Message):
    __slots__ = ("sessions",)
    SESSIONS_FIELD_NUMBER: _ClassVar[int]
    sessions: _containers.RepeatedCompositeFieldContainer[AccountSession]
    def __init__(self, sessions: _Optional[_Iterable[_Union[AccountSession, _Mapping]]] = ...) -> None: ...

class RevokeSessionRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class RevokeSessionResponse(_message.Message):
    __slots__ = ("revoked",)
    REVOKED_FIELD_NUMBER: _ClassVar[int]
    revoked: bool
    def __init__(self, revoked: _Optional[bool] = ...) -> None: ...

class RevokeOtherSessionsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class RevokeOtherSessionsResponse(_message.Message):
    __slots__ = ("revoked_count",)
    REVOKED_COUNT_FIELD_NUMBER: _ClassVar[int]
    revoked_count: int
    def __init__(self, revoked_count: _Optional[int] = ...) -> None: ...

class StartReauthenticationRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class StartReauthenticationResponse(_message.Message):
    __slots__ = ("expires_at", "passkey_options_json", "passkey_ceremony_id")
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    PASSKEY_OPTIONS_JSON_FIELD_NUMBER: _ClassVar[int]
    PASSKEY_CEREMONY_ID_FIELD_NUMBER: _ClassVar[int]
    expires_at: _timestamp_pb2.Timestamp
    passkey_options_json: str
    passkey_ceremony_id: str
    def __init__(self, expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., passkey_options_json: _Optional[str] = ..., passkey_ceremony_id: _Optional[str] = ...) -> None: ...

class ReauthenticateRequest(_message.Message):
    __slots__ = ("code", "email", "passkey_assertion", "password", "totp_code", "recovery_code", "passkey_ceremony_id")
    CODE_FIELD_NUMBER: _ClassVar[int]
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    PASSKEY_ASSERTION_FIELD_NUMBER: _ClassVar[int]
    PASSWORD_FIELD_NUMBER: _ClassVar[int]
    TOTP_CODE_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_CODE_FIELD_NUMBER: _ClassVar[int]
    PASSKEY_CEREMONY_ID_FIELD_NUMBER: _ClassVar[int]
    code: str
    email: str
    passkey_assertion: bytes
    password: str
    totp_code: str
    recovery_code: str
    passkey_ceremony_id: str
    def __init__(self, code: _Optional[str] = ..., email: _Optional[str] = ..., passkey_assertion: _Optional[bytes] = ..., password: _Optional[str] = ..., totp_code: _Optional[str] = ..., recovery_code: _Optional[str] = ..., passkey_ceremony_id: _Optional[str] = ...) -> None: ...

class ReauthenticateResponse(_message.Message):
    __slots__ = ("authenticated_at", "reauthenticated")
    AUTHENTICATED_AT_FIELD_NUMBER: _ClassVar[int]
    REAUTHENTICATED_FIELD_NUMBER: _ClassVar[int]
    authenticated_at: _timestamp_pb2.Timestamp
    reauthenticated: bool
    def __init__(self, authenticated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., reauthenticated: _Optional[bool] = ...) -> None: ...

class GetSignInDeliveryStatusRequest(_message.Message):
    __slots__ = ("email", "browser_binding")
    EMAIL_FIELD_NUMBER: _ClassVar[int]
    BROWSER_BINDING_FIELD_NUMBER: _ClassVar[int]
    email: str
    browser_binding: str
    def __init__(self, email: _Optional[str] = ..., browser_binding: _Optional[str] = ...) -> None: ...

class GetSignInDeliveryStatusResponse(_message.Message):
    __slots__ = ("status", "expires_at", "reason_class")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    REASON_CLASS_FIELD_NUMBER: _ClassVar[int]
    status: str
    expires_at: _timestamp_pb2.Timestamp
    reason_class: str
    def __init__(self, status: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., reason_class: _Optional[str] = ...) -> None: ...
