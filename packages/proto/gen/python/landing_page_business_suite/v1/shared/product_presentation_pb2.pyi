from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ProductPresentationDocument(_message.Message):
    __slots__ = ("schema_version", "bundle", "apps", "pages", "assets", "fixtures", "strings")
    class StringsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: PresentationStringTable
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[PresentationStringTable, _Mapping]] = ...) -> None: ...
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_FIELD_NUMBER: _ClassVar[int]
    APPS_FIELD_NUMBER: _ClassVar[int]
    PAGES_FIELD_NUMBER: _ClassVar[int]
    ASSETS_FIELD_NUMBER: _ClassVar[int]
    FIXTURES_FIELD_NUMBER: _ClassVar[int]
    STRINGS_FIELD_NUMBER: _ClassVar[int]
    schema_version: int
    bundle: PresentationBundle
    apps: _containers.RepeatedCompositeFieldContainer[PresentationApp]
    pages: _containers.RepeatedCompositeFieldContainer[PresentationPage]
    assets: _containers.RepeatedCompositeFieldContainer[PresentationAsset]
    fixtures: _containers.RepeatedCompositeFieldContainer[PresentationFixture]
    strings: _containers.MessageMap[str, PresentationStringTable]
    def __init__(self, schema_version: _Optional[int] = ..., bundle: _Optional[_Union[PresentationBundle, _Mapping]] = ..., apps: _Optional[_Iterable[_Union[PresentationApp, _Mapping]]] = ..., pages: _Optional[_Iterable[_Union[PresentationPage, _Mapping]]] = ..., assets: _Optional[_Iterable[_Union[PresentationAsset, _Mapping]]] = ..., fixtures: _Optional[_Iterable[_Union[PresentationFixture, _Mapping]]] = ..., strings: _Optional[_Mapping[str, PresentationStringTable]] = ...) -> None: ...

class PresentationStringTable(_message.Message):
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

class PresentationStringList(_message.Message):
    __slots__ = ("values",)
    VALUES_FIELD_NUMBER: _ClassVar[int]
    values: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, values: _Optional[_Iterable[str]] = ...) -> None: ...

class PresentationBundle(_message.Message):
    __slots__ = ("key", "name", "app_order", "max_app_slides", "page_id", "empty_page_id", "default_locale", "locales")
    KEY_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    APP_ORDER_FIELD_NUMBER: _ClassVar[int]
    MAX_APP_SLIDES_FIELD_NUMBER: _ClassVar[int]
    PAGE_ID_FIELD_NUMBER: _ClassVar[int]
    EMPTY_PAGE_ID_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_LOCALE_FIELD_NUMBER: _ClassVar[int]
    LOCALES_FIELD_NUMBER: _ClassVar[int]
    key: str
    name: str
    app_order: _containers.RepeatedScalarFieldContainer[str]
    max_app_slides: int
    page_id: str
    empty_page_id: str
    default_locale: str
    locales: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, key: _Optional[str] = ..., name: _Optional[str] = ..., app_order: _Optional[_Iterable[str]] = ..., max_app_slides: _Optional[int] = ..., page_id: _Optional[str] = ..., empty_page_id: _Optional[str] = ..., default_locale: _Optional[str] = ..., locales: _Optional[_Iterable[str]] = ...) -> None: ...

class PresentationApp(_message.Message):
    __slots__ = ("key", "slug", "name", "enabled", "visibility", "publication", "page_id", "tagline", "description", "capabilities", "preservation_ref")
    KEY_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    VISIBILITY_FIELD_NUMBER: _ClassVar[int]
    PUBLICATION_FIELD_NUMBER: _ClassVar[int]
    PAGE_ID_FIELD_NUMBER: _ClassVar[int]
    TAGLINE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    PRESERVATION_REF_FIELD_NUMBER: _ClassVar[int]
    key: str
    slug: str
    name: str
    enabled: bool
    visibility: str
    publication: str
    page_id: str
    tagline: str
    description: str
    capabilities: _containers.RepeatedCompositeFieldContainer[PresentationCapability]
    preservation_ref: str
    def __init__(self, key: _Optional[str] = ..., slug: _Optional[str] = ..., name: _Optional[str] = ..., enabled: _Optional[bool] = ..., visibility: _Optional[str] = ..., publication: _Optional[str] = ..., page_id: _Optional[str] = ..., tagline: _Optional[str] = ..., description: _Optional[str] = ..., capabilities: _Optional[_Iterable[_Union[PresentationCapability, _Mapping]]] = ..., preservation_ref: _Optional[str] = ...) -> None: ...

class PresentationCapability(_message.Message):
    __slots__ = ("id", "label", "benefits", "status", "evidence_refs", "owner_qualification", "constraints", "provider_requirements", "platform_requirements", "localized_labels", "localized_benefits", "status_label", "status_labels")
    class LocalizedLabelsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class LocalizedBenefitsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: PresentationStringList
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[PresentationStringList, _Mapping]] = ...) -> None: ...
    class StatusLabelsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    BENEFITS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    OWNER_QUALIFICATION_FIELD_NUMBER: _ClassVar[int]
    CONSTRAINTS_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_REQUIREMENTS_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_REQUIREMENTS_FIELD_NUMBER: _ClassVar[int]
    LOCALIZED_LABELS_FIELD_NUMBER: _ClassVar[int]
    LOCALIZED_BENEFITS_FIELD_NUMBER: _ClassVar[int]
    STATUS_LABEL_FIELD_NUMBER: _ClassVar[int]
    STATUS_LABELS_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    benefits: _containers.RepeatedScalarFieldContainer[str]
    status: str
    evidence_refs: _containers.RepeatedScalarFieldContainer[str]
    owner_qualification: PresentationOwnerQualification
    constraints: _containers.RepeatedScalarFieldContainer[str]
    provider_requirements: _containers.RepeatedScalarFieldContainer[str]
    platform_requirements: _containers.RepeatedScalarFieldContainer[str]
    localized_labels: _containers.ScalarMap[str, str]
    localized_benefits: _containers.MessageMap[str, PresentationStringList]
    status_label: str
    status_labels: _containers.ScalarMap[str, str]
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., benefits: _Optional[_Iterable[str]] = ..., status: _Optional[str] = ..., evidence_refs: _Optional[_Iterable[str]] = ..., owner_qualification: _Optional[_Union[PresentationOwnerQualification, _Mapping]] = ..., constraints: _Optional[_Iterable[str]] = ..., provider_requirements: _Optional[_Iterable[str]] = ..., platform_requirements: _Optional[_Iterable[str]] = ..., localized_labels: _Optional[_Mapping[str, str]] = ..., localized_benefits: _Optional[_Mapping[str, PresentationStringList]] = ..., status_label: _Optional[str] = ..., status_labels: _Optional[_Mapping[str, str]] = ...) -> None: ...

class PresentationOwnerQualification(_message.Message):
    __slots__ = ("owner", "evidence_ref", "release_ref", "qualified")
    OWNER_FIELD_NUMBER: _ClassVar[int]
    EVIDENCE_REF_FIELD_NUMBER: _ClassVar[int]
    RELEASE_REF_FIELD_NUMBER: _ClassVar[int]
    QUALIFIED_FIELD_NUMBER: _ClassVar[int]
    owner: str
    evidence_ref: str
    release_ref: str
    qualified: bool
    def __init__(self, owner: _Optional[str] = ..., evidence_ref: _Optional[str] = ..., release_ref: _Optional[str] = ..., qualified: _Optional[bool] = ...) -> None: ...

