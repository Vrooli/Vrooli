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
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

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

class DependencyGovernanceSummary(_message.Message):
    __slots__ = ("status", "approved", "approved_with_constraints", "needs_review", "blocked", "deprecated", "unrecorded", "observed")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    APPROVED_FIELD_NUMBER: _ClassVar[int]
    APPROVED_WITH_CONSTRAINTS_FIELD_NUMBER: _ClassVar[int]
    NEEDS_REVIEW_FIELD_NUMBER: _ClassVar[int]
    BLOCKED_FIELD_NUMBER: _ClassVar[int]
    DEPRECATED_FIELD_NUMBER: _ClassVar[int]
    UNRECORDED_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_FIELD_NUMBER: _ClassVar[int]
    status: str
    approved: int
    approved_with_constraints: int
    needs_review: int
    blocked: int
    deprecated: int
    unrecorded: int
    observed: int
    def __init__(self, status: _Optional[str] = ..., approved: _Optional[int] = ..., approved_with_constraints: _Optional[int] = ..., needs_review: _Optional[int] = ..., blocked: _Optional[int] = ..., deprecated: _Optional[int] = ..., unrecorded: _Optional[int] = ..., observed: _Optional[int] = ...) -> None: ...

class ApprovedDependencyRecord(_message.Message):
    __slots__ = ("ecosystem", "package_name", "version_range", "state", "allowed_surfaces", "use_cases", "rationale", "approved_by", "approved_date", "last_reviewed", "review_expires", "license_notes", "security_notes", "example_scenarios", "replacement", "keywords")
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
    def __init__(self, ecosystem: _Optional[str] = ..., package_name: _Optional[str] = ..., version_range: _Optional[str] = ..., state: _Optional[str] = ..., allowed_surfaces: _Optional[_Iterable[str]] = ..., use_cases: _Optional[_Iterable[str]] = ..., rationale: _Optional[str] = ..., approved_by: _Optional[str] = ..., approved_date: _Optional[str] = ..., last_reviewed: _Optional[str] = ..., review_expires: _Optional[str] = ..., license_notes: _Optional[str] = ..., security_notes: _Optional[str] = ..., example_scenarios: _Optional[_Iterable[str]] = ..., replacement: _Optional[str] = ..., keywords: _Optional[_Iterable[str]] = ...) -> None: ...

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

class ApprovedDependencyFinding(_message.Message):
    __slots__ = ("id", "severity", "title", "description", "remediation", "file_path", "ecosystem", "package_name", "observed", "expected")
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
    def __init__(self, id: _Optional[str] = ..., severity: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., remediation: _Optional[str] = ..., file_path: _Optional[str] = ..., ecosystem: _Optional[str] = ..., package_name: _Optional[str] = ..., observed: _Optional[str] = ..., expected: _Optional[str] = ...) -> None: ...
