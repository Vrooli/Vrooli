from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class FlowSummary(_message.Message):
    __slots__ = ("flow_id", "contract_path", "language", "schema_version", "scenario_id", "kind")
    FLOW_ID_FIELD_NUMBER: _ClassVar[int]
    CONTRACT_PATH_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    flow_id: str
    contract_path: str
    language: str
    schema_version: int
    scenario_id: str
    kind: str
    def __init__(self, flow_id: _Optional[str] = ..., contract_path: _Optional[str] = ..., language: _Optional[str] = ..., schema_version: _Optional[int] = ..., scenario_id: _Optional[str] = ..., kind: _Optional[str] = ...) -> None: ...

class FlowState(_message.Message):
    __slots__ = ("id", "quint", "initial", "terminal")
    ID_FIELD_NUMBER: _ClassVar[int]
    QUINT_FIELD_NUMBER: _ClassVar[int]
    INITIAL_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_FIELD_NUMBER: _ClassVar[int]
    id: str
    quint: str
    initial: bool
    terminal: bool
    def __init__(self, id: _Optional[str] = ..., quint: _Optional[str] = ..., initial: _Optional[bool] = ..., terminal: _Optional[bool] = ...) -> None: ...

class FlowEvent(_message.Message):
    __slots__ = ("id", "quint")
    ID_FIELD_NUMBER: _ClassVar[int]
    QUINT_FIELD_NUMBER: _ClassVar[int]
    id: str
    quint: str
    def __init__(self, id: _Optional[str] = ..., quint: _Optional[str] = ...) -> None: ...

class FlowTransition(_message.Message):
    __slots__ = ("event", "to", "want_error", "want_error_set")
    FROM_FIELD_NUMBER: _ClassVar[int]
    EVENT_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    WANT_ERROR_FIELD_NUMBER: _ClassVar[int]
    WANT_ERROR_SET_FIELD_NUMBER: _ClassVar[int]
    event: _containers.RepeatedScalarFieldContainer[str]
    to: str
    want_error: bool
    want_error_set: bool
    def __init__(self, event: _Optional[_Iterable[str]] = ..., to: _Optional[str] = ..., want_error: _Optional[bool] = ..., want_error_set: _Optional[bool] = ..., **kwargs) -> None: ...

class FlowInvariant(_message.Message):
    __slots__ = ("id", "quint", "description", "expression")
    ID_FIELD_NUMBER: _ClassVar[int]
    QUINT_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    EXPRESSION_FIELD_NUMBER: _ClassVar[int]
    id: str
    quint: str
    description: str
    expression: str
    def __init__(self, id: _Optional[str] = ..., quint: _Optional[str] = ..., description: _Optional[str] = ..., expression: _Optional[str] = ...) -> None: ...

class FlowTraceStep(_message.Message):
    __slots__ = ("event", "want", "want_error")
    EVENT_FIELD_NUMBER: _ClassVar[int]
    WANT_FIELD_NUMBER: _ClassVar[int]
    WANT_ERROR_FIELD_NUMBER: _ClassVar[int]
    event: str
    want: str
    want_error: bool
    def __init__(self, event: _Optional[str] = ..., want: _Optional[str] = ..., want_error: _Optional[bool] = ...) -> None: ...

class FlowTrace(_message.Message):
    __slots__ = ("name", "initial", "steps")
    NAME_FIELD_NUMBER: _ClassVar[int]
    INITIAL_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    name: str
    initial: str
    steps: _containers.RepeatedCompositeFieldContainer[FlowTraceStep]
    def __init__(self, name: _Optional[str] = ..., initial: _Optional[str] = ..., steps: _Optional[_Iterable[_Union[FlowTraceStep, _Mapping]]] = ...) -> None: ...

class FlowVerify(_message.Message):
    __slots__ = ("invariants",)
    INVARIANTS_FIELD_NUMBER: _ClassVar[int]
    invariants: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, invariants: _Optional[_Iterable[str]] = ...) -> None: ...

class FlowModel(_message.Message):
    __slots__ = ("module", "seed", "max_steps", "trace_count", "verify")
    MODULE_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    MAX_STEPS_FIELD_NUMBER: _ClassVar[int]
    TRACE_COUNT_FIELD_NUMBER: _ClassVar[int]
    VERIFY_FIELD_NUMBER: _ClassVar[int]
    module: str
    seed: str
    max_steps: int
    trace_count: int
    verify: FlowVerify
    def __init__(self, module: _Optional[str] = ..., seed: _Optional[str] = ..., max_steps: _Optional[int] = ..., trace_count: _Optional[int] = ..., verify: _Optional[_Union[FlowVerify, _Mapping]] = ...) -> None: ...