class PresentationPage(_message.Message):
    __slots__ = ("id", "locale", "title", "description", "theme", "navigation", "blocks", "footer", "display")
    ID_FIELD_NUMBER: _ClassVar[int]
    LOCALE_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    THEME_FIELD_NUMBER: _ClassVar[int]
    NAVIGATION_FIELD_NUMBER: _ClassVar[int]
    BLOCKS_FIELD_NUMBER: _ClassVar[int]
    FOOTER_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_FIELD_NUMBER: _ClassVar[int]
    id: str
    locale: str
    title: str
    description: str
    theme: PresentationTheme
    navigation: PresentationNavigation
    blocks: _containers.RepeatedCompositeFieldContainer[PresentationBlock]
    footer: PresentationFooter
    display: PresentationPageDisplay
    def __init__(self, id: _Optional[str] = ..., locale: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., theme: _Optional[_Union[PresentationTheme, _Mapping]] = ..., navigation: _Optional[_Union[PresentationNavigation, _Mapping]] = ..., blocks: _Optional[_Iterable[_Union[PresentationBlock, _Mapping]]] = ..., footer: _Optional[_Union[PresentationFooter, _Mapping]] = ..., display: _Optional[_Union[PresentationPageDisplay, _Mapping]] = ...) -> None: ...

class PresentationPageDisplay(_message.Message):
    __slots__ = ("shell", "asset_labels", "fixture_display", "blocks", "apps")
    class AssetLabelsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: PresentationAssetLabel
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[PresentationAssetLabel, _Mapping]] = ...) -> None: ...
    class FixtureDisplayEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: PresentationFixtureDisplay
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[PresentationFixtureDisplay, _Mapping]] = ...) -> None: ...
    class BlocksEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: PresentationBlockDisplay
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[PresentationBlockDisplay, _Mapping]] = ...) -> None: ...
    class AppsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: PresentationAppDisplay
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[PresentationAppDisplay, _Mapping]] = ...) -> None: ...
    SHELL_FIELD_NUMBER: _ClassVar[int]
    ASSET_LABELS_FIELD_NUMBER: _ClassVar[int]
    FIXTURE_DISPLAY_FIELD_NUMBER: _ClassVar[int]
    BLOCKS_FIELD_NUMBER: _ClassVar[int]
    APPS_FIELD_NUMBER: _ClassVar[int]
    shell: PresentationShellDisplay
    asset_labels: _containers.MessageMap[str, PresentationAssetLabel]
    fixture_display: _containers.MessageMap[str, PresentationFixtureDisplay]
    blocks: _containers.MessageMap[str, PresentationBlockDisplay]
    apps: _containers.MessageMap[str, PresentationAppDisplay]
    def __init__(self, shell: _Optional[_Union[PresentationShellDisplay, _Mapping]] = ..., asset_labels: _Optional[_Mapping[str, PresentationAssetLabel]] = ..., fixture_display: _Optional[_Mapping[str, PresentationFixtureDisplay]] = ..., blocks: _Optional[_Mapping[str, PresentationBlockDisplay]] = ..., apps: _Optional[_Mapping[str, PresentationAppDisplay]] = ...) -> None: ...

class PresentationShellDisplay(_message.Message):
    __slots__ = ("brand_name", "brand_mark", "brand_target", "brand_subtitle", "skip_label", "menu_label", "footer_brand_name", "footer_brand_mark", "footer_brand_target", "footer_tagline", "copyright", "footer_note", "unavailable_reason", "preview_label", "header_action")
    BRAND_NAME_FIELD_NUMBER: _ClassVar[int]
    BRAND_MARK_FIELD_NUMBER: _ClassVar[int]
    BRAND_TARGET_FIELD_NUMBER: _ClassVar[int]
    BRAND_SUBTITLE_FIELD_NUMBER: _ClassVar[int]
    SKIP_LABEL_FIELD_NUMBER: _ClassVar[int]
    MENU_LABEL_FIELD_NUMBER: _ClassVar[int]
    FOOTER_BRAND_NAME_FIELD_NUMBER: _ClassVar[int]
    FOOTER_BRAND_MARK_FIELD_NUMBER: _ClassVar[int]
    FOOTER_BRAND_TARGET_FIELD_NUMBER: _ClassVar[int]
    FOOTER_TAGLINE_FIELD_NUMBER: _ClassVar[int]
    COPYRIGHT_FIELD_NUMBER: _ClassVar[int]
    FOOTER_NOTE_FIELD_NUMBER: _ClassVar[int]
    UNAVAILABLE_REASON_FIELD_NUMBER: _ClassVar[int]
    PREVIEW_LABEL_FIELD_NUMBER: _ClassVar[int]
    HEADER_ACTION_FIELD_NUMBER: _ClassVar[int]
    brand_name: str
    brand_mark: str
    brand_target: str
    brand_subtitle: str
    skip_label: str
    menu_label: str
    footer_brand_name: str
    footer_brand_mark: str
    footer_brand_target: str
    footer_tagline: str
    copyright: str
    footer_note: str
    unavailable_reason: str
    preview_label: str
    header_action: PresentationAction
    def __init__(self, brand_name: _Optional[str] = ..., brand_mark: _Optional[str] = ..., brand_target: _Optional[str] = ..., brand_subtitle: _Optional[str] = ..., skip_label: _Optional[str] = ..., menu_label: _Optional[str] = ..., footer_brand_name: _Optional[str] = ..., footer_brand_mark: _Optional[str] = ..., footer_brand_target: _Optional[str] = ..., footer_tagline: _Optional[str] = ..., copyright: _Optional[str] = ..., footer_note: _Optional[str] = ..., unavailable_reason: _Optional[str] = ..., preview_label: _Optional[str] = ..., header_action: _Optional[_Union[PresentationAction, _Mapping]] = ...) -> None: ...

class PresentationAssetLabel(_message.Message):
    __slots__ = ("alt", "sizes")
    ALT_FIELD_NUMBER: _ClassVar[int]
    SIZES_FIELD_NUMBER: _ClassVar[int]
    alt: str
    sizes: str
    def __init__(self, alt: _Optional[str] = ..., sizes: _Optional[str] = ...) -> None: ...

class PresentationFixtureDisplay(_message.Message):
    __slots__ = ("mark", "avatar", "time", "tabs_label", "terminal_label", "messages_label", "file_changes")
    class FileChangesEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    MARK_FIELD_NUMBER: _ClassVar[int]
    AVATAR_FIELD_NUMBER: _ClassVar[int]
    TIME_FIELD_NUMBER: _ClassVar[int]
    TABS_LABEL_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_LABEL_FIELD_NUMBER: _ClassVar[int]
    MESSAGES_LABEL_FIELD_NUMBER: _ClassVar[int]
    FILE_CHANGES_FIELD_NUMBER: _ClassVar[int]
    mark: str
    avatar: str
    time: str
    tabs_label: str
    terminal_label: str
    messages_label: str
    file_changes: _containers.ScalarMap[str, str]
    def __init__(self, mark: _Optional[str] = ..., avatar: _Optional[str] = ..., time: _Optional[str] = ..., tabs_label: _Optional[str] = ..., terminal_label: _Optional[str] = ..., messages_label: _Optional[str] = ..., file_changes: _Optional[_Mapping[str, str]] = ...) -> None: ...

