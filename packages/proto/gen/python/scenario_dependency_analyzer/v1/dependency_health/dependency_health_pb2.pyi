from common.v1 import maturity_pb2 as _maturity_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ValidateDependencyHealthRequest(_message.Message):
    __slots__ = ("scenario", "use_cache")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    USE_CACHE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    use_cache: bool
    def __init__(self, scenario: _Optional[str] = ..., use_cache: _Optional[bool] = ...) -> None: ...

class DependencyHealthResponse(_message.Message):
    __slots__ = ("scenario", "passed", "summary", "sections", "findings", "surfaces", "command_results", "governance_summary", "policy_summary", "degraded_dependencies", "generated_at", "assessment")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    SECTIONS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    COMMAND_RESULTS_FIELD_NUMBER: _ClassVar[int]
    GOVERNANCE_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    POLICY_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    passed: bool
    summary: DependencyHealthSummary
    sections: _containers.RepeatedCompositeFieldContainer[DependencyHealthSection]
    findings: _containers.RepeatedCompositeFieldContainer[DependencyHealthFinding]
    surfaces: _containers.RepeatedCompositeFieldContainer[DependencyHealthSurface]
    command_results: _containers.RepeatedCompositeFieldContainer[DependencyHealthCommandResult]
    governance_summary: DependencyGovernanceSummary
    policy_summary: DependencyPolicySummary
    degraded_dependencies: _containers.RepeatedCompositeFieldContainer[DegradedDependency]
    generated_at: str
    assessment: _maturity_pb2.MaturityAssessment
    def __init__(self, scenario: _Optional[str] = ..., passed: _Optional[bool] = ..., summary: _Optional[_Union[DependencyHealthSummary, _Mapping]] = ..., sections: _Optional[_Iterable[_Union[DependencyHealthSection, _Mapping]]] = ..., findings: _Optional[_Iterable[_Union[DependencyHealthFinding, _Mapping]]] = ..., surfaces: _Optional[_Iterable[_Union[DependencyHealthSurface, _Mapping]]] = ..., command_results: _Optional[_Iterable[_Union[DependencyHealthCommandResult, _Mapping]]] = ..., governance_summary: _Optional[_Union[DependencyGovernanceSummary, _Mapping]] = ..., policy_summary: _Optional[_Union[DependencyPolicySummary, _Mapping]] = ..., degraded_dependencies: _Optional[_Iterable[_Union[DegradedDependency, _Mapping]]] = ..., generated_at: _Optional[str] = ..., assessment: _Optional[_Union[_maturity_pb2.MaturityAssessment, _Mapping]] = ...) -> None: ...

class DependencyHealthSummary(_message.Message):
    __slots__ = ("sections", "surfaces", "findings", "errors", "warnings", "infos", "degraded_dependencies")
    SECTIONS_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    INFOS_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_DEPENDENCIES_FIELD_NUMBER: _ClassVar[int]
    sections: int
    surfaces: int
    findings: int
    errors: int
    warnings: int
    infos: int
    degraded_dependencies: int
    def __init__(self, sections: _Optional[int] = ..., surfaces: _Optional[int] = ..., findings: _Optional[int] = ..., errors: _Optional[int] = ..., warnings: _Optional[int] = ..., infos: _Optional[int] = ..., degraded_dependencies: _Optional[int] = ...) -> None: ...

class DependencyHealthSection(_message.Message):
    __slots__ = ("id", "title", "status", "summary", "finding_ids")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    FINDING_IDS_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    status: str
    summary: str
    finding_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., status: _Optional[str] = ..., summary: _Optional[str] = ..., finding_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class DependencyHealthFinding(_message.Message):
    __slots__ = ("id", "severity", "source_domain", "title", "description", "remediation", "file_path", "surface_id", "rule_id", "observed", "expected")
    ID_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    SOURCE_DOMAIN_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    SURFACE_ID_FIELD_NUMBER: _ClassVar[int]
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_FIELD_NUMBER: _ClassVar[int]
    id: str
    severity: str
    source_domain: str
    title: str
    description: str
    remediation: str
    file_path: str
    surface_id: str
    rule_id: str
    observed: str
    expected: str
    def __init__(self, id: _Optional[str] = ..., severity: _Optional[str] = ..., source_domain: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., remediation: _Optional[str] = ..., file_path: _Optional[str] = ..., surface_id: _Optional[str] = ..., rule_id: _Optional[str] = ..., observed: _Optional[str] = ..., expected: _Optional[str] = ...) -> None: ...

