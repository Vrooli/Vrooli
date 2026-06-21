from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AnalyzeTraceRequest(_message.Message):
    __slots__ = ("scenario", "trace_artifact")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TRACE_ARTIFACT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    trace_artifact: str
    def __init__(self, scenario: _Optional[str] = ..., trace_artifact: _Optional[str] = ...) -> None: ...

class AnalyzeTraceResponse(_message.Message):
    __slots__ = ("scenario", "components", "long_task_ms", "lcp_ms", "fcp_ms", "findings")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    COMPONENTS_FIELD_NUMBER: _ClassVar[int]
    LONG_TASK_MS_FIELD_NUMBER: _ClassVar[int]
    LCP_MS_FIELD_NUMBER: _ClassVar[int]
    FCP_MS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    components: _containers.RepeatedCompositeFieldContainer[ComponentTiming]
    long_task_ms: int
    lcp_ms: int
    fcp_ms: int
    findings: _containers.RepeatedCompositeFieldContainer[PerfFinding]
    def __init__(self, scenario: _Optional[str] = ..., components: _Optional[_Iterable[_Union[ComponentTiming, _Mapping]]] = ..., long_task_ms: _Optional[int] = ..., lcp_ms: _Optional[int] = ..., fcp_ms: _Optional[int] = ..., findings: _Optional[_Iterable[_Union[PerfFinding, _Mapping]]] = ...) -> None: ...

class ComponentTiming(_message.Message):
    __slots__ = ("component", "commit_count", "avg_ms", "max_ms", "definition")
    COMPONENT_FIELD_NUMBER: _ClassVar[int]
    COMMIT_COUNT_FIELD_NUMBER: _ClassVar[int]
    AVG_MS_FIELD_NUMBER: _ClassVar[int]
    MAX_MS_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_FIELD_NUMBER: _ClassVar[int]
    component: str
    commit_count: int
    avg_ms: float
    max_ms: float
    definition: str
    def __init__(self, component: _Optional[str] = ..., commit_count: _Optional[int] = ..., avg_ms: _Optional[float] = ..., max_ms: _Optional[float] = ..., definition: _Optional[str] = ...) -> None: ...

class PerfFinding(_message.Message):
    __slots__ = ("code", "component", "definition", "message", "evidence", "severity")
    CODE_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    code: str
    component: str
    definition: str
    message: str
    evidence: str
    severity: str
    def __init__(self, code: _Optional[str] = ..., component: _Optional[str] = ..., definition: _Optional[str] = ..., message: _Optional[str] = ..., evidence: _Optional[str] = ..., severity: _Optional[str] = ...) -> None: ...

class CompareTracesRequest(_message.Message):
    __slots__ = ("scenario", "baseline_artifact", "candidate_artifact")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    BASELINE_ARTIFACT_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_ARTIFACT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    baseline_artifact: str
    candidate_artifact: str
    def __init__(self, scenario: _Optional[str] = ..., baseline_artifact: _Optional[str] = ..., candidate_artifact: _Optional[str] = ...) -> None: ...

class CompareTracesResponse(_message.Message):
    __slots__ = ("scenario", "components", "long_task_delta_ms", "lcp_delta_ms")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    COMPONENTS_FIELD_NUMBER: _ClassVar[int]
    LONG_TASK_DELTA_MS_FIELD_NUMBER: _ClassVar[int]
    LCP_DELTA_MS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    components: _containers.RepeatedCompositeFieldContainer[ComponentDelta]
    long_task_delta_ms: int
    lcp_delta_ms: int
    def __init__(self, scenario: _Optional[str] = ..., components: _Optional[_Iterable[_Union[ComponentDelta, _Mapping]]] = ..., long_task_delta_ms: _Optional[int] = ..., lcp_delta_ms: _Optional[int] = ...) -> None: ...

class ComponentDelta(_message.Message):
    __slots__ = ("component", "baseline_avg_ms", "candidate_avg_ms", "delta_ms", "baseline_count", "candidate_count", "count_delta", "baseline_max_ms", "candidate_max_ms", "max_delta_ms")
    COMPONENT_FIELD_NUMBER: _ClassVar[int]
    BASELINE_AVG_MS_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_AVG_MS_FIELD_NUMBER: _ClassVar[int]
    DELTA_MS_FIELD_NUMBER: _ClassVar[int]
    BASELINE_COUNT_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_COUNT_FIELD_NUMBER: _ClassVar[int]
    COUNT_DELTA_FIELD_NUMBER: _ClassVar[int]
    BASELINE_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    MAX_DELTA_MS_FIELD_NUMBER: _ClassVar[int]
    component: str
    baseline_avg_ms: float
    candidate_avg_ms: float
    delta_ms: float
    baseline_count: int
    candidate_count: int
    count_delta: int
    baseline_max_ms: float
    candidate_max_ms: float
    max_delta_ms: float
    def __init__(self, component: _Optional[str] = ..., baseline_avg_ms: _Optional[float] = ..., candidate_avg_ms: _Optional[float] = ..., delta_ms: _Optional[float] = ..., baseline_count: _Optional[int] = ..., candidate_count: _Optional[int] = ..., count_delta: _Optional[int] = ..., baseline_max_ms: _Optional[float] = ..., candidate_max_ms: _Optional[float] = ..., max_delta_ms: _Optional[float] = ...) -> None: ...
