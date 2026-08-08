import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ValidationTargetCapability(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VALIDATION_TARGET_CAPABILITY_UNSPECIFIED: _ClassVar[ValidationTargetCapability]
    VALIDATION_TARGET_CAPABILITY_ELECTRON_CDP: _ClassVar[ValidationTargetCapability]
    VALIDATION_TARGET_CAPABILITY_NATIVE_WINDOW: _ClassVar[ValidationTargetCapability]
    VALIDATION_TARGET_CAPABILITY_FILE_PICKER: _ClassVar[ValidationTargetCapability]
    VALIDATION_TARGET_CAPABILITY_TRAY: _ClassVar[ValidationTargetCapability]
    VALIDATION_TARGET_CAPABILITY_UPDATER: _ClassVar[ValidationTargetCapability]
    VALIDATION_TARGET_CAPABILITY_PROCESS_METRICS: _ClassVar[ValidationTargetCapability]
    VALIDATION_TARGET_CAPABILITY_CRASH_RECOVERY: _ClassVar[ValidationTargetCapability]
    VALIDATION_TARGET_CAPABILITY_OFFLINE_NETWORK: _ClassVar[ValidationTargetCapability]
    VALIDATION_TARGET_CAPABILITY_NETWORK_CONTROL: _ClassVar[ValidationTargetCapability]
    VALIDATION_TARGET_CAPABILITY_CREDENTIAL_CONTROL: _ClassVar[ValidationTargetCapability]
    VALIDATION_TARGET_CAPABILITY_PROVIDER_CONTROL: _ClassVar[ValidationTargetCapability]
    VALIDATION_TARGET_CAPABILITY_COMMUNICATION_PEER: _ClassVar[ValidationTargetCapability]
    VALIDATION_TARGET_CAPABILITY_NATIVE_MENU: _ClassVar[ValidationTargetCapability]
    VALIDATION_TARGET_CAPABILITY_NOTIFICATION: _ClassVar[ValidationTargetCapability]
    VALIDATION_TARGET_CAPABILITY_MULTI_WINDOW: _ClassVar[ValidationTargetCapability]
    VALIDATION_TARGET_CAPABILITY_CLEAN_SHUTDOWN: _ClassVar[ValidationTargetCapability]
    VALIDATION_TARGET_CAPABILITY_UPDATE_FEED: _ClassVar[ValidationTargetCapability]

class ValidationEnvironmentProfile(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VALIDATION_ENVIRONMENT_PROFILE_UNSPECIFIED: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_NORMAL: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_OFFLINE: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_SLOW_NETWORK: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_MISSING_CREDENTIAL: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_PROVIDER_FAILURE: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_UPDATE_INTERRUPTED: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_CRASH_RECOVERY: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_HIGH_LATENCY: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_PACKET_LOSS: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_PROVIDER_UNAVAILABLE: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_RECONNECT: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_EXPIRED_CREDENTIAL: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_UNAVAILABLE_CREDENTIAL: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_WRONG_SCOPE_CREDENTIAL: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_UPDATE_DISCOVERY: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_UPDATE_DOWNLOAD: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_UPDATE_VERIFICATION: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_UPDATE_ROLLBACK: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_UPDATE_RESTART: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_UPDATE_FAILURE: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_COMMUNICATION_BUNDLED_PRIVATE: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_COMMUNICATION_TIER_ONE: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_COMMUNICATION_SHARED_PROVIDER: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_COMMUNICATION_FALLBACK: _ClassVar[ValidationEnvironmentProfile]
    VALIDATION_ENVIRONMENT_PROFILE_COMMUNICATION_TIER_TWO_PEER: _ClassVar[ValidationEnvironmentProfile]

class ValidationDisposition(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VALIDATION_DISPOSITION_UNSPECIFIED: _ClassVar[ValidationDisposition]
    VALIDATION_DISPOSITION_PASS: _ClassVar[ValidationDisposition]
    VALIDATION_DISPOSITION_FAILED: _ClassVar[ValidationDisposition]
    VALIDATION_DISPOSITION_DEGRADED: _ClassVar[ValidationDisposition]
    VALIDATION_DISPOSITION_UNAVAILABLE: _ClassVar[ValidationDisposition]
    VALIDATION_DISPOSITION_UNSUPPORTED: _ClassVar[ValidationDisposition]
    VALIDATION_DISPOSITION_REFUSED: _ClassVar[ValidationDisposition]
    VALIDATION_DISPOSITION_NOT_RUN: _ClassVar[ValidationDisposition]
VALIDATION_TARGET_CAPABILITY_UNSPECIFIED: ValidationTargetCapability
VALIDATION_TARGET_CAPABILITY_ELECTRON_CDP: ValidationTargetCapability
VALIDATION_TARGET_CAPABILITY_NATIVE_WINDOW: ValidationTargetCapability
VALIDATION_TARGET_CAPABILITY_FILE_PICKER: ValidationTargetCapability
VALIDATION_TARGET_CAPABILITY_TRAY: ValidationTargetCapability
VALIDATION_TARGET_CAPABILITY_UPDATER: ValidationTargetCapability
VALIDATION_TARGET_CAPABILITY_PROCESS_METRICS: ValidationTargetCapability
VALIDATION_TARGET_CAPABILITY_CRASH_RECOVERY: ValidationTargetCapability
VALIDATION_TARGET_CAPABILITY_OFFLINE_NETWORK: ValidationTargetCapability
VALIDATION_TARGET_CAPABILITY_NETWORK_CONTROL: ValidationTargetCapability
VALIDATION_TARGET_CAPABILITY_CREDENTIAL_CONTROL: ValidationTargetCapability
VALIDATION_TARGET_CAPABILITY_PROVIDER_CONTROL: ValidationTargetCapability
VALIDATION_TARGET_CAPABILITY_COMMUNICATION_PEER: ValidationTargetCapability
VALIDATION_TARGET_CAPABILITY_NATIVE_MENU: ValidationTargetCapability
VALIDATION_TARGET_CAPABILITY_NOTIFICATION: ValidationTargetCapability
VALIDATION_TARGET_CAPABILITY_MULTI_WINDOW: ValidationTargetCapability
VALIDATION_TARGET_CAPABILITY_CLEAN_SHUTDOWN: ValidationTargetCapability
VALIDATION_TARGET_CAPABILITY_UPDATE_FEED: ValidationTargetCapability
VALIDATION_ENVIRONMENT_PROFILE_UNSPECIFIED: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_NORMAL: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_OFFLINE: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_SLOW_NETWORK: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_MISSING_CREDENTIAL: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_PROVIDER_FAILURE: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_UPDATE_INTERRUPTED: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_CRASH_RECOVERY: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_HIGH_LATENCY: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_PACKET_LOSS: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_PROVIDER_UNAVAILABLE: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_RECONNECT: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_EXPIRED_CREDENTIAL: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_UNAVAILABLE_CREDENTIAL: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_WRONG_SCOPE_CREDENTIAL: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_UPDATE_DISCOVERY: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_UPDATE_DOWNLOAD: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_UPDATE_VERIFICATION: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_UPDATE_ROLLBACK: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_UPDATE_RESTART: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_UPDATE_FAILURE: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_COMMUNICATION_BUNDLED_PRIVATE: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_COMMUNICATION_TIER_ONE: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_COMMUNICATION_SHARED_PROVIDER: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_COMMUNICATION_FALLBACK: ValidationEnvironmentProfile
VALIDATION_ENVIRONMENT_PROFILE_COMMUNICATION_TIER_TWO_PEER: ValidationEnvironmentProfile
VALIDATION_DISPOSITION_UNSPECIFIED: ValidationDisposition
VALIDATION_DISPOSITION_PASS: ValidationDisposition
VALIDATION_DISPOSITION_FAILED: ValidationDisposition
VALIDATION_DISPOSITION_DEGRADED: ValidationDisposition
VALIDATION_DISPOSITION_UNAVAILABLE: ValidationDisposition
VALIDATION_DISPOSITION_UNSUPPORTED: ValidationDisposition
VALIDATION_DISPOSITION_REFUSED: ValidationDisposition
VALIDATION_DISPOSITION_NOT_RUN: ValidationDisposition

class ElectronTarget(_message.Message):
    __slots__ = ("target_id", "scenario_name", "artifact_digest", "process_id", "cdp_endpoint", "cdp_transport", "loopback_only", "authenticated", "renderer_id", "bridge_node_id", "launched_at", "context_id", "renderer_url", "renderer_title")
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    PROCESS_ID_FIELD_NUMBER: _ClassVar[int]
    CDP_ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    CDP_TRANSPORT_FIELD_NUMBER: _ClassVar[int]
    LOOPBACK_ONLY_FIELD_NUMBER: _ClassVar[int]
    AUTHENTICATED_FIELD_NUMBER: _ClassVar[int]
    RENDERER_ID_FIELD_NUMBER: _ClassVar[int]
    BRIDGE_NODE_ID_FIELD_NUMBER: _ClassVar[int]
    LAUNCHED_AT_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_ID_FIELD_NUMBER: _ClassVar[int]
    RENDERER_URL_FIELD_NUMBER: _ClassVar[int]
    RENDERER_TITLE_FIELD_NUMBER: _ClassVar[int]
    target_id: str
    scenario_name: str
    artifact_digest: str
    process_id: str
    cdp_endpoint: str
    cdp_transport: str
    loopback_only: bool
    authenticated: bool
    renderer_id: str
    bridge_node_id: str
    launched_at: _timestamp_pb2.Timestamp
    context_id: str
    renderer_url: str
    renderer_title: str
    def __init__(self, target_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., artifact_digest: _Optional[str] = ..., process_id: _Optional[str] = ..., cdp_endpoint: _Optional[str] = ..., cdp_transport: _Optional[str] = ..., loopback_only: _Optional[bool] = ..., authenticated: _Optional[bool] = ..., renderer_id: _Optional[str] = ..., bridge_node_id: _Optional[str] = ..., launched_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., context_id: _Optional[str] = ..., renderer_url: _Optional[str] = ..., renderer_title: _Optional[str] = ...) -> None: ...

class ElectronLaunchRequest(_message.Message):
    __slots__ = ("scenario_name", "artifact_path", "artifact_digest", "validation_context_id", "enable_cdp", "require_loopback_cdp", "bridge_node_id")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_PATH_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_CONTEXT_ID_FIELD_NUMBER: _ClassVar[int]
    ENABLE_CDP_FIELD_NUMBER: _ClassVar[int]
    REQUIRE_LOOPBACK_CDP_FIELD_NUMBER: _ClassVar[int]
    BRIDGE_NODE_ID_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    artifact_path: str
    artifact_digest: str
    validation_context_id: str
    enable_cdp: bool
    require_loopback_cdp: bool
    bridge_node_id: str
    def __init__(self, scenario_name: _Optional[str] = ..., artifact_path: _Optional[str] = ..., artifact_digest: _Optional[str] = ..., validation_context_id: _Optional[str] = ..., enable_cdp: _Optional[bool] = ..., require_loopback_cdp: _Optional[bool] = ..., bridge_node_id: _Optional[str] = ...) -> None: ...

class ValidationContext(_message.Message):
    __slots__ = ("context_id", "scenario_name", "artifact_digest", "target_id", "journey_id", "profile_id", "isolation_lease_id", "expires_at")
    CONTEXT_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    JOURNEY_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    ISOLATION_LEASE_ID_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    context_id: str
    scenario_name: str
    artifact_digest: str
    target_id: str
    journey_id: str
    profile_id: str
    isolation_lease_id: str
    expires_at: _timestamp_pb2.Timestamp
    def __init__(self, context_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., artifact_digest: _Optional[str] = ..., target_id: _Optional[str] = ..., journey_id: _Optional[str] = ..., profile_id: _Optional[str] = ..., isolation_lease_id: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ValidationCell(_message.Message):
    __slots__ = ("cell_id", "scenario_name", "artifact_digest", "target_id", "journey_id", "environment_profile", "disposition", "required_capabilities", "reason", "isolation_lease_id", "evidence", "created_at", "completed_at", "required", "applicable")
    CELL_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    JOURNEY_ID_FIELD_NUMBER: _ClassVar[int]
    ENVIRONMENT_PROFILE_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    ISOLATION_LEASE_ID_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    APPLICABLE_FIELD_NUMBER: _ClassVar[int]
    cell_id: str
    scenario_name: str
    artifact_digest: str
    target_id: str
    journey_id: str
    environment_profile: ValidationEnvironmentProfile
    disposition: ValidationDisposition
    required_capabilities: _containers.RepeatedScalarFieldContainer[ValidationTargetCapability]
    reason: str
    isolation_lease_id: str
    evidence: _containers.RepeatedCompositeFieldContainer[LayeredEvidence]
    created_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    required: bool
    applicable: bool
    def __init__(self, cell_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., artifact_digest: _Optional[str] = ..., target_id: _Optional[str] = ..., journey_id: _Optional[str] = ..., environment_profile: _Optional[_Union[ValidationEnvironmentProfile, str]] = ..., disposition: _Optional[_Union[ValidationDisposition, str]] = ..., required_capabilities: _Optional[_Iterable[_Union[ValidationTargetCapability, str]]] = ..., reason: _Optional[str] = ..., isolation_lease_id: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[LayeredEvidence, _Mapping]]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., required: _Optional[bool] = ..., applicable: _Optional[bool] = ...) -> None: ...

class LayeredEvidence(_message.Message):
    __slots__ = ("kind", "evidence_id", "uri", "sha256", "media_type", "redacted")
    class Kind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        KIND_UNSPECIFIED: _ClassVar[LayeredEvidence.Kind]
        KIND_BAS_WORKFLOW: _ClassVar[LayeredEvidence.Kind]
        KIND_DESKTOP_RUNTIME: _ClassVar[LayeredEvidence.Kind]
        KIND_TARGET: _ClassVar[LayeredEvidence.Kind]
        KIND_MACHINE_ASSERTION: _ClassVar[LayeredEvidence.Kind]
    KIND_UNSPECIFIED: LayeredEvidence.Kind
    KIND_BAS_WORKFLOW: LayeredEvidence.Kind
    KIND_DESKTOP_RUNTIME: LayeredEvidence.Kind
    KIND_TARGET: LayeredEvidence.Kind
    KIND_MACHINE_ASSERTION: LayeredEvidence.Kind
    KIND_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_ID_FIELD_NUMBER: _ClassVar[int]
    URI_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    REDACTED_FIELD_NUMBER: _ClassVar[int]
    kind: LayeredEvidence.Kind
    evidence_id: str
    uri: str
    sha256: str
    media_type: str
    redacted: bool
    def __init__(self, kind: _Optional[_Union[LayeredEvidence.Kind, str]] = ..., evidence_id: _Optional[str] = ..., uri: _Optional[str] = ..., sha256: _Optional[str] = ..., media_type: _Optional[str] = ..., redacted: _Optional[bool] = ...) -> None: ...

class JourneyCatalogItem(_message.Message):
    __slots__ = ("journey_id", "display_name", "source_path", "execution_mode", "required", "required_capabilities")
    JOURNEY_ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_MODE_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    journey_id: str
    display_name: str
    source_path: str
    execution_mode: str
    required: bool
    required_capabilities: _containers.RepeatedScalarFieldContainer[ValidationTargetCapability]
    def __init__(self, journey_id: _Optional[str] = ..., display_name: _Optional[str] = ..., source_path: _Optional[str] = ..., execution_mode: _Optional[str] = ..., required: _Optional[bool] = ..., required_capabilities: _Optional[_Iterable[_Union[ValidationTargetCapability, str]]] = ...) -> None: ...

class ValidationTargetDescriptor(_message.Message):
    __slots__ = ("target_id", "display_name", "capabilities", "available", "reason")
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    target_id: str
    display_name: str
    capabilities: _containers.RepeatedScalarFieldContainer[ValidationTargetCapability]
    available: bool
    reason: str
    def __init__(self, target_id: _Optional[str] = ..., display_name: _Optional[str] = ..., capabilities: _Optional[_Iterable[_Union[ValidationTargetCapability, str]]] = ..., available: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class ValidationMatrix(_message.Message):
    __slots__ = ("matrix_id", "scenario_name", "artifact_digest", "journeys", "targets", "environment_profiles", "cells", "created_at")
    MATRIX_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    JOURNEYS_FIELD_NUMBER: _ClassVar[int]
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    ENVIRONMENT_PROFILES_FIELD_NUMBER: _ClassVar[int]
    CELLS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    matrix_id: str
    scenario_name: str
    artifact_digest: str
    journeys: _containers.RepeatedCompositeFieldContainer[JourneyCatalogItem]
    targets: _containers.RepeatedCompositeFieldContainer[ValidationTargetDescriptor]
    environment_profiles: _containers.RepeatedScalarFieldContainer[ValidationEnvironmentProfile]
    cells: _containers.RepeatedCompositeFieldContainer[ValidationCell]
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, matrix_id: _Optional[str] = ..., scenario_name: _Optional[str] = ..., artifact_digest: _Optional[str] = ..., journeys: _Optional[_Iterable[_Union[JourneyCatalogItem, _Mapping]]] = ..., targets: _Optional[_Iterable[_Union[ValidationTargetDescriptor, _Mapping]]] = ..., environment_profiles: _Optional[_Iterable[_Union[ValidationEnvironmentProfile, str]]] = ..., cells: _Optional[_Iterable[_Union[ValidationCell, _Mapping]]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ReleaseGate(_message.Message):
    __slots__ = ("matrix_id", "disposition", "passed", "required_cell_count", "passing_cell_count", "missing_cell_ids", "failed_cell_ids", "reason")
    MATRIX_ID_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_CELL_COUNT_FIELD_NUMBER: _ClassVar[int]
    PASSING_CELL_COUNT_FIELD_NUMBER: _ClassVar[int]
    MISSING_CELL_IDS_FIELD_NUMBER: _ClassVar[int]
    FAILED_CELL_IDS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    matrix_id: str
    disposition: ValidationDisposition
    passed: bool
    required_cell_count: int
    passing_cell_count: int
    missing_cell_ids: _containers.RepeatedScalarFieldContainer[str]
    failed_cell_ids: _containers.RepeatedScalarFieldContainer[str]
    reason: str
    def __init__(self, matrix_id: _Optional[str] = ..., disposition: _Optional[_Union[ValidationDisposition, str]] = ..., passed: _Optional[bool] = ..., required_cell_count: _Optional[int] = ..., passing_cell_count: _Optional[int] = ..., missing_cell_ids: _Optional[_Iterable[str]] = ..., failed_cell_ids: _Optional[_Iterable[str]] = ..., reason: _Optional[str] = ...) -> None: ...
