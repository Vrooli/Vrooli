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
    __slots__ = ("emissions", "destination", "queued_total")
    EMISSIONS_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_FIELD_NUMBER: _ClassVar[int]
    QUEUED_TOTAL_FIELD_NUMBER: _ClassVar[int]
    emissions: _containers.RepeatedCompositeFieldContainer[LedgerEmission]
    destination: LedgerDestination
    queued_total: int
    def __init__(self, emissions: _Optional[_Iterable[_Union[LedgerEmission, _Mapping]]] = ..., destination: _Optional[_Union[LedgerDestination, _Mapping]] = ..., queued_total: _Optional[int] = ...) -> None: ...

class LedgerDestination(_message.Message):
    __slots__ = ("configured", "book_id", "account_id", "book_name", "account_name", "currency", "source", "set_at", "unconfigured_reason")
    CONFIGURED_FIELD_NUMBER: _ClassVar[int]
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_ID_FIELD_NUMBER: _ClassVar[int]
    BOOK_NAME_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_NAME_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SET_AT_FIELD_NUMBER: _ClassVar[int]
    UNCONFIGURED_REASON_FIELD_NUMBER: _ClassVar[int]
    configured: bool
    book_id: str
    account_id: str
    book_name: str
    account_name: str
    currency: str
    source: str
    set_at: _timestamp_pb2.Timestamp
    unconfigured_reason: str
    def __init__(self, configured: _Optional[bool] = ..., book_id: _Optional[str] = ..., account_id: _Optional[str] = ..., book_name: _Optional[str] = ..., account_name: _Optional[str] = ..., currency: _Optional[str] = ..., source: _Optional[str] = ..., set_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., unconfigured_reason: _Optional[str] = ...) -> None: ...

class SetLedgerDestinationRequest(_message.Message):
    __slots__ = ("book_id", "account_id")
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_ID_FIELD_NUMBER: _ClassVar[int]
    book_id: str
    account_id: str
    def __init__(self, book_id: _Optional[str] = ..., account_id: _Optional[str] = ...) -> None: ...

class SetLedgerDestinationResponse(_message.Message):
    __slots__ = ("destination",)
    DESTINATION_FIELD_NUMBER: _ClassVar[int]
    destination: LedgerDestination
    def __init__(self, destination: _Optional[_Union[LedgerDestination, _Mapping]] = ...) -> None: ...

class GetLedgerDestinationRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetLedgerDestinationResponse(_message.Message):
    __slots__ = ("destination", "queued_total")
    DESTINATION_FIELD_NUMBER: _ClassVar[int]
    QUEUED_TOTAL_FIELD_NUMBER: _ClassVar[int]
    destination: LedgerDestination
    queued_total: int
    def __init__(self, destination: _Optional[_Union[LedgerDestination, _Mapping]] = ..., queued_total: _Optional[int] = ...) -> None: ...
