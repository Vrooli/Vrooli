from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class OperatingModeCatalogRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class OperatingModeGetRequest(_message.Message):
    __slots__ = ("mode",)
    MODE_FIELD_NUMBER: _ClassVar[int]
    mode: str
    def __init__(self, mode: _Optional[str] = ...) -> None: ...

class OperatingModeUpdateRequest(_message.Message):
    __slots__ = ("mode", "label", "description")
    MODE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    mode: str
    label: str
    description: str
    def __init__(self, mode: _Optional[str] = ..., label: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class OperatingModeSimulateRequest(_message.Message):
    __slots__ = ("mode", "preset", "draft")
    MODE_FIELD_NUMBER: _ClassVar[int]
    PRESET_FIELD_NUMBER: _ClassVar[int]
    DRAFT_FIELD_NUMBER: _ClassVar[int]
    mode: str
    preset: str
    draft: bool
    def __init__(self, mode: _Optional[str] = ..., preset: _Optional[str] = ..., draft: _Optional[bool] = ...) -> None: ...

class OperatingModeScaffoldRequest(_message.Message):
    __slots__ = ("id", "label", "description", "force", "start_from")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    FORCE_FIELD_NUMBER: _ClassVar[int]
    START_FROM_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    description: str
    force: bool
    start_from: str
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., description: _Optional[str] = ..., force: _Optional[bool] = ..., start_from: _Optional[str] = ...) -> None: ...

class OperatingModeValidateRequest(_message.Message):
    __slots__ = ("mode",)
    MODE_FIELD_NUMBER: _ClassVar[int]
    mode: str
    def __init__(self, mode: _Optional[str] = ...) -> None: ...

class OperatingModeRenderSimulationRequest(_message.Message):
    __slots__ = ("mode", "preset", "step_index")
    MODE_FIELD_NUMBER: _ClassVar[int]
    PRESET_FIELD_NUMBER: _ClassVar[int]
    STEP_INDEX_FIELD_NUMBER: _ClassVar[int]
    mode: str
    preset: str
    step_index: int
    def __init__(self, mode: _Optional[str] = ..., preset: _Optional[str] = ..., step_index: _Optional[int] = ...) -> None: ...

class OperatingModeWorkspaceRequest(_message.Message):
    __slots__ = ("initiative_name",)
    INITIATIVE_NAME_FIELD_NUMBER: _ClassVar[int]
    initiative_name: str
    def __init__(self, initiative_name: _Optional[str] = ...) -> None: ...

class OperatingModeSwitchRequest(_message.Message):
    __slots__ = ("initiative_name", "mode", "cancel_active_item_executions", "requested_by")
    INITIATIVE_NAME_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    CANCEL_ACTIVE_ITEM_EXECUTIONS_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_BY_FIELD_NUMBER: _ClassVar[int]
    initiative_name: str
    mode: str
    cancel_active_item_executions: bool
    requested_by: str
    def __init__(self, initiative_name: _Optional[str] = ..., mode: _Optional[str] = ..., cancel_active_item_executions: _Optional[bool] = ..., requested_by: _Optional[str] = ...) -> None: ...

class OperatingModeStartPhaseRequest(_message.Message):
    __slots__ = ("initiative_name", "phase", "note", "override", "requested_by")
    INITIATIVE_NAME_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_BY_FIELD_NUMBER: _ClassVar[int]
    initiative_name: str
    phase: str
    note: str
    override: bool
    requested_by: str
    def __init__(self, initiative_name: _Optional[str] = ..., phase: _Optional[str] = ..., note: _Optional[str] = ..., override: _Optional[bool] = ..., requested_by: _Optional[str] = ...) -> None: ...

class OperatingModeStartTargetPhaseRequest(_message.Message):
    __slots__ = ("mode", "target_ref", "phase", "note", "override", "requested_by")
    MODE_FIELD_NUMBER: _ClassVar[int]
    TARGET_REF_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    OVERRIDE_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_BY_FIELD_NUMBER: _ClassVar[int]
    mode: str
    target_ref: str
    phase: str
    note: str
    override: bool
    requested_by: str
    def __init__(self, mode: _Optional[str] = ..., target_ref: _Optional[str] = ..., phase: _Optional[str] = ..., note: _Optional[str] = ..., override: _Optional[bool] = ..., requested_by: _Optional[str] = ...) -> None: ...

class OperatingModeRenderLiveRequest(_message.Message):
    __slots__ = ("initiative_name", "phase", "round", "note")
    INITIATIVE_NAME_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    ROUND_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    initiative_name: str
    phase: str
    round: int
    note: str
    def __init__(self, initiative_name: _Optional[str] = ..., phase: _Optional[str] = ..., round: _Optional[int] = ..., note: _Optional[str] = ...) -> None: ...

class OperatingModeRoundActionRequest(_message.Message):
    __slots__ = ("initiative_name", "mode", "round")
    INITIATIVE_NAME_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    ROUND_FIELD_NUMBER: _ClassVar[int]
    initiative_name: str
    mode: str
    round: int
    def __init__(self, initiative_name: _Optional[str] = ..., mode: _Optional[str] = ..., round: _Optional[int] = ...) -> None: ...

class OperatingModeCompleteItemsRequest(_message.Message):
    __slots__ = ("initiative_name", "mode", "round", "run_id", "item_refs", "requested_by")
    INITIATIVE_NAME_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    ROUND_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_REFS_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_BY_FIELD_NUMBER: _ClassVar[int]
    initiative_name: str
    mode: str
    round: int
    run_id: str
    item_refs: _containers.RepeatedScalarFieldContainer[str]
    requested_by: str
    def __init__(self, initiative_name: _Optional[str] = ..., mode: _Optional[str] = ..., round: _Optional[int] = ..., run_id: _Optional[str] = ..., item_refs: _Optional[_Iterable[str]] = ..., requested_by: _Optional[str] = ...) -> None: ...

class OperatingModeApplyBacklogSyncRequest(_message.Message):
    __slots__ = ("initiative_name", "mode", "round", "run_id", "accepted_mutation_ids", "requested_by")
    INITIATIVE_NAME_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    ROUND_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_MUTATION_IDS_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_BY_FIELD_NUMBER: _ClassVar[int]
    initiative_name: str
    mode: str
    round: int
    run_id: str
    accepted_mutation_ids: _containers.RepeatedScalarFieldContainer[str]
    requested_by: str
    def __init__(self, initiative_name: _Optional[str] = ..., mode: _Optional[str] = ..., round: _Optional[int] = ..., run_id: _Optional[str] = ..., accepted_mutation_ids: _Optional[_Iterable[str]] = ..., requested_by: _Optional[str] = ...) -> None: ...

