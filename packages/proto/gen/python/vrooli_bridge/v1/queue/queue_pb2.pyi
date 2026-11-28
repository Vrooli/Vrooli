import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class QueueState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    QUEUE_STATE_UNSPECIFIED: _ClassVar[QueueState]
    QUEUE_STATE_QUEUED: _ClassVar[QueueState]
    QUEUE_STATE_RUNNING: _ClassVar[QueueState]
QUEUE_STATE_UNSPECIFIED: QueueState
QUEUE_STATE_QUEUED: QueueState
QUEUE_STATE_RUNNING: QueueState

class QueueEntry(_message.Message):
    __slots__ = ("run_id", "node_id", "scenario", "verb", "args", "state", "position", "enqueued_at", "started_at")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    VERB_FIELD_NUMBER: _ClassVar[int]
    ARGS_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    POSITION_FIELD_NUMBER: _ClassVar[int]
    ENQUEUED_AT_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    node_id: str
    scenario: str
    verb: str
    args: _containers.RepeatedScalarFieldContainer[str]
    state: QueueState
    position: int
    enqueued_at: _timestamp_pb2.Timestamp
    started_at: _timestamp_pb2.Timestamp
    def __init__(self, run_id: _Optional[str] = ..., node_id: _Optional[str] = ..., scenario: _Optional[str] = ..., verb: _Optional[str] = ..., args: _Optional[_Iterable[str]] = ..., state: _Optional[_Union[QueueState, str]] = ..., position: _Optional[int] = ..., enqueued_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class NodeQueue(_message.Message):
    __slots__ = ("node_id", "concurrency_limit", "running", "queued", "entries")
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    CONCURRENCY_LIMIT_FIELD_NUMBER: _ClassVar[int]
    RUNNING_FIELD_NUMBER: _ClassVar[int]
    QUEUED_FIELD_NUMBER: _ClassVar[int]
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    concurrency_limit: int
    running: int
    queued: int
    entries: _containers.RepeatedCompositeFieldContainer[QueueEntry]
    def __init__(self, node_id: _Optional[str] = ..., concurrency_limit: _Optional[int] = ..., running: _Optional[int] = ..., queued: _Optional[int] = ..., entries: _Optional[_Iterable[_Union[QueueEntry, _Mapping]]] = ...) -> None: ...

class ListQueueRequest(_message.Message):
    __slots__ = ("node_id",)
    NODE_ID_FIELD_NUMBER: _ClassVar[int]
    node_id: str
    def __init__(self, node_id: _Optional[str] = ...) -> None: ...

class ListQueueResponse(_message.Message):
    __slots__ = ("nodes",)
    NODES_FIELD_NUMBER: _ClassVar[int]
    nodes: _containers.RepeatedCompositeFieldContainer[NodeQueue]
    def __init__(self, nodes: _Optional[_Iterable[_Union[NodeQueue, _Mapping]]] = ...) -> None: ...
