import datetime

from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EvidenceRecord(_message.Message):
    __slots__ = ("id", "authorization_id", "mandate_id", "approval_id", "charge_id", "outcome", "basis", "request", "rail_response", "receipt", "recorded_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    AUTHORIZATION_ID_FIELD_NUMBER: _ClassVar[int]
    MANDATE_ID_FIELD_NUMBER: _ClassVar[int]
    APPROVAL_ID_FIELD_NUMBER: _ClassVar[int]
    CHARGE_ID_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    BASIS_FIELD_NUMBER: _ClassVar[int]
    REQUEST_FIELD_NUMBER: _ClassVar[int]
    RAIL_RESPONSE_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    authorization_id: str
    mandate_id: str
    approval_id: str
    charge_id: str
    outcome: str
    basis: str
    request: _struct_pb2.Struct
    rail_response: _struct_pb2.Struct
    receipt: _struct_pb2.Struct
    recorded_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., authorization_id: _Optional[str] = ..., mandate_id: _Optional[str] = ..., approval_id: _Optional[str] = ..., charge_id: _Optional[str] = ..., outcome: _Optional[str] = ..., basis: _Optional[str] = ..., request: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., rail_response: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., receipt: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., recorded_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
