from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ExecutionRecord(_message.Message):
    __slots__ = ("execution_id", "backlog_kind", "backlog_name", "task_id", "run_id", "status", "mode", "scheduled_at", "started_at", "finished_at", "failure_reason", "started_by", "operation", "created_at", "updated_at", "archive_context")
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
    ARCHIVE_CONTEXT_FIELD_NUMBER: _ClassVar[int]
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
    archive_context: ArchiveContext
    def __init__(self, execution_id: _Optional[str] = ..., backlog_kind: _Optional[str] = ..., backlog_name: _Optional[str] = ..., task_id: _Optional[str] = ..., run_id: _Optional[str] = ..., status: _Optional[str] = ..., mode: _Optional[str] = ..., scheduled_at: _Optional[str] = ..., started_at: _Optional[str] = ..., finished_at: _Optional[str] = ..., failure_reason: _Optional[str] = ..., started_by: _Optional[str] = ..., operation: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., archive_context: _Optional[_Union[ArchiveContext, _Mapping]] = ...) -> None: ...

class ArchiveContext(_message.Message):
    __slots__ = ("scenario_name", "scenario_path", "preset_or_custom", "preserve_paths", "preserve_preset")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_PATH_FIELD_NUMBER: _ClassVar[int]
    PRESET_OR_CUSTOM_FIELD_NUMBER: _ClassVar[int]
    PRESERVE_PATHS_FIELD_NUMBER: _ClassVar[int]
    PRESERVE_PRESET_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    scenario_path: str
    preset_or_custom: str
    preserve_paths: _containers.RepeatedScalarFieldContainer[str]
    preserve_preset: str
    def __init__(self, scenario_name: _Optional[str] = ..., scenario_path: _Optional[str] = ..., preset_or_custom: _Optional[str] = ..., preserve_paths: _Optional[_Iterable[str]] = ..., preserve_preset: _Optional[str] = ...) -> None: ...

class ExecutionPolicy(_message.Message):
    __slots__ = ("default_mode", "default_delay_seconds")
    DEFAULT_MODE_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_DELAY_SECONDS_FIELD_NUMBER: _ClassVar[int]
    default_mode: str
    default_delay_seconds: int
    def __init__(self, default_mode: _Optional[str] = ..., default_delay_seconds: _Optional[int] = ...) -> None: ...
