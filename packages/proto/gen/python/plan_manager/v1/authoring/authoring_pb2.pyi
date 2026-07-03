from plan_manager.v1.shared import model_pb2 as _model_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Section(_message.Message):
    __slots__ = ("key", "label", "content", "mandatory", "filled", "autofilled")
    KEY_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    MANDATORY_FIELD_NUMBER: _ClassVar[int]
    FILLED_FIELD_NUMBER: _ClassVar[int]
    AUTOFILLED_FIELD_NUMBER: _ClassVar[int]
    key: str
    label: str
    content: str
    mandatory: bool
    filled: bool
    autofilled: bool
    def __init__(self, key: _Optional[str] = ..., label: _Optional[str] = ..., content: _Optional[str] = ..., mandatory: _Optional[bool] = ..., filled: _Optional[bool] = ..., autofilled: _Optional[bool] = ...) -> None: ...

class AuthoringSession(_message.Message):
    __slots__ = ("id", "title", "plan_slug", "sections", "current_section_key", "finalized", "plan_id", "phase_drafts", "current_phase_id", "relevant_context")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    PLAN_SLUG_FIELD_NUMBER: _ClassVar[int]
    SECTIONS_FIELD_NUMBER: _ClassVar[int]
    CURRENT_SECTION_KEY_FIELD_NUMBER: _ClassVar[int]
    FINALIZED_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_DRAFTS_FIELD_NUMBER: _ClassVar[int]
    CURRENT_PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    RELEVANT_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    plan_slug: str
    sections: _containers.RepeatedCompositeFieldContainer[Section]
    current_section_key: str
    finalized: bool
    plan_id: str
    phase_drafts: _containers.RepeatedCompositeFieldContainer[PhaseDraft]
    current_phase_id: str
    relevant_context: _containers.RepeatedCompositeFieldContainer[_model_pb2.RelevantContextItem]
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., plan_slug: _Optional[str] = ..., sections: _Optional[_Iterable[_Union[Section, _Mapping]]] = ..., current_section_key: _Optional[str] = ..., finalized: _Optional[bool] = ..., plan_id: _Optional[str] = ..., phase_drafts: _Optional[_Iterable[_Union[PhaseDraft, _Mapping]]] = ..., current_phase_id: _Optional[str] = ..., relevant_context: _Optional[_Iterable[_Union[_model_pb2.RelevantContextItem, _Mapping]]] = ...) -> None: ...

