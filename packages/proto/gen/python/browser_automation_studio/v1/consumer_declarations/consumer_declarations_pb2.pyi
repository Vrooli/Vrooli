from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ValidateConsumerDeclarationRequest(_message.Message):
    __slots__ = ("declaration_json",)
    DECLARATION_JSON_FIELD_NUMBER: _ClassVar[int]
    declaration_json: str
    def __init__(self, declaration_json: _Optional[str] = ...) -> None: ...

class ValidateConsumerDeclarationResponse(_message.Message):
    __slots__ = ("valid", "issues", "profiles")
    VALID_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    PROFILES_FIELD_NUMBER: _ClassVar[int]
    valid: bool
    issues: _containers.RepeatedScalarFieldContainer[str]
    profiles: _containers.RepeatedCompositeFieldContainer[DeclaredProfile]
    def __init__(self, valid: _Optional[bool] = ..., issues: _Optional[_Iterable[str]] = ..., profiles: _Optional[_Iterable[_Union[DeclaredProfile, _Mapping]]] = ...) -> None: ...

class DeclaredProfile(_message.Message):
    __slots__ = ("key", "workflow_ref", "allowed_variables")
    KEY_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_REF_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_VARIABLES_FIELD_NUMBER: _ClassVar[int]
    key: str
    workflow_ref: str
    allowed_variables: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, key: _Optional[str] = ..., workflow_ref: _Optional[str] = ..., allowed_variables: _Optional[_Iterable[str]] = ...) -> None: ...
