import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import empty_pb2 as _empty_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class BackendState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    BACKEND_STATE_UNSPECIFIED: _ClassVar[BackendState]
    BACKEND_STATE_DISABLED: _ClassVar[BackendState]
    BACKEND_STATE_UNAVAILABLE: _ClassVar[BackendState]
    BACKEND_STATE_READY: _ClassVar[BackendState]
    BACKEND_STATE_PROTOCOL_MISMATCH: _ClassVar[BackendState]
    BACKEND_STATE_STARTING: _ClassVar[BackendState]
    BACKEND_STATE_STOPPED: _ClassVar[BackendState]

class BackendReason(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    BACKEND_REASON_UNSPECIFIED: _ClassVar[BackendReason]
    BACKEND_REASON_READY: _ClassVar[BackendReason]
    BACKEND_REASON_DISABLED: _ClassVar[BackendReason]
    BACKEND_REASON_NOT_INSTALLED: _ClassVar[BackendReason]
    BACKEND_REASON_STOPPED: _ClassVar[BackendReason]
    BACKEND_REASON_PERMISSION_DENIED: _ClassVar[BackendReason]
    BACKEND_REASON_PROTOCOL_INCOMPATIBLE: _ClassVar[BackendReason]
    BACKEND_REASON_UNSUPPORTED_PLATFORM: _ClassVar[BackendReason]
    BACKEND_REASON_PROBE_FAILED: _ClassVar[BackendReason]

class OperationState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OPERATION_STATE_UNSPECIFIED: _ClassVar[OperationState]
    OPERATION_STATE_COMMAND_SENT: _ClassVar[OperationState]
    OPERATION_STATE_FAILED: _ClassVar[OperationState]
    OPERATION_STATE_UNSUPPORTED: _ClassVar[OperationState]
    OPERATION_STATE_DEFERRED: _ClassVar[OperationState]
    OPERATION_STATE_BACKEND_ACKNOWLEDGED: _ClassVar[OperationState]
BACKEND_STATE_UNSPECIFIED: BackendState
BACKEND_STATE_DISABLED: BackendState
BACKEND_STATE_UNAVAILABLE: BackendState
BACKEND_STATE_READY: BackendState
BACKEND_STATE_PROTOCOL_MISMATCH: BackendState
BACKEND_STATE_STARTING: BackendState
BACKEND_STATE_STOPPED: BackendState
BACKEND_REASON_UNSPECIFIED: BackendReason
BACKEND_REASON_READY: BackendReason
BACKEND_REASON_DISABLED: BackendReason
BACKEND_REASON_NOT_INSTALLED: BackendReason
BACKEND_REASON_STOPPED: BackendReason
BACKEND_REASON_PERMISSION_DENIED: BackendReason
BACKEND_REASON_PROTOCOL_INCOMPATIBLE: BackendReason
BACKEND_REASON_UNSUPPORTED_PLATFORM: BackendReason
BACKEND_REASON_PROBE_FAILED: BackendReason
OPERATION_STATE_UNSPECIFIED: OperationState
OPERATION_STATE_COMMAND_SENT: OperationState
OPERATION_STATE_FAILED: OperationState
OPERATION_STATE_UNSUPPORTED: OperationState
OPERATION_STATE_DEFERRED: OperationState
OPERATION_STATE_BACKEND_ACKNOWLEDGED: OperationState

class BackendStatus(_message.Message):
    __slots__ = ("state", "version", "sdk_protocol_version", "detail", "checked_at", "reason")
    STATE_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    SDK_PROTOCOL_VERSION_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    state: BackendState
    version: str
    sdk_protocol_version: int
    detail: str
    checked_at: str
    reason: BackendReason
    def __init__(self, state: _Optional[_Union[BackendState, str]] = ..., version: _Optional[str] = ..., sdk_protocol_version: _Optional[int] = ..., detail: _Optional[str] = ..., checked_at: _Optional[str] = ..., reason: _Optional[_Union[BackendReason, str]] = ...) -> None: ...

class ZoneCapability(_message.Message):
    __slots__ = ("zone_id", "name", "led_count", "supports_static_color", "supports_brightness", "supported_effects")
    ZONE_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    LED_COUNT_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_STATIC_COLOR_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_BRIGHTNESS_FIELD_NUMBER: _ClassVar[int]
    SUPPORTED_EFFECTS_FIELD_NUMBER: _ClassVar[int]
    zone_id: str
    name: str
    led_count: int
    supports_static_color: bool
    supports_brightness: bool
    supported_effects: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, zone_id: _Optional[str] = ..., name: _Optional[str] = ..., led_count: _Optional[int] = ..., supports_static_color: _Optional[bool] = ..., supports_brightness: _Optional[bool] = ..., supported_effects: _Optional[_Iterable[str]] = ...) -> None: ...

