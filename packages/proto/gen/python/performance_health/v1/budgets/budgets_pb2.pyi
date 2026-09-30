from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Budget(_message.Message):
    __slots__ = ("scenario", "go_build_max_ms", "ui_build_max_ms", "bundle_max_bytes", "lcp_max_ms", "startup_max_ms", "component_commit_max_ms", "component_commit_avg_max_ms", "ratchet", "drawn_fps_min", "dropped_frame_rate_max", "long_task_total_max_ms", "long_task_max_ms", "raster_total_max_ms", "layout_total_max_ms", "paint_total_max_ms", "input_event_count_min", "flows", "load_only", "cls_max", "response_end_max_ms", "dom_interactive_max_ms", "dom_content_loaded_max_ms", "load_event_end_max_ms")
    class FlowsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: FlowBudget
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[FlowBudget, _Mapping]] = ...) -> None: ...
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    GO_BUILD_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    UI_BUILD_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_MAX_BYTES_FIELD_NUMBER: _ClassVar[int]
    LCP_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    STARTUP_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_COMMIT_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_COMMIT_AVG_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    RATCHET_FIELD_NUMBER: _ClassVar[int]
    DRAWN_FPS_MIN_FIELD_NUMBER: _ClassVar[int]
    DROPPED_FRAME_RATE_MAX_FIELD_NUMBER: _ClassVar[int]
    LONG_TASK_TOTAL_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    LONG_TASK_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    RASTER_TOTAL_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    LAYOUT_TOTAL_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    PAINT_TOTAL_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    INPUT_EVENT_COUNT_MIN_FIELD_NUMBER: _ClassVar[int]
    FLOWS_FIELD_NUMBER: _ClassVar[int]
    LOAD_ONLY_FIELD_NUMBER: _ClassVar[int]
    CLS_MAX_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_END_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    DOM_INTERACTIVE_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    DOM_CONTENT_LOADED_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    LOAD_EVENT_END_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    go_build_max_ms: int
    ui_build_max_ms: int
    bundle_max_bytes: int
    lcp_max_ms: int
    startup_max_ms: int
    component_commit_max_ms: float
    component_commit_avg_max_ms: float
    ratchet: bool
    drawn_fps_min: float
    dropped_frame_rate_max: float
    long_task_total_max_ms: int
    long_task_max_ms: float
    raster_total_max_ms: float
    layout_total_max_ms: float
    paint_total_max_ms: float
    input_event_count_min: int
    flows: _containers.MessageMap[str, FlowBudget]
    load_only: bool
    cls_max: float
    response_end_max_ms: int
    dom_interactive_max_ms: int
    dom_content_loaded_max_ms: int
    load_event_end_max_ms: int
    def __init__(self, scenario: _Optional[str] = ..., go_build_max_ms: _Optional[int] = ..., ui_build_max_ms: _Optional[int] = ..., bundle_max_bytes: _Optional[int] = ..., lcp_max_ms: _Optional[int] = ..., startup_max_ms: _Optional[int] = ..., component_commit_max_ms: _Optional[float] = ..., component_commit_avg_max_ms: _Optional[float] = ..., ratchet: _Optional[bool] = ..., drawn_fps_min: _Optional[float] = ..., dropped_frame_rate_max: _Optional[float] = ..., long_task_total_max_ms: _Optional[int] = ..., long_task_max_ms: _Optional[float] = ..., raster_total_max_ms: _Optional[float] = ..., layout_total_max_ms: _Optional[float] = ..., paint_total_max_ms: _Optional[float] = ..., input_event_count_min: _Optional[int] = ..., flows: _Optional[_Mapping[str, FlowBudget]] = ..., load_only: _Optional[bool] = ..., cls_max: _Optional[float] = ..., response_end_max_ms: _Optional[int] = ..., dom_interactive_max_ms: _Optional[int] = ..., dom_content_loaded_max_ms: _Optional[int] = ..., load_event_end_max_ms: _Optional[int] = ...) -> None: ...

