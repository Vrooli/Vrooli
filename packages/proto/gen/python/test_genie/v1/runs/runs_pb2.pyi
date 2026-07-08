from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RunEvent(_message.Message):
    __slots__ = ("event", "elapsed_seconds", "run_id", "scenario", "artifact_dir", "preset", "phase", "phase_index", "phase_total", "status", "duration_seconds", "quiet_seconds", "message", "success", "verdict", "error", "maturity_standing", "findings_summary")
    EVENT_FIELD_NUMBER: _ClassVar[int]
    ELAPSED_SECONDS_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DIR_FIELD_NUMBER: _ClassVar[int]
    PRESET_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    PHASE_INDEX_FIELD_NUMBER: _ClassVar[int]
    PHASE_TOTAL_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    QUIET_SECONDS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    MATURITY_STANDING_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    event: str
    elapsed_seconds: float
    run_id: str
    scenario: str
    artifact_dir: str
    preset: str
    phase: str
    phase_index: int
    phase_total: int
    status: str
    duration_seconds: int
    quiet_seconds: float
    message: str
    success: bool
    verdict: str
    error: str
    maturity_standing: PhaseMaturityStanding
    findings_summary: PhaseFindingsSummary
    def __init__(self, event: _Optional[str] = ..., elapsed_seconds: _Optional[float] = ..., run_id: _Optional[str] = ..., scenario: _Optional[str] = ..., artifact_dir: _Optional[str] = ..., preset: _Optional[str] = ..., phase: _Optional[str] = ..., phase_index: _Optional[int] = ..., phase_total: _Optional[int] = ..., status: _Optional[str] = ..., duration_seconds: _Optional[int] = ..., quiet_seconds: _Optional[float] = ..., message: _Optional[str] = ..., success: _Optional[bool] = ..., verdict: _Optional[str] = ..., error: _Optional[str] = ..., maturity_standing: _Optional[_Union[PhaseMaturityStanding, _Mapping]] = ..., findings_summary: _Optional[_Union[PhaseFindingsSummary, _Mapping]] = ...) -> None: ...

class RunLiveStatus(_message.Message):
    __slots__ = ("run_id", "scenario", "status", "active_phase", "phase_index", "phase_total", "started_at", "elapsed_seconds", "estimated_total_seconds", "estimated_remaining_seconds", "eta_known", "recommended_next_check_seconds", "verdict", "success", "error", "active", "terminal_standings", "terminal_findings_summaries")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_PHASE_FIELD_NUMBER: _ClassVar[int]
    PHASE_INDEX_FIELD_NUMBER: _ClassVar[int]
    PHASE_TOTAL_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    ELAPSED_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_TOTAL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_REMAINING_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ETA_KNOWN_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_NEXT_CHECK_SECONDS_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_STANDINGS_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_FINDINGS_SUMMARIES_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    scenario: str
    status: str
    active_phase: str
    phase_index: int
    phase_total: int
    started_at: str
    elapsed_seconds: float
    estimated_total_seconds: int
    estimated_remaining_seconds: int
    eta_known: bool
    recommended_next_check_seconds: int
    verdict: str
    success: bool
    error: str
    active: bool
    terminal_standings: _containers.RepeatedCompositeFieldContainer[PhaseMaturityStanding]
    terminal_findings_summaries: _containers.RepeatedCompositeFieldContainer[PhaseFindingsSummary]
    def __init__(self, run_id: _Optional[str] = ..., scenario: _Optional[str] = ..., status: _Optional[str] = ..., active_phase: _Optional[str] = ..., phase_index: _Optional[int] = ..., phase_total: _Optional[int] = ..., started_at: _Optional[str] = ..., elapsed_seconds: _Optional[float] = ..., estimated_total_seconds: _Optional[int] = ..., estimated_remaining_seconds: _Optional[int] = ..., eta_known: _Optional[bool] = ..., recommended_next_check_seconds: _Optional[int] = ..., verdict: _Optional[str] = ..., success: _Optional[bool] = ..., error: _Optional[str] = ..., active: _Optional[bool] = ..., terminal_standings: _Optional[_Iterable[_Union[PhaseMaturityStanding, _Mapping]]] = ..., terminal_findings_summaries: _Optional[_Iterable[_Union[PhaseFindingsSummary, _Mapping]]] = ...) -> None: ...

