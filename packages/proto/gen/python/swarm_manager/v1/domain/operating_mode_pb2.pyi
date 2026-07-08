from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class OperatingModeScopeKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OPERATING_MODE_SCOPE_KIND_UNSPECIFIED: _ClassVar[OperatingModeScopeKind]
    OPERATING_MODE_SCOPE_KIND_BACKLOG_ITEM: _ClassVar[OperatingModeScopeKind]
    OPERATING_MODE_SCOPE_KIND_INITIATIVE: _ClassVar[OperatingModeScopeKind]

class OperatingModeRunStrategyKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OPERATING_MODE_RUN_STRATEGY_KIND_UNSPECIFIED: _ClassVar[OperatingModeRunStrategyKind]
    OPERATING_MODE_RUN_STRATEGY_KIND_EXISTING_ITEM_FLOW: _ClassVar[OperatingModeRunStrategyKind]
    OPERATING_MODE_RUN_STRATEGY_KIND_SINGLE_PHASE_RUN: _ClassVar[OperatingModeRunStrategyKind]
    OPERATING_MODE_RUN_STRATEGY_KIND_SEQUENTIAL_HANDOFF: _ClassVar[OperatingModeRunStrategyKind]
    OPERATING_MODE_RUN_STRATEGY_KIND_OPERATOR_GATED_LOOP: _ClassVar[OperatingModeRunStrategyKind]

class OperatingModePhaseKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OPERATING_MODE_PHASE_KIND_UNSPECIFIED: _ClassVar[OperatingModePhaseKind]
    OPERATING_MODE_PHASE_KIND_INVESTIGATE: _ClassVar[OperatingModePhaseKind]
    OPERATING_MODE_PHASE_KIND_EXECUTE: _ClassVar[OperatingModePhaseKind]
    OPERATING_MODE_PHASE_KIND_REVIEW: _ClassVar[OperatingModePhaseKind]
    OPERATING_MODE_PHASE_KIND_RECONCILE: _ClassVar[OperatingModePhaseKind]

