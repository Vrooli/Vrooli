from google.protobuf import field_mask_pb2 as _field_mask_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Theme(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    THEME_UNSPECIFIED: _ClassVar[Theme]
    THEME_LIGHT: _ClassVar[Theme]
    THEME_DARK: _ClassVar[Theme]
    THEME_SYSTEM: _ClassVar[Theme]

class FontScale(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    FONT_SCALE_UNSPECIFIED: _ClassVar[FontScale]
    FONT_SCALE_SM: _ClassVar[FontScale]
    FONT_SCALE_MD: _ClassVar[FontScale]
    FONT_SCALE_LG: _ClassVar[FontScale]

class Density(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DENSITY_UNSPECIFIED: _ClassVar[Density]
    DENSITY_COMFORTABLE: _ClassVar[Density]
    DENSITY_COMPACT: _ClassVar[Density]
THEME_UNSPECIFIED: Theme
THEME_LIGHT: Theme
THEME_DARK: Theme
THEME_SYSTEM: Theme
FONT_SCALE_UNSPECIFIED: FontScale
FONT_SCALE_SM: FontScale
FONT_SCALE_MD: FontScale
FONT_SCALE_LG: FontScale
DENSITY_UNSPECIFIED: Density
DENSITY_COMFORTABLE: Density
DENSITY_COMPACT: Density

class InventorySortOrder(_message.Message):
    __slots__ = ("key", "dir")
    KEY_FIELD_NUMBER: _ClassVar[int]
    DIR_FIELD_NUMBER: _ClassVar[int]
    key: str
    dir: str
    def __init__(self, key: _Optional[str] = ..., dir: _Optional[str] = ...) -> None: ...

class InventoryFilters(_message.Message):
    __slots__ = ("search", "language", "status", "sort")
    SEARCH_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SORT_FIELD_NUMBER: _ClassVar[int]
    search: str
    language: str
    status: _containers.RepeatedScalarFieldContainer[str]
    sort: InventorySortOrder
    def __init__(self, search: _Optional[str] = ..., language: _Optional[str] = ..., status: _Optional[_Iterable[str]] = ..., sort: _Optional[_Union[InventorySortOrder, _Mapping]] = ...) -> None: ...

class Settings(_message.Message):
    __slots__ = ("principal_id", "theme", "font_scale", "reduced_motion", "rtl", "default_root", "density", "sidebar_width", "inventory_filters", "updated_at")
    PRINCIPAL_ID_FIELD_NUMBER: _ClassVar[int]
    THEME_FIELD_NUMBER: _ClassVar[int]
    FONT_SCALE_FIELD_NUMBER: _ClassVar[int]
    REDUCED_MOTION_FIELD_NUMBER: _ClassVar[int]
    RTL_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_ROOT_FIELD_NUMBER: _ClassVar[int]
    DENSITY_FIELD_NUMBER: _ClassVar[int]
    SIDEBAR_WIDTH_FIELD_NUMBER: _ClassVar[int]
    INVENTORY_FILTERS_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    principal_id: str
    theme: Theme
    font_scale: FontScale
    reduced_motion: bool
    rtl: bool
    default_root: str
    density: Density
    sidebar_width: int
    inventory_filters: InventoryFilters
    updated_at: str
    def __init__(self, principal_id: _Optional[str] = ..., theme: _Optional[_Union[Theme, str]] = ..., font_scale: _Optional[_Union[FontScale, str]] = ..., reduced_motion: _Optional[bool] = ..., rtl: _Optional[bool] = ..., default_root: _Optional[str] = ..., density: _Optional[_Union[Density, str]] = ..., sidebar_width: _Optional[int] = ..., inventory_filters: _Optional[_Union[InventoryFilters, _Mapping]] = ..., updated_at: _Optional[str] = ...) -> None: ...

class GetSettingsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetSettingsResponse(_message.Message):
    __slots__ = ("settings",)
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    settings: Settings
    def __init__(self, settings: _Optional[_Union[Settings, _Mapping]] = ...) -> None: ...

class UpdateSettingsRequest(_message.Message):
    __slots__ = ("settings", "update_mask")
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    UPDATE_MASK_FIELD_NUMBER: _ClassVar[int]
    settings: Settings
    update_mask: _field_mask_pb2.FieldMask
    def __init__(self, settings: _Optional[_Union[Settings, _Mapping]] = ..., update_mask: _Optional[_Union[_field_mask_pb2.FieldMask, _Mapping]] = ...) -> None: ...

class UpdateSettingsResponse(_message.Message):
    __slots__ = ("settings",)
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    settings: Settings
    def __init__(self, settings: _Optional[_Union[Settings, _Mapping]] = ...) -> None: ...
