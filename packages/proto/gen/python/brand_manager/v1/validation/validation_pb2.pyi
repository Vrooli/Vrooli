from common.v1 import maturity_pb2 as _maturity_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class BrandingScanResponse(_message.Message):
    __slots__ = ("scenario", "status", "findings", "summary", "assessment")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    status: str
    findings: _containers.RepeatedCompositeFieldContainer[BrandingFinding]
    summary: BrandingScanSummary
    assessment: _maturity_pb2.MaturityAssessment
    def __init__(self, scenario: _Optional[str] = ..., status: _Optional[str] = ..., findings: _Optional[_Iterable[_Union[BrandingFinding, _Mapping]]] = ..., summary: _Optional[_Union[BrandingScanSummary, _Mapping]] = ..., assessment: _Optional[_Union[_maturity_pb2.MaturityAssessment, _Mapping]] = ...) -> None: ...

class BrandingScanSummary(_message.Message):
    __slots__ = ("total_findings", "autofixable", "errors", "warnings", "infos")
    TOTAL_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    AUTOFIXABLE_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    INFOS_FIELD_NUMBER: _ClassVar[int]
    total_findings: int
    autofixable: int
    errors: int
    warnings: int
    infos: int
    def __init__(self, total_findings: _Optional[int] = ..., autofixable: _Optional[int] = ..., errors: _Optional[int] = ..., warnings: _Optional[int] = ..., infos: _Optional[int] = ...) -> None: ...

class BrandingFinding(_message.Message):
    __slots__ = ("id", "rule_id", "scenario", "file_path", "severity", "title", "description", "evidence", "why_it_matters", "recommended_remediation", "autofix_available")
    ID_FIELD_NUMBER: _ClassVar[int]
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    WHY_IT_MATTERS_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    AUTOFIX_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    id: str
    rule_id: str
    scenario: str
    file_path: str
    severity: str
    title: str
    description: str
    evidence: _struct_pb2.Struct
    why_it_matters: str
    recommended_remediation: str
    autofix_available: bool
    def __init__(self, id: _Optional[str] = ..., rule_id: _Optional[str] = ..., scenario: _Optional[str] = ..., file_path: _Optional[str] = ..., severity: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., evidence: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., why_it_matters: _Optional[str] = ..., recommended_remediation: _Optional[str] = ..., autofix_available: _Optional[bool] = ...) -> None: ...
