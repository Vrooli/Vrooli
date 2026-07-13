from plan_manager.v1.shared import model_pb2 as _model_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

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
    __slots__ = ("plan_id", "phase", "workspace")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    phase: _model_pb2.Phase
    workspace: WorkspaceScope
    def __init__(self, plan_id: _Optional[str] = ..., phase: _Optional[_Union[_model_pb2.Phase, _Mapping]] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ...) -> None: ...

class AddPhaseResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class UpdatePhaseRequest(_message.Message):
    __slots__ = ("plan_id", "phase", "workspace")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    phase: _model_pb2.Phase
    workspace: WorkspaceScope
    def __init__(self, plan_id: _Optional[str] = ..., phase: _Optional[_Union[_model_pb2.Phase, _Mapping]] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ...) -> None: ...

class UpdatePhaseResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

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

class UpdateRelevantContextRequest(_message.Message):
    __slots__ = ("id", "workspace", "phase_id", "item_id", "item")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_FIELD_NUMBER: _ClassVar[int]
    id: str
    workspace: WorkspaceScope
    phase_id: str
    item_id: str
    item: _model_pb2.RelevantContextItem
    def __init__(self, id: _Optional[str] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ..., phase_id: _Optional[str] = ..., item_id: _Optional[str] = ..., item: _Optional[_Union[_model_pb2.RelevantContextItem, _Mapping]] = ...) -> None: ...

class UpdateRelevantContextResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class RemoveRelevantContextRequest(_message.Message):
    __slots__ = ("id", "workspace", "phase_id", "item_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    workspace: WorkspaceScope
    phase_id: str
    item_id: str
    def __init__(self, id: _Optional[str] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ..., phase_id: _Optional[str] = ..., item_id: _Optional[str] = ...) -> None: ...

class RemoveRelevantContextResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

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

class UpdateReferenceRequest(_message.Message):
    __slots__ = ("id", "workspace", "phase_id", "reference_id", "reference")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_ID_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    workspace: WorkspaceScope
    phase_id: str
    reference_id: str
    reference: _model_pb2.Reference
    def __init__(self, id: _Optional[str] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ..., phase_id: _Optional[str] = ..., reference_id: _Optional[str] = ..., reference: _Optional[_Union[_model_pb2.Reference, _Mapping]] = ...) -> None: ...

class UpdateReferenceResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

class RemoveReferenceRequest(_message.Message):
    __slots__ = ("id", "workspace", "phase_id", "reference_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    workspace: WorkspaceScope
    phase_id: str
    reference_id: str
    def __init__(self, id: _Optional[str] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ..., phase_id: _Optional[str] = ..., reference_id: _Optional[str] = ...) -> None: ...

class RemoveReferenceResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

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
    __slots__ = ("source_path", "markdown", "title", "slug", "workspace")
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    MARKDOWN_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    source_path: str
    markdown: str
    title: str
    slug: str
    workspace: WorkspaceScope
    def __init__(self, source_path: _Optional[str] = ..., markdown: _Optional[str] = ..., title: _Optional[str] = ..., slug: _Optional[str] = ..., workspace: _Optional[_Union[WorkspaceScope, _Mapping]] = ...) -> None: ...

class ImportPlanResponse(_message.Message):
    __slots__ = ("plan",)
    PLAN_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ...) -> None: ...

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
