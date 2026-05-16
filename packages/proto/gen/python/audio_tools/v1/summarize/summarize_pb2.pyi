from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class SummarizeRequest(_message.Message):
    __slots__ = ("text", "level", "model", "timeout_seconds")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    text: str
    level: str
    model: str
    timeout_seconds: int
    def __init__(self, text: _Optional[str] = ..., level: _Optional[str] = ..., model: _Optional[str] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class SummarizeResponse(_message.Message):
    __slots__ = ("text", "prompt_tokens", "output_tokens", "provider_tier", "provider_id", "model_id", "latency_ms")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    PROMPT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_TIER_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_ID_FIELD_NUMBER: _ClassVar[int]
    MODEL_ID_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    text: str
    prompt_tokens: int
    output_tokens: int
    provider_tier: str
    provider_id: str
    model_id: str
    latency_ms: float
    def __init__(self, text: _Optional[str] = ..., prompt_tokens: _Optional[int] = ..., output_tokens: _Optional[int] = ..., provider_tier: _Optional[str] = ..., provider_id: _Optional[str] = ..., model_id: _Optional[str] = ..., latency_ms: _Optional[float] = ...) -> None: ...
