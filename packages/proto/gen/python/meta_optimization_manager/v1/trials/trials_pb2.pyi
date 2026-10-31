import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TrialVerdict(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TRIAL_VERDICT_UNSPECIFIED: _ClassVar[TrialVerdict]
    TRIAL_VERDICT_PASS: _ClassVar[TrialVerdict]
    TRIAL_VERDICT_FAIL: _ClassVar[TrialVerdict]
    TRIAL_VERDICT_ERROR: _ClassVar[TrialVerdict]
TRIAL_VERDICT_UNSPECIFIED: TrialVerdict
TRIAL_VERDICT_PASS: TrialVerdict
TRIAL_VERDICT_FAIL: TrialVerdict
TRIAL_VERDICT_ERROR: TrialVerdict

class TrialTask(_message.Message):
    __slots__ = ("id", "suite", "guide_task_id", "description", "negative")
    ID_FIELD_NUMBER: _ClassVar[int]
    SUITE_FIELD_NUMBER: _ClassVar[int]
    GUIDE_TASK_ID_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    NEGATIVE_FIELD_NUMBER: _ClassVar[int]
    id: str
    suite: str
    guide_task_id: str
    description: str
    negative: bool
    def __init__(self, id: _Optional[str] = ..., suite: _Optional[str] = ..., guide_task_id: _Optional[str] = ..., description: _Optional[str] = ..., negative: _Optional[bool] = ...) -> None: ...

class TrialRun(_message.Message):
    __slots__ = ("id", "task_id", "suite", "model", "verdict", "tokens", "duration_ms", "sandbox_diff_ref", "at")
    ID_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    SUITE_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    TOKENS_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    SANDBOX_DIFF_REF_FIELD_NUMBER: _ClassVar[int]
    AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    task_id: str
    suite: str
    model: str
    verdict: TrialVerdict
    tokens: int
    duration_ms: int
    sandbox_diff_ref: str
    at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., task_id: _Optional[str] = ..., suite: _Optional[str] = ..., model: _Optional[str] = ..., verdict: _Optional[_Union[TrialVerdict, str]] = ..., tokens: _Optional[int] = ..., duration_ms: _Optional[int] = ..., sandbox_diff_ref: _Optional[str] = ..., at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListTrialTasksRequest(_message.Message):
    __slots__ = ("suite",)
    SUITE_FIELD_NUMBER: _ClassVar[int]
    suite: str
    def __init__(self, suite: _Optional[str] = ...) -> None: ...

class ListTrialTasksResponse(_message.Message):
    __slots__ = ("tasks",)
    TASKS_FIELD_NUMBER: _ClassVar[int]
    tasks: _containers.RepeatedCompositeFieldContainer[TrialTask]
    def __init__(self, tasks: _Optional[_Iterable[_Union[TrialTask, _Mapping]]] = ...) -> None: ...

class RunTrialsRequest(_message.Message):
    __slots__ = ("suite", "task_id")
    SUITE_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    suite: str
    task_id: str
    def __init__(self, suite: _Optional[str] = ..., task_id: _Optional[str] = ...) -> None: ...

class RunTrialsResponse(_message.Message):
    __slots__ = ("runs",)
    RUNS_FIELD_NUMBER: _ClassVar[int]
    runs: _containers.RepeatedCompositeFieldContainer[TrialRun]
    def __init__(self, runs: _Optional[_Iterable[_Union[TrialRun, _Mapping]]] = ...) -> None: ...

class TrialHistoryPoint(_message.Message):
    __slots__ = ("at", "success_rate", "median_tokens", "median_duration_ms", "run_count")
    AT_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_RATE_FIELD_NUMBER: _ClassVar[int]
    MEDIAN_TOKENS_FIELD_NUMBER: _ClassVar[int]
    MEDIAN_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    RUN_COUNT_FIELD_NUMBER: _ClassVar[int]
    at: _timestamp_pb2.Timestamp
    success_rate: float
    median_tokens: int
    median_duration_ms: int
    run_count: int
    def __init__(self, at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., success_rate: _Optional[float] = ..., median_tokens: _Optional[int] = ..., median_duration_ms: _Optional[int] = ..., run_count: _Optional[int] = ...) -> None: ...

class GetTrialHistoryRequest(_message.Message):
    __slots__ = ("task_id", "suite")
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    SUITE_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    suite: str
    def __init__(self, task_id: _Optional[str] = ..., suite: _Optional[str] = ...) -> None: ...

class GetTrialHistoryResponse(_message.Message):
    __slots__ = ("points", "recent_runs")
    POINTS_FIELD_NUMBER: _ClassVar[int]
    RECENT_RUNS_FIELD_NUMBER: _ClassVar[int]
    points: _containers.RepeatedCompositeFieldContainer[TrialHistoryPoint]
    recent_runs: _containers.RepeatedCompositeFieldContainer[TrialRun]
    def __init__(self, points: _Optional[_Iterable[_Union[TrialHistoryPoint, _Mapping]]] = ..., recent_runs: _Optional[_Iterable[_Union[TrialRun, _Mapping]]] = ...) -> None: ...

class GetTrialRunRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetTrialRunResponse(_message.Message):
    __slots__ = ("run",)
    RUN_FIELD_NUMBER: _ClassVar[int]
    run: TrialRun
    def __init__(self, run: _Optional[_Union[TrialRun, _Mapping]] = ...) -> None: ...

class GetGateCoverageRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetGateCoverageResponse(_message.Message):
    __slots__ = ("guide_tasks_total", "guide_tasks_with_gate", "gate_coverage_ratio")
    GUIDE_TASKS_TOTAL_FIELD_NUMBER: _ClassVar[int]
    GUIDE_TASKS_WITH_GATE_FIELD_NUMBER: _ClassVar[int]
    GATE_COVERAGE_RATIO_FIELD_NUMBER: _ClassVar[int]
    guide_tasks_total: int
    guide_tasks_with_gate: int
    gate_coverage_ratio: float
    def __init__(self, guide_tasks_total: _Optional[int] = ..., guide_tasks_with_gate: _Optional[int] = ..., gate_coverage_ratio: _Optional[float] = ...) -> None: ...
