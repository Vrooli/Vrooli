from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Worktree(_message.Message):
    __slots__ = ("path", "name", "head_commit", "branch", "detached", "locked", "lock_reason", "prunable", "prunable_reason", "is_main")
    PATH_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    HEAD_COMMIT_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    DETACHED_FIELD_NUMBER: _ClassVar[int]
    LOCKED_FIELD_NUMBER: _ClassVar[int]
    LOCK_REASON_FIELD_NUMBER: _ClassVar[int]
    PRUNABLE_FIELD_NUMBER: _ClassVar[int]
    PRUNABLE_REASON_FIELD_NUMBER: _ClassVar[int]
    IS_MAIN_FIELD_NUMBER: _ClassVar[int]
    path: str
    name: str
    head_commit: str
    branch: str
    detached: bool
    locked: bool
    lock_reason: str
    prunable: bool
    prunable_reason: str
    is_main: bool
    def __init__(self, path: _Optional[str] = ..., name: _Optional[str] = ..., head_commit: _Optional[str] = ..., branch: _Optional[str] = ..., detached: _Optional[bool] = ..., locked: _Optional[bool] = ..., lock_reason: _Optional[str] = ..., prunable: _Optional[bool] = ..., prunable_reason: _Optional[str] = ..., is_main: _Optional[bool] = ...) -> None: ...

class ListWorktreesRequest(_message.Message):
    __slots__ = ("repo_path",)
    REPO_PATH_FIELD_NUMBER: _ClassVar[int]
    repo_path: str
    def __init__(self, repo_path: _Optional[str] = ...) -> None: ...

class ListWorktreesResponse(_message.Message):
    __slots__ = ("worktrees",)
    WORKTREES_FIELD_NUMBER: _ClassVar[int]
    worktrees: _containers.RepeatedCompositeFieldContainer[Worktree]
    def __init__(self, worktrees: _Optional[_Iterable[_Union[Worktree, _Mapping]]] = ...) -> None: ...

class GetWorktreeRequest(_message.Message):
    __slots__ = ("repo_path", "worktree_path")
    REPO_PATH_FIELD_NUMBER: _ClassVar[int]
    WORKTREE_PATH_FIELD_NUMBER: _ClassVar[int]
    repo_path: str
    worktree_path: str
    def __init__(self, repo_path: _Optional[str] = ..., worktree_path: _Optional[str] = ...) -> None: ...

class GetWorktreeResponse(_message.Message):
    __slots__ = ("worktree",)
    WORKTREE_FIELD_NUMBER: _ClassVar[int]
    worktree: Worktree
    def __init__(self, worktree: _Optional[_Union[Worktree, _Mapping]] = ...) -> None: ...

class NewBranchSpec(_message.Message):
    __slots__ = ("name", "start_point")
    NAME_FIELD_NUMBER: _ClassVar[int]
    START_POINT_FIELD_NUMBER: _ClassVar[int]
    name: str
    start_point: str
    def __init__(self, name: _Optional[str] = ..., start_point: _Optional[str] = ...) -> None: ...

class CreateWorktreeRequest(_message.Message):
    __slots__ = ("repo_path", "new_worktree_path", "existing_branch", "new_branch", "commit", "force", "track")
    REPO_PATH_FIELD_NUMBER: _ClassVar[int]
    NEW_WORKTREE_PATH_FIELD_NUMBER: _ClassVar[int]
    EXISTING_BRANCH_FIELD_NUMBER: _ClassVar[int]
    NEW_BRANCH_FIELD_NUMBER: _ClassVar[int]
    COMMIT_FIELD_NUMBER: _ClassVar[int]
    FORCE_FIELD_NUMBER: _ClassVar[int]
    TRACK_FIELD_NUMBER: _ClassVar[int]
    repo_path: str
    new_worktree_path: str
    existing_branch: str
    new_branch: NewBranchSpec
    commit: str
    force: bool
    track: bool
    def __init__(self, repo_path: _Optional[str] = ..., new_worktree_path: _Optional[str] = ..., existing_branch: _Optional[str] = ..., new_branch: _Optional[_Union[NewBranchSpec, _Mapping]] = ..., commit: _Optional[str] = ..., force: _Optional[bool] = ..., track: _Optional[bool] = ...) -> None: ...