class Controller(_message.Message):
    __slots__ = ("backend_id", "controller_id", "name", "vendor", "description", "enrolled", "zones", "last_seen_at", "identity_source")
    BACKEND_ID_FIELD_NUMBER: _ClassVar[int]
    CONTROLLER_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    VENDOR_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ENROLLED_FIELD_NUMBER: _ClassVar[int]
    ZONES_FIELD_NUMBER: _ClassVar[int]
    LAST_SEEN_AT_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_SOURCE_FIELD_NUMBER: _ClassVar[int]
    backend_id: str
    controller_id: str
    name: str
    vendor: str
    description: str
    enrolled: bool
    zones: _containers.RepeatedCompositeFieldContainer[ZoneCapability]
    last_seen_at: _timestamp_pb2.Timestamp
    identity_source: str
    def __init__(self, backend_id: _Optional[str] = ..., controller_id: _Optional[str] = ..., name: _Optional[str] = ..., vendor: _Optional[str] = ..., description: _Optional[str] = ..., enrolled: _Optional[bool] = ..., zones: _Optional[_Iterable[_Union[ZoneCapability, _Mapping]]] = ..., last_seen_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., identity_source: _Optional[str] = ...) -> None: ...

class Color(_message.Message):
    __slots__ = ("red", "green", "blue")
    RED_FIELD_NUMBER: _ClassVar[int]
    GREEN_FIELD_NUMBER: _ClassVar[int]
    BLUE_FIELD_NUMBER: _ClassVar[int]
    red: int
    green: int
    blue: int
    def __init__(self, red: _Optional[int] = ..., green: _Optional[int] = ..., blue: _Optional[int] = ...) -> None: ...

class ZoneState(_message.Message):
    __slots__ = ("controller_id", "zone_id", "off", "static_color", "effect")
    CONTROLLER_ID_FIELD_NUMBER: _ClassVar[int]
    ZONE_ID_FIELD_NUMBER: _ClassVar[int]
    OFF_FIELD_NUMBER: _ClassVar[int]
    STATIC_COLOR_FIELD_NUMBER: _ClassVar[int]
    EFFECT_FIELD_NUMBER: _ClassVar[int]
    controller_id: str
    zone_id: str
    off: _empty_pb2.Empty
    static_color: StaticZoneState
    effect: EffectZoneState
    def __init__(self, controller_id: _Optional[str] = ..., zone_id: _Optional[str] = ..., off: _Optional[_Union[_empty_pb2.Empty, _Mapping]] = ..., static_color: _Optional[_Union[StaticZoneState, _Mapping]] = ..., effect: _Optional[_Union[EffectZoneState, _Mapping]] = ...) -> None: ...

class StaticZoneState(_message.Message):
    __slots__ = ("color", "brightness_percent")
    COLOR_FIELD_NUMBER: _ClassVar[int]
    BRIGHTNESS_PERCENT_FIELD_NUMBER: _ClassVar[int]
    color: Color
    brightness_percent: int
    def __init__(self, color: _Optional[_Union[Color, _Mapping]] = ..., brightness_percent: _Optional[int] = ...) -> None: ...

