import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Instrument(_message.Message):
    __slots__ = ("id", "book_id", "mandate_id", "rail", "credential_reference", "cap_minor", "currency", "counterparty", "expires_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    MANDATE_ID_FIELD_NUMBER: _ClassVar[int]
    RAIL_FIELD_NUMBER: _ClassVar[int]
    CREDENTIAL_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    CAP_MINOR_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    COUNTERPARTY_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    book_id: str
    mandate_id: str
    rail: str
    credential_reference: str
    cap_minor: int
    currency: str
    counterparty: str
    expires_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., book_id: _Optional[str] = ..., mandate_id: _Optional[str] = ..., rail: _Optional[str] = ..., credential_reference: _Optional[str] = ..., cap_minor: _Optional[int] = ..., currency: _Optional[str] = ..., counterparty: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
