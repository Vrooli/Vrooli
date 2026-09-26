import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Asset(_message.Message):
    __slots__ = ("id", "brand_id", "filename", "mime_type", "size", "created_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    FILENAME_FIELD_NUMBER: _ClassVar[int]
    MIME_TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    brand_id: str
    filename: str
    mime_type: str
    size: int
    created_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., brand_id: _Optional[str] = ..., filename: _Optional[str] = ..., mime_type: _Optional[str] = ..., size: _Optional[int] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListAssetsRequest(_message.Message):
    __slots__ = ("brand_id",)
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    brand_id: str
    def __init__(self, brand_id: _Optional[str] = ...) -> None: ...

class ListAssetsResponse(_message.Message):
    __slots__ = ("assets",)
    ASSETS_FIELD_NUMBER: _ClassVar[int]
    assets: _containers.RepeatedCompositeFieldContainer[Asset]
    def __init__(self, assets: _Optional[_Iterable[_Union[Asset, _Mapping]]] = ...) -> None: ...

class UploadAssetRequest(_message.Message):
    __slots__ = ("brand_id", "filename", "mime_type", "content")
    BRAND_ID_FIELD_NUMBER: _ClassVar[int]
    FILENAME_FIELD_NUMBER: _ClassVar[int]
    MIME_TYPE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    brand_id: str
    filename: str
    mime_type: str
    content: bytes
    def __init__(self, brand_id: _Optional[str] = ..., filename: _Optional[str] = ..., mime_type: _Optional[str] = ..., content: _Optional[bytes] = ...) -> None: ...

class UploadAssetResponse(_message.Message):
    __slots__ = ("asset",)
    ASSET_FIELD_NUMBER: _ClassVar[int]
    asset: Asset
    def __init__(self, asset: _Optional[_Union[Asset, _Mapping]] = ...) -> None: ...

class GetAssetRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetAssetResponse(_message.Message):
    __slots__ = ("asset",)
    ASSET_FIELD_NUMBER: _ClassVar[int]
    asset: Asset
    def __init__(self, asset: _Optional[_Union[Asset, _Mapping]] = ...) -> None: ...

class DownloadAssetRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DownloadAssetResponse(_message.Message):
    __slots__ = ("filename", "mime_type", "content")
    FILENAME_FIELD_NUMBER: _ClassVar[int]
    MIME_TYPE_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    filename: str
    mime_type: str
    content: bytes
    def __init__(self, filename: _Optional[str] = ..., mime_type: _Optional[str] = ..., content: _Optional[bytes] = ...) -> None: ...

class DeleteAssetRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteAssetResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