class OperatingModeGuardOp(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OPERATING_MODE_GUARD_OP_UNSPECIFIED: _ClassVar[OperatingModeGuardOp]
    OPERATING_MODE_GUARD_OP_ALWAYS: _ClassVar[OperatingModeGuardOp]
    OPERATING_MODE_GUARD_OP_EQ: _ClassVar[OperatingModeGuardOp]
    OPERATING_MODE_GUARD_OP_NE: _ClassVar[OperatingModeGuardOp]
    OPERATING_MODE_GUARD_OP_GT: _ClassVar[OperatingModeGuardOp]
    OPERATING_MODE_GUARD_OP_GTE: _ClassVar[OperatingModeGuardOp]
    OPERATING_MODE_GUARD_OP_LT: _ClassVar[OperatingModeGuardOp]
    OPERATING_MODE_GUARD_OP_LTE: _ClassVar[OperatingModeGuardOp]
    OPERATING_MODE_GUARD_OP_IN: _ClassVar[OperatingModeGuardOp]
    OPERATING_MODE_GUARD_OP_NOT_IN: _ClassVar[OperatingModeGuardOp]
    OPERATING_MODE_GUARD_OP_EXISTS: _ClassVar[OperatingModeGuardOp]
    OPERATING_MODE_GUARD_OP_NOT_EXISTS: _ClassVar[OperatingModeGuardOp]
    OPERATING_MODE_GUARD_OP_ALL: _ClassVar[OperatingModeGuardOp]
    OPERATING_MODE_GUARD_OP_ANY: _ClassVar[OperatingModeGuardOp]
    OPERATING_MODE_GUARD_OP_NOT: _ClassVar[OperatingModeGuardOp]

class OperatingModeOutputFieldType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OPERATING_MODE_OUTPUT_FIELD_TYPE_UNSPECIFIED: _ClassVar[OperatingModeOutputFieldType]
    OPERATING_MODE_OUTPUT_FIELD_TYPE_STRING: _ClassVar[OperatingModeOutputFieldType]
    OPERATING_MODE_OUTPUT_FIELD_TYPE_BOOLEAN: _ClassVar[OperatingModeOutputFieldType]
    OPERATING_MODE_OUTPUT_FIELD_TYPE_INTEGER: _ClassVar[OperatingModeOutputFieldType]
    OPERATING_MODE_OUTPUT_FIELD_TYPE_NUMBER: _ClassVar[OperatingModeOutputFieldType]
    OPERATING_MODE_OUTPUT_FIELD_TYPE_OBJECT: _ClassVar[OperatingModeOutputFieldType]
    OPERATING_MODE_OUTPUT_FIELD_TYPE_ARRAY: _ClassVar[OperatingModeOutputFieldType]

class OperatingModeBacklogCapability(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OPERATING_MODE_BACKLOG_CAPABILITY_UNSPECIFIED: _ClassVar[OperatingModeBacklogCapability]
    OPERATING_MODE_BACKLOG_CAPABILITY_READ_ONLY: _ClassVar[OperatingModeBacklogCapability]
    OPERATING_MODE_BACKLOG_CAPABILITY_PROPOSE_MUTATIONS: _ClassVar[OperatingModeBacklogCapability]
    OPERATING_MODE_BACKLOG_CAPABILITY_MARK_COMPLETE: _ClassVar[OperatingModeBacklogCapability]
    OPERATING_MODE_BACKLOG_CAPABILITY_CREATE_FOLLOWUPS: _ClassVar[OperatingModeBacklogCapability]
    OPERATING_MODE_BACKLOG_CAPABILITY_UPDATE_SCOPE: _ClassVar[OperatingModeBacklogCapability]

class OperatingModeApplyMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OPERATING_MODE_APPLY_MODE_UNSPECIFIED: _ClassVar[OperatingModeApplyMode]
    OPERATING_MODE_APPLY_MODE_OPERATOR_GATED: _ClassVar[OperatingModeApplyMode]
    OPERATING_MODE_APPLY_MODE_AUTO_APPLY_SAFE: _ClassVar[OperatingModeApplyMode]
    OPERATING_MODE_APPLY_MODE_AUTO_APPLY_ALL: _ClassVar[OperatingModeApplyMode]

class OperatingModeResultBindingKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    OPERATING_MODE_RESULT_BINDING_KIND_UNSPECIFIED: _ClassVar[OperatingModeResultBindingKind]
    OPERATING_MODE_RESULT_BINDING_KIND_PROGRESS_ARTIFACT: _ClassVar[OperatingModeResultBindingKind]
OPERATING_MODE_SCOPE_KIND_UNSPECIFIED: OperatingModeScopeKind
OPERATING_MODE_SCOPE_KIND_BACKLOG_ITEM: OperatingModeScopeKind
OPERATING_MODE_SCOPE_KIND_INITIATIVE: OperatingModeScopeKind
OPERATING_MODE_RUN_STRATEGY_KIND_UNSPECIFIED: OperatingModeRunStrategyKind
OPERATING_MODE_RUN_STRATEGY_KIND_EXISTING_ITEM_FLOW: OperatingModeRunStrategyKind
OPERATING_MODE_RUN_STRATEGY_KIND_SINGLE_PHASE_RUN: OperatingModeRunStrategyKind
OPERATING_MODE_RUN_STRATEGY_KIND_SEQUENTIAL_HANDOFF: OperatingModeRunStrategyKind
OPERATING_MODE_RUN_STRATEGY_KIND_OPERATOR_GATED_LOOP: OperatingModeRunStrategyKind
OPERATING_MODE_PHASE_KIND_UNSPECIFIED: OperatingModePhaseKind
OPERATING_MODE_PHASE_KIND_INVESTIGATE: OperatingModePhaseKind
OPERATING_MODE_PHASE_KIND_EXECUTE: OperatingModePhaseKind
OPERATING_MODE_PHASE_KIND_REVIEW: OperatingModePhaseKind
OPERATING_MODE_PHASE_KIND_RECONCILE: OperatingModePhaseKind
OPERATING_MODE_GUARD_OP_UNSPECIFIED: OperatingModeGuardOp
OPERATING_MODE_GUARD_OP_ALWAYS: OperatingModeGuardOp
OPERATING_MODE_GUARD_OP_EQ: OperatingModeGuardOp
OPERATING_MODE_GUARD_OP_NE: OperatingModeGuardOp
OPERATING_MODE_GUARD_OP_GT: OperatingModeGuardOp
OPERATING_MODE_GUARD_OP_GTE: OperatingModeGuardOp
OPERATING_MODE_GUARD_OP_LT: OperatingModeGuardOp
OPERATING_MODE_GUARD_OP_LTE: OperatingModeGuardOp
OPERATING_MODE_GUARD_OP_IN: OperatingModeGuardOp
OPERATING_MODE_GUARD_OP_NOT_IN: OperatingModeGuardOp
OPERATING_MODE_GUARD_OP_EXISTS: OperatingModeGuardOp
OPERATING_MODE_GUARD_OP_NOT_EXISTS: OperatingModeGuardOp
OPERATING_MODE_GUARD_OP_ALL: OperatingModeGuardOp
OPERATING_MODE_GUARD_OP_ANY: OperatingModeGuardOp
OPERATING_MODE_GUARD_OP_NOT: OperatingModeGuardOp
OPERATING_MODE_OUTPUT_FIELD_TYPE_UNSPECIFIED: OperatingModeOutputFieldType
OPERATING_MODE_OUTPUT_FIELD_TYPE_STRING: OperatingModeOutputFieldType
OPERATING_MODE_OUTPUT_FIELD_TYPE_BOOLEAN: OperatingModeOutputFieldType
OPERATING_MODE_OUTPUT_FIELD_TYPE_INTEGER: OperatingModeOutputFieldType
OPERATING_MODE_OUTPUT_FIELD_TYPE_NUMBER: OperatingModeOutputFieldType
OPERATING_MODE_OUTPUT_FIELD_TYPE_OBJECT: OperatingModeOutputFieldType
OPERATING_MODE_OUTPUT_FIELD_TYPE_ARRAY: OperatingModeOutputFieldType
OPERATING_MODE_BACKLOG_CAPABILITY_UNSPECIFIED: OperatingModeBacklogCapability
OPERATING_MODE_BACKLOG_CAPABILITY_READ_ONLY: OperatingModeBacklogCapability
OPERATING_MODE_BACKLOG_CAPABILITY_PROPOSE_MUTATIONS: OperatingModeBacklogCapability
OPERATING_MODE_BACKLOG_CAPABILITY_MARK_COMPLETE: OperatingModeBacklogCapability
OPERATING_MODE_BACKLOG_CAPABILITY_CREATE_FOLLOWUPS: OperatingModeBacklogCapability
OPERATING_MODE_BACKLOG_CAPABILITY_UPDATE_SCOPE: OperatingModeBacklogCapability
OPERATING_MODE_APPLY_MODE_UNSPECIFIED: OperatingModeApplyMode
OPERATING_MODE_APPLY_MODE_OPERATOR_GATED: OperatingModeApplyMode
OPERATING_MODE_APPLY_MODE_AUTO_APPLY_SAFE: OperatingModeApplyMode
OPERATING_MODE_APPLY_MODE_AUTO_APPLY_ALL: OperatingModeApplyMode
OPERATING_MODE_RESULT_BINDING_KIND_UNSPECIFIED: OperatingModeResultBindingKind
OPERATING_MODE_RESULT_BINDING_KIND_PROGRESS_ARTIFACT: OperatingModeResultBindingKind

class OperatingMode(_message.Message):
    __slots__ = ("id", "label", "description", "best_for", "not_for", "tradeoffs", "when_in_doubt_pick_instead", "scope", "run_strategy", "phase_graph", "prompt", "artifact", "profile", "backlog_sync", "metrics", "lock", "ui", "schema_version")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    BEST_FOR_FIELD_NUMBER: _ClassVar[int]
    NOT_FOR_FIELD_NUMBER: _ClassVar[int]
    TRADEOFFS_FIELD_NUMBER: _ClassVar[int]
    WHEN_IN_DOUBT_PICK_INSTEAD_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    RUN_STRATEGY_FIELD_NUMBER: _ClassVar[int]
    PHASE_GRAPH_FIELD_NUMBER: _ClassVar[int]
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    BACKLOG_SYNC_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    LOCK_FIELD_NUMBER: _ClassVar[int]
    UI_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    description: str
    best_for: _containers.RepeatedScalarFieldContainer[str]
    not_for: _containers.RepeatedScalarFieldContainer[str]
    tradeoffs: _containers.RepeatedScalarFieldContainer[str]
    when_in_doubt_pick_instead: str
    scope: OperatingModeScope
    run_strategy: OperatingModeRunStrategy
    phase_graph: OperatingModePhaseGraph
    prompt: OperatingModePromptPolicy
    artifact: OperatingModeArtifactPolicy
    profile: OperatingModeProfilePolicy
    backlog_sync: OperatingModeBacklogSyncPolicy
    metrics: OperatingModeMetricsPolicy
    lock: OperatingModeLockPolicy
    ui: OperatingModeUiPolicy
    schema_version: str
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., description: _Optional[str] = ..., best_for: _Optional[_Iterable[str]] = ..., not_for: _Optional[_Iterable[str]] = ..., tradeoffs: _Optional[_Iterable[str]] = ..., when_in_doubt_pick_instead: _Optional[str] = ..., scope: _Optional[_Union[OperatingModeScope, _Mapping]] = ..., run_strategy: _Optional[_Union[OperatingModeRunStrategy, _Mapping]] = ..., phase_graph: _Optional[_Union[OperatingModePhaseGraph, _Mapping]] = ..., prompt: _Optional[_Union[OperatingModePromptPolicy, _Mapping]] = ..., artifact: _Optional[_Union[OperatingModeArtifactPolicy, _Mapping]] = ..., profile: _Optional[_Union[OperatingModeProfilePolicy, _Mapping]] = ..., backlog_sync: _Optional[_Union[OperatingModeBacklogSyncPolicy, _Mapping]] = ..., metrics: _Optional[_Union[OperatingModeMetricsPolicy, _Mapping]] = ..., lock: _Optional[_Union[OperatingModeLockPolicy, _Mapping]] = ..., ui: _Optional[_Union[OperatingModeUiPolicy, _Mapping]] = ..., schema_version: _Optional[str] = ...) -> None: ...

class OperatingModeScope(_message.Message):
    __slots__ = ("kind",)
    KIND_FIELD_NUMBER: _ClassVar[int]
    kind: OperatingModeScopeKind
    def __init__(self, kind: _Optional[_Union[OperatingModeScopeKind, str]] = ...) -> None: ...

class OperatingModeRunStrategy(_message.Message):
    __slots__ = ("kind",)
    KIND_FIELD_NUMBER: _ClassVar[int]
    kind: OperatingModeRunStrategyKind
    def __init__(self, kind: _Optional[_Union[OperatingModeRunStrategyKind, str]] = ...) -> None: ...

class OperatingModePhaseGraph(_message.Message):
    __slots__ = ("start_phase", "terminal", "phases")
    START_PHASE_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_FIELD_NUMBER: _ClassVar[int]
    PHASES_FIELD_NUMBER: _ClassVar[int]
    start_phase: str
    terminal: _containers.RepeatedScalarFieldContainer[str]
    phases: _containers.RepeatedCompositeFieldContainer[OperatingModePhaseDefinition]
    def __init__(self, start_phase: _Optional[str] = ..., terminal: _Optional[_Iterable[str]] = ..., phases: _Optional[_Iterable[_Union[OperatingModePhaseDefinition, _Mapping]]] = ...) -> None: ...

class OperatingModePhaseDefinition(_message.Message):
    __slots__ = ("id", "kind", "activity_purpose", "lock_purpose", "auto_start_after", "writes_repo", "requires_criteria", "profile_key", "prompt", "declared_output", "output_artifacts", "result_bindings", "transitions", "metrics")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    ACTIVITY_PURPOSE_FIELD_NUMBER: _ClassVar[int]
    LOCK_PURPOSE_FIELD_NUMBER: _ClassVar[int]
    AUTO_START_AFTER_FIELD_NUMBER: _ClassVar[int]
    WRITES_REPO_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_CRITERIA_FIELD_NUMBER: _ClassVar[int]
    PROFILE_KEY_FIELD_NUMBER: _ClassVar[int]
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    DECLARED_OUTPUT_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_ARTIFACTS_FIELD_NUMBER: _ClassVar[int]
    RESULT_BINDINGS_FIELD_NUMBER: _ClassVar[int]
    TRANSITIONS_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: OperatingModePhaseKind
    activity_purpose: str
    lock_purpose: str
    auto_start_after: _containers.RepeatedScalarFieldContainer[str]
    writes_repo: bool
    requires_criteria: bool
    profile_key: str
    prompt: OperatingModePhasePrompt
    declared_output: OperatingModeDeclaredOutput
    output_artifacts: _containers.RepeatedCompositeFieldContainer[OperatingModeArtifact]
    result_bindings: _containers.RepeatedCompositeFieldContainer[OperatingModeResultBinding]
    transitions: _containers.RepeatedCompositeFieldContainer[OperatingModeTransition]
    metrics: OperatingModePhaseMetrics
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[_Union[OperatingModePhaseKind, str]] = ..., activity_purpose: _Optional[str] = ..., lock_purpose: _Optional[str] = ..., auto_start_after: _Optional[_Iterable[str]] = ..., writes_repo: _Optional[bool] = ..., requires_criteria: _Optional[bool] = ..., profile_key: _Optional[str] = ..., prompt: _Optional[_Union[OperatingModePhasePrompt, _Mapping]] = ..., declared_output: _Optional[_Union[OperatingModeDeclaredOutput, _Mapping]] = ..., output_artifacts: _Optional[_Iterable[_Union[OperatingModeArtifact, _Mapping]]] = ..., result_bindings: _Optional[_Iterable[_Union[OperatingModeResultBinding, _Mapping]]] = ..., transitions: _Optional[_Iterable[_Union[OperatingModeTransition, _Mapping]]] = ..., metrics: _Optional[_Union[OperatingModePhaseMetrics, _Mapping]] = ...) -> None: ...

