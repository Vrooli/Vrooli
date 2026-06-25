import datetime

from google.api import annotations_pb2 as _annotations_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ScriptExecutionStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SCRIPT_EXECUTION_STATUS_UNSPECIFIED: _ClassVar[ScriptExecutionStatus]
    SCRIPT_EXECUTION_STATUS_RUNNING: _ClassVar[ScriptExecutionStatus]
    SCRIPT_EXECUTION_STATUS_COMPLETED: _ClassVar[ScriptExecutionStatus]
    SCRIPT_EXECUTION_STATUS_FAILED: _ClassVar[ScriptExecutionStatus]
SCRIPT_EXECUTION_STATUS_UNSPECIFIED: ScriptExecutionStatus
SCRIPT_EXECUTION_STATUS_RUNNING: ScriptExecutionStatus
SCRIPT_EXECUTION_STATUS_COMPLETED: ScriptExecutionStatus
SCRIPT_EXECUTION_STATUS_FAILED: ScriptExecutionStatus

class InvestigationScript(_message.Message):
    __slots__ = ("id", "name", "description", "category", "created_at", "updated_at", "author", "enabled")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    AUTHOR_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    category: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    author: str
    enabled: bool
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., category: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., author: _Optional[str] = ..., enabled: _Optional[bool] = ...) -> None: ...

class ScriptExecution(_message.Message):
    __slots__ = ("script_id", "execution_id", "status", "started_at", "completed_at", "output", "error", "exit_code", "stdout", "stderr", "timed_out", "duration_seconds")
    SCRIPT_ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    STDOUT_FIELD_NUMBER: _ClassVar[int]
    STDERR_FIELD_NUMBER: _ClassVar[int]
    TIMED_OUT_FIELD_NUMBER: _ClassVar[int]
    DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    script_id: str
    execution_id: str
    status: ScriptExecutionStatus
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    output: str
    error: str
    exit_code: int
    stdout: str
    stderr: str
    timed_out: bool
    duration_seconds: float
    def __init__(self, script_id: _Optional[str] = ..., execution_id: _Optional[str] = ..., status: _Optional[_Union[ScriptExecutionStatus, str]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., output: _Optional[str] = ..., error: _Optional[str] = ..., exit_code: _Optional[int] = ..., stdout: _Optional[str] = ..., stderr: _Optional[str] = ..., timed_out: _Optional[bool] = ..., duration_seconds: _Optional[float] = ...) -> None: ...

class ListScriptsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListScriptsResponse(_message.Message):
    __slots__ = ("scripts",)
    SCRIPTS_FIELD_NUMBER: _ClassVar[int]
    scripts: _containers.RepeatedCompositeFieldContainer[InvestigationScript]
    def __init__(self, scripts: _Optional[_Iterable[_Union[InvestigationScript, _Mapping]]] = ...) -> None: ...

class GetScriptRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetScriptResponse(_message.Message):
    __slots__ = ("script", "content")
    SCRIPT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    script: InvestigationScript
    content: str
    def __init__(self, script: _Optional[_Union[InvestigationScript, _Mapping]] = ..., content: _Optional[str] = ...) -> None: ...

class ExecuteScriptRequest(_message.Message):
    __slots__ = ("id", "content")
    ID_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    id: str
    content: str
    def __init__(self, id: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class ExecuteScriptResponse(_message.Message):
    __slots__ = ("execution",)
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    execution: ScriptExecution
    def __init__(self, execution: _Optional[_Union[ScriptExecution, _Mapping]] = ...) -> None: ...
