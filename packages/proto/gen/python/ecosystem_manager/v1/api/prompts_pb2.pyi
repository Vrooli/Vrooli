from buf.validate import validate_pb2 as _validate_pb2
from ecosystem_manager.v1.domain import prompt_pb2 as _prompt_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListPromptFilesResponse(_message.Message):
    __slots__ = ("files",)
    FILES_FIELD_NUMBER: _ClassVar[int]
    files: _containers.RepeatedCompositeFieldContainer[_prompt_pb2.PromptFileInfo]
    def __init__(self, files: _Optional[_Iterable[_Union[_prompt_pb2.PromptFileInfo, _Mapping]]] = ...) -> None: ...

class PromptFileResponse(_message.Message):
    __slots__ = ("file",)
    FILE_FIELD_NUMBER: _ClassVar[int]
    file: _prompt_pb2.PromptFile
    def __init__(self, file: _Optional[_Union[_prompt_pb2.PromptFile, _Mapping]] = ...) -> None: ...

class CreatePromptFileRequest(_message.Message):
    __slots__ = ("path", "content")
    PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    path: str
    content: str
    def __init__(self, path: _Optional[str] = ..., content: _Optional[str] = ...) -> None: ...

class UpdatePromptFileRequest(_message.Message):
    __slots__ = ("content",)
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    content: str
    def __init__(self, content: _Optional[str] = ...) -> None: ...

class PromptPreviewRequest(_message.Message):
    __slots__ = ("type", "operation", "target", "steer_set")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    TARGET_FIELD_NUMBER: _ClassVar[int]
    STEER_SET_FIELD_NUMBER: _ClassVar[int]
    type: str
    operation: str
    target: str
    steer_set: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, type: _Optional[str] = ..., operation: _Optional[str] = ..., target: _Optional[str] = ..., steer_set: _Optional[_Iterable[str]] = ...) -> None: ...

class PromptPreviewResponse(_message.Message):
    __slots__ = ("prompt", "size", "sections")
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    SECTIONS_FIELD_NUMBER: _ClassVar[int]
    prompt: str
    size: int
    sections: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, prompt: _Optional[str] = ..., size: _Optional[int] = ..., sections: _Optional[_Iterable[str]] = ...) -> None: ...
