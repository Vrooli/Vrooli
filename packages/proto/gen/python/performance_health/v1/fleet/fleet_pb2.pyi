from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ScanFleetRequest(_message.Message):
    __slots__ = ("scenarios",)
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    scenarios: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenarios: _Optional[_Iterable[str]] = ...) -> None: ...

class ScanFleetResponse(_message.Message):
    __slots__ = ("entries", "tier_distribution", "scenario_count", "no_budget_count", "regressed_count", "errors")
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    TIER_DISTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_COUNT_FIELD_NUMBER: _ClassVar[int]
    NO_BUDGET_COUNT_FIELD_NUMBER: _ClassVar[int]
    REGRESSED_COUNT_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[FleetScenarioEntry]
    tier_distribution: _containers.RepeatedCompositeFieldContainer[TierDistribution]
    scenario_count: int
    no_budget_count: int
    regressed_count: int
    errors: _containers.RepeatedCompositeFieldContainer[FleetScanError]
    def __init__(self, entries: _Optional[_Iterable[_Union[FleetScenarioEntry, _Mapping]]] = ..., tier_distribution: _Optional[_Iterable[_Union[TierDistribution, _Mapping]]] = ..., scenario_count: _Optional[int] = ..., no_budget_count: _Optional[int] = ..., regressed_count: _Optional[int] = ..., errors: _Optional[_Iterable[_Union[FleetScanError, _Mapping]]] = ...) -> None: ...

class FleetScenarioEntry(_message.Message):
    __slots__ = ("scenario", "tier", "has_budget", "go_build_ms", "ui_build_ms", "regressed", "degraded_reason")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    HAS_BUDGET_FIELD_NUMBER: _ClassVar[int]
    GO_BUILD_MS_FIELD_NUMBER: _ClassVar[int]
    UI_BUILD_MS_FIELD_NUMBER: _ClassVar[int]
    REGRESSED_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASON_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    tier: str
    has_budget: bool
    go_build_ms: int
    ui_build_ms: int
    regressed: bool
    degraded_reason: str
    def __init__(self, scenario: _Optional[str] = ..., tier: _Optional[str] = ..., has_budget: _Optional[bool] = ..., go_build_ms: _Optional[int] = ..., ui_build_ms: _Optional[int] = ..., regressed: _Optional[bool] = ..., degraded_reason: _Optional[str] = ...) -> None: ...

class TierDistribution(_message.Message):
    __slots__ = ("tier", "scenario_count")
    TIER_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_COUNT_FIELD_NUMBER: _ClassVar[int]
    tier: str
    scenario_count: int
    def __init__(self, tier: _Optional[str] = ..., scenario_count: _Optional[int] = ...) -> None: ...

class FleetScanError(_message.Message):
    __slots__ = ("scenario", "reason")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    reason: str
    def __init__(self, scenario: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...
