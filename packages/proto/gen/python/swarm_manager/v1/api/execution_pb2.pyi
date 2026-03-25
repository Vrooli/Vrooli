from buf.validate import validate_pb2 as _validate_pb2
from swarm_manager.v1.domain import execution_pb2 as _execution_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListExecutionResponse(_message.Message):
    __slots__ = ("items",)
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[_execution_pb2.ExecutionRecord]
    def __init__(self, items: _Optional[_Iterable[_Union[_execution_pb2.ExecutionRecord, _Mapping]]] = ...) -> None: ...

class ExecutionResponse(_message.Message):
    __slots__ = ("execution",)
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    execution: _execution_pb2.ExecutionRecord
    def __init__(self, execution: _Optional[_Union[_execution_pb2.ExecutionRecord, _Mapping]] = ...) -> None: ...

class ExecutionPolicyResponse(_message.Message):
    __slots__ = ("policy",)
    POLICY_FIELD_NUMBER: _ClassVar[int]
    policy: _execution_pb2.ExecutionPolicy
    def __init__(self, policy: _Optional[_Union[_execution_pb2.ExecutionPolicy, _Mapping]] = ...) -> None: ...

class CreateExecutionRequest(_message.Message):
    __slots__ = ("backlog_kind", "backlog_name", "mode", "delay_seconds", "started_by", "operation")
    BACKLOG_KIND_FIELD_NUMBER: _ClassVar[int]
    BACKLOG_NAME_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    DELAY_SECONDS_FIELD_NUMBER: _ClassVar[int]
    STARTED_BY_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    backlog_kind: str
    backlog_name: str
    mode: str
    delay_seconds: int
    started_by: str
    operation: str
    def __init__(self, backlog_kind: _Optional[str] = ..., backlog_name: _Optional[str] = ..., mode: _Optional[str] = ..., delay_seconds: _Optional[int] = ..., started_by: _Optional[str] = ..., operation: _Optional[str] = ...) -> None: ...

class FollowUpExecutionRequest(_message.Message):
    __slots__ = ("execution_id", "follow_up_type", "context", "run_mode")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    FOLLOW_UP_TYPE_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    RUN_MODE_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    follow_up_type: str
    context: str
    run_mode: str
    def __init__(self, execution_id: _Optional[str] = ..., follow_up_type: _Optional[str] = ..., context: _Optional[str] = ..., run_mode: _Optional[str] = ...) -> None: ...
