import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Tier(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TIER_UNSPECIFIED: _ClassVar[Tier]
    TIER_CORE: _ClassVar[Tier]
    TIER_LEASED: _ClassVar[Tier]

class RouteSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ROUTE_SOURCE_UNSPECIFIED: _ClassVar[RouteSource]
    ROUTE_SOURCE_SCENARIO: _ClassVar[RouteSource]
    ROUTE_SOURCE_EXTERNAL: _ClassVar[RouteSource]
TIER_UNSPECIFIED: Tier
TIER_CORE: Tier
TIER_LEASED: Tier
ROUTE_SOURCE_UNSPECIFIED: RouteSource
ROUTE_SOURCE_SCENARIO: RouteSource
ROUTE_SOURCE_EXTERNAL: RouteSource

class Route(_message.Message):
    __slots__ = ("id", "subdomain", "scenario", "domain", "local_port", "tier", "lease_id", "enabled", "health_path", "public_url", "created_at", "updated_at", "source", "service_target")
    ID_FIELD_NUMBER: _ClassVar[int]
    SUBDOMAIN_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    LOCAL_PORT_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    LEASE_ID_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    HEALTH_PATH_FIELD_NUMBER: _ClassVar[int]
    PUBLIC_URL_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SERVICE_TARGET_FIELD_NUMBER: _ClassVar[int]
    id: str
    subdomain: str
    scenario: str
    domain: str
    local_port: int
    tier: Tier
    lease_id: str
    enabled: bool
    health_path: str
    public_url: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    source: RouteSource
    service_target: str
    def __init__(self, id: _Optional[str] = ..., subdomain: _Optional[str] = ..., scenario: _Optional[str] = ..., domain: _Optional[str] = ..., local_port: _Optional[int] = ..., tier: _Optional[_Union[Tier, str]] = ..., lease_id: _Optional[str] = ..., enabled: _Optional[bool] = ..., health_path: _Optional[str] = ..., public_url: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., source: _Optional[_Union[RouteSource, str]] = ..., service_target: _Optional[str] = ...) -> None: ...

class ListRoutesRequest(_message.Message):
    __slots__ = ("tier",)
    TIER_FIELD_NUMBER: _ClassVar[int]
    tier: Tier
    def __init__(self, tier: _Optional[_Union[Tier, str]] = ...) -> None: ...

class ListRoutesResponse(_message.Message):
    __slots__ = ("routes",)
    ROUTES_FIELD_NUMBER: _ClassVar[int]
    routes: _containers.RepeatedCompositeFieldContainer[Route]
    def __init__(self, routes: _Optional[_Iterable[_Union[Route, _Mapping]]] = ...) -> None: ...

class GetRouteRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetRouteResponse(_message.Message):
    __slots__ = ("route",)
    ROUTE_FIELD_NUMBER: _ClassVar[int]
    route: Route
    def __init__(self, route: _Optional[_Union[Route, _Mapping]] = ...) -> None: ...

class CreateRouteRequest(_message.Message):
    __slots__ = ("subdomain", "scenario", "domain", "local_port", "tier", "lease_id", "health_path", "enabled", "source", "service_target")
    SUBDOMAIN_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    LOCAL_PORT_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    LEASE_ID_FIELD_NUMBER: _ClassVar[int]
    HEALTH_PATH_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SERVICE_TARGET_FIELD_NUMBER: _ClassVar[int]
    subdomain: str
    scenario: str
    domain: str
    local_port: int
    tier: Tier
    lease_id: str
    health_path: str
    enabled: bool
    source: RouteSource
    service_target: str
    def __init__(self, subdomain: _Optional[str] = ..., scenario: _Optional[str] = ..., domain: _Optional[str] = ..., local_port: _Optional[int] = ..., tier: _Optional[_Union[Tier, str]] = ..., lease_id: _Optional[str] = ..., health_path: _Optional[str] = ..., enabled: _Optional[bool] = ..., source: _Optional[_Union[RouteSource, str]] = ..., service_target: _Optional[str] = ...) -> None: ...

class CreateRouteResponse(_message.Message):
    __slots__ = ("route",)
    ROUTE_FIELD_NUMBER: _ClassVar[int]
    route: Route
    def __init__(self, route: _Optional[_Union[Route, _Mapping]] = ...) -> None: ...

class UpdateRouteRequest(_message.Message):
    __slots__ = ("id", "subdomain", "scenario", "domain", "local_port", "tier", "health_path", "enabled", "source", "service_target")
    ID_FIELD_NUMBER: _ClassVar[int]
    SUBDOMAIN_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    LOCAL_PORT_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    HEALTH_PATH_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SERVICE_TARGET_FIELD_NUMBER: _ClassVar[int]
    id: str
    subdomain: str
    scenario: str
    domain: str
    local_port: int
    tier: Tier
    health_path: str
    enabled: bool
    source: RouteSource
    service_target: str
    def __init__(self, id: _Optional[str] = ..., subdomain: _Optional[str] = ..., scenario: _Optional[str] = ..., domain: _Optional[str] = ..., local_port: _Optional[int] = ..., tier: _Optional[_Union[Tier, str]] = ..., health_path: _Optional[str] = ..., enabled: _Optional[bool] = ..., source: _Optional[_Union[RouteSource, str]] = ..., service_target: _Optional[str] = ...) -> None: ...

class UpdateRouteResponse(_message.Message):
    __slots__ = ("route",)
    ROUTE_FIELD_NUMBER: _ClassVar[int]
    route: Route
    def __init__(self, route: _Optional[_Union[Route, _Mapping]] = ...) -> None: ...

class DeleteRouteRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteRouteResponse(_message.Message):
    __slots__ = ("deleted",)
    DELETED_FIELD_NUMBER: _ClassVar[int]
    deleted: bool
    def __init__(self, deleted: _Optional[bool] = ...) -> None: ...
