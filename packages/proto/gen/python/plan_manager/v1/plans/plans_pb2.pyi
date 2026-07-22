from google.protobuf import field_mask_pb2 as _field_mask_pb2
from plan_manager.v1.shared import model_pb2 as _model_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CandidateRevisionState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CANDIDATE_REVISION_STATE_UNSPECIFIED: _ClassVar[CandidateRevisionState]
    CANDIDATE_REVISION_STATE_PENDING: _ClassVar[CandidateRevisionState]
    CANDIDATE_REVISION_STATE_APPLIED: _ClassVar[CandidateRevisionState]
    CANDIDATE_REVISION_STATE_DISCARDED: _ClassVar[CandidateRevisionState]
    CANDIDATE_REVISION_STATE_EXPIRED: _ClassVar[CandidateRevisionState]

class ReconcileConflictPolicy(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RECONCILE_CONFLICT_POLICY_UNSPECIFIED: _ClassVar[ReconcileConflictPolicy]
    RECONCILE_CONFLICT_POLICY_REPORT_ONLY: _ClassVar[ReconcileConflictPolicy]
    RECONCILE_CONFLICT_POLICY_SKIP_EXISTING: _ClassVar[ReconcileConflictPolicy]

class ReconcileAction(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RECONCILE_ACTION_UNSPECIFIED: _ClassVar[ReconcileAction]
    RECONCILE_ACTION_ALREADY_CANONICAL: _ClassVar[ReconcileAction]
    RECONCILE_ACTION_MIRROR_FRESH: _ClassVar[ReconcileAction]
    RECONCILE_ACTION_MIRROR_REPAIR_NEEDED: _ClassVar[ReconcileAction]
    RECONCILE_ACTION_MIRROR_REPAIRED: _ClassVar[ReconcileAction]
    RECONCILE_ACTION_IMPORT_PLANNED: _ClassVar[ReconcileAction]
    RECONCILE_ACTION_IMPORTED: _ClassVar[ReconcileAction]
    RECONCILE_ACTION_SKIPPED_DUPLICATE: _ClassVar[ReconcileAction]
    RECONCILE_ACTION_PARSE_FAILED: _ClassVar[ReconcileAction]
    RECONCILE_ACTION_CONFLICT: _ClassVar[ReconcileAction]
CANDIDATE_REVISION_STATE_UNSPECIFIED: CandidateRevisionState
CANDIDATE_REVISION_STATE_PENDING: CandidateRevisionState
CANDIDATE_REVISION_STATE_APPLIED: CandidateRevisionState
CANDIDATE_REVISION_STATE_DISCARDED: CandidateRevisionState
CANDIDATE_REVISION_STATE_EXPIRED: CandidateRevisionState
RECONCILE_CONFLICT_POLICY_UNSPECIFIED: ReconcileConflictPolicy
RECONCILE_CONFLICT_POLICY_REPORT_ONLY: ReconcileConflictPolicy
RECONCILE_CONFLICT_POLICY_SKIP_EXISTING: ReconcileConflictPolicy
RECONCILE_ACTION_UNSPECIFIED: ReconcileAction
RECONCILE_ACTION_ALREADY_CANONICAL: ReconcileAction
RECONCILE_ACTION_MIRROR_FRESH: ReconcileAction
RECONCILE_ACTION_MIRROR_REPAIR_NEEDED: ReconcileAction
RECONCILE_ACTION_MIRROR_REPAIRED: ReconcileAction
RECONCILE_ACTION_IMPORT_PLANNED: ReconcileAction
RECONCILE_ACTION_IMPORTED: ReconcileAction
RECONCILE_ACTION_SKIPPED_DUPLICATE: ReconcileAction
RECONCILE_ACTION_PARSE_FAILED: ReconcileAction
RECONCILE_ACTION_CONFLICT: ReconcileAction

class ListPlansRequest(_message.Message):
    __slots__ = ("status", "include_archived", "workspace")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_ARCHIVED_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    status: _model_pb2.PlanStatus
    include_archived: bool
    workspace: WorkspaceScope
    def __init__(self, status: _Optional[_Union[_model_pb2.PlanStatus, str]] = ..., include_archived: _Optional[bool] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ...) -> None: ...

class ListPlansResponse(_message.Message):
    __slots__ = ("plans",)
    PLANS_FIELD_NUMBER: _ClassVar[int]
    plans: _containers.RepeatedCompositeFieldContainer[_model_pb2.Plan]
    def __init__(self, plans: _Optional[_Iterable[_Union[_model_pb2.Plan, _Mapping]]] = ...) -> None: ...

class GetPlanRequest(_message.Message):
    __slots__ = ("id", "workspace")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    id: str
    workspace: WorkspaceScope
    def __init__(self, id: _Optional[str] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ...) -> None: ...

class GetPlanResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class CreatePlanRequest(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class CreatePlanResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class UpdatePlanRequest(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class UpdatePlanResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class CandidateRevision(_message.Message):
    __slots__ = ("id", "plan_id", "expected_base_content_hash", "proposal_provenance", "candidate_plan", "workspace", "state", "created_at", "updated_at", "expires_at", "applied_at", "applied_content_hash", "discard_reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_BASE_CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    PROPOSAL_PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_PLAN_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    APPLIED_AT_FIELD_NUMBER: _ClassVar[int]
    APPLIED_CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    DISCARD_REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    plan_id: str
    expected_base_content_hash: str
    proposal_provenance: str
    candidate_plan: _model_pb2.Plan
    workspace: WorkspaceScope
    state: CandidateRevisionState
    created_at: str
    updated_at: str
    expires_at: str
    applied_at: str
    applied_content_hash: str
    discard_reason: str
    def __init__(self, id: _Optional[str] = ..., plan_id: _Optional[str] = ..., expected_base_content_hash: _Optional[str] = ..., proposal_provenance: _Optional[str] = ..., candidate_plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ..., state: _Optional[_Union[CandidateRevisionState, str]] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., expires_at: _Optional[str] = ..., applied_at: _Optional[str] = ..., applied_content_hash: _Optional[str] = ..., discard_reason: _Optional[str] = ...) -> None: ...

class CandidateFieldChange(_message.Message):
    __slots__ = ("field", "before_json", "after_json")
    FIELD_FIELD_NUMBER: _ClassVar[int]
    BEFORE_JSON_FIELD_NUMBER: _ClassVar[int]
    AFTER_JSON_FIELD_NUMBER: _ClassVar[int]
    field: str
    before_json: str
    after_json: str
    def __init__(self, field: _Optional[str] = ..., before_json: _Optional[str] = ..., after_json: _Optional[str] = ...) -> None: ...

class CandidateRevisionDiff(_message.Message):
    __slots__ = ("changes",)
    CHANGES_FIELD_NUMBER: _ClassVar[int]
    changes: _containers.RepeatedCompositeFieldContainer[CandidateFieldChange]
    def __init__(self, changes: _Optional[_Iterable[_Union[CandidateFieldChange, _Mapping]]] = ...) -> None: ...

class CandidateValidationDiagnostic(_message.Message):
    __slots__ = ("severity", "code", "location", "message", "guidance")
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    GUIDANCE_FIELD_NUMBER: _ClassVar[int]
    severity: str
    code: str
    location: str
    message: str
    guidance: str
    def __init__(self, severity: _Optional[str] = ..., code: _Optional[str] = ..., location: _Optional[str] = ..., message: _Optional[str] = ..., guidance: _Optional[str] = ...) -> None: ...

class CandidateRevisionPreview(_message.Message):
    __slots__ = ("candidate", "base_plan", "diff", "impact", "rendered_markdown", "quality_status", "diagnostics")
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    BASE_PLAN_FIELD_NUMBER: _ClassVar[int]
    DIFF_FIELD_NUMBER: _ClassVar[int]
    IMPACT_FIELD_NUMBER: _ClassVar[int]
    RENDERED_MARKDOWN_FIELD_NUMBER: _ClassVar[int]
    QUALITY_STATUS_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTICS_FIELD_NUMBER: _ClassVar[int]
    candidate: CandidateRevision
    base_plan: _model_pb2.Plan
    diff: CandidateRevisionDiff
    impact: PlanMutationImpact
    rendered_markdown: str
    quality_status: str
    diagnostics: _containers.RepeatedCompositeFieldContainer[CandidateValidationDiagnostic]
    def __init__(self, candidate: _Optional[_Union[CandidateRevision, _Mapping]] = ..., base_plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ..., diff: _Optional[_Union[CandidateRevisionDiff, _Mapping]] = ..., impact: _Optional[_Union[PlanMutationImpact, _Mapping]] = ..., rendered_markdown: _Optional[str] = ..., quality_status: _Optional[str] = ..., diagnostics: _Optional[_Iterable[_Union[CandidateValidationDiagnostic, _Mapping]]] = ...) -> None: ...

class CreateCandidateRevisionRequest(_message.Message):
    __slots__ = ("plan_id", "expected_base_content_hash", "proposal_provenance", "candidate_plan", "workspace", "expires_at")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_BASE_CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    PROPOSAL_PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_PLAN_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    expected_base_content_hash: str
    proposal_provenance: str
    candidate_plan: _model_pb2.Plan
    workspace: WorkspaceScope
    expires_at: str
    def __init__(self, plan_id: _Optional[str] = ..., expected_base_content_hash: _Optional[str] = ..., proposal_provenance: _Optional[str] = ..., candidate_plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ..., expires_at: _Optional[str] = ...) -> None: ...

class CreateCandidateRevisionResponse(_message.Message):
    __slots__ = ("candidate",)
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    candidate: CandidateRevision
    def __init__(self, candidate: _Optional[_Union[CandidateRevision, _Mapping]] = ...) -> None: ...

class GetCandidateRevisionRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetCandidateRevisionResponse(_message.Message):
    __slots__ = ("candidate",)
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    candidate: CandidateRevision
    def __init__(self, candidate: _Optional[_Union[CandidateRevision, _Mapping]] = ...) -> None: ...

class PreviewCandidateRevisionRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class PreviewCandidateRevisionResponse(_message.Message):
    __slots__ = ("preview",)
    PREVIEW_FIELD_NUMBER: _ClassVar[int]
    preview: CandidateRevisionPreview
    def __init__(self, preview: _Optional[_Union[CandidateRevisionPreview, _Mapping]] = ...) -> None: ...

class ValidateCandidateRevisionRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ValidateCandidateRevisionResponse(_message.Message):
    __slots__ = ("preview",)
    PREVIEW_FIELD_NUMBER: _ClassVar[int]
    preview: CandidateRevisionPreview
    def __init__(self, preview: _Optional[_Union[CandidateRevisionPreview, _Mapping]] = ...) -> None: ...

class ApplyCandidateRevisionRequest(_message.Message):
    __slots__ = ("id", "expected_base_content_hash", "acknowledge_quality_impact")
    ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_BASE_CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    ACKNOWLEDGE_QUALITY_IMPACT_FIELD_NUMBER: _ClassVar[int]
    id: str
    expected_base_content_hash: str
    acknowledge_quality_impact: bool
    def __init__(self, id: _Optional[str] = ..., expected_base_content_hash: _Optional[str] = ..., acknowledge_quality_impact: _Optional[bool] = ...) -> None: ...

class ApplyCandidateRevisionResponse(_message.Message):
    __slots__ = ("candidate", "plan", "preview")
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    PLAN_FIELD_NUMBER: _ClassVar[int]
    PREVIEW_FIELD_NUMBER: _ClassVar[int]
    candidate: CandidateRevision
    plan: _model_pb2.Plan
    preview: CandidateRevisionPreview
    def __init__(self, candidate: _Optional[_Union[CandidateRevision, _Mapping]] = ..., plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ..., preview: _Optional[_Union[CandidateRevisionPreview, _Mapping]] = ...) -> None: ...

class DiscardCandidateRevisionRequest(_message.Message):
    __slots__ = ("id", "reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    reason: str
    def __init__(self, id: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class DiscardCandidateRevisionResponse(_message.Message):
    __slots__ = ("candidate",)
    CANDIDATE_FIELD_NUMBER: _ClassVar[int]
    candidate: CandidateRevision
    def __init__(self, candidate: _Optional[_Union[CandidateRevision, _Mapping]] = ...) -> None: ...

class ArchivePlanRequest(_message.Message):
    __slots__ = ("id", "workspace")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    id: str
    workspace: WorkspaceScope
    def __init__(self, id: _Optional[str] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ...) -> None: ...

class ArchivePlanResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class RenderMarkdownRequest(_message.Message):
    __slots__ = ("id", "workspace", "compact")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    COMPACT_FIELD_NUMBER: _ClassVar[int]
    id: str
    workspace: WorkspaceScope
    compact: bool
    def __init__(self, id: _Optional[str] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ..., compact: _Optional[bool] = ...) -> None: ...

class RenderMarkdownResponse(_message.Message):
    __slots__ = ("markdown", "mirror", "repaired", "plan", "quality_status", "quality_findings")
    MARKDOWN_FIELD_NUMBER: _ClassVar[int]
    MIRROR_FIELD_NUMBER: _ClassVar[int]
    REPAIRED_FIELD_NUMBER: _ClassVar[int]
    PLAN_FIELD_NUMBER: _ClassVar[int]
    QUALITY_STATUS_FIELD_NUMBER: _ClassVar[int]
    QUALITY_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    markdown: str
    mirror: _model_pb2.RenderedPlanMirror
    repaired: bool
    plan: _model_pb2.Plan
    quality_status: str
    quality_findings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, markdown: _Optional[str] = ..., mirror: _Optional[_Union[_model_pb2.RenderedPlanMirror, _Mapping]] = ..., repaired: _Optional[bool] = ..., plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ..., quality_status: _Optional[str] = ..., quality_findings: _Optional[_Iterable[str]] = ...) -> None: ...

class AddPhaseRequest(_message.Message):
    __slots__ = ("plan_id", "phase", "workspace", "allow_quality_regression")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    ALLOW_QUALITY_REGRESSION_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    phase: _model_pb2.Phase
    workspace: WorkspaceScope
    allow_quality_regression: bool
    def __init__(self, plan_id: _Optional[str] = ..., phase: _Optional[_Union[_model_pb2.Phase, _Mapping]] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ..., allow_quality_regression: _Optional[bool] = ...) -> None: ...

class AddPhaseResponse(_message.Message):
    __slots__ = ("plan", "impact")
    PLAN_FIELD_NUMBER: _ClassVar[int]
    IMPACT_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    impact: PlanMutationImpact
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ..., impact: _Optional[_Union[PlanMutationImpact, _Mapping]] = ...) -> None: ...

class UpdatePhaseRequest(_message.Message):
    __slots__ = ("plan_id", "phase", "workspace", "update_mask", "allow_quality_regression")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    UPDATE_MASK_FIELD_NUMBER: _ClassVar[int]
    ALLOW_QUALITY_REGRESSION_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    phase: _model_pb2.Phase
    workspace: WorkspaceScope
    update_mask: _field_mask_pb2.FieldMask
    allow_quality_regression: bool
    def __init__(self, plan_id: _Optional[str] = ..., phase: _Optional[_Union[_model_pb2.Phase, _Mapping]] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ..., update_mask: _Optional[_Union[_field_mask_pb2.FieldMask, _Mapping]] = ..., allow_quality_regression: _Optional[bool] = ...) -> None: ...

class UpdatePhaseResponse(_message.Message):
    __slots__ = ("plan", "impact")
    PLAN_FIELD_NUMBER: _ClassVar[int]
    IMPACT_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    impact: PlanMutationImpact
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ..., impact: _Optional[_Union[PlanMutationImpact, _Mapping]] = ...) -> None: ...

class PlanMutationImpact(_message.Message):
    __slots__ = ("before_grade", "after_grade", "added_issue_codes", "cleared_issue_codes", "execution_grade_regression", "regression_acknowledged")
    BEFORE_GRADE_FIELD_NUMBER: _ClassVar[int]
    AFTER_GRADE_FIELD_NUMBER: _ClassVar[int]
    ADDED_ISSUE_CODES_FIELD_NUMBER: _ClassVar[int]
    CLEARED_ISSUE_CODES_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_GRADE_REGRESSION_FIELD_NUMBER: _ClassVar[int]
    REGRESSION_ACKNOWLEDGED_FIELD_NUMBER: _ClassVar[int]
    before_grade: str
    after_grade: str
    added_issue_codes: _containers.RepeatedScalarFieldContainer[str]
    cleared_issue_codes: _containers.RepeatedScalarFieldContainer[str]
    execution_grade_regression: bool
    regression_acknowledged: bool
    def __init__(self, before_grade: _Optional[str] = ..., after_grade: _Optional[str] = ..., added_issue_codes: _Optional[_Iterable[str]] = ..., cleared_issue_codes: _Optional[_Iterable[str]] = ..., execution_grade_regression: _Optional[bool] = ..., regression_acknowledged: _Optional[bool] = ...) -> None: ...

class ListRelevantContextRequest(_message.Message):
    __slots__ = ("id", "workspace", "phase_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    workspace: WorkspaceScope
    phase_id: str
    def __init__(self, id: _Optional[str] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ..., phase_id: _Optional[str] = ...) -> None: ...

class ListRelevantContextResponse(_message.Message):
    __slots__ = ("items",)
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[_model_pb2.RelevantContextItem]
    def __init__(self, items: _Optional[_Iterable[_Union[_model_pb2.RelevantContextItem, _Mapping]]] = ...) -> None: ...

class AddRelevantContextRequest(_message.Message):
    __slots__ = ("id", "workspace", "phase_id", "item", "allow_quality_regression")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_FIELD_NUMBER: _ClassVar[int]
    ALLOW_QUALITY_REGRESSION_FIELD_NUMBER: _ClassVar[int]
    id: str
    workspace: WorkspaceScope
    phase_id: str
    item: _model_pb2.RelevantContextItem
    allow_quality_regression: bool
    def __init__(self, id: _Optional[str] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ..., phase_id: _Optional[str] = ..., item: _Optional[_Union[_model_pb2.RelevantContextItem, _Mapping]] = ..., allow_quality_regression: _Optional[bool] = ...) -> None: ...

class AddRelevantContextResponse(_message.Message):
    __slots__ = ("plan", "impact")
    PLAN_FIELD_NUMBER: _ClassVar[int]
    IMPACT_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    impact: PlanMutationImpact
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ..., impact: _Optional[_Union[PlanMutationImpact, _Mapping]] = ...) -> None: ...

class UpdateRelevantContextRequest(_message.Message):
    __slots__ = ("id", "workspace", "phase_id", "item_id", "item", "allow_quality_regression")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_FIELD_NUMBER: _ClassVar[int]
    ALLOW_QUALITY_REGRESSION_FIELD_NUMBER: _ClassVar[int]
    id: str
    workspace: WorkspaceScope
    phase_id: str
    item_id: str
    item: _model_pb2.RelevantContextItem
    allow_quality_regression: bool
    def __init__(self, id: _Optional[str] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ..., phase_id: _Optional[str] = ..., item_id: _Optional[str] = ..., item: _Optional[_Union[_model_pb2.RelevantContextItem, _Mapping]] = ..., allow_quality_regression: _Optional[bool] = ...) -> None: ...

class UpdateRelevantContextResponse(_message.Message):
    __slots__ = ("plan", "impact")
    PLAN_FIELD_NUMBER: _ClassVar[int]
    IMPACT_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    impact: PlanMutationImpact
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ..., impact: _Optional[_Union[PlanMutationImpact, _Mapping]] = ...) -> None: ...

class RemoveRelevantContextRequest(_message.Message):
    __slots__ = ("id", "workspace", "phase_id", "item_id", "allow_quality_regression")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    ALLOW_QUALITY_REGRESSION_FIELD_NUMBER: _ClassVar[int]
    id: str
    workspace: WorkspaceScope
    phase_id: str
    item_id: str
    allow_quality_regression: bool
    def __init__(self, id: _Optional[str] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ..., phase_id: _Optional[str] = ..., item_id: _Optional[str] = ..., allow_quality_regression: _Optional[bool] = ...) -> None: ...

class RemoveRelevantContextResponse(_message.Message):
    __slots__ = ("plan", "impact")
    PLAN_FIELD_NUMBER: _ClassVar[int]
    IMPACT_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    impact: PlanMutationImpact
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ..., impact: _Optional[_Union[PlanMutationImpact, _Mapping]] = ...) -> None: ...

class ListReferencesRequest(_message.Message):
    __slots__ = ("id", "workspace", "phase_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    workspace: WorkspaceScope
    phase_id: str
    def __init__(self, id: _Optional[str] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ..., phase_id: _Optional[str] = ...) -> None: ...

class ListReferencesResponse(_message.Message):
    __slots__ = ("references",)
    REFERENCES_FIELD_NUMBER: _ClassVar[int]
    references: _containers.RepeatedCompositeFieldContainer[_model_pb2.Reference]
    def __init__(self, references: _Optional[_Iterable[_Union[_model_pb2.Reference, _Mapping]]] = ...) -> None: ...

class AddReferenceRequest(_message.Message):
    __slots__ = ("id", "workspace", "phase_id", "reference", "allow_quality_regression")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    ALLOW_QUALITY_REGRESSION_FIELD_NUMBER: _ClassVar[int]
    id: str
    workspace: WorkspaceScope
    phase_id: str
    reference: _model_pb2.Reference
    allow_quality_regression: bool
    def __init__(self, id: _Optional[str] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ..., phase_id: _Optional[str] = ..., reference: _Optional[_Union[_model_pb2.Reference, _Mapping]] = ..., allow_quality_regression: _Optional[bool] = ...) -> None: ...

class AddReferenceResponse(_message.Message):
    __slots__ = ("plan", "impact")
    PLAN_FIELD_NUMBER: _ClassVar[int]
    IMPACT_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    impact: PlanMutationImpact
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ..., impact: _Optional[_Union[PlanMutationImpact, _Mapping]] = ...) -> None: ...

class UpdateReferenceRequest(_message.Message):
    __slots__ = ("id", "workspace", "phase_id", "reference_id", "reference", "allow_quality_regression")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_ID_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    ALLOW_QUALITY_REGRESSION_FIELD_NUMBER: _ClassVar[int]
    id: str
    workspace: WorkspaceScope
    phase_id: str
    reference_id: str
    reference: _model_pb2.Reference
    allow_quality_regression: bool
    def __init__(self, id: _Optional[str] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ..., phase_id: _Optional[str] = ..., reference_id: _Optional[str] = ..., reference: _Optional[_Union[_model_pb2.Reference, _Mapping]] = ..., allow_quality_regression: _Optional[bool] = ...) -> None: ...

class UpdateReferenceResponse(_message.Message):
    __slots__ = ("plan", "impact")
    PLAN_FIELD_NUMBER: _ClassVar[int]
    IMPACT_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    impact: PlanMutationImpact
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ..., impact: _Optional[_Union[PlanMutationImpact, _Mapping]] = ...) -> None: ...

class RemoveReferenceRequest(_message.Message):
    __slots__ = ("id", "workspace", "phase_id", "reference_id", "allow_quality_regression")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_ID_FIELD_NUMBER: _ClassVar[int]
    ALLOW_QUALITY_REGRESSION_FIELD_NUMBER: _ClassVar[int]
    id: str
    workspace: WorkspaceScope
    phase_id: str
    reference_id: str
    allow_quality_regression: bool
    def __init__(self, id: _Optional[str] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ..., phase_id: _Optional[str] = ..., reference_id: _Optional[str] = ..., allow_quality_regression: _Optional[bool] = ...) -> None: ...

class RemoveReferenceResponse(_message.Message):
    __slots__ = ("plan", "impact")
    PLAN_FIELD_NUMBER: _ClassVar[int]
    IMPACT_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    impact: PlanMutationImpact
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ..., impact: _Optional[_Union[PlanMutationImpact, _Mapping]] = ...) -> None: ...

