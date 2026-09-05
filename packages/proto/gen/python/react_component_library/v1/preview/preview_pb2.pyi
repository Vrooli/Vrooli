from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class GetPreviewBundleRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetPreviewBundleResponse(_message.Message):
    __slots__ = ("js", "source_path", "sha256", "warnings")
    JS_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    SHA256_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    js: str
    source_path: str
    sha256: str
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, js: _Optional[str] = ..., source_path: _Optional[str] = ..., sha256: _Optional[str] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...
