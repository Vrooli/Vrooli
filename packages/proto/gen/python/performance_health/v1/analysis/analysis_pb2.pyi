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
    __slots__ = ("scenario", "components", "long_task_ms", "lcp_ms", "fcp_ms", "findings", "frame_summary", "browser_work", "input_events", "cls", "response_end_ms", "dom_interactive_ms", "dom_content_loaded_ms", "load_event_end_ms", "navigation_type")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    COMPONENTS_FIELD_NUMBER: _ClassVar[int]
    LONG_TASK_MS_FIELD_NUMBER: _ClassVar[int]
    LCP_MS_FIELD_NUMBER: _ClassVar[int]
    FCP_MS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    FRAME_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    BROWSER_WORK_FIELD_NUMBER: _ClassVar[int]
    INPUT_EVENTS_FIELD_NUMBER: _ClassVar[int]
    CLS_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_END_MS_FIELD_NUMBER: _ClassVar[int]
    DOM_INTERACTIVE_MS_FIELD_NUMBER: _ClassVar[int]
    DOM_CONTENT_LOADED_MS_FIELD_NUMBER: _ClassVar[int]
    LOAD_EVENT_END_MS_FIELD_NUMBER: _ClassVar[int]
    NAVIGATION_TYPE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    components: _containers.RepeatedCompositeFieldContainer[ComponentTiming]
    long_task_ms: int
    lcp_ms: int
    fcp_ms: int
    findings: _containers.RepeatedCompositeFieldContainer[PerfFinding]
    frame_summary: FrameSummary
    browser_work: _containers.RepeatedCompositeFieldContainer[EventSummary]
    input_events: _containers.RepeatedCompositeFieldContainer[EventSummary]
    cls: float
    response_end_ms: int
    dom_interactive_ms: int
    dom_content_loaded_ms: int
    load_event_end_ms: int
    navigation_type: str
    def __init__(self, scenario: _Optional[str] = ..., components: _Optional[_Iterable[_Union[ComponentTiming, _Mapping]]] = ..., long_task_ms: _Optional[int] = ..., lcp_ms: _Optional[int] = ..., fcp_ms: _Optional[int] = ..., findings: _Optional[_Iterable[_Union[PerfFinding, _Mapping]]] = ..., frame_summary: _Optional[_Union[FrameSummary, _Mapping]] = ..., browser_work: _Optional[_Iterable[_Union[EventSummary, _Mapping]]] = ..., input_events: _Optional[_Iterable[_Union[EventSummary, _Mapping]]] = ..., cls: _Optional[float] = ..., response_end_ms: _Optional[int] = ..., dom_interactive_ms: _Optional[int] = ..., dom_content_loaded_ms: _Optional[int] = ..., load_event_end_ms: _Optional[int] = ..., navigation_type: _Optional[str] = ...) -> None: ...

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

class FrameSummary(_message.Message):
    __slots__ = ("trace_duration_ms", "begin_frame_count", "drawn_frame_count", "dropped_frame_count", "approx_drawn_fps", "dropped_frame_rate")
    TRACE_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    BEGIN_FRAME_COUNT_FIELD_NUMBER: _ClassVar[int]
    DRAWN_FRAME_COUNT_FIELD_NUMBER: _ClassVar[int]
    DROPPED_FRAME_COUNT_FIELD_NUMBER: _ClassVar[int]
    APPROX_DRAWN_FPS_FIELD_NUMBER: _ClassVar[int]
    DROPPED_FRAME_RATE_FIELD_NUMBER: _ClassVar[int]
    trace_duration_ms: float
    begin_frame_count: int
    drawn_frame_count: int
    dropped_frame_count: int
    approx_drawn_fps: float
    dropped_frame_rate: float
    def __init__(self, trace_duration_ms: _Optional[float] = ..., begin_frame_count: _Optional[int] = ..., drawn_frame_count: _Optional[int] = ..., dropped_frame_count: _Optional[int] = ..., approx_drawn_fps: _Optional[float] = ..., dropped_frame_rate: _Optional[float] = ...) -> None: ...

