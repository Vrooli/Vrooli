import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from swarm_manager.v1.api import transition_pb2 as _transition_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GetDevelopmentRequest(_message.Message):
    __slots__ = ("work_item",)
    WORK_ITEM_FIELD_NUMBER: _ClassVar[int]
    work_item: str
    def __init__(self, work_item: _Optional[str] = ...) -> None: ...

class ApproveDevelopmentRequest(_message.Message):
    __slots__ = ("proposal", "reviewed_digest", "expected_version", "reason")
    PROPOSAL_FIELD_NUMBER: _ClassVar[int]
    REVIEWED_DIGEST_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_VERSION_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    proposal: _transition_pb2.PreviewDevelopmentRequest
    reviewed_digest: str
    expected_version: int
    reason: str
    def __init__(self, proposal: _Optional[_Union[_transition_pb2.PreviewDevelopmentRequest, _Mapping]] = ..., reviewed_digest: _Optional[str] = ..., expected_version: _Optional[int] = ..., reason: _Optional[str] = ...) -> None: ...

class RevokeDevelopmentRequest(_message.Message):
    __slots__ = ("work_item", "expected_version", "reason")
    WORK_ITEM_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_VERSION_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    work_item: str
    expected_version: int
    reason: str
    def __init__(self, work_item: _Optional[str] = ..., expected_version: _Optional[int] = ..., reason: _Optional[str] = ...) -> None: ...

class AcceptDevelopmentRequest(_message.Message):
    __slots__ = ("work_item", "expected_version", "evidence_refs")
    class EvidenceRefsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    WORK_ITEM_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_VERSION_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    work_item: str
    expected_version: int
    evidence_refs: _containers.ScalarMap[str, str]
    def __init__(self, work_item: _Optional[str] = ..., expected_version: _Optional[int] = ..., evidence_refs: _Optional[_Mapping[str, str]] = ...) -> None: ...

class DevelopmentUsage(_message.Message):
    __slots__ = ("tokens", "wall_seconds")
    TOKENS_FIELD_NUMBER: _ClassVar[int]
    WALL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    tokens: int
    wall_seconds: int
    def __init__(self, tokens: _Optional[int] = ..., wall_seconds: _Optional[int] = ...) -> None: ...