class PresentationBlockDisplay(_message.Message):
    __slots__ = ("eyebrow", "description", "note", "accessibility_label", "badge", "mark", "formats", "anchors", "heading_breaks", "fixture_ref", "hero_fixture_refs")
    class AnchorsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class HeroFixtureRefsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    EYEBROW_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    ACCESSIBILITY_LABEL_FIELD_NUMBER: _ClassVar[int]
    BADGE_FIELD_NUMBER: _ClassVar[int]
    MARK_FIELD_NUMBER: _ClassVar[int]
    FORMATS_FIELD_NUMBER: _ClassVar[int]
    ANCHORS_FIELD_NUMBER: _ClassVar[int]
    HEADING_BREAKS_FIELD_NUMBER: _ClassVar[int]
    FIXTURE_REF_FIELD_NUMBER: _ClassVar[int]
    HERO_FIXTURE_REFS_FIELD_NUMBER: _ClassVar[int]
    eyebrow: str
    description: str
    note: str
    accessibility_label: str
    badge: str
    mark: str
    formats: _containers.RepeatedScalarFieldContainer[str]
    anchors: _containers.ScalarMap[str, str]
    heading_breaks: _containers.RepeatedScalarFieldContainer[int]
    fixture_ref: str
    hero_fixture_refs: _containers.ScalarMap[str, str]
    def __init__(self, eyebrow: _Optional[str] = ..., description: _Optional[str] = ..., note: _Optional[str] = ..., accessibility_label: _Optional[str] = ..., badge: _Optional[str] = ..., mark: _Optional[str] = ..., formats: _Optional[_Iterable[str]] = ..., anchors: _Optional[_Mapping[str, str]] = ..., heading_breaks: _Optional[_Iterable[int]] = ..., fixture_ref: _Optional[str] = ..., hero_fixture_refs: _Optional[_Mapping[str, str]] = ...) -> None: ...

class PresentationAppDisplay(_message.Message):
    __slots__ = ("fixture_ref", "visual_ref", "mark", "tone", "detail_label")
    FIXTURE_REF_FIELD_NUMBER: _ClassVar[int]
    VISUAL_REF_FIELD_NUMBER: _ClassVar[int]
    MARK_FIELD_NUMBER: _ClassVar[int]
    TONE_FIELD_NUMBER: _ClassVar[int]
    DETAIL_LABEL_FIELD_NUMBER: _ClassVar[int]
    fixture_ref: str
    visual_ref: str
    mark: str
    tone: str
    detail_label: str
    def __init__(self, fixture_ref: _Optional[str] = ..., visual_ref: _Optional[str] = ..., mark: _Optional[str] = ..., tone: _Optional[str] = ..., detail_label: _Optional[str] = ...) -> None: ...

class PresentationTheme(_message.Message):
    __slots__ = ("variant", "primary", "background", "accent")
    VARIANT_FIELD_NUMBER: _ClassVar[int]
    PRIMARY_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_FIELD_NUMBER: _ClassVar[int]
    ACCENT_FIELD_NUMBER: _ClassVar[int]
    variant: str
    primary: str
    background: str
    accent: str
    def __init__(self, variant: _Optional[str] = ..., primary: _Optional[str] = ..., background: _Optional[str] = ..., accent: _Optional[str] = ...) -> None: ...

class PresentationNavigation(_message.Message):
    __slots__ = ("label", "items")
    LABEL_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    label: str
    items: _containers.RepeatedCompositeFieldContainer[PresentationNavigationItem]
    def __init__(self, label: _Optional[str] = ..., items: _Optional[_Iterable[_Union[PresentationNavigationItem, _Mapping]]] = ...) -> None: ...

class PresentationNavigationItem(_message.Message):
    __slots__ = ("label", "accessible_label", "target")
    LABEL_FIELD_NUMBER: _ClassVar[int]
    ACCESSIBLE_LABEL_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    label: str
    accessible_label: str
    target: str
    def __init__(self, label: _Optional[str] = ..., accessible_label: _Optional[str] = ..., target: _Optional[str] = ...) -> None: ...

class PresentationFooter(_message.Message):
    __slots__ = ("label", "links")
    LABEL_FIELD_NUMBER: _ClassVar[int]
    LINKS_FIELD_NUMBER: _ClassVar[int]
    label: str
    links: _containers.RepeatedCompositeFieldContainer[PresentationNavigationItem]
    def __init__(self, label: _Optional[str] = ..., links: _Optional[_Iterable[_Union[PresentationNavigationItem, _Mapping]]] = ...) -> None: ...

class PresentationBlock(_message.Message):
    __slots__ = ("id", "kind", "version", "variant", "content")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    VARIANT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    version: int
    variant: str
    content: PresentationBlockContent
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., version: _Optional[int] = ..., variant: _Optional[str] = ..., content: _Optional[_Union[PresentationBlockContent, _Mapping]] = ...) -> None: ...

class PresentationBlockContent(_message.Message):
    __slots__ = ("product_hero", "bundle_hero", "capability_strip", "product_story", "product_demo", "app_spotlights", "artifact_explorer", "voice_story", "device_story", "capability_roadmap", "pricing", "closing_action", "faq", "footer")
    PRODUCT_HERO_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_HERO_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_STRIP_FIELD_NUMBER: _ClassVar[int]
    PRODUCT_STORY_FIELD_NUMBER: _ClassVar[int]
    PRODUCT_DEMO_FIELD_NUMBER: _ClassVar[int]
    APP_SPOTLIGHTS_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_EXPLORER_FIELD_NUMBER: _ClassVar[int]
    VOICE_STORY_FIELD_NUMBER: _ClassVar[int]
    DEVICE_STORY_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_ROADMAP_FIELD_NUMBER: _ClassVar[int]
    PRICING_FIELD_NUMBER: _ClassVar[int]
    CLOSING_ACTION_FIELD_NUMBER: _ClassVar[int]
    FAQ_FIELD_NUMBER: _ClassVar[int]
    FOOTER_FIELD_NUMBER: _ClassVar[int]
    product_hero: PresentationProductHero
    bundle_hero: PresentationBundleHero
    capability_strip: PresentationCapabilityStrip
    product_story: PresentationProductStory
    product_demo: PresentationProductDemo
    app_spotlights: PresentationAppSpotlights
    artifact_explorer: PresentationArtifactExplorer
    voice_story: PresentationVoiceStory
    device_story: PresentationDeviceStory
    capability_roadmap: PresentationCapabilityRoadmap
    pricing: PresentationPricing
    closing_action: PresentationClosingAction
    faq: PresentationFAQ
    footer: PresentationFooter
    def __init__(self, product_hero: _Optional[_Union[PresentationProductHero, _Mapping]] = ..., bundle_hero: _Optional[_Union[PresentationBundleHero, _Mapping]] = ..., capability_strip: _Optional[_Union[PresentationCapabilityStrip, _Mapping]] = ..., product_story: _Optional[_Union[PresentationProductStory, _Mapping]] = ..., product_demo: _Optional[_Union[PresentationProductDemo, _Mapping]] = ..., app_spotlights: _Optional[_Union[PresentationAppSpotlights, _Mapping]] = ..., artifact_explorer: _Optional[_Union[PresentationArtifactExplorer, _Mapping]] = ..., voice_story: _Optional[_Union[PresentationVoiceStory, _Mapping]] = ..., device_story: _Optional[_Union[PresentationDeviceStory, _Mapping]] = ..., capability_roadmap: _Optional[_Union[PresentationCapabilityRoadmap, _Mapping]] = ..., pricing: _Optional[_Union[PresentationPricing, _Mapping]] = ..., closing_action: _Optional[_Union[PresentationClosingAction, _Mapping]] = ..., faq: _Optional[_Union[PresentationFAQ, _Mapping]] = ..., footer: _Optional[_Union[PresentationFooter, _Mapping]] = ...) -> None: ...

class PresentationAction(_message.Message):
    __slots__ = ("kind", "label", "accessible_label", "target", "plan_ref", "app_key", "reason")
    KIND_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    ACCESSIBLE_LABEL_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    PLAN_REF_FIELD_NUMBER: _ClassVar[int]
    APP_KEY_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    kind: str
    label: str
    accessible_label: str
    target: str
    plan_ref: str
    app_key: str
    reason: str
    def __init__(self, kind: _Optional[str] = ..., label: _Optional[str] = ..., accessible_label: _Optional[str] = ..., target: _Optional[str] = ..., plan_ref: _Optional[str] = ..., app_key: _Optional[str] = ..., reason: _Optional[str] = ...) -> None: ...

