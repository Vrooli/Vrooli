from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DownloadStorefront(_message.Message):
    __slots__ = ("store", "label", "url", "badge")
    STORE_FIELD_NUMBER: _ClassVar[int]
    LABEL_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    BADGE_FIELD_NUMBER: _ClassVar[int]
    store: str
    label: str
    url: str
    badge: str
    def __init__(self, store: _Optional[str] = ..., label: _Optional[str] = ..., url: _Optional[str] = ..., badge: _Optional[str] = ...) -> None: ...

class DownloadAsset(_message.Message):
    __slots__ = ("id", "bundle_key", "app_key", "platform", "artifact_url", "release_version", "release_notes", "checksum", "requires_entitlement", "metadata")
    ID_FIELD_NUMBER: _ClassVar[int]
    BUNDLE_KEY_FIELD_NUMBER: _ClassVar[int]
    APP_KEY_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    ARTIFACT_URL_FIELD_NUMBER: _ClassVar[int]
    RELEASE_VERSION_FIELD_NUMBER: _ClassVar[int]
    RELEASE_NOTES_FIELD_NUMBER: _ClassVar[int]
    CHECKSUM_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_ENTITLEMENT_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    id: int
    bundle_key: str
    app_key: str
    platform: str
    artifact_url: str
    release_version: str
    release_notes: str
    checksum: str
    requires_entitlement: bool
    metadata: _struct_pb2.Struct
    def __init__(self, id: _Optional[int] = ..., bundle_key: _Optional[str] = ..., app_key: _Optional[str] = ..., platform: _Optional[str] = ..., artifact_url: _Optional[str] = ..., release_version: _Optional[str] = ..., release_notes: _Optional[str] = ..., checksum: _Optional[str] = ..., requires_entitlement: _Optional[bool] = ..., metadata: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class DownloadApp(_message.Message):
    __slots__ = ("bundle_key", "app_key", "name", "tagline", "description", "install_overview", "install_steps", "storefronts", "metadata", "display_order", "platforms")
    BUNDLE_KEY_FIELD_NUMBER: _ClassVar[int]
    APP_KEY_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    TAGLINE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    INSTALL_OVERVIEW_FIELD_NUMBER: _ClassVar[int]
    INSTALL_STEPS_FIELD_NUMBER: _ClassVar[int]
    STOREFRONTS_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_ORDER_FIELD_NUMBER: _ClassVar[int]
    PLATFORMS_FIELD_NUMBER: _ClassVar[int]
    bundle_key: str
    app_key: str
    name: str
    tagline: str
    description: str
    install_overview: str
    install_steps: _containers.RepeatedScalarFieldContainer[str]
    storefronts: _containers.RepeatedCompositeFieldContainer[DownloadStorefront]
    metadata: _struct_pb2.Struct
    display_order: int
    platforms: _containers.RepeatedCompositeFieldContainer[DownloadAsset]
    def __init__(self, bundle_key: _Optional[str] = ..., app_key: _Optional[str] = ..., name: _Optional[str] = ..., tagline: _Optional[str] = ..., description: _Optional[str] = ..., install_overview: _Optional[str] = ..., install_steps: _Optional[_Iterable[str]] = ..., storefronts: _Optional[_Iterable[_Union[DownloadStorefront, _Mapping]]] = ..., metadata: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., display_order: _Optional[int] = ..., platforms: _Optional[_Iterable[_Union[DownloadAsset, _Mapping]]] = ...) -> None: ...