class GetGraphRequest(_message.Message):
    __slots__ = ("plan_id",)
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    def __init__(self, plan_id: _Optional[str] = ...) -> None: ...

class GetGraphResponse(_message.Message):
    __slots__ = ("edges",)
    EDGES_FIELD_NUMBER: _ClassVar[int]
    edges: _containers.RepeatedCompositeFieldContainer[_model_pb2.PlanEdge]
    def __init__(self, edges: _Optional[_Iterable[_Union[_model_pb2.PlanEdge, _Mapping]]] = ...) -> None: ...

class LinkSupersessionRequest(_message.Message):
    __slots__ = ("superseding_plan_id", "superseded_plan_id")
    SUPERSEDING_PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDED_PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    superseding_plan_id: str
    superseded_plan_id: str
    def __init__(self, superseding_plan_id: _Optional[str] = ..., superseded_plan_id: _Optional[str] = ...) -> None: ...

class LinkSupersessionResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class LinkDependencyRequest(_message.Message):
    __slots__ = ("depending_plan_id", "dependency_plan_id")
    DEPENDING_PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCY_PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    depending_plan_id: str
    dependency_plan_id: str
    def __init__(self, depending_plan_id: _Optional[str] = ..., dependency_plan_id: _Optional[str] = ...) -> None: ...

class LinkDependencyResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class ImportPlanRequest(_message.Message):
    __slots__ = ("source_path", "markdown", "title", "slug", "workspace", "supersede")
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    MARKDOWN_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDE_FIELD_NUMBER: _ClassVar[int]
    source_path: str
    markdown: str
    title: str
    slug: str
    workspace: WorkspaceScope
    supersede: str
    def __init__(self, source_path: _Optional[str] = ..., markdown: _Optional[str] = ..., title: _Optional[str] = ..., slug: _Optional[str] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ..., supersede: _Optional[str] = ...) -> None: ...

class ImportPlanResponse(_message.Message):
    __slots__ = ("plan", "superseded_plan")
    PLAN_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDED_PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    superseded_plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ..., superseded_plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class MigratePlanRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class MigratePlanResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class ReconcilePlansRequest(_message.Message):
    __slots__ = ("dry_run", "repair_mirrors", "source_intake", "include_archived", "include_archived_sources", "conflict_policy", "source_runtime_home_plans", "source_docs_plans", "source_repo_plans", "workspace", "retire_sources")
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    REPAIR_MIRRORS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_INTAKE_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_ARCHIVED_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_ARCHIVED_SOURCES_FIELD_NUMBER: _ClassVar[int]
    CONFLICT_POLICY_FIELD_NUMBER: _ClassVar[int]
    SOURCE_RUNTIME_HOME_PLANS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_DOCS_PLANS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REPO_PLANS_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    RETIRE_SOURCES_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    repair_mirrors: bool
    source_intake: bool
    include_archived: bool
    include_archived_sources: bool
    conflict_policy: ReconcileConflictPolicy
    source_runtime_home_plans: bool
    source_docs_plans: bool
    source_repo_plans: bool
    workspace: WorkspaceScope
    retire_sources: bool
    def __init__(self, dry_run: _Optional[bool] = ..., repair_mirrors: _Optional[bool] = ..., source_intake: _Optional[bool] = ..., include_archived: _Optional[bool] = ..., include_archived_sources: _Optional[bool] = ..., conflict_policy: _Optional[_Union[ReconcileConflictPolicy, str]] = ..., source_runtime_home_plans: _Optional[bool] = ..., source_docs_plans: _Optional[bool] = ..., source_repo_plans: _Optional[bool] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ..., retire_sources: _Optional[bool] = ...) -> None: ...

