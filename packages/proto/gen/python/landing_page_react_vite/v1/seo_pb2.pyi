from google.protobuf import struct_pb2 as _struct_pb2
from landing_page_react_vite.v1 import variant_pb2 as _variant_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SEOResponse(_message.Message):
    __slots__ = ("site_name", "title", "description", "og_title", "og_description", "og_image_url", "twitter_card", "canonical_url", "favicon_url", "apple_touch_icon_url", "theme_primary_color", "noindex", "structured_data")
    SITE_NAME_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    OG_TITLE_FIELD_NUMBER: _ClassVar[int]
    OG_DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    OG_IMAGE_URL_FIELD_NUMBER: _ClassVar[int]
    TWITTER_CARD_FIELD_NUMBER: _ClassVar[int]
    CANONICAL_URL_FIELD_NUMBER: _ClassVar[int]
    FAVICON_URL_FIELD_NUMBER: _ClassVar[int]
    APPLE_TOUCH_ICON_URL_FIELD_NUMBER: _ClassVar[int]
    THEME_PRIMARY_COLOR_FIELD_NUMBER: _ClassVar[int]
    NOINDEX_FIELD_NUMBER: _ClassVar[int]
    STRUCTURED_DATA_FIELD_NUMBER: _ClassVar[int]
    site_name: str
    title: str
    description: str
    og_title: str
    og_description: str
    og_image_url: str
    twitter_card: str
    canonical_url: str
    favicon_url: str
    apple_touch_icon_url: str
    theme_primary_color: str
    noindex: bool
    structured_data: _struct_pb2.Struct
    def __init__(self, site_name: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., og_title: _Optional[str] = ..., og_description: _Optional[str] = ..., og_image_url: _Optional[str] = ..., twitter_card: _Optional[str] = ..., canonical_url: _Optional[str] = ..., favicon_url: _Optional[str] = ..., apple_touch_icon_url: _Optional[str] = ..., theme_primary_color: _Optional[str] = ..., noindex: _Optional[bool] = ..., structured_data: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class GetVariantSEORequest(_message.Message):
    __slots__ = ("slug",)
    SLUG_FIELD_NUMBER: _ClassVar[int]
    slug: str
    def __init__(self, slug: _Optional[str] = ...) -> None: ...

class UpdateVariantSEORequest(_message.Message):
    __slots__ = ("slug", "config")
    SLUG_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    slug: str
    config: _variant_pb2.VariantSEOConfig
    def __init__(self, slug: _Optional[str] = ..., config: _Optional[_Union[_variant_pb2.VariantSEOConfig, _Mapping]] = ...) -> None: ...

class UpdateVariantSEOResponse(_message.Message):
    __slots__ = ("success", "updated_at")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    updated_at: str
    def __init__(self, success: _Optional[bool] = ..., updated_at: _Optional[str] = ...) -> None: ...
