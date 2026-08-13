from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ResolveWorkspaceRequest(_message.Message):
    __slots__ = ("sandbox_id",)
    SANDBOX_ID_FIELD_NUMBER: _ClassVar[int]
    sandbox_id: str
    def __init__(self, sandbox_id: _Optional[str] = ...) -> None: ...

class ResolveWorkspaceResponse(_message.Message):
    __slots__ = ("success", "sandbox_id", "workspace_root", "isolation_mode")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    SANDBOX_ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_ROOT_FIELD_NUMBER: _ClassVar[int]
    ISOLATION_MODE_FIELD_NUMBER: _ClassVar[int]
    success: bool
    sandbox_id: str
    workspace_root: str
    isolation_mode: str
    def __init__(self, success: _Optional[bool] = ..., sandbox_id: _Optional[str] = ..., workspace_root: _Optional[str] = ..., isolation_mode: _Optional[str] = ...) -> None: ...

class CreateSandboxRequest(_message.Message):
    __slots__ = ("name", "scope_path", "project_root", "owner", "reserved_paths", "idempotency_key")
    NAME_FIELD_NUMBER: _ClassVar[int]
    SCOPE_PATH_FIELD_NUMBER: _ClassVar[int]
    PROJECT_ROOT_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    RESERVED_PATHS_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    name: str
    scope_path: str
    project_root: str
    owner: str
    reserved_paths: _containers.RepeatedScalarFieldContainer[str]
    idempotency_key: str
    def __init__(self, name: _Optional[str] = ..., scope_path: _Optional[str] = ..., project_root: _Optional[str] = ..., owner: _Optional[str] = ..., reserved_paths: _Optional[_Iterable[str]] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class SandboxSummary(_message.Message):
    __slots__ = ("sandbox_id", "status", "workspace_root", "isolation_mode", "scope_path", "project_root")
    SANDBOX_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_ROOT_FIELD_NUMBER: _ClassVar[int]
    ISOLATION_MODE_FIELD_NUMBER: _ClassVar[int]
    SCOPE_PATH_FIELD_NUMBER: _ClassVar[int]
    PROJECT_ROOT_FIELD_NUMBER: _ClassVar[int]
    sandbox_id: str
    status: str
    workspace_root: str
    isolation_mode: str
    scope_path: str
    project_root: str
    def __init__(self, sandbox_id: _Optional[str] = ..., status: _Optional[str] = ..., workspace_root: _Optional[str] = ..., isolation_mode: _Optional[str] = ..., scope_path: _Optional[str] = ..., project_root: _Optional[str] = ...) -> None: ...

class CreateSandboxResponse(_message.Message):
    __slots__ = ("success", "sandbox")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    SANDBOX_FIELD_NUMBER: _ClassVar[int]
    success: bool
    sandbox: SandboxSummary
    def __init__(self, success: _Optional[bool] = ..., sandbox: _Optional[_Union[SandboxSummary, _Mapping]] = ...) -> None: ...

class GetSandboxDiffRequest(_message.Message):
    __slots__ = ("sandbox_id",)
    SANDBOX_ID_FIELD_NUMBER: _ClassVar[int]
    sandbox_id: str
    def __init__(self, sandbox_id: _Optional[str] = ...) -> None: ...

class DiffFile(_message.Message):
    __slots__ = ("path", "change_type", "size", "approval_status")
    PATH_FIELD_NUMBER: _ClassVar[int]
    CHANGE_TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    APPROVAL_STATUS_FIELD_NUMBER: _ClassVar[int]
    path: str
    change_type: str
    size: int
    approval_status: str
    def __init__(self, path: _Optional[str] = ..., change_type: _Optional[str] = ..., size: _Optional[int] = ..., approval_status: _Optional[str] = ...) -> None: ...

class DiffStats(_message.Message):
    __slots__ = ("files_changed", "files_added", "files_modified", "files_deleted", "lines_added", "lines_removed", "total_bytes")
    FILES_CHANGED_FIELD_NUMBER: _ClassVar[int]
    FILES_ADDED_FIELD_NUMBER: _ClassVar[int]
    FILES_MODIFIED_FIELD_NUMBER: _ClassVar[int]
    FILES_DELETED_FIELD_NUMBER: _ClassVar[int]
    LINES_ADDED_FIELD_NUMBER: _ClassVar[int]
    LINES_REMOVED_FIELD_NUMBER: _ClassVar[int]
    TOTAL_BYTES_FIELD_NUMBER: _ClassVar[int]
    files_changed: int
    files_added: int
    files_modified: int
    files_deleted: int
    lines_added: int
    lines_removed: int
    total_bytes: int
    def __init__(self, files_changed: _Optional[int] = ..., files_added: _Optional[int] = ..., files_modified: _Optional[int] = ..., files_deleted: _Optional[int] = ..., lines_added: _Optional[int] = ..., lines_removed: _Optional[int] = ..., total_bytes: _Optional[int] = ...) -> None: ...

class GetSandboxDiffResponse(_message.Message):
    __slots__ = ("success", "sandbox_id", "files", "unified_diff", "stats", "archive_state")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    SANDBOX_ID_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    UNIFIED_DIFF_FIELD_NUMBER: _ClassVar[int]
    STATS_FIELD_NUMBER: _ClassVar[int]
    ARCHIVE_STATE_FIELD_NUMBER: _ClassVar[int]
    success: bool
    sandbox_id: str
    files: _containers.RepeatedCompositeFieldContainer[DiffFile]
    unified_diff: str
    stats: DiffStats
    archive_state: str
    def __init__(self, success: _Optional[bool] = ..., sandbox_id: _Optional[str] = ..., files: _Optional[_Iterable[_Union[DiffFile, _Mapping]]] = ..., unified_diff: _Optional[str] = ..., stats: _Optional[_Union[DiffStats, _Mapping]] = ..., archive_state: _Optional[str] = ...) -> None: ...

class PromoteSandboxRequest(_message.Message):
    __slots__ = ("sandbox_id", "mode", "actor", "commit_message", "create_commit", "force", "override_acceptance", "confirm")
    SANDBOX_ID_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    COMMIT_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    CREATE_COMMIT_FIELD_NUMBER: _ClassVar[int]
    FORCE_FIELD_NUMBER: _ClassVar[int]
    OVERRIDE_ACCEPTANCE_FIELD_NUMBER: _ClassVar[int]
    CONFIRM_FIELD_NUMBER: _ClassVar[int]
    sandbox_id: str
    mode: str
    actor: str
    commit_message: str
    create_commit: bool
    force: bool
    override_acceptance: bool
    confirm: bool
    def __init__(self, sandbox_id: _Optional[str] = ..., mode: _Optional[str] = ..., actor: _Optional[str] = ..., commit_message: _Optional[str] = ..., create_commit: _Optional[bool] = ..., force: _Optional[bool] = ..., override_acceptance: _Optional[bool] = ..., confirm: _Optional[bool] = ...) -> None: ...

class PromoteSandboxResponse(_message.Message):
    __slots__ = ("success", "sandbox_id", "applied", "failed", "remaining", "is_partial", "commit_hash", "error", "diff_path")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    SANDBOX_ID_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    REMAINING_FIELD_NUMBER: _ClassVar[int]
    IS_PARTIAL_FIELD_NUMBER: _ClassVar[int]
    COMMIT_HASH_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    DIFF_PATH_FIELD_NUMBER: _ClassVar[int]
    success: bool
    sandbox_id: str
    applied: int
    failed: int
    remaining: int
    is_partial: bool
    commit_hash: str
    error: str
    diff_path: str
    def __init__(self, success: _Optional[bool] = ..., sandbox_id: _Optional[str] = ..., applied: _Optional[int] = ..., failed: _Optional[int] = ..., remaining: _Optional[int] = ..., is_partial: _Optional[bool] = ..., commit_hash: _Optional[str] = ..., error: _Optional[str] = ..., diff_path: _Optional[str] = ...) -> None: ...
