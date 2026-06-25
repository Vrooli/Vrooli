from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PlanStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PLAN_STATUS_UNSPECIFIED: _ClassVar[PlanStatus]
    PLAN_STATUS_DRAFT: _ClassVar[PlanStatus]
    PLAN_STATUS_ACTIVE: _ClassVar[PlanStatus]
    PLAN_STATUS_COMPLETE: _ClassVar[PlanStatus]
    PLAN_STATUS_ARCHIVED: _ClassVar[PlanStatus]

class PhaseStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PHASE_STATUS_UNSPECIFIED: _ClassVar[PhaseStatus]
    PHASE_STATUS_TODO: _ClassVar[PhaseStatus]
    PHASE_STATUS_ACTIVE: _ClassVar[PhaseStatus]
    PHASE_STATUS_DONE: _ClassVar[PhaseStatus]
    PHASE_STATUS_BLOCKED: _ClassVar[PhaseStatus]

class StalenessTier(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    STALENESS_TIER_UNSPECIFIED: _ClassVar[StalenessTier]
    STALENESS_TIER_FRESH: _ClassVar[StalenessTier]
    STALENESS_TIER_LIGHTLY_STALE: _ClassVar[StalenessTier]
    STALENESS_TIER_DEFINITELY_STALE: _ClassVar[StalenessTier]

class ReferenceKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    REFERENCE_KIND_UNSPECIFIED: _ClassVar[ReferenceKind]
    REFERENCE_KIND_CODE: _ClassVar[ReferenceKind]
    REFERENCE_KIND_REQ: _ClassVar[ReferenceKind]
    REFERENCE_KIND_DOC: _ClassVar[ReferenceKind]

class ReferenceResolution(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    REFERENCE_RESOLUTION_UNSPECIFIED: _ClassVar[ReferenceResolution]
    REFERENCE_RESOLUTION_RESOLVED: _ClassVar[ReferenceResolution]
    REFERENCE_RESOLUTION_UNRESOLVED: _ClassVar[ReferenceResolution]
    REFERENCE_RESOLUTION_FUTURE: _ClassVar[ReferenceResolution]
    REFERENCE_RESOLUTION_MISSING: _ClassVar[ReferenceResolution]

class FindingTriage(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FINDING_TRIAGE_UNSPECIFIED: _ClassVar[FindingTriage]
    FINDING_TRIAGE_CANDIDATE: _ClassVar[FindingTriage]
    FINDING_TRIAGE_PROMOTED: _ClassVar[FindingTriage]
    FINDING_TRIAGE_DISMISSED: _ClassVar[FindingTriage]

class Completeness(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COMPLETENESS_UNSPECIFIED: _ClassVar[Completeness]
    COMPLETENESS_FULL: _ClassVar[Completeness]
    COMPLETENESS_PARTIAL: _ClassVar[Completeness]

class ValidationVerdict(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VALIDATION_VERDICT_UNSPECIFIED: _ClassVar[ValidationVerdict]
    VALIDATION_VERDICT_PASS: _ClassVar[ValidationVerdict]
    VALIDATION_VERDICT_FAIL: _ClassVar[ValidationVerdict]
    VALIDATION_VERDICT_UNKNOWN: _ClassVar[ValidationVerdict]
PLAN_STATUS_UNSPECIFIED: PlanStatus
PLAN_STATUS_DRAFT: PlanStatus
PLAN_STATUS_ACTIVE: PlanStatus
PLAN_STATUS_COMPLETE: PlanStatus
PLAN_STATUS_ARCHIVED: PlanStatus
PHASE_STATUS_UNSPECIFIED: PhaseStatus
PHASE_STATUS_TODO: PhaseStatus
PHASE_STATUS_ACTIVE: PhaseStatus
PHASE_STATUS_DONE: PhaseStatus
PHASE_STATUS_BLOCKED: PhaseStatus
STALENESS_TIER_UNSPECIFIED: StalenessTier
STALENESS_TIER_FRESH: StalenessTier
STALENESS_TIER_LIGHTLY_STALE: StalenessTier
STALENESS_TIER_DEFINITELY_STALE: StalenessTier
REFERENCE_KIND_UNSPECIFIED: ReferenceKind
REFERENCE_KIND_CODE: ReferenceKind
REFERENCE_KIND_REQ: ReferenceKind
REFERENCE_KIND_DOC: ReferenceKind
REFERENCE_RESOLUTION_UNSPECIFIED: ReferenceResolution
REFERENCE_RESOLUTION_RESOLVED: ReferenceResolution
REFERENCE_RESOLUTION_UNRESOLVED: ReferenceResolution
REFERENCE_RESOLUTION_FUTURE: ReferenceResolution
REFERENCE_RESOLUTION_MISSING: ReferenceResolution
FINDING_TRIAGE_UNSPECIFIED: FindingTriage
FINDING_TRIAGE_CANDIDATE: FindingTriage
FINDING_TRIAGE_PROMOTED: FindingTriage
FINDING_TRIAGE_DISMISSED: FindingTriage
COMPLETENESS_UNSPECIFIED: Completeness
COMPLETENESS_FULL: Completeness
COMPLETENESS_PARTIAL: Completeness
VALIDATION_VERDICT_UNSPECIFIED: ValidationVerdict
VALIDATION_VERDICT_PASS: ValidationVerdict
VALIDATION_VERDICT_FAIL: ValidationVerdict
VALIDATION_VERDICT_UNKNOWN: ValidationVerdict

class Reference(_message.Message):
    __slots__ = ("id", "kind", "target", "future", "resolution", "staleness", "change_factor", "note")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    FUTURE_FIELD_NUMBER: _ClassVar[int]
    RESOLUTION_FIELD_NUMBER: _ClassVar[int]
    STALENESS_FIELD_NUMBER: _ClassVar[int]
    CHANGE_FACTOR_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: ReferenceKind
    target: str
    future: bool
    resolution: ReferenceResolution
    staleness: StalenessTier
    change_factor: float
    note: str
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[_Union[ReferenceKind, str]] = ..., target: _Optional[str] = ..., future: _Optional[bool] = ..., resolution: _Optional[_Union[ReferenceResolution, str]] = ..., staleness: _Optional[_Union[StalenessTier, str]] = ..., change_factor: _Optional[float] = ..., note: _Optional[str] = ...) -> None: ...

class RegressionAnchor(_message.Message):
    __slots__ = ("strategy", "scenario", "baseline_name", "head_sha", "allowlist_paths", "commands", "captured_at", "unavailable")
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    BASELINE_NAME_FIELD_NUMBER: _ClassVar[int]
    HEAD_SHA_FIELD_NUMBER: _ClassVar[int]
    ALLOWLIST_PATHS_FIELD_NUMBER: _ClassVar[int]
    COMMANDS_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_FIELD_NUMBER: _ClassVar[int]
    strategy: str
    scenario: str
    baseline_name: str
    head_sha: str
    allowlist_paths: _containers.RepeatedScalarFieldContainer[str]
    commands: _containers.RepeatedScalarFieldContainer[str]
    captured_at: str
    unavailable: bool
    def __init__(self, strategy: _Optional[str] = ..., scenario: _Optional[str] = ..., baseline_name: _Optional[str] = ..., head_sha: _Optional[str] = ..., allowlist_paths: _Optional[_Iterable[str]] = ..., commands: _Optional[_Iterable[str]] = ..., captured_at: _Optional[str] = ..., unavailable: _Optional[bool] = ...) -> None: ...

class Decision(_message.Message):
    __slots__ = ("id", "summary", "detail", "phase_id", "recorded_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    summary: str
    detail: str
    phase_id: str
    recorded_at: str
    def __init__(self, id: _Optional[str] = ..., summary: _Optional[str] = ..., detail: _Optional[str] = ..., phase_id: _Optional[str] = ..., recorded_at: _Optional[str] = ...) -> None: ...

class Finding(_message.Message):
    __slots__ = ("id", "title", "detail", "triage", "phase_id", "recorded_at", "attribution_run_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    TRIAGE_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTION_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    detail: str
    triage: FindingTriage
    phase_id: str
    recorded_at: str
    attribution_run_id: str
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., detail: _Optional[str] = ..., triage: _Optional[_Union[FindingTriage, str]] = ..., phase_id: _Optional[str] = ..., recorded_at: _Optional[str] = ..., attribution_run_id: _Optional[str] = ...) -> None: ...

class ValidationResult(_message.Message):
    __slots__ = ("id", "plan_id", "phase_id", "verdict", "staleness", "commands_run", "detail", "ran_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    STALENESS_FIELD_NUMBER: _ClassVar[int]
    COMMANDS_RUN_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    RAN_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    plan_id: str
    phase_id: str
    verdict: ValidationVerdict
    staleness: StalenessTier
    commands_run: _containers.RepeatedScalarFieldContainer[str]
    detail: str
    ran_at: str
    def __init__(self, id: _Optional[str] = ..., plan_id: _Optional[str] = ..., phase_id: _Optional[str] = ..., verdict: _Optional[_Union[ValidationVerdict, str]] = ..., staleness: _Optional[_Union[StalenessTier, str]] = ..., commands_run: _Optional[_Iterable[str]] = ..., detail: _Optional[str] = ..., ran_at: _Optional[str] = ...) -> None: ...

class Phase(_message.Message):
    __slots__ = ("id", "order", "title", "intent", "required_reading", "reminders", "baseline_scope", "acceptance", "status", "last_validation", "decisions", "findings", "references")
    ID_FIELD_NUMBER: _ClassVar[int]
    ORDER_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    INTENT_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_READING_FIELD_NUMBER: _ClassVar[int]
    REMINDERS_FIELD_NUMBER: _ClassVar[int]
    BASELINE_SCOPE_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LAST_VALIDATION_FIELD_NUMBER: _ClassVar[int]
    DECISIONS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    REFERENCES_FIELD_NUMBER: _ClassVar[int]
    id: str
    order: int
    title: str
    intent: str
    required_reading: _containers.RepeatedScalarFieldContainer[str]
    reminders: _containers.RepeatedScalarFieldContainer[str]
    baseline_scope: _containers.RepeatedScalarFieldContainer[str]
    acceptance: str
    status: PhaseStatus
    last_validation: ValidationResult
    decisions: _containers.RepeatedCompositeFieldContainer[Decision]
    findings: _containers.RepeatedCompositeFieldContainer[Finding]
    references: _containers.RepeatedCompositeFieldContainer[Reference]
    def __init__(self, id: _Optional[str] = ..., order: _Optional[int] = ..., title: _Optional[str] = ..., intent: _Optional[str] = ..., required_reading: _Optional[_Iterable[str]] = ..., reminders: _Optional[_Iterable[str]] = ..., baseline_scope: _Optional[_Iterable[str]] = ..., acceptance: _Optional[str] = ..., status: _Optional[_Union[PhaseStatus, str]] = ..., last_validation: _Optional[_Union[ValidationResult, _Mapping]] = ..., decisions: _Optional[_Iterable[_Union[Decision, _Mapping]]] = ..., findings: _Optional[_Iterable[_Union[Finding, _Mapping]]] = ..., references: _Optional[_Iterable[_Union[Reference, _Mapping]]] = ...) -> None: ...

class Plan(_message.Message):
    __slots__ = ("id", "slug", "title", "status", "content_hash", "created_at", "updated_at", "purpose", "scope", "constraints", "non_goals", "references", "regression_anchor", "definition_of_done", "phases", "supersedes", "superseded_by")
    ID_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    PURPOSE_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    CONSTRAINTS_FIELD_NUMBER: _ClassVar[int]
    NON_GOALS_FIELD_NUMBER: _ClassVar[int]
    REFERENCES_FIELD_NUMBER: _ClassVar[int]
    REGRESSION_ANCHOR_FIELD_NUMBER: _ClassVar[int]
    DEFINITION_OF_DONE_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDES_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDED_BY_FIELD_NUMBER: _ClassVar[int]
    id: str
    slug: str
    title: str
    status: PlanStatus
    content_hash: str
    created_at: str
    updated_at: str
    purpose: str
    scope: str
    constraints: str
    non_goals: str
    references: _containers.RepeatedCompositeFieldContainer[Reference]
    regression_anchor: RegressionAnchor
    definition_of_done: str
    phases: _containers.RepeatedCompositeFieldContainer[Phase]
    supersedes: _containers.RepeatedScalarFieldContainer[str]
    superseded_by: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., slug: _Optional[str] = ..., title: _Optional[str] = ..., status: _Optional[_Union[PlanStatus, str]] = ..., content_hash: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., purpose: _Optional[str] = ..., scope: _Optional[str] = ..., constraints: _Optional[str] = ..., non_goals: _Optional[str] = ..., references: _Optional[_Iterable[_Union[Reference, _Mapping]]] = ..., regression_anchor: _Optional[_Union[RegressionAnchor, _Mapping]] = ..., definition_of_done: _Optional[str] = ..., phases: _Optional[_Iterable[_Union[Phase, _Mapping]]] = ..., supersedes: _Optional[_Iterable[str]] = ..., superseded_by: _Optional[_Iterable[str]] = ...) -> None: ...

class PlanEdge(_message.Message):
    __slots__ = ("from_plan_id", "to_plan_id", "kind")
    FROM_PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    TO_PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    from_plan_id: str
    to_plan_id: str
    kind: str
    def __init__(self, from_plan_id: _Optional[str] = ..., to_plan_id: _Optional[str] = ..., kind: _Optional[str] = ...) -> None: ...

class VelocityPoint(_message.Message):
    __slots__ = ("id", "plan_id", "run_id", "wall_time_seconds", "tokens", "iterations", "completeness", "recorded_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    WALL_TIME_SECONDS_FIELD_NUMBER: _ClassVar[int]
    TOKENS_FIELD_NUMBER: _ClassVar[int]
    ITERATIONS_FIELD_NUMBER: _ClassVar[int]
    COMPLETENESS_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    plan_id: str
    run_id: str
    wall_time_seconds: int
    tokens: int
    iterations: int
    completeness: Completeness
    recorded_at: str
    def __init__(self, id: _Optional[str] = ..., plan_id: _Optional[str] = ..., run_id: _Optional[str] = ..., wall_time_seconds: _Optional[int] = ..., tokens: _Optional[int] = ..., iterations: _Optional[int] = ..., completeness: _Optional[_Union[Completeness, str]] = ..., recorded_at: _Optional[str] = ...) -> None: ...

class Handoff(_message.Message):
    __slots__ = ("id", "execution_id", "plan_id", "completeness", "resume_phase_id", "decisions", "candidate_findings", "last_validation", "staleness", "prose_handoff_ref", "assembled_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    COMPLETENESS_FIELD_NUMBER: _ClassVar[int]
    RESUME_PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    DECISIONS_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    LAST_VALIDATION_FIELD_NUMBER: _ClassVar[int]
    STALENESS_FIELD_NUMBER: _ClassVar[int]
    PROSE_HANDOFF_REF_FIELD_NUMBER: _ClassVar[int]
    ASSEMBLED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    execution_id: str
    plan_id: str
    completeness: Completeness
    resume_phase_id: str
    decisions: _containers.RepeatedCompositeFieldContainer[Decision]
    candidate_findings: _containers.RepeatedCompositeFieldContainer[Finding]
    last_validation: ValidationResult
    staleness: StalenessTier
    prose_handoff_ref: str
    assembled_at: str
    def __init__(self, id: _Optional[str] = ..., execution_id: _Optional[str] = ..., plan_id: _Optional[str] = ..., completeness: _Optional[_Union[Completeness, str]] = ..., resume_phase_id: _Optional[str] = ..., decisions: _Optional[_Iterable[_Union[Decision, _Mapping]]] = ..., candidate_findings: _Optional[_Iterable[_Union[Finding, _Mapping]]] = ..., last_validation: _Optional[_Union[ValidationResult, _Mapping]] = ..., staleness: _Optional[_Union[StalenessTier, str]] = ..., prose_handoff_ref: _Optional[str] = ..., assembled_at: _Optional[str] = ...) -> None: ...
