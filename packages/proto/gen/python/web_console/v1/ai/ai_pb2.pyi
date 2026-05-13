from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GenerateRequest(_message.Message):
    __slots__ = ("prompt", "context")
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    prompt: str
    context: str
    def __init__(self, prompt: _Optional[str] = ..., context: _Optional[str] = ...) -> None: ...

class GenerateResponse(_message.Message):
    __slots__ = ("command", "provider")
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    command: str
    provider: str
    def __init__(self, command: _Optional[str] = ..., provider: _Optional[str] = ...) -> None: ...

class SuggestRequest(_message.Message):
    __slots__ = ("prompt", "context")
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    prompt: str
    context: str
    def __init__(self, prompt: _Optional[str] = ..., context: _Optional[str] = ...) -> None: ...

class SuggestResponse(_message.Message):
    __slots__ = ("commands", "provider")
    COMMANDS_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    commands: _containers.RepeatedScalarFieldContainer[str]
    provider: str
    def __init__(self, commands: _Optional[_Iterable[str]] = ..., provider: _Optional[str] = ...) -> None: ...

class ProviderConfig(_message.Message):
    __slots__ = ("name", "enabled", "priority", "timeout_sec", "max_retries")
    NAME_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SEC_FIELD_NUMBER: _ClassVar[int]
    MAX_RETRIES_FIELD_NUMBER: _ClassVar[int]
    name: str
    enabled: bool
    priority: int
    timeout_sec: int
    max_retries: int
    def __init__(self, name: _Optional[str] = ..., enabled: _Optional[bool] = ..., priority: _Optional[int] = ..., timeout_sec: _Optional[int] = ..., max_retries: _Optional[int] = ...) -> None: ...

class ProviderHealth(_message.Message):
    __slots__ = ("name", "available", "last_check", "last_latency", "error_count", "success_count", "error_rate")
    NAME_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    LAST_CHECK_FIELD_NUMBER: _ClassVar[int]
    LAST_LATENCY_FIELD_NUMBER: _ClassVar[int]
    ERROR_COUNT_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_COUNT_FIELD_NUMBER: _ClassVar[int]
    ERROR_RATE_FIELD_NUMBER: _ClassVar[int]
    name: str
    available: bool
    last_check: str
    last_latency: str
    error_count: int
    success_count: int
    error_rate: float
    def __init__(self, name: _Optional[str] = ..., available: _Optional[bool] = ..., last_check: _Optional[str] = ..., last_latency: _Optional[str] = ..., error_count: _Optional[int] = ..., success_count: _Optional[int] = ..., error_rate: _Optional[float] = ...) -> None: ...

class GetConfigRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetConfigResponse(_message.Message):
    __slots__ = ("providers", "health")
    PROVIDERS_FIELD_NUMBER: _ClassVar[int]
    HEALTH_FIELD_NUMBER: _ClassVar[int]
    providers: _containers.RepeatedCompositeFieldContainer[ProviderConfig]
    health: _containers.RepeatedCompositeFieldContainer[ProviderHealth]
    def __init__(self, providers: _Optional[_Iterable[_Union[ProviderConfig, _Mapping]]] = ..., health: _Optional[_Iterable[_Union[ProviderHealth, _Mapping]]] = ...) -> None: ...

class UpdateConfigRequest(_message.Message):
    __slots__ = ("name", "enabled", "has_enabled", "priority", "has_priority", "timeout_sec", "has_timeout_sec", "max_retries", "has_max_retries")
    NAME_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    HAS_ENABLED_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    HAS_PRIORITY_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_SEC_FIELD_NUMBER: _ClassVar[int]
    HAS_TIMEOUT_SEC_FIELD_NUMBER: _ClassVar[int]
    MAX_RETRIES_FIELD_NUMBER: _ClassVar[int]
    HAS_MAX_RETRIES_FIELD_NUMBER: _ClassVar[int]
    name: str
    enabled: bool
    has_enabled: bool
    priority: int
    has_priority: bool
    timeout_sec: int
    has_timeout_sec: bool
    max_retries: int
    has_max_retries: bool
    def __init__(self, name: _Optional[str] = ..., enabled: _Optional[bool] = ..., has_enabled: _Optional[bool] = ..., priority: _Optional[int] = ..., has_priority: _Optional[bool] = ..., timeout_sec: _Optional[int] = ..., has_timeout_sec: _Optional[bool] = ..., max_retries: _Optional[int] = ..., has_max_retries: _Optional[bool] = ...) -> None: ...

class UpdateConfigResponse(_message.Message):
    __slots__ = ("providers", "health")
    PROVIDERS_FIELD_NUMBER: _ClassVar[int]
    HEALTH_FIELD_NUMBER: _ClassVar[int]
    providers: _containers.RepeatedCompositeFieldContainer[ProviderConfig]
    health: _containers.RepeatedCompositeFieldContainer[ProviderHealth]
    def __init__(self, providers: _Optional[_Iterable[_Union[ProviderConfig, _Mapping]]] = ..., health: _Optional[_Iterable[_Union[ProviderHealth, _Mapping]]] = ...) -> None: ...

class GetHealthRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetHealthResponse(_message.Message):
    __slots__ = ("health",)
    HEALTH_FIELD_NUMBER: _ClassVar[int]
    health: _containers.RepeatedCompositeFieldContainer[ProviderHealth]
    def __init__(self, health: _Optional[_Iterable[_Union[ProviderHealth, _Mapping]]] = ...) -> None: ...
