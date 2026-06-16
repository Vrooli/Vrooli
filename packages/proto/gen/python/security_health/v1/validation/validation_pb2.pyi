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
SEVERITY_UNSPECIFIED: Severity
SEVERITY_ERROR: Severity
SEVERITY_WARNING: Severity
SEVERITY_INFO: Severity

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

class ValidateScenarioRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class ValidateScenarioResponse(_message.Message):
    __slots__ = ("scenario", "passed", "findings", "summary", "skipped_scanners", "assessment")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_SCANNERS_FIELD_NUMBER: _ClassVar[int]
    ASSESSMENT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    passed: bool
    findings: _containers.RepeatedCompositeFieldContainer[Finding]
    summary: Summary
    skipped_scanners: _containers.RepeatedScalarFieldContainer[str]
    assessment: _maturity_pb2.MaturityAssessment
    def __init__(self, scenario: _Optional[str] = ..., passed: _Optional[bool] = ..., findings: _Optional[_Iterable[_Union[Finding, _Mapping]]] = ..., summary: _Optional[_Union[Summary, _Mapping]] = ..., skipped_scanners: _Optional[_Iterable[str]] = ..., assessment: _Optional[_Union[_maturity_pb2.MaturityAssessment, _Mapping]] = ...) -> None: ...
