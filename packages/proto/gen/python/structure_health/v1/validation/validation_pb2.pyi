from common.v1 import maturity_pb2 as _maturity_pb2
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
    __slots__ = ("run_id", "status", "summary", "scenario", "target_path", "degraded_reason", "profile", "surfaces", "findings", "assessment", "next_steps")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TARGET_PATH_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASON_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    SURFACES_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    NEXT_STEPS_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    status: str
    summary: str
    scenario: str
    target_path: str
    degraded_reason: str
    profile: DetectedProfile
    surfaces: _containers.RepeatedCompositeFieldContainer[SurfaceReconcile]
    findings: _containers.RepeatedCompositeFieldContainer[StructureFinding]
    assessment: _maturity_pb2.MaturityAssessment
    next_steps: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, run_id: _Optional[str] = ..., status: _Optional[str] = ..., summary: _Optional[str] = ..., scenario: _Optional[str] = ..., target_path: _Optional[str] = ..., degraded_reason: _Optional[str] = ..., profile: _Optional[_Union[DetectedProfile, _Mapping]] = ..., surfaces: _Optional[_Iterable[_Union[SurfaceReconcile, _Mapping]]] = ..., findings: _Optional[_Iterable[_Union[StructureFinding, _Mapping]]] = ..., assessment: _Optional[_Union[_maturity_pb2.MaturityAssessment, _Mapping]] = ..., next_steps: _Optional[_Iterable[str]] = ...) -> None: ...

class DetectedProfile(_message.Message):
    __slots__ = ("id", "backend_language", "ui_framework", "recognized", "evidence")
    ID_FIELD_NUMBER: _ClassVar[int]
    BACKEND_LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    UI_FRAMEWORK_FIELD_NUMBER: _ClassVar[int]
    RECOGNIZED_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    backend_language: str
    ui_framework: str
    recognized: bool
    evidence: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., backend_language: _Optional[str] = ..., ui_framework: _Optional[str] = ..., recognized: _Optional[bool] = ..., evidence: _Optional[_Iterable[str]] = ...) -> None: ...

class SurfaceReconcile(_message.Message):
    __slots__ = ("surface", "kind", "declared", "actual", "declared_detail", "actual_detail")
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    DECLARED_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_FIELD_NUMBER: _ClassVar[int]
    DECLARED_DETAIL_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_DETAIL_FIELD_NUMBER: _ClassVar[int]
    surface: str
    kind: str
    declared: bool
    actual: bool
    declared_detail: str
    actual_detail: str
    def __init__(self, surface: _Optional[str] = ..., kind: _Optional[str] = ..., declared: _Optional[bool] = ..., actual: _Optional[bool] = ..., declared_detail: _Optional[str] = ..., actual_detail: _Optional[str] = ...) -> None: ...

class StructureFinding(_message.Message):
    __slots__ = ("code", "severity", "title", "message", "location", "remediation", "surface", "autofix_available", "fix_class")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    AUTOFIX_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    FIX_CLASS_FIELD_NUMBER: _ClassVar[int]
    code: str
    severity: str
    title: str
    message: str
    location: str
    remediation: str
    surface: str
    autofix_available: bool
    fix_class: str
    def __init__(self, code: _Optional[str] = ..., severity: _Optional[str] = ..., title: _Optional[str] = ..., message: _Optional[str] = ..., location: _Optional[str] = ..., remediation: _Optional[str] = ..., surface: _Optional[str] = ..., autofix_available: _Optional[bool] = ..., fix_class: _Optional[str] = ...) -> None: ...

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
