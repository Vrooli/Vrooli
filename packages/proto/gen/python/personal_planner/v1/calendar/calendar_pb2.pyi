import datetime

from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Allocation(_message.Message):
    __slots__ = ("id", "work_item_id", "title", "source_label", "local_date", "start_minutes", "duration_minutes", "state", "created_at", "carried_from_id")
    ID_FIELD_NUMBER: _ClassVar[int]
    WORK_ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SOURCE_LABEL_FIELD_NUMBER: _ClassVar[int]
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    START_MINUTES_FIELD_NUMBER: _ClassVar[int]
    DURATION_MINUTES_FIELD_NUMBER: _ClassVar[int]
    STATE_FIELD_NUMBER: _ClassVar[int]
    CREATED_AT_FIELD_NUMBER: _ClassVar[int]
    CARRIED_FROM_ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    work_item_id: str
    title: str
    source_label: str
    local_date: str
    start_minutes: int
    duration_minutes: int
    state: str
    created_at: _timestamp_pb2.Timestamp
    carried_from_id: str
    def __init__(self, id: _Optional[str] = ..., work_item_id: _Optional[str] = ..., title: _Optional[str] = ..., source_label: _Optional[str] = ..., local_date: _Optional[str] = ..., start_minutes: _Optional[int] = ..., duration_minutes: _Optional[int] = ..., state: _Optional[str] = ..., created_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., carried_from_id: _Optional[str] = ...) -> None: ...

class ListTodayAllocationsRequest(_message.Message):
    __slots__ = ("local_date",)
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    local_date: str
    def __init__(self, local_date: _Optional[str] = ...) -> None: ...

class ListTodayAllocationsResponse(_message.Message):
    __slots__ = ("allocations", "planned_minutes", "available_minutes", "breathing_room_minutes", "external_busy_minutes", "external_event_count", "external_freshness")
    ALLOCATIONS_FIELD_NUMBER: _ClassVar[int]
    PLANNED_MINUTES_FIELD_NUMBER: _ClassVar[int]
    AVAILABLE_MINUTES_FIELD_NUMBER: _ClassVar[int]
    BREATHING_ROOM_MINUTES_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_BUSY_MINUTES_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_EVENT_COUNT_FIELD_NUMBER: _ClassVar[int]
    EXTERNAL_FRESHNESS_FIELD_NUMBER: _ClassVar[int]
    allocations: _containers.RepeatedCompositeFieldContainer[Allocation]
    planned_minutes: int
    available_minutes: int
    breathing_room_minutes: int
    external_busy_minutes: int
    external_event_count: int
    external_freshness: str
    def __init__(self, allocations: _Optional[_Iterable[_Union[Allocation, _Mapping]]] = ..., planned_minutes: _Optional[int] = ..., available_minutes: _Optional[int] = ..., breathing_room_minutes: _Optional[int] = ..., external_busy_minutes: _Optional[int] = ..., external_event_count: _Optional[int] = ..., external_freshness: _Optional[str] = ...) -> None: ...

class ListAllocationsRequest(_message.Message):
    __slots__ = ("start_local_date", "end_local_date")
    START_LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    END_LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    start_local_date: str
    end_local_date: str
    def __init__(self, start_local_date: _Optional[str] = ..., end_local_date: _Optional[str] = ...) -> None: ...

class ListAllocationsResponse(_message.Message):
    __slots__ = ("allocations",)
    ALLOCATIONS_FIELD_NUMBER: _ClassVar[int]
    allocations: _containers.RepeatedCompositeFieldContainer[Allocation]
    def __init__(self, allocations: _Optional[_Iterable[_Union[Allocation, _Mapping]]] = ...) -> None: ...

class CreateAllocationRequest(_message.Message):
    __slots__ = ("work_item_id", "local_date", "start_minutes", "duration_minutes")
    WORK_ITEM_ID_FIELD_NUMBER: _ClassVar[int]
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    START_MINUTES_FIELD_NUMBER: _ClassVar[int]
    DURATION_MINUTES_FIELD_NUMBER: _ClassVar[int]
    work_item_id: str
    local_date: str
    start_minutes: int
    duration_minutes: int
    def __init__(self, work_item_id: _Optional[str] = ..., local_date: _Optional[str] = ..., start_minutes: _Optional[int] = ..., duration_minutes: _Optional[int] = ...) -> None: ...

