from portal.v1.shared import common_pb2 as _common_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Chat(_message.Message):
    __slots__ = ("id", "title", "group_id", "sort_order", "model", "web_search_enabled", "mode", "agent_harness", "active_leaf_message_id", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    GROUP_ID_FIELD_NUMBER: _ClassVar[int]
    SORT_ORDER_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    WEB_SEARCH_ENABLED_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    AGENT_HARNESS_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_LEAF_MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    group_id: str
    sort_order: int
    model: str
    web_search_enabled: bool
    mode: _common_pb2.ChatMode
    agent_harness: _common_pb2.AgentHarness
    active_leaf_message_id: str
    created_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., group_id: _Optional[str] = ..., sort_order: _Optional[int] = ..., model: _Optional[str] = ..., web_search_enabled: _Optional[bool] = ..., mode: _Optional[_Union[_common_pb2.ChatMode, str]] = ..., agent_harness: _Optional[_Union[_common_pb2.AgentHarness, str]] = ..., active_leaf_message_id: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class ChatGroup(_message.Message):
    __slots__ = ("id", "name", "color", "collapsed", "sort_order", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    COLLAPSED_FIELD_NUMBER: _ClassVar[int]
    SORT_ORDER_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    color: str
    collapsed: bool
    sort_order: int
    created_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., color: _Optional[str] = ..., collapsed: _Optional[bool] = ..., sort_order: _Optional[int] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class ListChatsRequest(_message.Message):
    __slots__ = ("group_id", "query")
    GROUP_ID_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    group_id: str
    query: str
    def __init__(self, group_id: _Optional[str] = ..., query: _Optional[str] = ...) -> None: ...

class ListChatsResponse(_message.Message):
    __slots__ = ("chats", "groups")
    CHATS_FIELD_NUMBER: _ClassVar[int]
    GROUPS_FIELD_NUMBER: _ClassVar[int]
    chats: _containers.RepeatedCompositeFieldContainer[Chat]
    groups: _containers.RepeatedCompositeFieldContainer[ChatGroup]
    def __init__(self, chats: _Optional[_Iterable[_Union[Chat, _Mapping]]] = ..., groups: _Optional[_Iterable[_Union[ChatGroup, _Mapping]]] = ...) -> None: ...

class CreateChatRequest(_message.Message):
    __slots__ = ("title", "group_id", "model", "web_search_enabled", "mode", "agent_harness")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    GROUP_ID_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    WEB_SEARCH_ENABLED_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    AGENT_HARNESS_FIELD_NUMBER: _ClassVar[int]
    title: str
    group_id: str
    model: str
    web_search_enabled: bool
    mode: _common_pb2.ChatMode
    agent_harness: _common_pb2.AgentHarness
    def __init__(self, title: _Optional[str] = ..., group_id: _Optional[str] = ..., model: _Optional[str] = ..., web_search_enabled: _Optional[bool] = ..., mode: _Optional[_Union[_common_pb2.ChatMode, str]] = ..., agent_harness: _Optional[_Union[_common_pb2.AgentHarness, str]] = ...) -> None: ...

class CreateChatResponse(_message.Message):
    __slots__ = ("chat",)
    CHAT_FIELD_NUMBER: _ClassVar[int]
    chat: Chat
    def __init__(self, chat: _Optional[_Union[Chat, _Mapping]] = ...) -> None: ...

class GetChatRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetChatResponse(_message.Message):
    __slots__ = ("chat",)
    CHAT_FIELD_NUMBER: _ClassVar[int]
    chat: Chat
    def __init__(self, chat: _Optional[_Union[Chat, _Mapping]] = ...) -> None: ...

class UpdateChatRequest(_message.Message):
    __slots__ = ("id", "title", "has_title", "group_id", "has_group_id", "model", "has_model", "web_search_enabled", "has_web_search_enabled", "active_leaf_message_id", "has_active_leaf_message_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    HAS_TITLE_FIELD_NUMBER: _ClassVar[int]
    GROUP_ID_FIELD_NUMBER: _ClassVar[int]
    HAS_GROUP_ID_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    HAS_MODEL_FIELD_NUMBER: _ClassVar[int]
    WEB_SEARCH_ENABLED_FIELD_NUMBER: _ClassVar[int]
    HAS_WEB_SEARCH_ENABLED_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_LEAF_MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    HAS_ACTIVE_LEAF_MESSAGE_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    has_title: bool
    group_id: str
    has_group_id: bool
    model: str
    has_model: bool
    web_search_enabled: bool
    has_web_search_enabled: bool
    active_leaf_message_id: str
    has_active_leaf_message_id: bool
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., has_title: _Optional[bool] = ..., group_id: _Optional[str] = ..., has_group_id: _Optional[bool] = ..., model: _Optional[str] = ..., has_model: _Optional[bool] = ..., web_search_enabled: _Optional[bool] = ..., has_web_search_enabled: _Optional[bool] = ..., active_leaf_message_id: _Optional[str] = ..., has_active_leaf_message_id: _Optional[bool] = ...) -> None: ...

class UpdateChatResponse(_message.Message):
    __slots__ = ("chat",)
    CHAT_FIELD_NUMBER: _ClassVar[int]
    chat: Chat
    def __init__(self, chat: _Optional[_Union[Chat, _Mapping]] = ...) -> None: ...

class DeleteChatRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteChatResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListGroupsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListGroupsResponse(_message.Message):
    __slots__ = ("groups",)
    GROUPS_FIELD_NUMBER: _ClassVar[int]
    groups: _containers.RepeatedCompositeFieldContainer[ChatGroup]
    def __init__(self, groups: _Optional[_Iterable[_Union[ChatGroup, _Mapping]]] = ...) -> None: ...

class CreateGroupRequest(_message.Message):
    __slots__ = ("name", "color")
    NAME_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    name: str
    color: str
    def __init__(self, name: _Optional[str] = ..., color: _Optional[str] = ...) -> None: ...

class CreateGroupResponse(_message.Message):
    __slots__ = ("group",)
    GROUP_FIELD_NUMBER: _ClassVar[int]
    group: ChatGroup
    def __init__(self, group: _Optional[_Union[ChatGroup, _Mapping]] = ...) -> None: ...

class UpdateGroupRequest(_message.Message):
    __slots__ = ("id", "name", "has_name", "color", "has_color", "collapsed", "has_collapsed", "sort_order", "has_sort_order")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    HAS_NAME_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    HAS_COLOR_FIELD_NUMBER: _ClassVar[int]
    COLLAPSED_FIELD_NUMBER: _ClassVar[int]
    HAS_COLLAPSED_FIELD_NUMBER: _ClassVar[int]
    SORT_ORDER_FIELD_NUMBER: _ClassVar[int]
    HAS_SORT_ORDER_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    has_name: bool
    color: str
    has_color: bool
    collapsed: bool
    has_collapsed: bool
    sort_order: int
    has_sort_order: bool
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., has_name: _Optional[bool] = ..., color: _Optional[str] = ..., has_color: _Optional[bool] = ..., collapsed: _Optional[bool] = ..., has_collapsed: _Optional[bool] = ..., sort_order: _Optional[int] = ..., has_sort_order: _Optional[bool] = ...) -> None: ...

class UpdateGroupResponse(_message.Message):
    __slots__ = ("group",)
    GROUP_FIELD_NUMBER: _ClassVar[int]
    group: ChatGroup
    def __init__(self, group: _Optional[_Union[ChatGroup, _Mapping]] = ...) -> None: ...

class DeleteGroupRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteGroupResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
