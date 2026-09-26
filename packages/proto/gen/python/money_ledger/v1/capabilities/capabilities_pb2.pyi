from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class DescribeResponse(_message.Message):
    __slots__ = ("definitions", "states")
    DEFINITIONS_FIELD_NUMBER: _ClassVar[int]
    STATES_FIELD_NUMBER: _ClassVar[int]
    definitions: _containers.RepeatedCompositeFieldContainer[Definition]
    states: _containers.RepeatedCompositeFieldContainer[State]
    def __init__(self, definitions: _Optional[_Iterable[_Union[Definition, _Mapping]]] = ..., states: _Optional[_Iterable[_Union[State, _Mapping]]] = ...) -> None: ...

class Definition(_message.Message):
    __slots__ = ("id", "name", "description", "dependency_kind", "dependency_slug", "features", "action_kind", "action_label", "operator_command")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCY_KIND_FIELD_NUMBER: _ClassVar[int]
    DEPENDENCY_SLUG_FIELD_NUMBER: _ClassVar[int]
    FEATURES_FIELD_NUMBER: _ClassVar[int]
    ACTION_KIND_FIELD_NUMBER: _ClassVar[int]
    ACTION_LABEL_FIELD_NUMBER: _ClassVar[int]
    OPERATOR_COMMAND_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    dependency_kind: str
    dependency_slug: str
    features: _containers.RepeatedScalarFieldContainer[str]
    action_kind: str
    action_label: str
    operator_command: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., dependency_kind: _Optional[str] = ..., dependency_slug: _Optional[str] = ..., features: _Optional[_Iterable[str]] = ..., action_kind: _Optional[str] = ..., action_label: _Optional[str] = ..., operator_command: _Optional[str] = ...) -> None: ...

class State(_message.Message):
    __slots__ = ("id", "status", "message", "reason_code", "checked_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    REASON_CODE_FIELD_NUMBER: _ClassVar[int]
    CHECKED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    status: str
    message: str
    reason_code: str
    checked_at: str
    def __init__(self, id: _Optional[str] = ..., status: _Optional[str] = ..., message: _Optional[str] = ..., reason_code: _Optional[str] = ..., checked_at: _Optional[str] = ...) -> None: ...
