import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from browser_automation_studio.v1 import shared_pb2 as _shared_pb2
from browser_automation_studio.v1 import geometry_pb2 as _geometry_pb2
from browser_automation_studio.v1 import selectors_pb2 as _selectors_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ConsoleLogEntry(_message.Message):
    __slots__ = ()
    LEVEL_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    STACK_FIELD_NUMBER: _ClassVar[int]
    LOCATION_FIELD_NUMBER: _ClassVar[int]
    level: _shared_pb2.LogLevel
    text: str
    timestamp: _timestamp_pb2.Timestamp
    stack: str
    location: str
    def __init__(self, level: _Optional[_Union[_shared_pb2.LogLevel, str]] = ..., text: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., stack: _Optional[str] = ..., location: _Optional[str] = ...) -> None: ...

class NetworkEvent(_message.Message):
    __slots__ = ()
    TYPE_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    RESOURCE_TYPE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    OK_FIELD_NUMBER: _ClassVar[int]
    FAILURE_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    type: _shared_pb2.NetworkEventType
    url: str
    method: str
    resource_type: str
    status: int
    ok: bool
    failure: str
    timestamp: _timestamp_pb2.Timestamp
    def __init__(self, type: _Optional[_Union[_shared_pb2.NetworkEventType, str]] = ..., url: _Optional[str] = ..., method: _Optional[str] = ..., resource_type: _Optional[str] = ..., status: _Optional[int] = ..., ok: _Optional[bool] = ..., failure: _Optional[str] = ..., timestamp: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ActionTelemetry(_message.Message):
    __slots__ = ()
    URL_FIELD_NUMBER: _ClassVar[int]
    FRAME_ID_FIELD_NUMBER: _ClassVar[int]
    SCREENSHOT_FIELD_NUMBER: _ClassVar[int]
    DOM_SNAPSHOT_PREVIEW_FIELD_NUMBER: _ClassVar[int]
    DOM_SNAPSHOT_HTML_FIELD_NUMBER: _ClassVar[int]
    ELEMENT_BOUNDING_BOX_FIELD_NUMBER: _ClassVar[int]
    CLICK_POSITION_FIELD_NUMBER: _ClassVar[int]
    CURSOR_POSITION_FIELD_NUMBER: _ClassVar[int]
    CURSOR_TRAIL_FIELD_NUMBER: _ClassVar[int]
    HIGHLIGHT_REGIONS_FIELD_NUMBER: _ClassVar[int]
    MASK_REGIONS_FIELD_NUMBER: _ClassVar[int]
    ZOOM_FACTOR_FIELD_NUMBER: _ClassVar[int]
    CONSOLE_LOGS_FIELD_NUMBER: _ClassVar[int]
    NETWORK_EVENTS_FIELD_NUMBER: _ClassVar[int]
    url: str
    frame_id: str
    screenshot: TimelineScreenshot
    dom_snapshot_preview: str
    dom_snapshot_html: str
    element_bounding_box: _geometry_pb2.BoundingBox
    click_position: _geometry_pb2.Point
    cursor_position: _geometry_pb2.Point
    cursor_trail: _containers.RepeatedCompositeFieldContainer[_geometry_pb2.Point]
    highlight_regions: _containers.RepeatedCompositeFieldContainer[_selectors_pb2.HighlightRegion]
    mask_regions: _containers.RepeatedCompositeFieldContainer[_selectors_pb2.MaskRegion]
    zoom_factor: float
    console_logs: _containers.RepeatedCompositeFieldContainer[ConsoleLogEntry]
    network_events: _containers.RepeatedCompositeFieldContainer[NetworkEvent]
    def __init__(self, url: _Optional[str] = ..., frame_id: _Optional[str] = ..., screenshot: _Optional[_Union[TimelineScreenshot, _Mapping]] = ..., dom_snapshot_preview: _Optional[str] = ..., dom_snapshot_html: _Optional[str] = ..., element_bounding_box: _Optional[_Union[_geometry_pb2.BoundingBox, _Mapping]] = ..., click_position: _Optional[_Union[_geometry_pb2.Point, _Mapping]] = ..., cursor_position: _Optional[_Union[_geometry_pb2.Point, _Mapping]] = ..., cursor_trail: _Optional[_Iterable[_Union[_geometry_pb2.Point, _Mapping]]] = ..., highlight_regions: _Optional[_Iterable[_Union[_selectors_pb2.HighlightRegion, _Mapping]]] = ..., mask_regions: _Optional[_Iterable[_Union[_selectors_pb2.MaskRegion, _Mapping]]] = ..., zoom_factor: _Optional[float] = ..., console_logs: _Optional[_Iterable[_Union[ConsoleLogEntry, _Mapping]]] = ..., network_events: _Optional[_Iterable[_Union[NetworkEvent, _Mapping]]] = ...) -> None: ...

class TimelineScreenshot(_message.Message):
    __slots__ = ()
    ARTIFACT_ID_FIELD_NUMBER: _ClassVar[int]
    URL_FIELD_NUMBER: _ClassVar[int]
    THUMBNAIL_URL_FIELD_NUMBER: _ClassVar[int]
    WIDTH_FIELD_NUMBER: _ClassVar[int]
    HEIGHT_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    SIZE_BYTES_FIELD_NUMBER: _ClassVar[int]
    artifact_id: str
    url: str
    thumbnail_url: str
    width: int
    height: int
    content_type: str
    size_bytes: int
    def __init__(self, artifact_id: _Optional[str] = ..., url: _Optional[str] = ..., thumbnail_url: _Optional[str] = ..., width: _Optional[int] = ..., height: _Optional[int] = ..., content_type: _Optional[str] = ..., size_bytes: _Optional[int] = ...) -> None: ...