class OperatingModeCapabilities(_message.Message):
    __slots__ = ("supports_phases", "can_start_phases", "can_complete_items", "can_apply_backlog_sync_proposals", "requires_acceptance_criteria", "supports_artifacts", "supports_handoffs", "uses_item_execution_flow")
    SUPPORTS_PHASES_FIELD_NUMBER: _ClassVar[int]
    CAN_START_PHASES_FIELD_NUMBER: _ClassVar[int]
    CAN_COMPLETE_ITEMS_FIELD_NUMBER: _ClassVar[int]
    CAN_APPLY_BACKLOG_SYNC_PROPOSALS_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_ACCEPTANCE_CRITERIA_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_HANDOFFS_FIELD_NUMBER: _ClassVar[int]
    USES_ITEM_EXECUTION_FLOW_FIELD_NUMBER: _ClassVar[int]
    supports_phases: bool
    can_start_phases: bool
    can_complete_items: bool
    can_apply_backlog_sync_proposals: bool
    requires_acceptance_criteria: bool
    supports_artifacts: bool
    supports_handoffs: bool
    uses_item_execution_flow: bool
    def __init__(self, supports_phases: _Optional[bool] = ..., can_start_phases: _Optional[bool] = ..., can_complete_items: _Optional[bool] = ..., can_apply_backlog_sync_proposals: _Optional[bool] = ..., requires_acceptance_criteria: _Optional[bool] = ..., supports_artifacts: _Optional[bool] = ..., supports_handoffs: _Optional[bool] = ..., uses_item_execution_flow: _Optional[bool] = ...) -> None: ...

class OperatingModeArtifactDefinition(_message.Message):
    __slots__ = ("path", "content_type", "required")
    PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    path: str
    content_type: str
    required: bool
    def __init__(self, path: _Optional[str] = ..., content_type: _Optional[str] = ..., required: _Optional[bool] = ...) -> None: ...

class OperatingModeResultBindingSummary(_message.Message):
    __slots__ = ("kind", "artifact")
    KIND_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_FIELD_NUMBER: _ClassVar[int]
    kind: str
    artifact: OperatingModeArtifactDefinition
    def __init__(self, kind: _Optional[str] = ..., artifact: _Optional[_Union[OperatingModeArtifactDefinition, _Mapping]] = ...) -> None: ...

class OperatingModePhaseOutputContractSummary(_message.Message):
    __slots__ = ("requires_structured_result", "requires_progress", "requires_verdict", "requires_handoff", "requires_backlog_sync", "required_artifact_count")
    REQUIRES_STRUCTURED_RESULT_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_PROGRESS_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_VERDICT_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_HANDOFF_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_BACKLOG_SYNC_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_ARTIFACT_COUNT_FIELD_NUMBER: _ClassVar[int]
    requires_structured_result: bool
    requires_progress: bool
    requires_verdict: bool
    requires_handoff: bool
    requires_backlog_sync: bool
    required_artifact_count: int
    def __init__(self, requires_structured_result: _Optional[bool] = ..., requires_progress: _Optional[bool] = ..., requires_verdict: _Optional[bool] = ..., requires_handoff: _Optional[bool] = ..., requires_backlog_sync: _Optional[bool] = ..., required_artifact_count: _Optional[int] = ...) -> None: ...

class OperatingModeReadinessDimension(_message.Message):
    __slots__ = ("key", "label", "score", "rationale")
    KEY_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    key: str
    label: str
    score: float
    rationale: str
    def __init__(self, key: _Optional[str] = ..., label: _Optional[str] = ..., score: _Optional[float] = ..., rationale: _Optional[str] = ...) -> None: ...

class OperatingModeReadinessReport(_message.Message):
    __slots__ = ("dimensions", "overall_score", "ready")
    DIMENSIONS_FIELD_NUMBER: _ClassVar[int]
    OVERALL_SCORE_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    dimensions: _containers.RepeatedCompositeFieldContainer[OperatingModeReadinessDimension]
    overall_score: float
    ready: bool
    def __init__(self, dimensions: _Optional[_Iterable[_Union[OperatingModeReadinessDimension, _Mapping]]] = ..., overall_score: _Optional[float] = ..., ready: _Optional[bool] = ...) -> None: ...

class OperatingModeHandoff(_message.Message):
    __slots__ = ("summary", "completed_phases", "changed_files", "tests", "blockers", "next_step", "created_at", "frontier")
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_PHASES_FIELD_NUMBER: _ClassVar[int]
    CHANGED_FILES_FIELD_NUMBER: _ClassVar[int]
    TESTS_FIELD_NUMBER: _ClassVar[int]
    BLOCKERS_FIELD_NUMBER: _ClassVar[int]
    NEXT_STEP_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    FRONTIER_FIELD_NUMBER: _ClassVar[int]
    summary: str
    completed_phases: _containers.RepeatedScalarFieldContainer[str]
    changed_files: _containers.RepeatedScalarFieldContainer[str]
    tests: _containers.RepeatedScalarFieldContainer[str]
    blockers: _containers.RepeatedScalarFieldContainer[str]
    next_step: str
    created_at: str
    frontier: str
    def __init__(self, summary: _Optional[str] = ..., completed_phases: _Optional[_Iterable[str]] = ..., changed_files: _Optional[_Iterable[str]] = ..., tests: _Optional[_Iterable[str]] = ..., blockers: _Optional[_Iterable[str]] = ..., next_step: _Optional[str] = ..., created_at: _Optional[str] = ..., frontier: _Optional[str] = ...) -> None: ...

class OperatingModeRoundItem(_message.Message):
    __slots__ = ("ref", "title", "status", "priority", "effort")
    REF_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    EFFORT_FIELD_NUMBER: _ClassVar[int]
    ref: str
    title: str
    status: str
    priority: int
    effort: str
    def __init__(self, ref: _Optional[str] = ..., title: _Optional[str] = ..., status: _Optional[str] = ..., priority: _Optional[int] = ..., effort: _Optional[str] = ...) -> None: ...

