import datetime

from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from landing_page_react_vite.v1 import content_pb2 as _content_pb2
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

class Variant(_message.Message):
    __slots__ = ("id", "slug", "name", "description", "weight", "status", "created_at", "updated_at", "archived_at", "axes", "header_config", "seo_config")
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
    WEIGHT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_AT_FIELD_NUMBER: _ClassVar[int]
    AXES_FIELD_NUMBER: _ClassVar[int]
    HEADER_CONFIG_FIELD_NUMBER: _ClassVar[int]
    SEO_CONFIG_FIELD_NUMBER: _ClassVar[int]
    id: int
    slug: str
    name: str
    description: str
    weight: int
    status: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    archived_at: _timestamp_pb2.Timestamp
    axes: _containers.ScalarMap[str, str]
    header_config: LandingHeaderConfig
    seo_config: VariantSEOConfig
    def __init__(self, id: _Optional[int] = ..., slug: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., weight: _Optional[int] = ..., status: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., archived_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., axes: _Optional[_Mapping[str, str]] = ..., header_config: _Optional[_Union[LandingHeaderConfig, _Mapping]] = ..., seo_config: _Optional[_Union[VariantSEOConfig, _Mapping]] = ...) -> None: ...

class AxesSelection(_message.Message):
    __slots__ = ("values",)
    class ValuesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    VALUES_FIELD_NUMBER: _ClassVar[int]
    values: _containers.ScalarMap[str, str]
    def __init__(self, values: _Optional[_Mapping[str, str]] = ...) -> None: ...

class VariantSnapshot(_message.Message):
    __slots__ = ("slug", "name", "description", "weight", "status", "axes", "header_config", "seo_config", "sections")
    class AxesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SLUG_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    AXES_FIELD_NUMBER: _ClassVar[int]
    HEADER_CONFIG_FIELD_NUMBER: _ClassVar[int]
    SEO_CONFIG_FIELD_NUMBER: _ClassVar[int]
    SECTIONS_FIELD_NUMBER: _ClassVar[int]
    slug: str
    name: str
    description: str
    weight: int
    status: str
    axes: _containers.ScalarMap[str, str]
    header_config: LandingHeaderConfig
    seo_config: VariantSEOConfig
    sections: _containers.RepeatedCompositeFieldContainer[_content_pb2.ContentSection]
    def __init__(self, slug: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., weight: _Optional[int] = ..., status: _Optional[str] = ..., axes: _Optional[_Mapping[str, str]] = ..., header_config: _Optional[_Union[LandingHeaderConfig, _Mapping]] = ..., seo_config: _Optional[_Union[VariantSEOConfig, _Mapping]] = ..., sections: _Optional[_Iterable[_Union[_content_pb2.ContentSection, _Mapping]]] = ...) -> None: ...

class SelectVariantRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetPublicVariantRequest(_message.Message):
    __slots__ = ("slug",)
    SLUG_FIELD_NUMBER: _ClassVar[int]
    slug: str
    def __init__(self, slug: _Optional[str] = ...) -> None: ...

class GetVariantRequest(_message.Message):
    __slots__ = ("slug",)
    SLUG_FIELD_NUMBER: _ClassVar[int]
    slug: str
    def __init__(self, slug: _Optional[str] = ...) -> None: ...

class VariantResponse(_message.Message):
    __slots__ = ("variant",)
    VARIANT_FIELD_NUMBER: _ClassVar[int]
    variant: Variant
    def __init__(self, variant: _Optional[_Union[Variant, _Mapping]] = ...) -> None: ...

class ListVariantsRequest(_message.Message):
    __slots__ = ("status_filter",)
    STATUS_FILTER_FIELD_NUMBER: _ClassVar[int]
    status_filter: str
    def __init__(self, status_filter: _Optional[str] = ...) -> None: ...

class ListVariantsResponse(_message.Message):
    __slots__ = ("variants",)
    VARIANTS_FIELD_NUMBER: _ClassVar[int]
    variants: _containers.RepeatedCompositeFieldContainer[Variant]
    def __init__(self, variants: _Optional[_Iterable[_Union[Variant, _Mapping]]] = ...) -> None: ...

class CreateVariantRequest(_message.Message):
    __slots__ = ("slug", "name", "description", "weight", "axes")
    class AxesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    SLUG_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FIELD_NUMBER: _ClassVar[int]
    AXES_FIELD_NUMBER: _ClassVar[int]
    slug: str
    name: str
    description: str
    weight: int
    axes: _containers.ScalarMap[str, str]
    def __init__(self, slug: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., weight: _Optional[int] = ..., axes: _Optional[_Mapping[str, str]] = ...) -> None: ...

class UpdateVariantRequest(_message.Message):
    __slots__ = ("slug", "name", "description", "weight", "axes", "header_config")
    SLUG_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    WEIGHT_FIELD_NUMBER: _ClassVar[int]
    AXES_FIELD_NUMBER: _ClassVar[int]
    HEADER_CONFIG_FIELD_NUMBER: _ClassVar[int]
    slug: str
    name: str
    description: str
    weight: int
    axes: AxesSelection
    header_config: LandingHeaderConfig
    def __init__(self, slug: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., weight: _Optional[int] = ..., axes: _Optional[_Union[AxesSelection, _Mapping]] = ..., header_config: _Optional[_Union[LandingHeaderConfig, _Mapping]] = ...) -> None: ...

class ArchiveVariantRequest(_message.Message):
    __slots__ = ("slug",)
    SLUG_FIELD_NUMBER: _ClassVar[int]
    slug: str
    def __init__(self, slug: _Optional[str] = ...) -> None: ...

class DeleteVariantRequest(_message.Message):
    __slots__ = ("slug",)
    SLUG_FIELD_NUMBER: _ClassVar[int]
    slug: str
    def __init__(self, slug: _Optional[str] = ...) -> None: ...

class DeleteVariantResponse(_message.Message):
    __slots__ = ("deleted",)
    DELETED_FIELD_NUMBER: _ClassVar[int]
    deleted: bool
    def __init__(self, deleted: _Optional[bool] = ...) -> None: ...

class ExportVariantSnapshotRequest(_message.Message):
    __slots__ = ("slug",)
    SLUG_FIELD_NUMBER: _ClassVar[int]
    slug: str
    def __init__(self, slug: _Optional[str] = ...) -> None: ...

class ExportVariantSnapshotResponse(_message.Message):
    __slots__ = ("snapshot",)
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    snapshot: VariantSnapshot
    def __init__(self, snapshot: _Optional[_Union[VariantSnapshot, _Mapping]] = ...) -> None: ...

class ImportVariantSnapshotRequest(_message.Message):
    __slots__ = ("slug", "snapshot")
    SLUG_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    slug: str
    snapshot: VariantSnapshot
    def __init__(self, slug: _Optional[str] = ..., snapshot: _Optional[_Union[VariantSnapshot, _Mapping]] = ...) -> None: ...
