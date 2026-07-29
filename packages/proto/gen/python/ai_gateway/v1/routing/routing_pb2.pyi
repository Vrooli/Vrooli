from ai_gateway.v1.shared import gateway_pb2 as _gateway_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class MediaExecutionStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    MEDIA_EXECUTION_STATUS_UNSPECIFIED: _ClassVar[MediaExecutionStatus]
    MEDIA_EXECUTION_STATUS_QUEUED: _ClassVar[MediaExecutionStatus]
    MEDIA_EXECUTION_STATUS_RUNNING: _ClassVar[MediaExecutionStatus]
    MEDIA_EXECUTION_STATUS_SUCCEEDED: _ClassVar[MediaExecutionStatus]
    MEDIA_EXECUTION_STATUS_FAILED: _ClassVar[MediaExecutionStatus]
    MEDIA_EXECUTION_STATUS_CANCELLED: _ClassVar[MediaExecutionStatus]
MEDIA_EXECUTION_STATUS_UNSPECIFIED: MediaExecutionStatus
MEDIA_EXECUTION_STATUS_QUEUED: MediaExecutionStatus
MEDIA_EXECUTION_STATUS_RUNNING: MediaExecutionStatus
MEDIA_EXECUTION_STATUS_SUCCEEDED: MediaExecutionStatus
MEDIA_EXECUTION_STATUS_FAILED: MediaExecutionStatus
MEDIA_EXECUTION_STATUS_CANCELLED: MediaExecutionStatus

class RouteCandidate(_message.Message):
    __slots__ = ("provider", "role", "locality", "selected", "reasons", "fallback_eligible", "breaker_state", "half_open_probe", "rejection_reason", "capacity_verdict")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    LOCALITY_FIELD_NUMBER: _ClassVar[int]
    SELECTED_FIELD_NUMBER: _ClassVar[int]
    REASONS_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    BREAKER_STATE_FIELD_NUMBER: _ClassVar[int]
    HALF_OPEN_PROBE_FIELD_NUMBER: _ClassVar[int]
    REJECTION_REASON_FIELD_NUMBER: _ClassVar[int]
    CAPACITY_VERDICT_FIELD_NUMBER: _ClassVar[int]
    provider: str
    role: str
    locality: str
    selected: bool
    reasons: _containers.RepeatedScalarFieldContainer[str]
    fallback_eligible: bool
    breaker_state: str
    half_open_probe: bool
    rejection_reason: str
    capacity_verdict: str
    def __init__(self, provider: _Optional[str] = ..., role: _Optional[str] = ..., locality: _Optional[str] = ..., selected: _Optional[bool] = ..., reasons: _Optional[_Iterable[str]] = ..., fallback_eligible: _Optional[bool] = ..., breaker_state: _Optional[str] = ..., half_open_probe: _Optional[bool] = ..., rejection_reason: _Optional[str] = ..., capacity_verdict: _Optional[str] = ...) -> None: ...

class PreviewRouteRequest(_message.Message):
    __slots__ = ("request",)
    REQUEST_FIELD_NUMBER: _ClassVar[int]
    request: _gateway_pb2.GatewayRequest
    def __init__(self, request: _Optional[_Union[_gateway_pb2.GatewayRequest, _Mapping]] = ...) -> None: ...

class PreviewRouteResponse(_message.Message):
    __slots__ = ("valid", "issues", "candidates", "selected_provider", "policy_reasons", "fallback_allowed", "route_plan_id")
    VALID_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    CANDIDATES_FIELD_NUMBER: _ClassVar[int]
    SELECTED_PROVIDER_FIELD_NUMBER: _ClassVar[int]
    POLICY_REASONS_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_ALLOWED_FIELD_NUMBER: _ClassVar[int]
    ROUTE_PLAN_ID_FIELD_NUMBER: _ClassVar[int]
    valid: bool
    issues: _containers.RepeatedCompositeFieldContainer[_gateway_pb2.ValidationIssue]
    candidates: _containers.RepeatedCompositeFieldContainer[RouteCandidate]
    selected_provider: str
    policy_reasons: _containers.RepeatedScalarFieldContainer[str]
    fallback_allowed: bool
    route_plan_id: str
    def __init__(self, valid: _Optional[bool] = ..., issues: _Optional[_Iterable[_Union[_gateway_pb2.ValidationIssue, _Mapping]]] = ..., candidates: _Optional[_Iterable[_Union[RouteCandidate, _Mapping]]] = ..., selected_provider: _Optional[str] = ..., policy_reasons: _Optional[_Iterable[str]] = ..., fallback_allowed: _Optional[bool] = ..., route_plan_id: _Optional[str] = ...) -> None: ...

