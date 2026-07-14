import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CompatibilityStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COMPATIBILITY_STATUS_UNSPECIFIED: _ClassVar[CompatibilityStatus]
    COMPATIBILITY_STATUS_OK: _ClassVar[CompatibilityStatus]
    COMPATIBILITY_STATUS_NEEDS_UPDATE: _ClassVar[CompatibilityStatus]
    COMPATIBILITY_STATUS_INCOMPATIBLE: _ClassVar[CompatibilityStatus]

class RunEventKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RUN_EVENT_KIND_UNSPECIFIED: _ClassVar[RunEventKind]
    RUN_EVENT_KIND_LOG: _ClassVar[RunEventKind]
    RUN_EVENT_KIND_STATUS: _ClassVar[RunEventKind]
    RUN_EVENT_KIND_EXIT: _ClassVar[RunEventKind]
    RUN_EVENT_KIND_ARTIFACT_REF: _ClassVar[RunEventKind]
COMPATIBILITY_STATUS_UNSPECIFIED: CompatibilityStatus
COMPATIBILITY_STATUS_OK: CompatibilityStatus
COMPATIBILITY_STATUS_NEEDS_UPDATE: CompatibilityStatus
COMPATIBILITY_STATUS_INCOMPATIBLE: CompatibilityStatus
RUN_EVENT_KIND_UNSPECIFIED: RunEventKind
RUN_EVENT_KIND_LOG: RunEventKind
RUN_EVENT_KIND_STATUS: RunEventKind
RUN_EVENT_KIND_EXIT: RunEventKind
RUN_EVENT_KIND_ARTIFACT_REF: RunEventKind

class Handshake(_message.Message):
    __slots__ = ("protocol_version", "node_id", "agent_version", "os", "arch", "capabilities")
    PROTOCOL_VERSION_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_VERSION_FIELD_NUMBER: _ClassVar[int]
    OS_FIELD_NUMBER: _ClassVar[int]
    ARCH_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    protocol_version: int
    node_id: str
    agent_version: str
    os: str
    arch: str
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, protocol_version: _Optional[int] = ..., node_id: _Optional[str] = ..., agent_version: _Optional[str] = ..., os: _Optional[str] = ..., arch: _Optional[str] = ..., capabilities: _Optional[_Iterable[str]] = ...) -> None: ...

