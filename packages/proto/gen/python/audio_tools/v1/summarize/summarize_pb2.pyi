from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import field_mask_pb2 as _field_mask_pb2
from audio_tools.v1.common import common_pb2 as _common_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SummarizeLevel(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SUMMARIZE_LEVEL_UNSPECIFIED: _ClassVar[SummarizeLevel]
    SUMMARIZE_LEVEL_LIGHT: _ClassVar[SummarizeLevel]
    SUMMARIZE_LEVEL_MODERATE: _ClassVar[SummarizeLevel]
    SUMMARIZE_LEVEL_HEAVY: _ClassVar[SummarizeLevel]
SUMMARIZE_LEVEL_UNSPECIFIED: SummarizeLevel
SUMMARIZE_LEVEL_LIGHT: SummarizeLevel
SUMMARIZE_LEVEL_MODERATE: SummarizeLevel
SUMMARIZE_LEVEL_HEAVY: SummarizeLevel

class SummarizeConfig(_message.Message):
    __slots__ = ("enabled", "char_threshold", "level", "model", "timeout_seconds")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    CHAR_THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SECONDS_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    char_threshold: int
    level: SummarizeLevel
    model: str
    timeout_seconds: int
    def __init__(self, enabled: _Optional[bool] = ..., char_threshold: _Optional[int] = ..., level: _Optional[_Union[SummarizeLevel, str]] = ..., model: _Optional[str] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

class GetSummarizeConfigRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetSummarizeConfigResponse(_message.Message):
    __slots__ = ("config",)
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    config: SummarizeConfig
    def __init__(self, config: _Optional[_Union[SummarizeConfig, _Mapping]] = ...) -> None: ...

class UpdateSummarizeConfigRequest(_message.Message):
    __slots__ = ("update_mask", "config")
    UPDATE_MASK_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    update_mask: _field_mask_pb2.FieldMask
    config: SummarizeConfig
    def __init__(self, update_mask: _Optional[_Union[_field_mask_pb2.FieldMask, _Mapping]] = ..., config: _Optional[_Union[SummarizeConfig, _Mapping]] = ...) -> None: ...

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
    level: SummarizeLevel
    model: str
    timeout_seconds: int
    def __init__(self, text: _Optional[str] = ..., level: _Optional[_Union[SummarizeLevel, str]] = ..., model: _Optional[str] = ..., timeout_seconds: _Optional[int] = ...) -> None: ...

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
    provider_tier: _common_pb2.ProviderTier
    provider_id: str
    model_id: str
    latency_ms: float
    def __init__(self, text: _Optional[str] = ..., prompt_tokens: _Optional[int] = ..., output_tokens: _Optional[int] = ..., provider_tier: _Optional[_Union[_common_pb2.ProviderTier, str]] = ..., provider_id: _Optional[str] = ..., model_id: _Optional[str] = ..., latency_ms: _Optional[float] = ...) -> None: ...