class PresentationHeroItem(_message.Message):
    __slots__ = ("app_key", "visual_ref", "exhibit_kind", "detail_label")
    APP_KEY_FIELD_NUMBER: _ClassVar[int]
    VISUAL_REF_FIELD_NUMBER: _ClassVar[int]
    EXHIBIT_KIND_FIELD_NUMBER: _ClassVar[int]
    DETAIL_LABEL_FIELD_NUMBER: _ClassVar[int]
    app_key: str
    visual_ref: str
    exhibit_kind: str
    detail_label: str
    def __init__(self, app_key: _Optional[str] = ..., visual_ref: _Optional[str] = ..., exhibit_kind: _Optional[str] = ..., detail_label: _Optional[str] = ...) -> None: ...

class PresentationProductHero(_message.Message):
    __slots__ = ("app_key", "eyebrow", "title", "description", "visual_ref", "fixture_ref", "accessibility_label", "actions")
    APP_KEY_FIELD_NUMBER: _ClassVar[int]
    EYEBROW_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    VISUAL_REF_FIELD_NUMBER: _ClassVar[int]
    FIXTURE_REF_FIELD_NUMBER: _ClassVar[int]
    ACCESSIBILITY_LABEL_FIELD_NUMBER: _ClassVar[int]
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    app_key: str
    eyebrow: str
    title: str
    description: str
    visual_ref: str
    fixture_ref: str
    accessibility_label: str
    actions: _containers.RepeatedCompositeFieldContainer[PresentationAction]
    def __init__(self, app_key: _Optional[str] = ..., eyebrow: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., visual_ref: _Optional[str] = ..., fixture_ref: _Optional[str] = ..., accessibility_label: _Optional[str] = ..., actions: _Optional[_Iterable[_Union[PresentationAction, _Mapping]]] = ...) -> None: ...

class PresentationBundleHero(_message.Message):
    __slots__ = ("eyebrow", "title", "description", "accessibility_label", "hero_items", "actions")
    EYEBROW_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ACCESSIBILITY_LABEL_FIELD_NUMBER: _ClassVar[int]
    HERO_ITEMS_FIELD_NUMBER: _ClassVar[int]
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    eyebrow: str
    title: str
    description: str
    accessibility_label: str
    hero_items: _containers.RepeatedCompositeFieldContainer[PresentationHeroItem]
    actions: _containers.RepeatedCompositeFieldContainer[PresentationAction]
    def __init__(self, eyebrow: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., accessibility_label: _Optional[str] = ..., hero_items: _Optional[_Iterable[_Union[PresentationHeroItem, _Mapping]]] = ..., actions: _Optional[_Iterable[_Union[PresentationAction, _Mapping]]] = ...) -> None: ...

class PresentationCapabilityItem(_message.Message):
    __slots__ = ("capability_id", "label", "description")
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    capability_id: str
    label: str
    description: str
    def __init__(self, capability_id: _Optional[str] = ..., label: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class PresentationCapabilityStrip(_message.Message):
    __slots__ = ("heading", "items")
    HEADING_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    heading: str
    items: _containers.RepeatedCompositeFieldContainer[PresentationCapabilityItem]
    def __init__(self, heading: _Optional[str] = ..., items: _Optional[_Iterable[_Union[PresentationCapabilityItem, _Mapping]]] = ...) -> None: ...

class PresentationStoryItem(_message.Message):
    __slots__ = ("title", "description", "visual_ref", "alt_text")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    VISUAL_REF_FIELD_NUMBER: _ClassVar[int]
    ALT_TEXT_FIELD_NUMBER: _ClassVar[int]
    title: str
    description: str
    visual_ref: str
    alt_text: str
    def __init__(self, title: _Optional[str] = ..., description: _Optional[str] = ..., visual_ref: _Optional[str] = ..., alt_text: _Optional[str] = ...) -> None: ...

class PresentationProductStory(_message.Message):
    __slots__ = ("heading", "body", "items")
    HEADING_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    heading: str
    body: str
    items: _containers.RepeatedCompositeFieldContainer[PresentationStoryItem]
    def __init__(self, heading: _Optional[str] = ..., body: _Optional[str] = ..., items: _Optional[_Iterable[_Union[PresentationStoryItem, _Mapping]]] = ...) -> None: ...

class PresentationProductDemo(_message.Message):
    __slots__ = ("heading", "description", "renderer_ref", "fixture_ref", "poster_ref", "media_ref", "alt_text")
    HEADING_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    RENDERER_REF_FIELD_NUMBER: _ClassVar[int]
    FIXTURE_REF_FIELD_NUMBER: _ClassVar[int]
    POSTER_REF_FIELD_NUMBER: _ClassVar[int]
    MEDIA_REF_FIELD_NUMBER: _ClassVar[int]
    ALT_TEXT_FIELD_NUMBER: _ClassVar[int]
    heading: str
    description: str
    renderer_ref: str
    fixture_ref: str
    poster_ref: str
    media_ref: str
    alt_text: str
    def __init__(self, heading: _Optional[str] = ..., description: _Optional[str] = ..., renderer_ref: _Optional[str] = ..., fixture_ref: _Optional[str] = ..., poster_ref: _Optional[str] = ..., media_ref: _Optional[str] = ..., alt_text: _Optional[str] = ...) -> None: ...

class PresentationAppSpotlights(_message.Message):
    __slots__ = ("heading", "app_keys", "detail_link_label")
    HEADING_FIELD_NUMBER: _ClassVar[int]
    APP_KEYS_FIELD_NUMBER: _ClassVar[int]
    DETAIL_LINK_LABEL_FIELD_NUMBER: _ClassVar[int]
    heading: str
    app_keys: _containers.RepeatedScalarFieldContainer[str]
    detail_link_label: str
    def __init__(self, heading: _Optional[str] = ..., app_keys: _Optional[_Iterable[str]] = ..., detail_link_label: _Optional[str] = ...) -> None: ...

class PresentationArtifactExample(_message.Message):
    __slots__ = ("id", "kind", "label", "filename", "type", "title", "body", "steps", "checks", "caption", "alt_text", "icon", "width", "height", "source_ref", "source", "asset_ref", "brand", "accent", "frames", "preview_ref")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    FILENAME_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    CAPTION_FIELD_NUMBER: _ClassVar[int]
    ALT_TEXT_FIELD_NUMBER: _ClassVar[int]
    ICON_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_REF_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    ASSET_REF_FIELD_NUMBER: _ClassVar[int]
    BRAND_FIELD_NUMBER: _ClassVar[int]
    ACCENT_FIELD_NUMBER: _ClassVar[int]
    FRAMES_FIELD_NUMBER: _ClassVar[int]
    PREVIEW_REF_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    label: str
    filename: str
    type: str
    title: str
    body: str
    steps: _containers.RepeatedScalarFieldContainer[str]
    checks: _containers.RepeatedScalarFieldContainer[str]
    caption: str
    alt_text: str
    icon: str
    width: int
    height: int
    source_ref: str
    source: PresentationArtifactSource
    asset_ref: str
    brand: str
    accent: str
    frames: _containers.RepeatedCompositeFieldContainer[PresentationArtifactFrame]
    preview_ref: str
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., label: _Optional[str] = ..., filename: _Optional[str] = ..., type: _Optional[str] = ..., title: _Optional[str] = ..., body: _Optional[str] = ..., steps: _Optional[_Iterable[str]] = ..., checks: _Optional[_Iterable[str]] = ..., caption: _Optional[str] = ..., alt_text: _Optional[str] = ..., icon: _Optional[str] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., source_ref: _Optional[str] = ..., source: _Optional[_Union[PresentationArtifactSource, _Mapping]] = ..., asset_ref: _Optional[str] = ..., brand: _Optional[str] = ..., accent: _Optional[str] = ..., frames: _Optional[_Iterable[_Union[PresentationArtifactFrame, _Mapping]]] = ..., preview_ref: _Optional[str] = ...) -> None: ...

