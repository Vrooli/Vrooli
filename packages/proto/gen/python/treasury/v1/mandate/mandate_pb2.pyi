import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class MandateStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    MANDATE_STATUS_UNSPECIFIED: _ClassVar[MandateStatus]
    MANDATE_STATUS_DRAFT: _ClassVar[MandateStatus]
    MANDATE_STATUS_LIVE: _ClassVar[MandateStatus]
    MANDATE_STATUS_EXHAUSTED: _ClassVar[MandateStatus]
    MANDATE_STATUS_EXPIRED: _ClassVar[MandateStatus]
    MANDATE_STATUS_REVOKED: _ClassVar[MandateStatus]
MANDATE_STATUS_UNSPECIFIED: MandateStatus
MANDATE_STATUS_DRAFT: MandateStatus
MANDATE_STATUS_LIVE: MandateStatus
MANDATE_STATUS_EXHAUSTED: MandateStatus
MANDATE_STATUS_EXPIRED: MandateStatus
MANDATE_STATUS_REVOKED: MandateStatus

class Mandate(_message.Message):
    __slots__ = ("id", "book_id", "budget_id", "authorizer", "cap_minor", "currency", "allowed_counterparties", "denied_counterparties", "expires_at", "signature", "issued_at", "status", "idempotency_key", "required_evidence", "recurrence_seconds", "next_charge_at", "cancelled_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    BUDGET_ID_FIELD_NUMBER: _ClassVar[int]
    AUTHORIZER_FIELD_NUMBER: _ClassVar[int]
    CAP_MINOR_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_COUNTERPARTIES_FIELD_NUMBER: _ClassVar[int]
    DENIED_COUNTERPARTIES_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    ISSUED_AT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    RECURRENCE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    NEXT_CHARGE_AT_FIELD_NUMBER: _ClassVar[int]
    CANCELLED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    book_id: str
    budget_id: str
    authorizer: str
    cap_minor: int
    currency: str
    allowed_counterparties: _containers.RepeatedScalarFieldContainer[str]
    denied_counterparties: _containers.RepeatedScalarFieldContainer[str]
    expires_at: _timestamp_pb2.Timestamp
    signature: bytes
    issued_at: _timestamp_pb2.Timestamp
    status: MandateStatus
    idempotency_key: str
    required_evidence: _containers.RepeatedScalarFieldContainer[str]
    recurrence_seconds: int
    next_charge_at: _timestamp_pb2.Timestamp
    cancelled_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., book_id: _Optional[str] = ..., budget_id: _Optional[str] = ..., authorizer: _Optional[str] = ..., cap_minor: _Optional[int] = ..., currency: _Optional[str] = ..., allowed_counterparties: _Optional[_Iterable[str]] = ..., denied_counterparties: _Optional[_Iterable[str]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., signature: _Optional[bytes] = ..., issued_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., status: _Optional[_Union[MandateStatus, str]] = ..., idempotency_key: _Optional[str] = ..., required_evidence: _Optional[_Iterable[str]] = ..., recurrence_seconds: _Optional[int] = ..., next_charge_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., cancelled_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
