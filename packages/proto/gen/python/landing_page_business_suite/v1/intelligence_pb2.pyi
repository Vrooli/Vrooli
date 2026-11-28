from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AIMessage(_message.Message):
    __slots__ = ("role", "content")
    ROLE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    role: str
    content: str
    def __init__(self, role: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class AIMetadata(_message.Message):
    __slots__ = ("app_bundle_key", "operation")
    APP_BUNDLE_KEY_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    app_bundle_key: str
    operation: str
    def __init__(self, app_bundle_key: _Optional[str] = ..., operation: _Optional[str] = ...) -> None: ...

class ChatRequest(_message.Message):
    __slots__ = ("model", "messages", "max_tokens", "metadata")
    MODEL_FIELD_NUMBER: _ClassVar[int]
    MESSAGES_FIELD_NUMBER: _ClassVar[int]
    MAX_TOKENS_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    model: str
    messages: _containers.RepeatedCompositeFieldContainer[AIMessage]
    max_tokens: int
    metadata: AIMetadata
    def __init__(self, model: _Optional[str] = ..., messages: _Optional[_Iterable[_Union[AIMessage, _Mapping]]] = ..., max_tokens: _Optional[int] = ..., metadata: _Optional[_Union[AIMetadata, _Mapping]] = ...) -> None: ...

class ChatResponse(_message.Message):
    __slots__ = ("id", "model", "content", "prompt_tokens", "completion_tokens", "total_tokens", "credits_charged", "finish_reason")
    ID_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    PROMPT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    COMPLETION_TOKENS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_TOKENS_FIELD_NUMBER: _ClassVar[int]
    CREDITS_CHARGED_FIELD_NUMBER: _ClassVar[int]
    FINISH_REASON_FIELD_NUMBER: _ClassVar[int]
    id: str
    model: str
    content: str
    prompt_tokens: int
    completion_tokens: int
    total_tokens: int
    credits_charged: int
    finish_reason: str
    def __init__(self, id: _Optional[str] = ..., model: _Optional[str] = ..., content: _Optional[str] = ..., prompt_tokens: _Optional[int] = ..., completion_tokens: _Optional[int] = ..., total_tokens: _Optional[int] = ..., credits_charged: _Optional[int] = ..., finish_reason: _Optional[str] = ...) -> None: ...

class ListModelsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListModelsResponse(_message.Message):
    __slots__ = ("models",)
    MODELS_FIELD_NUMBER: _ClassVar[int]
    models: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, models: _Optional[_Iterable[str]] = ...) -> None: ...

class GetUsageRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetUsageResponse(_message.Message):
    __slots__ = ("user_identity", "tier", "billing_period", "reset_date", "ai_credits_used", "ai_credits_limit", "ai_credits_remaining", "display_used", "display_limit", "display_remaining")
    USER_IDENTITY_FIELD_NUMBER: _ClassVar[int]
    TIER_FIELD_NUMBER: _ClassVar[int]
    BILLING_PERIOD_FIELD_NUMBER: _ClassVar[int]
    RESET_DATE_FIELD_NUMBER: _ClassVar[int]
    AI_CREDITS_USED_FIELD_NUMBER: _ClassVar[int]
    AI_CREDITS_LIMIT_FIELD_NUMBER: _ClassVar[int]
    AI_CREDITS_REMAINING_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_USED_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_LIMIT_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_REMAINING_FIELD_NUMBER: _ClassVar[int]
    user_identity: str
    tier: str
    billing_period: str
    reset_date: str
    ai_credits_used: int
    ai_credits_limit: int
    ai_credits_remaining: int
    display_used: float
    display_limit: float
    display_remaining: float
    def __init__(self, user_identity: _Optional[str] = ..., tier: _Optional[str] = ..., billing_period: _Optional[str] = ..., reset_date: _Optional[str] = ..., ai_credits_used: _Optional[int] = ..., ai_credits_limit: _Optional[int] = ..., ai_credits_remaining: _Optional[int] = ..., display_used: _Optional[float] = ..., display_limit: _Optional[float] = ..., display_remaining: _Optional[float] = ...) -> None: ...

class HealthRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class HealthResponse(_message.Message):
    __slots__ = ("status", "provider")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    status: str
    provider: str
    def __init__(self, status: _Optional[str] = ..., provider: _Optional[str] = ...) -> None: ...
