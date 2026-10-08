import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ProvisioningStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PROVISIONING_STATUS_UNSPECIFIED: _ClassVar[ProvisioningStatus]
    PROVISIONING_STATUS_QUEUED: _ClassVar[ProvisioningStatus]
    PROVISIONING_STATUS_RUNNING: _ClassVar[ProvisioningStatus]
    PROVISIONING_STATUS_COMPLETED: _ClassVar[ProvisioningStatus]
    PROVISIONING_STATUS_FAILED: _ClassVar[ProvisioningStatus]
    PROVISIONING_STATUS_ROLLED_BACK: _ClassVar[ProvisioningStatus]

class ProvisionEventKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PROVISION_EVENT_KIND_UNSPECIFIED: _ClassVar[ProvisionEventKind]
    PROVISION_EVENT_KIND_LOG: _ClassVar[ProvisionEventKind]
    PROVISION_EVENT_KIND_STATUS: _ClassVar[ProvisionEventKind]
    PROVISION_EVENT_KIND_VERSION: _ClassVar[ProvisionEventKind]
    PROVISION_EVENT_KIND_EXIT: _ClassVar[ProvisionEventKind]
PROVISIONING_STATUS_UNSPECIFIED: ProvisioningStatus
PROVISIONING_STATUS_QUEUED: ProvisioningStatus
PROVISIONING_STATUS_RUNNING: ProvisioningStatus
PROVISIONING_STATUS_COMPLETED: ProvisioningStatus
PROVISIONING_STATUS_FAILED: ProvisioningStatus
PROVISIONING_STATUS_ROLLED_BACK: ProvisioningStatus
PROVISION_EVENT_KIND_UNSPECIFIED: ProvisionEventKind
PROVISION_EVENT_KIND_LOG: ProvisionEventKind
PROVISION_EVENT_KIND_STATUS: ProvisionEventKind
PROVISION_EVENT_KIND_VERSION: ProvisionEventKind
PROVISION_EVENT_KIND_EXIT: ProvisionEventKind

