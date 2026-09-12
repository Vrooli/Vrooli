import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CleanupStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CLEANUP_STATUS_UNSPECIFIED: _ClassVar[CleanupStatus]
    CLEANUP_STATUS_QUEUED: _ClassVar[CleanupStatus]
    CLEANUP_STATUS_PLANNING: _ClassVar[CleanupStatus]
    CLEANUP_STATUS_PLANNED: _ClassVar[CleanupStatus]
    CLEANUP_STATUS_CONFIRMED: _ClassVar[CleanupStatus]
    CLEANUP_STATUS_APPLYING: _ClassVar[CleanupStatus]
    CLEANUP_STATUS_COMPLETED: _ClassVar[CleanupStatus]
    CLEANUP_STATUS_FAILED: _ClassVar[CleanupStatus]
    CLEANUP_STATUS_BLOCKED: _ClassVar[CleanupStatus]
    CLEANUP_STATUS_CANCELED: _ClassVar[CleanupStatus]

class CleanupEventKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CLEANUP_EVENT_KIND_UNSPECIFIED: _ClassVar[CleanupEventKind]
    CLEANUP_EVENT_KIND_STATUS: _ClassVar[CleanupEventKind]
    CLEANUP_EVENT_KIND_LOG: _ClassVar[CleanupEventKind]
    CLEANUP_EVENT_KIND_PLAN: _ClassVar[CleanupEventKind]
    CLEANUP_EVENT_KIND_RECEIPT: _ClassVar[CleanupEventKind]
    CLEANUP_EVENT_KIND_EXIT: _ClassVar[CleanupEventKind]
CLEANUP_STATUS_UNSPECIFIED: CleanupStatus
CLEANUP_STATUS_QUEUED: CleanupStatus
CLEANUP_STATUS_PLANNING: CleanupStatus
CLEANUP_STATUS_PLANNED: CleanupStatus
CLEANUP_STATUS_CONFIRMED: CleanupStatus
CLEANUP_STATUS_APPLYING: CleanupStatus
CLEANUP_STATUS_COMPLETED: CleanupStatus
CLEANUP_STATUS_FAILED: CleanupStatus
CLEANUP_STATUS_BLOCKED: CleanupStatus
CLEANUP_STATUS_CANCELED: CleanupStatus
CLEANUP_EVENT_KIND_UNSPECIFIED: CleanupEventKind
CLEANUP_EVENT_KIND_STATUS: CleanupEventKind
CLEANUP_EVENT_KIND_LOG: CleanupEventKind
CLEANUP_EVENT_KIND_PLAN: CleanupEventKind
CLEANUP_EVENT_KIND_RECEIPT: CleanupEventKind
CLEANUP_EVENT_KIND_EXIT: CleanupEventKind

class CleanupOperation(_message.Message):
    __slots__ = ("id", "machine_id", "node_id", "target", "scope", "status", "transport", "transport_reason", "reason", "plan_hash", "plan_json", "receipt_json", "operator_id", "created_at", "updated_at", "finished_at", "sealing_public_key")
    ID_FIELD_NUMBER: _ClassVar[int]
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TRANSPORT_FIELD_NUMBER: _ClassVar[int]
    TRANSPORT_REASON_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    PLAN_HASH_FIELD_NUMBER: _ClassVar[int]
    PLAN_JSON_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_JSON_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    SEALING_PUBLIC_KEY_FIELD_NUMBER: _ClassVar[int]
    id: str
    machine_id: str
    node_id: str
    target: str
    scope: str
    status: CleanupStatus
    transport: str
    transport_reason: str
    reason: str
    plan_hash: str
    plan_json: bytes
    receipt_json: bytes
    operator_id: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    finished_at: _timestamp_pb2.Timestamp
    sealing_public_key: bytes
    def __init__(self, id: _Optional[str] = ..., machine_id: _Optional[str] = ..., node_id: _Optional[str] = ..., target: _Optional[str] = ..., scope: _Optional[str] = ..., status: _Optional[_Union[CleanupStatus, str]] = ..., transport: _Optional[str] = ..., transport_reason: _Optional[str] = ..., reason: _Optional[str] = ..., plan_hash: _Optional[str] = ..., plan_json: _Optional[bytes] = ..., receipt_json: _Optional[bytes] = ..., operator_id: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., finished_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., sealing_public_key: _Optional[bytes] = ...) -> None: ...

