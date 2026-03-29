from swarm_manager.v1.domain import agent_activity_pb2 as _agent_activity_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ListAgentActivitiesResponse(_message.Message):
    __slots__ = ("items",)
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[_agent_activity_pb2.AgentActivity]
    def __init__(self, items: _Optional[_Iterable[_Union[_agent_activity_pb2.AgentActivity, _Mapping]]] = ...) -> None: ...

class AgentActivityResponse(_message.Message):
    __slots__ = ("activity",)
    ACTIVITY_FIELD_NUMBER: _ClassVar[int]
    activity: _agent_activity_pb2.AgentActivity
    def __init__(self, activity: _Optional[_Union[_agent_activity_pb2.AgentActivity, _Mapping]] = ...) -> None: ...
