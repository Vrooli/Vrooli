from browser_automation_studio.v1.api import service_pb2 as _service_pb2
from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ProjectEntryKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    PROJECT_ENTRY_KIND_UNSPECIFIED: _ClassVar[ProjectEntryKind]
    PROJECT_ENTRY_KIND_FOLDER: _ClassVar[ProjectEntryKind]
    PROJECT_ENTRY_KIND_WORKFLOW_FILE: _ClassVar[ProjectEntryKind]
    PROJECT_ENTRY_KIND_ASSET_FILE: _ClassVar[ProjectEntryKind]
PROJECT_ENTRY_KIND_UNSPECIFIED: ProjectEntryKind
PROJECT_ENTRY_KIND_FOLDER: ProjectEntryKind
PROJECT_ENTRY_KIND_WORKFLOW_FILE: ProjectEntryKind
PROJECT_ENTRY_KIND_ASSET_FILE: ProjectEntryKind

class ProjectEntry(_message.Message):
    __slots__ = ("id", "project_id", "path", "kind", "workflow_id", "metadata")
    ID_FIELD_NUMBER: _ClassVar[int]
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    id: str
    project_id: str
    path: str
    kind: ProjectEntryKind
    workflow_id: str
    metadata: _struct_pb2.Struct
    def __init__(self, id: _Optional[str] = ..., project_id: _Optional[str] = ..., path: _Optional[str] = ..., kind: _Optional[_Union[ProjectEntryKind, str]] = ..., workflow_id: _Optional[str] = ..., metadata: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class GetProjectFileTreeRequest(_message.Message):
    __slots__ = ("project_id",)
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    def __init__(self, project_id: _Optional[str] = ...) -> None: ...

class GetProjectFileTreeResponse(_message.Message):
    __slots__ = ("entries",)
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[ProjectEntry]
    def __init__(self, entries: _Optional[_Iterable[_Union[ProjectEntry, _Mapping]]] = ...) -> None: ...

class ReadProjectFileRequest(_message.Message):
    __slots__ = ("project_id", "path")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    path: str
    def __init__(self, project_id: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class ReadProjectFileResponse(_message.Message):
    __slots__ = ("workflow",)
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    workflow: _service_pb2.WorkflowSummary
    def __init__(self, workflow: _Optional[_Union[_service_pb2.WorkflowSummary, _Mapping]] = ...) -> None: ...

class MkdirProjectPathRequest(_message.Message):
    __slots__ = ("project_id", "path")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    path: str
    def __init__(self, project_id: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class MkdirProjectPathResponse(_message.Message):
    __slots__ = ("path", "status")
    PATH_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    path: str
    status: str
    def __init__(self, path: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class ProjectWorkflowFileWrite(_message.Message):
    __slots__ = ("name", "type", "description", "tags", "flow_definition", "metadata", "settings")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    FLOW_DEFINITION_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    name: str
    type: str
    description: str
    tags: _containers.RepeatedScalarFieldContainer[str]
    flow_definition: _struct_pb2.Struct
    metadata: _struct_pb2.Struct
    settings: _struct_pb2.Struct
    def __init__(self, name: _Optional[str] = ..., type: _Optional[str] = ..., description: _Optional[str] = ..., tags: _Optional[_Iterable[str]] = ..., flow_definition: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., metadata: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., settings: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ...) -> None: ...

class WriteProjectWorkflowFileRequest(_message.Message):
    __slots__ = ("project_id", "path", "workflow")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    path: str
    workflow: ProjectWorkflowFileWrite
    def __init__(self, project_id: _Optional[str] = ..., path: _Optional[str] = ..., workflow: _Optional[_Union[ProjectWorkflowFileWrite, _Mapping]] = ...) -> None: ...

class WriteProjectWorkflowFileResponse(_message.Message):
    __slots__ = ("path", "workflow_id", "warnings")
    PATH_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    WARNINGS_FIELD_NUMBER: _ClassVar[int]
    path: str
    workflow_id: str
    warnings: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, path: _Optional[str] = ..., workflow_id: _Optional[str] = ..., warnings: _Optional[_Iterable[str]] = ...) -> None: ...

class MoveProjectFileRequest(_message.Message):
    __slots__ = ("project_id", "from_path", "to_path")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    FROM_PATH_FIELD_NUMBER: _ClassVar[int]
    TO_PATH_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    from_path: str
    to_path: str
    def __init__(self, project_id: _Optional[str] = ..., from_path: _Optional[str] = ..., to_path: _Optional[str] = ...) -> None: ...

class MoveProjectFileResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class DeleteProjectFileRequest(_message.Message):
    __slots__ = ("project_id", "path")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    path: str
    def __init__(self, project_id: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class DeleteProjectFileResponse(_message.Message):
    __slots__ = ("status",)
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: str
    def __init__(self, status: _Optional[str] = ...) -> None: ...

class ResyncProjectFilesRequest(_message.Message):
    __slots__ = ("project_id",)
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    def __init__(self, project_id: _Optional[str] = ...) -> None: ...

class ResyncProjectFilesResponse(_message.Message):
    __slots__ = ("project_id", "project_root", "entries_indexed", "workflows_indexed", "assets_indexed")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    PROJECT_ROOT_FIELD_NUMBER: _ClassVar[int]
    ENTRIES_INDEXED_FIELD_NUMBER: _ClassVar[int]
    WORKFLOWS_INDEXED_FIELD_NUMBER: _ClassVar[int]
    ASSETS_INDEXED_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    project_root: str
    entries_indexed: int
    workflows_indexed: int
    assets_indexed: int
    def __init__(self, project_id: _Optional[str] = ..., project_root: _Optional[str] = ..., entries_indexed: _Optional[int] = ..., workflows_indexed: _Optional[int] = ..., assets_indexed: _Optional[int] = ...) -> None: ...

class RevealProjectPathRequest(_message.Message):
    __slots__ = ("project_id", "path")
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    path: str
    def __init__(self, project_id: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class RevealProjectPathResponse(_message.Message):
    __slots__ = ("status", "path")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    status: str
    path: str
    def __init__(self, status: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class OpenProjectFolderRequest(_message.Message):
    __slots__ = ("project_id",)
    PROJECT_ID_FIELD_NUMBER: _ClassVar[int]
    project_id: str
    def __init__(self, project_id: _Optional[str] = ...) -> None: ...

class OpenProjectFolderResponse(_message.Message):
    __slots__ = ("status", "path")
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    status: str
    path: str
    def __init__(self, status: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...
