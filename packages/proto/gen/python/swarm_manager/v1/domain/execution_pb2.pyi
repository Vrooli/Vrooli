from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ExecutionRecord(_message.Message):
    __slots__ = ("execution_id", "backlog_kind", "backlog_name", "task_id", "run_id", "status", "mode", "started_at", "finished_at", "failure_reason", "started_by", "operation", "created_at", "updated_at", "archive_context", "parent_execution_id", "fixup_attempt", "finalization")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    BACKLOG_KIND_FIELD_NUMBER: _ClassVar[int]
    BACKLOG_NAME_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
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
    FINALIZATION_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    backlog_kind: str
    backlog_name: str
    task_id: str
    run_id: str
    status: str
    mode: str
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
    finalization: Finalization
    def __init__(self, execution_id: _Optional[str] = ..., backlog_kind: _Optional[str] = ..., backlog_name: _Optional[str] = ..., task_id: _Optional[str] = ..., run_id: _Optional[str] = ..., status: _Optional[str] = ..., mode: _Optional[str] = ..., started_at: _Optional[str] = ..., finished_at: _Optional[str] = ..., failure_reason: _Optional[str] = ..., started_by: _Optional[str] = ..., operation: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., archive_context: _Optional[_Union[ArchiveContext, _Mapping]] = ..., parent_execution_id: _Optional[str] = ..., fixup_attempt: _Optional[int] = ..., finalization: _Optional[_Union[Finalization, _Mapping]] = ...) -> None: ...

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

class Finalization(_message.Message):
    __slots__ = ("eligible", "status", "phase", "scope_source", "skip_reason", "started_at", "completed_at", "warnings", "affected_scenarios", "aggregate_classification", "aggregate_summary", "scenarios")
    ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    SCOPE_SOURCE_FIELD_NUMBER: _ClassVar[int]
    SKIP_REASON_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    AFFECTED_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    AGGREGATE_CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    AGGREGATE_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    eligible: bool
    status: str
    phase: str
    scope_source: str
    skip_reason: str
    started_at: str
    completed_at: str
    warnings: _containers.RepeatedCompositeFieldContainer[FinalizationWarning]
    affected_scenarios: _containers.RepeatedScalarFieldContainer[str]
    aggregate_classification: str
    aggregate_summary: str
    scenarios: _containers.RepeatedCompositeFieldContainer[ScenarioFinalization]
    def __init__(self, eligible: _Optional[bool] = ..., status: _Optional[str] = ..., phase: _Optional[str] = ..., scope_source: _Optional[str] = ..., skip_reason: _Optional[str] = ..., started_at: _Optional[str] = ..., completed_at: _Optional[str] = ..., warnings: _Optional[_Iterable[_Union[FinalizationWarning, _Mapping]]] = ..., affected_scenarios: _Optional[_Iterable[str]] = ..., aggregate_classification: _Optional[str] = ..., aggregate_summary: _Optional[str] = ..., scenarios: _Optional[_Iterable[_Union[ScenarioFinalization, _Mapping]]] = ...) -> None: ...

class FinalizationWarning(_message.Message):
    __slots__ = ("code", "scenario_name", "message", "retryable", "created_at")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    RETRYABLE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    code: str
    scenario_name: str
    message: str
    retryable: bool
    created_at: str
    def __init__(self, code: _Optional[str] = ..., scenario_name: _Optional[str] = ..., message: _Optional[str] = ..., retryable: _Optional[bool] = ..., created_at: _Optional[str] = ...) -> None: ...

class ScenarioFinalization(_message.Message):
    __slots__ = ("scenario_name", "changed_paths", "restart", "health", "review")
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    CHANGED_PATHS_FIELD_NUMBER: _ClassVar[int]
    RESTART_FIELD_NUMBER: _ClassVar[int]
    HEALTH_FIELD_NUMBER: _ClassVar[int]
    REVIEW_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    changed_paths: _containers.RepeatedScalarFieldContainer[str]
    restart: RestartResult
    health: HealthCheckResult
    review: ScenarioReview
    def __init__(self, scenario_name: _Optional[str] = ..., changed_paths: _Optional[_Iterable[str]] = ..., restart: _Optional[_Union[RestartResult, _Mapping]] = ..., health: _Optional[_Union[HealthCheckResult, _Mapping]] = ..., review: _Optional[_Union[ScenarioReview, _Mapping]] = ...) -> None: ...

class RestartResult(_message.Message):
    __slots__ = ("status", "attempts", "last_error", "started_at", "finished_at")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    LAST_ERROR_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    status: str
    attempts: int
    last_error: str
    started_at: str
    finished_at: str
    def __init__(self, status: _Optional[str] = ..., attempts: _Optional[int] = ..., last_error: _Optional[str] = ..., started_at: _Optional[str] = ..., finished_at: _Optional[str] = ...) -> None: ...

class HealthCheckResult(_message.Message):
    __slots__ = ("status", "scenario_status", "health_status", "schema_valid", "details", "checked_at")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_STATUS_FIELD_NUMBER: _ClassVar[int]
    HEALTH_STATUS_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VALID_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    status: str
    scenario_status: str
    health_status: str
    schema_valid: bool
    details: str
    checked_at: str
    def __init__(self, status: _Optional[str] = ..., scenario_status: _Optional[str] = ..., health_status: _Optional[str] = ..., schema_valid: _Optional[bool] = ..., details: _Optional[str] = ..., checked_at: _Optional[str] = ...) -> None: ...

class ScenarioReview(_message.Message):
    __slots__ = ("status", "job_id", "skip_reason", "result")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    JOB_ID_FIELD_NUMBER: _ClassVar[int]
    SKIP_REASON_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    status: str
    job_id: str
    skip_reason: str
    result: ReviewResult
    def __init__(self, status: _Optional[str] = ..., job_id: _Optional[str] = ..., skip_reason: _Optional[str] = ..., result: _Optional[_Union[ReviewResult, _Mapping]] = ...) -> None: ...

class ExecutionPolicy(_message.Message):
    __slots__ = ("default_mode", "auto_fixup", "max_fixup_attempts")
    DEFAULT_MODE_FIELD_NUMBER: _ClassVar[int]
    AUTO_FIXUP_FIELD_NUMBER: _ClassVar[int]
    MAX_FIXUP_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    default_mode: str
    auto_fixup: bool
    max_fixup_attempts: int
    def __init__(self, default_mode: _Optional[str] = ..., auto_fixup: _Optional[bool] = ..., max_fixup_attempts: _Optional[int] = ...) -> None: ...