class EffectZoneState(_message.Message):
    __slots__ = ("effect_id", "color", "brightness_percent")
    EFFECT_ID_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    BRIGHTNESS_PERCENT_FIELD_NUMBER: _ClassVar[int]
    effect_id: str
    color: Color
    brightness_percent: int
    def __init__(self, effect_id: _Optional[str] = ..., color: _Optional[_Union[Color, _Mapping]] = ..., brightness_percent: _Optional[int] = ...) -> None: ...

class ZoneApplyResult(_message.Message):
    __slots__ = ("desired", "operation", "reason", "completed_at", "backend_acknowledged", "backend_ack_status", "local_write_completed")
    DESIRED_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    BACKEND_ACKNOWLEDGED_FIELD_NUMBER: _ClassVar[int]
    BACKEND_ACK_STATUS_FIELD_NUMBER: _ClassVar[int]
    LOCAL_WRITE_COMPLETED_FIELD_NUMBER: _ClassVar[int]
    desired: ZoneState
    operation: OperationState
    reason: str
    completed_at: _timestamp_pb2.Timestamp
    backend_acknowledged: bool
    backend_ack_status: int
    local_write_completed: bool
    def __init__(self, desired: _Optional[_Union[ZoneState, _Mapping]] = ..., operation: _Optional[_Union[OperationState, str]] = ..., reason: _Optional[str] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., backend_acknowledged: _Optional[bool] = ..., backend_ack_status: _Optional[int] = ..., local_write_completed: _Optional[bool] = ...) -> None: ...

class OutputObservation(_message.Message):
    __slots__ = ("state", "source", "observed_at", "evidence")
    STATE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    state: ZoneState
    source: str
    observed_at: _timestamp_pb2.Timestamp
    evidence: str
    def __init__(self, state: _Optional[_Union[ZoneState, _Mapping]] = ..., source: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., evidence: _Optional[str] = ...) -> None: ...

class ZoneStatus(_message.Message):
    __slots__ = ("desired", "last_apply", "observation")
    DESIRED_FIELD_NUMBER: _ClassVar[int]
    LAST_APPLY_FIELD_NUMBER: _ClassVar[int]
    OBSERVATION_FIELD_NUMBER: _ClassVar[int]
    desired: ZoneState
    last_apply: ZoneApplyResult
    observation: OutputObservation
    def __init__(self, desired: _Optional[_Union[ZoneState, _Mapping]] = ..., last_apply: _Optional[_Union[ZoneApplyResult, _Mapping]] = ..., observation: _Optional[_Union[OutputObservation, _Mapping]] = ...) -> None: ...

