import datetime

from google.protobuf import duration_pb2 as _duration_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class LifecycleMaintenancePhase(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    LIFECYCLE_MAINTENANCE_PHASE_UNSPECIFIED: _ClassVar[LifecycleMaintenancePhase]
    LIFECYCLE_MAINTENANCE_PHASE_OPEN: _ClassVar[LifecycleMaintenancePhase]
    LIFECYCLE_MAINTENANCE_PHASE_CLOSING: _ClassVar[LifecycleMaintenancePhase]
    LIFECYCLE_MAINTENANCE_PHASE_DRAINED: _ClassVar[LifecycleMaintenancePhase]
    LIFECYCLE_MAINTENANCE_PHASE_RESUMED: _ClassVar[LifecycleMaintenancePhase]

class LifecycleBlockerDisposition(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    LIFECYCLE_BLOCKER_DISPOSITION_UNSPECIFIED: _ClassVar[LifecycleBlockerDisposition]
    LIFECYCLE_BLOCKER_DISPOSITION_BLOCKING: _ClassVar[LifecycleBlockerDisposition]
    LIFECYCLE_BLOCKER_DISPOSITION_OVERRIDEABLE: _ClassVar[LifecycleBlockerDisposition]
    LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN: _ClassVar[LifecycleBlockerDisposition]
LIFECYCLE_MAINTENANCE_PHASE_UNSPECIFIED: LifecycleMaintenancePhase
LIFECYCLE_MAINTENANCE_PHASE_OPEN: LifecycleMaintenancePhase
LIFECYCLE_MAINTENANCE_PHASE_CLOSING: LifecycleMaintenancePhase
LIFECYCLE_MAINTENANCE_PHASE_DRAINED: LifecycleMaintenancePhase
LIFECYCLE_MAINTENANCE_PHASE_RESUMED: LifecycleMaintenancePhase
LIFECYCLE_BLOCKER_DISPOSITION_UNSPECIFIED: LifecycleBlockerDisposition
LIFECYCLE_BLOCKER_DISPOSITION_BLOCKING: LifecycleBlockerDisposition
LIFECYCLE_BLOCKER_DISPOSITION_OVERRIDEABLE: LifecycleBlockerDisposition
LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN: LifecycleBlockerDisposition

class LifecyclePrepareRequest(_message.Message):
    __slots__ = ("operation_id", "reason", "emergency_override", "override_reason")
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EMERGENCY_OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    OVERRIDE_REASON_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    reason: str
    emergency_override: bool
    override_reason: str
    def __init__(self, operation_id: _Optional[str] = ..., reason: _Optional[str] = ..., emergency_override: _Optional[bool] = ..., override_reason: _Optional[str] = ...) -> None: ...

class LifecycleStatusRequest(_message.Message):
    __slots__ = ("operation_id", "fence_token", "revision")
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    FENCE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    fence_token: str
    revision: int
    def __init__(self, operation_id: _Optional[str] = ..., fence_token: _Optional[str] = ..., revision: _Optional[int] = ...) -> None: ...

class LifecycleDrainRequest(_message.Message):
    __slots__ = ("operation_id", "fence_token", "timeout", "revision")
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    FENCE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    fence_token: str
    timeout: _duration_pb2.Duration
    revision: int
    def __init__(self, operation_id: _Optional[str] = ..., fence_token: _Optional[str] = ..., timeout: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., revision: _Optional[int] = ...) -> None: ...

class LifecycleResumeRequest(_message.Message):
    __slots__ = ("operation_id", "fence_token", "revision")
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    FENCE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    fence_token: str
    revision: int
    def __init__(self, operation_id: _Optional[str] = ..., fence_token: _Optional[str] = ..., revision: _Optional[int] = ...) -> None: ...

class LifecycleWorkItem(_message.Message):
    __slots__ = ("id", "kind", "status", "physical_executor_present")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PHYSICAL_EXECUTOR_PRESENT_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    status: str
    physical_executor_present: bool
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., status: _Optional[str] = ..., physical_executor_present: _Optional[bool] = ...) -> None: ...

class LifecycleBlocker(_message.Message):
    __slots__ = ("code", "message", "disposition", "observed")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    disposition: LifecycleBlockerDisposition
    observed: bool
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., disposition: _Optional[_Union[LifecycleBlockerDisposition, str]] = ..., observed: _Optional[bool] = ...) -> None: ...