class GoRuntime(_message.Message):
    __slots__ = ("package", "status_type", "event_type", "constant_prefix")
    PACKAGE_FIELD_NUMBER: _ClassVar[int]
    STATUS_TYPE_FIELD_NUMBER: _ClassVar[int]
    EVENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    CONSTANT_PREFIX_FIELD_NUMBER: _ClassVar[int]
    package: str
    status_type: str
    event_type: str
    constant_prefix: str
    def __init__(self, package: _Optional[str] = ..., status_type: _Optional[str] = ..., event_type: _Optional[str] = ..., constant_prefix: _Optional[str] = ...) -> None: ...

class TypeScriptRuntime(_message.Message):
    __slots__ = ("status_type", "event_type", "statuses_const", "events_const", "formal_expectation_const", "state_union_type", "event_union_type", "payload_types", "state_variants", "event_variants")
    class PayloadTypesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class StateVariantsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: VariantFields
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[VariantFields, _Mapping]] = ...) -> None: ...
    class EventVariantsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: VariantFields
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[VariantFields, _Mapping]] = ...) -> None: ...
    STATUS_TYPE_FIELD_NUMBER: _ClassVar[int]
    EVENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    STATUSES_CONST_FIELD_NUMBER: _ClassVar[int]
    EVENTS_CONST_FIELD_NUMBER: _ClassVar[int]
    FORMAL_EXPECTATION_CONST_FIELD_NUMBER: _ClassVar[int]
    STATE_UNION_TYPE_FIELD_NUMBER: _ClassVar[int]
    EVENT_UNION_TYPE_FIELD_NUMBER: _ClassVar[int]
    PAYLOAD_TYPES_FIELD_NUMBER: _ClassVar[int]
    STATE_VARIANTS_FIELD_NUMBER: _ClassVar[int]
    EVENT_VARIANTS_FIELD_NUMBER: _ClassVar[int]
    status_type: str
    event_type: str
    statuses_const: str
    events_const: str
    formal_expectation_const: str
    state_union_type: str
    event_union_type: str
    payload_types: _containers.ScalarMap[str, str]
    state_variants: _containers.MessageMap[str, VariantFields]
    event_variants: _containers.MessageMap[str, VariantFields]
    def __init__(self, status_type: _Optional[str] = ..., event_type: _Optional[str] = ..., statuses_const: _Optional[str] = ..., events_const: _Optional[str] = ..., formal_expectation_const: _Optional[str] = ..., state_union_type: _Optional[str] = ..., event_union_type: _Optional[str] = ..., payload_types: _Optional[_Mapping[str, str]] = ..., state_variants: _Optional[_Mapping[str, VariantFields]] = ..., event_variants: _Optional[_Mapping[str, VariantFields]] = ...) -> None: ...

class VariantFields(_message.Message):
    __slots__ = ("fields",)
    class FieldsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    FIELDS_FIELD_NUMBER: _ClassVar[int]
    fields: _containers.ScalarMap[str, str]
    def __init__(self, fields: _Optional[_Mapping[str, str]] = ...) -> None: ...

class FlowRuntime(_message.Message):
    __slots__ = ("go", "typescript", "side_effects", "stale_completion")
    GO_FIELD_NUMBER: _ClassVar[int]
    TYPESCRIPT_FIELD_NUMBER: _ClassVar[int]
    SIDE_EFFECTS_FIELD_NUMBER: _ClassVar[int]
    STALE_COMPLETION_FIELD_NUMBER: _ClassVar[int]
    go: GoRuntime
    typescript: TypeScriptRuntime
    side_effects: _containers.RepeatedScalarFieldContainer[str]
    stale_completion: str
    def __init__(self, go: _Optional[_Union[GoRuntime, _Mapping]] = ..., typescript: _Optional[_Union[TypeScriptRuntime, _Mapping]] = ..., side_effects: _Optional[_Iterable[str]] = ..., stale_completion: _Optional[str] = ...) -> None: ...

