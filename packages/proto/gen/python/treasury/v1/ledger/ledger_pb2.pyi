import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class LedgerEmission(_message.Message):
    __slots__ = ("id", "charge_id", "external_id", "adapter_id", "amount_minor", "currency", "basis", "fetched_at", "accepted_at", "status", "attempts", "last_error", "settlement_id", "account_id", "book_id", "occurred_at", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    CHARGE_ID_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_MINOR_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    BASIS_FIELD_NUMBER: _ClassVar[int]
    FETCHED_AT_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_AT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    LAST_ERROR_FIELD_NUMBER: _ClassVar[int]
    SETTLEMENT_ID_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_ID_FIELD_NUMBER: _ClassVar[int]
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    OCCURRED_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    charge_id: str
    external_id: str
    adapter_id: str
    amount_minor: int
    currency: str
    basis: str
    fetched_at: _timestamp_pb2.Timestamp
    accepted_at: _timestamp_pb2.Timestamp
    status: str
    attempts: int
    last_error: str
    settlement_id: str
    account_id: str
    book_id: str
    occurred_at: _timestamp_pb2.Timestamp
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., charge_id: _Optional[str] = ..., external_id: _Optional[str] = ..., adapter_id: _Optional[str] = ..., amount_minor: _Optional[int] = ..., currency: _Optional[str] = ..., basis: _Optional[str] = ..., fetched_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., accepted_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., status: _Optional[str] = ..., attempts: _Optional[int] = ..., last_error: _Optional[str] = ..., settlement_id: _Optional[str] = ..., account_id: _Optional[str] = ..., book_id: _Optional[str] = ..., occurred_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListEmissionsRequest(_message.Message):
    __slots__ = ("external_id", "status", "limit")
    EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    external_id: str
    status: str
    limit: int
    def __init__(self, external_id: _Optional[str] = ..., status: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListEmissionsResponse(_message.Message):
    __slots__ = ("emissions",)
    EMISSIONS_FIELD_NUMBER: _ClassVar[int]
    emissions: _containers.RepeatedCompositeFieldContainer[LedgerEmission]
    def __init__(self, emissions: _Optional[_Iterable[_Union[LedgerEmission, _Mapping]]] = ...) -> None: ...
