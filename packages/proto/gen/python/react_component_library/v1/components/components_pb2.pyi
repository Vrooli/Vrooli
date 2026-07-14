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

class DesignAffinity(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DESIGN_AFFINITY_UNSPECIFIED: _ClassVar[DesignAffinity]
    DESIGN_AFFINITY_NATIVE: _ClassVar[DesignAffinity]
    DESIGN_AFFINITY_COMPATIBLE: _ClassVar[DesignAffinity]
    DESIGN_AFFINITY_DISCOURAGED: _ClassVar[DesignAffinity]

class StyleFitVerdictKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    STYLE_FIT_VERDICT_KIND_UNSPECIFIED: _ClassVar[StyleFitVerdictKind]
    STYLE_FIT_VERDICT_KIND_OK: _ClassVar[StyleFitVerdictKind]
    STYLE_FIT_VERDICT_KIND_INFO: _ClassVar[StyleFitVerdictKind]
    STYLE_FIT_VERDICT_KIND_WARN: _ClassVar[StyleFitVerdictKind]
COMPONENT_VERSION_INTENT_UNSPECIFIED: ComponentVersionIntent
COMPONENT_VERSION_INTENT_DRAFT: ComponentVersionIntent
COMPONENT_VERSION_INTENT_RELEASE: ComponentVersionIntent
COMPONENT_VERSION_STATUS_UNSPECIFIED: ComponentVersionStatus
COMPONENT_VERSION_STATUS_DRAFT: ComponentVersionStatus
COMPONENT_VERSION_STATUS_RELEASED: ComponentVersionStatus
COMPONENT_VERSION_STATUS_DEPRECATED: ComponentVersionStatus
COMPONENT_VERSION_STATUS_ARCHIVED: ComponentVersionStatus
DESIGN_AFFINITY_UNSPECIFIED: DesignAffinity
DESIGN_AFFINITY_NATIVE: DesignAffinity
DESIGN_AFFINITY_COMPATIBLE: DesignAffinity
DESIGN_AFFINITY_DISCOURAGED: DesignAffinity
STYLE_FIT_VERDICT_KIND_UNSPECIFIED: StyleFitVerdictKind
STYLE_FIT_VERDICT_KIND_OK: StyleFitVerdictKind
STYLE_FIT_VERDICT_KIND_INFO: StyleFitVerdictKind
STYLE_FIT_VERDICT_KIND_WARN: StyleFitVerdictKind

class Component(_message.Message):
    __slots__ = ("id", "library_id", "display_name", "description", "source_path", "version", "tags", "indexed_at", "updated_at", "headers", "slug", "manifest_path", "draft_version", "latest_version", "slot", "design_styles", "category")
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
    SLOT_FIELD_NUMBER: _ClassVar[int]
    DESIGN_STYLES_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
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
    slot: str
    design_styles: _containers.RepeatedCompositeFieldContainer[ComponentDesignAffinity]
    category: str
    def __init__(self, id: _Optional[str] = ..., library_id: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., source_path: _Optional[str] = ..., version: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., indexed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., headers: _Optional[_Mapping[str, str]] = ..., slug: _Optional[str] = ..., manifest_path: _Optional[str] = ..., draft_version: _Optional[str] = ..., latest_version: _Optional[str] = ..., slot: _Optional[str] = ..., design_styles: _Optional[_Iterable[_Union[ComponentDesignAffinity, _Mapping]]] = ..., category: _Optional[str] = ...) -> None: ...

class ListComponentsRequest(_message.Message):
    __slots__ = ("match", "tag", "limit", "tags", "category", "style_id", "affinity")
    MATCH_FIELD_NUMBER: _ClassVar[int]
    TAG_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    STYLE_ID_FIELD_NUMBER: _ClassVar[int]
    AFFINITY_FIELD_NUMBER: _ClassVar[int]
    match: str
    tag: str
    limit: int
    tags: _containers.RepeatedScalarFieldContainer[str]
    category: str
    style_id: str
    affinity: str
    def __init__(self, match: _Optional[str] = ..., tag: _Optional[str] = ..., limit: _Optional[int] = ..., tags: _Optional[_Iterable[str]] = ..., category: _Optional[str] = ..., style_id: _Optional[str] = ..., affinity: _Optional[str] = ...) -> None: ...

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

class IngestComponentRequest(_message.Message):
    __slots__ = ("scenario", "source_file", "slug", "display_name", "description", "tags", "slot", "source_files", "version")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FILE_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    SLOT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FILES_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    source_file: str
    slug: str
    display_name: str
    description: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    slot: str
    source_files: _containers.RepeatedScalarFieldContainer[str]
    version: str
    def __init__(self, scenario: _Optional[str] = ..., source_file: _Optional[str] = ..., slug: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., slot: _Optional[str] = ..., source_files: _Optional[_Iterable[str]] = ..., version: _Optional[str] = ...) -> None: ...

class IngestFinding(_message.Message):
    __slots__ = ("code", "message", "source_file")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FILE_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    source_file: str
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ..., source_file: _Optional[str] = ...) -> None: ...

