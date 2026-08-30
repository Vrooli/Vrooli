import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from vrooli_bridge.v1.shared import shared_pb2 as _shared_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PrivilegedOperation(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PRIVILEGED_OPERATION_UNSPECIFIED: _ClassVar[PrivilegedOperation]
    PRIVILEGED_OPERATION_PROVISION: _ClassVar[PrivilegedOperation]
    PRIVILEGED_OPERATION_INVENTORY_INSTALLATION: _ClassVar[PrivilegedOperation]
    PRIVILEGED_OPERATION_PLAN_UNINSTALL: _ClassVar[PrivilegedOperation]
    PRIVILEGED_OPERATION_PROVISION_BREAK_GLASS: _ClassVar[PrivilegedOperation]
    PRIVILEGED_OPERATION_ISSUE_CLEANUP_CAPABILITY: _ClassVar[PrivilegedOperation]
    PRIVILEGED_OPERATION_APPLY_FROZEN_PLAN: _ClassVar[PrivilegedOperation]
    PRIVILEGED_OPERATION_VERIFY_RESULT: _ClassVar[PrivilegedOperation]
    PRIVILEGED_OPERATION_ROTATE_BREAK_GLASS: _ClassVar[PrivilegedOperation]
    PRIVILEGED_OPERATION_RESET_BREAK_GLASS: _ClassVar[PrivilegedOperation]
PRIVILEGED_OPERATION_UNSPECIFIED: PrivilegedOperation
PRIVILEGED_OPERATION_PROVISION: PrivilegedOperation
PRIVILEGED_OPERATION_INVENTORY_INSTALLATION: PrivilegedOperation
PRIVILEGED_OPERATION_PLAN_UNINSTALL: PrivilegedOperation
PRIVILEGED_OPERATION_PROVISION_BREAK_GLASS: PrivilegedOperation
PRIVILEGED_OPERATION_ISSUE_CLEANUP_CAPABILITY: PrivilegedOperation
PRIVILEGED_OPERATION_APPLY_FROZEN_PLAN: PrivilegedOperation
PRIVILEGED_OPERATION_VERIFY_RESULT: PrivilegedOperation
PRIVILEGED_OPERATION_ROTATE_BREAK_GLASS: PrivilegedOperation
PRIVILEGED_OPERATION_RESET_BREAK_GLASS: PrivilegedOperation

class Handshake(_message.Message):
    __slots__ = ("protocol_version", "node_id", "agent_version", "os", "arch", "capabilities", "supports_websocket", "machine_arch", "binary_arch")
    PROTOCOL_VERSION_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_VERSION_FIELD_NUMBER: _ClassVar[int]
    OS_FIELD_NUMBER: _ClassVar[int]
    ARCH_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_WEBSOCKET_FIELD_NUMBER: _ClassVar[int]
    MACHINE_ARCH_FIELD_NUMBER: _ClassVar[int]
    BINARY_ARCH_FIELD_NUMBER: _ClassVar[int]
    protocol_version: int
    node_id: str
    agent_version: str
    os: str
    arch: str
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    supports_websocket: bool
    machine_arch: str
    binary_arch: str
    def __init__(self, protocol_version: _Optional[int] = ..., node_id: _Optional[str] = ..., agent_version: _Optional[str] = ..., os: _Optional[str] = ..., arch: _Optional[str] = ..., capabilities: _Optional[_Iterable[str]] = ..., supports_websocket: _Optional[bool] = ..., machine_arch: _Optional[str] = ..., binary_arch: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("run_id", "scenario", "verb", "args", "timeout_seconds", "outputs", "credential_injections")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    VERB_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    OUTPUTS_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_INJECTIONS_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    scenario: str
    verb: str
    args: _containers.RepeatedScalarFieldContainer[str]
    timeout_seconds: int
    outputs: _containers.RepeatedCompositeFieldContainer[ArtifactOutput]
    credential_injections: _containers.RepeatedCompositeFieldContainer[CredentialInjection]
    def __init__(self, run_id: _Optional[str] = ..., scenario: _Optional[str] = ..., verb: _Optional[str] = ..., args: _Optional[_Iterable[str]] = ..., timeout_seconds: _Optional[int] = ..., outputs: _Optional[_Iterable[_Union[ArtifactOutput, _Mapping]]] = ..., credential_injections: _Optional[_Iterable[_Union[CredentialInjection, _Mapping]]] = ...) -> None: ...

class CredentialInjection(_message.Message):
    __slots__ = ("logical_id", "field", "env_name")
    LOGICAL_ID_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    ENV_NAME_FIELD_NUMBER: _ClassVar[int]
    logical_id: str
    field: str
    env_name: str
    def __init__(self, logical_id: _Optional[str] = ..., field: _Optional[str] = ..., env_name: _Optional[str] = ...) -> None: ...

class CleanupCommand(_message.Message):
    __slots__ = ("operation", "op_id", "machine_id", "node_id", "target", "scope", "plan_id", "plan_hash", "sealed_passphrase", "capability", "operator_confirmed", "operator_id")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    OP_ID_FIELD_NUMBER: _ClassVar[int]
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_HASH_FIELD_NUMBER: _ClassVar[int]
    SEALED_PASSPHRASE_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_CONFIRMED_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_ID_FIELD_NUMBER: _ClassVar[int]
    operation: PrivilegedOperation
    op_id: str
    machine_id: str
    node_id: str
    target: str
    scope: str
    plan_id: str
    plan_hash: str
    sealed_passphrase: bytes
    capability: bytes
    operator_confirmed: bool
    operator_id: str
    def __init__(self, operation: _Optional[_Union[PrivilegedOperation, str]] = ..., op_id: _Optional[str] = ..., machine_id: _Optional[str] = ..., node_id: _Optional[str] = ..., target: _Optional[str] = ..., scope: _Optional[str] = ..., plan_id: _Optional[str] = ..., plan_hash: _Optional[str] = ..., sealed_passphrase: _Optional[bytes] = ..., capability: _Optional[bytes] = ..., operator_confirmed: _Optional[bool] = ..., operator_id: _Optional[str] = ...) -> None: ...

class CredentialPush(_message.Message):
    __slots__ = ("grant_id", "node_id", "logical_id", "field", "generation", "retention", "sealed_value", "aad")
    GRANT_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    LOGICAL_ID_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    RETENTION_FIELD_NUMBER: _ClassVar[int]
    SEALED_VALUE_FIELD_NUMBER: _ClassVar[int]
    AAD_FIELD_NUMBER: _ClassVar[int]
    grant_id: str
    node_id: str
    logical_id: str
    field: str
    generation: int
    retention: str
    sealed_value: bytes
    aad: bytes
    def __init__(self, grant_id: _Optional[str] = ..., node_id: _Optional[str] = ..., logical_id: _Optional[str] = ..., field: _Optional[str] = ..., generation: _Optional[int] = ..., retention: _Optional[str] = ..., sealed_value: _Optional[bytes] = ..., aad: _Optional[bytes] = ...) -> None: ...

class CredentialPurge(_message.Message):
    __slots__ = ("node_id", "addresses")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    ADDRESSES_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    addresses: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, node_id: _Optional[str] = ..., addresses: _Optional[_Iterable[str]] = ...) -> None: ...

class CredentialGrant(_message.Message):
    __slots__ = ("grant_id", "node_id", "logical_id", "field", "retention", "generation", "revoked")
    GRANT_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    LOGICAL_ID_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    CLASS_FIELD_NUMBER: _ClassVar[int]
    RETENTION_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    REVOKED_FIELD_NUMBER: _ClassVar[int]
    grant_id: str
    node_id: str
    logical_id: str
    field: str
    retention: str
    generation: int
    revoked: bool
    def __init__(self, grant_id: _Optional[str] = ..., node_id: _Optional[str] = ..., logical_id: _Optional[str] = ..., field: _Optional[str] = ..., retention: _Optional[str] = ..., generation: _Optional[int] = ..., revoked: _Optional[bool] = ..., **kwargs) -> None: ...

class CredentialReceipt(_message.Message):
    __slots__ = ("grant_id", "node_id", "logical_id", "field", "generation", "accepted", "reason")
    GRANT_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    LOGICAL_ID_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    grant_id: str
    node_id: str
    logical_id: str
    field: str
    generation: int
    accepted: bool
    reason: str
    def __init__(self, grant_id: _Optional[str] = ..., node_id: _Optional[str] = ..., logical_id: _Optional[str] = ..., field: _Optional[str] = ..., generation: _Optional[int] = ..., accepted: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

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

class RelayRequest(_message.Message):
    __slots__ = ("correlation_id", "scenario", "command", "args", "timeout_seconds", "max_response_bytes")
    CORRELATION_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    MAX_RESPONSE_BYTES_FIELD_NUMBER: _ClassVar[int]
    correlation_id: str
    scenario: str
    command: str
    args: _containers.RepeatedScalarFieldContainer[str]
    timeout_seconds: int
    max_response_bytes: int
    def __init__(self, correlation_id: _Optional[str] = ..., scenario: _Optional[str] = ..., command: _Optional[str] = ..., args: _Optional[_Iterable[str]] = ..., timeout_seconds: _Optional[int] = ..., max_response_bytes: _Optional[int] = ...) -> None: ...

class RelayCancel(_message.Message):
    __slots__ = ("correlation_id", "reason")
    CORRELATION_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    correlation_id: str
    reason: str
    def __init__(self, correlation_id: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class ServerFrame(_message.Message):
    __slots__ = ("frame_id", "ack", "job", "provision", "ping", "abort", "session", "relay", "relay_cancel", "cleanup", "credential_push", "credential_purge", "credential_grant")
    FRAME_ID_FIELD_NUMBER: _ClassVar[int]
    ACK_FIELD_NUMBER: _ClassVar[int]
    JOB_FIELD_NUMBER: _ClassVar[int]
    PROVISION_FIELD_NUMBER: _ClassVar[int]
    PING_FIELD_NUMBER: _ClassVar[int]
    ABORT_FIELD_NUMBER: _ClassVar[int]
    SESSION_FIELD_NUMBER: _ClassVar[int]
    RELAY_FIELD_NUMBER: _ClassVar[int]
    RELAY_CANCEL_FIELD_NUMBER: _ClassVar[int]
    CLEANUP_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_PUSH_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_PURGE_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_GRANT_FIELD_NUMBER: _ClassVar[int]
    frame_id: str
    ack: HandshakeAck
    job: JobPush
    provision: ProvisionCommand
    ping: ControlPing
    abort: AbortJob
    session: _shared_pb2.SessionFrame
    relay: RelayRequest
    relay_cancel: RelayCancel
    cleanup: CleanupCommand
    credential_push: CredentialPush
    credential_purge: CredentialPurge
    credential_grant: CredentialGrant
    def __init__(self, frame_id: _Optional[str] = ..., ack: _Optional[_Union[HandshakeAck, _Mapping]] = ..., job: _Optional[_Union[JobPush, _Mapping]] = ..., provision: _Optional[_Union[ProvisionCommand, _Mapping]] = ..., ping: _Optional[_Union[ControlPing, _Mapping]] = ..., abort: _Optional[_Union[AbortJob, _Mapping]] = ..., session: _Optional[_Union[_shared_pb2.SessionFrame, _Mapping]] = ..., relay: _Optional[_Union[RelayRequest, _Mapping]] = ..., relay_cancel: _Optional[_Union[RelayCancel, _Mapping]] = ..., cleanup: _Optional[_Union[CleanupCommand, _Mapping]] = ..., credential_push: _Optional[_Union[CredentialPush, _Mapping]] = ..., credential_purge: _Optional[_Union[CredentialPurge, _Mapping]] = ..., credential_grant: _Optional[_Union[CredentialGrant, _Mapping]] = ...) -> None: ...

class SignedServerFrame(_message.Message):
    __slots__ = ("frame", "signature")
    FRAME_FIELD_NUMBER: _ClassVar[int]
    SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    frame: bytes
    signature: bytes
    def __init__(self, frame: _Optional[bytes] = ..., signature: _Optional[bytes] = ...) -> None: ...

class NodeFrame(_message.Message):
    __slots__ = ("handshake", "heartbeat", "run_event", "delivery_ack", "session", "relay_response", "credential_receipt")
    HANDSHAKE_FIELD_NUMBER: _ClassVar[int]
    HEARTBEAT_FIELD_NUMBER: _ClassVar[int]
    RUN_EVENT_FIELD_NUMBER: _ClassVar[int]
    DELIVERY_ACK_FIELD_NUMBER: _ClassVar[int]
    SESSION_FIELD_NUMBER: _ClassVar[int]
    RELAY_RESPONSE_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_RECEIPT_FIELD_NUMBER: _ClassVar[int]
    handshake: Handshake
    heartbeat: _shared_pb2.Heartbeat
    run_event: _shared_pb2.RunEvent
    delivery_ack: _shared_pb2.DeliveryAck
    session: _shared_pb2.SessionFrame
    relay_response: _shared_pb2.RelayResponse
    credential_receipt: CredentialReceipt
    def __init__(self, handshake: _Optional[_Union[Handshake, _Mapping]] = ..., heartbeat: _Optional[_Union[_shared_pb2.Heartbeat, _Mapping]] = ..., run_event: _Optional[_Union[_shared_pb2.RunEvent, _Mapping]] = ..., delivery_ack: _Optional[_Union[_shared_pb2.DeliveryAck, _Mapping]] = ..., session: _Optional[_Union[_shared_pb2.SessionFrame, _Mapping]] = ..., relay_response: _Optional[_Union[_shared_pb2.RelayResponse, _Mapping]] = ..., credential_receipt: _Optional[_Union[CredentialReceipt, _Mapping]] = ...) -> None: ...
