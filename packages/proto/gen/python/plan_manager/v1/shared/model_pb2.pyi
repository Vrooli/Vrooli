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
VALIDATION_VERDICT_UNSPECIFIED: ValidationVerdict
VALIDATION_VERDICT_PASS: ValidationVerdict
VALIDATION_VERDICT_FAIL: ValidationVerdict
VALIDATION_VERDICT_UNKNOWN: ValidationVerdict
NEXT_ACTION_KIND_UNSPECIFIED: NextActionKind
NEXT_ACTION_KIND_RECOMMENDED: NextActionKind
NEXT_ACTION_KIND_ALTERNATIVE: NextActionKind
NEXT_ACTION_KIND_OPTIONAL: NextActionKind
NEXT_ACTION_KIND_RECOVERY: NextActionKind
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

class GuidedStep(_message.Message):
    __slots__ = ("step_kind", "title", "summary", "instructions", "required_inputs", "examples", "common_mistakes", "next_actions")
    STEP_KIND_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    INSTRUCTIONS_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_INPUTS_FIELD_NUMBER: _ClassVar[int]
    EXAMPLES_FIELD_NUMBER: _ClassVar[int]
    COMMON_MISTAKES_FIELD_NUMBER: _ClassVar[int]
    NEXT_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    step_kind: str
    title: str
    summary: str
    instructions: _containers.RepeatedScalarFieldContainer[str]
    required_inputs: _containers.RepeatedScalarFieldContainer[str]
    examples: _containers.RepeatedScalarFieldContainer[str]
    common_mistakes: _containers.RepeatedScalarFieldContainer[str]
    next_actions: _containers.RepeatedCompositeFieldContainer[NextAction]
    def __init__(self, step_kind: _Optional[str] = ..., title: _Optional[str] = ..., summary: _Optional[str] = ..., instructions: _Optional[_Iterable[str]] = ..., required_inputs: _Optional[_Iterable[str]] = ..., examples: _Optional[_Iterable[str]] = ..., common_mistakes: _Optional[_Iterable[str]] = ..., next_actions: _Optional[_Iterable[_Union[NextAction, _Mapping]]] = ...) -> None: ...

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

class DownstreamRef(_message.Message):
    __slots__ = ("system", "kind", "reference", "detail", "synced_at")
    SYSTEM_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    SYNCED_AT_FIELD_NUMBER: _ClassVar[int]
    system: str
    kind: str
    reference: str
    detail: str
    synced_at: str
    def __init__(self, system: _Optional[str] = ..., kind: _Optional[str] = ..., reference: _Optional[str] = ..., detail: _Optional[str] = ..., synced_at: _Optional[str] = ...) -> None: ...

class LogEntry(_message.Message):
    __slots__ = ("id", "type", "plan_id", "execution_id", "phase_id", "title", "detail", "severity", "triage", "sync_status", "downstream", "source_command", "evidence", "attribution_run_id", "idempotency_key", "supersedes_id", "promoted_from_id", "created_at", "updated_at")
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
    def __init__(self, id: _Optional[str] = ..., type: _Optional[_Union[LogEntryType, str]] = ..., plan_id: _Optional[str] = ..., execution_id: _Optional[str] = ..., phase_id: _Optional[str] = ..., title: _Optional[str] = ..., detail: _Optional[str] = ..., severity: _Optional[_Union[LogSeverity, str]] = ..., triage: _Optional[_Union[FindingTriage, str]] = ..., sync_status: _Optional[_Union[LogSyncStatus, str]] = ..., downstream: _Optional[_Union[DownstreamRef, _Mapping]] = ..., source_command: _Optional[str] = ..., evidence: _Optional[_Iterable[str]] = ..., attribution_run_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., supersedes_id: _Optional[str] = ..., promoted_from_id: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("id", "order", "title", "intent", "required_reading", "reminders", "baseline_scope", "acceptance", "status", "last_validation", "references", "relevant_context")
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
    def __init__(self, id: _Optional[str] = ..., order: _Optional[int] = ..., title: _Optional[str] = ..., intent: _Optional[str] = ..., required_reading: _Optional[_Iterable[str]] = ..., reminders: _Optional[_Iterable[str]] = ..., baseline_scope: _Optional[_Iterable[str]] = ..., acceptance: _Optional[str] = ..., status: _Optional[_Union[PhaseStatus, str]] = ..., last_validation: _Optional[_Union[ValidationResult, _Mapping]] = ..., references: _Optional[_Iterable[_Union[Reference, _Mapping]]] = ..., relevant_context: _Optional[_Iterable[_Union[RelevantContextItem, _Mapping]]] = ...) -> None: ...

class Plan(_message.Message):
    __slots__ = ("id", "slug", "title", "status", "content_hash", "created_at", "updated_at", "purpose", "scope", "constraints", "non_goals", "references", "regression_anchor", "definition_of_done", "phases", "supersedes", "superseded_by", "relevant_context")
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
    def __init__(self, id: _Optional[str] = ..., slug: _Optional[str] = ..., title: _Optional[str] = ..., status: _Optional[_Union[PlanStatus, str]] = ..., content_hash: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., purpose: _Optional[str] = ..., scope: _Optional[str] = ..., constraints: _Optional[str] = ..., non_goals: _Optional[str] = ..., references: _Optional[_Iterable[_Union[Reference, _Mapping]]] = ..., regression_anchor: _Optional[_Union[RegressionAnchor, _Mapping]] = ..., definition_of_done: _Optional[str] = ..., phases: _Optional[_Iterable[_Union[Phase, _Mapping]]] = ..., supersedes: _Optional[_Iterable[str]] = ..., superseded_by: _Optional[_Iterable[str]] = ..., relevant_context: _Optional[_Iterable[_Union[RelevantContextItem, _Mapping]]] = ...) -> None: ...

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
    __slots__ = ("id", "execution_id", "plan_id", "completeness", "resume_phase_id", "log_summary", "log_entries", "last_validation", "staleness", "prose_handoff_ref", "assembled_at")
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
    def __init__(self, id: _Optional[str] = ..., execution_id: _Optional[str] = ..., plan_id: _Optional[str] = ..., completeness: _Optional[_Union[Completeness, str]] = ..., resume_phase_id: _Optional[str] = ..., log_summary: _Optional[_Union[LogSummary, _Mapping]] = ..., log_entries: _Optional[_Iterable[_Union[LogEntry, _Mapping]]] = ..., last_validation: _Optional[_Union[ValidationResult, _Mapping]] = ..., staleness: _Optional[_Union[StalenessTier, str]] = ..., prose_handoff_ref: _Optional[str] = ..., assembled_at: _Optional[str] = ...) -> None: ...