class CreateWorktreeResponse(_message.Message):
    __slots__ = ("worktree", "dry_run")
    WORKTREE_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    worktree: Worktree
    dry_run: bool
    def __init__(self, worktree: _Optional[_Union[Worktree, _Mapping]] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class RemoveWorktreeRequest(_message.Message):
    __slots__ = ("repo_path", "worktree_path", "force")
    REPO_PATH_FIELD_NUMBER: _ClassVar[int]
    WORKTREE_PATH_FIELD_NUMBER: _ClassVar[int]
    FORCE_FIELD_NUMBER: _ClassVar[int]
    repo_path: str
    worktree_path: str
    force: bool
    def __init__(self, repo_path: _Optional[str] = ..., worktree_path: _Optional[str] = ..., force: _Optional[bool] = ...) -> None: ...

class RemoveWorktreeResponse(_message.Message):
    __slots__ = ("dry_run",)
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    def __init__(self, dry_run: _Optional[bool] = ...) -> None: ...

class LockWorktreeRequest(_message.Message):
    __slots__ = ("repo_path", "worktree_path", "reason")
    REPO_PATH_FIELD_NUMBER: _ClassVar[int]
    WORKTREE_PATH_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    repo_path: str
    worktree_path: str
    reason: str
    def __init__(self, repo_path: _Optional[str] = ..., worktree_path: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class LockWorktreeResponse(_message.Message):
    __slots__ = ("worktree", "dry_run")
    WORKTREE_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    worktree: Worktree
    dry_run: bool
    def __init__(self, worktree: _Optional[_Union[Worktree, _Mapping]] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class UnlockWorktreeRequest(_message.Message):
    __slots__ = ("repo_path", "worktree_path")
    REPO_PATH_FIELD_NUMBER: _ClassVar[int]
    WORKTREE_PATH_FIELD_NUMBER: _ClassVar[int]
    repo_path: str
    worktree_path: str
    def __init__(self, repo_path: _Optional[str] = ..., worktree_path: _Optional[str] = ...) -> None: ...

class UnlockWorktreeResponse(_message.Message):
    __slots__ = ("worktree", "dry_run")
    WORKTREE_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    worktree: Worktree
    dry_run: bool
    def __init__(self, worktree: _Optional[_Union[Worktree, _Mapping]] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class MoveWorktreeRequest(_message.Message):
    __slots__ = ("repo_path", "worktree_path", "new_worktree_path")
    REPO_PATH_FIELD_NUMBER: _ClassVar[int]
    WORKTREE_PATH_FIELD_NUMBER: _ClassVar[int]
    NEW_WORKTREE_PATH_FIELD_NUMBER: _ClassVar[int]
    repo_path: str
    worktree_path: str
    new_worktree_path: str
    def __init__(self, repo_path: _Optional[str] = ..., worktree_path: _Optional[str] = ..., new_worktree_path: _Optional[str] = ...) -> None: ...

class MoveWorktreeResponse(_message.Message):
    __slots__ = ("worktree", "dry_run")
    WORKTREE_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    worktree: Worktree
    dry_run: bool
    def __init__(self, worktree: _Optional[_Union[Worktree, _Mapping]] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class PruneWorktreesRequest(_message.Message):
    __slots__ = ("repo_path", "reason", "report_only")
    REPO_PATH_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    REPORT_ONLY_FIELD_NUMBER: _ClassVar[int]
    repo_path: str
    reason: str
    report_only: bool
    def __init__(self, repo_path: _Optional[str] = ..., reason: _Optional[str] = ..., report_only: _Optional[bool] = ...) -> None: ...

class PruneWorktreesResponse(_message.Message):
    __slots__ = ("pruned_paths", "dry_run")
    PRUNED_PATHS_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    pruned_paths: _containers.RepeatedScalarFieldContainer[str]
    dry_run: bool
    def __init__(self, pruned_paths: _Optional[_Iterable[str]] = ..., dry_run: _Optional[bool] = ...) -> None: ...