class CreateAllocationResponse(_message.Message):
    __slots__ = ("allocation",)
    ALLOCATION_FIELD_NUMBER: _ClassVar[int]
    allocation: Allocation
    def __init__(self, allocation: _Optional[_Union[Allocation, _Mapping]] = ...) -> None: ...

class CarryForwardAllocationRequest(_message.Message):
    __slots__ = ("allocation_id", "target_local_date", "start_minutes")
    ALLOCATION_ID_FIELD_NUMBER: _ClassVar[int]
    TARGET_LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    START_MINUTES_FIELD_NUMBER: _ClassVar[int]
    allocation_id: str
    target_local_date: str
    start_minutes: int
    def __init__(self, allocation_id: _Optional[str] = ..., target_local_date: _Optional[str] = ..., start_minutes: _Optional[int] = ...) -> None: ...

class CarryForwardAllocationResponse(_message.Message):
    __slots__ = ("allocation",)
    ALLOCATION_FIELD_NUMBER: _ClassVar[int]
    allocation: Allocation
    def __init__(self, allocation: _Optional[_Union[Allocation, _Mapping]] = ...) -> None: ...

class Routine(_message.Message):
    __slots__ = ("id", "title", "kind", "timezone", "start_date", "end_date", "weekdays", "start_minute", "duration_minutes", "frequency_per_week", "revision", "active")
    ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    TIMEZONE_FIELD_NUMBER: _ClassVar[int]
    START_DATE_FIELD_NUMBER: _ClassVar[int]
    END_DATE_FIELD_NUMBER: _ClassVar[int]
    WEEKDAYS_FIELD_NUMBER: _ClassVar[int]
    START_MINUTE_FIELD_NUMBER: _ClassVar[int]
    DURATION_MINUTES_FIELD_NUMBER: _ClassVar[int]
    FREQUENCY_PER_WEEK_FIELD_NUMBER: _ClassVar[int]
    REVISION_FIELD_NUMBER: _ClassVar[int]
    ACTIVE_FIELD_NUMBER: _ClassVar[int]
    id: str
    title: str
    kind: str
    timezone: str
    start_date: str
    end_date: str
    weekdays: _containers.RepeatedScalarFieldContainer[int]
    start_minute: int
    duration_minutes: int
    frequency_per_week: int
    revision: int
    active: bool
    def __init__(self, id: _Optional[str] = ..., title: _Optional[str] = ..., kind: _Optional[str] = ..., timezone: _Optional[str] = ..., start_date: _Optional[str] = ..., end_date: _Optional[str] = ..., weekdays: _Optional[_Iterable[int]] = ..., start_minute: _Optional[int] = ..., duration_minutes: _Optional[int] = ..., frequency_per_week: _Optional[int] = ..., revision: _Optional[int] = ..., active: _Optional[bool] = ...) -> None: ...

class RoutineOccurrence(_message.Message):
    __slots__ = ("routine_id", "title", "local_date", "start_minute", "duration_minutes", "kind", "generated")
    ROUTINE_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    START_MINUTE_FIELD_NUMBER: _ClassVar[int]
    DURATION_MINUTES_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    GENERATED_FIELD_NUMBER: _ClassVar[int]
    routine_id: str
    title: str
    local_date: str
    start_minute: int
    duration_minutes: int
    kind: str
    generated: bool
    def __init__(self, routine_id: _Optional[str] = ..., title: _Optional[str] = ..., local_date: _Optional[str] = ..., start_minute: _Optional[int] = ..., duration_minutes: _Optional[int] = ..., kind: _Optional[str] = ..., generated: _Optional[bool] = ...) -> None: ...

class ListRoutinesRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListRoutinesResponse(_message.Message):
    __slots__ = ("routines",)
    ROUTINES_FIELD_NUMBER: _ClassVar[int]
    routines: _containers.RepeatedCompositeFieldContainer[Routine]
    def __init__(self, routines: _Optional[_Iterable[_Union[Routine, _Mapping]]] = ...) -> None: ...

