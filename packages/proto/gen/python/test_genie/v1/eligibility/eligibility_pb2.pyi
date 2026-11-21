from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CheckRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class Violation(_message.Message):
    __slots__ = ("rule_id", "severity", "file", "line", "excerpt")
    RULE_ID_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    FILE_FIELD_NUMBER: _ClassVar[int]
    LINE_FIELD_NUMBER: _ClassVar[int]
    EXCERPT_FIELD_NUMBER: _ClassVar[int]
    rule_id: str
    severity: str
    file: str
    line: int
    excerpt: str
    def __init__(self, rule_id: _Optional[str] = ..., severity: _Optional[str] = ..., file: _Optional[str] = ..., line: _Optional[int] = ..., excerpt: _Optional[str] = ...) -> None: ...

class RuleAssertion(_message.Message):
    __slots__ = ("missing_rules",)
    MISSING_RULES_FIELD_NUMBER: _ClassVar[int]
    missing_rules: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, missing_rules: _Optional[_Iterable[str]] = ...) -> None: ...

class CheckResponse(_message.Message):
    __slots__ = ("routed", "violations", "rule_assertion", "disqualifying_reasons")
    ROUTED_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    RULE_ASSERTION_FIELD_NUMBER: _ClassVar[int]
    DISQUALIFYING_REASONS_FIELD_NUMBER: _ClassVar[int]
    routed: bool
    violations: _containers.RepeatedCompositeFieldContainer[Violation]
    rule_assertion: RuleAssertion
    disqualifying_reasons: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, routed: _Optional[bool] = ..., violations: _Optional[_Iterable[_Union[Violation, _Mapping]]] = ..., rule_assertion: _Optional[_Union[RuleAssertion, _Mapping]] = ..., disqualifying_reasons: _Optional[_Iterable[str]] = ...) -> None: ...
