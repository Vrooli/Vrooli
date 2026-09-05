import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ReviewIdentity(_message.Message):
    __slots__ = ("scenario", "profile_id", "candidate_commit", "artifact_digest", "targets", "channel", "policy_version")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_COMMIT_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    profile_id: str
    candidate_commit: str
    artifact_digest: str
    targets: _containers.RepeatedScalarFieldContainer[str]
    channel: str
    policy_version: int
    def __init__(self, scenario: _Optional[str] = ..., profile_id: _Optional[str] = ..., candidate_commit: _Optional[str] = ..., artifact_digest: _Optional[str] = ..., targets: _Optional[_Iterable[str]] = ..., channel: _Optional[str] = ..., policy_version: _Optional[int] = ...) -> None: ...

class PrepareReviewRequest(_message.Message):
    __slots__ = ("scenario", "profile_id", "candidate_commit", "artifact_digest", "targets", "channel", "policy_version", "deliverable", "trigger", "facts")
    class FactsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_COMMIT_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    DELIVERABLE_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_FIELD_NUMBER: _ClassVar[int]
    FACTS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    profile_id: str
    candidate_commit: str
    artifact_digest: str
    targets: _containers.RepeatedScalarFieldContainer[str]
    channel: str
    policy_version: int
    deliverable: str
    trigger: str
    facts: _containers.ScalarMap[str, str]
    def __init__(self, scenario: _Optional[str] = ..., profile_id: _Optional[str] = ..., candidate_commit: _Optional[str] = ..., artifact_digest: _Optional[str] = ..., targets: _Optional[_Iterable[str]] = ..., channel: _Optional[str] = ..., policy_version: _Optional[int] = ..., deliverable: _Optional[str] = ..., trigger: _Optional[str] = ..., facts: _Optional[_Mapping[str, str]] = ...) -> None: ...

class ReviewResponse(_message.Message):
    __slots__ = ("review_key", "status", "identity", "comparison_mode", "predecessor_release_id", "predecessor_commit", "predecessor_artifact_digest", "goal_ref", "goal_closed_at", "approved_at", "approved_by", "findings", "evidence", "next_actions", "deduped")
    REVIEW_KEY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    COMPARISON_MODE_FIELD_NUMBER: _ClassVar[int]
    PREDECESSOR_RELEASE_ID_FIELD_NUMBER: _ClassVar[int]
    PREDECESSOR_COMMIT_FIELD_NUMBER: _ClassVar[int]
    PREDECESSOR_ARTIFACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    GOAL_REF_FIELD_NUMBER: _ClassVar[int]
    GOAL_CLOSED_AT_FIELD_NUMBER: _ClassVar[int]
    APPROVED_AT_FIELD_NUMBER: _ClassVar[int]
    APPROVED_BY_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    DEDUPED_FIELD_NUMBER: _ClassVar[int]
    review_key: str
    status: str
    identity: ReviewIdentity
    comparison_mode: str
    predecessor_release_id: str
    predecessor_commit: str
    predecessor_artifact_digest: str
    goal_ref: str
    goal_closed_at: _timestamp_pb2.Timestamp
    approved_at: _timestamp_pb2.Timestamp
    approved_by: str
    findings: _containers.RepeatedCompositeFieldContainer[_struct_pb2.Struct]
    evidence: _containers.RepeatedCompositeFieldContainer[_struct_pb2.Struct]
    next_actions: _containers.RepeatedScalarFieldContainer[str]
    deduped: bool
    def __init__(self, review_key: _Optional[str] = ..., status: _Optional[str] = ..., identity: _Optional[_Union[ReviewIdentity, _Mapping]] = ..., comparison_mode: _Optional[str] = ..., predecessor_release_id: _Optional[str] = ..., predecessor_commit: _Optional[str] = ..., predecessor_artifact_digest: _Optional[str] = ..., goal_ref: _Optional[str] = ..., goal_closed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., approved_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., approved_by: _Optional[str] = ..., findings: _Optional[_Iterable[_Union[_struct_pb2.Struct, _Mapping]]] = ..., evidence: _Optional[_Iterable[_Union[_struct_pb2.Struct, _Mapping]]] = ..., next_actions: _Optional[_Iterable[str]] = ..., deduped: _Optional[bool] = ...) -> None: ...

class GetReviewRequest(_message.Message):
    __slots__ = ("review_key",)
    REVIEW_KEY_FIELD_NUMBER: _ClassVar[int]
    review_key: str
    def __init__(self, review_key: _Optional[str] = ...) -> None: ...

class ListReviewsRequest(_message.Message):
    __slots__ = ("status", "page_size")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    status: str
    page_size: int
    def __init__(self, status: _Optional[str] = ..., page_size: _Optional[int] = ...) -> None: ...

