from buf.validate import validate_pb2 as _validate_pb2
from swarm_manager.v1.domain import backlog_pb2 as _backlog_pb2
from swarm_manager.v1.domain import execution_pb2 as _execution_pb2
from swarm_manager.v1.domain import plan_ref_pb2 as _plan_ref_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CreateBacklogItemRequest(_message.Message):
    __slots__ = ("name", "title", "description", "priority", "tags", "kind", "depends_on", "milestone", "effort", "acceptance_allow", "acceptance_deny", "spawned_from", "note", "creates", "plan_ref")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    DEPENDS_ON_FIELD_NUMBER: _ClassVar[int]
    MILESTONE_FIELD_NUMBER: _ClassVar[int]
    EFFORT_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_ALLOW_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_DENY_FIELD_NUMBER: _ClassVar[int]
    SPAWNED_FROM_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    CREATES_FIELD_NUMBER: _ClassVar[int]
    PLAN_REF_FIELD_NUMBER: _ClassVar[int]
    name: str
    title: str
    description: str
    priority: int
    tags: _containers.RepeatedScalarFieldContainer[str]
    kind: str
    depends_on: _containers.RepeatedScalarFieldContainer[str]
    milestone: str
    effort: str
    acceptance_allow: _containers.RepeatedScalarFieldContainer[str]
    acceptance_deny: _containers.RepeatedScalarFieldContainer[str]
    spawned_from: str
    note: str
    creates: _containers.RepeatedScalarFieldContainer[str]
    plan_ref: _plan_ref_pb2.PlanRef
    def __init__(self, name: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., priority: _Optional[int] = ..., tags: _Optional[_Iterable[str]] = ..., kind: _Optional[str] = ..., depends_on: _Optional[_Iterable[str]] = ..., milestone: _Optional[str] = ..., effort: _Optional[str] = ..., acceptance_allow: _Optional[_Iterable[str]] = ..., acceptance_deny: _Optional[_Iterable[str]] = ..., spawned_from: _Optional[str] = ..., note: _Optional[str] = ..., creates: _Optional[_Iterable[str]] = ..., plan_ref: _Optional[_Union[_plan_ref_pb2.PlanRef, _Mapping]] = ...) -> None: ...

class UpdateBacklogItemRequest(_message.Message):
    __slots__ = ("title", "description", "status", "priority", "tags", "depends_on", "milestone", "effort", "acceptance_allow", "acceptance_deny", "spawned_from", "note", "creates", "plan_ref")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    DEPENDS_ON_FIELD_NUMBER: _ClassVar[int]
    MILESTONE_FIELD_NUMBER: _ClassVar[int]
    EFFORT_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_ALLOW_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_DENY_FIELD_NUMBER: _ClassVar[int]
    SPAWNED_FROM_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    CREATES_FIELD_NUMBER: _ClassVar[int]
    PLAN_REF_FIELD_NUMBER: _ClassVar[int]
    title: str
    description: str
    status: str
    priority: int
    tags: _containers.RepeatedScalarFieldContainer[str]
    depends_on: _containers.RepeatedScalarFieldContainer[str]
    milestone: str
    effort: str
    acceptance_allow: _containers.RepeatedScalarFieldContainer[str]
    acceptance_deny: _containers.RepeatedScalarFieldContainer[str]
    spawned_from: str
    note: str
    creates: _containers.RepeatedScalarFieldContainer[str]
    plan_ref: _plan_ref_pb2.PlanRef
    def __init__(self, title: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ..., priority: _Optional[int] = ..., tags: _Optional[_Iterable[str]] = ..., depends_on: _Optional[_Iterable[str]] = ..., milestone: _Optional[str] = ..., effort: _Optional[str] = ..., acceptance_allow: _Optional[_Iterable[str]] = ..., acceptance_deny: _Optional[_Iterable[str]] = ..., spawned_from: _Optional[str] = ..., note: _Optional[str] = ..., creates: _Optional[_Iterable[str]] = ..., plan_ref: _Optional[_Union[_plan_ref_pb2.PlanRef, _Mapping]] = ...) -> None: ...

class BlockingReason(_message.Message):
    __slots__ = ("message", "forceable")
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    FORCEABLE_FIELD_NUMBER: _ClassVar[int]
    message: str
    forceable: bool
    def __init__(self, message: _Optional[str] = ..., forceable: _Optional[bool] = ...) -> None: ...

