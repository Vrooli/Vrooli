from common.v1 import maturity_pb2 as _maturity_pb2
from common.v1 import operations_pb2 as _operations_pb2
from common.v1 import validation_target_pb2 as _validation_target_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PhaseComparisonReasonCode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PHASE_COMPARISON_REASON_CODE_UNSPECIFIED: _ClassVar[PhaseComparisonReasonCode]
    PHASE_COMPARISON_REASON_CODE_NEW_PHASE: _ClassVar[PhaseComparisonReasonCode]
    PHASE_COMPARISON_REASON_CODE_RETIRED_PHASE: _ClassVar[PhaseComparisonReasonCode]
    PHASE_COMPARISON_REASON_CODE_INAPPLICABLE: _ClassVar[PhaseComparisonReasonCode]
    PHASE_COMPARISON_REASON_CODE_SKIPPED: _ClassVar[PhaseComparisonReasonCode]
    PHASE_COMPARISON_REASON_CODE_PROVIDER_UNAVAILABLE: _ClassVar[PhaseComparisonReasonCode]
    PHASE_COMPARISON_REASON_CODE_MISSING_ARTIFACT: _ClassVar[PhaseComparisonReasonCode]
    PHASE_COMPARISON_REASON_CODE_INCOMPATIBLE_SCHEMA: _ClassVar[PhaseComparisonReasonCode]
    PHASE_COMPARISON_REASON_CODE_LEGACY_METADATA_UNAVAILABLE: _ClassVar[PhaseComparisonReasonCode]