class OperatingModeArtifactUpdate(_message.Message):
    __slots__ = ("path", "content_type", "required", "updated_at", "source")
    PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    path: str
    content_type: str
    required: bool
    updated_at: str
    source: str
    def __init__(self, path: _Optional[str] = ..., content_type: _Optional[str] = ..., required: _Optional[bool] = ..., updated_at: _Optional[str] = ..., source: _Optional[str] = ...) -> None: ...

class OperatingModeArtifactSnapshot(_message.Message):
    __slots__ = ("path", "content_type", "required", "content", "updated_at", "size_bytes")
    PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    path: str
    content_type: str
    required: bool
    content: str
    updated_at: str
    size_bytes: int
    def __init__(self, path: _Optional[str] = ..., content_type: _Optional[str] = ..., required: _Optional[bool] = ..., content: _Optional[str] = ..., updated_at: _Optional[str] = ..., size_bytes: _Optional[int] = ...) -> None: ...

class OperatingModeInitiativeSnapshot(_message.Message):
    __slots__ = ("name", "title", "description", "mode", "items", "acceptance_criteria")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_CRITERIA_FIELD_NUMBER: _ClassVar[int]
    name: str
    title: str
    description: str
    mode: str
    items: _containers.RepeatedScalarFieldContainer[str]
    acceptance_criteria: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., mode: _Optional[str] = ..., items: _Optional[_Iterable[str]] = ..., acceptance_criteria: _Optional[_Iterable[str]] = ...) -> None: ...

