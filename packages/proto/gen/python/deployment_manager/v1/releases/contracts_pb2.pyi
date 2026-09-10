import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ReceiptOutcome(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RECEIPT_OUTCOME_UNSPECIFIED: _ClassVar[ReceiptOutcome]
    RECEIPT_OUTCOME_PREPARED: _ClassVar[ReceiptOutcome]
    RECEIPT_OUTCOME_STAGED: _ClassVar[ReceiptOutcome]
    RECEIPT_OUTCOME_PUBLISHED: _ClassVar[ReceiptOutcome]
    RECEIPT_OUTCOME_VERIFIED: _ClassVar[ReceiptOutcome]
    RECEIPT_OUTCOME_FAILED: _ClassVar[ReceiptOutcome]
    RECEIPT_OUTCOME_AMBIGUOUS: _ClassVar[ReceiptOutcome]
    RECEIPT_OUTCOME_UNAVAILABLE: _ClassVar[ReceiptOutcome]

class ReleaseOperationStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RELEASE_OPERATION_STATUS_UNSPECIFIED: _ClassVar[ReleaseOperationStatus]
    RELEASE_OPERATION_STATUS_QUEUED: _ClassVar[ReleaseOperationStatus]
    RELEASE_OPERATION_STATUS_RUNNING: _ClassVar[ReleaseOperationStatus]
    RELEASE_OPERATION_STATUS_COMPLETE: _ClassVar[ReleaseOperationStatus]
    RELEASE_OPERATION_STATUS_FAILED: _ClassVar[ReleaseOperationStatus]
    RELEASE_OPERATION_STATUS_CANCELED: _ClassVar[ReleaseOperationStatus]
    RELEASE_OPERATION_STATUS_AMBIGUOUS: _ClassVar[ReleaseOperationStatus]
RECEIPT_OUTCOME_UNSPECIFIED: ReceiptOutcome
RECEIPT_OUTCOME_PREPARED: ReceiptOutcome
RECEIPT_OUTCOME_STAGED: ReceiptOutcome
RECEIPT_OUTCOME_PUBLISHED: ReceiptOutcome
RECEIPT_OUTCOME_VERIFIED: ReceiptOutcome
RECEIPT_OUTCOME_FAILED: ReceiptOutcome
RECEIPT_OUTCOME_AMBIGUOUS: ReceiptOutcome
RECEIPT_OUTCOME_UNAVAILABLE: ReceiptOutcome
RELEASE_OPERATION_STATUS_UNSPECIFIED: ReleaseOperationStatus
RELEASE_OPERATION_STATUS_QUEUED: ReleaseOperationStatus
RELEASE_OPERATION_STATUS_RUNNING: ReleaseOperationStatus
RELEASE_OPERATION_STATUS_COMPLETE: ReleaseOperationStatus
RELEASE_OPERATION_STATUS_FAILED: ReleaseOperationStatus
RELEASE_OPERATION_STATUS_CANCELED: ReleaseOperationStatus
RELEASE_OPERATION_STATUS_AMBIGUOUS: ReleaseOperationStatus

class TargetIdentity(_message.Message):
    __slots__ = ("id", "platform", "os", "architecture", "format")
    ID_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    OS_FIELD_NUMBER: _ClassVar[int]
    ARCHITECTURE_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    id: str
    platform: str
    os: str
    architecture: str
    format: str
    def __init__(self, id: _Optional[str] = ..., platform: _Optional[str] = ..., os: _Optional[str] = ..., architecture: _Optional[str] = ..., format: _Optional[str] = ...) -> None: ...

class CandidateArtifact(_message.Message):
    __slots__ = ("target", "immutable_ref", "digest", "size_bytes", "signature_digest", "signer_ref")
    TARGET_FIELD_NUMBER: _ClassVar[int]
    IMMUTABLE_REF_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    SIGNATURE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    SIGNER_REF_FIELD_NUMBER: _ClassVar[int]
    target: TargetIdentity
    immutable_ref: str
    digest: str
    size_bytes: int
    signature_digest: str
    signer_ref: str
    def __init__(self, target: _Optional[_Union[TargetIdentity, _Mapping]] = ..., immutable_ref: _Optional[str] = ..., digest: _Optional[str] = ..., size_bytes: _Optional[int] = ..., signature_digest: _Optional[str] = ..., signer_ref: _Optional[str] = ...) -> None: ...

class CapabilityDeclaration(_message.Message):
    __slots__ = ("support_owner", "incident_owner", "customer_contact", "release_authority", "rollback_authority", "degraded_mode_authority")
    SUPPORT_OWNER_FIELD_NUMBER: _ClassVar[int]
    INCIDENT_OWNER_FIELD_NUMBER: _ClassVar[int]
    CUSTOMER_CONTACT_FIELD_NUMBER: _ClassVar[int]
    RELEASE_AUTHORITY_FIELD_NUMBER: _ClassVar[int]
    ROLLBACK_AUTHORITY_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_MODE_AUTHORITY_FIELD_NUMBER: _ClassVar[int]
    support_owner: str
    incident_owner: str
    customer_contact: str
    release_authority: str
    rollback_authority: str
    degraded_mode_authority: str
    def __init__(self, support_owner: _Optional[str] = ..., incident_owner: _Optional[str] = ..., customer_contact: _Optional[str] = ..., release_authority: _Optional[str] = ..., rollback_authority: _Optional[str] = ..., degraded_mode_authority: _Optional[str] = ...) -> None: ...

class Candidate(_message.Message):
    __slots__ = ("source_revision", "profile_revision", "build_inputs", "dependency_lock_digest", "policy_digest", "artifacts", "candidate_id", "capability_declaration")
    class BuildInputsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SOURCE_REVISION_FIELD_NUMBER: _ClassVar[int]
    PROFILE_REVISION_FIELD_NUMBER: _ClassVar[int]
    BUILD_INPUTS_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCY_LOCK_DIGEST_FIELD_NUMBER: _ClassVar[int]
    POLICY_DIGEST_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_DECLARATION_FIELD_NUMBER: _ClassVar[int]
    source_revision: str
    profile_revision: str
    build_inputs: _containers.ScalarMap[str, str]
    dependency_lock_digest: str
    policy_digest: str
    artifacts: _containers.RepeatedCompositeFieldContainer[CandidateArtifact]
    candidate_id: str
    capability_declaration: CapabilityDeclaration
    def __init__(self, source_revision: _Optional[str] = ..., profile_revision: _Optional[str] = ..., build_inputs: _Optional[_Mapping[str, str]] = ..., dependency_lock_digest: _Optional[str] = ..., policy_digest: _Optional[str] = ..., artifacts: _Optional[_Iterable[_Union[CandidateArtifact, _Mapping]]] = ..., candidate_id: _Optional[str] = ..., capability_declaration: _Optional[_Union[CapabilityDeclaration, _Mapping]] = ...) -> None: ...

class DestinationRevision(_message.Message):
    __slots__ = ("kind", "destination_id", "configuration_digest", "channel", "expected_channel_revision", "destination_revision_id")
    KIND_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_ID_FIELD_NUMBER: _ClassVar[int]
    CONFIGURATION_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_CHANNEL_REVISION_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_REVISION_ID_FIELD_NUMBER: _ClassVar[int]
    kind: str
    destination_id: str
    configuration_digest: str
    channel: str
    expected_channel_revision: str
    destination_revision_id: str
    def __init__(self, kind: _Optional[str] = ..., destination_id: _Optional[str] = ..., configuration_digest: _Optional[str] = ..., channel: _Optional[str] = ..., expected_channel_revision: _Optional[str] = ..., destination_revision_id: _Optional[str] = ...) -> None: ...

class ReviewBinding(_message.Message):
    __slots__ = ("candidate_id", "destination_revision_id", "targets", "channel", "evidence_set_digest", "policy_digest", "authorization_epoch", "review_id")
    CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_REVISION_ID_FIELD_NUMBER: _ClassVar[int]
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_SET_DIGEST_FIELD_NUMBER: _ClassVar[int]
    POLICY_DIGEST_FIELD_NUMBER: _ClassVar[int]
    AUTHORIZATION_EPOCH_FIELD_NUMBER: _ClassVar[int]
    REVIEW_ID_FIELD_NUMBER: _ClassVar[int]
    candidate_id: str
    destination_revision_id: str
    targets: _containers.RepeatedScalarFieldContainer[str]
    channel: str
    evidence_set_digest: str
    policy_digest: str
    authorization_epoch: int
    review_id: str
    def __init__(self, candidate_id: _Optional[str] = ..., destination_revision_id: _Optional[str] = ..., targets: _Optional[_Iterable[str]] = ..., channel: _Optional[str] = ..., evidence_set_digest: _Optional[str] = ..., policy_digest: _Optional[str] = ..., authorization_epoch: _Optional[int] = ..., review_id: _Optional[str] = ...) -> None: ...

class PublicationReceipt(_message.Message):
    __slots__ = ("candidate_id", "destination_revision_id", "target_id", "artifact_digest", "destination_object", "producer", "external_receipt", "outcome", "observed_at")
    CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_REVISION_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_OBJECT_FIELD_NUMBER: _ClassVar[int]
    PRODUCER_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_RECEIPT_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    candidate_id: str
    destination_revision_id: str
    target_id: str
    artifact_digest: str
    destination_object: str
    producer: str
    external_receipt: str
    outcome: ReceiptOutcome
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, candidate_id: _Optional[str] = ..., destination_revision_id: _Optional[str] = ..., target_id: _Optional[str] = ..., artifact_digest: _Optional[str] = ..., destination_object: _Optional[str] = ..., producer: _Optional[str] = ..., external_receipt: _Optional[str] = ..., outcome: _Optional[_Union[ReceiptOutcome, str]] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ClientUpdateReceipt(_message.Message):
    __slots__ = ("candidate_id", "predecessor_ref", "successor_digest", "target_id", "verified_version", "outcome", "producer", "external_receipt", "observed_at")
    CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    PREDECESSOR_REF_FIELD_NUMBER: _ClassVar[int]
    SUCCESSOR_DIGEST_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    VERIFIED_VERSION_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    PRODUCER_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_RECEIPT_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    candidate_id: str
    predecessor_ref: str
    successor_digest: str
    target_id: str
    verified_version: str
    outcome: ReceiptOutcome
    producer: str
    external_receipt: str
    observed_at: _timestamp_pb2.Timestamp
    def __init__(self, candidate_id: _Optional[str] = ..., predecessor_ref: _Optional[str] = ..., successor_digest: _Optional[str] = ..., target_id: _Optional[str] = ..., verified_version: _Optional[str] = ..., outcome: _Optional[_Union[ReceiptOutcome, str]] = ..., producer: _Optional[str] = ..., external_receipt: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RecoveryReceipt(_message.Message):
    __slots__ = ("release_id", "candidate_id", "destination_revision_id", "deployment_id", "action", "outcome", "health", "external_receipt", "observed_at", "dry_run", "bundle_sha256")
    RELEASE_ID_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_REVISION_ID_FIELD_NUMBER: _ClassVar[int]
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    HEALTH_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_RECEIPT_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_SHA256_FIELD_NUMBER: _ClassVar[int]
    release_id: str
    candidate_id: str
    destination_revision_id: str
    deployment_id: str
    action: str
    outcome: str
    health: str
    external_receipt: str
    observed_at: _timestamp_pb2.Timestamp
    dry_run: bool
    bundle_sha256: str
    def __init__(self, release_id: _Optional[str] = ..., candidate_id: _Optional[str] = ..., destination_revision_id: _Optional[str] = ..., deployment_id: _Optional[str] = ..., action: _Optional[str] = ..., outcome: _Optional[str] = ..., health: _Optional[str] = ..., external_receipt: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., dry_run: _Optional[bool] = ..., bundle_sha256: _Optional[str] = ...) -> None: ...

class ReleaseOperation(_message.Message):
    __slots__ = ("operation_id", "release_id", "profile_id", "idempotency_key", "status", "active_stage", "error", "created_at", "updated_at", "completed_at")
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    RELEASE_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_STAGE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    release_id: str
    profile_id: str
    idempotency_key: str
    status: ReleaseOperationStatus
    active_stage: str
    error: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    completed_at: _timestamp_pb2.Timestamp
    def __init__(self, operation_id: _Optional[str] = ..., release_id: _Optional[str] = ..., profile_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., status: _Optional[_Union[ReleaseOperationStatus, str]] = ..., active_stage: _Optional[str] = ..., error: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., completed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ReleasePlatformView(_message.Message):
    __slots__ = ("platform", "status", "error")
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    platform: str
    status: str
    error: str
    def __init__(self, platform: _Optional[str] = ..., status: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...

class ReleaseView(_message.Message):
    __slots__ = ("release_id", "profile_id", "git_commit_hash", "artifact_digest", "candidate_id", "destination_revision_id", "authorization_epoch", "readiness_review_key", "release_version", "channel", "status", "release_notes", "released_by", "platforms", "created_at", "published_at", "updated_at", "candidate", "destination_revision", "review_binding", "publication_receipts", "client_update_receipts", "deployment_id", "recovery_receipts")
    RELEASE_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    GIT_COMMIT_HASH_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_REVISION_ID_FIELD_NUMBER: _ClassVar[int]
    AUTHORIZATION_EPOCH_FIELD_NUMBER: _ClassVar[int]
    READINESS_REVIEW_KEY_FIELD_NUMBER: _ClassVar[int]
    RELEASE_VERSION_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    RELEASE_NOTES_FIELD_NUMBER: _ClassVar[int]
    RELEASED_BY_FIELD_NUMBER: _ClassVar[int]
    PLATFORMS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    PUBLISHED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_REVISION_FIELD_NUMBER: _ClassVar[int]
    REVIEW_BINDING_FIELD_NUMBER: _ClassVar[int]
    PUBLICATION_RECEIPTS_FIELD_NUMBER: _ClassVar[int]
    CLIENT_UPDATE_RECEIPTS_FIELD_NUMBER: _ClassVar[int]
    DEPLOYMENT_ID_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_RECEIPTS_FIELD_NUMBER: _ClassVar[int]
    release_id: str
    profile_id: str
    git_commit_hash: str
    artifact_digest: str
    candidate_id: str
    destination_revision_id: str
    authorization_epoch: int
    readiness_review_key: str
    release_version: str
    channel: str
    status: str
    release_notes: str
    released_by: str
    platforms: _containers.RepeatedCompositeFieldContainer[ReleasePlatformView]
    created_at: _timestamp_pb2.Timestamp
    published_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    candidate: Candidate
    destination_revision: DestinationRevision
    review_binding: ReviewBinding
    publication_receipts: _containers.RepeatedCompositeFieldContainer[PublicationReceipt]
    client_update_receipts: _containers.RepeatedCompositeFieldContainer[ClientUpdateReceipt]
    deployment_id: str
    recovery_receipts: _containers.RepeatedCompositeFieldContainer[RecoveryReceipt]
    def __init__(self, release_id: _Optional[str] = ..., profile_id: _Optional[str] = ..., git_commit_hash: _Optional[str] = ..., artifact_digest: _Optional[str] = ..., candidate_id: _Optional[str] = ..., destination_revision_id: _Optional[str] = ..., authorization_epoch: _Optional[int] = ..., readiness_review_key: _Optional[str] = ..., release_version: _Optional[str] = ..., channel: _Optional[str] = ..., status: _Optional[str] = ..., release_notes: _Optional[str] = ..., released_by: _Optional[str] = ..., platforms: _Optional[_Iterable[_Union[ReleasePlatformView, _Mapping]]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., published_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., candidate: _Optional[_Union[Candidate, _Mapping]] = ..., destination_revision: _Optional[_Union[DestinationRevision, _Mapping]] = ..., review_binding: _Optional[_Union[ReviewBinding, _Mapping]] = ..., publication_receipts: _Optional[_Iterable[_Union[PublicationReceipt, _Mapping]]] = ..., client_update_receipts: _Optional[_Iterable[_Union[ClientUpdateReceipt, _Mapping]]] = ..., deployment_id: _Optional[str] = ..., recovery_receipts: _Optional[_Iterable[_Union[RecoveryReceipt, _Mapping]]] = ...) -> None: ...

class ListReleasesRequest(_message.Message):
    __slots__ = ("profile_id", "limit")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    limit: int
    def __init__(self, profile_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListReleasesResponse(_message.Message):
    __slots__ = ("releases",)
    RELEASES_FIELD_NUMBER: _ClassVar[int]
    releases: _containers.RepeatedCompositeFieldContainer[ReleaseView]
    def __init__(self, releases: _Optional[_Iterable[_Union[ReleaseView, _Mapping]]] = ...) -> None: ...

class GetReleaseRequest(_message.Message):
    __slots__ = ("release_id",)
    RELEASE_ID_FIELD_NUMBER: _ClassVar[int]
    release_id: str
    def __init__(self, release_id: _Optional[str] = ...) -> None: ...

class GetReleaseResponse(_message.Message):
    __slots__ = ("release",)
    RELEASE_FIELD_NUMBER: _ClassVar[int]
    release: ReleaseView
    def __init__(self, release: _Optional[_Union[ReleaseView, _Mapping]] = ...) -> None: ...

class GetReleaseOperationRequest(_message.Message):
    __slots__ = ("operation_id",)
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    def __init__(self, operation_id: _Optional[str] = ...) -> None: ...

class GetReleaseOperationResponse(_message.Message):
    __slots__ = ("operation",)
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    operation: ReleaseOperation
    def __init__(self, operation: _Optional[_Union[ReleaseOperation, _Mapping]] = ...) -> None: ...

class ReleaseAlert(_message.Message):
    __slots__ = ("code", "severity", "target", "message", "next_action")
    CODE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    code: str
    severity: str
    target: str
    message: str
    next_action: str
    def __init__(self, code: _Optional[str] = ..., severity: _Optional[str] = ..., target: _Optional[str] = ..., message: _Optional[str] = ..., next_action: _Optional[str] = ...) -> None: ...

class ReleaseHealth(_message.Message):
    __slots__ = ("release_id", "status", "observed_at", "publication_verified", "client_updates_healthy", "recovery_standing", "alerts", "known_durations_millis", "supported_controls", "unsupported_controls")
    class KnownDurationsMillisEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    RELEASE_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    PUBLICATION_VERIFIED_FIELD_NUMBER: _ClassVar[int]
    CLIENT_UPDATES_HEALTHY_FIELD_NUMBER: _ClassVar[int]
    RECOVERY_STANDING_FIELD_NUMBER: _ClassVar[int]
    ALERTS_FIELD_NUMBER: _ClassVar[int]
    KNOWN_DURATIONS_MILLIS_FIELD_NUMBER: _ClassVar[int]
    SUPPORTED_CONTROLS_FIELD_NUMBER: _ClassVar[int]
    UNSUPPORTED_CONTROLS_FIELD_NUMBER: _ClassVar[int]
    release_id: str
    status: str
    observed_at: _timestamp_pb2.Timestamp
    publication_verified: bool
    client_updates_healthy: bool
    recovery_standing: str
    alerts: _containers.RepeatedCompositeFieldContainer[ReleaseAlert]
    known_durations_millis: _containers.ScalarMap[str, int]
    supported_controls: _containers.RepeatedScalarFieldContainer[str]
    unsupported_controls: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, release_id: _Optional[str] = ..., status: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., publication_verified: _Optional[bool] = ..., client_updates_healthy: _Optional[bool] = ..., recovery_standing: _Optional[str] = ..., alerts: _Optional[_Iterable[_Union[ReleaseAlert, _Mapping]]] = ..., known_durations_millis: _Optional[_Mapping[str, int]] = ..., supported_controls: _Optional[_Iterable[str]] = ..., unsupported_controls: _Optional[_Iterable[str]] = ...) -> None: ...

class CandidateRecord(_message.Message):
    __slots__ = ("candidate_id", "candidate", "artifact_manifest_digest", "created_at")
    CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_MANIFEST_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    candidate_id: str
    candidate: Candidate
    artifact_manifest_digest: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, candidate_id: _Optional[str] = ..., candidate: _Optional[_Union[Candidate, _Mapping]] = ..., artifact_manifest_digest: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class DestinationRevisionRecord(_message.Message):
    __slots__ = ("destination_revision_id", "revision", "created_at")
    DESTINATION_REVISION_ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    destination_revision_id: str
    revision: DestinationRevision
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, destination_revision_id: _Optional[str] = ..., revision: _Optional[_Union[DestinationRevision, _Mapping]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ReviewRecord(_message.Message):
    __slots__ = ("review_id", "binding", "status", "approved_at", "revoked_at", "created_at")
    REVIEW_ID_FIELD_NUMBER: _ClassVar[int]
    BINDING_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    APPROVED_AT_FIELD_NUMBER: _ClassVar[int]
    REVOKED_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    review_id: str
    binding: ReviewBinding
    status: str
    approved_at: _timestamp_pb2.Timestamp
    revoked_at: _timestamp_pb2.Timestamp
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, review_id: _Optional[str] = ..., binding: _Optional[_Union[ReviewBinding, _Mapping]] = ..., status: _Optional[str] = ..., approved_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., revoked_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ReleaseDossier(_message.Message):
    __slots__ = ("schema_version", "generated_at", "release", "health", "candidate", "destination", "review", "missing_proof")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    RELEASE_FIELD_NUMBER: _ClassVar[int]
    HEALTH_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_FIELD_NUMBER: _ClassVar[int]
    REVIEW_FIELD_NUMBER: _ClassVar[int]
    MISSING_PROOF_FIELD_NUMBER: _ClassVar[int]
    schema_version: int
    generated_at: _timestamp_pb2.Timestamp
    release: ReleaseView
    health: ReleaseHealth
    candidate: CandidateRecord
    destination: DestinationRevisionRecord
    review: ReviewRecord
    missing_proof: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, schema_version: _Optional[int] = ..., generated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., release: _Optional[_Union[ReleaseView, _Mapping]] = ..., health: _Optional[_Union[ReleaseHealth, _Mapping]] = ..., candidate: _Optional[_Union[CandidateRecord, _Mapping]] = ..., destination: _Optional[_Union[DestinationRevisionRecord, _Mapping]] = ..., review: _Optional[_Union[ReviewRecord, _Mapping]] = ..., missing_proof: _Optional[_Iterable[str]] = ...) -> None: ...

class GetReleaseDossierRequest(_message.Message):
    __slots__ = ("release_id",)
    RELEASE_ID_FIELD_NUMBER: _ClassVar[int]
    release_id: str
    def __init__(self, release_id: _Optional[str] = ...) -> None: ...

class GetReleaseDossierResponse(_message.Message):
    __slots__ = ("dossier",)
    DOSSIER_FIELD_NUMBER: _ClassVar[int]
    dossier: ReleaseDossier
    def __init__(self, dossier: _Optional[_Union[ReleaseDossier, _Mapping]] = ...) -> None: ...

class ReverifyReleaseRequest(_message.Message):
    __slots__ = ("release_id", "deep")
    RELEASE_ID_FIELD_NUMBER: _ClassVar[int]
    DEEP_FIELD_NUMBER: _ClassVar[int]
    release_id: str
    deep: bool
    def __init__(self, release_id: _Optional[str] = ..., deep: _Optional[bool] = ...) -> None: ...

class ReverifyReleaseResponse(_message.Message):
    __slots__ = ("release",)
    RELEASE_FIELD_NUMBER: _ClassVar[int]
    release: ReleaseView
    def __init__(self, release: _Optional[_Union[ReleaseView, _Mapping]] = ...) -> None: ...

class ReconcileReleaseRequest(_message.Message):
    __slots__ = ("release_id", "deep")
    RELEASE_ID_FIELD_NUMBER: _ClassVar[int]
    DEEP_FIELD_NUMBER: _ClassVar[int]
    release_id: str
    deep: bool
    def __init__(self, release_id: _Optional[str] = ..., deep: _Optional[bool] = ...) -> None: ...

class ReconcileReleaseResponse(_message.Message):
    __slots__ = ("release",)
    RELEASE_FIELD_NUMBER: _ClassVar[int]
    release: ReleaseView
    def __init__(self, release: _Optional[_Union[ReleaseView, _Mapping]] = ...) -> None: ...

class RecoverReleaseRequest(_message.Message):
    __slots__ = ("release_id", "review_key", "candidate_id", "destination_revision_id", "action", "expected_predecessor_revision", "data_compatibility", "repair_artifact_ids", "repair_bundle_sha256", "idempotency_key", "confirmation", "dry_run")
    class RepairArtifactIdsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    RELEASE_ID_FIELD_NUMBER: _ClassVar[int]
    REVIEW_KEY_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_REVISION_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_PREDECESSOR_REVISION_FIELD_NUMBER: _ClassVar[int]
    DATA_COMPATIBILITY_FIELD_NUMBER: _ClassVar[int]
    REPAIR_ARTIFACT_IDS_FIELD_NUMBER: _ClassVar[int]
    REPAIR_BUNDLE_SHA256_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    CONFIRMATION_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    release_id: str
    review_key: str
    candidate_id: str
    destination_revision_id: str
    action: str
    expected_predecessor_revision: int
    data_compatibility: str
    repair_artifact_ids: _containers.ScalarMap[str, int]
    repair_bundle_sha256: str
    idempotency_key: str
    confirmation: str
    dry_run: bool
    def __init__(self, release_id: _Optional[str] = ..., review_key: _Optional[str] = ..., candidate_id: _Optional[str] = ..., destination_revision_id: _Optional[str] = ..., action: _Optional[str] = ..., expected_predecessor_revision: _Optional[int] = ..., data_compatibility: _Optional[str] = ..., repair_artifact_ids: _Optional[_Mapping[str, int]] = ..., repair_bundle_sha256: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., confirmation: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class RecoverReleaseResponse(_message.Message):
    __slots__ = ("release_id", "dry_run", "receipt")
    RELEASE_ID_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_FIELD_NUMBER: _ClassVar[int]
    release_id: str
    dry_run: bool
    receipt: RecoveryReceipt
    def __init__(self, release_id: _Optional[str] = ..., dry_run: _Optional[bool] = ..., receipt: _Optional[_Union[RecoveryReceipt, _Mapping]] = ...) -> None: ...

class StartReleaseRequest(_message.Message):
    __slots__ = ("profile_id", "channel", "git_commit_hash", "artifact_digest", "candidate_id", "destination_revision_id", "authorization_epoch", "idempotency_key", "readiness_review_key", "release_version", "release_notes", "platforms", "cloud_manifest_json", "cloud_deployment_name", "cloud_bundle_path", "cloud_bundle_sha256", "cloud_bundle_size_bytes", "cloud_run_preflight")
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    GIT_COMMIT_HASH_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_ID_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_REVISION_ID_FIELD_NUMBER: _ClassVar[int]
    AUTHORIZATION_EPOCH_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    READINESS_REVIEW_KEY_FIELD_NUMBER: _ClassVar[int]
    RELEASE_VERSION_FIELD_NUMBER: _ClassVar[int]
    RELEASE_NOTES_FIELD_NUMBER: _ClassVar[int]
    PLATFORMS_FIELD_NUMBER: _ClassVar[int]
    CLOUD_MANIFEST_JSON_FIELD_NUMBER: _ClassVar[int]
    CLOUD_DEPLOYMENT_NAME_FIELD_NUMBER: _ClassVar[int]
    CLOUD_BUNDLE_PATH_FIELD_NUMBER: _ClassVar[int]
    CLOUD_BUNDLE_SHA256_FIELD_NUMBER: _ClassVar[int]
    CLOUD_BUNDLE_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    CLOUD_RUN_PREFLIGHT_FIELD_NUMBER: _ClassVar[int]
    profile_id: str
    channel: str
    git_commit_hash: str
    artifact_digest: str
    candidate_id: str
    destination_revision_id: str
    authorization_epoch: int
    idempotency_key: str
    readiness_review_key: str
    release_version: str
    release_notes: str
    platforms: _containers.RepeatedScalarFieldContainer[str]
    cloud_manifest_json: str
    cloud_deployment_name: str
    cloud_bundle_path: str
    cloud_bundle_sha256: str
    cloud_bundle_size_bytes: int
    cloud_run_preflight: bool
    def __init__(self, profile_id: _Optional[str] = ..., channel: _Optional[str] = ..., git_commit_hash: _Optional[str] = ..., artifact_digest: _Optional[str] = ..., candidate_id: _Optional[str] = ..., destination_revision_id: _Optional[str] = ..., authorization_epoch: _Optional[int] = ..., idempotency_key: _Optional[str] = ..., readiness_review_key: _Optional[str] = ..., release_version: _Optional[str] = ..., release_notes: _Optional[str] = ..., platforms: _Optional[_Iterable[str]] = ..., cloud_manifest_json: _Optional[str] = ..., cloud_deployment_name: _Optional[str] = ..., cloud_bundle_path: _Optional[str] = ..., cloud_bundle_sha256: _Optional[str] = ..., cloud_bundle_size_bytes: _Optional[int] = ..., cloud_run_preflight: _Optional[bool] = ...) -> None: ...

class StartReleaseResponse(_message.Message):
    __slots__ = ("operation_id", "release_id", "status", "release", "operation")
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    RELEASE_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    RELEASE_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    release_id: str
    status: str
    release: ReleaseView
    operation: ReleaseOperation
    def __init__(self, operation_id: _Optional[str] = ..., release_id: _Optional[str] = ..., status: _Optional[str] = ..., release: _Optional[_Union[ReleaseView, _Mapping]] = ..., operation: _Optional[_Union[ReleaseOperation, _Mapping]] = ...) -> None: ...
