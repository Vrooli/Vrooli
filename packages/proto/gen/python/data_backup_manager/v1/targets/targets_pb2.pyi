import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from data_backup_manager.v1.sources import sources_pb2 as _sources_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Target(_message.Message):
    __slots__ = ("id", "owner", "name", "source_kind", "locator", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SOURCE_KIND_FIELD_NUMBER: _ClassVar[int]
    LOCATOR_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    owner: str
    name: str
    source_kind: _sources_pb2.SourceKind
    locator: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., owner: _Optional[str] = ..., name: _Optional[str] = ..., source_kind: _Optional[_Union[_sources_pb2.SourceKind, str]] = ..., locator: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RegisterTargetRequest(_message.Message):
    __slots__ = ("owner", "name", "source_kind", "locator")
    OWNER_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SOURCE_KIND_FIELD_NUMBER: _ClassVar[int]
    LOCATOR_FIELD_NUMBER: _ClassVar[int]
    owner: str
    name: str
    source_kind: _sources_pb2.SourceKind
    locator: str
    def __init__(self, owner: _Optional[str] = ..., name: _Optional[str] = ..., source_kind: _Optional[_Union[_sources_pb2.SourceKind, str]] = ..., locator: _Optional[str] = ...) -> None: ...

class RegisterTargetResponse(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: Target
    def __init__(self, target: _Optional[_Union[Target, _Mapping]] = ...) -> None: ...

class DeregisterTargetRequest(_message.Message):
    __slots__ = ("owner", "name")
    OWNER_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    owner: str
    name: str
    def __init__(self, owner: _Optional[str] = ..., name: _Optional[str] = ...) -> None: ...

class DeregisterTargetResponse(_message.Message):
    __slots__ = ("removed",)
    REMOVED_FIELD_NUMBER: _ClassVar[int]
    removed: bool
    def __init__(self, removed: _Optional[bool] = ...) -> None: ...

class GetTargetRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetTargetResponse(_message.Message):
    __slots__ = ("target",)
    TARGET_FIELD_NUMBER: _ClassVar[int]
    target: Target
    def __init__(self, target: _Optional[_Union[Target, _Mapping]] = ...) -> None: ...

class ListTargetsRequest(_message.Message):
    __slots__ = ("owner", "page_size", "page_token")
    OWNER_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    owner: str
    page_size: int
    page_token: str
    def __init__(self, owner: _Optional[str] = ..., page_size: _Optional[int] = ..., page_token: _Optional[str] = ...) -> None: ...

class ListTargetsResponse(_message.Message):
    __slots__ = ("targets", "next_page_token")
    TARGETS_FIELD_NUMBER: _ClassVar[int]
    NEXT_PAGE_TOKEN_FIELD_NUMBER: _ClassVar[int]
    targets: _containers.RepeatedCompositeFieldContainer[Target]
    next_page_token: str
    def __init__(self, targets: _Optional[_Iterable[_Union[Target, _Mapping]]] = ..., next_page_token: _Optional[str] = ...) -> None: ...
