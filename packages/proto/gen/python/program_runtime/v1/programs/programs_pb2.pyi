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

class Program(_message.Message):
    __slots__ = ("id", "session_id", "source", "provenance", "status", "stdout", "failure_detail", "failure_shape", "context_bytes", "created_at", "output_limit_bytes", "agent_bytes", "completed_at", "wall_time_millis", "cpu_time_millis", "library_version", "failure_cause")
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
    def __init__(self, id: _Optional[str] = ..., session_id: _Optional[str] = ..., source: _Optional[str] = ..., provenance: _Optional[_Union[Provenance, str]] = ..., status: _Optional[_Union[ProgramStatus, str]] = ..., stdout: _Optional[str] = ..., failure_detail: _Optional[str] = ..., failure_shape: _Optional[str] = ..., context_bytes: _Optional[int] = ..., created_at: _Optional[str] = ..., output_limit_bytes: _Optional[int] = ..., agent_bytes: _Optional[int] = ..., completed_at: _Optional[str] = ..., wall_time_millis: _Optional[int] = ..., cpu_time_millis: _Optional[int] = ..., library_version: _Optional[str] = ..., failure_cause: _Optional[_Union[FailureCause, str]] = ...) -> None: ...

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
    __slots__ = ("session_id", "source", "provenance", "include_materialized", "explain")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_MATERIALIZED_FIELD_NUMBER: _ClassVar[int]
    ASYNC_FIELD_NUMBER: _ClassVar[int]
    EXPLAIN_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    source: str
    provenance: Provenance
    include_materialized: bool
    explain: bool
    def __init__(self, session_id: _Optional[str] = ..., source: _Optional[str] = ..., provenance: _Optional[_Union[Provenance, str]] = ..., include_materialized: _Optional[bool] = ..., explain: _Optional[bool] = ..., **kwargs) -> None: ...

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

class ListProgramsRequest(_message.Message):
    __slots__ = ("session_id", "include_operator")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_OPERATOR_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    include_operator: bool
    def __init__(self, session_id: _Optional[str] = ..., include_operator: _Optional[bool] = ...) -> None: ...

class ListProgramsResponse(_message.Message):
    __slots__ = ("programs",)
    PROGRAMS_FIELD_NUMBER: _ClassVar[int]
    programs: _containers.RepeatedCompositeFieldContainer[Program]
    def __init__(self, programs: _Optional[_Iterable[_Union[Program, _Mapping]]] = ...) -> None: ...

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
    __slots__ = ()
    def __init__(self) -> None: ...

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

class RunAuthoringEvalRequest(_message.Message):
    __slots__ = ("suite", "max_cases")
    SUITE_FIELD_NUMBER: _ClassVar[int]
    MAX_CASES_FIELD_NUMBER: _ClassVar[int]
    suite: str
    max_cases: int
    def __init__(self, suite: _Optional[str] = ..., max_cases: _Optional[int] = ...) -> None: ...

class AuthoringCaseResult(_message.Message):
    __slots__ = ("case_id", "authored", "first_attempt_ok", "cause", "agent_bytes", "model")
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    AUTHORED_FIELD_NUMBER: _ClassVar[int]
    FIRST_ATTEMPT_OK_FIELD_NUMBER: _ClassVar[int]
    CAUSE_FIELD_NUMBER: _ClassVar[int]
    AGENT_BYTES_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    authored: bool
    first_attempt_ok: bool
    cause: str
    agent_bytes: int
    model: str
    def __init__(self, case_id: _Optional[str] = ..., authored: _Optional[bool] = ..., first_attempt_ok: _Optional[bool] = ..., cause: _Optional[str] = ..., agent_bytes: _Optional[int] = ..., model: _Optional[str] = ...) -> None: ...

class RunAuthoringEvalResponse(_message.Message):
    __slots__ = ("suite", "status", "reason", "cases", "met", "missed", "wrong_result", "unavailable", "floor", "results")
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
    def __init__(self, suite: _Optional[str] = ..., status: _Optional[str] = ..., reason: _Optional[str] = ..., cases: _Optional[int] = ..., met: _Optional[int] = ..., missed: _Optional[int] = ..., wrong_result: _Optional[int] = ..., unavailable: _Optional[int] = ..., floor: _Optional[int] = ..., results: _Optional[_Iterable[_Union[AuthoringCaseResult, _Mapping]]] = ...) -> None: ...
