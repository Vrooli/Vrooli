import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EventKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EVENT_KIND_UNSPECIFIED: _ClassVar[EventKind]
    EVENT_KIND_MINT: _ClassVar[EventKind]
    EVENT_KIND_CREDIT: _ClassVar[EventKind]
    EVENT_KIND_DEBIT: _ClassVar[EventKind]
    EVENT_KIND_REVERSAL: _ClassVar[EventKind]
    EVENT_KIND_EXPIRY: _ClassVar[EventKind]

class ActorKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ACTOR_KIND_UNSPECIFIED: _ClassVar[ActorKind]
    ACTOR_KIND_OPERATOR: _ClassVar[ActorKind]
    ACTOR_KIND_AGENT: _ClassVar[ActorKind]

class VerificationStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VERIFICATION_STATUS_UNSPECIFIED: _ClassVar[VerificationStatus]
    VERIFICATION_STATUS_VERIFIED: _ClassVar[VerificationStatus]
    VERIFICATION_STATUS_UNAVAILABLE: _ClassVar[VerificationStatus]
    VERIFICATION_STATUS_INVALID: _ClassVar[VerificationStatus]
    VERIFICATION_STATUS_ABSENT: _ClassVar[VerificationStatus]
EVENT_KIND_UNSPECIFIED: EventKind
EVENT_KIND_MINT: EventKind
EVENT_KIND_CREDIT: EventKind
EVENT_KIND_DEBIT: EventKind
EVENT_KIND_REVERSAL: EventKind
EVENT_KIND_EXPIRY: EventKind
ACTOR_KIND_UNSPECIFIED: ActorKind
ACTOR_KIND_OPERATOR: ActorKind
ACTOR_KIND_AGENT: ActorKind
VERIFICATION_STATUS_UNSPECIFIED: VerificationStatus
VERIFICATION_STATUS_VERIFIED: VerificationStatus
VERIFICATION_STATUS_UNAVAILABLE: VerificationStatus
VERIFICATION_STATUS_INVALID: VerificationStatus
VERIFICATION_STATUS_ABSENT: VerificationStatus

class Event(_message.Message):
    __slots__ = ("id", "token_type_id", "holder_id", "amount", "kind", "cause_reference", "created_at", "actor_identity", "reason", "actor_kind", "actor_verification_status", "actor_run_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    TOKEN_TYPE_ID_FIELD_NUMBER: _ClassVar[int]
    HOLDER_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    CAUSE_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    ACTOR_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    ACTOR_KIND_FIELD_NUMBER: _ClassVar[int]
    ACTOR_VERIFICATION_STATUS_FIELD_NUMBER: _ClassVar[int]
    ACTOR_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    token_type_id: str
    holder_id: str
    amount: int
    kind: EventKind
    cause_reference: str
    created_at: _timestamp_pb2.Timestamp
    actor_identity: str
    reason: str
    actor_kind: ActorKind
    actor_verification_status: VerificationStatus
    actor_run_id: str
    def __init__(self, id: _Optional[str] = ..., token_type_id: _Optional[str] = ..., holder_id: _Optional[str] = ..., amount: _Optional[int] = ..., kind: _Optional[_Union[EventKind, str]] = ..., cause_reference: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., actor_identity: _Optional[str] = ..., reason: _Optional[str] = ..., actor_kind: _Optional[_Union[ActorKind, str]] = ..., actor_verification_status: _Optional[_Union[VerificationStatus, str]] = ..., actor_run_id: _Optional[str] = ...) -> None: ...
