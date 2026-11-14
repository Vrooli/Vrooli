from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class HeaderBrandingConfig(_message.Message):
    __slots__ = ("mode", "label", "subtitle", "mobile_preference")
    MODE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    SUBTITLE_FIELD_NUMBER: _ClassVar[int]
    MOBILE_PREFERENCE_FIELD_NUMBER: _ClassVar[int]
    mode: str
    label: str
    subtitle: str
    mobile_preference: str
    def __init__(self, mode: _Optional[str] = ..., label: _Optional[str] = ..., subtitle: _Optional[str] = ..., mobile_preference: _Optional[str] = ...) -> None: ...

class HeaderVisibilityConfig(_message.Message):
    __slots__ = ("desktop", "mobile")
    DESKTOP_FIELD_NUMBER: _ClassVar[int]
    MOBILE_FIELD_NUMBER: _ClassVar[int]
    desktop: bool
    mobile: bool
    def __init__(self, desktop: _Optional[bool] = ..., mobile: _Optional[bool] = ...) -> None: ...

class HeaderNavLink(_message.Message):
    __slots__ = ("id", "type", "label", "section_type", "section_id", "anchor", "href", "visible_on", "children")
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    SECTION_TYPE_FIELD_NUMBER: _ClassVar[int]
    SECTION_ID_FIELD_NUMBER: _ClassVar[int]
    ANCHOR_FIELD_NUMBER: _ClassVar[int]
    HREF_FIELD_NUMBER: _ClassVar[int]
    VISIBLE_ON_FIELD_NUMBER: _ClassVar[int]
    CHILDREN_FIELD_NUMBER: _ClassVar[int]
    id: str
    type: str
    label: str
    section_type: str
    section_id: int
    anchor: str
    href: str
    visible_on: HeaderVisibilityConfig
    children: _containers.RepeatedCompositeFieldContainer[HeaderNavLink]
    def __init__(self, id: _Optional[str] = ..., type: _Optional[str] = ..., label: _Optional[str] = ..., section_type: _Optional[str] = ..., section_id: _Optional[int] = ..., anchor: _Optional[str] = ..., href: _Optional[str] = ..., visible_on: _Optional[_Union[HeaderVisibilityConfig, _Mapping]] = ..., children: _Optional[_Iterable[_Union[HeaderNavLink, _Mapping]]] = ...) -> None: ...

class HeaderNavConfig(_message.Message):
    __slots__ = ("links",)
    LINKS_FIELD_NUMBER: _ClassVar[int]
    links: _containers.RepeatedCompositeFieldContainer[HeaderNavLink]
    def __init__(self, links: _Optional[_Iterable[_Union[HeaderNavLink, _Mapping]]] = ...) -> None: ...

class HeaderCTAConfig(_message.Message):
    __slots__ = ("mode", "label", "href", "variant")
    MODE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    HREF_FIELD_NUMBER: _ClassVar[int]
    VARIANT_FIELD_NUMBER: _ClassVar[int]
    mode: str
    label: str
    href: str
    variant: str
    def __init__(self, mode: _Optional[str] = ..., label: _Optional[str] = ..., href: _Optional[str] = ..., variant: _Optional[str] = ...) -> None: ...

class HeaderCTAGroup(_message.Message):
    __slots__ = ("primary", "secondary")
    PRIMARY_FIELD_NUMBER: _ClassVar[int]
    SECONDARY_FIELD_NUMBER: _ClassVar[int]
    primary: HeaderCTAConfig
    secondary: HeaderCTAConfig
    def __init__(self, primary: _Optional[_Union[HeaderCTAConfig, _Mapping]] = ..., secondary: _Optional[_Union[HeaderCTAConfig, _Mapping]] = ...) -> None: ...

class HeaderBehaviorConfig(_message.Message):
    __slots__ = ("sticky", "hide_on_scroll")
    STICKY_FIELD_NUMBER: _ClassVar[int]
    HIDE_ON_SCROLL_FIELD_NUMBER: _ClassVar[int]
    sticky: bool
    hide_on_scroll: bool
    def __init__(self, sticky: _Optional[bool] = ..., hide_on_scroll: _Optional[bool] = ...) -> None: ...

class LandingHeaderConfig(_message.Message):
    __slots__ = ("branding", "nav", "ctas", "behavior")
    BRANDING_FIELD_NUMBER: _ClassVar[int]
    NAV_FIELD_NUMBER: _ClassVar[int]
    CTAS_FIELD_NUMBER: _ClassVar[int]
    BEHAVIOR_FIELD_NUMBER: _ClassVar[int]
    branding: HeaderBrandingConfig
    nav: HeaderNavConfig
    ctas: HeaderCTAGroup
    behavior: HeaderBehaviorConfig
    def __init__(self, branding: _Optional[_Union[HeaderBrandingConfig, _Mapping]] = ..., nav: _Optional[_Union[HeaderNavConfig, _Mapping]] = ..., ctas: _Optional[_Union[HeaderCTAGroup, _Mapping]] = ..., behavior: _Optional[_Union[HeaderBehaviorConfig, _Mapping]] = ...) -> None: ...

class VariantSEOConfig(_message.Message):
    __slots__ = ("title", "description", "og_title", "og_description", "og_image_url", "twitter_card", "canonical_path", "noindex", "structured_data")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    OG_TITLE_FIELD_NUMBER: _ClassVar[int]
    OG_DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    OG_IMAGE_URL_FIELD_NUMBER: _ClassVar[int]
    TWITTER_CARD_FIELD_NUMBER: _ClassVar[int]
    CANONICAL_PATH_FIELD_NUMBER: _ClassVar[int]
    NOINDEX_FIELD_NUMBER: _ClassVar[int]
    STRUCTURED_DATA_FIELD_NUMBER: _ClassVar[int]
    title: str
    description: str
    og_title: str
    og_description: str
    og_image_url: str
    twitter_card: str
    canonical_path: str
    noindex: bool
    structured_data: _struct_pb2.Struct
    def __init__(self, title: _Optional[str] = ..., description: _Optional[str] = ..., og_title: _Optional[str] = ..., og_description: _Optional[str] = ..., og_image_url: _Optional[str] = ..., twitter_card: _Optional[str] = ..., canonical_path: _Optional[str] = ..., noindex: _Optional[bool] = ..., structured_data: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...
