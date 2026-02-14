from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class Scenario(_message.Message):
    __slots__ = ("name", "display_name", "description", "status", "priority", "completeness_score", "is_greenfield", "tags")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    COMPLETENESS_SCORE_FIELD_NUMBER: _ClassVar[int]
    IS_GREENFIELD_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    name: str
    display_name: str
    description: str
    status: str
    priority: int
    completeness_score: int
    is_greenfield: bool
    tags: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., display_name: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ..., priority: _Optional[int] = ..., completeness_score: _Optional[int] = ..., is_greenfield: _Optional[bool] = ..., tags: _Optional[_Iterable[str]] = ...) -> None: ...

class ScenarioMetadata(_message.Message):
    __slots__ = ("is_greenfield",)
    IS_GREENFIELD_FIELD_NUMBER: _ClassVar[int]
    is_greenfield: bool
    def __init__(self, is_greenfield: _Optional[bool] = ...) -> None: ...
