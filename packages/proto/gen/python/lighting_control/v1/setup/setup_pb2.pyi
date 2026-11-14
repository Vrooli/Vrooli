from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class InstallationSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    INSTALLATION_SOURCE_UNSPECIFIED: _ClassVar[InstallationSource]
    INSTALLATION_SOURCE_MANAGED: _ClassVar[InstallationSource]
    INSTALLATION_SOURCE_EXISTING: _ClassVar[InstallationSource]
INSTALLATION_SOURCE_UNSPECIFIED: InstallationSource
INSTALLATION_SOURCE_MANAGED: InstallationSource
INSTALLATION_SOURCE_EXISTING: InstallationSource

class LightingSelection(_message.Message):
    __slots__ = ("source", "version", "sha256", "device_policy_id")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    DEVICE_POLICY_ID_FIELD_NUMBER: _ClassVar[int]
    source: InstallationSource
    version: str
    sha256: str
    device_policy_id: str
    def __init__(self, source: _Optional[_Union[InstallationSource, str]] = ..., version: _Optional[str] = ..., sha256: _Optional[str] = ..., device_policy_id: _Optional[str] = ...) -> None: ...

class GetSetupStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SetupLightingRequest(_message.Message):
    __slots__ = ("selection",)
    SELECTION_FIELD_NUMBER: _ClassVar[int]
    selection: LightingSelection
    def __init__(self, selection: _Optional[_Union[LightingSelection, _Mapping]] = ...) -> None: ...

class EnableLightingRequest(_message.Message):
    __slots__ = ("selection",)
    SELECTION_FIELD_NUMBER: _ClassVar[int]
    selection: LightingSelection
    def __init__(self, selection: _Optional[_Union[LightingSelection, _Mapping]] = ...) -> None: ...

class DisableLightingRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetSetupDevicesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class LightingDeviceCandidate(_message.Message):
    __slots__ = ("id", "label", "kind", "detail", "selected")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    SELECTED_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    kind: str
    detail: str
    selected: bool
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., kind: _Optional[str] = ..., detail: _Optional[str] = ..., selected: _Optional[bool] = ...) -> None: ...

class LightingDeviceDiscovery(_message.Message):
    __slots__ = ("candidates", "state", "message")
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    candidates: _containers.RepeatedCompositeFieldContainer[LightingDeviceCandidate]
    state: str
    message: str
    def __init__(self, candidates: _Optional[_Iterable[_Union[LightingDeviceCandidate, _Mapping]]] = ..., state: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class PlanLightingDevicesRequest(_message.Message):
    __slots__ = ("device_ids",)
    DEVICE_IDS_FIELD_NUMBER: _ClassVar[int]
    device_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, device_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class LightingDevicePlan(_message.Message):
    __slots__ = ("id", "devices", "changes", "warnings", "requires_privilege", "message")
    ID_FIELD_NUMBER: _ClassVar[int]
    DEVICES_FIELD_NUMBER: _ClassVar[int]
    CHANGES_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_PRIVILEGE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    id: str
    devices: _containers.RepeatedCompositeFieldContainer[LightingDeviceCandidate]
    changes: _containers.RepeatedScalarFieldContainer[str]
    warnings: _containers.RepeatedScalarFieldContainer[str]
    requires_privilege: bool
    message: str
    def __init__(self, id: _Optional[str] = ..., devices: _Optional[_Iterable[_Union[LightingDeviceCandidate, _Mapping]]] = ..., changes: _Optional[_Iterable[str]] = ..., warnings: _Optional[_Iterable[str]] = ..., requires_privilege: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...

class ApplyLightingDevicesRequest(_message.Message):
    __slots__ = ("plan_id",)
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    def __init__(self, plan_id: _Optional[str] = ...) -> None: ...

class RollbackLightingDevicesRequest(_message.Message):
    __slots__ = ("plan_id",)
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    def __init__(self, plan_id: _Optional[str] = ...) -> None: ...

class LightingDeviceAccessResult(_message.Message):
    __slots__ = ("state", "message", "plan_id", "needs_sudo", "device_policy_id")
    STATE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    NEEDS_SUDO_FIELD_NUMBER: _ClassVar[int]
    DEVICE_POLICY_ID_FIELD_NUMBER: _ClassVar[int]
    state: str
    message: str
    plan_id: str
    needs_sudo: bool
    device_policy_id: str
    def __init__(self, state: _Optional[str] = ..., message: _Optional[str] = ..., plan_id: _Optional[str] = ..., needs_sudo: _Optional[bool] = ..., device_policy_id: _Optional[str] = ...) -> None: ...

class LightingRuntimeStatus(_message.Message):
    __slots__ = ("phase", "state", "installed", "running", "ready", "degraded", "message", "selection", "pinned_selection", "setup_verified", "launch_blocker", "resource_observed", "device_access_plan_id", "device_access_pending")
    PHASE_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    INSTALLED_FIELD_NUMBER: _ClassVar[int]
    RUNNING_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SELECTION_FIELD_NUMBER: _ClassVar[int]
    PINNED_SELECTION_FIELD_NUMBER: _ClassVar[int]
    SETUP_VERIFIED_FIELD_NUMBER: _ClassVar[int]
    LAUNCH_BLOCKER_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_OBSERVED_FIELD_NUMBER: _ClassVar[int]
    DEVICE_ACCESS_PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    DEVICE_ACCESS_PENDING_FIELD_NUMBER: _ClassVar[int]
    phase: str
    state: str
    installed: bool
    running: bool
    ready: bool
    degraded: bool
    message: str
    selection: LightingSelection
    pinned_selection: LightingSelection
    setup_verified: bool
    launch_blocker: str
    resource_observed: bool
    device_access_plan_id: str
    device_access_pending: bool
    def __init__(self, phase: _Optional[str] = ..., state: _Optional[str] = ..., installed: _Optional[bool] = ..., running: _Optional[bool] = ..., ready: _Optional[bool] = ..., degraded: _Optional[bool] = ..., message: _Optional[str] = ..., selection: _Optional[_Union[LightingSelection, _Mapping]] = ..., pinned_selection: _Optional[_Union[LightingSelection, _Mapping]] = ..., setup_verified: _Optional[bool] = ..., launch_blocker: _Optional[str] = ..., resource_observed: _Optional[bool] = ..., device_access_plan_id: _Optional[str] = ..., device_access_pending: _Optional[bool] = ...) -> None: ...
