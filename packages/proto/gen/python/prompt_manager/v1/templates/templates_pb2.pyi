from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListAgentFileTemplatesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class AgentFileTemplate(_message.Message):
    __slots__ = ("id", "name", "description", "file_name", "content")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    FILE_NAME_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    file_name: str
    content: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., file_name: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class ListAgentFileTemplatesResponse(_message.Message):
    __slots__ = ("templates", "count")
    TEMPLATES_FIELD_NUMBER: _ClassVar[int]
    COUNT_FIELD_NUMBER: _ClassVar[int]
    templates: _containers.RepeatedCompositeFieldContainer[AgentFileTemplate]
    count: int
    def __init__(self, templates: _Optional[_Iterable[_Union[AgentFileTemplate, _Mapping]]] = ..., count: _Optional[int] = ...) -> None: ...
