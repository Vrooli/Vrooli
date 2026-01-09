import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Project(_message.Message):
    __slots__ = ()
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    FOLDER_PATH_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    folder_path: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., folder_path: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ProjectStats(_message.Message):
    __slots__ = ()
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_COUNT_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_COUNT_FIELD_NUMBER: _ClassVar[int]
    LAST_EXECUTION_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    workflow_count: int
    execution_count: int
    last_execution: _timestamp_pb2.Timestamp
    def __init__(self, project_id: _Optional[str] = ..., workflow_count: _Optional[int] = ..., execution_count: _Optional[int] = ..., last_execution: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ProjectWithStats(_message.Message):
    __slots__ = ()
    PROJECT_FIELD_NUMBER: _ClassVar[int]
    STATS_FIELD_NUMBER: _ClassVar[int]
    project: Project
    stats: ProjectStats
    def __init__(self, project: _Optional[_Union[Project, _Mapping]] = ..., stats: _Optional[_Union[ProjectStats, _Mapping]] = ...) -> None: ...

class ProjectList(_message.Message):
    __slots__ = ()
    PROJECTS_FIELD_NUMBER: _ClassVar[int]
    projects: _containers.RepeatedCompositeFieldContainer[ProjectWithStats]
    def __init__(self, projects: _Optional[_Iterable[_Union[ProjectWithStats, _Mapping]]] = ...) -> None: ...
