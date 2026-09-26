import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class MemberRole(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    MEMBER_ROLE_UNSPECIFIED: _ClassVar[MemberRole]
    MEMBER_ROLE_PRIMARY: _ClassVar[MemberRole]
    MEMBER_ROLE_SUPPORTING: _ClassVar[MemberRole]
    MEMBER_ROLE_EXPERIMENT: _ClassVar[MemberRole]

class MemberState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    MEMBER_STATE_UNSPECIFIED: _ClassVar[MemberState]
    MEMBER_STATE_PENDING: _ClassVar[MemberState]
    MEMBER_STATE_RUNNING: _ClassVar[MemberState]
    MEMBER_STATE_SUCCEEDED: _ClassVar[MemberState]
    MEMBER_STATE_FAILED: _ClassVar[MemberState]
    MEMBER_STATE_BLOCKED: _ClassVar[MemberState]
    MEMBER_STATE_SKIPPED: _ClassVar[MemberState]

class ClaimKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CLAIM_KIND_UNSPECIFIED: _ClassVar[ClaimKind]
    CLAIM_KIND_PATH: _ClassVar[ClaimKind]
    CLAIM_KIND_API_CONTRACT: _ClassVar[ClaimKind]
    CLAIM_KIND_GENERATED_OUTPUT: _ClassVar[ClaimKind]
    CLAIM_KIND_SCHEMA: _ClassVar[ClaimKind]
    CLAIM_KIND_RUNTIME_PLANT: _ClassVar[ClaimKind]
    CLAIM_KIND_VALIDATION_TARGET: _ClassVar[ClaimKind]
    CLAIM_KIND_UNKNOWN_INTERACTION: _ClassVar[ClaimKind]

class ClaimAccess(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CLAIM_ACCESS_UNSPECIFIED: _ClassVar[ClaimAccess]
    CLAIM_ACCESS_READ: _ClassVar[ClaimAccess]
    CLAIM_ACCESS_SHARED_WRITE: _ClassVar[ClaimAccess]
    CLAIM_ACCESS_EXCLUSIVE_WRITE: _ClassVar[ClaimAccess]

class EdgeKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EDGE_KIND_UNSPECIFIED: _ClassVar[EdgeKind]
    EDGE_KIND_DEPENDENCY: _ClassVar[EdgeKind]
    EDGE_KIND_CONFLICT: _ClassVar[EdgeKind]

class EdgeProvenance(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EDGE_PROVENANCE_UNSPECIFIED: _ClassVar[EdgeProvenance]
    EDGE_PROVENANCE_EXPLICIT: _ClassVar[EdgeProvenance]
    EDGE_PROVENANCE_INFERRED: _ClassVar[EdgeProvenance]
    EDGE_PROVENANCE_SUGGESTED: _ClassVar[EdgeProvenance]
    EDGE_PROVENANCE_CORRECTED: _ClassVar[EdgeProvenance]
    EDGE_PROVENANCE_MIGRATED: _ClassVar[EdgeProvenance]

class ReviewDecision(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    REVIEW_DECISION_UNSPECIFIED: _ClassVar[ReviewDecision]
    REVIEW_DECISION_APPROVED: _ClassVar[ReviewDecision]
    REVIEW_DECISION_CORRECTED: _ClassVar[ReviewDecision]
    REVIEW_DECISION_REJECTED: _ClassVar[ReviewDecision]

class FamilyExecutionState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FAMILY_EXECUTION_STATE_UNSPECIFIED: _ClassVar[FamilyExecutionState]
    FAMILY_EXECUTION_STATE_PENDING: _ClassVar[FamilyExecutionState]
    FAMILY_EXECUTION_STATE_RUNNING: _ClassVar[FamilyExecutionState]
    FAMILY_EXECUTION_STATE_BLOCKED: _ClassVar[FamilyExecutionState]
    FAMILY_EXECUTION_STATE_SUCCEEDED: _ClassVar[FamilyExecutionState]
    FAMILY_EXECUTION_STATE_FAILED: _ClassVar[FamilyExecutionState]
MEMBER_ROLE_UNSPECIFIED: MemberRole
MEMBER_ROLE_PRIMARY: MemberRole
MEMBER_ROLE_SUPPORTING: MemberRole
MEMBER_ROLE_EXPERIMENT: MemberRole
MEMBER_STATE_UNSPECIFIED: MemberState
MEMBER_STATE_PENDING: MemberState
MEMBER_STATE_RUNNING: MemberState
MEMBER_STATE_SUCCEEDED: MemberState
MEMBER_STATE_FAILED: MemberState
MEMBER_STATE_BLOCKED: MemberState
MEMBER_STATE_SKIPPED: MemberState
CLAIM_KIND_UNSPECIFIED: ClaimKind
CLAIM_KIND_PATH: ClaimKind
CLAIM_KIND_API_CONTRACT: ClaimKind
CLAIM_KIND_GENERATED_OUTPUT: ClaimKind
CLAIM_KIND_SCHEMA: ClaimKind
CLAIM_KIND_RUNTIME_PLANT: ClaimKind
CLAIM_KIND_VALIDATION_TARGET: ClaimKind
CLAIM_KIND_UNKNOWN_INTERACTION: ClaimKind
CLAIM_ACCESS_UNSPECIFIED: ClaimAccess
CLAIM_ACCESS_READ: ClaimAccess
CLAIM_ACCESS_SHARED_WRITE: ClaimAccess
CLAIM_ACCESS_EXCLUSIVE_WRITE: ClaimAccess
EDGE_KIND_UNSPECIFIED: EdgeKind
EDGE_KIND_DEPENDENCY: EdgeKind
EDGE_KIND_CONFLICT: EdgeKind
EDGE_PROVENANCE_UNSPECIFIED: EdgeProvenance
EDGE_PROVENANCE_EXPLICIT: EdgeProvenance
EDGE_PROVENANCE_INFERRED: EdgeProvenance
EDGE_PROVENANCE_SUGGESTED: EdgeProvenance
EDGE_PROVENANCE_CORRECTED: EdgeProvenance
EDGE_PROVENANCE_MIGRATED: EdgeProvenance
REVIEW_DECISION_UNSPECIFIED: ReviewDecision
REVIEW_DECISION_APPROVED: ReviewDecision
REVIEW_DECISION_CORRECTED: ReviewDecision
REVIEW_DECISION_REJECTED: ReviewDecision
FAMILY_EXECUTION_STATE_UNSPECIFIED: FamilyExecutionState
FAMILY_EXECUTION_STATE_PENDING: FamilyExecutionState
FAMILY_EXECUTION_STATE_RUNNING: FamilyExecutionState
FAMILY_EXECUTION_STATE_BLOCKED: FamilyExecutionState
FAMILY_EXECUTION_STATE_SUCCEEDED: FamilyExecutionState
FAMILY_EXECUTION_STATE_FAILED: FamilyExecutionState

class FamilyPolicy(_message.Message):
    __slots__ = ("maximum_parallel_plans", "unknown_interactions_sequential", "require_review_before_launch", "validation_policy")
    MAXIMUM_PARALLEL_PLANS_FIELD_NUMBER: _ClassVar[int]
    UNKNOWN_INTERACTIONS_SEQUENTIAL_FIELD_NUMBER: _ClassVar[int]
    REQUIRE_REVIEW_BEFORE_LAUNCH_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_POLICY_FIELD_NUMBER: _ClassVar[int]
    maximum_parallel_plans: int
    unknown_interactions_sequential: bool
    require_review_before_launch: bool
    validation_policy: str
    def __init__(self, maximum_parallel_plans: _Optional[int] = ..., unknown_interactions_sequential: _Optional[bool] = ..., require_review_before_launch: _Optional[bool] = ..., validation_policy: _Optional[str] = ...) -> None: ...

class FamilyMember(_message.Message):
    __slots__ = ("plan_id", "role", "state", "execution_id", "detail", "source_revision", "admission_key")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REVISION_FIELD_NUMBER: _ClassVar[int]
    ADMISSION_KEY_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    role: MemberRole
    state: MemberState
    execution_id: str
    detail: str
    source_revision: int
    admission_key: str
    def __init__(self, plan_id: _Optional[str] = ..., role: _Optional[_Union[MemberRole, str]] = ..., state: _Optional[_Union[MemberState, str]] = ..., execution_id: _Optional[str] = ..., detail: _Optional[str] = ..., source_revision: _Optional[int] = ..., admission_key: _Optional[str] = ...) -> None: ...

class ResourceClaim(_message.Message):
    __slots__ = ("claim_id", "plan_id", "kind", "access", "resource", "source", "detail", "resolved")
    CLAIM_ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    ACCESS_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_FIELD_NUMBER: _ClassVar[int]
    claim_id: str
    plan_id: str
    kind: ClaimKind
    access: ClaimAccess
    resource: str
    source: str
    detail: str
    resolved: bool
    def __init__(self, claim_id: _Optional[str] = ..., plan_id: _Optional[str] = ..., kind: _Optional[_Union[ClaimKind, str]] = ..., access: _Optional[_Union[ClaimAccess, str]] = ..., resource: _Optional[str] = ..., source: _Optional[str] = ..., detail: _Optional[str] = ..., resolved: _Optional[bool] = ...) -> None: ...

class FamilyEdge(_message.Message):
    __slots__ = ("from_plan_id", "to_plan_id", "kind", "provenance", "reason", "claim_ids")
    FROM_PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    TO_PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    CLAIM_IDS_FIELD_NUMBER: _ClassVar[int]
    from_plan_id: str
    to_plan_id: str
    kind: EdgeKind
    provenance: EdgeProvenance
    reason: str
    claim_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, from_plan_id: _Optional[str] = ..., to_plan_id: _Optional[str] = ..., kind: _Optional[_Union[EdgeKind, str]] = ..., provenance: _Optional[_Union[EdgeProvenance, str]] = ..., reason: _Optional[str] = ..., claim_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class FrontierBatch(_message.Message):
    __slots__ = ("ordinal", "plan_ids")
    ORDINAL_FIELD_NUMBER: _ClassVar[int]
    PLAN_IDS_FIELD_NUMBER: _ClassVar[int]
    ordinal: int
    plan_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, ordinal: _Optional[int] = ..., plan_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class GraphRevision(_message.Message):
    __slots__ = ("revision", "edges", "proposed_frontier", "cyclic", "diagnostics", "created_at")
    REVISION_FIELD_NUMBER: _ClassVar[int]
    EDGES_FIELD_NUMBER: _ClassVar[int]
    PROPOSED_FRONTIER_FIELD_NUMBER: _ClassVar[int]
    CYCLIC_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTICS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    revision: int
    edges: _containers.RepeatedCompositeFieldContainer[FamilyEdge]
    proposed_frontier: _containers.RepeatedCompositeFieldContainer[FrontierBatch]
    cyclic: bool
    diagnostics: _containers.RepeatedScalarFieldContainer[str]
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, revision: _Optional[int] = ..., edges: _Optional[_Iterable[_Union[FamilyEdge, _Mapping]]] = ..., proposed_frontier: _Optional[_Iterable[_Union[FrontierBatch, _Mapping]]] = ..., cyclic: _Optional[bool] = ..., diagnostics: _Optional[_Iterable[str]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class GraphReview(_message.Message):
    __slots__ = ("graph_revision", "decision", "reviewer", "rationale", "corrected_edges", "reviewed_at")
    GRAPH_REVISION_FIELD_NUMBER: _ClassVar[int]
    DECISION_FIELD_NUMBER: _ClassVar[int]
    REVIEWER_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    CORRECTED_EDGES_FIELD_NUMBER: _ClassVar[int]
    REVIEWED_AT_FIELD_NUMBER: _ClassVar[int]
    graph_revision: int
    decision: ReviewDecision
    reviewer: str
    rationale: str
    corrected_edges: _containers.RepeatedCompositeFieldContainer[FamilyEdge]
    reviewed_at: _timestamp_pb2.Timestamp
    def __init__(self, graph_revision: _Optional[int] = ..., decision: _Optional[_Union[ReviewDecision, str]] = ..., reviewer: _Optional[str] = ..., rationale: _Optional[str] = ..., corrected_edges: _Optional[_Iterable[_Union[FamilyEdge, _Mapping]]] = ..., reviewed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class PlanFamily(_message.Message):
    __slots__ = ("schema_version", "family_id", "slug", "outcome", "shared_context", "policy", "members", "claims", "graph", "review", "revision", "created_at", "updated_at", "execution_state", "execution_id", "active_graph_revision", "last_activity_at")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    FAMILY_ID_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    SHARED_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    POLICY_FIELD_NUMBER: _ClassVar[int]
    MEMBERS_FIELD_NUMBER: _ClassVar[int]
    CLAIMS_FIELD_NUMBER: _ClassVar[int]
    GRAPH_FIELD_NUMBER: _ClassVar[int]
    REVIEW_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_STATE_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_GRAPH_REVISION_FIELD_NUMBER: _ClassVar[int]
    LAST_ACTIVITY_AT_FIELD_NUMBER: _ClassVar[int]
    schema_version: int
    family_id: str
    slug: str
    outcome: str
    shared_context: str
    policy: FamilyPolicy
    members: _containers.RepeatedCompositeFieldContainer[FamilyMember]
    claims: _containers.RepeatedCompositeFieldContainer[ResourceClaim]
    graph: GraphRevision
    review: GraphReview
    revision: int
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    execution_state: FamilyExecutionState
    execution_id: str
    active_graph_revision: int
    last_activity_at: _timestamp_pb2.Timestamp
    def __init__(self, schema_version: _Optional[int] = ..., family_id: _Optional[str] = ..., slug: _Optional[str] = ..., outcome: _Optional[str] = ..., shared_context: _Optional[str] = ..., policy: _Optional[_Union[FamilyPolicy, _Mapping]] = ..., members: _Optional[_Iterable[_Union[FamilyMember, _Mapping]]] = ..., claims: _Optional[_Iterable[_Union[ResourceClaim, _Mapping]]] = ..., graph: _Optional[_Union[GraphRevision, _Mapping]] = ..., review: _Optional[_Union[GraphReview, _Mapping]] = ..., revision: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., execution_state: _Optional[_Union[FamilyExecutionState, str]] = ..., execution_id: _Optional[str] = ..., active_graph_revision: _Optional[int] = ..., last_activity_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CreateFamilyRequest(_message.Message):
    __slots__ = ("slug", "outcome", "shared_context", "policy")
    SLUG_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    SHARED_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    POLICY_FIELD_NUMBER: _ClassVar[int]
    slug: str
    outcome: str
    shared_context: str
    policy: FamilyPolicy
    def __init__(self, slug: _Optional[str] = ..., outcome: _Optional[str] = ..., shared_context: _Optional[str] = ..., policy: _Optional[_Union[FamilyPolicy, _Mapping]] = ...) -> None: ...

class CreateFamilyResponse(_message.Message):
    __slots__ = ("family",)
    FAMILY_FIELD_NUMBER: _ClassVar[int]
    family: PlanFamily
    def __init__(self, family: _Optional[_Union[PlanFamily, _Mapping]] = ...) -> None: ...

class GetFamilyRequest(_message.Message):
    __slots__ = ("family_id",)
    FAMILY_ID_FIELD_NUMBER: _ClassVar[int]
    family_id: str
    def __init__(self, family_id: _Optional[str] = ...) -> None: ...

class GetFamilyResponse(_message.Message):
    __slots__ = ("family",)
    FAMILY_FIELD_NUMBER: _ClassVar[int]
    family: PlanFamily
    def __init__(self, family: _Optional[_Union[PlanFamily, _Mapping]] = ...) -> None: ...

class ListFamiliesRequest(_message.Message):
    __slots__ = ("page_size", "page_token")
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    page_size: int
    page_token: str
    def __init__(self, page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListFamiliesResponse(_message.Message):
    __slots__ = ("families", "next_page_token")
    FAMILIES_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    families: _containers.RepeatedCompositeFieldContainer[PlanFamily]
    next_page_token: str
    def __init__(self, families: _Optional[_Iterable[_Union[PlanFamily, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class UpdateFamilyRequest(_message.Message):
    __slots__ = ("family_id", "expected_revision", "outcome", "shared_context", "policy")
    FAMILY_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    SHARED_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    POLICY_FIELD_NUMBER: _ClassVar[int]
    family_id: str
    expected_revision: int
    outcome: str
    shared_context: str
    policy: FamilyPolicy
    def __init__(self, family_id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., outcome: _Optional[str] = ..., shared_context: _Optional[str] = ..., policy: _Optional[_Union[FamilyPolicy, _Mapping]] = ...) -> None: ...

class UpdateFamilyResponse(_message.Message):
    __slots__ = ("family",)
    FAMILY_FIELD_NUMBER: _ClassVar[int]
    family: PlanFamily
    def __init__(self, family: _Optional[_Union[PlanFamily, _Mapping]] = ...) -> None: ...

class PutMemberRequest(_message.Message):
    __slots__ = ("family_id", "expected_revision", "member")
    FAMILY_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    MEMBER_FIELD_NUMBER: _ClassVar[int]
    family_id: str
    expected_revision: int
    member: FamilyMember
    def __init__(self, family_id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., member: _Optional[_Union[FamilyMember, _Mapping]] = ...) -> None: ...

class PutMemberResponse(_message.Message):
    __slots__ = ("family",)
    FAMILY_FIELD_NUMBER: _ClassVar[int]
    family: PlanFamily
    def __init__(self, family: _Optional[_Union[PlanFamily, _Mapping]] = ...) -> None: ...

class RemoveMemberRequest(_message.Message):
    __slots__ = ("family_id", "expected_revision", "plan_id")
    FAMILY_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    family_id: str
    expected_revision: int
    plan_id: str
    def __init__(self, family_id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., plan_id: _Optional[str] = ...) -> None: ...

class RemoveMemberResponse(_message.Message):
    __slots__ = ("family",)
    FAMILY_FIELD_NUMBER: _ClassVar[int]
    family: PlanFamily
    def __init__(self, family: _Optional[_Union[PlanFamily, _Mapping]] = ...) -> None: ...

class PutClaimRequest(_message.Message):
    __slots__ = ("family_id", "expected_revision", "claim")
    FAMILY_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    CLAIM_FIELD_NUMBER: _ClassVar[int]
    family_id: str
    expected_revision: int
    claim: ResourceClaim
    def __init__(self, family_id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., claim: _Optional[_Union[ResourceClaim, _Mapping]] = ...) -> None: ...

class PutClaimResponse(_message.Message):
    __slots__ = ("family",)
    FAMILY_FIELD_NUMBER: _ClassVar[int]
    family: PlanFamily
    def __init__(self, family: _Optional[_Union[PlanFamily, _Mapping]] = ...) -> None: ...

class RemoveClaimRequest(_message.Message):
    __slots__ = ("family_id", "expected_revision", "claim_id")
    FAMILY_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    CLAIM_ID_FIELD_NUMBER: _ClassVar[int]
    family_id: str
    expected_revision: int
    claim_id: str
    def __init__(self, family_id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., claim_id: _Optional[str] = ...) -> None: ...

class RemoveClaimResponse(_message.Message):
    __slots__ = ("family",)
    FAMILY_FIELD_NUMBER: _ClassVar[int]
    family: PlanFamily
    def __init__(self, family: _Optional[_Union[PlanFamily, _Mapping]] = ...) -> None: ...

class ProposeGraphRequest(_message.Message):
    __slots__ = ("family_id", "expected_revision", "explicit_edges")
    FAMILY_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    EXPLICIT_EDGES_FIELD_NUMBER: _ClassVar[int]
    family_id: str
    expected_revision: int
    explicit_edges: _containers.RepeatedCompositeFieldContainer[FamilyEdge]
    def __init__(self, family_id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., explicit_edges: _Optional[_Iterable[_Union[FamilyEdge, _Mapping]]] = ...) -> None: ...

class ProposeGraphResponse(_message.Message):
    __slots__ = ("family",)
    FAMILY_FIELD_NUMBER: _ClassVar[int]
    family: PlanFamily
    def __init__(self, family: _Optional[_Union[PlanFamily, _Mapping]] = ...) -> None: ...

class ReviewGraphRequest(_message.Message):
    __slots__ = ("family_id", "expected_revision", "graph_revision", "decision", "reviewer", "rationale", "corrected_edges")
    FAMILY_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    GRAPH_REVISION_FIELD_NUMBER: _ClassVar[int]
    DECISION_FIELD_NUMBER: _ClassVar[int]
    REVIEWER_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    CORRECTED_EDGES_FIELD_NUMBER: _ClassVar[int]
    family_id: str
    expected_revision: int
    graph_revision: int
    decision: ReviewDecision
    reviewer: str
    rationale: str
    corrected_edges: _containers.RepeatedCompositeFieldContainer[FamilyEdge]
    def __init__(self, family_id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., graph_revision: _Optional[int] = ..., decision: _Optional[_Union[ReviewDecision, str]] = ..., reviewer: _Optional[str] = ..., rationale: _Optional[str] = ..., corrected_edges: _Optional[_Iterable[_Union[FamilyEdge, _Mapping]]] = ...) -> None: ...

class ReviewGraphResponse(_message.Message):
    __slots__ = ("family",)
    FAMILY_FIELD_NUMBER: _ClassVar[int]
    family: PlanFamily
    def __init__(self, family: _Optional[_Union[PlanFamily, _Mapping]] = ...) -> None: ...

class GetFrontierRequest(_message.Message):
    __slots__ = ("family_id",)
    FAMILY_ID_FIELD_NUMBER: _ClassVar[int]
    family_id: str
    def __init__(self, family_id: _Optional[str] = ...) -> None: ...

class GetFrontierResponse(_message.Message):
    __slots__ = ("graph_revision", "batches", "launchable", "diagnostics", "review", "projected_batches")
    GRAPH_REVISION_FIELD_NUMBER: _ClassVar[int]
    BATCHES_FIELD_NUMBER: _ClassVar[int]
    LAUNCHABLE_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTICS_FIELD_NUMBER: _ClassVar[int]
    REVIEW_FIELD_NUMBER: _ClassVar[int]
    PROJECTED_BATCHES_FIELD_NUMBER: _ClassVar[int]
    graph_revision: int
    batches: _containers.RepeatedCompositeFieldContainer[FrontierBatch]
    launchable: bool
    diagnostics: _containers.RepeatedScalarFieldContainer[str]
    review: GraphReview
    projected_batches: _containers.RepeatedCompositeFieldContainer[FrontierBatch]
    def __init__(self, graph_revision: _Optional[int] = ..., batches: _Optional[_Iterable[_Union[FrontierBatch, _Mapping]]] = ..., launchable: _Optional[bool] = ..., diagnostics: _Optional[_Iterable[str]] = ..., review: _Optional[_Union[GraphReview, _Mapping]] = ..., projected_batches: _Optional[_Iterable[_Union[FrontierBatch, _Mapping]]] = ...) -> None: ...

class RenderFamilyRequest(_message.Message):
    __slots__ = ("family_id",)
    FAMILY_ID_FIELD_NUMBER: _ClassVar[int]
    family_id: str
    def __init__(self, family_id: _Optional[str] = ...) -> None: ...

class RenderFamilyResponse(_message.Message):
    __slots__ = ("family", "markdown")
    FAMILY_FIELD_NUMBER: _ClassVar[int]
    MARKDOWN_FIELD_NUMBER: _ClassVar[int]
    family: PlanFamily
    markdown: str
    def __init__(self, family: _Optional[_Union[PlanFamily, _Mapping]] = ..., markdown: _Optional[str] = ...) -> None: ...
