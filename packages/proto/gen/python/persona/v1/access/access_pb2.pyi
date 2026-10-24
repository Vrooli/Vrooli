import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GrantLevel(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    GRANT_LEVEL_UNSPECIFIED: _ClassVar[GrantLevel]
    GRANT_LEVEL_ACT: _ClassVar[GrantLevel]
    GRANT_LEVEL_PROPOSE: _ClassVar[GrantLevel]

class ResolvedPersonaKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RESOLVED_PERSONA_KIND_UNSPECIFIED: _ClassVar[ResolvedPersonaKind]
    RESOLVED_PERSONA_KIND_PERSONAL: _ClassVar[ResolvedPersonaKind]
    RESOLVED_PERSONA_KIND_BUSINESS: _ClassVar[ResolvedPersonaKind]
GRANT_LEVEL_UNSPECIFIED: GrantLevel
GRANT_LEVEL_ACT: GrantLevel
GRANT_LEVEL_PROPOSE: GrantLevel
RESOLVED_PERSONA_KIND_UNSPECIFIED: ResolvedPersonaKind
RESOLVED_PERSONA_KIND_PERSONAL: ResolvedPersonaKind
RESOLVED_PERSONA_KIND_BUSINESS: ResolvedPersonaKind

class PersonaGrant(_message.Message):
    __slots__ = ("id", "persona_id", "human_subject", "level", "source", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    HUMAN_SUBJECT_FIELD_NUMBER: _ClassVar[int]
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    persona_id: str
    human_subject: str
    level: GrantLevel
    source: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., persona_id: _Optional[str] = ..., human_subject: _Optional[str] = ..., level: _Optional[_Union[GrantLevel, str]] = ..., source: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ActAsRequest(_message.Message):
    __slots__ = ("persona_id", "action")
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    action: str
    def __init__(self, persona_id: _Optional[str] = ..., action: _Optional[str] = ...) -> None: ...

class ActAsResponse(_message.Message):
    __slots__ = ("persona_id", "run_id", "account_subject", "authorising_human", "granted_at")
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_SUBJECT_FIELD_NUMBER: _ClassVar[int]
    AUTHORISING_HUMAN_FIELD_NUMBER: _ClassVar[int]
    GRANTED_AT_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    run_id: str
    account_subject: str
    authorising_human: str
    granted_at: _timestamp_pb2.Timestamp
    def __init__(self, persona_id: _Optional[str] = ..., run_id: _Optional[str] = ..., account_subject: _Optional[str] = ..., authorising_human: _Optional[str] = ..., granted_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ResolvePersonaRequest(_message.Message):
    __slots__ = ("persona_id", "fields")
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    FIELDS_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    fields: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, persona_id: _Optional[str] = ..., fields: _Optional[_Iterable[str]] = ...) -> None: ...

class PersonaResolution(_message.Message):
    __slots__ = ("persona_id", "kind", "display_name", "legal_subject_id", "controlled_email", "address_ids", "returned_fields")
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    LEGAL_SUBJECT_ID_FIELD_NUMBER: _ClassVar[int]
    CONTROLLED_EMAIL_FIELD_NUMBER: _ClassVar[int]
    ADDRESS_IDS_FIELD_NUMBER: _ClassVar[int]
    RETURNED_FIELDS_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    kind: ResolvedPersonaKind
    display_name: str
    legal_subject_id: str
    controlled_email: str
    address_ids: _containers.RepeatedScalarFieldContainer[str]
    returned_fields: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, persona_id: _Optional[str] = ..., kind: _Optional[_Union[ResolvedPersonaKind, str]] = ..., display_name: _Optional[str] = ..., legal_subject_id: _Optional[str] = ..., controlled_email: _Optional[str] = ..., address_ids: _Optional[_Iterable[str]] = ..., returned_fields: _Optional[_Iterable[str]] = ...) -> None: ...

class ResolvePersonaResponse(_message.Message):
    __slots__ = ("persona",)
    PERSONA_FIELD_NUMBER: _ClassVar[int]
    persona: PersonaResolution
    def __init__(self, persona: _Optional[_Union[PersonaResolution, _Mapping]] = ...) -> None: ...

class CreateGrantRequest(_message.Message):
    __slots__ = ("persona_id", "human_subject", "level", "source")
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    HUMAN_SUBJECT_FIELD_NUMBER: _ClassVar[int]
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    human_subject: str
    level: GrantLevel
    source: str
    def __init__(self, persona_id: _Optional[str] = ..., human_subject: _Optional[str] = ..., level: _Optional[_Union[GrantLevel, str]] = ..., source: _Optional[str] = ...) -> None: ...

class CreateGrantResponse(_message.Message):
    __slots__ = ("grant",)
    GRANT_FIELD_NUMBER: _ClassVar[int]
    grant: PersonaGrant
    def __init__(self, grant: _Optional[_Union[PersonaGrant, _Mapping]] = ...) -> None: ...

class ListGrantsRequest(_message.Message):
    __slots__ = ("persona_id",)
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    def __init__(self, persona_id: _Optional[str] = ...) -> None: ...

class ListGrantsResponse(_message.Message):
    __slots__ = ("grants",)
    GRANTS_FIELD_NUMBER: _ClassVar[int]
    grants: _containers.RepeatedCompositeFieldContainer[PersonaGrant]
    def __init__(self, grants: _Optional[_Iterable[_Union[PersonaGrant, _Mapping]]] = ...) -> None: ...

class RemoveGrantRequest(_message.Message):
    __slots__ = ("grant_id",)
    GRANT_ID_FIELD_NUMBER: _ClassVar[int]
    grant_id: str
    def __init__(self, grant_id: _Optional[str] = ...) -> None: ...

class RemoveGrantResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class IssueAttestationRequest(_message.Message):
    __slots__ = ("persona_id", "audience", "expires_at_unix")
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    AUDIENCE_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_UNIX_FIELD_NUMBER: _ClassVar[int]
    persona_id: str
    audience: str
    expires_at_unix: int
    def __init__(self, persona_id: _Optional[str] = ..., audience: _Optional[str] = ..., expires_at_unix: _Optional[int] = ...) -> None: ...

class IdentityAttestation(_message.Message):
    __slots__ = ("issuer", "audience", "legal_person", "persona_id", "account_subject", "run_id", "issued_at_unix", "expires_at_unix", "claim_format", "signature", "key_id")
    ISSUER_FIELD_NUMBER: _ClassVar[int]
    AUDIENCE_FIELD_NUMBER: _ClassVar[int]
    LEGAL_PERSON_FIELD_NUMBER: _ClassVar[int]
    PERSONA_ID_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_SUBJECT_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    ISSUED_AT_UNIX_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_UNIX_FIELD_NUMBER: _ClassVar[int]
    CLAIM_FORMAT_FIELD_NUMBER: _ClassVar[int]
    SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    KEY_ID_FIELD_NUMBER: _ClassVar[int]
    issuer: str
    audience: str
    legal_person: str
    persona_id: str
    account_subject: str
    run_id: str
    issued_at_unix: int
    expires_at_unix: int
    claim_format: str
    signature: bytes
    key_id: str
    def __init__(self, issuer: _Optional[str] = ..., audience: _Optional[str] = ..., legal_person: _Optional[str] = ..., persona_id: _Optional[str] = ..., account_subject: _Optional[str] = ..., run_id: _Optional[str] = ..., issued_at_unix: _Optional[int] = ..., expires_at_unix: _Optional[int] = ..., claim_format: _Optional[str] = ..., signature: _Optional[bytes] = ..., key_id: _Optional[str] = ...) -> None: ...

class IssueAttestationResponse(_message.Message):
    __slots__ = ("attestation",)
    ATTESTATION_FIELD_NUMBER: _ClassVar[int]
    attestation: IdentityAttestation
    def __init__(self, attestation: _Optional[_Union[IdentityAttestation, _Mapping]] = ...) -> None: ...
