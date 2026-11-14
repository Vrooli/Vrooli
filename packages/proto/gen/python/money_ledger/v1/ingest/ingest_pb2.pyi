import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from money_ledger.v1.shared import ledger_types_pb2 as _ledger_types_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AdapterKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ADAPTER_KIND_UNSPECIFIED: _ClassVar[AdapterKind]
    ADAPTER_KIND_MANUAL: _ClassVar[AdapterKind]
    ADAPTER_KIND_FILE: _ClassVar[AdapterKind]
    ADAPTER_KIND_AGGREGATOR: _ClassVar[AdapterKind]
    ADAPTER_KIND_COMMERCE: _ClassVar[AdapterKind]

class SourceMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SOURCE_MODE_UNSPECIFIED: _ClassVar[SourceMode]
    SOURCE_MODE_FIXTURE: _ClassVar[SourceMode]
    SOURCE_MODE_OPERATOR_SUPPLIED: _ClassVar[SourceMode]

class CryptoObservationState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CRYPTO_OBSERVATION_STATE_UNSPECIFIED: _ClassVar[CryptoObservationState]
    CRYPTO_OBSERVATION_STATE_PENDING: _ClassVar[CryptoObservationState]
    CRYPTO_OBSERVATION_STATE_INCLUDED: _ClassVar[CryptoObservationState]
    CRYPTO_OBSERVATION_STATE_FINALIZED: _ClassVar[CryptoObservationState]
    CRYPTO_OBSERVATION_STATE_REMOVED: _ClassVar[CryptoObservationState]
    CRYPTO_OBSERVATION_STATE_REVERTED: _ClassVar[CryptoObservationState]

class CryptoLegKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CRYPTO_LEG_KIND_UNSPECIFIED: _ClassVar[CryptoLegKind]
    CRYPTO_LEG_KIND_BALANCE: _ClassVar[CryptoLegKind]
    CRYPTO_LEG_KIND_DEPOSIT: _ClassVar[CryptoLegKind]
    CRYPTO_LEG_KIND_FEE: _ClassVar[CryptoLegKind]
    CRYPTO_LEG_KIND_APPROVAL: _ClassVar[CryptoLegKind]

class CryptoReceiptAction(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CRYPTO_RECEIPT_ACTION_UNSPECIFIED: _ClassVar[CryptoReceiptAction]
    CRYPTO_RECEIPT_ACTION_ORIGINAL: _ClassVar[CryptoReceiptAction]
    CRYPTO_RECEIPT_ACTION_REINCLUSION: _ClassVar[CryptoReceiptAction]
    CRYPTO_RECEIPT_ACTION_REMOVAL: _ClassVar[CryptoReceiptAction]
    CRYPTO_RECEIPT_ACTION_CORRECTION: _ClassVar[CryptoReceiptAction]
ADAPTER_KIND_UNSPECIFIED: AdapterKind
ADAPTER_KIND_MANUAL: AdapterKind
ADAPTER_KIND_FILE: AdapterKind
ADAPTER_KIND_AGGREGATOR: AdapterKind
ADAPTER_KIND_COMMERCE: AdapterKind
SOURCE_MODE_UNSPECIFIED: SourceMode
SOURCE_MODE_FIXTURE: SourceMode
SOURCE_MODE_OPERATOR_SUPPLIED: SourceMode
CRYPTO_OBSERVATION_STATE_UNSPECIFIED: CryptoObservationState
CRYPTO_OBSERVATION_STATE_PENDING: CryptoObservationState
CRYPTO_OBSERVATION_STATE_INCLUDED: CryptoObservationState
CRYPTO_OBSERVATION_STATE_FINALIZED: CryptoObservationState
CRYPTO_OBSERVATION_STATE_REMOVED: CryptoObservationState
CRYPTO_OBSERVATION_STATE_REVERTED: CryptoObservationState
CRYPTO_LEG_KIND_UNSPECIFIED: CryptoLegKind
CRYPTO_LEG_KIND_BALANCE: CryptoLegKind
CRYPTO_LEG_KIND_DEPOSIT: CryptoLegKind
CRYPTO_LEG_KIND_FEE: CryptoLegKind
CRYPTO_LEG_KIND_APPROVAL: CryptoLegKind
CRYPTO_RECEIPT_ACTION_UNSPECIFIED: CryptoReceiptAction
CRYPTO_RECEIPT_ACTION_ORIGINAL: CryptoReceiptAction
CRYPTO_RECEIPT_ACTION_REINCLUSION: CryptoReceiptAction
CRYPTO_RECEIPT_ACTION_REMOVAL: CryptoReceiptAction
CRYPTO_RECEIPT_ACTION_CORRECTION: CryptoReceiptAction

class Adapter(_message.Message):
    __slots__ = ("id", "name", "kind", "enabled", "last_success_at", "availability_reason", "cursor")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    LAST_SUCCESS_AT_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_REASON_FIELD_NUMBER: _ClassVar[int]
    CURSOR_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    kind: AdapterKind
    enabled: bool
    last_success_at: _timestamp_pb2.Timestamp
    availability_reason: str
    cursor: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., kind: _Optional[_Union[AdapterKind, str]] = ..., enabled: _Optional[bool] = ..., last_success_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., availability_reason: _Optional[str] = ..., cursor: _Optional[str] = ...) -> None: ...

