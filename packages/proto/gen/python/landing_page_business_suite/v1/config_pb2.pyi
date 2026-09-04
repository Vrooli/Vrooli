from google.protobuf import struct_pb2 as _struct_pb2
from landing_page_business_suite.v1.shared import commerce_pb2 as _commerce_pb2
from landing_page_business_suite.v1.shared import downloads_pb2 as _downloads_pb2
from landing_page_business_suite.v1.shared import presentation_pb2 as _presentation_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class LandingVariantSummary(_message.Message):
    __slots__ = ("id", "slug", "name", "description", "axes")
    class AxesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    AXES_FIELD_NUMBER: _ClassVar[int]
    id: int
    slug: str
    name: str
    description: str
    axes: _containers.ScalarMap[str, str]
    def __init__(self, id: _Optional[int] = ..., slug: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., axes: _Optional[_Mapping[str, str]] = ...) -> None: ...

class LandingSection(_message.Message):
    __slots__ = ("section_type", "content", "order", "enabled", "section_key")
    SECTION_TYPE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    ORDER_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    SECTION_KEY_FIELD_NUMBER: _ClassVar[int]
    section_type: str
    content: _struct_pb2.Struct
    order: int
    enabled: bool
    section_key: str
    def __init__(self, section_type: _Optional[str] = ..., content: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., order: _Optional[int] = ..., enabled: _Optional[bool] = ..., section_key: _Optional[str] = ...) -> None: ...

class LandingBranding(_message.Message):
    __slots__ = ("site_name", "tagline", "logo_url", "logo_icon_url", "favicon_url", "theme_primary_color", "theme_background_color", "support_chat_url", "support_email", "coming_soon_enabled", "coming_soon_message")
    SITE_NAME_FIELD_NUMBER: _ClassVar[int]
    TAGLINE_FIELD_NUMBER: _ClassVar[int]
    LOGO_URL_FIELD_NUMBER: _ClassVar[int]
    LOGO_ICON_URL_FIELD_NUMBER: _ClassVar[int]
    FAVICON_URL_FIELD_NUMBER: _ClassVar[int]
    THEME_PRIMARY_COLOR_FIELD_NUMBER: _ClassVar[int]
    THEME_BACKGROUND_COLOR_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_CHAT_URL_FIELD_NUMBER: _ClassVar[int]
    SUPPORT_EMAIL_FIELD_NUMBER: _ClassVar[int]
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
    support_email: str
    coming_soon_enabled: bool
    coming_soon_message: str
    def __init__(self, site_name: _Optional[str] = ..., tagline: _Optional[str] = ..., logo_url: _Optional[str] = ..., logo_icon_url: _Optional[str] = ..., favicon_url: _Optional[str] = ..., theme_primary_color: _Optional[str] = ..., theme_background_color: _Optional[str] = ..., support_chat_url: _Optional[str] = ..., support_email: _Optional[str] = ..., coming_soon_enabled: _Optional[bool] = ..., coming_soon_message: _Optional[str] = ...) -> None: ...

class IntroOffer(_message.Message):
    __slots__ = ("id", "name", "amount_off", "percent_off", "currency", "duration", "duration_in_months", "max_redemptions", "redeem_by", "times_redeemed", "valid", "created", "is_intro_coupon", "intro_tier")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    AMOUNT_OFF_FIELD_NUMBER: _ClassVar[int]
    PERCENT_OFF_FIELD_NUMBER: _ClassVar[int]
    CURRENCY_FIELD_NUMBER: _ClassVar[int]
    DURATION_FIELD_NUMBER: _ClassVar[int]
    DURATION_IN_MONTHS_FIELD_NUMBER: _ClassVar[int]
    MAX_REDEMPTIONS_FIELD_NUMBER: _ClassVar[int]
    REDEEM_BY_FIELD_NUMBER: _ClassVar[int]
    TIMES_REDEEMED_FIELD_NUMBER: _ClassVar[int]
    VALID_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    IS_INTRO_COUPON_FIELD_NUMBER: _ClassVar[int]
    INTRO_TIER_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    amount_off: int
    percent_off: float
    currency: str
    duration: str
    duration_in_months: int
    max_redemptions: int
    redeem_by: int
    times_redeemed: int
    valid: bool
    created: int
    is_intro_coupon: bool
    intro_tier: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., amount_off: _Optional[int] = ..., percent_off: _Optional[float] = ..., currency: _Optional[str] = ..., duration: _Optional[str] = ..., duration_in_months: _Optional[int] = ..., max_redemptions: _Optional[int] = ..., redeem_by: _Optional[int] = ..., times_redeemed: _Optional[int] = ..., valid: _Optional[bool] = ..., created: _Optional[int] = ..., is_intro_coupon: _Optional[bool] = ..., intro_tier: _Optional[str] = ...) -> None: ...

class GetLandingConfigRequest(_message.Message):
    __slots__ = ("variant_slug", "visitor_id")
    VARIANT_SLUG_FIELD_NUMBER: _ClassVar[int]
    VISITOR_ID_FIELD_NUMBER: _ClassVar[int]
    variant_slug: str
    visitor_id: str
    def __init__(self, variant_slug: _Optional[str] = ..., visitor_id: _Optional[str] = ...) -> None: ...

class LandingConfigResponse(_message.Message):
    __slots__ = ("variant", "sections", "pricing", "downloads", "header", "branding", "fallback", "coupon_mappings", "intro_offers")
    class CouponMappingsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    VARIANT_FIELD_NUMBER: _ClassVar[int]
    SECTIONS_FIELD_NUMBER: _ClassVar[int]
    PRICING_FIELD_NUMBER: _ClassVar[int]
    DOWNLOADS_FIELD_NUMBER: _ClassVar[int]
    HEADER_FIELD_NUMBER: _ClassVar[int]
    BRANDING_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_FIELD_NUMBER: _ClassVar[int]
    COUPON_MAPPINGS_FIELD_NUMBER: _ClassVar[int]
    INTRO_OFFERS_FIELD_NUMBER: _ClassVar[int]
    variant: LandingVariantSummary
    sections: _containers.RepeatedCompositeFieldContainer[LandingSection]
    pricing: _commerce_pb2.PricingOverview
    downloads: _containers.RepeatedCompositeFieldContainer[_downloads_pb2.DownloadApp]
    header: _presentation_pb2.LandingHeaderConfig
    branding: LandingBranding
    fallback: bool
    coupon_mappings: _containers.ScalarMap[str, str]
    intro_offers: _containers.RepeatedCompositeFieldContainer[IntroOffer]
    def __init__(self, variant: _Optional[_Union[LandingVariantSummary, _Mapping]] = ..., sections: _Optional[_Iterable[_Union[LandingSection, _Mapping]]] = ..., pricing: _Optional[_Union[_commerce_pb2.PricingOverview, _Mapping]] = ..., downloads: _Optional[_Iterable[_Union[_downloads_pb2.DownloadApp, _Mapping]]] = ..., header: _Optional[_Union[_presentation_pb2.LandingHeaderConfig, _Mapping]] = ..., branding: _Optional[_Union[LandingBranding, _Mapping]] = ..., fallback: _Optional[bool] = ..., coupon_mappings: _Optional[_Mapping[str, str]] = ..., intro_offers: _Optional[_Iterable[_Union[IntroOffer, _Mapping]]] = ...) -> None: ...