class DependencyHealthSurface(_message.Message):
    __slots__ = ("id", "kind", "language", "framework", "root_path", "parse_unit_root", "config_path", "status", "package_manager", "confidence")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    FRAMEWORK_FIELD_NUMBER: _ClassVar[int]
    ROOT_PATH_FIELD_NUMBER: _ClassVar[int]
    PARSE_UNIT_ROOT_FIELD_NUMBER: _ClassVar[int]
    CONFIG_PATH_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_MANAGER_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    language: str
    framework: str
    root_path: str
    parse_unit_root: str
    config_path: str
    status: str
    package_manager: str
    confidence: float
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., language: _Optional[str] = ..., framework: _Optional[str] = ..., root_path: _Optional[str] = ..., parse_unit_root: _Optional[str] = ..., config_path: _Optional[str] = ..., status: _Optional[str] = ..., package_manager: _Optional[str] = ..., confidence: _Optional[float] = ...) -> None: ...

class DependencyHealthCommandResult(_message.Message):
    __slots__ = ("id", "command", "status", "exit_code", "summary")
    ID_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    id: str
    command: str
    status: str
    exit_code: int
    summary: str
    def __init__(self, id: _Optional[str] = ..., command: _Optional[str] = ..., status: _Optional[str] = ..., exit_code: _Optional[int] = ..., summary: _Optional[str] = ...) -> None: ...

class DependencyGovernanceSummary(_message.Message):
    __slots__ = ("status", "approved", "approved_with_constraints", "needs_review", "blocked", "deprecated", "guidance")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    APPROVED_FIELD_NUMBER: _ClassVar[int]
    APPROVED_WITH_CONSTRAINTS_FIELD_NUMBER: _ClassVar[int]
    NEEDS_REVIEW_FIELD_NUMBER: _ClassVar[int]
    BLOCKED_FIELD_NUMBER: _ClassVar[int]
    DEPRECATED_FIELD_NUMBER: _ClassVar[int]
    GUIDANCE_FIELD_NUMBER: _ClassVar[int]
    status: str
    approved: int
    approved_with_constraints: int
    needs_review: int
    blocked: int
    deprecated: int
    guidance: str
    def __init__(self, status: _Optional[str] = ..., approved: _Optional[int] = ..., approved_with_constraints: _Optional[int] = ..., needs_review: _Optional[int] = ..., blocked: _Optional[int] = ..., deprecated: _Optional[int] = ..., guidance: _Optional[str] = ...) -> None: ...

class DependencyPolicySummary(_message.Message):
    __slots__ = ("status", "release_age_minimum_minutes", "release_age_exceptions", "policies")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    RELEASE_AGE_MINIMUM_MINUTES_FIELD_NUMBER: _ClassVar[int]
    RELEASE_AGE_EXCEPTIONS_FIELD_NUMBER: _ClassVar[int]
    POLICIES_FIELD_NUMBER: _ClassVar[int]
    status: str
    release_age_minimum_minutes: int
    release_age_exceptions: int
    policies: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, status: _Optional[str] = ..., release_age_minimum_minutes: _Optional[int] = ..., release_age_exceptions: _Optional[int] = ..., policies: _Optional[_Iterable[str]] = ...) -> None: ...

class DegradedDependency(_message.Message):
    __slots__ = ("id", "dependency", "domain", "reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCY_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    dependency: str
    domain: str
    reason: str
    def __init__(self, id: _Optional[str] = ..., dependency: _Optional[str] = ..., domain: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...
