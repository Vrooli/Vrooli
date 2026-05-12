import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Component(_message.Message):
    __slots__ = ("id", "library_id", "display_name", "description", "source_path", "version", "tags", "indexed_at", "updated_at", "headers")
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
    def __init__(self, id: _Optional[str] = ..., library_id: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., source_path: _Optional[str] = ..., version: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., indexed_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., headers: _Optional[_Mapping[str, str]] = ...) -> None: ...

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
