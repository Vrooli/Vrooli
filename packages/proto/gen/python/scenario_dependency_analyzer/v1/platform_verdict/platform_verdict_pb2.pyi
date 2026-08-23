from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListPlatformVerdictsRequest(_message.Message):
    __slots__ = ("scenario", "refresh")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    REFRESH_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    refresh: bool
    def __init__(self, scenario: _Optional[str] = ..., refresh: _Optional[bool] = ...) -> None: ...

class ListPlatformVerdictsResponse(_message.Message):
    __slots__ = ("available", "reason", "computed_at", "scenarios", "docker_blocked", "tier_upgrades")
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    DOCKER_BLOCKED_FIELD_NUMBER: _ClassVar[int]
    TIER_UPGRADES_FIELD_NUMBER: _ClassVar[int]
    available: bool
    reason: str
    computed_at: str
    scenarios: _containers.RepeatedCompositeFieldContainer[ScenarioPlatformVerdict]
    docker_blocked: _containers.RepeatedCompositeFieldContainer[FleetDependencyBlock]
    tier_upgrades: _containers.RepeatedCompositeFieldContainer[FleetTierUpgrade]
    def __init__(self, available: _Optional[bool] = ..., reason: _Optional[str] = ..., computed_at: _Optional[str] = ..., scenarios: _Optional[_Iterable[_Union[ScenarioPlatformVerdict, _Mapping]]] = ..., docker_blocked: _Optional[_Iterable[_Union[FleetDependencyBlock, _Mapping]]] = ..., tier_upgrades: _Optional[_Iterable[_Union[FleetTierUpgrade, _Mapping]]] = ...) -> None: ...

class FleetDependencyBlock(_message.Message):
    __slots__ = ("scenario", "host_os", "dependency", "reason")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    HOST_OS_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCY_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    host_os: str
    dependency: str
    reason: str
    def __init__(self, scenario: _Optional[str] = ..., host_os: _Optional[str] = ..., dependency: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class FleetTierUpgrade(_message.Message):
    __slots__ = ("scenario", "host_os", "current_tier", "next_tier", "change", "blocking_dependency")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    HOST_OS_FIELD_NUMBER: _ClassVar[int]
    CURRENT_TIER_FIELD_NUMBER: _ClassVar[int]
    NEXT_TIER_FIELD_NUMBER: _ClassVar[int]
    CHANGE_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_DEPENDENCY_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    host_os: str
    current_tier: str
    next_tier: str
    change: str
    blocking_dependency: str
    def __init__(self, scenario: _Optional[str] = ..., host_os: _Optional[str] = ..., current_tier: _Optional[str] = ..., next_tier: _Optional[str] = ..., change: _Optional[str] = ..., blocking_dependency: _Optional[str] = ...) -> None: ...

class ScenarioPlatformVerdict(_message.Message):
    __slots__ = ("scenario", "platforms", "overridden", "override_reason")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PLATFORMS_FIELD_NUMBER: _ClassVar[int]
    OVERRIDDEN_FIELD_NUMBER: _ClassVar[int]
    OVERRIDE_REASON_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    platforms: _containers.RepeatedCompositeFieldContainer[PlatformVerdict]
    overridden: bool
    override_reason: str
    def __init__(self, scenario: _Optional[str] = ..., platforms: _Optional[_Iterable[_Union[PlatformVerdict, _Mapping]]] = ..., overridden: _Optional[bool] = ..., override_reason: _Optional[str] = ...) -> None: ...

class PlatformVerdict(_message.Message):
    __slots__ = ("host_os", "status", "reason", "blocking_dependency", "derived", "overridden")
    HOST_OS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_DEPENDENCY_FIELD_NUMBER: _ClassVar[int]
    DERIVED_FIELD_NUMBER: _ClassVar[int]
    OVERRIDDEN_FIELD_NUMBER: _ClassVar[int]
    host_os: str
    status: str
    reason: str
    blocking_dependency: str
    derived: bool
    overridden: bool
    def __init__(self, host_os: _Optional[str] = ..., status: _Optional[str] = ..., reason: _Optional[str] = ..., blocking_dependency: _Optional[str] = ..., derived: _Optional[bool] = ..., overridden: _Optional[bool] = ...) -> None: ...
