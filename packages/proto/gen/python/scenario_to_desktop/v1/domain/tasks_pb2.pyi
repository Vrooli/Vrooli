import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TaskType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TASK_TYPE_UNSPECIFIED: _ClassVar[TaskType]
    TASK_TYPE_INVESTIGATE: _ClassVar[TaskType]
    TASK_TYPE_FIX: _ClassVar[TaskType]

class InvestigationEffort(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    INVESTIGATION_EFFORT_UNSPECIFIED: _ClassVar[InvestigationEffort]
    INVESTIGATION_EFFORT_CHECKS: _ClassVar[InvestigationEffort]
    INVESTIGATION_EFFORT_LOGS: _ClassVar[InvestigationEffort]
    INVESTIGATION_EFFORT_TRACE: _ClassVar[InvestigationEffort]

class InvestigationStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    INVESTIGATION_STATUS_UNSPECIFIED: _ClassVar[InvestigationStatus]
    INVESTIGATION_STATUS_PENDING: _ClassVar[InvestigationStatus]
    INVESTIGATION_STATUS_RUNNING: _ClassVar[InvestigationStatus]
    INVESTIGATION_STATUS_COMPLETED: _ClassVar[InvestigationStatus]
    INVESTIGATION_STATUS_FAILED: _ClassVar[InvestigationStatus]
    INVESTIGATION_STATUS_CANCELLED: _ClassVar[InvestigationStatus]
TASK_TYPE_UNSPECIFIED: TaskType
TASK_TYPE_INVESTIGATE: TaskType
TASK_TYPE_FIX: TaskType
INVESTIGATION_EFFORT_UNSPECIFIED: InvestigationEffort
INVESTIGATION_EFFORT_CHECKS: InvestigationEffort
INVESTIGATION_EFFORT_LOGS: InvestigationEffort
INVESTIGATION_EFFORT_TRACE: InvestigationEffort
INVESTIGATION_STATUS_UNSPECIFIED: InvestigationStatus
INVESTIGATION_STATUS_PENDING: InvestigationStatus
INVESTIGATION_STATUS_RUNNING: InvestigationStatus
INVESTIGATION_STATUS_COMPLETED: InvestigationStatus
INVESTIGATION_STATUS_FAILED: InvestigationStatus
INVESTIGATION_STATUS_CANCELLED: InvestigationStatus

class TaskFocus(_message.Message):
    __slots__ = ("harness", "subject")
    HARNESS_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    harness: bool
    subject: bool
    def __init__(self, harness: _Optional[bool] = ..., subject: _Optional[bool] = ...) -> None: ...

class FixPermissions(_message.Message):
    __slots__ = ("immediate", "permanent", "prevention")
    IMMEDIATE_FIELD_NUMBER: _ClassVar[int]
    PERMANENT_FIELD_NUMBER: _ClassVar[int]
    PREVENTION_FIELD_NUMBER: _ClassVar[int]
    immediate: bool
    permanent: bool
    prevention: bool
    def __init__(self, immediate: _Optional[bool] = ..., permanent: _Optional[bool] = ..., prevention: _Optional[bool] = ...) -> None: ...

class CreateTaskRequest(_message.Message):
    __slots__ = ("pipeline_id", "task_type", "focus", "note", "effort", "permissions", "source_investigation_id", "max_iterations", "include_contexts")
    PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_TYPE_FIELD_NUMBER: _ClassVar[int]
    FOCUS_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    EFFORT_FIELD_NUMBER: _ClassVar[int]
    PERMISSIONS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_INVESTIGATION_ID_FIELD_NUMBER: _ClassVar[int]
    MAX_ITERATIONS_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_CONTEXTS_FIELD_NUMBER: _ClassVar[int]
    pipeline_id: str
    task_type: TaskType
    focus: TaskFocus
    note: str
    effort: InvestigationEffort
    permissions: FixPermissions
    source_investigation_id: str
    max_iterations: int
    include_contexts: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, pipeline_id: _Optional[str] = ..., task_type: _Optional[_Union[TaskType, str]] = ..., focus: _Optional[_Union[TaskFocus, _Mapping]] = ..., note: _Optional[str] = ..., effort: _Optional[_Union[InvestigationEffort, str]] = ..., permissions: _Optional[_Union[FixPermissions, _Mapping]] = ..., source_investigation_id: _Optional[str] = ..., max_iterations: _Optional[int] = ..., include_contexts: _Optional[_Iterable[str]] = ...) -> None: ...