class OperatingModeProgressState(_message.Message):
    __slots__ = ("decision", "completed_phases", "current_phase", "rationale", "updated_at")
    DECISION_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_PHASES_FIELD_NUMBER: _ClassVar[int]
    CURRENT_PHASE_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    decision: str
    completed_phases: _containers.RepeatedScalarFieldContainer[str]
    current_phase: str
    rationale: str
    updated_at: str
    def __init__(self, decision: _Optional[str] = ..., completed_phases: _Optional[_Iterable[str]] = ..., current_phase: _Optional[str] = ..., rationale: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class OperatingModePhaseResolutionRecord(_message.Message):
    __slots__ = ("outcome", "layer", "chosen_message_index", "messages_scanned", "missing", "violations", "notes", "classified_field", "classified_value")
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    LAYER_FIELD_NUMBER: _ClassVar[int]
    CHOSEN_MESSAGE_INDEX_FIELD_NUMBER: _ClassVar[int]
    MESSAGES_SCANNED_FIELD_NUMBER: _ClassVar[int]
    MISSING_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    CLASSIFIED_FIELD_FIELD_NUMBER: _ClassVar[int]
    CLASSIFIED_VALUE_FIELD_NUMBER: _ClassVar[int]
    outcome: str
    layer: str
    chosen_message_index: int
    messages_scanned: int
    missing: _containers.RepeatedScalarFieldContainer[str]
    violations: _containers.RepeatedScalarFieldContainer[str]
    notes: _containers.RepeatedScalarFieldContainer[str]
    classified_field: str
    classified_value: str
    def __init__(self, outcome: _Optional[str] = ..., layer: _Optional[str] = ..., chosen_message_index: _Optional[int] = ..., messages_scanned: _Optional[int] = ..., missing: _Optional[_Iterable[str]] = ..., violations: _Optional[_Iterable[str]] = ..., notes: _Optional[_Iterable[str]] = ..., classified_field: _Optional[str] = ..., classified_value: _Optional[str] = ...) -> None: ...

class OperatingModeArtifactResult(_message.Message):
    __slots__ = ("path", "content", "content_type")
    PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    path: str
    content: str
    content_type: str
    def __init__(self, path: _Optional[str] = ..., content: _Optional[str] = ..., content_type: _Optional[str] = ...) -> None: ...

class OperatingModeBacklogSyncPlan(_message.Message):
    __slots__ = ("completed_items", "created_items", "updated_items", "proposal", "rationale")
    COMPLETED_ITEMS_FIELD_NUMBER: _ClassVar[int]
    CREATED_ITEMS_FIELD_NUMBER: _ClassVar[int]
    UPDATED_ITEMS_FIELD_NUMBER: _ClassVar[int]
    PROPOSAL_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    completed_items: _containers.RepeatedScalarFieldContainer[str]
    created_items: _containers.RepeatedScalarFieldContainer[str]
    updated_items: _containers.RepeatedScalarFieldContainer[str]
    proposal: _struct_pb2.Struct
    rationale: str
    def __init__(self, completed_items: _Optional[_Iterable[str]] = ..., created_items: _Optional[_Iterable[str]] = ..., updated_items: _Optional[_Iterable[str]] = ..., proposal: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., rationale: _Optional[str] = ...) -> None: ...

class OperatingModeRoundEnvelope(_message.Message):
    __slots__ = ("round", "mode", "scope_kind", "scope_id", "initiative_name", "phase", "run_strategy", "agent_profile_key", "generated_at", "run_id", "status", "readiness", "items", "artifact_updates", "handoffs", "payload", "error", "resolution", "transition_classification")
    ROUND_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    SCOPE_KIND_FIELD_NUMBER: _ClassVar[int]
    SCOPE_ID_FIELD_NUMBER: _ClassVar[int]
    INITIATIVE_NAME_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    RUN_STRATEGY_FIELD_NUMBER: _ClassVar[int]
    AGENT_PROFILE_KEY_FIELD_NUMBER: _ClassVar[int]
    GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    READINESS_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_UPDATES_FIELD_NUMBER: _ClassVar[int]
    HANDOFFS_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    RESOLUTION_FIELD_NUMBER: _ClassVar[int]
    TRANSITION_CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    round: int
    mode: str
    scope_kind: str
    scope_id: str
    initiative_name: str
    phase: str
    run_strategy: str
    agent_profile_key: str
    generated_at: str
    run_id: str
    status: str
    readiness: OperatingModeReadinessReport
    items: _containers.RepeatedCompositeFieldContainer[OperatingModeRoundItem]
    artifact_updates: _containers.RepeatedCompositeFieldContainer[OperatingModeArtifactUpdate]
    handoffs: _containers.RepeatedCompositeFieldContainer[OperatingModeHandoff]
    payload: _struct_pb2.Struct
    error: str
    resolution: OperatingModePhaseResolutionRecord
    transition_classification: OperatingModePhaseResolutionRecord
    def __init__(self, round: _Optional[int] = ..., mode: _Optional[str] = ..., scope_kind: _Optional[str] = ..., scope_id: _Optional[str] = ..., initiative_name: _Optional[str] = ..., phase: _Optional[str] = ..., run_strategy: _Optional[str] = ..., agent_profile_key: _Optional[str] = ..., generated_at: _Optional[str] = ..., run_id: _Optional[str] = ..., status: _Optional[str] = ..., readiness: _Optional[_Union[OperatingModeReadinessReport, _Mapping]] = ..., items: _Optional[_Iterable[_Union[OperatingModeRoundItem, _Mapping]]] = ..., artifact_updates: _Optional[_Iterable[_Union[OperatingModeArtifactUpdate, _Mapping]]] = ..., handoffs: _Optional[_Iterable[_Union[OperatingModeHandoff, _Mapping]]] = ..., payload: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., error: _Optional[str] = ..., resolution: _Optional[_Union[OperatingModePhaseResolutionRecord, _Mapping]] = ..., transition_classification: _Optional[_Union[OperatingModePhaseResolutionRecord, _Mapping]] = ...) -> None: ...

class OperatingModeCatalogPhase(_message.Message):
    __slots__ = ("phase", "phase_kind", "label", "title", "purpose", "trigger", "profile_key", "writes_repo", "requires_criteria", "is_start", "is_terminal", "output_artifacts", "output_contract", "catalog_id", "skill_id", "activity_purpose", "lock_purpose", "result_bindings", "samples_replan_rate", "samples_acceptance_rate", "auto_start_after", "reads", "executed_by", "classification")
    PHASE_FIELD_NUMBER: _ClassVar[int]
    PHASE_KIND_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    PURPOSE_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_FIELD_NUMBER: _ClassVar[int]
    PROFILE_KEY_FIELD_NUMBER: _ClassVar[int]
    WRITES_REPO_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_CRITERIA_FIELD_NUMBER: _ClassVar[int]
    IS_START_FIELD_NUMBER: _ClassVar[int]
    IS_TERMINAL_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_CONTRACT_FIELD_NUMBER: _ClassVar[int]
    CATALOG_ID_FIELD_NUMBER: _ClassVar[int]
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    ACTIVITY_PURPOSE_FIELD_NUMBER: _ClassVar[int]
    LOCK_PURPOSE_FIELD_NUMBER: _ClassVar[int]
    RESULT_BINDINGS_FIELD_NUMBER: _ClassVar[int]
    SAMPLES_REPLAN_RATE_FIELD_NUMBER: _ClassVar[int]
    SAMPLES_ACCEPTANCE_RATE_FIELD_NUMBER: _ClassVar[int]
    AUTO_START_AFTER_FIELD_NUMBER: _ClassVar[int]
    READS_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_BY_FIELD_NUMBER: _ClassVar[int]
    CLASSIFICATION_FIELD_NUMBER: _ClassVar[int]
    phase: str
    phase_kind: str
    label: str
    title: str
    purpose: str
    trigger: str
    profile_key: str
    writes_repo: bool
    requires_criteria: bool
    is_start: bool
    is_terminal: bool
    output_artifacts: _containers.RepeatedCompositeFieldContainer[OperatingModeArtifactDefinition]
    output_contract: OperatingModePhaseOutputContractSummary
    catalog_id: str
    skill_id: str
    activity_purpose: str
    lock_purpose: str
    result_bindings: _containers.RepeatedCompositeFieldContainer[OperatingModeResultBindingSummary]
    samples_replan_rate: bool
    samples_acceptance_rate: bool
    auto_start_after: _containers.RepeatedScalarFieldContainer[str]
    reads: OperatingModePhaseReads
    executed_by: str
    classification: OperatingModeTransitionClassification
    def __init__(self, phase: _Optional[str] = ..., phase_kind: _Optional[str] = ..., label: _Optional[str] = ..., title: _Optional[str] = ..., purpose: _Optional[str] = ..., trigger: _Optional[str] = ..., profile_key: _Optional[str] = ..., writes_repo: _Optional[bool] = ..., requires_criteria: _Optional[bool] = ..., is_start: _Optional[bool] = ..., is_terminal: _Optional[bool] = ..., output_artifacts: _Optional[_Iterable[_Union[OperatingModeArtifactDefinition, _Mapping]]] = ..., output_contract: _Optional[_Union[OperatingModePhaseOutputContractSummary, _Mapping]] = ..., catalog_id: _Optional[str] = ..., skill_id: _Optional[str] = ..., activity_purpose: _Optional[str] = ..., lock_purpose: _Optional[str] = ..., result_bindings: _Optional[_Iterable[_Union[OperatingModeResultBindingSummary, _Mapping]]] = ..., samples_replan_rate: _Optional[bool] = ..., samples_acceptance_rate: _Optional[bool] = ..., auto_start_after: _Optional[_Iterable[str]] = ..., reads: _Optional[_Union[OperatingModePhaseReads, _Mapping]] = ..., executed_by: _Optional[str] = ..., classification: _Optional[_Union[OperatingModeTransitionClassification, _Mapping]] = ...) -> None: ...

class OperatingModeTransitionClassification(_message.Message):
    __slots__ = ("field", "enum", "description")
    FIELD_FIELD_NUMBER: _ClassVar[int]
    ENUM_FIELD_NUMBER: _ClassVar[int]
    FROM_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    field: str
    enum: _containers.RepeatedScalarFieldContainer[str]
    description: str
    def __init__(self, field: _Optional[str] = ..., enum: _Optional[_Iterable[str]] = ..., description: _Optional[str] = ..., **kwargs) -> None: ...

class OperatingModePhaseReads(_message.Message):
    __slots__ = ("base", "target")
    BASE_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    base: _containers.RepeatedScalarFieldContainer[str]
    target: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, base: _Optional[_Iterable[str]] = ..., target: _Optional[_Iterable[str]] = ...) -> None: ...

class OperatingModeCatalogTransition(_message.Message):
    __slots__ = ("to", "condition_kind", "label", "field", "value", "classified")
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    CONDITION_KIND_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    CLASSIFIED_FIELD_NUMBER: _ClassVar[int]
    to: str
    condition_kind: str
    label: str
    field: str
    value: str
    classified: bool
    def __init__(self, to: _Optional[str] = ..., condition_kind: _Optional[str] = ..., label: _Optional[str] = ..., field: _Optional[str] = ..., value: _Optional[str] = ..., classified: _Optional[bool] = ..., **kwargs) -> None: ...

