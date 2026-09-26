from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Provenance(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PROVENANCE_UNSPECIFIED: _ClassVar[Provenance]
    PROVENANCE_AGENT: _ClassVar[Provenance]
    PROVENANCE_OPERATOR: _ClassVar[Provenance]
    PROVENANCE_TEST: _ClassVar[Provenance]
    PROVENANCE_REPLAY: _ClassVar[Provenance]

class ProgramStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PROGRAM_STATUS_UNSPECIFIED: _ClassVar[ProgramStatus]
    PROGRAM_STATUS_ACCEPTED: _ClassVar[ProgramStatus]
    PROGRAM_STATUS_RUNNING: _ClassVar[ProgramStatus]
    PROGRAM_STATUS_SUCCEEDED: _ClassVar[ProgramStatus]
    PROGRAM_STATUS_FAILED: _ClassVar[ProgramStatus]
    PROGRAM_STATUS_CANCELLED: _ClassVar[ProgramStatus]

class FailureCause(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FAILURE_CAUSE_UNSPECIFIED: _ClassVar[FailureCause]
    FAILURE_CAUSE_UNRESOLVED_NAME: _ClassVar[FailureCause]
    FAILURE_CAUSE_UNKNOWN_FIELD: _ClassVar[FailureCause]
    FAILURE_CAUSE_AMBIGUOUS_RESPONSE: _ClassVar[FailureCause]
    FAILURE_CAUSE_UNREACHABLE_SCENARIO: _ClassVar[FailureCause]
    FAILURE_CAUSE_REFUSED_NO_GRANT: _ClassVar[FailureCause]
    FAILURE_CAUSE_REFUSED_NOT_RUN_ELIGIBLE: _ClassVar[FailureCause]
    FAILURE_CAUSE_INFERENCE_SPEND_EXCEEDED: _ClassVar[FailureCause]
    FAILURE_CAUSE_DELEGATED_RUN_SPEND_EXCEEDED: _ClassVar[FailureCause]
    FAILURE_CAUSE_DEADLINE_EXCEEDED: _ClassVar[FailureCause]
    FAILURE_CAUSE_KERNEL_SYNTAX: _ClassVar[FailureCause]
    FAILURE_CAUSE_KERNEL_RUNTIME: _ClassVar[FailureCause]
    FAILURE_CAUSE_BRIDGE_TRANSPORT: _ClassVar[FailureCause]
    FAILURE_CAUSE_UNCLASSIFIED: _ClassVar[FailureCause]
    FAILURE_CAUSE_PROTECTED_NAME_MISUSE: _ClassVar[FailureCause]
    FAILURE_CAUSE_RUNTIME_INTERRUPTED: _ClassVar[FailureCause]
PROVENANCE_UNSPECIFIED: Provenance
PROVENANCE_AGENT: Provenance
PROVENANCE_OPERATOR: Provenance
PROVENANCE_TEST: Provenance
PROVENANCE_REPLAY: Provenance
PROGRAM_STATUS_UNSPECIFIED: ProgramStatus
PROGRAM_STATUS_ACCEPTED: ProgramStatus
PROGRAM_STATUS_RUNNING: ProgramStatus
PROGRAM_STATUS_SUCCEEDED: ProgramStatus
PROGRAM_STATUS_FAILED: ProgramStatus
PROGRAM_STATUS_CANCELLED: ProgramStatus
FAILURE_CAUSE_UNSPECIFIED: FailureCause
FAILURE_CAUSE_UNRESOLVED_NAME: FailureCause
FAILURE_CAUSE_UNKNOWN_FIELD: FailureCause
FAILURE_CAUSE_AMBIGUOUS_RESPONSE: FailureCause
FAILURE_CAUSE_UNREACHABLE_SCENARIO: FailureCause
FAILURE_CAUSE_REFUSED_NO_GRANT: FailureCause
FAILURE_CAUSE_REFUSED_NOT_RUN_ELIGIBLE: FailureCause
FAILURE_CAUSE_INFERENCE_SPEND_EXCEEDED: FailureCause
FAILURE_CAUSE_DELEGATED_RUN_SPEND_EXCEEDED: FailureCause
FAILURE_CAUSE_DEADLINE_EXCEEDED: FailureCause
FAILURE_CAUSE_KERNEL_SYNTAX: FailureCause
FAILURE_CAUSE_KERNEL_RUNTIME: FailureCause
FAILURE_CAUSE_BRIDGE_TRANSPORT: FailureCause
FAILURE_CAUSE_UNCLASSIFIED: FailureCause
FAILURE_CAUSE_PROTECTED_NAME_MISUSE: FailureCause
FAILURE_CAUSE_RUNTIME_INTERRUPTED: FailureCause

class Caller(_message.Message):
    __slots__ = ("run_id", "agent_profile", "skill_id", "harness")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_PROFILE_FIELD_NUMBER: _ClassVar[int]
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    HARNESS_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    agent_profile: str
    skill_id: str
    harness: str
    def __init__(self, run_id: _Optional[str] = ..., agent_profile: _Optional[str] = ..., skill_id: _Optional[str] = ..., harness: _Optional[str] = ...) -> None: ...

class Program(_message.Message):
    __slots__ = ("id", "session_id", "source", "provenance", "status", "stdout", "failure_detail", "failure_shape", "context_bytes", "created_at", "output_limit_bytes", "agent_bytes", "completed_at", "wall_time_millis", "cpu_time_millis", "library_version", "failure_cause", "program_name", "program_digest", "caller_run_id", "caller_agent_profile", "caller_skill_id", "caller_harness", "learning_json")
    ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STDOUT_FIELD_NUMBER: _ClassVar[int]
    FAILURE_DETAIL_FIELD_NUMBER: _ClassVar[int]
    FAILURE_SHAPE_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_BYTES_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_LIMIT_BYTES_FIELD_NUMBER: _ClassVar[int]
    AGENT_BYTES_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    WALL_TIME_MILLIS_FIELD_NUMBER: _ClassVar[int]
    CPU_TIME_MILLIS_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_VERSION_FIELD_NUMBER: _ClassVar[int]
    FAILURE_CAUSE_FIELD_NUMBER: _ClassVar[int]
    PROGRAM_NAME_FIELD_NUMBER: _ClassVar[int]
    PROGRAM_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CALLER_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    CALLER_AGENT_PROFILE_FIELD_NUMBER: _ClassVar[int]
    CALLER_SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    CALLER_HARNESS_FIELD_NUMBER: _ClassVar[int]
    LEARNING_JSON_FIELD_NUMBER: _ClassVar[int]
    id: str
    session_id: str
    source: str
    provenance: Provenance
    status: ProgramStatus
    stdout: str
    failure_detail: str
    failure_shape: str
    context_bytes: int
    created_at: str
    output_limit_bytes: int
    agent_bytes: int
    completed_at: str
    wall_time_millis: int
    cpu_time_millis: int
    library_version: str
    failure_cause: FailureCause
    program_name: str
    program_digest: str
    caller_run_id: str
    caller_agent_profile: str
    caller_skill_id: str
    caller_harness: str
    learning_json: str
    def __init__(self, id: _Optional[str] = ..., session_id: _Optional[str] = ..., source: _Optional[str] = ..., provenance: _Optional[_Union[Provenance, str]] = ..., status: _Optional[_Union[ProgramStatus, str]] = ..., stdout: _Optional[str] = ..., failure_detail: _Optional[str] = ..., failure_shape: _Optional[str] = ..., context_bytes: _Optional[int] = ..., created_at: _Optional[str] = ..., output_limit_bytes: _Optional[int] = ..., agent_bytes: _Optional[int] = ..., completed_at: _Optional[str] = ..., wall_time_millis: _Optional[int] = ..., cpu_time_millis: _Optional[int] = ..., library_version: _Optional[str] = ..., failure_cause: _Optional[_Union[FailureCause, str]] = ..., program_name: _Optional[str] = ..., program_digest: _Optional[str] = ..., caller_run_id: _Optional[str] = ..., caller_agent_profile: _Optional[str] = ..., caller_skill_id: _Optional[str] = ..., caller_harness: _Optional[str] = ..., learning_json: _Optional[str] = ...) -> None: ...

class Diagnostic(_message.Message):
    __slots__ = ("severity", "line", "name", "message", "nearest_match")
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    LINE_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    NEAREST_MATCH_FIELD_NUMBER: _ClassVar[int]
    severity: str
    line: int
    name: str
    message: str
    nearest_match: str
    def __init__(self, severity: _Optional[str] = ..., line: _Optional[int] = ..., name: _Optional[str] = ..., message: _Optional[str] = ..., nearest_match: _Optional[str] = ...) -> None: ...

class SubmitProgramRequest(_message.Message):
    __slots__ = ("session_id", "source", "provenance", "include_materialized", "explain", "caller")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_MATERIALIZED_FIELD_NUMBER: _ClassVar[int]
    ASYNC_FIELD_NUMBER: _ClassVar[int]
    EXPLAIN_FIELD_NUMBER: _ClassVar[int]
    CALLER_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    source: str
    provenance: Provenance
    include_materialized: bool
    explain: bool
    caller: Caller
    def __init__(self, session_id: _Optional[str] = ..., source: _Optional[str] = ..., provenance: _Optional[_Union[Provenance, str]] = ..., include_materialized: _Optional[bool] = ..., explain: _Optional[bool] = ..., caller: _Optional[_Union[Caller, _Mapping]] = ..., **kwargs) -> None: ...

class SubmitProgramResponse(_message.Message):
    __slots__ = ("program", "diagnostics")
    PROGRAM_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTICS_FIELD_NUMBER: _ClassVar[int]
    program: Program
    diagnostics: _containers.RepeatedCompositeFieldContainer[Diagnostic]
    def __init__(self, program: _Optional[_Union[Program, _Mapping]] = ..., diagnostics: _Optional[_Iterable[_Union[Diagnostic, _Mapping]]] = ...) -> None: ...

class GetProgramRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetProgramResponse(_message.Message):
    __slots__ = ("program",)
    PROGRAM_FIELD_NUMBER: _ClassVar[int]
    program: Program
    def __init__(self, program: _Optional[_Union[Program, _Mapping]] = ...) -> None: ...

class WaitForProgramRequest(_message.Message):
    __slots__ = ("id", "timeout_millis")
    ID_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_MILLIS_FIELD_NUMBER: _ClassVar[int]
    id: str
    timeout_millis: int
    def __init__(self, id: _Optional[str] = ..., timeout_millis: _Optional[int] = ...) -> None: ...

class WaitForProgramResponse(_message.Message):
    __slots__ = ("program", "terminal", "waited_millis")
    PROGRAM_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_FIELD_NUMBER: _ClassVar[int]
    WAITED_MILLIS_FIELD_NUMBER: _ClassVar[int]
    program: Program
    terminal: bool
    waited_millis: int
    def __init__(self, program: _Optional[_Union[Program, _Mapping]] = ..., terminal: _Optional[bool] = ..., waited_millis: _Optional[int] = ...) -> None: ...

class ListProgramsRequest(_message.Message):
    __slots__ = ("session_id", "include_operator", "provenance", "since_seconds", "until", "limit", "program_name", "program_digest")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_OPERATOR_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    SINCE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    UNTIL_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    PROGRAM_NAME_FIELD_NUMBER: _ClassVar[int]
    PROGRAM_DIGEST_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    include_operator: bool
    provenance: str
    since_seconds: int
    until: str
    limit: int
    program_name: str
    program_digest: str
    def __init__(self, session_id: _Optional[str] = ..., include_operator: _Optional[bool] = ..., provenance: _Optional[str] = ..., since_seconds: _Optional[int] = ..., until: _Optional[str] = ..., limit: _Optional[int] = ..., program_name: _Optional[str] = ..., program_digest: _Optional[str] = ...) -> None: ...

class ListProgramsResponse(_message.Message):
    __slots__ = ("programs",)
    PROGRAMS_FIELD_NUMBER: _ClassVar[int]
    programs: _containers.RepeatedCompositeFieldContainer[Program]
    def __init__(self, programs: _Optional[_Iterable[_Union[Program, _Mapping]]] = ...) -> None: ...

class PortfolioStatsRequest(_message.Message):
    __slots__ = ("window_days", "scenario", "provenance", "include_ad_hoc")
    WINDOW_DAYS_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_AD_HOC_FIELD_NUMBER: _ClassVar[int]
    window_days: int
    scenario: str
    provenance: str
    include_ad_hoc: bool
    def __init__(self, window_days: _Optional[int] = ..., scenario: _Optional[str] = ..., provenance: _Optional[str] = ..., include_ad_hoc: _Optional[bool] = ...) -> None: ...

class ProgramPortfolioRow(_message.Message):
    __slots__ = ("name", "scenario", "runs", "succeeded", "failed", "success_rate", "p50_millis", "p95_millis", "declared_wall_millis", "budget_pressure", "agent_runs", "operator_runs", "test_runs", "distinct_callers", "distinct_days", "first_seen", "last_seen", "top_failure_cause", "called_by_programs", "current_digest", "digest_drifted")
    NAME_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RUNS_FIELD_NUMBER: _ClassVar[int]
    SUCCEEDED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_RATE_FIELD_NUMBER: _ClassVar[int]
    P50_MILLIS_FIELD_NUMBER: _ClassVar[int]
    P95_MILLIS_FIELD_NUMBER: _ClassVar[int]
    DECLARED_WALL_MILLIS_FIELD_NUMBER: _ClassVar[int]
    BUDGET_PRESSURE_FIELD_NUMBER: _ClassVar[int]
    AGENT_RUNS_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_RUNS_FIELD_NUMBER: _ClassVar[int]
    TEST_RUNS_FIELD_NUMBER: _ClassVar[int]
    DISTINCT_CALLERS_FIELD_NUMBER: _ClassVar[int]
    DISTINCT_DAYS_FIELD_NUMBER: _ClassVar[int]
    FIRST_SEEN_FIELD_NUMBER: _ClassVar[int]
    LAST_SEEN_FIELD_NUMBER: _ClassVar[int]
    TOP_FAILURE_CAUSE_FIELD_NUMBER: _ClassVar[int]
    CALLED_BY_PROGRAMS_FIELD_NUMBER: _ClassVar[int]
    CURRENT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    DIGEST_DRIFTED_FIELD_NUMBER: _ClassVar[int]
    name: str
    scenario: str
    runs: int
    succeeded: int
    failed: int
    success_rate: float
    p50_millis: int
    p95_millis: int
    declared_wall_millis: int
    budget_pressure: float
    agent_runs: int
    operator_runs: int
    test_runs: int
    distinct_callers: int
    distinct_days: int
    first_seen: str
    last_seen: str
    top_failure_cause: str
    called_by_programs: int
    current_digest: str
    digest_drifted: bool
    def __init__(self, name: _Optional[str] = ..., scenario: _Optional[str] = ..., runs: _Optional[int] = ..., succeeded: _Optional[int] = ..., failed: _Optional[int] = ..., success_rate: _Optional[float] = ..., p50_millis: _Optional[int] = ..., p95_millis: _Optional[int] = ..., declared_wall_millis: _Optional[int] = ..., budget_pressure: _Optional[float] = ..., agent_runs: _Optional[int] = ..., operator_runs: _Optional[int] = ..., test_runs: _Optional[int] = ..., distinct_callers: _Optional[int] = ..., distinct_days: _Optional[int] = ..., first_seen: _Optional[str] = ..., last_seen: _Optional[str] = ..., top_failure_cause: _Optional[str] = ..., called_by_programs: _Optional[int] = ..., current_digest: _Optional[str] = ..., digest_drifted: _Optional[bool] = ...) -> None: ...

class PortfolioStatsResponse(_message.Message):
    __slots__ = ("rows", "programs_declared", "programs_executed", "never_executed", "rows_without_identity", "unattributed_agent_runs", "window_start", "window_end", "contract_index_reason")
    ROWS_FIELD_NUMBER: _ClassVar[int]
    PROGRAMS_DECLARED_FIELD_NUMBER: _ClassVar[int]
    PROGRAMS_EXECUTED_FIELD_NUMBER: _ClassVar[int]
    NEVER_EXECUTED_FIELD_NUMBER: _ClassVar[int]
    ROWS_WITHOUT_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    UNATTRIBUTED_AGENT_RUNS_FIELD_NUMBER: _ClassVar[int]
    WINDOW_START_FIELD_NUMBER: _ClassVar[int]
    WINDOW_END_FIELD_NUMBER: _ClassVar[int]
    CONTRACT_INDEX_REASON_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[ProgramPortfolioRow]
    programs_declared: int
    programs_executed: int
    never_executed: _containers.RepeatedScalarFieldContainer[str]
    rows_without_identity: int
    unattributed_agent_runs: int
    window_start: str
    window_end: str
    contract_index_reason: str
    def __init__(self, rows: _Optional[_Iterable[_Union[ProgramPortfolioRow, _Mapping]]] = ..., programs_declared: _Optional[int] = ..., programs_executed: _Optional[int] = ..., never_executed: _Optional[_Iterable[str]] = ..., rows_without_identity: _Optional[int] = ..., unattributed_agent_runs: _Optional[int] = ..., window_start: _Optional[str] = ..., window_end: _Optional[str] = ..., contract_index_reason: _Optional[str] = ...) -> None: ...

class MineFailuresRequest(_message.Message):
    __slots__ = ("include_operator",)
    INCLUDE_OPERATOR_FIELD_NUMBER: _ClassVar[int]
    include_operator: bool
    def __init__(self, include_operator: _Optional[bool] = ...) -> None: ...

class FailureShape(_message.Message):
    __slots__ = ("shape", "count", "first_seen", "last_seen", "sample_program_id")
    SHAPE_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    FIRST_SEEN_FIELD_NUMBER: _ClassVar[int]
    LAST_SEEN_FIELD_NUMBER: _ClassVar[int]
    SAMPLE_PROGRAM_ID_FIELD_NUMBER: _ClassVar[int]
    shape: str
    count: int
    first_seen: str
    last_seen: str
    sample_program_id: str
    def __init__(self, shape: _Optional[str] = ..., count: _Optional[int] = ..., first_seen: _Optional[str] = ..., last_seen: _Optional[str] = ..., sample_program_id: _Optional[str] = ...) -> None: ...

class MineFailuresResponse(_message.Message):
    __slots__ = ("shapes", "count")
    SHAPES_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    shapes: _containers.RepeatedCompositeFieldContainer[FailureShape]
    count: int
    def __init__(self, shapes: _Optional[_Iterable[_Union[FailureShape, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class MineRefusalsRequest(_message.Message):
    __slots__ = ("include_operator",)
    INCLUDE_OPERATOR_FIELD_NUMBER: _ClassVar[int]
    include_operator: bool
    def __init__(self, include_operator: _Optional[bool] = ...) -> None: ...

class RefusalShape(_message.Message):
    __slots__ = ("binding_id", "reason", "count", "last_seen")
    BINDING_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    LAST_SEEN_FIELD_NUMBER: _ClassVar[int]
    binding_id: str
    reason: str
    count: int
    last_seen: str
    def __init__(self, binding_id: _Optional[str] = ..., reason: _Optional[str] = ..., count: _Optional[int] = ..., last_seen: _Optional[str] = ...) -> None: ...

class MineRefusalsResponse(_message.Message):
    __slots__ = ("shapes", "count")
    SHAPES_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    shapes: _containers.RepeatedCompositeFieldContainer[RefusalShape]
    count: int
    def __init__(self, shapes: _Optional[_Iterable[_Union[RefusalShape, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class MineUnresolvedBindingsRequest(_message.Message):
    __slots__ = ("include_operator",)
    INCLUDE_OPERATOR_FIELD_NUMBER: _ClassVar[int]
    include_operator: bool
    def __init__(self, include_operator: _Optional[bool] = ...) -> None: ...

class UnresolvedBindingShape(_message.Message):
    __slots__ = ("attempted_name", "count", "last_seen")
    ATTEMPTED_NAME_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    LAST_SEEN_FIELD_NUMBER: _ClassVar[int]
    attempted_name: str
    count: int
    last_seen: str
    def __init__(self, attempted_name: _Optional[str] = ..., count: _Optional[int] = ..., last_seen: _Optional[str] = ...) -> None: ...

class MineUnresolvedBindingsResponse(_message.Message):
    __slots__ = ("shapes", "count")
    SHAPES_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    shapes: _containers.RepeatedCompositeFieldContainer[UnresolvedBindingShape]
    count: int
    def __init__(self, shapes: _Optional[_Iterable[_Union[UnresolvedBindingShape, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class GovernanceShareRequest(_message.Message):
    __slots__ = ("window_seconds", "include_operator")
    WINDOW_SECONDS_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_OPERATOR_FIELD_NUMBER: _ClassVar[int]
    window_seconds: int
    include_operator: bool
    def __init__(self, window_seconds: _Optional[int] = ..., include_operator: _Optional[bool] = ...) -> None: ...

class ObservedCommand(_message.Message):
    __slots__ = ("attempted_name", "count", "last_seen")
    ATTEMPTED_NAME_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    LAST_SEEN_FIELD_NUMBER: _ClassVar[int]
    attempted_name: str
    count: int
    last_seen: str
    def __init__(self, attempted_name: _Optional[str] = ..., count: _Optional[int] = ..., last_seen: _Optional[str] = ...) -> None: ...

class GovernanceShareResponse(_message.Message):
    __slots__ = ("governed_calls", "observed_calls", "governed_share", "window_seconds", "window_start", "window_end", "observed_commands")
    GOVERNED_CALLS_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_CALLS_FIELD_NUMBER: _ClassVar[int]
    GOVERNED_SHARE_FIELD_NUMBER: _ClassVar[int]
    WINDOW_SECONDS_FIELD_NUMBER: _ClassVar[int]
    WINDOW_START_FIELD_NUMBER: _ClassVar[int]
    WINDOW_END_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_COMMANDS_FIELD_NUMBER: _ClassVar[int]
    governed_calls: int
    observed_calls: int
    governed_share: float
    window_seconds: int
    window_start: str
    window_end: str
    observed_commands: _containers.RepeatedCompositeFieldContainer[ObservedCommand]
    def __init__(self, governed_calls: _Optional[int] = ..., observed_calls: _Optional[int] = ..., governed_share: _Optional[float] = ..., window_seconds: _Optional[int] = ..., window_start: _Optional[str] = ..., window_end: _Optional[str] = ..., observed_commands: _Optional[_Iterable[_Union[ObservedCommand, _Mapping]]] = ...) -> None: ...

class RunAuthoringEvalRequest(_message.Message):
    __slots__ = ("suite", "max_cases", "no_gate")
    SUITE_FIELD_NUMBER: _ClassVar[int]
    MAX_CASES_FIELD_NUMBER: _ClassVar[int]
    NO_GATE_FIELD_NUMBER: _ClassVar[int]
    suite: str
    max_cases: int
    no_gate: bool
    def __init__(self, suite: _Optional[str] = ..., max_cases: _Optional[int] = ..., no_gate: _Optional[bool] = ...) -> None: ...

class AuthoringCaseResult(_message.Message):
    __slots__ = ("case_id", "authored", "first_attempt_ok", "cause", "agent_bytes", "model", "rule_id", "failure_detail")
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    AUTHORED_FIELD_NUMBER: _ClassVar[int]
    FIRST_ATTEMPT_OK_FIELD_NUMBER: _ClassVar[int]
    CAUSE_FIELD_NUMBER: _ClassVar[int]
    AGENT_BYTES_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    FAILURE_DETAIL_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    authored: bool
    first_attempt_ok: bool
    cause: str
    agent_bytes: int
    model: str
    rule_id: str
    failure_detail: str
    def __init__(self, case_id: _Optional[str] = ..., authored: _Optional[bool] = ..., first_attempt_ok: _Optional[bool] = ..., cause: _Optional[str] = ..., agent_bytes: _Optional[int] = ..., model: _Optional[str] = ..., rule_id: _Optional[str] = ..., failure_detail: _Optional[str] = ...) -> None: ...

class AuthoringRuleMiss(_message.Message):
    __slots__ = ("rule_id", "count")
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    rule_id: str
    count: int
    def __init__(self, rule_id: _Optional[str] = ..., count: _Optional[int] = ...) -> None: ...

class RunAuthoringEvalResponse(_message.Message):
    __slots__ = ("suite", "status", "reason", "cases", "met", "missed", "wrong_result", "unavailable", "floor", "results", "not_attempted", "harness_stamp", "rule_misses", "floor_met")
    SUITE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    CASES_FIELD_NUMBER: _ClassVar[int]
    MET_FIELD_NUMBER: _ClassVar[int]
    MISSED_FIELD_NUMBER: _ClassVar[int]
    WRONG_RESULT_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_FIELD_NUMBER: _ClassVar[int]
    FLOOR_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    NOT_ATTEMPTED_FIELD_NUMBER: _ClassVar[int]
    HARNESS_STAMP_FIELD_NUMBER: _ClassVar[int]
    RULE_MISSES_FIELD_NUMBER: _ClassVar[int]
    FLOOR_MET_FIELD_NUMBER: _ClassVar[int]
    suite: str
    status: str
    reason: str
    cases: int
    met: int
    missed: int
    wrong_result: int
    unavailable: int
    floor: int
    results: _containers.RepeatedCompositeFieldContainer[AuthoringCaseResult]
    not_attempted: int
    harness_stamp: str
    rule_misses: _containers.RepeatedCompositeFieldContainer[AuthoringRuleMiss]
    floor_met: bool
    def __init__(self, suite: _Optional[str] = ..., status: _Optional[str] = ..., reason: _Optional[str] = ..., cases: _Optional[int] = ..., met: _Optional[int] = ..., missed: _Optional[int] = ..., wrong_result: _Optional[int] = ..., unavailable: _Optional[int] = ..., floor: _Optional[int] = ..., results: _Optional[_Iterable[_Union[AuthoringCaseResult, _Mapping]]] = ..., not_attempted: _Optional[int] = ..., harness_stamp: _Optional[str] = ..., rule_misses: _Optional[_Iterable[_Union[AuthoringRuleMiss, _Mapping]]] = ..., floor_met: _Optional[bool] = ...) -> None: ...

class RunDiscoveryEvalRequest(_message.Message):
    __slots__ = ("suite", "mode", "max_cases", "no_gate")
    SUITE_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    MAX_CASES_FIELD_NUMBER: _ClassVar[int]
    NO_GATE_FIELD_NUMBER: _ClassVar[int]
    suite: str
    mode: str
    max_cases: int
    no_gate: bool
    def __init__(self, suite: _Optional[str] = ..., mode: _Optional[str] = ..., max_cases: _Optional[int] = ..., no_gate: _Optional[bool] = ...) -> None: ...

class DiscoveryCaseResult(_message.Message):
    __slots__ = ("case_id", "intent", "expected_binding_id", "selected_binding_id", "met", "null_verdict", "wrong_selection", "reason")
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    INTENT_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_BINDING_ID_FIELD_NUMBER: _ClassVar[int]
    SELECTED_BINDING_ID_FIELD_NUMBER: _ClassVar[int]
    MET_FIELD_NUMBER: _ClassVar[int]
    NULL_VERDICT_FIELD_NUMBER: _ClassVar[int]
    WRONG_SELECTION_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    intent: str
    expected_binding_id: str
    selected_binding_id: str
    met: bool
    null_verdict: bool
    wrong_selection: bool
    reason: str
    def __init__(self, case_id: _Optional[str] = ..., intent: _Optional[str] = ..., expected_binding_id: _Optional[str] = ..., selected_binding_id: _Optional[str] = ..., met: _Optional[bool] = ..., null_verdict: _Optional[bool] = ..., wrong_selection: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class RunDiscoveryEvalResponse(_message.Message):
    __slots__ = ("suite", "status", "reason", "cases", "met", "missed", "wrong_selection", "null_verdict", "floor", "floor_reason", "results", "floor_met")
    SUITE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    CASES_FIELD_NUMBER: _ClassVar[int]
    MET_FIELD_NUMBER: _ClassVar[int]
    MISSED_FIELD_NUMBER: _ClassVar[int]
    WRONG_SELECTION_FIELD_NUMBER: _ClassVar[int]
    NULL_VERDICT_FIELD_NUMBER: _ClassVar[int]
    FLOOR_FIELD_NUMBER: _ClassVar[int]
    FLOOR_REASON_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    FLOOR_MET_FIELD_NUMBER: _ClassVar[int]
    suite: str
    status: str
    reason: str
    cases: int
    met: int
    missed: int
    wrong_selection: int
    null_verdict: int
    floor: int
    floor_reason: str
    results: _containers.RepeatedCompositeFieldContainer[DiscoveryCaseResult]
    floor_met: bool
    def __init__(self, suite: _Optional[str] = ..., status: _Optional[str] = ..., reason: _Optional[str] = ..., cases: _Optional[int] = ..., met: _Optional[int] = ..., missed: _Optional[int] = ..., wrong_selection: _Optional[int] = ..., null_verdict: _Optional[int] = ..., floor: _Optional[int] = ..., floor_reason: _Optional[str] = ..., results: _Optional[_Iterable[_Union[DiscoveryCaseResult, _Mapping]]] = ..., floor_met: _Optional[bool] = ...) -> None: ...

class LearningFinding(_message.Message):
    __slots__ = ("finding_id", "owner", "state", "dimension", "correction", "evidence", "updated_at")
    FINDING_ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    DIMENSION_FIELD_NUMBER: _ClassVar[int]
    CORRECTION_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    finding_id: str
    owner: str
    state: str
    dimension: str
    correction: str
    evidence: _containers.RepeatedScalarFieldContainer[str]
    updated_at: str
    def __init__(self, finding_id: _Optional[str] = ..., owner: _Optional[str] = ..., state: _Optional[str] = ..., dimension: _Optional[str] = ..., correction: _Optional[str] = ..., evidence: _Optional[_Iterable[str]] = ..., updated_at: _Optional[str] = ...) -> None: ...

class ListLearningFindingsRequest(_message.Message):
    __slots__ = ("owner", "limit")
    OWNER_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    owner: str
    limit: int
    def __init__(self, owner: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListLearningFindingsResponse(_message.Message):
    __slots__ = ("findings", "truncated")
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    findings: _containers.RepeatedCompositeFieldContainer[LearningFinding]
    truncated: bool
    def __init__(self, findings: _Optional[_Iterable[_Union[LearningFinding, _Mapping]]] = ..., truncated: _Optional[bool] = ...) -> None: ...
