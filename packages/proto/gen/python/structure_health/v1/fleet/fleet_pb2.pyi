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
    __slots__ = ("entries", "rule_conformance", "profile_distribution", "scenario_count", "passing_count", "missing_freshness_count", "autofixable_total", "errors")
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    RULE_CONFORMANCE_FIELD_NUMBER: _ClassVar[int]
    PROFILE_DISTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_COUNT_FIELD_NUMBER: _ClassVar[int]
    PASSING_COUNT_FIELD_NUMBER: _ClassVar[int]
    MISSING_FRESHNESS_COUNT_FIELD_NUMBER: _ClassVar[int]
    AUTOFIXABLE_TOTAL_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[FleetScenarioEntry]
    rule_conformance: _containers.RepeatedCompositeFieldContainer[RuleConformance]
    profile_distribution: _containers.RepeatedCompositeFieldContainer[ProfileDistribution]
    scenario_count: int
    passing_count: int
    missing_freshness_count: int
    autofixable_total: int
    errors: _containers.RepeatedCompositeFieldContainer[FleetScanError]
    def __init__(self, entries: _Optional[_Iterable[_Union[FleetScenarioEntry, _Mapping]]] = ..., rule_conformance: _Optional[_Iterable[_Union[RuleConformance, _Mapping]]] = ..., profile_distribution: _Optional[_Iterable[_Union[ProfileDistribution, _Mapping]]] = ..., scenario_count: _Optional[int] = ..., passing_count: _Optional[int] = ..., missing_freshness_count: _Optional[int] = ..., autofixable_total: _Optional[int] = ..., errors: _Optional[_Iterable[_Union[FleetScanError, _Mapping]]] = ...) -> None: ...

class FleetScenarioEntry(_message.Message):
    __slots__ = ("scenario", "passed", "profile_id", "profile_recognized", "error_count", "warning_count", "total_findings", "autofixable_count", "missing_freshness_check", "surfaces", "degraded_reason")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_RECOGNIZED_FIELD_NUMBER: _ClassVar[int]
    ERROR_COUNT_FIELD_NUMBER: _ClassVar[int]
    WARNING_COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    AUTOFIXABLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    MISSING_FRESHNESS_CHECK_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASON_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    passed: bool
    profile_id: str
    profile_recognized: bool
    error_count: int
    warning_count: int
    total_findings: int
    autofixable_count: int
    missing_freshness_check: bool
    surfaces: _containers.RepeatedScalarFieldContainer[str]
    degraded_reason: str
    def __init__(self, scenario: _Optional[str] = ..., passed: _Optional[bool] = ..., profile_id: _Optional[str] = ..., profile_recognized: _Optional[bool] = ..., error_count: _Optional[int] = ..., warning_count: _Optional[int] = ..., total_findings: _Optional[int] = ..., autofixable_count: _Optional[int] = ..., missing_freshness_check: _Optional[bool] = ..., surfaces: _Optional[_Iterable[str]] = ..., degraded_reason: _Optional[str] = ...) -> None: ...

class RuleConformance(_message.Message):
    __slots__ = ("code", "offending_scenarios", "total_findings", "autofixable", "worst_severity")
    CODE_FIELD_NUMBER: _ClassVar[int]
    OFFENDING_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    AUTOFIXABLE_FIELD_NUMBER: _ClassVar[int]
    WORST_SEVERITY_FIELD_NUMBER: _ClassVar[int]
    code: str
    offending_scenarios: int
    total_findings: int
    autofixable: int
    worst_severity: str
    def __init__(self, code: _Optional[str] = ..., offending_scenarios: _Optional[int] = ..., total_findings: _Optional[int] = ..., autofixable: _Optional[int] = ..., worst_severity: _Optional[str] = ...) -> None: ...

class ProfileDistribution(_message.Message):
    __slots__ = ("profile_id", "scenario_count", "recognized")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_COUNT_FIELD_NUMBER: _ClassVar[int]
    RECOGNIZED_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    scenario_count: int
    recognized: bool
    def __init__(self, profile_id: _Optional[str] = ..., scenario_count: _Optional[int] = ..., recognized: _Optional[bool] = ...) -> None: ...

class FleetScanError(_message.Message):
    __slots__ = ("scenario", "reason")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    reason: str
    def __init__(self, scenario: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...