class LifecycleStanding(_message.Message):
    __slots__ = ("provider", "provider_instance_id", "scenario", "instance_id", "operation_id", "fence_token", "revision", "phase", "admission_closed", "admitting", "remaining", "inventory_complete", "drained", "work", "blockers", "observed_at", "expires_at", "interlock")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_INSTANCE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    INSTANCE_ID_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    FENCE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    ADMISSION_CLOSED_FIELD_NUMBER: _ClassVar[int]
    ADMITTING_FIELD_NUMBER: _ClassVar[int]
    REMAINING_FIELD_NUMBER: _ClassVar[int]
    INVENTORY_COMPLETE_FIELD_NUMBER: _ClassVar[int]
    DRAINED_FIELD_NUMBER: _ClassVar[int]
    WORK_FIELD_NUMBER: _ClassVar[int]
    BLOCKERS_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    INTERLOCK_FIELD_NUMBER: _ClassVar[int]
    provider: str
    provider_instance_id: str
    scenario: str
    instance_id: str
    operation_id: str
    fence_token: str
    revision: int
    phase: LifecycleMaintenancePhase
    admission_closed: bool
    admitting: int
    remaining: int
    inventory_complete: bool
    drained: bool
    work: _containers.RepeatedCompositeFieldContainer[LifecycleWorkItem]
    blockers: _containers.RepeatedCompositeFieldContainer[LifecycleBlocker]
    observed_at: _timestamp_pb2.Timestamp
    expires_at: _timestamp_pb2.Timestamp
    interlock: str
    def __init__(self, provider: _Optional[str] = ..., provider_instance_id: _Optional[str] = ..., scenario: _Optional[str] = ..., instance_id: _Optional[str] = ..., operation_id: _Optional[str] = ..., fence_token: _Optional[str] = ..., revision: _Optional[int] = ..., phase: _Optional[_Union[LifecycleMaintenancePhase, str]] = ..., admission_closed: _Optional[bool] = ..., admitting: _Optional[int] = ..., remaining: _Optional[int] = ..., inventory_complete: _Optional[bool] = ..., drained: _Optional[bool] = ..., work: _Optional[_Iterable[_Union[LifecycleWorkItem, _Mapping]]] = ..., blockers: _Optional[_Iterable[_Union[LifecycleBlocker, _Mapping]]] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., interlock: _Optional[str] = ...) -> None: ...

class LifecyclePrepareResponse(_message.Message):
    __slots__ = ("standing",)
    STANDING_FIELD_NUMBER: _ClassVar[int]
    standing: LifecycleStanding
    def __init__(self, standing: _Optional[_Union[LifecycleStanding, _Mapping]] = ...) -> None: ...

class LifecycleStatusResponse(_message.Message):
    __slots__ = ("standing",)
    STANDING_FIELD_NUMBER: _ClassVar[int]
    standing: LifecycleStanding
    def __init__(self, standing: _Optional[_Union[LifecycleStanding, _Mapping]] = ...) -> None: ...

class LifecycleDrainResponse(_message.Message):
    __slots__ = ("standing",)
    STANDING_FIELD_NUMBER: _ClassVar[int]
    standing: LifecycleStanding
    def __init__(self, standing: _Optional[_Union[LifecycleStanding, _Mapping]] = ...) -> None: ...

class LifecycleResumeResponse(_message.Message):
    __slots__ = ("standing",)
    STANDING_FIELD_NUMBER: _ClassVar[int]
    standing: LifecycleStanding
    def __init__(self, standing: _Optional[_Union[LifecycleStanding, _Mapping]] = ...) -> None: ...