class PresentationArtifactSource(_message.Message):
    __slots__ = ("ref", "media_ref", "text_excerpt", "integrity_hash", "isolation")
    REF_FIELD_NUMBER: _ClassVar[int]
    MEDIA_REF_FIELD_NUMBER: _ClassVar[int]
    TEXT_EXCERPT_FIELD_NUMBER: _ClassVar[int]
    INTEGRITY_HASH_FIELD_NUMBER: _ClassVar[int]
    ISOLATION_FIELD_NUMBER: _ClassVar[int]
    ref: str
    media_ref: str
    text_excerpt: str
    integrity_hash: str
    isolation: str
    def __init__(self, ref: _Optional[str] = ..., media_ref: _Optional[str] = ..., text_excerpt: _Optional[str] = ..., integrity_hash: _Optional[str] = ..., isolation: _Optional[str] = ...) -> None: ...

class PresentationArtifactFrame(_message.Message):
    __slots__ = ("asset_ref", "time", "label")
    ASSET_REF_FIELD_NUMBER: _ClassVar[int]
    TIME_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    asset_ref: str
    time: str
    label: str
    def __init__(self, asset_ref: _Optional[str] = ..., time: _Optional[str] = ..., label: _Optional[str] = ...) -> None: ...

class PresentationArtifactExplorer(_message.Message):
    __slots__ = ("heading", "capability_id", "examples", "selected_example_id")
    HEADING_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    EXAMPLES_FIELD_NUMBER: _ClassVar[int]
    SELECTED_EXAMPLE_ID_FIELD_NUMBER: _ClassVar[int]
    heading: str
    capability_id: str
    examples: _containers.RepeatedCompositeFieldContainer[PresentationArtifactExample]
    selected_example_id: str
    def __init__(self, heading: _Optional[str] = ..., capability_id: _Optional[str] = ..., examples: _Optional[_Iterable[_Union[PresentationArtifactExample, _Mapping]]] = ..., selected_example_id: _Optional[str] = ...) -> None: ...

class PresentationVoiceFeature(_message.Message):
    __slots__ = ("title", "description", "capability_id")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_ID_FIELD_NUMBER: _ClassVar[int]
    title: str
    description: str
    capability_id: str
    def __init__(self, title: _Optional[str] = ..., description: _Optional[str] = ..., capability_id: _Optional[str] = ...) -> None: ...

class PresentationVoiceStory(_message.Message):
    __slots__ = ("eyebrow", "heading", "body", "features", "note", "input_label", "transcript", "summary_label", "summary_title", "summary_items", "output_label", "demo_note", "provider_qualification", "waveform", "capability_ids")
    EYEBROW_FIELD_NUMBER: _ClassVar[int]
    HEADING_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    FEATURES_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    INPUT_LABEL_FIELD_NUMBER: _ClassVar[int]
    TRANSCRIPT_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_LABEL_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_TITLE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_ITEMS_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_LABEL_FIELD_NUMBER: _ClassVar[int]
    DEMO_NOTE_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_QUALIFICATION_FIELD_NUMBER: _ClassVar[int]
    WAVEFORM_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_IDS_FIELD_NUMBER: _ClassVar[int]
    eyebrow: str
    heading: str
    body: str
    features: _containers.RepeatedCompositeFieldContainer[PresentationVoiceFeature]
    note: str
    input_label: str
    transcript: str
    summary_label: str
    summary_title: str
    summary_items: _containers.RepeatedScalarFieldContainer[str]
    output_label: str
    demo_note: str
    provider_qualification: str
    waveform: _containers.RepeatedScalarFieldContainer[float]
    capability_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, eyebrow: _Optional[str] = ..., heading: _Optional[str] = ..., body: _Optional[str] = ..., features: _Optional[_Iterable[_Union[PresentationVoiceFeature, _Mapping]]] = ..., note: _Optional[str] = ..., input_label: _Optional[str] = ..., transcript: _Optional[str] = ..., summary_label: _Optional[str] = ..., summary_title: _Optional[str] = ..., summary_items: _Optional[_Iterable[str]] = ..., output_label: _Optional[str] = ..., demo_note: _Optional[str] = ..., provider_qualification: _Optional[str] = ..., waveform: _Optional[_Iterable[float]] = ..., capability_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class PresentationDeviceStory(_message.Message):
    __slots__ = ("heading", "description", "device", "visual_ref", "alt_text")
    HEADING_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    DEVICE_FIELD_NUMBER: _ClassVar[int]
    VISUAL_REF_FIELD_NUMBER: _ClassVar[int]
    ALT_TEXT_FIELD_NUMBER: _ClassVar[int]
    heading: str
    description: str
    device: str
    visual_ref: str
    alt_text: str
    def __init__(self, heading: _Optional[str] = ..., description: _Optional[str] = ..., device: _Optional[str] = ..., visual_ref: _Optional[str] = ..., alt_text: _Optional[str] = ...) -> None: ...

