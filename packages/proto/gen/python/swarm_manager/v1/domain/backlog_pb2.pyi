from buf.validate import validate_pb2 as _validate_pb2
from swarm_manager.v1.shared import agent_session_pb2 as _agent_session_pb2
from swarm_manager.v1.shared import backlog_pb2 as _backlog_pb2
from swarm_manager.v1.shared import plan_ref_pb2 as _plan_ref_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class BacklogItem(_message.Message):
    __slots__ = ("name", "title", "description", "status", "priority", "tags", "created", "updated", "kind", "depends_on", "milestone", "effort", "acceptance_allow", "acceptance_deny", "spawned_from", "note", "archived_at", "suggested_skills", "creates", "created_by", "queue_position", "plan_ref", "finding_ref", "stale", "last_review", "plan_acceptance", "acceptance_criteria", "execution_strategy", "execution_limits", "continuation", "scope_policy", "continuation_halted_at", "continuation_halted_by", "continuation_stopped_reason")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    UPDATED_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    DEPENDS_ON_FIELD_NUMBER: _ClassVar[int]
    MILESTONE_FIELD_NUMBER: _ClassVar[int]
    EFFORT_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_ALLOW_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_DENY_FIELD_NUMBER: _ClassVar[int]
    SPAWNED_FROM_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_AT_FIELD_NUMBER: _ClassVar[int]
    SUGGESTED_SKILLS_FIELD_NUMBER: _ClassVar[int]
    CREATES_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    QUEUE_POSITION_FIELD_NUMBER: _ClassVar[int]
    PLAN_REF_FIELD_NUMBER: _ClassVar[int]
    FINDING_REF_FIELD_NUMBER: _ClassVar[int]
    STALE_FIELD_NUMBER: _ClassVar[int]
    LAST_REVIEW_FIELD_NUMBER: _ClassVar[int]
    PLAN_ACCEPTANCE_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_CRITERIA_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_STRATEGY_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_LIMITS_FIELD_NUMBER: _ClassVar[int]
    CONTINUATION_FIELD_NUMBER: _ClassVar[int]
    SCOPE_POLICY_FIELD_NUMBER: _ClassVar[int]
    CONTINUATION_HALTED_AT_FIELD_NUMBER: _ClassVar[int]
    CONTINUATION_HALTED_BY_FIELD_NUMBER: _ClassVar[int]
    CONTINUATION_STOPPED_REASON_FIELD_NUMBER: _ClassVar[int]
    name: str
    title: str
    description: str
    status: str
    priority: int
    tags: _containers.RepeatedScalarFieldContainer[str]
    created: str
    updated: str
    kind: str
    depends_on: _containers.RepeatedScalarFieldContainer[str]
    milestone: str
    effort: str
    acceptance_allow: _containers.RepeatedScalarFieldContainer[str]
    acceptance_deny: _containers.RepeatedScalarFieldContainer[str]
    spawned_from: str
    note: str
    archived_at: str
    suggested_skills: _containers.RepeatedScalarFieldContainer[str]
    creates: _containers.RepeatedScalarFieldContainer[str]
    created_by: _agent_session_pb2.AgentSessionAttribution
    queue_position: int
    plan_ref: _plan_ref_pb2.PlanRef
    finding_ref: str
    stale: bool
    last_review: BacklogReviewRecord
    plan_acceptance: PlanAcceptance
    acceptance_criteria: _containers.RepeatedCompositeFieldContainer[_backlog_pb2.BacklogCriterion]
    execution_strategy: str
    execution_limits: ExecutionLimits
    continuation: str
    scope_policy: str
    continuation_halted_at: str
    continuation_halted_by: str
    continuation_stopped_reason: str
    def __init__(self, name: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ..., priority: _Optional[int] = ..., tags: _Optional[_Iterable[str]] = ..., created: _Optional[str] = ..., updated: _Optional[str] = ..., kind: _Optional[str] = ..., depends_on: _Optional[_Iterable[str]] = ..., milestone: _Optional[str] = ..., effort: _Optional[str] = ..., acceptance_allow: _Optional[_Iterable[str]] = ..., acceptance_deny: _Optional[_Iterable[str]] = ..., spawned_from: _Optional[str] = ..., note: _Optional[str] = ..., archived_at: _Optional[str] = ..., suggested_skills: _Optional[_Iterable[str]] = ..., creates: _Optional[_Iterable[str]] = ..., created_by: _Optional[_Union[_agent_session_pb2.AgentSessionAttribution, _Mapping]] = ..., queue_position: _Optional[int] = ..., plan_ref: _Optional[_Union[_plan_ref_pb2.PlanRef, _Mapping]] = ..., finding_ref: _Optional[str] = ..., stale: _Optional[bool] = ..., last_review: _Optional[_Union[BacklogReviewRecord, _Mapping]] = ..., plan_acceptance: _Optional[_Union[PlanAcceptance, _Mapping]] = ..., acceptance_criteria: _Optional[_Iterable[_Union[_backlog_pb2.BacklogCriterion, _Mapping]]] = ..., execution_strategy: _Optional[str] = ..., execution_limits: _Optional[_Union[ExecutionLimits, _Mapping]] = ..., continuation: _Optional[str] = ..., scope_policy: _Optional[str] = ..., continuation_halted_at: _Optional[str] = ..., continuation_halted_by: _Optional[str] = ..., continuation_stopped_reason: _Optional[str] = ...) -> None: ...

