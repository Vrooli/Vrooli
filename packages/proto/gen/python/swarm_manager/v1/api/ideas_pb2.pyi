from buf.validate import validate_pb2 as _validate_pb2
from swarm_manager.v1.domain import idea_pb2 as _idea_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CreateIdeaRequest(_message.Message):
    __slots__ = ("name", "title", "description", "priority", "tags")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    name: str
    title: str
    description: str
    priority: int
    tags: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., priority: _Optional[int] = ..., tags: _Optional[_Iterable[str]] = ...) -> None: ...

class UpdateIdeaRequest(_message.Message):
    __slots__ = ("title", "description", "status", "priority", "tags")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    title: str
    description: str
    status: str
    priority: int
    tags: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, title: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ..., priority: _Optional[int] = ..., tags: _Optional[_Iterable[str]] = ...) -> None: ...

class ListIdeasResponse(_message.Message):
    __slots__ = ("ideas",)
    IDEAS_FIELD_NUMBER: _ClassVar[int]
    ideas: _containers.RepeatedCompositeFieldContainer[_idea_pb2.Idea]
    def __init__(self, ideas: _Optional[_Iterable[_Union[_idea_pb2.Idea, _Mapping]]] = ...) -> None: ...

class IdeaResponse(_message.Message):
    __slots__ = ("idea",)
    IDEA_FIELD_NUMBER: _ClassVar[int]
    idea: _idea_pb2.Idea
    def __init__(self, idea: _Optional[_Union[_idea_pb2.Idea, _Mapping]] = ...) -> None: ...

class IdeaFilesResponse(_message.Message):
    __slots__ = ("files",)
    FILES_FIELD_NUMBER: _ClassVar[int]
    files: _containers.RepeatedCompositeFieldContainer[_idea_pb2.IdeaFile]
    def __init__(self, files: _Optional[_Iterable[_Union[_idea_pb2.IdeaFile, _Mapping]]] = ...) -> None: ...

class IdeaFileResponse(_message.Message):
    __slots__ = ("file",)
    FILE_FIELD_NUMBER: _ClassVar[int]
    file: _idea_pb2.IdeaFile
    def __init__(self, file: _Optional[_Union[_idea_pb2.IdeaFile, _Mapping]] = ...) -> None: ...

class QueueIdeaRequest(_message.Message):
    __slots__ = ("operation",)
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    operation: str
    def __init__(self, operation: _Optional[str] = ...) -> None: ...

class QueueIdeaResponse(_message.Message):
    __slots__ = ("idea", "task_id")
    IDEA_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    idea: _idea_pb2.Idea
    task_id: str
    def __init__(self, idea: _Optional[_Union[_idea_pb2.Idea, _Mapping]] = ..., task_id: _Optional[str] = ...) -> None: ...
