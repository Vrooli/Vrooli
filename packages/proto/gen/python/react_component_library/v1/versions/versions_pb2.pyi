import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DiffOp(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    DIFF_OP_UNSPECIFIED: _ClassVar[DiffOp]
    DIFF_OP_EQUAL: _ClassVar[DiffOp]
    DIFF_OP_REMOVE: _ClassVar[DiffOp]
    DIFF_OP_ADD: _ClassVar[DiffOp]
    DIFF_OP_EMPTY: _ClassVar[DiffOp]
DIFF_OP_UNSPECIFIED: DiffOp
DIFF_OP_EQUAL: DiffOp
DIFF_OP_REMOVE: DiffOp
DIFF_OP_ADD: DiffOp
DIFF_OP_EMPTY: DiffOp

class Version(_message.Message):
    __slots__ = ("id", "component_id", "version", "content_sha256", "changelog_md", "recorded_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CONTENT_SHA256_FIELD_NUMBER: _ClassVar[int]
    CHANGELOG_MD_FIELD_NUMBER: _ClassVar[int]
    RECORDED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    component_id: str
    version: str
    content_sha256: str
    changelog_md: str
    recorded_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., component_id: _Optional[str] = ..., version: _Optional[str] = ..., content_sha256: _Optional[str] = ..., changelog_md: _Optional[str] = ..., recorded_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListVersionsRequest(_message.Message):
    __slots__ = ("component_id", "limit")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    limit: int
    def __init__(self, component_id: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class ListVersionsResponse(_message.Message):
    __slots__ = ("versions",)
    VERSIONS_FIELD_NUMBER: _ClassVar[int]
    versions: _containers.RepeatedCompositeFieldContainer[Version]
    def __init__(self, versions: _Optional[_Iterable[_Union[Version, _Mapping]]] = ...) -> None: ...

class GetVersionRequest(_message.Message):
    __slots__ = ("component_id", "version", "include_content")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    INCLUDE_CONTENT_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    version: str
    include_content: bool
    def __init__(self, component_id: _Optional[str] = ..., version: _Optional[str] = ..., include_content: _Optional[bool] = ...) -> None: ...

class GetVersionResponse(_message.Message):
    __slots__ = ("version", "content")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    version: Version
    content: str
    def __init__(self, version: _Optional[_Union[Version, _Mapping]] = ..., content: _Optional[str] = ...) -> None: ...

class DiffCell(_message.Message):
    __slots__ = ("line_number", "text", "op")
    LINE_NUMBER_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    OP_FIELD_NUMBER: _ClassVar[int]
    line_number: int
    text: str
    op: DiffOp
    def __init__(self, line_number: _Optional[int] = ..., text: _Optional[str] = ..., op: _Optional[_Union[DiffOp, str]] = ...) -> None: ...

class DiffRow(_message.Message):
    __slots__ = ("left", "right")
    LEFT_FIELD_NUMBER: _ClassVar[int]
    RIGHT_FIELD_NUMBER: _ClassVar[int]
    left: DiffCell
    right: DiffCell
    def __init__(self, left: _Optional[_Union[DiffCell, _Mapping]] = ..., right: _Optional[_Union[DiffCell, _Mapping]] = ...) -> None: ...

class DiffVersionsRequest(_message.Message):
    __slots__ = ("component_id", "to")
    COMPONENT_ID_FIELD_NUMBER: _ClassVar[int]
    FROM_FIELD_NUMBER: _ClassVar[int]
    TO_FIELD_NUMBER: _ClassVar[int]
    component_id: str
    to: str
    def __init__(self, component_id: _Optional[str] = ..., to: _Optional[str] = ..., **kwargs) -> None: ...

class DiffVersionsResponse(_message.Message):
    __slots__ = ("rows", "additions", "removals", "from_label", "to_label")
    ROWS_FIELD_NUMBER: _ClassVar[int]
    ADDITIONS_FIELD_NUMBER: _ClassVar[int]
    REMOVALS_FIELD_NUMBER: _ClassVar[int]
    FROM_LABEL_FIELD_NUMBER: _ClassVar[int]
    TO_LABEL_FIELD_NUMBER: _ClassVar[int]
    rows: _containers.RepeatedCompositeFieldContainer[DiffRow]
    additions: int
    removals: int
    from_label: str
    to_label: str
    def __init__(self, rows: _Optional[_Iterable[_Union[DiffRow, _Mapping]]] = ..., additions: _Optional[int] = ..., removals: _Optional[int] = ..., from_label: _Optional[str] = ..., to_label: _Optional[str] = ...) -> None: ...
