from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListApprovedDependenciesRequest(_message.Message):
    __slots__ = ("ecosystem", "state", "surface", "use_case")
    ECOSYSTEM_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    USE_CASE_FIELD_NUMBER: _ClassVar[int]
    ecosystem: str
    state: str
    surface: str
    use_case: str
    def __init__(self, ecosystem: _Optional[str] = ..., state: _Optional[str] = ..., surface: _Optional[str] = ..., use_case: _Optional[str] = ...) -> None: ...

class SearchApprovedDependenciesRequest(_message.Message):
    __slots__ = ("query", "ecosystem", "limit")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    ECOSYSTEM_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    query: str
    ecosystem: str
    limit: int
    def __init__(self, query: _Optional[str] = ..., ecosystem: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ExplainApprovedDependencyRequest(_message.Message):
    __slots__ = ("ecosystem", "package_name")
    ECOSYSTEM_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_NAME_FIELD_NUMBER: _ClassVar[int]
    ecosystem: str
    package_name: str
    def __init__(self, ecosystem: _Optional[str] = ..., package_name: _Optional[str] = ...) -> None: ...

class ValidateApprovedDependenciesRequest(_message.Message):
    __slots__ = ("scenario", "policy_mode")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    POLICY_MODE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    policy_mode: str
    def __init__(self, scenario: _Optional[str] = ..., policy_mode: _Optional[str] = ...) -> None: ...

class ValidateFleetApprovedDependenciesRequest(_message.Message):
    __slots__ = ("policy_mode",)
    POLICY_MODE_FIELD_NUMBER: _ClassVar[int]
    policy_mode: str
    def __init__(self, policy_mode: _Optional[str] = ...) -> None: ...

class UpsertApprovedDependencyRequest(_message.Message):
    __slots__ = ("record", "dry_run")
    RECORD_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    record: ApprovedDependencyRecord
    dry_run: bool
    def __init__(self, record: _Optional[_Union[ApprovedDependencyRecord, _Mapping]] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class PreviewVulnerabilityRemediationRequest(_message.Message):
    __slots__ = ("ecosystem", "package_name", "vulnerability_id")
    ECOSYSTEM_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_NAME_FIELD_NUMBER: _ClassVar[int]
    VULNERABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    ecosystem: str
    package_name: str
    vulnerability_id: str
    def __init__(self, ecosystem: _Optional[str] = ..., package_name: _Optional[str] = ..., vulnerability_id: _Optional[str] = ...) -> None: ...

class DenyVulnerableDependencyRequest(_message.Message):
    __slots__ = ("ecosystem", "package_name", "vulnerability_id", "affected_range", "fixed_range", "rationale", "approved_by", "dry_run")
    ECOSYSTEM_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_NAME_FIELD_NUMBER: _ClassVar[int]
    VULNERABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    AFFECTED_RANGE_FIELD_NUMBER: _ClassVar[int]
    FIXED_RANGE_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    APPROVED_BY_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    ecosystem: str
    package_name: str
    vulnerability_id: str
    affected_range: str
    fixed_range: str
    rationale: str
    approved_by: str
    dry_run: bool
    def __init__(self, ecosystem: _Optional[str] = ..., package_name: _Optional[str] = ..., vulnerability_id: _Optional[str] = ..., affected_range: _Optional[str] = ..., fixed_range: _Optional[str] = ..., rationale: _Optional[str] = ..., approved_by: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class ApprovedDependencyListResponse(_message.Message):
    __slots__ = ("records", "summary", "guidance")
    RECORDS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    GUIDANCE_FIELD_NUMBER: _ClassVar[int]
    records: _containers.RepeatedCompositeFieldContainer[ApprovedDependencyRecord]
    summary: DependencyGovernanceSummary
    guidance: str
    def __init__(self, records: _Optional[_Iterable[_Union[ApprovedDependencyRecord, _Mapping]]] = ..., summary: _Optional[_Union[DependencyGovernanceSummary, _Mapping]] = ..., guidance: _Optional[str] = ...) -> None: ...

class ApprovedDependencySearchResponse(_message.Message):
    __slots__ = ("records", "summary", "guidance")
    RECORDS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    GUIDANCE_FIELD_NUMBER: _ClassVar[int]
    records: _containers.RepeatedCompositeFieldContainer[ApprovedDependencyRecord]
    summary: DependencyGovernanceSummary
    guidance: str
    def __init__(self, records: _Optional[_Iterable[_Union[ApprovedDependencyRecord, _Mapping]]] = ..., summary: _Optional[_Union[DependencyGovernanceSummary, _Mapping]] = ..., guidance: _Optional[str] = ...) -> None: ...

class ApprovedDependencyExplainResponse(_message.Message):
    __slots__ = ("record", "found", "guidance")
    RECORD_FIELD_NUMBER: _ClassVar[int]
    FOUND_FIELD_NUMBER: _ClassVar[int]
    GUIDANCE_FIELD_NUMBER: _ClassVar[int]
    record: ApprovedDependencyRecord
    found: bool
    guidance: str
    def __init__(self, record: _Optional[_Union[ApprovedDependencyRecord, _Mapping]] = ..., found: _Optional[bool] = ..., guidance: _Optional[str] = ...) -> None: ...

class ApprovedDependencyValidationResponse(_message.Message):
    __slots__ = ("scenario", "passed", "summary", "findings", "observed_dependencies", "guidance")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    GUIDANCE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    passed: bool
    summary: DependencyGovernanceSummary
    findings: _containers.RepeatedCompositeFieldContainer[ApprovedDependencyFinding]
    observed_dependencies: _containers.RepeatedCompositeFieldContainer[ObservedDependency]
    guidance: str
    def __init__(self, scenario: _Optional[str] = ..., passed: _Optional[bool] = ..., summary: _Optional[_Union[DependencyGovernanceSummary, _Mapping]] = ..., findings: _Optional[_Iterable[_Union[ApprovedDependencyFinding, _Mapping]]] = ..., observed_dependencies: _Optional[_Iterable[_Union[ObservedDependency, _Mapping]]] = ..., guidance: _Optional[str] = ...) -> None: ...

class FleetApprovedDependencyValidationResponse(_message.Message):
    __slots__ = ("passed", "summary", "scenarios", "usage_groups", "findings", "guidance")
    PASSED_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    USAGE_GROUPS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    GUIDANCE_FIELD_NUMBER: _ClassVar[int]
    passed: bool
    summary: DependencyGovernanceSummary
    scenarios: _containers.RepeatedCompositeFieldContainer[ApprovedDependencyValidationResponse]
    usage_groups: _containers.RepeatedCompositeFieldContainer[DependencyUsageGroup]
    findings: _containers.RepeatedCompositeFieldContainer[ApprovedDependencyFinding]
    guidance: str
    def __init__(self, passed: _Optional[bool] = ..., summary: _Optional[_Union[DependencyGovernanceSummary, _Mapping]] = ..., scenarios: _Optional[_Iterable[_Union[ApprovedDependencyValidationResponse, _Mapping]]] = ..., usage_groups: _Optional[_Iterable[_Union[DependencyUsageGroup, _Mapping]]] = ..., findings: _Optional[_Iterable[_Union[ApprovedDependencyFinding, _Mapping]]] = ..., guidance: _Optional[str] = ...) -> None: ...

class UpsertApprovedDependencyResponse(_message.Message):
    __slots__ = ("record", "previous_record", "dry_run", "changed", "message", "summary", "guidance")
    RECORD_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_RECORD_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    CHANGED_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    GUIDANCE_FIELD_NUMBER: _ClassVar[int]
    record: ApprovedDependencyRecord
    previous_record: ApprovedDependencyRecord
    dry_run: bool
    changed: bool
    message: str
    summary: DependencyGovernanceSummary
    guidance: str
    def __init__(self, record: _Optional[_Union[ApprovedDependencyRecord, _Mapping]] = ..., previous_record: _Optional[_Union[ApprovedDependencyRecord, _Mapping]] = ..., dry_run: _Optional[bool] = ..., changed: _Optional[bool] = ..., message: _Optional[str] = ..., summary: _Optional[_Union[DependencyGovernanceSummary, _Mapping]] = ..., guidance: _Optional[str] = ...) -> None: ...

class VulnerabilityRemediationResponse(_message.Message):
    __slots__ = ("found", "vulnerability", "suggested_record", "mutation", "affected_scenarios", "source_files", "remediation", "guidance")
    FOUND_FIELD_NUMBER: _ClassVar[int]
    VULNERABILITY_FIELD_NUMBER: _ClassVar[int]
    SUGGESTED_RECORD_FIELD_NUMBER: _ClassVar[int]
    MUTATION_FIELD_NUMBER: _ClassVar[int]
    AFFECTED_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FILES_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    GUIDANCE_FIELD_NUMBER: _ClassVar[int]
    found: bool
    vulnerability: SecurityVulnerabilityEvidence
    suggested_record: ApprovedDependencyRecord
    mutation: UpsertApprovedDependencyResponse
    affected_scenarios: _containers.RepeatedScalarFieldContainer[str]
    source_files: _containers.RepeatedScalarFieldContainer[str]
    remediation: str
    guidance: str
    def __init__(self, found: _Optional[bool] = ..., vulnerability: _Optional[_Union[SecurityVulnerabilityEvidence, _Mapping]] = ..., suggested_record: _Optional[_Union[ApprovedDependencyRecord, _Mapping]] = ..., mutation: _Optional[_Union[UpsertApprovedDependencyResponse, _Mapping]] = ..., affected_scenarios: _Optional[_Iterable[str]] = ..., source_files: _Optional[_Iterable[str]] = ..., remediation: _Optional[str] = ..., guidance: _Optional[str] = ...) -> None: ...

class SecurityVulnerabilityEvidence(_message.Message):
    __slots__ = ("vulnerability_id", "aliases", "ecosystem", "package_name", "observed_version", "affected_ranges", "fixed_ranges", "severity", "normalized_severity", "advisory_url", "summary", "source", "reachability", "confidence", "production", "dev_only", "remediation")
    VULNERABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    ALIASES_FIELD_NUMBER: _ClassVar[int]
    ECOSYSTEM_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_NAME_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_VERSION_FIELD_NUMBER: _ClassVar[int]
    AFFECTED_RANGES_FIELD_NUMBER: _ClassVar[int]
    FIXED_RANGES_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    NORMALIZED_SEVERITY_FIELD_NUMBER: _ClassVar[int]
    ADVISORY_URL_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    REACHABILITY_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    PRODUCTION_FIELD_NUMBER: _ClassVar[int]
    DEV_ONLY_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    vulnerability_id: str
    aliases: _containers.RepeatedScalarFieldContainer[str]
    ecosystem: str
    package_name: str
    observed_version: str
    affected_ranges: _containers.RepeatedCompositeFieldContainer[SecurityVersionRange]
    fixed_ranges: _containers.RepeatedCompositeFieldContainer[SecurityVersionRange]
    severity: str
    normalized_severity: str
    advisory_url: str
    summary: str
    source: str
    reachability: str
    confidence: str
    production: bool
    dev_only: bool
    remediation: str
    def __init__(self, vulnerability_id: _Optional[str] = ..., aliases: _Optional[_Iterable[str]] = ..., ecosystem: _Optional[str] = ..., package_name: _Optional[str] = ..., observed_version: _Optional[str] = ..., affected_ranges: _Optional[_Iterable[_Union[SecurityVersionRange, _Mapping]]] = ..., fixed_ranges: _Optional[_Iterable[_Union[SecurityVersionRange, _Mapping]]] = ..., severity: _Optional[str] = ..., normalized_severity: _Optional[str] = ..., advisory_url: _Optional[str] = ..., summary: _Optional[str] = ..., source: _Optional[str] = ..., reachability: _Optional[str] = ..., confidence: _Optional[str] = ..., production: _Optional[bool] = ..., dev_only: _Optional[bool] = ..., remediation: _Optional[str] = ...) -> None: ...

class SecurityVersionRange(_message.Message):
    __slots__ = ("range", "version", "introduced", "fixed", "last_affected")
    RANGE_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    INTRODUCED_FIELD_NUMBER: _ClassVar[int]
    FIXED_FIELD_NUMBER: _ClassVar[int]
    LAST_AFFECTED_FIELD_NUMBER: _ClassVar[int]
    range: str
    version: str
    introduced: str
    fixed: str
    last_affected: str
    def __init__(self, range: _Optional[str] = ..., version: _Optional[str] = ..., introduced: _Optional[str] = ..., fixed: _Optional[str] = ..., last_affected: _Optional[str] = ...) -> None: ...

class DependencyGovernanceSummary(_message.Message):
    __slots__ = ("status", "approved", "approved_with_constraints", "needs_review", "blocked", "deprecated", "unrecorded", "observed", "policy_mode", "denied", "out_of_range", "out_of_scope", "expired", "scenario_count", "dependency_count", "finding_count", "error_count", "warning_count", "info_count")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    APPROVED_FIELD_NUMBER: _ClassVar[int]
    APPROVED_WITH_CONSTRAINTS_FIELD_NUMBER: _ClassVar[int]
    NEEDS_REVIEW_FIELD_NUMBER: _ClassVar[int]
    BLOCKED_FIELD_NUMBER: _ClassVar[int]
    DEPRECATED_FIELD_NUMBER: _ClassVar[int]
    UNRECORDED_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_FIELD_NUMBER: _ClassVar[int]
    POLICY_MODE_FIELD_NUMBER: _ClassVar[int]
    DENIED_FIELD_NUMBER: _ClassVar[int]
    OUT_OF_RANGE_FIELD_NUMBER: _ClassVar[int]
    OUT_OF_SCOPE_FIELD_NUMBER: _ClassVar[int]
    EXPIRED_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_COUNT_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCY_COUNT_FIELD_NUMBER: _ClassVar[int]
    FINDING_COUNT_FIELD_NUMBER: _ClassVar[int]
    ERROR_COUNT_FIELD_NUMBER: _ClassVar[int]
    WARNING_COUNT_FIELD_NUMBER: _ClassVar[int]
    INFO_COUNT_FIELD_NUMBER: _ClassVar[int]
    status: str
    approved: int
    approved_with_constraints: int
    needs_review: int
    blocked: int
    deprecated: int
    unrecorded: int
    observed: int
    policy_mode: str
    denied: int
    out_of_range: int
    out_of_scope: int
    expired: int
    scenario_count: int
    dependency_count: int
    finding_count: int
    error_count: int
    warning_count: int
    info_count: int
    def __init__(self, status: _Optional[str] = ..., approved: _Optional[int] = ..., approved_with_constraints: _Optional[int] = ..., needs_review: _Optional[int] = ..., blocked: _Optional[int] = ..., deprecated: _Optional[int] = ..., unrecorded: _Optional[int] = ..., observed: _Optional[int] = ..., policy_mode: _Optional[str] = ..., denied: _Optional[int] = ..., out_of_range: _Optional[int] = ..., out_of_scope: _Optional[int] = ..., expired: _Optional[int] = ..., scenario_count: _Optional[int] = ..., dependency_count: _Optional[int] = ..., finding_count: _Optional[int] = ..., error_count: _Optional[int] = ..., warning_count: _Optional[int] = ..., info_count: _Optional[int] = ...) -> None: ...

class ApprovedDependencyRecord(_message.Message):
    __slots__ = ("ecosystem", "package_name", "version_range", "state", "allowed_surfaces", "use_cases", "rationale", "approved_by", "approved_date", "last_reviewed", "review_expires", "license_notes", "security_notes", "example_scenarios", "replacement", "keywords", "allowed_scenarios", "denied_scenarios", "allowed_dependency_groups")
    ECOSYSTEM_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_RANGE_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_SURFACES_FIELD_NUMBER: _ClassVar[int]
    USE_CASES_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    APPROVED_BY_FIELD_NUMBER: _ClassVar[int]
    APPROVED_DATE_FIELD_NUMBER: _ClassVar[int]
    LAST_REVIEWED_FIELD_NUMBER: _ClassVar[int]
    REVIEW_EXPIRES_FIELD_NUMBER: _ClassVar[int]
    LICENSE_NOTES_FIELD_NUMBER: _ClassVar[int]
    SECURITY_NOTES_FIELD_NUMBER: _ClassVar[int]
    EXAMPLE_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    REPLACEMENT_FIELD_NUMBER: _ClassVar[int]
    KEYWORDS_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    DENIED_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_DEPENDENCY_GROUPS_FIELD_NUMBER: _ClassVar[int]
    ecosystem: str
    package_name: str
    version_range: str
    state: str
    allowed_surfaces: _containers.RepeatedScalarFieldContainer[str]
    use_cases: _containers.RepeatedScalarFieldContainer[str]
    rationale: str
    approved_by: str
    approved_date: str
    last_reviewed: str
    review_expires: str
    license_notes: str
    security_notes: str
    example_scenarios: _containers.RepeatedScalarFieldContainer[str]
    replacement: str
    keywords: _containers.RepeatedScalarFieldContainer[str]
    allowed_scenarios: _containers.RepeatedScalarFieldContainer[str]
    denied_scenarios: _containers.RepeatedScalarFieldContainer[str]
    allowed_dependency_groups: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, ecosystem: _Optional[str] = ..., package_name: _Optional[str] = ..., version_range: _Optional[str] = ..., state: _Optional[str] = ..., allowed_surfaces: _Optional[_Iterable[str]] = ..., use_cases: _Optional[_Iterable[str]] = ..., rationale: _Optional[str] = ..., approved_by: _Optional[str] = ..., approved_date: _Optional[str] = ..., last_reviewed: _Optional[str] = ..., review_expires: _Optional[str] = ..., license_notes: _Optional[str] = ..., security_notes: _Optional[str] = ..., example_scenarios: _Optional[_Iterable[str]] = ..., replacement: _Optional[str] = ..., keywords: _Optional[_Iterable[str]] = ..., allowed_scenarios: _Optional[_Iterable[str]] = ..., denied_scenarios: _Optional[_Iterable[str]] = ..., allowed_dependency_groups: _Optional[_Iterable[str]] = ...) -> None: ...

class ObservedDependency(_message.Message):
    __slots__ = ("ecosystem", "package_name", "version", "surface_id", "file_path", "dependency_group")
    ECOSYSTEM_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_NAME_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    SURFACE_ID_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCY_GROUP_FIELD_NUMBER: _ClassVar[int]
    ecosystem: str
    package_name: str
    version: str
    surface_id: str
    file_path: str
    dependency_group: str
    def __init__(self, ecosystem: _Optional[str] = ..., package_name: _Optional[str] = ..., version: _Optional[str] = ..., surface_id: _Optional[str] = ..., file_path: _Optional[str] = ..., dependency_group: _Optional[str] = ...) -> None: ...

class DependencyUsageGroup(_message.Message):
    __slots__ = ("ecosystem", "package_name", "scenario_count", "usage_count", "scenarios", "observed_dependencies", "finding_count", "highest_severity", "state")
    ECOSYSTEM_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_NAME_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_COUNT_FIELD_NUMBER: _ClassVar[int]
    USAGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    FINDING_COUNT_FIELD_NUMBER: _ClassVar[int]
    HIGHEST_SEVERITY_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    ecosystem: str
    package_name: str
    scenario_count: int
    usage_count: int
    scenarios: _containers.RepeatedScalarFieldContainer[str]
    observed_dependencies: _containers.RepeatedCompositeFieldContainer[ObservedDependency]
    finding_count: int
    highest_severity: str
    state: str
    def __init__(self, ecosystem: _Optional[str] = ..., package_name: _Optional[str] = ..., scenario_count: _Optional[int] = ..., usage_count: _Optional[int] = ..., scenarios: _Optional[_Iterable[str]] = ..., observed_dependencies: _Optional[_Iterable[_Union[ObservedDependency, _Mapping]]] = ..., finding_count: _Optional[int] = ..., highest_severity: _Optional[str] = ..., state: _Optional[str] = ...) -> None: ...

class ApprovedDependencyFinding(_message.Message):
    __slots__ = ("id", "severity", "title", "description", "remediation", "file_path", "ecosystem", "package_name", "observed", "expected", "scenario", "finding_class", "policy_mode")
    ID_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    ECOSYSTEM_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_NAME_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    FINDING_CLASS_FIELD_NUMBER: _ClassVar[int]
    POLICY_MODE_FIELD_NUMBER: _ClassVar[int]
    id: str
    severity: str
    title: str
    description: str
    remediation: str
    file_path: str
    ecosystem: str
    package_name: str
    observed: str
    expected: str
    scenario: str
    finding_class: str
    policy_mode: str
    def __init__(self, id: _Optional[str] = ..., severity: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., remediation: _Optional[str] = ..., file_path: _Optional[str] = ..., ecosystem: _Optional[str] = ..., package_name: _Optional[str] = ..., observed: _Optional[str] = ..., expected: _Optional[str] = ..., scenario: _Optional[str] = ..., finding_class: _Optional[str] = ..., policy_mode: _Optional[str] = ...) -> None: ...
