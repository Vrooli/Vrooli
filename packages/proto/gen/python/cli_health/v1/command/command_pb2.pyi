from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CommandReferencePolicy(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COMMAND_REFERENCE_POLICY_UNSPECIFIED: _ClassVar[CommandReferencePolicy]
    COMMAND_REFERENCE_POLICY_DOCS: _ClassVar[CommandReferencePolicy]
    COMMAND_REFERENCE_POLICY_SKILL: _ClassVar[CommandReferencePolicy]
    COMMAND_REFERENCE_POLICY_PLAN: _ClassVar[CommandReferencePolicy]
    COMMAND_REFERENCE_POLICY_ACTION: _ClassVar[CommandReferencePolicy]

class CommandReferenceVerdict(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COMMAND_REFERENCE_VERDICT_UNSPECIFIED: _ClassVar[CommandReferenceVerdict]
    COMMAND_REFERENCE_VERDICT_VALID: _ClassVar[CommandReferenceVerdict]
    COMMAND_REFERENCE_VERDICT_INVALID: _ClassVar[CommandReferenceVerdict]
    COMMAND_REFERENCE_VERDICT_PARTIAL: _ClassVar[CommandReferenceVerdict]
    COMMAND_REFERENCE_VERDICT_SKIPPED: _ClassVar[CommandReferenceVerdict]
    COMMAND_REFERENCE_VERDICT_UNKNOWN: _ClassVar[CommandReferenceVerdict]
    COMMAND_REFERENCE_VERDICT_UNSUPPORTED: _ClassVar[CommandReferenceVerdict]

class CommandReferenceValidationLevel(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COMMAND_REFERENCE_VALIDATION_LEVEL_UNSPECIFIED: _ClassVar[CommandReferenceValidationLevel]
    COMMAND_REFERENCE_VALIDATION_LEVEL_PARSED: _ClassVar[CommandReferenceValidationLevel]
    COMMAND_REFERENCE_VALIDATION_LEVEL_OWNER_IDENTIFIED: _ClassVar[CommandReferenceValidationLevel]
    COMMAND_REFERENCE_VALIDATION_LEVEL_COMMAND_EXISTS: _ClassVar[CommandReferenceValidationLevel]
    COMMAND_REFERENCE_VALIDATION_LEVEL_ARGUMENT_SHAPE_VALIDATED: _ClassVar[CommandReferenceValidationLevel]
    COMMAND_REFERENCE_VALIDATION_LEVEL_SKIPPED_BY_QUALIFIER: _ClassVar[CommandReferenceValidationLevel]
    COMMAND_REFERENCE_VALIDATION_LEVEL_UNSUPPORTED_SYNTAX: _ClassVar[CommandReferenceValidationLevel]

class CommandReferenceRefreshPolicy(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COMMAND_REFERENCE_REFRESH_POLICY_UNSPECIFIED: _ClassVar[CommandReferenceRefreshPolicy]
    COMMAND_REFERENCE_REFRESH_POLICY_NEVER: _ClassVar[CommandReferenceRefreshPolicy]
    COMMAND_REFERENCE_REFRESH_POLICY_ON_MISS: _ClassVar[CommandReferenceRefreshPolicy]
COMMAND_REFERENCE_POLICY_UNSPECIFIED: CommandReferencePolicy
COMMAND_REFERENCE_POLICY_DOCS: CommandReferencePolicy
COMMAND_REFERENCE_POLICY_SKILL: CommandReferencePolicy
COMMAND_REFERENCE_POLICY_PLAN: CommandReferencePolicy
COMMAND_REFERENCE_POLICY_ACTION: CommandReferencePolicy
COMMAND_REFERENCE_VERDICT_UNSPECIFIED: CommandReferenceVerdict
COMMAND_REFERENCE_VERDICT_VALID: CommandReferenceVerdict
COMMAND_REFERENCE_VERDICT_INVALID: CommandReferenceVerdict
COMMAND_REFERENCE_VERDICT_PARTIAL: CommandReferenceVerdict
COMMAND_REFERENCE_VERDICT_SKIPPED: CommandReferenceVerdict
COMMAND_REFERENCE_VERDICT_UNKNOWN: CommandReferenceVerdict
COMMAND_REFERENCE_VERDICT_UNSUPPORTED: CommandReferenceVerdict
COMMAND_REFERENCE_VALIDATION_LEVEL_UNSPECIFIED: CommandReferenceValidationLevel
COMMAND_REFERENCE_VALIDATION_LEVEL_PARSED: CommandReferenceValidationLevel
COMMAND_REFERENCE_VALIDATION_LEVEL_OWNER_IDENTIFIED: CommandReferenceValidationLevel
COMMAND_REFERENCE_VALIDATION_LEVEL_COMMAND_EXISTS: CommandReferenceValidationLevel
COMMAND_REFERENCE_VALIDATION_LEVEL_ARGUMENT_SHAPE_VALIDATED: CommandReferenceValidationLevel
COMMAND_REFERENCE_VALIDATION_LEVEL_SKIPPED_BY_QUALIFIER: CommandReferenceValidationLevel
COMMAND_REFERENCE_VALIDATION_LEVEL_UNSUPPORTED_SYNTAX: CommandReferenceValidationLevel
COMMAND_REFERENCE_REFRESH_POLICY_UNSPECIFIED: CommandReferenceRefreshPolicy
COMMAND_REFERENCE_REFRESH_POLICY_NEVER: CommandReferenceRefreshPolicy
COMMAND_REFERENCE_REFRESH_POLICY_ON_MISS: CommandReferenceRefreshPolicy

class ValidateCommandReferenceRequest(_message.Message):
    __slots__ = ("command_text", "policy", "qualifiers", "refresh_policy")
    COMMAND_TEXT_FIELD_NUMBER: _ClassVar[int]
    POLICY_FIELD_NUMBER: _ClassVar[int]
    QUALIFIERS_FIELD_NUMBER: _ClassVar[int]
    REFRESH_POLICY_FIELD_NUMBER: _ClassVar[int]
    command_text: str
    policy: CommandReferencePolicy
    qualifiers: _containers.RepeatedScalarFieldContainer[str]
    refresh_policy: CommandReferenceRefreshPolicy
    def __init__(self, command_text: _Optional[str] = ..., policy: _Optional[_Union[CommandReferencePolicy, str]] = ..., qualifiers: _Optional[_Iterable[str]] = ..., refresh_policy: _Optional[_Union[CommandReferenceRefreshPolicy, str]] = ...) -> None: ...

class ValidateCommandReferenceResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: CommandReferenceValidationResult
    def __init__(self, result: _Optional[_Union[CommandReferenceValidationResult, _Mapping]] = ...) -> None: ...

class ValidateCommandReferencesRequest(_message.Message):
    __slots__ = ("requests",)
    REQUESTS_FIELD_NUMBER: _ClassVar[int]
    requests: _containers.RepeatedCompositeFieldContainer[ValidateCommandReferenceRequest]
    def __init__(self, requests: _Optional[_Iterable[_Union[ValidateCommandReferenceRequest, _Mapping]]] = ...) -> None: ...

class ValidateCommandReferencesResponse(_message.Message):
    __slots__ = ("results",)
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[CommandReferenceValidationResult]
    def __init__(self, results: _Optional[_Iterable[_Union[CommandReferenceValidationResult, _Mapping]]] = ...) -> None: ...

class CommandReferenceValidationResult(_message.Message):
    __slots__ = ("command_text", "verdict", "validation_level", "canonical_command", "owner", "source", "issues", "suggestions", "guidance")
    COMMAND_TEXT_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_LEVEL_FIELD_NUMBER: _ClassVar[int]
    CANONICAL_COMMAND_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    SUGGESTIONS_FIELD_NUMBER: _ClassVar[int]
    GUIDANCE_FIELD_NUMBER: _ClassVar[int]
    command_text: str
    verdict: CommandReferenceVerdict
    validation_level: CommandReferenceValidationLevel
    canonical_command: str
    owner: str
    source: str
    issues: _containers.RepeatedCompositeFieldContainer[CommandReferenceIssue]
    suggestions: _containers.RepeatedCompositeFieldContainer[CommandReferenceSuggestion]
    guidance: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, command_text: _Optional[str] = ..., verdict: _Optional[_Union[CommandReferenceVerdict, str]] = ..., validation_level: _Optional[_Union[CommandReferenceValidationLevel, str]] = ..., canonical_command: _Optional[str] = ..., owner: _Optional[str] = ..., source: _Optional[str] = ..., issues: _Optional[_Iterable[_Union[CommandReferenceIssue, _Mapping]]] = ..., suggestions: _Optional[_Iterable[_Union[CommandReferenceSuggestion, _Mapping]]] = ..., guidance: _Optional[_Iterable[str]] = ...) -> None: ...

class CommandReferenceIssue(_message.Message):
    __slots__ = ("code", "message", "severity", "fix")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    FIX_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    severity: str
    fix: str
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., severity: _Optional[str] = ..., fix: _Optional[str] = ...) -> None: ...

class CommandReferenceSuggestion(_message.Message):
    __slots__ = ("command", "reason")
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    command: str
    reason: str
    def __init__(self, command: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...
