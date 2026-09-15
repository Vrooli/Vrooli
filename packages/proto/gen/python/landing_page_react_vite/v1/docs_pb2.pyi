from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DocEntry(_message.Message):
    __slots__ = ("name", "path", "is_dir", "children")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    IS_DIR_FIELD_NUMBER: _ClassVar[int]
    CHILDREN_FIELD_NUMBER: _ClassVar[int]
    name: str
    path: str
    is_dir: bool
    children: _containers.RepeatedCompositeFieldContainer[DocEntry]
    def __init__(self, name: _Optional[str] = ..., path: _Optional[str] = ..., is_dir: _Optional[bool] = ..., children: _Optional[_Iterable[_Union[DocEntry, _Mapping]]] = ...) -> None: ...

class GetDocsTreeRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class GetDocsTreeResponse(_message.Message):
    __slots__ = ("entries",)
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[DocEntry]
    def __init__(self, entries: _Optional[_Iterable[_Union[DocEntry, _Mapping]]] = ...) -> None: ...

class GetDocContentRequest(_message.Message):
    __slots__ = ("path",)
    PATH_FIELD_NUMBER: _ClassVar[int]
    path: str
    def __init__(self, path: _Optional[str] = ...) -> None: ...

class GetDocContentResponse(_message.Message):
    __slots__ = ("path", "content", "title")
    PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    path: str
    content: str
    title: str
    def __init__(self, path: _Optional[str] = ..., content: _Optional[str] = ..., title: _Optional[str] = ...) -> None: ...
