from swarm_manager.v1.domain import transition_pb2 as _transition_pb2
from swarm_manager.v1.shared import plan_ref_pb2 as _plan_ref_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DevelopmentOutcome(_message.Message):
    __slots__ = ("id", "criterion", "evidence_source")
    ID_FIELD_NUMBER: _ClassVar[int]
    CRITERION_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_SOURCE_FIELD_NUMBER: _ClassVar[int]
    id: str
    criterion: str
    evidence_source: str
    def __init__(self, id: _Optional[str] = ..., criterion: _Optional[str] = ..., evidence_source: _Optional[str] = ...) -> None: ...

class PreviewDevelopmentRequest(_message.Message):
    __slots__ = ("scenario", "work_item", "objective", "artifact_paths", "outcomes", "acceptance_allow", "acceptance_deny", "allowed_effects", "max_tokens", "max_wall_seconds", "guidance", "budget_policy", "plan_ref", "execution_strategy")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    WORK_ITEM_FIELD_NUMBER: _ClassVar[int]
    OBJECTIVE_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_PATHS_FIELD_NUMBER: _ClassVar[int]
    OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_ALLOW_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_DENY_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_EFFECTS_FIELD_NUMBER: _ClassVar[int]
    MAX_TOKENS_FIELD_NUMBER: _ClassVar[int]
    MAX_WALL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    GUIDANCE_FIELD_NUMBER: _ClassVar[int]
    BUDGET_POLICY_FIELD_NUMBER: _ClassVar[int]
    PLAN_REF_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_STRATEGY_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    work_item: str
    objective: str
    artifact_paths: _containers.RepeatedScalarFieldContainer[str]
    outcomes: _containers.RepeatedCompositeFieldContainer[DevelopmentOutcome]
    acceptance_allow: _containers.RepeatedScalarFieldContainer[str]
    acceptance_deny: _containers.RepeatedScalarFieldContainer[str]
    allowed_effects: _containers.RepeatedScalarFieldContainer[str]
    max_tokens: int
    max_wall_seconds: int
    guidance: DevelopmentGuidance
    budget_policy: str
    plan_ref: _plan_ref_pb2.PlanRef
    execution_strategy: str
    def __init__(self, scenario: _Optional[str] = ..., work_item: _Optional[str] = ..., objective: _Optional[str] = ..., artifact_paths: _Optional[_Iterable[str]] = ..., outcomes: _Optional[_Iterable[_Union[DevelopmentOutcome, _Mapping]]] = ..., acceptance_allow: _Optional[_Iterable[str]] = ..., acceptance_deny: _Optional[_Iterable[str]] = ..., allowed_effects: _Optional[_Iterable[str]] = ..., max_tokens: _Optional[int] = ..., max_wall_seconds: _Optional[int] = ..., guidance: _Optional[_Union[DevelopmentGuidance, _Mapping]] = ..., budget_policy: _Optional[str] = ..., plan_ref: _Optional[_Union[_plan_ref_pb2.PlanRef, _Mapping]] = ..., execution_strategy: _Optional[str] = ...) -> None: ...

class DevelopmentGuidance(_message.Message):
    __slots__ = ("effort", "starting_state", "validation", "repair_related_code", "additional_instructions")
    EFFORT_FIELD_NUMBER: _ClassVar[int]
    STARTING_STATE_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_FIELD_NUMBER: _ClassVar[int]
    REPAIR_RELATED_CODE_FIELD_NUMBER: _ClassVar[int]
    ADDITIONAL_INSTRUCTIONS_FIELD_NUMBER: _ClassVar[int]
    effort: str
    starting_state: str
    validation: str
    repair_related_code: bool
    additional_instructions: str
    def __init__(self, effort: _Optional[str] = ..., starting_state: _Optional[str] = ..., validation: _Optional[str] = ..., repair_related_code: _Optional[bool] = ..., additional_instructions: _Optional[str] = ...) -> None: ...

class DevelopmentArtifact(_message.Message):
    __slots__ = ("path", "sha256", "size_bytes")
    PATH_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    path: str
    sha256: str
    size_bytes: int
    def __init__(self, path: _Optional[str] = ..., sha256: _Optional[str] = ..., size_bytes: _Optional[int] = ...) -> None: ...

