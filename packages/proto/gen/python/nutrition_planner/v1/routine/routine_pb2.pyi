from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Template(_message.Message):
    __slots__ = ("id", "revision", "slot_name", "recipe_id", "quantity", "weekdays", "start_date", "end_date", "mode", "active")
    ID_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    SLOT_NAME_FIELD_NUMBER: _ClassVar[int]
    RECIPE_ID_FIELD_NUMBER: _ClassVar[int]
    QUANTITY_FIELD_NUMBER: _ClassVar[int]
    WEEKDAYS_FIELD_NUMBER: _ClassVar[int]
    START_DATE_FIELD_NUMBER: _ClassVar[int]
    END_DATE_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    id: str
    revision: int
    slot_name: str
    recipe_id: str
    quantity: str
    weekdays: _containers.RepeatedScalarFieldContainer[int]
    start_date: str
    end_date: str
    mode: str
    active: bool
    def __init__(self, id: _Optional[str] = ..., revision: _Optional[int] = ..., slot_name: _Optional[str] = ..., recipe_id: _Optional[str] = ..., quantity: _Optional[str] = ..., weekdays: _Optional[_Iterable[int]] = ..., start_date: _Optional[str] = ..., end_date: _Optional[str] = ..., mode: _Optional[str] = ..., active: _Optional[bool] = ...) -> None: ...

class Occurrence(_message.Message):
    __slots__ = ("date", "slot_name", "recipe_id", "quantity", "template_id", "template_revision", "mode")
    DATE_FIELD_NUMBER: _ClassVar[int]
    SLOT_NAME_FIELD_NUMBER: _ClassVar[int]
    RECIPE_ID_FIELD_NUMBER: _ClassVar[int]
    QUANTITY_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_ID_FIELD_NUMBER: _ClassVar[int]
    TEMPLATE_REVISION_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    date: str
    slot_name: str
    recipe_id: str
    quantity: str
    template_id: str
    template_revision: int
    mode: str
    def __init__(self, date: _Optional[str] = ..., slot_name: _Optional[str] = ..., recipe_id: _Optional[str] = ..., quantity: _Optional[str] = ..., template_id: _Optional[str] = ..., template_revision: _Optional[int] = ..., mode: _Optional[str] = ...) -> None: ...

class ListTemplatesRequest(_message.Message):
    __slots__ = ("workspace_id",)
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    def __init__(self, workspace_id: _Optional[str] = ...) -> None: ...

class ListTemplatesResponse(_message.Message):
    __slots__ = ("templates",)
    TEMPLATES_FIELD_NUMBER: _ClassVar[int]
    templates: _containers.RepeatedCompositeFieldContainer[Template]
    def __init__(self, templates: _Optional[_Iterable[_Union[Template, _Mapping]]] = ...) -> None: ...

class CreateTemplateRequest(_message.Message):
    __slots__ = ("workspace_id", "slot_name", "recipe_id", "quantity", "weekdays", "start_date", "end_date", "mode", "active")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    SLOT_NAME_FIELD_NUMBER: _ClassVar[int]
    RECIPE_ID_FIELD_NUMBER: _ClassVar[int]
    QUANTITY_FIELD_NUMBER: _ClassVar[int]
    WEEKDAYS_FIELD_NUMBER: _ClassVar[int]
    START_DATE_FIELD_NUMBER: _ClassVar[int]
    END_DATE_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    slot_name: str
    recipe_id: str
    quantity: str
    weekdays: _containers.RepeatedScalarFieldContainer[int]
    start_date: str
    end_date: str
    mode: str
    active: bool
    def __init__(self, workspace_id: _Optional[str] = ..., slot_name: _Optional[str] = ..., recipe_id: _Optional[str] = ..., quantity: _Optional[str] = ..., weekdays: _Optional[_Iterable[int]] = ..., start_date: _Optional[str] = ..., end_date: _Optional[str] = ..., mode: _Optional[str] = ..., active: _Optional[bool] = ...) -> None: ...

class CreateTemplateResponse(_message.Message):
    __slots__ = ("template",)
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    template: Template
    def __init__(self, template: _Optional[_Union[Template, _Mapping]] = ...) -> None: ...

class UpdateTemplateRequest(_message.Message):
    __slots__ = ("workspace_id", "id", "expected_revision", "slot_name", "recipe_id", "quantity", "weekdays", "start_date", "end_date", "mode", "active")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    SLOT_NAME_FIELD_NUMBER: _ClassVar[int]
    RECIPE_ID_FIELD_NUMBER: _ClassVar[int]
    QUANTITY_FIELD_NUMBER: _ClassVar[int]
    WEEKDAYS_FIELD_NUMBER: _ClassVar[int]
    START_DATE_FIELD_NUMBER: _ClassVar[int]
    END_DATE_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    id: str
    expected_revision: int
    slot_name: str
    recipe_id: str
    quantity: str
    weekdays: _containers.RepeatedScalarFieldContainer[int]
    start_date: str
    end_date: str
    mode: str
    active: bool
    def __init__(self, workspace_id: _Optional[str] = ..., id: _Optional[str] = ..., expected_revision: _Optional[int] = ..., slot_name: _Optional[str] = ..., recipe_id: _Optional[str] = ..., quantity: _Optional[str] = ..., weekdays: _Optional[_Iterable[int]] = ..., start_date: _Optional[str] = ..., end_date: _Optional[str] = ..., mode: _Optional[str] = ..., active: _Optional[bool] = ...) -> None: ...

class UpdateTemplateResponse(_message.Message):
    __slots__ = ("template",)
    TEMPLATE_FIELD_NUMBER: _ClassVar[int]
    template: Template
    def __init__(self, template: _Optional[_Union[Template, _Mapping]] = ...) -> None: ...

class GenerateOccurrencesRequest(_message.Message):
    __slots__ = ("workspace_id", "from_date", "to_date")
    WORKSPACE_ID_FIELD_NUMBER: _ClassVar[int]
    FROM_DATE_FIELD_NUMBER: _ClassVar[int]
    TO_DATE_FIELD_NUMBER: _ClassVar[int]
    workspace_id: str
    from_date: str
    to_date: str
    def __init__(self, workspace_id: _Optional[str] = ..., from_date: _Optional[str] = ..., to_date: _Optional[str] = ...) -> None: ...

class GenerateOccurrencesResponse(_message.Message):
    __slots__ = ("occurrences",)
    OCCURRENCES_FIELD_NUMBER: _ClassVar[int]
    occurrences: _containers.RepeatedCompositeFieldContainer[Occurrence]
    def __init__(self, occurrences: _Optional[_Iterable[_Union[Occurrence, _Mapping]]] = ...) -> None: ...
