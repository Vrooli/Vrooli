import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Chain(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CHAIN_UNSPECIFIED: _ClassVar[Chain]
    CHAIN_BITCOIN: _ClassVar[Chain]
    CHAIN_ETHEREUM: _ClassVar[Chain]
    CHAIN_BASE: _ClassVar[Chain]
    CHAIN_SOLANA: _ClassVar[Chain]
    CHAIN_ARBITRUM: _ClassVar[Chain]

class EvidenceKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EVIDENCE_KIND_UNSPECIFIED: _ClassVar[EvidenceKind]
    EVIDENCE_KIND_OBSERVATION: _ClassVar[EvidenceKind]
    EVIDENCE_KIND_LEGAL_CLAIM: _ClassVar[EvidenceKind]
    EVIDENCE_KIND_PUBLIC_LABEL: _ClassVar[EvidenceKind]
    EVIDENCE_KIND_DERIVATION: _ClassVar[EvidenceKind]

class EvidenceGrade(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EVIDENCE_GRADE_UNSPECIFIED: _ClassVar[EvidenceGrade]
    EVIDENCE_GRADE_OBSERVED: _ClassVar[EvidenceGrade]
    EVIDENCE_GRADE_PROTOCOL_CORRELATED: _ClassVar[EvidenceGrade]
    EVIDENCE_GRADE_SOURCE_CORROBORATED: _ClassVar[EvidenceGrade]
    EVIDENCE_GRADE_HYPOTHESIS: _ClassVar[EvidenceGrade]
    EVIDENCE_GRADE_UNRESOLVED: _ClassVar[EvidenceGrade]
CHAIN_UNSPECIFIED: Chain
CHAIN_BITCOIN: Chain
CHAIN_ETHEREUM: Chain
CHAIN_BASE: Chain
CHAIN_SOLANA: Chain
CHAIN_ARBITRUM: Chain
EVIDENCE_KIND_UNSPECIFIED: EvidenceKind
EVIDENCE_KIND_OBSERVATION: EvidenceKind
EVIDENCE_KIND_LEGAL_CLAIM: EvidenceKind
EVIDENCE_KIND_PUBLIC_LABEL: EvidenceKind
EVIDENCE_KIND_DERIVATION: EvidenceKind
EVIDENCE_GRADE_UNSPECIFIED: EvidenceGrade
EVIDENCE_GRADE_OBSERVED: EvidenceGrade
EVIDENCE_GRADE_PROTOCOL_CORRELATED: EvidenceGrade
EVIDENCE_GRADE_SOURCE_CORROBORATED: EvidenceGrade
EVIDENCE_GRADE_HYPOTHESIS: EvidenceGrade
EVIDENCE_GRADE_UNRESOLVED: EvidenceGrade

class Source(_message.Message):
    __slots__ = ("id", "name", "url", "retrieved_at", "sha256", "citation", "limitations", "result_body", "raw_response_body", "raw_response_sha256", "selected_by_source_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    RETRIEVED_AT_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    CITATION_FIELD_NUMBER: _ClassVar[int]
    LIMITATIONS_FIELD_NUMBER: _ClassVar[int]
    RESULT_BODY_FIELD_NUMBER: _ClassVar[int]
    RAW_RESPONSE_BODY_FIELD_NUMBER: _ClassVar[int]
    RAW_RESPONSE_SHA256_FIELD_NUMBER: _ClassVar[int]
    SELECTED_BY_SOURCE_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    url: str
    retrieved_at: str
    sha256: str
    citation: str
    limitations: str
    result_body: bytes
    raw_response_body: bytes
    raw_response_sha256: str
    selected_by_source_id: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., url: _Optional[str] = ..., retrieved_at: _Optional[str] = ..., sha256: _Optional[str] = ..., citation: _Optional[str] = ..., limitations: _Optional[str] = ..., result_body: _Optional[bytes] = ..., raw_response_body: _Optional[bytes] = ..., raw_response_sha256: _Optional[str] = ..., selected_by_source_id: _Optional[str] = ...) -> None: ...

class Claim(_message.Message):
    __slots__ = ("id", "text", "source_id", "kind")
    ID_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    id: str
    text: str
    source_id: str
    kind: EvidenceKind
    def __init__(self, id: _Optional[str] = ..., text: _Optional[str] = ..., source_id: _Optional[str] = ..., kind: _Optional[_Union[EvidenceKind, str]] = ...) -> None: ...

class Occurrence(_message.Message):
    __slots__ = ("id", "chain", "transaction_id", "block_height", "timestamp", "event_index", "direction", "previous_transaction_id", "previous_output_index", "to", "asset", "atomic_quantity", "scale", "kind", "source_id", "measurement", "account")
    ID_FIELD_NUMBER: _ClassVar[int]
    CHAIN_FIELD_NUMBER: _ClassVar[int]
    TRANSACTION_ID_FIELD_NUMBER: _ClassVar[int]
    BLOCK_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    EVENT_INDEX_FIELD_NUMBER: _ClassVar[int]
    DIRECTION_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_TRANSACTION_ID_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_OUTPUT_INDEX_FIELD_NUMBER: _ClassVar[int]
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    ASSET_FIELD_NUMBER: _ClassVar[int]
    ATOMIC_QUANTITY_FIELD_NUMBER: _ClassVar[int]
    SCALE_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ID_FIELD_NUMBER: _ClassVar[int]
    MEASUREMENT_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_FIELD_NUMBER: _ClassVar[int]
    id: str
    chain: Chain
    transaction_id: str
    block_height: int
    timestamp: str
    event_index: int
    direction: str
    previous_transaction_id: str
    previous_output_index: int
    to: str
    asset: str
    atomic_quantity: str
    scale: int
    kind: EvidenceKind
    source_id: str
    measurement: str
    account: str
    def __init__(self, id: _Optional[str] = ..., chain: _Optional[_Union[Chain, str]] = ..., transaction_id: _Optional[str] = ..., block_height: _Optional[int] = ..., timestamp: _Optional[str] = ..., event_index: _Optional[int] = ..., direction: _Optional[str] = ..., previous_transaction_id: _Optional[str] = ..., previous_output_index: _Optional[int] = ..., to: _Optional[str] = ..., asset: _Optional[str] = ..., atomic_quantity: _Optional[str] = ..., scale: _Optional[int] = ..., kind: _Optional[_Union[EvidenceKind, str]] = ..., source_id: _Optional[str] = ..., measurement: _Optional[str] = ..., account: _Optional[str] = ..., **kwargs) -> None: ...

class FlowEdge(_message.Message):
    __slots__ = ("id", "from_occurrence", "to_occurrence", "grade", "rule", "evidence_ids", "caveats")
    ID_FIELD_NUMBER: _ClassVar[int]
    FROM_OCCURRENCE_FIELD_NUMBER: _ClassVar[int]
    TO_OCCURRENCE_FIELD_NUMBER: _ClassVar[int]
    GRADE_FIELD_NUMBER: _ClassVar[int]
    RULE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_IDS_FIELD_NUMBER: _ClassVar[int]
    CAVEATS_FIELD_NUMBER: _ClassVar[int]
    id: str
    from_occurrence: str
    to_occurrence: str
    grade: EvidenceGrade
    rule: str
    evidence_ids: _containers.RepeatedScalarFieldContainer[str]
    caveats: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., from_occurrence: _Optional[str] = ..., to_occurrence: _Optional[str] = ..., grade: _Optional[_Union[EvidenceGrade, str]] = ..., rule: _Optional[str] = ..., evidence_ids: _Optional[_Iterable[str]] = ..., caveats: _Optional[_Iterable[str]] = ...) -> None: ...

class EvidenceCase(_message.Message):
    __slots__ = ("id", "title", "summary", "private", "sources", "claims", "occurrences", "edges", "gaps", "counterexamples", "source_disagreements", "created_at", "updated_at", "wallet_snapshots", "pump_instructions", "unresolved_pump_instructions", "solana_token_account_snapshots")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    PRIVATE_FIELD_NUMBER: _ClassVar[int]
    SOURCES_FIELD_NUMBER: _ClassVar[int]
    CLAIMS_FIELD_NUMBER: _ClassVar[int]
    OCCURRENCES_FIELD_NUMBER: _ClassVar[int]
    EDGES_FIELD_NUMBER: _ClassVar[int]
    GAPS_FIELD_NUMBER: _ClassVar[int]
    COUNTEREXAMPLES_FIELD_NUMBER: _ClassVar[int]
    SOURCE_DISAGREEMENTS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    WALLET_SNAPSHOTS_FIELD_NUMBER: _ClassVar[int]
    PUMP_INSTRUCTIONS_FIELD_NUMBER: _ClassVar[int]
    UNRESOLVED_PUMP_INSTRUCTIONS_FIELD_NUMBER: _ClassVar[int]
    SOLANA_TOKEN_ACCOUNT_SNAPSHOTS_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    summary: str
    private: bool
    sources: _containers.RepeatedCompositeFieldContainer[Source]
    claims: _containers.RepeatedCompositeFieldContainer[Claim]
    occurrences: _containers.RepeatedCompositeFieldContainer[Occurrence]
    edges: _containers.RepeatedCompositeFieldContainer[FlowEdge]
    gaps: _containers.RepeatedScalarFieldContainer[str]
    counterexamples: _containers.RepeatedScalarFieldContainer[str]
    source_disagreements: _containers.RepeatedScalarFieldContainer[str]
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    wallet_snapshots: _containers.RepeatedCompositeFieldContainer[EVMWalletSnapshot]
    pump_instructions: _containers.RepeatedCompositeFieldContainer[PumpInstruction]
    unresolved_pump_instructions: _containers.RepeatedCompositeFieldContainer[UnresolvedPumpInstruction]
    solana_token_account_snapshots: _containers.RepeatedCompositeFieldContainer[SolanaTokenAccountSnapshot]
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., summary: _Optional[str] = ..., private: _Optional[bool] = ..., sources: _Optional[_Iterable[_Union[Source, _Mapping]]] = ..., claims: _Optional[_Iterable[_Union[Claim, _Mapping]]] = ..., occurrences: _Optional[_Iterable[_Union[Occurrence, _Mapping]]] = ..., edges: _Optional[_Iterable[_Union[FlowEdge, _Mapping]]] = ..., gaps: _Optional[_Iterable[str]] = ..., counterexamples: _Optional[_Iterable[str]] = ..., source_disagreements: _Optional[_Iterable[str]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., wallet_snapshots: _Optional[_Iterable[_Union[EVMWalletSnapshot, _Mapping]]] = ..., pump_instructions: _Optional[_Iterable[_Union[PumpInstruction, _Mapping]]] = ..., unresolved_pump_instructions: _Optional[_Iterable[_Union[UnresolvedPumpInstruction, _Mapping]]] = ..., solana_token_account_snapshots: _Optional[_Iterable[_Union[SolanaTokenAccountSnapshot, _Mapping]]] = ...) -> None: ...

class SolanaTokenAccount(_message.Message):
    __slots__ = ("address", "mint", "owner", "program_id", "atomic_quantity", "scale", "scale_known")
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    MINT_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    PROGRAM_ID_FIELD_NUMBER: _ClassVar[int]
    ATOMIC_QUANTITY_FIELD_NUMBER: _ClassVar[int]
    SCALE_FIELD_NUMBER: _ClassVar[int]
    SCALE_KNOWN_FIELD_NUMBER: _ClassVar[int]
    address: str
    mint: str
    owner: str
    program_id: str
    atomic_quantity: str
    scale: int
    scale_known: bool
    def __init__(self, address: _Optional[str] = ..., mint: _Optional[str] = ..., owner: _Optional[str] = ..., program_id: _Optional[str] = ..., atomic_quantity: _Optional[str] = ..., scale: _Optional[int] = ..., scale_known: _Optional[bool] = ...) -> None: ...

class SolanaTokenAccountSnapshot(_message.Message):
    __slots__ = ("id", "owner", "program_id", "context_slot", "source_id", "accounts")
    ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    PROGRAM_ID_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_SLOT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ID_FIELD_NUMBER: _ClassVar[int]
    ACCOUNTS_FIELD_NUMBER: _ClassVar[int]
    id: str
    owner: str
    program_id: str
    context_slot: int
    source_id: str
    accounts: _containers.RepeatedCompositeFieldContainer[SolanaTokenAccount]
    def __init__(self, id: _Optional[str] = ..., owner: _Optional[str] = ..., program_id: _Optional[str] = ..., context_slot: _Optional[int] = ..., source_id: _Optional[str] = ..., accounts: _Optional[_Iterable[_Union[SolanaTokenAccount, _Mapping]]] = ...) -> None: ...

class PumpInstruction(_message.Message):
    __slots__ = ("id", "transaction_id", "slot", "instruction_index", "program_id", "source_id", "kind", "decoder_version", "mint", "user", "creator", "name", "symbol", "metadata_uri", "mayhem_mode", "cashback_enabled", "creator_fee_bps", "holder_reward")
    ID_FIELD_NUMBER: _ClassVar[int]
    TRANSACTION_ID_FIELD_NUMBER: _ClassVar[int]
    SLOT_FIELD_NUMBER: _ClassVar[int]
    INSTRUCTION_INDEX_FIELD_NUMBER: _ClassVar[int]
    PROGRAM_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    DECODER_VERSION_FIELD_NUMBER: _ClassVar[int]
    MINT_FIELD_NUMBER: _ClassVar[int]
    USER_FIELD_NUMBER: _ClassVar[int]
    CREATOR_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SYMBOL_FIELD_NUMBER: _ClassVar[int]
    METADATA_URI_FIELD_NUMBER: _ClassVar[int]
    MAYHEM_MODE_FIELD_NUMBER: _ClassVar[int]
    CASHBACK_ENABLED_FIELD_NUMBER: _ClassVar[int]
    CREATOR_FEE_BPS_FIELD_NUMBER: _ClassVar[int]
    HOLDER_REWARD_FIELD_NUMBER: _ClassVar[int]
    id: str
    transaction_id: str
    slot: int
    instruction_index: int
    program_id: str
    source_id: str
    kind: str
    decoder_version: str
    mint: str
    user: str
    creator: str
    name: str
    symbol: str
    metadata_uri: str
    mayhem_mode: bool
    cashback_enabled: bool
    creator_fee_bps: int
    holder_reward: bool
    def __init__(self, id: _Optional[str] = ..., transaction_id: _Optional[str] = ..., slot: _Optional[int] = ..., instruction_index: _Optional[int] = ..., program_id: _Optional[str] = ..., source_id: _Optional[str] = ..., kind: _Optional[str] = ..., decoder_version: _Optional[str] = ..., mint: _Optional[str] = ..., user: _Optional[str] = ..., creator: _Optional[str] = ..., name: _Optional[str] = ..., symbol: _Optional[str] = ..., metadata_uri: _Optional[str] = ..., mayhem_mode: _Optional[bool] = ..., cashback_enabled: _Optional[bool] = ..., creator_fee_bps: _Optional[int] = ..., holder_reward: _Optional[bool] = ...) -> None: ...

class UnresolvedPumpInstruction(_message.Message):
    __slots__ = ("id", "transaction_id", "slot", "instruction_index", "program_id", "source_id", "discriminator", "reason", "candidate_kind", "decoder_version", "expected_data_length", "observed_data_length")
    ID_FIELD_NUMBER: _ClassVar[int]
    TRANSACTION_ID_FIELD_NUMBER: _ClassVar[int]
    SLOT_FIELD_NUMBER: _ClassVar[int]
    INSTRUCTION_INDEX_FIELD_NUMBER: _ClassVar[int]
    PROGRAM_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_ID_FIELD_NUMBER: _ClassVar[int]
    DISCRIMINATOR_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_KIND_FIELD_NUMBER: _ClassVar[int]
    DECODER_VERSION_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_DATA_LENGTH_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_DATA_LENGTH_FIELD_NUMBER: _ClassVar[int]
    id: str
    transaction_id: str
    slot: int
    instruction_index: int
    program_id: str
    source_id: str
    discriminator: str
    reason: str
    candidate_kind: str
    decoder_version: str
    expected_data_length: int
    observed_data_length: int
    def __init__(self, id: _Optional[str] = ..., transaction_id: _Optional[str] = ..., slot: _Optional[int] = ..., instruction_index: _Optional[int] = ..., program_id: _Optional[str] = ..., source_id: _Optional[str] = ..., discriminator: _Optional[str] = ..., reason: _Optional[str] = ..., candidate_kind: _Optional[str] = ..., decoder_version: _Optional[str] = ..., expected_data_length: _Optional[int] = ..., observed_data_length: _Optional[int] = ...) -> None: ...

class EVMTokenBalance(_message.Message):
    __slots__ = ("contract", "atomic_quantity", "scale", "scale_known")
    CONTRACT_FIELD_NUMBER: _ClassVar[int]
    ATOMIC_QUANTITY_FIELD_NUMBER: _ClassVar[int]
    SCALE_FIELD_NUMBER: _ClassVar[int]
    SCALE_KNOWN_FIELD_NUMBER: _ClassVar[int]
    contract: str
    atomic_quantity: str
    scale: int
    scale_known: bool
    def __init__(self, contract: _Optional[str] = ..., atomic_quantity: _Optional[str] = ..., scale: _Optional[int] = ..., scale_known: _Optional[bool] = ...) -> None: ...

class EVMWalletSnapshot(_message.Message):
    __slots__ = ("id", "chain", "address", "block_height", "block_hash", "timestamp", "native_asset", "native_atomic_quantity", "native_scale", "tokens", "source_ids")
    ID_FIELD_NUMBER: _ClassVar[int]
    CHAIN_FIELD_NUMBER: _ClassVar[int]
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    BLOCK_HEIGHT_FIELD_NUMBER: _ClassVar[int]
    BLOCK_HASH_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    NATIVE_ASSET_FIELD_NUMBER: _ClassVar[int]
    NATIVE_ATOMIC_QUANTITY_FIELD_NUMBER: _ClassVar[int]
    NATIVE_SCALE_FIELD_NUMBER: _ClassVar[int]
    TOKENS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_IDS_FIELD_NUMBER: _ClassVar[int]
    id: str
    chain: Chain
    address: str
    block_height: int
    block_hash: str
    timestamp: str
    native_asset: str
    native_atomic_quantity: str
    native_scale: int
    tokens: _containers.RepeatedCompositeFieldContainer[EVMTokenBalance]
    source_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., chain: _Optional[_Union[Chain, str]] = ..., address: _Optional[str] = ..., block_height: _Optional[int] = ..., block_hash: _Optional[str] = ..., timestamp: _Optional[str] = ..., native_asset: _Optional[str] = ..., native_atomic_quantity: _Optional[str] = ..., native_scale: _Optional[int] = ..., tokens: _Optional[_Iterable[_Union[EVMTokenBalance, _Mapping]]] = ..., source_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class CaseSummary(_message.Message):
    __slots__ = ("id", "title", "private", "source_count", "occurrence_count", "edge_count", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    PRIVATE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_COUNT_FIELD_NUMBER: _ClassVar[int]
    OCCURRENCE_COUNT_FIELD_NUMBER: _ClassVar[int]
    EDGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    private: bool
    source_count: int
    occurrence_count: int
    edge_count: int
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., private: _Optional[bool] = ..., source_count: _Optional[int] = ..., occurrence_count: _Optional[int] = ..., edge_count: _Optional[int] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListCasesRequest(_message.Message):
    __slots__ = ("limit",)
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    limit: int
    def __init__(self, limit: _Optional[int] = ...) -> None: ...

class ListCasesResponse(_message.Message):
    __slots__ = ("cases",)
    CASES_FIELD_NUMBER: _ClassVar[int]
    cases: _containers.RepeatedCompositeFieldContainer[CaseSummary]
    def __init__(self, cases: _Optional[_Iterable[_Union[CaseSummary, _Mapping]]] = ...) -> None: ...

class ImportCaseRequest(_message.Message):
    __slots__ = ("case",)
    CASE_FIELD_NUMBER: _ClassVar[int]
    case: EvidenceCase
    def __init__(self, case: _Optional[_Union[EvidenceCase, _Mapping]] = ...) -> None: ...

class CaseResponse(_message.Message):
    __slots__ = ("case",)
    CASE_FIELD_NUMBER: _ClassVar[int]
    case: EvidenceCase
    def __init__(self, case: _Optional[_Union[EvidenceCase, _Mapping]] = ...) -> None: ...

class GetCaseRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ExportPrivatePacketRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ExportPrivatePacketResponse(_message.Message):
    __slots__ = ("schema_version", "packet_json", "sha256")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    PACKET_JSON_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    packet_json: bytes
    sha256: str
    def __init__(self, schema_version: _Optional[str] = ..., packet_json: _Optional[bytes] = ..., sha256: _Optional[str] = ...) -> None: ...

class AnalystLabel(_message.Message):
    __slots__ = ("id", "case_id", "subject", "label", "source_url", "source_citation", "grade", "counter_explanation", "unresolved_gap", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    SOURCE_URL_FIELD_NUMBER: _ClassVar[int]
    SOURCE_CITATION_FIELD_NUMBER: _ClassVar[int]
    GRADE_FIELD_NUMBER: _ClassVar[int]
    COUNTER_EXPLANATION_FIELD_NUMBER: _ClassVar[int]
    UNRESOLVED_GAP_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    case_id: str
    subject: str
    label: str
    source_url: str
    source_citation: str
    grade: EvidenceGrade
    counter_explanation: str
    unresolved_gap: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., case_id: _Optional[str] = ..., subject: _Optional[str] = ..., label: _Optional[str] = ..., source_url: _Optional[str] = ..., source_citation: _Optional[str] = ..., grade: _Optional[_Union[EvidenceGrade, str]] = ..., counter_explanation: _Optional[str] = ..., unresolved_gap: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class AddAnalystLabelRequest(_message.Message):
    __slots__ = ("label",)
    LABEL_FIELD_NUMBER: _ClassVar[int]
    label: AnalystLabel
    def __init__(self, label: _Optional[_Union[AnalystLabel, _Mapping]] = ...) -> None: ...

class AnalystLabelResponse(_message.Message):
    __slots__ = ("label",)
    LABEL_FIELD_NUMBER: _ClassVar[int]
    label: AnalystLabel
    def __init__(self, label: _Optional[_Union[AnalystLabel, _Mapping]] = ...) -> None: ...

class ListAnalystLabelsRequest(_message.Message):
    __slots__ = ("case_id",)
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    def __init__(self, case_id: _Optional[str] = ...) -> None: ...

class ListAnalystLabelsResponse(_message.Message):
    __slots__ = ("labels",)
    LABELS_FIELD_NUMBER: _ClassVar[int]
    labels: _containers.RepeatedCompositeFieldContainer[AnalystLabel]
    def __init__(self, labels: _Optional[_Iterable[_Union[AnalystLabel, _Mapping]]] = ...) -> None: ...

class CaptureEVMTransactionRequest(_message.Message):
    __slots__ = ("case_id", "chain", "transaction_hash")
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    CHAIN_FIELD_NUMBER: _ClassVar[int]
    TRANSACTION_HASH_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    chain: Chain
    transaction_hash: str
    def __init__(self, case_id: _Optional[str] = ..., chain: _Optional[_Union[Chain, str]] = ..., transaction_hash: _Optional[str] = ...) -> None: ...

class CaptureEVMTransactionResponse(_message.Message):
    __slots__ = ("case", "captured_occurrence_count")
    CASE_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_OCCURRENCE_COUNT_FIELD_NUMBER: _ClassVar[int]
    case: EvidenceCase
    captured_occurrence_count: int
    def __init__(self, case: _Optional[_Union[EvidenceCase, _Mapping]] = ..., captured_occurrence_count: _Optional[int] = ...) -> None: ...

class CaptureBitcoinTransactionRequest(_message.Message):
    __slots__ = ("case_id", "transaction_id")
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    TRANSACTION_ID_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    transaction_id: str
    def __init__(self, case_id: _Optional[str] = ..., transaction_id: _Optional[str] = ...) -> None: ...

class CaptureBitcoinTransactionResponse(_message.Message):
    __slots__ = ("case", "captured_occurrence_count")
    CASE_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_OCCURRENCE_COUNT_FIELD_NUMBER: _ClassVar[int]
    case: EvidenceCase
    captured_occurrence_count: int
    def __init__(self, case: _Optional[_Union[EvidenceCase, _Mapping]] = ..., captured_occurrence_count: _Optional[int] = ...) -> None: ...

class CaptureBitcoinAddressPageRequest(_message.Message):
    __slots__ = ("case_id", "address", "after_txid")
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    AFTER_TXID_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    address: str
    after_txid: str
    def __init__(self, case_id: _Optional[str] = ..., address: _Optional[str] = ..., after_txid: _Optional[str] = ...) -> None: ...

class CaptureBitcoinAddressPageResponse(_message.Message):
    __slots__ = ("case", "captured_transaction_count", "captured_occurrence_count", "next_after_txid")
    CASE_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_TRANSACTION_COUNT_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_OCCURRENCE_COUNT_FIELD_NUMBER: _ClassVar[int]
    NEXT_AFTER_TXID_FIELD_NUMBER: _ClassVar[int]
    case: EvidenceCase
    captured_transaction_count: int
    captured_occurrence_count: int
    next_after_txid: str
    def __init__(self, case: _Optional[_Union[EvidenceCase, _Mapping]] = ..., captured_transaction_count: _Optional[int] = ..., captured_occurrence_count: _Optional[int] = ..., next_after_txid: _Optional[str] = ...) -> None: ...

class CaptureEVMWalletSnapshotRequest(_message.Message):
    __slots__ = ("case_id", "chain", "address", "token_contracts")
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    CHAIN_FIELD_NUMBER: _ClassVar[int]
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    TOKEN_CONTRACTS_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    chain: Chain
    address: str
    token_contracts: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, case_id: _Optional[str] = ..., chain: _Optional[_Union[Chain, str]] = ..., address: _Optional[str] = ..., token_contracts: _Optional[_Iterable[str]] = ...) -> None: ...

class CaptureEVMWalletSnapshotResponse(_message.Message):
    __slots__ = ("case",)
    CASE_FIELD_NUMBER: _ClassVar[int]
    case: EvidenceCase
    def __init__(self, case: _Optional[_Union[EvidenceCase, _Mapping]] = ...) -> None: ...

class CaptureSolanaAddressPageRequest(_message.Message):
    __slots__ = ("case_id", "address", "before_signature", "signature_limit", "signatures_only", "signature_page_count")
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    BEFORE_SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    SIGNATURE_LIMIT_FIELD_NUMBER: _ClassVar[int]
    SIGNATURES_ONLY_FIELD_NUMBER: _ClassVar[int]
    SIGNATURE_PAGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    address: str
    before_signature: str
    signature_limit: int
    signatures_only: bool
    signature_page_count: int
    def __init__(self, case_id: _Optional[str] = ..., address: _Optional[str] = ..., before_signature: _Optional[str] = ..., signature_limit: _Optional[int] = ..., signatures_only: _Optional[bool] = ..., signature_page_count: _Optional[int] = ...) -> None: ...

class CaptureSolanaAddressPageResponse(_message.Message):
    __slots__ = ("case", "captured_transaction_count", "captured_occurrence_count", "next_before_signature", "captured_pump_instruction_count", "unresolved_pump_instruction_count", "captured_signature_count", "signatures_only", "case_id", "captured_page_count", "stopped_reason")
    CASE_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_TRANSACTION_COUNT_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_OCCURRENCE_COUNT_FIELD_NUMBER: _ClassVar[int]
    NEXT_BEFORE_SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_PUMP_INSTRUCTION_COUNT_FIELD_NUMBER: _ClassVar[int]
    UNRESOLVED_PUMP_INSTRUCTION_COUNT_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_SIGNATURE_COUNT_FIELD_NUMBER: _ClassVar[int]
    SIGNATURES_ONLY_FIELD_NUMBER: _ClassVar[int]
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_PAGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    STOPPED_REASON_FIELD_NUMBER: _ClassVar[int]
    case: EvidenceCase
    captured_transaction_count: int
    captured_occurrence_count: int
    next_before_signature: str
    captured_pump_instruction_count: int
    unresolved_pump_instruction_count: int
    captured_signature_count: int
    signatures_only: bool
    case_id: str
    captured_page_count: int
    stopped_reason: str
    def __init__(self, case: _Optional[_Union[EvidenceCase, _Mapping]] = ..., captured_transaction_count: _Optional[int] = ..., captured_occurrence_count: _Optional[int] = ..., next_before_signature: _Optional[str] = ..., captured_pump_instruction_count: _Optional[int] = ..., unresolved_pump_instruction_count: _Optional[int] = ..., captured_signature_count: _Optional[int] = ..., signatures_only: _Optional[bool] = ..., case_id: _Optional[str] = ..., captured_page_count: _Optional[int] = ..., stopped_reason: _Optional[str] = ...) -> None: ...

class CaptureSolanaTransactionRequest(_message.Message):
    __slots__ = ("case_id", "signature", "selection_source_id")
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    SELECTION_SOURCE_ID_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    signature: str
    selection_source_id: str
    def __init__(self, case_id: _Optional[str] = ..., signature: _Optional[str] = ..., selection_source_id: _Optional[str] = ...) -> None: ...

class CaptureSolanaTransactionResponse(_message.Message):
    __slots__ = ("case", "captured_occurrence_count", "captured_pump_instruction_count", "unresolved_pump_instruction_count", "transaction_failed", "signature", "selection_source_id")
    CASE_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_OCCURRENCE_COUNT_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_PUMP_INSTRUCTION_COUNT_FIELD_NUMBER: _ClassVar[int]
    UNRESOLVED_PUMP_INSTRUCTION_COUNT_FIELD_NUMBER: _ClassVar[int]
    TRANSACTION_FAILED_FIELD_NUMBER: _ClassVar[int]
    SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    SELECTION_SOURCE_ID_FIELD_NUMBER: _ClassVar[int]
    case: EvidenceCase
    captured_occurrence_count: int
    captured_pump_instruction_count: int
    unresolved_pump_instruction_count: int
    transaction_failed: bool
    signature: str
    selection_source_id: str
    def __init__(self, case: _Optional[_Union[EvidenceCase, _Mapping]] = ..., captured_occurrence_count: _Optional[int] = ..., captured_pump_instruction_count: _Optional[int] = ..., unresolved_pump_instruction_count: _Optional[int] = ..., transaction_failed: _Optional[bool] = ..., signature: _Optional[str] = ..., selection_source_id: _Optional[str] = ...) -> None: ...

class CaptureSolanaTokenAccountsRequest(_message.Message):
    __slots__ = ("case_id", "owner", "program_id")
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    PROGRAM_ID_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    owner: str
    program_id: str
    def __init__(self, case_id: _Optional[str] = ..., owner: _Optional[str] = ..., program_id: _Optional[str] = ...) -> None: ...

class CaptureSolanaTokenAccountsResponse(_message.Message):
    __slots__ = ("case", "captured_account_count", "context_slot")
    CASE_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_ACCOUNT_COUNT_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_SLOT_FIELD_NUMBER: _ClassVar[int]
    case: EvidenceCase
    captured_account_count: int
    context_slot: int
    def __init__(self, case: _Optional[_Union[EvidenceCase, _Mapping]] = ..., captured_account_count: _Optional[int] = ..., context_slot: _Optional[int] = ...) -> None: ...

class ReviseCaseRequest(_message.Message):
    __slots__ = ("case_id", "expected_updated_at", "sources", "claims", "edges", "gaps", "counterexamples", "source_disagreements", "occurrences", "summary", "resolved_gaps", "edge_revisions", "pump_instructions", "unresolved_pump_instructions")
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    SOURCES_FIELD_NUMBER: _ClassVar[int]
    CLAIMS_FIELD_NUMBER: _ClassVar[int]
    EDGES_FIELD_NUMBER: _ClassVar[int]
    GAPS_FIELD_NUMBER: _ClassVar[int]
    COUNTEREXAMPLES_FIELD_NUMBER: _ClassVar[int]
    SOURCE_DISAGREEMENTS_FIELD_NUMBER: _ClassVar[int]
    OCCURRENCES_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_GAPS_FIELD_NUMBER: _ClassVar[int]
    EDGE_REVISIONS_FIELD_NUMBER: _ClassVar[int]
    PUMP_INSTRUCTIONS_FIELD_NUMBER: _ClassVar[int]
    UNRESOLVED_PUMP_INSTRUCTIONS_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    expected_updated_at: _timestamp_pb2.Timestamp
    sources: _containers.RepeatedCompositeFieldContainer[Source]
    claims: _containers.RepeatedCompositeFieldContainer[Claim]
    edges: _containers.RepeatedCompositeFieldContainer[FlowEdge]
    gaps: _containers.RepeatedScalarFieldContainer[str]
    counterexamples: _containers.RepeatedScalarFieldContainer[str]
    source_disagreements: _containers.RepeatedScalarFieldContainer[str]
    occurrences: _containers.RepeatedCompositeFieldContainer[Occurrence]
    summary: str
    resolved_gaps: _containers.RepeatedScalarFieldContainer[str]
    edge_revisions: _containers.RepeatedCompositeFieldContainer[EdgeRevision]
    pump_instructions: _containers.RepeatedCompositeFieldContainer[PumpInstruction]
    unresolved_pump_instructions: _containers.RepeatedCompositeFieldContainer[UnresolvedPumpInstruction]
    def __init__(self, case_id: _Optional[str] = ..., expected_updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., sources: _Optional[_Iterable[_Union[Source, _Mapping]]] = ..., claims: _Optional[_Iterable[_Union[Claim, _Mapping]]] = ..., edges: _Optional[_Iterable[_Union[FlowEdge, _Mapping]]] = ..., gaps: _Optional[_Iterable[str]] = ..., counterexamples: _Optional[_Iterable[str]] = ..., source_disagreements: _Optional[_Iterable[str]] = ..., occurrences: _Optional[_Iterable[_Union[Occurrence, _Mapping]]] = ..., summary: _Optional[str] = ..., resolved_gaps: _Optional[_Iterable[str]] = ..., edge_revisions: _Optional[_Iterable[_Union[EdgeRevision, _Mapping]]] = ..., pump_instructions: _Optional[_Iterable[_Union[PumpInstruction, _Mapping]]] = ..., unresolved_pump_instructions: _Optional[_Iterable[_Union[UnresolvedPumpInstruction, _Mapping]]] = ...) -> None: ...

class EdgeRevision(_message.Message):
    __slots__ = ("supersedes_edge_id", "replacement")
    SUPERSEDES_EDGE_ID_FIELD_NUMBER: _ClassVar[int]
    REPLACEMENT_FIELD_NUMBER: _ClassVar[int]
    supersedes_edge_id: str
    replacement: FlowEdge
    def __init__(self, supersedes_edge_id: _Optional[str] = ..., replacement: _Optional[_Union[FlowEdge, _Mapping]] = ...) -> None: ...

class ReviseCaseResponse(_message.Message):
    __slots__ = ("revision_id", "case", "previous_updated_at")
    REVISION_ID_FIELD_NUMBER: _ClassVar[int]
    CASE_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    revision_id: str
    case: EvidenceCase
    previous_updated_at: _timestamp_pb2.Timestamp
    def __init__(self, revision_id: _Optional[str] = ..., case: _Optional[_Union[EvidenceCase, _Mapping]] = ..., previous_updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CaseRevision(_message.Message):
    __slots__ = ("id", "case_id", "previous_updated_at", "applied_at", "additions", "previous_case")
    ID_FIELD_NUMBER: _ClassVar[int]
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    APPLIED_AT_FIELD_NUMBER: _ClassVar[int]
    ADDITIONS_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_CASE_FIELD_NUMBER: _ClassVar[int]
    id: str
    case_id: str
    previous_updated_at: _timestamp_pb2.Timestamp
    applied_at: _timestamp_pb2.Timestamp
    additions: ReviseCaseRequest
    previous_case: EvidenceCase
    def __init__(self, id: _Optional[str] = ..., case_id: _Optional[str] = ..., previous_updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., applied_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., additions: _Optional[_Union[ReviseCaseRequest, _Mapping]] = ..., previous_case: _Optional[_Union[EvidenceCase, _Mapping]] = ...) -> None: ...

class ListCaseRevisionsRequest(_message.Message):
    __slots__ = ("case_id",)
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    def __init__(self, case_id: _Optional[str] = ...) -> None: ...

class ListCaseRevisionsResponse(_message.Message):
    __slots__ = ("revisions",)
    REVISIONS_FIELD_NUMBER: _ClassVar[int]
    revisions: _containers.RepeatedCompositeFieldContainer[CaseRevision]
    def __init__(self, revisions: _Optional[_Iterable[_Union[CaseRevision, _Mapping]]] = ...) -> None: ...
