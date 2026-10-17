from ai_gateway.v1.shared import gateway_pb2 as _gateway_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

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
INFERENCE_ERROR_CODE_UNSPECIFIED: InferenceErrorCode
INFERENCE_ERROR_CODE_UNAVAILABLE: InferenceErrorCode
INFERENCE_ERROR_CODE_INVALID_REQUEST: InferenceErrorCode
INFERENCE_ERROR_CODE_UNSUPPORTED_SCHEMA: InferenceErrorCode
INFERENCE_ERROR_CODE_VALIDATION_FAILED: InferenceErrorCode
INFERENCE_ERROR_CODE_PROVIDER_FAILED: InferenceErrorCode
INFERENCE_ERROR_CODE_UNSUPPORTED_SAMPLING: InferenceErrorCode
INFERENCE_ERROR_CODE_CONTEXT_OVERFLOW: InferenceErrorCode

class Usage(_message.Message):
    __slots__ = ("input_tokens", "output_tokens", "cost_micros")
    INPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    COST_MICROS_FIELD_NUMBER: _ClassVar[int]
    input_tokens: int
    output_tokens: int
    cost_micros: int
    def __init__(self, input_tokens: _Optional[int] = ..., output_tokens: _Optional[int] = ..., cost_micros: _Optional[int] = ...) -> None: ...

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

class RunRequest(_message.Message):
    __slots__ = ("source", "schema_json", "instruction", "role", "turns", "attachments", "profile", "sampling", "max_output_tokens")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    SCHEMA_JSON_FIELD_NUMBER: _ClassVar[int]
    INSTRUCTION_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    TURNS_FIELD_NUMBER: _ClassVar[int]
    ATTACHMENTS_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    SAMPLING_FIELD_NUMBER: _ClassVar[int]
    MAX_OUTPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    source: str
    schema_json: str
    instruction: str
    role: str
    turns: _containers.RepeatedCompositeFieldContainer[Turn]
    attachments: _containers.RepeatedCompositeFieldContainer[_gateway_pb2.Attachment]
    profile: _gateway_pb2.Profile
    sampling: _gateway_pb2.SamplingControls
    max_output_tokens: int
    def __init__(self, source: _Optional[str] = ..., schema_json: _Optional[str] = ..., instruction: _Optional[str] = ..., role: _Optional[str] = ..., turns: _Optional[_Iterable[_Union[Turn, _Mapping]]] = ..., attachments: _Optional[_Iterable[_Union[_gateway_pb2.Attachment, _Mapping]]] = ..., profile: _Optional[_Union[_gateway_pb2.Profile, str]] = ..., sampling: _Optional[_Union[_gateway_pb2.SamplingControls, _Mapping]] = ..., max_output_tokens: _Optional[int] = ...) -> None: ...

class RunResponse(_message.Message):
    __slots__ = ("value_json", "provider", "model", "validated", "usage", "error", "applied")
    VALUE_JSON_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    VALIDATED_FIELD_NUMBER: _ClassVar[int]
    USAGE_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    APPLIED_FIELD_NUMBER: _ClassVar[int]
    value_json: str
    provider: str
    model: str
    validated: bool
    usage: Usage
    error: InferenceError
    applied: _gateway_pb2.AppliedSettings
    def __init__(self, value_json: _Optional[str] = ..., provider: _Optional[str] = ..., model: _Optional[str] = ..., validated: _Optional[bool] = ..., usage: _Optional[_Union[Usage, _Mapping]] = ..., error: _Optional[_Union[InferenceError, _Mapping]] = ..., applied: _Optional[_Union[_gateway_pb2.AppliedSettings, _Mapping]] = ...) -> None: ...

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
