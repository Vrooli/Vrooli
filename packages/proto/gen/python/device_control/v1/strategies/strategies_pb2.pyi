from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListStrategiesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListStrategiesResponse(_message.Message):
    __slots__ = ("strategies",)
    STRATEGIES_FIELD_NUMBER: _ClassVar[int]
    strategies: _containers.RepeatedCompositeFieldContainer[Strategy]
    def __init__(self, strategies: _Optional[_Iterable[_Union[Strategy, _Mapping]]] = ...) -> None: ...

class VerifyStrategyRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class Strategy(_message.Message):
    __slots__ = ("id", "description", "status", "capabilities", "tiers", "executable_step_kinds", "next_actions", "promotable", "evidence_class", "minimum_useful_fps")
    ID_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    TIERS_FIELD_NUMBER: _ClassVar[int]
    EXECUTABLE_STEP_KINDS_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    PROMOTABLE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_CLASS_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_USEFUL_FPS_FIELD_NUMBER: _ClassVar[int]
    id: str
    description: str
    status: str
    capabilities: _containers.RepeatedCompositeFieldContainer[Capability]
    tiers: _containers.RepeatedScalarFieldContainer[str]
    executable_step_kinds: _containers.RepeatedScalarFieldContainer[str]
    next_actions: _containers.RepeatedScalarFieldContainer[str]
    promotable: bool
    evidence_class: str
    minimum_useful_fps: float
    def __init__(self, id: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ..., capabilities: _Optional[_Iterable[_Union[Capability, _Mapping]]] = ..., tiers: _Optional[_Iterable[str]] = ..., executable_step_kinds: _Optional[_Iterable[str]] = ..., next_actions: _Optional[_Iterable[str]] = ..., promotable: _Optional[bool] = ..., evidence_class: _Optional[str] = ..., minimum_useful_fps: _Optional[float] = ...) -> None: ...

class Capability(_message.Message):
    __slots__ = ("name", "status", "prerequisite", "next_action")
    NAME_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PREREQUISITE_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    name: str
    status: str
    prerequisite: str
    next_action: str
    def __init__(self, name: _Optional[str] = ..., status: _Optional[str] = ..., prerequisite: _Optional[str] = ..., next_action: _Optional[str] = ...) -> None: ...

class ConformanceReport(_message.Message):
    __slots__ = ("strategy_id", "status", "passed", "failed", "tiers", "executable_step_kinds", "next_actions", "promotable", "evidence_class", "minimum_useful_fps")
    STRATEGY_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    TIERS_FIELD_NUMBER: _ClassVar[int]
    EXECUTABLE_STEP_KINDS_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    PROMOTABLE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_CLASS_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_USEFUL_FPS_FIELD_NUMBER: _ClassVar[int]
    strategy_id: str
    status: str
    passed: _containers.RepeatedScalarFieldContainer[str]
    failed: _containers.RepeatedScalarFieldContainer[str]
    tiers: _containers.RepeatedScalarFieldContainer[str]
    executable_step_kinds: _containers.RepeatedScalarFieldContainer[str]
    next_actions: _containers.RepeatedScalarFieldContainer[str]
    promotable: bool
    evidence_class: str
    minimum_useful_fps: float
    def __init__(self, strategy_id: _Optional[str] = ..., status: _Optional[str] = ..., passed: _Optional[_Iterable[str]] = ..., failed: _Optional[_Iterable[str]] = ..., tiers: _Optional[_Iterable[str]] = ..., executable_step_kinds: _Optional[_Iterable[str]] = ..., next_actions: _Optional[_Iterable[str]] = ..., promotable: _Optional[bool] = ..., evidence_class: _Optional[str] = ..., minimum_useful_fps: _Optional[float] = ...) -> None: ...
