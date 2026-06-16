from common.v1 import maturity_pb2 as _maturity_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AuditQualityRequest(_message.Message):
    __slots__ = ("scenario", "path", "rule_ids", "surfaces", "include_command_execution", "include_autofix_preview", "use_cache")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    RULE_IDS_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_COMMAND_EXECUTION_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_AUTOFIX_PREVIEW_FIELD_NUMBER: _ClassVar[int]
    USE_CACHE_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    rule_ids: _containers.RepeatedScalarFieldContainer[str]
    surfaces: _containers.RepeatedScalarFieldContainer[str]
    include_command_execution: bool
    include_autofix_preview: bool
    use_cache: bool
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ..., rule_ids: _Optional[_Iterable[str]] = ..., surfaces: _Optional[_Iterable[str]] = ..., include_command_execution: _Optional[bool] = ..., include_autofix_preview: _Optional[bool] = ..., use_cache: _Optional[bool] = ...) -> None: ...

class AuditQualityResponse(_message.Message):
    __slots__ = ("run_id", "status", "summary", "scenario", "target_kind", "target_path", "surfaces", "contracts", "findings", "command_results", "maturity", "counts", "next_steps", "degraded_reason", "autofix_candidates", "assessment")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TARGET_KIND_FIELD_NUMBER: _ClassVar[int]
    TARGET_PATH_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    CONTRACTS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    COMMAND_RESULTS_FIELD_NUMBER: _ClassVar[int]
    MATURITY_FIELD_NUMBER: _ClassVar[int]
    COUNTS_FIELD_NUMBER: _ClassVar[int]
    NEXT_STEPS_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASON_FIELD_NUMBER: _ClassVar[int]
    AUTOFIX_CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    status: str
    summary: str
    scenario: str
    target_kind: str
    target_path: str
    surfaces: _containers.RepeatedCompositeFieldContainer[QualitySurface]
    contracts: _containers.RepeatedCompositeFieldContainer[ContractEvaluation]
    findings: _containers.RepeatedCompositeFieldContainer[QualityFinding]
    command_results: _containers.RepeatedCompositeFieldContainer[CommandResult]
    maturity: MaturitySummary
    counts: AuditSummary
    next_steps: _containers.RepeatedScalarFieldContainer[str]
    degraded_reason: str
    autofix_candidates: _containers.RepeatedCompositeFieldContainer[AutofixCandidate]
    assessment: _maturity_pb2.MaturityAssessment
    def __init__(self, run_id: _Optional[str] = ..., status: _Optional[str] = ..., summary: _Optional[str] = ..., scenario: _Optional[str] = ..., target_kind: _Optional[str] = ..., target_path: _Optional[str] = ..., surfaces: _Optional[_Iterable[_Union[QualitySurface, _Mapping]]] = ..., contracts: _Optional[_Iterable[_Union[ContractEvaluation, _Mapping]]] = ..., findings: _Optional[_Iterable[_Union[QualityFinding, _Mapping]]] = ..., command_results: _Optional[_Iterable[_Union[CommandResult, _Mapping]]] = ..., maturity: _Optional[_Union[MaturitySummary, _Mapping]] = ..., counts: _Optional[_Union[AuditSummary, _Mapping]] = ..., next_steps: _Optional[_Iterable[str]] = ..., degraded_reason: _Optional[str] = ..., autofix_candidates: _Optional[_Iterable[_Union[AutofixCandidate, _Mapping]]] = ..., assessment: _Optional[_Union[_maturity_pb2.MaturityAssessment, _Mapping]] = ...) -> None: ...

class ListContractsRequest(_message.Message):
    __slots__ = ("language", "framework", "surface_kind", "rule_ids")
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    FRAMEWORK_FIELD_NUMBER: _ClassVar[int]
    SURFACE_KIND_FIELD_NUMBER: _ClassVar[int]
    RULE_IDS_FIELD_NUMBER: _ClassVar[int]
    language: str
    framework: str
    surface_kind: str
    rule_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, language: _Optional[str] = ..., framework: _Optional[str] = ..., surface_kind: _Optional[str] = ..., rule_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class ListContractsResponse(_message.Message):
    __slots__ = ("contracts",)
    CONTRACTS_FIELD_NUMBER: _ClassVar[int]
    contracts: _containers.RepeatedCompositeFieldContainer[QualityContract]
    def __init__(self, contracts: _Optional[_Iterable[_Union[QualityContract, _Mapping]]] = ...) -> None: ...

