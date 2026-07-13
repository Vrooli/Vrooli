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

class RelevantContextKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RELEVANT_CONTEXT_KIND_UNSPECIFIED: _ClassVar[RelevantContextKind]
    RELEVANT_CONTEXT_KIND_SKILL: _ClassVar[RelevantContextKind]
    RELEVANT_CONTEXT_KIND_DOC: _ClassVar[RelevantContextKind]
    RELEVANT_CONTEXT_KIND_COMMAND: _ClassVar[RelevantContextKind]
    RELEVANT_CONTEXT_KIND_SEARCH: _ClassVar[RelevantContextKind]
    RELEVANT_CONTEXT_KIND_CODE_REF: _ClassVar[RelevantContextKind]
    RELEVANT_CONTEXT_KIND_REQ_REF: _ClassVar[RelevantContextKind]
    RELEVANT_CONTEXT_KIND_NOTE: _ClassVar[RelevantContextKind]

class RelevantContextScope(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RELEVANT_CONTEXT_SCOPE_UNSPECIFIED: _ClassVar[RelevantContextScope]
    RELEVANT_CONTEXT_SCOPE_GLOBAL: _ClassVar[RelevantContextScope]
    RELEVANT_CONTEXT_SCOPE_PHASE: _ClassVar[RelevantContextScope]

class RelevantContextRepeatPolicy(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RELEVANT_CONTEXT_REPEAT_POLICY_UNSPECIFIED: _ClassVar[RelevantContextRepeatPolicy]
    RELEVANT_CONTEXT_REPEAT_POLICY_ONCE_PER_EXECUTION: _ClassVar[RelevantContextRepeatPolicy]
    RELEVANT_CONTEXT_REPEAT_POLICY_ON_RESUME: _ClassVar[RelevantContextRepeatPolicy]
    RELEVANT_CONTEXT_REPEAT_POLICY_EVERY_PHASE: _ClassVar[RelevantContextRepeatPolicy]
    RELEVANT_CONTEXT_REPEAT_POLICY_PHASE_ENTRY: _ClassVar[RelevantContextRepeatPolicy]
    RELEVANT_CONTEXT_REPEAT_POLICY_AS_NEEDED: _ClassVar[RelevantContextRepeatPolicy]

class RelevantContextSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RELEVANT_CONTEXT_SOURCE_UNSPECIFIED: _ClassVar[RelevantContextSource]
    RELEVANT_CONTEXT_SOURCE_AUTHORED: _ClassVar[RelevantContextSource]
    RELEVANT_CONTEXT_SOURCE_DISCOVERED: _ClassVar[RelevantContextSource]
    RELEVANT_CONTEXT_SOURCE_MIGRATED: _ClassVar[RelevantContextSource]
    RELEVANT_CONTEXT_SOURCE_AUTOFILLED: _ClassVar[RelevantContextSource]

class RelevantContextStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RELEVANT_CONTEXT_STATUS_UNSPECIFIED: _ClassVar[RelevantContextStatus]
    RELEVANT_CONTEXT_STATUS_READY: _ClassVar[RelevantContextStatus]
    RELEVANT_CONTEXT_STATUS_DEGRADED: _ClassVar[RelevantContextStatus]
    RELEVANT_CONTEXT_STATUS_UNRESOLVED: _ClassVar[RelevantContextStatus]

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

class WorkPosture(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WORK_POSTURE_UNSPECIFIED: _ClassVar[WorkPosture]
    WORK_POSTURE_GREENFIELD: _ClassVar[WorkPosture]
    WORK_POSTURE_BROWNFIELD: _ClassVar[WorkPosture]

class WorkPostureSource(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WORK_POSTURE_SOURCE_UNSPECIFIED: _ClassVar[WorkPostureSource]
    WORK_POSTURE_SOURCE_DEFAULT: _ClassVar[WorkPostureSource]
    WORK_POSTURE_SOURCE_SERVICE_MATURITY: _ClassVar[WorkPostureSource]
    WORK_POSTURE_SOURCE_EXPLICIT_OVERRIDE: _ClassVar[WorkPostureSource]
    WORK_POSTURE_SOURCE_IMPORT_LEGACY: _ClassVar[WorkPostureSource]

class RenderedMirrorStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RENDERED_MIRROR_STATUS_UNSPECIFIED: _ClassVar[RenderedMirrorStatus]
    RENDERED_MIRROR_STATUS_FRESH: _ClassVar[RenderedMirrorStatus]
    RENDERED_MIRROR_STATUS_MISSING: _ClassVar[RenderedMirrorStatus]
    RENDERED_MIRROR_STATUS_STALE: _ClassVar[RenderedMirrorStatus]
    RENDERED_MIRROR_STATUS_WRITE_FAILED: _ClassVar[RenderedMirrorStatus]
    RENDERED_MIRROR_STATUS_UNKNOWN: _ClassVar[RenderedMirrorStatus]

class ValidationVerdict(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VALIDATION_VERDICT_UNSPECIFIED: _ClassVar[ValidationVerdict]
    VALIDATION_VERDICT_PASS: _ClassVar[ValidationVerdict]
    VALIDATION_VERDICT_FAIL: _ClassVar[ValidationVerdict]
    VALIDATION_VERDICT_UNKNOWN: _ClassVar[ValidationVerdict]

class NextActionKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    NEXT_ACTION_KIND_UNSPECIFIED: _ClassVar[NextActionKind]
    NEXT_ACTION_KIND_RECOMMENDED: _ClassVar[NextActionKind]
    NEXT_ACTION_KIND_ALTERNATIVE: _ClassVar[NextActionKind]
    NEXT_ACTION_KIND_OPTIONAL: _ClassVar[NextActionKind]
    NEXT_ACTION_KIND_RECOVERY: _ClassVar[NextActionKind]

class ValidationScopeMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VALIDATION_SCOPE_MODE_UNSPECIFIED: _ClassVar[ValidationScopeMode]
    VALIDATION_SCOPE_MODE_NARROW: _ClassVar[ValidationScopeMode]
    VALIDATION_SCOPE_MODE_FULL_PLAN: _ClassVar[ValidationScopeMode]

class LogEntryType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    LOG_ENTRY_TYPE_UNSPECIFIED: _ClassVar[LogEntryType]
    LOG_ENTRY_TYPE_DECISION: _ClassVar[LogEntryType]
    LOG_ENTRY_TYPE_FINDING: _ClassVar[LogEntryType]
    LOG_ENTRY_TYPE_BUG_REPORT: _ClassVar[LogEntryType]
    LOG_ENTRY_TYPE_RECORD: _ClassVar[LogEntryType]
    LOG_ENTRY_TYPE_NOTE: _ClassVar[LogEntryType]

class LogSyncStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    LOG_SYNC_STATUS_UNSPECIFIED: _ClassVar[LogSyncStatus]
    LOG_SYNC_STATUS_LOCAL: _ClassVar[LogSyncStatus]
    LOG_SYNC_STATUS_PENDING: _ClassVar[LogSyncStatus]
    LOG_SYNC_STATUS_SYNCED: _ClassVar[LogSyncStatus]
    LOG_SYNC_STATUS_FAILED: _ClassVar[LogSyncStatus]

class LogSeverity(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    LOG_SEVERITY_UNSPECIFIED: _ClassVar[LogSeverity]
    LOG_SEVERITY_INFO: _ClassVar[LogSeverity]
    LOG_SEVERITY_LOW: _ClassVar[LogSeverity]
    LOG_SEVERITY_MEDIUM: _ClassVar[LogSeverity]
    LOG_SEVERITY_HIGH: _ClassVar[LogSeverity]
    LOG_SEVERITY_CRITICAL: _ClassVar[LogSeverity]
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
RELEVANT_CONTEXT_KIND_UNSPECIFIED: RelevantContextKind
RELEVANT_CONTEXT_KIND_SKILL: RelevantContextKind
RELEVANT_CONTEXT_KIND_DOC: RelevantContextKind
RELEVANT_CONTEXT_KIND_COMMAND: RelevantContextKind
RELEVANT_CONTEXT_KIND_SEARCH: RelevantContextKind
RELEVANT_CONTEXT_KIND_CODE_REF: RelevantContextKind
RELEVANT_CONTEXT_KIND_REQ_REF: RelevantContextKind
RELEVANT_CONTEXT_KIND_NOTE: RelevantContextKind
RELEVANT_CONTEXT_SCOPE_UNSPECIFIED: RelevantContextScope
RELEVANT_CONTEXT_SCOPE_GLOBAL: RelevantContextScope
RELEVANT_CONTEXT_SCOPE_PHASE: RelevantContextScope
RELEVANT_CONTEXT_REPEAT_POLICY_UNSPECIFIED: RelevantContextRepeatPolicy
RELEVANT_CONTEXT_REPEAT_POLICY_ONCE_PER_EXECUTION: RelevantContextRepeatPolicy
RELEVANT_CONTEXT_REPEAT_POLICY_ON_RESUME: RelevantContextRepeatPolicy
RELEVANT_CONTEXT_REPEAT_POLICY_EVERY_PHASE: RelevantContextRepeatPolicy
RELEVANT_CONTEXT_REPEAT_POLICY_PHASE_ENTRY: RelevantContextRepeatPolicy
RELEVANT_CONTEXT_REPEAT_POLICY_AS_NEEDED: RelevantContextRepeatPolicy
RELEVANT_CONTEXT_SOURCE_UNSPECIFIED: RelevantContextSource
RELEVANT_CONTEXT_SOURCE_AUTHORED: RelevantContextSource
RELEVANT_CONTEXT_SOURCE_DISCOVERED: RelevantContextSource
RELEVANT_CONTEXT_SOURCE_MIGRATED: RelevantContextSource
RELEVANT_CONTEXT_SOURCE_AUTOFILLED: RelevantContextSource
RELEVANT_CONTEXT_STATUS_UNSPECIFIED: RelevantContextStatus
RELEVANT_CONTEXT_STATUS_READY: RelevantContextStatus
RELEVANT_CONTEXT_STATUS_DEGRADED: RelevantContextStatus
RELEVANT_CONTEXT_STATUS_UNRESOLVED: RelevantContextStatus
FINDING_TRIAGE_UNSPECIFIED: FindingTriage
FINDING_TRIAGE_CANDIDATE: FindingTriage
FINDING_TRIAGE_PROMOTED: FindingTriage
FINDING_TRIAGE_DISMISSED: FindingTriage
COMPLETENESS_UNSPECIFIED: Completeness
COMPLETENESS_FULL: Completeness
COMPLETENESS_PARTIAL: Completeness
WORK_POSTURE_UNSPECIFIED: WorkPosture
WORK_POSTURE_GREENFIELD: WorkPosture
WORK_POSTURE_BROWNFIELD: WorkPosture
WORK_POSTURE_SOURCE_UNSPECIFIED: WorkPostureSource
WORK_POSTURE_SOURCE_DEFAULT: WorkPostureSource
WORK_POSTURE_SOURCE_SERVICE_MATURITY: WorkPostureSource
WORK_POSTURE_SOURCE_EXPLICIT_OVERRIDE: WorkPostureSource
WORK_POSTURE_SOURCE_IMPORT_LEGACY: WorkPostureSource
RENDERED_MIRROR_STATUS_UNSPECIFIED: RenderedMirrorStatus
RENDERED_MIRROR_STATUS_FRESH: RenderedMirrorStatus
RENDERED_MIRROR_STATUS_MISSING: RenderedMirrorStatus
RENDERED_MIRROR_STATUS_STALE: RenderedMirrorStatus
RENDERED_MIRROR_STATUS_WRITE_FAILED: RenderedMirrorStatus
RENDERED_MIRROR_STATUS_UNKNOWN: RenderedMirrorStatus
VALIDATION_VERDICT_UNSPECIFIED: ValidationVerdict
VALIDATION_VERDICT_PASS: ValidationVerdict
VALIDATION_VERDICT_FAIL: ValidationVerdict
VALIDATION_VERDICT_UNKNOWN: ValidationVerdict
NEXT_ACTION_KIND_UNSPECIFIED: NextActionKind
NEXT_ACTION_KIND_RECOMMENDED: NextActionKind
NEXT_ACTION_KIND_ALTERNATIVE: NextActionKind
NEXT_ACTION_KIND_OPTIONAL: NextActionKind
NEXT_ACTION_KIND_RECOVERY: NextActionKind
VALIDATION_SCOPE_MODE_UNSPECIFIED: ValidationScopeMode
VALIDATION_SCOPE_MODE_NARROW: ValidationScopeMode
VALIDATION_SCOPE_MODE_FULL_PLAN: ValidationScopeMode
LOG_ENTRY_TYPE_UNSPECIFIED: LogEntryType
LOG_ENTRY_TYPE_DECISION: LogEntryType
LOG_ENTRY_TYPE_FINDING: LogEntryType
LOG_ENTRY_TYPE_BUG_REPORT: LogEntryType
LOG_ENTRY_TYPE_RECORD: LogEntryType
LOG_ENTRY_TYPE_NOTE: LogEntryType
LOG_SYNC_STATUS_UNSPECIFIED: LogSyncStatus
LOG_SYNC_STATUS_LOCAL: LogSyncStatus
LOG_SYNC_STATUS_PENDING: LogSyncStatus
LOG_SYNC_STATUS_SYNCED: LogSyncStatus
LOG_SYNC_STATUS_FAILED: LogSyncStatus
LOG_SEVERITY_UNSPECIFIED: LogSeverity
LOG_SEVERITY_INFO: LogSeverity
LOG_SEVERITY_LOW: LogSeverity
LOG_SEVERITY_MEDIUM: LogSeverity
LOG_SEVERITY_HIGH: LogSeverity
LOG_SEVERITY_CRITICAL: LogSeverity

class RenderedPlanMirror(_message.Message):
    __slots__ = ("path", "relative_path", "content_hash", "render_version", "rendered_at", "status", "last_error")
    PATH_FIELD_NUMBER: _ClassVar[int]
    RELATIVE_PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    RENDER_VERSION_FIELD_NUMBER: _ClassVar[int]
    RENDERED_AT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    LAST_ERROR_FIELD_NUMBER: _ClassVar[int]
    path: str
    relative_path: str
    content_hash: str
    render_version: str
    rendered_at: str
    status: RenderedMirrorStatus
    last_error: str
    def __init__(self, path: _Optional[str] = ..., relative_path: _Optional[str] = ..., content_hash: _Optional[str] = ..., render_version: _Optional[str] = ..., rendered_at: _Optional[str] = ..., status: _Optional[_Union[RenderedMirrorStatus, str]] = ..., last_error: _Optional[str] = ...) -> None: ...

class LegacySection(_message.Message):
    __slots__ = ("heading", "content", "mapped_to", "preservation_reason")
    HEADING_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    MAPPED_TO_FIELD_NUMBER: _ClassVar[int]
    PRESERVATION_REASON_FIELD_NUMBER: _ClassVar[int]
    heading: str
    content: str
    mapped_to: str
    preservation_reason: str
    def __init__(self, heading: _Optional[str] = ..., content: _Optional[str] = ..., mapped_to: _Optional[str] = ..., preservation_reason: _Optional[str] = ...) -> None: ...

class ImportProvenance(_message.Message):
    __slots__ = ("source_path", "imported_at", "original_format", "note", "workspace_id", "workspace_root")
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    IMPORTED_AT_FIELD_NUMBER: _ClassVar[int]
    ORIGINAL_FORMAT_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_ROOT_FIELD_NUMBER: _ClassVar[int]
    source_path: str
    imported_at: str
    original_format: str
    note: str
    workspace_id: str
    workspace_root: str
    def __init__(self, source_path: _Optional[str] = ..., imported_at: _Optional[str] = ..., original_format: _Optional[str] = ..., note: _Optional[str] = ..., workspace_id: _Optional[str] = ..., workspace_root: _Optional[str] = ...) -> None: ...

class NextAction(_message.Message):
    __slots__ = ("id", "kind", "label", "reason", "argv", "content_placeholder", "blocked_by")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    ARGV_FIELD_NUMBER: _ClassVar[int]
    CONTENT_PLACEHOLDER_FIELD_NUMBER: _ClassVar[int]
    BLOCKED_BY_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: NextActionKind
    label: str
    reason: str
    argv: _containers.RepeatedScalarFieldContainer[str]
    content_placeholder: str
    blocked_by: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[_Union[NextActionKind, str]] = ..., label: _Optional[str] = ..., reason: _Optional[str] = ..., argv: _Optional[_Iterable[str]] = ..., content_placeholder: _Optional[str] = ..., blocked_by: _Optional[_Iterable[str]] = ...) -> None: ...

class ChecklistItem(_message.Message):
    __slots__ = ("key", "label", "state", "detail")
    KEY_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    key: str
    label: str
    state: str
    detail: str
    def __init__(self, key: _Optional[str] = ..., label: _Optional[str] = ..., state: _Optional[str] = ..., detail: _Optional[str] = ...) -> None: ...

class GuidedStep(_message.Message):
    __slots__ = ("step_kind", "title", "summary", "instructions", "required_inputs", "examples", "common_mistakes", "next_actions", "checklist")
    STEP_KIND_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    INSTRUCTIONS_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_INPUTS_FIELD_NUMBER: _ClassVar[int]
    EXAMPLES_FIELD_NUMBER: _ClassVar[int]
    COMMON_MISTAKES_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    CHECKLIST_FIELD_NUMBER: _ClassVar[int]
    step_kind: str
    title: str
    summary: str
    instructions: _containers.RepeatedScalarFieldContainer[str]
    required_inputs: _containers.RepeatedScalarFieldContainer[str]
    examples: _containers.RepeatedScalarFieldContainer[str]
    common_mistakes: _containers.RepeatedScalarFieldContainer[str]
    next_actions: _containers.RepeatedCompositeFieldContainer[NextAction]
    checklist: _containers.RepeatedCompositeFieldContainer[ChecklistItem]
    def __init__(self, step_kind: _Optional[str] = ..., title: _Optional[str] = ..., summary: _Optional[str] = ..., instructions: _Optional[_Iterable[str]] = ..., required_inputs: _Optional[_Iterable[str]] = ..., examples: _Optional[_Iterable[str]] = ..., common_mistakes: _Optional[_Iterable[str]] = ..., next_actions: _Optional[_Iterable[_Union[NextAction, _Mapping]]] = ..., checklist: _Optional[_Iterable[_Union[ChecklistItem, _Mapping]]] = ...) -> None: ...

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

class RelevantContextItem(_message.Message):
    __slots__ = ("id", "kind", "scope", "phase_id", "label", "reason", "instruction", "command", "argv", "target", "required", "repeat_policy", "source", "status", "status_detail")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    INSTRUCTION_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    ARGV_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    REPEAT_POLICY_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STATUS_DETAIL_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: RelevantContextKind
    scope: RelevantContextScope
    phase_id: str
    label: str
    reason: str
    instruction: str
    command: str
    argv: _containers.RepeatedScalarFieldContainer[str]
    target: str
    required: bool
    repeat_policy: RelevantContextRepeatPolicy
    source: RelevantContextSource
    status: RelevantContextStatus
    status_detail: str
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[_Union[RelevantContextKind, str]] = ..., scope: _Optional[_Union[RelevantContextScope, str]] = ..., phase_id: _Optional[str] = ..., label: _Optional[str] = ..., reason: _Optional[str] = ..., instruction: _Optional[str] = ..., command: _Optional[str] = ..., argv: _Optional[_Iterable[str]] = ..., target: _Optional[str] = ..., required: _Optional[bool] = ..., repeat_policy: _Optional[_Union[RelevantContextRepeatPolicy, str]] = ..., source: _Optional[_Union[RelevantContextSource, str]] = ..., status: _Optional[_Union[RelevantContextStatus, str]] = ..., status_detail: _Optional[str] = ...) -> None: ...

class ChangeBoundary(_message.Message):
    __slots__ = ("acceptance_allow", "acceptance_deny", "operator_only_reason")
    ACCEPTANCE_ALLOW_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_DENY_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_ONLY_REASON_FIELD_NUMBER: _ClassVar[int]
    acceptance_allow: _containers.RepeatedScalarFieldContainer[str]
    acceptance_deny: _containers.RepeatedScalarFieldContainer[str]
    operator_only_reason: str
    def __init__(self, acceptance_allow: _Optional[_Iterable[str]] = ..., acceptance_deny: _Optional[_Iterable[str]] = ..., operator_only_reason: _Optional[str] = ...) -> None: ...

class ValidationScope(_message.Message):
    __slots__ = ("mode", "boundary", "rationale")
    MODE_FIELD_NUMBER: _ClassVar[int]
    BOUNDARY_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    mode: ValidationScopeMode
    boundary: ChangeBoundary
    rationale: str
    def __init__(self, mode: _Optional[_Union[ValidationScopeMode, str]] = ..., boundary: _Optional[_Union[ChangeBoundary, _Mapping]] = ..., rationale: _Optional[str] = ...) -> None: ...

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

class BaselineSetIntent(_message.Message):
    __slots__ = ("name", "scenario_targets", "repo_paths", "capture_policy", "compatibility")
    NAME_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_TARGETS_FIELD_NUMBER: _ClassVar[int]
    REPO_PATHS_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_POLICY_FIELD_NUMBER: _ClassVar[int]
    COMPATIBILITY_FIELD_NUMBER: _ClassVar[int]
    name: str
    scenario_targets: _containers.RepeatedScalarFieldContainer[str]
    repo_paths: _containers.RepeatedScalarFieldContainer[str]
    capture_policy: str
    compatibility: str
    def __init__(self, name: _Optional[str] = ..., scenario_targets: _Optional[_Iterable[str]] = ..., repo_paths: _Optional[_Iterable[str]] = ..., capture_policy: _Optional[str] = ..., compatibility: _Optional[str] = ...) -> None: ...

class DownstreamRef(_message.Message):
    __slots__ = ("system", "kind", "reference", "detail", "synced_at", "capture")
    SYSTEM_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    SYNCED_AT_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_FIELD_NUMBER: _ClassVar[int]
    system: str
    kind: str
    reference: str
    detail: str
    synced_at: str
    capture: CaptureDisposition
    def __init__(self, system: _Optional[str] = ..., kind: _Optional[str] = ..., reference: _Optional[str] = ..., detail: _Optional[str] = ..., synced_at: _Optional[str] = ..., capture: _Optional[_Union[CaptureDisposition, _Mapping]] = ...) -> None: ...

class CaptureDiagnostic(_message.Message):
    __slots__ = ("field", "value", "message")
    FIELD_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    field: str
    value: str
    message: str
    def __init__(self, field: _Optional[str] = ..., value: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class CaptureDisposition(_message.Message):
    __slots__ = ("state", "draft_id", "needs", "invalid", "warnings", "next_action")
    STATE_FIELD_NUMBER: _ClassVar[int]
    DRAFT_ID_FIELD_NUMBER: _ClassVar[int]
    NEEDS_FIELD_NUMBER: _ClassVar[int]
    INVALID_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTION_FIELD_NUMBER: _ClassVar[int]
    state: str
    draft_id: str
    needs: _containers.RepeatedScalarFieldContainer[str]
    invalid: _containers.RepeatedCompositeFieldContainer[CaptureDiagnostic]
    warnings: _containers.RepeatedScalarFieldContainer[str]
    next_action: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, state: _Optional[str] = ..., draft_id: _Optional[str] = ..., needs: _Optional[_Iterable[str]] = ..., invalid: _Optional[_Iterable[_Union[CaptureDiagnostic, _Mapping]]] = ..., warnings: _Optional[_Iterable[str]] = ..., next_action: _Optional[_Iterable[str]] = ...) -> None: ...

class BugReportPayload(_message.Message):
    __slots__ = ("signal_type", "severity", "repro", "expected", "actual", "description", "context", "honesty_flags")
    class ContextEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SIGNAL_TYPE_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    REPRO_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    HONESTY_FLAGS_FIELD_NUMBER: _ClassVar[int]
    signal_type: str
    severity: str
    repro: _containers.RepeatedScalarFieldContainer[str]
    expected: str
    actual: str
    description: str
    context: _containers.ScalarMap[str, str]
    honesty_flags: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, signal_type: _Optional[str] = ..., severity: _Optional[str] = ..., repro: _Optional[_Iterable[str]] = ..., expected: _Optional[str] = ..., actual: _Optional[str] = ..., description: _Optional[str] = ..., context: _Optional[_Mapping[str, str]] = ..., honesty_flags: _Optional[_Iterable[str]] = ...) -> None: ...

class RecordPayload(_message.Message):
    __slots__ = ("kind", "scenario", "trigger", "approach", "evidence", "outcome", "created_by")
    KIND_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_FIELD_NUMBER: _ClassVar[int]
    APPROACH_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    kind: str
    scenario: str
    trigger: str
    approach: str
    evidence: str
    outcome: str
    created_by: str
    def __init__(self, kind: _Optional[str] = ..., scenario: _Optional[str] = ..., trigger: _Optional[str] = ..., approach: _Optional[str] = ..., evidence: _Optional[str] = ..., outcome: _Optional[str] = ..., created_by: _Optional[str] = ...) -> None: ...

class LogEntry(_message.Message):
    __slots__ = ("id", "type", "plan_id", "execution_id", "phase_id", "title", "detail", "severity", "triage", "sync_status", "downstream", "source_command", "evidence", "attribution_run_id", "idempotency_key", "supersedes_id", "promoted_from_id", "created_at", "updated_at", "bug", "record", "capture")
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    SEVERITY_FIELD_NUMBER: _ClassVar[int]
    TRIAGE_FIELD_NUMBER: _ClassVar[int]
    SYNC_STATUS_FIELD_NUMBER: _ClassVar[int]
    DOWNSTREAM_FIELD_NUMBER: _ClassVar[int]
    SOURCE_COMMAND_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    ATTRIBUTION_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    SUPERSEDES_ID_FIELD_NUMBER: _ClassVar[int]
    PROMOTED_FROM_ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    BUG_FIELD_NUMBER: _ClassVar[int]
    RECORD_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_FIELD_NUMBER: _ClassVar[int]
    id: str
    type: LogEntryType
    plan_id: str
    execution_id: str
    phase_id: str
    title: str
    detail: str
    severity: LogSeverity
    triage: FindingTriage
    sync_status: LogSyncStatus
    downstream: DownstreamRef
    source_command: str
    evidence: _containers.RepeatedScalarFieldContainer[str]
    attribution_run_id: str
    idempotency_key: str
    supersedes_id: str
    promoted_from_id: str
    created_at: str
    updated_at: str
    bug: BugReportPayload
    record: RecordPayload
    capture: CaptureDisposition
    def __init__(self, id: _Optional[str] = ..., type: _Optional[_Union[LogEntryType, str]] = ..., plan_id: _Optional[str] = ..., execution_id: _Optional[str] = ..., phase_id: _Optional[str] = ..., title: _Optional[str] = ..., detail: _Optional[str] = ..., severity: _Optional[_Union[LogSeverity, str]] = ..., triage: _Optional[_Union[FindingTriage, str]] = ..., sync_status: _Optional[_Union[LogSyncStatus, str]] = ..., downstream: _Optional[_Union[DownstreamRef, _Mapping]] = ..., source_command: _Optional[str] = ..., evidence: _Optional[_Iterable[str]] = ..., attribution_run_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., supersedes_id: _Optional[str] = ..., promoted_from_id: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., bug: _Optional[_Union[BugReportPayload, _Mapping]] = ..., record: _Optional[_Union[RecordPayload, _Mapping]] = ..., capture: _Optional[_Union[CaptureDisposition, _Mapping]] = ...) -> None: ...

class LogSummaryItem(_message.Message):
    __slots__ = ("id", "type", "title", "sync_status", "triage", "phase_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SYNC_STATUS_FIELD_NUMBER: _ClassVar[int]
    TRIAGE_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    type: LogEntryType
    title: str
    sync_status: LogSyncStatus
    triage: FindingTriage
    phase_id: str
    def __init__(self, id: _Optional[str] = ..., type: _Optional[_Union[LogEntryType, str]] = ..., title: _Optional[str] = ..., sync_status: _Optional[_Union[LogSyncStatus, str]] = ..., triage: _Optional[_Union[FindingTriage, str]] = ..., phase_id: _Optional[str] = ...) -> None: ...

class LogSummary(_message.Message):
    __slots__ = ("total", "decisions", "findings", "bug_reports", "records", "notes", "candidate_findings", "pending_sync", "failed_sync", "recent")
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    DECISIONS_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    BUG_REPORTS_FIELD_NUMBER: _ClassVar[int]
    RECORDS_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    PENDING_SYNC_FIELD_NUMBER: _ClassVar[int]
    FAILED_SYNC_FIELD_NUMBER: _ClassVar[int]
    RECENT_FIELD_NUMBER: _ClassVar[int]
    total: int
    decisions: int
    findings: int
    bug_reports: int
    records: int
    notes: int
    candidate_findings: int
    pending_sync: int
    failed_sync: int
    recent: _containers.RepeatedCompositeFieldContainer[LogSummaryItem]
    def __init__(self, total: _Optional[int] = ..., decisions: _Optional[int] = ..., findings: _Optional[int] = ..., bug_reports: _Optional[int] = ..., records: _Optional[int] = ..., notes: _Optional[int] = ..., candidate_findings: _Optional[int] = ..., pending_sync: _Optional[int] = ..., failed_sync: _Optional[int] = ..., recent: _Optional[_Iterable[_Union[LogSummaryItem, _Mapping]]] = ...) -> None: ...

class ValidationResult(_message.Message):
    __slots__ = ("id", "plan_id", "phase_id", "verdict", "staleness", "commands_run", "detail", "ran_at", "command_findings")
    ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    STALENESS_FIELD_NUMBER: _ClassVar[int]
    COMMANDS_RUN_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    RAN_AT_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FINDINGS_FIELD_NUMBER: _ClassVar[int]
    id: str
    plan_id: str
    phase_id: str
    verdict: ValidationVerdict
    staleness: StalenessTier
    commands_run: _containers.RepeatedScalarFieldContainer[str]
    detail: str
    ran_at: str
    command_findings: _containers.RepeatedCompositeFieldContainer[CommandValidationFinding]
    def __init__(self, id: _Optional[str] = ..., plan_id: _Optional[str] = ..., phase_id: _Optional[str] = ..., verdict: _Optional[_Union[ValidationVerdict, str]] = ..., staleness: _Optional[_Union[StalenessTier, str]] = ..., commands_run: _Optional[_Iterable[str]] = ..., detail: _Optional[str] = ..., ran_at: _Optional[str] = ..., command_findings: _Optional[_Iterable[_Union[CommandValidationFinding, _Mapping]]] = ...) -> None: ...

class CommandValidationFinding(_message.Message):
    __slots__ = ("command_text", "verdict", "validation_level", "message", "location", "issue_codes", "suggestions", "guidance")
    COMMAND_TEXT_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_LEVEL_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    ISSUE_CODES_FIELD_NUMBER: _ClassVar[int]
    SUGGESTIONS_FIELD_NUMBER: _ClassVar[int]
    GUIDANCE_FIELD_NUMBER: _ClassVar[int]
    command_text: str
    verdict: str
    validation_level: str
    message: str
    location: str
    issue_codes: _containers.RepeatedScalarFieldContainer[str]
    suggestions: _containers.RepeatedScalarFieldContainer[str]
    guidance: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, command_text: _Optional[str] = ..., verdict: _Optional[str] = ..., validation_level: _Optional[str] = ..., message: _Optional[str] = ..., location: _Optional[str] = ..., issue_codes: _Optional[_Iterable[str]] = ..., suggestions: _Optional[_Iterable[str]] = ..., guidance: _Optional[_Iterable[str]] = ...) -> None: ...

class Phase(_message.Message):
    __slots__ = ("id", "order", "title", "intent", "required_reading", "reminders", "baseline_scope", "acceptance", "status", "last_validation", "references", "relevant_context", "affected_areas", "steps", "expected_outputs", "validation", "handoff_notes", "risks_hazards", "change_boundary", "validation_scope")
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
    REFERENCES_FIELD_NUMBER: _ClassVar[int]
    RELEVANT_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    AFFECTED_AREAS_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_OUTPUTS_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_FIELD_NUMBER: _ClassVar[int]
    HANDOFF_NOTES_FIELD_NUMBER: _ClassVar[int]
    RISKS_HAZARDS_FIELD_NUMBER: _ClassVar[int]
    CHANGE_BOUNDARY_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_SCOPE_FIELD_NUMBER: _ClassVar[int]
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
    references: _containers.RepeatedCompositeFieldContainer[Reference]
    relevant_context: _containers.RepeatedCompositeFieldContainer[RelevantContextItem]
    affected_areas: _containers.RepeatedScalarFieldContainer[str]
    steps: _containers.RepeatedScalarFieldContainer[str]
    expected_outputs: _containers.RepeatedScalarFieldContainer[str]
    validation: str
    handoff_notes: str
    risks_hazards: _containers.RepeatedScalarFieldContainer[str]
    change_boundary: ChangeBoundary
    validation_scope: ValidationScope
    def __init__(self, id: _Optional[str] = ..., order: _Optional[int] = ..., title: _Optional[str] = ..., intent: _Optional[str] = ..., required_reading: _Optional[_Iterable[str]] = ..., reminders: _Optional[_Iterable[str]] = ..., baseline_scope: _Optional[_Iterable[str]] = ..., acceptance: _Optional[str] = ..., status: _Optional[_Union[PhaseStatus, str]] = ..., last_validation: _Optional[_Union[ValidationResult, _Mapping]] = ..., references: _Optional[_Iterable[_Union[Reference, _Mapping]]] = ..., relevant_context: _Optional[_Iterable[_Union[RelevantContextItem, _Mapping]]] = ..., affected_areas: _Optional[_Iterable[str]] = ..., steps: _Optional[_Iterable[str]] = ..., expected_outputs: _Optional[_Iterable[str]] = ..., validation: _Optional[str] = ..., handoff_notes: _Optional[str] = ..., risks_hazards: _Optional[_Iterable[str]] = ..., change_boundary: _Optional[_Union[ChangeBoundary, _Mapping]] = ..., validation_scope: _Optional[_Union[ValidationScope, _Mapping]] = ...) -> None: ...

class PlanDecision(_message.Message):
    __slots__ = ("title", "statement")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    STATEMENT_FIELD_NUMBER: _ClassVar[int]
    title: str
    statement: str
    def __init__(self, title: _Optional[str] = ..., statement: _Optional[str] = ...) -> None: ...

class PlanAssumption(_message.Message):
    __slots__ = ("statement", "mitigation")
    STATEMENT_FIELD_NUMBER: _ClassVar[int]
    MITIGATION_FIELD_NUMBER: _ClassVar[int]
    statement: str
    mitigation: str
    def __init__(self, statement: _Optional[str] = ..., mitigation: _Optional[str] = ...) -> None: ...

class Plan(_message.Message):
    __slots__ = ("id", "slug", "title", "status", "content_hash", "created_at", "updated_at", "purpose", "scope", "constraints", "non_goals", "references", "regression_anchor", "definition_of_done", "phases", "supersedes", "superseded_by", "relevant_context", "problem_statement", "target_outcome", "assumptions", "technical_approach", "validation_strategy", "final_validation_commands", "risks_hazards", "prohibited_approaches", "work_posture", "work_posture_source", "work_posture_detail", "import_provenance", "preserved_legacy_sections", "change_boundary", "mirror", "workspace_id", "workspace_root", "decisions", "assumption_risks", "baseline_set")
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
    RELEVANT_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    PROBLEM_STATEMENT_FIELD_NUMBER: _ClassVar[int]
    TARGET_OUTCOME_FIELD_NUMBER: _ClassVar[int]
    ASSUMPTIONS_FIELD_NUMBER: _ClassVar[int]
    TECHNICAL_APPROACH_FIELD_NUMBER: _ClassVar[int]
    VALIDATION_STRATEGY_FIELD_NUMBER: _ClassVar[int]
    FINAL_VALIDATION_COMMANDS_FIELD_NUMBER: _ClassVar[int]
    RISKS_HAZARDS_FIELD_NUMBER: _ClassVar[int]
    PROHIBITED_APPROACHES_FIELD_NUMBER: _ClassVar[int]
    WORK_POSTURE_FIELD_NUMBER: _ClassVar[int]
    WORK_POSTURE_SOURCE_FIELD_NUMBER: _ClassVar[int]
    WORK_POSTURE_DETAIL_FIELD_NUMBER: _ClassVar[int]
    IMPORT_PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    PRESERVED_LEGACY_SECTIONS_FIELD_NUMBER: _ClassVar[int]
    CHANGE_BOUNDARY_FIELD_NUMBER: _ClassVar[int]
    MIRROR_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_ROOT_FIELD_NUMBER: _ClassVar[int]
    DECISIONS_FIELD_NUMBER: _ClassVar[int]
    ASSUMPTION_RISKS_FIELD_NUMBER: _ClassVar[int]
    BASELINE_SET_FIELD_NUMBER: _ClassVar[int]
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
    relevant_context: _containers.RepeatedCompositeFieldContainer[RelevantContextItem]
    problem_statement: str
    target_outcome: str
    assumptions: str
    technical_approach: str
    validation_strategy: str
    final_validation_commands: _containers.RepeatedScalarFieldContainer[str]
    risks_hazards: str
    prohibited_approaches: str
    work_posture: WorkPosture
    work_posture_source: WorkPostureSource
    work_posture_detail: str
    import_provenance: ImportProvenance
    preserved_legacy_sections: _containers.RepeatedCompositeFieldContainer[LegacySection]
    change_boundary: ChangeBoundary
    mirror: RenderedPlanMirror
    workspace_id: str
    workspace_root: str
    decisions: _containers.RepeatedCompositeFieldContainer[PlanDecision]
    assumption_risks: _containers.RepeatedCompositeFieldContainer[PlanAssumption]
    baseline_set: BaselineSetIntent
    def __init__(self, id: _Optional[str] = ..., slug: _Optional[str] = ..., title: _Optional[str] = ..., status: _Optional[_Union[PlanStatus, str]] = ..., content_hash: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., purpose: _Optional[str] = ..., scope: _Optional[str] = ..., constraints: _Optional[str] = ..., non_goals: _Optional[str] = ..., references: _Optional[_Iterable[_Union[Reference, _Mapping]]] = ..., regression_anchor: _Optional[_Union[RegressionAnchor, _Mapping]] = ..., definition_of_done: _Optional[str] = ..., phases: _Optional[_Iterable[_Union[Phase, _Mapping]]] = ..., supersedes: _Optional[_Iterable[str]] = ..., superseded_by: _Optional[_Iterable[str]] = ..., relevant_context: _Optional[_Iterable[_Union[RelevantContextItem, _Mapping]]] = ..., problem_statement: _Optional[str] = ..., target_outcome: _Optional[str] = ..., assumptions: _Optional[str] = ..., technical_approach: _Optional[str] = ..., validation_strategy: _Optional[str] = ..., final_validation_commands: _Optional[_Iterable[str]] = ..., risks_hazards: _Optional[str] = ..., prohibited_approaches: _Optional[str] = ..., work_posture: _Optional[_Union[WorkPosture, str]] = ..., work_posture_source: _Optional[_Union[WorkPostureSource, str]] = ..., work_posture_detail: _Optional[str] = ..., import_provenance: _Optional[_Union[ImportProvenance, _Mapping]] = ..., preserved_legacy_sections: _Optional[_Iterable[_Union[LegacySection, _Mapping]]] = ..., change_boundary: _Optional[_Union[ChangeBoundary, _Mapping]] = ..., mirror: _Optional[_Union[RenderedPlanMirror, _Mapping]] = ..., workspace_id: _Optional[str] = ..., workspace_root: _Optional[str] = ..., decisions: _Optional[_Iterable[_Union[PlanDecision, _Mapping]]] = ..., assumption_risks: _Optional[_Iterable[_Union[PlanAssumption, _Mapping]]] = ..., baseline_set: _Optional[_Union[BaselineSetIntent, _Mapping]] = ...) -> None: ...

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
    __slots__ = ("id", "execution_id", "plan_id", "completeness", "resume_phase_id", "log_summary", "log_entries", "last_validation", "staleness", "prose_handoff_ref", "assembled_at", "change_boundary")
    ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    COMPLETENESS_FIELD_NUMBER: _ClassVar[int]
    RESUME_PHASE_ID_FIELD_NUMBER: _ClassVar[int]
    LOG_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    LOG_ENTRIES_FIELD_NUMBER: _ClassVar[int]
    LAST_VALIDATION_FIELD_NUMBER: _ClassVar[int]
    STALENESS_FIELD_NUMBER: _ClassVar[int]
    PROSE_HANDOFF_REF_FIELD_NUMBER: _ClassVar[int]
    ASSEMBLED_AT_FIELD_NUMBER: _ClassVar[int]
    CHANGE_BOUNDARY_FIELD_NUMBER: _ClassVar[int]
    id: str
    execution_id: str
    plan_id: str
    completeness: Completeness
    resume_phase_id: str
    log_summary: LogSummary
    log_entries: _containers.RepeatedCompositeFieldContainer[LogEntry]
    last_validation: ValidationResult
    staleness: StalenessTier
    prose_handoff_ref: str
    assembled_at: str
    change_boundary: ChangeBoundary
    def __init__(self, id: _Optional[str] = ..., execution_id: _Optional[str] = ..., plan_id: _Optional[str] = ..., completeness: _Optional[_Union[Completeness, str]] = ..., resume_phase_id: _Optional[str] = ..., log_summary: _Optional[_Union[LogSummary, _Mapping]] = ..., log_entries: _Optional[_Iterable[_Union[LogEntry, _Mapping]]] = ..., last_validation: _Optional[_Union[ValidationResult, _Mapping]] = ..., staleness: _Optional[_Union[StalenessTier, str]] = ..., prose_handoff_ref: _Optional[str] = ..., assembled_at: _Optional[str] = ..., change_boundary: _Optional[_Union[ChangeBoundary, _Mapping]] = ...) -> None: ...
