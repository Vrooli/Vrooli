from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Basis(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    BASIS_UNSPECIFIED: _ClassVar[Basis]
    BASIS_DERIVED: _ClassVar[Basis]
    BASIS_VALIDATED: _ClassVar[Basis]
    BASIS_DECLARED_UNVERIFIED: _ClassVar[Basis]
    BASIS_CONTRADICTED: _ClassVar[Basis]
    BASIS_ABSENT: _ClassVar[Basis]

class Sufficiency(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SUFFICIENCY_UNSPECIFIED: _ClassVar[Sufficiency]
    SUFFICIENCY_FULL: _ClassVar[Sufficiency]
    SUFFICIENCY_PARTIAL: _ClassVar[Sufficiency]
    SUFFICIENCY_INSUFFICIENT: _ClassVar[Sufficiency]
BASIS_UNSPECIFIED: Basis
BASIS_DERIVED: Basis
BASIS_VALIDATED: Basis
BASIS_DECLARED_UNVERIFIED: Basis
BASIS_CONTRADICTED: Basis
BASIS_ABSENT: Basis
SUFFICIENCY_UNSPECIFIED: Sufficiency
SUFFICIENCY_FULL: Sufficiency
SUFFICIENCY_PARTIAL: Sufficiency
SUFFICIENCY_INSUFFICIENT: Sufficiency

class AttestedAnswer(_message.Message):
    __slots__ = ("claim", "citations", "basis", "sufficiency", "gaps", "suggested_follow_ups")
    CLAIM_FIELD_NUMBER: _ClassVar[int]
    CITATIONS_FIELD_NUMBER: _ClassVar[int]
    BASIS_FIELD_NUMBER: _ClassVar[int]
    SUFFICIENCY_FIELD_NUMBER: _ClassVar[int]
    GAPS_FIELD_NUMBER: _ClassVar[int]
    SUGGESTED_FOLLOW_UPS_FIELD_NUMBER: _ClassVar[int]
    claim: str
    citations: _containers.RepeatedCompositeFieldContainer[Citation]
    basis: Basis
    sufficiency: Sufficiency
    gaps: _containers.RepeatedScalarFieldContainer[str]
    suggested_follow_ups: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, claim: _Optional[str] = ..., citations: _Optional[_Iterable[_Union[Citation, _Mapping]]] = ..., basis: _Optional[_Union[Basis, str]] = ..., sufficiency: _Optional[_Union[Sufficiency, str]] = ..., gaps: _Optional[_Iterable[str]] = ..., suggested_follow_ups: _Optional[_Iterable[str]] = ...) -> None: ...

class Citation(_message.Message):
    __slots__ = ("locator", "kind", "note")
    LOCATOR_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    locator: str
    kind: str
    note: str
    def __init__(self, locator: _Optional[str] = ..., kind: _Optional[str] = ..., note: _Optional[str] = ...) -> None: ...