class ItemBlockingInfo(_message.Message):
    __slots__ = ("blocked", "blocking_dep_keys", "all_forceable")
    BLOCKED_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_DEP_KEYS_FIELD_NUMBER: _ClassVar[int]
    ALL_FORCEABLE_FIELD_NUMBER: _ClassVar[int]
    blocked: bool
    blocking_dep_keys: _containers.RepeatedScalarFieldContainer[str]
    all_forceable: bool
    def __init__(self, blocked: _Optional[bool] = ..., blocking_dep_keys: _Optional[_Iterable[str]] = ..., all_forceable: _Optional[bool] = ...) -> None: ...

class ListBacklogItemsResponse(_message.Message):
    __slots__ = ("items", "blocking")
    class BlockingEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: ItemBlockingInfo
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[ItemBlockingInfo, _Mapping]] = ...) -> None: ...
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[_backlog_pb2.BacklogItem]
    blocking: _containers.MessageMap[str, ItemBlockingInfo]
    def __init__(self, items: _Optional[_Iterable[_Union[_backlog_pb2.BacklogItem, _Mapping]]] = ..., blocking: _Optional[_Mapping[str, ItemBlockingInfo]] = ...) -> None: ...

class BacklogItemResponse(_message.Message):
    __slots__ = ("item", "deduped")
    ITEM_FIELD_NUMBER: _ClassVar[int]
    DEDUPED_FIELD_NUMBER: _ClassVar[int]
    item: _backlog_pb2.BacklogItem
    deduped: bool
    def __init__(self, item: _Optional[_Union[_backlog_pb2.BacklogItem, _Mapping]] = ..., deduped: _Optional[bool] = ...) -> None: ...

class GetBacklogItemRequest(_message.Message):
    __slots__ = ("kind", "name")
    KIND_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    kind: str
    name: str
    def __init__(self, kind: _Optional[str] = ..., name: _Optional[str] = ...) -> None: ...

class AutoFilerStatusRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class AutoFilerRunNowRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class AutoFilerBrakeState(_message.Message):
    __slots__ = ("window_days", "minimum", "observed", "braked", "window_start_utc", "window_end_utc")
    WINDOW_DAYS_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_FIELD_NUMBER: _ClassVar[int]
    OBSERVED_FIELD_NUMBER: _ClassVar[int]
    BRAKED_FIELD_NUMBER: _ClassVar[int]
    WINDOW_START_UTC_FIELD_NUMBER: _ClassVar[int]
    WINDOW_END_UTC_FIELD_NUMBER: _ClassVar[int]
    window_days: int
    minimum: int
    observed: int
    braked: bool
    window_start_utc: str
    window_end_utc: str
    def __init__(self, window_days: _Optional[int] = ..., minimum: _Optional[int] = ..., observed: _Optional[int] = ..., braked: _Optional[bool] = ..., window_start_utc: _Optional[str] = ..., window_end_utc: _Optional[str] = ...) -> None: ...

class AutoFilerStatusResponse(_message.Message):
    __slots__ = ("enabled", "mode", "strategy", "last_cycle_time", "last_error", "candidates", "findings", "created", "skipped_dismissed", "open_auto_filed", "max_open_auto_filed", "remaining_budget", "brake", "dismissal_count", "reconciled_closed", "reconciled_noted")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    LAST_CYCLE_TIME_FIELD_NUMBER: _ClassVar[int]
    LAST_ERROR_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_DISMISSED_FIELD_NUMBER: _ClassVar[int]
    OPEN_AUTO_FILED_FIELD_NUMBER: _ClassVar[int]
    MAX_OPEN_AUTO_FILED_FIELD_NUMBER: _ClassVar[int]
    REMAINING_BUDGET_FIELD_NUMBER: _ClassVar[int]
    BRAKE_FIELD_NUMBER: _ClassVar[int]
    DISMISSAL_COUNT_FIELD_NUMBER: _ClassVar[int]
    RECONCILED_CLOSED_FIELD_NUMBER: _ClassVar[int]
    RECONCILED_NOTED_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    mode: str
    strategy: str
    last_cycle_time: str
    last_error: str
    candidates: int
    findings: int
    created: int
    skipped_dismissed: int
    open_auto_filed: int
    max_open_auto_filed: int
    remaining_budget: int
    brake: AutoFilerBrakeState
    dismissal_count: int
    reconciled_closed: int
    reconciled_noted: int
    def __init__(self, enabled: _Optional[bool] = ..., mode: _Optional[str] = ..., strategy: _Optional[str] = ..., last_cycle_time: _Optional[str] = ..., last_error: _Optional[str] = ..., candidates: _Optional[int] = ..., findings: _Optional[int] = ..., created: _Optional[int] = ..., skipped_dismissed: _Optional[int] = ..., open_auto_filed: _Optional[int] = ..., max_open_auto_filed: _Optional[int] = ..., remaining_budget: _Optional[int] = ..., brake: _Optional[_Union[AutoFilerBrakeState, _Mapping]] = ..., dismissal_count: _Optional[int] = ..., reconciled_closed: _Optional[int] = ..., reconciled_noted: _Optional[int] = ...) -> None: ...