class OperatingModePhasePrompt(_message.Message):
    __slots__ = ("template", "suffix", "title", "trigger", "purpose")
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    SUFFIX_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    TRIGGER_FIELD_NUMBER: _ClassVar[int]
    PURPOSE_FIELD_NUMBER: _ClassVar[int]
    template: str
    suffix: str
    title: str
    trigger: str
    purpose: str
    def __init__(self, template: _Optional[str] = ..., suffix: _Optional[str] = ..., title: _Optional[str] = ..., trigger: _Optional[str] = ..., purpose: _Optional[str] = ...) -> None: ...

class OperatingModeDeclaredOutput(_message.Message):
    __slots__ = ("envelope_key", "requires_structured_result", "fields", "resolution")
    ENVELOPE_KEY_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_STRUCTURED_RESULT_FIELD_NUMBER: _ClassVar[int]
    FIELDS_FIELD_NUMBER: _ClassVar[int]
    RESOLUTION_FIELD_NUMBER: _ClassVar[int]
    envelope_key: str
    requires_structured_result: bool
    fields: _containers.RepeatedCompositeFieldContainer[OperatingModeOutputField]
    resolution: OperatingModeResolutionPolicy
    def __init__(self, envelope_key: _Optional[str] = ..., requires_structured_result: _Optional[bool] = ..., fields: _Optional[_Iterable[_Union[OperatingModeOutputField, _Mapping]]] = ..., resolution: _Optional[_Union[OperatingModeResolutionPolicy, _Mapping]] = ...) -> None: ...

