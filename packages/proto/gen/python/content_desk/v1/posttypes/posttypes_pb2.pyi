from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PostType(_message.Message):
    __slots__ = ("id", "status", "failure_modes")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    FAILURE_MODES_FIELD_NUMBER: _ClassVar[int]
    id: str
    status: str
    failure_modes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., status: _Optional[str] = ..., failure_modes: _Optional[_Iterable[str]] = ...) -> None: ...

class ListPostTypesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListPostTypesResponse(_message.Message):
    __slots__ = ("post_types",)
    POST_TYPES_FIELD_NUMBER: _ClassVar[int]
    post_types: _containers.RepeatedCompositeFieldContainer[PostType]
    def __init__(self, post_types: _Optional[_Iterable[_Union[PostType, _Mapping]]] = ...) -> None: ...

class RegisterPostTypeRequest(_message.Message):
    __slots__ = ("id", "paired_skill", "skill_exists", "doc_v1", "responsibilities_declared", "activate", "failure_modes")
    ID_FIELD_NUMBER: _ClassVar[int]
    PAIRED_SKILL_FIELD_NUMBER: _ClassVar[int]
    SKILL_EXISTS_FIELD_NUMBER: _ClassVar[int]
    DOC_V1_FIELD_NUMBER: _ClassVar[int]
    RESPONSIBILITIES_DECLARED_FIELD_NUMBER: _ClassVar[int]
    ACTIVATE_FIELD_NUMBER: _ClassVar[int]
    FAILURE_MODES_FIELD_NUMBER: _ClassVar[int]
    id: str
    paired_skill: str
    skill_exists: bool
    doc_v1: bool
    responsibilities_declared: bool
    activate: bool
    failure_modes: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, id: _Optional[str] = ..., paired_skill: _Optional[str] = ..., skill_exists: _Optional[bool] = ..., doc_v1: _Optional[bool] = ..., responsibilities_declared: _Optional[bool] = ..., activate: _Optional[bool] = ..., failure_modes: _Optional[_Iterable[str]] = ...) -> None: ...

class RegisterPostTypeResponse(_message.Message):
    __slots__ = ("post_type",)
    POST_TYPE_FIELD_NUMBER: _ClassVar[int]
    post_type: PostType
    def __init__(self, post_type: _Optional[_Union[PostType, _Mapping]] = ...) -> None: ...