class PresentationCapabilityRoadmap(_message.Message):
    __slots__ = ("heading", "capability_ids", "status_label", "description")
    HEADING_FIELD_NUMBER: _ClassVar[int]
    CAPABILITY_IDS_FIELD_NUMBER: _ClassVar[int]
    STATUS_LABEL_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    heading: str
    capability_ids: _containers.RepeatedScalarFieldContainer[str]
    status_label: str
    description: str
    def __init__(self, heading: _Optional[str] = ..., capability_ids: _Optional[_Iterable[str]] = ..., status_label: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class PresentationPricing(_message.Message):
    __slots__ = ("heading", "description", "plan_refs", "actions")
    HEADING_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PLAN_REFS_FIELD_NUMBER: _ClassVar[int]
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    heading: str
    description: str
    plan_refs: _containers.RepeatedScalarFieldContainer[str]
    actions: _containers.RepeatedCompositeFieldContainer[PresentationAction]
    def __init__(self, heading: _Optional[str] = ..., description: _Optional[str] = ..., plan_refs: _Optional[_Iterable[str]] = ..., actions: _Optional[_Iterable[_Union[PresentationAction, _Mapping]]] = ...) -> None: ...

class PresentationClosingAction(_message.Message):
    __slots__ = ("heading", "description", "actions", "visual_ref")
    HEADING_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    VISUAL_REF_FIELD_NUMBER: _ClassVar[int]
    heading: str
    description: str
    actions: _containers.RepeatedCompositeFieldContainer[PresentationAction]
    visual_ref: str
    def __init__(self, heading: _Optional[str] = ..., description: _Optional[str] = ..., actions: _Optional[_Iterable[_Union[PresentationAction, _Mapping]]] = ..., visual_ref: _Optional[str] = ...) -> None: ...

class PresentationFAQItem(_message.Message):
    __slots__ = ("question", "answer", "accessible_label")
    QUESTION_FIELD_NUMBER: _ClassVar[int]
    ANSWER_FIELD_NUMBER: _ClassVar[int]
    ACCESSIBLE_LABEL_FIELD_NUMBER: _ClassVar[int]
    question: str
    answer: str
    accessible_label: str
    def __init__(self, question: _Optional[str] = ..., answer: _Optional[str] = ..., accessible_label: _Optional[str] = ...) -> None: ...

class PresentationFAQ(_message.Message):
    __slots__ = ("heading", "items")
    HEADING_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    heading: str
    items: _containers.RepeatedCompositeFieldContainer[PresentationFAQItem]
    def __init__(self, heading: _Optional[str] = ..., items: _Optional[_Iterable[_Union[PresentationFAQItem, _Mapping]]] = ...) -> None: ...

class PresentationFixture(_message.Message):
    __slots__ = ("id", "kind", "workspace", "backdrop", "workflow")
    ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    WORKSPACE_FIELD_NUMBER: _ClassVar[int]
    BACKDROP_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    id: str
    kind: str
    workspace: PresentationWorkspaceFixture
    backdrop: PresentationBackdropFixture
    workflow: PresentationWorkflowFixture
    def __init__(self, id: _Optional[str] = ..., kind: _Optional[str] = ..., workspace: _Optional[_Union[PresentationWorkspaceFixture, _Mapping]] = ..., backdrop: _Optional[_Union[PresentationBackdropFixture, _Mapping]] = ..., workflow: _Optional[_Union[PresentationWorkflowFixture, _Mapping]] = ...) -> None: ...

class PresentationWorkspaceFixture(_message.Message):
    __slots__ = ("title", "group", "group_label", "groups", "sessions_label", "sessions", "role", "model", "reviewer", "reviewer_model", "branch", "prompt", "answer", "files", "file_label", "diff", "command", "checks", "ready", "composer", "return_label", "return_title", "review_message", "message_label", "reply_label", "status", "today", "keyboard")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    GROUP_FIELD_NUMBER: _ClassVar[int]
    GROUP_LABEL_FIELD_NUMBER: _ClassVar[int]
    GROUPS_FIELD_NUMBER: _ClassVar[int]
    SESSIONS_LABEL_FIELD_NUMBER: _ClassVar[int]
    SESSIONS_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    REVIEWER_FIELD_NUMBER: _ClassVar[int]
    REVIEWER_MODEL_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    ANSWER_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    FILE_LABEL_FIELD_NUMBER: _ClassVar[int]
    DIFF_FIELD_NUMBER: _ClassVar[int]
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    CHECKS_FIELD_NUMBER: _ClassVar[int]
    READY_FIELD_NUMBER: _ClassVar[int]
    COMPOSER_FIELD_NUMBER: _ClassVar[int]
    RETURN_LABEL_FIELD_NUMBER: _ClassVar[int]
    RETURN_TITLE_FIELD_NUMBER: _ClassVar[int]
    REVIEW_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_LABEL_FIELD_NUMBER: _ClassVar[int]
    REPLY_LABEL_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    TODAY_FIELD_NUMBER: _ClassVar[int]
    KEYBOARD_FIELD_NUMBER: _ClassVar[int]
    title: str
    group: str
    group_label: str
    groups: _containers.RepeatedScalarFieldContainer[str]
    sessions_label: str
    sessions: _containers.RepeatedScalarFieldContainer[str]
    role: str
    model: str
    reviewer: str
    reviewer_model: str
    branch: str
    prompt: str
    answer: str
    files: _containers.RepeatedScalarFieldContainer[str]
    file_label: str
    diff: _containers.RepeatedScalarFieldContainer[str]
    command: str
    checks: _containers.RepeatedScalarFieldContainer[str]
    ready: str
    composer: str
    return_label: str
    return_title: str
    review_message: str
    message_label: str
    reply_label: str
    status: str
    today: str
    keyboard: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, title: _Optional[str] = ..., group: _Optional[str] = ..., group_label: _Optional[str] = ..., groups: _Optional[_Iterable[str]] = ..., sessions_label: _Optional[str] = ..., sessions: _Optional[_Iterable[str]] = ..., role: _Optional[str] = ..., model: _Optional[str] = ..., reviewer: _Optional[str] = ..., reviewer_model: _Optional[str] = ..., branch: _Optional[str] = ..., prompt: _Optional[str] = ..., answer: _Optional[str] = ..., files: _Optional[_Iterable[str]] = ..., file_label: _Optional[str] = ..., diff: _Optional[_Iterable[str]] = ..., command: _Optional[str] = ..., checks: _Optional[_Iterable[str]] = ..., ready: _Optional[str] = ..., composer: _Optional[str] = ..., return_label: _Optional[str] = ..., return_title: _Optional[str] = ..., review_message: _Optional[str] = ..., message_label: _Optional[str] = ..., reply_label: _Optional[str] = ..., status: _Optional[str] = ..., today: _Optional[str] = ..., keyboard: _Optional[_Iterable[str]] = ...) -> None: ...

class PresentationBackdropFixture(_message.Message):
    __slots__ = ("title", "label", "selected", "styles", "surface", "palette", "panel", "caption", "export", "options", "badge", "asset_refs")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    SELECTED_FIELD_NUMBER: _ClassVar[int]
    STYLES_FIELD_NUMBER: _ClassVar[int]
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    PALETTE_FIELD_NUMBER: _ClassVar[int]
    PANEL_FIELD_NUMBER: _ClassVar[int]
    CAPTION_FIELD_NUMBER: _ClassVar[int]
    EXPORT_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    BADGE_FIELD_NUMBER: _ClassVar[int]
    ASSET_REFS_FIELD_NUMBER: _ClassVar[int]
    title: str
    label: str
    selected: str
    styles: _containers.RepeatedScalarFieldContainer[str]
    surface: str
    palette: str
    panel: str
    caption: str
    export: str
    options: _containers.RepeatedScalarFieldContainer[str]
    badge: str
    asset_refs: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, title: _Optional[str] = ..., label: _Optional[str] = ..., selected: _Optional[str] = ..., styles: _Optional[_Iterable[str]] = ..., surface: _Optional[str] = ..., palette: _Optional[str] = ..., panel: _Optional[str] = ..., caption: _Optional[str] = ..., export: _Optional[str] = ..., options: _Optional[_Iterable[str]] = ..., badge: _Optional[str] = ..., asset_refs: _Optional[_Iterable[str]] = ...) -> None: ...

