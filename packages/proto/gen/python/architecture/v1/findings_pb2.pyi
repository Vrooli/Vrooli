from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class FindingSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FINDING_SOURCE_UNSPECIFIED: _ClassVar[FindingSource]
    FINDING_SOURCE_STRUCTURE: _ClassVar[FindingSource]
    FINDING_SOURCE_CLI: _ClassVar[FindingSource]
    FINDING_SOURCE_UI: _ClassVar[FindingSource]
    FINDING_SOURCE_DOCS: _ClassVar[FindingSource]
    FINDING_SOURCE_STANDARDS: _ClassVar[FindingSource]
    FINDING_SOURCE_ARCHITECTURE: _ClassVar[FindingSource]
    FINDING_SOURCE_TIDINESS: _ClassVar[FindingSource]

class FindingSeverity(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FINDING_SEVERITY_UNSPECIFIED: _ClassVar[FindingSeverity]
    FINDING_SEVERITY_INFO: _ClassVar[FindingSeverity]
    FINDING_SEVERITY_WARNING: _ClassVar[FindingSeverity]
    FINDING_SEVERITY_ERROR: _ClassVar[FindingSeverity]
    FINDING_SEVERITY_BLOCKER: _ClassVar[FindingSeverity]

class EffortHint(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EFFORT_HINT_UNSPECIFIED: _ClassVar[EffortHint]
    EFFORT_HINT_TRIVIAL: _ClassVar[EffortHint]
    EFFORT_HINT_SMALL: _ClassVar[EffortHint]
    EFFORT_HINT_MEDIUM: _ClassVar[EffortHint]
    EFFORT_HINT_LARGE: _ClassVar[EffortHint]
FINDING_SOURCE_UNSPECIFIED: FindingSource
FINDING_SOURCE_STRUCTURE: FindingSource
FINDING_SOURCE_CLI: FindingSource
FINDING_SOURCE_UI: FindingSource
FINDING_SOURCE_DOCS: FindingSource
FINDING_SOURCE_STANDARDS: FindingSource
FINDING_SOURCE_ARCHITECTURE: FindingSource
FINDING_SOURCE_TIDINESS: FindingSource
FINDING_SEVERITY_UNSPECIFIED: FindingSeverity
FINDING_SEVERITY_INFO: FindingSeverity
FINDING_SEVERITY_WARNING: FindingSeverity
FINDING_SEVERITY_ERROR: FindingSeverity
FINDING_SEVERITY_BLOCKER: FindingSeverity
EFFORT_HINT_UNSPECIFIED: EffortHint
EFFORT_HINT_TRIVIAL: EffortHint
EFFORT_HINT_SMALL: EffortHint
EFFORT_HINT_MEDIUM: EffortHint
EFFORT_HINT_LARGE: EffortHint

class Evidence(_message.Message):
    __slots__ = ("kind", "summary", "locator", "payload")
    KIND_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    LOCATOR_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    kind: str
    summary: str
    locator: str
    payload: bytes
    def __init__(self, kind: _Optional[str] = ..., summary: _Optional[str] = ..., locator: _Optional[str] = ..., payload: _Optional[bytes] = ...) -> None: ...

class SuggestedFix(_message.Message):
    __slots__ = ("id", "kind", "resolver", "summary", "payload", "confidence")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    RESOLVER_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    resolver: str
    summary: str
    payload: bytes
    confidence: float
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., resolver: _Optional[str] = ..., summary: _Optional[str] = ..., payload: _Optional[bytes] = ..., confidence: _Optional[float] = ...) -> None: ...

class ArchitectureFinding(_message.Message):
    __slots__ = ("scenario", "source", "code", "severity", "locations", "domains", "message", "suggestion", "stable_id", "evidence", "suggested_fixes", "effort")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    LOCATIONS_FIELD_NUMBER: _ClassVar[int]
    DOMAINS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SUGGESTION_FIELD_NUMBER: _ClassVar[int]
    STABLE_ID_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    SUGGESTED_FIXES_FIELD_NUMBER: _ClassVar[int]
    EFFORT_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    source: FindingSource
    code: str
    severity: FindingSeverity
    locations: _containers.RepeatedScalarFieldContainer[str]
    domains: _containers.RepeatedScalarFieldContainer[str]
    message: str
    suggestion: str
    stable_id: str
    evidence: _containers.RepeatedCompositeFieldContainer[Evidence]
    suggested_fixes: _containers.RepeatedCompositeFieldContainer[SuggestedFix]
    effort: EffortHint
    def __init__(self, scenario: _Optional[str] = ..., source: _Optional[_Union[FindingSource, str]] = ..., code: _Optional[str] = ..., severity: _Optional[_Union[FindingSeverity, str]] = ..., locations: _Optional[_Iterable[str]] = ..., domains: _Optional[_Iterable[str]] = ..., message: _Optional[str] = ..., suggestion: _Optional[str] = ..., stable_id: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[Evidence, _Mapping]]] = ..., suggested_fixes: _Optional[_Iterable[_Union[SuggestedFix, _Mapping]]] = ..., effort: _Optional[_Union[EffortHint, str]] = ...) -> None: ...
