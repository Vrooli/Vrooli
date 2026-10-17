from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PreviewKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PREVIEW_KIND_UNSPECIFIED: _ClassVar[PreviewKind]
    PREVIEW_KIND_MARKDOWN: _ClassVar[PreviewKind]
    PREVIEW_KIND_CODE: _ClassVar[PreviewKind]
    PREVIEW_KIND_TEXT: _ClassVar[PreviewKind]
    PREVIEW_KIND_SVG: _ClassVar[PreviewKind]
    PREVIEW_KIND_IMAGE: _ClassVar[PreviewKind]
    PREVIEW_KIND_PDF: _ClassVar[PreviewKind]
    PREVIEW_KIND_AUDIO: _ClassVar[PreviewKind]
    PREVIEW_KIND_VIDEO: _ClassVar[PreviewKind]
    PREVIEW_KIND_CSV: _ClassVar[PreviewKind]
    PREVIEW_KIND_DIFF: _ClassVar[PreviewKind]
    PREVIEW_KIND_UNSUPPORTED: _ClassVar[PreviewKind]
    PREVIEW_KIND_DIRECTORY: _ClassVar[PreviewKind]

class DirectorySort(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DIRECTORY_SORT_UNSPECIFIED: _ClassVar[DirectorySort]
    DIRECTORY_SORT_DIRS_FIRST_NAME: _ClassVar[DirectorySort]
    DIRECTORY_SORT_NAME: _ClassVar[DirectorySort]
    DIRECTORY_SORT_SIZE_DESC: _ClassVar[DirectorySort]
    DIRECTORY_SORT_MTIME_DESC: _ClassVar[DirectorySort]

class EntryType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ENTRY_TYPE_UNSPECIFIED: _ClassVar[EntryType]
    ENTRY_TYPE_FILE: _ClassVar[EntryType]
    ENTRY_TYPE_DIRECTORY: _ClassVar[EntryType]
    ENTRY_TYPE_SYMLINK: _ClassVar[EntryType]
    ENTRY_TYPE_OTHER: _ClassVar[EntryType]

class SourceContext(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SOURCE_CONTEXT_UNSPECIFIED: _ClassVar[SourceContext]
    SOURCE_CONTEXT_MESSAGE_LINK: _ClassVar[SourceContext]
    SOURCE_CONTEXT_INLINE_CODE: _ClassVar[SourceContext]
    SOURCE_CONTEXT_CLI: _ClassVar[SourceContext]
PREVIEW_KIND_UNSPECIFIED: PreviewKind
PREVIEW_KIND_MARKDOWN: PreviewKind
PREVIEW_KIND_CODE: PreviewKind
PREVIEW_KIND_TEXT: PreviewKind
PREVIEW_KIND_SVG: PreviewKind
PREVIEW_KIND_IMAGE: PreviewKind
PREVIEW_KIND_PDF: PreviewKind
PREVIEW_KIND_AUDIO: PreviewKind
PREVIEW_KIND_VIDEO: PreviewKind
PREVIEW_KIND_CSV: PreviewKind
PREVIEW_KIND_DIFF: PreviewKind
PREVIEW_KIND_UNSUPPORTED: PreviewKind
PREVIEW_KIND_DIRECTORY: PreviewKind
DIRECTORY_SORT_UNSPECIFIED: DirectorySort
DIRECTORY_SORT_DIRS_FIRST_NAME: DirectorySort
DIRECTORY_SORT_NAME: DirectorySort
DIRECTORY_SORT_SIZE_DESC: DirectorySort
DIRECTORY_SORT_MTIME_DESC: DirectorySort
ENTRY_TYPE_UNSPECIFIED: EntryType
ENTRY_TYPE_FILE: EntryType
ENTRY_TYPE_DIRECTORY: EntryType
ENTRY_TYPE_SYMLINK: EntryType
ENTRY_TYPE_OTHER: EntryType
SOURCE_CONTEXT_UNSPECIFIED: SourceContext
SOURCE_CONTEXT_MESSAGE_LINK: SourceContext
SOURCE_CONTEXT_INLINE_CODE: SourceContext
SOURCE_CONTEXT_CLI: SourceContext

class ResolveRequest(_message.Message):
    __slots__ = ("session_id", "path", "source_context")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    SOURCE_CONTEXT_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    path: str
    source_context: SourceContext
    def __init__(self, session_id: _Optional[str] = ..., path: _Optional[str] = ..., source_context: _Optional[_Union[SourceContext, str]] = ...) -> None: ...

class ResolveResponse(_message.Message):
    __slots__ = ("preview_id", "input_path", "resolved_path", "basename", "line", "has_line", "resolution_basis", "preview_kind", "mime_type", "size_bytes", "mtime_unix_nano", "can_preview", "can_download", "supports_range", "text_content_available", "blob_url", "expires_unix_nano", "warnings", "listing_available")
    PREVIEW_ID_FIELD_NUMBER: _ClassVar[int]
    INPUT_PATH_FIELD_NUMBER: _ClassVar[int]
    RESOLVED_PATH_FIELD_NUMBER: _ClassVar[int]
    BASENAME_FIELD_NUMBER: _ClassVar[int]
    LINE_FIELD_NUMBER: _ClassVar[int]
    HAS_LINE_FIELD_NUMBER: _ClassVar[int]
    RESOLUTION_BASIS_FIELD_NUMBER: _ClassVar[int]
    PREVIEW_KIND_FIELD_NUMBER: _ClassVar[int]
    MIME_TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    MTIME_UNIX_NANO_FIELD_NUMBER: _ClassVar[int]
    CAN_PREVIEW_FIELD_NUMBER: _ClassVar[int]
    CAN_DOWNLOAD_FIELD_NUMBER: _ClassVar[int]
    SUPPORTS_RANGE_FIELD_NUMBER: _ClassVar[int]
    TEXT_CONTENT_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    BLOB_URL_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_UNIX_NANO_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    LISTING_AVAILABLE_FIELD_NUMBER: _ClassVar[int]
    preview_id: str
    input_path: str
    resolved_path: str
    basename: str
    line: int
    has_line: bool
    resolution_basis: str
    preview_kind: PreviewKind
    mime_type: str
    size_bytes: int
    mtime_unix_nano: int
    can_preview: bool
    can_download: bool
    supports_range: bool
    text_content_available: bool
    blob_url: str
    expires_unix_nano: int
    warnings: _containers.RepeatedScalarFieldContainer[str]
    listing_available: bool
    def __init__(self, preview_id: _Optional[str] = ..., input_path: _Optional[str] = ..., resolved_path: _Optional[str] = ..., basename: _Optional[str] = ..., line: _Optional[int] = ..., has_line: _Optional[bool] = ..., resolution_basis: _Optional[str] = ..., preview_kind: _Optional[_Union[PreviewKind, str]] = ..., mime_type: _Optional[str] = ..., size_bytes: _Optional[int] = ..., mtime_unix_nano: _Optional[int] = ..., can_preview: _Optional[bool] = ..., can_download: _Optional[bool] = ..., supports_range: _Optional[bool] = ..., text_content_available: _Optional[bool] = ..., blob_url: _Optional[str] = ..., expires_unix_nano: _Optional[int] = ..., warnings: _Optional[_Iterable[str]] = ..., listing_available: _Optional[bool] = ...) -> None: ...

class GetTextContentRequest(_message.Message):
    __slots__ = ("session_id", "preview_id")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    PREVIEW_ID_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    preview_id: str
    def __init__(self, session_id: _Optional[str] = ..., preview_id: _Optional[str] = ...) -> None: ...

class GetTextContentResponse(_message.Message):
    __slots__ = ("resolved_path", "preview_kind", "mime_type", "content", "truncated", "line", "has_line")
    RESOLVED_PATH_FIELD_NUMBER: _ClassVar[int]
    PREVIEW_KIND_FIELD_NUMBER: _ClassVar[int]
    MIME_TYPE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    LINE_FIELD_NUMBER: _ClassVar[int]
    HAS_LINE_FIELD_NUMBER: _ClassVar[int]
    resolved_path: str
    preview_kind: PreviewKind
    mime_type: str
    content: str
    truncated: bool
    line: int
    has_line: bool
    def __init__(self, resolved_path: _Optional[str] = ..., preview_kind: _Optional[_Union[PreviewKind, str]] = ..., mime_type: _Optional[str] = ..., content: _Optional[str] = ..., truncated: _Optional[bool] = ..., line: _Optional[int] = ..., has_line: _Optional[bool] = ...) -> None: ...

class ListDirectoryRequest(_message.Message):
    __slots__ = ("session_id", "preview_id", "sort", "show_hidden", "page_size", "page_token")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    PREVIEW_ID_FIELD_NUMBER: _ClassVar[int]
    SORT_FIELD_NUMBER: _ClassVar[int]
    SHOW_HIDDEN_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    preview_id: str
    sort: DirectorySort
    show_hidden: bool
    page_size: int
    page_token: str
    def __init__(self, session_id: _Optional[str] = ..., preview_id: _Optional[str] = ..., sort: _Optional[_Union[DirectorySort, str]] = ..., show_hidden: _Optional[bool] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class DirectoryEntry(_message.Message):
    __slots__ = ("name", "entry_type", "preview_kind", "size_bytes", "mtime_unix_nano", "can_preview", "symlink_target", "symlink_broken", "mode", "child_count")
    NAME_FIELD_NUMBER: _ClassVar[int]
    ENTRY_TYPE_FIELD_NUMBER: _ClassVar[int]
    PREVIEW_KIND_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    MTIME_UNIX_NANO_FIELD_NUMBER: _ClassVar[int]
    CAN_PREVIEW_FIELD_NUMBER: _ClassVar[int]
    SYMLINK_TARGET_FIELD_NUMBER: _ClassVar[int]
    SYMLINK_BROKEN_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    CHILD_COUNT_FIELD_NUMBER: _ClassVar[int]
    name: str
    entry_type: EntryType
    preview_kind: PreviewKind
    size_bytes: int
    mtime_unix_nano: int
    can_preview: bool
    symlink_target: str
    symlink_broken: bool
    mode: str
    child_count: int
    def __init__(self, name: _Optional[str] = ..., entry_type: _Optional[_Union[EntryType, str]] = ..., preview_kind: _Optional[_Union[PreviewKind, str]] = ..., size_bytes: _Optional[int] = ..., mtime_unix_nano: _Optional[int] = ..., can_preview: _Optional[bool] = ..., symlink_target: _Optional[str] = ..., symlink_broken: _Optional[bool] = ..., mode: _Optional[str] = ..., child_count: _Optional[int] = ...) -> None: ...

class ListDirectoryResponse(_message.Message):
    __slots__ = ("resolved_path", "parent_path", "entries", "total_entries", "truncated", "next_page_token", "effective_sort", "warnings")
    RESOLVED_PATH_FIELD_NUMBER: _ClassVar[int]
    PARENT_PATH_FIELD_NUMBER: _ClassVar[int]
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    TOTAL_ENTRIES_FIELD_NUMBER: _ClassVar[int]
    TRUNCATED_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    EFFECTIVE_SORT_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    resolved_path: str
    parent_path: str
    entries: _containers.RepeatedCompositeFieldContainer[DirectoryEntry]
    total_entries: int
    truncated: bool
    next_page_token: str
    effective_sort: DirectorySort
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, resolved_path: _Optional[str] = ..., parent_path: _Optional[str] = ..., entries: _Optional[_Iterable[_Union[DirectoryEntry, _Mapping]]] = ..., total_entries: _Optional[int] = ..., truncated: _Optional[bool] = ..., next_page_token: _Optional[str] = ..., effective_sort: _Optional[_Union[DirectorySort, str]] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...
