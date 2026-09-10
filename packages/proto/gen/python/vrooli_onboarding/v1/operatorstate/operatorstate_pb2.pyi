from google.protobuf import field_mask_pb2 as _field_mask_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetOperatorStateRequest(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: str
    def __init__(self, target: _Optional[str] = ...) -> None: ...

class PatchOperatorStateRequest(_message.Message):
    __slots__ = ("target", "state", "update_mask", "expected_revision")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    UPDATE_MASK_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    target: str
    state: OperatorState
    update_mask: _field_mask_pb2.FieldMask
    expected_revision: str
    def __init__(self, target: _Optional[str] = ..., state: _Optional[_Union[OperatorState, _Mapping]] = ..., update_mask: _Optional[_Union[_field_mask_pb2.FieldMask, _Mapping]] = ..., expected_revision: _Optional[str] = ...) -> None: ...

class GetOperatorStateResponse(_message.Message):
    __slots__ = ("state",)
    STATE_FIELD_NUMBER: _ClassVar[int]
    state: OperatorState
    def __init__(self, state: _Optional[_Union[OperatorState, _Mapping]] = ...) -> None: ...

class PatchOperatorStateResponse(_message.Message):
    __slots__ = ("state",)
    STATE_FIELD_NUMBER: _ClassVar[int]
    state: OperatorState
    def __init__(self, state: _Optional[_Union[OperatorState, _Mapping]] = ...) -> None: ...

class OperatorState(_message.Message):
    __slots__ = ("schema", "version", "updated_at", "trust_posture", "host_workload_posture", "capacity_posture", "accel_preference", "update_control", "operator_id", "identity_provider", "session_mode", "enrollment_reference", "enrolled_at", "core", "active_profile", "capacity", "scenarios", "resources", "host_tools", "host_safeguards", "completion", "session", "notifications")
    class ScenariosEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: ScenarioChoice
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[ScenarioChoice, _Mapping]] = ...) -> None: ...
    class ResourcesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: EnabledChoice
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[EnabledChoice, _Mapping]] = ...) -> None: ...
    class HostToolsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: OptInChoice
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[OptInChoice, _Mapping]] = ...) -> None: ...
    class HostSafeguardsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: OptInChoice
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[OptInChoice, _Mapping]] = ...) -> None: ...
    SCHEMA_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    TRUST_POSTURE_FIELD_NUMBER: _ClassVar[int]
    HOST_WORKLOAD_POSTURE_FIELD_NUMBER: _ClassVar[int]
    CAPACITY_POSTURE_FIELD_NUMBER: _ClassVar[int]
    ACCEL_PREFERENCE_FIELD_NUMBER: _ClassVar[int]
    UPDATE_CONTROL_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_ID_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_PROVIDER_FIELD_NUMBER: _ClassVar[int]
    SESSION_MODE_FIELD_NUMBER: _ClassVar[int]
    ENROLLMENT_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    ENROLLED_AT_FIELD_NUMBER: _ClassVar[int]
    CORE_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_PROFILE_FIELD_NUMBER: _ClassVar[int]
    CAPACITY_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    RESOURCES_FIELD_NUMBER: _ClassVar[int]
    HOST_TOOLS_FIELD_NUMBER: _ClassVar[int]
    HOST_SAFEGUARDS_FIELD_NUMBER: _ClassVar[int]
    COMPLETION_FIELD_NUMBER: _ClassVar[int]
    SESSION_FIELD_NUMBER: _ClassVar[int]
    NOTIFICATIONS_FIELD_NUMBER: _ClassVar[int]
    schema: str
    version: str
    updated_at: str
    trust_posture: str
    host_workload_posture: str
    capacity_posture: str
    accel_preference: str
    update_control: str
    operator_id: str
    identity_provider: str
    session_mode: str
    enrollment_reference: str
    enrolled_at: str
    core: CoreSet
    active_profile: _struct_pb2.Value
    capacity: CapacitySettings
    scenarios: _containers.MessageMap[str, ScenarioChoice]
    resources: _containers.MessageMap[str, EnabledChoice]
    host_tools: _containers.MessageMap[str, OptInChoice]
    host_safeguards: _containers.MessageMap[str, OptInChoice]
    completion: Completion
    session: Session
    notifications: Notifications
    def __init__(self, schema: _Optional[str] = ..., version: _Optional[str] = ..., updated_at: _Optional[str] = ..., trust_posture: _Optional[str] = ..., host_workload_posture: _Optional[str] = ..., capacity_posture: _Optional[str] = ..., accel_preference: _Optional[str] = ..., update_control: _Optional[str] = ..., operator_id: _Optional[str] = ..., identity_provider: _Optional[str] = ..., session_mode: _Optional[str] = ..., enrollment_reference: _Optional[str] = ..., enrolled_at: _Optional[str] = ..., core: _Optional[_Union[CoreSet, _Mapping]] = ..., active_profile: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., capacity: _Optional[_Union[CapacitySettings, _Mapping]] = ..., scenarios: _Optional[_Mapping[str, ScenarioChoice]] = ..., resources: _Optional[_Mapping[str, EnabledChoice]] = ..., host_tools: _Optional[_Mapping[str, OptInChoice]] = ..., host_safeguards: _Optional[_Mapping[str, OptInChoice]] = ..., completion: _Optional[_Union[Completion, _Mapping]] = ..., session: _Optional[_Union[Session, _Mapping]] = ..., notifications: _Optional[_Union[Notifications, _Mapping]] = ...) -> None: ...

