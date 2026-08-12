import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from vrooli_bridge.v1.shared import shared_pb2 as _shared_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Handshake(_message.Message):
    __slots__ = ("protocol_version", "node_id", "agent_version", "os", "arch", "capabilities", "supports_websocket")
    PROTOCOL_VERSION_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_VERSION_FIELD_NUMBER: _ClassVar[int]
    OS_FIELD_NUMBER: _ClassVar[int]
    ARCH_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_WEBSOCKET_FIELD_NUMBER: _ClassVar[int]
    protocol_version: int
    node_id: str
    agent_version: str
    os: str
    arch: str
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    supports_websocket: bool
    def __init__(self, protocol_version: _Optional[int] = ..., node_id: _Optional[str] = ..., agent_version: _Optional[str] = ..., os: _Optional[str] = ..., arch: _Optional[str] = ..., capabilities: _Optional[_Iterable[str]] = ..., supports_websocket: _Optional[bool] = ...) -> None: ...

class HandshakeAck(_message.Message):
    __slots__ = ("accepted", "compatibility", "control_plane_protocol_version", "session_id", "reason")
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    COMPATIBILITY_FIELD_NUMBER: _ClassVar[int]
    CONTROL_PLANE_PROTOCOL_VERSION_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    accepted: bool
    compatibility: _shared_pb2.CompatibilityStatus
    control_plane_protocol_version: int
    session_id: str
    reason: str
    def __init__(self, accepted: _Optional[bool] = ..., compatibility: _Optional[_Union[_shared_pb2.CompatibilityStatus, str]] = ..., control_plane_protocol_version: _Optional[int] = ..., session_id: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class JobPush(_message.Message):
    __slots__ = ("run_id", "scenario", "verb", "args", "timeout_seconds", "outputs")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    VERB_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    OUTPUTS_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    scenario: str
    verb: str
    args: _containers.RepeatedScalarFieldContainer[str]
    timeout_seconds: int
    outputs: _containers.RepeatedCompositeFieldContainer[ArtifactOutput]
    def __init__(self, run_id: _Optional[str] = ..., scenario: _Optional[str] = ..., verb: _Optional[str] = ..., args: _Optional[_Iterable[str]] = ..., timeout_seconds: _Optional[int] = ..., outputs: _Optional[_Iterable[_Union[ArtifactOutput, _Mapping]]] = ...) -> None: ...

class ArtifactOutput(_message.Message):
    __slots__ = ("name", "media_type", "output_flag", "max_bytes")
    NAME_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FLAG_FIELD_NUMBER: _ClassVar[int]
    MAX_BYTES_FIELD_NUMBER: _ClassVar[int]
    name: str
    media_type: str
    output_flag: str
    max_bytes: int
    def __init__(self, name: _Optional[str] = ..., media_type: _Optional[str] = ..., output_flag: _Optional[str] = ..., max_bytes: _Optional[int] = ...) -> None: ...

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

class ServerFrame(_message.Message):
    __slots__ = ("frame_id", "ack", "job", "provision", "ping", "abort", "session")
    FRAME_ID_FIELD_NUMBER: _ClassVar[int]
    ACK_FIELD_NUMBER: _ClassVar[int]
    JOB_FIELD_NUMBER: _ClassVar[int]
    PROVISION_FIELD_NUMBER: _ClassVar[int]
    PING_FIELD_NUMBER: _ClassVar[int]
    ABORT_FIELD_NUMBER: _ClassVar[int]
    SESSION_FIELD_NUMBER: _ClassVar[int]
    frame_id: str
    ack: HandshakeAck
    job: JobPush
    provision: ProvisionCommand
    ping: ControlPing
    abort: AbortJob
    session: _shared_pb2.SessionFrame
    def __init__(self, frame_id: _Optional[str] = ..., ack: _Optional[_Union[HandshakeAck, _Mapping]] = ..., job: _Optional[_Union[JobPush, _Mapping]] = ..., provision: _Optional[_Union[ProvisionCommand, _Mapping]] = ..., ping: _Optional[_Union[ControlPing, _Mapping]] = ..., abort: _Optional[_Union[AbortJob, _Mapping]] = ..., session: _Optional[_Union[_shared_pb2.SessionFrame, _Mapping]] = ...) -> None: ...

class SignedServerFrame(_message.Message):
    __slots__ = ("frame", "signature")
    FRAME_FIELD_NUMBER: _ClassVar[int]
    SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    frame: bytes
    signature: bytes
    def __init__(self, frame: _Optional[bytes] = ..., signature: _Optional[bytes] = ...) -> None: ...

class NodeFrame(_message.Message):
    __slots__ = ("handshake", "heartbeat", "run_event", "delivery_ack", "session")
    HANDSHAKE_FIELD_NUMBER: _ClassVar[int]
    HEARTBEAT_FIELD_NUMBER: _ClassVar[int]
    RUN_EVENT_FIELD_NUMBER: _ClassVar[int]
    DELIVERY_ACK_FIELD_NUMBER: _ClassVar[int]
    SESSION_FIELD_NUMBER: _ClassVar[int]
    handshake: Handshake
    heartbeat: _shared_pb2.Heartbeat
    run_event: _shared_pb2.RunEvent
    delivery_ack: _shared_pb2.DeliveryAck
    session: _shared_pb2.SessionFrame
    def __init__(self, handshake: _Optional[_Union[Handshake, _Mapping]] = ..., heartbeat: _Optional[_Union[_shared_pb2.Heartbeat, _Mapping]] = ..., run_event: _Optional[_Union[_shared_pb2.RunEvent, _Mapping]] = ..., delivery_ack: _Optional[_Union[_shared_pb2.DeliveryAck, _Mapping]] = ..., session: _Optional[_Union[_shared_pb2.SessionFrame, _Mapping]] = ...) -> None: ...
