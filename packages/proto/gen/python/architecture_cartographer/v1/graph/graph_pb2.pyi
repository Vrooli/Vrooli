import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Language(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    LANGUAGE_UNSPECIFIED: _ClassVar[Language]
    LANGUAGE_GO: _ClassVar[Language]
    LANGUAGE_TYPESCRIPT: _ClassVar[Language]
LANGUAGE_UNSPECIFIED: Language
LANGUAGE_GO: Language
LANGUAGE_TYPESCRIPT: Language

class FileNode(_message.Message):
    __slots__ = ("id", "path", "package_id", "language", "lines", "is_test")
    ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_ID_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    LINES_FIELD_NUMBER: _ClassVar[int]
    IS_TEST_FIELD_NUMBER: _ClassVar[int]
    id: str
    path: str
    package_id: str
    language: Language
    lines: int
    is_test: bool
    def __init__(self, id: _Optional[str] = ..., path: _Optional[str] = ..., package_id: _Optional[str] = ..., language: _Optional[_Union[Language, str]] = ..., lines: _Optional[int] = ..., is_test: _Optional[bool] = ...) -> None: ...

class PackageNode(_message.Message):
    __slots__ = ("id", "import_path", "directory", "language", "internal")
    ID_FIELD_NUMBER: _ClassVar[int]
    IMPORT_PATH_FIELD_NUMBER: _ClassVar[int]
    DIRECTORY_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_FIELD_NUMBER: _ClassVar[int]
    INTERNAL_FIELD_NUMBER: _ClassVar[int]
    id: str
    import_path: str
    directory: str
    language: Language
    internal: bool
    def __init__(self, id: _Optional[str] = ..., import_path: _Optional[str] = ..., directory: _Optional[str] = ..., language: _Optional[_Union[Language, str]] = ..., internal: _Optional[bool] = ...) -> None: ...

class SymbolNode(_message.Message):
    __slots__ = ("id", "name", "package_id", "file_id", "kind", "exported")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    PACKAGE_ID_FIELD_NUMBER: _ClassVar[int]
    FILE_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    EXPORTED_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    package_id: str
    file_id: str
    kind: str
    exported: bool
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., package_id: _Optional[str] = ..., file_id: _Optional[str] = ..., kind: _Optional[str] = ..., exported: _Optional[bool] = ...) -> None: ...

class ImportEdge(_message.Message):
    __slots__ = ("to_package_id", "symbol_ids", "test_only")
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_PACKAGE_ID_FIELD_NUMBER: _ClassVar[int]
    SYMBOL_IDS_FIELD_NUMBER: _ClassVar[int]
    TEST_ONLY_FIELD_NUMBER: _ClassVar[int]
    to_package_id: str
    symbol_ids: _containers.RepeatedScalarFieldContainer[str]
    test_only: bool
    def __init__(self, to_package_id: _Optional[str] = ..., symbol_ids: _Optional[_Iterable[str]] = ..., test_only: _Optional[bool] = ..., **kwargs) -> None: ...

class Chunk(_message.Message):
    __slots__ = ("id", "file_id", "path", "current_domain")
    ID_FIELD_NUMBER: _ClassVar[int]
    FILE_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    CURRENT_DOMAIN_FIELD_NUMBER: _ClassVar[int]
    id: str
    file_id: str
    path: str
    current_domain: str
    def __init__(self, id: _Optional[str] = ..., file_id: _Optional[str] = ..., path: _Optional[str] = ..., current_domain: _Optional[str] = ...) -> None: ...