class ProvisioningOp(_message.Message):
    __slots__ = ("id", "node_id", "target_revision", "rollback_revision", "status", "resulting_revision", "exit_code", "created_at", "started_at", "finished_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_REVISION_FIELD_NUMBER: _ClassVar[int]
    ROLLBACK_REVISION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    RESULTING_REVISION_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    node_id: str
    target_revision: str
    rollback_revision: str
    status: ProvisioningStatus
    resulting_revision: str
    exit_code: int
    created_at: _timestamp_pb2.Timestamp
    started_at: _timestamp_pb2.Timestamp
    finished_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., node_id: _Optional[str] = ..., target_revision: _Optional[str] = ..., rollback_revision: _Optional[str] = ..., status: _Optional[_Union[ProvisioningStatus, str]] = ..., resulting_revision: _Optional[str] = ..., exit_code: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., finished_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class NodeVersion(_message.Message):
    __slots__ = ("node_id", "revision", "op_id", "reported_at")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    OP_ID_FIELD_NUMBER: _ClassVar[int]
    REPORTED_AT_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    revision: str
    op_id: str
    reported_at: _timestamp_pb2.Timestamp
    def __init__(self, node_id: _Optional[str] = ..., revision: _Optional[str] = ..., op_id: _Optional[str] = ..., reported_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ProvisionEvent(_message.Message):
    __slots__ = ("op_id", "kind", "sequence", "log_chunk", "status", "revision", "exit_code", "emitted_at")
    OP_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    LOG_CHUNK_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    EMITTED_AT_FIELD_NUMBER: _ClassVar[int]
    op_id: str
    kind: ProvisionEventKind
    sequence: int
    log_chunk: str
    status: str
    revision: str
    exit_code: int
    emitted_at: _timestamp_pb2.Timestamp
    def __init__(self, op_id: _Optional[str] = ..., kind: _Optional[_Union[ProvisionEventKind, str]] = ..., sequence: _Optional[int] = ..., log_chunk: _Optional[str] = ..., status: _Optional[str] = ..., revision: _Optional[str] = ..., exit_code: _Optional[int] = ..., emitted_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class SyncToRevisionRequest(_message.Message):
    __slots__ = ("node_id", "target_revision", "rollback_revision", "timeout_seconds")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_REVISION_FIELD_NUMBER: _ClassVar[int]
    ROLLBACK_REVISION_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    target_revision: str
    rollback_revision: str
    timeout_seconds: int
    def __init__(self, node_id: _Optional[str] = ..., target_revision: _Optional[str] = ..., rollback_revision: _Optional[str] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class SyncToRevisionResponse(_message.Message):
    __slots__ = ("op_id", "dry_run", "node_id", "target_revision", "rollback_revision")
    OP_ID_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_REVISION_FIELD_NUMBER: _ClassVar[int]
    ROLLBACK_REVISION_FIELD_NUMBER: _ClassVar[int]
    op_id: str
    dry_run: bool
    node_id: str
    target_revision: str
    rollback_revision: str
    def __init__(self, op_id: _Optional[str] = ..., dry_run: _Optional[bool] = ..., node_id: _Optional[str] = ..., target_revision: _Optional[str] = ..., rollback_revision: _Optional[str] = ...) -> None: ...

class GetProvisioningOpRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetProvisioningOpResponse(_message.Message):
    __slots__ = ("op", "events")
    OP_FIELD_NUMBER: _ClassVar[int]
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    op: ProvisioningOp
    events: _containers.RepeatedCompositeFieldContainer[ProvisionEvent]
    def __init__(self, op: _Optional[_Union[ProvisioningOp, _Mapping]] = ..., events: _Optional[_Iterable[_Union[ProvisionEvent, _Mapping]]] = ...) -> None: ...

class ListProvisioningOpsRequest(_message.Message):
    __slots__ = ("node_id", "limit")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    limit: int
    def __init__(self, node_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListProvisioningOpsResponse(_message.Message):
    __slots__ = ("ops",)
    OPS_FIELD_NUMBER: _ClassVar[int]
    ops: _containers.RepeatedCompositeFieldContainer[ProvisioningOp]
    def __init__(self, ops: _Optional[_Iterable[_Union[ProvisioningOp, _Mapping]]] = ...) -> None: ...

class WaitProvisioningOpRequest(_message.Message):
    __slots__ = ("id", "timeout_seconds")
    ID_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    id: str
    timeout_seconds: int
    def __init__(self, id: _Optional[str] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class WaitProvisioningOpResponse(_message.Message):
    __slots__ = ("op", "timed_out")
    OP_FIELD_NUMBER: _ClassVar[int]
    TIMED_OUT_FIELD_NUMBER: _ClassVar[int]
    op: ProvisioningOp
    timed_out: bool
    def __init__(self, op: _Optional[_Union[ProvisioningOp, _Mapping]] = ..., timed_out: _Optional[bool] = ...) -> None: ...

class GetNodeVersionRequest(_message.Message):
    __slots__ = ("node_id",)
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    def __init__(self, node_id: _Optional[str] = ...) -> None: ...

class GetNodeVersionResponse(_message.Message):
    __slots__ = ("version", "has_version")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    HAS_VERSION_FIELD_NUMBER: _ClassVar[int]
    version: NodeVersion
    has_version: bool
    def __init__(self, version: _Optional[_Union[NodeVersion, _Mapping]] = ..., has_version: _Optional[bool] = ...) -> None: ...

class ReportProvisionEventRequest(_message.Message):
    __slots__ = ("event",)
    EVENT_FIELD_NUMBER: _ClassVar[int]
    event: ProvisionEvent
    def __init__(self, event: _Optional[_Union[ProvisionEvent, _Mapping]] = ...) -> None: ...

class ReportProvisionEventResponse(_message.Message):
    __slots__ = ("accepted",)
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    accepted: bool
    def __init__(self, accepted: _Optional[bool] = ...) -> None: ...