class DismissAutoFilerSuggestionRequest(_message.Message):
    __slots__ = ("kind", "name", "reason")
    KIND_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    kind: str
    name: str
    reason: str
    def __init__(self, kind: _Optional[str] = ..., name: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class DismissAutoFilerSuggestionResponse(_message.Message):
    __slots__ = ("item", "dismissed")
    ITEM_FIELD_NUMBER: _ClassVar[int]
    DISMISSED_FIELD_NUMBER: _ClassVar[int]
    item: _backlog_pb2.BacklogItem
    dismissed: bool
    def __init__(self, item: _Optional[_Union[_backlog_pb2.BacklogItem, _Mapping]] = ..., dismissed: _Optional[bool] = ...) -> None: ...

class BacklogFilesResponse(_message.Message):
    __slots__ = ("files",)
    FILES_FIELD_NUMBER: _ClassVar[int]
    files: _containers.RepeatedCompositeFieldContainer[_backlog_pb2.BacklogFile]
    def __init__(self, files: _Optional[_Iterable[_Union[_backlog_pb2.BacklogFile, _Mapping]]] = ...) -> None: ...

class BacklogFileResponse(_message.Message):
    __slots__ = ("file",)
    FILE_FIELD_NUMBER: _ClassVar[int]
    file: _backlog_pb2.BacklogFile
    def __init__(self, file: _Optional[_Union[_backlog_pb2.BacklogFile, _Mapping]] = ...) -> None: ...

class BacklogFileOperationRequest(_message.Message):
    __slots__ = ("operation", "source_path", "destination_path")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_PATH_FIELD_NUMBER: _ClassVar[int]
    operation: str
    source_path: str
    destination_path: str
    def __init__(self, operation: _Optional[str] = ..., source_path: _Optional[str] = ..., destination_path: _Optional[str] = ...) -> None: ...

class BacklogFileOperationResponse(_message.Message):
    __slots__ = ("file", "deleted_path")
    FILE_FIELD_NUMBER: _ClassVar[int]
    DELETED_PATH_FIELD_NUMBER: _ClassVar[int]
    file: _backlog_pb2.BacklogFile
    deleted_path: str
    def __init__(self, file: _Optional[_Union[_backlog_pb2.BacklogFile, _Mapping]] = ..., deleted_path: _Optional[str] = ...) -> None: ...

class QueueBacklogItemRequest(_message.Message):
    __slots__ = ("operation", "mode", "started_by", "confirm", "force", "strategy", "max_slices")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    STARTED_BY_FIELD_NUMBER: _ClassVar[int]
    CONFIRM_FIELD_NUMBER: _ClassVar[int]
    FORCE_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    MAX_SLICES_FIELD_NUMBER: _ClassVar[int]
    operation: str
    mode: str
    started_by: str
    confirm: bool
    force: bool
    strategy: str
    max_slices: int
    def __init__(self, operation: _Optional[str] = ..., mode: _Optional[str] = ..., started_by: _Optional[str] = ..., confirm: _Optional[bool] = ..., force: _Optional[bool] = ..., strategy: _Optional[str] = ..., max_slices: _Optional[int] = ...) -> None: ...

class QueueBacklogItemResponse(_message.Message):
    __slots__ = ("item", "task_id", "run_id", "base_url", "created", "dry_run", "queued", "message", "blocking_reasons", "unanswered_questions", "pending_suggestions", "advisories")
    ITEM_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    BASE_URL_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    QUEUED_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_REASONS_FIELD_NUMBER: _ClassVar[int]
    UNANSWERED_QUESTIONS_FIELD_NUMBER: _ClassVar[int]
    PENDING_SUGGESTIONS_FIELD_NUMBER: _ClassVar[int]
    ADVISORIES_FIELD_NUMBER: _ClassVar[int]
    item: _backlog_pb2.BacklogItem
    task_id: str
    run_id: str
    base_url: str
    created: str
    dry_run: bool
    queued: bool
    message: str
    blocking_reasons: _containers.RepeatedCompositeFieldContainer[BlockingReason]
    unanswered_questions: int
    pending_suggestions: int
    advisories: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, item: _Optional[_Union[_backlog_pb2.BacklogItem, _Mapping]] = ..., task_id: _Optional[str] = ..., run_id: _Optional[str] = ..., base_url: _Optional[str] = ..., created: _Optional[str] = ..., dry_run: _Optional[bool] = ..., queued: _Optional[bool] = ..., message: _Optional[str] = ..., blocking_reasons: _Optional[_Iterable[_Union[BlockingReason, _Mapping]]] = ..., unanswered_questions: _Optional[int] = ..., pending_suggestions: _Optional[int] = ..., advisories: _Optional[_Iterable[str]] = ...) -> None: ...

