from google.api import annotations_pb2 as _annotations_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DatabaseStats(_message.Message):
    __slots__ = ("page_size", "page_count", "freelist_count", "size_bytes", "metric_rows")
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    FREELIST_COUNT_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    METRIC_ROWS_FIELD_NUMBER: _ClassVar[int]
    page_size: int
    page_count: int
    freelist_count: int
    size_bytes: int
    metric_rows: int
    def __init__(self, page_size: _Optional[int] = ..., page_count: _Optional[int] = ..., freelist_count: _Optional[int] = ..., size_bytes: _Optional[int] = ..., metric_rows: _Optional[int] = ...) -> None: ...

class RetentionEstimate(_message.Message):
    __slots__ = ("row_count", "payload_bytes", "oldest_affected", "newest_affected", "cutoff")
    ROW_COUNT_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_BYTES_FIELD_NUMBER: _ClassVar[int]
    OLDEST_AFFECTED_FIELD_NUMBER: _ClassVar[int]
    NEWEST_AFFECTED_FIELD_NUMBER: _ClassVar[int]
    CUTOFF_FIELD_NUMBER: _ClassVar[int]
    row_count: int
    payload_bytes: int
    oldest_affected: str
    newest_affected: str
    cutoff: str
    def __init__(self, row_count: _Optional[int] = ..., payload_bytes: _Optional[int] = ..., oldest_affected: _Optional[str] = ..., newest_affected: _Optional[str] = ..., cutoff: _Optional[str] = ...) -> None: ...

class RetentionResult(_message.Message):
    __slots__ = ("deleted_rows", "cutoff")
    DELETED_ROWS_FIELD_NUMBER: _ClassVar[int]
    CUTOFF_FIELD_NUMBER: _ClassVar[int]
    deleted_rows: int
    cutoff: str
    def __init__(self, deleted_rows: _Optional[int] = ..., cutoff: _Optional[str] = ...) -> None: ...

class MetricsRetentionPreviewRequest(_message.Message):
    __slots__ = ("retention_days",)
    RETENTION_DAYS_FIELD_NUMBER: _ClassVar[int]
    retention_days: int
    def __init__(self, retention_days: _Optional[int] = ...) -> None: ...

class MetricsRetentionPreviewResponse(_message.Message):
    __slots__ = ("success", "estimate", "database_stats", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ESTIMATE_FIELD_NUMBER: _ClassVar[int]
    DATABASE_STATS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    estimate: RetentionEstimate
    database_stats: DatabaseStats
    error: str
    def __init__(self, success: _Optional[bool] = ..., estimate: _Optional[_Union[RetentionEstimate, _Mapping]] = ..., database_stats: _Optional[_Union[DatabaseStats, _Mapping]] = ..., error: _Optional[str] = ...) -> None: ...

class MetricsRetentionApplyRequest(_message.Message):
    __slots__ = ("retention_days", "confirm")
    RETENTION_DAYS_FIELD_NUMBER: _ClassVar[int]
    CONFIRM_FIELD_NUMBER: _ClassVar[int]
    retention_days: int
    confirm: bool
    def __init__(self, retention_days: _Optional[int] = ..., confirm: _Optional[bool] = ...) -> None: ...

class MetricsRetentionApplyResponse(_message.Message):
    __slots__ = ("success", "result", "database_stats_before", "database_stats_after", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    DATABASE_STATS_BEFORE_FIELD_NUMBER: _ClassVar[int]
    DATABASE_STATS_AFTER_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    result: RetentionResult
    database_stats_before: DatabaseStats
    database_stats_after: DatabaseStats
    error: str
    def __init__(self, success: _Optional[bool] = ..., result: _Optional[_Union[RetentionResult, _Mapping]] = ..., database_stats_before: _Optional[_Union[DatabaseStats, _Mapping]] = ..., database_stats_after: _Optional[_Union[DatabaseStats, _Mapping]] = ..., error: _Optional[str] = ...) -> None: ...

class MetricsCompactionPreviewRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class MetricsCompactionPreviewResponse(_message.Message):
    __slots__ = ("success", "database_stats", "estimated_reclaimable_bytes", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATABASE_STATS_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_RECLAIMABLE_BYTES_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    database_stats: DatabaseStats
    estimated_reclaimable_bytes: int
    error: str
    def __init__(self, success: _Optional[bool] = ..., database_stats: _Optional[_Union[DatabaseStats, _Mapping]] = ..., estimated_reclaimable_bytes: _Optional[int] = ..., error: _Optional[str] = ...) -> None: ...

class MetricsCompactionApplyRequest(_message.Message):
    __slots__ = ("confirm",)
    CONFIRM_FIELD_NUMBER: _ClassVar[int]
    confirm: bool
    def __init__(self, confirm: _Optional[bool] = ...) -> None: ...

class MetricsCompactionApplyResponse(_message.Message):
    __slots__ = ("success", "database_stats_before", "database_stats_after", "reclaimed_bytes", "error")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    DATABASE_STATS_BEFORE_FIELD_NUMBER: _ClassVar[int]
    DATABASE_STATS_AFTER_FIELD_NUMBER: _ClassVar[int]
    RECLAIMED_BYTES_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    success: bool
    database_stats_before: DatabaseStats
    database_stats_after: DatabaseStats
    reclaimed_bytes: int
    error: str
    def __init__(self, success: _Optional[bool] = ..., database_stats_before: _Optional[_Union[DatabaseStats, _Mapping]] = ..., database_stats_after: _Optional[_Union[DatabaseStats, _Mapping]] = ..., reclaimed_bytes: _Optional[int] = ..., error: _Optional[str] = ...) -> None: ...