class Receipt(_message.Message):
    __slots__ = ("id", "adapter_id", "to", "read", "written", "skipped_duplicates", "status", "error", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_ID_FIELD_NUMBER: _ClassVar[int]
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    READ_FIELD_NUMBER: _ClassVar[int]
    WRITTEN_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_DUPLICATES_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    adapter_id: str
    to: _timestamp_pb2.Timestamp
    read: int
    written: int
    skipped_duplicates: int
    status: str
    error: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., adapter_id: _Optional[str] = ..., to: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., read: _Optional[int] = ..., written: _Optional[int] = ..., skipped_duplicates: _Optional[int] = ..., status: _Optional[str] = ..., error: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., **kwargs) -> None: ...

class Availability(_message.Message):
    __slots__ = ("adapter_id", "reason", "last_success_at")
    ADAPTER_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    LAST_SUCCESS_AT_FIELD_NUMBER: _ClassVar[int]
    adapter_id: str
    reason: str
    last_success_at: _timestamp_pb2.Timestamp
    def __init__(self, adapter_id: _Optional[str] = ..., reason: _Optional[str] = ..., last_success_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RegisterAdapterRequest(_message.Message):
    __slots__ = ("adapter",)
    ADAPTER_FIELD_NUMBER: _ClassVar[int]
    adapter: Adapter
    def __init__(self, adapter: _Optional[_Union[Adapter, _Mapping]] = ...) -> None: ...

class RegisterAdapterResponse(_message.Message):
    __slots__ = ("adapter",)
    ADAPTER_FIELD_NUMBER: _ClassVar[int]
    adapter: Adapter
    def __init__(self, adapter: _Optional[_Union[Adapter, _Mapping]] = ...) -> None: ...

class ListAdaptersRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListAdaptersResponse(_message.Message):
    __slots__ = ("adapters",)
    ADAPTERS_FIELD_NUMBER: _ClassVar[int]
    adapters: _containers.RepeatedCompositeFieldContainer[Adapter]
    def __init__(self, adapters: _Optional[_Iterable[_Union[Adapter, _Mapping]]] = ...) -> None: ...

class IngestEventRequest(_message.Message):
    __slots__ = ("event",)
    EVENT_FIELD_NUMBER: _ClassVar[int]
    event: _ledger_types_pb2.MoneyEvent
    def __init__(self, event: _Optional[_Union[_ledger_types_pb2.MoneyEvent, _Mapping]] = ...) -> None: ...

class IngestEventResponse(_message.Message):
    __slots__ = ("posting", "duplicate", "receipt")
    POSTING_FIELD_NUMBER: _ClassVar[int]
    DUPLICATE_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    posting: _ledger_types_pb2.Posting
    duplicate: bool
    receipt: Receipt
    def __init__(self, posting: _Optional[_Union[_ledger_types_pb2.Posting, _Mapping]] = ..., duplicate: _Optional[bool] = ..., receipt: _Optional[_Union[Receipt, _Mapping]] = ...) -> None: ...

class RunAdapterRequest(_message.Message):
    __slots__ = ("adapter_id", "to")
    ADAPTER_ID_FIELD_NUMBER: _ClassVar[int]
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    adapter_id: str
    to: _timestamp_pb2.Timestamp
    def __init__(self, adapter_id: _Optional[str] = ..., to: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., **kwargs) -> None: ...

class RunAdapterResponse(_message.Message):
    __slots__ = ("receipt", "availability")
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    AVAILABILITY_FIELD_NUMBER: _ClassVar[int]
    receipt: Receipt
    availability: _containers.RepeatedCompositeFieldContainer[Availability]
    def __init__(self, receipt: _Optional[_Union[Receipt, _Mapping]] = ..., availability: _Optional[_Iterable[_Union[Availability, _Mapping]]] = ...) -> None: ...

class ImportFileRequest(_message.Message):
    __slots__ = ("adapter_id", "csv", "to")
    ADAPTER_ID_FIELD_NUMBER: _ClassVar[int]
    CSV_FIELD_NUMBER: _ClassVar[int]
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    adapter_id: str
    csv: bytes
    to: _timestamp_pb2.Timestamp
    def __init__(self, adapter_id: _Optional[str] = ..., csv: _Optional[bytes] = ..., to: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., **kwargs) -> None: ...

class ImportFileResponse(_message.Message):
    __slots__ = ("receipt",)
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    receipt: Receipt
    def __init__(self, receipt: _Optional[_Union[Receipt, _Mapping]] = ...) -> None: ...

class OperatorInputField(_message.Message):
    __slots__ = ("path", "status", "written", "unit", "window_days", "observed_at", "kind", "reason")
    PATH_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    WRITTEN_FIELD_NUMBER: _ClassVar[int]
    UNIT_FIELD_NUMBER: _ClassVar[int]
    WINDOW_DAYS_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    path: str
    status: str
    written: bool
    unit: str
    window_days: int
    observed_at: _timestamp_pb2.Timestamp
    kind: str
    reason: str
    def __init__(self, path: _Optional[str] = ..., status: _Optional[str] = ..., written: _Optional[bool] = ..., unit: _Optional[str] = ..., window_days: _Optional[int] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., kind: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class OperatorImportRequest(_message.Message):
    __slots__ = ("source_path", "source_mode", "apply", "adapter_id", "book_id", "account_id", "source_json")
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    SOURCE_MODE_FIELD_NUMBER: _ClassVar[int]
    APPLY_FIELD_NUMBER: _ClassVar[int]
    ADAPTER_ID_FIELD_NUMBER: _ClassVar[int]
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_JSON_FIELD_NUMBER: _ClassVar[int]
    source_path: str
    source_mode: SourceMode
    apply: bool
    adapter_id: str
    book_id: str
    account_id: str
    source_json: bytes
    def __init__(self, source_path: _Optional[str] = ..., source_mode: _Optional[_Union[SourceMode, str]] = ..., apply: _Optional[bool] = ..., adapter_id: _Optional[str] = ..., book_id: _Optional[str] = ..., account_id: _Optional[str] = ..., source_json: _Optional[bytes] = ...) -> None: ...

class OperatorImportResponse(_message.Message):
    __slots__ = ("fields", "read", "written", "findings", "applied")
    FIELDS_FIELD_NUMBER: _ClassVar[int]
    READ_FIELD_NUMBER: _ClassVar[int]
    WRITTEN_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    fields: _containers.RepeatedCompositeFieldContainer[OperatorInputField]
    read: int
    written: int
    findings: int
    applied: bool
    def __init__(self, fields: _Optional[_Iterable[_Union[OperatorInputField, _Mapping]]] = ..., read: _Optional[int] = ..., written: _Optional[int] = ..., findings: _Optional[int] = ..., applied: _Optional[bool] = ...) -> None: ...

class OperatorInputStatusRequest(_message.Message):
    __slots__ = ("book_id",)
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    book_id: str
    def __init__(self, book_id: _Optional[str] = ...) -> None: ...

class OperatorInputStatusResponse(_message.Message):
    __slots__ = ("fields",)
    FIELDS_FIELD_NUMBER: _ClassVar[int]
    fields: _containers.RepeatedCompositeFieldContainer[OperatorInputField]
    def __init__(self, fields: _Optional[_Iterable[_Union[OperatorInputField, _Mapping]]] = ...) -> None: ...

class PreviewCryptoObservationRequest(_message.Message):
    __slots__ = ("network", "asset_id", "atomic_amount", "decimals", "state", "leg_kind", "source_reference", "transaction_hash", "block_hash", "leg_index", "book_id", "account_id", "address")
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    ATOMIC_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    DECIMALS_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    LEG_KIND_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    TRANSACTION_HASH_FIELD_NUMBER: _ClassVar[int]
    BLOCK_HASH_FIELD_NUMBER: _ClassVar[int]
    LEG_INDEX_FIELD_NUMBER: _ClassVar[int]
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_ID_FIELD_NUMBER: _ClassVar[int]
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    network: str
    asset_id: str
    atomic_amount: str
    decimals: int
    state: CryptoObservationState
    leg_kind: CryptoLegKind
    source_reference: str
    transaction_hash: str
    block_hash: str
    leg_index: int
    book_id: str
    account_id: str
    address: str
    def __init__(self, network: _Optional[str] = ..., asset_id: _Optional[str] = ..., atomic_amount: _Optional[str] = ..., decimals: _Optional[int] = ..., state: _Optional[_Union[CryptoObservationState, str]] = ..., leg_kind: _Optional[_Union[CryptoLegKind, str]] = ..., source_reference: _Optional[str] = ..., transaction_hash: _Optional[str] = ..., block_hash: _Optional[str] = ..., leg_index: _Optional[int] = ..., book_id: _Optional[str] = ..., account_id: _Optional[str] = ..., address: _Optional[str] = ...) -> None: ...

class PreviewCryptoObservationResponse(_message.Message):
    __slots__ = ("network", "asset_id", "atomic_amount", "decimals", "source_reported_state", "leg_kind", "source_reference", "occurrence_key", "disposition", "posting_state", "budget_state", "holds", "book_account_status", "candidate_effect_atomic", "correction_requires_original_receipt", "address", "inclusion_revision")
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    ATOMIC_AMOUNT_FIELD_NUMBER: _ClassVar[int]
    DECIMALS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REPORTED_STATE_FIELD_NUMBER: _ClassVar[int]
    LEG_KIND_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    OCCURRENCE_KEY_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    POSTING_STATE_FIELD_NUMBER: _ClassVar[int]
    BUDGET_STATE_FIELD_NUMBER: _ClassVar[int]
    HOLDS_FIELD_NUMBER: _ClassVar[int]
    BOOK_ACCOUNT_STATUS_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_EFFECT_ATOMIC_FIELD_NUMBER: _ClassVar[int]
    CORRECTION_REQUIRES_ORIGINAL_RECEIPT_FIELD_NUMBER: _ClassVar[int]
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    INCLUSION_REVISION_FIELD_NUMBER: _ClassVar[int]
    network: str
    asset_id: str
    atomic_amount: str
    decimals: int
    source_reported_state: CryptoObservationState
    leg_kind: CryptoLegKind
    source_reference: str
    occurrence_key: str
    disposition: str
    posting_state: str
    budget_state: str
    holds: _containers.RepeatedScalarFieldContainer[str]
    book_account_status: str
    candidate_effect_atomic: str
    correction_requires_original_receipt: bool
    address: str
    inclusion_revision: str
    def __init__(self, network: _Optional[str] = ..., asset_id: _Optional[str] = ..., atomic_amount: _Optional[str] = ..., decimals: _Optional[int] = ..., source_reported_state: _Optional[_Union[CryptoObservationState, str]] = ..., leg_kind: _Optional[_Union[CryptoLegKind, str]] = ..., source_reference: _Optional[str] = ..., occurrence_key: _Optional[str] = ..., disposition: _Optional[str] = ..., posting_state: _Optional[str] = ..., budget_state: _Optional[str] = ..., holds: _Optional[_Iterable[str]] = ..., book_account_status: _Optional[str] = ..., candidate_effect_atomic: _Optional[str] = ..., correction_requires_original_receipt: _Optional[bool] = ..., address: _Optional[str] = ..., inclusion_revision: _Optional[str] = ...) -> None: ...

class TreasuryReceiptLink(_message.Message):
    __slots__ = ("check_id", "wallet_id", "operator_identity", "source_reference")
    CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    WALLET_ID_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    check_id: str
    wallet_id: str
    operator_identity: str
    source_reference: str
    def __init__(self, check_id: _Optional[str] = ..., wallet_id: _Optional[str] = ..., operator_identity: _Optional[str] = ..., source_reference: _Optional[str] = ...) -> None: ...

class CryptoValuation(_message.Message):
    __slots__ = ("amount_minor", "currency", "basis_reference", "valued_at", "basis")
    AMOUNT_MINOR_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    BASIS_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    VALUED_AT_FIELD_NUMBER: _ClassVar[int]
    BASIS_FIELD_NUMBER: _ClassVar[int]
    amount_minor: int
    currency: str
    basis_reference: str
    valued_at: _timestamp_pb2.Timestamp
    basis: _ledger_types_pb2.Basis
    def __init__(self, amount_minor: _Optional[int] = ..., currency: _Optional[str] = ..., basis_reference: _Optional[str] = ..., valued_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., basis: _Optional[_Union[_ledger_types_pb2.Basis, str]] = ...) -> None: ...

class CryptoPayerAttribution(_message.Message):
    __slots__ = ("payer_address", "payer_identity", "credit_lot_id", "credit_lot_owner")
    PAYER_ADDRESS_FIELD_NUMBER: _ClassVar[int]
    PAYER_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    CREDIT_LOT_ID_FIELD_NUMBER: _ClassVar[int]
    CREDIT_LOT_OWNER_FIELD_NUMBER: _ClassVar[int]
    payer_address: str
    payer_identity: str
    credit_lot_id: str
    credit_lot_owner: str
    def __init__(self, payer_address: _Optional[str] = ..., payer_identity: _Optional[str] = ..., credit_lot_id: _Optional[str] = ..., credit_lot_owner: _Optional[str] = ...) -> None: ...

class AcceptCryptoReceiptRequest(_message.Message):
    __slots__ = ("observation", "source", "action", "original_receipt_id", "valuation", "attribution", "occurred_at", "correction_reason", "previous_receipt_id")
    OBSERVATION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    ORIGINAL_RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    VALUATION_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    OCCURRED_AT_FIELD_NUMBER: _ClassVar[int]
    CORRECTION_REASON_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    observation: PreviewCryptoObservationRequest
    source: TreasuryReceiptLink
    action: CryptoReceiptAction
    original_receipt_id: str
    valuation: CryptoValuation
    attribution: CryptoPayerAttribution
    occurred_at: _timestamp_pb2.Timestamp
    correction_reason: str
    previous_receipt_id: str
    def __init__(self, observation: _Optional[_Union[PreviewCryptoObservationRequest, _Mapping]] = ..., source: _Optional[_Union[TreasuryReceiptLink, _Mapping]] = ..., action: _Optional[_Union[CryptoReceiptAction, str]] = ..., original_receipt_id: _Optional[str] = ..., valuation: _Optional[_Union[CryptoValuation, _Mapping]] = ..., attribution: _Optional[_Union[CryptoPayerAttribution, _Mapping]] = ..., occurred_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., correction_reason: _Optional[str] = ..., previous_receipt_id: _Optional[str] = ...) -> None: ...

class AcceptedCryptoReceipt(_message.Message):
    __slots__ = ("id", "posting_id", "source", "occurrence_key", "inclusion_revision", "atomic_delta", "valuation_delta_minor", "currency", "accepted_at", "original_receipt_id", "action", "attribution", "no_chargeback", "previous_receipt_id", "observation", "valuation", "occurred_at", "correction_reason", "accepted_by")
    ID_FIELD_NUMBER: _ClassVar[int]
    POSTING_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    OCCURRENCE_KEY_FIELD_NUMBER: _ClassVar[int]
    INCLUSION_REVISION_FIELD_NUMBER: _ClassVar[int]
    ATOMIC_DELTA_FIELD_NUMBER: _ClassVar[int]
    VALUATION_DELTA_MINOR_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_AT_FIELD_NUMBER: _ClassVar[int]
    ORIGINAL_RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTION_FIELD_NUMBER: _ClassVar[int]
    NO_CHARGEBACK_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    OBSERVATION_FIELD_NUMBER: _ClassVar[int]
    VALUATION_FIELD_NUMBER: _ClassVar[int]
    OCCURRED_AT_FIELD_NUMBER: _ClassVar[int]
    CORRECTION_REASON_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_BY_FIELD_NUMBER: _ClassVar[int]
    id: str
    posting_id: str
    source: TreasuryReceiptLink
    occurrence_key: str
    inclusion_revision: str
    atomic_delta: str
    valuation_delta_minor: int
    currency: str
    accepted_at: _timestamp_pb2.Timestamp
    original_receipt_id: str
    action: CryptoReceiptAction
    attribution: CryptoPayerAttribution
    no_chargeback: bool
    previous_receipt_id: str
    observation: PreviewCryptoObservationRequest
    valuation: CryptoValuation
    occurred_at: _timestamp_pb2.Timestamp
    correction_reason: str
    accepted_by: str
    def __init__(self, id: _Optional[str] = ..., posting_id: _Optional[str] = ..., source: _Optional[_Union[TreasuryReceiptLink, _Mapping]] = ..., occurrence_key: _Optional[str] = ..., inclusion_revision: _Optional[str] = ..., atomic_delta: _Optional[str] = ..., valuation_delta_minor: _Optional[int] = ..., currency: _Optional[str] = ..., accepted_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., original_receipt_id: _Optional[str] = ..., action: _Optional[_Union[CryptoReceiptAction, str]] = ..., attribution: _Optional[_Union[CryptoPayerAttribution, _Mapping]] = ..., no_chargeback: _Optional[bool] = ..., previous_receipt_id: _Optional[str] = ..., observation: _Optional[_Union[PreviewCryptoObservationRequest, _Mapping]] = ..., valuation: _Optional[_Union[CryptoValuation, _Mapping]] = ..., occurred_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., correction_reason: _Optional[str] = ..., accepted_by: _Optional[str] = ...) -> None: ...

class AcceptCryptoReceiptResponse(_message.Message):
    __slots__ = ("receipt", "duplicate")
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    DUPLICATE_FIELD_NUMBER: _ClassVar[int]
    receipt: AcceptedCryptoReceipt
    duplicate: bool
    def __init__(self, receipt: _Optional[_Union[AcceptedCryptoReceipt, _Mapping]] = ..., duplicate: _Optional[bool] = ...) -> None: ...

class GetCryptoReceiptRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetCryptoReceiptResponse(_message.Message):
    __slots__ = ("receipt",)
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    receipt: AcceptedCryptoReceipt
    def __init__(self, receipt: _Optional[_Union[AcceptedCryptoReceipt, _Mapping]] = ...) -> None: ...

class ListCryptoReceiptsRequest(_message.Message):
    __slots__ = ("book_id", "wallet_id", "credit_lot_id", "limit", "after_receipt_id")
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    WALLET_ID_FIELD_NUMBER: _ClassVar[int]
    CREDIT_LOT_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    AFTER_RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    book_id: str
    wallet_id: str
    credit_lot_id: str
    limit: int
    after_receipt_id: str
    def __init__(self, book_id: _Optional[str] = ..., wallet_id: _Optional[str] = ..., credit_lot_id: _Optional[str] = ..., limit: _Optional[int] = ..., after_receipt_id: _Optional[str] = ...) -> None: ...

class ListCryptoReceiptsResponse(_message.Message):
    __slots__ = ("receipts", "next_after_receipt_id")
    RECEIPTS_FIELD_NUMBER: _ClassVar[int]
    NEXT_AFTER_RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    receipts: _containers.RepeatedCompositeFieldContainer[AcceptedCryptoReceipt]
    next_after_receipt_id: str
    def __init__(self, receipts: _Optional[_Iterable[_Union[AcceptedCryptoReceipt, _Mapping]]] = ..., next_after_receipt_id: _Optional[str] = ...) -> None: ...