class ExecutionLimits(_message.Message):
    __slots__ = ("max_slices", "max_tokens", "max_wall_seconds", "max_turns", "max_charge_micro_usd", "max_children", "max_node_attempts", "max_retries")
    MAX_SLICES_FIELD_NUMBER: _ClassVar[int]
    MAX_TOKENS_FIELD_NUMBER: _ClassVar[int]
    MAX_WALL_SECONDS_FIELD_NUMBER: _ClassVar[int]
    MAX_TURNS_FIELD_NUMBER: _ClassVar[int]
    MAX_CHARGE_MICRO_USD_FIELD_NUMBER: _ClassVar[int]
    MAX_CHILDREN_FIELD_NUMBER: _ClassVar[int]
    MAX_NODE_ATTEMPTS_FIELD_NUMBER: _ClassVar[int]
    MAX_RETRIES_FIELD_NUMBER: _ClassVar[int]
    max_slices: int
    max_tokens: int
    max_wall_seconds: int
    max_turns: int
    max_charge_micro_usd: int
    max_children: int
    max_node_attempts: int
    max_retries: int
    def __init__(self, max_slices: _Optional[int] = ..., max_tokens: _Optional[int] = ..., max_wall_seconds: _Optional[int] = ..., max_turns: _Optional[int] = ..., max_charge_micro_usd: _Optional[int] = ..., max_children: _Optional[int] = ..., max_node_attempts: _Optional[int] = ..., max_retries: _Optional[int] = ...) -> None: ...

class ExecutionPreferences(_message.Message):
    __slots__ = ("preferred_runner", "model", "effort")
    PREFERRED_RUNNER_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    EFFORT_FIELD_NUMBER: _ClassVar[int]
    preferred_runner: str
    model: str
    effort: str
    def __init__(self, preferred_runner: _Optional[str] = ..., model: _Optional[str] = ..., effort: _Optional[str] = ...) -> None: ...

class PlanAcceptance(_message.Message):
    __slots__ = ("actor", "accepted_at", "plan_content_hash", "subject_version")
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_AT_FIELD_NUMBER: _ClassVar[int]
    PLAN_CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    SUBJECT_VERSION_FIELD_NUMBER: _ClassVar[int]
    actor: str
    accepted_at: str
    plan_content_hash: str
    subject_version: str
    def __init__(self, actor: _Optional[str] = ..., accepted_at: _Optional[str] = ..., plan_content_hash: _Optional[str] = ..., subject_version: _Optional[str] = ...) -> None: ...

class BacklogReviewRecord(_message.Message):
    __slots__ = ("reviewed_at", "session_id", "proposal_id", "rationale")
    REVIEWED_AT_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    PROPOSAL_ID_FIELD_NUMBER: _ClassVar[int]
    RATIONALE_FIELD_NUMBER: _ClassVar[int]
    reviewed_at: str
    session_id: str
    proposal_id: str
    rationale: str
    def __init__(self, reviewed_at: _Optional[str] = ..., session_id: _Optional[str] = ..., proposal_id: _Optional[str] = ..., rationale: _Optional[str] = ...) -> None: ...

class ClarificationMessage(_message.Message):
    __slots__ = ("role", "content", "created_at", "attachment_ids")
    ROLE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENT_IDS_FIELD_NUMBER: _ClassVar[int]
    role: str
    content: str
    created_at: str
    attachment_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, role: _Optional[str] = ..., content: _Optional[str] = ..., created_at: _Optional[str] = ..., attachment_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class ClarificationImpact(_message.Message):
    __slots__ = ("level", "reasoning", "context_note", "suggested_update")
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    REASONING_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_NOTE_FIELD_NUMBER: _ClassVar[int]
    SUGGESTED_UPDATE_FIELD_NUMBER: _ClassVar[int]
    level: str
    reasoning: str
    context_note: str
    suggested_update: str
    def __init__(self, level: _Optional[str] = ..., reasoning: _Optional[str] = ..., context_note: _Optional[str] = ..., suggested_update: _Optional[str] = ...) -> None: ...

class ClarificationThread(_message.Message):
    __slots__ = ("id", "round_number", "item_id", "run_id", "messages", "latest_impact", "status", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    ROUND_NUMBER_FIELD_NUMBER: _ClassVar[int]
    ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    MESSAGES_FIELD_NUMBER: _ClassVar[int]
    LATEST_IMPACT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    round_number: int
    item_id: str
    run_id: str
    messages: _containers.RepeatedCompositeFieldContainer[ClarificationMessage]
    latest_impact: ClarificationImpact
    status: str
    created_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., round_number: _Optional[int] = ..., item_id: _Optional[str] = ..., run_id: _Optional[str] = ..., messages: _Optional[_Iterable[_Union[ClarificationMessage, _Mapping]]] = ..., latest_impact: _Optional[_Union[ClarificationImpact, _Mapping]] = ..., status: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...
