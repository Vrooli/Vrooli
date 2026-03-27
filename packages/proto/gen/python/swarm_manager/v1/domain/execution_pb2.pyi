from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ExecutionRecord(_message.Message):
    __slots__ = ("execution_id", "backlog_kind", "backlog_name", "task_id", "run_id", "status", "mode", "scheduled_at", "started_at", "finished_at", "failure_reason", "started_by", "operation", "created_at", "updated_at", "archive_context", "parent_execution_id", "fixup_attempt", "review_result", "review_job_id", "review_skip_reason", "review_started_at")
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
    PARENT_EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    FIXUP_ATTEMPT_FIELD_NUMBER: _ClassVar[int]
    REVIEW_RESULT_FIELD_NUMBER: _ClassVar[int]
    REVIEW_JOB_ID_FIELD_NUMBER: _ClassVar[int]
    REVIEW_SKIP_REASON_FIELD_NUMBER: _ClassVar[int]
    REVIEW_STARTED_AT_FIELD_NUMBER: _ClassVar[int]
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
    parent_execution_id: str
    fixup_attempt: int
    review_result: ReviewResult
    review_job_id: str
    review_skip_reason: str
    review_started_at: str
    def __init__(self, execution_id: _Optional[str] = ..., backlog_kind: _Optional[str] = ..., backlog_name: _Optional[str] = ..., task_id: _Optional[str] = ..., run_id: _Optional[str] = ..., status: _Optional[str] = ..., mode: _Optional[str] = ..., scheduled_at: _Optional[str] = ..., started_at: _Optional[str] = ..., finished_at: _Optional[str] = ..., failure_reason: _Optional[str] = ..., started_by: _Optional[str] = ..., operation: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., archive_context: _Optional[_Union[ArchiveContext, _Mapping]] = ..., parent_execution_id: _Optional[str] = ..., fixup_attempt: _Optional[int] = ..., review_result: _Optional[_Union[ReviewResult, _Mapping]] = ..., review_job_id: _Optional[str] = ..., review_skip_reason: _Optional[str] = ..., review_started_at: _Optional[str] = ...) -> None: ...

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

class ReviewResult(_message.Message):
    __slots__ = ("job_id", "classification", "dimensions", "summary", "reviewed_at")
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    DIMENSIONS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    REVIEWED_AT_FIELD_NUMBER: _ClassVar[int]
    job_id: str
    classification: str
    dimensions: _containers.RepeatedCompositeFieldContainer[ReviewDimension]
    summary: str
    reviewed_at: str
    def __init__(self, job_id: _Optional[str] = ..., classification: _Optional[str] = ..., dimensions: _Optional[_Iterable[_Union[ReviewDimension, _Mapping]]] = ..., summary: _Optional[str] = ..., reviewed_at: _Optional[str] = ...) -> None: ...

class ReviewDimension(_message.Message):
    __slots__ = ("name", "status", "details")
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    name: str
    status: str
    details: str
    def __init__(self, name: _Optional[str] = ..., status: _Optional[str] = ..., details: _Optional[str] = ...) -> None: ...

class ExecutionPolicy(_message.Message):
    __slots__ = ("default_mode", "default_delay_seconds", "auto_fixup", "max_fixup_attempts")
    DEFAULT_MODE_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_DELAY_SECONDS_FIELD_NUMBER: _ClassVar[int]
    AUTO_FIXUP_FIELD_NUMBER: _ClassVar[int]
    MAX_FIXUP_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    default_mode: str
    default_delay_seconds: int
    auto_fixup: bool
    max_fixup_attempts: int
    def __init__(self, default_mode: _Optional[str] = ..., default_delay_seconds: _Optional[int] = ..., auto_fixup: _Optional[bool] = ..., max_fixup_attempts: _Optional[int] = ...) -> None: ...
