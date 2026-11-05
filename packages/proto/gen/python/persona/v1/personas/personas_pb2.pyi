import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PersonaKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PERSONA_KIND_UNSPECIFIED: _ClassVar[PersonaKind]
    PERSONA_KIND_PERSONAL: _ClassVar[PersonaKind]
    PERSONA_KIND_BUSINESS: _ClassVar[PersonaKind]

class PersonaStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PERSONA_STATUS_UNSPECIFIED: _ClassVar[PersonaStatus]
    PERSONA_STATUS_ACTIVE: _ClassVar[PersonaStatus]
    PERSONA_STATUS_ARCHIVED: _ClassVar[PersonaStatus]
PERSONA_KIND_UNSPECIFIED: PersonaKind
PERSONA_KIND_PERSONAL: PersonaKind
PERSONA_KIND_BUSINESS: PersonaKind
PERSONA_STATUS_UNSPECIFIED: PersonaStatus
PERSONA_STATUS_ACTIVE: PersonaStatus
PERSONA_STATUS_ARCHIVED: PersonaStatus

class LegalBasis(_message.Message):
    __slots__ = ("subject_id", "subject_name", "basis_type")
    SUBJECT_ID_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_NAME_FIELD_NUMBER: _ClassVar[int]
    BASIS_TYPE_FIELD_NUMBER: _ClassVar[int]
    subject_id: str
    subject_name: str
    basis_type: str
    def __init__(self, subject_id: _Optional[str] = ..., subject_name: _Optional[str] = ..., basis_type: _Optional[str] = ...) -> None: ...

class Identifier(_message.Message):
    __slots__ = ("type", "value")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    type: str
    value: str
    def __init__(self, type: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...

class Persona(_message.Message):
    __slots__ = ("id", "kind", "legal_basis", "display_name", "identifiers", "status", "created_at", "archived_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    LEGAL_BASIS_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    IDENTIFIERS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: PersonaKind
    legal_basis: LegalBasis
    display_name: str
    identifiers: _containers.RepeatedCompositeFieldContainer[Identifier]
    status: PersonaStatus
    created_at: _timestamp_pb2.Timestamp
    archived_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[_Union[PersonaKind, str]] = ..., legal_basis: _Optional[_Union[LegalBasis, _Mapping]] = ..., display_name: _Optional[str] = ..., identifiers: _Optional[_Iterable[_Union[Identifier, _Mapping]]] = ..., status: _Optional[_Union[PersonaStatus, str]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., archived_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CreatePersonaRequest(_message.Message):
    __slots__ = ("kind", "legal_basis", "display_name", "identifiers")
    KIND_FIELD_NUMBER: _ClassVar[int]
    LEGAL_BASIS_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    IDENTIFIERS_FIELD_NUMBER: _ClassVar[int]
    kind: PersonaKind
    legal_basis: LegalBasis
    display_name: str
    identifiers: _containers.RepeatedCompositeFieldContainer[Identifier]
    def __init__(self, kind: _Optional[_Union[PersonaKind, str]] = ..., legal_basis: _Optional[_Union[LegalBasis, _Mapping]] = ..., display_name: _Optional[str] = ..., identifiers: _Optional[_Iterable[_Union[Identifier, _Mapping]]] = ...) -> None: ...

class CreatePersonaResponse(_message.Message):
    __slots__ = ("persona",)
    PERSONA_FIELD_NUMBER: _ClassVar[int]
    persona: Persona
    def __init__(self, persona: _Optional[_Union[Persona, _Mapping]] = ...) -> None: ...

class GetPersonaRequest(_message.Message):
    __slots__ = ("persona_id",)
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    def __init__(self, persona_id: _Optional[str] = ...) -> None: ...

class GetPersonaResponse(_message.Message):
    __slots__ = ("persona",)
    PERSONA_FIELD_NUMBER: _ClassVar[int]
    persona: Persona
    def __init__(self, persona: _Optional[_Union[Persona, _Mapping]] = ...) -> None: ...

class ListPersonasRequest(_message.Message):
    __slots__ = ("limit", "include_archived")
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_ARCHIVED_FIELD_NUMBER: _ClassVar[int]
    limit: int
    include_archived: bool
    def __init__(self, limit: _Optional[int] = ..., include_archived: _Optional[bool] = ...) -> None: ...

class ListPersonasResponse(_message.Message):
    __slots__ = ("personas",)
    PERSONAS_FIELD_NUMBER: _ClassVar[int]
    personas: _containers.RepeatedCompositeFieldContainer[Persona]
    def __init__(self, personas: _Optional[_Iterable[_Union[Persona, _Mapping]]] = ...) -> None: ...

class ArchivePersonaRequest(_message.Message):
    __slots__ = ("persona_id",)
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    def __init__(self, persona_id: _Optional[str] = ...) -> None: ...

class ArchivePersonaResponse(_message.Message):
    __slots__ = ("persona",)
    PERSONA_FIELD_NUMBER: _ClassVar[int]
    persona: Persona
    def __init__(self, persona: _Optional[_Union[Persona, _Mapping]] = ...) -> None: ...

class CheckHealthRequest(_message.Message):
    __slots__ = ("persona_id",)
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    def __init__(self, persona_id: _Optional[str] = ...) -> None: ...

class HealthFinding(_message.Message):
    __slots__ = ("code", "message", "blocking")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    blocking: bool
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., blocking: _Optional[bool] = ...) -> None: ...

class CheckHealthResponse(_message.Message):
    __slots__ = ("findings",)
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    findings: _containers.RepeatedCompositeFieldContainer[HealthFinding]
    def __init__(self, findings: _Optional[_Iterable[_Union[HealthFinding, _Mapping]]] = ...) -> None: ...
