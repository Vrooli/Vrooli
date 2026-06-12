from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PlansRecord(_message.Message):
    __slots__ = ("id", "title", "slug", "path", "created_at", "updated_at", "archived", "archived_at", "source_path", "content_hash")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SLUG_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_FIELD_NUMBER: _ClassVar[int]
    ARCHIVED_AT_FIELD_NUMBER: _ClassVar[int]
    SOURCE_PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_HASH_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    slug: str
    path: str
    created_at: str
    updated_at: str
    archived: bool
    archived_at: str
    source_path: str
    content_hash: str
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., slug: _Optional[str] = ..., path: _Optional[str] = ..., created_at: _Optional[str] = ..., updated_at: _Optional[str] = ..., archived: _Optional[bool] = ..., archived_at: _Optional[str] = ..., source_path: _Optional[str] = ..., content_hash: _Optional[str] = ...) -> None: ...

class PlansAddOutput(_message.Message):
    __slots__ = ("success", "plan")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    PLAN_FIELD_NUMBER: _ClassVar[int]
    success: bool
    plan: PlansRecord
    def __init__(self, success: _Optional[bool] = ..., plan: _Optional[_Union[PlansRecord, _Mapping]] = ...) -> None: ...

class PlansListOutput(_message.Message):
    __slots__ = ("success", "plans")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    PLANS_FIELD_NUMBER: _ClassVar[int]
    success: bool
    plans: _containers.RepeatedCompositeFieldContainer[PlansRecord]
    def __init__(self, success: _Optional[bool] = ..., plans: _Optional[_Iterable[_Union[PlansRecord, _Mapping]]] = ...) -> None: ...

class PlansShowOutput(_message.Message):
    __slots__ = ("success", "plan", "content")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    PLAN_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    success: bool
    plan: PlansRecord
    content: str
    def __init__(self, success: _Optional[bool] = ..., plan: _Optional[_Union[PlansRecord, _Mapping]] = ..., content: _Optional[str] = ...) -> None: ...

class PlansPathOutput(_message.Message):
    __slots__ = ("success", "id", "path")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    success: bool
    id: str
    path: str
    def __init__(self, success: _Optional[bool] = ..., id: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class PlansArchiveOutput(_message.Message):
    __slots__ = ("success", "plan")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    PLAN_FIELD_NUMBER: _ClassVar[int]
    success: bool
    plan: PlansRecord
    def __init__(self, success: _Optional[bool] = ..., plan: _Optional[_Union[PlansRecord, _Mapping]] = ...) -> None: ...

class PlansImportOutput(_message.Message):
    __slots__ = ("success", "plan", "deleted_source")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    PLAN_FIELD_NUMBER: _ClassVar[int]
    DELETED_SOURCE_FIELD_NUMBER: _ClassVar[int]
    success: bool
    plan: PlansRecord
    deleted_source: bool
    def __init__(self, success: _Optional[bool] = ..., plan: _Optional[_Union[PlansRecord, _Mapping]] = ..., deleted_source: _Optional[bool] = ...) -> None: ...

class PlansExportOutput(_message.Message):
    __slots__ = ("success", "id", "path")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    success: bool
    id: str
    path: str
    def __init__(self, success: _Optional[bool] = ..., id: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...
