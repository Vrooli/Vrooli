from buf.validate import validate_pb2 as _validate_pb2
from swarm_manager.v1.domain import backlog_pb2 as _backlog_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CreateBacklogItemRequest(_message.Message):
    __slots__ = ("name", "title", "description", "priority", "tags", "kind", "research_target")
    NAME_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    RESEARCH_TARGET_FIELD_NUMBER: _ClassVar[int]
    name: str
    title: str
    description: str
    priority: int
    tags: _containers.RepeatedScalarFieldContainer[str]
    kind: str
    research_target: str
    def __init__(self, name: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., priority: _Optional[int] = ..., tags: _Optional[_Iterable[str]] = ..., kind: _Optional[str] = ..., research_target: _Optional[str] = ...) -> None: ...

class UpdateBacklogItemRequest(_message.Message):
    __slots__ = ("title", "description", "status", "priority", "tags", "research_target")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    PRIORITY_FIELD_NUMBER: _ClassVar[int]
    TAGS_FIELD_NUMBER: _ClassVar[int]
    RESEARCH_TARGET_FIELD_NUMBER: _ClassVar[int]
    title: str
    description: str
    status: str
    priority: int
    tags: _containers.RepeatedScalarFieldContainer[str]
    research_target: str
    def __init__(self, title: _Optional[str] = ..., description: _Optional[str] = ..., status: _Optional[str] = ..., priority: _Optional[int] = ..., tags: _Optional[_Iterable[str]] = ..., research_target: _Optional[str] = ...) -> None: ...

class ListBacklogItemsResponse(_message.Message):
    __slots__ = ("items",)
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[_backlog_pb2.BacklogItem]
    def __init__(self, items: _Optional[_Iterable[_Union[_backlog_pb2.BacklogItem, _Mapping]]] = ...) -> None: ...

class BacklogItemResponse(_message.Message):
    __slots__ = ("item",)
    ITEM_FIELD_NUMBER: _ClassVar[int]
    item: _backlog_pb2.BacklogItem
    def __init__(self, item: _Optional[_Union[_backlog_pb2.BacklogItem, _Mapping]] = ...) -> None: ...

class BacklogFilesResponse(_message.Message):
    __slots__ = ("files",)
    FILES_FIELD_NUMBER: _ClassVar[int]
    files: _containers.RepeatedCompositeFieldContainer[_backlog_pb2.BacklogFile]
    def __init__(self, files: _Optional[_Iterable[_Union[_backlog_pb2.BacklogFile, _Mapping]]] = ...) -> None: ...

class BacklogFileResponse(_message.Message):
    __slots__ = ("file",)
    FILE_FIELD_NUMBER: _ClassVar[int]
    file: _backlog_pb2.BacklogFile
    def __init__(self, file: _Optional[_Union[_backlog_pb2.BacklogFile, _Mapping]] = ...) -> None: ...

class QueueBacklogItemRequest(_message.Message):
    __slots__ = ("operation",)
    OPERATION_FIELD_NUMBER: _ClassVar[int]
    operation: str
    def __init__(self, operation: _Optional[str] = ...) -> None: ...

class QueueBacklogItemResponse(_message.Message):
    __slots__ = ("item", "task_id", "run_id", "base_url", "created")
    ITEM_FIELD_NUMBER: _ClassVar[int]
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    BASE_URL_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    item: _backlog_pb2.BacklogItem
    task_id: str
    run_id: str
    base_url: str
    created: str
    def __init__(self, item: _Optional[_Union[_backlog_pb2.BacklogItem, _Mapping]] = ..., task_id: _Optional[str] = ..., run_id: _Optional[str] = ..., base_url: _Optional[str] = ..., created: _Optional[str] = ...) -> None: ...

class BacklogResearchRequest(_message.Message):
    __slots__ = ("prompt", "scope_path", "project_root", "mode", "target_kind")
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    SCOPE_PATH_FIELD_NUMBER: _ClassVar[int]
    PROJECT_ROOT_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    TARGET_KIND_FIELD_NUMBER: _ClassVar[int]
    prompt: str
    scope_path: str
    project_root: str
    mode: str
    target_kind: str
    def __init__(self, prompt: _Optional[str] = ..., scope_path: _Optional[str] = ..., project_root: _Optional[str] = ..., mode: _Optional[str] = ..., target_kind: _Optional[str] = ...) -> None: ...

class BacklogResearchResponse(_message.Message):
    __slots__ = ("task_id", "run_id", "base_url", "created")
    TASK_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    BASE_URL_FIELD_NUMBER: _ClassVar[int]
    CREATED_FIELD_NUMBER: _ClassVar[int]
    task_id: str
    run_id: str
    base_url: str
    created: str
    def __init__(self, task_id: _Optional[str] = ..., run_id: _Optional[str] = ..., base_url: _Optional[str] = ..., created: _Optional[str] = ...) -> None: ...

class ConvertBacklogItemRequest(_message.Message):
    __slots__ = ("target_kind", "target_name")
    TARGET_KIND_FIELD_NUMBER: _ClassVar[int]
    TARGET_NAME_FIELD_NUMBER: _ClassVar[int]
    target_kind: str
    target_name: str
    def __init__(self, target_kind: _Optional[str] = ..., target_name: _Optional[str] = ...) -> None: ...
