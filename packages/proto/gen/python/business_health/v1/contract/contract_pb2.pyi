import datetime

from common.v1 import maturity_pb2 as _maturity_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ValidateScenarioRequest(_message.Message):
    __slots__ = ("scenario", "path", "include_execution")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_EXECUTION_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    include_execution: bool
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ..., include_execution: _Optional[bool] = ...) -> None: ...

class ValidateScenarioResponse(_message.Message):
    __slots__ = ("scenario", "status", "summary", "target_path", "degraded_reason", "report", "assessment", "next_steps")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    TARGET_PATH_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASON_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    NEXT_STEPS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    status: str
    summary: str
    target_path: str
    degraded_reason: str
    report: BusinessContractReport
    assessment: _maturity_pb2.MaturityAssessment
    next_steps: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., status: _Optional[str] = ..., summary: _Optional[str] = ..., target_path: _Optional[str] = ..., degraded_reason: _Optional[str] = ..., report: _Optional[_Union[BusinessContractReport, _Mapping]] = ..., assessment: _Optional[_Union[_maturity_pb2.MaturityAssessment, _Mapping]] = ..., next_steps: _Optional[_Iterable[str]] = ...) -> None: ...

class BusinessContractReport(_message.Message):
    __slots__ = ("capabilities", "matrix", "drift", "findings", "registry")
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    MATRIX_FIELD_NUMBER: _ClassVar[int]
    DRIFT_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    REGISTRY_FIELD_NUMBER: _ClassVar[int]
    capabilities: _containers.RepeatedCompositeFieldContainer[CapabilityRollup]
    matrix: _containers.RepeatedCompositeFieldContainer[MatrixRow]
    drift: _containers.RepeatedCompositeFieldContainer[DriftEntry]
    findings: _containers.RepeatedCompositeFieldContainer[ContractFinding]
    registry: RegistrySummary
    def __init__(self, capabilities: _Optional[_Iterable[_Union[CapabilityRollup, _Mapping]]] = ..., matrix: _Optional[_Iterable[_Union[MatrixRow, _Mapping]]] = ..., drift: _Optional[_Iterable[_Union[DriftEntry, _Mapping]]] = ..., findings: _Optional[_Iterable[_Union[ContractFinding, _Mapping]]] = ..., registry: _Optional[_Union[RegistrySummary, _Mapping]] = ...) -> None: ...

class CapabilityRollup(_message.Message):
    __slots__ = ("capability_id", "level_id", "level_name", "finding_count", "error_count", "warning_count")
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    LEVEL_ID_FIELD_NUMBER: _ClassVar[int]
    LEVEL_NAME_FIELD_NUMBER: _ClassVar[int]
    FINDING_COUNT_FIELD_NUMBER: _ClassVar[int]
    ERROR_COUNT_FIELD_NUMBER: _ClassVar[int]
    WARNING_COUNT_FIELD_NUMBER: _ClassVar[int]
    capability_id: str
    level_id: str
    level_name: str
    finding_count: int
    error_count: int
    warning_count: int
    def __init__(self, capability_id: _Optional[str] = ..., level_id: _Optional[str] = ..., level_name: _Optional[str] = ..., finding_count: _Optional[int] = ..., error_count: _Optional[int] = ..., warning_count: _Optional[int] = ...) -> None: ...

class MatrixRow(_message.Message):
    __slots__ = ("ot_id", "ot_title", "ot_checked", "ot_priority", "requirement_id", "requirement_title", "requirement_status", "criticality", "validations", "evidence", "unproven", "unproven_reason")
    OT_ID_FIELD_NUMBER: _ClassVar[int]
    OT_TITLE_FIELD_NUMBER: _ClassVar[int]
    OT_CHECKED_FIELD_NUMBER: _ClassVar[int]
    OT_PRIORITY_FIELD_NUMBER: _ClassVar[int]
    REQUIREMENT_ID_FIELD_NUMBER: _ClassVar[int]
    REQUIREMENT_TITLE_FIELD_NUMBER: _ClassVar[int]
    REQUIREMENT_STATUS_FIELD_NUMBER: _ClassVar[int]
    CRITICALITY_FIELD_NUMBER: _ClassVar[int]
    VALIDATIONS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    UNPROVEN_FIELD_NUMBER: _ClassVar[int]
    UNPROVEN_REASON_FIELD_NUMBER: _ClassVar[int]
    ot_id: str
    ot_title: str
    ot_checked: bool
    ot_priority: str
    requirement_id: str
    requirement_title: str
    requirement_status: str
    criticality: str
    validations: _containers.RepeatedCompositeFieldContainer[ValidationCell]
    evidence: EvidenceCell
    unproven: bool
    unproven_reason: str
    def __init__(self, ot_id: _Optional[str] = ..., ot_title: _Optional[str] = ..., ot_checked: _Optional[bool] = ..., ot_priority: _Optional[str] = ..., requirement_id: _Optional[str] = ..., requirement_title: _Optional[str] = ..., requirement_status: _Optional[str] = ..., criticality: _Optional[str] = ..., validations: _Optional[_Iterable[_Union[ValidationCell, _Mapping]]] = ..., evidence: _Optional[_Union[EvidenceCell, _Mapping]] = ..., unproven: _Optional[bool] = ..., unproven_reason: _Optional[str] = ...) -> None: ...

class ValidationCell(_message.Message):
    __slots__ = ("type", "phase", "status", "ref", "ref_exists")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REF_FIELD_NUMBER: _ClassVar[int]
    REF_EXISTS_FIELD_NUMBER: _ClassVar[int]
    type: str
    phase: str
    status: str
    ref: str
    ref_exists: bool
    def __init__(self, type: _Optional[str] = ..., phase: _Optional[str] = ..., status: _Optional[str] = ..., ref: _Optional[str] = ..., ref_exists: _Optional[bool] = ...) -> None: ...

