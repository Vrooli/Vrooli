from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RunSweepRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class FlowSweepResult(_message.Message):
    __slots__ = ("flow", "outcome", "within_budget", "violations", "reason")
    FLOW_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    WITHIN_BUDGET_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    flow: str
    outcome: str
    within_budget: bool
    violations: _containers.RepeatedScalarFieldContainer[str]
    reason: str
    def __init__(self, flow: _Optional[str] = ..., outcome: _Optional[str] = ..., within_budget: _Optional[bool] = ..., violations: _Optional[_Iterable[str]] = ..., reason: _Optional[str] = ...) -> None: ...

class RunSweepResponse(_message.Message):
    __slots__ = ("scenario", "results")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    results: _containers.RepeatedCompositeFieldContainer[FlowSweepResult]
    def __init__(self, scenario: _Optional[str] = ..., results: _Optional[_Iterable[_Union[FlowSweepResult, _Mapping]]] = ...) -> None: ...
