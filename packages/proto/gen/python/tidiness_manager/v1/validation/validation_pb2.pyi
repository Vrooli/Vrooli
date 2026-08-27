from common.v1 import maturity_pb2 as _maturity_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TidinessScanResponse(_message.Message):
    __slots__ = ("scenario", "status", "findings", "violations", "summary", "assessment")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    status: str
    findings: _containers.RepeatedCompositeFieldContainer[TidinessFinding]
    violations: _containers.RepeatedCompositeFieldContainer[TidinessFinding]
    summary: TidinessScanSummary
    assessment: _maturity_pb2.MaturityAssessment
    def __init__(self, scenario: _Optional[str] = ..., status: _Optional[str] = ..., findings: _Optional[_Iterable[_Union[TidinessFinding, _Mapping]]] = ..., violations: _Optional[_Iterable[_Union[TidinessFinding, _Mapping]]] = ..., summary: _Optional[_Union[TidinessScanSummary, _Mapping]] = ..., assessment: _Optional[_Union[_maturity_pb2.MaturityAssessment, _Mapping]] = ...) -> None: ...

class TidinessScanSummary(_message.Message):
    __slots__ = ("total_findings", "long_files", "complexity", "duplication", "tech_debt", "coupling", "duplication_line_debt")
    TOTAL_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    LONG_FILES_FIELD_NUMBER: _ClassVar[int]
    COMPLEXITY_FIELD_NUMBER: _ClassVar[int]
    DUPLICATION_FIELD_NUMBER: _ClassVar[int]
    TECH_DEBT_FIELD_NUMBER: _ClassVar[int]
    COUPLING_FIELD_NUMBER: _ClassVar[int]
    DUPLICATION_LINE_DEBT_FIELD_NUMBER: _ClassVar[int]
    total_findings: int
    long_files: int
    complexity: int
    duplication: int
    tech_debt: int
    coupling: int
    duplication_line_debt: int
    def __init__(self, total_findings: _Optional[int] = ..., long_files: _Optional[int] = ..., complexity: _Optional[int] = ..., duplication: _Optional[int] = ..., tech_debt: _Optional[int] = ..., coupling: _Optional[int] = ..., duplication_line_debt: _Optional[int] = ...) -> None: ...

class TidinessFinding(_message.Message):
    __slots__ = ("id", "rule_id", "scenario", "file_path", "symbol", "line_number", "category", "severity", "title", "description", "evidence", "why_it_matters", "recommended_remediation", "remediation", "campaign_group_hint")
    ID_FIELD_NUMBER: _ClassVar[int]
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    SYMBOL_FIELD_NUMBER: _ClassVar[int]
    LINE_NUMBER_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    WHY_IT_MATTERS_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    CAMPAIGN_GROUP_HINT_FIELD_NUMBER: _ClassVar[int]
    id: str
    rule_id: str
    scenario: str
    file_path: str
    symbol: str
    line_number: int
    category: str
    severity: str
    title: str
    description: str
    evidence: _struct_pb2.Struct
    why_it_matters: str
    recommended_remediation: str
    remediation: str
    campaign_group_hint: str
    def __init__(self, id: _Optional[str] = ..., rule_id: _Optional[str] = ..., scenario: _Optional[str] = ..., file_path: _Optional[str] = ..., symbol: _Optional[str] = ..., line_number: _Optional[int] = ..., category: _Optional[str] = ..., severity: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., evidence: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., why_it_matters: _Optional[str] = ..., recommended_remediation: _Optional[str] = ..., remediation: _Optional[str] = ..., campaign_group_hint: _Optional[str] = ...) -> None: ...
