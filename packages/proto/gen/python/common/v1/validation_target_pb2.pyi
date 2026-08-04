from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ValidationTargetKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VALIDATION_TARGET_KIND_UNSPECIFIED: _ClassVar[ValidationTargetKind]
    VALIDATION_TARGET_KIND_SCENARIO: _ClassVar[ValidationTargetKind]
    VALIDATION_TARGET_KIND_RESOURCE: _ClassVar[ValidationTargetKind]
    VALIDATION_TARGET_KIND_TOOL: _ClassVar[ValidationTargetKind]
    VALIDATION_TARGET_KIND_SAFEGUARD: _ClassVar[ValidationTargetKind]
    VALIDATION_TARGET_KIND_TEAM: _ClassVar[ValidationTargetKind]
    VALIDATION_TARGET_KIND_PACKAGE: _ClassVar[ValidationTargetKind]
    VALIDATION_TARGET_KIND_CONTROL_PLANE: _ClassVar[ValidationTargetKind]
    VALIDATION_TARGET_KIND_DOCS: _ClassVar[ValidationTargetKind]
VALIDATION_TARGET_KIND_UNSPECIFIED: ValidationTargetKind
VALIDATION_TARGET_KIND_SCENARIO: ValidationTargetKind
VALIDATION_TARGET_KIND_RESOURCE: ValidationTargetKind
VALIDATION_TARGET_KIND_TOOL: ValidationTargetKind
VALIDATION_TARGET_KIND_SAFEGUARD: ValidationTargetKind
VALIDATION_TARGET_KIND_TEAM: ValidationTargetKind
VALIDATION_TARGET_KIND_PACKAGE: ValidationTargetKind
VALIDATION_TARGET_KIND_CONTROL_PLANE: ValidationTargetKind
VALIDATION_TARGET_KIND_DOCS: ValidationTargetKind

class ValidationTarget(_message.Message):
    __slots__ = ("kind", "id", "root")
    KIND_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    ROOT_FIELD_NUMBER: _ClassVar[int]
    kind: ValidationTargetKind
    id: str
    root: str
    def __init__(self, kind: _Optional[_Union[ValidationTargetKind, str]] = ..., id: _Optional[str] = ..., root: _Optional[str] = ...) -> None: ...
