import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from money_ledger.v1.ingest import ingest_pb2 as _ingest_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ReceiptFundingState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RECEIPT_FUNDING_STATE_UNSPECIFIED: _ClassVar[ReceiptFundingState]
    RECEIPT_FUNDING_STATE_QUEUED: _ClassVar[ReceiptFundingState]
    RECEIPT_FUNDING_STATE_ACCEPTED: _ClassVar[ReceiptFundingState]
RECEIPT_FUNDING_STATE_UNSPECIFIED: ReceiptFundingState
RECEIPT_FUNDING_STATE_QUEUED: ReceiptFundingState
RECEIPT_FUNDING_STATE_ACCEPTED: ReceiptFundingState

class SubmitReceiptFundingRequest(_message.Message):
    __slots__ = ("wallet_id", "transaction_hash", "check_id", "log_index", "budget_id", "ledger_book_id", "ledger_account_id", "action", "original_receipt_id", "previous_receipt_id", "reason", "valuation", "attribution")
    WALLET_ID_FIELD_NUMBER: _ClassVar[int]
    TRANSACTION_HASH_FIELD_NUMBER: _ClassVar[int]
    CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    LOG_INDEX_FIELD_NUMBER: _ClassVar[int]
    BUDGET_ID_FIELD_NUMBER: _ClassVar[int]
    LEDGER_BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    LEDGER_ACCOUNT_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    ORIGINAL_RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    VALUATION_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    wallet_id: str
    transaction_hash: str
    check_id: str
    log_index: int
    budget_id: str
    ledger_book_id: str
    ledger_account_id: str
    action: _ingest_pb2.CryptoReceiptAction
    original_receipt_id: str
    previous_receipt_id: str
    reason: str
    valuation: _ingest_pb2.CryptoValuation
    attribution: _ingest_pb2.CryptoPayerAttribution
    def __init__(self, wallet_id: _Optional[str] = ..., transaction_hash: _Optional[str] = ..., check_id: _Optional[str] = ..., log_index: _Optional[int] = ..., budget_id: _Optional[str] = ..., ledger_book_id: _Optional[str] = ..., ledger_account_id: _Optional[str] = ..., action: _Optional[_Union[_ingest_pb2.CryptoReceiptAction, str]] = ..., original_receipt_id: _Optional[str] = ..., previous_receipt_id: _Optional[str] = ..., reason: _Optional[str] = ..., valuation: _Optional[_Union[_ingest_pb2.CryptoValuation, _Mapping]] = ..., attribution: _Optional[_Union[_ingest_pb2.CryptoPayerAttribution, _Mapping]] = ...) -> None: ...

class ReceiptFundingDelivery(_message.Message):
    __slots__ = ("id", "wallet_id", "budget_id", "source_check_id", "state", "attempts", "last_error", "created_at", "accepted_at", "receipt", "credit_lot_id", "credit_lot_owner", "credit_lot_receipt_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    WALLET_ID_FIELD_NUMBER: _ClassVar[int]
    BUDGET_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    LAST_ERROR_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_AT_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    CREDIT_LOT_ID_FIELD_NUMBER: _ClassVar[int]
    CREDIT_LOT_OWNER_FIELD_NUMBER: _ClassVar[int]
    CREDIT_LOT_RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    wallet_id: str
    budget_id: str
    source_check_id: str
    state: ReceiptFundingState
    attempts: int
    last_error: str
    created_at: _timestamp_pb2.Timestamp
    accepted_at: _timestamp_pb2.Timestamp
    receipt: _ingest_pb2.AcceptedCryptoReceipt
    credit_lot_id: str
    credit_lot_owner: str
    credit_lot_receipt_id: str
    def __init__(self, id: _Optional[str] = ..., wallet_id: _Optional[str] = ..., budget_id: _Optional[str] = ..., source_check_id: _Optional[str] = ..., state: _Optional[_Union[ReceiptFundingState, str]] = ..., attempts: _Optional[int] = ..., last_error: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., accepted_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., receipt: _Optional[_Union[_ingest_pb2.AcceptedCryptoReceipt, _Mapping]] = ..., credit_lot_id: _Optional[str] = ..., credit_lot_owner: _Optional[str] = ..., credit_lot_receipt_id: _Optional[str] = ...) -> None: ...

class ReceiptFundingResponse(_message.Message):
    __slots__ = ("delivery",)
    DELIVERY_FIELD_NUMBER: _ClassVar[int]
    delivery: ReceiptFundingDelivery
    def __init__(self, delivery: _Optional[_Union[ReceiptFundingDelivery, _Mapping]] = ...) -> None: ...

class GetReceiptFundingRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...