class OperatingModeOutputField(_message.Message):
    __slots__ = ("name", "type", "required", "enum_values", "minimum", "maximum", "min_length", "max_length", "description", "fields")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    ENUM_VALUES_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_FIELD_NUMBER: _ClassVar[int]
    MAXIMUM_FIELD_NUMBER: _ClassVar[int]
    MIN_LENGTH_FIELD_NUMBER: _ClassVar[int]
    MAX_LENGTH_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    FIELDS_FIELD_NUMBER: _ClassVar[int]
    name: str
    type: OperatingModeOutputFieldType
    required: bool
    enum_values: _containers.RepeatedCompositeFieldContainer[_struct_pb2.Value]
    minimum: float
    maximum: float
    min_length: int
    max_length: int
    description: str
    fields: _containers.RepeatedCompositeFieldContainer[OperatingModeOutputField]
    def __init__(self, name: _Optional[str] = ..., type: _Optional[_Union[OperatingModeOutputFieldType, str]] = ..., required: _Optional[bool] = ..., enum_values: _Optional[_Iterable[_Union[_struct_pb2.Value, _Mapping]]] = ..., minimum: _Optional[float] = ..., maximum: _Optional[float] = ..., min_length: _Optional[int] = ..., max_length: _Optional[int] = ..., description: _Optional[str] = ..., fields: _Optional[_Iterable[_Union[OperatingModeOutputField, _Mapping]]] = ...) -> None: ...