class ArtifactAccessCapability(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ARTIFACT_ACCESS_CAPABILITY_UNSPECIFIED: _ClassVar[ArtifactAccessCapability]
    ARTIFACT_ACCESS_CAPABILITY_STREAM: _ClassVar[ArtifactAccessCapability]

class ArtifactProvenance(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ARTIFACT_PROVENANCE_UNSPECIFIED: _ClassVar[ArtifactProvenance]
    ARTIFACT_PROVENANCE_CATALOG: _ClassVar[ArtifactProvenance]
    ARTIFACT_PROVENANCE_LEGACY_DISCOVERY: _ClassVar[ArtifactProvenance]
PHASE_COMPARISON_REASON_CODE_UNSPECIFIED: PhaseComparisonReasonCode
PHASE_COMPARISON_REASON_CODE_NEW_PHASE: PhaseComparisonReasonCode
PHASE_COMPARISON_REASON_CODE_RETIRED_PHASE: PhaseComparisonReasonCode
PHASE_COMPARISON_REASON_CODE_INAPPLICABLE: PhaseComparisonReasonCode
PHASE_COMPARISON_REASON_CODE_SKIPPED: PhaseComparisonReasonCode
PHASE_COMPARISON_REASON_CODE_PROVIDER_UNAVAILABLE: PhaseComparisonReasonCode
PHASE_COMPARISON_REASON_CODE_MISSING_ARTIFACT: PhaseComparisonReasonCode
PHASE_COMPARISON_REASON_CODE_INCOMPATIBLE_SCHEMA: PhaseComparisonReasonCode
PHASE_COMPARISON_REASON_CODE_LEGACY_METADATA_UNAVAILABLE: PhaseComparisonReasonCode
ARTIFACT_ACCESS_CAPABILITY_UNSPECIFIED: ArtifactAccessCapability
ARTIFACT_ACCESS_CAPABILITY_STREAM: ArtifactAccessCapability
ARTIFACT_PROVENANCE_UNSPECIFIED: ArtifactProvenance
ARTIFACT_PROVENANCE_CATALOG: ArtifactProvenance
ARTIFACT_PROVENANCE_LEGACY_DISCOVERY: ArtifactProvenance

class RunEvent(_message.Message):
    __slots__ = ("event", "elapsed_seconds", "run_id", "scenario", "artifact_dir", "preset", "phase", "phase_index", "phase_total", "status", "duration_seconds", "quiet_seconds", "message", "success", "verdict", "error", "maturity_standing", "findings_summary", "phase_presentation", "sequence", "target")
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
    PHASE_PRESENTATION_FIELD_NUMBER: _ClassVar[int]
    SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
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
    phase_presentation: _maturity_pb2.PhasePresentation
    sequence: int
    target: _validation_target_pb2.ValidationTarget
    def __init__(self, event: _Optional[str] = ..., elapsed_seconds: _Optional[float] = ..., run_id: _Optional[str] = ..., scenario: _Optional[str] = ..., artifact_dir: _Optional[str] = ..., preset: _Optional[str] = ..., phase: _Optional[str] = ..., phase_index: _Optional[int] = ..., phase_total: _Optional[int] = ..., status: _Optional[str] = ..., duration_seconds: _Optional[int] = ..., quiet_seconds: _Optional[float] = ..., message: _Optional[str] = ..., success: _Optional[bool] = ..., verdict: _Optional[str] = ..., error: _Optional[str] = ..., maturity_standing: _Optional[_Union[PhaseMaturityStanding, _Mapping]] = ..., findings_summary: _Optional[_Union[PhaseFindingsSummary, _Mapping]] = ..., phase_presentation: _Optional[_Union[_maturity_pb2.PhasePresentation, _Mapping]] = ..., sequence: _Optional[int] = ..., target: _Optional[_Union[_validation_target_pb2.ValidationTarget, _Mapping]] = ...) -> None: ...

class RunLiveStatus(_message.Message):
    __slots__ = ("run_id", "scenario", "status", "active_phase", "phase_index", "phase_total", "started_at", "elapsed_seconds", "estimated_total_seconds", "estimated_remaining_seconds", "eta_known", "recommended_next_check_seconds", "verdict", "success", "error", "active", "terminal_standings", "terminal_findings_summaries", "degraded_reasons", "terminal_presentations", "standing", "target")
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
    DEGRADED_REASONS_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_PRESENTATIONS_FIELD_NUMBER: _ClassVar[int]
    STANDING_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
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
    degraded_reasons: _containers.RepeatedScalarFieldContainer[str]
    terminal_presentations: _containers.RepeatedCompositeFieldContainer[_maturity_pb2.PhasePresentation]
    standing: _operations_pb2.OperationStanding
    target: _validation_target_pb2.ValidationTarget
    def __init__(self, run_id: _Optional[str] = ..., scenario: _Optional[str] = ..., status: _Optional[str] = ..., active_phase: _Optional[str] = ..., phase_index: _Optional[int] = ..., phase_total: _Optional[int] = ..., started_at: _Optional[str] = ..., elapsed_seconds: _Optional[float] = ..., estimated_total_seconds: _Optional[int] = ..., estimated_remaining_seconds: _Optional[int] = ..., eta_known: _Optional[bool] = ..., recommended_next_check_seconds: _Optional[int] = ..., verdict: _Optional[str] = ..., success: _Optional[bool] = ..., error: _Optional[str] = ..., active: _Optional[bool] = ..., terminal_standings: _Optional[_Iterable[_Union[PhaseMaturityStanding, _Mapping]]] = ..., terminal_findings_summaries: _Optional[_Iterable[_Union[PhaseFindingsSummary, _Mapping]]] = ..., degraded_reasons: _Optional[_Iterable[str]] = ..., terminal_presentations: _Optional[_Iterable[_Union[_maturity_pb2.PhasePresentation, _Mapping]]] = ..., standing: _Optional[_Union[_operations_pb2.OperationStanding, _Mapping]] = ..., target: _Optional[_Union[_validation_target_pb2.ValidationTarget, _Mapping]] = ...) -> None: ...

class StartRunRequest(_message.Message):
    __slots__ = ("scenario", "preset", "phases", "skip", "fail_fast", "diagnostics_preset", "ui_url", "api_url", "scenario_path", "logical_repo_root", "logical_scenario_rel_path", "suite_request_id", "capture_profile", "require_gate_quality", "target")
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
    REQUIRE_GATE_QUALITY_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
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
    require_gate_quality: bool
    target: _validation_target_pb2.ValidationTarget
    def __init__(self, scenario: _Optional[str] = ..., preset: _Optional[str] = ..., phases: _Optional[_Iterable[str]] = ..., skip: _Optional[_Iterable[str]] = ..., fail_fast: _Optional[bool] = ..., diagnostics_preset: _Optional[str] = ..., ui_url: _Optional[str] = ..., api_url: _Optional[str] = ..., scenario_path: _Optional[str] = ..., logical_repo_root: _Optional[str] = ..., logical_scenario_rel_path: _Optional[str] = ..., suite_request_id: _Optional[str] = ..., capture_profile: _Optional[str] = ..., require_gate_quality: _Optional[bool] = ..., target: _Optional[_Union[_validation_target_pb2.ValidationTarget, _Mapping]] = ...) -> None: ...

class StartRunResponse(_message.Message):
    __slots__ = ("run_id", "scenario", "estimated_total_seconds", "eta_known", "coalesced", "target")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_TOTAL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    ETA_KNOWN_FIELD_NUMBER: _ClassVar[int]
    COALESCED_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    scenario: str
    estimated_total_seconds: int
    eta_known: bool
    coalesced: bool
    target: _validation_target_pb2.ValidationTarget
    def __init__(self, run_id: _Optional[str] = ..., scenario: _Optional[str] = ..., estimated_total_seconds: _Optional[int] = ..., eta_known: _Optional[bool] = ..., coalesced: _Optional[bool] = ..., target: _Optional[_Union[_validation_target_pb2.ValidationTarget, _Mapping]] = ...) -> None: ...

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
    __slots__ = ("scenario", "run_id", "suppress_heartbeats", "after_sequence")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    SUPPRESS_HEARTBEATS_FIELD_NUMBER: _ClassVar[int]
    AFTER_SEQUENCE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    suppress_heartbeats: bool
    after_sequence: int
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ..., suppress_heartbeats: _Optional[bool] = ..., after_sequence: _Optional[int] = ...) -> None: ...

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
    __slots__ = ("status", "timed_out", "terminal_run", "terminal_snapshot_schema_version", "degraded_reasons")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TIMED_OUT_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_RUN_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_SNAPSHOT_SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASONS_FIELD_NUMBER: _ClassVar[int]
    status: RunLiveStatus
    timed_out: bool
    terminal_run: RunInfo
    terminal_snapshot_schema_version: int
    degraded_reasons: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, status: _Optional[_Union[RunLiveStatus, _Mapping]] = ..., timed_out: _Optional[bool] = ..., terminal_run: _Optional[_Union[RunInfo, _Mapping]] = ..., terminal_snapshot_schema_version: _Optional[int] = ..., degraded_reasons: _Optional[_Iterable[str]] = ...) -> None: ...

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

class GetCostReportRequest(_message.Message):
    __slots__ = ("scenario", "window_seconds", "compare_window_seconds")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    WINDOW_SECONDS_FIELD_NUMBER: _ClassVar[int]
    COMPARE_WINDOW_SECONDS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    window_seconds: int
    compare_window_seconds: int
    def __init__(self, scenario: _Optional[str] = ..., window_seconds: _Optional[int] = ..., compare_window_seconds: _Optional[int] = ...) -> None: ...

