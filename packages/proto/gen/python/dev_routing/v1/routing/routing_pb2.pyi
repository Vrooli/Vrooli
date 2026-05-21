from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class InstallTestPoolRequest(_message.Message):
    __slots__ = ("dsn", "lease_id")
    DSN_FIELD_NUMBER: _ClassVar[int]
    LEASE_ID_FIELD_NUMBER: _ClassVar[int]
    dsn: str
    lease_id: str
    def __init__(self, dsn: _Optional[str] = ..., lease_id: _Optional[str] = ...) -> None: ...

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
    __slots__ = ()
    def __init__(self) -> None: ...