class Investigation(_message.Message):
    __slots__ = ("id", "pipeline_id", "status", "findings", "progress", "details", "agent_run_id", "error_message", "created_at", "updated_at", "completed_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    AGENT_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    pipeline_id: str
    status: InvestigationStatus
    findings: str
    progress: int
    details: _struct_pb2.Struct
    agent_run_id: str
    error_message: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., pipeline_id: _Optional[str] = ..., status: _Optional[_Union[InvestigationStatus, str]] = ..., findings: _Optional[str] = ..., progress: _Optional[int] = ..., details: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., agent_run_id: _Optional[str] = ..., error_message: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class InvestigationSummary(_message.Message):
    __slots__ = ("id", "pipeline_id", "status", "progress", "has_findings", "error_message", "source_investigation_id", "created_at", "completed_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    HAS_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_INVESTIGATION_ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    pipeline_id: str
    status: InvestigationStatus
    progress: int
    has_findings: bool
    error_message: str
    source_investigation_id: str
    created_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., pipeline_id: _Optional[str] = ..., status: _Optional[_Union[InvestigationStatus, str]] = ..., progress: _Optional[int] = ..., has_findings: _Optional[bool] = ..., error_message: _Optional[str] = ..., source_investigation_id: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CreateTaskResponse(_message.Message):
    __slots__ = ("task",)
    TASK_FIELD_NUMBER: _ClassVar[int]
    task: Investigation
    def __init__(self, task: _Optional[_Union[Investigation, _Mapping]] = ...) -> None: ...

class GetTaskRequest(_message.Message):
    __slots__ = ("pipeline_id", "task_id")
    PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    pipeline_id: str
    task_id: str
    def __init__(self, pipeline_id: _Optional[str] = ..., task_id: _Optional[str] = ...) -> None: ...

class GetTaskResponse(_message.Message):
    __slots__ = ("task",)
    TASK_FIELD_NUMBER: _ClassVar[int]
    task: Investigation
    def __init__(self, task: _Optional[_Union[Investigation, _Mapping]] = ...) -> None: ...

class ListTasksRequest(_message.Message):
    __slots__ = ("pipeline_id", "limit")
    PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    pipeline_id: str
    limit: int
    def __init__(self, pipeline_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListTasksResponse(_message.Message):
    __slots__ = ("tasks",)
    TASKS_FIELD_NUMBER: _ClassVar[int]
    tasks: _containers.RepeatedCompositeFieldContainer[InvestigationSummary]
    def __init__(self, tasks: _Optional[_Iterable[_Union[InvestigationSummary, _Mapping]]] = ...) -> None: ...

class StopTaskRequest(_message.Message):
    __slots__ = ("pipeline_id", "task_id")
    PIPELINE_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    pipeline_id: str
    task_id: str
    def __init__(self, pipeline_id: _Optional[str] = ..., task_id: _Optional[str] = ...) -> None: ...

class StopTaskResponse(_message.Message):
    __slots__ = ("success", "message")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    success: bool
    message: str
    def __init__(self, success: _Optional[bool] = ..., message: _Optional[str] = ...) -> None: ...

class AgentManagerStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class AgentManagerStatusResponse(_message.Message):
    __slots__ = ("available", "url", "reason")
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    available: bool
    url: str
    reason: str
    def __init__(self, available: _Optional[bool] = ..., url: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...
