from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import field_mask_pb2 as _field_mask_pb2
from audio_tools.v1.common import common_pb2 as _common_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
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

class ListSummarizeModelsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class SummarizeModel(_message.Message):
    __slots__ = ("id", "display_name", "installed", "recommended", "default_eligible", "reasoning", "status_label", "pull_command", "size_bytes", "parameter_size", "source_url", "notes")
    ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    INSTALLED_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDED_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_ELIGIBLE_FIELD_NUMBER: _ClassVar[int]
    REASONING_FIELD_NUMBER: _ClassVar[int]
    STATUS_LABEL_FIELD_NUMBER: _ClassVar[int]
    PULL_COMMAND_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    PARAMETER_SIZE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_URL_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    id: str
    display_name: str
    installed: bool
    recommended: bool
    default_eligible: bool
    reasoning: bool
    status_label: str
    pull_command: str
    size_bytes: int
    parameter_size: str
    source_url: str
    notes: str
    def __init__(self, id: _Optional[str] = ..., display_name: _Optional[str] = ..., installed: _Optional[bool] = ..., recommended: _Optional[bool] = ..., default_eligible: _Optional[bool] = ..., reasoning: _Optional[bool] = ..., status_label: _Optional[str] = ..., pull_command: _Optional[str] = ..., size_bytes: _Optional[int] = ..., parameter_size: _Optional[str] = ..., source_url: _Optional[str] = ..., notes: _Optional[str] = ...) -> None: ...

class ListSummarizeModelsResponse(_message.Message):
    __slots__ = ("models",)
    MODELS_FIELD_NUMBER: _ClassVar[int]
    models: _containers.RepeatedCompositeFieldContainer[SummarizeModel]
    def __init__(self, models: _Optional[_Iterable[_Union[SummarizeModel, _Mapping]]] = ...) -> None: ...

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
