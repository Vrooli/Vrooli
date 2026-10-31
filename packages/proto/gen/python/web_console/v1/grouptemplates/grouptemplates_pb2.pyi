from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TemplateRole(_message.Message):
    __slots__ = ("label", "command", "working_dir", "incoming_prompt", "backend", "target_id", "start_mode")
    LABEL_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    WORKING_DIR_FIELD_NUMBER: _ClassVar[int]
    INCOMING_PROMPT_FIELD_NUMBER: _ClassVar[int]
    BACKEND_FIELD_NUMBER: _ClassVar[int]
    TARGET_ID_FIELD_NUMBER: _ClassVar[int]
    START_MODE_FIELD_NUMBER: _ClassVar[int]
    label: str
    command: str
    working_dir: str
    incoming_prompt: str
    backend: str
    target_id: str
    start_mode: str
    def __init__(self, label: _Optional[str] = ..., command: _Optional[str] = ..., working_dir: _Optional[str] = ..., incoming_prompt: _Optional[str] = ..., backend: _Optional[str] = ..., target_id: _Optional[str] = ..., start_mode: _Optional[str] = ...) -> None: ...

class Template(_message.Message):
    __slots__ = ("id", "name", "color", "roles", "use_count", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    ROLES_FIELD_NUMBER: _ClassVar[int]
    USE_COUNT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    color: str
    roles: _containers.RepeatedCompositeFieldContainer[TemplateRole]
    use_count: int
    created_at: str
    updated_at: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., color: _Optional[str] = ..., roles: _Optional[_Iterable[_Union[TemplateRole, _Mapping]]] = ..., use_count: _Optional[int] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ...) -> None: ...

class ListTemplatesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListTemplatesResponse(_message.Message):
    __slots__ = ("templates",)
    TEMPLATES_FIELD_NUMBER: _ClassVar[int]
    templates: _containers.RepeatedCompositeFieldContainer[Template]
    def __init__(self, templates: _Optional[_Iterable[_Union[Template, _Mapping]]] = ...) -> None: ...

class UpsertTemplateRequest(_message.Message):
    __slots__ = ("id", "name", "color", "roles", "use_count", "has_use_count")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    COLOR_FIELD_NUMBER: _ClassVar[int]
    ROLES_FIELD_NUMBER: _ClassVar[int]
    USE_COUNT_FIELD_NUMBER: _ClassVar[int]
    HAS_USE_COUNT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    color: str
    roles: _containers.RepeatedCompositeFieldContainer[TemplateRole]
    use_count: int
    has_use_count: bool
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., color: _Optional[str] = ..., roles: _Optional[_Iterable[_Union[TemplateRole, _Mapping]]] = ..., use_count: _Optional[int] = ..., has_use_count: _Optional[bool] = ...) -> None: ...

class UpsertTemplateResponse(_message.Message):
    __slots__ = ("template",)
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    template: Template
    def __init__(self, template: _Optional[_Union[Template, _Mapping]] = ...) -> None: ...

class DeleteTemplateRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteTemplateResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
