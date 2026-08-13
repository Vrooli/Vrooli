from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ValidateFlowRequest(_message.Message):
    __slots__ = ("flow", "strategy_id")
    FLOW_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_ID_FIELD_NUMBER: _ClassVar[int]
    flow: Flow
    strategy_id: str
    def __init__(self, flow: _Optional[_Union[Flow, _Mapping]] = ..., strategy_id: _Optional[str] = ...) -> None: ...

class RunFlowRequest(_message.Message):
    __slots__ = ("flow", "device_id", "actor", "lease_token")
    FLOW_FIELD_NUMBER: _ClassVar[int]
    DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    LEASE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    flow: Flow
    device_id: str
    actor: str
    lease_token: str
    def __init__(self, flow: _Optional[_Union[Flow, _Mapping]] = ..., device_id: _Optional[str] = ..., actor: _Optional[str] = ..., lease_token: _Optional[str] = ...) -> None: ...

class Flow(_message.Message):
    __slots__ = ("id", "name", "steps", "allow_unredacted_capture", "transport", "require_unlocked", "auth_profile_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    ALLOW_UNREDACTED_CAPTURE_FIELD_NUMBER: _ClassVar[int]
    TRANSPORT_FIELD_NUMBER: _ClassVar[int]
    REQUIRE_UNLOCKED_FIELD_NUMBER: _ClassVar[int]
    AUTH_PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    steps: _containers.RepeatedCompositeFieldContainer[Step]
    allow_unredacted_capture: bool
    transport: str
    require_unlocked: bool
    auth_profile_id: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., steps: _Optional[_Iterable[_Union[Step, _Mapping]]] = ..., allow_unredacted_capture: _Optional[bool] = ..., transport: _Optional[str] = ..., require_unlocked: _Optional[bool] = ..., auth_profile_id: _Optional[str] = ...) -> None: ...

class Step(_message.Message):
    __slots__ = ("id", "kind", "required_capabilities", "target", "timeout_ms", "arguments")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MS_FIELD_NUMBER: _ClassVar[int]
    ARGUMENTS_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    required_capabilities: _containers.RepeatedScalarFieldContainer[str]
    target: str
    timeout_ms: int
    arguments: _struct_pb2.Struct
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., required_capabilities: _Optional[_Iterable[str]] = ..., target: _Optional[str] = ..., timeout_ms: _Optional[int] = ..., arguments: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class CapabilityGapReport(_message.Message):
    __slots__ = ("runnable", "gaps", "warnings")
    RUNNABLE_FIELD_NUMBER: _ClassVar[int]
    GAPS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    runnable: bool
    gaps: _containers.RepeatedScalarFieldContainer[str]
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, runnable: _Optional[bool] = ..., gaps: _Optional[_Iterable[str]] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...

class RunResult(_message.Message):
    __slots__ = ("run_id", "disposition", "chapters", "resolutions", "evidence", "incomplete", "disconnect_reason", "disconnect_step")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    CHAPTERS_FIELD_NUMBER: _ClassVar[int]
    RESOLUTIONS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    INCOMPLETE_FIELD_NUMBER: _ClassVar[int]
    DISCONNECT_REASON_FIELD_NUMBER: _ClassVar[int]
    DISCONNECT_STEP_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    disposition: str
    chapters: _containers.RepeatedCompositeFieldContainer[Chapter]
    resolutions: _containers.RepeatedCompositeFieldContainer[Resolution]
    evidence: _containers.RepeatedCompositeFieldContainer[EvidenceReference]
    incomplete: bool
    disconnect_reason: str
    disconnect_step: str
    def __init__(self, run_id: _Optional[str] = ..., disposition: _Optional[str] = ..., chapters: _Optional[_Iterable[_Union[Chapter, _Mapping]]] = ..., resolutions: _Optional[_Iterable[_Union[Resolution, _Mapping]]] = ..., evidence: _Optional[_Iterable[_Union[EvidenceReference, _Mapping]]] = ..., incomplete: _Optional[bool] = ..., disconnect_reason: _Optional[str] = ..., disconnect_step: _Optional[str] = ...) -> None: ...

class Chapter(_message.Message):
    __slots__ = ("id", "title", "disposition", "message")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    disposition: str
    message: str
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., disposition: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class Resolution(_message.Message):
    __slots__ = ("target", "rung", "confidence")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    RUNG_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    target: str
    rung: str
    confidence: float
    def __init__(self, target: _Optional[str] = ..., rung: _Optional[str] = ..., confidence: _Optional[float] = ...) -> None: ...

class EvidenceReference(_message.Message):
    __slots__ = ("id", "sha256", "size_bytes", "created_at", "redaction_verified", "recording_method", "effective_fps", "producer", "kind", "applied_rules", "opted_out", "claim_class", "minimum_useful_fps", "disposition", "disposition_reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    REDACTION_VERIFIED_FIELD_NUMBER: _ClassVar[int]
    RECORDING_METHOD_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_FPS_FIELD_NUMBER: _ClassVar[int]
    PRODUCER_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    APPLIED_RULES_FIELD_NUMBER: _ClassVar[int]
    OPTED_OUT_FIELD_NUMBER: _ClassVar[int]
    CLAIM_CLASS_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_USEFUL_FPS_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    sha256: str
    size_bytes: int
    created_at: str
    redaction_verified: bool
    recording_method: str
    effective_fps: float
    producer: str
    kind: str
    applied_rules: _containers.RepeatedScalarFieldContainer[str]
    opted_out: bool
    claim_class: str
    minimum_useful_fps: float
    disposition: str
    disposition_reason: str
    def __init__(self, id: _Optional[str] = ..., sha256: _Optional[str] = ..., size_bytes: _Optional[int] = ..., created_at: _Optional[str] = ..., redaction_verified: _Optional[bool] = ..., recording_method: _Optional[str] = ..., effective_fps: _Optional[float] = ..., producer: _Optional[str] = ..., kind: _Optional[str] = ..., applied_rules: _Optional[_Iterable[str]] = ..., opted_out: _Optional[bool] = ..., claim_class: _Optional[str] = ..., minimum_useful_fps: _Optional[float] = ..., disposition: _Optional[str] = ..., disposition_reason: _Optional[str] = ...) -> None: ...
