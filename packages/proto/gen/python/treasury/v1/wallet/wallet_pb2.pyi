import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from treasury.v1.shared import transfer_observation_pb2 as _transfer_observation_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class BaseUSDCReceiptRelation(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    BASE_USDC_RECEIPT_RELATION_UNSPECIFIED: _ClassVar[BaseUSDCReceiptRelation]
    BASE_USDC_RECEIPT_RELATION_SAME_INCLUSION: _ClassVar[BaseUSDCReceiptRelation]
    BASE_USDC_RECEIPT_RELATION_REINCLUDED_NEW_REVISION: _ClassVar[BaseUSDCReceiptRelation]
    BASE_USDC_RECEIPT_RELATION_PROVIDER_REPORTS_NOT_CANONICAL: _ClassVar[BaseUSDCReceiptRelation]
    BASE_USDC_RECEIPT_RELATION_PROVIDER_REPORTS_PENDING: _ClassVar[BaseUSDCReceiptRelation]
    BASE_USDC_RECEIPT_RELATION_PROVIDER_CHECK_FAILED: _ClassVar[BaseUSDCReceiptRelation]
    BASE_USDC_RECEIPT_RELATION_PROVIDER_OBSERVATION_CHANGED: _ClassVar[BaseUSDCReceiptRelation]
BASE_USDC_RECEIPT_RELATION_UNSPECIFIED: BaseUSDCReceiptRelation
BASE_USDC_RECEIPT_RELATION_SAME_INCLUSION: BaseUSDCReceiptRelation
BASE_USDC_RECEIPT_RELATION_REINCLUDED_NEW_REVISION: BaseUSDCReceiptRelation
BASE_USDC_RECEIPT_RELATION_PROVIDER_REPORTS_NOT_CANONICAL: BaseUSDCReceiptRelation
BASE_USDC_RECEIPT_RELATION_PROVIDER_REPORTS_PENDING: BaseUSDCReceiptRelation
BASE_USDC_RECEIPT_RELATION_PROVIDER_CHECK_FAILED: BaseUSDCReceiptRelation
BASE_USDC_RECEIPT_RELATION_PROVIDER_OBSERVATION_CHANGED: BaseUSDCReceiptRelation

class ControlProof(_message.Message):
    __slots__ = ("wallet", "message", "signature", "scheme", "verified")
    WALLET_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    SCHEME_FIELD_NUMBER: _ClassVar[int]
    VERIFIED_FIELD_NUMBER: _ClassVar[int]
    wallet: _transfer_observation_pb2.Wallet
    message: str
    signature: str
    scheme: str
    verified: bool
    def __init__(self, wallet: _Optional[_Union[_transfer_observation_pb2.Wallet, _Mapping]] = ..., message: _Optional[str] = ..., signature: _Optional[str] = ..., scheme: _Optional[str] = ..., verified: _Optional[bool] = ...) -> None: ...

class BaseUSDCReceiptCheck(_message.Message):
    __slots__ = ("id", "wallet_id", "network", "transaction_hash", "provider_id", "outcome", "error_code", "checked_at", "transfers", "prior_check_id", "relation")
    ID_FIELD_NUMBER: _ClassVar[int]
    WALLET_ID_FIELD_NUMBER: _ClassVar[int]
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    TRANSACTION_HASH_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    ERROR_CODE_FIELD_NUMBER: _ClassVar[int]
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    TRANSFERS_FIELD_NUMBER: _ClassVar[int]
    PRIOR_CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    RELATION_FIELD_NUMBER: _ClassVar[int]
    id: str
    wallet_id: str
    network: str
    transaction_hash: str
    provider_id: str
    outcome: str
    error_code: str
    checked_at: _timestamp_pb2.Timestamp
    transfers: _containers.RepeatedCompositeFieldContainer[_transfer_observation_pb2.BaseUSDCTransferObservation]
    prior_check_id: str
    relation: BaseUSDCReceiptRelation
    def __init__(self, id: _Optional[str] = ..., wallet_id: _Optional[str] = ..., network: _Optional[str] = ..., transaction_hash: _Optional[str] = ..., provider_id: _Optional[str] = ..., outcome: _Optional[str] = ..., error_code: _Optional[str] = ..., checked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., transfers: _Optional[_Iterable[_Union[_transfer_observation_pb2.BaseUSDCTransferObservation, _Mapping]]] = ..., prior_check_id: _Optional[str] = ..., relation: _Optional[_Union[BaseUSDCReceiptRelation, str]] = ...) -> None: ...

class CreateResearchWalletRequest(_message.Message):
    __slots__ = ("id", "book_id", "network")
    ID_FIELD_NUMBER: _ClassVar[int]
    BOOK_ID_FIELD_NUMBER: _ClassVar[int]
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    id: str
    book_id: str
    network: str
    def __init__(self, id: _Optional[str] = ..., book_id: _Optional[str] = ..., network: _Optional[str] = ...) -> None: ...

class RotateResearchWalletRequest(_message.Message):
    __slots__ = ("id", "network", "successor_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    SUCCESSOR_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    network: str
    successor_id: str
    def __init__(self, id: _Optional[str] = ..., network: _Optional[str] = ..., successor_id: _Optional[str] = ...) -> None: ...

class ResearchWalletRequest(_message.Message):
    __slots__ = ("id", "network")
    ID_FIELD_NUMBER: _ClassVar[int]
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    id: str
    network: str
    def __init__(self, id: _Optional[str] = ..., network: _Optional[str] = ...) -> None: ...

class ResearchWalletResponse(_message.Message):
    __slots__ = ("wallet",)
    WALLET_FIELD_NUMBER: _ClassVar[int]
    wallet: _transfer_observation_pb2.Wallet
    def __init__(self, wallet: _Optional[_Union[_transfer_observation_pb2.Wallet, _Mapping]] = ...) -> None: ...

class ExportPortableResearchWalletRecoveryRequest(_message.Message):
    __slots__ = ("id", "network", "operator_secret")
    ID_FIELD_NUMBER: _ClassVar[int]
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_SECRET_FIELD_NUMBER: _ClassVar[int]
    id: str
    network: str
    operator_secret: str
    def __init__(self, id: _Optional[str] = ..., network: _Optional[str] = ..., operator_secret: _Optional[str] = ...) -> None: ...

class ExportPortableResearchWalletRecoveryResponse(_message.Message):
    __slots__ = ("relative_path", "sha256", "size_bytes", "witness_relative_path")
    RELATIVE_PATH_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    WITNESS_RELATIVE_PATH_FIELD_NUMBER: _ClassVar[int]
    relative_path: str
    sha256: str
    size_bytes: int
    witness_relative_path: str
    def __init__(self, relative_path: _Optional[str] = ..., sha256: _Optional[str] = ..., size_bytes: _Optional[int] = ..., witness_relative_path: _Optional[str] = ...) -> None: ...

class RestorePortableResearchWalletRecoveryRequest(_message.Message):
    __slots__ = ("id", "network", "bundle", "operator_secret", "witness", "revocation")
    ID_FIELD_NUMBER: _ClassVar[int]
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_SECRET_FIELD_NUMBER: _ClassVar[int]
    WITNESS_FIELD_NUMBER: _ClassVar[int]
    REVOCATION_FIELD_NUMBER: _ClassVar[int]
    id: str
    network: str
    bundle: bytes
    operator_secret: str
    witness: bytes
    revocation: bytes
    def __init__(self, id: _Optional[str] = ..., network: _Optional[str] = ..., bundle: _Optional[bytes] = ..., operator_secret: _Optional[str] = ..., witness: _Optional[bytes] = ..., revocation: _Optional[bytes] = ...) -> None: ...

class PreparePortableResearchWalletRotationRequest(_message.Message):
    __slots__ = ("id", "network", "successor_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    SUCCESSOR_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    network: str
    successor_id: str
    def __init__(self, id: _Optional[str] = ..., network: _Optional[str] = ..., successor_id: _Optional[str] = ...) -> None: ...

class PreparePortableResearchWalletRotationResponse(_message.Message):
    __slots__ = ("wallet", "revocation_tombstone_json", "tombstone_sha256")
    WALLET_FIELD_NUMBER: _ClassVar[int]
    REVOCATION_TOMBSTONE_JSON_FIELD_NUMBER: _ClassVar[int]
    TOMBSTONE_SHA256_FIELD_NUMBER: _ClassVar[int]
    wallet: _transfer_observation_pb2.Wallet
    revocation_tombstone_json: str
    tombstone_sha256: str
    def __init__(self, wallet: _Optional[_Union[_transfer_observation_pb2.Wallet, _Mapping]] = ..., revocation_tombstone_json: _Optional[str] = ..., tombstone_sha256: _Optional[str] = ...) -> None: ...

class CommitPortableResearchWalletRotationRequest(_message.Message):
    __slots__ = ("id", "network", "successor_id", "destination_id", "destination_backend", "plan_id", "run_id", "target_name", "snapshot_id", "latest_snapshot_id", "tombstone_sha256", "restored_revocation_sha256", "bundle_present", "dbm_verify_passed", "restored_tombstone_matches", "verified_at", "address", "reservation_id", "reservation_expires_at", "dbm_attestation_json")
    ID_FIELD_NUMBER: _ClassVar[int]
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    SUCCESSOR_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_BACKEND_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_NAME_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    LATEST_SNAPSHOT_ID_FIELD_NUMBER: _ClassVar[int]
    TOMBSTONE_SHA256_FIELD_NUMBER: _ClassVar[int]
    RESTORED_REVOCATION_SHA256_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_PRESENT_FIELD_NUMBER: _ClassVar[int]
    DBM_VERIFY_PASSED_FIELD_NUMBER: _ClassVar[int]
    RESTORED_TOMBSTONE_MATCHES_FIELD_NUMBER: _ClassVar[int]
    VERIFIED_AT_FIELD_NUMBER: _ClassVar[int]
    ADDRESS_FIELD_NUMBER: _ClassVar[int]
    RESERVATION_ID_FIELD_NUMBER: _ClassVar[int]
    RESERVATION_EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    DBM_ATTESTATION_JSON_FIELD_NUMBER: _ClassVar[int]
    id: str
    network: str
    successor_id: str
    destination_id: str
    destination_backend: str
    plan_id: str
    run_id: str
    target_name: str
    snapshot_id: str
    latest_snapshot_id: str
    tombstone_sha256: str
    restored_revocation_sha256: str
    bundle_present: bool
    dbm_verify_passed: bool
    restored_tombstone_matches: bool
    verified_at: str
    address: str
    reservation_id: str
    reservation_expires_at: str
    dbm_attestation_json: str
    def __init__(self, id: _Optional[str] = ..., network: _Optional[str] = ..., successor_id: _Optional[str] = ..., destination_id: _Optional[str] = ..., destination_backend: _Optional[str] = ..., plan_id: _Optional[str] = ..., run_id: _Optional[str] = ..., target_name: _Optional[str] = ..., snapshot_id: _Optional[str] = ..., latest_snapshot_id: _Optional[str] = ..., tombstone_sha256: _Optional[str] = ..., restored_revocation_sha256: _Optional[str] = ..., bundle_present: _Optional[bool] = ..., dbm_verify_passed: _Optional[bool] = ..., restored_tombstone_matches: _Optional[bool] = ..., verified_at: _Optional[str] = ..., address: _Optional[str] = ..., reservation_id: _Optional[str] = ..., reservation_expires_at: _Optional[str] = ..., dbm_attestation_json: _Optional[str] = ...) -> None: ...

class CommitPortableResearchWalletRotationResponse(_message.Message):
    __slots__ = ("wallet",)
    WALLET_FIELD_NUMBER: _ClassVar[int]
    wallet: _transfer_observation_pb2.Wallet
    def __init__(self, wallet: _Optional[_Union[_transfer_observation_pb2.Wallet, _Mapping]] = ...) -> None: ...

class ProveResearchWalletRequest(_message.Message):
    __slots__ = ("id", "network", "nonce")
    ID_FIELD_NUMBER: _ClassVar[int]
    NETWORK_FIELD_NUMBER: _ClassVar[int]
    NONCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    network: str
    nonce: str
    def __init__(self, id: _Optional[str] = ..., network: _Optional[str] = ..., nonce: _Optional[str] = ...) -> None: ...

class ProveResearchWalletResponse(_message.Message):
    __slots__ = ("proof",)
    PROOF_FIELD_NUMBER: _ClassVar[int]
    proof: ControlProof
    def __init__(self, proof: _Optional[_Union[ControlProof, _Mapping]] = ...) -> None: ...

class VerifyBaseUSDCReceiptRequest(_message.Message):
    __slots__ = ("wallet_id", "transaction_hash")
    WALLET_ID_FIELD_NUMBER: _ClassVar[int]
    TRANSACTION_HASH_FIELD_NUMBER: _ClassVar[int]
    wallet_id: str
    transaction_hash: str
    def __init__(self, wallet_id: _Optional[str] = ..., transaction_hash: _Optional[str] = ...) -> None: ...

class VerifyBaseUSDCReceiptResponse(_message.Message):
    __slots__ = ("transfers", "check_id", "check_outcome", "provider_id", "checked_at")
    TRANSFERS_FIELD_NUMBER: _ClassVar[int]
    CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    CHECK_OUTCOME_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    transfers: _containers.RepeatedCompositeFieldContainer[_transfer_observation_pb2.BaseUSDCTransferObservation]
    check_id: str
    check_outcome: str
    provider_id: str
    checked_at: _timestamp_pb2.Timestamp
    def __init__(self, transfers: _Optional[_Iterable[_Union[_transfer_observation_pb2.BaseUSDCTransferObservation, _Mapping]]] = ..., check_id: _Optional[str] = ..., check_outcome: _Optional[str] = ..., provider_id: _Optional[str] = ..., checked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RecheckBaseUSDCReceiptRequest(_message.Message):
    __slots__ = ("wallet_id", "transaction_hash", "prior_check_id")
    WALLET_ID_FIELD_NUMBER: _ClassVar[int]
    TRANSACTION_HASH_FIELD_NUMBER: _ClassVar[int]
    PRIOR_CHECK_ID_FIELD_NUMBER: _ClassVar[int]
    wallet_id: str
    transaction_hash: str
    prior_check_id: str
    def __init__(self, wallet_id: _Optional[str] = ..., transaction_hash: _Optional[str] = ..., prior_check_id: _Optional[str] = ...) -> None: ...

class RecheckBaseUSDCReceiptResponse(_message.Message):
    __slots__ = ("check",)
    CHECK_FIELD_NUMBER: _ClassVar[int]
    check: BaseUSDCReceiptCheck
    def __init__(self, check: _Optional[_Union[BaseUSDCReceiptCheck, _Mapping]] = ...) -> None: ...

class ListBaseUSDCReceiptChecksRequest(_message.Message):
    __slots__ = ("wallet_id", "transaction_hash", "limit")
    WALLET_ID_FIELD_NUMBER: _ClassVar[int]
    TRANSACTION_HASH_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    wallet_id: str
    transaction_hash: str
    limit: int
    def __init__(self, wallet_id: _Optional[str] = ..., transaction_hash: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListBaseUSDCReceiptChecksResponse(_message.Message):
    __slots__ = ("checks",)
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    checks: _containers.RepeatedCompositeFieldContainer[BaseUSDCReceiptCheck]
    def __init__(self, checks: _Optional[_Iterable[_Union[BaseUSDCReceiptCheck, _Mapping]]] = ...) -> None: ...