class ExecuteRouteRequest(_message.Message):
    __slots__ = ("request", "input_text")
    REQUEST_FIELD_NUMBER: _ClassVar[int]
    INPUT_TEXT_FIELD_NUMBER: _ClassVar[int]
    request: _gateway_pb2.GatewayRequest
    input_text: str
    def __init__(self, request: _Optional[_Union[_gateway_pb2.GatewayRequest, _Mapping]] = ..., input_text: _Optional[str] = ...) -> None: ...

class RouteEvidence(_message.Message):
    __slots__ = ("event_id", "request_id", "scenario", "operation", "role", "profile", "privacy_class", "selected_provider", "selected_locality", "status", "policy_reasons", "failure_reasons", "fallback_used", "prompt_redacted", "response_redacted", "latency_ms", "created_at", "breaker_state", "failure_class", "rejection_reason", "capacity_verdict", "capacity_claim_id", "capacity_required_bytes", "capacity_granted_bytes", "capacity_reclaim_required", "input_tokens", "output_tokens", "cost_estimate", "selected_model")
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    PRIVACY_CLASS_FIELD_NUMBER: _ClassVar[int]
    SELECTED_PROVIDER_FIELD_NUMBER: _ClassVar[int]
    SELECTED_LOCALITY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    POLICY_REASONS_FIELD_NUMBER: _ClassVar[int]
    FAILURE_REASONS_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_USED_FIELD_NUMBER: _ClassVar[int]
    PROMPT_REDACTED_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_REDACTED_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    BREAKER_STATE_FIELD_NUMBER: _ClassVar[int]
    FAILURE_CLASS_FIELD_NUMBER: _ClassVar[int]
    REJECTION_REASON_FIELD_NUMBER: _ClassVar[int]
    CAPACITY_VERDICT_FIELD_NUMBER: _ClassVar[int]
    CAPACITY_CLAIM_ID_FIELD_NUMBER: _ClassVar[int]
    CAPACITY_REQUIRED_BYTES_FIELD_NUMBER: _ClassVar[int]
    CAPACITY_GRANTED_BYTES_FIELD_NUMBER: _ClassVar[int]
    CAPACITY_RECLAIM_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    INPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    COST_ESTIMATE_FIELD_NUMBER: _ClassVar[int]
    SELECTED_MODEL_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    request_id: str
    scenario: str
    operation: str
    role: str
    profile: _gateway_pb2.Profile
    privacy_class: _gateway_pb2.PrivacyClass
    selected_provider: str
    selected_locality: str
    status: str
    policy_reasons: _containers.RepeatedScalarFieldContainer[str]
    failure_reasons: _containers.RepeatedScalarFieldContainer[str]
    fallback_used: bool
    prompt_redacted: bool
    response_redacted: bool
    latency_ms: int
    created_at: str
    breaker_state: str
    failure_class: str
    rejection_reason: str
    capacity_verdict: str
    capacity_claim_id: str
    capacity_required_bytes: int
    capacity_granted_bytes: int
    capacity_reclaim_required: bool
    input_tokens: int
    output_tokens: int
    cost_estimate: float
    selected_model: str
    def __init__(self, event_id: _Optional[str] = ..., request_id: _Optional[str] = ..., scenario: _Optional[str] = ..., operation: _Optional[str] = ..., role: _Optional[str] = ..., profile: _Optional[_Union[_gateway_pb2.Profile, str]] = ..., privacy_class: _Optional[_Union[_gateway_pb2.PrivacyClass, str]] = ..., selected_provider: _Optional[str] = ..., selected_locality: _Optional[str] = ..., status: _Optional[str] = ..., policy_reasons: _Optional[_Iterable[str]] = ..., failure_reasons: _Optional[_Iterable[str]] = ..., fallback_used: _Optional[bool] = ..., prompt_redacted: _Optional[bool] = ..., response_redacted: _Optional[bool] = ..., latency_ms: _Optional[int] = ..., created_at: _Optional[str] = ..., breaker_state: _Optional[str] = ..., failure_class: _Optional[str] = ..., rejection_reason: _Optional[str] = ..., capacity_verdict: _Optional[str] = ..., capacity_claim_id: _Optional[str] = ..., capacity_required_bytes: _Optional[int] = ..., capacity_granted_bytes: _Optional[int] = ..., capacity_reclaim_required: _Optional[bool] = ..., input_tokens: _Optional[int] = ..., output_tokens: _Optional[int] = ..., cost_estimate: _Optional[float] = ..., selected_model: _Optional[str] = ...) -> None: ...