class GraphSnapshot(_message.Message):
    __slots__ = ("id", "scenario", "content_hash", "languages", "extracted_at", "extraction_ms", "files", "packages", "symbols", "imports")
    ID_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    LANGUAGES_FIELD_NUMBER: _ClassVar[int]
    EXTRACTED_AT_FIELD_NUMBER: _ClassVar[int]
    EXTRACTION_MS_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    PACKAGES_FIELD_NUMBER: _ClassVar[int]
    SYMBOLS_FIELD_NUMBER: _ClassVar[int]
    IMPORTS_FIELD_NUMBER: _ClassVar[int]
    id: str
    scenario: str
    content_hash: str
    languages: _containers.RepeatedScalarFieldContainer[Language]
    extracted_at: _timestamp_pb2.Timestamp
    extraction_ms: int
    files: _containers.RepeatedCompositeFieldContainer[FileNode]
    packages: _containers.RepeatedCompositeFieldContainer[PackageNode]
    symbols: _containers.RepeatedCompositeFieldContainer[SymbolNode]
    imports: _containers.RepeatedCompositeFieldContainer[ImportEdge]
    def __init__(self, id: _Optional[str] = ..., scenario: _Optional[str] = ..., content_hash: _Optional[str] = ..., languages: _Optional[_Iterable[_Union[Language, str]]] = ..., extracted_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., extraction_ms: _Optional[int] = ..., files: _Optional[_Iterable[_Union[FileNode, _Mapping]]] = ..., packages: _Optional[_Iterable[_Union[PackageNode, _Mapping]]] = ..., symbols: _Optional[_Iterable[_Union[SymbolNode, _Mapping]]] = ..., imports: _Optional[_Iterable[_Union[ImportEdge, _Mapping]]] = ...) -> None: ...

class ExtractGraphRequest(_message.Message):
    __slots__ = ("scenario", "languages", "idempotency_key")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    LANGUAGES_FIELD_NUMBER: _ClassVar[int]
    IDEMPOTENCY_KEY_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    languages: _containers.RepeatedScalarFieldContainer[Language]
    idempotency_key: str
    def __init__(self, scenario: _Optional[str] = ..., languages: _Optional[_Iterable[_Union[Language, str]]] = ..., idempotency_key: _Optional[str] = ...) -> None: ...

class ExtractGraphResponse(_message.Message):
    __slots__ = ("snapshot", "from_cache")
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    FROM_CACHE_FIELD_NUMBER: _ClassVar[int]
    snapshot: GraphSnapshot
    from_cache: bool
    def __init__(self, snapshot: _Optional[_Union[GraphSnapshot, _Mapping]] = ..., from_cache: _Optional[bool] = ...) -> None: ...

class GetGraphSnapshotRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetGraphSnapshotResponse(_message.Message):
    __slots__ = ("snapshot",)
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    snapshot: GraphSnapshot
    def __init__(self, snapshot: _Optional[_Union[GraphSnapshot, _Mapping]] = ...) -> None: ...

class ListGraphSnapshotsRequest(_message.Message):
    __slots__ = ("scenario", "page_size", "page_token")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    page_size: int
    page_token: str
    def __init__(self, scenario: _Optional[str] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListGraphSnapshotsResponse(_message.Message):
    __slots__ = ("snapshots", "next_page_token")
    SNAPSHOTS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    snapshots: _containers.RepeatedCompositeFieldContainer[GraphSnapshot]
    next_page_token: str
    def __init__(self, snapshots: _Optional[_Iterable[_Union[GraphSnapshot, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...

class ClearGraphSnapshotsRequest(_message.Message):
    __slots__ = ("scenario", "dry_run")
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    scenario: str
    dry_run: bool
    def __init__(self, scenario: _Optional[str] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class ClearGraphSnapshotsResponse(_message.Message):
    __slots__ = ("deleted", "dry_run")
    DELETED_FIELD_NUMBER: _ClassVar[int]
    DRY_RUN_FIELD_NUMBER: _ClassVar[int]
    deleted: int
    dry_run: bool
    def __init__(self, deleted: _Optional[int] = ..., dry_run: _Optional[bool] = ...) -> None: ...

class ExportGraphRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ExportGraphResponse(_message.Message):
    __slots__ = ("payload", "content_type")
    PAYLOAD_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    payload: bytes
    content_type: str
    def __init__(self, payload: _Optional[bytes] = ..., content_type: _Optional[str] = ...) -> None: ...
