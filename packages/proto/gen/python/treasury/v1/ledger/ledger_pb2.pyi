import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class LedgerEmission(_message.Message):
    __slots__ = ("id", "charge_id", "external_id", "adapter_id", "amount_minor", "currency", "basis", "fetched_at", "accepted_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    CHARGE_ID_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_ID_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_ID_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_MINOR_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    BASIS_FIELD_NUMBER: _ClassVar[int]
    FETCHED_AT_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    charge_id: str
    external_id: str
    adapter_id: str
    amount_minor: int
    currency: str
    basis: str
    fetched_at: _timestamp_pb2.Timestamp
    accepted_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., charge_id: _Optional[str] = ..., external_id: _Optional[str] = ..., adapter_id: _Optional[str] = ..., amount_minor: _Optional[int] = ..., currency: _Optional[str] = ..., basis: _Optional[str] = ..., fetched_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., accepted_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
