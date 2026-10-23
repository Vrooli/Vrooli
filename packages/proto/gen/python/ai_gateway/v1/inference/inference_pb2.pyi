from ai_gateway.v1.shared import gateway_pb2 as _gateway_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class UsageProvenance(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    USAGE_PROVENANCE_UNSPECIFIED: _ClassVar[UsageProvenance]
    USAGE_PROVENANCE_UNKNOWN: _ClassVar[UsageProvenance]
    USAGE_PROVENANCE_PROVIDER_REPORTED: _ClassVar[UsageProvenance]
    USAGE_PROVENANCE_GATEWAY_ESTIMATE: _ClassVar[UsageProvenance]
    USAGE_PROVENANCE_MIXED: _ClassVar[UsageProvenance]

class InferenceErrorCode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    INFERENCE_ERROR_CODE_UNSPECIFIED: _ClassVar[InferenceErrorCode]
    INFERENCE_ERROR_CODE_UNAVAILABLE: _ClassVar[InferenceErrorCode]
    INFERENCE_ERROR_CODE_INVALID_REQUEST: _ClassVar[InferenceErrorCode]
    INFERENCE_ERROR_CODE_UNSUPPORTED_SCHEMA: _ClassVar[InferenceErrorCode]
    INFERENCE_ERROR_CODE_VALIDATION_FAILED: _ClassVar[InferenceErrorCode]
    INFERENCE_ERROR_CODE_PROVIDER_FAILED: _ClassVar[InferenceErrorCode]
    INFERENCE_ERROR_CODE_UNSUPPORTED_SAMPLING: _ClassVar[InferenceErrorCode]
    INFERENCE_ERROR_CODE_CONTEXT_OVERFLOW: _ClassVar[InferenceErrorCode]
USAGE_PROVENANCE_UNSPECIFIED: UsageProvenance
USAGE_PROVENANCE_UNKNOWN: UsageProvenance
USAGE_PROVENANCE_PROVIDER_REPORTED: UsageProvenance
USAGE_PROVENANCE_GATEWAY_ESTIMATE: UsageProvenance
USAGE_PROVENANCE_MIXED: UsageProvenance
INFERENCE_ERROR_CODE_UNSPECIFIED: InferenceErrorCode
INFERENCE_ERROR_CODE_UNAVAILABLE: InferenceErrorCode
INFERENCE_ERROR_CODE_INVALID_REQUEST: InferenceErrorCode
INFERENCE_ERROR_CODE_UNSUPPORTED_SCHEMA: InferenceErrorCode
INFERENCE_ERROR_CODE_VALIDATION_FAILED: InferenceErrorCode
INFERENCE_ERROR_CODE_PROVIDER_FAILED: InferenceErrorCode
INFERENCE_ERROR_CODE_UNSUPPORTED_SAMPLING: InferenceErrorCode
INFERENCE_ERROR_CODE_CONTEXT_OVERFLOW: InferenceErrorCode

class Usage(_message.Message):
    __slots__ = ("input_tokens", "output_tokens", "cost_micros", "input_tokens_provenance", "output_tokens_provenance", "cost_provenance", "credits_charged", "accounting_status")
    INPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    COST_MICROS_FIELD_NUMBER: _ClassVar[int]
    INPUT_TOKENS_PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_TOKENS_PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    COST_PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    CREDITS_CHARGED_FIELD_NUMBER: _ClassVar[int]
    ACCOUNTING_STATUS_FIELD_NUMBER: _ClassVar[int]
    input_tokens: int
    output_tokens: int
    cost_micros: int
    input_tokens_provenance: UsageProvenance
    output_tokens_provenance: UsageProvenance
    cost_provenance: UsageProvenance
    credits_charged: int
    accounting_status: str
    def __init__(self, input_tokens: _Optional[int] = ..., output_tokens: _Optional[int] = ..., cost_micros: _Optional[int] = ..., input_tokens_provenance: _Optional[_Union[UsageProvenance, str]] = ..., output_tokens_provenance: _Optional[_Union[UsageProvenance, str]] = ..., cost_provenance: _Optional[_Union[UsageProvenance, str]] = ..., credits_charged: _Optional[int] = ..., accounting_status: _Optional[str] = ...) -> None: ...

class InferenceError(_message.Message):
    __slots__ = ("code", "message", "construct")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    CONSTRUCT_FIELD_NUMBER: _ClassVar[int]
    code: InferenceErrorCode
    message: str
    construct: str
    def __init__(self, code: _Optional[_Union[InferenceErrorCode, str]] = ..., message: _Optional[str] = ..., construct: _Optional[str] = ...) -> None: ...

class Turn(_message.Message):
    __slots__ = ("role", "text", "attachments")
    ROLE_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENTS_FIELD_NUMBER: _ClassVar[int]
    role: str
    text: str
    attachments: _containers.RepeatedCompositeFieldContainer[_gateway_pb2.Attachment]
    def __init__(self, role: _Optional[str] = ..., text: _Optional[str] = ..., attachments: _Optional[_Iterable[_Union[_gateway_pb2.Attachment, _Mapping]]] = ...) -> None: ...

class SpendLimit(_message.Message):
    __slots__ = ("max_cost_micros", "operation_id", "input_fingerprint", "reference_sha256", "image_count", "funding_route")
    MAX_COST_MICROS_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    INPUT_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_SHA256_FIELD_NUMBER: _ClassVar[int]
    IMAGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    FUNDING_ROUTE_FIELD_NUMBER: _ClassVar[int]
    max_cost_micros: int
    operation_id: str
    input_fingerprint: str
    reference_sha256: _containers.RepeatedScalarFieldContainer[str]
    image_count: int
    funding_route: str
    def __init__(self, max_cost_micros: _Optional[int] = ..., operation_id: _Optional[str] = ..., input_fingerprint: _Optional[str] = ..., reference_sha256: _Optional[_Iterable[str]] = ..., image_count: _Optional[int] = ..., funding_route: _Optional[str] = ...) -> None: ...

class SpendAuthorization(_message.Message):
    __slots__ = ("authorization_id", "status", "max_cost_micros", "input_tokens_bound", "output_tokens_bound", "image_count", "context_source", "operation_id", "input_fingerprint", "reference_sha256", "funding_route", "account_identity", "model", "provider_controls_sha256", "context_window_tokens")
    AUTHORIZATION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MAX_COST_MICROS_FIELD_NUMBER: _ClassVar[int]
    INPUT_TOKENS_BOUND_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_TOKENS_BOUND_FIELD_NUMBER: _ClassVar[int]
    IMAGE_COUNT_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_SOURCE_FIELD_NUMBER: _ClassVar[int]
    OPERATION_ID_FIELD_NUMBER: _ClassVar[int]
    INPUT_FINGERPRINT_FIELD_NUMBER: _ClassVar[int]
    REFERENCE_SHA256_FIELD_NUMBER: _ClassVar[int]
    FUNDING_ROUTE_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_CONTROLS_SHA256_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_WINDOW_TOKENS_FIELD_NUMBER: _ClassVar[int]
    authorization_id: str
    status: str
    max_cost_micros: int
    input_tokens_bound: int
    output_tokens_bound: int
    image_count: int
    context_source: str
    operation_id: str
    input_fingerprint: str
    reference_sha256: _containers.RepeatedScalarFieldContainer[str]
    funding_route: str
    account_identity: str
    model: str
    provider_controls_sha256: str
    context_window_tokens: int
    def __init__(self, authorization_id: _Optional[str] = ..., status: _Optional[str] = ..., max_cost_micros: _Optional[int] = ..., input_tokens_bound: _Optional[int] = ..., output_tokens_bound: _Optional[int] = ..., image_count: _Optional[int] = ..., context_source: _Optional[str] = ..., operation_id: _Optional[str] = ..., input_fingerprint: _Optional[str] = ..., reference_sha256: _Optional[_Iterable[str]] = ..., funding_route: _Optional[str] = ..., account_identity: _Optional[str] = ..., model: _Optional[str] = ..., provider_controls_sha256: _Optional[str] = ..., context_window_tokens: _Optional[int] = ...) -> None: ...

class RunRequest(_message.Message):
    __slots__ = ("source", "schema_json", "instruction", "role", "turns", "attachments", "profile", "sampling", "max_output_tokens", "max_provider_calls", "spend_limit")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_JSON_FIELD_NUMBER: _ClassVar[int]
    INSTRUCTION_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    TURNS_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENTS_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    SAMPLING_FIELD_NUMBER: _ClassVar[int]
    MAX_OUTPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    MAX_PROVIDER_CALLS_FIELD_NUMBER: _ClassVar[int]
    SPEND_LIMIT_FIELD_NUMBER: _ClassVar[int]
    source: str
    schema_json: str
    instruction: str
    role: str
    turns: _containers.RepeatedCompositeFieldContainer[Turn]
    attachments: _containers.RepeatedCompositeFieldContainer[_gateway_pb2.Attachment]
    profile: _gateway_pb2.Profile
    sampling: _gateway_pb2.SamplingControls
    max_output_tokens: int
    max_provider_calls: int
    spend_limit: SpendLimit
    def __init__(self, source: _Optional[str] = ..., schema_json: _Optional[str] = ..., instruction: _Optional[str] = ..., role: _Optional[str] = ..., turns: _Optional[_Iterable[_Union[Turn, _Mapping]]] = ..., attachments: _Optional[_Iterable[_Union[_gateway_pb2.Attachment, _Mapping]]] = ..., profile: _Optional[_Union[_gateway_pb2.Profile, str]] = ..., sampling: _Optional[_Union[_gateway_pb2.SamplingControls, _Mapping]] = ..., max_output_tokens: _Optional[int] = ..., max_provider_calls: _Optional[int] = ..., spend_limit: _Optional[_Union[SpendLimit, _Mapping]] = ...) -> None: ...

class RunResponse(_message.Message):
    __slots__ = ("value_json", "provider", "model", "validated", "usage", "error", "applied", "provider_calls_made", "spend_authorization")
    VALUE_JSON_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    VALIDATED_FIELD_NUMBER: _ClassVar[int]
    USAGE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_CALLS_MADE_FIELD_NUMBER: _ClassVar[int]
    SPEND_AUTHORIZATION_FIELD_NUMBER: _ClassVar[int]
    value_json: str
    provider: str
    model: str
    validated: bool
    usage: Usage
    error: InferenceError
    applied: _gateway_pb2.AppliedSettings
    provider_calls_made: int
    spend_authorization: SpendAuthorization
    def __init__(self, value_json: _Optional[str] = ..., provider: _Optional[str] = ..., model: _Optional[str] = ..., validated: _Optional[bool] = ..., usage: _Optional[_Union[Usage, _Mapping]] = ..., error: _Optional[_Union[InferenceError, _Mapping]] = ..., applied: _Optional[_Union[_gateway_pb2.AppliedSettings, _Mapping]] = ..., provider_calls_made: _Optional[int] = ..., spend_authorization: _Optional[_Union[SpendAuthorization, _Mapping]] = ...) -> None: ...

class RunBatchItem(_message.Message):
    __slots__ = ("source",)
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    source: str
    def __init__(self, source: _Optional[str] = ...) -> None: ...

class RunBatchRequest(_message.Message):
    __slots__ = ("items", "schema_json", "instruction", "role")
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_JSON_FIELD_NUMBER: _ClassVar[int]
    INSTRUCTION_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[RunBatchItem]
    schema_json: str
    instruction: str
    role: str
    def __init__(self, items: _Optional[_Iterable[_Union[RunBatchItem, _Mapping]]] = ..., schema_json: _Optional[str] = ..., instruction: _Optional[str] = ..., role: _Optional[str] = ...) -> None: ...

class RunBatchResponse(_message.Message):
    __slots__ = ("results", "usage")
    RESULTS_FIELD_NUMBER: _ClassVar[int]
    USAGE_FIELD_NUMBER: _ClassVar[int]
    results: _containers.RepeatedCompositeFieldContainer[RunResponse]
    usage: Usage
    def __init__(self, results: _Optional[_Iterable[_Union[RunResponse, _Mapping]]] = ..., usage: _Optional[_Union[Usage, _Mapping]] = ...) -> None: ...

class EmbedRequest(_message.Message):
    __slots__ = ("texts", "role", "sampling")
    TEXTS_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    SAMPLING_FIELD_NUMBER: _ClassVar[int]
    texts: _containers.RepeatedScalarFieldContainer[str]
    role: str
    sampling: _gateway_pb2.SamplingControls
    def __init__(self, texts: _Optional[_Iterable[str]] = ..., role: _Optional[str] = ..., sampling: _Optional[_Union[_gateway_pb2.SamplingControls, _Mapping]] = ...) -> None: ...

class EmbedResponse(_message.Message):
    __slots__ = ("vectors", "provider", "model", "dimension", "usage", "error")
    VECTORS_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    DIMENSION_FIELD_NUMBER: _ClassVar[int]
    USAGE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    vectors: _containers.RepeatedCompositeFieldContainer[EmbeddingVector]
    provider: str
    model: str
    dimension: int
    usage: Usage
    error: InferenceError
    def __init__(self, vectors: _Optional[_Iterable[_Union[EmbeddingVector, _Mapping]]] = ..., provider: _Optional[str] = ..., model: _Optional[str] = ..., dimension: _Optional[int] = ..., usage: _Optional[_Union[Usage, _Mapping]] = ..., error: _Optional[_Union[InferenceError, _Mapping]] = ...) -> None: ...

class EmbeddingVector(_message.Message):
    __slots__ = ("values",)
    VALUES_FIELD_NUMBER: _ClassVar[int]
    values: _containers.RepeatedScalarFieldContainer[float]
    def __init__(self, values: _Optional[_Iterable[float]] = ...) -> None: ...
