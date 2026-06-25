import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class LeaseStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    LEASE_STATUS_UNSPECIFIED: _ClassVar[LeaseStatus]
    LEASE_STATUS_ACTIVE: _ClassVar[LeaseStatus]
    LEASE_STATUS_EXPIRED: _ClassVar[LeaseStatus]
    LEASE_STATUS_REVOKED: _ClassVar[LeaseStatus]
LEASE_STATUS_UNSPECIFIED: LeaseStatus
LEASE_STATUS_ACTIVE: LeaseStatus
LEASE_STATUS_EXPIRED: LeaseStatus
LEASE_STATUS_REVOKED: LeaseStatus

class Lease(_message.Message):
    __slots__ = ("id", "scenario", "requested_by", "created_at", "expires_at", "extended_count", "status")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_BY_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    EXTENDED_COUNT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario: str
    requested_by: str
    created_at: _timestamp_pb2.Timestamp
    expires_at: _timestamp_pb2.Timestamp
    extended_count: int
    status: LeaseStatus
    def __init__(self, id: _Optional[str] = ..., scenario: _Optional[str] = ..., requested_by: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., extended_count: _Optional[int] = ..., status: _Optional[_Union[LeaseStatus, str]] = ...) -> None: ...

class Exposure(_message.Message):
    __slots__ = ("scenario", "subdomain", "public_url", "local_port", "tier", "enabled", "lease")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SUBDOMAIN_FIELD_NUMBER: _ClassVar[int]
    PUBLIC_URL_FIELD_NUMBER: _ClassVar[int]
    LOCAL_PORT_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    LEASE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    subdomain: str
    public_url: str
    local_port: int
    tier: str
    enabled: bool
    lease: Lease
    def __init__(self, scenario: _Optional[str] = ..., subdomain: _Optional[str] = ..., public_url: _Optional[str] = ..., local_port: _Optional[int] = ..., tier: _Optional[str] = ..., enabled: _Optional[bool] = ..., lease: _Optional[_Union[Lease, _Mapping]] = ...) -> None: ...

class ExposeRequest(_message.Message):
    __slots__ = ("scenario", "ttl_seconds", "requested_by")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TTL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_BY_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    ttl_seconds: int
    requested_by: str
    def __init__(self, scenario: _Optional[str] = ..., ttl_seconds: _Optional[int] = ..., requested_by: _Optional[str] = ...) -> None: ...

class ExposeResponse(_message.Message):
    __slots__ = ("lease", "public_url")
    LEASE_FIELD_NUMBER: _ClassVar[int]
    PUBLIC_URL_FIELD_NUMBER: _ClassVar[int]
    lease: Lease
    public_url: str
    def __init__(self, lease: _Optional[_Union[Lease, _Mapping]] = ..., public_url: _Optional[str] = ...) -> None: ...

class ExtendLeaseRequest(_message.Message):
    __slots__ = ("lease_id", "ttl_seconds")
    LEASE_ID_FIELD_NUMBER: _ClassVar[int]
    TTL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    lease_id: str
    ttl_seconds: int
    def __init__(self, lease_id: _Optional[str] = ..., ttl_seconds: _Optional[int] = ...) -> None: ...

class ExtendLeaseResponse(_message.Message):
    __slots__ = ("lease",)
    LEASE_FIELD_NUMBER: _ClassVar[int]
    lease: Lease
    def __init__(self, lease: _Optional[_Union[Lease, _Mapping]] = ...) -> None: ...

class RevokeLeaseRequest(_message.Message):
    __slots__ = ("lease_id",)
    LEASE_ID_FIELD_NUMBER: _ClassVar[int]
    lease_id: str
    def __init__(self, lease_id: _Optional[str] = ...) -> None: ...

class RevokeLeaseResponse(_message.Message):
    __slots__ = ("retracted",)
    RETRACTED_FIELD_NUMBER: _ClassVar[int]
    retracted: bool
    def __init__(self, retracted: _Optional[bool] = ...) -> None: ...

class UnexposeRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class UnexposeResponse(_message.Message):
    __slots__ = ("retracted", "lease_id")
    RETRACTED_FIELD_NUMBER: _ClassVar[int]
    LEASE_ID_FIELD_NUMBER: _ClassVar[int]
    retracted: bool
    lease_id: str
    def __init__(self, retracted: _Optional[bool] = ..., lease_id: _Optional[str] = ...) -> None: ...

class ListLeasesRequest(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: LeaseStatus
    def __init__(self, status: _Optional[_Union[LeaseStatus, str]] = ...) -> None: ...

class ListLeasesResponse(_message.Message):
    __slots__ = ("leases",)
    LEASES_FIELD_NUMBER: _ClassVar[int]
    leases: _containers.RepeatedCompositeFieldContainer[Lease]
    def __init__(self, leases: _Optional[_Iterable[_Union[Lease, _Mapping]]] = ...) -> None: ...

class ListExposuresRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListExposuresResponse(_message.Message):
    __slots__ = ("exposures",)
    EXPOSURES_FIELD_NUMBER: _ClassVar[int]
    exposures: _containers.RepeatedCompositeFieldContainer[Exposure]
    def __init__(self, exposures: _Optional[_Iterable[_Union[Exposure, _Mapping]]] = ...) -> None: ...

class IsExposedRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class IsExposedResponse(_message.Message):
    __slots__ = ("exposed", "public_url")
    EXPOSED_FIELD_NUMBER: _ClassVar[int]
    PUBLIC_URL_FIELD_NUMBER: _ClassVar[int]
    exposed: bool
    public_url: str
    def __init__(self, exposed: _Optional[bool] = ..., public_url: _Optional[str] = ...) -> None: ...

class ReconcileRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ReconcileResponse(_message.Message):
    __slots__ = ("core_ensured", "leases_reaped")
    CORE_ENSURED_FIELD_NUMBER: _ClassVar[int]
    LEASES_REAPED_FIELD_NUMBER: _ClassVar[int]
    core_ensured: int
    leases_reaped: int
    def __init__(self, core_ensured: _Optional[int] = ..., leases_reaped: _Optional[int] = ...) -> None: ...