class WorkspaceScope(_message.Message):
    __slots__ = ("id", "root")
    ID_FIELD_NUMBER: _ClassVar[int]
    ROOT_FIELD_NUMBER: _ClassVar[int]
    id: str
    root: str
    def __init__(self, id: _Optional[str] = ..., root: _Optional[str] = ...) -> None: ...

class ReconcilePlanItem(_message.Message):
    __slots__ = ("action", "plan_id", "slug", "title", "source_path", "mirror", "source_untouched", "error", "source_retirement_planned", "source_removed")
    ACTION_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    MIRROR_FIELD_NUMBER: _ClassVar[int]
    SOURCE_UNTOUCHED_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    SOURCE_RETIREMENT_PLANNED_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REMOVED_FIELD_NUMBER: _ClassVar[int]
    action: ReconcileAction
    plan_id: str
    slug: str
    title: str
    source_path: str
    mirror: _model_pb2.RenderedPlanMirror
    source_untouched: bool
    error: str
    source_retirement_planned: bool
    source_removed: bool
    def __init__(self, action: _Optional[_Union[ReconcileAction, str]] = ..., plan_id: _Optional[str] = ..., slug: _Optional[str] = ..., title: _Optional[str] = ..., source_path: _Optional[str] = ..., mirror: _Optional[_Union[_model_pb2.RenderedPlanMirror, _Mapping]] = ..., source_untouched: _Optional[bool] = ..., error: _Optional[str] = ..., source_retirement_planned: _Optional[bool] = ..., source_removed: _Optional[bool] = ...) -> None: ...

