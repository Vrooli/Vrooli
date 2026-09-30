import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListBranchesRequest(_message.Message):
    __slots__ = ("repository_id",)
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    def __init__(self, repository_id: _Optional[str] = ...) -> None: ...

class BranchInfo(_message.Message):
    __slots__ = ("name", "upstream", "oid", "last_commit_at", "ahead", "behind", "is_current", "checked_out_in_worktree")
    NAME_FIELD_NUMBER: _ClassVar[int]
    UPSTREAM_FIELD_NUMBER: _ClassVar[int]
    OID_FIELD_NUMBER: _ClassVar[int]
    LAST_COMMIT_AT_FIELD_NUMBER: _ClassVar[int]
    AHEAD_FIELD_NUMBER: _ClassVar[int]
    BEHIND_FIELD_NUMBER: _ClassVar[int]
    IS_CURRENT_FIELD_NUMBER: _ClassVar[int]
    CHECKED_OUT_IN_WORKTREE_FIELD_NUMBER: _ClassVar[int]
    name: str
    upstream: str
    oid: str
    last_commit_at: _timestamp_pb2.Timestamp
    ahead: int
    behind: int
    is_current: bool
    checked_out_in_worktree: str
    def __init__(self, name: _Optional[str] = ..., upstream: _Optional[str] = ..., oid: _Optional[str] = ..., last_commit_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., ahead: _Optional[int] = ..., behind: _Optional[int] = ..., is_current: _Optional[bool] = ..., checked_out_in_worktree: _Optional[str] = ...) -> None: ...

class BranchWarning(_message.Message):
    __slots__ = ("message", "requires_confirmation", "requires_tracking", "requires_fetch", "dirty_summary")
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_CONFIRMATION_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_TRACKING_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_FETCH_FIELD_NUMBER: _ClassVar[int]
    DIRTY_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    message: str
    requires_confirmation: bool
    requires_tracking: bool
    requires_fetch: bool
    dirty_summary: DirtySummary
    def __init__(self, message: _Optional[str] = ..., requires_confirmation: _Optional[bool] = ..., requires_tracking: _Optional[bool] = ..., requires_fetch: _Optional[bool] = ..., dirty_summary: _Optional[_Union[DirtySummary, _Mapping]] = ...) -> None: ...

class DirtySummary(_message.Message):
    __slots__ = ("staged", "unstaged", "untracked", "conflicts")
    STAGED_FIELD_NUMBER: _ClassVar[int]
    UNSTAGED_FIELD_NUMBER: _ClassVar[int]
    UNTRACKED_FIELD_NUMBER: _ClassVar[int]
    CONFLICTS_FIELD_NUMBER: _ClassVar[int]
    staged: int
    unstaged: int
    untracked: int
    conflicts: int
    def __init__(self, staged: _Optional[int] = ..., unstaged: _Optional[int] = ..., untracked: _Optional[int] = ..., conflicts: _Optional[int] = ...) -> None: ...

class ListBranchesResponse(_message.Message):
    __slots__ = ("current", "locals", "remotes", "timestamp")
    CURRENT_FIELD_NUMBER: _ClassVar[int]
    LOCALS_FIELD_NUMBER: _ClassVar[int]
    REMOTES_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    current: str
    locals: _containers.RepeatedCompositeFieldContainer[BranchInfo]
    remotes: _containers.RepeatedCompositeFieldContainer[BranchInfo]
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, current: _Optional[str] = ..., locals: _Optional[_Iterable[_Union[BranchInfo, _Mapping]]] = ..., remotes: _Optional[_Iterable[_Union[BranchInfo, _Mapping]]] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CreateBranchRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "name", "checkout", "allow_dirty")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    FROM_FIELD_NUMBER: _ClassVar[int]
    CHECKOUT_FIELD_NUMBER: _ClassVar[int]
    ALLOW_DIRTY_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    name: str
    checkout: bool
    allow_dirty: bool
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., name: _Optional[str] = ..., checkout: _Optional[bool] = ..., allow_dirty: _Optional[bool] = ..., **kwargs) -> None: ...

class CreateBranchResponse(_message.Message):
    __slots__ = ("success", "branch", "warning", "error", "validation_errors", "timestamp")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    WARNING_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_ERRORS_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    success: bool
    branch: BranchInfo
    warning: BranchWarning
    error: str
    validation_errors: _containers.RepeatedScalarFieldContainer[str]
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, success: _Optional[bool] = ..., branch: _Optional[_Union[BranchInfo, _Mapping]] = ..., warning: _Optional[_Union[BranchWarning, _Mapping]] = ..., error: _Optional[str] = ..., validation_errors: _Optional[_Iterable[str]] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class SwitchBranchRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "name", "allow_dirty", "track_remote")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    ALLOW_DIRTY_FIELD_NUMBER: _ClassVar[int]
    TRACK_REMOTE_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    name: str
    allow_dirty: bool
    track_remote: bool
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., name: _Optional[str] = ..., allow_dirty: _Optional[bool] = ..., track_remote: _Optional[bool] = ...) -> None: ...

class SwitchBranchResponse(_message.Message):
    __slots__ = ("success", "branch", "warning", "error", "timestamp")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    WARNING_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    success: bool
    branch: BranchInfo
    warning: BranchWarning
    error: str
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, success: _Optional[bool] = ..., branch: _Optional[_Union[BranchInfo, _Mapping]] = ..., warning: _Optional[_Union[BranchWarning, _Mapping]] = ..., error: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class PublishBranchRequest(_message.Message):
    __slots__ = ("repository_id", "intent_id", "remote", "branch", "set_upstream", "fetch")
    REPOSITORY_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_ID_FIELD_NUMBER: _ClassVar[int]
    REMOTE_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    SET_UPSTREAM_FIELD_NUMBER: _ClassVar[int]
    FETCH_FIELD_NUMBER: _ClassVar[int]
    repository_id: str
    intent_id: str
    remote: str
    branch: str
    set_upstream: bool
    fetch: bool
    def __init__(self, repository_id: _Optional[str] = ..., intent_id: _Optional[str] = ..., remote: _Optional[str] = ..., branch: _Optional[str] = ..., set_upstream: _Optional[bool] = ..., fetch: _Optional[bool] = ...) -> None: ...

class PublishBranchResponse(_message.Message):
    __slots__ = ("success", "remote", "branch", "warning", "error", "timestamp")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    REMOTE_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    WARNING_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    success: bool
    remote: str
    branch: str
    warning: BranchWarning
    error: str
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, success: _Optional[bool] = ..., remote: _Optional[str] = ..., branch: _Optional[str] = ..., warning: _Optional[_Union[BranchWarning, _Mapping]] = ..., error: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
