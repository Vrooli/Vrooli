from buf.validate import validate_pb2 as _validate_pb2
from swarm_manager.v1.domain import operating_mode_pb2 as _operating_mode_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AgentOpsCapability(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    AGENT_OPS_CAPABILITY_UNSPECIFIED: _ClassVar[AgentOpsCapability]
    AGENT_OPS_CAPABILITY_PROVIDES_PLAN_REF: _ClassVar[AgentOpsCapability]
    AGENT_OPS_CAPABILITY_PROVIDES_PLAN_CONTEXT: _ClassVar[AgentOpsCapability]
    AGENT_OPS_CAPABILITY_PROVIDES_SPEC_DOCUMENT: _ClassVar[AgentOpsCapability]
    AGENT_OPS_CAPABILITY_PROVIDES_REVIEW_ARTIFACTS: _ClassVar[AgentOpsCapability]
    AGENT_OPS_CAPABILITY_PROVIDES_EVIDENCE_LEDGER: _ClassVar[AgentOpsCapability]
    AGENT_OPS_CAPABILITY_PROVIDES_MEMBER_ITEMS: _ClassVar[AgentOpsCapability]
    AGENT_OPS_CAPABILITY_PROVIDES_ACCEPTANCE_CRITERIA: _ClassVar[AgentOpsCapability]
    AGENT_OPS_CAPABILITY_PROVIDES_CLARIFICATION_THREAD: _ClassVar[AgentOpsCapability]
    AGENT_OPS_CAPABILITY_PROVIDES_EXECUTION_WORKSPACE: _ClassVar[AgentOpsCapability]

class AgentOpsBindingLayer(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    AGENT_OPS_BINDING_LAYER_UNSPECIFIED: _ClassVar[AgentOpsBindingLayer]
    AGENT_OPS_BINDING_LAYER_SYSTEM_DEFAULT: _ClassVar[AgentOpsBindingLayer]
    AGENT_OPS_BINDING_LAYER_INITIATIVE_OVERRIDE: _ClassVar[AgentOpsBindingLayer]
    AGENT_OPS_BINDING_LAYER_BACKLOG_ITEM_OVERRIDE: _ClassVar[AgentOpsBindingLayer]
    AGENT_OPS_BINDING_LAYER_AUTHORIZED_INVOCATION: _ClassVar[AgentOpsBindingLayer]

class AgentOpsWorkflowState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    AGENT_OPS_WORKFLOW_STATE_UNSPECIFIED: _ClassVar[AgentOpsWorkflowState]
    AGENT_OPS_WORKFLOW_STATE_OPEN: _ClassVar[AgentOpsWorkflowState]
    AGENT_OPS_WORKFLOW_STATE_RUNNING: _ClassVar[AgentOpsWorkflowState]
    AGENT_OPS_WORKFLOW_STATE_AWAITING_DECISION: _ClassVar[AgentOpsWorkflowState]
    AGENT_OPS_WORKFLOW_STATE_BLOCKED: _ClassVar[AgentOpsWorkflowState]
    AGENT_OPS_WORKFLOW_STATE_TERMINAL_COMPLETE: _ClassVar[AgentOpsWorkflowState]
    AGENT_OPS_WORKFLOW_STATE_TERMINAL_ABANDONED: _ClassVar[AgentOpsWorkflowState]
    AGENT_OPS_WORKFLOW_STATE_TERMINAL_FAILED: _ClassVar[AgentOpsWorkflowState]

class AgentOpsDomainAction(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    AGENT_OPS_DOMAIN_ACTION_UNSPECIFIED: _ClassVar[AgentOpsDomainAction]
    AGENT_OPS_DOMAIN_ACTION_SAVE_DECISIONS: _ClassVar[AgentOpsDomainAction]
    AGENT_OPS_DOMAIN_ACTION_COMMIT_WORKSHOP_ROUND: _ClassVar[AgentOpsDomainAction]
    AGENT_OPS_DOMAIN_ACTION_START_CLARIFICATION: _ClassVar[AgentOpsDomainAction]
    AGENT_OPS_DOMAIN_ACTION_RESOLVE_CLARIFICATION: _ClassVar[AgentOpsDomainAction]
    AGENT_OPS_DOMAIN_ACTION_BIND_PLAN: _ClassVar[AgentOpsDomainAction]
    AGENT_OPS_DOMAIN_ACTION_QUEUE_PLAN_EXECUTION: _ClassVar[AgentOpsDomainAction]
    AGENT_OPS_DOMAIN_ACTION_START_EXECUTION: _ClassVar[AgentOpsDomainAction]
    AGENT_OPS_DOMAIN_ACTION_COMMIT_REVIEW_ROUND: _ClassVar[AgentOpsDomainAction]
    AGENT_OPS_DOMAIN_ACTION_REQUEST_REVISION: _ClassVar[AgentOpsDomainAction]
    AGENT_OPS_DOMAIN_ACTION_REQUEST_EVIDENCE: _ClassVar[AgentOpsDomainAction]
    AGENT_OPS_DOMAIN_ACTION_COMPLETE_ITEM: _ClassVar[AgentOpsDomainAction]
    AGENT_OPS_DOMAIN_ACTION_FAIL_ITEM: _ClassVar[AgentOpsDomainAction]
    AGENT_OPS_DOMAIN_ACTION_CREATE_FOLLOWUP: _ClassVar[AgentOpsDomainAction]
    AGENT_OPS_DOMAIN_ACTION_OPEN_REVIEW: _ClassVar[AgentOpsDomainAction]
    AGENT_OPS_DOMAIN_ACTION_ESCALATE_NEEDS_ATTENTION: _ClassVar[AgentOpsDomainAction]
    AGENT_OPS_DOMAIN_ACTION_MARK_INITIATIVE_REVIEWED: _ClassVar[AgentOpsDomainAction]

class AgentOpsMemberItemStrategy(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    AGENT_OPS_MEMBER_ITEM_STRATEGY_UNSPECIFIED: _ClassVar[AgentOpsMemberItemStrategy]
    AGENT_OPS_MEMBER_ITEM_STRATEGY_PARALLEL_ITEMS: _ClassVar[AgentOpsMemberItemStrategy]
    AGENT_OPS_MEMBER_ITEM_STRATEGY_SEQUENTIAL_ITEMS: _ClassVar[AgentOpsMemberItemStrategy]
    AGENT_OPS_MEMBER_ITEM_STRATEGY_PRIORITIZED_ITEMS: _ClassVar[AgentOpsMemberItemStrategy]
AGENT_OPS_CAPABILITY_UNSPECIFIED: AgentOpsCapability
AGENT_OPS_CAPABILITY_PROVIDES_PLAN_REF: AgentOpsCapability
AGENT_OPS_CAPABILITY_PROVIDES_PLAN_CONTEXT: AgentOpsCapability
AGENT_OPS_CAPABILITY_PROVIDES_SPEC_DOCUMENT: AgentOpsCapability
AGENT_OPS_CAPABILITY_PROVIDES_REVIEW_ARTIFACTS: AgentOpsCapability
AGENT_OPS_CAPABILITY_PROVIDES_EVIDENCE_LEDGER: AgentOpsCapability
AGENT_OPS_CAPABILITY_PROVIDES_MEMBER_ITEMS: AgentOpsCapability
AGENT_OPS_CAPABILITY_PROVIDES_ACCEPTANCE_CRITERIA: AgentOpsCapability
AGENT_OPS_CAPABILITY_PROVIDES_CLARIFICATION_THREAD: AgentOpsCapability
AGENT_OPS_CAPABILITY_PROVIDES_EXECUTION_WORKSPACE: AgentOpsCapability
AGENT_OPS_BINDING_LAYER_UNSPECIFIED: AgentOpsBindingLayer
AGENT_OPS_BINDING_LAYER_SYSTEM_DEFAULT: AgentOpsBindingLayer
AGENT_OPS_BINDING_LAYER_INITIATIVE_OVERRIDE: AgentOpsBindingLayer
AGENT_OPS_BINDING_LAYER_BACKLOG_ITEM_OVERRIDE: AgentOpsBindingLayer
AGENT_OPS_BINDING_LAYER_AUTHORIZED_INVOCATION: AgentOpsBindingLayer
AGENT_OPS_WORKFLOW_STATE_UNSPECIFIED: AgentOpsWorkflowState
AGENT_OPS_WORKFLOW_STATE_OPEN: AgentOpsWorkflowState
AGENT_OPS_WORKFLOW_STATE_RUNNING: AgentOpsWorkflowState
AGENT_OPS_WORKFLOW_STATE_AWAITING_DECISION: AgentOpsWorkflowState
AGENT_OPS_WORKFLOW_STATE_BLOCKED: AgentOpsWorkflowState
AGENT_OPS_WORKFLOW_STATE_TERMINAL_COMPLETE: AgentOpsWorkflowState
AGENT_OPS_WORKFLOW_STATE_TERMINAL_ABANDONED: AgentOpsWorkflowState
AGENT_OPS_WORKFLOW_STATE_TERMINAL_FAILED: AgentOpsWorkflowState
AGENT_OPS_DOMAIN_ACTION_UNSPECIFIED: AgentOpsDomainAction
AGENT_OPS_DOMAIN_ACTION_SAVE_DECISIONS: AgentOpsDomainAction
AGENT_OPS_DOMAIN_ACTION_COMMIT_WORKSHOP_ROUND: AgentOpsDomainAction
AGENT_OPS_DOMAIN_ACTION_START_CLARIFICATION: AgentOpsDomainAction
AGENT_OPS_DOMAIN_ACTION_RESOLVE_CLARIFICATION: AgentOpsDomainAction
AGENT_OPS_DOMAIN_ACTION_BIND_PLAN: AgentOpsDomainAction
AGENT_OPS_DOMAIN_ACTION_QUEUE_PLAN_EXECUTION: AgentOpsDomainAction
AGENT_OPS_DOMAIN_ACTION_START_EXECUTION: AgentOpsDomainAction
AGENT_OPS_DOMAIN_ACTION_COMMIT_REVIEW_ROUND: AgentOpsDomainAction
AGENT_OPS_DOMAIN_ACTION_REQUEST_REVISION: AgentOpsDomainAction
AGENT_OPS_DOMAIN_ACTION_REQUEST_EVIDENCE: AgentOpsDomainAction
AGENT_OPS_DOMAIN_ACTION_COMPLETE_ITEM: AgentOpsDomainAction
AGENT_OPS_DOMAIN_ACTION_FAIL_ITEM: AgentOpsDomainAction
AGENT_OPS_DOMAIN_ACTION_CREATE_FOLLOWUP: AgentOpsDomainAction
AGENT_OPS_DOMAIN_ACTION_OPEN_REVIEW: AgentOpsDomainAction
AGENT_OPS_DOMAIN_ACTION_ESCALATE_NEEDS_ATTENTION: AgentOpsDomainAction
AGENT_OPS_DOMAIN_ACTION_MARK_INITIATIVE_REVIEWED: AgentOpsDomainAction
AGENT_OPS_MEMBER_ITEM_STRATEGY_UNSPECIFIED: AgentOpsMemberItemStrategy
AGENT_OPS_MEMBER_ITEM_STRATEGY_PARALLEL_ITEMS: AgentOpsMemberItemStrategy
AGENT_OPS_MEMBER_ITEM_STRATEGY_SEQUENTIAL_ITEMS: AgentOpsMemberItemStrategy
AGENT_OPS_MEMBER_ITEM_STRATEGY_PRIORITIZED_ITEMS: AgentOpsMemberItemStrategy

class AgentOpsTargetCapabilityDescriptor(_message.Message):
    __slots__ = ("target_kind", "description", "provides")
    TARGET_KIND_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PROVIDES_FIELD_NUMBER: _ClassVar[int]
    target_kind: _operating_mode_pb2.OperatingModeTargetKind
    description: str
    provides: _containers.RepeatedScalarFieldContainer[AgentOpsCapability]
    def __init__(self, target_kind: _Optional[_Union[_operating_mode_pb2.OperatingModeTargetKind, str]] = ..., description: _Optional[str] = ..., provides: _Optional[_Iterable[_Union[AgentOpsCapability, str]]] = ...) -> None: ...

class AgentOpsCallerInput(_message.Message):
    __slots__ = ("name", "type", "required", "sensitivity", "retention", "description")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    SENSITIVITY_FIELD_NUMBER: _ClassVar[int]
    RETENTION_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    name: str
    type: str
    required: bool
    sensitivity: str
    retention: str
    description: str
    def __init__(self, name: _Optional[str] = ..., type: _Optional[str] = ..., required: _Optional[bool] = ..., sensitivity: _Optional[str] = ..., retention: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class AgentOpsResultField(_message.Message):
    __slots__ = ("name", "type", "required", "enum", "description")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    ENUM_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    name: str
    type: str
    required: bool
    enum: _containers.RepeatedScalarFieldContainer[str]
    description: str
    def __init__(self, name: _Optional[str] = ..., type: _Optional[str] = ..., required: _Optional[bool] = ..., enum: _Optional[_Iterable[str]] = ..., description: _Optional[str] = ...) -> None: ...

class AgentOpsOutcome(_message.Message):
    __slots__ = ("name", "disposition", "description")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DISPOSITION_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    name: str
    disposition: str
    description: str
    def __init__(self, name: _Optional[str] = ..., disposition: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class AgentOpsOperationContract(_message.Message):
    __slots__ = ("id", "version", "summary", "description", "required_capabilities", "inputs", "result_fields", "outcomes")
    ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    INPUTS_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELDS_FIELD_NUMBER: _ClassVar[int]
    OUTCOMES_FIELD_NUMBER: _ClassVar[int]
    id: str
    version: str
    summary: str
    description: str
    required_capabilities: _containers.RepeatedScalarFieldContainer[AgentOpsCapability]
    inputs: _containers.RepeatedCompositeFieldContainer[AgentOpsCallerInput]
    result_fields: _containers.RepeatedCompositeFieldContainer[AgentOpsResultField]
    outcomes: _containers.RepeatedCompositeFieldContainer[AgentOpsOutcome]
    def __init__(self, id: _Optional[str] = ..., version: _Optional[str] = ..., summary: _Optional[str] = ..., description: _Optional[str] = ..., required_capabilities: _Optional[_Iterable[_Union[AgentOpsCapability, str]]] = ..., inputs: _Optional[_Iterable[_Union[AgentOpsCallerInput, _Mapping]]] = ..., result_fields: _Optional[_Iterable[_Union[AgentOpsResultField, _Mapping]]] = ..., outcomes: _Optional[_Iterable[_Union[AgentOpsOutcome, _Mapping]]] = ...) -> None: ...

class AgentOpsBindingOwner(_message.Message):
    __slots__ = ("kind", "id")
    KIND_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    kind: str
    id: str
    def __init__(self, kind: _Optional[str] = ..., id: _Optional[str] = ...) -> None: ...

class AgentOpsOperationBinding(_message.Message):
    __slots__ = ("operation", "operation_version", "layer", "owner", "mode", "mode_revision", "disabled")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    OPERATION_VERSION_FIELD_NUMBER: _ClassVar[int]
    LAYER_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    MODE_REVISION_FIELD_NUMBER: _ClassVar[int]
    DISABLED_FIELD_NUMBER: _ClassVar[int]
    operation: str
    operation_version: str
    layer: AgentOpsBindingLayer
    owner: AgentOpsBindingOwner
    mode: str
    mode_revision: str
    disabled: bool
    def __init__(self, operation: _Optional[str] = ..., operation_version: _Optional[str] = ..., layer: _Optional[_Union[AgentOpsBindingLayer, str]] = ..., owner: _Optional[_Union[AgentOpsBindingOwner, _Mapping]] = ..., mode: _Optional[str] = ..., mode_revision: _Optional[str] = ..., disabled: _Optional[bool] = ...) -> None: ...

class AgentOpsProvenanceBinding(_message.Message):
    __slots__ = ("layer", "owner_kind", "owner_id")
    LAYER_FIELD_NUMBER: _ClassVar[int]
    OWNER_KIND_FIELD_NUMBER: _ClassVar[int]
    OWNER_ID_FIELD_NUMBER: _ClassVar[int]
    layer: AgentOpsBindingLayer
    owner_kind: str
    owner_id: str
    def __init__(self, layer: _Optional[_Union[AgentOpsBindingLayer, str]] = ..., owner_kind: _Optional[str] = ..., owner_id: _Optional[str] = ...) -> None: ...

class AgentOpsProvenanceTarget(_message.Message):
    __slots__ = ("kind", "id")
    KIND_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    kind: _operating_mode_pb2.OperatingModeTargetKind
    id: str
    def __init__(self, kind: _Optional[_Union[_operating_mode_pb2.OperatingModeTargetKind, str]] = ..., id: _Optional[str] = ...) -> None: ...

class AgentOpsExecutionProvenance(_message.Message):
    __slots__ = ("operation", "operation_version", "binding", "mode", "mode_revision", "compiled_mode_digest", "prompt_catalog_revision", "prompt_catalog_digest", "target", "caller_input_digest", "policy_revision", "workflow_instance_id")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    OPERATION_VERSION_FIELD_NUMBER: _ClassVar[int]
    BINDING_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    MODE_REVISION_FIELD_NUMBER: _ClassVar[int]
    COMPILED_MODE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    PROMPT_CATALOG_REVISION_FIELD_NUMBER: _ClassVar[int]
    PROMPT_CATALOG_DIGEST_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    CALLER_INPUT_DIGEST_FIELD_NUMBER: _ClassVar[int]
    POLICY_REVISION_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_INSTANCE_ID_FIELD_NUMBER: _ClassVar[int]
    operation: str
    operation_version: str
    binding: AgentOpsProvenanceBinding
    mode: str
    mode_revision: str
    compiled_mode_digest: str
    prompt_catalog_revision: str
    prompt_catalog_digest: str
    target: AgentOpsProvenanceTarget
    caller_input_digest: str
    policy_revision: str
    workflow_instance_id: str
    def __init__(self, operation: _Optional[str] = ..., operation_version: _Optional[str] = ..., binding: _Optional[_Union[AgentOpsProvenanceBinding, _Mapping]] = ..., mode: _Optional[str] = ..., mode_revision: _Optional[str] = ..., compiled_mode_digest: _Optional[str] = ..., prompt_catalog_revision: _Optional[str] = ..., prompt_catalog_digest: _Optional[str] = ..., target: _Optional[_Union[AgentOpsProvenanceTarget, _Mapping]] = ..., caller_input_digest: _Optional[str] = ..., policy_revision: _Optional[str] = ..., workflow_instance_id: _Optional[str] = ...) -> None: ...

class AgentOpsOperationExecutionRecord(_message.Message):
    __slots__ = ("operation", "execution_id", "idempotency_key", "provenance_digest", "state", "outcome", "run_id")
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_DIGEST_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    operation: str
    execution_id: str
    idempotency_key: str
    provenance_digest: str
    state: str
    outcome: str
    run_id: str
    def __init__(self, operation: _Optional[str] = ..., execution_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., provenance_digest: _Optional[str] = ..., state: _Optional[str] = ..., outcome: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class AgentOpsHumanDecision(_message.Message):
    __slots__ = ("decision", "actor", "at_version", "note")
    DECISION_FIELD_NUMBER: _ClassVar[int]
    ACTOR_FIELD_NUMBER: _ClassVar[int]
    AT_VERSION_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    decision: str
    actor: str
    at_version: int
    note: str
    def __init__(self, decision: _Optional[str] = ..., actor: _Optional[str] = ..., at_version: _Optional[int] = ..., note: _Optional[str] = ...) -> None: ...

class AgentOpsScheduledIntent(_message.Message):
    __slots__ = ("intent", "action", "not_before")
    INTENT_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    NOT_BEFORE_FIELD_NUMBER: _ClassVar[int]
    intent: str
    action: AgentOpsDomainAction
    not_before: str
    def __init__(self, intent: _Optional[str] = ..., action: _Optional[_Union[AgentOpsDomainAction, str]] = ..., not_before: _Optional[str] = ...) -> None: ...

class AgentOpsMemberItemStrategyConfig(_message.Message):
    __slots__ = ("strategy", "item_operation", "max_concurrency", "stop_on_first_failure", "item_selection")
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    ITEM_OPERATION_FIELD_NUMBER: _ClassVar[int]
    MAX_CONCURRENCY_FIELD_NUMBER: _ClassVar[int]
    STOP_ON_FIRST_FAILURE_FIELD_NUMBER: _ClassVar[int]
    ITEM_SELECTION_FIELD_NUMBER: _ClassVar[int]
    strategy: AgentOpsMemberItemStrategy
    item_operation: str
    max_concurrency: int
    stop_on_first_failure: bool
    item_selection: str
    def __init__(self, strategy: _Optional[_Union[AgentOpsMemberItemStrategy, str]] = ..., item_operation: _Optional[str] = ..., max_concurrency: _Optional[int] = ..., stop_on_first_failure: _Optional[bool] = ..., item_selection: _Optional[str] = ...) -> None: ...

class AgentOpsWorkflowInstance(_message.Message):
    __slots__ = ("schema_version", "instance_id", "domain_kind", "domain_id", "strategy", "state", "operations", "decisions", "timers", "legal_actions", "idempotency_keys", "version")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    INSTANCE_ID_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_KIND_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_ID_FIELD_NUMBER: _ClassVar[int]
    STRATEGY_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    DECISIONS_FIELD_NUMBER: _ClassVar[int]
    TIMERS_FIELD_NUMBER: _ClassVar[int]
    LEGAL_ACTIONS_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEYS_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    schema_version: str
    instance_id: str
    domain_kind: str
    domain_id: str
    strategy: AgentOpsMemberItemStrategyConfig
    state: AgentOpsWorkflowState
    operations: _containers.RepeatedCompositeFieldContainer[AgentOpsOperationExecutionRecord]
    decisions: _containers.RepeatedCompositeFieldContainer[AgentOpsHumanDecision]
    timers: _containers.RepeatedCompositeFieldContainer[AgentOpsScheduledIntent]
    legal_actions: _containers.RepeatedScalarFieldContainer[AgentOpsDomainAction]
    idempotency_keys: _containers.RepeatedScalarFieldContainer[str]
    version: int
    def __init__(self, schema_version: _Optional[str] = ..., instance_id: _Optional[str] = ..., domain_kind: _Optional[str] = ..., domain_id: _Optional[str] = ..., strategy: _Optional[_Union[AgentOpsMemberItemStrategyConfig, _Mapping]] = ..., state: _Optional[_Union[AgentOpsWorkflowState, str]] = ..., operations: _Optional[_Iterable[_Union[AgentOpsOperationExecutionRecord, _Mapping]]] = ..., decisions: _Optional[_Iterable[_Union[AgentOpsHumanDecision, _Mapping]]] = ..., timers: _Optional[_Iterable[_Union[AgentOpsScheduledIntent, _Mapping]]] = ..., legal_actions: _Optional[_Iterable[_Union[AgentOpsDomainAction, str]]] = ..., idempotency_keys: _Optional[_Iterable[str]] = ..., version: _Optional[int] = ...) -> None: ...

class AgentOpsPolicyTransition(_message.Message):
    __slots__ = ("from_state", "on_outcome", "action", "to_state")
    FROM_STATE_FIELD_NUMBER: _ClassVar[int]
    ON_OUTCOME_FIELD_NUMBER: _ClassVar[int]
    ACTION_FIELD_NUMBER: _ClassVar[int]
    TO_STATE_FIELD_NUMBER: _ClassVar[int]
    from_state: AgentOpsWorkflowState
    on_outcome: str
    action: AgentOpsDomainAction
    to_state: AgentOpsWorkflowState
    def __init__(self, from_state: _Optional[_Union[AgentOpsWorkflowState, str]] = ..., on_outcome: _Optional[str] = ..., action: _Optional[_Union[AgentOpsDomainAction, str]] = ..., to_state: _Optional[_Union[AgentOpsWorkflowState, str]] = ...) -> None: ...

class AgentOpsTransitionPolicy(_message.Message):
    __slots__ = ("id", "version", "domain_kind", "transitions")
    ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_KIND_FIELD_NUMBER: _ClassVar[int]
    TRANSITIONS_FIELD_NUMBER: _ClassVar[int]
    id: str
    version: str
    domain_kind: str
    transitions: _containers.RepeatedCompositeFieldContainer[AgentOpsPolicyTransition]
    def __init__(self, id: _Optional[str] = ..., version: _Optional[str] = ..., domain_kind: _Optional[str] = ..., transitions: _Optional[_Iterable[_Union[AgentOpsPolicyTransition, _Mapping]]] = ...) -> None: ...
