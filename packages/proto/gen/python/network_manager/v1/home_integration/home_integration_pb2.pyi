from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class HomeAction(_message.Message):
    __slots__ = ("name", "description", "effect", "approval_required")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    EFFECT_FIELD_NUMBER: _ClassVar[int]
    APPROVAL_REQUIRED_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    effect: str
    approval_required: bool
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., effect: _Optional[str] = ..., approval_required: _Optional[bool] = ...) -> None: ...

class HomeEvent(_message.Message):
    __slots__ = ("id", "type", "summary", "occurred_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    OCCURRED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    type: str
    summary: str
    occurred_at: str
    def __init__(self, id: _Optional[str] = ..., type: _Optional[str] = ..., summary: _Optional[str] = ..., occurred_at: _Optional[str] = ...) -> None: ...

class ListActionsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListActionsResponse(_message.Message):
    __slots__ = ("actions",)
    ACTIONS_FIELD_NUMBER: _ClassVar[int]
    actions: _containers.RepeatedCompositeFieldContainer[HomeAction]
    def __init__(self, actions: _Optional[_Iterable[_Union[HomeAction, _Mapping]]] = ...) -> None: ...

class InvokeActionRequest(_message.Message):
    __slots__ = ("name", "params", "approved")
    class ParamsEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    PARAMS_FIELD_NUMBER: _ClassVar[int]
    APPROVED_FIELD_NUMBER: _ClassVar[int]
    name: str
    params: _containers.ScalarMap[str, str]
    approved: bool
    def __init__(self, name: _Optional[str] = ..., params: _Optional[_Mapping[str, str]] = ..., approved: _Optional[bool] = ...) -> None: ...

class InvokeActionResponse(_message.Message):
    __slots__ = ("status", "message", "event")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    EVENT_FIELD_NUMBER: _ClassVar[int]
    status: str
    message: str
    event: HomeEvent
    def __init__(self, status: _Optional[str] = ..., message: _Optional[str] = ..., event: _Optional[_Union[HomeEvent, _Mapping]] = ...) -> None: ...

class ListRecentEventsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListRecentEventsResponse(_message.Message):
    __slots__ = ("events",)
    EVENTS_FIELD_NUMBER: _ClassVar[int]
    events: _containers.RepeatedCompositeFieldContainer[HomeEvent]
    def __init__(self, events: _Optional[_Iterable[_Union[HomeEvent, _Mapping]]] = ...) -> None: ...
