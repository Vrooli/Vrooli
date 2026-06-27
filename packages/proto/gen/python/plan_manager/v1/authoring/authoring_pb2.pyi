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
    __slots__ = ("id", "title", "plan_slug", "sections", "current_section_key", "finalized", "plan_id", "phase_drafts", "current_phase_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    PLAN_SLUG_FIELD_NUMBER: _ClassVar[int]
    SECTIONS_FIELD_NUMBER: _ClassVar[int]
    CURRENT_SECTION_KEY_FIELD_NUMBER: _ClassVar[int]
    FINALIZED_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_DRAFTS_FIELD_NUMBER: _ClassVar[int]
    CURRENT_PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    plan_slug: str
    sections: _containers.RepeatedCompositeFieldContainer[Section]
    current_section_key: str
    finalized: bool
    plan_id: str
    phase_drafts: _containers.RepeatedCompositeFieldContainer[PhaseDraft]
    current_phase_id: str
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., plan_slug: _Optional[str] = ..., sections: _Optional[_Iterable[_Union[Section, _Mapping]]] = ..., current_section_key: _Optional[str] = ..., finalized: _Optional[bool] = ..., plan_id: _Optional[str] = ..., phase_drafts: _Optional[_Iterable[_Union[PhaseDraft, _Mapping]]] = ..., current_phase_id: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("id", "order", "title", "intent", "references", "required_reading", "reminders", "acceptance", "no_code_refs_reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    ORDER_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    INTENT_FIELD_NUMBER: _ClassVar[int]
    REFERENCES_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_READING_FIELD_NUMBER: _ClassVar[int]
    REMINDERS_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_FIELD_NUMBER: _ClassVar[int]
    NO_CODE_REFS_REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    order: int
    title: str
    intent: str
    references: _containers.RepeatedCompositeFieldContainer[_model_pb2.Reference]
    required_reading: _containers.RepeatedScalarFieldContainer[str]
    reminders: _containers.RepeatedScalarFieldContainer[str]
    acceptance: str
    no_code_refs_reason: str
    def __init__(self, id: _Optional[str] = ..., order: _Optional[int] = ..., title: _Optional[str] = ..., intent: _Optional[str] = ..., references: _Optional[_Iterable[_Union[_model_pb2.Reference, _Mapping]]] = ..., required_reading: _Optional[_Iterable[str]] = ..., reminders: _Optional[_Iterable[str]] = ..., acceptance: _Optional[str] = ..., no_code_refs_reason: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("session", "violations", "step")
    SESSION_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    session: AuthoringSession
    violations: _containers.RepeatedCompositeFieldContainer[StructureViolation]
    step: _model_pb2.GuidedStep
    def __init__(self, session: _Optional[_Union[AuthoringSession, _Mapping]] = ..., violations: _Optional[_Iterable[_Union[StructureViolation, _Mapping]]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

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
    __slots__ = ("session", "results", "step")
    SESSION_FIELD_NUMBER: _ClassVar[int]
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    session: AuthoringSession
    results: _containers.RepeatedCompositeFieldContainer[AutofillResult]
    step: _model_pb2.GuidedStep
    def __init__(self, session: _Optional[_Union[AuthoringSession, _Mapping]] = ..., results: _Optional[_Iterable[_Union[AutofillResult, _Mapping]]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

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
    __slots__ = ("session", "phase", "violations", "step")
    SESSION_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    session: AuthoringSession
    phase: PhaseDraft
    violations: _containers.RepeatedCompositeFieldContainer[StructureViolation]
    step: _model_pb2.GuidedStep
    def __init__(self, session: _Optional[_Union[AuthoringSession, _Mapping]] = ..., phase: _Optional[_Union[PhaseDraft, _Mapping]] = ..., violations: _Optional[_Iterable[_Union[StructureViolation, _Mapping]]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

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
    __slots__ = ("session", "violations", "step")
    SESSION_FIELD_NUMBER: _ClassVar[int]
    VIOLATIONS_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    session: AuthoringSession
    violations: _containers.RepeatedCompositeFieldContainer[StructureViolation]
    step: _model_pb2.GuidedStep
    def __init__(self, session: _Optional[_Union[AuthoringSession, _Mapping]] = ..., violations: _Optional[_Iterable[_Union[StructureViolation, _Mapping]]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

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

class FinalizeRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class FinalizeResponse(_message.Message):
    __slots__ = ("plan", "step")
    PLAN_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    plan: _model_pb2.Plan
    step: _model_pb2.GuidedStep
    def __init__(self, plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...
