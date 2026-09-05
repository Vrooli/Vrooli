from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetRepoStatusRequest(_message.Message):
    __slots__ = ("repo_path",)
    REPO_PATH_FIELD_NUMBER: _ClassVar[int]
    repo_path: str
    def __init__(self, repo_path: _Optional[str] = ...) -> None: ...

class WorktreeIdentity(_message.Message):
    __slots__ = ("is_linked_worktree", "common_repo_root", "worktree_name", "worktree_head", "linked_worktree_count")
    IS_LINKED_WORKTREE_FIELD_NUMBER: _ClassVar[int]
    COMMON_REPO_ROOT_FIELD_NUMBER: _ClassVar[int]
    WORKTREE_NAME_FIELD_NUMBER: _ClassVar[int]
    WORKTREE_HEAD_FIELD_NUMBER: _ClassVar[int]
    LINKED_WORKTREE_COUNT_FIELD_NUMBER: _ClassVar[int]
    is_linked_worktree: bool
    common_repo_root: str
    worktree_name: str
    worktree_head: str
    linked_worktree_count: int
    def __init__(self, is_linked_worktree: _Optional[bool] = ..., common_repo_root: _Optional[str] = ..., worktree_name: _Optional[str] = ..., worktree_head: _Optional[str] = ..., linked_worktree_count: _Optional[int] = ...) -> None: ...

class GetRepoStatusResponse(_message.Message):
    __slots__ = ("branch", "detached", "worktree")
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    DETACHED_FIELD_NUMBER: _ClassVar[int]
    WORKTREE_FIELD_NUMBER: _ClassVar[int]
    branch: str
    detached: bool
    worktree: WorktreeIdentity
    def __init__(self, branch: _Optional[str] = ..., detached: _Optional[bool] = ..., worktree: _Optional[_Union[WorktreeIdentity, _Mapping]] = ...) -> None: ...
