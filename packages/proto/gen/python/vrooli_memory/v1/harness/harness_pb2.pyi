from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RuntimeRequest(_message.Message):
    __slots__ = ("runtime",)
    RUNTIME_FIELD_NUMBER: _ClassVar[int]
    runtime: str
    def __init__(self, runtime: _Optional[str] = ...) -> None: ...

class RunImportRequest(_message.Message):
    __slots__ = ("runtime", "dry_run")
    RUNTIME_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    runtime: str
    dry_run: bool
    def __init__(self, runtime: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class RunImportResponse(_message.Message):
    __slots__ = ("imported_count", "run", "joined_existing_run", "dry_run")
    IMPORTED_COUNT_FIELD_NUMBER: _ClassVar[int]
    RUN_FIELD_NUMBER: _ClassVar[int]
    JOINED_EXISTING_RUN_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    imported_count: int
    run: ImportRun
    joined_existing_run: bool
    dry_run: bool
    def __init__(self, imported_count: _Optional[int] = ..., run: _Optional[_Union[ImportRun, _Mapping]] = ..., joined_existing_run: _Optional[bool] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class GetImportStatusRequest(_message.Message):
    __slots__ = ("run_id", "runtime")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    runtime: str
    def __init__(self, run_id: _Optional[str] = ..., runtime: _Optional[str] = ...) -> None: ...

class GetImportStatusResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: ImportRun
    def __init__(self, run: _Optional[_Union[ImportRun, _Mapping]] = ...) -> None: ...

class ImportRun(_message.Message):
    __slots__ = ("id", "runtime", "source_root", "status", "total_sources", "processed_sources", "imported_count", "existing_count", "failed_count", "current_path", "error_message", "started_at", "completed_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ROOT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_SOURCES_FIELD_NUMBER: _ClassVar[int]
    PROCESSED_SOURCES_FIELD_NUMBER: _ClassVar[int]
    IMPORTED_COUNT_FIELD_NUMBER: _ClassVar[int]
    EXISTING_COUNT_FIELD_NUMBER: _ClassVar[int]
    FAILED_COUNT_FIELD_NUMBER: _ClassVar[int]
    CURRENT_PATH_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    runtime: str
    source_root: str
    status: str
    total_sources: int
    processed_sources: int
    imported_count: int
    existing_count: int
    failed_count: int
    current_path: str
    error_message: str
    started_at: str
    completed_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., runtime: _Optional[str] = ..., source_root: _Optional[str] = ..., status: _Optional[str] = ..., total_sources: _Optional[int] = ..., processed_sources: _Optional[int] = ..., imported_count: _Optional[int] = ..., existing_count: _Optional[int] = ..., failed_count: _Optional[int] = ..., current_path: _Optional[str] = ..., error_message: _Optional[str] = ..., started_at: _Optional[str] = ..., completed_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class RefreshProjectionRequest(_message.Message):
    __slots__ = ("runtime", "dry_run")
    RUNTIME_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    runtime: str
    dry_run: bool
    def __init__(self, runtime: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class RefreshProjectionResponse(_message.Message):
    __slots__ = ("path", "size_bytes", "overflow", "dry_run", "rendered_content")
    PATH_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    OVERFLOW_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    RENDERED_CONTENT_FIELD_NUMBER: _ClassVar[int]
    path: str
    size_bytes: int
    overflow: bool
    dry_run: bool
    rendered_content: str
    def __init__(self, path: _Optional[str] = ..., size_bytes: _Optional[int] = ..., overflow: _Optional[bool] = ..., dry_run: _Optional[bool] = ..., rendered_content: _Optional[str] = ...) -> None: ...

class InstallPromptBlockRequest(_message.Message):
    __slots__ = ("runtime",)
    RUNTIME_FIELD_NUMBER: _ClassVar[int]
    runtime: str
    def __init__(self, runtime: _Optional[str] = ...) -> None: ...

class InstallPromptBlockResponse(_message.Message):
    __slots__ = ("installed",)
    INSTALLED_FIELD_NUMBER: _ClassVar[int]
    installed: bool
    def __init__(self, installed: _Optional[bool] = ...) -> None: ...

class CaptureWriteRequest(_message.Message):
    __slots__ = ("runtime", "content", "source_path")
    RUNTIME_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    runtime: str
    content: str
    source_path: str
    def __init__(self, runtime: _Optional[str] = ..., content: _Optional[str] = ..., source_path: _Optional[str] = ...) -> None: ...

class CaptureWriteResponse(_message.Message):
    __slots__ = ("entry_id",)
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    entry_id: str
    def __init__(self, entry_id: _Optional[str] = ...) -> None: ...
