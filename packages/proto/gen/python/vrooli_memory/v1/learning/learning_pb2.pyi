from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AdviceUse(_message.Message):
    __slots__ = ("entry_id", "decision", "decision_change", "verdict", "evidence_refs")
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    DECISION_FIELD_NUMBER: _ClassVar[int]
    DECISION_CHANGE_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    entry_id: str
    decision: str
    decision_change: str
    verdict: str
    evidence_refs: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, entry_id: _Optional[str] = ..., decision: _Optional[str] = ..., decision_change: _Optional[str] = ..., verdict: _Optional[str] = ..., evidence_refs: _Optional[_Iterable[str]] = ...) -> None: ...

class Attempt(_message.Message):
    __slots__ = ("attempt_id", "task_id", "operation", "context_key", "started_at", "finished_at", "outcome", "failure_fingerprint", "evidence_refs", "advice", "recall_status", "provenance", "trigger", "approach", "task_started_at", "attempt_number", "first_action_at", "tool_round_trips", "visual_reasoning_calls", "reused_workflow")
    ATTEMPT_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_KEY_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    FINISHED_AT_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    FAILURE_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    ADVICE_FIELD_NUMBER: _ClassVar[int]
    RECALL_STATUS_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_FIELD_NUMBER: _ClassVar[int]
    APPROACH_FIELD_NUMBER: _ClassVar[int]
    TASK_STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    ATTEMPT_NUMBER_FIELD_NUMBER: _ClassVar[int]
    FIRST_ACTION_AT_FIELD_NUMBER: _ClassVar[int]
    TOOL_ROUND_TRIPS_FIELD_NUMBER: _ClassVar[int]
    VISUAL_REASONING_CALLS_FIELD_NUMBER: _ClassVar[int]
    REUSED_WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    attempt_id: str
    task_id: str
    operation: str
    context_key: str
    started_at: str
    finished_at: str
    outcome: str
    failure_fingerprint: str
    evidence_refs: _containers.RepeatedScalarFieldContainer[str]
    advice: _containers.RepeatedCompositeFieldContainer[AdviceUse]
    recall_status: str
    provenance: str
    trigger: str
    approach: str
    task_started_at: str
    attempt_number: int
    first_action_at: str
    tool_round_trips: int
    visual_reasoning_calls: int
    reused_workflow: bool
    def __init__(self, attempt_id: _Optional[str] = ..., task_id: _Optional[str] = ..., operation: _Optional[str] = ..., context_key: _Optional[str] = ..., started_at: _Optional[str] = ..., finished_at: _Optional[str] = ..., outcome: _Optional[str] = ..., failure_fingerprint: _Optional[str] = ..., evidence_refs: _Optional[_Iterable[str]] = ..., advice: _Optional[_Iterable[_Union[AdviceUse, _Mapping]]] = ..., recall_status: _Optional[str] = ..., provenance: _Optional[str] = ..., trigger: _Optional[str] = ..., approach: _Optional[str] = ..., task_started_at: _Optional[str] = ..., attempt_number: _Optional[int] = ..., first_action_at: _Optional[str] = ..., tool_round_trips: _Optional[int] = ..., visual_reasoning_calls: _Optional[int] = ..., reused_workflow: _Optional[bool] = ...) -> None: ...

class RecordAttemptRequest(_message.Message):
    __slots__ = ("scope", "attempt")
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    ATTEMPT_FIELD_NUMBER: _ClassVar[int]
    scope: str
    attempt: Attempt
    def __init__(self, scope: _Optional[str] = ..., attempt: _Optional[_Union[Attempt, _Mapping]] = ...) -> None: ...

class RecordAttemptResponse(_message.Message):
    __slots__ = ("entry_id", "existing")
    ENTRY_ID_FIELD_NUMBER: _ClassVar[int]
    EXISTING_FIELD_NUMBER: _ClassVar[int]
    entry_id: str
    existing: bool
    def __init__(self, entry_id: _Optional[str] = ..., existing: _Optional[bool] = ...) -> None: ...

