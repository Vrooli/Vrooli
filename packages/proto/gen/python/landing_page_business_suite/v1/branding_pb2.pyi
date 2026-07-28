import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SiteBranding(_message.Message):
    __slots__ = ("id", "site_name", "tagline", "logo_url", "logo_icon_url", "favicon_url", "apple_touch_icon_url", "default_title", "default_description", "default_og_image_url", "theme_primary_color", "theme_background_color", "canonical_base_url", "google_site_verification", "robots_txt", "created_at", "updated_at", "support_chat_url", "support_email", "smtp_host", "smtp_port", "smtp_username", "smtp_password", "smtp_from", "coming_soon_enabled", "coming_soon_message")
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
    SUPPORT_CHAT_URL_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_EMAIL_FIELD_NUMBER: _ClassVar[int]
    SMTP_HOST_FIELD_NUMBER: _ClassVar[int]
    SMTP_PORT_FIELD_NUMBER: _ClassVar[int]
    SMTP_USERNAME_FIELD_NUMBER: _ClassVar[int]
    SMTP_PASSWORD_FIELD_NUMBER: _ClassVar[int]
    SMTP_FROM_FIELD_NUMBER: _ClassVar[int]
    COMING_SOON_ENABLED_FIELD_NUMBER: _ClassVar[int]
    COMING_SOON_MESSAGE_FIELD_NUMBER: _ClassVar[int]
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
    support_chat_url: str
    support_email: str
    smtp_host: str
    smtp_port: int
    smtp_username: str
    smtp_password: str
    smtp_from: str
    coming_soon_enabled: bool
    coming_soon_message: str
    def __init__(self, id: _Optional[int] = ..., site_name: _Optional[str] = ..., tagline: _Optional[str] = ..., logo_url: _Optional[str] = ..., logo_icon_url: _Optional[str] = ..., favicon_url: _Optional[str] = ..., apple_touch_icon_url: _Optional[str] = ..., default_title: _Optional[str] = ..., default_description: _Optional[str] = ..., default_og_image_url: _Optional[str] = ..., theme_primary_color: _Optional[str] = ..., theme_background_color: _Optional[str] = ..., canonical_base_url: _Optional[str] = ..., google_site_verification: _Optional[str] = ..., robots_txt: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., support_chat_url: _Optional[str] = ..., support_email: _Optional[str] = ..., smtp_host: _Optional[str] = ..., smtp_port: _Optional[int] = ..., smtp_username: _Optional[str] = ..., smtp_password: _Optional[str] = ..., smtp_from: _Optional[str] = ..., coming_soon_enabled: _Optional[bool] = ..., coming_soon_message: _Optional[str] = ...) -> None: ...

class PublicBranding(_message.Message):
    __slots__ = ("site_name", "tagline", "logo_url", "logo_icon_url", "favicon_url", "theme_primary_color", "theme_background_color", "support_chat_url", "coming_soon_enabled", "coming_soon_message")
    SITE_NAME_FIELD_NUMBER: _ClassVar[int]
    TAGLINE_FIELD_NUMBER: _ClassVar[int]
    LOGO_URL_FIELD_NUMBER: _ClassVar[int]
    LOGO_ICON_URL_FIELD_NUMBER: _ClassVar[int]
    FAVICON_URL_FIELD_NUMBER: _ClassVar[int]
    THEME_PRIMARY_COLOR_FIELD_NUMBER: _ClassVar[int]
    THEME_BACKGROUND_COLOR_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_CHAT_URL_FIELD_NUMBER: _ClassVar[int]
    COMING_SOON_ENABLED_FIELD_NUMBER: _ClassVar[int]
    COMING_SOON_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    site_name: str
    tagline: str
    logo_url: str
    logo_icon_url: str
    favicon_url: str
    theme_primary_color: str
    theme_background_color: str
    support_chat_url: str
    coming_soon_enabled: bool
    coming_soon_message: str
    def __init__(self, site_name: _Optional[str] = ..., tagline: _Optional[str] = ..., logo_url: _Optional[str] = ..., logo_icon_url: _Optional[str] = ..., favicon_url: _Optional[str] = ..., theme_primary_color: _Optional[str] = ..., theme_background_color: _Optional[str] = ..., support_chat_url: _Optional[str] = ..., coming_soon_enabled: _Optional[bool] = ..., coming_soon_message: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("site_name", "tagline", "logo_url", "logo_icon_url", "favicon_url", "apple_touch_icon_url", "default_title", "default_description", "default_og_image_url", "theme_primary_color", "theme_background_color", "canonical_base_url", "google_site_verification", "robots_txt", "support_chat_url", "support_email", "smtp_host", "smtp_port", "smtp_username", "smtp_password", "smtp_from", "coming_soon_enabled", "coming_soon_message")
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
    SUPPORT_CHAT_URL_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_EMAIL_FIELD_NUMBER: _ClassVar[int]
    SMTP_HOST_FIELD_NUMBER: _ClassVar[int]
    SMTP_PORT_FIELD_NUMBER: _ClassVar[int]
    SMTP_USERNAME_FIELD_NUMBER: _ClassVar[int]
    SMTP_PASSWORD_FIELD_NUMBER: _ClassVar[int]
    SMTP_FROM_FIELD_NUMBER: _ClassVar[int]
    COMING_SOON_ENABLED_FIELD_NUMBER: _ClassVar[int]
    COMING_SOON_MESSAGE_FIELD_NUMBER: _ClassVar[int]
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
    support_chat_url: str
    support_email: str
    smtp_host: str
    smtp_port: int
    smtp_username: str
    smtp_password: str
    smtp_from: str
    coming_soon_enabled: bool
    coming_soon_message: str
    def __init__(self, site_name: _Optional[str] = ..., tagline: _Optional[str] = ..., logo_url: _Optional[str] = ..., logo_icon_url: _Optional[str] = ..., favicon_url: _Optional[str] = ..., apple_touch_icon_url: _Optional[str] = ..., default_title: _Optional[str] = ..., default_description: _Optional[str] = ..., default_og_image_url: _Optional[str] = ..., theme_primary_color: _Optional[str] = ..., theme_background_color: _Optional[str] = ..., canonical_base_url: _Optional[str] = ..., google_site_verification: _Optional[str] = ..., robots_txt: _Optional[str] = ..., support_chat_url: _Optional[str] = ..., support_email: _Optional[str] = ..., smtp_host: _Optional[str] = ..., smtp_port: _Optional[int] = ..., smtp_username: _Optional[str] = ..., smtp_password: _Optional[str] = ..., smtp_from: _Optional[str] = ..., coming_soon_enabled: _Optional[bool] = ..., coming_soon_message: _Optional[str] = ...) -> None: ...

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