class EventSummary(_message.Message):
    __slots__ = ("name", "count", "total_ms", "max_ms", "avg_ms")
    NAME_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_MS_FIELD_NUMBER: _ClassVar[int]
    MAX_MS_FIELD_NUMBER: _ClassVar[int]
    AVG_MS_FIELD_NUMBER: _ClassVar[int]
    name: str
    count: int
    total_ms: float
    max_ms: float
    avg_ms: float
    def __init__(self, name: _Optional[str] = ..., count: _Optional[int] = ..., total_ms: _Optional[float] = ..., max_ms: _Optional[float] = ..., avg_ms: _Optional[float] = ...) -> None: ...

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
    __slots__ = ("scenario", "components", "long_task_delta_ms", "lcp_delta_ms", "frame_delta", "browser_work", "input_events")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    COMPONENTS_FIELD_NUMBER: _ClassVar[int]
    LONG_TASK_DELTA_MS_FIELD_NUMBER: _ClassVar[int]
    LCP_DELTA_MS_FIELD_NUMBER: _ClassVar[int]
    FRAME_DELTA_FIELD_NUMBER: _ClassVar[int]
    BROWSER_WORK_FIELD_NUMBER: _ClassVar[int]
    INPUT_EVENTS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    components: _containers.RepeatedCompositeFieldContainer[ComponentDelta]
    long_task_delta_ms: int
    lcp_delta_ms: int
    frame_delta: FrameDelta
    browser_work: _containers.RepeatedCompositeFieldContainer[EventDelta]
    input_events: _containers.RepeatedCompositeFieldContainer[EventDelta]
    def __init__(self, scenario: _Optional[str] = ..., components: _Optional[_Iterable[_Union[ComponentDelta, _Mapping]]] = ..., long_task_delta_ms: _Optional[int] = ..., lcp_delta_ms: _Optional[int] = ..., frame_delta: _Optional[_Union[FrameDelta, _Mapping]] = ..., browser_work: _Optional[_Iterable[_Union[EventDelta, _Mapping]]] = ..., input_events: _Optional[_Iterable[_Union[EventDelta, _Mapping]]] = ...) -> None: ...

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

class FrameDelta(_message.Message):
    __slots__ = ("trace_duration_delta_ms", "begin_frame_count_delta", "drawn_frame_count_delta", "dropped_frame_count_delta", "approx_drawn_fps_delta", "dropped_frame_rate_delta")
    TRACE_DURATION_DELTA_MS_FIELD_NUMBER: _ClassVar[int]
    BEGIN_FRAME_COUNT_DELTA_FIELD_NUMBER: _ClassVar[int]
    DRAWN_FRAME_COUNT_DELTA_FIELD_NUMBER: _ClassVar[int]
    DROPPED_FRAME_COUNT_DELTA_FIELD_NUMBER: _ClassVar[int]
    APPROX_DRAWN_FPS_DELTA_FIELD_NUMBER: _ClassVar[int]
    DROPPED_FRAME_RATE_DELTA_FIELD_NUMBER: _ClassVar[int]
    trace_duration_delta_ms: float
    begin_frame_count_delta: int
    drawn_frame_count_delta: int
    dropped_frame_count_delta: int
    approx_drawn_fps_delta: float
    dropped_frame_rate_delta: float
    def __init__(self, trace_duration_delta_ms: _Optional[float] = ..., begin_frame_count_delta: _Optional[int] = ..., drawn_frame_count_delta: _Optional[int] = ..., dropped_frame_count_delta: _Optional[int] = ..., approx_drawn_fps_delta: _Optional[float] = ..., dropped_frame_rate_delta: _Optional[float] = ...) -> None: ...

class EventDelta(_message.Message):
    __slots__ = ("name", "baseline_count", "candidate_count", "count_delta", "baseline_total_ms", "candidate_total_ms", "total_delta_ms", "baseline_max_ms", "candidate_max_ms", "max_delta_ms", "baseline_avg_ms", "candidate_avg_ms", "avg_delta_ms")
    NAME_FIELD_NUMBER: _ClassVar[int]
    BASELINE_COUNT_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_COUNT_FIELD_NUMBER: _ClassVar[int]
    COUNT_DELTA_FIELD_NUMBER: _ClassVar[int]
    BASELINE_TOTAL_MS_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_TOTAL_MS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_DELTA_MS_FIELD_NUMBER: _ClassVar[int]
    BASELINE_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    MAX_DELTA_MS_FIELD_NUMBER: _ClassVar[int]
    BASELINE_AVG_MS_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_AVG_MS_FIELD_NUMBER: _ClassVar[int]
    AVG_DELTA_MS_FIELD_NUMBER: _ClassVar[int]
    name: str
    baseline_count: int
    candidate_count: int
    count_delta: int
    baseline_total_ms: float
    candidate_total_ms: float
    total_delta_ms: float
    baseline_max_ms: float
    candidate_max_ms: float
    max_delta_ms: float
    baseline_avg_ms: float
    candidate_avg_ms: float
    avg_delta_ms: float
    def __init__(self, name: _Optional[str] = ..., baseline_count: _Optional[int] = ..., candidate_count: _Optional[int] = ..., count_delta: _Optional[int] = ..., baseline_total_ms: _Optional[float] = ..., candidate_total_ms: _Optional[float] = ..., total_delta_ms: _Optional[float] = ..., baseline_max_ms: _Optional[float] = ..., candidate_max_ms: _Optional[float] = ..., max_delta_ms: _Optional[float] = ..., baseline_avg_ms: _Optional[float] = ..., candidate_avg_ms: _Optional[float] = ..., avg_delta_ms: _Optional[float] = ...) -> None: ...
