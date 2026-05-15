import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ComponentVersionIntent(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COMPONENT_VERSION_INTENT_UNSPECIFIED: _ClassVar[ComponentVersionIntent]
    COMPONENT_VERSION_INTENT_DRAFT: _ClassVar[ComponentVersionIntent]
    COMPONENT_VERSION_INTENT_RELEASE: _ClassVar[ComponentVersionIntent]

class ComponentVersionStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COMPONENT_VERSION_STATUS_UNSPECIFIED: _ClassVar[ComponentVersionStatus]
    COMPONENT_VERSION_STATUS_DRAFT: _ClassVar[ComponentVersionStatus]
    COMPONENT_VERSION_STATUS_RELEASED: _ClassVar[ComponentVersionStatus]
    COMPONENT_VERSION_STATUS_DEPRECATED: _ClassVar[ComponentVersionStatus]
    COMPONENT_VERSION_STATUS_ARCHIVED: _ClassVar[ComponentVersionStatus]
COMPONENT_VERSION_INTENT_UNSPECIFIED: ComponentVersionIntent
COMPONENT_VERSION_INTENT_DRAFT: ComponentVersionIntent
COMPONENT_VERSION_INTENT_RELEASE: ComponentVersionIntent
COMPONENT_VERSION_STATUS_UNSPECIFIED: ComponentVersionStatus
COMPONENT_VERSION_STATUS_DRAFT: ComponentVersionStatus
COMPONENT_VERSION_STATUS_RELEASED: ComponentVersionStatus
COMPONENT_VERSION_STATUS_DEPRECATED: ComponentVersionStatus
COMPONENT_VERSION_STATUS_ARCHIVED: ComponentVersionStatus

class Component(_message.Message):
    __slots__ = ("id", "library_id", "display_name", "description", "source_path", "version", "tags", "indexed_at", "updated_at", "headers", "slug", "manifest_path", "draft_version", "latest_version")
    class HeadersEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    INDEXED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    HEADERS_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_PATH_FIELD_NUMBER: _ClassVar[int]
    DRAFT_VERSION_FIELD_NUMBER: _ClassVar[int]
    LATEST_VERSION_FIELD_NUMBER: _ClassVar[int]
    id: str
    library_id: str
    display_name: str
    description: str
    source_path: str
    version: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    indexed_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    headers: _containers.ScalarMap[str, str]
    slug: str
    manifest_path: str
    draft_version: str
    latest_version: str
    def __init__(self, id: _Optional[str] = ..., library_id: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., source_path: _Optional[str] = ..., version: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., indexed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., headers: _Optional[_Mapping[str, str]] = ..., slug: _Optional[str] = ..., manifest_path: _Optional[str] = ..., draft_version: _Optional[str] = ..., latest_version: _Optional[str] = ...) -> None: ...

class ListComponentsRequest(_message.Message):
    __slots__ = ("match", "tag", "limit", "tags", "category")
    MATCH_FIELD_NUMBER: _ClassVar[int]
    TAG_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    match: str
    tag: str
    limit: int
    tags: _containers.RepeatedScalarFieldContainer[str]
    category: str
    def __init__(self, match: _Optional[str] = ..., tag: _Optional[str] = ..., limit: _Optional[int] = ..., tags: _Optional[_Iterable[str]] = ..., category: _Optional[str] = ...) -> None: ...

class ListComponentsResponse(_message.Message):
    __slots__ = ("components",)
    COMPONENTS_FIELD_NUMBER: _ClassVar[int]
    components: _containers.RepeatedCompositeFieldContainer[Component]
    def __init__(self, components: _Optional[_Iterable[_Union[Component, _Mapping]]] = ...) -> None: ...

class GetComponentRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetComponentResponse(_message.Message):
    __slots__ = ("component",)
    COMPONENT_FIELD_NUMBER: _ClassVar[int]
    component: Component
    def __init__(self, component: _Optional[_Union[Component, _Mapping]] = ...) -> None: ...

class GetComponentByLibraryIdRequest(_message.Message):
    __slots__ = ("library_id",)
    LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    library_id: str
    def __init__(self, library_id: _Optional[str] = ...) -> None: ...

class GetComponentByLibraryIdResponse(_message.Message):
    __slots__ = ("component",)
    COMPONENT_FIELD_NUMBER: _ClassVar[int]
    component: Component
    def __init__(self, component: _Optional[_Union[Component, _Mapping]] = ...) -> None: ...

class IndexComponentsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class IndexComponentsResponse(_message.Message):
    __slots__ = ("scanned", "indexed", "skipped", "deleted", "errors", "library_ids")
    SCANNED_FIELD_NUMBER: _ClassVar[int]
    INDEXED_FIELD_NUMBER: _ClassVar[int]
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    DELETED_FIELD_NUMBER: _ClassVar[int]
    ERRORS_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_IDS_FIELD_NUMBER: _ClassVar[int]
    scanned: int
    indexed: int
    skipped: int
    deleted: int
    errors: _containers.RepeatedScalarFieldContainer[str]
    library_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, scanned: _Optional[int] = ..., indexed: _Optional[int] = ..., skipped: _Optional[int] = ..., deleted: _Optional[int] = ..., errors: _Optional[_Iterable[str]] = ..., library_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class InitializeComponentRequest(_message.Message):
    __slots__ = ("library_id", "slug", "display_name", "description", "tags", "initial_version", "file_name", "initial_source")
    LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    INITIAL_VERSION_FIELD_NUMBER: _ClassVar[int]
    FILE_NAME_FIELD_NUMBER: _ClassVar[int]
    INITIAL_SOURCE_FIELD_NUMBER: _ClassVar[int]
    library_id: str
    slug: str
    display_name: str
    description: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    initial_version: str
    file_name: str
    initial_source: str
    def __init__(self, library_id: _Optional[str] = ..., slug: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., initial_version: _Optional[str] = ..., file_name: _Optional[str] = ..., initial_source: _Optional[str] = ...) -> None: ...

class InitializeComponentResponse(_message.Message):
    __slots__ = ("component", "manifest_path", "source_path")
    COMPONENT_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_PATH_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    component: Component
    manifest_path: str
    source_path: str
    def __init__(self, component: _Optional[_Union[Component, _Mapping]] = ..., manifest_path: _Optional[str] = ..., source_path: _Optional[str] = ...) -> None: ...

class CreateComponentVersionRequest(_message.Message):
    __slots__ = ("component_id", "version", "from_version", "intent", "file_name", "source", "changelog_md")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    FROM_VERSION_FIELD_NUMBER: _ClassVar[int]
    INTENT_FIELD_NUMBER: _ClassVar[int]
    FILE_NAME_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CHANGELOG_MD_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    version: str
    from_version: str
    intent: ComponentVersionIntent
    file_name: str
    source: str
    changelog_md: str
    def __init__(self, component_id: _Optional[str] = ..., version: _Optional[str] = ..., from_version: _Optional[str] = ..., intent: _Optional[_Union[ComponentVersionIntent, str]] = ..., file_name: _Optional[str] = ..., source: _Optional[str] = ..., changelog_md: _Optional[str] = ...) -> None: ...

class CreateComponentVersionResponse(_message.Message):
    __slots__ = ("component", "version", "source_path")
    COMPONENT_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    component: Component
    version: ComponentVersion
    source_path: str
    def __init__(self, component: _Optional[_Union[Component, _Mapping]] = ..., version: _Optional[_Union[ComponentVersion, _Mapping]] = ..., source_path: _Optional[str] = ...) -> None: ...

class UpdateComponentManifestRequest(_message.Message):
    __slots__ = ("component_id", "display_name", "description", "tags", "latest_version", "draft_version", "deprecated_versions")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    LATEST_VERSION_FIELD_NUMBER: _ClassVar[int]
    DRAFT_VERSION_FIELD_NUMBER: _ClassVar[int]
    DEPRECATED_VERSIONS_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    display_name: str
    description: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    latest_version: str
    draft_version: str
    deprecated_versions: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, component_id: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., latest_version: _Optional[str] = ..., draft_version: _Optional[str] = ..., deprecated_versions: _Optional[_Iterable[str]] = ...) -> None: ...