class ManualOverride(_message.Message):
    __slots__ = ("active", "all_enrolled", "controller_ids", "created_at", "expires_at")
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    ALL_ENROLLED_FIELD_NUMBER: _ClassVar[int]
    CONTROLLER_IDS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    active: bool
    all_enrolled: bool
    controller_ids: _containers.RepeatedScalarFieldContainer[str]
    created_at: _timestamp_pb2.Timestamp
    expires_at: _timestamp_pb2.Timestamp
    def __init__(self, active: _Optional[bool] = ..., all_enrolled: _Optional[bool] = ..., controller_ids: _Optional[_Iterable[str]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GetBackendStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetBackendStatusResponse(_message.Message):
    __slots__ = ("backend",)
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    backend: BackendStatus
    def __init__(self, backend: _Optional[_Union[BackendStatus, _Mapping]] = ...) -> None: ...

class ListControllersRequest(_message.Message):
    __slots__ = ("include_unavailable",)
    INCLUDE_UNAVAILABLE_FIELD_NUMBER: _ClassVar[int]
    include_unavailable: bool
    def __init__(self, include_unavailable: _Optional[bool] = ...) -> None: ...

class ListControllersResponse(_message.Message):
    __slots__ = ("backend", "controllers")
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    CONTROLLERS_FIELD_NUMBER: _ClassVar[int]
    backend: BackendStatus
    controllers: _containers.RepeatedCompositeFieldContainer[Controller]
    def __init__(self, backend: _Optional[_Union[BackendStatus, _Mapping]] = ..., controllers: _Optional[_Iterable[_Union[Controller, _Mapping]]] = ...) -> None: ...

class EnrollControllerRequest(_message.Message):
    __slots__ = ("controller_id", "enrolled")
    CONTROLLER_ID_FIELD_NUMBER: _ClassVar[int]
    ENROLLED_FIELD_NUMBER: _ClassVar[int]
    controller_id: str
    enrolled: bool
    def __init__(self, controller_id: _Optional[str] = ..., enrolled: _Optional[bool] = ...) -> None: ...

class EnrollControllerResponse(_message.Message):
    __slots__ = ("controller",)
    CONTROLLER_FIELD_NUMBER: _ClassVar[int]
    controller: Controller
    def __init__(self, controller: _Optional[_Union[Controller, _Mapping]] = ...) -> None: ...

class SetZoneStateRequest(_message.Message):
    __slots__ = ("desired",)
    DESIRED_FIELD_NUMBER: _ClassVar[int]
    desired: ZoneState
    def __init__(self, desired: _Optional[_Union[ZoneState, _Mapping]] = ...) -> None: ...

class SetZoneStateResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: ZoneApplyResult
    def __init__(self, result: _Optional[_Union[ZoneApplyResult, _Mapping]] = ...) -> None: ...

class SetManualOffRequest(_message.Message):
    __slots__ = ("all_enrolled", "controller_ids", "expires_at")
    ALL_ENROLLED_FIELD_NUMBER: _ClassVar[int]
    CONTROLLER_IDS_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    all_enrolled: bool
    controller_ids: _containers.RepeatedScalarFieldContainer[str]
    expires_at: _timestamp_pb2.Timestamp
    def __init__(self, all_enrolled: _Optional[bool] = ..., controller_ids: _Optional[_Iterable[str]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class SetManualOffResponse(_message.Message):
    __slots__ = ("override", "results")
    OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    override: ManualOverride
    results: _containers.RepeatedCompositeFieldContainer[ZoneApplyResult]
    def __init__(self, override: _Optional[_Union[ManualOverride, _Mapping]] = ..., results: _Optional[_Iterable[_Union[ZoneApplyResult, _Mapping]]] = ...) -> None: ...

class GetDesiredStateRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetDesiredStateResponse(_message.Message):
    __slots__ = ("zones", "manual_override")
    ZONES_FIELD_NUMBER: _ClassVar[int]
    MANUAL_OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    zones: _containers.RepeatedCompositeFieldContainer[ZoneStatus]
    manual_override: ManualOverride
    def __init__(self, zones: _Optional[_Iterable[_Union[ZoneStatus, _Mapping]]] = ..., manual_override: _Optional[_Union[ManualOverride, _Mapping]] = ...) -> None: ...

class ClearManualOverrideRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ClearManualOverrideResponse(_message.Message):
    __slots__ = ("cleared",)
    CLEARED_FIELD_NUMBER: _ClassVar[int]
    cleared: bool
    def __init__(self, cleared: _Optional[bool] = ...) -> None: ...

class ReconcileZoneResult(_message.Message):
    __slots__ = ("controller_id", "zone_id", "result")
    CONTROLLER_ID_FIELD_NUMBER: _ClassVar[int]
    ZONE_ID_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    controller_id: str
    zone_id: str
    result: ZoneApplyResult
    def __init__(self, controller_id: _Optional[str] = ..., zone_id: _Optional[str] = ..., result: _Optional[_Union[ZoneApplyResult, _Mapping]] = ...) -> None: ...

class ReconcileRequest(_message.Message):
    __slots__ = ("trigger",)
    TRIGGER_FIELD_NUMBER: _ClassVar[int]
    trigger: str
    def __init__(self, trigger: _Optional[str] = ...) -> None: ...

class ReconcileResponse(_message.Message):
    __slots__ = ("zones", "completed_at")
    ZONES_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    zones: _containers.RepeatedCompositeFieldContainer[ReconcileZoneResult]
    completed_at: _timestamp_pb2.Timestamp
    def __init__(self, zones: _Optional[_Iterable[_Union[ReconcileZoneResult, _Mapping]]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