class EvidenceCell(_message.Message):
    __slots__ = ("live_status", "last_run_id", "last_synced_at", "stale", "manual")
    LIVE_STATUS_FIELD_NUMBER: _ClassVar[int]
    LAST_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    LAST_SYNCED_AT_FIELD_NUMBER: _ClassVar[int]
    STALE_FIELD_NUMBER: _ClassVar[int]
    MANUAL_FIELD_NUMBER: _ClassVar[int]
    live_status: str
    last_run_id: str
    last_synced_at: _timestamp_pb2.Timestamp
    stale: bool
    manual: ManualAttestation
    def __init__(self, live_status: _Optional[str] = ..., last_run_id: _Optional[str] = ..., last_synced_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., stale: _Optional[bool] = ..., manual: _Optional[_Union[ManualAttestation, _Mapping]] = ...) -> None: ...

class ManualAttestation(_message.Message):
    __slots__ = ("requirement_id", "attested_by", "attested_at", "expires_at", "expired", "notes")
    REQUIREMENT_ID_FIELD_NUMBER: _ClassVar[int]
    ATTESTED_BY_FIELD_NUMBER: _ClassVar[int]
    ATTESTED_AT_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    EXPIRED_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    requirement_id: str
    attested_by: str
    attested_at: _timestamp_pb2.Timestamp
    expires_at: _timestamp_pb2.Timestamp
    expired: bool
    notes: str
    def __init__(self, requirement_id: _Optional[str] = ..., attested_by: _Optional[str] = ..., attested_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., expired: _Optional[bool] = ..., notes: _Optional[str] = ...) -> None: ...

class DriftEntry(_message.Message):
    __slots__ = ("kind", "subject_id", "detail")
    KIND_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_ID_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    kind: str
    subject_id: str
    detail: str
    def __init__(self, kind: _Optional[str] = ..., subject_id: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class ContractFinding(_message.Message):
    __slots__ = ("code", "severity", "title", "message", "location", "remediation", "autofix_available", "fix_class")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    AUTOFIX_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    FIX_CLASS_FIELD_NUMBER: _ClassVar[int]
    code: str
    severity: str
    title: str
    message: str
    location: str
    remediation: str
    autofix_available: bool
    fix_class: str
    def __init__(self, code: _Optional[str] = ..., severity: _Optional[str] = ..., title: _Optional[str] = ..., message: _Optional[str] = ..., location: _Optional[str] = ..., remediation: _Optional[str] = ..., autofix_available: _Optional[bool] = ..., fix_class: _Optional[str] = ...) -> None: ...

class RegistrySummary(_message.Message):
    __slots__ = ("module_count", "requirement_count", "operational_target_count", "status_counts", "starter_template")
    class StatusCountsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    MODULE_COUNT_FIELD_NUMBER: _ClassVar[int]
    REQUIREMENT_COUNT_FIELD_NUMBER: _ClassVar[int]
    OPERATIONAL_TARGET_COUNT_FIELD_NUMBER: _ClassVar[int]
    STATUS_COUNTS_FIELD_NUMBER: _ClassVar[int]
    STARTER_TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    module_count: int
    requirement_count: int
    operational_target_count: int
    status_counts: _containers.ScalarMap[str, int]
    starter_template: bool
    def __init__(self, module_count: _Optional[int] = ..., requirement_count: _Optional[int] = ..., operational_target_count: _Optional[int] = ..., status_counts: _Optional[_Mapping[str, int]] = ..., starter_template: _Optional[bool] = ...) -> None: ...

class GetMatrixRequest(_message.Message):
    __slots__ = ("scenario", "path")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class GetMatrixResponse(_message.Message):
    __slots__ = ("scenario", "matrix", "registry", "degraded_reason")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    MATRIX_FIELD_NUMBER: _ClassVar[int]
    REGISTRY_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASON_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    matrix: _containers.RepeatedCompositeFieldContainer[MatrixRow]
    registry: RegistrySummary
    degraded_reason: str
    def __init__(self, scenario: _Optional[str] = ..., matrix: _Optional[_Iterable[_Union[MatrixRow, _Mapping]]] = ..., registry: _Optional[_Union[RegistrySummary, _Mapping]] = ..., degraded_reason: _Optional[str] = ...) -> None: ...

class GetDriftRequest(_message.Message):
    __slots__ = ("scenario", "path")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class GetDriftResponse(_message.Message):
    __slots__ = ("scenario", "drift")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DRIFT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    drift: _containers.RepeatedCompositeFieldContainer[DriftEntry]
    def __init__(self, scenario: _Optional[str] = ..., drift: _Optional[_Iterable[_Union[DriftEntry, _Mapping]]] = ...) -> None: ...

class LogManualValidationRequest(_message.Message):
    __slots__ = ("scenario", "requirement_id", "attested_by", "notes")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    REQUIREMENT_ID_FIELD_NUMBER: _ClassVar[int]
    ATTESTED_BY_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    requirement_id: str
    attested_by: str
    notes: str
    def __init__(self, scenario: _Optional[str] = ..., requirement_id: _Optional[str] = ..., attested_by: _Optional[str] = ..., notes: _Optional[str] = ...) -> None: ...

class LogManualValidationResponse(_message.Message):
    __slots__ = ("attestation", "ledger_path")
    ATTESTATION_FIELD_NUMBER: _ClassVar[int]
    LEDGER_PATH_FIELD_NUMBER: _ClassVar[int]
    attestation: ManualAttestation
    ledger_path: str
    def __init__(self, attestation: _Optional[_Union[ManualAttestation, _Mapping]] = ..., ledger_path: _Optional[str] = ...) -> None: ...
