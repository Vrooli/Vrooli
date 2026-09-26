import datetime

from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf import struct_pb2 as _struct_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Export(_message.Message):
    __slots__ = ("id", "execution_id", "workflow_id", "name", "format", "settings", "storage_url", "thumbnail_url", "file_size_bytes", "duration_ms", "frame_count", "ai_caption", "ai_caption_generated_at", "status", "error", "created_at", "updated_at", "workflow_name", "execution_date")
    ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    STORAGE_URL_FIELD_NUMBER: _ClassVar[int]
    THUMBNAIL_URL_FIELD_NUMBER: _ClassVar[int]
    FILE_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    FRAME_COUNT_FIELD_NUMBER: _ClassVar[int]
    AI_CAPTION_FIELD_NUMBER: _ClassVar[int]
    AI_CAPTION_GENERATED_AT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_NAME_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_DATE_FIELD_NUMBER: _ClassVar[int]
    id: str
    execution_id: str
    workflow_id: str
    name: str
    format: str
    settings: _struct_pb2.Struct
    storage_url: str
    thumbnail_url: str
    file_size_bytes: int
    duration_ms: int
    frame_count: int
    ai_caption: str
    ai_caption_generated_at: _timestamp_pb2.Timestamp
    status: str
    error: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    workflow_name: str
    execution_date: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., execution_id: _Optional[str] = ..., workflow_id: _Optional[str] = ..., name: _Optional[str] = ..., format: _Optional[str] = ..., settings: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., storage_url: _Optional[str] = ..., thumbnail_url: _Optional[str] = ..., file_size_bytes: _Optional[int] = ..., duration_ms: _Optional[int] = ..., frame_count: _Optional[int] = ..., ai_caption: _Optional[str] = ..., ai_caption_generated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., status: _Optional[str] = ..., error: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., workflow_name: _Optional[str] = ..., execution_date: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListExportsRequest(_message.Message):
    __slots__ = ("execution_id", "workflow_id", "limit", "offset")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    OFFSET_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    workflow_id: str
    limit: int
    offset: int
    def __init__(self, execution_id: _Optional[str] = ..., workflow_id: _Optional[str] = ..., limit: _Optional[int] = ..., offset: _Optional[int] = ...) -> None: ...

class ListExportsResponse(_message.Message):
    __slots__ = ("exports", "total")
    EXPORTS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    exports: _containers.RepeatedCompositeFieldContainer[Export]
    total: int
    def __init__(self, exports: _Optional[_Iterable[_Union[Export, _Mapping]]] = ..., total: _Optional[int] = ...) -> None: ...

class CreateExportRequest(_message.Message):
    __slots__ = ("execution_id", "workflow_id", "name", "format", "settings", "storage_url", "thumbnail_url", "file_size_bytes", "duration_ms", "frame_count", "status")
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    WORKFLOW_ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    STORAGE_URL_FIELD_NUMBER: _ClassVar[int]
    THUMBNAIL_URL_FIELD_NUMBER: _ClassVar[int]
    FILE_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    FRAME_COUNT_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    execution_id: str
    workflow_id: str
    name: str
    format: str
    settings: _struct_pb2.Struct
    storage_url: str
    thumbnail_url: str
    file_size_bytes: int
    duration_ms: int
    frame_count: int
    status: str
    def __init__(self, execution_id: _Optional[str] = ..., workflow_id: _Optional[str] = ..., name: _Optional[str] = ..., format: _Optional[str] = ..., settings: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., storage_url: _Optional[str] = ..., thumbnail_url: _Optional[str] = ..., file_size_bytes: _Optional[int] = ..., duration_ms: _Optional[int] = ..., frame_count: _Optional[int] = ..., status: _Optional[str] = ...) -> None: ...

class CreateExportResponse(_message.Message):
    __slots__ = ("export_id", "status", "export")
    EXPORT_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EXPORT_FIELD_NUMBER: _ClassVar[int]
    export_id: str
    status: str
    export: Export
    def __init__(self, export_id: _Optional[str] = ..., status: _Optional[str] = ..., export: _Optional[_Union[Export, _Mapping]] = ...) -> None: ...

class GetExportRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetExportResponse(_message.Message):
    __slots__ = ("export",)
    EXPORT_FIELD_NUMBER: _ClassVar[int]
    export: Export
    def __init__(self, export: _Optional[_Union[Export, _Mapping]] = ...) -> None: ...

class UpdateExportRequest(_message.Message):
    __slots__ = ("id", "name", "settings", "storage_url", "thumbnail_url", "file_size_bytes", "duration_ms", "frame_count", "ai_caption", "status", "error")
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    SETTINGS_FIELD_NUMBER: _ClassVar[int]
    STORAGE_URL_FIELD_NUMBER: _ClassVar[int]
    THUMBNAIL_URL_FIELD_NUMBER: _ClassVar[int]
    FILE_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    FRAME_COUNT_FIELD_NUMBER: _ClassVar[int]
    AI_CAPTION_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    id: str
    name: str
    settings: _struct_pb2.Struct
    storage_url: str
    thumbnail_url: str
    file_size_bytes: int
    duration_ms: int
    frame_count: int
    ai_caption: str
    status: str
    error: str
    def __init__(self, id: _Optional[str] = ..., name: _Optional[str] = ..., settings: _Optional[_Union[_struct_pb2.Struct, _Mapping]] = ..., storage_url: _Optional[str] = ..., thumbnail_url: _Optional[str] = ..., file_size_bytes: _Optional[int] = ..., duration_ms: _Optional[int] = ..., frame_count: _Optional[int] = ..., ai_caption: _Optional[str] = ..., status: _Optional[str] = ..., error: _Optional[str] = ...) -> None: ...

class UpdateExportResponse(_message.Message):
    __slots__ = ("export_id", "status", "export")
    EXPORT_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    EXPORT_FIELD_NUMBER: _ClassVar[int]
    export_id: str
    status: str
    export: Export
    def __init__(self, export_id: _Optional[str] = ..., status: _Optional[str] = ..., export: _Optional[_Union[Export, _Mapping]] = ...) -> None: ...

class DeleteExportRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteExportResponse(_message.Message):
    __slots__ = ("export_id", "status")
    EXPORT_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    export_id: str
    status: str
    def __init__(self, export_id: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class GetExportStatusRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetExportStatusResponse(_message.Message):
    __slots__ = ("export_id", "execution_id", "status", "format", "name", "storage_url", "file_size_bytes", "error")
    EXPORT_ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_ID_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    FORMAT_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    STORAGE_URL_FIELD_NUMBER: _ClassVar[int]
    FILE_SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    export_id: str
    execution_id: str
    status: str
    format: str
    name: str
    storage_url: str
    file_size_bytes: int
    error: str
    def __init__(self, export_id: _Optional[str] = ..., execution_id: _Optional[str] = ..., status: _Optional[str] = ..., format: _Optional[str] = ..., name: _Optional[str] = ..., storage_url: _Optional[str] = ..., file_size_bytes: _Optional[int] = ..., error: _Optional[str] = ...) -> None: ...

class GenerateExportCaptionRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GenerateExportCaptionResponse(_message.Message):
    __slots__ = ("export_id", "caption", "export")
    EXPORT_ID_FIELD_NUMBER: _ClassVar[int]
    CAPTION_FIELD_NUMBER: _ClassVar[int]
    EXPORT_FIELD_NUMBER: _ClassVar[int]
    export_id: str
    caption: str
    export: Export
    def __init__(self, export_id: _Optional[str] = ..., caption: _Optional[str] = ..., export: _Optional[_Union[Export, _Mapping]] = ...) -> None: ...

class RevealExportRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class RevealExportResponse(_message.Message):
    __slots__ = ("export_id", "path", "status")
    EXPORT_ID_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    export_id: str
    path: str
    status: str
    def __init__(self, export_id: _Optional[str] = ..., path: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...

class OpenExportFolderRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class OpenExportFolderResponse(_message.Message):
    __slots__ = ("export_id", "folder", "status")
    EXPORT_ID_FIELD_NUMBER: _ClassVar[int]
    FOLDER_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    export_id: str
    folder: str
    status: str
    def __init__(self, export_id: _Optional[str] = ..., folder: _Optional[str] = ..., status: _Optional[str] = ...) -> None: ...
