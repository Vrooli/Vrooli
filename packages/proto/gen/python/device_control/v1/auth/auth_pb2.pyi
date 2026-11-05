import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class UnlockPolicy(_message.Message):
    __slots__ = ("max_attempts", "attempt_limit_ms", "settle_ms")
    MAX_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    ATTEMPT_LIMIT_MS_FIELD_NUMBER: _ClassVar[int]
    SETTLE_MS_FIELD_NUMBER: _ClassVar[int]
    max_attempts: int
    attempt_limit_ms: int
    settle_ms: int
    def __init__(self, max_attempts: _Optional[int] = ..., attempt_limit_ms: _Optional[int] = ..., settle_ms: _Optional[int] = ...) -> None: ...

class AuthenticationProfile(_message.Message):
    __slots__ = ("id", "device_id", "method", "credential_identity", "credential_field", "verification", "policy", "status", "last_outcome", "revoked_at", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_FIELD_FIELD_NUMBER: _ClassVar[int]
    VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    POLICY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LAST_OUTCOME_FIELD_NUMBER: _ClassVar[int]
    REVOKED_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    device_id: str
    method: str
    credential_identity: str
    credential_field: str
    verification: str
    policy: UnlockPolicy
    status: str
    last_outcome: str
    revoked_at: _timestamp_pb2.Timestamp
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., device_id: _Optional[str] = ..., method: _Optional[str] = ..., credential_identity: _Optional[str] = ..., credential_field: _Optional[str] = ..., verification: _Optional[str] = ..., policy: _Optional[_Union[UnlockPolicy, _Mapping]] = ..., status: _Optional[str] = ..., last_outcome: _Optional[str] = ..., revoked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ProviderStatus(_message.Message):
    __slots__ = ("provider", "provider_state", "configured", "provider_detail")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_STATE_FIELD_NUMBER: _ClassVar[int]
    CONFIGURED_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_DETAIL_FIELD_NUMBER: _ClassVar[int]
    provider: str
    provider_state: str
    configured: bool
    provider_detail: str
    def __init__(self, provider: _Optional[str] = ..., provider_state: _Optional[str] = ..., configured: _Optional[bool] = ..., provider_detail: _Optional[str] = ...) -> None: ...

class UnlockResult(_message.Message):
    __slots__ = ("profile_id", "device_id", "method", "outcome", "next_action", "attempts", "provider_state", "before_lock_state", "after_lock_state")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_STATE_FIELD_NUMBER: _ClassVar[int]
    BEFORE_LOCK_STATE_FIELD_NUMBER: _ClassVar[int]
    AFTER_LOCK_STATE_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    device_id: str
    method: str
    outcome: str
    next_action: str
    attempts: int
    provider_state: str
    before_lock_state: str
    after_lock_state: str
    def __init__(self, profile_id: _Optional[str] = ..., device_id: _Optional[str] = ..., method: _Optional[str] = ..., outcome: _Optional[str] = ..., next_action: _Optional[str] = ..., attempts: _Optional[int] = ..., provider_state: _Optional[str] = ..., before_lock_state: _Optional[str] = ..., after_lock_state: _Optional[str] = ...) -> None: ...

class ListProfilesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListProfilesResponse(_message.Message):
    __slots__ = ("profiles",)
    PROFILES_FIELD_NUMBER: _ClassVar[int]
    profiles: _containers.RepeatedCompositeFieldContainer[AuthenticationProfile]
    def __init__(self, profiles: _Optional[_Iterable[_Union[AuthenticationProfile, _Mapping]]] = ...) -> None: ...

class GetProfileRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetProfileResponse(_message.Message):
    __slots__ = ("profile", "provider")
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    profile: AuthenticationProfile
    provider: ProviderStatus
    def __init__(self, profile: _Optional[_Union[AuthenticationProfile, _Mapping]] = ..., provider: _Optional[_Union[ProviderStatus, _Mapping]] = ...) -> None: ...

class CreateProfileRequest(_message.Message):
    __slots__ = ("profile", "actor")
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    profile: AuthenticationProfile
    actor: str
    def __init__(self, profile: _Optional[_Union[AuthenticationProfile, _Mapping]] = ..., actor: _Optional[str] = ...) -> None: ...

class UpdateProfileRequest(_message.Message):
    __slots__ = ("id", "profile", "actor")
    ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    id: str
    profile: AuthenticationProfile
    actor: str
    def __init__(self, id: _Optional[str] = ..., profile: _Optional[_Union[AuthenticationProfile, _Mapping]] = ..., actor: _Optional[str] = ...) -> None: ...

class ProfileResponse(_message.Message):
    __slots__ = ("profile", "provider")
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    profile: AuthenticationProfile
    provider: ProviderStatus
    def __init__(self, profile: _Optional[_Union[AuthenticationProfile, _Mapping]] = ..., provider: _Optional[_Union[ProviderStatus, _Mapping]] = ...) -> None: ...

class RevokeProfileRequest(_message.Message):
    __slots__ = ("id", "actor")
    ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    id: str
    actor: str
    def __init__(self, id: _Optional[str] = ..., actor: _Optional[str] = ...) -> None: ...

class TestProfileRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class UnlockDeviceRequest(_message.Message):
    __slots__ = ("profile_id", "device_id", "actor", "lease_token")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    LEASE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    device_id: str
    actor: str
    lease_token: str
    def __init__(self, profile_id: _Optional[str] = ..., device_id: _Optional[str] = ..., actor: _Optional[str] = ..., lease_token: _Optional[str] = ...) -> None: ...

class UnlockDeviceResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: UnlockResult
    def __init__(self, result: _Optional[_Union[UnlockResult, _Mapping]] = ...) -> None: ...