class ExecuteRouteResponse(_message.Message):
    __slots__ = ("valid", "issues", "evidence", "output_text", "policy_reasons")
    VALID_FIELD_NUMBER: _ClassVar[int]
    ISSUES_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_TEXT_FIELD_NUMBER: _ClassVar[int]
    POLICY_REASONS_FIELD_NUMBER: _ClassVar[int]
    valid: bool
    issues: _containers.RepeatedCompositeFieldContainer[_gateway_pb2.ValidationIssue]
    evidence: RouteEvidence
    output_text: str
    policy_reasons: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, valid: _Optional[bool] = ..., issues: _Optional[_Iterable[_Union[_gateway_pb2.ValidationIssue, _Mapping]]] = ..., evidence: _Optional[_Union[RouteEvidence, _Mapping]] = ..., output_text: _Optional[str] = ..., policy_reasons: _Optional[_Iterable[str]] = ...) -> None: ...

class MediaInput(_message.Message):
    __slots__ = ("reference", "media_type")
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    reference: str
    media_type: str
    def __init__(self, reference: _Optional[str] = ..., media_type: _Optional[str] = ...) -> None: ...

class MediaOutput(_message.Message):
    __slots__ = ("reference", "media_type", "bytes", "checksum")
    REFERENCE_FIELD_NUMBER: _ClassVar[int]
    MEDIA_TYPE_FIELD_NUMBER: _ClassVar[int]
    BYTES_FIELD_NUMBER: _ClassVar[int]
    CHECKSUM_FIELD_NUMBER: _ClassVar[int]
    reference: str
    media_type: str
    bytes: int
    checksum: str
    def __init__(self, reference: _Optional[str] = ..., media_type: _Optional[str] = ..., bytes: _Optional[int] = ..., checksum: _Optional[str] = ...) -> None: ...

