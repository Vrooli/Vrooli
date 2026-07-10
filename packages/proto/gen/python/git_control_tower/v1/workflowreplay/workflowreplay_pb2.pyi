from test_genie.v1.runs import runs_pb2 as _runs_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RunSummary(_message.Message):
    __slots__ = ("run_id", "status", "started_at", "completed_at", "git_sha", "git_branch", "git_dirty")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    GIT_SHA_FIELD_NUMBER: _ClassVar[int]
    GIT_BRANCH_FIELD_NUMBER: _ClassVar[int]
    GIT_DIRTY_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    status: str
    started_at: str
    completed_at: str
    git_sha: str
    git_branch: str
    git_dirty: bool
    def __init__(self, run_id: _Optional[str] = ..., status: _Optional[str] = ..., started_at: _Optional[str] = ..., completed_at: _Optional[str] = ..., git_sha: _Optional[str] = ..., git_branch: _Optional[str] = ..., git_dirty: _Optional[bool] = ...) -> None: ...

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

class GetRunDetailRequest(_message.Message):
    __slots__ = ("scenario", "run_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class GetRunDetailResponse(_message.Message):
    __slots__ = ("run", "artifacts")
    RUN_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    run: RunSummary
    artifacts: _containers.RepeatedCompositeFieldContainer[_runs_pb2.ArtifactRef]
    def __init__(self, run: _Optional[_Union[RunSummary, _Mapping]] = ..., artifacts: _Optional[_Iterable[_Union[_runs_pb2.ArtifactRef, _Mapping]]] = ...) -> None: ...