class IngestComponentResponse(_message.Message):
    __slots__ = ("component", "manifest_path", "source_path", "draft_version", "findings", "checklist_path", "parity_report")
    COMPONENT_FIELD_NUMBER: _ClassVar[int]
    MANIFEST_PATH_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    DRAFT_VERSION_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    CHECKLIST_PATH_FIELD_NUMBER: _ClassVar[int]
    PARITY_REPORT_FIELD_NUMBER: _ClassVar[int]
    component: Component
    manifest_path: str
    source_path: str
    draft_version: str
    findings: _containers.RepeatedCompositeFieldContainer[IngestFinding]
    checklist_path: str
    parity_report: IngestParityReport
    def __init__(self, component: _Optional[_Union[Component, _Mapping]] = ..., manifest_path: _Optional[str] = ..., source_path: _Optional[str] = ..., draft_version: _Optional[str] = ..., findings: _Optional[_Iterable[_Union[IngestFinding, _Mapping]]] = ..., checklist_path: _Optional[str] = ..., parity_report: _Optional[_Union[IngestParityReport, _Mapping]] = ...) -> None: ...

class IngestParityReport(_message.Message):
    __slots__ = ("origin_files", "harvested_files", "findings", "acknowledged")
    ORIGIN_FILES_FIELD_NUMBER: _ClassVar[int]
    HARVESTED_FILES_FIELD_NUMBER: _ClassVar[int]
    FINDINGS_FIELD_NUMBER: _ClassVar[int]
    ACKNOWLEDGED_FIELD_NUMBER: _ClassVar[int]
    origin_files: _containers.RepeatedScalarFieldContainer[str]
    harvested_files: _containers.RepeatedScalarFieldContainer[str]
    findings: _containers.RepeatedCompositeFieldContainer[IngestFinding]
    acknowledged: bool
    def __init__(self, origin_files: _Optional[_Iterable[str]] = ..., harvested_files: _Optional[_Iterable[str]] = ..., findings: _Optional[_Iterable[_Union[IngestFinding, _Mapping]]] = ..., acknowledged: _Optional[bool] = ...) -> None: ...