class StructureViolation(_message.Message):
    __slots__ = ("section_key", "message")
    SECTION_KEY_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    section_key: str
    message: str
    def __init__(self, section_key: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class AutofillResult(_message.Message):
    __slots__ = ("source", "section_key", "filled", "degraded", "detail")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SECTION_KEY_FIELD_NUMBER: _ClassVar[int]
    FILLED_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    source: str
    section_key: str
    filled: bool
    degraded: bool
    detail: str
    def __init__(self, source: _Optional[str] = ..., section_key: _Optional[str] = ..., filled: _Optional[bool] = ..., degraded: _Optional[bool] = ..., detail: _Optional[str] = ...) -> None: ...

class PhaseDraft(_message.Message):
    __slots__ = ("id", "order", "title", "intent", "references", "required_reading", "reminders", "acceptance", "no_code_refs_reason", "relevant_context", "affected_areas", "steps", "expected_outputs", "validation", "risks_hazards", "handoff_notes")
    ID_FIELD_NUMBER: _ClassVar[int]
    ORDER_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    INTENT_FIELD_NUMBER: _ClassVar[int]
    REFERENCES_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_READING_FIELD_NUMBER: _ClassVar[int]
    REMINDERS_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_FIELD_NUMBER: _ClassVar[int]
    NO_CODE_REFS_REASON_FIELD_NUMBER: _ClassVar[int]
    RELEVANT_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    AFFECTED_AREAS_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_OUTPUTS_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_FIELD_NUMBER: _ClassVar[int]
    RISKS_HAZARDS_FIELD_NUMBER: _ClassVar[int]
    HANDOFF_NOTES_FIELD_NUMBER: _ClassVar[int]
    id: str
    order: int
    title: str
    intent: str
    references: _containers.RepeatedCompositeFieldContainer[_model_pb2.Reference]
    required_reading: _containers.RepeatedScalarFieldContainer[str]
    reminders: _containers.RepeatedScalarFieldContainer[str]
    acceptance: str
    no_code_refs_reason: str
    relevant_context: _containers.RepeatedCompositeFieldContainer[_model_pb2.RelevantContextItem]
    affected_areas: _containers.RepeatedScalarFieldContainer[str]
    steps: _containers.RepeatedScalarFieldContainer[str]
    expected_outputs: _containers.RepeatedScalarFieldContainer[str]
    validation: str
    risks_hazards: _containers.RepeatedScalarFieldContainer[str]
    handoff_notes: str
    def __init__(self, id: _Optional[str] = ..., order: _Optional[int] = ..., title: _Optional[str] = ..., intent: _Optional[str] = ..., references: _Optional[_Iterable[_Union[_model_pb2.Reference, _Mapping]]] = ..., required_reading: _Optional[_Iterable[str]] = ..., reminders: _Optional[_Iterable[str]] = ..., acceptance: _Optional[str] = ..., no_code_refs_reason: _Optional[str] = ..., relevant_context: _Optional[_Iterable[_Union[_model_pb2.RelevantContextItem, _Mapping]]] = ..., affected_areas: _Optional[_Iterable[str]] = ..., steps: _Optional[_Iterable[str]] = ..., expected_outputs: _Optional[_Iterable[str]] = ..., validation: _Optional[str] = ..., risks_hazards: _Optional[_Iterable[str]] = ..., handoff_notes: _Optional[str] = ...) -> None: ...

class AuthoringProgress(_message.Message):
    __slots__ = ("session_id", "current_section_key", "current_phase_id", "mandatory_sections_total", "mandatory_sections_filled", "phases_total", "phases_complete", "remaining_required_inputs", "ready_to_finalize")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    CURRENT_SECTION_KEY_FIELD_NUMBER: _ClassVar[int]
    CURRENT_PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    MANDATORY_SECTIONS_TOTAL_FIELD_NUMBER: _ClassVar[int]
    MANDATORY_SECTIONS_FILLED_FIELD_NUMBER: _ClassVar[int]
    PHASES_TOTAL_FIELD_NUMBER: _ClassVar[int]
    PHASES_COMPLETE_FIELD_NUMBER: _ClassVar[int]
    REMAINING_REQUIRED_INPUTS_FIELD_NUMBER: _ClassVar[int]
    READY_TO_FINALIZE_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    current_section_key: str
    current_phase_id: str
    mandatory_sections_total: int
    mandatory_sections_filled: int
    phases_total: int
    phases_complete: int
    remaining_required_inputs: _containers.RepeatedScalarFieldContainer[str]
    ready_to_finalize: bool
    def __init__(self, session_id: _Optional[str] = ..., current_section_key: _Optional[str] = ..., current_phase_id: _Optional[str] = ..., mandatory_sections_total: _Optional[int] = ..., mandatory_sections_filled: _Optional[int] = ..., phases_total: _Optional[int] = ..., phases_complete: _Optional[int] = ..., remaining_required_inputs: _Optional[_Iterable[str]] = ..., ready_to_finalize: _Optional[bool] = ...) -> None: ...

class AuthoringMutationSummary(_message.Message):
    __slots__ = ("object_kind", "object_id", "field", "summary")
    OBJECT_KIND_FIELD_NUMBER: _ClassVar[int]
    OBJECT_ID_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    object_kind: str
    object_id: str
    field: str
    summary: str
    def __init__(self, object_kind: _Optional[str] = ..., object_id: _Optional[str] = ..., field: _Optional[str] = ..., summary: _Optional[str] = ...) -> None: ...

class StartSessionRequest(_message.Message):
    __slots__ = ("title", "slug", "template_id")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_ID_FIELD_NUMBER: _ClassVar[int]
    title: str
    slug: str
    template_id: str
    def __init__(self, title: _Optional[str] = ..., slug: _Optional[str] = ..., template_id: _Optional[str] = ...) -> None: ...

class StartSessionResponse(_message.Message):
    __slots__ = ("session", "step")
    SESSION_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    session: AuthoringSession
    step: _model_pb2.GuidedStep
    def __init__(self, session: _Optional[_Union[AuthoringSession, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class GetSessionRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class GetSessionResponse(_message.Message):
    __slots__ = ("session", "step")
    SESSION_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    session: AuthoringSession
    step: _model_pb2.GuidedStep
    def __init__(self, session: _Optional[_Union[AuthoringSession, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class GetSectionRequest(_message.Message):
    __slots__ = ("session_id", "section_key")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SECTION_KEY_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    section_key: str
    def __init__(self, session_id: _Optional[str] = ..., section_key: _Optional[str] = ...) -> None: ...

class GetSectionResponse(_message.Message):
    __slots__ = ("section", "step")
    SECTION_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    section: Section
    step: _model_pb2.GuidedStep
    def __init__(self, section: _Optional[_Union[Section, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class SubmitSectionRequest(_message.Message):
    __slots__ = ("session_id", "section_key", "content")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SECTION_KEY_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    section_key: str
    content: str
    def __init__(self, session_id: _Optional[str] = ..., section_key: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class SubmitSectionResponse(_message.Message):
    __slots__ = ("summary", "progress", "violations", "step")
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    summary: AuthoringMutationSummary
    progress: AuthoringProgress
    violations: _containers.RepeatedCompositeFieldContainer[StructureViolation]
    step: _model_pb2.GuidedStep
    def __init__(self, summary: _Optional[_Union[AuthoringMutationSummary, _Mapping]] = ..., progress: _Optional[_Union[AuthoringProgress, _Mapping]] = ..., violations: _Optional[_Iterable[_Union[StructureViolation, _Mapping]]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class PhaseFieldRef(_message.Message):
    __slots__ = ("phase_ref", "field")
    PHASE_REF_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    phase_ref: str
    field: str
    def __init__(self, phase_ref: _Optional[str] = ..., field: _Optional[str] = ...) -> None: ...

class FieldWrite(_message.Message):
    __slots__ = ("section_key", "phase", "content")
    SECTION_KEY_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    section_key: str
    phase: PhaseFieldRef
    content: str
    def __init__(self, section_key: _Optional[str] = ..., phase: _Optional[_Union[PhaseFieldRef, _Mapping]] = ..., content: _Optional[str] = ...) -> None: ...

class SubmitFieldsRequest(_message.Message):
    __slots__ = ("session_id", "items")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    items: _containers.RepeatedCompositeFieldContainer[FieldWrite]
    def __init__(self, session_id: _Optional[str] = ..., items: _Optional[_Iterable[_Union[FieldWrite, _Mapping]]] = ...) -> None: ...

class FieldWriteResult(_message.Message):
    __slots__ = ("index", "accepted", "summary", "violations")
    INDEX_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    index: int
    accepted: bool
    summary: str
    violations: _containers.RepeatedCompositeFieldContainer[StructureViolation]
    def __init__(self, index: _Optional[int] = ..., accepted: _Optional[bool] = ..., summary: _Optional[str] = ..., violations: _Optional[_Iterable[_Union[StructureViolation, _Mapping]]] = ...) -> None: ...

class SubmitFieldsResponse(_message.Message):
    __slots__ = ("results", "progress", "step")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[FieldWriteResult]
    progress: AuthoringProgress
    step: _model_pb2.GuidedStep
    def __init__(self, results: _Optional[_Iterable[_Union[FieldWriteResult, _Mapping]]] = ..., progress: _Optional[_Union[AuthoringProgress, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class NextRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class NextResponse(_message.Message):
    __slots__ = ("section", "complete", "step")
    SECTION_FIELD_NUMBER: _ClassVar[int]
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    section: Section
    complete: bool
    step: _model_pb2.GuidedStep
    def __init__(self, section: _Optional[_Union[Section, _Mapping]] = ..., complete: _Optional[bool] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class ContinueAuthoringRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class ContinueAuthoringResponse(_message.Message):
    __slots__ = ("section", "phase", "progress", "ready_to_finalize", "violations", "step")
    SECTION_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    READY_TO_FINALIZE_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    section: Section
    phase: PhaseDraft
    progress: AuthoringProgress
    ready_to_finalize: bool
    violations: _containers.RepeatedCompositeFieldContainer[StructureViolation]
    step: _model_pb2.GuidedStep
    def __init__(self, section: _Optional[_Union[Section, _Mapping]] = ..., phase: _Optional[_Union[PhaseDraft, _Mapping]] = ..., progress: _Optional[_Union[AuthoringProgress, _Mapping]] = ..., ready_to_finalize: _Optional[bool] = ..., violations: _Optional[_Iterable[_Union[StructureViolation, _Mapping]]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class ValidateStructureRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class ValidateStructureResponse(_message.Message):
    __slots__ = ("valid", "violations", "step")
    VALID_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    valid: bool
    violations: _containers.RepeatedCompositeFieldContainer[StructureViolation]
    step: _model_pb2.GuidedStep
    def __init__(self, valid: _Optional[bool] = ..., violations: _Optional[_Iterable[_Union[StructureViolation, _Mapping]]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class AutofillRequest(_message.Message):
    __slots__ = ("session_id", "sources")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCES_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    sources: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, session_id: _Optional[str] = ..., sources: _Optional[_Iterable[str]] = ...) -> None: ...

class AutofillResponse(_message.Message):
    __slots__ = ("results", "progress", "step")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[AutofillResult]
    progress: AuthoringProgress
    step: _model_pb2.GuidedStep
    def __init__(self, results: _Optional[_Iterable[_Union[AutofillResult, _Mapping]]] = ..., progress: _Optional[_Union[AuthoringProgress, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class SubmitRelevantContextItemRequest(_message.Message):
    __slots__ = ("session_id", "phase_id", "item")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    phase_id: str
    item: _model_pb2.RelevantContextItem
    def __init__(self, session_id: _Optional[str] = ..., phase_id: _Optional[str] = ..., item: _Optional[_Union[_model_pb2.RelevantContextItem, _Mapping]] = ...) -> None: ...

class SubmitRelevantContextItemResponse(_message.Message):
    __slots__ = ("item", "summary", "progress", "violations", "step", "accepted")
    ITEM_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_FIELD_NUMBER: _ClassVar[int]
    item: _model_pb2.RelevantContextItem
    summary: AuthoringMutationSummary
    progress: AuthoringProgress
    violations: _containers.RepeatedCompositeFieldContainer[StructureViolation]
    step: _model_pb2.GuidedStep
    accepted: bool
    def __init__(self, item: _Optional[_Union[_model_pb2.RelevantContextItem, _Mapping]] = ..., summary: _Optional[_Union[AuthoringMutationSummary, _Mapping]] = ..., progress: _Optional[_Union[AuthoringProgress, _Mapping]] = ..., violations: _Optional[_Iterable[_Union[StructureViolation, _Mapping]]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ..., accepted: _Optional[bool] = ...) -> None: ...

class ListRelevantContextRequest(_message.Message):
    __slots__ = ("session_id", "phase_id")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    phase_id: str
    def __init__(self, session_id: _Optional[str] = ..., phase_id: _Optional[str] = ...) -> None: ...

class ListRelevantContextResponse(_message.Message):
    __slots__ = ("items", "step")
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[_model_pb2.RelevantContextItem]
    step: _model_pb2.GuidedStep
    def __init__(self, items: _Optional[_Iterable[_Union[_model_pb2.RelevantContextItem, _Mapping]]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class UpdateRelevantContextItemRequest(_message.Message):
    __slots__ = ("session_id", "phase_id", "item_id", "item")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    phase_id: str
    item_id: str
    item: _model_pb2.RelevantContextItem
    def __init__(self, session_id: _Optional[str] = ..., phase_id: _Optional[str] = ..., item_id: _Optional[str] = ..., item: _Optional[_Union[_model_pb2.RelevantContextItem, _Mapping]] = ...) -> None: ...

class UpdateRelevantContextItemResponse(_message.Message):
    __slots__ = ("item", "summary", "progress", "violations", "step")
    ITEM_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    item: _model_pb2.RelevantContextItem
    summary: AuthoringMutationSummary
    progress: AuthoringProgress
    violations: _containers.RepeatedCompositeFieldContainer[StructureViolation]
    step: _model_pb2.GuidedStep
    def __init__(self, item: _Optional[_Union[_model_pb2.RelevantContextItem, _Mapping]] = ..., summary: _Optional[_Union[AuthoringMutationSummary, _Mapping]] = ..., progress: _Optional[_Union[AuthoringProgress, _Mapping]] = ..., violations: _Optional[_Iterable[_Union[StructureViolation, _Mapping]]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class RemoveRelevantContextItemRequest(_message.Message):
    __slots__ = ("session_id", "phase_id", "item_id")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    phase_id: str
    item_id: str
    def __init__(self, session_id: _Optional[str] = ..., phase_id: _Optional[str] = ..., item_id: _Optional[str] = ...) -> None: ...

class RemoveRelevantContextItemResponse(_message.Message):
    __slots__ = ("summary", "progress", "violations", "step")
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    summary: AuthoringMutationSummary
    progress: AuthoringProgress
    violations: _containers.RepeatedCompositeFieldContainer[StructureViolation]
    step: _model_pb2.GuidedStep
    def __init__(self, summary: _Optional[_Union[AuthoringMutationSummary, _Mapping]] = ..., progress: _Optional[_Union[AuthoringProgress, _Mapping]] = ..., violations: _Optional[_Iterable[_Union[StructureViolation, _Mapping]]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class DiscoverSkillPackRequest(_message.Message):
    __slots__ = ("session_id", "concepts", "complexity")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    CONCEPTS_FIELD_NUMBER: _ClassVar[int]
    COMPLEXITY_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    concepts: _containers.RepeatedScalarFieldContainer[str]
    complexity: str
    def __init__(self, session_id: _Optional[str] = ..., concepts: _Optional[_Iterable[str]] = ..., complexity: _Optional[str] = ...) -> None: ...

class DiscoverSkillPackResponse(_message.Message):
    __slots__ = ("added_items", "kept_items", "read_command", "recommended_read_command", "budget_status", "results_summary", "progress", "step", "violations", "degraded", "degraded_reason")
    ADDED_ITEMS_FIELD_NUMBER: _ClassVar[int]
    KEPT_ITEMS_FIELD_NUMBER: _ClassVar[int]
    READ_COMMAND_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_READ_COMMAND_FIELD_NUMBER: _ClassVar[int]
    BUDGET_STATUS_FIELD_NUMBER: _ClassVar[int]
    RESULTS_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_REASON_FIELD_NUMBER: _ClassVar[int]
    added_items: _containers.RepeatedCompositeFieldContainer[_model_pb2.RelevantContextItem]
    kept_items: _containers.RepeatedCompositeFieldContainer[_model_pb2.RelevantContextItem]
    read_command: str
    recommended_read_command: str
    budget_status: str
    results_summary: str
    progress: AuthoringProgress
    step: _model_pb2.GuidedStep
    violations: _containers.RepeatedCompositeFieldContainer[StructureViolation]
    degraded: bool
    degraded_reason: str
    def __init__(self, added_items: _Optional[_Iterable[_Union[_model_pb2.RelevantContextItem, _Mapping]]] = ..., kept_items: _Optional[_Iterable[_Union[_model_pb2.RelevantContextItem, _Mapping]]] = ..., read_command: _Optional[str] = ..., recommended_read_command: _Optional[str] = ..., budget_status: _Optional[str] = ..., results_summary: _Optional[str] = ..., progress: _Optional[_Union[AuthoringProgress, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ..., violations: _Optional[_Iterable[_Union[StructureViolation, _Mapping]]] = ..., degraded: _Optional[bool] = ..., degraded_reason: _Optional[str] = ...) -> None: ...

class AddPhaseRequest(_message.Message):
    __slots__ = ("session_id", "title", "intent")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    INTENT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    title: str
    intent: str
    def __init__(self, session_id: _Optional[str] = ..., title: _Optional[str] = ..., intent: _Optional[str] = ...) -> None: ...

class AddPhaseResponse(_message.Message):
    __slots__ = ("phase", "summary", "progress", "violations", "step")
    PHASE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    phase: PhaseDraft
    summary: AuthoringMutationSummary
    progress: AuthoringProgress
    violations: _containers.RepeatedCompositeFieldContainer[StructureViolation]
    step: _model_pb2.GuidedStep
    def __init__(self, phase: _Optional[_Union[PhaseDraft, _Mapping]] = ..., summary: _Optional[_Union[AuthoringMutationSummary, _Mapping]] = ..., progress: _Optional[_Union[AuthoringProgress, _Mapping]] = ..., violations: _Optional[_Iterable[_Union[StructureViolation, _Mapping]]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class MovePhaseRequest(_message.Message):
    __slots__ = ("session_id", "phase_id", "before_phase_id", "after_phase_id")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    BEFORE_PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    AFTER_PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    phase_id: str
    before_phase_id: str
    after_phase_id: str
    def __init__(self, session_id: _Optional[str] = ..., phase_id: _Optional[str] = ..., before_phase_id: _Optional[str] = ..., after_phase_id: _Optional[str] = ...) -> None: ...

class MovePhaseResponse(_message.Message):
    __slots__ = ("phase", "summary", "progress", "violations", "step")
    PHASE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    phase: PhaseDraft
    summary: AuthoringMutationSummary
    progress: AuthoringProgress
    violations: _containers.RepeatedCompositeFieldContainer[StructureViolation]
    step: _model_pb2.GuidedStep
    def __init__(self, phase: _Optional[_Union[PhaseDraft, _Mapping]] = ..., summary: _Optional[_Union[AuthoringMutationSummary, _Mapping]] = ..., progress: _Optional[_Union[AuthoringProgress, _Mapping]] = ..., violations: _Optional[_Iterable[_Union[StructureViolation, _Mapping]]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class GetPhaseRequest(_message.Message):
    __slots__ = ("session_id", "phase_id")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    phase_id: str
    def __init__(self, session_id: _Optional[str] = ..., phase_id: _Optional[str] = ...) -> None: ...

class GetPhaseResponse(_message.Message):
    __slots__ = ("phase", "step")
    PHASE_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    phase: PhaseDraft
    step: _model_pb2.GuidedStep
    def __init__(self, phase: _Optional[_Union[PhaseDraft, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class SubmitPhaseFieldRequest(_message.Message):
    __slots__ = ("session_id", "phase_id", "field", "content")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    phase_id: str
    field: str
    content: str
    def __init__(self, session_id: _Optional[str] = ..., phase_id: _Optional[str] = ..., field: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class SubmitPhaseFieldResponse(_message.Message):
    __slots__ = ("phase", "summary", "progress", "violations", "step")
    PHASE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    PROGRESS_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    phase: PhaseDraft
    summary: AuthoringMutationSummary
    progress: AuthoringProgress
    violations: _containers.RepeatedCompositeFieldContainer[StructureViolation]
    step: _model_pb2.GuidedStep
    def __init__(self, phase: _Optional[_Union[PhaseDraft, _Mapping]] = ..., summary: _Optional[_Union[AuthoringMutationSummary, _Mapping]] = ..., progress: _Optional[_Union[AuthoringProgress, _Mapping]] = ..., violations: _Optional[_Iterable[_Union[StructureViolation, _Mapping]]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class NextPhaseRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class NextPhaseResponse(_message.Message):
    __slots__ = ("phase", "complete", "step")
    PHASE_FIELD_NUMBER: _ClassVar[int]
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    phase: PhaseDraft
    complete: bool
    step: _model_pb2.GuidedStep
    def __init__(self, phase: _Optional[_Union[PhaseDraft, _Mapping]] = ..., complete: _Optional[bool] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class PreviewPlanRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class PreviewPlanResponse(_message.Message):
    __slots__ = ("markdown", "step")
    MARKDOWN_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    markdown: str
    step: _model_pb2.GuidedStep
    def __init__(self, markdown: _Optional[str] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class FinalizeRequest(_message.Message):
    __slots__ = ("session_id", "workspace_root")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_ROOT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    workspace_root: str
    def __init__(self, session_id: _Optional[str] = ..., workspace_root: _Optional[str] = ...) -> None: ...

class FinalizeResponse(_message.Message):
    __slots__ = ("plan", "step", "store_path", "mirror", "already_finalized", "finalized_at", "workspace_root")
    PLAN_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    STORE_PATH_FIELD_NUMBER: _ClassVar[int]
    MIRROR_FIELD_NUMBER: _ClassVar[int]
    ALREADY_FINALIZED_FIELD_NUMBER: _ClassVar[int]
    FINALIZED_AT_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_ROOT_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    step: _model_pb2.GuidedStep
    store_path: str
    mirror: _model_pb2.RenderedPlanMirror
    already_finalized: bool
    finalized_at: str
    workspace_root: str
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ..., store_path: _Optional[str] = ..., mirror: _Optional[_Union[_model_pb2.RenderedPlanMirror, _Mapping]] = ..., already_finalized: _Optional[bool] = ..., finalized_at: _Optional[str] = ..., workspace_root: _Optional[str] = ...) -> None: ...
