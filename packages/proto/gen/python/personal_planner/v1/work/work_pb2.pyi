import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class WorkItem(_message.Message):
    __slots__ = ("id", "title", "description", "remaining_minutes", "source_label", "created_at", "updated_at")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    REMAINING_MINUTES_FIELD_NUMBER: _ClassVar[int]
    SOURCE_LABEL_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    UPDATED_AT_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    description: str
    remaining_minutes: int
    source_label: str
    created_at: _timestamp_pb2.Timestamp
    updated_at: _timestamp_pb2.Timestamp
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., description: _Optional[str] = ..., remaining_minutes: _Optional[int] = ..., source_label: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., updated_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class ListWorkItemsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListWorkItemsResponse(_message.Message):
    __slots__ = ("work_items",)
    WORK_ITEMS_FIELD_NUMBER: _ClassVar[int]
    work_items: _containers.RepeatedCompositeFieldContainer[WorkItem]
    def __init__(self, work_items: _Optional[_Iterable[_Union[WorkItem, _Mapping]]] = ...) -> None: ...

class CreateWorkItemRequest(_message.Message):
    __slots__ = ("title", "description", "remaining_minutes", "source_label")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
    REMAINING_MINUTES_FIELD_NUMBER: _ClassVar[int]
    SOURCE_LABEL_FIELD_NUMBER: _ClassVar[int]
    title: str
    description: str
    remaining_minutes: int
    source_label: str
    def __init__(self, title: _Optional[str] = ..., description: _Optional[str] = ..., remaining_minutes: _Optional[int] = ..., source_label: _Optional[str] = ...) -> None: ...

class CreateWorkItemResponse(_message.Message):
    __slots__ = ("work_item",)
    WORK_ITEM_FIELD_NUMBER: _ClassVar[int]
    work_item: WorkItem
    def __init__(self, work_item: _Optional[_Union[WorkItem, _Mapping]] = ...) -> None: ...

class GetWorkItemRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetWorkItemResponse(_message.Message):
    __slots__ = ("work_item",)
    WORK_ITEM_FIELD_NUMBER: _ClassVar[int]
    work_item: WorkItem
    def __init__(self, work_item: _Optional[_Union[WorkItem, _Mapping]] = ...) -> None: ...

class GetTodayPlanRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class TodayPlanEntry(_message.Message):
    __slots__ = ("work_item_id", "title", "source_label", "start_minutes", "duration_minutes")
    WORK_ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_LABEL_FIELD_NUMBER: _ClassVar[int]
    START_MINUTES_FIELD_NUMBER: _ClassVar[int]
    DURATION_MINUTES_FIELD_NUMBER: _ClassVar[int]
    work_item_id: str
    title: str
    source_label: str
    start_minutes: int
    duration_minutes: int
    def __init__(self, work_item_id: _Optional[str] = ..., title: _Optional[str] = ..., source_label: _Optional[str] = ..., start_minutes: _Optional[int] = ..., duration_minutes: _Optional[int] = ...) -> None: ...

class GetTodayPlanResponse(_message.Message):
    __slots__ = ("entries", "planned_minutes", "available_minutes", "breathing_room_minutes")
    ENTRIES_FIELD_NUMBER: _ClassVar[int]
    PLANNED_MINUTES_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_MINUTES_FIELD_NUMBER: _ClassVar[int]
    BREATHING_ROOM_MINUTES_FIELD_NUMBER: _ClassVar[int]
    entries: _containers.RepeatedCompositeFieldContainer[TodayPlanEntry]
    planned_minutes: int
    available_minutes: int
    breathing_room_minutes: int
    def __init__(self, entries: _Optional[_Iterable[_Union[TodayPlanEntry, _Mapping]]] = ..., planned_minutes: _Optional[int] = ..., available_minutes: _Optional[int] = ..., breathing_room_minutes: _Optional[int] = ...) -> None: ...