class CostPhaseSummary(_message.Message):
    __slots__ = ("scenario", "phase", "sample_count", "reliable_sample_count", "excluded_sample_count", "total_wall_clock_ms", "median_wall_clock_ms", "p90_wall_clock_ms", "total_cpu_user_ms", "max_peak_rss_bytes", "change_wall_clock_ms", "change_percent", "prediction_sample_count", "prediction_error_total_ms", "prediction_mean_absolute_error_ms", "prediction_mean_absolute_error_percent", "passing_sample_count", "failing_sample_count", "passing_median_wall_clock_ms", "passing_p90_wall_clock_ms", "failing_median_wall_clock_ms", "failing_p90_wall_clock_ms")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    RELIABLE_SAMPLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    EXCLUDED_SAMPLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_WALL_CLOCK_MS_FIELD_NUMBER: _ClassVar[int]
    MEDIAN_WALL_CLOCK_MS_FIELD_NUMBER: _ClassVar[int]
    P90_WALL_CLOCK_MS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_CPU_USER_MS_FIELD_NUMBER: _ClassVar[int]
    MAX_PEAK_RSS_BYTES_FIELD_NUMBER: _ClassVar[int]
    CHANGE_WALL_CLOCK_MS_FIELD_NUMBER: _ClassVar[int]
    CHANGE_PERCENT_FIELD_NUMBER: _ClassVar[int]
    PREDICTION_SAMPLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    PREDICTION_ERROR_TOTAL_MS_FIELD_NUMBER: _ClassVar[int]
    PREDICTION_MEAN_ABSOLUTE_ERROR_MS_FIELD_NUMBER: _ClassVar[int]
    PREDICTION_MEAN_ABSOLUTE_ERROR_PERCENT_FIELD_NUMBER: _ClassVar[int]
    PASSING_SAMPLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    FAILING_SAMPLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    PASSING_MEDIAN_WALL_CLOCK_MS_FIELD_NUMBER: _ClassVar[int]
    PASSING_P90_WALL_CLOCK_MS_FIELD_NUMBER: _ClassVar[int]
    FAILING_MEDIAN_WALL_CLOCK_MS_FIELD_NUMBER: _ClassVar[int]
    FAILING_P90_WALL_CLOCK_MS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    phase: str
    sample_count: int
    reliable_sample_count: int
    excluded_sample_count: int
    total_wall_clock_ms: int
    median_wall_clock_ms: int
    p90_wall_clock_ms: int
    total_cpu_user_ms: int
    max_peak_rss_bytes: int
    change_wall_clock_ms: int
    change_percent: float
    prediction_sample_count: int
    prediction_error_total_ms: int
    prediction_mean_absolute_error_ms: int
    prediction_mean_absolute_error_percent: float
    passing_sample_count: int
    failing_sample_count: int
    passing_median_wall_clock_ms: int
    passing_p90_wall_clock_ms: int
    failing_median_wall_clock_ms: int
    failing_p90_wall_clock_ms: int
    def __init__(self, scenario: _Optional[str] = ..., phase: _Optional[str] = ..., sample_count: _Optional[int] = ..., reliable_sample_count: _Optional[int] = ..., excluded_sample_count: _Optional[int] = ..., total_wall_clock_ms: _Optional[int] = ..., median_wall_clock_ms: _Optional[int] = ..., p90_wall_clock_ms: _Optional[int] = ..., total_cpu_user_ms: _Optional[int] = ..., max_peak_rss_bytes: _Optional[int] = ..., change_wall_clock_ms: _Optional[int] = ..., change_percent: _Optional[float] = ..., prediction_sample_count: _Optional[int] = ..., prediction_error_total_ms: _Optional[int] = ..., prediction_mean_absolute_error_ms: _Optional[int] = ..., prediction_mean_absolute_error_percent: _Optional[float] = ..., passing_sample_count: _Optional[int] = ..., failing_sample_count: _Optional[int] = ..., passing_median_wall_clock_ms: _Optional[int] = ..., passing_p90_wall_clock_ms: _Optional[int] = ..., failing_median_wall_clock_ms: _Optional[int] = ..., failing_p90_wall_clock_ms: _Optional[int] = ...) -> None: ...

class GetCostReportResponse(_message.Message):
    __slots__ = ("phases", "window_seconds", "compare_window_seconds")
    PHASES_FIELD_NUMBER: _ClassVar[int]
    WINDOW_SECONDS_FIELD_NUMBER: _ClassVar[int]
    COMPARE_WINDOW_SECONDS_FIELD_NUMBER: _ClassVar[int]
    phases: _containers.RepeatedCompositeFieldContainer[CostPhaseSummary]
    window_seconds: int
    compare_window_seconds: int
    def __init__(self, phases: _Optional[_Iterable[_Union[CostPhaseSummary, _Mapping]]] = ..., window_seconds: _Optional[int] = ..., compare_window_seconds: _Optional[int] = ...) -> None: ...

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
    __slots__ = ("name", "status", "duration_seconds", "comparable", "advisory", "artifact_backed", "non_comparable", "maturity_standing", "findings_summary", "phase_presentation")
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    COMPARABLE_FIELD_NUMBER: _ClassVar[int]
    ADVISORY_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_BACKED_FIELD_NUMBER: _ClassVar[int]
    NON_COMPARABLE_FIELD_NUMBER: _ClassVar[int]
    MATURITY_STANDING_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    PHASE_PRESENTATION_FIELD_NUMBER: _ClassVar[int]
    name: str
    status: str
    duration_seconds: float
    comparable: bool
    advisory: bool
    artifact_backed: bool
    non_comparable: bool
    maturity_standing: PhaseMaturityStanding
    findings_summary: PhaseFindingsSummary
    phase_presentation: _maturity_pb2.PhasePresentation
    def __init__(self, name: _Optional[str] = ..., status: _Optional[str] = ..., duration_seconds: _Optional[float] = ..., comparable: _Optional[bool] = ..., advisory: _Optional[bool] = ..., artifact_backed: _Optional[bool] = ..., non_comparable: _Optional[bool] = ..., maturity_standing: _Optional[_Union[PhaseMaturityStanding, _Mapping]] = ..., findings_summary: _Optional[_Union[PhaseFindingsSummary, _Mapping]] = ..., phase_presentation: _Optional[_Union[_maturity_pb2.PhasePresentation, _Mapping]] = ...) -> None: ...

