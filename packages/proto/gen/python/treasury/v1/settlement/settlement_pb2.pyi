import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ChargeOutcome(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CHARGE_OUTCOME_UNSPECIFIED: _ClassVar[ChargeOutcome]
    CHARGE_OUTCOME_READY: _ClassVar[ChargeOutcome]
    CHARGE_OUTCOME_CALLING: _ClassVar[ChargeOutcome]
    CHARGE_OUTCOME_SETTLED: _ClassVar[ChargeOutcome]
    CHARGE_OUTCOME_FAILED: _ClassVar[ChargeOutcome]
    CHARGE_OUTCOME_UNKNOWN: _ClassVar[ChargeOutcome]
    CHARGE_OUTCOME_ABANDONED: _ClassVar[ChargeOutcome]
CHARGE_OUTCOME_UNSPECIFIED: ChargeOutcome
CHARGE_OUTCOME_READY: ChargeOutcome
CHARGE_OUTCOME_CALLING: ChargeOutcome
CHARGE_OUTCOME_SETTLED: ChargeOutcome
CHARGE_OUTCOME_FAILED: ChargeOutcome
CHARGE_OUTCOME_UNKNOWN: ChargeOutcome
CHARGE_OUTCOME_ABANDONED: ChargeOutcome

class Charge(_message.Message):
    __slots__ = ("id", "authorization_id", "mandate_id", "rail", "idempotency_key", "amount_minor", "currency", "counterparty", "outcome", "external_id", "created_at", "settled_at", "receipt_reference", "basis", "detail", "occurred_at", "updated_at", "retain_until", "instrument_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    AUTHORIZATION_ID_FIELD_NUMBER: _ClassVar[int]
    MANDATE_ID_FIELD_NUMBER: _ClassVar[int]
    RAIL_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_MINOR_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    COUNTERPARTY_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    SETTLED_AT_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    BASIS_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    OCCURRED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    RETAIN_UNTIL_FIELD_NUMBER: _ClassVar[int]
    INSTRUMENT_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    authorization_id: str
    mandate_id: str
    rail: str
    idempotency_key: str
    amount_minor: int
    currency: str
    counterparty: str
    outcome: ChargeOutcome
    external_id: str
    created_at: _timestamp_pb2.Timestamp
    settled_at: _timestamp_pb2.Timestamp
    receipt_reference: str
    basis: str
    detail: str
    occurred_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    retain_until: _timestamp_pb2.Timestamp
    instrument_id: str
    def __init__(self, id: _Optional[str] = ..., authorization_id: _Optional[str] = ..., mandate_id: _Optional[str] = ..., rail: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., amount_minor: _Optional[int] = ..., currency: _Optional[str] = ..., counterparty: _Optional[str] = ..., outcome: _Optional[_Union[ChargeOutcome, str]] = ..., external_id: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., settled_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., receipt_reference: _Optional[str] = ..., basis: _Optional[str] = ..., detail: _Optional[str] = ..., occurred_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., retain_until: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., instrument_id: _Optional[str] = ...) -> None: ...

class ManualAttestation(_message.Message):
    __slots__ = ("external_reference", "receipt_reference", "occurred_at")
    EXTERNAL_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    OCCURRED_AT_FIELD_NUMBER: _ClassVar[int]
    external_reference: str
    receipt_reference: str
    occurred_at: _timestamp_pb2.Timestamp
    def __init__(self, external_reference: _Optional[str] = ..., receipt_reference: _Optional[str] = ..., occurred_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