class OperatingModeResolutionPolicy(_message.Message):
    __slots__ = ("detect_true_final_message", "scan_last_n_messages", "allow_classifier")
    DETECT_TRUE_FINAL_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SCAN_LAST_N_MESSAGES_FIELD_NUMBER: _ClassVar[int]
    ALLOW_CLASSIFIER_FIELD_NUMBER: _ClassVar[int]
    detect_true_final_message: bool
    scan_last_n_messages: int
    allow_classifier: bool
    def __init__(self, detect_true_final_message: _Optional[bool] = ..., scan_last_n_messages: _Optional[int] = ..., allow_classifier: _Optional[bool] = ...) -> None: ...

class OperatingModeTransition(_message.Message):
    __slots__ = ("when", "to")
    WHEN_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    when: OperatingModeGuard
    to: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, when: _Optional[_Union[OperatingModeGuard, _Mapping]] = ..., to: _Optional[_Iterable[str]] = ...) -> None: ...

class OperatingModeGuard(_message.Message):
    __slots__ = ("op", "field", "value", "values", "guards")
    OP_FIELD_NUMBER: _ClassVar[int]
    FIELD_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    VALUES_FIELD_NUMBER: _ClassVar[int]
    GUARDS_FIELD_NUMBER: _ClassVar[int]
    op: OperatingModeGuardOp
    field: str
    value: _struct_pb2.Value
    values: _containers.RepeatedCompositeFieldContainer[_struct_pb2.Value]
    guards: _containers.RepeatedCompositeFieldContainer[OperatingModeGuard]
    def __init__(self, op: _Optional[_Union[OperatingModeGuardOp, str]] = ..., field: _Optional[str] = ..., value: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., values: _Optional[_Iterable[_Union[_struct_pb2.Value, _Mapping]]] = ..., guards: _Optional[_Iterable[_Union[OperatingModeGuard, _Mapping]]] = ...) -> None: ...