class PhaseDescriptorPolicy(_message.Message):
    __slots__ = ("selection", "provider_readiness", "provider_lifecycle", "freshness", "result_gating", "unavailable")
    SELECTION_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_READINESS_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_LIFECYCLE_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    RESULT_GATING_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_FIELD_NUMBER: _ClassVar[int]
    selection: str
    provider_readiness: str
    provider_lifecycle: str
    freshness: str
    result_gating: str
    unavailable: str
    def __init__(self, selection: _Optional[str] = ..., provider_readiness: _Optional[str] = ..., provider_lifecycle: _Optional[str] = ..., freshness: _Optional[str] = ..., result_gating: _Optional[str] = ..., unavailable: _Optional[str] = ...) -> None: ...

class PhaseApplicabilityDecision(_message.Message):
    __slots__ = ("status", "reason_codes", "reasons", "planned")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REASON_CODES_FIELD_NUMBER: _ClassVar[int]
    REASONS_FIELD_NUMBER: _ClassVar[int]
    PLANNED_FIELD_NUMBER: _ClassVar[int]
    status: str
    reason_codes: _containers.RepeatedScalarFieldContainer[str]
    reasons: _containers.RepeatedScalarFieldContainer[str]
    planned: bool
    def __init__(self, status: _Optional[str] = ..., reason_codes: _Optional[_Iterable[str]] = ..., reasons: _Optional[_Iterable[str]] = ..., planned: _Optional[bool] = ...) -> None: ...

class RunPhaseDescriptor(_message.Message):
    __slots__ = ("phase", "display_name", "description", "provider", "order_hint", "phase_class", "runtime_class", "dimensions", "finding_source", "policy", "docs_path", "maturity_reference", "applicability_default", "evidence_kinds", "aliases", "supersedes", "applicability", "comparison_fingerprint", "comparison_mode")
    PHASE_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    ORDER_HINT_FIELD_NUMBER: _ClassVar[int]
    PHASE_CLASS_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_CLASS_FIELD_NUMBER: _ClassVar[int]
    DIMENSIONS_FIELD_NUMBER: _ClassVar[int]
    FINDING_SOURCE_FIELD_NUMBER: _ClassVar[int]
    POLICY_FIELD_NUMBER: _ClassVar[int]
    DOCS_PATH_FIELD_NUMBER: _ClassVar[int]
    MATURITY_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    APPLICABILITY_DEFAULT_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_KINDS_FIELD_NUMBER: _ClassVar[int]
    ALIASES_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDES_FIELD_NUMBER: _ClassVar[int]
    APPLICABILITY_FIELD_NUMBER: _ClassVar[int]
    COMPARISON_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    COMPARISON_MODE_FIELD_NUMBER: _ClassVar[int]
    phase: str
    display_name: str
    description: str
    provider: str
    order_hint: int
    phase_class: str
    runtime_class: str
    dimensions: _containers.RepeatedScalarFieldContainer[str]
    finding_source: str
    policy: PhaseDescriptorPolicy
    docs_path: str
    maturity_reference: str
    applicability_default: str
    evidence_kinds: _containers.RepeatedScalarFieldContainer[str]
    aliases: _containers.RepeatedScalarFieldContainer[str]
    supersedes: _containers.RepeatedScalarFieldContainer[str]
    applicability: PhaseApplicabilityDecision
    comparison_fingerprint: str
    comparison_mode: str
    def __init__(self, phase: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., provider: _Optional[str] = ..., order_hint: _Optional[int] = ..., phase_class: _Optional[str] = ..., runtime_class: _Optional[str] = ..., dimensions: _Optional[_Iterable[str]] = ..., finding_source: _Optional[str] = ..., policy: _Optional[_Union[PhaseDescriptorPolicy, _Mapping]] = ..., docs_path: _Optional[str] = ..., maturity_reference: _Optional[str] = ..., applicability_default: _Optional[str] = ..., evidence_kinds: _Optional[_Iterable[str]] = ..., aliases: _Optional[_Iterable[str]] = ..., supersedes: _Optional[_Iterable[str]] = ..., applicability: _Optional[_Union[PhaseApplicabilityDecision, _Mapping]] = ..., comparison_fingerprint: _Optional[str] = ..., comparison_mode: _Optional[str] = ...) -> None: ...

class RunDescriptorSnapshot(_message.Message):
    __slots__ = ("schema_version", "digest", "phases")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    schema_version: int
    digest: str
    phases: _containers.RepeatedCompositeFieldContainer[RunPhaseDescriptor]
    def __init__(self, schema_version: _Optional[int] = ..., digest: _Optional[str] = ..., phases: _Optional[_Iterable[_Union[RunPhaseDescriptor, _Mapping]]] = ...) -> None: ...