class StartRunRequest(_message.Message):
    __slots__ = ("scenario", "preset", "phases", "skip", "fail_fast", "diagnostics_preset", "ui_url", "api_url", "scenario_path", "logical_repo_root", "logical_scenario_rel_path", "suite_request_id", "capture_profile")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PRESET_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    SKIP_FIELD_NUMBER: _ClassVar[int]
    FAIL_FAST_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTICS_PRESET_FIELD_NUMBER: _ClassVar[int]
    UI_URL_FIELD_NUMBER: _ClassVar[int]
    API_URL_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_PATH_FIELD_NUMBER: _ClassVar[int]
    LOGICAL_REPO_ROOT_FIELD_NUMBER: _ClassVar[int]
    LOGICAL_SCENARIO_REL_PATH_FIELD_NUMBER: _ClassVar[int]
    SUITE_REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_PROFILE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    preset: str
    phases: _containers.RepeatedScalarFieldContainer[str]
    skip: _containers.RepeatedScalarFieldContainer[str]
    fail_fast: bool
    diagnostics_preset: str
    ui_url: str
    api_url: str
    scenario_path: str
    logical_repo_root: str
    logical_scenario_rel_path: str
    suite_request_id: str
    capture_profile: str
    def __init__(self, scenario: _Optional[str] = ..., preset: _Optional[str] = ..., phases: _Optional[_Iterable[str]] = ..., skip: _Optional[_Iterable[str]] = ..., fail_fast: _Optional[bool] = ..., diagnostics_preset: _Optional[str] = ..., ui_url: _Optional[str] = ..., api_url: _Optional[str] = ..., scenario_path: _Optional[str] = ..., logical_repo_root: _Optional[str] = ..., logical_scenario_rel_path: _Optional[str] = ..., suite_request_id: _Optional[str] = ..., capture_profile: _Optional[str] = ...) -> None: ...

class StartRunResponse(_message.Message):
    __slots__ = ("run_id", "scenario", "estimated_total_seconds", "eta_known", "coalesced")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_TOTAL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ETA_KNOWN_FIELD_NUMBER: _ClassVar[int]
    COALESCED_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    scenario: str
    estimated_total_seconds: int
    eta_known: bool
    coalesced: bool
    def __init__(self, run_id: _Optional[str] = ..., scenario: _Optional[str] = ..., estimated_total_seconds: _Optional[int] = ..., eta_known: _Optional[bool] = ..., coalesced: _Optional[bool] = ...) -> None: ...

class RunBusyInfo(_message.Message):
    __slots__ = ("scenario", "run_id", "preset")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    PRESET_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    preset: str
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ..., preset: _Optional[str] = ...) -> None: ...

class FollowRunRequest(_message.Message):
    __slots__ = ("scenario", "run_id", "suppress_heartbeats")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SUPPRESS_HEARTBEATS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    suppress_heartbeats: bool
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ..., suppress_heartbeats: _Optional[bool] = ...) -> None: ...

class WaitRunRequest(_message.Message):
    __slots__ = ("scenario", "run_id", "timeout_seconds")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    timeout_seconds: int
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class WaitRunResponse(_message.Message):
    __slots__ = ("status", "timed_out")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TIMED_OUT_FIELD_NUMBER: _ClassVar[int]
    status: RunLiveStatus
    timed_out: bool
    def __init__(self, status: _Optional[_Union[RunLiveStatus, _Mapping]] = ..., timed_out: _Optional[bool] = ...) -> None: ...

