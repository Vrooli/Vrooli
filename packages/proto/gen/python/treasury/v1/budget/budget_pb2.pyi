import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Budget(_message.Message):
    __slots__ = ("id", "book_id", "total_cap_minor", "periodic_cap_minor", "per_transaction_cap_minor", "currency", "allowed_counterparties", "denied_counterparties", "requires_approval", "frozen", "period_seconds")
    ID_FIELD_NUMBER: _ClassVar[int]
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    TOTAL_CAP_MINOR_FIELD_NUMBER: _ClassVar[int]
    PERIODIC_CAP_MINOR_FIELD_NUMBER: _ClassVar[int]
    PER_TRANSACTION_CAP_MINOR_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_COUNTERPARTIES_FIELD_NUMBER: _ClassVar[int]
    DENIED_COUNTERPARTIES_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_APPROVAL_FIELD_NUMBER: _ClassVar[int]
    FROZEN_FIELD_NUMBER: _ClassVar[int]
    PERIOD_SECONDS_FIELD_NUMBER: _ClassVar[int]
    id: str
    book_id: str
    total_cap_minor: int
    periodic_cap_minor: int
    per_transaction_cap_minor: int
    currency: str
    allowed_counterparties: _containers.RepeatedScalarFieldContainer[str]
    denied_counterparties: _containers.RepeatedScalarFieldContainer[str]
    requires_approval: bool
    frozen: bool
    period_seconds: int
    def __init__(self, id: _Optional[str] = ..., book_id: _Optional[str] = ..., total_cap_minor: _Optional[int] = ..., periodic_cap_minor: _Optional[int] = ..., per_transaction_cap_minor: _Optional[int] = ..., currency: _Optional[str] = ..., allowed_counterparties: _Optional[_Iterable[str]] = ..., denied_counterparties: _Optional[_Iterable[str]] = ..., requires_approval: _Optional[bool] = ..., frozen: _Optional[bool] = ..., period_seconds: _Optional[int] = ...) -> None: ...

class Headroom(_message.Message):
    __slots__ = ("budget_id", "book_id", "currency", "total_cap_minor", "total_used_minor", "total_remaining_minor", "periodic_cap_minor", "period_used_minor", "period_remaining_minor", "per_transaction_cap_minor", "available_minor", "period_started_at", "computed_at")
    BUDGET_ID_FIELD_NUMBER: _ClassVar[int]
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    TOTAL_CAP_MINOR_FIELD_NUMBER: _ClassVar[int]
    TOTAL_USED_MINOR_FIELD_NUMBER: _ClassVar[int]
    TOTAL_REMAINING_MINOR_FIELD_NUMBER: _ClassVar[int]
    PERIODIC_CAP_MINOR_FIELD_NUMBER: _ClassVar[int]
    PERIOD_USED_MINOR_FIELD_NUMBER: _ClassVar[int]
    PERIOD_REMAINING_MINOR_FIELD_NUMBER: _ClassVar[int]
    PER_TRANSACTION_CAP_MINOR_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_MINOR_FIELD_NUMBER: _ClassVar[int]
    PERIOD_STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPUTED_AT_FIELD_NUMBER: _ClassVar[int]
    budget_id: str
    book_id: str
    currency: str
    total_cap_minor: int
    total_used_minor: int
    total_remaining_minor: int
    periodic_cap_minor: int
    period_used_minor: int
    period_remaining_minor: int
    per_transaction_cap_minor: int
    available_minor: int
    period_started_at: _timestamp_pb2.Timestamp
    computed_at: _timestamp_pb2.Timestamp
    def __init__(self, budget_id: _Optional[str] = ..., book_id: _Optional[str] = ..., currency: _Optional[str] = ..., total_cap_minor: _Optional[int] = ..., total_used_minor: _Optional[int] = ..., total_remaining_minor: _Optional[int] = ..., periodic_cap_minor: _Optional[int] = ..., period_used_minor: _Optional[int] = ..., period_remaining_minor: _Optional[int] = ..., per_transaction_cap_minor: _Optional[int] = ..., available_minor: _Optional[int] = ..., period_started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., computed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
