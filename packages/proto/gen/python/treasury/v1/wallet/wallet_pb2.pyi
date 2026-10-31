import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Wallet(_message.Message):
    __slots__ = ("id", "book_id", "operator_identity", "network", "address", "public_key", "custody_reference", "created_at", "recovery_status", "recovery_sha256", "independent_recovery_verified")
    ID_FIELD_NUMBER: _ClassVar[int]
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    PUBLIC_KEY_FIELD_NUMBER: _ClassVar[int]
    CUSTODY_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_STATUS_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_SHA256_FIELD_NUMBER: _ClassVar[int]
    INDEPENDENT_RECOVERY_VERIFIED_FIELD_NUMBER: _ClassVar[int]
    id: str
    book_id: str
    operator_identity: str
    network: str
    address: str
    public_key: str
    custody_reference: str
    created_at: _timestamp_pb2.Timestamp
    recovery_status: str
    recovery_sha256: str
    independent_recovery_verified: bool
    def __init__(self, id: _Optional[str] = ..., book_id: _Optional[str] = ..., operator_identity: _Optional[str] = ..., network: _Optional[str] = ..., address: _Optional[str] = ..., public_key: _Optional[str] = ..., custody_reference: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., recovery_status: _Optional[str] = ..., recovery_sha256: _Optional[str] = ..., independent_recovery_verified: _Optional[bool] = ...) -> None: ...

class ControlProof(_message.Message):
    __slots__ = ("wallet", "message", "signature", "scheme", "verified")
    WALLET_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    SCHEME_FIELD_NUMBER: _ClassVar[int]
    VERIFIED_FIELD_NUMBER: _ClassVar[int]
    wallet: Wallet
    message: str
    signature: str
    scheme: str
    verified: bool
    def __init__(self, wallet: _Optional[_Union[Wallet, _Mapping]] = ..., message: _Optional[str] = ..., signature: _Optional[str] = ..., scheme: _Optional[str] = ..., verified: _Optional[bool] = ...) -> None: ...
