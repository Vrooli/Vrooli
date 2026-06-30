from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable
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
    __slots__ = ("preview_id", "input_path", "resolved_path", "basename", "line", "has_line", "resolution_basis", "preview_kind", "mime_type", "size_bytes", "mtime_unix_nano", "can_preview", "can_download", "supports_range", "text_content_available", "blob_url", "expires_unix_nano", "warnings")
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
    def __init__(self, preview_id: _Optional[str] = ..., input_path: _Optional[str] = ..., resolved_path: _Optional[str] = ..., basename: _Optional[str] = ..., line: _Optional[int] = ..., has_line: _Optional[bool] = ..., resolution_basis: _Optional[str] = ..., preview_kind: _Optional[_Union[PreviewKind, str]] = ..., mime_type: _Optional[str] = ..., size_bytes: _Optional[int] = ..., mtime_unix_nano: _Optional[int] = ..., can_preview: _Optional[bool] = ..., can_download: _Optional[bool] = ..., supports_range: _Optional[bool] = ..., text_content_available: _Optional[bool] = ..., blob_url: _Optional[str] = ..., expires_unix_nano: _Optional[int] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...

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
