from program_runtime.v1.shared import library_pb2 as _library_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class UnboundReason(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    UNBOUND_REASON_UNSPECIFIED: _ClassVar[UnboundReason]
    UNBOUND_REASON_NO_MANIFEST: _ClassVar[UnboundReason]
    UNBOUND_REASON_LOCAL_BINDING: _ClassVar[UnboundReason]
    UNBOUND_REASON_OMITTED_RPC: _ClassVar[UnboundReason]
    UNBOUND_REASON_EXTERNAL_TOOL_ONLY: _ClassVar[UnboundReason]
    UNBOUND_REASON_MALFORMED_MANIFEST: _ClassVar[UnboundReason]

class ConditionStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CONDITION_STATUS_UNSPECIFIED: _ClassVar[ConditionStatus]
    CONDITION_STATUS_HEALTHY: _ClassVar[ConditionStatus]
    CONDITION_STATUS_DEGRADED: _ClassVar[ConditionStatus]
    CONDITION_STATUS_DORMANT: _ClassVar[ConditionStatus]
    CONDITION_STATUS_UNINSTRUMENTED: _ClassVar[ConditionStatus]
    CONDITION_STATUS_UNAVAILABLE: _ClassVar[ConditionStatus]

class ActVerdict(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ACT_VERDICT_UNSPECIFIED: _ClassVar[ActVerdict]
    ACT_VERDICT_NOW: _ClassVar[ActVerdict]
    ACT_VERDICT_IN_REACH: _ClassVar[ActVerdict]
    ACT_VERDICT_AUTHORED: _ClassVar[ActVerdict]
UNBOUND_REASON_UNSPECIFIED: UnboundReason
UNBOUND_REASON_NO_MANIFEST: UnboundReason
UNBOUND_REASON_LOCAL_BINDING: UnboundReason
UNBOUND_REASON_OMITTED_RPC: UnboundReason
UNBOUND_REASON_EXTERNAL_TOOL_ONLY: UnboundReason
UNBOUND_REASON_MALFORMED_MANIFEST: UnboundReason
CONDITION_STATUS_UNSPECIFIED: ConditionStatus
CONDITION_STATUS_HEALTHY: ConditionStatus
CONDITION_STATUS_DEGRADED: ConditionStatus
CONDITION_STATUS_DORMANT: ConditionStatus
CONDITION_STATUS_UNINSTRUMENTED: ConditionStatus
CONDITION_STATUS_UNAVAILABLE: ConditionStatus
ACT_VERDICT_UNSPECIFIED: ActVerdict
ACT_VERDICT_NOW: ActVerdict
ACT_VERDICT_IN_REACH: ActVerdict
ACT_VERDICT_AUTHORED: ActVerdict

class Binding(_message.Message):
    __slots__ = ("id", "scenario", "group", "command", "service", "method", "request_type", "response_type", "effect", "run_eligible", "requires_confirmation", "permissions", "description", "signature", "reachable", "reachability_reason", "rows_field", "meta_fields", "row_field_candidates")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    GROUP_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    SERVICE_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    REQUEST_TYPE_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_TYPE_FIELD_NUMBER: _ClassVar[int]
    EFFECT_FIELD_NUMBER: _ClassVar[int]
    RUN_ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_CONFIRMATION_FIELD_NUMBER: _ClassVar[int]
    PERMISSIONS_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    SIGNATURE_FIELD_NUMBER: _ClassVar[int]
    REACHABLE_FIELD_NUMBER: _ClassVar[int]
    REACHABILITY_REASON_FIELD_NUMBER: _ClassVar[int]
    ROWS_FIELD_FIELD_NUMBER: _ClassVar[int]
    META_FIELDS_FIELD_NUMBER: _ClassVar[int]
    ROW_FIELD_CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario: str
    group: str
    command: str
    service: str
    method: str
    request_type: str
    response_type: str
    effect: str
    run_eligible: bool
    requires_confirmation: bool
    permissions: _containers.RepeatedScalarFieldContainer[str]
    description: str
    signature: str
    reachable: bool
    reachability_reason: str
    rows_field: str
    meta_fields: _containers.RepeatedScalarFieldContainer[str]
    row_field_candidates: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., scenario: _Optional[str] = ..., group: _Optional[str] = ..., command: _Optional[str] = ..., service: _Optional[str] = ..., method: _Optional[str] = ..., request_type: _Optional[str] = ..., response_type: _Optional[str] = ..., effect: _Optional[str] = ..., run_eligible: _Optional[bool] = ..., requires_confirmation: _Optional[bool] = ..., permissions: _Optional[_Iterable[str]] = ..., description: _Optional[str] = ..., signature: _Optional[str] = ..., reachable: _Optional[bool] = ..., reachability_reason: _Optional[str] = ..., rows_field: _Optional[str] = ..., meta_fields: _Optional[_Iterable[str]] = ..., row_field_candidates: _Optional[_Iterable[str]] = ...) -> None: ...

class UnboundCapability(_message.Message):
    __slots__ = ("scenario", "group", "command", "service", "method", "reason", "detail")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    GROUP_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    SERVICE_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    group: str
    command: str
    service: str
    method: str
    reason: UnboundReason
    detail: str
    def __init__(self, scenario: _Optional[str] = ..., group: _Optional[str] = ..., command: _Optional[str] = ..., service: _Optional[str] = ..., method: _Optional[str] = ..., reason: _Optional[_Union[UnboundReason, str]] = ..., detail: _Optional[str] = ...) -> None: ...

class ListBindingsRequest(_message.Message):
    __slots__ = ("scenario", "group", "reachable_only")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    GROUP_FIELD_NUMBER: _ClassVar[int]
    REACHABLE_ONLY_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    group: str
    reachable_only: bool
    def __init__(self, scenario: _Optional[str] = ..., group: _Optional[str] = ..., reachable_only: _Optional[bool] = ...) -> None: ...

class ListBindingsResponse(_message.Message):
    __slots__ = ("bindings", "reachability_checked_at")
    BINDINGS_FIELD_NUMBER: _ClassVar[int]
    REACHABILITY_CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    bindings: _containers.RepeatedCompositeFieldContainer[Binding]
    reachability_checked_at: str
    def __init__(self, bindings: _Optional[_Iterable[_Union[Binding, _Mapping]]] = ..., reachability_checked_at: _Optional[str] = ...) -> None: ...

class SweepBindingsRequest(_message.Message):
    __slots__ = ("scenario", "effect", "dry_run")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    EFFECT_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    effect: str
    dry_run: bool
    def __init__(self, scenario: _Optional[str] = ..., effect: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class SweepBindingResult(_message.Message):
    __slots__ = ("binding_id", "scenario", "attempted", "skipped_reason", "outcome", "reason", "latency_ms")
    BINDING_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    ATTEMPTED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_REASON_FIELD_NUMBER: _ClassVar[int]
    OUTCOME_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    binding_id: str
    scenario: str
    attempted: bool
    skipped_reason: str
    outcome: str
    reason: str
    latency_ms: int
    def __init__(self, binding_id: _Optional[str] = ..., scenario: _Optional[str] = ..., attempted: _Optional[bool] = ..., skipped_reason: _Optional[str] = ..., outcome: _Optional[str] = ..., reason: _Optional[str] = ..., latency_ms: _Optional[int] = ...) -> None: ...

class SweepBindingsResponse(_message.Message):
    __slots__ = ("results", "eligible", "attempted", "skipped", "succeeded", "failed", "refused", "provenance")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    ATTEMPTED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    SUCCEEDED_FIELD_NUMBER: _ClassVar[int]
    FAILED_FIELD_NUMBER: _ClassVar[int]
    REFUSED_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[SweepBindingResult]
    eligible: int
    attempted: int
    skipped: int
    succeeded: int
    failed: int
    refused: int
    provenance: str
    def __init__(self, results: _Optional[_Iterable[_Union[SweepBindingResult, _Mapping]]] = ..., eligible: _Optional[int] = ..., attempted: _Optional[int] = ..., skipped: _Optional[int] = ..., succeeded: _Optional[int] = ..., failed: _Optional[int] = ..., refused: _Optional[int] = ..., provenance: _Optional[str] = ...) -> None: ...

class ListUnboundRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class ListUnboundResponse(_message.Message):
    __slots__ = ("capabilities",)
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    capabilities: _containers.RepeatedCompositeFieldContainer[UnboundCapability]
    def __init__(self, capabilities: _Optional[_Iterable[_Union[UnboundCapability, _Mapping]]] = ...) -> None: ...

class DoctorBindingsRequest(_message.Message):
    __slots__ = ("scenario",)
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    def __init__(self, scenario: _Optional[str] = ...) -> None: ...

class SkippedManifest(_message.Message):
    __slots__ = ("path", "scenario", "parse_error")
    PATH_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PARSE_ERROR_FIELD_NUMBER: _ClassVar[int]
    path: str
    scenario: str
    parse_error: str
    def __init__(self, path: _Optional[str] = ..., scenario: _Optional[str] = ..., parse_error: _Optional[str] = ...) -> None: ...

class BindingIssue(_message.Message):
    __slots__ = ("scenario", "binding_id", "argument", "request_type", "reason", "proto_path", "candidate_fields")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    BINDING_ID_FIELD_NUMBER: _ClassVar[int]
    ARGUMENT_FIELD_NUMBER: _ClassVar[int]
    REQUEST_TYPE_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    PROTO_PATH_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_FIELDS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    binding_id: str
    argument: str
    request_type: str
    reason: str
    proto_path: str
    candidate_fields: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., binding_id: _Optional[str] = ..., argument: _Optional[str] = ..., request_type: _Optional[str] = ..., reason: _Optional[str] = ..., proto_path: _Optional[str] = ..., candidate_fields: _Optional[_Iterable[str]] = ...) -> None: ...

class DoctorBindingsResponse(_message.Message):
    __slots__ = ("bindings", "callable", "uncallable", "partial", "zero_arg", "misroutes", "issues", "field_collisions", "control_flags_bound", "required_fields_unpopulated", "binds_where_rename_suffices", "scalar_bound_to_message", "skipped_manifests", "skipped_manifest_count", "reachable_scenarios", "unreachable_scenarios", "manifest_scenarios", "total_scenarios", "reachability_checked_at", "reachability_checked_at_by_scenario")
    class ReachabilityCheckedAtByScenarioEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    BINDINGS_FIELD_NUMBER: _ClassVar[int]
    CALLABLE_FIELD_NUMBER: _ClassVar[int]
    UNCALLABLE_FIELD_NUMBER: _ClassVar[int]
    PARTIAL_FIELD_NUMBER: _ClassVar[int]
    ZERO_ARG_FIELD_NUMBER: _ClassVar[int]
    MISROUTES_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    FIELD_COLLISIONS_FIELD_NUMBER: _ClassVar[int]
    CONTROL_FLAGS_BOUND_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELDS_UNPOPULATED_FIELD_NUMBER: _ClassVar[int]
    BINDS_WHERE_RENAME_SUFFICES_FIELD_NUMBER: _ClassVar[int]
    SCALAR_BOUND_TO_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_MANIFESTS_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_MANIFEST_COUNT_FIELD_NUMBER: _ClassVar[int]
    REACHABLE_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    UNREACHABLE_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_SCENARIOS_FIELD_NUMBER: _ClassVar[int]
    REACHABILITY_CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    REACHABILITY_CHECKED_AT_BY_SCENARIO_FIELD_NUMBER: _ClassVar[int]
    bindings: int
    callable: int
    uncallable: int
    partial: int
    zero_arg: int
    misroutes: int
    issues: _containers.RepeatedCompositeFieldContainer[BindingIssue]
    field_collisions: int
    control_flags_bound: int
    required_fields_unpopulated: int
    binds_where_rename_suffices: int
    scalar_bound_to_message: int
    skipped_manifests: _containers.RepeatedCompositeFieldContainer[SkippedManifest]
    skipped_manifest_count: int
    reachable_scenarios: _containers.RepeatedScalarFieldContainer[str]
    unreachable_scenarios: _containers.RepeatedScalarFieldContainer[str]
    manifest_scenarios: int
    total_scenarios: int
    reachability_checked_at: str
    reachability_checked_at_by_scenario: _containers.ScalarMap[str, str]
    def __init__(self, bindings: _Optional[int] = ..., callable: _Optional[int] = ..., uncallable: _Optional[int] = ..., partial: _Optional[int] = ..., zero_arg: _Optional[int] = ..., misroutes: _Optional[int] = ..., issues: _Optional[_Iterable[_Union[BindingIssue, _Mapping]]] = ..., field_collisions: _Optional[int] = ..., control_flags_bound: _Optional[int] = ..., required_fields_unpopulated: _Optional[int] = ..., binds_where_rename_suffices: _Optional[int] = ..., scalar_bound_to_message: _Optional[int] = ..., skipped_manifests: _Optional[_Iterable[_Union[SkippedManifest, _Mapping]]] = ..., skipped_manifest_count: _Optional[int] = ..., reachable_scenarios: _Optional[_Iterable[str]] = ..., unreachable_scenarios: _Optional[_Iterable[str]] = ..., manifest_scenarios: _Optional[int] = ..., total_scenarios: _Optional[int] = ..., reachability_checked_at: _Optional[str] = ..., reachability_checked_at_by_scenario: _Optional[_Mapping[str, str]] = ...) -> None: ...

class ConditionFamily(_message.Message):
    __slots__ = ("status", "reason")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    status: ConditionStatus
    reason: str
    def __init__(self, status: _Optional[_Union[ConditionStatus, str]] = ..., reason: _Optional[str] = ...) -> None: ...

class ServingCondition(_message.Message):
    __slots__ = ("family", "failure_rate", "degradation_rate", "latency_p50_ms", "latency_p95_ms", "probe_invocations", "organic_invocations", "synthetic_invocations")
    FAMILY_FIELD_NUMBER: _ClassVar[int]
    FAILURE_RATE_FIELD_NUMBER: _ClassVar[int]
    DEGRADATION_RATE_FIELD_NUMBER: _ClassVar[int]
    LATENCY_P50_MS_FIELD_NUMBER: _ClassVar[int]
    LATENCY_P95_MS_FIELD_NUMBER: _ClassVar[int]
    PROBE_INVOCATIONS_FIELD_NUMBER: _ClassVar[int]
    ORGANIC_INVOCATIONS_FIELD_NUMBER: _ClassVar[int]
    SYNTHETIC_INVOCATIONS_FIELD_NUMBER: _ClassVar[int]
    family: ConditionFamily
    failure_rate: float
    degradation_rate: float
    latency_p50_ms: int
    latency_p95_ms: int
    probe_invocations: int
    organic_invocations: int
    synthetic_invocations: int
    def __init__(self, family: _Optional[_Union[ConditionFamily, _Mapping]] = ..., failure_rate: _Optional[float] = ..., degradation_rate: _Optional[float] = ..., latency_p50_ms: _Optional[int] = ..., latency_p95_ms: _Optional[int] = ..., probe_invocations: _Optional[int] = ..., organic_invocations: _Optional[int] = ..., synthetic_invocations: _Optional[int] = ...) -> None: ...

class FreshnessCondition(_message.Message):
    __slots__ = ("family", "age_seconds", "drift_status", "drift_reason", "source_path", "source_mtime", "generation_mtime")
    FAMILY_FIELD_NUMBER: _ClassVar[int]
    AGE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    DRIFT_STATUS_FIELD_NUMBER: _ClassVar[int]
    DRIFT_REASON_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    SOURCE_MTIME_FIELD_NUMBER: _ClassVar[int]
    GENERATION_MTIME_FIELD_NUMBER: _ClassVar[int]
    family: ConditionFamily
    age_seconds: int
    drift_status: ConditionStatus
    drift_reason: str
    source_path: str
    source_mtime: str
    generation_mtime: str
    def __init__(self, family: _Optional[_Union[ConditionFamily, _Mapping]] = ..., age_seconds: _Optional[int] = ..., drift_status: _Optional[_Union[ConditionStatus, str]] = ..., drift_reason: _Optional[str] = ..., source_path: _Optional[str] = ..., source_mtime: _Optional[str] = ..., generation_mtime: _Optional[str] = ...) -> None: ...

class ExerciseCondition(_message.Message):
    __slots__ = ("family", "invocations", "distinct_callers", "last_invoked_at", "synthetic_invocations")
    FAMILY_FIELD_NUMBER: _ClassVar[int]
    INVOCATIONS_FIELD_NUMBER: _ClassVar[int]
    DISTINCT_CALLERS_FIELD_NUMBER: _ClassVar[int]
    LAST_INVOKED_AT_FIELD_NUMBER: _ClassVar[int]
    SYNTHETIC_INVOCATIONS_FIELD_NUMBER: _ClassVar[int]
    family: ConditionFamily
    invocations: int
    distinct_callers: int
    last_invoked_at: str
    synthetic_invocations: int
    def __init__(self, family: _Optional[_Union[ConditionFamily, _Mapping]] = ..., invocations: _Optional[int] = ..., distinct_callers: _Optional[int] = ..., last_invoked_at: _Optional[str] = ..., synthetic_invocations: _Optional[int] = ...) -> None: ...

class BindingCondition(_message.Message):
    __slots__ = ("binding_id", "scenario", "status", "verdict", "serving", "freshness", "exercise", "sustained_degradation", "sustained_degradation_reason")
    BINDING_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    SERVING_FIELD_NUMBER: _ClassVar[int]
    FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    EXERCISE_FIELD_NUMBER: _ClassVar[int]
    SUSTAINED_DEGRADATION_FIELD_NUMBER: _ClassVar[int]
    SUSTAINED_DEGRADATION_REASON_FIELD_NUMBER: _ClassVar[int]
    binding_id: str
    scenario: str
    status: ConditionStatus
    verdict: str
    serving: ServingCondition
    freshness: FreshnessCondition
    exercise: ExerciseCondition
    sustained_degradation: bool
    sustained_degradation_reason: str
    def __init__(self, binding_id: _Optional[str] = ..., scenario: _Optional[str] = ..., status: _Optional[_Union[ConditionStatus, str]] = ..., verdict: _Optional[str] = ..., serving: _Optional[_Union[ServingCondition, _Mapping]] = ..., freshness: _Optional[_Union[FreshnessCondition, _Mapping]] = ..., exercise: _Optional[_Union[ExerciseCondition, _Mapping]] = ..., sustained_degradation: _Optional[bool] = ..., sustained_degradation_reason: _Optional[str] = ...) -> None: ...

class ScenarioCondition(_message.Message):
    __slots__ = ("scenario", "status", "verdict", "binding_count", "dormant_bindings", "degraded_bindings", "healthy_bindings")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    BINDING_COUNT_FIELD_NUMBER: _ClassVar[int]
    DORMANT_BINDINGS_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_BINDINGS_FIELD_NUMBER: _ClassVar[int]
    HEALTHY_BINDINGS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    status: ConditionStatus
    verdict: str
    binding_count: int
    dormant_bindings: int
    degraded_bindings: int
    healthy_bindings: int
    def __init__(self, scenario: _Optional[str] = ..., status: _Optional[_Union[ConditionStatus, str]] = ..., verdict: _Optional[str] = ..., binding_count: _Optional[int] = ..., dormant_bindings: _Optional[int] = ..., degraded_bindings: _Optional[int] = ..., healthy_bindings: _Optional[int] = ...) -> None: ...

class GetBindingConditionRequest(_message.Message):
    __slots__ = ("binding_id", "scenario", "window_seconds")
    BINDING_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    WINDOW_SECONDS_FIELD_NUMBER: _ClassVar[int]
    binding_id: str
    scenario: str
    window_seconds: int
    def __init__(self, binding_id: _Optional[str] = ..., scenario: _Optional[str] = ..., window_seconds: _Optional[int] = ...) -> None: ...

class GetBindingConditionResponse(_message.Message):
    __slots__ = ("conditions", "window_seconds", "total_bindings", "instrumented_bindings", "scenario_conditions")
    CONDITIONS_FIELD_NUMBER: _ClassVar[int]
    WINDOW_SECONDS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_BINDINGS_FIELD_NUMBER: _ClassVar[int]
    INSTRUMENTED_BINDINGS_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_CONDITIONS_FIELD_NUMBER: _ClassVar[int]
    conditions: _containers.RepeatedCompositeFieldContainer[BindingCondition]
    window_seconds: int
    total_bindings: int
    instrumented_bindings: int
    scenario_conditions: _containers.RepeatedCompositeFieldContainer[ScenarioCondition]
    def __init__(self, conditions: _Optional[_Iterable[_Union[BindingCondition, _Mapping]]] = ..., window_seconds: _Optional[int] = ..., total_bindings: _Optional[int] = ..., instrumented_bindings: _Optional[int] = ..., scenario_conditions: _Optional[_Iterable[_Union[ScenarioCondition, _Mapping]]] = ...) -> None: ...

class DescribeBindingRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class BindingArgument(_message.Message):
    __slots__ = ("name", "proto_path", "kind", "required", "reason")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PROTO_PATH_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    REQUIRED_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    name: str
    proto_path: str
    kind: str
    required: bool
    reason: str
    def __init__(self, name: _Optional[str] = ..., proto_path: _Optional[str] = ..., kind: _Optional[str] = ..., required: _Optional[bool] = ..., reason: _Optional[str] = ...) -> None: ...

class DescribeBindingResponse(_message.Message):
    __slots__ = ("binding", "resolved_source", "callable", "arguments")
    BINDING_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_SOURCE_FIELD_NUMBER: _ClassVar[int]
    CALLABLE_FIELD_NUMBER: _ClassVar[int]
    ARGUMENTS_FIELD_NUMBER: _ClassVar[int]
    binding: Binding
    resolved_source: str
    callable: bool
    arguments: _containers.RepeatedCompositeFieldContainer[BindingArgument]
    def __init__(self, binding: _Optional[_Union[Binding, _Mapping]] = ..., resolved_source: _Optional[str] = ..., callable: _Optional[bool] = ..., arguments: _Optional[_Iterable[_Union[BindingArgument, _Mapping]]] = ...) -> None: ...

class ActCell(_message.Message):
    __slots__ = ("id", "operations", "authored_status")
    ID_FIELD_NUMBER: _ClassVar[int]
    OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    AUTHORED_STATUS_FIELD_NUMBER: _ClassVar[int]
    id: str
    operations: _containers.RepeatedScalarFieldContainer[str]
    authored_status: str
    def __init__(self, id: _Optional[str] = ..., operations: _Optional[_Iterable[str]] = ..., authored_status: _Optional[str] = ...) -> None: ...

class ActCellVerdict(_message.Message):
    __slots__ = ("id", "verdict", "resolved_operations", "unresolved_operations", "reasons", "authored_status", "audited")
    ID_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    UNRESOLVED_OPERATIONS_FIELD_NUMBER: _ClassVar[int]
    REASONS_FIELD_NUMBER: _ClassVar[int]
    AUTHORED_STATUS_FIELD_NUMBER: _ClassVar[int]
    AUDITED_FIELD_NUMBER: _ClassVar[int]
    id: str
    verdict: ActVerdict
    resolved_operations: _containers.RepeatedScalarFieldContainer[str]
    unresolved_operations: _containers.RepeatedScalarFieldContainer[str]
    reasons: _containers.RepeatedScalarFieldContainer[str]
    authored_status: str
    audited: bool
    def __init__(self, id: _Optional[str] = ..., verdict: _Optional[_Union[ActVerdict, str]] = ..., resolved_operations: _Optional[_Iterable[str]] = ..., unresolved_operations: _Optional[_Iterable[str]] = ..., reasons: _Optional[_Iterable[str]] = ..., authored_status: _Optional[str] = ..., audited: _Optional[bool] = ...) -> None: ...

class ResolveActCellsRequest(_message.Message):
    __slots__ = ("cells",)
    CELLS_FIELD_NUMBER: _ClassVar[int]
    cells: _containers.RepeatedCompositeFieldContainer[ActCell]
    def __init__(self, cells: _Optional[_Iterable[_Union[ActCell, _Mapping]]] = ...) -> None: ...

class ResolveActCellsResponse(_message.Message):
    __slots__ = ("cells", "audited_cells", "total_cells", "denominator_confidence")
    CELLS_FIELD_NUMBER: _ClassVar[int]
    AUDITED_CELLS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_CELLS_FIELD_NUMBER: _ClassVar[int]
    DENOMINATOR_CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    cells: _containers.RepeatedCompositeFieldContainer[ActCellVerdict]
    audited_cells: int
    total_cells: int
    denominator_confidence: str
    def __init__(self, cells: _Optional[_Iterable[_Union[ActCellVerdict, _Mapping]]] = ..., audited_cells: _Optional[int] = ..., total_cells: _Optional[int] = ..., denominator_confidence: _Optional[str] = ...) -> None: ...

class ResolveIntentRequest(_message.Message):
    __slots__ = ("intent", "limit", "mode")
    INTENT_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    intent: str
    limit: int
    mode: str
    def __init__(self, intent: _Optional[str] = ..., limit: _Optional[int] = ..., mode: _Optional[str] = ...) -> None: ...

class DiscoverResult(_message.Message):
    __slots__ = ("binding_id", "confidence", "method", "reason", "alternatives", "binding", "arguments", "library", "unavailable")
    BINDING_ID_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    ALTERNATIVES_FIELD_NUMBER: _ClassVar[int]
    BINDING_FIELD_NUMBER: _ClassVar[int]
    ARGUMENTS_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_FIELD_NUMBER: _ClassVar[int]
    binding_id: str
    confidence: str
    method: str
    reason: str
    alternatives: _containers.RepeatedScalarFieldContainer[str]
    binding: Binding
    arguments: _containers.RepeatedCompositeFieldContainer[BindingArgument]
    library: _library_pb2.LibraryProgram
    unavailable: bool
    def __init__(self, binding_id: _Optional[str] = ..., confidence: _Optional[str] = ..., method: _Optional[str] = ..., reason: _Optional[str] = ..., alternatives: _Optional[_Iterable[str]] = ..., binding: _Optional[_Union[Binding, _Mapping]] = ..., arguments: _Optional[_Iterable[_Union[BindingArgument, _Mapping]]] = ..., library: _Optional[_Union[_library_pb2.LibraryProgram, _Mapping]] = ..., unavailable: _Optional[bool] = ...) -> None: ...

class ResolveIntentResponse(_message.Message):
    __slots__ = ("bindings", "reason", "fallback", "result", "mode")
    BINDINGS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    bindings: _containers.RepeatedCompositeFieldContainer[Binding]
    reason: str
    fallback: bool
    result: DiscoverResult
    mode: str
    def __init__(self, bindings: _Optional[_Iterable[_Union[Binding, _Mapping]]] = ..., reason: _Optional[str] = ..., fallback: _Optional[bool] = ..., result: _Optional[_Union[DiscoverResult, _Mapping]] = ..., mode: _Optional[str] = ...) -> None: ...