class FlowDetail(_message.Message):
    __slots__ = ("flow_id", "domain", "description", "contract_path", "language", "schema_version", "kind", "initial_state", "states", "events", "transitions", "traces", "invariants", "model", "runtime", "report")
    FLOW_ID_FIELD_NUMBER: _ClassVar[int]
    DOMAIN_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CONTRACT_PATH_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    INITIAL_STATE_FIELD_NUMBER: _ClassVar[int]
    STATES_FIELD_NUMBER: _ClassVar[int]
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    TRANSITIONS_FIELD_NUMBER: _ClassVar[int]
    TRACES_FIELD_NUMBER: _ClassVar[int]
    INVARIANTS_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    RUNTIME_FIELD_NUMBER: _ClassVar[int]
    REPORT_FIELD_NUMBER: _ClassVar[int]
    flow_id: str
    domain: str
    description: str
    contract_path: str
    language: str
    schema_version: int
    kind: str
    initial_state: str
    states: _containers.RepeatedCompositeFieldContainer[FlowState]
    events: _containers.RepeatedCompositeFieldContainer[FlowEvent]
    transitions: _containers.RepeatedCompositeFieldContainer[FlowTransition]
    traces: _containers.RepeatedCompositeFieldContainer[FlowTrace]
    invariants: _containers.RepeatedCompositeFieldContainer[FlowInvariant]
    model: FlowModel
    runtime: FlowRuntime
    report: str
    def __init__(self, flow_id: _Optional[str] = ..., domain: _Optional[str] = ..., description: _Optional[str] = ..., contract_path: _Optional[str] = ..., language: _Optional[str] = ..., schema_version: _Optional[int] = ..., kind: _Optional[str] = ..., initial_state: _Optional[str] = ..., states: _Optional[_Iterable[_Union[FlowState, _Mapping]]] = ..., events: _Optional[_Iterable[_Union[FlowEvent, _Mapping]]] = ..., transitions: _Optional[_Iterable[_Union[FlowTransition, _Mapping]]] = ..., traces: _Optional[_Iterable[_Union[FlowTrace, _Mapping]]] = ..., invariants: _Optional[_Iterable[_Union[FlowInvariant, _Mapping]]] = ..., model: _Optional[_Union[FlowModel, _Mapping]] = ..., runtime: _Optional[_Union[FlowRuntime, _Mapping]] = ..., report: _Optional[str] = ...) -> None: ...

class ListFlowsRequest(_message.Message):
    __slots__ = ("root", "flow_id", "kind")
    ROOT_FIELD_NUMBER: _ClassVar[int]
    FLOW_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    root: str
    flow_id: str
    kind: str
    def __init__(self, root: _Optional[str] = ..., flow_id: _Optional[str] = ..., kind: _Optional[str] = ...) -> None: ...

class ListFlowsResponse(_message.Message):
    __slots__ = ("flows",)
    FLOWS_FIELD_NUMBER: _ClassVar[int]
    flows: _containers.RepeatedCompositeFieldContainer[FlowSummary]
    def __init__(self, flows: _Optional[_Iterable[_Union[FlowSummary, _Mapping]]] = ...) -> None: ...

class GetFlowRequest(_message.Message):
    __slots__ = ("flow_id", "root")
    FLOW_ID_FIELD_NUMBER: _ClassVar[int]
    ROOT_FIELD_NUMBER: _ClassVar[int]
    flow_id: str
    root: str
    def __init__(self, flow_id: _Optional[str] = ..., root: _Optional[str] = ...) -> None: ...

class GetFlowResponse(_message.Message):
    __slots__ = ("flow",)
    FLOW_FIELD_NUMBER: _ClassVar[int]
    flow: FlowDetail
    def __init__(self, flow: _Optional[_Union[FlowDetail, _Mapping]] = ...) -> None: ...

class CreateFlowRequest(_message.Message):
    __slots__ = ("parent_dir", "flow_id", "language", "root", "kind")
    PARENT_DIR_FIELD_NUMBER: _ClassVar[int]
    FLOW_ID_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    ROOT_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    parent_dir: str
    flow_id: str
    language: str
    root: str
    kind: str
    def __init__(self, parent_dir: _Optional[str] = ..., flow_id: _Optional[str] = ..., language: _Optional[str] = ..., root: _Optional[str] = ..., kind: _Optional[str] = ...) -> None: ...

class CreateFlowResponse(_message.Message):
    __slots__ = ("flow_dir",)
    FLOW_DIR_FIELD_NUMBER: _ClassVar[int]
    flow_dir: str
    def __init__(self, flow_dir: _Optional[str] = ...) -> None: ...

class ValidateFlowRequest(_message.Message):
    __slots__ = ("flow_id", "root")
    FLOW_ID_FIELD_NUMBER: _ClassVar[int]
    ROOT_FIELD_NUMBER: _ClassVar[int]
    flow_id: str
    root: str
    def __init__(self, flow_id: _Optional[str] = ..., root: _Optional[str] = ...) -> None: ...

class ValidateFlowResponse(_message.Message):
    __slots__ = ("flows",)
    FLOWS_FIELD_NUMBER: _ClassVar[int]
    flows: _containers.RepeatedCompositeFieldContainer[FlowSummary]
    def __init__(self, flows: _Optional[_Iterable[_Union[FlowSummary, _Mapping]]] = ...) -> None: ...

class ExplainFlowRequest(_message.Message):
    __slots__ = ("flow_id", "root")
    FLOW_ID_FIELD_NUMBER: _ClassVar[int]
    ROOT_FIELD_NUMBER: _ClassVar[int]
    flow_id: str
    root: str
    def __init__(self, flow_id: _Optional[str] = ..., root: _Optional[str] = ...) -> None: ...

class ExplainFlowResponse(_message.Message):
    __slots__ = ("report",)
    REPORT_FIELD_NUMBER: _ClassVar[int]
    report: str
    def __init__(self, report: _Optional[str] = ...) -> None: ...