class BacklogResearchRequest(_message.Message):
    __slots__ = ("prompt", "project_root", "mode", "context_paths", "context_target_ids", "context_requirement_ids", "confirm", "force")
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    PROJECT_ROOT_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_PATHS_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_TARGET_IDS_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_REQUIREMENT_IDS_FIELD_NUMBER: _ClassVar[int]
    CONFIRM_FIELD_NUMBER: _ClassVar[int]
    FORCE_FIELD_NUMBER: _ClassVar[int]
    prompt: str
    project_root: str
    mode: str
    context_paths: _containers.RepeatedScalarFieldContainer[str]
    context_target_ids: _containers.RepeatedScalarFieldContainer[str]
    context_requirement_ids: _containers.RepeatedScalarFieldContainer[str]
    confirm: bool
    force: bool
    def __init__(self, prompt: _Optional[str] = ..., project_root: _Optional[str] = ..., mode: _Optional[str] = ..., context_paths: _Optional[_Iterable[str]] = ..., context_target_ids: _Optional[_Iterable[str]] = ..., context_requirement_ids: _Optional[_Iterable[str]] = ..., confirm: _Optional[bool] = ..., force: _Optional[bool] = ...) -> None: ...

class BacklogResearchResponse(_message.Message):
    __slots__ = ("task_id", "run_id", "base_url", "created", "dry_run", "started", "message", "blocking_reasons")
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    BASE_URL_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    STARTED_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    BLOCKING_REASONS_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    run_id: str
    base_url: str
    created: str
    dry_run: bool
    started: bool
    message: str
    blocking_reasons: _containers.RepeatedCompositeFieldContainer[BlockingReason]
    def __init__(self, task_id: _Optional[str] = ..., run_id: _Optional[str] = ..., base_url: _Optional[str] = ..., created: _Optional[str] = ..., dry_run: _Optional[bool] = ..., started: _Optional[bool] = ..., message: _Optional[str] = ..., blocking_reasons: _Optional[_Iterable[_Union[BlockingReason, _Mapping]]] = ...) -> None: ...

class ExportBacklogRequest(_message.Message):
    __slots__ = ("kinds", "statuses", "names", "priority_max", "tags", "include_prd", "include_requirements", "include_clarify_questions", "include_suggestions", "include_notes", "include_template")
    KINDS_FIELD_NUMBER: _ClassVar[int]
    STATUSES_FIELD_NUMBER: _ClassVar[int]
    NAMES_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_MAX_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_PRD_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_REQUIREMENTS_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_CLARIFY_QUESTIONS_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_SUGGESTIONS_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_NOTES_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    kinds: _containers.RepeatedScalarFieldContainer[str]
    statuses: _containers.RepeatedScalarFieldContainer[str]
    names: _containers.RepeatedScalarFieldContainer[str]
    priority_max: int
    tags: _containers.RepeatedScalarFieldContainer[str]
    include_prd: bool
    include_requirements: bool
    include_clarify_questions: bool
    include_suggestions: bool
    include_notes: bool
    include_template: bool
    def __init__(self, kinds: _Optional[_Iterable[str]] = ..., statuses: _Optional[_Iterable[str]] = ..., names: _Optional[_Iterable[str]] = ..., priority_max: _Optional[int] = ..., tags: _Optional[_Iterable[str]] = ..., include_prd: _Optional[bool] = ..., include_requirements: _Optional[bool] = ..., include_clarify_questions: _Optional[bool] = ..., include_suggestions: _Optional[bool] = ..., include_notes: _Optional[bool] = ..., include_template: _Optional[bool] = ...) -> None: ...

