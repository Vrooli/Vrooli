from buf.validate import validate_pb2 as _validate_pb2
from swarm_manager.v1.domain import initiative_pb2 as _initiative_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CreateInitiativeRequest(_message.Message):
    __slots__ = ("name", "title", "description", "items")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    name: str
    title: str
    description: str
    items: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., items: _Optional[_Iterable[str]] = ...) -> None: ...

class UpdateInitiativeRequest(_message.Message):
    __slots__ = ("title", "description", "status", "items")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    title: str
    description: str
    status: str
    items: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, title: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ..., items: _Optional[_Iterable[str]] = ...) -> None: ...

class InitiativeResponse(_message.Message):
    __slots__ = ("initiative", "rollup")
    INITIATIVE_FIELD_NUMBER: _ClassVar[int]
    ROLLUP_FIELD_NUMBER: _ClassVar[int]
    initiative: _initiative_pb2.Initiative
    rollup: _initiative_pb2.InitiativeRollup
    def __init__(self, initiative: _Optional[_Union[_initiative_pb2.Initiative, _Mapping]] = ..., rollup: _Optional[_Union[_initiative_pb2.InitiativeRollup, _Mapping]] = ...) -> None: ...

class ListInitiativesResponse(_message.Message):
    __slots__ = ("items",)
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[InitiativeResponse]
    def __init__(self, items: _Optional[_Iterable[_Union[InitiativeResponse, _Mapping]]] = ...) -> None: ...