class ListReviewsResponse(_message.Message):
    __slots__ = ("reviews", "count")
    REVIEWS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    reviews: _containers.RepeatedCompositeFieldContainer[ReviewResponse]
    count: int
    def __init__(self, reviews: _Optional[_Iterable[_Union[ReviewResponse, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class ListReviewWaiversRequest(_message.Message):
    __slots__ = ("review_key", "page_size")
    REVIEW_KEY_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    review_key: str
    page_size: int
    def __init__(self, review_key: _Optional[str] = ..., page_size: _Optional[int] = ...) -> None: ...

class ReviewWaiver(_message.Message):
    __slots__ = ("review_key", "criterion_id", "actor", "reason", "expires_at", "invalidation_trigger", "created_at")
    REVIEW_KEY_FIELD_NUMBER: _ClassVar[int]
    CRITERION_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    INVALIDATION_TRIGGER_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    review_key: str
    criterion_id: str
    actor: str
    reason: str
    expires_at: _timestamp_pb2.Timestamp
    invalidation_trigger: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, review_key: _Optional[str] = ..., criterion_id: _Optional[str] = ..., actor: _Optional[str] = ..., reason: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., invalidation_trigger: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListReviewWaiversResponse(_message.Message):
    __slots__ = ("waivers", "count")
    WAIVERS_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    waivers: _containers.RepeatedCompositeFieldContainer[ReviewWaiver]
    count: int
    def __init__(self, waivers: _Optional[_Iterable[_Union[ReviewWaiver, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...

class SynchronizeGoalClosureRequest(_message.Message):
    __slots__ = ("review_key",)
    REVIEW_KEY_FIELD_NUMBER: _ClassVar[int]
    review_key: str
    def __init__(self, review_key: _Optional[str] = ...) -> None: ...

class ApproveReviewRequest(_message.Message):
    __slots__ = ("review_key", "identity", "actor")
    REVIEW_KEY_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    review_key: str
    identity: ReviewIdentity
    actor: str
    def __init__(self, review_key: _Optional[str] = ..., identity: _Optional[_Union[ReviewIdentity, _Mapping]] = ..., actor: _Optional[str] = ...) -> None: ...

class CreateWaiverRequest(_message.Message):
    __slots__ = ("review_key", "criterion_id", "actor", "reason", "expires_at", "invalidation_trigger")
    REVIEW_KEY_FIELD_NUMBER: _ClassVar[int]
    CRITERION_ID_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    INVALIDATION_TRIGGER_FIELD_NUMBER: _ClassVar[int]
    review_key: str
    criterion_id: str
    actor: str
    reason: str
    expires_at: _timestamp_pb2.Timestamp
    invalidation_trigger: str
    def __init__(self, review_key: _Optional[str] = ..., criterion_id: _Optional[str] = ..., actor: _Optional[str] = ..., reason: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., invalidation_trigger: _Optional[str] = ...) -> None: ...

class RecordHumanCheckRequest(_message.Message):
    __slots__ = ("review_key", "criterion_id", "verdict", "actor", "evidence_reference", "reviewed_at")
    REVIEW_KEY_FIELD_NUMBER: _ClassVar[int]
    CRITERION_ID_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    REVIEWED_AT_FIELD_NUMBER: _ClassVar[int]
    review_key: str
    criterion_id: str
    verdict: str
    actor: str
    evidence_reference: str
    reviewed_at: _timestamp_pb2.Timestamp
    def __init__(self, review_key: _Optional[str] = ..., criterion_id: _Optional[str] = ..., verdict: _Optional[str] = ..., actor: _Optional[str] = ..., evidence_reference: _Optional[str] = ..., reviewed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CheckPolicyProjectionRequest(_message.Message):
    __slots__ = ("policy_json",)
    POLICY_JSON_FIELD_NUMBER: _ClassVar[int]
    policy_json: bytes
    def __init__(self, policy_json: _Optional[bytes] = ...) -> None: ...

class CheckPolicyProjectionResponse(_message.Message):
    __slots__ = ("policy_version", "criterion_count", "matches")
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    CRITERION_COUNT_FIELD_NUMBER: _ClassVar[int]
    MATCHES_FIELD_NUMBER: _ClassVar[int]
    policy_version: int
    criterion_count: int
    matches: bool
    def __init__(self, policy_version: _Optional[int] = ..., criterion_count: _Optional[int] = ..., matches: _Optional[bool] = ...) -> None: ...

class ReportEvidenceRequest(_message.Message):
    __slots__ = ("scenario", "profile_id", "candidate_commit", "artifact_digest", "targets", "channel", "policy_version", "criterion_id", "producer_binding", "producer_version", "status", "observed_at", "evidence_reference", "detail")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PROFILE_ID_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_COMMIT_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    POLICY_VERSION_FIELD_NUMBER: _ClassVar[int]
    CRITERION_ID_FIELD_NUMBER: _ClassVar[int]
    PRODUCER_BINDING_FIELD_NUMBER: _ClassVar[int]
    PRODUCER_VERSION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_AT_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFERENCE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    profile_id: str
    candidate_commit: str
    artifact_digest: str
    targets: _containers.RepeatedScalarFieldContainer[str]
    channel: str
    policy_version: int
    criterion_id: str
    producer_binding: str
    producer_version: str
    status: str
    observed_at: _timestamp_pb2.Timestamp
    evidence_reference: str
    detail: str
    def __init__(self, scenario: _Optional[str] = ..., profile_id: _Optional[str] = ..., candidate_commit: _Optional[str] = ..., artifact_digest: _Optional[str] = ..., targets: _Optional[_Iterable[str]] = ..., channel: _Optional[str] = ..., policy_version: _Optional[int] = ..., criterion_id: _Optional[str] = ..., producer_binding: _Optional[str] = ..., producer_version: _Optional[str] = ..., status: _Optional[str] = ..., observed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., evidence_reference: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class ReportEvidenceResponse(_message.Message):
    __slots__ = ("identity_key", "criterion_id", "producer_binding", "accepted")
    IDENTITY_KEY_FIELD_NUMBER: _ClassVar[int]
    CRITERION_ID_FIELD_NUMBER: _ClassVar[int]
    PRODUCER_BINDING_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    identity_key: str
    criterion_id: str
    producer_binding: str
    accepted: bool
    def __init__(self, identity_key: _Optional[str] = ..., criterion_id: _Optional[str] = ..., producer_binding: _Optional[str] = ..., accepted: _Optional[bool] = ...) -> None: ...
