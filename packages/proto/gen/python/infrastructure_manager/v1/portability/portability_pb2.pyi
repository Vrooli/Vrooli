import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class HostOS(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    HOST_OS_UNSPECIFIED: _ClassVar[HostOS]
    HOST_OS_LINUX: _ClassVar[HostOS]
    HOST_OS_MACOS: _ClassVar[HostOS]
    HOST_OS_WINDOWS: _ClassVar[HostOS]

class ResolutionStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RESOLUTION_STATUS_UNSPECIFIED: _ClassVar[ResolutionStatus]
    RESOLUTION_STATUS_IMPLEMENTED: _ClassVar[ResolutionStatus]
    RESOLUTION_STATUS_DEGRADED: _ClassVar[ResolutionStatus]
    RESOLUTION_STATUS_INELIGIBLE: _ClassVar[ResolutionStatus]
    RESOLUTION_STATUS_UNWIRED: _ClassVar[ResolutionStatus]
    RESOLUTION_STATUS_PEERLESS: _ClassVar[ResolutionStatus]
    RESOLUTION_STATUS_STATUS_INVALID: _ClassVar[ResolutionStatus]
    RESOLUTION_STATUS_CONTROLS_INCOMPLETE: _ClassVar[ResolutionStatus]

class Qualification(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    QUALIFICATION_UNSPECIFIED: _ClassVar[Qualification]
    QUALIFICATION_UNDECLARED: _ClassVar[Qualification]
    QUALIFICATION_INELIGIBLE: _ClassVar[Qualification]
    QUALIFICATION_DEGRADED: _ClassVar[Qualification]
    QUALIFICATION_UNQUALIFIED: _ClassVar[Qualification]
    QUALIFICATION_BUILD_VERIFIED: _ClassVar[Qualification]
    QUALIFICATION_QUALIFIED: _ClassVar[Qualification]

class CapabilitySituation(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CAPABILITY_SITUATION_UNSPECIFIED: _ClassVar[CapabilitySituation]
    CAPABILITY_SITUATION_BUILT_EVERYWHERE: _ClassVar[CapabilitySituation]
    CAPABILITY_SITUATION_NO_WORK_REQUIRED: _ClassVar[CapabilitySituation]
    CAPABILITY_SITUATION_NO_EQUIVALENT_EVER: _ClassVar[CapabilitySituation]
    CAPABILITY_SITUATION_REAL_PEER_NOBODY_WIRED: _ClassVar[CapabilitySituation]
    CAPABILITY_SITUATION_CONTROLS_UNPORTED: _ClassVar[CapabilitySituation]
    CAPABILITY_SITUATION_SCOPED_OUT: _ClassVar[CapabilitySituation]

class Verdict(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VERDICT_UNSPECIFIED: _ClassVar[Verdict]
    VERDICT_ELIGIBLE: _ClassVar[Verdict]
    VERDICT_INELIGIBLE: _ClassVar[Verdict]
    VERDICT_UNKNOWN: _ClassVar[Verdict]

class DeliveryTier(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DELIVERY_TIER_UNSPECIFIED: _ClassVar[DeliveryTier]
    DELIVERY_TIER_LOCAL: _ClassVar[DeliveryTier]
    DELIVERY_TIER_DESKTOP: _ClassVar[DeliveryTier]
    DELIVERY_TIER_MOBILE: _ClassVar[DeliveryTier]
HOST_OS_UNSPECIFIED: HostOS
HOST_OS_LINUX: HostOS
HOST_OS_MACOS: HostOS
HOST_OS_WINDOWS: HostOS
RESOLUTION_STATUS_UNSPECIFIED: ResolutionStatus
RESOLUTION_STATUS_IMPLEMENTED: ResolutionStatus
RESOLUTION_STATUS_DEGRADED: ResolutionStatus
RESOLUTION_STATUS_INELIGIBLE: ResolutionStatus
RESOLUTION_STATUS_UNWIRED: ResolutionStatus
RESOLUTION_STATUS_PEERLESS: ResolutionStatus
RESOLUTION_STATUS_STATUS_INVALID: ResolutionStatus
RESOLUTION_STATUS_CONTROLS_INCOMPLETE: ResolutionStatus
QUALIFICATION_UNSPECIFIED: Qualification
QUALIFICATION_UNDECLARED: Qualification
QUALIFICATION_INELIGIBLE: Qualification
QUALIFICATION_DEGRADED: Qualification
QUALIFICATION_UNQUALIFIED: Qualification
QUALIFICATION_BUILD_VERIFIED: Qualification
QUALIFICATION_QUALIFIED: Qualification
CAPABILITY_SITUATION_UNSPECIFIED: CapabilitySituation
CAPABILITY_SITUATION_BUILT_EVERYWHERE: CapabilitySituation
CAPABILITY_SITUATION_NO_WORK_REQUIRED: CapabilitySituation
CAPABILITY_SITUATION_NO_EQUIVALENT_EVER: CapabilitySituation
CAPABILITY_SITUATION_REAL_PEER_NOBODY_WIRED: CapabilitySituation
CAPABILITY_SITUATION_CONTROLS_UNPORTED: CapabilitySituation
CAPABILITY_SITUATION_SCOPED_OUT: CapabilitySituation
VERDICT_UNSPECIFIED: Verdict
VERDICT_ELIGIBLE: Verdict
VERDICT_INELIGIBLE: Verdict
VERDICT_UNKNOWN: Verdict
DELIVERY_TIER_UNSPECIFIED: DeliveryTier
DELIVERY_TIER_LOCAL: DeliveryTier
DELIVERY_TIER_DESKTOP: DeliveryTier
DELIVERY_TIER_MOBILE: DeliveryTier

class PlatformEntry(_message.Message):
    __slots__ = ("host_os", "architecture", "status", "qualification", "implementer", "mechanism", "reason", "qualification_reason", "has_implementation", "controls", "absent", "absent_controls", "absent_providers", "declarers", "observed_qualification", "observed_qualification_reason", "observed_declarers", "evidence", "policy")
    HOST_OS_FIELD_NUMBER: _ClassVar[int]
    ARCHITECTURE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    QUALIFICATION_FIELD_NUMBER: _ClassVar[int]
    IMPLEMENTER_FIELD_NUMBER: _ClassVar[int]
    MECHANISM_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    QUALIFICATION_REASON_FIELD_NUMBER: _ClassVar[int]
    HAS_IMPLEMENTATION_FIELD_NUMBER: _ClassVar[int]
    CONTROLS_FIELD_NUMBER: _ClassVar[int]
    ABSENT_FIELD_NUMBER: _ClassVar[int]
    ABSENT_CONTROLS_FIELD_NUMBER: _ClassVar[int]
    ABSENT_PROVIDERS_FIELD_NUMBER: _ClassVar[int]
    DECLARERS_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_QUALIFICATION_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_QUALIFICATION_REASON_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_DECLARERS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    POLICY_FIELD_NUMBER: _ClassVar[int]
    host_os: HostOS
    architecture: str
    status: ResolutionStatus
    qualification: Qualification
    implementer: str
    mechanism: str
    reason: str
    qualification_reason: str
    has_implementation: bool
    controls: _containers.RepeatedScalarFieldContainer[str]
    absent: _containers.RepeatedScalarFieldContainer[str]
    absent_controls: _containers.RepeatedScalarFieldContainer[str]
    absent_providers: _containers.RepeatedScalarFieldContainer[str]
    declarers: _containers.RepeatedCompositeFieldContainer[CapabilityDeclarer]
    observed_qualification: Qualification
    observed_qualification_reason: str
    observed_declarers: _containers.RepeatedCompositeFieldContainer[ObservedDeclarer]
    evidence: Evidence
    policy: str
    def __init__(self, host_os: _Optional[_Union[HostOS, str]] = ..., architecture: _Optional[str] = ..., status: _Optional[_Union[ResolutionStatus, str]] = ..., qualification: _Optional[_Union[Qualification, str]] = ..., implementer: _Optional[str] = ..., mechanism: _Optional[str] = ..., reason: _Optional[str] = ..., qualification_reason: _Optional[str] = ..., has_implementation: _Optional[bool] = ..., controls: _Optional[_Iterable[str]] = ..., absent: _Optional[_Iterable[str]] = ..., absent_controls: _Optional[_Iterable[str]] = ..., absent_providers: _Optional[_Iterable[str]] = ..., declarers: _Optional[_Iterable[_Union[CapabilityDeclarer, _Mapping]]] = ..., observed_qualification: _Optional[_Union[Qualification, str]] = ..., observed_qualification_reason: _Optional[str] = ..., observed_declarers: _Optional[_Iterable[_Union[ObservedDeclarer, _Mapping]]] = ..., evidence: _Optional[_Union[Evidence, _Mapping]] = ..., policy: _Optional[str] = ...) -> None: ...

class Evidence(_message.Message):
    __slots__ = ("run_id", "host", "os", "arch", "date", "surface", "artifact_uri")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    HOST_FIELD_NUMBER: _ClassVar[int]
    OS_FIELD_NUMBER: _ClassVar[int]
    ARCH_FIELD_NUMBER: _ClassVar[int]
    DATE_FIELD_NUMBER: _ClassVar[int]
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_URI_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    host: str
    os: str
    arch: str
    date: str
    surface: str
    artifact_uri: str
    def __init__(self, run_id: _Optional[str] = ..., host: _Optional[str] = ..., os: _Optional[str] = ..., arch: _Optional[str] = ..., date: _Optional[str] = ..., surface: _Optional[str] = ..., artifact_uri: _Optional[str] = ...) -> None: ...

class CapabilityDeclarer(_message.Message):
    __slots__ = ("name", "role", "declared_status", "resolved", "reason")
    NAME_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    DECLARED_STATUS_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    name: str
    role: str
    declared_status: str
    resolved: bool
    reason: str
    def __init__(self, name: _Optional[str] = ..., role: _Optional[str] = ..., declared_status: _Optional[str] = ..., resolved: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class ObservedDeclarer(_message.Message):
    __slots__ = ("name", "state", "qualification", "reason")
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    QUALIFICATION_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    name: str
    state: str
    qualification: Qualification
    reason: str
    def __init__(self, name: _Optional[str] = ..., state: _Optional[str] = ..., qualification: _Optional[_Union[Qualification, str]] = ..., reason: _Optional[str] = ...) -> None: ...

class CapabilityEntry(_message.Message):
    __slots__ = ("capability", "situation", "situation_reason", "platforms")
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    SITUATION_FIELD_NUMBER: _ClassVar[int]
    SITUATION_REASON_FIELD_NUMBER: _ClassVar[int]
    PLATFORMS_FIELD_NUMBER: _ClassVar[int]
    capability: str
    situation: CapabilitySituation
    situation_reason: str
    platforms: _containers.RepeatedCompositeFieldContainer[PlatformEntry]
    def __init__(self, capability: _Optional[str] = ..., situation: _Optional[_Union[CapabilitySituation, str]] = ..., situation_reason: _Optional[str] = ..., platforms: _Optional[_Iterable[_Union[PlatformEntry, _Mapping]]] = ...) -> None: ...

class Grid(_message.Message):
    __slots__ = ("capabilities", "manifest_root", "manifests_read", "computed_at", "observed_safeguards", "native_evidence")
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_ROOT_FIELD_NUMBER: _ClassVar[int]
    MANIFESTS_READ_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_SAFEGUARDS_FIELD_NUMBER: _ClassVar[int]
    NATIVE_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    capabilities: _containers.RepeatedCompositeFieldContainer[CapabilityEntry]
    manifest_root: str
    manifests_read: int
    computed_at: _timestamp_pb2.Timestamp
    observed_safeguards: _containers.RepeatedCompositeFieldContainer[ObservedSafeguard]
    native_evidence: _containers.RepeatedCompositeFieldContainer[NativeEvidence]
    def __init__(self, capabilities: _Optional[_Iterable[_Union[CapabilityEntry, _Mapping]]] = ..., manifest_root: _Optional[str] = ..., manifests_read: _Optional[int] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., observed_safeguards: _Optional[_Iterable[_Union[ObservedSafeguard, _Mapping]]] = ..., native_evidence: _Optional[_Iterable[_Union[NativeEvidence, _Mapping]]] = ...) -> None: ...

class NativeEvidence(_message.Message):
    __slots__ = ("kind", "host_os", "architecture", "commit", "generated_at", "passed", "source", "run_id", "host", "surface", "artifact_uri", "capabilities")
    KIND_FIELD_NUMBER: _ClassVar[int]
    HOST_OS_FIELD_NUMBER: _ClassVar[int]
    ARCHITECTURE_FIELD_NUMBER: _ClassVar[int]
    COMMIT_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    HOST_FIELD_NUMBER: _ClassVar[int]
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_URI_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    kind: str
    host_os: HostOS
    architecture: str
    commit: str
    generated_at: _timestamp_pb2.Timestamp
    passed: bool
    source: str
    run_id: str
    host: str
    surface: str
    artifact_uri: str
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, kind: _Optional[str] = ..., host_os: _Optional[_Union[HostOS, str]] = ..., architecture: _Optional[str] = ..., commit: _Optional[str] = ..., generated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., passed: _Optional[bool] = ..., source: _Optional[str] = ..., run_id: _Optional[str] = ..., host: _Optional[str] = ..., surface: _Optional[str] = ..., artifact_uri: _Optional[str] = ..., capabilities: _Optional[_Iterable[str]] = ...) -> None: ...

class ObservedSafeguard(_message.Message):
    __slots__ = ("name", "capability", "capability_role", "platforms", "support_class", "execution_state", "notes", "observed_at")
    NAME_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_ROLE_FIELD_NUMBER: _ClassVar[int]
    PLATFORMS_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_CLASS_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_STATE_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    name: str
    capability: str
    capability_role: str
    platforms: _containers.RepeatedScalarFieldContainer[str]
    support_class: str
    execution_state: str
    notes: _containers.RepeatedScalarFieldContainer[str]
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, name: _Optional[str] = ..., capability: _Optional[str] = ..., capability_role: _Optional[str] = ..., platforms: _Optional[_Iterable[str]] = ..., support_class: _Optional[str] = ..., execution_state: _Optional[str] = ..., notes: _Optional[_Iterable[str]] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetGridRequest(_message.Message):
    __slots__ = ("capabilities",)
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, capabilities: _Optional[_Iterable[str]] = ...) -> None: ...

class GetGridResponse(_message.Message):
    __slots__ = ("grid",)
    GRID_FIELD_NUMBER: _ClassVar[int]
    grid: Grid
    def __init__(self, grid: _Optional[_Union[Grid, _Mapping]] = ...) -> None: ...

class GetCapabilityRequest(_message.Message):
    __slots__ = ("capability",)
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    capability: str
    def __init__(self, capability: _Optional[str] = ...) -> None: ...

class GetCapabilityResponse(_message.Message):
    __slots__ = ("capability", "manifest_root", "computed_at")
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_ROOT_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    capability: CapabilityEntry
    manifest_root: str
    computed_at: _timestamp_pb2.Timestamp
    def __init__(self, capability: _Optional[_Union[CapabilityEntry, _Mapping]] = ..., manifest_root: _Optional[str] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListSituationsRequest(_message.Message):
    __slots__ = ("situation",)
    SITUATION_FIELD_NUMBER: _ClassVar[int]
    situation: CapabilitySituation
    def __init__(self, situation: _Optional[_Union[CapabilitySituation, str]] = ...) -> None: ...

class ListSituationsResponse(_message.Message):
    __slots__ = ("capabilities", "manifest_root", "computed_at")
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_ROOT_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    capabilities: _containers.RepeatedCompositeFieldContainer[CapabilityEntry]
    manifest_root: str
    computed_at: _timestamp_pb2.Timestamp
    def __init__(self, capabilities: _Optional[_Iterable[_Union[CapabilityEntry, _Mapping]]] = ..., manifest_root: _Optional[str] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class DependencyReason(_message.Message):
    __slots__ = ("code", "dependency", "requirement", "message")
    CODE_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCY_FIELD_NUMBER: _ClassVar[int]
    REQUIREMENT_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    code: str
    dependency: str
    requirement: str
    message: str
    def __init__(self, code: _Optional[str] = ..., dependency: _Optional[str] = ..., requirement: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class DependencyResult(_message.Message):
    __slots__ = ("kind", "name", "required", "verdict", "reasons")
    KIND_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    REASONS_FIELD_NUMBER: _ClassVar[int]
    kind: str
    name: str
    required: bool
    verdict: Verdict
    reasons: _containers.RepeatedCompositeFieldContainer[DependencyReason]
    def __init__(self, kind: _Optional[str] = ..., name: _Optional[str] = ..., required: _Optional[bool] = ..., verdict: _Optional[_Union[Verdict, str]] = ..., reasons: _Optional[_Iterable[_Union[DependencyReason, _Mapping]]] = ...) -> None: ...

class ScenarioBlock(_message.Message):
    __slots__ = ("scenario", "host_os", "dependencies")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    HOST_OS_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    host_os: HostOS
    dependencies: _containers.RepeatedCompositeFieldContainer[DependencyResult]
    def __init__(self, scenario: _Optional[str] = ..., host_os: _Optional[_Union[HostOS, str]] = ..., dependencies: _Optional[_Iterable[_Union[DependencyResult, _Mapping]]] = ...) -> None: ...

class ScenarioPeerless(_message.Message):
    __slots__ = ("scenario", "host_os", "capabilities")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    HOST_OS_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    host_os: HostOS
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., host_os: _Optional[_Union[HostOS, str]] = ..., capabilities: _Optional[_Iterable[str]] = ...) -> None: ...

class TierUpgrade(_message.Message):
    __slots__ = ("scenario", "host_os", "current_tier", "next_tier", "single_change", "blocking_dependency")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    HOST_OS_FIELD_NUMBER: _ClassVar[int]
    CURRENT_TIER_FIELD_NUMBER: _ClassVar[int]
    NEXT_TIER_FIELD_NUMBER: _ClassVar[int]
    SINGLE_CHANGE_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_DEPENDENCY_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    host_os: HostOS
    current_tier: DeliveryTier
    next_tier: DeliveryTier
    single_change: str
    blocking_dependency: str
    def __init__(self, scenario: _Optional[str] = ..., host_os: _Optional[_Union[HostOS, str]] = ..., current_tier: _Optional[_Union[DeliveryTier, str]] = ..., next_tier: _Optional[_Union[DeliveryTier, str]] = ..., single_change: _Optional[str] = ..., blocking_dependency: _Optional[str] = ...) -> None: ...

class DesktopBundlingVerdict(_message.Message):
    __slots__ = ("resources", "host_required", "vendorable", "prohibited", "unknown", "database_blocked", "reason")
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    HOST_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    VENDORABLE_FIELD_NUMBER: _ClassVar[int]
    PROHIBITED_FIELD_NUMBER: _ClassVar[int]
    UNKNOWN_FIELD_NUMBER: _ClassVar[int]
    DATABASE_BLOCKED_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    resources: int
    host_required: int
    vendorable: int
    prohibited: int
    unknown: int
    database_blocked: bool
    reason: str
    def __init__(self, resources: _Optional[int] = ..., host_required: _Optional[int] = ..., vendorable: _Optional[int] = ..., prohibited: _Optional[int] = ..., unknown: _Optional[int] = ..., database_blocked: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class FleetReadout(_message.Message):
    __slots__ = ("blocked_by_os", "docker_blocked", "peerless", "tier_upgrades", "desktop_bundling", "manifest_root", "computed_at", "available", "reason")
    BLOCKED_BY_OS_FIELD_NUMBER: _ClassVar[int]
    DOCKER_BLOCKED_FIELD_NUMBER: _ClassVar[int]
    PEERLESS_FIELD_NUMBER: _ClassVar[int]
    TIER_UPGRADES_FIELD_NUMBER: _ClassVar[int]
    DESKTOP_BUNDLING_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_ROOT_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    blocked_by_os: _containers.RepeatedCompositeFieldContainer[ScenarioBlock]
    docker_blocked: _containers.RepeatedCompositeFieldContainer[ScenarioBlock]
    peerless: _containers.RepeatedCompositeFieldContainer[ScenarioPeerless]
    tier_upgrades: _containers.RepeatedCompositeFieldContainer[TierUpgrade]
    desktop_bundling: DesktopBundlingVerdict
    manifest_root: str
    computed_at: _timestamp_pb2.Timestamp
    available: bool
    reason: str
    def __init__(self, blocked_by_os: _Optional[_Iterable[_Union[ScenarioBlock, _Mapping]]] = ..., docker_blocked: _Optional[_Iterable[_Union[ScenarioBlock, _Mapping]]] = ..., peerless: _Optional[_Iterable[_Union[ScenarioPeerless, _Mapping]]] = ..., tier_upgrades: _Optional[_Iterable[_Union[TierUpgrade, _Mapping]]] = ..., desktop_bundling: _Optional[_Union[DesktopBundlingVerdict, _Mapping]] = ..., manifest_root: _Optional[str] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., available: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class GetFleetRequest(_message.Message):
    __slots__ = ("view",)
    VIEW_FIELD_NUMBER: _ClassVar[int]
    view: str
    def __init__(self, view: _Optional[str] = ...) -> None: ...

class GetFleetResponse(_message.Message):
    __slots__ = ("fleet",)
    FLEET_FIELD_NUMBER: _ClassVar[int]
    fleet: FleetReadout
    def __init__(self, fleet: _Optional[_Union[FleetReadout, _Mapping]] = ...) -> None: ...