class ExplainFindingRequest(_message.Message):
    __slots__ = ("finding_id", "rule_id", "scenario")
    FINDING_ID_FIELD_NUMBER: _ClassVar[int]
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    finding_id: str
    rule_id: str
    scenario: str
    def __init__(self, finding_id: _Optional[str] = ..., rule_id: _Optional[str] = ..., scenario: _Optional[str] = ...) -> None: ...

class ExplainFindingResponse(_message.Message):
    __slots__ = ("finding", "contract", "why_it_matters", "remediation", "next_steps")
    FINDING_FIELD_NUMBER: _ClassVar[int]
    CONTRACT_FIELD_NUMBER: _ClassVar[int]
    WHY_IT_MATTERS_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    NEXT_STEPS_FIELD_NUMBER: _ClassVar[int]
    finding: QualityFinding
    contract: QualityContract
    why_it_matters: str
    remediation: str
    next_steps: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, finding: _Optional[_Union[QualityFinding, _Mapping]] = ..., contract: _Optional[_Union[QualityContract, _Mapping]] = ..., why_it_matters: _Optional[str] = ..., remediation: _Optional[str] = ..., next_steps: _Optional[_Iterable[str]] = ...) -> None: ...

class FixConfigRequest(_message.Message):
    __slots__ = ("scenario", "path", "rule_ids", "apply")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    RULE_IDS_FIELD_NUMBER: _ClassVar[int]
    APPLY_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    rule_ids: _containers.RepeatedScalarFieldContainer[str]
    apply: bool
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ..., rule_ids: _Optional[_Iterable[str]] = ..., apply: _Optional[bool] = ...) -> None: ...

class FixConfigResponse(_message.Message):
    __slots__ = ("scenario", "applied", "candidates", "messages")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    MESSAGES_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    applied: bool
    candidates: _containers.RepeatedCompositeFieldContainer[AutofixCandidate]
    messages: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., applied: _Optional[bool] = ..., candidates: _Optional[_Iterable[_Union[AutofixCandidate, _Mapping]]] = ..., messages: _Optional[_Iterable[str]] = ...) -> None: ...

class QualitySurface(_message.Message):
    __slots__ = ("id", "kind", "language", "framework", "root_path", "package_manager", "status", "confidence")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    FRAMEWORK_FIELD_NUMBER: _ClassVar[int]
    ROOT_PATH_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_MANAGER_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    language: str
    framework: str
    root_path: str
    package_manager: str
    status: str
    confidence: float
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., language: _Optional[str] = ..., framework: _Optional[str] = ..., root_path: _Optional[str] = ..., package_manager: _Optional[str] = ..., status: _Optional[str] = ..., confidence: _Optional[float] = ...) -> None: ...

class QualityContract(_message.Message):
    __slots__ = ("id", "title", "category", "severity", "language", "framework", "surface_kind", "rule_ids", "description", "why_it_matters", "remediation", "autofix_available", "fix_class")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    FRAMEWORK_FIELD_NUMBER: _ClassVar[int]
    SURFACE_KIND_FIELD_NUMBER: _ClassVar[int]
    RULE_IDS_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    WHY_IT_MATTERS_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    AUTOFIX_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    FIX_CLASS_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    category: str
    severity: str
    language: str
    framework: str
    surface_kind: str
    rule_ids: _containers.RepeatedScalarFieldContainer[str]
    description: str
    why_it_matters: str
    remediation: str
    autofix_available: bool
    fix_class: str
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., category: _Optional[str] = ..., severity: _Optional[str] = ..., language: _Optional[str] = ..., framework: _Optional[str] = ..., surface_kind: _Optional[str] = ..., rule_ids: _Optional[_Iterable[str]] = ..., description: _Optional[str] = ..., why_it_matters: _Optional[str] = ..., remediation: _Optional[str] = ..., autofix_available: _Optional[bool] = ..., fix_class: _Optional[str] = ...) -> None: ...