class OperatingModeArtifact(_message.Message):
    __slots__ = ("path", "content_type", "required")
    PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    path: str
    content_type: str
    required: bool
    def __init__(self, path: _Optional[str] = ..., content_type: _Optional[str] = ..., required: _Optional[bool] = ...) -> None: ...

class OperatingModeResultBinding(_message.Message):
    __slots__ = ("kind", "artifact")
    KIND_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_FIELD_NUMBER: _ClassVar[int]
    kind: OperatingModeResultBindingKind
    artifact: OperatingModeArtifact
    def __init__(self, kind: _Optional[_Union[OperatingModeResultBindingKind, str]] = ..., artifact: _Optional[_Union[OperatingModeArtifact, _Mapping]] = ...) -> None: ...

class OperatingModePhaseMetrics(_message.Message):
    __slots__ = ("counts_replan_sample", "counts_acceptance_sample")
    COUNTS_REPLAN_SAMPLE_FIELD_NUMBER: _ClassVar[int]
    COUNTS_ACCEPTANCE_SAMPLE_FIELD_NUMBER: _ClassVar[int]
    counts_replan_sample: bool
    counts_acceptance_sample: bool
    def __init__(self, counts_replan_sample: _Optional[bool] = ..., counts_acceptance_sample: _Optional[bool] = ...) -> None: ...