class MeasureLearningRequest(_message.Message):
    __slots__ = ("scope", "to", "operation", "context_key")
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_KEY_FIELD_NUMBER: _ClassVar[int]
    scope: str
    to: str
    operation: str
    context_key: str
    def __init__(self, scope: _Optional[str] = ..., to: _Optional[str] = ..., operation: _Optional[str] = ..., context_key: _Optional[str] = ..., **kwargs) -> None: ...

class Cohort(_message.Message):
    __slots__ = ("operation", "context_key", "attempts", "tasks", "verified_successes", "failed", "unavailable", "unknown", "recurring_failure_fingerprints", "repeated_failures", "applied_advice", "rejected_advice", "supported_advice", "contradicted_advice", "unassessed_advice", "contradiction_rate", "completed_tasks", "unresolved_tasks", "median_attempts_to_success", "median_seconds_to_success", "left_censored_tasks", "no_match", "recall_unavailable", "median_seconds_to_first_action", "median_tool_round_trips", "median_visual_reasoning_calls", "workflow_reuse_rate", "first_action_samples", "tool_round_trip_samples", "visual_reasoning_samples", "reuse_samples")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_KEY_FIELD_NUMBER: _ClassVar[int]
    ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    TASKS_FIELD_NUMBER: _ClassVar[int]
    VERIFIED_SUCCESSES_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_FIELD_NUMBER: _ClassVar[int]
    UNKNOWN_FIELD_NUMBER: _ClassVar[int]
    RECURRING_FAILURE_FINGERPRINTS_FIELD_NUMBER: _ClassVar[int]
    REPEATED_FAILURES_FIELD_NUMBER: _ClassVar[int]
    APPLIED_ADVICE_FIELD_NUMBER: _ClassVar[int]
    REJECTED_ADVICE_FIELD_NUMBER: _ClassVar[int]
    SUPPORTED_ADVICE_FIELD_NUMBER: _ClassVar[int]
    CONTRADICTED_ADVICE_FIELD_NUMBER: _ClassVar[int]
    UNASSESSED_ADVICE_FIELD_NUMBER: _ClassVar[int]
    CONTRADICTION_RATE_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_TASKS_FIELD_NUMBER: _ClassVar[int]
    UNRESOLVED_TASKS_FIELD_NUMBER: _ClassVar[int]
    MEDIAN_ATTEMPTS_TO_SUCCESS_FIELD_NUMBER: _ClassVar[int]
    MEDIAN_SECONDS_TO_SUCCESS_FIELD_NUMBER: _ClassVar[int]
    LEFT_CENSORED_TASKS_FIELD_NUMBER: _ClassVar[int]
    NO_MATCH_FIELD_NUMBER: _ClassVar[int]
    RECALL_UNAVAILABLE_FIELD_NUMBER: _ClassVar[int]
    MEDIAN_SECONDS_TO_FIRST_ACTION_FIELD_NUMBER: _ClassVar[int]
    MEDIAN_TOOL_ROUND_TRIPS_FIELD_NUMBER: _ClassVar[int]
    MEDIAN_VISUAL_REASONING_CALLS_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_REUSE_RATE_FIELD_NUMBER: _ClassVar[int]
    FIRST_ACTION_SAMPLES_FIELD_NUMBER: _ClassVar[int]
    TOOL_ROUND_TRIP_SAMPLES_FIELD_NUMBER: _ClassVar[int]
    VISUAL_REASONING_SAMPLES_FIELD_NUMBER: _ClassVar[int]
    REUSE_SAMPLES_FIELD_NUMBER: _ClassVar[int]
    operation: str
    context_key: str
    attempts: int
    tasks: int
    verified_successes: int
    failed: int
    unavailable: int
    unknown: int
    recurring_failure_fingerprints: int
    repeated_failures: int
    applied_advice: int
    rejected_advice: int
    supported_advice: int
    contradicted_advice: int
    unassessed_advice: int
    contradiction_rate: float
    completed_tasks: int
    unresolved_tasks: int
    median_attempts_to_success: float
    median_seconds_to_success: float
    left_censored_tasks: int
    no_match: int
    recall_unavailable: int
    median_seconds_to_first_action: float
    median_tool_round_trips: float
    median_visual_reasoning_calls: float
    workflow_reuse_rate: float
    first_action_samples: int
    tool_round_trip_samples: int
    visual_reasoning_samples: int
    reuse_samples: int
    def __init__(self, operation: _Optional[str] = ..., context_key: _Optional[str] = ..., attempts: _Optional[int] = ..., tasks: _Optional[int] = ..., verified_successes: _Optional[int] = ..., failed: _Optional[int] = ..., unavailable: _Optional[int] = ..., unknown: _Optional[int] = ..., recurring_failure_fingerprints: _Optional[int] = ..., repeated_failures: _Optional[int] = ..., applied_advice: _Optional[int] = ..., rejected_advice: _Optional[int] = ..., supported_advice: _Optional[int] = ..., contradicted_advice: _Optional[int] = ..., unassessed_advice: _Optional[int] = ..., contradiction_rate: _Optional[float] = ..., completed_tasks: _Optional[int] = ..., unresolved_tasks: _Optional[int] = ..., median_attempts_to_success: _Optional[float] = ..., median_seconds_to_success: _Optional[float] = ..., left_censored_tasks: _Optional[int] = ..., no_match: _Optional[int] = ..., recall_unavailable: _Optional[int] = ..., median_seconds_to_first_action: _Optional[float] = ..., median_tool_round_trips: _Optional[float] = ..., median_visual_reasoning_calls: _Optional[float] = ..., workflow_reuse_rate: _Optional[float] = ..., first_action_samples: _Optional[int] = ..., tool_round_trip_samples: _Optional[int] = ..., visual_reasoning_samples: _Optional[int] = ..., reuse_samples: _Optional[int] = ...) -> None: ...

