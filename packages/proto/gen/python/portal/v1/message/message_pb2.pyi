from portal.v1.shared import common_pb2 as _common_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class MessageRole(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    MESSAGE_ROLE_UNSPECIFIED: _ClassVar[MessageRole]
    MESSAGE_ROLE_SYSTEM: _ClassVar[MessageRole]
    MESSAGE_ROLE_USER: _ClassVar[MessageRole]
    MESSAGE_ROLE_ASSISTANT: _ClassVar[MessageRole]
    MESSAGE_ROLE_AGENT: _ClassVar[MessageRole]

class CompletionEventKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COMPLETION_EVENT_KIND_UNSPECIFIED: _ClassVar[CompletionEventKind]
    COMPLETION_EVENT_KIND_STATUS: _ClassVar[CompletionEventKind]
    COMPLETION_EVENT_KIND_TOKEN: _ClassVar[CompletionEventKind]
    COMPLETION_EVENT_KIND_SEARCH_ATTACHMENT: _ClassVar[CompletionEventKind]
    COMPLETION_EVENT_KIND_AGENT_ACTIVITY: _ClassVar[CompletionEventKind]
    COMPLETION_EVENT_KIND_DONE: _ClassVar[CompletionEventKind]
    COMPLETION_EVENT_KIND_ERROR: _ClassVar[CompletionEventKind]
MESSAGE_ROLE_UNSPECIFIED: MessageRole
MESSAGE_ROLE_SYSTEM: MessageRole
MESSAGE_ROLE_USER: MessageRole
MESSAGE_ROLE_ASSISTANT: MessageRole
MESSAGE_ROLE_AGENT: MessageRole
COMPLETION_EVENT_KIND_UNSPECIFIED: CompletionEventKind
COMPLETION_EVENT_KIND_STATUS: CompletionEventKind
COMPLETION_EVENT_KIND_TOKEN: CompletionEventKind
COMPLETION_EVENT_KIND_SEARCH_ATTACHMENT: CompletionEventKind
COMPLETION_EVENT_KIND_AGENT_ACTIVITY: CompletionEventKind
COMPLETION_EVENT_KIND_DONE: CompletionEventKind
COMPLETION_EVENT_KIND_ERROR: CompletionEventKind

class Message(_message.Message):
    __slots__ = ("id", "chat_id", "parent_message_id", "sibling_index", "role", "content", "model", "created_at", "updated_at", "search_attachments")
    ID_FIELD_NUMBER: _ClassVar[int]
    CHAT_ID_FIELD_NUMBER: _ClassVar[int]
    PARENT_MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    SIBLING_INDEX_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    SEARCH_ATTACHMENTS_FIELD_NUMBER: _ClassVar[int]
    id: str
    chat_id: str
    parent_message_id: str
    sibling_index: int
    role: MessageRole
    content: str
    model: str
    created_at: str
    updated_at: str
    search_attachments: _containers.RepeatedCompositeFieldContainer[SearchAttachment]
    def __init__(self, id: _Optional[str] = ..., chat_id: _Optional[str] = ..., parent_message_id: _Optional[str] = ..., sibling_index: _Optional[int] = ..., role: _Optional[_Union[MessageRole, str]] = ..., content: _Optional[str] = ..., model: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., search_attachments: _Optional[_Iterable[_Union[SearchAttachment, _Mapping]]] = ...) -> None: ...

class SearchAttachment(_message.Message):
    __slots__ = ("id", "query", "hits", "degraded", "reason", "latency_ms", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    HITS_FIELD_NUMBER: _ClassVar[int]
    DEGRADED_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    query: str
    hits: _containers.RepeatedCompositeFieldContainer[_common_pb2.SearchHit]
    degraded: bool
    reason: str
    latency_ms: int
    created_at: str
    def __init__(self, id: _Optional[str] = ..., query: _Optional[str] = ..., hits: _Optional[_Iterable[_Union[_common_pb2.SearchHit, _Mapping]]] = ..., degraded: _Optional[bool] = ..., reason: _Optional[str] = ..., latency_ms: _Optional[int] = ..., created_at: _Optional[str] = ...) -> None: ...

class UsageRecord(_message.Message):
    __slots__ = ("message_id", "provider", "model", "prompt_tokens", "completion_tokens", "cost_usd")
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    PROMPT_TOKENS_FIELD_NUMBER: _ClassVar[int]
    COMPLETION_TOKENS_FIELD_NUMBER: _ClassVar[int]
    COST_USD_FIELD_NUMBER: _ClassVar[int]
    message_id: str
    provider: str
    model: str
    prompt_tokens: int
    completion_tokens: int
    cost_usd: float
    def __init__(self, message_id: _Optional[str] = ..., provider: _Optional[str] = ..., model: _Optional[str] = ..., prompt_tokens: _Optional[int] = ..., completion_tokens: _Optional[int] = ..., cost_usd: _Optional[float] = ...) -> None: ...

class GetTreeRequest(_message.Message):
    __slots__ = ("chat_id",)
    CHAT_ID_FIELD_NUMBER: _ClassVar[int]
    chat_id: str
    def __init__(self, chat_id: _Optional[str] = ...) -> None: ...

class GetTreeResponse(_message.Message):
    __slots__ = ("messages", "active_leaf_message_id")
    MESSAGES_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_LEAF_MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    messages: _containers.RepeatedCompositeFieldContainer[Message]
    active_leaf_message_id: str
    def __init__(self, messages: _Optional[_Iterable[_Union[Message, _Mapping]]] = ..., active_leaf_message_id: _Optional[str] = ...) -> None: ...

class SendMessageRequest(_message.Message):
    __slots__ = ("chat_id", "parent_message_id", "content", "model", "web_search_enabled", "selected_skill_ids")
    CHAT_ID_FIELD_NUMBER: _ClassVar[int]
    PARENT_MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    WEB_SEARCH_ENABLED_FIELD_NUMBER: _ClassVar[int]
    SELECTED_SKILL_IDS_FIELD_NUMBER: _ClassVar[int]
    chat_id: str
    parent_message_id: str
    content: str
    model: str
    web_search_enabled: bool
    selected_skill_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, chat_id: _Optional[str] = ..., parent_message_id: _Optional[str] = ..., content: _Optional[str] = ..., model: _Optional[str] = ..., web_search_enabled: _Optional[bool] = ..., selected_skill_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class SendMessageResponse(_message.Message):
    __slots__ = ("user_message",)
    USER_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    user_message: Message
    def __init__(self, user_message: _Optional[_Union[Message, _Mapping]] = ...) -> None: ...

class EditMessageRequest(_message.Message):
    __slots__ = ("message_id", "content")
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    message_id: str
    content: str
    def __init__(self, message_id: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class EditMessageResponse(_message.Message):
    __slots__ = ("message",)
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    message: Message
    def __init__(self, message: _Optional[_Union[Message, _Mapping]] = ...) -> None: ...

class RegenerateRequest(_message.Message):
    __slots__ = ("message_id", "model")
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    message_id: str
    model: str
    def __init__(self, message_id: _Optional[str] = ..., model: _Optional[str] = ...) -> None: ...

class RegenerateResponse(_message.Message):
    __slots__ = ("assistant_message",)
    ASSISTANT_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    assistant_message: Message
    def __init__(self, assistant_message: _Optional[_Union[Message, _Mapping]] = ...) -> None: ...

class StreamCompletionRequest(_message.Message):
    __slots__ = ("chat_id", "from_message_id", "model", "web_search_enabled", "selected_skill_ids", "mode")
    CHAT_ID_FIELD_NUMBER: _ClassVar[int]
    FROM_MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    WEB_SEARCH_ENABLED_FIELD_NUMBER: _ClassVar[int]
    SELECTED_SKILL_IDS_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    chat_id: str
    from_message_id: str
    model: str
    web_search_enabled: bool
    selected_skill_ids: _containers.RepeatedScalarFieldContainer[str]
    mode: _common_pb2.ChatMode
    def __init__(self, chat_id: _Optional[str] = ..., from_message_id: _Optional[str] = ..., model: _Optional[str] = ..., web_search_enabled: _Optional[bool] = ..., selected_skill_ids: _Optional[_Iterable[str]] = ..., mode: _Optional[_Union[_common_pb2.ChatMode, str]] = ...) -> None: ...

class CompletionEvent(_message.Message):
    __slots__ = ("kind", "message_id", "text", "search_attachment", "usage", "error_code", "error_message")
    KIND_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    SEARCH_ATTACHMENT_FIELD_NUMBER: _ClassVar[int]
    USAGE_FIELD_NUMBER: _ClassVar[int]
    ERROR_CODE_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    kind: CompletionEventKind
    message_id: str
    text: str
    search_attachment: SearchAttachment
    usage: UsageRecord
    error_code: str
    error_message: str
    def __init__(self, kind: _Optional[_Union[CompletionEventKind, str]] = ..., message_id: _Optional[str] = ..., text: _Optional[str] = ..., search_attachment: _Optional[_Union[SearchAttachment, _Mapping]] = ..., usage: _Optional[_Union[UsageRecord, _Mapping]] = ..., error_code: _Optional[str] = ..., error_message: _Optional[str] = ...) -> None: ...
