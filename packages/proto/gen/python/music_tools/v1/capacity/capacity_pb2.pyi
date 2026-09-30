from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class StatusRequest(_message.Message):
    __slots__ = ("resource",)
    RESOURCE_FIELD_NUMBER: _ClassVar[int]
    resource: str
    def __init__(self, resource: _Optional[str] = ...) -> None: ...

class CapacityStatus(_message.Message):
    __slots__ = ("resource", "applied_rung", "free_vram_bytes", "preferred_vram_bytes", "degraded")
    RESOURCE_FIELD_NUMBER: _ClassVar[int]
    APPLIED_RUNG_FIELD_NUMBER: _ClassVar[int]
    FREE_VRAM_BYTES_FIELD_NUMBER: _ClassVar[int]
    PREFERRED_VRAM_BYTES_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    resource: str
    applied_rung: str
    free_vram_bytes: int
    preferred_vram_bytes: int
    degraded: bool
    def __init__(self, resource: _Optional[str] = ..., applied_rung: _Optional[str] = ..., free_vram_bytes: _Optional[int] = ..., preferred_vram_bytes: _Optional[int] = ..., degraded: _Optional[bool] = ...) -> None: ...

class ClaimRequest(_message.Message):
    __slots__ = ("resource", "resource_kind", "preferred_bytes", "floor_bytes")
    RESOURCE_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_KIND_FIELD_NUMBER: _ClassVar[int]
    PREFERRED_BYTES_FIELD_NUMBER: _ClassVar[int]
    FLOOR_BYTES_FIELD_NUMBER: _ClassVar[int]
    resource: str
    resource_kind: str
    preferred_bytes: int
    floor_bytes: int
    def __init__(self, resource: _Optional[str] = ..., resource_kind: _Optional[str] = ..., preferred_bytes: _Optional[int] = ..., floor_bytes: _Optional[int] = ...) -> None: ...

class CapacityLease(_message.Message):
    __slots__ = ("id", "verdict", "applied_rung", "granted_bytes", "reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    APPLIED_RUNG_FIELD_NUMBER: _ClassVar[int]
    GRANTED_BYTES_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    verdict: str
    applied_rung: str
    granted_bytes: int
    reason: str
    def __init__(self, id: _Optional[str] = ..., verdict: _Optional[str] = ..., applied_rung: _Optional[str] = ..., granted_bytes: _Optional[int] = ..., reason: _Optional[str] = ...) -> None: ...

class ReleaseRequest(_message.Message):
    __slots__ = ("lease_id",)
    LEASE_ID_FIELD_NUMBER: _ClassVar[int]
    lease_id: str
    def __init__(self, lease_id: _Optional[str] = ...) -> None: ...

class ReleaseResponse(_message.Message):
    __slots__ = ("released",)
    RELEASED_FIELD_NUMBER: _ClassVar[int]
    released: bool
    def __init__(self, released: _Optional[bool] = ...) -> None: ...
