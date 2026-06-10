from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DiagnosticsInfo(_message.Message):
    __slots__ = ("video", "console", "network", "har", "trace", "dom")
    VIDEO_FIELD_NUMBER: _ClassVar[int]
    CONSOLE_FIELD_NUMBER: _ClassVar[int]
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    HAR_FIELD_NUMBER: _ClassVar[int]
    TRACE_FIELD_NUMBER: _ClassVar[int]
    DOM_FIELD_NUMBER: _ClassVar[int]
    video: bool
    console: bool
    network: bool
    har: bool
    trace: bool
    dom: bool
    def __init__(self, video: _Optional[bool] = ..., console: _Optional[bool] = ..., network: _Optional[bool] = ..., har: _Optional[bool] = ..., trace: _Optional[bool] = ..., dom: _Optional[bool] = ...) -> None: ...

class PinInfo(_message.Message):
    __slots__ = ("pinned_by", "pinned_at", "reason")
    PINNED_BY_FIELD_NUMBER: _ClassVar[int]
    PINNED_AT_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    pinned_by: str
    pinned_at: str
    reason: str
    def __init__(self, pinned_by: _Optional[str] = ..., pinned_at: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class PhaseInfo(_message.Message):
    __slots__ = ("name", "status", "duration_seconds")
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    name: str
    status: str
    duration_seconds: float
    def __init__(self, name: _Optional[str] = ..., status: _Optional[str] = ..., duration_seconds: _Optional[float] = ...) -> None: ...

class RunInfo(_message.Message):
    __slots__ = ("run_id", "scenario", "started_at", "completed_at", "status", "phases", "git_sha", "git_branch", "git_dirty", "git_dirty_summary", "diagnostics", "pins", "tree_digest")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    GIT_SHA_FIELD_NUMBER: _ClassVar[int]
    GIT_BRANCH_FIELD_NUMBER: _ClassVar[int]
    GIT_DIRTY_FIELD_NUMBER: _ClassVar[int]
    GIT_DIRTY_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTICS_FIELD_NUMBER: _ClassVar[int]
    PINS_FIELD_NUMBER: _ClassVar[int]
    TREE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    scenario: str
    started_at: str
    completed_at: str
    status: str
    phases: _containers.RepeatedCompositeFieldContainer[PhaseInfo]
    git_sha: str
    git_branch: str
    git_dirty: bool
    git_dirty_summary: str
    diagnostics: DiagnosticsInfo
    pins: _containers.RepeatedCompositeFieldContainer[PinInfo]
    tree_digest: str
    def __init__(self, run_id: _Optional[str] = ..., scenario: _Optional[str] = ..., started_at: _Optional[str] = ..., completed_at: _Optional[str] = ..., status: _Optional[str] = ..., phases: _Optional[_Iterable[_Union[PhaseInfo, _Mapping]]] = ..., git_sha: _Optional[str] = ..., git_branch: _Optional[str] = ..., git_dirty: _Optional[bool] = ..., git_dirty_summary: _Optional[str] = ..., diagnostics: _Optional[_Union[DiagnosticsInfo, _Mapping]] = ..., pins: _Optional[_Iterable[_Union[PinInfo, _Mapping]]] = ..., tree_digest: _Optional[str] = ...) -> None: ...

class ListRunsRequest(_message.Message):
    __slots__ = ("scenario", "status", "limit")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    status: str
    limit: int
    def __init__(self, scenario: _Optional[str] = ..., status: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListRunsResponse(_message.Message):
    __slots__ = ("runs",)
    RUNS_FIELD_NUMBER: _ClassVar[int]
    runs: _containers.RepeatedCompositeFieldContainer[RunInfo]
    def __init__(self, runs: _Optional[_Iterable[_Union[RunInfo, _Mapping]]] = ...) -> None: ...

class GetRunRequest(_message.Message):
    __slots__ = ("scenario", "run_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class GetRunResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: RunInfo
    def __init__(self, run: _Optional[_Union[RunInfo, _Mapping]] = ...) -> None: ...

class DeleteRunRequest(_message.Message):
    __slots__ = ("scenario", "run_id", "force")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    FORCE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    force: bool
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ..., force: _Optional[bool] = ...) -> None: ...

class DeleteRunResponse(_message.Message):
    __slots__ = ("deleted",)
    DELETED_FIELD_NUMBER: _ClassVar[int]
    deleted: bool
    def __init__(self, deleted: _Optional[bool] = ...) -> None: ...

class PinRunRequest(_message.Message):
    __slots__ = ("scenario", "run_id", "pinned_by", "reason")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    PINNED_BY_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    pinned_by: str
    reason: str
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ..., pinned_by: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class PinRunResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: RunInfo
    def __init__(self, run: _Optional[_Union[RunInfo, _Mapping]] = ...) -> None: ...

class UnpinRunRequest(_message.Message):
    __slots__ = ("scenario", "run_id", "pinned_by")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    PINNED_BY_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    pinned_by: str
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ..., pinned_by: _Optional[str] = ...) -> None: ...

class UnpinRunResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: RunInfo
    def __init__(self, run: _Optional[_Union[RunInfo, _Mapping]] = ...) -> None: ...

class CompareRunsRequest(_message.Message):
    __slots__ = ("scenario", "run_id_a", "run_id_b", "phase")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_A_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_B_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id_a: str
    run_id_b: str
    phase: str
    def __init__(self, scenario: _Optional[str] = ..., run_id_a: _Optional[str] = ..., run_id_b: _Optional[str] = ..., phase: _Optional[str] = ...) -> None: ...

class PhaseDiff(_message.Message):
    __slots__ = ("phase", "status_a", "status_b", "verdict", "regressions", "new_failures", "preexisting_failures", "cleared_failures")
    PHASE_FIELD_NUMBER: _ClassVar[int]
    STATUS_A_FIELD_NUMBER: _ClassVar[int]
    STATUS_B_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    REGRESSIONS_FIELD_NUMBER: _ClassVar[int]
    NEW_FAILURES_FIELD_NUMBER: _ClassVar[int]
    PREEXISTING_FAILURES_FIELD_NUMBER: _ClassVar[int]
    CLEARED_FAILURES_FIELD_NUMBER: _ClassVar[int]
    phase: str
    status_a: str
    status_b: str
    verdict: str
    regressions: _containers.RepeatedScalarFieldContainer[str]
    new_failures: _containers.RepeatedScalarFieldContainer[str]
    preexisting_failures: _containers.RepeatedScalarFieldContainer[str]
    cleared_failures: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, phase: _Optional[str] = ..., status_a: _Optional[str] = ..., status_b: _Optional[str] = ..., verdict: _Optional[str] = ..., regressions: _Optional[_Iterable[str]] = ..., new_failures: _Optional[_Iterable[str]] = ..., preexisting_failures: _Optional[_Iterable[str]] = ..., cleared_failures: _Optional[_Iterable[str]] = ...) -> None: ...

