from landing_page_business_suite.v1.shared import downloads_pb2 as _downloads_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AuthorizeDownloadRequest(_message.Message):
    __slots__ = ("app", "platform")
    APP_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    app: str
    platform: str
    def __init__(self, app: _Optional[str] = ..., platform: _Optional[str] = ...) -> None: ...

class AuthorizeDownloadResponse(_message.Message):
    __slots__ = ("asset",)
    ASSET_FIELD_NUMBER: _ClassVar[int]
    asset: _downloads_pb2.DownloadAsset
    def __init__(self, asset: _Optional[_Union[_downloads_pb2.DownloadAsset, _Mapping]] = ...) -> None: ...

class ListDownloadAppsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListDownloadAppsResponse(_message.Message):
    __slots__ = ("apps",)
    APPS_FIELD_NUMBER: _ClassVar[int]
    apps: _containers.RepeatedCompositeFieldContainer[_downloads_pb2.DownloadApp]
    def __init__(self, apps: _Optional[_Iterable[_Union[_downloads_pb2.DownloadApp, _Mapping]]] = ...) -> None: ...

class SaveDownloadAppRequest(_message.Message):
    __slots__ = ("app_key", "app")
    APP_KEY_FIELD_NUMBER: _ClassVar[int]
    APP_FIELD_NUMBER: _ClassVar[int]
    app_key: str
    app: _downloads_pb2.DownloadApp
    def __init__(self, app_key: _Optional[str] = ..., app: _Optional[_Union[_downloads_pb2.DownloadApp, _Mapping]] = ...) -> None: ...

class CreateDownloadAppRequest(_message.Message):
    __slots__ = ("app",)
    APP_FIELD_NUMBER: _ClassVar[int]
    app: _downloads_pb2.DownloadApp
    def __init__(self, app: _Optional[_Union[_downloads_pb2.DownloadApp, _Mapping]] = ...) -> None: ...

class DownloadAppResponse(_message.Message):
    __slots__ = ("app",)
    APP_FIELD_NUMBER: _ClassVar[int]
    app: _downloads_pb2.DownloadApp
    def __init__(self, app: _Optional[_Union[_downloads_pb2.DownloadApp, _Mapping]] = ...) -> None: ...

class DeleteDownloadAppRequest(_message.Message):
    __slots__ = ("app_key",)
    APP_KEY_FIELD_NUMBER: _ClassVar[int]
    app_key: str
    def __init__(self, app_key: _Optional[str] = ...) -> None: ...

class DeleteDownloadAppResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
