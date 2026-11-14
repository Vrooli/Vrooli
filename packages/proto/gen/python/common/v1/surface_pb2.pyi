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

class SurfaceKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SURFACE_KIND_UNSPECIFIED: _ClassVar[SurfaceKind]
    SURFACE_KIND_SCENARIO: _ClassVar[SurfaceKind]
    SURFACE_KIND_TERMINAL: _ClassVar[SurfaceKind]
    SURFACE_KIND_DESKTOP: _ClassVar[SurfaceKind]
    SURFACE_KIND_BROWSER: _ClassVar[SurfaceKind]
    SURFACE_KIND_DEVICE_PANEL: _ClassVar[SurfaceKind]

class SurfaceCapabilityState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SURFACE_CAPABILITY_STATE_UNSPECIFIED: _ClassVar[SurfaceCapabilityState]
    SURFACE_CAPABILITY_STATE_READY: _ClassVar[SurfaceCapabilityState]
    SURFACE_CAPABILITY_STATE_MISSING: _ClassVar[SurfaceCapabilityState]
    SURFACE_CAPABILITY_STATE_UNSUPPORTED: _ClassVar[SurfaceCapabilityState]
    SURFACE_CAPABILITY_STATE_UNKNOWN: _ClassVar[SurfaceCapabilityState]
    SURFACE_CAPABILITY_STATE_DENIED: _ClassVar[SurfaceCapabilityState]
SURFACE_KIND_UNSPECIFIED: SurfaceKind
SURFACE_KIND_SCENARIO: SurfaceKind
SURFACE_KIND_TERMINAL: SurfaceKind
SURFACE_KIND_DESKTOP: SurfaceKind
SURFACE_KIND_BROWSER: SurfaceKind
SURFACE_KIND_DEVICE_PANEL: SurfaceKind
SURFACE_CAPABILITY_STATE_UNSPECIFIED: SurfaceCapabilityState
SURFACE_CAPABILITY_STATE_READY: SurfaceCapabilityState
SURFACE_CAPABILITY_STATE_MISSING: SurfaceCapabilityState
SURFACE_CAPABILITY_STATE_UNSUPPORTED: SurfaceCapabilityState
SURFACE_CAPABILITY_STATE_UNKNOWN: SurfaceCapabilityState
SURFACE_CAPABILITY_STATE_DENIED: SurfaceCapabilityState

class TargetRef(_message.Message):
    __slots__ = ("owner_scenario", "resource_id", "host_node_id")
    OWNER_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_ID_FIELD_NUMBER: _ClassVar[int]
    HOST_NODE_ID_FIELD_NUMBER: _ClassVar[int]
    owner_scenario: str
    resource_id: str
    host_node_id: str
    def __init__(self, owner_scenario: _Optional[str] = ..., resource_id: _Optional[str] = ..., host_node_id: _Optional[str] = ...) -> None: ...

class SurfaceRef(_message.Message):
    __slots__ = ("target", "owner_scenario", "surface_id")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    OWNER_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SURFACE_ID_FIELD_NUMBER: _ClassVar[int]
    target: TargetRef
    owner_scenario: str
    surface_id: str
    def __init__(self, target: _Optional[_Union[TargetRef, _Mapping]] = ..., owner_scenario: _Optional[str] = ..., surface_id: _Optional[str] = ...) -> None: ...

class SessionRef(_message.Message):
    __slots__ = ("surface", "session_id", "desktop_session_id")
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    DESKTOP_SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    surface: SurfaceRef
    session_id: str
    desktop_session_id: str
    def __init__(self, surface: _Optional[_Union[SurfaceRef, _Mapping]] = ..., session_id: _Optional[str] = ..., desktop_session_id: _Optional[str] = ...) -> None: ...

class SurfaceCapabilityFact(_message.Message):
    __slots__ = ("capability", "state", "reason_code", "evidence_id", "observed_at", "expires_at")
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_ID_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    capability: str
    state: SurfaceCapabilityState
    reason_code: str
    evidence_id: str
    observed_at: _timestamp_pb2.Timestamp
    expires_at: _timestamp_pb2.Timestamp
    def __init__(self, capability: _Optional[str] = ..., state: _Optional[_Union[SurfaceCapabilityState, str]] = ..., reason_code: _Optional[str] = ..., evidence_id: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class SurfaceDescriptor(_message.Message):
    __slots__ = ("ref", "kind", "display_label", "capabilities", "protocol_versions", "desktop_session_id", "display_ids")
    REF_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_LABEL_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    PROTOCOL_VERSIONS_FIELD_NUMBER: _ClassVar[int]
    DESKTOP_SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_IDS_FIELD_NUMBER: _ClassVar[int]
    ref: SurfaceRef
    kind: SurfaceKind
    display_label: str
    capabilities: _containers.RepeatedCompositeFieldContainer[SurfaceCapabilityFact]
    protocol_versions: _containers.RepeatedScalarFieldContainer[str]
    desktop_session_id: str
    display_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, ref: _Optional[_Union[SurfaceRef, _Mapping]] = ..., kind: _Optional[_Union[SurfaceKind, str]] = ..., display_label: _Optional[str] = ..., capabilities: _Optional[_Iterable[_Union[SurfaceCapabilityFact, _Mapping]]] = ..., protocol_versions: _Optional[_Iterable[str]] = ..., desktop_session_id: _Optional[str] = ..., display_ids: _Optional[_Iterable[str]] = ...) -> None: ...