class ContractEvaluation(_message.Message):
    __slots__ = ("contract_id", "surface_id", "status", "rule_ids")
    CONTRACT_ID_FIELD_NUMBER: _ClassVar[int]
    SURFACE_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    RULE_IDS_FIELD_NUMBER: _ClassVar[int]
    contract_id: str
    surface_id: str
    status: str
    rule_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, contract_id: _Optional[str] = ..., surface_id: _Optional[str] = ..., status: _Optional[str] = ..., rule_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class QualityFinding(_message.Message):
    __slots__ = ("id", "scenario", "target_kind", "surface_id", "surface_kind", "language", "framework", "rule_id", "category", "severity", "file_path", "symbol", "message", "evidence", "expected", "observed", "why_it_matters", "remediation", "autofix_available", "autofix_command", "source_command", "created_at", "fix_class")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TARGET_KIND_FIELD_NUMBER: _ClassVar[int]
    SURFACE_ID_FIELD_NUMBER: _ClassVar[int]
    SURFACE_KIND_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    FRAMEWORK_FIELD_NUMBER: _ClassVar[int]
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    SYMBOL_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_FIELD_NUMBER: _ClassVar[int]
    WHY_IT_MATTERS_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    AUTOFIX_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    AUTOFIX_COMMAND_FIELD_NUMBER: _ClassVar[int]
    SOURCE_COMMAND_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    FIX_CLASS_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario: str
    target_kind: str
    surface_id: str
    surface_kind: str
    language: str
    framework: str
    rule_id: str
    category: str
    severity: str
    file_path: str
    symbol: str
    message: str
    evidence: str
    expected: str
    observed: str
    why_it_matters: str
    remediation: str
    autofix_available: bool
    autofix_command: str
    source_command: str
    created_at: str
    fix_class: str
    def __init__(self, id: _Optional[str] = ..., scenario: _Optional[str] = ..., target_kind: _Optional[str] = ..., surface_id: _Optional[str] = ..., surface_kind: _Optional[str] = ..., language: _Optional[str] = ..., framework: _Optional[str] = ..., rule_id: _Optional[str] = ..., category: _Optional[str] = ..., severity: _Optional[str] = ..., file_path: _Optional[str] = ..., symbol: _Optional[str] = ..., message: _Optional[str] = ..., evidence: _Optional[str] = ..., expected: _Optional[str] = ..., observed: _Optional[str] = ..., why_it_matters: _Optional[str] = ..., remediation: _Optional[str] = ..., autofix_available: _Optional[bool] = ..., autofix_command: _Optional[str] = ..., source_command: _Optional[str] = ..., created_at: _Optional[str] = ..., fix_class: _Optional[str] = ...) -> None: ...

class CommandResult(_message.Message):
    __slots__ = ("name", "command", "working_directory", "status", "exit_code", "stdout_excerpt", "stderr_excerpt", "timeout_seconds", "failure_reason")
    NAME_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    WORKING_DIRECTORY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    STDOUT_EXCERPT_FIELD_NUMBER: _ClassVar[int]
    STDERR_EXCERPT_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    FAILURE_REASON_FIELD_NUMBER: _ClassVar[int]
    name: str
    command: str
    working_directory: str
    status: str
    exit_code: int
    stdout_excerpt: str
    stderr_excerpt: str
    timeout_seconds: int
    failure_reason: str
    def __init__(self, name: _Optional[str] = ..., command: _Optional[str] = ..., working_directory: _Optional[str] = ..., status: _Optional[str] = ..., exit_code: _Optional[int] = ..., stdout_excerpt: _Optional[str] = ..., stderr_excerpt: _Optional[str] = ..., timeout_seconds: _Optional[int] = ..., failure_reason: _Optional[str] = ...) -> None: ...

class MaturitySummary(_message.Message):
    __slots__ = ("rung", "label", "rationale")
    RUNG_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    rung: int
    label: str
    rationale: str
    def __init__(self, rung: _Optional[int] = ..., label: _Optional[str] = ..., rationale: _Optional[str] = ...) -> None: ...

class AuditSummary(_message.Message):
    __slots__ = ("errors", "warnings", "infos", "surfaces", "contracts", "autofixable_count")
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    INFOS_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    CONTRACTS_FIELD_NUMBER: _ClassVar[int]
    AUTOFIXABLE_COUNT_FIELD_NUMBER: _ClassVar[int]
    errors: int
    warnings: int
    infos: int
    surfaces: int
    contracts: int
    autofixable_count: int
    def __init__(self, errors: _Optional[int] = ..., warnings: _Optional[int] = ..., infos: _Optional[int] = ..., surfaces: _Optional[int] = ..., contracts: _Optional[int] = ..., autofixable_count: _Optional[int] = ...) -> None: ...

class AutofixCandidate(_message.Message):
    __slots__ = ("rule_id", "file_path", "description", "before", "after", "applied")
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    BEFORE_FIELD_NUMBER: _ClassVar[int]
    AFTER_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    rule_id: str
    file_path: str
    description: str
    before: str
    after: str
    applied: bool
    def __init__(self, rule_id: _Optional[str] = ..., file_path: _Optional[str] = ..., description: _Optional[str] = ..., before: _Optional[str] = ..., after: _Optional[str] = ..., applied: _Optional[bool] = ...) -> None: ...
