import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
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

class BaseUSDCTransferObservation(_message.Message):
    __slots__ = ("economic_key", "inclusion_revision", "wallet_id", "network", "asset_id", "decimals", "to", "atomic_amount", "transaction_hash", "log_index", "block_number", "block_hash", "block_time", "finality", "finalized_head_number", "finalized_head_hash", "provider_id", "observed_at")
    ECONOMIC_KEY_FIELD_NUMBER: _ClassVar[int]
    INCLUSION_REVISION_FIELD_NUMBER: _ClassVar[int]
    WALLET_ID_FIELD_NUMBER: _ClassVar[int]
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    DECIMALS_FIELD_NUMBER: _ClassVar[int]
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    ATOMIC_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    TRANSACTION_HASH_FIELD_NUMBER: _ClassVar[int]
    LOG_INDEX_FIELD_NUMBER: _ClassVar[int]
    BLOCK_NUMBER_FIELD_NUMBER: _ClassVar[int]
    BLOCK_HASH_FIELD_NUMBER: _ClassVar[int]
    BLOCK_TIME_FIELD_NUMBER: _ClassVar[int]
    FINALITY_FIELD_NUMBER: _ClassVar[int]
    FINALIZED_HEAD_NUMBER_FIELD_NUMBER: _ClassVar[int]
    FINALIZED_HEAD_HASH_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    economic_key: str
    inclusion_revision: str
    wallet_id: str
    network: str
    asset_id: str
    decimals: int
    to: str
    atomic_amount: str
    transaction_hash: str
    log_index: int
    block_number: int
    block_hash: str
    block_time: _timestamp_pb2.Timestamp
    finality: str
    finalized_head_number: int
    finalized_head_hash: str
    provider_id: str
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, economic_key: _Optional[str] = ..., inclusion_revision: _Optional[str] = ..., wallet_id: _Optional[str] = ..., network: _Optional[str] = ..., asset_id: _Optional[str] = ..., decimals: _Optional[int] = ..., to: _Optional[str] = ..., atomic_amount: _Optional[str] = ..., transaction_hash: _Optional[str] = ..., log_index: _Optional[int] = ..., block_number: _Optional[int] = ..., block_hash: _Optional[str] = ..., block_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., finality: _Optional[str] = ..., finalized_head_number: _Optional[int] = ..., finalized_head_hash: _Optional[str] = ..., provider_id: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., **kwargs) -> None: ...

class BaseUSDCReceiptCheck(_message.Message):
    __slots__ = ("id", "wallet_id", "network", "transaction_hash", "provider_id", "outcome", "error_code", "checked_at", "transfers")
    ID_FIELD_NUMBER: _ClassVar[int]
    WALLET_ID_FIELD_NUMBER: _ClassVar[int]
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    TRANSACTION_HASH_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    ERROR_CODE_FIELD_NUMBER: _ClassVar[int]
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    TRANSFERS_FIELD_NUMBER: _ClassVar[int]
    id: str
    wallet_id: str
    network: str
    transaction_hash: str
    provider_id: str
    outcome: str
    error_code: str
    checked_at: _timestamp_pb2.Timestamp
    transfers: _containers.RepeatedCompositeFieldContainer[BaseUSDCTransferObservation]
    def __init__(self, id: _Optional[str] = ..., wallet_id: _Optional[str] = ..., network: _Optional[str] = ..., transaction_hash: _Optional[str] = ..., provider_id: _Optional[str] = ..., outcome: _Optional[str] = ..., error_code: _Optional[str] = ..., checked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., transfers: _Optional[_Iterable[_Union[BaseUSDCTransferObservation, _Mapping]]] = ...) -> None: ...
