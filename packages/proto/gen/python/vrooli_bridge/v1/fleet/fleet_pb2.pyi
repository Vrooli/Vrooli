import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RolloutStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ROLLOUT_STATUS_UNSPECIFIED: _ClassVar[RolloutStatus]
    ROLLOUT_STATUS_DISPATCHED: _ClassVar[RolloutStatus]
    ROLLOUT_STATUS_PARTIAL: _ClassVar[RolloutStatus]
    ROLLOUT_STATUS_FAILED: _ClassVar[RolloutStatus]

class NodeRolloutDisposition(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    NODE_ROLLOUT_DISPOSITION_UNSPECIFIED: _ClassVar[NodeRolloutDisposition]
    NODE_ROLLOUT_DISPOSITION_DISPATCHED: _ClassVar[NodeRolloutDisposition]
    NODE_ROLLOUT_DISPOSITION_SKIPPED_OFFLINE: _ClassVar[NodeRolloutDisposition]
    NODE_ROLLOUT_DISPOSITION_SKIPPED_NEEDS_UPDATE: _ClassVar[NodeRolloutDisposition]
    NODE_ROLLOUT_DISPOSITION_SKIPPED_REVOKED: _ClassVar[NodeRolloutDisposition]
    NODE_ROLLOUT_DISPOSITION_FAILED: _ClassVar[NodeRolloutDisposition]
ROLLOUT_STATUS_UNSPECIFIED: RolloutStatus
ROLLOUT_STATUS_DISPATCHED: RolloutStatus
ROLLOUT_STATUS_PARTIAL: RolloutStatus
ROLLOUT_STATUS_FAILED: RolloutStatus
NODE_ROLLOUT_DISPOSITION_UNSPECIFIED: NodeRolloutDisposition
NODE_ROLLOUT_DISPOSITION_DISPATCHED: NodeRolloutDisposition
NODE_ROLLOUT_DISPOSITION_SKIPPED_OFFLINE: NodeRolloutDisposition
NODE_ROLLOUT_DISPOSITION_SKIPPED_NEEDS_UPDATE: NodeRolloutDisposition
NODE_ROLLOUT_DISPOSITION_SKIPPED_REVOKED: NodeRolloutDisposition
NODE_ROLLOUT_DISPOSITION_FAILED: NodeRolloutDisposition

class NodeRolloutResult(_message.Message):
    __slots__ = ("node_id", "disposition", "op_id", "detail")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    OP_ID_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    disposition: NodeRolloutDisposition
    op_id: str
    detail: str
    def __init__(self, node_id: _Optional[str] = ..., disposition: _Optional[_Union[NodeRolloutDisposition, str]] = ..., op_id: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class Rollout(_message.Message):
    __slots__ = ("id", "target_revision", "status", "total_nodes", "dispatched", "skipped", "failed", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_REVISION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_NODES_FIELD_NUMBER: _ClassVar[int]
    DISPATCHED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    target_revision: str
    status: RolloutStatus
    total_nodes: int
    dispatched: int
    skipped: int
    failed: int
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., target_revision: _Optional[str] = ..., status: _Optional[_Union[RolloutStatus, str]] = ..., total_nodes: _Optional[int] = ..., dispatched: _Optional[int] = ..., skipped: _Optional[int] = ..., failed: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RollFleetRequest(_message.Message):
    __slots__ = ("target_revision", "node_ids", "timeout_seconds")
    TARGET_REVISION_FIELD_NUMBER: _ClassVar[int]
    NODE_IDS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    target_revision: str
    node_ids: _containers.RepeatedScalarFieldContainer[str]
    timeout_seconds: int
    def __init__(self, target_revision: _Optional[str] = ..., node_ids: _Optional[_Iterable[str]] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class RollFleetResponse(_message.Message):
    __slots__ = ("rollout_id", "dry_run", "status", "results")
    ROLLOUT_ID_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    rollout_id: str
    dry_run: bool
    status: RolloutStatus
    results: _containers.RepeatedCompositeFieldContainer[NodeRolloutResult]
    def __init__(self, rollout_id: _Optional[str] = ..., dry_run: _Optional[bool] = ..., status: _Optional[_Union[RolloutStatus, str]] = ..., results: _Optional[_Iterable[_Union[NodeRolloutResult, _Mapping]]] = ...) -> None: ...

class GetRolloutRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetRolloutResponse(_message.Message):
    __slots__ = ("rollout", "results")
    ROLLOUT_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    rollout: Rollout
    results: _containers.RepeatedCompositeFieldContainer[NodeRolloutResult]
    def __init__(self, rollout: _Optional[_Union[Rollout, _Mapping]] = ..., results: _Optional[_Iterable[_Union[NodeRolloutResult, _Mapping]]] = ...) -> None: ...

class ListRolloutsRequest(_message.Message):
    __slots__ = ("limit",)
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    def __init__(self, limit: _Optional[int] = ...) -> None: ...

class ListRolloutsResponse(_message.Message):
    __slots__ = ("rollouts",)
    ROLLOUTS_FIELD_NUMBER: _ClassVar[int]
    rollouts: _containers.RepeatedCompositeFieldContainer[Rollout]
    def __init__(self, rollouts: _Optional[_Iterable[_Union[Rollout, _Mapping]]] = ...) -> None: ...
