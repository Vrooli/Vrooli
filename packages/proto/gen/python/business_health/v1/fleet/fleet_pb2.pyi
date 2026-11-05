import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
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
    __slots__ = ("entries", "as_of", "scenario_count", "passing_count", "starter_registry_count", "template_laggard_count", "errors")
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    AS_OF_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_COUNT_FIELD_NUMBER: _ClassVar[int]
    PASSING_COUNT_FIELD_NUMBER: _ClassVar[int]
    STARTER_REGISTRY_COUNT_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_LAGGARD_COUNT_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[FleetScenarioEntry]
    as_of: _timestamp_pb2.Timestamp
    scenario_count: int
    passing_count: int
    starter_registry_count: int
    template_laggard_count: int
    errors: _containers.RepeatedCompositeFieldContainer[FleetScanError]
    def __init__(self, entries: _Optional[_Iterable[_Union[FleetScenarioEntry, _Mapping]]] = ..., as_of: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., scenario_count: _Optional[int] = ..., passing_count: _Optional[int] = ..., starter_registry_count: _Optional[int] = ..., template_laggard_count: _Optional[int] = ..., errors: _Optional[_Iterable[_Union[FleetScanError, _Mapping]]] = ...) -> None: ...

class FleetScenarioEntry(_message.Message):
    __slots__ = ("scenario", "passed", "error_count", "warning_count", "total_findings", "autofixable_count", "starter_registry", "template_version", "template_laggard", "orphaned_targets", "unproven_claims", "debt_score", "degraded_reason")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    ERROR_COUNT_FIELD_NUMBER: _ClassVar[int]
    WARNING_COUNT_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    AUTOFIXABLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    STARTER_REGISTRY_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_VERSION_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_LAGGARD_FIELD_NUMBER: _ClassVar[int]
    ORPHANED_TARGETS_FIELD_NUMBER: _ClassVar[int]
    UNPROVEN_CLAIMS_FIELD_NUMBER: _ClassVar[int]
    DEBT_SCORE_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASON_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    passed: bool
    error_count: int
    warning_count: int
    total_findings: int
    autofixable_count: int
    starter_registry: bool
    template_version: str
    template_laggard: bool
    orphaned_targets: int
    unproven_claims: int
    debt_score: int
    degraded_reason: str
    def __init__(self, scenario: _Optional[str] = ..., passed: _Optional[bool] = ..., error_count: _Optional[int] = ..., warning_count: _Optional[int] = ..., total_findings: _Optional[int] = ..., autofixable_count: _Optional[int] = ..., starter_registry: _Optional[bool] = ..., template_version: _Optional[str] = ..., template_laggard: _Optional[bool] = ..., orphaned_targets: _Optional[int] = ..., unproven_claims: _Optional[int] = ..., debt_score: _Optional[int] = ..., degraded_reason: _Optional[str] = ...) -> None: ...

class FleetScanError(_message.Message):
    __slots__ = ("scenario", "reason")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    reason: str
    def __init__(self, scenario: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...