class RunInfo(_message.Message):
    __slots__ = ("run_id", "scenario", "started_at", "completed_at", "status", "phases", "git_sha", "git_branch", "git_dirty", "git_dirty_summary", "diagnostics", "pins", "tree_digest", "preset", "capture_profile", "planned_phases", "phase_set_digest", "descriptor_snapshot_schema_version", "descriptor_snapshot_digest", "descriptor_snapshot", "execution_configuration_fingerprint", "gate_quality", "evidence_tier", "source_scope", "source_stable", "target")
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
    DESCRIPTOR_SNAPSHOT_SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTOR_SNAPSHOT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTOR_SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_CONFIGURATION_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    GATE_QUALITY_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_TIER_FIELD_NUMBER: _ClassVar[int]
    SOURCE_SCOPE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_STABLE_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
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
    descriptor_snapshot_schema_version: int
    descriptor_snapshot_digest: str
    descriptor_snapshot: RunDescriptorSnapshot
    execution_configuration_fingerprint: str
    gate_quality: bool
    evidence_tier: str
    source_scope: str
    source_stable: bool
    target: _validation_target_pb2.ValidationTarget
    def __init__(self, run_id: _Optional[str] = ..., scenario: _Optional[str] = ..., started_at: _Optional[str] = ..., completed_at: _Optional[str] = ..., status: _Optional[str] = ..., phases: _Optional[_Iterable[_Union[PhaseInfo, _Mapping]]] = ..., git_sha: _Optional[str] = ..., git_branch: _Optional[str] = ..., git_dirty: _Optional[bool] = ..., git_dirty_summary: _Optional[str] = ..., diagnostics: _Optional[_Union[DiagnosticsInfo, _Mapping]] = ..., pins: _Optional[_Iterable[_Union[PinInfo, _Mapping]]] = ..., tree_digest: _Optional[str] = ..., preset: _Optional[str] = ..., capture_profile: _Optional[str] = ..., planned_phases: _Optional[_Iterable[str]] = ..., phase_set_digest: _Optional[str] = ..., descriptor_snapshot_schema_version: _Optional[int] = ..., descriptor_snapshot_digest: _Optional[str] = ..., descriptor_snapshot: _Optional[_Union[RunDescriptorSnapshot, _Mapping]] = ..., execution_configuration_fingerprint: _Optional[str] = ..., gate_quality: _Optional[bool] = ..., evidence_tier: _Optional[str] = ..., source_scope: _Optional[str] = ..., source_stable: _Optional[bool] = ..., target: _Optional[_Union[_validation_target_pb2.ValidationTarget, _Mapping]] = ...) -> None: ...

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
    __slots__ = ("run", "terminal_snapshot_schema_version", "degraded_reasons")
    RUN_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_SNAPSHOT_SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASONS_FIELD_NUMBER: _ClassVar[int]
    run: RunInfo
    terminal_snapshot_schema_version: int
    degraded_reasons: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, run: _Optional[_Union[RunInfo, _Mapping]] = ..., terminal_snapshot_schema_version: _Optional[int] = ..., degraded_reasons: _Optional[_Iterable[str]] = ...) -> None: ...

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

class PhaseComparisonReason(_message.Message):
    __slots__ = ("code", "detail")
    CODE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    code: PhaseComparisonReasonCode
    detail: str
    def __init__(self, code: _Optional[_Union[PhaseComparisonReasonCode, str]] = ..., detail: _Optional[str] = ...) -> None: ...

class ComparisonDiagnostic(_message.Message):
    __slots__ = ("side", "code", "detail", "classification", "lifecycle_action", "remediation", "observations")
    SIDE_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    LIFECYCLE_ACTION_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    OBSERVATIONS_FIELD_NUMBER: _ClassVar[int]
    side: str
    code: str
    detail: str
    classification: str
    lifecycle_action: str
    remediation: str
    observations: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, side: _Optional[str] = ..., code: _Optional[str] = ..., detail: _Optional[str] = ..., classification: _Optional[str] = ..., lifecycle_action: _Optional[str] = ..., remediation: _Optional[str] = ..., observations: _Optional[_Iterable[str]] = ...) -> None: ...