class CreateComponentVersionRequest(_message.Message):
    __slots__ = ("component_id", "version", "from_version", "intent", "file_name", "source", "changelog_md", "acknowledge_parity_waiver")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    FROM_VERSION_FIELD_NUMBER: _ClassVar[int]
    INTENT_FIELD_NUMBER: _ClassVar[int]
    FILE_NAME_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    CHANGELOG_MD_FIELD_NUMBER: _ClassVar[int]
    ACKNOWLEDGE_PARITY_WAIVER_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    version: str
    from_version: str
    intent: ComponentVersionIntent
    file_name: str
    source: str
    changelog_md: str
    acknowledge_parity_waiver: bool
    def __init__(self, component_id: _Optional[str] = ..., version: _Optional[str] = ..., from_version: _Optional[str] = ..., intent: _Optional[_Union[ComponentVersionIntent, str]] = ..., file_name: _Optional[str] = ..., source: _Optional[str] = ..., changelog_md: _Optional[str] = ..., acknowledge_parity_waiver: _Optional[bool] = ...) -> None: ...

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
    __slots__ = ("id", "path")
    ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    id: str
    path: str
    def __init__(self, id: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("id", "content", "expected_sha256", "path")
    ID_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_SHA256_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    id: str
    content: str
    expected_sha256: str
    path: str
    def __init__(self, id: _Optional[str] = ..., content: _Optional[str] = ..., expected_sha256: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class UpdateComponentContentResponse(_message.Message):
    __slots__ = ("sha256", "source_path")
    SHA256_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    sha256: str
    source_path: str
    def __init__(self, sha256: _Optional[str] = ..., source_path: _Optional[str] = ...) -> None: ...

class ComponentVersion(_message.Message):
    __slots__ = ("id", "component_id", "library_id", "version", "status", "source_path", "content_sha256", "changelog_md", "indexed_at", "released_at", "files", "parity_report")
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
    FILES_FIELD_NUMBER: _ClassVar[int]
    PARITY_REPORT_FIELD_NUMBER: _ClassVar[int]
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
    files: _containers.RepeatedCompositeFieldContainer[ComponentVersionFile]
    parity_report: IngestParityReport
    def __init__(self, id: _Optional[str] = ..., component_id: _Optional[str] = ..., library_id: _Optional[str] = ..., version: _Optional[str] = ..., status: _Optional[_Union[ComponentVersionStatus, str]] = ..., source_path: _Optional[str] = ..., content_sha256: _Optional[str] = ..., changelog_md: _Optional[str] = ..., indexed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., released_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., files: _Optional[_Iterable[_Union[ComponentVersionFile, _Mapping]]] = ..., parity_report: _Optional[_Union[IngestParityReport, _Mapping]] = ...) -> None: ...

class ComponentVersionFile(_message.Message):
    __slots__ = ("path", "content_sha256", "is_entry")
    PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_SHA256_FIELD_NUMBER: _ClassVar[int]
    IS_ENTRY_FIELD_NUMBER: _ClassVar[int]
    path: str
    content_sha256: str
    is_entry: bool
    def __init__(self, path: _Optional[str] = ..., content_sha256: _Optional[str] = ..., is_entry: _Optional[bool] = ...) -> None: ...

class ComponentDesignAffinity(_message.Message):
    __slots__ = ("style_id", "affinity", "reason")
    STYLE_ID_FIELD_NUMBER: _ClassVar[int]
    AFFINITY_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    style_id: str
    affinity: DesignAffinity
    reason: str
    def __init__(self, style_id: _Optional[str] = ..., affinity: _Optional[_Union[DesignAffinity, str]] = ..., reason: _Optional[str] = ...) -> None: ...

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
    __slots__ = ("component_id", "version", "path")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    version: str
    path: str
    def __init__(self, component_id: _Optional[str] = ..., version: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class GetComponentVersionContentResponse(_message.Message):
    __slots__ = ("version", "content")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    version: ComponentVersion
    content: str
    def __init__(self, version: _Optional[_Union[ComponentVersion, _Mapping]] = ..., content: _Optional[str] = ...) -> None: ...

class ComponentExample(_message.Message):
    __slots__ = ("id", "component_id", "library_id", "version", "name", "display_name", "props_json", "setup_json", "expect_json", "source_path", "indexed_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    LIBRARY_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    PROPS_JSON_FIELD_NUMBER: _ClassVar[int]
    SETUP_JSON_FIELD_NUMBER: _ClassVar[int]
    EXPECT_JSON_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    INDEXED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    component_id: str
    library_id: str
    version: str
    name: str
    display_name: str
    props_json: str
    setup_json: str
    expect_json: str
    source_path: str
    indexed_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., component_id: _Optional[str] = ..., library_id: _Optional[str] = ..., version: _Optional[str] = ..., name: _Optional[str] = ..., display_name: _Optional[str] = ..., props_json: _Optional[str] = ..., setup_json: _Optional[str] = ..., expect_json: _Optional[str] = ..., source_path: _Optional[str] = ..., indexed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListComponentExamplesRequest(_message.Message):
    __slots__ = ("component_id", "version", "limit")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    version: str
    limit: int
    def __init__(self, component_id: _Optional[str] = ..., version: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListComponentExamplesResponse(_message.Message):
    __slots__ = ("examples",)
    EXAMPLES_FIELD_NUMBER: _ClassVar[int]
    examples: _containers.RepeatedCompositeFieldContainer[ComponentExample]
    def __init__(self, examples: _Optional[_Iterable[_Union[ComponentExample, _Mapping]]] = ...) -> None: ...

class DesignStyle(_message.Message):
    __slots__ = ("id", "name", "tags", "supports")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    supports: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., supports: _Optional[_Iterable[str]] = ...) -> None: ...

class ListDesignStylesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListDesignStylesResponse(_message.Message):
    __slots__ = ("styles",)
    STYLES_FIELD_NUMBER: _ClassVar[int]
    styles: _containers.RepeatedCompositeFieldContainer[DesignStyle]
    def __init__(self, styles: _Optional[_Iterable[_Union[DesignStyle, _Mapping]]] = ...) -> None: ...

class ValidateStyleFitRequest(_message.Message):
    __slots__ = ("component_id", "scenario", "version")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    scenario: str
    version: str
    def __init__(self, component_id: _Optional[str] = ..., scenario: _Optional[str] = ..., version: _Optional[str] = ...) -> None: ...

class ValidateStyleFitResponse(_message.Message):
    __slots__ = ("kind", "component_id", "version", "scenario", "scenario_style", "affinity", "detail")
    KIND_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_STYLE_FIELD_NUMBER: _ClassVar[int]
    AFFINITY_FIELD_NUMBER: _ClassVar[int]
    DETAIL_FIELD_NUMBER: _ClassVar[int]
    kind: StyleFitVerdictKind
    component_id: str
    version: str
    scenario: str
    scenario_style: str
    affinity: DesignAffinity
    detail: str
    def __init__(self, kind: _Optional[_Union[StyleFitVerdictKind, str]] = ..., component_id: _Optional[str] = ..., version: _Optional[str] = ..., scenario: _Optional[str] = ..., scenario_style: _Optional[str] = ..., affinity: _Optional[_Union[DesignAffinity, str]] = ..., detail: _Optional[str] = ...) -> None: ...
