from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DocumentationManifestRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class DocumentationManifestResponse(_message.Message):
    __slots__ = ("version", "title", "description", "default_document", "sections", "primary_navigation", "secondary_navigation")
    VERSION_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    DEFAULT_DOCUMENT_FIELD_NUMBER: _ClassVar[int]
    SECTIONS_FIELD_NUMBER: _ClassVar[int]
    PRIMARY_NAVIGATION_FIELD_NUMBER: _ClassVar[int]
    SECONDARY_NAVIGATION_FIELD_NUMBER: _ClassVar[int]
    version: str
    title: str
    description: str
    default_document: str
    sections: _containers.RepeatedCompositeFieldContainer[DocumentationSection]
    primary_navigation: _containers.RepeatedScalarFieldContainer[str]
    secondary_navigation: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, version: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., default_document: _Optional[str] = ..., sections: _Optional[_Iterable[_Union[DocumentationSection, _Mapping]]] = ..., primary_navigation: _Optional[_Iterable[str]] = ..., secondary_navigation: _Optional[_Iterable[str]] = ...) -> None: ...

class DocumentationSection(_message.Message):
    __slots__ = ("id", "title", "icon", "description", "documents")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    ICON_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    DOCUMENTS_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    icon: str
    description: str
    documents: _containers.RepeatedCompositeFieldContainer[DocumentationDocument]
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., icon: _Optional[str] = ..., description: _Optional[str] = ..., documents: _Optional[_Iterable[_Union[DocumentationDocument, _Mapping]]] = ...) -> None: ...

class DocumentationDocument(_message.Message):
    __slots__ = ("path", "title", "description")
    PATH_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    path: str
    title: str
    description: str
    def __init__(self, path: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ...) -> None: ...

class DocumentationContentRequest(_message.Message):
    __slots__ = ("path",)
    PATH_FIELD_NUMBER: _ClassVar[int]
    path: str
    def __init__(self, path: _Optional[str] = ...) -> None: ...

class DocumentationContentResponse(_message.Message):
    __slots__ = ("path", "content")
    PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    path: str
    content: str
    def __init__(self, path: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...
