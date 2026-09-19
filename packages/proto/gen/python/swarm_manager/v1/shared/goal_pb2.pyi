from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class Milestone(_message.Message):
    __slots__ = ("name", "title", "description", "items", "acceptance_criteria", "depends_on", "archived_at")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    ACCEPTANCE_CRITERIA_FIELD_NUMBER: _ClassVar[int]
    DEPENDS_ON_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_AT_FIELD_NUMBER: _ClassVar[int]
    name: str
    title: str
    description: str
    items: _containers.RepeatedScalarFieldContainer[str]
    acceptance_criteria: _containers.RepeatedScalarFieldContainer[str]
    depends_on: _containers.RepeatedScalarFieldContainer[str]
    archived_at: str
    def __init__(self, name: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., items: _Optional[_Iterable[str]] = ..., acceptance_criteria: _Optional[_Iterable[str]] = ..., depends_on: _Optional[_Iterable[str]] = ..., archived_at: _Optional[str] = ...) -> None: ...