class CleanupEvent(_message.Message):
    __slots__ = ("operation_id", "kind", "sequence", "status", "log_chunk", "plan_json", "receipt_json", "reason", "exit_code", "emitted_at")
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LOG_CHUNK_FIELD_NUMBER: _ClassVar[int]
    PLAN_JSON_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_JSON_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    EMITTED_AT_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    kind: CleanupEventKind
    sequence: int
    status: str
    log_chunk: str
    plan_json: bytes
    receipt_json: bytes
    reason: str
    exit_code: int
    emitted_at: _timestamp_pb2.Timestamp
    def __init__(self, operation_id: _Optional[str] = ..., kind: _Optional[_Union[CleanupEventKind, str]] = ..., sequence: _Optional[int] = ..., status: _Optional[str] = ..., log_chunk: _Optional[str] = ..., plan_json: _Optional[bytes] = ..., receipt_json: _Optional[bytes] = ..., reason: _Optional[str] = ..., exit_code: _Optional[int] = ..., emitted_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CleanupTarget(_message.Message):
    __slots__ = ("machine_id", "node_id", "target", "scope", "transport", "transport_reason", "operator_id", "sealing_public_key", "operation_id", "capabilities", "approved_scopes")
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    TRANSPORT_FIELD_NUMBER: _ClassVar[int]
    TRANSPORT_REASON_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_ID_FIELD_NUMBER: _ClassVar[int]
    SEALING_PUBLIC_KEY_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    APPROVED_SCOPES_FIELD_NUMBER: _ClassVar[int]
    machine_id: str
    node_id: str
    target: str
    scope: str
    transport: str
    transport_reason: str
    operator_id: str
    sealing_public_key: bytes
    operation_id: str
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    approved_scopes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, machine_id: _Optional[str] = ..., node_id: _Optional[str] = ..., target: _Optional[str] = ..., scope: _Optional[str] = ..., transport: _Optional[str] = ..., transport_reason: _Optional[str] = ..., operator_id: _Optional[str] = ..., sealing_public_key: _Optional[bytes] = ..., operation_id: _Optional[str] = ..., capabilities: _Optional[_Iterable[str]] = ..., approved_scopes: _Optional[_Iterable[str]] = ...) -> None: ...

class PrepareCleanupRequest(_message.Message):
    __slots__ = ("machine_id", "node_id", "target", "scope")
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    machine_id: str
    node_id: str
    target: str
    scope: str
    def __init__(self, machine_id: _Optional[str] = ..., node_id: _Optional[str] = ..., target: _Optional[str] = ..., scope: _Optional[str] = ...) -> None: ...