class CompareRunsResponse(_message.Message):
    __slots__ = ("phases", "verdict")
    PHASES_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    phases: _containers.RepeatedCompositeFieldContainer[PhaseDiff]
    verdict: str
    def __init__(self, phases: _Optional[_Iterable[_Union[PhaseDiff, _Mapping]]] = ..., verdict: _Optional[str] = ...) -> None: ...

class GetPhaseArtifactRequest(_message.Message):
    __slots__ = ("scenario", "run_id", "phase")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    phase: str
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ..., phase: _Optional[str] = ...) -> None: ...

class GetPhaseArtifactResponse(_message.Message):
    __slots__ = ("content", "content_type")
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    content: str
    content_type: str
    def __init__(self, content: _Optional[str] = ..., content_type: _Optional[str] = ...) -> None: ...

class RunVideo(_message.Message):
    __slots__ = ("workflow", "rel_path", "size_bytes")
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    REL_PATH_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    workflow: str
    rel_path: str
    size_bytes: int
    def __init__(self, workflow: _Optional[str] = ..., rel_path: _Optional[str] = ..., size_bytes: _Optional[int] = ...) -> None: ...

class ListRunVideosRequest(_message.Message):
    __slots__ = ("scenario", "run_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class ListRunVideosResponse(_message.Message):
    __slots__ = ("videos",)
    VIDEOS_FIELD_NUMBER: _ClassVar[int]
    videos: _containers.RepeatedCompositeFieldContainer[RunVideo]
    def __init__(self, videos: _Optional[_Iterable[_Union[RunVideo, _Mapping]]] = ...) -> None: ...

class CheckFreshnessRequest(_message.Message):
    __slots__ = ("scenario", "phases")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    phases: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., phases: _Optional[_Iterable[str]] = ...) -> None: ...

class PhaseFreshness(_message.Message):
    __slots__ = ("phase", "status", "last_run_id", "last_run_completed_at")
    PHASE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LAST_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    LAST_RUN_COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    phase: str
    status: str
    last_run_id: str
    last_run_completed_at: str
    def __init__(self, phase: _Optional[str] = ..., status: _Optional[str] = ..., last_run_id: _Optional[str] = ..., last_run_completed_at: _Optional[str] = ...) -> None: ...

class CheckFreshnessResponse(_message.Message):
    __slots__ = ("scenario", "tree_digest", "phases", "suggested_command")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TREE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    SUGGESTED_COMMAND_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    tree_digest: str
    phases: _containers.RepeatedCompositeFieldContainer[PhaseFreshness]
    suggested_command: str
    def __init__(self, scenario: _Optional[str] = ..., tree_digest: _Optional[str] = ..., phases: _Optional[_Iterable[_Union[PhaseFreshness, _Mapping]]] = ..., suggested_command: _Optional[str] = ...) -> None: ...