class PresentationWorkflowStep(_message.Message):
    __slots__ = ("number", "title", "description")
    NUMBER_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    number: str
    title: str
    description: str
    def __init__(self, number: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class PresentationWorkflowFixture(_message.Message):
    __slots__ = ("title", "label", "steps", "browser_title", "browser_rows", "note")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    STEPS_FIELD_NUMBER: _ClassVar[int]
    BROWSER_TITLE_FIELD_NUMBER: _ClassVar[int]
    BROWSER_ROWS_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    title: str
    label: str
    steps: _containers.RepeatedCompositeFieldContainer[PresentationWorkflowStep]
    browser_title: str
    browser_rows: _containers.RepeatedScalarFieldContainer[str]
    note: str
    def __init__(self, title: _Optional[str] = ..., label: _Optional[str] = ..., steps: _Optional[_Iterable[_Union[PresentationWorkflowStep, _Mapping]]] = ..., browser_title: _Optional[str] = ..., browser_rows: _Optional[_Iterable[str]] = ..., note: _Optional[str] = ...) -> None: ...

class PresentationAsset(_message.Message):
    __slots__ = ("id", "release_ref", "content_hash", "width", "height", "mime", "surface", "responsive_alternatives", "focal_point", "crop_policy", "provenance", "overlay_regions", "public_url", "private_evidence_refs")
    ID_FIELD_NUMBER: _ClassVar[int]
    RELEASE_REF_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    MIME_FIELD_NUMBER: _ClassVar[int]
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    RESPONSIVE_ALTERNATIVES_FIELD_NUMBER: _ClassVar[int]
    FOCAL_POINT_FIELD_NUMBER: _ClassVar[int]
    CROP_POLICY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    OVERLAY_REGIONS_FIELD_NUMBER: _ClassVar[int]
    PUBLIC_URL_FIELD_NUMBER: _ClassVar[int]
    PRIVATE_EVIDENCE_REFS_FIELD_NUMBER: _ClassVar[int]
    id: str
    release_ref: str
    content_hash: str
    width: int
    height: int
    mime: str
    surface: str
    responsive_alternatives: _containers.RepeatedCompositeFieldContainer[PresentationAssetVariant]
    focal_point: PresentationFocalPoint
    crop_policy: str
    provenance: PresentationAssetProvenance
    overlay_regions: _containers.RepeatedCompositeFieldContainer[PresentationOverlayRegion]
    public_url: str
    private_evidence_refs: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., release_ref: _Optional[str] = ..., content_hash: _Optional[str] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., mime: _Optional[str] = ..., surface: _Optional[str] = ..., responsive_alternatives: _Optional[_Iterable[_Union[PresentationAssetVariant, _Mapping]]] = ..., focal_point: _Optional[_Union[PresentationFocalPoint, _Mapping]] = ..., crop_policy: _Optional[str] = ..., provenance: _Optional[_Union[PresentationAssetProvenance, _Mapping]] = ..., overlay_regions: _Optional[_Iterable[_Union[PresentationOverlayRegion, _Mapping]]] = ..., public_url: _Optional[str] = ..., private_evidence_refs: _Optional[_Iterable[str]] = ...) -> None: ...

class PresentationAssetVariant(_message.Message):
    __slots__ = ("surface", "asset_id")
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    ASSET_ID_FIELD_NUMBER: _ClassVar[int]
    surface: str
    asset_id: str
    def __init__(self, surface: _Optional[str] = ..., asset_id: _Optional[str] = ...) -> None: ...

class PresentationFocalPoint(_message.Message):
    __slots__ = ("x", "y")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    x: float
    y: float
    def __init__(self, x: _Optional[float] = ..., y: _Optional[float] = ...) -> None: ...

class PresentationAssetProvenance(_message.Message):
    __slots__ = ("provider", "job_ref", "candidate_ref")
    PROVIDER_FIELD_NUMBER: _ClassVar[int]
    JOB_REF_FIELD_NUMBER: _ClassVar[int]
    CANDIDATE_REF_FIELD_NUMBER: _ClassVar[int]
    provider: str
    job_ref: str
    candidate_ref: str
    def __init__(self, provider: _Optional[str] = ..., job_ref: _Optional[str] = ..., candidate_ref: _Optional[str] = ...) -> None: ...

class PresentationOverlayRegion(_message.Message):
    __slots__ = ("name", "x", "y", "width", "height", "measurement")
    NAME_FIELD_NUMBER: _ClassVar[int]
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    MEASUREMENT_FIELD_NUMBER: _ClassVar[int]
    name: str
    x: float
    y: float
    width: float
    height: float
    measurement: PresentationLegibilityMeasurement
    def __init__(self, name: _Optional[str] = ..., x: _Optional[float] = ..., y: _Optional[float] = ..., width: _Optional[float] = ..., height: _Optional[float] = ..., measurement: _Optional[_Union[PresentationLegibilityMeasurement, _Mapping]] = ...) -> None: ...

class PresentationLegibilityMeasurement(_message.Message):
    __slots__ = ("contrast_ratio", "minimum_contrast_ratio", "threshold", "verdict", "measurement_ref")
    CONTRAST_RATIO_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_CONTRAST_RATIO_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    MEASUREMENT_REF_FIELD_NUMBER: _ClassVar[int]
    contrast_ratio: float
    minimum_contrast_ratio: float
    threshold: float
    verdict: str
    measurement_ref: str
    def __init__(self, contrast_ratio: _Optional[float] = ..., minimum_contrast_ratio: _Optional[float] = ..., threshold: _Optional[float] = ..., verdict: _Optional[str] = ..., measurement_ref: _Optional[str] = ...) -> None: ...

class ResolvedProductPresentation(_message.Message):
    __slots__ = ("schema_version", "mode", "scope", "app_key", "page", "selected_app_keys", "spotlights", "capabilities", "assets", "fixtures", "diagnostics", "actions")
    SCHEMA_VERSION_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    SCOPE_FIELD_NUMBER: _ClassVar[int]
    APP_KEY_FIELD_NUMBER: _ClassVar[int]
    PAGE_FIELD_NUMBER: _ClassVar[int]
    SELECTED_APP_KEYS_FIELD_NUMBER: _ClassVar[int]
    SPOTLIGHTS_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    ASSETS_FIELD_NUMBER: _ClassVar[int]
    FIXTURES_FIELD_NUMBER: _ClassVar[int]
    DIAGNOSTICS_FIELD_NUMBER: _ClassVar[int]
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    schema_version: int
    mode: str
    scope: str
    app_key: str
    page: PresentationPage
    selected_app_keys: _containers.RepeatedScalarFieldContainer[str]
    spotlights: _containers.RepeatedCompositeFieldContainer[PresentationAppSpotlight]
    capabilities: _containers.RepeatedCompositeFieldContainer[ResolvedPresentationCapability]
    assets: _containers.RepeatedCompositeFieldContainer[ResolvedPresentationAsset]
    fixtures: _containers.RepeatedCompositeFieldContainer[PresentationFixture]
    diagnostics: PresentationDiagnostics
    actions: _containers.RepeatedCompositeFieldContainer[ResolvedPresentationAction]
    def __init__(self, schema_version: _Optional[int] = ..., mode: _Optional[str] = ..., scope: _Optional[str] = ..., app_key: _Optional[str] = ..., page: _Optional[_Union[PresentationPage, _Mapping]] = ..., selected_app_keys: _Optional[_Iterable[str]] = ..., spotlights: _Optional[_Iterable[_Union[PresentationAppSpotlight, _Mapping]]] = ..., capabilities: _Optional[_Iterable[_Union[ResolvedPresentationCapability, _Mapping]]] = ..., assets: _Optional[_Iterable[_Union[ResolvedPresentationAsset, _Mapping]]] = ..., fixtures: _Optional[_Iterable[_Union[PresentationFixture, _Mapping]]] = ..., diagnostics: _Optional[_Union[PresentationDiagnostics, _Mapping]] = ..., actions: _Optional[_Iterable[_Union[ResolvedPresentationAction, _Mapping]]] = ...) -> None: ...

class PresentationAppSpotlight(_message.Message):
    __slots__ = ("app_key", "slug", "name", "tagline", "description", "detail_route")
    APP_KEY_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    TAGLINE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    DETAIL_ROUTE_FIELD_NUMBER: _ClassVar[int]
    app_key: str
    slug: str
    name: str
    tagline: str
    description: str
    detail_route: str
    def __init__(self, app_key: _Optional[str] = ..., slug: _Optional[str] = ..., name: _Optional[str] = ..., tagline: _Optional[str] = ..., description: _Optional[str] = ..., detail_route: _Optional[str] = ...) -> None: ...