class UpdateComponentManifestResponse(_message.Message):
    __slots__ = ("component",)
    COMPONENT_FIELD_NUMBER: _ClassVar[int]
    component: Component
    def __init__(self, component: _Optional[_Union[Component, _Mapping]] = ...) -> None: ...

class GetComponentContentRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetComponentContentResponse(_message.Message):
    __slots__ = ("content", "source_path", "sha256")
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    content: str
    source_path: str
    sha256: str
    def __init__(self, content: _Optional[str] = ..., source_path: _Optional[str] = ..., sha256: _Optional[str] = ...) -> None: ...

class UpdateComponentContentRequest(_message.Message):
    __slots__ = ("id", "content", "expected_sha256")
    ID_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_SHA256_FIELD_NUMBER: _ClassVar[int]
    id: str
    content: str
    expected_sha256: str
    def __init__(self, id: _Optional[str] = ..., content: _Optional[str] = ..., expected_sha256: _Optional[str] = ...) -> None: ...

class UpdateComponentContentResponse(_message.Message):
    __slots__ = ("sha256", "source_path")
    SHA256_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    sha256: str
    source_path: str
    def __init__(self, sha256: _Optional[str] = ..., source_path: _Optional[str] = ...) -> None: ...

class ComponentVersion(_message.Message):
    __slots__ = ("id", "component_id", "library_id", "version", "status", "source_path", "content_sha256", "changelog_md", "indexed_at", "released_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_SHA256_FIELD_NUMBER: _ClassVar[int]
    CHANGELOG_MD_FIELD_NUMBER: _ClassVar[int]
    INDEXED_AT_FIELD_NUMBER: _ClassVar[int]
    RELEASED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    component_id: str
    library_id: str
    version: str
    status: ComponentVersionStatus
    source_path: str
    content_sha256: str
    changelog_md: str
    indexed_at: _timestamp_pb2.Timestamp
    released_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., component_id: _Optional[str] = ..., library_id: _Optional[str] = ..., version: _Optional[str] = ..., status: _Optional[_Union[ComponentVersionStatus, str]] = ..., source_path: _Optional[str] = ..., content_sha256: _Optional[str] = ..., changelog_md: _Optional[str] = ..., indexed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., released_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListComponentVersionsRequest(_message.Message):
    __slots__ = ("component_id", "limit")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    limit: int
    def __init__(self, component_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListComponentVersionsResponse(_message.Message):
    __slots__ = ("versions",)
    VERSIONS_FIELD_NUMBER: _ClassVar[int]
    versions: _containers.RepeatedCompositeFieldContainer[ComponentVersion]
    def __init__(self, versions: _Optional[_Iterable[_Union[ComponentVersion, _Mapping]]] = ...) -> None: ...

class GetComponentVersionContentRequest(_message.Message):
    __slots__ = ("component_id", "version")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    version: str
    def __init__(self, component_id: _Optional[str] = ..., version: _Optional[str] = ...) -> None: ...

class GetComponentVersionContentResponse(_message.Message):
    __slots__ = ("version", "content")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    version: ComponentVersion
    content: str
    def __init__(self, version: _Optional[_Union[ComponentVersion, _Mapping]] = ..., content: _Optional[str] = ...) -> None: ...
