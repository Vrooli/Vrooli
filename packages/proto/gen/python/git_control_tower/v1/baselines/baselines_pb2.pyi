from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GitState(_message.Message):
    __slots__ = ("sha", "branch", "detached", "dirty", "dirty_summary", "commit_message", "commit_author", "commit_date", "sandboxed")
    SHA_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    DETACHED_FIELD_NUMBER: _ClassVar[int]
    DIRTY_FIELD_NUMBER: _ClassVar[int]
    DIRTY_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    COMMIT_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    COMMIT_AUTHOR_FIELD_NUMBER: _ClassVar[int]
    COMMIT_DATE_FIELD_NUMBER: _ClassVar[int]
    SANDBOXED_FIELD_NUMBER: _ClassVar[int]
    sha: str
    branch: str
    detached: bool
    dirty: bool
    dirty_summary: str
    commit_message: str
    commit_author: str
    commit_date: str
    sandboxed: bool
    def __init__(self, sha: _Optional[str] = ..., branch: _Optional[str] = ..., detached: _Optional[bool] = ..., dirty: _Optional[bool] = ..., dirty_summary: _Optional[str] = ..., commit_message: _Optional[str] = ..., commit_author: _Optional[str] = ..., commit_date: _Optional[str] = ..., sandboxed: _Optional[bool] = ...) -> None: ...

class SurfacePointer(_message.Message):
    __slots__ = ("surface_id", "kind", "ref", "captured_at", "summary")
    SURFACE_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    REF_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    surface_id: str
    kind: str
    ref: str
    captured_at: str
    summary: str
    def __init__(self, surface_id: _Optional[str] = ..., kind: _Optional[str] = ..., ref: _Optional[str] = ..., captured_at: _Optional[str] = ..., summary: _Optional[str] = ...) -> None: ...

class BaselineManifest(_message.Message):
    __slots__ = ("name", "scenario", "branch", "created_at", "created_by", "git", "surfaces", "schema_version")
    class SurfacesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: SurfacePointer
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[SurfacePointer, _Mapping]] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    GIT_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    name: str
    scenario: str
    branch: str
    created_at: str
    created_by: str
    git: GitState
    surfaces: _containers.MessageMap[str, SurfacePointer]
    schema_version: int
    def __init__(self, name: _Optional[str] = ..., scenario: _Optional[str] = ..., branch: _Optional[str] = ..., created_at: _Optional[str] = ..., created_by: _Optional[str] = ..., git: _Optional[_Union[GitState, _Mapping]] = ..., surfaces: _Optional[_Mapping[str, SurfacePointer]] = ..., schema_version: _Optional[int] = ...) -> None: ...

class SurfaceDiff(_message.Message):
    __slots__ = ("surface_id", "verdict", "regressions", "new_failures", "preexisting", "cleared", "summary")
    SURFACE_ID_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    REGRESSIONS_FIELD_NUMBER: _ClassVar[int]
    NEW_FAILURES_FIELD_NUMBER: _ClassVar[int]
    PREEXISTING_FIELD_NUMBER: _ClassVar[int]
    CLEARED_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    surface_id: str
    verdict: str
    regressions: _containers.RepeatedScalarFieldContainer[str]
    new_failures: _containers.RepeatedScalarFieldContainer[str]
    preexisting: _containers.RepeatedScalarFieldContainer[str]
    cleared: _containers.RepeatedScalarFieldContainer[str]
    summary: str
    def __init__(self, surface_id: _Optional[str] = ..., verdict: _Optional[str] = ..., regressions: _Optional[_Iterable[str]] = ..., new_failures: _Optional[_Iterable[str]] = ..., preexisting: _Optional[_Iterable[str]] = ..., cleared: _Optional[_Iterable[str]] = ..., summary: _Optional[str] = ...) -> None: ...

class Staleness(_message.Message):
    __slots__ = ("commits_since", "files_changed", "likely_stale")
    COMMITS_SINCE_FIELD_NUMBER: _ClassVar[int]
    FILES_CHANGED_FIELD_NUMBER: _ClassVar[int]
    LIKELY_STALE_FIELD_NUMBER: _ClassVar[int]
    commits_since: int
    files_changed: int
    likely_stale: bool
    def __init__(self, commits_since: _Optional[int] = ..., files_changed: _Optional[int] = ..., likely_stale: _Optional[bool] = ...) -> None: ...

