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
    __slots__ = ("name", "scenario", "branch", "created_at", "created_by", "git", "surfaces", "schema_version", "skipped")
    class SurfacesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: SurfacePointer
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[SurfacePointer, _Mapping]] = ...) -> None: ...
    class SkippedEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    GIT_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    name: str
    scenario: str
    branch: str
    created_at: str
    created_by: str
    git: GitState
    surfaces: _containers.MessageMap[str, SurfacePointer]
    schema_version: int
    skipped: _containers.ScalarMap[str, str]
    def __init__(self, name: _Optional[str] = ..., scenario: _Optional[str] = ..., branch: _Optional[str] = ..., created_at: _Optional[str] = ..., created_by: _Optional[str] = ..., git: _Optional[_Union[GitState, _Mapping]] = ..., surfaces: _Optional[_Mapping[str, SurfacePointer]] = ..., schema_version: _Optional[int] = ..., skipped: _Optional[_Mapping[str, str]] = ...) -> None: ...

class SurfaceDiff(_message.Message):
    __slots__ = ("surface_id", "verdict", "regressions", "new_failures", "preexisting", "cleared", "summary", "changed")
    SURFACE_ID_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    REGRESSIONS_FIELD_NUMBER: _ClassVar[int]
    NEW_FAILURES_FIELD_NUMBER: _ClassVar[int]
    PREEXISTING_FIELD_NUMBER: _ClassVar[int]
    CLEARED_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    CHANGED_FIELD_NUMBER: _ClassVar[int]
    surface_id: str
    verdict: str
    regressions: _containers.RepeatedScalarFieldContainer[str]
    new_failures: _containers.RepeatedScalarFieldContainer[str]
    preexisting: _containers.RepeatedScalarFieldContainer[str]
    cleared: _containers.RepeatedScalarFieldContainer[str]
    summary: str
    changed: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, surface_id: _Optional[str] = ..., verdict: _Optional[str] = ..., regressions: _Optional[_Iterable[str]] = ..., new_failures: _Optional[_Iterable[str]] = ..., preexisting: _Optional[_Iterable[str]] = ..., cleared: _Optional[_Iterable[str]] = ..., summary: _Optional[str] = ..., changed: _Optional[_Iterable[str]] = ...) -> None: ...