class PrepareCleanupResponse(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: CleanupTarget
    def __init__(self, target: _Optional[_Union[CleanupTarget, _Mapping]] = ...) -> None: ...

class ProvisionBreakGlassRequest(_message.Message):
    __slots__ = ("machine_id", "node_id", "target", "scope", "sealed_passphrase", "operator_id", "operation_id")
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    SEALED_PASSPHRASE_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_ID_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    machine_id: str
    node_id: str
    target: str
    scope: str
    sealed_passphrase: bytes
    operator_id: str
    operation_id: str
    def __init__(self, machine_id: _Optional[str] = ..., node_id: _Optional[str] = ..., target: _Optional[str] = ..., scope: _Optional[str] = ..., sealed_passphrase: _Optional[bytes] = ..., operator_id: _Optional[str] = ..., operation_id: _Optional[str] = ...) -> None: ...

class ProvisionBreakGlassResponse(_message.Message):
    __slots__ = ("operation",)
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    operation: CleanupOperation
    def __init__(self, operation: _Optional[_Union[CleanupOperation, _Mapping]] = ...) -> None: ...

class ResetBreakGlassRequest(_message.Message):
    __slots__ = ("machine_id", "node_id", "target", "scope")
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    machine_id: str
    node_id: str
    target: str
    scope: str
    def __init__(self, machine_id: _Optional[str] = ..., node_id: _Optional[str] = ..., target: _Optional[str] = ..., scope: _Optional[str] = ...) -> None: ...

class ResetBreakGlassResponse(_message.Message):
    __slots__ = ("operation",)
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    operation: CleanupOperation
    def __init__(self, operation: _Optional[_Union[CleanupOperation, _Mapping]] = ...) -> None: ...

class StartCleanupRequest(_message.Message):
    __slots__ = ("machine_id", "node_id", "target", "scope")
    MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    machine_id: str
    node_id: str
    target: str
    scope: str
    def __init__(self, machine_id: _Optional[str] = ..., node_id: _Optional[str] = ..., target: _Optional[str] = ..., scope: _Optional[str] = ...) -> None: ...

class StartCleanupResponse(_message.Message):
    __slots__ = ("operation",)
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    operation: CleanupOperation
    def __init__(self, operation: _Optional[_Union[CleanupOperation, _Mapping]] = ...) -> None: ...

class GetCleanupRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetCleanupResponse(_message.Message):
    __slots__ = ("operation", "events")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    operation: CleanupOperation
    events: _containers.RepeatedCompositeFieldContainer[CleanupEvent]
    def __init__(self, operation: _Optional[_Union[CleanupOperation, _Mapping]] = ..., events: _Optional[_Iterable[_Union[CleanupEvent, _Mapping]]] = ...) -> None: ...

class PlanCleanupRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class PlanCleanupResponse(_message.Message):
    __slots__ = ("operation",)
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    operation: CleanupOperation
    def __init__(self, operation: _Optional[_Union[CleanupOperation, _Mapping]] = ...) -> None: ...

class ConfirmCleanupRequest(_message.Message):
    __slots__ = ("id", "target", "plan_hash", "sealed_passphrase", "capability", "operator_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    PLAN_HASH_FIELD_NUMBER: _ClassVar[int]
    SEALED_PASSPHRASE_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    target: str
    plan_hash: str
    sealed_passphrase: bytes
    capability: bytes
    operator_id: str
    def __init__(self, id: _Optional[str] = ..., target: _Optional[str] = ..., plan_hash: _Optional[str] = ..., sealed_passphrase: _Optional[bytes] = ..., capability: _Optional[bytes] = ..., operator_id: _Optional[str] = ...) -> None: ...

class ConfirmCleanupResponse(_message.Message):
    __slots__ = ("operation",)
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    operation: CleanupOperation
    def __init__(self, operation: _Optional[_Union[CleanupOperation, _Mapping]] = ...) -> None: ...

class ApplyCleanupRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ApplyCleanupResponse(_message.Message):
    __slots__ = ("operation",)
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    operation: CleanupOperation
    def __init__(self, operation: _Optional[_Union[CleanupOperation, _Mapping]] = ...) -> None: ...

class VerifyCleanupRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class VerifyCleanupResponse(_message.Message):
    __slots__ = ("operation",)
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    operation: CleanupOperation
    def __init__(self, operation: _Optional[_Union[CleanupOperation, _Mapping]] = ...) -> None: ...

class CancelCleanupRequest(_message.Message):
    __slots__ = ("id", "reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    reason: str
    def __init__(self, id: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class CancelCleanupResponse(_message.Message):
    __slots__ = ("operation",)
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    operation: CleanupOperation
    def __init__(self, operation: _Optional[_Union[CleanupOperation, _Mapping]] = ...) -> None: ...

class ReportCleanupEventRequest(_message.Message):
    __slots__ = ("event",)
    EVENT_FIELD_NUMBER: _ClassVar[int]
    event: CleanupEvent
    def __init__(self, event: _Optional[_Union[CleanupEvent, _Mapping]] = ...) -> None: ...

class ReportCleanupEventResponse(_message.Message):
    __slots__ = ("accepted",)
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    accepted: bool
    def __init__(self, accepted: _Optional[bool] = ...) -> None: ...
