from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RunSummary(_message.Message):
    __slots__ = ("run_id", "status", "started_at", "completed_at", "git_sha", "git_branch", "git_dirty", "playbooks_status", "playbooks_duration_seconds")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    GIT_SHA_FIELD_NUMBER: _ClassVar[int]
    GIT_BRANCH_FIELD_NUMBER: _ClassVar[int]
    GIT_DIRTY_FIELD_NUMBER: _ClassVar[int]
    PLAYBOOKS_STATUS_FIELD_NUMBER: _ClassVar[int]
    PLAYBOOKS_DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    status: str
    started_at: str
    completed_at: str
    git_sha: str
    git_branch: str
    git_dirty: bool
    playbooks_status: str
    playbooks_duration_seconds: float
    def __init__(self, run_id: _Optional[str] = ..., status: _Optional[str] = ..., started_at: _Optional[str] = ..., completed_at: _Optional[str] = ..., git_sha: _Optional[str] = ..., git_branch: _Optional[str] = ..., git_dirty: _Optional[bool] = ..., playbooks_status: _Optional[str] = ..., playbooks_duration_seconds: _Optional[float] = ...) -> None: ...

class ListRecentRunsRequest(_message.Message):
    __slots__ = ("scenario", "limit")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    limit: int
    def __init__(self, scenario: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListRecentRunsResponse(_message.Message):
    __slots__ = ("runs",)
    RUNS_FIELD_NUMBER: _ClassVar[int]
    runs: _containers.RepeatedCompositeFieldContainer[RunSummary]
    def __init__(self, runs: _Optional[_Iterable[_Union[RunSummary, _Mapping]]] = ...) -> None: ...

class WorkflowVideo(_message.Message):
    __slots__ = ("workflow", "rel_path", "size_bytes")
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    REL_PATH_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    workflow: str
    rel_path: str
    size_bytes: int
    def __init__(self, workflow: _Optional[str] = ..., rel_path: _Optional[str] = ..., size_bytes: _Optional[int] = ...) -> None: ...

class GetRunDetailRequest(_message.Message):
    __slots__ = ("scenario", "run_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class GetRunDetailResponse(_message.Message):
    __slots__ = ("run", "videos")
    RUN_FIELD_NUMBER: _ClassVar[int]
    VIDEOS_FIELD_NUMBER: _ClassVar[int]
    run: RunSummary
    videos: _containers.RepeatedCompositeFieldContainer[WorkflowVideo]
    def __init__(self, run: _Optional[_Union[RunSummary, _Mapping]] = ..., videos: _Optional[_Iterable[_Union[WorkflowVideo, _Mapping]]] = ...) -> None: ...
