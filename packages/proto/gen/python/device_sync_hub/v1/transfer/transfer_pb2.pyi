import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ItemKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    ITEM_KIND_UNSPECIFIED: _ClassVar[ItemKind]
    ITEM_KIND_TEXT: _ClassVar[ItemKind]
    ITEM_KIND_FILE: _ClassVar[ItemKind]

class Retention(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RETENTION_UNSPECIFIED: _ClassVar[Retention]
    RETENTION_LIVE: _ClassVar[Retention]
    RETENTION_HELD: _ClassVar[Retention]
    RETENTION_PINNED: _ClassVar[Retention]
ITEM_KIND_UNSPECIFIED: ItemKind
ITEM_KIND_TEXT: ItemKind
ITEM_KIND_FILE: ItemKind
RETENTION_UNSPECIFIED: Retention
RETENTION_LIVE: Retention
RETENTION_HELD: Retention
RETENTION_PINNED: Retention

class Item(_message.Message):
    __slots__ = ("id", "owner_id", "origin_device_id", "kind", "name", "mime", "size_bytes", "text", "has_thumbnail", "retention", "target_device_id", "expires_at", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    OWNER_ID_FIELD_NUMBER: _ClassVar[int]
    ORIGIN_DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    MIME_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    HAS_THUMBNAIL_FIELD_NUMBER: _ClassVar[int]
    RETENTION_FIELD_NUMBER: _ClassVar[int]
    TARGET_DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    owner_id: str
    origin_device_id: str
    kind: ItemKind
    name: str
    mime: str
    size_bytes: int
    text: str
    has_thumbnail: bool
    retention: Retention
    target_device_id: str
    expires_at: _timestamp_pb2.Timestamp
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., owner_id: _Optional[str] = ..., origin_device_id: _Optional[str] = ..., kind: _Optional[_Union[ItemKind, str]] = ..., name: _Optional[str] = ..., mime: _Optional[str] = ..., size_bytes: _Optional[int] = ..., text: _Optional[str] = ..., has_thumbnail: _Optional[bool] = ..., retention: _Optional[_Union[Retention, str]] = ..., target_device_id: _Optional[str] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class CreateTextItemRequest(_message.Message):
    __slots__ = ("text", "name", "retention", "target_device_id")
    TEXT_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    RETENTION_FIELD_NUMBER: _ClassVar[int]
    TARGET_DEVICE_ID_FIELD_NUMBER: _ClassVar[int]
    text: str
    name: str
    retention: Retention
    target_device_id: str
    def __init__(self, text: _Optional[str] = ..., name: _Optional[str] = ..., retention: _Optional[_Union[Retention, str]] = ..., target_device_id: _Optional[str] = ...) -> None: ...

class CreateTextItemResponse(_message.Message):
    __slots__ = ("item",)
    ITEM_FIELD_NUMBER: _ClassVar[int]
    item: Item
    def __init__(self, item: _Optional[_Union[Item, _Mapping]] = ...) -> None: ...

class ListItemsRequest(_message.Message):
    __slots__ = ("query", "kind")
    QUERY_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    query: str
    kind: ItemKind
    def __init__(self, query: _Optional[str] = ..., kind: _Optional[_Union[ItemKind, str]] = ...) -> None: ...

class ListItemsResponse(_message.Message):
    __slots__ = ("items",)
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[Item]
    def __init__(self, items: _Optional[_Iterable[_Union[Item, _Mapping]]] = ...) -> None: ...

class GetItemRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetItemResponse(_message.Message):
    __slots__ = ("item",)
    ITEM_FIELD_NUMBER: _ClassVar[int]
    item: Item
    def __init__(self, item: _Optional[_Union[Item, _Mapping]] = ...) -> None: ...

class DeleteItemRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteItemResponse(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class UploadItemResponse(_message.Message):
    __slots__ = ("item",)
    ITEM_FIELD_NUMBER: _ClassVar[int]
    item: Item
    def __init__(self, item: _Optional[_Union[Item, _Mapping]] = ...) -> None: ...
