import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SiteBranding(_message.Message):
    __slots__ = ("id", "site_name", "tagline", "logo_url", "logo_icon_url", "favicon_url", "apple_touch_icon_url", "default_title", "default_description", "default_og_image_url", "theme_primary_color", "theme_background_color", "canonical_base_url", "google_site_verification", "robots_txt", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    SITE_NAME_FIELD_NUMBER: _ClassVar[int]
    TAGLINE_FIELD_NUMBER: _ClassVar[int]
    LOGO_URL_FIELD_NUMBER: _ClassVar[int]
    LOGO_ICON_URL_FIELD_NUMBER: _ClassVar[int]
    FAVICON_URL_FIELD_NUMBER: _ClassVar[int]
    APPLE_TOUCH_ICON_URL_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_TITLE_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_OG_IMAGE_URL_FIELD_NUMBER: _ClassVar[int]
    THEME_PRIMARY_COLOR_FIELD_NUMBER: _ClassVar[int]
    THEME_BACKGROUND_COLOR_FIELD_NUMBER: _ClassVar[int]
    CANONICAL_BASE_URL_FIELD_NUMBER: _ClassVar[int]
    GOOGLE_SITE_VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    ROBOTS_TXT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: int
    site_name: str
    tagline: str
    logo_url: str
    logo_icon_url: str
    favicon_url: str
    apple_touch_icon_url: str
    default_title: str
    default_description: str
    default_og_image_url: str
    theme_primary_color: str
    theme_background_color: str
    canonical_base_url: str
    google_site_verification: str
    robots_txt: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[int] = ..., site_name: _Optional[str] = ..., tagline: _Optional[str] = ..., logo_url: _Optional[str] = ..., logo_icon_url: _Optional[str] = ..., favicon_url: _Optional[str] = ..., apple_touch_icon_url: _Optional[str] = ..., default_title: _Optional[str] = ..., default_description: _Optional[str] = ..., default_og_image_url: _Optional[str] = ..., theme_primary_color: _Optional[str] = ..., theme_background_color: _Optional[str] = ..., canonical_base_url: _Optional[str] = ..., google_site_verification: _Optional[str] = ..., robots_txt: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class PublicBranding(_message.Message):
    __slots__ = ("site_name", "tagline", "logo_url", "logo_icon_url", "favicon_url", "theme_primary_color", "theme_background_color")
    SITE_NAME_FIELD_NUMBER: _ClassVar[int]
    TAGLINE_FIELD_NUMBER: _ClassVar[int]
    LOGO_URL_FIELD_NUMBER: _ClassVar[int]
    LOGO_ICON_URL_FIELD_NUMBER: _ClassVar[int]
    FAVICON_URL_FIELD_NUMBER: _ClassVar[int]
    THEME_PRIMARY_COLOR_FIELD_NUMBER: _ClassVar[int]
    THEME_BACKGROUND_COLOR_FIELD_NUMBER: _ClassVar[int]
    site_name: str
    tagline: str
    logo_url: str
    logo_icon_url: str
    favicon_url: str
    theme_primary_color: str
    theme_background_color: str
    def __init__(self, site_name: _Optional[str] = ..., tagline: _Optional[str] = ..., logo_url: _Optional[str] = ..., logo_icon_url: _Optional[str] = ..., favicon_url: _Optional[str] = ..., theme_primary_color: _Optional[str] = ..., theme_background_color: _Optional[str] = ...) -> None: ...

class GetBrandingRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetPublicBrandingRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class PublicBrandingResponse(_message.Message):
    __slots__ = ("branding",)
    BRANDING_FIELD_NUMBER: _ClassVar[int]
    branding: PublicBranding
    def __init__(self, branding: _Optional[_Union[PublicBranding, _Mapping]] = ...) -> None: ...

class UpdateBrandingRequest(_message.Message):
    __slots__ = ("site_name", "tagline", "logo_url", "logo_icon_url", "favicon_url", "apple_touch_icon_url", "default_title", "default_description", "default_og_image_url", "theme_primary_color", "theme_background_color", "canonical_base_url", "google_site_verification", "robots_txt")
    SITE_NAME_FIELD_NUMBER: _ClassVar[int]
    TAGLINE_FIELD_NUMBER: _ClassVar[int]
    LOGO_URL_FIELD_NUMBER: _ClassVar[int]
    LOGO_ICON_URL_FIELD_NUMBER: _ClassVar[int]
    FAVICON_URL_FIELD_NUMBER: _ClassVar[int]
    APPLE_TOUCH_ICON_URL_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_TITLE_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_OG_IMAGE_URL_FIELD_NUMBER: _ClassVar[int]
    THEME_PRIMARY_COLOR_FIELD_NUMBER: _ClassVar[int]
    THEME_BACKGROUND_COLOR_FIELD_NUMBER: _ClassVar[int]
    CANONICAL_BASE_URL_FIELD_NUMBER: _ClassVar[int]
    GOOGLE_SITE_VERIFICATION_FIELD_NUMBER: _ClassVar[int]
    ROBOTS_TXT_FIELD_NUMBER: _ClassVar[int]
    site_name: str
    tagline: str
    logo_url: str
    logo_icon_url: str
    favicon_url: str
    apple_touch_icon_url: str
    default_title: str
    default_description: str
    default_og_image_url: str
    theme_primary_color: str
    theme_background_color: str
    canonical_base_url: str
    google_site_verification: str
    robots_txt: str
    def __init__(self, site_name: _Optional[str] = ..., tagline: _Optional[str] = ..., logo_url: _Optional[str] = ..., logo_icon_url: _Optional[str] = ..., favicon_url: _Optional[str] = ..., apple_touch_icon_url: _Optional[str] = ..., default_title: _Optional[str] = ..., default_description: _Optional[str] = ..., default_og_image_url: _Optional[str] = ..., theme_primary_color: _Optional[str] = ..., theme_background_color: _Optional[str] = ..., canonical_base_url: _Optional[str] = ..., google_site_verification: _Optional[str] = ..., robots_txt: _Optional[str] = ...) -> None: ...

class ClearBrandingFieldRequest(_message.Message):
    __slots__ = ("field",)
    FIELD_FIELD_NUMBER: _ClassVar[int]
    field: str
    def __init__(self, field: _Optional[str] = ...) -> None: ...

class BrandingResponse(_message.Message):
    __slots__ = ("branding",)
    BRANDING_FIELD_NUMBER: _ClassVar[int]
    branding: SiteBranding
    def __init__(self, branding: _Optional[_Union[SiteBranding, _Mapping]] = ...) -> None: ...