class HandshakeAck(_message.Message):
    __slots__ = ("accepted", "compatibility", "control_plane_protocol_version", "session_id", "reason")
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    COMPATIBILITY_FIELD_NUMBER: _ClassVar[int]
    CONTROL_PLANE_PROTOCOL_VERSION_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    accepted: bool
    compatibility: CompatibilityStatus
    control_plane_protocol_version: int
    session_id: str
    reason: str
    def __init__(self, accepted: _Optional[bool] = ..., compatibility: _Optional[_Union[CompatibilityStatus, str]] = ..., control_plane_protocol_version: _Optional[int] = ..., session_id: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class HealthSnapshot(_message.Message):
    __slots__ = ("toolchain_present", "disk_headroom_bytes", "container_runtime_up", "details", "reported_at")
    class DetailsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TOOLCHAIN_PRESENT_FIELD_NUMBER: _ClassVar[int]
    DISK_HEADROOM_BYTES_FIELD_NUMBER: _ClassVar[int]
    CONTAINER_RUNTIME_UP_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    REPORTED_AT_FIELD_NUMBER: _ClassVar[int]
    toolchain_present: bool
    disk_headroom_bytes: int
    container_runtime_up: bool
    details: _containers.ScalarMap[str, str]
    reported_at: _timestamp_pb2.Timestamp
    def __init__(self, toolchain_present: _Optional[bool] = ..., disk_headroom_bytes: _Optional[int] = ..., container_runtime_up: _Optional[bool] = ..., details: _Optional[_Mapping[str, str]] = ..., reported_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Heartbeat(_message.Message):
    __slots__ = ("node_id", "sequence", "health", "sent_at")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    HEALTH_FIELD_NUMBER: _ClassVar[int]
    SENT_AT_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    sequence: int
    health: HealthSnapshot
    sent_at: _timestamp_pb2.Timestamp
    def __init__(self, node_id: _Optional[str] = ..., sequence: _Optional[int] = ..., health: _Optional[_Union[HealthSnapshot, _Mapping]] = ..., sent_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class JobPush(_message.Message):
    __slots__ = ("run_id", "scenario", "verb", "args", "timeout_seconds")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    VERB_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    scenario: str
    verb: str
    args: _containers.RepeatedScalarFieldContainer[str]
    timeout_seconds: int
    def __init__(self, run_id: _Optional[str] = ..., scenario: _Optional[str] = ..., verb: _Optional[str] = ..., args: _Optional[_Iterable[str]] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class ProvisionCommand(_message.Message):
    __slots__ = ("op_id", "target_revision", "rollback_revision")
    OP_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_REVISION_FIELD_NUMBER: _ClassVar[int]
    ROLLBACK_REVISION_FIELD_NUMBER: _ClassVar[int]
    op_id: str
    target_revision: str
    rollback_revision: str
    def __init__(self, op_id: _Optional[str] = ..., target_revision: _Optional[str] = ..., rollback_revision: _Optional[str] = ...) -> None: ...

class ControlPing(_message.Message):
    __slots__ = ("sent_at",)
    SENT_AT_FIELD_NUMBER: _ClassVar[int]
    sent_at: _timestamp_pb2.Timestamp
    def __init__(self, sent_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class AbortJob(_message.Message):
    __slots__ = ("run_id", "reason")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    reason: str
    def __init__(self, run_id: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class RunEvent(_message.Message):
    __slots__ = ("run_id", "kind", "sequence", "log_chunk", "status", "exit_code", "artifact_ref", "emitted_at")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    LOG_CHUNK_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_REF_FIELD_NUMBER: _ClassVar[int]
    EMITTED_AT_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    kind: RunEventKind
    sequence: int
    log_chunk: str
    status: str
    exit_code: int
    artifact_ref: str
    emitted_at: _timestamp_pb2.Timestamp
    def __init__(self, run_id: _Optional[str] = ..., kind: _Optional[_Union[RunEventKind, str]] = ..., sequence: _Optional[int] = ..., log_chunk: _Optional[str] = ..., status: _Optional[str] = ..., exit_code: _Optional[int] = ..., artifact_ref: _Optional[str] = ..., emitted_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ServerFrame(_message.Message):
    __slots__ = ("ack", "job", "provision", "ping", "abort")
    ACK_FIELD_NUMBER: _ClassVar[int]
    JOB_FIELD_NUMBER: _ClassVar[int]
    PROVISION_FIELD_NUMBER: _ClassVar[int]
    PING_FIELD_NUMBER: _ClassVar[int]
    ABORT_FIELD_NUMBER: _ClassVar[int]
    ack: HandshakeAck
    job: JobPush
    provision: ProvisionCommand
    ping: ControlPing
    abort: AbortJob
    def __init__(self, ack: _Optional[_Union[HandshakeAck, _Mapping]] = ..., job: _Optional[_Union[JobPush, _Mapping]] = ..., provision: _Optional[_Union[ProvisionCommand, _Mapping]] = ..., ping: _Optional[_Union[ControlPing, _Mapping]] = ..., abort: _Optional[_Union[AbortJob, _Mapping]] = ...) -> None: ...

class SignedServerFrame(_message.Message):
    __slots__ = ("frame", "signature")
    FRAME_FIELD_NUMBER: _ClassVar[int]
    SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    frame: bytes
    signature: bytes
    def __init__(self, frame: _Optional[bytes] = ..., signature: _Optional[bytes] = ...) -> None: ...

class NodeFrame(_message.Message):
    __slots__ = ("handshake", "heartbeat", "run_event")
    HANDSHAKE_FIELD_NUMBER: _ClassVar[int]
    HEARTBEAT_FIELD_NUMBER: _ClassVar[int]
    RUN_EVENT_FIELD_NUMBER: _ClassVar[int]
    handshake: Handshake
    heartbeat: Heartbeat
    run_event: RunEvent
    def __init__(self, handshake: _Optional[_Union[Handshake, _Mapping]] = ..., heartbeat: _Optional[_Union[Heartbeat, _Mapping]] = ..., run_event: _Optional[_Union[RunEvent, _Mapping]] = ...) -> None: ...
