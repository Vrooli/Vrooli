from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class ExecutionRecord(_message.Message):
    __slots__ = ("execution_id", "backlog_kind", "backlog_name", "task_id", "run_id", "status", "mode", "scheduled_at", "started_at", "finished_at", "failure_reason", "started_by", "operation", "created_at", "updated_at")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    BACKLOG_KIND_FIELD_NUMBER: _ClassVar[int]
    BACKLOG_NAME_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    SCHEDULED_AT_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    FAILURE_REASON_FIELD_NUMBER: _ClassVar[int]
    STARTED_BY_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    backlog_kind: str
    backlog_name: str
    task_id: str
    run_id: str
    status: str
    mode: str
    scheduled_at: str
    started_at: str
    finished_at: str
    failure_reason: str
    started_by: str
    operation: str
    created_at: str
    updated_at: str
    def __init__(self, execution_id: _Optional[str] = ..., backlog_kind: _Optional[str] = ..., backlog_name: _Optional[str] = ..., task_id: _Optional[str] = ..., run_id: _Optional[str] = ..., status: _Optional[str] = ..., mode: _Optional[str] = ..., scheduled_at: _Optional[str] = ..., started_at: _Optional[str] = ..., finished_at: _Optional[str] = ..., failure_reason: _Optional[str] = ..., started_by: _Optional[str] = ..., operation: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class ExecutionPolicy(_message.Message):
    __slots__ = ("default_mode", "default_delay_seconds")
    DEFAULT_MODE_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_DELAY_SECONDS_FIELD_NUMBER: _ClassVar[int]
    default_mode: str
    default_delay_seconds: int
    def __init__(self, default_mode: _Optional[str] = ..., default_delay_seconds: _Optional[int] = ...) -> None: ...