class SubmitMediaRequest(_message.Message):
    __slots__ = ("request", "prompt", "inputs", "output_count", "idempotency_key")
    REQUEST_FIELD_NUMBER: _ClassVar[int]
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    INPUTS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_COUNT_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    request: _gateway_pb2.GatewayRequest
    prompt: str
    inputs: _containers.RepeatedCompositeFieldContainer[MediaInput]
    output_count: int
    idempotency_key: str
    def __init__(self, request: _Optional[_Union[_gateway_pb2.GatewayRequest, _Mapping]] = ..., prompt: _Optional[str] = ..., inputs: _Optional[_Iterable[_Union[MediaInput, _Mapping]]] = ..., output_count: _Optional[int] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class MediaExecution(_message.Message):
    __slots__ = ("execution_id", "idempotency_key", "status", "created_at", "started_at", "completed_at", "route_evidence", "outputs", "actual_cost_usd", "resolved_model", "seed", "warnings", "error_code", "error_message")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    STARTED_AT_FIELD_NUMBER: _ClassVar[int]
    COMPLETED_AT_FIELD_NUMBER: _ClassVar[int]
    ROUTE_EVIDENCE_FIELD_NUMBER: _ClassVar[int]
    OUTPUTS_FIELD_NUMBER: _ClassVar[int]
    ACTUAL_COST_USD_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_MODEL_FIELD_NUMBER: _ClassVar[int]
    SEED_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    ERROR_CODE_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    idempotency_key: str
    status: MediaExecutionStatus
    created_at: str
    started_at: str
    completed_at: str
    route_evidence: RouteEvidence
    outputs: _containers.RepeatedCompositeFieldContainer[MediaOutput]
    actual_cost_usd: float
    resolved_model: str
    seed: str
    warnings: _containers.RepeatedScalarFieldContainer[str]
    error_code: str
    error_message: str
    def __init__(self, execution_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ..., status: _Optional[_Union[MediaExecutionStatus, str]] = ..., created_at: _Optional[str] = ..., started_at: _Optional[str] = ..., completed_at: _Optional[str] = ..., route_evidence: _Optional[_Union[RouteEvidence, _Mapping]] = ..., outputs: _Optional[_Iterable[_Union[MediaOutput, _Mapping]]] = ..., actual_cost_usd: _Optional[float] = ..., resolved_model: _Optional[str] = ..., seed: _Optional[str] = ..., warnings: _Optional[_Iterable[str]] = ..., error_code: _Optional[str] = ..., error_message: _Optional[str] = ...) -> None: ...

class SubmitMediaResponse(_message.Message):
    __slots__ = ("execution",)
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    execution: MediaExecution
    def __init__(self, execution: _Optional[_Union[MediaExecution, _Mapping]] = ...) -> None: ...

class GetMediaExecutionRequest(_message.Message):
    __slots__ = ("execution_id",)
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    def __init__(self, execution_id: _Optional[str] = ...) -> None: ...

class GetMediaExecutionResponse(_message.Message):
    __slots__ = ("execution",)
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    execution: MediaExecution
    def __init__(self, execution: _Optional[_Union[MediaExecution, _Mapping]] = ...) -> None: ...

class WaitMediaExecutionRequest(_message.Message):
    __slots__ = ("execution_id",)
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    def __init__(self, execution_id: _Optional[str] = ...) -> None: ...

class WaitMediaExecutionResponse(_message.Message):
    __slots__ = ("execution",)
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    execution: MediaExecution
    def __init__(self, execution: _Optional[_Union[MediaExecution, _Mapping]] = ...) -> None: ...

class CancelMediaExecutionRequest(_message.Message):
    __slots__ = ("execution_id",)
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    def __init__(self, execution_id: _Optional[str] = ...) -> None: ...

class CancelMediaExecutionResponse(_message.Message):
    __slots__ = ("execution",)
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    execution: MediaExecution
    def __init__(self, execution: _Optional[_Union[MediaExecution, _Mapping]] = ...) -> None: ...

class RetryMediaExecutionRequest(_message.Message):
    __slots__ = ("execution_id", "idempotency_key")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    idempotency_key: str
    def __init__(self, execution_id: _Optional[str] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class RetryMediaExecutionResponse(_message.Message):
    __slots__ = ("execution",)
    EXECUTION_FIELD_NUMBER: _ClassVar[int]
    execution: MediaExecution
    def __init__(self, execution: _Optional[_Union[MediaExecution, _Mapping]] = ...) -> None: ...

class ListRouteEvidenceRequest(_message.Message):
    __slots__ = ("limit", "scenario")
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    limit: int
    scenario: str
    def __init__(self, limit: _Optional[int] = ..., scenario: _Optional[str] = ...) -> None: ...

class ListRouteEvidenceResponse(_message.Message):
    __slots__ = ("events",)
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    events: _containers.RepeatedCompositeFieldContainer[RouteEvidence]
    def __init__(self, events: _Optional[_Iterable[_Union[RouteEvidence, _Mapping]]] = ...) -> None: ...

class GetRouteEvidenceRequest(_message.Message):
    __slots__ = ("event_id",)
    EVENT_ID_FIELD_NUMBER: _ClassVar[int]
    event_id: str
    def __init__(self, event_id: _Optional[str] = ...) -> None: ...

class GetRouteEvidenceResponse(_message.Message):
    __slots__ = ("event",)
    EVENT_FIELD_NUMBER: _ClassVar[int]
    event: RouteEvidence
    def __init__(self, event: _Optional[_Union[RouteEvidence, _Mapping]] = ...) -> None: ...

class ProviderHealth(_message.Message):
    __slots__ = ("provider", "role", "kind", "state", "effective_state", "consecutive_failures", "last_failure_class", "last_success_at", "last_failure_at", "cooldown_until", "opened_at", "generation", "updated_at")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_STATE_FIELD_NUMBER: _ClassVar[int]
    CONSECUTIVE_FAILURES_FIELD_NUMBER: _ClassVar[int]
    LAST_FAILURE_CLASS_FIELD_NUMBER: _ClassVar[int]
    LAST_SUCCESS_AT_FIELD_NUMBER: _ClassVar[int]
    LAST_FAILURE_AT_FIELD_NUMBER: _ClassVar[int]
    COOLDOWN_UNTIL_FIELD_NUMBER: _ClassVar[int]
    OPENED_AT_FIELD_NUMBER: _ClassVar[int]
    GENERATION_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    provider: str
    role: str
    kind: _gateway_pb2.RequestKind
    state: str
    effective_state: str
    consecutive_failures: int
    last_failure_class: str
    last_success_at: str
    last_failure_at: str
    cooldown_until: str
    opened_at: str
    generation: int
    updated_at: str
    def __init__(self, provider: _Optional[str] = ..., role: _Optional[str] = ..., kind: _Optional[_Union[_gateway_pb2.RequestKind, str]] = ..., state: _Optional[str] = ..., effective_state: _Optional[str] = ..., consecutive_failures: _Optional[int] = ..., last_failure_class: _Optional[str] = ..., last_success_at: _Optional[str] = ..., last_failure_at: _Optional[str] = ..., cooldown_until: _Optional[str] = ..., opened_at: _Optional[str] = ..., generation: _Optional[int] = ..., updated_at: _Optional[str] = ...) -> None: ...

class ListProviderHealthRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListProviderHealthResponse(_message.Message):
    __slots__ = ("items",)
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[ProviderHealth]
    def __init__(self, items: _Optional[_Iterable[_Union[ProviderHealth, _Mapping]]] = ...) -> None: ...
