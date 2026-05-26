from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class InstallTestPoolRequest(_message.Message):
    __slots__ = ("dsn", "lease_id", "lease_ttl_ms")
    DSN_FIELD_NUMBER: _ClassVar[int]
    LEASE_ID_FIELD_NUMBER: _ClassVar[int]
    LEASE_TTL_MS_FIELD_NUMBER: _ClassVar[int]
    dsn: str
    lease_id: str
    lease_ttl_ms: int
    def __init__(self, dsn: _Optional[str] = ..., lease_id: _Optional[str] = ..., lease_ttl_ms: _Optional[int] = ...) -> None: ...

class InstallTestPoolResponse(_message.Message):
    __slots__ = ("active_lease_id",)
    ACTIVE_LEASE_ID_FIELD_NUMBER: _ClassVar[int]
    active_lease_id: str
    def __init__(self, active_lease_id: _Optional[str] = ...) -> None: ...

class ClearTestPoolRequest(_message.Message):
    __slots__ = ("lease_id",)
    LEASE_ID_FIELD_NUMBER: _ClassVar[int]
    lease_id: str
    def __init__(self, lease_id: _Optional[str] = ...) -> None: ...

class ClearTestPoolResponse(_message.Message):
    __slots__ = ("stats",)
    STATS_FIELD_NUMBER: _ClassVar[int]
    stats: LeaseStats
    def __init__(self, stats: _Optional[_Union[LeaseStats, _Mapping]] = ...) -> None: ...

class HeartbeatTestPoolRequest(_message.Message):
    __slots__ = ("lease_id",)
    LEASE_ID_FIELD_NUMBER: _ClassVar[int]
    lease_id: str
    def __init__(self, lease_id: _Optional[str] = ...) -> None: ...

class HeartbeatTestPoolResponse(_message.Message):
    __slots__ = ("expires_at_unix_ms",)
    EXPIRES_AT_UNIX_MS_FIELD_NUMBER: _ClassVar[int]
    expires_at_unix_ms: int
    def __init__(self, expires_at_unix_ms: _Optional[int] = ...) -> None: ...

class LeaseStats(_message.Message):
    __slots__ = ("test_pool_requests", "primary_during_test_mode_requests")
    TEST_POOL_REQUESTS_FIELD_NUMBER: _ClassVar[int]
    PRIMARY_DURING_TEST_MODE_REQUESTS_FIELD_NUMBER: _ClassVar[int]
    test_pool_requests: int
    primary_during_test_mode_requests: int
    def __init__(self, test_pool_requests: _Optional[int] = ..., primary_during_test_mode_requests: _Optional[int] = ...) -> None: ...
