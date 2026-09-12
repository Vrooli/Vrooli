from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ConformanceFinding(_message.Message):
    __slots__ = ("rule_id", "severity", "path", "message", "remediation")
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    REMEDIATION_FIELD_NUMBER: _ClassVar[int]
    rule_id: str
    severity: str
    path: str
    message: str
    remediation: str
    def __init__(self, rule_id: _Optional[str] = ..., severity: _Optional[str] = ..., path: _Optional[str] = ..., message: _Optional[str] = ..., remediation: _Optional[str] = ...) -> None: ...

class ScanScenarioRequest(_message.Message):
    __slots__ = ("scenario", "path")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    path: str
    def __init__(self, scenario: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class ScanScenarioResponse(_message.Message):
    __slots__ = ("scenario", "maturity_level", "findings", "recommendations")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    MATURITY_LEVEL_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATIONS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    maturity_level: str
    findings: _containers.RepeatedCompositeFieldContainer[ConformanceFinding]
    recommendations: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., maturity_level: _Optional[str] = ..., findings: _Optional[_Iterable[_Union[ConformanceFinding, _Mapping]]] = ..., recommendations: _Optional[_Iterable[str]] = ...) -> None: ...