class ImportBacklogResponse(_message.Message):
    __slots__ = ("dry_run", "changes", "errors", "summary")
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    CHANGES_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    dry_run: bool
    changes: _containers.RepeatedCompositeFieldContainer[ImportChange]
    errors: _containers.RepeatedScalarFieldContainer[str]
    summary: str
    def __init__(self, dry_run: _Optional[bool] = ..., changes: _Optional[_Iterable[_Union[ImportChange, _Mapping]]] = ..., errors: _Optional[_Iterable[str]] = ..., summary: _Optional[str] = ...) -> None: ...

class ImportChange(_message.Message):
    __slots__ = ("item", "action", "details")
    ITEM_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    DETAILS_FIELD_NUMBER: _ClassVar[int]
    item: str
    action: str
    details: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, item: _Optional[str] = ..., action: _Optional[str] = ..., details: _Optional[_Iterable[str]] = ...) -> None: ...

class WorkshopSaveRequest(_message.Message):
    __slots__ = ("round_number", "content")
    ROUND_NUMBER_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    round_number: int
    content: str
    def __init__(self, round_number: _Optional[int] = ..., content: _Optional[str] = ...) -> None: ...

class WorkshopSaveResponse(_message.Message):
    __slots__ = ("file", "auto_advance")
    FILE_FIELD_NUMBER: _ClassVar[int]
    AUTO_ADVANCE_FIELD_NUMBER: _ClassVar[int]
    file: _backlog_pb2.BacklogFile
    auto_advance: WorkshopAutoAdvance
    def __init__(self, file: _Optional[_Union[_backlog_pb2.BacklogFile, _Mapping]] = ..., auto_advance: _Optional[_Union[WorkshopAutoAdvance, _Mapping]] = ...) -> None: ...

class WorkshopAutoAdvance(_message.Message):
    __slots__ = ("triggered", "run_id", "task_id", "reason", "next_mode", "pending", "advance_at", "delay_seconds")
    TRIGGERED_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    NEXT_MODE_FIELD_NUMBER: _ClassVar[int]
    PENDING_FIELD_NUMBER: _ClassVar[int]
    ADVANCE_AT_FIELD_NUMBER: _ClassVar[int]
    DELAY_SECONDS_FIELD_NUMBER: _ClassVar[int]
    triggered: bool
    run_id: str
    task_id: str
    reason: str
    next_mode: str
    pending: bool
    advance_at: str
    delay_seconds: int
    def __init__(self, triggered: _Optional[bool] = ..., run_id: _Optional[str] = ..., task_id: _Optional[str] = ..., reason: _Optional[str] = ..., next_mode: _Optional[str] = ..., pending: _Optional[bool] = ..., advance_at: _Optional[str] = ..., delay_seconds: _Optional[int] = ...) -> None: ...

class WorkshopDeleteRoundRequest(_message.Message):
    __slots__ = ("round_number",)
    ROUND_NUMBER_FIELD_NUMBER: _ClassVar[int]
    round_number: int
    def __init__(self, round_number: _Optional[int] = ...) -> None: ...

class WorkshopDeleteRoundResponse(_message.Message):
    __slots__ = ("deleted_round", "remaining_rounds")
    DELETED_ROUND_FIELD_NUMBER: _ClassVar[int]
    REMAINING_ROUNDS_FIELD_NUMBER: _ClassVar[int]
    deleted_round: int
    remaining_rounds: int
    def __init__(self, deleted_round: _Optional[int] = ..., remaining_rounds: _Optional[int] = ...) -> None: ...

class WorkshopResetRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class WorkshopResetResponse(_message.Message):
    __slots__ = ("deleted_rounds", "status_reverted")
    DELETED_ROUNDS_FIELD_NUMBER: _ClassVar[int]
    STATUS_REVERTED_FIELD_NUMBER: _ClassVar[int]
    deleted_rounds: int
    status_reverted: bool
    def __init__(self, deleted_rounds: _Optional[int] = ..., status_reverted: _Optional[bool] = ...) -> None: ...

class CreateClarificationRequest(_message.Message):
    __slots__ = ("round_number", "item_id", "message", "attachment_ids")
    ROUND_NUMBER_FIELD_NUMBER: _ClassVar[int]
    ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENT_IDS_FIELD_NUMBER: _ClassVar[int]
    round_number: int
    item_id: str
    message: str
    attachment_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, round_number: _Optional[int] = ..., item_id: _Optional[str] = ..., message: _Optional[str] = ..., attachment_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class CreateClarificationResponse(_message.Message):
    __slots__ = ("thread",)
    THREAD_FIELD_NUMBER: _ClassVar[int]
    thread: _backlog_pb2.ClarificationThread
    def __init__(self, thread: _Optional[_Union[_backlog_pb2.ClarificationThread, _Mapping]] = ...) -> None: ...

class ContinueClarificationRequest(_message.Message):
    __slots__ = ("message", "attachment_ids")
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENT_IDS_FIELD_NUMBER: _ClassVar[int]
    message: str
    attachment_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, message: _Optional[str] = ..., attachment_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class ContinueClarificationResponse(_message.Message):
    __slots__ = ("thread",)
    THREAD_FIELD_NUMBER: _ClassVar[int]
    thread: _backlog_pb2.ClarificationThread
    def __init__(self, thread: _Optional[_Union[_backlog_pb2.ClarificationThread, _Mapping]] = ...) -> None: ...

class GetClarificationResponse(_message.Message):
    __slots__ = ("thread",)
    THREAD_FIELD_NUMBER: _ClassVar[int]
    thread: _backlog_pb2.ClarificationThread
    def __init__(self, thread: _Optional[_Union[_backlog_pb2.ClarificationThread, _Mapping]] = ...) -> None: ...

class ClarificationActionRequest(_message.Message):
    __slots__ = ("action", "updated_item_json")
    ACTION_FIELD_NUMBER: _ClassVar[int]
    UPDATED_ITEM_JSON_FIELD_NUMBER: _ClassVar[int]
    action: str
    updated_item_json: str
    def __init__(self, action: _Optional[str] = ..., updated_item_json: _Optional[str] = ...) -> None: ...

class ClarificationActionResponse(_message.Message):
    __slots__ = ("action", "success", "message", "run_id", "task_id")
    ACTION_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    action: str
    success: bool
    message: str
    run_id: str
    task_id: str
    def __init__(self, action: _Optional[str] = ..., success: _Optional[bool] = ..., message: _Optional[str] = ..., run_id: _Optional[str] = ..., task_id: _Optional[str] = ...) -> None: ...

class ListReviewRoundsResponse(_message.Message):
    __slots__ = ("rounds",)
    ROUNDS_FIELD_NUMBER: _ClassVar[int]
    rounds: _containers.RepeatedCompositeFieldContainer[_execution_pb2.ReviewEvidenceRound]
    def __init__(self, rounds: _Optional[_Iterable[_Union[_execution_pb2.ReviewEvidenceRound, _Mapping]]] = ...) -> None: ...

class VerifyEvidenceRequest(_message.Message):
    __slots__ = ("verified",)
    VERIFIED_FIELD_NUMBER: _ClassVar[int]
    verified: bool
    def __init__(self, verified: _Optional[bool] = ...) -> None: ...

class RequestMoreEvidenceRequest(_message.Message):
    __slots__ = ("message", "evidence_id")
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_ID_FIELD_NUMBER: _ClassVar[int]
    message: str
    evidence_id: str
    def __init__(self, message: _Optional[str] = ..., evidence_id: _Optional[str] = ...) -> None: ...

class RequestMoreEvidenceResponse(_message.Message):
    __slots__ = ("thread_id",)
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    thread_id: str
    def __init__(self, thread_id: _Optional[str] = ...) -> None: ...

class ContinueEvidenceRequestRequest(_message.Message):
    __slots__ = ("message",)
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    message: str
    def __init__(self, message: _Optional[str] = ...) -> None: ...
