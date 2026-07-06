from ai_gateway.v1.shared import gateway_pb2 as _gateway_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RouteCandidate(_message.Message):
    __slots__ = ("provider", "role", "locality", "selected", "reasons", "fallback_eligible")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    LOCALITY_FIELD_NUMBER: _ClassVar[int]
    SELECTED_FIELD_NUMBER: _ClassVar[int]
    REASONS_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    provider: str
    role: str
    locality: str
    selected: bool
    reasons: _containers.RepeatedScalarFieldContainer[str]
    fallback_eligible: bool
    def __init__(self, provider: _Optional[str] = ..., role: _Optional[str] = ..., locality: _Optional[str] = ..., selected: _Optional[bool] = ..., reasons: _Optional[_Iterable[str]] = ..., fallback_eligible: _Optional[bool] = ...) -> None: ...

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
    __slots__ = ("event_id", "request_id", "scenario", "operation", "role", "profile", "privacy_class", "selected_provider", "selected_locality", "status", "policy_reasons", "failure_reasons", "fallback_used", "prompt_redacted", "response_redacted", "latency_ms", "created_at")
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
    def __init__(self, event_id: _Optional[str] = ..., request_id: _Optional[str] = ..., scenario: _Optional[str] = ..., operation: _Optional[str] = ..., role: _Optional[str] = ..., profile: _Optional[_Union[_gateway_pb2.Profile, str]] = ..., privacy_class: _Optional[_Union[_gateway_pb2.PrivacyClass, str]] = ..., selected_provider: _Optional[str] = ..., selected_locality: _Optional[str] = ..., status: _Optional[str] = ..., policy_reasons: _Optional[_Iterable[str]] = ..., failure_reasons: _Optional[_Iterable[str]] = ..., fallback_used: _Optional[bool] = ..., prompt_redacted: _Optional[bool] = ..., response_redacted: _Optional[bool] = ..., latency_ms: _Optional[int] = ..., created_at: _Optional[str] = ...) -> None: ...

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
