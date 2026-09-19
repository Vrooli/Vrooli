from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class BacklogFile(_message.Message):
    __slots__ = ("name", "path", "type", "size", "children")
    NAME_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_FIELD_NUMBER: _ClassVar[int]
    CHILDREN_FIELD_NUMBER: _ClassVar[int]
    name: str
    path: str
    type: str
    size: int
    children: _containers.RepeatedCompositeFieldContainer[BacklogFile]
    def __init__(self, name: _Optional[str] = ..., path: _Optional[str] = ..., type: _Optional[str] = ..., size: _Optional[int] = ..., children: _Optional[_Iterable[_Union[BacklogFile, _Mapping]]] = ...) -> None: ...

class BacklogCriterion(_message.Message):
    __slots__ = ("id", "gherkin", "check")
    ID_FIELD_NUMBER: _ClassVar[int]
    GHERKIN_FIELD_NUMBER: _ClassVar[int]
    CHECK_FIELD_NUMBER: _ClassVar[int]
    id: str
    gherkin: str
    check: CriterionCheck
    def __init__(self, id: _Optional[str] = ..., gherkin: _Optional[str] = ..., check: _Optional[_Union[CriterionCheck, _Mapping]] = ...) -> None: ...

class CriterionCheck(_message.Message):
    __slots__ = ("kind", "scenario", "phase", "argv", "expect_exit")
    KIND_FIELD_NUMBER: _ClassVar[int]
    SCENARIO_FIELD_NUMBER: _ClassVar[int]
    PHASE_FIELD_NUMBER: _ClassVar[int]
    ARGV_FIELD_NUMBER: _ClassVar[int]
    EXPECT_EXIT_FIELD_NUMBER: _ClassVar[int]
    kind: str
    scenario: str
    phase: str
    argv: _containers.RepeatedScalarFieldContainer[str]
    expect_exit: int
    def __init__(self, kind: _Optional[str] = ..., scenario: _Optional[str] = ..., phase: _Optional[str] = ..., argv: _Optional[_Iterable[str]] = ..., expect_exit: _Optional[int] = ...) -> None: ...
