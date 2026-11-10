import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from money_ledger.v1.ingest import ingest_pb2 as _ingest_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class BudgetWallet(_message.Message):
    __slots__ = ("budget_id", "wallet_id", "network", "address", "agent_subject", "agent_run_id", "mandate_id", "session_generation", "session_expires_at")
    BUDGET_ID_FIELD_NUMBER: _ClassVar[int]
    WALLET_ID_FIELD_NUMBER: _ClassVar[int]
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    AGENT_SUBJECT_FIELD_NUMBER: _ClassVar[int]
    AGENT_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    MANDATE_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_GENERATION_FIELD_NUMBER: _ClassVar[int]
    SESSION_EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    budget_id: str
    wallet_id: str
    network: str
    address: str
    agent_subject: str
    agent_run_id: str
    mandate_id: str
    session_generation: int
    session_expires_at: _timestamp_pb2.Timestamp
    def __init__(self, budget_id: _Optional[str] = ..., wallet_id: _Optional[str] = ..., network: _Optional[str] = ..., address: _Optional[str] = ..., agent_subject: _Optional[str] = ..., agent_run_id: _Optional[str] = ..., mandate_id: _Optional[str] = ..., session_generation: _Optional[int] = ..., session_expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ProvisionBudgetWalletRequest(_message.Message):
    __slots__ = ("budget_id", "network", "agent_subject", "agent_run_id", "mandate_id", "rotate_session")
    BUDGET_ID_FIELD_NUMBER: _ClassVar[int]
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    AGENT_SUBJECT_FIELD_NUMBER: _ClassVar[int]
    AGENT_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    MANDATE_ID_FIELD_NUMBER: _ClassVar[int]
    ROTATE_SESSION_FIELD_NUMBER: _ClassVar[int]
    budget_id: str
    network: str
    agent_subject: str
    agent_run_id: str
    mandate_id: str
    rotate_session: bool
    def __init__(self, budget_id: _Optional[str] = ..., network: _Optional[str] = ..., agent_subject: _Optional[str] = ..., agent_run_id: _Optional[str] = ..., mandate_id: _Optional[str] = ..., rotate_session: _Optional[bool] = ...) -> None: ...

class GetBudgetWalletRequest(_message.Message):
    __slots__ = ("budget_id", "mandate_id")
    BUDGET_ID_FIELD_NUMBER: _ClassVar[int]
    MANDATE_ID_FIELD_NUMBER: _ClassVar[int]
    budget_id: str
    mandate_id: str
    def __init__(self, budget_id: _Optional[str] = ..., mandate_id: _Optional[str] = ...) -> None: ...

class BudgetWalletResponse(_message.Message):
    __slots__ = ("wallet",)
    WALLET_FIELD_NUMBER: _ClassVar[int]
    wallet: BudgetWallet
    def __init__(self, wallet: _Optional[_Union[BudgetWallet, _Mapping]] = ...) -> None: ...

class RevokeMandateSessionRequest(_message.Message):
    __slots__ = ("mandate_id",)
    MANDATE_ID_FIELD_NUMBER: _ClassVar[int]
    mandate_id: str
    def __init__(self, mandate_id: _Optional[str] = ...) -> None: ...

class RevokeMandateSessionResponse(_message.Message):
    __slots__ = ("revoked",)
    REVOKED_FIELD_NUMBER: _ClassVar[int]
    revoked: bool
    def __init__(self, revoked: _Optional[bool] = ...) -> None: ...

class FundingIntent(_message.Message):
    __slots__ = ("id", "budget_id", "wallet", "atomic_amount", "payment_uri", "state", "transaction_hash", "receipt_delivery_id", "created_at", "updated_at", "notification_state", "last_error")
    ID_FIELD_NUMBER: _ClassVar[int]
    BUDGET_ID_FIELD_NUMBER: _ClassVar[int]
    WALLET_FIELD_NUMBER: _ClassVar[int]
    ATOMIC_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    PAYMENT_URI_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    TRANSACTION_HASH_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_DELIVERY_ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    NOTIFICATION_STATE_FIELD_NUMBER: _ClassVar[int]
    LAST_ERROR_FIELD_NUMBER: _ClassVar[int]
    id: str
    budget_id: str
    wallet: BudgetWallet
    atomic_amount: str
    payment_uri: str
    state: str
    transaction_hash: str
    receipt_delivery_id: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    notification_state: str
    last_error: str
    def __init__(self, id: _Optional[str] = ..., budget_id: _Optional[str] = ..., wallet: _Optional[_Union[BudgetWallet, _Mapping]] = ..., atomic_amount: _Optional[str] = ..., payment_uri: _Optional[str] = ..., state: _Optional[str] = ..., transaction_hash: _Optional[str] = ..., receipt_delivery_id: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., notification_state: _Optional[str] = ..., last_error: _Optional[str] = ...) -> None: ...

class CreateFundingIntentRequest(_message.Message):
    __slots__ = ("id", "budget_id", "atomic_amount", "valuation", "ledger_book_id", "ledger_account_id", "attribution")
    ID_FIELD_NUMBER: _ClassVar[int]
    BUDGET_ID_FIELD_NUMBER: _ClassVar[int]
    ATOMIC_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    VALUATION_FIELD_NUMBER: _ClassVar[int]
    LEDGER_BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    LEDGER_ACCOUNT_ID_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    id: str
    budget_id: str
    atomic_amount: str
    valuation: _ingest_pb2.CryptoValuation
    ledger_book_id: str
    ledger_account_id: str
    attribution: _ingest_pb2.CryptoPayerAttribution
    def __init__(self, id: _Optional[str] = ..., budget_id: _Optional[str] = ..., atomic_amount: _Optional[str] = ..., valuation: _Optional[_Union[_ingest_pb2.CryptoValuation, _Mapping]] = ..., ledger_book_id: _Optional[str] = ..., ledger_account_id: _Optional[str] = ..., attribution: _Optional[_Union[_ingest_pb2.CryptoPayerAttribution, _Mapping]] = ...) -> None: ...

class FundingIntentRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class FundingIntentResponse(_message.Message):
    __slots__ = ("intent",)
    INTENT_FIELD_NUMBER: _ClassVar[int]
    intent: FundingIntent
    def __init__(self, intent: _Optional[_Union[FundingIntent, _Mapping]] = ...) -> None: ...

class ConfirmFundingTransferRequest(_message.Message):
    __slots__ = ("id", "transaction_hash")
    ID_FIELD_NUMBER: _ClassVar[int]
    TRANSACTION_HASH_FIELD_NUMBER: _ClassVar[int]
    id: str
    transaction_hash: str
    def __init__(self, id: _Optional[str] = ..., transaction_hash: _Optional[str] = ...) -> None: ...

class ListFundingIntentsRequest(_message.Message):
    __slots__ = ("budget_id",)
    BUDGET_ID_FIELD_NUMBER: _ClassVar[int]
    budget_id: str
    def __init__(self, budget_id: _Optional[str] = ...) -> None: ...

class ListFundingIntentsResponse(_message.Message):
    __slots__ = ("intents",)
    INTENTS_FIELD_NUMBER: _ClassVar[int]
    intents: _containers.RepeatedCompositeFieldContainer[FundingIntent]
    def __init__(self, intents: _Optional[_Iterable[_Union[FundingIntent, _Mapping]]] = ...) -> None: ...

class WithdrawalDestination(_message.Message):
    __slots__ = ("budget_id", "address", "label")
    BUDGET_ID_FIELD_NUMBER: _ClassVar[int]
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    budget_id: str
    address: str
    label: str
    def __init__(self, budget_id: _Optional[str] = ..., address: _Optional[str] = ..., label: _Optional[str] = ...) -> None: ...

class SetWithdrawalDestinationRequest(_message.Message):
    __slots__ = ("destination",)
    DESTINATION_FIELD_NUMBER: _ClassVar[int]
    destination: WithdrawalDestination
    def __init__(self, destination: _Optional[_Union[WithdrawalDestination, _Mapping]] = ...) -> None: ...

class WithdrawalDestinationResponse(_message.Message):
    __slots__ = ("destination",)
    DESTINATION_FIELD_NUMBER: _ClassVar[int]
    destination: WithdrawalDestination
    def __init__(self, destination: _Optional[_Union[WithdrawalDestination, _Mapping]] = ...) -> None: ...

class PreviewWithdrawalRequest(_message.Message):
    __slots__ = ("budget_id", "destination", "atomic_amount", "valuation", "fee_valuation_minor", "max_fee_wei", "mandate_id")
    BUDGET_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_FIELD_NUMBER: _ClassVar[int]
    ATOMIC_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    VALUATION_FIELD_NUMBER: _ClassVar[int]
    FEE_VALUATION_MINOR_FIELD_NUMBER: _ClassVar[int]
    MAX_FEE_WEI_FIELD_NUMBER: _ClassVar[int]
    MANDATE_ID_FIELD_NUMBER: _ClassVar[int]
    budget_id: str
    destination: str
    atomic_amount: str
    valuation: _ingest_pb2.CryptoValuation
    fee_valuation_minor: int
    max_fee_wei: str
    mandate_id: str
    def __init__(self, budget_id: _Optional[str] = ..., destination: _Optional[str] = ..., atomic_amount: _Optional[str] = ..., valuation: _Optional[_Union[_ingest_pb2.CryptoValuation, _Mapping]] = ..., fee_valuation_minor: _Optional[int] = ..., max_fee_wei: _Optional[str] = ..., mandate_id: _Optional[str] = ...) -> None: ...

class WithdrawalQuote(_message.Message):
    __slots__ = ("id", "budget_id", "destination", "atomic_amount", "network", "max_fee_wei", "value_minor", "fee_valuation_minor", "currency", "estimated_seconds", "expires_at", "execution_mode", "mandate_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    BUDGET_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_FIELD_NUMBER: _ClassVar[int]
    ATOMIC_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    MAX_FEE_WEI_FIELD_NUMBER: _ClassVar[int]
    VALUE_MINOR_FIELD_NUMBER: _ClassVar[int]
    FEE_VALUATION_MINOR_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    ESTIMATED_SECONDS_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_MODE_FIELD_NUMBER: _ClassVar[int]
    MANDATE_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    budget_id: str
    destination: str
    atomic_amount: str
    network: str
    max_fee_wei: str
    value_minor: int
    fee_valuation_minor: int
    currency: str
    estimated_seconds: int
    expires_at: _timestamp_pb2.Timestamp
    execution_mode: str
    mandate_id: str
    def __init__(self, id: _Optional[str] = ..., budget_id: _Optional[str] = ..., destination: _Optional[str] = ..., atomic_amount: _Optional[str] = ..., network: _Optional[str] = ..., max_fee_wei: _Optional[str] = ..., value_minor: _Optional[int] = ..., fee_valuation_minor: _Optional[int] = ..., currency: _Optional[str] = ..., estimated_seconds: _Optional[int] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., execution_mode: _Optional[str] = ..., mandate_id: _Optional[str] = ...) -> None: ...

class WithdrawalQuoteResponse(_message.Message):
    __slots__ = ("quote",)
    QUOTE_FIELD_NUMBER: _ClassVar[int]
    quote: WithdrawalQuote
    def __init__(self, quote: _Optional[_Union[WithdrawalQuote, _Mapping]] = ...) -> None: ...

class ExecuteWithdrawalRequest(_message.Message):
    __slots__ = ("quote_id", "idempotency_key")
    QUOTE_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    quote_id: str
    idempotency_key: str
    def __init__(self, quote_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class GetWithdrawalRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class Withdrawal(_message.Message):
    __slots__ = ("id", "quote_id", "authorization_id", "settlement_id", "state", "transaction_hash", "last_error")
    ID_FIELD_NUMBER: _ClassVar[int]
    QUOTE_ID_FIELD_NUMBER: _ClassVar[int]
    AUTHORIZATION_ID_FIELD_NUMBER: _ClassVar[int]
    SETTLEMENT_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    TRANSACTION_HASH_FIELD_NUMBER: _ClassVar[int]
    LAST_ERROR_FIELD_NUMBER: _ClassVar[int]
    id: str
    quote_id: str
    authorization_id: str
    settlement_id: str
    state: str
    transaction_hash: str
    last_error: str
    def __init__(self, id: _Optional[str] = ..., quote_id: _Optional[str] = ..., authorization_id: _Optional[str] = ..., settlement_id: _Optional[str] = ..., state: _Optional[str] = ..., transaction_hash: _Optional[str] = ..., last_error: _Optional[str] = ...) -> None: ...

class WithdrawalResponse(_message.Message):
    __slots__ = ("withdrawal",)
    WITHDRAWAL_FIELD_NUMBER: _ClassVar[int]
    withdrawal: Withdrawal
    def __init__(self, withdrawal: _Optional[_Union[Withdrawal, _Mapping]] = ...) -> None: ...