class MeasureLearningResponse(_message.Message):
    __slots__ = ("scope", "to", "cohorts", "scanned_entries", "eligible_attempts", "excluded_test_attempts", "legacy_task_records", "invalid_records", "duplicate_attempts", "truncated", "reliable", "reason", "evidence_refs", "interpretation")
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    COHORTS_FIELD_NUMBER: _ClassVar[int]
    SCANNED_ENTRIES_FIELD_NUMBER: _ClassVar[int]
    ELIGIBLE_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    EXCLUDED_TEST_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    LEGACY_TASK_RECORDS_FIELD_NUMBER: _ClassVar[int]
    INVALID_RECORDS_FIELD_NUMBER: _ClassVar[int]
    DUPLICATE_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    RELIABLE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    INTERPRETATION_FIELD_NUMBER: _ClassVar[int]
    scope: str
    to: str
    cohorts: _containers.RepeatedCompositeFieldContainer[Cohort]
    scanned_entries: int
    eligible_attempts: int
    excluded_test_attempts: int
    legacy_task_records: int
    invalid_records: int
    duplicate_attempts: int
    truncated: bool
    reliable: bool
    reason: str
    evidence_refs: _containers.RepeatedScalarFieldContainer[str]
    interpretation: str
    def __init__(self, scope: _Optional[str] = ..., to: _Optional[str] = ..., cohorts: _Optional[_Iterable[_Union[Cohort, _Mapping]]] = ..., scanned_entries: _Optional[int] = ..., eligible_attempts: _Optional[int] = ..., excluded_test_attempts: _Optional[int] = ..., legacy_task_records: _Optional[int] = ..., invalid_records: _Optional[int] = ..., duplicate_attempts: _Optional[int] = ..., truncated: _Optional[bool] = ..., reliable: _Optional[bool] = ..., reason: _Optional[str] = ..., evidence_refs: _Optional[_Iterable[str]] = ..., interpretation: _Optional[str] = ..., **kwargs) -> None: ...