class CreateRoutineRequest(_message.Message):
    __slots__ = ("title", "kind", "timezone", "start_date", "end_date", "weekdays", "start_minute", "duration_minutes", "frequency_per_week")
    TITLE_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    TIMEZONE_FIELD_NUMBER: _ClassVar[int]
    START_DATE_FIELD_NUMBER: _ClassVar[int]
    END_DATE_FIELD_NUMBER: _ClassVar[int]
    WEEKDAYS_FIELD_NUMBER: _ClassVar[int]
    START_MINUTE_FIELD_NUMBER: _ClassVar[int]
    DURATION_MINUTES_FIELD_NUMBER: _ClassVar[int]
    FREQUENCY_PER_WEEK_FIELD_NUMBER: _ClassVar[int]
    title: str
    kind: str
    timezone: str
    start_date: str
    end_date: str
    weekdays: _containers.RepeatedScalarFieldContainer[int]
    start_minute: int
    duration_minutes: int
    frequency_per_week: int
    def __init__(self, title: _Optional[str] = ..., kind: _Optional[str] = ..., timezone: _Optional[str] = ..., start_date: _Optional[str] = ..., end_date: _Optional[str] = ..., weekdays: _Optional[_Iterable[int]] = ..., start_minute: _Optional[int] = ..., duration_minutes: _Optional[int] = ..., frequency_per_week: _Optional[int] = ...) -> None: ...

class CreateRoutineResponse(_message.Message):
    __slots__ = ("routine",)
    ROUTINE_FIELD_NUMBER: _ClassVar[int]
    routine: Routine
    def __init__(self, routine: _Optional[_Union[Routine, _Mapping]] = ...) -> None: ...

class ListRoutineOccurrencesRequest(_message.Message):
    __slots__ = ("start_local_date", "end_local_date")
    START_LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    END_LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    start_local_date: str
    end_local_date: str
    def __init__(self, start_local_date: _Optional[str] = ..., end_local_date: _Optional[str] = ...) -> None: ...

class ListRoutineOccurrencesResponse(_message.Message):
    __slots__ = ("occurrences",)
    OCCURRENCES_FIELD_NUMBER: _ClassVar[int]
    occurrences: _containers.RepeatedCompositeFieldContainer[RoutineOccurrence]
    def __init__(self, occurrences: _Optional[_Iterable[_Union[RoutineOccurrence, _Mapping]]] = ...) -> None: ...

class SkipRoutineOccurrenceRequest(_message.Message):
    __slots__ = ("routine_id", "local_date", "expected_revision")
    ROUTINE_ID_FIELD_NUMBER: _ClassVar[int]
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    routine_id: str
    local_date: str
    expected_revision: int
    def __init__(self, routine_id: _Optional[str] = ..., local_date: _Optional[str] = ..., expected_revision: _Optional[int] = ...) -> None: ...

class SkipRoutineOccurrenceResponse(_message.Message):
    __slots__ = ("skipped",)
    SKIPPED_FIELD_NUMBER: _ClassVar[int]
    skipped: bool
    def __init__(self, skipped: _Optional[bool] = ...) -> None: ...

class RescheduleRoutineOccurrenceRequest(_message.Message):
    __slots__ = ("routine_id", "local_date", "start_minute", "expected_revision")
    ROUTINE_ID_FIELD_NUMBER: _ClassVar[int]
    LOCAL_DATE_FIELD_NUMBER: _ClassVar[int]
    START_MINUTE_FIELD_NUMBER: _ClassVar[int]
    EXPECTED_REVISION_FIELD_NUMBER: _ClassVar[int]
    routine_id: str
    local_date: str
    start_minute: int
    expected_revision: int
    def __init__(self, routine_id: _Optional[str] = ..., local_date: _Optional[str] = ..., start_minute: _Optional[int] = ..., expected_revision: _Optional[int] = ...) -> None: ...

class RescheduleRoutineOccurrenceResponse(_message.Message):
    __slots__ = ("rescheduled",)
    RESCHEDULED_FIELD_NUMBER: _ClassVar[int]
    rescheduled: bool
    def __init__(self, rescheduled: _Optional[bool] = ...) -> None: ...