class OperatingModePromptPolicy(_message.Message):
    __slots__ = ("catalog_prefix",)
    CATALOG_PREFIX_FIELD_NUMBER: _ClassVar[int]
    catalog_prefix: str
    def __init__(self, catalog_prefix: _Optional[str] = ...) -> None: ...

class OperatingModeArtifactPolicy(_message.Message):
    __slots__ = ("root", "round_root")
    ROOT_FIELD_NUMBER: _ClassVar[int]
    ROUND_ROOT_FIELD_NUMBER: _ClassVar[int]
    root: str
    round_root: str
    def __init__(self, root: _Optional[str] = ..., round_root: _Optional[str] = ...) -> None: ...

class OperatingModeProfilePolicy(_message.Message):
    __slots__ = ("default_profile_key",)
    DEFAULT_PROFILE_KEY_FIELD_NUMBER: _ClassVar[int]
    default_profile_key: str
    def __init__(self, default_profile_key: _Optional[str] = ...) -> None: ...

class OperatingModeBacklogSyncPolicy(_message.Message):
    __slots__ = ("capabilities", "requires_run_id", "requires_membership", "event_source", "apply_mode")
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_RUN_ID_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_MEMBERSHIP_FIELD_NUMBER: _ClassVar[int]
    EVENT_SOURCE_FIELD_NUMBER: _ClassVar[int]
    APPLY_MODE_FIELD_NUMBER: _ClassVar[int]
    capabilities: _containers.RepeatedScalarFieldContainer[OperatingModeBacklogCapability]
    requires_run_id: bool
    requires_membership: bool
    event_source: str
    apply_mode: OperatingModeApplyMode
    def __init__(self, capabilities: _Optional[_Iterable[_Union[OperatingModeBacklogCapability, str]]] = ..., requires_run_id: _Optional[bool] = ..., requires_membership: _Optional[bool] = ..., event_source: _Optional[str] = ..., apply_mode: _Optional[_Union[OperatingModeApplyMode, str]] = ...) -> None: ...

class OperatingModeMetricsPolicy(_message.Message):
    __slots__ = ("event_source", "accepted_verdicts")
    EVENT_SOURCE_FIELD_NUMBER: _ClassVar[int]
    ACCEPTED_VERDICTS_FIELD_NUMBER: _ClassVar[int]
    event_source: str
    accepted_verdicts: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, event_source: _Optional[str] = ..., accepted_verdicts: _Optional[_Iterable[str]] = ...) -> None: ...

class OperatingModeLockPolicy(_message.Message):
    __slots__ = ("initiative_exclusive",)
    INITIATIVE_EXCLUSIVE_FIELD_NUMBER: _ClassVar[int]
    initiative_exclusive: bool
    def __init__(self, initiative_exclusive: _Optional[bool] = ...) -> None: ...

class OperatingModeUiPolicy(_message.Message):
    __slots__ = ("workspace_tab_id",)
    WORKSPACE_TAB_ID_FIELD_NUMBER: _ClassVar[int]
    workspace_tab_id: str
    def __init__(self, workspace_tab_id: _Optional[str] = ...) -> None: ...

class OperatingModeExampleRun(_message.Message):
    __slots__ = ("id", "mode", "label", "description", "steps", "expected_path")
    ID_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_PATH_FIELD_NUMBER: _ClassVar[int]
    id: str
    mode: str
    label: str
    description: str
    steps: _containers.RepeatedCompositeFieldContainer[OperatingModeExampleRunStep]
    expected_path: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., mode: _Optional[str] = ..., label: _Optional[str] = ..., description: _Optional[str] = ..., steps: _Optional[_Iterable[_Union[OperatingModeExampleRunStep, _Mapping]]] = ..., expected_path: _Optional[_Iterable[str]] = ...) -> None: ...

class OperatingModeExampleRunStep(_message.Message):
    __slots__ = ("phase", "output", "note")
    PHASE_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    phase: str
    output: _struct_pb2.Struct
    note: str
    def __init__(self, phase: _Optional[str] = ..., output: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., note: _Optional[str] = ...) -> None: ...
