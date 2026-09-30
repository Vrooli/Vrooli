from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DiscoverScenarioRequest(_message.Message):
    __slots__ = ("scenario_name",)
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    def __init__(self, scenario_name: _Optional[str] = ...) -> None: ...

class ImportBrandRequest(_message.Message):
    __slots__ = ("scenario_name",)
    SCENARIO_NAME_FIELD_NUMBER: _ClassVar[int]
    scenario_name: str
    def __init__(self, scenario_name: _Optional[str] = ...) -> None: ...

class DiscoverySource(_message.Message):
    __slots__ = ("file", "type", "confidence", "fields")
    FILE_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    FIELDS_FIELD_NUMBER: _ClassVar[int]
    file: str
    type: str
    confidence: float
    fields: int
    def __init__(self, file: _Optional[str] = ..., type: _Optional[str] = ..., confidence: _Optional[float] = ..., fields: _Optional[int] = ...) -> None: ...

class DraftIdentity(_message.Message):
    __slots__ = ("display_name", "tagline", "logo_path", "favicon_path", "icon_path")
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    TAGLINE_FIELD_NUMBER: _ClassVar[int]
    LOGO_PATH_FIELD_NUMBER: _ClassVar[int]
    FAVICON_PATH_FIELD_NUMBER: _ClassVar[int]
    ICON_PATH_FIELD_NUMBER: _ClassVar[int]
    display_name: str
    tagline: str
    logo_path: str
    favicon_path: str
    icon_path: str
    def __init__(self, display_name: _Optional[str] = ..., tagline: _Optional[str] = ..., logo_path: _Optional[str] = ..., favicon_path: _Optional[str] = ..., icon_path: _Optional[str] = ...) -> None: ...

class DraftColors(_message.Message):
    __slots__ = ("primary", "secondary", "accent", "background", "surface", "text", "error")
    PRIMARY_FIELD_NUMBER: _ClassVar[int]
    SECONDARY_FIELD_NUMBER: _ClassVar[int]
    ACCENT_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_FIELD_NUMBER: _ClassVar[int]
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    primary: str
    secondary: str
    accent: str
    background: str
    surface: str
    text: str
    error: str
    def __init__(self, primary: _Optional[str] = ..., secondary: _Optional[str] = ..., accent: _Optional[str] = ..., background: _Optional[str] = ..., surface: _Optional[str] = ..., text: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...

class DraftBrand(_message.Message):
    __slots__ = ("name", "description", "identity", "colors")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    COLORS_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    identity: DraftIdentity
    colors: DraftColors
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., identity: _Optional[_Union[DraftIdentity, _Mapping]] = ..., colors: _Optional[_Union[DraftColors, _Mapping]] = ...) -> None: ...

class DiscoveryResult(_message.Message):
    __slots__ = ("scenario", "sources", "draft_brand", "confidence", "suggestions")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SOURCES_FIELD_NUMBER: _ClassVar[int]
    DRAFT_BRAND_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    SUGGESTIONS_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    sources: _containers.RepeatedCompositeFieldContainer[DiscoverySource]
    draft_brand: DraftBrand
    confidence: float
    suggestions: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scenario: _Optional[str] = ..., sources: _Optional[_Iterable[_Union[DiscoverySource, _Mapping]]] = ..., draft_brand: _Optional[_Union[DraftBrand, _Mapping]] = ..., confidence: _Optional[float] = ..., suggestions: _Optional[_Iterable[str]] = ...) -> None: ...

class ImportBrandResponse(_message.Message):
    __slots__ = ("brand_id", "brand_name", "brand_version", "sources", "confidence")
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    BRAND_NAME_FIELD_NUMBER: _ClassVar[int]
    BRAND_VERSION_FIELD_NUMBER: _ClassVar[int]
    SOURCES_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    brand_id: str
    brand_name: str
    brand_version: int
    sources: _containers.RepeatedCompositeFieldContainer[DiscoverySource]
    confidence: float
    def __init__(self, brand_id: _Optional[str] = ..., brand_name: _Optional[str] = ..., brand_version: _Optional[int] = ..., sources: _Optional[_Iterable[_Union[DiscoverySource, _Mapping]]] = ..., confidence: _Optional[float] = ...) -> None: ...
