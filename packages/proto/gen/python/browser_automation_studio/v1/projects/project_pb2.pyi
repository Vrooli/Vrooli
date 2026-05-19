import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from browser_automation_studio.v1.api import service_pb2 as _service_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PresetKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PRESET_KIND_UNSPECIFIED: _ClassVar[PresetKind]
    PRESET_KIND_EMPTY: _ClassVar[PresetKind]
    PRESET_KIND_RECOMMENDED: _ClassVar[PresetKind]
    PRESET_KIND_CUSTOM: _ClassVar[PresetKind]
PRESET_KIND_UNSPECIFIED: PresetKind
PRESET_KIND_EMPTY: PresetKind
PRESET_KIND_RECOMMENDED: PresetKind
PRESET_KIND_CUSTOM: PresetKind

class Project(_message.Message):
    __slots__ = ("id", "name", "description", "folder_path", "created_at", "updated_at")
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
    __slots__ = ("project_id", "workflow_count", "execution_count", "last_execution")
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
    __slots__ = ("project", "stats")
    PROJECT_FIELD_NUMBER: _ClassVar[int]
    STATS_FIELD_NUMBER: _ClassVar[int]
    project: Project
    stats: ProjectStats
    def __init__(self, project: _Optional[_Union[Project, _Mapping]] = ..., stats: _Optional[_Union[ProjectStats, _Mapping]] = ...) -> None: ...

class ProjectList(_message.Message):
    __slots__ = ("projects",)
    PROJECTS_FIELD_NUMBER: _ClassVar[int]
    projects: _containers.RepeatedCompositeFieldContainer[ProjectWithStats]
    def __init__(self, projects: _Optional[_Iterable[_Union[ProjectWithStats, _Mapping]]] = ...) -> None: ...

class CreateProjectRequest(_message.Message):
    __slots__ = ("name", "description", "folder_path", "preset", "preset_paths")
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    FOLDER_PATH_FIELD_NUMBER: _ClassVar[int]
    PRESET_FIELD_NUMBER: _ClassVar[int]
    PRESET_PATHS_FIELD_NUMBER: _ClassVar[int]
    name: str
    description: str
    folder_path: str
    preset: PresetKind
    preset_paths: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, name: _Optional[str] = ..., description: _Optional[str] = ..., folder_path: _Optional[str] = ..., preset: _Optional[_Union[PresetKind, str]] = ..., preset_paths: _Optional[_Iterable[str]] = ...) -> None: ...

class CreateProjectResponse(_message.Message):
    __slots__ = ("project", "stats")
    PROJECT_FIELD_NUMBER: _ClassVar[int]
    STATS_FIELD_NUMBER: _ClassVar[int]
    project: Project
    stats: ProjectStats
    def __init__(self, project: _Optional[_Union[Project, _Mapping]] = ..., stats: _Optional[_Union[ProjectStats, _Mapping]] = ...) -> None: ...

class ListProjectsRequest(_message.Message):
    __slots__ = ("limit", "offset")
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    limit: int
    offset: int
    def __init__(self, limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListProjectsResponse(_message.Message):
    __slots__ = ("projects",)
    PROJECTS_FIELD_NUMBER: _ClassVar[int]
    projects: _containers.RepeatedCompositeFieldContainer[ProjectWithStats]
    def __init__(self, projects: _Optional[_Iterable[_Union[ProjectWithStats, _Mapping]]] = ...) -> None: ...

class GetProjectRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetProjectResponse(_message.Message):
    __slots__ = ("project", "stats")
    PROJECT_FIELD_NUMBER: _ClassVar[int]
    STATS_FIELD_NUMBER: _ClassVar[int]
    project: Project
    stats: ProjectStats
    def __init__(self, project: _Optional[_Union[Project, _Mapping]] = ..., stats: _Optional[_Union[ProjectStats, _Mapping]] = ...) -> None: ...

class UpdateProjectRequest(_message.Message):
    __slots__ = ("id", "name", "description", "folder_path")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    FOLDER_PATH_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    description: str
    folder_path: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., folder_path: _Optional[str] = ...) -> None: ...

class UpdateProjectResponse(_message.Message):
    __slots__ = ("project",)
    PROJECT_FIELD_NUMBER: _ClassVar[int]
    project: Project
    def __init__(self, project: _Optional[_Union[Project, _Mapping]] = ...) -> None: ...

class DeleteProjectRequest(_message.Message):
    __slots__ = ("id", "delete_files")
    ID_FIELD_NUMBER: _ClassVar[int]
    DELETE_FILES_FIELD_NUMBER: _ClassVar[int]
    id: str
    delete_files: bool
    def __init__(self, id: _Optional[str] = ..., delete_files: _Optional[bool] = ...) -> None: ...

class DeleteProjectResponse(_message.Message):
    __slots__ = ("files_deleted",)
    FILES_DELETED_FIELD_NUMBER: _ClassVar[int]
    files_deleted: bool
    def __init__(self, files_deleted: _Optional[bool] = ...) -> None: ...

class ListProjectWorkflowsRequest(_message.Message):
    __slots__ = ("project_id", "limit", "offset")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    limit: int
    offset: int
    def __init__(self, project_id: _Optional[str] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class BulkDeleteProjectWorkflowsRequest(_message.Message):
    __slots__ = ("project_id", "workflow_ids")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_IDS_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    workflow_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, project_id: _Optional[str] = ..., workflow_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class BulkDeleteProjectWorkflowsResponse(_message.Message):
    __slots__ = ("deleted_count", "deleted_ids")
    DELETED_COUNT_FIELD_NUMBER: _ClassVar[int]
    DELETED_IDS_FIELD_NUMBER: _ClassVar[int]
    deleted_count: int
    deleted_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, deleted_count: _Optional[int] = ..., deleted_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class ExecuteAllProjectWorkflowsRequest(_message.Message):
    __slots__ = ("project_id",)
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    def __init__(self, project_id: _Optional[str] = ...) -> None: ...

class ProjectWorkflowExecutionResult(_message.Message):
    __slots__ = ("workflow_id", "workflow_name", "execution_id", "status", "error")
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_NAME_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    workflow_id: str
    workflow_name: str
    execution_id: str
    status: str
    error: str
    def __init__(self, workflow_id: _Optional[str] = ..., workflow_name: _Optional[str] = ..., execution_id: _Optional[str] = ..., status: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...

class ExecuteAllProjectWorkflowsResponse(_message.Message):
    __slots__ = ("message", "executions")
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    EXECUTIONS_FIELD_NUMBER: _ClassVar[int]
    message: str
    executions: _containers.RepeatedCompositeFieldContainer[ProjectWorkflowExecutionResult]
    def __init__(self, message: _Optional[str] = ..., executions: _Optional[_Iterable[_Union[ProjectWorkflowExecutionResult, _Mapping]]] = ...) -> None: ...