class FlowBudget(_message.Message):
    __slots__ = ("lcp_max_ms", "component_commit_avg_max_ms", "component_commit_max_ms", "drawn_fps_min", "dropped_frame_rate_max", "long_task_total_max_ms", "long_task_max_ms", "raster_total_max_ms", "layout_total_max_ms", "paint_total_max_ms", "input_event_count_min", "load_only", "cls_max")
    LCP_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_COMMIT_AVG_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_COMMIT_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    DRAWN_FPS_MIN_FIELD_NUMBER: _ClassVar[int]
    DROPPED_FRAME_RATE_MAX_FIELD_NUMBER: _ClassVar[int]
    LONG_TASK_TOTAL_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    LONG_TASK_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    RASTER_TOTAL_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    LAYOUT_TOTAL_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    PAINT_TOTAL_MAX_MS_FIELD_NUMBER: _ClassVar[int]
    INPUT_EVENT_COUNT_MIN_FIELD_NUMBER: _ClassVar[int]
    LOAD_ONLY_FIELD_NUMBER: _ClassVar[int]
    CLS_MAX_FIELD_NUMBER: _ClassVar[int]
    lcp_max_ms: int
    component_commit_avg_max_ms: float
    component_commit_max_ms: float
    drawn_fps_min: float
    dropped_frame_rate_max: float
    long_task_total_max_ms: int
    long_task_max_ms: float
    raster_total_max_ms: float
    layout_total_max_ms: float
    paint_total_max_ms: float
    input_event_count_min: int
    load_only: bool
    cls_max: float
    def __init__(self, lcp_max_ms: _Optional[int] = ..., component_commit_avg_max_ms: _Optional[float] = ..., component_commit_max_ms: _Optional[float] = ..., drawn_fps_min: _Optional[float] = ..., dropped_frame_rate_max: _Optional[float] = ..., long_task_total_max_ms: _Optional[int] = ..., long_task_max_ms: _Optional[float] = ..., raster_total_max_ms: _Optional[float] = ..., layout_total_max_ms: _Optional[float] = ..., paint_total_max_ms: _Optional[float] = ..., input_event_count_min: _Optional[int] = ..., load_only: _Optional[bool] = ..., cls_max: _Optional[float] = ...) -> None: ...

class GetBudgetRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class GetBudgetResponse(_message.Message):
    __slots__ = ("budget", "declared")
    BUDGET_FIELD_NUMBER: _ClassVar[int]
    DECLARED_FIELD_NUMBER: _ClassVar[int]
    budget: Budget
    declared: bool
    def __init__(self, budget: _Optional[_Union[Budget, _Mapping]] = ..., declared: _Optional[bool] = ...) -> None: ...

class SetBudgetRequest(_message.Message):
    __slots__ = ("budget", "flow")
    BUDGET_FIELD_NUMBER: _ClassVar[int]
    FLOW_FIELD_NUMBER: _ClassVar[int]
    budget: Budget
    flow: str
    def __init__(self, budget: _Optional[_Union[Budget, _Mapping]] = ..., flow: _Optional[str] = ...) -> None: ...

class SetBudgetResponse(_message.Message):
    __slots__ = ("budget", "dry_run")
    BUDGET_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    budget: Budget
    dry_run: bool
    def __init__(self, budget: _Optional[_Union[Budget, _Mapping]] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class CheckBudgetRequest(_message.Message):
    __slots__ = ("scenario", "flow")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    FLOW_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    flow: str
    def __init__(self, scenario: _Optional[str] = ..., flow: _Optional[str] = ...) -> None: ...

class CheckBudgetResponse(_message.Message):
    __slots__ = ("scenario", "passed", "violations")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    passed: bool
    violations: _containers.RepeatedCompositeFieldContainer[BudgetViolation]
    def __init__(self, scenario: _Optional[str] = ..., passed: _Optional[bool] = ..., violations: _Optional[_Iterable[_Union[BudgetViolation, _Mapping]]] = ...) -> None: ...

class BudgetViolation(_message.Message):
    __slots__ = ("axis", "measured", "budget", "unit", "flow", "mode", "detail", "measured_value", "budget_value")
    AXIS_FIELD_NUMBER: _ClassVar[int]
    MEASURED_FIELD_NUMBER: _ClassVar[int]
    BUDGET_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    FLOW_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    MEASURED_VALUE_FIELD_NUMBER: _ClassVar[int]
    BUDGET_VALUE_FIELD_NUMBER: _ClassVar[int]
    axis: str
    measured: int
    budget: int
    unit: str
    flow: str
    mode: str
    detail: str
    measured_value: float
    budget_value: float
    def __init__(self, axis: _Optional[str] = ..., measured: _Optional[int] = ..., budget: _Optional[int] = ..., unit: _Optional[str] = ..., flow: _Optional[str] = ..., mode: _Optional[str] = ..., detail: _Optional[str] = ..., measured_value: _Optional[float] = ..., budget_value: _Optional[float] = ...) -> None: ...
