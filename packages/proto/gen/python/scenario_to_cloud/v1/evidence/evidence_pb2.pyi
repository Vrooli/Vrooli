import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Disposition(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DISPOSITION_UNSPECIFIED: _ClassVar[Disposition]
    DISPOSITION_PASSED: _ClassVar[Disposition]
    DISPOSITION_FAILED: _ClassVar[Disposition]
    DISPOSITION_SKIPPED: _ClassVar[Disposition]
    DISPOSITION_UNSUPPORTED: _ClassVar[Disposition]
    DISPOSITION_UNAVAILABLE: _ClassVar[Disposition]
    DISPOSITION_MISSING: _ClassVar[Disposition]
DISPOSITION_UNSPECIFIED: Disposition
DISPOSITION_PASSED: Disposition
DISPOSITION_FAILED: Disposition
DISPOSITION_SKIPPED: Disposition
DISPOSITION_UNSUPPORTED: Disposition
DISPOSITION_UNAVAILABLE: Disposition
DISPOSITION_MISSING: Disposition

class ReviewIdentity(_message.Message):
    __slots__ = ("scenario_id", "profile_id", "candidate_commit", "artifact_digest", "release_digest", "configuration_digest", "target_set", "environment", "channel", "policy_version", "candidate_id", "destination_revision_id", "authorization_epoch", "digest")
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_COMMIT_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    RELEASE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CONFIGURATION_DIGEST_FIELD_NUMBER: _ClassVar[int]
    TARGET_SET_FIELD_NUMBER: _ClassVar[int]
    ENVIRONMENT_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_REVISION_ID_FIELD_NUMBER: _ClassVar[int]
    AUTHORIZATION_EPOCH_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    scenario_id: str
    profile_id: str
    candidate_commit: str
    artifact_digest: str
    release_digest: str
    configuration_digest: str
    target_set: _containers.RepeatedScalarFieldContainer[str]
    environment: str
    channel: str
    policy_version: int
    candidate_id: str
    destination_revision_id: str
    authorization_epoch: int
    digest: str
    def __init__(self, scenario_id: _Optional[str] = ..., profile_id: _Optional[str] = ..., candidate_commit: _Optional[str] = ..., artifact_digest: _Optional[str] = ..., release_digest: _Optional[str] = ..., configuration_digest: _Optional[str] = ..., target_set: _Optional[_Iterable[str]] = ..., environment: _Optional[str] = ..., channel: _Optional[str] = ..., policy_version: _Optional[int] = ..., candidate_id: _Optional[str] = ..., destination_revision_id: _Optional[str] = ..., authorization_epoch: _Optional[int] = ..., digest: _Optional[str] = ...) -> None: ...

class EvidenceCell(_message.Message):
    __slots__ = ("case_id", "lane", "disposition", "record_id", "operation_id", "observation_id", "receipt_refs", "observed_at", "reason", "required")
    CASE_ID_FIELD_NUMBER: _ClassVar[int]
    LANE_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    RECORD_ID_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    OBSERVATION_ID_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_REFS_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    case_id: str
    lane: str
    disposition: Disposition
    record_id: str
    operation_id: str
    observation_id: str
    receipt_refs: _containers.RepeatedScalarFieldContainer[str]
    observed_at: _timestamp_pb2.Timestamp
    reason: str
    required: bool
    def __init__(self, case_id: _Optional[str] = ..., lane: _Optional[str] = ..., disposition: _Optional[_Union[Disposition, str]] = ..., record_id: _Optional[str] = ..., operation_id: _Optional[str] = ..., observation_id: _Optional[str] = ..., receipt_refs: _Optional[_Iterable[str]] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., reason: _Optional[str] = ..., required: _Optional[bool] = ...) -> None: ...

class GetReleaseEvidenceRequest(_message.Message):
    __slots__ = ("release_digest", "deployment_id")
    RELEASE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    release_digest: str
    deployment_id: str
    def __init__(self, release_digest: _Optional[str] = ..., deployment_id: _Optional[str] = ...) -> None: ...

class ReleaseEvidence(_message.Message):
    __slots__ = ("schema_version", "profile_id", "release_digest", "target_key", "cells", "required_cells", "passed", "blocking_cells", "producer_ref", "evaluated_at")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    RELEASE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    TARGET_KEY_FIELD_NUMBER: _ClassVar[int]
    CELLS_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_CELLS_FIELD_NUMBER: _ClassVar[int]
    PASSED_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_CELLS_FIELD_NUMBER: _ClassVar[int]
    PRODUCER_REF_FIELD_NUMBER: _ClassVar[int]
    EVALUATED_AT_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    profile_id: str
    release_digest: str
    target_key: str
    cells: _containers.RepeatedCompositeFieldContainer[EvidenceCell]
    required_cells: int
    passed: bool
    blocking_cells: _containers.RepeatedScalarFieldContainer[str]
    producer_ref: str
    evaluated_at: _timestamp_pb2.Timestamp
    def __init__(self, schema_version: _Optional[str] = ..., profile_id: _Optional[str] = ..., release_digest: _Optional[str] = ..., target_key: _Optional[str] = ..., cells: _Optional[_Iterable[_Union[EvidenceCell, _Mapping]]] = ..., required_cells: _Optional[int] = ..., passed: _Optional[bool] = ..., blocking_cells: _Optional[_Iterable[str]] = ..., producer_ref: _Optional[str] = ..., evaluated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class PublicationRequest(_message.Message):
    __slots__ = ("deployment_id", "request_key", "identity")
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    REQUEST_KEY_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    deployment_id: str
    request_key: str
    identity: ReviewIdentity
    def __init__(self, deployment_id: _Optional[str] = ..., request_key: _Optional[str] = ..., identity: _Optional[_Union[ReviewIdentity, _Mapping]] = ...) -> None: ...

class ApplyPublicationRequest(_message.Message):
    __slots__ = ("deployment_id", "request_key", "review_ref", "plan_digest")
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    REQUEST_KEY_FIELD_NUMBER: _ClassVar[int]
    REVIEW_REF_FIELD_NUMBER: _ClassVar[int]
    PLAN_DIGEST_FIELD_NUMBER: _ClassVar[int]
    deployment_id: str
    request_key: str
    review_ref: str
    plan_digest: str
    def __init__(self, deployment_id: _Optional[str] = ..., request_key: _Optional[str] = ..., review_ref: _Optional[str] = ..., plan_digest: _Optional[str] = ...) -> None: ...

class GetPublicationRequest(_message.Message):
    __slots__ = ("deployment_id", "request_key")
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    REQUEST_KEY_FIELD_NUMBER: _ClassVar[int]
    deployment_id: str
    request_key: str
    def __init__(self, deployment_id: _Optional[str] = ..., request_key: _Optional[str] = ...) -> None: ...

class Refusal(_message.Message):
    __slots__ = ("code", "reason")
    CODE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    code: str
    reason: str
    def __init__(self, code: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class TargetReceipt(_message.Message):
    __slots__ = ("active_release", "previous_release", "activated_at", "operation_id", "fence", "receipt_digest")
    ACTIVE_RELEASE_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_RELEASE_FIELD_NUMBER: _ClassVar[int]
    ACTIVATED_AT_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    FENCE_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    active_release: str
    previous_release: str
    activated_at: str
    operation_id: str
    fence: int
    receipt_digest: str
    def __init__(self, active_release: _Optional[str] = ..., previous_release: _Optional[str] = ..., activated_at: _Optional[str] = ..., operation_id: _Optional[str] = ..., fence: _Optional[int] = ..., receipt_digest: _Optional[str] = ...) -> None: ...

class Publication(_message.Message):
    __slots__ = ("schema_version", "id", "deployment_id", "request_key", "identity", "review_ref", "release_digest", "plan_digest", "state", "operation_id", "activated_release_digest", "predecessor_release_digest", "target_key", "target_receipt", "refusal", "requested_at", "published_at", "evidence")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    REQUEST_KEY_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    REVIEW_REF_FIELD_NUMBER: _ClassVar[int]
    RELEASE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    PLAN_DIGEST_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    ACTIVATED_RELEASE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    PREDECESSOR_RELEASE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    TARGET_KEY_FIELD_NUMBER: _ClassVar[int]
    TARGET_RECEIPT_FIELD_NUMBER: _ClassVar[int]
    REFUSAL_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_AT_FIELD_NUMBER: _ClassVar[int]
    PUBLISHED_AT_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    id: str
    deployment_id: str
    request_key: str
    identity: ReviewIdentity
    review_ref: str
    release_digest: str
    plan_digest: str
    state: str
    operation_id: str
    activated_release_digest: str
    predecessor_release_digest: str
    target_key: str
    target_receipt: TargetReceipt
    refusal: Refusal
    requested_at: _timestamp_pb2.Timestamp
    published_at: _timestamp_pb2.Timestamp
    evidence: ReleaseEvidence
    def __init__(self, schema_version: _Optional[str] = ..., id: _Optional[str] = ..., deployment_id: _Optional[str] = ..., request_key: _Optional[str] = ..., identity: _Optional[_Union[ReviewIdentity, _Mapping]] = ..., review_ref: _Optional[str] = ..., release_digest: _Optional[str] = ..., plan_digest: _Optional[str] = ..., state: _Optional[str] = ..., operation_id: _Optional[str] = ..., activated_release_digest: _Optional[str] = ..., predecessor_release_digest: _Optional[str] = ..., target_key: _Optional[str] = ..., target_receipt: _Optional[_Union[TargetReceipt, _Mapping]] = ..., refusal: _Optional[_Union[Refusal, _Mapping]] = ..., requested_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., published_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., evidence: _Optional[_Union[ReleaseEvidence, _Mapping]] = ...) -> None: ...