class OperatingModeCatalogPhaseGraph(_message.Message):
    __slots__ = ("start_phase", "terminal", "transitions", "accepted_verdicts")
    START_PHASE_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_FIELD_NUMBER: _ClassVar[int]
    TRANSITIONS_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_VERDICTS_FIELD_NUMBER: _ClassVar[int]
    start_phase: str
    terminal: _containers.RepeatedScalarFieldContainer[str]
    transitions: _containers.RepeatedCompositeFieldContainer[OperatingModeCatalogTransition]
    accepted_verdicts: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, start_phase: _Optional[str] = ..., terminal: _Optional[_Iterable[str]] = ..., transitions: _Optional[_Iterable[_Union[OperatingModeCatalogTransition, _Mapping]]] = ..., accepted_verdicts: _Optional[_Iterable[str]] = ...) -> None: ...

class OperatingModeCatalogEntry(_message.Message):
    __slots__ = ("mode", "label", "description", "best_for", "not_for", "tradeoffs", "when_in_doubt_pick_instead", "usage_count", "run_strategy", "workspace_tab_id", "capabilities", "default", "switchable", "supports_phases", "phases", "phase_graph", "target_kind")
    MODE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    BEST_FOR_FIELD_NUMBER: _ClassVar[int]
    NOT_FOR_FIELD_NUMBER: _ClassVar[int]
    TRADEOFFS_FIELD_NUMBER: _ClassVar[int]
    WHEN_IN_DOUBT_PICK_INSTEAD_FIELD_NUMBER: _ClassVar[int]
    USAGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    RUN_STRATEGY_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_TAB_ID_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_FIELD_NUMBER: _ClassVar[int]
    SWITCHABLE_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_PHASES_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    PHASE_GRAPH_FIELD_NUMBER: _ClassVar[int]
    TARGET_KIND_FIELD_NUMBER: _ClassVar[int]
    mode: str
    label: str
    description: str
    best_for: _containers.RepeatedScalarFieldContainer[str]
    not_for: _containers.RepeatedScalarFieldContainer[str]
    tradeoffs: _containers.RepeatedScalarFieldContainer[str]
    when_in_doubt_pick_instead: str
    usage_count: int
    run_strategy: str
    workspace_tab_id: str
    capabilities: OperatingModeCapabilities
    default: bool
    switchable: bool
    supports_phases: bool
    phases: _containers.RepeatedCompositeFieldContainer[OperatingModeCatalogPhase]
    phase_graph: OperatingModeCatalogPhaseGraph
    target_kind: str
    def __init__(self, mode: _Optional[str] = ..., label: _Optional[str] = ..., description: _Optional[str] = ..., best_for: _Optional[_Iterable[str]] = ..., not_for: _Optional[_Iterable[str]] = ..., tradeoffs: _Optional[_Iterable[str]] = ..., when_in_doubt_pick_instead: _Optional[str] = ..., usage_count: _Optional[int] = ..., run_strategy: _Optional[str] = ..., workspace_tab_id: _Optional[str] = ..., capabilities: _Optional[_Union[OperatingModeCapabilities, _Mapping]] = ..., default: _Optional[bool] = ..., switchable: _Optional[bool] = ..., supports_phases: _Optional[bool] = ..., phases: _Optional[_Iterable[_Union[OperatingModeCatalogPhase, _Mapping]]] = ..., phase_graph: _Optional[_Union[OperatingModeCatalogPhaseGraph, _Mapping]] = ..., target_kind: _Optional[str] = ...) -> None: ...

class OperatingModeCatalogResponse(_message.Message):
    __slots__ = ("modes",)
    MODES_FIELD_NUMBER: _ClassVar[int]
    modes: _containers.RepeatedCompositeFieldContainer[OperatingModeCatalogEntry]
    def __init__(self, modes: _Optional[_Iterable[_Union[OperatingModeCatalogEntry, _Mapping]]] = ...) -> None: ...

class OperatingModeInitiativeRef(_message.Message):
    __slots__ = ("name", "title", "status", "updated")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    UPDATED_FIELD_NUMBER: _ClassVar[int]
    name: str
    title: str
    status: str
    updated: str
    def __init__(self, name: _Optional[str] = ..., title: _Optional[str] = ..., status: _Optional[str] = ..., updated: _Optional[str] = ...) -> None: ...

class OperatingModeDetailResponse(_message.Message):
    __slots__ = ("entry", "linked_initiatives")
    ENTRY_FIELD_NUMBER: _ClassVar[int]
    LINKED_INITIATIVES_FIELD_NUMBER: _ClassVar[int]
    entry: OperatingModeCatalogEntry
    linked_initiatives: _containers.RepeatedCompositeFieldContainer[OperatingModeInitiativeRef]
    def __init__(self, entry: _Optional[_Union[OperatingModeCatalogEntry, _Mapping]] = ..., linked_initiatives: _Optional[_Iterable[_Union[OperatingModeInitiativeRef, _Mapping]]] = ...) -> None: ...

class OperatingModeStringList(_message.Message):
    __slots__ = ("values",)
    VALUES_FIELD_NUMBER: _ClassVar[int]
    values: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, values: _Optional[_Iterable[str]] = ...) -> None: ...

class OperatingModeWorkspacePhase(_message.Message):
    __slots__ = ("phase", "phase_kind", "activity_purpose", "profile_key", "writes_repo", "output_artifacts", "requires_criteria", "startable", "reason", "next", "auto_start_after", "executed_by")
    PHASE_FIELD_NUMBER: _ClassVar[int]
    PHASE_KIND_FIELD_NUMBER: _ClassVar[int]
    ACTIVITY_PURPOSE_FIELD_NUMBER: _ClassVar[int]
    PROFILE_KEY_FIELD_NUMBER: _ClassVar[int]
    WRITES_REPO_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_CRITERIA_FIELD_NUMBER: _ClassVar[int]
    STARTABLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    NEXT_FIELD_NUMBER: _ClassVar[int]
    AUTO_START_AFTER_FIELD_NUMBER: _ClassVar[int]
    EXECUTED_BY_FIELD_NUMBER: _ClassVar[int]
    phase: str
    phase_kind: str
    activity_purpose: str
    profile_key: str
    writes_repo: bool
    output_artifacts: _containers.RepeatedCompositeFieldContainer[OperatingModeArtifactDefinition]
    requires_criteria: bool
    startable: bool
    reason: str
    next: bool
    auto_start_after: _containers.RepeatedScalarFieldContainer[str]
    executed_by: str
    def __init__(self, phase: _Optional[str] = ..., phase_kind: _Optional[str] = ..., activity_purpose: _Optional[str] = ..., profile_key: _Optional[str] = ..., writes_repo: _Optional[bool] = ..., output_artifacts: _Optional[_Iterable[_Union[OperatingModeArtifactDefinition, _Mapping]]] = ..., requires_criteria: _Optional[bool] = ..., startable: _Optional[bool] = ..., reason: _Optional[str] = ..., next: _Optional[bool] = ..., auto_start_after: _Optional[_Iterable[str]] = ..., executed_by: _Optional[str] = ...) -> None: ...

