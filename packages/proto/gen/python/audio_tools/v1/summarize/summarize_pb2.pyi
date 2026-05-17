from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SummarizeConfig(_message.Message):
    __slots__ = ("enabled", "char_threshold", "level", "model", "timeout_seconds")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    CHAR_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    char_threshold: int
    level: str
    model: str
    timeout_seconds: int
    def __init__(self, enabled: _Optional[bool] = ..., char_threshold: _Optional[int] = ..., level: _Optional[str] = ..., model: _Optional[str] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class GetSummarizeConfigRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetSummarizeConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: SummarizeConfig
    def __init__(self, config: _Optional[_Union[SummarizeConfig, _Mapping]] = ...) -> None: ...

class UpdateSummarizeConfigRequest(_message.Message):
    __slots__ = ("enabled", "has_enabled", "char_threshold", "has_char_threshold", "level", "has_level", "model", "has_model", "timeout_seconds", "has_timeout_seconds")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    HAS_ENABLED_FIELD_NUMBER: _ClassVar[int]
    CHAR_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    HAS_CHAR_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    HAS_LEVEL_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    HAS_MODEL_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    HAS_TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    has_enabled: bool
    char_threshold: int
    has_char_threshold: bool
    level: str
    has_level: bool
    model: str
    has_model: bool
    timeout_seconds: int
    has_timeout_seconds: bool
    def __init__(self, enabled: _Optional[bool] = ..., has_enabled: _Optional[bool] = ..., char_threshold: _Optional[int] = ..., has_char_threshold: _Optional[bool] = ..., level: _Optional[str] = ..., has_level: _Optional[bool] = ..., model: _Optional[str] = ..., has_model: _Optional[bool] = ..., timeout_seconds: _Optional[int] = ..., has_timeout_seconds: _Optional[bool] = ...) -> None: ...

class UpdateSummarizeConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: SummarizeConfig
    def __init__(self, config: _Optional[_Union[SummarizeConfig, _Mapping]] = ...) -> None: ...

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
