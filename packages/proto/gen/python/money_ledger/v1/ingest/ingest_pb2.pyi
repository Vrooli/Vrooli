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
    __slots__ = ("network", "asset_id", "atomic_amount", "decimals", "source_reported_state", "leg_kind", "source_reference", "occurrence_key", "disposition", "posting_state", "budget_state", "holds", "book_account_status", "candidate_effect_atomic", "correction_requires_original_receipt", "address")
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
    def __init__(self, network: _Optional[str] = ..., asset_id: _Optional[str] = ..., atomic_amount: _Optional[str] = ..., decimals: _Optional[int] = ..., source_reported_state: _Optional[_Union[CryptoObservationState, str]] = ..., leg_kind: _Optional[_Union[CryptoLegKind, str]] = ..., source_reference: _Optional[str] = ..., occurrence_key: _Optional[str] = ..., disposition: _Optional[str] = ..., posting_state: _Optional[str] = ..., budget_state: _Optional[str] = ..., holds: _Optional[_Iterable[str]] = ..., book_account_status: _Optional[str] = ..., candidate_effect_atomic: _Optional[str] = ..., correction_requires_original_receipt: _Optional[bool] = ..., address: _Optional[str] = ...) -> None: ...
