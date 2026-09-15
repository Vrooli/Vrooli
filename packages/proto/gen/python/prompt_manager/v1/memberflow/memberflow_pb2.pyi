from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EmptyRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class MemberRequest(_message.Message):
    __slots__ = ("team_id", "agent_id")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    agent_id: str
    def __init__(self, team_id: _Optional[str] = ..., agent_id: _Optional[str] = ...) -> None: ...

class TeamRequest(_message.Message):
    __slots__ = ("team_id",)
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    def __init__(self, team_id: _Optional[str] = ...) -> None: ...

class UpdateMemberTopicsRequest(_message.Message):
    __slots__ = ("team_id", "agent_id", "topics")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    TOPICS_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    agent_id: str
    topics: _struct_pb2.Value
    def __init__(self, team_id: _Optional[str] = ..., agent_id: _Optional[str] = ..., topics: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...

class JsonResponse(_message.Message):
    __slots__ = ("data",)
    DATA_FIELD_NUMBER: _ClassVar[int]
    data: _struct_pb2.Value
    def __init__(self, data: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...
