from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class FetchOpenGraphRequest(_message.Message):
    __slots__ = ("url",)
    URL_FIELD_NUMBER: _ClassVar[int]
    url: str
    def __init__(self, url: _Optional[str] = ...) -> None: ...

class OpenGraphMetadata(_message.Message):
    __slots__ = ("url", "title", "description", "image", "site_name", "type", "favicon")
    URL_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    IMAGE_FIELD_NUMBER: _ClassVar[int]
    SITE_NAME_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    FAVICON_FIELD_NUMBER: _ClassVar[int]
    url: str
    title: str
    description: str
    image: str
    site_name: str
    type: str
    favicon: str
    def __init__(self, url: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., image: _Optional[str] = ..., site_name: _Optional[str] = ..., type: _Optional[str] = ..., favicon: _Optional[str] = ...) -> None: ...
