import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RedemptionState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    REDEMPTION_STATE_UNSPECIFIED: _ClassVar[RedemptionState]
    REDEMPTION_STATE_PENDING_APPROVAL: _ClassVar[RedemptionState]
    REDEMPTION_STATE_SETTLED: _ClassVar[RedemptionState]
    REDEMPTION_STATE_DENIED: _ClassVar[RedemptionState]

class ReservationState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RESERVATION_STATE_UNSPECIFIED: _ClassVar[ReservationState]
    RESERVATION_STATE_ACTIVE: _ClassVar[ReservationState]
    RESERVATION_STATE_SETTLED: _ClassVar[ReservationState]
    RESERVATION_STATE_RELEASED: _ClassVar[ReservationState]
REDEMPTION_STATE_UNSPECIFIED: RedemptionState
REDEMPTION_STATE_PENDING_APPROVAL: RedemptionState
REDEMPTION_STATE_SETTLED: RedemptionState
REDEMPTION_STATE_DENIED: RedemptionState
RESERVATION_STATE_UNSPECIFIED: ReservationState
RESERVATION_STATE_ACTIVE: ReservationState
RESERVATION_STATE_SETTLED: ReservationState
RESERVATION_STATE_RELEASED: ReservationState

class Redemption(_message.Message):
    __slots__ = ("id", "holder_id", "catalog_entry_id", "token_type_id", "grant_id", "amount", "idempotency_key", "state", "decider_subject", "decision_reason", "requested_at", "decided_at", "settled_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    HOLDER_ID_FIELD_NUMBER: _ClassVar[int]
    CATALOG_ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    TOKEN_TYPE_ID_FIELD_NUMBER: _ClassVar[int]
    GRANT_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    DECIDER_SUBJECT_FIELD_NUMBER: _ClassVar[int]
    DECISION_REASON_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_AT_FIELD_NUMBER: _ClassVar[int]
    DECIDED_AT_FIELD_NUMBER: _ClassVar[int]
    SETTLED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    holder_id: str
    catalog_entry_id: str
    token_type_id: str
    grant_id: str
    amount: int
    idempotency_key: str
    state: RedemptionState
    decider_subject: str
    decision_reason: str
    requested_at: _timestamp_pb2.Timestamp
    decided_at: _timestamp_pb2.Timestamp
    settled_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., holder_id: _Optional[str] = ..., catalog_entry_id: _Optional[str] = ..., token_type_id: _Optional[str] = ..., grant_id: _Optional[str] = ..., amount: _Optional[int] = ..., idempotency_key: _Optional[str] = ..., state: _Optional[_Union[RedemptionState, str]] = ..., decider_subject: _Optional[str] = ..., decision_reason: _Optional[str] = ..., requested_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., decided_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., settled_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Reservation(_message.Message):
    __slots__ = ("id", "redemption_id", "holder_id", "token_type_id", "amount", "state", "created_at", "released_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    REDEMPTION_ID_FIELD_NUMBER: _ClassVar[int]
    HOLDER_ID_FIELD_NUMBER: _ClassVar[int]
    TOKEN_TYPE_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    RELEASED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    redemption_id: str
    holder_id: str
    token_type_id: str
    amount: int
    state: ReservationState
    created_at: _timestamp_pb2.Timestamp
    released_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., redemption_id: _Optional[str] = ..., holder_id: _Optional[str] = ..., token_type_id: _Optional[str] = ..., amount: _Optional[int] = ..., state: _Optional[_Union[ReservationState, str]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., released_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
