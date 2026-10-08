from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class WidgetSlot(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WIDGET_SLOT_UNSPECIFIED: _ClassVar[WidgetSlot]
    WIDGET_SLOT_INLINE: _ClassVar[WidgetSlot]
    WIDGET_SLOT_SIDEBAR: _ClassVar[WidgetSlot]
    WIDGET_SLOT_FULL: _ClassVar[WidgetSlot]

class WidgetScope(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    WIDGET_SCOPE_UNSPECIFIED: _ClassVar[WidgetScope]
    WIDGET_SCOPE_SCENARIO: _ClassVar[WidgetScope]
    WIDGET_SCOPE_GLOBAL: _ClassVar[WidgetScope]
WIDGET_SLOT_UNSPECIFIED: WidgetSlot
WIDGET_SLOT_INLINE: WidgetSlot
WIDGET_SLOT_SIDEBAR: WidgetSlot
WIDGET_SLOT_FULL: WidgetSlot
WIDGET_SCOPE_UNSPECIFIED: WidgetScope
WIDGET_SCOPE_SCENARIO: WidgetScope
WIDGET_SCOPE_GLOBAL: WidgetScope

class WidgetDeclaration(_message.Message):
    __slots__ = ("widget_id", "component_name", "props_schema_json", "slot", "scope", "description", "file_path")
    WIDGET_ID_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_NAME_FIELD_NUMBER: _ClassVar[int]
    PROPS_SCHEMA_JSON_FIELD_NUMBER: _ClassVar[int]
    SLOT_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    widget_id: str
    component_name: str
    props_schema_json: str
    slot: WidgetSlot
    scope: WidgetScope
    description: str
    file_path: str
    def __init__(self, widget_id: _Optional[str] = ..., component_name: _Optional[str] = ..., props_schema_json: _Optional[str] = ..., slot: _Optional[_Union[WidgetSlot, str]] = ..., scope: _Optional[_Union[WidgetScope, str]] = ..., description: _Optional[str] = ..., file_path: _Optional[str] = ...) -> None: ...
