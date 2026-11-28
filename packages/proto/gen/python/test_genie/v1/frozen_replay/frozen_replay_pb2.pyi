from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class FrozenReplayVariant(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FROZEN_REPLAY_VARIANT_UNSPECIFIED: _ClassVar[FrozenReplayVariant]
    FROZEN_REPLAY_VARIANT_PREIMAGE: _ClassVar[FrozenReplayVariant]
    FROZEN_REPLAY_VARIANT_CANDIDATE: _ClassVar[FrozenReplayVariant]
    FROZEN_REPLAY_VARIANT_REFERENCE: _ClassVar[FrozenReplayVariant]

class FrozenReplayTestStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FROZEN_REPLAY_TEST_STATUS_UNSPECIFIED: _ClassVar[FrozenReplayTestStatus]
    FROZEN_REPLAY_TEST_STATUS_PASSED: _ClassVar[FrozenReplayTestStatus]
    FROZEN_REPLAY_TEST_STATUS_FAILED: _ClassVar[FrozenReplayTestStatus]

class FrozenReplayDisposition(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FROZEN_REPLAY_DISPOSITION_UNSPECIFIED: _ClassVar[FrozenReplayDisposition]
    FROZEN_REPLAY_DISPOSITION_QUALIFIED: _ClassVar[FrozenReplayDisposition]
    FROZEN_REPLAY_DISPOSITION_NOT_QUALIFIED: _ClassVar[FrozenReplayDisposition]
FROZEN_REPLAY_VARIANT_UNSPECIFIED: FrozenReplayVariant
FROZEN_REPLAY_VARIANT_PREIMAGE: FrozenReplayVariant
FROZEN_REPLAY_VARIANT_CANDIDATE: FrozenReplayVariant
FROZEN_REPLAY_VARIANT_REFERENCE: FrozenReplayVariant
FROZEN_REPLAY_TEST_STATUS_UNSPECIFIED: FrozenReplayTestStatus
FROZEN_REPLAY_TEST_STATUS_PASSED: FrozenReplayTestStatus
FROZEN_REPLAY_TEST_STATUS_FAILED: FrozenReplayTestStatus
FROZEN_REPLAY_DISPOSITION_UNSPECIFIED: FrozenReplayDisposition
FROZEN_REPLAY_DISPOSITION_QUALIFIED: FrozenReplayDisposition
FROZEN_REPLAY_DISPOSITION_NOT_QUALIFIED: FrozenReplayDisposition

class FrozenReplayRunEvidence(_message.Message):
    __slots__ = ("variant", "run_id", "producer_receipt_id", "source_identity", "evaluator_digest", "status", "artifact_digest")
    VARIANT_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    PRODUCER_RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    EVALUATOR_DIGEST_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    variant: FrozenReplayVariant
    run_id: str
    producer_receipt_id: str
    source_identity: str
    evaluator_digest: str
    status: FrozenReplayTestStatus
    artifact_digest: str
    def __init__(self, variant: _Optional[_Union[FrozenReplayVariant, str]] = ..., run_id: _Optional[str] = ..., producer_receipt_id: _Optional[str] = ..., source_identity: _Optional[str] = ..., evaluator_digest: _Optional[str] = ..., status: _Optional[_Union[FrozenReplayTestStatus, str]] = ..., artifact_digest: _Optional[str] = ...) -> None: ...

class FrozenReplayAdjudication(_message.Message):
    __slots__ = ("case_id", "project_id", "frozen_evaluator_digest", "preimage_identity", "candidate_seal_digest", "reference_identity", "evidence")
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    FROZEN_EVALUATOR_DIGEST_FIELD_NUMBER: _ClassVar[int]
    PREIMAGE_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_SEAL_DIGEST_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    project_id: str
    frozen_evaluator_digest: str
    preimage_identity: str
    candidate_seal_digest: str
    reference_identity: str
    evidence: _containers.RepeatedCompositeFieldContainer[FrozenReplayRunEvidence]
    def __init__(self, case_id: _Optional[str] = ..., project_id: _Optional[str] = ..., frozen_evaluator_digest: _Optional[str] = ..., preimage_identity: _Optional[str] = ..., candidate_seal_digest: _Optional[str] = ..., reference_identity: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[FrozenReplayRunEvidence, _Mapping]]] = ...) -> None: ...

class FrozenReplayReceipt(_message.Message):
    __slots__ = ("receipt_id", "adjudication", "disposition", "receipt_digest")
    RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    ADJUDICATION_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    receipt_id: str
    adjudication: FrozenReplayAdjudication
    disposition: FrozenReplayDisposition
    receipt_digest: str
    def __init__(self, receipt_id: _Optional[str] = ..., adjudication: _Optional[_Union[FrozenReplayAdjudication, _Mapping]] = ..., disposition: _Optional[_Union[FrozenReplayDisposition, str]] = ..., receipt_digest: _Optional[str] = ...) -> None: ...

class CreateFrozenReplayReceiptRequest(_message.Message):
    __slots__ = ("idempotency_key", "adjudication")
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    ADJUDICATION_FIELD_NUMBER: _ClassVar[int]
    idempotency_key: str
    adjudication: FrozenReplayAdjudication
    def __init__(self, idempotency_key: _Optional[str] = ..., adjudication: _Optional[_Union[FrozenReplayAdjudication, _Mapping]] = ...) -> None: ...

class CreateFrozenReplayReceiptResponse(_message.Message):
    __slots__ = ("receipt", "replayed")
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    REPLAYED_FIELD_NUMBER: _ClassVar[int]
    receipt: FrozenReplayReceipt
    replayed: bool
    def __init__(self, receipt: _Optional[_Union[FrozenReplayReceipt, _Mapping]] = ..., replayed: _Optional[bool] = ...) -> None: ...

class GetFrozenReplayReceiptRequest(_message.Message):
    __slots__ = ("receipt_id",)
    RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    receipt_id: str
    def __init__(self, receipt_id: _Optional[str] = ...) -> None: ...

class GetFrozenReplayReceiptResponse(_message.Message):
    __slots__ = ("receipt",)
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    receipt: FrozenReplayReceipt
    def __init__(self, receipt: _Optional[_Union[FrozenReplayReceipt, _Mapping]] = ...) -> None: ...
