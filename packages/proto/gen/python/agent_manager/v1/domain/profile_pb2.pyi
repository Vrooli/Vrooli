import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import duration_pb2 as _duration_pb2
from buf.validate import validate_pb2 as _validate_pb2
from agent_manager.v1.domain import types_pb2 as _types_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class AgentProfile(_message.Message):
    __slots__ = ()
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    RUNNER_TYPE_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    MAX_TURNS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_TOOLS_FIELD_NUMBER: _ClassVar[int]
    DENIED_TOOLS_FIELD_NUMBER: _ClassVar[int]
    SKIP_PERMISSION_PROMPT_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_SANDBOX_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_APPROVAL_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_PATHS_FIELD_NUMBER: _ClassVar[int]
    DENIED_PATHS_FIELD_NUMBER: _ClassVar[int]
    CREATED_BY_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    runner_type: _types_pb2.RunnerType
    model: str
    max_turns: int
    timeout: _duration_pb2.Duration
    allowed_tools: _containers.RepeatedScalarFieldContainer[str]
    denied_tools: _containers.RepeatedScalarFieldContainer[str]
    skip_permission_prompt: bool
    requires_sandbox: bool
    requires_approval: bool
    allowed_paths: _containers.RepeatedScalarFieldContainer[str]
    denied_paths: _containers.RepeatedScalarFieldContainer[str]
    created_by: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., runner_type: _Optional[_Union[_types_pb2.RunnerType, str]] = ..., model: _Optional[str] = ..., max_turns: _Optional[int] = ..., timeout: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., allowed_tools: _Optional[_Iterable[str]] = ..., denied_tools: _Optional[_Iterable[str]] = ..., skip_permission_prompt: _Optional[bool] = ..., requires_sandbox: _Optional[bool] = ..., requires_approval: _Optional[bool] = ..., allowed_paths: _Optional[_Iterable[str]] = ..., denied_paths: _Optional[_Iterable[str]] = ..., created_by: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class RunConfig(_message.Message):
    __slots__ = ()
    RUNNER_TYPE_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    MAX_TURNS_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_TOOLS_FIELD_NUMBER: _ClassVar[int]
    DENIED_TOOLS_FIELD_NUMBER: _ClassVar[int]
    SKIP_PERMISSION_PROMPT_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_SANDBOX_FIELD_NUMBER: _ClassVar[int]
    REQUIRES_APPROVAL_FIELD_NUMBER: _ClassVar[int]
    ALLOWED_PATHS_FIELD_NUMBER: _ClassVar[int]
    DENIED_PATHS_FIELD_NUMBER: _ClassVar[int]
    runner_type: _types_pb2.RunnerType
    model: str
    max_turns: int
    timeout: _duration_pb2.Duration
    allowed_tools: _containers.RepeatedScalarFieldContainer[str]
    denied_tools: _containers.RepeatedScalarFieldContainer[str]
    skip_permission_prompt: bool
    requires_sandbox: bool
    requires_approval: bool
    allowed_paths: _containers.RepeatedScalarFieldContainer[str]
    denied_paths: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, runner_type: _Optional[_Union[_types_pb2.RunnerType, str]] = ..., model: _Optional[str] = ..., max_turns: _Optional[int] = ..., timeout: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., allowed_tools: _Optional[_Iterable[str]] = ..., denied_tools: _Optional[_Iterable[str]] = ..., skip_permission_prompt: _Optional[bool] = ..., requires_sandbox: _Optional[bool] = ..., requires_approval: _Optional[bool] = ..., allowed_paths: _Optional[_Iterable[str]] = ..., denied_paths: _Optional[_Iterable[str]] = ...) -> None: ...

class HeartbeatConfig(_message.Message):
    __slots__ = ()
    INTERVAL_FIELD_NUMBER: _ClassVar[int]
    TIMEOUT_FIELD_NUMBER: _ClassVar[int]
    MAX_MISSED_BEATS_FIELD_NUMBER: _ClassVar[int]
    interval: _duration_pb2.Duration
    timeout: _duration_pb2.Duration
    max_missed_beats: int
    def __init__(self, interval: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., timeout: _Optional[_Union[datetime.timedelta, _duration_pb2.Duration, _Mapping]] = ..., max_missed_beats: _Optional[int] = ...) -> None: ...
