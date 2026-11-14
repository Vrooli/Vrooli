from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Pane(_message.Message):
    __slots__ = ("session_id", "name", "header_color", "theme_id", "font_size", "sort_order", "group_id", "supports_messages_view", "manually_unread", "created_at", "updated_at")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    HEADER_COLOR_FIELD_NUMBER: _ClassVar[int]
    THEME_ID_FIELD_NUMBER: _ClassVar[int]
    FONT_SIZE_FIELD_NUMBER: _ClassVar[int]
    SORT_ORDER_FIELD_NUMBER: _ClassVar[int]
    GROUP_ID_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_MESSAGES_VIEW_FIELD_NUMBER: _ClassVar[int]
    MANUALLY_UNREAD_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    name: str
    header_color: str
    theme_id: str
    font_size: int
    sort_order: int
    group_id: str
    supports_messages_view: bool
    manually_unread: bool
    created_at: str
    updated_at: str
    def __init__(self, session_id: _Optional[str] = ..., name: _Optional[str] = ..., header_color: _Optional[str] = ..., theme_id: _Optional[str] = ..., font_size: _Optional[int] = ..., sort_order: _Optional[int] = ..., group_id: _Optional[str] = ..., supports_messages_view: _Optional[bool] = ..., manually_unread: _Optional[bool] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class Group(_message.Message):
    __slots__ = ("id", "name", "color", "sort_order", "is_collapsed", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    SORT_ORDER_FIELD_NUMBER: _ClassVar[int]
    IS_COLLAPSED_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    color: str
    sort_order: int
    is_collapsed: bool
    created_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., color: _Optional[str] = ..., sort_order: _Optional[int] = ..., is_collapsed: _Optional[bool] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class GetLayoutRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class Role(_message.Message):
    __slots__ = ("id", "group_id", "label", "command", "working_dir", "incoming_prompt", "session_id", "sort_order", "backend", "target_id", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    GROUP_ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    WORKING_DIR_FIELD_NUMBER: _ClassVar[int]
    INCOMING_PROMPT_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SORT_ORDER_FIELD_NUMBER: _ClassVar[int]
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    group_id: str
    label: str
    command: str
    working_dir: str
    incoming_prompt: str
    session_id: str
    sort_order: int
    backend: str
    target_id: str
    created_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., group_id: _Optional[str] = ..., label: _Optional[str] = ..., command: _Optional[str] = ..., working_dir: _Optional[str] = ..., incoming_prompt: _Optional[str] = ..., session_id: _Optional[str] = ..., sort_order: _Optional[int] = ..., backend: _Optional[str] = ..., target_id: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class GetLayoutResponse(_message.Message):
    __slots__ = ("active_pane", "panes", "groups", "roles")
    ACTIVE_PANE_FIELD_NUMBER: _ClassVar[int]
    PANES_FIELD_NUMBER: _ClassVar[int]
    GROUPS_FIELD_NUMBER: _ClassVar[int]
    ROLES_FIELD_NUMBER: _ClassVar[int]
    active_pane: str
    panes: _containers.RepeatedCompositeFieldContainer[Pane]
    groups: _containers.RepeatedCompositeFieldContainer[Group]
    roles: _containers.RepeatedCompositeFieldContainer[Role]
    def __init__(self, active_pane: _Optional[str] = ..., panes: _Optional[_Iterable[_Union[Pane, _Mapping]]] = ..., groups: _Optional[_Iterable[_Union[Group, _Mapping]]] = ..., roles: _Optional[_Iterable[_Union[Role, _Mapping]]] = ...) -> None: ...

class SaveLayoutRequest(_message.Message):
    __slots__ = ("active_pane", "pane_order")
    ACTIVE_PANE_FIELD_NUMBER: _ClassVar[int]
    PANE_ORDER_FIELD_NUMBER: _ClassVar[int]
    active_pane: str
    pane_order: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, active_pane: _Optional[str] = ..., pane_order: _Optional[_Iterable[str]] = ...) -> None: ...

class SaveLayoutResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class UpdatePaneRequest(_message.Message):
    __slots__ = ("session_id", "name", "has_name", "header_color", "has_header_color", "theme_id", "has_theme_id", "font_size", "has_font_size", "sort_order", "has_sort_order", "group_id", "has_group_id", "supports_messages_view", "has_supports_messages_view", "manually_unread", "has_manually_unread")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    HAS_NAME_FIELD_NUMBER: _ClassVar[int]
    HEADER_COLOR_FIELD_NUMBER: _ClassVar[int]
    HAS_HEADER_COLOR_FIELD_NUMBER: _ClassVar[int]
    THEME_ID_FIELD_NUMBER: _ClassVar[int]
    HAS_THEME_ID_FIELD_NUMBER: _ClassVar[int]
    FONT_SIZE_FIELD_NUMBER: _ClassVar[int]
    HAS_FONT_SIZE_FIELD_NUMBER: _ClassVar[int]
    SORT_ORDER_FIELD_NUMBER: _ClassVar[int]
    HAS_SORT_ORDER_FIELD_NUMBER: _ClassVar[int]
    GROUP_ID_FIELD_NUMBER: _ClassVar[int]
    HAS_GROUP_ID_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_MESSAGES_VIEW_FIELD_NUMBER: _ClassVar[int]
    HAS_SUPPORTS_MESSAGES_VIEW_FIELD_NUMBER: _ClassVar[int]
    MANUALLY_UNREAD_FIELD_NUMBER: _ClassVar[int]
    HAS_MANUALLY_UNREAD_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    name: str
    has_name: bool
    header_color: str
    has_header_color: bool
    theme_id: str
    has_theme_id: bool
    font_size: int
    has_font_size: bool
    sort_order: int
    has_sort_order: bool
    group_id: str
    has_group_id: bool
    supports_messages_view: bool
    has_supports_messages_view: bool
    manually_unread: bool
    has_manually_unread: bool
    def __init__(self, session_id: _Optional[str] = ..., name: _Optional[str] = ..., has_name: _Optional[bool] = ..., header_color: _Optional[str] = ..., has_header_color: _Optional[bool] = ..., theme_id: _Optional[str] = ..., has_theme_id: _Optional[bool] = ..., font_size: _Optional[int] = ..., has_font_size: _Optional[bool] = ..., sort_order: _Optional[int] = ..., has_sort_order: _Optional[bool] = ..., group_id: _Optional[str] = ..., has_group_id: _Optional[bool] = ..., supports_messages_view: _Optional[bool] = ..., has_supports_messages_view: _Optional[bool] = ..., manually_unread: _Optional[bool] = ..., has_manually_unread: _Optional[bool] = ...) -> None: ...

class UpdatePaneResponse(_message.Message):
    __slots__ = ("pane",)
    PANE_FIELD_NUMBER: _ClassVar[int]
    pane: Pane
    def __init__(self, pane: _Optional[_Union[Pane, _Mapping]] = ...) -> None: ...

class DeletePaneRequest(_message.Message):
    __slots__ = ("session_id",)
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    def __init__(self, session_id: _Optional[str] = ...) -> None: ...

class DeletePaneResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

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
    group: Group
    def __init__(self, group: _Optional[_Union[Group, _Mapping]] = ...) -> None: ...

class UpdateGroupRequest(_message.Message):
    __slots__ = ("id", "name", "has_name", "color", "has_color", "is_collapsed", "has_is_collapsed")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    HAS_NAME_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    HAS_COLOR_FIELD_NUMBER: _ClassVar[int]
    IS_COLLAPSED_FIELD_NUMBER: _ClassVar[int]
    HAS_IS_COLLAPSED_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    has_name: bool
    color: str
    has_color: bool
    is_collapsed: bool
    has_is_collapsed: bool
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., has_name: _Optional[bool] = ..., color: _Optional[str] = ..., has_color: _Optional[bool] = ..., is_collapsed: _Optional[bool] = ..., has_is_collapsed: _Optional[bool] = ...) -> None: ...

class UpdateGroupResponse(_message.Message):
    __slots__ = ("group",)
    GROUP_FIELD_NUMBER: _ClassVar[int]
    group: Group
    def __init__(self, group: _Optional[_Union[Group, _Mapping]] = ...) -> None: ...

class DeleteGroupRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteGroupResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListRolesRequest(_message.Message):
    __slots__ = ("group_id",)
    GROUP_ID_FIELD_NUMBER: _ClassVar[int]
    group_id: str
    def __init__(self, group_id: _Optional[str] = ...) -> None: ...

class ListRolesResponse(_message.Message):
    __slots__ = ("roles",)
    ROLES_FIELD_NUMBER: _ClassVar[int]
    roles: _containers.RepeatedCompositeFieldContainer[Role]
    def __init__(self, roles: _Optional[_Iterable[_Union[Role, _Mapping]]] = ...) -> None: ...

class CreateRoleRequest(_message.Message):
    __slots__ = ("group_id", "label", "command", "working_dir", "incoming_prompt", "session_id", "sort_order", "backend", "target_id")
    GROUP_ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    WORKING_DIR_FIELD_NUMBER: _ClassVar[int]
    INCOMING_PROMPT_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SORT_ORDER_FIELD_NUMBER: _ClassVar[int]
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    group_id: str
    label: str
    command: str
    working_dir: str
    incoming_prompt: str
    session_id: str
    sort_order: int
    backend: str
    target_id: str
    def __init__(self, group_id: _Optional[str] = ..., label: _Optional[str] = ..., command: _Optional[str] = ..., working_dir: _Optional[str] = ..., incoming_prompt: _Optional[str] = ..., session_id: _Optional[str] = ..., sort_order: _Optional[int] = ..., backend: _Optional[str] = ..., target_id: _Optional[str] = ...) -> None: ...

class CreateRoleResponse(_message.Message):
    __slots__ = ("role",)
    ROLE_FIELD_NUMBER: _ClassVar[int]
    role: Role
    def __init__(self, role: _Optional[_Union[Role, _Mapping]] = ...) -> None: ...

class UpdateRoleRequest(_message.Message):
    __slots__ = ("id", "label", "has_label", "command", "has_command", "working_dir", "has_working_dir", "incoming_prompt", "has_incoming_prompt", "session_id", "has_session_id", "sort_order", "has_sort_order", "backend", "has_backend", "target_id", "has_target_id", "group_id", "has_group_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    HAS_LABEL_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    HAS_COMMAND_FIELD_NUMBER: _ClassVar[int]
    WORKING_DIR_FIELD_NUMBER: _ClassVar[int]
    HAS_WORKING_DIR_FIELD_NUMBER: _ClassVar[int]
    INCOMING_PROMPT_FIELD_NUMBER: _ClassVar[int]
    HAS_INCOMING_PROMPT_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    HAS_SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SORT_ORDER_FIELD_NUMBER: _ClassVar[int]
    HAS_SORT_ORDER_FIELD_NUMBER: _ClassVar[int]
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    HAS_BACKEND_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    HAS_TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    GROUP_ID_FIELD_NUMBER: _ClassVar[int]
    HAS_GROUP_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    has_label: bool
    command: str
    has_command: bool
    working_dir: str
    has_working_dir: bool
    incoming_prompt: str
    has_incoming_prompt: bool
    session_id: str
    has_session_id: bool
    sort_order: int
    has_sort_order: bool
    backend: str
    has_backend: bool
    target_id: str
    has_target_id: bool
    group_id: str
    has_group_id: bool
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., has_label: _Optional[bool] = ..., command: _Optional[str] = ..., has_command: _Optional[bool] = ..., working_dir: _Optional[str] = ..., has_working_dir: _Optional[bool] = ..., incoming_prompt: _Optional[str] = ..., has_incoming_prompt: _Optional[bool] = ..., session_id: _Optional[str] = ..., has_session_id: _Optional[bool] = ..., sort_order: _Optional[int] = ..., has_sort_order: _Optional[bool] = ..., backend: _Optional[str] = ..., has_backend: _Optional[bool] = ..., target_id: _Optional[str] = ..., has_target_id: _Optional[bool] = ..., group_id: _Optional[str] = ..., has_group_id: _Optional[bool] = ...) -> None: ...

class UpdateRoleResponse(_message.Message):
    __slots__ = ("role",)
    ROLE_FIELD_NUMBER: _ClassVar[int]
    role: Role
    def __init__(self, role: _Optional[_Union[Role, _Mapping]] = ...) -> None: ...

class DeleteRoleRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteRoleResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
