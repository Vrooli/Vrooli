import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Brand(_message.Message):
    __slots__ = ("id", "name", "description", "identity", "colors", "typography", "voice", "notes", "version", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    COLORS_FIELD_NUMBER: _ClassVar[int]
    TYPOGRAPHY_FIELD_NUMBER: _ClassVar[int]
    VOICE_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    identity: Identity
    colors: Colors
    typography: Typography
    voice: Voice
    notes: str
    version: int
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., identity: _Optional[_Union[Identity, _Mapping]] = ..., colors: _Optional[_Union[Colors, _Mapping]] = ..., typography: _Optional[_Union[Typography, _Mapping]] = ..., voice: _Optional[_Union[Voice, _Mapping]] = ..., notes: _Optional[str] = ..., version: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class Identity(_message.Message):
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

class Colors(_message.Message):
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

class Typography(_message.Message):
    __slots__ = ("heading_font", "body_font", "mono_font", "base_font_size")
    HEADING_FONT_FIELD_NUMBER: _ClassVar[int]
    BODY_FONT_FIELD_NUMBER: _ClassVar[int]
    MONO_FONT_FIELD_NUMBER: _ClassVar[int]
    BASE_FONT_SIZE_FIELD_NUMBER: _ClassVar[int]
    heading_font: str
    body_font: str
    mono_font: str
    base_font_size: str
    def __init__(self, heading_font: _Optional[str] = ..., body_font: _Optional[str] = ..., mono_font: _Optional[str] = ..., base_font_size: _Optional[str] = ...) -> None: ...

class Voice(_message.Message):
    __slots__ = ("tone", "style", "keywords")
    TONE_FIELD_NUMBER: _ClassVar[int]
    STYLE_FIELD_NUMBER: _ClassVar[int]
    KEYWORDS_FIELD_NUMBER: _ClassVar[int]
    tone: str
    style: str
    keywords: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, tone: _Optional[str] = ..., style: _Optional[str] = ..., keywords: _Optional[_Iterable[str]] = ...) -> None: ...

class BrandVersion(_message.Message):
    __slots__ = ("id", "brand_id", "version", "snapshot", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    brand_id: str
    version: int
    snapshot: str
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., brand_id: _Optional[str] = ..., version: _Optional[int] = ..., snapshot: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListBrandsRequest(_message.Message):
    __slots__ = ("name_contains", "limit", "offset")
    NAME_CONTAINS_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    name_contains: str
    limit: int
    offset: int
    def __init__(self, name_contains: _Optional[str] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListBrandsResponse(_message.Message):
    __slots__ = ("brands",)
    BRANDS_FIELD_NUMBER: _ClassVar[int]
    brands: _containers.RepeatedCompositeFieldContainer[Brand]
    def __init__(self, brands: _Optional[_Iterable[_Union[Brand, _Mapping]]] = ...) -> None: ...

class CreateBrandRequest(_message.Message):
    __slots__ = ("name", "description", "notes", "identity", "colors", "typography", "voice")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    COLORS_FIELD_NUMBER: _ClassVar[int]
    TYPOGRAPHY_FIELD_NUMBER: _ClassVar[int]
    VOICE_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    notes: str
    identity: Identity
    colors: Colors
    typography: Typography
    voice: Voice
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., notes: _Optional[str] = ..., identity: _Optional[_Union[Identity, _Mapping]] = ..., colors: _Optional[_Union[Colors, _Mapping]] = ..., typography: _Optional[_Union[Typography, _Mapping]] = ..., voice: _Optional[_Union[Voice, _Mapping]] = ...) -> None: ...

class CreateBrandResponse(_message.Message):
    __slots__ = ("brand",)
    BRAND_FIELD_NUMBER: _ClassVar[int]
    brand: Brand
    def __init__(self, brand: _Optional[_Union[Brand, _Mapping]] = ...) -> None: ...

class GetBrandRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetBrandResponse(_message.Message):
    __slots__ = ("brand",)
    BRAND_FIELD_NUMBER: _ClassVar[int]
    brand: Brand
    def __init__(self, brand: _Optional[_Union[Brand, _Mapping]] = ...) -> None: ...

class UpdateBrandRequest(_message.Message):
    __slots__ = ("id", "name", "description", "notes", "identity", "colors", "typography", "voice", "expected_version")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    NOTES_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    COLORS_FIELD_NUMBER: _ClassVar[int]
    TYPOGRAPHY_FIELD_NUMBER: _ClassVar[int]
    VOICE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_VERSION_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    notes: str
    identity: Identity
    colors: Colors
    typography: Typography
    voice: Voice
    expected_version: int
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., notes: _Optional[str] = ..., identity: _Optional[_Union[Identity, _Mapping]] = ..., colors: _Optional[_Union[Colors, _Mapping]] = ..., typography: _Optional[_Union[Typography, _Mapping]] = ..., voice: _Optional[_Union[Voice, _Mapping]] = ..., expected_version: _Optional[int] = ...) -> None: ...

class UpdateBrandResponse(_message.Message):
    __slots__ = ("brand",)
    BRAND_FIELD_NUMBER: _ClassVar[int]
    brand: Brand
    def __init__(self, brand: _Optional[_Union[Brand, _Mapping]] = ...) -> None: ...

class DeleteBrandRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteBrandResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListBrandVersionsRequest(_message.Message):
    __slots__ = ("brand_id",)
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    brand_id: str
    def __init__(self, brand_id: _Optional[str] = ...) -> None: ...

class ListBrandVersionsResponse(_message.Message):
    __slots__ = ("versions",)
    VERSIONS_FIELD_NUMBER: _ClassVar[int]
    versions: _containers.RepeatedCompositeFieldContainer[BrandVersion]
    def __init__(self, versions: _Optional[_Iterable[_Union[BrandVersion, _Mapping]]] = ...) -> None: ...
