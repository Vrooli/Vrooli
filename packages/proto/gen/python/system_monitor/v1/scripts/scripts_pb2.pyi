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
    SCRIPT_EXECUTION_STATUS_SKIPPED: _ClassVar[ScriptExecutionStatus]
SCRIPT_EXECUTION_STATUS_UNSPECIFIED: ScriptExecutionStatus
SCRIPT_EXECUTION_STATUS_RUNNING: ScriptExecutionStatus
SCRIPT_EXECUTION_STATUS_COMPLETED: ScriptExecutionStatus
SCRIPT_EXECUTION_STATUS_FAILED: ScriptExecutionStatus
SCRIPT_EXECUTION_STATUS_SKIPPED: ScriptExecutionStatus

class InvestigationScript(_message.Message):
    __slots__ = ("id", "name", "description", "category", "created_at", "updated_at", "author", "enabled", "execution_mode", "required_tools", "skip_reason", "platforms", "source")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    AUTHOR_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_MODE_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_TOOLS_FIELD_NUMBER: _ClassVar[int]
    SKIP_REASON_FIELD_NUMBER: _ClassVar[int]
    PLATFORMS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    category: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    author: str
    enabled: bool
    execution_mode: str
    required_tools: _containers.RepeatedScalarFieldContainer[str]
    skip_reason: str
    platforms: _containers.RepeatedScalarFieldContainer[str]
    source: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., category: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., author: _Optional[str] = ..., enabled: _Optional[bool] = ..., execution_mode: _Optional[str] = ..., required_tools: _Optional[_Iterable[str]] = ..., skip_reason: _Optional[str] = ..., platforms: _Optional[_Iterable[str]] = ..., source: _Optional[str] = ...) -> None: ...

class ScriptExecution(_message.Message):
    __slots__ = ("script_id", "execution_id", "status", "started_at", "completed_at", "output", "error", "exit_code", "stdout", "stderr", "timed_out", "duration_seconds", "execution_mode", "skip_reason")
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
    EXECUTION_MODE_FIELD_NUMBER: _ClassVar[int]
    SKIP_REASON_FIELD_NUMBER: _ClassVar[int]
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
    execution_mode: str
    skip_reason: str
    def __init__(self, script_id: _Optional[str] = ..., execution_id: _Optional[str] = ..., status: _Optional[_Union[ScriptExecutionStatus, str]] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., output: _Optional[str] = ..., error: _Optional[str] = ..., exit_code: _Optional[int] = ..., stdout: _Optional[str] = ..., stderr: _Optional[str] = ..., timed_out: _Optional[bool] = ..., duration_seconds: _Optional[float] = ..., execution_mode: _Optional[str] = ..., skip_reason: _Optional[str] = ...) -> None: ...

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

class UpdateScriptRequest(_message.Message):
    __slots__ = ("id", "content")
    ID_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    id: str
    content: str
    def __init__(self, id: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

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

class InvestigationFinding(_message.Message):
    __slots__ = ("severity", "code", "summary", "detail_json")
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    DETAIL_JSON_FIELD_NUMBER: _ClassVar[int]
    severity: str
    code: str
    summary: str
    detail_json: str
    def __init__(self, severity: _Optional[str] = ..., code: _Optional[str] = ..., summary: _Optional[str] = ..., detail_json: _Optional[str] = ...) -> None: ...

class InvestigationRun(_message.Message):
    __slots__ = ("id", "entry_id", "execution_mode", "status", "skip_reason", "exit_code", "timed_out", "started_at", "completed_at", "duration_seconds", "host_os", "host_arch", "result_json", "stderr_tail", "anomaly_id", "findings")
    ID_FIELD_NUMBER: _ClassVar[int]
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_MODE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SKIP_REASON_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    TIMED_OUT_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    HOST_OS_FIELD_NUMBER: _ClassVar[int]
    HOST_ARCH_FIELD_NUMBER: _ClassVar[int]
    RESULT_JSON_FIELD_NUMBER: _ClassVar[int]
    STDERR_TAIL_FIELD_NUMBER: _ClassVar[int]
    ANOMALY_ID_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    id: str
    entry_id: str
    execution_mode: str
    status: str
    skip_reason: str
    exit_code: int
    timed_out: bool
    started_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    duration_seconds: float
    host_os: str
    host_arch: str
    result_json: str
    stderr_tail: str
    anomaly_id: str
    findings: _containers.RepeatedCompositeFieldContainer[InvestigationFinding]
    def __init__(self, id: _Optional[str] = ..., entry_id: _Optional[str] = ..., execution_mode: _Optional[str] = ..., status: _Optional[str] = ..., skip_reason: _Optional[str] = ..., exit_code: _Optional[int] = ..., timed_out: _Optional[bool] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., duration_seconds: _Optional[float] = ..., host_os: _Optional[str] = ..., host_arch: _Optional[str] = ..., result_json: _Optional[str] = ..., stderr_tail: _Optional[str] = ..., anomaly_id: _Optional[str] = ..., findings: _Optional[_Iterable[_Union[InvestigationFinding, _Mapping]]] = ...) -> None: ...

class ListRunsRequest(_message.Message):
    __slots__ = ("entry_id", "since", "limit")
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    SINCE_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    entry_id: str
    since: str
    limit: int
    def __init__(self, entry_id: _Optional[str] = ..., since: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListRunsResponse(_message.Message):
    __slots__ = ("runs",)
    RUNS_FIELD_NUMBER: _ClassVar[int]
    runs: _containers.RepeatedCompositeFieldContainer[InvestigationRun]
    def __init__(self, runs: _Optional[_Iterable[_Union[InvestigationRun, _Mapping]]] = ...) -> None: ...

class GetRunRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetRunResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: InvestigationRun
    def __init__(self, run: _Optional[_Union[InvestigationRun, _Mapping]] = ...) -> None: ...

class PruneRunsRequest(_message.Message):
    __slots__ = ("dry_run",)
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    def __init__(self, dry_run: _Optional[bool] = ...) -> None: ...

class PruneRunsResponse(_message.Message):
    __slots__ = ("deleted", "dry_run")
    DELETED_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    deleted: int
    dry_run: bool
    def __init__(self, deleted: _Optional[int] = ..., dry_run: _Optional[bool] = ...) -> None: ...
