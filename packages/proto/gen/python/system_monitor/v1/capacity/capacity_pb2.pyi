from google.api import annotations_pb2 as _annotations_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CapacityClaim(_message.Message):
    __slots__ = ("claim_id", "owner_kind", "owner_id", "instance_id", "resource_kind", "gpu_index", "amount_bytes", "preferred_bytes", "floor_bytes", "priority", "priority_tier", "protected", "status", "activity_state", "generation", "last_active_at", "created_at", "updated_at")
    CLAIM_ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_KIND_FIELD_NUMBER: _ClassVar[int]
    OWNER_ID_FIELD_NUMBER: _ClassVar[int]
    INSTANCE_ID_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_KIND_FIELD_NUMBER: _ClassVar[int]
    GPU_INDEX_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_BYTES_FIELD_NUMBER: _ClassVar[int]
    PREFERRED_BYTES_FIELD_NUMBER: _ClassVar[int]
    FLOOR_BYTES_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_TIER_FIELD_NUMBER: _ClassVar[int]
    PROTECTED_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ACTIVITY_STATE_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    LAST_ACTIVE_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    claim_id: str
    owner_kind: str
    owner_id: str
    instance_id: str
    resource_kind: str
    gpu_index: int
    amount_bytes: int
    preferred_bytes: int
    floor_bytes: int
    priority: int
    priority_tier: str
    protected: bool
    status: str
    activity_state: str
    generation: int
    last_active_at: str
    created_at: str
    updated_at: str
    def __init__(self, claim_id: _Optional[str] = ..., owner_kind: _Optional[str] = ..., owner_id: _Optional[str] = ..., instance_id: _Optional[str] = ..., resource_kind: _Optional[str] = ..., gpu_index: _Optional[int] = ..., amount_bytes: _Optional[int] = ..., preferred_bytes: _Optional[int] = ..., floor_bytes: _Optional[int] = ..., priority: _Optional[int] = ..., priority_tier: _Optional[str] = ..., protected: _Optional[bool] = ..., status: _Optional[str] = ..., activity_state: _Optional[str] = ..., generation: _Optional[int] = ..., last_active_at: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class GpuCapacity(_message.Message):
    __slots__ = ("index", "name", "total_bytes", "used_bytes", "free_bytes", "claimed_bytes", "memory_utilization_percent")
    INDEX_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    TOTAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    USED_BYTES_FIELD_NUMBER: _ClassVar[int]
    FREE_BYTES_FIELD_NUMBER: _ClassVar[int]
    CLAIMED_BYTES_FIELD_NUMBER: _ClassVar[int]
    MEMORY_UTILIZATION_PERCENT_FIELD_NUMBER: _ClassVar[int]
    index: int
    name: str
    total_bytes: int
    used_bytes: int
    free_bytes: int
    claimed_bytes: int
    memory_utilization_percent: float
    def __init__(self, index: _Optional[int] = ..., name: _Optional[str] = ..., total_bytes: _Optional[int] = ..., used_bytes: _Optional[int] = ..., free_bytes: _Optional[int] = ..., claimed_bytes: _Optional[int] = ..., memory_utilization_percent: _Optional[float] = ...) -> None: ...

class CapacityFinding(_message.Message):
    __slots__ = ("owner_id", "owner_kind", "resource_kind", "gpu_index", "pid", "process_name", "observed_bytes", "claimed_bytes", "claim_id", "severity", "message")
    CLASS_FIELD_NUMBER: _ClassVar[int]
    OWNER_ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_KIND_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_KIND_FIELD_NUMBER: _ClassVar[int]
    GPU_INDEX_FIELD_NUMBER: _ClassVar[int]
    PID_FIELD_NUMBER: _ClassVar[int]
    PROCESS_NAME_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_BYTES_FIELD_NUMBER: _ClassVar[int]
    CLAIMED_BYTES_FIELD_NUMBER: _ClassVar[int]
    CLAIM_ID_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    owner_id: str
    owner_kind: str
    resource_kind: str
    gpu_index: int
    pid: int
    process_name: str
    observed_bytes: int
    claimed_bytes: int
    claim_id: str
    severity: str
    message: str
    def __init__(self, owner_id: _Optional[str] = ..., owner_kind: _Optional[str] = ..., resource_kind: _Optional[str] = ..., gpu_index: _Optional[int] = ..., pid: _Optional[int] = ..., process_name: _Optional[str] = ..., observed_bytes: _Optional[int] = ..., claimed_bytes: _Optional[int] = ..., claim_id: _Optional[str] = ..., severity: _Optional[str] = ..., message: _Optional[str] = ..., **kwargs) -> None: ...

class PolicyLever(_message.Message):
    __slots__ = ("key", "value")
    KEY_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    key: str
    value: str
    def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...

class GetCapacityOverviewRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetCapacityOverviewResponse(_message.Message):
    __slots__ = ("success", "gpus", "claims", "sensing_available", "warnings", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    GPUS_FIELD_NUMBER: _ClassVar[int]
    CLAIMS_FIELD_NUMBER: _ClassVar[int]
    SENSING_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    gpus: _containers.RepeatedCompositeFieldContainer[GpuCapacity]
    claims: _containers.RepeatedCompositeFieldContainer[CapacityClaim]
    sensing_available: bool
    warnings: _containers.RepeatedScalarFieldContainer[str]
    error: str
    def __init__(self, success: _Optional[bool] = ..., gpus: _Optional[_Iterable[_Union[GpuCapacity, _Mapping]]] = ..., claims: _Optional[_Iterable[_Union[CapacityClaim, _Mapping]]] = ..., sensing_available: _Optional[bool] = ..., warnings: _Optional[_Iterable[str]] = ..., error: _Optional[str] = ...) -> None: ...

class ListCapacityClaimsRequest(_message.Message):
    __slots__ = ("owner_id", "active_only")
    OWNER_ID_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_ONLY_FIELD_NUMBER: _ClassVar[int]
    owner_id: str
    active_only: bool
    def __init__(self, owner_id: _Optional[str] = ..., active_only: _Optional[bool] = ...) -> None: ...

class ListCapacityClaimsResponse(_message.Message):
    __slots__ = ("success", "claims", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    CLAIMS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    claims: _containers.RepeatedCompositeFieldContainer[CapacityClaim]
    error: str
    def __init__(self, success: _Optional[bool] = ..., claims: _Optional[_Iterable[_Union[CapacityClaim, _Mapping]]] = ..., error: _Optional[str] = ...) -> None: ...

class ReconcileCapacityRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ReconcileCapacityResponse(_message.Message):
    __slots__ = ("success", "findings", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    findings: _containers.RepeatedCompositeFieldContainer[CapacityFinding]
    error: str
    def __init__(self, success: _Optional[bool] = ..., findings: _Optional[_Iterable[_Union[CapacityFinding, _Mapping]]] = ..., error: _Optional[str] = ...) -> None: ...

class GetCapacityPolicyRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetCapacityPolicyResponse(_message.Message):
    __slots__ = ("success", "levers", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    LEVERS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    levers: _containers.RepeatedCompositeFieldContainer[PolicyLever]
    error: str
    def __init__(self, success: _Optional[bool] = ..., levers: _Optional[_Iterable[_Union[PolicyLever, _Mapping]]] = ..., error: _Optional[str] = ...) -> None: ...

class SetCapacityPolicyRequest(_message.Message):
    __slots__ = ("key", "value")
    KEY_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    key: str
    value: str
    def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...

class SetCapacityPolicyResponse(_message.Message):
    __slots__ = ("success", "levers", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    LEVERS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    levers: _containers.RepeatedCompositeFieldContainer[PolicyLever]
    error: str
    def __init__(self, success: _Optional[bool] = ..., levers: _Optional[_Iterable[_Union[PolicyLever, _Mapping]]] = ..., error: _Optional[str] = ...) -> None: ...