class DevelopmentApproval(_message.Message):
    __slots__ = ("digest", "actor", "reason", "at")
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    AT_FIELD_NUMBER: _ClassVar[int]
    digest: str
    actor: str
    reason: str
    at: _timestamp_pb2.Timestamp
    def __init__(self, digest: _Optional[str] = ..., actor: _Optional[str] = ..., reason: _Optional[str] = ..., at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class DevelopmentAttempt(_message.Message):
    __slots__ = ("key", "digest", "mode", "reserved", "used", "execution_id", "started_at", "settled_at", "checkpoint", "workflow_digest", "grant_digest", "capability_revision", "selection_reason")
    KEY_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    RESERVED_FIELD_NUMBER: _ClassVar[int]
    USED_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    SETTLED_AT_FIELD_NUMBER: _ClassVar[int]
    CHECKPOINT_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_DIGEST_FIELD_NUMBER: _ClassVar[int]
    GRANT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_REVISION_FIELD_NUMBER: _ClassVar[int]
    SELECTION_REASON_FIELD_NUMBER: _ClassVar[int]
    key: str
    digest: str
    mode: str
    reserved: DevelopmentUsage
    used: DevelopmentUsage
    execution_id: str
    started_at: _timestamp_pb2.Timestamp
    settled_at: _timestamp_pb2.Timestamp
    checkpoint: str
    workflow_digest: str
    grant_digest: str
    capability_revision: str
    selection_reason: str
    def __init__(self, key: _Optional[str] = ..., digest: _Optional[str] = ..., mode: _Optional[str] = ..., reserved: _Optional[_Union[DevelopmentUsage, _Mapping]] = ..., used: _Optional[_Union[DevelopmentUsage, _Mapping]] = ..., execution_id: _Optional[str] = ..., started_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., settled_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., checkpoint: _Optional[str] = ..., workflow_digest: _Optional[str] = ..., grant_digest: _Optional[str] = ..., capability_revision: _Optional[str] = ..., selection_reason: _Optional[str] = ...) -> None: ...

class DevelopmentEvidence(_message.Message):
    __slots__ = ("outcome_id", "source", "receipt_id", "digest", "execution_id", "subject_revision", "observed_at", "resolver_id", "receipt_schema", "cohort", "fresh_until")
    OUTCOME_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_ID_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_REVISION_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    RESOLVER_ID_FIELD_NUMBER: _ClassVar[int]
    RECEIPT_SCHEMA_FIELD_NUMBER: _ClassVar[int]
    COHORT_FIELD_NUMBER: _ClassVar[int]
    FRESH_UNTIL_FIELD_NUMBER: _ClassVar[int]
    outcome_id: str
    source: str
    receipt_id: str
    digest: str
    execution_id: str
    subject_revision: str
    observed_at: _timestamp_pb2.Timestamp
    resolver_id: str
    receipt_schema: str
    cohort: str
    fresh_until: _timestamp_pb2.Timestamp
    def __init__(self, outcome_id: _Optional[str] = ..., source: _Optional[str] = ..., receipt_id: _Optional[str] = ..., digest: _Optional[str] = ..., execution_id: _Optional[str] = ..., subject_revision: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., resolver_id: _Optional[str] = ..., receipt_schema: _Optional[str] = ..., cohort: _Optional[str] = ..., fresh_until: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class DevelopmentCheckpointReference(_message.Message):
    __slots__ = ("kind", "value", "digest")
    KIND_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    kind: str
    value: str
    digest: str
    def __init__(self, kind: _Optional[str] = ..., value: _Optional[str] = ..., digest: _Optional[str] = ...) -> None: ...

class DevelopmentCampaignCheckpoint(_message.Message):
    __slots__ = ("digest", "approval_digest", "attempt_key", "owner_execution_id", "pending", "required_outcome_ids", "completed_outcome_ids", "remaining_outcome_ids", "last_outcome", "last_checkpoint", "no_progress_cycles")
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    APPROVAL_DIGEST_FIELD_NUMBER: _ClassVar[int]
    ATTEMPT_KEY_FIELD_NUMBER: _ClassVar[int]
    OWNER_EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PENDING_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_OUTCOME_IDS_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_OUTCOME_IDS_FIELD_NUMBER: _ClassVar[int]
    REMAINING_OUTCOME_IDS_FIELD_NUMBER: _ClassVar[int]
    LAST_OUTCOME_FIELD_NUMBER: _ClassVar[int]
    LAST_CHECKPOINT_FIELD_NUMBER: _ClassVar[int]
    NO_PROGRESS_CYCLES_FIELD_NUMBER: _ClassVar[int]
    digest: str
    approval_digest: str
    attempt_key: str
    owner_execution_id: str
    pending: bool
    required_outcome_ids: _containers.RepeatedScalarFieldContainer[str]
    completed_outcome_ids: _containers.RepeatedScalarFieldContainer[str]
    remaining_outcome_ids: _containers.RepeatedScalarFieldContainer[str]
    last_outcome: str
    last_checkpoint: DevelopmentCheckpointReference
    no_progress_cycles: int
    def __init__(self, digest: _Optional[str] = ..., approval_digest: _Optional[str] = ..., attempt_key: _Optional[str] = ..., owner_execution_id: _Optional[str] = ..., pending: _Optional[bool] = ..., required_outcome_ids: _Optional[_Iterable[str]] = ..., completed_outcome_ids: _Optional[_Iterable[str]] = ..., remaining_outcome_ids: _Optional[_Iterable[str]] = ..., last_outcome: _Optional[str] = ..., last_checkpoint: _Optional[_Union[DevelopmentCheckpointReference, _Mapping]] = ..., no_progress_cycles: _Optional[int] = ...) -> None: ...

class DevelopmentCancellationIntent(_message.Message):
    __slots__ = ("operation_id", "reason", "state", "requested_at", "ack_at")
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_AT_FIELD_NUMBER: _ClassVar[int]
    ACK_AT_FIELD_NUMBER: _ClassVar[int]
    operation_id: str
    reason: str
    state: str
    requested_at: _timestamp_pb2.Timestamp
    ack_at: _timestamp_pb2.Timestamp
    def __init__(self, operation_id: _Optional[str] = ..., reason: _Optional[str] = ..., state: _Optional[str] = ..., requested_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., ack_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class DevelopmentResponse(_message.Message):
    __slots__ = ("work_item", "work_shape", "version", "digest", "status", "approvals", "attempts", "used", "reserved", "checkpoint", "evidence", "accepted_by", "accepted_at", "stop_reason", "approved_proposal", "artifacts", "goal_message", "launch_blockers", "campaign", "cancellation")
    WORK_ITEM_FIELD_NUMBER: _ClassVar[int]
    WORK_SHAPE_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    APPROVALS_FIELD_NUMBER: _ClassVar[int]
    ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    USED_FIELD_NUMBER: _ClassVar[int]
    RESERVED_FIELD_NUMBER: _ClassVar[int]
    CHECKPOINT_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_BY_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_AT_FIELD_NUMBER: _ClassVar[int]
    STOP_REASON_FIELD_NUMBER: _ClassVar[int]
    APPROVED_PROPOSAL_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    GOAL_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    LAUNCH_BLOCKERS_FIELD_NUMBER: _ClassVar[int]
    CAMPAIGN_FIELD_NUMBER: _ClassVar[int]
    CANCELLATION_FIELD_NUMBER: _ClassVar[int]
    work_item: str
    work_shape: str
    version: int
    digest: str
    status: str
    approvals: _containers.RepeatedCompositeFieldContainer[DevelopmentApproval]
    attempts: _containers.RepeatedCompositeFieldContainer[DevelopmentAttempt]
    used: DevelopmentUsage
    reserved: DevelopmentUsage
    checkpoint: str
    evidence: _containers.RepeatedCompositeFieldContainer[DevelopmentEvidence]
    accepted_by: str
    accepted_at: _timestamp_pb2.Timestamp
    stop_reason: str
    approved_proposal: _transition_pb2.PreviewDevelopmentRequest
    artifacts: _containers.RepeatedCompositeFieldContainer[_transition_pb2.DevelopmentArtifact]
    goal_message: str
    launch_blockers: _containers.RepeatedScalarFieldContainer[str]
    campaign: DevelopmentCampaignCheckpoint
    cancellation: DevelopmentCancellationIntent
    def __init__(self, work_item: _Optional[str] = ..., work_shape: _Optional[str] = ..., version: _Optional[int] = ..., digest: _Optional[str] = ..., status: _Optional[str] = ..., approvals: _Optional[_Iterable[_Union[DevelopmentApproval, _Mapping]]] = ..., attempts: _Optional[_Iterable[_Union[DevelopmentAttempt, _Mapping]]] = ..., used: _Optional[_Union[DevelopmentUsage, _Mapping]] = ..., reserved: _Optional[_Union[DevelopmentUsage, _Mapping]] = ..., checkpoint: _Optional[str] = ..., evidence: _Optional[_Iterable[_Union[DevelopmentEvidence, _Mapping]]] = ..., accepted_by: _Optional[str] = ..., accepted_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., stop_reason: _Optional[str] = ..., approved_proposal: _Optional[_Union[_transition_pb2.PreviewDevelopmentRequest, _Mapping]] = ..., artifacts: _Optional[_Iterable[_Union[_transition_pb2.DevelopmentArtifact, _Mapping]]] = ..., goal_message: _Optional[str] = ..., launch_blockers: _Optional[_Iterable[str]] = ..., campaign: _Optional[_Union[DevelopmentCampaignCheckpoint, _Mapping]] = ..., cancellation: _Optional[_Union[DevelopmentCancellationIntent, _Mapping]] = ...) -> None: ...

class GetDevelopmentArtifactRequest(_message.Message):
    __slots__ = ("work_item", "digest", "path")
    WORK_ITEM_FIELD_NUMBER: _ClassVar[int]
    DIGEST_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    work_item: str
    digest: str
    path: str
    def __init__(self, work_item: _Optional[str] = ..., digest: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class GetDevelopmentArtifactResponse(_message.Message):
    __slots__ = ("artifact", "content")
    ARTIFACT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    artifact: _transition_pb2.DevelopmentArtifact
    content: bytes
    def __init__(self, artifact: _Optional[_Union[_transition_pb2.DevelopmentArtifact, _Mapping]] = ..., content: _Optional[bytes] = ...) -> None: ...
