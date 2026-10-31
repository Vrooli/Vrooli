import datetime

from common.v1 import surface_pb2 as _surface_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ChannelProtocol(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CHANNEL_PROTOCOL_UNSPECIFIED: _ClassVar[ChannelProtocol]
    CHANNEL_PROTOCOL_WEBRTC_VP8: _ClassVar[ChannelProtocol]

class ChannelRole(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CHANNEL_ROLE_UNSPECIFIED: _ClassVar[ChannelRole]
    CHANNEL_ROLE_CONTROLLER: _ClassVar[ChannelRole]
    CHANNEL_ROLE_VIEWER: _ClassVar[ChannelRole]

class RouteKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ROUTE_KIND_UNSPECIFIED: _ClassVar[RouteKind]
    ROUTE_KIND_DIRECT: _ClassVar[RouteKind]
    ROUTE_KIND_TURN: _ClassVar[RouteKind]

class SignalKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SIGNAL_KIND_UNSPECIFIED: _ClassVar[SignalKind]
    SIGNAL_KIND_OFFER: _ClassVar[SignalKind]
    SIGNAL_KIND_ANSWER: _ClassVar[SignalKind]
    SIGNAL_KIND_ICE: _ClassVar[SignalKind]

class RevokeReason(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    REVOKE_REASON_UNSPECIFIED: _ClassVar[RevokeReason]
    REVOKE_REASON_OPERATOR: _ClassVar[RevokeReason]
    REVOKE_REASON_NODE_REVOKED: _ClassVar[RevokeReason]
    REVOKE_REASON_LEASE_EXPIRED: _ClassVar[RevokeReason]
    REVOKE_REASON_COMPANION_LOST: _ClassVar[RevokeReason]
    REVOKE_REASON_POLICY: _ClassVar[RevokeReason]
CHANNEL_PROTOCOL_UNSPECIFIED: ChannelProtocol
CHANNEL_PROTOCOL_WEBRTC_VP8: ChannelProtocol
CHANNEL_ROLE_UNSPECIFIED: ChannelRole
CHANNEL_ROLE_CONTROLLER: ChannelRole
CHANNEL_ROLE_VIEWER: ChannelRole
ROUTE_KIND_UNSPECIFIED: RouteKind
ROUTE_KIND_DIRECT: RouteKind
ROUTE_KIND_TURN: RouteKind
SIGNAL_KIND_UNSPECIFIED: SignalKind
SIGNAL_KIND_OFFER: SignalKind
SIGNAL_KIND_ANSWER: SignalKind
SIGNAL_KIND_ICE: SignalKind
REVOKE_REASON_UNSPECIFIED: RevokeReason
REVOKE_REASON_OPERATOR: RevokeReason
REVOKE_REASON_NODE_REVOKED: RevokeReason
REVOKE_REASON_LEASE_EXPIRED: RevokeReason
REVOKE_REASON_COMPANION_LOST: RevokeReason
REVOKE_REASON_POLICY: RevokeReason

class ChannelGrant(_message.Message):
    __slots__ = ("channel_id", "node_id", "surface", "session_id", "lease_id", "lease_epoch", "protocol", "role", "expires_at", "policy_revision", "routes")
    CHANNEL_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    LEASE_ID_FIELD_NUMBER: _ClassVar[int]
    LEASE_EPOCH_FIELD_NUMBER: _ClassVar[int]
    PROTOCOL_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    POLICY_REVISION_FIELD_NUMBER: _ClassVar[int]
    ROUTES_FIELD_NUMBER: _ClassVar[int]
    channel_id: str
    node_id: str
    surface: _surface_pb2.SurfaceRef
    session_id: str
    lease_id: str
    lease_epoch: int
    protocol: ChannelProtocol
    role: ChannelRole
    expires_at: _timestamp_pb2.Timestamp
    policy_revision: str
    routes: _containers.RepeatedCompositeFieldContainer[RouteCandidate]
    def __init__(self, channel_id: _Optional[str] = ..., node_id: _Optional[str] = ..., surface: _Optional[_Union[_surface_pb2.SurfaceRef, _Mapping]] = ..., session_id: _Optional[str] = ..., lease_id: _Optional[str] = ..., lease_epoch: _Optional[int] = ..., protocol: _Optional[_Union[ChannelProtocol, str]] = ..., role: _Optional[_Union[ChannelRole, str]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., policy_revision: _Optional[str] = ..., routes: _Optional[_Iterable[_Union[RouteCandidate, _Mapping]]] = ...) -> None: ...

class OpenChannelRequest(_message.Message):
    __slots__ = ("node_id", "surface", "session_id", "lease_id", "lease_epoch", "protocol", "role", "takeover", "ttl_seconds", "request_id")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    LEASE_ID_FIELD_NUMBER: _ClassVar[int]
    LEASE_EPOCH_FIELD_NUMBER: _ClassVar[int]
    PROTOCOL_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    TAKEOVER_FIELD_NUMBER: _ClassVar[int]
    TTL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    surface: _surface_pb2.SurfaceRef
    session_id: str
    lease_id: str
    lease_epoch: int
    protocol: ChannelProtocol
    role: ChannelRole
    takeover: bool
    ttl_seconds: int
    request_id: str
    def __init__(self, node_id: _Optional[str] = ..., surface: _Optional[_Union[_surface_pb2.SurfaceRef, _Mapping]] = ..., session_id: _Optional[str] = ..., lease_id: _Optional[str] = ..., lease_epoch: _Optional[int] = ..., protocol: _Optional[_Union[ChannelProtocol, str]] = ..., role: _Optional[_Union[ChannelRole, str]] = ..., takeover: _Optional[bool] = ..., ttl_seconds: _Optional[int] = ..., request_id: _Optional[str] = ...) -> None: ...

class RouteCandidate(_message.Message):
    __slots__ = ("kind", "url", "username", "credential", "priority")
    KIND_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    USERNAME_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    kind: RouteKind
    url: str
    username: str
    credential: str
    priority: int
    def __init__(self, kind: _Optional[_Union[RouteKind, str]] = ..., url: _Optional[str] = ..., username: _Optional[str] = ..., credential: _Optional[str] = ..., priority: _Optional[int] = ...) -> None: ...

class OpenChannelResponse(_message.Message):
    __slots__ = ("grant", "routes")
    GRANT_FIELD_NUMBER: _ClassVar[int]
    ROUTES_FIELD_NUMBER: _ClassVar[int]
    grant: ChannelGrant
    routes: _containers.RepeatedCompositeFieldContainer[RouteCandidate]
    def __init__(self, grant: _Optional[_Union[ChannelGrant, _Mapping]] = ..., routes: _Optional[_Iterable[_Union[RouteCandidate, _Mapping]]] = ...) -> None: ...

class SignalRequest(_message.Message):
    __slots__ = ("grant", "kind", "generation", "payload", "request_id")
    GRANT_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    grant: ChannelGrant
    kind: SignalKind
    generation: str
    payload: bytes
    request_id: str
    def __init__(self, grant: _Optional[_Union[ChannelGrant, _Mapping]] = ..., kind: _Optional[_Union[SignalKind, str]] = ..., generation: _Optional[str] = ..., payload: _Optional[bytes] = ..., request_id: _Optional[str] = ...) -> None: ...

class SignalResponse(_message.Message):
    __slots__ = ("accepted", "generation", "reason_code", "payload", "request_id")
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    accepted: bool
    generation: str
    reason_code: str
    payload: bytes
    request_id: str
    def __init__(self, accepted: _Optional[bool] = ..., generation: _Optional[str] = ..., reason_code: _Optional[str] = ..., payload: _Optional[bytes] = ..., request_id: _Optional[str] = ...) -> None: ...

class CloseChannelRequest(_message.Message):
    __slots__ = ("grant", "reason")
    GRANT_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    grant: ChannelGrant
    reason: str
    def __init__(self, grant: _Optional[_Union[ChannelGrant, _Mapping]] = ..., reason: _Optional[str] = ...) -> None: ...

class CloseChannelResponse(_message.Message):
    __slots__ = ("closed",)
    CLOSED_FIELD_NUMBER: _ClassVar[int]
    closed: bool
    def __init__(self, closed: _Optional[bool] = ...) -> None: ...

class RevokeChannelRequest(_message.Message):
    __slots__ = ("channel_id", "reason", "detail")
    CHANNEL_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    channel_id: str
    reason: RevokeReason
    detail: str
    def __init__(self, channel_id: _Optional[str] = ..., reason: _Optional[_Union[RevokeReason, str]] = ..., detail: _Optional[str] = ...) -> None: ...

class RevokeChannelResponse(_message.Message):
    __slots__ = ("revoked", "revoked_at")
    REVOKED_FIELD_NUMBER: _ClassVar[int]
    REVOKED_AT_FIELD_NUMBER: _ClassVar[int]
    revoked: bool
    revoked_at: _timestamp_pb2.Timestamp
    def __init__(self, revoked: _Optional[bool] = ..., revoked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