class PhaseDiff(_message.Message):
    __slots__ = ("phase", "status_a", "status_b", "verdict", "regressions", "new_failures", "preexisting_failures", "cleared_failures", "descriptor_a", "descriptor_b", "reasons", "behavior", "coverage", "compatibility", "provenance", "diagnostics")
    PHASE_FIELD_NUMBER: _ClassVar[int]
    STATUS_A_FIELD_NUMBER: _ClassVar[int]
    STATUS_B_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    REGRESSIONS_FIELD_NUMBER: _ClassVar[int]
    NEW_FAILURES_FIELD_NUMBER: _ClassVar[int]
    PREEXISTING_FAILURES_FIELD_NUMBER: _ClassVar[int]
    CLEARED_FAILURES_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTOR_A_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTOR_B_FIELD_NUMBER: _ClassVar[int]
    REASONS_FIELD_NUMBER: _ClassVar[int]
    BEHAVIOR_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    COMPATIBILITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTICS_FIELD_NUMBER: _ClassVar[int]
    phase: str
    status_a: str
    status_b: str
    verdict: str
    regressions: _containers.RepeatedScalarFieldContainer[str]
    new_failures: _containers.RepeatedScalarFieldContainer[str]
    preexisting_failures: _containers.RepeatedScalarFieldContainer[str]
    cleared_failures: _containers.RepeatedScalarFieldContainer[str]
    descriptor_a: RunPhaseDescriptor
    descriptor_b: RunPhaseDescriptor
    reasons: _containers.RepeatedCompositeFieldContainer[PhaseComparisonReason]
    behavior: str
    coverage: str
    compatibility: str
    provenance: str
    diagnostics: _containers.RepeatedCompositeFieldContainer[ComparisonDiagnostic]
    def __init__(self, phase: _Optional[str] = ..., status_a: _Optional[str] = ..., status_b: _Optional[str] = ..., verdict: _Optional[str] = ..., regressions: _Optional[_Iterable[str]] = ..., new_failures: _Optional[_Iterable[str]] = ..., preexisting_failures: _Optional[_Iterable[str]] = ..., cleared_failures: _Optional[_Iterable[str]] = ..., descriptor_a: _Optional[_Union[RunPhaseDescriptor, _Mapping]] = ..., descriptor_b: _Optional[_Union[RunPhaseDescriptor, _Mapping]] = ..., reasons: _Optional[_Iterable[_Union[PhaseComparisonReason, _Mapping]]] = ..., behavior: _Optional[str] = ..., coverage: _Optional[str] = ..., compatibility: _Optional[str] = ..., provenance: _Optional[str] = ..., diagnostics: _Optional[_Iterable[_Union[ComparisonDiagnostic, _Mapping]]] = ...) -> None: ...

class CompareRunsResponse(_message.Message):
    __slots__ = ("phases", "verdict", "schema_version", "behavior", "coverage", "compatibility", "provenance", "diagnostics")
    PHASES_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    BEHAVIOR_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    COMPATIBILITY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTICS_FIELD_NUMBER: _ClassVar[int]
    phases: _containers.RepeatedCompositeFieldContainer[PhaseDiff]
    verdict: str
    schema_version: int
    behavior: str
    coverage: str
    compatibility: str
    provenance: str
    diagnostics: _containers.RepeatedCompositeFieldContainer[ComparisonDiagnostic]
    def __init__(self, phases: _Optional[_Iterable[_Union[PhaseDiff, _Mapping]]] = ..., verdict: _Optional[str] = ..., schema_version: _Optional[int] = ..., behavior: _Optional[str] = ..., coverage: _Optional[str] = ..., compatibility: _Optional[str] = ..., provenance: _Optional[str] = ..., diagnostics: _Optional[_Iterable[_Union[ComparisonDiagnostic, _Mapping]]] = ...) -> None: ...

class FindRunRequest(_message.Message):
    __slots__ = ("scenario", "git_sha", "tree_digest", "preset", "capture_profile", "status", "require_clean", "phase_set_digest", "require_gate_quality", "match_current_source")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    GIT_SHA_FIELD_NUMBER: _ClassVar[int]
    TREE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    PRESET_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_PROFILE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REQUIRE_CLEAN_FIELD_NUMBER: _ClassVar[int]
    PHASE_SET_DIGEST_FIELD_NUMBER: _ClassVar[int]
    REQUIRE_GATE_QUALITY_FIELD_NUMBER: _ClassVar[int]
    MATCH_CURRENT_SOURCE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    git_sha: str
    tree_digest: str
    preset: str
    capture_profile: str
    status: str
    require_clean: bool
    phase_set_digest: str
    require_gate_quality: bool
    match_current_source: bool
    def __init__(self, scenario: _Optional[str] = ..., git_sha: _Optional[str] = ..., tree_digest: _Optional[str] = ..., preset: _Optional[str] = ..., capture_profile: _Optional[str] = ..., status: _Optional[str] = ..., require_clean: _Optional[bool] = ..., phase_set_digest: _Optional[str] = ..., require_gate_quality: _Optional[bool] = ..., match_current_source: _Optional[bool] = ...) -> None: ...

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

class ArtifactRelationship(_message.Message):
    __slots__ = ("type", "target_artifact_id")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    TARGET_ARTIFACT_ID_FIELD_NUMBER: _ClassVar[int]
    type: str
    target_artifact_id: str
    def __init__(self, type: _Optional[str] = ..., target_artifact_id: _Optional[str] = ...) -> None: ...

class ArtifactComparison(_message.Message):
    __slots__ = ("semantics", "analyzer", "metadata")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SEMANTICS_FIELD_NUMBER: _ClassVar[int]
    ANALYZER_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    semantics: str
    analyzer: str
    metadata: _containers.ScalarMap[str, str]
    def __init__(self, semantics: _Optional[str] = ..., analyzer: _Optional[str] = ..., metadata: _Optional[_Mapping[str, str]] = ...) -> None: ...

class ArtifactRef(_message.Message):
    __slots__ = ("id", "kind", "media_type", "label", "producing_phase", "size_bytes", "created_at", "access_capability", "access_path", "metadata", "relationships", "comparison", "provenance")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    PRODUCING_PHASE_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    ACCESS_CAPABILITY_FIELD_NUMBER: _ClassVar[int]
    ACCESS_PATH_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    RELATIONSHIPS_FIELD_NUMBER: _ClassVar[int]
    COMPARISON_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    media_type: str
    label: str
    producing_phase: str
    size_bytes: int
    created_at: str
    access_capability: ArtifactAccessCapability
    access_path: str
    metadata: _containers.ScalarMap[str, str]
    relationships: _containers.RepeatedCompositeFieldContainer[ArtifactRelationship]
    comparison: ArtifactComparison
    provenance: ArtifactProvenance
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., media_type: _Optional[str] = ..., label: _Optional[str] = ..., producing_phase: _Optional[str] = ..., size_bytes: _Optional[int] = ..., created_at: _Optional[str] = ..., access_capability: _Optional[_Union[ArtifactAccessCapability, str]] = ..., access_path: _Optional[str] = ..., metadata: _Optional[_Mapping[str, str]] = ..., relationships: _Optional[_Iterable[_Union[ArtifactRelationship, _Mapping]]] = ..., comparison: _Optional[_Union[ArtifactComparison, _Mapping]] = ..., provenance: _Optional[_Union[ArtifactProvenance, str]] = ...) -> None: ...