class PhaseDiff(_message.Message):
    __slots__ = ("phase", "surface_id", "verdict", "regressions", "new_failures", "preexisting", "cleared", "summary")
    PHASE_FIELD_NUMBER: _ClassVar[int]
    SURFACE_ID_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    REGRESSIONS_FIELD_NUMBER: _ClassVar[int]
    NEW_FAILURES_FIELD_NUMBER: _ClassVar[int]
    PREEXISTING_FIELD_NUMBER: _ClassVar[int]
    CLEARED_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    phase: str
    surface_id: str
    verdict: str
    regressions: _containers.RepeatedScalarFieldContainer[str]
    new_failures: _containers.RepeatedScalarFieldContainer[str]
    preexisting: _containers.RepeatedScalarFieldContainer[str]
    cleared: _containers.RepeatedScalarFieldContainer[str]
    summary: str
    def __init__(self, phase: _Optional[str] = ..., surface_id: _Optional[str] = ..., verdict: _Optional[str] = ..., regressions: _Optional[_Iterable[str]] = ..., new_failures: _Optional[_Iterable[str]] = ..., preexisting: _Optional[_Iterable[str]] = ..., cleared: _Optional[_Iterable[str]] = ..., summary: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("run_id", "scenario", "name", "branch", "estimated_total_seconds", "eta_known", "dirty_warning", "coalesced")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_TOTAL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ETA_KNOWN_FIELD_NUMBER: _ClassVar[int]
    DIRTY_WARNING_FIELD_NUMBER: _ClassVar[int]
    COALESCED_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    scenario: str
    name: str
    branch: str
    estimated_total_seconds: int
    eta_known: bool
    dirty_warning: str
    coalesced: bool
    def __init__(self, run_id: _Optional[str] = ..., scenario: _Optional[str] = ..., name: _Optional[str] = ..., branch: _Optional[str] = ..., estimated_total_seconds: _Optional[int] = ..., eta_known: _Optional[bool] = ..., dirty_warning: _Optional[str] = ..., coalesced: _Optional[bool] = ...) -> None: ...

class GetSnapshotStatusRequest(_message.Message):
    __slots__ = ("scenario", "name", "branch", "run_id", "repo_id", "wait")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    WAIT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    name: str
    branch: str
    run_id: str
    repo_id: int
    wait: bool
    def __init__(self, scenario: _Optional[str] = ..., name: _Optional[str] = ..., branch: _Optional[str] = ..., run_id: _Optional[str] = ..., repo_id: _Optional[int] = ..., wait: _Optional[bool] = ...) -> None: ...

class GetSnapshotStatusResponse(_message.Message):
    __slots__ = ("status", "scenario", "name", "branch", "run_id", "run_status", "baseline", "error", "similar_baselines", "recommended_next_check_seconds")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_STATUS_FIELD_NUMBER: _ClassVar[int]
    BASELINE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    SIMILAR_BASELINES_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_NEXT_CHECK_SECONDS_FIELD_NUMBER: _ClassVar[int]
    status: str
    scenario: str
    name: str
    branch: str
    run_id: str
    run_status: str
    baseline: BaselineManifest
    error: str
    similar_baselines: _containers.RepeatedScalarFieldContainer[str]
    recommended_next_check_seconds: int
    def __init__(self, status: _Optional[str] = ..., scenario: _Optional[str] = ..., name: _Optional[str] = ..., branch: _Optional[str] = ..., run_id: _Optional[str] = ..., run_status: _Optional[str] = ..., baseline: _Optional[_Union[BaselineManifest, _Mapping]] = ..., error: _Optional[str] = ..., similar_baselines: _Optional[_Iterable[str]] = ..., recommended_next_check_seconds: _Optional[int] = ...) -> None: ...

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

class StartDiffRequest(_message.Message):
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

class StartDiffResponse(_message.Message):
    __slots__ = ("run_id", "scenario", "name", "branch", "estimated_total_seconds", "eta_known", "coalesced", "reused_run", "reused_sha", "dirty_warning")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_TOTAL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ETA_KNOWN_FIELD_NUMBER: _ClassVar[int]
    COALESCED_FIELD_NUMBER: _ClassVar[int]
    REUSED_RUN_FIELD_NUMBER: _ClassVar[int]
    REUSED_SHA_FIELD_NUMBER: _ClassVar[int]
    DIRTY_WARNING_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    scenario: str
    name: str
    branch: str
    estimated_total_seconds: int
    eta_known: bool
    coalesced: bool
    reused_run: bool
    reused_sha: str
    dirty_warning: str
    def __init__(self, run_id: _Optional[str] = ..., scenario: _Optional[str] = ..., name: _Optional[str] = ..., branch: _Optional[str] = ..., estimated_total_seconds: _Optional[int] = ..., eta_known: _Optional[bool] = ..., coalesced: _Optional[bool] = ..., reused_run: _Optional[bool] = ..., reused_sha: _Optional[str] = ..., dirty_warning: _Optional[str] = ...) -> None: ...

class GetDiffResultRequest(_message.Message):
    __slots__ = ("scenario", "name", "branch", "run_id", "surface", "repo_id", "wait", "latest")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    REPO_ID_FIELD_NUMBER: _ClassVar[int]
    WAIT_FIELD_NUMBER: _ClassVar[int]
    LATEST_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    name: str
    branch: str
    run_id: str
    surface: str
    repo_id: int
    wait: bool
    latest: bool
    def __init__(self, scenario: _Optional[str] = ..., name: _Optional[str] = ..., branch: _Optional[str] = ..., run_id: _Optional[str] = ..., surface: _Optional[str] = ..., repo_id: _Optional[int] = ..., wait: _Optional[bool] = ..., latest: _Optional[bool] = ...) -> None: ...

class GetDiffResultResponse(_message.Message):
    __slots__ = ("status", "diff", "error", "recommended_next_check_seconds", "run_id")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DIFF_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_NEXT_CHECK_SECONDS_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    status: str
    diff: DiffResult
    error: str
    recommended_next_check_seconds: int
    run_id: str
    def __init__(self, status: _Optional[str] = ..., diff: _Optional[_Union[DiffResult, _Mapping]] = ..., error: _Optional[str] = ..., recommended_next_check_seconds: _Optional[int] = ..., run_id: _Optional[str] = ...) -> None: ...

class DiffResult(_message.Message):
    __slots__ = ("baseline", "current_git", "staleness", "surfaces", "verdict", "dirty_warning", "phases")
    BASELINE_FIELD_NUMBER: _ClassVar[int]
    CURRENT_GIT_FIELD_NUMBER: _ClassVar[int]
    STALENESS_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    DIRTY_WARNING_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    baseline: BaselineManifest
    current_git: GitState
    staleness: Staleness
    surfaces: _containers.RepeatedCompositeFieldContainer[SurfaceDiff]
    verdict: str
    dirty_warning: str
    phases: _containers.RepeatedCompositeFieldContainer[PhaseDiff]
    def __init__(self, baseline: _Optional[_Union[BaselineManifest, _Mapping]] = ..., current_git: _Optional[_Union[GitState, _Mapping]] = ..., staleness: _Optional[_Union[Staleness, _Mapping]] = ..., surfaces: _Optional[_Iterable[_Union[SurfaceDiff, _Mapping]]] = ..., verdict: _Optional[str] = ..., dirty_warning: _Optional[str] = ..., phases: _Optional[_Iterable[_Union[PhaseDiff, _Mapping]]] = ...) -> None: ...

class RunBusyInfo(_message.Message):
    __slots__ = ("scenario", "run_id", "preset")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    PRESET_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    preset: str
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ..., preset: _Optional[str] = ...) -> None: ...

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