class CreateBaselineRequest(_message.Message):
    __slots__ = ("scenario", "name", "branch", "include", "fast", "created_by", "reason", "repo_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_FIELD_NUMBER: _ClassVar[int]
    FAST_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    name: str
    branch: str
    include: _containers.RepeatedScalarFieldContainer[str]
    fast: bool
    created_by: str
    reason: str
    repo_id: int
    def __init__(self, scenario: _Optional[str] = ..., name: _Optional[str] = ..., branch: _Optional[str] = ..., include: _Optional[_Iterable[str]] = ..., fast: _Optional[bool] = ..., created_by: _Optional[str] = ..., reason: _Optional[str] = ..., repo_id: _Optional[int] = ...) -> None: ...

class CreateBaselineResponse(_message.Message):
    __slots__ = ("baseline", "skipped", "dirty_warning")
    class SkippedEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    BASELINE_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    DIRTY_WARNING_FIELD_NUMBER: _ClassVar[int]
    baseline: BaselineManifest
    skipped: _containers.ScalarMap[str, str]
    dirty_warning: str
    def __init__(self, baseline: _Optional[_Union[BaselineManifest, _Mapping]] = ..., skipped: _Optional[_Mapping[str, str]] = ..., dirty_warning: _Optional[str] = ...) -> None: ...

class SnapshotForBaselineRequest(_message.Message):
    __slots__ = ("scenario", "name", "branch", "include", "fast", "created_by", "reason", "repo_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_FIELD_NUMBER: _ClassVar[int]
    FAST_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    name: str
    branch: str
    include: _containers.RepeatedScalarFieldContainer[str]
    fast: bool
    created_by: str
    reason: str
    repo_id: int
    def __init__(self, scenario: _Optional[str] = ..., name: _Optional[str] = ..., branch: _Optional[str] = ..., include: _Optional[_Iterable[str]] = ..., fast: _Optional[bool] = ..., created_by: _Optional[str] = ..., reason: _Optional[str] = ..., repo_id: _Optional[int] = ...) -> None: ...

class SnapshotForBaselineResponse(_message.Message):
    __slots__ = ("baseline", "skipped", "dirty_warning")
    class SkippedEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    BASELINE_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    DIRTY_WARNING_FIELD_NUMBER: _ClassVar[int]
    baseline: BaselineManifest
    skipped: _containers.ScalarMap[str, str]
    dirty_warning: str
    def __init__(self, baseline: _Optional[_Union[BaselineManifest, _Mapping]] = ..., skipped: _Optional[_Mapping[str, str]] = ..., dirty_warning: _Optional[str] = ...) -> None: ...

class GetBaselineRequest(_message.Message):
    __slots__ = ("scenario", "name", "branch", "repo_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    name: str
    branch: str
    repo_id: int
    def __init__(self, scenario: _Optional[str] = ..., name: _Optional[str] = ..., branch: _Optional[str] = ..., repo_id: _Optional[int] = ...) -> None: ...

class GetBaselineResponse(_message.Message):
    __slots__ = ("baseline",)
    BASELINE_FIELD_NUMBER: _ClassVar[int]
    baseline: BaselineManifest
    def __init__(self, baseline: _Optional[_Union[BaselineManifest, _Mapping]] = ...) -> None: ...

class ListBaselinesRequest(_message.Message):
    __slots__ = ("scenario", "branch", "all_branches", "repo_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    ALL_BRANCHES_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    branch: str
    all_branches: bool
    repo_id: int
    def __init__(self, scenario: _Optional[str] = ..., branch: _Optional[str] = ..., all_branches: _Optional[bool] = ..., repo_id: _Optional[int] = ...) -> None: ...

class ListBaselinesResponse(_message.Message):
    __slots__ = ("baselines",)
    BASELINES_FIELD_NUMBER: _ClassVar[int]
    baselines: _containers.RepeatedCompositeFieldContainer[BaselineManifest]
    def __init__(self, baselines: _Optional[_Iterable[_Union[BaselineManifest, _Mapping]]] = ...) -> None: ...

class DiffBaselineRequest(_message.Message):
    __slots__ = ("scenario", "name", "branch", "surface", "repo_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    name: str
    branch: str
    surface: str
    repo_id: int
    def __init__(self, scenario: _Optional[str] = ..., name: _Optional[str] = ..., branch: _Optional[str] = ..., surface: _Optional[str] = ..., repo_id: _Optional[int] = ...) -> None: ...

class DiffBaselineResponse(_message.Message):
    __slots__ = ("baseline", "current_git", "staleness", "surfaces", "verdict")
    BASELINE_FIELD_NUMBER: _ClassVar[int]
    CURRENT_GIT_FIELD_NUMBER: _ClassVar[int]
    STALENESS_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    baseline: BaselineManifest
    current_git: GitState
    staleness: Staleness
    surfaces: _containers.RepeatedCompositeFieldContainer[SurfaceDiff]
    verdict: str
    def __init__(self, baseline: _Optional[_Union[BaselineManifest, _Mapping]] = ..., current_git: _Optional[_Union[GitState, _Mapping]] = ..., staleness: _Optional[_Union[Staleness, _Mapping]] = ..., surfaces: _Optional[_Iterable[_Union[SurfaceDiff, _Mapping]]] = ..., verdict: _Optional[str] = ...) -> None: ...

class DeleteBaselineRequest(_message.Message):
    __slots__ = ("scenario", "name", "branch", "repo_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    name: str
    branch: str
    repo_id: int
    def __init__(self, scenario: _Optional[str] = ..., name: _Optional[str] = ..., branch: _Optional[str] = ..., repo_id: _Optional[int] = ...) -> None: ...

class DeleteBaselineResponse(_message.Message):
    __slots__ = ("deleted",)
    DELETED_FIELD_NUMBER: _ClassVar[int]
    deleted: bool
    def __init__(self, deleted: _Optional[bool] = ...) -> None: ...

class EditBaselineRequest(_message.Message):
    __slots__ = ("scenario", "name", "branch", "surface", "pin_run_id", "reason", "repo_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    PIN_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    name: str
    branch: str
    surface: str
    pin_run_id: str
    reason: str
    repo_id: int
    def __init__(self, scenario: _Optional[str] = ..., name: _Optional[str] = ..., branch: _Optional[str] = ..., surface: _Optional[str] = ..., pin_run_id: _Optional[str] = ..., reason: _Optional[str] = ..., repo_id: _Optional[int] = ...) -> None: ...

class EditBaselineResponse(_message.Message):
    __slots__ = ("baseline",)
    BASELINE_FIELD_NUMBER: _ClassVar[int]
    baseline: BaselineManifest
    def __init__(self, baseline: _Optional[_Union[BaselineManifest, _Mapping]] = ...) -> None: ...
