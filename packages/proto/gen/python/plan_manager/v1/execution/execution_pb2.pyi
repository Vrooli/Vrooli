from plan_manager.v1.shared import model_pb2 as _model_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Execution(_message.Message):
    __slots__ = ("id", "plan_id", "run_id", "current_phase_id", "complete", "started_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    CURRENT_PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    plan_id: str
    run_id: str
    current_phase_id: str
    complete: bool
    started_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., plan_id: _Optional[str] = ..., run_id: _Optional[str] = ..., current_phase_id: _Optional[str] = ..., complete: _Optional[bool] = ..., started_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class PhaseContext(_message.Message):
    __slots__ = ("current_phase", "next_phase", "required_reading", "reminders", "last_validation", "staleness", "resume_phase_id", "completeness", "relevant_context")
    CURRENT_PHASE_FIELD_NUMBER: _ClassVar[int]
    NEXT_PHASE_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_READING_FIELD_NUMBER: _ClassVar[int]
    REMINDERS_FIELD_NUMBER: _ClassVar[int]
    LAST_VALIDATION_FIELD_NUMBER: _ClassVar[int]
    STALENESS_FIELD_NUMBER: _ClassVar[int]
    RESUME_PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    COMPLETENESS_FIELD_NUMBER: _ClassVar[int]
    RELEVANT_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    current_phase: _model_pb2.Phase
    next_phase: _model_pb2.Phase
    required_reading: _containers.RepeatedScalarFieldContainer[str]
    reminders: _containers.RepeatedScalarFieldContainer[str]
    last_validation: _model_pb2.ValidationResult
    staleness: _model_pb2.StalenessTier
    resume_phase_id: str
    completeness: _model_pb2.Completeness
    relevant_context: _containers.RepeatedCompositeFieldContainer[_model_pb2.RelevantContextItem]
    def __init__(self, current_phase: _Optional[_Union[_model_pb2.Phase, _Mapping]] = ..., next_phase: _Optional[_Union[_model_pb2.Phase, _Mapping]] = ..., required_reading: _Optional[_Iterable[str]] = ..., reminders: _Optional[_Iterable[str]] = ..., last_validation: _Optional[_Union[_model_pb2.ValidationResult, _Mapping]] = ..., staleness: _Optional[_Union[_model_pb2.StalenessTier, str]] = ..., resume_phase_id: _Optional[str] = ..., completeness: _Optional[_Union[_model_pb2.Completeness, str]] = ..., relevant_context: _Optional[_Iterable[_Union[_model_pb2.RelevantContextItem, _Mapping]]] = ...) -> None: ...

class CompletionNudge(_message.Message):
    __slots__ = ("kind", "message", "satisfied")
    KIND_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SATISFIED_FIELD_NUMBER: _ClassVar[int]
    kind: str
    message: str
    satisfied: bool
    def __init__(self, kind: _Optional[str] = ..., message: _Optional[str] = ..., satisfied: _Optional[bool] = ...) -> None: ...

class StartRequest(_message.Message):
    __slots__ = ("plan_id", "run_id")
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    run_id: str
    def __init__(self, plan_id: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class StartResponse(_message.Message):
    __slots__ = ("execution", "step", "context")
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    execution: Execution
    step: _model_pb2.GuidedStep
    context: PhaseContext
    def __init__(self, execution: _Optional[_Union[Execution, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ..., context: _Optional[_Union[PhaseContext, _Mapping]] = ...) -> None: ...

class GetStatusRequest(_message.Message):
    __slots__ = ("execution_id",)
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    def __init__(self, execution_id: _Optional[str] = ...) -> None: ...

class GetStatusResponse(_message.Message):
    __slots__ = ("execution", "context", "step")
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    execution: Execution
    context: PhaseContext
    step: _model_pb2.GuidedStep
    def __init__(self, execution: _Optional[_Union[Execution, _Mapping]] = ..., context: _Optional[_Union[PhaseContext, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class GetNextRequest(_message.Message):
    __slots__ = ("execution_id",)
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    def __init__(self, execution_id: _Optional[str] = ...) -> None: ...

class GetNextResponse(_message.Message):
    __slots__ = ("context", "complete", "step")
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    COMPLETE_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    context: PhaseContext
    complete: bool
    step: _model_pb2.GuidedStep
    def __init__(self, context: _Optional[_Union[PhaseContext, _Mapping]] = ..., complete: _Optional[bool] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class TransitionPhaseRequest(_message.Message):
    __slots__ = ("execution_id", "phase_id", "to_status")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    TO_STATUS_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    phase_id: str
    to_status: _model_pb2.PhaseStatus
    def __init__(self, execution_id: _Optional[str] = ..., phase_id: _Optional[str] = ..., to_status: _Optional[_Union[_model_pb2.PhaseStatus, str]] = ...) -> None: ...

class TransitionPhaseResponse(_message.Message):
    __slots__ = ("execution", "plan", "step")
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    PLAN_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    execution: Execution
    plan: _model_pb2.Plan
    step: _model_pb2.GuidedStep
    def __init__(self, execution: _Optional[_Union[Execution, _Mapping]] = ..., plan: _Optional[_Union[_model_pb2.Plan, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class RecordDecisionRequest(_message.Message):
    __slots__ = ("execution_id", "phase_id", "summary", "detail")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    phase_id: str
    summary: str
    detail: str
    def __init__(self, execution_id: _Optional[str] = ..., phase_id: _Optional[str] = ..., summary: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class RecordDecisionResponse(_message.Message):
    __slots__ = ("decision", "step")
    DECISION_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    decision: _model_pb2.Decision
    step: _model_pb2.GuidedStep
    def __init__(self, decision: _Optional[_Union[_model_pb2.Decision, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class RecordFindingRequest(_message.Message):
    __slots__ = ("execution_id", "phase_id", "title", "detail")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    phase_id: str
    title: str
    detail: str
    def __init__(self, execution_id: _Optional[str] = ..., phase_id: _Optional[str] = ..., title: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class RecordFindingResponse(_message.Message):
    __slots__ = ("finding", "step")
    FINDING_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    finding: _model_pb2.Finding
    step: _model_pb2.GuidedStep
    def __init__(self, finding: _Optional[_Union[_model_pb2.Finding, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class CompleteRequest(_message.Message):
    __slots__ = ("execution_id", "tokens", "iterations")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    TOKENS_FIELD_NUMBER: _ClassVar[int]
    ITERATIONS_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    tokens: int
    iterations: int
    def __init__(self, execution_id: _Optional[str] = ..., tokens: _Optional[int] = ..., iterations: _Optional[int] = ...) -> None: ...

class CompleteResponse(_message.Message):
    __slots__ = ("handoff", "nudges", "step")
    HANDOFF_FIELD_NUMBER: _ClassVar[int]
    NUDGES_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    handoff: _model_pb2.Handoff
    nudges: _containers.RepeatedCompositeFieldContainer[CompletionNudge]
    step: _model_pb2.GuidedStep
    def __init__(self, handoff: _Optional[_Union[_model_pb2.Handoff, _Mapping]] = ..., nudges: _Optional[_Iterable[_Union[CompletionNudge, _Mapping]]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class GetHandoffRequest(_message.Message):
    __slots__ = ("execution_id",)
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    def __init__(self, execution_id: _Optional[str] = ...) -> None: ...

class GetHandoffResponse(_message.Message):
    __slots__ = ("handoff", "step")
    HANDOFF_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    handoff: _model_pb2.Handoff
    step: _model_pb2.GuidedStep
    def __init__(self, handoff: _Optional[_Union[_model_pb2.Handoff, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class ListCandidateFindingsRequest(_message.Message):
    __slots__ = ("execution_id",)
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    def __init__(self, execution_id: _Optional[str] = ...) -> None: ...

class ListCandidateFindingsResponse(_message.Message):
    __slots__ = ("findings", "step")
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    findings: _containers.RepeatedCompositeFieldContainer[_model_pb2.Finding]
    step: _model_pb2.GuidedStep
    def __init__(self, findings: _Optional[_Iterable[_Union[_model_pb2.Finding, _Mapping]]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class TriageFindingRequest(_message.Message):
    __slots__ = ("finding_id", "triage")
    FINDING_ID_FIELD_NUMBER: _ClassVar[int]
    TRIAGE_FIELD_NUMBER: _ClassVar[int]
    finding_id: str
    triage: _model_pb2.FindingTriage
    def __init__(self, finding_id: _Optional[str] = ..., triage: _Optional[_Union[_model_pb2.FindingTriage, str]] = ...) -> None: ...

class TriageFindingResponse(_message.Message):
    __slots__ = ("finding", "step")
    FINDING_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    finding: _model_pb2.Finding
    step: _model_pb2.GuidedStep
    def __init__(self, finding: _Optional[_Union[_model_pb2.Finding, _Mapping]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...

class GetVelocityRequest(_message.Message):
    __slots__ = ("plan_id",)
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    plan_id: str
    def __init__(self, plan_id: _Optional[str] = ...) -> None: ...

class GetVelocityResponse(_message.Message):
    __slots__ = ("points", "step")
    POINTS_FIELD_NUMBER: _ClassVar[int]
    STEP_FIELD_NUMBER: _ClassVar[int]
    points: _containers.RepeatedCompositeFieldContainer[_model_pb2.VelocityPoint]
    step: _model_pb2.GuidedStep
    def __init__(self, points: _Optional[_Iterable[_Union[_model_pb2.VelocityPoint, _Mapping]]] = ..., step: _Optional[_Union[_model_pb2.GuidedStep, _Mapping]] = ...) -> None: ...
