from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class QualityCheckStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    QUALITY_CHECK_STATUS_UNSPECIFIED: _ClassVar[QualityCheckStatus]
    QUALITY_CHECK_STATUS_VIOLATION: _ClassVar[QualityCheckStatus]
    QUALITY_CHECK_STATUS_CHECKED_CLEAN: _ClassVar[QualityCheckStatus]
    QUALITY_CHECK_STATUS_UNKNOWN: _ClassVar[QualityCheckStatus]
    QUALITY_CHECK_STATUS_NOT_APPLICABLE: _ClassVar[QualityCheckStatus]

class QualityReason(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    QUALITY_REASON_UNSPECIFIED: _ClassVar[QualityReason]
    QUALITY_REASON_NONE: _ClassVar[QualityReason]
    QUALITY_REASON_MISSING_ANALYSIS: _ClassVar[QualityReason]
    QUALITY_REASON_UNSUPPORTED_ADAPTER: _ClassVar[QualityReason]
    QUALITY_REASON_UNSUPPORTED_VERSION: _ClassVar[QualityReason]
    QUALITY_REASON_UNSUPPORTED_TEST_KIND: _ClassVar[QualityReason]
    QUALITY_REASON_MISSING_INPUT: _ClassVar[QualityReason]
    QUALITY_REASON_PARSE_FAILURE: _ClassVar[QualityReason]
    QUALITY_REASON_EXTERNAL_HELPER_UNRESOLVED: _ClassVar[QualityReason]
    QUALITY_REASON_RESOLUTION_LIMIT: _ClassVar[QualityReason]
    QUALITY_REASON_BUILD_CONTEXT_UNAVAILABLE: _ClassVar[QualityReason]
    QUALITY_REASON_NOT_EXECUTED: _ClassVar[QualityReason]
    QUALITY_REASON_SKIPPED: _ClassVar[QualityReason]
    QUALITY_REASON_OWNER_UNAVAILABLE: _ClassVar[QualityReason]
    QUALITY_REASON_STALE_EVIDENCE: _ClassVar[QualityReason]
    QUALITY_REASON_UNRECOGNIZED_VALUE: _ClassVar[QualityReason]

class QualityEvidenceKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    QUALITY_EVIDENCE_KIND_UNSPECIFIED: _ClassVar[QualityEvidenceKind]
    QUALITY_EVIDENCE_KIND_STATIC: _ClassVar[QualityEvidenceKind]
    QUALITY_EVIDENCE_KIND_RUNTIME: _ClassVar[QualityEvidenceKind]
    QUALITY_EVIDENCE_KIND_REGISTRY: _ClassVar[QualityEvidenceKind]

class QualitySeverity(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    QUALITY_SEVERITY_UNSPECIFIED: _ClassVar[QualitySeverity]
    QUALITY_SEVERITY_INFO: _ClassVar[QualitySeverity]
    QUALITY_SEVERITY_WARNING: _ClassVar[QualitySeverity]
    QUALITY_SEVERITY_ERROR: _ClassVar[QualitySeverity]

class QualityEnforcement(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    QUALITY_ENFORCEMENT_UNSPECIFIED: _ClassVar[QualityEnforcement]
    QUALITY_ENFORCEMENT_ADVISORY: _ClassVar[QualityEnforcement]
    QUALITY_ENFORCEMENT_BLOCKING: _ClassVar[QualityEnforcement]

class RequirementRegistration(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    REQUIREMENT_REGISTRATION_UNSPECIFIED: _ClassVar[RequirementRegistration]
    REQUIREMENT_REGISTRATION_REGISTERED: _ClassVar[RequirementRegistration]
    REQUIREMENT_REGISTRATION_STALE: _ClassVar[RequirementRegistration]
    REQUIREMENT_REGISTRATION_UNKNOWN: _ClassVar[RequirementRegistration]

class RequirementExecutionState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    REQUIREMENT_EXECUTION_STATE_UNSPECIFIED: _ClassVar[RequirementExecutionState]
    REQUIREMENT_EXECUTION_STATE_PASSED: _ClassVar[RequirementExecutionState]
    REQUIREMENT_EXECUTION_STATE_FAILED: _ClassVar[RequirementExecutionState]
    REQUIREMENT_EXECUTION_STATE_SKIPPED: _ClassVar[RequirementExecutionState]
    REQUIREMENT_EXECUTION_STATE_NOT_RUN: _ClassVar[RequirementExecutionState]
    REQUIREMENT_EXECUTION_STATE_UNKNOWN: _ClassVar[RequirementExecutionState]

class RequirementApplicability(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    REQUIREMENT_APPLICABILITY_UNSPECIFIED: _ClassVar[RequirementApplicability]
    REQUIREMENT_APPLICABILITY_APPLICABLE: _ClassVar[RequirementApplicability]
    REQUIREMENT_APPLICABILITY_NOT_APPLICABLE: _ClassVar[RequirementApplicability]
    REQUIREMENT_APPLICABILITY_UNKNOWN: _ClassVar[RequirementApplicability]
QUALITY_CHECK_STATUS_UNSPECIFIED: QualityCheckStatus
QUALITY_CHECK_STATUS_VIOLATION: QualityCheckStatus
QUALITY_CHECK_STATUS_CHECKED_CLEAN: QualityCheckStatus
QUALITY_CHECK_STATUS_UNKNOWN: QualityCheckStatus
QUALITY_CHECK_STATUS_NOT_APPLICABLE: QualityCheckStatus
QUALITY_REASON_UNSPECIFIED: QualityReason
QUALITY_REASON_NONE: QualityReason
QUALITY_REASON_MISSING_ANALYSIS: QualityReason
QUALITY_REASON_UNSUPPORTED_ADAPTER: QualityReason
QUALITY_REASON_UNSUPPORTED_VERSION: QualityReason
QUALITY_REASON_UNSUPPORTED_TEST_KIND: QualityReason
QUALITY_REASON_MISSING_INPUT: QualityReason
QUALITY_REASON_PARSE_FAILURE: QualityReason
QUALITY_REASON_EXTERNAL_HELPER_UNRESOLVED: QualityReason
QUALITY_REASON_RESOLUTION_LIMIT: QualityReason
QUALITY_REASON_BUILD_CONTEXT_UNAVAILABLE: QualityReason
QUALITY_REASON_NOT_EXECUTED: QualityReason
QUALITY_REASON_SKIPPED: QualityReason
QUALITY_REASON_OWNER_UNAVAILABLE: QualityReason
QUALITY_REASON_STALE_EVIDENCE: QualityReason
QUALITY_REASON_UNRECOGNIZED_VALUE: QualityReason
QUALITY_EVIDENCE_KIND_UNSPECIFIED: QualityEvidenceKind
QUALITY_EVIDENCE_KIND_STATIC: QualityEvidenceKind
QUALITY_EVIDENCE_KIND_RUNTIME: QualityEvidenceKind
QUALITY_EVIDENCE_KIND_REGISTRY: QualityEvidenceKind
QUALITY_SEVERITY_UNSPECIFIED: QualitySeverity
QUALITY_SEVERITY_INFO: QualitySeverity
QUALITY_SEVERITY_WARNING: QualitySeverity
QUALITY_SEVERITY_ERROR: QualitySeverity
QUALITY_ENFORCEMENT_UNSPECIFIED: QualityEnforcement
QUALITY_ENFORCEMENT_ADVISORY: QualityEnforcement
QUALITY_ENFORCEMENT_BLOCKING: QualityEnforcement
REQUIREMENT_REGISTRATION_UNSPECIFIED: RequirementRegistration
REQUIREMENT_REGISTRATION_REGISTERED: RequirementRegistration
REQUIREMENT_REGISTRATION_STALE: RequirementRegistration
REQUIREMENT_REGISTRATION_UNKNOWN: RequirementRegistration
REQUIREMENT_EXECUTION_STATE_UNSPECIFIED: RequirementExecutionState
REQUIREMENT_EXECUTION_STATE_PASSED: RequirementExecutionState
REQUIREMENT_EXECUTION_STATE_FAILED: RequirementExecutionState
REQUIREMENT_EXECUTION_STATE_SKIPPED: RequirementExecutionState
REQUIREMENT_EXECUTION_STATE_NOT_RUN: RequirementExecutionState
REQUIREMENT_EXECUTION_STATE_UNKNOWN: RequirementExecutionState
REQUIREMENT_APPLICABILITY_UNSPECIFIED: RequirementApplicability
REQUIREMENT_APPLICABILITY_APPLICABLE: RequirementApplicability
REQUIREMENT_APPLICABILITY_NOT_APPLICABLE: RequirementApplicability
REQUIREMENT_APPLICABILITY_UNKNOWN: RequirementApplicability

class QualityTestTarget(_message.Message):
    __slots__ = ("workspace", "file", "test_id", "scope")
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    FILE_FIELD_NUMBER: _ClassVar[int]
    TEST_ID_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    workspace: str
    file: str
    test_id: str
    scope: str
    def __init__(self, workspace: _Optional[str] = ..., file: _Optional[str] = ..., test_id: _Optional[str] = ..., scope: _Optional[str] = ...) -> None: ...

class QualitySourceLocation(_message.Message):
    __slots__ = ("line", "column", "end_line", "end_column")
    LINE_FIELD_NUMBER: _ClassVar[int]
    COLUMN_FIELD_NUMBER: _ClassVar[int]
    END_LINE_FIELD_NUMBER: _ClassVar[int]
    END_COLUMN_FIELD_NUMBER: _ClassVar[int]
    line: int
    column: int
    end_line: int
    end_column: int
    def __init__(self, line: _Optional[int] = ..., column: _Optional[int] = ..., end_line: _Optional[int] = ..., end_column: _Optional[int] = ...) -> None: ...

class QualityCheckResult(_message.Message):
    __slots__ = ("rule_id", "rule_version", "target", "test_kind", "support_profile", "status", "reason", "evidence_kind", "severity", "enforcement", "location", "evidence_refs", "limitations", "diagnostics", "runtime_observation", "reason_guidance")
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    RULE_VERSION_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    TEST_KIND_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_PROFILE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_KIND_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    ENFORCEMENT_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    LIMITATIONS_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTICS_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_OBSERVATION_FIELD_NUMBER: _ClassVar[int]
    REASON_GUIDANCE_FIELD_NUMBER: _ClassVar[int]
    rule_id: str
    rule_version: str
    target: QualityTestTarget
    test_kind: str
    support_profile: str
    status: QualityCheckStatus
    reason: QualityReason
    evidence_kind: QualityEvidenceKind
    severity: QualitySeverity
    enforcement: QualityEnforcement
    location: QualitySourceLocation
    evidence_refs: _containers.RepeatedScalarFieldContainer[str]
    limitations: _containers.RepeatedScalarFieldContainer[str]
    diagnostics: _containers.RepeatedCompositeFieldContainer[QualityNativeDiagnostic]
    runtime_observation: QualityRuntimeObservation
    reason_guidance: str
    def __init__(self, rule_id: _Optional[str] = ..., rule_version: _Optional[str] = ..., target: _Optional[_Union[QualityTestTarget, _Mapping]] = ..., test_kind: _Optional[str] = ..., support_profile: _Optional[str] = ..., status: _Optional[_Union[QualityCheckStatus, str]] = ..., reason: _Optional[_Union[QualityReason, str]] = ..., evidence_kind: _Optional[_Union[QualityEvidenceKind, str]] = ..., severity: _Optional[_Union[QualitySeverity, str]] = ..., enforcement: _Optional[_Union[QualityEnforcement, str]] = ..., location: _Optional[_Union[QualitySourceLocation, _Mapping]] = ..., evidence_refs: _Optional[_Iterable[str]] = ..., limitations: _Optional[_Iterable[str]] = ..., diagnostics: _Optional[_Iterable[_Union[QualityNativeDiagnostic, _Mapping]]] = ..., runtime_observation: _Optional[_Union[QualityRuntimeObservation, _Mapping]] = ..., reason_guidance: _Optional[str] = ...) -> None: ...

class QualityRuntimeObservation(_message.Message):
    __slots__ = ("run_id", "state", "seed", "retry_count", "retry_ordinal")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    RETRY_COUNT_FIELD_NUMBER: _ClassVar[int]
    RETRY_ORDINAL_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    state: str
    seed: str
    retry_count: int
    retry_ordinal: int
    def __init__(self, run_id: _Optional[str] = ..., state: _Optional[str] = ..., seed: _Optional[str] = ..., retry_count: _Optional[int] = ..., retry_ordinal: _Optional[int] = ...) -> None: ...

class QualityNativeDiagnostic(_message.Message):
    __slots__ = ("native_rule_id", "message_id", "message", "native_severity", "location")
    NATIVE_RULE_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    NATIVE_SEVERITY_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    native_rule_id: str
    message_id: str
    message: str
    native_severity: int
    location: QualitySourceLocation
    def __init__(self, native_rule_id: _Optional[str] = ..., message_id: _Optional[str] = ..., message: _Optional[str] = ..., native_severity: _Optional[int] = ..., location: _Optional[_Union[QualitySourceLocation, _Mapping]] = ...) -> None: ...

class QualityAssessmentCoverage(_message.Message):
    __slots__ = ("rule_id", "support_profile", "discovered", "assessed", "unknown", "not_applicable", "scope")
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_PROFILE_FIELD_NUMBER: _ClassVar[int]
    DISCOVERED_FIELD_NUMBER: _ClassVar[int]
    ASSESSED_FIELD_NUMBER: _ClassVar[int]
    UNKNOWN_FIELD_NUMBER: _ClassVar[int]
    NOT_APPLICABLE_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    rule_id: str
    support_profile: str
    discovered: int
    assessed: int
    unknown: int
    not_applicable: int
    scope: str
    def __init__(self, rule_id: _Optional[str] = ..., support_profile: _Optional[str] = ..., discovered: _Optional[int] = ..., assessed: _Optional[int] = ..., unknown: _Optional[int] = ..., not_applicable: _Optional[int] = ..., scope: _Optional[str] = ...) -> None: ...

class TestQualityReport(_message.Message):
    __slots__ = ("schema_version", "catalog_version", "results", "coverage", "total_results", "truncated", "details_ref", "unavailable_reason", "reason_guidance", "collection_limitations")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    CATALOG_VERSION_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    COVERAGE_FIELD_NUMBER: _ClassVar[int]
    TOTAL_RESULTS_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    DETAILS_REF_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_REASON_FIELD_NUMBER: _ClassVar[int]
    REASON_GUIDANCE_FIELD_NUMBER: _ClassVar[int]
    COLLECTION_LIMITATIONS_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    catalog_version: str
    results: _containers.RepeatedCompositeFieldContainer[QualityCheckResult]
    coverage: _containers.RepeatedCompositeFieldContainer[QualityAssessmentCoverage]
    total_results: int
    truncated: bool
    details_ref: str
    unavailable_reason: QualityReason
    reason_guidance: str
    collection_limitations: _containers.RepeatedCompositeFieldContainer[QualityCollectionLimitation]
    def __init__(self, schema_version: _Optional[str] = ..., catalog_version: _Optional[str] = ..., results: _Optional[_Iterable[_Union[QualityCheckResult, _Mapping]]] = ..., coverage: _Optional[_Iterable[_Union[QualityAssessmentCoverage, _Mapping]]] = ..., total_results: _Optional[int] = ..., truncated: _Optional[bool] = ..., details_ref: _Optional[str] = ..., unavailable_reason: _Optional[_Union[QualityReason, str]] = ..., reason_guidance: _Optional[str] = ..., collection_limitations: _Optional[_Iterable[_Union[QualityCollectionLimitation, _Mapping]]] = ...) -> None: ...

class QualityCollectionLimitation(_message.Message):
    __slots__ = ("reason", "guidance")
    REASON_FIELD_NUMBER: _ClassVar[int]
    GUIDANCE_FIELD_NUMBER: _ClassVar[int]
    reason: QualityReason
    guidance: str
    def __init__(self, reason: _Optional[_Union[QualityReason, str]] = ..., guidance: _Optional[str] = ...) -> None: ...

class RequirementTraceLink(_message.Message):
    __slots__ = ("requirement_id", "target", "registration", "execution", "reason", "run_id")
    REQUIREMENT_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    REGISTRATION_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    requirement_id: str
    target: QualityTestTarget
    registration: RequirementRegistration
    execution: RequirementExecutionState
    reason: QualityReason
    run_id: str
    def __init__(self, requirement_id: _Optional[str] = ..., target: _Optional[_Union[QualityTestTarget, _Mapping]] = ..., registration: _Optional[_Union[RequirementRegistration, str]] = ..., execution: _Optional[_Union[RequirementExecutionState, str]] = ..., reason: _Optional[_Union[QualityReason, str]] = ..., run_id: _Optional[str] = ...) -> None: ...

class RequirementTraceScope(_message.Message):
    __slots__ = ("requirement_id", "applicability")
    REQUIREMENT_ID_FIELD_NUMBER: _ClassVar[int]
    APPLICABILITY_FIELD_NUMBER: _ClassVar[int]
    requirement_id: str
    applicability: RequirementApplicability
    def __init__(self, requirement_id: _Optional[str] = ..., applicability: _Optional[_Union[RequirementApplicability, str]] = ...) -> None: ...

class RequirementTraceabilityReport(_message.Message):
    __slots__ = ("schema_version", "unavailable_reason", "evidence_unavailable_reason", "requirements", "links", "limitations")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_REASON_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_UNAVAILABLE_REASON_FIELD_NUMBER: _ClassVar[int]
    REQUIREMENTS_FIELD_NUMBER: _ClassVar[int]
    LINKS_FIELD_NUMBER: _ClassVar[int]
    LIMITATIONS_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    unavailable_reason: QualityReason
    evidence_unavailable_reason: QualityReason
    requirements: _containers.RepeatedCompositeFieldContainer[RequirementTraceScope]
    links: _containers.RepeatedCompositeFieldContainer[RequirementTraceLink]
    limitations: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, schema_version: _Optional[str] = ..., unavailable_reason: _Optional[_Union[QualityReason, str]] = ..., evidence_unavailable_reason: _Optional[_Union[QualityReason, str]] = ..., requirements: _Optional[_Iterable[_Union[RequirementTraceScope, _Mapping]]] = ..., links: _Optional[_Iterable[_Union[RequirementTraceLink, _Mapping]]] = ..., limitations: _Optional[_Iterable[str]] = ...) -> None: ...
