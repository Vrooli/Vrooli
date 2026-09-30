from common.v1 import maturity_pb2 as _maturity_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Severity(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SEVERITY_UNSPECIFIED: _ClassVar[Severity]
    SEVERITY_ERROR: _ClassVar[Severity]
    SEVERITY_WARNING: _ClassVar[Severity]
    SEVERITY_INFO: _ClassVar[Severity]

class DomainStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DOMAIN_STATUS_UNSPECIFIED: _ClassVar[DomainStatus]
    DOMAIN_STATUS_COVERED: _ClassVar[DomainStatus]
    DOMAIN_STATUS_UNCOVERED: _ClassVar[DomainStatus]
    DOMAIN_STATUS_WAIVED: _ClassVar[DomainStatus]
    DOMAIN_STATUS_NOT_EXPECTED: _ClassVar[DomainStatus]

class Tier(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    TIER_UNSPECIFIED: _ClassVar[Tier]
    TIER_FULL: _ClassVar[Tier]
    TIER_PARTIAL: _ClassVar[Tier]
    TIER_FALLBACK: _ClassVar[Tier]
SEVERITY_UNSPECIFIED: Severity
SEVERITY_ERROR: Severity
SEVERITY_WARNING: Severity
SEVERITY_INFO: Severity
DOMAIN_STATUS_UNSPECIFIED: DomainStatus
DOMAIN_STATUS_COVERED: DomainStatus
DOMAIN_STATUS_UNCOVERED: DomainStatus
DOMAIN_STATUS_WAIVED: DomainStatus
DOMAIN_STATUS_NOT_EXPECTED: DomainStatus
TIER_UNSPECIFIED: Tier
TIER_FULL: Tier
TIER_PARTIAL: Tier
TIER_FALLBACK: Tier

class MeasureSummary(_message.Message):
    __slots__ = ("name", "intent", "tier", "effect", "question_count", "probe_passed", "probe_detail", "tier_note")
    NAME_FIELD_NUMBER: _ClassVar[int]
    INTENT_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    EFFECT_FIELD_NUMBER: _ClassVar[int]
    QUESTION_COUNT_FIELD_NUMBER: _ClassVar[int]
    PROBE_PASSED_FIELD_NUMBER: _ClassVar[int]
    PROBE_DETAIL_FIELD_NUMBER: _ClassVar[int]
    TIER_NOTE_FIELD_NUMBER: _ClassVar[int]
    name: str
    intent: str
    tier: Tier
    effect: str
    question_count: int
    probe_passed: bool
    probe_detail: str
    tier_note: str
    def __init__(self, name: _Optional[str] = ..., intent: _Optional[str] = ..., tier: _Optional[_Union[Tier, str]] = ..., effect: _Optional[str] = ..., question_count: _Optional[int] = ..., probe_passed: _Optional[bool] = ..., probe_detail: _Optional[str] = ..., tier_note: _Optional[str] = ...) -> None: ...

class DomainCoverage(_message.Message):
    __slots__ = ("domain", "status", "measure_count", "tier", "waiver_reason", "note", "measures")
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MEASURE_COUNT_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    WAIVER_REASON_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    MEASURES_FIELD_NUMBER: _ClassVar[int]
    domain: str
    status: DomainStatus
    measure_count: int
    tier: Tier
    waiver_reason: str
    note: str
    measures: _containers.RepeatedCompositeFieldContainer[MeasureSummary]
    def __init__(self, domain: _Optional[str] = ..., status: _Optional[_Union[DomainStatus, str]] = ..., measure_count: _Optional[int] = ..., tier: _Optional[_Union[Tier, str]] = ..., waiver_reason: _Optional[str] = ..., note: _Optional[str] = ..., measures: _Optional[_Iterable[_Union[MeasureSummary, _Mapping]]] = ...) -> None: ...

class Finding(_message.Message):
    __slots__ = ("rule_id", "severity", "title", "description", "remediation", "file_path", "scanner")
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    SCANNER_FIELD_NUMBER: _ClassVar[int]
    rule_id: str
    severity: Severity
    title: str
    description: str
    remediation: str
    file_path: str
    scanner: str
    def __init__(self, rule_id: _Optional[str] = ..., severity: _Optional[_Union[Severity, str]] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., remediation: _Optional[str] = ..., file_path: _Optional[str] = ..., scanner: _Optional[str] = ...) -> None: ...

class Summary(_message.Message):
    __slots__ = ("errors", "warnings", "infos")
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    INFOS_FIELD_NUMBER: _ClassVar[int]
    errors: int
    warnings: int
    infos: int
    def __init__(self, errors: _Optional[int] = ..., warnings: _Optional[int] = ..., infos: _Optional[int] = ...) -> None: ...

class ScenarioCoverageReport(_message.Message):
    __slots__ = ("scenario", "passed", "domains", "findings", "summary", "skipped_scanners", "assessment")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    DOMAINS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_SCANNERS_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    passed: bool
    domains: _containers.RepeatedCompositeFieldContainer[DomainCoverage]
    findings: _containers.RepeatedCompositeFieldContainer[Finding]
    summary: Summary
    skipped_scanners: _containers.RepeatedScalarFieldContainer[str]
    assessment: _maturity_pb2.MaturityAssessment
    def __init__(self, scenario: _Optional[str] = ..., passed: _Optional[bool] = ..., domains: _Optional[_Iterable[_Union[DomainCoverage, _Mapping]]] = ..., findings: _Optional[_Iterable[_Union[Finding, _Mapping]]] = ..., summary: _Optional[_Union[Summary, _Mapping]] = ..., skipped_scanners: _Optional[_Iterable[str]] = ..., assessment: _Optional[_Union[_maturity_pb2.MaturityAssessment, _Mapping]] = ...) -> None: ...

class ListFleetCoverageRequest(_message.Message):
    __slots__ = ("scenarios",)
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    scenarios: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenarios: _Optional[_Iterable[str]] = ...) -> None: ...

class FleetEntry(_message.Message):
    __slots__ = ("scenario", "passed", "expected", "covered", "waived", "uncovered", "worst_tier", "measure_count")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_FIELD_NUMBER: _ClassVar[int]
    COVERED_FIELD_NUMBER: _ClassVar[int]
    WAIVED_FIELD_NUMBER: _ClassVar[int]
    UNCOVERED_FIELD_NUMBER: _ClassVar[int]
    WORST_TIER_FIELD_NUMBER: _ClassVar[int]
    MEASURE_COUNT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    passed: bool
    expected: int
    covered: int
    waived: int
    uncovered: int
    worst_tier: Tier
    measure_count: int
    def __init__(self, scenario: _Optional[str] = ..., passed: _Optional[bool] = ..., expected: _Optional[int] = ..., covered: _Optional[int] = ..., waived: _Optional[int] = ..., uncovered: _Optional[int] = ..., worst_tier: _Optional[_Union[Tier, str]] = ..., measure_count: _Optional[int] = ...) -> None: ...

class ListFleetCoverageResponse(_message.Message):
    __slots__ = ("entries",)
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[FleetEntry]
    def __init__(self, entries: _Optional[_Iterable[_Union[FleetEntry, _Mapping]]] = ...) -> None: ...