class ResolvedPresentationCapability(_message.Message):
    __slots__ = ("id", "label", "benefits", "status", "status_label", "constraints", "provider_requirements", "platform_requirements")
    ID_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    BENEFITS_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    STATUS_LABEL_FIELD_NUMBER: _ClassVar[int]
    CONSTRAINTS_FIELD_NUMBER: _ClassVar[int]
    PROVIDER_REQUIREMENTS_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_REQUIREMENTS_FIELD_NUMBER: _ClassVar[int]
    id: str
    label: str
    benefits: _containers.RepeatedScalarFieldContainer[str]
    status: str
    status_label: str
    constraints: _containers.RepeatedScalarFieldContainer[str]
    provider_requirements: _containers.RepeatedScalarFieldContainer[str]
    platform_requirements: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., label: _Optional[str] = ..., benefits: _Optional[_Iterable[str]] = ..., status: _Optional[str] = ..., status_label: _Optional[str] = ..., constraints: _Optional[_Iterable[str]] = ..., provider_requirements: _Optional[_Iterable[str]] = ..., platform_requirements: _Optional[_Iterable[str]] = ...) -> None: ...

class ResolvedPresentationAsset(_message.Message):
    __slots__ = ("id", "release_ref", "content_hash", "width", "height", "mime", "surface", "responsive_alternatives", "focal_point", "crop_policy", "provenance", "overlay_regions", "public_url")
    ID_FIELD_NUMBER: _ClassVar[int]
    RELEASE_REF_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    MIME_FIELD_NUMBER: _ClassVar[int]
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    RESPONSIVE_ALTERNATIVES_FIELD_NUMBER: _ClassVar[int]
    FOCAL_POINT_FIELD_NUMBER: _ClassVar[int]
    CROP_POLICY_FIELD_NUMBER: _ClassVar[int]
    PROVENANCE_FIELD_NUMBER: _ClassVar[int]
    OVERLAY_REGIONS_FIELD_NUMBER: _ClassVar[int]
    PUBLIC_URL_FIELD_NUMBER: _ClassVar[int]
    id: str
    release_ref: str
    content_hash: str
    width: int
    height: int
    mime: str
    surface: str
    responsive_alternatives: _containers.RepeatedCompositeFieldContainer[PresentationAssetVariant]
    focal_point: PresentationFocalPoint
    crop_policy: str
    provenance: PresentationAssetProvenance
    overlay_regions: _containers.RepeatedCompositeFieldContainer[ResolvedPresentationOverlayRegion]
    public_url: str
    def __init__(self, id: _Optional[str] = ..., release_ref: _Optional[str] = ..., content_hash: _Optional[str] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., mime: _Optional[str] = ..., surface: _Optional[str] = ..., responsive_alternatives: _Optional[_Iterable[_Union[PresentationAssetVariant, _Mapping]]] = ..., focal_point: _Optional[_Union[PresentationFocalPoint, _Mapping]] = ..., crop_policy: _Optional[str] = ..., provenance: _Optional[_Union[PresentationAssetProvenance, _Mapping]] = ..., overlay_regions: _Optional[_Iterable[_Union[ResolvedPresentationOverlayRegion, _Mapping]]] = ..., public_url: _Optional[str] = ...) -> None: ...

class ResolvedPresentationOverlayRegion(_message.Message):
    __slots__ = ("name", "x", "y", "width", "height", "measurement")
    NAME_FIELD_NUMBER: _ClassVar[int]
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    MEASUREMENT_FIELD_NUMBER: _ClassVar[int]
    name: str
    x: float
    y: float
    width: float
    height: float
    measurement: ResolvedPresentationLegibility
    def __init__(self, name: _Optional[str] = ..., x: _Optional[float] = ..., y: _Optional[float] = ..., width: _Optional[float] = ..., height: _Optional[float] = ..., measurement: _Optional[_Union[ResolvedPresentationLegibility, _Mapping]] = ...) -> None: ...

class ResolvedPresentationLegibility(_message.Message):
    __slots__ = ("contrast_ratio", "minimum_contrast_ratio", "threshold", "verdict")
    CONTRAST_RATIO_FIELD_NUMBER: _ClassVar[int]
    MINIMUM_CONTRAST_RATIO_FIELD_NUMBER: _ClassVar[int]
    THRESHOLD_FIELD_NUMBER: _ClassVar[int]
    VERDICT_FIELD_NUMBER: _ClassVar[int]
    contrast_ratio: float
    minimum_contrast_ratio: float
    threshold: float
    verdict: str
    def __init__(self, contrast_ratio: _Optional[float] = ..., minimum_contrast_ratio: _Optional[float] = ..., threshold: _Optional[float] = ..., verdict: _Optional[str] = ...) -> None: ...

class PresentationDiagnostics(_message.Message):
    __slots__ = ("requested_route", "resolved_route", "requested_variant", "resolved_variant", "requested_revision", "resolved_revision", "locale", "bundle_key", "app_key", "mode", "fallback", "fallback_reason", "preview", "noindex", "no_store", "eligible_app_keys", "block_digest", "asset_release_refs", "commerce_snapshot_ref")
    REQUESTED_ROUTE_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_ROUTE_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_VARIANT_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_VARIANT_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_REVISION_FIELD_NUMBER: _ClassVar[int]
    LOCALE_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_KEY_FIELD_NUMBER: _ClassVar[int]
    APP_KEY_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_FIELD_NUMBER: _ClassVar[int]
    FALLBACK_REASON_FIELD_NUMBER: _ClassVar[int]
    PREVIEW_FIELD_NUMBER: _ClassVar[int]
    NOINDEX_FIELD_NUMBER: _ClassVar[int]
    NO_STORE_FIELD_NUMBER: _ClassVar[int]
    ELIGIBLE_APP_KEYS_FIELD_NUMBER: _ClassVar[int]
    BLOCK_DIGEST_FIELD_NUMBER: _ClassVar[int]
    ASSET_RELEASE_REFS_FIELD_NUMBER: _ClassVar[int]
    COMMERCE_SNAPSHOT_REF_FIELD_NUMBER: _ClassVar[int]
    requested_route: str
    resolved_route: str
    requested_variant: str
    resolved_variant: str
    requested_revision: str
    resolved_revision: str
    locale: str
    bundle_key: str
    app_key: str
    mode: str
    fallback: bool
    fallback_reason: str
    preview: bool
    noindex: bool
    no_store: bool
    eligible_app_keys: _containers.RepeatedScalarFieldContainer[str]
    block_digest: str
    asset_release_refs: _containers.RepeatedScalarFieldContainer[str]
    commerce_snapshot_ref: str
    def __init__(self, requested_route: _Optional[str] = ..., resolved_route: _Optional[str] = ..., requested_variant: _Optional[str] = ..., resolved_variant: _Optional[str] = ..., requested_revision: _Optional[str] = ..., resolved_revision: _Optional[str] = ..., locale: _Optional[str] = ..., bundle_key: _Optional[str] = ..., app_key: _Optional[str] = ..., mode: _Optional[str] = ..., fallback: _Optional[bool] = ..., fallback_reason: _Optional[str] = ..., preview: _Optional[bool] = ..., noindex: _Optional[bool] = ..., no_store: _Optional[bool] = ..., eligible_app_keys: _Optional[_Iterable[str]] = ..., block_digest: _Optional[str] = ..., asset_release_refs: _Optional[_Iterable[str]] = ..., commerce_snapshot_ref: _Optional[str] = ...) -> None: ...

class ResolvedPresentationAction(_message.Message):
    __slots__ = ("key", "status", "reason", "href", "app_key", "plan_ref")
    KEY_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    HREF_FIELD_NUMBER: _ClassVar[int]
    APP_KEY_FIELD_NUMBER: _ClassVar[int]
    PLAN_REF_FIELD_NUMBER: _ClassVar[int]
    key: str
    status: str
    reason: str
    href: str
    app_key: str
    plan_ref: str
    def __init__(self, key: _Optional[str] = ..., status: _Optional[str] = ..., reason: _Optional[str] = ..., href: _Optional[str] = ..., app_key: _Optional[str] = ..., plan_ref: _Optional[str] = ...) -> None: ...