class DevelopmentReviewFinding(_message.Message):
    __slots__ = ("code", "detail")
    CODE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    code: str
    detail: str
    def __init__(self, code: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class PreviewDevelopmentResponse(_message.Message):
    __slots__ = ("proposal_digest", "goal_message", "artifacts", "findings", "review_complete", "launch_ready", "launch_blockers")
    PROPOSAL_DIGEST_FIELD_NUMBER: _ClassVar[int]
    GOAL_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    REVIEW_COMPLETE_FIELD_NUMBER: _ClassVar[int]
    LAUNCH_READY_FIELD_NUMBER: _ClassVar[int]
    LAUNCH_BLOCKERS_FIELD_NUMBER: _ClassVar[int]
    proposal_digest: str
    goal_message: str
    artifacts: _containers.RepeatedCompositeFieldContainer[DevelopmentArtifact]
    findings: _containers.RepeatedCompositeFieldContainer[DevelopmentReviewFinding]
    review_complete: bool
    launch_ready: bool
    launch_blockers: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, proposal_digest: _Optional[str] = ..., goal_message: _Optional[str] = ..., artifacts: _Optional[_Iterable[_Union[DevelopmentArtifact, _Mapping]]] = ..., findings: _Optional[_Iterable[_Union[DevelopmentReviewFinding, _Mapping]]] = ..., review_complete: _Optional[bool] = ..., launch_ready: _Optional[bool] = ..., launch_blockers: _Optional[_Iterable[str]] = ...) -> None: ...

class ListTransitionsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListTransitionsResponse(_message.Message):
    __slots__ = ("transitions",)
    TRANSITIONS_FIELD_NUMBER: _ClassVar[int]
    transitions: _containers.RepeatedCompositeFieldContainer[_transition_pb2.Transition]
    def __init__(self, transitions: _Optional[_Iterable[_Union[_transition_pb2.Transition, _Mapping]]] = ...) -> None: ...

class StartTransitionRequest(_message.Message):
    __slots__ = ("transition_key", "subject_ref", "operator_inputs")
    class OperatorInputsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TRANSITION_KEY_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_REF_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_INPUTS_FIELD_NUMBER: _ClassVar[int]
    transition_key: str
    subject_ref: SubjectReference
    operator_inputs: _containers.ScalarMap[str, str]
    def __init__(self, transition_key: _Optional[str] = ..., subject_ref: _Optional[_Union[SubjectReference, _Mapping]] = ..., operator_inputs: _Optional[_Mapping[str, str]] = ...) -> None: ...

class SubjectReference(_message.Message):
    __slots__ = ("subject", "value")
    SUBJECT_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    subject: str
    value: str
    def __init__(self, subject: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...

class StartTransitionResponse(_message.Message):
    __slots__ = ("execution_id", "definition_digest", "entity_version", "apply_state", "outcome", "terminal_code")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_DIGEST_FIELD_NUMBER: _ClassVar[int]
    ENTITY_VERSION_FIELD_NUMBER: _ClassVar[int]
    APPLY_STATE_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_CODE_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    definition_digest: str
    entity_version: str
    apply_state: str
    outcome: str
    terminal_code: str
    def __init__(self, execution_id: _Optional[str] = ..., definition_digest: _Optional[str] = ..., entity_version: _Optional[str] = ..., apply_state: _Optional[str] = ..., outcome: _Optional[str] = ..., terminal_code: _Optional[str] = ...) -> None: ...

class ApplyTransitionRequest(_message.Message):
    __slots__ = ("transition_key", "execution_id")
    TRANSITION_KEY_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    transition_key: str
    execution_id: str
    def __init__(self, transition_key: _Optional[str] = ..., execution_id: _Optional[str] = ...) -> None: ...

class ApplyTransitionResponse(_message.Message):
    __slots__ = ("execution_id", "transition_key", "subject_ref", "outcome", "terminal_code", "applied_time", "definition_digest", "entity_version", "apply_state")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    TRANSITION_KEY_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_REF_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_CODE_FIELD_NUMBER: _ClassVar[int]
    APPLIED_TIME_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_DIGEST_FIELD_NUMBER: _ClassVar[int]
    ENTITY_VERSION_FIELD_NUMBER: _ClassVar[int]
    APPLY_STATE_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    transition_key: str
    subject_ref: str
    outcome: str
    terminal_code: str
    applied_time: str
    definition_digest: str
    entity_version: str
    apply_state: str
    def __init__(self, execution_id: _Optional[str] = ..., transition_key: _Optional[str] = ..., subject_ref: _Optional[str] = ..., outcome: _Optional[str] = ..., terminal_code: _Optional[str] = ..., applied_time: _Optional[str] = ..., definition_digest: _Optional[str] = ..., entity_version: _Optional[str] = ..., apply_state: _Optional[str] = ...) -> None: ...