class ListRunArtifactsRequest(_message.Message):
    __slots__ = ("scenario", "run_id", "kinds", "producing_phase")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    KINDS_FIELD_NUMBER: _ClassVar[int]
    PRODUCING_PHASE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    kinds: _containers.RepeatedScalarFieldContainer[str]
    producing_phase: str
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ..., kinds: _Optional[_Iterable[str]] = ..., producing_phase: _Optional[str] = ...) -> None: ...

class ListRunArtifactsResponse(_message.Message):
    __slots__ = ("schema_version", "digest", "artifacts", "legacy_discovered", "degraded_reasons")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    LEGACY_DISCOVERED_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASONS_FIELD_NUMBER: _ClassVar[int]
    schema_version: int
    digest: str
    artifacts: _containers.RepeatedCompositeFieldContainer[ArtifactRef]
    legacy_discovered: bool
    degraded_reasons: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, schema_version: _Optional[int] = ..., digest: _Optional[str] = ..., artifacts: _Optional[_Iterable[_Union[ArtifactRef, _Mapping]]] = ..., legacy_discovered: _Optional[bool] = ..., degraded_reasons: _Optional[_Iterable[str]] = ...) -> None: ...

class GetRunArtifactRequest(_message.Message):
    __slots__ = ("scenario", "run_id", "artifact_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    artifact_id: str
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ..., artifact_id: _Optional[str] = ...) -> None: ...

class GetRunArtifactResponse(_message.Message):
    __slots__ = ("artifact", "legacy_discovered")
    ARTIFACT_FIELD_NUMBER: _ClassVar[int]
    LEGACY_DISCOVERED_FIELD_NUMBER: _ClassVar[int]
    artifact: ArtifactRef
    legacy_discovered: bool
    def __init__(self, artifact: _Optional[_Union[ArtifactRef, _Mapping]] = ..., legacy_discovered: _Optional[bool] = ...) -> None: ...