class ReconcilePlansResponse(_message.Message):
    __slots__ = ("dry_run", "items")
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    items: _containers.RepeatedCompositeFieldContainer[ReconcilePlanItem]
    def __init__(self, dry_run: _Optional[bool] = ..., items: _Optional[_Iterable[_Union[ReconcilePlanItem, _Mapping]]] = ...) -> None: ...

class ListTemplatesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListTemplatesResponse(_message.Message):
    __slots__ = ("templates",)
    TEMPLATES_FIELD_NUMBER: _ClassVar[int]
    templates: _containers.RepeatedCompositeFieldContainer[PlanTemplate]
    def __init__(self, templates: _Optional[_Iterable[_Union[PlanTemplate, _Mapping]]] = ...) -> None: ...

class PlanTemplate(_message.Message):
    __slots__ = ("id", "name", "description", "surface")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    surface: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., surface: _Optional[str] = ...) -> None: ...

class CreateFromTemplateRequest(_message.Message):
    __slots__ = ("template_id", "title", "slug")
    TEMPLATE_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    template_id: str
    title: str
    slug: str
    def __init__(self, template_id: _Optional[str] = ..., title: _Optional[str] = ..., slug: _Optional[str] = ...) -> None: ...

class CreateFromTemplateResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class ListAuditFactsRequest(_message.Message):
    __slots__ = ("run_id",)
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    def __init__(self, run_id: _Optional[str] = ...) -> None: ...

class PlanAuditFact(_message.Message):
    __slots__ = ("event_id", "run_id", "task_id", "action", "plan_id", "content_digest", "occurred_at")
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    CONTENT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    OCCURRED_AT_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    run_id: str
    task_id: str
    action: str
    plan_id: str
    content_digest: str
    occurred_at: str
    def __init__(self, event_id: _Optional[str] = ..., run_id: _Optional[str] = ..., task_id: _Optional[str] = ..., action: _Optional[str] = ..., plan_id: _Optional[str] = ..., content_digest: _Optional[str] = ..., occurred_at: _Optional[str] = ...) -> None: ...

class ListAuditFactsResponse(_message.Message):
    __slots__ = ("facts",)
    FACTS_FIELD_NUMBER: _ClassVar[int]
    facts: _containers.RepeatedCompositeFieldContainer[PlanAuditFact]
    def __init__(self, facts: _Optional[_Iterable[_Union[PlanAuditFact, _Mapping]]] = ...) -> None: ...