class AbortRunRequest(_message.Message):
    __slots__ = ("scenario", "run_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class AbortRunResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: RunLiveStatus
    def __init__(self, status: _Optional[_Union[RunLiveStatus, _Mapping]] = ...) -> None: ...

class GetRunStatusRequest(_message.Message):
    __slots__ = ("scenario", "run_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

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

class PhaseFindingsSummary(_message.Message):
    __slots__ = ("blockers", "errors", "warnings", "infos", "total")
    BLOCKERS_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    INFOS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    blockers: int
    errors: int
    warnings: int
    infos: int
    total: int
    def __init__(self, blockers: _Optional[int] = ..., errors: _Optional[int] = ..., warnings: _Optional[int] = ..., infos: _Optional[int] = ..., total: _Optional[int] = ...) -> None: ...

class PhaseCapabilityStanding(_message.Message):
    __slots__ = ("id", "label", "current_level", "current_level_label", "next_level", "current_summary", "next_unlock", "clean", "blocking_finding_count", "blocking_finding_codes", "priority_rank", "priority_reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    CURRENT_LEVEL_FIELD_NUMBER: _ClassVar[int]
    CURRENT_LEVEL_LABEL_FIELD_NUMBER: _ClassVar[int]
    NEXT_LEVEL_FIELD_NUMBER: _ClassVar[int]
    CURRENT_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    NEXT_UNLOCK_FIELD_NUMBER: _ClassVar[int]
    CLEAN_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_FINDING_COUNT_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_FINDING_CODES_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_RANK_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    current_level: str
    current_level_label: str
    next_level: str
    current_summary: str
    next_unlock: str
    clean: bool
    blocking_finding_count: int
    blocking_finding_codes: _containers.RepeatedScalarFieldContainer[str]
    priority_rank: int
    priority_reason: str
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., current_level: _Optional[str] = ..., current_level_label: _Optional[str] = ..., next_level: _Optional[str] = ..., current_summary: _Optional[str] = ..., next_unlock: _Optional[str] = ..., clean: _Optional[bool] = ..., blocking_finding_count: _Optional[int] = ..., blocking_finding_codes: _Optional[_Iterable[str]] = ..., priority_rank: _Optional[int] = ..., priority_reason: _Optional[str] = ...) -> None: ...

class PhaseMaturityStanding(_message.Message):
    __slots__ = ("provider", "phase", "current_level", "current_level_label", "next_level", "ceiling_level", "clean", "unknown_count", "blocking_finding_codes", "next_move", "next_move_reason", "priority_capability_id", "priority_capability_label", "north_star", "at_maximum", "capabilities")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    CURRENT_LEVEL_FIELD_NUMBER: _ClassVar[int]
    CURRENT_LEVEL_LABEL_FIELD_NUMBER: _ClassVar[int]
    NEXT_LEVEL_FIELD_NUMBER: _ClassVar[int]
    CEILING_LEVEL_FIELD_NUMBER: _ClassVar[int]
    CLEAN_FIELD_NUMBER: _ClassVar[int]
    UNKNOWN_COUNT_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_FINDING_CODES_FIELD_NUMBER: _ClassVar[int]
    NEXT_MOVE_FIELD_NUMBER: _ClassVar[int]
    NEXT_MOVE_REASON_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_CAPABILITY_LABEL_FIELD_NUMBER: _ClassVar[int]
    NORTH_STAR_FIELD_NUMBER: _ClassVar[int]
    AT_MAXIMUM_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    provider: str
    phase: str
    current_level: str
    current_level_label: str
    next_level: str
    ceiling_level: str
    clean: bool
    unknown_count: int
    blocking_finding_codes: _containers.RepeatedScalarFieldContainer[str]
    next_move: str
    next_move_reason: str
    priority_capability_id: str
    priority_capability_label: str
    north_star: str
    at_maximum: bool
    capabilities: _containers.RepeatedCompositeFieldContainer[PhaseCapabilityStanding]
    def __init__(self, provider: _Optional[str] = ..., phase: _Optional[str] = ..., current_level: _Optional[str] = ..., current_level_label: _Optional[str] = ..., next_level: _Optional[str] = ..., ceiling_level: _Optional[str] = ..., clean: _Optional[bool] = ..., unknown_count: _Optional[int] = ..., blocking_finding_codes: _Optional[_Iterable[str]] = ..., next_move: _Optional[str] = ..., next_move_reason: _Optional[str] = ..., priority_capability_id: _Optional[str] = ..., priority_capability_label: _Optional[str] = ..., north_star: _Optional[str] = ..., at_maximum: _Optional[bool] = ..., capabilities: _Optional[_Iterable[_Union[PhaseCapabilityStanding, _Mapping]]] = ...) -> None: ...

class PhaseInfo(_message.Message):
    __slots__ = ("name", "status", "duration_seconds", "comparable", "advisory", "artifact_backed", "non_comparable", "maturity_standing", "findings_summary")
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    COMPARABLE_FIELD_NUMBER: _ClassVar[int]
    ADVISORY_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_BACKED_FIELD_NUMBER: _ClassVar[int]
    NON_COMPARABLE_FIELD_NUMBER: _ClassVar[int]
    MATURITY_STANDING_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    name: str
    status: str
    duration_seconds: float
    comparable: bool
    advisory: bool
    artifact_backed: bool
    non_comparable: bool
    maturity_standing: PhaseMaturityStanding
    findings_summary: PhaseFindingsSummary
    def __init__(self, name: _Optional[str] = ..., status: _Optional[str] = ..., duration_seconds: _Optional[float] = ..., comparable: _Optional[bool] = ..., advisory: _Optional[bool] = ..., artifact_backed: _Optional[bool] = ..., non_comparable: _Optional[bool] = ..., maturity_standing: _Optional[_Union[PhaseMaturityStanding, _Mapping]] = ..., findings_summary: _Optional[_Union[PhaseFindingsSummary, _Mapping]] = ...) -> None: ...

class RunInfo(_message.Message):
    __slots__ = ("run_id", "scenario", "started_at", "completed_at", "status", "phases", "git_sha", "git_branch", "git_dirty", "git_dirty_summary", "diagnostics", "pins", "tree_digest", "preset", "capture_profile", "planned_phases", "phase_set_digest")
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
    PRESET_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_PROFILE_FIELD_NUMBER: _ClassVar[int]
    PLANNED_PHASES_FIELD_NUMBER: _ClassVar[int]
    PHASE_SET_DIGEST_FIELD_NUMBER: _ClassVar[int]
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
    preset: str
    capture_profile: str
    planned_phases: _containers.RepeatedScalarFieldContainer[str]
    phase_set_digest: str
    def __init__(self, run_id: _Optional[str] = ..., scenario: _Optional[str] = ..., started_at: _Optional[str] = ..., completed_at: _Optional[str] = ..., status: _Optional[str] = ..., phases: _Optional[_Iterable[_Union[PhaseInfo, _Mapping]]] = ..., git_sha: _Optional[str] = ..., git_branch: _Optional[str] = ..., git_dirty: _Optional[bool] = ..., git_dirty_summary: _Optional[str] = ..., diagnostics: _Optional[_Union[DiagnosticsInfo, _Mapping]] = ..., pins: _Optional[_Iterable[_Union[PinInfo, _Mapping]]] = ..., tree_digest: _Optional[str] = ..., preset: _Optional[str] = ..., capture_profile: _Optional[str] = ..., planned_phases: _Optional[_Iterable[str]] = ..., phase_set_digest: _Optional[str] = ...) -> None: ...

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

class FindRunRequest(_message.Message):
    __slots__ = ("scenario", "git_sha", "tree_digest", "preset", "capture_profile", "status", "require_clean", "phase_set_digest")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    GIT_SHA_FIELD_NUMBER: _ClassVar[int]
    TREE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    PRESET_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_PROFILE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REQUIRE_CLEAN_FIELD_NUMBER: _ClassVar[int]
    PHASE_SET_DIGEST_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    git_sha: str
    tree_digest: str
    preset: str
    capture_profile: str
    status: str
    require_clean: bool
    phase_set_digest: str
    def __init__(self, scenario: _Optional[str] = ..., git_sha: _Optional[str] = ..., tree_digest: _Optional[str] = ..., preset: _Optional[str] = ..., capture_profile: _Optional[str] = ..., status: _Optional[str] = ..., require_clean: _Optional[bool] = ..., phase_set_digest: _Optional[str] = ...) -> None: ...

class FindRunResponse(_message.Message):
    __slots__ = ("found", "run")
    FOUND_FIELD_NUMBER: _ClassVar[int]
    RUN_FIELD_NUMBER: _ClassVar[int]
    found: bool
    run: RunInfo
    def __init__(self, found: _Optional[bool] = ..., run: _Optional[_Union[RunInfo, _Mapping]] = ...) -> None: ...

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

class GetRunFindingsRequest(_message.Message):
    __slots__ = ("scenario", "run_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class RunFindingsPhase(_message.Message):
    __slots__ = ("name", "status", "finding_source", "maturity_standing", "findings_summary")
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    FINDING_SOURCE_FIELD_NUMBER: _ClassVar[int]
    MATURITY_STANDING_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    name: str
    status: str
    finding_source: str
    maturity_standing: PhaseMaturityStanding
    findings_summary: PhaseFindingsSummary
    def __init__(self, name: _Optional[str] = ..., status: _Optional[str] = ..., finding_source: _Optional[str] = ..., maturity_standing: _Optional[_Union[PhaseMaturityStanding, _Mapping]] = ..., findings_summary: _Optional[_Union[PhaseFindingsSummary, _Mapping]] = ...) -> None: ...

class GetRunFindingsResponse(_message.Message):
    __slots__ = ("scenario", "run_id", "verdict", "completed_at", "phases")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    verdict: str
    completed_at: str
    phases: _containers.RepeatedCompositeFieldContainer[RunFindingsPhase]
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ..., verdict: _Optional[str] = ..., completed_at: _Optional[str] = ..., phases: _Optional[_Iterable[_Union[RunFindingsPhase, _Mapping]]] = ...) -> None: ...

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

class RunVisual(_message.Message):
    __slots__ = ("page", "label", "screenshot_rel_path", "video_rel_path", "screenshot_size_bytes")
    PAGE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    SCREENSHOT_REL_PATH_FIELD_NUMBER: _ClassVar[int]
    VIDEO_REL_PATH_FIELD_NUMBER: _ClassVar[int]
    SCREENSHOT_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    page: str
    label: str
    screenshot_rel_path: str
    video_rel_path: str
    screenshot_size_bytes: int
    def __init__(self, page: _Optional[str] = ..., label: _Optional[str] = ..., screenshot_rel_path: _Optional[str] = ..., video_rel_path: _Optional[str] = ..., screenshot_size_bytes: _Optional[int] = ...) -> None: ...

class ListRunVisualsRequest(_message.Message):
    __slots__ = ("scenario", "run_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class ListRunVisualsResponse(_message.Message):
    __slots__ = ("visuals",)
    VISUALS_FIELD_NUMBER: _ClassVar[int]
    visuals: _containers.RepeatedCompositeFieldContainer[RunVisual]
    def __init__(self, visuals: _Optional[_Iterable[_Union[RunVisual, _Mapping]]] = ...) -> None: ...

class CompareRunVisualsRequest(_message.Message):
    __slots__ = ("scenario", "base_run_id", "current_run_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    BASE_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    CURRENT_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    base_run_id: str
    current_run_id: str
    def __init__(self, scenario: _Optional[str] = ..., base_run_id: _Optional[str] = ..., current_run_id: _Optional[str] = ...) -> None: ...

class VisualDelta(_message.Message):
    __slots__ = ("page", "label", "status", "changed_fraction")
    PAGE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CHANGED_FRACTION_FIELD_NUMBER: _ClassVar[int]
    page: str
    label: str
    status: str
    changed_fraction: float
    def __init__(self, page: _Optional[str] = ..., label: _Optional[str] = ..., status: _Optional[str] = ..., changed_fraction: _Optional[float] = ...) -> None: ...

class CompareRunVisualsResponse(_message.Message):
    __slots__ = ("deltas",)
    DELTAS_FIELD_NUMBER: _ClassVar[int]
    deltas: _containers.RepeatedCompositeFieldContainer[VisualDelta]
    def __init__(self, deltas: _Optional[_Iterable[_Union[VisualDelta, _Mapping]]] = ...) -> None: ...

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

class GetSelfHealthRequest(_message.Message):
    __slots__ = ("window_days", "skip_conformance", "include_trend")
    WINDOW_DAYS_FIELD_NUMBER: _ClassVar[int]
    SKIP_CONFORMANCE_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_TREND_FIELD_NUMBER: _ClassVar[int]
    window_days: int
    skip_conformance: bool
    include_trend: bool
    def __init__(self, window_days: _Optional[int] = ..., skip_conformance: _Optional[bool] = ..., include_trend: _Optional[bool] = ...) -> None: ...

class GetSelfHealthResponse(_message.Message):
    __slots__ = ("self_health",)
    SELF_HEALTH_FIELD_NUMBER: _ClassVar[int]
    self_health: SelfHealth
    def __init__(self, self_health: _Optional[_Union[SelfHealth, _Mapping]] = ...) -> None: ...

class GetFleetHealthRequest(_message.Message):
    __slots__ = ("window_days", "include_roster")
    WINDOW_DAYS_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_ROSTER_FIELD_NUMBER: _ClassVar[int]
    window_days: int
    include_roster: bool
    def __init__(self, window_days: _Optional[int] = ..., include_roster: _Optional[bool] = ...) -> None: ...

class GetFleetHealthResponse(_message.Message):
    __slots__ = ("fleet_health",)
    FLEET_HEALTH_FIELD_NUMBER: _ClassVar[int]
    fleet_health: FleetHealth
    def __init__(self, fleet_health: _Optional[_Union[FleetHealth, _Mapping]] = ...) -> None: ...

class FleetHealth(_message.Message):
    __slots__ = ("window_days", "captured_at", "scenarios_tested", "scenarios_total", "total_runs", "total_issues", "scenarios", "top_finding_sources", "never_tested_in_window")
    WINDOW_DAYS_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_TESTED_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_TOTAL_FIELD_NUMBER: _ClassVar[int]
    TOTAL_RUNS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_ISSUES_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    TOP_FINDING_SOURCES_FIELD_NUMBER: _ClassVar[int]
    NEVER_TESTED_IN_WINDOW_FIELD_NUMBER: _ClassVar[int]
    window_days: int
    captured_at: str
    scenarios_tested: int
    scenarios_total: int
    total_runs: int
    total_issues: int
    scenarios: _containers.RepeatedCompositeFieldContainer[FleetScenarioHealth]
    top_finding_sources: _containers.RepeatedCompositeFieldContainer[FleetFindingSource]
    never_tested_in_window: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, window_days: _Optional[int] = ..., captured_at: _Optional[str] = ..., scenarios_tested: _Optional[int] = ..., scenarios_total: _Optional[int] = ..., total_runs: _Optional[int] = ..., total_issues: _Optional[int] = ..., scenarios: _Optional[_Iterable[_Union[FleetScenarioHealth, _Mapping]]] = ..., top_finding_sources: _Optional[_Iterable[_Union[FleetFindingSource, _Mapping]]] = ..., never_tested_in_window: _Optional[_Iterable[str]] = ...) -> None: ...

class FleetScenarioHealth(_message.Message):
    __slots__ = ("scenario", "runs", "passed_runs", "failed_runs", "availability", "failure_rate", "issues", "last_run_at", "last_outcome", "age_days")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUNS_FIELD_NUMBER: _ClassVar[int]
    PASSED_RUNS_FIELD_NUMBER: _ClassVar[int]
    FAILED_RUNS_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    FAILURE_RATE_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    LAST_RUN_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_OUTCOME_FIELD_NUMBER: _ClassVar[int]
    AGE_DAYS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    runs: int
    passed_runs: int
    failed_runs: int
    availability: float
    failure_rate: float
    issues: int
    last_run_at: str
    last_outcome: str
    age_days: float
    def __init__(self, scenario: _Optional[str] = ..., runs: _Optional[int] = ..., passed_runs: _Optional[int] = ..., failed_runs: _Optional[int] = ..., availability: _Optional[float] = ..., failure_rate: _Optional[float] = ..., issues: _Optional[int] = ..., last_run_at: _Optional[str] = ..., last_outcome: _Optional[str] = ..., age_days: _Optional[float] = ...) -> None: ...

class FleetFindingSource(_message.Message):
    __slots__ = ("source", "issues")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    source: str
    issues: int
    def __init__(self, source: _Optional[str] = ..., issues: _Optional[int] = ...) -> None: ...

class SelfHealth(_message.Message):
    __slots__ = ("catalog", "conformance", "conformance_freshness", "ledger", "trend_series")
    CATALOG_FIELD_NUMBER: _ClassVar[int]
    CONFORMANCE_FIELD_NUMBER: _ClassVar[int]
    CONFORMANCE_FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    LEDGER_FIELD_NUMBER: _ClassVar[int]
    TREND_SERIES_FIELD_NUMBER: _ClassVar[int]
    catalog: CatalogSummary
    conformance: _containers.RepeatedCompositeFieldContainer[ProviderConformance]
    conformance_freshness: str
    ledger: ReliabilityLedger
    trend_series: _containers.RepeatedCompositeFieldContainer[SelfHealthTrendPoint]
    def __init__(self, catalog: _Optional[_Union[CatalogSummary, _Mapping]] = ..., conformance: _Optional[_Iterable[_Union[ProviderConformance, _Mapping]]] = ..., conformance_freshness: _Optional[str] = ..., ledger: _Optional[_Union[ReliabilityLedger, _Mapping]] = ..., trend_series: _Optional[_Iterable[_Union[SelfHealthTrendPoint, _Mapping]]] = ...) -> None: ...

class SelfHealthTrendPoint(_message.Message):
    __slots__ = ("captured_at", "availability", "run_count", "hard_violations", "metrics_adopted")
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    RUN_COUNT_FIELD_NUMBER: _ClassVar[int]
    HARD_VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    METRICS_ADOPTED_FIELD_NUMBER: _ClassVar[int]
    captured_at: str
    availability: float
    run_count: int
    hard_violations: int
    metrics_adopted: int
    def __init__(self, captured_at: _Optional[str] = ..., availability: _Optional[float] = ..., run_count: _Optional[int] = ..., hard_violations: _Optional[int] = ..., metrics_adopted: _Optional[int] = ...) -> None: ...

class TrendDelta(_message.Message):
    __slots__ = ("previous_captured_at", "previous_availability", "previous_run_count", "availability_delta", "run_count_delta")
    PREVIOUS_CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_RUN_COUNT_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_DELTA_FIELD_NUMBER: _ClassVar[int]
    RUN_COUNT_DELTA_FIELD_NUMBER: _ClassVar[int]
    previous_captured_at: str
    previous_availability: float
    previous_run_count: int
    availability_delta: float
    run_count_delta: int
    def __init__(self, previous_captured_at: _Optional[str] = ..., previous_availability: _Optional[float] = ..., previous_run_count: _Optional[int] = ..., availability_delta: _Optional[float] = ..., run_count_delta: _Optional[int] = ...) -> None: ...

class CatalogSummary(_message.Message):
    __slots__ = ("total_phases", "delegated_phases", "native_phases", "phases")
    TOTAL_PHASES_FIELD_NUMBER: _ClassVar[int]
    DELEGATED_PHASES_FIELD_NUMBER: _ClassVar[int]
    NATIVE_PHASES_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    total_phases: int
    delegated_phases: int
    native_phases: int
    phases: _containers.RepeatedCompositeFieldContainer[CatalogPhase]
    def __init__(self, total_phases: _Optional[int] = ..., delegated_phases: _Optional[int] = ..., native_phases: _Optional[int] = ..., phases: _Optional[_Iterable[_Union[CatalogPhase, _Mapping]]] = ...) -> None: ...

class CatalogPhase(_message.Message):
    __slots__ = ("name", "optional", "source", "delegated", "provider", "finding_source")
    NAME_FIELD_NUMBER: _ClassVar[int]
    OPTIONAL_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    DELEGATED_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    FINDING_SOURCE_FIELD_NUMBER: _ClassVar[int]
    name: str
    optional: bool
    source: str
    delegated: bool
    provider: str
    finding_source: str
    def __init__(self, name: _Optional[str] = ..., optional: _Optional[bool] = ..., source: _Optional[str] = ..., delegated: _Optional[bool] = ..., provider: _Optional[str] = ..., finding_source: _Optional[str] = ...) -> None: ...

class ProviderConformance(_message.Message):
    __slots__ = ("provider", "phase", "reachable", "contract_valid", "identity_ok", "spec_valid", "metrics_adopted", "adoption_score", "violations", "autofix")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    REACHABLE_FIELD_NUMBER: _ClassVar[int]
    CONTRACT_VALID_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_OK_FIELD_NUMBER: _ClassVar[int]
    SPEC_VALID_FIELD_NUMBER: _ClassVar[int]
    METRICS_ADOPTED_FIELD_NUMBER: _ClassVar[int]
    ADOPTION_SCORE_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    AUTOFIX_FIELD_NUMBER: _ClassVar[int]
    provider: str
    phase: str
    reachable: bool
    contract_valid: bool
    identity_ok: bool
    spec_valid: bool
    metrics_adopted: bool
    adoption_score: float
    violations: _containers.RepeatedScalarFieldContainer[str]
    autofix: AutofixCoverage
    def __init__(self, provider: _Optional[str] = ..., phase: _Optional[str] = ..., reachable: _Optional[bool] = ..., contract_valid: _Optional[bool] = ..., identity_ok: _Optional[bool] = ..., spec_valid: _Optional[bool] = ..., metrics_adopted: _Optional[bool] = ..., adoption_score: _Optional[float] = ..., violations: _Optional[_Iterable[str]] = ..., autofix: _Optional[_Union[AutofixCoverage, _Mapping]] = ...) -> None: ...

class AutofixCoverage(_message.Message):
    __slots__ = ("total", "fixable_universe", "implemented", "pending", "manual", "declared", "declaration_complete", "implementation_rate")
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    FIXABLE_UNIVERSE_FIELD_NUMBER: _ClassVar[int]
    IMPLEMENTED_FIELD_NUMBER: _ClassVar[int]
    PENDING_FIELD_NUMBER: _ClassVar[int]
    MANUAL_FIELD_NUMBER: _ClassVar[int]
    DECLARED_FIELD_NUMBER: _ClassVar[int]
    DECLARATION_COMPLETE_FIELD_NUMBER: _ClassVar[int]
    IMPLEMENTATION_RATE_FIELD_NUMBER: _ClassVar[int]
    total: int
    fixable_universe: int
    implemented: int
    pending: int
    manual: int
    declared: int
    declaration_complete: bool
    implementation_rate: float
    def __init__(self, total: _Optional[int] = ..., fixable_universe: _Optional[int] = ..., implemented: _Optional[int] = ..., pending: _Optional[int] = ..., manual: _Optional[int] = ..., declared: _Optional[int] = ..., declaration_complete: _Optional[bool] = ..., implementation_rate: _Optional[float] = ...) -> None: ...

class ReliabilityLedger(_message.Message):
    __slots__ = ("window_days", "run_count", "availability", "run_outcomes", "phases", "providers", "captured_at", "trend")
    WINDOW_DAYS_FIELD_NUMBER: _ClassVar[int]
    RUN_COUNT_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    RUN_OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    PROVIDERS_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    TREND_FIELD_NUMBER: _ClassVar[int]
    window_days: int
    run_count: int
    availability: float
    run_outcomes: _containers.RepeatedCompositeFieldContainer[RunOutcomeCount]
    phases: _containers.RepeatedCompositeFieldContainer[PhaseReliability]
    providers: _containers.RepeatedCompositeFieldContainer[ProviderReliability]
    captured_at: str
    trend: TrendDelta
    def __init__(self, window_days: _Optional[int] = ..., run_count: _Optional[int] = ..., availability: _Optional[float] = ..., run_outcomes: _Optional[_Iterable[_Union[RunOutcomeCount, _Mapping]]] = ..., phases: _Optional[_Iterable[_Union[PhaseReliability, _Mapping]]] = ..., providers: _Optional[_Iterable[_Union[ProviderReliability, _Mapping]]] = ..., captured_at: _Optional[str] = ..., trend: _Optional[_Union[TrendDelta, _Mapping]] = ...) -> None: ...

class RunOutcomeCount(_message.Message):
    __slots__ = ("outcome", "count")
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    outcome: str
    count: int
    def __init__(self, outcome: _Optional[str] = ..., count: _Optional[int] = ...) -> None: ...

class LabeledCount(_message.Message):
    __slots__ = ("label", "count")
    LABEL_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    label: str
    count: int
    def __init__(self, label: _Optional[str] = ..., count: _Optional[int] = ...) -> None: ...

class DurationStats(_message.Message):
    __slots__ = ("samples", "p50", "p95", "min", "max", "avg")
    SAMPLES_FIELD_NUMBER: _ClassVar[int]
    P50_FIELD_NUMBER: _ClassVar[int]
    P95_FIELD_NUMBER: _ClassVar[int]
    MIN_FIELD_NUMBER: _ClassVar[int]
    MAX_FIELD_NUMBER: _ClassVar[int]
    AVG_FIELD_NUMBER: _ClassVar[int]
    samples: int
    p50: int
    p95: int
    min: int
    max: int
    avg: int
    def __init__(self, samples: _Optional[int] = ..., p50: _Optional[int] = ..., p95: _Optional[int] = ..., min: _Optional[int] = ..., max: _Optional[int] = ..., avg: _Optional[int] = ...) -> None: ...

class ScenarioFailureRate(_message.Message):
    __slots__ = ("scenario", "executed", "failures", "failure_rate")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_FIELD_NUMBER: _ClassVar[int]
    FAILURES_FIELD_NUMBER: _ClassVar[int]
    FAILURE_RATE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    executed: int
    failures: int
    failure_rate: float
    def __init__(self, scenario: _Optional[str] = ..., executed: _Optional[int] = ..., failures: _Optional[int] = ..., failure_rate: _Optional[float] = ...) -> None: ...

class PhaseReliability(_message.Message):
    __slots__ = ("phase", "provider", "finding_source", "total_observations", "passed", "failed", "skipped", "degraded", "availability", "failure_rate", "metrics_adopted", "skip_reasons", "classifications", "duration", "worst_scenarios")
    PHASE_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    FINDING_SOURCE_FIELD_NUMBER: _ClassVar[int]
    TOTAL_OBSERVATIONS_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    FAILURE_RATE_FIELD_NUMBER: _ClassVar[int]
    METRICS_ADOPTED_FIELD_NUMBER: _ClassVar[int]
    SKIP_REASONS_FIELD_NUMBER: _ClassVar[int]
    CLASSIFICATIONS_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    WORST_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    phase: str
    provider: str
    finding_source: str
    total_observations: int
    passed: int
    failed: int
    skipped: int
    degraded: int
    availability: float
    failure_rate: float
    metrics_adopted: int
    skip_reasons: _containers.RepeatedCompositeFieldContainer[LabeledCount]
    classifications: _containers.RepeatedCompositeFieldContainer[LabeledCount]
    duration: DurationStats
    worst_scenarios: _containers.RepeatedCompositeFieldContainer[ScenarioFailureRate]
    def __init__(self, phase: _Optional[str] = ..., provider: _Optional[str] = ..., finding_source: _Optional[str] = ..., total_observations: _Optional[int] = ..., passed: _Optional[int] = ..., failed: _Optional[int] = ..., skipped: _Optional[int] = ..., degraded: _Optional[int] = ..., availability: _Optional[float] = ..., failure_rate: _Optional[float] = ..., metrics_adopted: _Optional[int] = ..., skip_reasons: _Optional[_Iterable[_Union[LabeledCount, _Mapping]]] = ..., classifications: _Optional[_Iterable[_Union[LabeledCount, _Mapping]]] = ..., duration: _Optional[_Union[DurationStats, _Mapping]] = ..., worst_scenarios: _Optional[_Iterable[_Union[ScenarioFailureRate, _Mapping]]] = ...) -> None: ...

class ProviderReliability(_message.Message):
    __slots__ = ("provider", "phases", "total_observations", "passed", "failed", "skipped", "availability", "failure_rate", "metrics_adopted", "duration")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_OBSERVATIONS_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    FAILURE_RATE_FIELD_NUMBER: _ClassVar[int]
    METRICS_ADOPTED_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    provider: str
    phases: _containers.RepeatedScalarFieldContainer[str]
    total_observations: int
    passed: int
    failed: int
    skipped: int
    availability: float
    failure_rate: float
    metrics_adopted: int
    duration: DurationStats
    def __init__(self, provider: _Optional[str] = ..., phases: _Optional[_Iterable[str]] = ..., total_observations: _Optional[int] = ..., passed: _Optional[int] = ..., failed: _Optional[int] = ..., skipped: _Optional[int] = ..., availability: _Optional[float] = ..., failure_rate: _Optional[float] = ..., metrics_adopted: _Optional[int] = ..., duration: _Optional[_Union[DurationStats, _Mapping]] = ...) -> None: ...
