import datetime

from common.v1 import surface_pb2 as _surface_pb2
from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ImportState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    IMPORT_STATE_UNSPECIFIED: _ClassVar[ImportState]
    IMPORT_STATE_ABSENT: _ClassVar[ImportState]
    IMPORT_STATE_STAGING: _ClassVar[ImportState]
    IMPORT_STATE_READY: _ClassVar[ImportState]
    IMPORT_STATE_UNAVAILABLE: _ClassVar[ImportState]
IMPORT_STATE_UNSPECIFIED: ImportState
IMPORT_STATE_ABSENT: ImportState
IMPORT_STATE_STAGING: ImportState
IMPORT_STATE_READY: ImportState
IMPORT_STATE_UNAVAILABLE: ImportState

class Bounds(_message.Message):
    __slots__ = ("x", "y", "width", "height")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    x: int
    y: int
    width: int
    height: int
    def __init__(self, x: _Optional[int] = ..., y: _Optional[int] = ..., width: _Optional[int] = ..., height: _Optional[int] = ...) -> None: ...

class Region(_message.Message):
    __slots__ = ("x", "y", "width", "height")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    x: int
    y: int
    width: int
    height: int
    def __init__(self, x: _Optional[int] = ..., y: _Optional[int] = ..., width: _Optional[int] = ..., height: _Optional[int] = ...) -> None: ...

class Point(_message.Message):
    __slots__ = ("x", "y")
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    x: float
    y: float
    def __init__(self, x: _Optional[float] = ..., y: _Optional[float] = ...) -> None: ...

class Stroke(_message.Message):
    __slots__ = ("points",)
    POINTS_FIELD_NUMBER: _ClassVar[int]
    points: _containers.RepeatedCompositeFieldContainer[Point]
    def __init__(self, points: _Optional[_Iterable[_Union[Point, _Mapping]]] = ...) -> None: ...

class Source(_message.Message):
    __slots__ = ("surface", "capture_id", "display_id", "geometry_revision", "captured_at", "bounds")
    SURFACE_FIELD_NUMBER: _ClassVar[int]
    CAPTURE_ID_FIELD_NUMBER: _ClassVar[int]
    DISPLAY_ID_FIELD_NUMBER: _ClassVar[int]
    GEOMETRY_REVISION_FIELD_NUMBER: _ClassVar[int]
    CAPTURED_AT_FIELD_NUMBER: _ClassVar[int]
    BOUNDS_FIELD_NUMBER: _ClassVar[int]
    surface: _surface_pb2.SurfaceRef
    capture_id: str
    display_id: str
    geometry_revision: str
    captured_at: _timestamp_pb2.Timestamp
    bounds: Bounds
    def __init__(self, surface: _Optional[_Union[_surface_pb2.SurfaceRef, _Mapping]] = ..., capture_id: _Optional[str] = ..., display_id: _Optional[str] = ..., geometry_revision: _Optional[str] = ..., captured_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., bounds: _Optional[_Union[Bounds, _Mapping]] = ...) -> None: ...

class ImportRequest(_message.Message):
    __slots__ = ("request_id", "source", "region", "strokes", "png", "retention_seconds")
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    REGION_FIELD_NUMBER: _ClassVar[int]
    STROKES_FIELD_NUMBER: _ClassVar[int]
    PNG_FIELD_NUMBER: _ClassVar[int]
    RETENTION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    request_id: str
    source: Source
    region: Region
    strokes: _containers.RepeatedCompositeFieldContainer[Stroke]
    png: bytes
    retention_seconds: int
    def __init__(self, request_id: _Optional[str] = ..., source: _Optional[_Union[Source, _Mapping]] = ..., region: _Optional[_Union[Region, _Mapping]] = ..., strokes: _Optional[_Iterable[_Union[Stroke, _Mapping]]] = ..., png: _Optional[bytes] = ..., retention_seconds: _Optional[int] = ...) -> None: ...

class Document(_message.Message):
    __slots__ = ("request_id", "id", "source", "region", "strokes", "original_sha256", "created_at", "expires_at")
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    REGION_FIELD_NUMBER: _ClassVar[int]
    STROKES_FIELD_NUMBER: _ClassVar[int]
    ORIGINAL_SHA256_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    EXPIRES_AT_FIELD_NUMBER: _ClassVar[int]
    request_id: str
    id: str
    source: Source
    region: Region
    strokes: _containers.RepeatedCompositeFieldContainer[Stroke]
    original_sha256: str
    created_at: _timestamp_pb2.Timestamp
    expires_at: _timestamp_pb2.Timestamp
    def __init__(self, request_id: _Optional[str] = ..., id: _Optional[str] = ..., source: _Optional[_Union[Source, _Mapping]] = ..., region: _Optional[_Union[Region, _Mapping]] = ..., strokes: _Optional[_Iterable[_Union[Stroke, _Mapping]]] = ..., original_sha256: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., expires_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ReferenceRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class ReadResponse(_message.Message):
    __slots__ = ("document", "png")
    DOCUMENT_FIELD_NUMBER: _ClassVar[int]
    PNG_FIELD_NUMBER: _ClassVar[int]
    document: Document
    png: bytes
    def __init__(self, document: _Optional[_Union[Document, _Mapping]] = ..., png: _Optional[bytes] = ...) -> None: ...

class DeleteResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ReconcileImportRequest(_message.Message):
    __slots__ = ("request_id",)
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    request_id: str
    def __init__(self, request_id: _Optional[str] = ...) -> None: ...

class ReconcileImportResponse(_message.Message):
    __slots__ = ("request_id", "state", "document")
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_FIELD_NUMBER: _ClassVar[int]
    request_id: str
    state: ImportState
    document: Document
    def __init__(self, request_id: _Optional[str] = ..., state: _Optional[_Union[ImportState, str]] = ..., document: _Optional[_Union[Document, _Mapping]] = ...) -> None: ...

class RenderResponse(_message.Message):
    __slots__ = ("document", "png", "rendered_sha256")
    DOCUMENT_FIELD_NUMBER: _ClassVar[int]
    PNG_FIELD_NUMBER: _ClassVar[int]
    RENDERED_SHA256_FIELD_NUMBER: _ClassVar[int]
    document: Document
    png: bytes
    rendered_sha256: str
    def __init__(self, document: _Optional[_Union[Document, _Mapping]] = ..., png: _Optional[bytes] = ..., rendered_sha256: _Optional[str] = ...) -> None: ...