class OperatingModeWorkspaceMode(_message.Message):
    __slots__ = ("mode", "label", "description", "capabilities", "phases", "terminal", "transitions", "run_strategy", "target_kind")
    class TransitionsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: OperatingModeStringList
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[OperatingModeStringList, _Mapping]] = ...) -> None: ...
    MODE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_FIELD_NUMBER: _ClassVar[int]
    TRANSITIONS_FIELD_NUMBER: _ClassVar[int]
    RUN_STRATEGY_FIELD_NUMBER: _ClassVar[int]
    TARGET_KIND_FIELD_NUMBER: _ClassVar[int]
    mode: str
    label: str
    description: str
    capabilities: OperatingModeCapabilities
    phases: _containers.RepeatedCompositeFieldContainer[OperatingModeWorkspacePhase]
    terminal: _containers.RepeatedScalarFieldContainer[str]
    transitions: _containers.MessageMap[str, OperatingModeStringList]
    run_strategy: str
    target_kind: str
    def __init__(self, mode: _Optional[str] = ..., label: _Optional[str] = ..., description: _Optional[str] = ..., capabilities: _Optional[_Union[OperatingModeCapabilities, _Mapping]] = ..., phases: _Optional[_Iterable[_Union[OperatingModeWorkspacePhase, _Mapping]]] = ..., terminal: _Optional[_Iterable[str]] = ..., transitions: _Optional[_Mapping[str, OperatingModeStringList]] = ..., run_strategy: _Optional[str] = ..., target_kind: _Optional[str] = ...) -> None: ...

class OperatingModeLockHolder(_message.Message):
    __slots__ = ("run_id", "purpose", "round_number", "acquired_at", "acquired_by", "initiative_name")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    PURPOSE_FIELD_NUMBER: _ClassVar[int]
    ROUND_NUMBER_FIELD_NUMBER: _ClassVar[int]
    ACQUIRED_AT_FIELD_NUMBER: _ClassVar[int]
    ACQUIRED_BY_FIELD_NUMBER: _ClassVar[int]
    INITIATIVE_NAME_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    purpose: str
    round_number: int
    acquired_at: str
    acquired_by: str
    initiative_name: str
    def __init__(self, run_id: _Optional[str] = ..., purpose: _Optional[str] = ..., round_number: _Optional[int] = ..., acquired_at: _Optional[str] = ..., acquired_by: _Optional[str] = ..., initiative_name: _Optional[str] = ...) -> None: ...

class OperatingModeWorkspace(_message.Message):
    __slots__ = ("initiative_name", "mode", "definition", "lock", "artifacts", "rounds")
    INITIATIVE_NAME_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_FIELD_NUMBER: _ClassVar[int]
    LOCK_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    ROUNDS_FIELD_NUMBER: _ClassVar[int]
    initiative_name: str
    mode: str
    definition: OperatingModeWorkspaceMode
    lock: OperatingModeLockHolder
    artifacts: _containers.RepeatedCompositeFieldContainer[OperatingModeArtifactSnapshot]
    rounds: _containers.RepeatedCompositeFieldContainer[OperatingModeRoundEnvelope]
    def __init__(self, initiative_name: _Optional[str] = ..., mode: _Optional[str] = ..., definition: _Optional[_Union[OperatingModeWorkspaceMode, _Mapping]] = ..., lock: _Optional[_Union[OperatingModeLockHolder, _Mapping]] = ..., artifacts: _Optional[_Iterable[_Union[OperatingModeArtifactSnapshot, _Mapping]]] = ..., rounds: _Optional[_Iterable[_Union[OperatingModeRoundEnvelope, _Mapping]]] = ...) -> None: ...

