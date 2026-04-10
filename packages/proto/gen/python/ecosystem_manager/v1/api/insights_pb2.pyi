from buf.validate import validate_pb2 as _validate_pb2
from ecosystem_manager.v1.domain import insights_pb2 as _insights_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class InsightGenerateRequest(_message.Message):
    __slots__ = ("limit", "status_filter", "custom_prompt")
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FILTER_FIELD_NUMBER: _ClassVar[int]
    CUSTOM_PROMPT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    status_filter: str
    custom_prompt: str
    def __init__(self, limit: _Optional[int] = ..., status_filter: _Optional[str] = ..., custom_prompt: _Optional[str] = ...) -> None: ...

class InsightReportResponse(_message.Message):
    __slots__ = ("report", "task_id", "generated_at")
    REPORT_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    report: _insights_pb2.InsightReport
    task_id: str
    generated_at: str
    def __init__(self, report: _Optional[_Union[_insights_pb2.InsightReport, _Mapping]] = ..., task_id: _Optional[str] = ..., generated_at: _Optional[str] = ...) -> None: ...

class InsightReportListResponse(_message.Message):
    __slots__ = ("task_id", "reports", "count")
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    REPORTS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    reports: _containers.RepeatedCompositeFieldContainer[_insights_pb2.InsightReport]
    count: int
    def __init__(self, task_id: _Optional[str] = ..., reports: _Optional[_Iterable[_Union[_insights_pb2.InsightReport, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class InsightPromptPreviewResponse(_message.Message):
    __slots__ = ("task_id", "prompt", "limit", "status_filter", "executions")
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FILTER_FIELD_NUMBER: _ClassVar[int]
    EXECUTIONS_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    prompt: str
    limit: int
    status_filter: str
    executions: int
    def __init__(self, task_id: _Optional[str] = ..., prompt: _Optional[str] = ..., limit: _Optional[int] = ..., status_filter: _Optional[str] = ..., executions: _Optional[int] = ...) -> None: ...

class SuggestionStatusUpdateRequest(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class SuggestionStatusUpdateResponse(_message.Message):
    __slots__ = ("success", "message", "task_id", "report_id", "suggestion_id", "status")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    REPORT_ID_FIELD_NUMBER: _ClassVar[int]
    SUGGESTION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    message: str
    task_id: str
    report_id: str
    suggestion_id: str
    status: str
    def __init__(self, success: _Optional[bool] = ..., message: _Optional[str] = ..., task_id: _Optional[str] = ..., report_id: _Optional[str] = ..., suggestion_id: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class SuggestionApplyRequest(_message.Message):
    __slots__ = ("dry_run",)
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    def __init__(self, dry_run: _Optional[bool] = ...) -> None: ...

class SuggestionApplyResponse(_message.Message):
    __slots__ = ("success", "message", "files_changed", "backup_path", "dry_run", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    FILES_CHANGED_FIELD_NUMBER: _ClassVar[int]
    BACKUP_PATH_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    message: str
    files_changed: _containers.RepeatedScalarFieldContainer[str]
    backup_path: str
    dry_run: bool
    error: str
    def __init__(self, success: _Optional[bool] = ..., message: _Optional[str] = ..., files_changed: _Optional[_Iterable[str]] = ..., backup_path: _Optional[str] = ..., dry_run: _Optional[bool] = ..., error: _Optional[str] = ...) -> None: ...

class SystemInsightsResponse(_message.Message):
    __slots__ = ("reports", "total_reports", "unique_tasks", "total_executions", "patterns_by_type", "suggestions_by_type")
    class PatternsByTypeEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    class SuggestionsByTypeEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    REPORTS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_REPORTS_FIELD_NUMBER: _ClassVar[int]
    UNIQUE_TASKS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_EXECUTIONS_FIELD_NUMBER: _ClassVar[int]
    PATTERNS_BY_TYPE_FIELD_NUMBER: _ClassVar[int]
    SUGGESTIONS_BY_TYPE_FIELD_NUMBER: _ClassVar[int]
    reports: _containers.RepeatedCompositeFieldContainer[_insights_pb2.InsightReport]
    total_reports: int
    unique_tasks: int
    total_executions: int
    patterns_by_type: _containers.ScalarMap[str, int]
    suggestions_by_type: _containers.ScalarMap[str, int]
    def __init__(self, reports: _Optional[_Iterable[_Union[_insights_pb2.InsightReport, _Mapping]]] = ..., total_reports: _Optional[int] = ..., unique_tasks: _Optional[int] = ..., total_executions: _Optional[int] = ..., patterns_by_type: _Optional[_Mapping[str, int]] = ..., suggestions_by_type: _Optional[_Mapping[str, int]] = ...) -> None: ...

class SystemInsightReportResponse(_message.Message):
    __slots__ = ("message", "report_id", "report")
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    REPORT_ID_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    message: str
    report_id: str
    report: _insights_pb2.SystemInsightReport
    def __init__(self, message: _Optional[str] = ..., report_id: _Optional[str] = ..., report: _Optional[_Union[_insights_pb2.SystemInsightReport, _Mapping]] = ...) -> None: ...