class CoreSet(_message.Message):
    __slots__ = ("seed", "trusted_base")
    SEED_FIELD_NUMBER: _ClassVar[int]
    TRUSTED_BASE_FIELD_NUMBER: _ClassVar[int]
    seed: _containers.RepeatedScalarFieldContainer[str]
    trusted_base: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, seed: _Optional[_Iterable[str]] = ..., trusted_base: _Optional[_Iterable[str]] = ...) -> None: ...

class ScenarioChoice(_message.Message):
    __slots__ = ("enabled", "auto_restart")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    AUTO_RESTART_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    auto_restart: bool
    def __init__(self, enabled: _Optional[bool] = ..., auto_restart: _Optional[bool] = ...) -> None: ...

class EnabledChoice(_message.Message):
    __slots__ = ("enabled", "capacity")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    CAPACITY_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    capacity: CapacityChoice
    def __init__(self, enabled: _Optional[bool] = ..., capacity: _Optional[_Union[CapacityChoice, _Mapping]] = ...) -> None: ...

class CapacityChoice(_message.Message):
    __slots__ = ("rung", "tunables", "gpu_index", "priority", "yield_when_idle", "idle_grace_seconds")
    class TunablesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: _struct_pb2.Value
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...
    RUNG_FIELD_NUMBER: _ClassVar[int]
    TUNABLES_FIELD_NUMBER: _ClassVar[int]
    GPU_INDEX_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    YIELD_WHEN_IDLE_FIELD_NUMBER: _ClassVar[int]
    IDLE_GRACE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    rung: str
    tunables: _containers.MessageMap[str, _struct_pb2.Value]
    gpu_index: int
    priority: str
    yield_when_idle: bool
    idle_grace_seconds: int
    def __init__(self, rung: _Optional[str] = ..., tunables: _Optional[_Mapping[str, _struct_pb2.Value]] = ..., gpu_index: _Optional[int] = ..., priority: _Optional[str] = ..., yield_when_idle: _Optional[bool] = ..., idle_grace_seconds: _Optional[int] = ...) -> None: ...

class CapacitySettings(_message.Message):
    __slots__ = ("transient_headroom_reserve_bytes",)
    TRANSIENT_HEADROOM_RESERVE_BYTES_FIELD_NUMBER: _ClassVar[int]
    transient_headroom_reserve_bytes: int
    def __init__(self, transient_headroom_reserve_bytes: _Optional[int] = ...) -> None: ...

class OptInChoice(_message.Message):
    __slots__ = ("opted_in", "config")
    OPTED_IN_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    opted_in: bool
    config: _struct_pb2.Struct
    def __init__(self, opted_in: _Optional[bool] = ..., config: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class Completion(_message.Message):
    __slots__ = ("selection_digest", "applied_at", "degraded_acknowledgement")
    SELECTION_DIGEST_FIELD_NUMBER: _ClassVar[int]
    APPLIED_AT_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_ACKNOWLEDGEMENT_FIELD_NUMBER: _ClassVar[int]
    selection_digest: str
    applied_at: str
    degraded_acknowledgement: DegradedAcknowledgement
    def __init__(self, selection_digest: _Optional[str] = ..., applied_at: _Optional[str] = ..., degraded_acknowledgement: _Optional[_Union[DegradedAcknowledgement, _Mapping]] = ...) -> None: ...

class DegradedAcknowledgement(_message.Message):
    __slots__ = ("readiness_digest", "acknowledged_at")
    READINESS_DIGEST_FIELD_NUMBER: _ClassVar[int]
    ACKNOWLEDGED_AT_FIELD_NUMBER: _ClassVar[int]
    readiness_digest: str
    acknowledged_at: str
    def __init__(self, readiness_digest: _Optional[str] = ..., acknowledged_at: _Optional[str] = ...) -> None: ...

class Session(_message.Message):
    __slots__ = ("step", "step_id")
    STEP_FIELD_NUMBER: _ClassVar[int]
    STEP_ID_FIELD_NUMBER: _ClassVar[int]
    step: int
    step_id: str
    def __init__(self, step: _Optional[int] = ..., step_id: _Optional[str] = ...) -> None: ...

class Notifications(_message.Message):
    __slots__ = ("recipient",)
    RECIPIENT_FIELD_NUMBER: _ClassVar[int]
    recipient: str
    def __init__(self, recipient: _Optional[str] = ...) -> None: ...
