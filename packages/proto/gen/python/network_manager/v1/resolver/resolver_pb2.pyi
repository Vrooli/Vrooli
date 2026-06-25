from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ResolverStatus(_message.Message):
    __slots__ = ("backend", "status", "base_url", "upstreams", "filtering_enabled", "warnings", "enforcement_status", "enforcement_evidence")
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    BASE_URL_FIELD_NUMBER: _ClassVar[int]
    UPSTREAMS_FIELD_NUMBER: _ClassVar[int]
    FILTERING_ENABLED_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    ENFORCEMENT_STATUS_FIELD_NUMBER: _ClassVar[int]
    ENFORCEMENT_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    backend: str
    status: str
    base_url: str
    upstreams: _containers.RepeatedScalarFieldContainer[str]
    filtering_enabled: bool
    warnings: _containers.RepeatedScalarFieldContainer[str]
    enforcement_status: str
    enforcement_evidence: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, backend: _Optional[str] = ..., status: _Optional[str] = ..., base_url: _Optional[str] = ..., upstreams: _Optional[_Iterable[str]] = ..., filtering_enabled: _Optional[bool] = ..., warnings: _Optional[_Iterable[str]] = ..., enforcement_status: _Optional[str] = ..., enforcement_evidence: _Optional[_Iterable[str]] = ...) -> None: ...

class GetResolverStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetResolverStatusResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: ResolverStatus
    def __init__(self, status: _Optional[_Union[ResolverStatus, _Mapping]] = ...) -> None: ...

class ConfigureAdGuardHomeRequest(_message.Message):
    __slots__ = ("base_url", "username", "token_ref", "dry_run")
    BASE_URL_FIELD_NUMBER: _ClassVar[int]
    USERNAME_FIELD_NUMBER: _ClassVar[int]
    TOKEN_REF_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    base_url: str
    username: str
    token_ref: str
    dry_run: bool
    def __init__(self, base_url: _Optional[str] = ..., username: _Optional[str] = ..., token_ref: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class ConfigureAdGuardHomeResponse(_message.Message):
    __slots__ = ("status", "next_steps")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    NEXT_STEPS_FIELD_NUMBER: _ClassVar[int]
    status: ResolverStatus
    next_steps: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, status: _Optional[_Union[ResolverStatus, _Mapping]] = ..., next_steps: _Optional[_Iterable[str]] = ...) -> None: ...

class UpdateUpstreamsRequest(_message.Message):
    __slots__ = ("upstreams", "dry_run")
    UPSTREAMS_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    upstreams: _containers.RepeatedScalarFieldContainer[str]
    dry_run: bool
    def __init__(self, upstreams: _Optional[_Iterable[str]] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class UpdateUpstreamsResponse(_message.Message):
    __slots__ = ("status", "changes")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CHANGES_FIELD_NUMBER: _ClassVar[int]
    status: ResolverStatus
    changes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, status: _Optional[_Union[ResolverStatus, _Mapping]] = ..., changes: _Optional[_Iterable[str]] = ...) -> None: ...

class CheckResolverHealthRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class CheckResolverHealthResponse(_message.Message):
    __slots__ = ("status", "checks")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    status: ResolverStatus
    checks: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, status: _Optional[_Union[ResolverStatus, _Mapping]] = ..., checks: _Optional[_Iterable[str]] = ...) -> None: ...

class RolloutCheck(_message.Message):
    __slots__ = ("id", "title", "status", "evidence", "recommendations")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATIONS_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    status: str
    evidence: str
    recommendations: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., status: _Optional[str] = ..., evidence: _Optional[str] = ..., recommendations: _Optional[_Iterable[str]] = ...) -> None: ...

class AdGuardRollout(_message.Message):
    __slots__ = ("status", "summary", "dns_bind_ip", "resolver_status", "checks", "router_settings", "next_steps", "warnings")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    DNS_BIND_IP_FIELD_NUMBER: _ClassVar[int]
    RESOLVER_STATUS_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    ROUTER_SETTINGS_FIELD_NUMBER: _ClassVar[int]
    NEXT_STEPS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    status: str
    summary: str
    dns_bind_ip: str
    resolver_status: ResolverStatus
    checks: _containers.RepeatedCompositeFieldContainer[RolloutCheck]
    router_settings: _containers.RepeatedScalarFieldContainer[str]
    next_steps: _containers.RepeatedScalarFieldContainer[str]
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, status: _Optional[str] = ..., summary: _Optional[str] = ..., dns_bind_ip: _Optional[str] = ..., resolver_status: _Optional[_Union[ResolverStatus, _Mapping]] = ..., checks: _Optional[_Iterable[_Union[RolloutCheck, _Mapping]]] = ..., router_settings: _Optional[_Iterable[str]] = ..., next_steps: _Optional[_Iterable[str]] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...

class GetAdGuardRolloutRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetAdGuardRolloutResponse(_message.Message):
    __slots__ = ("rollout",)
    ROLLOUT_FIELD_NUMBER: _ClassVar[int]
    rollout: AdGuardRollout
    def __init__(self, rollout: _Optional[_Union[AdGuardRollout, _Mapping]] = ...) -> None: ...
