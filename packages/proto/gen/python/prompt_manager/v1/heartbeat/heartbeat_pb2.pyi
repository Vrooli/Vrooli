from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class EmptyRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class QueryRequest(_message.Message):
    __slots__ = ("query",)
    class QueryEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    QUERY_FIELD_NUMBER: _ClassVar[int]
    query: _containers.ScalarMap[str, str]
    def __init__(self, query: _Optional[_Mapping[str, str]] = ...) -> None: ...

class TeamRequest(_message.Message):
    __slots__ = ("team_id",)
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    def __init__(self, team_id: _Optional[str] = ...) -> None: ...

class TeamQueryRequest(_message.Message):
    __slots__ = ("team_id", "query")
    class QueryEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    query: _containers.ScalarMap[str, str]
    def __init__(self, team_id: _Optional[str] = ..., query: _Optional[_Mapping[str, str]] = ...) -> None: ...

class TeamMutationRequest(_message.Message):
    __slots__ = ("team_id", "body", "query")
    class QueryEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    body: _struct_pb2.Value
    query: _containers.ScalarMap[str, str]
    def __init__(self, team_id: _Optional[str] = ..., body: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., query: _Optional[_Mapping[str, str]] = ...) -> None: ...

class MemberRequest(_message.Message):
    __slots__ = ("team_id", "agent_id")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    agent_id: str
    def __init__(self, team_id: _Optional[str] = ..., agent_id: _Optional[str] = ...) -> None: ...

class MemberQueryRequest(_message.Message):
    __slots__ = ("team_id", "agent_id", "query")
    class QueryEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    agent_id: str
    query: _containers.ScalarMap[str, str]
    def __init__(self, team_id: _Optional[str] = ..., agent_id: _Optional[str] = ..., query: _Optional[_Mapping[str, str]] = ...) -> None: ...

class MemberMutationRequest(_message.Message):
    __slots__ = ("team_id", "agent_id", "body", "query")
    class QueryEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    agent_id: str
    body: _struct_pb2.Value
    query: _containers.ScalarMap[str, str]
    def __init__(self, team_id: _Optional[str] = ..., agent_id: _Optional[str] = ..., body: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ..., query: _Optional[_Mapping[str, str]] = ...) -> None: ...

class TaskMutationRequest(_message.Message):
    __slots__ = ("team_id", "task_id", "body")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    task_id: str
    body: _struct_pb2.Value
    def __init__(self, team_id: _Optional[str] = ..., task_id: _Optional[str] = ..., body: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...

class BugMutationRequest(_message.Message):
    __slots__ = ("team_id", "draft_id", "body")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    DRAFT_ID_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    draft_id: str
    body: _struct_pb2.Value
    def __init__(self, team_id: _Optional[str] = ..., draft_id: _Optional[str] = ..., body: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...

class LogRequest(_message.Message):
    __slots__ = ("team_id", "agent_id", "log_id")
    TEAM_ID_FIELD_NUMBER: _ClassVar[int]
    AGENT_ID_FIELD_NUMBER: _ClassVar[int]
    LOG_ID_FIELD_NUMBER: _ClassVar[int]
    team_id: str
    agent_id: str
    log_id: str
    def __init__(self, team_id: _Optional[str] = ..., agent_id: _Optional[str] = ..., log_id: _Optional[str] = ...) -> None: ...

class RunRequest(_message.Message):
    __slots__ = ("run_id",)
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    def __init__(self, run_id: _Optional[str] = ...) -> None: ...

class RunQueryRequest(_message.Message):
    __slots__ = ("run_id", "query")
    class QueryEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    QUERY_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    query: _containers.ScalarMap[str, str]
    def __init__(self, run_id: _Optional[str] = ..., query: _Optional[_Mapping[str, str]] = ...) -> None: ...

class RunMutationRequest(_message.Message):
    __slots__ = ("run_id", "body")
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    run_id: str
    body: _struct_pb2.Value
    def __init__(self, run_id: _Optional[str] = ..., body: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...

class JsonMutationRequest(_message.Message):
    __slots__ = ("body",)
    BODY_FIELD_NUMBER: _ClassVar[int]
    body: _struct_pb2.Value
    def __init__(self, body: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...

class JsonResponse(_message.Message):
    __slots__ = ("data",)
    DATA_FIELD_NUMBER: _ClassVar[int]
    data: _struct_pb2.Value
    def __init__(self, data: _Optional[_Union[_struct_pb2.Value, _Mapping]] = ...) -> None: ...