class OperatingModeActiveItemExecution(_message.Message):
    __slots__ = ("item_ref", "execution_id", "run_id", "status")
    ITEM_REF_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    item_ref: str
    execution_id: str
    run_id: str
    status: str
    def __init__(self, item_ref: _Optional[str] = ..., execution_id: _Optional[str] = ..., run_id: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class OperatingModeSwitchResult(_message.Message):
    __slots__ = ("initiative_name", "from_mode", "to_mode", "canceled_item_executions", "active_item_executions", "requires_cancellation", "operating_mode_workspace_id")
    INITIATIVE_NAME_FIELD_NUMBER: _ClassVar[int]
    FROM_MODE_FIELD_NUMBER: _ClassVar[int]
    TO_MODE_FIELD_NUMBER: _ClassVar[int]
    CANCELED_ITEM_EXECUTIONS_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_ITEM_EXECUTIONS_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_CANCELLATION_FIELD_NUMBER: _ClassVar[int]
    OPERATING_MODE_WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    initiative_name: str
    from_mode: str
    to_mode: str
    canceled_item_executions: _containers.RepeatedCompositeFieldContainer[OperatingModeActiveItemExecution]
    active_item_executions: _containers.RepeatedCompositeFieldContainer[OperatingModeActiveItemExecution]
    requires_cancellation: bool
    operating_mode_workspace_id: str
    def __init__(self, initiative_name: _Optional[str] = ..., from_mode: _Optional[str] = ..., to_mode: _Optional[str] = ..., canceled_item_executions: _Optional[_Iterable[_Union[OperatingModeActiveItemExecution, _Mapping]]] = ..., active_item_executions: _Optional[_Iterable[_Union[OperatingModeActiveItemExecution, _Mapping]]] = ..., requires_cancellation: _Optional[bool] = ..., operating_mode_workspace_id: _Optional[str] = ...) -> None: ...

class OperatingModeActiveItemExecutionsConflict(_message.Message):
    __slots__ = ("initiative_name", "from_mode", "to_mode", "executions")
    INITIATIVE_NAME_FIELD_NUMBER: _ClassVar[int]
    FROM_MODE_FIELD_NUMBER: _ClassVar[int]
    TO_MODE_FIELD_NUMBER: _ClassVar[int]
    EXECUTIONS_FIELD_NUMBER: _ClassVar[int]
    initiative_name: str
    from_mode: str
    to_mode: str
    executions: _containers.RepeatedCompositeFieldContainer[OperatingModeActiveItemExecution]
    def __init__(self, initiative_name: _Optional[str] = ..., from_mode: _Optional[str] = ..., to_mode: _Optional[str] = ..., executions: _Optional[_Iterable[_Union[OperatingModeActiveItemExecution, _Mapping]]] = ...) -> None: ...

class OperatingModeBacklogCompletionResult(_message.Message):
    __slots__ = ("item_ref", "from_status", "to_status")
    ITEM_REF_FIELD_NUMBER: _ClassVar[int]
    FROM_STATUS_FIELD_NUMBER: _ClassVar[int]
    TO_STATUS_FIELD_NUMBER: _ClassVar[int]
    item_ref: str
    from_status: str
    to_status: str
    def __init__(self, item_ref: _Optional[str] = ..., from_status: _Optional[str] = ..., to_status: _Optional[str] = ...) -> None: ...

class OperatingModeProposalOutcome(_message.Message):
    __slots__ = ("mutation_id", "op", "target", "applied", "skipped", "error")
    MUTATION_ID_FIELD_NUMBER: _ClassVar[int]
    OP_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    mutation_id: str
    op: str
    target: str
    applied: bool
    skipped: bool
    error: str
    def __init__(self, mutation_id: _Optional[str] = ..., op: _Optional[str] = ..., target: _Optional[str] = ..., applied: _Optional[bool] = ..., skipped: _Optional[bool] = ..., error: _Optional[str] = ...) -> None: ...

class OperatingModeProposalApplyResult(_message.Message):
    __slots__ = ("outcomes", "applied", "failed", "skipped", "created", "updated")
    OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    UPDATED_FIELD_NUMBER: _ClassVar[int]
    outcomes: _containers.RepeatedCompositeFieldContainer[OperatingModeProposalOutcome]
    applied: int
    failed: int
    skipped: int
    created: int
    updated: int
    def __init__(self, outcomes: _Optional[_Iterable[_Union[OperatingModeProposalOutcome, _Mapping]]] = ..., applied: _Optional[int] = ..., failed: _Optional[int] = ..., skipped: _Optional[int] = ..., created: _Optional[int] = ..., updated: _Optional[int] = ...) -> None: ...

class OperatingModeBacklogSyncResult(_message.Message):
    __slots__ = ("initiative_name", "mode", "phase", "round", "run_id", "completed_items", "proposal_result", "noop")
    INITIATIVE_NAME_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    ROUND_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_ITEMS_FIELD_NUMBER: _ClassVar[int]
    PROPOSAL_RESULT_FIELD_NUMBER: _ClassVar[int]
    NOOP_FIELD_NUMBER: _ClassVar[int]
    initiative_name: str
    mode: str
    phase: str
    round: int
    run_id: str
    completed_items: _containers.RepeatedCompositeFieldContainer[OperatingModeBacklogCompletionResult]
    proposal_result: OperatingModeProposalApplyResult
    noop: bool
    def __init__(self, initiative_name: _Optional[str] = ..., mode: _Optional[str] = ..., phase: _Optional[str] = ..., round: _Optional[int] = ..., run_id: _Optional[str] = ..., completed_items: _Optional[_Iterable[_Union[OperatingModeBacklogCompletionResult, _Mapping]]] = ..., proposal_result: _Optional[_Union[OperatingModeProposalApplyResult, _Mapping]] = ..., noop: _Optional[bool] = ...) -> None: ...

class OperatingModePhaseResult(_message.Message):
    __slots__ = ("artifacts", "handoff", "handoffs", "readiness", "progress", "verdict", "replan_needed", "backlog_sync")
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    HANDOFF_FIELD_NUMBER: _ClassVar[int]
    HANDOFFS_FIELD_NUMBER: _ClassVar[int]
    READINESS_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    REPLAN_NEEDED_FIELD_NUMBER: _ClassVar[int]
    BACKLOG_SYNC_FIELD_NUMBER: _ClassVar[int]
    artifacts: _containers.RepeatedCompositeFieldContainer[OperatingModeArtifactResult]
    handoff: OperatingModeHandoff
    handoffs: _containers.RepeatedCompositeFieldContainer[OperatingModeHandoff]
    readiness: OperatingModeReadinessReport
    progress: OperatingModeProgressState
    verdict: str
    replan_needed: bool
    backlog_sync: OperatingModeBacklogSyncPlan
    def __init__(self, artifacts: _Optional[_Iterable[_Union[OperatingModeArtifactResult, _Mapping]]] = ..., handoff: _Optional[_Union[OperatingModeHandoff, _Mapping]] = ..., handoffs: _Optional[_Iterable[_Union[OperatingModeHandoff, _Mapping]]] = ..., readiness: _Optional[_Union[OperatingModeReadinessReport, _Mapping]] = ..., progress: _Optional[_Union[OperatingModeProgressState, _Mapping]] = ..., verdict: _Optional[str] = ..., replan_needed: _Optional[bool] = ..., backlog_sync: _Optional[_Union[OperatingModeBacklogSyncPlan, _Mapping]] = ...) -> None: ...

class OperatingModeSimulationInputs(_message.Message):
    __slots__ = ("initiative", "items", "artifacts", "prior_rounds", "acceptance_criteria")
    INITIATIVE_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    PRIOR_ROUNDS_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_CRITERIA_FIELD_NUMBER: _ClassVar[int]
    initiative: OperatingModeInitiativeSnapshot
    items: _containers.RepeatedCompositeFieldContainer[OperatingModeRoundItem]
    artifacts: _containers.RepeatedCompositeFieldContainer[OperatingModeArtifactSnapshot]
    prior_rounds: _containers.RepeatedCompositeFieldContainer[OperatingModeRoundEnvelope]
    acceptance_criteria: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, initiative: _Optional[_Union[OperatingModeInitiativeSnapshot, _Mapping]] = ..., items: _Optional[_Iterable[_Union[OperatingModeRoundItem, _Mapping]]] = ..., artifacts: _Optional[_Iterable[_Union[OperatingModeArtifactSnapshot, _Mapping]]] = ..., prior_rounds: _Optional[_Iterable[_Union[OperatingModeRoundEnvelope, _Mapping]]] = ..., acceptance_criteria: _Optional[_Iterable[str]] = ...) -> None: ...