class GetRunFindingsRequest(_message.Message):
    __slots__ = ("scenario", "run_id")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    run_id: str
    def __init__(self, scenario: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class RunFindingsPhase(_message.Message):
    __slots__ = ("name", "status", "finding_source", "maturity_standing", "findings_summary", "phase_presentation")
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    FINDING_SOURCE_FIELD_NUMBER: _ClassVar[int]
    MATURITY_STANDING_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    PHASE_PRESENTATION_FIELD_NUMBER: _ClassVar[int]
    name: str
    status: str
    finding_source: str
    maturity_standing: PhaseMaturityStanding
    findings_summary: PhaseFindingsSummary
    phase_presentation: _maturity_pb2.PhasePresentation
    def __init__(self, name: _Optional[str] = ..., status: _Optional[str] = ..., finding_source: _Optional[str] = ..., maturity_standing: _Optional[_Union[PhaseMaturityStanding, _Mapping]] = ..., findings_summary: _Optional[_Union[PhaseFindingsSummary, _Mapping]] = ..., phase_presentation: _Optional[_Union[_maturity_pb2.PhasePresentation, _Mapping]] = ...) -> None: ...

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
    __slots__ = ("page", "label", "status", "changed_fraction", "base_artifact_id", "current_artifact_id")
    PAGE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CHANGED_FRACTION_FIELD_NUMBER: _ClassVar[int]
    BASE_ARTIFACT_ID_FIELD_NUMBER: _ClassVar[int]
    CURRENT_ARTIFACT_ID_FIELD_NUMBER: _ClassVar[int]
    page: str
    label: str
    status: str
    changed_fraction: float
    base_artifact_id: str
    current_artifact_id: str
    def __init__(self, page: _Optional[str] = ..., label: _Optional[str] = ..., status: _Optional[str] = ..., changed_fraction: _Optional[float] = ..., base_artifact_id: _Optional[str] = ..., current_artifact_id: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("window_days", "captured_at", "scenarios_tested", "scenarios_total", "total_runs", "total_issues", "scenarios", "top_finding_sources", "never_tested_in_window", "alerts")
    WINDOW_DAYS_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_TESTED_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_TOTAL_FIELD_NUMBER: _ClassVar[int]
    TOTAL_RUNS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_ISSUES_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    TOP_FINDING_SOURCES_FIELD_NUMBER: _ClassVar[int]
    NEVER_TESTED_IN_WINDOW_FIELD_NUMBER: _ClassVar[int]
    ALERTS_FIELD_NUMBER: _ClassVar[int]
    window_days: int
    captured_at: str
    scenarios_tested: int
    scenarios_total: int
    total_runs: int
    total_issues: int
    scenarios: _containers.RepeatedCompositeFieldContainer[FleetScenarioHealth]
    top_finding_sources: _containers.RepeatedCompositeFieldContainer[FleetFindingSource]
    never_tested_in_window: _containers.RepeatedScalarFieldContainer[str]
    alerts: _containers.RepeatedCompositeFieldContainer[FleetAlert]
    def __init__(self, window_days: _Optional[int] = ..., captured_at: _Optional[str] = ..., scenarios_tested: _Optional[int] = ..., scenarios_total: _Optional[int] = ..., total_runs: _Optional[int] = ..., total_issues: _Optional[int] = ..., scenarios: _Optional[_Iterable[_Union[FleetScenarioHealth, _Mapping]]] = ..., top_finding_sources: _Optional[_Iterable[_Union[FleetFindingSource, _Mapping]]] = ..., never_tested_in_window: _Optional[_Iterable[str]] = ..., alerts: _Optional[_Iterable[_Union[FleetAlert, _Mapping]]] = ...) -> None: ...

class FleetAlert(_message.Message):
    __slots__ = ("code", "severity", "scenario", "source", "message", "evidence_age_days", "owner", "next_action", "rollback_path")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_AGE_DAYS_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    ROLLBACK_PATH_FIELD_NUMBER: _ClassVar[int]
    code: str
    severity: str
    scenario: str
    source: str
    message: str
    evidence_age_days: float
    owner: str
    next_action: str
    rollback_path: str
    def __init__(self, code: _Optional[str] = ..., severity: _Optional[str] = ..., scenario: _Optional[str] = ..., source: _Optional[str] = ..., message: _Optional[str] = ..., evidence_age_days: _Optional[float] = ..., owner: _Optional[str] = ..., next_action: _Optional[str] = ..., rollback_path: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("provider", "phase", "reachable", "contract_valid", "identity_ok", "spec_valid", "metrics_adopted", "adoption_score", "violations", "autofix", "classification", "reason_codes", "fix_contract_required", "fix_contract_valid", "concurrency_declared", "metrics_reachable")
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
    CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    REASON_CODES_FIELD_NUMBER: _ClassVar[int]
    FIX_CONTRACT_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    FIX_CONTRACT_VALID_FIELD_NUMBER: _ClassVar[int]
    CONCURRENCY_DECLARED_FIELD_NUMBER: _ClassVar[int]
    METRICS_REACHABLE_FIELD_NUMBER: _ClassVar[int]
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
    classification: str
    reason_codes: _containers.RepeatedScalarFieldContainer[str]
    fix_contract_required: bool
    fix_contract_valid: bool
    concurrency_declared: bool
    metrics_reachable: bool
    def __init__(self, provider: _Optional[str] = ..., phase: _Optional[str] = ..., reachable: _Optional[bool] = ..., contract_valid: _Optional[bool] = ..., identity_ok: _Optional[bool] = ..., spec_valid: _Optional[bool] = ..., metrics_adopted: _Optional[bool] = ..., adoption_score: _Optional[float] = ..., violations: _Optional[_Iterable[str]] = ..., autofix: _Optional[_Union[AutofixCoverage, _Mapping]] = ..., classification: _Optional[str] = ..., reason_codes: _Optional[_Iterable[str]] = ..., fix_contract_required: _Optional[bool] = ..., fix_contract_valid: _Optional[bool] = ..., concurrency_declared: _Optional[bool] = ..., metrics_reachable: _Optional[bool] = ...) -> None: ...

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

class SecurityFriction(_message.Message):
    __slots__ = ("failed_attempts", "green_transitions", "recurring_failures", "time_to_green_samples", "time_to_green")
    FAILED_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    GREEN_TRANSITIONS_FIELD_NUMBER: _ClassVar[int]
    RECURRING_FAILURES_FIELD_NUMBER: _ClassVar[int]
    TIME_TO_GREEN_SAMPLES_FIELD_NUMBER: _ClassVar[int]
    TIME_TO_GREEN_FIELD_NUMBER: _ClassVar[int]
    failed_attempts: int
    green_transitions: int
    recurring_failures: int
    time_to_green_samples: int
    time_to_green: DurationStats
    def __init__(self, failed_attempts: _Optional[int] = ..., green_transitions: _Optional[int] = ..., recurring_failures: _Optional[int] = ..., time_to_green_samples: _Optional[int] = ..., time_to_green: _Optional[_Union[DurationStats, _Mapping]] = ...) -> None: ...

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
    __slots__ = ("phase", "provider", "finding_source", "total_observations", "passed", "failed", "skipped", "degraded", "availability", "failure_rate", "metrics_adopted", "skip_reasons", "classifications", "duration", "worst_scenarios", "security_friction")
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
    SECURITY_FRICTION_FIELD_NUMBER: _ClassVar[int]
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
    security_friction: SecurityFriction
    def __init__(self, phase: _Optional[str] = ..., provider: _Optional[str] = ..., finding_source: _Optional[str] = ..., total_observations: _Optional[int] = ..., passed: _Optional[int] = ..., failed: _Optional[int] = ..., skipped: _Optional[int] = ..., degraded: _Optional[int] = ..., availability: _Optional[float] = ..., failure_rate: _Optional[float] = ..., metrics_adopted: _Optional[int] = ..., skip_reasons: _Optional[_Iterable[_Union[LabeledCount, _Mapping]]] = ..., classifications: _Optional[_Iterable[_Union[LabeledCount, _Mapping]]] = ..., duration: _Optional[_Union[DurationStats, _Mapping]] = ..., worst_scenarios: _Optional[_Iterable[_Union[ScenarioFailureRate, _Mapping]]] = ..., security_friction: _Optional[_Union[SecurityFriction, _Mapping]] = ...) -> None: ...

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