class OperatingModeSimulationTransition(_message.Message):
    __slots__ = ("to", "condition_kind", "label", "field", "value")
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    CONDITION_KIND_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    to: str
    condition_kind: str
    label: str
    field: str
    value: str
    def __init__(self, to: _Optional[str] = ..., condition_kind: _Optional[str] = ..., label: _Optional[str] = ..., field: _Optional[str] = ..., value: _Optional[str] = ..., **kwargs) -> None: ...

class OperatingModeSimulationStep(_message.Message):
    __slots__ = ("index", "phase", "phase_kind", "inputs", "output", "round", "transition", "terminal", "skill_id", "profile_key", "prompt_variables")
    class PromptVariablesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    INDEX_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    PHASE_KIND_FIELD_NUMBER: _ClassVar[int]
    INPUTS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    ROUND_FIELD_NUMBER: _ClassVar[int]
    TRANSITION_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_FIELD_NUMBER: _ClassVar[int]
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_KEY_FIELD_NUMBER: _ClassVar[int]
    PROMPT_VARIABLES_FIELD_NUMBER: _ClassVar[int]
    index: int
    phase: str
    phase_kind: str
    inputs: OperatingModeSimulationInputs
    output: OperatingModePhaseResult
    round: OperatingModeRoundEnvelope
    transition: OperatingModeSimulationTransition
    terminal: bool
    skill_id: str
    profile_key: str
    prompt_variables: _containers.ScalarMap[str, str]
    def __init__(self, index: _Optional[int] = ..., phase: _Optional[str] = ..., phase_kind: _Optional[str] = ..., inputs: _Optional[_Union[OperatingModeSimulationInputs, _Mapping]] = ..., output: _Optional[_Union[OperatingModePhaseResult, _Mapping]] = ..., round: _Optional[_Union[OperatingModeRoundEnvelope, _Mapping]] = ..., transition: _Optional[_Union[OperatingModeSimulationTransition, _Mapping]] = ..., terminal: _Optional[bool] = ..., skill_id: _Optional[str] = ..., profile_key: _Optional[str] = ..., prompt_variables: _Optional[_Mapping[str, str]] = ...) -> None: ...

class OperatingModeSimulationPreset(_message.Message):
    __slots__ = ("id", "label", "description", "branch", "scenario")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    description: str
    branch: str
    scenario: str
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., description: _Optional[str] = ..., branch: _Optional[str] = ..., scenario: _Optional[str] = ...) -> None: ...

class OperatingModeSimulationResponse(_message.Message):
    __slots__ = ("mode", "label", "presets", "active_preset", "initiative", "trace")
    MODE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    PRESETS_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_PRESET_FIELD_NUMBER: _ClassVar[int]
    INITIATIVE_FIELD_NUMBER: _ClassVar[int]
    TRACE_FIELD_NUMBER: _ClassVar[int]
    mode: str
    label: str
    presets: _containers.RepeatedCompositeFieldContainer[OperatingModeSimulationPreset]
    active_preset: str
    initiative: OperatingModeInitiativeSnapshot
    trace: _containers.RepeatedCompositeFieldContainer[OperatingModeSimulationStep]
    def __init__(self, mode: _Optional[str] = ..., label: _Optional[str] = ..., presets: _Optional[_Iterable[_Union[OperatingModeSimulationPreset, _Mapping]]] = ..., active_preset: _Optional[str] = ..., initiative: _Optional[_Union[OperatingModeInitiativeSnapshot, _Mapping]] = ..., trace: _Optional[_Iterable[_Union[OperatingModeSimulationStep, _Mapping]]] = ...) -> None: ...

class OperatingModeRenderPromptResponse(_message.Message):
    __slots__ = ("mode", "preset", "step_index", "phase", "skill_id", "profile_key", "variables", "prompt", "degraded", "degraded_reason")
    class VariablesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    MODE_FIELD_NUMBER: _ClassVar[int]
    PRESET_FIELD_NUMBER: _ClassVar[int]
    STEP_INDEX_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    SKILL_ID_FIELD_NUMBER: _ClassVar[int]
    PROFILE_KEY_FIELD_NUMBER: _ClassVar[int]
    VARIABLES_FIELD_NUMBER: _ClassVar[int]
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASON_FIELD_NUMBER: _ClassVar[int]
    mode: str
    preset: str
    step_index: int
    phase: str
    skill_id: str
    profile_key: str
    variables: _containers.ScalarMap[str, str]
    prompt: str
    degraded: bool
    degraded_reason: str
    def __init__(self, mode: _Optional[str] = ..., preset: _Optional[str] = ..., step_index: _Optional[int] = ..., phase: _Optional[str] = ..., skill_id: _Optional[str] = ..., profile_key: _Optional[str] = ..., variables: _Optional[_Mapping[str, str]] = ..., prompt: _Optional[str] = ..., degraded: _Optional[bool] = ..., degraded_reason: _Optional[str] = ...) -> None: ...

class OperatingModeScaffoldResponse(_message.Message):
    __slots__ = ("mode", "dir", "created_files")
    MODE_FIELD_NUMBER: _ClassVar[int]
    DIR_FIELD_NUMBER: _ClassVar[int]
    CREATED_FILES_FIELD_NUMBER: _ClassVar[int]
    mode: str
    dir: str
    created_files: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, mode: _Optional[str] = ..., dir: _Optional[str] = ..., created_files: _Optional[_Iterable[str]] = ...) -> None: ...

class OperatingModeValidateResponse(_message.Message):
    __slots__ = ("mode", "ok", "errors", "phase_count", "example_runs", "summary", "uncovered_branches")
    MODE_FIELD_NUMBER: _ClassVar[int]
    OK_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    PHASE_COUNT_FIELD_NUMBER: _ClassVar[int]
    EXAMPLE_RUNS_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    UNCOVERED_BRANCHES_FIELD_NUMBER: _ClassVar[int]
    mode: str
    ok: bool
    errors: _containers.RepeatedScalarFieldContainer[str]
    phase_count: int
    example_runs: int
    summary: str
    uncovered_branches: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, mode: _Optional[str] = ..., ok: _Optional[bool] = ..., errors: _Optional[_Iterable[str]] = ..., phase_count: _Optional[int] = ..., example_runs: _Optional[int] = ..., summary: _Optional[str] = ..., uncovered_branches: _Optional[_Iterable[str]] = ...) -> None: ...
